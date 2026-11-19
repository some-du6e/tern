package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"testing"
	"time"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
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
	// A stop on its way: Stop stays, off at 64% over the glass as Chrome
	// composites it, #fd868b in light, not the paler #fca8aa Gio's layer
	// opacity makes.
	s.Stop = StopPending
	wantPixel(t, drawComposer(t, th, s, 200, 0), image.Pt(699, 87), rgb(0xfd868b), "stopping")
}

// With nothing to send the button is at 64% over the glass, the way
// Chrome composites it in sRGB (§3.5, measured live): its art faded onto
// the glass and its arrow white at 64%, which on dark's #0e0e0e glass is
// #a8a8a8.
func TestDisabledSend(t *testing.T) {
	defer SetDark(false)
	data, err := artFiles.ReadFile("art/send-night@1x.png")
	if err != nil {
		t.Fatal(err)
	}
	art, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Model: "Opus 5.5"}
		g := glass()
		sky := color.NRGBAModel.Convert(art.At(5, 9)).(color.NRGBA)
		wantPixel(t, drawComposer(t, th, s, 200, 0), image.Pt(700, 80), faded(sky, g, 0xa3), "the art at 64%")
		// The arrow's 1.8dp stroke covers whole pixels at 2x.
		img := drawn(t, image.Pt(736, 200), 2, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, 1472, 400), 0)
			return th.composerArea(gtx, s, false)
		})
		wantPixel(t, img, image.Pt(1422, 173), faded(rgb(0xffffff), g, 0xa3), "the arrow at 64%")
	}
	if c := faded(rgb(0xffffff), rgb(0x0e0e0e), 0xa3); c != rgb(0xa8a8a8) {
		t.Errorf("white at 64%% over #0e0e0e is %v, want T3's #a8a8a8", c)
	}
}

// The question's Submit is off until it has its answer: at 64% over the
// glass, as Chrome composites it, and without its shadow.
func TestSubmitDisabled(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Questions: []Question{{Header: "Session", Text: "Which one?", Options: []string{"A", "B"}, Chosen: []bool{false, false}, NoCustom: true}}}
	img := drawComposer(t, th, s, 400, 0)
	if n := countAll(img, faded(P.MessageAction, glass(), 0xa3)); n < 1000 {
		t.Errorf("%d pixels of the faded Submit, want the pill", n)
	}
	if n := countAll(img, P.MessageAction); n != 0 {
		t.Errorf("%d pixels of --message-action at full strength on a disabled Submit", n)
	}
}

// While the answers are on their way the options are at 50% over the
// glass (§3.12), composited as Chrome does: the picked radio's --primary
// half way to the glass.
func TestAnsweringOptions(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Answering: true, Questions: []Question{{Header: "Session", Text: "Which one?", Options: []string{"A", "B"}, Chosen: []bool{false, true}}}}
	img := drawComposer(t, th, s, 400, 0)
	wantPixel(t, img, image.Pt(61, 119), faded(P.Primary, glass(), 0x80), "the picked radio at 50%")
}

// The trigger's provider mark is its 16dp wrapper at the content's start,
// the name 22 on (ProviderModelPicker.tsx; measured live: 157.75 wide for
// Opus 5.5 at Medium, the label at content + 22).
func TestTriggerGeometry(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "hi", Provider: "Claude", Model: "Opus 5.5", Effort: "Medium", Picker: &ModelPicker{Tabs: []PickerTab{{}}}}
	img := drawComposer(t, th, s, 200, 0)
	accent := over(P.Accent, glass())
	// Along a row under its text, the open trigger's fill runs from its
	// start, less the antialiased edge of its corner.
	left := 682
	for left > 45 && near(img.RGBAAt(left-1, 97), accent) {
		left--
	}
	left--
	if w := 687 - left; w < 155 || w > 161 {
		t.Errorf("the trigger is %d wide from x %d, want T3's 157.75", w, left)
	}
	mark := -1
	for x := left; x < 687 && mark < 0; x++ {
		for y := 75; y < 100; y++ {
			if near(img.RGBAAt(x, y), rgb(0xd97757)) {
				mark = x
				break
			}
		}
	}
	if mark < left+9 || mark > left+11 {
		t.Errorf("the provider's mark starts at %d, want the content's start, %d", mark, left+9)
	}
	if l, _ := ink(img, 78, 96, left+26, 687); l < left+31 || l > left+34 {
		t.Errorf("the name's ink starts at %d, want the content's start + 22, %d", l, left+31)
	}
}

