package ui

import (
	"image"
	"image/color"
	"io"
	"runtime"
	"strconv"
	"strings"

	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// The terminal drawer, T3's bottom panel: a tab per shell over the open
// one, drawn cell by cell from its emulator. Targets: "terminal.toggle",
// "terminal.new", "terminal.tab:i", "terminal.close:i".

// TermView is the open thread's drawer.
type TermView struct {
	Tabs   []TermTab
	Active int
	Exited bool // the open shell has ended
	Screen TermScreen
	// The drawer's input goes straight to the shell through these.
	Key    func(uv.KeyPressEvent)
	Text   func(string)
	Paste  func(string)
	Resize func(cols, rows int)
}

type TermTab struct {
	Label        string
	Busy, Exited bool
}

// TermScreen is what the drawer reads off an emulator.
type TermScreen interface {
	Width() int
	Height() int
	CellAt(x, y int) *uv.Cell
	CursorPosition() uv.Position
	CursorHidden() bool
	IsAltScreen() bool
	ScrollbackLen() int
	ScrollbackCellAt(x, y int) *uv.Cell
}

const (
	termH    = 280 // the drawer's default height, dp
	termSize = unit.Sp(12.5)
)

// termPalette is the 16 ANSI colours, VS Code's dark terminal theme as T3
// uses it.
var termPalette = [16]color.NRGBA{
	rgb(0x000000), rgb(0xcd3131), rgb(0x0dbc79), rgb(0xe5e510), rgb(0x2472c8), rgb(0xbc3fbc), rgb(0x11a8cd), rgb(0xe5e5e5),
	rgb(0x666666), rgb(0xf14c4c), rgb(0x23d18b), rgb(0xf5f543), rgb(0x3b8eea), rgb(0xd670d6), rgb(0x29b8db), rgb(0xffffff),
}

func termColor(c color.Color, def color.NRGBA) color.NRGBA {
	switch c := c.(type) {
	case nil:
		return def
	case ansi.BasicColor:
		return termPalette[c&15]
	case ansi.IndexedColor:
		if c < 16 {
			return termPalette[c]
		}
	}
	r, g, b, _ := c.RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xff}
}

// terminalDrawer is the drawer: tabs, then the open shell.
// terminalDrawer fills the height the shell gives it (drawerPx): T3's
// frame, a 1dp top border at 80% and a 6dp strip along the top that drags
// it taller or shorter.
func (th *Theme) terminalDrawer(gtx C, s *Scene) D {
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	hline(gtx, over(alpha(P.Border, uint8(int(P.Border.A)*8/10)), P.Background), 0, 0, size.X)
	// The strip goes over what's drawn, so it's last, and the drag
	// shows at the next frame.
	defer func() {
		if d := th.dragEdge(gtx, &th.in.termEdge, image.Pt(size.X, dp(gtx, 6)), false); d != 0 {
			th.in.termH -= d / gtx.Metric.PxPerDp
			gtx.Execute(op.InvalidateCmd{})
		}
	}()
	return Col(gtx,
		rigid(func(gtx C) D { return th.termTabs(gtx, s.Term) }),
		flexed(func(gtx C) D {
			return pad(4, 10, 8, 12).Layout(gtx, func(gtx C) D { return th.termScreen(gtx, s.Term) })
		}),
	)
}

func (th *Theme) termTabs(gtx C, v *TermView) D {
	return FixedH(gtx, 34, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			kids := []layout.FlexChild{}
			for i, t := range v.Tabs {
				i, t := i, t
				on := i == v.Active
				kids = append(kids, rigid(Middle(th.Click("terminal.tab:"+strconv.Itoa(i), func(gtx C) D {
					bg := color.NRGBA{}
					if on {
						bg = ColFrameActive
					}
					return Box{Bg: bg, Radius: 5, Pad: pad(4, 4, 4, 8)}.Layout(gtx, func(gtx C) D {
						col := ColDim
						if on {
							col = ColText
						}
						if t.Exited {
							col = ColFaint
						}
						return Row(gtx,
							rigid(Middle(func(gtx C) D {
								if t.Busy {
									return Spinner(ColAccentHi, 12, 0)(gtx)
								}
								return IconW(IconTerminal, col, 14)(gtx)
							})),
							rigid(Spacer(6, 0)),
							rigid(Middle(func(gtx C) D { return th.Label(gtx, col, font.Medium, t.Label) })),
							rigid(Spacer(4, 0)),
							rigid(Middle(th.Click("terminal.close:"+strconv.Itoa(i), func(gtx C) D {
								return pad(2, 2, 2, 2).Layout(gtx, IconW(IconClose, ColFaint, 11))
							}))),
						)
					})
				}))), rigid(Spacer(4, 0)))
			}
			kids = append(kids,
				rigid(Middle(th.Tip("terminal.new", "New terminal", true, 0, func(gtx C) D {
					return pad(5, 5, 5, 5).Layout(gtx, IconW(IconPlus, ColDim, 14))
				}))),
				flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
				rigid(Middle(th.Tip("terminal.toggle", "Hide terminal ("+Shortcut("J")+")", true, 1, func(gtx C) D {
					return pad(5, 5, 5, 5).Layout(gtx, IconW(IconClose, ColDim, 14))
				}))),
			)
			return Row(gtx, kids...)
		})
	})
}

