package ui

import (
	"image"
	"runtime"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The lane rule reproduces T3's measured lanes at 1440x900 with the
// sidebar open (docs/ui/03-shell.md §3.5, §5.4): the canvas is 1184 wide,
// the details card 280 wide at 892.
func TestLane(t *testing.T) {
	for _, c := range []struct {
		name               string
		maxChat, cardLeft  int
		wantLeft, wantWide int
	}{
		{"comfortable, card inline", 736, 892, 380 - 256, 736},
		{"comfortable, no card", 736, -1, 480 - 256, 736},
		{"wide, card inline", 1152, 892, 304 - 256, 812},
		{"wide, no card", 1152, -1, 304 - 256, 1088},
	} {
		left, w := lane(1184, 48, c.maxChat, 32, c.cardLeft)
		if left != c.wantLeft || w != c.wantWide {
			t.Errorf("%s: lane at %d, %d wide; want %d, %d", c.name, left, w, c.wantLeft, c.wantWide)
		}
	}
}

// The sidebar: its 255dp surface, then the 1dp border at x 255, then the
// main area; the stage art at the top (its sky is #07152f in both themes).
func TestSidebarFrame(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, &Scene{Live: true}) })
		wantPixel(t, img, image.Pt(254, 500), SP.Sidebar, "sidebar surface")
		wantPixel(t, img, image.Pt(255, 500), over(SP.ContrastBorder, P.Background), "sidebar border")
		wantPixel(t, img, image.Pt(300, 500), P.Background, "main area")
		wantPixel(t, img, image.Pt(10, 6), rgb(0x07152f), "stage art sky")
	}
}

// The title bar's panel toggles sit where T3's do at 1440 wide (x 1335,
// 1367, 1399; 28dp at y 12), and a pressed one is filled with --accent.
func TestTitlebarControls(t *testing.T) {
	th := NewTheme()
	s := &Scene{Live: true, Title: "A thread", Card: &ThreadDetails{}, Term: nil}
	img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		th.titleButtonW("terminal.toggle", "", titleButton{icon: "panel-bottom", pressed: true})(gtx)
		return D{}
	})
	wantPixel(t, img, image.Pt(2, 14), over(P.Accent, P.Background), "pressed toggle")
	img = drawn(t, image.Pt(1440, 900), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rect(0, 0, 1440, 900), 0)
		th.titlebarControls(gtx, s, image.Pt(1440, 900), false)
		return D{}
	})
	for _, x := range []int{1335, 1367, 1399} {
		wantPixel(t, img, image.Pt(x+2, 14), P.Background, "unpressed toggle")
	}
}

// Without a project the terminal toggle (at 1367, 12) is off: a click does
// nothing, but resting on it still brings up its tooltip under it, centred
// on it and moved to keep 5dp inside the window, its right edge at 1435 as
// T3's (§3.2). In a thread in a project it toggles; a draft, which has no
// terminal until its first message, keeps it off.
func TestTerminalToggleNeedsProject(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := &Scene{Live: true, Title: "A thread", Header: &ChatHeader{Server: true}, Card: &ThreadDetails{}}
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	toggled := func() bool {
		l.click(1367+14, 12+14)
		for _, a := range l.frame(10 * time.Millisecond) {
			if a.ID == "terminal.toggle" {
				return true
			}
		}
		return false
	}
	if toggled() {
		t.Fatal("the toggle acted without a project")
	}
	l.move(1367+14, 12+14)
	l.frame(0)
	l.frame(800 * time.Millisecond)
	img := l.pixels(t)
	y := 12 + 28 + 4 + 13
	wantPixel(t, img, image.Pt(1435-3, y), P.Popover, "the tooltip's right end, 5dp in")
	wantPixel(t, img, image.Pt(1435+4, y), P.Background, "past it, the window's edge")

	s.Header.Project = "atlas"
	l.frame(time.Second)
	if !toggled() {
		t.Fatal("in a thread in a project the toggle should toggle")
	}
	s.Header.Server = false
	l.frame(time.Second)
	if toggled() {
		t.Fatal("a draft's toggle acted")
	}
}

