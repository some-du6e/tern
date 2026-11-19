package ui

import (
	"image"
	"math"
	"runtime"
	"testing"
	"time"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

// The drawer's height (docs/ui/09-terminal.md §3.2): at 900 tall the most
// is 675; it's never under 180 unless the room left is; a stored height
// that isn't a number is 280.
func TestTermHeight(t *testing.T) {
	if got := termMaxHeight(900, 900, 52, 150); got != 675 {
		t.Errorf("max at 900 tall = %v, want 675", got)
	}
	if got := termMaxHeight(900, 900, 52, 300); got != 548 {
		t.Errorf("max with a 300 composer = %v, want 548 (900 - 52 - 300)", got)
	}
	most := termMaxHeight(300, 300, 52, 150)
	for _, c := range []struct{ stored, most, want float32 }{
		{280, 675, 280}, {100, 675, 180}, {2000, 675, 675}, {402.6, 675, 403},
		{280, most, 98}, {float32(math.NaN()), 675, 280},
	} {
		if got := termHeight(c.stored, c.most); got != c.want {
			t.Errorf("height %v under %v = %v, want %v", c.stored, c.most, got, c.want)
		}
	}
}

// The grid and its origin: T3's 1184x279 drawer with Liberation Mono's
// 7.80127x18 cell holds 150x15, bottom-anchored at 5 once there's
// history; a 900x420 mount holds 114x22 from 4.
func TestTermGrid(t *testing.T) {
	if c, r := termGrid(1184, 279, 7.80127, 18); c != 150 || r != 15 {
		t.Errorf("1184x279 = %dx%d, want 150x15", c, r)
	}
	if c, r := termGrid(900, 420, 7.80127, 18); c != 114 || r != 22 {
		t.Errorf("900x420 = %dx%d, want 114x22", c, r)
	}
	if got := termOriginY(279, 15, 18, true); got != 5 {
		t.Errorf("origin with history = %v, want 5", got)
	}
	if got := termOriginY(279, 15, 18, false); got != 4 {
		t.Errorf("origin without history = %v, want 4", got)
	}
}

// The cell is the font's "M" and at least 1.35 lines: 18 for 13px, its
// baseline 12 in Liberation Mono.
func TestTermCellMetrics(t *testing.T) {
	th := NewTheme()
	gtx := layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
	c := th.termMetrics(gtx)
	if c.h != 18 {
		t.Errorf("cell height %d, want 18", c.h)
	}
	if runtime.GOOS == "linux" && (math.Abs(float64(c.w/2)-7.80127) > 0.02 || c.base != 12) {
		t.Errorf("Liberation Mono cell %v wide with its baseline at %d, want 7.80127 and 12", c.w/2, c.base)
	}
}

// The scrollbar's thumb: T3's two live readings (29 rows, 15 shown, 14 or
// 13 scrolled) and its 18 minimum; none without history.
func TestTermThumb(t *testing.T) {
	near := func(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }
	if h, top, most, ok := termThumb(29, 15, 14, 271); !ok || !near(h, 140.17) || !near(top, 130.83) || most != 14 {
		t.Errorf("29/15/14: %v at %v (most %d), want 140.17 at 130.83", h, top, most)
	}
	if h, top, _, _ := termThumb(28, 15, 13, 271); !near(h, 145.18) || !near(top, 125.82) {
		t.Errorf("28/15/13: %v at %v, want 145.18 at 125.82", h, top)
	}
	if h, _, _, _ := termThumb(10000, 15, 0, 271); h != 18 {
		t.Errorf("a long history's thumb is %v, want 18", h)
	}
	if _, _, _, ok := termThumb(15, 15, 0, 271); ok {
		t.Error("no history should hide the bar")
	}
}

// The wheel turns pixels into rows of the cell's height, carrying the
// rest; a drag past an edge scrolls 1 row a tick there, 15 at 50 past.
func TestTermWheelAndAutoscroll(t *testing.T) {
	carry, total := float32(0), 0
	for range 5 {
		var n int
		n, carry = termWheelRows(8, 18, carry)
		total += n
	}
	if total != 2 || math.Abs(float64(carry-4.0/18)) > 1e-4 {
		t.Errorf("5 x 8px over 18px rows = %d rows and %v, want 2 and 4/18", total, carry)
	}
	if n, _ := termWheelRows(-40, 18, 0); n != -2 {
		t.Errorf("40px up = %d rows, want -2", n)
	}
	for _, c := range []struct {
		y    float32
		want int
	}{{100, 0}, {201, 1}, {225, 8}, {250, 15}, {400, 15}, {-50, -15}} {
		if got := termAutoscroll(c.y, 0, 200); got != c.want {
			t.Errorf("autoscroll at %v = %d, want %d", c.y, got, c.want)
		}
	}
}

// Faint text blends 155:100 into the background (§3.7).
func TestTermFaint(t *testing.T) {
	for _, c := range []struct{ fg, bg, want uint32 }{
		{0x18181b, 0xfcfcfc, 0x717173}, {0xf5f5f5, 0x000000, 0x949494},
		{0xcd3131, 0xfcfcfc, 0xdf8080}, {0xcd3131, 0x000000, 0x7c1d1d},
	} {
		if got := termFaint(rgb(c.fg), rgb(c.bg)); got != rgb(c.want) {
			t.Errorf("faint #%06x on #%06x = %v, want #%06x", c.fg, c.bg, got, c.want)
		}
	}
}

// T3's URLs: trailing punctuation and unmatched brackets come off; a colon
// at the end stays.
func TestTermURLs(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{"see https://example.com/docs.", "https://example.com/docs"},
		{"(https://example.com/docs), ./src/main.ts:12.", "https://example.com/docs"},
		{"GET https://example.test/items/foo:", "https://example.test/items/foo:"},
		{"open https://example.com/a_(b) now", "https://example.com/a_(b)"},
		{`"https://example.com/q?id=42"`, "https://example.com/q?id=42"},
		{"no links here", ""},
	} {
		got := ""
		if ms := termURLs(c.line); len(ms) > 0 {
			got = ms[0].text
			if c.line[ms[0].start:ms[0].end] != got {
				t.Errorf("%q: offsets %d-%d don't hold %q", c.line, ms[0].start, ms[0].end, got)
			}
		}
		if got != c.want {
			t.Errorf("%q: url %q, want %q", c.line, got, c.want)
		}
	}
}

