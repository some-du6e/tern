package ui

import (
	"bytes"
	"image"
	"image/color"
	"regexp"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/draw"
)

// The messages (TimelineUserMessageRow.tsx, TimelineAssistantRows.tsx;
// docs/ui/06-timeline.md §2.3, §2.4, §3.3, §3.4): the user's in a bubble
// at the lane's right, with its pictures above it and its time and copy
// button under it; the assistant's as markdown across the lane, with the
// response's copy button and time under its last message.

// Picture is a sent image, scaled once for the timeline: 384px tall at
// most (the lone image's 192dp at 2x), its shape kept.
type Picture struct {
	Name string
	op   paint.ImageOp
	size image.Point
	ok   bool
}

// NewPicture decodes an image file's bytes for the timeline. A picture it
// can't decode shows as its name.
func NewPicture(data []byte) Picture {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Picture{}
	}
	b := src.Bounds()
	if b.Empty() {
		return Picture{}
	}
	h := min(b.Dy(), 384)
	w := max(1, b.Dx()*h/b.Dy())
	if w > 1200 {
		w, h = 1200, max(1, b.Dy()*1200/b.Dx())
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return Picture{op: paint.NewImageOp(dst), size: image.Pt(w, h), ok: true}
}

// cover draws p filling r, cropped to its middle as object-cover does.
func (p Picture) cover(gtx C, r image.Rectangle) {
	if !p.ok || r.Empty() {
		return
	}
	k := max(float32(r.Dx())/float32(p.size.X), float32(r.Dy())/float32(p.size.Y))
	w, h := float32(p.size.X)*k, float32(p.size.Y)*k
	at := f32.Pt(float32(r.Min.X)+(float32(r.Dx())-w)/2, float32(r.Min.Y)+(float32(r.Dy())-h)/2)
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	defer op.Affine(f32.AffineId().Scale(f32.Point{}, f32.Pt(k, k)).Offset(at)).Push(gtx.Ops).Pop()
	p.op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// reveal is the opacity of something that fades in over 200ms while on is
// set and out when it isn't (transition-opacity duration-200). Like a CSS
// transition, it starts where it should be the first time it's drawn.
func (th *Theme) reveal(gtx C, id string, on bool) float32 {
	if th.tl.reveals == nil {
		th.tl.reveals = map[string]revealState{}
	}
	s, ok := th.tl.reveals[id]
	if !ok || s.on != on {
		from := float32(0)
		if ok {
			from = s.at(gtx.Now)
		} else if on {
			from = 1
		}
		s = revealState{on: on, since: gtx.Now, from: from}
		th.tl.reveals[id] = s
	}
	a := s.at(gtx.Now)
	if (on && a < 1) || (!on && a > 0) {
		gtx.Execute(op.InvalidateCmd{})
	}
	return a
}

// revealState is a fade in flight.
type revealState struct {
	on    bool
	since time.Time
	from  float32
}

func (s revealState) at(now time.Time) float32 {
	t := easeStandard(clamp01(float32(now.Sub(s.since)) / float32(200*time.Millisecond)))
	if s.on {
		return s.from + (1-s.from)*t
	}
	return s.from * (1 - t)
}

// easeStandard is cubic-bezier(0.4, 0, 0.2, 1), T3's default easing.
func easeStandard(x float32) float32 { return cubicEase(x, 0.4, 0.2) }

// cubicEase is cubic-bezier(x1, 0, x2, 1) at x, solved for x by bisection.
func cubicEase(x, x1, x2 float32) float32 {
	bez := func(t, a, b float32) float32 { return 3*a*t*(1-t)*(1-t) + 3*b*t*t*(1-t) + t*t*t }
	lo, hi := float32(0), float32(1)
	for range 20 {
		mid := (lo + hi) / 2
		if bez(mid, x1, x2) < x {
			lo = mid
		} else {
			hi = mid
		}
	}
	return bez((lo+hi)/2, 0, 1)
}

// userRow is a user's message: what goes over the bubble, the bubble, its
// status pill and its actions, right-aligned 4dp apart.
func (th *Theme) userRow(gtx C, r *TimelineRow, hot bool) int {
	m := r.Message
	w := gtx.Constraints.Max.X
	maxW := w * 80 / 100
	gap := dp(gtx, 4)
	y := 0
	place := func(rec recording, right int) image.Rectangle {
		at := image.Pt(w-right-rec.Size.X, y)
		off := op.Offset(at).Push(gtx.Ops)
		rec.call.Add(gtx.Ops)
		off.Pop()
		y += rec.Size.Y + gap
		return image.Rectangle{Min: at, Max: at.Add(rec.Size)}
	}
	if m.SentByT3 {
		place(th.recorded(gtx, func(gtx C) D {
			return th.textLine(gtx, Txt{Size: 11, Color: over(faded(P.MutedForeground, 0xb3), P.Background)}, 16, "Sent by T3 Code")
		}), dp(gtx, 4))
	}
	if m.Steer != SteerNone {
		label := "Steer"
		if m.SteerUnread {
			label = "Steer · waiting to be read"
		}
		place(th.recorded(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(LucideW("redo-2", P.MutedForeground, 12)),
				layout.Rigid(Spacer(4, 0)),
				layout.Rigid(func(gtx C) D { return th.textLine(gtx, Txt{Size: 12, Color: P.MutedForeground}, 12, label) }),
			)
		}), dp(gtx, 4))
	}
	words := m.Text != ""
	var body image.Rectangle // the bubble, or the media alone
	if len(m.Pictures) > 0 {
		body = place(th.recorded(within(gtx, maxW), func(gtx C) D { return th.media(gtx, m, words) }), 0)
	}
	switch {
	case !words && len(m.Pictures) == 0 && len(m.Files) > 0:
		// Files sent without words are cards, not a bubble.
		cards := th.recorded(within(gtx, maxW), func(gtx C) D {
			y := 0
			for i, f := range m.Files {
				if i > 0 {
					y += gap
				}
				off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
				y += th.fileCard(gtx, f).Size.Y
				off.Pop()
			}
			return D{Size: image.Pt(min(gtx.Constraints.Max.X, dp(gtx, 256)), y)}
		})
		body = place(cards, 0)
	case words || len(m.Files) > 0:
		body = place(th.recorded(within(gtx, maxW), func(gtx C) D { return th.bubble(gtx, r, len(m.Pictures) > 0) }), 0)
	}
	if m.Status != "" {
		place(th.recorded(gtx, func(gtx C) D { return th.statusPill(gtx, m.Status) }), dp(gtx, 4))
	}
	// The actions: the time, then copy and the prompt's link, 2dp apart;
	// they fade in with the row's hover. A message another of yours
	// follows floats them 8dp left of the bubble, so the two sit 4dp
	// apart; a lane under 37rem keeps them on their own line
	// (TimelineUserMessageRow.tsx:788-840, 1038-1056).
	float := m.GroupedNext && w >= dp(gtx, 592)
	a := th.reveal(gtx, r.ID+":actions", hot)
	actions := th.recorded(gtx, func(gtx C) D {
		cid := "copy:" + r.ID
		th.copyTarget(cid, StripChips(m.Text))
		kids := []layout.FlexChild{
			layout.Rigid(th.stampText(r.ID, m.At, 16)),
			layout.Rigid(Spacer(8, 0)),
			layout.Rigid(th.messageCopy(cid, "Copy message", "copy", P.Background)),
		}
		if m.PromptLink != "" {
			lid := "copy.link:" + r.ID
			th.copyTarget(lid, m.PromptLink)
			kids = append(kids, layout.Rigid(Spacer(2, 0)), layout.Rigid(th.messageCopy(lid, "Copy prompt link", "link", P.Background)))
		}
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
	})
	at := image.Pt(w-dp(gtx, 4)-actions.Size.X, y)
	if float {
		at = image.Pt(body.Min.X-dp(gtx, 8)-actions.Size.X, body.Min.Y+(body.Dy()-actions.Size.Y)/2)
	}
	if a > 0 {
		off := op.Offset(at).Push(gtx.Ops)
		fade := paint.PushOpacity(gtx.Ops, a)
		actions.call.Add(gtx.Ops)
		fade.Pop()
		off.Pop()
	}
	if float {
		return y - gap
	}
	return y + dp(gtx, 24)
}

