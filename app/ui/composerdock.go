package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"

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
)

// What T3 attaches above the composer's card (ComposerBanner.tsx): notice
// banners and the agent's questions, 22dp in from the card's sides with
// their bottoms tucked 17dp under it; the @, / and $ menu, which rises out
// of the card the same way; the messages queued behind a run; and the
// context strip under a new thread's card.

// bannerVariant is an attached surface's tone (ComposerBanner.tsx).
type bannerVariant int

const (
	bannerDefault bannerVariant = iota
	bannerError
	bannerWarning
)

// colors are the variant's outline and tint over the glass: neutral ones
// take the card's outline, error and warning their colour at 32% and 28%
// with an 8% tint.
func (v bannerVariant) colors() (outline, tint color.NRGBA) {
	switch v {
	case bannerError:
		return alpha(P.Error, 0x52), alpha(P.Error, 0x14)
	case bannerWarning:
		return alpha(P.Warning, 0x47), alpha(P.Warning, 0x14)
	}
	return composerOutline(), color.NRGBA{}
}

// attachedGlass paints an attached surface over r. Its bottom 16dp are
// masked away and what follows covers it from 17dp up, so only the top
// rounds (16): a joined surface, one after another, is square and has no
// top edge. In dark, a shadow rises 10dp off the card below.
func attachedGlass(gtx C, r image.Rectangle, joined bool, v bannerVariant) color.NRGBA {
	outline, tint := v.colors()
	g := over(tint, glass())
	px := max(1, dp(gtx, 1))
	defer clip.Rect(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-dp(gtx, 16))).Push(gtx.Ops).Pop()
	edge := over(outline, g)
	if joined {
		fill(gtx, edge, r, 0)
		fill(gtx, g, image.Rect(r.Min.X+px, r.Min.Y, r.Max.X-px, r.Max.Y), 0)
	} else {
		rad := dp(gtx, composerRadius)
		fill(gtx, g, r, rad)
		strokeRect(gtx, edge, r, rad, float32(px))
	}
	if dark {
		seam := r.Max.Y - dp(gtx, tuck)
		band := image.Rect(r.Min.X+px, seam-dp(gtx, 10), r.Max.X-px, seam)
		st := clip.Rect(band).Push(gtx.Ops)
		paint.LinearGradientOp{
			Stop1: f32.Pt(0, float32(band.Max.Y)), Color1: alpha(rgb(0), 0x2e),
			Stop2: f32.Pt(0, float32(band.Min.Y)), Color2: alpha(rgb(0), 0),
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		st.Pop()
	}
	return g
}

// composerDock is what sits on the card's top edge: the notices, then the
// agent's question. A new thread shows its notices under the composer
// instead (heroNotices).
func (th *Theme) composerDock(gtx C, s *Scene) D {
	if s.Hero != "" || s.Scratch {
		return D{}
	}
	w := gtx.Constraints.Max.X
	inset := dp(gtx, drawerInset)
	cw := w - 2*inset
	y := 0
	n := 0
	attach := func(card func(gtx C, joined bool) D) {
		off := op.Offset(image.Pt(inset, y)).Push(gtx.Ops)
		g := gtx
		g.Constraints = layout.Constraints{Min: image.Pt(cw, 0), Max: image.Pt(cw, gtx.Constraints.Max.Y)}
		d := card(g, n > 0)
		off.Pop()
		y += d.Size.Y - dp(gtx, tuck)
		n++
	}
	if s.Error != "" {
		attach(func(gtx C, joined bool) D {
			return th.noticeCard(gtx, s, bannerError, "circle-alert", s.Error, liveID(s, "error"), joined, false)
		})
	}
	if s.Notice != "" {
		attach(func(gtx C, joined bool) D {
			return th.noticeCard(gtx, s, bannerDefault, "info", s.Notice, "", joined, false)
		})
	}
	if len(s.Questions) > 0 {
		attach(func(gtx C, joined bool) D { return th.questionDrawer(gtx, s, joined) })
	}
	th.questionKeys(gtx, s)
	if n == 0 {
		return D{}
	}
	return D{Size: image.Pt(w, y+dp(gtx, tuck))}
}

