package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

// The terminal drawer, ported from T3 Code (docs/ui/09-terminal.md): the
// frame at the bottom of the chat column and its resize strip, the shown
// group's split panes, and either the toolbar floating over a lone
// terminal or the list of terminals at the right. termsurface.go draws a
// terminal itself. Targets: "terminal.split", "terminal.splitv",
// "terminal.new", "terminal.close" (the active one) and "terminal.close:<id>",
// "terminal.activate:<id>", "terminal.confirm", "terminal.cancel",
// "terminal.link:<url>", and the "terminal.height" resize.

// TermView is the open thread's drawer.
type TermView struct {
	Thread  string  // the thread's ref, which keeps each pane's own state apart
	Height  float32 // the stored height, dp
	Groups  []TermGroup
	Shown   int    // the group whose panes show
	Active  string // the active terminal's id
	Panes   []TermPane
	Confirm *TermConfirm // asking before a terminal closes
	// Settle sends the panes' sizes to the server now instead of 150ms
	// after they last changed, as T3 does at the end of a drag.
	Settle func()
}

// TermGroup is a set of split panes: stacked when Vertical, else side by
// side.
type TermGroup struct {
	Vertical bool
	Tabs     []TermTab
}

// TermTab is one terminal in the list.
type TermTab struct{ ID, Label string }

// TermPane is a shown terminal: what it draws from, and where its keys,
// pastes, mouse reports and fitted size go.
type TermPane struct {
	ID, Label string
	Screen    TermScreen // nil while it attaches
	Ver       uint64     // counts the output it has taken
	Mouse     TermMouse  // the app's mouse tracking
	AppCursor bool       // application cursor keys (DECCKM)
	Paused    bool       // inside a synchronized update (DEC 2026)
	Key       func(uv.KeyPressEvent)
	Send      func(string) // bytes for the shell as they are
	Paste     func(string)
	Report    func(uv.MouseEvent)
	Fit       func(cols, rows int)
}

// TermMouse is the mouse tracking an app turned on: presses only (DEC 9,
// 1000), presses and drags (1002), or all motion (1003).
type TermMouse int

const (
	TermMouseOff TermMouse = iota
	TermMousePress
	TermMouseDrag
	TermMouseMotion
)

// TermConfirm is T3's close confirmation: its question and what closing
// does.
type TermConfirm struct{ Title, Description string }

// TermScreen is what the drawer reads off an emulator.
type TermScreen interface {
	Width() int
	Height() int
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
	CursorHidden() bool
	CursorStyle() (vt.CursorStyle, bool)
	IsAltScreen() bool
	ScrollbackLen() int
	ScrollbackCellAt(x, y int) *uv.Cell
}

const (
	termMinH   = 180  // MIN_DRAWER_HEIGHT
	termMaxK   = 0.75 // MAX_DRAWER_HEIGHT_RATIO
	termListW  = 144  // the list's w-36
	termGap    = 6    // between the panes and the list
	termTipGap = 6    // the toolbar labels' sideOffset
	splitMax   = 4    // MAX_TERMINALS_PER_GROUP
)

// termMaxHeight is the drawer's tallest (ThreadTerminalDrawer.tsx:121–133):
// three quarters of the window, but never more than the column has under
// its header and above the composer, which overlays the timeline.
func termMaxHeight(winH, colH, headerH, composerH float32) float32 {
	return min(max(termMinH, float32(math.Floor(float64(winH*termMaxK)))), max(0, colH-headerH-composerH))
}

// termHeight is the drawer's height for a stored one: rounded, no shorter
// than 180 unless most is, no taller than most.
func termHeight(stored, most float32) float32 {
	if math.IsNaN(float64(stored)) || math.IsInf(float64(stored), 0) {
		stored = 280
	}
	return min(max(float32(math.Round(float64(stored))), termMinH), most)
}

// termUI is the drawer's own state: the strip's drag, the toolbar labels'
// hover, the list rows' hover, an open menu, and each pane's scroll,
// selection and the rest.
type termUI struct {
	strip  termStrip
	tips   map[string]time.Time      // when the pointer came onto each toolbar button
	hovers map[string]*gesture.Hover // the list's rows
	menu   *termMenu
	panes  map[string]*termPane // by thread and terminal id
	focus  bool                 // focus the active terminal at its next layout
	most   float32              // the drawer's tallest this frame, dp
	shown  float32              // its height this frame, dp
	// The composer's height in the canvas with its spacer, px, as the
	// last frame measured it: the drawer may not push it into the header.
	composer int
	open     time.Time // when the close confirmation opened
	dialog   int       // its keyboard focus
	cells    termCells // the cell metrics, for one text scale
}

// termStrip is the 6dp resize strip along the drawer's top. The drag runs
// in the column's coordinates, which hold still while the drawer's top
// moves: T3's height is the start height plus how far the pointer went up.
type termStrip struct {
	tag    int
	held   bool
	startY float32 // the press, px down the column
	startH float32 // the drawer's height then, dp
	h      float32 // the height so far, dp
	at     int     // the drawer's top in the column this frame, px
	top    int     // and when the strip last took events
}

