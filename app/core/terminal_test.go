package core

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"

	"github.com/some-du6e/tern/app/ui"
)

// Run in terminal opens the drawer on a shell that isn't running anything,
// the active one first (findReusableProjectScriptTerminal).
func TestRunInTerminalTakesAnIdleShell(t *testing.T) {
	for _, c := range []struct {
		name         string
		busy         []bool
		active, want int
	}{
		{"the active shell is idle", []bool{false, false}, 1, 1},
		{"the active shell is busy", []bool{false, true}, 1, 0},
	} {
		a := testApp(t)
		a.openThread("swiss/t1")
		d := &drawer{active: c.active}
		for i, busy := range c.busy {
			d.terms = append(d.terms, &terminal{id: string(rune('a' + i)), label: "Terminal", status: "running", busy: busy,
				emu: vt.NewSafeEmulator(termCols, termRows), input: make(chan func(), 1)})
		}
		a.drawers["swiss/t1"] = d
		a.Handle(nil, []ui.Action{{ID: "terminal.run", Kind: "click", Text: "ls -la"}})
		if s := a.build(time.Now()); s.Term == nil || s.Term.Active != c.want {
			t.Errorf("%s: drawer %+v, want it open on terminal %d", c.name, s.Term, c.want)
		}
	}
}
