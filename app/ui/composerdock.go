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
	"gioui.org/unit"
)

// What T3 attaches above the composer's card (ComposerBanner.tsx): the
// status notice and the agent's questions, 22dp in from the card's sides
// with their bottoms tucked 17dp under it; the @, / and $ menu and the
// stash, which rise out of the card the same way; the messages queued
// behind a run, at the foot of the timeline; and the context strip under
// a new thread's card.

// attachedGlass paints an attached surface over r. Its bottom 16dp are
// masked away and what follows covers it from 17dp up, so only the top
// rounds (16): a joined surface, one after another, is square and has no
// top edge. Its shadow is the composer's, light and dark (shadow-composer,
// shadow-composer-dark), masked with it. A surface portaled out of the
// composer (the @ menu, the stash) misses the composer's outline and takes
// foreground 8% instead. In dark, a seam rises 10dp off the card below.
func attachedGlass(gtx C, r image.Rectangle, joined, portaled bool) color.NRGBA {
	g := glass()
	px := max(1, dp(gtx, 1))
	outline := composerOutline()
	if portaled && !dark {
		outline = alpha(P.Foreground, 0x14)
	}
	edge := over(outline, g)
	rad := dp(gtx, composerRadius)
	defer clip.Rect(image.Rect(r.Min.X-dp(gtx, 32), r.Min.Y-dp(gtx, 32), r.Max.X+dp(gtx, 32), r.Max.Y-dp(gtx, 16))).Push(gtx.Ops).Pop()
	if !joined {
		shadow(gtx, r, rad, composerShadow())
	}
	if joined {
		fill(gtx, edge, r, 0)
		fill(gtx, g, image.Rect(r.Min.X+px, r.Min.Y, r.Max.X-px, r.Max.Y), 0)
	} else {
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

// composerShadow is shadow-composer: 0 1px 2px black 5% in light; dark's
// banners and detached cards cast shadow-composer-dark, 0 14px 32px -18px
// black 75%.
func composerShadow() []boxShadow {
	if dark {
		return []boxShadow{{0, 14, 32, -18, alpha(rgb(0), 0xbf)}}
	}
	return []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}}
}

// composerDock is what sits on the card's top edge (ComposerBanner.Dock):
// a column of the armed /clear's banner and the status notice (not while
// a question is open, which keeps only the machine's own banners), then
// the agent's question, and
// at its end the stash's tab, their bottoms lined up. A new thread's
// notices go under its composer (heroNotices); its tab still shows here.
func (th *Theme) composerDock(gtx C, s *Scene) D {
	w := gtx.Constraints.Max.X
	inset := dp(gtx, drawerInset)
	cw := w - 2*inset
	var tab func()
	tabH := 0
	if len(s.Stash) > 0 {
		td := record(gtx, func(gtx C) D { return th.stashTab(gtx, s) })
		tabH = td.Size.Y
		cw -= td.Size.X + dp(gtx, 4)
		tab = func() {
			defer op.Offset(image.Pt(w-inset-td.Size.X, 0)).Push(gtx.Ops).Pop()
			td.call.Add(gtx.Ops)
		}
	}
	hero := s.Hero != "" || s.Scratch
	y, n := 0, 0
	col := record(gtx, func(gtx C) D {
		attach := func(card func(gtx C, joined bool) D) {
			off := op.Offset(image.Pt(inset, y)).Push(gtx.Ops)
			g := gtx
			g.Constraints = layout.Constraints{Min: image.Pt(cw, 0), Max: image.Pt(cw, gtx.Constraints.Max.Y)}
			d := card(g, n > 0)
			off.Pop()
			y += d.Size.Y - dp(gtx, tuck)
			n++
		}
		if s.Fresh && !hero && len(s.Questions) == 0 {
			attach(func(gtx C, joined bool) D { return th.noticeCard(gtx, freshNotice(s), joined, false) })
		}
		if s.Notice != "" && !hero && len(s.Questions) == 0 {
			attach(func(gtx C, joined bool) D {
				return th.noticeCard(gtx, notice{icon: "info", title: s.Notice}, joined, false)
			})
		}
		if len(s.Questions) > 0 && !hero {
			attach(func(gtx C, joined bool) D { return th.questionDrawer(gtx, s, joined) })
		}
		return D{}
	})
	th.questionKeys(gtx, s)
	colH := 0
	if n > 0 {
		colH = y + dp(gtx, tuck)
	}
	h := max(colH, tabH)
	if h == 0 {
		return D{}
	}
	// The column and the tab sit on the dock's bottom (items-end).
	off := op.Offset(image.Pt(0, h-colH)).Push(gtx.Ops)
	col.call.Add(gtx.Ops)
	off.Pop()
	if tab != nil {
		off := op.Offset(image.Pt(0, h-tabH)).Push(gtx.Ops)
		tab()
		off.Pop()
	}
	return D{Size: image.Pt(w, h)}
}

