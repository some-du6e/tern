package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/some-du6e/tern/app/diff"
)

// The Diff surface's code view, as T3 draws it through @pierre/diffs
// (docs/ui/13-diff.md §2.4, §3.3-§3.8): one file after another, each a
// 32dp header that sticks to the top while its body scrolls under it,
// then its hunks, each after a separator naming the unchanged lines
// above it, then 8dp. Only what's in view is drawn.

// The viewer's fixed metrics (StyledDiffCodeView.tsx:253-264).
const (
	fileHeaderH = 32
	separatorH  = 24
	codeLineH   = 20
	fileBottom  = 8
	codeSize    = unit.Sp(13) // --diffs-font-size
	tabSize     = 2
)

// diffView is the Diff surface's own state between frames.
type diffView struct {
	ch      map[chKey]float32
	scroll  map[string]int // the code view's scroll top in px, per DiffPanel.Key
	xscroll map[string]int // a file's horizontal scroll in px, wrap off
	tag     int            // the code view's pointer events
	reveal  int            // the last RevealID scrolled to
	sel     lineSel
	drag    int // 0, or 1 while a number drag extends the selection, 2 while the scrollbar's thumb drags
	dragY   float32
	dragTop int
	hover   string // the file under the pointer
	sub     string // the scope menu's open submenu
	// Where the scope menu sits in the panel, for a submenu to flip.
	panelW, menuX int
	// The file tree pane's dragged width in dp, and its edge; and the
	// pointer over the tree, which shows its indent guides.
	treeW    float32
	treeEdge handle
	treeTag  int
	treeHot  bool
	layouts  map[layoutKey]*fileLayout
	rev      string // the files the layouts are for
	flips    map[string]flipState
}

// flipState is a two-way state and when it last changed, for its
// transition; zero for a state first seen as it is.
type flipState struct {
	on bool
	at time.Time
}

// flipped is how far id has eased toward on since it last changed, 0 to
// 1 over d: T3's 120, 150 and 200ms transitions. A state seen for the
// first time is already there.
func (th *Theme) flipped(gtx C, id string, on bool, d time.Duration) float32 {
	if th.dv.flips == nil {
		th.dv.flips = map[string]flipState{}
	}
	f, ok := th.dv.flips[id]
	switch {
	case !ok:
		f = flipState{on: on}
	case f.on != on:
		f = flipState{on, gtx.Now}
	}
	th.dv.flips[id] = f
	if f.at.IsZero() {
		return 1
	}
	t := clamp01(float32(gtx.Now.Sub(f.at)) / float32(d))
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	return easeOut(t)
}

// toward is how far a colour has gone from off to on, p of the way into
// the latest change.
func toward(off, on color.NRGBA, now bool, p float32) color.NRGBA {
	if now {
		return lerpCol(off, on, p)
	}
	return lerpCol(on, off, p)
}

// lineSel is a run of selected lines in one file, from row a to row b.
type lineSel struct {
	file string
	a, b int
}

// layoutKey names a file's layout at a wrap width.
type layoutKey struct {
	path  string
	cols  int // 0 with wrapping off
	split bool
}

// fileLayout is a file's body: its rows, their heights, and the gutter.
type fileLayout struct {
	rows   []codeRow
	h      int // the body's height in px, rows and the 8dp after them
	digits int // the widest line number's
	cols   int // the longest line's columns, for horizontal scroll
}

// codeRow is one row of a file's body: a separator over hidden unchanged
// lines, a line (in split view a pair of them), or the no-newline note.
type codeRow struct {
	hidden      int // a separator: the lines it stands for
	y, h        int // in px, in the body
	left, right *diff.Line
	noNewline   bool
	segs        int // wrapped lines
}

