package ui

import (
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// AlertVariant is an Alert's tone (alert.tsx:7-29, docs/ui/11-empty-auth.md
// §2.17, §3.5).
type AlertVariant int

const (
	AlertDefault AlertVariant = iota
	AlertError
	AlertInfo
	AlertSuccess
	AlertWarning
	AlertSidebar
)

// pct is Tailwind's token/n: the token at n% of its own alpha.
func pct(c color.NRGBA, n int) color.NRGBA { return alpha(c, uint8((int(c.A)*n+50)/100)) }

// alertColors is a variant's fill, border, title, description and icon.
type alertColors struct{ bg, border, title, desc, icon color.NRGBA }

func (v AlertVariant) colors() alertColors {
	muted := P.MutedForeground
	switch v {
	case AlertError:
		return alertColors{P.ErrorSurface, pct(P.Error, 32), P.ErrorForeground, pct(P.ErrorForeground, 80), P.Error}
	case AlertInfo:
		return alertColors{pct(P.Info, 4), pct(P.Info, 32), P.CardForeground, muted, P.Info}
	case AlertSuccess:
		return alertColors{pct(P.Success, 4), pct(P.Success, 32), P.CardForeground, muted, P.Success}
	case AlertWarning:
		return alertColors{P.WarningSurface, pct(P.Warning, 32), P.WarningForeground, pct(P.WarningForeground, 80), P.Warning}
	case AlertSidebar:
		return alertColors{SP.SidebarControlSurface, SP.SidebarBorder, P.CardForeground, SP.SidebarMutedForeground, muted}
	}
	bg := color.NRGBA{}
	if dark {
		bg = pct(P.Input, 32)
	}
	return alertColors{bg, P.Border, P.CardForeground, muted, muted}
}

// T3Alert is T3's Alert on the surface on: a lucide icon, a title and a
// description, any of them left out with "". The box is rounded 12 and
// padded 12 14 around 14/20 text (the sidebar variant: 8, 6 8 and 11/16);
// the icon sits 8 before the text, centred on it, and the title 2 over the
// description, which wraps.
func (th *Theme) T3Alert(v AlertVariant, icon, title, desc string, on color.NRGBA) W {
	return func(gtx C) D {
		if on.A == 0 {
			on = P.Background
		}
		cl := v.colors()
		px := max(1, dp(gtx, 1))
		padX, padY, rad := dp(gtx, 14), dp(gtx, 12), dp(gtx, 12)
		size, line := unit.Sp(14), float32(20.0/14)
		if v == AlertSidebar {
			padX, padY, rad = dp(gtx, 8), dp(gtx, 6), dp(gtx, 8)
			size, line = 11, 16.0/11
		}
		bg := over(cl.bg, on)
		w := gtx.Constraints.Max.X
		lead := 0
		if icon != "" {
			lead = dp(gtx, 16+8)
		}
		cw := max(0, w-2*px-2*padX-lead)
		m := op.Record(gtx.Ops)
		cg := gtx
		cg.Constraints = layout.Constraints{Min: image.Pt(cw, 0), Max: image.Pt(cw, gtx.Constraints.Max.Y)}
		cd := Col(cg,
			rigid(func(gtx C) D {
				if title == "" {
					return D{}
				}
				return th.cssText(gtx, Txt{Size: size, LineH: line, Font: UI(font.Medium), Color: over(cl.title, bg)}, title)
			}),
			rigid(func(gtx C) D {
				if title == "" || desc == "" {
					return D{}
				}
				return D{Size: image.Pt(0, dp(gtx, 2))}
			}),
			rigid(func(gtx C) D {
				if desc == "" {
					return D{}
				}
				return th.cssText(gtx, Txt{Size: size, LineH: line, Color: over(cl.desc, bg)}, desc)
			}),
		)
		call := m.Stop()
		r := image.Rect(0, 0, w, cd.Size.Y+2*padY+2*px)
		fill(gtx, bg, r, rad)
		strokeRect(gtx, over(cl.border, bg), r, rad, float32(px))
		if icon != "" {
			off := op.Offset(image.Pt(px+padX, px+padY+(cd.Size.Y-dp(gtx, 16))/2)).Push(gtx.Ops)
			LucideW(icon, cl.icon, 16)(gtx)
			off.Pop()
		}
		defer op.Offset(image.Pt(px+padX+lead, px+padY)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return D{Size: r.Max}
	}
}
