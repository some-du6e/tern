package ui

import (
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

// mdDrawn draws src in style st w dp wide at scale, on the background.
func mdDrawn(t *testing.T, src string, st mdStyle, w, h int, scale float32) *image.RGBA {
	t.Helper()
	th := NewTheme()
	return drawn(t, image.Pt(w, h), scale, func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		return th.markdown(gtx, src, st, nil)
	})
}

// inked reports whether c is a text pixel on the light canvas: darker than
// the background by more than antialiasing's faintest step.
func inked(c color.RGBA) bool { return int(c.R)+int(c.G)+int(c.B) < 3*0xe0 }

// firstInk is the leftmost inked column in rows y0..y1 from x0 on, or -1.
func firstInk(img *image.RGBA, x0, y0, y1 int) int {
	for x := x0; x < img.Bounds().Dx(); x++ {
		for y := y0; y < y1; y++ {
			if inked(img.RGBAAt(x, y)) {
				return x
			}
		}
	}
	return -1
}

// A bullet is Chrome's outside disc, 4.5dp, ending 11.5dp before its
// item's text (index.css ul { padding-inline-start: 1.25rem }).
func TestBulletPosition(t *testing.T) {
	img := mdDrawn(t, "- item", mdBody(), 300, 30, 2)
	// At 2x the disc spans 8..17px across and 20..29px down.
	wantPixel(t, img, image.Pt(12, 24), mdBody().color, "the disc's middle")
	wantPixel(t, img, image.Pt(20, 24), P.Background, "between the disc and the text")
}

// A task's checkbox sits in the gutter and keeps 0.35em before the text,
// so a task's first line starts 2.1dp before a bullet item's text does
// (index.css li.task-list-item input[type=checkbox]).
func TestTaskTextIndent(t *testing.T) {
	plain := mdDrawn(t, "- Item", mdBody(), 300, 30, 4)
	task := mdDrawn(t, "- [ ] Item", mdBody(), 300, 30, 4)
	// Past the 13dp checkbox, and past the bullet.
	p, k := firstInk(plain, 4*14, 4*4, 4*20), firstInk(task, 4*14, 4*4, 4*20)
	if p < 0 || k < 0 || p-k < 7 || p-k > 10 {
		t.Fatalf("the text starts at %dpx in a task and %dpx after a bullet, want 8.4px (2.1dp) apart", k, p)
	}
}

// Code in a link is --contrast-foreground on its 10% box, not the link's
// colour (index.css .chat-markdown :not(pre) > code); only the link's
// favicon before it, 3.5 + 14 + 2.8dp, is.
func TestCodeInLink(t *testing.T) {
	img := mdDrawn(t, "[`io.Writer`](https://pkg.go.dev/io#Writer)", mdBody(), 300, 30, 2)
	link := rgb(0x1447e6)
	box := false
	for y := 0; y < 2*23; y++ {
		for x := 2 * 21; x < 2*300; x++ {
			c := img.RGBAAt(x, y)
			if near(c.R, link.R, 24) && near(c.G, link.G, 24) && near(c.B, link.B, 24) {
				t.Fatalf("a pixel at %d,%d is the link's colour", x, y)
			}
			box = box || near(c.R, 0xe5, 2) && near(c.G, 0xe5, 2) && near(c.B, 0xe5, 2)
		}
	}
	if !box {
		t.Error("no pixel of the code's #e5e5e5 box")
	}
}

// An external link's favicon is 14dp, 3.5dp after the line's start and
// 2.8dp before its words, in the link's colour: GitHub's filled mark,
// whose middle is the cat's face, for its links, and the globe, whose
// equator crosses its middle, for others (§3.5, MarkdownExternalLink.tsx).
func TestLinkFavicon(t *testing.T) {
	link := rgb(0x1447e6)
	isLink := func(c color.RGBA) bool { return near(c.R, link.R, 40) && near(c.G, link.G, 40) && near(c.B, link.B, 40) }
	for _, c := range []struct {
		src    string
		middle bool // the icon's middle is the link's colour
	}{{"[the repo](https://github.com/acme/repo)", false}, {"[the docs](https://example.com/docs)", true}} {
		img := mdDrawn(t, c.src, mdBody(), 300, 30, 2)
		// The 22.75 line centres the icon: its middle is 11.4dp down, 10.5dp in.
		if got := isLink(img.RGBAAt(21, 23)); got != c.middle {
			t.Errorf("%s: the icon's middle is the link's colour: %v, want %v", c.src, got, c.middle)
		}
		ink := 0
		for y := 0; y < 2*23; y++ {
			for x := 0; x < 2*21; x++ {
				px := img.RGBAAt(x, y)
				switch {
				case x < 2*3 || x >= 2*18 && x < 2*20:
					if inked(px) {
						t.Fatalf("%s: ink at %d,%d, outside the icon and before the words", c.src, x, y)
					}
				case isLink(px):
					ink++
				}
			}
		}
		if ink < 40 {
			t.Errorf("%s: %d pixels of the icon are the link's colour", c.src, ink)
		}
	}
}