// codeView draws the files, the scrollbar on the right, and takes the
// pointer: the wheel scrolls, a number click selects a line.
func (th *Theme) codeView(gtx C, s *Scene, d *DiffPanel) {
	size := gtx.Constraints.Max
	fill(gtx, dc().bg, image.Rectangle{Max: size}, 0)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if th.dv.scroll == nil {
		th.dv.scroll, th.dv.xscroll = map[string]int{}, map[string]int{}
	}
	if r := d.rev(); th.dv.layouts == nil || th.dv.rev != r {
		th.dv.layouts, th.dv.rev = map[layoutKey]*fileLayout{}, r
	}
	ch := th.monoAdvance(gtx, codeSize)
	sb := dp(gtx, 6)
	// Lay out every file at the width with the scrollbar, and again
	// without it when it all fits.
	tops, total := th.fileTops(gtx, d, size.X-sb, ch)
	w := size.X - sb
	if total <= size.Y {
		tops, total = th.fileTops(gtx, d, size.X, ch)
		w = size.X
	}
	top := th.dv.scroll[d.Key]
	if d.RevealID != th.dv.reveal {
		th.dv.reveal = d.RevealID
		for i, f := range d.Files {
			if f.Path() == d.Revealed {
				top = tops[i]
			}
		}
	}
	top = th.codeInput(gtx, d, size, w, total, top, tops, ch)
	top = max(0, min(top, total-size.Y))
	th.dv.scroll[d.Key] = top

	// The bodies, then the headers over them, sticking to the top.
	for i := range d.Files {
		f := &d.Files[i]
		y0 := tops[i] - top
		y1 := y0 + dp(gtx, fileHeaderH)
		if !f.Collapsed {
			y1 += th.layoutFor(gtx, d, f, w, ch).h
		}
		// An unfolded body fades in over 200ms; folding cuts it at once.
		fadeIn := th.flipped(gtx, "unfold:"+d.Key+"\x00"+f.Path(), !f.Collapsed, 200*time.Millisecond)
		if y1 <= 0 || y0 >= size.Y {
			continue
		}
		if !f.Collapsed {
			func() {
				defer paint.PushOpacity(gtx.Ops, fadeIn).Pop()
				th.fileBody(gtx, d, f, image.Pt(0, y0+dp(gtx, fileHeaderH)), w, size.Y, ch)
			}()
		}
		hy := max(y0, min(0, y1-dp(gtx, fileHeaderH)))
		func() {
			defer op.Offset(image.Pt(0, hy)).Push(gtx.Ops).Pop()
			th.fileHeader(gtx, s, f, w)
		}()
	}
	if total > size.Y {
		th.scrollThumb(gtx, size, top, total)
	}
}

// fileTops is each file's top in the code view, at width w, and the total
// height.
func (th *Theme) fileTops(gtx C, d *DiffPanel, w int, ch float32) ([]int, int) {
	tops := make([]int, len(d.Files))
	y := 0
	for i := range d.Files {
		tops[i] = y
		y += dp(gtx, fileHeaderH)
		if !d.Files[i].Collapsed {
			y += th.layoutFor(gtx, d, &d.Files[i], w, ch).h
		}
	}
	return tops, y
}

// numberW is the line number cell's width: 2ch, the widest number's
// digits, 1ch, and its 2dp right border in the code background.
func numberW(gtx C, digits int, ch float32) int {
	return int(float32(3+digits)*ch+0.5) + dp(gtx, 2)
}

