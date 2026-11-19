package ui

import (
	"image"
	"math"
	"strconv"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// The image dialog (ExpandedImageDialog.tsx; docs/ui/06-timeline.md
// §3.10, port note 17): a message's picture whole over the window, the
// message's others a step away.

// Gallery is the open image dialog: the picture shown, decoded whole once
// it loads (its thumbnail until then), its name, and its place among the
// message's pictures.
type Gallery struct {
	Picture         Picture
	Name            string
	Index, Count    int
	Loading, Failed bool
}

// veil is the dialog's backdrop, black at 75%. Gio blends in linear light,
// where it takes black at 95.3% (1 - 0.25^2.2) to darken the window as
// the browser's 75% does.
var veil = alpha(rgb(0), 0xf3)

// gallery draws the image dialog over a window of size: the veil, which a
// click closes ("gallery.close"); the picture at its own size, or fitted
// into 92% of the window's width (96dp less from 640dp up, for the
// arrows) and the lesser of 86% of its height and its height less 160dp,
// radius 8 on the background with a 1dp ring at border/70 and
// shadow-2xl; its name and "(2/5)" 8dp under it, 12/16 in white at 80%;
// the 28dp close button at the dialog's top right; with more pictures,
// 32dp arrows at its sides ("gallery.prev", "gallery.next"). Escape and
// the arrow keys do the same.
func (th *Theme) gallery(gtx C, g *Gallery) {
	size := gtx.Constraints.Max
	th.galleryKeys(gtx)
	fill(gtx, veil, image.Rectangle{Max: size}, 0)
	func() {
		// The veil closes the dialog on a click, under the default cursor.
		c, _ := th.events(gtx, "gallery.close")
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		c.Add(gtx.Ops)
	}()
	pw, ph := size.X*92/100, size.Y*92/100
	popup := image.Rectangle{Min: image.Pt((size.X-pw)/2, (size.Y-ph)/2)}
	popup.Max = popup.Min.Add(image.Pt(pw, ph))
	mediaW := pw
	if size.X >= dp(gtx, 640) {
		mediaW -= dp(gtx, 96)
	}
	mediaH := min(size.Y*86/100, size.Y-dp(gtx, 160))
	caption := g.Name
	if g.Count > 1 {
		caption += " (" + strconv.Itoa(g.Index+1) + "/" + strconv.Itoa(g.Count) + ")"
	}
	text := th.recorded(within(gtx, mediaW), func(gtx C) D {
		return th.textLine(gtx, Txt{Size: 12, Color: rgb(0xcccccc)}, 16, caption)
	})
	// The picture, or why there's none: 192dp of black, as wide as 32rem.
	var frame image.Point
	p := g.Picture
	if p.ok && !g.Failed {
		nat := image.Pt(dp(gtx, float32(p.natural.X)), dp(gtx, float32(p.natural.Y)))
		k := min(1, float32(mediaW)/float32(nat.X), float32(mediaH)/float32(nat.Y))
		frame = image.Pt(max(1, int(float32(nat.X)*k)), max(1, int(float32(nat.Y)*k)))
	} else {
		frame = image.Pt(min(mediaW, dp(gtx, 512)), dp(gtx, 192))
	}
	gap := dp(gtx, 8)
	at := image.Pt((size.X-frame.X)/2, (size.Y-frame.Y-gap-text.Size.Y)/2)
	r := image.Rectangle{Min: at, Max: at.Add(frame)}
	rad := dp(gtx, 8)
	shadow(gtx, r, rad, []boxShadow{{0, 25, 50, -12, alpha(rgb(0), 0x40)}})
	switch {
	case p.ok && !g.Failed:
		func() {
			defer clip.UniformRRect(r, rad).Push(gtx.Ops).Pop()
			fill(gtx, P.Background, r, 0)
			p.cover(gtx, r)
		}()
		strokeRect(gtx, over(faded(P.Border, 0xb3), P.Background), r, rad, float32(max(1, dp(gtx, 1))))
	case g.Loading:
		fill(gtx, rgb(0), r, rad)
	default:
		fill(gtx, rgb(0), r, rad)
		strokeRect(gtx, over(faded(P.Border, 0xb3), rgb(0)), r, rad, float32(max(1, dp(gtx, 1))))
		msg := th.recorded(within(gtx, frame.X-dp(gtx, 48)), func(gtx C) D {
			return th.textLine(gtx, Txt{Size: 14, Color: rgb(0xffffff)}, 20, "This image could not be loaded.")
		})
		mo := op.Offset(r.Min.Add(image.Pt((frame.X-msg.Size.X)/2, (frame.Y-msg.Size.Y)/2))).Push(gtx.Ops)
		msg.call.Add(gtx.Ops)
		mo.Pop()
	}
	// The picture takes its own clicks: only the veil closes the dialog.
	func() {
		defer op.Offset(r.Min).Push(gtx.Ops).Pop()
		th.hit(gtx, "gallery.image", frame)
	}()
	to := op.Offset(image.Pt((size.X-text.Size.X)/2, r.Max.Y+gap)).Push(gtx.Ops)
	text.call.Add(gtx.Ops)
	to.Pop()
	th.galleryButton(gtx, "gallery.close", "x", 16, image.Pt(popup.Max.X-dp(gtx, 28), popup.Min.Y), 28, true)
	if g.Count > 1 {
		mid := popup.Min.Y + (ph-dp(gtx, 32))/2
		th.galleryButton(gtx, "gallery.prev", "chevron-left", 20, image.Pt(popup.Min.X, mid), 32, false)
		th.galleryButton(gtx, "gallery.next", "chevron-right", 20, image.Pt(popup.Max.X-dp(gtx, 32), mid), 32, false)
	}
}

// galleryButton is the dialog's media-close button, black at 65% in a 1dp
// ring of white at 20%, or its media-navigation one, white at 10% while
// hovered, a size dp square at at; its icon white, at 90% on the arrows
// at rest. They sit on the veil, near black, so they're painted on black.
func (th *Theme) galleryButton(gtx C, id, icon string, iconSize float32, at image.Point, size float32, closer bool) {
	c, _ := th.events(gtx, id)
	defer op.Offset(at).Push(gtx.Ops).Pop()
	s := dp(gtx, size)
	r := image.Rectangle{Max: image.Pt(s, s)}
	rad, px := dp(gtx, 8), max(1, dp(gtx, 1))
	black := rgb(0)
	switch {
	case closer:
		fill(gtx, over(alpha(rgb(0xffffff), 0x33), black), r.Inset(-px), rad+px)
		fill(gtx, black, r, rad)
	case c.Hovered():
		fill(gtx, over(alpha(rgb(0xffffff), 0x1a), black), r, rad)
	}
	col := rgb(0xffffff)
	if !closer && !c.Hovered() {
		col = over(alpha(col, 0xe6), black)
	}
	ic := dp(gtx, iconSize)
	io := op.Offset(image.Pt((s-ic)/2, (s-ic)/2)).Push(gtx.Ops)
	if icon == "chevron-left" {
		// Lucide's left chevron is its right one turned round.
		h := float32(ic) / 2
		turn := op.Affine(f32.Affine2D{}.Rotate(f32.Pt(h, h), math.Pi)).Push(gtx.Ops)
		LucideW("chevron-right", col, iconSize)(gtx)
		turn.Pop()
	} else {
		LucideW(icon, col, iconSize)(gtx)
	}
	io.Pop()
	th.hit(gtx, id, r.Max)
}

// galleryKeys turns Escape and the arrows into the dialog's buttons.
func (th *Theme) galleryKeys(gtx C) {
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape}, key.Filter{Name: key.NameLeftArrow}, key.Filter{Name: key.NameRightArrow})
		if !ok {
			return
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		id := map[key.Name]string{key.NameEscape: "gallery.close", key.NameLeftArrow: "gallery.prev", key.NameRightArrow: "gallery.next"}[ke.Name]
		th.push(Action{ID: id, Kind: "click"})
	}
}
