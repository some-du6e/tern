package ui

import (
	"image"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// A sidebar row's menu, as T3's browser build draws it (the DOM menu of
// contextMenuFallback.ts, docs/ui/04-sidebar.md §3.16): the dropdown-glass
// surface with no shadow, 28dp rows with their icons stroked at 2, and
// submenus beside their rows. The glass is solid, as T3 paints it without
// backdrop blur.

// menuState is the open context menu: the submenu open at each level, by
// item index, and the highlighted item at each level, -1 for none.
type menuState struct {
	open  bool
	path  []int
	hi    []int
	tag   int // the menu's keyboard focus
	hover map[string]*gesture.Hover
}

// contextMenu draws s.Menu at the right-click (or the bottom left of the
// row the context menu key was pressed on), kept 4dp inside the window.
// Opened from the keyboard it starts on its first item.
func (th *Theme) contextMenu(gtx C, win image.Point, s *Scene) {
	ms := &th.in.menu
	if !ms.open {
		*ms = menuState{open: true, hi: []int{-1}, hover: ms.hover}
		if s.Menu.Keyboard {
			ms.hi[0] = nextEnabled(s.Menu.Items, -1, 1)
		}
		gtx.Execute(key.FocusCmd{Tag: &ms.tag})
	}
	th.menuKeys(gtx, s.Menu.Items)
	items := s.Menu.Items
	var from image.Rectangle // the row the submenu opens from
	at := th.in.ctxAt
	if !s.Live {
		at = s.Menu.At
	}
	for level := 0; level <= len(ms.path); level++ {
		for len(ms.hi) <= level {
			ms.hi = append(ms.hi, -1)
		}
		m := op.Record(gtx.Ops)
		d, rows := th.menuLevel(gtx, items, level)
		call := m.Stop()
		if level > 0 {
			// Beside its row, 4dp off; on the other side when it won't fit.
			at = image.Pt(from.Max.X+dp(gtx, 4), from.Min.Y)
			if at.X+d.Size.X > win.X {
				at.X = from.Min.X - d.Size.X - dp(gtx, 4)
			}
		}
		edge := dp(gtx, 4)
		at.X = max(edge, min(at.X, win.X-d.Size.X-edge))
		at.Y = max(edge, min(at.Y, win.Y-d.Size.Y-edge))
		m2 := op.Record(gtx.Ops)
		op.Offset(at).Add(gtx.Ops)
		call.Add(gtx.Ops)
		op.Defer(gtx.Ops, m2.Stop())
		if level == len(ms.path) {
			break
		}
		i := ms.path[level]
		if i < 0 || i >= len(items) || len(items[i].Children) == 0 {
			ms.path = ms.path[:level]
			break
		}
		from = rows[i].Add(at)
		items = items[i].Children
	}
}

// nextEnabled is the next item from i in dir that can be picked, wrapping
// around; -1 if there's none.
func nextEnabled(items []MenuItem, i, dir int) int {
	n := len(items)
	if i < 0 && dir < 0 {
		i = n
	}
	for k := 1; k <= n; k++ {
		j := ((i+dir*k)%n + n) % n
		if !items[j].Off {
			return j
		}
	}
	return -1
}

// menuKeys is the menu's keyboard: the arrows move through the items that
// can be picked, wrapping; Home and End go to the ends; Right and Enter
// open a submenu on its first item, Left and Escape close it; Escape on
// the first menu, or Tab, closes the lot; Enter and Space pick the item.
// They act on the deepest menu with a highlight.
func (th *Theme) menuKeys(gtx C, root []MenuItem) {
	ms := &th.in.menu
	// The items can change under an open menu: keep the path to submenus
	// that are still there.
	items := root
	for l, i := range ms.path {
		if i < 0 || i >= len(items) || len(items[i].Children) == 0 {
			ms.path = ms.path[:l]
			break
		}
		items = items[i].Children
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &ms.tag},
			key.Filter{Focus: &ms.tag, Name: key.NameDownArrow}, key.Filter{Focus: &ms.tag, Name: key.NameUpArrow},
			key.Filter{Focus: &ms.tag, Name: key.NameHome}, key.Filter{Focus: &ms.tag, Name: key.NameEnd},
			key.Filter{Focus: &ms.tag, Name: key.NameRightArrow}, key.Filter{Focus: &ms.tag, Name: key.NameLeftArrow},
			key.Filter{Focus: &ms.tag, Name: key.NameEscape}, key.Filter{Focus: &ms.tag, Name: key.NameTab},
			key.Filter{Focus: &ms.tag, Name: key.NameReturn}, key.Filter{Focus: &ms.tag, Name: key.NameEnter},
			key.Filter{Focus: &ms.tag, Name: key.NameSpace},
		)
		if !ok {
			return
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		lvl := 0
		for l := len(ms.path); l >= 0; l-- {
			if l < len(ms.hi) && ms.hi[l] >= 0 {
				lvl = l
				break
			}
		}
		items := root
		for _, i := range ms.path[:min(lvl, len(ms.path))] {
			items = items[i].Children
		}
		hi := ms.hi[lvl]
		if hi >= len(items) {
			hi = -1
		}
		openSub := func() {
			ms.path = append(ms.path[:lvl], hi)
			for len(ms.hi) <= lvl+1 {
				ms.hi = append(ms.hi, -1)
			}
			ms.hi[lvl+1] = nextEnabled(items[hi].Children, -1, 1)
		}
		switch e.Name {
		case key.NameDownArrow, key.NameUpArrow:
			dir := 1
			if e.Name == key.NameUpArrow {
				dir = -1
			}
			ms.hi[lvl], ms.path = nextEnabled(items, hi, dir), ms.path[:min(lvl, len(ms.path))]
		case key.NameHome:
			ms.hi[lvl] = nextEnabled(items, -1, 1)
		case key.NameEnd:
			ms.hi[lvl] = nextEnabled(items, 0, -1)
		case key.NameRightArrow:
			if hi >= 0 && len(items[hi].Children) > 0 {
				openSub()
			}
		case key.NameLeftArrow, key.NameEscape:
			if lvl > 0 {
				ms.hi[lvl], ms.path = -1, ms.path[:lvl-1]
			} else if e.Name == key.NameEscape {
				th.push(Action{ID: "menus.close", Kind: "click"})
			}
		case key.NameTab:
			th.push(Action{ID: "menus.close", Kind: "click"})
		default: // Enter, Space
			switch {
			case hi < 0 || items[hi].Off:
			case len(items[hi].Children) > 0:
				openSub()
			default:
				th.push(Action{ID: "ctx:" + items[hi].ID, Kind: "click"})
			}
		}
	}
}

