package ui

import (
	"image"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
)

// signInAnim is the sign-in screen's motion: the card rising in, steps
// sliding, a shake on errors, and a light drifting over the dot grid.
type signInAnim struct {
	start  time.Time
	step   int
	stepAt time.Time
	dir    float32 // +1 when a step moves forward, -1 back
	err    string
	errAt  time.Time
	last   time.Time // the last frame drawn, to replay the entrance after sign-out

	gridOps   op.Ops // the dim dots, recorded once per window size
	grid      op.CallOp
	gridSize  image.Point
	gridScale float32
}

func clamp01(t float32) float32 { return max(0, min(1, t)) }

func since(now, t time.Time, d time.Duration) float32 {
	return clamp01(float32(now.Sub(t)) / float32(d))
}

// LiveSignIn is the real sign-in card: email, then the emailed code (or a
// password), or a pairing URL from `t3 pair` for a machine that isn't on
// T3 Connect. Targets: "signin.next", "signin.change", "signin.password",
// "signin.pairmode", "signin.connectmode"; fields "signin.email",
// "signin.code", "signin.pair".
func (th *Theme) LiveSignIn(gtx C, s *Scene) D {
	return Fill(gtx, ColWindow, func(gtx C) D {
		size := gtx.Constraints.Max
		l := s.Login
		a := &th.signIn
		now := gtx.Now
		if a.start.IsZero() || now.Sub(a.last) > 5*time.Second {
			a.start, a.step, a.stepAt, a.errAt, a.err = now, l.Step, time.Time{}, time.Time{}, l.Error
		}
		a.last = now
		if l.Step != a.step {
			a.dir = 1
			if l.Step < a.step || (a.step == SignInPair && l.Step == SignInEmail) {
				a.dir = -1
			}
			a.step, a.stepAt = l.Step, now
		}
		if l.Error != a.err {
			if l.Error != "" {
				a.errAt = now
			}
			a.err = l.Error
		}
		th.dragArea(gtx, image.Pt(size.X, dp(gtx, topBarH)))
		enter := easeOut(since(now, a.start, 600*time.Millisecond))
		th.signInGrid(gtx, size, now.Sub(a.start).Seconds(), enter)
		// The light keeps moving, so keep drawing at about 30 fps.
		gtx.Execute(op.InvalidateCmd{At: now.Add(33 * time.Millisecond)})
		label := func(t string) layout.FlexChild {
			return rigid(func(gtx C) D { return pad(0, 0, 6, 0).Layout(gtx, func(gtx C) D { return th.Small(gtx, ColDim, t) }) })
		}
		primary := func(id, text string) layout.FlexChild {
			return rigid(th.Click(id, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				bg := ColAccent
				if l.Sending {
					bg = alpha(ColAccent, 0x80)
					text = "Working…"
					if l.SendingCode {
						text = "Sending code…"
					}
				}
				return Box{Bg: bg, Radius: 5}.Layout(gtx, func(gtx C) D {
					return FixedH(gtx, 32, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Center.Layout(gtx, func(gtx C) D {
							if !l.Sending {
								return th.Label(gtx, rgb(0xffffff), font.SemiBold, text)
							}
							phase := now.Sub(a.start).Seconds() * 5
							return Row(gtx,
								rigid(Spinner(rgb(0xffffff), 12, phase)),
								rigid(Spacer(8, 0)),
								rigid(func(gtx C) D { return th.Label(gtx, rgb(0xffffff), font.SemiBold, text) }),
							)
						})
					})
				})
			}))
		}
		link := func(id, text string) layout.FlexChild {
			return rigid(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Center.Layout(gtx, th.Click(id, func(gtx C) D {
					return padXY(6, 3).Layout(gtx, func(gtx C) D { return th.Label(gtx, ColDim, font.Medium, text) })
				}))
			})
		}
		// The card rises and fades in; an error shakes it.
		dx := float32(0)
		if !a.errAt.IsZero() {
			if t := since(now, a.errAt, 420*time.Millisecond); t < 1 {
				dx = float32(math.Sin(float64(t)*math.Pi*5)) * (1 - t) * float32(dp(gtx, 7))
			}
		}
		dy := (1 - enter) * float32(dp(gtx, 18))
		defer op.Offset(image.Pt(int(dx), int(dy))).Push(gtx.Ops).Pop()
		defer paint.PushOpacity(gtx.Ops, enter).Pop()
		stepT := float32(1)
		if !a.stepAt.IsZero() {
			stepT = easeOut(since(now, a.stepAt, 320*time.Millisecond))
		}
		layout.Center.Layout(gtx, func(gtx C) D {
			return popupFrame(gtx, func(gtx C) D {
				return FixedW(gtx, 392, func(gtx C) D {
					return pad(28, 28, 24, 28).Layout(gtx, func(gtx C) D {
						// A new step slides in from the side it comes from.
						off := op.Offset(image.Pt(int(a.dir*(1-stepT)*float32(dp(gtx, 22))), 0)).Push(gtx.Ops)
						defer off.Pop()
						defer paint.PushOpacity(gtx.Ops, 0.25+0.75*stepT).Pop()
						title, blurb := "Sign in with T3 Connect", "Your environments and threads, from every machine linked to your account."
						if l.Step == SignInPair {
							title, blurb = "Pair with a machine", "Run `t3 pair` on the machine and paste the URL it prints. The session lasts 30 days."
						}
						kids := []layout.FlexChild{
							rigid(func(gtx C) D {
								return Row(gtx,
									rigid(flapping(Logo(22), now.Sub(a.start).Seconds(), l.Sending)),
									rigid(Spacer(10, 0)),
									rigid(func(gtx C) D { return th.Text(gtx, Txt{Size: 20, Font: UI(font.SemiBold)}, "tern") }),
								)
							}),
							rigid(Spacer(0, 18)),
							rigid(func(gtx C) D { return th.Text(gtx, Txt{Size: SizeTitle, Font: UI(font.SemiBold)}, title) }),
							rigid(Spacer(0, 5)),
							rigid(func(gtx C) D { return th.Text(gtx, Txt{Color: ColDim, LineH: 1.4}, blurb) }),
							rigid(Spacer(0, 20)),
						}
						switch l.Step {
						case SignInEmail:
							kids = append(kids,
								label("Email"),
								rigid(th.Field("signin.email", 0, false, "you@example.com", false)),
								rigid(Spacer(0, 20)),
								primary("signin.next", "Continue"),
								rigid(Spacer(0, 10)),
								link("signin.pairmode", "Pair with a URL instead"),
							)
						case SignInCode:
							what, field := "Code", "signin.code"
							if l.Password {
								what = "Password"
							}
							kids = append(kids,
								label("Email"),
								rigid(func(gtx C) D {
									return Row(gtx,
										flexed(th.Input(0, false, l.Email, "", false)),
										rigid(Spacer(6, 0)),
										rigid(th.Click("signin.change", th.Button(Ghost, 0, false, "Change"))),
									)
								}),
								rigid(Spacer(0, 16)),
								rigid(func(gtx C) D {
									return Row(gtx,
										rigid(func(gtx C) D { return th.Small(gtx, ColDim, what) }),
										flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
										rigid(func(gtx C) D {
											if l.Password {
												return D{}
											}
											return th.Small(gtx, ColFaint, "Sent to "+l.Email)
										}),
									)
								}),
								rigid(Spacer(0, 6)),
								rigid(func(gtx C) D {
									if l.Password {
										return th.Field(field, 0, false, "Password", true)(gtx)
									}
									return th.codeField(gtx, field, 6)
								}),
								rigid(Spacer(0, 20)),
								primary("signin.next", "Sign in"),
							)
							if l.HasPassword {
								other := "Use password instead"
								if l.Password {
									other = "Use the emailed code instead"
								}
								kids = append(kids, rigid(Spacer(0, 10)), link("signin.password", other))
							}
						case SignInPair:
							kids = append(kids,
								label("Pairing URL"),
								rigid(th.Field("signin.pair", 0, false, "https://host/pair#token=…", false)),
								rigid(Spacer(0, 20)),
								primary("signin.next", "Pair"),
								rigid(Spacer(0, 10)),
								link("signin.connectmode", "Sign in with T3 Connect instead"),
							)
						}
						if l.Error != "" {
							kids = append(kids, rigid(Spacer(0, 14)), rigid(func(gtx C) D {
								return th.Text(gtx, Txt{Size: SizeSmall, Color: ColDelFg, LineH: 1.4}, l.Error)
							}))
						}
						kids = append(kids,
							rigid(Spacer(0, 18)),
							rigid(func(gtx C) D {
								w := gtx.Constraints.Max.X
								hline(gtx, ColBorder, 0, 0, w)
								return D{Size: image.Pt(w, max(1, dp(gtx, 1)))}
							}),
							rigid(Spacer(0, 14)),
							rigid(func(gtx C) D {
								return Row(gtx,
									rigid(IconW(IconLock, ColFaint, 13)),
									rigid(Spacer(7, 0)),
									flexed(func(gtx C) D {
										return th.Text(gtx, Txt{Size: SizeSmall, Color: ColFaint, LineH: 1.4}, "Sessions are bound to a key that never leaves this device.")
									}),
								)
							}),
						)
						return Col(gtx, kids...)
					})
				})
			})
		})
		return D{Size: size}
	})
}

