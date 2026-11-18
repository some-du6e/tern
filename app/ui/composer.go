package ui

import (
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// T3's chat composer (docs/ui/07-composer.md, ChatComposer.tsx): a glass
// card in the chat lane holding the prompt editor over a footer of
// controls. What T3 attaches above the card (banners, the agent's
// questions, the @ / $ menu) is in composerdock.go; the new thread's hero
// in hero.go. All lengths are dp.

const (
	composerRadius = 16    // rounded-2xl
	drawerInset    = 22    // --chat-composer-drawer-inset: attached surfaces sit this far in
	tuck           = 17    // attached surfaces tuck 1rem + 1px under what's below them
	promptSize     = 14    // --font-size-prompt
	promptLine     = 22.75 // leading-relaxed
	promptMax      = 208   // the editor's max-h-52, padding included
	tileSize       = 64    // an attachment tile
)

// RuntimeMode is one of T3's permission modes (runtimeModeConfig.ts).
type RuntimeMode struct{ ID, Label, Desc, Icon string }

// RuntimeModes are T3's modes in its order.
var RuntimeModes = []RuntimeMode{
	{"approval-required", "Supervised", "Ask before commands and file changes.", "lock"},
	{"auto-accept-edits", "Auto-accept edits", "Auto-approve edits, ask before other actions.", "pen-line"},
	{"auto", "Auto", "Supported providers approve routine actions; others still ask.", "shield-check"},
	{"full-access", "Full access", "Allow commands and edits without prompts.", "shield-off"},
}

// runtimeMode is mode id's entry; T3 falls back to Supervised.
func runtimeMode(id string) RuntimeMode {
	for _, m := range RuntimeModes {
		if m.ID == id {
			return m
		}
	}
	return RuntimeModes[0]
}

// glass is the card's glass as Chrome paints it over the canvas:
// --chat-composer-glass-surface (--card; --surface-raised in dark) at the
// 80% glass opacity, #ffffff / #0e0e0e. Gio has no backdrop blur, so it's
// solid; the blur only shows where the timeline scrolls under the card.
func glass() color.NRGBA {
	g := P.Card
	if dark {
		g = P.SurfaceRaised
	}
	return over(alpha(g, 0xcc), P.Background)
}

// composerOutline is --chat-composer-outline, the card's 1dp edge.
func composerOutline() color.NRGBA {
	if dark {
		return alpha(rgb(0xffffff), 0x0d)
	}
	return alpha(rgb(0), 0x14)
}

// composerAnchors are where the footer's triggers landed, relative to the
// card, for the menus that open off them.
type composerAnchors struct {
	model, traits, access image.Rectangle
}

// composerArea is the thread's composer, bottom of the lane: the queue,
// the surfaces attached above the card, then the card.
func (th *Theme) composerArea(gtx C, s *Scene, compact bool) D {
	w := gtx.Constraints.Max.X
	if compact {
		// The phone gives it 10dp margins and 14 under.
		w -= 2 * dp(gtx, 10)
	}
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	m := op.Record(gtx.Ops)
	d := Col(g,
		rigid(func(gtx C) D { return th.queue(gtx, s) }),
		rigid(func(gtx C) D { return th.composerStack(gtx, s, false) }),
	)
	call := m.Stop()
	defer op.Offset(image.Pt((gtx.Constraints.Max.X-w)/2, 0)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	h := d.Size.Y
	if compact {
		h += dp(gtx, 14)
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

// composerStack is the attached surfaces, the card over their tucked
// bottoms and, for a new thread, the context strip under it.
func (th *Theme) composerStack(gtx C, s *Scene, strip bool) D {
	w := gtx.Constraints.Max.X
	if s.Live {
		th.composerChords(gtx, s)
	}
	// The dock first, so the card covers its tuck.
	dm := op.Record(gtx.Ops)
	dg := gtx
	dg.Constraints = layout.Constraints{Max: image.Pt(w, gtx.Constraints.Max.Y)}
	dock := th.composerDock(dg, s)
	dockCall := dm.Stop()
	top := 0
	if dock.Size.Y > 0 {
		top = dock.Size.Y - dp(gtx, tuck)
	}
	dockCall.Add(gtx.Ops)

	var at composerAnchors
	off := op.Offset(image.Pt(0, top)).Push(gtx.Ops)
	cg := gtx
	cg.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	cm := op.Record(gtx.Ops)
	card := th.composerCard(cg, s, &at)
	cardCall := cm.Stop()
	h := card.Size.Y
	if strip {
		// The strip sits under the card, which shadows its top edge.
		so := op.Offset(image.Pt(0, h)).Push(gtx.Ops)
		var chip image.Rectangle
		h += th.contextStrip(cg, s, &chip).Size.Y
		so.Pop()
		at.access = chip.Add(image.Pt(0, card.Size.Y))
	}
	cardCall.Add(gtx.Ops)
	th.composerMenus(gtx, s, at)
	if len(s.Suggest) > 0 || s.SuggestEmpty != "" {
		th.commandMenu(gtx, s, w)
	}
	off.Pop()
	return D{Size: image.Pt(w, top+h)}
}

// composerMenus opens tern's existing menus off the triggers that T3's
// picker would open from: above them, flush with their end (the access
// menu with the footer chip's start), 4dp up.
func (th *Theme) composerMenus(gtx C, s *Scene, at composerAnchors) {
	switch {
	case s.Picker != nil:
		th.popAbove(gtx, at.model, true, func(gtx C) D { return th.modelPicker(gtx, s, s.Picker) })
	case s.Traits != nil:
		th.popAbove(gtx, at.traits, true, func(gtx C) D { return th.traitsMenu(gtx, s, "trait", s.Traits) })
	case s.Access != nil:
		th.popAbove(gtx, at.access, !s.RuntimeChip, func(gtx C) D { return th.traitsMenu(gtx, s, "access", s.Access) })
	}
}

// popAbove draws w over everything, its bottom 4dp above anchor and its
// start (or with end, its end) lined up with anchor's.
func (th *Theme) popAbove(gtx C, anchor image.Rectangle, end bool, w W) {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 640), dp(gtx, 640))}
	d := w(g)
	call := m.Stop()
	x := anchor.Min.X
	if end {
		x = anchor.Max.X - d.Size.X
	}
	m = op.Record(gtx.Ops)
	op.Offset(image.Pt(x, anchor.Min.Y-dp(gtx, 4)-d.Size.Y)).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m.Stop())
}