// termCell measures a cell of the mono font at termSize.
func (th *Theme) termCell(gtx C) image.Point {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	d := widget.Label{MaxLines: 1}.Layout(g, th.Shaper, Mono(), termSize, strings.Repeat("M", 20), op.CallOp{})
	m.Stop()
	return image.Pt(max(1, d.Size.X/20), max(1, d.Size.Y))
}

// termScreen draws the shell and feeds it the keyboard and the wheel.
func (th *Theme) termScreen(gtx C, v *TermView) D {
	size := gtx.Constraints.Max
	cell := th.termCell(gtx)
	cols, rows := max(2, size.X/cell.X), max(1, size.Y/cell.Y)
	if v.Resize != nil {
		v.Resize(cols, rows)
	}
	tag := &th.in.term
	focused := gtx.Focused(tag)
	scr := v.Screen
	th.termInput(gtx, v, tag, cell)

	// The input area: a click focuses the shell, the wheel scrolls back.
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	pointer.CursorText.Add(gtx.Ops)
	area.Pop()

	back := 0
	if !scr.IsAltScreen() {
		th.in.termScroll = min(th.in.termScroll, scr.ScrollbackLen())
		back = th.in.termScroll
	} else {
		th.in.termScroll = 0
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	w, h := min(cols, scr.Width()), min(rows, scr.Height())
	at := func(x, y int) *uv.Cell {
		if y < back {
			return scr.ScrollbackCellAt(x, scr.ScrollbackLen()-back+y)
		}
		return scr.CellAt(x, y-back)
	}
	cx, cy := -1, -1
	if p := scr.CursorPosition(); !scr.CursorHidden() {
		cx, cy = p.X, p.Y+back
	}
	type run struct {
		x0     int
		fg     color.NRGBA
		bold   bool
		under  bool
		strike bool
		text   strings.Builder
	}
	for y := 0; y < h; y++ {
		var r *run
		flush := func() {
			if r == nil || strings.TrimSpace(r.text.String()) == "" {
				r = nil
				return
			}
			f := Mono()
			if r.bold {
				f.Weight = font.Medium
			}
			off := op.Offset(image.Pt(r.x0*cell.X, y*cell.Y)).Push(gtx.Ops)
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(1<<20, cell.Y)}
			m := op.Record(gtx.Ops)
			paint.ColorOp{Color: r.fg}.Add(gtx.Ops)
			col := m.Stop()
			d := widget.Label{MaxLines: 1}.Layout(g, th.Shaper, f, termSize, r.text.String(), col)
			if r.under {
				fill(gtx, r.fg, image.Rect(0, cell.Y-max(1, dp(gtx, 1))-dp(gtx, 1), d.Size.X, cell.Y-dp(gtx, 1)), 0)
			}
			if r.strike {
				fill(gtx, r.fg, image.Rect(0, cell.Y/2, d.Size.X, cell.Y/2+max(1, dp(gtx, 1))), 0)
			}
			off.Pop()
			r = nil
		}
		for x := 0; x < w; x++ {
			c := at(x, y)
			content, wide := " ", 1
			var st uv.Style
			if c != nil {
				if c.Width == 0 && c.Content == "" {
					continue // the second half of a wide character
				}
				st = c.Style
				if c.Content != "" {
					content = c.Content
				}
				wide = max(1, c.Width)
			}
			fg, bg := termColor(st.Fg, ColText), termColor(st.Bg, ColWindow)
			if st.Attrs&uv.AttrReverse != 0 {
				fg, bg = bg, fg
			}
			if st.Attrs&uv.AttrFaint != 0 {
				fg = alpha(fg, 0x99)
			}
			if x == cx && y == cy && focused {
				fg, bg = ColWindow, ColText
			}
			if th.in.termSel.has(v.Active, scr.ScrollbackLen()-back+y, x) {
				bg = mix(ColAccent, bg, 0.45)
			}
			if bg != ColWindow {
				fill(gtx, bg, image.Rect(x*cell.X, y*cell.Y, (x+wide)*cell.X, (y+1)*cell.Y), 0)
			}
			if x == cx && y == cy && !focused {
				strokeRect(gtx, alpha(ColText, 0xa0), image.Rect(x*cell.X, y*cell.Y, (x+1)*cell.X, (y+1)*cell.Y), 0, float32(max(1, dp(gtx, 1))))
			}
			bold := st.Attrs&uv.AttrBold != 0
			under, strike := st.Underline != uv.UnderlineNone, st.Attrs&uv.AttrStrikethrough != 0
			if r == nil || r.fg != fg || r.bold != bold || r.under != under || r.strike != strike || wide > 1 {
				flush()
				r = &run{x0: x, fg: fg, bold: bold, under: under, strike: strike}
			}
			r.text.WriteString(content)
			if wide > 1 {
				// A wide character gets a run of its own, so the cells after
				// it stay on the grid.
				flush()
			}
		}
		flush()
	}
	if v.Exited {
		m := op.Record(gtx.Ops)
		d := th.Small(gtx, ColFaint, "The shell ended. Close the tab, or open a new one with +.")
		call := m.Stop()
		off := op.Offset(image.Pt(0, size.Y-d.Size.Y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
	return D{Size: size}
}

// termInput takes the drawer's keys, typing, pastes and wheel.
func (th *Theme) termInput(gtx C, v *TermView, tag event.Tag, cell image.Point) {
	mods := key.ModCtrl | key.ModShift | key.ModAlt
	mac := runtime.GOOS == "darwin"
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: tag},
			key.Filter{Focus: tag, Name: "", Optional: mods},
			key.Filter{Focus: tag, Name: key.NameTab, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: "V", Required: key.ModShortcut},
			transfer.TargetFilter{Target: tag, Type: "application/text"},
			key.Filter{Focus: tag, Name: "C", Required: key.ModShortcut, Optional: key.ModShift},
			pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Scroll, ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}},
		)
		if !ok {
			return
		}
		switch e := ev.(type) {
		case pointer.Event:
			// A press starts a selection, dragging grows it; the cell is
			// counted from the top of the scrollback, so it stays put as
			// output scrolls.
			col := max(0, int(e.Position.X)/cell.X)
			row := th.termRow(v.Screen, int(e.Position.Y)/cell.Y)
			sel := &th.in.termSel
			switch e.Kind {
			case pointer.Press:
				gtx.Execute(key.FocusCmd{Tag: tag})
				*sel = termSel{tab: v.Active, from: [2]int{row, col}, to: [2]int{row, col}, dragging: true}
				continue
			case pointer.Drag:
				if sel.dragging {
					sel.to = [2]int{row, col}
				}
				continue
			case pointer.Release:
				sel.dragging = false
				if sel.from == sel.to {
					*sel = termSel{}
				}
				continue
			}
			th.in.termWheel += e.Scroll.Y
			lines := int(th.in.termWheel / float32(cell.Y))
			th.in.termWheel -= float32(lines * cell.Y)
			th.in.termScroll = max(0, th.in.termScroll-lines)
		case key.EditEvent:
			th.in.termScroll = 0
			if v.Text != nil && e.Text != "" {
				v.Text(e.Text)
			}
			// Keep the input method's idea of the text empty: everything
			// typed has gone to the shell.
			gtx.Execute(key.SnippetCmd{Tag: tag})
			gtx.Execute(key.SelectionCmd{Tag: tag})
		case transfer.DataEvent:
			b, _ := io.ReadAll(e.Open())
			if v.Paste != nil && len(b) > 0 {
				v.Paste(string(b))
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			if e.Name == "V" && e.Modifiers.Contain(key.ModShortcut) {
				gtx.Execute(clipboard.ReadCmd{Tag: tag})
				continue
			}
			if e.Name == "C" && e.Modifiers.Contain(key.ModShortcut) {
				// Copy what's selected. Without a selection, Ctrl+C is the
				// shell's interrupt; ⌘C on a Mac does nothing.
				if text := th.termSelected(v.Screen); text != "" {
					gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
					th.in.termSel = termSel{}
					continue
				}
				if mac {
					continue
				}
			}
			k, ok := termKey(e, mac)
			if ok && v.Key != nil {
				th.in.termScroll, th.in.termSel = 0, termSel{}
				v.Key(k)
			}
		}
	}
}

