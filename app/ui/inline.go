package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// Inline text as CSS lays it out (docs/ui/06-timeline.md §3.2, §3.5):
// runs in different fonts and sizes on shared baselines, in line boxes as
// tall as the block's line-height with the half-leading above and below,
// wrapping between words and breaking inside a word only when it can't
// fit a line by itself (overflow-wrap: anywhere). Inline code is a padded
// box behind its text, cloned on each line it spans; a mark's box opens on
// its first line and closes on its last; links and chips keep their boxes
// for the pointer. A clamped paragraph ends its last line with "…".

// inlineRun is a stretch of text in one style.
type inlineRun struct {
	text      string
	font      font.Font
	size      float32 // the font size in dp
	color     color.NRGBA
	code      bool // inline code: padded 0.1em by 0.35em in a 1dp border, radius 4
	mark      bool // highlighted: on the paragraph's markBg, padded 2dp either end, radius 4
	strike    bool
	underline bool
	italic    bool // Inter has no italic, so Chrome slants the roman: so does this
	link      int  // 1 + the index of its link in the paragraph, or 0
	chip      int  // a picture chip's number, or 0
	brk       bool // a hard line break
	// gap is a fixed space of gap dp, a margin between inline boxes; icon
	// a lucide icon size dp square, centred on the line.
	gap  float32
	icon string
}

// inlineText is one paragraph: its runs, and the block's own font and line
// height, which set where the baselines fall.
type inlineText struct {
	runs  []inlineRun
	font  font.Font
	size  float32 // dp
	lineH float32 // dp
	// chips are the labels of the pictures its chip runs draw.
	chips []string
	// codeBg is inline code's fill, composited on the surface it sits on;
	// markBg a mark's.
	codeBg, markBg color.NRGBA
	// maxLines clamps the paragraph to that many lines (line-clamp), the
	// last ending in "…" in ellipsis; 0 doesn't.
	maxLines int
	ellipsis color.NRGBA
	// indent is where the first line starts, in dp: a task's checkbox
	// pulls it into the list's gutter.
	indent float32
	// align places each line: textStart, textCenter or textEnd.
	align textAlign
}

// textAlign is CSS's text-align.
type textAlign int

const (
	textStart textAlign = iota
	textCenter
	textEnd
)

// linkBox is where a link's text landed, for the pointer.
type linkBox struct {
	link int
	r    image.Rectangle
}

// inlineLayout is a laid-out paragraph, recorded once per width.
type inlineLayout struct {
	call  op.CallOp
	size  image.Point
	links []linkBox
	lines int
}

// inlineKey names a paragraph's layout: the text, the width and the scale.
type inlineKey struct {
	text  *inlineText
	w     int
	scale float32
	dark  bool
}

// fontMetrics are a face's ascent and descent in px at a size.
type fontMetrics struct{ ascent, descent float32 }

type metricKey struct {
	font font.Font
	px   int
}

// piece is one run's text on one line, or a chip.
type piece struct {
	run    *inlineRun
	glyphs []text.Glyph
	text   string // the text the glyphs draw
	x, w   int    // where it starts on its line and its advance, with its padding
	// lpad and rpad are the padding (and inline code's border) before and
	// after the glyphs.
	lpad, rpad int
	ascent     int
	descent    int
	chip       string
}

// metrics is the font's ascent and descent at px, measured once.
func (th *Theme) metrics(f font.Font, px int) fontMetrics {
	k := metricKey{f, px}
	if m, ok := th.tl.metrics[k]; ok {
		return m
	}
	th.Shaper.LayoutString(text.Parameters{Font: f, PxPerEm: fixed.I(px), MaxLines: 1}, "x")
	var m fontMetrics
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		m = fontMetrics{float32(g.Ascent) / 64, float32(g.Descent) / 64}
	}
	if th.tl.metrics == nil {
		th.tl.metrics = map[metricKey]fontMetrics{}
	}
	th.tl.metrics[k] = m
	return m
}

