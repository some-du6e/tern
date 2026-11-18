package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"runtime"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gpu/headless"
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
	filter := &Scene{Sidebar: Sidebar{HasProjects: true, Filter: &ProjectFilter{Items: []FilterItem{{Label: "All projects", Selected: true}}}}}
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
		// The search field is 60..92; its input's line, and the text in it,
		// centre in it.
		{"the search's placeholder", rowsScene(), image.Rect(40, 60, 90, 92), image.Pt(40, 60),
			Txt{Size: 14, LineH: 32.0 / 14, Font: UI(font.Medium), Color: SP.SidebarMutedForeground, MaxLines: 1}, "Search", SP.Sidebar},
		// The filter's popup is at 191, 94: its 26dp input is 1 + 10 - 1
		// down, 1 + 12 + 20 in.
		{"the filter search's placeholder", filter, image.Rect(224, 104, 300, 130), image.Pt(224, 104),
			Txt{Size: 14, LineH: 26.0 / 14, Color: P.MutedForeground, MaxLines: 1}, "Search projects...", P.Popover},
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

// A result found by its messages is 44dp, the matching message on a 16dp
// line under the title (§2.6, §3.11): "You:" in the info foreground or
// "Agent:" in the success foreground, a space, then the snippet 12/16 in
// the muted foreground at 85% with the query's words 600 in the
// foreground, its line breaks set as one space; cut with "…" where it
// would pass the content's 193dp. The icon centres on the row.
func TestSearchExcerpt(t *testing.T) {
	th := NewTheme()
	icon := ProjectIcon{Text: "TN", Color: "amber"}
	s := &Scene{Sidebar: Sidebar{HasProjects: true, Query: "parser", Results: []ThreadRow{
		{ID: "e/1", Title: "Rewrite", Project: "tern", Icon: icon, Age: "2h",
			Excerpt: &Excerpt{User: true, Snippet: "the\n\n  parser is slow", Marks: [][2]int{{7, 13}}}},
		{ID: "e/2", Title: "Docs", Project: "tern", Icon: icon, Age: "3d",
			Excerpt: &Excerpt{Snippet: strings.Repeat("m", 80)}},
	}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	active := over(SP.SidebarRowActive, SP.Sidebar)
	// The highlighted first result is 100..144, the second 145..189.
	wantPixel(t, img, image.Pt(240, 101), active, "the first result's top")
	wantPixel(t, img, image.Pt(240, 143), active, "its bottom")
	wantPixel(t, img, image.Pt(240, 144), SP.Sidebar, "the 1dp under it")
	if top, bottom, ok := inkRows(img, image.Rect(18, 100, 34, 144), active); !ok || top != 114 || bottom != 129 {
		t.Errorf("the icon is on rows %d..%d, want 114..129", top, bottom)
	}

	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	width := func(txt Txt, s string) int { return th.textWidth(gtx, txt, s) }
	// same checks text is drawn as txt alone would be in its line box at.
	same := func(what string, txt Txt, text string, at image.Point, bg color.NRGBA) {
		t.Helper()
		ref := textAt(t, th, txt, text, at, bg)
		h := int(float32(txt.Size)*txt.LineH + 0.5)
		for y := at.Y; y < at.Y+h; y++ {
			for x := at.X; x < at.X+width(txt, text); x++ {
				if got, want := img.RGBAAt(x, y), ref.RGBAAt(x, y); !nearInk(got, want) {
					t.Errorf("%s: %v at %d,%d, want %v", what, got, x, y, want)
					return
				}
			}
		}
	}
	tone := func(bg color.NRGBA) (who, muted, mark Txt) {
		muted = Txt{Size: 12, LineH: 16.0 / 12, Color: over(faded(SP.MutedForeground, 0.85), bg), MaxLines: 1}
		who, mark = muted, muted
		mark.Font, mark.Color = UI(font.SemiBold), over(SP.Foreground, bg)
		return who, muted, mark
	}
	same("the title", Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: over(SP.Foreground, active), MaxLines: 1}, "Rewrite", image.Pt(44, 104), active)
	you, muted, mark := tone(active)
	you.Color = over(SP.InfoForeground, active)
	x := 44 + width(you, "You:")
	same("You:", you, "You:", image.Pt(44, 124), active)
	same("the words before the mark", muted, " the ", image.Pt(x, 124), active)
	x += width(muted, " the ")
	same("the mark", mark, "parser", image.Pt(x, 124), active)
	same("the words after it", muted, " is slow", image.Pt(x+width(mark, "parser"), 124), active)

	agent, muted, _ := tone(SP.Sidebar)
	agent.Color = over(SP.SuccessForeground, SP.Sidebar)
	same("Agent:", agent, "Agent:", image.Pt(44, 169), SP.Sidebar)
	k := 80
	for 44+width(agent, "Agent:")+width(muted, " "+strings.Repeat("m", k))+width(muted, "…") > 44+193 {
		k--
	}
	same("the cut", muted, "…", image.Pt(44+width(agent, "Agent:")+width(muted, " "+strings.Repeat("m", k)), 169), SP.Sidebar)
}

// A result whose pull request the query matched shows its number before
// the age, 10dp from it: the state's 12dp glyph and the number, 12/16,
// 2dp apart, in the state's tone, centred on the title's line
// (ThreadSearchPullRequestNumber).
func TestSearchPullRequestNumber(t *testing.T) {
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{HasProjects: true, Query: "#42", Results: []ThreadRow{
		{ID: "e/1", Title: "Ship it", Age: "2h", PR: &PRBadge{State: "open", Text: "42"}},
	}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	active := over(SP.SidebarRowActive, SP.Sidebar)
	tone := over(Tailwind["emerald-600"], active)
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	num := Txt{Size: 12, LineH: 16.0 / 12, Color: tone, MaxLines: 1}
	// The age is 205..237, so the number ends at 195; the title's line is
	// 108..128, the number's 16dp line 110..126 and the glyph 112..124.
	x := 195 - th.textWidth(gtx, num, "42")
	ref := textAt(t, th, num, "42", image.Pt(x, 110), active)
	glyph := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		fill(gtx, active, image.Rect(0, 0, 1440, 900), 0)
		defer op.Offset(image.Pt(x-14, 112)).Push(gtx.Ops).Pop()
		return LucideW("git-pull-request-arrow", tone, 12)(gtx)
	})
	for _, c := range []struct {
		what string
		r    image.Rectangle
		want *image.RGBA
	}{
		{"the number", image.Rect(x, 110, 195, 126), ref},
		{"its glyph", image.Rect(x-14, 112, x-2, 124), glyph},
	} {
		if _, _, ok := inkRows(img, c.r, active); !ok {
			t.Errorf("%s: nothing drawn in %v", c.what, c.r)
		}
	check:
		for y := c.r.Min.Y; y < c.r.Max.Y; y++ {
			for x := c.r.Min.X; x < c.r.Max.X; x++ {
				if got, want := img.RGBAAt(x, y), c.want.RGBAAt(x, y); !nearInk(got, want) {
					t.Errorf("%s: %v at %d,%d, want %v", c.what, got, x, y, want)
					break check
				}
			}
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
// wake time at the end; it keeps 5dp inside the window.
func TestSnoozeMenu(t *testing.T) {
	defer SetDark(false)
	SetDark(true) // black under it, where its shadow doesn't show
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{Snooze: &SnoozeMenu{Row: "e/1", Choices: []MenuItem{
		{ID: "a", Label: "In 1 hour", Detail: "11:00 AM"},
		{ID: "b", Label: "Next week", Detail: "Mon 9:00 AM"},
	}}}}
	black := rgb(0)
	// The clock's box is 24 by 20 at x.
	at := func(x int) *image.RGBA {
		return drawn(t, image.Pt(600, 300), 1, func(gtx C) D {
			fill(gtx, black, image.Rect(0, 0, 600, 300), 0)
			defer op.Offset(image.Pt(x, 50)).Push(gtx.Ops).Pop()
			th.snoozeMenu(gtx, s, image.Pt(24, 20), image.Rect(x, 50, x+24, 70))
			return D{}
		})
	}
	// Near the window's left edge it shifts along to stay 5dp inside.
	if left, _, _ := inkCols(at(20), image.Rect(0, 93, 600, 94), black); left != 5 {
		t.Errorf("by the window's edge the menu starts at x %d, want 5", left)
	}
	img := at(400)
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

// A newly opened thread's row scrolls into view, as near as it can stay
// (block: nearest), and a new project scope starts the list at its top
// (routeReveal.ts, §5.8).
func TestListRevealsOpenRow(t *testing.T) {
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{HasProjects: true}}
	for i := range 20 {
		r := &ThreadRow{ID: fmt.Sprint("e/", i), Title: fmt.Sprint("Thread ", i), Age: "1h"}
		r.Draft, r.Active = i == 0, i == 14
		s.Sidebar.Items = append(s.Sidebar.Items, SidebarItem{Row: r})
	}
	active := over(SP.SidebarRowActive, SP.Sidebar)
	draft := over(alpha(SP.Warning, 0x0a), SP.Sidebar)
	// Unopened, the list is at its top: the first card, a draft, at 102.
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(20, 110), draft, "the first card, at the top")
	// Cards are 64 apart plus the 1dp gap, so the 15th ends at 974 in the
	// list, 218 past its 756: scrolled that far, it sits at the bottom,
	// 792..856 in the window, its surface 2dp in. (The list's last 12dp
	// fade, as there's more below.)
	s.Sidebar.Open = "e/14"
	img = drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(20, 793), SP.Sidebar, "above the open card")
	wantPixel(t, img, image.Pt(20, 796), active, "the open card, scrolled in")
	wantPixel(t, img, image.Pt(200, 840), active, "its lower half")
	// A new scope starts from the top again.
	s.Sidebar.ScopeKey = "e/project"
	img = drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(20, 110), draft, "the first card, back at the top")
}

