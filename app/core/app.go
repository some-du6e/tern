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
	operate   bool      // the session may change threads, not just read them
}

func newEnv(e t3.Environment, paired bool) *envState {
	return &envState{env: e, status: "offline", paired: paired, projects: map[string]t3.Project{}, threads: map[string]*t3.Thread{}, operate: true}
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
	// OpenURL opens a URL on this computer: a pull request's, in its
	// browser.
	OpenURL func(string) error

	mu       sync.Mutex
	booted   bool // boot has decided between sign-in and the inbox
	signedIn bool
	login    ui.SignIn
	signIn   *t3.SignIn

	envs []*envState
	net  string

	openID   string // ref of the open thread
	visit    visits // its read marker
	tabs     []string
	view     *threadView
	draft    string // project id while composing a new thread
	draftEnv string // and its environment

	choices map[string]*choice       // per thread ref, or "draft"
	images  map[string][]pastedImage // pasted and not sent yet, same keys
	trash   map[string][]pastedImage // their chips were deleted; undo brings them back
	sent    map[string]*sentThumb    // sent pictures' thumbnails, by env/attachment id

	// The sidebar: the search, its highlighted result's ref ("" for the
	// first) and the results' refs; the project scope ("<env>/<project>"
	// keys) and the filter's popup; which shelves are open and how many
	// Settled pages show; the multi-selection and the row a Shift range
	// starts from; the row being renamed; the card whose snooze menu is
	// open. working and returned hold threads on the Working shelf past the
	// end of their work and note when they left it; ordered is the list's
	// rows top to bottom, for the keyboard and the jump keys.
	search      string
	searchHi    string
	results     []string
	scope       []string
	scopeDirty  bool       // changed since it was last saved
	scopeSave   sync.Mutex // held while saving it, outside mu
	filterOpen  bool
	filterQuery string
	filterKeys  []string
	filterShown []string // the popup's rows, "" for All projects
	filterHi    int      // its highlighted row, -1 for the scope's own
	shelfOpen   map[string]bool
	settledMore int
	selected    map[string]bool
	anchor      string
	renamingRow string
	snoozeRef   string
	snoozeMenu  []snoozeChoice
	working     map[string]workingState
	returned    map[string]time.Time
	ordered     []string
	openRow     string          // the open thread's row: its nearest listed ancestor's for a subagent or side chat
	parked      map[string]bool // the Snoozed and Settled rows, which advancing skips
	timer       *time.Timer     // a scene due when the sidebar next changes by itself: a hold, a wake, a grace
	timerAt     time.Time       // when it's due
	draftHome   string          // the new thread's draft, "<env>/<project>", kept while it isn't open
	ctxDraft    bool            // the open menu is the draft row's
	// The footer's undo notice: the parks and discards it can still undo,
	// oldest first; the timer that ends it 5s after the last one, and how
	// many have joined it, which tells the last one's timer from the ones
	// before; and whether mod+Z undoes it even in a text field, until
	// another key.
	undos     []undoEntry
	undoTimer *time.Timer
	undoGen   int
	undoClaim bool
	// The servers' message search: the one they were last asked, the one
	// before it that last settled, shown while the next refines it, the
	// timer that waits for the typing to stop, and the order the message
	// results were last shown in, for the query they were shown for.
	msgSearch   msgSearch
	msgSettled  msgSearch
	msgTimer    *time.Timer
	msgOrder    []string
	msgOrderFor string

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
	ctxBulk       bool                // the open menu is the multi-selection's
	ctxKeyboard   bool                // it was opened from the keyboard
	ctxPresets    []snoozeChoice      // its snooze presets, as of when it opened
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
		selected: map[string]bool{},
		working:  map[string]workingState{},
		returned: map[string]time.Time{},
	}
	_ = store.Delete("env-filter") // from when a machine's chip filtered the inbox
	a.detailsCard = true
	_, _ = store.Get("details-card", &a.detailsCard)
	_, _ = store.Get("sidebar-hidden", &a.sidebarHidden)
	_, _ = store.Get("sidebar-width", &a.sidebarWidth)
	// The shelves start folded, as T3's do, and remember being opened.
	a.shelfOpen = map[string]bool{}
	_, _ = store.Get("sidebar-shelves", &a.shelfOpen)
	_, _ = store.Get("sidebar-scope", &a.scope)
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
		e.operate = conn.Session().CanOperate()
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

// HIDDEN carries nothing worth a row of its own.
var hidden = map[string]bool{"checkpoint": true, "notification": true, "approval_request": true, "user_input_request": true, "run_interrupt_request": true}

var toolTypes = map[string]bool{"command_execution": true, "file_change": true, "file_search": true, "web_search": true, "dynamic_tool": true, "subagent": true}

type threadView struct {
	id       string // the thread's own id
	env      *envState
	raw      map[string]map[string]json.RawMessage // row key -> merged item fields
	items    map[string]*t3.TurnItem
	runs     map[string]t3.Run
	messages map[string]t3.Message
	cursor   string
	more     bool
	loading  bool
	loaded   bool
	unfolded map[string]bool
	open     map[string]bool // tool and thought rows the user expanded
	full     map[string]*t3.TurnItem
	cancel   func()
}

