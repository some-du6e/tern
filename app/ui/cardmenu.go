package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
)

// The popups the details card's rows open: T3's Menu (menu.tsx,
// docs/ui/02-primitives.md §3.5) as wide as the row it hangs from, the
// branch picker's Combobox (BranchPicker.tsx, §3.13), and the dialog a Git
// action on the default branch asks first in (§3.21).

// inBorder lays w out inside its popup's 1dp border, which T3 counts
// outside the padding; popupFrame paints it over the content's edge.
func inBorder(w W) W {
	return func(gtx C) D { return pad(1, 1, 1, 1).Layout(gtx, w) }
}

// cardMenuState is the open card menu's keyboard: its focus, the menu it
// is for, the row the arrows or the pointer light, the row the pointer was
// over last frame (the pointer lights a row as it comes onto it, so the
// keyboard can move the light off one it rests on), and the typeahead.
type cardMenuState struct {
	tag     int
	open    string
	hi      int  // the lit row, -1 for none
	pointer bool // the pointer lit it, and unlights it when it leaves
	under   int
	typed   string
	typedAt time.Time
}

// menuRows reports whether row r takes the keyboard: items, radio and
// checkbox rows. Disabled ones stay in the order but do nothing.
func menuRows(r MenuRow) bool {
	return r.Kind == MenuItemRow || r.Kind == MenuRadioRow || r.Kind == MenuCheckRow
}

// nextRow is the next row of rows from i in dir that takes the keyboard,
// wrapping around; -1 if there's none.
func nextRow(rows []MenuRow, i, dir int) int {
	n := len(rows)
	if i < 0 && dir < 0 {
		i = n
	}
	for k := 1; k <= n; k++ {
		j := ((i+dir*k)%n + n) % n
		if menuRows(rows[j]) {
			return j
		}
	}
	return -1
}

// cardMenuKeys is a card menu's keyboard (02-primitives.md port note 9):
// opening moves the focus into it, lighting the first row when the
// keyboard opened it; the arrows move the light, wrapping, Home and End
// go to the ends, letters jump to the next row they start (500ms apart
// they start a new search, the same letter again cycles), Enter and Space
// pick the lit row, and Escape and Tab close it a level at a time.
func (th *Theme) cardMenuKeys(gtx C, m *CardMenu) {
	cs := &th.in.cardMenu
	if cs.open != m.For {
		*cs = cardMenuState{open: m.For, hi: -1, under: -1}
		if th.in.keyFocus {
			cs.hi = nextRow(m.Rows, -1, 1)
		}
		gtx.Execute(key.FocusCmd{Tag: &cs.tag})
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &cs.tag},
			key.Filter{Focus: &cs.tag, Name: ""},
			key.Filter{Focus: &cs.tag, Name: key.NameTab, Optional: key.ModShift},
		)
		if !ok {
			return
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		cs.pointer = false
		switch e.Name {
		case key.NameDownArrow:
			cs.hi = nextRow(m.Rows, cs.hi, 1)
		case key.NameUpArrow:
			cs.hi = nextRow(m.Rows, cs.hi, -1)
		case key.NameHome:
			cs.hi = nextRow(m.Rows, -1, 1)
		case key.NameEnd:
			cs.hi = nextRow(m.Rows, 0, -1)
		case key.NameEscape, key.NameTab:
			th.push(Action{ID: "menus.escape", Kind: "click"})
		case key.NameReturn, key.NameEnter, key.NameSpace:
			if cs.hi >= 0 && cs.hi < len(m.Rows) && !m.Rows[cs.hi].Disabled {
				th.push(Action{ID: fmt.Sprintf("card.%s:%d", m.For, cs.hi), Kind: "click"})
			}
		default:
			if len([]rune(string(e.Name))) != 1 {
				continue
			}
			if gtx.Now.Sub(cs.typedAt) > 500*time.Millisecond {
				cs.typed = ""
			}
			cs.typed, cs.typedAt = cs.typed+strings.ToLower(string(e.Name)), gtx.Now
			q, from := cs.typed, cs.hi
			if strings.Count(q, q[:1]) == len(q) {
				// One letter again and again steps through its rows.
				q = q[:1]
			} else {
				from--
			}
			for k := 1; k <= len(m.Rows); k++ {
				j := ((from+k)%len(m.Rows) + len(m.Rows)) % len(m.Rows)
				if menuRows(m.Rows[j]) && strings.HasPrefix(strings.ToLower(m.Rows[j].Label), q) {
					cs.hi = j
					break
				}
			}
		}
	}
}