// A search result scrolls into view as it lights, nearest edge first
// (Sidebar.tsx's highlighted option); the first lights at the top.
func TestSearchRevealsHighlight(t *testing.T) {
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{HasProjects: true, Query: "thread"}}
	for i := range 30 {
		s.Sidebar.Results = append(s.Sidebar.Results, ThreadRow{ID: fmt.Sprint("e/", i), Title: fmt.Sprint("Thread ", i), Age: "1h"})
	}
	active := over(SP.SidebarRowActive, SP.Sidebar)
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(20, 101), active, "the first result")
	// Results are 37 apart, so the 26th ends at 961 in the list, 205 past
	// its 756: it sits at the bottom, 820..856 in the window.
	s.Sidebar.Highlight = 25
	img = drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	wantPixel(t, img, image.Pt(20, 818), SP.Sidebar, "above the lit result")
	wantPixel(t, img, image.Pt(20, 822), active, "the lit result, scrolled in")
	wantPixel(t, img, image.Pt(240, 840), active, "its lower half")
}

// The new thread's draft sits atop the list, 78dp in 2dp of padding on
// warning at 4%; under it, 6dp clear, the sidebar border at 60% rules the
// list 10dp in from the row's sides (§3.12).
func TestDraftRow(t *testing.T) {
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{HasProjects: true, Items: []SidebarItem{
		{Draft: &DraftRow{ID: "draft", Project: "tern", Icon: ProjectIcon{Text: "TN", Color: "amber"}, Preview: "Plan the release"}},
		{Divider: true},
		{Row: &ThreadRow{ID: "e/1", Title: "One", Age: "1h"}},
	}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	tint := over(alpha(SP.Warning, 0x0a), SP.Sidebar)
	wantPixel(t, img, image.Pt(20, 101), SP.Sidebar, "the draft's padding")
	wantPixel(t, img, image.Pt(20, 103), tint, "the draft")
	wantPixel(t, img, image.Pt(20, 179), tint, "its bottom")
	wantPixel(t, img, image.Pt(20, 181), SP.Sidebar, "below it")
	// The draft's item is 100..182; then the list's 1dp gap and 6 clear.
	rule := over(faded(SP.SidebarBorder, 0.6), SP.Sidebar)
	wantPixel(t, img, image.Pt(20, 188), SP.Sidebar, "above the rule")
	wantPixel(t, img, image.Pt(20, 189), rule, "the rule")
	wantPixel(t, img, image.Pt(17, 189), SP.Sidebar, "the rule's inset")
}

// The footer's undo notice takes what its buttons leave, 80..247, its text
// right-aligned: the summary, then "<shortcut> to undo", on a 24dp line the
// footer's row centres at 864.
func TestUndoNotice(t *testing.T) {
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{Undo: "Settled"}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	l, r, ok := inkCols(img, image.Rect(80, 856, 248, 900), SP.Sidebar)
	if !ok || l < 90 || r < 244 || r > 247 {
		t.Fatalf("the notice's text spans x %d..%d, want it to end at the footer's padding", l, r)
	}
	if top, bottom, _ := inkRows(img, image.Rect(80, 856, 248, 900), SP.Sidebar); top < 866 || bottom > 886 {
		t.Errorf("the notice's text is on rows %d..%d, want it within its line at 864..888", top, bottom)
	}
}

// liveWindow draws frames through Gio's router at the times a test picks,
// so the pointer's hover and rest show as they would in the app.
type liveWindow struct {
	win  *headless.Window
	r    gioinput.Router
	ops  op.Ops
	size image.Point
}

func newLiveWindow(t *testing.T, size image.Point) *liveWindow {
	t.Helper()
	win, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		if runtime.GOOS == "linux" {
			t.Fatalf("headless GPU: %v", err)
		}
		t.Skipf("no headless GPU here: %v", err)
	}
	t.Cleanup(win.Release)
	return &liveWindow{win: win, size: size}
}

