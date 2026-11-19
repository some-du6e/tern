package ui

import (
	"bytes"
	"embed"
	"image"
	"image/png"
	"io"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The window shell, ported from T3 Code (docs/ui/03-shell.md): the sidebar
// on the left, resizable by its rail; the chat column with its header, the
// canvas and the terminal drawer; the right panel; and the controls fixed
// over the title bar's corners. All lengths are dp.

const (
	sidebarDefault = 256 // threadSidebarWidth.ts
	sidebarMinW    = 208
	mainMin        = 640 // a drag never leaves the main area narrower
	controlSize    = 28  // --workspace-titlebar-control-size
	cardW          = 280 // --thread-details-panel-width
	cardInset      = 12
	lanePad        = 48
	laneMax        = 736 // --chat-content-max-width, comfortable
	laneMin        = 640
	panelDefault   = 540 // the right panel's width
	panelMinW      = 360
)

// contentLeft is where content may start right of the fixed sidebar
// trigger: 52 in a browser, 130 on the Mac's desktop app.
func (th *Theme) contentLeft() float32 { return th.ControlsLeft + controlSize + 12 }

// Desktop draws the whole window.
func (th *Theme) Desktop(gtx C, s *Scene) D {
	if s.Live {
		defer th.trackPointer(gtx, gtx.Constraints.Max)
	}
	if th.in.blur {
		gtx.Execute(key.FocusCmd{})
		th.in.blur = false
	}
	if s.Live && th.in.clipOut != nil {
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(*th.in.clipOut))})
		th.in.clipOut = nil
	}
	if s.Live {
		th.shellKeys(gtx, s)
	}
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)

	sw := th.sidebarPx(gtx, s, size.X)
	rp := 0
	if s.Web != nil || s.Diff != nil {
		rp = th.panelPx(gtx, size.X, sw)
	}
	colW := size.X - sw - rp
	at := func(x int, w int, draw func(gtx C)) {
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(w, size.Y))
		draw(g)
	}
	at(sw, colW, func(gtx C) { th.chatColumn(gtx, s, sw == 0, rp > 0) })
	if rp > 0 {
		at(size.X-rp, rp, func(gtx C) { th.panelHost(gtx, s) })
	}
	// The sidebar sits over the main area, as in T3 (z 10).
	if sw > 0 {
		at(0, sw, func(gtx C) { th.sidebarFrame(gtx, s) })
	}
	// The fixed title bar controls go over everything else in the shell.
	th.sidebarTrigger(gtx, s, sw > 0)
	th.titlebarControls(gtx, s, size, rp > 0)

	if s.Context != nil {
		th.contextMenu(gtx, size, s)
	}
	if s.Live && (s.Picker != nil || s.Traits != nil || s.Access != nil || s.Account != nil || s.Machines != nil || s.Projects != nil || s.Context != nil || s.Details || s.Diff.menuOpen()) {
		// A click anywhere outside an open menu closes it. The menus are
		// deferred, so they sit above this.
		th.hit(gtx, "menus.close", size)
	}
	return D{Size: size}
}

// shellKeys handles T3's shell shortcuts (keybindings.ts): mod+B the
// sidebar, mod+J the terminal, mod+Alt+B the right panel, mod+Shift+B the
// thread details, mod+D the Diff surface (a focused terminal keeps it).
func (th *Theme) shellKeys(gtx C, s *Scene) {
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: "B", Required: key.ModShortcut, Optional: key.ModShift | key.ModAlt},
			key.Filter{Name: "J", Required: key.ModShortcut},
			key.Filter{Name: "D", Required: key.ModShortcut},
		)
		if !ok {
			return
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		switch {
		case ke.Name == "J" && s.Run != nil:
			th.push(Action{ID: "terminal.toggle", Kind: "click"})
		case ke.Name == "D" && s.Run != nil:
			th.push(Action{ID: "diff.toggle", Kind: "click"})
		case ke.Name == "B" && ke.Modifiers.Contain(key.ModShift) && s.Run != nil:
			th.push(th.detailsAction())
		case ke.Name == "B" && ke.Modifiers.Contain(key.ModAlt) && s.Run != nil:
			th.push(Action{ID: "browser.toggle", Kind: "click"})
		case ke.Name == "B" && !ke.Modifiers.Contain(key.ModShift|key.ModAlt):
			th.push(Action{ID: "sidebar.toggle", Kind: "click"})
		}
	}
}

