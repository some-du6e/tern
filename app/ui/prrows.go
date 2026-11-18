package ui

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"gioui.org/op"
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
// less". Measuring for the density, the others it shows don't count.
func (th *Theme) prRows(p *PullRequests, measuring bool) W {
	return func(gtx C) D {
		w := gtx.Constraints.Max.X
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
// watches it a split whose eye (crossed out under the pointer) stops that.
func (th *Theme) prRow(gtx C, i int, r PullRequestRow) D {
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
