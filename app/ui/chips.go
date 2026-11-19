package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"unicode"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"golang.org/x/image/math/fixed"
)

// A picture in text is a chip, like T3's web composer: ChipRune(n), then
// fill that reserves the chip's width and doesn't wrap, figure spaces and
// for the last few dp narrow no-break spaces. The text keeps the run
// whole: the caret steps over it, Backspace and Delete take all of it, and
// a run broken some other way is cleared. drawChips paints the chip over
// it.

const (
	chipFill  = '\u2007' // FIGURE SPACE: wide and non-breaking
	chipFine  = '\u202f' // NARROW NO-BREAK SPACE: the fine steps
	chipPad   = 7        // ChipText's fill
	chipRunes = 0xE000   // ChipRune(n) is this plus n, in the private use area
	// MaxChips is how many pictures one message takes (chatAttachment.ts).
	MaxChips = 100
)

// ChipRune is the rune that starts picture n's chip.
func ChipRune(n int) rune { return chipRunes + rune(n) }

// ChipN is the picture number r starts a chip for, or 0.
func ChipN(r rune) int {
	if r > chipRunes && r <= chipRunes+MaxChips {
		return int(r - chipRunes)
	}
	return 0
}

func isFill(r rune) bool { return r == chipFill || r == chipFine }

// ChipText is picture n's chip with a set fill, for text whose chips
// aren't sized to their labels.
func ChipText(n int) string { return string(ChipRune(n)) + strings.Repeat(string(chipFill), chipPad) }

// HasChips reports whether s has a chip in it.
func HasChips(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return ChipN(r) != 0 })
}

// Chips is the numbers of the pictures s has chips for, in order.
func Chips(s string) []int {
	var out []int
	for _, c := range chipSpans(s) {
		out = append(out, c.n)
	}
	return out
}

// ReplaceChips is s with each chip swapped for with(n).
func ReplaceChips(s string, with func(n int) string) string {
	spans := chipSpans(s)
	if len(spans) == 0 {
		return s
	}
	rs := []rune(s)
	var b strings.Builder
	at := 0
	for _, c := range spans {
		b.WriteString(string(rs[at:c.start]))
		b.WriteString(with(c.n))
		at = c.end
	}
	b.WriteString(string(rs[at:]))
	return b.String()
}

// TrimText trims s's surrounding white space but not a chip's fill, which
// counts as a space and would leave a chip at the end broken.
func TrimText(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return !isFill(r) && unicode.IsSpace(r) })
}

// StripChips is s without its chips.
func StripChips(s string) string { return ReplaceChips(s, func(int) string { return "" }) }

// chipSpan is one chip's rune range [start, end) and number.
type chipSpan struct{ start, end, n int }

// chipSpans finds the whole chips in s: a chip rune and all the fill
// after it.
func chipSpans(s string) []chipSpan {
	rs := []rune(s)
	var out []chipSpan
	for i := 0; i < len(rs); i++ {
		n := ChipN(rs[i])
		if n == 0 {
			continue
		}
		j := i + 1
		for j < len(rs) && isFill(rs[j]) {
			j++
		}
		if j > i+1 {
			out = append(out, chipSpan{i, j, n})
			i = j - 1
		}
	}
	return out
}

// fixChips clears broken chips: a chip rune without its fill, and fill
// that's not in a chip. It reports whether it changed anything.
func fixChips(ed *widget.Editor) bool {
	txt := ed.Text()
	if !HasChips(txt) && !strings.ContainsFunc(txt, isFill) {
		return false
	}
	rs := []rune(txt)
	keep := make([]bool, len(rs))
	for i := range keep {
		keep[i] = ChipN(rs[i]) == 0 && !isFill(rs[i])
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

func mix(a, b color.NRGBA, t float32) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(float32(x)*t + float32(y)*(1-t) + 0.5) }
	return color.NRGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 0xff}
}