// Dark's top highlight is an inset shadow, 1dp of #ffffff08 down inside
// the outline: it follows the corners' curve, painted from about x 12 on
// the row under the top border (§3.2, measured live #151515).
func TestDarkHighlight(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	th := NewTheme()
	img := drawComposer(t, th, &Scene{Model: "Opus 5.5"}, 200, 0)
	hl := over(alpha(rgb(0xffffff), 0x08), glass())
	wantPixel(t, img, image.Pt(14, 1), hl, "the highlight into the corner")
	wantPixel(t, img, image.Pt(300, 1), hl, "the highlight")
	wantPixel(t, img, image.Pt(300, 2), glass(), "under the highlight")
}

// hoverOn lays s's live composer out at y in a 736dp lane with the
// pointer moved to at, so the next drawComposer draws what it's over.
func hoverOn(th *Theme, s *Scene, h, y int, at f32.Point) {
	var r gioinput.Router
	var ops op.Ops
	frame := func() {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(736, h)), Source: r.Source(), Now: time.Now()}
		func() {
			defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
			th.composerArea(gtx, s, false)
		}()
		r.Frame(&ops)
		th.Actions()
	}
	frame()
	r.Queue(pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse})
	frame()
}

// A banner's dismiss fills with --accent under the pointer; its x stays
// the ghost button's muted icon colour (ComposerBanner.tsx Dismiss).
func TestDismissHover(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Model: "Opus 5.5", Fresh: true}
	// The dismiss is the banner's last 24dp: x 686..710, y 5..29.
	hoverOn(th, s, 200, 0, f32.Pt(698, 17))
	img := drawComposer(t, th, s, 200, 0)
	wantPixel(t, img, image.Pt(692, 12), over(P.Accent, glass()), "the hovered dismiss")
	darkest := 0xff
	for y := 5; y < 29; y++ {
		for x := 686; x < 710; x++ {
			darkest = min(darkest, int(img.RGBAAt(x, y).R))
		}
	}
	if darkest < int(P.ContrastMutedForeground.R)-6 {
		t.Errorf("the hovered x's darkest pixel is %#x, want --contrast-muted-foreground's %#x", darkest, P.ContrastMutedForeground.R)
	}
}

// With no rows the menu says why, 14dp under its top and 28 over the
// card: the surface's own 17dp under that tucks under the card
// (ComposerCommandMenu.tsx: pt-3.5 pb-7 in a pb-[17px] surface).
func TestCommandMenuEmpty(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "/zz", Model: "Opus 5.5", SuggestEmpty: "No matching command."}
	img := drawComposer(t, th, s, 300, 200)
	first, last := inkRows(img, 100, 200, 42, 600)
	if first < 0 {
		t.Fatal("no line in the empty menu")
	}
	top := firstRow(img, 300, 100, 200, over(alpha(P.Foreground, 0x14), glass()))
	if first-top < 16 || first-top > 22 {
		t.Errorf("the line's ink starts %d under the menu's top, want its 14dp padding and the line's own", first-top)
	}
	if gap := 200 - last - 1; gap < 24 || gap > 31 {
		t.Errorf("%ddp between the line's ink and the card, want T3's 28", gap)
	}
}

