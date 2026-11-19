package ui

import (
	"image"
	"net/url"
	"strings"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// citeHref is a quoted response's link, as T3's formatAssistantCitationHref
// writes it.
func citeHref(env, thread, message string, q url.Values) string {
	return "t3-citation://v1/" + url.PathEscape(env) + "/" + url.PathEscape(thread) + "/" + url.PathEscape(message) + "?" + q.Encode()
}

// Quote links parse as T3's parseAssistantCitationHref does, with the cases
// of its assistantCitations.test.ts.
func TestParseCitation(t *testing.T) {
	c, ok := parseCitation("t3-citation://v1/a/b/c?text=A+quote+%26+a+newline.%0A&start=0&end=21&prefix=&suffix=+Next.")
	if !ok || c != (citation{env: "a", thread: "b", message: "c", text: "A quote & a newline.\n", suffix: " Next."}) {
		t.Errorf("the legacy link: %+v %v", c, ok)
	}
	href := citeHref("environment/remote", "thread:one", "assistant?one",
		url.Values{"text": {"Use `cache[key]`"}, "start": {"42"}, "end": {"118"}, "prefix": {"Before. "}, "suffix": {" After."}, "comment": {"Why?"}})
	if c, ok := parseCitation(href); !ok || c.env != "environment/remote" || c.thread != "thread:one" || c.message != "assistant?one" || c.comment != "Why?" {
		t.Errorf("a round trip: %+v %v", c, ok)
	}
	for _, bad := range []string{
		"t3-citation://v2/a/b/c?text=quote&start=0&end=5&prefix=&suffix=",
		"t3-citation://v1/a/b/c?text=quote&start=5&end=0&prefix=&suffix=",
		"t3-citation://v1/a/b/c?text=quote&start=0&end=5&prefix=&suffix=&text=other",
		"t3-citation://v1/a/b/c?text=quote&start=0&end=5&prefix=&suffix=&unknown=value",
		"t3-citation://v1/a/b/c?text=quote&start=0&end=5&prefix=&suffix=&comment=one&comment=two",
		"t3-citation://v1/a/b/c?text=++&start=0&end=5&prefix=&suffix=",
		"t3-citation://v1/a/b/c?text=quote&start=0&end=5&prefix=" + strings.Repeat("a", 33) + "&suffix=",
		"t3-citation://v1/a/b?text=quote&start=0&end=5&prefix=&suffix=",
		"https://example.com/?text=quote&start=0&end=5&prefix=&suffix=",
	} {
		if c, ok := parseCitation(bad); ok {
			t.Errorf("%s parsed: %+v", bad, c)
		}
	}
}

// A quoted response in your message is a card on the bubble of black at
// 10%, painted #d4d4d5 on light's #ebebed and #141414 on dark's #1e1e1e,
// 10.4dp under the words before it; its passage is clamped to two 20dp
// lines and padded 6, so a long one is 52dp tall and a short one 32
// (§3.3).
func TestCitationCard(t *testing.T) {
	defer SetDark(false)
	long := strings.Repeat("the passage goes on ", 30)
	href := citeHref("env", "thread", "reply", url.Values{"text": {long}, "start": {"0"}, "end": {"600"}, "prefix": {"before "}, "suffix": {" after"}})
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		want := rgb(0xd4d4d5)
		if d {
			want = rgb(0x141414)
		}
		r := TimelineRow{ID: "u", Kind: RowUser, Message: &Message{ID: "u", Text: "See [Assistant quote](" + href + ") and say why."}}
		img := rowDrawn(t, th, r, image.Pt(736, 240), 2)
		// The card's top padding: 12 of the bubble, the first line's 22.75,
		// then 10.4, near the bubble's right edge.
		wantPixel(t, img, image.Pt(2*(736-12-4), 2*47), want, "the card's fill")
		// Its passage is marked: foreground at 12% on the card.
		mark := over(faded(P.Foreground, 0x1f), want)
		found := false
		for y := 2 * 51; y < 2*(51+20) && !found; y++ {
			for x := 2 * 150; x < 2*724 && !found; x++ {
				c := img.RGBAAt(x, y)
				found = near(c.R, mark.R, 2) && near(c.G, mark.G, 2) && near(c.B, mark.B, 2)
			}
		}
		if !found {
			t.Errorf("dark=%v: no pixel of the card's first line is the mark's #%02x%02x%02x", d, mark.R, mark.G, mark.B)
		}
	}
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Constraints{Max: image.Pt(2*560, 2*400)}, Now: time.Now()}
	th := NewTheme()
	st := mdStyle{color: P.Foreground, on: P.MessageSurface, user: true}
	for _, c := range []struct {
		text string
		want int
	}{{long, 52}, {"a short passage", 32}} {
		cite := citation{env: "env", thread: "thread", message: "reply", text: c.text}
		if h := th.citeCard(gtx, &cite, st); h != 2*c.want {
			t.Errorf("%.20q: %.1fdp tall, want %d", c.text, float32(h)/2, c.want)
		}
	}
}

// A row the timeline is told to reveal comes in 24dp under the canvas's
// top, the end let go: here a fold, whose rule is 35dp down its row.
func TestTimelineReveal(t *testing.T) {
	th := NewTheme()
	s := &Scene{Timeline: Timeline{ID: "reveal"}}
	for i := 0; i < 30; i++ {
		id := strings.Repeat("u", i+1)
		r := TimelineRow{ID: id, Kind: RowUser, Message: &Message{ID: id, Text: "Hello"}}
		if i == 5 {
			r = TimelineRow{ID: "fold", Kind: RowTurnFold, Group: "run", Label: "Worked for 2m 14s"}
		}
		s.Timeline.Rows = append(s.Timeline.Rows, r)
	}
	size := image.Pt(800, 600)
	draw := func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
		th.timeline(gtx, s, 32, 736, 0)
		return D{Size: size}
	}
	drawn(t, size, 1, draw)
	s.Timeline.Reveal = "fold"
	img := drawn(t, size, 1, draw)
	wantPixel(t, img, image.Pt(32+400, 24+35), over(faded(P.Border, 0x99), P.Background), "the revealed fold's rule")
	wantPixel(t, img, image.Pt(32+400, 24+33), P.Background, "over the rule")
	// It stays put on the next frame.
	img = drawn(t, size, 1, draw)
	wantPixel(t, img, image.Pt(32+400, 24+35), over(faded(P.Border, 0x99), P.Background), "the fold's rule a frame later")
}

// Clicking the card goes to its reply: "cite:<environment>/<thread>/<message>",
// each part escaped.
func TestCitationCardClick(t *testing.T) {
	href := citeHref("env/remote", "thread", "reply", url.Values{"text": {"the passage"}, "start": {"0"}, "end": {"11"}, "prefix": {""}, "suffix": {""}})
	s := &Scene{Timeline: Timeline{ID: "cite", Rows: []TimelineRow{{ID: "u", Kind: RowUser, Message: &Message{ID: "u", Text: "[Assistant quote](" + href + ")"}}}}}
	click := timelineClicks(NewTheme(), s)
	// The bubble's right edge is the lane's, 32+736; its card 12dp in.
	acts := click(image.Pt(32+736-20, 24+12+16))
	if len(acts) != 1 || acts[0] != (Action{ID: "cite:env%2Fremote/thread/reply", Kind: "click"}) {
		t.Fatalf("clicking the card sent %+v", acts)
	}
}
