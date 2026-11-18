package ui

import (
	"image"
	"testing"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The composer: Enter submits, Alt+Enter submits as a queue, Shift+Enter is
// a newline.
func TestComposerKeys(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 200)), Source: r.Source()}
		ed := th.Editor("composer")
		ed.Submit = true
		th.editor(gtx, "composer", Txt{}, "")
		r.Frame(&ops)
		return th.Actions()
	}
	th.Focus("composer")
	frame()
	frame()
	r.Queue(key.EditEvent{Text: "hello"})
	frame()
	th.Editor("composer").SetCaret(5, 5) // where real typing leaves it
	r.Queue(key.Event{Name: key.NameReturn, Modifiers: key.ModShift, State: key.Press})
	frame()
	if got := th.EditorText("composer"); got != "hello\n" {
		t.Fatalf("Shift+Enter: text %q, want a newline", got)
	}
	r.Queue(key.Event{Name: key.NameReturn, Modifiers: key.ModAlt, State: key.Press})
	acts := frame()
	if len(acts) == 0 || acts[0].Kind != "submit" || !acts[0].Alt {
		t.Fatalf("Alt+Enter: actions %+v, want an Alt submit", acts)
	}
	r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	acts = frame()
	found := false
	for _, a := range acts {
		if a.Kind == "submit" && !a.Alt {
			found = true
		}
	}
	if !found {
		t.Fatalf("Enter: actions %+v, want a submit", acts)
	}
}

// Caret keys and word deletion reach the editor: Left, Up and Down move the
// caret, and Ctrl+Backspace deletes a word.
func TestEditingKeys(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 200)), Source: r.Source()}
		th.editor(gtx, "composer", Txt{}, "")
		r.Frame(&ops)
		return th.Actions()
	}
	press := func(name key.Name, mods key.Modifiers) {
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
		frame()
	}
	th.Focus("composer")
	frame()
	frame()
	th.SetEditorText("composer", "one two\nthree four")
	frame()
	ed := th.Editor("composer")
	press(key.NameLeftArrow, 0)
	if s, _ := ed.Selection(); s != 17 {
		t.Fatalf("Left: caret at %d, want 17", s)
	}
	press(key.NameUpArrow, 0)
	if s, _ := ed.Selection(); s >= 8 {
		t.Fatalf("Up: caret at %d, want it on the first line", s)
	}
	press(key.NameDownArrow, 0)
	if s, _ := ed.Selection(); s < 8 {
		t.Fatalf("Down: caret at %d, want it on the second line", s)
	}
	th.SetEditorText("composer", "one two")
	frame()
	press(key.NameDeleteBackward, key.ModShortcutAlt)
	if got := th.EditorText("composer"); got != "one " {
		t.Fatalf("Ctrl+Backspace: %q, want %q", got, "one ")
	}
	press(key.NameDeleteBackward, key.ModShortcutAlt)
	if got := th.EditorText("composer"); got != "" {
		t.Fatalf("Ctrl+Backspace again: %q, want it empty", got)
	}
	th.SetEditorText("composer", "one two")
	ed.SetCaret(0, 0)
	frame()
	press(key.NameDeleteForward, key.ModShortcutAlt)
	if got := th.EditorText("composer"); got != " two" {
		t.Fatalf("Ctrl+Delete: %q, want %q", got, " two")
	}
	press(key.NameRightArrow, key.ModShortcutAlt)
	if s, _ := ed.Selection(); s != 4 {
		t.Fatalf("Ctrl+Right: caret at %d, want 4", s)
	}
}

// A right-click on a target reports a "context" action, and the window-wide
// watch remembers where it landed for the menu.
func TestRightClick(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 600)), Source: r.Source()}
		func() {
			defer op.Offset(image.Pt(100, 200)).Push(gtx.Ops).Pop()
			th.hit(gtx, "thread:a/b", image.Pt(200, 40))
		}()
		th.trackPointer(gtx, image.Pt(800, 600))
		r.Frame(&ops)
		return th.Actions()
	}
	frame()
	at := f32.Pt(150, 215)
	r.Queue(
		pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonSecondary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	acts := frame()
	if len(acts) != 1 || acts[0].Kind != "context" || acts[0].ID != "thread:a/b" {
		t.Fatalf("actions = %+v, want one context on the row", acts)
	}
	if th.in.ctxAt != image.Pt(150, 215) {
		t.Fatalf("menu at %v, want the click", th.in.ctxAt)
	}
	// A left click is still a click.
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	if acts = frame(); len(acts) != 1 || acts[0].Kind != "click" {
		t.Fatalf("left click: %+v", acts)
	}
}