// noticeCard is a notice (ComposerBannerStack.tsx): a comfortable root
// padded 5 4, one 24dp row of the icon, the 12/24 medium title and the
// dismiss button, then the 17dp the next surface covers. Detached, under a
// new thread's composer, it's a whole 16dp-rounded card.
func (th *Theme) noticeCard(gtx C, s *Scene, v bannerVariant, icon, title, dismiss string, joined, detached bool) D {
	w := gtx.Constraints.Max.X
	rowH := dp(gtx, 24)
	padY, padX := dp(gtx, 5), dp(gtx, 4)
	h := 2*padY + rowH
	if !detached {
		h += dp(gtx, tuck)
	}
	r := image.Rect(0, 0, w, h)
	g := glass()
	if detached {
		outline, tint := v.colors()
		g = over(tint, glass())
		rad := dp(gtx, composerRadius)
		if !dark {
			shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}})
		}
		fill(gtx, g, r, rad)
		strokeRect(gtx, over(outline, g), r, rad, float32(max(1, dp(gtx, 1))))
	} else {
		g = attachedGlass(gtx, r, joined, v)
	}
	defer op.Offset(image.Pt(padX, padY)).Push(gtx.Ops).Pop()
	rw := w - 2*padX
	col := P.MutedForeground
	if v == bannerError {
		col = P.Error
	}
	func() {
		off := op.Offset(image.Pt((rowH-dp(gtx, 12))/2, (rowH-dp(gtx, 12))/2)).Push(gtx.Ops)
		defer off.Pop()
		LucideW(icon, col, 12)(gtx)
	}()
	actions := 0
	if dismiss != "" {
		actions = dp(gtx, 48)
		corner := dp(gtx, 12)
		if joined {
			corner = dp(gtx, 8)
		}
		bottom := dp(gtx, 8)
		if detached {
			bottom = dp(gtx, 12)
		}
		off := op.Offset(image.Pt(rw-rowH, 0)).Push(gtx.Ops)
		th.bannerDismiss(gtx, dismiss, g, corner, bottom)
		off.Pop()
	}
	tg := gtx
	tg.Constraints = layout.Constraints{Max: image.Pt(max(0, rw-rowH-dp(gtx, 4)-actions), rowH)}
	m := op.Record(gtx.Ops)
	td := th.Text(tg, Txt{Size: 12, LineH: 24.0 / 12, Font: UI(font.Medium), Color: bodyText(), MaxLines: 1}, title)
	call := m.Stop()
	off := op.Offset(image.Pt(rowH+dp(gtx, 4), (rowH-td.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return D{Size: r.Max}
}

// bannerDismiss is a banner's ghost icon-xs dismiss: 24dp, the x at 14,
// rounded 8 but for its top-right corner, which follows the card's.
func (th *Theme) bannerDismiss(gtx C, id string, on color.NRGBA, corner, bottom int) {
	c, _ := th.events(gtx, id)
	sz := dp(gtx, 24)
	col := P.ContrastMutedForeground
	if c.Hovered() {
		rad := dp(gtx, 8)
		paint.FillShape(gtx.Ops, over(P.Accent, on), clip.RRect{Rect: image.Rect(0, 0, sz, sz), NW: rad, NE: corner, SE: bottom, SW: rad}.Op(gtx.Ops))
		col = P.Foreground
	}
	off := op.Offset(image.Pt((sz-dp(gtx, 14))/2, (sz-dp(gtx, 14))/2)).Push(gtx.Ops)
	LucideW("x", col, 14)(gtx)
	off.Pop()
	th.hit(gtx, id, image.Pt(sz, sz))
}

// heroNotices are a new thread's notices, detached under its composer:
// 16dp below it, 4 apart.
func (th *Theme) heroNotices(gtx C, s *Scene) D {
	if s.Error == "" && s.Notice == "" {
		return D{}
	}
	var kids []layout.FlexChild
	add := func(v bannerVariant, icon, text, dismiss string) {
		if len(kids) > 0 {
			kids = append(kids, rigid(Spacer(0, 4)))
		}
		kids = append(kids, rigid(func(gtx C) D { return th.noticeCard(gtx, s, v, icon, text, dismiss, false, true) }))
	}
	if s.Error != "" {
		add(bannerError, "circle-alert", s.Error, liveID(s, "error"))
	}
	if s.Notice != "" {
		add(bannerDefault, "info", s.Notice, "")
	}
	return pad(16, 0, 0, 0).Layout(gtx, func(gtx C) D { return Col(gtx, kids...) })
}

// questionDrawer is the agent's question on show (ComposerPendingUserInputPanel.tsx):
// a header row that folds it away, then the question and its options,
// each with its digit.
func (th *Theme) questionDrawer(gtx C, s *Scene, joined bool) D {
	w := gtx.Constraints.Max.X
	q := s.Questions[min(s.Question, len(s.Questions)-1)]
	folded := th.in.composer.qFolded == q.Header+"\x00"+q.Text
	m := op.Record(gtx.Ops)
	body := pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Col(gtx,
			rigid(func(gtx C) D { return th.questionHeader(gtx, s, q, folded) }),
			rigid(func(gtx C) D {
				if folded {
					return D{}
				}
				return th.questionBody(gtx, s, q)
			}),
		)
	})
	call := m.Stop()
	r := image.Rect(0, 0, w, body.Size.Y+dp(gtx, tuck))
	attachedGlass(gtx, r, joined, bannerDefault)
	call.Add(gtx.Ops)
	return D{Size: r.Max}
}

