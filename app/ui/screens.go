package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/x/richtext"
)

const (
	sidebarW = 268
	panelW   = 440
	menubarH = 30
	statusH  = 24
)

// divider is a vertical rule as tall as the row's minimum height.
func divider(gtx C) D {
	h := gtx.Constraints.Min.Y
	vline(gtx, ColBorder, 0, 0, h)
	return D{Size: image.Pt(max(1, dp(gtx, 1)), h)}
}

// ---------- menu bar ----------

func (th *Theme) menubar(gtx C, s *Scene) D {
	return FixedH(gtx, menubarH, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		fill(gtx, ColPanel, image.Rectangle{Max: size}, 0)
		hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
		gtx.Constraints.Min.X = size.X
		return pad(0, 8, 0, 12).Layout(gtx, func(gtx C) D {
			kids := []layout.FlexChild{
				rigid(Logo(15)),
				rigid(Spacer(7, 0)),
				rigid(func(gtx C) D { return th.Label(gtx, ColText, font.SemiBold, "tern") }),
				rigid(Spacer(16, 0)),
			}
			menus := []string{"File", "Edit", "Thread", "View", "Window", "Help"}
			if s.Live {
				menus = nil
			}
			for _, m := range menus {
				m := m
				kids = append(kids, rigid(func(gtx C) D {
					return padXY(7, 0).Layout(gtx, func(gtx C) D { return th.Label(gtx, ColDim, font.Normal, m) })
				}))
			}
			kids = append(kids,
				flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
				rigid(func(gtx C) D {
					if s.Live {
						return D{}
					}
					return FixedW(gtx, 300, func(gtx C) D {
						return Box{Bg: ColFrame, Border: ColBorder, Radius: 4, Pad: padXY(8, 0)}.Layout(gtx, func(gtx C) D {
							return FixedH(gtx, 21, func(gtx C) D {
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return Row(gtx,
									rigid(IconW(IconSearch, ColFaint, 13)),
									rigid(Spacer(6, 0)),
									flexed(func(gtx C) D { return th.Small(gtx, ColFaint, "Jump to thread or run a command") }),
									rigid(th.Kbd(Shortcut("K"))),
								)
							})
						})
					})
				}),
				flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
				rigid(func(gtx C) D { return th.Small(gtx, ColFaint, s.FrameTime) }),
			)
			return Row(gtx, kids...)
		})
	})
}

// Logo is tern's mark: a swept wing.
func Logo(sizeDp float32) W {
	return func(gtx C) D {
		u := float32(dp(gtx, sizeDp)) / 16
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(f32.Pt(1*u, 10*u))
		p.QuadTo(f32.Pt(6*u, 3*u), f32.Pt(15*u, 2*u))
		p.QuadTo(f32.Pt(10*u, 5*u), f32.Pt(8.5*u, 9*u))
		p.QuadTo(f32.Pt(11*u, 9.5*u), f32.Pt(13*u, 14*u))
		p.QuadTo(f32.Pt(8*u, 10.5*u), f32.Pt(1*u, 10*u))
		p.Close()
		paint.FillShape(gtx.Ops, ColAccent, clip.Outline{Path: p.End()}.Op())
		return D{Size: image.Pt(int(16*u), int(16*u))}
	}
}

// ---------- sidebar ----------

// sidebar is the sidebar below its header row: search, the list and the
// footer. sidebarFrame paints the surface under it.
func (th *Theme) sidebar(gtx C, s *Scene) D {
	return Col(gtx,
		rigid(func(gtx C) D {
			return pad(8, 8, 8, 8).Layout(gtx, func(gtx C) D {
				search := th.Input(IconSearch, true, s.Search, "Search threads", false)
				if s.Live {
					search = th.Field("search", IconSearch, true, "Search threads", false)
				}
				return Row(gtx,
					flexed(search),
					rigid(Spacer(6, 0)),
					rigid(th.Click(liveID(s, "new"), th.Button(Plain, IconPlus, true, ""))),
				)
			})
		}),
		rigid(func(gtx C) D {
			if s.Filter == "" {
				return D{}
			}
			return pad(0, 10, 2, 10).Layout(gtx, th.Combo(nil, s.Filter, Stretch))
		}),
		flexed(func(gtx C) D {
			var rows []W
			section := func(label string, n int, chevron int, trailing, id string) {
				rows = append(rows, th.Click(liveID(s, id), th.shelf(label, n, chevron, trailing)))
			}
			add := func(ts []ThreadRow) {
				for _, t := range ts {
					rows = append(rows, th.Click(prefixID("thread:", t.ID), th.threadRow(t, s.Phase)))
				}
			}
			if len(s.Pinned) > 0 {
				section("PINNED", 0, shelfFixed, "", "")
				add(s.Pinned)
			}
			section("ACTIVE", len(s.Active), shelfFixed, "", "")
			add(s.Active)
			if len(s.Snoozed) > 0 {
				open := shelfOpen
				if s.Live && !s.ShowSnoozed {
					open = shelfClosed
				}
				section("SNOOZED", len(s.Snoozed), open, "", "shelf:snoozed")
				if !s.Live || s.ShowSnoozed {
					add(s.Snoozed)
				}
			}
			if s.Settled > 0 {
				if s.Live && s.ShowSettled {
					section("SETTLED", s.Settled, shelfOpen, "", "shelf:settled")
					add(s.SettledRows)
				} else {
					section("SETTLED", s.Settled, shelfClosed, "Show", "shelf:settled")
				}
			}
			if s.Live && len(rows) == 1 && len(s.Active) == 0 {
				rows = append(rows, func(gtx C) D {
					return pad(8, 14, 0, 16).Layout(gtx, func(gtx C) D { return th.Small(gtx, ColFaint, s.SidebarNote) })
				})
			}
			return th.scrollList(gtx, th.List("sidebar", false), rows)
		}),
		rigid(func(gtx C) D {
			if s.Live {
				return th.sidebarFooter(gtx, s)
			}
			w := gtx.Constraints.Max.X
			hline(gtx, ColBorder, 0, 0, w)
			return pad(7, 10, 7, 12).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				kids := []layout.FlexChild{}
				for _, e := range s.Envs {
					e := e
					kids = append(kids,
						rigid(func(gtx C) D {
							return padXY(2, 3).Layout(gtx, func(gtx C) D {
								return Row(gtx,
									rigid(Dot(envColor(e.Status), e.Status == EnvOnline)),
									rigid(Spacer(4, 0)),
									rigid(func(gtx C) D {
										col := ColDim
										if e.Status == EnvOffline {
											col = ColFaint
										}
										return th.Text(gtx, Txt{Size: SizeSmall, Color: col, Font: UI(font.Medium), MaxLines: 1}, e.Label)
									}),
								)
							})
						}),
						rigid(Spacer(7, 0)),
					)
				}
				kids = append(kids,
					flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
					rigid(th.Button(Ghost, IconQueue, true, "")),
				)
				return Row(gtx, kids...)
			})
		}),
	)
}

// liveID is id in a live scene and "" in a mockup, so mock buttons stay inert.
func liveID(s *Scene, id string) string {
	if !s.Live {
		return ""
	}
	return id
}

func prefixID(prefix, id string) string {
	if id == "" {
		return ""
	}
	return prefix + id
}

// scrollList lays rows out in a persistent list that fills the area, with an
// ImGui-style scrollbar when they overflow.
func (th *Theme) scrollList(gtx C, l *layout.List, rows []W) D {
	size := gtx.Constraints.Max
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	gtx.Constraints.Min = image.Point{X: size.X}
	l.Layout(gtx, len(rows), func(gtx C, i int) D { return rows[i](gtx) })
	p := l.Position
	if p.Length > size.Y && p.Length > 0 {
		before := 0
		if p.Count > 0 {
			before = p.Offset + p.First*p.Length/max(1, len(rows))
		}
		start := float32(before) / float32(p.Length)
		frac := float32(size.Y) / float32(p.Length)
		Scrollbar(gtx, size, min(start, 1-frac), frac)
	}
	return D{Size: size}
}

func envColor(st EnvStatus) color.NRGBA {
	switch st {
	case EnvOnline:
		return ColGreen
	case EnvBusy:
		return ColAmber
	}
	return ColFaint
}

const (
	shelfFixed = iota
	shelfOpen
	shelfClosed
)

// shelf is a section heading in the inbox. Snoozed and Settled fold.
func (th *Theme) shelf(label string, n int, chevron int, trailing string) W {
	return func(gtx C) D {
		return pad(10, 12, 3, 12).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return FixedH(gtx, 18, func(gtx C) D {
				return Row(gtx,
					rigid(func(gtx C) D {
						if chevron == shelfFixed {
							return D{}
						}
						ic := IconChevronDown
						if chevron == shelfClosed {
							ic = IconChevronRight
						}
						return pad(0, 3, 0, 0).Layout(gtx, IconW(ic, ColFaint, 12))
					}),
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 10.5, Color: ColFaint, Font: UI(font.SemiBold), MaxLines: 1}, label)
					}),
					rigid(func(gtx C) D {
						if n == 0 {
							return D{}
						}
						return pad(0, 0, 0, 6).Layout(gtx, func(gtx C) D {
							return th.Text(gtx, Txt{Size: 10.5, Color: alpha(ColFaint, 0xb0), Font: UI(font.Medium), MaxLines: 1}, fmt.Sprint(n))
						})
					}),
					flexed(func(gtx C) D {
						y := gtx.Constraints.Min.Y / 2
						x0 := dp(gtx, 8)
						if gtx.Constraints.Max.X > x0 {
							hline(gtx, alpha(ColBorder, 0xb0), y, x0, gtx.Constraints.Max.X-dp(gtx, 8))
						}
						return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)}
					}),
					rigid(func(gtx C) D {
						if trailing == "" {
							return D{}
						}
						return th.Small(gtx, ColDim, trailing)
					}),
				)
			})
		})
	}
}

// projectColor gives each project a stable tint so rows from different
// projects can be told apart in the mixed list.
func projectColor(name string) color.NRGBA {
	h := fnv.New32a()
	h.Write([]byte(name))
	cols := []color.NRGBA{ColAccentHi, ColGreen, ColViolet, ColCyan, ColCode, rgb(0xf27fb0)}
	return cols[h.Sum32()%uint32(len(cols))]
}

// statusChip is the one chip a row shows, in T3's priority order.
func (th *Theme) statusChip(t ThreadRow, phase float64) W {
	switch t.State {
	case ThreadApproval:
		return th.Chip(ColAmber, "Approval", nil)
	case ThreadQuestion:
		return th.Chip(ColViolet, "Input", nil)
	case ThreadRunning:
		label := "Working"
		if t.Timer != "" {
			label = t.Timer
		}
		return th.Chip(ColAccentHi, label, Spinner(ColAccentHi, 10, phase))
	case ThreadError:
		return th.Chip(ColRed, "Failed", nil)
	}
	if t.Unread {
		return th.Chip(ColGreen, "Done", nil)
	}
	return nil
}

func (th *Theme) threadRow(t ThreadRow, phase float64) W {
	return func(gtx C) D {
		return FixedH(gtx, 46, func(gtx C) D {
			size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
			if t.Selected {
				r := image.Rect(dp(gtx, 6), dp(gtx, 1), size.X-dp(gtx, 6), size.Y-dp(gtx, 1))
				fill(gtx, ColSelected, r, dp(gtx, 4))
				fill(gtx, ColAccent, image.Rect(r.Min.X, dp(gtx, 9), r.Min.X+dp(gtx, 2), size.Y-dp(gtx, 9)), dp(gtx, 1))
			}
			gtx.Constraints.Min = image.Point{X: size.X}
			pad(6, 14, 0, 16).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return Col(gtx,
					rigid(func(gtx C) D {
						return FixedH(gtx, 19, func(gtx C) D {
							chip := th.statusChip(t, phase)
							return Row(gtx,
								flexed(func(gtx C) D {
									col, w := ColText, font.Normal
									switch {
									case t.Wakes != "":
										col = ColDim
									case t.Selected:
										w = font.Medium
									case t.Unread || t.State == ThreadApproval || t.State == ThreadQuestion:
										w = font.SemiBold
									}
									return th.Label(gtx, col, w, t.Title)
								}),
								rigid(Spacer(8, 0)),
								rigid(func(gtx C) D {
									if chip != nil {
										return chip(gtx)
									}
									if t.Wakes != "" {
										return th.Small(gtx, ColFaint, "wakes "+t.Wakes)
									}
									return D{}
								}),
							)
						})
					}),
					rigid(Spacer(0, 3)),
					rigid(func(gtx C) D {
						return Row(gtx,
							rigid(func(gtx C) D {
								s := dp(gtx, 6)
								fill(gtx, projectColor(t.Project), image.Rect(0, 0, s, s), dp(gtx, 1.5))
								return D{Size: image.Pt(s, s)}
							}),
							rigid(Spacer(6, 0)),
							rigid(func(gtx C) D { return th.Small(gtx, ColDim, t.Project) }),
							rigid(func(gtx C) D { return th.Small(gtx, ColFaint, "  ·  ") }),
							flexed(func(gtx C) D {
								col, s := ColFaint, t.Branch
								if t.Note != "" {
									s = t.Note
									switch t.State {
									case ThreadApproval:
										col = alpha(ColAmber, 0xd0)
									case ThreadQuestion:
										col = alpha(ColViolet, 0xe0)
									case ThreadError:
										col = alpha(ColRed, 0xc8)
									}
								}
								return th.Small(gtx, col, s)
							}),
							rigid(func(gtx C) D {
								if t.Env == "" {
									return D{}
								}
								return pad(0, 0, 0, 8).Layout(gtx, func(gtx C) D {
									return Row(gtx,
										rigid(IconW(IconMachine, ColFaint, 12)),
										rigid(Spacer(3, 0)),
										rigid(func(gtx C) D { return th.Small(gtx, ColDim, t.Env) }),
									)
								})
							}),
							rigid(func(gtx C) D {
								if t.Age == "" {
									return D{}
								}
								return pad(0, 0, 0, 8).Layout(gtx, func(gtx C) D { return th.Small(gtx, ColFaint, t.Age) })
							}),
						)
					}),
				)
			})
			return D{Size: size}
		})
	}
}

