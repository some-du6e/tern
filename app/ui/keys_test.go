package ui

import (
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The composer: Enter submits, Mod+Enter submits the other way (T3's
// alternate dispatch, a queue while a run works), Shift+Enter is a newline.
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
	r.Queue(key.Event{Name: key.NameReturn, Modifiers: key.ModShortcut, State: key.Press})
	acts := frame()
	if len(acts) == 0 || acts[0].Kind != "submit" || !acts[0].Alt {
		t.Fatalf("Mod+Enter: actions %+v, want an alternate submit", acts)
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

// With the @, / or $ menu open, its keys are the menu's: the arrows,
// Enter, Tab and Escape go to it, not to the prompt. With it shut, Up on
// an empty prompt goes back through the history, and Down comes forward
// only while the prompt is the one it brought back.
func TestComposerMenuKeys(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 200)), Source: r.Source()}
		th.Editor("composer").Submit = true
		th.editor(gtx, "composer", Txt{}, "")
		r.Frame(&ops)
		return th.Actions()
	}
	press := func(name key.Name) []Action {
		r.Queue(key.Event{Name: name, State: key.Press}, key.Event{Name: name, State: key.Release})
		return frame()
	}
	keyed := func(acts []Action, id, kind, text string) bool {
		for _, a := range acts {
			if a.ID == id && a.Kind == kind && a.Text == text {
				return true
			}
		}
		return false
	}
	th.Focus("composer")
	frame()
	th.SetEditorText("composer", "@comp")
	frame()
	th.in.composer.menu = true
	for _, k := range []key.Name{key.NameDownArrow, key.NameUpArrow, key.NameReturn, key.NameTab, key.NameEscape} {
		if acts := press(k); !keyed(acts, "suggest", "key", string(k)) {
			t.Fatalf("%s with the menu open: %+v", k, acts)
		}
	}
	if got := th.EditorText("composer"); got != "@comp" {
		t.Fatalf("the menu's keys reached the prompt: %q", got)
	}

	th.in.composer.menu, th.in.composer.history = false, true
	th.SetEditorText("composer", "")
	frame()
	if acts := press(key.NameUpArrow); !keyed(acts, "composer", "history", "back") {
		t.Fatalf("Up on an empty prompt: %+v", acts)
	}
	th.RecallPrompt("an earlier prompt")
	frame()
	if acts := press(key.NameDownArrow); !keyed(acts, "composer", "history", "forward") {
		t.Fatalf("Down on a recalled prompt: %+v", acts)
	}
	th.SetEditorText("composer", "typed")
	frame()
	if acts := press(key.NameUpArrow); keyed(acts, "composer", "history", "back") {
		t.Fatalf("Up on a typed prompt walked the history: %+v", acts)
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

// A reply's code block has a copy button, like T3's.
func TestCodeBlockCopy(t *testing.T) {
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	src := "Run this:\n\n```powershell\nGet-Process tern | Stop-Process\n```"
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 300)), Source: r.Source(), Now: time.Now()}
		th.Markdown(gtx, src)
		r.Frame(&ops)
		th.Actions()
	}
	frame()
	// The button is the last thing in the header, at the right.
	var at image.Point
	for id := range th.in.copies {
		if !strings.HasPrefix(id, "copy:code:") {
			t.Fatalf("copy id %q", id)
		}
	}
	for y := 0; y < 60 && at == (image.Point{}); y += 2 {
		for x := 599; x > 500; x -= 2 {
			p := f32.Pt(float32(x), float32(y))
			r.Queue(pointer.Event{Kind: pointer.Move, Position: p, Source: pointer.Mouse})
			frame()
			if c := th.in.clicks["copy:"+firstKey(th.in.copies)[len("copy:"):]]; c != nil && c.Hovered() {
				at = image.Pt(x, y)
				break
			}
		}
	}
	if at == (image.Point{}) {
		t.Fatal("no copy button in the header")
	}
	p := f32.Pt(float32(at.X), float32(at.Y))
	r.Queue(pointer.Event{Kind: pointer.Press, Position: p, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: p, Source: pointer.Mouse})
	frame()
	if mime, data, ok := r.WriteClipboard(); !ok || mime != "application/text" || string(data) != "Get-Process tern | Stop-Process" {
		t.Fatalf("clipboard = %q %q %v", mime, data, ok)
	}
}

func firstKey(m map[string]string) string {
	for k := range m {
		return k
	}
	return ""
}

// PowerShell blocks get a ▷ that runs them on this PC; other languages don't.
func TestCodeBlockRun(t *testing.T) {
	th := NewTheme()
	var ran [][2]string
	th.CanRun = func(lang string) bool { return lang == "powershell" }
	th.Run = func(lang, code string) error { ran = append(ran, [2]string{lang, code}); return nil }
	var r gioinput.Router
	var ops op.Ops
	src := "```powershell\nStart-Process notepad\n```\n\n```go\nfunc main() {}\n```"
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 300)), Source: r.Source(), Now: time.Now()}
		th.Markdown(gtx, src)
		r.Frame(&ops)
		th.Actions()
	}
	frame()
	if len(th.in.runs) != 1 {
		t.Fatalf("run buttons: %d, want 1 (PowerShell only)", len(th.in.runs))
	}
	var rid string
	for k := range th.in.runs {
		rid = k
	}
	// Find it by hovering along the header, then click it.
	for x := 599; x > 400; x-- {
		p := f32.Pt(float32(x), 12)
		r.Queue(pointer.Event{Kind: pointer.Move, Position: p, Source: pointer.Mouse})
		frame()
		if c := th.in.clicks[rid]; c != nil && c.Hovered() {
			r.Queue(pointer.Event{Kind: pointer.Press, Position: p, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
				pointer.Event{Kind: pointer.Release, Position: p, Source: pointer.Mouse})
			frame()
			break
		}
	}
	if len(ran) != 1 || ran[0] != [2]string{"powershell", "Start-Process notepad"} {
		t.Fatalf("ran %v", ran)
	}
	if _, _, copied := r.WriteClipboard(); copied {
		t.Fatal("the run button copied too")
	}
}
