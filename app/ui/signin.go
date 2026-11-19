package ui

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// T3 Connect's sign-in (docs/ui/11-empty-auth.md §2.6, §3.6). T3 has no
// sign-in form of its own: it opens Clerk's SignIn modal, themed by
// clerkAppearance.ts, so this is Clerk's card as it measures in T3. tern
// can't sign in with a social account or a passkey, so their buttons, the
// "or" divider and the sign-up footer are left out; the passkey link's
// place takes the way to pair with a link instead ("signin.pair").

// clerkW is the card's width, its edge included; 40 in from each side
// leaves the content 320.
const clerkW = 400

// clerkEdge is the line Clerk draws its inputs' outline and the footer's
// rule in: its neutral at 11% and 7%.
func clerkEdge(strong bool) color.NRGBA {
	switch {
	case dark && strong:
		return alpha(rgb(0xf6f6f6), 0x1c)
	case dark:
		return alpha(rgb(0xf1f1f1), 0x12)
	case strong:
		return alpha(rgb(0x1b1b1b), 0x1c)
	}
	return alpha(rgb(0x1c1c1c), 0x12)
}

// signIn is Clerk's card for l's step, 64 from the window's top over the
// backdrop: dialog-glass, rounded 16, padded 32 40, the header, the form
// and the footer link 32 apart, then the footer's "Secured by" under a
// hairline. T3 opens it over the app; tern's window has only the app's
// background behind it, which the backdrop (--background at 60%) leaves
// as it is.
func (th *Theme) signIn(gtx C, l *SignIn) {
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	th.dragArea(gtx, image.Pt(size.X, dp(gtx, topBarH)))
	w := min(dp(gtx, clerkW), size.X-dp(gtx, 32))
	padX, padY, px := dp(gtx, 40), dp(gtx, 32), max(1, dp(gtx, 1))
	bg := over(pct(P.Background, 80), P.Background)
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(w-2*padX, 0), Max: image.Pt(w-2*padX, 1<<20)}
	body := th.clerkBody(g, l, bg)
	bodyCall := m.Stop()
	m = op.Record(gtx.Ops)
	foot := th.clerkFooter(gtx, w, bg)
	footCall := m.Stop()
	// The card is the box's width (Clerk pulls it 1dp out over each side
	// edge), its padding under the top edge.
	top := px + padY + body.Size.Y + padY
	r := image.Rect(0, 0, w, top+foot.Size.Y+px)
	defer op.Offset(image.Pt((size.X-w)/2, dp(gtx, 64))).Push(gtx.Ops).Pop()
	dialogGlass(gtx, r)
	func() {
		defer op.Offset(image.Pt(padX, px+padY)).Push(gtx.Ops).Pop()
		bodyCall.Add(gtx.Ops)
	}()
	defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
	footCall.Add(gtx.Ops)
	if l.ResendIn > 0 {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second)})
	}
}

// clerkBody is the card's content for l's step, on the card's fill bg.
func (th *Theme) clerkBody(gtx C, l *SignIn, bg color.NRGBA) D {
	gap := rigid(Spacer(0, 32))
	var kids []layout.FlexChild
	switch l.Step {
	case SignInStart:
		kids = append(kids,
			rigid(th.clerkHeader("Sign in to T3 Code", "Welcome back! Please sign in to continue", "")),
			gap,
			rigid(th.clerkLabel("Email address")),
			rigid(Spacer(0, 8)),
			rigid(func(gtx C) D {
				return th.clerkField(gtx, "signin.email", "Enter your email address", false, l.Sending, bg)
			}),
			rigid(th.clerkError(l.FieldError)),
			gap,
			rigid(th.clerkPrimary(l.Sending, bg)),
			gap,
			rigid(th.clerkLink("signin.pair", "Pair with a link instead")),
		)
	case SignInEmailCode:
		kids = append(kids,
			rigid(th.clerkHeader("Check your email", "to continue to T3 Code", l.Email)),
			gap,
			rigid(func(gtx C) D { return th.otpField(gtx, "signin.code", 6, l.Sending, bg) }),
			rigid(th.clerkError(l.FieldError)),
			rigid(Spacer(0, 16)),
			rigid(th.clerkResend(l.ResendIn)),
			gap,
			rigid(th.clerkPrimary(l.Sending, bg)),
		)
	case SignInPassword:
		kids = append(kids,
			rigid(th.clerkHeader("Enter your password", "Enter the password associated with your account", l.Email)),
			gap,
			rigid(th.clerkLabel("Password")),
			rigid(Spacer(0, 8)),
			rigid(func(gtx C) D { return th.clerkField(gtx, "signin.password", "", true, l.Sending, bg) }),
			rigid(th.clerkError(l.FieldError)),
			gap,
			rigid(th.clerkPrimary(l.Sending, bg)),
		)
	}
	if l.Step != SignInStart && l.Other {
		kids = append(kids, gap, rigid(th.clerkLink("signin.method", "Use another method")))
	}
	return Col(gtx, kids...)
}

