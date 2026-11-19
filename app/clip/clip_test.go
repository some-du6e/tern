package clip

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// dib builds a 2x2 bitmap: header, then rows of pixels.
func dib(hsize int, h int32, bpp uint16, comp uint32, masks []uint32, rows ...[]byte) []byte {
	var b bytes.Buffer
	le := binary.LittleEndian
	hdr := make([]byte, hsize)
	le.PutUint32(hdr[0:], uint32(hsize))
	le.PutUint32(hdr[4:], 2)
	le.PutUint32(hdr[8:], uint32(h))
	le.PutUint16(hdr[12:], 1)
	le.PutUint16(hdr[14:], bpp)
	le.PutUint32(hdr[16:], comp)
	if hsize >= 56 {
		for i, m := range masks {
			le.PutUint32(hdr[40+4*i:], m)
		}
		masks = nil
	}
	b.Write(hdr)
	for _, m := range masks {
		binary.Write(&b, le, m)
	}
	for _, r := range rows {
		b.Write(r)
	}
	return b.Bytes()
}

func at(t *testing.T, img image.Image, x, y int) color.NRGBA {
	t.Helper()
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

func TestDIB24BottomUp(t *testing.T) {
	// Bottom row first, BGR, rows padded to 4 bytes.
	b := dib(40, 2, 24, 0, nil,
		[]byte{0, 0, 255, 0, 255, 0, 0, 0}, // y=1: red, green
		[]byte{255, 0, 0, 9, 9, 9, 0, 0})   // y=0: blue, grey
	img, err := DIB(b)
	if err != nil {
		t.Fatal(err)
	}
	if c := at(t, img, 0, 0); c != (color.NRGBA{0, 0, 255, 255}) {
		t.Fatalf("top left = %v, want blue", c)
	}
	if c := at(t, img, 0, 1); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Fatalf("bottom left = %v, want red", c)
	}
}

func TestDIBV5TopDownAlpha(t *testing.T) {
	m := []uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0xff000000}
	b := dib(124, -2, 32, 3, m,
		[]byte{0, 0, 255, 128, 0, 255, 0, 255},
		[]byte{255, 0, 0, 255, 0, 0, 0, 0})
	img, err := DIB(b)
	if err != nil {
		t.Fatal(err)
	}
	if c := at(t, img, 0, 0); c != (color.NRGBA{255, 0, 0, 128}) {
		t.Fatalf("top left = %v, want half-clear red", c)
	}
	if c := at(t, img, 1, 1); c.A != 0 {
		t.Fatalf("bottom right = %v, want clear", c)
	}
}

func TestDIBZeroAlphaIsOpaque(t *testing.T) {
	m := []uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0xff000000}
	b := dib(124, 2, 32, 0, m, make([]byte, 8), []byte{1, 2, 3, 0, 0, 0, 0, 0})
	img, err := DIB(b)
	if err != nil {
		t.Fatal(err)
	}
	if c := at(t, img, 0, 0); c != (color.NRGBA{3, 2, 1, 255}) {
		t.Fatalf("got %v", c)
	}
}

func TestDIBBitfieldsAfterHeader(t *testing.T) {
	b := dib(40, 2, 32, 3, []uint32{0x00ff0000, 0x0000ff00, 0x000000ff},
		make([]byte, 8), []byte{0, 0, 200, 0, 0, 0, 0, 0})
	img, err := DIB(b)
	if err != nil {
		t.Fatal(err)
	}
	if c := at(t, img, 0, 0); c != (color.NRGBA{200, 0, 0, 255}) {
		t.Fatalf("got %v", c)
	}
}

func TestDIBRejectsJunk(t *testing.T) {
	for _, b := range [][]byte{nil, make([]byte, 39), dib(40, 2, 24, 0, nil)} {
		if _, err := DIB(b); err == nil {
			t.Fatalf("accepted %d bytes", len(b))
		}
	}
}

func TestEncodeAndFiles(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	im, err := Encode(img, "pasted")
	if err != nil || im.Mime != "image/png" || im.Name != "pasted.png" {
		t.Fatalf("%v %+v", err, im)
	}
	if _, err := png.Decode(bytes.NewReader(im.Data)); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	shot := filepath.Join(dir, "Shot.PNG")
	os.WriteFile(shot, im.Data, 0o600)
	notes := filepath.Join(dir, "notes.txt")
	os.WriteFile(notes, []byte("hi"), 0o600)
	got, err := fromFiles([]string{notes, shot})
	if err != nil || len(got) != 1 || got[0].Name != "Shot.PNG" || got[0].Mime != "image/png" {
		t.Fatalf("%v %+v", err, got)
	}
	if _, err := fromFiles([]string{notes}); err != ErrNone {
		t.Fatalf("no images should be ErrNone, got %v", err)
	}
}