// A column written ---: is right-aligned: its text ends at the table's
// edge, which its outer cells keep flush (GFM's align, tableCellAlignToStyle).
func TestTableColumnAlignment(t *testing.T) {
	img := mdDrawn(t, "| L | C | R |\n|:--|:-:|--:|\n| a | b | c |", mdBody(), 600, 120, 1)
	// The body row: 8.8 + 22.75 + 8.8 of head and its rule, then 7.2 down.
	right := -1
	for x := 599; x >= 0 && right < 0; x-- {
		for y := 41 + 1 + 7; y < 41+1+7+23; y++ {
			if inked(img.RGBAAt(x, y)) {
				right = x
				break
			}
		}
	}
	if right < 590 {
		t.Fatalf("the right column's text ends at x %d, want at the table's right edge", right)
	}
}

// A user's list keeps the numbers it was written with where they break
// the sequence, so 3 then 5 shows 5, not 4; a reply numbers on from its
// first as CommonMark does (markdownOrderedListNumbers.ts:22-41).
func TestUserListNumbers(t *testing.T) {
	user := mdStyle{color: P.Foreground, on: P.Background, user: true}
	same := func(a, b *image.RGBA) bool {
		for y := 0; y < a.Bounds().Dy(); y++ {
			for x := 0; x < 2*20; x++ { // the markers' gutter
				if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
					return false
				}
			}
		}
		return true
	}
	if same(mdDrawn(t, "3. three\n5. five", user, 300, 60, 2), mdDrawn(t, "3. three\n4. five", user, 300, 60, 2)) {
		t.Error("a user's 5. shows as 4.")
	}
	if !same(mdDrawn(t, "3. three\n5. five", mdBody(), 300, 60, 2), mdDrawn(t, "3. three\n4. five", mdBody(), 300, 60, 2)) {
		t.Error("a reply's list doesn't count on from its first number")
	}
}

// Claude's "★ Insight" callout is a quote with an Insight title in --info,
// its rule at full strength, and the reply keeps its line breaks
// (markdownPipeline.ts:739-818, ChatMarkdown.tsx:354-359).
func TestInsightCallout(t *testing.T) {
	src := "★ Insight ─────────────────────────────\nLine one\nLine two\n─────────────────────────────────"
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(600, 400)}}
	// The title's line, 10.4dp, then two lines.
	if h := th.markdown(gtx, src, mdBody(), nil).Size.Y; h != 23+10+46 {
		t.Errorf("the callout is %ddp tall, want 79", h)
	}
	img := mdDrawn(t, src, mdBody(), 600, 100, 1)
	wantPixel(t, img, image.Pt(1, 40), P.Info, "the callout's rule")
}

// A streaming reply is parsed block by block, so it looks the same as the
// whole parsed at once, and the blocks it streamed through don't pile up
// (ChatMarkdown.tsx:1821-1827).
func TestStreamingBlocks(t *testing.T) {
	text := "## Plan\n\nFirst, the **retry** loop.\n\n- one\n- two\n\n```go\nfunc main() {}\n```\n\nThen the tests"
	done := mdBody()
	done.owner = "m"
	live := done
	live.streaming = true
	a, b := mdDrawn(t, text, done, 400, 300, 1), mdDrawn(t, text, live, 400, 300, 1)
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				t.Fatalf("the streaming reply differs at %d,%d", x, y)
			}
		}
	}
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(400, 300)}}
	for i := 0; i < 50; i++ {
		ops.Reset()
		th.markdown(gtx, text+strings.Repeat(" more", i), live, nil)
	}
	if n := len(th.tl.docs); n > 8 {
		t.Errorf("%d parsed docs kept for one streaming reply", n)
	}
}

// A code block's wrap toggle is kept by the block's place in its message,
// so it survives the message being parsed again, here for the dark theme
// (MarkdownCodeBlock.tsx:163).
func TestCodeBlockWrapSurvivesReparse(t *testing.T) {
	defer SetDark(false)
	src := "```go\n" + strings.Repeat("retry ", 60) + "\n```"
	st := mdBody()
	st.owner = "m"
	th := NewTheme()
	var r gioinput.Router
	var ops op.Ops
	frame := func() int {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400)), Source: r.Source(), Now: time.Now()}
		h := th.markdown(gtx, src, st, nil).Size.Y
		r.Frame(&ops)
		th.Actions()
		return h
	}
	wrapped := frame()
	wrap := f32.Pt(600-1-6-24-2-12, 1+6+12)
	r.Queue(pointer.Event{Kind: pointer.Move, Position: wrap, Source: pointer.Mouse})
	frame()
	r.Queue(pointer.Event{Kind: pointer.Press, Position: wrap, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: wrap, Source: pointer.Mouse})
	frame()
	unwrapped := frame()
	if unwrapped >= wrapped {
		t.Fatalf("the toggle didn't unwrap: %d then %d", wrapped, unwrapped)
	}
	SetDark(true)
	if h := frame(); h != unwrapped {
		t.Errorf("parsed again, the block is %ddp tall, want %d unwrapped", h, unwrapped)
	}
}
