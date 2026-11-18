package ui

import (
	"gioui.org/f32"
	"image"
	"image/color"
	"io"
	"strings"
	"time"
	"unicode"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// input is the interactive state behind the immediate-mode screens. Every
// target is named by a string ID; a frame's clicks, submits and edits come
// back from Actions after layout. Scenes without IDs (the mockups) draw
// exactly as before and register nothing.
type input struct {
	// The terminal drawer's height and the browser panel's width, in dp,
	// once dragged; and their edges.
	termH, webW       float32
	termEdge, webEdge handle
	// The browser panel: its focus tag, the address shown, the last press
	// (for double clicks) and move.
	browser       int
	browserURL    string
	browserPress  time.Time
	browserAt     f32.Point
	browserClicks int
	browserBtn    string
	browserMoved  time.Time
	clipOut       *string // text to put on the clipboard at the next frame
	blur          bool    // drop the keyboard focus at the next frame
	// The terminal drawer: its focus tag, how many lines it has scrolled
	// back, and wheel movement short of a line.
	term        int
	termScroll  int
	termWheel   float32
	termSel     termSel
	popAt       map[string]time.Time // when each open popover opened, for popIn
	clicks      map[string]*gesture.Click
	context     map[string]*int // right-click targets, by the same ids
	ptr         int             // tag for the window-wide pointer watch
	ctxAt       image.Point     // where the last right-click landed, in window coordinates
	editors     map[string]*widget.Editor
	lists       map[string]*layout.List
	focus       string
	caret       [2]int                        // the composer's selection last frame, to step over chips
	selectables map[string]*widget.Selectable // selectable text: sent messages, code blocks
	copies      map[string]string             // copy buttons' text, by click id
	copied      map[string]time.Time          // when each was last clicked, for its check
	runs        map[string][2]string          // run buttons' language and code, by click id
	runFailed   map[string]bool               // a run that couldn't start, for its ✕
	paste       string                        // an editor to paste text into next frame
	actions     []Action
}

// Action is one thing the user did this frame: ID is the target, Kind is
// "click", "context" (a right-click), "middle" (a middle-click), "submit",
// "change", "key" or "paste" (Ctrl+V in the composer), Text the
// editor text or key name.
type Action struct {
	ID, Kind, Text string
	Alt            bool
}

// Actions returns and clears what happened since the last call.
func (th *Theme) Actions() []Action {
	a := th.in.actions
	th.in.actions = nil
	return a
}

func (th *Theme) push(a Action) { th.in.actions = append(th.in.actions, a) }

// Editor is the persistent editor behind a text field.
func (th *Theme) Editor(id string) *widget.Editor {
	if th.in.editors == nil {
		th.in.editors = map[string]*widget.Editor{}
	}
	e := th.in.editors[id]
	if e == nil {
		e = &widget.Editor{}
		th.in.editors[id] = e
	}
	return e
}

// Focus moves keyboard focus to the field with this ID on its next layout.
func (th *Theme) Focus(id string) { th.in.focus = id }

// List is a persistent scroll list. toEnd lists stay pinned to their last
// item, like a chat.
func (th *Theme) List(id string, toEnd bool) *layout.List {
	if th.in.lists == nil {
		th.in.lists = map[string]*layout.List{}
	}
	l := th.in.lists[id]
	if l == nil {
		l = &layout.List{Axis: layout.Vertical, ScrollToEnd: toEnd}
		th.in.lists[id] = l
	}
	return l
}

// hit makes a size-sized area at the current offset a click target and
// reports whether the pointer is over it.
func (th *Theme) hit(gtx C, id string, size image.Point) bool {
	c, rc := th.events(gtx, id)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
	event.Op(gtx.Ops, rc)
	pointer.CursorPointer.Add(gtx.Ops)
	return c.Hovered()
}

// events handles target id's clicks so far and returns its gesture and
// right-click tag. Click calls it before drawing, so the hover highlight
// is never a frame behind: a stale one would stay lit after the pointer
// left, since nothing asks for another frame.
func (th *Theme) events(gtx C, id string) (*gesture.Click, *int) {
	if th.in.clicks == nil {
		th.in.clicks = map[string]*gesture.Click{}
	}
	c := th.in.clicks[id]
	if c == nil {
		c = new(gesture.Click)
		th.in.clicks[id] = c
	}
	for {
		e, ok := c.Update(gtx.Source)
		if !ok {
			break
		}
		if e.Kind == gesture.KindClick {
			if r, ok := th.in.runs[id]; ok {
				err := th.Run(r[0], r[1])
				if th.in.copied == nil {
					th.in.copied = map[string]time.Time{}
				}
				if th.in.runFailed == nil {
					th.in.runFailed = map[string]bool{}
				}
				th.in.copied[id], th.in.runFailed[id] = gtx.Now, err != nil
				gtx.Execute(op.InvalidateCmd{})
				continue
			}
			if text, ok := th.in.copies[id]; ok {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
				th.in.copied[id] = gtx.Now
				gtx.Execute(op.InvalidateCmd{})
				continue
			}
			if f, ok := cutPrefix(id, "focus:"); ok {
				th.in.focus = f
				continue
			}
			th.push(Action{ID: id, Kind: "click"})
		}
	}
	if th.in.context == nil {
		th.in.context = map[string]*int{}
	}
	rc := th.in.context[id]
	if rc == nil {
		rc = new(int)
		th.in.context[id] = rc
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: rc, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			switch e.Buttons {
			case pointer.ButtonSecondary:
				th.push(Action{ID: id, Kind: "context"})
			case pointer.ButtonTertiary:
				th.push(Action{ID: id, Kind: "middle"})
			}
		}
	}
	return c, rc
}