// stashPulse is the tab's pulse after a stash (index.css): the row in the
// foreground for 1.2s, easing back over 0.2s; the count in --primary for
// 1.2s, rising 2dp into place as it fades in over 180ms.
const stashPulse = 1400 * time.Millisecond

// stashTab is the stash's shoulder tab (ComposerStashBadge.tsx): an
// attached surface as wide as its row, "Stash" and the count, muted
// until hovered; while the drawer's open it's foreground and inert.
func (th *Theme) stashTab(gtx C, s *Scene) D {
	id := liveID(s, "stash")
	hovered := false
	if id != "" && !s.StashOpen {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	col, count := P.MutedForeground, P.MutedForeground
	rise, fade := 0, float32(1)
	if since := gtx.Now.Sub(s.Stashed); !s.Stashed.IsZero() && since >= 0 && since < stashPulse {
		k := float32(1)
		if since > 1200*time.Millisecond {
			k = 1 - easeOut(float32(since-1200*time.Millisecond)/float32(200*time.Millisecond))
		}
		col = lerpCol(P.Foreground, P.MutedForeground, k)
		if since < 1200*time.Millisecond {
			count = P.Primary
		}
		if since < 180*time.Millisecond {
			t := easeOut(float32(since) / float32(180*time.Millisecond))
			rise, fade = dp(gtx, 2*(1-t)), t
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	if hovered || s.StashOpen {
		col = P.Foreground
	}
	rd := record(gtx, func(gtx C) D {
		return th.bannerRow(gtx, "bookmark", "Stash", strconv.Itoa(len(s.Stash)), col, count, false, 0, rise, fade)
	})
	padX, padY := dp(gtx, 4), dp(gtx, 5)
	r := image.Rect(0, 0, rd.Size.X+2*padX, rd.Size.Y+2*padY+dp(gtx, tuck))
	g := attachedGlass(gtx, r, false, false)
	defer op.Offset(image.Pt(padX, padY)).Push(gtx.Ops).Pop()
	if hovered {
		fill(gtx, over(alpha(P.Foreground, 0x0d), g), image.Rectangle{Max: rd.Size}, dp(gtx, 12))
	}
	rd.call.Add(gtx.Ops)
	if id != "" && !s.StashOpen {
		th.hit(gtx, id, rd.Size)
	}
	return D{Size: r.Max}
}

// bannerRow is a banner's 24dp row (ComposerBanner.Row): a [24][1fr][auto]
// grid of the 12dp icon in its column, the 12/16 content and the actions:
// with a count, a 24dp column for it in medium, tabular, in countCol
// (risen rise px and at fade, for the stash's pulse); with toggle, the
// open drawer's chevron after it. w stretches the row to w, the actions
// at its end; 0 sizes it to its content.
func (th *Theme) bannerRow(gtx C, icon, content, count string, col, countCol color.NRGBA, toggle bool, w, rise int, fade float32) D {
	gtx.Constraints.Min = image.Point{}
	rh := dp(gtx, 24)
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}
	off := op.Offset(image.Pt((rh-dp(gtx, 12))/2, (rh-dp(gtx, 12))/2)).Push(gtx.Ops)
	LucideW(icon, col, 12)(gtx)
	off.Pop()
	x := rh + dp(gtx, 4)
	cd := record(gtx, func(gtx C) D { return th.Text(gtx, t, content) })
	off = op.Offset(image.Pt(x, (rh-cd.Size.Y)/2)).Push(gtx.Ops)
	cd.call.Add(gtx.Ops)
	off.Pop()
	x += cd.Size.X
	// The actions: the count's column and the chevron's.
	ct := t
	ct.Font, ct.Color = UI(font.Medium), countCol
	nd := record(gtx, func(gtx C) D { return th.Text(gtx, ct, count) })
	aw := 0
	if count != "" {
		aw += dp(gtx, 4) + max(rh, nd.Size.X)
	}
	if toggle {
		aw += rh
	}
	ax := x
	if w > 0 {
		x = max(x+aw, w)
		ax = x - aw
	} else {
		x += aw
	}
	if count != "" {
		ax += dp(gtx, 4)
		cw := max(rh, nd.Size.X)
		func() {
			defer op.Offset(image.Pt(ax+(cw-nd.Size.X)/2, (rh-nd.Size.Y)/2+rise)).Push(gtx.Ops).Pop()
			defer paint.PushOpacity(gtx.Ops, fade).Pop()
			nd.call.Add(gtx.Ops)
		}()
		ax += cw
	}
	if toggle {
		off := op.Offset(image.Pt(ax+(rh-dp(gtx, 14))/2, (rh-dp(gtx, 14))/2)).Push(gtx.Ops)
		LucideW("chevron-down", P.ContrastMutedForeground, 14)(gtx)
		off.Pop()
	}
	return D{Size: image.Pt(x, rh)}
}

// stashDrawer is the open stash (ComposerStashMenu.tsx), rising out of
// the card like the @ menu: a row that closes it, its count and chevron
// at the end, then the stashed prompts' rows (snippet, pictures, age,
// delete) in a scroll area at most min(384, 40% of the window) tall, the
// highlighted one filled with --accent; or the line saying how to stash.
func (th *Theme) stashDrawer(gtx C, s *Scene, w int) {
	inset := dp(gtx, drawerInset)
	cw := w - 2*inset
	g := glass()
	inner := cw - dp(gtx, 8)
	cg := gtx
	cg.Constraints = layout.Constraints{Min: image.Pt(inner, 0), Max: image.Pt(inner, 1<<20)}
	head := record(cg, func(gtx C) D {
		id := liveID(s, "stash")
		hovered := false
		if id != "" {
			c, _ := th.events(gtx, id)
			hovered = c.Hovered()
		}
		rd := record(gtx, func(gtx C) D {
			return th.bannerRow(gtx, "bookmark", "Stash", strconv.Itoa(len(s.Stash)), P.MutedForeground, P.MutedForeground, true, inner, 0, 1)
		})
		if hovered {
			fill(gtx, over(alpha(P.Foreground, 0x0d), g), image.Rectangle{Max: rd.Size}, dp(gtx, 12))
		}
		rd.call.Add(gtx.Ops)
		if id != "" {
			th.hit(gtx, id, rd.Size)
		}
		return rd.D
	})
	rows := record(cg, func(gtx C) D {
		if len(s.Stash) == 0 {
			return th.bannerRow(gtx, "", "Nothing stashed yet. Press "+Shortcut("S")+" with a prompt in the composer to stash it.", "", P.MutedForeground, P.MutedForeground, false, inner, 0, 1)
		}
		var kids []layout.FlexChild
		for i, e := range s.Stash {
			if i > 0 {
				kids = append(kids, rigid(Spacer(0, 1)))
			}
			kids = append(kids, rigid(func(gtx C) D { return th.stashRow(gtx, s, i, e, g) }))
		}
		return Col(gtx, kids...)
	})
	view := min(rows.Size.Y, dp(gtx, 384), th.in.window.Y*40/100)
	if th.in.window.Y == 0 {
		view = min(rows.Size.Y, dp(gtx, 384))
	}
	ui := &th.in.composer
	if !ui.stashWas {
		th.scrollArea("stash").y = 0
	}
	ui.stashShown = true
	if s.StashSel >= 0 && s.StashSel < len(s.Stash) && (!ui.stashWas || ui.stashSel != s.StashSel) {
		ui.stashSel = s.StashSel
		top := s.StashSel * (dp(gtx, 24) + dp(gtx, 1))
		th.scrollTo("stash", top, top+dp(gtx, 24), view)
	}
	pad4 := dp(gtx, 4)
	h := pad4 + head.Size.Y + view + pad4 + dp(gtx, tuck)
	defer op.Offset(image.Pt(inset, dp(gtx, tuck)-h)).Push(gtx.Ops).Pop()
	attachedGlass(gtx, image.Rect(0, 0, cw, h), false, true)
	defer op.Offset(image.Pt(pad4, pad4)).Push(gtx.Ops).Pop()
	head.call.Add(gtx.Ops)
	defer op.Offset(image.Pt(0, head.Size.Y)).Push(gtx.Ops).Pop()
	th.vscroll(gtx, "stash", image.Pt(inner, view), rows.Size.Y, rows.call)
}

// stashRow is one stashed prompt: 24dp, rounded 4, a click restores it;
// a pointer moving over it moves the highlight to it.
func (th *Theme) stashRow(gtx C, s *Scene, i int, e Stashed, g color.NRGBA) D {
	w := gtx.Constraints.Max.X
	gtx.Constraints.Min = image.Point{}
	rh := dp(gtx, 24)
	id := liveID(s, "stash.restore:"+strconv.Itoa(i))
	on := g
	text := over(alpha(P.Foreground, 0xcc), g)
	if i == s.StashSel {
		on = over(P.Accent, g)
		fill(gtx, on, image.Rect(0, 0, w, rh), dp(gtx, 4))
		text = P.AccentForeground
	}
	// The actions, right to left: delete, age, pictures, what was dropped.
	x := w - rh
	func() {
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		th.bannerDismiss(gtx, liveID(s, "stash.delete:"+strconv.Itoa(i)), on, dp(gtx, 8), dp(gtx, 8))
	}()
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}
	place := func(w W) {
		d := record(gtx, w)
		x -= d.Size.X + dp(gtx, 4)
		off := op.Offset(image.Pt(x, (rh-d.Size.Y)/2)).Push(gtx.Ops)
		d.call.Add(gtx.Ops)
		off.Pop()
	}
	if e.Age != "" {
		place(func(gtx C) D { return th.Text(gtx, t, e.Age) })
	}
	if n := min(3, len(e.Thumbs)); n > 0 {
		place(func(gtx C) D {
			// -space-x-1.5: each one 6dp over the one before, on top of it.
			sz, step := dp(gtx, 16), dp(gtx, 16-6)
			for j := 0; j < n; j++ {
				off := op.Offset(image.Pt(j*step, 0)).Push(gtx.Ops)
				r := image.Rect(0, 0, sz, sz)
				if !drawThumb(gtx, e.Thumbs[j], sz, dp(gtx, 4)) {
					fill(gtx, over(P.Muted, on), r, dp(gtx, 4))
				}
				strokeRect(gtx, over(alpha(P.Border, uint8(int(P.Border.A)*7/10)), on), r, dp(gtx, 4), float32(max(1, dp(gtx, 1))))
				off.Pop()
			}
			return D{Size: image.Pt((n-1)*step+sz, sz)}
		})
	}
	if e.Dropped > 0 {
		dt := t
		dt.Color = P.WarningForeground
		place(func(gtx C) D { return th.Text(gtx, dt, plural(e.Dropped, "image")+" dropped") })
	}
	off := op.Offset(image.Pt((rh-dp(gtx, 12))/2, (rh-dp(gtx, 12))/2)).Push(gtx.Ops)
	LucideW("file-text", P.MutedForeground, 12)(gtx)
	off.Pop()
	sg := gtx
	sg.Constraints = layout.Constraints{Max: image.Pt(max(0, x-rh-dp(gtx, 8)), rh)}
	st := t
	st.Color = text
	sd := record(sg, func(gtx C) D { return th.Text(gtx, st, e.Snippet) })
	so := op.Offset(image.Pt(rh+dp(gtx, 4), (rh-sd.Size.Y)/2)).Push(gtx.Ops)
	sd.call.Add(gtx.Ops)
	so.Pop()
	if id != "" {
		th.hit(gtx, id, image.Pt(x, rh))
		if th.pointerMoved(gtx, id, image.Pt(w, rh)) && i != s.StashSel {
			th.push(Action{ID: "stash", Kind: "hover", Text: strconv.Itoa(i)})
		}
	}
	return D{Size: image.Pt(w, rh)}
}

// pointerMoved watches a size-sized area for the pointer moving over it,
// without taking its clicks, and reports whether it did since the last
// frame: a list's highlight follows the mouse's moves, not a mouse at
// rest (onMouseMove), so the arrow keys aren't undone. It goes over the
// row's own target: Gio hands a pointer event down only through areas
// that pass it on, so under the target it would never see one.
func (th *Theme) pointerMoved(gtx C, id string, size image.Point) bool {
	if th.in.movers == nil {
		th.in.movers = map[string]*int{}
	}
	tag := th.in.movers[id]
	if tag == nil {
		tag = new(int)
		th.in.movers[id] = tag
	}
	moved := false
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Move})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Move {
			moved = true
		}
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, tag)
	return moved
}

