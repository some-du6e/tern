package ui

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// T3's thread details card (ThreadDetailsCard.tsx, ThreadDetailsPanel.tsx,
// ThreadDetailsControl.tsx, threadDetailsPanelStyles.ts;
// docs/ui/05-header.md §2.5–2.7, §3.5–3.8, §4.3–4.5): a glass card of
// sections, each a column of 32dp rows, inline at the canvas's top right
// or as a popover off the header.

// detailsW is the thread details card's width (--thread-details-panel-width).
const detailsW = cardW

const (
	rowH      = 32  // h-8
	rowLabelX = 37  // every label starts here: border, padding 10, icon 16, gap 10
	halfW     = 229 // a split's primary, or a select row's label half
	cardFade  = 24  // the scroll area's --fade-size
)

// iconToggle is an icon button with a lucide icon, lit while what it opens
// is open.
func (th *Theme) iconToggle(icon string, on bool) W {
	return func(gtx C) D {
		bg, col := color.NRGBA{}, ColDim
		if on {
			bg, col = ColFrameActive, ColText
		}
		return Box{Bg: bg, Radius: 6, Pad: pad(6, 6, 6, 6)}.Layout(gtx, LucideW(icon, col, 16))
	}
}

// popIn plays a popover's opening the way T3's do: over 150ms on its
// ease-in-out it fades in, growing from 98% about origin (in its own box)
// or, with none, rising a little. Each time id opens again it plays again.
func (th *Theme) popIn(gtx C, id string, open bool, origin *f32.Point, w W) D {
	if th.in.popAt == nil {
		th.in.popAt = map[string]time.Time{}
	}
	if !open {
		delete(th.in.popAt, id)
		return D{}
	}
	start, ok := th.in.popAt[id]
	if !ok {
		start = gtx.Now
		th.in.popAt[id] = start
	}
	t := easeInOut(since(gtx.Now, start, transitionDur))
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	defer paint.PushOpacity(gtx.Ops, t).Pop()
	if origin != nil {
		k := 0.98 + 0.02*t
		defer op.Affine(f32.Affine2D{}.Scale(*origin, f32.Pt(k, k))).Push(gtx.Ops).Pop()
	} else {
		defer op.Offset(image.Pt(0, int((1-t)*float32(dp(gtx, 4))))).Push(gtx.Ops).Pop()
	}
	return w(gtx)
}

// The card's density (threadDetailsCardLayout.ts): everything; then
// without the machine row, the Git actions row (unless one is running or
// just finished) and the Version Control rule; then also without Open in.
type density int

const (
	densityFull density = iota
	densityCompact
	densityEssential
)

// cardFill is the card's glass, popover at 83.6%, as it paints over the
// canvas: Gio has no backdrop blur, so this is T3's flat case.
func cardFill() color.NRGBA { return over(alpha(P.Popover, 0xd5), P.Background) }

// rowTint is a row's hover fill, black 3.5% or white 8% in dark mode.
func rowTint() color.NRGBA {
	if dark {
		return alpha(rgb(0xffffff), 0x14)
	}
	return alpha(rgb(0), 0x09)
}

// rowText is the rows' text, foreground at 80%.
func rowText() color.NRGBA { return over(alpha(P.Foreground, 0xcc), cardFill()) }

// rowFont is the rows' type: 13/19.5, medium.
func rowFont(col color.NRGBA) Txt {
	return Txt{Size: 13, LineH: 19.5 / 13, Font: UI(font.Medium), Color: col}
}

// densityMemo is the density the card settled on, for the card it was
// given, its width and the room it had: the scene builds a new card
// whenever what's in it changes, so the card is measured once per build,
// not every frame (05-header.md port note 6).
type densityMemo struct {
	card     *ThreadDetails
	w, avail int
	dn       density
}

// details draws the card in gtx.Constraints.Max (its width and the most
// it may grow to), judging its density against avail, and scrolls what
// doesn't fit with fades at the overflowing edges. It returns the card's
// size.
func (th *Theme) details(gtx C, s *Scene, avail int) D {
	w, most := gtx.Constraints.Max.X, gtx.Constraints.Max.Y
	px := max(1, dp(gtx, 1))
	measure := func(dn density) int {
		m := op.Record(gtx.Ops)
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(w-2*px, 1<<20)}
		th.in.measuring = true
		h := th.cardContent(g, s, dn, true).Size.Y
		th.in.measuring = false
		m.Stop()
		return h + 2*px
	}
	memo := &th.in.density
	if memo.card != s.Card || memo.w != w || memo.avail != avail {
		dn := densityFull
		if full := measure(densityFull); full > avail {
			dn = densityCompact
			if measure(densityCompact) > avail {
				dn = densityEssential
			}
		}
		*memo = densityMemo{card: s.Card, w: w, avail: avail, dn: dn}
	}
	dn := memo.dn
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(w-2*px, 1<<20)}
	content := th.cardContent(g, s, dn, false)
	call := m.Stop()
	h := min(content.Size.Y+2*px, most)
	r := image.Rect(0, 0, w, h)
	rad := dp(gtx, 16)
	fill(gtx, cardFill(), r, rad)
	// The card takes the presses between its rows, so they don't fall
	// through to what's under it.
	th.hover(gtx, "card", r.Max)

	// The scroll area, inside the border.
	view := image.Rect(px, px, w-px, h-px)
	overflow := content.Size.Y - view.Dy()
	if th.in.cardScroll > overflow {
		th.in.cardScroll = max(0, overflow)
	}
	if overflow > 0 {
		d := th.in.cardScrollG.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical,
			pointer.ScrollRange{}, pointer.ScrollRange{Min: -th.in.cardScroll, Max: overflow - th.in.cardScroll})
		th.in.cardScroll = max(0, min(overflow, th.in.cardScroll+d))
	}
	func() {
		defer op.Offset(view.Min).Push(gtx.Ops).Pop()
		defer clip.UniformRRect(image.Rectangle{Max: view.Size()}, rad-px).Push(gtx.Ops).Pop()
		if overflow > 0 {
			th.in.cardScrollG.Add(gtx.Ops)
		}
		func() {
			defer op.Offset(image.Pt(0, -th.in.cardScroll)).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
		}()
		// Each overflowing edge fades over as much of 24dp as it overflows by.
		top, bottom := min(dp(gtx, cardFade), th.in.cardScroll), min(dp(gtx, cardFade), overflow-th.in.cardScroll)
		edgeFade(gtx, cardFill(), image.Rect(0, 0, view.Dx(), top), true)
		edgeFade(gtx, cardFill(), image.Rect(0, view.Dy()-bottom, view.Dx(), view.Dy()), false)
	}()
	strokeRect(gtx, over(alpha(P.ContrastForeground, 0x1a), cardFill()), r, rad, float32(px))
	return D{Size: r.Max}
}

