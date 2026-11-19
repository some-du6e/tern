package ui

import (
	"image"
	"strconv"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
)

// The Diff surface's menus (docs/ui/13-diff.md §2.3, §3.2): the scope
// menu, T3's Menu with a radio group and two submenus, and the base ref
// combobox.

// menuBox is T3's menu popup around rows inner wide: the popup frame, its
// 1dp border, 4dp padding, the rows stacked.
func menuBox(gtx C, inner int, rows []W) D {
	px := max(1, dp(gtx, 1))
	return popupFrame(gtx, func(gtx C) D {
		y := px + dp(gtx, 4)
		for _, r := range rows {
			func() {
				defer op.Offset(image.Pt(px+dp(gtx, 4), y)).Push(gtx.Ops).Pop()
				g := gtx
				g.Constraints = layout.Constraints{Min: image.Pt(inner, 0), Max: image.Pt(inner, 1<<20)}
				y += r(g).Size.Y
			}()
		}
		return D{Size: image.Pt(inner+2*px+dp(gtx, 8), y+dp(gtx, 4)+px)}
	})
}

// menuInner is the row width of a menu at least 160 wide.
func menuInner(gtx C, need int) int {
	return max(need, dp(gtx, 160)-2*max(1, dp(gtx, 1))-dp(gtx, 8))
}

// diffScopeMenu is the scope menu: Changes, Uncommitted and Latest turn,
// then the Turn and Commits submenus, which open on hover, flush with
// their row, 5dp above it and over the menu's edge by 5dp.
func (th *Theme) diffScopeMenu(gtx C, s *Scene, d *DiffPanel) D {
	m := d.Menu
	sub := th.dv.sub
	if m.Sub != "" {
		sub = m.Sub
	}
	inner := menuInner(gtx, 0)
	radio := func(value, label string) W {
		return th.diffMenuRow("diff.scope:"+value, label, d.Value == value, false, false)
	}
	trigger := func(name, label string) W {
		id := "diff.sub:" + name
		return func(gtx C) D {
			if c, _ := th.events(gtx, id); c.Hovered() && s.Live {
				th.dv.sub = name
			}
			return th.diffMenuRow(id, label, false, sub == name, true)(gtx)
		}
	}
	rows := []W{radio("branch", "Changes"), radio("unstaged", "Uncommitted"), radio("latest", "Latest turn"),
		trigger("turn", "Turn"), trigger("commits", "Commits")}
	dims := menuBox(gtx, inner, rows)
	if sub == "" {
		return dims
	}
	// The submenu goes over the menu, so it's drawn deferred.
	mm := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 384), dp(gtx, 320))}
	var sd D
	if sub == "turn" {
		sd = th.diffTurnMenu(g, d)
	} else {
		sd = th.diffCommitMenu(g, d)
	}
	call := mm.Stop()
	i := 3
	if sub == "commits" {
		i = 4
	}
	top := max(1, dp(gtx, 1)) + dp(gtx, 4) + i*dp(gtx, 28) - dp(gtx, 5)
	x := dims.Size.X - dp(gtx, 5)
	// It flips to the menu's left when the window ends first.
	if th.dv.panelW-th.dv.menuX-x-dp(gtx, 5) < sd.Size.X {
		x = dp(gtx, 5) - sd.Size.X
	}
	m2 := op.Record(gtx.Ops)
	op.Offset(image.Pt(x, top)).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
	return dims
}

