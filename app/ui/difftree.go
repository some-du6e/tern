package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/some-du6e/tern/app/diff"
)

// The Diff surface's file tree pane (DiffFileTree.tsx on @pierre/trees,
// FileExplorerPane.tsx; docs/ui/13-diff.md §2.7, §3.10): the changed
// files as a tree beside the code view, in Pierre's own compact style,
// not the changed files card's. A click reveals a file in the code view.

// The pane's widths (FileExplorerPane.tsx:8-10).
const (
	treeMin      = 160
	treeMax      = 560
	treeDefault  = 256
	treeFileMinW = 256 // the code view keeps at least this
	treeRowH     = 24
	treeIndent   = 18.69 // a level's step (§3.10)
)

// treePanePx is the pane's width in px beside a row w px wide: the
// dragged width, or 40% of the row up to 256, kept between 160 and the
// row less 256.
func (th *Theme) treePanePx(gtx C, w int) int {
	row := float32(w) / gtx.Metric.PxPerDp
	initial := max(treeMin, min(treeDefault, 0.4*row))
	most := max(treeMin, min(treeMax, max(row-treeFileMinW, initial)))
	tw := initial
	if th.dv.treeW > 0 {
		tw = th.dv.treeW
	}
	return dp(gtx, max(treeMin, min(tw, most)))
}

// treePane is the pane: its left border and resize handle, the "Files"
// header, then the rows.
func (th *Theme) treePane(gtx C, d *DiffPanel) {
	size := gtx.Constraints.Max
	// The pointer over the tree shows its indent guides.
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &th.dv.treeTag, Kinds: pointer.Enter | pointer.Leave | pointer.Move})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok {
			th.dv.treeHot = e.Kind != pointer.Leave
		}
	}
	defer func() {
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, &th.dv.treeTag)
	}()
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	border := max(1, dp(gtx, 1))
	vline(gtx, over(twAlpha(P.Border, 60), P.Background), 0, 0, size.Y)
	func() {
		// The 4dp handle inside the border: a line on hover, dragged to
		// resize.
		h := &th.dv.treeEdge
		if dx := th.dragEdge(gtx, h, image.Pt(dp(gtx, 4), size.Y), true); dx != 0 {
			th.dv.treeW = float32(size.X)/gtx.Metric.PxPerDp - dx/gtx.Metric.PxPerDp
		}
		if h.hovered || h.held {
			vline(gtx, over(P.Border, P.Background), 0, 0, size.Y)
		}
	}()
	defer op.Offset(image.Pt(border, 0)).Push(gtx.Ops).Pop()
	w := size.X - border
	th.treeHeader(gtx, d, w)
	top := dp(gtx, diffHeaderH+diffHeaderGap)
	defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(w, max(0, size.Y-top)))
	th.treeRows(g, d)
}