// clerkHeader is the card's header: the title in 20/28 semibold, the
// subtitle 4 under it in 14/20 muted and, past the start, the address the
// code went to with Clerk's pen to change it ("signin.change").
func (th *Theme) clerkHeader(title, subtitle, email string) W {
	return func(gtx C) D {
		kids := []layout.FlexChild{
			rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 20, LineH: 28.0 / 20, Font: UI(font.SemiBold), Color: P.ContrastForeground}, title)
			}),
			rigid(Spacer(0, 4)),
			rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, subtitle)
			}),
		}
		if email != "" {
			kids = append(kids, rigid(Spacer(0, 8)), rigid(func(gtx C) D {
				return Row(gtx,
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 13, LineH: 18.0 / 13, Font: UI(font.Medium), Color: P.ContrastForeground, MaxLines: 1}, email)
					}),
					rigid(Spacer(4, 0)),
					rigid(func(gtx C) D {
						c, _ := th.events(gtx, "signin.change")
						col := P.MutedForeground
						if c.Hovered() {
							col = P.ContrastForeground
						}
						s := dp(gtx, 24)
						off := op.Offset(image.Pt((s-dp(gtx, 16))/2, (s-dp(gtx, 16))/2)).Push(gtx.Ops)
						clerkPen(gtx, col, 16)
						off.Pop()
						th.hit(gtx, "signin.change", image.Pt(s, s))
						th.tipAt(gtx, "signin.change", "Edit email address", image.Pt(s, s), false, 0)
						return D{Size: image.Pt(s, s)}
					}),
				)
			}))
		}
		return Col(gtx, kids...)
	}
}

// clerkLabel is a field's label, 13/18 medium.
func (th *Theme) clerkLabel(label string) W {
	return func(gtx C) D {
		return th.Text(gtx, Txt{Size: 13, LineH: 18.0 / 13, Font: UI(font.Medium), Color: P.ContrastForeground, MaxLines: 1}, label)
	}
}

// clerkError is the step's field error, 8 under the field in
// --error-foreground (formFieldErrorText).
func (th *Theme) clerkError(msg string) W {
	return func(gtx C) D {
		if msg == "" {
			return D{}
		}
		return pad(8, 0, 0, 0).Layout(gtx, func(gtx C) D {
			return th.Text(gtx, Txt{Size: 13, LineH: 18.0 / 13, Color: P.ErrorForeground}, msg)
		})
	}
}

// clerkOutline draws Clerk's input outline around r on bg, a 1dp line
// just outside it, or focused, a 4dp ring in --ring (T3's colorRing is
// opaque) with the 1dp line over its inner edge.
func clerkOutline(gtx C, r image.Rectangle, rad int, focused bool, bg color.NRGBA) {
	px := max(1, dp(gtx, 1))
	if focused {
		ring := dp(gtx, 4)
		strokeRect(gtx, over(P.Ring, bg), r.Inset(-ring), rad+ring, float32(ring))
		strokeRect(gtx, over(pct(P.ContrastForeground, 28), P.Ring), r.Inset(-px), rad+px, float32(px))
		return
	}
	strokeRect(gtx, over(clerkEdge(true), bg), r.Inset(-px), rad+px, float32(px))
}

