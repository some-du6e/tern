package ui

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

type (
	C = layout.Context
	D = layout.Dimensions
	W = layout.Widget
)

func dp(gtx C, v float32) int { return gtx.Dp(unit.Dp(v)) }

func fill(gtx C, col color.NRGBA, r image.Rectangle, radius int) {
	paint.FillShape(gtx.Ops, col, clip.UniformRRect(r, radius).Op(gtx.Ops))
}

// strokeRect paints a border the way CSS does: inside r, width wide, the
// ring between the rounded border box and the padding box (radius less
// the width). A stroke centred on the edge would straddle two pixels.
func strokeRect(gtx C, col color.NRGBA, r image.Rectangle, radius int, width float32) {
	var p clip.Path
	p.Begin(gtx.Ops)
	x0, y0, x1, y1 := float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X), float32(r.Max.Y)
	rrect(&p, x0, y0, x1, y1, float32(radius), true)
	if x1-x0 > 2*width && y1-y0 > 2*width {
		rrect(&p, x0+width, y0+width, x1-width, y1-width, max(0, float32(radius)-width), false)
	}
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

// rrect adds the rounded rect (x0, y0)-(x1, y1) to p, clockwise or counterclockwise, so a
// second one wound the other way cuts a hole in the first.
func rrect(p *clip.Path, x0, y0, x1, y1, rad float32, cw bool) {
	rad = min(rad, (x1-x0)/2, (y1-y0)/2)
	const k = 0.5522847 // a quarter circle as a cubic
	c := rad * (1 - k)
	if cw {
		p.MoveTo(f32.Pt(x0+rad, y0))
		p.LineTo(f32.Pt(x1-rad, y0))
		p.CubeTo(f32.Pt(x1-c, y0), f32.Pt(x1, y0+c), f32.Pt(x1, y0+rad))
		p.LineTo(f32.Pt(x1, y1-rad))
		p.CubeTo(f32.Pt(x1, y1-c), f32.Pt(x1-c, y1), f32.Pt(x1-rad, y1))
		p.LineTo(f32.Pt(x0+rad, y1))
		p.CubeTo(f32.Pt(x0+c, y1), f32.Pt(x0, y1-c), f32.Pt(x0, y1-rad))
		p.LineTo(f32.Pt(x0, y0+rad))
		p.CubeTo(f32.Pt(x0, y0+c), f32.Pt(x0+c, y0), f32.Pt(x0+rad, y0))
	} else {
		p.MoveTo(f32.Pt(x0+rad, y0))
		p.CubeTo(f32.Pt(x0+c, y0), f32.Pt(x0, y0+c), f32.Pt(x0, y0+rad))
		p.LineTo(f32.Pt(x0, y1-rad))
		p.CubeTo(f32.Pt(x0, y1-c), f32.Pt(x0+c, y1), f32.Pt(x0+rad, y1))
		p.LineTo(f32.Pt(x1-rad, y1))
		p.CubeTo(f32.Pt(x1-c, y1), f32.Pt(x1, y1-c), f32.Pt(x1, y1-rad))
		p.LineTo(f32.Pt(x1, y0+rad))
		p.CubeTo(f32.Pt(x1, y0+c), f32.Pt(x1-c, y0), f32.Pt(x1-rad, y0))
		p.LineTo(f32.Pt(x0+rad, y0))
	}
	p.Close()
}

func hline(gtx C, col color.NRGBA, y, x0, x1 int) {
	fill(gtx, col, image.Rect(x0, y, x1, y+max(1, dp(gtx, 1))), 0)
}

func vline(gtx C, col color.NRGBA, x, y0, y1 int) {
	fill(gtx, col, image.Rect(x, y0, x+max(1, dp(gtx, 1)), y1), 0)
}

func circle(gtx C, col color.NRGBA, center image.Point, r int) {
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: center.Sub(image.Pt(r, r)), Max: center.Add(image.Pt(r, r))}.Op(gtx.Ops))
}

func polyline(gtx C, col color.NRGBA, width float32, pts ...f32.Point) {
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(pts[0])
	for _, pt := range pts[1:] {
		p.LineTo(pt)
	}
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: width}.Op())
}

// Box paints a background and optional border behind its child, sized to it.
type Box struct {
	Bg, Border color.NRGBA
	Radius     unit.Dp
	Pad        layout.Inset
}