// notice is one of the composer's notice cards: its icon, its title and
// description, and whether it has a dismiss button, whose click target
// is dismiss ("" in a mockup).
type notice struct {
	icon, title, desc string
	dismissible       bool
	dismiss           string
}

// freshNotice is the banner while /clear is armed (ComposerFreshSession.tsx);
// its dismiss keeps the current session.
func freshNotice(s *Scene) notice {
	return notice{icon: "eraser", title: "Your next message starts a fresh session", desc: "The agent won't see earlier messages in this thread.",
		dismissible: true, dismiss: liveID(s, "fresh.keep")}
}

// noticeCard is a notice (ComposerBannerStack.tsx): a comfortable root
// padded 5 4, one 24dp wrap-actions row of the 12dp icon, then the 12/24
// medium title and the muted 12/16 description filling what's left (none
// on a card under 400), then a 48dp column ending in its dismiss; then the
// 17dp the next surface covers. Detached, under a new thread's composer,
// it's a whole 16dp-rounded card.
func (th *Theme) noticeCard(gtx C, n notice, joined, detached bool) D {
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
		rad := dp(gtx, composerRadius)
		shadow(gtx, r, rad, composerShadow())
		fill(gtx, g, r, rad)
		strokeRect(gtx, over(composerOutline(), g), r, rad, float32(max(1, dp(gtx, 1))))
	} else {
		g = attachedGlass(gtx, r, joined, false)
	}
	defer op.Offset(image.Pt(padX, padY)).Push(gtx.Ops).Pop()
	rw := w - 2*padX
	func() {
		off := op.Offset(image.Pt((rowH-dp(gtx, 12))/2, (rowH-dp(gtx, 12))/2)).Push(gtx.Ops)
		defer off.Pop()
		LucideW(n.icon, P.MutedForeground, 12)(gtx)
	}()
	room := rw - rowH - dp(gtx, 4)
	if n.dismissible {
		room -= dp(gtx, 4+48)
		// The first card's dismiss follows its top-right corner; a
		// detached one's, its bottom-right too.
		corner, bottom := dp(gtx, 12), dp(gtx, 8)
		if joined {
			corner = dp(gtx, 8)
		}
		if detached {
			bottom = dp(gtx, 12)
		}
		off := op.Offset(image.Pt(rw-rowH, 0)).Push(gtx.Ops)
		th.bannerDismiss(gtx, n.dismiss, g, corner, bottom)
		off.Pop()
	}
	tg := gtx
	tg.Constraints = layout.Constraints{Max: image.Pt(max(0, room), rowH)}
	td := record(tg, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 12, LineH: 24.0 / 12, Font: UI(font.Medium), Color: bodyText(), MaxLines: 1}, n.title)
	})
	x := rowH + dp(gtx, 4)
	off := op.Offset(image.Pt(x, (rowH-td.Size.Y)/2)).Push(gtx.Ops)
	td.call.Add(gtx.Ops)
	off.Pop()
	if rest := room - td.Size.X - dp(gtx, 4); n.desc != "" && w >= dp(gtx, 400) && rest > 0 {
		dg := gtx
		dg.Constraints = layout.Constraints{Max: image.Pt(rest, rowH)}
		dd := record(dg, func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}, n.desc)
		})
		off := op.Offset(image.Pt(x+td.Size.X+dp(gtx, 4), (rowH-dd.Size.Y)/2)).Push(gtx.Ops)
		dd.call.Add(gtx.Ops)
		off.Pop()
	}
	return D{Size: r.Max}
}

