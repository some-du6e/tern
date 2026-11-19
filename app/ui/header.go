package ui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// T3's chat header (ChatHeader.tsx, WorkspaceBreadcrumb.tsx;
// docs/ui/05-header.md §2.2, §3.1, §4.1, §5.1–5.2): a breadcrumb of the
// project, a subagent's parent and the thread's title across the chat
// column's title row. Each crumb's box is its cap height plus 6dp above and
// below, centred in the row, so the row centres on the letters.

const (
	crumbTextMax = 160 // the project crumb's text, max-w-40
	crumbIcon    = 14  // the project's mark and the title's chevron, size-3.5
	titleMin     = 40  // the title crumb never shrinks past min-w-10
)

// crumbFont is the breadcrumb's type: 14/20, medium.
func crumbFont(col color.NRGBA) Txt {
	return Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: col}
}

// chatHeader draws the header's content in its content box
// (gtx.Constraints.Max: the row less its padding and the reserve for the
// fixed toggles): the breadcrumb, then a shared thread's guest link, which
// keeps its width. at is that box's top left in the window, for the thread
// menu and the rename editor. Narrow is the window under 640dp, where the
// gaps close to 8.
func (th *Theme) chatHeader(gtx C, s *Scene, at image.Point, narrow bool) D {
	h := s.Header
	size := gtx.Constraints.Max
	if h == nil {
		return D{Size: size}
	}
	// A right-click anywhere in the header opens the thread's menu there,
	// except while renaming.
	if h.Server && !h.Renaming {
		th.contextArea(gtx, "header", size)
	}
	mid := size.Y / 2
	nav, gap := size.X, dp(gtx, 12)
	if narrow {
		gap = dp(gtx, 8)
	}
	if h.Guest != nil {
		nav = max(0, size.X-th.guestLineW(gtx, h.Guest)-gap)
	}
	th.breadcrumb(gtx, s, nav, mid, at, narrow)
	if h.Guest != nil {
		defer op.Offset(image.Pt(nav+gap, 0)).Push(gtx.Ops).Pop()
		th.guestLine(gtx, h.Guest, mid)
	}
	return D{Size: size}
}

// breadcrumb draws the crumbs and the title in the first w of the row,
// their middles on mid, clipped 2dp past w as T3's nav clips them.
func (th *Theme) breadcrumb(gtx C, s *Scene, w, mid int, at image.Point, narrow bool) {
	h := s.Header
	l := th.layoutCrumbs(gtx, h, s.Title, w, narrow)
	defer clip.Rect{Max: image.Pt(w+dp(gtx, 2), gtx.Constraints.Max.Y)}.Push(gtx.Ops).Pop()
	x := 0
	sep := func() {
		off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
		th.capText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.IconMuted}, "/", mid)
		off.Pop()
		x += l.sep + l.gap
	}
	projW, parentW, collapsed := l.project, l.parent, l.collapsed
	if h.Project != "" {
		th.crumb(gtx, "header.project", "New thread in "+h.Project, image.Pt(x, mid), projW, func(gtx C, col color.NRGBA, w int) {
			off := op.Offset(image.Pt(0, -dp(gtx, crumbIcon)/2)).Push(gtx.Ops)
			th.projectIcon(gtx, h.Icon, crumbIcon, rowInk{bg: P.Background, k: 1}, 0)
			off.Pop()
			tx := dp(gtx, crumbIcon+6)
			g := gtx
			g.Constraints.Max.X = max(0, w-tx)
			defer op.Offset(image.Pt(tx, 0)).Push(gtx.Ops).Pop()
			th.capText(g, crumbFont(col), h.Project, 0)
		})
		x += projW + l.gap
		sep()
	}
	if h.Parent != "" {
		th.crumb(gtx, "header.parent", h.Parent, image.Pt(x, mid), parentW, func(gtx C, col color.NRGBA, w int) {
			label := h.Parent
			if collapsed {
				label = "..."
			}
			th.capText(gtx, crumbFont(col), label, 0)
		})
		x += parentW + l.gap
		sep()
	}
	th.titleCrumb(gtx, s, image.Pt(x, mid), l.title, at)
}

// guestFont is the guest link's type: 12/16 in muted foreground.
func guestFont() Txt { return Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1} }