func (b Box) Layout(gtx C, w W) D {
	m := op.Record(gtx.Ops)
	dims := b.Pad.Layout(gtx, w)
	call := m.Stop()
	r := image.Rectangle{Max: dims.Size}
	rad := gtx.Dp(b.Radius)
	if b.Bg.A > 0 {
		fill(gtx, b.Bg, r, rad)
	}
	if b.Border.A > 0 {
		strokeRect(gtx, b.Border, r, rad, float32(max(1, dp(gtx, 1))))
	}
	call.Add(gtx.Ops)
	return dims
}

// Fill paints the whole constrained area, then lays out the child in it.
func Fill(gtx C, col color.NRGBA, w W) D {
	size := gtx.Constraints.Max
	fill(gtx, col, image.Rectangle{Max: size}, 0)
	gtx.Constraints.Min = size
	w(gtx)
	return D{Size: size}
}

func pad(t, r, b, l float32) layout.Inset {
	return layout.Inset{Top: unit.Dp(t), Right: unit.Dp(r), Bottom: unit.Dp(b), Left: unit.Dp(l)}
}

func padXY(x, y float32) layout.Inset { return pad(y, x, y, x) }

// FixedH lays out w at exactly h dp tall.
func FixedH(gtx C, h float32, w W) D {
	px := dp(gtx, h)
	gtx.Constraints.Min.Y, gtx.Constraints.Max.Y = px, px
	return w(gtx)
}

func FixedW(gtx C, wdp float32, w W) D {
	px := dp(gtx, wdp)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = px, px
	return w(gtx)
}

func Spacer(wdp, hdp float32) W {
	return func(gtx C) D { return D{Size: image.Pt(dp(gtx, wdp), dp(gtx, hdp))} }
}

// ---------- text ----------

type Txt struct {
	Size     unit.Sp
	Color    color.NRGBA
	Font     font.Font
	MaxLines int
	Align    text.Alignment
	LineH    float32 // line height scale, 0 for the default
}

func (th *Theme) Text(gtx C, t Txt, s string) D {
	if t.Size == 0 {
		t.Size = SizeUI
	}
	if t.Color.A == 0 {
		t.Color = ColText
	}
	if t.Font.Typeface == "" {
		t.Font = UI(font.Normal)
	}
	s = strings.ReplaceAll(s, "\t", "    ")
	m := op.Record(gtx.Ops)
	paint.ColorOp{Color: t.Color}.Add(gtx.Ops)
	mat := m.Stop()
	l := widget.Label{MaxLines: t.MaxLines, Alignment: t.Align, LineHeightScale: t.LineH}
	if t.MaxLines > 0 {
		l.Truncator = "…"
	}
	return l.Layout(gtx, th.Shaper, t.Font, t.Size, s, mat)
}