// detailsAction toggles the details card where it fits, else its popover.
func (th *Theme) detailsAction() Action { return Action{ID: th.detailsID(), Kind: "click"} }

func (th *Theme) detailsID() string {
	if th.in.cardFits {
		return "details:card"
	}
	return "details"
}

// sidebarPx is the sidebar's width in px, 0 when it's collapsed: the
// dragged width or T3's 256, between 208 and the window less 640.
func (th *Theme) sidebarPx(gtx C, s *Scene, win int) int {
	if s.SidebarHidden {
		return 0
	}
	w := float32(sidebarDefault)
	if s.SidebarW > 0 {
		w = s.SidebarW
	}
	if th.in.railDrag {
		w = th.in.railW
	}
	wd := float32(win) / gtx.Metric.PxPerDp
	return dp(gtx, max(sidebarMinW, min(w, wd-mainMin)))
}

// panelPx is the inline right panel's width in px: 540 until dragged,
// between 360 and the smaller of 70% of the window and the main area less
// 360.
func (th *Theme) panelPx(gtx C, win, sidebar int) int {
	w := float32(panelDefault)
	if th.in.webW > 0 {
		w = th.in.webW
	}
	wd, layout := float32(win)/gtx.Metric.PxPerDp, float32(win-sidebar)/gtx.Metric.PxPerDp
	most := max(panelMinW, min(float32(int(0.7*wd)), layout-panelMinW))
	return dp(gtx, max(panelMinW, min(w, most)))
}

// ---------- the sidebar's frame ----------

//go:embed art/*.png
var artFiles embed.FS

// stageArt is T3's nightly sky behind the sidebar's header, as Chrome
// renders T3's own SVG at 800x80, already faded into each theme's sidebar
// colour (docs/ui/03-shell.md §3.4), at 1x and 2x.
var stageArt = map[string]paint.ImageOp{}

func stageArtImage(theme string, scale int) paint.ImageOp {
	k := theme + "@" + strconv.Itoa(scale)
	if im, ok := stageArt[k]; ok {
		return im
	}
	data, err := artFiles.ReadFile("art/stage-night-" + theme + "@" + strconv.Itoa(scale) + "x.png")
	if err != nil {
		panic(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	im := paint.NewImageOp(img)
	stageArt[k] = im
	return im
}

// sidebarFrame is the sidebar: its surface, the stage art and brand in the
// header row, the list below, the border on its right and the rail over it.
func (th *Theme) sidebarFrame(gtx C, s *Scene) {
	size := gtx.Constraints.Max
	border := max(1, dp(gtx, 1))
	inner := size.X - border
	// The rail reaches 8dp into the main area, so it goes outside the clip.
	defer th.rail(gtx, s, inner)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	fill(gtx, SP.Sidebar, image.Rect(0, 0, inner, size.Y), 0)
	// The border paints over the window behind the sidebar's edge.
	vline(gtx, over(SP.ContrastBorder, P.Background), inner, 0, size.Y)

	// The art spans 80dp, past the 52dp header and under the search row.
	func() {
		scale, theme := 1, "light"
		if gtx.Metric.PxPerDp >= 1.5 {
			scale = 2
		}
		if dark {
			theme = "dark"
		}
		defer clip.Rect{Max: image.Pt(inner, dp(gtx, 80))}.Push(gtx.Ops).Pop()
		k := float32(dp(gtx, 800)) / float32(800*scale)
		defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(k, k))).Push(gtx.Ops).Pop()
		stageArtImage(theme, scale).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}()
	th.dragArea(gtx, image.Pt(inner, dp(gtx, topBarH)))
	th.brand(gtx)

	func() {
		defer op.Offset(image.Pt(0, dp(gtx, topBarH))).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(inner, size.Y-dp(gtx, topBarH)))
		th.sidebar(g, s)
	}()
}

// brand is the wordmark at the start of the header row, white on the art:
// "tern" where T3 has its "T3 Code", in the same 14/20 medium type.
func (th *Theme) brand(gtx C) {
	defer op.Offset(image.Pt(dp(gtx, th.contentLeft()), dp(gtx, 12))).Push(gtx.Ops).Pop()
	FixedH(gtx, controlSize, func(gtx C) D {
		return layout.W.Layout(gtx, func(gtx C) D {
			return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: rgb(0xffffff), MaxLines: 1}, "tern")
		})
	})
}