// bannerDismiss is a banner's ghost icon-xs dismiss: 24dp, the x at 14,
// rounded 8 but for its top-right corner, which follows the card's. With
// no id (a mockup) it's drawn and takes no clicks.
func (th *Theme) bannerDismiss(gtx C, id string, on color.NRGBA, corner, bottom int) {
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	sz := dp(gtx, 24)
	col := P.ContrastMutedForeground
	if hovered {
		rad := dp(gtx, 8)
		paint.FillShape(gtx.Ops, over(P.Accent, on), clip.RRect{Rect: image.Rect(0, 0, sz, sz), NW: rad, NE: corner, SE: bottom, SW: rad}.Op(gtx.Ops))
		col = P.Foreground
	}
	off := op.Offset(image.Pt((sz-dp(gtx, 14))/2, (sz-dp(gtx, 14))/2)).Push(gtx.Ops)
	LucideW("x", col, 14)(gtx)
	off.Pop()
	if id != "" {
		th.hit(gtx, id, image.Pt(sz, sz))
	}
}

// heroNotices are a new thread's notices, detached under its composer:
// 16dp below it, 4 apart.
func (th *Theme) heroNotices(gtx C, s *Scene) D {
	if s.Notice == "" {
		return D{}
	}
	return pad(16, 0, 0, 0).Layout(gtx, func(gtx C) D { return th.noticeCard(gtx, notice{icon: "info", title: s.Notice}, false, true) })
}