// clerkField is Clerk's form field input, the live editor id: 32 tall,
// rounded 8, its 13/18 text padded 12 on --secondary, the placeholder
// muted.
func (th *Theme) clerkField(gtx C, id, placeholder string, mask, disabled bool, bg color.NRGBA) D {
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit, ed.ReadOnly = true, true, disabled
	if mask {
		ed.Mask = '•'
	}
	focused := gtx.Focused(ed)
	w, h := gtx.Constraints.Max.X, dp(gtx, 32)
	r := image.Rect(0, 0, w, h)
	rad := dp(gtx, 8)
	fg := over(P.Secondary, bg)
	clerkOutline(gtx, r, rad, focused, bg)
	fill(gtx, fg, r, rad)
	if !disabled {
		th.focusArea(gtx, id, r.Max)
	}
	x := dp(gtx, 12)
	g := gtx
	g.Constraints = layout.Exact(image.Pt(w-2*x, dp(gtx, 18)))
	t := Txt{Size: 13, LineH: 18.0 / 13, Color: P.ContrastForeground}
	m := op.Record(gtx.Ops)
	th.editor(g, id, t, "")
	call := m.Stop()
	defer op.Offset(image.Pt(x, (h-dp(gtx, 18))/2)).Push(gtx.Ops).Pop()
	if ed.Len() == 0 && placeholder != "" {
		th.Text(g, Txt{Size: 13, LineH: 18.0 / 13, Color: over(P.MutedForeground, fg), MaxLines: 1}, placeholder)
	}
	call.Add(gtx.Ops)
	return D{Size: r.Max}
}

// otpField is Clerk's code field: n boxes 40 square, 8 apart, centred,
// over the hidden editor id that takes the typing, paste and Enter; the
// box the next digit goes in has the focus ring.
func (th *Theme) otpField(gtx C, id string, n int, disabled bool, bg color.NRGBA) D {
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit, ed.MaxLen, ed.ReadOnly = true, true, n, disabled
	hidden := gtx
	hidden.Constraints = layout.Exact(image.Pt(1, 1))
	th.editor(hidden, id, Txt{Size: 1, Color: alpha(P.Foreground, 0)}, "")
	focused := gtx.Focused(ed)
	code := []rune(ed.Text())
	box, gap := dp(gtx, 40), dp(gtx, 8)
	total := n*box + (n-1)*gap
	w := gtx.Constraints.Max.X
	x0 := (w - total) / 2
	rad := dp(gtx, 8)
	for i := 0; i < n; i++ {
		r := image.Rect(x0+i*(box+gap), 0, x0+i*(box+gap)+box, box)
		active := focused && (i == len(code) || (i == n-1 && len(code) == n))
		clerkOutline(gtx, r, rad, active, bg)
		fg := over(P.Secondary, bg)
		fill(gtx, fg, r, rad)
		if i < len(code) {
			m := op.Record(gtx.Ops)
			d := th.Text(gtx, Txt{Size: 16, LineH: 24.0 / 16, Font: UI(font.Medium), Color: P.ContrastForeground, MaxLines: 1}, string(code[i]))
			call := m.Stop()
			off := op.Offset(image.Pt(r.Min.X+(box-d.Size.X)/2, (box-d.Size.Y)/2)).Push(gtx.Ops)
			call.Add(gtx.Ops)
			off.Pop()
		}
	}
	if !disabled {
		th.focusArea(gtx, id, image.Pt(w, box))
	}
	return D{Size: image.Pt(w, box)}
}

// clerkResend is "Didn't receive a code? Resend", muted with the seconds
// left while it waits, then in colorPrimary ("signin.resend").
func (th *Theme) clerkResend(left int) W {
	return func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		label := "Didn't receive a code? Resend"
		if left > 0 {
			return layout.N.Layout(gtx, func(gtx C) D {
				return th.Text(gtx, Txt{Size: 13, LineH: 18.0 / 13, Color: P.MutedForeground, MaxLines: 1}, label+" ("+strconv.Itoa(left)+")")
			})
		}
		return th.clerkLink("signin.resend", label)(gtx)
	}
}

// clerkLink is a footer action link, centred: 13/18 medium in Clerk's
// colorPrimary, T3's --update-foreground, underlined on hover.
func (th *Theme) clerkLink(id, label string) W {
	return func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.N.Layout(gtx, func(gtx C) D {
			c, _ := th.events(gtx, id)
			td := th.Text(gtx, Txt{Size: 13, LineH: 18.0 / 13, Font: UI(font.Medium), Color: P.UpdateForeground, MaxLines: 1}, label)
			if c.Hovered() {
				y := td.Size.Y - td.Baseline + dp(gtx, 2)
				fill(gtx, P.UpdateForeground, image.Rect(0, y, td.Size.X, y+max(1, dp(gtx, 1))), 0)
			}
			th.hit(gtx, id, td.Size)
			return td
		})
	}
}

