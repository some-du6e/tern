package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
)

// A pending approval as the timeline shows it (TimelinePendingApprovalRow.tsx,
// docs/ui/06-timeline.md §2 "approval"): a work row with the request's
// kind icon and detail, then its answers. T3's composer draws no card for
// it; a send declines it instead (07-composer.md §2.15).

// approvalIcons are the row's icons by request kind.
var approvalIcons = map[string]string{"command": "terminal", "file-change": "square-pen", "file-read": "eye", "permission": "wrench", "mcp-elicitation": "hammer"}

// approvalLabels name what an approval asks for when it says nothing.
var approvalLabels = map[string]string{"command": "Command approval", "file-change": "File change approval", "file-read": "File read approval",
	"permission": "App permission approval", "mcp-elicitation": "App access approval"}

// defaultApprovalOptions are T3's answers when the request offers none.
var defaultApprovalOptions = []ApprovalOption{{Decision: "cancel", Label: "Cancel"}, {Decision: "decline", Label: "Decline"},
	{Decision: "acceptForSession", Label: "Always allow this session"}, {Decision: "accept", Label: "Approve"}}

// approvalRow is the row: the kind's icon in its 24dp slot, the detail in
// mono 12/20 at most 160 tall (it scrolls past that), then from 30dp in
// the app and the reason in muted 12/16 and the answers.
func (th *Theme) approvalRow(gtx C, a *PendingApproval) D {
	w := gtx.Constraints.Max.X
	gtx.Constraints.Min = image.Point{}
	icon := approvalIcons[a.Kind]
	if icon == "" {
		icon = "wrench"
	}
	detail := a.Detail
	if detail == "" {
		detail = approvalLabels[a.Kind]
	}
	off := op.Offset(image.Pt(dp(gtx, 4), dp(gtx, 4))).Push(gtx.Ops)
	LucideW(icon, P.IconMuted, 16)(gtx)
	off.Pop()
	in := dp(gtx, 30)
	inner := gtx
	inner.Constraints = layout.Constraints{Max: image.Pt(max(0, w-in), 1<<20)}
	text := record(inner, func(gtx C) D {
		return th.Text(gtx, Txt{Size: 12, LineH: 20.0 / 12, Font: Mono(), Color: P.Foreground}, detail)
	})
	h := min(text.Size.Y, dp(gtx, 160))
	func() {
		defer op.Offset(image.Pt(in, dp(gtx, 2))).Push(gtx.Ops).Pop()
		th.vscroll(gtx, "approval:"+a.ID, image.Pt(w-in, h), text.Size.Y, text.call)
	}()
	y := max(dp(gtx, 24), h+dp(gtx, 2))
	line := func(w W) {
		y += dp(gtx, 4)
		o := op.Offset(image.Pt(in, y)).Push(gtx.Ops)
		y += w(inner).Size.Y
		o.Pop()
	}
	for _, s := range []string{a.App, a.Reason} {
		if s != "" {
			line(func(gtx C) D { return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, s) })
		}
	}
	line(func(gtx C) D { return th.approvalActions(gtx, a) })
	return D{Size: image.Pt(w, y+dp(gtx, 4))}
}

// approvalActions is the row's 12/16 answers (ComposerPendingApprovalActions,
// inline): the shield and "Waiting for approval" in --warning, a dot,
// then Decline and Approve as inline buttons, Approve in the foreground,
// and the rest under More. A provider that's gone can't take one, and
// while an answer is on its way they're off.
func (th *Theme) approvalActions(gtx C, a *PendingApproval) D {
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: P.Warning}
	if !a.Live {
		return th.Text(gtx, t, "Provider process is gone. Interrupt or restart the run to respond.")
	}
	opts := a.Options
	if len(opts) == 0 {
		opts = defaultApprovalOptions
	}
	gtx.Constraints.Min = image.Point{}
	kids := []layout.FlexChild{
		rigid(LucideW("shield", P.Warning, 12)),
		rigid(Spacer(4, 0)),
		rigid(func(gtx C) D { return th.Text(gtx, t, "Waiting for approval") }),
		rigid(Spacer(8, 0)),
		rigid(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, "·")
		}),
	}
	var more []ApprovalOption
	for _, o := range opts {
		if o.Decision != "decline" && o.Decision != "accept" {
			more = append(more, o)
			continue
		}
		id := ""
		if !a.Responding {
			id = "approval:" + o.Decision + ":" + a.ID
		}
		tone := toneMuted
		if o.Decision == "accept" {
			tone = toneDefault
		}
		label := o.Label
		kids = append(kids, rigid(Spacer(8, 0)), rigid(func(gtx C) D {
			return th.inlineButton(gtx, id, label, Txt{Size: 12, LineH: 16.0 / 12}, tone)
		}))
	}
	if len(more) > 0 {
		kids = append(kids, rigid(Spacer(8, 0)), rigid(func(gtx C) D { return th.approvalMore(gtx, a, more) }))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
}

// approvalMore is "More": a muted inline button whose menu, over it and
// flush with its end, holds the answers the row doesn't show. A click
// anywhere else shuts the menu.
func (th *Theme) approvalMore(gtx C, a *PendingApproval, more []ApprovalOption) D {
	ui := &th.in.composer
	id := "approval.more:" + a.ID
	t := Txt{Size: 12, LineH: 16.0 / 12}
	if a.Responding {
		return th.inlineButton(gtx, "", "More", t, toneMuted)
	}
	rec := record(gtx, func(gtx C) D { return th.inlineLabel(gtx, "More", t, toneMuted, false) })
	clicked, hovered := th.localClick(gtx, id, rec.Size)
	if clicked {
		if ui.approvalMenu == a.ID {
			ui.approvalMenu = ""
		} else {
			ui.approvalMenu = a.ID
		}
	}
	d := th.inlineLabel(gtx, "More", t, toneMuted, hovered)
	if ui.approvalMenu != a.ID {
		return d
	}
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 320), dp(gtx, 320))}
	menu := record(g, func(gtx C) D {
		return popupFrame(gtx, func(gtx C) D {
			return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
				var rows []layout.FlexChild
				for _, o := range more {
					rid, label, warn := "approval:"+o.Decision+":"+a.ID, o.Label, o.Warning
					rows = append(rows, rigid(th.menuItem(rid, func(gtx C) D {
						return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								rigid(func(gtx C) D {
									if warn == "" {
										return D{}
									}
									return pad(0, 6, 0, 0).Layout(gtx, LucideW("triangle-alert", P.Warning, 12))
								}),
								rigid(func(gtx C) D {
									return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}, label)
								}),
							)
						})
					})))
				}
				return Col(gtx, rows...)
			})
		})
	})
	m := op.Record(gtx.Ops)
	func() {
		// Under the menu, the rest of the window shuts it.
		defer op.Offset(image.Pt(-1<<14, -1<<14)).Push(gtx.Ops).Pop()
		if shut, _ := th.localClick(gtx, id+":outside", image.Pt(1<<15, 1<<15)); shut {
			ui.approvalMenu = ""
		}
	}()
	op.Offset(image.Pt(d.Size.X-menu.Size.X, -menu.Size.Y-dp(gtx, 4))).Add(gtx.Ops)
	menu.call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m.Stop())
	return d
}
