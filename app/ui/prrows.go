package ui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
)

// The details card's pull request rows (ThreadDetailsPrRows.tsx,
// ThreadDetailsPrRow.tsx, pullRequestIcons.tsx; docs/ui/05-header.md §3.8):
// a row per pull request with its state's icon in its tone, and a muted
// row that folds the thread's other linked ones away.

// prWatchTip is the eye's tooltip while the agent watches a pull request.
const prWatchTip = "Watching: the agent wakes when checks finish, someone comments, or the branch conflicts. Click to stop."

// prStateIcon is a pull request state's icon and tone: emerald open, zinc
// draft, red closed and violet merged, their dark shades a little
// translucent; muted while the state isn't known.
func prStateIcon(state string) (string, color.NRGBA) {
	tone := func(light, darkShade string, a uint8) color.NRGBA {
		if dark {
			return over(alpha(Tailwind[darkShade], a), cardFill())
		}
		return Tailwind[light]
	}
	switch state {
	case "open":
		return "git-pull-request-arrow", tone("emerald-600", "emerald-300", 0xe6)
	case "draft":
		return "git-pull-request-draft", tone("zinc-500", "zinc-400", 0xcc)
	case "closed":
		return "git-pull-request-closed", tone("red-600", "red-300", 0xe6)
	case "merged":
		return "git-merge", tone("violet-600", "violet-300", 0xe6)
	}
	return "git-pull-request-arrow", P.MutedForeground
}

// prRows are the thread's pull request rows, then "Show N more" or "Show
// less". Measuring for the density, the others it shows don't count. A
// checks popover opens at the top of its list.
func (th *Theme) prRows(p *PullRequests, measuring bool) W {
	return func(gtx C) D {
		w := gtx.Constraints.Max.X
		popup := false
		for _, r := range p.Rows {
			popup = popup || (r.Detail != nil && r.Detail.Checks != nil && r.Detail.Checks.Popup != nil)
		}
		if !popup {
			delete(th.in.lists, "card.checks")
		}
		y := 0
		for i, r := range p.Rows {
			if i > 0 && measuring {
				break
			}
			off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
			y += th.prRow(gtx, i, r).Size.Y
			off.Pop()
		}
		if p.Others > 0 {
			icon, label := "chevron-down", "Show "+strconv.Itoa(p.Others)+" more"
			if p.Expanded {
				icon, label = "chevron-up", "Show less"
			}
			defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
			y += th.mutedRow(gtx, "card.prs", icon, label).Size.Y
		}
		return D{Size: image.Pt(w, y)}
	}
}

// prRow is one pull request: a row that opens it, or while the agent
// watches it a split whose eye (crossed out under the pointer) stops that;
// or its detailed form.
func (th *Theme) prRow(gtx C, i int, r PullRequestRow) D {
	if r.Detail != nil {
		return th.prDetailRow(gtx, i, r)
	}
	icon, col := prStateIcon(r.State)
	open := ctl{id: fmt.Sprintf("card.pr:%d", i), tip: r.Tip}
	label := func(gtx C, w int) {
		rowIcon(gtx, icon, col, 0, 16)
		th.rowLabel(gtx, rowFont(rowText()), r.Label, dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+10))
	}
	if r.Watching {
		watch := ctl{id: fmt.Sprintf("card.prwatch:%d", i), tip: prWatchTip}
		return th.splitWith(gtx, open, watch, label, func(gtx C, x int, hovered bool) {
			eye := "eye"
			if hovered {
				eye = "eye-off"
			}
			defer op.Offset(image.Pt(x+dp(gtx, 8), dp(gtx, 8))).Push(gtx.Ops).Pop()
			LucideW(eye, rowText(), 16)(gtx)
		})
	}
	w := gtx.Constraints.Max.X
	rr := image.Rect(0, 0, w, dp(gtx, rowH))
	th.control(gtx, open, rr, corners{8, 8, 8, 8}.scaled(gtx), cardFill(), func(gtx C) { label(gtx, w-dp(gtx, 1)) })
	return D{Size: rr.Max}
}

// mutedRow is a row in the muted tone, its icon in the text's colour,
// which turns foreground 80% under the pointer; it doesn't shrink when
// pressed (active:scale-100).
func (th *Theme) mutedRow(gtx C, id, icon, label string) D {
	c, _ := th.events(gtx, id)
	col := P.MutedForeground
	if c.Hovered() {
		col = rowText()
	}
	w := gtx.Constraints.Max.X
	r := image.Rect(0, 0, w, dp(gtx, rowH))
	th.control(gtx, ctl{id: id, popup: true}, r, corners{8, 8, 8, 8}.scaled(gtx), cardFill(), func(gtx C) {
		rowIcon(gtx, icon, col, 0, 16)
		th.rowLabel(gtx, rowFont(col), label, dp(gtx, rowLabelX), w-dp(gtx, rowLabelX+11))
	})
	return D{Size: r.Max}
}