// ---------- thread view ----------

func (th *Theme) threadView(gtx C, s *Scene, compact bool) D {
	return Fill(gtx, ColPanelHi, func(gtx C) D {
		if s.Live {
			th.dragArea(gtx, image.Pt(gtx.Constraints.Max.X, dp(gtx, topBarH)))
		}
		return Col(gtx,
			rigid(func(gtx C) D {
				if compact {
					return th.compactHeader(gtx, s)
				}
				if s.Live {
					// One thread at a time, like T3: the sidebar is the switcher.
					return D{}
				}
				return th.tabs(gtx, s)
			}),
			rigid(func(gtx C) D {
				if compact || (s.Live && s.Title == "") {
					// Nothing open: no title, so no header either.
					return D{}
				}
				return th.threadHeader(gtx, s)
			}),
			flexed(func(gtx C) D {
				if s.Live && s.Hero != "" && len(s.Items) == 0 {
					return th.hero(gtx, s, compact)
				}
				return th.messages(gtx, s, compact)
			}),
			rigid(func(gtx C) D {
				if s.Live && s.Hero != "" && len(s.Items) == 0 {
					return D{}
				}
				return th.composerArea(gtx, s, compact)
			}),
			rigid(func(gtx C) D {
				if s.Term == nil {
					return D{}
				}
				return th.terminalDrawer(gtx, s)
			}),
		)
	})
}

// hero is a new thread the way T3 opens one: "What should we build in
// tern?" with the composer right under it, both a little above the middle.
// The project name opens the project menu under it, kept on screen.
func (th *Theme) hero(gtx C, s *Scene, compact bool) D {
	size := gtx.Constraints.Max
	big := Txt{Size: 26, Color: ColText}
	bold := big
	bold.Font = UI(font.Medium)
	lg := gtx
	lg.Constraints.Min = image.Point{}

	// The heading, in three parts so the name can be a button.
	var nameAt image.Point
	var nameSize image.Point
	hm := op.Record(gtx.Ops)
	x := 0
	pre := th.Text(lg, big, "What should we build in ")
	x += pre.Size.X
	off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	name := th.Click(liveID(s, "project.hero"), func(gtx C) D {
		d := th.Text(gtx, bold, s.Hero)
		// A dotted underline says it can be changed.
		y, r, step := d.Size.Y-dp(gtx, 1), max(1, dp(gtx, 1)), dp(gtx, 4)
		for x := r; x < d.Size.X; x += step {
			circle(gtx, ColDim, image.Pt(x, y), r)
		}
		return D{Size: image.Pt(d.Size.X, d.Size.Y+dp(gtx, 3))}
	})(lg)
	off.Pop()
	nameAt, nameSize = image.Pt(x, 0), name.Size
	x += name.Size.X
	off = op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	q := th.Text(lg, big, "?")
	off.Pop()
	heading := hm.Stop()
	headW, headH := x+q.Size.X, max(pre.Size.Y, name.Size.Y)

	cm := op.Record(gtx.Ops)
	cg := gtx
	cg.Constraints.Min = image.Point{}
	cd := th.composerArea(cg, s, compact)
	composer := cm.Stop()

	gap := dp(gtx, 22)
	total := headH + gap + cd.Size.Y
	top := max(0, (size.Y-total)*10/24)
	hx := max(0, (size.X-headW)/2)

	t := op.Offset(image.Pt(hx, top)).Push(gtx.Ops)
	heading.Add(gtx.Ops)
	t.Pop()
	t = op.Offset(image.Pt(0, top+headH+gap)).Push(gtx.Ops)
	composer.Add(gtx.Ops)
	t.Pop()

	if s.Projects != nil && s.ProjectsHero {
		mm := op.Record(gtx.Ops)
		md := th.traitsMenu(lg, s, "project", s.Projects)
		menu := mm.Stop()
		at := image.Pt(hx+nameAt.X, top+nameSize.Y+dp(gtx, 6))
		at.X = max(0, min(at.X, size.X-md.Size.X))
		at.Y = max(0, min(at.Y, size.Y-md.Size.Y-dp(gtx, 8)))
		dm := op.Record(gtx.Ops)
		op.Offset(at).Add(gtx.Ops)
		menu.Add(gtx.Ops)
		op.Defer(gtx.Ops, dm.Stop())
	}
	return D{Size: size}
}

func (th *Theme) tabs(gtx C, s *Scene) D {
	return FixedH(gtx, 32, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		fill(gtx, ColPanel, image.Rectangle{Max: size}, 0)
		hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
		var kids []layout.FlexChild
		kids = append(kids, rigid(Spacer(6, 0)))
		for i, name := range s.Tabs {
			i, name := i, name
			id := ""
			if i < len(s.TabIDs) {
				id = s.TabIDs[i]
			}
			kids = append(kids, rigid(th.Click(prefixID("tab:", id), func(gtx C) D {
				return th.tab(gtx, name, i == s.TabSel, i == s.TabSel && s.State == ThreadRunning, s.Phase, prefixID("tabclose:", id))
			})))
		}
		kids = append(kids, rigid(th.Click(liveID(s, "new"), func(gtx C) D {
			return padXY(6, 6).Layout(gtx, IconW(IconPlus, ColFaint, 14))
		})))
		gtx.Constraints.Min.Y = size.Y
		layout.Flex{Alignment: layout.End}.Layout(gtx, kids...)
		return D{Size: size}
	})
}

func (th *Theme) tab(gtx C, name string, active, running bool, phase float64, closeID string) D {
	return FixedW(gtx, 176, func(gtx C) D {
		return FixedH(gtx, 27, func(gtx C) D {
			size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
			r := image.Rect(0, 0, size.X, size.Y+dp(gtx, 4))
			if active {
				fill(gtx, ColPanelHi, r, dp(gtx, 5))
				strokeRect(gtx, ColBorder, r, dp(gtx, 5), float32(max(1, dp(gtx, 1))))
				fill(gtx, ColAccent, image.Rect(dp(gtx, 1), 0, size.X-dp(gtx, 1), dp(gtx, 2)), dp(gtx, 1))
				// Merge into the panel below.
				fill(gtx, ColPanelHi, image.Rect(dp(gtx, 1), size.Y-dp(gtx, 1), size.X-dp(gtx, 1), size.Y+dp(gtx, 1)), 0)
			}
			gtx.Constraints.Min = size
			pad(0, 8, 0, 10).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				col := ColDim
				if active {
					col = ColText
				}
				return Row(gtx,
					rigid(func(gtx C) D {
						if !running {
							return D{}
						}
						return pad(0, 6, 0, 0).Layout(gtx, Spinner(ColAccentHi, 11, phase))
					}),
					flexed(func(gtx C) D { return th.Label(gtx, col, font.Medium, name) }),
					rigid(func(gtx C) D {
						if !active {
							return D{}
						}
						return th.Click(closeID, IconW(IconClose, ColFaint, 13))(gtx)
					}),
				)
			})
			return D{Size: size}
		})
	})
}

func stateChip(th *Theme, s *Scene) W {
	switch s.State {
	case ThreadRunning:
		return th.Chip(ColAccentHi, s.StateLabel, Spinner(ColAccentHi, 10, s.Phase))
	case ThreadApproval, ThreadQuestion:
		return th.Chip(ColAmber, s.StateLabel, Dot(ColAmber, false))
	case ThreadError:
		return th.Chip(ColRed, s.StateLabel, nil)
	}
	return th.Chip(ColGreen, s.StateLabel, IconW(IconCheck, ColGreen, 10))
}

func (th *Theme) threadHeader(gtx C, s *Scene) D {
	return FixedH(gtx, topBarH, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Row(gtx,
			rigid(func(gtx C) D {
				// Long titles truncate so the buttons keep their size.
				gtx.Constraints.Max.X = gtx.Constraints.Max.X / 2
				return th.Text(gtx, Txt{Size: SizeTitle, Font: UI(font.SemiBold), MaxLines: 1}, s.Title)
			}),
			rigid(Spacer(10, 0)),
			rigid(stateChip(th, s)),
			rigid(Spacer(10, 0)),
			flexed(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				if !s.PickProject {
					return th.Small(gtx, ColFaint, s.Project)
				}
				// A draft's project is a dropdown: the menu drops below it.
				return Row(gtx,
					rigid(func(gtx C) D { return th.Small(gtx, ColFaint, "in") }),
					rigid(Spacer(4, 0)),
					rigid(func(gtx C) D {
						dims := th.Click("project", th.footPick(IconW(IconFolder, ColDim, 14), s.Project, s.Projects != nil && !s.ProjectsHero))(gtx)
						if s.Projects != nil && !s.ProjectsHero {
							th.deferBelow(gtx, image.Pt(0, dims.Size.Y+dp(gtx, 4)), func(gtx C) D { return th.traitsMenu(gtx, s, "project", s.Projects) })
						}
						return dims
					}),
				)
			}),
			rigid(Spacer(8, 0)),
			rigid(func(gtx C) D {
				if s.State != ThreadRunning && s.State != ThreadApproval && s.State != ThreadQuestion {
					return D{}
				}
				return th.Click(liveID(s, "stop"), th.Button(Danger, IconStop, true, "Stop"))(gtx)
			}),
		)
	})
}

func (th *Theme) compactHeader(gtx C, s *Scene) D {
	return FixedH(gtx, 52, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		fill(gtx, ColPanel, image.Rectangle{Max: size}, 0)
		hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
		gtx.Constraints.Min.X = size.X
		return pad(0, 10, 0, 8).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				rigid(func(gtx C) D {
					t := op.Affine(f32.Affine2D{}.Rotate(f32.Pt(float32(dp(gtx, 9)), float32(dp(gtx, 9))), 3.14159)).Push(gtx.Ops)
					d := DrawIcon(gtx, IconChevronRight, ColText, 18)
					t.Pop()
					return d
				}),
				rigid(Spacer(6, 0)),
				flexed(func(gtx C) D {
					return Col(gtx,
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: 14, Font: UI(font.SemiBold), MaxLines: 1}, s.Title)
						}),
						rigid(Spacer(0, 2)),
						rigid(func(gtx C) D {
							glyph, col := Spinner(ColAccentHi, 10, s.Phase), ColAccentHi
							if s.State == ThreadApproval || s.State == ThreadQuestion {
								glyph, col = Dot(ColAmber, false), ColAmber
							}
							return Row(gtx,
								rigid(glyph),
								rigid(Spacer(5, 0)),
								rigid(func(gtx C) D { return th.Small(gtx, col, s.StateLabel) }),
								rigid(func(gtx C) D { return th.Small(gtx, ColFaint, "  ·  "+s.Project) }),
							)
						}),
					)
				}),
				rigid(th.Button(Danger, IconStop, true, "")),
				rigid(Spacer(6, 0)),
				rigid(th.Button(Ghost, IconDots, true, "")),
			)
		})
	})
}

func (th *Theme) messages(gtx C, s *Scene, compact bool) D {
	size := gtx.Constraints.Max
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if s.Empty != "" && len(s.Items) == 0 {
		gtx.Constraints.Min = size
		layout.Center.Layout(gtx, func(gtx C) D { return th.Label(gtx, ColFaint, font.Normal, s.Empty) })
		return D{Size: size}
	}
	// The shell gives the timeline its lane; the phone gives it margins.
	w := size.X
	if compact {
		w = size.X - 2*dp(gtx, 14)
	}
	x := (size.X - w) / 2

	var rows []W
	rows = append(rows, Spacer(0, 18))
	if s.MoreAbove {
		rows = append(rows, func(gtx C) D {
			return pad(0, 0, 10, 0).Layout(gtx, func(gtx C) D {
				return layout.Center.Layout(gtx, th.Click(liveID(s, "more"), th.Button(Ghost, IconChevronDown, false, "Load earlier")))
			})
		})
	}
	for i, it := range s.Items {
		it, gap := it, float32(0)
		if i > 0 {
			gap = itemGap(s.Items[i-1], it)
		}
		rows = append(rows, func(gtx C) D {
			return pad(gap, 0, 0, 0).Layout(gtx, func(gtx C) D { return th.item(gtx, it, s.Phase) })
		})
	}
	rows = append(rows, Spacer(0, 18))
	// Pinned to the bottom like a chat: the newest row sits above the composer.
	l := th.List("messages", true)
	gtx.Constraints = layout.Exact(size)
	gtx.Constraints.Min.X = 0
	defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
	gtx.Constraints.Max.X = w
	m := op.Record(gtx.Ops)
	gtx.Constraints.Min = image.Point{X: w}
	l.Layout(gtx, len(rows), func(gtx C, i int) D { return rows[i](gtx) })
	m.Stop().Add(gtx.Ops)
	p := l.Position
	if p.Length > size.Y {
		before := p.Offset + p.First*p.Length/max(1, len(rows))
		frac := float32(size.Y) / float32(p.Length)
		start := float32(before) / float32(p.Length)
		if !p.BeforeEnd {
			start = 1 - frac
		}
		op.Offset(image.Pt(-x, 0)).Add(gtx.Ops)
		Scrollbar(gtx, size, min(start, 1-frac), frac)
	}
	return D{Size: size}
}

