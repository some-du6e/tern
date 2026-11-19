// Package core is tern's client: it signs in, keeps a socket alive to every
// environment on the account, mirrors their shells and the open thread, and
// turns that state into a ui.Scene each frame. All network work runs on
// goroutines; state sits behind one mutex, and every change asks the window
// to redraw.
package core

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// envState is one environment: its socket, config and thread list.
type envState struct {
	env    t3.Environment
	status string // "online", "offline", "busy"
	paired bool

	conn      *t3.Conn
	gen       int // bumps on every (re)connect so stale goroutines drop out
	retry     int
	config    *t3.ServerConfig
	projects  map[string]t3.Project
	threads   map[string]*t3.Thread
	shellSave time.Time // last time the shell cache was written
}

func newEnv(e t3.Environment, paired bool) *envState {
	return &envState{env: e, status: "offline", paired: paired, projects: map[string]t3.Project{}, threads: map[string]*t3.Thread{}}
}

func (e *envState) id() string { return e.env.EnvironmentID }

// A ref names a thread across environments: "<environment>/<thread>".
func ref(envID, threadID string) string { return envID + "/" + threadID }

func splitRef(r string) (envID, threadID string) {
	envID, threadID, _ = strings.Cut(r, "/")
	return
}

type App struct {
	store *t3.Store
	key   *t3.Key
	clerk *t3.Clerk
	relay *t3.Relay

	redraw func()

	mu       sync.Mutex
	booted   bool // boot has decided between sign-in and the inbox
	signedIn bool
	login    ui.SignIn
	signIn   *t3.SignIn

	envs []*envState
	net  string

	openID   string // ref of the open thread
	tabs     []string
	view     *threadView
	draft    string // project id while composing a new thread
	draftEnv string // and its environment
	// What the user opened in each thread's timeline, for the last 100.
	disclosures     map[string]*disclosure
	disclosureOrder []string
	// cited is the reply a quoted response's card goes to, by its thread's
	// ref and its message id, until the timeline has shown it; forking the
	// reply whose fork is on its way.
	cited   citedReply
	forking string

	choices map[string]*choice       // per thread ref, or "draft"
	images  map[string][]pastedImage // pasted and not sent yet, same keys
	trash   map[string][]pastedImage // their chips were deleted; undo brings them back
	sent    map[string]*sentThumb    // sent pictures' thumbnails, by env/attachment id
	// sentOrder is sent's keys, shown longest ago first; scenes counts the
	// scenes built, for which one showed a picture last.
	sentOrder []string
	scenes    uint64

	search       string
	showSnoozed  bool
	showSettled  bool
	picker       *pickerState
	traitsOpen   bool
	accessOpen   bool
	projectsOpen bool
	projectsHero bool // the project menu hangs off the heading, not the header
	accountOpen  bool
	detailsOpen  bool // the thread details popover is up
	// T3's shell: the details card shown inline (on by default), the
	// sidebar collapsed, and its dragged width in dp (0 for the default).
	// All three persist.
	detailsCard   bool
	sidebarHidden bool
	sidebarWidth  float32
	machinesOpen  bool                // the footer's machines menu is up
	drawers       map[string]*drawer  // each thread's terminals, by ref
	browsers      map[string]*browser // each thread's browser panel, by ref
	ctxRef        string              // the thread whose right-click menu is open
	ctxConfirm    bool                // Delete was clicked once and wants a second click
	errText       string
	notice        string
	answers       map[string]map[int]bool // question id -> picked option indexes

	uiOps []func(*ui.Theme) // editor and focus changes from goroutines, run on the UI goroutine

	// ver counts state changes. Scene reuses its last result while ver and
	// the clock's second stay the same, so idle frames cost almost nothing.
	ver     atomic.Uint64
	memo    *ui.Scene
	memoVer uint64
	memoSec int64
}

// onUI queues f to run on the UI goroutine before the next frame. Editors
// aren't safe to touch from anywhere else.
func (a *App) onUI(f func(*ui.Theme)) {
	a.mu.Lock()
	a.uiOps = append(a.uiOps, f)
	a.mu.Unlock()
	a.changed()
}