// FocusTerminal puts the keyboard on the active terminal once it's laid
// out, as T3's focus requests wait for the viewport to mount.
func (th *Theme) FocusTerminal() { th.in.terms.focus = true }

// drawerTop is where the shell lays the drawer in the chat column, px.
func (th *Theme) drawerTop(top int) { th.in.terms.strip.at = top }

// terminalDrawer is the drawer, in the size the shell gives it: the
// background, the 1dp top border at 80% and what's in it, then the strip.
func (th *Theme) terminalDrawer(gtx C, s *Scene) {
	v := s.Term
	if v.Confirm == nil {
		th.in.terms.open = time.Time{}
	}
	size := gtx.Constraints.Max
	px := max(1, dp(gtx, 1))
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	hline(gtx, over(faded(P.ContrastBorder, 0.8), P.Background), 0, 0, size.X)
	body := image.Rect(0, px, size.X, size.Y)
	// The labels and menus stay inside the window; with the right panel
	// closed the drawer's right edge is the window's.
	right := -1
	if s.Web == nil {
		right = size.X
	}
	if len(v.Groups) == 0 {
		th.termEmpty(gtx, body)
	} else {
		th.termBody(gtx, v, body, right)
	}
	th.termStripLayout(gtx, v, image.Pt(size.X, dp(gtx, 6)), px)
	th.termMenuLayout(gtx, v, size, right)
}

// termStripLayout is the strip: 6dp inside the top border, no paint, the
// row-resize cursor. Dragging it writes only the drawer's live height; the
// height is stored on release, if it moved from where it started and from
// the stored one.
func (th *Theme) termStripLayout(gtx C, v *TermView, size image.Point, top int) {
	st := &th.in.terms.strip
	k := gtx.Metric.PxPerDp
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &st.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		y := e.Position.Y + float32(st.top+top)
		switch e.Kind {
		case pointer.Press:
			if e.Buttons != pointer.ButtonPrimary {
				continue
			}
			st.held, st.startY, st.startH = true, y, th.in.terms.shown
			st.h = st.startH
		case pointer.Drag:
			if st.held {
				st.h = termHeight(st.startH+(st.startY-y)/k, th.in.terms.most)
				gtx.Execute(op.InvalidateCmd{})
			}
		case pointer.Release, pointer.Cancel:
			if !st.held {
				continue
			}
			st.held = false
			if st.h != st.startH && st.h != v.Height {
				th.push(Action{ID: "terminal.height", Kind: "resize", Text: strconv.FormatFloat(float64(st.h), 'f', -1, 32)})
			}
			if v.Settle != nil {
				v.Settle()
			}
			gtx.Execute(op.InvalidateCmd{})
		}
	}
	// The next events come relative to where the strip is now.
	st.top = st.at
	defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	pointer.CursorRowResize.Add(gtx.Ops)
	event.Op(gtx.Ops, &st.tag)
}

// termEmpty is the drawer with no terminals: the note and New Terminal,
// centred in 24 by 16 of padding.
func (th *Theme) termEmpty(gtx C, r image.Rectangle) {
	defer op.Offset(r.Min).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(r.Size())
	pad(24, 16, 24, 16).Layout(g, func(gtx C) D {
		return layout.Center.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground, Align: text.Middle}, "No terminal sessions for this thread yet.")
				}),
				layout.Rigid(Spacer(0, 12)),
				layout.Rigid(th.T3Button("terminal.new", Button{Variant: BtnOutline, Size: BtnSizeXs, Label: "New Terminal (" + Shortcut("N") + ")"})),
			)
		})
	})
}

// shownGroup is the group whose panes show, if there is one.
func (v *TermView) shownGroup() (TermGroup, bool) {
	if v.Shown < 0 || v.Shown >= len(v.Groups) {
		return TermGroup{}, false
	}
	return v.Groups[v.Shown], true
}

// termCount counts the drawer's terminals.
func (v *TermView) termCount() int {
	n := 0
	for _, g := range v.Groups {
		n += len(g.Tabs)
	}
	return n
}

// termBody is the panes on the terminal background, then the toolbar over
// a lone terminal's top right corner or, with more, the list 6dp to the
// right of the panes.
func (th *Theme) termBody(gtx C, v *TermView, r image.Rectangle, right int) {
	fill(gtx, P.TerminalBackground, r, 0)
	panes := r
	list := v.termCount() > 1
	if list {
		panes.Max.X -= dp(gtx, termListW+termGap)
	}
	th.termPanes(gtx, v, panes)
	full := false
	if g, ok := v.shownGroup(); ok {
		full = len(g.Tabs) >= splitMax
	}
	if list {
		lr := image.Rect(r.Max.X-dp(gtx, termListW), r.Min.Y, r.Max.X, r.Max.Y)
		th.termList(gtx, v, lr, full, right)
		return
	}
	w := 2*max(1, dp(gtx, 1)) + 4*dp(gtx, 21) + 3*max(1, dp(gtx, 1))
	at := image.Pt(r.Max.X-dp(gtx, 8)-w, r.Min.Y+dp(gtx, 8))
	th.termToolbar(gtx, at, full, right)
}

