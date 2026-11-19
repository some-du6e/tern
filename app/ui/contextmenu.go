package ui

import (
	"image"
	"slices"
	"strconv"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

// A thread's actions menu, as T3's browser build draws it (the DOM menu
// of contextMenuFallback.ts, docs/ui/05-header.md §2.3, §3.3, §5.2;
// 04-sidebar.md §3.16): the dropdown-glass surface with no shadow, 28dp
// rows with T3's own copies of their icons stroked at 2, and submenus
// beside their rows. The glass is solid, as T3 paints it without backdrop
// blur. The header's title and right-click open it, and a sidebar row's.

// menuState is the open context menu: the item each open submenu hangs
// from, level by level; where the keyboard's focus is, an item of a level
// or that level's menu itself (-1), as T3 moves the DOM focus between its
// buttons; and what the pointer was over last frame, since T3 acts when
// the pointer comes onto a row or into a menu (mouseenter), not while it
// rests there.
type menuState struct {
	open        bool
	path        []int
	level, item int
	under       menuSpot // the row under the pointer, level -1 for none
	in          int      // the menu under the pointer, -1 for none
	tag         int      // the keyboard's focus
	outside     int      // the window's presses, which close it
	subs        map[string]*gesture.Click
}

// menuSpot is a row of the open menus: its level and its index there.
type menuSpot struct{ level, item int }

// items is the menu at level, down the open submenus from root.
func (ms *menuState) items(root []MenuItem, level int) []MenuItem {
	items := root
	for _, i := range ms.path[:level] {
		items = items[i].Children
	}
	return items
}

// prune keeps the open submenus and the focus to what's still in root:
// the items can change under an open menu.
func (ms *menuState) prune(root []MenuItem) {
	items := root
	for l, i := range ms.path {
		if i < 0 || i >= len(items) || len(items[i].Children) == 0 {
			ms.path = ms.path[:l]
			break
		}
		items = items[i].Children
	}
	if ms.level > len(ms.path) {
		ms.level, ms.item = len(ms.path), -1
	}
	if items := ms.items(root, ms.level); ms.item >= len(items) || ms.item >= 0 && items[ms.item].Off {
		ms.item = -1
	}
}

// openSub opens the focused item's submenu and moves the focus onto its
// first item that can be picked, as a click or Enter on its row does.
func (ms *menuState) openSub(items []MenuItem) {
	ms.path = append(ms.path[:ms.level], ms.item)
	if first := nextEnabled(items[ms.item].Children, -1, 1); first >= 0 {
		ms.level, ms.item = ms.level+1, first
	}
}

// contextMenu draws s.Menu at the right-click (or under the header's
// title) and its open submenus, each 4dp right of its row. One frame after
// a menu opens T3 moves it to stay 4dp inside the window
// (clampMenuPosition), a submenu from that spot right of its row, so one
// that won't fit there ends up against the window's edge rather than on
// the row's other side.
func (th *Theme) contextMenu(gtx C, win image.Point, s *Scene) {
	ms := &th.in.menu
	if !ms.open {
		*ms = menuState{open: true, item: -1, under: menuSpot{-1, -1}, in: -1, subs: ms.subs}
		// Opened from the keyboard it starts on its first item; from the
		// pointer, on the menu itself (contextMenuFallback.ts:611–618).
		if s.Menu.Keyboard || th.in.keyFocus {
			ms.item = nextEnabled(s.Menu.Items, -1, 1)
		}
		gtx.Execute(key.FocusCmd{Tag: &ms.tag})
	}
	ms.prune(s.Menu.Items)
	th.menuKeys(gtx, s.Menu.Items)
	at := th.in.ctxAt
	if !s.Live {
		at = image.Pt(dp(gtx, float32(s.Menu.At.X)), dp(gtx, float32(s.Menu.At.Y)))
	}
	under, in := menuSpot{-1, -1}, -1
	edge := dp(gtx, 4)
	for level := 0; ; level++ {
		items := ms.items(s.Menu.Items, level)
		m := op.Record(gtx.Ops)
		d, rows, hovered, inside := th.menuLevel(gtx, items, level)
		call := m.Stop()
		if hovered >= 0 {
			under = menuSpot{level, hovered}
		}
		if inside {
			in = level
		}
		at.X = max(edge, min(at.X, win.X-d.Size.X-edge))
		at.Y = max(edge, min(at.Y, win.Y-d.Size.Y-edge))
		m2 := op.Record(gtx.Ops)
		op.Offset(at).Add(gtx.Ops)
		call.Add(gtx.Ops)
		op.Defer(gtx.Ops, m2.Stop())
		if level == len(ms.path) {
			break
		}
		row := rows[ms.path[level]].Add(at)
		at = image.Pt(row.Max.X+edge, row.Min.Y)
	}
	th.menuPointer(gtx, s.Menu.Items, under, in)
}

// menuPointer follows the pointer as T3's menu does: coming into a menu
// closes the submenus past it; coming onto a row focuses it, and opens its
// submenu or closes the ones past it. It acts on what changed since the
// last frame, so the keyboard can move the focus off a row the pointer
// rests on.
func (th *Theme) menuPointer(gtx C, root []MenuItem, under menuSpot, in int) {
	ms := &th.in.menu
	changed := false
	if in != ms.in {
		ms.in = in
		if in >= 0 && len(ms.path) > in {
			ms.path, changed = ms.path[:in], true
		}
	}
	if under != ms.under {
		ms.under, changed = under, true
		if under.level >= 0 && under.level <= len(ms.path) {
			items := ms.items(root, under.level)
			ms.level, ms.item, ms.path = under.level, under.item, ms.path[:under.level]
			if len(items[under.item].Children) > 0 {
				ms.path = append(ms.path, under.item)
			}
		}
	}
	if changed {
		gtx.Execute(op.InvalidateCmd{})
	}
}

// menuOutside closes the open context menu on a press anywhere outside it
// and lets the press go on to what's under it, as T3's pointerdown watch
// does: a second click on the header's title closes the menu and opens it
// again. Call it before the frame's other input; watchOutside, after
// everything else is drawn, takes the presses.
func (th *Theme) menuOutside(gtx C) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &th.in.menu.outside, Kinds: pointer.Press})
		if !ok {
			return
		}
		if _, ok := ev.(pointer.Event); ok {
			th.push(Action{ID: "menus.close", Kind: "click"})
		}
	}
}