// layoutFor is f's body laid out at width w, cached until the files
// change (codeView checks).
func (th *Theme) layoutFor(gtx C, d *DiffPanel, f *DiffFile, w int, ch float32) *fileLayout {
	digits := 1
	for _, h := range f.Hunks {
		for _, l := range h.Lines {
			digits = max(digits, len(strconv.Itoa(max(l.Old, l.New))))
		}
	}
	cols := 0
	if d.Wrap {
		content := w - numberW(gtx, digits, ch)
		if d.Split {
			content = w/2 - numberW(gtx, digits, ch)
		}
		cols = max(1, int(float32(content)/ch)-2)
	}
	k := layoutKey{f.Path(), cols, d.Split}
	if l := th.dv.layouts[k]; l != nil {
		return l
	}
	l := &fileLayout{digits: digits}
	y := 0
	row := func(r codeRow) {
		r.y = y
		y += r.h
		l.rows = append(l.rows, r)
	}
	lineRows := func(ln *diff.Line) int {
		n := len([]rune(spaceTabs(ln.Text)))
		l.cols = max(l.cols, n)
		if cols == 0 {
			return 1
		}
		return len(wrapLine(spaceTabs(ln.Text), cols))
	}
	for hi := range f.Hunks {
		h := &f.Hunks[hi]
		if h.Hidden > 0 {
			row(codeRow{hidden: h.Hidden, h: dp(gtx, separatorH)})
		}
		lines := h.Lines
		for i := 0; i < len(lines); {
			ln := &lines[i]
			if !d.Split || ln.Kind == diff.Context {
				n := lineRows(ln)
				r := codeRow{left: ln, segs: n, h: n * dp(gtx, codeLineH)}
				if d.Split {
					r.right = ln
				}
				row(r)
				if ln.NoNewline {
					row(codeRow{noNewline: true, h: dp(gtx, codeLineH)})
				}
				i++
				continue
			}
			// A change in split view: its deletions beside its additions.
			j := i
			for j < len(lines) && lines[j].Kind == diff.Deletion {
				j++
			}
			end := j
			for end < len(lines) && lines[end].Kind == diff.Addition {
				end++
			}
			dels, adds := lines[i:j], lines[j:end]
			for p := 0; p < max(len(dels), len(adds)); p++ {
				r := codeRow{segs: 1}
				if p < len(dels) {
					r.left = &dels[p]
					r.segs = lineRows(r.left)
				}
				if p < len(adds) {
					r.right = &adds[p]
					r.segs = max(r.segs, lineRows(r.right))
				}
				r.h = r.segs * dp(gtx, codeLineH)
				row(r)
			}
			i = end
		}
	}
	l.h = y + dp(gtx, fileBottom)
	th.dv.layouts[k] = l
	return l
}

// rev names the panel's files and how they're shown, for the layout cache.
func (d *DiffPanel) rev() string {
	var b strings.Builder
	b.WriteString(d.Key + "\x00" + strconv.Itoa(d.Rev))
	for _, f := range d.Files {
		b.WriteString("\x00" + f.Path() + ":" + strconv.Itoa(f.Additions) + ":" + strconv.Itoa(f.Deletions) + ":" + strconv.Itoa(len(f.Hunks)))
	}
	return b.String()
}

// spaceTabs sets tabs to the next multiple of tabSize columns.
func spaceTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := tabSize - col%tabSize
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

// columns maps each byte offset of s, and its end, to the column it
// starts at once tabs are expanded.
func columns(s string) []int {
	cols := make([]int, len(s)+1)
	col := 0
	for i, r := range s {
		w := 1
		if r == '\t' {
			w = tabSize - col%tabSize
		}
		for j := i; j < i+utf8.RuneLen(r); j++ {
			cols[j] = col
		}
		col += w
	}
	cols[len(s)] = col
	return cols
}

// wrapLine breaks a line into rows of at most cols columns the way
// pre-wrap with break-word does: after a run of spaces where it can, mid
// word where a word is longer than a row. It returns each row's start
// column.
func wrapLine(s string, cols int) []int {
	runes := []rune(s)
	starts := []int{0}
	start := 0
	for start+cols < len(runes) {
		end := start + cols
		// Spaces at the end hang; a break goes after them.
		cut := end
		for cut < len(runes) && runes[cut] == ' ' {
			cut++
		}
		if cut == len(runes) {
			break
		}
		if cut == end {
			for b := end; b > start; b-- {
				if runes[b-1] == ' ' {
					cut = b
					break
				}
			}
		}
		start = cut
		starts = append(starts, start)
	}
	return starts
}