// capText draws s on one line with its capitals centred on y mid (px from
// the current origin), as T3 trims a label's box to its cap height
// (text-box: trim-both cap alphabetic) and centres that. It returns the
// line's width.
func (th *Theme) capText(gtx C, t Txt, s string, mid int) int {
	if t.Size == 0 {
		t.Size = SizeUI
	}
	t.MaxLines = 1
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	d := th.Text(g, t, s)
	call := m.Stop()
	baseline := d.Size.Y - d.Baseline
	capH := capEm * float32(gtx.Sp(t.Size))
	defer op.Offset(image.Pt(0, mid+int(capH/2+0.5)-baseline)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return d.Size.X
}

// boxText draws s wrapped in the width gtx allows, up to t.MaxLines lines
// of CSS line boxes t.LineH tall from the current origin, each line's caps
// centred in its box as capText centres one (Inter's and SF Pro's caps
// centre within a twentieth of a dp of where CSS's half-leading puts
// them). Its size is the line boxes'.
func (th *Theme) boxText(gtx C, t Txt, s string) D {
	g := gtx
	g.Constraints.Min = image.Point{}
	one := t
	one.MaxLines = 1
	m := op.Record(gtx.Ops)
	first := th.Text(g, one, s)
	m.Stop()
	m = op.Record(gtx.Ops)
	d := th.Text(g, t, s)
	call := m.Stop()
	// A label's baseline is its first line's; each line after it adds a
	// line's height.
	lineH := float32(gtx.Sp(t.Size)) * t.LineH
	baseline := first.Size.Y - first.Baseline
	lines := 1 + int(float32(d.Size.Y-first.Size.Y)/lineH+0.5)
	capH := capEm * float32(gtx.Sp(t.Size))
	defer op.Offset(image.Pt(0, int(lineH/2+capH/2+0.5)-baseline)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: image.Pt(d.Size.X, int(lineH*float32(lines)+0.5))}
}

// Label is the common case: one line of UI text.
func (th *Theme) Label(gtx C, col color.NRGBA, w font.Weight, s string) D {
	return th.Text(gtx, Txt{Color: col, Font: UI(w), MaxLines: 1}, s)
}

func (th *Theme) Small(gtx C, col color.NRGBA, s string) D {
	return th.Text(gtx, Txt{Size: SizeSmall, Color: col, Font: UI(font.Medium), MaxLines: 1}, s)
}

// ---------- icons, drawn so they don't depend on font glyphs ----------

type Icon int

const (
	IconChevronRight Icon = iota
	IconChevronDown
	IconPlus
	IconClose
	IconSearch
	IconDots
	IconStop
	IconSend
	IconPin
	IconCheck
	IconTerminal
	IconFile
	IconAgent
	IconSpark
	IconQueue
	IconImage
	IconLock
	IconGlobe
	IconMachine
	IconBack
	IconForward
	IconReload
	IconSplit
	IconGauge
	IconStar
	IconStarFill
	IconFolder
	IconUser
	IconCopy
	IconPlay
)

// DrawIcon draws a 16dp-box icon at the origin.
func DrawIcon(gtx C, ic Icon, col color.NRGBA, sizeDp float32) D {
	s := float32(dp(gtx, sizeDp))
	u := s / 16
	sw := 1.5 * u
	pt := func(x, y float32) f32.Point { return f32.Pt(x*u, y*u) }
	switch ic {
	case IconChevronRight:
		polyline(gtx, col, sw, pt(6, 4), pt(10, 8), pt(6, 12))
	case IconChevronDown:
		polyline(gtx, col, sw, pt(4, 6), pt(8, 10), pt(12, 6))
	case IconPlus:
		polyline(gtx, col, sw, pt(8, 3), pt(8, 13))
		polyline(gtx, col, sw, pt(3, 8), pt(13, 8))
	case IconClose:
		polyline(gtx, col, sw, pt(4.5, 4.5), pt(11.5, 11.5))
		polyline(gtx, col, sw, pt(11.5, 4.5), pt(4.5, 11.5))
	case IconSearch:
		ring(gtx, col, pt(7, 7), 4.2*u, sw)
		polyline(gtx, col, sw, pt(10.2, 10.2), pt(13.5, 13.5))
	case IconDots:
		for _, x := range []float32{3.5, 8, 12.5} {
			circle(gtx, col, pt(x, 8).Round(), int(1.3*u+0.5))
		}
	case IconStop:
		fill(gtx, col, image.Rect(int(4*u), int(4*u), int(12*u), int(12*u)), int(1.5*u))
	case IconSend:
		polyline(gtx, col, sw, pt(8, 13), pt(8, 3.5))
		polyline(gtx, col, sw, pt(4, 7.5), pt(8, 3.5), pt(12, 7.5))
	case IconPin:
		polyline(gtx, col, sw, pt(6, 2.8), pt(10, 2.8))
		polyline(gtx, col, sw, pt(6.7, 3), pt(6.2, 7.5), pt(4.2, 9.6), pt(11.8, 9.6), pt(9.8, 7.5), pt(9.3, 3))
		polyline(gtx, col, sw, pt(8, 9.6), pt(8, 13.5))
	case IconCheck:
		polyline(gtx, col, sw*1.1, pt(3.5, 8.5), pt(6.5, 11.5), pt(12.5, 4.5))
	case IconTerminal:
		polyline(gtx, col, sw, pt(3, 4.5), pt(7, 8), pt(3, 11.5))
		polyline(gtx, col, sw, pt(8.5, 12), pt(13, 12))
	case IconFile:
		polyline(gtx, col, sw, pt(9, 2.5), pt(4, 2.5), pt(4, 13.5), pt(12, 13.5), pt(12, 5.5), pt(9, 2.5), pt(9, 5.5), pt(12, 5.5))
	case IconAgent:
		ring(gtx, col, pt(8, 5.5), 2.6*u, sw)
		polyline(gtx, col, sw, pt(3, 13.5), pt(3.8, 11), pt(6, 9.6), pt(10, 9.6), pt(12.2, 11), pt(13, 13.5))
	case IconSpark:
		polyline(gtx, col, sw, pt(8, 2), pt(9.3, 6.7), pt(14, 8), pt(9.3, 9.3), pt(8, 14), pt(6.7, 9.3), pt(2, 8), pt(6.7, 6.7), pt(8, 2))
	case IconQueue:
		for _, y := range []float32{4.5, 8, 11.5} {
			polyline(gtx, col, sw, pt(3, y), pt(13, y))
		}
	case IconPlay:
		polyline(gtx, col, sw, pt(5, 3.5), pt(12.5, 8), pt(5, 12.5), pt(5, 3.5))
	case IconCopy:
		strokeRect(gtx, col, image.Rect(int(5.5*u), int(5.5*u), int(13.5*u), int(13.5*u)), int(1.5*u), sw)
		polyline(gtx, col, sw, pt(10.5, 3.2), pt(10.5, 2.5), pt(2.5, 2.5), pt(2.5, 10.5), pt(3.2, 10.5))
	case IconImage:
		strokeRect(gtx, col, image.Rect(int(2.5*u), int(3.5*u), int(13.5*u), int(12.5*u)), int(1.5*u), sw)
		polyline(gtx, col, sw, pt(3, 11.5), pt(6.5, 8), pt(9, 10.5), pt(10.5, 9), pt(13, 11.5))
	case IconLock:
		strokeRect(gtx, col, image.Rect(int(3.5*u), int(7*u), int(12.5*u), int(13.5*u)), int(1.5*u), sw)
		polyline(gtx, col, sw, pt(5.5, 7), pt(5.5, 5), pt(6.4, 3.3), pt(8, 2.7), pt(9.6, 3.3), pt(10.5, 5), pt(10.5, 7))
	case IconGlobe:
		ring(gtx, col, pt(8, 8), 5.5*u, sw)
		polyline(gtx, col, sw, pt(2.5, 8), pt(13.5, 8))
		polyline(gtx, col, sw, pt(8, 2.5), pt(6, 5), pt(5.5, 8), pt(6, 11), pt(8, 13.5))
		polyline(gtx, col, sw, pt(8, 2.5), pt(10, 5), pt(10.5, 8), pt(10, 11), pt(8, 13.5))
	case IconBack:
		polyline(gtx, col, sw, pt(13, 8), pt(3.5, 8))
		polyline(gtx, col, sw, pt(7.5, 4), pt(3.5, 8), pt(7.5, 12))
	case IconForward:
		polyline(gtx, col, sw, pt(3, 8), pt(12.5, 8))
		polyline(gtx, col, sw, pt(8.5, 4), pt(12.5, 8), pt(8.5, 12))
	case IconReload:
		var arc []f32.Point
		for i := 0; i <= 20; i++ {
			a := -math.Pi/2 + float64(i)/20*1.6*math.Pi
			arc = append(arc, pt(8+5*float32(math.Cos(a)), 8+5*float32(math.Sin(a))))
		}
		polyline(gtx, col, sw, arc...)
		polyline(gtx, col, sw, pt(5.2, 1.6), pt(8, 3), pt(6.2, 5.6))
	case IconStar, IconStarFill:
		var pts []f32.Point
		for i := 0; i <= 10; i++ {
			r := float32(6.2)
			if i%2 == 1 {
				r = 2.6
			}
			a := -math.Pi/2 + float64(i)*math.Pi/5
			pts = append(pts, pt(8+r*float32(math.Cos(a)), 8.6+r*float32(math.Sin(a))))
		}
		if ic == IconStar {
			polyline(gtx, col, sw*0.9, pts...)
			break
		}
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(pts[0])
		for _, q := range pts[1:] {
			p.LineTo(q)
		}
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
	case IconGauge:
		var arc []f32.Point
		for i := 0; i <= 20; i++ {
			a := math.Pi*0.8 + float64(i)/20*math.Pi*1.4
			arc = append(arc, pt(8+5.5*float32(math.Cos(a)), 9+5.5*float32(math.Sin(a))))
		}
		polyline(gtx, col, sw, arc...)
		polyline(gtx, col, sw, pt(8, 9), pt(11, 5.8))
		circle(gtx, col, pt(8, 9).Round(), int(1.2*u+0.5))
	case IconSplit:
		strokeRect(gtx, col, image.Rect(int(2.5*u), int(2.5*u), int(13.5*u), int(13.5*u)), int(1.5*u), sw)
		polyline(gtx, col, sw, pt(2.5, 8), pt(13.5, 8))
	case IconMachine:
		strokeRect(gtx, col, image.Rect(int(2.5*u), int(3*u), int(13.5*u), int(10.5*u)), int(1.2*u), sw)
		polyline(gtx, col, sw, pt(8, 10.5), pt(8, 13))
		polyline(gtx, col, sw, pt(5, 13.2), pt(11, 13.2))
	case IconFolder:
		polyline(gtx, col, sw, pt(2.5, 12.5), pt(2.5, 3.5), pt(6.5, 3.5), pt(8, 5), pt(13.5, 5), pt(13.5, 12.5), pt(2.5, 12.5))
	case IconUser:
		ring(gtx, col, pt(8, 5.8), 2.6*u, sw)
		polyline(gtx, col, sw, pt(3, 13.5), pt(3.6, 11.2), pt(5.6, 9.8), pt(10.4, 9.8), pt(12.4, 11.2), pt(13, 13.5))
	}
	return D{Size: image.Pt(int(s), int(s))}
}

func ring(gtx C, col color.NRGBA, c f32.Point, r, width float32) {
	var p clip.Path
	p.Begin(gtx.Ops)
	const n = 32
	for i := 0; i <= n; i++ {
		a := float64(i) / n * 2 * math.Pi
		q := f32.Pt(c.X+r*float32(math.Cos(a)), c.Y+r*float32(math.Sin(a)))
		if i == 0 {
			p.MoveTo(q)
		} else {
			p.LineTo(q)
		}
	}
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: width}.Op())
}

