package ui

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The window's toasts (02-primitives.md §2.22, §3.24, §4.11, §5.11, §6;
// toast.tsx over Base UI's Toast): stacked at the top right, 360 wide, 32
// in from the right and 84 down, under the 52dp top bar. The newest is in
// front and the two before it peek 12 below it, scaled to 90 and 80% and
// blank; with the pointer on the stack they spread out 12 apart. Each
// closes 5s after it came unless it's sticky, the time not counting while
// the pointer is on the stack. They slide in from the right and out to it,
// and move between places over 500ms on T3's toast curve.

const (
	toastW       = 360
	toastTimeout = 5 * time.Second
	toastMove    = 500 * time.Millisecond
	toastLimit   = 3
)

// toastState is what the toasts keep from frame to frame.
type toastState struct {
	life    map[string]*toastLife
	leaving []*toastLife
	hover   gesture.Hover
	hovered bool
	last    time.Time
}

// toastLife is one toast's: how it last looked, when it came and went,
// how long it's been up, and its place, moving from one to the next.
type toastLife struct {
	toast      Toast
	came, went time.Time
	up         time.Duration
	asked      bool // its time is up and core has been asked to close it
	placed     bool // it has had a place; a new toast starts in its own
	from, to   toastPlace
	at         time.Time
}

// toastPlace is where a toast is drawn: its offset down the stack, scale,
// height, its content's opacity and its own.
type toastPlace struct {
	y, scale, h, content, alpha float32
}

func (p toastPlace) lerp(q toastPlace, t float32) toastPlace {
	l := func(a, b float32) float32 { return a + (b-a)*t }
	return toastPlace{l(p.y, q.y), l(p.scale, q.scale), l(p.h, q.h), l(p.content, q.content), l(p.alpha, q.alpha)}
}

// now is where l is at t.
func (l *toastLife) now(t time.Time) toastPlace {
	return l.from.lerp(l.to, toastEase(since(t, l.at, toastMove)))
}

// toastEase is cubic-bezier(.22, 1, .36, 1).
func toastEase(t float32) float32 { return bezier(0.22, 1, 0.36, 1, t) }