// shapeLine shapes as much of s as fits in maxW px on one line, breaking
// at words unless anywhere is set. It returns the glyphs, how many bytes
// of s they cover, and their advance.
func (th *Theme) shapeLine(f font.Font, px int, s string, maxW int, anywhere bool) (gs []text.Glyph, n, w int) {
	policy := text.WrapWords
	if anywhere {
		policy = text.WrapGraphemes
	}
	th.Shaper.LayoutString(text.Parameters{Font: f, PxPerEm: fixed.I(px), MaxWidth: maxW, MaxLines: 1,
		Truncator: "​", WrapPolicy: policy}, s)
	runes := 0
	var first, end fixed.Int26_6
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		if g.Flags&text.FlagTruncator != 0 {
			continue
		}
		if len(gs) == 0 {
			first = g.X
		}
		runes += int(g.Runes)
		if e := g.X + g.Advance; e > end {
			end = e
		}
		gs = append(gs, g)
	}
	for i := 0; i < runes && n < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[n:])
		n += size
	}
	return gs, n, (end - first).Ceil()
}

// layoutInline lays t out maxW px wide.
func (th *Theme) layoutInline(gtx C, t *inlineText, maxW int) *inlineLayout {
	k := inlineKey{t, maxW, gtx.Metric.PxPerDp, dark}
	if l := th.tl.layouts[k]; l != nil {
		return l
	}
	px := func(dp float32) int { return int(dp*gtx.Metric.PxPerDp + 0.5) }
	var lines [][]piece
	var line []piece
	x := int(math.Round(float64(t.indent * gtx.Metric.PxPerDp)))
	start := x // where the line being filled starts
	newLine := func() {
		lines = append(lines, line)
		line, x, start = nil, 0, 0
	}
	for i := range t.runs {
		r := &t.runs[i]
		switch {
		case r.brk:
			newLine()
			continue
		case r.icon != "" || r.gap > 0:
			w := px(r.gap)
			if r.icon != "" {
				w = px(r.size)
			}
			if x > start && x+w > maxW {
				newLine()
			}
			line = append(line, piece{run: r, x: x, w: w})
			x += w
			continue
		case r.chip > 0:
			label := ImageLabel(r.chip)
			if r.chip <= len(t.chips) {
				label = t.chips[r.chip-1]
			}
			_, _, lw := th.shapeLine(r.font, px(r.size), label, 1<<20, false)
			w := px(r.size) + px(0.2*r.size) + lw
			if x > start && x+w > maxW {
				newLine()
			}
			m := th.metrics(r.font, px(r.size))
			line = append(line, piece{run: r, x: x, w: w, chip: label, ascent: int(m.ascent + 0.5), descent: int(m.descent + 0.5)})
			x += w
			continue
		}
		// Code pads each line's piece; a mark only its first and last.
		pad, ends := 0, false
		switch {
		case r.code:
			pad = px(0.35*r.size) + max(1, px(1))
		case r.mark:
			pad, ends = px(2), true
		}
		rest := r.text
		for first := true; rest != ""; first = false {
			if x == start {
				// Spaces at the start of a wrapped line collapse away.
				rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
				if rest == "" {
					break
				}
			}
			lpad := pad
			if ends && !first {
				lpad = 0
			}
			avail := maxW - x - lpad - pad
			gs, n, w := th.shapeLine(r.font, px(r.size), rest, max(1, avail), false)
			if n == 0 {
				if x > start {
					newLine()
					continue
				}
				// Not even a word fits a whole line: break it anywhere.
				gs, n, w = th.shapeLine(r.font, px(r.size), rest, max(1, avail), true)
				if n == 0 {
					_, n = utf8.DecodeRuneInString(rest)
					gs, _, w = th.shapeLine(r.font, px(r.size), rest[:n], 1<<20, false)
				}
			}
			rpad := pad
			if ends && n < len(rest) {
				rpad = 0
			}
			m := th.metrics(r.font, px(r.size))
			line = append(line, piece{run: r, glyphs: gs, text: rest[:n], x: x, w: lpad + w + rpad, lpad: lpad, rpad: rpad,
				ascent: int(m.ascent + 0.5), descent: int(m.descent + 0.5)})
			x += lpad + w + rpad
			rest = rest[n:]
			if rest != "" {
				newLine()
			}
		}
	}
	if len(line) > 0 || len(lines) == 0 {
		lines = append(lines, line)
	}
	if t.maxLines > 0 && len(lines) > t.maxLines {
		lines = lines[:t.maxLines]
		lines[len(lines)-1] = th.ellipsize(lines[len(lines)-1], t, maxW, px)
	}

	l := &inlineLayout{lines: len(lines)}
	l.call, l.size, l.links = th.paintInline(gtx, t, lines, maxW)
	if th.tl.layouts == nil || len(th.tl.layouts) > 4000 {
		th.tl.layouts = map[inlineKey]*inlineLayout{}
	}
	th.tl.layouts[k] = l
	return l
}

