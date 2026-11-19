package ui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The banner that the open thread's computer is reconnecting or offline
// (docs/ui/11-empty-auth.md §2.18, §3.11, and the banner parts in
// 07-composer.md §2.9, §3.9; ComposerBanner.tsx, useSystemComposerBanners.tsx).
// In a thread it's attached above the composer's card, tucked under it; on
// a new thread's hero it's a card of its own under the composer.

const (
	bannerInset = 22 // --chat-composer-drawer-inset
	bannerTuck  = 17 // calc(1rem + 1px) of it goes under the card
)

// withBanner lays out the composer's card with s.Banner, the composer
// area's one call for it: attached, the banner first and the card over
// its tucked bottom; detached, on the hero, 16 under the card.
func (th *Theme) withBanner(gtx C, s *Scene, card W) D {
	fade := th.bannerFade(gtx, s)
	if s.Banner == nil {
		return card(gtx)
	}
	w := gtx.Constraints.Max.X
	banner := func(detached bool) D {
		defer paint.PushOpacity(gtx.Ops, fade).Pop()
		return th.machineBanner(gtx, s.Banner, w, detached)
	}
	if s.Hero != "" && len(s.Items) == 0 {
		cd := card(gtx)
		defer op.Offset(image.Pt(0, cd.Size.Y+dp(gtx, 16))).Push(gtx.Ops).Pop()
		bd := banner(true)
		return D{Size: image.Pt(cd.Size.X, cd.Size.Y+dp(gtx, 16)+bd.Size.Y)}
	}
	bd := banner(false)
	top := bd.Size.Y - dp(gtx, bannerTuck)
	defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
	cd := card(gtx)
	return D{Size: image.Pt(cd.Size.X, top+cd.Size.Y)}
}

// bannerFade is the banner's opacity: it fades in over 150ms when it
// comes, on T3's ease-out (starting:opacity-0); a mockup's is already in.
func (th *Theme) bannerFade(gtx C, s *Scene) float32 {
	if th.in.popAt == nil {
		th.in.popAt = map[string]time.Time{}
	}
	if s.Banner == nil {
		delete(th.in.popAt, "banner")
		return 0
	}
	if !s.Live {
		return 1
	}
	start, ok := th.in.popAt["banner"]
	if !ok {
		start = gtx.Now
		th.in.popAt["banner"] = start
	}
	t := bezier(0, 0, 0.2, 1, since(gtx.Now, start, 150*time.Millisecond))
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	return t
}