// cardMenu is m's rows in T3's menu, w wide (menu.tsx): 28dp rows padded
// 4 by 8; items with 16dp icons in muted at 80%, radio rows with their 12dp
// icon in the row's colour, the checked one filled with foreground at 8%;
// checkbox rows with T3's check before the label; group labels,
// separators and notes. The lit row is filled with accent, over the
// checked fill.
func (th *Theme) cardMenu(gtx C, m *CardMenu, w int) D {
	cs := &th.in.cardMenu
	th.cardMenuKeys(gtx, m)
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	hovered := -1
	rec := op.Record(gtx.Ops)
	d := popupFrame(gtx, inBorder(func(gtx C) D {
		return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
			var kids []layout.FlexChild
			for i, r := range m.Rows {
				i, r, id := i, r, fmt.Sprintf("card.%s:%d", m.For, i)
				row := func(gtx C) D { return th.cardMenuRow(gtx, r, i == cs.hi) }
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
						// A disabled row takes no presses; one with a reason
						// explains itself in a popover to its left while the
						// pointer rests on it.
						off := "off." + id
						kids = append(kids, rigid(func(gtx C) D {
							d := row(gtx)
							if r.Reason != "" {
								th.hover(gtx, off, d.Size)
								th.reasonPopover(gtx, off, r.Reason, d.Size, true)
							}
							return d
						}))
						continue
					}
					kids = append(kids, rigid(func(gtx C) D {
						c, _ := th.events(gtx, id)
						if c.Hovered() {
							hovered = i
						}
						d := row(gtx)
						th.hit(gtx, id, d.Size)
						return d
					}))
				}
			}
			return Col(gtx, kids...)
		})
	}))
	call := rec.Stop()
	// Under its rows, the menu holds the keyboard's focus and takes the
	// presses between them, so they don't close it.
	func() {
		defer clip.Rect{Max: d.Size}.Push(gtx.Ops).Pop()
		event.Op(gtx.Ops, &cs.tag)
	}()
	call.Add(gtx.Ops)
	// The pointer lights the row it comes onto (unless the menu doesn't
	// light rows on hover), and unlights it when it leaves.
	if hovered != cs.under {
		cs.under = hovered
		switch {
		case m.NoHover:
		case hovered >= 0:
			cs.hi, cs.pointer = hovered, true
		case cs.pointer:
			cs.hi, cs.pointer = -1, false
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	return d
}

// menuAtPointer draws the menu w where the last right-click landed, as a
// Base UI menu anchored there opens (side bottom, align start, 4dp off):
// when it would run off the window's right it lines its right edge up
// with the click instead, and off the bottom it opens above it; then it
// keeps 5dp inside the win-sized window.
func (th *Theme) menuAtPointer(gtx C, win image.Point, w W) {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: win}
	d := w(g)
	call := m.Stop()
	edge, off := dp(gtx, 5), dp(gtx, 4)
	at := th.in.ctxAt.Add(image.Pt(0, off))
	if at.X+d.Size.X > win.X-edge {
		at.X = th.in.ctxAt.X - d.Size.X
	}
	if at.Y+d.Size.Y > win.Y-edge {
		at.Y = th.in.ctxAt.Y - off - d.Size.Y
	}
	at.X = max(edge, min(at.X, win.X-edge-d.Size.X))
	at.Y = max(edge, min(at.Y, win.Y-edge-d.Size.Y))
	m2 := op.Record(gtx.Ops)
	op.Offset(at).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
}

