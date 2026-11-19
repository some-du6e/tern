package ui

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"

	"github.com/some-du6e/tern/app/diff"
)

// The changed files card under an assistant message (ChangedFilesTree.tsx;
// docs/ui/13-diff.md §2.8, §3.11): what the message's run changed, as a
// tree, with Open diff. Targets: "files.open:<run>", "files.all:<run>",
// "files.folder:<run>:<path>", "files.file:<run>:<path>".

// ChangedFiles is one run's card.
type ChangedFiles struct {
	Run                  string
	Count                int // all the files the run changed
	Additions, Deletions int
	Nodes                []diff.CardNode // the files it lists, as a tree
	// AllOpen opens every folder; Open flips one folder from that.
	AllOpen bool
	Open    map[string]bool
}

// cardRow is a node of the card's tree and how deep it is.
type cardRow struct {
	node  diff.CardNode
	depth int
}

func (f *ChangedFiles) rows() []cardRow {
	var rows []cardRow
	var walk func(nodes []diff.CardNode, depth int)
	walk = func(nodes []diff.CardNode, depth int) {
		for _, n := range nodes {
			rows = append(rows, cardRow{n, depth})
			if n.Dir && f.open(n.Path) {
				walk(n.Children, depth+1)
			}
		}
	}
	walk(f.Nodes, 0)
	return rows
}

func (f *ChangedFiles) open(path string) bool {
	if o, ok := f.Open[path]; ok {
		return o
	}
	return f.AllOpen
}

// changedFilesCard is the card, 16dp under the message, as wide as the
// lane: its 40dp header with the count, the totals and the buttons, the
// rows, and a button for the files it doesn't list.
func (th *Theme) changedFilesCard(gtx C, f *ChangedFiles) D {
	w := gtx.Constraints.Max.X
	bg := over(P.Secondary, P.Background)
	if dark {
		bg = over(twAlpha(P.Input, 20), P.Background)
	}
	listed := 0
	folders := false
	var count func(nodes []diff.CardNode)
	count = func(nodes []diff.CardNode) {
		for _, n := range nodes {
			if n.Dir {
				folders = true
				count(n.Children)
			} else {
				listed++
			}
		}
	}
	count(f.Nodes)
	rows := f.rows()
	more := f.Count - listed

	m := op.Record(gtx.Ops)
	y := th.cardHeader(gtx, f, w, bg, folders)
	y += dp(gtx, 8)
	for _, r := range rows {
		func() {
			defer op.Offset(image.Pt(dp(gtx, 8), y)).Push(gtx.Ops).Pop()
			th.cardFileRow(gtx, f, r, w-dp(gtx, 16), bg, folders)
		}()
		y += dp(gtx, 28)
	}
	y += dp(gtx, 8)
	if more > 0 {
		func() {
			defer op.Offset(image.Pt(dp(gtx, 12), y)).Push(gtx.Ops).Pop()
			label := groupThousands(more) + " more " + pluralWord(more, "file") + " in the full diff"
			y += th.mutedButton(gtx, "files.open:"+f.Run, "", label, bg, true).Y + dp(gtx, 8)
		}()
	}
	body := m.Stop()
	top := dp(gtx, 16)
	defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
	fill(gtx, bg, image.Rect(0, 0, w, y), dp(gtx, 8))
	body.Add(gtx.Ops)
	return D{Size: image.Pt(w, top+y)}
}

// pluralWord is word for one, else word with an s.
func pluralWord(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// cardHeader is the card's 40dp header, padded 8 by 12: "N changed
// files" in 12/16 medium and the totals 12dp after it, then Expand all
// folders (with folders) and Open diff, whose label shows only in a card
// at least 24rem wide. It returns its height.
func (th *Theme) cardHeader(gtx C, f *ChangedFiles, w int, bg color.NRGBA, folders bool) int {
	h := dp(gtx, 40)
	right := w - dp(gtx, 12)
	open := th.measure(gtx, func(gtx C) image.Point {
		return th.mutedButton(gtx, "files.open:"+f.Run, "file-diff", "Open diff", bg, w >= dp(gtx, 384))
	})
	right -= open.X
	func() {
		defer op.Offset(image.Pt(right, (h-open.Y)/2)).Push(gtx.Ops).Pop()
		th.mutedButton(gtx, "files.open:"+f.Run, "file-diff", "Open diff", bg, w >= dp(gtx, 384))
		th.tipAt(gtx, "files.open:"+f.Run, "Open the full diff", open, true, 0)
	}()
	if folders {
		right -= dp(gtx, 4+24)
		tip, icon := "Expand all folders", "chevrons-up-down"
		if f.AllOpen {
			tip, icon = "Collapse all folders", "chevrons-down-up"
		}
		func() {
			defer op.Offset(image.Pt(right, (h-dp(gtx, 24))/2)).Push(gtx.Ops).Pop()
			id := "files.all:" + f.Run
			c, _ := th.events(gtx, id)
			col := P.MutedForeground
			if c.Hovered() {
				fill(gtx, over(P.Accent, bg), image.Rect(0, 0, dp(gtx, 24), dp(gtx, 24)), dp(gtx, 8))
				col = P.Foreground
			}
			func() {
				defer op.Offset(image.Pt(dp(gtx, 6), dp(gtx, 6))).Push(gtx.Ops).Pop()
				LucideW(icon, col, 12)(gtx)
			}()
			th.hit(gtx, id, image.Pt(dp(gtx, 24), dp(gtx, 24)))
			th.tipAt(gtx, id, tip, image.Pt(dp(gtx, 24), dp(gtx, 24)), true, 0)
		}()
	}
	x := dp(gtx, 12)
	title := Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.Foreground, MaxLines: 1}
	label := groupThousands(f.Count) + " changed " + pluralWord(f.Count, "file")
	func() {
		defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		x += th.Text(gtx, title, label).Size.X + dp(gtx, 12)
	}()
	if f.Additions > 0 || f.Deletions > 0 {
		defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		th.diffStat(gtx, f.Additions, f.Deletions, false, 12, 16, font.Medium)
	}
	return h
}