func itemGap(prev, next Item) float32 {
	tight := func(it Item) bool { return it.Kind == ItemTool || it.Kind == ItemThought || it.Kind == ItemThinking }
	if tight(prev) && tight(next) {
		return 2
	}
	return 16
}

func (th *Theme) item(gtx C, it Item, phase float64) D {
	switch it.Kind {
	case ItemUser:
		return layout.E.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Max.X = gtx.Constraints.Max.X * 82 / 100
			return Box{Bg: ColFrameHover, Border: ColBorderHi, Radius: 10, Pad: padXY(14, 10)}.Layout(gtx, func(gtx C) D {
				return Col(gtx,
					rigid(func(gtx C) D {
						if len(it.Thumbs) == 0 {
							return D{}
						}
						return pad(0, 0, 8, 0).Layout(gtx, func(gtx C) D {
							var kids []layout.FlexChild
							for i, t := range it.Thumbs {
								kids = append(kids, rigid(th.pasted(t, i, false)), rigid(Spacer(6, 0)))
							}
							return Row(gtx, kids...)
						})
					}),
					rigid(func(gtx C) D {
						if HasChips(it.Text) {
							return th.chipText(gtx, it.ID, it.Text, it.Chips)
						}
						return th.Text(gtx, Txt{Size: SizeBody, LineH: 1.42}, it.Text)
					}),
					rigid(func(gtx C) D {
						if it.Attached == 0 || len(it.Thumbs) > 0 {
							return D{}
						}
						return pad(8, 0, 0, 0).Layout(gtx, func(gtx C) D {
							return Row(gtx,
								rigid(IconW(IconImage, ColDim, 13)),
								rigid(Spacer(5, 0)),
								rigid(func(gtx C) D { return th.Small(gtx, ColDim, plural(it.Attached, "image")) }),
							)
						})
					}),
				)
			})
		})
	case ItemAssistant:
		return th.Markdown(gtx, it.Text)
	case ItemFold:
		return th.Click(prefixID("fold:", it.ID), func(gtx C) D { return th.foldRow(gtx, it) })(gtx)
	case ItemTool:
		return th.toolRow(gtx, it, phase)
	case ItemThought, ItemThinking:
		return th.Click(prefixID("thought:", it.ID), func(gtx C) D { return th.thought(gtx, it, phase) })(gtx)
	case ItemTodo:
		return Box{Bg: ColPanel, Border: ColBorder, Radius: 6, Pad: padXY(12, 10)}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			kids := []layout.FlexChild{rigid(func(gtx C) D {
				done := 0
				for _, t := range it.Todos {
					if t.Done {
						done++
					}
				}
				return pad(0, 0, 6, 0).Layout(gtx, func(gtx C) D {
					return th.Small(gtx, ColFaint, fmt.Sprintf("PLAN  ·  %d of %d done", done, len(it.Todos)))
				})
			})}
			for _, t := range it.Todos {
				t := t
				kids = append(kids, rigid(func(gtx C) D {
					return FixedH(gtx, 22, func(gtx C) D {
						col := ColDim
						var mark W = func(gtx C) D {
							s := dp(gtx, 12)
							strokeRect(gtx, ColBorderHi, image.Rect(0, 0, s, s), dp(gtx, 3), float32(max(1, dp(gtx, 1))))
							return D{Size: image.Pt(s, s)}
						}
						switch {
						case t.Done:
							col = ColFaint
							mark = IconW(IconCheck, ColGreen, 12)
						case t.Now:
							col = ColText
							mark = Spinner(ColAccentHi, 12, phase)
						}
						return Row(gtx,
							rigid(mark),
							rigid(Spacer(8, 0)),
							flexed(func(gtx C) D { return th.Label(gtx, col, font.Normal, t.Text) }),
						)
					})
				}))
			}
			return Col(gtx, kids...)
		})
	case ItemNotice:
		return layout.Center.Layout(gtx, func(gtx C) D { return th.Small(gtx, ColFaint, it.Text) })
	case ItemError:
		return Box{Bg: alpha(ColRed, 0x14), Border: alpha(ColRed, 0x48), Radius: 6, Pad: padXY(12, 9)}.Layout(gtx, func(gtx C) D {
			return th.Text(gtx, Txt{Color: ColDelFg}, it.Text)
		})
	}
	return D{}
}

// foldRow is "Worked for 2m 14s", the folded work of a finished run.
func (th *Theme) foldRow(gtx C, it Item) D {
	return FixedH(gtx, 24, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		ic := IconChevronRight
		if it.Open {
			ic = IconChevronDown
		}
		return Row(gtx,
			rigid(IconW(ic, ColFaint, 14)),
			rigid(Spacer(4, 0)),
			rigid(func(gtx C) D { return th.Label(gtx, ColDim, font.Medium, it.Text) }),
			rigid(Spacer(10, 0)),
			rigid(func(gtx C) D { return th.Small(gtx, ColFaint, it.Meta) }),
			rigid(Spacer(10, 0)),
			flexed(func(gtx C) D {
				w := gtx.Constraints.Max.X
				hline(gtx, ColBorder, dp(gtx, 1), 0, w)
				return D{Size: image.Pt(w, dp(gtx, 2))}
			}),
		)
	})
}

func toolIcon(t ToolKind) Icon {
	switch t {
	case ToolEdit:
		return IconFile
	case ToolSearch:
		return IconSearch
	case ToolWeb:
		return IconGlobe
	case ToolAgent:
		return IconAgent
	case ToolMCP:
		return IconSpark
	}
	return IconTerminal
}

func (th *Theme) toolRow(gtx C, it Item, phase float64) D {
	return Col(gtx,
		rigid(th.Click(prefixID("tool:", it.ID), func(gtx C) D {
			return FixedH(gtx, 26, func(gtx C) D {
				size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
				if it.Open {
					fill(gtx, ColFrame, image.Rectangle{Max: size}, dp(gtx, 4))
				}
				gtx.Constraints.Min.X = size.X
				return pad(0, 8, 0, 6).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					col := ColDim
					var lead W = IconW(toolIcon(it.Tool), ColFaint, 14)
					switch {
					case it.Live:
						lead, col = Spinner(ColAccentHi, 13, phase), ColText
					case it.Failed:
						lead = IconW(toolIcon(it.Tool), ColRed, 14)
					}
					return Row(gtx,
						rigid(lead),
						rigid(Spacer(8, 0)),
						flexed(func(gtx C) D {
							return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: col, MaxLines: 1}, it.Title)
						}),
						rigid(Spacer(8, 0)),
						rigid(func(gtx C) D { return th.toolMeta(gtx, it) }),
					)
				})
			})
		})),
		rigid(func(gtx C) D {
			if !it.Open || it.Body == "" {
				return D{}
			}
			return pad(3, 0, 6, 28).Layout(gtx, func(gtx C) D { return th.toolBody(gtx, it) })
		}),
	)
}

func (th *Theme) toolMeta(gtx C, it Item) D {
	if it.Tool == ToolEdit && it.Meta != "" {
		var add, del string
		fmt.Sscanf(it.Meta, "%s %s", &add, &del)
		return Row(gtx,
			rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColAddFg, MaxLines: 1}, add)
			}),
			rigid(Spacer(5, 0)),
			rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColDelFg, MaxLines: 1}, del)
			}),
		)
	}
	col := ColFaint
	if it.Failed {
		col = ColRed
	}
	return th.Small(gtx, col, it.Meta)
}

func (th *Theme) toolBody(gtx C, it Item) D {
	if it.Tool != ToolEdit {
		return th.CodeBlock(gtx, it.Body, 12)
	}
	lines := strings.Split(it.Body, "\n")
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: ColWindow, Border: ColBorder, Radius: 5, Pad: padXY(0, 6)}.Layout(gtx, func(gtx C) D {
		var kids []layout.FlexChild
		for _, l := range lines {
			l := l
			kids = append(kids, rigid(func(gtx C) D {
				bg, fg := color.NRGBA{}, rgb(0xc9cfd8)
				switch {
				case strings.HasPrefix(l, "@@"):
					fg = ColHunkFg
				case strings.HasPrefix(l, "+"):
					bg, fg = ColAddBg, ColAddFg
				case strings.HasPrefix(l, "-"):
					bg, fg = ColDelBg, ColDelFg
				}
				return Box{Bg: bg, Pad: padXY(12, 1)}.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: fg, MaxLines: 1}, l)
				})
			}))
		}
		return Col(gtx, kids...)
	})
}

func (th *Theme) thought(gtx C, it Item, phase float64) D {
	return pad(3, 0, 3, 6).Layout(gtx, func(gtx C) D {
		return layout.Flex{}.Layout(gtx,
			rigid(func(gtx C) D {
				return pad(1, 0, 0, 0).Layout(gtx, func(gtx C) D {
					if it.Live {
						return Spinner(ColViolet, 13, phase)(gtx)
					}
					return IconW(IconSpark, alpha(ColViolet, 0xb0), 14)(gtx)
				})
			}),
			rigid(Spacer(8, 0)),
			flexed(func(gtx C) D {
				label := "Thought"
				if it.Live {
					label = "Thinking"
				}
				body := it.Text
				var sp []spanT
				sp = append(sp, spanT{label, ColViolet, font.SemiBold})
				if body != "" {
					sp = append(sp, spanT{"  " + body, ColDim, font.Normal})
				}
				lines := 2
				if it.Open {
					lines = 0
				}
				return th.inline(gtx, sp, lines)
			}),
		)
	})
}

type spanT struct {
	s   string
	col color.NRGBA
	w   font.Weight
}

// inline lays out mixed-style text that wraps, cut to maxLines by height.
func (th *Theme) inline(gtx C, sp []spanT, maxLines int) D {
	var styles []richtext.SpanStyle
	for _, s := range sp {
		styles = append(styles, richtext.SpanStyle{Content: s.s, Color: s.col, Font: UI(s.w), Size: SizeUI})
	}
	m := op.Record(gtx.Ops)
	t := richtext.Text(new(richtext.InteractiveText), th.Shaper, styles...)
	t.LineHeightScale = 1.4
	dims := t.Layout(gtx)
	call := m.Stop()
	if limit := int((float32(maxLines)*1.4 + 0.35) * float32(gtx.Sp(SizeUI))); maxLines > 0 && dims.Size.Y > limit {
		dims.Size.Y = limit
	}
	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return dims
}

// ---------- composer ----------

func (th *Theme) composerArea(gtx C, s *Scene, compact bool) D {
	size := gtx.Constraints.Max
	// The shell gives the composer its lane and the spacer under it; the
	// phone gives it margins.
	w, bottom := size.X, float32(0)
	if compact {
		w, bottom = size.X-2*dp(gtx, 10), 14
	}
	x := (size.X - w) / 2
	gtx.Constraints = layout.Exact(image.Pt(w, 0))
	gtx.Constraints.Max.Y = size.Y
	m := op.Record(gtx.Ops)
	dims := pad(0, 0, bottom, 0).Layout(gtx, func(gtx C) D {
		return Col(gtx,
			rigid(func(gtx C) D {
				if s.Error == "" {
					return D{}
				}
				return pad(0, 0, 8, 0).Layout(gtx, th.Click(liveID(s, "error"), func(gtx C) D { return th.errorBanner(gtx, s.Error) }))
			}),
			rigid(func(gtx C) D {
				if s.Approval == nil {
					return D{}
				}
				return pad(0, 0, 8, 0).Layout(gtx, func(gtx C) D { return th.approval(gtx, s, s.Approval, compact) })
			}),
			rigid(func(gtx C) D {
				if len(s.Questions) == 0 {
					return D{}
				}
				return pad(0, 0, 8, 0).Layout(gtx, func(gtx C) D { return th.questions(gtx, s, s.Questions) })
			}),
			rigid(func(gtx C) D {
				if len(s.Queue) == 0 {
					return D{}
				}
				return pad(0, 0, 8, 0).Layout(gtx, func(gtx C) D { return th.queue(gtx, s, s.Queue) })
			}),
			rigid(func(gtx C) D {
				if s.Notice == "" {
					return D{}
				}
				return pad(0, 4, 6, 4).Layout(gtx, func(gtx C) D { return th.Small(gtx, ColFaint, s.Notice) })
			}),
			rigid(func(gtx C) D { return th.composer(gtx, s, compact) }),
		)
	})
	call := m.Stop()
	t := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	t.Pop()
	return D{Size: image.Pt(size.X, dims.Size.Y)}
}

