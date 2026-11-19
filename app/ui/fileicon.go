package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// File icons as T3 draws them (PierreEntryIcon.tsx, pierre-icons.ts;
// docs/ui/06-timeline.md §3.6): @pierre/trees' set and T3's own, picked by
// a file's name and extension, in its token's colour. ui/pierre.go holds
// the shapes and tables, from tools/pierre/gen.py.

// pierreIcon is one file icon: its box and its shapes.
type pierreIcon struct {
	box    float32
	shapes []pierreShape
}

// pierreShape is one path of a file icon: filled, or stroked stroke units
// wide; in the icon's colour, or its own light and dark ones when own; at
// alpha, opaque when 0; when fade is set, fading out from its first point
// to its second; filled by SVG's evenodd rule when evenOdd.
type pierreShape struct {
	d           string
	stroke      float32
	own         bool
	light, dark uint32
	alpha       float32
	fade        [4]float32
	evenOdd     bool
}

// fileIcon is T3's resolvePierreIconForEntry for a file at path: its
// icon's name and the token that colours it. T3's own picks carry no
// token (they colour themselves, or take the default's); a built-in image
// one takes "image".
func fileIcon(path string) (name, token string) {
	lower := strings.ToLower(path[strings.LastIndex(path, "/")+1:])
	// Every extension the name could have: "a.test.ts" has "test.ts", "ts".
	var exts []string
	for i, c := range lower {
		if c == '.' {
			exts = append(exts, lower[i+1:])
		}
	}
	own, ok := t3IconNames[lower]
	for i := 0; !ok && i < len(exts); i++ {
		own, ok = t3IconExtensions[exts[i]]
	}
	switch {
	case ok && own == "file-tree-builtin-image":
		return own, "image"
	case ok:
		return own, ""
	}
	token, ok := pierreNames[lower]
	for i := 0; !ok && i < len(exts); i++ {
		if token, ok = pierreComplete[exts[i]]; !ok {
			token, ok = pierreExtensions[exts[i]]
		}
	}
	if !ok {
		token = "default"
	}
	return "file-tree-builtin-" + token, token
}

// languageFile is T3's syntheticFileNameForLanguageId: a file name for a
// fence's language, to pick its icon by.
func languageFile(lang string) string {
	l := strings.ToLower(lang)
	if n, ok := languageFileNames[l]; ok {
		return n
	}
	if e, ok := languageExtensions[l]; ok {
		return "file." + e
	}
	return "file." + l
}

// specificIcon is T3's hasSpecificPierreIconForFileName: name has an icon
// of its own, not the default one.
func specificIcon(name string) bool {
	_, token := fileIcon(name)
	return token != "default"
}

// fileIconW draws path's file icon sizeDp square on surface on: its token's
// colour, or its shapes' own, the translucent ones composited on on.
func fileIconW(path string, sizeDp float32, on color.NRGBA) W {
	name, token := fileIcon(path)
	ic := pierreIcons[name]
	c, ok := pierreColors[token]
	if !ok {
		c = pierreColors["default"]
	}
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		k := s / ic.box
		for i, sh := range ic.shapes {
			col := c
			if sh.own {
				col = [2]uint32{sh.light, sh.dark}
			}
			fg := rgb(col[0])
			if dark {
				fg = rgb(col[1])
			}
			if sh.alpha > 0 {
				fg = over(alpha(fg, uint8(sh.alpha*255+0.5)), on)
			}
			segs := cachedPath(name+"#"+strconv.Itoa(i), sh.d)
			if sh.evenOdd {
				segs = cachedEvenOdd(name+"#"+strconv.Itoa(i), segs)
			}
			var p clip.Path
			p.Begin(gtx.Ops)
			addSegs(&p, segs, func(q f32.Point) f32.Point { return q.Mul(k) })
			var area clip.Op
			if sh.stroke > 0 {
				area = clip.Stroke{Path: p.End(), Width: sh.stroke * k}.Op()
			} else {
				area = clip.Outline{Path: p.End()}.Op()
			}
			st := area.Push(gtx.Ops)
			if sh.fade != [4]float32{} {
				paint.LinearGradientOp{Stop1: f32.Pt(sh.fade[0]*k, sh.fade[1]*k), Color1: fg,
					Stop2: f32.Pt(sh.fade[2]*k, sh.fade[3]*k), Color2: on}.Add(gtx.Ops)
			} else {
				paint.ColorOp{Color: fg}.Add(gtx.Ops)
			}
			paint.PaintOp{}.Add(gtx.Ops)
			st.Pop()
		}
		return D{Size: image.Pt(int(s), int(s))}
	}
}