// A middle-click reports "middle", so tabs can close on it.
func TestMiddleClick(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(400, 100)), Source: r.Source()}
		th.hit(gtx, "panel:details", image.Pt(100, 30))
		r.Frame(&ops)
		return th.Actions()
	}
	frame()
	at := f32.Pt(20, 10)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonTertiary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	if acts := frame(); len(acts) != 1 || acts[0].Kind != "middle" || acts[0].ID != "panel:details" {
		t.Fatalf("actions = %+v", acts)
	}
}

// Ctrl+V in the composer goes to the app first (it may be a picture), and
// PasteText then does the editor's own text paste.
func TestComposerPaste(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 200)), Source: r.Source()}
		th.editor(gtx, "composer", Txt{}, "")
		r.Frame(&ops)
		return th.Actions()
	}
	th.Focus("composer")
	frame()
	frame()
	r.Queue(key.Event{Name: "V", Modifiers: key.ModShortcut, State: key.Press})
	acts := frame()
	if len(acts) != 1 || acts[0].Kind != "paste" {
		t.Fatalf("Ctrl+V: actions %+v, want one paste", acts)
	}
	if r.ClipboardRequested() {
		t.Fatal("the editor read the clipboard itself")
	}
	th.PasteText("composer")
	frame()
	if !r.ClipboardRequested() {
		t.Fatal("PasteText didn't ask for the clipboard")
	}
}

// A chip is one thing in the composer: Backspace and Delete take all of it,
// undo puts all of it back, the arrows step over it, and a chip broken some
// other way is cleared.
func TestComposerChips(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 200)), Source: r.Source()}
		th.editor(gtx, "composer", Txt{}, "")
		r.Frame(&ops)
		th.Actions()
	}
	press := func(name key.Name, mods key.Modifiers) {
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press}, key.Event{Name: name, Modifiers: mods, State: key.Release})
		frame()
	}
	ed := th.Editor("composer")
	th.Focus("composer")
	frame()
	frame()
	base := "a " + ChipText(1) + " b"
	th.SetEditorText("composer", base)
	ed.SetCaret(2+chipLen, 2+chipLen) // just after the chip
	frame()

	press(key.NameDeleteBackward, 0)
	if got := ed.Text(); got != "a  b" {
		t.Fatalf("Backspace: %q", got)
	}
	press("Z", key.ModShortcut)
	if got := ed.Text(); got != base {
		t.Fatalf("undo: %q, want the chip back whole", got)
	}

	ed.SetCaret(2+chipLen, 2+chipLen)
	frame()
	press(key.NameLeftArrow, 0)
	if s, _ := ed.Selection(); s != 2 {
		t.Fatalf("Left: caret at %d, want 2, before the chip", s)
	}
	press(key.NameRightArrow, 0)
	if s, _ := ed.Selection(); s != 2+chipLen {
		t.Fatalf("Right: caret at %d, want %d, after the chip", s, 2+chipLen)
	}
	ed.SetCaret(2, 2)
	frame()
	press(key.NameDeleteForward, 0)
	if got := ed.Text(); got != "a  b" {
		t.Fatalf("Delete: %q", got)
	}

	// Half a chip, the way a cut across one could leave it.
	th.SetEditorText("composer", "x "+string(ChipRune(2))+"   y"+string(chipFill))
	frame()
	if got := ed.Text(); got != "x    y" {
		t.Fatalf("broken chip: %q", got)
	}
	if got := StripChips("hi " + ChipText(3) + "!"); got != "hi !" {
		t.Fatalf("StripChips: %q", got)
	}
}
