package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"testing"
	"time"

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

// firstRow is the first y in [y0, y1) at column x painted c, or -1.
func firstRow(img *image.RGBA, x, y0, y1 int, c color.NRGBA) int {
	for y := y0; y < y1; y++ {
		if near(img.RGBAAt(x, y), c) {
			return y
		}
	}
	return -1
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

// countAll is how many pixels of img are c.
func countAll(img *image.RGBA, c color.NRGBA) int {
	n := 0
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		n += count(img, y, img.Rect.Min.X, img.Rect.Max.X, c)
	}
	return n
}

// The card is T3's at 1440x900 (docs/ui/07-composer.md §3.1, §3.2): the
// glass, #ffffff / #0e0e0e over the canvas, inside an 8% outline painted
// #ebebeb / #1a1a1a, with dark's 3% highlight under its top edge; 111.5
// tall empty; the round send button 32 wide at x 695 wearing the nightly
// stage art, at 64% while there's nothing to send.
func TestComposerCard(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Placeholder: "Ask anything. @ files, $ skills, / commands", Provider: "Claude", Model: "Opus 5.5", Effort: "High"}
		img := drawComposer(t, th, s, 200, 0)
		glassWant, outlineWant := rgb(0xffffff), rgb(0xebebeb)
		if d {
			glassWant, outlineWant = rgb(0x0e0e0e), rgb(0x1a1a1a)
		}
		wantPixel(t, img, image.Pt(500, 50), glassWant, "glass")
		wantPixel(t, img, image.Pt(300, 0), outlineWant, "outline")
		if bottom := lastRow(img, 300, 100, 130, outlineWant); bottom < 110 || bottom > 112 {
			t.Errorf("dark %v: the card ends at %d, want 111.5 tall", d, bottom+1)
		}
		if d {
			wantPixel(t, img, image.Pt(300, 1), over(alpha(rgb(0xffffff), 0x08), glass()), "highlight")
		}
		// Left of the arrow, the art's dark sky shows through at 64%.
		if c := img.RGBAAt(700, 80); !d && (c.R < 90 || c.R > 210 || int(c.B)-int(c.R) > 40) {
			t.Errorf("disabled send at (700, 80) is %v, want the art at 64%% over the glass", c)
		}
		wantPixel(t, img, image.Pt(693, 87), glass(), "the gap before send")
	}
}

// With something to send the art is whole: its navy sky, not
// --message-action; while a run works and nothing's typed it's Stop,
// --destructive at 90%.
func TestComposerSendStates(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "hi", Model: "Opus 5.5"}
	sky := func(img *image.RGBA, what string) {
		t.Helper()
		if c := img.RGBAAt(700, 80); c.R > 45 || c.G > 45 || c.B < 40 || c.B > 100 {
			t.Errorf("%s: (700, 80) is %v, want the nightly sky", what, c)
		}
	}
	sky(drawComposer(t, th, s, 200, 0), "send")
	s.State = ThreadRunning
	sky(drawComposer(t, th, s, 200, 0), "steer")
	s.Composer = ""
	wantPixel(t, drawComposer(t, th, s, 200, 0), image.Pt(699, 87), over(alpha(P.Destructive, 0xe6), glass()), "stop")
	// A stop on its way: Stop stays, off at 64%.
	s.Stop = StopPending
	if c := drawComposer(t, th, s, 200, 0).RGBAAt(699, 87); near(c, over(alpha(P.Destructive, 0xe6), glass())) {
		t.Errorf("stopping: Stop is %v, still at full strength", c)
	}
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

// The model trigger is one target, filled with --accent while its picker
// is open, ending 8dp before send. The permissions chip, shown when the
// mode isn't the default, comes after it (ComposerControlsBar.tsx), 2dp
// on, against send; it fills while its menu is open, and the trigger
// doesn't.
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
	s.Access = &Permissions{Modes: RuntimeModes}
	img = drawComposer(t, th, s, 200, 0)
	wantPixel(t, img, image.Pt(685, 97), accent, "open chip")
	// Along a row under its label, the chip's fill runs back to its start.
	left := 682
	for left > 45 && near(img.RGBAAt(left-1, 97), accent) {
		left--
	}
	if w := 687 - left; w < 60 || w > 200 {
		t.Errorf("open chip: %d wide, want its icon and label", w)
	}
	if n := count(img, 97, 45, left-2, accent); n != 0 {
		t.Errorf("the trigger is filled too: %d pixels", n)
	}
	if l, r := ink(img, 80, 95, 45, left-2); l < left-170 || r < left-20 {
		t.Errorf("the trigger's ink spans x %d..%d, want it just before the chip at %d", l, r, left)
	}
}

