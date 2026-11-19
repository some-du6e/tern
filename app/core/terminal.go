package core

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// T3's terminal drawer (docs/ui/09-terminal.md §5): each thread's shells
// on its machine, grouped into split panes as termState keeps them. A
// thread whose drawer is open keeps its shown panes attached, and so do
// the last ten threads left with their drawers open, so switching back
// finds them as they were. Terminal ids are the client's, term-1 and up,
// so T3's web app and tern show the same shells.

const termHiddenMax = 10 // MAX_HIDDEN_MOUNTED_TERMINAL_THREADS

// terminal is one attached terminal: the emulator its output runs
// through, the attach stream, and the queue of key presses going back.
type terminal struct {
	ref, id string // the thread's ref, and the terminal's id
	emu     *termEmu
	input   chan func() // key presses, applied to the emulator in order
	open    t3.TermOpen // the attach's payload
	conn    *t3.Conn    // read with the lock held: a reconnect changes it
	stop    func()
	replied bool // the stream has answered, so a retry resumes
	// T3's TerminalSessionState: the status ("closed" before the first
	// event), how many events have applied, and the attach generation,
	// which a snapshot after the first starts anew.
	status  string
	version int
	gen     int
	synced  string // the status the exit check last saw
	exiting bool   // it exited and is on its way out
	grid    [2]int // the server's shared grid, 0 until it says
	fit     [2]int // the drawer's last fit
	sentGen int    // the generation the fit was last sent for, -1 for none
	resize  *time.Timer
	gone    bool // detached: nothing reattaches it
}

func termKey(ref, id string) string { return ref + "\x00" + id }

// termEmu is a terminal's emulator, with the modes the drawer reads that
// x/vt keeps to itself: mouse tracking, application cursor keys and
// synchronized output; and a count of its writes.
type termEmu struct {
	*vt.SafeEmulator
	mouse  atomic.Uint32 // a bit for each of DEC 9, 1000, 1002 and 1003
	cursor atomic.Bool
	synced atomic.Bool
	ver    atomic.Uint64
}

var termMouseModes = []ansi.Mode{ansi.ModeMouseX10, ansi.ModeMouseNormal, ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent}

// newTermEmu is an emulator in T3's terminal colours, which it answers
// apps that ask for them with.
func newTermEmu(cols, rows int) *termEmu {
	e := &termEmu{SafeEmulator: vt.NewSafeEmulator(cols, rows)}
	e.SetDefaultForegroundColor(ui.P.TerminalForeground)
	e.SetDefaultBackgroundColor(ui.P.TerminalBackground)
	e.SetDefaultCursorColor(ui.P.TerminalCursor)
	set := func(m ansi.Mode, on bool) {
		if i := slices.Index(termMouseModes, m); i >= 0 {
			for {
				old := e.mouse.Load()
				bits := old &^ (1 << i)
				if on {
					bits |= 1 << i
				}
				if e.mouse.CompareAndSwap(old, bits) {
					break
				}
			}
		}
		switch m {
		case ansi.ModeCursorKeys:
			e.cursor.Store(on)
		case ansi.ModeSynchronizedOutput:
			e.synced.Store(on)
		}
	}
	e.SetCallbacks(vt.Callbacks{
		EnableMode:  func(m ansi.Mode) { set(m, true) },
		DisableMode: func(m ansi.Mode) { set(m, false) },
	})
	return e
}

func (e *termEmu) Write(p []byte) (int, error) {
	n, err := e.SafeEmulator.Write(p)
	e.ver.Add(1)
	return n, err
}

// tracking is the app's mouse tracking, its most reporting mode.
func (e *termEmu) tracking() ui.TermMouse {
	switch bits := e.mouse.Load(); {
	case bits&8 != 0:
		return ui.TermMouseMotion
	case bits&4 != 0:
		return ui.TermMouseDrag
	case bits != 0:
		return ui.TermMousePress
	}
	return ui.TermMouseOff
}

// reset puts the emulator back to a fresh screen with no history, then
// writes history: T3's resetAndWrite.
func (e *termEmu) reset(history string) {
	e.Write([]byte("\x1bc"))
	e.ClearScrollback()
	if history != "" {
		e.Write([]byte(history))
	}
}

