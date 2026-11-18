package ui

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// T3's primitives (apps/web/src/components/ui, docs/ui/02-primitives.md),
// at the ≥640px sizes T3 uses on a desktop window.

// ButtonVariant is a Button's look, T3's buttonVariants.
type ButtonVariant int

const (
	BtnDefault ButtonVariant = iota
	BtnDestructive
	BtnDestructiveOutline
	BtnOutline
	BtnSecondary
	BtnGhost
	BtnGhostMuted
	BtnGhostDestructive
	BtnLink
)

// ButtonSize is a Button's size.
type ButtonSize int

const (
	BtnSizeDefault ButtonSize = iota
	BtnSizeSm
	BtnSizeLg
	BtnSizeXs
	BtnSizeIcon
	BtnSizeIconSm
	BtnSizeIconLg
	BtnSizeIconXs
)

// buttonSize is one size's geometry (§3.1): height, padding x (4n - 1, so
// text sits on the grid once the 1dp border counts), gap, font and icon.
type buttonSize struct {
	h, padX, gap float32
	text, line   unit.Sp
	icon         float32
	square       bool
}

var buttonSizes = map[ButtonSize]buttonSize{
	BtnSizeDefault: {32, 11, 8, 14, 20, 16, false},
	BtnSizeSm:      {28, 9, 6, 14, 20, 16, false},
	BtnSizeLg:      {36, 13, 8, 14, 20, 16, false},
	BtnSizeXs:      {24, 7, 4, 12, 16, 14, false},
	BtnSizeIcon:    {32, 0, 0, 14, 20, 16, true},
	BtnSizeIconSm:  {28, 0, 0, 14, 20, 16, true},
	BtnSizeIconLg:  {36, 0, 0, 14, 20, 16, true},
	BtnSizeIconXs:  {24, 0, 0, 14, 20, 14, true},
}

// Button is T3's Button: a variant, a size, a lucide icon, a label.
type Button struct {
	Variant  ButtonVariant
	Size     ButtonSize
	Icon     string
	Label    string
	Disabled bool
	// Pressed is data-pressed, a trigger whose popup is open: it shows the
	// hover colours and doesn't scale.
	Pressed bool
	// On is the surface under the button, for its translucent colours;
	// zero means the window background.
	On color.NRGBA
}

// buttonColors is a variant's fill, text, border and icon at rest and on
// hover (§3.1, §4.1).
type buttonColors struct {
	bg, bgHover, text, textHover, border, borderHover, icon color.NRGBA
	shadow                                                  bool // outline's 0 1px 2px and lip
	solid                                                   bool // default and destructive: inset highlight, coloured drop shadow
}

func (b Button) colors() buttonColors {
	none := color.NRGBA{}
	switch b.Variant {
	case BtnDefault:
		return buttonColors{bg: P.Primary, bgHover: alpha(P.Primary, 0xe6), text: P.PrimaryForeground, border: P.Primary, icon: P.PrimaryForeground, solid: true}
	case BtnDestructive:
		return buttonColors{bg: P.Destructive, bgHover: alpha(P.Destructive, 0xe6), text: rgb(0xffffff), border: P.Destructive, icon: rgb(0xffffff), solid: true}
	case BtnDestructiveOutline:
		bg := P.Popover
		if dark {
			bg = alpha(rgb(0xffffff), 0x09)
		}
		return buttonColors{bg: bg, bgHover: alpha(P.Destructive, 0x0a), text: P.DestructiveForeground, border: P.Input,
			borderHover: alpha(P.Destructive, 0x52), icon: P.DestructiveForeground, shadow: true}
	case BtnOutline:
		bg, hover := P.Popover, alpha(P.Accent, 0x80)
		if dark {
			bg, hover = alpha(rgb(0xffffff), 0x09), alpha(rgb(0xffffff), 0x12)
		}
		return buttonColors{bg: bg, bgHover: hover, text: P.Foreground, border: P.Input, icon: P.ContrastMutedForeground, shadow: true}
	case BtnSecondary:
		return buttonColors{bg: P.Secondary, bgHover: alpha(P.Secondary, uint8(int(P.Secondary.A)*9/10)), text: P.SecondaryForeground, icon: P.SecondaryForeground}
	case BtnGhost:
		return buttonColors{bgHover: P.Accent, text: P.Foreground, icon: P.ContrastMutedForeground}
	case BtnGhostMuted:
		return buttonColors{bgHover: P.Accent, text: P.MutedForeground, textHover: P.Foreground, icon: P.MutedForeground}
	case BtnGhostDestructive:
		return buttonColors{bgHover: P.Accent, text: P.MutedForeground, textHover: P.Destructive, icon: P.MutedForeground}
	case BtnLink:
		return buttonColors{text: bodyText(), icon: bodyText()}
	}
	return buttonColors{bg: none}
}

// bodyText is the text colour elements inherit without a colour class: the
// boot style's #262626 / #f5f5f5, not the foreground token.
func bodyText() color.NRGBA {
	if dark {
		return rgb(0xf5f5f5)
	}
	return rgb(0x262626)
}