// Under the pointer, a picture's chip underlines its name and size in its
// own colour, 0.2em under the baseline (ContextChipLabel).
func TestChipUnderline(t *testing.T) {
	th := NewTheme()
	chip := th.ImageChip(1, "shot.png", 2048)
	s := &Scene{Live: true, Model: "Opus 5.5", Thumbs: []Thumb{{N: 1, Name: "shot.png", Size: 2048}}}
	th.SetEditorText("composer", "see "+chip+" now")
	inkOf := func() int {
		img := drawComposer(t, th, s, 300, 0)
		n := 0
		for y := 93; y < 125; y++ {
			n += count(img, y, 17, 400, P.ContextChipImage)
		}
		return n
	}
	hoverOn(th, s, 300, 0, f32.Pt(10, 280))
	plain := inkOf()
	// The prompt's first line is under the tile row: 17 + 64 + 12, and
	// the chip is just after "see ".
	hoverOn(th, s, 300, 0, f32.Pt(60, 104))
	if c := th.in.composer.clicks["chip:1"]; c == nil || !c.Hovered() {
		t.Fatal("the pointer isn't on the chip")
	}
	if n := inkOf() - plain; n < 30 {
		t.Errorf("hovered, the chip has %d more pixels of its colour, want its label's underline", n)
	}
}

// A tile's remove button is black at 65% over the picture as Chrome
// composites it, #0d1721 over a #24425f screenshot, its white 20% ring
// outside it and not under it (button.tsx media-close).
func TestTileRemove(t *testing.T) {
	th := NewTheme()
	pic := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = 0x24, 0x42, 0x5f, 0xff
	}
	var b bytes.Buffer
	if err := png.Encode(&b, pic); err != nil {
		t.Fatal(err)
	}
	img := drawComposer(t, th, &Scene{Model: "Opus 5.5", Thumbs: []Thumb{NewThumb(b.Bytes())}}, 200, 0)
	// The tile is at 17, 17; its 24dp button 4 in from its top right.
	got, want := img.RGBAAt(56, 30), rgb(0x0d1721)
	for _, d := range [][2]uint8{{got.R, want.R}, {got.G, want.G}, {got.B, want.B}} {
		if diff := int(d[0]) - int(d[1]); diff < -6 || diff > 6 {
			t.Fatalf("the remove button is #%02x%02x%02x, want about #0d1721", got.R, got.G, got.B)
		}
	}
}

// Short of room, the footer's mode block goes to its icon, then into the
// "…" menu, so the trigger never runs over the attach button
// (composerFooterLayout.ts): whole at 736, its 32dp icon at 300, the 34dp
// "…" at 240.
func TestFooterOverflow(t *testing.T) {
	th := NewTheme()
	s := &Scene{Composer: "hi", Provider: "Claude", Model: "Opus 5.5", Effort: "High", Runtime: "approval-required", RuntimeChip: true,
		Access: &Permissions{Modes: RuntimeModes, Sel: 0}}
	accent := over(P.Accent, glass())
	for _, c := range []struct {
		w, lo, hi int
		lock      bool // the mode's own icon, not the "…"
	}{{736, 90, 130, true}, {300, 27, 33, true}, {240, 29, 35, false}} {
		img := drawn(t, image.Pt(c.w, 200), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, c.w, 200), 0)
			return th.composerArea(gtx, s, false)
		})
		// The open block is lit, ending 8dp before send; along a row under
		// its icon its fill runs end to end, less its corners' edges.
		end := c.w - 1 - 8 - 32 - 8
		l, r := end-6, end-6
		for l > 0 && near(img.RGBAAt(l-1, 97), accent) {
			l--
		}
		for r < c.w-1 && near(img.RGBAAt(r+1, 97), accent) {
			r++
		}
		if n := r - l + 1; n < c.lo || n > c.hi {
			t.Errorf("%ddp lane: the mode block is %d wide, want %d..%d", c.w, n, c.lo, c.hi)
		}
		// The lock's ink runs down most of its 16dp; the "…" is one row of dots.
		rows := 0
		for y := 75; y < 100; y++ {
			for x := l; x <= r; x++ {
				if img.RGBAAt(x, y).R < 0xc8 {
					rows++
					break
				}
			}
		}
		if c.lock != (rows >= 10) {
			t.Errorf("%ddp lane: the block's ink is %d rows tall, want the mode's icon %v", c.w, rows, c.lock)
		}
		if l, _ := ink(img, 75, 100, 32, 45); l >= 0 {
			t.Errorf("%ddp lane: ink at x %d, between the attach button and the controls", c.w, l)
		}
	}
}

