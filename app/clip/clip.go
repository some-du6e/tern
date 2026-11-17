// Package clip reads images off the system clipboard. Gio's clipboard only
// carries text.
package clip

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// MaxBytes is the server's attachment limit.
const MaxBytes = 10 << 20

// Image is one picture ready to upload.
type Image struct {
	Name, Mime string
	Data       []byte
}

// ErrNone means the clipboard holds no image.
var ErrNone = errors.New("no image on the clipboard")

// Images returns the clipboard's picture, or the image files copied in
// Explorer or the Finder. It's ErrNone when there are neither.
func Images() ([]Image, error) { return read() }

var mimes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".gif": "image/gif", ".webp": "image/webp",
}

// fromFiles loads the image files among paths and skips the rest.
func fromFiles(paths []string) ([]Image, error) {
	var out []Image
	for _, p := range paths {
		m := mimes[strings.ToLower(filepath.Ext(p))]
		if m == "" {
			continue
		}
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		if st.Size() > MaxBytes {
			return out, errors.New(filepath.Base(p) + " is over 10 MB")
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return out, err
		}
		out = append(out, Image{Name: filepath.Base(p), Mime: m, Data: b})
	}
	if len(out) == 0 {
		return nil, ErrNone
	}
	return out, nil
}

// fromPNG is a clipboard's PNG, shrunk to fit the limit when it's over.
func fromPNG(b []byte) ([]Image, error) {
	if len(b) <= MaxBytes {
		return []Image{{Name: "pasted.png", Mime: "image/png", Data: b}}, nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, ErrNone
	}
	im, err := Encode(img, "pasted")
	return []Image{im}, err
}

// Encode turns a decoded picture into a PNG, or a JPEG when the PNG would be
// over the limit (big photos and 4K screenshots).
func Encode(img image.Image, name string) (Image, error) {
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, img); err != nil {
		return Image{}, err
	}
	if buf.Len() <= MaxBytes {
		return Image{Name: name + ".png", Mime: "image/png", Data: buf.Bytes()}, nil
	}
	for _, q := range []int{90, 75, 60} {
		buf.Reset()
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return Image{}, err
		}
		if buf.Len() <= MaxBytes {
			return Image{Name: name + ".jpg", Mime: "image/jpeg", Data: buf.Bytes()}, nil
		}
	}
	return Image{}, errors.New("that image is too big to send")
}

// DIB decodes a CF_DIB or CF_DIBV5 block: a BITMAPINFOHEADER (or V4/V5)
// and the pixels, 24 or 32 bits, uncompressed or with bit fields.
func DIB(b []byte) (image.Image, error) {
	if len(b) < 40 {
		return nil, errors.New("short bitmap header")
	}
	le := binary.LittleEndian
	hsize := int(le.Uint32(b[0:]))
	w := int(int32(le.Uint32(b[4:])))
	h := int(int32(le.Uint32(b[8:])))
	bpp := int(le.Uint16(b[14:]))
	comp := le.Uint32(b[16:])
	used := int(le.Uint32(b[32:]))
	if hsize < 40 || hsize > len(b) || w <= 0 || h == 0 || w > 1<<15 || h > 1<<15 || h < -(1<<15) {
		return nil, errors.New("bad bitmap header")
	}
	topDown := h < 0
	if topDown {
		h = -h
	}
	if bpp != 24 && bpp != 32 {
		return nil, errors.New("unsupported bitmap depth")
	}
	// Default masks are BGRA bytes.
	masks := [4]uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0}
	off := hsize
	switch comp {
	case 0: // BI_RGB
	case 3, 6: // BI_BITFIELDS, BI_ALPHABITFIELDS
		if hsize >= 52 {
			for i := 0; i < 3; i++ {
				masks[i] = le.Uint32(b[40+4*i:])
			}
			if hsize >= 56 {
				masks[3] = le.Uint32(b[52:])
			}
		} else {
			n := 3
			if comp == 6 {
				n = 4
			}
			if len(b) < 40+4*n {
				return nil, errors.New("short bitmap masks")
			}
			for i := 0; i < n; i++ {
				masks[i] = le.Uint32(b[40+4*i:])
			}
			off += 4 * n
		}
	default:
		return nil, errors.New("compressed bitmaps aren't supported")
	}
	off += 4 * used // a palette, which 24/32-bit images rarely carry
	if bpp == 32 && comp == 0 && hsize >= 56 {
		// V4/V5 headers carry an alpha mask even with BI_RGB.
		masks[3] = le.Uint32(b[52:])
	}
	stride := (w*bpp/8 + 3) &^ 3
	if off < 0 || off+stride*h > len(b) {
		return nil, errors.New("short bitmap pixels")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	anyAlpha := false
	for y := 0; y < h; y++ {
		row := b[off+stride*y:]
		dy := h - 1 - y
		if topDown {
			dy = y
		}
		for x := 0; x < w; x++ {
			var c color.NRGBA
			if bpp == 24 {
				p := row[3*x:]
				c = color.NRGBA{p[2], p[1], p[0], 0xff}
			} else {
				v := le.Uint32(row[4*x:])
				c = color.NRGBA{field(v, masks[0]), field(v, masks[1]), field(v, masks[2]), 0xff}
				if masks[3] != 0 {
					c.A = field(v, masks[3])
					anyAlpha = anyAlpha || c.A != 0
				}
			}
			img.SetNRGBA(x, dy, c)
		}
	}
	if bpp == 32 && masks[3] != 0 && !anyAlpha {
		// Plenty of apps leave alpha at zero. Read that as opaque.
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 0xff
		}
	}
	return img, nil
}

// field pulls the channel under mask out of v, scaled to 8 bits.
func field(v, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := 0
	for mask&1 == 0 {
		mask >>= 1
		shift++
	}
	x := (v >> shift) & mask
	if mask == 0xff {
		return uint8(x)
	}
	return uint8(x * 255 / mask)
}
