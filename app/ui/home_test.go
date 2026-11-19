package ui

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The main column's empty states (docs/ui/11-empty-auth.md §2.10-2.13,
// §3.9) at 1440x900 with the sidebar open: the column is 1184 wide at x
// 256, the Empty block padded 48 and centred under the 52dp top bar.
// NoActiveThreadState's bar has its rule and "No active thread"; its
// 20/28 title over the 14/20 description sits in a 148 block, so the
// title's line is y 450-478. The hero's 30/36 title is 4 higher, at 446.
// Neither has the thread's panel toggles.
func TestEmptyStates(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		for _, c := range []struct {
			home  Home
			title int
			rule  bool
		}{{HomeNoThread, 450, true}, {HomeNoProjects, 446, false}} {
			th := NewTheme()
			img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, &Scene{Home: c.home}) })
			bg := P.Background
			lane := func(y0, y1 int) image.Rectangle { return image.Rect(560, y0, 1150, y1) }
			if !inked(img, lane(c.title+6, c.title+26), bg) {
				t.Errorf("dark %v, home %d: no title on y %d", d, c.home, c.title)
			}
			if inked(img, lane(c.title-40, c.title-2), bg) {
				t.Errorf("dark %v, home %d: something above the title's padding", d, c.home)
			}
			if inked(img, lane(c.title+60, c.title+100), bg) {
				t.Errorf("dark %v, home %d: something under the description", d, c.home)
			}
			rule := over(P.Border, bg)
			if got := near(img, 800, 51, rule); got != c.rule {
				t.Errorf("dark %v, home %d: top bar's rule is %v, want %v", d, c.home, got, c.rule)
			}
			if c.rule && !inked(img, image.Rect(276, 16, 400, 36), bg) {
				t.Errorf("dark %v: no \"No active thread\" in the top bar", d)
			}
			wantPixel(t, img, image.Pt(1349, 26), bg, "where the panel toggles would be")
		}
	}
}

// The hosted home's CommandBlock (§3.9) is 46 tall, rounded 8, its
// --border edge round --muted at 50%, the text 12 in.
func TestCommandBlock(t *testing.T) {
	defer SetDark(false)
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(364, 100)}}
	if got := th.commandBlock(0, "t3 connect")(gtx).Size; got != image.Pt(364, 46) {
		t.Errorf("command block is %v, want 364x46", got)
	}
	for _, d := range []bool{false, true} {
		SetDark(d)
		img := drawn(t, image.Pt(364, 46), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, 364, 46), 0)
			return th.commandBlock(0, "t3 connect")(gtx)
		})
		bg := over(pct(P.Muted, 50), P.Background)
		wantPixel(t, img, image.Pt(6, 23), bg, "command block")
		wantPixel(t, img, image.Pt(0, 23), over(P.Border, bg), "its edge")
		if !inked(img, image.Rect(13, 13, 60, 33), bg) {
			t.Errorf("dark %v: no \"$ t3 connect\" 12 in", d)
		}
	}
}