// composerCard is the card: the glass with the light theme's soft
// shadow, the body and footer, and the outline on top (ComposerSurface.tsx).
func (th *Theme) composerCard(gtx C, s *Scene, at *composerAnchors) D {
	w := gtx.Constraints.Max.X
	px := max(1, dp(gtx, 1))
	g := glass()
	m := op.Record(gtx.Ops)
	inner := gtx
	inner.Constraints = layout.Constraints{Min: image.Pt(w-2*px, 0), Max: image.Pt(w-2*px, gtx.Constraints.Max.Y)}
	off := op.Offset(image.Pt(px, px)).Push(gtx.Ops)
	body := th.composerBody(inner, s)
	fo := op.Offset(image.Pt(0, body.Size.Y)).Push(gtx.Ops)
	foot := th.composerFooter(inner, s, at)
	fo.Pop()
	off.Pop()
	call := m.Stop()
	// The anchors were laid out in the footer's frame.
	shift := image.Pt(px, px+body.Size.Y)
	at.model, at.traits, at.access = at.model.Add(shift), at.traits.Add(shift), at.access.Add(shift)

	r := image.Rect(0, 0, w, 2*px+body.Size.Y+foot.Size.Y)
	rad := dp(gtx, composerRadius)
	if !dark {
		shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}})
	}
	fill(gtx, g, r, rad)
	call.Add(gtx.Ops)
	strokeRect(gtx, over(composerOutline(), g), r, rad, float32(px))
	if dark {
		// The outline's inset-shadow-2xs: 1dp of --chat-composer-highlight
		// inside the top edge.
		hline(gtx, over(alpha(rgb(0xffffff), 0x08), g), px, rad, w-rad)
	}
	return D{Size: r.Max}
}

// composerBody is the attachment tiles and the prompt editor, padded 16
// 16 8 (data-chat-composer-body).
func (th *Theme) composerBody(gtx C, s *Scene) D {
	w := gtx.Constraints.Max.X
	return pad(16, 16, 8, 16).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Col(gtx,
			rigid(func(gtx C) D {
				if len(s.Thumbs) == 0 {
					return D{}
				}
				return pad(0, 0, 12, 0).Layout(gtx, func(gtx C) D { return th.mediaRow(gtx, s) })
			}),
			rigid(func(gtx C) D { return th.prompt(gtx, s, w-dp(gtx, 32)) }),
		)
	})
}

// mediaRow is the composer's pictures as 64dp tiles, 8 apart, wrapping.
func (th *Theme) mediaRow(gtx C, s *Scene) D {
	size, gap := dp(gtx, tileSize), dp(gtx, 8)
	per := max(1, (gtx.Constraints.Max.X+gap)/(size+gap))
	rows := (len(s.Thumbs) + per - 1) / per
	for i, t := range s.Thumbs {
		off := op.Offset(image.Pt(i%per*(size+gap), i/per*(size+gap))).Push(gtx.Ops)
		th.composerTile(t, i, s.Live)(gtx)
		off.Pop()
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, rows*size+(rows-1)*gap)}
}

// promptText is the editor's type: 14/22.75 in the UI font.
func promptText(col color.NRGBA) Txt {
	return Txt{Size: promptSize, LineH: promptLine / promptSize, Color: col, Font: UI(font.Normal)}
}

// lineBox is how CSS sets t's lines and Gio doesn't: each LineH × Size
// tall with the glyphs centred in it. Gio puts the first line's ascent at
// the top and rounds each line's advance; lead is how far down the text
// goes to centre it, and height turns Gio's laid out height into lines ×
// the exact line height.
type lineBox struct {
	lead, one, adv int
	exact          float32
}