// rail is the sidebar's resize handle: 16dp over its border, a 2dp line
// on hover, dragged to resize (released, the width persists) and
// double-clicked back to the default.
func (th *Theme) rail(gtx C, s *Scene, border int) {
	h := gtx.Constraints.Max.Y
	half := dp(gtx, 8)
	defer op.Offset(image.Pt(border-half, 0)).Push(gtx.Ops).Pop()
	d := th.dragEdge(gtx, &th.in.railEdge, image.Pt(2*half, h), true)
	switch {
	case th.in.railEdge.held && !th.in.railDrag:
		th.in.railDrag, th.in.railW = true, float32(border+max(1, dp(gtx, 1)))/gtx.Metric.PxPerDp
	case !th.in.railEdge.held && th.in.railDrag:
		th.in.railDrag = false
		th.push(Action{ID: "sidebar.width", Kind: "resize", Text: strconv.FormatFloat(float64(th.in.railW), 'f', 0, 32)})
	}
	if th.in.railDrag {
		th.in.railW += d / gtx.Metric.PxPerDp
		gtx.Execute(op.InvalidateCmd{})
	}
	if th.in.railEdge.hovered || th.in.railDrag {
		fill(gtx, over(SP.SidebarBorder, P.Background), image.Rect(half, 0, half+dp(gtx, 2), h), 0)
	}
	if th.in.railEdge.double {
		th.in.railEdge.double = false
		th.push(Action{ID: "sidebar.width", Kind: "resize", Text: ""})
	}
}

// ---------- the title bar's fixed controls ----------

// titleButton is a 28dp title bar button: T3's ghost toggle, or the
// media-navigation variant over the stage art.
type titleButton struct {
	icon     string
	onArt    bool // white, over the stage art
	pressed  bool
	disabled bool
	muted    bool // the sidebar trigger's grey icon (ghost variant)
	// tipAlign keeps the tooltip in the window by the edges: -1 lines it up
	// with the button's left, 1 its right, as T3's collision shift does.
	tipAlign int
}

func (th *Theme) titleButtonW(id, tip string, b titleButton) W {
	return func(gtx C) D {
		hovered := false
		if !b.disabled {
			c, _ := th.events(gtx, id)
			hovered = c.Hovered()
		}
		s := dp(gtx, controlSize)
		r := image.Rectangle{Max: image.Pt(s, s)}
		icon := alpha(P.Foreground, 0xcc) // the icon at 80%
		switch {
		case b.onArt && hovered:
			fill(gtx, alpha(rgb(0xffffff), 0x1a), r, dp(gtx, 8))
			icon = rgb(0xffffff)
		case b.onArt:
			icon = alpha(rgb(0xffffff), 0xe6)
		case b.disabled:
			icon = alpha(P.MutedForeground, 0xcc)
		case hovered || b.pressed:
			fill(gtx, over(P.Accent, P.Background), r, dp(gtx, 8))
			if b.pressed {
				icon = alpha(P.AccentForeground, 0xcc)
			}
		}
		if b.muted && !b.onArt {
			icon = P.ContrastMutedForeground
		}
		off := op.Offset(image.Pt((s-dp(gtx, 16))/2, (s-dp(gtx, 16))/2)).Push(gtx.Ops)
		LucideW(b.icon, icon, 16)(gtx)
		off.Pop()
		if !b.disabled {
			th.hit(gtx, id, r.Max)
			if tip != "" {
				th.tipAt(gtx, id, tip, r.Max, false, b.tipAlign)
			}
		}
		return D{Size: r.Max}
	}
}

// sidebarTrigger is the window-level sidebar toggle, just right of the
// Mac's traffic lights (or 13dp in), centred in the 52dp top bar.
func (th *Theme) sidebarTrigger(gtx C, s *Scene, visible bool) {
	defer op.Offset(image.Pt(dp(gtx, th.ControlsLeft+1), dp(gtx, 12))).Push(gtx.Ops).Pop()
	icon := "panel-left"
	if visible {
		icon = "panel-left-close"
	}
	th.titleButtonW("sidebar.toggle", "Toggle main sidebar ("+Shortcut("B")+")",
		titleButton{icon: icon, onArt: visible, muted: true, tipAlign: -1})(gtx)
}