func (l *liveWindow) frame(t *testing.T, now time.Time, w layout.Widget) *image.RGBA {
	t.Helper()
	l.ops.Reset()
	w(layout.Context{Ops: &l.ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(l.size), Source: l.r.Source(), Now: now})
	l.r.Frame(&l.ops)
	if err := l.win.Frame(&l.ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: l.size})
	if err := l.win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	return img
}

// The hover card opens once the pointer has rested 150ms on its row,
// growing and fading in over 150ms, and fades out over 150ms as the
// pointer leaves (§3.15, §6).
func TestHoverCardMotion(t *testing.T) {
	defer SetDark(false)
	SetDark(true) // the card's border shows against the black sidebar
	th := NewTheme()
	s := rowsScene()
	s.Live = true
	for _, it := range s.Sidebar.Items {
		if it.Row != nil {
			it.Row.Hover = false
		}
	}
	lw := newLiveWindow(t, image.Pt(1440, 900))
	t0 := time.Now()
	at := func(d time.Duration) *image.RGBA {
		return lw.frame(t, t0.Add(d), func(gtx C) D { return th.Desktop(gtx, s) })
	}
	glass := over(alpha(P.Popover, 0xd5), P.Background)
	border := over(alpha(P.Foreground, 0x1a), glass)
	// The second card is 185..245; its card's left border is at 251.
	edge := image.Pt(251, 200)
	at(0)
	lw.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, 215), Source: pointer.Mouse})
	at(10 * time.Millisecond) // the pointer comes to rest
	wantPixel(t, at(100*time.Millisecond), edge, SP.Sidebar, "before the rest is up")
	at(160 * time.Millisecond) // the frame the card asked for, as it opens
	if c := at(235*time.Millisecond).RGBAAt(edge.X, edge.Y); c.R < 4 || c.R+4 > border.R {
		t.Errorf("half way in the border is #%02x%02x%02x, want between black and #%02x%02x%02x", c.R, c.G, c.B, border.R, border.G, border.B)
	}
	wantPixel(t, at(400*time.Millisecond), edge, border, "the card, open")
	lw.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(700, 400), Source: pointer.Mouse})
	at(500 * time.Millisecond) // the pointer leaves
	if c := at(560*time.Millisecond).RGBAAt(edge.X, edge.Y); c.R < 4 || c.R+4 > border.R {
		t.Errorf("half way out the border is #%02x%02x%02x, want between black and #%02x%02x%02x", c.R, c.G, c.B, border.R, border.G, border.B)
	}
	wantPixel(t, at(700*time.Millisecond), edge, SP.Sidebar, "the card, closed")
}