// cardMenuRow is one item, radio or checkbox row, lit or not. Radio rows
// lay out their own content (BranchToolbar.tsx): the 12dp icon a dp into
// each side's padding, 6 to the label, and a second line in 12px muted
// under it with the icon at the first line.
func (th *Theme) cardMenuRow(gtx C, r MenuRow, lit bool) D {
	text := P.PopoverForeground
	if lit {
		text = P.AccentForeground
	}
	m := op.Record(gtx.Ops)
	d := func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		right := float32(8)
		if r.Kind == MenuCheckRow {
			right = 16
		}
		return pad(4, right, 4, 8).Layout(gtx, func(gtx C) D {
			// At least 28 tall: Gio's Inset keeps the minimum it's given, so
			// the padding comes off it here.
			gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, dp(gtx, 28-8))
			label := func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: text, MaxLines: 1}, r.Label)
			}
			var kids []layout.FlexChild
			switch {
			case r.Kind == MenuRadioRow && r.Detail != "":
				// The previous worktree (PreviousWorktreeItemContent.tsx):
				// its icon 4dp down, by the first line, then the label's 20dp
				// line and the branch's 16dp one in 12px muted under it.
				return layout.Flex{Alignment: layout.Start}.Layout(gtx,
					rigid(func(gtx C) D { return pad(4, 4, 0, -2).Layout(gtx, LucideW(r.Icon, text, 12)) }),
					flexed(func(gtx C) D {
						w := gtx.Constraints.Max.X
						th.capText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: text}, r.Label, dp(gtx, 10))
						func() {
							defer op.Offset(image.Pt(0, dp(gtx, 20))).Push(gtx.Ops).Pop()
							th.middleTruncate(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground}, r.Detail, dp(gtx, 8))
						}()
						return D{Size: image.Pt(w, dp(gtx, 20+16))}
					}))
			case r.Kind == MenuRadioRow:
				kids = append(kids, rigid(func(gtx C) D { return pad(0, 4, 0, -2).Layout(gtx, LucideW(r.Icon, text, 12)) }))
			case r.Kind == MenuCheckRow:
				// A 16dp column for T3's own check, then 8 to the label.
				kids = append(kids, rigid(func(gtx C) D {
					if !r.Checked {
						return D{Size: image.Pt(dp(gtx, 16), dp(gtx, 16))}
					}
					return LucideW("menu-check", text, 16)(gtx)
				}), rigid(Spacer(8, 0)))
			case r.Icon != "":
				col := over(alpha(P.MutedForeground, 0xcc), P.Popover)
				icon := LucideW(r.Icon, col, 16)
				if kind, ok := strings.CutPrefix(r.Icon, "provider:"); ok {
					icon = ProviderMark(kind, col, 16)
				}
				if id, ok := strings.CutPrefix(r.Icon, "editor:"); ok {
					// Brand marks at full strength (opacity-100).
					icon = editorIcon(id, col, 16)
				}
				kids = append(kids, rigid(func(gtx C) D { return pad(0, -2, 0, -2).Layout(gtx, icon) }), rigid(Spacer(8, 0)))
			}
			kids = append(kids, flexed(label))
			if r.Shortcut != "" {
				kids = append(kids, rigid(func(gtx C) D {
					return pad(0, 0, 0, 16).Layout(gtx, func(gtx C) D {
						return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, Font: UI(font.Medium), MaxLines: 1}, r.Shortcut)
					})
				}))
			}
			return Row(gtx, kids...)
		})
	}(gtx)
	call := m.Stop()
	switch {
	case lit:
		fill(gtx, over(P.Accent, P.Popover), image.Rectangle{Max: d.Size}, dp(gtx, 4))
	case r.Kind == MenuRadioRow && r.Checked:
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
	return popupFrame(gtx, inBorder(func(gtx C) D {
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
		l := th.List("branch.refs", false)
		if q := th.EditorText("branch.search"); q != th.in.pickQuery {
			// A new search starts at the top of its list.
			th.in.pickQuery, l.Position = q, layout.Position{}
		}
		switch {
		case len(rows) > 0:
			kids = append(kids, rigid(func(gtx C) D {
				// The list scrolls past 224dp (max-h-56): the list padded 4
				// round, the refs in it 8 above and below and 4 at the start,
				// with the 6dp scrollbar gutter kept at the end.
				gtx.Constraints.Max.Y = dp(gtx, 224)
				return pad(4+8, 4+6, 4+8, 4+4).Layout(gtx, func(gtx C) D {
					if hi := p.Highlight; hi != th.in.pickHi {
						// The arrows bring the highlight into view.
						th.in.pickHi = hi
						switch first, n := l.Position.First, l.Position.Count; {
						case hi < first:
							l.ScrollTo(hi)
						case n > 0 && hi >= first+n-1:
							l.Position.First, l.Position.Offset = hi+1, -gtx.Constraints.Max.Y
						}
					}
					d := l.Layout(gtx, len(rows), func(gtx C, i int) D { return rows[i](gtx) })
					// Half a page from its end it asks for the next page.
					if p.More && l.Position.First+l.Position.Count >= len(rows)-max(1, l.Position.Count/2) {
						th.push(Action{ID: "branch.more", Kind: "click"})
					}
					return d
				})
			}))
		case !p.Loading:
			// ComboboxEmpty: 14/20 muted, centred, padded 8.
			kids = append(kids, rigid(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return pad(8, 8, 8, 8).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, dp(gtx, 20))
					return Row(gtx, flexed(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground, Align: text.Middle, MaxLines: 1}, "No refs found.")
					}))
				})
			}))
		}
		// The notice, then the status, each a ComboboxStatus, which wraps.
		for _, line := range []string{p.Notice, p.Status} {
			if line == "" {
				continue
			}
			kids = append(kids, rigid(func(gtx C) D {
				return pad(8, 12, 8, 12).Layout(gtx, func(gtx C) D {
					return th.boxText(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, Font: UI(font.Medium)}, line)
				})
			}))
		}
		return Col(gtx, kids...)
	}))
}