// The prompt's lines sit where Chrome puts them: at 2x it lays text out in
// device pixels, the font's ascent and descent rounded and the
// half-leading floored, so 14/22.75 Inter has its baseline 32px under the
// line's top (§3.3).
func TestPromptBaseline(t *testing.T) {
	th := NewTheme()
	img := drawn(t, image.Pt(736, 200), 2, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 1472, 400), 0)
		return th.composerArea(gtx, &Scene{Model: "Opus 5.5", Placeholder: "Ask"}, false)
	})
	// The line's top is 17dp down, 34px; "Ask" sits on the baseline.
	last := -1
	for y := 34; y < 90; y++ {
		n := 0
		for x := 34; x < 200; x++ {
			if img.RGBAAt(x, y).R < 0xa0 {
				n++
			}
		}
		if n >= 3 {
			last = y
		}
	}
	if last != 34+32-1 {
		t.Errorf("the placeholder's ink ends on row %d, want it on the baseline at %d", last, 34+32)
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
	// The chip is 73..101 tall and ends at 687: 5dp in from its end, clear
	// of its rounded corners and of the open menu's shadow, which reaches
	// row 86.
	wantPixel(t, img, image.Pt(682, 90), accent, "open chip")
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

// The trigger's bolt follows the model's speed (FastBolt): 12dp for Fast,
// the echoed 15 for Ultrafast, and Ultracode adds its 12dp workflow icon
// 6 after. The trigger ends 8 before send, so its provider's mark moves
// left by as much.
func TestTriggerSpeed(t *testing.T) {
	th := NewTheme()
	mark := func(s *Scene) int {
		t.Helper()
		img := drawComposer(t, th, s, 200, 0)
		for x := 45; x < 690; x++ {
			for y := 75; y < 100; y++ {
				if near(img.RGBAAt(x, y), rgb(0xd97757)) {
					return x
				}
			}
		}
		t.Fatal("no provider's mark on the trigger")
		return 0
	}
	s := &Scene{Composer: "hi", Provider: "Claude", Model: "Opus 5.5", Effort: "High", Speed: "Fast"}
	fast := mark(s)
	s.Speed = "Ultrafast"
	if d := fast - mark(s); d < 2 || d > 4 {
		t.Errorf("Ultrafast's bolt widened the trigger %d, want 3", d)
	}
	s.Ultracode = true
	if d := fast - mark(s); d < 20 || d > 22 {
		t.Errorf("with Ultracode the trigger is %d wider, want 21", d)
	}
}

// With no provider to send to, the trigger is T3's unavailable control,
// off at 64% (ComposerControlsBar.tsx): no provider's mark, and its words
// faded, foreground 80% (#464646) at 64% over the glass, #898989.
func TestNoProviderTrigger(t *testing.T) {
	th := NewTheme()
	img := drawComposer(t, th, &Scene{Composer: "hi", Provider: "Claude", Model: "Opus 5.5", NoProvider: true}, 200, 0)
	darkest := 0xff
	for y := 75; y < 100; y++ {
		for x := 400; x < 690; x++ {
			c := img.RGBAAt(x, y)
			if near(c, rgb(0xd97757)) {
				t.Fatalf("the provider's mark at (%d, %d)", x, y)
			}
			darkest = min(darkest, int(c.R))
		}
	}
	if darkest < 0x80 || darkest > 0xb4 {
		t.Errorf("the trigger's darkest ink is %#x, want its words at 64%%", darkest)
	}
}

// The card's light shadow, 0 1px 2px black 5%, blurred as Chrome blurs
// it and at its full strength over the page, not the half Gio's linear
// blending would make of it: at 2x, T3's device rows under the card read
// #f3 #f5 #f7 #f9 #fb over its first 2.5px (§3.2).
func TestComposerShadow(t *testing.T) {
	th := NewTheme()
	img := drawComposer(t, th, &Scene{Model: "Opus 5.5"}, 200, 0)
	for _, c := range []struct{ y, lo, hi int }{{112, 0xf1, 0xf5}, {113, 0xf6, 0xfa}, {114, 0xfa, 0xfc}} {
		if r := int(img.RGBAAt(300, c.y).R); r < c.lo || r > c.hi {
			t.Errorf("1x row %d under the card is %#x, want %#x..%#x", c.y, r, c.lo, c.hi)
		}
	}
	img = drawn(t, image.Pt(736, 200), 2, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 1472, 400), 0)
		return th.composerArea(gtx, &Scene{Model: "Opus 5.5"}, false)
	})
	// The card is 111.5 tall: device rows 0 to 222.
	for i, want := range []int{0xf3, 0xf5, 0xf7, 0xf9, 0xfb} {
		if r := int(img.RGBAAt(600, 223+i).R); r < want-2 || r > want+2 {
			t.Errorf("2x row %d under the card is %#x, want T3's %#x", 223+i, r, want)
		}
	}
}