// Holding mod on its own for 200ms puts each of the first rows' jump keys,
// T3's Kbd, in its slot; letting go takes them away (§4, §5.12), and so
// does the window losing the keyboard, where mod is let go of somewhere
// this window never hears (shortcutModifierState.ts resets on blur).
func TestJumpHints(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{Items: []SidebarItem{{Row: &ThreadRow{ID: "e/1", Title: "One", Jump: 1}}}}}
	lw := newLiveWindow(t, image.Pt(1440, 900))
	t0 := time.Now()
	at := func(d time.Duration) *image.RGBA {
		return lw.frame(t, t0.Add(d), func(gtx C) D { return th.Desktop(gtx, s) })
	}
	mod := key.NameCtrl
	if key.ModShortcut == key.ModCommand {
		mod = key.NameCommand
	}
	// The card's first line is 110..130 and its slot ends at 237, which
	// the Kbd's right padding reaches.
	kbd := over(alpha(P.Foreground, 0x12), SP.Sidebar)
	probe := image.Pt(235, 120)
	at(0)
	lw.r.Queue(key.Event{Name: mod, State: key.Press, Modifiers: key.ModShortcut})
	at(10 * time.Millisecond)
	wantPixel(t, at(100*time.Millisecond), probe, SP.Sidebar, "before the hold is up")
	wantPixel(t, at(250*time.Millisecond), probe, kbd, "the jump key, held")
	lw.r.Queue(key.Event{Name: mod, State: key.Release})
	wantPixel(t, at(300*time.Millisecond), probe, SP.Sidebar, "let go")
	lw.r.Queue(key.Event{Name: mod, State: key.Press, Modifiers: key.ModShortcut})
	at(400 * time.Millisecond)
	wantPixel(t, at(650*time.Millisecond), probe, kbd, "the jump key, held again")
	th.WindowBlurred()
	wantPixel(t, at(700*time.Millisecond), probe, SP.Sidebar, "the window lost the keyboard")
}

// mod+Shift+[ and ] open the thread before and after (thread.previous,
// thread.next, §5.12). Gio names the keys by what Shift makes of them on a
// Mac and on Linux, "{" and "}", and by the keys themselves on Windows.
func TestThreadStepKeys(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true}
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(1440, 900)), Source: r.Source(), Now: time.Now()}
		th.Desktop(gtx, s)
		r.Frame(&ops)
		return th.Actions()
	}
	frame()
	for _, c := range []struct {
		name key.Name
		want string
	}{{"{", "thread.prev"}, {"}", "thread.next"}, {"[", "thread.prev"}, {"]", "thread.next"}} {
		r.Queue(key.Event{Name: c.name, Modifiers: key.ModShortcut | key.ModShift, State: key.Press})
		if acts := frame(); len(acts) != 1 || acts[0].ID != c.want {
			t.Errorf("mod+Shift+%s: actions %+v, want just %s", c.name, acts, c.want)
		}
	}
}

// The shell's shortcuts cost a frame nothing: shellKeys asks for its keys
// without allocating, so a frame that only animates doesn't for them.
func TestShellKeysDontAllocate(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true}
	var r gioinput.Router
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 600)), Source: r.Source()}
	frame := func(keys bool) func() {
		return func() {
			ops.Reset()
			if keys {
				th.shellKeys(gtx, s)
			}
			r.Frame(&ops)
		}
	}
	frame(true)()
	if n, base := testing.AllocsPerRun(50, frame(true)), testing.AllocsPerRun(50, frame(false)); n > base {
		t.Errorf("a frame allocates %v times with the shell's keys, %v without", n, base)
	}
}

// A click on a card's pull request is the badge's: it opens the pull
// request, not the thread.
func TestPullRequestIsntARowClick(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{Items: []SidebarItem{
		{Row: &ThreadRow{ID: "e/1", Title: "One", Age: "5m", Branch: "main", PR: &PRBadge{State: "open", Text: "42"}}},
	}}}
	frame, r := sidebarFrames(th, s)
	frame()
	// The meta line is 16dp at 48 + 2 + 54; the badge ends 6dp short of
	// the line's end at 237, where the empty trailing icons sit.
	at := f32.Pt(225, 112)
	r.Queue(pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse})
	frame()
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	if acts := frame(); len(acts) != 1 || acts[0].ID != "pr:e/1" {
		t.Fatalf("actions %+v, want just pr:e/1", acts)
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
		th.projectIcon(gtx, ic, 16, rowInk{bg: bg, k: 1}, 0)
		op.Offset(image.Pt(20, 0)).Add(gtx.Ops)
		return th.projectIcon(gtx, ic, 16, rowInk{bg: bg, k: 1}, 1)
	})
	tile := over(alpha(Tailwind["amber-600"], 0x24), bg)
	wantPixel(t, img, image.Pt(1, 8), tile, "the tile")
	wantPixel(t, img, image.Pt(21, 8), over(alpha(grey(tile), 0x66), bg), "the quiet tile")
}

