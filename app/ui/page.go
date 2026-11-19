package ui

import (
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// The window's gates (docs/ui/11-empty-auth.md): T3's boot splash and boot
// failure page, its StandalonePage with the root error view and the
// pairing form on it, and T3 Connect's sign-in (signin.go). All lengths
// are dp.

// Gate draws s.Gate, the page that fills the window instead of the app,
// with the toasts over it: T3 mounts its toasts above every branch.
func (th *Theme) Gate(gtx C, s *Scene) D {
	size := gtx.Constraints.Max
	switch s.Gate {
	case GateBooting:
		th.bootSplash(gtx)
	case GateBootFailed:
		th.bootFailed(gtx, s.Failure)
	case GateSessionError:
		th.sessionError(gtx, s.Failure)
	case GateSignIn:
		th.signIn(gtx, s.Login)
	case GatePairing:
		th.pairing(gtx, s.Pair)
	}
	th.toasts(gtx, s)
	return D{Size: size}
}

// bootBackground and bootForeground are the boot style's body colours
// (index.html:398-408), which the splash and the boot failure page keep:
// white and #262626, black and #f5f5f5 in dark, not the app's tokens.
func bootBackground() color.NRGBA {
	if dark {
		return rgb(0)
	}
	return rgb(0xffffff)
}

func bootForeground() color.NRGBA {
	if dark {
		return rgb(0xf5f5f5)
	}
	return rgb(0x262626)
}

// bootSplash is index.html's #boot-shell while the saved session loads:
// the body's colour and the app's 64dp icon in the middle of the window.
func (th *Theme) bootSplash(gtx C) {
	size := gtx.Constraints.Max
	fill(gtx, bootBackground(), image.Rectangle{Max: size}, 0)
	th.dragArea(gtx, image.Pt(size.X, dp(gtx, topBarH)))
	s := dp(gtx, 64)
	defer op.Offset(image.Pt((size.X-s)/2, (size.Y-s)/2)).Push(gtx.Ops).Pop()
	appIcon(gtx, 64)
}

// appIcon is tern's icon, the one its window and executable carry
// (tools/winicon): the wing on a dark tile with a lighter edge.
func appIcon(gtx C, sizeDp float32) D {
	s := float32(dp(gtx, sizeDp))
	tile := func(col color.NRGBA, inset, radius float32) {
		r := image.Rect(int(inset), int(inset), int(s-inset+0.5), int(s-inset+0.5))
		fill(gtx, col, r, int(radius))
	}
	tile(rgb(0x2a303a), 0, s*0.22)
	tile(rgb(0x161a20), max(1, s/64), s*0.21)
	u := s * 0.72 / 16
	off := (s - 16*u) / 2
	pt := func(x, y float32) f32.Point { return f32.Pt(off+x*u, off+y*u+u*0.4) }
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(pt(1, 10))
	p.QuadTo(pt(6, 3), pt(15, 2))
	p.QuadTo(pt(10, 5), pt(8.5, 9))
	p.QuadTo(pt(11, 9.5), pt(13, 14))
	p.QuadTo(pt(8, 10.5), pt(1, 10))
	p.Close()
	paint.FillShape(gtx.Ops, rgb(0x4c8dff), clip.Outline{Path: p.End()}.Op())
	return D{Size: image.Pt(int(s), int(s))}
}

// bootFailed is T3's boot failure page (bootError.ts:257-395, index.html's
// #boot-error), for a tern that couldn't start: a 400dp column 22% down
// the window in the boot style's colours and 13/20 type, the 15/22
// heading and its note at 70%, the solid Reload ("boot.reload"), then
// "More options if it keeps failing" ("boot.more"), which opens on the
// report with Copy error details and the bug report link ("boot.issue").
func (th *Theme) bootFailed(gtx C, f *Failure) {
	size := gtx.Constraints.Max
	bg, fg := bootBackground(), bootForeground()
	fill(gtx, bg, image.Rectangle{Max: size}, 0)
	th.dragArea(gtx, image.Pt(size.X, dp(gtx, topBarH)))
	line := over(pct(fg, 12), bg)
	dim := over(pct(fg, 70), bg)
	w := min(dp(gtx, 400), size.X-dp(gtx, 48))
	top := max(dp(gtx, 24), min(size.Y*22/100, dp(gtx, 220)))
	defer op.Offset(image.Pt((size.X-w)/2, top)).Push(gtx.Ops).Pop()
	gtx.Constraints = layout.Constraints{Max: image.Pt(w, size.Y)}
	body := func(t Txt, s string) W {
		if t.Size == 0 {
			t.Size, t.LineH = 13, 20.0/13
		}
		return func(gtx C) D { return th.cssText(gtx, t, s) }
	}
	button := func(id, label string, h, padX float32, solid bool) W {
		return func(gtx C) D {
			c, _ := th.events(gtx, id)
			m := op.Record(gtx.Ops)
			col := fg
			if solid {
				col = bg
			}
			td := th.Text(gtx, Txt{Size: 13, LineH: 20.0 / 13, Font: UI(font.Medium), Color: col, MaxLines: 1}, label)
			call := m.Stop()
			px := max(1, dp(gtx, 1))
			r := image.Rect(0, 0, td.Size.X+2*dp(gtx, padX)+2*px, dp(gtx, h))
			if solid {
				if c.Hovered() {
					defer paint.PushOpacity(gtx.Ops, 0.9).Pop()
				}
				fill(gtx, fg, r, dp(gtx, 6))
			} else {
				strokeRect(gtx, line, r, dp(gtx, 6), float32(px))
			}
			off := op.Offset(image.Pt((r.Dx()-td.Size.X)/2, (r.Dy()-td.Size.Y)/2)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			off.Pop()
			th.hit(gtx, id, r.Max)
			return D{Size: r.Max}
		}
	}
	kids := []layout.FlexChild{
		rigid(body(Txt{Size: 15, LineH: 22.0 / 15, Font: UI(font.SemiBold), Color: fg}, "tern failed to start")),
		rigid(Spacer(0, 4)),
		rigid(body(Txt{Color: dim}, "We are not sure why. Try a reload first.")),
		rigid(Spacer(0, 16)),
		rigid(button("boot.reload", "Reload", 32, 14, true)),
		rigid(Spacer(0, 32)),
		rigid(func(gtx C) D {
			// The summary: its marker, a 5dp corner of lines turned to
			// point right, or down while open. They're declared 1.5px,
			// which Chrome draws 1px.
			c, _ := th.events(gtx, "boot.more")
			col := dim
			if f.Open || c.Hovered() {
				col = fg
			}
			m := op.Record(gtx.Ops)
			td := body(Txt{Color: col}, "More options if it keeps failing")(gtx)
			call := m.Stop()
			mark := dp(gtx, 5)
			func() {
				defer op.Offset(image.Pt(0, (td.Size.Y-mark)/2)).Push(gtx.Ops).Pop()
				turn := float32(-45)
				if f.Open {
					turn = 45
				}
				c := f32.Pt(float32(mark)/2, float32(mark)/2)
				defer op.Affine(f32.Affine2D{}.Rotate(c, turn*3.14159265/180)).Push(gtx.Ops).Pop()
				lw := float32(max(1, dp(gtx, 1)))
				polyline(gtx, col, lw, f32.Pt(float32(mark), 0), f32.Pt(float32(mark), float32(mark)), f32.Pt(0, float32(mark)))
			}()
			x := mark + dp(gtx, 8)
			off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			off.Pop()
			size := image.Pt(x+td.Size.X, td.Size.Y)
			th.hit(gtx, "boot.more", size)
			return D{Size: size}
		}),
	}
	if f.Open {
		kids = append(kids,
			rigid(Spacer(0, 20)),
			rigid(func(gtx C) D {
				// The report: 12/16 mono at 80% in a 6dp-rounded frame, at
				// most 200 tall.
				px := max(1, dp(gtx, 1))
				m := op.Record(gtx.Ops)
				tg := gtx
				tg.Constraints = layout.Constraints{Min: image.Pt(w-2*px-2*dp(gtx, 12), 0), Max: image.Pt(w-2*px-2*dp(gtx, 12), 1<<20)}
				td := th.cssText(tg, Txt{Size: 12, LineH: 16.0 / 12, Font: Mono(), Color: over(pct(fg, 80), bg)}, f.Report)
				call := m.Stop()
				h := min(td.Size.Y+2*dp(gtx, 10)+2*px, dp(gtx, 200))
				r := image.Rect(0, 0, w, h)
				// Its 80% takes the 12% line to 9.6%.
				strokeRect(gtx, over(alpha(fg, 24), bg), r, dp(gtx, 6), float32(px))
				defer clip.Rect(r.Inset(px)).Push(gtx.Ops).Pop()
				defer op.Offset(image.Pt(px+dp(gtx, 12), px+dp(gtx, 10))).Push(gtx.Ops).Pop()
				call.Add(gtx.Ops)
				return D{Size: r.Max}
			}),
			rigid(Spacer(0, 8)),
			rigid(body(Txt{Size: 12, LineH: 18.0 / 12, Color: dim}, "Details may include local file paths and names. Check before sharing.")),
			rigid(Spacer(0, 8)),
			rigid(func(gtx C) D {
				th.offerCopy("copy:boot.report", f.Report)
				label := "Copy error details"
				if _, ok := th.in.copied["copy:boot.report"]; ok {
					label = "Copied"
				}
				return Row(gtx,
					rigid(button("copy:boot.report", label, 28, 10, false)),
					rigid(Spacer(16, 0)),
					rigid(th.textLink("boot.issue", "File a bug report on GitHub", Txt{Size: 13, LineH: 20.0 / 13, Color: fg}, color.NRGBA{}, bg, 3, 40)),
				)
			}),
		)
	}
	Col(gtx, kids...)
}

// textLink is an underlined link on the surface on, text t, its line
// offset below the baseline in decoration% of the text's colour. Hovered,
// the line is the text's colour, and the text turns hover unless that's
// zero.
func (th *Theme) textLink(id, label string, t Txt, hover, on color.NRGBA, offset float32, decoration int) W {
	return func(gtx C) D {
		// As wide as its text, whatever the row it's in would make it.
		gtx.Constraints.Min = image.Point{}
		c, _ := th.events(gtx, id)
		col := over(pct(t.Color, decoration), on)
		if c.Hovered() {
			if hover.A != 0 {
				t.Color = hover
			}
			col = t.Color
		}
		td := th.cssText(gtx, t, label)
		y := td.Size.Y - td.Baseline + dp(gtx, offset)
		fill(gtx, col, image.Rect(0, y, td.Size.X, y+max(1, dp(gtx, 1))), 0)
		th.hit(gtx, id, td.Size)
		return td
	}
}

// offerCopy makes click target id copy text, as copy buttons do.
func (th *Theme) offerCopy(id, text string) {
	if th.in.copies == nil {
		th.in.copies = map[string]string{}
	}
	if th.in.copied == nil {
		th.in.copied = map[string]time.Time{}
	}
	th.in.copies[id] = text
}

// copiedWithin reports whether id's text was copied in the last d.
func (th *Theme) copiedWithin(gtx C, id string, d time.Duration) bool {
	at, ok := th.in.copied[id]
	if !ok {
		return false
	}
	if left := d - gtx.Now.Sub(at); left > 0 {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
		return true
	}
	delete(th.in.copied, id)
	return false
}

func clamp01(t float32) float32 { return max(0, min(1, t)) }

// since is how far through d now is from t, from 0 to 1.
func since(now, t time.Time, d time.Duration) float32 {
	return clamp01(float32(now.Sub(t)) / float32(d))
}

// ---------- the standalone page ----------

// standalonePage is T3's StandalonePage (standalone-page.tsx): the page in
// --background, padded 40 by 24, and the 576dp card centred on it, or at
// its top when it's taller than the window. The card is dialog-glass,
// rounded 16 and padded 24, so body gets 526 of its width.
func (th *Theme) standalonePage(gtx C, body W) D {
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	th.dragArea(gtx, image.Pt(size.X, dp(gtx, topBarH)))
	px, padding := max(1, dp(gtx, 1)), dp(gtx, 24)
	w := min(dp(gtx, 576), size.X-2*dp(gtx, 24))
	inner := w - 2*px - 2*padding
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(inner, 0), Max: image.Pt(inner, 1<<20)}
	cd := body(g)
	call := m.Stop()
	h := cd.Size.Y + 2*padding + 2*px
	at := image.Pt((size.X-w)/2, max(dp(gtx, 40), (size.Y-h)/2))
	defer op.Offset(at).Push(gtx.Ops).Pop()
	dialogGlass(gtx, image.Rect(0, 0, w, h))
	defer op.Offset(image.Pt(px+padding, px+padding)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: size}
}

// dialogGlass paints T3's dialog-glass over r and returns its fill. With
// no backdrop blur in Gio that's its tint over the page, on T3's flat
// page the page's own colour (01-tokens.md §3.12); its 1dp edge is
// foreground 10% (white 8% in dark), its shadow 0 24 64 -24 black 65%
// (0 24 72 -20 black 90% in dark, with a 1dp white 4% light inside its
// top edge).
func dialogGlass(gtx C, r image.Rectangle) color.NRGBA {
	rad, px := dp(gtx, 16), max(1, dp(gtx, 1))
	bg := over(pct(P.Background, 80), P.Background)
	edge := pct(P.Foreground, 10)
	sh := []boxShadow{{0, 24, 64, -24, alpha(rgb(0), 0xa6)}}
	if dark {
		edge = alpha(rgb(0xffffff), 0x14)
		sh = []boxShadow{{0, 24, 72, -20, alpha(rgb(0), 0xe6)}}
	}
	shadow(gtx, r, rad, sh)
	fill(gtx, bg, r, rad)
	strokeRect(gtx, over(edge, bg), r, rad, float32(px))
	if dark {
		hline(gtx, over(alpha(rgb(0xffffff), 0x0a), bg), r.Min.Y+px, r.Min.X+rad, r.Max.X-rad)
	}
	return bg
}

// wordmarkPath is T3Wordmark's "T3" (T3Wordmark.tsx:5-8), in its viewBox
// 15.5309 37 94.3941 56.96.
const wordmarkPath = `M33.4509 93V47.56H15.5309V37H64.3309V47.56H46.4109V93H33.4509ZM86.7253 93.96C82.832 93.96 78.9653 93.4533 75.1253 92.44C71.2853 91.3733 68.032 89.88 65.3653 87.96L70.4053 78.04C72.5386 79.5867 75.0186 80.8133 77.8453 81.72C80.672 82.6267 83.5253 83.08 86.4053 83.08C89.6586 83.08 92.2186 82.44 94.0853 81.16C95.952 79.88 96.8853 78.12 96.8853 75.88C96.8853 73.7467 96.0586 72.0667 94.4053 70.84C92.752 69.6133 90.0853 69 86.4053 69H80.4853V60.44L96.0853 42.76L97.5253 47.4H68.1653V37H107.365V45.4L91.8453 63.08L85.2853 59.32H89.0453C95.9253 59.32 101.125 60.8667 104.645 63.96C108.165 67.0533 109.925 71.0267 109.925 75.88C109.925 79.0267 109.099 81.9867 107.445 84.76C105.792 87.48 103.259 89.6933 99.8453 91.4C96.432 93.1067 92.0586 93.96 86.7253 93.96Z`

var wordmarkSegs = parseSVGPath(wordmarkPath)

// wordmark is the "T3" glyph hdp tall, filled in col: 19.88 wide at 12.
func wordmark(col color.NRGBA, hdp float32) W {
	return func(gtx C) D {
		h := float32(dp(gtx, hdp))
		k := h / 56.96
		func() {
			defer op.Affine(f32.Affine2D{}.Offset(f32.Pt(-15.5309*k, -37*k))).Push(gtx.Ops).Pop()
			fillSegs(gtx, wordmarkSegs, k, col)
		}()
		return D{Size: image.Pt(int(94.3941*k+0.5), int(h+0.5))}
	}
}

// pageHeader is StandalonePageHeader: the wordmark row, 28 tall (the 12dp
// glyph in --foreground, 6 before "Code" in 18/28 medium muted, tracked
// -0.45, on its baseline); 24 below it the title, 20/25 semibold; 8 below
// that the description, 14/20 muted.
func (th *Theme) pageHeader(gtx C, title, desc string) D {
	return Col(gtx,
		rigid(func(gtx C) D {
			lg := gtx
			lg.Constraints.Min = image.Point{}
			m := op.Record(gtx.Ops)
			code := th.tracked(lg, Txt{Size: 18, LineH: 28.0 / 18, Font: UI(font.Medium), Color: P.MutedForeground}, "Code", -0.45)
			call := m.Stop()
			base := code.Size.Y - code.Baseline
			glyph := dp(gtx, 12)
			func() {
				defer op.Offset(image.Pt(0, base-glyph)).Push(gtx.Ops).Pop()
				d := wordmark(P.Foreground, 12)(gtx)
				defer op.Offset(image.Pt(d.Size.X+dp(gtx, 6), glyph-base)).Push(gtx.Ops).Pop()
				call.Add(gtx.Ops)
			}()
			return D{Size: image.Pt(gtx.Constraints.Max.X, code.Size.Y)}
		}),
		rigid(Spacer(0, 24)),
		rigid(func(gtx C) D {
			return th.cssText(gtx, Txt{Size: 20, LineH: 25.0 / 20, Font: UI(font.SemiBold), Color: P.Foreground}, title)
		}),
		rigid(Spacer(0, 8)),
		rigid(func(gtx C) D { return th.cssText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, desc) }),
	)
}

