package ui

import (
	"image"
	"image/color"
)

// near reports whether the pixel at x, y is c, give or take a step.
func near(img *image.RGBA, x, y int, c color.NRGBA) bool {
	got := img.RGBAAt(x, y)
	d := func(a, b uint8) bool { return a-b < 3 || b-a < 3 }
	return d(got.R, c.R) && d(got.G, c.G) && d(got.B, c.B)
}

// firstNot is the first y from y0 down column x whose pixel isn't c, or -1.
func firstNot(img *image.RGBA, x, y0 int, c color.NRGBA) int {
	for y := y0; y < img.Bounds().Dy(); y++ {
		if !near(img, x, y, c) {
			return y
		}
	}
	return -1
}

// inked reports whether anything but c is drawn in r.
func inked(img *image.RGBA, r image.Rectangle, c color.NRGBA) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if !near(img, x, y, c) {
				return true
			}
		}
	}
	return false
}
