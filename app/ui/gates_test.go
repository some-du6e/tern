package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// glassEdge is dialog-glass's border over the page (§3.1).
func glassEdge() color.NRGBA {
	if dark {
		return over(alpha(rgb(0xffffff), 0x14), P.Background)
	}
	return over(alpha(P.Foreground, 0x1a), P.Background)
}

// cardRows finds a standalone card's top and bottom edges down column x.
func cardRows(t *testing.T, img *image.RGBA, x int) (top, bottom int) {
	t.Helper()
	top = firstNot(img, x, 0, P.Background)
	// The dark card has a 1dp light inside its top edge.
	bottom = firstNot(img, x, top+2, P.Background)
	if top < 0 || bottom < 0 {
		t.Fatalf("no card down x %d", x)
	}
	return top, bottom
}

// The pairing form's StandalonePage card (docs/ui/11-empty-auth.md §3.1,
// §3.2): 576 wide, centred at x 432 on a 1440 window, 368 tall, 430 with a
// one-line error Alert; dialog-glass's edge on the page's --background.
func TestPairingCard(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		for _, c := range []struct {
			err string
			h   int
		}{{"", 368}, {"Enter a pairing link to continue.", 430}} {
			th := NewTheme()
			s := &Scene{Gate: GatePairing, Pair: &Pairing{Error: c.err}}
			img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Gate(gtx, s) })
			top, bottom := cardRows(t, img, 452)
			if h := bottom - top + 1; h != c.h {
				t.Errorf("dark %v, error %q: card %d tall, want %d", d, c.err, h, c.h)
			}
			if top != (900-c.h)/2 {
				t.Errorf("dark %v: card top at %d, want it centred at %d", d, top, (900-c.h)/2)
			}
			wantPixel(t, img, image.Pt(720, top), glassEdge(), "card's top edge")
			if near(img, 431, top+40, glassEdge()) || !near(img, 432, top+40, glassEdge()) {
				t.Errorf("dark %v: card's left edge isn't at x 432", d)
			}
			wantPixel(t, img, image.Pt(100, 100), P.Background, "page")
		}
	}
}

// The header's "T3" glyph (T3Wordmark.tsx) is 12 tall and 19.88 wide in
// --foreground: its T's stem runs 3.8 to 6.5dp in, down from 2.2dp.
func TestWordmark(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		img := drawn(t, image.Pt(200, 60), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, 200, 60), 0)
			return wordmark(P.Foreground, 12)(gtx)
		})
		wantPixel(t, img, image.Pt(5, 6), P.Foreground, "the T's stem")
		wantPixel(t, img, image.Pt(5, 13), P.Background, "under the glyph")
		wantPixel(t, img, image.Pt(21, 6), P.Background, "past the glyph's 19.88")
	}
}

// tracked text draws what cssText draws, each glyph moved by the
// spacing: none moves nothing, and -0.45 on "Code" takes 4 x 0.45 off.
func TestTrackedText(t *testing.T) {
	th := NewTheme()
	tx := Txt{Size: 18, LineH: 28.0 / 18, Font: UI(font.Medium), Color: rgb(0)}
	plain := drawn(t, image.Pt(80, 30), 2, func(gtx C) D {
		fill(gtx, rgb(0xffffff), image.Rect(0, 0, 160, 60), 0)
		return th.cssText(gtx, tx, "Code")
	})
	same := drawn(t, image.Pt(80, 30), 2, func(gtx C) D {
		fill(gtx, rgb(0xffffff), image.Rect(0, 0, 160, 60), 0)
		return th.tracked(gtx, tx, "Code", 0)
	})
	for y := 0; y < 60; y++ {
		for x := 0; x < 160; x++ {
			if plain.RGBAAt(x, y) != same.RGBAAt(x, y) {
				t.Fatalf("untracked text differs from cssText's at %d,%d", x, y)
			}
		}
	}
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Constraints{Max: image.Pt(400, 100)}}
	d0, d1 := th.tracked(gtx, tx, "Code", 0), th.tracked(gtx, tx, "Code", -0.45)
	if d0.Size.X-d1.Size.X != 3 {
		t.Errorf("tracked -0.45 is %dpx narrower at 2x, want 3 (4 x 0.9)", d0.Size.X-d1.Size.X)
	}
	if d1.Size.Y != 56 {
		t.Errorf("tracked 18/28 is %dpx tall at 2x, want its line box's 56", d1.Size.Y)
	}
}