// titlebarControls are the panel toggles fixed at the top right: thread
// details, terminal drawer, right panel. While the right panel is open the
// details toggle moves into the chat header and Maximize joins them.
func (th *Theme) titlebarControls(gtx C, s *Scene, size image.Point, panelOpen bool) {
	none := s.Run == nil
	type ctl struct {
		id, tip string
		b       titleButton
	}
	var ctls []ctl
	if !panelOpen {
		ctls = append(ctls, ctl{th.detailsID(), "Toggle thread details (" + ShortcutShift("B") + ")",
			titleButton{icon: "square-menu", pressed: th.detailsShown(s), disabled: none, tipAlign: 1}})
	}
	term := "Toggle terminal drawer (" + Shortcut("J") + ")"
	if none {
		term = "Terminal drawer is unavailable"
	}
	ctls = append(ctls,
		ctl{"terminal.toggle", term, titleButton{icon: "panel-bottom", pressed: s.Term != nil, disabled: none, tipAlign: 1}},
		ctl{"browser.toggle", "Toggle right panel (" + ShortcutAlt("B") + ")", titleButton{icon: "panel-right", pressed: s.Web != nil || s.Diff != nil, disabled: none, tipAlign: 1}},
	)
	n := len(ctls)
	gap := dp(gtx, 4)
	x := size.X - dp(gtx, 13) - n*dp(gtx, controlSize) - (n-1)*gap
	for _, c := range ctls {
		off := op.Offset(image.Pt(x, dp(gtx, 12))).Push(gtx.Ops)
		th.titleButtonW(c.id, c.tip, c.b)(gtx)
		off.Pop()
		x += dp(gtx, controlSize) + gap
	}
}

// detailsShown reports whether the details card or its popover is up.
func (th *Theme) detailsShown(s *Scene) bool {
	if th.in.cardFits {
		return s.DetailsCard
	}
	return s.Details
}

// ---------- the chat column ----------

// chatColumn is the header, the canvas below it and the terminal drawer at
// the bottom.
func (th *Theme) chatColumn(gtx C, s *Scene, collapsed, panelOpen bool) {
	size := gtx.Constraints.Max
	top := dp(gtx, topBarH)
	drawer := 0
	if s.Term != nil {
		drawer = th.drawerPx(gtx, size.Y)
	}
	// The canvas first, so the header draws over the timeline's top.
	func() {
		defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X, size.Y-top-drawer))
		th.canvas(g, s)
	}()
	func() {
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X, top))
		fill(g, P.Background, image.Rectangle{Max: g.Constraints.Max}, 0)
		th.dragArea(g, g.Constraints.Max)
		left := float32(20)
		if collapsed {
			left = th.contentLeft()
		}
		// The header's own 20dp end padding plus its content's reserve for
		// the fixed controls (pr-24), or pr-10 while the panel is open.
		right := float32(20 + 96)
		if panelOpen {
			right = 20 + 40
		}
		if s.Title != "" || !s.Live {
			pad(0, right, 0, left).Layout(g, func(gtx C) D { return th.threadHeader(gtx, s) })
		}
		// Without room for the card, the details are a popover off the
		// header's bottom right, 4dp in.
		th.deferBelow(g, image.Pt(size.X-dp(gtx, 4+cardW), top), func(gtx C) D {
			origin := f32.Pt(float32(dp(gtx, cardW)), 0)
			return th.popIn(gtx, "details", !th.in.cardFits && s.Details && s.Run != nil, &origin, func(gtx C) D { return th.details(gtx, s) })
		})
		if panelOpen {
			// The details toggle moves into the header while the panel is open.
			defer op.Offset(image.Pt(size.X-dp(gtx, 12+controlSize), dp(gtx, 12))).Push(gtx.Ops).Pop()
			th.titleButtonW(th.detailsID(), "Toggle thread details ("+ShortcutShift("B")+")",
				titleButton{icon: "square-menu", pressed: th.detailsShown(s), disabled: s.Run == nil, tipAlign: 1})(gtx)
		}
	}()
	if drawer > 0 {
		defer op.Offset(image.Pt(0, size.Y-drawer)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X, drawer))
		th.terminalDrawer(g, s)
	}
}

// drawerPx is the terminal drawer's height in px: 280 until dragged,
// between 180 and three quarters of the window.
func (th *Theme) drawerPx(gtx C, colH int) int {
	if th.in.termH == 0 {
		th.in.termH = termH
	}
	most := max(180, float32(int(0.75*float32(colH)/gtx.Metric.PxPerDp)))
	th.in.termH = max(180, min(th.in.termH, most))
	return dp(gtx, th.in.termH)
}

// lane is where the timeline's rows and the composer go in a canvas
// width wide (chatCanvasLayout.ts): at most 736 wide, centred, but kept
// 32 clear of the details card when it's inline (cardLeft >= 0).
func lane(width, padding, maxChat, gap, cardLeft int) (left, w int) {
	centred := max(0, min(width-2*padding, maxChat))
	right := width - padding
	if cardLeft >= 0 {
		right = cardLeft - gap
	}
	w = max(0, min(centred, right-padding))
	left = max(padding, min((width-w)/2, right-w))
	return left, w
}

