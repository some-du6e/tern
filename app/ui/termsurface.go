package ui

import (
	"image"
	"image/color"
	"io"
	"math"
	"runtime"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"golang.org/x/image/math/fixed"
)

// A terminal's surface, ported from T3's canvas renderer over libghostty-vt
// (apps/web/src/terminal/ghostty; docs/ui/09-terminal.md §3.5–3.9 and
// §5.6–5.9): the cells in the terminal font on T3's terminal theme, the
// selection, decorations, cursor, links and scrollbar, and the keys,
// mouse and wheel going to the shell. Lengths are CSS px (dp) unless they
// say px.

const (
	termFontSize = 13 // the code font's default size
	termPad      = 4  // CONTENT_PADDING
	termThumbMin = 18 // MIN_SCROLLBAR_THUMB_HEIGHT
	termBarW     = 6  // --app-scrollbar-width
	termSlop     = 4  // TERMINAL_LINK_DRAG_THRESHOLD_PX
	termCompared = 3000
	termBlink    = 500 * time.Millisecond
	termSyncMax  = 500 * time.Millisecond // SYNCHRONIZED_OUTPUT_RENDER_TIMEOUT_MS
	termClicks   = 500 * time.Millisecond // SELECTION_MULTI_CLICK_INTERVAL_MS
	termTick     = 80 * time.Millisecond  // the selection's autoscroll
)

// termFamily is T3's terminal stack (surface.ts:27–35): SF Mono on a Mac,
// Consolas on Windows, Liberation Mono on Linux, then the Nerd Font
// symbols T3 bundles (fonts/SymbolsNerdFontMono-Regular.ttf, converted
// from its woff2) for prompts' icons; JetBrains Mono, tern's own, stands
// in for CSS's monospace.
const termFamily = "SF Mono, Menlo, Consolas, Liberation Mono, Symbols Nerd Font Mono, JetBrains Mono"

func termFont(bold, italic bool) font.Font {
	f := font.Font{Typeface: termFamily, Weight: font.Normal}
	if bold {
		f.Weight = font.Bold
	}
	if italic {
		f.Style = font.Italic
	}
	return f
}

// termPalettes is the drawer's 16 ANSI colours, light then dark
// (ThreadTerminalDrawer.tsx:243–280). They follow the scheme, not the
// theme; 16–255 are xterm's.
var termPalettes = [2][16]color.NRGBA{
	{rgb(0x000000), rgb(0xcd3131), rgb(0x107c10), rgb(0x949800), rgb(0x0451a5), rgb(0xbc05bc), rgb(0x0598bc), rgb(0x555555),
		rgb(0x666666), rgb(0xcd3131), rgb(0x14ce14), rgb(0xb5ba00), rgb(0x0451a5), rgb(0xbc05bc), rgb(0x0598bc), rgb(0xa5a5a5)},
	{rgb(0x000000), rgb(0xcd3131), rgb(0x0dbc79), rgb(0xe5e510), rgb(0x2472c8), rgb(0xbc3fbc), rgb(0x11a8cd), rgb(0xe5e5e5),
		rgb(0x666666), rgb(0xf14c4c), rgb(0x23d18b), rgb(0xf5f543), rgb(0x3b8eea), rgb(0xd670d6), rgb(0x29b8db), rgb(0xe5e5e5)},
}

func termPalette() *[16]color.NRGBA {
	if dark {
		return &termPalettes[1]
	}
	return &termPalettes[0]
}

// termColor is a cell's colour, def when it has none.
func termColor(c color.Color, def color.NRGBA) color.NRGBA {
	switch c := c.(type) {
	case nil:
		return def
	case ansi.BasicColor:
		return termPalette()[c&15]
	case ansi.IndexedColor:
		if c < 16 {
			return termPalette()[c]
		}
	}
	r, g, b, _ := c.RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff}
}

// termFaint is SGR 2's colour: the foreground blended 155:100 over the
// background, rounding down (core.ts:168–175).
func termFaint(fg, bg color.NRGBA) color.NRGBA {
	ch := func(f, b uint8) uint8 { return uint8((int(f)*155 + int(b)*100) / 255) }
	return color.NRGBA{ch(fg.R, bg.R), ch(fg.G, bg.G), ch(fg.B, bg.B), 0xff}
}

// termCells is the terminal font's cell (renderer.ts:158–175): the advance
// of "M", kept fractional; the row, at least 1.35 lines and the ink of
// "Mg"; and the baseline centring that ink in the row. w is px.
type termCells struct {
	scale   float32 // the px per sp they were measured at
	w       float32 // px
	h, base int     // dp
}

// termMetrics measures the cell once per text scale.
func (th *Theme) termMetrics(gtx C) termCells {
	tc := &th.in.terms.cells
	k := gtx.Metric.PxPerSp
	if tc.scale == k && tc.w > 0 {
		return *tc
	}
	params := text.Parameters{Font: termFont(false, false), PxPerEm: fixed.Int26_6(termFontSize*k*64 + 0.5), MaxLines: 1, MaxWidth: 1 << 20}
	f := func(v fixed.Int26_6) float32 { return float32(v) / 64 }
	th.Shaper.LayoutString(params, "M")
	w := float32(0)
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		w += f(g.Advance)
	}
	th.Shaper.LayoutString(params, "Mg")
	var asc, desc float32
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		asc, desc = max(asc, -f(g.Bounds.Min.Y)), max(desc, f(g.Bounds.Max.Y))
	}
	asc, desc = asc/k, desc/k
	if asc == 0 {
		asc = termFontSize
	}
	ink := asc + desc
	h := max(1, int(math.Round(termFontSize*1.35)), int(math.Ceil(float64(ink))))
	base := int(math.Round(float64((float32(h)-ink)/2 + asc)))
	*tc = termCells{scale: k, w: max(1, w), h: h, base: base}
	return *tc
}

// termGrid is how many cells fit a mount w by h dp, inside the 4dp
// padding, cw (dp) by ch apart.
func termGrid(w, h, cw float32, ch int) (cols, rows int) {
	return max(1, int((w-2*termPad)/cw)), max(1, int((h-2*termPad)/float32(ch)))
}

// termOriginY is the grid's top: 4dp in, or once there's scrollback,
// anchored to the bottom so the last row sits on the padding
// (surface.ts:133–143).
func termOriginY(mountH float32, rows, ch int, anchor bool) float32 {
	if !anchor {
		return termPad
	}
	return termPad + max(0, mountH-2*termPad-float32(rows*ch))
}

// termThumb is the scrollbar's thumb in a track px tall, for total rows
// with len on screen and offset scrolled down from the top: its height and
// top, and how far it can scroll; ok is false when there's nothing to
// scroll (surface.ts:151–170).
func termThumb(total, length, offset int, track float32) (h, top float32, most int, ok bool) {
	total = max(0, total)
	length = max(0, min(length, total))
	most = max(0, total-length)
	if track <= 0 || length <= 0 || most == 0 {
		return 0, 0, 0, false
	}
	h = min(track, max(termThumbMin, track*float32(length)/float32(total)))
	offset = max(0, min(offset, most))
	return h, (track - h) * float32(offset) / float32(most), most, true
}

