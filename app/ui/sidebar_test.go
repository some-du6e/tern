package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The sidebar's content (docs/ui/04-sidebar.md), at T3's measured places:
// at 1440x900 the list starts at y 100, its rows at x 8, 239 wide.

// rowsScene is a list with an open card (it has a branch, so a meta line),
// a hovered card without one, and the Settled shelf open on a hovered row.
func rowsScene() *Scene {
	icon := ProjectIcon{Text: "TN", Color: "amber"}
	return &Scene{Sidebar: Sidebar{HasProjects: true, Items: []SidebarItem{
		{Row: &ThreadRow{ID: "e/1", Title: "Open", Project: "tern", Icon: icon, Branch: "main", Active: true, Age: "2h"}},
		{Row: &ThreadRow{ID: "e/2", Title: "Hovered", Project: "tern", Icon: icon, Age: "5m", Hover: true}},
		{Shelf: &Shelf{ID: "settled", Label: "Settled", Count: 1, Open: true}},
		{Row: &ThreadRow{ID: "e/3", Slim: true, Shelf: "settled", Title: "Settled", Project: "tern", Icon: icon, Age: "3d", Hover: true}},
	}}}
}

// Cards are 78dp with a meta line and 60 without, in 2dp of padding and 1dp
// apart; slim rows are 36; a short list keeps its shelves at the bottom,
// above 8dp of padding. The open row and hovered rows wear T3's fills
// (dark, where they differ most from the sidebar).
func TestSidebarRows(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, rowsScene()) })
	cols := map[string]color.NRGBA{"sidebar": SP.Sidebar, "active": over(SP.SidebarRowActive, SP.Sidebar), "hover": over(SP.SidebarRowHover, SP.Sidebar)}
	for _, c := range []struct {
		y    int
		want string
	}{
		{101, "sidebar"}, {103, "active"}, {179, "active"}, {181, "sidebar"},
		{184, "sidebar"}, {186, "hover"}, {244, "hover"}, {246, "sidebar"},
		{811, "sidebar"}, {813, "hover"}, {847, "hover"}, {849, "sidebar"},
	} {
		wantPixel(t, img, image.Pt(20, c.y), cols[c.want], c.want)
	}
	// The hovered card's hover card: the glass, 4dp right of the row (which
	// ends at 247) and level with its top at 185.
	glass := over(alpha(P.Popover, 0xd5), P.Background)
	wantPixel(t, img, image.Pt(256, 189), glass, "the hover card")
	wantPixel(t, img, image.Pt(249, 187), SP.Sidebar, "the 4dp before it")
	wantPixel(t, img, image.Pt(260, 183), P.Background, "above it")
}

// textAt is txt drawn on its own on bg, in the CSS line box whose top left
// is p: its glyphs' box (a line's ascent and descent, which is all Gio
// sizes a line by) centred in the line height, as CSS's half-leading puts
// it.
func textAt(t *testing.T, th *Theme, txt Txt, s string, p image.Point, bg color.NRGBA) *image.RGBA {
	t.Helper()
	return drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		fill(gtx, bg, image.Rect(0, 0, 1440, 900), 0)
		gtx.Constraints = layout.Constraints{Max: image.Pt(400, 100)}
		m := op.Record(gtx.Ops)
		d := th.Text(gtx, txt, s)
		call := m.Stop()
		lineH := int(float32(txt.Size)*txt.LineH + 0.5)
		defer op.Offset(p.Add(image.Pt(0, (lineH-d.Size.Y)/2))).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return d
	})
}

