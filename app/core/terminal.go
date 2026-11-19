package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// T3's terminal drawer: shells on the thread's machine, in its folder, one
// set per thread. The terminal ids are the thread's own, term-1 and up, so
// T3's web app and tern show the same shells.

// drawer is a thread's terminals.
type drawer struct {
	open   bool
	terms  []*terminal
	active int
	next   int // the last term-N handed out
}

// terminal is one shell: the emulator its output runs through, the attach
// stream, and the queue of key presses going back.
type terminal struct {
	id     string
	label  string
	status string // running, exited, error
	busy   bool   // a command is running in it
	emu    *vt.SafeEmulator
	stop   func()
	input  chan func() // key presses, applied to the emulator in order
	cols   int
	rows   int
	// Where it lives. conn changes when the machine reconnects; read it
	// with the lock held.
	ref, cwd string
	conn     *t3.Conn
	threadID string
	attached bool   // the server has the shell: a resize before that is lost
	closed   bool   // the tab was closed: nothing reattaches it
	run      string // typed into the shell once it's there: a project script
}

const (
	termCols, termRows = 120, 24 // until the drawer has measured itself
)

// toggleTerminal opens or hides the open thread's drawer. Opening it the
// first time starts a shell.
func (a *App) toggleTerminal() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.openID == "" {
		return
	}
	d := a.drawers[a.openID]
	if d == nil {
		d = &drawer{}
		a.drawers[a.openID] = d
	}
	d.open = !d.open
	if d.open && len(d.terms) == 0 {
		if err := a.newTerminalLocked(d, ""); err != nil {
			d.open = false
			a.fail(err)
		}
	}
}

// newTerminal adds a shell to the open thread's drawer.
func (a *App) newTerminal() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if d := a.drawers[a.openID]; d != nil {
		if err := a.newTerminalLocked(d, ""); err != nil {
			a.fail(err)
		}
	}
}

// newTerminalLocked starts a shell in d, in the thread's folder, and types
// run into it once the server has it.
func (a *App) newTerminalLocked(d *drawer, run string) error {
	e, t := a.thread(a.openID)
	if t == nil {
		return errors.New("open a thread first")
	}
	if e.conn == nil {
		return errors.New(e.env.Label + " isn't connected")
	}
	cwd := t.WorktreePath
	if cwd == "" {
		cwd = e.projects[t.ProjectID].WorkspaceRoot
	}
	d.next++
	term := &terminal{id: fmt.Sprintf("term-%d", d.next), label: fmt.Sprintf("Terminal %d", d.next), status: "running",
		emu: vt.NewSafeEmulator(termCols, termRows), input: make(chan func(), 256), cols: termCols, rows: termRows}
	d.terms = append(d.terms, term)
	d.active = len(d.terms) - 1
	term.ref, term.cwd, term.conn, term.threadID, term.run = a.openID, cwd, e.conn, t.ID, run
	go term.pump(a)
	go func() {
		for f := range term.input {
			f()
		}
	}()
	a.attachLocked(term, false)
	return nil
}

// attachLocked streams the terminal from its machine. A stream that drops
// with the shell still open comes back once the machine reconnects,
// resuming the same shell.
func (a *App) attachLocked(term *terminal, resume bool) {
	conn := term.conn
	o := t3.TermOpen{ThreadID: term.threadID, TerminalID: term.id, Cwd: term.cwd, Cols: term.cols, Rows: term.rows, Restart: !resume, Resume: resume}
	term.stop = conn.TerminalAttach(o,
		func(ev t3.TermEvent) { a.termEvent(term, ev) },
		func(err error) {
			a.mu.Lock()
			reattach := err != nil && !term.closed && term.status == "running"
			a.mu.Unlock()
			if reattach {
				go a.reattach(term, conn)
			}
			a.changed()
		})
}

// reattach waits for the terminal's machine to connect again after old
// dropped, then resumes the stream.
func (a *App) reattach(term *terminal, old *t3.Conn) {
	for {
		time.Sleep(time.Second)
		a.mu.Lock()
		if term.closed {
			a.mu.Unlock()
			return
		}
		if e, t := a.thread(term.ref); t != nil && e.conn != nil && e.conn != old {
			term.conn = e.conn
			a.attachLocked(term, true)
			a.mu.Unlock()
			// The drawer may have changed size meanwhile.
			go e.conn.TerminalResize(term.threadID, term.id, term.cols, term.rows)
			return
		}
		a.mu.Unlock()
	}
}