// refSearch is the picker's search block: padded 10 by 12, a 16dp search
// icon at muted 55% and the field over a 1dp rule at border 70%, which
// turns ring colour while it has the focus, all drawn a dp up
// (-translate-y-px) in the room they take. The arrows move the highlight
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
		defer op.Offset(image.Pt(0, -max(1, dp(gtx, 1)))).Push(gtx.Ops).Pop()
		rule := over(faded(P.Border, 0.7), P.Popover)
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
			return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, dp(gtx, 28-8))
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
	// A press on the backdrop closes it, like Abort.
	th.dialog(gtx, win, 576, "confirm:abort", func(gtx C) D { return th.confirmBody(gtx, c) })
}

// alertDialog is a as T3's AlertDialog (alert-dialog.tsx, docs/ui/
// 02-primitives.md §2.20, §3.22): up to 512 wide (max-w-lg), its title and
// description, then the default footer, a rule over muted at 72%, with
// Cancel and the action at its end. Its backdrop keeps its presses, as
// Base UI's alert dialogs ignore them.
func (th *Theme) alertDialog(gtx C, a *Alert, win image.Point) {
	th.dialog(gtx, win, 512, "", func(gtx C) D {
		w := gtx.Constraints.Max.X
		return Col(gtx,
			rigid(func(gtx C) D {
				gtx.Constraints.Min.X = w
				return pad(24, 24, 24, 24).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return Col(gtx,
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: 20, LineH: 27.5 / 20, Font: UI(font.SemiBold), Color: P.PopoverForeground}, a.Title)
						}),
						rigid(Spacer(0, 8)),
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, a.Description)
						}),
					)
				})
			}),
			rigid(func(gtx C) D {
				m := op.Record(gtx.Ops)
				gtx.Constraints.Min.X = w
				d := pad(16, 24, 16, 24).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return Row(gtx,
						flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
						rigid(th.T3Button("alert:cancel", Button{Variant: BtnOutline, Size: a.Size, Label: "Cancel", On: footerFill()})),
						rigid(Spacer(8, 0)),
						rigid(th.T3Button("alert:ok", Button{Variant: a.Variant, Size: a.Size, Label: a.Action, On: footerFill()})),
					)
				})
				call := m.Stop()
				// The footer's fill runs into the popup's bottom corners,
				// inside its border.
				px := max(1, dp(gtx, 1))
				rad := dp(gtx, 16) - px
				paint.FillShape(gtx.Ops, footerFill(), clip.RRect{Rect: image.Rect(px, 0, d.Size.X-px, d.Size.Y-px), SE: rad, SW: rad}.Op(gtx.Ops))
				hline(gtx, over(P.Border, P.Popover), 0, px, d.Size.X-px)
				call.Add(gtx.Ops)
				return d
			}),
		)
	})
}

// footerFill is a dialog's default footer: muted at 72% over the popup.
func footerFill() color.NRGBA { return over(faded(P.Muted, 0.72), P.Popover) }