// lineBox is how CSS sets t's lines and Gio doesn't: each LineH x Size
// tall with the glyphs centred in it. Gio makes a single line only its
// ascent and descent tall, with the ascent at the top; lead is how far
// down the text goes to centre it, and height turns Gio's laid out height
// into lines x the exact line height.
type lineBox struct {
	lead, one, adv int
	exact          float32
}

func (th *Theme) lineBox(gtx C, t Txt) lineBox {
	exact := float32(t.Size) * gtx.Metric.PxPerSp * t.LineH
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	one := th.Text(g, t, "x").Size.Y
	m.Stop()
	return lineBox{lead: int((exact-float32(one))/2 + 0.5), one: one, adv: int(exact + 0.5), exact: exact}
}

// height is the CSS height of text Gio laid out gioH tall.
func (b lineBox) height(gioH int) int {
	lines := 1
	if gioH > b.one {
		lines += (gioH - b.one + b.adv/2) / b.adv
	}
	return int(float32(lines)*b.exact + 0.5)
}

// cssText lays out t in CSS line boxes (lineBox).
func (th *Theme) cssText(gtx C, t Txt, s string) D {
	b := th.lineBox(gtx, t)
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints.Min.Y = 0
	d := th.Text(g, t, s)
	call := m.Stop()
	off := op.Offset(image.Pt(0, b.lead)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	h := b.height(d.Size.Y)
	return D{Size: image.Pt(d.Size.X, h), Baseline: h - b.lead - (d.Size.Y - d.Baseline)}
}

// tracked is one line of t with spacing dp after each glyph, as CSS's
// letter-spacing adds it: Gio's labels have no tracking. Its box is the
// CSS line box (cssText's), narrower or wider by the spacing.
func (th *Theme) tracked(gtx C, t Txt, s string, spacing float32) D {
	b := th.lineBox(gtx, t)
	m := op.Record(gtx.Ops)
	d := th.Text(gtx, t, s)
	m.Stop()
	th.Shaper.LayoutString(text.Parameters{
		Font: t.Font, PxPerEm: fixed.I(gtx.Sp(t.Size)), MaxLines: 1, MaxWidth: 1 << 24,
		LineHeightScale: t.LineH, Locale: gtx.Locale,
	}, s)
	step := fixed.Int26_6(spacing * gtx.Metric.PxPerDp * 64)
	var gs []text.Glyph
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		g.X += step * fixed.Int26_6(len(gs))
		gs = append(gs, g)
	}
	if len(gs) > 0 {
		// The glyphs' shape is drawn from the first one's dot, on its
		// baseline, the way a label draws a line.
		at := f32.Pt(float32(gs[0].X)/64, float32(gs[0].Y+int32(b.lead)))
		defer op.Affine(f32.AffineId().Offset(at)).Push(gtx.Ops).Pop()
		paint.FillShape(gtx.Ops, t.Color, clip.Outline{Path: th.Shaper.Shape(gs)}.Op())
	}
	h := b.height(d.Size.Y)
	w := d.Size.X + int(float32(len(gs))*spacing*gtx.Metric.PxPerDp)
	return D{Size: image.Pt(w, h), Baseline: h - b.lead - (d.Size.Y - d.Baseline)}
}