// termWheelRows turns a wheel's pixels into whole rows of ch px, carrying
// what's left over (terminalWheelDeltaRows).
func termWheelRows(px, ch, carry float32) (rows int, left float32) {
	total := carry + px/ch
	rows = int(total)
	return rows, total - float32(rows)
}

// termAutoscroll is how many rows a selection dragged to y scrolls per
// tick: 1 at the edge, ramping to 15 50dp past it (xterm.js's ramp).
func termAutoscroll(y, top, bottom float32) int {
	d := float32(0)
	switch {
	case y < top:
		d = y - top
	case y > bottom:
		d = y - bottom
	}
	if d == 0 {
		return 0
	}
	ramp := max(-1, min(1, d/50))
	sign := 1
	if ramp < 0 {
		sign = -1
	}
	return sign + int(math.Round(float64(ramp*14)))
}

// termPos is a cell, its row counted from the top of the scrollback, so
// output scrolling past doesn't move what it points at.
type termPos struct{ row, col int }

func (a termPos) before(b termPos) bool { return a.row < b.row || a.row == b.row && a.col < b.col }

// termSel is a selection between two cells, both in it: by cell, word or
// line, from the base a double or triple click took.
type termSel struct {
	on           bool
	from, to     termPos
	mode         int // 1 cells, 2 words, 3 lines
	base0, base1 termPos
}

// span is the selection's ends in reading order.
func (s termSel) span() (termPos, termPos) {
	if s.to.before(s.from) {
		return s.to, s.from
	}
	return s.from, s.to
}

func (s termSel) has(row, col int) bool {
	if !s.on {
		return false
	}
	a, b := s.span()
	p := termPos{row, col}
	return !p.before(a) && !b.before(p)
}

// termLink is a link under the pointer: its target and the cells it spans.
type termLink struct {
	url      string
	from, to termPos
}

func (l *termLink) has(row, col int) bool {
	p := termPos{row, col}
	return l != nil && !p.before(l.from) && !l.to.before(p)
}

func (l *termLink) same(o *termLink) bool {
	return l != nil && o != nil && *l == *o
}

// termPane is a pane's own state, kept from frame to frame by its thread
// and terminal.
type termPane struct {
	tag, bar int     // the surface's and the scrollbar's pointer events
	back     int     // rows scrolled back from the bottom
	sb       int     // the scrollback's length when back was counted
	wheel    float32 // wheel travel short of a row
	sel      termSel
	selText  string // what the selection held, to see output change it
	// The click sequence: within 500ms and 4dp of the last, a click
	// counts up to a triple.
	clicks    int
	clickAt   time.Time
	clickPt   f32.Point
	selecting bool // a left drag is selecting
	moved     bool // and has left its first cell
	scroll    int  // rows per autoscroll tick
	scrollAt  time.Time
	pointer   f32.Point
	link      *termLink // under the pointer
	press     *termLink // pressed: it opens on release
	pressAt   f32.Point
	reporting bool // a press went to the app's mouse tracking
	button    uv.MouseButton
	motion    termPos // the cell motion was last reported in
	barHeld   bool
	barGrab   float32 // where the thumb was taken, px down it
	barHover  gesture.Hover
	on        bool      // the cursor's blink phase
	blinkAt   time.Time // when it turns next
	ver       uint64
	focused   bool
	paste     bool // read the clipboard into the shell at the next layout
	// The last paint, kept through a synchronized update.
	paint    op.Ops
	painted  op.CallOp
	pausedAt time.Time
	g        termGeom
}

// termGeom is a pane's last layout, in px: what the pointer lands on.
type termGeom struct {
	cw, ch     float32 // the cell
	x0, y0     float32 // the grid's top left
	cols, rows int
	top        int  // the first row shown, from the top of the scrollback
	sb         int  // the scrollback's length
	alt        bool // the alternate screen
	size       image.Point
}

// cellAt is the cell at p, clamped to the grid.
func (g termGeom) cellAt(p f32.Point) termPos {
	x := int(math.Floor(float64((p.X - g.x0) / g.cw)))
	y := int(math.Floor(float64((p.Y - g.y0) / g.ch)))
	return termPos{g.top + max(0, min(g.rows-1, y)), max(0, min(g.cols-1, x))}
}

// inGrid reports whether p is on a cell rather than the padding.
func (g termGeom) inGrid(p f32.Point) bool {
	return p.X >= g.x0 && p.Y >= g.y0 && p.X < g.x0+float32(g.cols)*g.cw && p.Y < g.y0+float32(g.rows)*g.ch
}

// paneKey keeps panes of different threads apart.
func (th *Theme) paneKey(v *TermView, id string) string { return v.Thread + "\x00" + id }

func (th *Theme) pane(v *TermView, id string) *termPane {
	tu := &th.in.terms
	if tu.panes == nil {
		tu.panes = map[string]*termPane{}
	}
	k := th.paneKey(v, id)
	p := tu.panes[k]
	if p == nil {
		p = &termPane{on: true}
		tu.panes[k] = p
	}
	return p
}

// termCell is a cell as it paints: its text ("" for the second half of a
// wide glyph) and its colours after inverse and faint.
type termCell struct {
	text                                string
	wide                                bool
	fg, bg                              color.NRGBA
	bold, italic, hidden, under, strike bool
}

// cellAt reads the cell at row (from the top of the scrollback), col.
func termCellAt(scr TermScreen, sb, row, col int) *uv.Cell {
	if row < sb {
		return scr.ScrollbackCellAt(col, row)
	}
	return scr.CellAt(col, row-sb)
}

func resolveCell(c *uv.Cell, fg0, bg0 color.NRGBA) termCell {
	if c == nil {
		return termCell{text: " ", fg: fg0, bg: bg0}
	}
	st := c.Style
	t := termCell{text: c.Content, wide: c.Width > 1,
		fg: termColor(st.Fg, fg0), bg: termColor(st.Bg, bg0),
		bold: st.Attrs&uv.AttrBold != 0, italic: st.Attrs&uv.AttrItalic != 0, hidden: st.Attrs&uv.AttrConceal != 0,
		under: st.Underline != uv.UnderlineNone, strike: st.Attrs&uv.AttrStrikethrough != 0}
	if c.Width == 0 {
		t.text = ""
	} else if t.text == "" {
		t.text = " "
	}
	if st.Attrs&uv.AttrReverse != 0 {
		t.fg, t.bg = t.bg, t.fg
	}
	if st.Attrs&uv.AttrFaint != 0 {
		t.fg = termFaint(t.fg, t.bg)
	}
	return t
}

// Powerline's solid and rounded separators, drawn as shapes from the
// cell's edges rather than glyphs (renderer.ts:21–83).
const (
	plRight      = ""
	plLeft       = ""
	plRoundRight = ""
	plRoundLeft  = ""
)

func isPowerline(s string) bool {
	return s == plRight || s == plLeft || s == plRoundRight || s == plRoundLeft
}