// RunUI runs the queued UI work. Call it at the start of each frame.
func (a *App) RunUI(th *ui.Theme) {
	a.mu.Lock()
	ops := a.uiOps
	a.uiOps = nil
	a.mu.Unlock()
	for _, f := range ops {
		f(th)
	}
}

type choice struct {
	sel     t3.ModelSelection
	runtime string
}

type pickerState struct {
	tab   int
	query string
}

// New opens the store and the DPoP key. redraw is called, from any
// goroutine, whenever the scene may have changed.
func New(redraw func()) (*App, error) {
	store, err := t3.OpenStore()
	if err != nil {
		return nil, err
	}
	return NewWithStore(store, redraw)
}

// NewWithStore is New with an explicit store, for tests and tools.
func NewWithStore(store *t3.Store, redraw func()) (*App, error) {
	key, err := t3.LoadKey(store)
	if err != nil {
		return nil, err
	}
	clerk := t3.NewClerk(store)
	a := &App{
		store:    store,
		key:      key,
		clerk:    clerk,
		relay:    t3.NewRelay(clerk, key),
		redraw:   redraw,
		choices:  map[string]*choice{},
		images:   map[string][]pastedImage{},
		trash:    map[string][]pastedImage{},
		sent:     map[string]*sentThumb{},
		drawers:  map[string]*drawer{},
		browsers: map[string]*browser{},
		answers:  map[string]map[int]bool{},
	}
	a.disclosures = map[string]*disclosure{}
	_ = store.Delete("env-filter") // from when a machine's chip filtered the inbox
	a.detailsCard = true
	_, _ = store.Get("details-card", &a.detailsCard)
	_, _ = store.Get("sidebar-hidden", &a.sidebarHidden)
	_, _ = store.Get("sidebar-width", &a.sidebarWidth)
	_ = store.Delete("panel-hidden") // from when details were a side panel
	return a, nil
}

func (a *App) changed() {
	a.ver.Add(1)
	if a.redraw != nil {
		a.redraw()
	}
}

// fail shows an error banner. Call with the lock held.
func (a *App) fail(err error) {
	if err != nil {
		a.errText = err.Error()
	}
}

// Start shows sign-in, or loads environments if there's a credential.
func (a *App) Start() {
	go a.boot()
}

func (a *App) boot() {
	defer func() {
		a.mu.Lock()
		a.booted = true
		a.mu.Unlock()
		a.changed()
	}()
	paired := t3.Paired(a.store)
	var envs []*envState
	var cachedList []t3.Environment
	if a.clerk.HasClient() {
		_, _ = a.store.Get("envs.json", &cachedList)
	}
	if len(cachedList) > 0 {
		// Start on the list from last time, and refresh it in the background.
		for _, e := range cachedList {
			envs = append(envs, newEnv(e, false))
		}
		go a.refreshEnvs()
	} else if a.clerk.HasClient() {
		a.setNet("loading environments…")
		list, err := a.relay.ListEnvironments()
		if err == nil {
			_ = a.store.Put("envs.json", list)
		}
		if err != nil && len(paired) == 0 {
			a.mu.Lock()
			a.signedIn = false
			if !errors.Is(err, t3.ErrSignedOut) {
				a.login.Error = err.Error()
			}
			a.mu.Unlock()
			a.changed()
			return
		}
		for _, e := range list {
			envs = append(envs, newEnv(e, false))
		}
	}
	for _, e := range paired {
		envs = append(envs, newEnv(e, true))
	}
	a.mu.Lock()
	if len(envs) == 0 {
		a.signedIn = false
		a.mu.Unlock()
		a.changed()
		return
	}
	a.signedIn = true
	a.envs = envs
	a.mu.Unlock()
	a.changed()
	go a.refreshStatuses()
	for _, e := range envs {
		a.loadShellCache(e)
		a.connect(e)
	}
}