func (th *Theme) approval(gtx C, s *Scene, a *Approval, compact bool) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: alpha(ColAmber, 0x10), Border: alpha(ColAmber, 0x60), Radius: 8, Pad: padXY(12, 10)}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		buttons := func(gtx C) D {
			if s.Live && !a.YesNo {
				return th.Small(gtx, ColFaint, "Answer this one in T3 for now.")
			}
			return Row(gtx,
				rigid(th.Click(liveID(s, "approval:decline"), th.Button(Ghost, 0, false, "Decline"))),
				rigid(Spacer(6, 0)),
				rigid(th.Click(liveID(s, "approval:acceptForSession"), th.Button(Plain, 0, false, "For session"))),
				rigid(Spacer(6, 0)),
				rigid(th.Click(liveID(s, "approval:accept"), th.Button(Primary, IconCheck, true, "Approve"))),
			)
		}
		body := func(gtx C) D {
			return Col(gtx,
				rigid(func(gtx C) D {
					return Row(gtx,
						rigid(IconW(IconLock, ColAmber, 14)),
						rigid(Spacer(7, 0)),
						rigid(func(gtx C) D { return th.Label(gtx, ColAmber, font.SemiBold, "Approve "+a.Kind) }),
						rigid(func(gtx C) D { return th.Small(gtx, ColFaint, "   in "+a.Cwd) }),
					)
				}),
				rigid(Spacer(0, 7)),
				rigid(func(gtx C) D {
					return Box{Bg: ColWindow, Border: ColBorder, Radius: 4, Pad: padXY(9, 6)}.Layout(gtx, func(gtx C) D {
						return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: ColText, MaxLines: 2}, "$ "+a.Detail)
					})
				}),
			)
		}
		if compact {
			return Col(gtx, rigid(body), rigid(Spacer(0, 10)), rigid(func(gtx C) D { return layout.E.Layout(gtx, buttons) }))
		}
		return layout.Flex{Alignment: layout.End}.Layout(gtx, flexed(body), rigid(Spacer(14, 0)), rigid(buttons))
	})
}

func (th *Theme) questions(gtx C, s *Scene, qs []Question) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: ColPanel, Border: alpha(ColAccent, 0x60), Radius: 8, Pad: padXY(14, 12)}.Layout(gtx, func(gtx C) D {
		var kids []layout.FlexChild
		for i, q := range qs {
			q := q
			if i > 0 {
				kids = append(kids, rigid(Spacer(0, 12)))
			}
			kids = append(kids,
				rigid(func(gtx C) D {
					return Row(gtx,
						rigid(func(gtx C) D { return th.Chip(ColAccentHi, q.Header, nil)(gtx) }),
						rigid(Spacer(8, 0)),
						flexed(func(gtx C) D { return th.Label(gtx, ColText, font.Medium, q.Text) }),
					)
				}),
				rigid(Spacer(0, 8)),
				rigid(func(gtx C) D {
					var opts []layout.FlexChild
					for j, o := range q.Options {
						v := Plain
						if (q.Chosen == nil && j == q.Picked) || (j < len(q.Chosen) && q.Chosen[j]) {
							v = Toggled
						}
						opts = append(opts, rigid(th.Click(liveID(s, fmt.Sprintf("q:%d:%d", i, j)), th.Button(v, 0, false, o))), rigid(Spacer(6, 0)))
					}
					return Row(gtx, opts...)
				}),
			)
		}
		kids = append(kids, rigid(Spacer(0, 12)), rigid(func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				flexed(func(gtx C) D {
					if s.Live {
						return th.Field("answer", 0, false, "Or type your own answer", false)(gtx)
					}
					return th.Input(0, false, "", "Or type your own answer", false)(gtx)
				}),
				rigid(Spacer(8, 0)),
				rigid(th.Click(liveID(s, "dismiss"), th.Button(Ghost, 0, false, "Dismiss"))),
				rigid(Spacer(6, 0)),
				rigid(th.Click(liveID(s, "answer"), th.Button(Primary, 0, false, "Answer"))),
			)
		}))
		return Col(gtx, kids...)
	})
}

func (th *Theme) queue(gtx C, s *Scene, q []string) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: ColPanel, Border: ColBorder, Radius: 8, Pad: padXY(10, 6)}.Layout(gtx, func(gtx C) D {
		var kids []layout.FlexChild
		for i, text := range q {
			text, i := text, i
			kids = append(kids, rigid(func(gtx C) D {
				return FixedH(gtx, 28, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return Row(gtx,
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColFaint}, fmt.Sprintf("%d", i+1))
						}),
						rigid(Spacer(9, 0)),
						rigid(IconW(IconQueue, ColFaint, 13)),
						rigid(Spacer(8, 0)),
						flexed(func(gtx C) D { return th.Label(gtx, ColDim, font.Normal, text) }),
						rigid(th.Click(liveID(s, fmt.Sprintf("queue:%d:steer", i)), th.Button(Ghost, 0, false, "Send now"))),
						rigid(func(gtx C) D {
							if s.Live {
								return D{}
							}
							return th.Button(Ghost, 0, false, "Edit")(gtx)
						}),
						rigid(th.Click(liveID(s, fmt.Sprintf("queue:%d:cancel", i)), th.Button(Ghost, IconClose, true, ""))),
					)
				})
			}))
		}
		return Col(gtx, kids...)
	})
}

func (th *Theme) composer(gtx C, s *Scene, compact bool) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	border := ColBorderHi
	if s.Composer != "" {
		border = alpha(ColAccent, 0x90)
	}
	var modelBtn, effortBtn image.Point
	dims := Box{Bg: ColFrame, Border: border, Radius: 10, Pad: pad(11, 10, 9, 12)}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Col(gtx,
			rigid(func(gtx C) D {
				if s.Images == 0 && len(s.Thumbs) == 0 {
					return D{}
				}
				return pad(0, 0, 10, 0).Layout(gtx, func(gtx C) D {
					var kids []layout.FlexChild
					for i := 0; i < s.Images; i++ {
						kids = append(kids, rigid(thumb(i)), rigid(Spacer(6, 0)))
					}
					for i, t := range s.Thumbs {
						kids = append(kids, rigid(th.pasted(t, i, true)), rigid(Spacer(6, 0)))
					}
					return Row(gtx, kids...)
				})
			}),
			rigid(func(gtx C) D {
				gtx.Constraints.Min.Y = dp(gtx, 38)
				if s.Live {
					ed := th.Editor("composer")
					ed.Submit = true
					gtx.Constraints.Max.Y = dp(gtx, 220)
					d := th.editor(gtx, "composer", Txt{Size: SizeBody, LineH: 1.42}, s.Placeholder)
					d.Size.Y = max(d.Size.Y, gtx.Constraints.Min.Y)
					return d
				}
				if s.Composer == "" {
					return th.Text(gtx, Txt{Size: SizeBody, Color: ColFaint}, s.Placeholder)
				}
				lines := strings.Split(s.Composer, "\n")
				var kids []layout.FlexChild
				for i, l := range lines {
					l, last := l, i == len(lines)-1
					kids = append(kids, rigid(func(gtx C) D {
						return Row(gtx,
							rigid(func(gtx C) D { return th.Text(gtx, Txt{Size: SizeBody, LineH: 1.42}, l) }),
							rigid(func(gtx C) D {
								if !last {
									return D{}
								}
								return Caret(gtx, 17)
							}),
						)
					}))
				}
				d := Col(gtx, kids...)
				d.Size.Y = max(d.Size.Y, gtx.Constraints.Min.Y)
				return d
			}),
			rigid(Spacer(0, 8)),
			rigid(func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				var kids []layout.FlexChild
				kids = append(kids,
					rigid(func(gtx C) D {
						if s.Live {
							return D{Size: image.Pt(0, dp(gtx, 26))}
						}
						return pad(0, 4, 0, 0).Layout(gtx, th.Button(Ghost, IconPlus, true, ""))
					}),
					rigid(func(gtx C) D {
						d := th.Click(liveID(s, "model"), th.footPick(ProviderLogo(s.Provider, 14), s.Model, s.Picker != nil))(gtx)
						modelBtn = d.Size
						return d
					}),
				)
				if s.Effort != "" || !s.Live {
					kids = append(kids, rigid(footSep), rigid(func(gtx C) D {
						d := th.Click(liveID(s, "traits"), th.footPick(IconW(IconGauge, ColDim, 14), s.Effort, s.Traits != nil))(gtx)
						effortBtn = d.Size
						return d
					}))
				}
				if !compact {
					// Access is an icon; the label lives in the menu.
					kids = append(kids, rigid(footSep), rigid(th.Click(liveID(s, "access"), th.footPick(IconW(IconLock, ColDim, 14), "", s.Access != nil))))
				}
				kids = append(kids, flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }))
				if !compact {
					kids = append(kids,
						rigid(func(gtx C) D {
							return pad(0, 10, 0, 0).Layout(gtx, func(gtx C) D {
								return Row(gtx,
									rigid(th.Kbd("Enter")),
									rigid(func(gtx C) D { return th.Small(gtx, ColFaint, " send   ") }),
									rigid(th.Kbd("Alt Enter")),
									rigid(func(gtx C) D { return th.Small(gtx, ColFaint, " queue") }),
								)
							})
						}),
					)
				}
				v := Primary
				if (s.Live && th.EditorText("composer") == "") || (!s.Live && s.Composer == "") {
					v = Plain
				}
				kids = append(kids, rigid(th.Click(liveID(s, "send"), th.Button(v, IconSend, true, ""))))
				return Row(gtx, kids...)
			}),
		)
	})
	if s.Suggest != nil {
		th.deferAbove(gtx, image.Pt(0, 0), func(gtx C) D { return th.suggest(gtx, s) })
	}
	if s.Picker != nil {
		// The combo sits at the toolbar's left, after the + button.
		x := dp(gtx, 12+30)
		if s.Live {
			x = dp(gtx, 12)
		}
		at := image.Pt(x, dims.Size.Y-dp(gtx, 9)-modelBtn.Y)
		th.deferAbove(gtx, at, func(gtx C) D { return th.modelPicker(gtx, s, s.Picker) })
	}
	if s.Traits != nil {
		// The options menu opens over the effort button, right of the model.
		x := dp(gtx, 12+30+9) + modelBtn.X
		if s.Live {
			x = dp(gtx, 12+9) + modelBtn.X
		}
		at := image.Pt(x, dims.Size.Y-dp(gtx, 9)-modelBtn.Y)
		th.deferAbove(gtx, at, func(gtx C) D { return th.traitsMenu(gtx, s, "trait", s.Traits) })
	}
	if s.Access != nil {
		x := dp(gtx, 12+9+9) + modelBtn.X + effortBtn.X
		at := image.Pt(x, dims.Size.Y-dp(gtx, 9)-modelBtn.Y)
		th.deferAbove(gtx, at, func(gtx C) D { return th.traitsMenu(gtx, s, "access", s.Access) })
	}
	return dims
}

// errorBanner is a failed action, red and dismissable.
func (th *Theme) errorBanner(gtx C, msg string) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: alpha(ColRed, 0x14), Border: alpha(ColRed, 0x48), Radius: 6, Pad: padXY(12, 8)}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Row(gtx,
			flexed(func(gtx C) D { return th.Text(gtx, Txt{Color: ColDelFg, MaxLines: 3}, msg) }),
			rigid(Spacer(8, 0)),
			rigid(IconW(IconClose, ColDelFg, 13)),
		)
	})
}

// footPick is a borderless dropdown in the composer footer: icon, value,
// chevron, highlighted while its menu is open.
func (th *Theme) footPick(lead W, label string, open bool) W {
	return func(gtx C) D {
		bg := color.NRGBA{}
		if open {
			bg = ColFrameActive
		}
		return Box{Bg: bg, Radius: 5, Pad: padXY(6, 0)}.Layout(gtx, func(gtx C) D {
			return FixedH(gtx, 26, func(gtx C) D {
				kids := []layout.FlexChild{rigid(lead)}
				if label != "" {
					kids = append(kids, rigid(Spacer(6, 0)), rigid(func(gtx C) D { return th.Label(gtx, ColDim, font.Medium, label) }))
				}
				kids = append(kids, rigid(Spacer(3, 0)), rigid(IconW(IconChevronDown, ColFaint, 12)))
				return Row(gtx, kids...)
			})
		})
	}
}

// footSep is the thin rule between footer dropdowns.
func footSep(gtx C) D {
	h := dp(gtx, 14)
	w := dp(gtx, 9)
	vline(gtx, ColBorderHi, w/2, 0, h)
	return D{Size: image.Pt(w, h)}
}

