package ui

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strconv"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// thumbPx is the stored thumbnail's side: 52 dp at 2x.
const thumbPx = 104

// Thumb is a pasted image, shrunk once for the composer.
type Thumb struct {
	N  int // its number, 1 to 9: the composer's text holds ChipRune(N) for it
	op paint.ImageOp
	ok bool
}

// NewThumb decodes an image file's bytes and crops it to a square
// thumbnail. A picture it can't decode gets the placeholder tile.
func NewThumb(data []byte) Thumb {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Thumb{}
	}
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	if side == 0 {
		return Thumb{}
	}
	// The middle square, like a photo grid.
	sq := image.Rect(0, 0, side, side).Add(b.Min).Add(image.Pt((b.Dx()-side)/2, (b.Dy()-side)/2))
	dst := image.NewRGBA(image.Rect(0, 0, thumbPx, thumbPx))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, sq, draw.Src, nil)
	return Thumb{op: paint.NewImageOp(dst), ok: true}
}

// pasted draws thumbnail i of the composer, with its remove button when
// remove is set.
func (th *Theme) pasted(t Thumb, i int, remove bool) W {
	return func(gtx C) D {
		s := dp(gtx, 52)
		r := image.Rect(0, 0, s, s)
		if !t.ok {
			thumb(i)(gtx)
		} else {
			cl := clip.UniformRRect(r, dp(gtx, 6)).Push(gtx.Ops)
			scale := float32(s) / thumbPx
			st := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(scale, scale))).Push(gtx.Ops)
			t.op.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			st.Pop()
			cl.Pop()
			strokeRect(gtx, ColBorderHi, r, dp(gtx, 6), float32(max(1, dp(gtx, 1))))
		}
		if t.N != 0 {
			// The number, bottom left, to match the marker in the text.
			m := op.Record(gtx.Ops)
			ld := th.Text(gtx, Txt{Size: SizeSmall, Color: ColText}, strconv.Itoa(t.N))
			call := m.Stop()
			px, py := dp(gtx, 5), dp(gtx, 1)
			bw, bh := ld.Size.X+2*px, ld.Size.Y+2*py
			at := op.Offset(image.Pt(dp(gtx, 3), s-bh-dp(gtx, 3))).Push(gtx.Ops)
			fill(gtx, alpha(ColWindow, 0xd8), image.Rect(0, 0, bw, bh), bh/2)
			tx := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			tx.Pop()
			at.Pop()
		}
		if !remove {
			return D{Size: r.Max}
		}
		// The × sits in the top-right corner.
		b := dp(gtx, 18)
		off := op.Offset(image.Pt(s-b-dp(gtx, 3), dp(gtx, 3))).Push(gtx.Ops)
		th.Click("image.remove:"+strconv.Itoa(i), func(gtx C) D {
			circle(gtx, alpha(ColWindow, 0xd8), image.Pt(b/2, b/2), b/2)
			off := op.Offset(image.Pt((b-dp(gtx, 12))/2, (b-dp(gtx, 12))/2)).Push(gtx.Ops)
			DrawIcon(gtx, IconClose, ColText, 12)
			off.Pop()
			return D{Size: image.Pt(b, b)}
		})(gtx)
		off.Pop()
		return D{Size: r.Max}
	}
}