// powerline fills separator s over the cell r, overlapping the segment
// it closes by seam px so no hairline shows.
func powerline(gtx C, s string, r image.Rectangle, col color.NRGBA, seam float32) {
	x0, y0, x1, y1 := float32(r.Min.X), float32(r.Min.Y), float32(r.Max.X), float32(r.Max.Y)
	mid := (y0 + y1) / 2
	var p clip.Path
	p.Begin(gtx.Ops)
	switch s {
	case plRight, plLeft:
		base, point := x0, x1
		if s == plLeft {
			base, point = x1, x0
		}
		p.MoveTo(f32.Pt(base, y0))
		p.LineTo(f32.Pt(point, mid))
		p.LineTo(f32.Pt(base, y1))
		p.Close()
	default:
		// A half ellipse from the base edge, w wide and the row tall.
		base, dir := x0, float32(1)
		if s == plRoundLeft {
			base, dir = x1, -1
		}
		w, hh := x1-x0, (y1-y0)/2
		const k = 0.5522847
		p.MoveTo(f32.Pt(base, y0))
		p.CubeTo(f32.Pt(base+dir*w*k, y0), f32.Pt(base+dir*w, mid-hh*k), f32.Pt(base+dir*w, mid))
		p.CubeTo(f32.Pt(base+dir*w, mid+hh*k), f32.Pt(base+dir*w*k, y1), f32.Pt(base, y1))
		p.Close()
		if seam > 0 {
			sx := base - dir*seam
			p.MoveTo(f32.Pt(min(sx, base), y0))
			p.LineTo(f32.Pt(max(sx, base), y0))
			p.LineTo(f32.Pt(max(sx, base), y1))
			p.LineTo(f32.Pt(min(sx, base), y1))
			p.Close()
		}
	}
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

// termSurface is a pane's terminal: its mount on the terminal background,
// the grid painted from the emulator, and the scrollbar; it takes the
// pointer, and the keyboard while it's the active pane. at is where it
// sits in the drawer.
func (th *Theme) termSurface(gtx C, v *TermView, tp *TermPane, active bool, at image.Point) {
	size := gtx.Constraints.Max
	p := th.pane(v, tp.ID)
	fill(gtx, P.TerminalBackground, image.Rectangle{Max: size}, 0)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	k := gtx.Metric.PxPerDp
	cells := th.termMetrics(gtx)
	scr := tp.Screen
	if scr != nil {
		mw, mh := float32(size.X)/k, float32(size.Y)/k
		if cols, rows := termGrid(mw, mh, cells.w/k, cells.h); tp.Fit != nil {
			tp.Fit(cols, rows)
		}
		th.termPointer(gtx, v, tp, p, active, at)
		if active {
			th.termKeys(gtx, v, tp, p)
		}
		p.g = th.termLayout(gtx, scr, p, cells, size)
		th.termFollow(scr, tp, p)
		th.termBlink(gtx, scr, tp, p)
		th.termPaint(gtx, scr, tp, p, cells)
	}
	// The surface takes the pointer: a text cursor, a hand on a link, an
	// arrow while the app tracks the mouse. The scrollbar goes over it.
	func() {
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		cur := pointer.CursorText
		switch {
		case p.link != nil:
			cur = pointer.CursorPointer
		case tp.Mouse != TermMouseOff:
			cur = pointer.CursorDefault
		}
		cur.Add(gtx.Ops)
		event.Op(gtx.Ops, &p.tag)
		if active {
			event.Op(gtx.Ops, &th.in.term)
		}
	}()
	if scr != nil {
		th.termScrollbar(gtx, p, size)
	}
	if active && th.in.terms.focus {
		th.in.terms.focus = false
		gtx.Execute(key.FocusCmd{Tag: &th.in.term})
	}
	if active && p.paste {
		p.paste = false
		gtx.Execute(clipboard.ReadCmd{Tag: &th.in.term})
	}
}

// termLayout is where the grid sits this frame: its cells, the rows on
// screen, and the origin.
func (th *Theme) termLayout(gtx C, scr TermScreen, p *termPane, cells termCells, size image.Point) termGeom {
	k := gtx.Metric.PxPerDp
	g := termGeom{cw: cells.w, ch: float32(cells.h) * k, x0: termPad * k, cols: scr.Width(), rows: scr.Height(), alt: scr.IsAltScreen(), size: size}
	if !g.alt {
		g.sb = scr.ScrollbackLen()
	}
	// Scrolled back, the view stays on what it showed as output pushes
	// rows into the scrollback.
	if p.back > 0 {
		p.back += g.sb - p.sb
	}
	p.back = max(0, min(p.back, g.sb))
	p.sb = g.sb
	g.top = g.sb - p.back
	g.y0 = termOriginY(float32(size.Y)/k, g.rows, cells.h, g.sb > 0) * k
	return g
}

// termFollow clears a selection that output changed under it, as T3 does
// after each write; one over 3,000 cells is cleared without comparing.
func (th *Theme) termFollow(scr TermScreen, tp *TermPane, p *termPane) {
	if p.ver == tp.Ver {
		return
	}
	p.ver = tp.Ver
	if !p.sel.on {
		return
	}
	a, b := p.sel.span()
	if (b.row-a.row+1)*p.g.cols > termCompared || termSelText(scr, p.g, p.sel) != p.selText {
		p.sel = termSel{}
	}
}

// termBlink runs the cursor's 500ms blink while the terminal has the
// keyboard, the cursor shows and the app wants it blinking. Like T3's, the
// next turn waits 500ms from the last repaint.
func (th *Theme) termBlink(gtx C, scr TermScreen, tp *TermPane, p *termPane) {
	focused := gtx.Focused(&th.in.term)
	_, blink := scr.CursorStyle()
	enabled := focused && blink && !scr.CursorHidden()
	if focused != p.focused {
		p.focused, p.on, p.blinkAt = focused, true, gtx.Now.Add(termBlink)
	}
	if !enabled {
		p.on = true
		return
	}
	if p.blinkAt.IsZero() {
		p.blinkAt = gtx.Now.Add(termBlink)
	}
	if !gtx.Now.Before(p.blinkAt) {
		p.on, p.blinkAt = !p.on, gtx.Now.Add(termBlink)
	}
	gtx.Execute(op.InvalidateCmd{At: p.blinkAt})
}

// termPaint paints the grid: row by row the default background, the
// cells' own backgrounds and the selection over them (never the padding),
// the text in runs of one style, the decorations, then the cursor. During
// a synchronized update it shows the last paint, for at most 500ms.
func (th *Theme) termPaint(gtx C, scr TermScreen, tp *TermPane, p *termPane, cells termCells) {
	if tp.Paused {
		if p.pausedAt.IsZero() {
			p.pausedAt = gtx.Now
		}
		if gtx.Now.Sub(p.pausedAt) < termSyncMax && p.painted != (op.CallOp{}) {
			gtx.Execute(op.InvalidateCmd{At: p.pausedAt.Add(termSyncMax)})
			p.painted.Add(gtx.Ops)
			return
		}
	} else {
		p.pausedAt = time.Time{}
	}
	p.paint.Reset()
	g := gtx
	g.Ops = &p.paint
	m := op.Record(g.Ops)
	th.termPaintGrid(g, scr, p, cells)
	p.painted = m.Stop()
	p.painted.Add(gtx.Ops)
}

func (th *Theme) termPaintGrid(gtx C, scr TermScreen, p *termPane, cells termCells) {
	geo := p.g
	k := gtx.Metric.PxPerDp
	fg0, bg0 := P.TerminalForeground, P.TerminalBackground
	sel := P.TerminalSelectionBackground
	origin := geo.y0 / k
	rowTop := func(r int) int { return int(math.Round(float64((origin + float32(r*cells.h)) * k))) }
	colX := func(c int) int { return int(math.Round(float64(geo.x0 + float32(c)*geo.cw))) }
	line := make([]termCell, geo.cols)
	for y := 0; y < geo.rows; y++ {
		row := geo.top + y
		top, bot := rowTop(y), rowTop(y+1)
		fill(gtx, bg0, image.Rect(0, top, geo.size.X, bot), 0)
		for x := range line {
			line[x] = resolveCell(termCellAt(scr, geo.sb, row, x), fg0, bg0)
		}
		// Backgrounds, in runs of one colour and selection.
		for x0 := 0; x0 < len(line); {
			x1 := x0 + 1
			on := p.sel.has(row, x0)
			for x1 < len(line) && line[x1].bg == line[x0].bg && p.sel.has(row, x1) == on {
				x1++
			}
			bg := line[x0].bg
			if on {
				bg = over(sel, bg)
			}
			if bg != bg0 {
				fill(gtx, bg, image.Rect(colX(x0), top, colX(x1), bot), 0)
			}
			x0 = x1
		}
		// Text, in runs of one colour, weight, slant and visibility.
		for x0 := 0; x0 < len(line); {
			c := line[x0]
			if c.text == "" {
				x0++
				continue
			}
			if isPowerline(c.text) {
				if !c.hidden {
					powerline(gtx, c.text, image.Rect(colX(x0), top, colX(x0+1), bot), c.fg, float32(max(1, dp(gtx, 1))))
				}
				x0++
				continue
			}
			x1 := x0 + 1
			for x1 < len(line) {
				n := line[x1]
				if n.text == "" && line[x1-1].wide {
					x1++
					continue
				}
				if n.text == "" || isPowerline(n.text) || n.fg != c.fg || n.bold != c.bold || n.italic != c.italic || n.hidden != c.hidden {
					break
				}
				x1++
			}
			if !c.hidden {
				th.termRun(gtx, line, x0, x1, colX, top, bot, cells)
			}
			x0 = x1
		}
		// Underlines (every style, and the hovered link), strikethrough.
		px := max(1, dp(gtx, 1))
		for x, c := range line {
			link := p.link.has(row, x)
			if !c.under && !c.strike && !link {
				continue
			}
			l, r := colX(x), colX(x+1)
			if c.under || link {
				uy := top + dp(gtx, float32(cells.h-2))
				fill(gtx, c.fg, image.Rect(l, uy, r, uy+px), 0)
			}
			if c.strike {
				sy := top + dp(gtx, float32(math.Floor(float64(cells.h)*0.55)))
				fill(gtx, c.fg, image.Rect(l, sy, r, sy+px), 0)
			}
		}
	}
	th.termCursor(gtx, scr, p, cells, rowTop, colX)
}

// termRun draws cells x0 to x1 of a row as text: runs of narrow cells as
// one string from their first column, squeezed to the cells when the font
// sets them wider, and each wide glyph centred in its two cells.
func (th *Theme) termRun(gtx C, line []termCell, x0, x1 int, colX func(int) int, top, bot int, cells termCells) {
	c := line[x0]
	var b strings.Builder
	for _, cl := range line[x0:x1] {
		b.WriteString(cl.text)
	}
	if strings.TrimSpace(b.String()) == "" {
		return
	}
	defer clip.Rect{Min: image.Pt(colX(x0), top), Max: image.Pt(colX(x1), bot)}.Push(gtx.Ops).Pop()
	f := termFont(c.bold, c.italic)
	for seg := x0; seg < x1; {
		end := seg
		for end < x1 && !line[end].wide {
			end++
		}
		if end > seg {
			b.Reset()
			for _, cl := range line[seg:end] {
				b.WriteString(cl.text)
			}
			th.termText(gtx, f, c.fg, b.String(), colX(seg), colX(end)-colX(seg), top, cells, false)
		}
		if end == x1 {
			break
		}
		th.termText(gtx, f, c.fg, line[end].text, colX(end), colX(end+2)-colX(end), top, cells, true)
		seg = end + 2
	}
}

// termText draws s with its baseline on the cell's, from x in w px:
// centred when it's a wide glyph, else squeezed into w if it sets wider.
func (th *Theme) termText(gtx C, f font.Font, col color.NRGBA, s string, x, w, top int, cells termCells, centre bool) {
	m := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	mat := m.Stop()
	m = op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	d := widget.Label{MaxLines: 1}.Layout(g, th.Shaper, f, unit.Sp(termFontSize), s, mat)
	call := m.Stop()
	y := top + dp(gtx, float32(cells.base)) - (d.Size.Y - d.Baseline)
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	switch {
	case centre:
		defer op.Offset(image.Pt((w-d.Size.X)/2, 0)).Push(gtx.Ops).Pop()
	case d.Size.X > w && d.Size.X > 0:
		defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(float32(w)/float32(d.Size.X), 1))).Push(gtx.Ops).Pop()
	}
	call.Add(gtx.Ops)
}