// termPanes lays the shown group's terminals out in equal parts, the last
// taking what's left over, each after the first behind a 1dp divider in
// --border, at 70% unless it's the active one. A press in a pane makes it
// the active terminal.
func (th *Theme) termPanes(gtx C, v *TermView, r image.Rectangle) {
	g, _ := v.shownGroup()
	n := len(v.Panes)
	if n == 0 {
		return
	}
	px := max(1, dp(gtx, 1))
	span := r.Dx()
	if g.Vertical {
		span = r.Dy()
	}
	part := span / n
	for i := range v.Panes {
		p := &v.Panes[i]
		a, b := i*part, (i+1)*part
		if i == n-1 {
			b = span
		}
		pr := image.Rect(r.Min.X+a, r.Min.Y, r.Min.X+b, r.Max.Y)
		if g.Vertical {
			pr = image.Rect(r.Min.X, r.Min.Y+a, r.Max.X, r.Min.Y+b)
		}
		active := p.ID == v.Active
		if i > 0 {
			line := faded(P.ContrastBorder, 0.7)
			if active {
				line = P.ContrastBorder
			}
			line = over(line, P.TerminalBackground)
			if g.Vertical {
				fill(gtx, line, image.Rect(pr.Min.X, pr.Min.Y, pr.Max.X, pr.Min.Y+px), 0)
				pr.Min.Y += px
			} else {
				fill(gtx, line, image.Rect(pr.Min.X, pr.Min.Y, pr.Min.X+px, pr.Max.Y), 0)
				pr.Min.X += px
			}
		}
		func() {
			defer op.Offset(pr.Min).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Exact(pr.Size())
			th.termSurface(g, v, p, active, pr.Min)
		}()
	}
}

// termToolbar is the floating group over a lone terminal: split side by
// side, split stacked, new and close, 21dp each between 1x16 separators, in
// a 1dp border at 80% with radius 6 and shadow-xs.
func (th *Theme) termToolbar(gtx C, at image.Point, full bool, right int) {
	defer op.Offset(at).Push(gtx.Ops).Pop()
	px := max(1, dp(gtx, 1))
	b := dp(gtx, 21)
	r := image.Rect(0, 0, 2*px+4*b+3*px, 2*px+b)
	rad := dp(gtx, 6)
	border := over(faded(P.ContrastBorder, 0.8), P.Background)
	shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}})
	fill(gtx, P.Background, r, rad)
	strokeRect(gtx, border, r, rad, float32(px))
	defer clip.UniformRRect(r.Inset(px), max(0, rad-px)).Push(gtx.Ops).Pop()
	x := px
	for i, t := range termTools(full) {
		if i > 0 {
			fill(gtx, border, image.Rect(x, px+(b-dp(gtx, 16))/2, x+px, px+(b-dp(gtx, 16))/2+dp(gtx, 16)), 0)
			x += px
		}
		func() {
			defer op.Offset(image.Pt(x, px)).Push(gtx.Ops).Pop()
			th.termTool(gtx, t, image.Pt(b, b), P.Background, P.Accent, at.Add(image.Pt(x, px)), right)
		}()
		x += b
	}
}

// termToolDef is one of the four toolbar buttons.
type termToolDef struct {
	id, icon, label string
	off             bool // a split at the limit: dimmed, no hover fill, does nothing
}

// termTools is the toolbar's buttons and their labels, with the default
// keys (keybindings.ts): mod+D, mod+Shift+D, mod+N, mod+W.
func termTools(full bool) []termToolDef {
	split, vsplit := "Split Terminal Horizontally ("+Shortcut("D")+")", "Split Terminal Vertically ("+ShortcutShift("D")+")"
	if full {
		split = "Split Terminal Horizontally (max " + strconv.Itoa(splitMax) + " per group)"
		vsplit = "Split Terminal Vertically (max " + strconv.Itoa(splitMax) + " per group)"
	}
	return []termToolDef{
		{"terminal.split", "square-split-horizontal", split, full},
		{"terminal.splitv", "square-split-vertical", vsplit, full},
		{"terminal.new", "plus", "New Terminal (" + Shortcut("N") + ")", false},
		{"terminal.close", "trash-2", "Close Terminal (" + Shortcut("W") + ")", false},
	}
}