// stopSharing is the guest link's button: xs, ghost-muted, off while the
// close is on its way.
func stopSharing(g *GuestLine) Button {
	return Button{Variant: BtnGhostMuted, Size: BtnSizeXs, Label: "Stop sharing", Disabled: g.Closing}
}

// guestLineW is the guest link's width: the 14dp icon, its status and
// Stop sharing, 4dp apart.
func (th *Theme) guestLineW(gtx C, g *GuestLine) int {
	m := op.Record(gtx.Ops)
	b := th.T3Button("", stopSharing(g))(gtx).Size.X
	m.Stop()
	return dp(gtx, crumbIcon+4) + th.textWidth(gtx, guestFont(), g.Status) + dp(gtx, 4) + b
}

// guestLine is a shared thread's guest link (GuestShareIndicator), its
// middle on mid: the user icon, who it's shared with or that the link is
// active or expired, and Stop sharing, a stop for Tab after the title.
// Its 16dp line centres as the caps do: in Inter and SF Pro the two
// middles are within a twentieth of a dp.
func (th *Theme) guestLine(gtx C, g *GuestLine, mid int) {
	const id = "header.unshare"
	icon := dp(gtx, crumbIcon)
	func() {
		defer op.Offset(image.Pt(0, mid-icon/2)).Push(gtx.Ops).Pop()
		LucideW("user", P.MutedForeground, crumbIcon)(gtx)
	}()
	x := icon + dp(gtx, 4)
	func() {
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		x += th.capText(gtx, guestFont(), g.Status, mid) + dp(gtx, 4)
	}()
	focused := false
	if !g.Closing {
		focused = th.keyTarget(gtx, id)
	}
	b := stopSharing(g)
	h := dp(gtx, buttonSizes[b.Size].h)
	defer op.Offset(image.Pt(x, mid-h/2)).Push(gtx.Ops).Pop()
	d := th.T3Button(id, b)(gtx)
	if focused {
		// T3's control ring, inside the button's 1dp border: 1dp of
		// --background, then 2dp of --ring. The label is clear of it, so
		// it can go on top.
		px := max(1, dp(gtx, 1))
		r, rad := image.Rectangle{Max: d.Size}, dp(gtx, 8)
		strokeRect(gtx, P.Background, r.Inset(px), rad-px, float32(px))
		strokeRect(gtx, P.Ring, r.Inset(2*px), rad-2*px, float32(2*px))
	}
}

// crumbs is the breadcrumb laid out across its list: each crumb's width
// (0 for one that isn't there), the separators' and the gap between
// items, and whether the parent's title folded to "...".
type crumbs struct {
	project, parent, title int
	sep, gap               int
	collapsed              bool
}

// layoutCrumbs fits h's crumbs and title into w (WorkspaceBreadcrumb.tsx,
// docs/ui/05-header.md §3.1): the project crumb's text caps at 160dp, the
// parent never shrinks but folds to "..." once the crumbs, uncut, would
// overflow (ChatHeader.tsx:120–158), and the title grows from nothing into
// what's left, down to its 40dp minimum; only then does the project crumb
// give way.
func (th *Theme) layoutCrumbs(gtx C, h *ChatHeader, title string, w int, narrow bool) crumbs {
	l := crumbs{gap: dp(gtx, 12), sep: th.textWidth(gtx, Txt{Size: 14, LineH: 20.0 / 14}, "/")}
	if narrow {
		l.gap = dp(gtx, 8)
	}
	muted := P.MutedForeground
	if h.Project != "" {
		l.project = dp(gtx, crumbIcon+6) + min(th.textWidth(gtx, crumbFont(muted), h.Project), dp(gtx, crumbTextMax))
	}
	ellipsis := 0
	if h.Parent != "" {
		l.parent, ellipsis = th.textWidth(gtx, crumbFont(muted), h.Parent), th.textWidth(gtx, crumbFont(muted), "...")
	}
	full := th.textWidth(gtx, crumbFont(P.Foreground), title)
	if h.Server {
		full += dp(gtx, 4+crumbIcon)
	}
	if h.Origin != "" && !h.Renaming {
		full += th.badgeSize(gtx, BadgeSizeSm, "From "+h.Origin).X
	}
	items, fixed := 1, 0
	if h.Project != "" {
		items, fixed = items+2, fixed+l.sep
	}
	if h.Parent != "" {
		items, fixed = items+2, fixed+l.sep
	}
	gaps := (items - 1) * l.gap
	if h.Parent != "" && l.project+fixed+l.parent+full+gaps > w {
		l.collapsed, l.parent = true, ellipsis
	}
	fixed += l.parent
	l.title = w - gaps - fixed - l.project
	if l.title < dp(gtx, titleMin) {
		l.title = dp(gtx, titleMin)
		l.project = max(0, w-gaps-fixed-l.title)
	}
	return l
}