// threadError is T3's ThreadErrorBanner in the chat's banner lane
// (ChatBannerLane.tsx), over the top of the conversation, 12dp down and
// centred in a lane w wide, at most 768 and 16dp in from each side: the
// error Alert on alert glass, rounded 12, padded 12 14, the alert icon
// and the 14/20 error in --error-foreground at 80%, its dismiss at the end.
func (th *Theme) threadError(gtx C, s *Scene, w int) {
	if s.Error == "" {
		return
	}
	most := min(dp(gtx, 768), w-dp(gtx, 32))
	padX, padY := dp(gtx, 14), dp(gtx, 12)
	// alert-glass[data-variant=error]: --background at 80% over the page,
	// --error at 4% over that.
	surface := over(alpha(P.Error, 0x0a), over(alpha(P.Background, 0xcc), P.Background))
	id := liveID(s, "error")
	room := most - 2*padX - 2*max(1, dp(gtx, 1)) - dp(gtx, 16+8) - dp(gtx, 24+8)
	tg := gtx
	tg.Constraints = layout.Constraints{Max: image.Pt(max(0, room), 1<<20)}
	text := record(tg, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: over(alpha(P.ErrorForeground, 0xcc), surface)}, s.Error)
	})
	inner := dp(gtx, 16+8) + text.Size.X + dp(gtx, 8+24)
	px := max(1, dp(gtx, 1))
	bw := inner + 2*padX + 2*px
	// With an action the row is 24 tall and the first line 2dp down.
	top := dp(gtx, 2)
	bh := max(text.Size.Y+top, dp(gtx, 24)) + 2*padY + 2*px
	at := image.Pt((gtx.Constraints.Max.X-bw)/2, dp(gtx, 12))
	defer op.Offset(at).Push(gtx.Ops).Pop()
	r := image.Rect(0, 0, bw, bh)
	rad := dp(gtx, 12)
	fill(gtx, surface, r, rad)
	strokeRect(gtx, over(alpha(P.Error, 0x52), surface), r, rad, float32(px))
	o := op.Offset(image.Pt(px+padX, px+padY+top)).Push(gtx.Ops)
	func() {
		defer op.Offset(image.Pt(0, (dp(gtx, 20)-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		LucideW("circle-alert", P.Error, 16)(gtx)
	}()
	func() {
		defer op.Offset(image.Pt(dp(gtx, 16+8), 0)).Push(gtx.Ops).Pop()
		text.call.Add(gtx.Ops)
	}()
	o.Pop()
	defer op.Offset(image.Pt(bw-px-padX-dp(gtx, 24), px+padY)).Push(gtx.Ops).Pop()
	th.bannerDismiss(gtx, id, surface, dp(gtx, 8), dp(gtx, 8))
	if id != "" {
		th.tipAt(gtx, id, "Dismiss error", image.Pt(dp(gtx, 24), dp(gtx, 24)), true, 0)
	}
}

// questionDrawer is the agent's question on show (ComposerPendingUserInputPanel.tsx):
// a header row that folds it away, then the question and its options,
// each with its digit, in a scroll area at most min(384, 40% of the
// window) tall (ComposerBanner.Scroll).
func (th *Theme) questionDrawer(gtx C, s *Scene, joined bool) D {
	w := gtx.Constraints.Max.X
	q := s.Questions[min(s.Question, len(s.Questions)-1)]
	folded := th.in.composer.qFolded == q.Header+"\x00"+q.Text
	inner := w - dp(gtx, 8)
	ig := gtx
	ig.Constraints = layout.Constraints{Min: image.Pt(inner, 0), Max: image.Pt(inner, 1<<20)}
	head := record(ig, func(gtx C) D { return th.questionHeader(gtx, s, q, folded) })
	h := dp(gtx, 4) + head.Size.Y
	var body recording
	view := 0
	if !folded {
		body = record(ig, func(gtx C) D { return th.questionBody(gtx, s, q) })
		view = min(body.Size.Y, dp(gtx, 384))
		if th.in.window.Y > 0 {
			view = min(view, th.in.window.Y*40/100)
		}
	}
	h += view + dp(gtx, 4)
	r := image.Rect(0, 0, w, h+dp(gtx, tuck))
	attachedGlass(gtx, r, joined, false)
	defer op.Offset(image.Pt(dp(gtx, 4), dp(gtx, 4))).Push(gtx.Ops).Pop()
	head.call.Add(gtx.Ops)
	if !folded {
		defer op.Offset(image.Pt(0, head.Size.Y)).Push(gtx.Ops).Pop()
		y := th.vscroll(gtx, "question", image.Pt(inner, view), body.Size.Y, body.call)
		th.scrollFade(gtx, image.Pt(inner, view), y, body.Size.Y-view-y, glass())
	}
	return D{Size: r.Max}
}

// scrollFade is ScrollArea's scrollFade: the content fades into the
// surface over up to 24dp at an edge it can scroll past, as far as it can
// (02-primitives port note 16).
func (th *Theme) scrollFade(gtx C, size image.Point, above, below int, surface color.NRGBA) {
	band := func(y0, dir, depth int) {
		if depth <= 0 {
			return
		}
		n := min(depth, dp(gtx, 24))
		for i := 0; i < n; i++ {
			a := float64(n-i) / float64(n)
			y := y0 + dir*i
			fill(gtx, alpha(surface, uint8(math.Round(a*a*255))), image.Rect(0, y, size.X, y+1), 0)
		}
	}
	band(0, 1, above)
	band(size.Y-1, -1, below)
}

// questionHeader is the drawer's toggle row: 36dp, padded 6 16, the
// header in muted 12/16 (folded, the question after a dot), then the
// count, the chevron and the dismiss button.
func (th *Theme) questionHeader(gtx C, s *Scene, q Question, folded bool) D {
	w := gtx.Constraints.Max.X
	gtx.Constraints.Min = image.Point{}
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
	gtx.Constraints.Min = image.Point{}
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

// queue is the messages waiting behind the run (QueuedRunsControl.tsx),
// at the foot of the timeline: each a dashed, unfilled bubble at the end
// of the lane with its actions under it, 12 apart, 16 over what follows.
// Past three, the first two show and a button shows the rest.
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
	for i, q := range rows {
		if i > 0 {
			kids = append(kids, rigid(Spacer(0, 12)))
		}
		kids = append(kids, rigid(func(gtx C) D { return th.queuedRow(gtx, s, i, q) }))
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
					rec := record(gtx, func(gtx C) D { return th.inlineLabel(gtx, label, t, toneMuted, false) })
					size := image.Pt(rec.Size.X+dp(gtx, 2+14), rec.Size.Y)
					clicked, hovered := th.localClick(gtx, "queue.all", size)
					if clicked {
						th.in.composer.queueAll = !th.in.composer.queueAll
					}
					d := th.inlineLabel(gtx, label, t, toneMuted, hovered)
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
// then 4 under it, its actions 2 apart: Send now, then edit and remove,
// or while it's in the composer, that and Cancel.
func (th *Theme) queuedRow(gtx C, s *Scene, i int, q QueuedMessage) D {
	w := gtx.Constraints.Max.X
	gtx.Constraints.Min = image.Point{}
	bg := gtx
	bg.Constraints = layout.Constraints{Max: image.Pt(w*8/10-dp(gtx, 24), gtx.Constraints.Max.Y)}
	text := record(bg, func(gtx C) D {
		return th.cssText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, q.Text)
	})
	bw, bh := text.Size.X+dp(gtx, 24), text.Size.Y+dp(gtx, 16)
	func() {
		defer op.Offset(image.Pt(w-bw, 0)).Push(gtx.Ops).Pop()
		dashedRRect(gtx, over(alpha(P.MutedForeground, 0x66), P.Background), image.Rect(0, 0, bw, bh), dp(gtx, 16))
		defer op.Offset(image.Pt(dp(gtx, 12), dp(gtx, 8))).Push(gtx.Ops).Pop()
		text.call.Add(gtx.Ops)
	}()
	type action struct {
		id, tip string
		b       Button
		note    string // text in its place
	}
	id := func(what string) string { return liveID(s, "queue:"+strconv.Itoa(i)+":"+what) }
	steer := "Send into the current turn"
	if s.State != ThreadRunning {
		steer = "Start a turn with this message"
	}
	if i == 0 {
		steer += " (" + ShortcutShift("Enter") + ")"
	}
	var acts []action
	if !q.Editing {
		acts = append(acts, action{id: id("steer"), tip: steer, b: Button{Variant: BtnGhostMuted, Size: BtnSizeXs, Icon: "arrow-up", Label: "Send now"}})
	}
	if q.Editing {
		acts = append(acts, action{note: "Editing in the composer"}, action{id: id("cancel-edit"), b: Button{Variant: BtnGhost, Size: BtnSizeXs, Label: "Cancel"}})
	} else {
		edit := "Edit in the composer"
		if i == len(s.Queue)-1 {
			edit += " (" + AltKey("Up") + ")"
		}
		acts = append(acts,
			action{id: id("edit"), tip: edit, b: Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "pencil"}},
			action{id: id("cancel"), tip: "Remove from queue", b: Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "x"}})
	}
	var recs []recording
	var ids, tips []string
	aw := 0
	for j, a := range acts {
		var r recording
		if a.note != "" {
			r = record(gtx, func(gtx C) D {
				return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
					return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.Foreground, MaxLines: 1}, a.note)
				})
			})
		} else {
			r = record(gtx, th.T3Button(a.id, a.b))
		}
		if j > 0 {
			aw += dp(gtx, 2)
		}
		aw += r.Size.X
		recs, ids, tips = append(recs, r), append(ids, a.id), append(tips, a.tip)
	}
	y := bh + dp(gtx, 4)
	x := w - aw - dp(gtx, 4)
	ah := 0
	for j, r := range recs {
		func() {
			defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
			r.call.Add(gtx.Ops)
			if ids[j] != "" {
				th.tipAt(gtx, ids[j], tips[j], r.Size, true, 0)
			}
		}()
		x += r.Size.X + dp(gtx, 2)
		ah = max(ah, r.Size.Y)
	}
	return D{Size: image.Pt(w, y+ah)}
}