// ---------- the detailed form ----------

// prTitleFont is a detailed row's title: 13/18 medium, two lines at most.
func prTitleFont() Txt {
	return Txt{Size: 13, LineH: 18.0 / 13, Font: UI(font.Medium), Color: rowText(), MaxLines: 2}
}

// metaFont is the meta line's text: 12/16 in muted foreground.
func metaFont() Txt { return Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground} }

// prDetailRow is a pull request's detailed form (ThreadDetailsPrRow.tsx,
// §3.8): a multiline row that opens it, its state's icon a dp into its
// first line and its title on up to two 18dp lines; 6dp under it the meta
// line, 20dp tall; and 6dp under that.
func (th *Theme) prDetailRow(gtx C, i int, r PullRequestRow) D {
	w := gtx.Constraints.Max.X
	icon, col := prStateIcon(r.State)
	px := max(1, dp(gtx, 1))
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(max(0, w-dp(gtx, rowLabelX+11)), 1<<20)}
	td := th.boxText(g, prTitleFont(), r.Detail.Title)
	title := m.Stop()
	h := max(dp(gtx, rowH), td.Size.Y+2*(px+dp(gtx, 6)))
	rr := image.Rect(0, 0, w, h)
	th.control(gtx, ctl{id: fmt.Sprintf("card.pr:%d", i), tip: r.Tip}, rr, corners{8, 8, 8, 8}.scaled(gtx), cardFill(), func(gtx C) {
		top := px + dp(gtx, 6)
		func() {
			defer op.Offset(image.Pt(dp(gtx, 11), top+px)).Push(gtx.Ops).Pop()
			LucideW(icon, col, 16)(gtx)
		}()
		defer op.Offset(image.Pt(dp(gtx, rowLabelX), top)).Push(gtx.Ops).Pop()
		title.Add(gtx.Ops)
	})
	y := h + dp(gtx, 6)
	func() {
		defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
		th.prMeta(gtx, i, r, w)
	}()
	return D{Size: image.Pt(w, y+dp(gtx, 20+6))}
}

// prMeta is a detailed row's meta line, w wide and 20dp tall, its 24dp
// controls overhanging it 2dp each way: from 36dp in "#N", 4dp and the
// line's 2dp gap on the checks and then the eye while the agent watches
// it, and the pill 10dp in from the end.
func (th *Theme) prMeta(gtx C, i int, r PullRequestRow, w int) {
	dr := r.Detail
	mid := dp(gtx, 10)
	x := dp(gtx, 36)
	func() {
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		x += th.capText(gtx, metaFont(), "#"+strconv.Itoa(dr.Number), mid) + dp(gtx, 4+2)
	}()
	ctlTop := mid - dp(gtx, 12)
	if c := dr.Checks; c != nil {
		func() {
			defer op.Offset(image.Pt(x, ctlTop)).Push(gtx.Ops).Pop()
			x += th.checksTrigger(gtx, i, c).X + dp(gtx, 2)
		}()
	}
	if r.Watching {
		func() {
			defer op.Offset(image.Pt(x, ctlTop)).Push(gtx.Ops).Pop()
			th.watchIcon(gtx, i)
		}()
	}
	if p := dr.Pill; p != nil {
		pw := th.pillW(gtx, p)
		defer op.Offset(image.Pt(w-dp(gtx, 10)-pw, mid-dp(gtx, 10))).Push(gtx.Ops).Pop()
		th.metaPill(gtx, fmt.Sprintf("card.prpill:%d", i), p, pw)
	}
}

// checkTone is a check's or the checks' tone over under
// (pullRequestPresentation.tsx): emerald when they pass, destructive when
// they fail, amber while one waits on someone, amber-500 while one runs,
// muted at 70% for the rest; the dark shades a little translucent. The
// rollup's pending is "pending-rollup", apart from a running check's.
func checkTone(status string, under color.NRGBA) color.NRGBA {
	tint := func(light, darkShade string, a uint8) color.NRGBA {
		if dark {
			return over(alpha(Tailwind[darkShade], a), under)
		}
		return Tailwind[light]
	}
	switch status {
	case "passing", "success":
		return tint("emerald-600", "emerald-300", 0xe6)
	case "failing", "failure":
		return P.Destructive
	case "pending-rollup", "action-required":
		return tint("amber-600", "amber-400", 0xe6)
	case "pending":
		return Tailwind["amber-500"]
	}
	return over(alpha(P.MutedForeground, 0xb3), under)
}