// system writes one of T3's "[terminal] …" lines into the emulator only.
func (e *termEmu) system(msg string) { e.Write([]byte("\r\n[terminal] " + msg + "\r\n")) }

// termState is ref's drawer, the default when it has none. Call with the
// lock held.
func (a *App) termState(ref string) termState {
	if s, ok := a.termStates[ref]; ok {
		return s.normalized()
	}
	return termState{Height: termDefaultHeight}
}

// putTermState keeps ref's drawer, forgetting one back at its defaults,
// and saves them all, as T3's terminal-state store persists. Call with the
// lock held.
func (a *App) putTermState(ref string, s termState) {
	if s.isDefault() {
		delete(a.termStates, ref)
	} else {
		a.termStates[ref] = s.normalized()
	}
	_ = a.store.Put("terminal-state", a.termStates)
}

// unhide lets id back into ref's drawer: New, split and reuse take it out
// of the closed ids a lagging server list could otherwise bring back.
func (a *App) unhide(ref, id string) {
	a.termHidden[ref] = slices.DeleteFunc(a.termHidden[ref], func(x string) bool { return x == id })
}

// knownTerms is the server's terminals for ref's thread, in id order.
// Call with the lock held.
func (a *App) knownTerms(ref string) []t3.TermSummary {
	e, t := a.thread(ref)
	if t == nil {
		return nil
	}
	var out []t3.TermSummary
	for _, s := range e.terms {
		if s.ThreadID == t.ID {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return naturalLess(out[i].TerminalID, out[j].TerminalID) })
	return out
}

// shownGroup is the group whose panes show, by its place in the list: the
// active group, else the active terminal's, else the first.
func shownGroup(s termState) int {
	groups := s.shownGroups()
	if i := groupIndex(groups, s.ActiveGroup); i >= 0 {
		return i
	}
	return max(0, groupOf(groups, s.Active))
}

// shownIDs is the terminals whose panes show.
func shownIDs(s termState) []string {
	if groups, i := s.shownGroups(), shownGroup(s); i < len(groups) {
		return groups[i].IDs
	}
	return nil
}

// remountTerminalsLocked works out which drawers are mounted, as T3's
// ChatTerminalDrawers does: the open thread's while it's open, and the
// last ten others left open. Each takes the server's list where it has
// moved on; then the shown panes attach and the rest detach. Call with
// the lock held.
func (a *App) remountTerminalsLocked() {
	var mounted []string
	for _, r := range a.termMounted {
		if _, t := a.thread(r); r != a.openID && t != nil && a.termState(r).Open {
			mounted = append(mounted, r)
		}
	}
	mounted = mounted[max(0, len(mounted)-termHiddenMax):]
	if a.openID != "" && a.termState(a.openID).Open {
		mounted = append(mounted, a.openID)
	}
	a.termMounted = mounted
	want := map[string]bool{}
	for _, r := range mounted {
		a.reconcileTermsLocked(r)
		for _, id := range shownIDs(a.termState(r)) {
			k := termKey(r, id)
			want[k] = true
			if a.terms[k] == nil {
				a.attachTerminalLocked(r, id)
			}
		}
	}
	for k, term := range a.terms {
		if !want[k] {
			a.detachLocked(term)
		}
	}
}

// reconcileTermsLocked takes the server's list of ref's terminals into
// its drawer when they differ, unless the server's is a strict subset of
// the drawer's, as it is while a new shell starts. Ids the drawer closed
// stay out. Call with the lock held.
func (a *App) reconcileTermsLocked(ref string) {
	e, _ := a.thread(ref)
	if e == nil || !e.termsKnown {
		return
	}
	var server []string
	for _, s := range a.knownTerms(ref) {
		server = append(server, s.TerminalID)
	}
	s := a.termState(ref)
	if sameIDs(server, s.IDs) || strictSubset(server, s.IDs) {
		return
	}
	hidden := a.termHidden[ref]
	server = slices.DeleteFunc(server, func(id string) bool { return slices.Contains(hidden, id) })
	if !slices.Equal(server, s.IDs) {
		a.putTermState(ref, s.reconcile(server))
	}
}

func sameIDs(a, b []string) bool {
	return len(a) == len(b) && !slices.ContainsFunc(a, func(id string) bool { return !slices.Contains(b, id) })
}

