package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"

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
// 80% glass opacity over what it blurs, #ffffff / #0e0e0e. Gio has no
// backdrop blur, so it's solid; the blur only shows where the timeline
// scrolls under the card.
func glass() color.NRGBA { return over(alpha(glassSurface(), 0xcc), composerBackdrop()) }

// glassSurface is --chat-composer-glass-surface itself, opaque.
func glassSurface() color.NRGBA {
	if dark {
		return P.SurfaceRaised
	}
	return P.Card
}

// composerBackdrop is what the composer's glass blurs as Chrome samples it
// under the lane (§3.2): white in light, which the card's #ffffff and the
// strip's #fdfdfd come from, and the page's black in dark.
func composerBackdrop() color.NRGBA {
	if dark {
		return P.Background
	}
	return rgb(0xffffff)
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
	model, access image.Rectangle
}

// composerArea is the thread's composer, bottom of the lane: the surfaces
// attached above the card, then the card.
func (th *Theme) composerArea(gtx C, s *Scene, compact bool) D {
	return th.phoneMargins(gtx, compact, func(gtx C) D { return th.composerStack(gtx, s, false) })
}

// phoneMargins lays w out the lane wide, or on a phone (below 640) in
// T3's 12dp of lane padding with the 16dp spacer under it (03-shell.md
// port note 9).
func (th *Theme) phoneMargins(gtx C, compact bool, w W) D {
	if !compact {
		return w(gtx)
	}
	full := gtx.Constraints.Max.X
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(full-2*dp(gtx, 12), 0), Max: image.Pt(full-2*dp(gtx, 12), gtx.Constraints.Max.Y-dp(gtx, 16))}
	defer op.Offset(image.Pt(dp(gtx, 12), 0)).Push(gtx.Ops).Pop()
	d := w(g)
	return D{Size: image.Pt(full, d.Size.Y+dp(gtx, 16))}
}

// composerStack is the attached surfaces, the card over their tucked
// bottoms and, for a new thread, the context strip under it.
func (th *Theme) composerStack(gtx C, s *Scene, strip bool) D {
	w := gtx.Constraints.Max.X
	if s.Live {
		th.composerChords(gtx, s)
	}
	ui := &th.in.composer
	ui.menuWas, ui.menuShown = ui.menuShown, false
	ui.stashWas, ui.stashShown = ui.stashShown, false
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
	switch {
	case len(s.Suggest) > 0 || s.SuggestEmpty != "":
		th.commandMenu(gtx, s, w)
	case s.StashOpen:
		th.stashDrawer(gtx, s, w)
	}
	off.Pop()
	return D{Size: image.Pt(w, top+h)}
}