// A row's text sits centred on its line, as a CSS flex row centres it, not
// pinned to the line's top: each of these is where the same text, drawn
// alone in its line box, is.
func TestRowTextCentred(t *testing.T) {
	th := NewTheme()
	active, hover := over(SP.SidebarRowActive, SP.Sidebar), over(SP.SidebarRowHover, SP.Sidebar)
	search := &Scene{Sidebar: Sidebar{HasProjects: true, Query: "pairing", Results: []ThreadRow{
		{ID: "e/1", Title: "Watch pairing flow", Project: "tern", Icon: ProjectIcon{Text: "TN", Color: "amber"}, Age: "41m"},
	}}}
	menu := &Scene{Menu: &ContextMenu{At: image.Pt(600, 300), Items: []MenuItem{
		{ID: "pin", Label: "Pin thread", Icon: "pin"},
		{ID: "settle", Label: "Settle thread", Icon: "circle-check"},
	}}}
	for _, c := range []struct {
		what string
		s    *Scene
		line image.Rectangle // the line the text is centred on
		at   image.Point     // where its line box starts
		txt  Txt
		text string
		bg   color.NRGBA
	}{
		// The open card's line 1 is 20dp at 102 + 8; its 12/16 project
		// name starts at x 40.
		{"the card's project", rowsScene(), image.Rect(40, 110, 70, 130), image.Pt(40, 112),
			Txt{Size: 12, LineH: 16.0 / 12, Color: SP.SecondaryLabel, MaxLines: 1}, "tern", active},
		// The Settled header is 779..811, its label 8dp in from x 10.
		{"the shelf's label", rowsScene(), image.Rect(18, 779, 60, 811), image.Pt(18, 787),
			Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: SP.SidebarMutedForeground, MaxLines: 1}, "Settled", SP.Sidebar},
		// The hovered slim row is 812..848, its title after the icon at 44.
		{"the slim row's title", rowsScene(), image.Rect(44, 812, 90, 848), image.Pt(44, 820),
			Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: SP.Foreground, MaxLines: 1}, "Settled", hover},
		// The highlighted result is 100..136.
		{"the result's title", search, image.Rect(44, 100, 90, 136), image.Pt(44, 108),
			Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: SP.Foreground, MaxLines: 1}, "Watch pairing flow", active},
		// The menu's second row is 333..361, its label after the 16dp icon.
		{"the menu row's label", menu, image.Rect(637, 333, 680, 361), image.Pt(637, 337),
			Txt{Size: 14, LineH: 20.0 / 14, Color: P.ContrastForeground, MaxLines: 1}, "Settle thread", P.Popover},
	} {
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, c.s) })
		ref := textAt(t, th, c.txt, c.text, c.at, c.bg)
		gt, gb, ok := inkRows(img, c.line, c.bg)
		wt, wb, _ := inkRows(ref, c.line, c.bg)
		if !ok || gt != wt || gb != wb {
			t.Errorf("%s: ink on rows %d..%d, want %d..%d", c.what, gt, gb, wt, wb)
		}
	}
}

// The empty list's message sits 24dp down, and 8dp under it its button:
// 26dp of 1dp border round its label, centred in the list's 8..247.
func TestEmptyState(t *testing.T) {
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{HasProjects: true, Empty: &SidebarEmpty{Text: "No threads in notes yet", Button: "Show all projects"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	if top, bottom, ok := inkRows(img, image.Rect(8, 100, 248, 147), SP.Sidebar); !ok || top < 124 || bottom >= 140 {
		t.Errorf("the message's ink is on rows %d..%d, want within 124..139", top, bottom)
	}
	border := over(SP.SidebarBorder, SP.Sidebar)
	left, right := -1, -1
	for x := 8; x < 248; x++ {
		if nearPixel(img.RGBAAt(x, 160), border) {
			if left < 0 {
				left = x
			}
			right = x
		}
	}
	if mid := (left + right) / 2; left < 8 || right > 247 || mid < 126 || mid > 128 {
		t.Fatalf("the button's sides are at x %d and %d, want them either side of 127", left, right)
	}
	mid := (left + right) / 2
	wantPixel(t, img, image.Pt(mid, 147), SP.Sidebar, "above the button")
	wantPixel(t, img, image.Pt(mid, 148), border, "the button's top")
	wantPixel(t, img, image.Pt(mid, 173), border, "the button's bottom")
	wantPixel(t, img, image.Pt(mid, 174), SP.Sidebar, "below the button")
	// The label is 10dp inside the border, give or take its glyphs' sides.
	if l, r, ok := inkCols(img, image.Rect(left+1, 149, right, 173), SP.Sidebar); !ok || l-left < 10 || l-left > 13 || right-r < 10 || right-r > 13 {
		t.Errorf("the label is at x %d..%d in a button at %d..%d", l, r, left, right)
	}
}

// The snooze menu hangs 4dp under its clock with its right edge on the
// clock's, as wide as its widest row needs (its detail 8 + 16dp past the
// label) and at least 160dp, each label at the start of its row and each
// wake time at the end.
func TestSnoozeMenu(t *testing.T) {
	defer SetDark(false)
	SetDark(true) // black under it, where its shadow doesn't show
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{Snooze: &SnoozeMenu{Row: "e/1", Choices: []MenuItem{
		{ID: "a", Label: "In 1 hour", Detail: "11:00 AM"},
		{ID: "b", Label: "Next week", Detail: "Mon 9:00 AM"},
	}}}}
	black := rgb(0)
	img := drawn(t, image.Pt(600, 300), 1, func(gtx C) D {
		fill(gtx, black, image.Rect(0, 0, 600, 300), 0)
		// The clock's box: 24 by 20 at 400, 50.
		defer op.Offset(image.Pt(400, 50)).Push(gtx.Ops).Pop()
		th.snoozeMenu(gtx, s, image.Pt(24, 20))
		return D{}
	})
	// Its 1dp border, at the middle of the first row (79..107).
	left, right, ok := inkCols(img, image.Rect(0, 93, 600, 94), black)
	if !ok || right != 423 || right-left+1 < 160 {
		t.Fatalf("the menu spans x %d..%d, want it to end at 423 and be at least 160 wide", left, right)
	}
	if top, _, _ := inkRows(img, image.Rect(left+20, 60, left+21, 120), black); top != 74 {
		t.Errorf("the menu's top is at %d, want 74", top)
	}
	for i, row := range []image.Rectangle{image.Rect(left+2, 83, right-1, 103), image.Rect(left+2, 111, right-1, 131)} {
		l, r, ok := inkCols(img, row, black)
		if !ok || l-left < 12 || l-left > 15 || right-r < 12 || right-r > 15 {
			t.Errorf("row %d's text spans x %d..%d in a menu at %d..%d", i, l, r, left, right)
		}
	}
	// The widest row, the second: its widest gap in ink is the one between
	// label and detail.
	gap, run := 0, 0
	for x := left + 14; x < right-13; x++ {
		if _, _, ok := inkRows(img, image.Rect(x, 111, x+1, 131), black); ok {
			run = 0
			continue
		}
		run++
		gap = max(gap, run)
	}
	if gap < 22 || gap > 27 {
		t.Errorf("the widest row's label and detail are %d apart, want 24 give or take their glyphs' sides", gap)
	}
}

// In light the open row is rung 1dp inside with the sidebar border at 70%.
func TestOpenRowRing(t *testing.T) {
	th := NewTheme()
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, rowsScene()) })
	active := over(SP.SidebarRowActive, SP.Sidebar)
	wantPixel(t, img, image.Pt(8, 140), over(alpha(SP.SidebarBorder, 0xb3), active), "ring")
	wantPixel(t, img, image.Pt(9, 140), active, "inside the ring")
}

