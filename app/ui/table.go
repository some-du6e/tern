package ui

import (
	"fmt"
	"image"
	"io"
	"strings"
	"time"
	"unicode"

	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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
	id := th.blockID(st)
	cols := 0
	for _, r := range n.rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return 0
	}
	padX := dp(gtx, 12)
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
	minW, maxW := th.tableColumns(gtx, n, cols, pads)
	sumMin, sumMax := 0, 0
	for c := range cols {
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
		th.hscroll(gtx, id, image.Pt(w, y+dp(gtx, 15)), total, func(gtx C) { call.Add(gtx.Ops) })
		y += dp(gtx, 15)
	} else {
		call.Add(gtx.Ops)
	}
	// The copy button, 2dp under the table, reaching 4dp past its edge.
	id += ":copy"
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
	switch th.rowMenu(gtx, id, at.Add(image.Pt(dp(gtx, 24), dp(gtx, 28))), false, []menuChoice{{label: "Copy as Markdown"}, {label: "Copy as CSV"}}) {
	case 0:
		th.copyNow(gtx, "copy:"+id, tableMarkdown(n))
	case 1:
		th.copyNow(gtx, "copy:"+id, tableCSV(n))
	}
	return y + dp(gtx, 24)
}

// tableColumns is each column's least and most width, its cells' padding
// in: its longest word and its widest line, capped at 24rem. They're
// measured once for the table at a scale.
func (th *Theme) tableColumns(gtx C, n *mdNode, cols int, pads func(int) int) (minW, maxW []int) {
	if c := n.cols; c != nil && c.scale == gtx.Metric.PxPerDp {
		return c.min, c.max
	}
	capW := dp(gtx, 384)
	minW, maxW = make([]int, cols), make([]int, cols)
	for _, r := range n.rows {
		for c, cell := range r {
			minW[c] = max(minW[c], th.longestWord(gtx, cell))
			maxW[c] = max(maxW[c], min(capW, th.layoutInline(gtx, cell, 1<<20).size.X))
		}
	}
	for c := range cols {
		minW[c] += pads(c)
		maxW[c] = max(maxW[c]+pads(c), minW[c])
	}
	n.cols = &tableCols{gtx.Metric.PxPerDp, minW, maxW}
	return minW, maxW
}

// tableCols is a table's measured columns, at a scale.
type tableCols struct {
	scale    float32
	min, max []int
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
		if r.file != nil {
			// A file chip stays on one line: its icon, its gap, its label.
			_, _, lw := th.shapeLine(r.font, dp(gtx, r.size), r.file.label, 1<<20, false)
			w = max(w, dp(gtx, r.size)+dp(gtx, 0.2*r.size)+lw)
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
		case r.file != nil:
			sb.WriteString(r.file.label)
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
	gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(s))})
	if th.in.copied == nil {
		th.in.copied = map[string]time.Time{}
	}
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

// menuChoice is a timeline menu's row: its label, and the warning a risky
// one carries, as a triangle-alert before it and its tooltip.
type menuChoice struct{ label, warning string }

// rowMenu is the open timeline menu id: T3's menu with its top-right
// corner at at, or its bottom-right there when up, one row per choice,
// over a layer that closes it on any other click. It returns the row
// picked this frame, or -1.
func (th *Theme) rowMenu(gtx C, id string, at image.Point, up bool, choices []menuChoice) int {
	if th.tl.menu != id {
		return -1
	}
	// The rows' clicks are taken before the rows are drawn, so a pick is
	// this frame's and never reaches core.
	picked := -1
	for i := range choices {
		row := fmt.Sprintf("%s:%d", id, i)
		th.events(gtx, row)
		if th.clicked(row) {
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
					row := fmt.Sprintf("%s:%d", id, i)
					rows = append(rows, layout.Rigid(func(gtx C) D {
						d := th.menuItem(row, func(gtx C) D {
							gtx.Constraints.Min.X = max(gtx.Constraints.Min.X, dp(gtx, 160)-dp(gtx, 8))
							return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
								label := func(gtx C) D {
									return th.textLine(gtx, Txt{Size: 14, Color: P.PopoverForeground}, 20, c.label)
								}
								if c.warning == "" {
									return label(gtx)
								}
								// The 12dp icon at 80%, its -2dp margins, then the 8dp gap.
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx C) D {
										defer op.Offset(image.Pt(-dp(gtx, 2), 0)).Push(gtx.Ops).Pop()
										defer paint.PushOpacity(gtx.Ops, 0.8).Pop()
										d := LucideW("triangle-alert", P.Warning, 12)(gtx)
										d.Size.X -= dp(gtx, 4)
										return d
									}),
									layout.Rigid(Spacer(8, 0)),
									layout.Rigid(label),
								)
							})
						})(gtx)
						th.tipAt(gtx, row, c.warning, d.Size, true, 0)
						return d
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			})
		})
	})
	// A click anywhere else closes the menu without doing anything else:
	// the layer is deferred with the menu, so it covers the whole window.
	far := image.Rect(-1<<15, -1<<15, 1<<15, 1<<15)
	cl := clip.Rect(far).Push(gtx.Ops)
	th.tl.menuClose.Add(gtx.Ops)
	pointer.CursorDefault.Add(gtx.Ops)
	cl.Pop()
	y := at.Y
	if up {
		y -= menu.Size.Y
	}
	off := op.Offset(image.Pt(at.X-menu.Size.X, y)).Push(gtx.Ops)
	menu.call.Add(gtx.Ops)
	off.Pop()
	op.Defer(gtx.Ops, m.Stop())
	return -1
}