// composerMenus opens the menus off their triggers, above them and 4dp
// up: the model picker flush with the trigger's end (side top, align
// end), the permissions menu with the footer chip's start or the strip
// chip's end.
func (th *Theme) composerMenus(gtx C, s *Scene, at composerAnchors) {
	switch {
	case s.Picker != nil:
		th.popAbove(gtx, at.model, true, func(gtx C) D { return th.modelPicker(gtx, s, s.Picker) })
	case s.Access != nil:
		th.popAbove(gtx, at.access, !s.RuntimeChip, func(gtx C) D { return th.permissionsMenu(gtx, s) })
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
	foot := th.composerFooter(inner, s, at, w)
	fo.Pop()
	off.Pop()
	call := m.Stop()
	// The anchors were laid out in the footer's frame.
	shift := image.Pt(px, px+body.Size.Y)
	at.model, at.access = at.model.Add(shift), at.access.Add(shift)

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
// lines in CSS line boxes (lineBox). Live, it's read-only while a
// question takes only its options or its answers are on their way, and it
// tells core when it gains and loses the focus (the typing guard).
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
		ed.Submit = true
		asking := len(s.Questions) > 0
		ed.ReadOnly = asking && (s.Answering || s.Questions[min(s.Question, len(s.Questions)-1)].NoCustom)
		ui := &th.in.composer
		ui.menu, ui.rows = len(s.Suggest) > 0 || s.SuggestEmpty != "", len(s.Suggest) > 0
		ui.stash = s.StashOpen
		ui.history = len(s.Thumbs) == 0 && !asking
		ui.suggestion = s.Suggestion != "" && s.Placeholder == s.Suggestion
		ui.queue = len(s.Queue)
		ui.thumbs = s.Thumbs
		if f := gtx.Focused(ed); f != ui.focused {
			ui.focused = f
			th.push(Action{ID: "composer", Kind: map[bool]string{true: "focus", false: "blur"}[f]})
		}
		d = th.editor(g, "composer", t, "")
		empty = ed.Len() == 0
	} else {
		empty = s.Composer == ""
		if !empty {
			var sel *widget.Selectable
			d, sel = th.selectableSel(g, "composer", s.Composer, t.Font, t.Size, t.Color, t.LineH)
			th.drawChips(g, sel, d.Size, glass(), float32(t.Size), chipPictures(s.Thumbs), false)
		}
	}
	if empty && s.Placeholder != "" {
		pg := g
		pg.Constraints.Min = image.Point{}
		pd := th.Text(pg, promptText(P.ContrastPlaceholder), s.Placeholder)
		if s.Suggestion != "" && s.Placeholder == s.Suggestion {
			// The suggestion's key, 8dp after it and a pixel up (align-[1px]).
			kd := record(gtx, th.T3KbdOutline("⇥ Tab", glass()))
			ko := op.Offset(image.Pt(pd.Size.X+dp(gtx, 8), (int(b.exact+0.5)-kd.Size.Y)/2-b.lead-dp(gtx, 1))).Push(gtx.Ops)
			kd.call.Add(gtx.Ops)
			ko.Pop()
		}
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
// the other way (queued, while a run works; in the background, from a new
// thread), Mod+Alt+Enter in the background, Alt+Enter is a newline and
// never a send, Alt+Up edits the last queued message; with the @, / or $
// menu open, the arrows move its highlight, Enter and Tab pick, Escape
// dismisses it; with an empty prompt showing a suggestion, Tab takes it
// and Escape dismisses it; else Up on the first line, with the prompt
// empty or still a recalled one, goes back through the thread's prompts,
// and Down on the last line comes forward.
func (th *Theme) composerKeys(gtx C, ed *widget.Editor) {
	ui := &th.in.composer
	filters := []event.Filter{
		key.Filter{Focus: ed, Name: key.NameReturn, Required: key.ModShortcut, Optional: key.ModAlt},
		key.Filter{Focus: ed, Name: key.NameEnter, Required: key.ModShortcut, Optional: key.ModAlt},
		key.Filter{Focus: ed, Name: key.NameReturn, Required: key.ModAlt},
		key.Filter{Focus: ed, Name: key.NameEnter, Required: key.ModAlt},
	}
	if ui.queue > 0 {
		filters = append(filters, key.Filter{Focus: ed, Name: key.NameUpArrow, Required: key.ModAlt})
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
	case ui.stash:
		for _, n := range []key.Name{key.NameUpArrow, key.NameDownArrow, key.NameReturn, key.NameEnter, key.NameEscape} {
			filters = append(filters, key.Filter{Focus: ed, Name: n})
		}
		filters = append(filters, key.Filter{Focus: ed, Name: key.NameDeleteBackward, Required: key.ModShortcut})
	case ui.suggestion && ed.Len() == 0:
		filters = append(filters, key.Filter{Focus: ed, Name: key.NameTab}, key.Filter{Focus: ed, Name: key.NameEscape})
	}
	if ui.history && !ui.menu && !ui.stash {
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
		enter := ke.Name == key.NameReturn || ke.Name == key.NameEnter
		switch {
		case enter && ke.Modifiers.Contain(key.ModShortcut):
			th.push(Action{ID: "composer", Kind: "submit", Text: ed.Text(), Alt: true, Background: ke.Modifiers.Contain(key.ModAlt)})
		case enter && ke.Modifiers.Contain(key.ModAlt):
			if !ed.ReadOnly {
				th.InsertText("composer", "\n")
			}
		case ke.Name == key.NameUpArrow && ke.Modifiers.Contain(key.ModAlt):
			th.push(Action{ID: "queue:-1:edit", Kind: "click"})
		case ui.stash && !ui.menu && ke.Name == key.NameDeleteBackward:
			th.push(Action{ID: "stash", Kind: "key", Text: string(ke.Name)})
		case ui.menu:
			th.push(Action{ID: "suggest", Kind: "key", Text: string(ke.Name)})
		case ui.stash:
			th.push(Action{ID: "stash", Kind: "key", Text: string(ke.Name)})
		case ke.Name == key.NameTab:
			th.push(Action{ID: "composer", Kind: "suggestion", Text: "take"})
		case ke.Name == key.NameEscape:
			th.push(Action{ID: "composer", Kind: "suggestion", Text: "dismiss"})
		case ke.Name == key.NameUpArrow:
			th.push(Action{ID: "composer", Kind: "history", Text: "back"})
		default:
			th.push(Action{ID: "composer", Kind: "history", Text: "forward"})
		}
	}
}

// caretLines reports whether the caret is on the prompt's first visual
// line and on its last. The caret's coordinates and the regions are both
// in the view's frame, scroll taken off, and regions cover only the lines
// in view, so a last line out of view isn't the caret's.
func caretLines(ed *widget.Editor) (first, last bool) {
	n := ed.Len()
	if n == 0 {
		return true, true
	}
	line, _ := ed.CaretPos()
	tail := ed.Regions(n-1, n, nil)
	if len(tail) == 0 {
		return line == 0, false
	}
	t := tail[len(tail)-1]
	return line == 0, int(ed.CaretCoords().Y+0.5) >= t.Bounds.Max.Y-t.Baseline
}

// composerChords are the composer's shortcuts from anywhere
// (composerShortcuts.ts, keybindings.ts): Mod+Shift+M the model picker,
// Mod+Shift+E the same picker, for its effort column, Mod+Shift+A the
// permissions, Mod+S the stash, Mod+Shift+Enter sends the first queued
// message now, Mod+Alt+N starts a thread without a project; and a letter
// typed while nothing has the focus goes into the prompt.
func (th *Theme) composerChords(gtx C, s *Scene) {
	filters := []event.Filter{
		key.Filter{Name: "M", Required: key.ModShortcut | key.ModShift},
		key.Filter{Name: "E", Required: key.ModShortcut | key.ModShift},
		key.Filter{Name: "A", Required: key.ModShortcut | key.ModShift},
		key.Filter{Name: "S", Required: key.ModShortcut},
		key.Filter{Name: key.NameReturn, Required: key.ModShortcut | key.ModShift},
		key.Filter{Name: key.NameEnter, Required: key.ModShortcut | key.ModShift},
		key.Filter{Name: "N", Required: key.ModShortcut | key.ModAlt},
		key.Filter{Name: key.NameSpace},
		// ⌘ or Ctrl held swaps the send button's steer for queue.
		key.Filter{Name: key.NameCtrl, Optional: key.ModCtrl | key.ModShift | key.ModAlt | key.ModCommand},
		key.Filter{Name: key.NameCommand, Optional: key.ModCtrl | key.ModShift | key.ModAlt | key.ModCommand},
	}
	for c := 'A'; c <= 'Z'; c++ {
		filters = append(filters, key.Filter{Name: key.Name(string(c)), Optional: key.ModShift})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return
		}
		ke, ok := ev.(key.Event)
		if !ok {
			continue
		}
		if ke.Name == key.NameCtrl || ke.Name == key.NameCommand {
			th.in.composer.held = ke.State == key.Press
			continue
		}
		if ke.State != key.Press {
			continue
		}
		short := ke.Modifiers.Contain(key.ModShortcut)
		switch {
		case !short && !ke.Modifiers.Contain(key.ModAlt):
			// Type to focus (composerFocusRedirect.ts), for the keys whose
			// text Gio's names give whatever the keyboard.
			if th.typing(gtx) || s.Picker != nil || s.Access != nil || s.Projects != nil || th.in.preview != nil {
				continue
			}
			r := " "
			if ke.Name != key.NameSpace {
				r = string(ke.Name)
				if !ke.Modifiers.Contain(key.ModShift) {
					r = strings.ToLower(r)
				}
			}
			ed := th.Editor("composer")
			n := ed.Len()
			ed.SetCaret(n, n)
			th.InsertText("composer", r)
			th.Focus("composer")
		case ke.Name == "N":
			if s.CanScratch {
				th.push(Action{ID: "project.scratch", Kind: "click"})
			}
		case ke.Name == key.NameReturn || ke.Name == key.NameEnter:
			if len(s.Queue) > 0 {
				th.push(Action{ID: "queue:0:steer", Kind: "click"})
			}
		default:
			id := map[key.Name]string{"M": "model", "E": "model", "A": "access", "S": "stash.save"}[ke.Name]
			th.push(Action{ID: id, Kind: "click"})
		}
	}
}

// composerFooter is the controls row, padded 0 8 8 (8 on top too while a
// question is open, when it's the 28dp answer actions): attach, then the
// model trigger and permissions chip flush right against the primary
// actions. formW is the composer's width, for the compact answer actions.
func (th *Theme) composerFooter(gtx C, s *Scene, at *composerAnchors, formW int) D {
	top, row := float32(0), float32(32)
	asking := len(s.Questions) > 0
	if asking {
		top, row = 8, 28
	}
	w := gtx.Constraints.Max.X
	return pad(top, 8, 8, 8).Layout(gtx, func(gtx C) D {
		// Everything here is placed by its own size.
		gtx.Constraints.Min = image.Point{}
		h := dp(gtx, row)
		x := 0
		left := dp(gtx, top) // the anchors' frame starts at the footer's top-left, 8 in
		if !asking || !s.Questions[min(s.Question, len(s.Questions)-1)].NoCustom {
			// The attach button, then its 8dp margin; a question takes
			// pictures unless only its options answer it.
			off := op.Offset(image.Pt(0, (h-dp(gtx, 28))/2)).Push(gtx.Ops)
			th.attachButton(gtx, s)
			off.Pop()
			x = dp(gtx, 28+8)
		}
		// The primary actions take what they need from the right.
		pg := gtx
		pg.Constraints = layout.Constraints{Max: image.Pt(gtx.Constraints.Max.X, h)}
		compact := asking && formW < dp(gtx, 780) // composerFooterLayout.ts
		pd := record(pg, func(gtx C) D { return th.primaryActions(gtx, s, compact) })
		end := gtx.Constraints.Max.X
		po := op.Offset(image.Pt(end-pd.Size.X, (h-pd.Size.Y)/2)).Push(gtx.Ops)
		pd.call.Add(gtx.Ops)
		po.Pop()
		end -= pd.Size.X + dp(gtx, 8)

		// The controls, right to left: the chip, then the trigger.
		cy := (h - dp(gtx, 28)) / 2
		if s.RuntimeChip {
			cd := record(gtx, func(gtx C) D { return th.permissionsChip(gtx, s, false) })
			end -= cd.Size.X
			co := op.Offset(image.Pt(end, cy)).Push(gtx.Ops)
			cd.call.Add(gtx.Ops)
			co.Pop()
			at.access = image.Rectangle{Min: image.Pt(end+dp(gtx, 8), cy+left), Max: image.Pt(end+cd.Size.X+dp(gtx, 8), cy+left+cd.Size.Y)}
			end -= dp(gtx, 2)
		}
		mg := gtx
		mg.Constraints = layout.Constraints{Max: image.Pt(max(0, end-x), dp(gtx, 28))}
		md := record(mg, func(gtx C) D { return th.modelTrigger(gtx, s) })
		mx := end - md.Size.X
		mo := op.Offset(image.Pt(mx, cy)).Push(gtx.Ops)
		md.call.Add(gtx.Ops)
		mo.Pop()
		at.model = image.Rectangle{Max: md.Size}.Add(image.Pt(mx+dp(gtx, 8), cy+left))
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
// --accent and its text turns foreground; disabled, it's at 64%.
type composerControl struct {
	xs        bool
	hot       bool // hovered, or its menu is open
	off       bool // disabled
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

// composerControlW draws the control's fill, icon and label and returns
// its size.
func (th *Theme) composerControlW(gtx C, c composerControl) D {
	size, line, weight, padX, iconSize := unit.Sp(14), float32(20), font.Medium, float32(8), float32(16)
	if c.xs {
		size, line, weight, padX, iconSize = 12, 16, font.Normal, 7, 12
	}
	ps := padX
	if c.padStart > 0 {
		ps = c.padStart
	}
	gtx.Constraints.Min = image.Point{}
	// opacity-64 as Chrome composites it, in sRGB over the surface; Gio's
	// layer opacity would blend in linear light and come out paler.
	fade := func(col color.NRGBA) color.NRGBA {
		if c.off {
			return over(alpha(col, 0xa3), c.on)
		}
		return col
	}
	txt := fade(over(c.text(), c.on))
	cd := record(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			rigid(func(gtx C) D {
				if c.icon == "" {
					return D{}
				}
				col := txt
				if !c.iconLabel && !c.xs {
					col = fade(P.MutedForeground)
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
	})
	h := dp(gtx, c.height())
	px := max(1, dp(gtx, 1))
	w := cd.Size.X + dp(gtx, ps) + dp(gtx, padX) + 2*px
	r := image.Rect(0, 0, w, h)
	if c.hot {
		fill(gtx, over(P.Accent, c.on), r, dp(gtx, 8))
	}
	defer op.Offset(image.Pt(px+dp(gtx, ps), (h-cd.Size.Y)/2)).Push(gtx.Ops).Pop()
	cd.call.Add(gtx.Ops)
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

// permissionsMenu is the permissions chip's menu, T3's
// ComposerPermissionsSection in the menu frame (§3.8): 288 wide, padded 6
// 8, 6 apart, "Permissions", the modes as one segmented row of icons and
// under it the mode on show, the pointer's or the thread's, with its
// description, two lines kept for it.
func (th *Theme) permissionsMenu(gtx C, s *Scene) D {
	p := s.Access
	return popupFrame(gtx, func(gtx C) D {
		return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
			return pad(6, 8, 6, 8).Layout(gtx, func(gtx C) D {
				w := dp(gtx, 288-16)
				gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
				muted := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}
				head := muted
				head.Font = UI(font.Medium)
				shown := p.Sel
				if pm := th.in.composer.previewMode - 1; pm >= 0 && pm < len(p.Modes) {
					shown = pm
				}
				th.in.composer.previewMode = 0
				var mode RuntimeMode
				if shown >= 0 && shown < len(p.Modes) {
					mode = p.Modes[shown]
				}
				label := Txt{Size: 14, LineH: 20.0 / 14, Color: P.Foreground}
				if shown != p.Sel {
					label.Color = P.MutedForeground
				}
				return Col(gtx,
					rigid(func(gtx C) D { return th.Text(gtx, head, "Permissions") }),
					rigid(Spacer(0, 6)),
					rigid(func(gtx C) D { return th.permissionSegments(gtx, s, w) }),
					rigid(Spacer(0, 6)),
					rigid(func(gtx C) D { return th.Text(gtx, label, mode.Label) }),
					rigid(Spacer(0, 2)),
					rigid(func(gtx C) D {
						d := th.Text(gtx, muted, mode.Desc)
						return D{Size: image.Pt(d.Size.X, max(d.Size.Y, dp(gtx, 32)))}
					}),
				)
			})
		})
	})
}

// permissionSegments is the modes' row: 28 tall, rounded 8, padded 2 on
// --input at 40%, an equal 24dp segment each with its icon at 14; the
// thread's on --input at 72% in the foreground, the pointer's on --accent.
func (th *Theme) permissionSegments(gtx C, s *Scene, w int) D {
	p := s.Access
	h, padIn := dp(gtx, 28), dp(gtx, 2)
	group := over(alpha(P.Input, uint8(int(P.Input.A)*40/100)), P.Popover)
	fill(gtx, group, image.Rect(0, 0, w, h), dp(gtx, 8))
	n := max(1, len(p.Modes))
	segW := (w - 2*padIn) / n
	for i, m := range p.Modes {
		x := padIn + i*segW
		sw := segW
		if i == n-1 {
			sw = w - padIn - x
		}
		func() {
			defer op.Offset(image.Pt(x, padIn)).Push(gtx.Ops).Pop()
			id := liveID(s, "access:0:"+strconv.Itoa(i))
			hovered := false
			if id != "" {
				c, _ := th.events(gtx, id)
				hovered = c.Hovered()
			}
			if hovered {
				th.in.composer.previewMode = i + 1
			}
			r := image.Rect(0, 0, sw, h-2*padIn)
			rad := dp(gtx, 6)
			nw, ne := 0, 0
			if i == 0 {
				nw = rad
			}
			if i == n-1 {
				ne = rad
			}
			col := P.MutedForeground
			var bg color.NRGBA
			switch {
			case i == p.Sel:
				bg, col = over(alpha(P.Input, uint8(int(P.Input.A)*72/100)), group), P.Foreground
			case hovered:
				bg, col = over(P.Accent, group), P.Foreground
			}
			if bg.A != 0 {
				paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: r, NW: nw, SW: nw, NE: ne, SE: ne}.Op(gtx.Ops))
			}
			io := op.Offset(image.Pt((sw-dp(gtx, 14))/2, (r.Dy()-dp(gtx, 14))/2)).Push(gtx.Ops)
			LucideW(m.Icon, col, 14)(gtx)
			io.Pop()
			if id != "" {
				th.hit(gtx, id, r.Max)
			}
		}()
	}
	return D{Size: image.Pt(w, h)}
}

// modelTrigger is the model and effort trigger (ProviderModelPicker.tsx):
// one button, the provider's mark, the model's short name and the effort
// suffix, which opens T3's one picker for both, the model list beside the
// effort column. It has no tooltip; the composer names it. With no
// provider to send to it's T3's unavailable control instead, off: tern
// has no provider settings for it to open.
func (th *Theme) modelTrigger(gtx C, s *Scene) D {
	if s.NoProvider {
		return th.composerControlW(gtx, composerControl{off: true, on: glass(), icon: "circle-alert", label: "No provider available"})
	}
	id := liveID(s, "model")
	hot := s.Picker != nil
	if id != "" {
		c, _ := th.events(gtx, id)
		hot = hot || c.Hovered()
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

	// The suffix: the effort word, the bolt, Ultracode's icon, the context
	// window.
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
	switch s.Speed {
	case "Fast":
		add(LucideFill("zap", muted, 12))
	case "Ultrafast":
		add(ultrafastBolt(muted))
	}
	if s.Ultracode {
		add(LucideW("workflow", muted, 12))
	}
	if s.Window != "" {
		add(func(gtx C) D {
			gtx.Constraints.Max.X = dp(gtx, 96)
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: muted, MaxLines: 1}, s.Window)
		})
	}
	suffix := record(sg, func(gtx C) D { return layout.Flex{Alignment: layout.Middle}.Layout(gtx, parts...) })

	// The label truncates to what's left, at most 224.
	icon := 0
	if s.Provider != "" {
		icon = dp(gtx, 16-4) + gap // its -2dp side margins
	}
	tail := 0
	if suffix.Size.X > 0 {
		tail = gap - dp(gtx, 2) + suffix.Size.X
	}
	lg := gtx
	lg.Constraints = layout.Constraints{Max: image.Pt(max(0, min(dp(gtx, 224), gtx.Constraints.Max.X-2*px-2*padX-icon-tail)), gtx.Constraints.Max.Y)}
	t.Color = label
	ld := record(lg, func(gtx C) D { return th.Text(gtx, t, name) })

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
	ld.call.Add(gtx.Ops)
	off.Pop()
	x += ld.Size.X
	if suffix.Size.X > 0 {
		off := op.Offset(image.Pt(x+gap-dp(gtx, 2), (h-suffix.Size.Y)/2)).Push(gtx.Ops)
		suffix.call.Add(gtx.Ops)
		off.Pop()
	}
	if id != "" {
		th.hit(gtx, id, r.Max)
	}
	return D{Size: r.Max}
}

// primaryActions is T3's ComposerPrimaryActions (§5.3): with a question
// open, the answer actions; while a run can be stopped (or a stop is on
// its way) with nothing typed, Stop; else send, with Stop after it while
// a stop is pending or landed, so the target clicked stays at the end.
func (th *Theme) primaryActions(gtx C, s *Scene, compact bool) D {
	running := s.State == ThreadRunning
	if len(s.Questions) > 0 {
		return th.answerActions(gtx, s, running, compact)
	}
	stopping := s.Stop == StopPending || s.Stop == StopTimedOut
	if (running || stopping) && !th.hasContent(s) && !s.EditingQueued {
		return th.stopButton(gtx, s, false)
	}
	if s.Stop == StopIdle {
		return th.sendButton(gtx, s, running)
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		rigid(func(gtx C) D { return th.sendButton(gtx, s, running) }),
		rigid(Spacer(8, 0)),
		rigid(func(gtx C) D { return th.stopButton(gtx, s, false) }),
	)
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

// sendArt is the send button's nightly artwork at 1x and 2x, and the same
// at 110% brightness for hover (send-night@*.png, from tools/t3ui/sendart.py).
var sendArt = map[[2]int]paint.ImageOp{}

func sendArtImage(scale int, bright bool) paint.ImageOp {
	k := [2]int{scale, 0}
	if bright {
		k[1] = 1
	}
	if im, ok := sendArt[k]; ok {
		return im
	}
	data, err := artFiles.ReadFile("art/send-night@" + strconv.Itoa(scale) + "x.png")
	if err != nil {
		panic(err)
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	img := image.NewNRGBA(src.Bounds())
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			if bright {
				// filter: brightness(1.1), in sRGB as Chrome applies it.
				up := func(v uint8) uint8 { return uint8(min(255, int(float32(v)*1.1+0.5))) }
				c.R, c.G, c.B = up(c.R), up(c.G), up(c.B)
			}
			img.SetNRGBA(x, y, c)
		}
	}
	im := paint.NewImageOp(img)
	sendArt[k] = im
	return im
}

// sendButton is the 32dp round send button. tern runs against T3's
// nightly, whose button wears the stage art (StageBackdropButtonArt):
// the nightly sky behind a white arrow, brightened and grown to 105% on
// hover, at 64% while there's nothing to send. While a run works it
// steers (corner-up-right), or queues (list-plus) with ⌘ or Ctrl held
// and on a ⌘- or Ctrl-click; for a queued message being edited it's a
// check; a spinner while it sends, when it's off.
func (th *Theme) sendButton(gtx C, s *Scene, running bool) D {
	id := liveID(s, "send")
	enabled := th.hasContent(s) && !s.Sending
	sz := dp(gtx, 32)
	r := image.Rect(0, 0, sz, sz)
	hovered := false
	if id != "" && enabled {
		hovered = th.modClick(gtx, id, r.Max)
	}
	queueing := running && !s.EditingQueued && th.in.composer.held
	label := "Submit message"
	switch {
	case s.EditingQueued:
		label = "Update queued message"
	case queueing:
		label = "Queue message"
	case running:
		label = "Steer message"
	}
	tip := label
	switch {
	case s.Sending && s.EditingQueued:
		tip = "Updating queued message"
	case s.Sending:
		tip = "Submitting message"
	case running && !s.EditingQueued:
		mod := "Ctrl"
		if Shortcut("")[0] != 'C' {
			mod = "⌘"
		}
		tip = "Click to steer, " + mod + "-click or " + Shortcut("Enter") + " to queue"
	}
	func() {
		if hovered {
			c := f32.Pt(float32(sz)/2, float32(sz)/2)
			defer op.Affine(f32.Affine2D{}.Scale(c, f32.Pt(1.05, 1.05))).Push(gtx.Ops).Pop()
		}
		on := glass()
		if !enabled {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		} else {
			// The control highlight is a 1dp line under it, not inset.
			fill(gtx, over(alpha(rgb(0xffffff), 0x29), on), r.Add(image.Pt(0, max(1, dp(gtx, 1)))), sz/2)
			shadow(gtx, r, sz/2, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x3d)}})
		}
		func() {
			defer clip.Ellipse{Max: r.Max}.Push(gtx.Ops).Pop()
			scale := 1
			if gtx.Metric.PxPerDp >= 1.5 {
				scale = 2
			}
			k := float32(sz) / float32(32*scale)
			defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(k, k))).Push(gtx.Ops).Pop()
			sendArtImage(scale, hovered).Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}()
		fg := rgb(0xffffff)
		icon := ""
		switch {
		case s.Sending:
			off := op.Offset(image.Pt((sz-dp(gtx, 14))/2, (sz-dp(gtx, 14))/2)).Push(gtx.Ops)
			T3Spinner(fg, 14, s.Phase/5)(gtx)
			off.Pop()
		case s.EditingQueued:
			icon = "check"
		case queueing:
			icon = "list-plus"
		case running:
			icon = "corner-up-right"
		default:
			sendArrow(gtx, fg, image.Pt((sz-dp(gtx, 14))/2, (sz-dp(gtx, 14))/2))
		}
		if icon != "" {
			off := op.Offset(image.Pt((sz-dp(gtx, 16))/2, (sz-dp(gtx, 16))/2)).Push(gtx.Ops)
			LucideW(icon, fg, 16)(gtx)
			off.Pop()
		}
	}()
	if id != "" {
		if !enabled {
			// The tooltip sits on a wrapper, so a disabled button still
			// explains itself; it takes no clicks.
			hovered = th.hoverArea(gtx, id+".tip", r.Max)
			th.tipShow(gtx, id+".tip", hovered, tip, r.Max, true, 0)
		} else {
			th.tipAt(gtx, id, tip, r.Max, true, 0)
		}
	}
	return D{Size: r.Max}
}

