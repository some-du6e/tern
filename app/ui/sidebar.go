package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
)

// The sidebar below its header row, ported from T3 Code
// (docs/ui/04-sidebar.md): the search and filter row, the thread list with
// its cards, slim rows and shelves, the footer, and the hover card, menus
// and keyboard that go with them. Lengths are dp. Colours are the sidebar's
// scoped palette (SP), composited onto the surface under them with over.

const (
	sideInset  = 8  // --sidebar-content-inset
	searchRowH = 48 // the search and filter row: 32 in 8 of padding
	footerH    = 44
	cardH      = 78 // a card with its meta line
	cardFlatH  = 60 // a card without one
	slimH      = 36
	excerptH   = 44 // a search result with its matching message
	shelfH     = 32
	// The list's tooltips (and hover cards) open after a 150ms rest, and
	// at once within 400ms of the last one closing (its TooltipProvider).
	listTipDelay   = 150 * time.Millisecond
	listTipInstant = 400 * time.Millisecond
)

// sidebar is the sidebar below its header row: the search and filter row,
// the list and the footer. sidebarFrame paints the surface under it.
func (th *Theme) sidebar(gtx C, s *Scene) D {
	size := gtx.Constraints.Max
	th.in.undoClaim = s.Live && s.Sidebar.UndoClaim
	th.in.jumping = th.jumpHints(gtx, s)
	at := func(y, h int, draw func(gtx C)) {
		defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X, h))
		draw(g)
	}
	top, foot := dp(gtx, searchRowH), dp(gtx, footerH)
	at(top, max(0, size.Y-top-foot), func(gtx C) { th.threadList(gtx, s, dp(gtx, topBarH)+top) })
	at(size.Y-foot, foot, func(gtx C) { th.sidebarFooter(gtx, s) })
	at(0, top, func(gtx C) { th.searchRow(gtx, s) })
	return D{Size: size}
}

// jumpHints reports whether the rows show their jump keys: once the jump
// modifier has been held on its own for 200ms, and not for the terminal or
// the model picker, which have keys of their own
// (useThreadJumpHintsVisible).
func (th *Theme) jumpHints(gtx C, s *Scene) bool {
	if !s.Live {
		return s.Sidebar.JumpHints
	}
	if th.in.heldMods != key.ModShortcut || gtx.Focused(&th.in.term) || s.Picker != nil {
		return false
	}
	if left := 200*time.Millisecond - gtx.Now.Sub(th.in.heldSince); left > 0 {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
		return false
	}
	return true
}

// jumpKey is a row's jump hint: T3's Kbd with mod+n on the row's surface.
func (th *Theme) jumpKey(r *ThreadRow, in rowInk) W {
	return th.T3Kbd(Shortcut(strconv.Itoa(r.Jump)), in.bg)
}

// ---------- the search and filter row ----------

// searchRow is the search field and the header buttons (§2.2, §3.1): the
// field takes what the 28dp buttons, 4dp clear of it, leave. tern has no
// Add project, so the filter and New thread are the buttons.
func (th *Theme) searchRow(gtx C, s *Scene) {
	sb := &s.Sidebar
	w := gtx.Constraints.Max.X
	inset, b, h := dp(gtx, sideInset), dp(gtx, controlSize), dp(gtx, 32)
	n := 1
	if sb.HasProjects {
		n = 2
	}
	x := w - inset - n*b
	func() {
		defer op.Offset(image.Pt(inset, inset)).Push(gtx.Ops).Pop()
		th.searchField(gtx, s, image.Pt(x-dp(gtx, 4)-inset, h))
	}()
	y := inset + (h-b)/2
	if sb.HasProjects {
		func() {
			defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
			th.filterButton(gtx, s)
		}()
		x += b
	}
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	th.headerButton(gtx, liveID(s, "new"), "New thread ("+Shortcut("N")+")", sb.NoNew, false, func(col color.NRGBA) W {
		return LucideW("square-pen", col, 16)
	})
}

// searchField is T3's sidebar search: 32dp, radius 6, padded 6 by 8, the
// search icon, the field and, while there's a query, its clear button.
// Hovered or focused it takes the row hover fill; focused, it also rings
// 1dp inside at 70% of the ring colour.
func (th *Theme) searchField(gtx C, s *Scene, size image.Point) {
	const id = "search"
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit = true, true
	query := s.Sidebar.Query
	hovered, focused := false, false
	if s.Live {
		query = ed.Text()
		focused = gtx.Focused(ed)
		c, _ := th.events(gtx, "focus:search")
		hovered = c.Hovered()
	}
	r := image.Rectangle{Max: size}
	bg := SP.Sidebar
	if hovered || focused {
		bg = over(SP.SidebarRowHover, SP.Sidebar)
		fill(gtx, bg, r, dp(gtx, 6))
	}
	if focused {
		strokeRect(gtx, over(alpha(SP.Ring, 0xb3), bg), r, dp(gtx, 6), float32(max(1, dp(gtx, 1))))
	}
	var pop func()
	if s.Live {
		pop = th.area(gtx, "focus:search", size, pointer.CursorText)
	}
	icon := dp(gtx, 16)
	func() {
		defer op.Offset(image.Pt(dp(gtx, 8), (size.Y-icon)/2)).Push(gtx.Ops).Pop()
		LucideW("search", SP.SidebarIconColor, 16)(gtx)
	}()
	x0 := dp(gtx, 8+16+8)
	right := size.X - dp(gtx, 8)
	clear := strings.TrimSpace(query) != ""
	if clear {
		right -= dp(gtx, 20+8)
	}
	// The input's 21dp line (the html default, 1.5) is centred in the
	// field and its text in the line, so the text centres in the field.
	t := Txt{Size: 14, LineH: 1.5, Font: UI(font.Medium), Color: SP.SidebarForeground}
	func() {
		defer op.Offset(image.Pt(x0, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(max(0, right-x0), size.Y))
		Row(g, flexed(func(gtx C) D {
			var d D
			if query == "" {
				d = th.Text(gtx, Txt{Size: 14, LineH: 1.5, Font: UI(font.Medium), Color: over(SP.SidebarMutedForeground, bg), MaxLines: 1}, "Search")
			}
			if s.Live {
				th.searchKeys(gtx, ed)
				d = th.editor(gtx, id, t, "")
			} else if query != "" {
				d = th.Text(gtx, Txt{Size: 14, LineH: 1.5, Font: UI(font.Medium), Color: SP.SidebarForeground, MaxLines: 1}, query)
			}
			return d
		}))
	}()
	if clear {
		defer op.Offset(image.Pt(size.X-dp(gtx, 8+20), (size.Y-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
		th.clearButton(gtx, liveID(s, "search.clear"), bg)
	}
	if pop != nil {
		pop()
	}
}

// searchKeys hands the search field's arrows to the results: they move
// the highlight rather than the caret.
func (th *Theme) searchKeys(gtx C, ed any) {
	for {
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameDownArrow}, key.Filter{Focus: ed, Name: key.NameUpArrow})
		if !ok {
			return
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			th.push(Action{ID: "search", Kind: "key", Text: string(ke.Name)})
		}
	}
}

// clearButton is the search's clear button: T3's icon-micro ghost-muted
// Button, 20dp, radius 4, a 12dp x, accent on hover.
func (th *Theme) clearButton(gtx C, id string, on color.NRGBA) {
	s := dp(gtx, 20)
	col := SP.MutedForeground
	if id != "" {
		if c, _ := th.events(gtx, id); c.Hovered() {
			fill(gtx, over(SP.Accent, on), image.Rect(0, 0, s, s), dp(gtx, 4))
			col = SP.AccentForeground
		}
	}
	func() {
		defer op.Offset(image.Pt((s-dp(gtx, 12))/2, (s-dp(gtx, 12))/2)).Push(gtx.Ops).Pop()
		LucideW("x", col, 12)(gtx)
	}()
	if id != "" {
		th.hit(gtx, id, image.Pt(s, s))
	}
}

// headerButton is a header icon button (SidebarHeaderIconButton): 28dp,
// radius 8, the sidebar's icon colour; the row hover fill and the
// foreground on hover or while its popup is open, the active row fill
// while pressed. Its tooltip goes under it: T3's goes on top, but flips
// under at the window's top edge.
func (th *Theme) headerButton(gtx C, id, tip string, off, open bool, icon func(col color.NRGBA) W) D {
	s := dp(gtx, controlSize)
	r := image.Rectangle{Max: image.Pt(s, s)}
	hovered, pressed := false, false
	if id != "" && !off {
		c, _ := th.events(gtx, id)
		hovered, pressed = c.Hovered(), c.Pressed()
	}
	col := SP.SidebarIconColor
	switch {
	case pressed:
		fill(gtx, over(SP.SidebarRowActive, SP.Sidebar), r, dp(gtx, 8))
		col = SP.SidebarForeground
	case hovered || open:
		fill(gtx, over(SP.SidebarRowHover, SP.Sidebar), r, dp(gtx, 8))
		col = SP.SidebarForeground
	}
	if off {
		col = over(alpha(col, 0xa3), SP.Sidebar) // opacity 0.64
	}
	func() {
		defer op.Offset(image.Pt((s-dp(gtx, 16))/2, (s-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		icon(col)(gtx)
	}()
	if id != "" && !off {
		th.hit(gtx, id, r.Max)
		if !open {
			th.tipAt(gtx, id, tip, r.Max, false, 0)
		}
	}
	return D{Size: r.Max}
}

// filterButton is the project filter's trigger and its popup. Its icon
// is list-filter, or the scoped project's icon, or two of them, the second
// overlapping by 4dp inside a 2dp ring of the sidebar's colour.
func (th *Theme) filterButton(gtx C, s *Scene) {
	sb := &s.Sidebar
	icon := func(col color.NRGBA) W {
		return func(gtx C) D {
			switch len(sb.Scope) {
			case 0:
				return LucideW("list-filter", col, 16)(gtx)
			case 1:
				return th.projectIcon(gtx, sb.Scope[0], 16, SP.Sidebar, 0)
			}
			off := op.Offset(image.Pt(0, dp(gtx, 1))).Push(gtx.Ops)
			th.projectIcon(gtx, sb.Scope[0], 14, SP.Sidebar, 0)
			off.Pop()
			x := dp(gtx, 14-4)
			ring := dp(gtx, 2)
			fill(gtx, SP.Sidebar, image.Rect(x-ring, dp(gtx, 1)-ring, x+dp(gtx, 14)+ring, dp(gtx, 15)+ring), dp(gtx, 14*0.25)+ring)
			defer op.Offset(image.Pt(x, dp(gtx, 1))).Push(gtx.Ops).Pop()
			th.projectIcon(gtx, sb.Scope[1], 14, SP.Sidebar, 0)
			return D{Size: image.Pt(dp(gtx, 16), dp(gtx, 16))}
		}
	}
	th.headerButton(gtx, liveID(s, "filter"), sb.ScopeLabel, false, sb.Filter != nil, icon)
	if sb.Filter == nil {
		th.popIn(gtx, "filter", false, nil, nil) // so it plays again next time
		return
	}
	// It drops 4dp below the button, lined up with its left edge.
	th.deferBelow(gtx, image.Pt(0, dp(gtx, controlSize+4)), func(gtx C) D {
		return th.popUp(gtx, s, "filter", &f32.Point{}, func(gtx C) D { return th.filterPopup(gtx, s) })
	})
}

// filterPopup is the project filter (§3.2): 288dp of T3's menu surface, a
// search field over a rule, then "All projects" and each project in 32dp
// rows; a chosen one carries a check badge on its icon.
func (th *Theme) filterPopup(gtx C, s *Scene) D {
	f := s.Sidebar.Filter
	w := dp(gtx, 288)
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, dp(gtx, 370))}
	return popupFrame(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = w
		return Col(gtx,
			rigid(func(gtx C) D { return th.filterSearch(gtx, s) }),
			rigid(func(gtx C) D {
				if len(f.Items) == 0 {
					return pad(9, 9, 9, 9).Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.MutedForeground, Align: text.Middle}, "No matching projects.")
					})
				}
				// The rows scroll within the popup's height.
				return pad(4, 5, 5, 5).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, len(f.Items)*dp(gtx, 32))
					return th.List("filter", false).Layout(gtx, len(f.Items), func(gtx C, i int) D {
						return th.filterItem(gtx, s, f.Items[i])
					})
				})
			}),
		)
	})
}