func (th *Theme) lineBox(gtx C, t Txt) lineBox {
	exact := float32(t.Size) * gtx.Metric.PxPerSp * t.LineH
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	one := th.Text(g, t, "x").Size.Y
	m.Stop()
	return lineBox{lead: int((exact-float32(one))/2 + 0.5), one: one, adv: int(exact + 0.5), exact: exact}
}

// height is the CSS height of text Gio laid out gioH tall.
func (b lineBox) height(gioH int) int {
	lines := 1
	if gioH > b.one {
		lines += (gioH - b.one + b.adv/2) / b.adv
	}
	return int(float32(lines)*b.exact + 0.5)
}

// cssText lays out t in CSS line boxes (lineBox).
func (th *Theme) cssText(gtx C, t Txt, s string) D {
	b := th.lineBox(gtx, t)
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints.Min.Y = 0
	d := th.Text(g, t, s)
	call := m.Stop()
	off := op.Offset(image.Pt(0, b.lead)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	h := b.height(d.Size.Y)
	return D{Size: image.Pt(d.Size.X, h), Baseline: h - b.lead - (d.Size.Y - d.Baseline)}
}

// prompt is the editor in its content box, w wide: at least two lines
// tall, scrolling past 208dp with its padding (the editor's -m-1 p-1), its
// lines in CSS line boxes (lineBox).
func (th *Theme) prompt(gtx C, s *Scene, w int) D {
	t := promptText(P.ContrastForeground)
	b := th.lineBox(gtx, t)
	minH, maxH := int(2*b.exact+0.5), dp(gtx, promptMax-8)
	lead := b.lead

	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, maxH-lead)}
	off := op.Offset(image.Pt(0, lead)).Push(gtx.Ops)
	var d D
	empty := false
	if s.Live {
		ed := th.Editor("composer")
		empty = ed.Len() == 0
		ed.Submit = true
		ui := &th.in.composer
		ui.menu, ui.rows = len(s.Suggest) > 0 || s.SuggestEmpty != "", len(s.Suggest) > 0
		ui.history = len(s.Thumbs) == 0 && len(s.Questions) == 0
		ui.thumbs = s.Thumbs
		d = th.editor(g, "composer", t, "")
	} else {
		empty = s.Composer == ""
		if !empty {
			var sel *widget.Selectable
			d, sel = th.selectableSel(g, "composer", s.Composer, t.Font, t.Size, t.Color, t.LineH)
			th.drawChips(g, sel, d.Size, image.Point{}, glass(), float32(t.Size), chipPictures(s.Thumbs))
		}
	}
	if empty && s.Placeholder != "" {
		pt := promptText(P.ContrastPlaceholder)
		pg := g
		pg.Constraints.Min = image.Point{}
		th.Text(pg, pt, s.Placeholder)
	}
	off.Pop()
	h := max(minH, min(maxH, b.height(d.Size.Y)))
	return D{Size: image.Pt(w, h)}
}

// selection is the prompt's selection. T3 sets none, so it's the
// platform's; Gio can't recolour selected text, so it's the Mac's light
// highlight, which keeps the text readable.
func selection() color.NRGBA {
	if dark {
		return rgb(0x3f638b)
	}
	return rgb(0xb3d7ff)
}

// composerKeys is the prompt's own keys, taken before the editor's
// (ComposerPromptEditorTiptap.tsx, composerCommandKey.ts): Mod+Enter sends
// the other way (queued, while a run works); with the @, / or $ menu
// open, the arrows move its highlight, Enter and Tab pick, Escape
// dismisses it; else Up on the first line, with the prompt empty or still
// a recalled one, goes back through the thread's prompts, and Down on the
// last line comes forward.
func (th *Theme) composerKeys(gtx C, ed *widget.Editor) {
	ui := &th.in.composer
	filters := []event.Filter{
		key.Filter{Focus: ed, Name: key.NameReturn, Required: key.ModShortcut},
		key.Filter{Focus: ed, Name: key.NameEnter, Required: key.ModShortcut},
	}
	switch {
	case ui.menu:
		// An empty menu still closes on Escape; Enter then sends.
		filters = append(filters, key.Filter{Focus: ed, Name: key.NameEscape})
		if ui.rows {
			for _, n := range []key.Name{key.NameUpArrow, key.NameDownArrow, key.NameReturn, key.NameEnter, key.NameTab} {
				filters = append(filters, key.Filter{Focus: ed, Name: n})
			}
		}
	case ui.history:
		browsing := ui.recalled != "" && ed.Text() == ui.recalled
		first, last := caretLines(ed)
		if first && (ed.Len() == 0 || browsing) {
			filters = append(filters, key.Filter{Focus: ed, Name: key.NameUpArrow})
		}
		if last && browsing {
			filters = append(filters, key.Filter{Focus: ed, Name: key.NameDownArrow})
		}
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		switch {
		case ke.Modifiers.Contain(key.ModShortcut):
			th.push(Action{ID: "composer", Kind: "submit", Text: ed.Text(), Alt: true})
		case ui.menu:
			th.push(Action{ID: "suggest", Kind: "key", Text: string(ke.Name)})
		case ke.Name == key.NameUpArrow:
			th.push(Action{ID: "composer", Kind: "history", Text: "back"})
		default:
			th.push(Action{ID: "composer", Kind: "history", Text: "forward"})
		}
	}
}