// termCursor draws the cursor where it's on screen and in its blink's on
// phase: without the keyboard a 1dp outline inside the cell; else a 2dp
// bar, a 2dp underline, or the block, with the glyph under it redrawn in
// the background colour. A cursor on a wide glyph covers both cells.
func (th *Theme) termCursor(gtx C, scr TermScreen, p *termPane, cells termCells, rowTop, colX func(int) int) {
	geo := p.g
	if !p.on || scr.CursorHidden() {
		return
	}
	pos := scr.CursorPosition()
	y := geo.sb + pos.Y - geo.top
	if y < 0 || y >= geo.rows || pos.X < 0 || pos.X >= geo.cols {
		return
	}
	c := resolveCell(scr.CellAt(pos.X, pos.Y), P.TerminalForeground, P.TerminalBackground)
	n := 1
	if c.wide {
		n = 2
	}
	r := image.Rect(colX(pos.X), rowTop(y), colX(pos.X+n), rowTop(y+1))
	col := P.TerminalCursor
	style, _ := scr.CursorStyle()
	switch {
	case !p.focused:
		strokeRect(gtx, col, r, 0, float32(max(1, dp(gtx, 1))))
	case style == vt.CursorBar:
		fill(gtx, col, image.Rect(r.Min.X, r.Min.Y, r.Min.X+dp(gtx, 2), r.Max.Y), 0)
	case style == vt.CursorUnderline:
		fill(gtx, col, image.Rect(r.Min.X, r.Max.Y-dp(gtx, 2), r.Max.X, r.Max.Y), 0)
	default:
		fill(gtx, col, r, 0)
		switch {
		case c.text == "" || c.text == " " || c.hidden:
		case isPowerline(c.text):
			powerline(gtx, c.text, image.Rect(r.Min.X, r.Min.Y, colX(pos.X+1), r.Max.Y), P.TerminalBackground, 0)
		default:
			th.termText(gtx, termFont(c.bold, c.italic), P.TerminalBackground, c.text, r.Min.X, r.Dx(), r.Min.Y, cells, c.wide)
		}
	}
}