// filterSearch is the popup's search, padded 10 12 0 inside its 1dp
// border, over a 1dp rule at border 70% that turns the ring colour while
// it's focused. The block sits 1dp up (-translate-y-px): the rule is
// placed for it, and the 26dp input moves up with it, its text centred.
func (th *Theme) filterSearch(gtx C, s *Scene) D {
	return pad(11, 13, 0, 13).Layout(gtx, func(gtx C) D {
		w := gtx.Constraints.Max.X
		h := dp(gtx, 26)
		ed := th.Editor("filter.search")
		ed.SingleLine = true
		focused := s.Live && gtx.Focused(ed)
		func() {
			defer op.Offset(image.Pt(0, (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			LucideW("search", over(alpha(P.MutedForeground, 0x8c), P.Popover), 16)(gtx)
		}()
		func() {
			defer op.Offset(image.Pt(dp(gtx, 20), -dp(gtx, 1))).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Exact(image.Pt(w-dp(gtx, 20+9), h))
			Row(g, flexed(func(gtx C) D {
				var d D
				if !s.Live || ed.Len() == 0 {
					d = th.Text(gtx, Txt{Size: 14, LineH: 26.0 / 14, Color: P.MutedForeground, MaxLines: 1}, "Search projects...")
				}
				if s.Live {
					th.filterKeys(gtx, ed)
					d = th.editor(gtx, "filter.search", Txt{Size: 14, LineH: 26.0 / 14, Color: P.PopoverForeground}, "")
				}
				return d
			}))
		}()
		rule := over(faded(P.Border, 0.7), P.Popover)
		if focused {
			rule = P.Ring
		}
		hline(gtx, rule, h+dp(gtx, 6)-max(1, dp(gtx, 1)), 0, w)
		return D{Size: image.Pt(w, h+dp(gtx, 6))}
	})
}

// filterKeys hands the filter search's arrows and Shift+Enter to its rows:
// the arrows move the highlight rather than the caret, and Shift+Enter
// adds or drops the highlighted project where Enter would pick it.
func (th *Theme) filterKeys(gtx C, ed any) {
	for {
		ev, ok := gtx.Event(key.Filter{Focus: ed, Name: key.NameDownArrow}, key.Filter{Focus: ed, Name: key.NameUpArrow},
			key.Filter{Focus: ed, Name: key.NameReturn, Required: key.ModShift})
		if !ok {
			return
		}
		if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
			th.push(Action{ID: "filter.search", Kind: "key", Text: string(ke.Name), Shift: ke.Modifiers.Contain(key.ModShift)})
		}
	}
}

// filterItem is one of the popup's rows: 32dp at least, padded 4 by 8,
// radius 4, the icon and the name 8dp apart, foreground at 9% while it's
// highlighted or under the pointer, which highlights it; a 24dp gap at the
// end where T3 keeps its project buttons.
func (th *Theme) filterItem(gtx C, s *Scene, it FilterItem) D {
	id := liveID(s, "filter.project:"+it.ID)
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
		switch {
		case hovered && th.in.filterHover != id:
			th.in.filterHover = id
			th.push(Action{ID: id, Kind: "hover"})
		case !hovered && th.in.filterHover == id:
			th.in.filterHover = ""
		}
	}
	w, h := gtx.Constraints.Max.X, dp(gtx, 32)
	r := image.Rect(0, 0, w, h)
	bg := P.Popover
	if hovered || it.Highlighted {
		bg = over(alpha(P.Foreground, 0x17), P.Popover)
		fill(gtx, bg, r, dp(gtx, 4))
	}
	func() {
		defer op.Offset(image.Pt(dp(gtx, 8), (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		if it.ID == "" {
			LucideW("folder", P.MutedForeground, 16)(gtx)
		} else {
			th.projectIcon(gtx, it.Icon, 16, bg, 0)
		}
		if it.Selected {
			// The badge: 12dp of primary in a 2dp ring of the popover, a
			// check at 8dp, 4dp past the icon's bottom right.
			c := image.Pt(dp(gtx, 16+4)-dp(gtx, 6), dp(gtx, 16+4)-dp(gtx, 6))
			circle(gtx, P.Popover, c, dp(gtx, 8))
			circle(gtx, P.Primary, c, dp(gtx, 6))
			defer op.Offset(c.Sub(image.Pt(dp(gtx, 4), dp(gtx, 4)))).Push(gtx.Ops).Pop()
			lucideStroke("check", rgb(0xffffff), 8, 4)(gtx)
		}
	}()
	func() {
		x := dp(gtx, 8+16+8)
		defer op.Offset(image.Pt(x, (h-dp(gtx, 20))/2)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(max(0, w-x-dp(gtx, 8+24)), dp(gtx, 20)))
		th.Text(g, Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}, it.Label)
	}()
	if id != "" {
		th.hit(gtx, id, r.Max)
	}
	return D{Size: r.Max}
}

// ---------- the list ----------

// sideItem is one entry of the drawn list, with its height and its place
// in the keyboard's tree: nav is its id there, parent its shelf's, and
// open is 1 or 0 for a shelf's header, -1 for the rest.
type sideItem struct {
	h      int
	draw   func(gtx C, top int) D
	nav    string
	parent string
	open   int
}

// listNav is one frame's keyboard tree over the list: each focusable row
// in order, with the rows' tops in the list.
type listNav struct {
	items []sideItem
	tops  []int
}

// threadList is the scrolling list (§3.3): rows 8dp in from the sides and
// 1dp apart, 8dp of padding below, no scrollbar, and a fade of up to 12dp
// at an edge it has scrolled past. A short list puts its spare height
// above the first shelf, so the shelves sit at the bottom. top is the
// list's y in the window.
func (th *Theme) threadList(gtx C, s *Scene, top int) {
	size := gtx.Constraints.Max
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if s.Sidebar.Renaming == "" {
		th.in.renaming = false
	}
	items := th.sideItems(gtx, s)
	gap, bottom := max(1, dp(gtx, 1)), dp(gtx, sideInset)
	total := bottom
	firstShelf := -1
	for i, it := range items {
		total += it.h
		if i > 0 {
			total += gap
		}
		if firstShelf < 0 && it.open >= 0 && strings.HasPrefix(it.nav, "shelf:") {
			firstShelf = i
		}
	}
	if spare := size.Y - total; spare > 0 && firstShelf >= 0 {
		spacer := sideItem{h: spare - gap, open: -1, draw: func(gtx C, _ int) D { return D{} }}
		items = append(items[:firstShelf], append([]sideItem{spacer}, items[firstShelf:]...)...)
		total = size.Y
	}
	// Each item's top, with the gap below it, then the bottom padding.
	tops := make([]int, len(items)+1)
	y := 0
	for i, it := range items {
		tops[i] = y
		y += it.h + gap
	}
	tops[len(items)] = y - gap
	th.in.nav = listNav{items: items, tops: tops}
	th.in.listView = size.Y
	l := th.List("sidebar", false)
	scrolled := func() int {
		if p := l.Position; p.First < len(items) {
			return tops[p.First] + p.Offset
		}
		return 0
	}
	// A new project scope starts the list at its top. Each newly opened
	// thread's row scrolls into view once, unless the pointer is on it,
	// where a click leaves it; while the row isn't listed (a search
	// replaces the list), it waits (routeReveal.ts).
	sb := &s.Sidebar
	if sb.ScopeKey != th.in.listScope {
		th.in.listScope = sb.ScopeKey
		l.Position = layout.Position{}
	}
	th.in.listScroll = scrolled()
	switch {
	case sb.Open == "":
		th.in.listOpen = ""
	case sb.Open != th.in.listOpen:
		for i, it := range items {
			if it.nav != "thread:"+sb.Open {
				continue
			}
			th.in.listOpen = sb.Open
			if c := th.in.clicks[it.nav]; c == nil || !c.Hovered() {
				th.revealItem(i)
			}
		}
	}
	scroll := scrolled()
	th.in.listScroll = scroll
	inset := dp(gtx, sideInset)
	l.Layout(gtx, len(items)+1, func(gtx C, i int) D {
		if i == len(items) {
			return D{Size: image.Pt(size.X, bottom)}
		}
		it := items[i]
		h := it.h
		if i < len(items)-1 {
			h += gap
		}
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X-2*inset, it.h))
		func() {
			defer op.Offset(image.Pt(inset, 0)).Push(gtx.Ops).Pop()
			it.draw(g, top+tops[i]-scroll)
		}()
		return D{Size: image.Pt(size.X, h)}
	})
	th.listFade(gtx, size, scroll, max(0, total-size.Y-scroll))
}

// listFade is the list's edge mask: the sidebar's colour painted over the
// content, fading out across min(12dp, the distance scrolled from that
// edge).
func (th *Theme) listFade(gtx C, size image.Point, fromTop, fromBottom int) {
	edge := func(y0, y1 int, far bool) {
		if y1 == y0 {
			return
		}
		c0, c1 := SP.Sidebar, alpha(SP.Sidebar, 0)
		if far {
			c0, c1 = c1, c0
		}
		st := clip.Rect{Min: image.Pt(0, min(y0, y1)), Max: image.Pt(size.X, max(y0, y1))}.Push(gtx.Ops)
		paint.LinearGradientOp{Stop1: f32.Pt(0, float32(y0)), Color1: c0, Stop2: f32.Pt(0, float32(y1)), Color2: c1}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		st.Pop()
	}
	f := dp(gtx, 12)
	edge(0, min(f, fromTop), false)
	edge(size.Y-min(f, fromBottom), size.Y, true)
}

// sideItems is the list's entries for this frame: the search's results,
// or the empty state and the rows, shelves and "Show N more".
func (th *Theme) sideItems(gtx C, s *Scene) []sideItem {
	sb := &s.Sidebar
	var items []sideItem
	add := func(h float32, nav, parent string, open int, draw func(gtx C, top int) D) {
		items = append(items, sideItem{h: dp(gtx, h), nav: nav, parent: parent, open: open, draw: draw})
	}
	if strings.TrimSpace(sb.Query) != "" || sb.Results != nil || sb.NoResults != "" {
		for i := range sb.Results {
			r, i := &sb.Results[i], i
			h := float32(slimH)
			if r.Excerpt != nil {
				h = excerptH
			}
			add(h, "", "", -1, func(gtx C, top int) D { return th.resultRow(gtx, s, r, i, top) })
		}
		if sb.NoResults != "" {
			add(64, "", "", -1, func(gtx C, _ int) D { return th.listNote(gtx, sb.NoResults) })
		}
		return items
	}
	if e := sb.Empty; e != nil {
		// Its message wraps, so its height is the message's.
		m := op.Record(gtx.Ops)
		h := 2*dp(gtx, 24) + th.emptyText(gtx, e, gtx.Constraints.Max.X-2*dp(gtx, sideInset+8)).Size.Y
		m.Stop()
		if e.Button != "" {
			h += dp(gtx, 8+26)
		}
		items = append(items, sideItem{h: h, open: -1, draw: func(gtx C, _ int) D { return th.emptyState(gtx, s, e) }})
	}
	for _, it := range sb.Items {
		switch {
		case it.Row != nil && it.Row.Slim:
			r := it.Row
			add(slimH, "thread:"+r.ID, shelfParent(r.Shelf), -1, func(gtx C, top int) D { return th.slimRow(gtx, s, r, top) })
		case it.Row != nil:
			r := it.Row
			h := float32(cardH + 4)
			if !hasMeta(r) {
				h = cardFlatH + 4
			}
			add(h, "thread:"+r.ID, shelfParent(r.Shelf), -1, func(gtx C, top int) D { return th.cardRow(gtx, s, r, top) })
		case it.Draft != nil:
			d := it.Draft
			add(cardH+4, "draft:"+d.ID, "", -1, func(gtx C, top int) D { return th.draftRow(gtx, s, d, top) })
		case it.Divider:
			add(1+2*6, "", "", -1, th.draftDivider)
		case it.Shelf != nil:
			sh := it.Shelf
			open := 0
			if sh.Open {
				open = 1
			}
			add(shelfH, "shelf:"+sh.ID, "", open, func(gtx C, _ int) D { return th.shelfHeader(gtx, s, sh) })
		case it.More > 0:
			n := it.More
			add(slimH, "settled.more", "shelf:settled", -1, func(gtx C, _ int) D { return th.moreRow(gtx, s, n) })
		}
	}
	return items
}

func shelfParent(shelf string) string {
	if shelf == "" {
		return ""
	}
	return "shelf:" + shelf
}

// hasMeta reports a card with a meta line: a branch, a scratch machine, a
// pull request or a running terminal. Without one the card is 60dp and its
// trailing icons move to the title's line.
func hasMeta(r *ThreadRow) bool {
	return r.Branch != "" || r.Scratch != "" || r.PR != nil || r.Terminals > 0
}

// ---------- rows: targets, focus and the keyboard ----------

// area makes size at the current offset the click target id and keeps
// its clip pushed until pop: targets drawn before pop nest inside it, so
// it stays hovered over them.
func (th *Theme) area(gtx C, id string, size image.Point, cursor pointer.Cursor) (pop func()) {
	c, rc := th.events(gtx, id)
	st := clip.Rect{Max: size}.Push(gtx.Ops)
	c.Add(gtx.Ops)
	event.Op(gtx.Ops, rc)
	cursor.Add(gtx.Ops)
	return st.Pop
}

// rowTarget makes a list row the target nav at the current offset: its
// clicks, right clicks and hover, and its keyboard focus (§5.12). Controls
// drawn before pop nest in it, and a click on one isn't the row's too.
// rect is the row's box in the window, where its context menu key opens
// the menu. It reports whether the row is hovered and keyboard-focused.
func (th *Theme) rowTarget(gtx C, s *Scene, nav string, size image.Point, rect image.Rectangle) (hovered, focused bool, pop func()) {
	th.in.tipBusy = false
	if !s.Live {
		return false, false, func() {}
	}
	start := len(th.in.actions)
	c, rc := th.events(gtx, nav)
	mine := len(th.in.actions)
	tag := th.rowKey(nav)
	th.rowKeys(gtx, nav, tag, rect)
	focused = gtx.Focused(tag) && th.in.keyFocus
	st := clip.Rect{Max: size}.Push(gtx.Ops)
	c.Add(gtx.Ops)
	event.Op(gtx.Ops, rc)
	event.Op(gtx.Ops, tag)
	pointer.CursorPointer.Add(gtx.Ops)
	return c.Hovered(), focused, func() {
		st.Pop()
		if len(th.in.actions) > mine && mine > start {
			// A control inside took the click.
			th.in.actions = append(th.in.actions[:start], th.in.actions[mine:]...)
		}
	}
}

// rowKey is nav's keyboard focus tag.
func (th *Theme) rowKey(nav string) *int {
	if th.in.rowTags == nil {
		th.in.rowTags = map[string]*int{}
	}
	t := th.in.rowTags[nav]
	if t == nil {
		t = new(int)
		th.in.rowTags[nav] = t
	}
	return t
}

// rowKeys runs the list's keyboard on a row (sidebarTree.ts): the arrows
// move focus between rows without opening them, clamped; Home and End go
// to the ends; Right opens a folded shelf or steps into an open one, Left
// folds it or steps out to the row's shelf; Enter and Space open the row,
// F2 renames it, and Shift+F10 opens its menu at its bottom left.
func (th *Theme) rowKeys(gtx C, nav string, tag *int, rect image.Rectangle) {
	if th.in.focusRow == nav {
		gtx.Execute(key.FocusCmd{Tag: tag})
		th.in.focusRow = ""
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: tag},
			key.Filter{Focus: tag, Name: key.NameDownArrow}, key.Filter{Focus: tag, Name: key.NameUpArrow},
			key.Filter{Focus: tag, Name: key.NameHome}, key.Filter{Focus: tag, Name: key.NameEnd},
			key.Filter{Focus: tag, Name: key.NameRightArrow}, key.Filter{Focus: tag, Name: key.NameLeftArrow},
			key.Filter{Focus: tag, Name: key.NameReturn}, key.Filter{Focus: tag, Name: key.NameEnter},
			key.Filter{Focus: tag, Name: key.NameSpace}, key.Filter{Focus: tag, Name: key.NameF2},
			key.Filter{Focus: tag, Name: key.NameF10, Required: key.ModShift},
		)
		if !ok {
			return
		}
		switch e := ev.(type) {
		case key.FocusEvent:
			// Rows only take focus from the keyboard: Tab, or the arrows.
			if e.Focus {
				th.in.keyFocus = true
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			th.in.keyFocus = true
			th.rowKey1(nav, e.Name, rect)
		}
	}
}