// A project's lucide icon draws in its colour; one tern doesn't have is
// folder-code, as T3 draws one it can't find.
func TestProjectLucideIcon(t *testing.T) {
	th := NewTheme()
	bg := rgb(0xffffff)
	icon := func(name string) *image.RGBA {
		return drawn(t, image.Pt(16, 16), 1, func(gtx C) D {
			fill(gtx, bg, image.Rect(0, 0, 16, 16), 0)
			return th.projectIcon(gtx, ProjectIcon{Lucide: name, Color: "gray"}, 16, rowInk{bg: bg, k: 1}, 0)
		})
	}
	got, want := icon("not-an-icon"), icon("folder-code")
	if _, _, ok := inkRows(got, got.Bounds(), bg); !ok || !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("an unknown lucide icon should draw as folder-code")
	}
}

// sameInk fails unless got and want are drawn the same in r, give or take
// the glyphs' antialiasing.
func sameInk(t *testing.T, got, want *image.RGBA, r image.Rectangle, what string) {
	t.Helper()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if g, w := got.RGBAAt(x, y), want.RGBAAt(x, y); !nearInk(g, w) {
				t.Errorf("%s: %v at %d,%d, want %v", what, g, x, y, w)
				return
			}
		}
	}
}

// A branch keeps its end (MiddleTruncate, 02 §8.17): where it doesn't fit,
// its head stops after the last whole glyph that leaves room, and "…" sits
// flush before the tail, which is never cut. The tails are T3's own (its
// middle-truncate tests); a value that fits, or has nothing worth cutting,
// is drawn whole.
func TestMiddleTruncate(t *testing.T) {
	th := NewTheme()
	txt := Txt{Size: 12, LineH: 16.0 / 12, Color: SP.SecondaryLabel, MaxLines: 1}
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	width := func(s string) int { return th.textWidth(gtx, txt, s) }
	type cut struct {
		value string
		w     int
		want  string
	}
	cases := []cut{{"feature/ui-polish", 400, "feature/ui-polish"}, {"short", 400, "short"}}
	for value, tail := range map[string]string{
		"feature/ui-polish":                  "ui-polish",
		"t3/clone-repo-to-other-machine":     "machine",
		"t3/investigate-release-build-times": "times",
		"t3/compare-long-featurenames-ab":    "featurenames-ab",
		"t3/abcdefghijklmnopqrstuvwxyz":      "qrstuvwxyz",
	} {
		// Room for the head's first four glyphs and a pixel: they show, and
		// then "…" and the tail.
		head := value[:4]
		cases = append(cases, cut{value, width(head) + width("…"+tail) + 1, head + "…" + tail})
	}
	at := func(w int, draw func(gtx C) D) *image.RGBA {
		return drawn(t, image.Pt(420, 36), 1, func(gtx C) D {
			fill(gtx, SP.Sidebar, image.Rect(0, 0, 420, 36), 0)
			defer op.Offset(image.Pt(10, 10)).Push(gtx.Ops).Pop()
			gtx.Constraints = layout.Exact(image.Pt(w, 16))
			return draw(gtx)
		})
	}
	for _, c := range cases {
		got := at(c.w, func(gtx C) D { return th.middleTruncate(gtx, txt, c.value) })
		want := at(400, func(gtx C) D { return th.Text(gtx, txt, c.want) })
		sameInk(t, got, want, got.Bounds(), fmt.Sprintf("%q in %dpx", c.value, c.w))
	}
}

// The hover card's branch is in the lines' muted foreground, not their 75%
// one (T3's MiddleTruncate there has no colour of its own), and it's cut to
// the card's 320dp as the meta line's is: the head's whole glyphs, "…" and
// the tail.
func TestHoverCardBranch(t *testing.T) {
	th := NewTheme()
	branch := strings.Repeat("investigate-", 4) + "times"
	s := &Scene{Sidebar: Sidebar{Items: []SidebarItem{{Row: &ThreadRow{ID: "e/1", Title: "T", Branch: branch, Age: "1h", Hover: true}}}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	glass := over(alpha(P.Popover, 0xd5), P.Background)
	txt := Txt{Size: 12, LineH: 16.0 / 12, Color: over(P.MutedForeground, glass), MaxLines: 1}
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(400, 100)}}
	width := func(s string) int { return th.textWidth(gtx, txt, s) }
	// The tail starts on the last word, after its "-"
	// (splitForMiddleTruncate's shorter word tail).
	head, tail := strings.TrimSuffix(branch, "times"), "times"
	n := len(head)
	for width(head[:n])+width("…"+tail) > 272 {
		n--
	}
	// The card is at 251, 102: its 13dp of padding and border, the title's
	// line and 8dp more put the branch under it; the 2dp indent, the 12dp
	// icon and its 8dp gap put it at 286, 272 wide.
	y := 102 + 13 + th.Text(gtx, Txt{Size: 12, LineH: 15.0 / 12, Font: UI(font.Medium), MaxLines: 1}, "T").Size.Y + 8
	ref := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		fill(gtx, glass, image.Rect(0, 0, 1440, 900), 0)
		defer op.Offset(image.Pt(286, y)).Push(gtx.Ops).Pop()
		gtx.Constraints = layout.Constraints{Max: image.Pt(400, 100)}
		return th.Text(gtx, txt, head[:n]+"…"+tail)
	})
	sameInk(t, img, ref, image.Rect(286, y, 286+272, y+16), "the card's branch")
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

