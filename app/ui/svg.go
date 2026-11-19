package ui

import (
	"image"
	"image/color"
	"sync"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

var lucideSegs sync.Map // name → []seg

// LucideW is the lucide icon name, the set T3 uses, sizeDp square and
// stroked in col at T3's weight: 1.75, not lucide's 2 (index.css:1728).
func LucideW(name string, col color.NRGBA, sizeDp float32) W {
	return lucideStroke(name, col, sizeDp, 1.75)
}

// lucideStroke is LucideW at a stroke of width units in the icon's 24.
func lucideStroke(name string, col color.NRGBA, sizeDp, width float32) W {
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		segs := cachedPath(name, lucide[name])
		k, width := s/24, width*s/24
		sc := func(q f32.Point) f32.Point { return q.Mul(k) }
		var p clip.Path
		p.Begin(gtx.Ops)
		addSegs(&p, segs, sc)
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: width}.Op())
		// Lucide draws dots as zero-length strokes with round ends, which
		// stroke to nothing here, so they're filled in after.
		for _, q := range pathDots(segs) {
			c, r := sc(q), width/2
			paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: image.Pt(int(c.X-r+0.5), int(c.Y-r+0.5)), Max: image.Pt(int(c.X+r+0.5), int(c.Y+r+0.5))}.Op(gtx.Ops))
		}
		return D{Size: image.Pt(int(s), int(s))}
	}
}

// brands are the marks T3 draws for source control hosts: filled path data
// in a box units square (packages/source-control-github/src/shared/icon.ts).
var brands = map[string]struct {
	d   string
	box float32
}{
	"github": {"M8 0C3.58 0 0 3.58 0 8C0 11.54 2.29 14.53 5.47 15.59C5.87 15.66 6.02 15.42 6.02 15.21C6.02 15.02 6.01 14.39 6.01 13.72" +
		"C4 14.09 3.48 13.23 3.32 12.78C3.23 12.55 2.84 11.84 2.5 11.65C2.22 11.5 1.82 11.13 2.49 11.12C3.12 11.11 3.57 11.7 3.72 11.94" +
		"C4.44 13.15 5.59 12.81 6.05 12.6C6.12 12.08 6.33 11.73 6.56 11.53C4.78 11.33 2.92 10.64 2.92 7.58C2.92 6.71 3.23 5.99 3.74 5.43" +
		"C3.66 5.23 3.38 4.41 3.82 3.31C3.82 3.31 4.49 3.1 6.02 4.13C6.66 3.95 7.34 3.86 8.02 3.86C8.7 3.86 9.38 3.95 10.02 4.13" +
		"C11.55 3.09 12.22 3.31 12.22 3.31C12.66 4.41 12.38 5.23 12.3 5.43C12.81 5.99 13.12 6.7 13.12 7.58C13.12 10.65 11.25 11.33 9.47 11.53" +
		"C9.76 11.78 10.01 12.26 10.01 13.01C10.01 14.08 10 14.94 10 15.21C10 15.42 10.15 15.67 10.55 15.59C13.71 14.53 16 11.53 16 8" +
		"C16 3.58 12.42 0 8 0Z", 16},
}

// iconW is the icon name, lucide's or a brand's mark, sizeDp square in col.
func iconW(name string, col color.NRGBA, sizeDp float32) W {
	b, ok := brands[name]
	if !ok {
		return LucideW(name, col, sizeDp)
	}
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		k := s / b.box
		var p clip.Path
		p.Begin(gtx.Ops)
		addSegs(&p, cachedPath("brand:"+name, b.d), func(q f32.Point) f32.Point { return q.Mul(k) })
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return D{Size: image.Pt(int(s), int(s))}
	}
}

// cachedPath is path data d parsed once, under name.
func cachedPath(name, d string) []seg {
	v, ok := lucideSegs.Load(name)
	if !ok {
		v, _ = lucideSegs.LoadOrStore(name, parseSVGPath(d))
	}
	return v.([]seg)
}

// addSegs adds segments to p, each point mapped by at.
func addSegs(p *clip.Path, segs []seg, at func(f32.Point) f32.Point) {
	for _, sg := range segs {
		switch sg.op {
		case 'M':
			p.MoveTo(at(sg.p[0]))
		case 'L':
			p.LineTo(at(sg.p[0]))
		case 'Q':
			p.QuadTo(at(sg.p[0]), at(sg.p[1]))
		case 'C':
			p.CubeTo(at(sg.p[0]), at(sg.p[1]), at(sg.p[2]))
		case 'Z':
			p.Close()
		}
	}
}

// pathDots are the subpaths that are single points.
func pathDots(segs []seg) []f32.Point {
	var dots []f32.Point
	var start f32.Point
	point := false // the subpath so far is a single point
	for _, sg := range segs {
		switch sg.op {
		case 'M':
			if point {
				dots = append(dots, start)
			}
			start, point = sg.p[0], true
		case 'L':
			d := sg.p[0].Sub(start)
			point = point && d.X*d.X+d.Y*d.Y < 0.01
		case 'Q', 'C':
			point = false
		}
	}
	if point {
		dots = append(dots, start)
	}
	return dots
}
