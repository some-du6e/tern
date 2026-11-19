package ui

import (
	"image"
	"math"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
)

// CompactCount writes a line count as T3's DiffStatLabel does
// (formatCompactDiffCount): as is below 1,000, then thousands with one
// decimal below 10k and none after (a trailing ".0" dropped), then
// millions and billions the same way.
func CompactCount(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	unit, div := "k", 1e3
	switch {
	case n >= 999_500_000:
		unit, div = "b", 1e9
	case n >= 999_500:
		unit, div = "m", 1e6
	}
	v := float64(n) / div
	if v < 10 {
		return strings.TrimSuffix(strconv.FormatFloat(v, 'f', 1, 64), ".0") + unit
	}
	return strconv.Itoa(int(math.Floor(v+0.5))) + unit
}

// diffStat is T3's DiffStatLabel (DiffStatLabel.tsx): "+a" then "-d" in
// mono, in --diff-addition-foreground and --diff-deletion-foreground,
// both always shown. Inline sets them 4dp apart; aligned right-aligns
// each in a 4ch column, the columns 8dp apart.
func (th *Theme) diffStat(gtx C, add, del int, aligned bool, size unit.Sp, lineH float32) D {
	a := Txt{Size: size, LineH: lineH / float32(size), Font: Mono(), MaxLines: 1, Color: P.DiffAdditionForeground}
	d := a
	d.Color = P.DiffDeletionForeground
	as, ds := "+"+CompactCount(add), "-"+CompactCount(del)
	gtx.Constraints.Min = image.Point{}
	if !aligned {
		ad := th.Text(gtx, a, as)
		x := ad.Size.X + dp(gtx, 4)
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		dd := th.Text(gtx, d, ds)
		return D{Size: image.Pt(x+dd.Size.X, max(ad.Size.Y, dd.Size.Y))}
	}
	col := int(4*th.monoAdvance(gtx, size) + 0.5)
	a.Align, d.Align = text.End, text.End
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = col, col
	ad := th.Text(gtx, a, as)
	defer op.Offset(image.Pt(col+dp(gtx, 8), 0)).Push(gtx.Ops).Pop()
	th.Text(gtx, d, ds)
	return D{Size: image.Pt(2*col+dp(gtx, 8), ad.Size.Y)}
}

// monoAdvance is 1ch of the code font at size, in px: the advance of "0".
func (th *Theme) monoAdvance(gtx C, size unit.Sp) float32 {
	k := chKey{size, gtx.Metric.PxPerSp}
	if w, ok := th.dv.ch[k]; ok {
		return w
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	w := th.Text(g, Txt{Size: size, Font: Mono(), MaxLines: 1}, "0000000000").Size.X
	m.Stop()
	if th.dv.ch == nil {
		th.dv.ch = map[chKey]float32{}
	}
	th.dv.ch[k] = float32(w) / 10
	return th.dv.ch[k]
}

// chKey names a measured 1ch: the font size, at a pixel density.
type chKey struct {
	size  unit.Sp
	pxPer float32
}