func thumb(i int) W {
	return func(gtx C) D {
		s := dp(gtx, 52)
		r := image.Rect(0, 0, s, s)
		cols := []color.NRGBA{rgb(0x24425f), rgb(0x3b2b52)}
		fill(gtx, cols[i%2], r, dp(gtx, 6))
		// A fake screenshot: title bar and two blocks.
		fill(gtx, alpha(rgb(0xffffff), 0x22), image.Rect(dp(gtx, 6), dp(gtx, 6), s-dp(gtx, 6), dp(gtx, 11)), dp(gtx, 2))
		fill(gtx, alpha(rgb(0xffffff), 0x16), image.Rect(dp(gtx, 6), dp(gtx, 15), dp(gtx, 24), s-dp(gtx, 6)), dp(gtx, 2))
		fill(gtx, alpha(rgb(0xffffff), 0x10), image.Rect(dp(gtx, 28), dp(gtx, 15), s-dp(gtx, 6), s-dp(gtx, 6)), dp(gtx, 2))
		strokeRect(gtx, ColBorderHi, r, dp(gtx, 6), float32(max(1, dp(gtx, 1))))
		return D{Size: r.Max}
	}
}

// deferAbove draws w on top of everything, its bottom-left corner at at
// (relative to the current origin), the way ImGui popups float.
func (th *Theme) deferAbove(gtx C, at image.Point, w W) {
	m := op.Record(gtx.Ops)
	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max = image.Pt(dp(gtx, 640), dp(gtx, 640))
	dims := w(gtx)
	call := m.Stop()
	m2 := op.Record(gtx.Ops)
	op.Offset(image.Pt(at.X, at.Y-dims.Size.Y-dp(gtx, 6))).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
}

// contextMenu draws the right-click menu at the click, flipped up or left
// when it would run off the window.
func (th *Theme) contextMenu(gtx C, win image.Point, s *Scene) {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: win}
	dims := th.traitsMenu(g, s, "ctx", s.Context)
	call := m.Stop()
	at := th.in.ctxAt
	if at.X+dims.Size.X > win.X {
		at.X = max(0, at.X-dims.Size.X)
	}
	if at.Y+dims.Size.Y > win.Y {
		at.Y = max(0, at.Y-dims.Size.Y)
	}
	m2 := op.Record(gtx.Ops)
	op.Offset(at).Add(gtx.Ops)
	call.Add(gtx.Ops)
	op.Defer(gtx.Ops, m2.Stop())
}

// deferBelow is deferAbove for menus that drop down: w's top-left corner
// sits at at.
func (th *Theme) deferBelow(gtx C, at image.Point, w W) {
	m := op.Record(gtx.Ops)
	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max = image.Pt(dp(gtx, 640), dp(gtx, 640))
	op.Offset(at).Add(gtx.Ops)
	w(gtx)
	op.Defer(gtx.Ops, m.Stop())
}

// popupFrame is T3's menu and popover surface (docs/ui/02-primitives.md
// §3.5, §3.6): popover with a 1dp border at foreground 10%, radius 8, the
// popup elevation and the lip. It's the solid fallback T3 itself paints
// without backdrop-filter; Gio has no backdrop blur.
func popupFrame(gtx C, w W) D {
	m := op.Record(gtx.Ops)
	dims := w(gtx)
	call := m.Stop()
	r := image.Rectangle{Max: dims.Size}
	rad := dp(gtx, 8)
	elev := boxShadow{0, 16, 40, -18, alpha(rgb(0), 0x8c)}
	border, lip := alpha(rgb(0x14141d), 0x1a), alpha(rgb(0), 0x0a)
	if dark {
		elev = boxShadow{0, 18, 44, -18, alpha(rgb(0), 0xcc)}
		border, lip = alpha(rgb(0xf5f5f5), 0x1a), alpha(rgb(0xffffff), 0x0f)
	}
	shadow(gtx, r, rad, []boxShadow{elev})
	fill(gtx, P.Popover, r, rad)
	strokeRect(gtx, over(border, P.Popover), r, rad, float32(max(1, dp(gtx, 1))))
	px := max(1, dp(gtx, 1))
	if dark {
		hline(gtx, over(lip, P.Popover), px, rad, r.Max.X-rad)
	} else {
		hline(gtx, over(lip, P.Popover), r.Max.Y-2*px, rad, r.Max.X-rad)
	}
	call.Add(gtx.Ops)
	return dims
}

// ModelPicker draws the model menu on its own.
func (th *Theme) ModelPicker(gtx C, p *ModelPicker) D { return th.modelPicker(gtx, &Scene{}, p) }

// TraitsMenu draws the model options menu on its own, for the animation render.
func (th *Theme) TraitsMenu(gtx C, t *Traits) D { return th.traitsMenu(gtx, &Scene{}, "trait", t) }

// modelPicker is T3's model menu: search on top, a rail of Favorites and
// providers on the left, that provider's models on the right. Options live
// in the traits menu, not here.
func (th *Theme) modelPicker(gtx C, s *Scene, p *ModelPicker) D {
	return popupFrame(gtx, func(gtx C) D {
		return FixedW(gtx, 420, func(gtx C) D {
			list := func(gtx C) D {
				return pad(6, 6, 6, 6).Layout(gtx, func(gtx C) D {
					tab := p.Tabs[p.Tab]
					name := tab.Provider
					if name == "" {
						name = "Favorites"
					}
					kids := []layout.FlexChild{rigid(func(gtx C) D {
						return pad(4, 8, 6, 8).Layout(gtx, func(gtx C) D {
							return th.Text(gtx, Txt{Size: 10.5, Color: ColFaint, Font: UI(font.SemiBold), MaxLines: 1}, strings.ToUpper(name))
						})
					})}
					for _, m := range p.Models {
						m := m
						kids = append(kids, rigid(th.Click(prefixID("picker.model:", m.ID), func(gtx C) D { return th.pickerRow(gtx, m) })))
					}
					if len(p.Models) == 0 {
						kids = append(kids, rigid(func(gtx C) D {
							return pad(6, 8, 8, 8).Layout(gtx, func(gtx C) D { return th.Small(gtx, ColFaint, "No models match.") })
						}))
					}
					if p.Legacy > 0 {
						kids = append(kids, rigid(func(gtx C) D {
							return FixedH(gtx, 28, func(gtx C) D {
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return pad(0, 8, 0, 6).Layout(gtx, func(gtx C) D {
									return Row(gtx,
										rigid(IconW(IconChevronRight, ColFaint, 13)),
										rigid(Spacer(4, 0)),
										rigid(func(gtx C) D { return th.Label(gtx, ColDim, font.Normal, "Legacy models") }),
										rigid(Spacer(6, 0)),
										rigid(func(gtx C) D { return th.Small(gtx, ColFaint, fmt.Sprint(p.Legacy)) }),
									)
								})
							})
						}))
					}
					return Col(gtx, kids...)
				})
			}
			body := func(gtx C) D {
				m := op.Record(gtx.Ops)
				lgtx := gtx
				lgtx.Constraints.Max.X -= dp(gtx, 45)
				h := list(lgtx).Size.Y
				m.Stop()
				gtx.Constraints.Min.Y = h
				return layout.Flex{}.Layout(gtx,
					rigid(func(gtx C) D { return th.pickerRail(gtx, s, p) }),
					rigid(divider),
					flexed(list),
				)
			}
			return Col(gtx,
				rigid(func(gtx C) D {
					if s.Live {
						return pad(8, 8, 8, 8).Layout(gtx, th.Field("picker.search", IconSearch, true, "Search models", false))
					}
					return pad(8, 8, 8, 8).Layout(gtx, th.Input(IconSearch, true, p.Query, "Search models", true))
				}),
				rigid(func(gtx C) D {
					hline(gtx, ColBorder, 0, 0, gtx.Constraints.Max.X)
					return D{Size: image.Pt(gtx.Constraints.Max.X, max(1, dp(gtx, 1)))}
				}),
				rigid(body),
			)
		})
	})
}

// pickerRail is the column of tabs: a star for Favorites, then each ready
// provider's logo.
func (th *Theme) pickerRail(gtx C, s *Scene, p *ModelPicker) D {
	return FixedW(gtx, 44, func(gtx C) D {
		var kids []layout.FlexChild
		for i, t := range p.Tabs {
			t, on := t, i == p.Tab
			kids = append(kids, rigid(th.Click(liveID(s, fmt.Sprintf("picker.tab:%d", i)), func(gtx C) D {
				sz := dp(gtx, 32)
				x := (gtx.Constraints.Max.X - sz) / 2
				defer op.Offset(image.Pt(x, dp(gtx, 6))).Push(gtx.Ops).Pop()
				if on {
					fill(gtx, ColFrameActive, image.Rect(0, 0, sz, sz), dp(gtx, 6))
				}
				defer op.Offset(image.Pt((sz-dp(gtx, 16))/2, (sz-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
				if t.Provider == "" {
					col := ColFaint
					if on {
						col = ColAmber
					}
					DrawIcon(gtx, IconStar, col, 16)
				} else {
					ProviderLogo(t.Provider, 16)(gtx)
				}
				return D{Size: image.Pt(gtx.Constraints.Max.X, sz+dp(gtx, 2))}
			})))
		}
		return pad(0, 0, 6, 0).Layout(gtx, func(gtx C) D { return Col(gtx, kids...) })
	})
}

// pickerRow is one model: name, a faint note like "1M", the favourite star
// and a check on the current one.
func (th *Theme) pickerRow(gtx C, m PickerModel) D {
	return FixedH(gtx, 28, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		if m.Selected {
			fill(gtx, ColSelected, image.Rectangle{Max: size}, dp(gtx, 4))
		}
		gtx.Constraints.Min.X = size.X
		return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				rigid(func(gtx C) D { return th.Label(gtx, ColText, font.Normal, m.Name) }),
				rigid(func(gtx C) D {
					if m.Note == "" {
						return D{}
					}
					return pad(0, 0, 0, 7).Layout(gtx, func(gtx C) D { return th.Small(gtx, ColFaint, m.Note) })
				}),
				flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
				rigid(func(gtx C) D {
					if !m.Selected {
						return D{}
					}
					return pad(0, 8, 0, 0).Layout(gtx, IconW(IconCheck, ColAccentHi, 13))
				}),
				rigid(func(gtx C) D {
					if m.Fav {
						return DrawIcon(gtx, IconStarFill, ColAmber, 13)
					}
					return DrawIcon(gtx, IconStar, alpha(ColFaint, 0x90), 13)
				}),
			)
		})
	})
}

// traitsMenu is T3's model options menu, built from the model's option
// descriptors: a select is a list of choices, a boolean is a switch.
func (th *Theme) traitsMenu(gtx C, s *Scene, prefix string, t *Traits) D {
	return popupFrame(gtx, func(gtx C) D {
		return FixedW(gtx, 236, func(gtx C) D {
			return pad(4, 4, 4, 4).Layout(gtx, func(gtx C) D {
				var kids []layout.FlexChild
				for i, g := range t.Groups {
					g, i := g, i
					if i > 0 {
						// T3's menu separator: 1dp of border, 4dp above and below.
						kids = append(kids, rigid(func(gtx C) D {
							hline(gtx, over(P.Border, P.Popover), dp(gtx, 4), dp(gtx, 4), gtx.Constraints.Max.X-dp(gtx, 4))
							return D{Size: image.Pt(gtx.Constraints.Max.X, dp(gtx, 9))}
						}))
					}
					if g.Bool {
						kids = append(kids, rigid(th.menuItem(liveID(s, fmt.Sprintf("%s:%d", prefix, i)), func(gtx C) D { return th.traitSwitch(gtx, g) })))
						continue
					}
					if g.Label != "" {
						// The group label: 28 tall, 12/16 medium, muted.
						kids = append(kids, rigid(func(gtx C) D {
							return FixedH(gtx, 28, func(gtx C) D {
								return pad(6, 8, 6, 8).Layout(gtx, func(gtx C) D {
									return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, Font: UI(font.Medium), MaxLines: 1}, g.Label)
								})
							})
						}))
					}
					for j, c := range g.Choices {
						c, on := c, j == g.Sel
						id := liveID(s, fmt.Sprintf("%s:%d:%d", prefix, i, j))
						if c.Off {
							id = ""
						}
						kids = append(kids, rigid(th.menuItem(id, func(gtx C) D { return th.traitChoice(gtx, c, on) })))
					}
				}
				return Col(gtx, kids...)
			})
		})
	})
}

