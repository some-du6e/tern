package ui

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/some-du6e/tern/app/diff"
)

// A count reads as T3's formatCompactDiffCount writes it, at its
// boundaries (docs/ui/13-diff.md §5.11, §8 note 10).
func TestCompactCount(t *testing.T) {
	for n, want := range map[int]string{
		0: "0", 999: "999", 1000: "1k", 1234: "1.2k", 9950: "9.9k", 10000: "10k", 12345: "12k",
		999_499: "999k", 999_500: "1m", 1_000_000: "1m", 12_500_000: "13m", 999_500_000: "1b",
	} {
		if got := CompactCount(n); got != want {
			t.Errorf("CompactCount(%d) = %q, want %q", n, got, want)
		}
	}
}

func measureCtx() layout.Context {
	var ops op.Ops
	return layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(1000, 100)}}
}

// An aligned DiffStatLabel is two right-aligned 4ch columns 8 apart: 56
// wide in a card row's mono 10 (§3.11).
func TestDiffStatAligned(t *testing.T) {
	th := NewTheme()
	if w := th.diffStat(measureCtx(), 12, 3, true, 10, 14, font.Normal).Size.X; w != 56 {
		t.Fatalf("aligned label is %d wide, want 56", w)
	}
}

// A file header shows its additions unless only lines went, its deletions
// unless only lines came, and nothing for a new empty file (§2.4).
func TestHeaderCounts(t *testing.T) {
	for _, c := range []struct {
		f    diff.File
		want []string
	}{
		{diff.File{Type: diff.Change, Additions: 2, Deletions: 1}, []string{"+2", "-1"}},
		{diff.File{Type: diff.Deleted, Deletions: 3}, []string{"-3"}},
		{diff.File{Type: diff.New, Additions: 2382}, []string{"+2,382"}},
		{diff.File{Type: diff.Change}, []string{"+0", "-0"}},
		{diff.File{Type: diff.New}, nil},
	} {
		if got := headerCounts(&DiffFile{File: c.f}); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v: %q, want %q", c.f, got, c.want)
		}
	}
}

const testPatch = `diff --git a/src/sum.ts b/src/sum.ts
--- a/src/sum.ts
+++ b/src/sum.ts
@@ -37,2 +37,2 @@ export function sum() {
 const a = 1;
-return alpha;
+return gamma;
`

func testPanel(t *testing.T, patch string) *DiffPanel {
	t.Helper()
	files, err := diff.Parse(patch)
	if err != nil {
		t.Fatal(err)
	}
	d := &DiffPanel{Active: true, Key: t.Name(), Scope: "Latest turn", Value: "latest", Wrap: true, Whitespace: true}
	for _, f := range files {
		d.Files = append(d.Files, DiffFile{File: f})
	}
	return d
}

// An unfolded file draws as T3's viewer measures (§3.5, §3.6): the 32dp
// header, a 24dp separator pill inset 8, then 20dp rows; a deletion's row
// and number cell tinted with a dashed bar, an addition's with a solid
// one, the 41dp number column's 2dp border in the background, and the
// changed word behind its text.
func TestCodeViewRows(t *testing.T) {
	th := NewTheme()
	d := testPanel(t, testPatch)
	c := diffLight
	img := drawn(t, image.Pt(539, 300), 1, func(gtx C) D {
		th.codeView(gtx, &Scene{}, d)
		return D{}
	})
	wantPixel(t, img, image.Pt(400, 44), c.sepBg, "separator pill")
	wantPixel(t, img, image.Pt(4, 44), c.bg, "separator's inset")
	wantPixel(t, img, image.Pt(300, 66), c.bg, "context row")
	wantPixel(t, img, image.Pt(300, 86), c.delRow, "deletion row")
	wantPixel(t, img, image.Pt(9, 86), c.delNumBg, "deletion number cell")
	wantPixel(t, img, image.Pt(40, 86), c.bg, "number column's border")
	wantPixel(t, img, image.Pt(1, 76), c.del, "deletion bar's dash")
	wantPixel(t, img, image.Pt(1, 77), c.delDash, "deletion bar's gap")
	wantPixel(t, img, image.Pt(300, 106), c.addRow, "addition row")
	wantPixel(t, img, image.Pt(9, 106), c.addNumBg, "addition number cell")
	wantPixel(t, img, image.Pt(1, 107), c.add, "addition bar")
	// "gamma" starts 7ch into the code, which starts 1ch past the 41dp
	// gutter: its emphasis's top edge, clear of the glyphs.
	wantPixel(t, img, image.Pt(49+55+8, 99), over(c.addWord, c.addRow), "changed word")
	wantPixel(t, img, image.Pt(300, 130), c.bg, "8dp after the last row, then nothing")
}

