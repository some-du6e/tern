package ui

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/x/richtext"
)

// The main column with no thread in it (docs/ui/11-empty-auth.md §2.10-2.13,
// §3.9): T3's NoProjectsHero, its hosted home with no computer on the
// account, NoActiveThreadState and DraftStartError, each under its
// WorkspacePageHeader, as T3's desktop app draws them. All lengths are dp.

// home draws s.Home over the chat column: the 52dp top bar, with its
// rule and title for the states that have one, and the Empty block
// centred in what's below it, padded 48. (T3's narrower padding and title
// colour are for windows under 768 wide, which tern's never are.)
func (th *Theme) home(gtx C, s *Scene, collapsed bool) {
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	top := dp(gtx, topBarH)
	th.dragArea(gtx, image.Pt(size.X, top))
	left := float32(20)
	if collapsed {
		left = th.contentLeft()
	}
	var title W
	switch s.Home {
	case HomeNoThread:
		title = func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: over(pct(P.MutedForeground, 50), P.Background), MaxLines: 1}, "No active thread")
		}
	case HomeNoComputer:
		title = func(gtx C) D {
			return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: over(pct(P.MutedForeground, 60), P.Background), MaxLines: 1}, "tern")
		}
	}
	if title != nil {
		hline(gtx, over(P.Border, P.Background), top-max(1, dp(gtx, 1)), 0, size.X)
		g := gtx
		g.Constraints = layout.Constraints{Min: image.Pt(0, top), Max: image.Pt(size.X, top)}
		pad(0, 20, 0, left).Layout(g, func(gtx C) D { return Row(gtx, rigid(title)) })
	}
	var body W
	switch s.Home {
	case HomeNoProjects:
		body = th.noProjects
	case HomeNoComputer:
		body = th.noComputer
	case HomeNoThread:
		body = th.noThread
	case HomeDraftError:
		body = th.draftError
	default:
		return
	}
	padding := dp(gtx, 48)
	area := image.Pt(size.X-2*padding, size.Y-top-2*padding)
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: area}
	bd := body(g)
	call := m.Stop()
	defer op.Offset(image.Pt(padding+(area.X-bd.Size.X)/2, top+padding+(area.Y-bd.Size.Y)/2)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// emptyHeader is EmptyHeader, at most w wide: the title (20/28 semibold,
// 30/36 for the hero) in the inherited text colour, the description 4
// under it in 14/20 muted, both centred and wrapping, then below.
func (th *Theme) emptyHeader(gtx C, w int, hero bool, title, desc string, below W) D {
	size, line := unit.Sp(20), float32(28)
	if hero {
		size, line = 30, 36
	}
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, w)
	centred := func(t Txt, s string) layout.FlexChild {
		t.Align = text.Middle
		return rigid(func(gtx C) D { return th.balanced(gtx, t, s) })
	}
	kids := []layout.FlexChild{
		centred(Txt{Size: size, LineH: line / float32(size), Font: UI(font.SemiBold), Color: bodyText()}, title),
		rigid(Spacer(0, 4)),
		centred(Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, desc),
	}
	if below != nil {
		kids = append(kids, rigid(below))
	}
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx, kids...)
}

// balanced lays t out at the narrowest width that keeps its lines, as
// text-wrap: balance does, and never wider than it needs.
func (th *Theme) balanced(gtx C, t Txt, s string) D {
	m := op.Record(gtx.Ops)
	lg := gtx
	lg.Constraints.Min = image.Point{}
	d := th.Text(lg, t, s)
	m.Stop()
	w := d.Size.X
	if lines := d.Size.Y / max(1, int(float32(t.Size)*t.LineH*gtx.Metric.PxPerSp+0.5)); lines > 1 {
		// Narrow it while it keeps as many lines.
		lo, hi := w/lines, w
		for lo < hi {
			mid := (lo + hi) / 2
			g := lg
			g.Constraints.Max.X = mid
			m := op.Record(gtx.Ops)
			if th.Text(g, t, s).Size.Y > d.Size.Y {
				lo = mid + 1
			} else {
				hi = mid
			}
			m.Stop()
		}
		w = lo
	}
	lg.Constraints = layout.Exact(image.Pt(w, d.Size.Y))
	return th.Text(lg, t, s)
}

