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
	term       int
	termScroll int
	termWheel  float32
	termSel    termSel
	popAt      map[string]time.Time // when each open popover opened, for popIn
	tipSince   map[string]time.Time // when the pointer came to rest on each tooltip's target
	fades      map[string]*fade     // transitions in flight, by name
	// Keyboard focus: whether it shows (Tab brought it; a pointer press
	// hides it, as :focus-visible does), the targets that tell a double
	// click from two, and the last press, in window coordinates.
	keyFocus bool
	// The details card takes the focus into its first control when it
	// opens; quietly (no ring) when a pointer opened it.
	focusCard, quietFocus bool
	doubles               map[string]bool
	pressAt               image.Point
	pressed               bool                   // a press landed this frame
	anchors               map[string]image.Point // where targets drop their menus, in the window
	hovers                map[string]*hoverTag
	// Whether the last press landed on a button (a click target other
	// than a sidebar row, T3's are divs), which the rename field's blur
	// asks; the click targets that are buttons; and a button's press seen
	// this frame.
	pressButton, buttonPressed bool
	buttons                    map[string]bool
	// The details card's scroll; whether it's being laid out only to
	// measure it, which leaves its popups and focus alone; and the density
	// it settled on for the card it was last given.
	cardScroll  int
	cardScrollG gesture.Scroll
	measuring   bool
	density     densityMemo
	// The card's top in the window and the top of the row it's laying out,
	// which put a pull request's tooltip above or below its row; and the
	// tooltips that have asked for what they show while they're up.
	cardTop, rowTop int
	tipAsked        map[string]bool
	// The branch picker's search and highlight last frame: a new search
	// scrolls its list to the top, a new highlight into view.
	pickQuery string
	pickHi    int
	// The menus: the row being drawn under the pointer, the open context
	// menu's submenus and highlight, and the open card menu's focus and lit
	// row; and the click target that takes the focus on its next layout, a
	// menu's trigger once Escape closes the menu.
	menuHover bool
	menu      menuState
	cardMenu  cardMenuState
	refocus   string
	// The shell: the window's width, which tooltips keep inside; whether
	// the details card fits inline (from the last layout), and the sidebar
	// rail's drag.
	winW        int
	cardFits    bool
	railDrag    bool
	railW       float32
	railEdge    handle
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
	if th.in.buttons == nil {
		th.in.buttons = map[string]bool{}
	}
	th.in.buttons[id] = !strings.HasPrefix(id, "thread:")
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
		if e.Kind == gesture.KindPress && th.in.buttons[id] {
			th.in.buttonPressed = true
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
			if th.in.doubles[id] && e.NumClicks > 1 {
				th.push(Action{ID: id, Kind: "double"})
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
				th.in.menu.open = false // a new menu starts afresh
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
	th.in.pressed = false
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &th.in.ptr, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			if e.Buttons == pointer.ButtonSecondary {
				th.in.ctxAt = e.Position.Round()
			}
			th.in.pressAt, th.in.pressed, th.in.keyFocus = e.Position.Round(), true, false
			th.in.pressButton, th.in.buttonPressed = th.in.buttonPressed, false
		}
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &th.in.ptr)
}

// keyTarget makes click target id a stop for Tab: Enter and Space click it
// while it has the focus. It reports whether the focus ring shows, which
// is only while focus came from the keyboard.
func (th *Theme) keyTarget(gtx C, id string) bool {
	c, _ := th.events(gtx, id)
	if th.in.refocus == id {
		th.in.refocus = ""
		gtx.Execute(key.FocusCmd{Tag: c})
	}
	for {
		ev, ok := gtx.Event(key.FocusFilter{Target: c},
			key.Filter{Focus: c, Name: key.NameReturn}, key.Filter{Focus: c, Name: key.NameEnter}, key.Filter{Focus: c, Name: key.NameSpace})
		if !ok {
			break
		}
		switch ev := ev.(type) {
		case key.FocusEvent:
			if ev.Focus {
				th.in.keyFocus, th.in.quietFocus = !th.in.quietFocus, false
			}
		case key.Event:
			if ev.State == key.Press {
				th.push(Action{ID: id, Kind: "click"})
			}
		}
	}
	return gtx.Focused(c) && th.in.keyFocus
}