// pageBox is the pages' quiet box, rounded 8 with a --border 70% edge on
// --background 55% (over the card's fill), its content 12 in.
func pageBox(gtx C, pad layout.Inset, w W) D {
	card := over(pct(P.Background, 80), P.Background)
	bg := over(pct(P.Background, 55), card)
	px := max(1, dp(gtx, 1))
	m := op.Record(gtx.Ops)
	cg := gtx
	cg.Constraints.Min.X = gtx.Constraints.Max.X - 2*px
	cg.Constraints.Max.X -= 2 * px
	cd := pad.Layout(cg, w)
	call := m.Stop()
	r := image.Rect(0, 0, gtx.Constraints.Max.X, cd.Size.Y+2*px)
	fill(gtx, bg, r, dp(gtx, 8))
	strokeRect(gtx, over(pct(P.Border, 70), bg), r, dp(gtx, 8), float32(px))
	defer op.Offset(image.Pt(px, px)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: r.Max}
}

// ---------- the root error view ----------

// sessionError is T3's RootRouteErrorView (__root.tsx:234-274), for a first
// request for the account's computers that failed: "Something went
// wrong." and the error, Try again ("session.retry"), Reload app
// ("session.reload") and Copy error, then the report, at most 256 tall,
// scrolling.
func (th *Theme) sessionError(gtx C, f *Failure) {
	th.standalonePage(gtx, func(gtx C) D {
		return Col(gtx,
			rigid(func(gtx C) D { return th.pageHeader(gtx, "Something went wrong.", f.Message) }),
			rigid(Spacer(0, 20)),
			rigid(func(gtx C) D {
				th.offerCopy("copy:session.report", f.Report)
				copied := th.copiedWithin(gtx, "copy:session.report", 2*time.Second)
				cp := Button{Variant: BtnOutline, Size: BtnSizeSm, Icon: "copy", Label: "Copy error"}
				if copied {
					cp.Icon, cp.IconColor, cp.Label = "check", P.Success, "Copied"
				}
				return Row(gtx,
					rigid(th.T3Button("session.retry", Button{Size: BtnSizeSm, Label: "Try again"})),
					rigid(Spacer(8, 0)),
					rigid(th.T3Button("session.reload", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Reload app"})),
					rigid(Spacer(8, 0)),
					rigid(th.T3Button("copy:session.report", cp)),
				)
			}),
			rigid(Spacer(0, 20)),
			rigid(func(gtx C) D {
				return pageBox(gtx, layout.Inset{}, func(gtx C) D {
					card := over(pct(P.Background, 80), P.Background)
					box := over(pct(P.Background, 55), card)
					return Col(gtx,
						rigid(func(gtx C) D {
							return pad(6, 12, 6, 12).Layout(gtx, func(gtx C) D {
								return th.cssText(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.MutedForeground, MaxLines: 1}, "Error report")
							})
						}),
						rigid(func(gtx C) D {
							// The pre, its rule on top, at most 256 tall with it.
							w := gtx.Constraints.Max.X
							bg := over(pct(P.Background, 80), box)
							hline(gtx, over(pct(P.Border, 70), box), 0, 0, w)
							px := max(1, dp(gtx, 1))
							defer op.Offset(image.Pt(0, px)).Push(gtx.Ops).Pop()
							h := min(dp(gtx, 256)-px, th.reportHeight(gtx, f.Report, w))
							r := image.Rect(0, 0, w, h)
							fill(gtx, bg, r, 0)
							defer clip.Rect(r).Push(gtx.Ops).Pop()
							l := th.List("session.report", false)
							g := gtx
							g.Constraints = layout.Exact(r.Max)
							l.Layout(g, 1, func(gtx C, _ int) D {
								return pad(8, 12, 8, 12).Layout(gtx, func(gtx C) D {
									return th.cssText(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: Mono(), Color: over(pct(P.Foreground, 85), bg)}, f.Report)
								})
							})
							return D{Size: image.Pt(r.Dx(), h+px)}
						}),
					)
				})
			}),
		)
	})
}