// drawChips paints T3's image chip over each chip in t (ContextChip.tsx,
// ImageChipButton in contextChipParts.tsx): no pill, the picture itself at
// 1em, rounded 4, then 0.2em on its name in medium and 0.33em on, its size
// at 10, all in --context-chip-image, on the text's baseline. chip gives
// chip n's picture, its name and size; one still loading shows lucide's
// image. bg is what the text sits on, to hide the run the chip holds;
// scroll is how far the text has scrolled. Live, a chip is a button: its
// tooltip is the name over the size, and a click opens the picture.
func (th *Theme) drawChips(gtx C, t textRegions, size image.Point, scroll image.Point, bg color.NRGBA, sizeSp float32, chip func(n int) Thumb, live bool) {
	spans := chipSpans(t.Text())
	if len(spans) == 0 {
		return
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	em := gtx.Metric.PxPerSp * sizeSp
	ink := P.ContextChipImage
	label := Txt{Size: unit.Sp(sizeSp), Color: ink, Font: UI(font.Medium), MaxLines: 1}
	small := Txt{Size: 10, LineH: 14.0 / 10, Color: ink, Font: UI(font.Medium), MaxLines: 1}
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
		base := b.Max.Y - regs[0].Baseline
		// Hide the rune and its fill.
		fill(gtx, bg, image.Rect(b.Min.X, b.Min.Y-dp(gtx, 2), b.Max.X, b.Max.Y+dp(gtx, 3)), 0)
		pic := chip(c.n)
		name, sz := chipLabel(pic)
		icon := int(em + 0.5)
		// The picture centres on the chip's line, the text's ascent over
		// its descent (line-height: normal).
		mid := base - int(0.36*em+0.5)
		off := op.Offset(image.Pt(b.Min.X, mid-icon/2)).Push(gtx.Ops)
		if !drawThumb(gtx, pic, icon, dp(gtx, 4)) {
			LucideW("image", ink, sizeSp)(gtx)
		}
		off.Pop()
		x := b.Min.X + icon + int(0.2*em+0.5)
		for i, part := range []struct {
			t Txt
			s string
		}{{label, name}, {small, sz}} {
			if part.s == "" {
				continue
			}
			if i > 0 {
				x += int(0.33*em + 0.5)
			}
			m := op.Record(gtx.Ops)
			lg := gtx
			lg.Constraints = layout.Constraints{Max: image.Pt(max(0, b.Max.X-x), b.Dy())}
			ld := th.Text(lg, part.t, part.s)
			call := m.Stop()
			lo := op.Offset(image.Pt(x, base-(ld.Size.Y-ld.Baseline))).Push(gtx.Ops)
			call.Add(gtx.Ops)
			lo.Pop()
			x += ld.Size.X
		}
		if !live {
			continue
		}
		id := "chip:" + strconv.Itoa(c.n)
		co := op.Offset(b.Min).Push(gtx.Ops)
		clicked, hovered := th.localClick(gtx, id, b.Size())
		th.tipShow(gtx, id, hovered, strings.TrimSpace(pic.Name+"\n"+sz), b.Size(), true, 0)
		co.Pop()
		if clicked && pic.ok {
			th.openPreview(pic)
		}
	}
}

// chipLabel is what an image chip says: the file's name cut in the middle
// to 36 characters (middleTruncateAttachmentName) and its size.
func chipLabel(t Thumb) (name, size string) {
	if t.Size > 0 {
		size = FormatSize(t.Size)
	}
	return MiddleTruncate(t.Name, 36), size
}

// ImageChip is picture n's chip in the composer, named name and size
// bytes big. Its fill comes as close as the fill's steps allow to what
// drawChips draws over it, so the text after it sits where T3's does.
func (th *Theme) ImageChip(n int, name string, size int) string {
	// Measured at ten times the prompt's 14, for the fractions.
	const em = 140
	nm, sz := chipLabel(Thumb{Name: name, Size: size})
	want := float32(1.2 * em)
	if nm != "" {
		want += th.advance(UI(font.Medium), em, nm)
	}
	if sz != "" {
		want += 0.33*em + th.advance(UI(font.Medium), em*10/promptSize, sz)
	}
	have := th.advance(UI(font.Normal), em, string(ChipRune(n)))
	wide, fine := th.advance(UI(font.Normal), em, string(chipFill)), th.advance(UI(font.Normal), em, string(chipFine))
	bestA, bestB, bestErr := 1, 0, float32(1<<30)
	for a := 0; wide > 0 && float32(a-1)*wide <= want-have; a++ {
		b := 0
		if rest := want - have - float32(a)*wide; fine > 0 && rest > 0 {
			b = int(rest/fine + 0.5)
		}
		if a+b == 0 {
			continue
		}
		if e := abs32(have + float32(a)*wide + float32(b)*fine - want); e < bestErr {
			bestA, bestB, bestErr = a, b, e
		}
	}
	return string(ChipRune(n)) + strings.Repeat(string(chipFill), bestA) + strings.Repeat(string(chipFine), bestB)
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// advance is how wide s sets on one line in f at px pixels to the em.
func (th *Theme) advance(f font.Font, px int, s string) float32 {
	th.Shaper.LayoutString(text.Parameters{Font: f, PxPerEm: fixed.I(px), MaxLines: 1, MaxWidth: 1 << 24}, s)
	var w fixed.Int26_6
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		w = max(w, g.X+g.Advance)
	}
	return float32(w) / 64
}

// MiddleTruncate is middleTruncateAttachmentName (composerInlineChip.ts):
// name cut to most runes, the end kept, with "…" for what's gone.
func MiddleTruncate(name string, most int) string {
	rs := []rune(name)
	switch {
	case len(rs) <= most:
		return name
	case most <= 0:
		return ""
	case most == 1:
		return "…"
	}
	avail := most - 1
	tail := (avail + 1) / 2
	if avail >= 18 {
		tail = 14
	}
	tail = min(avail-1, tail)
	return string(rs[:avail-tail]) + "…" + string(rs[len(rs)-tail:])
}

// FormatSize is formatAttachmentSize: "N KB", rounded up and at least 1,
// or from 1024 KB on "N.N MB".
func FormatSize(bytes int) string {
	kb := max(1, (bytes+1023)/1024)
	if kb >= 1024 {
		return strconv.FormatFloat(float64(bytes)/(1024*1024), 'f', 1, 64) + " MB"
	}
	return strconv.Itoa(kb) + " KB"
}

// chipPictures is drawChips' chip for the composer: picture n among thumbs.
func chipPictures(thumbs []Thumb) func(n int) Thumb {
	return func(n int) Thumb {
		for _, t := range thumbs {
			if t.N == n {
				return t
			}
		}
		return Thumb{}
	}
}

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
	th.drawChips(gtx, sel, d.Size, image.Point{}, ColFrameHover, float32(SizeBody), func(n int) Thumb {
		if n <= len(labels) {
			return Thumb{Name: labels[n-1]}
		}
		return Thumb{}
	}, false)
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