// messageCopy is MessageCopyButton: a 24dp ghost button with a 12dp icon,
// label in its tooltip; once clicked, a check in --primary for a second,
// the button dimmed, and "Copied!" above it.
func (th *Theme) messageCopy(id, label, icon string, on color.NRGBA) W {
	return func(gtx C) D {
		b := Button{Variant: BtnGhostMuted, Size: BtnSizeXs, Icon: icon, IconSize: 12, On: on}
		copied := th.copiedFor(gtx, id, 1000)
		if !copied {
			d := th.T3Button(id, b)(gtx)
			th.tipAt(gtx, id, label, d.Size, true, 0)
			return d
		}
		size := image.Pt(dp(gtx, 24), dp(gtx, 24))
		ic := dp(gtx, 12)
		off := op.Offset(image.Pt((size.X-ic)/2, (size.Y-ic)/2)).Push(gtx.Ops)
		fade := paint.PushOpacity(gtx.Ops, 0.64) // disabled while it says so
		LucideW("check", P.Primary, 12)(gtx)
		fade.Pop()
		off.Pop()
		th.copiedToast(gtx, size)
		return D{Size: size}
	}
}

// copiedToast is the anchored "Copied!" toast 4dp above an anchor of size,
// in the tooltip's look.
func (th *Theme) copiedToast(gtx C, size image.Point) {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 256), dp(gtx, 100))}
	td := th.tooltip(g, "Copied!")
	call := m.Stop()
	m2 := op.Record(gtx.Ops)
	op.Offset(image.Pt((size.X-td.Size.X)/2, -td.Size.Y-dp(gtx, 4))).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
}

