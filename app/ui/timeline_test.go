package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Each row kind's height is T3's measured one (docs/ui/06-timeline.md
// §3.1), its padding below included; at 4x every one lands on whole
// pixels.
func TestTimelineRowHeights(t *testing.T) {
	th := NewTheme()
	three := []WorkEntry{{ID: "e1", Icon: "terminal", Label: "go test ./..."}, {ID: "e2", Icon: "eye", Label: "Read go.mod"}, {ID: "e3", Icon: "search", Label: "Searched retry"}}
	for _, c := range []struct {
		name string
		r    TimelineRow
		want float32
	}{
		{"user message, one line", TimelineRow{ID: "u", Kind: RowUser, Message: &Message{ID: "u", Text: "Hello"}}, 90.75},
		{"user message another follows", TimelineRow{ID: "g", Kind: RowUser, Message: &Message{ID: "g", Text: "Hello", GroupedNext: true}}, 46.75 + 4},
		{"assistant message, its meta inline", TimelineRow{ID: "a", Kind: RowAssistant, Message: &Message{ID: "a", Text: "Done.", Meta: true, Copy: true}}, 2 + 22.75 + 6 + 24 + 2 + 16},
		{"turn fold", TimelineRow{ID: "f", Kind: RowTurnFold, Group: "run", Label: "Worked for 2m 14s"}, 41.75},
		{"work row, standalone", TimelineRow{ID: "w", Kind: RowWork, Entries: three[:1]}, 36},
		{"work toggle, open", TimelineRow{ID: "g", Kind: RowWorkToggle, Group: "g", Open: true, Icon: "hammer", Label: "Ran 1 command and read 1 file"}, 28},
		{"open group list", TimelineRow{ID: "l", Kind: RowWork, Group: "g", Open: true, Entries: three}, 3*24 + 4},
		{"working row", TimelineRow{ID: "k", Kind: RowWorking}, 43},
		{"thinking row", TimelineRow{ID: "t", Kind: RowThinking, Label: "Thinking", Live: true}, 36},
		{"compaction", TimelineRow{ID: "c", Kind: RowCompaction, Label: "Context compacted", Before: "180K", After: "12K"}, 40},
		{"event divider", TimelineRow{ID: "d", Kind: RowDivider, Label: "Run interrupted", Icon: "circle-stop"}, 48},
		// The bordered button is 26 tall, the row 42 (§3.9).
		{"divider with an action", TimelineRow{ID: "d", Kind: RowDivider, Label: "Forked from conversation", Icon: "git-fork",
			Action: "Open source conversation", Thread: "env/t"}, 42 + 8},
		{"a detail that wraps", TimelineRow{ID: "d", Kind: RowDivider, Label: "Run interrupted", Icon: "circle-stop",
			Detail: "first line\nsecond line", Wrap: true}, 40 + 16 + 8},
	} {
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 4, PxPerSp: 4}, Constraints: layout.Constraints{Max: image.Pt(4*800, 4*2000)}, Now: time.Now()}
		if got := th.timelineRow(gtx, &Scene{}, &c.r, 0, 4*736).Size.Y; got != int(c.want*4) {
			t.Errorf("%s: %.2fdp tall, want %.2f", c.name, float32(got)/4, c.want)
		}
	}
}

// rowDrawn draws r alone at the top of a 736dp lane on the background.
func rowDrawn(t *testing.T, th *Theme, r TimelineRow, size image.Point, scale float32) *image.RGBA {
	t.Helper()
	return drawn(t, size, scale, func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		return th.timelineRow(gtx, &Scene{}, &r, 0, dp(gtx, 736))
	})
}

// A user's message is a 46.75dp bubble of --message-surface at the lane's
// right edge, with 12dp of padding round its line (§3.3).
func TestUserBubble(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		surface := rgb(0xebebed)
		if d {
			surface = rgb(0x1e1e1e)
		}
		img := rowDrawn(t, NewTheme(), TimelineRow{ID: "u", Kind: RowUser, Message: &Message{ID: "u", Text: "Hello"}}, image.Pt(736, 100), 2)
		x := 2 * (736 - 24) // over the line's last characters, clear of the corners
		wantPixel(t, img, image.Pt(x, 2*4), surface, "the bubble's top padding")
		wantPixel(t, img, image.Pt(x, 92), surface, "the bubble's last row of pixels")
		wantPixel(t, img, image.Pt(x, 96), P.Background, "under the bubble")
		wantPixel(t, img, image.Pt(2*400, 2*20), P.Background, "left of the bubble")
	}
}