// bezier is the CSS timing function cubic-bezier(x1, y1, x2, y2) at t.
func bezier(x1, y1, x2, y2, t float32) float32 {
	if t <= 0 || t >= 1 {
		return clamp01(t)
	}
	at := func(a, b, s float64) float64 { return 3*a*s*(1-s)*(1-s) + 3*b*s*s*(1-s) + s*s*s }
	lo, hi := 0.0, 1.0
	for i := 0; i < 24; i++ {
		mid := (lo + hi) / 2
		if at(float64(x1), float64(x2), mid) < float64(t) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return float32(at(float64(y1), float64(y2), (lo+hi)/2))
}

// toasts draws s.Toasts over the window, and asks core to close those
// whose time is up ("toast.close:<ID>").
func (th *Theme) toasts(gtx C, s *Scene) {
	ts := &th.toast
	now := gtx.Now
	if ts.life == nil {
		ts.life = map[string]*toastLife{}
	}
	dt := now.Sub(ts.last)
	if ts.last.IsZero() || dt < 0 {
		dt = 0
	}
	ts.last = now
	ts.hovered = ts.hover.Update(gtx.Source)
	live := make([]*toastLife, 0, len(s.Toasts))
	seen := map[string]bool{}
	for i := len(s.Toasts) - 1; i >= 0; i-- {
		t := s.Toasts[i]
		seen[t.ID] = true
		l := ts.life[t.ID]
		if l == nil {
			l = &toastLife{came: now, at: now}
			if !s.Live {
				// A mockup's toasts are already in.
				l.came = now.Add(-toastMove)
			}
			ts.life[t.ID] = l
		}
		l.toast = t
		if !ts.hovered {
			l.up += dt
		}
		if !t.Sticky && !l.asked && l.up >= toastTimeout {
			l.asked = true
			th.push(Action{ID: "toast.close:" + t.ID, Kind: "click"})
		}
		live = append(live, l)
	}
	for id, l := range ts.life {
		if !seen[id] {
			delete(ts.life, id)
			l.went = now
			if s.Live {
				ts.leaving = append(ts.leaving, l)
			}
		}
	}
	kept := ts.leaving[:0]
	for _, l := range ts.leaving {
		if now.Sub(l.went) < toastMove {
			kept = append(kept, l)
		}
	}
	ts.leaving = kept
	if len(live) == 0 && len(kept) == 0 {
		ts.hovered = false
		return
	}
	win := gtx.Constraints.Max
	inset := dp(gtx, 32)
	w := min(dp(gtx, toastW), win.X-2*inset)
	defer op.Offset(image.Pt(win.X-inset-w, inset+dp(gtx, topBarH))).Push(gtx.Ops).Pop()

	// Each toast's content at its own height, then its place.
	calls := make([]op.CallOp, len(live))
	heights := make([]float32, len(live))
	for i, l := range live {
		m := op.Record(gtx.Ops)
		heights[i] = float32(th.toastBody(gtx, l.toast, w).Size.Y)
		calls[i] = m.Stop()
	}
	gap := float32(dp(gtx, 12))
	moving := false
	var newer float32
	for i, l := range live {
		p := toastPlace{y: newer + gap*float32(i), scale: 1, h: heights[i], content: 1, alpha: 1}
		if !ts.hovered && i > 0 {
			k := max(0, 1-0.1*float32(i))
			p = toastPlace{y: gap*float32(i) + (1-k)*heights[0], scale: k, h: heights[0], content: 0, alpha: 1}
		}
		if i >= toastLimit {
			p.alpha = 0
		}
		newer += heights[i]
		switch {
		case !l.placed:
			l.from, l.to, l.placed = p, p, true
		case p != l.to:
			l.from, l.to, l.at = l.now(now), p, now
		}
		moving = moving || now.Sub(l.at) < toastMove || now.Sub(l.came) < toastMove
	}
	// Back to front, so the newest is on top.
	for _, l := range kept {
		th.drawToast(gtx, l, w, l.now(l.went), float32(w+inset)*toastEase(since(now, l.went, toastMove)), 1-since(now, l.went, toastMove), op.CallOp{}, false)
	}
	for i := len(live) - 1; i >= 0; i-- {
		l := live[i]
		slide := float32(w+inset) * (1 - toastEase(since(now, l.came, toastMove)))
		th.drawToast(gtx, l, w, l.now(now), slide, 1, calls[i], true)
	}
	// The pointer anywhere on the stack spreads it, and holds the timers.
	var reach float32
	for i, l := range live {
		if i < toastLimit {
			pl := l.now(now)
			reach = max(reach, pl.y+pl.h*pl.scale)
		}
	}
	func() {
		defer clip.Rect{Min: image.Pt(0, -dp(gtx, 6)), Max: image.Pt(w+dp(gtx, 6), int(reach+gap))}.Push(gtx.Ops).Pop()
		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		ts.hover.Add(gtx.Ops)
	}()
	switch {
	case moving || len(kept) > 0:
		gtx.Execute(op.InvalidateCmd{})
	case !ts.hovered:
		next := toastTimeout
		for _, l := range live {
			if !l.toast.Sticky && !l.asked {
				next = min(next, toastTimeout-l.up)
			}
		}
		gtx.Execute(op.InvalidateCmd{At: now.Add(max(next, 0))})
	}
}

// drawToast draws l at place p, slid x to the right and faded to fade,
// with content; a toast that's leaving (not live) lays its content out
// afresh from how it last looked, and takes no input.
func (th *Theme) drawToast(gtx C, l *toastLife, w int, p toastPlace, x, fade float32, content op.CallOp, live bool) {
	fade *= p.alpha
	if fade <= 0 {
		return
	}
	if fade < 1 {
		defer paint.PushOpacity(gtx.Ops, fade).Pop()
	}
	h := int(p.h + 0.5)
	tr := f32.Affine2D{}.Scale(f32.Pt(float32(w)/2, 0), f32.Pt(p.scale, p.scale)).Offset(f32.Pt(x, p.y))
	defer op.Affine(tr).Push(gtx.Ops).Pop()
	r := image.Rect(0, 0, w, h)
	rad, px := dp(gtx, 8), max(1, dp(gtx, 1))
	shadow(gtx, r, rad, []boxShadow{{0, 20, 25, -5, alpha(rgb(0), 0x40)}, {0, 8, 10, -6, alpha(rgb(0), 0x40)}})
	edge := alpha(rgb(0x14141d), 0x1a)
	if dark {
		edge = alpha(rgb(0xf5f5f5), 0x1a)
	}
	fill(gtx, P.Popover, r, rad)
	strokeRect(gtx, over(edge, P.Popover), r, rad, float32(px))
	if live {
		// The toast takes the pointer, under its own controls, so what's
		// behind it doesn't.
		func() {
			defer clip.Rect(r).Push(gtx.Ops).Pop()
			event.Op(gtx.Ops, l)
		}()
	} else {
		m := op.Record(gtx.Ops)
		th.toastBody(gtx, l.toast, w)
		content = m.Stop()
	}
	if p.content > 0 {
		func() {
			defer clip.UniformRRect(r, rad).Push(gtx.Ops).Pop()
			if p.content < 1 {
				defer paint.PushOpacity(gtx.Ops, p.content).Pop()
			}
			content.Add(gtx.Ops)
		}()
	}
	th.toastClose(gtx, "toast.close:"+l.toast.ID, w, live)
}

// toastClose is the close orb: 24 round, 6 out of the toast's padding box
// at its top right, popover with a border 60% edge and shadow-sm, and the
// 12dp x, muted, or foreground on hover.
func (th *Theme) toastClose(gtx C, id string, w int, live bool) {
	hovered := false
	if live {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	px := max(1, dp(gtx, 1))
	s := dp(gtx, 24)
	at := image.Pt(w-px-s+dp(gtx, 6), px-dp(gtx, 6))
	defer op.Offset(at).Push(gtx.Ops).Pop()
	r := image.Rect(0, 0, s, s)
	// Popover at 92% over the toast's own popover, or all of it on hover,
	// comes to the popover either way; hover turns the x foreground.
	bg, col := P.Popover, P.MutedForeground
	if hovered {
		col = P.Foreground
	}
	shadow(gtx, r, s/2, []boxShadow{{0, 1, 3, 0, alpha(rgb(0), 0x1a)}, {0, 1, 2, -1, alpha(rgb(0), 0x1a)}})
	fill(gtx, bg, r, s/2)
	strokeRect(gtx, over(pct(P.Border, 60), bg), r, s/2, float32(px))
	// lucide's x, 12dp, stroked 2.25.
	u := float32(dp(gtx, 12)) / 24
	o := float32(s-dp(gtx, 12)) / 2
	pt := func(x, y float32) f32.Point { return f32.Pt(o+x*u, o+y*u) }
	polyline(gtx, col, 2.25*u, pt(18, 6), pt(6, 18))
	polyline(gtx, col, 2.25*u, pt(6, 6), pt(18, 18))
	if live {
		th.hit(gtx, id, r.Max)
	}
}

// toastBody lays out t's content, w wide (toast.tsx ToastBodyContent):
// padded 12 14, 24 at the end beside its trailing controls (an error's
// copy button); the title row, its type's icon in a 16 by 20 box 8 before
// the 14/20 medium title; 2 under it the 14/20 muted description, an
// error's cut to four lines from 180 characters, or the toggle for its
// details, which open 8 under it.
func (th *Theme) toastBody(gtx C, t Toast, w int) D {
	px := max(1, dp(gtx, 1))
	copyErr := t.Type == ToastError && t.Description != ""
	end := dp(gtx, 14)
	trail := 0
	if copyErr {
		end, trail = dp(gtx, 24), dp(gtx, 20+6)
	}
	bw := w - 2*px - dp(gtx, 14) - end - trail
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(bw, 0), Max: image.Pt(bw, 1<<20)}
	bd := Col(g,
		rigid(func(gtx C) D {
			titleEnd := 0
			if !copyErr {
				titleEnd = dp(gtx, 24)
			}
			lead := 0
			if icon := th.toastIcon(gtx, t.Type); icon.Size.X > 0 {
				lead = icon.Size.X + dp(gtx, 8)
			}
			tg := gtx
			tg.Constraints = layout.Constraints{Max: image.Pt(max(0, gtx.Constraints.Max.X-lead-titleEnd), 1<<20)}
			defer op.Offset(image.Pt(lead, 0)).Push(gtx.Ops).Pop()
			d := th.cssText(tg, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.PopoverForeground}, t.Title)
			return D{Size: image.Pt(gtx.Constraints.Max.X, d.Size.Y)}
		}),
		rigid(func(gtx C) D {
			if t.Description == "" {
				return D{}
			}
			return pad(2, 0, 0, 0).Layout(gtx, func(gtx C) D { return th.toastDescription(gtx, t) })
		}),
	)
	body := m.Stop()
	h := bd.Size.Y + 2*dp(gtx, 12) + 2*px
	defer op.Offset(image.Pt(px+dp(gtx, 14), 0)).Push(gtx.Ops).Pop()
	func() {
		defer op.Offset(image.Pt(0, px+dp(gtx, 12))).Push(gtx.Ops).Pop()
		body.Add(gtx.Ops)
	}()
	if copyErr {
		th.toastCopy(gtx, t, image.Pt(bw+dp(gtx, 6), (h-dp(gtx, 20))/2))
	}
	return D{Size: image.Pt(w, h)}
}

