package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/op"
)

// drawComposer draws a thread's composer the way the canvas does, in a
// 736dp lane on the background, top at y.
func drawComposer(t *testing.T, th *Theme, s *Scene, h, y int) *image.RGBA {
	t.Helper()
	return drawn(t, image.Pt(736, h), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 736, h), 0)
		defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
		return th.composerArea(gtx, s, false)
	})
}

// near reports whether two colours are a step apart at most.
func near(a color.RGBA, b color.NRGBA) bool {
	d := func(x, y uint8) bool { return x-y < 3 || y-x < 3 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

// lastRow is the last y in [y0, y1) at column x painted c, or -1.
func lastRow(img *image.RGBA, x, y0, y1 int, c color.NRGBA) int {
	last := -1
	for y := y0; y < y1; y++ {
		if near(img.RGBAAt(x, y), c) {
			last = y
		}
	}
	return last
}

// count is how many pixels of row y in [x0, x1) are c.
func count(img *image.RGBA, y, x0, x1 int, c color.NRGBA) int {
	n := 0
	for x := x0; x < x1; x++ {
		if near(img.RGBAAt(x, y), c) {
			n++
		}
	}
	return n
}

// The card is T3's at 1440x900 (docs/ui/07-composer.md §3.1, §3.2): the
// glass, #ffffff / #0e0e0e over the canvas, inside an 8% outline painted
// #ebebeb / #1a1a1a, with dark's 3% highlight under its top edge; 111.5
// tall empty; the round send button 32 wide at x 695, greyed while there's
// nothing to send.
func TestComposerCard(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Placeholder: "Ask anything. @ files, $ skills, / commands", Provider: "Claude", Model: "Opus 5.5", Effort: "High"}
		img := drawComposer(t, th, s, 200, 0)
		g := glass()
		want := rgb(0xffffff)
		if d {
			want = rgb(0x0e0e0e)
		}
		wantPixel(t, img, image.Pt(500, 50), want, "glass")
		outline := over(composerOutline(), g)
		wantPixel(t, img, image.Pt(300, 0), outline, "outline")
		if bottom := lastRow(img, 300, 100, 130, outline); bottom < 110 || bottom > 112 {
			t.Errorf("dark %v: the card ends at %d, want 111.5 tall", d, bottom+1)
		}
		if d {
			wantPixel(t, img, image.Pt(300, 1), over(alpha(rgb(0xffffff), 0x08), g), "highlight")
		}
		wantPixel(t, img, image.Pt(699, 87), over(alpha(P.Foreground, 0x1f), g), "disabled send")
		wantPixel(t, img, image.Pt(693, 87), g, "the gap before send")
	}
}

// With something to send the button is --message-action; while a run
// works and nothing's typed it's Stop, --destructive at 90%.
func TestComposerSendStates(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "hi", Model: "Opus 5.5"}
	wantPixel(t, drawComposer(t, th, s, 200, 0), image.Pt(699, 87), P.MessageAction, "send")
	s.State = ThreadRunning
	wantPixel(t, drawComposer(t, th, s, 200, 0), image.Pt(699, 87), P.MessageAction, "steer")
	s.Composer = ""
	wantPixel(t, drawComposer(t, th, s, 200, 0), image.Pt(699, 87), over(alpha(P.Destructive, 0xe6), glass()), "stop")
}

// The editor holds two lines at least and grows a line at a time to
// 208dp with its padding (§3.1: 111.5, 134.25, then capped at 266).
func TestPromptGrowth(t *testing.T) {
	th := NewTheme()
	outline := over(composerOutline(), glass())
	for _, c := range []struct {
		text string
		want int
	}{
		{"one", 112}, {"one\ntwo\nthree", 134},
		{"1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14", 266},
	} {
		img := drawComposer(t, th, &Scene{Composer: c.text, Model: "Opus 5.5"}, 300, 0)
		if got := lastRow(img, 300, 100, 300, outline) + 1; got < c.want-1 || got > c.want+1 {
			t.Errorf("%d lines: the card is %d tall, want %d", len(c.text), got, c.want)
		}
	}
}

// The model trigger stays filled with --accent while its menu is open,
// ending 8dp before send. The permissions chip, shown when the mode isn't
// the default, comes after it (ComposerControlsBar.tsx), 2dp on, against
// send; it fills while its menu is open, and the trigger doesn't.
func TestComposerTriggers(t *testing.T) {
	th := NewTheme()
	g := glass()
	accent := over(P.Accent, g)
	s := &Scene{Composer: "hi", Provider: "Claude", Model: "Opus 5.5", Effort: "High", Picker: &ModelPicker{Tabs: []PickerTab{{}}}}
	img := drawComposer(t, th, s, 200, 0)
	wantPixel(t, img, image.Pt(685, 87), accent, "open trigger")
	wantPixel(t, img, image.Pt(690, 87), g, "the gap after the trigger")

	s.Picker, s.Runtime, s.RuntimeChip = nil, "approval-required", true
	if n := count(drawComposer(t, th, s, 200, 0), 75, 45, 687, accent); n != 0 {
		t.Errorf("closed chip: %d pixels filled", n)
	}
	s.Access = &Traits{}
	img = drawComposer(t, th, s, 200, 0)
	wantPixel(t, img, image.Pt(685, 87), accent, "open chip")
	// Along a row above its label, the chip's fill runs back to its start.
	left := 685
	for left > 45 && near(img.RGBAAt(left-1, 77), accent) {
		left--
	}
	if w := 687 - left; w < 60 {
		t.Errorf("open chip: %d wide, want its icon and label", w)
	}
	if n := count(img, 87, 45, left-2, accent); n != 0 {
		t.Errorf("the trigger is filled too: %d pixels", n)
	}
}