func strictSubset(a, b []string) bool {
	return len(a) < len(b) && !slices.ContainsFunc(a, func(id string) bool { return !slices.Contains(b, id) })
}

// termLaunch is where ref's terminal id runs: where the server says it
// does, else the thread's worktree or its project's root; and the
// environment T3 gives it. cwd is "" without a project. Call with the lock
// held.
func (a *App) termLaunch(ref, id string) (cwd, worktree string, env map[string]string) {
	e, t := a.thread(ref)
	if t == nil {
		return "", "", nil
	}
	root := e.projects[t.ProjectID].WorkspaceRoot
	if root == "" {
		return "", "", nil
	}
	cwd, worktree = t.WorktreePath, t.WorktreePath
	if cwd == "" {
		cwd = root
	}
	for _, s := range a.knownTerms(ref) {
		if s.TerminalID == id {
			cwd, worktree = s.Cwd, s.WorktreePath
		}
	}
	env = map[string]string{"T3CODE_PROJECT_ROOT": root}
	if worktree != "" {
		env["T3CODE_WORKTREE_PATH"] = worktree
	}
	return cwd, worktree, env
}

// attachTerminalLocked attaches ref's terminal id: an emulator, the pump
// sending its keys, and the attach stream, which starts the shell if the
// server has none. Call with the lock held.
func (a *App) attachTerminalLocked(ref, id string) {
	e, t := a.thread(ref)
	cwd, worktree, env := a.termLaunch(ref, id)
	if t == nil || e.conn == nil || cwd == "" {
		return
	}
	term := &terminal{ref: ref, id: id, emu: newTermEmu(120, 30), input: make(chan func(), 256), conn: e.conn,
		status: "closed", synced: "closed", sentGen: -1,
		open: t3.TermOpen{ThreadID: t.ID, TerminalID: id, Cwd: cwd, WorktreePath: worktree, Env: env}}
	a.terms[termKey(ref, id)] = term
	go term.pump(a)
	go func() {
		for f := range term.input {
			f()
		}
	}()
	a.streamTerminalLocked(term)
}

// streamTerminalLocked runs term's attach stream. One that drops comes
// back once the machine reconnects, resuming without restarting. Call
// with the lock held.
func (a *App) streamTerminalLocked(term *terminal) {
	o := term.open
	o.Resume = term.replied
	conn := term.conn
	term.stop = conn.TerminalAttach(o, func(ev t3.TermEvent) { a.termEvent(term, ev) }, func(err error) {
		a.mu.Lock()
		gone := term.gone
		a.mu.Unlock()
		switch {
		case err == nil || gone:
		case errors.Is(err, t3.ErrClosed):
			go a.reattach(term, conn)
		default:
			// The server refused it: say why in the terminal, as T3 does.
			term.emu.system(err.Error())
			a.changed()
		}
	})
}

// reattach waits for the terminal's machine to connect again after old
// dropped, then resumes the stream.
func (a *App) reattach(term *terminal, old *t3.Conn) {
	for {
		time.Sleep(time.Second)
		a.mu.Lock()
		if term.gone {
			a.mu.Unlock()
			return
		}
		if e, t := a.thread(term.ref); t != nil && e.conn != nil && e.conn != old {
			term.conn = e.conn
			a.streamTerminalLocked(term)
			a.mu.Unlock()
			return
		}
		a.mu.Unlock()
	}
}

