package ui

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The popups the details card's rows open: T3's Menu (menu.tsx,
// docs/ui/02-primitives.md §3.5) as wide as the row it hangs from, the
// branch picker's Combobox (BranchPicker.tsx, §3.13), and the dialog a Git
// action on the default branch asks first in (§3.21).

// cardMenu is m's rows in T3's menu, w wide: 28dp rows padded 4 by 8 with
// 16dp icons at 80%, radio rows with their check at the end, group labels,
// separators and notes.
func (th *Theme) cardMenu(gtx C, m *CardMenu, w int) D {
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	return popupFrame(gtx, func(gtx C) D {
		return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
			var kids []layout.FlexChild
			for i, r := range m.Rows {
				r, id := r, fmt.Sprintf("card.%s:%d", m.For, i)
				switch r.Kind {
				case MenuSeparator:
					kids = append(kids, rigid(func(gtx C) D {
						w := gtx.Constraints.Max.X
						hline(gtx, over(P.Border, P.Popover), dp(gtx, 4), dp(gtx, 8), w-dp(gtx, 8))
						return D{Size: image.Pt(w, dp(gtx, 9))}
					}))
				case MenuGroupLabel:
					kids = append(kids, rigid(func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return FixedH(gtx, 28, func(gtx C) D {
							return pad(6, 8, 6, 8).Layout(gtx, func(gtx C) D {
								return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, Font: UI(font.Medium), MaxLines: 1}, r.Label)
							})
						})
					}))
				case MenuNote:
					kids = append(kids, rigid(func(gtx C) D {
						col := P.Warning
						if r.Tone == "destructive" {
							col = P.Destructive
						}
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return pad(6, 8, 6, 8).Layout(gtx, func(gtx C) D {
							return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: col}, r.Label)
						})
					}))
				default:
					if r.Disabled {
						id = ""
					}
					kids = append(kids, rigid(th.menuItem(id, func(gtx C) D { return th.cardMenuRow(gtx, r) })))
				}
			}
			return Col(gtx, kids...)
		})
	})
}

// cardMenuRow is one item or radio row.
func (th *Theme) cardMenuRow(gtx C, r MenuRow) D {
	m := op.Record(gtx.Ops)
	d := func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		gtx.Constraints.Min.Y = dp(gtx, 28)
		return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			text := P.PopoverForeground
			var kids []layout.FlexChild
			if r.Icon != "" {
				if r.Kind == MenuRadioRow {
					// Radio rows lay out their own content: a 12dp icon, gap 6.
					kids = append(kids, rigid(LucideW(r.Icon, alpha(P.MutedForeground, 0xcc), 12)), rigid(Spacer(6, 0)))
				} else {
					icon := LucideW(r.Icon, alpha(P.MutedForeground, 0xcc), 16)
					if kind, ok := strings.CutPrefix(r.Icon, "provider:"); ok {
						icon = ProviderMark(kind, alpha(P.MutedForeground, 0xcc), 16)
					}
					kids = append(kids, rigid(func(gtx C) D { return pad(0, -2, 0, -2).Layout(gtx, icon) }), rigid(Spacer(8, 0)))
				}
			}
			kids = append(kids, flexed(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: text, MaxLines: 1}, r.Label)
			}))
			if r.Shortcut != "" {
				kids = append(kids, rigid(func(gtx C) D {
					return pad(0, 0, 0, 16).Layout(gtx, func(gtx C) D {
						return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, Font: UI(font.Medium), MaxLines: 1}, r.Shortcut)
					})
				}))
			}
			if r.Kind == MenuRadioRow && r.Checked {
				kids = append(kids, rigid(func(gtx C) D {
					return pad(0, -2, 0, 6).Layout(gtx, LucideW("check", text, 14))
				}))
			}
			return Row(gtx, kids...)
		})
	}(gtx)
	call := m.Stop()
	if r.Kind == MenuRadioRow && r.Checked && !th.in.menuHover {
		fill(gtx, over(alpha(P.Foreground, 0x14), P.Popover), image.Rectangle{Max: d.Size}, dp(gtx, 4))
	}
	if r.Disabled {
		defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
	}
	call.Add(gtx.Ops)
	return d
}

