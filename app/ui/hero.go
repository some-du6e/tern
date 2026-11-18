package ui

import (
	"image"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// A new thread's hero (DraftHeroHeadline.tsx, docs/ui/07-composer.md
// §2.16): the composer and its context strip centred in the canvas, the
// question 16dp over them and any notices 16dp under.

// heroSize is the heading's type: text-3xl, 30/36, tracked tight
// (−0.025em).
const heroSize, heroLine, heroTrack = 30, 36, -0.025

// hero lays out the new thread's canvas, the lane wide.
func (th *Theme) hero(gtx C, s *Scene, compact bool) D {
	size := gtx.Constraints.Max
	w := size.X
	m := op.Record(gtx.Ops)
	sg := gtx
	sg.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, size.Y)}
	sd := th.composerStack(sg, s, true)
	stack := m.Stop()
	// The 20dp spacer goes under the stack; T3 centres the two, or below
	// 34rem of window drops them to the bottom.
	h := sd.Size.Y + dp(gtx, 20)
	top := max(0, (size.Y-h)/2)
	if size.Y+dp(gtx, topBarH) < dp(gtx, 544) {
		top = max(0, size.Y-h)
	}

	hm := op.Record(gtx.Ops)
	hg := gtx
	hg.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, size.Y)}
	hd := th.heroHeadline(hg, s)
	head := hm.Stop()
	off := op.Offset(image.Pt(0, top-dp(gtx, 16)-hd.Size.Y)).Push(gtx.Ops)
	head.Add(gtx.Ops)
	off.Pop()

	off = op.Offset(image.Pt(0, top)).Push(gtx.Ops)
	stack.Add(gtx.Ops)
	off.Pop()

	off = op.Offset(image.Pt(0, top+sd.Size.Y)).Push(gtx.Ops)
	th.heroNotices(sg, s)
	off.Pop()
	return D{Size: size}
}

// heroHeadline is the h1, centred: "What should we build in <project>?"
// on one line where it fits, else the question over the project; "What
// should we work on?" without one; and under it, a 24dp line offering a
// thread without a project, or for one, the project picker.
func (th *Theme) heroHeadline(gtx C, s *Scene) D {
	w := gtx.Constraints.Max.X
	t := Txt{Size: heroSize, LineH: heroLine / heroSize, Color: P.Foreground}
	name := t
	name.Font = UI(font.Medium)
	var kids []layout.FlexChild
	centre := func(row W) layout.FlexChild {
		return rigid(func(gtx C) D {
			gtx.Constraints.Min.X = w
			return layout.N.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = 0
				return row(gtx)
			})
		})
	}
	switch {
	case s.Scratch:
		kids = append(kids, centre(func(gtx C) D { return th.tracked(gtx, t, "What should we work on?") }))
	case s.Hero == "":
		kids = append(kids, centre(func(gtx C) D { return th.tracked(gtx, t, "Add a project to start") }))
	default:
		lead := th.trackedWidth(gtx, t, "What should we build in")
		rest := th.trackedWidth(gtx, name, s.Hero) + th.trackedWidth(gtx, t, "?")
		if lead+dp(gtx, 8)+rest <= w {
			kids = append(kids, centre(func(gtx C) D {
				return layout.Flex{Alignment: layout.Baseline}.Layout(gtx,
					rigid(func(gtx C) D { return th.tracked(gtx, t, "What should we build in") }),
					rigid(Spacer(8, 0)),
					rigid(func(gtx C) D { return th.heroProject(gtx, s, name) }),
					rigid(func(gtx C) D { return th.tracked(gtx, t, "?") }),
				)
			}))
		} else {
			// Stacked: the question, then the project in 16/24, 8 apart.
			small := Txt{Size: 16, LineH: 24.0 / 16, Color: P.Foreground, Font: UI(font.Medium)}
			kids = append(kids,
				centre(func(gtx C) D { return th.tracked(gtx, t, "What should we build?") }),
				rigid(Spacer(0, 8)),
				centre(func(gtx C) D { return th.heroProject(gtx, s, small) }),
			)
		}
	}
	if s.Scratch || s.CanScratch {
		kids = append(kids, rigid(Spacer(0, 8)), centre(func(gtx C) D {
			return FixedH(gtx, 24, func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					sub := Txt{Size: 14, LineH: 20.0 / 14}
					if s.Scratch {
						sub.Font, sub.Color = UI(font.Medium), P.Foreground
						return th.heroProjectText(gtx, s, sub, "No project")
					}
					return th.inlineButton(gtx, liveID(s, "project.scratch"), "or start without a project", sub, false)
				})
			})
		}))
	}
	return Col(gtx, kids...)
}

