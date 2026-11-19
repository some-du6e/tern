package ui

import (
	"image"
	"math"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// scrollArea is T3's ScrollArea for a box of known content (02-primitives
// §3.21): the wheel scrolls it, and its 6dp overlay thumb shows while the
// pointer's over it. Each is kept by id.
type scrollArea struct {
	y     int // how far the content has scrolled, in px
	wheel gesture.Scroll
	hover gesture.Hover
	over  bool
}

func (th *Theme) scrollArea(id string) *scrollArea {
	if th.in.scrolls == nil {
		th.in.scrolls = map[string]*scrollArea{}
	}
	s := th.in.scrolls[id]
	if s == nil {
		s = new(scrollArea)
		th.in.scrolls[id] = s
	}
	return s
}

// scrollTo scrolls area id just enough to show its content's rows [top,
// bottom) in a view h tall (scrollIntoView, block nearest).
func (th *Theme) scrollTo(id string, top, bottom, h int) {
	s := th.scrollArea(id)
	switch {
	case top < s.y:
		s.y = top
	case bottom > s.y+h:
		s.y = bottom - h
	}
}

// vscroll draws content, height tall, in a box size big scrolled by area
// id, and returns how far it has scrolled.
func (th *Theme) vscroll(gtx C, id string, size image.Point, height int, content op.CallOp) int {
	s := th.scrollArea(id)
	most := max(0, height-size.Y)
	s.y += s.wheel.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -s.y, Max: most - s.y})
	s.y = max(0, min(s.y, most))
	s.over = s.hover.Update(gtx.Source)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	s.hover.Add(gtx.Ops)
	s.wheel.Add(gtx.Ops)
	func() {
		defer op.Offset(image.Pt(0, -s.y)).Push(gtx.Ops).Pop()
		content.Add(gtx.Ops)
	}()
	if most > 0 && s.over {
		// The thumb: 6dp wide, 1dp in from the end, 4dp from the ends, at
		// least 16 tall.
		track := size.Y - 2*dp(gtx, 4)
		h := max(dp(gtx, 16), int(math.Round(float64(track)*float64(size.Y)/float64(height))))
		y := dp(gtx, 4) + (track-h)*s.y/most
		x := size.X - dp(gtx, 1) - dp(gtx, 6)
		thumb := rgb(0xd9d9d9)
		if dark {
			thumb = over(alpha(rgb(0xffffff), 0x14), P.Background)
		}
		fill(gtx, thumb, image.Rect(x, y, x+dp(gtx, 6), y+h), dp(gtx, 3))
	}
	return s.y
}