// Moving from one row's open card to another's, up the list or down it,
// swaps the cards in the first frame: the new one opens at once and the
// old one goes at once, whichever row is drawn first (§3.15, the list's
// tooltip group).
func TestHoverCardHandoff(t *testing.T) {
	defer SetDark(false)
	SetDark(true) // the card's border shows against the black sidebar
	th := NewTheme()
	s := rowsScene()
	s.Live = true
	for _, it := range s.Sidebar.Items {
		if it.Row != nil {
			it.Row.Hover = false
		}
	}
	lw := newLiveWindow(t, image.Pt(1440, 900))
	t0 := time.Now()
	at := func(d time.Duration) *image.RGBA {
		return lw.frame(t, t0.Add(d), func(gtx C) D { return th.Desktop(gtx, s) })
	}
	move := func(y float32) {
		lw.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, y), Source: pointer.Mouse})
	}
	glass := over(alpha(P.Popover, 0xd5), P.Background)
	border := over(alpha(P.Foreground, 0x1a), glass)
	// The first card's card is 102..189 at x 251, the second's from 185.
	first, second := image.Pt(251, 140), image.Pt(251, 200)
	at(0)
	move(215)
	at(10 * time.Millisecond)
	at(160 * time.Millisecond)
	wantPixel(t, at(400*time.Millisecond), second, border, "the second card's card")
	move(140)
	img := at(500 * time.Millisecond)
	wantPixel(t, img, first, border, "up onto the first: its card, at once")
	wantPixel(t, img, second, SP.Sidebar, "up onto the first: the second's, gone at once")
	move(215)
	img = at(600 * time.Millisecond)
	wantPixel(t, img, second, border, "back down: the second's card, at once")
	wantPixel(t, img, first, SP.Sidebar, "back down: the first's, gone at once")
}

// The list's top fade is there in the very frame a wheel scrolls it from
// its top: the fade, like the rows, is drawn where the list is once this
// frame's scrolling is in, not where it was (§3.3, port note 4).
func TestListFadeFollowsScroll(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{HasProjects: true}}
	for i := range 20 {
		s.Sidebar.Items = append(s.Sidebar.Items, SidebarItem{Row: &ThreadRow{ID: fmt.Sprint("e/", i), Title: fmt.Sprint("Thread ", i), Age: "1h", Draft: true}})
	}
	lw := newLiveWindow(t, image.Pt(1440, 900))
	frame := func() *image.RGBA { return lw.frame(t, time.Now(), func(gtx C) D { return th.Desktop(gtx, s) }) }
	frame()
	at := f32.Pt(100, 400)
	lw.r.Queue(pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse})
	frame()
	lw.r.Queue(pointer.Event{Kind: pointer.Scroll, Position: at, Scroll: f32.Pt(0, 30), Source: pointer.Mouse})
	img := frame()
	draft := over(alpha(SP.Warning, 0x0a), SP.Sidebar)
	// Cards are 64 apart plus the 1dp gap: scrolled 30, the first one's
	// surface is at 72..132 and the second's from 137.
	wantPixel(t, img, image.Pt(20, 128), draft, "the first card, scrolled")
	wantPixel(t, img, image.Pt(20, 134), SP.Sidebar, "the gap after it")
	wantPixel(t, img, image.Pt(20, 100), SP.Sidebar, "the top fade over it")
}