// refreshEnvs reloads the Connect environment list. A signed-out Clerk
// session sends the user back to sign-in.
func (a *App) refreshEnvs() {
	list, err := a.relay.ListEnvironments()
	if errors.Is(err, t3.ErrSignedOut) {
		a.mu.Lock()
		a.signedIn = false
		a.mu.Unlock()
		_ = a.store.Delete("envs.json")
		a.changed()
		return
	}
	if err != nil {
		return
	}
	_ = a.store.Put("envs.json", list)
	a.mu.Lock()
	known := map[string]*envState{}
	for _, e := range a.envs {
		known[e.id()] = e
	}
	var envs, added []*envState
	for _, e := range list {
		if k := known[e.EnvironmentID]; k != nil && !k.paired {
			k.env = e
			envs = append(envs, k)
			delete(known, e.EnvironmentID)
			continue
		}
		n := newEnv(e, false)
		envs = append(envs, n)
		added = append(added, n)
	}
	for _, e := range a.envs {
		if e.paired {
			envs = append(envs, e)
			delete(known, e.id())
		}
	}
	// Environments gone from the account: close their sockets.
	for _, e := range known {
		e.gen++
		if e.conn != nil {
			go e.conn.Close()
			e.conn = nil
		}
	}
	a.envs = envs
	a.mu.Unlock()
	a.changed()
	for _, e := range added {
		a.loadShellCache(e)
		a.connect(e)
	}
}

func (a *App) setNet(s string) {
	a.mu.Lock()
	a.net = s
	a.mu.Unlock()
	a.changed()
}

// refreshStatuses asks the relay which machines are up, every minute. One
// that comes back online gets a socket straight away instead of waiting out
// its backoff.
func (a *App) refreshStatuses() {
	for {
		a.mu.Lock()
		envs := append([]*envState(nil), a.envs...)
		signedIn := a.signedIn
		a.mu.Unlock()
		if !signedIn {
			return
		}
		for _, e := range envs {
			if e.paired {
				continue
			}
			st, err := a.relay.Status(e.id())
			if err != nil {
				st = "offline"
			}
			a.mu.Lock()
			wake := false
			if e.conn == nil && e.status != "busy" {
				wake = st == "online" && e.status == "offline"
				e.status = st
			}
			a.mu.Unlock()
			if wake {
				a.connect(e)
			}
		}
		a.changed()
		time.Sleep(time.Minute)
	}
}

// env finds an environment by id. Call with the lock held.
func (a *App) env(id string) *envState {
	for _, e := range a.envs {
		if e.id() == id {
			return e
		}
	}
	return nil
}

// thread finds a thread by ref. Call with the lock held.
func (a *App) thread(r string) (*envState, *t3.Thread) {
	envID, id := splitRef(r)
	e := a.env(envID)
	if e == nil {
		return nil, nil
	}
	return e, e.threads[id]
}

// cur is the environment of the open thread or draft. Call with the lock held.
func (a *App) cur() *envState {
	if a.draft != "" {
		return a.env(a.draftEnv)
	}
	if a.openID != "" {
		envID, _ := splitRef(a.openID)
		return a.env(envID)
	}
	return nil
}

// curThread is the open thread. Call with the lock held.
func (a *App) curThread() *t3.Thread {
	if a.openID == "" {
		return nil
	}
	_, t := a.thread(a.openID)
	return t
}

// cfg is the config that applies to the composer: the open environment's,
// else any. Call with the lock held.
func (a *App) cfg() *t3.ServerConfig {
	if e := a.cur(); e != nil {
		return e.config
	}
	for _, e := range a.envs {
		if e.config != nil {
			return e.config
		}
	}
	return nil
}

func (a *App) connect(e *envState) {
	a.mu.Lock()
	if old := e.conn; old != nil {
		e.conn = nil
		go old.Close()
	}
	e.gen++
	gen := e.gen
	e.status = "busy"
	a.net = "connecting to " + e.env.Label + "…"
	a.mu.Unlock()
	a.changed()
	go func() {
		started := time.Now()
		conn, err := a.dial(e)
		a.mu.Lock()
		if gen != e.gen {
			a.mu.Unlock()
			if conn != nil {
				conn.Close()
			}
			return
		}
		if err != nil {
			e.status = "offline"
			a.net = e.env.Label + " offline: " + err.Error()
			a.mu.Unlock()
			a.changed()
			a.scheduleReconnect(e, gen)
			return
		}
		e.conn = conn
		e.retry = 0
		e.status = "online"
		a.net = e.env.Label + " · socket open"
		var openID string
		if envID, _ := splitRef(a.openID); envID == e.id() {
			openID = a.openID
		}
		a.mu.Unlock()
		a.changed()
		a.subscribeShell(e, conn, gen, started)
		go a.loadConfig(e, conn, gen)
		if openID != "" {
			a.subscribeThread(e, conn, gen, openID)
		}
		go func() {
			<-conn.Done()
			a.mu.Lock()
			if gen != e.gen || e.conn != conn {
				a.mu.Unlock()
				return
			}
			e.conn = nil
			e.status = "busy"
			a.mu.Unlock()
			a.changed()
			a.scheduleReconnect(e, gen)
		}()
		a.renewLater(e, conn, gen)
	}()
}