func newView(e *envState, id string) *threadView {
	return &threadView{
		id: id, env: e, raw: map[string]map[string]json.RawMessage{}, items: map[string]*t3.TurnItem{},
		runs: map[string]t3.Run{}, messages: map[string]t3.Message{},
		unfolded: map[string]bool{}, open: map[string]bool{}, full: map[string]*t3.TurnItem{},
	}
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
	// A thread the project filter hides widens it to every project, so its
	// row shows (resolveSidebarRouteReveal).
	if scope := a.scopeSet(); scope != nil {
		if e, t := a.thread(a.rowThread(id)); t != nil && listed(t) && t.ArchivedAt == "" && !scope[ref(e.id(), t.ProjectID)] {
			a.setScope(nil)
		}
	}
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
		a.view = newView(e, id)
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
}

// ---------- visits ----------

// visitThrottle is how often the open thread's visit goes out while a run
// streams (VISIT_DISPATCH_THROTTLE_MS): each of its updates moves the
// thread's updatedAt.
const visitThrottle = 10 * time.Second

// visits is the open thread's read marker as T3's tracker keeps it while
// the thread is shown (createThreadVisitTracker): the thread, whether its
// open is yet to go with the first visit, the marker last sent (or found
// already there) and when the last visit was sent.
type visits struct {
	ref  string
	open bool
	sent time.Time
	at   time.Time
}

// next is the visit to send for t, v's thread, as of now: the marker, ""
// for none, and whether the open goes with it. Each update the user is
// looking at moves the marker to the thread's updatedAt, at most every 10s
// while a run streams, but at once for a run that ended unseen, so a run
// that finished as they watched isn't left saying Done once they leave.
// retry is when a visit the throttle held back is due.
func (v *visits) next(t *t3.Thread, now time.Time) (at string, opened bool, retry time.Time) {
	updated := parseTime(t.UpdatedAt)
	if updated.IsZero() {
		return "", false, time.Time{}
	}
	send := func(opened bool) (string, bool, time.Time) {
		v.sent, v.at = updated, now
		return t.UpdatedAt, opened, time.Time{}
	}
	if v.open {
		v.open = false
		return send(true)
	}
	seen := parseTime(t.LastVisitedAt)
	if !seen.IsZero() && !seen.Before(updated) {
		// Seen already, so a mark-unread after it stays put.
		if v.sent.Before(updated) {
			v.sent = updated
		}
		return "", false, time.Time{}
	}
	if !v.sent.Before(updated) {
		return "", false, time.Time{}
	}
	done := latestRunDone(t)
	if !done.IsZero() && (seen.IsZero() || done.After(seen)) || now.Sub(v.at) >= visitThrottle {
		return send(false)
	}
	return "", false, v.at.Add(visitThrottle)
}

// trackVisit sends the open thread's visits (ThreadVisitWatermark), which
// move T3's read marker so it stops showing as unread. Call with the lock
// held.
func (a *App) trackVisit(now time.Time) {
	if a.visit.ref != a.openID {
		a.visit = visits{ref: a.openID, open: true}
	}
	e, t := a.thread(a.openID)
	if e == nil || t == nil || e.conn == nil {
		// The open waits for the thread and its machine.
		return
	}
	at, opened, retry := a.visit.next(t, now)
	if !retry.IsZero() {
		a.redrawAt(retry)
	}
	if at == "" {
		return
	}
	cmd := visit(t.ID, at)
	if opened {
		cmd["opened"] = true
	}
	conn := e.conn
	go func() { _ = conn.Dispatch(cmd) }()
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
	dst.QueuePosition, dst.QueueHeld = src.QueuePosition, src.QueueHeld
}

func (a *App) loadEarlier() {
	a.mu.Lock()
	v := a.view
	var conn *t3.Conn
	if v != nil {
		conn = v.env.conn
	}
	if v == nil || conn == nil || !v.more || v.loading || v.cursor == "" {
		a.mu.Unlock()
		return
	}
	v.loading = true
	cursor := v.cursor
	a.mu.Unlock()
	a.changed()
	go func() {
		page, err := conn.HistoryPage(v.id, cursor)
		a.mu.Lock()
		defer a.changed()
		defer a.mu.Unlock()
		v.loading = false
		if err != nil {
			a.fail(err)
			return
		}
		for _, row := range page.Items {
			v.put(row.Item)
		}
		v.cursor, v.more = page.NextCursor, page.HasMoreHistory
	}()
}

// loadTool fetches a tool row's full item: the socket strips command output
// and diffs.
func (a *App) loadTool(item *t3.TurnItem) {
	a.mu.Lock()
	v := a.view
	var conn *t3.Conn
	if v != nil {
		conn = v.env.conn
	}
	if v == nil || conn == nil || !toolTypes[item.Type] {
		a.mu.Unlock()
		return
	}
	if f := v.full[item.ID]; f != nil && f.UpdatedAt == item.UpdatedAt {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()
	go func() {
		var out struct {
			Item json.RawMessage `json:"item"`
		}
		err := conn.Call("orchestration.getTurnItem", map[string]any{"threadId": v.id, "itemId": item.ID}, &out)
		a.mu.Lock()
		defer a.changed()
		defer a.mu.Unlock()
		if err != nil {
			a.fail(err)
			return
		}
		var full t3.TurnItem
		if json.Unmarshal(out.Item, &full) == nil {
			full.UpdatedAt = item.UpdatedAt
			v.full[item.ID] = &full
		}
	}()
}