// termEvent runs one attach event through the terminal.
func (a *App) termEvent(term *terminal, ev t3.TermEvent) {
	replay := func(s *t3.TermSnapshot) {
		// RIS resets the screen and modes; the history is then the screen
		// as the server has it.
		term.emu.Write([]byte("\x1bc"))
		term.emu.ClearScrollback()
		if s != nil {
			term.emu.Write([]byte(s.History))
		}
	}
	switch ev.Type {
	case "snapshot", "restarted":
		replay(ev.Snapshot)
		a.mu.Lock()
		if !term.attached {
			// The drawer may have measured itself before the shell existed.
			term.attached = true
			if cols, rows := term.cols, term.rows; cols != termCols || rows != termRows {
				go term.conn.TerminalResize(term.threadID, term.id, cols, rows)
			}
			if term.run != "" {
				go term.conn.TerminalWrite(term.threadID, term.id, term.run)
				term.run = ""
			}
		}
		if ev.Snapshot != nil {
			term.status = ev.Snapshot.Status
			if ev.Snapshot.Label != "" {
				term.label = ev.Snapshot.Label
			}
		}
		a.mu.Unlock()
	case "output":
		term.emu.Write([]byte(ev.Data))
	case "cleared":
		replay(nil)
	case "exited", "closed", "error":
		a.mu.Lock()
		term.status = map[string]string{"exited": "exited", "closed": "exited", "error": "error"}[ev.Type]
		a.mu.Unlock()
	case "activity":
		a.mu.Lock()
		term.busy = ev.Busy
		a.mu.Unlock()
	}
	a.changed()
}

// pump sends what the emulator writes back (key presses, answers to the
// shell's queries) to the server. Reads go on a channel at once, so the
// emulator never waits on the network.
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
		_ = conn.TerminalWrite(term.threadID, term.id, b.String())
	}
}

// closeTerminal ends the open thread's terminal i, and hides the drawer
// when it was the last.
func (a *App) closeTerminal(i int) {
	a.mu.Lock()
	d := a.drawers[a.openID]
	e, t := a.thread(a.openID)
	if d == nil || i < 0 || i >= len(d.terms) || t == nil {
		a.mu.Unlock()
		return
	}
	term := d.terms[i]
	d.terms = append(d.terms[:i:i], d.terms[i+1:]...)
	d.active = min(d.active, len(d.terms)-1)
	if len(d.terms) == 0 {
		d.open, d.active = false, 0
	}
	conn, threadID := e.conn, t.ID
	term.closed = true
	a.mu.Unlock()
	term.shut()
	if conn != nil {
		go conn.TerminalClose(threadID, term.id)
	}
}

// shut stops the stream and the goroutines behind it.
func (term *terminal) shut() {
	if term.stop != nil {
		term.stop()
	}
	close(term.input)
	term.emu.Close()
}

// termView is the open thread's drawer for the scene, with the controls
// the drawer calls back on. Call with the lock held.
func (a *App) termView() *ui.TermView {
	d := a.drawers[a.openID]
	if d == nil || !d.open || len(d.terms) == 0 {
		return nil
	}
	term := d.terms[d.active]
	v := &ui.TermView{Active: d.active, Screen: term.emu, Exited: term.status != "running" && term.status != "starting"}
	for _, tm := range d.terms {
		v.Tabs = append(v.Tabs, ui.TermTab{Label: tm.label, Busy: tm.busy, Exited: tm.status != "running" && tm.status != "starting"})
	}
	send := func(f func()) {
		defer func() { recover() }() // the terminal closed meanwhile
		select {
		case term.input <- f:
		default: // a full queue: drop the key rather than freeze the window
		}
	}
	v.Key = func(k uv.KeyPressEvent) { send(func() { term.emu.SendKey(k) }) }
	v.Text = func(s string) { send(func() { term.emu.SendText(s) }) }
	v.Paste = func(s string) { send(func() { term.emu.Paste(s) }) }
	v.Resize = func(cols, rows int) {
		a.mu.Lock()
		same := term.cols == cols && term.rows == rows
		term.cols, term.rows = cols, rows
		attached, conn, threadID := term.attached, term.conn, term.threadID
		a.mu.Unlock()
		if same {
			return
		}
		term.emu.Resize(cols, rows)
		if conn != nil && attached {
			go conn.TerminalResize(threadID, term.id, cols, rows)
		}
	}
	return v
}

// forgetTerminals ends every shell stream, on sign-out.
func (a *App) forgetTerminals() {
	for _, d := range a.drawers {
		for _, term := range d.terms {
			term.closed = true
			term.shut()
		}
	}
	a.drawers = map[string]*drawer{}
}