// trackPointer watches the pointer over a size-sized window without taking
// its events, so a right-click menu opens where the click was. Call it at the
// window's origin after everything else: the topmost area sees events first,
// and passes them on.
func (th *Theme) trackPointer(gtx C, size image.Point) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &th.in.ptr, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Buttons == pointer.ButtonSecondary {
			th.in.ctxAt = e.Position.Round()
		}
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &th.in.ptr)
}

// Click makes w a click target named id, with a faint highlight on hover.
// An empty id draws w as is.
func (th *Theme) Click(id string, w W) W {
	if id == "" {
		return w
	}
	return func(gtx C) D {
		c, _ := th.events(gtx, id)
		m := op.Record(gtx.Ops)
		dims := w(gtx)
		call := m.Stop()
		if c.Hovered() {
			fill(gtx, P.Accent, image.Rectangle{Max: dims.Size}, dp(gtx, 4))
		}
		call.Add(gtx.Ops)
		th.hit(gtx, id, dims.Size)
		return dims
	}
}

// Tip is Click with T3's tooltip while the pointer is over w: above it
// when up is set, else below. align lines it up with w's left edge (-1),
// middle (0) or right edge (1), so one by the window's edge stays in.
func (th *Theme) Tip(id, tip string, up bool, align int, w W) W {
	click := th.Click(id, w)
	return func(gtx C) D {
		dims := click(gtx)
		if c := th.in.clicks[id]; c == nil || !c.Hovered() || tip == "" {
			return dims
		}
		m := op.Record(gtx.Ops)
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 320), dp(gtx, 60))}
		td := Box{Bg: ColPanelHi, Border: ColBorderHi, Radius: 6, Pad: padXY(8, 4)}.Layout(g, func(gtx C) D {
			return th.Text(gtx, Txt{Size: SizeSmall, Color: ColText, MaxLines: 1}, tip)
		})
		call := m.Stop()
		at := image.Pt((dims.Size.X-td.Size.X)/2, dims.Size.Y+dp(gtx, 6))
		switch align {
		case -1:
			at.X = 0
		case 1:
			at.X = dims.Size.X - td.Size.X
		}
		if up {
			at.Y = -td.Size.Y - dp(gtx, 6)
		}
		m2 := op.Record(gtx.Ops)
		op.Offset(at).Add(gtx.Ops)
		call.Add(gtx.Ops)
		op.Defer(gtx.Ops, m2.Stop())
		return dims
	}
}

// Field is a live single-line text field: frame, optional icon, the editor,
// and the placeholder while it's empty. Enter submits.
func (th *Theme) Field(id string, ic Icon, hasIcon bool, placeholder string, mask bool) W {
	return func(gtx C) D {
		ed := th.Editor(id)
		ed.SingleLine, ed.Submit = true, true
		if mask {
			ed.Mask = '•'
		}
		focused := gtx.Focused(ed)
		border := ColBorder
		if focused {
			border = alpha(ColAccent, 0xa0)
		}
		return Box{Bg: ColFrame, Border: border, Radius: 4, Pad: padXY(8, 0)}.Layout(gtx, func(gtx C) D {
			return FixedH(gtx, 26, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Row(gtx,
					rigid(func(gtx C) D {
						if !hasIcon {
							return D{}
						}
						return pad(0, 6, 0, 0).Layout(gtx, IconW(ic, ColFaint, 14))
					}),
					flexed(func(gtx C) D {
						return th.editor(gtx, id, Txt{Size: SizeUI}, placeholder)
					}),
				)
			})
		})
	}
}