// fileHeader is a file's 32dp header (§3.3): the fold chevron, its change
// icon, the path cut from its start, the copy-path button while the file
// is hovered, and its counts at the right. A click folds it.
func (th *Theme) fileHeader(gtx C, s *Scene, f *DiffFile, w int) {
	c := dc()
	h := dp(gtx, fileHeaderH)
	id, fid, tid, cid := "diff.file:"+f.Path(), "diff.fold:"+f.Path(), "diff.title:"+f.Path(), "diff.copy:"+f.Path()
	ev, _ := th.events(gtx, id)
	fc, _ := th.events(gtx, fid)
	tc, _ := th.events(gtx, tid)
	cc, _ := th.events(gtx, cid)
	fill(gtx, c.bg, image.Rect(0, 0, w, h), 0)
	th.hit(gtx, id, image.Pt(w, h))
	if ev.Hovered() || fc.Hovered() || tc.Hovered() || cc.Hovered() {
		fill(gtx, over(c.edge, c.bg), image.Rect(0, 0, dp(gtx, 3), h), 0)
	}
	// The chevron's 20dp ghost button, 2dp into its 18dp slot at x 8.
	x := dp(gtx, 8)
	func() {
		defer op.Offset(image.Pt(x-dp(gtx, 2), (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
		if fc.Hovered() {
			fill(gtx, over(P.Accent, c.bg), image.Rect(0, 0, dp(gtx, 20), dp(gtx, 20)), dp(gtx, 4))
		}
		icon, tip := "chevron-down", "Collapse diff"
		if f.Collapsed {
			icon, tip = "chevron-right", "Expand diff"
		}
		func() {
			defer op.Offset(image.Pt(dp(gtx, 2), dp(gtx, 2))).Push(gtx.Ops).Pop()
			LucideW(icon, over(alpha(P.MutedForeground, 0xcc), c.bg), 16)(gtx)
		}()
		th.hit(gtx, fid, image.Pt(dp(gtx, 20), dp(gtx, 20)))
		th.tipAt(gtx, fid, tip, image.Pt(dp(gtx, 20), dp(gtx, 20)), true, -1)
	}()
	x += dp(gtx, 18+8)
	func() {
		defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		icon, col := changeIcon(f.Type)
		spriteW(icon, col, c.bg, 16)(gtx)
	}()
	x += dp(gtx, 16+8)
	// The counts, right to left from 12dp in: deletions, then additions,
	// 1ch of 12px sans apart, in mono 11.
	right := w - dp(gtx, 12)
	for i, n := range headerCounts(f) {
		t := Txt{Size: 11, LineH: 1, Font: Mono(), Color: c.del, MaxLines: 1}
		if n[0] == '+' {
			t.Color = c.add
		}
		if i > 0 {
			right -= int(th.uiAdvance(gtx, 12) + 0.5)
		}
		right -= th.lineWidth(gtx, t, n)
		func() {
			defer op.Offset(image.Pt(right, (h-dp(gtx, 11))/2)).Push(gtx.Ops).Pop()
			th.Text(gtx, t, n)
		}()
	}
	// The path, its start cut away when it doesn't fit, so its end shows.
	// The path turns toward primary on hover, over 120ms.
	title := Txt{Size: 12, LineH: 16.0 / 12, Color: c.title, MaxLines: 1}
	title.Color = toward(c.title, c.titleHot, tc.Hovered(), th.flipped(gtx, tid, tc.Hovered(), 120*time.Millisecond))
	room := right - dp(gtx, 8) - x - dp(gtx, 8+20)
	if f.PrevName != "" {
		prev := title
		prev.Color = over(alpha(title.Color, 0xb3), c.bg)
		pw := min(th.lineWidth(gtx, prev, f.PrevName), room/2)
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			th.textFromEnd(gtx, prev, f.PrevName, pw)
		}()
		x += pw
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			spriteW("diffs-icon-arrow-right-short", title.Color, c.bg, 16)(gtx)
		}()
		x += dp(gtx, 16)
		room -= pw + dp(gtx, 16)
	}
	tw := min(th.lineWidth(gtx, title, f.Name), max(0, room))
	func() {
		defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		th.textFromEnd(gtx, title, f.Name, tw)
		th.tipHover(gtx, tid, image.Pt(tw, dp(gtx, 16)))
	}()
	x += tw + dp(gtx, 8)
	// The copy-path button shows while the file is hovered (a mockup's
	// always shows it).
	if th.in.copies == nil {
		th.in.copies = map[string]string{}
	}
	if th.in.copied == nil {
		th.in.copied = map[string]time.Time{}
	}
	th.in.copies[cid] = f.Path()
	if !s.Live || th.dv.hover == f.Path() || ev.Hovered() || cc.Hovered() || th.copiedRecently(gtx, cid) {
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
			icon, col := "copy", P.MutedForeground
			if cc.Hovered() {
				fill(gtx, over(P.Accent, c.bg), image.Rect(0, 0, dp(gtx, 20), dp(gtx, 20)), dp(gtx, 4))
				col = P.Foreground
			}
			tip := "Copy path"
			if th.copiedRecently(gtx, cid) {
				icon, col, tip = "check", P.Success, "Copied"
			}
			func() {
				defer op.Offset(image.Pt(dp(gtx, 4), dp(gtx, 4))).Push(gtx.Ops).Pop()
				LucideW(icon, col, 12)(gtx)
			}()
			th.hit(gtx, cid, image.Pt(dp(gtx, 20), dp(gtx, 20)))
			th.tipAt(gtx, cid, tip, image.Pt(dp(gtx, 20), dp(gtx, 20)), true, 0)
		}()
	}
}