// ink is the leftmost and rightmost x in [x0, x1) of rows [y0, y1) that
// text darkened (light theme), or -1, -1.
func ink(img *image.RGBA, y0, y1, x0, x1 int) (left, right int) {
	left, right = -1, -1
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if c := img.RGBAAt(x, y); c.R < 0x90 && c.G < 0x90 {
				if left < 0 || x < left {
					left = x
				}
				right = max(right, x)
			}
		}
	}
	return left, right
}

// inkRows is the first and last row of [y0, y1) with ink in [x0, x1), or
// -1, -1.
func inkRows(img *image.RGBA, y0, y1, x0, x1 int) (first, last int) {
	first, last = -1, -1
	for y := y0; y < y1; y++ {
		if l, _ := ink(img, y, y+1, x0, x1); l >= 0 {
			if first < 0 {
				first = y
			}
			last = y
		}
	}
	return first, last
}

// The permissions menu is T3's section (§3.8): over the chip, a 28dp
// segmented row of the modes on --input at 40%, the thread's segment on
// --input at 72%, 67dp of 24 for four modes.
func TestPermissionsMenu(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "hi", Model: "Opus 5.5", Runtime: "approval-required", RuntimeChip: true,
		Access: &Permissions{Modes: RuntimeModes, Sel: 0}}
	img := drawComposer(t, th, s, 400, 250)
	group := over(alpha(P.Input, uint8(int(P.Input.A)*40/100)), P.Popover)
	checked := over(alpha(P.Input, uint8(int(P.Input.A)*72/100)), group)
	if n := countAll(img, checked); n < 1200 || n > 1900 {
		t.Errorf("the checked segment has %d pixels, want one 67x24 segment", n)
	}
	if n := countAll(img, group); n < 3000 {
		t.Errorf("the segmented row has %d pixels of its fill, want the 284x28 row", n)
	}
}

// Queued messages are the timeline's footer, not the composer's: dashed
// bubbles at the lane's end with Send now, edit and remove under them,
// also at the end (QueuedRunsControl.tsx).
func TestQueue(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Queue: []QueuedMessage{{Text: "then add a tray icon"}}}
	if img := drawComposer(t, th, s, 300, 0); lastRow(img, 300, 0, 300, over(composerOutline(), glass())) > 113 {
		t.Error("the composer still draws the queue")
	}
	img := drawn(t, image.Pt(736, 300), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 736, 300), 0)
		return th.messages(gtx, s, false)
	})
	first, last := inkRows(img, 0, 300, 0, 736)
	if first < 0 {
		t.Fatal("no queue in the timeline")
	}
	if l, r := ink(img, first, first+20, 0, 736); l < 500 || r > 736 {
		t.Errorf("the bubble's ink spans x %d..%d, want it at the end", l, r)
	}
	// The actions' row: Send now (86), edit (24) and remove (24), 2 apart.
	if l, r := ink(img, last-18, last+1, 0, 736); l < 560 || r < 700 {
		t.Errorf("the actions span x %d..%d, want them at the end", l, r)
	}
}

// A notice is an attached card 22dp in from each side (692 of 736), its
// visible 34dp above the card (§3.9), casting the composer's shadow; a
// thread's error isn't one of the composer's banners.
func TestNoticeBanner(t *testing.T) {
	th := NewTheme()
	img := drawComposer(t, th, &Scene{Notice: "Loading earlier turns…", Model: "Opus 5.5"}, 200, 0)
	g := glass()
	wantPixel(t, img, image.Pt(650, 15), g, "the notice's glass")
	wantPixel(t, img, image.Pt(22, 15), over(composerOutline(), g), "its edge")
	if c := img.RGBAAt(21, 15); near(c, P.Background) {
		t.Error("no shadow beside the notice")
	}
	wantPixel(t, img, image.Pt(300, 34), over(composerOutline(), g), "the composer's top edge")
	if img := drawComposer(t, th, &Scene{Error: "Couldn't upload the picture", Model: "Opus 5.5"}, 200, 0); lastRow(img, 300, 0, 200, over(composerOutline(), g)) > 113 {
		t.Error("the error is drawn over the composer")
	}
}