// termScrollbar is the scrollback's bar while there's history: a 6dp
// track 1dp from the right and 4dp from the top and bottom, its 4dp thumb
// radius 3 in the scrollbar colour, lighter or darker on hover. A press on
// the thumb drags it from where it was taken; on the track, from its
// middle.
func (th *Theme) termScrollbar(gtx C, p *termPane, size image.Point) {
	geo := p.g
	k := gtx.Metric.PxPerDp
	track := float32(size.Y) - 2*termPad*k
	h, top, most, ok := termThumb(geo.sb+geo.rows, geo.rows, geo.top, track)
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &p.bar, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			if e.Buttons != pointer.ButtonPrimary {
				continue
			}
			p.barHeld = true
			p.barGrab = h / 2
			if e.Position.Y >= top && e.Position.Y < top+h {
				p.barGrab = e.Position.Y - top
			}
			p.barTo(e.Position.Y, h, track, most)
		case pointer.Drag:
			if p.barHeld {
				p.barTo(e.Position.Y, h, track, most)
			}
		case pointer.Release, pointer.Cancel:
			p.barHeld = false
		}
	}
	hovered := p.barHover.Update(gtx.Source)
	if !ok {
		return
	}
	x := size.X - dp(gtx, 1+termBarW)
	defer op.Offset(image.Pt(x, int(termPad*k))).Push(gtx.Ops).Pop()
	thumb := P.AppScrollbarThumb
	if hovered || p.barHeld {
		thumb = P.AppScrollbarThumbHover
	}
	px := max(1, dp(gtx, 1))
	r := image.Rect(px, int(math.Round(float64(top))), dp(gtx, termBarW)-px, int(math.Round(float64(top+h))))
	fill(gtx, over(thumb, P.TerminalBackground), r, min(dp(gtx, 3), r.Dx()/2))
	defer clip.Rect{Max: image.Pt(dp(gtx, termBarW), int(track))}.Push(gtx.Ops).Pop()
	pointer.CursorDefault.Add(gtx.Ops)
	p.barHover.Add(gtx.Ops)
	event.Op(gtx.Ops, &p.bar)
}

// barTo scrolls so the thumb's grabbed point sits at y in the track.
func (p *termPane) barTo(y, h, track float32, most int) {
	travel := track - h
	if travel <= 0 {
		return
	}
	at := max(0, min(y-p.barGrab, travel))
	p.back = most - int(math.Round(float64(at/travel*float32(most))))
}

// scrollRows moves the view n rows, down toward the latest when n > 0.
func (p *termPane) scrollRows(n int) {
	p.back = max(0, min(p.back-n, p.g.sb))
}

// selectAll selects everything, history and screen.
func (p *termPane) selectAll(scr TermScreen) {
	if scr == nil {
		return
	}
	last := p.g.sb + p.g.rows - 1
	p.sel = termSel{on: true, from: termPos{0, 0}, to: termPos{last, p.g.cols - 1}, mode: 1}
	p.selText = termSelText(scr, p.g, p.sel)
}

// termBlank reports whether a cell holds nothing to read.
func termBlank(c *uv.Cell) bool { return c == nil || c.Content == "" || c.Content == " " }

// wraps reports whether row runs on into the next as one line, which the
// emulator doesn't record: its last column is written.
func termWraps(scr TermScreen, geo termGeom, row int) bool {
	return !termBlank(termCellAt(scr, geo.sb, row, geo.cols-1))
}

// termRowText is a row's text from col x0 to x1, the second halves of
// wide glyphs left out, and with each cell's byte offset in it.
func termRowText(scr TermScreen, geo termGeom, row, x0, x1 int) (string, []int) {
	var b strings.Builder
	at := make([]int, 0, x1-x0+1)
	for x := x0; x <= x1; x++ {
		at = append(at, b.Len())
		c := termCellAt(scr, geo.sb, row, x)
		switch {
		case c == nil:
			b.WriteByte(' ')
		case c.Width == 0:
		case c.Content == "":
			b.WriteByte(' ')
		default:
			b.WriteString(c.Content)
		}
	}
	at = append(at, b.Len())
	return b.String(), at
}

