package ui

import (
	"image"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// press queues a press and release of the key name.
func (l *live) press(name key.Name) {
	l.r.Queue(key.Event{Name: name, State: key.Press}, key.Event{Name: name, State: key.Release})
}

// threadMenu is a thread's actions menu as the header opens it, cut down:
// Rename, Mark unread, Copy with its submenu, a row that's off, Delete.
func threadMenu() *ContextMenu {
	return &ContextMenu{Items: []MenuItem{
		{ID: "rename", Label: "Rename thread", Icon: "pencil"},
		{ID: "mark-unread", Label: "Mark unread", Icon: "mail-open"},
		{ID: "copy", Label: "Copy", Icon: "copy", Sep: true, Children: []MenuItem{
			{ID: "copy-path", Label: "Path", Icon: "folder"}, {ID: "copy-thread-id", Label: "Thread ID", Icon: "hash"},
		}},
		{ID: "archive", Label: "Archive thread", Icon: "archive", Off: true, Sep: true},
		{ID: "delete", Label: "Delete", Icon: "trash", Danger: true},
	}}
}

// menuWidth is a thread menu's width as T3 lays it out: its widest row
// (padding 8, the 16dp icon, 8, the label, a submenu's 8 and chevron 14,
// padding 8) inside the padding and border, 128 at least.
func menuWidth(th *Theme, gtx C, items []MenuItem) int {
	w := 0
	for _, it := range items {
		row := 8 + 16 + 8 + th.textW(gtx, Txt{Size: 14, LineH: 20.0 / 14}, it.Label) + 8
		if len(it.Children) > 0 {
			row += 8 + 14
		}
		w = max(w, row)
	}
	return max(128, w+2*5)
}

// The thread's actions menu is T3's in-page one (§3.3, port note 4): the
// dropdown glass as it paints over the window (#ffffff, #0e0e0e dark)
// inside a 1dp border at the foreground's 10%, radius 8, padded 4, no
// shadow; 28dp rows, 1dp separators 8 in from the padding and 4 clear
// above and below; as wide as its widest row, and 128 at least.
func TestThreadMenuDrawn(t *testing.T) {
	defer SetDark(false)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	for _, d := range []bool{false, true} {
		SetDark(d)
		s := headerScene()
		s.Live = false
		s.Menu = threadMenu()
		s.Menu.At = image.Pt(300, 41)
		th := NewTheme()
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
		glass := rgb(0xffffff)
		if d {
			glass = rgb(0x0e0e0e)
		}
		border := over(alpha(P.ContrastForeground, 0x1a), glass)
		// 1 + 4, five rows and two separators, 4 + 1.
		height, width := 2*5+5*28+2*9, menuWidth(th, gtx, s.Menu.Items)
		x, y := 300, 41
		wantExact(t, img, image.Pt(x+60, y+3), glass, "the menu's glass")
		wantPixel(t, img, image.Pt(x, y+60), border, "its border")
		wantPixel(t, img, image.Pt(x+60, y+height+2), P.Background, "no shadow under it")
		wantPixel(t, img, image.Pt(x+width-1, y+height/2), border, "its right border, after its widest row")
		wantPixel(t, img, image.Pt(x+width, y+height/2), P.Background, "past it")
		// The first separator, after two rows: 1dp at 4 below them, from 8
		// past the padding.
		sep := y + 5 + 2*28 + 4
		wantPixel(t, img, image.Pt(x+5+8, sep), over(P.ContrastBorder, glass), "the separator's start")
		wantPixel(t, img, image.Pt(x+5+7, sep), glass, "before it")
		wantPixel(t, img, image.Pt(x+5+8, sep-1), glass, "above it")
		// The Copy row's chevron at its end, a dp into the padding.
		chev := x + width - 5 - 8 + 2
		if ink := darkest(img, image.Rect(chev-16, y+5+2*28+9+6, chev, y+5+3*28+9-6)); !d && ink > 0xe0 {
			t.Errorf("no chevron at the submenu's row's end (darkest %#x)", ink)
		}
	}
	// A menu of short rows is 128 wide.
	s := headerScene()
	s.Live = false
	s.Menu = &ContextMenu{At: image.Pt(300, 41), Items: []MenuItem{{ID: "pin", Label: "Pin", Icon: "pin"}}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	border := over(alpha(P.ContrastForeground, 0x1a), rgb(0xffffff))
	wantPixel(t, img, image.Pt(300+127, 41+19), border, "a short menu's right border, at 128")
	wantPixel(t, img, image.Pt(300+128, 41+19), P.Background, "past it")
}

// The menu's keyboard (contextMenuFallback.ts:284–343), on the menu with
// the focus: opened with the pointer it's on the menu itself; Down and Up
// move it through the rows that can be picked, wrapping; Home and End go
// to the ends; Enter picks, or on a submenu's row opens it on its first
// row, as Right does; Left and Escape in the submenu go back to its row;
// moving off that row closes it; Escape on the first menu, and Tab, close
// the lot, and Left there does nothing.
func TestThreadMenuKeys(t *testing.T) {
	s := headerScene()
	s.Menu = threadMenu()
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	l.frame(0)
	picks := func(keys ...key.Name) string {
		for _, k := range keys {
			l.press(k)
		}
		l.press(key.NameReturn)
		for _, a := range l.frame(0) {
			if strings.HasPrefix(a.ID, "ctx:") || a.ID == "menus.escape" {
				return a.ID
			}
		}
		return ""
	}
	for _, c := range []struct {
		keys []key.Name
		want string
	}{
		{[]key.Name{key.NameDownArrow}, "ctx:rename"},
		{[]key.Name{key.NameUpArrow}, "ctx:delete"}, // round to the end
		{[]key.Name{key.NameDownArrow}, "ctx:rename"},
		{[]key.Name{key.NameDownArrow, key.NameDownArrow, key.NameDownArrow}, "ctx:delete"}, // past the row that's off
		{[]key.Name{key.NameHome}, "ctx:rename"},
		{[]key.Name{key.NameEnd}, "ctx:delete"},
		{[]key.Name{key.NameHome, key.NameDownArrow, key.NameDownArrow}, ""}, // Enter on Copy opens it, picking nothing
		{nil, "ctx:copy-path"},
		{[]key.Name{key.NameDownArrow}, "ctx:copy-thread-id"},
		{[]key.Name{key.NameDownArrow}, "ctx:copy-path"},                 // round within the submenu
		{[]key.Name{key.NameLeftArrow, key.NameDownArrow}, "ctx:delete"}, // back on Copy, then on
		{[]key.Name{key.NameUpArrow, key.NameRightArrow}, "ctx:copy-path"},
		{[]key.Name{key.NameEscape, key.NameDownArrow}, "ctx:delete"},
	} {
		if got := picks(c.keys...); got != c.want {
			t.Errorf("%v then Enter: %q, want %q", c.keys, got, c.want)
		}
	}
	l.press(key.NameLeftArrow)
	if acts := l.frame(0); len(acts) != 0 {
		t.Errorf("Left on the first menu: %+v, want nothing", acts)
	}
	for _, k := range []key.Name{key.NameEscape, key.NameTab} {
		l.press(k)
		if acts := l.frame(0); len(acts) != 1 || acts[0].ID != "menus.escape" {
			t.Errorf("%s: %+v, want the menu closed", k, acts)
		}
	}
}

// The pointer coming onto a row focuses it (§5.2): onto Copy it opens the
// submenu beside it, 4dp right of the row, and leaves the focus on Copy,
// so one Escape closes the lot, as measured; onto a row without one it
// closes it. The keyboard can move the focus off the row the pointer
// rests on.
func TestThreadMenuHover(t *testing.T) {
	s := headerScene()
	s.Menu = threadMenu()
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	// The menu opens at the window's corner, kept 4dp in.
	w := menuWidth(l.th, gtx, s.Menu.Items)
	copyY := 4 + 5 + 2*28 + 9
	sub := image.Pt(4+w-5+4, copyY+14) // the submenu's left border
	border := over(alpha(P.ContrastForeground, 0x1a), rgb(0xffffff))
	l.move(40, float32(copyY+14))
	l.frame(0)
	l.frame(0)
	wantPixel(t, l.pixels(t), sub, border, "the submenu, 4dp right of Copy")
	l.press(key.NameEscape)
	if acts := l.frame(0); len(acts) != 1 || acts[0].ID != "menus.escape" {
		t.Errorf("Escape with the submenu open by hover: %+v, want the menu closed", acts)
	}
	l.move(40, 4+5+14) // onto Rename
	l.frame(0)
	l.frame(0)
	wantPixel(t, l.pixels(t), sub, P.Background, "the submenu, closed")
	l.press(key.NameDownArrow)
	l.frame(0)
	l.press(key.NameReturn)
	var got []string
	for _, a := range l.frame(0) {
		got = append(got, a.ID)
	}
	if strings.Join(got, " ") != "ctx:mark-unread" {
		t.Errorf("Down then Enter with the pointer on Rename: %v, want Mark unread", got)
	}
}

// A press outside the open menu closes it and goes on to what's under it
// (contextMenuFallback.ts's pointerdown watch): a second click on the
// title closes the menu, then opens it again.
func TestThreadMenuOutside(t *testing.T) {
	s := headerScene()
	s.Menu = threadMenu()
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	lay := l.th.layoutCrumbs(gtx, s.Header, s.Title, 1000, false)
	x := float32(256 + 20 + lay.project + lay.gap + lay.sep + lay.gap + 10)
	l.click(x, 26)
	acts := l.frame(10 * time.Millisecond)
	var ids []string
	for _, a := range acts {
		ids = append(ids, a.ID)
	}
	if strings.Join(ids, " ") != "menus.close header.title" {
		t.Fatalf("a click on the title with the menu open: %v, want it closed, then the title's click", ids)
	}
}

// Escape reaches what has the focus: with nothing open the rename field's
// Escape cancels it, rather than the shell taking the key; with the
// details popover up (a window too narrow for the card) and the focus on
// none of those, the shell closes it, from the first frame without the
// field.
func TestEscapeReachesTheFocus(t *testing.T) {
	s := headerScene()
	s.Header.Renaming = true
	l := newLive(image.Pt(1000, 800), nil)
	l.th.SetEditorText("header.title", s.Title)
	l.th.Focus("header.title")
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	l.frame(0)
	l.press(key.NameEscape)
	if acts := l.frame(0); len(acts) != 1 || acts[0] != (Action{ID: "header.title", Kind: "key", Text: "⎋"}) {
		t.Fatalf("Escape in the rename field: %+v", acts)
	}
	s.Header.Renaming, s.Details = false, true
	l.frame(0)
	l.press(key.NameEscape)
	if acts := l.frame(0); len(acts) != 1 || acts[0].ID != "menus.escape" {
		t.Fatalf("Escape with the popover up: %+v", acts)
	}
}

// A press outside the rename field blurs it, as T3's input: on a button
// (here the terminal toggle) that cancels the rename; anywhere else (the
// empty canvas) it commits.
func TestRenameBlur(t *testing.T) {
	s := headerScene()
	s.Header.Project = "atlas"
	s.Header.Renaming = true
	l := newLive(image.Pt(1440, 900), nil)
	l.th.SetEditorText("header.title", "Renamed")
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	blur := func(x, y float32) string {
		l.click(x, y)
		l.frame(10 * time.Millisecond)
		for _, a := range l.frame(10 * time.Millisecond) {
			if a.ID == "header.title" {
				return a.Kind
			}
		}
		return ""
	}
	if got := blur(1367+14, 12+14); got != "cancel" {
		t.Errorf("a press on the terminal toggle: %q, want the rename cancelled", got)
	}
	if got := blur(700, 600); got != "blur" {
		t.Errorf("a press on the canvas: %q, want the rename committed", got)
	}
}

// The rename field's text sits on the title's baseline: the caps centred
// on the row, at 26 + capHeight/2 (port note 3), not a dp above where the
// editor's own line box puts it.
func TestRenameBaseline(t *testing.T) {
	s := headerScene()
	s.Title = "HHHH"
	th := NewTheme()
	th.SetEditorText("header.title", s.Title)
	bottom := func(renaming bool) int {
		s.Header.Renaming = renaming
		img := drawn(t, image.Pt(800, 52), 1, func(gtx C) D {
			fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
			return th.chatHeader(gtx, s, image.Point{}, false)
		})
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
		lay := th.layoutCrumbs(gtx, s.Header, s.Title, 800, false)
		x := lay.project + lay.gap + lay.sep + lay.gap
		last := -1
		for y := 16; y < 36; y++ {
			if darkest(img, image.Rect(x+1, y, x+30, y+1)) < 0x80 {
				last = y
			}
		}
		return last
	}
	if title, field := bottom(false), bottom(true); title < 0 || field != title {
		t.Errorf("the H's foot: title at %d, rename field at %d", title, field)
	}
}

// Escape hands the focus back to a menu's trigger, with its ring: here the
// title, which the thread's menu opened from.
func TestFocusBackToTrigger(t *testing.T) {
	s := headerScene()
	l := newLive(image.Pt(900, 52), nil)
	l.draw = func(gtx C) {
		g := gtx
		g.Constraints = layout.Exact(image.Pt(800, 52))
		l.th.chatHeader(g, s, image.Point{}, false)
	}
	l.frame(0)
	l.th.FocusTarget("header.title")
	l.frame(0)
	l.frame(0)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	lay := l.th.layoutCrumbs(gtx, s.Header, s.Title, 800, false)
	wantPixel(t, l.pixels(t), image.Pt(lay.project+lay.gap+lay.sep+lay.gap, 26), P.Ring, "the title's focus ring")
}

// A card menu takes the keyboard (02-primitives.md port note 9): opened
// from the keyboard its first row is lit; the arrows move the light,
// wrapping, a letter jumps to the row it starts, Enter picks the lit row,
// and Escape closes the menu.
func TestCardMenuKeys(t *testing.T) {
	s := cardScene()
	s.Card.Menu = &CardMenu{For: "git", Rows: []MenuRow{
		{Label: "Commit", Icon: "git-commit-horizontal"}, {Label: "Push", Icon: "cloud-upload"}, {Label: "Create PR", Icon: "provider:github"},
		{Kind: MenuNote, Tone: "warning", Label: "Behind upstream. Pull/rebase first."}}}
	l := newLive(image.Pt(1440, 900), nil)
	l.th.in.keyFocus = true
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	l.frame(0)
	picks := func(keys ...key.Name) string {
		for _, k := range keys {
			l.press(k)
		}
		l.press(key.NameReturn)
		for _, a := range l.frame(time.Second) {
			return a.ID
		}
		return ""
	}
	for _, c := range []struct {
		keys []key.Name
		want string
	}{
		{nil, "card.git:0"},
		{[]key.Name{key.NameDownArrow}, "card.git:1"},
		{[]key.Name{key.NameDownArrow, key.NameDownArrow}, "card.git:0"}, // past the note, round to the top
		{[]key.Name{key.NameUpArrow}, "card.git:2"},
		{[]key.Name{"P"}, "card.git:1"},
		{[]key.Name{"C"}, "card.git:2"}, // the next row that starts with C
		{[]key.Name{key.NameHome}, "card.git:0"},
	} {
		if got := picks(c.keys...); got != c.want {
			t.Errorf("%v then Enter: %q, want %q", c.keys, got, c.want)
		}
	}
	l.press(key.NameEscape)
	if acts := l.frame(0); len(acts) != 1 || acts[0].ID != "menus.escape" {
		t.Errorf("Escape: %+v", acts)
	}
}

// The draft's workspace menu draws T3's radio rows (MenuRadioItem): the
// 12dp icon in the row's colour a dp into its padding at x + 6, the label
// at x + 22, no check, the checked row filled with foreground at 8%; the
// previous worktree's row 44 tall with its branch under its label. The
// menu hangs under the run row at 109, as T3's (§3.7).
func TestRunMenuRows(t *testing.T) {
	s := cardScene()
	s.Card.Run.Locked = false
	s.Card.Menu = &CardMenu{For: "run", Rows: []MenuRow{
		{Kind: MenuGroupLabel, Label: "Workspace"},
		{Kind: MenuRadioRow, Label: "Current checkout", Icon: "folder", Checked: true},
		{Kind: MenuRadioRow, Label: "New worktree", Icon: "folder-git-2"},
		{Kind: MenuRadioRow, Label: "Previous worktree", Icon: "history", Detail: "fix/lexer"},
	}}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	x, row := 1157+1+4, 109+1+4+28 // the checked row's top left
	checked := over(alpha(P.Foreground, 0x14), P.Popover)
	wantPixel(t, img, image.Pt(x+2, row+2), checked, "the checked row's fill")
	wantPixel(t, img, image.Pt(x+254-8, row+14), checked, "no check at its end")
	if ink := darkest(img, image.Rect(x+6, row+8, x+18, row+20)); ink > 0x60 {
		t.Errorf("no 12dp icon at x + 6 (darkest %#x)", ink)
	}
	if ink := darkest(img, image.Rect(x+18, row+4, x+22, row+24)); ink < 0xb0 {
		t.Errorf("ink between the icon and the label (darkest %#x)", ink)
	}
	if ink := darkest(img, image.Rect(x+22, row+8, x+30, row+20)); ink > 0x60 {
		t.Errorf("no label at x + 22 (darkest %#x)", ink)
	}
	// Border, padding, the label, two rows, the 44dp previous worktree with
	// its branch in its second line, padding, border: 138 tall, its bottom
	// border on the card's white below.
	border := over(alpha(rgb(0x14141d), 0x1a), P.Popover)
	wantPixel(t, img, image.Pt(1157+130, 109+138-1), border, "the menu's bottom border")
	wantPixel(t, img, image.Pt(1157+130, 109+138-3), P.Popover, "its padding above it")
	if ink := darkest(img, image.Rect(x+22, row+2*28+20+2, x+60, row+2*28+20+14)); ink > 0xa0 {
		t.Errorf("no branch on the previous worktree's second line (darkest %#x)", ink)
	}
}

// The card's own menu opens at the right-click as a Base UI menu, side
// bottom 4dp off, align start; with no room on the right it lines its
// right edge up with the click instead (§3.9): at (1300, 69), from x 1068.
func TestMenuAtPointerFlips(t *testing.T) {
	s := cardScene()
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	l.r.Queue(pointer.Event{Kind: pointer.Press, Position: f32.Pt(1300, 69), Buttons: pointer.ButtonSecondary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: f32.Pt(1300, 69), Source: pointer.Mouse})
	l.frame(0)
	s.Card.Menu = &CardMenu{For: "items", AtPointer: true, Rows: []MenuRow{{Kind: MenuGroupLabel, Label: "Show in thread details"},
		{Kind: MenuCheckRow, Label: "Changes", Checked: true}}}
	l.frame(0)
	img := l.pixels(t)
	y := 69 + 4 + 20
	wantPixel(t, img, image.Pt(1300-6, y), P.Popover, "the menu's right end, at the click")
	wantPixel(t, img, image.Pt(1300-232+6, y), P.Popover, "its left end, 232 back")
	if c := img.RGBAAt(1300+3, y); nearPixel(c, P.Popover) {
		t.Errorf("past the click is the menu's #%02x%02x%02x", c.R, c.G, c.B)
	}
}

// T3's tooltip is 16.5 of text padded 4 by 8 inside its 1dp border:
// 26.5dp tall (§3.2), and its text's width and 18 wide.
func TestTooltipBox(t *testing.T) {
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}, Constraints: layout.Constraints{Max: image.Pt(640, 400)}}
	label := "Toggle thread details (Ctrl+Shift+B)"
	d := th.tooltip(gtx, label)
	textW := th.textW(gtx, Txt{Size: 12, LineH: 16.5 / 12}, label)
	if d.Size.Y != 53 || d.Size.X != textW+36 {
		t.Errorf("tooltip %v at 2x, want %dx53", d.Size, textW+36)
	}
}