// reportHeight is how tall the report's text and padding are, w wide.
func (th *Theme) reportHeight(gtx C, report string, w int) int {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, 1<<20)}
	d := pad(8, 12, 8, 12).Layout(g, func(gtx C) D {
		return th.cssText(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: Mono()}, report)
	})
	m.Stop()
	return d.Size.Y
}

// ---------- the pairing form ----------

// pairing is T3's /pair form (PairingRouteSurface.tsx:85-136) for the link
// `t3 pair` prints: the field, the error Alert under it, Continue
// ("Pairing..." while it goes) and the way back to T3 Connect's sign-in
// ("pair.connect"), then a note on where the link comes from.
func (th *Theme) pairing(gtx C, p *Pairing) {
	th.standalonePage(gtx, func(gtx C) D {
		kids := []layout.FlexChild{
			rigid(func(gtx C) D {
				return th.pageHeader(gtx, "Pair with this environment", "Enter a pairing link to start a session with this environment.")
			}),
			rigid(Spacer(0, 24)),
			rigid(func(gtx C) D {
				// The label is inline, so it takes the form's 24dp line box.
				return FixedH(gtx, 24, func(gtx C) D {
					return Row(gtx, rigid(func(gtx C) D {
						return th.cssText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.Foreground, MaxLines: 1}, "Pairing link")
					}))
				})
			}),
			rigid(func(gtx C) D {
				return th.inputField(gtx, "pair.link", "Paste a one-time pairing link", p.Submitting, over(pct(P.Background, 80), P.Background))
			}),
		}
		if p.Error != "" {
			kids = append(kids, rigid(Spacer(0, 16)), rigid(th.T3Alert(AlertError, "", "", p.Error, over(pct(P.Background, 80), P.Background))))
		}
		next := "Continue"
		if p.Submitting {
			next = "Pairing..."
		}
		kids = append(kids,
			rigid(Spacer(0, 16)),
			rigid(func(gtx C) D {
				return Row(gtx,
					rigid(th.T3Button("pair.next", Button{Size: BtnSizeSm, Label: next, Disabled: p.Submitting})),
					rigid(Spacer(8, 0)),
					rigid(th.T3Button("pair.connect", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Sign in with T3 Connect", Disabled: p.Submitting})),
				)
			}),
			rigid(Spacer(0, 24)),
			rigid(func(gtx C) D {
				return pageBox(gtx, padXY(12, 12), func(gtx C) D {
					return th.cssText(gtx, Txt{Size: 12, LineH: 19.5 / 12, Color: P.MutedForeground},
						"Run t3 pair on the computer that runs T3 Code and paste the link it prints. Each link pairs once, and the session lasts 30 days.")
				})
			}),
		)
		return Col(gtx, kids...)
	})
}