// The toolbar's labels carry the default keys, and a full group says so.
func TestTermToolLabels(t *testing.T) {
	var got []string
	for _, tool := range termTools(false) {
		got = append(got, tool.label)
	}
	want := []string{"Split Terminal Horizontally (" + Shortcut("D") + ")", "Split Terminal Vertically (" + ShortcutShift("D") + ")",
		"New Terminal (" + Shortcut("N") + ")", "Close Terminal (" + Shortcut("W") + ")"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("label %d = %q, want %q", i, got[i], want[i])
		}
	}
	if runtime.GOOS != "darwin" && want[1] != "Split Terminal Vertically (Ctrl+Shift+D)" {
		t.Errorf("label %q, want Ctrl+Shift+D", want[1])
	}
	full := termTools(true)
	if full[0].label != "Split Terminal Horizontally (max 4 per group)" || !full[0].off || full[2].off {
		t.Errorf("a full group's split is %q (off %v)", full[0].label, full[0].off)
	}
}

// The right-click menu: Copy wants a selection; the rest are always there.
func TestTermMenuItems(t *testing.T) {
	for _, sel := range []bool{false, true} {
		items := termMenuItems(sel)
		var labels []string
		for _, it := range items {
			labels = append(labels, it.label)
		}
		if len(labels) != 4 || labels[0] != "Copy" || labels[1] != "Paste" || labels[2] != "Select all" || labels[3] != "Jump to latest" {
			t.Fatalf("menu %v", labels)
		}
		if items[0].off == sel || items[1].off || items[2].off || items[3].off {
			t.Errorf("with a selection %v, Copy off is %v", sel, items[0].off)
		}
	}
}