// termTool is one toolbar button size big: the 13dp icon in foreground at
// 90%, hover on a fill, and its label below after 300ms. at is its place
// in the drawer, for the label to stay inside the window's right edge.
func (th *Theme) termTool(gtx C, t termToolDef, size image.Point, bg, hoverFill color.NRGBA, at image.Point, right int) {
	c, _ := th.events(gtx, t.id)
	hovered := c.Hovered()
	if hovered && !t.off {
		fill(gtx, over(hoverFill, bg), image.Rectangle{Max: size}, 0)
		bg = over(hoverFill, bg)
	}
	func() {
		if t.off {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		}
		ic := dp(gtx, 13)
		defer op.Offset(image.Pt((size.X-ic)/2, (size.Y-ic)/2)).Push(gtx.Ops).Pop()
		LucideW(t.icon, over(faded(P.Foreground, 0.9), bg), 13)(gtx)
	}()
	cursor := pointer.CursorDefault
	if t.off {
		cursor = pointer.CursorNotAllowed
	}
	th.termHit(gtx, t.id, size, cursor)
	th.termToolLabel(gtx, t.id, t.label, hovered, size, at, right)
}

// termHit is hit with the cursor T3 gives the target.
func (th *Theme) termHit(gtx C, id string, size image.Point, cursor pointer.Cursor) {
	c, rc := th.events(gtx, id)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
	event.Op(gtx.Ops, rc)
	cursor.Add(gtx.Ops)
}

// termToolLabel is a toolbar button's label, T3's hover popover in its
// tooltip style: 300ms after the pointer comes onto the button, 6dp below
// it, centred, but kept 5dp inside the window's right edge (right, when
// it's known); it fades and grows in over 150ms.
func (th *Theme) termToolLabel(gtx C, id, label string, hovered bool, size, at image.Point, right int) {
	tu := &th.in.terms
	if !hovered {
		delete(tu.tips, id)
		return
	}
	if tu.tips == nil {
		tu.tips = map[string]time.Time{}
	}
	since0, ok := tu.tips[id]
	if !ok {
		since0 = gtx.Now
		tu.tips[id] = since0
	}
	shown := gtx.Now.Sub(since0) - 300*time.Millisecond
	if shown < 0 {
		gtx.Execute(op.InvalidateCmd{At: since0.Add(300 * time.Millisecond)})
		return
	}
	t := easeOut(clamp01(float32(shown) / float32(150*time.Millisecond)))
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 320), dp(gtx, 200))}
	d := th.termLabelPopup(g, label)
	call := m.Stop()
	x := (size.X - d.Size.X) / 2
	if right >= 0 {
		x = min(x, right-dp(gtx, 5)-at.X-d.Size.X)
	}
	m2 := op.Record(gtx.Ops)
	op.Offset(image.Pt(x, size.Y+dp(gtx, termTipGap))).Add(gtx.Ops)
	paint.PushOpacity(gtx.Ops, t)
	k := 0.98 + 0.02*t
	op.Affine(f32.Affine2D{}.Scale(f32.Pt(float32(d.Size.X)/2, 0), f32.Pt(k, k))).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
}

// termLabelPopup is a popover in T3's tooltip style: the dropdown glass's
// solid fallback, its 1dp border, radius 6, shadow-md/5 and lip, 12/16
// text padded 4 by 8.
func (th *Theme) termLabelPopup(gtx C, label string) D {
	m := op.Record(gtx.Ops)
	d := pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.PopoverForeground}, label)
	})
	call := m.Stop()
	r := image.Rectangle{Max: d.Size}
	rad := dp(gtx, 6)
	px := max(1, dp(gtx, 1))
	shadow(gtx, r, rad, []boxShadow{{0, 4, 6, -1, alpha(rgb(0), 0x0d)}, {0, 2, 4, -2, alpha(rgb(0), 0x0d)}})
	fill(gtx, P.Popover, r, rad)
	strokeRect(gtx, over(glassBorder(), P.Popover), r, rad, float32(px))
	if dark {
		hline(gtx, over(alpha(rgb(0xffffff), 0x0f), P.Popover), px, rad, r.Max.X-rad)
	} else {
		hline(gtx, over(alpha(rgb(0), 0x0a), P.Popover), r.Max.Y-2*px, rad, r.Max.X-rad)
	}
	call.Add(gtx.Ops)
	return d
}

// glassBorder is the dropdown glass's border: foreground at 10%.
func glassBorder() color.NRGBA {
	if dark {
		return alpha(rgb(0xf5f5f5), 0x1a)
	}
	return alpha(rgb(0x14141d), 0x1a)
}

