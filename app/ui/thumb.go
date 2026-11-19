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

// thumbPx is the stored thumbnail's side: a 64dp tile at 2x.
const thumbPx = 128

// Thumb is a picture, shrunk once for its tile and chip.
type Thumb struct {
	N    int    // its chip's number when the text holds ChipRune(N); 0 for a tile alone
	Name string // the file's name
	Size int    // and its size in bytes
	op   paint.ImageOp
	ok   bool
	data []byte // the file, for the picture at full size
}

// NewThumb decodes an image file's bytes and crops it to a square
// thumbnail. A picture it can't decode gets the placeholder tile.
func NewThumb(data []byte) Thumb {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Thumb{Size: len(data)}
	}
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	if side == 0 {
		return Thumb{Size: len(data)}
	}
	// The middle square, like a photo grid (object-cover).
	sq := image.Rect(0, 0, side, side).Add(b.Min).Add(image.Pt((b.Dx()-side)/2, (b.Dy()-side)/2))
	dst := image.NewRGBA(image.Rect(0, 0, thumbPx, thumbPx))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, sq, draw.Src, nil)
	return Thumb{Size: len(data), op: paint.NewImageOp(dst), ok: true, data: data}
}

// drawThumb paints t sizePx square at the origin, rounded rad, or does
// nothing while it has no picture.
func drawThumb(gtx C, t Thumb, sizePx, rad int) bool {
	if !t.ok {
		return false
	}
	defer clip.UniformRRect(image.Rect(0, 0, sizePx, sizePx), rad).Push(gtx.Ops).Pop()
	k := float32(sizePx) / thumbPx
	defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(k, k))).Push(gtx.Ops).Pop()
	t.op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return true
}

// pasted draws a sent message's picture i as a 52dp thumbnail, numbered
// to match its marker in the text.
func (th *Theme) pasted(t Thumb, i int) W {
	return func(gtx C) D {
		s := dp(gtx, 52)
		r := image.Rect(0, 0, s, s)
		if !drawThumb(gtx, t, s, dp(gtx, 6)) {
			fill(gtx, over(P.Muted, P.Background), r, dp(gtx, 6))
			off := op.Offset(image.Pt((s-dp(gtx, 16))/2, (s-dp(gtx, 16))/2)).Push(gtx.Ops)
			LucideW("image", P.MutedForeground, 16)(gtx)
			off.Pop()
		}
		strokeRect(gtx, ColBorderHi, r, dp(gtx, 6), float32(max(1, dp(gtx, 1))))
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
		return D{Size: r.Max}
	}
}
