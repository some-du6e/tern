package ui

import (
	"image"
	"testing"
)

// Toasts (02-primitives.md §3.24) stack 32 in from the right and 84 down,
// 360 wide: 46 tall with a title, 68 with a one-line description, 88 with
// two; the close orb overhangs the top right corner by 5.
func TestToasts(t *testing.T) {
	defer SetDark(false)
	two := "Its CPU is overloaded, so T3 Code on studio was too busy to answer you."
	for _, d := range []bool{false, true} {
		SetDark(d)
		edge := over(alpha(rgb(0x14141d), 0x1a), P.Popover)
		if d {
			edge = over(alpha(rgb(0xf5f5f5), 0x1a), P.Popover)
		}
		for _, c := range []struct {
			desc string
			h    int
		}{{"", 46}, {"Its CPU is overloaded (load 9 on 4 cores).", 68}, {two, 88}} {
			th := NewTheme()
			s := &Scene{Toasts: []Toast{{ID: "1", Type: ToastWarning, Title: "studio froze for 12s", Description: c.desc}}}
			img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
				fill(gtx, P.Background, image.Rect(0, 0, 1440, 900), 0)
				th.toasts(gtx, s)
				return D{}
			})
			wantPixel(t, img, image.Pt(1228, 84), edge, "toast's top edge")
			wantPixel(t, img, image.Pt(1048, 120), edge, "its left edge, 32+360 in")
			bottom := firstNot(img, 1060, 86, P.Popover)
			if h := bottom - 84 + 1; h != c.h {
				t.Errorf("dark %v: toast with %q is %d tall, want %d", d, c.desc, h, c.h)
			}
			wantPixel(t, img, image.Pt(1389, 91), over(alpha(P.Border, uint8(int(P.Border.A)*60/100)), P.Popover), "close orb's edge")
		}
	}
}