// edgeFade paints col over r, opaque at the top (down) or bottom edge and
// clear at the other, the way a mask fades content into what's behind it.
func edgeFade(gtx C, col color.NRGBA, r image.Rectangle, down bool) {
	if r.Dy() <= 0 {
		return
	}
	from, to := f32.Pt(0, float32(r.Min.Y)), f32.Pt(0, float32(r.Max.Y))
	if !down {
		from, to = to, from
	}
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	paint.LinearGradientOp{Stop1: from, Color1: col, Stop2: to, Color2: alpha(col, 0)}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// cardContent is the card's sections at density dn, gtx.Constraints.Max.X
// wide (the scroll area's 278). Measuring for the density, it leaves out
// the lists that scroll rather than drop sections (a thread's other pull
// requests), as T3's measure does.
func (th *Theme) cardContent(gtx C, s *Scene, dn density, measuring bool) D {
	d := s.Card
	w := gtx.Constraints.Max.X
	if d.Picker == nil {
		// The branch picker opens at the top of its list.
		delete(th.in.lists, "branch.refs")
		th.in.pickHi = 0
	}
	// Each row with the popup it opens, which hangs 4dp under it as wide
	// as it is.
	type row struct {
		w     W
		popup string
	}
	var workspace, vcs []row
	if d.Run != nil && dn == densityFull {
		workspace = append(workspace, row{th.runRow(d), "run"})
	}
	if d.OpenIn != nil && dn != densityEssential {
		workspace = append(workspace, row{th.openInRow(d), "openin"})
	}
	if d.Scripts != nil {
		workspace = append(workspace, row{th.scriptsRow(d), "scripts"})
	}
	if d.Branch != nil {
		vcs = append(vcs, row{th.branchRow(d), "branch"})
	}
	if d.PRs != nil {
		vcs = append(vcs, row{th.prRows(d.PRs, measuring), ""})
	}
	if d.InitGit {
		vcs = append(vcs, row{th.initGitRow(d), ""})
	}
	// Compact density drops the Git actions row unless it's showing an
	// action at work or its result.
	if g := d.Git; g != nil && (dn == densityFull || g.Running != nil || g.Success != nil) {
		vcs = append(vcs, row{th.gitRow(d), "git"})
	}
	if d.Changes != nil {
		vcs = append(vcs, row{th.changesRow(d), ""})
	}
	rowW := w - dp(gtx, 16)
	y := 0
	section := func(rows []row, rule bool) {
		if len(rows) == 0 {
			return
		}
		if rule && y > 0 {
			// The divider, border at 65%, only after a section with rows.
			fill(gtx, over(faded(P.Border, 0.65), cardFill()), image.Rect(0, y, w, y+max(1, dp(gtx, 1))), 0)
			y += max(1, dp(gtx, 1))
		}
		y += dp(gtx, 8)
		for _, r := range rows {
			off := op.Offset(image.Pt(dp(gtx, 8), y)).Push(gtx.Ops)
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(rowW, 1<<20)}
			h := r.w(g).Size.Y
			off.Pop()
			at := image.Pt(dp(gtx, 8), y+h+dp(gtx, 4))
			switch {
			case measuring:
			case r.popup == "branch" && d.Picker != nil:
				th.deferBelow(gtx, at, func(gtx C) D { return th.branchPicker(gtx, d.Picker, rowW) })
			case r.popup != "" && d.Menu != nil && d.Menu.For == r.popup:
				th.deferBelow(gtx, at, func(gtx C) D { return th.cardMenu(gtx, d.Menu, rowW) })
			}
			y += h
		}
		y += dp(gtx, 10)
	}
	section(workspace, false)
	section(vcs, dn == densityFull)
	if y == 0 {
		// With every row hidden, a way to bring them back, padded 8; the
		// items menu hangs from its bottom left.
		defer op.Offset(image.Pt(dp(gtx, 8), dp(gtx, 8))).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(rowW, 1<<20)}
		open := d.Menu != nil && d.Menu.For == "items"
		h := th.row(g, ctl{id: "card.choose", popup: true, open: open}, "sliders-horizontal", 16, "Choose what to show").Size.Y
		if open && !d.Menu.AtPointer && !measuring {
			th.deferBelow(gtx, image.Pt(0, h+dp(gtx, 4)), func(gtx C) D { return th.cardMenu(gtx, d.Menu, dp(gtx, 232)) })
		}
		return D{Size: image.Pt(w, h+dp(gtx, 16))}
	}
	return D{Size: image.Pt(w, y)}
}