// heroProject is the heading's project picker: T3's picker-tone
// InlineButton, underlined 4dp under while its menu is open, with the
// project menu hanging under it.
func (th *Theme) heroProject(gtx C, s *Scene, t Txt) D {
	return th.heroProjectText(gtx, s, t, s.Hero)
}

func (th *Theme) heroProjectText(gtx C, s *Scene, t Txt, label string) D {
	id := liveID(s, "project.hero")
	if id != "" {
		th.events(gtx, id)
	}
	tr := float32(0)
	if t.Size == heroSize {
		tr = heroTrack
	}
	d := th.trackedTrack(gtx, t, label, tr)
	open := s.Projects != nil && s.ProjectsHero
	if open {
		underline(gtx, t.Color, d, 4, t.Size)
	}
	if id != "" {
		th.hit(gtx, id, d.Size)
		th.tipAt(gtx, id, label, d.Size, true, 0)
	}
	if open {
		// The picker opens centred under the name, 5dp down.
		m := op.Record(gtx.Ops)
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 640), dp(gtx, 640))}
		md := th.traitsMenu(g, s, "project", s.Projects)
		menu := m.Stop()
		m = op.Record(gtx.Ops)
		op.Offset(image.Pt((d.Size.X-md.Size.X)/2, d.Size.Y+dp(gtx, 5))).Add(gtx.Ops)
		menu.Add(gtx.Ops)
		op.Defer(gtx.Ops, m.Stop())
	}
	return d
}

// tracked is one line of t with the heading's tight tracking when it's in
// the heading's size.
func (th *Theme) tracked(gtx C, t Txt, s string) D {
	tr := float32(0)
	if t.Size == heroSize {
		tr = heroTrack
	}
	return th.trackedTrack(gtx, t, s, tr)
}

// trackedWidth is how wide tracked lays s out.
func (th *Theme) trackedWidth(gtx C, t Txt, s string) int {
	m := op.Record(gtx.Ops)
	d := th.tracked(gtx, t, s)
	m.Stop()
	return d.Size.X
}

// trackedTrack is one line of t with CSS letter-spacing: track ems after
// every glyph. Gio's labels have no letter spacing, so this shapes the
// line itself and moves each glyph along.
func (th *Theme) trackedTrack(gtx C, t Txt, s string, track float32) D {
	if t.Font.Typeface == "" {
		t.Font = UI(font.Normal)
	}
	em := gtx.Sp(t.Size)
	th.Shaper.LayoutString(text.Parameters{Font: t.Font, PxPerEm: fixed.I(em), MaxLines: 1, MaxWidth: 1 << 24, Locale: gtx.Locale}, s)
	var glyphs []text.Glyph
	var asc, desc fixed.Int26_6
	step := fixed.Int26_6(track * float32(em) * 64)
	for g, ok := th.Shaper.NextGlyph(); ok; g, ok = th.Shaper.NextGlyph() {
		g.X += step * fixed.Int26_6(len(glyphs))
		asc, desc = max(asc, g.Ascent), max(desc, g.Descent)
		glyphs = append(glyphs, g)
	}
	lh := int(float32(gtx.Sp(t.Size))*t.LineH + 0.5)
	if t.LineH == 0 {
		lh = asc.Ceil() + desc.Ceil()
	}
	if len(glyphs) == 0 {
		return D{Size: image.Pt(0, lh)}
	}
	last := glyphs[len(glyphs)-1]
	width := (last.X + last.Advance + step).Ceil()
	// CSS splits the leading above and below the glyphs.
	top := (lh - asc.Ceil() - desc.Ceil()) / 2
	col := t.Color
	if col.A == 0 {
		col = ColText
	}
	origin := f32.Pt(float32(glyphs[0].X)/64, float32(top+asc.Ceil()))
	tr := op.Affine(f32.AffineId().Offset(origin)).Push(gtx.Ops)
	cl := clip.Outline{Path: th.Shaper.Shape(glyphs)}.Op().Push(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	cl.Pop()
	tr.Pop()
	return D{Size: image.Pt(width, lh), Baseline: lh - top - asc.Ceil()}
}