// focusTarget moves keyboard focus to click target id, as T3 does when a
// panel opens into its first control.
func (th *Theme) focusTarget(gtx C, id string) {
	if c := th.in.clicks[id]; c != nil {
		gtx.Execute(key.FocusCmd{Tag: c})
	}
}

// FocusTarget sends the focus back to click target id on its next layout,
// with its ring: Escape hands it back to the trigger of the menu it
// closes.
func (th *Theme) FocusTarget(id string) { th.in.refocus = id }

// FocusCard sends the focus into the details card's first control on its
// next frame, as T3 does when the card opens. Opened from the keyboard,
// the control shows its ring; opened with the pointer, it doesn't.
func (th *Theme) FocusCard() {
	th.in.focusCard, th.in.quietFocus = true, !th.in.keyFocus
}

// hoverTag is a watch on the pointer that takes none of its events.
type hoverTag struct{ in bool }

func (th *Theme) hoverTag(id string) *hoverTag {
	if th.in.hovers == nil {
		th.in.hovers = map[string]*hoverTag{}
	}
	h := th.in.hovers[id]
	if h == nil {
		h = new(hoverTag)
		th.in.hovers[id] = h
	}
	return h
}

// hoverState reports whether the pointer is over the area hoverArea
// watches as id.
func (th *Theme) hoverState(gtx C, id string) bool {
	h := th.hoverTag(id)
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: h, Kinds: pointer.Enter | pointer.Leave | pointer.Cancel})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			h.in = e.Kind == pointer.Enter
		}
	}
	return h.in
}

// hoverArea watches a size-sized area for the pointer as id, passing every
// event on to what's under it.
func (th *Theme) hoverArea(gtx C, id string, size image.Point) {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, th.hoverTag(id))
}

// contextArea takes right-clicks on a size-sized area for target id
// without taking anything else: presses pass on to what's under it, like
// the window's drag area.
func (th *Theme) contextArea(gtx C, id string, size image.Point) {
	_, rc := th.events(gtx, id)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, rc)
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
			fill(gtx, over(P.Accent, P.Background), image.Rectangle{Max: dims.Size}, dp(gtx, 4))
		}
		call.Add(gtx.Ops)
		th.hit(gtx, id, dims.Size)
		return dims
	}
}

// Tip is Click with T3's tooltip: above w when up is set, else below.
// align lines it up with w's left edge (-1), middle (0) or right edge (1),
// so one by the window's edge stays in.
func (th *Theme) Tip(id, tip string, up bool, align int, w W) W {
	click := th.Click(id, w)
	return func(gtx C) D {
		dims := click(gtx)
		th.tipAt(gtx, id, tip, dims.Size, up, align)
		return dims
	}
}

// tipAt shows T3's tooltip for click target id (size big at the current
// offset) once the pointer has rested on it for 600ms, fading and scaling
// in over 150ms (docs/ui/02-primitives.md §3.7, port note 10). align lines
// it up with the target's left edge (-1), middle (0) or right edge (1).
func (th *Theme) tipAt(gtx C, id, tip string, size image.Point, up bool, align int) {
	th.tipPlaced(gtx, id, tip, size, up, func(w int) int {
		switch align {
		case -1:
			return 0
		case 1:
			return size.X - w
		}
		return (size.X - w) / 2
	})
}