func (th *Theme) rowKey1(nav string, name key.Name, rect image.Rectangle) {
	items := th.in.nav.items
	var rows []int
	at := -1
	for i, it := range items {
		if it.nav == "" {
			continue
		}
		if it.nav == nav {
			at = len(rows)
		}
		rows = append(rows, i)
	}
	if at < 0 {
		return
	}
	cur := items[rows[at]]
	target := -1
	switch name {
	case key.NameDownArrow:
		target = rows[min(at+1, len(rows)-1)]
	case key.NameUpArrow:
		target = rows[max(at-1, 0)]
	case key.NameHome:
		target = rows[0]
	case key.NameEnd:
		target = rows[len(rows)-1]
	case key.NameRightArrow, key.NameLeftArrow:
		if (name == key.NameRightArrow && cur.open == 0) || (name == key.NameLeftArrow && cur.open == 1) {
			th.push(Action{ID: nav, Kind: "click"})
			return
		}
		for _, i := range rows {
			if (name == key.NameRightArrow && items[i].parent == nav) || (name == key.NameLeftArrow && items[i].nav == cur.parent) {
				target = i
				break
			}
		}
	case key.NameReturn, key.NameEnter, key.NameSpace:
		th.push(Action{ID: nav, Kind: "click"})
	case key.NameF2:
		if ref, ok := cutPrefix(nav, "thread:"); ok {
			th.push(Action{ID: "rename:" + ref, Kind: "click"})
		}
	case key.NameF10:
		if strings.HasPrefix(nav, "thread:") || strings.HasPrefix(nav, "draft:") {
			th.in.ctxAt = image.Pt(rect.Min.X, rect.Max.Y)
			th.in.menu.open = false
			th.push(Action{ID: nav, Kind: "context", Text: "keyboard"})
		}
	}
	if target >= 0 {
		th.in.focusRow = items[target].nav
		th.revealItem(target)
	}
}

// revealItem scrolls the list so item i shows, moving it as little as it
// can (block: nearest).
func (th *Theme) revealItem(i int) {
	n := th.in.nav
	l := th.List("sidebar", false)
	view := th.in.listView
	top, bottom := n.tops[i], n.tops[i]+n.items[i].h
	scroll := th.in.listScroll
	switch {
	case top < scroll:
		scroll = top
	case bottom > scroll+view:
		scroll = bottom - view
	default:
		return
	}
	for j := len(n.items) - 1; j >= 0; j-- {
		if n.tops[j] <= scroll {
			l.Position.First, l.Position.Offset, l.Position.BeforeEnd = j, scroll-n.tops[j], true
			return
		}
	}
}

// ---------- the parts of a row ----------

// rowInk paints a row's colours as T3 composites them: onto the row's own
// fill, then the row at opacity k onto the sidebar.
type rowInk struct {
	bg color.NRGBA // the row's surface, opaque
	k  float32
}

func (in rowInk) of(c color.NRGBA) color.NRGBA {
	return over(alpha(over(c, in.bg), uint8(in.k*255+0.5)), SP.Sidebar)
}

// rowSurface is a row's fill by state (§3.5): the open thread, a selected
// one, an unsent draft, hover. The light theme rings the open and
// selected rows 1dp inside with the sidebar border at 70%. Without one
// the row is the sidebar.
func rowSurface(r *ThreadRow, hot bool) (bg, ring color.NRGBA) {
	bg = SP.Sidebar
	switch {
	case r.Active:
		bg = over(SP.SidebarRowActive, SP.Sidebar)
	case r.Selected:
		bg = over(SP.SidebarRowSelected, SP.Sidebar)
	case r.Draft && hot:
		bg = over(alpha(SP.Warning, 0x14), SP.Sidebar)
	case r.Draft:
		bg = over(alpha(SP.Warning, 0x0a), SP.Sidebar)
	case hot:
		bg = over(SP.SidebarRowHover, SP.Sidebar)
	}
	if !dark && (r.Active || r.Selected) {
		ring = over(alpha(SP.SidebarBorder, 0xb3), bg)
	}
	return bg, ring
}

// paintRow fills a row's surface, its light ring and, while it has
// keyboard focus, the 2dp focus ring inside it.
func paintRow(gtx C, size image.Point, in rowInk, ring color.NRGBA, focused bool) {
	r := image.Rectangle{Max: size}
	rad := dp(gtx, 6)
	if in.bg != SP.Sidebar {
		fill(gtx, in.of(in.bg), r, rad)
	}
	px := float32(max(1, dp(gtx, 1)))
	if ring.A != 0 {
		strokeRect(gtx, in.of(ring), r, rad, px)
	}
	if focused {
		strokeRect(gtx, SP.Ring, r, rad, 2*px)
	}
}

// cardRow is a pinned, active or working thread (§2.4, §3.4–§3.6): 78dp
// (60 with no meta line) in 2dp of padding, radius 6, padded 8 by 10. Line
// 1 is the project and the status slot, line 2 the title, line 3 the
// branch, terminal, pull request, machine and providers. Hovering swaps
// the status for the controls, at once.
func (th *Theme) cardRow(gtx C, s *Scene, r *ThreadRow, top int) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, cardH)
	meta := hasMeta(r)
	if !meta {
		h = dp(gtx, cardFlatH)
	}
	size := image.Pt(w, h)
	defer op.Offset(image.Pt(0, dp(gtx, 2))).Push(gtx.Ops).Pop()
	x0 := dp(gtx, sideInset)
	rect := image.Rectangle{Min: image.Pt(x0, top+dp(gtx, 2)), Max: image.Pt(x0+w, top+dp(gtx, 2)+h)}
	nav := "thread:" + r.ID
	hovered, focused, pop := th.rowTarget(gtx, s, nav, size, rect)
	defer pop()
	hot := hovered || r.Hover
	menuOpen := s.Sidebar.Snooze != nil && s.Sidebar.Snooze.Row == r.ID
	bg, ring := rowSurface(r, hot)
	in := rowInk{bg: bg, k: 1}
	// A receded working row is at 70% until hovered or focused.
	if r.Recede && r.Status == StatusWorking {
		in.k = 0.7 + 0.3*th.transition(gtx, nav+".fade", hot || focused, 150*time.Millisecond)
	}
	paintRow(gtx, size, in, ring, focused)
	cw := w - 2*dp(gtx, 10)
	at := func(y, lh int, draw func(gtx C) D) {
		defer op.Offset(image.Pt(dp(gtx, 10), y)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(cw, lh))
		draw(g)
	}

	// Line 1: the draft pen, the project, the pin, its name, the status slot.
	at(dp(gtx, 8), dp(gtx, 20), func(gtx C) D {
		var kids []layout.FlexChild
		if r.Draft {
			// An unsent draft's pen, in the warning foreground.
			kids = append(kids, rigid(LucideW("square-pen", in.of(SP.WarningForeground), 12)))
		}
		if r.Project != "" {
			kids = append(kids, rigid(func(gtx C) D { return th.projectIcon(gtx, r.Icon, 16, in.bg, 0) }))
		}
		if r.Pinned {
			kids = append(kids, rigid(th.pin(s, r, in)))
		}
		kids = append(kids,
			flexed(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: in.of(SP.SecondaryLabel), MaxLines: 1}, r.Project)
			}),
			rigid(func(gtx C) D {
				return th.statusSlot(gtx, s, r, in, hot || menuOpen, image.Pt(rect.Max.X-dp(gtx, 10), rect.Min.Y+dp(gtx, 8)))
			}),
		)
		return RowGap(gtx, 6, kids...)
	})
	// Line 2: the title, and the trailing icons when there's no line 3.
	at(dp(gtx, 8+20+4), dp(gtx, 20), func(gtx C) D {
		kids := []layout.FlexChild{flexed(func(gtx C) D { return th.title(gtx, s, r, in, hot || focused, false) })}
		if !meta {
			kids = append(kids, rigid(func(gtx C) D { return th.trailingIcons(gtx, r, in) }))
		}
		return RowGap(gtx, 6, kids...)
	})
	if meta {
		at(dp(gtx, 8+20+4+20+2), dp(gtx, 16), func(gtx C) D { return th.metaLine(gtx, s, r, in) })
	}
	th.hoverCard(gtx, s, r, rect, hot && !menuOpen)
	return D{Size: image.Pt(w, h+dp(gtx, 4))}
}