// traitChoice is a menu row, T3's radio item: 14/20 text padded 4 by 8,
// at least 28 tall; checked rows get foreground at 8% and a 14dp check at
// the end. tern's rows can carry a description, under the label in muted
// 12/16.
func (th *Theme) traitChoice(gtx C, c TraitChoice, on bool) D {
	m := op.Record(gtx.Ops)
	d := func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		gtx.Constraints.Min.Y = dp(gtx, 28)
		return pad(4, 8, 4, 8).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				flexed(func(gtx C) D {
					return Col(gtx,
						rigid(func(gtx C) D {
							col := P.PopoverForeground
							if c.Danger {
								col = P.DestructiveForeground
							}
							return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: col, MaxLines: 1}, c.Label)
						}),
						rigid(func(gtx C) D {
							if c.Desc == "" {
								return D{}
							}
							return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}, c.Desc)
						}),
					)
				}),
				rigid(func(gtx C) D {
					if !on {
						return D{}
					}
					return pad(3, 0, 0, 8).Layout(gtx, LucideW("check", P.PopoverForeground, 14))
				}),
			)
		})
	}(gtx)
	call := m.Stop()
	if on && !th.in.menuHover {
		fill(gtx, over(alpha(P.Foreground, 0x14), P.Popover), image.Rectangle{Max: d.Size}, dp(gtx, 4))
	}
	if c.Off {
		defer paint.PushOpacity(gtx.Ops, 0.64).Pop()
	}
	call.Add(gtx.Ops)
	return d
}

// traitSwitch is a boolean option: label, then a switch whose knob slides
// over ~180 ms when it changes.
func (th *Theme) traitSwitch(gtx C, g Trait) D {
	return FixedH(gtx, 30, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				flexed(func(gtx C) D { return th.Label(gtx, ColText, font.Normal, g.Label) }),
				rigid(func(gtx C) D {
					amt := onAmt(g.On)
					if g.Moving {
						e := easeOut(g.T)
						amt = onAmt(!g.On) + (onAmt(g.On)-onAmt(!g.On))*e
					}
					return Switch(gtx, amt)
				}),
			)
		})
	})
}

// Switch draws an on/off switch; on runs 0 to 1 so it can be eased.
func Switch(gtx C, on float32) D {
	w, h := dp(gtx, 28), dp(gtx, 16)
	fill(gtx, lerpCol(ColBorderHi, ColAccent, on), image.Rect(0, 0, w, h), h/2)
	k := h - dp(gtx, 4)
	x := dp(gtx, 2) + int(float32(w-k-dp(gtx, 4))*on)
	circle(gtx, rgb(0xffffff), image.Pt(x+k/2, h/2), k/2)
	return D{Size: image.Pt(w, h)}
}

func onAmt(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

func easeOut(t float32) float32 { return 1 - (1-t)*(1-t)*(1-t) }

func lerpCol(a, b color.NRGBA, t float32) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t + 0.5) }
	return color.NRGBA{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: l(a.A, b.A)}
}

func (th *Theme) suggest(gtx C, s *Scene) D {
	return popupFrame(gtx, func(gtx C) D {
		return FixedW(gtx, 420, func(gtx C) D {
			return pad(6, 6, 6, 6).Layout(gtx, func(gtx C) D {
				kids := []layout.FlexChild{rigid(func(gtx C) D {
					return pad(4, 8, 6, 8).Layout(gtx, func(gtx C) D {
						return th.Text(gtx, Txt{Size: 10.5, Color: ColFaint, Font: UI(font.SemiBold), MaxLines: 1}, s.SuggestHead)
					})
				})}
				for i, sg := range s.Suggest {
					sg, on := sg, i == s.SuggestSel
					kids = append(kids, rigid(func(gtx C) D {
						return FixedH(gtx, 30, func(gtx C) D {
							size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
							if on {
								fill(gtx, ColSelected, image.Rectangle{Max: size}, dp(gtx, 4))
							}
							gtx.Constraints.Min.X = size.X
							return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return Row(gtx,
									rigid(func(gtx C) D {
										return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: ColText, MaxLines: 1}, sg.Name)
									}),
									rigid(func(gtx C) D {
										if sg.Hint == "" {
											return D{}
										}
										return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: ColFaint, MaxLines: 1}, " "+sg.Hint)
									}),
									rigid(Spacer(12, 0)),
									flexed(func(gtx C) D { return th.Small(gtx, ColDim, sg.Desc) }),
									rigid(func(gtx C) D {
										if !on {
											return D{}
										}
										return th.Kbd("Tab")(gtx)
									}),
								)
							})
						})
					}))
				}
				return Col(gtx, kids...)
			})
		})
	})
}

// ---------- inspector ----------

func (th *Theme) inspector(gtx C, s *Scene) D {
	return Fill(gtx, ColPanel, func(gtx C) D {
		var kids []layout.FlexChild
		if s.Run != nil {
			r := s.Run
			kids = append(kids, rigid(th.SectionLabel("RUN", nil)))
			rows := [][2]string{{"Status", r.Status}, {"Model", r.Model}, {"Elapsed", r.Elapsed}, {"Access", r.Mode}, {"Branch", r.Branch}, {"Folder", r.Worktree}}
			for _, kv := range rows {
				kv := kv
				if kv[1] == "" {
					continue
				}
				kids = append(kids, rigid(func(gtx C) D {
					return FixedH(gtx, 24, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return pad(0, 12, 0, 12).Layout(gtx, func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return Row(gtx,
								rigid(func(gtx C) D {
									return FixedW(gtx, 72, func(gtx C) D { return th.Label(gtx, ColFaint, font.Normal, kv[0]) })
								}),
								flexed(func(gtx C) D {
									if kv[0] == "Status" {
										glyph, col := Spinner(ColAccentHi, 11, s.Phase), ColAccentHi
										switch s.State {
										case ThreadApproval, ThreadQuestion:
											glyph, col = Dot(ColAmber, true), ColAmber
										case ThreadError:
											glyph, col = Dot(ColRed, false), ColRed
										case ThreadIdle:
											glyph, col = Dot(ColGreen, false), ColText
										}
										return Row(gtx,
											rigid(glyph),
											rigid(Spacer(6, 0)),
											rigid(func(gtx C) D { return th.Label(gtx, col, font.Medium, kv[1]) }),
										)
									}
									f := Txt{MaxLines: 1, Color: ColText}
									if kv[0] == "Branch" || kv[0] == "Folder" {
										f.Font, f.Size = Mono(), SizeMono
									}
									return th.Text(gtx, f, kv[1])
								}),
							)
						})
					})
				}))
			}
		}
		if len(s.Changes) > 0 {
			add, del := 0, 0
			for _, c := range s.Changes {
				add += c.Add
				del += c.Del
			}
			kids = append(kids,
				rigid(Spacer(0, 8)),
				rigid(th.SectionLabel("CHANGES", func(gtx C) D {
					return Row(gtx,
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColAddFg}, fmt.Sprintf("+%d", add))
						}),
						rigid(Spacer(5, 0)),
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColDelFg}, fmt.Sprintf("−%d", del))
						}),
					)
				})),
				rigid(func(gtx C) D {
					return pad(2, 12, 6, 12).Layout(gtx, func(gtx C) D { return th.segmented(gtx, []string{"Last turn", "Whole thread"}, 0) })
				}),
			)
			for _, c := range s.Changes {
				c := c
				kids = append(kids, rigid(func(gtx C) D { return th.fileRow(gtx, c) }))
			}
		}
		return Col(gtx, kids...)
	})
}

func (th *Theme) segmented(gtx C, opts []string, sel int) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: ColWindow, Border: ColBorder, Radius: 5, Pad: padXY(2, 2)}.Layout(gtx, func(gtx C) D {
		var kids []layout.FlexChild
		for i, o := range opts {
			o, on := o, i == sel
			kids = append(kids, flexed(func(gtx C) D {
				return FixedH(gtx, 22, func(gtx C) D {
					size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
					if on {
						fill(gtx, ColFrameActive, image.Rectangle{Max: size}, dp(gtx, 4))
					}
					col := ColDim
					if on {
						col = ColText
					}
					gtx.Constraints.Min = size
					layout.Center.Layout(gtx, func(gtx C) D { return th.Small(gtx, col, o) })
					return D{Size: size}
				})
			}))
		}
		return layout.Flex{}.Layout(gtx, kids...)
	})
}

func (th *Theme) fileRow(gtx C, c FileChange) D {
	return FixedH(gtx, 25, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		if c.Selected {
			fill(gtx, ColSelected, image.Rect(dp(gtx, 6), 0, size.X-dp(gtx, 6), size.Y), dp(gtx, 4))
		}
		gtx.Constraints.Min.X = size.X
		return pad(0, 12, 0, 12).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			kc := map[byte]color.NRGBA{'M': ColAmber, 'A': ColGreen, 'D': ColRed}[c.Kind]
			dir, name := "", c.Path
			if i := strings.LastIndexByte(c.Path, '/'); i >= 0 {
				dir, name = c.Path[:i+1], c.Path[i+1:]
			}
			return Row(gtx,
				rigid(func(gtx C) D {
					return FixedW(gtx, 16, func(gtx C) D {
						return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: kc}, string(c.Kind))
					})
				}),
				rigid(func(gtx C) D { return th.Label(gtx, ColText, font.Normal, name) }),
				rigid(Spacer(6, 0)),
				flexed(func(gtx C) D { return th.Small(gtx, ColFaint, dir) }),
				rigid(func(gtx C) D {
					return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColAddFg}, fmt.Sprintf("+%d", c.Add))
				}),
				rigid(Spacer(5, 0)),
				rigid(func(gtx C) D {
					return FixedW(gtx, 26, func(gtx C) D {
						return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColDelFg}, fmt.Sprintf("−%d", c.Del))
					})
				}),
			)
		})
	})
}

// ---------- status bar ----------

func (th *Theme) statusbar(gtx C, s *Scene) D {
	return FixedH(gtx, statusH, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		fill(gtx, ColPanel, image.Rectangle{Max: size}, 0)
		hline(gtx, ColBorder, 0, 0, size.X)
		gtx.Constraints.Min.X = size.X
		return pad(0, 12, 0, 12).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				rigid(Dot(ColGreen, false)),
				rigid(Spacer(6, 0)),
				rigid(func(gtx C) D { return th.Small(gtx, ColDim, s.Net) }),
				flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
				rigid(func(gtx C) D { return th.Small(gtx, ColFaint, s.Stats) }),
			)
		})
	})
}

// ---------- diff window ----------

func (th *Theme) diffWindow(gtx C, d *DiffView) {
	size := gtx.Constraints.Max
	ww, wh := dp(gtx, 920), dp(gtx, 560)
	at := image.Pt((size.X-ww)/2+dp(gtx, 40), (size.Y-wh)/2+dp(gtx, 10))
	t := op.Offset(at).Push(gtx.Ops)
	gtx.Constraints = layout.Exact(image.Pt(ww, wh))
	popupFrame(gtx, func(gtx C) D {
		return Col(gtx,
			rigid(func(gtx C) D {
				return FixedH(gtx, 32, func(gtx C) D {
					s := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
					fill(gtx, ColFrame, image.Rectangle{Max: s}, dp(gtx, 7))
					fill(gtx, ColFrame, image.Rect(0, s.Y/2, s.X, s.Y), 0)
					hline(gtx, ColBorderHi, s.Y-max(1, dp(gtx, 1)), 0, s.X)
					gtx.Constraints.Min.X = s.X
					return pad(0, 8, 0, 12).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return Row(gtx,
							rigid(IconW(IconFile, ColDim, 14)),
							rigid(Spacer(7, 0)),
							rigid(func(gtx C) D { return th.Label(gtx, ColText, font.SemiBold, d.Title) }),
							flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
							rigid(th.Checkbox(true, "Ignore whitespace")),
							rigid(Spacer(14, 0)),
							rigid(th.Button(Ghost, IconClose, true, "")),
						)
					})
				})
			}),
			flexed(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					rigid(func(gtx C) D {
						return FixedW(gtx, 230, func(gtx C) D {
							return pad(8, 0, 0, 0).Layout(gtx, func(gtx C) D {
								var kids []layout.FlexChild
								for _, f := range d.Files {
									f := f
									kids = append(kids, rigid(func(gtx C) D { return th.fileRow(gtx, f) }))
								}
								return Col(gtx, kids...)
							})
						})
					}),
					rigid(divider),
					flexed(func(gtx C) D { return th.diffBody(gtx, d) }),
				)
			}),
		)
	})
	t.Pop()
}

func (th *Theme) diffBody(gtx C, d *DiffView) D {
	size := gtx.Constraints.Max
	fill(gtx, ColWindow, image.Rectangle{Max: size}, 0)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	gtx.Constraints.Min.X = size.X
	return Col(gtx,
		rigid(func(gtx C) D {
			return FixedH(gtx, 30, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				hline(gtx, ColBorder, dp(gtx, 30)-1, 0, gtx.Constraints.Max.X)
				return pad(0, 12, 0, 12).Layout(gtx, func(gtx C) D {
					return Row(gtx,
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: ColText, MaxLines: 1}, d.File)
						}),
					)
				})
			})
		}),
		flexed(func(gtx C) D {
			var kids []layout.FlexChild
			for _, l := range d.Lines {
				l := l
				kids = append(kids, rigid(func(gtx C) D { return th.diffLine(gtx, l) }))
			}
			return Col(gtx, kids...)
		}),
	)
}