// A thread's error is T3's ThreadErrorBanner in the banner lane over the
// conversation (§2.9; ChatBannerLane.tsx): 12dp from the canvas top,
// centred in the lane, rounded 12, edged in --error at 32% on alert glass.
func TestThreadError(t *testing.T) {
	th := NewTheme()
	s := &Scene{Title: "A thread", Error: "the server said 413", Model: "Opus 5.5", Items: []Item{{Kind: ItemAssistant, Text: "hi"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	surface := over(alpha(P.Error, 0x0a), over(alpha(P.Background, 0xcc), P.Background))
	edge := over(alpha(P.Error, 0x52), surface)
	top := firstRow(img, 848, 52, 200, edge)
	if top < 52+12-1 || top > 52+12+1 {
		t.Fatalf("the banner's top edge is at %d, want 12dp under the canvas top (64)", top)
	}
	wantPixel(t, img, image.Pt(848, top+3), surface, "the banner's fill")
	l, r := 848, 848
	for l > 300 && !near(img.RGBAAt(l, top+20), edge) {
		l--
	}
	for r < 1400 && !near(img.RGBAAt(r, top+20), edge) {
		r++
	}
	if mid := (l + r) / 2; mid < 840 || mid > 856 {
		t.Errorf("the banner spans x %d..%d, want it centred in the lane on 848", l, r)
	}
}

// The agent's question is an attached drawer (§3.12): a 36dp header row,
// the question, then its options, 32dp each, the picked one's radio
// filled with --primary around a white dot.
func TestQuestionDrawer(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Questions: []Question{{Header: "Session", Text: "Which one?", Options: []string{"A", "B"}, Chosen: []bool{false, true}}}}
	img := drawComposer(t, th, s, 400, 0)
	g := glass()
	wantPixel(t, img, image.Pt(56, 119), P.PrimaryForeground, "the picked radio's dot")
	wantPixel(t, img, image.Pt(61, 119), P.Primary, "the picked radio")
	wantPixel(t, img, image.Pt(56, 87), P.Background, "the other radio")
	wantPixel(t, img, image.Pt(300, 143), over(composerOutline(), g), "the composer's top edge")
	// The digit's Kbd ends the row, 20dp in a 24dp column.
	wantPixel(t, img, image.Pt(674, 79), over(alpha(P.Foreground, 0x12), g), "the first option's key")
	wantPixel(t, img, image.Pt(668, 79), g, "before the key")
	// The header: its text 26dp in, the chevron at the end.
	if l, r := ink(img, 10, 34, 22, 714); l < 44 || l > 52 || r < 670 || r > 710 {
		t.Errorf("the header's ink spans x %d..%d, want the header from 48 and the chevron by the end", l, r)
	}
	// With a question open the footer is pt-2, the 28dp row and pb-2: the
	// card is 115.5 tall (ChatComposer.tsx:3627).
	outline := over(composerOutline(), g)
	if bottom := lastRow(img, 300, 150, 400, outline); bottom-143 < 114 || bottom-143 > 117 {
		t.Errorf("the card runs from 143 to %d, want 115.5 tall", bottom)
	}
}

// A long question scrolls in its drawer, at most min(384, 40% of the
// window) tall, and the card stays in the canvas.
func TestQuestionScrolls(t *testing.T) {
	th := NewTheme()
	th.in.window = image.Pt(1440, 900)
	text := ""
	for i := 0; i < 40; i++ {
		text += "A long question that goes on. "
	}
	s := &Scene{Model: "Opus 5.5", Questions: []Question{{Header: "Session", Text: text, Options: []string{"A", "B", "C", "D", "E", "F", "G", "H"}}}}
	var d D
	drawn(t, image.Pt(736, 900), 1, func(gtx C) D {
		d = th.composerArea(gtx, s, false)
		return d
	})
	// The header (36 + 4), at most 360 of body, 4 under, then the card
	// over the tuck: 116 with a question.
	if d.Size.Y > 4+36+360+4+116+2 {
		t.Errorf("the composer is %d tall with a long question, want it capped", d.Size.Y)
	}
}

// The stash's tab sits on the card's top edge at its end, 22dp in, as
// wide as "Stash" and its count (§3.10); its list rises out of the card
// with a 24dp header, its count and chevron at the end, and a 24dp row
// per prompt, the highlighted one filled with --accent.
func TestStash(t *testing.T) {
	th := NewTheme()
	g := glass()
	s := &Scene{Model: "Opus 5.5", Stash: []Stashed{{Snippet: "first idea", Age: "4m ago"}, {Snippet: "second", Age: "1h ago"}}}
	img := drawComposer(t, th, s, 200, 0)
	wantPixel(t, img, image.Pt(713, 15), over(composerOutline(), g), "the tab's end")
	wantPixel(t, img, image.Pt(300, 15), P.Background, "no column of notices")
	wantPixel(t, img, image.Pt(300, 34), over(composerOutline(), g), "the composer's top edge")
	// From its bookmark, 6dp into its 24dp icon column, to its count.
	if l, r := ink(img, 5, 29, 0, 736); l < 625 || l > 640 || r > 712 {
		t.Errorf("the tab's ink spans x %d..%d, want the bookmark from about 630 and the count by 712", l, r)
	}
	s.StashOpen, s.StashSel = true, 1
	img = drawComposer(t, th, s, 320, 150)
	wantPixel(t, img, image.Pt(400, 168), over(P.Accent, g), "the highlighted row")
	wantPixel(t, img, image.Pt(400, 143), g, "the row above it")
	wantPixel(t, img, image.Pt(22, 120), over(alpha(P.Foreground, 0x14), g), "the list's edge, portaled")
	if l, _ := ink(img, 135, 151, 22, 50); l < 30 || l > 40 {
		t.Errorf("the first row's ink starts at x %d, want its file icon at about 32", l)
	}
	// The header row: "Stash" by the bookmark, its count and chevron at the
	// drawer's end (the Row's [24][1fr][auto] grid).
	if _, r := ink(img, 108, 130, 22, 714); r < 695 {
		t.Errorf("the header's ink ends at x %d, want the chevron by the drawer's end (710)", r)
	}
}

// An empty stash's list says how to stash (ComposerStashMenu.tsx).
func TestStashEmpty(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", StashOpen: true}
	img := drawComposer(t, th, s, 320, 200)
	if l, r := ink(img, 150, 199, 22, 714); l < 0 || r-l < 200 {
		t.Errorf("the empty list's ink spans x %d..%d, want its line", l, r)
	}
}

// A long stash scrolls in its list, min(384, 40% of the window), rather
// than spill over the card.
func TestStashScrolls(t *testing.T) {
	th := NewTheme()
	th.in.window = image.Pt(1440, 900)
	s := &Scene{Model: "Opus 5.5", StashOpen: true}
	for i := 0; i < 20; i++ {
		s.Stash = append(s.Stash, Stashed{Snippet: "idea " + strconv.Itoa(i), Age: "4m ago"})
	}
	img := drawComposer(t, th, s, 900, 600)
	g := glass()
	top := firstRow(img, 22, 0, 600, over(alpha(P.Foreground, 0x14), g))
	if top < 600+tuck-(4+24+360+4+tuck)-2 {
		t.Errorf("the list's edge starts at %d, want it capped at 360 of rows", top)
	}
	wantPixel(t, img, image.Pt(400, 650), g, "the card, clear of the list")
}

// Stashed pictures overlap 6dp, each later one on top (-space-x-1.5).
func TestStashThumbs(t *testing.T) {
	th := NewTheme()
	tile := func(c color.RGBA) Thumb {
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				img.SetRGBA(x, y, c)
			}
		}
		var b bytes.Buffer
		if err := png.Encode(&b, img); err != nil {
			t.Fatal(err)
		}
		return NewThumb(b.Bytes())
	}
	red, blue := color.RGBA{0xff, 0, 0, 0xff}, color.RGBA{0, 0, 0xff, 0xff}
	s := &Scene{Model: "Opus 5.5", StashOpen: true, Stash: []Stashed{{Snippet: "two pictures", Thumbs: []Thumb{tile(red), tile(blue)}, Age: "4m ago"}}}
	img := drawComposer(t, th, s, 320, 200)
	// Find the red tile's right end; past it, under blue's left 6dp, blue wins.
	for y := 150; y < 200; y++ {
		for x := 400; x < 714; x++ {
			if c := img.RGBAAt(x, y); c.R > 0xe0 && c.B < 0x20 {
				// Red is the first; 10dp on from its start the blue one begins on top.
				if b := img.RGBAAt(x+12, y); b.B < 0xc0 || b.R > 0x40 {
					t.Fatalf("at (%d, %d) the later picture is %v, want it over the first", x+12, y, b)
				}
				return
			}
		}
	}
	t.Fatal("no stashed pictures drawn")
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
	wantPixel(t, img, image.Pt(22, 100), over(alpha(P.Foreground, 0x14), g), "the menu's edge, portaled")
	wantPixel(t, img, image.Pt(300, 200), g, "the menu over the composer's edge")
	wantPixel(t, img, image.Pt(18, 200), over(composerOutline(), g), "the composer's top edge")
}

