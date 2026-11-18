package ui

import (
	"image"
	"image/color"
	"runtime"
	"testing"
	"time"

	"gioui.org/font"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// drawn renders w at size (in dp) and scale the way the window would, and
// returns the pixels, so tests check what is actually drawn. CI's Linux job
// draws with Mesa's software GPU; elsewhere a missing GPU skips the test.
func drawn(t *testing.T, size image.Point, scale float32, w layout.Widget) *image.RGBA {
	t.Helper()
	px := image.Pt(int(float32(size.X)*scale), int(float32(size.Y)*scale))
	win, err := headless.NewWindow(px.X, px.Y)
	if err != nil {
		if runtime.GOOS == "linux" {
			t.Fatalf("headless GPU: %v", err)
		}
		t.Skipf("no headless GPU here: %v", err)
	}
	defer win.Release()
	var ops op.Ops
	w(layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: scale, PxPerSp: scale}, Constraints: layout.Exact(px), Now: time.Now()})
	if err := win.Frame(&ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: px})
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	return img
}

// wantPixel fails unless the pixel at p (in px) is c, give or take a step.
func wantPixel(t *testing.T, img *image.RGBA, p image.Point, c color.NRGBA, what string) {
	t.Helper()
	if got := img.RGBAAt(p.X, p.Y); !nearPixel(got, c) {
		t.Errorf("%s at %v is #%02x%02x%02x, want #%02x%02x%02x", what, p, got.R, got.G, got.B, c.R, c.G, c.B)
	}
}

// nearPixel reports a pixel that is c, give or take a step.
func nearPixel(got color.RGBA, c color.NRGBA) bool {
	near := func(a, b uint8) bool { d := int(a) - int(b); return d < 3 && d > -3 }
	return near(got.R, c.R) && near(got.G, c.G) && near(got.B, c.B)
}

// nearInk reports two drawn pixels within a few steps of each other: the
// GPU's antialiasing of a glyph shifts a little with what else the frame
// draws, far less than a change of weight or colour makes.
func nearInk(a, b color.RGBA) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= 16 && int(y)-int(x) <= 16 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

// inkRows is the first and last row of pixels in r that something is drawn
// on: more than a few steps off bg. ok is false if nothing is.
func inkRows(img *image.RGBA, r image.Rectangle, bg color.NRGBA) (top, bottom int, ok bool) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if inked(img.RGBAAt(x, y), bg) {
				if !ok {
					top, ok = y, true
				}
				bottom = y
				break
			}
		}
	}
	return top, bottom, ok
}

// inkCols is inkRows across: the first and last column drawn on.
func inkCols(img *image.RGBA, r image.Rectangle, bg color.NRGBA) (left, right int, ok bool) {
	for x := r.Min.X; x < r.Max.X; x++ {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			if inked(img.RGBAAt(x, y), bg) {
				if !ok {
					left, ok = x, true
				}
				right = x
				break
			}
		}
	}
	return left, right, ok
}

// inked reports a pixel more than a few steps off bg.
func inked(c color.RGBA, bg color.NRGBA) bool {
	off := func(a, b uint8) bool { return int(a)-int(b) > 24 || int(b)-int(a) > 24 }
	return off(c.R, bg.R) || off(c.G, bg.G) || off(c.B, bg.B)
}

// The window wears T3's light or dark theme with the system: the
// conversation on --background, the sidebar on its own --sidebar (black in
// dark mode, from the sidebar's scoped palette).
func TestThemeFollowsSystem(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Live: true}
		img := drawn(t, image.Pt(1000, 700), 1, func(gtx C) D { return th.Desktop(gtx, s) })
		main, side := T3Light.Background, T3LightSidebar.Sidebar
		if d {
			main, side = T3Dark.Background, T3DarkSidebar.Sidebar
		}
		wantPixel(t, img, image.Pt(700, 300), main, "conversation background")
		wantPixel(t, img, image.Pt(100, 600), side, "sidebar background")
	}
}

// Text comes in the four weights T3 uses, each its own face: bolder Inter
// (or SF Pro on a Mac) sets wider.
func TestT3Weights(t *testing.T) {
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Constraints{Max: image.Pt(4000, 400)}}
	last := 0
	for _, w := range []font.Weight{font.Normal, font.Medium, font.SemiBold, font.Bold} {
		got := th.Text(gtx, Txt{Size: 14, Font: UI(w)}, "Hamburgefonstiv").Size.X
		if got <= last {
			t.Fatalf("weight %v is %dpx wide, no wider than the lighter weight's %dpx", w, got, last)
		}
		last = got
	}
}
