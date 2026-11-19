package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The Alert (docs/ui/11-empty-auth.md §3.5): rounded 12, padded 12 by 14
// round 14/20 text, so a one-line description is 46 tall, two lines 66,
// and a title over one line 68; the error variant on --error-surface
// inside error 32%, its icon in the 16dp box 14 in.
func TestAlert(t *testing.T) {
	defer SetDark(false)
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(526, 400)}}
	long := "Verify the backend is reachable from this browser, supports CORS for hosted clients, and is served over HTTPS."
	for _, c := range []struct {
		title, desc string
		h           int
	}{{"", "Enter a pairing link to continue.", 46}, {"", long, 66}, {"Pairing failed", "Try again.", 68}} {
		if got := th.T3Alert(AlertError, "", c.title, c.desc, color.NRGBA{})(gtx).Size; got != image.Pt(526, c.h) {
			t.Errorf("alert %q / %q is %v, want 526x%d", c.title, c.desc, got, c.h)
		}
	}
	for _, d := range []bool{false, true} {
		SetDark(d)
		img := drawn(t, image.Pt(526, 60), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rect(0, 0, 526, 60), 0)
			return th.T3Alert(AlertError, "circle-alert", "", "Enter a pairing link to continue.", color.NRGBA{})(gtx)
		})
		bg := over(P.ErrorSurface, P.Background)
		wantPixel(t, img, image.Pt(400, 6), bg, "error surface")
		wantPixel(t, img, image.Pt(400, 0), over(alpha(P.Error, 0x52), bg), "error edge")
		if !inked(img, image.Rect(15, 15, 31, 31), bg) {
			t.Errorf("dark %v: no icon in the 16dp box 14 in", d)
		}
	}
}