// T3Button draws b as click target id.
func (th *Theme) T3Button(id string, b Button) W {
	return func(gtx C) D {
		sz := buttonSizes[b.Size]
		on := b.On
		if on.A == 0 {
			on = P.Background
		}
		cl := b.colors()
		hovered, pressed := false, false
		if !b.Disabled && id != "" {
			c, _ := th.events(gtx, id)
			hovered, pressed = c.Hovered(), c.Pressed()
		}
		hot := hovered || b.Pressed
		bg, text, border, icon := cl.bg, cl.text, cl.border, cl.icon
		if hot {
			if cl.bgHover.A != 0 {
				bg = cl.bgHover
			}
			if cl.textHover.A != 0 {
				text, icon = cl.textHover, cl.textHover
			}
			if cl.borderHover.A != 0 {
				border = cl.borderHover
			}
		}

		// Lay the content out first: the box is as wide as it needs.
		m := op.Record(gtx.Ops)
		cg := gtx
		cg.Constraints.Min = image.Point{}
		content := layout.Flex{Alignment: layout.Middle}.Layout(cg,
			layout.Rigid(func(gtx C) D {
				if b.Icon == "" {
					return D{}
				}
				// Icons sit 2dp into the padding on both sides.
				return pad(0, -2, 0, -2).Layout(gtx, LucideW(b.Icon, icon, sz.icon))
			}),
			layout.Rigid(func(gtx C) D {
				if b.Label == "" || b.Icon == "" {
					return D{}
				}
				return D{Size: image.Pt(dp(gtx, sz.gap), 0)}
			}),
			layout.Rigid(func(gtx C) D {
				if b.Label == "" {
					return D{}
				}
				return th.Text(gtx, Txt{Size: sz.text, LineH: float32(sz.line) / float32(sz.text), Font: UI(font.Medium), Color: text, MaxLines: 1}, b.Label)
			}),
		)
		call := m.Stop()
		h := dp(gtx, sz.h)
		w := h
		if !sz.square {
			w = content.Size.X + 2*dp(gtx, sz.padX) + 2*max(1, dp(gtx, 1))
		}
		r := image.Rectangle{Max: image.Pt(w, h)}
		rad := dp(gtx, 8)
		if b.Disabled {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		}
		// :active shrinks a button that doesn't open a popup to 97%.
		if pressed && !b.Pressed {
			c := f32.Pt(float32(w)/2, float32(h)/2)
			defer op.Affine(f32.Affine2D{}.Scale(c, f32.Pt(0.97, 0.97))).Push(gtx.Ops).Pop()
		}
		px := max(1, dp(gtx, 1))
		switch {
		case cl.solid:
			if !pressed && !b.Disabled {
				shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(bg, 0x3d)}})
			}
			fill(gtx, over(bg, on), r, rad)
			// The inset highlight along the top, black 8% while pressed.
			hl := over(alpha(rgb(0xffffff), 0x29), over(bg, on))
			if pressed {
				hl = over(alpha(rgb(0), 0x14), over(bg, on))
			}
			fill(gtx, hl, image.Rect(rad/2, px, w-rad/2, 2*px), 0)
		case cl.shadow:
			if !hot && !b.Disabled {
				shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}})
			}
			fill(gtx, over(bg, on), r, rad)
			strokeRect(gtx, over(border, on), r, rad, float32(px))
			if !hot && !b.Disabled {
				th.lip(gtx, r, rad, on)
			}
		default:
			if bg.A != 0 {
				fill(gtx, over(bg, on), r, rad)
			}
		}
		x := (w - content.Size.X) / 2
		if sz.square {
			x = (w - content.Size.X) / 2
		}
		off := op.Offset(image.Pt(x, (h-content.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		if !b.Disabled && id != "" {
			th.hit(gtx, id, r.Max)
		}
		return D{Size: r.Max}
	}
}

// lip is T3's bordered-control highlight: a 1dp line along the bottom edge
// in light (black 4%), the top in dark (white 6%), inside the corners.
func (th *Theme) lip(gtx C, r image.Rectangle, rad int, on color.NRGBA) {
	px := max(1, dp(gtx, 1))
	if dark {
		hline(gtx, over(alpha(rgb(0xffffff), 0x0f), on), r.Min.Y+px, r.Min.X+rad, r.Max.X-rad)
		return
	}
	hline(gtx, over(alpha(rgb(0), 0x0a), on), r.Max.Y-2*px, r.Min.X+rad, r.Max.X-rad)
}

// T3Kbd is T3's Kbd: 20 tall, at least 20 wide, padded 4, radius 4, 12/16
// medium in muted foreground on foreground at 7% (§3.3).
func (th *Theme) T3Kbd(label string, on color.NRGBA) W {
	return func(gtx C) D {
		if on.A == 0 {
			on = P.Background
		}
		m := op.Record(gtx.Ops)
		td := th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.MutedForeground, MaxLines: 1}, label)
		call := m.Stop()
		h := dp(gtx, 20)
		w := max(h, td.Size.X+2*dp(gtx, 4))
		fill(gtx, over(alpha(P.Foreground, 0x12), on), image.Rect(0, 0, w, h), dp(gtx, 4))
		defer op.Offset(image.Pt((w-td.Size.X)/2, (h-td.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return D{Size: image.Pt(w, h)}
	}
}

// BadgeVariant is a Badge's tone (§3.4).
type BadgeVariant int

const (
	BadgeDefault BadgeVariant = iota
	BadgeDestructive
	BadgeError
	BadgeInfo
	BadgeSuccess
	BadgeWarning
	BadgeOutline
	BadgeSecondary
)

func (v BadgeVariant) colors() (bg, text, border color.NRGBA) {
	tint := func(c color.NRGBA) color.NRGBA {
		if dark {
			return alpha(c, 0x29) // 16%
		}
		return alpha(c, 0x14) // 8%
	}
	switch v {
	case BadgeDestructive:
		return P.Destructive, rgb(0xffffff), color.NRGBA{}
	case BadgeError:
		return tint(P.Destructive), P.DestructiveForeground, color.NRGBA{}
	case BadgeInfo:
		return tint(P.Info), P.InfoForeground, color.NRGBA{}
	case BadgeSuccess:
		return tint(P.Success), P.SuccessForeground, color.NRGBA{}
	case BadgeWarning:
		return tint(P.Warning), P.WarningForeground, color.NRGBA{}
	case BadgeOutline:
		bg := P.Background
		if dark {
			bg = alpha(rgb(0xffffff), 0x09)
		}
		return bg, P.Foreground, P.Input
	case BadgeSecondary:
		return P.Secondary, P.SecondaryForeground, color.NRGBA{}
	}
	return P.Primary, P.PrimaryForeground, color.NRGBA{}
}

// BadgeSize is a Badge's size (§3.4).
type BadgeSize int

const (
	BadgeSizeDefault BadgeSize = iota // 18 tall, 12/16
	BadgeSizeSm                       // 16 tall, 10/10
)

// metrics is the size's height and type.
func (sz BadgeSize) metrics() (h float32, text, line unit.Sp) {
	if sz == BadgeSizeSm {
		return 16, 10, 10
	}
	return 18, 12, 16
}

// badgeSize is how big a badge with label comes out.
func (th *Theme) badgeSize(gtx C, sz BadgeSize, label string) image.Point {
	m := op.Record(gtx.Ops)
	d := th.T3Badge(BadgeDefault, sz, "", label, P.Background)(gtx)
	m.Stop()
	return d.Size
}

// T3Badge is T3's Badge: padded 3, radius 4, medium type, with an optional
// 12dp icon at 80%, in the default (18 tall, 12/16) or sm (16, 10/10) size.
func (th *Theme) T3Badge(v BadgeVariant, sz BadgeSize, icon, label string, on color.NRGBA) W {
	return func(gtx C) D {
		if on.A == 0 {
			on = P.Background
		}
		bg, text, border := v.colors()
		bh, ts, tl := sz.metrics()
		m := op.Record(gtx.Ops)
		cg := gtx
		cg.Constraints.Min = image.Point{}
		cd := layout.Flex{Alignment: layout.Middle}.Layout(cg,
			layout.Rigid(func(gtx C) D {
				if icon == "" {
					return D{}
				}
				return pad(0, 4, 0, 0).Layout(gtx, LucideW(icon, alpha(text, 0xcc), 12))
			}),
			layout.Rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: ts, LineH: float32(tl) / float32(ts), Font: UI(font.Medium), Color: text, MaxLines: 1}, label)
			}),
		)
		call := m.Stop()
		px := max(1, dp(gtx, 1))
		h := dp(gtx, bh)
		w := max(h, cd.Size.X+2*dp(gtx, 3)+2*px)
		r := image.Rect(0, 0, w, h)
		fill(gtx, over(bg, on), r, dp(gtx, 4))
		if border.A != 0 {
			strokeRect(gtx, over(border, on), r, dp(gtx, 4), float32(px))
		}
		defer op.Offset(image.Pt((w-cd.Size.X)/2, (h-cd.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return D{Size: r.Max}
	}
}

// T3Spinner is T3's Spinner: lucide's loader-circle at stroke 1.75,
// turning once a second. phase is in seconds.
func T3Spinner(col color.NRGBA, sizeDp float32, phase float64) W {
	return func(gtx C) D {
		s := float32(dp(gtx, sizeDp))
		a := float32(math.Mod(phase, 1) * 2 * math.Pi)
		c := f32.Pt(s/2, s/2)
		defer op.Affine(f32.Affine2D{}.Rotate(c, a)).Push(gtx.Ops).Pop()
		return LucideW("loader-circle", col, sizeDp)(gtx)
	}
}