// termList is the list beside the panes when the drawer holds two
// terminals or more: its toolbar row, then each group, under a header
// when there's more than one group or any is split.
func (th *Theme) termList(gtx C, v *TermView, r image.Rectangle, full bool, right int) {
	px := max(1, dp(gtx, 1))
	line := over(faded(P.ContrastBorder, 0.7), P.TerminalBackground)
	bg := over(faded(P.Muted, 0.1), P.TerminalBackground)
	fill(gtx, bg, r, 0)
	strokeRect(gtx, line, r, 0, float32(px))
	inner := r.Inset(px)
	defer op.Offset(inner.Min).Push(gtx.Ops).Pop()
	w := inner.Dx()
	// The toolbar row: 22dp with its bottom border, the buttons at its end.
	row := dp(gtx, 22)
	fill(gtx, line, image.Rect(0, row-px, w, row), 0)
	tools := termTools(full)
	x := w
	for i := len(tools) - 1; i >= 0; i-- {
		t := tools[i]
		bw := dp(gtx, 21)
		left := 0
		if i > 0 {
			left = px
		}
		x -= bw + left
		if left > 0 {
			fill(gtx, line, image.Rect(x, 0, x+px, row-px), 0)
		}
		func() {
			defer op.Offset(image.Pt(x+left, 0)).Push(gtx.Ops).Pop()
			th.termTool(gtx, t, image.Pt(bw, row-px), bg, faded(P.Accent, 0.7), r.Min.Add(image.Pt(px+x+left, px)), right)
		}()
	}
	// The groups, padded 4.
	headers := len(v.Groups) > 1
	for _, g := range v.Groups {
		headers = headers || len(g.Tabs) > 1
	}
	defer clip.Rect{Min: image.Pt(0, row), Max: image.Pt(w, inner.Dy())}.Push(gtx.Ops).Pop()
	y := row + dp(gtx, 4)
	cw := w - 2*dp(gtx, 4)
	for gi, g := range v.Groups {
		func() {
			defer op.Offset(image.Pt(dp(gtx, 4), y)).Push(gtx.Ops).Pop()
			gy := 0
			if headers {
				gy += th.termGroupHeader(gtx, v, gi, g, cw, bg)
			}
			for j, t := range g.Tabs {
				if j > 0 {
					gy += dp(gtx, 2)
				}
				func() {
					defer op.Offset(image.Pt(0, gy)).Push(gtx.Ops).Pop()
					th.termRow(gtx, v, t, cw, bg)
				}()
				gy += dp(gtx, 24)
			}
			y += gy + dp(gtx, 2)
		}()
	}
}

// termGroupHeader is a group's header: its kind and how many terminals it
// holds, 22dp, 11/16 type; the active group's on accent at 50%. A click
// shows the group, at its active terminal or its first.
func (th *Theme) termGroupHeader(gtx C, v *TermView, gi int, g TermGroup, w int, bg color.NRGBA) int {
	h := dp(gtx, 22)
	active := gi == v.Shown
	target := g.Tabs[0].ID
	for _, t := range g.Tabs {
		if t.ID == v.Active {
			target = t.ID
		}
	}
	id := "terminal.group:" + target
	c, _ := th.events(gtx, id)
	col := P.MutedForeground
	switch {
	case active:
		fill(gtx, over(faded(P.Accent, 0.5), bg), image.Rect(0, 0, w, h), dp(gtx, 4))
		col = P.Foreground
	case c.Hovered():
		fill(gtx, over(faded(P.Accent, 0.4), bg), image.Rect(0, 0, w, h), dp(gtx, 4))
		col = P.Foreground
	}
	label, icon := "Single", "square"
	if len(g.Tabs) > 1 {
		label, icon = "Side by side", "square-split-horizontal"
		if g.Vertical {
			label, icon = "Stacked", "square-split-vertical"
		}
	}
	n := strconv.Itoa(len(g.Tabs))
	func() {
		lg := gtx
		lg.Constraints = layout.Exact(image.Pt(w, h))
		pad(0, 6, 0, 6).Layout(lg, func(gtx C) D {
			return Row(gtx,
				rigid(LucideW(icon, col, 12)),
				rigid(Spacer(4, 0)),
				flexed(func(gtx C) D {
					return th.Text(gtx, Txt{Size: 11, LineH: 16.0 / 11, Color: col, MaxLines: 1}, label)
				}),
				rigid(Spacer(4, 0)),
				rigid(func(gtx C) D {
					return th.Text(gtx, Txt{Size: 10, LineH: 14.0 / 10, Color: faded(P.MutedForeground, 0.7), MaxLines: 1}, n)
				}),
			)
		})
	}()
	th.termHit(gtx, id, image.Pt(w, h), pointer.CursorPointer)
	return h
}