// termTestView is a drawer holding the emulators given, one group a
// screen, the last screen active.
func termTestView(groups ...[]TermScreen) *TermView {
	v := &TermView{Thread: "env/thread", Height: 280}
	n := 0
	for gi, g := range groups {
		tg := TermGroup{Vertical: true}
		for _, scr := range g {
			n++
			id := "term-" + string(rune('0'+n))
			tg.Tabs = append(tg.Tabs, TermTab{ID: id, Label: "Terminal " + id[5:]})
			if gi == len(groups)-1 {
				v.Panes = append(v.Panes, TermPane{ID: id, Screen: scr})
			}
			v.Active = id
		}
		v.Groups = append(v.Groups, tg)
	}
	v.Shown = len(groups) - 1
	return v
}

// The drawer at 1440x900: its 280 bottom of the chat column with the 1dp
// top border at 80% at y 620, the terminal below, and the lone terminal's
// toolbar 8 in from the top right: 89 wide, its border, its separators.
func TestTermDrawerFrame(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		s := &Scene{Term: termTestView([]TermScreen{vt.NewSafeEmulator(120, 30)})}
		img := drawn(t, image.Pt(1440, 900), 1, func(gtx C) D { return th.Desktop(gtx, s) })
		line := over(faded(P.ContrastBorder, 0.8), P.Background)
		wantPixel(t, img, image.Pt(700, 620), line, "top border")
		wantPixel(t, img, image.Pt(700, 700), P.TerminalBackground, "terminal")
		x, y := 1440-8-89, 621+8
		wantPixel(t, img, image.Pt(x, y+11), line, "toolbar's left border")
		wantPixel(t, img, image.Pt(x+88, y+11), line, "toolbar's right border")
		wantPixel(t, img, image.Pt(x+22, y+11), line, "first separator")
		wantPixel(t, img, image.Pt(x+2, y+2), P.Background, "a button")
	}
}

// Two groups, a single terminal and a stacked pair with the second active:
// the 144 list at the right after a 6 gap, its rows 24 tall from y 50,
// the active group's header on accent at 50% and the active row on accent;
// the panes stacked with the active one's divider in --border.
func TestTermDrawerList(t *testing.T) {
	th := NewTheme()
	s := &Scene{Term: termTestView([]TermScreen{nil}, []TermScreen{nil, nil})}
	img := drawn(t, image.Pt(1184, 280), 1, func(gtx C) D {
		th.terminalDrawer(gtx, s)
		return D{Size: gtx.Constraints.Max}
	})
	listBg := over(faded(P.Muted, 0.1), P.TerminalBackground)
	line := over(faded(P.ContrastBorder, 0.7), P.TerminalBackground)
	lx := 1184 - 144
	wantPixel(t, img, image.Pt(lx, 150), line, "list border")
	wantPixel(t, img, image.Pt(lx-3, 150), P.TerminalBackground, "gap")
	wantPixel(t, img, image.Pt(lx+60, 23), line, "list toolbar's bottom border")
	wantPixel(t, img, image.Pt(lx+100, 87), over(faded(P.Accent, 0.5), listBg), "active group's header")
	wantPixel(t, img, image.Pt(lx+100, 39), listBg, "the other group's header")
	wantPixel(t, img, image.Pt(lx+100, 110), listBg, "inactive row")
	wantPixel(t, img, image.Pt(lx+100, 136), over(P.Accent, listBg), "active row")
	wantPixel(t, img, image.Pt(500, 140), over(P.ContrastBorder, P.TerminalBackground), "active pane's divider")
	wantPixel(t, img, image.Pt(500, 100), P.TerminalBackground, "first pane")
}

// With no terminals, the note and the outline New Terminal button sit in
// the middle.
func TestTermDrawerEmpty(t *testing.T) {
	th := NewTheme()
	s := &Scene{Term: &TermView{Height: 280}}
	img := drawn(t, image.Pt(1184, 280), 1, func(gtx C) D {
		th.terminalDrawer(gtx, s)
		return D{Size: gtx.Constraints.Max}
	})
	// The text is 20 tall, then 12, then the 24dp button, centred in
	// 279 less 48 of padding: the button's top is at 1+24+87+32 = 144,
	// and its label starts a few dp lower.
	wantPixel(t, img, image.Pt(592, 146), over(P.Popover, P.Background), "New Terminal button")
	wantPixel(t, img, image.Pt(592, 143), P.Background, "above the button")
	wantPixel(t, img, image.Pt(592, 60), P.Background, "above the note")
}