func (th *Theme) diffLine(gtx C, l DiffLine) D {
	return FixedH(gtx, 20, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		bg, fg, gut := color.NRGBA{}, rgb(0xc9cfd8), ColFaint
		switch l.Kind {
		case '+':
			bg, fg, gut = ColAddBg, ColAddFg, alpha(ColAddFg, 0x90)
		case '-':
			bg, fg, gut = ColDelBg, ColDelFg, alpha(ColDelFg, 0x90)
		case '@':
			bg, fg = alpha(ColAccent, 0x12), ColHunkFg
		}
		if bg.A > 0 {
			fill(gtx, bg, image.Rectangle{Max: size}, 0)
		}
		num := func(n int) string {
			if n == 0 {
				return ""
			}
			return fmt.Sprint(n)
		}
		mono := func(col color.NRGBA, s string, align bool) W {
			return func(gtx C) D {
				t := Txt{Size: SizeMono, Font: Mono(), Color: col, MaxLines: 1}
				if align {
					t.Align = 2 // text.End
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
				}
				return th.Text(gtx, t, s)
			}
		}
		gtx.Constraints.Min.X = size.X
		return Row(gtx,
			rigid(func(gtx C) D { return FixedW(gtx, 40, mono(gut, num(l.Old), true)) }),
			rigid(func(gtx C) D { return FixedW(gtx, 40, mono(gut, num(l.New), true)) }),
			rigid(Spacer(10, 0)),
			rigid(func(gtx C) D {
				sign := string(l.Kind)
				if l.Kind == '@' {
					sign = " "
				}
				return FixedW(gtx, 14, mono(gut, sign, false))
			}),
			flexed(mono(fg, l.Text, false)),
		)
	})
}

// ---------- phone ----------

// Phone is the same thread view at phone width: one column, a back button
// instead of the sidebar, the composer's secondary controls tucked away.
func (th *Theme) Phone(gtx C, s *Scene) D {
	return Fill(gtx, ColPanelHi, func(gtx C) D {
		return Col(gtx,
			rigid(func(gtx C) D {
				// Status bar inset.
				return FixedH(gtx, 30, func(gtx C) D {
					size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
					fill(gtx, ColPanel, image.Rectangle{Max: size}, 0)
					gtx.Constraints.Min.X = size.X
					return pad(0, 22, 0, 26).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return Row(gtx,
							rigid(func(gtx C) D { return th.Label(gtx, ColText, font.SemiBold, "9:41") }),
							flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
							rigid(func(gtx C) D { return th.Small(gtx, ColDim, "5G  87%") }),
						)
					})
				})
			}),
			flexed(func(gtx C) D { return th.threadView(gtx, s, true) }),
			rigid(func(gtx C) D {
				return FixedH(gtx, 20, func(gtx C) D {
					size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
					bw := dp(gtx, 134)
					fill(gtx, ColDim, image.Rect((size.X-bw)/2, dp(gtx, 8), (size.X+bw)/2, dp(gtx, 13)), dp(gtx, 3))
					return D{Size: size}
				})
			}),
		)
	})
}

// ---------- right panel ----------

// Panel surfaces, T3's right panel tab kinds.
const (
	PanelTerminal = iota
	PanelBrowser
	PanelChanges
)

// rightPanel is T3's tabbed side panel: every surface (browser, terminal,
// changes) is a tab, and "+" opens another.
func (th *Theme) rightPanel(gtx C, s *Scene) D {
	return Fill(gtx, ColPanel, func(gtx C) D {
		return Col(gtx,
			rigid(func(gtx C) D {
				if s.Live {
					return th.panelHeader(gtx, s)
				}
				return th.panelTabs(gtx, s)
			}),
			flexed(func(gtx C) D {
				if s.Live {
					return th.inspector(gtx, s)
				}
				switch s.Panel {
				case PanelBrowser:
					return th.browser(gtx, s)
				case PanelChanges:
					return th.inspector(gtx, s)
				}
				return th.splitTerminal(gtx, s)
			}),
		)
	})
}

// panelHeader tops the live details sidebar: its name and a close button.
// Middle-click on it closes it too.
func (th *Theme) panelHeader(gtx C, s *Scene) D {
	return th.Click("panel:details", func(gtx C) D {
		return FixedH(gtx, 34, func(gtx C) D {
			size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
			hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
			gtx.Constraints.Min.X = size.X
			return pad(0, 8, 0, 14).Layout(gtx, func(gtx C) D {
				return Row(gtx,
					rigid(Middle(func(gtx C) D { return th.Label(gtx, ColDim, font.Medium, "Details") })),
					flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
					rigid(Middle(th.Click(liveID(s, "panel.close"), func(gtx C) D {
						return pad(5, 5, 5, 5).Layout(gtx, IconW(IconClose, ColFaint, 12))
					}))),
				)
			})
		})
	})(gtx)
}

func (th *Theme) panelTabs(gtx C, s *Scene) D {
	return FixedH(gtx, 34, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
		gtx.Constraints.Min.X = size.X
		tabW := func(kind int, ic Icon, label string, trail W) W {
			return func(gtx C) D {
				on := s.Panel == kind
				m := op.Record(gtx.Ops)
				dims := pad(0, 8, 0, 10).Layout(gtx, func(gtx C) D {
					return FixedH(gtx, 34, func(gtx C) D {
						col := ColDim
						if on {
							col = ColText
						}
						kids := []layout.FlexChild{
							rigid(IconW(ic, col, 13)),
							rigid(Spacer(6, 0)),
							rigid(func(gtx C) D { return th.Label(gtx, col, font.Medium, label) }),
						}
						if trail != nil {
							kids = append(kids, rigid(Spacer(6, 0)), rigid(trail))
						}
						if on {
							kids = append(kids, rigid(Spacer(8, 0)), rigid(th.Click(liveID(s, "panel.close"), IconW(IconClose, ColFaint, 12))))
						}
						return Row(gtx, kids...)
					})
				})
				call := m.Stop()
				r := image.Rectangle{Max: dims.Size}
				if on {
					fill(gtx, ColPanelHi, r, 0)
					fill(gtx, ColAccent, image.Rect(0, 0, r.Max.X, dp(gtx, 2)), 0)
				}
				vline(gtx, ColBorder, r.Max.X-max(1, dp(gtx, 1)), 0, r.Max.Y)
				call.Add(gtx.Ops)
				return dims
			}
		}
		tab := func(kind int, ic Icon, label string, trail W) layout.FlexChild {
			return rigid(tabW(kind, ic, label, trail))
		}
		kids := []layout.FlexChild{}
		if s.Browser != nil {
			label := s.Browser.Title
			if label == "" {
				label = "New tab"
			}
			kids = append(kids, tab(PanelBrowser, IconGlobe, label, nil))
		}
		if !s.Live {
			kids = append(kids, tab(PanelTerminal, IconTerminal, "Terminal", func(gtx C) D {
				return Box{Bg: ColFrame, Radius: 7, Pad: padXY(5, 0)}.Layout(gtx, func(gtx C) D {
					n := "2"
					if s.Run == nil {
						n = "1" // a draft has only the fresh shell
					}
					return th.Small(gtx, ColDim, n)
				})
			}))
		}
		if len(s.Changes) > 0 {
			add, del := 0, 0
			for _, c := range s.Changes {
				add, del = add+c.Add, del+c.Del
			}
			kids = append(kids, tab(PanelChanges, IconFile, "Changes", func(gtx C) D {
				return Row(gtx,
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColAddFg, MaxLines: 1}, fmt.Sprintf("+%d", add))
					}),
					rigid(Spacer(4, 0)),
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: SizeSmall, Font: Mono(), Color: ColDelFg, MaxLines: 1}, fmt.Sprintf("−%d", del))
					}),
				)
			}))
		}
		if s.Live {
			// Middle-click closes it, like a browser tab.
			kids = kids[:0]
			kids = append(kids, rigid(th.Click("panel:details", tabW(PanelChanges, IconFile, "Details", nil))))
		}
		kids = append(kids,
			rigid(func(gtx C) D {
				if s.Live {
					return D{}
				}
				return padXY(8, 0).Layout(gtx, IconW(IconPlus, ColDim, 14))
			}),
			flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
			rigid(func(gtx C) D { return pad(0, 10, 0, 0).Layout(gtx, IconW(IconDots, ColFaint, 14)) }),
		)
		return Row(gtx, kids...)
	})
}

// termSpan is a run of terminal text in one colour.
type termSpan struct {
	text string
	col  color.NRGBA
}

func prompt(cmd string) []termSpan {
	return []termSpan{{"~/tern ", ColCyan}, {"❯ ", ColGreen}, {cmd, ColText}}
}

var termOut = rgb(0xaeb5c0)

// shellLines is the user's own shell in the thread's worktree.
func shellLines(fresh bool) [][]termSpan {
	if fresh {
		return [][]termSpan{{{"Last login: Sun Oct 25 09:14", ColFaint}}, prompt("")}
	}
	ok := func(pkg, t string) []termSpan {
		return []termSpan{{"ok   ", ColGreen}, {pkg, termOut}, {t, ColFaint}}
	}
	return [][]termSpan{
		prompt("git log --oneline -3"),
		{{"79cdcbc ", ColCode}, {"feat(connect): add a T3 Connect browser client", termOut}},
		{{"b5d25db ", ColCode}, {"chore: initial commit", termOut}},
		{{"d4ca7b8 ", ColCode}, {"chore: initial commit", termOut}},
		prompt("git status -sb"),
		{{"## ", ColFaint}, {"t3/go-client", ColGreen}, {"...origin/t3/go-client [ahead 3]", ColFaint}},
		{{" M ", ColRed}, {"app/t3/relay.go", termOut}},
		{{" M ", ColRed}, {"app/t3/rpc.go", termOut}},
		{{"?? ", ColRed}, {"app/app.go", termOut}},
		{{"?? ", ColRed}, {"app/t3/dpop_test.go", termOut}},
		prompt("go test ./t3/... ./ui/..."),
		ok("github.com/some-du6e/tern/app/t3       ", "0.412s"),
		ok("github.com/some-du6e/tern/app/t3/rpc   ", "0.218s"),
		ok("github.com/some-du6e/tern/app/ui       ", "0.087s"),
		prompt(""),
	}
}

// appLines is the second pane: the app itself, talking to bowen.
func appLines() [][]termSpan {
	t := func(ts string) termSpan { return termSpan{ts + "  ", ColFaint} }
	ev := func(ts, k, v string, col color.NRGBA, rest string) []termSpan {
		return []termSpan{t(ts), {k, ColViolet}, {v, col}, {rest, ColFaint}}
	}
	return [][]termSpan{
		prompt("go run ./cmd/tern -env bowen -v"),
		{{"tern 0.1.0  gio 0.10.3  opengl  1440x900 @2x", ColFaint}},
		ev("10:21:04.118", "clerk   ", "session ok", termOut, "      sess_2q…f9 · 52m left"),
		ev("10:21:04.149", "dpop    ", "proof", termOut, "           jkt Xq3v…tA"),
		ev("10:21:04.337", "relay   ", "connect 200", ColGreen, "     188ms"),
		ev("10:21:04.431", "relay   ", "token 200", ColGreen, "       94ms · 30m"),
		ev("10:21:04.512", "ws      ", "ticket 404", ColAmber, "      /rpc → bare route"),
		ev("10:21:04.540", "ws      ", "ticket 200", ColGreen, "      26ms"),
		ev("10:21:04.611", "ws      ", "open", termOut, "            wss://bowen:3773/ws"),
		ev("10:21:04.702", "shell   ", "snapshot", termOut, "        1,024 threads · 18 projects"),
		ev("10:21:04.703", "shell   ", "synchronized", ColGreen, "    seq 88412"),
		ev("10:21:05.990", "thread  ", "subscribe", termOut, "       t_9fk2 after 88390"),
		ev("10:21:06.004", "thread  ", "chunk ×22", termOut, "       acked"),
		ev("10:21:36.004", "ws      ", "ping", termOut, "            38ms"),
		ev("10:22:06.010", "ws      ", "ping", termOut, "            41ms"),
	}
}

// splitTerminal is two terminals stacked, the way T3 splits its drawer.
func (th *Theme) splitTerminal(gtx C, s *Scene) D {
	size := gtx.Constraints.Max
	fill(gtx, ColWindow, image.Rectangle{Max: size}, 0)
	gtx.Constraints.Min = size
	fresh := s.Run == nil
	if fresh {
		return th.termPane(gtx, "zsh", "~/tern", IconW(IconTerminal, ColDim, 12), shellLines(true), true, s.Phase)
	}
	top := size.Y * 45 / 100
	gtx.Constraints = layout.Exact(image.Pt(size.X, top))
	th.termPane(gtx, "zsh", "~/.t3/worktrees/tern", IconW(IconTerminal, ColDim, 12), shellLines(false), true, s.Phase)
	hline(gtx, ColBorderHi, top, 0, size.X)
	t := op.Offset(image.Pt(0, top+max(1, dp(gtx, 1)))).Push(gtx.Ops)
	gtx.Constraints = layout.Exact(image.Pt(size.X, size.Y-top-max(1, dp(gtx, 1))))
	th.termPane(gtx, "go run ./cmd/tern", "bowen", Spinner(ColGreen, 10, s.Phase+2), appLines(), false, s.Phase)
	t.Pop()
	return D{Size: size}
}