// ---------- the rows' parts ----------

// ctl is one of the card's controls (ThreadDetailsControl).
type ctl struct {
	id       string
	disabled bool // :disabled at 64%, taking no pointer
	explains bool // aria-disabled at 64%: it still hovers, to explain itself
	popup    bool // opens a menu: no press scale
	open     bool // its menu is open: the hover fill stays
	tip      string
}

// state handles c's events and reports whether it's hovered, pressed and
// showing keyboard focus.
func (th *Theme) state(gtx C, c ctl) (hovered, pressed, focused bool) {
	if c.disabled || c.id == "" {
		return false, false, false
	}
	g, _ := th.events(gtx, c.id)
	focused = th.keyTarget(gtx, c.id)
	if th.in.focusCard && !th.in.measuring {
		// The first control the card draws at its density takes the focus.
		th.in.focusCard = false
		th.focusTarget(gtx, c.id)
	}
	return g.Hovered(), g.Pressed(), focused
}

// corners is a rect's four radii, clockwise from the top left.
type corners struct{ nw, ne, se, sw int }

func (k corners) fill(gtx C, col color.NRGBA, r image.Rectangle) {
	paint.FillShape(gtx.Ops, col, clip.RRect{Rect: r, NW: k.nw, NE: k.ne, SE: k.se, SW: k.sw}.Op(gtx.Ops))
}

func (k corners) inset(d int) corners {
	return corners{max(0, k.nw-d), max(0, k.ne-d), max(0, k.se-d), max(0, k.sw-d)}
}

// focusRing is a control's keyboard focus, over under (the colour it sits
// on): inset box shadows start inside the 1dp transparent border, so 1dp
// of --background, then 2dp of --ring.
func focusRing(gtx C, r image.Rectangle, k corners, under color.NRGBA) {
	b := max(1, dp(gtx, 1))
	one, three := b+dp(gtx, 1), b+dp(gtx, 3)
	k.inset(b).fill(gtx, P.Background, r.Inset(b))
	k.inset(one).fill(gtx, P.Ring, r.Inset(one))
	k.inset(three).fill(gtx, under, r.Inset(three))
}

// control draws c over r (relative to the current origin) with corners k:
// the hover or open fill on top of under, the focus ring fading in over
// 150ms, the 97% press, and its tooltip above. A disabled control fades
// to 64% whole, its fill and ring with it.
func (th *Theme) control(gtx C, c ctl, r image.Rectangle, k corners, under color.NRGBA, content func(gtx C)) {
	hovered, pressed, focused := th.state(gtx, c)
	ring, scale := float32(0), float32(1)
	if c.id != "" {
		// transition-[box-shadow,scale]: the ring fades in and out, and a
		// control that opens no menu shrinks while pressed.
		ring = th.transition(gtx, c.id+".ring", focused, transitionDur)
		if !c.popup {
			scale = 1 - 0.03*th.transition(gtx, c.id+".press", pressed, transitionDur)
		}
	}
	func() {
		if c.disabled || c.explains {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		}
		bg := under
		if hovered || c.open {
			bg = over(rowTint(), under)
			k.fill(gtx, bg, r)
		}
		if ring > 0 {
			func() {
				defer paint.PushOpacity(gtx.Ops, ring).Pop()
				focusRing(gtx, r, k, bg)
			}()
		}
		if scale != 1 {
			mid := f32.Pt(float32(r.Min.X+r.Max.X)/2, float32(r.Min.Y+r.Max.Y)/2)
			defer op.Affine(f32.Affine2D{}.Scale(mid, f32.Pt(scale, scale))).Push(gtx.Ops).Pop()
		}
		content(gtx)
	}()
	if c.id != "" && !c.disabled {
		off := op.Offset(r.Min).Push(gtx.Ops)
		th.hit(gtx, c.id, r.Size())
		th.tipAt(gtx, c.id, c.tip, r.Size(), true, 0)
		off.Pop()
	}
}

// rowIcon draws a lucide icon at the rows' icon column, centred on a 32dp
// row from x (the row's left edge).
func rowIcon(gtx C, name string, col color.NRGBA, x int, sizeDp float32) {
	s := dp(gtx, sizeDp)
	defer op.Offset(image.Pt(x+dp(gtx, 11), (dp(gtx, rowH)-s)/2)).Push(gtx.Ops).Pop()
	LucideW(name, col, sizeDp)(gtx)
}

// rowLabel draws a row's label from x, w at most, its caps centred on the row.
func (th *Theme) rowLabel(gtx C, t Txt, s string, x, w int) int {
	defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints.Max.X = max(0, w)
	return th.capText(g, t, s, dp(gtx, rowH)/2)
}