func IconW(ic Icon, col color.NRGBA, sizeDp float32) W {
	return func(gtx C) D { return DrawIcon(gtx, ic, col, sizeDp) }
}

// Dot is a status light: online, busy, needs-you, failed.
func Dot(col color.NRGBA, halo bool) W {
	return func(gtx C) D {
		s := dp(gtx, 8)
		c := image.Pt(s/2, s/2)
		if halo {
			circle(gtx, alpha(col, 0x38), c, dp(gtx, 4))
		}
		circle(gtx, col, c, dp(gtx, 2.6))
		return D{Size: image.Pt(s, s)}
	}
}

// Spinner draws a partial ring, the run-is-alive indicator.
func Spinner(col color.NRGBA, sizeDp float32, phase float64) W {
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		r := s/2 - float32(dp(gtx, 1.5))
		c := f32.Pt(s/2, s/2)
		ring(gtx, alpha(col, 0x40), c, r, float32(dp(gtx, 1.6)))
		var p clip.Path
		p.Begin(gtx.Ops)
		const n = 16
		for i := 0; i <= n; i++ {
			a := phase + float64(i)/n*math.Pi*0.6
			q := f32.Pt(c.X+r*float32(math.Cos(a)), c.Y+r*float32(math.Sin(a)))
			if i == 0 {
				p.MoveTo(q)
			} else {
				p.LineTo(q)
			}
		}
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: float32(dp(gtx, 1.6))}.Op())
		return D{Size: image.Pt(int(s), int(s))}
	}
}