// questionHeader is the drawer's toggle row: 36dp, padded 6 16, the
// header in muted 12/16 (folded, the question after a dot), then the
// count, the chevron and the dismiss button.
func (th *Theme) questionHeader(gtx C, s *Scene, q Question, folded bool) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, 36)
	g := glass()
	if s.Live {
		// Folding is the drawer's own: the row's hover shows before what's on it.
		m := op.Record(gtx.Ops)
		clicked, hovered := th.localClick(gtx, "q.fold", image.Pt(w, h))
		hit := m.Stop()
		if clicked {
			key := q.Header + "\x00" + q.Text
			if folded {
				key = ""
			}
			th.in.composer.qFolded = key
		}
		if hovered {
			fill(gtx, over(alpha(P.Foreground, 0x0d), g), image.Rect(0, 0, w, h), dp(gtx, 12))
		}
		hit.Add(gtx.Ops)
	}
	defer op.Offset(image.Pt(dp(gtx, 16), dp(gtx, 6))).Push(gtx.Ops).Pop()
	rw, rh := w-dp(gtx, 32), dp(gtx, 24)
	// The actions, right to left: dismiss, chevron, count.
	x := rw
	if s.Dismissible {
		x -= rh
		off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
		th.bannerDismiss(gtx, liveID(s, "dismiss"), g, dp(gtx, 8), dp(gtx, 8))
		off.Pop()
	}
	x -= rh
	func() {
		defer op.Offset(image.Pt(x+(rh-dp(gtx, 14))/2, (rh-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
		icon := "chevron-down"
		if folded {
			icon = "chevron-up"
		}
		LucideW(icon, P.ContrastMutedForeground, 14)(gtx)
	}()
	if len(s.Questions) > 1 {
		count := strconv.Itoa(s.Question+1) + "/" + strconv.Itoa(len(s.Questions))
		m := op.Record(gtx.Ops)
		cd := th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.MutedForeground, MaxLines: 1}, count)
		call := m.Stop()
		cw := max(rh, cd.Size.X)
		x -= cw
		off := op.Offset(image.Pt(x+(cw-cd.Size.X)/2, (rh-cd.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
	// The header, from 6dp in.
	tx := dp(gtx, 6)
	room := x - dp(gtx, 4) - tx
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}
	hg := gtx
	hg.Constraints = layout.Constraints{Max: image.Pt(max(0, room), rh)}
	if folded {
		hg.Constraints.Max.X = room / 2
	}
	m := op.Record(gtx.Ops)
	hd := th.Text(hg, t, q.Header)
	call := m.Stop()
	off := op.Offset(image.Pt(tx, (rh-hd.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	if folded {
		sx := tx + hd.Size.X + dp(gtx, 4)
		m := op.Record(gtx.Ops)
		sd := th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: over(alpha(P.MutedForeground, 0x66), g), MaxLines: 1}, "·")
		call := m.Stop()
		so := op.Offset(image.Pt(sx, (rh-sd.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		so.Pop()
		qx := sx + sd.Size.X + dp(gtx, 4)
		qg := gtx
		qg.Constraints = layout.Constraints{Max: image.Pt(max(0, x-dp(gtx, 4)-qx), rh)}
		qt := t
		qt.Color = P.SecondaryLabel
		m = op.Record(gtx.Ops)
		qd := th.Text(qg, qt, q.Text)
		call = m.Stop()
		qo := op.Offset(image.Pt(qx, (rh-qd.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		qo.Pop()
	}
	return D{Size: image.Pt(w, h)}
}

// questionBody is the question in 14/22.75 at foreground 85%, then its
// options: a radio (a checkbox when several can be picked), the label and
// description, and the digit that picks it.
func (th *Theme) questionBody(gtx C, s *Scene, q Question) D {
	g := glass()
	return pad(0, 16, 4, 22).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		kids := []layout.FlexChild{rigid(func(gtx C) D {
			return pad(2, 0, 0, 0).Layout(gtx, func(gtx C) D {
				return th.cssText(gtx, promptText(over(alpha(P.Foreground, 0xd9), g)), q.Text)
			})
		})}
		for i, o := range q.Options {
			i, o := i, o
			kids = append(kids, rigid(func(gtx C) D {
				top := float32(0)
				if i == 0 {
					top = 6
				}
				return pad(top, 0, 0, -6).Layout(gtx, func(gtx C) D { return th.questionOption(gtx, s, q, i, o) })
			}))
		}
		return Col(gtx, kids...)
	})
}

// questionOption is one option: padded 6 0 6 6, 10 between its parts,
// rounded 6; the picked one in foreground, the rest at 70% and lit by
// --muted at 30% under the pointer.
func (th *Theme) questionOption(gtx C, s *Scene, q Question, i int, label string) D {
	w := gtx.Constraints.Max.X
	g := glass()
	chosen := i < len(q.Chosen) && q.Chosen[i]
	id := liveID(s, "q:"+strconv.Itoa(s.Question)+":"+strconv.Itoa(i))
	hovered := false
	if id != "" && !s.Answering {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	text := over(alpha(P.Foreground, 0xb3), g)
	if chosen {
		text = P.Foreground
	}
	desc := ""
	if i < len(q.Descs) && q.Descs[i] != label {
		desc = q.Descs[i]
	}
	gap, slot := dp(gtx, 10), dp(gtx, 20)
	lx := dp(gtx, 6) + dp(gtx, 16) + gap
	kw := dp(gtx, 24)
	m := op.Record(gtx.Ops)
	lg := gtx
	lg.Constraints = layout.Constraints{Min: image.Pt(max(0, w-lx-gap-kw), 0), Max: image.Pt(max(0, w-lx-gap-kw), gtx.Constraints.Max.Y)}
	ld := Col(lg,
		rigid(func(gtx C) D {
			return th.cssText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: text}, label)
		}),
		rigid(func(gtx C) D {
			if desc == "" {
				return D{}
			}
			return pad(2, 0, 0, 0).Layout(gtx, func(gtx C) D {
				return th.cssText(gtx, Txt{Size: 11, LineH: 16.0 / 11, Color: P.SecondaryLabel}, desc)
			})
		}),
	)
	call := m.Stop()
	h := ld.Size.Y + dp(gtx, 12)
	r := image.Rect(0, 0, w, h)
	if s.Answering {
		defer paint.PushOpacity(gtx.Ops, 0.5).Pop()
	}
	if hovered && !chosen {
		fill(gtx, over(alpha(P.Muted, uint8(int(P.Muted.A)*3/10)), g), r, dp(gtx, 6))
	}
	py := dp(gtx, 6)
	func() {
		defer op.Offset(image.Pt(dp(gtx, 6), py+(slot-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		if q.MultiSel {
			checkMark(gtx, chosen, g)
		} else {
			optionRadio(gtx, chosen, g)
		}
	}()
	off := op.Offset(image.Pt(lx, py)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	if i < 9 {
		m := op.Record(gtx.Ops)
		kd := th.T3Kbd(strconv.Itoa(i+1), g)(gtx)
		kc := m.Stop()
		ko := op.Offset(image.Pt(w-kw+(kw-kd.Size.X)/2, py+(slot-kd.Size.Y)/2)).Push(gtx.Ops)
		kc.Add(gtx.Ops)
		ko.Pop()
	}
	if id != "" && !s.Answering {
		th.hit(gtx, id, r.Max)
	}
	return D{Size: r.Max}
}

// optionRadio is the drawer's radio, drawn with ui/radio-group's look: a
// 16dp circle with a border at foreground 55% on --background (on
// --input at 32% in dark); picked, --primary with a 6dp white dot.
func optionRadio(gtx C, on bool, bg color.NRGBA) {
	sz := dp(gtx, 16)
	r := image.Rect(0, 0, sz, sz)
	if on {
		circle(gtx, P.Primary, image.Pt(sz/2, sz/2), sz/2)
		circle(gtx, P.PrimaryForeground, image.Pt(sz/2, sz/2), dp(gtx, 3))
		return
	}
	fillc := P.Background
	if dark {
		fillc = over(alpha(P.Input, uint8(int(P.Input.A)*32/100)), bg)
	} else {
		shadow(gtx, r, sz/2, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}})
	}
	circle(gtx, fillc, image.Pt(sz/2, sz/2), sz/2)
	strokeRect(gtx, over(alpha(P.Foreground, 0x8c), fillc), r, sz/2, float32(max(1, dp(gtx, 1))))
}

// checkMark is the multi-select's checkbox: a 16dp square rounded 4,
// bordered like the radio; picked, --primary with a white check.
func checkMark(gtx C, on bool, bg color.NRGBA) {
	sz := dp(gtx, 16)
	r := image.Rect(0, 0, sz, sz)
	rad := dp(gtx, 4)
	if on {
		fill(gtx, P.Primary, r, rad)
		off := op.Offset(image.Pt((sz-dp(gtx, 12))/2, (sz-dp(gtx, 12))/2)).Push(gtx.Ops)
		LucideW("check", P.PrimaryForeground, 12)(gtx)
		off.Pop()
		return
	}
	fillc := P.Background
	if dark {
		fillc = over(alpha(P.Input, uint8(int(P.Input.A)*32/100)), bg)
	}
	fill(gtx, fillc, r, rad)
	strokeRect(gtx, over(alpha(P.Foreground, 0x8c), fillc), r, rad, float32(max(1, dp(gtx, 1))))
}

// questionKeys picks the question's options with 1 to 9, while the focus
// isn't in a text field (T3 leaves digits to fields).
func (th *Theme) questionKeys(gtx C, s *Scene) {
	if !s.Live || len(s.Questions) == 0 || s.Answering {
		return
	}
	var filters []event.Filter
	for n := 1; n <= 9; n++ {
		filters = append(filters, key.Filter{Name: key.Name(strconv.Itoa(n))})
	}
	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			return
		}
		ke, isKey := ev.(key.Event)
		if !isKey || ke.State != key.Press || th.typing(gtx) {
			continue
		}
		i := int(ke.Name[0] - '1')
		if q := s.Questions[s.Question]; i < len(q.Options) {
			th.push(Action{ID: "q:" + strconv.Itoa(s.Question) + ":" + strconv.Itoa(i), Kind: "click"})
		}
	}
}

// typing reports whether one of the app's text fields has the focus.
func (th *Theme) typing(gtx C) bool {
	for _, ed := range th.in.editors {
		if gtx.Focused(ed) {
			return true
		}
	}
	return false
}

// queue is the messages waiting behind the run (QueuedRunsControl.tsx):
// each a dashed, unfilled bubble at the end of the lane with Send now and
// remove under it, 12 apart and 16 over the composer. Past three, the
// first two show and a button shows the rest.
func (th *Theme) queue(gtx C, s *Scene) D {
	if len(s.Queue) == 0 {
		return D{}
	}
	rows := s.Queue
	more := len(rows) > 3 && !th.in.composer.queueAll
	if more {
		rows = rows[:2]
	}
	var kids []layout.FlexChild
	for i, text := range rows {
		i, text := i, text
		if i > 0 {
			kids = append(kids, rigid(Spacer(0, 12)))
		}
		kids = append(kids, rigid(func(gtx C) D { return th.queuedRow(gtx, s, i, text) }))
	}
	if len(s.Queue) > 3 {
		kids = append(kids, rigid(Spacer(0, 8)), rigid(func(gtx C) D {
			label := "Show fewer"
			if more {
				label = "Show all " + strconv.Itoa(len(s.Queue)) + " queued"
			}
			return layout.E.Layout(gtx, func(gtx C) D {
				return pad(0, 4, 0, 0).Layout(gtx, func(gtx C) D {
					t := Txt{Size: 12, LineH: 16.0 / 12}
					m := op.Record(gtx.Ops)
					d := th.inlineButton(gtx, "", label, t, false)
					call := m.Stop()
					size := image.Pt(d.Size.X+dp(gtx, 2+14), d.Size.Y)
					clicked, hovered := th.localClick(gtx, "queue.all", size)
					if clicked {
						th.in.composer.queueAll = !th.in.composer.queueAll
					}
					if hovered {
						t.Color = P.Foreground
						d = th.inlineButton(gtx, "", label, t, false)
						underline(gtx, P.Foreground, d, 2, t.Size)
					} else {
						call.Add(gtx.Ops)
					}
					icon, col := "chevron-down", P.MutedForeground
					if !more {
						icon = "chevron-up"
					}
					if hovered {
						col = P.Foreground
					}
					off := op.Offset(image.Pt(d.Size.X+dp(gtx, 2), (d.Size.Y-dp(gtx, 14))/2)).Push(gtx.Ops)
					LucideW(icon, col, 14)(gtx)
					off.Pop()
					return D{Size: size}
				})
			})
		}))
	}
	return pad(0, 0, 16, 0).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Col(gtx, kids...)
	})
}

// queuedRow is one queued message: the bubble, at most 80% of the lane,
// rounded 16 with a dashed border at muted 40%, its text in muted 14/20;
// then 4 under it, its actions.
func (th *Theme) queuedRow(gtx C, s *Scene, i int, text string) D {
	w := gtx.Constraints.Max.X
	m := op.Record(gtx.Ops)
	bg := gtx
	bg.Constraints = layout.Constraints{Max: image.Pt(w*8/10-dp(gtx, 24), gtx.Constraints.Max.Y)}
	td := th.cssText(bg, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, text)
	call := m.Stop()
	bw, bh := td.Size.X+dp(gtx, 24), td.Size.Y+dp(gtx, 16)
	func() {
		defer op.Offset(image.Pt(w-bw, 0)).Push(gtx.Ops).Pop()
		dashedRRect(gtx, over(alpha(P.MutedForeground, 0x66), P.Background), image.Rect(0, 0, bw, bh), dp(gtx, 16))
		defer op.Offset(image.Pt(dp(gtx, 12), dp(gtx, 8))).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
	}()
	y := bh + dp(gtx, 4)
	am := op.Record(gtx.Ops)
	ad := layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		rigid(th.T3Button(liveID(s, "queue:"+strconv.Itoa(i)+":steer"), Button{Variant: BtnGhostMuted, Size: BtnSizeXs, Icon: "arrow-up", Label: "Send now"})),
		rigid(Spacer(2, 0)),
		rigid(th.T3Button(liveID(s, "queue:"+strconv.Itoa(i)+":cancel"), Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "x"})),
	)
	acts := am.Stop()
	off := op.Offset(image.Pt(w-ad.Size.X-dp(gtx, 4), y)).Push(gtx.Ops)
	acts.Add(gtx.Ops)
	off.Pop()
	if id := liveID(s, "queue:"+strconv.Itoa(i)+":steer"); id != "" {
		tip := "Send into the current turn"
		if s.State != ThreadRunning {
			tip = "Start a turn with this message"
		}
		th.tipAt(gtx, id, tip, image.Pt(ad.Size.X, ad.Size.Y), true, 0)
	}
	return D{Size: image.Pt(w, y+ad.Size.Y)}
}

// inlineButton is T3's InlineButton (button.tsx), click target id: medium
// text, muted until the pointer's on it, then foreground and underlined
// 2dp under the baseline. The picker tone is foreground throughout and
// underlines 4dp under while open. t gives the size, and the colour when
// there's no target.
func (th *Theme) inlineButton(gtx C, id, label string, t Txt, picker bool) D {
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	t.Font, t.MaxLines = UI(font.Medium), 1
	if id != "" {
		t.Color = P.MutedForeground
		if picker || hovered {
			t.Color = P.Foreground
		}
	} else if t.Color.A == 0 {
		t.Color = P.MutedForeground
	}
	d := th.Text(gtx, t, label)
	if hovered && !picker {
		underline(gtx, t.Color, d, 2, t.Size)
	}
	if id != "" {
		th.hit(gtx, id, d.Size)
	}
	return d
}

// underline draws text d's underline offset dp under its baseline, as
// thick as the font's (about a sixteenth of its size).
func underline(gtx C, col color.NRGBA, d D, offset float32, size unit.Sp) {
	y := d.Size.Y - d.Baseline + dp(gtx, offset)
	thick := max(1, int(float32(gtx.Sp(size))/16+0.5))
	fill(gtx, col, image.Rect(0, y, d.Size.X, y+thick), 0)
}

// composerUI is the composer's own state: what's folded or shown, the
// targets that change it, and what its keys need from the scene.
type composerUI struct {
	qFolded  string // the folded question, by header and text
	queueAll bool   // the whole queue shows
	clicks   map[string]*gesture.Click
	menu     bool    // the @, / or $ menu is open: the arrows, Enter and Tab are its
	history  bool    // the arrows may walk the prompt history: no pictures, no question
	recalled string  // the prompt the history last put in
	thumbs   []Thumb // the pictures, for their chips
}

// localClick makes a size-sized area a target for ui state alone: it
// reports a click on it since the last frame, and whether the pointer's on it.
func (th *Theme) localClick(gtx C, id string, size image.Point) (clicked, hovered bool) {
	ui := &th.in.composer
	if ui.clicks == nil {
		ui.clicks = map[string]*gesture.Click{}
	}
	c := ui.clicks[id]
	if c == nil {
		c = new(gesture.Click)
		ui.clicks[id] = c
	}
	for {
		e, ok := c.Update(gtx.Source)
		if !ok {
			break
		}
		clicked = clicked || e.Kind == gesture.KindClick
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
	pointer.CursorPointer.Add(gtx.Ops)
	return clicked, c.Hovered()
}

// dashedRRect strokes the rounded rect r with Chrome's 1dp dashes, 3 on
// and 3 off, inside its edge.
func dashedRRect(gtx C, col color.NRGBA, r image.Rectangle, rad int) {
	w := float32(max(1, dp(gtx, 1)))
	x0, y0 := float32(r.Min.X)+w/2, float32(r.Min.Y)+w/2
	x1, y1 := float32(r.Max.X)-w/2, float32(r.Max.Y)-w/2
	rr := min(float32(rad)-w/2, (x1-x0)/2, (y1-y0)/2)
	// The outline as points, the corners as quarter circles.
	var pts []f32.Point
	arc := func(cx, cy float32, a0 float64) {
		for i := 0; i <= 8; i++ {
			a := a0 + float64(i)/8*math.Pi/2
			pts = append(pts, f32.Pt(cx+rr*float32(math.Cos(a)), cy+rr*float32(math.Sin(a))))
		}
	}
	arc(x1-rr, y0+rr, -math.Pi/2)
	arc(x1-rr, y1-rr, 0)
	arc(x0+rr, y1-rr, math.Pi/2)
	arc(x0+rr, y0+rr, math.Pi)
	pts = append(pts, pts[0])
	dash := float32(dp(gtx, 3))
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(pts[0])
	left, drawing := dash, true
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		seg := b.Sub(a)
		l := float32(math.Hypot(float64(seg.X), float64(seg.Y)))
		if l == 0 {
			continue
		}
		dir := seg.Mul(1 / l)
		for l > 0 {
			step := min(l, left)
			a = a.Add(dir.Mul(step))
			if drawing {
				p.LineTo(a)
			} else {
				p.MoveTo(a)
			}
			l -= step
			left -= step
			if left <= 0 {
				drawing, left = !drawing, dash
			}
		}
	}
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: w}.Op())
}

// stripGlass is the context strip's surface-glass: --background at 80%
// over the canvas.
func stripGlass() color.NRGBA { return over(alpha(P.Background, 0xcc), P.Background) }

// contextStrip is the row under a new thread's card (BranchToolbar.tsx):
// page-coloured glass, padded 8 8 0, its permissions chip at the end.
func (th *Theme) contextStrip(gtx C, s *Scene, access *image.Rectangle) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, 32)
	fill(gtx, stripGlass(), image.Rect(0, 0, w, h), 0)
	m := op.Record(gtx.Ops)
	cd := th.permissionsChip(gtx, s, true)
	call := m.Stop()
	at := image.Pt(w-dp(gtx, 8)-cd.Size.X, dp(gtx, 8))
	off := op.Offset(at).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	*access = image.Rectangle{Min: at, Max: at.Add(cd.Size)}
	return D{Size: image.Pt(w, h)}
}