// chevron draws a split's or select row's chevron, centred in its 32dp half from x.
func chevron(gtx C, x int) {
	defer op.Offset(image.Pt(x+dp(gtx, 8), dp(gtx, 8))).Push(gtx.Ops).Pop()
	LucideW("chevron-down", P.MutedForeground, 16)(gtx)
}

// hairline is a split's 1×16 divider at x, border at 65%, shown while the
// pair is in use.
func hairline(gtx C, x int, under color.NRGBA) {
	col := over(faded(P.Border, 0.65), under)
	fill(gtx, col, image.Rect(x, dp(gtx, 8), x+max(1, dp(gtx, 1)), dp(gtx, 24)), 0)
}

// row is a 32dp row control as wide as gtx allows, with icon (size sizeDp,
// muted at 16dp, in the text colour at 14dp) and label.
func (th *Theme) row(gtx C, c ctl, icon string, sizeDp float32, label string) D {
	w := gtx.Constraints.Max.X
	r := image.Rect(0, 0, w, dp(gtx, rowH))
	th.control(gtx, c, r, corners{8, 8, 8, 8}.scaled(gtx), cardFill(), func(gtx C) {
		col := P.MutedForeground
		if sizeDp < 16 {
			col = rowText()
		}
		rowIcon(gtx, icon, col, 0, sizeDp)
		th.rowLabel(gtx, rowFont(rowText()), label, dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+11))
	})
	return D{Size: r.Max}
}

// scaled is k's radii from dp to px.
func (k corners) scaled(gtx C) corners {
	return corners{dp(gtx, float32(k.nw)), dp(gtx, float32(k.ne)), dp(gtx, float32(k.se)), dp(gtx, float32(k.sw))}
}

// split is a split group whose secondary is a chevron that opens a menu.
func (th *Theme) split(gtx C, p, sec ctl, primary func(gtx C, w int)) D {
	return th.splitWith(gtx, p, sec, primary, func(gtx C, x int, _ bool) { chevron(gtx, x) })
}

// splitWith is a split group: primary and secondary controls sharing one
// fill, with the hairline between them while the pair is hovered or its
// menu is open. primary draws the left half's content, second the 32dp
// right half's from x, told whether it's hovered.
func (th *Theme) splitWith(gtx C, p, sec ctl, primary func(gtx C, w int), second func(gtx C, x int, hovered bool)) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, rowH)
	pw := w - dp(gtx, 1+32)
	pr, sr := image.Rect(0, 0, pw, h), image.Rect(pw+dp(gtx, 1), 0, w, h)
	ph, sh := false, false
	if !p.disabled && p.id != "" {
		g, _ := th.events(gtx, p.id)
		ph = g.Hovered()
	}
	if !sec.disabled && sec.id != "" {
		g, _ := th.events(gtx, sec.id)
		sh = g.Hovered()
	}
	group := cardFill()
	if ph || sh || sec.open {
		group = over(rowTint(), group)
		fill(gtx, group, image.Rect(0, 0, w, h), dp(gtx, 8))
		hairline(gtx, pw, group)
	}
	r8 := dp(gtx, 8)
	th.control(gtx, p, pr, corners{nw: r8, sw: r8}, group, func(gtx C) { primary(gtx, pw) })
	th.control(gtx, sec, sr, corners{ne: r8, se: r8}, group, func(gtx C) { second(gtx, sr.Min.X, sh) })
	return D{Size: image.Pt(w, h)}
}

// selectRow is a row whose right 32dp is a chevron half: hovering either
// half stacks a second fill on it, and the hairline before the chevron
// shows while the row is hovered or open. label draws the left half.
func (th *Theme) selectRow(gtx C, c ctl, label func(gtx C, w int)) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, rowH)
	r := image.Rect(0, 0, w, h)
	hovered := false
	if !c.disabled && c.id != "" {
		g, _ := th.events(gtx, c.id)
		hovered = g.Hovered()
	}
	px := max(1, dp(gtx, 1))
	half := w - dp(gtx, 32) // the chevron half overlaps the right border by 1dp
	left, right := image.Rect(px, px, half, h-px), image.Rect(half, px, w, h-px)
	onLeft, onRight := th.hoverState(gtx, c.id+".label"), th.hoverState(gtx, c.id+".icon")
	th.control(gtx, c, r, corners{8, 8, 8, 8}.scaled(gtx), cardFill(), func(gtx C) {
		under := cardFill()
		if hovered || c.open {
			under = over(rowTint(), under)
			hairline(gtx, half-px, under)
		}
		// The half under the pointer gets a fill of its own.
		r8 := dp(gtx, 8) - px
		switch {
		case hovered && onLeft:
			corners{nw: r8, sw: r8}.fill(gtx, over(rowTint(), under), left)
		case hovered && onRight:
			corners{ne: r8, se: r8}.fill(gtx, over(rowTint(), under), right)
		}
		label(gtx, half)
		chevron(gtx, half)
	})
	if !c.disabled && c.id != "" {
		func() {
			defer op.Offset(left.Min).Push(gtx.Ops).Pop()
			th.hoverArea(gtx, c.id+".label", left.Size())
		}()
		defer op.Offset(right.Min).Push(gtx.Ops).Pop()
		th.hoverArea(gtx, c.id+".icon", right.Size())
	}
	return D{Size: r.Max}
}

// ---------- the rows ----------