// editor lays out the editor named id with style t, handling its events.
func (th *Theme) editor(gtx C, id string, t Txt, placeholder string) D {
	ed := th.Editor(id)
	if th.in.focus == id {
		gtx.Execute(key.FocusCmd{Tag: ed})
		th.in.focus = ""
	}
	// Alt+Enter has its own meaning in the composer (queue), so take it
	// before the editor turns it into a submit.
	for {
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameReturn, Required: key.ModAlt})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			th.push(Action{ID: id, Kind: "submit", Text: ed.Text(), Alt: true})
		}
	}
	// The composer takes pictures too. Its Ctrl+V goes to the app, which
	// looks for an image and calls PasteText when there's none.
	if id == "composer" {
		for {
			ev, ok := gtx.Event(key.Filter{Focus: ed, Name: "V", Required: key.ModShortcut})
			if !ok {
				break
			}
			if ke, ok := ev.(key.Event); ok && ke.State == key.Press && !ed.ReadOnly {
				th.push(Action{ID: id, Kind: "paste"})
			}
		}
	}
	if id == "composer" {
		th.chipKeys(gtx, id, ed)
	}
	if th.in.paste == id {
		th.in.paste = ""
		gtx.Execute(clipboard.ReadCmd{Tag: ed})
	}
	// Gio's editor treats every Ctrl chord as a shortcut on Windows and
	// Linux and returns before its word deletion, so Ctrl+Backspace and
	// Ctrl+Delete do nothing. Do them here.
	for {
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameDeleteBackward, Required: key.ModShortcutAlt, Optional: key.ModShift},
			key.Filter{Focus: ed, Name: key.NameDeleteForward, Required: key.ModShortcutAlt, Optional: key.ModShift})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press && !ed.ReadOnly {
			dir := -1
			if ke.Name == key.NameDeleteForward {
				dir = 1
			}
			if deleteWord(ed, dir) {
				th.push(Action{ID: id, Kind: "change", Text: ed.Text()})
			}
		}
	}
	for {
		// Only Escape: the arrows, Home, End and the rest belong to the editor.
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameEscape})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			th.push(Action{ID: id, Kind: "key", Text: string(ke.Name)})
		}
	}
	for {
		ev, ok := ed.Update(gtx)
		if !ok {
			break
		}
		switch ev := ev.(type) {
		case widget.SubmitEvent:
			th.push(Action{ID: id, Kind: "submit", Text: ev.Text})
		case widget.ChangeEvent:
			th.push(Action{ID: id, Kind: "change", Text: ed.Text()})
		}
	}
	if id == "composer" {
		if fixChips(ed) {
			th.push(Action{ID: id, Kind: "change", Text: ed.Text()})
		}
		th.snapChips(ed)
	}
	if t.Size == 0 {
		t.Size = SizeUI
	}
	if t.Font.Typeface == "" {
		t.Font = UI(font.Normal)
	}
	if t.Color.A == 0 {
		t.Color = ColText
	}
	col := func(c color.NRGBA) op.CallOp {
		m := op.Record(gtx.Ops)
		paint.ColorOp{Color: c}.Add(gtx.Ops)
		return m.Stop()
	}
	if ed.Len() == 0 && placeholder != "" {
		th.Text(gtx, Txt{Size: t.Size, Color: ColFaint, Font: t.Font, MaxLines: 1}, placeholder)
	}
	ed.LineHeightScale = t.LineH
	d := ed.Layout(gtx, th.Shaper, t.Font, unit.Sp(t.Size), col(t.Color), col(alpha(ColAccent, 0x60)))
	if id == "composer" {
		th.drawChips(gtx, ed, d.Size, editorScroll(ed, d.Size), ColFrame, float32(t.Size), ImageLabel)
	}
	return d
}

// InsertText types s at the caret of the editor named id.
func (th *Theme) InsertText(id, s string) {
	th.Editor(id).Insert(s)
	th.push(Action{ID: id, Kind: "change", Text: th.EditorText(id)})
}

// DeleteChip deletes picture n's chip from the editor named id, keeping
// the caret where it was in the rest of the text.
func (th *Theme) DeleteChip(id string, n int) {
	ed := th.Editor(id)
	for _, c := range chipSpans(ed.Text()) {
		if c.n != n {
			continue
		}
		s, e := ed.Selection()
		ed.SetCaret(c.start, c.end)
		ed.Insert("")
		shift := func(x int) int {
			switch {
			case x >= c.end:
				return x - (c.end - c.start)
			case x > c.start:
				return c.start
			}
			return x
		}
		ed.SetCaret(shift(s), shift(e))
		th.push(Action{ID: id, Kind: "change", Text: ed.Text()})
		return
	}
}

// PasteText pastes the clipboard's text into the editor named id on the
// next frame, the way the editor's own Ctrl+V would.
func (th *Theme) PasteText(id string) { th.in.paste = id }

// EditorText is the current text of the field named id.
func (th *Theme) EditorText(id string) string { return th.Editor(id).Text() }

// SetEditorText replaces a field's text and puts the caret at the end.
func (th *Theme) SetEditorText(id, s string) {
	ed := th.Editor(id)
	ed.SetText(s)
	n := len([]rune(s))
	ed.SetCaret(n, n)
}

// deleteWord deletes the selection, or else from the caret to the edge of
// the word before it (dir -1) or after it (dir 1), with the spaces between.
func deleteWord(ed *widget.Editor, dir int) bool {
	start, end := ed.Selection()
	if start == end {
		text := []rune(ed.Text())
		i := start
		at := func(i int) rune {
			if dir < 0 {
				return text[i-1]
			}
			return text[i]
		}
		more := func(i int) bool { return (dir < 0 && i > 0) || (dir > 0 && i < len(text)) }
		for more(i) && unicode.IsSpace(at(i)) {
			i += dir
		}
		for more(i) && !unicode.IsSpace(at(i)) {
			i += dir
		}
		if i == start {
			return false
		}
		ed.SetCaret(i, start)
	}
	return ed.Delete(1) != 0
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return s, false
}