// A long @ menu is at most 288dp with its padding and scrolls; rows past
// that don't paint over the card, and the highlighted row scrolls into
// view (ComposerCommandMenu.tsx: max-h-72, scrollIntoView).
func TestCommandMenuScrolls(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "@a", Model: "Opus 5.5", SuggestSel: 19}
	for i := 0; i < 20; i++ {
		s.Suggest = append(s.Suggest, Suggestion{Icon: "file", Label: "file" + strconv.Itoa(i) + ".go", Desc: "src"})
	}
	img := drawComposer(t, th, s, 700, 400)
	g := glass()
	accent := over(P.Accent, g)
	if top := firstRow(img, 22, 0, 400, over(alpha(P.Foreground, 0x14), g)); top < 400+tuck-(288+tuck)-1 {
		t.Errorf("the menu's edge starts at %d, want it capped at 288 (from %d)", top, 400+tuck-(288+tuck))
	}
	// The last row, highlighted, is in view at the list's foot.
	if y := firstRow(img, 600, 100, 400, accent); y < 300 || y > 400 {
		t.Errorf("the highlighted last row starts at %d, want it scrolled into view near the list's foot", y)
	}
	// Nothing of the menu inside the card.
	for _, y := range []int{420, 450, 480} {
		wantPixel(t, img, image.Pt(300, y), g, "the card under the menu")
	}
}