// statusPill is a user message's status other than completed: 10/14
// medium destructive in a pill at destructive 8%, its border at 25%.
func (th *Theme) statusPill(gtx C, status string) D {
	m := op.Record(gtx.Ops)
	d := th.textLine(gtx, Txt{Size: 10, Font: UI(font.Medium), Color: P.Destructive}, 14, status)
	call := m.Stop()
	px := max(1, dp(gtx, 1))
	size := image.Pt(d.Size.X+2*dp(gtx, 6)+2*px, d.Size.Y+2*dp(gtx, 2)+2*px)
	r := image.Rectangle{Max: size}
	fill(gtx, over(faded(P.Destructive, 0x14), P.Background), r, size.Y/2)
	strokeRect(gtx, over(faded(P.Destructive, 0x40), P.Background), r, size.Y/2, float32(px))
	off := op.Offset(image.Pt(dp(gtx, 6)+px, dp(gtx, 2)+px)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return D{Size: size}
}

// fileLabel is T3's userMessageFileLabel.
func fileLabel(f File) string {
	if f.Pasted {
		return "Pasted text"
	}
	return f.Name
}

// fileCard is UserFileCard (TimelineUserMessageRow.tsx:104-178): 256dp of
// --message-surface, radius 16, padded 8 by 8 4 by 8; a 36dp box at
// background/60 holding the file's icon, then its name medium 14/20 over
// "PDF · 2.40 MB" 12/16 muted, then its download button.
func (th *Theme) fileCard(gtx C, f File) D {
	w, h := min(gtx.Constraints.Max.X, dp(gtx, 256)), dp(gtx, 52)
	surface := P.MessageSurface
	fill(gtx, surface, image.Rect(0, 0, w, h), dp(gtx, 16))
	in := dp(gtx, 8)
	box := image.Rect(in, in, in+dp(gtx, 36), in+dp(gtx, 36))
	boxBg := over(faded(P.Background, 0x99), surface)
	fill(gtx, boxBg, box, dp(gtx, 8))
	io := op.Offset(box.Min.Add(image.Pt((box.Dx()-dp(gtx, 16))/2, (box.Dy()-dp(gtx, 16))/2))).Push(gtx.Ops)
	fileIconW(f.Name, 16, boxBg)(gtx)
	io.Pop()
	label := fileLabel(f)
	meta := FormatBytes(f.Size)
	if !f.Pasted {
		ext := "File"
		if m := fileExt.FindStringSubmatch(f.Name); m != nil {
			ext = strings.ToUpper(m[1])
		}
		meta = ext + " · " + meta
	}
	tx := box.Max.X + dp(gtx, 12)
	tw := w - tx - dp(gtx, 4+24+4)
	to := op.Offset(image.Pt(tx, in)).Push(gtx.Ops)
	th.textLine(within(gtx, tw), Txt{Size: 14, Font: UI(font.Medium), Color: P.MessageForeground}, 20, label)
	mo := op.Offset(image.Pt(0, dp(gtx, 20))).Push(gtx.Ops)
	th.textLine(within(gtx, tw), Txt{Size: 12, Color: P.MutedForeground}, 16, meta)
	mo.Pop()
	to.Pop()
	if !f.Unknown {
		id := "file.download:" + f.ID
		bo := op.Offset(image.Pt(w-dp(gtx, 4+24), (h-dp(gtx, 24))/2)).Push(gtx.Ops)
		d := th.T3Button(id, Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "download", On: surface})(gtx)
		th.tipAt(gtx, id, "Download "+label, d.Size, true, 0)
		bo.Pop()
	}
	return D{Size: image.Pt(w, h)}
}