// termRow is one terminal in the list: 24dp, radius 6, its close button
// (the terminal icon, an x while the row is hovered) and its label; the
// active row on accent, the others muted until hovered.
func (th *Theme) termRow(gtx C, v *TermView, t TermTab, w int, bg color.NRGBA) {
	h := dp(gtx, 24)
	tu := &th.in.terms
	if tu.hovers == nil {
		tu.hovers = map[string]*gesture.Hover{}
	}
	hv := tu.hovers[t.ID]
	if hv == nil {
		hv = new(gesture.Hover)
		tu.hovers[t.ID] = hv
	}
	hovered := hv.Update(gtx.Source)
	active := t.ID == v.Active
	col := P.MutedForeground
	switch {
	case active:
		fill(gtx, over(P.Accent, bg), image.Rect(0, 0, w, h), dp(gtx, 6))
		bg, col = over(P.Accent, bg), P.Foreground
	case hovered:
		fill(gtx, over(faded(P.Accent, 0.6), bg), image.Rect(0, 0, w, h), dp(gtx, 6))
		bg, col = over(faded(P.Accent, 0.6), bg), P.Foreground
	}
	defer clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops).Pop()
	hv.Add(gtx.Ops)
	// The close button: 16dp at 6 in, its icon 12.
	closeID := "terminal.close:" + t.ID
	cs := dp(gtx, 16)
	func() {
		defer op.Offset(image.Pt(dp(gtx, 6), (h-cs)/2)).Push(gtx.Ops).Pop()
		c, _ := th.events(gtx, closeID)
		if c.Hovered() {
			fill(gtx, over(P.Muted, bg), image.Rect(0, 0, cs, cs), dp(gtx, 4))
		}
		icon := "square-terminal"
		if hovered {
			icon = "x"
		}
		func() {
			defer op.Offset(image.Pt((cs-dp(gtx, 12))/2, (cs-dp(gtx, 12))/2)).Push(gtx.Ops).Pop()
			LucideW(icon, col, 12)(gtx)
		}()
		th.termHit(gtx, closeID, image.Pt(cs, cs), pointer.CursorPointer)
		tip := "Close " + t.Label
		if active {
			tip += " (" + Shortcut("W") + ")"
		}
		th.tipAt(gtx, closeID, tip, image.Pt(cs, cs), true, 0)
	}()
	// The label fills the rest, up to 8 from the end.
	lx := dp(gtx, 6) + cs + dp(gtx, 2)
	lw := w - lx - dp(gtx, 8)
	func() {
		defer op.Offset(image.Pt(lx, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(max(0, lw), h))
		layout.W.Layout(g, func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}, t.Label)
		})
		th.termHit(gtx, "terminal.activate:"+t.ID, image.Pt(max(0, lw), h), pointer.CursorPointer)
	}()
}

// termMenu is a terminal's right-click menu, or the menu that comes up
// after a mouse selection: T3's DOM menu, opened at the pointer.
type termMenu struct {
	pane    string      // the pane it's for
	at      image.Point // where it opened, px in the drawer
	items   []termMenuItem
	rows    []gesture.Click
	hot     int // the row the keyboard is on, -1 for none
	tag     int // its keyboard focus
	focused bool
	r       image.Rectangle
	wait    time.Time // the selection menu waits out a double click
}

type termMenuItem struct {
	label string
	off   bool
	do    func(th *Theme, p *termPane, tp *TermPane)
}

// termMenuItems is the right-click menu (ThreadTerminalDrawer.tsx:408–434),
// without Add to chat, which needs the composer's terminal contexts: Copy
// wants a selection.
func termMenuItems(selection bool) []termMenuItem {
	return []termMenuItem{
		{label: "Copy", off: !selection, do: func(th *Theme, p *termPane, tp *TermPane) {
			th.WriteClipboard(termSelText(tp.Screen, p.g, p.sel))
		}},
		{label: "Paste", do: func(th *Theme, p *termPane, _ *TermPane) { p.paste = true }},
		{label: "Select all", do: func(th *Theme, p *termPane, tp *TermPane) { p.selectAll(tp.Screen) }},
		{label: "Jump to latest", do: func(th *Theme, p *termPane, _ *TermPane) { p.back = 0 }},
	}
}