// termEvent runs one attach event through the terminal, as T3's
// applyTerminalAttachStreamEvent does, then the exit check and the size
// the server should hear of.
func (a *App) termEvent(term *terminal, ev t3.TermEvent) {
	a.mu.Lock()
	if term.gone {
		a.mu.Unlock()
		return
	}
	term.replied = true
	grid := func(cols, rows int) {
		if cols > 0 && rows > 0 {
			term.grid = [2]int{cols, rows}
		}
	}
	history, reset, msg := "", false, ""
	switch ev.Type {
	case "snapshot", "restarted":
		if s := ev.Snapshot; s != nil {
			// A snapshot after the first is a new attachment, after a
			// reconnect: a new generation, which hears the size again.
			if ev.Type == "snapshot" && term.version > 0 {
				term.gen++
			}
			history, reset, term.status = s.History, true, s.Status
			grid(s.Cols, s.Rows)
		}
	case "output":
		if term.status == "closed" {
			term.status = "running"
		}
	case "cleared":
		reset = true
	case "exited", "closed":
		term.status = ev.Type
	case "error":
		term.status, msg = "error", ev.Message
	case "activity":
		grid(ev.Cols, ev.Rows)
	}
	if ev.Type != "activity" {
		term.version++
	}
	cols, rows := term.drawn()
	exit := a.termExitLocked(term)
	if term.version > 0 && term.status == "running" && term.sentGen != term.gen {
		term.sentGen = term.gen
		a.notifyResizeLocked(term)
	}
	a.mu.Unlock()
	if cols > 0 && (cols != term.emu.Width() || rows != term.emu.Height()) {
		term.emu.Resize(cols, rows)
	}
	switch {
	case reset:
		term.emu.reset(history)
	case ev.Type == "output":
		term.emu.Write([]byte(ev.Data))
	}
	if msg != "" {
		term.emu.system(msg)
	}
	if exit != "" {
		term.emu.system(exit)
		go a.terminalExited(term)
	}
	a.changed()
}

// termExitLocked is T3's shouldHandleTerminalExit: a terminal it has
// heard from that goes exited or closed says so once, and then goes. Call
// with the lock held.
func (a *App) termExitLocked(term *terminal) string {
	msg := ""
	switch {
	case term.version > 0 && term.status == "running":
		term.exiting = false
	case term.version > 0 && (term.status == "exited" || term.status == "closed") && term.status != term.synced && !term.exiting:
		term.exiting = true
		msg = "Process exited"
		if term.status == "closed" {
			msg = "Terminal closed"
		}
	}
	if term.version > 0 {
		term.synced = term.status
	}
	return msg
}

// terminalExited closes an exited terminal on its machine, with its
// history, and takes it out of the drawer.
func (a *App) terminalExited(term *terminal) {
	a.mu.Lock()
	if !term.gone && term.exiting {
		a.closeTerminalLocked(term.ref, term.id)
	}
	a.mu.Unlock()
	a.changed()
}

// drawn is the grid the terminal draws: the drawer's fit, but no wider or
// taller than the server's shared one, so every viewer wraps alike.
func (term *terminal) drawn() (cols, rows int) {
	cols, rows = term.fit[0], term.fit[1]
	if term.grid[0] > 0 {
		cols, rows = min(cols, term.grid[0]), min(rows, term.grid[1])
	}
	return cols, rows
}

// fitTerminal takes the drawer's fit for term: the emulator follows at
// once, the server 150ms after the last change.
func (a *App) fitTerminal(term *terminal, cols, rows int) {
	a.mu.Lock()
	if term.fit == [2]int{cols, rows} || term.gone {
		a.mu.Unlock()
		return
	}
	term.fit = [2]int{cols, rows}
	cols, rows = term.drawn()
	a.notifyResizeLocked(term)
	a.mu.Unlock()
	if cols != term.emu.Width() || rows != term.emu.Height() {
		term.emu.Resize(cols, rows)
	}
}

// notifyResizeLocked sends term's fit to its machine in 150ms, unless it
// changes again first. Call with the lock held.
func (a *App) notifyResizeLocked(term *terminal) {
	if term.resize != nil {
		term.resize.Stop()
	}
	term.resize = time.AfterFunc(150*time.Millisecond, func() { a.sendResize(term) })
}

// sendResize tells the machine the fit, for a running terminal that has
// attached; any other fit is replayed when it does.
func (a *App) sendResize(term *terminal) {
	a.mu.Lock()
	ok := !term.gone && term.version > 0 && term.status == "running" && term.fit[0] > 0
	conn, threadID, cols, rows := term.conn, term.open.ThreadID, term.fit[0], term.fit[1]
	a.mu.Unlock()
	if ok {
		_ = conn.TerminalResize(threadID, term.id, cols, rows)
	}
}

// settleTerminals sends the open drawer's pending fits now, as T3 does on
// a pointer release that ends a drag.
func (a *App) settleTerminals() {
	a.mu.Lock()
	var due []*terminal
	for _, term := range a.terms {
		if term.ref == a.openID && term.resize != nil && term.resize.Stop() {
			due = append(due, term)
		}
	}
	a.mu.Unlock()
	for _, term := range due {
		go a.sendResize(term)
	}
}

