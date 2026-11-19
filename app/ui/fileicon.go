package ui

import (
	"image"
	"image/color"
	"path"
	"strings"
	"sync"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The sprite icons of T3's diff viewer and file trees (pierre.go), and
// which one a file gets (pierre-icons.ts, PierreEntryIcon.tsx;
// docs/ui/13-diff.md §2.12, §8 note 7).

// pierreIcon is one sprite icon: its viewBox side and its shapes, drawn in
// order.
type pierreIcon struct {
	box    float32
	shapes []pierreShape
}

// pierreShape is a filled shape, or with stroke a stroked one that wide in
// the icon's units: in the current colour, or with own its light and dark
// colours, at an opacity (0 for opaque). An evenOdd shape's holes are
// subpaths inside others.
type pierreShape struct {
	d           string
	opacity     float32
	light, dark uint32
	own         bool
	stroke      float32
	evenOdd     bool
}

var pierreSegs sync.Map // d → []seg

// PierreW draws the sprite icon name sizeDp square in col, its own
// coloured shapes in theirs, the translucent ones composited onto on, the
// surface under it.
func PierreW(name string, col, on color.NRGBA, sizeDp float32) W {
	return func(gtx C) D {
		ic := pierreIcons[name]
		s := float32(dp(gtx, sizeDp))
		k := s / ic.box
		for _, sh := range ic.shapes {
			v, ok := pierreSegs.Load(sh.d)
			if !ok {
				segs := parseSVGPath(sh.d)
				if sh.evenOdd {
					segs = evenOdd(segs)
				}
				v, _ = pierreSegs.LoadOrStore(sh.d, segs)
			}
			var p clip.Path
			p.Begin(gtx.Ops)
			for _, sg := range v.([]seg) {
				pt := func(i int) f32.Point { return sg.p[i].Mul(k) }
				switch sg.op {
				case 'M':
					p.MoveTo(pt(0))
				case 'L':
					p.LineTo(pt(0))
				case 'Q':
					p.QuadTo(pt(0), pt(1))
				case 'C':
					p.CubeTo(pt(0), pt(1), pt(2))
				case 'Z':
					p.Close()
				}
			}
			c := col
			if sh.own {
				c = rgb(sh.light)
				if dark {
					c = rgb(sh.dark)
				}
			}
			if sh.opacity > 0 {
				c.A = uint8(float32(c.A)*sh.opacity + 0.5)
			}
			if on.A != 0 && c.A != 0xff {
				c = over(c, on)
			}
			if sh.stroke > 0 {
				paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: sh.stroke * k}.Op())
				continue
			}
			paint.FillShape(gtx.Ops, c, clip.Outline{Path: p.End()}.Op())
		}
		return D{Size: image.Pt(int(s), int(s))}
	}
}

// evenOdd turns an even-odd shape into the non-zero one Gio fills: each
// subpath winds against the subpaths it sits in, so the ones inside an odd
// number of others are holes.
func evenOdd(segs []seg) []seg {
	var subs [][]seg
	for _, sg := range segs {
		if sg.op == 'M' || len(subs) == 0 {
			subs = append(subs, nil)
		}
		subs[len(subs)-1] = append(subs[len(subs)-1], sg)
	}
	polys := make([][]f32.Point, len(subs))
	for i, sub := range subs {
		polys[i] = flatten(sub)
	}
	var out []seg
	for i, sub := range subs {
		depth := 0
		for j := range subs {
			if i != j && len(polys[i]) > 0 && inside(polys[j], polys[i][0]) {
				depth++
			}
		}
		if (signedArea(polys[i]) > 0) != (depth%2 == 0) {
			sub = reverseSub(sub)
		}
		out = append(out, sub...)
	}
	return out
}

// flatten is a subpath's outline as points, its curves in eight steps.
func flatten(sub []seg) []f32.Point {
	var pts []f32.Point
	var at f32.Point
	for _, sg := range sub {
		switch sg.op {
		case 'M', 'L':
			at = sg.p[0]
			pts = append(pts, at)
		case 'Q':
			for i := 1; i <= 8; i++ {
				t := float32(i) / 8
				u := 1 - t
				pts = append(pts, at.Mul(u*u).Add(sg.p[0].Mul(2*u*t)).Add(sg.p[1].Mul(t*t)))
			}
			at = sg.p[1]
		case 'C':
			for i := 1; i <= 8; i++ {
				t := float32(i) / 8
				u := 1 - t
				pts = append(pts, at.Mul(u*u*u).Add(sg.p[0].Mul(3*u*u*t)).Add(sg.p[1].Mul(3*u*t*t)).Add(sg.p[2].Mul(t*t*t)))
			}
			at = sg.p[2]
		}
	}
	return pts
}

// area is a polygon's signed area: positive when it winds clockwise on
// screen.
func signedArea(pts []f32.Point) float32 {
	a := float32(0)
	for i := range pts {
		p, q := pts[i], pts[(i+1)%len(pts)]
		a += p.X*q.Y - q.X*p.Y
	}
	return a / 2
}

// inside reports whether pt is inside the polygon, by crossings.
func inside(poly []f32.Point, pt f32.Point) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a.Y > pt.Y) != (b.Y > pt.Y) && pt.X < (b.X-a.X)*(pt.Y-a.Y)/(b.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}

