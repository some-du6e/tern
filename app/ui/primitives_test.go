package ui

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// Button heights and shapes follow T3's size table (docs/ui/02-primitives.md
// §3.1): default 32, sm 28, lg 36, xs 24, and the icon sizes are square.
func TestT3ButtonSizes(t *testing.T) {
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(400, 100)}}
	for _, c := range []struct {
		size   ButtonSize
		h      int
		square bool
	}{
		{BtnSizeDefault, 32, false}, {BtnSizeSm, 28, false}, {BtnSizeLg, 36, false}, {BtnSizeXs, 24, false},
		{BtnSizeIcon, 32, true}, {BtnSizeIconSm, 28, true}, {BtnSizeIconLg, 36, true}, {BtnSizeIconXs, 24, true},
	} {
		b := Button{Variant: BtnOutline, Size: c.size, Icon: "plus"}
		if !c.square {
			b.Label = "Label"
		}
		d := th.T3Button("", b)(gtx)
		if d.Size.Y != c.h || (c.square && d.Size.X != c.h) {
			t.Errorf("size %d: %v, want %d tall%s", c.size, d.Size, c.h, map[bool]string{true: " and square"}[c.square])
		}
	}
}

// An outline button is the popover colour inside its input-coloured border;
// a default one is primary.
func TestT3ButtonColors(t *testing.T) {
	th := NewTheme()
	img := drawn(t, image.Pt(200, 100), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 200, 100), 0)
		th.T3Button("", Button{Variant: BtnOutline, Size: BtnSizeDefault, Label: "Outline"})(gtx)
		op.Offset(image.Pt(0, 50)).Add(gtx.Ops)
		th.T3Button("", Button{Variant: BtnDefault, Size: BtnSizeDefault, Label: "Default"})(gtx)
		return D{}
	})
	wantPixel(t, img, image.Pt(4, 16), over(P.Popover, P.Background), "outline fill")
	wantPixel(t, img, image.Pt(0, 16), over(P.Input, P.Background), "outline border")
	wantPixel(t, img, image.Pt(4, 66), P.Primary, "default fill")
}