// A message another of yours follows tightens its bottom-right corner to
// 6dp (rounded-br-md), and floats its actions beside the bubble (§5.2.10).
func TestGroupedUserBubble(t *testing.T) {
	th := NewTheme()
	corner := image.Pt(2*736-5, 88) // 5px in from the bubble's bottom-right corner, at 2x
	for _, grouped := range []bool{false, true} {
		img := rowDrawn(t, th, TimelineRow{ID: fmt.Sprint("g", grouped), Kind: RowUser, Message: &Message{ID: "g", Text: "Hello", GroupedNext: grouped}}, image.Pt(736, 100), 2)
		want, what := P.Background, "outside the bubble's 16dp corner"
		if grouped {
			want, what = rgb(0xebebed), "inside the bubble's 6dp corner"
		}
		wantPixel(t, img, corner, want, what)
	}
}

// The turn fold's rule is border/60: #eeeeef in light, and in dark, where
// --border is itself white at 8%, #0c0c0c (§3.7).
func TestTurnFoldRule(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		want := rgb(0xeeeeef)
		if d {
			want = rgb(0x0c0c0c)
		}
		img := rowDrawn(t, NewTheme(), TimelineRow{ID: "f", Kind: RowTurnFold, Group: "run", Label: "Worked for 2m 14s"}, image.Pt(736, 50), 2)
		// 4 + 22.75 + 8dp down: px 8 + 46 + 16.
		wantPixel(t, img, image.Pt(2*400, 70), want, "the fold's rule")
		wantPixel(t, img, image.Pt(2*400, 66), P.Background, "over the rule")
	}
}

// Inline code sits in a 4dp-round box of foreground at 10% (#e5e5e5 on
// the light canvas) padded 0.35em; links are --info-foreground (§3.5).
func TestInlineCodeAndLinks(t *testing.T) {
	th := NewTheme()
	img := drawn(t, image.Pt(600, 40), 2, func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		return th.markdown(gtx, "`go test` and [the docs](https://example.com/docs)", mdBody(), nil)
	})
	wantPixel(t, img, image.Pt(2*2, 2*11), rgb(0xe5e5e5), "the code box's left padding")
	link := rgb(0x1447e6)
	found := false
	for y := 0; y < 2*23 && !found; y++ {
		for x := 0; x < 2*600 && !found; x++ {
			c := img.RGBAAt(x, y)
			found = near(c.R, link.R, 6) && near(c.G, link.G, 6) && near(c.B, link.B, 6)
		}
	}
	if !found {
		t.Error("no pixel of the link is #1447e6")
	}
}

func near(a, b, d uint8) bool { return a-b <= d || b-a <= d }

// A code block's frame is --secondary with a 1dp border at border/70 in
// light, and input/32 without a border in dark (§3.6).
func TestCodeBlockFrame(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		img := drawn(t, image.Pt(600, 100), 2, func(gtx C) D {
			fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
			return th.markdown(gtx, "```go\nfunc main() {}\n```", mdBody(), nil)
		})
		frame, edge := rgb(0xfafafa), rgb(0xebebed)
		if d {
			frame, edge = rgb(0x090909), rgb(0x090909)
		}
		wantPixel(t, img, image.Pt(2*300, 2*3), frame, "the frame over its header")
		wantPixel(t, img, image.Pt(0, 2*40), edge, "the frame's left edge")
	}
}