// termPaneShot draws a pane of scr into w by h dp at scale 1, and the
// pane's own state for what the test sets.
func termPaneShot(t *testing.T, th *Theme, scr TermScreen, w, h int, set func(p *termPane)) *image.RGBA {
	t.Helper()
	v := termTestView([]TermScreen{scr})
	if set != nil {
		set(th.pane(v, v.Panes[0].ID))
	}
	return drawn(t, image.Pt(w, h), 1, func(gtx C) D {
		th.termSurface(gtx, v, &v.Panes[0], true, image.Point{})
		return D{Size: gtx.Constraints.Max}
	})
}

// Cells paint T3's ANSI colours by scheme on the terminal background,
// inverse swaps it with the foreground, the selection overlays the cells
// and never the padding, and without the keyboard the cursor is a 1dp
// outline in --terminal-cursor.
func TestTermCells(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		cells := th.termMetrics(layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}})
		cx := func(c int) int { return int(math.Round(float64(termPad + float32(c)*cells.w))) }
		mid := func(c int) int { return (cx(c) + cx(c+1)) / 2 }
		row := func(r int) int { return termPad + r*cells.h + cells.h/2 }
		scr := vt.NewSafeEmulator(20, 3)
		scr.Write([]byte("\x1b[41m  \x1b[42m  \x1b[7m  \x1b[0m\r\n\x1b[38;2;18;52;86m\x1b[48;2;18;52;86m  \x1b[0m"))
		img := termPaneShot(t, th, scr, 200, 70, func(p *termPane) {
			p.sel = termSel{on: true, from: termPos{2, 0}, to: termPos{2, 4}, mode: 1}
		})
		pal := termPalette()
		wantPixel(t, img, image.Pt(mid(0), row(0)), pal[1], "red background")
		wantPixel(t, img, image.Pt(mid(2), row(0)), pal[2], "green background")
		wantPixel(t, img, image.Pt(mid(4), row(0)), P.TerminalForeground, "inverse")
		wantPixel(t, img, image.Pt(mid(0), row(1)), rgb(0x123456), "truecolor")
		wantPixel(t, img, image.Pt(mid(1), row(2)), over(P.TerminalSelectionBackground, P.TerminalBackground), "selection")
		wantPixel(t, img, image.Pt(1, row(2)), P.TerminalBackground, "padding beside the selection")
		// The cursor sits after the truecolor cells, at (2, 1).
		wantPixel(t, img, image.Pt(cx(2), row(1)), P.TerminalCursor, "hollow cursor's left edge")
		wantPixel(t, img, image.Pt(mid(2), row(1)), P.TerminalBackground, "inside the hollow cursor")
	}
	if termPalettes[0][2] != rgb(0x107c10) || termPalettes[1][2] != rgb(0x0dbc79) || termPalettes[0][15] != rgb(0xa5a5a5) {
		t.Error("green and bright white differ by scheme: #107c10 and #0dbc79, #a5a5a5 in light")
	}
}

// With the keyboard the cursor is the full block in --terminal-cursor.
func TestTermBlockCursor(t *testing.T) {
	th := NewTheme()
	scr := vt.NewSafeEmulator(20, 3)
	v := termTestView([]TermScreen{scr})
	var r gioinput.Router
	var ops op.Ops
	th.FocusTerminal()
	for range 2 {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(200, 70)), Source: r.Source(), Now: time.Now()}
		th.termSurface(gtx, v, &v.Panes[0], true, image.Point{})
		r.Frame(&ops)
	}
	cells := th.termMetrics(layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}})
	img := drawn(t, image.Pt(200, 70), 1, func(gtx C) D {
		gtx.Source = r.Source()
		th.termSurface(gtx, v, &v.Panes[0], true, image.Point{})
		return D{Size: gtx.Constraints.Max}
	})
	wantPixel(t, img, image.Pt(int(termPad+cells.w/2), termPad+cells.h/2), P.TerminalCursor, "block cursor")
}

