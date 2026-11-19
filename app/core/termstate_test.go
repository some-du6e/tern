package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// termTestApp is testApp with the swiss thread t1 open, which has a
// project, so it can have a drawer.
func termTestApp(t *testing.T) (*App, string) {
	t.Helper()
	a := testApp(t)
	ref := "swiss/t1"
	a.openThread(ref)
	return a, ref
}

// shown is what the drawer shows: each group's labels, the shown group
// starred, the active terminal in brackets.
func shown(v *ui.TermView) string {
	if v == nil {
		return "closed"
	}
	var groups []string
	for i, g := range v.Groups {
		var tabs []string
		for _, tab := range g.Tabs {
			l := tab.Label
			if tab.ID == v.Active {
				l = "[" + l + "]"
			}
			tabs = append(tabs, l)
		}
		s := strings.Join(tabs, " ")
		if i == v.Shown {
			s = "*" + s
		}
		groups = append(groups, s)
	}
	return strings.Join(groups, " | ")
}

// setDrawer puts s as ref's drawer and lays the drawers out again.
func setDrawer(a *App, ref string, s termState) *ui.TermView {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.putTermState(ref, s)
	a.remountTerminalsLocked()
	return a.termView()
}

// Splitting goes after the active terminal in its group, up to four; the
// fifth is refused. A new terminal gets a group of its own, which shows.
func TestTermSplitLimit(t *testing.T) {
	a, ref := termTestApp(t)
	s := termState{}.add("term-1", false, false)
	for _, id := range []string{"term-2", "term-3", "term-4", "term-5"} {
		s = s.add(id, true, true)
	}
	if got, want := shown(setDrawer(a, ref, s)), "*Terminal 1 Terminal 2 Terminal 3 [Terminal 4]"; got != want {
		t.Fatalf("after five splits: %s, want %s", got, want)
	}
	s = s.activate("term-2").add("term-6", true, false)
	if got, want := shown(setDrawer(a, ref, s)), "*Terminal 1 [Terminal 2] Terminal 3 Terminal 4"; got != want {
		t.Fatalf("a split of a full group: %s, want %s", got, want)
	}
	s = s.add("term-6", false, false)
	if got, want := shown(setDrawer(a, ref, s)), "Terminal 1 Terminal 2 Terminal 3 Terminal 4 | *[Terminal 6]"; got != want {
		t.Fatalf("a new terminal: %s, want %s", got, want)
	}
	if !s.shownGroups()[0].Vertical {
		t.Error("a vertical split stacks its group")
	}
}

// Closing the active terminal hands over to the next in its group, else
// the one before it, else the one now in its place; closing the last
// resets the drawer, dragged height and all.
func TestTermCloseOrder(t *testing.T) {
	a, ref := termTestApp(t)
	s := termState{}.add("term-1", false, false).add("term-2", true, false).add("term-3", true, false).add("term-4", false, false)
	s = s.activate("term-2").setHeight(400)
	for _, c := range []struct{ close, want string }{
		{"term-2", "*Terminal 1 [Terminal 3] | Terminal 4"},
		{"term-3", "*[Terminal 1] | Terminal 4"},
		{"term-1", "*[Terminal 4]"},
	} {
		s = s.activate(c.close).close(c.close)
		if got := shown(setDrawer(a, ref, s)); got != c.want {
			t.Fatalf("closing %s: %s, want %s", c.close, got, c.want)
		}
	}
	if s.Height != 400 {
		t.Fatalf("height %v, want 400 while terminals remain", s.Height)
	}
	s = s.close("term-4")
	if got := shown(setDrawer(a, ref, s)); got != "closed" || s.Height != termDefaultHeight {
		t.Fatalf("after the last: %s at %v, want closed at 280", got, s.Height)
	}
	if _, kept := a.termStates[ref]; kept {
		t.Error("a drawer back at its defaults should be forgotten")
	}
}