// machineIcons are the machine kinds' marks (EnvironmentMachineIcon.tsx);
// Mac mini and Mac Studio are T3's own drawings, the rest lucide's.
var machineIcons = map[string]string{
	"server": "server", "cloud": "cloud", "linux": "server", "desktop": "monitor", "laptop": "laptop",
	"mac-mini": "mac-mini", "mac-studio": "mac-studio",
}

// runRow is the machine and workspace row: the machine's mark (18dp, a
// pixel into the padding each side), its name, and the workspace's icon at
// the end, or with no machine the workspace's icon leading. A started
// thread's is a plain row; a draft's opens its menu. Each icon has its
// tooltip over the row, which still takes the pointer's presses there, so
// a right-click on an icon is the row's.
func (th *Theme) runRow(d *ThreadDetails) W {
	return func(gtx C) D {
		rc := d.Run
		w, h := gtx.Constraints.Max.X, dp(gtx, rowH)
		end := w - dp(gtx, 1)
		if !rc.Locked {
			end = w - dp(gtx, 32) // the chevron half
		}
		trailing := end - dp(gtx, 10+16)
		s16, s18 := dp(gtx, 16), dp(gtx, 18)
		ws := image.Rect(dp(gtx, 11), (h-s16)/2, dp(gtx, 11)+s16, (h-s16)/2+s16)
		var mark image.Rectangle
		if rc.Machine != "" {
			mark = image.Rect(dp(gtx, 10), (h-s18)/2, dp(gtx, 10)+s18, (h-s18)/2+s18)
			ws = ws.Add(image.Pt(trailing-dp(gtx, 11), 0))
		}
		content := func(gtx C, col color.NRGBA) {
			if rc.Machine != "" {
				icon := machineIcons[rc.Machine]
				if icon == "" {
					icon = "server"
				}
				off := op.Offset(mark.Min).Push(gtx.Ops)
				LucideW(icon, P.MutedForeground, 18)(gtx)
				off.Pop()
			}
			off := op.Offset(ws.Min).Push(gtx.Ops)
			LucideW(rc.Workspace, P.MutedForeground, 16)(gtx)
			off.Pop()
			th.rowLabel(gtx, rowFont(col), rc.Label, dp(gtx, rowLabelX), trailing-dp(gtx, 10+rowLabelX))
		}
		var dims D
		if rc.Locked {
			col := rowText()
			if dark {
				col = P.MutedForeground
			}
			// Not a button, but its right-click copies the path.
			th.hover(gtx, "card.runlocked", image.Pt(w, h))
			content(gtx, col)
			dims = D{Size: image.Pt(w, h)}
		} else {
			dims = th.selectRow(gtx, ctl{id: "card.run", popup: true, open: d.Menu != nil && d.Menu.For == "run"}, func(gtx C, half int) {
				content(gtx, rowText())
			})
		}
		if rc.Machine != "" {
			th.tipArea(gtx, "card.machine", rc.Tip, mark)
		}
		th.tipArea(gtx, "card.workspace", rc.WorkspaceTip, ws)
		return dims
	}
}

// openInRow is the Open in editor split: the preferred editor's mark and
// "Open in <Editor>", and the chevron that picks another.
func (th *Theme) openInRow(d *ThreadDetails) W {
	return func(gtx C) D {
		o := d.OpenIn
		return th.split(gtx,
			ctl{id: "card.openin", disabled: o.Disabled, tip: o.Label},
			ctl{id: "card.editors", popup: true, open: d.Menu != nil && d.Menu.For == "openin", tip: "Choose editor"},
			func(gtx C, w int) {
				// The preferred editor's mark at the icon column.
				s := dp(gtx, 16)
				func() {
					defer op.Offset(image.Pt(dp(gtx, 11), (dp(gtx, rowH)-s)/2)).Push(gtx.Ops).Pop()
					editorIcon(o.Editor, P.MutedForeground, 16)(gtx)
				}()
				th.rowLabel(gtx, rowFont(rowText()), o.Label, dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+10))
			})
	}
}

// scriptIcons are the project script icons' lucide names (ProjectScriptsControl.tsx).
var scriptIcons = map[string]string{
	"play": "play", "test": "flask-conical", "lint": "list-checks", "configure": "wrench", "build": "hammer", "debug": "bug",
}

// ScriptIcon is a project script icon's lucide name, play by default.
func ScriptIcon(kind string) string {
	if icon := scriptIcons[kind]; icon != "" {
		return icon
	}
	return "play"
}

// scriptsRow is the project's scripts: a split that runs the primary
// script, a split whose halves both open the menu when there's none, or a
// row that adds the first. In the Actions form the primary isn't a menu
// trigger to T3 (no aria-haspopup), so a press shrinks it.
func (th *Theme) scriptsRow(d *ThreadDetails) W {
	return func(gtx C) D {
		sc := d.Scripts
		open := d.Menu != nil && d.Menu.For == "scripts"
		switch {
		case sc.Primary != "":
			return th.split(gtx,
				ctl{id: "card.script", tip: "Run " + sc.Primary},
				ctl{id: "card.scripts", popup: true, open: open, tip: "Script actions"},
				func(gtx C, w int) {
					rowIcon(gtx, ScriptIcon(sc.Icon), P.MutedForeground, 0, 16)
					th.rowLabel(gtx, rowFont(rowText()), sc.Primary, dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+10))
				})
		case sc.Any:
			return th.split(gtx,
				ctl{id: "card.scripts", open: open},
				ctl{id: "card.scripts.more", popup: true, open: open},
				func(gtx C, w int) {
					rowIcon(gtx, "wrench", P.MutedForeground, 0, 16)
					th.rowLabel(gtx, rowFont(rowText()), "Actions", dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+10))
				})
		}
		// T3's row opens its Add Action dialog, which tern doesn't have, so
		// it stays off here.
		return th.row(gtx, ctl{id: "card.addscript", disabled: true}, "plus", 14, "Add project script")
	}
}