// modClick is click target id, size big, whose click carries ⌘ or Ctrl
// as Alt (T3's alternate dispatch); it reports whether the pointer's on
// it.
func (th *Theme) modClick(gtx C, id string, size image.Point) bool {
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
			th.push(Action{ID: id, Kind: "click", Alt: e.Modifiers.Contain(key.ModShortcut)})
		}
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	return c.Hovered()
}

// hoverArea reports whether the pointer's over a size-sized area that
// takes no clicks.
func (th *Theme) hoverArea(gtx C, id string, size image.Point) bool {
	if th.in.hovers == nil {
		th.in.hovers = map[string]*gesture.Hover{}
	}
	h := th.in.hovers[id]
	if h == nil {
		h = new(gesture.Hover)
		th.in.hovers[id] = h
	}
	over := h.Update(gtx.Source)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	h.Add(gtx.Ops)
	return over
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

// stopButton is Stop: a round button in --destructive at 90% (full on
// hover) with a 12dp white square, 32dp or 28 in the answer row (size-8
// sm:size-7). After a click it's "Stopping…", then "Stopped" once the run
// can't be stopped, both off at 64% until it can again (or 10s pass).
func (th *Theme) stopButton(gtx C, s *Scene, inRow bool) D {
	id := liveID(s, "stop")
	off := s.Stop == StopPending || s.Stop == StopLanded
	hovered := false
	if id != "" && !off {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	sz := dp(gtx, 32)
	if inRow {
		sz = dp(gtx, 28)
	}
	r := image.Rect(0, 0, sz, sz)
	on := glass()
	bg := over(alpha(P.Destructive, 0xe6), on)
	if hovered {
		bg = P.Destructive
	}
	tip := "Interrupt"
	switch s.Stop {
	case StopPending:
		tip = "Stopping…"
	case StopLanded:
		tip = "Stopped"
	}
	func() {
		if off {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		}
		fill(gtx, over(alpha(rgb(0xffffff), 0x29), on), r.Add(image.Pt(0, max(1, dp(gtx, 1)))), sz/2)
		shadow(gtx, r, sz/2, []boxShadow{{0, 1, 2, 0, alpha(P.Destructive, 0x3d)}})
		fill(gtx, bg, r, sz/2)
		k := float32(dp(gtx, 12)) / 12
		o := image.Pt((sz-dp(gtx, 12))/2, (sz-dp(gtx, 12))/2)
		sq := image.Rect(o.X+int(2*k+0.5), o.Y+int(2*k+0.5), o.X+int(10*k+0.5), o.Y+int(10*k+0.5))
		fill(gtx, rgb(0xffffff), sq, int(1.5*k+0.5))
	}()
	if id != "" {
		if off {
			hovered = th.hoverArea(gtx, id+".tip", r.Max)
			th.tipShow(gtx, id+".tip", hovered, tip, r.Max, true, 0)
		} else {
			th.hit(gtx, id, r.Max)
			th.tipAt(gtx, id, tip, r.Max, true, 0)
		}
	}
	return D{Size: r.Max}
}

// answerActions is the footer while a question is open: Previous past the
// first, the Submit pill, and Stop while the run can be stopped, 8 apart;
// under a 780dp composer they go compact, 6 apart, Previous an icon.
func (th *Theme) answerActions(gtx C, s *Scene, running, compact bool) D {
	gap := float32(8)
	if compact {
		gap = 6
	}
	var kids []layout.FlexChild
	if s.Question > 0 {
		id := liveID(s, "q.prev")
		b := Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Previous", Disabled: s.Answering, On: glass()}
		if compact {
			b = Button{Variant: BtnOutline, Size: BtnSizeIconSm, Icon: "chevron-left", IconSize: 14, Disabled: s.Answering, On: glass()}
		}
		kids = append(kids, rigid(func(gtx C) D {
			d := th.T3Button(id, b)(gtx)
			if compact && id != "" {
				th.tipAt(gtx, id, "Previous question", d.Size, true, 0)
			}
			return d
		}), rigid(Spacer(gap, 0)))
	}
	kids = append(kids, rigid(func(gtx C) D { return th.submitPill(gtx, s, compact) }))
	if running || s.Stop != StopIdle {
		kids = append(kids, rigid(Spacer(gap, 0)), rigid(func(gtx C) D { return th.stopButton(gtx, s, true) }))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
}

// submitPill is the question's primary action: a 28dp --message-action
// pill padded 16 (12 compact), "Next question", "Submit answer" or
// "Submit answers" ("Next" or "Submit" compact; "Submitting..." while the
// answers go), off until the question (or, at the last, every one) has
// its answer.
func (th *Theme) submitPill(gtx C, s *Scene, compact bool) D {
	last := s.Question >= len(s.Questions)-1
	label := "Next question"
	switch {
	case s.Answering:
		label = "Submitting..."
	case compact && last:
		label = "Submit"
	case compact:
		label = "Next"
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
	td := record(gtx, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.MessageActionForeground, MaxLines: 1}, label)
	})
	h, padX := dp(gtx, 28), dp(gtx, 16)
	if compact {
		padX = dp(gtx, 12)
	}
	r := image.Rect(0, 0, td.Size.X+2*padX, h)
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
	off := op.Offset(image.Pt(padX, (h-td.Size.Y)/2)).Push(gtx.Ops)
	td.call.Add(gtx.Ops)
	off.Pop()
	if id != "" && enabled {
		th.hit(gtx, id, r.Max)
	}
	return D{Size: r.Max}
}

// answered reports whether question i has an answer: an option picked,
// or typed into the composer, live for the question on show.
func (th *Theme) answered(s *Scene, i int) bool {
	q := s.Questions[i]
	for _, on := range q.Chosen {
		if on {
			return true
		}
	}
	switch {
	case q.NoCustom:
		return false
	case i != s.Question:
		return q.Typed
	case s.Live:
		return TrimText(th.EditorText("composer")) != ""
	}
	return s.Composer != ""
}

// composerTile is a picture of the composer's as T3's 64dp tile: rounded
// 8, a border at 80%, a click that opens it ("Preview <name>") and its
// remove button in the corner (ComposerAttachmentStrip.tsx).
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
			// The mockups show the remove button too, with nothing behind it.
			defer op.Offset(image.Pt(sz-dp(gtx, 24+4), dp(gtx, 4))).Push(gtx.Ops).Pop()
			mediaClose(gtx, 24, 14, false)
			return D{Size: r.Max}
		}
		pid := "image.preview:" + strconv.Itoa(i)
		if clicked, hovered := th.localClick(gtx, pid, r.Max); clicked && t.ok {
			th.openPreview(t)
		} else {
			th.tipShow(gtx, pid, hovered && t.Name != "", "Preview "+t.Name, r.Max, true, 0)
		}
		// media-close: 24dp, black at 65% (80% on hover), a white x.
		id := "image.remove:" + strconv.Itoa(i)
		c, _ := th.events(gtx, id)
		defer op.Offset(image.Pt(sz-dp(gtx, 24+4), dp(gtx, 4))).Push(gtx.Ops).Pop()
		mediaClose(gtx, 24, 14, c.Hovered())
		th.hit(gtx, id, image.Pt(dp(gtx, 24), dp(gtx, 24)))
		return D{Size: r.Max}
	}
}