// noThread is NoActiveThreadState's block: 512 wide at most, padded 32
// by 48, the header full width.
func (th *Theme) noThread(gtx C) D {
	w := min(gtx.Constraints.Max.X, dp(gtx, 512))
	return pad(48, 32, 48, 32).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = w - 2*dp(gtx, 32)
		return th.emptyHeader(gtx, w-2*dp(gtx, 32), false, "Pick a thread to continue", "Select an existing thread or create a new one to get started.", nil)
	})
}

// noProjects is NoProjectsHero's block: the hero header in the same
// 512 by 48 32 box. T3's Add project and Start without a project are
// tern's to grow: it can neither add a project nor start a thread
// without one, so the hero has T3's words for that case and no buttons.
func (th *Theme) noProjects(gtx C) D {
	w := min(gtx.Constraints.Max.X, dp(gtx, 512))
	return pad(48, 32, 48, 32).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = w - 2*dp(gtx, 32)
		return th.emptyHeader(gtx, w-2*dp(gtx, 32), true, "What should we work on?", "Add a project to start your first thread.", nil)
	})
}

// draftError is DraftStartError's block: a header 448 wide at most and
// Try again ("home.retry"), small with its refresh icon, 20 under it.
func (th *Theme) draftError(gtx C) D {
	return th.emptyHeader(gtx, dp(gtx, 448), false, "Couldn’t start a new thread", "The project is still available. Try opening the draft again.", func(gtx C) D {
		return pad(20, 0, 0, 0).Layout(gtx, th.T3Button("home.retry", Button{Size: BtnSizeSm, Icon: "refresh-cw", Label: "Try again"}))
	})
}

// noComputer is HostedEmptyState with T3 Connect on and the account signed
// in (tern has no other way in): the header, then 24 under it the steps,
// 384 wide. Step 1 reads Signed in; step 2 is the commands that install
// and connect T3 Code, and the wait for the computer to appear.
func (th *Theme) noComputer(gtx C) D {
	w := min(gtx.Constraints.Max.X, dp(gtx, 384))
	gtx.Constraints.Max.X = w
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		rigid(func(gtx C) D {
			return th.emptyHeader(gtx, w, false, "No computer connected", "Your agents run on your computer.", nil)
		}),
		rigid(Spacer(0, 24)),
		rigid(func(gtx C) D {
			gtx.Constraints.Min.X = w
			return Col(gtx,
				rigid(th.setupStep(1, func(gtx C) D {
					return Row(gtx,
						rigid(LucideW("check", P.MutedForeground, 16)),
						rigid(Spacer(6, 0)),
						rigid(func(gtx C) D { return th.Text(gtx, homeText(P.MutedForeground), "Signed in") }),
					)
				})),
				rigid(Spacer(0, 16)),
				rigid(th.setupStep(2, func(gtx C) D {
					return Col(gtx,
						rigid(th.runOnComputer("t3 connect")),
						rigid(Spacer(0, 16)),
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, "Waiting for your computer…")
						}),
					)
				})),
			)
		}),
	)
}

// homeText is the hosted home's body type, 14/20.
func homeText(col color.NRGBA) Txt { return Txt{Size: 14, LineH: 20.0 / 14, Color: col} }