// codeField is the six code boxes over a hidden editor: typing fills the
// boxes, and the editor keeps paste, backspace and Enter. Clicking the boxes
// focuses it.
func (th *Theme) codeField(gtx C, id string, n int) D {
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit, ed.MaxLen = true, true, n
	small := gtx
	small.Constraints = layout.Exact(image.Pt(1, 1))
	th.editor(small, id, Txt{Size: 1, Color: alpha(ColText, 0)}, "")
	focused := gtx.Focused(ed)
	return th.Click("focus:"+id, func(gtx C) D { return th.codeBoxesFocus(gtx, ed.Text(), n, focused) })(gtx)
}

// signInGrid is the dot grid behind the card. It fades in from the middle,
// and a soft light wanders over it, warming the dots it passes.
func (th *Theme) signInGrid(gtx C, size image.Point, t float64, enter float32) {
	a := &th.signIn
	step := dp(gtx, 22)
	r := max(1, dp(gtx, 0.8))
	if a.gridSize != size || a.gridScale != gtx.Metric.PxPerDp {
		a.gridOps.Reset()
		ggtx := gtx
		ggtx.Ops = &a.gridOps
		m := op.Record(&a.gridOps)
		for y := step / 2; y < size.Y; y += step {
			for x := step / 2; x < size.X; x += step {
				circle(ggtx, alpha(rgb(0xffffff), 0x0b), image.Pt(x, y), r)
			}
		}
		a.grid, a.gridSize, a.gridScale = m.Stop(), size, gtx.Metric.PxPerDp
	}
	op1 := paint.PushOpacity(gtx.Ops, enter)
	a.grid.Add(gtx.Ops)
	op1.Pop()

	// The light: a slow Lissajous loop around the card.
	cx := float64(size.X)/2 + math.Cos(t*0.21)*float64(size.X)*0.32
	cy := float64(size.Y)/2 + math.Sin(t*0.29+1)*float64(size.Y)*0.30
	rad := float64(dp(gtx, 260))
	x0, x1 := int(cx-rad), int(cx+rad)
	y0, y1 := int(cy-rad), int(cy+rad)
	for y := step / 2; y < size.Y; y += step {
		if y < y0 || y > y1 {
			continue
		}
		for x := step / 2; x < size.X; x += step {
			if x < x0 || x > x1 {
				continue
			}
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / rad
			if d >= 1 {
				continue
			}
			k := float32(1 - d*d) // a broad, soft falloff
			col := lerpCol(rgb(0xffffff), ColAccent, k)
			circle(gtx, alpha(col, uint8(float32(0x90)*k*enter)), image.Pt(x, y), r+int(float32(r)*k*0.8))
		}
	}
}

// flapping beats the wing now and then, and steadily while working.
func flapping(w W, t float64, busy bool) W {
	return func(gtx C) D {
		var amt float64
		if busy {
			amt = math.Sin(t*9) * 0.22
		} else if p := math.Mod(t+3.2, 4.5); p < 0.7 {
			// A single beat every few seconds, the first just after the card lands.
			amt = math.Sin(p/0.7*2*math.Pi) * 0.2
		}
		m := op.Record(gtx.Ops)
		dims := w(gtx)
		call := m.Stop()
		c := f32.Pt(float32(dims.Size.X)/2, float32(dims.Size.Y)*0.62)
		tr := op.Affine(f32.Affine2D{}.Scale(c, f32.Pt(1, float32(1-amt)))).Push(gtx.Ops)
		call.Add(gtx.Ops)
		tr.Pop()
		return dims
	}
}
