package ui

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"time"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// solidPicture is a w by h picture of one colour, decoded for the dialog.
func solidPicture(t *testing.T, w, h int) Picture {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 0x33, 0x66, 0xcc, 0xff
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return NewFullPicture(b.Bytes())
}

// The image dialog shows a picture at its own size when it fits, centred
// over its caption, on black at 75% (T3's painted #3f3f3f over the light
// canvas), and its buttons and the veil each send their own click
// (ExpandedImageDialog.tsx; §3.10).
func TestGallery(t *testing.T) {
	th := NewTheme()
	g := &Gallery{Picture: solidPicture(t, 480, 300), Name: "plot.png", Index: 0, Count: 2}
	size := image.Pt(800, 600)
	draw := func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
		th.gallery(gtx, g)
		return D{Size: size}
	}
	img := drawn(t, size, 1, draw)
	// 480 by 300 in 800 by 600: 160 in, (600-300-8-16)/2 down.
	wantPixel(t, img, image.Pt(160+240, 138+150), rgb(0x3366cc), "the picture's middle")
	wantPixel(t, img, image.Pt(166, 144), rgb(0x3366cc), "inside the picture's corner, at its own size")
	if c := img.RGBAAt(10, 300); !near(c.R, 0x3f, 8) || !near(c.G, 0x3f, 8) || !near(c.B, 0x3f, 8) {
		t.Errorf("the veil is #%02x%02x%02x, want about #3f3f3f", c.R, c.G, c.B)
	}
	var r gioinput.Router
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(size), Source: r.Source(), Now: time.Now()}
		th.gallery(gtx, g)
		r.Frame(&ops)
		return th.Actions()
	}
	frame()
	for _, c := range []struct {
		at   image.Point
		want string
	}{{image.Pt(10, 10), "gallery.close"}, {image.Pt(400, 288), "gallery.image"},
		// The arrows are 32dp at the popup's sides, 4% in, halfway down.
		{image.Pt(768-16, 300), "gallery.next"}, {image.Pt(32+16, 300), "gallery.prev"},
		// The close button is 28dp at its top right.
		{image.Pt(768-14, 24+14), "gallery.close"}} {
		p := f32.Pt(float32(c.at.X), float32(c.at.Y))
		r.Queue(pointer.Event{Kind: pointer.Move, Position: p, Source: pointer.Mouse})
		frame()
		r.Queue(pointer.Event{Kind: pointer.Press, Position: p, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			pointer.Event{Kind: pointer.Release, Position: p, Source: pointer.Mouse})
		if acts := frame(); len(acts) != 1 || acts[0] != (Action{ID: c.want, Kind: "click"}) {
			t.Errorf("a click at %v sent %+v, want %s", c.at, acts, c.want)
		}
	}
}