// The keyboard's End scrolls the last row into view between frames, not
// while the list lays its rows out: the frame that takes the key draws the
// list as it was, whole, and the next as it's going to be, never the old
// top rows over rows from the new place (§5.12, block nearest).
func TestKeyboardReveal(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true}
	for i := range 20 {
		r := &ThreadRow{ID: fmt.Sprint("e/", i), Title: fmt.Sprint("Thread ", i), Age: "1h"}
		r.Active, r.Draft = i == 0, i > 0
		s.Sidebar.Items = append(s.Sidebar.Items, SidebarItem{Row: r})
	}
	lw := newLiveWindow(t, image.Pt(256, 800))
	frame := func() *image.RGBA {
		return lw.frame(t, time.Now(), func(gtx C) D {
			fill(gtx, SP.Sidebar, image.Rectangle{Max: gtx.Constraints.Max}, 0)
			th.sidebar(gtx, s)
			return D{Size: gtx.Constraints.Max}
		})
	}
	frame()
	lw.r.MoveFocus(key.FocusForward)
	frame()
	lw.r.Queue(key.Event{Name: key.NameEnd, State: key.Press})
	took := frame()
	next := frame()
	active, draft := over(SP.SidebarRowActive, SP.Sidebar), over(alpha(SP.Warning, 0x0a), SP.Sidebar)
	// The list starts at 48. As it was, the open card's surface is 50..110
	// and the next one's from 115; scrolled to the last card (591dp), the
	// tenth is at 44..104 and the eleventh from 109.
	ys := []int{70, 107, 112}
	was := []color.NRGBA{active, active, SP.Sidebar}
	now := []color.NRGBA{draft, SP.Sidebar, draft}
	is := func(img *image.RGBA, want []color.NRGBA) bool {
		for i, y := range ys {
			if !nearPixel(img.RGBAAt(20, y), want[i]) {
				return false
			}
		}
		return true
	}
	if !is(took, was) && !is(took, now) {
		t.Errorf("the frame that took End has %v, %v and %v at y %v: neither where the list was nor where it went",
			took.RGBAAt(20, ys[0]), took.RGBAAt(20, ys[1]), took.RGBAAt(20, ys[2]), ys)
	}
	if !is(next, now) {
		t.Errorf("the frame after End has %v, %v and %v at y %v, not the list at its last card",
			next.RGBAAt(20, ys[0]), next.RGBAAt(20, ys[1]), next.RGBAAt(20, ys[2]), ys)
	}
	wantPixel(t, next, image.Pt(20, 740), draft, "the last card, at the bottom (694..754)")
}

// A right-click on a hovered card's control opens the row's menu, as T3's
// contextmenu bubbles up to the row (§5.11, §5.13); a click on one is
// still the control's alone (TestSettleIsntARowClick).
func TestRightClickOnAControl(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{Items: []SidebarItem{
		{Row: &ThreadRow{ID: "e/1", Title: "One", Age: "5m", CanSettle: true}},
	}}}
	frame, r := sidebarFrames(th, s)
	frame()
	r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, 68), Source: pointer.Mouse})
	frame()
	frame()
	at := f32.Pt(225, 68) // Settle
	r.Queue(pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse})
	frame()
	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonSecondary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	if acts := frame(); len(acts) != 1 || acts[0].ID != "thread:e/1" || acts[0].Kind != "context" {
		t.Fatalf("actions %+v, want just the row's context menu", acts)
	}
}

// A snooze menu whose card is out of sight closes: nothing would draw it,
// and the window-wide catch under menus would swallow the next click.
func TestSnoozeMenuGoesWithItsCard(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true}
	for i := range 20 {
		s.Sidebar.Items = append(s.Sidebar.Items, SidebarItem{Row: &ThreadRow{ID: fmt.Sprint("e/", i), Title: fmt.Sprint("Thread ", i), Age: "1h", CanSnooze: true}})
	}
	s.Sidebar.Snooze = &SnoozeMenu{Row: "e/1", Choices: []MenuItem{{ID: "a", Label: "In 1 hour", Detail: "11:00 AM"}}}
	frame, _ := sidebarFrames(th, s)
	if acts := frame(); len(acts) != 0 {
		t.Fatalf("with its card in sight: actions %+v, want none", acts)
	}
	s.Sidebar.Snooze.Row = "e/19"
	if acts := frame(); len(acts) != 1 || acts[0].ID != "menus.close" {
		t.Fatalf("with its card out of sight: actions %+v, want menus.close", acts)
	}
}

// A receded working card is drawn at 70% (opacity-70 on the whole row,
// SidebarThreadRow.tsx, §3.5), its project's icon and its jump key with
// the rest of it.
func TestRecededCardFades(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	s := &Scene{Sidebar: Sidebar{JumpHints: true, Items: []SidebarItem{{Row: &ThreadRow{ID: "e/1", Title: "Working", Project: "tern",
		Icon: ProjectIcon{Text: "TN", Color: "amber"}, Status: StatusWorking, Label: "Working", Recede: true, Jump: 1}}}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	in := rowInk{bg: SP.Sidebar, k: 0.7}
	// Line 1 is 110..130: the project's icon at 18..34, its tile 1dp in;
	// the jump key at the slot's end, 237, which its padding reaches.
	wantPixel(t, img, image.Pt(19, 120), in.of(over(alpha(projectColor("amber"), 0x24), SP.Sidebar)), "the project's tile, at 70%")
	wantPixel(t, img, image.Pt(235, 120), in.of(alpha(P.Foreground, 0x12)), "the jump key, at 70%")
}

// A slim row's Un-settle button has its tooltip to itself: the row's hover
// card stands aside while it shows, as one tooltip in the list's group
// does for another (02 §8.10, 04 §3.15).
func TestSlimRowTipStandsTheCardAside(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{HasProjects: true, Items: []SidebarItem{
		{Shelf: &Shelf{ID: "settled", Label: "Settled", Count: 1, Open: true}},
		{Row: &ThreadRow{ID: "e/1", Slim: true, Shelf: "settled", Title: "Settled", Project: "tern", Icon: ProjectIcon{Text: "TN", Color: "amber"}, Age: "3d", CanSettle: true}},
	}}}
	lw := newLiveWindow(t, image.Pt(1440, 900))
	t0 := time.Now()
	at := func(d time.Duration) *image.RGBA {
		return lw.frame(t, t0.Add(d), func(gtx C) D { return th.Desktop(gtx, s) })
	}
	glass := over(alpha(P.Popover, 0xd5), P.Background)
	border := over(alpha(P.Foreground, 0x1a), glass)
	// The row is 812..848, its card's left border at 251.
	edge := image.Pt(251, 830)
	at(0)
	lw.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, 830), Source: pointer.Mouse})
	at(10 * time.Millisecond)
	at(160 * time.Millisecond)
	wantPixel(t, at(400*time.Millisecond), edge, border, "the row's card")
	// Un-settle is 26 by 24 at the slot's end, 4dp into the padding:
	// 215..241, 818..842. Its tooltip sits 4dp over it, centred.
	lw.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(228, 830), Source: pointer.Mouse})
	at(500 * time.Millisecond)
	at(660 * time.Millisecond)
	img := at(900 * time.Millisecond)
	wantPixel(t, img, edge, SP.Sidebar, "the card, stood aside")
	tip := th.tooltip(layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(320, 200)}}, "Un-settle thread").Size
	wantPixel(t, img, image.Pt(215+(26-tip.X)/2+3, 818-4-tip.Y/2), P.Popover, "the button's tooltip")
}