// inputField is T3's Input at its default size (input.tsx, 02-primitives
// §2.7, §3.8, §4.5), the live field id on the surface on: a 32dp box
// rounded 8 with its --input edge, lip and 0 1 2 black 5% shadow, 14dp
// text 11 in from the edge, the --placeholder placeholder. Focused, the
// edge turns --ring with a 3dp ring at 24% inside it, and the shadow and
// lip go; disabled, it's at 64%.
func (th *Theme) inputField(gtx C, id, placeholder string, disabled bool, on color.NRGBA) D {
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit, ed.ReadOnly = true, true, disabled
	focused := gtx.Focused(ed)
	w, h := gtx.Constraints.Max.X, dp(gtx, 32)
	r := image.Rect(0, 0, w, h)
	rad, px := dp(gtx, 8), max(1, dp(gtx, 1))
	bg := over(P.Background, on)
	if dark {
		bg = over(pct(P.Input, 32), on)
	}
	if disabled {
		defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
	}
	if !focused && !disabled {
		shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x0d)}})
	}
	fill(gtx, bg, r, rad)
	switch {
	case focused:
		strokeRect(gtx, over(P.Ring, bg), r, rad, float32(px))
		strokeRect(gtx, over(pct(P.Ring, 24), bg), r.Inset(px), rad-px, float32(dp(gtx, 3)))
	case disabled:
		strokeRect(gtx, over(P.Input, bg), r, rad, float32(px))
	default:
		strokeRect(gtx, over(P.Input, bg), r, rad, float32(px))
		th.lip(gtx, r, rad, bg)
	}
	if !disabled {
		th.focusArea(gtx, id, r.Max)
	}
	x := px + dp(gtx, 11)
	tw := w - 2*x
	t := Txt{Size: 14, LineH: 20.0 / 14, Color: over(P.Foreground, bg)}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Exact(image.Pt(tw, dp(gtx, 20)))
	th.editor(g, id, t, "")
	call := m.Stop()
	defer op.Offset(image.Pt(x, (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
	if ed.Len() == 0 && placeholder != "" {
		th.Text(g, Txt{Size: 14, LineH: 20.0 / 14, Color: over(P.Placeholder, bg), MaxLines: 1}, placeholder)
	}
	call.Add(gtx.Ops)
	return D{Size: r.Max}
}

// focusArea gives a field's editor the focus on a press anywhere in size,
// its padding too, with the text cursor.
func (th *Theme) focusArea(gtx C, id string, size image.Point) {
	c, _ := th.events(gtx, "focus:"+id)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
	pointer.CursorText.Add(gtx.Ops)
}