// termKey turns a key press into the emulator's: named keys, and letters
// with Ctrl (or Alt, which on a Mac types its own characters instead).
func termKey(e key.Event, mac bool) (uv.KeyPressEvent, bool) {
	var mod uv.KeyMod
	if e.Modifiers.Contain(key.ModCtrl) {
		mod |= uv.ModCtrl
	}
	if e.Modifiers.Contain(key.ModAlt) {
		mod |= uv.ModAlt
	}
	if e.Modifiers.Contain(key.ModShift) {
		mod |= uv.ModShift
	}
	named := map[key.Name]rune{
		key.NameReturn: uv.KeyEnter, key.NameEnter: uv.KeyEnter, key.NameDeleteBackward: uv.KeyBackspace,
		key.NameDeleteForward: uv.KeyDelete, key.NameEscape: uv.KeyEscape, key.NameTab: uv.KeyTab,
		key.NameUpArrow: uv.KeyUp, key.NameDownArrow: uv.KeyDown, key.NameLeftArrow: uv.KeyLeft, key.NameRightArrow: uv.KeyRight,
		key.NameHome: uv.KeyHome, key.NameEnd: uv.KeyEnd, key.NamePageUp: uv.KeyPgUp, key.NamePageDown: uv.KeyPgDown,
		key.NameF1: uv.KeyF1, key.NameF2: uv.KeyF2, key.NameF3: uv.KeyF3, key.NameF4: uv.KeyF4, key.NameF5: uv.KeyF5, key.NameF6: uv.KeyF6,
		key.NameF7: uv.KeyF7, key.NameF8: uv.KeyF8, key.NameF9: uv.KeyF9, key.NameF10: uv.KeyF10, key.NameF11: uv.KeyF11, key.NameF12: uv.KeyF12,
	}
	if c, ok := named[e.Name]; ok {
		return uv.KeyPressEvent{Code: c, Mod: mod}, true
	}
	ctrl := mod&uv.ModCtrl != 0
	alt := mod&uv.ModAlt != 0 && !mac
	if !ctrl && !alt {
		return uv.KeyPressEvent{}, false // typing arrives as text
	}
	if e.Name == key.NameSpace {
		return uv.KeyPressEvent{Code: uv.KeySpace, Mod: mod}, true
	}
	rs := []rune(string(e.Name))
	if len(rs) != 1 {
		return uv.KeyPressEvent{}, false
	}
	c := []rune(strings.ToLower(string(rs[0])))[0]
	if !mac || ctrl {
		mod &^= uv.ModShift
	}
	k := uv.KeyPressEvent{Code: c, Mod: mod}
	if alt && !ctrl {
		k.Text = string(c)
	}
	return k, true
}

