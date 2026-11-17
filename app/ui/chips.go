package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"unicode"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// A picture in text is a chip, like T3's web composer: ChipRune(n) and then
// chipPad figure spaces, which reserve the pill's width and don't wrap. The
// text keeps the run whole: the caret steps over it, Backspace and Delete
// take all of it, and a run broken some other way is cleared. drawChips
// paints the pill over it.

const (
	chipFill = ' ' // FIGURE SPACE: wide and non-breaking
	chipPad  = 7
)

// ChipRune is the rune that starts picture n's chip, ① to ⑨.
func ChipRune(n int) rune { return '①' + rune(n-1) }

// ChipN is the picture number r starts a chip for, or 0.
func ChipN(r rune) int {
	if r >= '①' && r <= '⑨' {
		return int(r-'①') + 1
	}
	return 0
}

// ChipText is picture n's whole chip.
func ChipText(n int) string { return string(ChipRune(n)) + strings.Repeat(string(chipFill), chipPad) }

const chipLen = 1 + chipPad // in runes

// HasChips reports whether s has a chip in it.
func HasChips(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return ChipN(r) != 0 })
}

// TrimText trims s's surrounding white space but not a chip's fill, which
// counts as a space and would leave a chip at the end broken.
func TrimText(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r != chipFill && unicode.IsSpace(r) })
}

// StripChips is s without its chips.
func StripChips(s string) string {
	spans := chipSpans(s)
	if len(spans) == 0 {
		return s
	}
	rs := []rune(s)
	var b strings.Builder
	at := 0
	for _, c := range spans {
		b.WriteString(string(rs[at:c.start]))
		at = c.end
	}
	b.WriteString(string(rs[at:]))
	return b.String()
}

// chipSpan is one chip's rune range [start, end) and number.
type chipSpan struct{ start, end, n int }

// chipSpans finds the whole chips in s.
func chipSpans(s string) []chipSpan {
	rs := []rune(s)
	var out []chipSpan
	for i := 0; i < len(rs); i++ {
		n := ChipN(rs[i])
		if n == 0 || i+chipLen > len(rs) {
			continue
		}
		whole := true
		for j := i + 1; j < i+chipLen; j++ {
			whole = whole && rs[j] == chipFill
		}
		if whole {
			out = append(out, chipSpan{i, i + chipLen, n})
			i += chipLen - 1
		}
	}
	return out
}

// fixChips clears broken chips: a chip rune without all its fill, and fill
// that's not in a chip. It reports whether it changed anything.
func fixChips(ed *widget.Editor) bool {
	txt := ed.Text()
	if !HasChips(txt) && !strings.ContainsRune(txt, chipFill) {
		return false
	}
	rs := []rune(txt)
	keep := make([]bool, len(rs))
	for i := range keep {
		keep[i] = ChipN(rs[i]) == 0 && rs[i] != chipFill
	}
	for _, c := range chipSpans(txt) {
		for i := c.start; i < c.end; i++ {
			keep[i] = true
		}
	}
	changed := false
	// Delete from the end so earlier indexes hold.
	for i := len(rs) - 1; i >= 0; i-- {
		if keep[i] {
			continue
		}
		j := i
		for j > 0 && !keep[j-1] {
			j--
		}
		s, e := ed.Selection()
		ed.SetCaret(j, i+1)
		ed.Insert("")
		shift := func(x int) int {
			switch {
			case x > i:
				return x - (i + 1 - j)
			case x > j:
				return j
			}
			return x
		}
		ed.SetCaret(shift(s), shift(e))
		changed = true
		i = j
	}
	return changed
}