// caretLines reports whether the caret is on the prompt's first line and
// on its last, by its baseline against the first and last runes'.
func caretLines(ed *widget.Editor) (first, last bool) {
	n := ed.Len()
	if n == 0 {
		return true, true
	}
	y := int(ed.CaretCoords().Y + 0.5)
	base := func(regs []widget.Region) int { return regs[0].Bounds.Max.Y - regs[0].Baseline }
	head, tail := ed.Regions(0, 1, nil), ed.Regions(n-1, n, nil)
	if len(head) == 0 || len(tail) == 0 {
		return true, true
	}
	return y <= base(head), y >= base(tail)
}

// composerChords are the composer's shortcuts from anywhere
// (composerShortcuts.ts): Mod+Shift+M the model menu, Mod+Shift+E its
// options, Mod+Shift+A the permissions.
func (th *Theme) composerChords(gtx C, s *Scene) {
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: "M", Required: key.ModShortcut | key.ModShift},
			key.Filter{Name: "E", Required: key.ModShortcut | key.ModShift},
			key.Filter{Name: "A", Required: key.ModShortcut | key.ModShift},
		)
		if !ok {
			return
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		id := map[key.Name]string{"M": "model", "E": "traits", "A": "access"}[ke.Name]
		if id == "traits" && s.Effort == "" && !s.Fast && s.Window == "" {
			id = "model"
		}
		th.push(Action{ID: id, Kind: "click"})
	}
}

// composerFooter is the controls row, padded 0 8 8 (8 on top too while a
// question is open): attach, then the model trigger and permissions chip
// flush right against the primary actions.
func (th *Theme) composerFooter(gtx C, s *Scene, at *composerAnchors) D {
	top := float32(0)
	asking := len(s.Questions) > 0
	if asking {
		top = 8
	}
	w := gtx.Constraints.Max.X
	return pad(top, 8, 8, 8).Layout(gtx, func(gtx C) D {
		h := dp(gtx, 32)
		x := 0
		left := dp(gtx, top) // the anchors' frame starts at the footer's top-left, 8 in
		if !asking {
			// The attach button, then its 8dp margin.
			off := op.Offset(image.Pt(0, (h-dp(gtx, 28))/2)).Push(gtx.Ops)
			th.attachButton(gtx, s)
			off.Pop()
			x = dp(gtx, 28+8)
		}
		// The primary actions take what they need from the right.
		pm := op.Record(gtx.Ops)
		pg := gtx
		pg.Constraints = layout.Constraints{Max: image.Pt(gtx.Constraints.Max.X, h)}
		pd := th.primaryActions(pg, s)
		primary := pm.Stop()
		end := gtx.Constraints.Max.X
		po := op.Offset(image.Pt(end-pd.Size.X, (h-pd.Size.Y)/2)).Push(gtx.Ops)
		primary.Add(gtx.Ops)
		po.Pop()
		end -= pd.Size.X + dp(gtx, 8)

		// The controls, right to left: the chip, then the trigger.
		cy := (h - dp(gtx, 28)) / 2
		if s.RuntimeChip {
			cm := op.Record(gtx.Ops)
			cd := th.permissionsChip(gtx, s, false)
			chip := cm.Stop()
			end -= cd.Size.X
			co := op.Offset(image.Pt(end, cy)).Push(gtx.Ops)
			chip.Add(gtx.Ops)
			co.Pop()
			at.access = image.Rectangle{Min: image.Pt(end+dp(gtx, 8), cy+left), Max: image.Pt(end+cd.Size.X+dp(gtx, 8), cy+left+cd.Size.Y)}
			end -= dp(gtx, 2)
		}
		mm := op.Record(gtx.Ops)
		mg := gtx
		mg.Constraints = layout.Constraints{Max: image.Pt(max(0, end-x), dp(gtx, 28))}
		md, split := th.modelTrigger(mg, s)
		trigger := mm.Stop()
		mx := end - md.Size.X
		mo := op.Offset(image.Pt(mx, cy)).Push(gtx.Ops)
		trigger.Add(gtx.Ops)
		mo.Pop()
		o := image.Pt(mx+dp(gtx, 8), cy+left)
		at.model = image.Rectangle{Max: md.Size}.Add(o)
		at.traits = image.Rect(split, 0, md.Size.X, md.Size.Y).Add(o)
		if !s.RuntimeChip {
			at.access = at.model
		}
		return D{Size: image.Pt(w-dp(gtx, 16), h)}
	})
}

