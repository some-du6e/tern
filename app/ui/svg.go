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

// t3Glyphs are the lucide-style icons T3 draws itself
// (EnvironmentMachineIcon.tsx): its mini PC and workstation.
var t3Glyphs = map[string]string{
	"mac-mini":   `M5.5 6H18.5A2.5 2.5 0 0 1 21 8.5V15.5A2.5 2.5 0 0 1 18.5 18H5.5A2.5 2.5 0 0 1 3 15.5V8.5A2.5 2.5 0 0 1 5.5 6ZM7.5 12L7.5 12`,
	"mac-studio": `M5 5H19A2 2 0 0 1 21 7V17A2 2 0 0 1 19 19H5A2 2 0 0 1 3 17V7A2 2 0 0 1 5 5ZM7 15h.01M11 15h.01M15 15h.01`,
}

// LucideW is the lucide icon name, the set T3 uses, sizeDp square and
// stroked in col at T3's weight: 1.75, not lucide's 2 (index.css:1728).
func LucideW(name string, col color.NRGBA, sizeDp float32) W {
	return lucideStroke(name, col, sizeDp, 1.75)
}

// lucideStroke is LucideW at a stroke of width units in the icon's 24.
func lucideStroke(name string, col color.NRGBA, sizeDp, width float32) W {
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		d, known := lucide[name]
		if !known {
			d = t3Glyphs[name]
		}
		segs := cachedPath(name, d)
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