// draftRow is the new thread's unsent draft (§2.7, §3.12): 78dp in 2dp of
// padding, radius 6, warning at 4% (8% hovered) or, while it's open, the
// open row's fill without its ring; padded 8 by 10. Line 1 is the pen, the
// project's icon, its name in 12/16 500 and the discard button; line 2,
// 2dp under it, the first line written in 14/20 500, the foreground at 90%.
func (th *Theme) draftRow(gtx C, s *Scene, d *DraftRow, top int) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, cardH)
	size := image.Pt(w, h)
	defer op.Offset(image.Pt(0, dp(gtx, 2))).Push(gtx.Ops).Pop()
	x0 := dp(gtx, sideInset)
	rect := image.Rectangle{Min: image.Pt(x0, top+dp(gtx, 2)), Max: image.Pt(x0+w, top+dp(gtx, 2)+h)}
	hovered, focused, pop := th.rowTarget(gtx, s, "draft:"+d.ID, size, rect)
	defer pop()
	hot := hovered || d.Hover
	bg := over(alpha(SP.Warning, 0x0a), SP.Sidebar)
	switch {
	case d.Active:
		bg = over(SP.SidebarRowActive, SP.Sidebar)
	case hot:
		bg = over(alpha(SP.Warning, 0x14), SP.Sidebar)
	}
	in := rowInk{bg: bg, k: 1}
	paintRow(gtx, size, in, color.NRGBA{}, focused)
	at := func(y, lh int, draw func(gtx C) D) {
		defer op.Offset(image.Pt(dp(gtx, 10), y)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(w-2*dp(gtx, 10), lh))
		draw(g)
	}
	at(dp(gtx, 8), dp(gtx, 20), func(gtx C) D {
		return RowGap(gtx, 6,
			rigid(LucideW("square-pen", in.of(SP.WarningForeground), 12)),
			rigid(func(gtx C) D { return th.projectIcon(gtx, d.Icon, 16, in.bg, 0) }),
			flexed(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: in.of(SP.SecondaryLabel), MaxLines: 1}, d.Project)
			}),
			rigid(func(gtx C) D { return th.discardButton(gtx, s, d, in, hot) }),
		)
	})
	at(dp(gtx, 8+20+2), dp(gtx, 20), func(gtx C) D {
		return Row(gtx, flexed(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: in.of(faded(SP.Foreground, 0.9)), MaxLines: 1}, d.Preview)
		}))
	})
	return D{Size: image.Pt(w, h+dp(gtx, 4))}
}

// discardButton is the draft row's Discard draft: a 12dp x padded 0 by 4
// in the muted foreground, the foreground hovered, which fades in over
// 150ms while the row is hovered.
func (th *Theme) discardButton(gtx C, s *Scene, d *DraftRow, in rowInk, hot bool) D {
	size := image.Pt(dp(gtx, 12+2*4), gtx.Constraints.Max.Y)
	k := th.transition(gtx, "discard:"+d.ID, hot, 150*time.Millisecond)
	id := liveID(s, "discard:"+d.ID)
	col := SP.MutedForeground
	if id != "" {
		if c, _ := th.events(gtx, id); c.Hovered() {
			col = SP.Foreground
		}
	}
	func() {
		defer op.Offset(image.Pt(dp(gtx, 4), (size.Y-dp(gtx, 12))/2)).Push(gtx.Ops).Pop()
		LucideW("x", over(faded(in.of(col), k), in.bg), 12)(gtx)
	}()
	if id != "" && k > 0 {
		th.hit(gtx, id, size)
		th.listTip(gtx, id, "Discard draft", size)
	}
	return D{Size: size}
}

// draftDivider is the rule under the drafts: 1dp of the sidebar border at
// 60%, 10dp in from the sides and 6 clear above and below.
func (th *Theme) draftDivider(gtx C, _ int) D {
	w := gtx.Constraints.Max.X
	hline(gtx, over(faded(SP.SidebarBorder, 0.6), SP.Sidebar), dp(gtx, 6), dp(gtx, 10), w-dp(gtx, 10))
	return D{Size: image.Pt(w, dp(gtx, 1+2*6))}
}

// faded is c at k of its own opacity, as Tailwind's color/NN is: a
// translucent token keeps its own see-through, times k.
func faded(c color.NRGBA, k float32) color.NRGBA {
	c.A = uint8(float32(c.A)*k + 0.5)
	return c
}

// title is a row's title, or its editor while it's renamed.
func (th *Theme) title(gtx C, s *Scene, r *ThreadRow, in rowInk, lit, slim bool) D {
	if s.Live && s.Sidebar.Renaming == r.ID {
		return th.renameField(gtx)
	}
	return th.rowTitle(gtx, r, in, lit, slim)
}

// renameField is the title's editor (§3.17): 22dp, padded 0 by 4, radius
// 4, the card in a 1dp input border that turns the foreground while it's
// focused, 14px medium. Enter or leaving it commits, Escape cancels.
func (th *Theme) renameField(gtx C) D {
	const id = "rename"
	ed := th.Editor(id)
	ed.SingleLine, ed.Submit = true, true
	focused := gtx.Focused(ed)
	if th.in.renaming && !focused {
		th.push(Action{ID: id, Kind: "blur"})
	}
	th.in.renaming = focused
	w, h := gtx.Constraints.Max.X, dp(gtx, 22)
	r := image.Rect(0, 0, w, h)
	border := SP.Input
	if focused {
		border = SP.Foreground
	}
	fill(gtx, SP.Card, r, dp(gtx, 4))
	strokeRect(gtx, over(border, SP.Card), r, dp(gtx, 4), float32(max(1, dp(gtx, 1))))
	lineH := dp(gtx, 20)
	defer op.Offset(image.Pt(dp(gtx, 4), (h-lineH)/2)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(max(0, w-dp(gtx, 8)), lineH))
	th.editor(g, id, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: SP.CardForeground}, "")
	return D{Size: r.Max}
}

// rowTitle is a row's title, 14/20: 500, or 400 receded. On a card a
// receded title is the foreground at 75% until the row is hovered or
// focused, and a failed one seen is at 95%. The slim rows' rule is in
// slimTitle. A title being regenerated is at 55%.
func (th *Theme) rowTitle(gtx C, r *ThreadRow, in rowInk, lit, slim bool) D {
	weight := font.Medium
	if r.Recede {
		weight = font.Normal
	}
	col := SP.Foreground
	switch {
	case slim:
		col = slimTitle(r, lit)
	case r.Recede && !lit:
		col = alpha(SP.Foreground, 0xbf)
	case r.Status == StatusFailed && !r.Unread:
		col = alpha(SP.Foreground, 0xf2)
	}
	if r.Regenerating {
		col.A = uint8(float32(col.A)*0.55 + 0.5)
	}
	return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Font: UI(weight), Color: in.of(col), MaxLines: 1}, r.Title)
}

// slimTitle is a slim row's title colour: a settled row's is the
// secondary label at rest; the others' follow its state; hovered, all are
// the foreground.
func slimTitle(r *ThreadRow, lit bool) color.NRGBA {
	switch {
	case lit:
		return SP.Foreground
	case r.Shelf == "settled" && !r.Active:
		return SP.SecondaryLabel
	case r.Recede:
		return SP.SecondaryLabel
	case r.Active || r.Woke || r.Status == StatusInput || r.Status == StatusRequest:
		return SP.Foreground
	case r.Unread:
		return SP.MutedForeground
	}
	return alpha(SP.SecondaryLabel, 0xb3)
}

// statusIcon and statusColor are a status label's lucide icon and colour.
func statusIcon(st RowStatus) string {
	switch st {
	case StatusWorking:
		return "circle-dashed"
	case StatusWaiting:
		return "circle-dot-dashed"
	case StatusApproval:
		return "shield-question-mark"
	case StatusInput:
		return "message-circle-question-mark"
	case StatusRequest:
		return "user-round"
	case StatusLimited, StatusFailed:
		return "circle-alert"
	case StatusWoke:
		return "alarm-clock"
	}
	return "circle-check"
}

func statusColor(st RowStatus) color.NRGBA {
	switch st {
	case StatusWorking:
		return SP.InfoForeground
	case StatusWaiting:
		return SP.MutedForeground
	case StatusApproval, StatusLimited, StatusWoke:
		return SP.WarningForeground
	case StatusInput:
		if dark {
			return Tailwind["indigo-300"]
		}
		return Tailwind["indigo-600"]
	case StatusRequest:
		return SP.GuestForeground
	case StatusFailed:
		return SP.ErrorForeground
	}
	return SP.SuccessForeground
}

// statusSlot is the end of a card's first line: at least 32dp, right
// aligned. It shows the status label, or the age; hovered (or while its
// snooze menu is open) it shows the controls instead, beside a Woke label,
// which stays. While the jump keys show, it shows only the row's. end is
// the line's top right corner in the window.
func (th *Theme) statusSlot(gtx C, s *Scene, r *ThreadRow, in rowInk, hot bool, end image.Point) D {
	var kids []layout.FlexChild
	woke := r.Status == StatusWoke
	menuOpen := s.Sidebar.Snooze != nil && s.Sidebar.Snooze.Row == r.ID
	if th.in.jumping && r.Jump > 0 && !menuOpen {
		kids = append(kids, rigid(th.jumpKey(r, in)))
	} else {
		if !hot || woke {
			kids = append(kids, rigid(func(gtx C) D { return th.statusLabel(gtx, s, r, in) }))
		}
		if hot && (r.CanSettle || r.CanSnooze || r.Draft) {
			kids = append(kids, rigid(func(gtx C) D { return th.rowControls(gtx, s, r, in, end) }))
		}
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints.Min = image.Pt(0, gtx.Constraints.Max.Y)
	d := Row(g, kids...)
	call := m.Stop()
	w := max(d.Size.X, dp(gtx, 32))
	defer op.Offset(image.Pt(w-d.Size.X, 0)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: image.Pt(w, gtx.Constraints.Max.Y)}
}

// statusLabel is a card's label (§3.6): a 16dp icon and the label in 12/16
// medium in the status's colour, then a working or waiting timer 4dp on;
// Woke is a button that clears it. With no status, the row's age.
func (th *Theme) statusLabel(gtx C, s *Scene, r *ThreadRow, in rowInk) D {
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: in.of(SP.SecondaryLabel), MaxLines: 1}
	if r.Status == StatusNone {
		return th.Text(gtx, t, r.Age)
	}
	col := in.of(statusColor(r.Status))
	t.Color, t.Font = col, UI(font.Medium)
	label := func(gtx C) D {
		kids := []layout.FlexChild{
			rigid(LucideW(statusIcon(r.Status), col, 16)),
			rigid(func(gtx C) D { return th.Text(gtx, t, r.Label) }),
		}
		if r.Timer != "" {
			kids = append(kids, rigid(func(gtx C) D { return th.Text(gtx, t, r.Timer) }))
		}
		return RowGap(gtx, 4, kids...)
	}
	if r.Status != StatusWoke {
		return label(gtx)
	}
	return th.wokeButton(gtx, s, r, label)
}

// wokeButton is the Woke label as T3's button: a click records the visit
// that clears it, without opening the thread; it underlines on hover.
func (th *Theme) wokeButton(gtx C, s *Scene, r *ThreadRow, label W) D {
	id := liveID(s, "woke:"+r.ID)
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	d := label(gtx)
	if hovered {
		hline(gtx, statusColor(StatusWoke), d.Size.Y-dp(gtx, 2), dp(gtx, 16+4), d.Size.X)
	}
	if id != "" {
		th.hit(gtx, id, d.Size)
		th.listTip(gtx, id, "Dismiss Woke notification", d.Size)
	}
	return d
}