// ---------- rows and frames ----------

func rigid(w W) layout.FlexChild  { return layout.Rigid(w) }
func flexed(w W) layout.FlexChild { return layout.Flexed(1, w) }

// Row lays children out left to right and centers them in the strip's
// minimum height. Flex alone would pass that minimum down and pin text to
// the top.
func Row(gtx C, children ...layout.FlexChild) D { return RowGap(gtx, 0, children...) }

// RowGap is Row with gapDp between the children, as a CSS flex row with a
// gap centres its items.
func RowGap(gtx C, gapDp float32, children ...layout.FlexChild) D {
	h := gtx.Constraints.Min.Y
	gtx.Constraints.Min.Y = 0
	m := op.Record(gtx.Ops)
	dims := layout.Flex{Alignment: layout.Middle, Gap: dp(gtx, gapDp)}.Layout(gtx, children...)
	call := m.Stop()
	if dims.Size.Y >= h {
		call.Add(gtx.Ops)
		return dims
	}
	t := op.Offset(image.Pt(0, (h-dims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
	return D{Size: image.Pt(dims.Size.X, h), Baseline: dims.Baseline}
}

func Col(gtx C, children ...layout.FlexChild) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// Centered vertically within a fixed-height strip.
func Middle(w W) W {
	return func(gtx C) D {
		return layout.W.Layout(gtx, w)
	}
}

type Variant int

const (
	Plain Variant = iota
	Primary
	Ghost
	Danger
	Toggled
)

// Button is an ImGui-style flat button: frame color, no shadow, small radius.
func (th *Theme) Button(v Variant, ic Icon, hasIcon bool, label string) W {
	return func(gtx C) D {
		bg, fg, border := ColFrame, ColText, ColBorder
		switch v {
		case Primary:
			bg, fg, border = ColAccent, rgb(0xffffff), color.NRGBA{}
		case Ghost:
			bg, fg, border = color.NRGBA{}, ColDim, color.NRGBA{}
		case Danger:
			bg, fg, border = alpha(ColRed, 0x1e), ColRed, alpha(ColRed, 0x50)
		case Toggled:
			bg, fg, border = ColAccentSoft, ColAccentHi, alpha(ColAccent, 0x70)
		}
		px := float32(9)
		if label == "" {
			px = 5
		}
		return Box{Bg: bg, Border: border, Radius: 4, Pad: padXY(px, 0)}.Layout(gtx, func(gtx C) D {
			return FixedH(gtx, 24, func(gtx C) D {
				var kids []layout.FlexChild
				if hasIcon {
					kids = append(kids, rigid(IconW(ic, fg, 14)))
					if label != "" {
						kids = append(kids, rigid(Spacer(5, 0)))
					}
				}
				if label != "" {
					kids = append(kids, rigid(func(gtx C) D { return th.Label(gtx, fg, font.Medium, label) }))
				}
				return Row(gtx, kids...)
			})
		})
	}
}

// Combo is a closed ImGui combo: value, then the arrow square. widthDp 0
// sizes it to its value; Stretch fills the available width.
func (th *Theme) Combo(prefix W, value string, widthDp float32) W {
	return func(gtx C) D {
		body := func(gtx C) D {
			return Box{Bg: ColFrame, Border: ColBorder, Radius: 4}.Layout(gtx, func(gtx C) D {
				return FixedH(gtx, 24, func(gtx C) D {
					return Row(gtx,
						rigid(Spacer(8, 0)),
						rigid(func(gtx C) D {
							if prefix == nil {
								return D{}
							}
							return pad(0, 6, 0, 0).Layout(gtx, prefix)
						}),
						flexOrRigid(widthDp != 0, func(gtx C) D { return th.Label(gtx, ColText, font.Normal, value) }),
						rigid(Spacer(6, 0)),
						rigid(func(gtx C) D {
							size := image.Pt(dp(gtx, 22), dp(gtx, 24))
							fill(gtx, ColFrameHover, image.Rectangle{Max: size}, 0)
							vline(gtx, ColBorder, 0, 0, size.Y)
							op.Offset(image.Pt(dp(gtx, 4), (size.Y-dp(gtx, 14))/2)).Add(gtx.Ops)
							DrawIcon(gtx, IconChevronDown, ColDim, 14)
							return D{Size: size}
						}),
					)
				})
			})
		}
		switch {
		case widthDp == Stretch:
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
		case widthDp > 0:
			return FixedW(gtx, widthDp, body)
		}
		return body(gtx)
	}
}

const Stretch = -1

func flexOrRigid(flex bool, w W) layout.FlexChild {
	if flex {
		return layout.Flexed(1, w)
	}
	return layout.Rigid(w)
}

// Input is a single-line frame with an optional icon and placeholder.
func (th *Theme) Input(ic Icon, hasIcon bool, value, placeholder string, focused bool) W {
	return func(gtx C) D {
		border := ColBorder
		if focused {
			border = alpha(ColAccent, 0xa0)
		}
		return Box{Bg: ColFrame, Border: border, Radius: 4, Pad: padXY(8, 0)}.Layout(gtx, func(gtx C) D {
			return FixedH(gtx, 26, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Row(gtx,
					rigid(func(gtx C) D {
						if !hasIcon {
							return D{}
						}
						return pad(0, 6, 0, 0).Layout(gtx, IconW(ic, ColFaint, 14))
					}),
					flexed(func(gtx C) D {
						if value == "" {
							return th.Label(gtx, ColFaint, font.Normal, placeholder)
						}
						return Row(gtx,
							rigid(func(gtx C) D { return th.Label(gtx, ColText, font.Normal, value) }),
							rigid(func(gtx C) D {
								if !focused {
									return D{}
								}
								return Caret(gtx, 15)
							}),
						)
					}),
				)
			})
		})
	}
}

func Caret(gtx C, hdp float32) D {
	w, h := max(1, dp(gtx, 1.4)), dp(gtx, hdp)
	fill(gtx, ColAccentHi, image.Rect(dp(gtx, 1), 0, dp(gtx, 1)+w, h), 0)
	return D{Size: image.Pt(dp(gtx, 3), h)}
}

func (th *Theme) Checkbox(on bool, label string) W {
	return func(gtx C) D {
		return Row(gtx,
			rigid(func(gtx C) D {
				s := dp(gtx, 15)
				r := image.Rectangle{Max: image.Pt(s, s)}
				if on {
					fill(gtx, ColAccent, r, dp(gtx, 3))
					op.Offset(image.Pt(dp(gtx, 0.5), dp(gtx, 0.5))).Add(gtx.Ops)
					DrawIcon(gtx, IconCheck, rgb(0xffffff), 14)
				} else {
					fill(gtx, ColFrame, r, dp(gtx, 3))
					strokeRect(gtx, ColBorderHi, r, dp(gtx, 3), float32(max(1, dp(gtx, 1))))
				}
				return D{Size: r.Max}
			}),
			rigid(Spacer(7, 0)),
			rigid(func(gtx C) D { return th.Label(gtx, ColText, font.Normal, label) }),
		)
	}
}

// Kbd is a key hint like "⌘K".
// Shortcut is how this computer writes the shortcut key with k, as T3
// does: ⌘K on a Mac, Ctrl+K elsewhere.
func Shortcut(k string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + k
	}
	return "Ctrl+" + k
}

func (th *Theme) Kbd(s string) W {
	return func(gtx C) D {
		return Box{Bg: ColPanelHi, Border: ColBorder, Radius: 3, Pad: padXY(4, 1)}.Layout(gtx, func(gtx C) D {
			return th.Text(gtx, Txt{Size: 10.5, Color: ColDim, Font: Mono(), MaxLines: 1}, s)
		})
	}
}

// Chip is a small rounded status tag.
func (th *Theme) Chip(col color.NRGBA, label string, lead W) W {
	return func(gtx C) D {
		return Box{Bg: alpha(col, 0x1c), Border: alpha(col, 0x48), Radius: 9, Pad: pad(2, 8, 2, 6)}.Layout(gtx, func(gtx C) D {
			return Row(gtx,
				rigid(func(gtx C) D {
					if lead == nil {
						return D{}
					}
					return pad(0, 5, 0, 0).Layout(gtx, lead)
				}),
				rigid(func(gtx C) D {
					return th.Text(gtx, Txt{Size: SizeSmall, Color: col, Font: UI(font.SemiBold), MaxLines: 1}, label)
				}),
			)
		})
	}
}

// SectionLabel is a small caps-ish heading with a rule after it.
func (th *Theme) SectionLabel(label string, trailing W) W {
	return func(gtx C) D {
		return pad(10, 10, 4, 12).Layout(gtx, func(gtx C) D {
			return Row(gtx,
				rigid(func(gtx C) D {
					return th.Text(gtx, Txt{Size: 10.5, Color: ColFaint, Font: UI(font.SemiBold), MaxLines: 1}, label)
				}),
				flexed(func(gtx C) D {
					h := dp(gtx, 14)
					return D{Size: image.Pt(gtx.Constraints.Min.X, h)}
				}),
				rigid(func(gtx C) D {
					if trailing == nil {
						return D{}
					}
					return trailing(gtx)
				}),
			)
		})
	}
}

// Scrollbar draws an ImGui-style thin thumb on the right edge of size.
func Scrollbar(gtx C, size image.Point, start, length float32) {
	if length >= 1 {
		return
	}
	w := dp(gtx, 5)
	x := size.X - w - dp(gtx, 3)
	track := image.Rect(x, dp(gtx, 4), x+w, size.Y-dp(gtx, 4))
	h := track.Dy()
	thumb := image.Rect(x, track.Min.Y+int(start*float32(h)), x+w, track.Min.Y+int((start+length)*float32(h)))
	fill(gtx, alpha(ColBorderHi, 0xc0), thumb, w/2)
}

// boxShadow is one CSS box-shadow: offsets, blur and spread in dp, colour.
type boxShadow struct {
	x, y, blur, spread float32
	col                color.NRGBA
}

// shadowKey is one drawn box shadow, in px: the element's size and radius,
// the shadow's offset, blur and spread, and its colour.
type shadowKey struct {
	w, h, radius       int
	x, y, blur, spread int
	col                color.NRGBA
}

// shadowImages are the shadows drawn so far, by shape; a handful at most
// are in use, so the lot goes once it grows past that.
var shadowImages = map[shadowKey]paint.ImageOp{}

// shadow paints CSS box shadows for the rounded rect r (radius in px)
// smoothly, as Chrome does (01-tokens.md port note 9): each is r grown by
// its spread and moved by its offset, blurred by a Gaussian of half its
// blur radius, and clipped out of r itself. Gio has no blur, so each is
// drawn once into an image and kept.
func shadow(gtx C, r image.Rectangle, radius int, shadows []boxShadow) {
	for _, sh := range shadows {
		k := shadowKey{w: r.Dx(), h: r.Dy(), radius: radius, x: dp(gtx, sh.x), y: dp(gtx, sh.y),
			blur: dp(gtx, sh.blur), spread: dp(gtx, sh.spread), col: sh.col}
		if k.w+2*k.spread <= 0 || k.h+2*k.spread <= 0 {
			continue
		}
		im, ok := shadowImages[k]
		if !ok {
			if len(shadowImages) > 32 {
				clear(shadowImages)
			}
			im = paint.NewImageOp(shadowImage(k))
			shadowImages[k] = im
		}
		func() {
			ext := shadowExtent(k.blur)
			defer op.Offset(r.Min.Add(image.Pt(k.x-k.spread-ext, k.y-k.spread-ext))).Push(gtx.Ops).Pop()
			im.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
		}()
	}
}

// shadowExtent is how far a shadow of blur radius blur reaches past its
// shape: three of its Gaussian's deviations, half the blur radius each.
func shadowExtent(blur int) int { return (3*blur + 1) / 2 }

// shadowImage is k's shadow, from its shape's top left less its extent:
// at each pixel the colour at the share of the blurred shape covering it,
// 0.5·erfc(d/(σ√2)) at signed distance d from the shape's edge, none
// inside the element. Gio blends in linear light, so the alpha is raised
// to darken what's under it as much as Chrome's sRGB blend of black does.
func shadowImage(k shadowKey) *image.NRGBA {
	ext := shadowExtent(k.blur)
	sw, shh := k.w+2*k.spread, k.h+2*k.spread
	img := image.NewNRGBA(image.Rect(0, 0, sw+2*ext, shh+2*ext))
	sigma := math.Max(float64(k.blur)/2, 0.5)
	sdf := func(px, py float64, w, h, rad float64) float64 {
		// The signed distance from (px, py) to a w×h box with corners rad.
		rad = math.Min(rad, math.Min(w, h)/2)
		qx := math.Abs(px-w/2) - (w/2 - rad)
		qy := math.Abs(py-h/2) - (h/2 - rad)
		out := math.Hypot(math.Max(qx, 0), math.Max(qy, 0))
		return out + math.Min(math.Max(qx, qy), 0) - rad
	}
	a := float64(k.col.A) / 255
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			px, py := float64(x-ext)+0.5, float64(y-ext)+0.5
			// The element, where its own box hides its shadow.
			if sdf(px-float64(k.spread-k.x), py-float64(k.spread-k.y), float64(k.w), float64(k.h), float64(k.radius)) < 0 {
				continue
			}
			d := sdf(px, py, float64(sw), float64(shh), float64(max(0, k.radius+k.spread)))
			cover := a * 0.5 * math.Erfc(d/(sigma*math.Sqrt2))
			if cover <= 0.5/255 {
				continue
			}
			// What sRGB's blend leaves of the light under black, in linear light.
			lin := 1 - math.Pow(1-cover, 2.2)
			img.SetNRGBA(x, y, color.NRGBA{R: k.col.R, G: k.col.G, B: k.col.B, A: uint8(lin*255 + 0.5)})
		}
	}
	return img
}