// "Load earlier turns" is 28dp under the 24dp the top fade keeps, then
// 8dp: the first row starts 60dp down, at y 112 in the window (§3.1).
// Clicking it asks for the next page.
func TestHistoryControl(t *testing.T) {
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 4, PxPerSp: 4}, Constraints: layout.Constraints{Max: image.Pt(4*800, 4*400)}}
	if h := th.timelineHeader(gtx, &Timeline{MoreHistory: true}, 0, 4*736).Size.Y; h != 4*60 {
		t.Errorf("history control: %.2fdp, want 60", float32(h)/4)
	}
	if h := th.timelineHeader(gtx, &Timeline{}, 0, 4*736).Size.Y; h != 4*24 {
		t.Errorf("no history: %.2fdp, want 24", float32(h)/4)
	}
	click := timelineClicks(th, &Scene{Timeline: Timeline{ID: "h", MoreHistory: true, Rows: []TimelineRow{{ID: "d", Kind: RowDivider, Label: "Run interrupted"}}}})
	if acts := click(image.Pt(32+100, 24+14)); len(acts) != 1 || acts[0] != (Action{ID: "more", Kind: "click", Clicks: 1}) {
		t.Fatalf("clicking the history control sent %+v", acts)
	}
}

// timelineClicks lays s's timeline out in an 800x600 canvas at 1x, the
// lane 32dp in, and returns a click at a point and the frame's actions.
func timelineClicks(th *Theme, s *Scene) func(p image.Point) []Action {
	click, _ := timelineRouter(th, s)
	return click
}

// timelineRouter is timelineClicks with the router, for the clipboard.
func timelineRouter(th *Theme, s *Scene) (func(p image.Point) []Action, *gioinput.Router) {
	r := new(gioinput.Router)
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 600)), Source: r.Source(), Now: time.Now()}
		th.timeline(gtx, s, 32, 736, 0)
		r.Frame(&ops)
		return th.Actions()
	}
	frame()
	return func(p image.Point) []Action {
		fp := f32.Pt(float32(p.X), float32(p.Y))
		r.Queue(pointer.Event{Kind: pointer.Move, Position: fp, Source: pointer.Mouse})
		frame()
		r.Queue(pointer.Event{Kind: pointer.Press, Position: fp, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			pointer.Event{Kind: pointer.Release, Position: fp, Source: pointer.Mouse})
		return frame()
	}, r
}

// A turn fold, a group's header and an entry that opens each send their
// own toggle: "fold:<run>", "group:<group>", "entry:<item>".
func TestWorkLogToggles(t *testing.T) {
	th := NewTheme()
	s := &Scene{Timeline: Timeline{ID: "toggles", Rows: []TimelineRow{
		{ID: "turn-fold:run-1", Kind: RowTurnFold, Group: "run-1", Label: "Worked for 4m 12s"},
		{ID: "work-toggle:e1", Kind: RowWorkToggle, Group: "work-group:e1", Icon: "terminal", Label: "Ran 2 commands"},
		{ID: "e3", Kind: RowWork, Entries: []WorkEntry{{ID: "e3", Icon: "terminal", Label: "go vet ./...", Expands: true}}},
	}}}
	click := timelineClicks(th, s)
	// The fold's 22.75dp button is 4dp under the 24dp header; the group's
	// 28dp header follows the fold's 41.75; the entry the group's 36.
	for _, c := range []struct {
		y    int
		want string
	}{{24 + 4 + 11, "fold:run-1"}, {24 + 42 + 14, "group:work-group:e1"}, {24 + 42 + 36 + 14, "entry:e3"}} {
		if acts := click(image.Pt(32+40, c.y)); len(acts) != 1 || acts[0] != (Action{ID: c.want, Kind: "click", Clicks: 1}) {
			t.Errorf("click at y %d sent %+v, want %s", c.y, acts, c.want)
		}
	}
}