// dial uses a cached session when there is one, else asks the relay for a
// new one. A cached session the server refuses is dropped and replaced.
func (a *App) dial(e *envState) (*t3.Conn, error) {
	id := e.id()
	cached := t3.CachedSession(a.store, id)
	if cached != nil {
		conn, err := t3.Dial(context.Background(), a.key, cached)
		if err == nil || e.paired {
			return conn, err
		}
		t3.ForgetSession(a.store, id)
	}
	if e.paired {
		return nil, errors.New("the pairing session expired; pair again")
	}
	sess, err := t3.OpenSession(a.relay, a.store, e.env)
	if err != nil {
		return nil, err
	}
	return t3.Dial(context.Background(), a.key, sess)
}

// renewLater swaps in a fresh Connect session three minutes before this one
// expires. The socket itself outlives its session, but tickets, history and
// uploads need a valid one.
func (a *App) renewLater(e *envState, conn *t3.Conn, gen int) {
	s := conn.Session()
	if s.Bearer {
		return
	}
	wait := time.Until(s.ExpiresAt) - 3*time.Minute
	if wait <= 0 {
		return
	}
	time.AfterFunc(wait, func() {
		a.mu.Lock()
		live := e.gen == gen && e.conn == conn
		a.mu.Unlock()
		if live {
			t3.ForgetSession(a.store, e.id())
			a.connect(e)
		}
	})
}

func (a *App) scheduleReconnect(e *envState, gen int) {
	a.mu.Lock()
	d := time.Duration(min(30_000, 1000<<min(e.retry, 5))) * time.Millisecond
	if !e.paired && e.status == "offline" && e.retry > 0 {
		// The relay says it's off; refreshStatuses wakes it when it's back.
		d = 5 * time.Minute
	}
	d = time.Duration(float64(d) * (0.5 + rand.Float64()))
	e.retry++
	a.mu.Unlock()
	a.changed()
	time.AfterFunc(d, func() {
		a.mu.Lock()
		ok := e.gen == gen && e.conn == nil && a.env(e.id()) == e
		a.mu.Unlock()
		if ok {
			a.connect(e)
		}
	})
}

func (a *App) loadConfig(e *envState, conn *t3.Conn, gen int) {
	var cfg t3.ServerConfig
	err := conn.Call("server.getConfig", map[string]any{}, &cfg)
	a.mu.Lock()
	defer a.changed()
	defer a.mu.Unlock()
	if gen != e.gen {
		return
	}
	if err != nil {
		a.fail(err)
		return
	}
	e.config = &cfg
}

