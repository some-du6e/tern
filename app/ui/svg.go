package ui

import (
	"image"
	"image/color"
	"sync"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

var lucideSegs sync.Map // icon → []seg

// drawnIcons are T3's own icons in lucide's style, 24x24 outlines drawn
// like lucide's: the Mac mini and Mac Studio machine marks
// (EnvironmentMachineIcon.tsx), whose filled dots become lucide dots, the
// check a menu checkbox and a Checkbox draw (menu.tsx MenuCheckboxItem,
// checkbox.tsx) and a mixed Checkbox's dash.
var drawnIcons = map[string]string{
	"menu-check": `M5.252 12.7 10.2 18.63 18.748 5.37`,
	"mixed":      `M5.252 12h13.496`,
	"mac-mini":   `M5.5 6H18.5A2.5 2.5 0 0 1 21 8.5V15.5A2.5 2.5 0 0 1 18.5 18H5.5A2.5 2.5 0 0 1 3 15.5V8.5A2.5 2.5 0 0 1 5.5 6ZM7.5 12h.01`,
	"mac-studio": `M5 5H19A2 2 0 0 1 21 7V17A2 2 0 0 1 19 19H5A2 2 0 0 1 3 17V7A2 2 0 0 1 5 5ZM7 15h.01M11 15h.01M15 15h.01`,
}

// LucideW is the lucide icon name, the set T3 uses, sizeDp square and
// stroked in col at T3's weight: 1.75, not lucide's 2 (index.css:1728).
func LucideW(name string, col color.NRGBA, sizeDp float32) W {
	d := lucide[name]
	if d == "" {
		d = drawnIcons[name]
	}
	return strokeIcon(name, d, col, sizeDp, 1.75)
}

// menuIcon is the in-page context menu's own copy of icon name
// (contextMenuFallback.ts), sizeDp square in col at lucide's own 2: its
// SVGs carry no lucide class for T3's 1.75 to reach.
func menuIcon(name string, col color.NRGBA, sizeDp float32) W {
	return strokeIcon("menu:"+name, menuIcons[name], col, sizeDp, 2)
}

// strokeIcon strokes the 24x24 outline d (parsed once, kept as key)
// sizeDp square in col, its lines stroke units of the 24 wide.
func strokeIcon(key, d string, col color.NRGBA, sizeDp, stroke float32) W {
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		v, ok := lucideSegs.Load(key)
		if !ok {
			v, _ = lucideSegs.LoadOrStore(key, parseSVGPath(d))
		}
		k, width := s/24, stroke*s/24
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