// menuLevel lays out one menu, and each row's box in it: at least 128dp
// and at most 384 wide, radius 8, the popover inside a 1dp border at the
// foreground's 10%, padded 4; rows 28dp, padded 4 by 8, 14/20 in the
// foreground with 16dp icons 8dp before the label, a detail 8 + 16dp after
// it in 12px muted, a chevron on a submenu's row; 1dp separators 4 by 8 in.
func (th *Theme) menuLevel(gtx C, items []MenuItem, level int) (D, []image.Rectangle) {
	ms := &th.in.menu
	px := max(1, dp(gtx, 1))
	bg := P.Popover
	padX, gap, icon := dp(gtx, 8), dp(gtx, 8), dp(gtx, 16)
	label := Txt{Size: 14, LineH: 20.0 / 14, MaxLines: 1}
	detail := Txt{Size: 12, LineH: 20.0 / 12, MaxLines: 1}
	inner := 0
	for _, it := range items {
		w := 2*padX + th.textWidth(gtx, label, it.Label)
		if it.Icon != "" || it.Radio {
			w += icon + gap
		}
		if it.Detail != "" {
			w += gap + dp(gtx, 16) + th.textWidth(gtx, detail, it.Detail)
		}
		if len(it.Children) > 0 {
			w += gap + icon - dp(gtx, 2)
		}
		inner = max(inner, w)
	}
	edge := dp(gtx, 4) + px
	width := max(dp(gtx, 128), min(dp(gtx, 384), inner+2*edge))
	rowH, sepH := dp(gtx, 28), dp(gtx, 9)
	height := 2 * edge
	for i, it := range items {
		if it.Sep && i > 0 {
			height += sepH
		}
		height += rowH
	}
	size := image.Pt(width, height)
	r := image.Rectangle{Max: size}
	fill(gtx, bg, r, dp(gtx, 8))
	strokeRect(gtx, over(alpha(P.Foreground, 0x1a), bg), r, dp(gtx, 8), float32(px))
	if level == 0 {
		st := clip.Rect(r).Push(gtx.Ops)
		event.Op(gtx.Ops, &ms.tag)
		st.Pop()
	}
	rows := make([]image.Rectangle, len(items))
	y := edge
	for i, it := range items {
		if it.Sep && i > 0 {
			hline(gtx, over(P.ContrastBorder, bg), y+dp(gtx, 4), edge+dp(gtx, 4), width-edge-dp(gtx, 4))
			y += sepH
		}
		rows[i] = image.Rect(edge, y, width-edge, y+rowH)
		func() {
			defer op.Offset(rows[i].Min).Push(gtx.Ops).Pop()
			th.menuRow(gtx, it, level, i, rows[i].Size())
		}()
		y += rowH
	}
	return D{Size: size}, rows
}