// branchPicker is the branch combobox, w wide: the search field over its
// rule, then the refs (the highlighted one at foreground 9%, a muted tag
// after each name), the ref it would create, and a status line.
func (th *Theme) branchPicker(gtx C, p *BranchPicker, w int) D {
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	return popupFrame(gtx, func(gtx C) D {
		var kids []layout.FlexChild
		kids = append(kids, rigid(func(gtx C) D { return th.refSearch(gtx) }))
		var rows []W
		for i, r := range p.Refs {
			i, r := i, r
			id := fmt.Sprintf("branch.ref:%d", i)
			if r.Disabled {
				id = ""
			}
			rows = append(rows, th.pickerItem(id, i == p.Highlight, r.Disabled, func(gtx C) D {
				return Row(gtx,
					flexed(func(gtx C) D {
						g := gtx
						th.middleTruncate(g, Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground}, r.Name, dp(gtx, 10))
						return D{Size: image.Pt(gtx.Constraints.Max.X, dp(gtx, 20))}
					}),
					rigid(func(gtx C) D {
						if r.Tag == "" {
							return D{}
						}
						return pad(0, 0, 0, 8).Layout(gtx, func(gtx C) D {
							return th.Text(gtx, Txt{Size: 10, LineH: 1.4, Color: over(alpha(P.MutedForeground, 0x73), P.Popover), MaxLines: 1}, r.Tag)
						})
					}),
				)
			}))
		}
		if p.Create != "" {
			rows = append(rows, th.pickerItem("branch.create", p.Highlight == len(p.Refs), false, func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}, p.Create)
			}))
		}
		if len(rows) > 0 {
			kids = append(kids, rigid(func(gtx C) D {
				// The list scrolls past 224dp (max-h-56), padded 8 0 8 4.
				gtx.Constraints.Max.Y = dp(gtx, 224)
				return pad(8, 4, 8, 4).Layout(gtx, func(gtx C) D {
					l := th.List("branch.refs", false)
					return l.Layout(gtx, len(rows), func(gtx C, i int) D { return rows[i](gtx) })
				})
			}))
		}
		if p.Status != "" {
			kids = append(kids, rigid(func(gtx C) D {
				return pad(8, 12, 8, 12).Layout(gtx, func(gtx C) D {
					return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, Font: UI(font.Medium), MaxLines: 1}, p.Status)
				})
			}))
		}
		return Col(gtx, kids...)
	})
}

// refSearch is the picker's search block: padded 10 by 12, a 16dp search
// icon at muted 55% and the field over a 1dp rule at border 70%, which
// turns ring colour while it has the focus. The arrows move the highlight
// and Enter picks it ("branch.search" key and submit actions).
func (th *Theme) refSearch(gtx C) D {
	const id = "branch.search"
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit = true, true
	for {
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameUpArrow}, key.Filter{Focus: ed, Name: key.NameDownArrow})
		if !ok {
			break
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			th.push(Action{ID: id, Kind: "key", Text: string(ke.Name)})
		}
	}
	return pad(10, 12, 0, 12).Layout(gtx, func(gtx C) D {
		w := gtx.Constraints.Max.X
		h := dp(gtx, 26)
		rule := over(alpha(P.Border, uint8(int(P.Border.A)*70/100)), P.Popover)
		if gtx.Focused(ed) {
			rule = P.Ring
		}
		func() {
			defer op.Offset(image.Pt(0, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			LucideW("search", over(alpha(P.MutedForeground, 0x8c), P.Popover), 16)(gtx)
		}()
		func() {
			defer op.Offset(image.Pt(dp(gtx, 20), 0)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Exact(image.Pt(max(0, w-dp(gtx, 20+9)), h))
			th.editor(g, id, Txt{Size: 14, LineH: 26.0 / 14, Color: P.PopoverForeground}, "Search refs...")
		}()
		bottom := h + dp(gtx, 6)
		hline(gtx, rule, bottom, 0, w)
		return D{Size: image.Pt(w, bottom+max(1, dp(gtx, 1)))}
	})
}

// pickerItem is a combobox item: 28dp, padded 4 by 8, radius 4, filled
// with foreground at 9% while highlighted or hovered.
func (th *Theme) pickerItem(id string, highlighted, disabled bool, w W) W {
	return func(gtx C) D {
		hovered := false
		if id != "" {
			c, _ := th.events(gtx, id)
			hovered = c.Hovered()
		}
		m := op.Record(gtx.Ops)
		d := func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			gtx.Constraints.Min.Y = dp(gtx, 28)
			return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return w(gtx)
			})
		}(gtx)
		call := m.Stop()
		if highlighted || hovered {
			fill(gtx, over(alpha(P.Foreground, 0x17), P.Popover), image.Rectangle{Max: d.Size}, dp(gtx, 4))
		}
		if disabled {
			defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
		}
		call.Add(gtx.Ops)
		if id != "" {
			th.hit(gtx, id, d.Size)
		}
		return d
	}
}