func (a *App) subscribeShell(e *envState, conn *t3.Conn, gen int, started time.Time) {
	conn.Stream("orchestration.subscribeShell", map[string]any{"requestCompletionMarker": true}, func(v json.RawMessage) {
		var it t3.ShellItem
		if json.Unmarshal(v, &it) != nil {
			return
		}
		a.mu.Lock()
		if gen != e.gen {
			a.mu.Unlock()
			return
		}
		switch it.Kind {
		case "snapshot":
			if it.Snapshot == nil {
				break
			}
			// A snapshot with resolvedRepositoryIdentityRoots only refreshes
			// projects whose repo identity changed. Merge it.
			if it.ResolvedRoots != nil {
				for _, p := range it.Snapshot.Projects {
					e.projects[p.ID] = p
				}
				break
			}
			e.projects = map[string]t3.Project{}
			for _, p := range it.Snapshot.Projects {
				e.projects[p.ID] = p
			}
			e.threads = map[string]*t3.Thread{}
			for i := range it.Snapshot.Threads {
				t := it.Snapshot.Threads[i]
				e.threads[t.ID] = &t
			}
			a.net = e.env.Label + " · " + len3(len(e.threads)) + " threads in " + time.Since(started).Round(time.Millisecond).String()
		case "project.updated":
			if it.Project != nil {
				e.projects[it.Project.ID] = *it.Project
			}
		case "project.removed":
			delete(e.projects, it.ProjectID)
		case "thread.updated":
			if it.Thread != nil {
				if (it.Location == "" || it.Location == "active") && it.Thread.DeletedAt == "" {
					t := *it.Thread
					e.threads[t.ID] = &t
				} else {
					delete(e.threads, it.Thread.ID)
				}
			}
		case "thread.removed":
			delete(e.threads, it.ThreadID)
		}
		save := it.Kind == "snapshot" || time.Since(e.shellSave) > 15*time.Second
		if save {
			e.shellSave = time.Now()
			a.saveShellCacheLocked(e)
		}
		a.mu.Unlock()
		a.changed()
	}, func(err error) {
		if err != nil && !errors.Is(err, t3.ErrClosed) {
			a.mu.Lock()
			if gen == e.gen {
				a.fail(err)
			}
			a.mu.Unlock()
			a.changed()
		}
	})
}

func len3(n int) string {
	s := []byte(strconv.Itoa(n))
	var out []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}

// shellCache is the last thread list, so the inbox shows at once on start
// while the live snapshot loads.
type shellCache struct {
	Projects []t3.Project `json:"projects"`
	Threads  []t3.Thread  `json:"threads"`
}

func shellName(envID string) string { return "shell." + envID + ".json" }

func (a *App) loadShellCache(e *envState) {
	var c shellCache
	if ok, err := a.store.Get(shellName(e.id()), &c); !ok || err != nil {
		return
	}
	a.mu.Lock()
	if len(e.threads) == 0 {
		for _, p := range c.Projects {
			e.projects[p.ID] = p
		}
		for i := range c.Threads {
			t := c.Threads[i]
			e.threads[t.ID] = &t
		}
	}
	a.mu.Unlock()
	a.changed()
}

// saveShellCacheLocked snapshots the shell and writes it off the UI path.
func (a *App) saveShellCacheLocked(e *envState) {
	c := shellCache{}
	for _, p := range e.projects {
		c.Projects = append(c.Projects, p)
	}
	for _, t := range e.threads {
		c.Threads = append(c.Threads, *t)
	}
	id := e.id()
	go func() { _ = a.store.Put(shellName(id), c) }()
}

// ---------- the open thread ----------

type threadView struct {
	id         string // the thread's own id
	env        *envState
	raw        map[string]map[string]json.RawMessage // row key -> merged item fields
	items      map[string]*t3.TurnItem
	runs       map[string]t3.Run
	messages   map[string]t3.Message
	cursor     string
	more       bool
	loading    bool
	loaded     bool
	historyErr string // why the last page of history didn't load
	// What the user opened in the timeline: folds by run, work groups and
	// entries by id, kept per thread (disclosureFor).
	unfolded, groups, open map[string]bool
	// The provider's turns, threads and sessions, and the runtime requests,
	// by id.
	turns    map[string]t3.ProviderTurn
	pthreads map[string]t3.ProviderThread
	sessions map[string]t3.ProviderSession
	requests map[string]t3.RuntimeRequestRecord
	// The latest run and its status as the timeline last saw them.
	seenRun, seenStatus string
	// Open entries' full items, fetched when the socket left part out: the
	// item, the revision it was fetched for, the revision being fetched,
	// and why one didn't load.
	full     map[string]*t3.TurnItem
	fullRev  map[string]string
	fetching map[string]string
	fetchErr map[string]string
	cancel   func()
}