// The list follows its end (§5.10): rows taller than the canvas show the
// last one over the footer; a wheel up lets go, "Scroll to end" shows
// 150ms after the list left the end, and clicking it follows again.
func TestTimelineFollow(t *testing.T) {
	th := NewTheme()
	s := &Scene{Timeline: Timeline{ID: "follow"}}
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("u%d", i)
		s.Timeline.Rows = append(s.Timeline.Rows, TimelineRow{ID: id, Kind: RowUser, Message: &Message{ID: id, Text: "Hello"}})
	}
	size := image.Pt(800, 600)
	draw := func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
		th.timeline(gtx, s, 32, 736, 0)
		return D{Size: size}
	}
	// The last bubble ends 60dp over the canvas's bottom: the footer's 16,
	// then its row's padding, actions and gap. Probe its bottom padding,
	// under the text.
	last := image.Pt(32+736-16, 600-60-4)
	wantPixel(t, drawn(t, size, 1, draw), last, P.MessageSurface, "the last bubble, followed")

	var r gioinput.Router
	var ops op.Ops
	at := time.Now().Add(-time.Second)
	frame := func() {
		ops.Reset()
		draw(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size), Source: r.Source(), Now: at})
		r.Frame(&ops)
	}
	frame()
	// 60dp up puts the row before's actions, which show only on hover,
	// where the last bubble was.
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: f32.Pt(400, 300), Scroll: f32.Pt(0, -60)})
	frame()
	at = at.Add(200 * time.Millisecond)
	frame()
	img := drawn(t, size, 1, draw)
	wantPixel(t, img, last, P.Background, "where the last bubble was, scrolled up")
	// The pill: 24dp tall, centred on the lane, 4 + 6dp over the bottom;
	// its border is border/60.
	wantPixel(t, img, image.Pt(32+736/2, 600-4-6-24), over(faded(P.Border, 0x99), P.Background), "the scroll-to-end pill's top border")
	p := f32.Pt(32+736/2, 600-4-6-12)
	r.Queue(pointer.Event{Kind: pointer.Move, Position: p, Source: pointer.Mouse})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Press, Position: p, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: p, Source: pointer.Mouse})
	frame()
	if acts := th.Actions(); len(acts) != 0 {
		t.Errorf("the pill sent %+v; it scrolls the timeline itself", acts)
	}
	wantPixel(t, drawn(t, size, 1, draw), last, P.MessageSurface, "the last bubble, followed again")
}

// Rows that fit start at the top, under the 24dp the top fade keeps.
func TestShortTimelineStartsAtTheTop(t *testing.T) {
	th := NewTheme()
	s := &Scene{Timeline: Timeline{ID: "short", Rows: []TimelineRow{{ID: "u", Kind: RowUser, Message: &Message{ID: "u", Text: "Hello"}}}}}
	img := drawn(t, image.Pt(800, 600), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		th.timeline(gtx, s, 32, 736, 0)
		return D{Size: gtx.Constraints.Max}
	})
	wantPixel(t, img, image.Pt(32+736-16, 24+4), P.MessageSurface, "the bubble under the header")
	wantPixel(t, img, image.Pt(32+736-16, 600-40), P.Background, "the canvas's bottom")
}

// A table's copy menu copies the pick the frame it's clicked, then closes
// (MarkdownTable.tsx:15-70).
func TestTableCopyMenu(t *testing.T) {
	click, r := markdownClicks("| A | B |\n|---|---|\n| 1 | 2 |", mdBody())
	// The head row is 8.8 + 22.75 + 8.8 and its rule, the body row 7.2 +
	// 22.75 + 7.2 and its; the button is 2dp under them, 4dp past the edge.
	button := image.Pt(600+4-12, 41+1+37+1+2+12)
	click(button)
	// The menu hangs 4dp under the button by its top-right corner, its
	// rows 28dp tall inside 4dp of padding.
	csv := image.Pt(600-60, 41+1+37+1+2+24+4+4+28+14)
	if acts := click(csv); len(acts) != 0 {
		t.Fatalf("the pick reached core: %+v", acts)
	}
	if mime, data, ok := r.WriteClipboard(); !ok || mime != "application/text" || string(data) != "A,B\n1,2" {
		t.Fatalf("clipboard = %q %q %v", mime, data, ok)
	}
	click(csv)
	if _, data, ok := r.WriteClipboard(); ok {
		t.Errorf("the menu stayed open: a second click copied %q", data)
	}
}