// The scrollbar shows once there's history: its 4dp thumb, 2dp in from the
// right, at the bottom while the view is at the latest.
func TestTermScrollbar(t *testing.T) {
	th := NewTheme()
	scr := vt.NewSafeEmulator(20, 3)
	for range 10 {
		scr.Write([]byte("line\r\n"))
	}
	img := termPaneShot(t, th, scr, 200, 70, nil)
	wantPixel(t, img, image.Pt(200-4, 70-termPad-2), over(P.AppScrollbarThumb, P.TerminalBackground), "thumb")
	wantPixel(t, img, image.Pt(200-4, termPad+2), P.TerminalBackground, "track above the thumb")
}

// The copied text: a line per row with trailing blanks gone, rows that
// fill the width joined as the one line they wrapped from; a double click
// takes the word.
func TestTermSelectionText(t *testing.T) {
	scr := vt.NewSafeEmulator(10, 4)
	scr.Write([]byte("one   two\r\nabcdefghijklm\r\nlast"))
	geo := termGeom{cols: 10, rows: 4}
	sel := termSel{on: true, from: termPos{0, 0}, to: termPos{3, 9}, mode: 1}
	if got := termSelText(scr, geo, sel); got != "one   two\nabcdefghijklm\nlast" {
		t.Errorf("copied %q", got)
	}
	if a, b := termWord(scr, geo, termPos{0, 7}); a.col != 6 || b.col != 8 {
		t.Errorf("the word at col 7 is %d-%d, want 6-8", a.col, b.col)
	}
	if a, b := termLine(scr, geo, termPos{2, 1}); a.row != 1 || b.row != 2 {
		t.Errorf("the line at row 2 is rows %d-%d, want 1-2", a.row, b.row)
	}
}

// The keys the terminal keeps: word and line moves, delete to line start
// and clear go to the shell as T3's sequences; the drawer's chords split,
// add and close; the rest go through the encoder.
func TestTermLocalKeys(t *testing.T) {
	th := NewTheme()
	var sent []string
	tp := &TermPane{Send: func(s string) { sent = append(sent, s) }}
	p := &termPane{}
	gtx := layout.Context{Ops: new(op.Ops)}
	mac := runtime.GOOS == "darwin"
	cases := []struct {
		e    key.Event
		want string
	}{
		{key.Event{Name: key.NameLeftArrow, Modifiers: key.ModAlt}, "\x1bb"},
		{key.Event{Name: key.NameRightArrow, Modifiers: key.ModAlt}, "\x1bf"},
		{key.Event{Name: "L", Modifiers: key.ModCtrl}, "\x0c"},
	}
	if mac {
		cases = append(cases,
			struct {
				e    key.Event
				want string
			}{key.Event{Name: key.NameLeftArrow, Modifiers: key.ModCommand}, "\x01"},
			struct {
				e    key.Event
				want string
			}{key.Event{Name: key.NameDeleteBackward, Modifiers: key.ModCommand}, "\x15"})
	}
	for _, c := range cases {
		sent = nil
		if !th.termLocalKey(gtx, tp, p, c.e, mac) || len(sent) != 1 || sent[0] != c.want {
			t.Errorf("%v sent %q, want %q", c.e, sent, c.want)
		}
	}
	for _, c := range []struct {
		e    key.Event
		want string
	}{
		{key.Event{Name: "D", Modifiers: key.ModShortcut}, "terminal.split"},
		{key.Event{Name: "D", Modifiers: key.ModShortcut | key.ModShift}, "terminal.splitv"},
		{key.Event{Name: "N", Modifiers: key.ModShortcut}, "terminal.new"},
		{key.Event{Name: "W", Modifiers: key.ModShortcut}, "terminal.close"},
	} {
		th.Actions()
		if !th.termChord(&TermView{}, c.e) {
			t.Errorf("%v isn't a chord", c.e)
		}
		if acts := th.Actions(); len(acts) != 1 || acts[0].ID != c.want {
			t.Errorf("%v: actions %+v, want %s", c.e, acts, c.want)
		}
	}
	if th.termChord(&TermView{}, key.Event{Name: "C", Modifiers: key.ModCtrl | key.ModAlt}) {
		t.Error("Ctrl+Alt+C is the shell's")
	}
}

