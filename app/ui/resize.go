package ui

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/op/clip"
)

// A panel's edge can be dragged to resize it, like T3's. The handle is a
// strip a few dp wide over the edge; dragging it reports how far the edge
// moved.

// handle is one edge's drag state: whether the pointer is over it or
// dragging it, and a double click (which the owner clears).
type handle struct {
	tag       int
	press     float32 // where in the strip the drag started
	held      bool
	hovered   bool
	double    bool
	lastPress time.Time
}

// dragEdge makes the size-sized strip at the current offset a resize
// handle and returns how far it was dragged since the last frame, across
// (x) or up and down (y).
func (th *Theme) dragEdge(gtx C, h *handle, size image.Point, across bool) float32 {
	delta := float32(0)
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &h.tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Enter | pointer.Leave})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		at := e.Position.Y
		if across {
			at = e.Position.X
		}
		switch e.Kind {
		case pointer.Enter:
			h.hovered = true
		case pointer.Leave:
			h.hovered = false
		case pointer.Press:
			h.press, h.held = at, true
			h.double = gtx.Now.Sub(h.lastPress) < 400*time.Millisecond
			h.lastPress = gtx.Now
		case pointer.Drag:
			if h.held {
				delta += at - h.press
				h.press = at
			}
		case pointer.Release, pointer.Cancel:
			h.held = false
		}
	}
	// The strip moves with the edge by the next frame, and the pointer's
	// place in it with it.
	h.press -= delta
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if across {
		pointer.CursorColResize.Add(gtx.Ops)
	} else {
		pointer.CursorRowResize.Add(gtx.Ops)
	}
	event.Op(gtx.Ops, &h.tag)
	return delta
}