// fileExt is a file name's extension, as a file card names its kind.
var fileExt = regexp.MustCompile(`(?i)\.([a-z0-9]{1,8})$`)

// fileRows are a message's files inside its bubble, over its words, 4dp
// apart: each its icon, its label (a pasted text's size under it) and an
// eye, then its download button (TimelineUserMessageRow.tsx:954-1023).
func (th *Theme) fileRows(gtx C, files []File) D {
	w := gtx.Constraints.Max.X
	y := 0
	for i, f := range files {
		if i > 0 {
			y += dp(gtx, 4)
		}
		label := fileLabel(f)
		lh := dp(gtx, 20)
		if f.Pasted {
			lh += dp(gtx, 16)
		}
		rowH := lh + 2*dp(gtx, 4) // py-1
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		io := op.Offset(image.Pt(0, (rowH-dp(gtx, 16))/2)).Push(gtx.Ops)
		fileIconW(f.Name, 16, P.MessageSurface)(gtx)
		io.Pop()
		tx := dp(gtx, 16+8)
		tail := dp(gtx, 8+16)
		if !f.Unknown {
			tail += dp(gtx, 4+24)
		}
		to := op.Offset(image.Pt(tx, dp(gtx, 4))).Push(gtx.Ops)
		d := th.textLine(within(gtx, w-tx-tail), Txt{Size: 14, Color: P.MessageForeground}, 20, label)
		if f.Pasted {
			so := op.Offset(image.Pt(0, dp(gtx, 20))).Push(gtx.Ops)
			th.textLine(within(gtx, w-tx-tail), Txt{Size: 12, Color: P.MutedForeground}, 16, FormatBytes(f.Size))
			so.Pop()
		}
		to.Pop()
		if !f.Unknown {
			eo := op.Offset(image.Pt(tx+d.Size.X+dp(gtx, 8), (rowH-dp(gtx, 16))/2)).Push(gtx.Ops)
			LucideW("eye", P.MessageForeground, 16)(gtx)
			eo.Pop()
			id := "file.download:" + f.ID
			bo := op.Offset(image.Pt(w-dp(gtx, 24), (rowH-dp(gtx, 24))/2)).Push(gtx.Ops)
			bd := th.T3Button(id, Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "download", On: P.MessageSurface})(gtx)
			th.tipAt(gtx, id, "Download "+label, bd.Size, true, 0)
			bo.Pop()
		}
		off.Pop()
		y += rowH
	}
	return D{Size: image.Pt(w, y)}
}

// fileRowsWidth is how wide a message's file rows would be, at most maxW.
func (th *Theme) fileRowsWidth(gtx C, files []File, maxW int) int {
	w := 0
	for _, f := range files {
		lw := th.recorded(within(gtx, maxW), func(gtx C) D {
			return th.textLine(gtx, Txt{Size: 14, Color: P.MessageForeground}, 20, fileLabel(f))
		}).Size.X
		row := dp(gtx, 16+8) + lw + dp(gtx, 8+16)
		if !f.Unknown {
			row += dp(gtx, 4+24)
		}
		w = max(w, row)
	}
	return min(w, maxW)
}

// collapseAt and collapsedH are T3's long message fold: a body 2 × 11rem
// tall or more shows 11rem of itself, fading out over its last 28dp
// (useTallEnoughToFold.ts, CollapsibleUserMessageBody).
const collapseAt, collapsedH, collapseFade = 352, 176, 28