// ellipsize ends a clamped paragraph's last line with "…" in the block's
// font and its ellipsis colour, dropping the characters it has no room
// for, as line-clamp does.
func (th *Theme) ellipsize(line []piece, t *inlineText, maxW int, px func(float32) int) []piece {
	er := &inlineRun{text: "…", font: t.font, size: t.size, color: t.ellipsis}
	gs, _, ew := th.shapeLine(er.font, px(er.size), er.text, 1<<20, false)
	for len(line) > 0 {
		p := &line[len(line)-1]
		s := strings.TrimRightFunc(p.text, unicode.IsSpace)
		if p.x+p.w+ew <= maxW && s == p.text {
			break
		}
		if p.text == "" {
			// A chip, an icon or a gap without room after it.
			line = line[:len(line)-1]
			continue
		}
		r := p.run
		g, n, w := th.shapeLine(r.font, px(r.size), s, max(1, maxW-ew-p.x-p.lpad), true)
		if s == "" || n == 0 {
			line = line[:len(line)-1]
			continue
		}
		// A mark the clamp cuts doesn't close here.
		p.glyphs, p.text, p.rpad, p.w = g, s[:n], 0, p.lpad+w
		break
	}
	x := 0
	if n := len(line); n > 0 {
		x = line[n-1].x + line[n-1].w
	}
	m := th.metrics(er.font, px(er.size))
	return append(line, piece{run: er, glyphs: gs, text: er.text, x: x, w: ew, ascent: int(m.ascent + 0.5), descent: int(m.descent + 0.5)})
}

// paintInline records the laid-out lines' drawing, each line placed by the
// text's alignment within maxW. Its size is the widest line's.
func (th *Theme) paintInline(gtx C, t *inlineText, lines [][]piece, maxW int) (op.CallOp, image.Point, []linkBox) {
	ops := new(op.Ops)
	g := gtx
	g.Ops = ops
	rec := op.Record(ops)
	lineH := t.lineH * gtx.Metric.PxPerDp
	strut := th.metrics(t.font, int(t.size*gtx.Metric.PxPerDp+0.5))
	half := (lineH - (strut.ascent + strut.descent)) / 2
	var links []linkBox
	width := 0
	for i, ln := range lines {
		top := float32(i) * lineH
		base := int(top + half + strut.ascent + 0.5)
		lineW := 0
		for _, p := range ln {
			lineW = max(lineW, p.x+p.w)
		}
		width = max(width, lineW)
		dx := 0
		switch t.align {
		case textCenter:
			dx = (maxW - lineW) / 2
		case textEnd:
			dx = maxW - lineW
		}
		lo := op.Offset(image.Pt(dx, 0)).Push(ops)
		for _, p := range ln {
			r := p.run
			col := r.color
			if r.icon != "" {
				s := int(r.size*gtx.Metric.PxPerDp + 0.5)
				off := op.Offset(image.Pt(p.x, int(top+(lineH-float32(s))/2+0.5))).Push(ops)
				LucideW(r.icon, col, r.size)(g)
				off.Pop()
				continue
			}
			if r.gap > 0 {
				continue
			}
			switch {
			case r.code:
				em := r.size * gtx.Metric.PxPerDp
				padY := 0.1*em + float32(max(1, dp(gtx, 1)))
				box := image.Rect(p.x, base-p.ascent-int(padY+0.5), p.x+p.w, base+p.descent+int(padY+0.5))
				fill(g, t.codeBg, box, dp(gtx, 4))
			case r.mark:
				fill(g, t.markBg, image.Rect(p.x, base-p.ascent, p.x+p.w, base+p.descent), dp(gtx, 4))
			}
			if r.link > 0 {
				links = append(links, linkBox{r.link - 1, image.Rect(dx+p.x, base-p.ascent, dx+p.x+p.w, base+p.descent)})
			}
			if p.chip != "" {
				em := int(r.size*gtx.Metric.PxPerDp + 0.5)
				off := op.Offset(image.Pt(p.x, base-p.ascent+(p.ascent+p.descent-em)/2)).Push(ops)
				LucideW("image", col, r.size)(g)
				off.Pop()
				lx := p.x + em + int(0.2*float32(em)+0.5)
				gs, _, _ := th.shapeLine(r.font, em, p.chip, 1<<20, false)
				th.paintGlyphs(g, gs, image.Pt(lx, base), col, false)
				continue
			}
			th.paintGlyphs(g, p.glyphs, image.Pt(p.x+p.lpad, base), col, r.italic)
			if r.strike {
				em := r.size * gtx.Metric.PxPerDp
				y := base - int(0.27*em+0.5)
				fill(g, col, image.Rect(p.x, y, p.x+p.w, y+max(1, dp(gtx, 1))), 0)
			}
			if r.underline {
				y := base + dp(gtx, 1.5)
				fill(g, col, image.Rect(p.x, y, p.x+p.w, y+max(1, dp(gtx, 1))), 0)
			}
		}
		lo.Pop()
	}
	call := rec.Stop()
	h := int(float32(len(lines))*lineH + 0.5)
	return call, image.Pt(min(width, maxW), h), links
}