// The plan card's menu copies the plan as T3 exports it, one line break
// at its end, and downloads it under its heading's name
// (ProposedPlanCard.tsx:41-140, proposedPlan.ts:56-63).
func TestPlanCardMenu(t *testing.T) {
	th := NewTheme()
	plan := "# Retry the uploader!\n\nRetry each chunk.\n\n\n"
	s := &Scene{Timeline: Timeline{ID: "plan", Rows: []TimelineRow{{ID: "plan-1", Kind: RowPlan, Plan: plan}}}}
	click, r := timelineRouter(th, s)
	// The ellipsis is the header's last, 24dp, the card padded 20.
	menu := image.Pt(32+736-20-12, 24+20+12)
	// The menu's top-right is 4dp under the header; rows 28dp, padded 4.
	row := func(i int) image.Point { return image.Pt(32+736-20-80, 24+20+24+4+4+28*i+14) }
	click(menu)
	if acts := click(row(0)); len(acts) != 0 {
		t.Fatalf("copy reached core: %+v", acts)
	}
	if _, data, ok := r.WriteClipboard(); !ok || string(data) != "# Retry the uploader!\n\nRetry each chunk.\n" {
		t.Fatalf("clipboard = %q %v", data, ok)
	}
	click(menu)
	want := Action{ID: "plan.download:retry-the-uploader.md", Kind: "click", Text: "# Retry the uploader!\n\nRetry each chunk.\n"}
	if acts := click(row(1)); len(acts) != 1 || acts[0] != want {
		t.Fatalf("download sent %+v", acts)
	}
}

// T3's plan title needs white space after its hashes; a plan without a
// heading downloads as plan.md, and an empty preview says so
// (proposedPlanText.ts:1-4, proposedPlan.ts:3-59).
func TestPlanTitle(t *testing.T) {
	for _, c := range []struct{ md, title, file string }{
		{"# Ship it\nwork", "Ship it", "ship-it.md"},
		{"#hashtag\n\n## Real title", "Real title", "real-title.md"},
		{"no heading", "Proposed plan", "plan.md"},
		{"## It's (finally) done.", "It's (finally) done.", "its-finally-done.md"},
	} {
		if got := planTitle(c.md); got != c.title {
			t.Errorf("planTitle(%q) = %q, want %q", c.md, got, c.title)
		}
		if got := planFile(c.md); got != c.file {
			t.Errorf("planFile(%q) = %q, want %q", c.md, got, c.file)
		}
	}
	if got := planPreview("# Only a title"); got != "Only a title" {
		t.Errorf("a title-only preview is %q", got)
	}
	if got := planPreview("   \n"); got != "Plan preview unavailable." {
		t.Errorf("an empty preview is %q", got)
	}
}

// The working row asks for its next frame on its own, on the second, so
// the window needn't redraw in between while a run works (§8 note 7).
func TestWorkingRowTicksEachSecond(t *testing.T) {
	th := NewTheme()
	now := time.Now()
	since := now.Add(-12300 * time.Millisecond)
	s := &Scene{Timeline: Timeline{ID: "tick", Rows: []TimelineRow{{ID: "working-indicator-row", Kind: RowWorking, Since: since}}}}
	var r gioinput.Router
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(800, 600)), Source: r.Source(), Now: now}
	th.timeline(gtx, s, 32, 736, 0)
	r.Frame(&ops)
	if at, ok := r.WakeupTime(); !ok || !at.Equal(since.Add(13*time.Second)) {
		t.Fatalf("the next frame is at %v (%v), want %v", at, ok, since.Add(13*time.Second))
	}
}

// A fork's divider is a button that opens the conversation on its other
// side, with that in its tooltip (V2LifecycleRow.tsx:97-105).
func TestForkDividerOpensThread(t *testing.T) {
	th := NewTheme()
	s := &Scene{Timeline: Timeline{ID: "fork", Rows: []TimelineRow{{ID: "fork-1", Kind: RowDivider, Label: "Forked from conversation",
		Icon: "git-fork", Action: "Open source conversation", Thread: "env/source"}}}}
	click := timelineClicks(th, s)
	if acts := click(image.Pt(32+736/2, 24+21)); len(acts) != 1 || acts[0] != (Action{ID: "thread:env/source", Kind: "click", Clicks: 1}) {
		t.Fatalf("clicking the fork's button sent %+v", acts)
	}
}