// inlineTone is an InlineButton's tone (button.tsx).
type inlineTone int

const (
	toneMuted   inlineTone = iota // muted; on hover foreground, underlined 2dp under
	toneDefault                   // foreground, underlined 2dp under on hover
	tonePicker                    // foreground, underlined 4dp under on hover and while open
)

// inlineButton is T3's InlineButton, click target id: medium text in its
// tone, at t's size.
func (th *Theme) inlineButton(gtx C, id, label string, t Txt, tone inlineTone) D {
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	d := th.inlineLabel(gtx, label, t, tone, hovered)
	if id != "" {
		th.hit(gtx, id, d.Size)
	}
	return d
}

// inlineLabel draws an InlineButton's text, lit as hovered (or open) says.
func (th *Theme) inlineLabel(gtx C, label string, t Txt, tone inlineTone, hovered bool) D {
	t.Font, t.MaxLines, t.Color = UI(font.Medium), 1, P.MutedForeground
	if tone != toneMuted || hovered {
		t.Color = P.Foreground
	}
	d := th.Text(gtx, t, label)
	if hovered {
		offset := float32(2)
		if tone == tonePicker {
			offset = 4
		}
		underline(gtx, t.Color, d, offset, t.Size)
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
	menu     bool    // the @, / or $ menu is open: Escape is its
	rows     bool    // and has rows: so are the arrows, Enter and Tab
	stash    bool    // the stash's list is open: the arrows, Enter and Escape are its
	history  bool    // the arrows may walk the prompt history: no pictures, no question
	recalled string  // the prompt the history last put in
	thumbs   []Thumb // the pictures, for their chips
	// suggestion: the empty prompt shows the provider's guess, which Tab
	// takes and Escape dismisses; queue is how many messages are queued.
	suggestion bool
	queue      int
	focused    bool // the prompt had the focus last frame
	held       bool // ⌘ or Ctrl is down: the send button queues
	// previewMode is the permissions segment under the pointer, plus one.
	previewMode  int
	approvalMenu string // the approval whose More menu is open
	// The @ menu and the stash list: whether they showed this frame and
	// the last, and their highlight and rows then, to keep it in view.
	menuShown, menuWas, stashShown, stashWas bool
	menuSel, stashSel                        int
	menuKey                                  string
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
// over what it blurs (composerBackdrop).
func stripGlass() color.NRGBA { return over(alpha(P.Background, 0xcc), composerBackdrop()) }

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
// its top edge. Its list of 28dp rows, padded 8, is at most 288dp tall
// with its padding and scrolls past that, the highlighted row kept in
// view with 24dp under it (scroll-pb-6); with no rows, the line saying
// why.
func (th *Theme) commandMenu(gtx C, s *Scene, w int) {
	inset := dp(gtx, drawerInset)
	cw := w - 2*inset
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(cw, 0), Max: image.Pt(cw, 1<<20)}
	ui := &th.in.composer
	var list recording
	view := 0
	if len(s.Suggest) == 0 {
		list = record(g, func(gtx C) D {
			return pad(14, 20, 28-float32(tuck), 20).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.SecondaryLabel}, s.SuggestEmpty)
			})
		})
		view = list.Size.Y
	} else {
		list = record(g, func(gtx C) D {
			return pad(8, 8, 8, 8).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				var kids []layout.FlexChild
				for i, it := range s.Suggest {
					kids = append(kids, rigid(func(gtx C) D { return th.commandItem(gtx, s, i, it) }))
				}
				return Col(gtx, kids...)
			})
		})
		view = min(list.Size.Y, dp(gtx, 288))
		// A new menu starts at its top; a highlight the keys moved scrolls
		// into view.
		key := s.Suggest[0].Prefix + s.Suggest[0].Label + "\x00" + strconv.Itoa(len(s.Suggest))
		if !ui.menuWas || ui.menuKey != key {
			th.scrollArea("suggest").y = 0
		}
		if !ui.menuWas || ui.menuSel != s.SuggestSel || ui.menuKey != key {
			top := dp(gtx, 8) + s.SuggestSel*dp(gtx, 28)
			th.scrollTo("suggest", top-dp(gtx, 8), top+dp(gtx, 28)+dp(gtx, 24), view)
		}
		ui.menuKey, ui.menuSel = key, s.SuggestSel
	}
	ui.menuShown = true
	h := view + dp(gtx, tuck)
	defer op.Offset(image.Pt(inset, dp(gtx, tuck)-h)).Push(gtx.Ops).Pop()
	attachedGlass(gtx, image.Rect(0, 0, cw, h), false, true)
	if len(s.Suggest) == 0 {
		list.call.Add(gtx.Ops)
		return
	}
	th.vscroll(gtx, "suggest", image.Pt(cw, view), list.Size.Y, list.call)
}