// The panel toggles' tooltips are centred on them and kept 5dp inside the
// window (§3.2): the details toggle's right edge lands at 1435, as T3's.
func TestTitlebarTooltipsInside(t *testing.T) {
	s := &Scene{Live: true, Title: "A thread", Header: &ChatHeader{Server: true, Project: "atlas"}, Card: &ThreadDetails{}}
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	l.move(1335+14, 12+14)
	l.frame(0)
	l.frame(800 * time.Millisecond)
	img := l.pixels(t)
	y := 12 + 28 + 4 + 13
	wantPixel(t, img, image.Pt(1435-3, y), P.Popover, "the details tooltip's right end")
	wantPixel(t, img, image.Pt(1435+4, y), P.Background, "the window past it")
}

// The toggles' shortcuts are the server's (§3.4, §5.3): with its rules
// moving the terminal drawer to mod+Alt+J and the right panel to
// mod+Alt+Shift+B, as on the server T3 was measured against, the terminal
// toggle's tooltip says so, as wide as that label (and still held 5dp in),
// and mod+J does nothing while mod+Alt+J toggles the drawer. A later rule
// that takes a command's keys leaves it without a shortcut.
func TestShortcutsFromKeybindings(t *testing.T) {
	keys := MergeKeys([]Keybinding{
		{Command: "terminal.toggle", Key: "j", Mod: true, Alt: true},
		{Command: "rightPanel.toggle", Key: "b", Mod: true, Alt: true, Shift: true},
	})
	term, right := "Ctrl+Alt+J", "Ctrl+Alt+Shift+B"
	if runtime.GOOS == "darwin" {
		term, right = "⌥⌘J", "⌥⇧⌘B"
	}
	if got := KeyLabel(keys, "terminal.toggle"); got != term {
		t.Errorf("the terminal drawer's shortcut %q, want %q", got, term)
	}
	if got := KeyLabel(keys, "rightPanel.toggle"); got != right {
		t.Errorf("the right panel's shortcut %q, want %q", got, right)
	}
	if got := KeyLabel(MergeKeys([]Keybinding{{Command: "chat.find", Key: "j", Mod: true}}), "terminal.toggle"); got != "" {
		t.Errorf("mod+J taken by a later rule, the drawer's shortcut is still %q", got)
	}

	s := &Scene{Live: true, Title: "A thread", Header: &ChatHeader{Server: true, Project: "atlas"}, Card: &ThreadDetails{}, Keys: keys}
	l := newLive(image.Pt(1440, 900), nil)
	l.draw = func(gtx C) { l.th.Desktop(gtx, s) }
	l.frame(0)
	l.move(1367+14, 12+14)
	l.frame(0)
	l.frame(800 * time.Millisecond)
	img := l.pixels(t)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 100)}}
	w := l.th.tooltip(gtx, "Toggle terminal drawer ("+term+")").Size.X
	y := 12 + 28 + 4 + 13
	wantPixel(t, img, image.Pt(1435-w+3, y), P.Popover, "the tooltip's left end, as wide as its label")
	wantPixel(t, img, image.Pt(1435-w-4, y), P.Background, "left of it")
	chord := func(mods key.Modifiers) bool {
		l.r.Queue(key.Event{Name: "J", Modifiers: mods, State: key.Press}, key.Event{Name: "J", Modifiers: mods, State: key.Release})
		for _, a := range l.frame(10 * time.Millisecond) {
			if a.ID == "terminal.toggle" {
				return true
			}
		}
		return false
	}
	if chord(key.ModShortcut) {
		t.Error("mod+J toggled the drawer after the server moved it")
	}
	if !chord(key.ModShortcut | key.ModAlt) {
		t.Error("mod+Alt+J didn't toggle the drawer")
	}
}
