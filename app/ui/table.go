package ui

import (
	"fmt"
	"image"
	"strings"
	"unicode"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// T3's markdown tables (MarkdownTable.tsx, index.css .chat-markdown table;
// docs/ui/06-timeline.md §3.5): the table across the lane in CSS's
// automatic layout, each column at least its longest word and at most
// 24rem of text before the table's spare width is shared out; cells
// top-aligned, padded 7.2 by 12 (the header 8.8, semibold) but flush at
// the table's sides, with a rule at border/60 under each row. A table
// too wide for the lane scrolls sideways. Its copy button sits under it,
// offering Markdown or CSV.

func (th *Theme) mdTable(gtx C, doc *mdDoc, n *mdNode, st mdStyle) int {
	w := gtx.Constraints.Max.X
	cols := 0
	for _, r := range n.rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return 0
	}
	padX, capW := dp(gtx, 12), dp(gtx, 384)
	pads := func(c int) int {
		p := 2 * padX
		if c == 0 {
			p -= padX
		}
		if c == cols-1 {
			p -= padX
		}
		return p
	}
	minW, maxW := make([]int, cols), make([]int, cols)
	for _, r := range n.rows {
		for c, cell := range r {
			minW[c] = max(minW[c], th.longestWord(gtx, cell))
			maxW[c] = max(maxW[c], min(capW, th.layoutInline(gtx, cell, 1<<20).size.X))
		}
	}
	sumMin, sumMax := 0, 0
	for c := range cols {
		minW[c] += pads(c)
		maxW[c] = max(maxW[c]+pads(c), minW[c])
		sumMin += minW[c]
		sumMax += maxW[c]
	}
	widths := make([]int, cols)
	switch {
	case sumMax <= w:
		// width: 100%: the spare width goes to the columns by their size.
		left := w
		for c := range cols {
			widths[c] = maxW[c] + (w-sumMax)*maxW[c]/max(1, sumMax)
			left -= widths[c]
		}
		widths[cols-1] += left
	case sumMin <= w:
		for c := range cols {
			widths[c] = minW[c] + (maxW[c]-minW[c])*(w-sumMin)/max(1, sumMax-sumMin)
		}
	default:
		copy(widths, minW)
	}
	total := 0
	for _, cw := range widths {
		total += cw
	}
	rule := over(faded(P.ContrastBorder, 0x99), st.on)
	m := op.Record(gtx.Ops)
	y := 0
	for ri, r := range n.rows {
		padY := dp(gtx, 7.2)
		if ri < n.header {
			padY = dp(gtx, 8.8)
		}
		rowH := 0
		x := 0
		for c := range cols {
			if c < len(r) {
				left := padX
				if c == 0 {
					left = 0
				}
				off := op.Offset(image.Pt(x+left, y+padY)).Push(gtx.Ops)
				d := th.inline(within(gtx, widths[c]-pads(c)), r[c], doc.links)
				off.Pop()
				rowH = max(rowH, d.Size.Y)
			}
			x += widths[c]
		}
		y += rowH + 2*padY
		hline(gtx, rule, y, 0, total)
		y += max(1, dp(gtx, 1))
	}
	call := m.Stop()
	if total > w {
		th.hscroll(gtx, fmt.Sprintf("table:%p", n), image.Pt(w, y+dp(gtx, 15)), total, func(gtx C) { call.Add(gtx.Ops) })
		y += dp(gtx, 15)
	} else {
		call.Add(gtx.Ops)
	}
	// The copy button, 2dp under the table, reaching 4dp past its edge.
	id := fmt.Sprintf("table:%p:copy", n)
	b := Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "copy", IconSize: 12, On: st.on}
	if th.copiedFor(gtx, "copy:"+id, 1200) {
		b.Icon = "check"
	}
	y += dp(gtx, 2)
	at := image.Pt(w+dp(gtx, 4)-dp(gtx, 24), y)
	bo := op.Offset(at).Push(gtx.Ops)
	th.T3Button(id, b)(gtx)
	tip := "Copy table"
	if b.Icon == "check" {
		tip = "Copied"
	}
	if th.tl.menu != id {
		th.tipAt(gtx, id, tip, image.Pt(dp(gtx, 24), dp(gtx, 24)), true, 0)
	}
	bo.Pop()
	if th.clicked(id) {
		th.toggleMenu(id)
	}
	switch th.rowMenu(gtx, id, at.Add(image.Pt(dp(gtx, 24), dp(gtx, 28))), []string{"Copy as Markdown", "Copy as CSV"}) {
	case 0:
		th.copyNow(gtx, "copy:"+id, tableMarkdown(n))
	case 1:
		th.copyNow(gtx, "copy:"+id, tableCSV(n))
	}
	return y + dp(gtx, 24)
}