// headerCounts are a file header's counts from the right: Pierre shows the
// additions unless only lines were deleted, the deletions unless only
// lines were added, and nothing for a new empty file. Counts group their
// thousands.
func headerCounts(f *DiffFile) []string {
	if f.Type == diff.New && f.Additions == 0 && f.Deletions == 0 {
		return nil
	}
	var out []string
	if f.Additions > 0 || f.Deletions == 0 {
		out = append(out, "+"+groupThousands(f.Additions))
	}
	if f.Deletions > 0 || f.Additions == 0 {
		out = append(out, "-"+groupThousands(f.Deletions))
	}
	return out
}

// changeIcon is a file header's change icon and its colour (§3.4).
func changeIcon(t diff.ChangeType) (string, color.NRGBA) {
	c := dc()
	switch t {
	case diff.New:
		return "diffs-icon-symbol-added", c.added
	case diff.Deleted:
		return "diffs-icon-symbol-deleted", c.deleted
	case diff.RenamePure, diff.RenameChanged:
		return "diffs-icon-symbol-moved", c.modified
	}
	return "diffs-icon-symbol-modified", c.modified
}

// groupThousands writes n with commas, as toLocaleString does in English.
func groupThousands(n int) string {
	s := strconv.Itoa(n)
	var b []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, s[i])
	}
	return string(b)
}

// uiAdvance is 1ch of the UI font at size, in px.
func (th *Theme) uiAdvance(gtx C, size unit.Sp) float32 {
	return float32(th.lineWidth(gtx, Txt{Size: size}, "0000000000")) / 10
}