// bubble is the message's text in --message-surface, radius 16, padded
// 12, as wide as its text up to 80% of the lane; the corners next to a
// picture above or another of your messages tighten to 6.
func (th *Theme) bubble(gtx C, r *TimelineRow, mediaAbove bool) D {
	m := r.Message
	maxW := gtx.Constraints.Max.X
	surface := P.MessageSurface
	p := dp(gtx, 12)
	st := mdStyle{color: P.Foreground, on: surface, user: true, owner: r.ID}
	doc := th.parsed(m.Text, st, m.Chips)
	textW := min(maxW-2*p, th.mdNatural(gtx, doc, doc.blocks, st, maxW-2*p))
	// Its files over its words, 8dp above them (mb-2).
	filesH := 0
	var files recording
	if len(m.Files) > 0 {
		textW = max(textW, th.fileRowsWidth(gtx, m.Files, maxW-2*p))
		files = th.recorded(within(gtx, textW), func(gtx C) D { return th.fileRows(gtx, m.Files) })
		filesH = files.Size.Y
		if m.Text != "" {
			filesH += dp(gtx, 8)
		}
	}
	bm := op.Record(gtx.Ops)
	th.tl.blockSeq = 0
	bh := th.mdStack(within(gtx, textW), doc, doc.blocks, st)
	body := bm.Stop()
	full := th.tl.full[r.ID]
	tall := dp(gtx, collapseAt) <= bh
	shown := bh
	if tall && !full {
		shown = dp(gtx, collapsedH)
	}
	h := filesH + shown + 2*p
	var foot recording
	if tall {
		label := "Show full message"
		if full {
			label = "Show less"
		}
		foot = th.recorded(gtx, th.T3Button(r.ID+":full", Button{Variant: BtnGhostMuted, Size: BtnSizeXs, Label: label, On: surface}))
		h += dp(gtx, 6) + foot.Size.Y
		textW = max(textW, foot.Size.X-dp(gtx, 4))
		if th.clicked(r.ID + ":full") {
			th.tl.full[r.ID] = !full
		}
	}
	w := textW + 2*p
	rad, tight := float32(dp(gtx, 16)), float32(dp(gtx, 6))
	tr, br := rad, rad
	if mediaAbove || m.GroupedPrev {
		tr = tight
	}
	if m.GroupedNext {
		br = tight
	}
	rr := clip.RRect{Rect: image.Rect(0, 0, w, h), NW: int(rad), NE: int(tr), SE: int(br), SW: int(rad)}
	paint.FillShape(gtx.Ops, surface, rr.Op(gtx.Ops))
	if len(m.Files) > 0 {
		fo := op.Offset(image.Pt(p, p)).Push(gtx.Ops)
		files.call.Add(gtx.Ops)
		fo.Pop()
	}
	func() {
		defer op.Offset(image.Pt(p, p+filesH)).Push(gtx.Ops).Pop()
		defer clip.Rect{Max: image.Pt(textW, shown)}.Push(gtx.Ops).Pop()
		body.Add(gtx.Ops)
		if tall && !full {
			// mask: linear-gradient(#000 calc(100% - 28px), transparent)
			fade := dp(gtx, collapseFade)
			fadeDown(gtx, image.Rect(0, shown-fade, textW, shown), surface)
		}
	}()
	if tall {
		// The footer, its button's right edge on the text's (justify-end;
		// its -ml-1 only lets it reach 4dp past the text's left).
		off := op.Offset(image.Pt(w-p-foot.Size.X, p+filesH+shown+dp(gtx, 6))).Push(gtx.Ops)
		foot.call.Add(gtx.Ops)
		off.Pop()
	}
	return D{Size: image.Pt(w, h)}
}

// fadeDown paints col over r from transparent at the top to opaque at the
// bottom: content fading into the surface under it.
func fadeDown(gtx C, r image.Rectangle, col color.NRGBA) {
	steps := max(1, r.Dy()/max(1, dp(gtx, 2)))
	for i := 0; i < steps; i++ {
		a := (float32(i) + 0.5) / float32(steps)
		y0 := r.Min.Y + r.Dy()*i/steps
		y1 := r.Min.Y + r.Dy()*(i+1)/steps
		fill(gtx, alpha(col, uint8(a*255+0.5)), image.Rect(r.Min.X, y0, r.Max.X, y1), 0)
	}
}

