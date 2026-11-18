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
	got := img.RGBAAt(p.X, p.Y)
	near := func(a, b uint8) bool { return a-b < 3 || b-a < 3 }
	if !near(got.R, c.R) || !near(got.G, c.G) || !near(got.B, c.B) {
		t.Errorf("%s at %v is #%02x%02x%02x, want #%02x%02x%02x", what, p, got.R, got.G, got.B, c.R, c.G, c.B)
	}
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