// branchRow is the branch picker's trigger: the branch, middle-truncated
// so its end stays readable.
func (th *Theme) branchRow(d *ThreadDetails) W {
	return func(gtx C) D {
		b := d.Branch
		return th.selectRow(gtx, ctl{id: "card.branch", disabled: b.Disabled, popup: true, open: d.Picker != nil}, func(gtx C, half int) {
			rowIcon(gtx, "git-branch", P.MutedForeground, 0, 16)
			defer op.Offset(image.Pt(dp(gtx, rowLabelX), 0)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints.Max.X = max(0, half-dp(gtx, rowLabelX+10))
			th.middleTruncate(g, rowFont(rowText()), b.Label, dp(gtx, rowH)/2)
		})
	}
}

// initGitRow offers to make the folder a repository, and says so while
// it does.
func (th *Theme) initGitRow(d *ThreadDetails) W {
	return func(gtx C) D {
		label := "Initialize Git"
		if d.Initing {
			label = "Initializing..."
		}
		return th.row(gtx, ctl{id: "card.initgit", disabled: d.Initing}, "git-branch-plus", 14, label)
	}
}

// gitRow is the Git actions split: the quick action and the options
// chevron. While an action runs the pair is filled, the primary shows its
// progress and the chevron gives way to the time it's taken; a finished
// one leaves its result for ten seconds.
func (th *Theme) gitRow(d *ThreadDetails) W {
	return func(gtx C) D {
		g := d.Git
		w := gtx.Constraints.Max.X
		if g.Running != nil || g.Success != nil {
			return th.gitStatusRow(gtx, g, w)
		}
		primary := ctl{id: "card.git", tip: g.Label}
		if g.Hint != "" {
			// A hint is a popover trigger: it explains instead of acting.
			primary = ctl{id: "card.git", explains: true, popup: true}
		}
		dims := th.split(gtx, primary, ctl{id: "card.gitmenu", disabled: g.MenuDisabled, popup: true, open: d.Menu != nil && d.Menu.For == "git", tip: "Git action options"},
			func(gtx C, w int) {
				th.gitIcon(gtx, g.Icon)
				th.rowLabel(gtx, rowFont(rowText()), g.Label, dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+10))
			})
		if g.Hint != "" {
			th.reasonPopover(gtx, "card.git", g.Hint, image.Pt(w-dp(gtx, 1+32), dp(gtx, rowH)), false)
		}
		return dims
	}
}

// gitIcon is the quick action's icon at the rows' icon column: lucide's,
// or the source control host's mark.
func (th *Theme) gitIcon(gtx C, icon string) {
	if kind, ok := strings.CutPrefix(icon, "provider:"); ok {
		s := dp(gtx, 16)
		defer op.Offset(image.Pt(dp(gtx, 11), (dp(gtx, rowH)-s)/2)).Push(gtx.Ops).Pop()
		ProviderMark(kind, P.MutedForeground, 16)(gtx)
		return
	}
	rowIcon(gtx, icon, P.MutedForeground, 0, 16)
}