// The root error view (§3.4): with a report that scrolls, the card is
// 509 tall: the header, Try again and its row 20 under it, and the
// report box 20 under that, 286 tall with its 256 pre.
func TestRootErrorView(t *testing.T) {
	defer SetDark(false)
	report := "tern 0.1.0\nTime: 2026-11-18T17:27:22.221Z\n\n" + strings.Repeat("at somewhere (file.go:12)\n", 30)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Gate: GateSessionError, Failure: &Failure{Message: "List environments failed (500).", Report: report}}
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Gate(gtx, s) })
		top, bottom := cardRows(t, img, 452)
		if h := bottom - top + 1; h != 509 {
			t.Errorf("dark %v: card %d tall, want 509", d, h)
		}
		// Try again is --primary, 20 under the 105 of header from the
		// card's top edge, 25 in; 28 tall.
		wantPixel(t, img, image.Pt(432+25+4, top+25+105+20+14), P.Primary, "Try again")
	}
}

// The boot splash (§2.7, §3.7) is the boot style's white or black, not
// --background, with tern's 64dp icon in the middle.
func TestBootSplash(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Gate(gtx, &Scene{Gate: GateBooting}) })
		bg := rgb(0xffffff)
		if d {
			bg = rgb(0)
		}
		wantPixel(t, img, image.Pt(10, 10), bg, "window")
		wantPixel(t, img, image.Pt(686, 450), bg, "left of the icon")
		wantPixel(t, img, image.Pt(700, 430), rgb(0x161a20), "icon's tile")
	}
}

// The boot failure page (§3.8): a 400dp column at x 520, 198 down a 900
// tall window; Reload 16 under the 22 + 4 + 20 of message, 32 tall, filled
// with the boot foreground.
func TestBootFailurePage(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Gate: GateBootFailed, Failure: &Failure{Report: "open store: permission denied"}}
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Gate(gtx, s) })
		bg, fg := rgb(0xffffff), rgb(0x262626)
		if d {
			bg, fg = rgb(0), rgb(0xf5f5f5)
		}
		wantPixel(t, img, image.Pt(522, 198+22+4+20+16+16), fg, "Reload")
		wantPixel(t, img, image.Pt(518, 198+22+4+20+16+16), bg, "left of the column")
		if inked(img, image.Rect(520, 150, 920, 196), bg) {
			t.Errorf("dark %v: something above the column's 198dp padding", d)
		}
		if !inked(img, image.Rect(520, 198, 920, 220), bg) {
			t.Errorf("dark %v: no heading at the column's top", d)
		}
	}
}

// T3 Connect's sign-in is Clerk's card (§3.6): 400 wide at y 64, its email
// field 320x32 on --secondary inside a 1dp outline, and Continue 32 under
// it, filled --primary with no gloss.
func TestSignInCard(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Gate: GateSignIn, Login: &SignIn{}}
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Gate(gtx, s) })
		wantPixel(t, img, image.Pt(720, 64), glassEdge(), "card's top edge")
		wantPixel(t, img, image.Pt(720, 63), P.Background, "above the card")
		// 1 + 32 in, the header's 28 + 4 + 20, 32, the label's 18 and 8.
		field := 64 + 33 + 52 + 32 + 18 + 8
		wantPixel(t, img, image.Pt(570, field+16), over(P.Secondary, P.Background), "email field")
		wantPixel(t, img, image.Pt(559, field+16), over(clerkEdge(true), P.Background), "field's outline")
		wantPixel(t, img, image.Pt(557, field+16), P.Background, "outside the outline")
		button := field + 32 + 32
		wantPixel(t, img, image.Pt(570, button+2), P.Primary, "Continue's top")
		wantPixel(t, img, image.Pt(570, button+29), P.Primary, "Continue's foot")
	}
}