// commandMenu is the @, / and $ menu (ComposerCommandMenu.tsx): an
// attached surface 22dp in from the card's sides, rising from 17dp under
// its top edge, holding up to 288dp of 28dp rows padded 8; with no rows,
// the line saying why.
func (th *Theme) commandMenu(gtx C, s *Scene, w int) {
	inset := dp(gtx, drawerInset)
	cw := w - 2*inset
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(cw, 0), Max: image.Pt(cw, dp(gtx, 288))}
	var ld D
	if len(s.Suggest) == 0 {
		ld = pad(14, 20, 28-float32(tuck), 20).Layout(g, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.SecondaryLabel}, s.SuggestEmpty)
		})
	} else {
		ld = pad(8, 8, 8, 8).Layout(g, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			var kids []layout.FlexChild
			for i, it := range s.Suggest {
				i, it := i, it
				kids = append(kids, rigid(func(gtx C) D { return th.commandItem(gtx, s, i, it) }))
			}
			return Col(gtx, kids...)
		})
	}
	call := m.Stop()
	h := ld.Size.Y + dp(gtx, tuck)
	defer op.Offset(image.Pt(inset, dp(gtx, tuck)-h)).Push(gtx.Ops).Pop()
	r := image.Rect(0, 0, cw, h)
	attachedGlass(gtx, r, false, bannerDefault)
	call.Add(gtx.Ops)
}