// gitStatusRow is the Git row at work (a spinner, the phase and the last
// output line, and the elapsed time) or done (a check, the result and its
// description).
func (th *Theme) gitStatusRow(gtx C, g *GitActions, w int) D {
	title, detail, elapsed := "", "", ""
	if p := g.Running; p != nil {
		title, detail, elapsed = p.Status, p.Output, p.Elapsed
	} else {
		title, detail = g.Success.Title, g.Success.Description
	}
	// The detail line grows in over 200ms, once, on ease-out.
	t := th.transitionEase(gtx, "card.git.detail", detail != "", 200*time.Millisecond, easeOutCSS)
	lineH := dp(gtx, 18)
	h := max(dp(gtx, rowH), dp(gtx, 12)+lineH+int(t*float32(dp(gtx, 2+16))))
	r := image.Rect(0, 0, w, h)
	if g.Running != nil {
		fill(gtx, over(gitRunTint(), cardFill()), r, dp(gtx, 8))
	}
	pw := w - dp(gtx, 1+32)
	if elapsed != "" {
		pw = w - th.textW(gtx, Txt{Size: 11, LineH: 16.0 / 11}, elapsed) - dp(gtx, 10+12)
	}
	y := dp(gtx, 6)
	// At work, the control's own 16dp spinner in its text colour, a dp
	// into each side of the padding; done, a 14dp check in --success.
	// Either way the status is 12dp on (gap-x-3).
	tx := dp(gtx, 11+14+12)
	if g.Running != nil {
		s := dp(gtx, 16)
		func() {
			defer op.Offset(image.Pt(dp(gtx, 9), y+(lineH-s)/2)).Push(gtx.Ops).Pop()
			T3Spinner(rowText(), 16, float64(gtx.Now.UnixMilli()%1000)/1000)(gtx)
		}()
		// About 30 frames a second while it spins, as the window paces
		// its other live spinners.
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(33 * time.Millisecond)})
		tx = dp(gtx, 9+16-2+12)
	} else {
		icon := dp(gtx, 14)
		func() {
			defer op.Offset(image.Pt(dp(gtx, 11), y+(lineH-icon)/2)).Push(gtx.Ops).Pop()
			LucideW("check", P.Success, 14)(gtx)
		}()
	}
	func() {
		defer op.Offset(image.Pt(tx, y)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints.Max.X = max(0, pw-tx-dp(gtx, 10))
		th.capText(g, rowFont(rowText()), title, lineH/2)
	}()
	if detail != "" {
		func() {
			defer op.Offset(image.Pt(tx, y+lineH+dp(gtx, 2))).Push(gtx.Ops).Pop()
			defer paint.PushOpacity(gtx.Ops, t).Pop()
			g := gtx
			g.Constraints.Max.X = max(0, pw-tx-dp(gtx, 10))
			th.Text(g, Txt{Size: 11, LineH: 16.0 / 11, Color: P.MutedForeground, MaxLines: 1}, detail)
		}()
	}
	if elapsed != "" {
		func() {
			ew := th.textW(gtx, Txt{Size: 11, LineH: 16.0 / 11}, elapsed)
			defer op.Offset(image.Pt(w-dp(gtx, 10)-ew, 0)).Push(gtx.Ops).Pop()
			th.capText(gtx, Txt{Size: 11, LineH: 16.0 / 11, Color: P.MutedForeground}, elapsed, dp(gtx, rowH)/2)
		}()
	}
	return D{Size: r.Max}
}

// gitRunTint fills the Git actions pair while an action runs: black 3.5%,
// white 5.5% in dark mode.
func gitRunTint() color.NRGBA {
	if dark {
		return alpha(rgb(0xffffff), 0x0e)
	}
	return alpha(rgb(0), 0x09)
}

// changesRow is the change totals: "Changes", what they're against, and
// the insertions and deletions in mono at the end, coloured once they're
// above zero. It opens the diff. Unlike the other rows' labels it isn't
// cap-trimmed: the three sit on one baseline in an items-baseline span
// (GitActionsControl.tsx:1724–1769), which the mono face's line box makes
// taller, so they sit about a dp above where a centred label's caps do.
func (th *Theme) changesRow(d *ThreadDetails) W {
	return func(gtx C) D {
		ch := d.Changes
		w := gtx.Constraints.Max.X
		r := image.Rect(0, 0, w, dp(gtx, rowH))
		th.control(gtx, ctl{id: "card.changes", disabled: ch.Disabled}, r, corners{8, 8, 8, 8}.scaled(gtx), cardFill(), func(gtx C) {
			rowIcon(gtx, "file-diff", P.MutedForeground, 0, 16)
			mono := Txt{Size: 13, LineH: 19.5 / 13, Font: font.Font{Typeface: mono, Weight: font.Medium}}
			base := rowFont(P.MutedForeground)
			base.Font = UI(font.Normal)
			y := th.baselineIn(gtx, r.Dy(), rowFont(rowText()), base, mono)
			label := func(t Txt, s string, x, w int) int {
				defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
				g := gtx
				g.Constraints.Max.X = max(0, w)
				return th.baseText(g, t, s, y)
			}
			add, del := "+"+groupDigits(ch.Insertions), "-"+groupDigits(ch.Deletions)
			addW, delW := th.textW(gtx, mono, add), th.textW(gtx, mono, del)
			end := w - dp(gtx, 11)
			delX := end - delW
			addX := delX - dp(gtx, 8) - addW
			x := dp(gtx, rowLabelX)
			x += label(rowFont(rowText()), "Changes", x, addX-x)
			if ch.Base != "" {
				bx := x + dp(gtx, 6)
				bw := min(th.textW(gtx, base, ch.Base), addX-dp(gtx, 8)-bx)
				label(base, ch.Base, bx, bw)
				off := op.Offset(image.Pt(bx, (dp(gtx, rowH)-dp(gtx, 20))/2)).Push(gtx.Ops)
				th.hover(gtx, "card.changes.base", image.Pt(max(0, bw), dp(gtx, 20)))
				th.tipAt(gtx, "card.changes.base", ch.Base, image.Pt(max(0, bw), dp(gtx, 20)), true, 0)
				off.Pop()
			}
			tone := func(n int, c color.NRGBA) color.NRGBA {
				if n == 0 {
					return P.MutedForeground
				}
				return c
			}
			mono.Color = tone(ch.Insertions, P.DiffAdditionForeground)
			label(mono, add, addX, addW)
			mono.Color = tone(ch.Deletions, P.DiffDeletionForeground)
			label(mono, del, delX, delW)
		})
		return D{Size: r.Max}
	}
}

// groupDigits writes n with English thousands separators, as
// Intl.NumberFormat does: 2382 is "2,382".
func groupDigits(n int) string {
	s := fmt.Sprint(n)
	if n < 0 {
		return "-" + groupDigits(-n)
	}
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	return string(out)
}