// A row's time shows its long form in a tooltip once the pointer rests on
// it (TimelineRowParts.tsx:13-40).
func TestTimestampTooltip(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	s := &Scene{Timeline: Timeline{ID: "stamp", Rows: []TimelineRow{{ID: "turn-fold:r", Kind: RowTurnFold, Group: "r",
		Label: "Worked for 2m 14s", At: time.Now().Add(-time.Hour)}}}}
	size := image.Pt(800, 600)
	draw := func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
		th.timeline(gtx, s, 32, 736, 0)
		return D{Size: size}
	}
	var r gioinput.Router
	var ops op.Ops
	at := time.Now().Add(-10 * time.Second)
	frame := func() {
		ops.Reset()
		draw(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size), Source: r.Source(), Now: at})
		r.Frame(&ops)
	}
	frame()
	// The time sits at the row's end, 2dp in, its 16dp line 3dp under the
	// fold's 4dp top: rest on it.
	r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(32+736-2-20, 24+4+3+8), Source: pointer.Mouse})
	frame()
	frame()
	at = at.Add(time.Second)
	frame()
	// The tooltip's bottom padding, 4dp over the time.
	wantPixel(t, drawn(t, size, 1, draw), image.Pt(32+736-2-20, 24+4+3-4-2), P.Popover, "the tooltip over the time")
}

// "Show full message" ends at the text's edge, 12dp in from the bubble's,
// so its label ends 20dp in (TimelineUserMessageRow.tsx:1181-1196).
func TestShowFullMessageButton(t *testing.T) {
	long := strings.Repeat("a line of the message\n", 20)
	img := rowDrawn(t, NewTheme(), TimelineRow{ID: "u", Kind: RowUser, Message: &Message{ID: "u", Text: long}}, image.Pt(736, 300), 1)
	// The footer: 12 + 176 + 6dp down, 24 tall.
	surface := rgb(0xebebed)
	for x := 736 - 18; x < 736-12; x++ {
		for y := 12 + 176 + 6 + 4; y < 12+176+6+20; y++ {
			if c := img.RGBAAt(x, y); !near(c.R, surface.R, 4) || !near(c.G, surface.G, 4) || !near(c.B, surface.B, 4) {
				t.Fatalf("the label reaches x %d, under 18dp from the bubble's edge", x)
			}
		}
	}
	if firstInk(img, 736-60, 12+176+6+4, 12+176+6+20) < 0 {
		t.Error("no label at the footer")
	}
}

// An expanded command is coloured per shell word as T3's grammar has it:
// the command's name a function, its arguments variables
// (ShellCommandBlock.tsx; §3.6).
func TestShellCommandColours(t *testing.T) {
	th := NewTheme()
	img := drawn(t, image.Pt(400, 40), 2, func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		return D{Size: image.Pt(gtx.Constraints.Max.X, th.toolInspector(gtx, &EntryBody{Command: "go test ./upload/..."}, "e"))}
	})
	has := func(c color.NRGBA, x0, x1 int) bool {
		for y := 0; y < 2*22; y++ {
			for x := x0; x < x1; x++ {
				p := img.RGBAAt(x, y)
				if near(p.R, c.R, 12) && near(p.G, c.G, 12) && near(p.B, c.B, 12) {
					return true
				}
			}
		}
		return false
	}
	// Mono 13 is 7.8dp a character: "go" ends at 15.6, "test" starts at 23.4.
	if !has(rgb(0x693acf), 0, 2*16) {
		t.Error("the command's name isn't #693acf")
	}
	if !has(rgb(0x8f4e17), 2*23, 2*55) || has(rgb(0x693acf), 2*23, 2*55) {
		t.Error("its first argument isn't #8f4e17")
	}
}

// An open group keeps its scrollbar's 6dp gutter, so its rows' chevrons
// end 6dp further in than a standalone row's (§2.7, §3.7).
func TestOpenGroupGutter(t *testing.T) {
	entries := []WorkEntry{{ID: "e1", Icon: "terminal", Label: "ls", Expands: true}, {ID: "e2", Icon: "eye", Label: "Read go.mod"}}
	img := rowDrawn(t, NewTheme(), TimelineRow{ID: "l", Kind: RowWork, Group: "g", Open: true, Entries: entries}, image.Pt(736, 60), 1)
	ink := func(x0, x1 int) bool {
		for y := 0; y < 24; y++ {
			for x := x0; x < x1; x++ {
				if c := img.RGBAAt(x, y); !near(c.R, P.Background.R, 6) || !near(c.G, P.Background.G, 6) || !near(c.B, P.Background.B, 6) {
					return true
				}
			}
		}
		return false
	}
	if !ink(736-22, 736-10) {
		t.Error("no chevron 10 to 22dp in from the lane's end")
	}
	if ink(736-9, 736) {
		t.Error("the chevron reaches the scrollbar's gutter")
	}
}

