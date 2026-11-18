package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// cardScene is a started thread in a Git repository: its machine, Open
// in, a script, the branch, a quick action and the change totals.
func cardScene() *Scene {
	s := headerScene()
	s.DetailsCard = true
	s.Card = &ThreadDetails{
		Run:     &RunContext{Machine: "server", Label: "workstation", Workspace: "folder", Locked: true},
		OpenIn:  &OpenIn{Editor: "vscode", Label: "Open in VS Code"},
		Scripts: &Scripts{Primary: "Test", Icon: "test", Any: true},
		Branch:  &BranchRow{Label: "feature/ui-polish"},
		Git:     &GitActions{Label: "Commit & push", Icon: "cloud-upload"},
		Changes: &Changes{Insertions: 2382, Deletions: 7, Base: "vs main"},
	}
	return s
}

// The card's height is its sections (padded 8 above, 10 below) of 32dp
// rows inside a 1dp border, with the Version Control rule in full density;
// as the room shrinks it drops to compact (no machine row, no Git actions
// row, no rule), then essential (no Open in either) (§4.3).
func TestCardDensity(t *testing.T) {
	th := NewTheme()
	s := cardScene()
	var ops op.Ops
	size := func(avail int) int {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Now: time.Now(),
			Constraints: layout.Constraints{Max: image.Pt(280, 2000)}}
		return th.details(gtx, s, avail).Size.Y
	}
	full := 1 + (8 + 3*32 + 10) + 1 + (8 + 3*32 + 10) + 1
	compact := 1 + (8 + 2*32 + 10) + (8 + 2*32 + 10) + 1
	essential := 1 + (8 + 32 + 10) + (8 + 2*32 + 10) + 1
	for _, c := range []struct{ avail, want int }{{1000, full}, {full, full}, {full - 1, compact}, {compact, compact}, {compact - 1, essential}} {
		if got := size(c.avail); got != c.want {
			t.Errorf("with %d to fit in: %d tall, want %d", c.avail, got, c.want)
		}
	}
	// A running action keeps its row even in compact density.
	s.Card.Git.Running = &GitProgress{Status: "Committing...", Elapsed: "3s"}
	if got := size(compact + 32); got != compact+32 {
		t.Errorf("compact with an action running: %d, want %d", got, compact+32)
	}
}

// A pull request row sits under the branch row, then the muted "Show N
// more", 32dp each. Expanded, the others join them, but the density still
// judges the card without them, as T3's measure leaves scrolling lists out
// (§5.4). The row's icon is its state's: violet for merged.
func TestPRRows(t *testing.T) {
	defer SetDark(false)
	th := NewTheme()
	s := cardScene()
	s.Card.PRs = &PullRequests{Rows: []PullRequestRow{{State: "merged", Label: "#4: Ship the parser", Tip: "PR #4 - Merged: Ship the parser"}}, Others: 2}
	var ops op.Ops
	size := func(avail int) int {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Now: time.Now(),
			Constraints: layout.Constraints{Max: image.Pt(280, 2000)}}
		return th.details(gtx, s, avail).Size.Y
	}
	full := 1 + (8 + 3*32 + 10) + 1 + (8 + 5*32 + 10) + 1
	if got := size(1000); got != full {
		t.Errorf("with a pull request: %d tall, want %d", got, full)
	}
	s.Card.PRs.Expanded = true
	s.Card.PRs.Rows = append(s.Card.PRs.Rows, PullRequestRow{State: "open", Label: "#5"}, PullRequestRow{State: "closed", Label: "#6"})
	if got := size(full); got != full+2*32 {
		t.Errorf("expanded, with room for it folded: %d tall, want full density's %d", got, full+2*32)
	}

	for _, d := range []bool{false, true} {
		SetDark(d)
		s.Card.PRs.Expanded, s.Card.PRs.Rows = false, s.Card.PRs.Rows[:1]
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
		// The icon's 16dp box, 11dp into the row under the branch row
		// (bottom 220): a grey icon has as much green as blue.
		violet := false
		for y := 220 + 8; y < 220+24; y++ {
			for x := 1157 + 11; x < 1157+27; x++ {
				if c := img.RGBAAt(x, y); int(c.B) > int(c.G)+20 {
					violet = true
				}
			}
		}
		if !violet {
			t.Errorf("dark %v: the merged pull request's icon isn't violet", d)
		}
	}
}