// A popup's shadow, 0 16px 40px -18px black 55%, is a Gaussian ramp, not
// bands: #bebebe just under the permissions menu, easing out to the page
// some 45dp down (§3.8, measured live under T3's menu).
func TestPopupShadow(t *testing.T) {
	th := NewTheme()
	s := &Scene{Access: &Permissions{Modes: RuntimeModes, Sel: 0}}
	var d D
	img := drawn(t, image.Pt(600, 400), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 600, 400), 0)
		defer op.Offset(image.Pt(100, 50)).Push(gtx.Ops).Pop()
		d = th.permissionsMenu(gtx, s)
		return d
	})
	x, b := 100+d.Size.X/2, 50+d.Size.Y
	row := func(i int) int { return int(img.RGBAAt(x, b+i).R) }
	for _, c := range []struct{ i, lo, hi int }{{0, 0xb9, 0xc3}, {10, 0xd2, 0xdc}, {20, 0xe6, 0xee}, {45, 0xf9, 0xfc}} {
		if r := row(c.i); r < c.lo || r > c.hi {
			t.Errorf("%ddp under the menu the shadow is %#x, want %#x..%#x", c.i, r, c.lo, c.hi)
		}
	}
	levels := map[int]bool{}
	for i := 0; i < 30; i++ {
		if row(i+1) < row(i) {
			t.Errorf("the shadow darkens again %ddp down: %#x after %#x", i+1, row(i+1), row(i))
		}
		levels[row(i)] = true
	}
	if len(levels) < 20 {
		t.Errorf("the shadow's first 30dp take %d shades, want a smooth ramp", len(levels))
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
	img := drawComposer(t, th, &Scene{Fresh: true, Model: "Opus 5.5"}, 200, 0)
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

// While /clear is armed its banner sits over the card (ComposerFreshSession.tsx):
// the eraser 6dp into its icon column, the title, the description after
// it and the dismiss at the card's end; a card under 400 drops the
// description.
func TestFreshBanner(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Fresh: true}
	img := drawComposer(t, th, s, 200, 0)
	if l, r := ink(img, 5, 29, 22, 714); l < 28 || l > 40 || r < 690 {
		t.Errorf("the banner's ink spans x %d..%d, want the eraser from about 32 and the dismiss by its end", l, r)
	}
	if l, _ := ink(img, 5, 29, 320, 520); l < 0 {
		t.Error("no description after the title")
	}
	narrow := drawn(t, image.Pt(420, 200), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 420, 200), 0)
		return th.composerArea(gtx, s, false)
	})
	if l, _ := ink(narrow, 5, 29, 300, 360); l >= 0 {
		t.Errorf("a 376dp banner still shows its description, at x %d", l)
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
	// The tab puts the card 34dp lower, at 234, and the list rises 56dp
	// over it: its one row is 206..230, above the footer's send.
	for y := 150; y < 260; y++ {
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
	// The tab ends 22dp in from the lane's end (1194) and is about 93dp
	// wide: its top edge is straight at 1146, past its rounded corners.
	tab := firstRow(img, 1146, 200, 600, outline)
	if tab < 0 {
		t.Fatal("no stash tab on the new thread")
	}
	// The subline's ink, left of the tab and its label.
	if _, sub := inkRows(img, tab-60, tab, 500, 1090); tab-sub < 2 || tab-sub > 10 {
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
	// Each one's widest row, end to end: its label or square, drawn over
	// its middle, doesn't shorten that.
	widest := func(c color.NRGBA) int {
		most := 0
		for y := 0; y < 400; y++ {
			l, r := -1, -1
			for x := 400; x < 736; x++ {
				if near(img.RGBAAt(x, y), c) {
					if l < 0 {
						l = x
					}
					r = x
				}
			}
			if l >= 0 {
				most = max(most, r-l+1)
			}
		}
		return most
	}
	if n := widest(P.MessageAction); n < 60 || n > 85 {
		t.Errorf("the Submit pill is %d wide, want T3's compact 70.86", n)
	}
	if n := widest(over(alpha(P.Destructive, 0xe6), g)); n < 26 || n > 29 {
		t.Errorf("Stop is %d wide in the row, want 28", n)
	}
}

// The provider's suggestion is the placeholder, with its outline Tab key
// 8dp after it at its own size (ChatComposer.tsx:3571; T3's "⇥ Tab" is
// 50.7 wide, 20 tall).
func TestSuggestion(t *testing.T) {
	th := NewTheme()
	s := &Scene{Model: "Opus 5.5", Placeholder: "go ahead set it up", Suggestion: "go ahead set it up"}
	img := drawComposer(t, th, s, 200, 0)
	edge := over(alpha(P.Foreground, 0x26), glass())
	left, right, top, bottom := -1, -1, -1, -1
	for y := 10; y < 60; y++ {
		for x := 17; x < 736; x++ {
			if near(img.RGBAAt(x, y), edge) {
				if left < 0 || x < left {
					left = x
				}
				right, bottom = max(right, x), y
				if top < 0 {
					top = y
				}
			}
		}
	}
	if left < 0 {
		t.Fatal("no key after the suggestion")
	}
	if w := right - left + 1; w < 44 || w > 58 {
		t.Errorf("the key spans x %d..%d, %d wide, want about 51", left, right, w)
	}
	if h := bottom - top + 1; h < 19 || h > 21 {
		t.Errorf("the key spans rows %d..%d, want 20 tall", top, bottom)
	}
	if _, text := ink(img, 17, 45, 17, left-2); left-text < 7 || left-text > 11 {
		t.Errorf("the suggestion's ink ends at %d and its key starts at %d, want 8 between", text, left)
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
	// The picture is a tile too (useComposerImageGroups.ts), so the prompt's
	// first line is under the tile row: 17 + 64 + 12 = 93.
	end, start := -1, -1
	for y := 88; y < 124; y++ {
		for x := 17; x < 720; x++ {
			if near(img.RGBAAt(x, y), chip) || (img.RGBAAt(x, y).B > img.RGBAAt(x, y).R+40 && img.RGBAAt(x, y).R < 0xb0) {
				end = max(end, x)
			}
		}
	}
	if end < 0 {
		t.Fatal("no chip drawn")
	}
	for y := 88; y < 124; y++ {
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