// chipKeys makes Backspace and Delete take a whole chip next to the caret.
func (th *Theme) chipKeys(gtx C, id string, ed *widget.Editor) {
	for {
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameDeleteBackward, Optional: key.ModShift},
			key.Filter{Focus: ed, Name: key.NameDeleteForward, Optional: key.ModShift})
		if !ok {
			return
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press || ed.ReadOnly {
			continue
		}
		s, e := ed.Selection()
		if s == e {
			for _, c := range chipSpans(ed.Text()) {
				if ke.Name == key.NameDeleteBackward && s == c.end || ke.Name == key.NameDeleteForward && s == c.start {
					s, e = c.start, c.end
				}
			}
		}
		if s != e {
			ed.SetCaret(s, e)
			ed.Insert("")
		} else if ke.Name == key.NameDeleteBackward {
			ed.Delete(-1)
		} else {
			ed.Delete(1)
		}
		th.push(Action{ID: id, Kind: "change", Text: ed.Text()})
	}
}

// snapChips keeps the caret and the selection's ends out of chips. A caret
// that stepped in from one side lands on the other.
func (th *Theme) snapChips(ed *widget.Editor) {
	s, e := ed.Selection()
	old := th.in.caret
	snap := func(x, from int) int {
		for _, c := range chipSpans(ed.Text()) {
			if x <= c.start || x >= c.end {
				continue
			}
			switch {
			case from <= c.start:
				return c.end
			case from >= c.end:
				return c.start
			case x-c.start <= c.end-x:
				return c.start
			}
			return c.end
		}
		return x
	}
	ns, ne := snap(s, old[0]), snap(e, old[1])
	if ns != s || ne != e {
		ed.SetCaret(ns, ne)
	}
	th.in.caret = [2]int{ns, ne}
}

// textRegions is what drawChips needs from an editor or a selectable.
type textRegions interface {
	Text() string
	Regions(start, end int, regions []widget.Region) []widget.Region
}

// Chip colours, shaped like T3's context chips (the accent at 11% for the
// fill, mixed 34% into the border and 22% into the text) but in the app's
// blue: T3's image red read as an error.
var (
	chipAccent = ColAccent
	chipBorder = mix(chipAccent, ColBorderHi, 0.34)
	chipText   = mix(chipAccent, ColText, 0.22)
)

func mix(a, b color.NRGBA, t float32) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(float32(x)*t + float32(y)*(1-t) + 0.5) }
	return color.NRGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 0xff}
}