// longestWord is a cell's min-content width: its widest word, since
// tables keep words whole (overflow-wrap: normal).
func (th *Theme) longestWord(gtx C, t *inlineText) int {
	w := 0
	for _, r := range t.runs {
		if r.chip > 0 {
			w = max(w, dp(gtx, 4*r.size))
			continue
		}
		pad := 0
		if r.code {
			pad = 2 * (dp(gtx, 0.35*r.size) + max(1, dp(gtx, 1)))
		}
		for _, word := range strings.FieldsFunc(r.text, unicode.IsSpace) {
			_, _, ww := th.shapeLine(r.font, dp(gtx, r.size), word, 1<<20, false)
			w = max(w, ww+pad)
		}
	}
	return w
}

// cellText is a cell's plain text, for copying.
func cellText(t *inlineText) string {
	var sb strings.Builder
	for _, r := range t.runs {
		switch {
		case r.brk:
			sb.WriteByte(' ')
		case r.chip > 0 && r.chip <= len(t.chips):
			sb.WriteString(t.chips[r.chip-1])
		default:
			sb.WriteString(r.text)
		}
	}
	return strings.TrimSpace(sb.String())
}

// tableMarkdown is T3's serializeTableElementToMarkdown.
func tableMarkdown(n *mdNode) string {
	var lines []string
	for ri, r := range n.rows {
		var cells []string
		for _, c := range r {
			cells = append(cells, strings.ReplaceAll(cellText(c), "|", `\|`))
		}
		lines = append(lines, "| "+strings.Join(cells, " | ")+" |")
		if ri == n.header-1 || n.header == 0 && ri == 0 {
			sep := make([]string, len(r))
			for i := range sep {
				sep[i] = "---"
			}
			lines = append(lines, "| "+strings.Join(sep, " | ")+" |")
		}
	}
	return strings.Join(lines, "\n")
}

// tableCSV is T3's serializeTableElementToCsv.
func tableCSV(n *mdNode) string {
	var lines []string
	for _, r := range n.rows {
		var cells []string
		for _, c := range r {
			s := cellText(c)
			if strings.ContainsAny(s, "\",\n") {
				s = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
			}
			cells = append(cells, s)
		}
		lines = append(lines, strings.Join(cells, ","))
	}
	return strings.Join(lines, "\n")
}

// copyNow puts s on the clipboard and shows copy button id's check.
func (th *Theme) copyNow(gtx C, id, s string) {
	th.copyTarget(id, s)
	th.in.clipOut = &s
	th.in.copied[id] = gtx.Now
	gtx.Execute(op.InvalidateCmd{})
}

// toggleMenu opens the timeline menu id, or closes it if it's open.
func (th *Theme) toggleMenu(id string) {
	if th.tl.menu == id {
		th.tl.menu = ""
		return
	}
	th.tl.menu = id
}

// rowMenu is the open timeline menu id: T3's menu with its top-right
// corner at at, one row per choice, over a layer that closes it on any
// other click. It returns the row picked this frame, or -1.
func (th *Theme) rowMenu(gtx C, id string, at image.Point, choices []string) int {
	if th.tl.menu != id {
		return -1
	}
	picked := -1
	for i := range choices {
		if th.clicked(fmt.Sprintf("%s:%d", id, i)) {
			picked = i
		}
	}
	for {
		e, ok := th.tl.menuClose.Update(gtx.Source)
		if !ok {
			break
		}
		if e.Kind == gesture.KindClick {
			th.tl.menu = ""
		}
	}
	if picked >= 0 {
		th.tl.menu = ""
		return picked
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 320), dp(gtx, 400))}
	menu := th.recorded(g, func(gtx C) D {
		return popupFrame(gtx, func(gtx C) D {
			return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
				var rows []layout.FlexChild
				for i, c := range choices {
					rows = append(rows, layout.Rigid(th.menuItem(fmt.Sprintf("%s:%d", id, i), func(gtx C) D {
						gtx.Constraints.Min.X = max(gtx.Constraints.Min.X, dp(gtx, 160)-dp(gtx, 8))
						return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
							return th.textLine(gtx, Txt{Size: 14, Color: P.PopoverForeground}, 20, c)
						})
					})))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			})
		})
	})
	// A click anywhere else closes the menu without doing anything else.
	far := image.Rect(-1<<15, -1<<15, 1<<15, 1<<15)
	cl := clip.Rect(far).Push(gtx.Ops)
	th.tl.menuClose.Add(gtx.Ops)
	pointer.CursorDefault.Add(gtx.Ops)
	cl.Pop()
	off := op.Offset(image.Pt(at.X-menu.Size.X, at.Y)).Push(gtx.Ops)
	menu.call.Add(gtx.Ops)
	off.Pop()
	op.Defer(gtx.Ops, m.Stop())
	return -1
}