// watchOutside takes the presses that reach the window past an open
// context menu, passing them on.
func (th *Theme) watchOutside(gtx C, size image.Point) {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, &th.in.menu.outside)
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

// menuKeys is the menu's keyboard (contextMenuFallback.ts:284–343), on
// the menu that has the focus: Down and Up move it through the items that
// can be picked, wrapping, Home and End to the ends; Right opens the
// focused item's submenu on its first item, and Enter and Space pick the
// item or open it the same way; Left and Escape in a submenu close it,
// back on its row; Escape on the first menu, or Tab, closes them all.
// Moving off the row of an open submenu closes it. Alt, Ctrl and Cmd
// chords aren't the menu's.
func (th *Theme) menuKeys(gtx C, root []MenuItem) {
	ms := &th.in.menu
	f := func(name key.Name) key.Filter { return key.Filter{Focus: &ms.tag, Name: name, Optional: key.ModShift} }
	for {
		ev, ok := gtx.Event(key.FocusFilter{Target: &ms.tag},
			f(key.NameDownArrow), f(key.NameUpArrow), f(key.NameHome), f(key.NameEnd),
			f(key.NameRightArrow), f(key.NameLeftArrow), f(key.NameEscape), f(key.NameTab),
			f(key.NameReturn), f(key.NameEnter), f(key.NameSpace))
		if !ok {
			return
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		items := ms.items(root, ms.level)
		var on []int
		for i, it := range items {
			if !it.Off {
				on = append(on, i)
			}
		}
		at, target := slices.Index(on, ms.item), -1
		switch n := len(on); {
		case n == 0:
		case e.Name == key.NameDownArrow:
			target = on[(at+1)%n]
		case e.Name == key.NameUpArrow && at < 0:
			target = on[n-1]
		case e.Name == key.NameUpArrow:
			target = on[(at-1+n)%n]
		case e.Name == key.NameHome:
			target = on[0]
		case e.Name == key.NameEnd:
			target = on[n-1]
		}
		switch e.Name {
		case key.NameRightArrow:
			if ms.item >= 0 && len(items[ms.item].Children) > 0 {
				ms.openSub(items)
			}
		case key.NameReturn, key.NameEnter, key.NameSpace:
			switch {
			case ms.item < 0:
			case len(items[ms.item].Children) > 0:
				ms.openSub(items)
			default:
				th.push(Action{ID: "ctx:" + items[ms.item].ID, Kind: "click"})
			}
		case key.NameLeftArrow, key.NameEscape:
			switch {
			case ms.level > 0:
				ms.level--
				ms.item, ms.path = ms.path[ms.level], ms.path[:ms.level]
			case e.Name == key.NameEscape:
				th.push(Action{ID: "menus.escape", Kind: "click"})
			}
		case key.NameTab:
			th.push(Action{ID: "menus.escape", Kind: "click"})
		}
		if target >= 0 {
			if len(ms.path) > ms.level && ms.path[ms.level] != target {
				ms.path = ms.path[:ms.level]
			}
			ms.item = target
		}
	}
}

// menuLevel lays out one menu, and each row's box in it: at least 128dp
// and at most 384 wide, radius 8, the dropdown glass as it paints over the
// window inside a 1dp border at the foreground's 10%, padded 4; rows 28dp,
// padded 4 by 8, 14/20 in the foreground with 16dp icons 8dp before the
// label, a detail 8 + 16dp after it in 12px muted, a chevron on a
// submenu's row; 1dp separators 4 by 8 in. It reports the row under the
// pointer (-1 for none) and whether the pointer is in the menu, which
// takes the presses between its rows so they don't close it.
func (th *Theme) menuLevel(gtx C, items []MenuItem, level int) (D, []image.Rectangle, int, bool) {
	ms := &th.in.menu
	px := max(1, dp(gtx, 1))
	bg := cardFill()
	padX, gap, icon := dp(gtx, 8), dp(gtx, 8), dp(gtx, 16)
	label := Txt{Size: 14, LineH: 20.0 / 14, MaxLines: 1}
	detail := Txt{Size: 12, LineH: 20.0 / 12, MaxLines: 1}
	inner := 0
	for _, it := range items {
		w := 2*padX + th.textW(gtx, label, it.Label)
		if it.Icon != "" || it.Radio {
			w += icon + gap
		}
		if it.Detail != "" {
			w += gap + dp(gtx, 16) + th.textW(gtx, detail, it.Detail)
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
	strokeRect(gtx, over(alpha(P.ContrastForeground, 0x1a), bg), r, dp(gtx, 8), float32(px))
	id := "ctx.menu:" + strconv.Itoa(level)
	inside := th.hoverState(gtx, id)
	st := clip.Rect(r).Push(gtx.Ops)
	event.Op(gtx.Ops, th.hoverTag(id))
	if level == 0 {
		event.Op(gtx.Ops, &ms.tag)
	}
	rows := make([]image.Rectangle, len(items))
	hovered := -1
	y := edge
	for i, it := range items {
		if it.Sep && i > 0 {
			hline(gtx, over(P.ContrastBorder, bg), y+dp(gtx, 4), edge+dp(gtx, 8), width-edge-dp(gtx, 8))
			y += sepH
		}
		rows[i] = image.Rect(edge, y, width-edge, y+rowH)
		func() {
			defer op.Offset(rows[i].Min).Push(gtx.Ops).Pop()
			if th.menuRow(gtx, items, level, i, rows[i].Size()) {
				hovered = i
			}
		}()
		y += rowH
	}
	st.Pop()
	return D{Size: size}, rows, hovered, inside
}

// menuRow is row i of the menu at level: highlighted while it has the
// focus or the pointer with accent (a destructive row with its colour at
// 10%), muted at 64% when it can't be picked, which also takes no pointer.
// A click picks it, or opens its submenu on the submenu's first item. It
// reports whether the pointer is over it.
func (th *Theme) menuRow(gtx C, items []MenuItem, level, i int, size image.Point) bool {
	ms := &th.in.menu
	it := items[i]
	sub := len(it.Children) > 0
	hovered := false
	switch {
	case it.Off:
	case sub:
		if ms.subs == nil {
			ms.subs = map[string]*gesture.Click{}
		}
		c := ms.subs[it.ID]
		if c == nil {
			c = new(gesture.Click)
			ms.subs[it.ID] = c
		}
		for {
			e, ok := c.Update(gtx.Source)
			if !ok {
				break
			}
			if e.Kind == gesture.KindClick {
				ms.level, ms.item = level, i
				ms.openSub(items)
				gtx.Execute(op.InvalidateCmd{})
			}
		}
		hovered = c.Hovered()
		defer func() {
			st := clip.Rect{Max: size}.Push(gtx.Ops)
			c.Add(gtx.Ops)
			st.Pop()
		}()
	default:
		// T3's rows keep the default cursor.
		c, _ := th.events(gtx, "ctx:"+it.ID)
		hovered = c.Hovered()
		defer th.hover(gtx, "ctx:"+it.ID, size)
	}
	lit := !it.Off && (hovered || ms.level == level && ms.item == i)
	danger := it.Danger && !sub
	glass := cardFill()
	bg, text, iconCol := glass, P.ContrastForeground, P.MutedForeground
	switch {
	case it.Off:
		text, iconCol = over(alpha(P.ContrastMutedForeground, 0xa3), bg), over(alpha(P.MutedForeground, 0xa3), bg)
	case danger && lit:
		bg = over(alpha(P.Destructive, 0x1a), glass)
		text, iconCol = P.DestructiveForeground, P.DestructiveForeground
	case danger:
		text, iconCol = P.DestructiveForeground, P.DestructiveForeground
	case lit:
		bg = over(P.Accent, glass)
		text = P.ContrastAccentForeground
	}
	if bg != glass {
		fill(gtx, bg, image.Rectangle{Max: size}, dp(gtx, 4))
	}
	muted := P.ContrastMutedForeground
	if it.Off {
		muted = over(alpha(muted, 0xa3), glass)
	}
	defer op.Offset(image.Pt(dp(gtx, 8), 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(size.X-2*dp(gtx, 8), size.Y))
	var kids []layout.FlexChild
	switch {
	case it.Radio && it.Checked:
		kids = append(kids, rigid(menuIcon("check", iconCol, 16)))
	case it.Radio:
		kids = append(kids, rigid(Spacer(16, 16)))
	case it.Icon != "":
		kids = append(kids, rigid(menuIcon(it.Icon, iconCol, 16)))
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
			return pad(0, -2, 0, 0).Layout(gtx, menuIcon("chevron-right", chev, 16))
		}))
	}
	RowGap(g, 8, kids...)
	return hovered
}