// pump sends what the emulator writes back (key presses, answers to the
// shell's queries) to the server. Reads go on a channel at once, so the
// emulator never waits on the network. A write that fails says so in the
// terminal.
func (term *terminal) pump(a *App) {
	out := make(chan string, 256)
	go func() {
		defer close(out)
		buf := make([]byte, 4096)
		for {
			n, err := term.emu.Read(buf)
			if n > 0 {
				out <- string(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	for s := range out {
		// Send what piled up while the last write was out as one.
		var b strings.Builder
		b.WriteString(s)
	more:
		for b.Len() < 32<<10 {
			select {
			case s, ok := <-out:
				if !ok {
					break more
				}
				b.WriteString(s)
			default:
				break more
			}
		}
		a.mu.Lock()
		conn := term.conn
		a.mu.Unlock()
		if err := conn.TerminalWrite(term.open.ThreadID, term.id, b.String()); err != nil {
			term.emu.system(err.Error())
			a.changed()
		}
	}
}

// detachLocked ends term's stream; its shell runs on. Call with the lock
// held.
func (a *App) detachLocked(term *terminal) {
	delete(a.terms, termKey(term.ref, term.id))
	term.gone = true
	if term.resize != nil {
		term.resize.Stop()
	}
	go term.shut()
}

// shut stops the stream and the goroutines behind it.
func (term *terminal) shut() {
	if term.stop != nil {
		term.stop()
	}
	close(term.input)
	term.emu.Close()
}

// openDrawerRef is the open thread's ref when it can have a drawer: a
// thread with a project. Call with the lock held.
func (a *App) openDrawerRef() string {
	if cwd, _, _ := a.termLaunch(a.openID, ""); cwd == "" {
		return ""
	}
	return a.openID
}

// toggleTerminal opens or closes the open thread's drawer
// (useChatTerminalActions.ts:143–170). Opening an empty one reuses a
// terminal the server already has for the thread, one running a command
// first, before it starts a shell. Closing leaves the shells running.
func (a *App) toggleTerminal(th *ui.Theme) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ref := a.openDrawerRef()
	if ref == "" {
		return
	}
	s := a.termState(ref)
	switch {
	case s.Open:
		a.putTermState(ref, s.setOpen(false))
		th.Focus("composer")
	case len(s.IDs) > 0:
		a.putTermState(ref, s.setOpen(true))
		th.FocusTerminal()
	default:
		known := a.knownTerms(ref)
		reuse := ""
		if i := slices.IndexFunc(known, func(k t3.TermSummary) bool { return k.Busy }); i >= 0 {
			reuse = known[i].TerminalID
		} else if len(known) > 0 {
			reuse = known[0].TerminalID
		}
		if reuse != "" {
			a.unhide(ref, reuse)
			a.putTermState(ref, s.ensure(reuse, true))
		} else {
			a.launchTerminalLocked(ref, func(s termState, id string) termState { return s.ensure(id, true) })
		}
		th.FocusTerminal()
	}
	a.remountTerminalsLocked()
}

// launchTerminalLocked starts a terminal in ref's drawer, as T3's
// launchTerminal does: the next free id goes in first, through place, so
// its pane attaches at once; then terminal.open starts the shell, in the
// thread's worktree or project root. Call with the lock held.
func (a *App) launchTerminalLocked(ref string, place func(termState, string) termState) {
	e, t := a.thread(ref)
	if t == nil || e.conn == nil {
		return
	}
	ids := slices.Clone(a.termState(ref).IDs)
	for _, k := range a.knownTerms(ref) {
		ids = append(ids, k.TerminalID)
	}
	id := nextTermID(ids)
	cwd, worktree, env := a.termLaunch(ref, id)
	if cwd == "" {
		return
	}
	a.unhide(ref, id)
	a.putTermState(ref, place(a.termState(ref), id))
	conn := e.conn
	o := t3.TermOpen{ThreadID: t.ID, TerminalID: id, Cwd: cwd, WorktreePath: worktree, Env: env}
	a.termLife(ref, func() {
		if err := conn.TerminalOpen(o); err != nil {
			a.mu.Lock()
			a.fail(err)
			a.mu.Unlock()
			a.changed()
		}
	})
}

// termQueue is a thread's terminal opens and closes waiting their turn.
type termQueue struct {
	jobs    []func()
	running bool
}

// termLife runs ref's terminal opens and closes one after another, in the
// order they were asked for, as T3's lifecycle scheduler does: a close
// never overtakes the open before it. Call with the lock held.
func (a *App) termLife(ref string, f func()) {
	q := a.termOps[ref]
	if q == nil {
		q = &termQueue{}
		a.termOps[ref] = q
	}
	q.jobs = append(q.jobs, f)
	if q.running {
		return
	}
	q.running = true
	go func() {
		for {
			a.mu.Lock()
			if len(q.jobs) == 0 {
				q.running = false
				a.mu.Unlock()
				return
			}
			f := q.jobs[0]
			q.jobs = q.jobs[1:]
			a.mu.Unlock()
			f()
		}
	}()
}

// newTerminal adds a terminal to the open drawer: in a group of its own,
// or split into the shown group, side by side or stacked. A group at four
// takes no more.
func (a *App) newTerminal(th *ui.Theme, split, vertical bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ref := a.openDrawerRef()
	if ref == "" {
		return
	}
	if split && len(shownIDs(a.termState(ref))) >= termGroupMax {
		return
	}
	a.launchTerminalLocked(ref, func(s termState, id string) termState { return s.add(id, split, vertical) })
	a.remountTerminalsLocked()
	th.FocusTerminal()
}

// activateTerminal makes id the open drawer's active terminal, shows its
// group, and gives it the keyboard.
func (a *App) activateTerminal(th *ui.Theme, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ref := a.openDrawerRef(); ref != "" {
		a.putTermState(ref, a.termState(ref).activate(id))
		a.remountTerminalsLocked()
		th.FocusTerminal()
	}
}

// askCloseTerminal asks before closing the open drawer's terminal id, or
// its active one; T3 always asks. While it asks, more close chords do
// nothing.
func (a *App) askCloseTerminal(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ref := a.openDrawerRef()
	s := a.termState(ref)
	if ref == "" || a.termClose != nil || !s.Open {
		return
	}
	if id == "" {
		id = s.Active
	}
	if slices.Contains(s.IDs, id) {
		a.termClose = &termClosing{ref: ref, id: id}
	}
}

// termClosing is the terminal the close confirmation asks about.
type termClosing struct{ ref, id string }

// answerCloseTerminal closes the terminal the confirmation asked about, or
// leaves it, and gives the keyboard back to the drawer.
func (a *App) answerCloseTerminal(th *ui.Theme, yes bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.termClose
	a.termClose = nil
	if c != nil && yes {
		a.closeTerminalLocked(c.ref, c.id)
	}
	th.FocusTerminal()
}

// closeTerminalLocked ends ref's terminal id on its machine, history and
// all (writing exit if the close fails), and takes it out of the drawer;
// the server's list may not bring it back. Call with the lock held.
func (a *App) closeTerminalLocked(ref, id string) {
	if e, t := a.thread(ref); t != nil && e.conn != nil {
		conn, threadID := e.conn, t.ID
		a.termLife(ref, func() {
			if conn.TerminalClose(threadID, id) != nil {
				_ = conn.TerminalWrite(threadID, id, "exit\n")
			}
		})
	}
	a.unhide(ref, id)
	a.termHidden[ref] = append(a.termHidden[ref], id)
	a.putTermState(ref, a.termState(ref).close(id))
	a.remountTerminalsLocked()
}

// setTerminalHeight keeps the open drawer's dragged height.
func (a *App) setTerminalHeight(h float32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ref := a.openDrawerRef(); ref != "" {
		a.putTermState(ref, a.termState(ref).setHeight(h))
	}
}

// openTerminalLink opens a link clicked in a terminal in the system's
// browser, T3's "open links in" default.
func (a *App) openTerminalLink(url string) {
	if err := openExternal(url); err != nil {
		a.mu.Lock()
		a.fail(err)
		a.mu.Unlock()
		a.changed()
	}
}

// openExternal opens a web address in the system's browser, as T3's
// desktop app hands links to shell.openExternal. Tests replace it.
var openExternal = func(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// subscribeTerminals follows the environment's terminals, for the drawer's
// labels, the shells it reuses, the ids it hands out and the list it
// reconciles with.
func (a *App) subscribeTerminals(e *envState, conn *t3.Conn, gen int) {
	conn.TerminalMetadata(func(ev t3.TermMetaEvent) {
		a.mu.Lock()
		if gen != e.gen {
			a.mu.Unlock()
			return
		}
		same := func(thread, id string) func(t3.TermSummary) bool {
			return func(s t3.TermSummary) bool { return s.ThreadID == thread && s.TerminalID == id }
		}
		switch ev.Type {
		case "snapshot":
			e.terms, e.termsKnown = ev.Terminals, true
		case "upsert":
			if s := ev.Terminal; s != nil {
				e.terms = append(slices.DeleteFunc(e.terms, same(s.ThreadID, s.TerminalID)), *s)
			}
		case "remove":
			e.terms = slices.DeleteFunc(e.terms, same(ev.ThreadID, ev.TerminalID))
		}
		a.remountTerminalsLocked()
		a.mu.Unlock()
		a.changed()
	}, func(error) {})
}

// drawerShown reports whether the open thread's drawer is up. Call with
// the lock held.
func (a *App) drawerShown() bool {
	ref := a.openDrawerRef()
	return ref != "" && slices.Contains(a.termMounted, ref)
}

// termView is the open thread's drawer for the scene, with the controls
// its panes call back on. Call with the lock held.
func (a *App) termView() *ui.TermView {
	if !a.drawerShown() {
		return nil
	}
	ref := a.openID
	s := a.termState(ref)
	labels := map[string]string{}
	for _, k := range a.knownTerms(ref) {
		if l := strings.TrimSpace(k.Label); l != "" {
			labels[k.TerminalID] = l
		}
	}
	label := func(id string) string {
		if l := labels[id]; l != "" {
			return l
		}
		return termLabel(id)
	}
	v := &ui.TermView{Thread: ref, Height: s.Height, Active: s.Active, Shown: shownGroup(s), Settle: a.settleTerminals}
	for _, g := range s.shownGroups() {
		tg := ui.TermGroup{Vertical: g.Vertical}
		for _, id := range g.IDs {
			tg.Tabs = append(tg.Tabs, ui.TermTab{ID: id, Label: label(id)})
		}
		v.Groups = append(v.Groups, tg)
	}
	for _, id := range shownIDs(s) {
		v.Panes = append(v.Panes, a.termPane(ref, id, label(id)))
	}
	if c := a.termClose; c != nil && c.ref == ref {
		v.Confirm = &ui.TermConfirm{Title: fmt.Sprintf("Close terminal %q?", label(c.id)), Description: "This stops the running process and clears its history."}
	}
	return v
}

// termPane is one shown terminal for the scene: its emulator once it has
// attached, and where its input goes.
func (a *App) termPane(ref, id, label string) ui.TermPane {
	p := ui.TermPane{ID: id, Label: label}
	term := a.terms[termKey(ref, id)]
	if term == nil {
		return p
	}
	p.Screen, p.Ver = term.emu, term.emu.ver.Load()
	p.Mouse, p.AppCursor, p.Paused = term.emu.tracking(), term.emu.cursor.Load(), term.emu.synced.Load()
	send := func(f func()) {
		defer func() { recover() }() // the terminal closed meanwhile
		select {
		case term.input <- f:
		default: // a full queue: drop the key rather than freeze the window
		}
	}
	p.Key = func(k uv.KeyPressEvent) { send(func() { term.emu.SendKey(k) }) }
	p.Send = func(s string) { send(func() { term.emu.SendText(s) }) }
	p.Paste = func(s string) { send(func() { term.emu.Paste(s) }) }
	p.Report = func(m uv.MouseEvent) { send(func() { term.emu.SendMouse(m) }) }
	p.Fit = func(cols, rows int) { a.fitTerminal(term, cols, rows) }
	return p
}

// forgetTerminals ends every terminal stream, on sign-out.
func (a *App) forgetTerminals() {
	for _, term := range a.terms {
		a.detachLocked(term)
	}
	a.termMounted, a.termClose = nil, nil
}