// The server's list wins where it has moved on, but not over terminals
// the drawer closed, and not while it's a strict subset of the drawer's,
// as while a new shell starts. Its labels name the terminals.
func TestTermReconcile(t *testing.T) {
	a, ref := termTestApp(t)
	e := a.envs[0]
	e.termsKnown = true
	summary := func(id, label string) t3.TermSummary {
		return t3.TermSummary{ThreadID: "t1", TerminalID: id, Status: "running", Label: label}
	}
	e.terms = []t3.TermSummary{summary("term-1", "")}
	s := termState{}.add("term-1", false, false).add("term-2", false, false)
	if got := shown(setDrawer(a, ref, s)); got != "Terminal 1 | *[Terminal 2]" {
		t.Fatalf("a strict subset changed the drawer: %s", got)
	}
	a.termHidden[ref] = []string{"term-2"}
	e.terms = []t3.TermSummary{summary("term-10", "npm run dev"), summary("term-1", ""), summary("term-2", ""), summary("term-3", "")}
	if got, want := shown(setDrawer(a, ref, a.termState(ref))), "*[Terminal 1] | Terminal 3 | npm run dev"; got != want {
		t.Fatalf("after the server's list: %s, want %s", got, want)
	}
}

// Ids are the lowest free term-N; labels read "Terminal N" for T3's ids.
func TestTermIDsAndLabels(t *testing.T) {
	if got := nextTermID([]string{"term-1", "term-3", "other"}); got != "term-2" {
		t.Errorf("next id %s, want term-2", got)
	}
	for id, want := range map[string]string{
		"term-12": "Terminal 12", "terminal-3": "Terminal 3", "TERM-4": "Terminal 4",
		"term-2-0f8fad5b-d9cb-469f-a165-70867728950e": "Terminal 2", "setup": "setup",
	} {
		if got := termLabel(id); got != want {
			t.Errorf("label %s = %q, want %q", id, got, want)
		}
	}
	ids := []string{"term-10", "term-2", "term-1"}
	if naturalLess(ids[0], ids[1]) || !naturalLess(ids[2], ids[1]) {
		t.Error("term-2 sorts before term-10, after term-1")
	}
}

// The drawer reopens where it was: open, its height, its terminals.
func TestTermStatePersists(t *testing.T) {
	a, ref := termTestApp(t)
	setDrawer(a, ref, termState{}.add("term-1", false, false).add("term-2", true, true).setHeight(360))
	b, err := NewWithStore(a.store, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := b.termState(ref)
	if !s.Open || s.Height != 360 || strings.Join(s.IDs, ",") != "term-1,term-2" || !s.Groups[0].Vertical {
		t.Fatalf("reloaded %+v", s)
	}
}

// A terminal that exits says so in its own screen, then leaves the drawer.
func TestTermExit(t *testing.T) {
	a, ref := termTestApp(t)
	setDrawer(a, ref, termState{}.add("term-1", false, false).add("term-2", false, false))
	term := &terminal{ref: ref, id: "term-2", emu: newTermEmu(40, 4), input: make(chan func()), status: "closed", synced: "closed", sentGen: -1}
	a.mu.Lock()
	a.terms[termKey(ref, "term-2")] = term
	a.mu.Unlock()
	a.termEvent(term, t3.TermEvent{Type: "snapshot", Snapshot: &t3.TermSnapshot{Status: "running", History: "$ "}})
	a.termEvent(term, t3.TermEvent{Type: "exited"})
	screen := ""
	for y := 0; y < term.emu.Height(); y++ {
		for x := 0; x < term.emu.Width(); x++ {
			if c := term.emu.CellAt(x, y); c != nil {
				screen += c.Content
			}
		}
	}
	if !strings.Contains(screen, "[terminal] Process exited") {
		t.Fatalf("screen %q, want the exit line", screen)
	}
	for start := time.Now(); ; time.Sleep(10 * time.Millisecond) {
		a.mu.Lock()
		got := shown(a.termView())
		a.mu.Unlock()
		if got == "*[Terminal 1]" {
			break
		}
		if time.Since(start) > 5*time.Second {
			t.Fatalf("after the exit the drawer shows %s", got)
		}
	}
}