// A new thread's hero at 1440x900 (§3.1, §3.17): the stack's top at 394,
// the heading's 36dp line at 310 with its 30px type's baseline at 339 and
// its caps from about 317, the subline's 24dp row 8 under it; both centred
// in the lane, and the strip's permissions at its end.
func TestHero(t *testing.T) {
	th := NewTheme()
	s := &Scene{Hero: "tern", CanScratch: true, Model: "Opus 5.5", Title: "New thread"}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	// The lane is 480..1216 (no details card for a draft).
	first, last := inkRows(img, 290, 352, 500, 1200)
	if first < 314 || first > 319 || last < 336 || last > 340 {
		t.Errorf("heading ink in rows %d..%d, want caps from about 317 and the baseline at 339", first, last)
	}
	if f, l := inkRows(img, 352, 380, 500, 1200); f < 356 || l >= 378 {
		t.Errorf("subline ink in rows %d..%d, want inside 354..378", f, l)
	}
	wantPixel(t, img, image.Pt(980, 394+50), glass(), "the card")
	wantPixel(t, img, image.Pt(780, 394), over(composerOutline(), glass()), "the card's top edge")
	wantPixel(t, img, image.Pt(580, 394+112+20), stripGlass(), "the strip")
	for _, band := range [][2]int{{310, 346}, {354, 378}} {
		if l, r := ink(img, band[0], band[1], 480, 1216); l < 0 || (l+r)/2 < 844 || (l+r)/2 > 852 {
			t.Errorf("ink in rows %d..%d spans x %d..%d, want it centred on 848", band[0], band[1], l, r)
		}
	}
	if l, r := ink(img, 394+112+8, 394+112+32, 480, 1216); l < 1080 || r > 1208 {
		t.Errorf("the strip's ink spans x %d..%d, want the permissions at its end", l, r)
	}
}