// media is a message's pictures over its bubble: one keeps its shape, 192
// tall; several share a 320 wide collage split by hairlines.
func (th *Theme) media(gtx C, m *Message, words bool) D {
	maxW := gtx.Constraints.Max.X
	rad, tight := dp(gtx, 16), dp(gtx, 6)
	tr, br := rad, rad
	if m.GroupedPrev {
		tr = tight
	}
	if words || m.GroupedNext {
		br = tight
	}
	edge := alpha(rgb(0), 0x17)
	if dark {
		edge = alpha(rgb(0xffffff), 0x1f)
	}
	edge = over(edge, P.Background)
	pics := m.Pictures
	var size image.Point
	var cells []image.Rectangle
	gapPx := max(1, dp(gtx, 1))
	switch {
	case len(pics) == 1:
		h := dp(gtx, 192)
		pw := dp(gtx, 256)
		if p := pics[0]; p.ok {
			pw = p.size.X * h / max(1, p.size.Y)
		}
		size = image.Pt(max(dp(gtx, 96), min(maxW, pw)), h)
		cells = []image.Rectangle{{Max: size}}
	default:
		w := min(maxW, dp(gtx, 320))
		half := (w - gapPx) / 2
		switch len(pics) {
		case 2:
			h := dp(gtx, 192)
			size = image.Pt(w, h)
			cells = []image.Rectangle{image.Rect(0, 0, half, h), image.Rect(half+gapPx, 0, w, h)}
		case 3:
			// 3fr 2fr; the first spans both rows of 95.5dp.
			c0 := (w - gapPx) * 3 / 5
			row := dp(gtx, 95.5)
			size = image.Pt(w, 2*row+gapPx)
			cells = []image.Rectangle{image.Rect(0, 0, c0, size.Y), image.Rect(c0+gapPx, 0, w, row), image.Rect(c0+gapPx, row+gapPx, w, size.Y)}
		default:
			row := dp(gtx, 96)
			rows := (len(pics) + 1) / 2
			size = image.Pt(w, rows*row+(rows-1)*gapPx)
			for i := range pics {
				y := i / 2 * (row + gapPx)
				x0, x1 := 0, half
				if i%2 == 1 {
					x0, x1 = half+gapPx, w
				}
				if len(pics)%2 == 1 && i == len(pics)-1 {
					x0, x1 = 0, w // an odd last one spans both columns
				}
				cells = append(cells, image.Rect(x0, y, x1, y+row))
			}
		}
	}
	shape := clip.RRect{Rect: image.Rectangle{Max: size}, NW: rad, NE: tr, SE: br, SW: rad}
	func() {
		defer shape.Push(gtx.Ops).Pop()
		fill(gtx, edge, image.Rectangle{Max: size}, 0)
		for i, c := range cells {
			fill(gtx, over(faded(P.Background, 0xb3), P.Background), c, 0)
			if pics[i].ok {
				pics[i].cover(gtx, c)
				continue
			}
			lg := within(gtx, c.Dx()-dp(gtx, 16))
			name := th.recorded(lg, func(gtx C) D {
				return th.textLine(gtx, Txt{Size: 11, Color: over(faded(P.MutedForeground, 0xb3), P.Background)}, 16, pics[i].Name)
			})
			off := op.Offset(c.Min.Add(image.Pt((c.Dx()-name.Size.X)/2, (c.Dy()-name.Size.Y)/2))).Push(gtx.Ops)
			name.call.Add(gtx.Ops)
			off.Pop()
		}
	}()
	// The 1dp ring inside the edge, over the pictures.
	px := float32(max(1, dp(gtx, 1)))
	var ring clip.Path
	ring.Begin(gtx.Ops)
	rrect(&ring, 0, 0, float32(size.X), float32(size.Y), float32(rad), true)
	rrect(&ring, px, px, float32(size.X)-px, float32(size.Y)-px, float32(rad)-px, false)
	paint.FillShape(gtx.Ops, edge, clip.Outline{Path: ring.End()}.Op())
	return D{Size: size}
}