// A folded file is its header alone; the next file follows it.
func TestCodeViewFolded(t *testing.T) {
	th := NewTheme()
	d := testPanel(t, testPatch+strings.Replace(testPatch, "sum.ts", "two.ts", -1))
	d.Files[0].Collapsed = true
	c := diffLight
	img := drawn(t, image.Pt(539, 300), 1, func(gtx C) D {
		th.codeView(gtx, &Scene{}, d)
		return D{}
	})
	wantPixel(t, img, image.Pt(400, 32+44), c.sepBg, "the second file's separator, under the first's header")
}

// A file's header sticks to the top while its body scrolls under it.
func TestStickyHeader(t *testing.T) {
	th := NewTheme()
	var b strings.Builder
	b.WriteString("diff --git a/long.go b/long.go\n--- a/long.go\n+++ b/long.go\n@@ -40,30 +40,0 @@\n")
	for i := 0; i < 30; i++ {
		b.WriteString("-gone\n")
	}
	d := testPanel(t, b.String())
	th.dv.scroll = map[string]int{d.Key: 100}
	th.dv.xscroll = map[string]int{}
	c := diffLight
	img := drawn(t, image.Pt(539, 200), 1, func(gtx C) D {
		th.codeView(gtx, &Scene{}, d)
		return D{}
	})
	wantPixel(t, img, image.Pt(1, 16), c.bg, "the stuck header, over a deletion's bar")
	wantPixel(t, img, image.Pt(300, 60), c.delRow, "rows under it")
}

// The header row's toolbar sits where T3's does in a 539dp surface
// (§3.1): collapse all at 333, the stacked/split group 365-435 with its
// stacked segment raised, the wrap and whitespace toggles pressed at 439
// and 471, the file tree's not at 503.
func TestDiffToolbar(t *testing.T) {
	th := NewTheme()
	d := testPanel(t, testPatch)
	d.Stat = &DiffStat{Additions: 1, Deletions: 1}
	bg := P.Background
	img := drawn(t, image.Pt(539, 28), 1, func(gtx C) D {
		fill(gtx, bg, image.Rect(0, 0, 539, 28), 0)
		th.diffHeaderRow(gtx, &Scene{}, d)
		return D{}
	})
	accent := over(P.Accent, bg)
	wantPixel(t, img, image.Pt(336, 3), bg, "collapse all, a ghost button")
	wantPixel(t, img, image.Pt(404, 14), over(argb(0x66d4d4d8), bg), "the layout group, input at 40%, under the split segment")
	wantPixel(t, img, image.Pt(371, 6), bg, "the stacked segment, raised")
	wantPixel(t, img, image.Pt(437, 14), bg, "the gap after the group")
	wantPixel(t, img, image.Pt(442, 3), accent, "wrapping, pressed")
	wantPixel(t, img, image.Pt(474, 3), accent, "hiding whitespace, pressed")
	wantPixel(t, img, image.Pt(506, 3), bg, "the file tree toggle, off")
}

// The scope trigger is T3's xs secondary Button around its label and
// chevron: "Changes" measures 81 (§3.1).
func TestScopeTrigger(t *testing.T) {
	th := NewTheme()
	w := th.menuTrigger(measureCtx(), "", "Changes", BtnSecondary, false, 400, image.Point{})
	if w < 80 || w > 82 {
		t.Fatalf("the Changes trigger is %d wide, want 81", w)
	}
}

// The file tree pane starts at 40% of a 539 row (215.6), and drags
// between 160 and the row less 256 (§3.10).
func TestTreePaneWidth(t *testing.T) {
	th := NewTheme()
	gtx := measureCtx()
	for _, c := range []struct {
		dragged float32
		want    int
	}{{0, 216}, {1000, 283}, {100, 160}} {
		th.dv.treeW = c.dragged
		if got := th.treePanePx(gtx, 539); got != c.want {
			t.Errorf("dragged to %v: %d wide, want %d", c.dragged, got, c.want)
		}
	}
}