// treeHeader is the pane's 28dp header: "Files", the count at the right,
// and the fold-all button when there are folders.
func (th *Theme) treeHeader(gtx C, d *DiffPanel, w int) {
	h := dp(gtx, diffHeaderH)
	folders := false
	for _, e := range d.Tree {
		folders = folders || e.Dir
	}
	x := dp(gtx, 8)
	func() {
		defer op.Offset(image.Pt(x+dp(gtx, 4), (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.Foreground, MaxLines: 1}, "Files")
	}()
	right := w - dp(gtx, 8)
	if folders {
		open := false
		for _, e := range d.Tree {
			open = open || e.Dir && e.Open
		}
		tip, icon := "Expand all folders", "chevrons-up-down"
		if open {
			tip, icon = "Collapse all folders", "chevrons-down-up"
		}
		right -= dp(gtx, 24)
		func() {
			defer op.Offset(image.Pt(right, (h-dp(gtx, 24))/2)).Push(gtx.Ops).Pop()
			th.diffToolW(diffTool{id: "diff.folders", tip: tip, icon: icon, w: 24, h: 24, radius: 8, hot: P.Accent, col: P.MutedForeground, tipAlign: 1}, P.Background)(gtx)
		}()
		right -= dp(gtx, 4)
	}
	count := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}
	n := strconv.Itoa(len(d.Files))
	defer op.Offset(image.Pt(right-th.lineWidth(gtx, count, n), (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
	th.Text(gtx, count, n)
}

// treeStatus is a file's git status in the tree: its letter and colour.
func treeStatus(t diff.ChangeType) (string, color.NRGBA) {
	c := dc()
	switch t {
	case diff.New:
		return "A", c.treeAdded
	case diff.Deleted:
		return "D", c.treeDeleted
	case diff.RenamePure, diff.RenameChanged:
		return "R", c.treeRenamed
	}
	return "M", c.treeModified
}

// treeRows lists the tree (§3.10): 24dp rows 16dp in from either side,
// indented 18.69dp a level; folders with Pierre's chevron and a change
// dot in the git lane, files with their icon, their name and their status
// letter in its colour.
func (th *Theme) treeRows(gtx C, d *DiffPanel) D {
	l := th.List("diff.tree", false)
	guides := 0.75 * th.flipped(gtx, "diff.tree.hot", th.dv.treeHot, 150*time.Millisecond)
	if !th.dv.treeHot {
		guides = 0.75 - guides
	}
	c := dc()
	return l.Layout(gtx, len(d.Tree), func(gtx C, i int) D {
		e := d.Tree[i]
		w := gtx.Constraints.Max.X
		h := dp(gtx, treeRowH)
		id := "diff.treefile:" + e.Path
		if e.Dir {
			id = "diff.folder:" + e.Path
		}
		ev, _ := th.events(gtx, id)
		r := image.Rect(dp(gtx, 16), 0, w-dp(gtx, 16), h)
		rad := dp(gtx, 6)
		switch {
		case !e.Dir && e.Path == d.Revealed:
			fill(gtx, over(c.treeSel, P.Background), r, rad)
		case ev.Hovered():
			fill(gtx, over(c.treeHover, P.Background), r, rad)
		}
		// A guide down each level it's in, at its folder's chevron.
		if guides > 0 {
			for k := 0; k < e.Depth; k++ {
				vline(gtx, over(alpha(rgb(0x84848a), uint8(0x40*guides)), P.Background), r.Min.X+dp(gtx, 6.4+float32(k)*treeIndent+8), 0, h)
			}
		}
		x := r.Min.X + dp(gtx, 6.4+float32(e.Depth)*treeIndent)
		name := Txt{Size: 12, LineH: 24.0 / 12, Color: P.Foreground, MaxLines: 1}
		lane := r.Max.X - dp(gtx, 6.4) - dp(gtx, 12)
		if e.Dir {
			func() {
				defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				if !e.Open {
					// A closed folder's chevron turns a quarter back.
					mid := float32(dp(gtx, 16)) / 2
					defer op.Affine(f32.Affine2D{}.Rotate(f32.Pt(mid, mid), -math.Pi/2)).Push(gtx.Ops).Pop()
				}
				spriteW("file-tree-icon-chevron", c.treeFg, P.Background, 16)(gtx)
			}()
			// Every folder here holds a change: the lane's dot.
			func() {
				defer op.Offset(image.Pt(lane+dp(gtx, 3), (h-dp(gtx, 6))/2)).Push(gtx.Ops).Pop()
				spriteW("file-tree-icon-dot", over(alpha(c.treeModified, 0x80), P.Background), P.Background, 6)(gtx)
			}()
		} else {
			letter, col := treeStatus(e.Type)
			icon, token := fileIcon(e.Path)
			ic := iconTone(token)
			if token == "" {
				ic = col
			}
			func() {
				defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				spriteW(icon, ic, P.Background, 16)(gtx)
			}()
			name.Color = col
			lt := Txt{Size: 12, LineH: 24.0 / 12, Font: UI(font.SemiBold), Color: col, MaxLines: 1}
			func() {
				defer op.Offset(image.Pt(lane+(dp(gtx, 12)-th.lineWidth(gtx, lt, letter))/2, 0)).Push(gtx.Ops).Pop()
				th.Text(gtx, lt, letter)
			}()
		}
		x += dp(gtx, 16+4.8)
		label := strings.Join(e.Segments, " / ")
		func() {
			defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
			defer clip.Rect{Max: image.Pt(max(0, lane-dp(gtx, 4.8)-x), h)}.Push(gtx.Ops).Pop()
			th.cutMiddle(gtx, name, label, lane-dp(gtx, 4.8)-x)
		}()
		defer op.Offset(r.Min).Push(gtx.Ops).Pop()
		th.hit(gtx, id, r.Size())
		return D{Size: image.Pt(w, h)}
	})
}

// cutMiddle sets s in w px, cutting its middle when it doesn't fit
// so its start and its extension stay in view ("compo… json").
func (th *Theme) cutMiddle(gtx C, t Txt, s string, w int) {
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(0, w), 1<<20)}
	if th.lineWidth(gtx, t, s) <= w {
		th.Text(g, t, s)
		return
	}
	r := []rune(s)
	tail := len(r) - min(len(r), 4)
	if i := strings.LastIndexByte(s, '.'); i > 0 {
		tail = len([]rune(s[:i]))
	}
	head := tail
	for head > 1 && th.lineWidth(gtx, t, string(r[:head])+"…"+string(r[tail:])) > w {
		head--
	}
	th.Text(g, t, string(r[:head])+"…"+string(r[tail:]))
}
