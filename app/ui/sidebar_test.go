package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"gioui.org/f32"
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