// dialog draws body as T3's dialog in the middle of a win-sized window, up
// to maxW wide: the backdrop over everything, then the popup with the
// dialog glass's border, shadow and dark top light. A Dialog closes on a
// press on its backdrop, which clicks dismiss, and on its ✕, a 32dp ghost
// button 8dp inside the border at the top right ("dialog:close"). An
// AlertDialog, with no dismiss, has neither: its backdrop keeps its
// presses.
func (th *Theme) dialog(gtx C, win image.Point, maxW float32, dismiss string, body W) {
	m := op.Record(gtx.Ops)
	backdrop := alpha(rgb(0xfcfcfc), 0x99)
	if dark {
		backdrop = alpha(rgb(0), 0xa3)
	}
	paint.FillShape(gtx.Ops, backdrop, clip.Rect{Max: win}.Op())
	if dismiss != "" {
		th.hit(gtx, dismiss, win)
	} else {
		th.hover(gtx, "dialog.backdrop", win)
	}
	g := gtx
	w := min(dp(gtx, maxW), win.X-2*dp(gtx, 16))
	g.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, win.Y-2*dp(gtx, 16))}
	pm := op.Record(gtx.Ops)
	content := body(g)
	call := pm.Stop()
	r := image.Rectangle{Max: content.Size}
	at := image.Pt((win.X-w)/2, (win.Y-content.Size.Y)/2)
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
		th.hit(gtx, "dialog.popup", r.Max)
		call.Add(gtx.Ops)
		if dismiss != "" {
			px, s := max(1, dp(gtx, 1)), dp(gtx, 32)
			defer op.Offset(image.Pt(r.Max.X-px-dp(gtx, 8)-s, px+dp(gtx, 8))).Push(gtx.Ops).Pop()
			th.T3Button("dialog:close", Button{Variant: BtnGhost, Size: BtnSizeIcon, Icon: "x", On: P.Popover})(gtx)
		}
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

// commitDialog is c as T3's Commit dialog (GitActionsControl.tsx; Dialog,
// docs/ui/02-primitives.md §3.21) in the middle of a win-sized window: its
// header, then a panel of the summary and the message, then Cancel, Commit
// on new branch and Commit in the bare footer, the last two off while the
// commit takes no file.
func (th *Theme) commitDialog(gtx C, c *CommitDialog, win image.Point) {
	th.dialog(gtx, win, 512, "commit:cancel", func(gtx C) D { return th.commitBody(gtx, c) })
}

// commitBody is the dialog inside the popup's 1dp border: the header padded
// 24 (12 under it, as a panel follows), the panel padded 4 by 24 with 16
// between its parts, and the bare footer padded 16 by 24.
func (th *Theme) commitBody(gtx C, c *CommitDialog) D {
	taken := 0
	for _, f := range c.Files {
		if !f.Excluded {
			taken++
		}
	}
	block := func(top, bottom float32, w W) layout.FlexChild {
		return rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return pad(top, 24, bottom, 24).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return w(gtx)
			})
		})
	}
	return inBorder(func(gtx C) D {
		return Col(gtx,
			block(24, 12, func(gtx C) D {
				return Col(gtx,
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 20, LineH: 27.5 / 20, Font: UI(font.SemiBold), Color: P.PopoverForeground}, "Commit changes")
					}),
					rigid(Spacer(0, 8)),
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground}, "Review and confirm your commit. Leave the message blank to auto-generate one.")
					}),
				)
			}),
			block(4, 4, func(gtx C) D {
				return Col(gtx,
					rigid(func(gtx C) D { return th.commitSummary(gtx, c, taken) }),
					rigid(Spacer(0, 16)),
					rigid(func(gtx C) D {
						return th.boxText(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: P.PopoverForeground}, "Commit message (optional)")
					}),
					rigid(Spacer(0, 4)),
					rigid(th.T3Textarea("commit.message", "Leave empty to auto-generate", P.Popover)),
				)
			}),
			block(16, 16, func(gtx C) D {
				none := taken == 0
				return RowGap(gtx, 8,
					flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
					rigid(th.T3Button("commit:cancel", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Cancel", On: P.Popover})),
					rigid(th.T3Button("commit:branch", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: "Commit on new branch", Disabled: none, On: P.Popover})),
					rigid(th.T3Button("commit:ok", Button{Variant: BtnDefault, Size: BtnSizeSm, Label: "Commit", Disabled: none, On: P.Popover})),
				)
			}),
		)
	})(gtx)
}