// rowControls are a card's hover controls, laid out from the right:
// Discard draft, the snooze clock and Settle, which sits 4dp into the
// padding. Each is the muted foreground, the foreground on hover. end is
// their line's top right corner in the window.
func (th *Theme) rowControls(gtx C, s *Scene, r *ThreadRow, in rowInk, end image.Point) D {
	h := gtx.Constraints.Max.Y
	settle := Txt{Size: 12, LineH: 16.0 / 12, MaxLines: 1}
	// Each button is padded 0 by 6; then is what hangs off it.
	button := func(id, tip string, body func(col color.NRGBA) W, then func(gtx C, size image.Point)) layout.FlexChild {
		return rigid(func(gtx C) D {
			col := SP.MutedForeground
			if id != "" {
				if c, _ := th.events(gtx, id); c.Hovered() {
					col = SP.Foreground
				}
			}
			m := op.Record(gtx.Ops)
			g := gtx
			g.Constraints.Min = image.Point{}
			d := body(in.of(col))(g)
			call := m.Stop()
			size := image.Pt(d.Size.X+2*dp(gtx, 6), h)
			func() {
				defer op.Offset(image.Pt(dp(gtx, 6), (h-d.Size.Y)/2)).Push(gtx.Ops).Pop()
				call.Add(gtx.Ops)
			}()
			if id != "" {
				th.hit(gtx, id, size)
				th.listTip(gtx, id, tip, size)
			}
			if then != nil {
				then(gtx, size)
			}
			return D{Size: size}
		})
	}
	var kids []layout.FlexChild
	if r.Draft {
		kids = append(kids, button(liveID(s, "discard:"+r.ID), "Discard draft", func(col color.NRGBA) W { return LucideW("x", col, 14) }, nil))
	}
	if r.CanSnooze {
		open := s.Sidebar.Snooze != nil && s.Sidebar.Snooze.Row == r.ID
		tip := "Snooze thread"
		if open {
			tip = ""
		}
		kids = append(kids, button(liveID(s, "snooze:"+r.ID), tip, func(col color.NRGBA) W { return LucideW("clock", col, 12) },
			func(gtx C, size image.Point) {
				if !open {
					th.popIn(gtx, "snooze:"+r.ID, false, nil, nil)
					return
				}
				// In the window it ends where Settle, the only control
				// after it, starts.
				right := end.X
				if r.CanSettle {
					right += dp(gtx, 4) - (2*dp(gtx, 6) + dp(gtx, 14) + dp(gtx, 4) + th.textWidth(gtx, settle, "Settle"))
				}
				th.snoozeMenu(gtx, s, size, image.Rect(right-size.X, end.Y, right, end.Y+size.Y))
			}))
	}
	if r.CanSettle {
		kids = append(kids, button(liveID(s, "settle:"+r.ID), "Settle thread", func(col color.NRGBA) W {
			return func(gtx C) D {
				t := settle
				t.Color = col
				return RowGap(gtx, 4,
					rigid(LucideW("check", col, 14)),
					rigid(func(gtx C) D { return th.Text(gtx, t, "Settle") }),
				)
			}
		}, nil))
	}
	d := Row(gtx, kids...)
	if r.CanSettle {
		// Settle's -4dp end margin: it reaches into the row's padding.
		d.Size.X -= dp(gtx, 4)
	}
	return d
}

// pin is a pinned thread's pin: 12dp, the muted foreground at 65%. Where
// the user can write it's the Unpin button, which turns into pin-off on
// hover.
func (th *Theme) pin(s *Scene, r *ThreadRow, in rowInk) W {
	return func(gtx C) D {
		id := ""
		if r.CanUnpin {
			id = liveID(s, "unpin:"+r.ID)
		}
		icon, col := "pin", alpha(SP.MutedForeground, 0xa6)
		if id != "" {
			if c, _ := th.events(gtx, id); c.Hovered() {
				icon, col = "pin-off", SP.Foreground
			}
		}
		d := LucideW(icon, in.of(col), 12)(gtx)
		if id != "" {
			th.hit(gtx, id, d.Size)
			th.listTip(gtx, id, "Unpin thread", d.Size)
		}
		return d
	}
}

// metaLine is a card's third line, 12/16 in the secondary label: the
// branch (after the worktree icon), or a "No project" thread's machine;
// the running terminal and the pull request; and, at the end, the machine
// and provider icons.
func (th *Theme) metaLine(gtx C, s *Scene, r *ThreadRow, in rowInk) D {
	col := in.of(SP.SecondaryLabel)
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}
	var kids []layout.FlexChild
	switch {
	case r.Branch != "":
		if r.Worktree {
			kids = append(kids, rigid(LucideW("folder-git-2", in.of(SP.IconMuted), 14)))
		}
		kids = append(kids, flexed(func(gtx C) D { return th.middleTruncate(gtx, t, r.Branch) }))
	case r.Scratch != "":
		kids = append(kids,
			rigid(machineIcon(r.MachineIcon, col, 12)),
			flexed(func(gtx C) D { return th.Text(gtx, t, r.Scratch) }))
	default:
		kids = append(kids, flexed(func(gtx C) D { return D{Size: gtx.Constraints.Min} }))
	}
	if r.Terminals > 0 {
		kids = append(kids, rigid(th.terminalButton(s, r, in)))
	}
	if r.PR != nil {
		kids = append(kids, rigid(func(gtx C) D { return th.prBadge(gtx, s, r, in) }))
	}
	kids = append(kids, rigid(func(gtx C) D { return th.trailingIcons(gtx, r, in) }))
	return RowGap(gtx, 6, kids...)
}

// trailingIcons are a row's machine icon (on a list mixing machines) and
// its provider stack, 4dp apart.
func (th *Theme) trailingIcons(gtx C, r *ThreadRow, in rowInk) D {
	var kids []layout.FlexChild
	if r.Remote {
		kids = append(kids, rigid(machineIcon(r.MachineIcon, in.of(alpha(SP.SidebarMutedForeground, 0xb3)), 14)))
	}
	if len(r.Providers) > 0 {
		kids = append(kids, rigid(func(gtx C) D { return th.providerStack(gtx, r.Providers, in) }))
	}
	return RowGap(gtx, 4, kids...)
}

// providerStack is the providers a thread has run on (SidebarProviderStack):
// earlier ones at 12dp, grey and 35%; the current one at 14dp and 60%, on
// top; 6dp apart.
func (th *Theme) providerStack(gtx C, providers []string, in rowInk) D {
	var kids []layout.FlexChild
	for i, p := range providers {
		p, cur := p, i == len(providers)-1
		kids = append(kids, rigid(func(gtx C) D {
			col := logoColor(p, SP.SecondaryLabel)
			if !cur {
				return providerMark(gtx, p, 12, in.of(alpha(grey(col), 0x59)))
			}
			return providerMark(gtx, p, 14, in.of(alpha(col, 0x99)))
		}))
	}
	return RowGap(gtx, 6, kids...)
}

// statusPulse is T3's status-pulse at now: opacity 1 to 40% of its 2s, 0.5
// from 50 to 90%, 1 again at the end, every row in phase. Its steps(6)
// applies to each keyframe interval, so the two ramps take six steps each.
// next is when the value changes.
func statusPulse(now time.Time) (k float32, next time.Time) {
	const period = 2000 // ms
	ms := now.UnixMilli()
	p := float64(ms%period) / period
	at := func(q float64) time.Time { return time.UnixMilli(ms - ms%period + int64(q*period)) }
	ramp := func(from, span float64) (float64, time.Time) {
		step := math.Floor((p - from) / span * 6)
		return step / 6, at(from + (step+1)*span/6)
	}
	switch {
	case p < 0.4:
		return 1, at(0.4)
	case p < 0.5:
		s, n := ramp(0.4, 0.1)
		return float32(1 - 0.5*s), n
	case p < 0.9:
		return 0.5, at(0.9)
	}
	s, n := ramp(0.9, 0.1)
	return float32(0.5 + 0.5*s), n
}

// terminalButton is the meta line's running terminal: 14dp in teal,
// pulsing. A click opens the thread with its terminal.
func (th *Theme) terminalButton(s *Scene, r *ThreadRow, in rowInk) W {
	return func(gtx C) D {
		col := Tailwind["teal-600"]
		if dark {
			col = alpha(Tailwind["teal-300"], 0xe6)
		}
		k, next := statusPulse(gtx.Now)
		gtx.Execute(op.InvalidateCmd{At: next})
		col.A = uint8(float32(col.A) * k)
		d := LucideW("terminal", in.of(col), 14)(gtx)
		if id := liveID(s, "terminal:"+r.ID); id != "" {
			th.hit(gtx, id, d.Size)
		}
		return d
	}
}

// prBadge is the meta line's pull request: its state's glyph at 12dp in
// its tone, then the number in the secondary label, 2dp on. It's T3's
// InlineButton: a click opens the pull request, and hovered the number
// underlines 2dp under its baseline.
func (th *Theme) prBadge(gtx C, s *Scene, r *ThreadRow, in rowInk) D {
	pr := r.PR
	id := liveID(s, "pr:"+r.ID)
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	icon, tone := "git-pull-request-arrow", SP.MutedForeground
	pick := func(light, darkName string, a uint8) color.NRGBA {
		if dark {
			return alpha(Tailwind[darkName], a)
		}
		return Tailwind[light]
	}
	switch pr.State {
	case "open":
		tone = pick("emerald-600", "emerald-300", 0xe6)
	case "draft":
		icon, tone = "git-pull-request-draft", pick("zinc-500", "zinc-400", 0xcc)
	case "merged":
		icon, tone = "git-merge", pick("violet-600", "violet-300", 0xe6)
	case "closed":
		icon, tone = "git-pull-request-closed", pick("red-600", "red-300", 0xe6)
	}
	col := in.of(SP.SecondaryLabel)
	d := RowGap(gtx, 2,
		rigid(LucideW(icon, in.of(tone), 12)),
		rigid(func(gtx C) D {
			d := th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}, pr.Text)
			if hovered {
				hline(gtx, col, d.Size.Y-d.Baseline+dp(gtx, 2), 0, d.Size.X)
			}
			return d
		}),
	)
	if id != "" {
		th.hit(gtx, id, d.Size)
	}
	return d
}