// toastIcon draws a toast type's icon in its 16 by 20 box: circle-alert in
// --destructive, info in --info, circle-check in --success,
// triangle-alert in --warning, or T3's spinner at 80%.
func (th *Theme) toastIcon(gtx C, k ToastType) D {
	name, col := "", P.Foreground
	switch k {
	case ToastError:
		name, col = "circle-alert", P.Destructive
	case ToastInfo:
		name, col = "info", P.Info
	case ToastSuccess:
		name, col = "circle-check", P.Success
	case ToastWarning:
		name, col = "triangle-alert", P.Warning
	case ToastLoading:
		col = over(pct(P.Foreground, 80), P.Popover)
	default:
		return D{}
	}
	defer op.Offset(image.Pt(0, dp(gtx, 2))).Push(gtx.Ops).Pop()
	if k == ToastLoading {
		T3Spinner(col, 16, float64(gtx.Now.UnixMilli()%60000)/1000)(gtx)
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(33 * time.Millisecond)})
	} else {
		LucideW(name, col, 16)(gtx)
	}
	return D{Size: image.Pt(dp(gtx, 16), dp(gtx, 20))}
}

// toastDescription is t's description; with details, it's their toggle
// ("toast.toggle:<ID>"): the text and a 14dp chevron in a row that's muted
// 40% under the pointer, the text underlined then, and the details below
// it while open, at most 160 tall.
func (th *Theme) toastDescription(gtx C, t Toast) D {
	muted := over(P.MutedForeground, P.Popover)
	desc := Txt{Size: 14, LineH: 20.0 / 14, Color: muted}
	if t.Type == ToastError && len([]rune(t.Description)) >= 180 {
		desc.MaxLines = 4
	}
	if len(t.Details) == 0 {
		return th.cssText(gtx, desc, t.Description)
	}
	id := "toast.toggle:" + t.ID
	c, _ := th.events(gtx, id)
	w := gtx.Constraints.Max.X
	chev := dp(gtx, 14)
	m := op.Record(gtx.Ops)
	tg := gtx
	tg.Constraints = layout.Constraints{Max: image.Pt(w-chev-dp(gtx, 6), 1<<20)}
	td := th.cssText(tg, desc, t.Description)
	call := m.Stop()
	row := image.Pt(w, td.Size.Y)
	if c.Hovered() {
		fill(gtx, over(pct(P.Muted, 40), P.Popover), image.Rectangle{Max: row}, dp(gtx, 4))
		y := td.Size.Y - td.Baseline + dp(gtx, 2)
		fill(gtx, over(pct(P.MutedForeground, 60), P.Popover), image.Rect(0, y, td.Size.X, y+max(1, dp(gtx, 1))), 0)
	}
	call.Add(gtx.Ops)
	func() {
		// lucide's chevron, 14 in a 20 tall box, stroked 2.25 at 80%.
		u := float32(chev) / 24
		defer op.Offset(image.Pt(w-chev, (dp(gtx, 20)-chev)/2)).Push(gtx.Ops).Pop()
		pt := func(x, y float32) f32.Point { return f32.Pt(x*u, y*u) }
		col := over(pct(P.MutedForeground, 80), P.Popover)
		if t.Open {
			polyline(gtx, col, 2.25*u, pt(18, 15), pt(12, 9), pt(6, 15))
		} else {
			polyline(gtx, col, 2.25*u, pt(6, 9), pt(12, 15), pt(18, 9))
		}
	}()
	th.hit(gtx, id, row)
	tip := "Show " + t.Folded
	if t.Open {
		tip = "Hide " + t.Folded
	}
	th.tipAt(gtx, id, tip, row, true, 0)
	if !t.Open {
		return D{Size: row}
	}
	m = op.Record(gtx.Ops)
	dg := gtx
	dg.Constraints = layout.Constraints{Min: image.Pt(w-dp(gtx, 2), 0), Max: image.Pt(w-dp(gtx, 2), 1<<20)}
	var kids []layout.FlexChild
	for i, d := range t.Details {
		last := i == len(t.Details)-1
		kids = append(kids, rigid(func(gtx C) D {
			dd := Col(gtx,
				rigid(func(gtx C) D {
					return th.cssText(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.Foreground}, d.Title)
				}),
				rigid(Spacer(0, 2)),
				rigid(func(gtx C) D {
					return th.cssText(gtx, Txt{Size: 10, LineH: 14.0 / 10, Color: over(pct(P.MutedForeground, 75), P.Popover)}, d.Note)
				}),
			)
			if last {
				return dd
			}
			hline(gtx, over(pct(P.Border, 50), P.Popover), dd.Size.Y+dp(gtx, 8), 0, gtx.Constraints.Max.X)
			return D{Size: image.Pt(dd.Size.X, dd.Size.Y+dp(gtx, 8)+max(1, dp(gtx, 1))+dp(gtx, 10))}
		}))
	}
	list := Col(dg, kids...)
	listCall := m.Stop()
	h := min(list.Size.Y, dp(gtx, 160))
	func() {
		defer op.Offset(image.Pt(0, row.Y+dp(gtx, 8))).Push(gtx.Ops).Pop()
		defer clip.Rect{Max: image.Pt(w, h)}.Push(gtx.Ops).Pop()
		listCall.Add(gtx.Ops)
	}()
	return D{Size: image.Pt(w, row.Y+dp(gtx, 8)+h)}
}

// toastCopy is an error toast's copy button at at: 20 square, rounded 6,
// the 12dp copy icon in muted 80% (full on hover), the check in --success
// once copied; "Copy error" and "Copied error" as its tooltip.
func (th *Theme) toastCopy(gtx C, t Toast, at image.Point) {
	id := "copy:toast." + t.ID
	th.offerCopy(id, t.Description)
	c, _ := th.events(gtx, id)
	defer op.Offset(at).Push(gtx.Ops).Pop()
	s := dp(gtx, 20)
	icon, col, tip := "copy", over(pct(P.MutedForeground, 80), P.Popover), "Copy error"
	if c.Hovered() {
		col = P.MutedForeground
	}
	if th.copiedWithin(gtx, id, 2*time.Second) {
		icon, col, tip = "check", P.Success, "Copied error"
	}
	func() {
		defer op.Offset(image.Pt((s-dp(gtx, 12))/2, (s-dp(gtx, 12))/2)).Push(gtx.Ops).Pop()
		LucideW(icon, col, 12)(gtx)
	}()
	th.hit(gtx, id, image.Pt(s, s))
	th.tipAt(gtx, id, tip, image.Pt(s, s), true, 0)
}