// reasonPopover is T3's tooltip-styled popover for an action that can't
// run (popover.tsx tooltipStyle: the glass, radius 6, 12/16 text padded 4
// by 8): it opens 300ms after the pointer reaches target id (size big at
// the current origin), 4dp below it, or with left 4dp left of it and level
// with its middle. Declared below at its start, it flips to its end, since
// the card sits at the window's right.
func (th *Theme) reasonPopover(gtx C, id, reason string, size image.Point, left bool) {
	c := th.in.clicks[id]
	if c == nil || !c.Hovered() {
		delete(th.in.tipSince, id+".reason")
		return
	}
	if th.in.tipSince == nil {
		th.in.tipSince = map[string]time.Time{}
	}
	at, ok := th.in.tipSince[id+".reason"]
	if !ok {
		at = gtx.Now
		th.in.tipSince[id+".reason"] = at
	}
	if gtx.Now.Sub(at) < 300*time.Millisecond {
		gtx.Execute(op.InvalidateCmd{At: at.Add(300 * time.Millisecond)})
		return
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 320), dp(gtx, 200))}
	td := th.tipBox(g, reason, true)
	call := m.Stop()
	at2 := image.Pt(size.X-td.Size.X, size.Y+dp(gtx, 4))
	if left {
		at2 = image.Pt(-td.Size.X-dp(gtx, 4), (size.Y-td.Size.Y)/2)
	}
	m2 := op.Record(gtx.Ops)
	op.Offset(at2).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
}

// middleTruncate draws s in the width gtx allows, its caps centred on
// mid, cutting the middle rather than the end when it doesn't fit
// (MiddleTruncate, docs/ui/02-primitives.md §5.14): the tail keeps its full
// width, the head gives up its last glyphs and "…" joins the two.
func (th *Theme) middleTruncate(gtx C, t Txt, s string, mid int) {
	w := gtx.Constraints.Max.X
	if th.textW(gtx, t, s) <= w {
		th.capText(gtx, t, s, mid)
		return
	}
	head, tail := splitMiddle(s)
	if tail == "" {
		th.capText(gtx, t, s, mid)
		return
	}
	tailW := th.textW(gtx, t, "…"+tail)
	room := w - tailW
	hr := []rune(head)
	// The longest head that fits, found by halves (fitHead).
	n := sort.Search(len(hr), func(k int) bool { return th.textW(gtx, t, string(hr[:k+1])) > room })
	x := th.capText(gtx, t, string(hr[:n]), mid)
	defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
	th.capText(gtx, t, "…"+tail, mid)
}

// startTruncate draws s on one line in the width gtx allows, its baseline
// on y, cutting its start rather than its end when it doesn't fit, as
// StartTruncatedPath's right-to-left ellipsis does: "…" and the longest
// end of s that fits after it.
func (th *Theme) startTruncate(gtx C, t Txt, s string, y int) {
	w := gtx.Constraints.Max.X
	if th.textW(gtx, t, s) <= w {
		th.baseText(gtx, t, s, y)
		return
	}
	r := []rune(s)
	// The longest end that fits, found by halves.
	n := sort.Search(len(r), func(k int) bool { return th.textW(gtx, t, "…"+string(r[len(r)-k-1:])) > w })
	th.baseText(gtx, t, "…"+string(r[len(r)-n:]), y)
}

// splitMiddle is T3's splitForMiddleTruncate: the tail is the last path
// segment when that's at most 16 long, else the last 10, moved to start on
// a word where one is near. An empty tail means s ends with the usual
// ellipsis instead.
func splitMiddle(s string) (head, tail string) {
	chars := []rune(s)
	n := len(chars)
	keep := 10
	if slash := strings.LastIndex(s, "/"); slash > 0 && slash < len(s)-1 {
		if seg := n - len([]rune(s[:slash])) - 1; seg <= 16 {
			keep = seg
		}
	}
	if keep <= 0 || n <= keep+4 {
		return s, ""
	}
	brk := func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == '/' }
	cut := n - keep
	if !brk(chars[cut-1]) {
		snapped := 0
		for i := cut + 1; i <= n-4; i++ {
			if brk(chars[i-1]) {
				snapped = i
				break
			}
		}
		if snapped == 0 {
			for i := cut - 1; i >= n-16 && i > 4; i-- {
				if brk(chars[i-1]) {
					snapped = i
					break
				}
			}
		}
		if snapped > 0 {
			cut = snapped
		}
	}
	return string(chars[:cut]), string(chars[cut:])
}

// sidebarFooter is T3's: icon buttons with tooltips. One lists the
// machines, the other is the account.
func (th *Theme) sidebarFooter(gtx C, s *Scene) D {
	return pad(6, 8, 8, 8).Layout(gtx, func(gtx C) D {
		button := func(id, icon, tip string, menu *Traits) layout.FlexChild {
			return rigid(func(gtx C) D {
				if menu != nil {
					tip = ""
				}
				dims := th.Tip(id, tip, true, -1, th.iconToggle(icon, menu != nil))(gtx)
				th.deferAbove(gtx, image.Pt(0, -dp(gtx, 1)), func(gtx C) D {
					return th.popIn(gtx, id, menu != nil, nil, func(gtx C) D { return th.traitsMenu(gtx, s, id, menu) })
				})
				return dims
			})
		}
		return Row(gtx,
			button("machines", "server", "Machines", s.Machines),
			rigid(Spacer(2, 0)),
			button("account", "circle-user", "Account", s.Account),
		)
	})
}