// tipInside is tipAt below a target whose left edge is x in a window win
// wide: centred on it, then moved to stay 5dp inside the window, as Base
// UI's collision padding keeps it (02-primitives.md port note 8).
func (th *Theme) tipInside(gtx C, id, tip string, size image.Point, x, win int) {
	th.tipPlaced(gtx, id, tip, size, false, func(w int) int {
		edge := dp(gtx, 5)
		return max(edge-x, min((size.X-w)/2, win-edge-x-w))
	})
}

// tipArea shows tip above r (at the current offset) while the pointer
// rests on it, without taking the presses from what's under it: the
// tooltip of an icon inside a row.
func (th *Theme) tipArea(gtx C, id, tip string, r image.Rectangle) {
	defer op.Offset(r.Min).Push(gtx.Ops).Pop()
	th.hoverArea(gtx, id, r.Size())
	th.tipShown(gtx, id, tip, r.Size(), true, th.hoverState(gtx, id), false, func(w int) int { return (r.Dx() - w) / 2 })
}

// codeTip shows text in T3's code tooltip, up to 480 wide, centred above
// the size-sized area at the current offset while the pointer rests on it,
// without taking the presses from what's under it: StartTruncatedPath's,
// which names the whole path.
func (th *Theme) codeTip(gtx C, id, text string, size image.Point) {
	th.hoverArea(gtx, id, size)
	th.popupTip(gtx, id, th.hoverState(gtx, id), false, 480, func(gtx C) D { return th.tipBox(gtx, text, tipCode) },
		func(box image.Point) (image.Point, bool) { return image.Pt((size.X-box.X)/2, -box.Y-dp(gtx, 4)), true })
}

// tipPlaced is tipAt with the tooltip's left, given its width, from left.
func (th *Theme) tipPlaced(gtx C, id, tip string, size image.Point, up bool, left func(w int) int) {
	c := th.in.clicks[id]
	// Keyboard focus opens it at once, without the fade.
	focused := c != nil && gtx.Focused(c) && th.in.keyFocus
	th.tipShown(gtx, id, tip, size, up, c != nil && c.Hovered(), focused, left)
}

// tipShown draws target id's tooltip while it's hovered or focused.
func (th *Theme) tipShown(gtx C, id, tip string, size image.Point, up, hovered, focused bool, left func(w int) int) {
	if tip == "" {
		delete(th.in.tipSince, id)
		return
	}
	th.popupTip(gtx, id, hovered, focused, 320, func(gtx C) D { return th.tooltip(gtx, tip) }, func(box image.Point) (image.Point, bool) {
		gap := dp(gtx, 4)
		if up {
			return image.Pt(left(box.X), -box.Y-gap), true
		}
		return image.Pt(left(box.X), size.Y+gap), false
	})
}

// popupTip shows box, at most maxW wide, as target id's tooltip once the
// pointer has rested on it for 600ms, or at once while it has the
// keyboard's focus: fading in and growing from 98% over 150ms out of the
// side facing the target, where place puts it given its size (relative to
// the target, and whether that's above it). It reports whether it shows.
func (th *Theme) popupTip(gtx C, id string, hovered, focused bool, maxW float32, box W, place func(size image.Point) (image.Point, bool)) bool {
	if !(hovered || focused) {
		delete(th.in.tipSince, id)
		return false
	}
	if th.in.tipSince == nil {
		th.in.tipSince = map[string]time.Time{}
	}
	since, ok := th.in.tipSince[id]
	if !ok {
		since = gtx.Now
		th.in.tipSince[id] = since
	}
	shown := gtx.Now.Sub(since) - 600*time.Millisecond
	if focused {
		shown = 150 * time.Millisecond
	}
	if shown < 0 {
		gtx.Execute(op.InvalidateCmd{At: since.Add(600 * time.Millisecond)})
		return false
	}
	t := easeOut(clamp01(float32(shown) / float32(150*time.Millisecond)))
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, maxW), 1<<20)}
	td := box(g)
	call := m.Stop()
	at, up := place(td.Size)
	m2 := op.Record(gtx.Ops)
	op.Offset(at).Add(gtx.Ops)
	// Gio's renderer needs every opacity layer closed in the frame.
	fade := paint.PushOpacity(gtx.Ops, t)
	// It grows from 98% out of the side facing the trigger.
	origin := f32.Pt(float32(td.Size.X)/2, 0)
	if up {
		origin.Y = float32(td.Size.Y)
	}
	k := 0.98 + 0.02*t
	op.Affine(f32.Affine2D{}.Scale(origin, f32.Pt(k, k))).Add(gtx.Ops)
	call.Add(gtx.Ops)
	fade.Pop()
	op.Defer(gtx.Ops, m2.Stop())
	return true
}