// lucidePath is lucide icon name's shape sizeDp big, to fill or stroke.
func lucidePath(gtx C, name string, sizeDp float32) clip.PathSpec {
	k := float32(dp(gtx, sizeDp)) / 24
	v, ok := lucideSegs.Load(name)
	if !ok {
		v, _ = lucideSegs.LoadOrStore(name, parseSVGPath(lucide[name]))
	}
	var p clip.Path
	p.Begin(gtx.Ops)
	for _, sg := range v.([]seg) {
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
	return p.End()
}

// LucideFill is a lucide icon filled in col as well as stroked, as T3
// draws its bolt and play icons (fill-current).
func LucideFill(name string, col color.NRGBA, sizeDp float32) W {
	return func(gtx C) D {
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: lucidePath(gtx, name, sizeDp)}.Op())
		return LucideW(name, col, sizeDp)(gtx)
	}
}

// ultrafastBolt is FastBolt's Ultrafast (EffortGlyph.tsx): 15 wide by 12,
// the bolt outlined 3dp along, and over it at the start a filled one whose
// 4-unit stroke, under its fill, is the glass surface's colour, so the
// outline behind it is cut away round its edge.
func ultrafastBolt(col color.NRGBA) W {
	return func(gtx C) D {
		func() {
			defer op.Offset(image.Pt(dp(gtx, 3), 0)).Push(gtx.Ops).Pop()
			LucideW("zap", col, 12)(gtx)
		}()
		bolt := lucidePath(gtx, "zap", 12)
		paint.FillShape(gtx.Ops, glassSurface(), clip.Stroke{Path: bolt, Width: 4 * float32(dp(gtx, 12)) / 24}.Op())
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: bolt}.Op())
		return D{Size: image.Pt(dp(gtx, 15), dp(gtx, 12))}
	}
}