// commitSummary is the dialog's summary box: rounded 12, zinc-25 (white at
// 3.5% in dark) in a 1dp ring of black (white) at 5%, padded 12, its 14/20
// rows 12 apart. "Branch" and the branch, "Default branch" in --warning at
// the end on the default one; then the files: the header's checkbox while
// editing, "Files", "(N of M)" when some are left out and Edit or Done at
// the end; 4dp under it their list and the totals of the ones it takes, or
// "none".
func (th *Theme) commitSummary(gtx C, c *CommitDialog, taken int) D {
	bg, ring := over(Tailwind["zinc-25"], P.Popover), over(alpha(rgb(0), 0x0d), P.Popover)
	if dark {
		bg, ring = over(alpha(rgb(0xffffff), 0x09), P.Popover), over(alpha(rgb(0xffffff), 0x0d), P.Popover)
	}
	text := Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}
	muted, strong, warn := text, text, text
	muted.Color, strong.Font, warn.Color = P.MutedForeground, UI(font.Medium), P.Warning
	gap := func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }
	m := op.Record(gtx.Ops)
	d := pad(12, 12, 12, 12).Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		branch := func(gtx C) D {
			kids := []layout.FlexChild{
				rigid(func(gtx C) D { return th.Text(gtx, muted, "Branch") }),
				flexed(func(gtx C) D { return th.Text(gtx, strong, c.Branch) }),
			}
			if c.Default {
				kids = append(kids, rigid(func(gtx C) D { return th.Text(gtx, warn, "Default branch") }))
			}
			return RowGap(gtx, 8, kids...)
		}
		some := len(c.Files) > 0
		header := func(gtx C) D {
			var kids []layout.FlexChild
			if c.Editing && some {
				kids = append(kids, rigid(th.T3Checkbox("commit.all", taken == len(c.Files), taken > 0 && taken < len(c.Files), bg)))
			}
			kids = append(kids, rigid(func(gtx C) D { return th.Text(gtx, muted, "Files") }))
			if taken < len(c.Files) && !c.Editing {
				kids = append(kids, rigid(func(gtx C) D { return th.Text(gtx, muted, fmt.Sprintf("(%d of %d)", taken, len(c.Files))) }))
			}
			kids = append(kids, flexed(gap))
			if some {
				label := "Edit"
				if c.Editing {
					label = "Done"
				}
				kids = append(kids, rigid(th.T3Button("commit:edit", Button{Variant: BtnGhost, Size: BtnSizeXs, Label: label, On: bg})))
			}
			return RowGap(gtx, 8, kids...)
		}
		files := func(gtx C) D {
			if !some {
				return th.Text(gtx, strong, "none")
			}
			ins, del := 0, 0
			for _, f := range c.Files {
				if !f.Excluded {
					ins, del = ins+f.Insertions, del+f.Deletions
				}
			}
			return Col(gtx,
				rigid(func(gtx C) D { return th.commitFiles(gtx, c, bg) }),
				rigid(Spacer(0, 8)),
				rigid(func(gtx C) D {
					return Row(gtx, flexed(gap), rigid(func(gtx C) D { return th.diffCounts(gtx, ins, del) }))
				}),
			)
		}
		return Col(gtx, rigid(branch), rigid(Spacer(0, 12)), rigid(header), rigid(Spacer(0, 4)), rigid(files))
	})
	call := m.Stop()
	r, rad, px := image.Rectangle{Max: d.Size}, dp(gtx, 12), max(1, dp(gtx, 1))
	fill(gtx, ring, r.Inset(-px), rad+px)
	fill(gtx, bg, r, rad)
	call.Add(gtx.Ops)
	return d
}

