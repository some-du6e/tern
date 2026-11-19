package ui

import (
	"bytes"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// T3's image dialog (ExpandedImageDialog.tsx, docs/ui/06-timeline.md
// §3): one picture as large as the window allows, over a black 75%
// backdrop, framed and shadowed, its name under it in white at 80% and a
// close button at the top right. Escape, the button or a click outside
// it closes the dialog.

// imagePreview is the picture the dialog shows, decoded once.
type imagePreview struct {
	thumb Thumb
	op    paint.ImageOp
	size  image.Point
}

// openPreview opens the dialog on t, moving the focus out of the prompt
// so Escape reaches it.
func (th *Theme) openPreview(t Thumb) {
	src, _, err := image.Decode(bytes.NewReader(t.data))
	if err != nil {
		return
	}
	th.in.preview = &imagePreview{thumb: t, op: paint.NewImageOp(src), size: src.Bounds().Size()}
	th.in.blur = true
}

// closePreview closes the dialog and gives the prompt the focus back.
func (th *Theme) closePreview() {
	th.in.preview = nil
	th.Focus("composer")
}

// imageDialog draws the open picture over the whole window.
func (th *Theme) imageDialog(gtx C) {
	p := th.in.preview
	if p == nil {
		return
	}
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			th.closePreview()
			return
		}
	}
	win := gtx.Constraints.Max
	if clicked, _ := th.localClick(gtx, "preview.backdrop", win); clicked {
		th.closePreview()
		return
	}
	// Black at 75%, as a browser paints it over the page.
	scrim := rgb(0)
	scrim.A = gioAlpha(scrim, P.Background, over(alpha(scrim, 0xbf), P.Background))
	fill(gtx, scrim, image.Rectangle{Max: win}, 0)
	// The popup is 92% of the window; the picture keeps 96dp for the
	// arrows and stays under 86% of the window below the title bar.
	box := image.Pt(win.X*92/100, win.Y*92/100)
	maxW, maxH := box.X-dp(gtx, 96), int(float32(win.Y-dp(gtx, topBarH))*0.86)
	w, h := p.size.X, p.size.Y
	if w > maxW {
		w, h = maxW, h*maxW/max(1, w)
	}
	if h > maxH {
		w, h = w*maxH/max(1, h), maxH
	}
	w, h = max(1, w), max(1, h)
	caption := record(gtx, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: over(alpha(rgb(0xffffff), 0xcc), over(alpha(rgb(0), 0xbf), P.Background)), MaxLines: 1}, p.thumb.Name)
	})
	gap := dp(gtx, 8)
	at := image.Pt((win.X-w)/2, (win.Y-h-gap-caption.Size.Y)/2)
	r := image.Rectangle{Min: at, Max: at.Add(image.Pt(w, h))}
	rad := dp(gtx, 8)
	shadow(gtx, r, rad, []boxShadow{{0, 25, 50, -12, alpha(rgb(0), 0x40)}})
	fill(gtx, P.Background, r, rad)
	func() {
		defer clip.UniformRRect(r, rad).Push(gtx.Ops).Pop()
		defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(float32(w)/float32(p.size.X), float32(h)/float32(p.size.Y))).Offset(f32.Pt(float32(at.X), float32(at.Y)))).Push(gtx.Ops).Pop()
		p.op.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}()
	strokeRect(gtx, alpha(P.Border, uint8(int(P.Border.A)*7/10)), r, rad, float32(max(1, dp(gtx, 1))))
	// The picture itself takes its clicks, so only the backdrop closes it.
	func() {
		defer op.Offset(r.Min).Push(gtx.Ops).Pop()
		th.localClick(gtx, "preview.image", r.Size())
	}()
	func() {
		defer op.Offset(image.Pt((win.X-caption.Size.X)/2, r.Max.Y+gap)).Push(gtx.Ops).Pop()
		caption.call.Add(gtx.Ops)
	}()
	// The close button, media-close at icon-sm, at the popup's top right.
	bx := image.Pt((win.X+box.X)/2-dp(gtx, 28), (win.Y-box.Y)/2)
	defer op.Offset(bx).Push(gtx.Ops).Pop()
	if clicked, hovered := th.localClick(gtx, "preview.close", image.Pt(dp(gtx, 28), dp(gtx, 28))); clicked {
		th.closePreview()
	} else {
		mediaClose(gtx, 28, 16, hovered, over(alpha(rgb(0), 0xbf), P.Background))
	}
}

// mediaClose is the media-close button variant at size dp with its x at
// icon dp, over on, what's under it (a picture's average colour): black
// at 65% (80% hovered), a white x, and outside the box its white 20% ring
// and shadow-sm. Gio blends in linear light, so each translucent colour
// goes on at the alpha that paints over on what Chrome's sRGB blend
// would.
func mediaClose(gtx C, size, icon float32, hovered bool, on color.NRGBA) {
	b := dp(gtx, size)
	br := image.Rect(0, 0, b, b)
	rad := dp(gtx, 8)
	px := max(1, dp(gtx, 1))
	as := func(c color.NRGBA) color.NRGBA { return alpha(c, gioAlpha(c, on, over(c, on))) }
	shadowOn(gtx, br, rad, on, []boxShadow{{0, 1, 3, 0, alpha(rgb(0), 0x1a)}, {0, 1, 2, -1, alpha(rgb(0), 0x1a)}})
	strokeRect(gtx, as(alpha(rgb(0xffffff), 0x33)), br.Inset(-px), rad+px, float32(px))
	bg := alpha(rgb(0), 0xa6)
	if hovered {
		bg = alpha(rgb(0), 0xcc)
	}
	fill(gtx, as(bg), br, rad)
	defer op.Offset(image.Pt((b-dp(gtx, icon))/2, (b-dp(gtx, icon))/2)).Push(gtx.Ops).Pop()
	LucideW("x", rgb(0xffffff), icon)(gtx)
}