// termSelText is the selection as text, the way libghostty formats it
// plain: a line per row, trailing blanks trimmed, soft wraps joined.
func termSelText(scr TermScreen, geo termGeom, s termSel) string {
	if !s.on {
		return ""
	}
	a, b := s.span()
	var out strings.Builder
	for row := a.row; row <= b.row; row++ {
		x0, x1 := 0, geo.cols-1
		if row == a.row {
			x0 = a.col
		}
		if row == b.row {
			x1 = min(x1, b.col)
		}
		t, _ := termRowText(scr, geo, row, x0, x1)
		if row < b.row && termWraps(scr, geo, row) {
			out.WriteString(t)
			continue
		}
		out.WriteString(strings.TrimRight(t, " "))
		if row < b.row {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// termWordChars are Ghostty's default word boundaries
// (selection-word-chars).
const termWordChars = " \t'\"│`|:;,()[]{}<>$"

// termWord is the run of cells around p a double click selects: the word,
// or the run of boundary characters, it's on.
func termWord(scr TermScreen, geo termGeom, p termPos) (termPos, termPos) {
	boundary := func(x int) bool {
		c := termCellAt(scr, geo.sb, p.row, x)
		return termBlank(c) || strings.Contains(termWordChars, c.Content)
	}
	want := boundary(p.col)
	a, b := p.col, p.col
	for a > 0 && boundary(a-1) == want {
		a--
	}
	for b < geo.cols-1 && boundary(b+1) == want {
		b++
	}
	return termPos{p.row, a}, termPos{p.row, b}
}

// termLine is the logical line around p a triple click selects: its rows
// soft-wrapped together, end to end.
func termLine(scr TermScreen, geo termGeom, p termPos) (termPos, termPos) {
	a, b := p.row, p.row
	for a > 0 && termWraps(scr, geo, a-1) {
		a--
	}
	for b < geo.sb+geo.rows-1 && termWraps(scr, geo, b) {
		b++
	}
	return termPos{a, 0}, termPos{b, geo.cols - 1}
}

// termLinkAt is the link at cell p: an OSC 8 hyperlink, across the cells
// that carry it, else a URL found in p's logical line. A line that runs
// off the top or bottom of the screen gives none, since its text there is
// cut short.
func termLinkAt(scr TermScreen, geo termGeom, p termPos) *termLink {
	if c := termCellAt(scr, geo.sb, p.row, p.col); c != nil && c.Link.URL != "" {
		url := c.Link.URL
		same := func(q termPos) bool {
			c := termCellAt(scr, geo.sb, q.row, q.col)
			return c != nil && c.Link.URL == url
		}
		from, to := p, p
		for {
			q := termPos{from.row, from.col - 1}
			if from.col == 0 {
				if from.row <= geo.top || !termWraps(scr, geo, from.row-1) {
					break
				}
				q = termPos{from.row - 1, geo.cols - 1}
			}
			if !same(q) {
				break
			}
			from = q
		}
		for {
			q := termPos{to.row, to.col + 1}
			if to.col == geo.cols-1 {
				if to.row+1 >= geo.top+geo.rows || !termWraps(scr, geo, to.row) {
					break
				}
				q = termPos{to.row + 1, 0}
			}
			if !same(q) {
				break
			}
			to = q
		}
		return &termLink{url: url, from: from, to: to}
	}
	first, last := p.row, p.row
	for first > geo.top && termWraps(scr, geo, first-1) {
		first--
	}
	if first == geo.top && first > 0 && termWraps(scr, geo, first-1) {
		return nil
	}
	for last < geo.top+geo.rows-1 && termWraps(scr, geo, last) {
		last++
	}
	var text strings.Builder
	type seg struct {
		row, start int
		at         []int
	}
	var segs []seg
	for row := first; row <= last; row++ {
		t, at := termRowText(scr, geo, row, 0, geo.cols-1)
		if row == last {
			t = strings.TrimRight(t, " ")
		}
		segs = append(segs, seg{row, text.Len(), at})
		text.WriteString(t)
	}
	line := text.String()
	where := func(off int) termPos {
		for i := len(segs) - 1; i >= 0; i-- {
			if s := segs[i]; off >= s.start {
				col := 0
				for col+1 < len(s.at)-1 && s.start+s.at[col+1] <= off {
					col++
				}
				return termPos{s.row, col}
			}
		}
		return termPos{first, 0}
	}
	// A press on a wide glyph's second half is on the glyph.
	col := p.col
	if c := termCellAt(scr, geo.sb, p.row, col); col > 0 && c != nil && c.Width == 0 {
		col--
	}
	hit := 0
	for _, s := range segs {
		if s.row == p.row {
			hit = s.start + s.at[min(col, len(s.at)-1)]
		}
	}
	for _, m := range termURLs(line) {
		if hit < m.start || hit >= m.end {
			continue
		}
		if m.end == len(line) && termWraps(scr, geo, last) {
			return nil
		}
		to := where(m.end - 1)
		if c := termCellAt(scr, geo.sb, to.row, to.col+1); to.col+1 < geo.cols && c != nil && c.Width == 0 {
			to.col++
		}
		return &termLink{url: m.text, from: where(m.start), to: to}
	}
	return nil
}

// termPointer takes the pane's pointer: presses focus the terminal and
// make the pane active; with the app tracking the mouse (unless Shift,
// Ctrl or ⌘ is held) presses, motion and the wheel go to it; otherwise
// the left button selects by cell, word or line and opens links, the
// middle one pastes the selection on Linux, the right one opens the
// menu, and the wheel scrolls back through history.
func (th *Theme) termPointer(gtx C, v *TermView, tp *TermPane, p *termPane, active bool, at image.Point) {
	geo := p.g
	report := func(e pointer.Event) bool {
		return tp.Mouse != TermMouseOff && !e.Modifiers.Contain(key.ModShift) && !e.Modifiers.Contain(key.ModCtrl) && !e.Modifiers.Contain(key.ModCommand)
	}
	send := func(m uv.MouseEvent) {
		if tp.Report != nil {
			tp.Report(m)
		}
	}
	mouse := func(e pointer.Event, b uv.MouseButton) uv.Mouse {
		c := geo.cellAt(e.Position)
		var mod uv.KeyMod
		if e.Modifiers.Contain(key.ModShift) {
			mod |= uv.ModShift
		}
		if e.Modifiers.Contain(key.ModAlt) {
			mod |= uv.ModAlt
		}
		if e.Modifiers.Contain(key.ModCtrl) {
			mod |= uv.ModCtrl
		}
		return uv.Mouse{X: c.col, Y: c.row - geo.top, Button: b, Mod: mod}
	}
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &p.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Move | pointer.Leave | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok || geo.cols == 0 {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			gtx.Execute(key.FocusCmd{Tag: &th.in.term})
			if !active {
				th.push(Action{ID: "terminal.activate:" + tp.ID, Kind: "click"})
			}
			th.in.terms.menu = nil
			if report(e) {
				// The app gets every button; the right one opens no menu.
				p.reporting, p.button = true, termButton(e.Buttons)
				send(uv.MouseClickEvent(mouse(e, p.button)))
				continue
			}
			switch e.Buttons {
			case pointer.ButtonTertiary:
				if runtime.GOOS == "linux" && p.sel.on && tp.Paste != nil {
					tp.Paste(termSelText(tp.Screen, geo, p.sel))
				}
				continue
			case pointer.ButtonSecondary:
				th.openTermMenu(th.paneKey(v, tp.ID), at.Add(e.Position.Round()), termMenuItems(p.sel.on), time.Time{})
				continue
			case pointer.ButtonPrimary:
			default:
				continue
			}
			k := gtx.Metric.PxPerDp
			if d := e.Position.Sub(p.clickPt); gtx.Now.Sub(p.clickAt) <= termClicks && d.X*d.X+d.Y*d.Y <= termSlop*termSlop*k*k && p.clicks < 3 {
				p.clicks++
			} else {
				p.clicks = 1
			}
			p.clickAt, p.clickPt = gtx.Now, e.Position
			if l := termLinkAt(tp.Screen, geo, geo.cellAt(e.Position)); l != nil && geo.inGrid(e.Position) && isTermURL(l.url) && !e.Modifiers.Contain(key.ModShift) && p.clicks == 1 {
				p.press, p.pressAt = l, e.Position
				continue
			}
			p.link = nil
			p.beginSelection(tp.Screen, geo, e.Position)
		case pointer.Drag:
			if p.reporting {
				if tp.Mouse >= TermMouseDrag {
					c := geo.cellAt(e.Position)
					if c != p.motion {
						p.motion = c
						send(uv.MouseMotionEvent(mouse(e, p.button)))
					}
				}
				continue
			}
			if p.press != nil {
				k := gtx.Metric.PxPerDp
				if d := e.Position.Sub(p.pressAt); d.X*d.X+d.Y*d.Y <= termSlop*termSlop*k*k {
					continue
				}
				p.press = nil
				p.beginSelection(tp.Screen, geo, p.pressAt)
			}
			if p.selecting {
				p.pointer = e.Position
				k := gtx.Metric.PxPerDp
				p.scroll = termAutoscroll(e.Position.Y/k, termPad, float32(geo.size.Y)/k-termPad)
				p.extendSelection(tp.Screen, geo, e.Position)
			}
		case pointer.Release, pointer.Cancel:
			p.scroll = 0
			if p.press != nil {
				l := p.press
				p.press = nil
				if e.Kind == pointer.Release && l.same(termLinkAt(tp.Screen, geo, geo.cellAt(e.Position))) {
					th.push(Action{ID: "terminal.link:" + l.url, Kind: "click"})
				}
				continue
			}
			if p.reporting {
				p.reporting = false
				send(uv.MouseReleaseEvent(mouse(e, p.button)))
				continue
			}
			if !p.selecting {
				continue
			}
			p.selecting = false
			if !p.moved && p.sel.mode == 1 {
				p.sel = termSel{}
			}
			if p.sel.on && e.Kind == pointer.Release && strings.Trim(termSelText(tp.Screen, geo, p.sel), "\n") != "" {
				// The selection's own menu, at the release inside the
				// terminal, after any further click of a double or triple.
				wait := gtx.Now
				if p.clicks >= 2 {
					wait = wait.Add(termClicks)
				}
				pt := image.Pt(max(0, min(geo.size.X, e.Position.Round().X)), max(0, min(geo.size.Y, e.Position.Round().Y)))
				th.openTermMenu(th.paneKey(v, tp.ID), at.Add(pt), termMenuItems(true)[:1], wait)
			}
		case pointer.Move:
			if tp.Mouse == TermMouseMotion && report(e) {
				c := geo.cellAt(e.Position)
				if c != p.motion {
					p.motion = c
					send(uv.MouseMotionEvent(mouse(e, uv.MouseNone)))
				}
				p.link = nil
				continue
			}
			p.link = nil
			if geo.inGrid(e.Position) {
				if l := termLinkAt(tp.Screen, geo, geo.cellAt(e.Position)); l != nil && isTermURL(l.url) {
					p.link = l
				}
			}
		case pointer.Leave:
			p.link = nil
		case pointer.Scroll:
			rows, left := termWheelRows(e.Scroll.Y, geo.ch, p.wheel)
			p.wheel = left
			if rows == 0 {
				continue
			}
			switch {
			case report(e):
				b := uv.MouseWheelDown
				if rows < 0 {
					b = uv.MouseWheelUp
				}
				for range abs(rows) {
					send(uv.MouseWheelEvent(mouse(e, b)))
				}
			case geo.alt:
				// The alternate screen has no history: the wheel's rows go
				// as arrow keys, as in xterm.
				seq := "\x1b[B"
				if rows < 0 {
					seq = "\x1b[A"
				}
				if tp.AppCursor {
					seq = "\x1bO" + seq[2:]
				}
				if tp.Send != nil {
					tp.Send(strings.Repeat(seq, abs(rows)))
				}
			default:
				p.scrollRows(rows)
			}
		}
	}
	// Dragged past the top or bottom, the selection scrolls every 80ms.
	if p.selecting && p.scroll != 0 {
		if !gtx.Now.Before(p.scrollAt) {
			p.scrollAt = gtx.Now.Add(termTick)
			p.scrollRows(p.scroll)
			p.g.top = p.g.sb - p.back
			p.extendSelection(tp.Screen, p.g, p.pointer)
		}
		gtx.Execute(op.InvalidateCmd{At: p.scrollAt})
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// termButton is the button a press reports.
func termButton(b pointer.Buttons) uv.MouseButton {
	switch b {
	case pointer.ButtonTertiary:
		return uv.MouseMiddle
	case pointer.ButtonSecondary:
		return uv.MouseRight
	}
	return uv.MouseLeft
}

// beginSelection starts a selection at pos: the cell, or for a double or
// triple click the word or line, which a drag then grows by.
func (p *termPane) beginSelection(scr TermScreen, geo termGeom, pos f32.Point) {
	c := geo.cellAt(pos)
	p.selecting, p.moved, p.pointer = true, false, pos
	p.sel = termSel{on: true, from: c, to: c, mode: min(p.clicks, 3)}
	switch p.sel.mode {
	case 2:
		p.sel.base0, p.sel.base1 = termWord(scr, geo, c)
	case 3:
		p.sel.base0, p.sel.base1 = termLine(scr, geo, c)
	default:
		p.sel.mode = 1
		p.selText = termSelText(scr, geo, p.sel)
		return
	}
	p.sel.from, p.sel.to = p.sel.base0, p.sel.base1
	p.selText = termSelText(scr, geo, p.sel)
}

// extendSelection grows the selection to pos: by cells from its anchor,
// or by words or lines from the side of the base pos is on.
func (p *termPane) extendSelection(scr TermScreen, geo termGeom, pos f32.Point) {
	c := geo.cellAt(pos)
	if p.sel.mode == 1 {
		if c == p.sel.to {
			return
		}
		p.moved = true
		p.sel.to = c
	} else {
		a, b := termWord(scr, geo, c)
		if p.sel.mode == 3 {
			a, b = termLine(scr, geo, c)
		}
		p.moved = true
		if c.before(p.sel.base0) {
			p.sel.from, p.sel.to = p.sel.base1, a
		} else {
			p.sel.from, p.sel.to = p.sel.base0, b
		}
	}
	p.selText = termSelText(scr, geo, p.sel)
}

// termKeys takes the keyboard for the active pane. The drawer's own chords
// come first (mod+D splits, mod+Shift+D splits stacked, mod+N adds a
// terminal, mod+W closes one, keybindings.ts); then the terminal's
// (surface.ts:1111–1222, ThreadTerminalDrawer.tsx:1002–1038): select all,
// paging through history, copy and paste, the word and line moves, clear;
// everything else goes through the emulator's key encoder, and typing as
// text.
func (th *Theme) termKeys(gtx C, v *TermView, tp *TermPane, p *termPane) {
	tag := &th.in.term
	mac := runtime.GOOS == "darwin"
	all := key.ModCtrl | key.ModShift | key.ModAlt | key.ModCommand | key.ModSuper
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: tag},
			key.Filter{Focus: tag, Name: "", Optional: all},
			key.Filter{Focus: tag, Name: key.NameTab, Optional: key.ModShift},
			transfer.TargetFilter{Target: tag, Type: "application/text"},
		)
		if !ok {
			return
		}
		switch e := ev.(type) {
		case key.EditEvent:
			if tp.Send != nil && e.Text != "" {
				tp.Send(e.Text)
			}
			// Keep the input method's text empty: all of it went to the
			// shell. Its caret sits on the cursor's cell.
			gtx.Execute(key.SnippetCmd{Tag: tag})
			gtx.Execute(key.SelectionCmd{Tag: tag, Caret: th.termCaret(gtx, tp, p)})
		case transfer.DataEvent:
			b, _ := io.ReadAll(e.Open())
			if tp.Paste != nil && len(b) > 0 {
				tp.Paste(string(b))
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			if th.termChord(v, e) || th.termLocalKey(gtx, tp, p, e, mac) {
				continue
			}
			if k, ok := termKey(e, mac); ok && tp.Key != nil {
				tp.Key(k)
			}
		}
	}
}

// termChord runs the drawer's chords, which never reach the shell.
func (th *Theme) termChord(v *TermView, e key.Event) bool {
	if !e.Modifiers.Contain(key.ModShortcut) || e.Modifiers&^(key.ModShortcut|key.ModShift) != 0 {
		return false
	}
	shift := e.Modifiers.Contain(key.ModShift)
	switch {
	case e.Name == "D":
		id := "terminal.split"
		if shift {
			id = "terminal.splitv"
		}
		th.push(Action{ID: id, Kind: "click"})
	case e.Name == "N" && !shift:
		th.push(Action{ID: "terminal.new", Kind: "click"})
	case e.Name == "W" && !shift:
		th.push(Action{ID: "terminal.close", Kind: "click"})
	default:
		return false
	}
	return true
}

// termLocalKey runs the keys the terminal keeps for itself. It reports
// whether e was one.
func (th *Theme) termLocalKey(gtx C, tp *TermPane, p *termPane, e key.Event, mac bool) bool {
	m := e.Modifiers
	only := func(want key.Modifiers) bool { return m == want }
	send := func(s string) bool {
		if tp.Send != nil {
			tp.Send(s)
		}
		return true
	}
	primary := key.ModCtrl
	if mac {
		primary = key.ModCommand
	}
	left := e.Name == key.NameLeftArrow
	arrow := left || e.Name == key.NameRightArrow
	switch {
	// Word and line moves, delete to line start, clear.
	case arrow && only(key.ModAlt) && left:
		return send("\x1bb")
	case arrow && only(key.ModAlt):
		return send("\x1bf")
	case mac && arrow && only(key.ModCommand) && left:
		return send("\x01")
	case mac && arrow && only(key.ModCommand):
		return send("\x05")
	case mac && e.Name == key.NameDeleteBackward && only(key.ModCommand):
		return send("\x15")
	case e.Name == "L" && only(key.ModCtrl), mac && e.Name == "K" && only(key.ModCommand):
		return send("\x0c")
	// Select all: ⌘A, or Ctrl+Shift+A where Ctrl+A is the shell's.
	case e.Name == "A" && (mac && only(key.ModCommand) || !mac && only(key.ModCtrl|key.ModShift)):
		p.selectAll(tp.Screen)
		return true
	// A page of history, or all of it, on the main screen.
	case (e.Name == key.NamePageUp || e.Name == key.NamePageDown) && only(key.ModShift) && !p.g.alt:
		n := max(1, p.g.rows)
		if e.Name == key.NamePageUp {
			n = -n
		}
		p.scrollRows(n)
		return true
	case e.Name == key.NameHome && only(primary|key.ModShift) && !p.g.alt:
		p.back = p.g.sb
		return true
	case e.Name == key.NameEnd && only(primary|key.ModShift) && !p.g.alt:
		p.back = 0
		return true
	}
	// Copy, with a selection: ⌘C; Ctrl+C or Ctrl+Shift+C. (T3's Insert
	// chords have no key in Gio.) After a plain Ctrl+C the selection goes,
	// so the next one interrupts.
	copyKey := mac && e.Name == "C" && only(key.ModCommand) || !mac && e.Name == "C" && (only(key.ModCtrl) || only(key.ModCtrl|key.ModShift))
	if copyKey && p.sel.on {
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(termSelText(tp.Screen, p.g, p.sel)))})
		if !mac && e.Name == "C" && only(key.ModCtrl) {
			p.sel = termSel{}
		}
		return true
	}
	if mac && e.Name == "C" && only(key.ModCommand) {
		return true // ⌘C with nothing selected does nothing
	}
	// Paste: ⌘V, or Ctrl+Shift+V. Ctrl+V is the shell's.
	if e.Name == "V" && (mac && only(key.ModCommand) || !mac && only(key.ModCtrl|key.ModShift)) {
		gtx.Execute(clipboard.ReadCmd{Tag: &th.in.term})
		return true
	}
	return false
}