// commitFiles is the dialog's 176dp list of files (rounded 8, --card or
// white at 2.5% in dark, in a 1dp ring at 5%), scrolling past its height:
// rows of 14/20 mono padded 4 by 8, 4 apart inside its 4, accent at 50%
// under the pointer. Each starts with its checkbox while editing, then its
// path, cut at its start when it's too long, then its counts or
// "Excluded"; a click on the path or the counts opens the file, where the
// status says which folder it's in.
func (th *Theme) commitFiles(gtx C, c *CommitDialog, under color.NRGBA) D {
	card, ring := over(P.Card, under), over(alpha(rgb(0), 0x0d), under)
	if dark {
		card, ring = over(alpha(rgb(0xffffff), 0x06), under), over(alpha(rgb(0xffffff), 0x0d), under)
	}
	w, h, px := gtx.Constraints.Max.X, dp(gtx, 176), max(1, dp(gtx, 1))
	r, rad := image.Rect(0, 0, w, h), dp(gtx, 8)
	fill(gtx, ring, r.Inset(-px), rad+px)
	fill(gtx, card, r, rad)
	defer clip.UniformRRect(r, rad).Push(gtx.Ops).Pop()
	t := diffMono()
	base := dp(gtx, 4) + th.lineBase(gtx, t)
	l := th.List("commit.files", false)
	g := gtx
	g.Constraints = layout.Exact(r.Max)
	pad(4, 4, 4, 4).Layout(g, func(gtx C) D {
		return l.Layout(gtx, len(c.Files), func(gtx C, i int) D {
			f := c.Files[i]
			rw, rh := gtx.Constraints.Max.X, dp(gtx, 28)
			row := fmt.Sprintf("commit.row:%d", i)
			bg := card
			if th.hoverState(gtx, row) {
				bg = over(faded(P.Accent, 0.5), card)
				fill(gtx, bg, image.Rect(0, 0, rw, rh), dp(gtx, 6))
			}
			x := dp(gtx, 8)
			if c.Editing {
				func() {
					defer op.Offset(image.Pt(x, (rh-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
					th.T3Checkbox(fmt.Sprintf("commit.file:%d", i), !f.Excluded, false, bg)(gtx)
				}()
				x += dp(gtx, 16+8)
			}
			end := rw - dp(gtx, 8)
			path, countsW := t, th.textW(gtx, t, "Excluded")
			path.Color = P.PopoverForeground
			if f.Excluded {
				path.Color = P.MutedForeground
			} else {
				countsW = th.diffCountsW(gtx, f.Insertions, f.Deletions)
			}
			func() {
				defer op.Offset(image.Pt(end-countsW, 0)).Push(gtx.Ops).Pop()
				if f.Excluded {
					th.baseText(gtx, path, "Excluded", base)
					return
				}
				th.diffCountsAt(gtx, f.Insertions, f.Deletions, base)
			}()
			func() {
				defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
				g := gtx
				g.Constraints.Max.X = max(0, end-countsW-dp(gtx, 12)-x)
				th.startTruncate(g, path, f.Path, base)
			}()
			if c.Opens {
				func() {
					defer op.Offset(image.Pt(x, dp(gtx, 4))).Push(gtx.Ops).Pop()
					th.hit(gtx, fmt.Sprintf("commit.open:%d", i), image.Pt(max(0, end-x), dp(gtx, 20)))
				}()
			}
			th.hoverArea(gtx, row, image.Pt(rw, rh))
			if i < len(c.Files)-1 {
				rh += dp(gtx, 4)
			}
			return D{Size: image.Pt(rw, rh)}
		})
	})
	return D{Size: r.Max}
}

// diffMono is the Commit dialog's paths and counts: 14/20 mono.
func diffMono() Txt { return Txt{Size: 14, LineH: 20.0 / 14, Font: font.Font{Typeface: mono}} }

// diffCounts is "+N / -N" on a 20dp line: the insertions in
// --diff-addition and the deletions in --diff-deletion with a muted slash
// between, as the Commit dialog writes its counts.
func (th *Theme) diffCounts(gtx C, ins, del int) D {
	th.diffCountsAt(gtx, ins, del, th.lineBase(gtx, diffMono()))
	return D{Size: image.Pt(th.diffCountsW(gtx, ins, del), dp(gtx, 20))}
}

// diffCountsW is the width diffCounts takes.
func (th *Theme) diffCountsW(gtx C, ins, del int) int {
	return th.textW(gtx, diffMono(), "+"+groupDigits(ins)+" / -"+groupDigits(del))
}

// diffCountsAt draws diffCounts with its baseline on y.
func (th *Theme) diffCountsAt(gtx C, ins, del, y int) {
	t := diffMono()
	x := 0
	for _, part := range []struct {
		s   string
		col color.NRGBA
	}{{"+" + groupDigits(ins), P.DiffAddition}, {" / ", P.MutedForeground}, {"-" + groupDigits(del), P.DiffDeletion}} {
		t.Color = part.col
		func() {
			defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
			x += th.baseText(gtx, t, part.s, y)
		}()
	}
}