// setupStep is a numbered step: its number in an 8dp column, 4 down, then
// 12 to its content.
func (th *Theme) setupStep(n int, w W) W {
	return func(gtx C) D {
		num := dp(gtx, 8) + dp(gtx, 12)
		func() {
			defer op.Offset(image.Pt(0, dp(gtx, 4))).Push(gtx.Ops).Pop()
			th.Text(gtx, homeText(P.MutedForeground), strconv.Itoa(n))
		}()
		defer op.Offset(image.Pt(num, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints.Min.X = max(0, gtx.Constraints.Min.X-num)
		g.Constraints.Max.X -= num
		d := w(g)
		return D{Size: image.Pt(d.Size.X+num, d.Size.Y)}
	}
}

// runOnComputer is RunOnComputer: "Run on your computer:", the install
// commands 8 under it, and the desktop app's download link
// ("home.download").
func (th *Theme) runOnComputer(command string) W {
	return func(gtx C) D {
		caption := func(s string) layout.FlexChild {
			return rigid(func(gtx C) D { return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, s) })
		}
		gap := rigid(Spacer(0, 8))
		return Col(gtx,
			rigid(func(gtx C) D { return th.Text(gtx, homeText(bodyText()), "Run on your computer:") }),
			gap,
			caption("macOS, Linux"),
			gap,
			rigid(th.commandBlock(0, "curl -fsSL https://t3.codes/install.sh | sh")),
			gap,
			caption("Windows (PowerShell)"),
			gap,
			rigid(th.commandBlock(1, "irm https://t3.codes/install.ps1 | iex")),
			gap,
			caption("Then"),
			gap,
			rigid(th.commandBlock(2, command)),
			gap,
			rigid(func(gtx C) D {
				col := P.MutedForeground
				span := func(s string, f font.Font) richtext.SpanStyle {
					return richtext.SpanStyle{Content: s, Color: col, Font: f, Size: 12}
				}
				t := richtext.Text(new(richtext.InteractiveText), th.Shaper,
					span("Skip the install if you already have ", UI(font.Normal)), span("t3", Mono()),
					span(". If ", UI(font.Normal)), span("t3", Mono()),
					span(" is not found after installing, add the folder the installer names to your PATH.", UI(font.Normal)))
				t.LineHeightScale = 16.0 / 12
				return t.Layout(gtx)
			}),
			gap,
			rigid(th.textLink("home.download", "Or download the desktop app", Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, P.Foreground, P.Background, 2, 100)),
		)
	}
}

// commandBlock is CommandBlock: the command after a muted "$" in 14/20
// mono, truncated, in a 46 tall box rounded 8, --border on --muted 50%,
// padded 10 12, and its ghost copy button, a check for 1.5s once copied.
func (th *Theme) commandBlock(n int, command string) W {
	return func(gtx C) D {
		id := "copy:home.command." + strconv.Itoa(n)
		th.offerCopy(id, command)
		w := gtx.Constraints.Max.X
		px := max(1, dp(gtx, 1))
		h := dp(gtx, 46)
		r := image.Rect(0, 0, w, h)
		bg := over(pct(P.Muted, 50), P.Background)
		fill(gtx, bg, r, dp(gtx, 8))
		strokeRect(gtx, over(P.Border, bg), r, dp(gtx, 8), float32(px))
		icon := "copy"
		if th.copiedWithin(gtx, id, 1500*time.Millisecond) {
			icon = "check"
		}
		b := dp(gtx, 24)
		func() {
			defer op.Offset(image.Pt(w-px-dp(gtx, 12)-b, (h-b)/2)).Push(gtx.Ops).Pop()
			th.T3Button(id, Button{Variant: BtnGhost, Size: BtnSizeIconXs, Icon: icon, On: bg})(gtx)
		}()
		tw := w - 2*px - 2*dp(gtx, 12) - dp(gtx, 12) - b
		lg := gtx
		lg.Constraints = layout.Constraints{Max: image.Pt(tw, h)}
		m := op.Record(gtx.Ops)
		td := Row(lg,
			rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: Mono(), Color: P.MutedForeground, MaxLines: 1}, "$")
			}),
			rigid(Spacer(8, 0)),
			flexed(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: Mono(), Color: over(bodyText(), bg), MaxLines: 1}, command)
			}),
		)
		call := m.Stop()
		defer op.Offset(image.Pt(px+dp(gtx, 12), (h-td.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return D{Size: r.Max}
	}
}