// The encoder: named keys, and letters with Ctrl, or Alt off a Mac.
func TestTermKeys(t *testing.T) {
	cases := []struct {
		e    key.Event
		mac  bool
		want string
	}{
		{key.Event{Name: "C", Modifiers: key.ModCtrl}, true, "\x03"},
		{key.Event{Name: key.NameReturn}, false, "\r"},
		{key.Event{Name: key.NameUpArrow}, false, "\x1b[A"},
		{key.Event{Name: key.NameDeleteBackward}, true, "\x7f"},
		{key.Event{Name: "B", Modifiers: key.ModAlt}, false, "\x1bb"},
	}
	for _, c := range cases {
		k, ok := termKey(c.e, c.mac)
		if !ok {
			t.Fatalf("%v: not a terminal key", c.e)
		}
		emu := vt.NewEmulator(10, 2)
		go emu.SendKey(k)
		buf := make([]byte, 16)
		n, _ := emu.Read(buf)
		if got := string(buf[:n]); got != c.want {
			t.Errorf("%v = %q, want %q", c.e, got, c.want)
		}
	}
	// Plain letters come as typed text, and a Mac's Option types its own.
	for _, e := range []key.Event{{Name: "A"}, {Name: "E", Modifiers: key.ModAlt}} {
		if _, ok := termKey(e, true); ok {
			t.Errorf("%v should be left to text input", e)
		}
	}
}

// A drag selects from cell to cell; a click without moving clears it; a
// release over a selection brings up its menu, Copy alone.
func TestTermPointerSelection(t *testing.T) {
	th := NewTheme()
	scr := vt.NewSafeEmulator(20, 3)
	scr.Write([]byte("hello world"))
	v := termTestView([]TermScreen{scr})
	var r gioinput.Router
	var ops op.Ops
	frame := func(evs ...pointer.Event) {
		for _, e := range evs {
			r.Queue(e)
		}
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(200, 70)), Source: r.Source(), Now: time.Now()}
		th.terminalDrawer(gtx, &Scene{Term: v})
		r.Frame(&ops)
	}
	frame()
	cells := th.termMetrics(layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}})
	at := func(col int) f32.Point {
		return f32.Pt(termPad+(float32(col)+0.5)*cells.w, 1+termPad+float32(cells.h)/2)
	}
	frame(pointer.Event{Kind: pointer.Press, Position: at(0), Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Drag, Position: at(4), Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at(4), Source: pointer.Mouse})
	p := th.pane(v, v.Panes[0].ID)
	if got := termSelText(scr, p.g, p.sel); got != "hello" {
		t.Fatalf("selected %q, want hello", got)
	}
	if m := th.in.terms.menu; m == nil || len(m.items) != 1 || m.items[0].label != "Copy" {
		t.Fatalf("selection menu %+v, want Copy", m)
	}
	frame(pointer.Event{Kind: pointer.Press, Position: at(8), Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at(8), Source: pointer.Mouse})
	if p.sel.on || th.in.terms.menu != nil {
		t.Error("a click should clear the selection and its menu")
	}
}

// With the app tracking the mouse, presses go to it as reports instead.
func TestTermMouseReport(t *testing.T) {
	th := NewTheme()
	scr := vt.NewSafeEmulator(20, 3)
	v := termTestView([]TermScreen{scr})
	var got []uv.MouseEvent
	v.Panes[0].Mouse = TermMousePress
	v.Panes[0].Report = func(m uv.MouseEvent) { got = append(got, m) }
	var r gioinput.Router
	var ops op.Ops
	frame := func(evs ...pointer.Event) {
		for _, e := range evs {
			r.Queue(e)
		}
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(200, 70)), Source: r.Source(), Now: time.Now()}
		th.termSurface(gtx, v, &v.Panes[0], true, image.Point{})
		r.Frame(&ops)
	}
	frame()
	frame(pointer.Event{Kind: pointer.Press, Position: f32.Pt(20, 10), Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: f32.Pt(20, 10), Source: pointer.Mouse})
	if len(got) != 2 {
		t.Fatalf("reports %v, want a press and a release", got)
	}
	if _, ok := got[0].(uv.MouseClickEvent); !ok || got[0].Mouse().Button != uv.MouseLeft {
		t.Errorf("first report %v, want a left press", got[0])
	}
	if th.pane(v, v.Panes[0].ID).sel.on {
		t.Error("a reported press shouldn't select")
	}
}
