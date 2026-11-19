package ui

import (
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The diff viewer's own sprite icons (docs/ui/13-diff.md §2.12), beside
// the file icons in pierre.go: the change symbols and the rename arrow
// from @pierre/diffs's sprite, the file tree's chevron and dot from
// @pierre/trees's (Apache-2.0, Pierre Computer Company), as T3's
// getDiffFileInput and pierre-icons bundles carry them.
var viewerIcons = map[string]pierreIcon{
	"diffs-icon-symbol-modified":   {16, []pierreShape{{d: `M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706s.826.607 1.706.802c.898.2 2.091.288 3.704.288s2.806-.088 3.704-.288c.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5s-2.806.088-3.704.288c-.88.196-1.381.478-1.706.802s-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8M0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m8 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6`}}},
	"diffs-icon-symbol-added":      {16, []pierreShape{{d: `M8 4a.75.75 0 0 1 .75.75v2.5h2.5a.75.75 0 0 1 0 1.5h-2.5v2.5a.75.75 0 0 1-1.5 0v-2.5h-2.5a.75.75 0 0 1 0-1.5h2.5v-2.5A.75.75 0 0 1 8 4`}, {d: `M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8`}}},
	"diffs-icon-symbol-deleted":    {16, []pierreShape{{d: `M4 8a.75.75 0 0 1 .75-.75h6.5a.75.75 0 0 1 0 1.5h-6.5A.75.75 0 0 1 4 8`}, {d: `M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8`}}},
	"diffs-icon-symbol-moved":      {16, []pierreShape{{d: `M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8`}, {d: `M8.495 4.695a.75.75 0 0 0-.05 1.06L10.486 8l-2.041 2.246a.75.75 0 0 0 1.11 1.008l2.5-2.75a.75.75 0 0 0 0-1.008l-2.5-2.75a.75.75 0 0 0-1.06-.051m-4 0a.75.75 0 0 0-.05 1.06l2.044 2.248-1.796 1.995a.75.75 0 0 0 1.114 1.004l2.25-2.5a.75.75 0 0 0-.002-1.007l-2.5-2.75a.75.75 0 0 0-1.06-.05`}}},
	"diffs-icon-arrow-right-short": {16, []pierreShape{{d: `M8.47 4.22a.75.75 0 0 0 0 1.06l1.97 1.97H3.75a.75.75 0 0 0 0 1.5h6.69l-1.97 1.97a.75.75 0 1 0 1.06 1.06l3.25-3.25a.75.75 0 0 0 0-1.06L9.53 4.22a.75.75 0 0 0-1.06 0`}}},
	"file-tree-icon-chevron":       {16, []pierreShape{{d: `M12.4697 5.46973C12.7626 5.17684 13.2374 5.17684 13.5303 5.46973C13.8232 5.76262 13.8232 6.23738 13.5303 6.53028L8.53028 11.5303C8.23738 11.8232 7.76262 11.8232 7.46973 11.5303L2.46973 6.53028C2.17684 6.23738 2.17684 5.76262 2.46973 5.46973C2.76262 5.17684 3.23738 5.17684 3.53028 5.46973L8 9.93946L12.4697 5.46973Z`}}},
	"file-tree-icon-dot":           {6, []pierreShape{{d: `M0 3a3 3 0 1 0 6 0a3 3 0 1 0 -6 0Z`}}},
}

// spriteW draws a sprite icon, the viewer's or a file's, sizeDp square in
// col; its own coloured shapes keep their colours, and translucent ones
// are composited onto on, the surface under it.
func spriteW(name string, col, on color.NRGBA, sizeDp float32) W {
	ic, ok := viewerIcons[name]
	if !ok {
		ic = pierreIcons[name]
	}
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		k := s / ic.box
		for i, sh := range ic.shapes {
			fg := col
			if sh.own {
				fg = rgb(sh.light)
				if dark {
					fg = rgb(sh.dark)
				}
			}
			if sh.alpha > 0 {
				fg.A = uint8(float32(fg.A)*sh.alpha + 0.5)
			}
			if fg.A != 0xff && on.A != 0 {
				fg = over(fg, on)
			}
			key := name + "#" + strconv.Itoa(i)
			segs := cachedPath(key, sh.d)
			if sh.evenOdd {
				segs = cachedEvenOdd(key, segs)
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

// iconTone is a file icon token's colour in the current theme
// (PierreEntryIcon.tsx), the default's for an unknown token.
func iconTone(token string) color.NRGBA {
	c, ok := pierreColors[token]
	if !ok {
		c = pierreColors["default"]
	}
	if dark {
		return rgb(c[1])
	}
	return rgb(c[0])
}