// checkIcons are the checks' icons by status, and the rollup's.
var checkIcons = map[string]string{"passing": "circle-check", "failing": "circle-x", "pending-rollup": "circle-dot",
	"action-required": "circle-dot", "success": "circle-check", "failure": "circle-x", "cancelled": "octagon-x",
	"skipped": "circle-slash", "neutral": "circle-dashed"}

// checksTrigger is the meta line's checks control (part meta), opening
// their popover: 24dp, padded 6, the rollup's 14dp icon in its tone and
// the count in 12/16 medium muted, 4dp apart. The popover hangs 4dp under
// it at its end: the card sits at the window's right, where Base UI flips
// T3's start alignment.
func (th *Theme) checksTrigger(gtx C, i int, c *ChecksMeta) image.Point {
	id := fmt.Sprintf("card.checks:%d", i)
	count := rowFont(P.MutedForeground)
	count.Size, count.LineH = 12, 16.0/12
	w := dp(gtx, 1+6+14+6+1)
	if c.Count != "" {
		w += dp(gtx, 4) + th.textW(gtx, count, c.Count)
	}
	size := image.Pt(w, dp(gtx, 24))
	r := image.Rectangle{Max: size}
	state := c.State
	if state == "pending" {
		state = "pending-rollup"
	}
	th.control(gtx, ctl{id: id, popup: true, open: c.Popup != nil}, r, corners{6, 6, 6, 6}.scaled(gtx), cardFill(), func(gtx C) {
		s := dp(gtx, 14)
		func() {
			defer op.Offset(image.Pt(dp(gtx, 7), (size.Y-s)/2)).Push(gtx.Ops).Pop()
			LucideW(checkIcons[state], checkTone(state, cardFill()), 14)(gtx)
		}()
		if c.Count != "" {
			defer op.Offset(image.Pt(dp(gtx, 7+14+4), 0)).Push(gtx.Ops).Pop()
			th.capText(gtx, count, c.Count, size.Y/2)
		}
	})
	if c.Popup != nil {
		th.deferBelow(gtx, image.Pt(0, size.Y+dp(gtx, 4)), func(gtx C) D {
			m := op.Record(gtx.Ops)
			d := th.checksPopup(gtx, c.Popup)
			call := m.Stop()
			defer op.Offset(image.Pt(size.X-d.Size.X, 0)).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)
			return d
		})
	}
	return size
}

// watchIcon is the meta line's eye (part icon): 24dp, radius 6, the eye
// at 14dp, crossed out under the pointer; it stops the watch.
func (th *Theme) watchIcon(gtx C, i int) {
	id := fmt.Sprintf("card.prwatch:%d", i)
	c, _ := th.events(gtx, id)
	eye := "eye"
	if c.Hovered() {
		eye = "eye-off"
	}
	s := dp(gtx, 24)
	th.control(gtx, ctl{id: id, tip: prWatchTip}, image.Rect(0, 0, s, s), corners{6, 6, 6, 6}.scaled(gtx), cardFill(), func(gtx C) {
		defer op.Offset(image.Pt(dp(gtx, 5), dp(gtx, 5))).Push(gtx.Ops).Pop()
		LucideW(eye, P.MutedForeground, 14)(gtx)
	})
}

// pillFont is a pill's label: 12/16 medium.
func pillFont() Txt {
	return Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: P.PrimaryForeground}
}

// pillW is a pill's width: its label padded 6 inside a 1dp border.
func (th *Theme) pillW(gtx C, p *PullRequestPill) int {
	return th.textW(gtx, pillFont(), p.Label) + dp(gtx, 2*(1+6))
}

// metaPill is the action a pull request is ready for (part meta, tone
// primary): 20dp, radius 6, --primary at 85% under the pointer, its label
// in --primary-foreground; keyboard focus is a 1dp ring of --background
// inside its border. Off at 64% while an action runs.
func (th *Theme) metaPill(gtx C, id string, p *PullRequestPill, w int) {
	size := image.Pt(w, dp(gtx, 20))
	r := image.Rectangle{Max: size}
	hovered, pressed, focused := th.state(gtx, ctl{id: id, disabled: p.Disabled})
	bg := P.Primary
	if hovered {
		bg = over(alpha(P.Primary, 0xd9), cardFill())
	}
	k := corners{6, 6, 6, 6}.scaled(gtx)
	func() {
		if p.Disabled {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		}
		if t := th.transition(gtx, id+".press", pressed, transitionDur); t > 0 {
			mid := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
			k := 1 - 0.03*t
			defer op.Affine(f32.Affine2D{}.Scale(mid, f32.Pt(k, k))).Push(gtx.Ops).Pop()
		}
		k.fill(gtx, bg, r)
		if focused {
			b := max(1, dp(gtx, 1))
			strokeRect(gtx, P.Background, r.Inset(b), k.nw-b, float32(b))
		}
		defer op.Offset(image.Pt(dp(gtx, 7), 0)).Push(gtx.Ops).Pop()
		th.capText(gtx, pillFont(), p.Label, size.Y/2)
	}()
	if !p.Disabled {
		th.hit(gtx, id, size)
		th.tipAt(gtx, id, p.Tip, size, true, 0)
	}
}