// diffMenuRow is a menu row as wide as its constraints: 14/20 text padded
// 4 by 8, 28 tall, radius 4; accent under the pointer or while its
// submenu is open, foreground at 8% when checked (radio rows show no
// check); a sub-trigger ends in a 16dp chevron at 80%.
func (th *Theme) diffMenuRow(id, label string, checked, open, sub bool) W {
	return func(gtx C) D {
		c, _ := th.events(gtx, id)
		w, h := gtx.Constraints.Min.X, dp(gtx, 28)
		r := image.Rect(0, 0, w, h)
		switch {
		case c.Hovered() || open:
			fill(gtx, over(P.Accent, P.Popover), r, dp(gtx, 4))
		case checked:
			fill(gtx, over(alpha(P.Foreground, 0x14), P.Popover), r, dp(gtx, 4))
		}
		tw := w - dp(gtx, 16)
		if sub {
			tw -= dp(gtx, 16)
			func() {
				defer op.Offset(image.Pt(w-dp(gtx, 8+16-2), (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				LucideW("chevron-right", over(alpha(P.MutedForeground, 0xcc), P.Popover), 16)(gtx)
			}()
		}
		func() {
			defer op.Offset(image.Pt(dp(gtx, 8), (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(max(0, tw), dp(gtx, 20))}
			th.Text(g, Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}, label)
		}()
		th.hit(gtx, id, r.Max)
		return D{Size: r.Max}
	}
}

// diffTurnMenu lists the thread's turns, newest first: "Turn N", and the
// time it finished right-aligned in 12/16 muted, at least 8dp after.
func (th *Theme) diffTurnMenu(gtx C, d *DiffPanel) D {
	m := d.Menu
	label := Txt{Size: 14, LineH: 20.0 / 14, MaxLines: 1}
	when := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}
	need := 0
	for _, t := range m.Turns {
		need = max(need, dp(gtx, 8)+th.lineWidth(gtx, label, t.Label)+dp(gtx, 8)+th.lineWidth(gtx, when, t.Time)+dp(gtx, 8))
	}
	inner := menuInner(gtx, need)
	var rows []W
	for _, t := range m.Turns {
		t := t
		rows = append(rows, func(gtx C) D {
			rd := th.diffMenuRow("diff.scope:"+t.Value, t.Label, m.Turn == t.Value, false, false)(gtx)
			tw := th.lineWidth(gtx, when, t.Time)
			defer op.Offset(image.Pt(rd.Size.X-dp(gtx, 8)-tw, (rd.Size.Y-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			th.Text(gtx, when, t.Time)
			return rd
		})
	}
	return menuBox(gtx, inner, rows)
}

// diffCommitMenu lists the branch's commits in a menu at most 384 wide:
// the subject cut short, its age 12dp after, then the short SHA; and under
// them, disabled at 64%, why there are none or that only the latest are
// shown.
func (th *Theme) diffCommitMenu(gtx C, d *DiffPanel) D {
	m := d.Menu
	subject := Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}
	muted := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}
	sha := muted
	sha.Font = Mono()
	need := 0
	for _, c := range m.Commits {
		need = max(need, dp(gtx, 8)+th.lineWidth(gtx, subject, c.Subject)+dp(gtx, 12)+th.lineWidth(gtx, muted, c.When)+
			dp(gtx, 8+8)+th.lineWidth(gtx, sha, shortSHA(c.SHA))+dp(gtx, 8))
	}
	if m.Note != "" {
		need = max(need, th.lineWidth(gtx, subject, m.Note)+dp(gtx, 16))
	}
	most := dp(gtx, 384) - 2*max(1, dp(gtx, 1)) - dp(gtx, 8)
	inner := min(menuInner(gtx, need), most)
	var rows []W
	for _, c := range m.Commits {
		c := c
		rows = append(rows, func(gtx C) D {
			id := "diff.commit:" + c.SHA
			ev, _ := th.events(gtx, id)
			w, h := gtx.Constraints.Min.X, dp(gtx, 28)
			r := image.Rect(0, 0, w, h)
			switch {
			case ev.Hovered():
				fill(gtx, over(P.Accent, P.Popover), r, dp(gtx, 4))
			case m.Commit == c.SHA:
				fill(gtx, over(alpha(P.Foreground, 0x14), P.Popover), r, dp(gtx, 4))
			}
			// Right to left: the SHA in its 8dp-padded cell, then the age.
			short := shortSHA(c.SHA)
			x := w - dp(gtx, 8) - th.lineWidth(gtx, sha, short)
			func() {
				defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				th.Text(gtx, sha, short)
			}()
			x -= dp(gtx, 8+8) + th.lineWidth(gtx, muted, c.When)
			func() {
				defer op.Offset(image.Pt(x, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				th.Text(gtx, muted, c.When)
			}()
			func() {
				defer op.Offset(image.Pt(dp(gtx, 8), (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
				g := gtx
				g.Constraints = layout.Constraints{Max: image.Pt(max(0, x-dp(gtx, 12+8)), dp(gtx, 20))}
				th.Text(g, subject, c.Subject)
			}()
			th.hit(gtx, id, r.Max)
			th.tipAt(gtx, id, c.Subject, r.Max, false, -1)
			return D{Size: r.Max}
		})
	}
	if m.Note != "" {
		rows = append(rows, func(gtx C) D {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
			w, h := gtx.Constraints.Min.X, dp(gtx, 28)
			defer op.Offset(image.Pt(dp(gtx, 8), (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(w-dp(gtx, 16), dp(gtx, 20))}
			th.Text(g, subject, m.Note)
			return D{Size: image.Pt(w, h)}
		})
	}
	return menuBox(gtx, inner, rows)
}

// shortSHA is a commit's first 7 characters.
func shortSHA(sha string) string { return sha[:min(7, len(sha))] }

// diffBasePicker is the base ref combobox (§2.3): 288 wide, a search
// field, a "Branch / Remote" header, then Automatic and the refs, each
// with a switch between its local and remote copies when it has both, or
// a check when only the remote exists.
func (th *Theme) diffBasePicker(gtx C, s *Scene, c *DiffCompare) D {
	p := c.Picker
	px := max(1, dp(gtx, 1))
	inner := dp(gtx, 288) - 2*px - dp(gtx, 8)
	rule := over(twAlpha(P.Border, 70), P.Popover)
	label := Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}
	rows := []W{func(gtx C) D {
		// The search field: a 16dp icon, the editor, a rule under them.
		h := dp(gtx, 43)
		fill(gtx, rule, image.Rect(dp(gtx, 4), h-dp(gtx, 4)-px, inner-dp(gtx, 4), h-dp(gtx, 4)), 0)
		func() {
			defer op.Offset(image.Pt(dp(gtx, 4), dp(gtx, 6+5))).Push(gtx.Ops).Pop()
			LucideW("search", over(alpha(P.MutedForeground, 0x8c), P.Popover), 16)(gtx)
		}()
		defer op.Offset(image.Pt(dp(gtx, 4+20), dp(gtx, 6))).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(inner-dp(gtx, 4+20+9+4), dp(gtx, 26)))
		layout.W.Layout(g, func(gtx C) D {
			if !s.Live {
				return th.Text(gtx, Txt{Size: 14, Color: P.MutedForeground, MaxLines: 1}, "Search refs...")
			}
			return th.editor(gtx, "diff.base.search", Txt{Size: 14, Color: P.PopoverForeground}, "Search refs...")
		})
		return D{Size: image.Pt(inner, h)}
	}, func(gtx C) D {
		// BRANCH over the refs, REMOTE over their switches: 10/14 medium.
		head := Txt{Size: 10, LineH: 14.0 / 10, Font: UI(font.Medium), Color: P.MutedForeground, MaxLines: 1}
		h := dp(gtx, 8+14+6)
		func() {
			defer op.Offset(image.Pt(dp(gtx, 8+16+8), dp(gtx, 8))).Push(gtx.Ops).Pop()
			th.Text(gtx, head, "BRANCH")
		}()
		func() {
			defer op.Offset(image.Pt(inner-dp(gtx, 26)-th.lineWidth(gtx, head, "REMOTE"), dp(gtx, 8))).Push(gtx.Ops).Pop()
			th.Text(gtx, head, "REMOTE")
		}()
		hline(gtx, rule, h-px, -dp(gtx, 4), inner+dp(gtx, 4))
		return D{Size: image.Pt(inner, h+dp(gtx, 4))}
	}}
	item := func(id, text string, checked bool, end W) W {
		return func(gtx C) D {
			rd := th.diffMenuRow(id, "", false, false, false)(gtx)
			if checked {
				func() {
					defer op.Offset(image.Pt(dp(gtx, 8), (rd.Size.Y-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
					LucideW("check", P.PopoverForeground, 14)(gtx)
				}()
			}
			func() {
				defer op.Offset(image.Pt(dp(gtx, 8+16+8), (rd.Size.Y-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
				g := gtx
				g.Constraints = layout.Constraints{Max: image.Pt(rd.Size.X-dp(gtx, 8+16+8+32+8), dp(gtx, 20))}
				th.Text(g, label, text)
			}()
			if end != nil {
				defer op.Offset(image.Pt(rd.Size.X-dp(gtx, 8+26), (rd.Size.Y-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				end(gtx)
			}
			return rd
		}
	}
	n := len(rows)
	if s.Live && th.EditorText("diff.base.search") == "" || !s.Live {
		rows = append(rows, item("diff.base:", "Automatic", p.Automatic, nil))
	}
	for i, r := range p.Refs {
		// The list is 256dp tall; the search finds the rest.
		if i == 9 {
			break
		}
		r := r
		ref, end := r.Local, W(nil)
		switch {
		case r.Local != "" && r.Remote != "":
			remote := p.Selected == r.Remote
			if remote {
				ref = r.Remote
			}
			end = th.Click("diff.baseswitch:"+strconv.Itoa(i), func(gtx C) D { return miniSwitch(gtx, remote) })
		case r.Remote != "":
			ref = r.Remote
			end = func(gtx C) D {
				defer op.Offset(image.Pt(dp(gtx, 14), dp(gtx, 2))).Push(gtx.Ops).Pop()
				return LucideW("check", P.MutedForeground, 12)(gtx)
			}
		}
		rows = append(rows, item("diff.base:"+ref, r.Label, !p.Automatic && p.Selected == ref, end))
	}
	if len(rows) == n {
		rows = append(rows, func(gtx C) D {
			return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground, MaxLines: 1}, "No matching refs.")
			})
		})
	}
	return menuBox(gtx, inner, rows)
}

// miniSwitch is the base picker's small switch with its 12dp thumb:
// primary when on, the input colour off.
func miniSwitch(gtx C, on bool) D {
	w, h := dp(gtx, 26), dp(gtx, 16)
	track, x := over(P.Input, P.Popover), dp(gtx, 2)
	if on {
		track, x = P.Primary, w-dp(gtx, 2+12)
	}
	fill(gtx, track, image.Rect(0, 0, w, h), h/2)
	circle(gtx, P.Background, image.Pt(x+dp(gtx, 6), h/2), dp(gtx, 6))
	return D{Size: image.Pt(w, h)}
}