// clerkPrimary is Continue ("signin.next") on the card's fill bg: 32
// tall, rounded 8, 13/18 medium in --primary-foreground on a flat
// --primary (T3's bg-none drops Clerk's gloss), its shadow primary 24% and
// no inset highlight; primary 90% on hover; a spinner while it's on its
// way.
func (th *Theme) clerkPrimary(sending bool, bg color.NRGBA) W {
	return func(gtx C) D {
		hovered := false
		if !sending {
			c, _ := th.events(gtx, "signin.next")
			hovered = c.Hovered()
		}
		w, h := gtx.Constraints.Max.X, dp(gtx, 32)
		r := image.Rect(0, 0, w, h)
		rad := dp(gtx, 8)
		shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, pct(P.Primary, 24)}})
		fillCol := P.Primary
		if hovered {
			fillCol = over(pct(P.Primary, 90), bg)
		}
		fill(gtx, fillCol, r, rad)
		var content D
		m := op.Record(gtx.Ops)
		if sending {
			content = T3Spinner(P.PrimaryForeground, 16, float64(gtx.Now.UnixMilli()%60000)/1000)(gtx)
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(33 * time.Millisecond)})
		} else {
			content = th.Text(gtx, Txt{Size: 13, LineH: 18.0 / 13, Font: UI(font.Medium), Color: P.PrimaryForeground, MaxLines: 1}, "Continue")
		}
		call := m.Stop()
		off := op.Offset(image.Pt((w-content.Size.X)/2, (h-content.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		if !sending {
			th.hit(gtx, "signin.next", r.Max)
		}
		return D{Size: r.Max}
	}
}

// clerkFooter is the card's footer, w wide: a hairline, then "Secured by"
// (12/16 medium, muted) and Clerk's mark, centred and padded 16 32.
func (th *Theme) clerkFooter(gtx C, w int, bg color.NRGBA) D {
	px := max(1, dp(gtx, 1))
	hline(gtx, over(clerkEdge(false), bg), 0, px, w-px)
	m := op.Record(gtx.Ops)
	lg := gtx
	lg.Constraints.Min = image.Point{}
	row := Row(lg,
		rigid(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.MutedForeground, MaxLines: 1}, "Secured by")
		}),
		rigid(Spacer(4, 0)),
		rigid(func(gtx C) D { return clerkMark(gtx, over(P.MutedForeground, bg), 14) }),
	)
	call := m.Stop()
	h := px + dp(gtx, 16) + row.Size.Y + dp(gtx, 16)
	defer op.Offset(image.Pt((w-row.Size.X)/2, px+dp(gtx, 16))).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: image.Pt(w, h)}
}