// A menu opened from a row with the keyboard gives the focus back to the
// row as it closes, so the list's keys carry on from there
// (contextMenuFallback.ts restores the focus it took, §5.13).
func TestMenuGivesFocusBack(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Sidebar: Sidebar{Items: []SidebarItem{
		{Row: &ThreadRow{ID: "e/1", Title: "One", Age: "1h"}},
		{Row: &ThreadRow{ID: "e/2", Title: "Two", Age: "1h"}},
		{Row: &ThreadRow{ID: "e/3", Title: "Three", Age: "1h"}},
	}}}
	lw := newLiveWindow(t, image.Pt(1440, 900))
	frame := func() *image.RGBA { return lw.frame(t, time.Now(), func(gtx C) D { return th.Desktop(gtx, s) }) }
	press := func(name key.Name, mods key.Modifiers) ([]Action, *image.RGBA) {
		lw.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
		frame()
		acts := th.Actions()
		img := frame()
		return append(acts, th.Actions()...), img
	}
	has := func(acts []Action, id, kind string) bool {
		for _, a := range acts {
			if a.ID == id && a.Kind == kind {
				return true
			}
		}
		return false
	}
	// The cards are 60dp at 102, 167 and 232: a focused one's 2dp ring runs
	// down its left edge at x 8.
	ring := func(row int) image.Point { return image.Pt(8, 132+65*row) }
	img := frame()
	for i := 0; i < 6 && !nearPixel(img.RGBAAt(ring(0).X, ring(0).Y), SP.Ring); i++ {
		lw.r.MoveFocus(key.FocusForward)
		frame()
		img = frame()
	}
	wantPixel(t, img, ring(0), SP.Ring, "Tab onto the first row")
	_, img = press(key.NameDownArrow, 0)
	wantPixel(t, img, ring(1), SP.Ring, "Down onto the second")
	if acts, _ := press(key.NameF10, key.ModShift); !has(acts, "thread:e/2", "context") {
		t.Fatalf("Shift+F10: actions %+v, want the second row's menu", acts)
	}
	s.Menu = &ContextMenu{Keyboard: true, Items: []MenuItem{{ID: "pin", Label: "Pin thread", Icon: "pin"}}}
	frame()
	wantPixel(t, frame(), ring(1), SP.Sidebar, "the menu, with the focus")
	if acts, _ := press(key.NameEscape, 0); !has(acts, "menus.close", "click") {
		t.Fatalf("Escape: actions %+v, want the menu closed", acts)
	}
	s.Menu = nil
	frame()
	wantPixel(t, frame(), ring(1), SP.Ring, "the menu closed: the second row has the focus back")
	_, img = press(key.NameDownArrow, 0)
	wantPixel(t, img, ring(2), SP.Ring, "and Down goes on to the third")
}

// A row drawn again after a frame without it starts at rest, as a remounted
// row does: what the theme kept for it went with the frame that didn't draw
// it, so the rows a session shows don't pile up there (§6).
func TestRowComesBackAtRest(t *testing.T) {
	th := NewTheme()
	row := &ThreadRow{ID: "e/1", Slim: true, Shelf: "settled", Title: "Settled", Project: "tern", Icon: ProjectIcon{Text: "TN", Color: "amber"}, Age: "3d", Hover: true}
	list := func(rows ...*ThreadRow) *Scene {
		s := &Scene{Sidebar: Sidebar{HasProjects: true, Items: []SidebarItem{{Shelf: &Shelf{ID: "settled", Label: "Settled", Count: 1, Open: true}}}}}
		for _, r := range rows {
			s.Sidebar.Items = append(s.Sidebar.Items, SidebarItem{Row: r})
		}
		return s
	}
	draw := func(s *Scene) *image.RGBA {
		return drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	}
	draw(list(row)) // hovered: its icon in colour
	draw(list())    // gone for a frame
	row.Hover = false
	img := draw(list(row))
	// The slim row is 812..848: its icon at 18..34 and 822..838.
	tile := over(alpha(projectColor("amber"), 0x24), SP.Sidebar)
	wantPixel(t, img, image.Pt(19, 830), over(alpha(grey(tile), 0x66), SP.Sidebar), "the icon back, grey at 40% at once")
}