func newView(e *envState, id string, d *disclosure) *threadView {
	return &threadView{
		id: id, env: e, raw: map[string]map[string]json.RawMessage{}, items: map[string]*t3.TurnItem{},
		runs: map[string]t3.Run{}, messages: map[string]t3.Message{}, turns: map[string]t3.ProviderTurn{},
		pthreads: map[string]t3.ProviderThread{}, sessions: map[string]t3.ProviderSession{},
		requests: map[string]t3.RuntimeRequestRecord{}, unfolded: d.unfolded, groups: d.groups, open: d.open, full: map[string]*t3.TurnItem{},
		fullRev: map[string]string{}, fetching: map[string]string{}, fetchErr: map[string]string{},
	}
}

// disclosure is what the user opened in one thread's timeline. Like T3,
// tern keeps it for the session for the last 100 threads, so switching
// back finds the timeline as it was (timelineScrollAnchoring.ts:131-151).
type disclosure struct {
	unfolded, groups, open map[string]bool
}

// disclosureFor is thread ref's disclosure. Call with the lock held.
func (a *App) disclosureFor(ref string) *disclosure {
	for i, r := range a.disclosureOrder {
		if r == ref {
			a.disclosureOrder = append(append(a.disclosureOrder[:i:i], a.disclosureOrder[i+1:]...), ref)
			return a.disclosures[ref]
		}
	}
	d := &disclosure{unfolded: map[string]bool{}, groups: map[string]bool{}, open: map[string]bool{}}
	a.disclosures[ref] = d
	a.disclosureOrder = append(a.disclosureOrder, ref)
	if len(a.disclosureOrder) > 100 {
		delete(a.disclosures, a.disclosureOrder[0])
		a.disclosureOrder = a.disclosureOrder[1:]
	}
	return d
}

func rowKey(it *t3.TurnItem) string {
	if it.MessageID != "" {
		return "m:" + it.MessageID
	}
	return it.ID
}

// put merges a turn item in, like the web app's {...old, ...item}.
func (v *threadView) put(raw json.RawMessage) {
	var fields map[string]json.RawMessage
	var it t3.TurnItem
	if json.Unmarshal(raw, &fields) != nil || json.Unmarshal(raw, &it) != nil {
		return
	}
	k := rowKey(&it)
	merged := v.raw[k]
	if merged == nil {
		merged = map[string]json.RawMessage{}
		v.raw[k] = merged
	}
	for f, val := range fields {
		merged[f] = val
	}
	all, _ := json.Marshal(merged)
	var out t3.TurnItem
	if json.Unmarshal(all, &out) == nil {
		v.items[k] = &out
	}
}

func (v *threadView) sorted() []*t3.TurnItem {
	list := make([]*t3.TurnItem, 0, len(v.items))
	for _, it := range v.items {
		list = append(list, it)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Ordinal < list[j].Ordinal })
	return list
}

func (a *App) closeThreadLocked() {
	if a.view != nil && a.view.cancel != nil {
		a.view.cancel()
	}
	a.view = nil
	a.openID = ""
	a.picker, a.traitsOpen, a.accessOpen, a.projectsOpen = nil, false, false, false
}

// openThread shows a thread, by ref, and subscribes to it. Call without the
// lock.
func (a *App) openThread(id string) {
	a.mu.Lock()
	a.draft, a.projectsOpen = "", false
	if a.openID == id && a.view != nil {
		a.mu.Unlock()
		return
	}
	a.closeThreadLocked()
	a.openID = id
	found := false
	for _, t := range a.tabs {
		found = found || t == id
	}
	if !found {
		a.tabs = append(a.tabs, id)
		if len(a.tabs) > 6 {
			a.tabs = a.tabs[len(a.tabs)-6:]
		}
	}
	e, _ := a.thread(id)
	var conn *t3.Conn
	var gen int
	if e != nil {
		conn, gen = e.conn, e.gen
	}
	a.mu.Unlock()
	a.changed()
	if conn != nil {
		a.subscribeThread(e, conn, gen, id)
	}
}