// canvas is the conversation: the timeline in its lane, the composer over
// its bottom (or, for a new thread, the hero in the middle), and the
// details card at the top right when there's room for it.
func (th *Theme) canvas(gtx C, s *Scene) {
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	cardX := size.X - dp(gtx, cardW+cardInset)
	th.in.cardFits = s.Run != nil && cardX-dp(gtx, 32+lanePad) >= dp(gtx, laneMin) && size.Y-dp(gtx, 2*cardInset) >= dp(gtx, 160)
	cardLeft := -1
	if th.in.cardFits && s.DetailsCard {
		cardLeft = cardX
	}
	left, w := lane(size.X, dp(gtx, lanePad), dp(gtx, laneMax), dp(gtx, 32), cardLeft)
	in := func(y, h int, draw func(gtx C) D) D {
		defer op.Offset(image.Pt(left, y)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, h)}
		return draw(g)
	}
	switch {
	case s.Live && s.Hero != "" && len(s.Items) == 0:
		in(0, size.Y, func(gtx C) D {
			gtx.Constraints.Min.Y = size.Y
			return th.hero(gtx, s, false)
		})
	default:
		// The composer goes at the bottom with a 20dp spacer under it, and
		// the timeline fills what's above.
		m := op.Record(gtx.Ops)
		cd := in(0, size.Y, func(gtx C) D { return th.composerArea(gtx, s, false) })
		composer := m.Stop()
		ch := cd.Size.Y + dp(gtx, 20)
		in(0, size.Y-ch, func(gtx C) D {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			return th.messages(gtx, s, false)
		})
		th.topFade(gtx, image.Pt(size.X, size.Y))
		func() {
			defer op.Offset(image.Pt(0, size.Y-ch)).Push(gtx.Ops).Pop()
			composer.Add(gtx.Ops)
		}()
	}
	if cardLeft >= 0 {
		defer op.Offset(image.Pt(cardX, dp(gtx, cardInset))).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, cardW), size.Y-dp(gtx, 2*cardInset))}
		th.details(g, s)
	}
}

// topFade fades the timeline in under the header over 25dp, as T3's mask
// does, by painting the background over it at 1 - alpha in steps.
func (th *Theme) topFade(gtx C, size image.Point) {
	ramp := []struct{ y, a float32 }{{0, 0}, {2.5, 0.10}, {6, 0.30}, {10.5, 0.58}, {15.5, 0.82}, {20.5, 0.96}, {25, 1}}
	w := size.X - dp(gtx, 6) // the scrollbar lane isn't masked
	for i := 0; i+1 < len(ramp); i++ {
		y0, y1 := dp(gtx, ramp[i].y), dp(gtx, ramp[i+1].y)
		a := 1 - (ramp[i].a+ramp[i+1].a)/2
		fill(gtx, alpha(P.Background, uint8(a*255+0.5)), image.Rect(0, y0, w, y1), 0)
	}
}

// ---------- the right panel ----------

// panelHost is the inline right panel: its 1dp left border, the drag
// handle just inside it, and the surface.
func (th *Theme) panelHost(gtx C, s *Scene) {
	size := gtx.Constraints.Max
	border := max(1, dp(gtx, 1))
	vline(gtx, over(P.Border, P.Background), 0, 0, size.Y)
	func() {
		defer op.Offset(image.Pt(border, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X-border, size.Y))
		th.browserPanel(g, s)
	}()
	// The 4dp handle: a 1dp line on hover, primary at 60% while dragged.
	defer op.Offset(image.Pt(border, 0)).Push(gtx.Ops).Pop()
	d := th.dragEdge(gtx, &th.in.webEdge, image.Pt(dp(gtx, 4), size.Y), true)
	if d != 0 {
		th.in.webW = float32(size.X)/gtx.Metric.PxPerDp - d/gtx.Metric.PxPerDp
		gtx.Execute(op.InvalidateCmd{})
	}
	switch {
	case th.in.webEdge.held:
		fill(gtx, over(alpha(P.Primary, 0x99), P.Background), image.Rect(-border, 0, 0, size.Y), 0)
	case th.in.webEdge.hovered:
		fill(gtx, over(P.Border, P.Background), image.Rect(-border, 0, 0, size.Y), 0)
	}
}