// tooltip is T3's tooltip popup (TooltipPopup): popover surface, 1dp
// border, radius 6, 12/16.5 text padded 4 by 8 inside the border, and its
// lip and small shadow: 26.5dp tall on one line (§3.2).
func (th *Theme) tooltip(gtx C, tip string) D { return th.tipBox(gtx, tip, tipPlain) }

// tipStyle is a tooltip box's look.
type tipStyle int

const (
	tipPlain tipStyle = iota
	tipGlass          // a tooltip-styled popover (popover.tsx tooltipStyle)
	tipCode           // the code variant: 11px mono on relaxed lines
)

// tipBox is a tooltip's box in style: the plain one; a tooltip-styled
// popover's, the dropdown glass as it paints over the window inside a 1dp
// border at the foreground's 10%, with 12/16 text; or the code variant's,
// the plain box with 11px mono on 1.625 lines. Its lines are CSS line
// boxes, which Gio's own label doesn't make.
func (th *Theme) tipBox(gtx C, tip string, style tipStyle) D {
	t := Txt{Size: 12, LineH: 16.5 / 12, Color: P.PopoverForeground}
	bg, border := P.Popover, over(P.Border, P.Popover)
	switch style {
	case tipGlass:
		t.LineH = 16.0 / 12
		bg = cardFill()
		border = over(alpha(P.ContrastForeground, 0x1a), bg)
	case tipCode:
		t.Size, t.LineH, t.Font = 11, 1.625, font.Font{Typeface: mono}
	}
	m := op.Record(gtx.Ops)
	d := pad(5, 9, 5, 9).Layout(gtx, func(gtx C) D { return th.boxText(gtx, t, tip) })
	call := m.Stop()
	r := image.Rectangle{Max: d.Size}
	rad := dp(gtx, 6)
	shadow(gtx, r, rad, []boxShadow{{0, 4, 6, -1, alpha(rgb(0), 0x0d)}, {0, 2, 4, -2, alpha(rgb(0), 0x0d)}})
	fill(gtx, bg, r, rad)
	strokeRect(gtx, border, r, rad, float32(max(1, dp(gtx, 1))))
	if dark {
		hline(gtx, over(alpha(rgb(0xffffff), 0x0f), bg), max(1, dp(gtx, 1)), rad, r.Max.X-rad)
	} else {
		hline(gtx, over(alpha(rgb(0), 0x0a), bg), r.Max.Y-max(1, dp(gtx, 1))-1, rad, r.Max.X-rad)
	}
	call.Add(gtx.Ops)
	return d
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

// menuItem is Click for a menu row: T3 highlights the row under the pointer
// with accent, composited on the popover rather than the window.
func (th *Theme) menuItem(id string, w W) W {
	if id == "" {
		return w
	}
	return func(gtx C) D {
		c, _ := th.events(gtx, id)
		th.in.menuHover = c.Hovered()
		m := op.Record(gtx.Ops)
		dims := w(gtx)
		call := m.Stop()
		if th.in.menuHover {
			fill(gtx, over(P.Accent, P.Popover), image.Rectangle{Max: dims.Size}, dp(gtx, 4))
		}
		th.in.menuHover = false
		call.Add(gtx.Ops)
		th.hit(gtx, id, dims.Size)
		return dims
	}
}