// headerRow is the title row on its own, for a window without the
// shell's columns: the header's breadcrumb, the sidebar's trigger at its
// start and the panel toggles at its end, padded as T3's is below 640dp
// when narrow.
func (th *Theme) headerRow(gtx C, s *Scene, narrow bool) D {
	return FixedH(gtx, topBarH, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
		left, end := th.contentLeft(), float32(20)
		if narrow {
			left, end = 56, 12
		}
		g := gtx
		g.Constraints = layout.Exact(size)
		pad(0, end+96, 0, left).Layout(g, func(gtx C) D { return th.chatHeader(gtx, s, image.Pt(dp(gtx, left), 0), narrow) })
		th.sidebarTrigger(gtx, s, false)
		th.titlebarControls(gtx, s, size, false)
		return D{Size: size}
	})
}

// crumbBox is a crumb's height: its cap height and 6dp above and below.
func crumbBox(gtx C) int { return dp(gtx, 12) + int(capEm*float32(gtx.Sp(14))+0.5) }

// crumb is a breadcrumb button w wide whose box is centred on at.Y: muted,
// turning foreground on hover over 150ms, with T3's tooltip below it and
// the inset focus ring. draw lays its content out from the box's left at
// the cap band's middle, in the crumb's colour.
func (th *Theme) crumb(gtx C, id, tip string, at image.Point, w int, draw func(gtx C, col color.NRGBA, w int)) {
	box := crumbBox(gtx)
	defer op.Offset(image.Pt(at.X, at.Y-box/2)).Push(gtx.Ops).Pop()
	c, _ := th.events(gtx, id)
	focused := th.keyTarget(gtx, id)
	t := th.transition(gtx, id, c.Hovered(), transitionDur)
	col := lerpCol(P.MutedForeground, P.Foreground, t)
	func() {
		defer clip.Rect{Max: image.Pt(w, box)}.Push(gtx.Ops).Pop()
		defer op.Offset(image.Pt(0, box/2)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(w, box)}
		draw(g, col, w)
	}()
	r := image.Rect(0, 0, w, box)
	if focused {
		strokeRect(gtx, P.Ring, r, dp(gtx, 4), float32(dp(gtx, 2)))
	}
	th.hit(gtx, id, r.Max)
	th.tipAt(gtx, id, tip, r.Max, false, 0)
}