// Opening a fold while the end is followed keeps the fold where it is,
// its rows growing below it, rather than pinning the end and pushing the
// fold away; while the turn works the live end stays followed instead
// (MessagesTimeline.tsx:534-556).
func TestToggleKeepsTheRowInPlace(t *testing.T) {
	for _, working := range []bool{false, true} {
		th := NewTheme()
		s := &Scene{Timeline: Timeline{ID: fmt.Sprint("anchor", working), Working: working}}
		for i := 0; i < 20; i++ {
			id := fmt.Sprintf("u%d", i)
			s.Timeline.Rows = append(s.Timeline.Rows, TimelineRow{ID: id, Kind: RowUser, Message: &Message{ID: id, Text: "Hello"}})
		}
		fold := TimelineRow{ID: "turn-fold:run", Kind: RowTurnFold, Group: "run", Label: "Worked for 4m 12s"}
		reply := TimelineRow{ID: "a", Kind: RowAssistant, Message: &Message{ID: "a", Text: "Done.", Meta: true, Copy: true}}
		s.Timeline.Rows = append(s.Timeline.Rows, fold, reply)
		size := image.Pt(800, 600)
		draw := func(gtx C) D {
			fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
			th.timeline(gtx, s, 32, 736, 0)
			return D{Size: size}
		}
		rule := over(faded(P.Border, 0x99), P.Background)
		ruleAt := func(img *image.RGBA) int {
			for y := 599; y >= 0; y-- {
				if c := img.RGBAAt(32+400, y); near(c.R, rule.R, 1) && near(c.G, rule.G, 1) && near(c.B, rule.B, 1) {
					return y
				}
			}
			return -1
		}
		var r gioinput.Router
		var ops op.Ops
		frame := func() {
			ops.Reset()
			draw(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size), Source: r.Source(), Now: time.Now()})
			r.Frame(&ops)
			th.Actions()
		}
		frame()
		before := ruleAt(drawn(t, size, 1, draw))
		if before < 0 {
			t.Fatal("no fold rule")
		}
		// The fold's button is 8dp over its rule, 22.75 tall.
		at := f32.Pt(32+40, float32(before-8-11))
		r.Queue(pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse})
		frame()
		r.Queue(pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse})
		frame()
		// Core opens the fold: its work shows under it.
		var work []TimelineRow
		for i := 0; i < 5; i++ {
			id := fmt.Sprintf("w%d", i)
			work = append(work, TimelineRow{ID: id, Kind: RowWork, Entries: []WorkEntry{{ID: id, Icon: "terminal", Label: "make"}}})
		}
		rows := s.Timeline.Rows
		s.Timeline.Rows = append(append(append([]TimelineRow{}, rows[:21]...), work...), rows[21:]...)
		s.Timeline.Rows[20].Open = true
		frame()
		frame()
		after := ruleAt(drawn(t, size, 1, draw))
		if !working && after != before {
			t.Errorf("the fold moved from y %d to %d", before, after)
		}
		if working && after >= before {
			t.Errorf("working: the fold stayed at y %d, the end unfollowed", after)
		}
	}
}

// A file sent without words is a 256 by 52dp card of --message-surface at
// the lane's end, its icon in a box at background/60 (§3.3 file card).
func TestFileCard(t *testing.T) {
	img := rowDrawn(t, NewTheme(), TimelineRow{ID: "u", Kind: RowUser, Message: &Message{ID: "u",
		Files: []File{{ID: "f", Name: "report.pdf", Size: 2516582}}}}, image.Pt(736, 100), 1)
	wantPixel(t, img, image.Pt(736-40, 26), P.MessageSurface, "the card between its text and its button")
	wantPixel(t, img, image.Pt(736-256+12, 26), over(faded(P.Background, 0x99), P.MessageSurface), "the icon's box")
	wantPixel(t, img, image.Pt(736-260, 26), P.Background, "left of the card")
}