// confirmDialog is c as T3's dialog in the middle of a win-sized window:
// the backdrop over everything, then a popup up to 576 wide (max-w-xl)
// with its title, description and, when the feature branch can't be
// named, why; then Abort at the start and the two ways on at the end.
func (th *Theme) confirmDialog(gtx C, c *Confirm, win image.Point) {
	m := op.Record(gtx.Ops)
	backdrop := alpha(rgb(0xfcfcfc), 0x99)
	if dark {
		backdrop = alpha(rgb(0), 0xa3)
	}
	paint.FillShape(gtx.Ops, backdrop, clip.Rect{Max: win}.Op())
	// A press on the backdrop closes it, like Abort.
	th.hit(gtx, "confirm:abort", win)
	g := gtx
	w := min(dp(gtx, 576), win.X-2*dp(gtx, 16))
	g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, win.Y-2*dp(gtx, 16))}
	pm := op.Record(gtx.Ops)
	body := th.confirmBody(g, c)
	call := pm.Stop()
	r := image.Rectangle{Max: body.Size}
	at := image.Pt((win.X-w)/2, (win.Y-body.Size.Y)/2)
	func() {
		defer op.Offset(at).Push(gtx.Ops).Pop()
		rad := dp(gtx, 16)
		border := alpha(rgb(0x14141d), 0x1a)
		shadows := []boxShadow{{0, 24, 64, -24, alpha(rgb(0), 0xa6)}}
		if dark {
			border = alpha(rgb(0xffffff), 0x14)
			shadows = []boxShadow{{0, 24, 72, -20, alpha(rgb(0), 0xe6)}}
		}
		shadow(gtx, r, rad, shadows)
		fill(gtx, P.Popover, r, rad)
		strokeRect(gtx, over(border, P.Popover), r, rad, float32(max(1, dp(gtx, 1))))
		if dark {
			hline(gtx, over(alpha(rgb(0xffffff), 0x0a), P.Popover), max(1, dp(gtx, 1)), rad, r.Max.X-rad)
		}
		// The popup keeps its presses.
		th.hit(gtx, "confirm.popup", r.Max)
		call.Add(gtx.Ops)
	}()
	op.Defer(gtx.Ops, m.Stop())
}

// confirmBody is the dialog's header and its bare footer.
func (th *Theme) confirmBody(gtx C, c *Confirm) D {
	w := gtx.Constraints.Max.X
	return Col(gtx,
		rigid(func(gtx C) D {
			gtx.Constraints.Min.X = w
			return pad(24, 24, 24, 24).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				kids := []layout.FlexChild{
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 20, LineH: 27.5 / 20, Font: UI(font.SemiBold), Color: P.PopoverForeground}, c.Title)
					}),
					rigid(Spacer(0, 8)),
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, c.Description)
					}),
				}
				if c.BranchOff != "" {
					kids = append(kids, rigid(Spacer(0, 8)), rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, c.BranchOff)
					}))
				}
				return Col(gtx, kids...)
			})
		}),
		rigid(func(gtx C) D {
			gtx.Constraints.Min.X = w
			return pad(16, 24, 16, 24).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Row(gtx,
					rigid(th.T3Button("confirm:abort", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Abort", On: P.Popover})),
					flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
					rigid(th.T3Button("confirm:continue", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: c.Continue, On: P.Popover})),
					rigid(Spacer(8, 0)),
					rigid(th.T3Button("confirm:branch", Button{Variant: BtnDefault, Size: BtnSizeSm, Label: "Check out feature branch & continue", Disabled: c.BranchOff != "", On: P.Popover})),
				)
			})
		}),
	)
}