// titleCrumb is the current crumb, w wide from at: a server thread's title
// is a button that opens its menu (a chevron fades in on hover or focus)
// and renames on a double click; a draft's is plain text; while renaming
// it's an editor. A linked thread's "From <origin>" badge follows.
func (th *Theme) titleCrumb(gtx C, s *Scene, at image.Point, w int, win image.Point) {
	h := s.Header
	if h.Renaming {
		th.renameField(gtx, at, w, win)
		return
	}
	box := crumbBox(gtx)
	defer op.Offset(image.Pt(at.X, at.Y-box/2)).Push(gtx.Ops).Pop()
	badge := image.Point{}
	if h.Origin != "" {
		badge = th.badgeSize(gtx, BadgeSizeSm, "From "+h.Origin)
	}
	textMax := max(0, w-badge.X)
	if !h.Server {
		// A draft's title is an h2 as wide as the crumb: no menu, no
		// rename, not a tab stop, but it still has its tooltip.
		func() {
			defer op.Offset(image.Pt(0, box/2)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints.Max.X = textMax
			th.capText(g, crumbFont(P.Foreground), s.Title, 0)
		}()
		th.hover(gtx, "header.draft", image.Pt(textMax, box))
		th.tipAt(gtx, "header.draft", s.Title, image.Pt(textMax, box), false, 0)
		th.originBadge(gtx, h.Origin, textMax, box)
		return
	}
	id := "header.title"
	if th.in.doubles == nil {
		th.in.doubles = map[string]bool{}
	}
	th.in.doubles[id] = h.CanRename
	c, _ := th.events(gtx, id)
	focused := th.keyTarget(gtx, id)
	chev := dp(gtx, 4+crumbIcon)
	textW := min(th.textWidth(gtx, crumbFont(P.Foreground), s.Title), max(0, textMax-chev))
	bw := textW + chev
	func() {
		defer op.Offset(image.Pt(0, box/2)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints.Max.X = textW
		th.capText(g, crumbFont(P.Foreground), s.Title, 0)
	}()
	if a := th.transition(gtx, id, c.Hovered() || focused, transitionDur); a > 0 {
		func() {
			defer op.Offset(image.Pt(textW+dp(gtx, 4), (box-dp(gtx, crumbIcon))/2)).Push(gtx.Ops).Pop()
			defer paint.PushOpacity(gtx.Ops, a).Pop()
			LucideW("chevron-down", P.MutedForeground, crumbIcon)(gtx)
		}()
	}
	r := image.Rect(0, 0, bw, box)
	if focused {
		strokeRect(gtx, P.Ring, r, dp(gtx, 4), float32(dp(gtx, 2)))
	}
	// The menu drops 4dp below the title's left edge.
	th.setAnchor(id, win.Add(image.Pt(at.X, at.Y-box/2+box+dp(gtx, 4))))
	th.hit(gtx, id, r.Max)
	th.tipAt(gtx, id, s.Title, r.Max, false, 0)
	th.originBadge(gtx, h.Origin, bw, box)
}

// originBadge is the "From <origin>" info badge after the title, centred
// on the crumb's box, with its tooltip.
func (th *Theme) originBadge(gtx C, origin string, x, box int) {
	if origin == "" {
		return
	}
	label := "From " + origin
	size := th.badgeSize(gtx, BadgeSizeSm, label)
	defer op.Offset(image.Pt(x, (box-size.Y)/2)).Push(gtx.Ops).Pop()
	th.T3Badge(BadgeInfo, BadgeSizeSm, "", label, P.Background)(gtx)
	th.hover(gtx, "header.origin", size)
	th.tipAt(gtx, "header.origin", "Started by an agent on "+origin+" through a linked environment.", size, false, 0)
}

// renameField is the title's rename editor, w wide from at: 20dp tall,
// transparent, inside a 1dp ring at half strength that goes full while it
// has the focus. Enter commits, Escape cancels, and a press outside it
// commits, as blurring T3's input does.
func (th *Theme) renameField(gtx C, at image.Point, w int, win image.Point) {
	const id = "header.title"
	hgt := dp(gtx, 20)
	defer op.Offset(image.Pt(at.X, at.Y-hgt/2)).Push(gtx.Ops).Pop()
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit = true, true
	r := image.Rect(0, 0, w, hgt)
	ring := alpha(P.Ring, 0x80)
	if gtx.Focused(ed) {
		ring = P.Ring
	}
	g := gtx
	g.Constraints = layout.Exact(r.Max)
	th.editor(g, id, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.Foreground}, "")
	strokeRect(gtx, over(ring, P.Background), r, dp(gtx, 4), float32(max(1, dp(gtx, 1))))
	if th.in.pressed && !th.in.pressAt.In(r.Add(win).Add(image.Pt(at.X, at.Y-hgt/2))) {
		th.in.pressed = false
		th.push(Action{ID: id, Kind: "blur", Text: ed.Text()})
	}
}

// hover is hit without the pointer cursor, for text that only has a
// tooltip, or a surface that keeps its presses. Right-clicks on it still
// report, as they bubble up to its container's menu in T3.
func (th *Theme) hover(gtx C, id string, size image.Point) {
	c, rc := th.events(gtx, id)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
	event.Op(gtx.Ops, rc)
}

// setAnchor remembers where the menu target id opens, in the window.
func (th *Theme) setAnchor(id string, at image.Point) {
	if th.in.anchors == nil {
		th.in.anchors = map[string]image.Point{}
	}
	th.in.anchors[id] = at
}

// MenuFrom places the next right-click style menu where target id opens
// one: the header's title drops the thread's menu under itself.
func (th *Theme) MenuFrom(id string) {
	if at, ok := th.in.anchors[id]; ok {
		th.in.ctxAt = at
	}
}

// SelectAll selects all of the editor id's text, as T3's rename input does
// when it takes the focus.
func (th *Theme) SelectAll(id string) {
	ed := th.Editor(id)
	ed.SetCaret(ed.Len(), 0)
}
