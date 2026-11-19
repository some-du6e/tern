package ui

import (
	"image"
	"testing"
)

// Your message's actions end 4dp in from the lane's right: the time, 8dp,
// "Copy message", 2dp, then "Copy prompt link", each 24dp; they copy the
// words and the link (§2.3, TimelineUserMessageRow.tsx:788-840).
func TestUserMessageActions(t *testing.T) {
	link := "https://app.t3.codes/new?prompt=Hello"
	s := &Scene{Timeline: Timeline{ID: "actions", Rows: []TimelineRow{{ID: "u", Kind: RowUser,
		Message: &Message{ID: "u", Text: "Hello", PromptLink: link}}}}}
	click, r := timelineRouter(NewTheme(), s)
	// The bubble is 46.75 tall from the header's 24, the actions 4 under it.
	y := 24 + 47 + 4 + 12
	right := 32 + 736 - 4
	for _, c := range []struct {
		x    int
		want string
	}{{right - 12, link}, {right - 24 - 2 - 12, "Hello"}} {
		if acts := click(image.Pt(c.x, y)); len(acts) != 0 {
			t.Fatalf("a copy reached core: %+v", acts)
		}
		if _, data, ok := r.WriteClipboard(); !ok || string(data) != c.want {
			t.Errorf("clicking x %d copied %q %v, want %q", c.x, data, ok, c.want)
		}
	}
	// A prompt too long for a link has no link button: copy ends the row.
	s.Timeline.Rows[0].Message.PromptLink = ""
	click, r = timelineRouter(NewTheme(), s)
	click(image.Pt(right-12, y))
	if _, data, ok := r.WriteClipboard(); !ok || string(data) != "Hello" {
		t.Errorf("without a link, the row's last button copied %q %v", data, ok)
	}
}

// The reply's meta row is copy, then "Fork from this response", 24dp each
// and 8 apart from the text's left (T3 measured copy at 384 and fork at
// 416 for text at 384, §3.4); clicking fork sends "fork:<message>", and
// not while the fork is on its way.
func TestAssistantForkButton(t *testing.T) {
	for _, forking := range []bool{false, true} {
		s := &Scene{Timeline: Timeline{ID: "fork", Rows: []TimelineRow{{ID: "a", Kind: RowAssistant,
			Message: &Message{ID: "a", Text: "Done.", Meta: true, Copy: true, Fork: true, Forking: forking}}}}}
		click := timelineClicks(NewTheme(), s)
		// The row's 2 padding, the line's 22.75 and the meta row's 6.
		acts := click(image.Pt(32+4+32+12, 24+2+23+6+12))
		want := 1
		if forking {
			want = 0
		}
		if len(acts) != want || want == 1 && acts[0] != (Action{ID: "fork:a", Kind: "click"}) {
			t.Errorf("forking=%v: clicking fork sent %+v", forking, acts)
		}
	}
}