// attachButton is T3's ghost icon-sm button with the paperclip, a pixel
// low (translate-y-px): "Attach files".
func (th *Theme) attachButton(gtx C, s *Scene) {
	id := liveID(s, "attach")
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	sz := dp(gtx, 28)
	r := image.Rectangle{Max: image.Pt(sz, sz)}
	if hovered {
		fill(gtx, over(P.Accent, glass()), r, dp(gtx, 8))
	}
	off := op.Offset(image.Pt((sz-dp(gtx, 16))/2, (sz-dp(gtx, 16))/2+dp(gtx, 1))).Push(gtx.Ops)
	LucideW("paperclip", P.ContrastMutedForeground, 16)(gtx)
	off.Pop()
	if id != "" {
		th.hit(gtx, id, r.Max)
		th.tipAt(gtx, id, "Attach files", r.Max, true, 0)
	}
}

// composerControl is T3's ComposerControl at sm (h-7 px-2 gap-1.5,
// 14/20 medium at foreground 80%) or xs (h-6 px-1.75 gap-1.5, 12/16 at
// foreground 70%, muted in dark): hovered or open, it's filled with
// --accent and its text turns foreground.
type composerControl struct {
	xs        bool
	hot       bool // hovered, or its menu is open
	padStart  float32
	on        color.NRGBA // the surface under it
	icon      string
	iconLabel bool // the icon takes the label's colour, not muted
	label     string
}

func (c composerControl) height() float32 {
	if c.xs {
		return 24
	}
	return 28
}

func (c composerControl) text() color.NRGBA {
	switch {
	case c.hot && (!c.xs || !dark):
		return P.Foreground
	case c.xs && dark:
		return P.MutedForeground
	case c.xs:
		return alpha(P.Foreground, 0xb3)
	}
	return alpha(P.Foreground, 0xcc)
}

// layout draws the control's fill, icon and label and returns its size.
func (th *Theme) composerControlW(gtx C, c composerControl) D {
	size, line, weight, padX, iconSize := unit.Sp(14), float32(20), font.Medium, float32(8), float32(16)
	if c.xs {
		size, line, weight, padX, iconSize = 12, 16, font.Normal, 7, 12
	}
	ps := padX
	if c.padStart > 0 {
		ps = c.padStart
	}
	txt := over(c.text(), c.on)
	m := op.Record(gtx.Ops)
	cd := layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		rigid(func(gtx C) D {
			if c.icon == "" {
				return D{}
			}
			col := txt
			if !c.iconLabel && !c.xs {
				col = P.MutedForeground
			}
			return LucideW(c.icon, col, iconSize)(gtx)
		}),
		rigid(func(gtx C) D {
			if c.icon == "" || c.label == "" {
				return D{}
			}
			return D{Size: image.Pt(dp(gtx, 6), 0)}
		}),
		rigid(func(gtx C) D {
			if c.label == "" {
				return D{}
			}
			return th.Text(gtx, Txt{Size: size, LineH: line / float32(size), Font: UI(weight), Color: txt, MaxLines: 1}, c.label)
		}),
	)
	call := m.Stop()
	h := dp(gtx, c.height())
	px := max(1, dp(gtx, 1))
	w := cd.Size.X + dp(gtx, ps) + dp(gtx, padX) + 2*px
	r := image.Rect(0, 0, w, h)
	if c.hot {
		fill(gtx, over(P.Accent, c.on), r, dp(gtx, 8))
	}
	defer op.Offset(image.Pt(px+dp(gtx, ps), (h-cd.Size.Y)/2)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: r.Max}
}

// permissionsChip is the permissions control: the mode's icon and label,
// in the footer (sm, ps-1.5) or the new thread's strip (xs).
func (th *Theme) permissionsChip(gtx C, s *Scene, xs bool) D {
	mode := runtimeMode(s.Runtime)
	id := liveID(s, "access")
	hot := s.Access != nil
	if id != "" {
		c, _ := th.events(gtx, id)
		hot = hot || c.Hovered()
	}
	on := glass()
	if xs {
		on = stripGlass()
	}
	cc := composerControl{xs: xs, hot: hot, on: on, icon: mode.Icon, iconLabel: true, label: mode.Label}
	if !xs {
		cc.padStart = 6
	}
	d := th.composerControlW(gtx, cc)
	if id != "" {
		th.hit(gtx, id, d.Size)
		tip := "Permissions: " + mode.Label + ". " + mode.Desc
		if xs {
			tip = "Permissions · " + ShortcutShift("A")
		}
		th.tipAt(gtx, id, tip, d.Size, true, 0)
	}
	return d
}