// slimRow is a snoozed or settled thread (§2.5, §3.8): 36dp, padded 0 by
// 10, 10dp gaps: the project's icon (grey at 40% until hovered), the
// title, and the slot with its age or wake time, which fades for the Wake
// or Un-settle button on hover.
func (th *Theme) slimRow(gtx C, s *Scene, r *ThreadRow, top int) D {
	w := gtx.Constraints.Max.X
	size := image.Pt(w, dp(gtx, slimH))
	x0 := dp(gtx, sideInset)
	rect := image.Rectangle{Min: image.Pt(x0, top), Max: image.Pt(x0+w, top+size.Y)}
	nav := "thread:" + r.ID
	hovered, focused, pop := th.rowTarget(gtx, s, nav, size, rect)
	defer pop()
	hot := hovered || r.Hover
	settled := r.Shelf == "settled"
	bg, ring := rowSurface(r, hot)
	in := rowInk{bg: bg, k: 1}
	paintRow(gtx, size, in, ring, focused)
	lit := hot || focused
	// At rest a settled row is all in the secondary label.
	rest := func(c color.NRGBA) color.NRGBA {
		if settled && !r.Active && !lit {
			return SP.SecondaryLabel
		}
		return c
	}
	dim := float32(0)
	if !r.Active || settled {
		dim = 1 - th.transition(gtx, nav+".icon", lit, 150*time.Millisecond)
	}
	shown := th.transition(gtx, nav+".slot", hot, 150*time.Millisecond)
	th.hoverCard(gtx, s, r, rect, hot)
	defer op.Offset(image.Pt(dp(gtx, 10), 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(w-2*dp(gtx, 10), size.Y))
	var kids []layout.FlexChild
	if r.Project != "" {
		kids = append(kids, rigid(func(gtx C) D { return th.projectIcon(gtx, r.Icon, 16, in.bg, dim) }))
	}
	if r.Draft {
		kids = append(kids, rigid(LucideW("square-pen", in.of(rest(SP.WarningForeground)), 12)))
	}
	kids = append(kids, flexed(func(gtx C) D { return th.title(gtx, s, r, in, lit, true) }))
	if r.Pinned {
		kids = append(kids, rigid(func(gtx C) D {
			return LucideW("pin", in.of(rest(alpha(SP.MutedForeground, 0xa6))), 12)(gtx)
		}))
	}
	if r.Terminals > 0 {
		kids = append(kids, rigid(th.terminalButton(s, r, in)))
	}
	kids = append(kids, rigid(func(gtx C) D { return th.slimSlot(gtx, s, r, in, rest, hot, shown) }))
	RowGap(g, 10, kids...)
	return D{Size: size}
}

// slimSlot is a slim row's trailing slot: 24dp, at least 32 wide, right
// aligned. Its age, wake time or Woke label fades out on hover (Woke
// stays), as the Wake or Un-settle button fades in, 4dp into the padding.
// While the jump keys show, it shows only the row's.
func (th *Theme) slimSlot(gtx C, s *Scene, r *ThreadRow, in rowInk, rest func(color.NRGBA) color.NRGBA, hot bool, shown float32) D {
	h := dp(gtx, 24)
	if th.in.jumping && r.Jump > 0 {
		g := gtx
		g.Constraints.Min = image.Point{}
		m := op.Record(gtx.Ops)
		d := th.jumpKey(r, in)(g)
		call := m.Stop()
		w := max(d.Size.X, dp(gtx, 32))
		defer op.Offset(image.Pt(w-d.Size.X, (h-d.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return D{Size: image.Pt(w, h)}
	}
	t := Txt{Size: 12, LineH: 16.0 / 12, Color: in.of(rest(SP.SecondaryLabel)), MaxLines: 1}
	fade := func(c color.NRGBA, k float32) color.NRGBA { return over(alpha(c, uint8(k*255+0.5)), in.bg) }
	label := func(gtx C) D {
		k := 1 - shown
		if r.Woke {
			k = 1
		}
		switch {
		case r.Shelf == "snoozed" && r.Wake != "":
			col := fade(in.of(rest(SP.InfoForeground)), k)
			return RowGap(gtx, 4,
				rigid(LucideW("alarm-clock", col, 12)),
				rigid(func(gtx C) D { return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}, r.Wake) }))
		case r.Woke:
			col := in.of(rest(SP.WarningForeground))
			return th.wokeButton(gtx, s, r, func(gtx C) D {
				return RowGap(gtx, 4,
					rigid(LucideW("alarm-clock", col, 12)),
					rigid(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: col, MaxLines: 1}, "Woke")
					}))
			})
		}
		t.Color = fade(t.Color, k)
		return th.Text(gtx, t, r.Age)
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints.Min = image.Point{}
	ld := label(g)
	call := m.Stop()
	w := max(ld.Size.X, dp(gtx, 32))
	func() {
		defer op.Offset(image.Pt(w-ld.Size.X, (h-ld.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
	}()
	if !r.CanSettle || (r.Shelf != "snoozed" && r.Shelf != "settled") {
		return D{Size: image.Pt(w, h)}
	}
	// The Wake or Un-settle button: padded 0 by 6, its icon 1dp high (mb-px),
	// over the label's end and 4dp into the padding; a woke row keeps its
	// label, and the button follows it.
	id, icon, sz, tip := "wake:"+r.ID, "alarm-clock-off", float32(12), ""
	if r.Shelf == "settled" {
		id, icon, sz, tip = "unsettle:"+r.ID, "undo-2", 14, "Un-settle thread"
	}
	id = liveID(s, id)
	bw := dp(gtx, sz+12)
	total, x := w, w-bw+dp(gtx, 4)
	if r.Woke && hot {
		total, x = w+bw-dp(gtx, 4), w
	}
	defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
	col := SP.MutedForeground
	if id != "" {
		if c, _ := th.events(gtx, id); c.Hovered() {
			col = SP.Foreground
		}
	}
	func() {
		defer op.Offset(image.Pt(dp(gtx, 6), (h-dp(gtx, sz))/2-max(1, dp(gtx, 1))/2)).Push(gtx.Ops).Pop()
		LucideW(icon, fade(in.of(col), shown), sz)(gtx)
	}()
	if id != "" && shown > 0 {
		th.hit(gtx, id, image.Pt(bw, h))
		th.listTip(gtx, id, tip, image.Pt(bw, h))
	}
	return D{Size: image.Pt(total, h)}
}

// shelfHeader is a shelf's header (CollapsibleSectionHeader, §3.9): 32dp,
// 2dp in from the rows, padded 0 by 8, 8dp gaps, 12/16 medium in the
// sidebar's muted foreground: the label, the count (400) and the chevron,
// which turns from -90° folded to 0° open over 150ms. No hover fill.
func (th *Theme) shelfHeader(gtx C, s *Scene, sh *Shelf) D {
	w := gtx.Constraints.Max.X
	in2 := dp(gtx, 2)
	size := image.Pt(w-2*in2, dp(gtx, shelfH))
	defer op.Offset(image.Pt(in2, 0)).Push(gtx.Ops).Pop()
	nav := "shelf:" + sh.ID
	_, focused, pop := th.rowTarget(gtx, s, nav, size, image.Rectangle{})
	defer pop()
	if focused {
		strokeRect(gtx, over(alpha(SP.Ring, 0xb3), SP.Sidebar), image.Rectangle{Max: size}, dp(gtx, 6), float32(2*max(1, dp(gtx, 1))))
	}
	col := SP.SidebarMutedForeground
	open := th.transition(gtx, nav+".chevron", sh.Open, 150*time.Millisecond)
	defer op.Offset(image.Pt(dp(gtx, 8), 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(size.X-2*dp(gtx, 8), size.Y))
	return RowGap(g, 8,
		rigid(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: col, MaxLines: 1}, sh.Label)
		}),
		flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 1)} }),
		rigid(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}, strconv.Itoa(sh.Count))
		}),
		rigid(func(gtx C) D {
			s := float32(dp(gtx, 12))
			c := f32.Pt(s/2, s/2)
			defer op.Affine(f32.Affine2D{}.Rotate(c, -(1-open)*math.Pi/2)).Push(gtx.Ops).Pop()
			return LucideW("chevron-down", col, 12)(gtx)
		}),
	)
}

// moreRow is the Settled shelf's "Show N more": 36dp, padded 0 by 10, a
// 16dp chevron and 14/20 text 10dp apart in the sidebar's muted
// foreground, the foreground on the row hover fill.
func (th *Theme) moreRow(gtx C, s *Scene, n int) D {
	w := gtx.Constraints.Max.X
	size := image.Pt(w, dp(gtx, slimH))
	hovered, focused, pop := th.rowTarget(gtx, s, "settled.more", size, image.Rectangle{})
	defer pop()
	col := SP.SidebarMutedForeground
	bg := SP.Sidebar
	if hovered {
		bg = over(SP.SidebarRowHover, SP.Sidebar)
		col = SP.SidebarForeground
	}
	paintRow(gtx, size, rowInk{bg: bg, k: 1}, color.NRGBA{}, focused)
	defer op.Offset(image.Pt(dp(gtx, 10), 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(w-2*dp(gtx, 10), size.Y))
	return RowGap(g, 10,
		rigid(LucideW("chevron-down", col, 16)),
		rigid(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: col, MaxLines: 1}, fmt.Sprintf("Show %d more", n))
		}),
	)
}

// emptyState is what an empty list says (§2.9, §3.13): a column centred in
// 24 by 8 of padding, its message and then, 8dp below, its button.
func (th *Theme) emptyState(gtx C, s *Scene, e *SidebarEmpty) D {
	w := gtx.Constraints.Max.X
	y := dp(gtx, 24)
	func() {
		defer op.Offset(image.Pt(dp(gtx, 8), y)).Push(gtx.Ops).Pop()
		y += th.emptyText(gtx, e, w-2*dp(gtx, 8)).Size.Y
	}()
	if e.Button != "" {
		defer op.Offset(image.Pt(0, y+dp(gtx, 8))).Push(gtx.Ops).Pop()
		th.emptyButton(gtx, liveID(s, "filter.clear"), e.Button, w)
	}
	return D{Size: image.Pt(w, gtx.Constraints.Max.Y)}
}

// emptyText is the empty list's message, w wide: 12/16 in the muted
// foreground at 60%, centred, wrapping when it's long. It takes whole 16dp
// lines, as CSS's line boxes do; Gio sizes the first by its glyphs.
func (th *Theme) emptyText(gtx C, e *SidebarEmpty, w int) D {
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, 1<<20)}
	d := th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: over(alpha(SP.MutedForeground, 0x99), SP.Sidebar), Align: text.Middle}, e.Text)
	lh := dp(gtx, 16)
	d.Size.Y = (d.Size.Y + lh - 1) / lh * lh
	return d
}

// emptyButton is the empty list's button, centred in w: 26dp, padded 4 by
// 10 inside a 1dp border in the sidebar border, radius 6, 11/16 medium in
// the sidebar's muted foreground; hovered, the row hover fill and the
// foreground, eased in over 150ms.
func (th *Theme) emptyButton(gtx C, id, label string, w int) {
	hovered := false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	k := th.transition(gtx, "empty.hover", hovered, 150*time.Millisecond)
	col := lerpCol(SP.SidebarMutedForeground, SP.SidebarForeground, k)
	bg := lerpCol(SP.Sidebar, over(SP.SidebarRowHover, SP.Sidebar), k)
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	ld := th.Text(g, Txt{Size: 11, LineH: 16.0 / 11, Font: UI(font.Medium), Color: col, MaxLines: 1}, label)
	call := m.Stop()
	px := max(1, dp(gtx, 1))
	r := image.Rect(0, 0, ld.Size.X+2*(dp(gtx, 10)+px), dp(gtx, 26))
	defer op.Offset(image.Pt((w-r.Dx())/2, 0)).Push(gtx.Ops).Pop()
	fill(gtx, bg, r, dp(gtx, 6))
	strokeRect(gtx, over(SP.SidebarBorder, bg), r, dp(gtx, 6), float32(px))
	func() {
		defer op.Offset(image.Pt(px+dp(gtx, 10), (r.Dy()-ld.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
	}()
	if id != "" {
		th.hit(gtx, id, r.Max)
	}
}

// listNote is the search's status line: 12/16 in the sidebar's muted
// foreground, centred in 24 by 8 of padding.
func (th *Theme) listNote(gtx C, note string) D {
	w := gtx.Constraints.Max.X
	defer op.Offset(image.Pt(0, dp(gtx, 24))).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(w, dp(gtx, 16)))
	th.Text(g, Txt{Size: 12, LineH: 16.0 / 12, Color: SP.SidebarMutedForeground, MaxLines: 1, Align: text.Middle}, note)
	return D{Size: image.Pt(w, dp(gtx, 64))}
}

// resultRow is a search result (§2.6, §3.11): 36dp, padded 4 by 10, 10dp
// gaps: the project's icon, the title and the age, 32dp wide in the muted
// foreground at 55%; a result found by its messages grows to 44dp for the
// matching one under the title. The highlighted result (keyboard or
// pointer) and the open thread take the active row fill. A settled
// thread's title is quieter, 400 in the secondary label at 70%, until it's
// lit.
func (th *Theme) resultRow(gtx C, s *Scene, r *ThreadRow, i, top int) D {
	w := gtx.Constraints.Max.X
	size := image.Pt(w, dp(gtx, slimH))
	if r.Excerpt != nil {
		size.Y = dp(gtx, excerptH)
	}
	id := liveID(s, "result:"+strconv.Itoa(i))
	hovered := false
	var pop func()
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
		if hovered && th.in.resultHover != i+1 {
			th.push(Action{ID: id, Kind: "hover"})
		}
		if hovered {
			th.in.resultHover = i + 1
		} else if th.in.resultHover == i+1 {
			th.in.resultHover = 0
		}
		pop = th.area(gtx, id, size, pointer.CursorPointer)
	}
	lit := i == s.Sidebar.Highlight || r.Active
	bg := SP.Sidebar
	switch {
	case lit:
		bg = over(SP.SidebarRowActive, SP.Sidebar)
	case hovered:
		bg = over(SP.SidebarRowHover, SP.Sidebar)
	}
	if bg != SP.Sidebar {
		fill(gtx, bg, image.Rectangle{Max: size}, dp(gtx, 6))
	}
	settled := r.Shelf == "settled"
	title := Txt{Size: 14, LineH: 20.0 / 14, Font: UI(font.Medium), Color: over(SP.Foreground, bg), MaxLines: 1}
	if settled && !lit && !hovered {
		title.Font, title.Color = UI(font.Normal), over(alpha(SP.SecondaryLabel, 0xb3), bg)
	}
	dim := float32(0)
	if settled && !lit && !hovered {
		dim = 1
	}
	func() {
		defer op.Offset(image.Pt(dp(gtx, 10), 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(w-2*dp(gtx, 10), size.Y))
		line := func(gtx C) D {
			gtx.Constraints = layout.Exact(image.Pt(gtx.Constraints.Max.X, dp(gtx, 20)))
			return RowGap(gtx, 10,
				flexed(func(gtx C) D { return th.Text(gtx, title, r.Title) }),
				rigid(func(gtx C) D {
					return FixedW(gtx, 32, func(gtx C) D {
						return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: over(alpha(SP.MutedForeground, 0x8c), bg), MaxLines: 1, Align: text.End}, r.Age)
					})
				}),
			)
		}
		var kids []layout.FlexChild
		if r.Project != "" {
			kids = append(kids, rigid(func(gtx C) D { return th.projectIcon(gtx, r.Icon, 16, bg, dim) }))
		}
		kids = append(kids, flexed(func(gtx C) D {
			if r.Excerpt == nil {
				return line(gtx)
			}
			w := gtx.Constraints.Max.X
			line(gtx)
			defer op.Offset(image.Pt(0, dp(gtx, 20))).Push(gtx.Ops).Pop()
			gtx.Constraints = layout.Exact(image.Pt(w, dp(gtx, 16)))
			th.excerpt(gtx, r.Excerpt, bg)
			return D{Size: image.Pt(w, dp(gtx, 36))}
		}))
		RowGap(g, 10, kids...)
	}()
	x0 := dp(gtx, sideInset)
	th.hoverCard(gtx, s, r, image.Rectangle{Min: image.Pt(x0, top), Max: image.Pt(x0+w, top+size.Y)}, hovered)
	if pop != nil {
		pop()
	}
	return D{Size: size}
}