// machineBanner is b on a composer w wide: one 24dp row padded 5 4 of the
// 12dp wifi-off in a 24dp column, the 12/24 medium title, after 4 the
// muted 12/16 diagnosis filling what's left (none under 400 wide), and the
// ghost xs action in a column at least 48 wide. Its glass is the warning
// tint inside a warning 28% edge, or the error's at 32% for a refusal.
// Attached it's 44 narrower and rounded only on top, and 17 more of it go
// under the card; detached it's rounded 16 all round.
func (th *Theme) machineBanner(gtx C, b *Banner, w int, detached bool) D {
	x, bw := 0, w
	h := dp(gtx, 5+24+5)
	if !detached {
		x, bw = dp(gtx, bannerInset), w-2*dp(gtx, bannerInset)
		h += dp(gtx, bannerTuck)
	}
	r := image.Rect(x, 0, x+bw, h)
	bg := bannerGlass(gtx, r, b.Error, detached)
	defer op.Offset(image.Pt(x+dp(gtx, 4), dp(gtx, 5))).Push(gtx.Ops).Pop()
	row, gap := dp(gtx, 24), dp(gtx, 4)
	inner := bw - 2*dp(gtx, 4)
	func() {
		defer op.Offset(image.Pt((row-dp(gtx, 12))/2, (row-dp(gtx, 12))/2)).Push(gtx.Ops).Pop()
		LucideW("wifi-off", P.MutedForeground, 12)(gtx)
	}()
	content := inner - row - gap
	if b.Action != "" {
		m := op.Record(gtx.Ops)
		ad := th.bannerAction(gtx, "banner.retry", b.Action, bg, detached)
		call := m.Stop()
		actions := max(ad.Size.X, dp(gtx, 48))
		content -= actions + gap
		func() {
			defer op.Offset(image.Pt(inner-ad.Size.X, 0)).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
		}()
	}
	lg := gtx
	lg.Constraints = layout.Constraints{Max: image.Pt(max(0, content), row)}
	defer op.Offset(image.Pt(row+gap, 0)).Push(gtx.Ops).Pop()
	title := th.Text(lg, Txt{Size: 12, LineH: 24.0 / 12, Font: UI(font.Medium), Color: bodyText(), MaxLines: 1}, b.Title)
	if rest := content - title.Size.X - gap; b.Detail != "" && bw >= dp(gtx, 400) && rest > 0 {
		dg := lg
		dg.Constraints = layout.Constraints{Max: image.Pt(rest, row)}
		m := op.Record(gtx.Ops)
		dd := th.Text(dg, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}, b.Detail)
		call := m.Stop()
		defer op.Offset(image.Pt(title.Size.X+gap, (row-dd.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
	}
	return D{Size: image.Pt(w, h)}
}

// bannerGlass paints a banner's surface over r and returns its fill: the
// attached surface (--card, --surface-raised in dark) at the 80% glass
// opacity over the page, Gio having no blur, under the variant's 8% tint,
// in its edge, with the composer's shadow. Attached, its bottom 16 are
// masked away, and in dark a seam darkens the 10 above the card's top.
func bannerGlass(gtx C, r image.Rectangle, refused, detached bool) color.NRGBA {
	tone, edge := P.Warning, 28
	if refused {
		tone, edge = P.Error, 32
	}
	surface := P.Card
	if dark {
		surface = P.SurfaceRaised
	}
	bg := over(pct(tone, 8), over(alpha(surface, 0xcc), P.Background))
	sh := []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}}
	if dark {
		sh = []boxShadow{{0, 14, 32, -18, alpha(rgb(0), 0xbf)}}
	}
	rad, px := dp(gtx, 16), max(1, dp(gtx, 1))
	if detached {
		shadow(gtx, r, rad, sh)
		fill(gtx, bg, r, rad)
		strokeRect(gtx, over(pct(tone, edge), bg), r, rad, float32(px))
		return bg
	}
	defer clip.Rect(image.Rect(r.Min.X-dp(gtx, 32), r.Min.Y-dp(gtx, 32), r.Max.X+dp(gtx, 32), r.Max.Y-dp(gtx, 16))).Push(gtx.Ops).Pop()
	shadow(gtx, r, rad, sh)
	paint.FillShape(gtx.Ops, over(pct(tone, edge), bg), clip.RRect{Rect: r, NW: rad, NE: rad}.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rect(r.Min.X+px, r.Min.Y+px, r.Max.X-px, r.Max.Y), NW: rad - px, NE: rad - px}.Op(gtx.Ops))
	if dark {
		seam := r.Max.Y - dp(gtx, bannerTuck)
		band := image.Rect(r.Min.X+px, seam-dp(gtx, 10), r.Max.X-px, seam)
		defer clip.Rect(band).Push(gtx.Ops).Pop()
		paint.LinearGradientOp{
			Stop1: f32.Pt(0, float32(band.Max.Y)), Color1: alpha(rgb(0), 0x2e),
			Stop2: f32.Pt(0, float32(band.Min.Y)), Color2: alpha(rgb(0), 0),
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
	return bg
}

// bannerAction is a banner's ghost xs button on bg: 24 tall, padded 7,
// 12/16 medium, rounded 8 but for its top-right corner, which nests in the
// banner's (12), and on a detached banner its bottom-right too; --accent
// behind it on hover.
func (th *Theme) bannerAction(gtx C, id, label string, bg color.NRGBA, detached bool) D {
	c, _ := th.events(gtx, id)
	m := op.Record(gtx.Ops)
	td := th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.Foreground, MaxLines: 1}, label)
	call := m.Stop()
	px := max(1, dp(gtx, 1))
	r := image.Rect(0, 0, td.Size.X+2*dp(gtx, 7)+2*px, dp(gtx, 24))
	if c.Hovered() {
		in, out := dp(gtx, 8), dp(gtx, 12)
		se := in
		if detached {
			se = out
		}
		paint.FillShape(gtx.Ops, over(P.Accent, bg), clip.RRect{Rect: r, NW: in, NE: out, SE: se, SW: in}.Op(gtx.Ops))
	}
	off := op.Offset(image.Pt((r.Dx()-td.Size.X)/2, (r.Dy()-td.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	th.hit(gtx, id, r.Max)
	return D{Size: r.Max}
}