// termMenuLayout draws the open menu over everything at its place, kept
// 4dp inside the window, and takes its keys: arrows wrap through the rows
// that are on, Home and End, Enter or Space to pick, Escape or Tab to
// close. A press outside closes it.
func (th *Theme) termMenuLayout(gtx C, v *TermView, size image.Point, right int) {
	mn := th.in.terms.menu
	if mn == nil {
		return
	}
	var tp *TermPane
	for i := range v.Panes {
		if th.paneKey(v, v.Panes[i].ID) == mn.pane {
			tp = &v.Panes[i]
		}
	}
	p := th.in.terms.panes[mn.pane]
	if tp == nil || p == nil {
		th.in.terms.menu = nil
		return
	}
	if gtx.Now.Before(mn.wait) {
		gtx.Execute(op.InvalidateCmd{At: mn.wait})
		return
	}
	if !mn.focused {
		mn.focused = true
		gtx.Execute(key.FocusCmd{Tag: &mn.tag})
	}
	pick := func(i int) {
		th.in.terms.menu = nil
		if it := mn.items[i]; !it.off {
			it.do(th, p, tp)
		}
		th.in.terms.focus = true
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &mn.tag},
			key.Filter{Focus: &mn.tag, Name: key.NameUpArrow},
			key.Filter{Focus: &mn.tag, Name: key.NameDownArrow},
			key.Filter{Focus: &mn.tag, Name: key.NameHome},
			key.Filter{Focus: &mn.tag, Name: key.NameEnd},
			key.Filter{Focus: &mn.tag, Name: key.NameReturn},
			key.Filter{Focus: &mn.tag, Name: key.NameEnter},
			key.Filter{Focus: &mn.tag, Name: key.NameSpace},
			key.Filter{Focus: &mn.tag, Name: key.NameEscape},
			key.Filter{Focus: &mn.tag, Name: key.NameTab, Optional: key.ModShift},
			pointer.Filter{Target: mn, Kinds: pointer.Press},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			if !e.Focus {
				th.in.terms.menu = nil
				return
			}
		case pointer.Event:
			if !e.Position.Round().In(mn.r) {
				th.in.terms.menu = nil
				return
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			step := func(from, dir int) {
				for i := 1; i <= len(mn.items); i++ {
					j := ((from+dir*i)%len(mn.items) + len(mn.items)) % len(mn.items)
					if !mn.items[j].off {
						mn.hot = j
						return
					}
				}
			}
			switch e.Name {
			case key.NameDownArrow:
				step(mn.hot, 1)
			case key.NameUpArrow:
				if mn.hot < 0 {
					mn.hot = 0
				}
				step(mn.hot, -1)
			case key.NameHome:
				step(-1, 1)
			case key.NameEnd:
				step(0, -1)
			case key.NameReturn, key.NameEnter, key.NameSpace:
				if mn.hot >= 0 {
					pick(mn.hot)
					return
				}
			case key.NameEscape, key.NameTab:
				th.in.terms.menu = nil
				th.in.terms.focus = true
				return
			}
		}
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 384), size.Y)}
	d := th.termMenuBox(g, mn, pick)
	call := m.Stop()
	at := mn.at
	if right >= 0 {
		at.X = min(at.X, right-dp(gtx, 4)-d.Size.X)
	}
	at.Y = min(at.Y, size.Y-dp(gtx, 4)-d.Size.Y)
	mn.r = image.Rectangle{Min: at, Max: at.Add(d.Size)}
	m2 := op.Record(gtx.Ops)
	// Presses anywhere go on through, but one outside the menu closes it.
	func() {
		defer clip.Rect{Min: image.Pt(-1<<20, -1<<20), Max: image.Pt(1<<20, 1<<20)}.Push(gtx.Ops).Pop()
		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, mn)
	}()
	op.Offset(at).Add(gtx.Ops)
	call.Add(gtx.Ops)
	func() {
		defer clip.Rect{Max: d.Size}.Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, &mn.tag)
	}()
	op.Defer(gtx.Ops, m2.Stop())
}

// termMenuBox is the menu: the glass's solid fallback with its 1dp border,
// radius 8 and no shadow, padded 4; rows 28dp, 14/20 text padded 4 by 8,
// accent on the hovered or keyboard row; rows that are off at 64%.
func (th *Theme) termMenuBox(gtx C, mn *termMenu, pick func(int)) D {
	px := max(1, dp(gtx, 1))
	inset := px + dp(gtx, 4)
	type row struct {
		call op.CallOp
		d    D
	}
	var rows []row
	w := dp(gtx, 128) - 2*inset
	for _, it := range mn.items {
		m := op.Record(gtx.Ops)
		g := gtx
		g.Constraints.Min = image.Point{}
		d := pad(4, 8, 4, 8).Layout(g, func(gtx C) D {
			return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.ContrastForeground, MaxLines: 1}, it.label)
		})
		rows = append(rows, row{m.Stop(), d})
		w = max(w, d.Size.X)
	}
	rh := dp(gtx, 28)
	box := image.Rect(0, 0, w+2*inset, len(rows)*rh+2*inset)
	rad := dp(gtx, 8)
	fill(gtx, P.Popover, box, rad)
	strokeRect(gtx, over(glassBorder(), P.Popover), box, rad, float32(px))
	for i, r := range rows {
		func() {
			defer op.Offset(image.Pt(inset, inset+i*rh)).Push(gtx.Ops).Pop()
			it := mn.items[i]
			c := &mn.rows[i]
			for {
				e, ok := c.Update(gtx.Source)
				if !ok {
					break
				}
				if e.Kind == gesture.KindClick && !it.off {
					pick(i)
				}
			}
			if !it.off && (c.Hovered() || mn.hot == i) {
				fill(gtx, over(P.Accent, P.Popover), image.Rect(0, 0, w, rh), dp(gtx, 4))
			}
			func() {
				if it.off {
					defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
					paint.ColorOp{Color: P.ContrastMutedForeground}.Add(gtx.Ops)
				}
				defer op.Offset(image.Pt(0, (rh-r.d.Size.Y)/2)).Push(gtx.Ops).Pop()
				r.call.Add(gtx.Ops)
			}()
			if !it.off {
				defer clip.Rect{Max: image.Pt(w, rh)}.Push(gtx.Ops).Pop()
				c.Add(gtx.Ops)
				pointer.CursorDefault.Add(gtx.Ops)
			}
		}()
	}
	return D{Size: box.Max}
}

// openTermMenu opens a menu for pane key at at (px in the drawer). It
// takes the keyboard, with no row lit: the terminal's input is a text
// field, so T3's menu focuses itself.
func (th *Theme) openTermMenu(pane string, at image.Point, items []termMenuItem, wait time.Time) {
	th.in.terms.menu = &termMenu{pane: pane, at: at, items: items, rows: make([]gesture.Click, len(items)), hot: -1, wait: wait}
}