// With prompts stashed, a new thread shows the stash's tab on its card,
// and the heading's wrapper loses its 16dp so the tab sits right under it.
func TestHeroStash(t *testing.T) {
	th := NewTheme()
	s := &Scene{Hero: "tern", CanScratch: true, Model: "Opus 5.5", Title: "New thread", Stash: []Stashed{{Snippet: "an idea"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
	outline := over(composerOutline(), glass())
	tab := firstRow(img, 1190, 200, 600, outline)
	if tab < 0 {
		t.Fatal("no stash tab on the new thread")
	}
	if _, sub := inkRows(img, tab-60, tab, 500, 1200); tab-sub < 2 || tab-sub > 10 {
		t.Errorf("the subline ends at %d and the tab starts at %d, want them flush", sub, tab)
	}
}

// With a question open under a 780dp composer, the answer actions go
// compact (composerFooterLayout.ts): "Submit" padded 12, 70dp or so,
// Previous a 28dp icon 6 before it, Stop 28 in the row.
func TestCompactAnswers(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", State: ThreadRunning, Question: 1, Questions: []Question{
		{Header: "One", Text: "First?", Options: []string{"A"}, Chosen: []bool{true}},
		{Header: "Two", Text: "Second?", Options: []string{"B"}, Chosen: []bool{true}},
	}}
	img := drawComposer(t, th, s, 400, 0)
	g := glass()
	pill := P.MessageAction
	stop := over(alpha(P.Destructive, 0xe6), g)
	var y int
	for y = 150; y < 400 && count(img, y, 600, 736, pill) == 0; y++ {
	}
	y += 14 // the pill's middle row
	if n := count(img, y, 400, 736, pill); n < 60 || n > 85 {
		t.Errorf("the Submit pill is %d wide, want T3's compact 70.86", n)
	}
	if n := count(img, y, 400, 736, stop); n < 24 || n > 29 {
		t.Errorf("Stop is %d wide in the row, want 28", n)
	}
}

// The provider's suggestion is the placeholder, with its outline Tab key
// 8dp after it (ChatComposer.tsx:3571).
func TestSuggestion(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Placeholder: "go ahead set it up", Suggestion: "go ahead set it up"}
	img := drawComposer(t, th, s, 200, 0)
	edge := over(alpha(P.Foreground, 0x26), glass())
	n := 0
	for y := 10; y < 50; y++ {
		n += count(img, y, 17, 720, edge)
	}
	if n < 60 {
		t.Errorf("%d pixels of the key's border after the suggestion, want its outline", n)
	}
}

// A picture's chip names its file and size, and is as wide as that: the
// text after it starts a word's gap on (ImageChipButton).
func TestChipWidth(t *testing.T) {
	th := NewTheme()
	thumb := Thumb{N: 1, Name: "shot.png", Size: 2048}
	s := &Scene{Model: "Opus 5.5", Composer: "see " + th.ImageChip(1, "shot.png", 2048) + " now", Thumbs: []Thumb{thumb}}
	img := drawComposer(t, th, s, 200, 0)
	chip := P.ContextChipImage
	end, start := -1, -1
	for y := 17; y < 45; y++ {
		for x := 17; x < 720; x++ {
			if near(img.RGBAAt(x, y), chip) || (img.RGBAAt(x, y).B > img.RGBAAt(x, y).R+40 && img.RGBAAt(x, y).R < 0xb0) {
				end = max(end, x)
			}
		}
	}
	if end < 0 {
		t.Fatal("no chip drawn")
	}
	for y := 17; y < 45; y++ {
		for x := end + 1; x < 720; x++ {
			if c := img.RGBAAt(x, y); c.R < 0x90 && c.G < 0x90 && c.B < 0x90 && (start < 0 || x < start) {
				start = x
			}
		}
	}
	if gap := start - end; start < 0 || gap < 2 || gap > 9 {
		t.Errorf("the text after the chip starts %d after its ink ends, want a word's gap", gap)
	}
}

// The tab pulses after a stash: the row in the foreground, the count in
// --primary.
func TestStashPulse(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Stash: []Stashed{{Snippet: "idea"}}, Stashed: time.Now().Add(-500 * time.Millisecond)}
	img := drawComposer(t, th, s, 200, 0)
	if n := countAll(img, over(P.Primary, glass())); n < 4 {
		t.Errorf("%d pixels of --primary in the tab's count, want the pulse", n)
	}
}