// modelTrigger is the model and effort trigger (ProviderModelPicker.tsx):
// the provider's mark, the model's short name and the effort suffix. T3
// opens one picker that holds both; until tern's picker has the effort
// column, the suffix opens the options menu and the rest the model menu.
// It returns where the suffix starts.
func (th *Theme) modelTrigger(gtx C, s *Scene) (D, int) {
	mid, tid := liveID(s, "model"), ""
	if s.Effort != "" || s.Fast || s.Window != "" {
		tid = liveID(s, "traits")
	}
	hot := s.Picker != nil || s.Traits != nil
	for _, id := range []string{mid, tid} {
		if id != "" {
			c, _ := th.events(gtx, id)
			hot = hot || c.Hovered()
		}
	}
	on := glass()
	label := over(alpha(P.Foreground, 0xcc), on)
	muted := P.MutedForeground
	if hot {
		label, muted = P.Foreground, P.Foreground
	}
	name := s.Model
	if name == "" {
		name = "Choose model"
	}
	t := Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), MaxLines: 1}
	px, padX, gap := max(1, dp(gtx, 1)), dp(gtx, 8), dp(gtx, 6)

	// The suffix: the effort word, the bolt, the context window.
	sm := op.Record(gtx.Ops)
	sg := gtx
	sg.Constraints.Min = image.Point{}
	var parts []layout.FlexChild
	add := func(w W) {
		if len(parts) > 0 {
			parts = append(parts, rigid(Spacer(6, 0)))
		}
		parts = append(parts, rigid(w))
	}
	if s.Effort != "" {
		add(func(gtx C) D {
			gtx.Constraints.Max.X = dp(gtx, 96)
			wt := t
			wt.Color = muted
			return th.Text(gtx, wt, s.Effort)
		})
	}
	if s.Fast {
		add(LucideFill("zap", muted, 12))
	}
	if s.Window != "" {
		add(func(gtx C) D {
			gtx.Constraints.Max.X = dp(gtx, 96)
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: muted, MaxLines: 1}, s.Window)
		})
	}
	suffix := layout.Flex{Alignment: layout.Middle}.Layout(sg, parts...)
	suffixCall := sm.Stop()

	// The label truncates to what's left, at most 224.
	icon := 0
	if s.Provider != "" {
		icon = dp(gtx, 16-4) + gap // its -2dp side margins
	}
	tail := 0
	if suffix.Size.X > 0 {
		tail = gap - dp(gtx, 2) + suffix.Size.X
	}
	lm := op.Record(gtx.Ops)
	lg := gtx
	lg.Constraints = layout.Constraints{Max: image.Pt(max(0, min(dp(gtx, 224), gtx.Constraints.Max.X-2*px-2*padX-icon-tail)), gtx.Constraints.Max.Y)}
	t.Color = label
	ld := th.Text(lg, t, name)
	labelCall := lm.Stop()

	h := dp(gtx, 28)
	w := max(dp(gtx, 52), 2*px+2*padX+icon+ld.Size.X+tail)
	r := image.Rect(0, 0, w, h)
	if hot {
		fill(gtx, over(P.Accent, on), r, dp(gtx, 8))
	}
	x := px + padX
	if s.Provider != "" {
		off := op.Offset(image.Pt(x-dp(gtx, 2), (h-dp(gtx, 16))/2)).Push(gtx.Ops)
		ProviderLogo(s.Provider, 16)(gtx)
		off.Pop()
		x += icon
	}
	off := op.Offset(image.Pt(x, (h-ld.Size.Y)/2)).Push(gtx.Ops)
	labelCall.Add(gtx.Ops)
	off.Pop()
	x += ld.Size.X
	split := w
	if suffix.Size.X > 0 {
		split = x
		off := op.Offset(image.Pt(x+gap-dp(gtx, 2), (h-suffix.Size.Y)/2)).Push(gtx.Ops)
		suffixCall.Add(gtx.Ops)
		off.Pop()
	}
	if mid != "" {
		th.hit(gtx, mid, r.Max)
	}
	if tid != "" {
		off := op.Offset(image.Pt(split, 0)).Push(gtx.Ops)
		th.hit(gtx, tid, image.Pt(w-split, h))
		off.Pop()
	}
	return D{Size: r.Max}, split
}

// primaryActions is the send button, or Stop while a run can be stopped
// and nothing's typed; with a question open, Previous and the Submit pill
// (ComposerPrimaryActions.tsx).
func (th *Theme) primaryActions(gtx C, s *Scene) D {
	running := s.State == ThreadRunning
	if len(s.Questions) > 0 {
		return th.answerActions(gtx, s, running)
	}
	if running && !th.hasContent(s) {
		return th.stopButton(gtx, s)
	}
	return th.sendButton(gtx, s, running)
}

// hasContent reports whether the composer has something to send.
func (th *Theme) hasContent(s *Scene) bool {
	if len(s.Thumbs) > 0 {
		return true
	}
	if s.Live {
		return TrimText(StripChips(th.EditorText("composer"))) != ""
	}
	return s.Composer != ""
}