// measure is the size w lays out, without drawing it.
func (th *Theme) measure(gtx C, w func(gtx C) image.Point) image.Point {
	m := op.Record(gtx.Ops)
	s := w(gtx)
	m.Stop()
	return s
}

// mutedButton is T3's xs ghost-muted Button: 24 tall, padded 7, a 12dp
// icon 2dp into the padding, 4dp, then the label in 12/16 medium; muted,
// foreground over accent on hover. It returns its size.
func (th *Theme) mutedButton(gtx C, id, icon, label string, on color.NRGBA, showLabel bool) image.Point {
	c, _ := th.events(gtx, id)
	col := P.MutedForeground
	t := Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), MaxLines: 1}
	px := max(1, dp(gtx, 1))
	w := 2*dp(gtx, 7) + 2*px
	if icon != "" {
		w += dp(gtx, 12-4)
	}
	if showLabel {
		if icon != "" {
			w += dp(gtx, 4)
		}
		w += th.lineWidth(gtx, t, label)
	}
	h := dp(gtx, 24)
	if c.Hovered() {
		fill(gtx, over(P.Accent, on), image.Rect(0, 0, w, h), dp(gtx, 8))
		col = P.Foreground
	}
	x := px + dp(gtx, 7)
	if icon != "" {
		func() {
			defer op.Offset(image.Pt(x-dp(gtx, 2), (h-dp(gtx, 12))/2)).Push(gtx.Ops).Pop()
			LucideW(icon, col, 12)(gtx)
		}()
		x += dp(gtx, 12-4+4)
	}
	if showLabel {
		t.Color = col
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			th.Text(gtx, t, label)
		}()
	}
	th.hit(gtx, id, image.Pt(w, h))
	return image.Pt(w, h)
}

// cardFileRow is one 28dp row of the card's tree, indented 14dp a level:
// a folder's chevron (turned when open), its folder icon, its name in mono
// 11 and its totals; or a file's icon, its name in mono 12 and its counts,
// lined up with the folders' names when there are any.
func (th *Theme) cardFileRow(gtx C, f *ChangedFiles, r cardRow, w int, bg color.NRGBA, folders bool) {
	n := r.node
	id := "files.file:" + f.Run + ":" + n.Path
	if n.Dir {
		id = "files.folder:" + f.Run + ":" + n.Path
	}
	c, _ := th.events(gtx, id)
	hot := c.Hovered()
	// Its colours and background ease in and out over 150ms.
	ease := th.flipped(gtx, "hot:"+id, hot, 150*time.Millisecond)
	tone := func(off, on color.NRGBA) color.NRGBA { return toward(off, on, hot, ease) }
	h := dp(gtx, 28)
	if hl := tone(bg, over(twAlpha(P.Accent, 60), bg)); hl != bg {
		fill(gtx, hl, image.Rect(0, 0, w, h), dp(gtx, 6))
	}
	x := dp(gtx, 8+14*float32(r.depth))
	right := w - dp(gtx, 8)
	if !n.Dir || n.Additions > 0 || n.Deletions > 0 {
		sw := th.measure(gtx, func(gtx C) image.Point {
			return th.diffStat(gtx, n.Additions, n.Deletions, true, 10, 14, font.Normal).Size
		})
		right -= sw.X
		func() {
			defer op.Offset(image.Pt(right, (h-sw.Y)/2)).Push(gtx.Ops).Pop()
			th.diffStat(gtx, n.Additions, n.Deletions, true, 10, 14, font.Normal)
		}()
		right -= dp(gtx, 8)
	}
	if n.Dir {
		open := f.open(n.Path)
		chev := tone(over(twAlpha(P.MutedForeground, 70), bg), over(twAlpha(P.Foreground, 80), bg))
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
			// It turns a quarter as the folder opens, over 150ms.
			turn := th.flipped(gtx, "open:"+id, open, 150*time.Millisecond)
			if !open {
				turn = 1 - turn
			}
			mid := float32(dp(gtx, 14)) / 2
			defer op.Affine(f32.Affine2D{}.Rotate(f32.Pt(mid, mid), turn*math.Pi/2)).Push(gtx.Ops).Pop()
			LucideW("chevron-right", chev, 14)(gtx)
		}()
		x += dp(gtx, 14+8)
		icon := "folder-closed"
		if open {
			icon = "folder"
		}
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
			LucideW(icon, over(alpha(P.MutedForeground, 0xbf), bg), 14)(gtx)
		}()
		x += dp(gtx, 14+8)
		name := Txt{Size: 11, LineH: 16.0 / 11, Font: Mono(), MaxLines: 1,
			Color: tone(over(twAlpha(P.MutedForeground, 90), bg), over(twAlpha(P.Foreground, 90), bg))}
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(max(0, right-x), dp(gtx, 16))}
			th.Text(g, name, n.Name)
		}()
	} else {
		if folders || r.depth > 0 {
			x += dp(gtx, 14+8)
		}
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
			fileIconW(n.Path, 14, bg)(gtx)
		}()
		x += dp(gtx, 14+8)
		name := Txt{Size: 12, LineH: 16.0 / 12, Font: Mono(), MaxLines: 1, Color: tone(over(twAlpha(P.Foreground, 85), bg), P.Foreground)}
		func() {
			defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			th.cutMiddle(gtx, name, n.Name, right-x)
		}()
	}
	th.hit(gtx, id, image.Pt(w, h))
}