// excerpt is a search result's matching message on one 16dp line, 12dp in
// the muted foreground at 85% (ThreadSearchMatch.tsx): "You:" in the info
// foreground or "Agent:" in the success foreground, a space, then the
// servers' snippet with the query's words 600 in the foreground. Its runs
// of spaces and line breaks set as one space, as white-space: nowrap has
// them, and where the line is cut it ends in "…".
func (th *Theme) excerpt(gtx C, e *Excerpt, bg color.NRGBA) D {
	size := gtx.Constraints.Max
	base := Txt{Size: 12, LineH: 16.0 / 12, Color: over(faded(SP.MutedForeground, 0.85), bg), MaxLines: 1}
	who := base
	who.Color = over(SP.SuccessForeground, bg)
	name := "Agent:"
	if e.User {
		who.Color, name = over(SP.InfoForeground, bg), "You:"
	}
	mark := base
	mark.Font, mark.Color = UI(font.SemiBold), over(SP.Foreground, bg)
	type part struct {
		t Txt
		s []rune
	}
	parts := []part{{who, []rune(name)}, {base, []rune(" ")}}
	marked := func(i int) bool {
		for _, m := range e.Marks {
			if i >= m[0] && i < m[1] {
				return true
			}
		}
		return false
	}
	space := true
	for i, c := range []rune(e.Snippet) {
		if strings.ContainsRune(" \t\n\r\f", c) {
			if space {
				continue
			}
			c, space = ' ', true
		} else {
			space = false
		}
		t := base
		if marked(i) {
			t = mark
		}
		if last := &parts[len(parts)-1]; last.t == t {
			last.s = append(last.s, c)
		} else {
			parts = append(parts, part{t, []rune{c}})
		}
	}
	// widthTo is how wide the line's first n characters set.
	total := 0
	for _, p := range parts {
		total += len(p.s)
	}
	widthTo := func(n int) int {
		x := 0
		for _, p := range parts {
			k := min(n, len(p.s))
			if k > 0 {
				x += th.textWidth(gtx, p.t, string(p.s[:k]))
			}
			if n -= k; n == 0 {
				break
			}
		}
		return x
	}
	n, dots := total, ""
	if widthTo(total) > size.X {
		dots = "…"
		room := size.X - th.textWidth(gtx, base, dots)
		lo, hi := 0, total
		for lo < hi {
			if mid := (lo + hi + 1) / 2; widthTo(mid) <= room {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		n = lo
	}
	m := op.Record(gtx.Ops)
	x, h := 0, 0
	set := func(t Txt, s string) {
		if s == "" {
			return
		}
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(1<<20, size.Y)}
		d := th.Text(g, t, s)
		x, h = x+d.Size.X, max(h, d.Size.Y)
	}
	for _, p := range parts {
		k := min(n, len(p.s))
		set(p.t, string(p.s[:k]))
		if n -= k; n == 0 {
			break
		}
	}
	set(base, dots)
	call := m.Stop()
	defer op.Offset(image.Pt(0, (size.Y-h)/2)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return D{Size: size}
}

// ---------- icons ----------

// projectColor is one of T3's project icon colours: Tailwind's 600 in
// light, 400 in dark.
func projectColor(name string) color.NRGBA {
	shade := "-600"
	if dark {
		shade = "-400"
	}
	if c, ok := Tailwind[name+shade]; ok {
		return c
	}
	return Tailwind["gray"+shade]
}

// grey is CSS grayscale(1) of c.
func grey(c color.NRGBA) color.NRGBA {
	l := uint8(0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B) + 0.5)
	return color.NRGBA{R: l, G: l, B: l, A: c.A}
}

// projectIcon draws a project's icon sizeDp square on bg: its monogram,
// mono 700 stretched to 6 (one glyph) or 12 (two) of the tile's 16 units,
// on its colour at 14% with 25% corners (ProjectMonogram.tsx); or its
// lucide icon. dim runs to 1 for the quiet rows' grey at 40%.
func (th *Theme) projectIcon(gtx C, ic ProjectIcon, sizeDp float32, bg color.NRGBA, dim float32) D {
	col := projectColor(ic.Color)
	if ic.Color == "" && ic.Lucide == "" && ic.Text == "" {
		col = SP.IconMuted
	}
	ink := func(c color.NRGBA) color.NRGBA {
		c = lerpCol(c, grey(c), dim)
		c.A = uint8(float32(c.A)*(1-0.6*dim) + 0.5)
		return over(c, bg)
	}
	s := dp(gtx, sizeDp)
	if ic.Lucide != "" || ic.Text == "" {
		// A lucide icon tern hasn't is folder-code, as T3 draws one it can't
		// find.
		name := ic.Lucide
		if _, ok := lucide[name]; !ok {
			name = "folder-code"
		}
		return LucideW(name, ink(col), sizeDp)(gtx)
	}
	tile := over(alpha(col, 0x24), bg)
	fill(gtx, ink(tile), image.Rect(0, 0, s, s), s/4)
	k := float32(s) / 16
	t := Txt{Size: unit.Sp(8.25 * k / gtx.Metric.PxPerSp), Font: font.Font{Typeface: Mono().Typeface, Weight: font.Bold}, Color: ink(col), MaxLines: 1}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(4*s, 2*s)}
	td := th.Text(g, t, ic.Text)
	call := m.Stop()
	span := float32(6)
	if utf8.RuneCountInString(ic.Text) > 1 {
		span = 12
	}
	if td.Size.X > 0 {
		sx := span * k / float32(td.Size.X)
		x := (8 - span/2) * k
		y := 10.8*k - float32(td.Size.Y-td.Baseline)
		defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(sx, 1)).Offset(f32.Pt(x, y))).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
	}
	return D{Size: image.Pt(s, s)}
}

// machineIcon is a machine's icon (EnvironmentMachineIcon): lucide's
// server, cloud, monitor and laptop, and T3's own Linux, mini PC and
// workstation glyphs.
func machineIcon(kind string, col color.NRGBA, sizeDp float32) W {
	switch kind {
	case "cloud", "laptop":
		return LucideW(kind, col, sizeDp)
	case "desktop":
		return LucideW("monitor", col, sizeDp)
	case "linux":
		return LinuxMark(col, sizeDp)
	case "mac-mini", "mac-studio":
		return LucideW(kind, col, sizeDp)
	}
	return LucideW("server", col, sizeDp)
}

// ---------- the branch ----------

// splitMiddle is T3's splitForMiddleTruncate: where to cut value so its
// end stays whole. The tail is 10 code points, or the last path segment if
// that's at most 16, moved to start on a word (after - _ . or /). With no
// cut worth making, tail is "".
func splitMiddle(value string) (head, tail string) {
	chars := []rune(value)
	n := len(chars)
	keep := 10
	if slash := strings.LastIndex(value, "/"); slash > 0 && slash < len(value)-1 {
		if seg := utf8.RuneCountInString(value[slash+1:]); seg <= 16 {
			keep = seg
		}
	}
	if keep <= 0 || n <= keep+4 {
		return value, ""
	}
	brk := func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == '/' }
	cut := n - keep
	if !brk(chars[cut-1]) {
		snapped := 0
		for i := cut + 1; i <= n-4; i++ {
			if brk(chars[i-1]) {
				snapped = i
				break
			}
		}
		if snapped == 0 {
			for i := cut - 1; i >= n-16 && i > 4; i-- {
				if brk(chars[i-1]) {
					snapped = i
					break
				}
			}
		}
		if snapped > 0 {
			cut = snapped
		}
	}
	return string(chars[:cut]), string(chars[cut:])
}

// middleTruncate draws value in t, keeping its end (MiddleTruncate): when it
// doesn't fit, the head is cut after its last whole glyph and "…" sits
// flush before the tail, which is never cut.
func (th *Theme) middleTruncate(gtx C, t Txt, value string) D {
	w := gtx.Constraints.Max.X
	t.MaxLines = 1
	width := func(s string) int { return th.textWidth(gtx, t, s) }
	head, tail := splitMiddle(value)
	if tail == "" || width(value) <= w {
		return th.Text(gtx, t, value)
	}
	tw := width("…" + tail)
	hr := []rune(head)
	lo, hi := 0, len(hr)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if width(string(hr[:mid]))+tw <= w {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return th.Text(gtx, t, string(hr[:lo])+"…"+tail)
}

// textWidth is how wide s sets in t, on one line.
func (th *Theme) textWidth(gtx C, t Txt, s string) int {
	m := op.Record(gtx.Ops)
	gtx.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	d := th.Text(gtx, t, s)
	m.Stop()
	return d.Size.X
}

// ---------- the hover card ----------

// hoverCard is a row's hover card (ThreadHoverCard.tsx, §3.15), shown once
// the pointer has rested on the row for 150ms: T3's glass tooltip 4dp
// right of the row, level with its top, kept inside the window. It grows
// from 98% and fades in over 150ms from the side facing the row, and back
// out as it closes; opening on the next row within 400ms of the last
// closing, it and the card it replaces change at once (Base UI's
// data-instant). rect is the row's box in the window.
func (th *Theme) hoverCard(gtx C, s *Scene, r *ThreadRow, rect image.Rectangle, hot bool) {
	key := "card:" + r.ID
	open, instant := r.Hover, true
	if s.Live {
		switch {
		case hot && th.in.cardRow != r.ID:
			th.in.cardRow, th.in.cardSince, th.in.cardInstant = r.ID, gtx.Now, false
			if gtx.Now.Sub(th.in.cardShut) < listTipInstant {
				th.in.cardSince, th.in.cardInstant = gtx.Now.Add(-listTipDelay), true
			}
		case !hot && th.in.cardRow == r.ID:
			th.in.cardRow, th.in.cardShut = "", gtx.Now
		}
		left := listTipDelay - gtx.Now.Sub(th.in.cardSince)
		if hot && left > 0 {
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
		}
		open = hot && !th.in.tipBusy && left <= 0
		instant = (open && th.in.cardInstant) || (!open && th.in.cardRow != "" && th.in.cardRow != r.ID)
	}
	if th.in.fades[key] == nil {
		if !open {
			return
		}
		// A card starts closed, so its first opening plays too.
		if th.in.fades == nil {
			th.in.fades = map[string]*fade{}
		}
		th.in.fades[key] = &fade{}
	}
	var k float32
	if instant {
		th.in.fades[key] = &fade{on: open, from: level(open)}
		k = level(open)
	} else {
		k = th.transition(gtx, key, open, 150*time.Millisecond)
	}
	if k == 0 {
		delete(th.in.fades, key)
		return
	}
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(dp(gtx, 320), 1<<20)}
	cd := th.cardBody(g, r)
	call := m.Stop()
	// Within the window: shifted up from its bottom, as T3's collision
	// padding does.
	win := th.in.winSize
	dy := 0
	if bottom := rect.Min.Y + cd.Size.Y + dp(gtx, 5); win.Y > 0 && bottom > win.Y {
		dy = max(-rect.Min.Y+dp(gtx, 5), win.Y-bottom)
	}
	m2 := op.Record(gtx.Ops)
	op.Offset(image.Pt(rect.Dx()+dp(gtx, 4), dy)).Add(gtx.Ops)
	// It grows from its left edge, level with the row's middle.
	sc := 0.98 + 0.02*k
	origin := f32.Pt(0, float32(min(cd.Size.Y, rect.Dy()/2-dy)))
	op.Affine(f32.Affine2D{}.Scale(origin, f32.Pt(sc, sc))).Add(gtx.Ops)
	fd := paint.PushOpacity(gtx.Ops, k)
	call.Add(gtx.Ops)
	fd.Pop()
	op.Defer(gtx.Ops, m2.Stop())
}