func (a *App) subscribeThread(e *envState, conn *t3.Conn, gen int, r string) {
	_, id := splitRef(r)
	a.mu.Lock()
	if a.openID != r {
		a.mu.Unlock()
		return
	}
	if a.view == nil || a.view.id != id || a.view.env != e {
		a.view = newView(e, id, a.disclosureFor(r))
	}
	if a.view.cancel != nil {
		a.view.cancel()
	}
	v := a.view
	a.mu.Unlock()
	// A bounded snapshot is the recent window plus a cursor for older
	// pages; compact mode drops the duplicate turnItems list.
	v.cancel = conn.Stream("orchestration.subscribeThread", map[string]any{
		"threadId": id, "requestCompletionMarker": true, "acceptBoundedSnapshot": true, "acceptCompactTurnItems": true,
	}, func(raw json.RawMessage) {
		var it t3.ThreadItem
		if json.Unmarshal(raw, &it) != nil {
			return
		}
		a.mu.Lock()
		if gen != e.gen || a.view != v {
			a.mu.Unlock()
			return
		}
		switch {
		case it.Kind == "snapshot" && it.Projection != nil:
			v.raw, v.items = map[string]map[string]json.RawMessage{}, map[string]*t3.TurnItem{}
			for _, row := range it.Projection.VisibleTurnItems {
				v.put(row.Item)
			}
			v.runs = map[string]t3.Run{}
			for _, r := range it.Projection.Runs {
				v.runs[r.ID] = r
			}
			v.messages = map[string]t3.Message{}
			for _, m := range it.Projection.Messages {
				v.messages[m.ID] = m
			}
			v.turns = map[string]t3.ProviderTurn{}
			for _, pt := range it.Projection.ProviderTurns {
				v.turns[pt.ID] = pt
			}
			v.pthreads, v.sessions = map[string]t3.ProviderThread{}, map[string]t3.ProviderSession{}
			for _, pt := range it.Projection.ProviderThreads {
				v.pthreads[pt.ID] = pt
			}
			for _, ps := range it.Projection.ProviderSessions {
				v.sessions[ps.ID] = ps
			}
			v.requests = map[string]t3.RuntimeRequestRecord{}
			for _, rr := range it.Projection.RuntimeRequests {
				v.requests[rr.ID] = rr
			}
			v.cursor, v.more, v.loaded = it.HistoryCursor, it.HasMoreHistory, true
		case it.Kind == "event" && it.Event != nil:
			switch t := it.Event.Type; {
			case t == "turn-item.updated":
				v.put(it.Event.Payload)
			case strings.HasPrefix(t, "run."):
				var r t3.Run
				if json.Unmarshal(it.Event.Payload, &r) == nil && r.ID != "" {
					old := v.runs[r.ID]
					mergeRun(&old, r)
					v.runs[r.ID] = old
				}
			case t == "message.updated":
				var m t3.Message
				if json.Unmarshal(it.Event.Payload, &m) == nil {
					v.messages[m.ID] = m
				}
			case t == "provider-turn.updated":
				var pt t3.ProviderTurn
				if json.Unmarshal(it.Event.Payload, &pt) == nil && pt.ID != "" {
					v.turns[pt.ID] = pt
				}
			case t == "provider-thread.updated":
				var pt t3.ProviderThread
				if json.Unmarshal(it.Event.Payload, &pt) == nil && pt.ID != "" {
					v.pthreads[pt.ID] = pt
				}
			case t == "provider-session.attached" || t == "provider-session.updated":
				var ps t3.ProviderSession
				if json.Unmarshal(it.Event.Payload, &ps) == nil && ps.ID != "" {
					v.sessions[ps.ID] = ps
				}
			case t == "runtime-request.updated":
				var rr t3.RuntimeRequestRecord
				if json.Unmarshal(it.Event.Payload, &rr) == nil && rr.ID != "" {
					v.requests[rr.ID] = rr
				}
			}
		}
		a.mu.Unlock()
		a.changed()
	}, func(err error) {
		if err != nil && !errors.Is(err, t3.ErrClosed) {
			a.mu.Lock()
			if a.view == v {
				a.fail(err)
			}
			a.mu.Unlock()
			a.changed()
		}
	})
	// Moves T3's read marker, so the thread stops showing as unread.
	go func() {
		_ = conn.Dispatch(map[string]any{"type": "thread.visit", "threadId": id, "visitedAt": time.Now().UTC().Format(time.RFC3339Nano), "opened": true})
	}()
}

