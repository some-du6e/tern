package ui

import (
	"fmt"
	"image"
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
		{"assistant message, its meta inline", TimelineRow{ID: "a", Kind: RowAssistant, Message: &Message{ID: "a", Text: "Done.", Meta: true, Copy: true}}, 2 + 22.75 + 6 + 24 + 2 + 16},
		{"turn fold", TimelineRow{ID: "f", Kind: RowTurnFold, Group: "run", Label: "Worked for 2m 14s"}, 41.75},
		{"work row, standalone", TimelineRow{ID: "w", Kind: RowWork, Entries: three[:1]}, 36},
		{"work toggle, open", TimelineRow{ID: "g", Kind: RowWorkToggle, Group: "g", Open: true, Icon: "hammer", Label: "Ran 1 command and read 1 file"}, 28},
		{"open group list", TimelineRow{ID: "l", Kind: RowWork, Group: "g", Open: true, Entries: three}, 3*24 + 4},
		{"working row", TimelineRow{ID: "k", Kind: RowWorking}, 43},
		{"thinking row", TimelineRow{ID: "t", Kind: RowThinking, Label: "Thinking", Live: true}, 36},
		{"compaction", TimelineRow{ID: "c", Kind: RowCompaction, Label: "Context compacted", Before: "180K", After: "12K"}, 40},
		{"event divider", TimelineRow{ID: "d", Kind: RowDivider, Label: "Run interrupted", Icon: "circle-stop"}, 48},
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
		x := 2 * (736 - 16) // in the line's last characters' column, clear of the corners
		wantPixel(t, img, image.Pt(x, 2*4), surface, "the bubble's top padding")
		wantPixel(t, img, image.Pt(x, 92), surface, "the bubble's last row of pixels")
		wantPixel(t, img, image.Pt(x, 96), P.Background, "under the bubble")
		wantPixel(t, img, image.Pt(2*400, 2*20), P.Background, "left of the bubble")
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
	if acts := click(image.Pt(32+100, 24+14)); len(acts) != 1 || acts[0] != (Action{ID: "more", Kind: "click"}) {
		t.Fatalf("clicking the history control sent %+v", acts)
	}
}

// timelineClicks lays s's timeline out in an 800x600 canvas at 1x, the
// lane 32dp in, and returns a click at a point and the frame's actions.
func timelineClicks(th *Theme, s *Scene) func(p image.Point) []Action {
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
	}
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
		if acts := click(image.Pt(32+40, c.y)); len(acts) != 1 || acts[0] != (Action{ID: c.want, Kind: "click"}) {
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
	// then its row's padding, actions and gap.
	last := image.Pt(32+736-16, 600-60-20)
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
	wantPixel(t, img, image.Pt(32+736/2, 600-4-6-24), over(alpha(P.Border, 0x99), P.Background), "the scroll-to-end pill's top border")
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