// termConfirm is T3's AlertDialog asking before a terminal closes
// (ConfirmDialogHost.tsx, docs/ui/02-primitives.md §3.21–3.22): the
// backdrop over the window, which takes presses but doesn't close it, and
// the popup centred up to 512 wide with its title, description and the
// default footer, Cancel and the destructive Confirm. It scales in from
// 98% and fades in over 200ms.
func (th *Theme) termConfirm(gtx C, c *TermConfirm, win image.Point) {
	tu := &th.in.terms
	if tu.open.IsZero() {
		// The dialog takes the keyboard from the terminal while it asks.
		tu.open = gtx.Now
		gtx.Execute(key.FocusCmd{Tag: &tu.dialog})
	}
	t := since(gtx.Now, tu.open, 200*time.Millisecond)
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	t = 1 - (1-t)*(1-t) // ease
	for {
		ev, ok := gtx.Event(key.FocusFilter{Target: &tu.dialog}, key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			th.push(Action{ID: "terminal.cancel", Kind: "click"})
		}
	}
	m := op.Record(gtx.Ops)
	func() {
		defer paint.PushOpacity(gtx.Ops, t).Pop()
		backdrop := alpha(rgb(0xfcfcfc), 0x99)
		if dark {
			backdrop = alpha(rgb(0), 0xa3)
		}
		paint.FillShape(gtx.Ops, backdrop, clip.Rect{Max: win}.Op())
		th.termHit(gtx, "terminal.backdrop", win, pointer.CursorDefault)
	}()
	w := min(dp(gtx, 512), win.X-2*dp(gtx, 16))
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, win.Y-2*dp(gtx, 16))}
	pm := op.Record(gtx.Ops)
	d := th.termConfirmBody(g, c, w)
	body := pm.Stop()
	r := image.Rectangle{Max: d.Size}
	func() {
		defer op.Offset(image.Pt((win.X-w)/2, (win.Y-d.Size.Y)/2)).Push(gtx.Ops).Pop()
		defer paint.PushOpacity(gtx.Ops, t).Pop()
		k := 0.98 + 0.02*t
		defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(float32(w)/2, float32(d.Size.Y)/2), f32.Pt(k, k))).Push(gtx.Ops).Pop()
		rad := dp(gtx, 16)
		border := alpha(rgb(0x14141d), 0x1a)
		shadows := []boxShadow{{0, 24, 64, -24, alpha(rgb(0), 0xa6)}}
		if dark {
			border = alpha(rgb(0xffffff), 0x14)
			shadows = []boxShadow{{0, 24, 72, -20, alpha(rgb(0), 0xe6)}}
		}
		shadow(gtx, r, rad, shadows)
		fill(gtx, P.Popover, r, rad)
		strokeRect(gtx, over(border, P.Popover), r, rad, float32(max(1, dp(gtx, 1))))
		if dark {
			hline(gtx, over(alpha(rgb(0xffffff), 0x0a), P.Popover), max(1, dp(gtx, 1)), rad, r.Max.X-rad)
		}
		th.termHit(gtx, "terminal.dialog", r.Max, pointer.CursorDefault)
		func() {
			defer clip.Rect{Max: r.Max}.Push(gtx.Ops).Pop()
			event.Op(gtx.Ops, &tu.dialog)
		}()
		body.Add(gtx.Ops)
	}()
	op.Defer(gtx.Ops, m.Stop())
}

// termConfirmBody is the dialog's header, padded 24 with an 8dp gap, and
// its footer: a rule over muted at 72%, Cancel and Confirm at the end.
func (th *Theme) termConfirmBody(gtx C, c *TermConfirm, w int) D {
	px := max(1, dp(gtx, 1))
	return Col(gtx,
		rigid(func(gtx C) D {
			gtx.Constraints.Min.X = w
			return pad(24, 24, 24, 24).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Col(gtx,
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 20, LineH: 27.5 / 20, Font: UI(font.SemiBold), Color: P.PopoverForeground}, c.Title)
					}),
					rigid(Spacer(0, 8)),
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, c.Description)
					}),
				)
			})
		}),
		rigid(func(gtx C) D {
			foot := over(faded(P.Muted, 0.72), P.Popover)
			m := op.Record(gtx.Ops)
			gtx.Constraints.Min.X = w
			d := pad(16, 24, 16, 24).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Row(gtx,
					flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
					rigid(th.T3Button("terminal.cancel", Button{Variant: BtnOutline, Label: "Cancel", On: foot})),
					rigid(Spacer(8, 0)),
					rigid(th.T3Button("terminal.confirm", Button{Variant: BtnDestructive, Label: "Confirm", On: foot})),
				)
			})
			call := m.Stop()
			rad := dp(gtx, 16) - px
			paint.FillShape(gtx.Ops, foot, clip.RRect{Rect: image.Rect(px, 0, d.Size.X-px, d.Size.Y-px), SE: rad, SW: rad}.Op(gtx.Ops))
			hline(gtx, over(P.ContrastBorder, P.Popover), 0, px, d.Size.X-px)
			call.Add(gtx.Ops)
			return d
		}),
	)
}