// drawChips paints a pill over each chip in t: the image icon and
// label(n). bg is what the text sits on, scroll how far it has scrolled.
func (th *Theme) drawChips(gtx C, t textRegions, size image.Point, scroll image.Point, bg color.NRGBA, sizeSp float32, label func(n int) string) {
	spans := chipSpans(t.Text())
	if len(spans) == 0 {
		return
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	em := gtx.Metric.PxPerSp * sizeSp
	var regs []widget.Region
	for _, c := range spans {
		regs = t.Regions(c.start, c.end, regs[:0])
		if len(regs) == 0 {
			continue
		}
		b := regs[0].Bounds.Sub(scroll)
		for _, r := range regs[1:] {
			if r.Bounds.Min.Y == regs[0].Bounds.Min.Y {
				b = b.Union(r.Bounds.Sub(scroll))
			}
		}
		// Hide the rune and its fill; the circled digit dips below the line.
		fill(gtx, bg, image.Rect(b.Min.X, b.Min.Y-dp(gtx, 2), b.Max.X, b.Max.Y+dp(gtx, 3)), 0)
		h := min(int(1.41*em+0.5), b.Dy())
		top := b.Min.Y + (b.Dy()-h)/2
		pill := image.Rect(b.Min.X+dp(gtx, 1), top, b.Max.X-dp(gtx, 1), top+h)
		rad := int(0.5*em + 0.5)
		fill(gtx, mix(chipAccent, bg, 0.11), pill, rad)
		strokeRect(gtx, chipBorder, pill, rad, float32(max(1, dp(gtx, 1))))
		padX := int(0.5*em + 0.5)
		iconPx := int(1.17*0.86*em + 0.5)
		off := op.Offset(image.Pt(pill.Min.X+padX, pill.Min.Y+(h-iconPx)/2)).Push(gtx.Ops)
		id := DrawIcon(gtx, IconImage, chipText, float32(iconPx)/gtx.Metric.PxPerDp)
		off.Pop()
		m := op.Record(gtx.Ops)
		lg := gtx
		lg.Constraints.Min = image.Point{}
		lg.Constraints.Max.X = max(0, pill.Dx()-2*padX-id.Size.X-int(0.33*em))
		ld := th.Text(lg, Txt{Size: unit.Sp(sizeSp * 0.86), Color: chipText, Font: UI(font.Medium), MaxLines: 1}, label(c.n))
		call := m.Stop()
		lo := op.Offset(image.Pt(pill.Min.X+padX+id.Size.X+int(0.33*em), pill.Min.Y+(h-ld.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		lo.Pop()
	}
}

// ImageLabel is a composer picture's name in its chip.
func ImageLabel(n int) string { return "Image " + strconv.Itoa(n) }

// editorScroll is how far the editor has scrolled. Regions are in document
// coordinates and the caret's are on screen, so the difference is the scroll.
func editorScroll(ed *widget.Editor, size image.Point) image.Point {
	c, _ := ed.Selection()
	n := ed.Len()
	if n == 0 {
		return image.Point{}
	}
	if all := ed.Regions(n-1, n, nil); len(all) > 0 && all[len(all)-1].Bounds.Max.Y <= size.Y {
		return image.Point{} // it all fits, so nothing scrolled
	}
	start, end, atEnd := c, c+1, false
	if c >= n {
		start, end, atEnd = n-1, n, true
	}
	regs := ed.Regions(start, end, nil)
	if len(regs) == 0 {
		return image.Point{}
	}
	rg := regs[0]
	x := rg.Bounds.Min.X
	if atEnd {
		rg = regs[len(regs)-1]
		x = rg.Bounds.Max.X
	}
	cc := ed.CaretCoords()
	return image.Pt(x-int(cc.X+0.5), rg.Bounds.Min.Y+rg.Baseline-int(cc.Y+0.5))
}

// chipText lays out a sent message's text with its pictures as chips. It's
// selectable, which the chips need for their positions.
func (th *Theme) chipText(gtx C, id, s string, labels []string) D {
	d, sel := th.selectableSel(gtx, id, s, UI(font.Normal), SizeBody, ColText, 1.42)
	th.drawChips(gtx, sel, d.Size, image.Point{}, ColFrameHover, float32(SizeBody), func(n int) string {
		if n <= len(labels) {
			return labels[n-1]
		}
		return ImageLabel(n)
	})
	return d
}

// selectable lays out s as text the user can select and copy, kept per id.
func (th *Theme) selectable(gtx C, id, s string, f font.Font, size unit.Sp, c color.NRGBA, lineH float32) D {
	d, _ := th.selectableSel(gtx, id, s, f, size, c, lineH)
	return d
}

func (th *Theme) selectableSel(gtx C, id, s string, f font.Font, size unit.Sp, c color.NRGBA, lineH float32) (D, *widget.Selectable) {
	if th.in.selectables == nil {
		th.in.selectables = map[string]*widget.Selectable{}
	}
	sel := th.in.selectables[id]
	if sel == nil {
		sel = &widget.Selectable{LineHeightScale: lineH}
		th.in.selectables[id] = sel
	}
	if sel.Text() != s {
		sel.SetText(s)
	}
	col := func(c color.NRGBA) op.CallOp {
		m := op.Record(gtx.Ops)
		paint.ColorOp{Color: c}.Add(gtx.Ops)
		return m.Stop()
	}
	return sel.Layout(gtx, th.Shaper, f, size, col(c), col(alpha(ColAccent, 0x60))), sel
}

// plural is "1 image", "2 images".
func plural(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return strconv.Itoa(n) + " " + what + "s"
}