// The footer's two buttons are T3's footer buttons: 32dp at x 8 and 44, at
// y 4 in the 44dp footer; one with its menu open wears the hover fill.
func TestFooterButtons(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	s := &Scene{Account: &Traits{Groups: []Trait{{Sel: -1, Choices: []TraitChoice{{Label: "Sign out"}}}}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	// The menu opens above, and its shadow falls on the button's top, so
	// look at its lower half.
	hover := over(SP.SidebarRowHover, SP.Sidebar)
	wantPixel(t, img, image.Pt(46, 884), hover, "the open account button")
	wantPixel(t, img, image.Pt(70, 888), hover, "toward its far corner")
	wantPixel(t, img, image.Pt(42, 884), SP.Sidebar, "the gap before it")
	wantPixel(t, img, image.Pt(10, 884), SP.Sidebar, "the machines button, closed")
}

// The row menu is T3's DOM menu: the popover in a 1dp border, no shadow,
// padded 4, 28dp rows; opened from the keyboard its first row is lit.
func TestContextMenu(t *testing.T) {
	th := NewTheme()
	s := &Scene{Menu: &ContextMenu{Keyboard: true, At: image.Pt(600, 300), Items: []MenuItem{
		{ID: "pin", Label: "Pin thread", Icon: "pin"},
		{ID: "settle", Label: "Settle thread", Icon: "circle-check"},
		{ID: "rename", Label: "Rename thread", Icon: "pencil", Sep: true},
	}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	// The rows start inside the 1dp border and 4dp of padding, at 605.
	wantPixel(t, img, image.Pt(610, 306), over(P.Accent, P.Popover), "the first row, lit")
	wantPixel(t, img, image.Pt(610, 336), P.Popover, "the second row")
	wantPixel(t, img, image.Pt(600, 340), over(alpha(P.Foreground, 0x1a), P.Popover), "the border")
	// 5 + 28 + 28 + 9 + 28 + 5: the menu ends at 403, with nothing under it.
	wantPixel(t, img, image.Pt(620, 400), P.Popover, "the menu's bottom padding")
	wantPixel(t, img, image.Pt(620, 406), P.Background, "below the menu: no shadow")
}

// Near the window's corner the menu keeps 4dp inside it.
func TestContextMenuStaysIn(t *testing.T) {
	th := NewTheme()
	s := &Scene{Menu: &ContextMenu{At: image.Pt(1430, 890), Items: []MenuItem{{ID: "pin", Label: "Pin thread", Icon: "pin"}}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(1426, 880), P.Popover, "the menu, moved in")
	wantPixel(t, img, image.Pt(1437, 880), P.Background, "the 4dp past it")
}

// A project's monogram sits on its colour at 14%; a quiet row's is grey at
// 40%.
func TestProjectMonogram(t *testing.T) {
	th := NewTheme()
	bg := rgb(0xffffff)
	ic := ProjectIcon{Text: "TN", Color: "amber"}
	img := drawn(t, image.Pt(40, 16), 1, func(gtx C) D {
		fill(gtx, bg, image.Rect(0, 0, 40, 16), 0)
		th.projectIcon(gtx, ic, 16, bg, 0)
		op.Offset(image.Pt(20, 0)).Add(gtx.Ops)
		return th.projectIcon(gtx, ic, 16, bg, 1)
	})
	tile := over(alpha(Tailwind["amber-600"], 0x24), bg)
	wantPixel(t, img, image.Pt(1, 8), tile, "the tile")
	wantPixel(t, img, image.Pt(21, 8), over(alpha(grey(tile), 0x66), bg), "the quiet tile")
}

// A branch keeps its end (T3's MiddleTruncate tests).
func TestSplitMiddle(t *testing.T) {
	for value, tail := range map[string]string{
		"feature/ui-polish":                  "ui-polish",
		"t3/clone-repo-to-other-machine":     "machine",
		"t3/investigate-release-build-times": "times",
		"t3/compare-long-featurenames-ab":    "featurenames-ab",
		"t3/abcdefghijklmnopqrstuvwxyz":      "qrstuvwxyz",
		"short":                              "",
	} {
		if _, got := splitMiddle(value); got != tail {
			t.Errorf("%q: tail %q, want %q", value, got, tail)
		}
	}
}

// sidebarFrames drives the sidebar through Gio's router, 256 by 800.
func sidebarFrames(th *Theme, s *Scene) (frame func() []Action, r *gioinput.Router) {
	r = new(gioinput.Router)
	var ops op.Ops
	return func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(256, 800)), Source: r.Source(), Now: time.Now()}
		th.sidebar(gtx, s)
		r.Frame(&ops)
		return th.Actions()
	}, r
}

// The list's keyboard (§5.12): Tab reaches the rows, the arrows move
// between them, Enter opens one, F2 renames it, Shift+F10 opens its menu,
// and Right opens a folded shelf.
func TestSidebarKeyboard(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{Items: []SidebarItem{
		{Row: &ThreadRow{ID: "e/1", Title: "One"}},
		{Row: &ThreadRow{ID: "e/2", Title: "Two"}},
		{Shelf: &Shelf{ID: "settled", Label: "Settled", Count: 3}},
	}}}
	frame, r := sidebarFrames(th, s)
	press := func(name key.Name, mods key.Modifiers) []Action {
		r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
		return append(frame(), frame()...)
	}
	want := func(acts []Action, id, kind, text string) {
		t.Helper()
		for _, a := range acts {
			if a.ID == id && a.Kind == kind && a.Text == text {
				return
			}
		}
		t.Fatalf("actions %+v, want %s %s %q", acts, kind, id, text)
	}
	frame()
	r.MoveFocus(key.FocusForward)
	frame()
	want(press(key.NameReturn, 0), "thread:e/1", "click", "")
	press(key.NameDownArrow, 0)
	want(press(key.NameReturn, 0), "thread:e/2", "click", "")
	want(press(key.NameF2, 0), "rename:e/2", "click", "")
	want(press(key.NameF10, key.ModShift), "thread:e/2", "context", "keyboard")
	press(key.NameEnd, 0)
	want(press(key.NameRightArrow, 0), "shelf:settled", "click", "")
	// Up from the shelf is the last row; the list doesn't wrap.
	press(key.NameUpArrow, 0)
	press(key.NameUpArrow, 0)
	press(key.NameUpArrow, 0)
	want(press(key.NameSpace, 0), "thread:e/1", "click", "")
}

// A click on a hovered card's Settle settles it, and doesn't also open it.
func TestSettleIsntARowClick(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{Items: []SidebarItem{
		{Row: &ThreadRow{ID: "e/1", Title: "One", Age: "5m", CanSettle: true}},
	}}}
	frame, r := sidebarFrames(th, s)
	frame()
	// The card's line 1 is at y 48 + 2 + 8, 20 tall; Settle ends 4dp into
	// its padding, at x 241.
	r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, 68), Source: pointer.Mouse})
	frame()
	frame()
	at := f32.Pt(225, 68)
	r.Queue(pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse})
	frame()
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	acts := frame()
	if len(acts) != 1 || acts[0].ID != "settle:e/1" {
		t.Fatalf("actions %+v, want just settle:e/1", acts)
	}
}