// The loading skeleton (§3.12): its separator's hairlines in border at
// 40% 12dp into its 24dp row, under a 32dp header whose bars are drawn.
func TestDiffSkeleton(t *testing.T) {
	th := NewTheme()
	bg := P.Background
	img := drawn(t, image.Pt(539, 400), 1, func(gtx C) D {
		fill(gtx, bg, image.Rect(0, 0, 539, 400), 0)
		th.diffSkeleton(gtx)
		return D{}
	})
	wantPixel(t, img, image.Pt(20, 44), over(twAlpha(P.Border, 40), bg), "separator hairline")
	if got := img.RGBAAt(18, 16); got.R >= bg.R-4 {
		t.Errorf("the header's first bar is #%02x%02x%02x, no darker than the surface", got.R, got.G, got.B)
	}
}

// The changed files card is 16dp under the message: a 40dp header, 8dp,
// a 28dp row per shown node, 8dp; folders start closed, and the header's
// toggle opens them all (§3.11).
func TestChangedFilesCard(t *testing.T) {
	th := NewTheme()
	c := &ChangedFiles{Run: "r", Count: 3, Nodes: diff.CardTree([]diff.FileStat{{Path: "a/x.go", Additions: 1}, {Path: "a/y.go", Additions: 2, Deletions: 1}, {Path: "b.go", Deletions: 4}})}
	gtx := measureCtx()
	gtx.Constraints.Max.X = 600
	if h := th.changedFilesCard(gtx, c).Size.Y; h != 16+40+8+2*28+8 {
		t.Errorf("closed: %d tall", h)
	}
	c.AllOpen = true
	if h := th.changedFilesCard(gtx, c).Size.Y; h != 16+40+8+4*28+8 {
		t.Errorf("open: %d tall", h)
	}
	c.Open = map[string]bool{"a": false}
	if h := th.changedFilesCard(gtx, c).Size.Y; h != 16+40+8+2*28+8 {
		t.Errorf("a folder closed on its own: %d tall", h)
	}
	// Files beyond those listed get a button under the rows.
	c.Count = 9
	if h := th.changedFilesCard(gtx, c).Size.Y; h != 16+40+8+2*28+8+24+8 {
		t.Errorf("with more files: %d tall", h)
	}
}

// In dark mode the card is input at 20% over the black background.
func TestChangedFilesCardDark(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	c := &ChangedFiles{Run: "r", Count: 1, Nodes: diff.CardTree([]diff.FileStat{{Path: "b.go", Additions: 1}})}
	img := drawn(t, image.Pt(600, 200), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 600, 200), 0)
		return th.changedFilesCard(gtx, c)
	})
	wantPixel(t, img, image.Pt(300, 8), P.Background, "the margin over the card")
	wantPixel(t, img, image.Pt(300, 20), rgb(0x060606), "the card, input at 20% over black")
}

// Pierre's file icons come from T3's tables: its own by name and
// extension first, then Pierre's names, suffixes and extensions.
func TestFileIcon(t *testing.T) {
	for path, want := range map[string][2]string{
		"web/package.json":    {"file-tree-builtin-npm", ""},
		"src/App.tsx":         {"file-tree-builtin-react", "react"},
		"src/util.ts":         {"file-tree-builtin-typescript", "typescript"},
		"README.md":           {"file-tree-builtin-markdown", "markdown"},
		".gitignore":          {"file-tree-builtin-git", "git"},
		"Main.java":           {"t3-file-icon-java", ""},
		"photo.heic":          {"file-tree-builtin-image", "image"},
		"notes.unknownformat": {"file-tree-builtin-default", "default"},
	} {
		name, token := fileIcon(path)
		if name != want[0] || token != want[1] {
			t.Errorf("%s: %s %q, want %s %q", path, name, token, want[0], want[1])
		}
		if _, ok := pierreIcons[name]; !ok {
			t.Errorf("%s: no sprite icon %s", path, name)
		}
	}
}