// With every row turned off, the card offers "Choose what to show",
// padded 8, which opens the items menu under itself.
func TestCardFallback(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := &Scene{Live: true, Title: "t", Header: &ChatHeader{}, DetailsCard: true, Card: &ThreadDetails{}}
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(280, 900)}}
	if h := th.details(gtx, s, 900).Size.Y; h != 1+8+32+8+1 {
		t.Fatalf("fallback card %d tall, want 50", h)
	}
	s.Card.Menu = &CardMenu{For: "items", Rows: []MenuRow{{Kind: MenuGroupLabel, Label: "Show in thread details"}, {Kind: MenuCheckRow, Label: "Changes"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	// The card is 50 tall from 64; the menu hangs from its row, 4 below.
	wantPixel(t, img, image.Pt(1148+1+8+200, 64+1+8+32+4+20), P.Popover, "the items menu under the row")
}

// Inline, the card is at the canvas's top right (1148, 64 at 1440 wide
// with the sidebar open): T3's glass as it paints over the canvas, inside
// a 1dp border at foreground 10%; and the rows' hover fills black 3.5%.
func TestCardDrawn(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		s := cardScene()
		l := newLive(image.Pt(1440, 900), nil)
		l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
		l.frame(0)
		img := l.pixels(t)
		fill := over(alpha(P.Popover, 0xd5), P.Background)
		wantPixel(t, img, image.Pt(1148+140, 64+4), fill, "card fill")
		wantPixel(t, img, image.Pt(1148, 64+60), over(alpha(P.ContrastForeground, 0x1a), fill), "card border")
		wantPixel(t, img, image.Pt(1148-4, 64+60), P.Background, "canvas beside the card")
		// The rule above Version Control: 1 + 8 + 96 + 10 down.
		wantPixel(t, img, image.Pt(1148+140, 64+1+114), over(alpha(P.Border, uint8(int(P.Border.A)*65/100)), fill), "section rule")
		// Hover the Changes row, the last: 1 + 114 + 1 + 8 + 64 down.
		changes := image.Pt(1148+1+8, 64+1+114+1+8+64)
		l.move(float32(changes.X+100), float32(changes.Y+16))
		l.frame(0)
		l.frame(0)
		img = l.pixels(t)
		wantPixel(t, img, image.Pt(changes.X+4, changes.Y+16), over(rowTint(), fill), "hovered row")
		wantPixel(t, img, image.Pt(changes.X+4, changes.Y-16), fill, "the row above, not hovered")
	}
}

// Hovering a select row fills it, stacks a second fill on the hovered
// half and shows the 1×16 hairline before its chevron (§3.6).
func TestSelectRowHalves(t *testing.T) {
	s := cardScene()
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	branch := image.Pt(1148+1+8, 64+1+114+1+8) // the branch row's top left
	l.move(float32(branch.X+250), float32(branch.Y+16))
	l.frame(0)
	l.frame(0)
	img := l.pixels(t)
	fill := over(alpha(P.Popover, 0xd5), P.Background)
	row := over(rowTint(), fill)
	wantPixel(t, img, image.Pt(branch.X+4, branch.Y+16), row, "the row's fill")
	wantPixel(t, img, image.Pt(branch.X+258, branch.Y+16), over(rowTint(), row), "the chevron half's second fill")
	wantPixel(t, img, image.Pt(branch.X+229, branch.Y+16), over(alpha(P.Border, uint8(int(P.Border.A)*65/100)), row), "the hairline")
}

// Without room inline (a 1000-wide window leaves a 744 canvas), the card
// is a popover: 8dp into a box 4dp in from the header's right, 8 below
// it, there only while open.
func TestCardPopover(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := cardScene()
	l := newLive(image.Pt(1000, 800), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	img := l.pixels(t)
	at := image.Pt(1000-4-8-280, 52+8)
	wantPixel(t, img, at.Add(image.Pt(140, 4)), P.Background, "closed: no card")
	s.Details = true
	l.frame(0)
	l.frame(200 * time.Millisecond)
	img = l.pixels(t)
	fill := over(alpha(P.Popover, 0xd5), P.Background)
	wantPixel(t, img, at.Add(image.Pt(140, 4)), fill, "open: the card")
	wantPixel(t, img, at.Add(image.Pt(140, -4)), P.Background, "8dp of the popover's padding above it")

	// A header narrower than the popover's 296 squeezes it to the header
	// less 8: in a 290-wide column the card runs from 12 to 278.
	s.SidebarHidden = true
	l = newLive(image.Pt(290, 800), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	img = l.pixels(t)
	wantPixel(t, img, image.Pt(12+3, 52+8+20), fill, "the squeezed card's left edge")
	wantPixel(t, img, image.Pt(12-3, 52+8+20), P.Background, "left of it")
	wantPixel(t, img, image.Pt(278-3, 52+8+20), fill, "its right edge")
	wantPixel(t, img, image.Pt(278+3, 52+8+20), P.Background, "right of it")
}

// The card's menus hang 4dp under their row as wide as it, here the Git
// actions menu past the card's bottom.
func TestCardMenu(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := cardScene()
	s.Card.Menu = &CardMenu{For: "git", Rows: []MenuRow{{Label: "Commit", Icon: "git-commit-horizontal"}, {Label: "Push", Icon: "cloud-upload"}, {Label: "Create PR", Icon: "provider:github"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	// The Git row's bottom: 64 + 1 + 114 + 1 + 8 + 64.
	top := 64 + 1 + 114 + 1 + 8 + 64 + 4
	cardBottom := 64 + 1 + 114 + 1 + 114 + 1
	wantPixel(t, img, image.Pt(1148+1+8+130, cardBottom+6), P.Popover, "the menu below the card")
	wantPixel(t, img, image.Pt(1148+1+8-3, cardBottom+6), P.Background, "nothing left of the row's edge")
	// Its surface runs inside the 1dp border, under the dark lip's line, for
	// its three 28dp rows padded 4.
	if first, last := run(img, 1148+1+8+130, top-8, top+200, P.Popover); first != top+2 || last != top+1+4+3*28+4-1 {
		t.Errorf("the menu's surface spans y %d..%d, want %d..%d", first, last, top+2, top+1+4+3*28+4-1)
	}
}

// The branch picker hangs 4dp under its row (whose bottom is at 220):
// inside its 1dp border, its search block over a rule 43dp down, then the
// refs 8dp further, each 28dp, the first highlighted at foreground 9%;
// with nothing to list, a 36dp "No refs found." block instead
// (BranchPicker.tsx, combobox.tsx).
func TestBranchPickerDrawn(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := cardScene()
	s.Card.Picker = &BranchPicker{Refs: []BranchRef{{Name: "main", Tag: "default"}, {Name: "feature/ui-polish", Tag: "current"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	// x is clear of the names and of the tags at the rows' ends.
	top, x := 220+4, 1157+173
	list := top + 1 + 43
	hi := over(alpha(P.Foreground, 0x17), P.Popover)
	// The popup's border is nearly the highlight's colour, so the scan
	// stops at the list's last row.
	if first, last := run(img, x, list, list+8+2*28, hi); first != list+8 || last != list+8+28-1 {
		t.Errorf("the highlighted ref spans y %d..%d, want %d..%d", first, last, list+8, list+8+28-1)
	}
	wantPixel(t, img, image.Pt(x, list+8+28+3), P.Popover, "the second ref, not highlighted")

	s.Card.Picker = &BranchPicker{}
	img = drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	if first, last := run(img, x, list, list+80, P.Popover); first != list || last != list+36-1 {
		t.Errorf("the empty block spans y %d..%d, want %d..%d", first, last, list, list+36-1)
	}
}

// A Git action on the default branch asks in T3's dialog: the backdrop
// over the window, the popup in the middle.
func TestConfirmDialogDrawn(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := cardScene()
	s.Confirm = &Confirm{Title: "Push to default ref?", Description: "This action will push local commits.", Continue: "Push to main"}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(720, 450), P.Popover, "the popup")
	fill := over(alpha(P.Popover, 0xd5), P.Background)
	wantPixel(t, img, image.Pt(1148+140, 64+4), over(alpha(rgb(0), 0xa3), fill), "the backdrop over the card")
}

// Branch names keep their end when they don't fit: T3's middle truncation
// splits off the last segment, or a word near the last ten characters.
func TestSplitMiddle(t *testing.T) {
	for _, c := range []struct{ in, head, tail string }{
		{"feature/ui-polish", "feature/", "ui-polish"},
		{"t3/clone-repo-to-other-machine", "t3/clone-repo-to-other-", "machine"},
		{"t3/investigate-release-build-times", "t3/investigate-release-build-", "times"},
		{"t3/compare-long-featurenames-ab", "t3/compare-long-", "featurenames-ab"},
		{"t3/abcdefghijklmnopqrstuvwxyz", "t3/abcdefghijklmnop", "qrstuvwxyz"},
		{"main", "main", ""},
	} {
		if head, tail := splitMiddle(c.in); head != c.head || tail != c.tail {
			t.Errorf("%s: %q + %q, want %q + %q", c.in, head, tail, c.head, c.tail)
		}
	}
	if got := groupDigits(2382); got != "2,382" {
		t.Errorf("2382 is %q", got)
	}
	if got := groupDigits(1234567); got != "1,234,567" {
		t.Errorf("1234567 is %q", got)
	}
}
