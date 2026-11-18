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
// stroked in col at T3's weight.
func LucideW(name string, col color.NRGBA, sizeDp float32) W {
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		v, ok := lucideSegs.Load(name)
		if !ok {
			v, _ = lucideSegs.LoadOrStore(name, parseSVGPath(lucide[name]))
		}
		k, width := s/24, 1.75*s/24 // T3 strokes lucide at 1.75, not 2 (index.css:1728)
		sc := func(q f32.Point) f32.Point { return q.Mul(k) }
		var p clip.Path
		p.Begin(gtx.Ops)
		// Lucide draws dots as zero-length strokes with round ends, which
		// stroke to nothing here, so they're filled in after.
		var dots []f32.Point
		var start f32.Point
		point := false // the subpath so far is a single point
		for _, sg := range v.([]seg) {
			switch sg.op {
			case 'M':
				if point {
					dots = append(dots, start)
				}
				start, point = sg.p[0], true
				p.MoveTo(sc(sg.p[0]))
			case 'L':
				point = point && sg.p[0].Sub(start).X*sg.p[0].Sub(start).X+sg.p[0].Sub(start).Y*sg.p[0].Sub(start).Y < 0.01
				p.LineTo(sc(sg.p[0]))
			case 'Q':
				point = false
				p.QuadTo(sc(sg.p[0]), sc(sg.p[1]))
			case 'C':
				point = false
				p.CubeTo(sc(sg.p[0]), sc(sg.p[1]), sc(sg.p[2]))
			case 'Z':
				p.Close()
			}
		}
		if point {
			dots = append(dots, start)
		}
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: width}.Op())
		for _, q := range dots {
			c, r := sc(q), width/2
			paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: image.Pt(int(c.X-r+0.5), int(c.Y-r+0.5)), Max: image.Pt(int(c.X+r+0.5), int(c.Y+r+0.5))}.Op(gtx.Ops))
		}
		return D{Size: image.Pt(int(s), int(s))}
	}
}