// termCaret is the input method's caret: on the cursor's cell.
func (th *Theme) termCaret(gtx C, tp *TermPane, p *termPane) key.Caret {
	if tp.Screen == nil {
		return key.Caret{}
	}
	pos := tp.Screen.CursorPosition()
	cells := th.termMetrics(gtx)
	k := gtx.Metric.PxPerDp
	y := p.g.y0 + float32(p.g.sb+pos.Y-p.g.top)*p.g.ch
	return key.Caret{Pos: f32.Pt(p.g.x0+float32(pos.X)*p.g.cw, y+float32(cells.base)*k), Ascent: float32(cells.base) * k, Descent: p.g.ch - float32(cells.base)*k}
}

// termKey turns a key press into the emulator's: named keys, and letters
// with Ctrl (or Alt, which on a Mac types its own characters instead).
func termKey(e key.Event, mac bool) (uv.KeyPressEvent, bool) {
	var mod uv.KeyMod
	if e.Modifiers.Contain(key.ModCtrl) {
		mod |= uv.ModCtrl
	}
	if e.Modifiers.Contain(key.ModAlt) {
		mod |= uv.ModAlt
	}
	if e.Modifiers.Contain(key.ModShift) {
		mod |= uv.ModShift
	}
	if e.Modifiers.Contain(key.ModCommand) {
		return uv.KeyPressEvent{}, false // ⌘ chords are the app's
	}
	named := map[key.Name]rune{
		key.NameReturn: uv.KeyEnter, key.NameEnter: uv.KeyEnter, key.NameDeleteBackward: uv.KeyBackspace,
		key.NameDeleteForward: uv.KeyDelete, key.NameEscape: uv.KeyEscape, key.NameTab: uv.KeyTab,
		key.NameUpArrow: uv.KeyUp, key.NameDownArrow: uv.KeyDown, key.NameLeftArrow: uv.KeyLeft, key.NameRightArrow: uv.KeyRight,
		key.NameHome: uv.KeyHome, key.NameEnd: uv.KeyEnd, key.NamePageUp: uv.KeyPgUp, key.NamePageDown: uv.KeyPgDown,
		key.NameF1: uv.KeyF1, key.NameF2: uv.KeyF2, key.NameF3: uv.KeyF3, key.NameF4: uv.KeyF4, key.NameF5: uv.KeyF5, key.NameF6: uv.KeyF6,
		key.NameF7: uv.KeyF7, key.NameF8: uv.KeyF8, key.NameF9: uv.KeyF9, key.NameF10: uv.KeyF10, key.NameF11: uv.KeyF11, key.NameF12: uv.KeyF12,
	}
	if c, ok := named[e.Name]; ok {
		return uv.KeyPressEvent{Code: c, Mod: mod}, true
	}
	ctrl := mod&uv.ModCtrl != 0
	alt := mod&uv.ModAlt != 0 && !mac
	if !ctrl && !alt {
		return uv.KeyPressEvent{}, false // typing arrives as text
	}
	if e.Name == key.NameSpace {
		return uv.KeyPressEvent{Code: uv.KeySpace, Mod: mod}, true
	}
	rs := []rune(string(e.Name))
	if len(rs) != 1 {
		return uv.KeyPressEvent{}, false
	}
	c := []rune(strings.ToLower(string(rs[0])))[0]
	if !mac || ctrl {
		mod &^= uv.ModShift
	}
	k := uv.KeyPressEvent{Code: c, Mod: mod}
	if alt && !ctrl {
		k.Text = string(c)
	}
	return k, true
}