// Hovering the quick action that only explains itself fades the whole
// control to 64%, its own hover fill with it (§2.6): over the pair's
// black 3.5% (#f6f6f6) its fill paints #f1f1f1, as measured on T3.
func TestHintHover(t *testing.T) {
	s := cardScene()
	s.Card.Git = &GitActions{Label: "Push", Icon: "cloud-upload", Hint: "No local commits to push."}
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	git := image.Pt(1157, 64+1+114+1+8+32) // the Git row's top left
	l.move(float32(git.X+150), float32(git.Y+16))
	l.frame(0)
	l.frame(0)
	img := l.pixels(t)
	wantPixel(t, img, image.Pt(git.X+200, git.Y+5), rgb(0xf1f1f1), "the hovered hint's fill")
	wantPixel(t, img, image.Pt(git.X+229+16, git.Y+5), rgb(0xf6f6f6), "the pair's fill on the chevron")
	wantPixel(t, img, image.Pt(git.X+229, git.Y+16), rgb(0xeaeaec), "the hairline")
}

// The Git row at work leads with the control's own 16dp spinner in its
// text colour, a dp into each side of its padding (x + 9), and the status
// 12 after it (x + 35); done, a 14dp check at x + 11 and the title at
// x + 37.
func TestGitProgressDrawn(t *testing.T) {
	s := cardScene()
	s.Card.Git = &GitActions{Label: "Commit", Running: &GitProgress{Status: "Hhh", Elapsed: "3s"}}
	git := image.Pt(1157, 64+1+114+1+8+32)
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return NewTheme().Desktop(gtx, s) })
	if ink := darkest(img, image.Rect(git.X+9, git.Y+6, git.X+25, git.Y+24)); ink > 0xa0 {
		t.Errorf("no spinner from x + 9 (darkest %#x)", ink)
	}
	if ink := darkest(img, image.Rect(git.X+26, git.Y+4, git.X+35, git.Y+28)); ink < 0xd0 {
		t.Errorf("ink between the spinner and the status (darkest %#x)", ink)
	}
	if ink := darkest(img, image.Rect(git.X+35, git.Y+8, git.X+42, git.Y+24)); ink > 0x80 {
		t.Errorf("no status at x + 35 (darkest %#x)", ink)
	}
}