// paintGlyphs draws glyphs from one shaped line with the first one's dot
// at dot, slanted a quarter for synthesized italics.
func (th *Theme) paintGlyphs(gtx C, gs []text.Glyph, dot image.Point, col color.NRGBA, italic bool) {
	if len(gs) == 0 {
		return
	}
	// Shape draws relative to the first glyph's dot.
	tr := f32.AffineId().Offset(f32.Pt(float32(dot.X), float32(dot.Y)))
	if italic {
		tr = f32.NewAffine2D(1, -0.25, float32(dot.X), 0, 1, float32(dot.Y))
	}
	st := op.Affine(tr).Push(gtx.Ops)
	cl := clip.Outline{Path: th.Shaper.Shape(gs)}.Op().Push(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	cl.Pop()
	if call := th.Shaper.Bitmaps(gs); call != (op.CallOp{}) {
		call.Add(gtx.Ops)
	}
	st.Pop()
}

// inline draws a laid-out paragraph, then its links: a dotted underline
// while hovered, the address in a tooltip, and click target
// "link:<href>".
func (th *Theme) inline(gtx C, t *inlineText, links []string) D {
	l := th.layoutInline(gtx, t, gtx.Constraints.Max.X)
	l.call.Add(gtx.Ops)
	for i, lb := range l.links {
		if lb.link >= len(links) {
			continue
		}
		lid := "link:" + links[lb.link]
		c, _ := th.events(gtx, lid)
		if c.Hovered() {
			th.dottedUnderline(gtx, lb.r, P.InfoForeground)
		}
		off := op.Offset(lb.r.Min).Push(gtx.Ops)
		th.hit(gtx, lid, lb.r.Size())
		if i == 0 || l.links[i-1].link != lb.link {
			th.tipAt(gtx, lid, links[lb.link], lb.r.Size(), true, 0)
		}
		off.Pop()
	}
	return D{Size: l.size}
}

// textLine draws s on one line, truncated with "…" at the width, its
// content area centred in a line box lineH dp tall as CSS's half-leading
// puts it. It returns the line box.
func (th *Theme) textLine(gtx C, t Txt, lineH float32, s string) D {
	t.MaxLines, t.LineH = 1, 0
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	d := th.Text(g, t, s)
	call := m.Stop()
	h := dp(gtx, lineH)
	top := (h - d.Size.Y) / 2
	off := op.Offset(image.Pt(0, top)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return D{Size: image.Pt(d.Size.X, h), Baseline: h - (top + d.Size.Y - d.Baseline)}
}

// dottedUnderline is T3's link hover: dots of 1.5dp every 4dp along the
// bottom of the text's box (radial-gradient(circle, currentColor 0.75px,
// transparent 1px) at 4px 2px, repeat-x).
func (th *Theme) dottedUnderline(gtx C, r image.Rectangle, col color.NRGBA) {
	step := float32(dp(gtx, 4))
	rad := 0.75 * gtx.Metric.PxPerDp
	y := float32(r.Max.Y) - gtx.Metric.PxPerDp
	for x := float32(r.Min.X) + step/2; x < float32(r.Max.X); x += step {
		c := f32.Pt(x, y)
		paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: image.Pt(int(c.X-rad+0.5), int(c.Y-rad+0.5)), Max: image.Pt(int(c.X+rad+0.5), int(c.Y+rad+0.5))}.Op(gtx.Ops))
	}
}