// termPane is one terminal: a slim title strip, then the scrollback with the
// newest line kept in view.
func (th *Theme) termPane(gtx C, title, where string, lead W, lines [][]termSpan, focused bool, phase float64) D {
	size := gtx.Constraints.Max
	gtx.Constraints.Min = size
	Col(gtx,
		rigid(func(gtx C) D {
			return FixedH(gtx, 26, func(gtx C) D {
				gtx.Constraints.Min.X = size.X
				return pad(0, 10, 0, 12).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					col := ColDim
					if focused {
						col = ColText
					}
					return Row(gtx,
						rigid(lead),
						rigid(Spacer(6, 0)),
						rigid(func(gtx C) D {
							return th.Text(gtx, Txt{Size: SizeSmall, Color: col, Font: UI(font.Medium), MaxLines: 1}, title)
						}),
						rigid(Spacer(8, 0)),
						flexed(func(gtx C) D { return th.Small(gtx, ColFaint, where) }),
						rigid(IconW(IconSplit, ColFaint, 13)),
						rigid(Spacer(8, 0)),
						rigid(IconW(IconClose, ColFaint, 12)),
					)
				})
			})
		}),
		flexed(func(gtx C) D {
			sz := gtx.Constraints.Max
			defer clip.Rect{Max: sz}.Push(gtx.Ops).Pop()
			m := op.Record(gtx.Ops)
			gtx.Constraints.Min = image.Point{}
			dims := pad(2, 12, 8, 12).Layout(gtx, func(gtx C) D {
				var rows []layout.FlexChild
				for i, ln := range lines {
					ln, last := ln, i == len(lines)-1
					rows = append(rows, rigid(func(gtx C) D {
						var segs []layout.FlexChild
						for _, sp := range ln {
							sp := sp
							segs = append(segs, rigid(func(gtx C) D {
								return th.Text(gtx, Txt{Size: 11.5, Font: Mono(), Color: sp.col, MaxLines: 1, LineH: 1.5}, sp.text)
							}))
						}
						if last && focused {
							segs = append(segs, rigid(func(gtx C) D {
								w, h := dp(gtx, 7), dp(gtx, 15)
								fill(gtx, alpha(ColText, 0xc0), image.Rect(0, 0, w, h), 0)
								return D{Size: image.Pt(w, h)}
							}))
						}
						return Row(gtx, segs...)
					}))
				}
				return Col(gtx, rows...)
			})
			call := m.Stop()
			op.Offset(image.Pt(0, min(0, sz.Y-dims.Size.Y))).Add(gtx.Ops)
			call.Add(gtx.Ops)
			return D{Size: sz}
		}),
	)
	return D{Size: size}
}

// ---------- browser ----------

func (th *Theme) browser(gtx C, s *Scene) D {
	b := s.Browser
	return Col(gtx,
		rigid(func(gtx C) D {
			return FixedH(gtx, 38, func(gtx C) D {
				size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
				fill(gtx, ColPanelHi, image.Rectangle{Max: size}, 0)
				hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
				gtx.Constraints.Min.X = size.X
				return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					nav := func(ic Icon, col color.NRGBA) layout.FlexChild {
						return rigid(func(gtx C) D { return padXY(4, 0).Layout(gtx, IconW(ic, col, 14)) })
					}
					kids := []layout.FlexChild{
						nav(IconBack, ColDim),
						nav(IconForward, ColFaint),
						nav(IconReload, ColDim),
						rigid(Spacer(4, 0)),
						flexed(func(gtx C) D {
							return Box{Bg: ColWindow, Border: borderIf(b.URL == "", alpha(ColAccent, 0xa0)), Radius: 12, Pad: padXY(10, 0)}.Layout(gtx, func(gtx C) D {
								return FixedH(gtx, 24, func(gtx C) D {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									if b.URL == "" {
										return Row(gtx,
											rigid(IconW(IconSearch, ColFaint, 12)),
											rigid(Spacer(6, 0)),
											rigid(func(gtx C) D { return Caret(gtx, 14) }),
											rigid(func(gtx C) D { return th.Small(gtx, ColFaint, "Search or enter a URL") }),
										)
									}
									return Row(gtx,
										rigid(IconW(IconGlobe, ColFaint, 12)),
										rigid(Spacer(6, 0)),
										rigid(func(gtx C) D { return th.Small(gtx, ColFaint, "http://") }),
										flexed(func(gtx C) D { return th.Small(gtx, ColText, b.URL) }),
									)
								})
							})
						}),
					}
					if b.Control != "" {
						kids = append(kids, rigid(Spacer(8, 0)), rigid(th.Chip(ColGreen, b.Control, nil)))
					}
					return Row(gtx, kids...)
				})
			})
		}),
		flexed(func(gtx C) D {
			if b.URL == "" {
				return th.browserEmpty(gtx)
			}
			return th.pocPage(gtx)
		}),
	)
}

func borderIf(cond bool, col color.NRGBA) color.NRGBA {
	if cond {
		return col
	}
	return ColBorder
}

// browserEmpty is T3's new-tab page: dev servers it found on the machine,
// then recent pages.
func (th *Theme) browserEmpty(gtx C) D {
	return Fill(gtx, ColPanelHi, func(gtx C) D {
		row := func(dot color.NRGBA, url, what string) layout.FlexChild {
			return rigid(func(gtx C) D {
				return FixedH(gtx, 30, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return pad(0, 14, 0, 14).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return Row(gtx,
							rigid(func(gtx C) D {
								if dot.A == 0 {
									return IconW(IconGlobe, ColFaint, 13)(gtx)
								}
								return pad(0, 3, 0, 2).Layout(gtx, Dot(dot, true))
							}),
							rigid(Spacer(9, 0)),
							rigid(func(gtx C) D {
								return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: ColText, MaxLines: 1}, url)
							}),
							rigid(Spacer(10, 0)),
							flexed(func(gtx C) D { return th.Small(gtx, ColFaint, what) }),
						)
					})
				})
			})
		}
		return pad(6, 0, 0, 0).Layout(gtx, func(gtx C) D {
			return Col(gtx,
				rigid(th.SectionLabel("RUNNING ON WORKSTATION", nil)),
				row(ColGreen, "localhost:8787", "node poc/server.mjs"),
				row(ColGreen, "localhost:5173", "vite · schoolwork-writer"),
				row(ColGreen, "localhost:3773", "t3 server"),
				rigid(Spacer(0, 8)),
				rigid(th.SectionLabel("RECENT", nil)),
				row(color.NRGBA{}, "github.com/some-du6e/tern/pull/1", "feat(connect): add a T3 Conn…"),
				row(color.NRGBA{}, "gioui.org/doc/architecture", "Gio · Architecture"),
				row(color.NRGBA{}, "pkg.go.dev/github.com/coder/websocket", "websocket"),
			)
		})
	})
}

// pocPage is the tern browser client from ../poc as the preview renders it:
// its own colours, not the app theme.
func (th *Theme) pocPage(gtx C) D {
	var (
		bg, panel, line = rgb(0x0d0e10), rgb(0x15171a), rgb(0x24272c)
		text, dim       = rgb(0xe6e6e6), rgb(0x8b9099)
		accent, ok      = rgb(0x7aa2ff), rgb(0x4ade80)
		user, on        = rgb(0x1d2433), rgb(0x20252d)
	)
	t := func(col color.NRGBA, w font.Weight, s string) W {
		return func(gtx C) D { return th.Text(gtx, Txt{Size: 11, Color: col, Font: UI(w), MaxLines: 1}, s) }
	}
	para := func(col color.NRGBA, s string) W {
		return func(gtx C) D { return th.Text(gtx, Txt{Size: 11, Color: col, LineH: 1.4}, s) }
	}
	return Fill(gtx, bg, func(gtx C) D {
		size := gtx.Constraints.Max
		return Col(gtx,
			rigid(func(gtx C) D {
				return FixedH(gtx, 32, func(gtx C) D {
					sz := image.Pt(size.X, gtx.Constraints.Min.Y)
					fill(gtx, panel, image.Rectangle{Max: sz}, 0)
					hline(gtx, line, sz.Y-1, 0, sz.X)
					gtx.Constraints.Min.X = sz.X
					return pad(0, 10, 0, 12).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return Row(gtx,
							rigid(t(text, font.SemiBold, "tern")),
							flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
							rigid(t(dim, font.Normal, "karim@example.com")),
							rigid(Spacer(8, 0)),
							rigid(func(gtx C) D {
								return Box{Bg: line, Radius: 5, Pad: padXY(8, 3)}.Layout(gtx, t(text, font.Normal, "Sign out"))
							}),
						)
					})
				})
			}),
			flexed(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					rigid(func(gtx C) D {
						return FixedW(gtx, 128, func(gtx C) D {
							h := gtx.Constraints.Max.Y
							fill(gtx, panel, image.Rect(0, 0, gtx.Constraints.Max.X, h), 0)
							vline(gtx, line, gtx.Constraints.Max.X-1, 0, h)
							item := func(lead W, s string, sel bool) layout.FlexChild {
								return rigid(func(gtx C) D {
									return FixedH(gtx, 24, func(gtx C) D {
										w := gtx.Constraints.Max.X
										if sel {
											fill(gtx, on, image.Rect(dp(gtx, 4), 0, w-dp(gtx, 4), gtx.Constraints.Min.Y), dp(gtx, 4))
										}
										gtx.Constraints.Min.X = w
										return pad(0, 8, 0, 10).Layout(gtx, func(gtx C) D {
											kids := []layout.FlexChild{}
											if lead != nil {
												kids = append(kids, rigid(lead), rigid(Spacer(6, 0)))
											}
											kids = append(kids, flexed(t(text, font.Normal, s)))
											gtx.Constraints.Min.X = gtx.Constraints.Max.X
											return Row(gtx, kids...)
										})
									})
								})
							}
							head := func(s string) layout.FlexChild {
								return rigid(func(gtx C) D {
									return pad(10, 0, 4, 10).Layout(gtx, func(gtx C) D {
										return th.Text(gtx, Txt{Size: 9.5, Color: dim, Font: UI(font.SemiBold), MaxLines: 1}, s)
									})
								})
							}
							return Col(gtx,
								head("ENVIRONMENTS"),
								item(Dot(ok, false), "workstation", true),
								item(Dot(ok, false), "bowen", false),
								item(Dot(rgb(0xf87171), false), "macbook", false),
								head("THREADS"),
								item(nil, "Port the T3 Connect…", true),
								item(nil, "Watch pairing flow", false),
								item(nil, "Gio sidebar: virtua…", false),
								item(nil, "Relay status polling", false),
								item(nil, "tern roadmap", false),
							)
						})
					}),
					flexed(func(gtx C) D {
						return pad(12, 12, 10, 12).Layout(gtx, func(gtx C) D {
							return Col(gtx,
								rigid(func(gtx C) D {
									return layout.E.Layout(gtx, func(gtx C) D {
										gtx.Constraints.Max.X = gtx.Constraints.Max.X * 85 / 100
										return Box{Bg: user, Radius: 10, Pad: padXY(10, 7)}.Layout(gtx, para(text, "nice. wire it into the UI and keep the socket alive across sleep"))
									})
								}),
								rigid(Spacer(0, 10)),
								rigid(t(accent, font.Medium, "▸ rg -n 'onclose|visibilitychange'")),
								rigid(Spacer(0, 4)),
								rigid(t(accent, font.Medium, "▸ Edited app/t3/rpc.go")),
								rigid(Spacer(0, 4)),
								rigid(t(accent, font.Medium, "▸ go test ./t3/...")),
								rigid(Spacer(0, 10)),
								rigid(para(text, "Ported. app/t3 now has Clerk, DPoP, Relay and the RPC socket. The client acks every Chunk and treats 15 s without a pong as a dead socket.")),
								rigid(Spacer(0, 8)),
								rigid(func(gtx C) D {
									gtx.Constraints.Min.X = gtx.Constraints.Max.X
									return Box{Bg: rgb(0x111316), Border: line, Radius: 6, Pad: padXY(9, 7)}.Layout(gtx, func(gtx C) D {
										return th.Text(gtx, Txt{Size: 10.5, Font: Mono(), Color: text, LineH: 1.4}, "ok  tern/app/t3      0.412s\nok  tern/app/t3/rpc  0.218s")
									})
								}),
								flexed(func(gtx C) D { return D{Size: image.Pt(0, gtx.Constraints.Min.Y)} }),
								rigid(func(gtx C) D {
									return Row(gtx,
										flexed(func(gtx C) D {
											return Box{Bg: bg, Border: line, Radius: 6, Pad: padXY(9, 8)}.Layout(gtx, func(gtx C) D {
												gtx.Constraints.Min.X = gtx.Constraints.Max.X
												return t(dim, font.Normal, "Message…")(gtx)
											})
										}),
										rigid(Spacer(6, 0)),
										rigid(func(gtx C) D {
											return Box{Bg: accent, Radius: 6, Pad: padXY(11, 7)}.Layout(gtx, t(rgb(0x0b1020), font.SemiBold, "Send"))
										}),
									)
								}),
							)
						})
					}),
				)
			}),
		)
	})
}