// sendButton is the 32dp round send button: --message-action with the
// arrow, the steer arrow while a run works, a spinner while it sends.
func (th *Theme) sendButton(gtx C, s *Scene, running bool) D {
	id := liveID(s, "send")
	enabled := th.hasContent(s) && !s.Sending
	hovered := false
	if id != "" && enabled {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	sz := dp(gtx, 32)
	r := image.Rect(0, 0, sz, sz)
	on := glass()
	label := "Submit message"
	switch {
	case s.Sending:
		label = "Submitting message"
	case running:
		label = "Click to steer, " + Shortcut("Enter") + " to queue"
	}
	func() {
		if hovered {
			c := f32.Pt(float32(sz)/2, float32(sz)/2)
			defer op.Affine(f32.Affine2D{}.Scale(c, f32.Pt(1.05, 1.05))).Push(gtx.Ops).Pop()
		}
		fg := P.MessageActionForeground
		if !enabled && !s.Sending {
			fill(gtx, over(alpha(P.Foreground, 0x1f), on), r, sz/2)
			fg = P.MutedForeground
		} else {
			bg := P.MessageAction
			if hovered {
				bg = P.MessageActionHover
			}
			// The control highlight is a 1dp line under it, not inset.
			fill(gtx, over(alpha(rgb(0xffffff), 0x29), on), r.Add(image.Pt(0, max(1, dp(gtx, 1)))), sz/2)
			shadow(gtx, r, sz/2, []boxShadow{{0, 1, 2, 0, alpha(P.MessageAction, 0x3d)}})
			fill(gtx, bg, r, sz/2)
		}
		switch {
		case s.Sending:
			off := op.Offset(image.Pt((sz-dp(gtx, 14))/2, (sz-dp(gtx, 14))/2)).Push(gtx.Ops)
			T3Spinner(fg, 14, s.Phase)(gtx)
			off.Pop()
		case running:
			off := op.Offset(image.Pt((sz-dp(gtx, 16))/2, (sz-dp(gtx, 16))/2)).Push(gtx.Ops)
			LucideW("corner-up-right", fg, 16)(gtx)
			off.Pop()
		default:
			sendArrow(gtx, fg, image.Pt((sz-dp(gtx, 14))/2, (sz-dp(gtx, 14))/2))
		}
	}()
	if id != "" {
		if enabled {
			th.hit(gtx, id, r.Max)
		} else {
			// The tooltip sits on a wrapper, so a disabled button explains itself.
			th.events(gtx, id)
			th.hover(gtx, id, r.Max)
		}
		th.tipAt(gtx, id, label, r.Max, true, 0)
	}
	return D{Size: r.Max}
}

// hover is hit for a target that only has a tooltip: no hand cursor.
func (th *Theme) hover(gtx C, id string, size image.Point) {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	th.in.clicks[id].Add(gtx.Ops)
}

// sendArrow is the idle send icon: T3's own 14dp arrow, stroked 1.8 with
// round ends (M7 11.5V2.5M7 2.5L3 6.5M7 2.5L11 6.5).
func sendArrow(gtx C, col color.NRGBA, at image.Point) {
	k := float32(dp(gtx, 14)) / 14
	o := f32.Pt(float32(at.X), float32(at.Y))
	pt := func(x, y float32) f32.Point { return o.Add(f32.Pt(x*k, y*k)) }
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(pt(7, 11.5))
	p.LineTo(pt(7, 2.5))
	p.MoveTo(pt(3, 6.5))
	p.LineTo(pt(7, 2.5))
	p.LineTo(pt(11, 6.5))
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.8 * k}.Op())
	for _, q := range []f32.Point{pt(7, 11.5), pt(3, 6.5), pt(11, 6.5)} {
		r := 0.9 * k
		paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: image.Pt(int(q.X-r+0.5), int(q.Y-r+0.5)), Max: image.Pt(int(q.X+r+0.5), int(q.Y+r+0.5))}.Op(gtx.Ops))
	}
}

// stopButton is Stop: a 32dp round button in --destructive at 90% (full on
// hover) with a 12dp white square: "Interrupt".
func (th *Theme) stopButton(gtx C, s *Scene) D {
	id := liveID(s, "stop")
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	sz := dp(gtx, 32)
	r := image.Rect(0, 0, sz, sz)
	on := glass()
	bg := over(alpha(P.Destructive, 0xe6), on)
	if hovered {
		bg = P.Destructive
	}
	fill(gtx, over(alpha(rgb(0xffffff), 0x29), on), r.Add(image.Pt(0, max(1, dp(gtx, 1)))), sz/2)
	shadow(gtx, r, sz/2, []boxShadow{{0, 1, 2, 0, alpha(P.Destructive, 0x3d)}})
	fill(gtx, bg, r, sz/2)
	k := float32(dp(gtx, 12)) / 12
	o := image.Pt((sz-dp(gtx, 12))/2, (sz-dp(gtx, 12))/2)
	sq := image.Rect(o.X+int(2*k+0.5), o.Y+int(2*k+0.5), o.X+int(10*k+0.5), o.Y+int(10*k+0.5))
	fill(gtx, rgb(0xffffff), sq, int(1.5*k+0.5))
	if id != "" {
		th.hit(gtx, id, r.Max)
		th.tipAt(gtx, id, "Interrupt", r.Max, true, 0)
	}
	return D{Size: r.Max}
}