func mergeRun(dst *t3.Run, src t3.Run) {
	dst.ID = src.ID
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.StartedAt != "" {
		dst.StartedAt = src.StartedAt
	}
	if src.CompletedAt != "" {
		dst.CompletedAt = src.CompletedAt
	}
	if src.UserMessageID != "" {
		dst.UserMessageID = src.UserMessageID
	}
	if src.Ordinal != 0 {
		dst.Ordinal = src.Ordinal
	}
	if src.RequestedAt != "" {
		dst.RequestedAt = src.RequestedAt
	}
	if src.ActiveAttemptID != "" {
		dst.ActiveAttemptID = src.ActiveAttemptID
	}
	dst.QueuePosition, dst.QueueHeld = src.QueuePosition, src.QueueHeld
}

func (a *App) loadEarlier() {
	a.mu.Lock()
	ok := a.loadEarlierLocked()
	a.mu.Unlock()
	if ok {
		a.changed()
	}
}

// loadEarlierLocked starts fetching the open thread's next page of history,
// unless there's none or it's on its way. Call with the lock held.
func (a *App) loadEarlierLocked() bool {
	v := a.view
	var conn *t3.Conn
	if v != nil {
		conn = v.env.conn
	}
	if v == nil || conn == nil || !v.more || v.loading || v.cursor == "" {
		return false
	}
	v.loading, v.historyErr = true, ""
	cursor := v.cursor
	go func() {
		page, err := conn.HistoryPage(v.id, cursor)
		a.mu.Lock()
		defer a.changed()
		defer a.mu.Unlock()
		v.loading = false
		if err != nil {
			// The history control says so under its button, as T3's does.
			v.historyErr = err.Error()
			return
		}
		for _, row := range page.Items {
			v.put(row.Item)
		}
		v.cursor, v.more = page.NextCursor, page.HasMoreHistory
	}()
	return true
}

// getTurnItem asks the server for a turn item in full. Tests replace it.
var getTurnItem = func(conn *t3.Conn, threadID, itemID string) (json.RawMessage, error) {
	var out struct {
		Item json.RawMessage `json:"item"`
	}
	err := conn.Call("orchestration.getTurnItem", map[string]any{"threadId": threadID, "itemId": itemID}, &out)
	return out.Item, err
}

// fetchTool fetches an open entry's full item when the socket left part
// of it out (T3's turnItemNeedsDetailFetch), once for each of its
// revisions: once while it runs and again when it's done. An entry
// opened afresh, or still open when its thread comes back, fetches the
// same way. Call with the lock held.
func (a *App) fetchTool(v *threadView, it *t3.TurnItem) {
	rev := detailRevision(it)
	if v.fullRev[it.ID] == rev || v.fetching[it.ID] == rev || v.env == nil || v.env.conn == nil {
		return
	}
	v.fetching[it.ID] = rev
	conn, id := v.env.conn, it.ID
	go func() {
		raw, err := getTurnItem(conn, v.id, id)
		a.mu.Lock()
		defer a.changed()
		defer a.mu.Unlock()
		if v.fetching[id] != rev {
			return // a later revision is on its way
		}
		delete(v.fetching, id)
		var full t3.TurnItem
		switch {
		case err != nil:
			// The open row says "Couldn't load output: …".
			v.full[id], v.fetchErr[id] = nil, err.Error()
		case len(raw) == 0 || string(raw) == "null":
			v.full[id], v.fetchErr[id] = nil, "Output is no longer available."
		case json.Unmarshal(raw, &full) == nil:
			v.full[id] = &full
			delete(v.fetchErr, id)
		default:
			v.full[id], v.fetchErr[id] = nil, "Output is no longer available."
		}
		v.fullRev[id] = rev
	}()
}

// fetched is the full item fetched for it's current revision, or why it
// couldn't load; nothing while that revision is on its way.
func (v *threadView) fetched(it *t3.TurnItem) (*t3.TurnItem, string) {
	if v.fullRev[it.ID] != detailRevision(it) {
		return nil, ""
	}
	return v.full[it.ID], v.fetchErr[it.ID]
}