// commandItem is one row: 28dp, padded 6 8, rounded 8; its icon, the
// 12/16 medium label (at most 45%), the muted description and a skill's
// source badge. The highlighted row is filled with --accent; the pointer
// moves the highlight.
func (th *Theme) commandItem(gtx C, s *Scene, i int, it Suggestion) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, 28)
	id := liveID(s, "suggest:"+strconv.Itoa(i))
	if id != "" {
		c, _ := th.events(gtx, id)
		if c.Hovered() && i != s.SuggestSel {
			th.push(Action{ID: "suggest", Kind: "hover", Text: strconv.Itoa(i)})
		}
	}
	g := glass()
	text := bodyText()
	if i == s.SuggestSel {
		fill(gtx, over(P.Accent, g), image.Rect(0, 0, w, h), dp(gtx, 8))
		text = P.AccentForeground
	}
	x := dp(gtx, 8)
	if it.Icon != "" {
		off := op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops)
		LucideW(it.Icon, P.IconMuted, 16)(gtx)
		off.Pop()
		x += dp(gtx, 16+8)
	}
	room := w - x - dp(gtx, 8)
	t := Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: text, MaxLines: 1}
	lx := x
	if it.Prefix != "" {
		pt := t
		pt.Color = P.SecondaryLabel
		m := op.Record(gtx.Ops)
		pd := th.Text(gtx, pt, it.Prefix)
		call := m.Stop()
		off := op.Offset(image.Pt(lx, (h-pd.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		lx += pd.Size.X
	}
	lg := gtx
	lg.Constraints = layout.Constraints{Max: image.Pt(max(0, room*45/100-(lx-x)), h)}
	m := op.Record(gtx.Ops)
	ld := th.Text(lg, t, it.Label)
	call := m.Stop()
	off := op.Offset(image.Pt(lx, (h-ld.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	dx := lx + ld.Size.X + dp(gtx, 8)
	badge := 0
	if it.Badge != "" {
		bm := op.Record(gtx.Ops)
		bd := th.T3Badge(BadgeSecondary, "", it.Badge, g)(gtx)
		bc := bm.Stop()
		badge = bd.Size.X + dp(gtx, 8)
		bo := op.Offset(image.Pt(w-dp(gtx, 8)-bd.Size.X, (h-bd.Size.Y)/2)).Push(gtx.Ops)
		bc.Add(gtx.Ops)
		bo.Pop()
	}
	dg := gtx
	dg.Constraints = layout.Constraints{Max: image.Pt(max(0, w-dp(gtx, 8)-badge-dx), h)}
	m = op.Record(gtx.Ops)
	dd := th.Text(dg, Txt{Size: 12, LineH: 16.0 / 12, Color: P.SecondaryLabel, MaxLines: 1}, it.Desc)
	call = m.Stop()
	do := op.Offset(image.Pt(dx, (h-dd.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	do.Pop()
	if id != "" {
		th.hit(gtx, id, image.Pt(w, h))
	}
	return D{Size: image.Pt(w, h)}
}