// A notice is an attached card 22dp in from each side (692 of 736), its
// visible 34dp above the card (§3.9); an error's tinted 8% and edged 32%.
func TestNoticeBanner(t *testing.T) {
	th := NewTheme()
	img := drawComposer(t, th, &Scene{Error: "Couldn't upload the picture", Model: "Opus 5.5"}, 200, 0)
	g := glass()
	tint := over(alpha(P.Error, 0x14), g)
	wantPixel(t, img, image.Pt(650, 15), tint, "error tint")
	wantPixel(t, img, image.Pt(22, 15), over(alpha(P.Error, 0x52), tint), "error edge")
	wantPixel(t, img, image.Pt(21, 15), P.Background, "outside the card")
	wantPixel(t, img, image.Pt(300, 34), over(composerOutline(), g), "the composer's top edge")
}

// The agent's question is an attached drawer (§3.12): a 36dp header row,
// the question, then its options, 32dp each, the picked one's radio
// filled with --primary around a white dot.
func TestQuestionDrawer(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Questions: []Question{{Header: "Session", Text: "Which one?", Options: []string{"A", "B"}, Chosen: []bool{false, true}}}}
	img := drawComposer(t, th, s, 400, 0)
	wantPixel(t, img, image.Pt(56, 119), P.PrimaryForeground, "the picked radio's dot")
	wantPixel(t, img, image.Pt(61, 119), P.Primary, "the picked radio")
	wantPixel(t, img, image.Pt(56, 87), P.Background, "the other radio")
	wantPixel(t, img, image.Pt(300, 143), over(composerOutline(), glass()), "the composer's top edge")
}

// The @ menu rises out of the card's top edge, 22dp in, its rows 28dp in
// 8dp of padding, the highlighted one filled with --accent. Its glass runs
// 1dp into the card, over the card's edge.
func TestCommandMenu(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "@comp", Model: "Opus 5.5", SuggestSel: 1, Suggest: []Suggestion{
		{Icon: "folder", Label: "components", Desc: "ui"}, {Icon: "file", Label: "composer.go", Desc: "ui"},
		{Icon: "file", Label: "composerdock.go", Desc: "ui"}, {Icon: "file", Label: "composer_test.go", Desc: "ui"},
	}}
	img := drawComposer(t, th, s, 400, 200)
	g := glass()
	wantPixel(t, img, image.Pt(600, 122), over(P.Accent, g), "the highlighted row")
	wantPixel(t, img, image.Pt(600, 94), g, "the row above it")
	wantPixel(t, img, image.Pt(22, 100), over(composerOutline(), g), "the menu's edge")
	wantPixel(t, img, image.Pt(300, 200), g, "the menu over the composer's edge")
	wantPixel(t, img, image.Pt(18, 200), over(composerOutline(), g), "the composer's top edge")
}

// A new thread's hero (§3.1, §3.17): the composer and its strip centred
// in the canvas (the stack's top at 342 in an 848dp canvas), the 36dp
// question 16dp over them with its 24dp line 8 below it.
func TestHero(t *testing.T) {
	th := NewTheme()
	s := &Scene{Hero: "tern", CanScratch: true, Model: "Opus 5.5"}
	img := drawn(t, image.Pt(736, 848), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 736, 848), 0)
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		return th.hero(gtx, s, false)
	})
	ink := func(y0, y1 int) (first, last int) {
		first, last = -1, -1
		for y := y0; y < y1; y++ {
			for x := 100; x < 636; x++ {
				if c := img.RGBAAt(x, y); c.R < 0x90 {
					if first < 0 {
						first = y
					}
					last = y
					break
				}
			}
		}
		return first, last
	}
	if f, l := ink(240, 300); f < 258 || l >= 294 {
		t.Errorf("heading ink in rows %d..%d, want inside 258..294", f, l)
	}
	if f, l := ink(294, 330); f < 302 || l >= 326 {
		t.Errorf("subline ink in rows %d..%d, want inside 302..326", f, l)
	}
	wantPixel(t, img, image.Pt(500, 342+50), glass(), "the card")
	wantPixel(t, img, image.Pt(300, 342), over(composerOutline(), glass()), "the card's top edge")
	wantPixel(t, img, image.Pt(100, 342+112+20), stripGlass(), "the strip")
}