// menuRow is one row of a menu: highlighted under the pointer or the
// keyboard with accent (a destructive row with its colour at 10%), muted
// at 64% when it can't be picked. Hovering a submenu's row opens it;
// hovering another closes the ones past it.
func (th *Theme) menuRow(gtx C, it MenuItem, level, i int, size image.Point) {
	ms := &th.in.menu
	sub := len(it.Children) > 0
	hovered := false
	switch {
	case it.Off:
	case sub:
		if ms.hover == nil {
			ms.hover = map[string]*gesture.Hover{}
		}
		k := it.ID
		h := ms.hover[k]
		if h == nil {
			h = new(gesture.Hover)
			ms.hover[k] = h
		}
		hovered = h.Update(gtx.Source)
		defer func() {
			st := clip.Rect{Max: size}.Push(gtx.Ops)
			h.Add(gtx.Ops)
			st.Pop()
		}()
	default:
		c, _ := th.events(gtx, "ctx:"+it.ID)
		hovered = c.Hovered()
		defer th.hit(gtx, "ctx:"+it.ID, size)
	}
	if hovered && (ms.hi[level] != i || (sub && (len(ms.path) <= level || ms.path[level] != i))) {
		ms.hi[level] = i
		ms.path = ms.path[:min(level, len(ms.path))]
		if sub {
			ms.path = append(ms.path, i)
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	lit := !it.Off && ms.hi[level] == i
	danger := it.Danger && !sub
	bg, text, iconCol := P.Popover, P.ContrastForeground, P.MutedForeground
	switch {
	case it.Off:
		text, iconCol = over(alpha(P.ContrastMutedForeground, 0xa3), bg), over(alpha(P.MutedForeground, 0xa3), bg)
	case danger && lit:
		bg = over(alpha(P.Destructive, 0x1a), P.Popover)
		text, iconCol = P.DestructiveForeground, P.DestructiveForeground
	case danger:
		text, iconCol = P.DestructiveForeground, P.DestructiveForeground
	case lit:
		bg = over(P.Accent, P.Popover)
		text = P.ContrastAccentForeground
	}
	if bg != P.Popover {
		fill(gtx, bg, image.Rectangle{Max: size}, dp(gtx, 4))
	}
	muted := P.ContrastMutedForeground
	if it.Off {
		muted = over(alpha(muted, 0xa3), P.Popover)
	}
	defer op.Offset(image.Pt(dp(gtx, 8), 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(size.X-2*dp(gtx, 8), size.Y))
	var kids []layout.FlexChild
	switch {
	case it.Radio && it.Checked:
		kids = append(kids, rigid(lucideStroke("check", iconCol, 16, 2)))
	case it.Radio:
		kids = append(kids, rigid(Spacer(16, 16)))
	case it.Icon != "":
		kids = append(kids, rigid(lucideStroke(it.Icon, iconCol, 16, 2)))
	}
	kids = append(kids, flexed(func(gtx C) D {
		return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: text, MaxLines: 1}, it.Label)
	}))
	if it.Detail != "" {
		kids = append(kids, rigid(func(gtx C) D {
			return pad(0, 0, 0, 16).Layout(gtx, func(gtx C) D {
				return th.Text(gtx, Txt{Size: 12, LineH: 20.0 / 12, Color: muted, MaxLines: 1}, it.Detail)
			})
		}))
	}
	if sub {
		chev := over(alpha(P.MutedForeground, 0xcc), bg)
		kids = append(kids, rigid(func(gtx C) D {
			return pad(0, -2, 0, 0).Layout(gtx, lucideStroke("chevron-right", chev, 16, 2))
		}))
	}
	RowGap(g, 8, kids...)
}