// assistantRow is an assistant's message, padded 4 by 2: its markdown at
// foreground 90%, then its status or its meta row; or the meta row alone,
// moved below the response's tool groups. run lets its shell blocks run.
func (th *Theme) assistantRow(gtx C, r *TimelineRow, hot, run bool) int {
	m := r.Message
	w := gtx.Constraints.Max.X
	px := dp(gtx, 4)
	if r.Kind == RowAssistantMeta {
		off := op.Offset(image.Pt(px, dp(gtx, 2))).Push(gtx.Ops)
		th.assistantMeta(within(gtx, w-2*px), r, 1)
		off.Pop()
		return dp(gtx, 2) + dp(gtx, 24)
	}
	y := dp(gtx, 2)
	st := mdBody()
	st.streaming, st.run, st.owner = m.Streaming, run, r.ID
	off := op.Offset(image.Pt(px, y)).Push(gtx.Ops)
	y += th.markdown(within(gtx, w-2*px), m.Text, st, nil).Size.Y
	off.Pop()
	switch {
	case m.Meta:
		y += dp(gtx, 6)
		mo := op.Offset(image.Pt(px, y)).Push(gtx.Ops)
		th.assistantMeta(within(gtx, w-2*px), r, th.reveal(gtx, r.ID+":meta", hot))
		mo.Pop()
		y += dp(gtx, 24)
	case m.Badge == "interrupted":
		y += dp(gtx, 6)
		bo := op.Offset(image.Pt(px, y)).Push(gtx.Ops)
		d := th.statusBadge(gtx, m.Badge)
		bo.Pop()
		y += d.Size.Y
	}
	return y + dp(gtx, 2)
}

// assistantMeta is the response's meta row: its status, copy, fork, and
// the time of its last update, 8dp apart, at opacity a
// (TimelineAssistantRows.tsx:166-218).
func (th *Theme) assistantMeta(gtx C, r *TimelineRow, a float32) {
	if a <= 0 {
		return
	}
	m := r.Message
	defer paint.PushOpacity(gtx.Ops, a).Pop()
	var kids []layout.FlexChild
	gap := func() { kids = append(kids, layout.Rigid(Spacer(8, 0))) }
	if m.Badge != "" {
		kids = append(kids, layout.Rigid(func(gtx C) D { return th.statusBadge(gtx, m.Badge) }))
		gap()
	}
	if m.Copy {
		cid := "copy:" + r.ID
		th.copyTarget(cid, m.Text)
		kids = append(kids, layout.Rigid(th.messageCopy(cid, "Copy message", "copy", P.Background)))
		gap()
	}
	if m.Fork {
		kids = append(kids, layout.Rigid(th.forkButton(m)))
		gap()
	}
	if !m.Streaming {
		kids = append(kids, layout.Rigid(th.stampText(r.ID, m.At, 24)))
	}
	layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
}

// forkButton is AssistantForkButton: a 24dp ghost button with a 12dp
// git-fork, "Fork from this response" in its tooltip, click target
// "fork:<message>". While the fork is on its way it's disabled and its
// icon pulses (animate-pulse).
func (th *Theme) forkButton(m *Message) W {
	return func(gtx C) D {
		id := "fork:" + m.ID
		b := Button{Variant: BtnGhost, Size: BtnSizeXs, Icon: "git-fork", IconSize: 12, Disabled: m.Forking}
		if !m.Forking {
			d := th.T3Button(id, b)(gtx)
			th.tipAt(gtx, id, "Fork from this response", d.Size, true, 0)
			return d
		}
		defer paint.PushOpacity(gtx.Ops, pulse(gtx.Now)).Pop()
		gtx.Execute(op.InvalidateCmd{})
		return th.T3Button("", b)(gtx)
	}
}

// pulse is animate-pulse's opacity: down to 50% and back every 2s, eased
// with cubic-bezier(0.4, 0, 0.6, 1).
func pulse(now time.Time) float32 {
	t := float32(now.UnixMilli()%2000) / 1000 // 0 to 2, the curve symmetric
	if t > 1 {
		t = 2 - t
	}
	return 1 - 0.5*cubicEase(t, 0.4, 0.6)
}

// statusBadge is AssistantStatusBadge: a mono 10/14 muted pill in a 1dp
// border at border/70.
func (th *Theme) statusBadge(gtx C, status string) D {
	m := op.Record(gtx.Ops)
	d := th.textLine(gtx, Txt{Size: 10, Font: Mono(), Color: P.MutedForeground}, 14, status)
	call := m.Stop()
	px := max(1, dp(gtx, 1))
	size := image.Pt(d.Size.X+2*dp(gtx, 6)+2*px, d.Size.Y+2*dp(gtx, 2)+2*px)
	strokeRect(gtx, over(faded(P.Border, 0xb3), P.Background), image.Rectangle{Max: size}, size.Y/2, float32(px))
	off := op.Offset(image.Pt(dp(gtx, 6)+px, dp(gtx, 2)+px)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return D{Size: size}
}