// textFromEnd sets s in w px, cutting its start with "…" when it doesn't
// fit, the way T3's rtl ellipsis keeps a path's end in view.
func (th *Theme) textFromEnd(gtx C, t Txt, s string, w int) {
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(0, w), 1<<20)}
	if th.lineWidth(gtx, t, s) <= w {
		th.Text(g, t, s)
		return
	}
	r := []rune(s)
	lo, hi := 0, len(r)
	for lo < hi {
		mid := (lo + hi) / 2
		if th.lineWidth(gtx, t, "…"+string(r[mid:])) <= w {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	th.Text(g, t, "…"+string(r[lo:]))
}

// fileBody draws f's rows that are in view, its top at at.
func (th *Theme) fileBody(gtx C, d *DiffPanel, f *DiffFile, at image.Point, w, viewH int, ch float32) {
	l := th.layoutFor(gtx, d, f, w, ch)
	c := dc()
	numW := numberW(gtx, l.digits, ch)
	x := th.dv.xscroll[f.Path()]
	if d.Wrap {
		x = 0
	}
	for i := range l.rows {
		r := &l.rows[i]
		y := at.Y + r.y
		if y+r.h <= 0 || y >= viewH {
			continue
		}
		func() {
			defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
			switch {
			case r.hidden > 0:
				th.separatorRow(gtx, r.hidden, w, d.Split)
			case r.noNewline:
				fill(gtx, c.bg, image.Rect(0, 0, w, r.h), 0)
				defer op.Offset(image.Pt(numW+int(ch+0.5), 0)).Push(gtx.Ops).Pop()
				defer paint.PushOpacity(gtx.Ops, 0.6).Pop()
				th.Text(gtx, Txt{Size: codeSize, LineH: 20.0 / 13, Font: Mono(), Color: c.text, MaxLines: 1}, "No newline at end of file")
			case d.Split:
				half := w / 2
				sel := th.selected(f.Path(), i)
				th.codeCell(gtx, r.left, false, image.Rect(0, 0, half, r.h), numW, ch, x, sel, d.Wrap)
				th.codeCell(gtx, r.right, true, image.Rect(half, 0, w, r.h), numW, ch, x, sel, d.Wrap)
				vline(gtx, c.bg, half-max(1, dp(gtx, 1)), 0, r.h)
				vline(gtx, c.bg, half, 0, r.h)
			default:
				th.codeCell(gtx, r.left, r.left.Kind == diff.Addition || r.left.Kind == diff.Context, image.Rect(0, 0, w, r.h), numW, ch, x, th.selected(f.Path(), i), d.Wrap)
			}
		}()
	}
}

// SelectDiffLines selects rows a to b of file in the code view, as a
// click on a line number and a drag would.
func (th *Theme) SelectDiffLines(file string, a, b int) { th.dv.sel = lineSel{file, a, b} }

// selected reports whether row i of file is in the line selection.
func (th *Theme) selected(file string, i int) bool {
	s := th.dv.sel
	return s.file == file && i >= min(s.a, s.b) && i <= max(s.a, s.b)
}

// separatorRow is a hunk separator (§3.6): a 24dp pill inset 8dp, in the
// separator colour, "N unmodified lines" in 11px sans. Split view puts
// the pill on the deletions' side and a plain bar on the additions'.
func (th *Theme) separatorRow(gtx C, hidden, w int, split bool) {
	c := dc()
	h := dp(gtx, separatorH)
	fill(gtx, c.bg, image.Rect(0, 0, w, h), 0)
	rad := dp(gtx, 6)
	pill := image.Rect(dp(gtx, 8), 0, w-dp(gtx, 8), h)
	if split {
		pill.Max.X = w/2 - max(1, dp(gtx, 1))
		fill(gtx, c.sepBg, image.Rect(w/2+max(1, dp(gtx, 1)), 0, w-dp(gtx, 8), h), rad)
		fill(gtx, c.sepBg, image.Rect(w/2+max(1, dp(gtx, 1)), 0, w/2+rad+max(1, dp(gtx, 1)), h), 0)
	}
	fill(gtx, c.sepBg, pill, rad)
	if split {
		fill(gtx, c.sepBg, image.Rect(pill.Max.X-rad, 0, pill.Max.X, h), 0)
	}
	label := plural(hidden, "unmodified line")
	t := Txt{Size: 11, LineH: 20.0 / 11, Color: c.sepText, MaxLines: 1}
	inset := int(th.uiAdvance(gtx, 11) + 0.5)
	defer op.Offset(image.Pt(pill.Min.X+inset, (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(0, pill.Dx()-2*inset), dp(gtx, 20))}
	th.Text(g, t, label)
}

// codeCell is one side of a row: the number cell with its change or
// selection bar, then the code, its changed words marked. newSide numbers
// it with the new file's lines. An empty side is the hatched buffer.
func (th *Theme) codeCell(gtx C, ln *diff.Line, newSide bool, cell image.Rectangle, numW int, ch float32, xscroll int, sel, wrap bool) {
	c := dc()
	defer op.Offset(cell.Min).Push(gtx.Ops).Pop()
	w, h := cell.Dx(), cell.Dy()
	if ln == nil {
		// The side with no counterpart: its gutter in the context gutter's
		// colour, then diagonal hatching in bg 92% toward the text's end.
		fill(gtx, c.bufferGutter, image.Rect(0, 0, numW-dp(gtx, 2), h), 0)
		fill(gtx, c.bg, image.Rect(numW-dp(gtx, 2), 0, w, h), 0)
		hatch(gtx, image.Rect(numW, 0, w, h), over(alpha(c.text, 0x14), c.bg))
		return
	}
	row, numBg, bar, num := c.bg, c.bg, color.NRGBA{}, c.number
	switch ln.Kind {
	case diff.Addition:
		row, numBg, bar, num = c.addRow, c.addNumBg, c.add, c.add
	case diff.Deletion:
		row, numBg, bar, num = c.delRow, c.delNumBg, c.del, c.del
	}
	if sel {
		row, numBg, bar, num = c.selRow, c.selNumBg, c.selBar, c.selNum
	}
	fill(gtx, row, image.Rect(numW, 0, w, h), 0)
	fill(gtx, numBg, image.Rect(0, 0, numW-dp(gtx, 2), h), 0)
	fill(gtx, c.bg, image.Rect(numW-dp(gtx, 2), 0, numW, h), 0)
	switch {
	case sel || ln.Kind == diff.Addition:
		fill(gtx, bar, image.Rect(0, 0, dp(gtx, 4), h), 0)
	case ln.Kind == diff.Deletion:
		// Dashed: in every 2dp, the base on top, the bar's own colour under.
		for y := 0; y < h; y += dp(gtx, 2) {
			fill(gtx, bar, image.Rect(0, y, dp(gtx, 4), y+dp(gtx, 1)), 0)
			fill(gtx, c.delDash, image.Rect(0, y+dp(gtx, 1), dp(gtx, 4), y+dp(gtx, 2)), 0)
		}
	}
	n := ln.Old
	if newSide && ln.Kind != diff.Deletion {
		n = ln.New
	}
	if ln.Kind == diff.Addition {
		n = ln.New
	}
	mono := Txt{Size: codeSize, LineH: 20.0 / 13, Font: Mono(), MaxLines: 1}
	func() {
		t := mono
		t.Color = num
		s := strconv.Itoa(n)
		right := numW - dp(gtx, 2) - int(ch+0.5)
		defer op.Offset(image.Pt(right-th.lineWidth(gtx, t, s), 0)).Push(gtx.Ops).Pop()
		th.Text(gtx, t, s)
	}()
	// The code, clipped to its column, scrolled when wrapping is off.
	defer clip.Rect{Min: image.Pt(numW, 0), Max: image.Pt(w, h)}.Push(gtx.Ops).Pop()
	text := spaceTabs(ln.Text)
	runes := []rune(text)
	starts := []int{0}
	if wrap {
		starts = wrapLine(text, max(1, int(float32(w-numW)/ch)-2))
	}
	cols := columns(ln.Text)
	x0 := numW + int(ch+0.5) - xscroll
	wordBg := c.addWord
	if ln.Kind == diff.Deletion {
		wordBg = c.delWord
	}
	for si, start := range starts {
		end := len(runes)
		if si+1 < len(starts) {
			end = starts[si+1]
		}
		y := si * dp(gtx, codeLineH)
		// The changed words behind the text, each piece of a wrapped one
		// rounded on its own.
		for _, rg := range ln.Words {
			a, b := max(cols[rg.Start], start), min(cols[rg.End], end)
			if a >= b {
				continue
			}
			wr := image.Rect(x0+int(float32(a-start)*ch+0.5), y+dp(gtx, 3), x0+int(float32(b-start)*ch+0.5), y+dp(gtx, 17))
			fill(gtx, over(wordBg, row), wr, dp(gtx, 3))
		}
		func() {
			defer op.Offset(image.Pt(x0, y)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(1<<20, dp(gtx, codeLineH))}
			t := mono
			t.Color, t.MaxLines = c.text, 0
			th.Text(g, t, string(runes[start:end]))
		}()
	}
}

// hatch fills r with Pierre's buffer stripes: 1.4dp lines at -45°, 8dp
// apart, in col.
func hatch(gtx C, r image.Rectangle, col color.NRGBA) {
	defer clip.Rect(r).Push(gtx.Ops).Pop()
	step := dp(gtx, 8)
	w := float32(dp(gtx, 1.4))
	for x := r.Min.X - r.Dy(); x < r.Max.X; x += step {
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(f32.Pt(float32(x), float32(r.Max.Y)))
		p.LineTo(f32.Pt(float32(x+r.Dy()), float32(r.Min.Y)))
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: w}.Op())
	}
}

