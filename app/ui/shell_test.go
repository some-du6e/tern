package ui

import (
	"image"
	"testing"
)

// The lane rule reproduces T3's measured lanes at 1440x900 with the
// sidebar open (docs/ui/03-shell.md §3.5, §5.4): the canvas is 1184 wide,
// the details card 280 wide at 892.
func TestLane(t *testing.T) {
	for _, c := range []struct {
		name               string
		maxChat, cardLeft  int
		wantLeft, wantWide int
	}{
		{"comfortable, card inline", 736, 892, 380 - 256, 736},
		{"comfortable, no card", 736, -1, 480 - 256, 736},
		{"wide, card inline", 1152, 892, 304 - 256, 812},
		{"wide, no card", 1152, -1, 304 - 256, 1088},
	} {
		left, w := lane(1184, 48, c.maxChat, 32, c.cardLeft)
		if left != c.wantLeft || w != c.wantWide {
			t.Errorf("%s: lane at %d, %d wide; want %d, %d", c.name, left, w, c.wantLeft, c.wantWide)
		}
	}
}

// The sidebar: its 255dp surface, then the 1dp border at x 255, then the
// main area; the stage art at the top (its sky is #07152f in both themes).
func TestSidebarFrame(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, &Scene{Live: true}) })
		wantPixel(t, img, image.Pt(254, 500), SP.Sidebar, "sidebar surface")
		wantPixel(t, img, image.Pt(255, 500), over(SP.ContrastBorder, P.Background), "sidebar border")
		wantPixel(t, img, image.Pt(300, 500), P.Background, "main area")
		wantPixel(t, img, image.Pt(10, 6), rgb(0x07152f), "stage art sky")
	}
}

// The title bar's panel toggles sit where T3's do at 1440 wide (x 1335,
// 1367, 1399; 28dp at y 12), and a pressed one is filled with --accent.
func TestTitlebarControls(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Title: "A thread", Run: &RunInfo{Status: "Idle"}, Term: nil}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		th.titleButtonW("terminal.toggle", "", titleButton{icon: "panel-bottom", pressed: true})(gtx)
		return D{}
	})
	wantPixel(t, img, image.Pt(2, 14), over(P.Accent, P.Background), "pressed toggle")
	img = drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 1440, 900), 0)
		th.titlebarControls(gtx, s, image.Pt(1440, 900), false)
		return D{}
	})
	for _, x := range []int{1335, 1367, 1399} {
		wantPixel(t, img, image.Pt(x+2, 14), P.Background, "unpressed toggle")
	}
}