// clerkMarkPaths are Clerk's mark (@clerk/ui icons/logo-mark-new.svg, in a
// 50 by 14 box): the ring's dot, its lower arc, its upper arc at half
// strength, and the word.
var clerkMarkPaths = []struct {
	segs []seg
	half bool
}{
	{parseSVGPath("M5.702 7a2.187 2.188 0 1 0 4.374 0a2.187 2.188 0 1 0 -4.374 0Z"), false},
	{parseSVGPath("M11.83 12.18a.415.415 0 0 1-.05.64A6.97 6.97 0 0 1 7.888 14a6.97 6.97 0 0 1-3.891-1.18.415.415 0 0 1-.051-.64l1.598-1.6a.47.47 0 0 1 .55-.074 3.9 3.9 0 0 0 1.794.431 3.9 3.9 0 0 0 1.792-.43.47.47 0 0 1 .551.074z"), false},
	{parseSVGPath("M11.78 1.18a.415.415 0 0 1 .05.64l-1.598 1.6a.47.47 0 0 1-.55.073 3.937 3.937 0 0 0-5.3 5.3.47.47 0 0 1-.074.55L2.71 10.942a.415.415 0 0 1-.641-.051 7 7 0 0 1 9.71-9.71"), true},
	{parseSVGPath("M23.749 1.422c0-.06.049-.11.109-.11h1.64c.06 0 .11.05.11.11v11.156a.11.11 0 0 1-.11.11h-1.64a.11.11 0 0 1-.11-.11zm-2.316 8.9a.11.11 0 0 0-.15.004 2.9 2.9 0 0 1-.884.569c-.36.148-.747.222-1.137.219-.33.01-.658-.047-.965-.166a2.4 2.4 0 0 1-.817-.527c-.424-.432-.668-1.05-.668-1.785 0-1.473.98-2.48 2.45-2.48.394-.005.785.074 1.144.234.325.144.617.35.86.607.04.044.11.049.155.01l1.108-.959a.107.107 0 0 0 .01-.152c-.832-.93-2.138-1.412-3.379-1.412-2.499 0-4.27 1.686-4.27 4.166 0 1.227.44 2.26 1.182 2.99.743.728 1.801 1.157 3.022 1.157 1.53 0 2.763-.587 3.485-1.34a.107.107 0 0 0-.009-.155zm13.212-1.14a.11.11 0 0 1-.107.096H28.79a.106.106 0 0 0-.104.132c.286 1.06 1.138 1.701 2.302 1.701a2.6 2.6 0 0 0 1.136-.236 2.55 2.55 0 0 0 .862-.645.08.08 0 0 1 .112-.01l1.155 1.006c.044.039.05.105.013.15-.698.823-1.828 1.42-3.38 1.42-2.386 0-4.185-1.651-4.185-4.162 0-1.232.424-2.264 1.13-2.994.373-.375.82-.67 1.314-.87a4 4 0 0 1 1.557-.285c2.419 0 3.983 1.701 3.983 4.05a7 7 0 0 1-.04.647m-5.924-1.524a.104.104 0 0 0 .103.133h3.821c.07 0 .123-.066.103-.134-.26-.901-.921-1.503-1.947-1.503a2.1 2.1 0 0 0-.88.16 2.1 2.1 0 0 0-.733.507 2.24 2.24 0 0 0-.467.837m11.651-3.172c.061-.001.11.048.11.109v1.837a.11.11 0 0 1-.117.109 7 7 0 0 0-.455-.024c-1.43 0-2.27 1.007-2.27 2.329v3.732a.11.11 0 0 1-.11.11h-1.64a.11.11 0 0 1-.11-.11v-7.87c0-.06.049-.109.11-.109h1.64c.06 0 .11.05.11.11v1.104a.011.011 0 0 0 .02.007c.64-.857 1.587-1.333 2.587-1.333zm4.444 4.81a.035.035 0 0 1 .056.006l2.075 3.334a.11.11 0 0 0 .093.052h1.865a.11.11 0 0 0 .093-.168L46.152 7.93a.11.11 0 0 1 .012-.131l2.742-3.026a.11.11 0 0 0-.081-.183h-1.946a.11.11 0 0 0-.08.036l-3.173 3.458a.11.11 0 0 1-.19-.074V1.422a.11.11 0 0 0-.11-.11h-1.64a.11.11 0 0 0-.11.11v11.156c0 .06.05.11.11.11h1.64a.11.11 0 0 0 .11-.11v-1.755a.1.1 0 0 1 .03-.075z"), false},
}

// clerkMark draws Clerk's mark hdp tall in col.
func clerkMark(gtx C, col color.NRGBA, hdp float32) D {
	k := float32(dp(gtx, hdp)) / 14
	for _, p := range clerkMarkPaths {
		c := col
		if p.half {
			c = alpha(col, col.A/2)
		}
		fillSegs(gtx, p.segs, k, c)
	}
	return D{Size: image.Pt(int(50*k+0.5), dp(gtx, hdp))}
}

// clerkPen is Clerk's pen icon (icons/pen.svg), stroked 1.5 in its 16 box.
var clerkPenSegs = parseSVGPath("M10.75 13.25h2.5M9.237 3.443l-5.055 4.89a2 2 0 0 0-.562 1.003l-.87 3.914 3.705-.872c.357-.084.683-.264.944-.52l5.228-5.138a2.376 2.376 0 0 0-.066-3.277 2.34 2.34 0 0 0-3.324 0")

func clerkPen(gtx C, col color.NRGBA, sizeDp float32) {
	k := float32(dp(gtx, sizeDp)) / 16
	var p clip.Path
	p.Begin(gtx.Ops)
	tracePath(&p, clerkPenSegs, k)
	paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.5 * k}.Op())
}

// fillSegs fills segs, scaled by k, in col.
func fillSegs(gtx C, segs []seg, k float32, col color.NRGBA) {
	var p clip.Path
	p.Begin(gtx.Ops)
	tracePath(&p, segs, k)
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}

// tracePath adds segs, scaled by k, to p.
func tracePath(p *clip.Path, segs []seg, k float32) {
	for _, sg := range segs {
		at := func(i int) f32.Point { return sg.p[i].Mul(k) }
		switch sg.op {
		case 'M':
			p.MoveTo(at(0))
		case 'L':
			p.LineTo(at(0))
		case 'Q':
			p.QuadTo(at(0), at(1))
		case 'C':
			p.CubeTo(at(0), at(1), at(2))
		case 'Z':
			p.Close()
		}
	}
}