// scrollThumb is the code view's 6dp scrollbar thumb on the right.
func (th *Theme) scrollThumb(gtx C, size image.Point, top, total int) {
	sb := dp(gtx, 6)
	h := max(dp(gtx, 16), size.Y*size.Y/max(1, total))
	y := (size.Y - h) * top / max(1, total-size.Y)
	col := rgb(0xd9d9d9)
	if th.dv.drag == 2 {
		col = rgb(0xbfbfbf)
	}
	if dark {
		col = over(alpha(rgb(0xffffff), 0x14), dc().bg)
		if th.dv.drag == 2 {
			col = over(alpha(rgb(0xffffff), 0x1f), dc().bg)
		}
	}
	fill(gtx, col, image.Rect(size.X-sb, y, size.X, y+h), sb/2)
}

// codeInput takes the code view's pointer: the wheel scrolls it (and a
// file across with wrapping off), a press on a line number selects that
// line, shift extends it, a drag extends it, and the scrollbar's lane
// drags. It returns the new scroll top.
func (th *Theme) codeInput(gtx C, d *DiffPanel, size image.Point, w, total, top int, tops []int, ch float32) int {
	tag := &th.dv.tag
	maxTop := max(0, total-size.Y)
	// The row under a point: its file, and its index in that file's rows.
	at := func(p image.Point) (fi, ri int, gutter bool) {
		y := p.Y + top
		fi, ri = -1, -1
		for i := range d.Files {
			if y >= tops[i] {
				fi = i
			}
		}
		if fi < 0 {
			return
		}
		f := &d.Files[fi]
		by := y - tops[fi] - dp(gtx, fileHeaderH)
		if f.Collapsed || by < 0 {
			return
		}
		l := th.layoutFor(gtx, d, f, w, ch)
		for i, r := range l.rows {
			if by >= r.y && by < r.y+r.h && r.left != nil {
				ri = i
			}
		}
		nw := numberW(gtx, l.digits, ch)
		gutter = p.X < nw || d.Split && p.X >= w/2 && p.X < w/2+nw
		return
	}
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Move | pointer.Leave | pointer.Scroll,
				ScrollX: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}, ScrollY: pointer.ScrollRange{Min: -top, Max: maxTop - top}},
			key.FocusFilter{Target: tag},
		)
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		p := e.Position.Round()
		switch e.Kind {
		case pointer.Scroll:
			top += int(e.Scroll.Y)
			if e.Scroll.X != 0 && !d.Wrap {
				if fi, _, _ := at(p); fi >= 0 {
					path := d.Files[fi].Path()
					l := th.layoutFor(gtx, d, &d.Files[fi], w, ch)
					most := max(0, int(float32(l.cols+2)*ch)+numberW(gtx, l.digits, ch)-w)
					th.dv.xscroll[path] = max(0, min(most, th.dv.xscroll[path]+int(e.Scroll.X)))
				}
			}
		case pointer.Move:
			th.dv.hover = ""
			if fi, _, _ := at(p); fi >= 0 {
				th.dv.hover = d.Files[fi].Path()
			}
		case pointer.Leave:
			th.dv.hover = ""
		case pointer.Press:
			gtx.Execute(key.FocusCmd{Tag: tag})
			if p.X >= size.X-dp(gtx, 6) && total > size.Y {
				th.dv.drag, th.dv.dragY, th.dv.dragTop = 2, e.Position.Y, top
				break
			}
			fi, ri, gutter := at(p)
			if fi < 0 || ri < 0 || !gutter {
				break
			}
			path := d.Files[fi].Path()
			if e.Modifiers.Contain(key.ModShift) && th.dv.sel.file == path {
				th.dv.sel.b = ri
			} else {
				th.dv.sel = lineSel{path, ri, ri}
			}
			th.dv.drag = 1
		case pointer.Drag:
			switch th.dv.drag {
			case 2:
				thumb := max(dp(gtx, 16), size.Y*size.Y/max(1, total))
				if span := size.Y - thumb; span > 0 {
					top = th.dv.dragTop + int((e.Position.Y-th.dv.dragY)*float32(maxTop)/float32(span))
				}
			case 1:
				if fi, ri, _ := at(p); fi >= 0 && ri >= 0 && d.Files[fi].Path() == th.dv.sel.file {
					th.dv.sel.b = ri
				}
			}
		case pointer.Release:
			th.dv.drag = 0
		}
	}
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	area.Pop()
	return top
}