// checksPopup is the checks popover (PullRequestChecksPopover.tsx), 320dp
// wide and padded 16 inside its border: what the checks come to in 14/20
// medium, its summary in 12/16 muted, 8dp under each, then a line per
// check, the icon, its name, its status muted and its "Details" in
// --primary, 8 apart and 4 between lines, scrolling past 256dp; then Show
// all or Show less when the finished checks fold.
func (th *Theme) checksPopup(gtx C, p *ChecksPopup) D {
	gtx.Constraints = layout.Exact(image.Pt(dp(gtx, 320), 0))
	gtx.Constraints.Max.Y = 1 << 20
	return popupFrame(gtx, inBorder(func(gtx C) D {
		return pad(16, 16, 16, 16).Layout(gtx, func(gtx C) D {
			w := gtx.Constraints.Max.X
			kids := []layout.FlexChild{
				rigid(func(gtx C) D {
					return th.boxText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.PopoverForeground}, p.Heading)
				}),
				rigid(Spacer(0, 8)),
				rigid(func(gtx C) D { return th.boxText(gtx, metaFont(), p.Summary) }),
				rigid(Spacer(0, 8)),
			}
			list := th.List("card.checks", false)
			kids = append(kids, rigid(func(gtx C) D {
				gtx.Constraints.Max.Y = dp(gtx, 256)
				return list.Layout(gtx, len(p.Checks), func(gtx C, i int) D {
					d := th.checkLine(gtx, i, p.Checks[i], w)
					if i < len(p.Checks)-1 {
						d.Size.Y += dp(gtx, 4)
					}
					return d
				})
			}))
			if p.More != "" {
				kids = append(kids, rigid(th.T3Button("card.checksall", Button{Variant: BtnGhost, Size: BtnSizeXs, Label: p.More, On: P.Popover})))
			}
			return Col(gtx, kids...)
		})
	}))
}

// checkLine is one check in the popover, w wide and 16dp tall: its 14dp
// status icon, its name cut to fit, its status and "Details", 8dp apart.
func (th *Theme) checkLine(gtx C, i int, c CheckRow, w int) D {
	h := dp(gtx, 16)
	mid := h / 2
	icon := c.Status
	func() {
		s := dp(gtx, 14)
		defer op.Offset(image.Pt(0, (h-s)/2)).Push(gtx.Ops).Pop()
		if icon == "pending" {
			T3Spinner(checkTone(icon, P.Popover), 14, float64(gtx.Now.UnixMilli()%1000)/1000)(gtx)
			gtx.Execute(op.InvalidateCmd{})
			return
		}
		LucideW(checkIcons[icon], checkTone(icon, P.Popover), 14)(gtx)
	}()
	gap := dp(gtx, 8)
	end := w
	if c.Details {
		link := metaFont()
		link.Color = P.Primary
		lw := th.textW(gtx, link, "Details")
		end -= lw
		id := fmt.Sprintf("card.check:%d", i)
		func() {
			defer op.Offset(image.Pt(end, 0)).Push(gtx.Ops).Pop()
			if c, _ := th.events(gtx, id); c.Hovered() {
				// hover:underline, a dp under the baseline.
				base := mid + int(capEm*float32(gtx.Sp(12))/2+0.5)
				hline(gtx, P.Primary, base+max(1, dp(gtx, 1)), 0, lw)
			}
			th.capText(gtx, link, "Details", mid)
			th.hit(gtx, id, image.Pt(lw, h))
		}()
		end -= gap
	}
	lw := th.textW(gtx, metaFont(), c.Label)
	end -= lw
	func() {
		defer op.Offset(image.Pt(end, 0)).Push(gtx.Ops).Pop()
		th.capText(gtx, metaFont(), c.Label, mid)
	}()
	name := metaFont()
	name.Color = P.PopoverForeground
	x := dp(gtx, 14) + gap
	defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints.Max.X = max(0, end-gap-x)
	th.capText(g, name, c.Name, mid)
	return D{Size: image.Pt(w, h)}
}
