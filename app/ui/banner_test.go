package ui

import (
	"image"
	"testing"
)

// solidCard is a stand-in for the composer's card, 736x100.
func solidCard(gtx C) D {
	fill(gtx, rgb(0x123456), image.Rect(0, 0, 736, 100), 0)
	return D{Size: image.Pt(736, 100)}
}

// The machine's banner (docs/ui/11-empty-auth.md §2.18, §3.11;
// 07-composer.md §3.9): in a thread it's attached, 44 narrower than the
// card and 34 of it above it, on the warning glass inside its 28% edge;
// the card covers its other 17. On a new thread's hero it's a whole
// 34-tall card 16 under the composer. A refusal is in the error's
// colours, its edge at 32%.
func TestMachineBanner(t *testing.T) {
	defer SetDark(false)
	b := &Banner{Title: "studio is reconnecting", Action: "Retry now"}
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		surface := P.Card
		if d {
			surface = P.SurfaceRaised
		}
		bg := over(alpha(P.Warning, 0x14), over(alpha(surface, 0xcc), P.Background))
		edge := over(alpha(P.Warning, 0x47), bg)
		var size image.Point
		img := drawn(t, image.Pt(736, 200), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, 736, 200), 0)
			size = th.withBanner(gtx, &Scene{Banner: b}, solidCard).Size
			return D{Size: size}
		})
		if size != image.Pt(736, 134) {
			t.Errorf("dark %v: attached banner and card are %v, want 736x134", d, size)
		}
		wantPixel(t, img, image.Pt(400, 3), bg, "attached banner")
		wantPixel(t, img, image.Pt(400, 0), edge, "its top edge")
		wantPixel(t, img, image.Pt(22, 20), edge, "its side edge, 22 in")
		wantPixel(t, img, image.Pt(300, 40), rgb(0x123456), "the card, 34 down")
		if !inked(img, image.Rect(54, 8, 120, 26), bg) {
			t.Errorf("dark %v: no title after the icon column", d)
		}
		if !inked(img, image.Rect(640, 8, 710, 26), bg) {
			t.Errorf("dark %v: no Retry now at the end", d)
		}

		img = drawn(t, image.Pt(736, 200), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, 736, 200), 0)
			size = th.withBanner(gtx, &Scene{Banner: b, Hero: "tern"}, solidCard).Size
			return D{Size: size}
		})
		if size != image.Pt(736, 150) {
			t.Errorf("dark %v: hero card and banner are %v, want 736x150", d, size)
		}
		wantPixel(t, img, image.Pt(400, 119), bg, "detached banner, 16 under the card")
		wantPixel(t, img, image.Pt(400, 116), edge, "its top edge")
		wantPixel(t, img, image.Pt(0, 133), edge, "its side edge")
	}
	SetDark(false)
	th := NewTheme()
	img := drawn(t, image.Pt(736, 200), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 736, 200), 0)
		return th.withBanner(gtx, &Scene{Banner: &Banner{Error: true, Title: "studio is offline", Action: "Reconnect"}}, solidCard)
	})
	bg := over(alpha(P.Error, 0x14), over(alpha(P.Card, 0xcc), P.Background))
	wantPixel(t, img, image.Pt(400, 3), bg, "refused banner")
	wantPixel(t, img, image.Pt(400, 0), over(alpha(P.Error, 0x52), bg), "its edge")
}