// answerActions is the footer while a question is open: Previous past the
// first, the Submit pill, and Stop while the run can be stopped.
func (th *Theme) answerActions(gtx C, s *Scene, running bool) D {
	var kids []layout.FlexChild
	if s.Question > 0 {
		kids = append(kids, rigid(th.T3Button(liveID(s, "q.prev"), Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Previous", Disabled: s.Answering, On: glass()})), rigid(Spacer(8, 0)))
	}
	kids = append(kids, rigid(func(gtx C) D { return th.submitPill(gtx, s) }))
	if running {
		kids = append(kids, rigid(Spacer(8, 0)), rigid(func(gtx C) D { return th.stopButton(gtx, s) }))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
}

// submitPill is the question's primary action: a 28dp --message-action
// pill, "Next question", "Submit answer" or "Submit answers", disabled
// until the question (or, at the last, every one) has its answer.
func (th *Theme) submitPill(gtx C, s *Scene) D {
	last := s.Question >= len(s.Questions)-1
	label := "Next question"
	switch {
	case s.Answering:
		label = "Submitting..."
	case last && s.Question > 0:
		label = "Submit answers"
	case last:
		label = "Submit answer"
	}
	ready := th.answered(s, s.Question)
	if last {
		for i := range s.Questions {
			ready = ready && th.answered(s, i)
		}
	}
	enabled := ready && !s.Answering
	id := liveID(s, "answer")
	hovered := false
	if id != "" && enabled {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	m := op.Record(gtx.Ops)
	td := th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.MessageActionForeground, MaxLines: 1}, label)
	call := m.Stop()
	h := dp(gtx, 28)
	r := image.Rect(0, 0, td.Size.X+2*dp(gtx, 16), h)
	if !enabled {
		defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
	} else {
		shadow(gtx, r, h/2, []boxShadow{{0, 1, 2, 0, alpha(P.MessageAction, 0x3d)}})
	}
	bg := P.MessageAction
	if hovered {
		bg = P.MessageActionHover
	}
	fill(gtx, bg, r, h/2)
	off := op.Offset(image.Pt(dp(gtx, 16), (h-td.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	if id != "" && enabled {
		th.hit(gtx, id, r.Max)
	}
	return D{Size: r.Max}
}

// answered reports whether question i has an answer: an option picked,
// or for the question on show, an answer typed into the composer.
func (th *Theme) answered(s *Scene, i int) bool {
	q := s.Questions[i]
	for _, on := range q.Chosen {
		if on {
			return true
		}
	}
	if i != s.Question || q.NoCustom {
		return false
	}
	if s.Live {
		return TrimText(th.EditorText("composer")) != ""
	}
	return s.Composer != ""
}

// composerTile is a picture of the composer's as T3's 64dp tile: rounded
// 8, a border at 80%, and its remove button in the corner
// (ComposerAttachmentStrip.tsx).
func (th *Theme) composerTile(t Thumb, i int, live bool) W {
	return func(gtx C) D {
		sz := dp(gtx, tileSize)
		r := image.Rect(0, 0, sz, sz)
		rad := dp(gtx, 8)
		fill(gtx, P.Background, r, rad)
		if !drawThumb(gtx, t, sz, rad) {
			off := op.Offset(image.Pt((sz-dp(gtx, 16))/2, (sz-dp(gtx, 16))/2)).Push(gtx.Ops)
			LucideW("image", P.MutedForeground, 16)(gtx)
			off.Pop()
		}
		strokeRect(gtx, over(alpha(P.Border, uint8(int(P.Border.A)*8/10)), glass()), r, rad, float32(max(1, dp(gtx, 1))))
		if !live {
			return D{Size: r.Max}
		}
		// media-close: 24dp, black at 65% (80% on hover), a white x.
		id := "image.remove:" + strconv.Itoa(i)
		c, _ := th.events(gtx, id)
		b := dp(gtx, 24)
		defer op.Offset(image.Pt(sz-b-dp(gtx, 4), dp(gtx, 4))).Push(gtx.Ops).Pop()
		br := image.Rect(0, 0, b, b)
		bg := alpha(rgb(0), 0xa6)
		if c.Hovered() {
			bg = alpha(rgb(0), 0xcc)
		}
		fill(gtx, bg, br, dp(gtx, 8))
		strokeRect(gtx, alpha(rgb(0xffffff), 0x33), br, dp(gtx, 8), float32(max(1, dp(gtx, 1))))
		off := op.Offset(image.Pt((b-dp(gtx, 14))/2, (b-dp(gtx, 14))/2)).Push(gtx.Ops)
		LucideW("x", rgb(0xffffff), 14)(gtx)
		off.Pop()
		th.hit(gtx, id, br.Max)
		return D{Size: r.Max}
	}
}

// LucideFill is a lucide icon filled in col as well as stroked, as T3
// draws its bolt and play icons (fill-current).
func LucideFill(name string, col color.NRGBA, sizeDp float32) W {
	return func(gtx C) D {
		k := float32(dp(gtx, sizeDp)) / 24
		var p clip.Path
		p.Begin(gtx.Ops)
		for _, sg := range parseSVGPath(lucide[name]) {
			pt := func(i int) f32.Point { return sg.p[i].Mul(k) }
			switch sg.op {
			case 'M':
				p.MoveTo(pt(0))
			case 'L':
				p.LineTo(pt(0))
			case 'Q':
				p.QuadTo(pt(0), pt(1))
			case 'C':
				p.CubeTo(pt(0), pt(1), pt(2))
			case 'Z':
				p.Close()
			}
		}
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return LucideW(name, col, sizeDp)(gtx)
	}
}