// termSel is a selection in the drawer, between two cells counted as
// (row from the top of the scrollback, column).
type termSel struct {
	tab      int
	from, to [2]int
	dragging bool
}

// ordered is the selection's ends in reading order.
func (s termSel) ordered() ([2]int, [2]int) {
	a, b := s.from, s.to
	if b[0] < a[0] || b[0] == a[0] && b[1] < a[1] {
		a, b = b, a
	}
	return a, b
}

func (s termSel) has(tab, row, col int) bool {
	if s.from == s.to || s.tab != tab {
		return false
	}
	a, b := s.ordered()
	switch {
	case row < a[0] || row > b[0]:
		return false
	case row == a[0] && col < a[1]:
		return false
	case row == b[0] && col > b[1]:
		return false
	}
	return true
}

// termRow is the scrollback-counted row at screen row y.
func (th *Theme) termRow(scr TermScreen, y int) int {
	back := 0
	if !scr.IsAltScreen() {
		back = th.in.termScroll
	}
	return scr.ScrollbackLen() - back + max(0, y)
}

// termSelected is the selected text, a line per row with trailing blanks
// cut.
func (th *Theme) termSelected(scr TermScreen) string {
	s := th.in.termSel
	if s.from == s.to {
		return ""
	}
	a, b := s.ordered()
	sb := scr.ScrollbackLen()
	var lines []string
	for row := a[0]; row <= b[0]; row++ {
		x0, x1 := 0, scr.Width()-1
		if row == a[0] {
			x0 = a[1]
		}
		if row == b[0] {
			x1 = min(x1, b[1])
		}
		var line strings.Builder
		for x := x0; x <= x1; x++ {
			var c *uv.Cell
			if row < sb {
				c = scr.ScrollbackCellAt(x, row)
			} else if row-sb < scr.Height() {
				c = scr.CellAt(x, row-sb)
			}
			switch {
			case c == nil:
				line.WriteByte(' ')
			case c.Content != "":
				line.WriteString(c.Content)
			case c.Width != 0:
				line.WriteByte(' ')
			}
		}
		lines = append(lines, strings.TrimRight(line.String(), " "))
	}
	return strings.Join(lines, "\n")
}