// reverseSub is a subpath drawn the other way round.
func reverseSub(sub []seg) []seg {
	closed := sub[len(sub)-1].op == 'Z'
	body := sub[1:]
	if closed {
		body = body[:len(body)-1]
	}
	ends := []f32.Point{sub[0].p[0]}
	for _, sg := range body {
		switch sg.op {
		case 'Q':
			ends = append(ends, sg.p[1])
		case 'C':
			ends = append(ends, sg.p[2])
		default:
			ends = append(ends, sg.p[0])
		}
	}
	out := []seg{{op: 'M', p: [3]f32.Point{ends[len(ends)-1]}}}
	for i := len(body) - 1; i >= 0; i-- {
		sg, from := body[i], ends[i]
		switch sg.op {
		case 'Q':
			out = append(out, seg{op: 'Q', p: [3]f32.Point{sg.p[0], from}})
		case 'C':
			out = append(out, seg{op: 'C', p: [3]f32.Point{sg.p[1], sg.p[0], from}})
		default:
			out = append(out, seg{op: 'L', p: [3]f32.Point{from}})
		}
	}
	if closed {
		out = append(out, seg{op: 'Z'})
	}
	return out
}

// fileIcon is the sprite icon T3 gives a file and the token that colours
// it: T3's own by name or extension (no token), then Pierre's complete
// set by name, by extension suffix, longest first, or its default.
func fileIcon(p string) (name, token string) {
	base := path.Base(p)
	low := strings.ToLower(base)
	if v, ok := t3FileNames[low]; ok {
		return v, ""
	}
	parts := strings.Split(low, ".")
	var suffixes []string
	for i := 1; i < len(parts); i++ {
		suffixes = append(suffixes, strings.Join(parts[i:], "."))
	}
	for _, sfx := range suffixes {
		if v, ok := t3Extensions[sfx]; ok {
			if v == "file-tree-builtin-image" {
				return v, "image"
			}
			return v, ""
		}
	}
	token = "default"
	if v, ok := pierreFileNames[low]; ok {
		token = v
	} else {
		for _, sfx := range suffixes {
			if v, ok := pierreCompleteExtensions[sfx]; ok {
				token = v
				break
			}
			if v, ok := pierreExtensions[sfx]; ok {
				token = v
				break
			}
		}
	}
	return "file-tree-builtin-" + token, token
}

// fileIconColors are the tokens' light and dark colours
// (PierreEntryIcon.tsx); an icon without a token takes the default's.
var fileIconColors = map[string][2]uint32{
	"astro": {0xa631be, 0xd568ea}, "babel": {0xd5a910, 0xffd452}, "bash": {0x199f43, 0x5ecc71},
	"biome": {0x1a85d4, 0x69b1ff}, "bootstrap": {0x693acf, 0x9d6afb}, "browserslist": {0xd5a910, 0xffd452},
	"bun": {0x594c5b, 0x79697b}, "c": {0x1a85d4, 0x69b1ff}, "claude": {0xd47628, 0xffa359},
	"cpp": {0x1a85d4, 0x69b1ff}, "css": {0x693acf, 0x9d6afb}, "database": {0xa631be, 0xd568ea},
	"default": {0x84848a, 0xadadb1}, "docker": {0x1a85d4, 0x69b1ff}, "eslint": {0x693acf, 0x9d6afb},
	"git": {0xff8c5b, 0xd5512f}, "go": {0x1ca1c7, 0x68cdf2}, "graphql": {0xd32a61, 0xff678d},
	"html": {0xd47628, 0xffa359}, "image": {0xd32a61, 0xff678d}, "javascript": {0xd5a910, 0xffd452},
	"json": {0xd47628, 0xffa359}, "markdown": {0x199f43, 0x5ecc71}, "mcp": {0x17a5af, 0x64d1db},
	"nextjs": {0x84848a, 0xadadb1}, "npm": {0xd52c36, 0xff6762}, "oxc": {0x1ca1c7, 0x68cdf2},
	"postcss": {0xd52c36, 0xff6762}, "prettier": {0x17a5af, 0x64d1db}, "python": {0x1a85d4, 0x69b1ff},
	"react": {0x1ca1c7, 0x68cdf2}, "ruby": {0xd52c36, 0xff6762}, "rust": {0xd47628, 0xffa359},
	"sass": {0xd32a61, 0xff678d}, "stylelint": {0x84848a, 0xadadb1}, "svelte": {0xd52c36, 0xff6762},
	"svg": {0xd47628, 0xffa359}, "svgo": {0x199f43, 0x5ecc71}, "swift": {0xd47628, 0xffa359},
	"table": {0x17a5af, 0x64d1db}, "tailwind": {0x1ca1c7, 0x68cdf2}, "terraform": {0x693acf, 0x9d6afb},
	"text": {0x84848a, 0xadadb1}, "typescript": {0x1a85d4, 0x69b1ff}, "vite": {0xa631be, 0xd568ea},
	"vscode": {0x1a85d4, 0x69b1ff}, "vue": {0x199f43, 0x5ecc71}, "wasm": {0x693acf, 0x9d6afb},
	"webpack": {0x1a85d4, 0x69b1ff}, "yml": {0xd52c36, 0xff6762}, "zig": {0xd47628, 0xffa359},
	"zip": {0xd47628, 0xffa359},
}

// fileIconColor is a token's colour in the current theme.
func fileIconColor(token string) color.NRGBA {
	c, ok := fileIconColors[token]
	if !ok {
		c = fileIconColors["default"]
	}
	if dark {
		return rgb(c[1])
	}
	return rgb(c[0])
}