// cardBody is the hover card: the glass tooltip surface (its fill alone,
// without the blur) in a 1dp border, radius 6, the tooltip's 4 by 8 padding
// around the card's 8 by 4; the title in 12/15 medium, then the lines 6dp
// apart, each a 12dp icon and 12/16 text in the foreground at 75%, 8dp apart.
func (th *Theme) cardBody(gtx C, r *ThreadRow) D {
	bg := over(alpha(P.Popover, 0xd5), P.Background)
	muted := over(P.MutedForeground, bg)
	line := func(icon W, text string, col color.NRGBA) layout.FlexChild {
		return rigid(func(gtx C) D {
			return pad(0, 0, 0, 2).Layout(gtx, func(gtx C) D {
				return RowGap(gtx, 8,
					rigid(icon),
					flexed(func(gtx C) D {
						return th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: col, MaxLines: 1}, text)
					}))
			})
		})
	}
	soft := over(alpha(P.Foreground, 0xbf), bg)
	var lines []layout.FlexChild
	gap := func() {
		if len(lines) > 0 {
			lines = append(lines, rigid(Spacer(0, 6)))
		}
	}
	add := func(icon W, text string, col color.NRGBA) {
		gap()
		lines = append(lines, line(icon, text, col))
	}
	if r.Project != "" {
		add(func(gtx C) D { return th.projectIcon(gtx, r.Icon, 12, bg, 0) }, r.Project, soft)
	}
	if r.Machine != "" {
		add(machineIcon(r.MachineIcon, muted, 12), r.Machine, soft)
	}
	if r.Branch != "" {
		gap()
		lines = append(lines, rigid(func(gtx C) D {
			return pad(0, 0, 0, 2).Layout(gtx, func(gtx C) D {
				return RowGap(gtx, 8,
					rigid(LucideW("git-branch", muted, 12)),
					flexed(func(gtx C) D { return th.middleTruncate(gtx, Txt{Size: 12, LineH: 16.0 / 12, Color: soft}, r.Branch) }))
			})
		}))
	}
	if r.Provider != "" {
		add(func(gtx C) D {
			return providerMark(gtx, r.Provider, 12, over(alpha(grey(logoColor(r.Provider, muted)), 0x99), bg))
		}, r.Model, soft)
	}
	if r.Handoff != "" {
		add(LucideW("arrow-right-left", muted, 12), r.Handoff, soft)
	}
	if r.Terminals > 0 {
		teal := Tailwind["teal-600"]
		if dark {
			teal = over(alpha(Tailwind["teal-300"], 0xe6), bg)
		}
		procs := "processes"
		if r.Terminals == 1 {
			procs = "process"
		}
		add(LucideW("terminal", teal, 12), fmt.Sprintf("%d terminal %s running", r.Terminals, procs), soft)
	}
	if r.Error != "" {
		col := P.Error
		if r.Limited {
			col = P.Warning
		}
		add(LucideW("circle-alert", col, 12), r.Error, col)
	}
	m := op.Record(gtx.Ops)
	d := pad(1+4+8, 1+8+4, 1+4+8, 1+8+4).Layout(gtx, func(gtx C) D {
		kids := []layout.FlexChild{rigid(func(gtx C) D {
			return th.Text(gtx, Txt{Size: 12, LineH: 15.0 / 12, Font: UI(font.Medium), Color: over(P.Foreground, bg), MaxLines: 1}, r.Title)
		})}
		if len(lines) > 0 {
			kids = append(kids, rigid(Spacer(0, 8)), rigid(func(gtx C) D { return Col(gtx, lines...) }))
		}
		return Col(gtx, kids...)
	})
	call := m.Stop()
	rr := image.Rectangle{Max: d.Size}
	rad := dp(gtx, 6)
	shadow(gtx, rr, rad, []boxShadow{{0, 20, 25, -5, alpha(rgb(0), 0x40)}, {0, 8, 10, -6, alpha(rgb(0), 0x40)}})
	fill(gtx, bg, rr, rad)
	strokeRect(gtx, over(alpha(P.Foreground, 0x1a), bg), rr, rad, float32(max(1, dp(gtx, 1))))
	call.Add(gtx.Ops)
	return d
}

// listTip is a tooltip inside the list: T3's, on top, after the list's
// 150ms rest. While one shows, the row's hover card stands aside.
func (th *Theme) listTip(gtx C, id, tip string, size image.Point) {
	if c := th.in.clicks[id]; c != nil && c.Hovered() && tip != "" {
		th.in.tipBusy = true
	}
	th.tipDelay(gtx, id, tip, size, true, 0, listTipDelay)
}

// ---------- the snooze menu ----------

// snoozeMenu is the menu a card's clock opens (SnoozeMenuButton): T3's
// menu 4dp under the clock (trigger is its size, at in the window), its
// right edge on the clock's, as wide as its widest row and at least 160dp;
// a row per preset with its wake time at the end in the secondary label.
// Like T3's positioner it shifts along to stay 5dp inside the window, and
// goes above the clock when there's no room below.
func (th *Theme) snoozeMenu(gtx C, s *Scene, trigger image.Point, at image.Rectangle) {
	sm := s.Sidebar.Snooze
	th.deferBelow(gtx, image.Point{}, func(gtx C) D {
		// Inside the 1dp border and 4dp of padding.
		w := dp(gtx, 160-2*5)
		for _, c := range sm.Choices {
			w = max(w, th.snoozeChoiceWidth(gtx, c))
		}
		m := op.Record(gtx.Ops)
		d := th.popUp(gtx, s, "snooze:"+sm.Row, nil, func(gtx C) D {
			return popupFrame(gtx, func(gtx C) D {
				return pad(5, 5, 5, 5).Layout(gtx, func(gtx C) D {
					var kids []layout.FlexChild
					for _, c := range sm.Choices {
						c := c
						kids = append(kids, rigid(th.menuItem(liveID(s, "snooze.pick:"+c.ID), func(gtx C) D {
							return th.snoozeChoice(gtx, c, w)
						})))
					}
					return Col(gtx, kids...)
				})
			})
		})
		call := m.Stop()
		win, edge, gap := th.in.winSize, dp(gtx, 5), dp(gtx, 4)
		x := trigger.X - d.Size.X
		if left := at.Max.X - d.Size.X; left < edge {
			x += edge - left
		} else if win.X > 0 && at.Max.X > win.X-edge {
			x -= at.Max.X - (win.X - edge)
		}
		y := trigger.Y + gap
		if win.Y > 0 && at.Max.Y+gap+d.Size.Y > win.Y-edge && at.Min.Y-gap-d.Size.Y >= edge {
			y = -gap - d.Size.Y
		}
		defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return d
	})
}

// snoozeChoice is a snooze menu row w wide: T3's menu item, 28dp, 14/20
// padded 4 by 8, the label and, at the end, its detail.
func (th *Theme) snoozeChoice(gtx C, c MenuItem, w int) D {
	gtx.Constraints = layout.Exact(image.Pt(w, dp(gtx, 28)))
	return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
		return Row(gtx,
			flexed(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.PopoverForeground, MaxLines: 1}, c.Label)
			}),
			rigid(func(gtx C) D {
				return th.Text(gtx, Txt{Size: 14, LineH: 20.0 / 14, Color: P.SecondaryLabel, MaxLines: 1}, c.Detail)
			}),
		)
	})
}

// snoozeChoiceWidth is what a snooze menu row needs: its label, then the
// row's 8dp gap and the detail's 16dp of padding, then the detail, all
// inside 8dp each side (MenuItemDetail: ms-auto ps-4).
func (th *Theme) snoozeChoiceWidth(gtx C, c MenuItem) int {
	t := Txt{Size: 14, LineH: 20.0 / 14}
	return dp(gtx, 8) + th.textWidth(gtx, t, c.Label) + dp(gtx, 8+16) + th.textWidth(gtx, t, c.Detail) + dp(gtx, 8)
}

// ---------- the footer ----------

// sidebarFooter is T3's footer (SidebarChrome.tsx): padded 4 8 8, a row
// of 32dp icon buttons 4dp apart, then the undo notice in what's left. T3's
// buttons are Settings, Pull requests and Usage; tern's are the machines
// and the account, which open their menus above.
func (th *Theme) sidebarFooter(gtx C, s *Scene) {
	x := dp(gtx, sideInset)
	button := func(id, icon, tip string, menu *Traits) {
		defer op.Offset(image.Pt(x, dp(gtx, 4))).Push(gtx.Ops).Pop()
		id = liveID(s, id)
		th.footerButton(gtx, id, icon, tip, menu != nil)
		if menu == nil {
			th.popIn(gtx, id, false, nil, nil)
		} else {
			th.deferAbove(gtx, image.Pt(0, -dp(gtx, 1)), func(gtx C) D {
				return th.popUp(gtx, s, id, nil, func(gtx C) D { return th.traitsMenu(gtx, s, id, menu) })
			})
		}
		x += dp(gtx, 32+4)
	}
	button("machines", "server", "Machines", s.Machines)
	button("account", "circle-user", "Account", s.Account)
	// The row's 32dp centre the notice's 24.
	defer op.Offset(image.Pt(x, dp(gtx, 4+4))).Push(gtx.Ops).Pop()
	th.undoNotice(gtx, s, gtx.Constraints.Max.X-dp(gtx, sideInset)-x)
}

// undoNotice is the footer's undo notice (SidebarThreadUndoNotice.tsx), w
// wide: 24dp of 11/24 in the sidebar's muted foreground, right-aligned,
// "Settled," and then T3's InlineButton "<shortcut> to undo", 500 in the
// foreground and underlined while hovered. It fades in over 200ms on
// Tailwind's ease-out, and out with its last text when the notice ends.
// Short of room the summary goes, then the button's label for its icon.
func (th *Theme) undoNotice(gtx C, s *Scene, w int) {
	on := s.Sidebar.Undo != ""
	if on {
		th.in.undoText = s.Sidebar.Undo
	}
	f := th.in.undoFade
	if f == nil {
		f = &fade{on: on, from: level(on)}
		th.in.undoFade = f
	}
	shown := func() float32 {
		to := level(f.on)
		if f.from == to {
			return to
		}
		return f.from + (to-f.from)*easeOutCSS(since(gtx.Now, f.at, 200*time.Millisecond))
	}
	if f.on != on {
		f.from, f.on, f.at = shown(), on, gtx.Now
	}
	k := shown()
	switch {
	case !s.Live:
		// A mockup is one frame: it shows the notice settled.
		k = level(on)
	case k != level(on):
		gtx.Execute(op.InvalidateCmd{})
	}
	if k == 0 {
		return
	}
	defer paint.PushOpacity(gtx.Ops, k).Pop()
	id := liveID(s, "undo")
	hovered := false
	if id != "" && on {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	h := dp(gtx, 24)
	summary := Txt{Size: 11, LineH: 24.0 / 11, Color: SP.SidebarMutedForeground, MaxLines: 1}
	label := Txt{Size: 11, LineH: 24.0 / 11, Font: UI(font.Medium), Color: SP.Foreground, MaxLines: 1}
	text, undo := th.in.undoText+",", Shortcut("Z")+" to undo"
	sw, lw, gap := th.textWidth(gtx, summary, text), th.textWidth(gtx, label, undo), dp(gtx, 4)
	// Right to left: the button, then the summary before it.
	at := func(x int, draw func(gtx C) D) D {
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(w-x, h))
		return Row(g, rigid(draw))
	}
	bx := w - lw
	if bx < 0 {
		// Only the undo icon fits.
		bx = w - dp(gtx, 16)
		at(bx, LucideW("undo-2", SP.Foreground, 16))
	} else {
		at(bx, func(gtx C) D {
			d := th.Text(gtx, label, undo)
			if hovered {
				hline(gtx, label.Color, d.Size.Y-d.Baseline+dp(gtx, 2), 0, d.Size.X)
			}
			return d
		})
		if bx-gap-sw >= 0 {
			at(bx-gap-sw, func(gtx C) D { return th.Text(gtx, summary, text) })
		}
	}
	if id != "" && on {
		func() {
			defer op.Offset(image.Pt(bx, 0)).Push(gtx.Ops).Pop()
			th.hit(gtx, id, image.Pt(w-bx, h))
		}()
	}
}

// footerButton is a footer icon button (SidebarMenuButton size=icon):
// 32dp, radius 8, a 16dp icon in the sidebar's icon colour; the row hover
// fill and the foreground on hover or while its menu is open, the active
// fill while pressed. Its tooltip goes on top, 600ms in.
func (th *Theme) footerButton(gtx C, id, icon, tip string, open bool) {
	s := dp(gtx, 32)
	r := image.Rect(0, 0, s, s)
	hovered, pressed := false, false
	if id != "" {
		c, _ := th.events(gtx, id)
		hovered, pressed = c.Hovered(), c.Pressed()
	}
	col := SP.SidebarIconColor
	switch {
	case pressed:
		fill(gtx, over(SP.SidebarRowActive, SP.Sidebar), r, dp(gtx, 8))
		col = SP.SidebarForeground
	case hovered || open:
		fill(gtx, over(SP.SidebarRowHover, SP.Sidebar), r, dp(gtx, 8))
		col = SP.SidebarForeground
	}
	func() {
		defer op.Offset(image.Pt((s-dp(gtx, 16))/2, (s-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		LucideW(icon, col, 16)(gtx)
	}()
	if id != "" {
		th.hit(gtx, id, r.Max)
		if !open {
			th.tipAt(gtx, id, tip, r.Max, true, -1)
		}
	}
}