// commandItem is one row: 28dp, padded 6 8, rounded 8; its icon, the
// 12/16 medium label (at most 45%), the muted description and a skill's
// source badge with its icon. The highlighted row is filled with
// --accent; a pointer moving over a row moves the highlight to it.
func (th *Theme) commandItem(gtx C, s *Scene, i int, it Suggestion) D {
	w := gtx.Constraints.Max.X
	gtx.Constraints.Min = image.Point{}
	h := dp(gtx, 28)
	id := liveID(s, "suggest:"+strconv.Itoa(i))
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
		pd := record(gtx, func(gtx C) D { return th.Text(gtx, pt, it.Prefix) })
		off := op.Offset(image.Pt(lx, (h-pd.Size.Y)/2)).Push(gtx.Ops)
		pd.call.Add(gtx.Ops)
		off.Pop()
		lx += pd.Size.X
	}
	lg := gtx
	lg.Constraints = layout.Constraints{Max: image.Pt(max(0, room*45/100-(lx-x)), h)}
	ld := record(lg, func(gtx C) D { return th.Text(gtx, t, it.Label) })
	off := op.Offset(image.Pt(lx, (h-ld.Size.Y)/2)).Push(gtx.Ops)
	ld.call.Add(gtx.Ops)
	off.Pop()
	dx := lx + ld.Size.X + dp(gtx, 8)
	badge := 0
	if it.Badge != "" {
		bd := record(gtx, th.T3Badge(BadgeSecondary, it.BadgeIcon, it.Badge, g))
		badge = bd.Size.X + dp(gtx, 8)
		bo := op.Offset(image.Pt(w-dp(gtx, 8)-bd.Size.X, (h-bd.Size.Y)/2)).Push(gtx.Ops)
		bd.call.Add(gtx.Ops)
		bo.Pop()
	}
	dg := gtx
	dg.Constraints = layout.Constraints{Max: image.Pt(max(0, w-dp(gtx, 8)-badge-dx), h)}
	dd := record(dg, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.SecondaryLabel, MaxLines: 1}, it.Desc)
	})
	do := op.Offset(image.Pt(dx, (h-dd.Size.Y)/2)).Push(gtx.Ops)
	dd.call.Add(gtx.Ops)
	do.Pop()
	if id != "" {
		th.hit(gtx, id, image.Pt(w, h))
		if th.pointerMoved(gtx, id, image.Pt(w, h)) && i != s.SuggestSel {
			th.push(Action{ID: "suggest", Kind: "hover", Text: strconv.Itoa(i)})
		}
	}
	return D{Size: image.Pt(w, h)}
}
