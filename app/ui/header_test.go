package ui

import (
	"image"
	"runtime"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	gioinput "gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// live lays a widget out frame by frame with input, at 1dp to the pixel,
// and can render its last frame like drawn does.
type live struct {
	th   *Theme
	r    gioinput.Router
	ops  op.Ops
	size image.Point
	now  time.Time
	draw func(gtx C)
}

func newLive(size image.Point, draw func(gtx C)) *live {
	return &live{th: NewTheme(), size: size, now: time.Now(), draw: draw}
}

// frame runs one frame, d after the last, and returns its actions.
func (l *live) frame(d time.Duration) []Action {
	l.now = l.now.Add(d)
	l.ops.Reset()
	gtx := layout.Context{Ops: &l.ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(l.size), Source: l.r.Source(), Now: l.now}
	l.draw(gtx)
	l.r.Frame(&l.ops)
	return l.th.Actions()
}

// pixels renders the last frame.
func (l *live) pixels(t *testing.T) *image.RGBA {
	t.Helper()
	win, err := headless.NewWindow(l.size.X, l.size.Y)
	if err != nil {
		if runtime.GOOS == "linux" {
			t.Fatalf("headless GPU: %v", err)
		}
		t.Skipf("no headless GPU here: %v", err)
	}
	defer win.Release()
	if err := win.Frame(&l.ops); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: l.size})
	if err := win.Screenshot(img); err != nil {
		t.Fatal(err)
	}
	return img
}

func (l *live) move(x, y float32) {
	l.r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(x, y), Source: pointer.Mouse})
}

func (l *live) click(x, y float32) {
	p := f32.Pt(x, y)
	l.r.Queue(pointer.Event{Kind: pointer.Press, Position: p, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: p, Source: pointer.Mouse})
}

// headerScene is a server thread in a project, as the header shows it.
func headerScene() *Scene {
	return &Scene{Live: true, Title: "Wire up the parser", State: ThreadIdle,
		Header: &ChatHeader{Project: "atlas", Icon: ProjectIcon{Text: "AS", Color: "amber"}, Server: true, CanRename: true},
		Card:   &ThreadDetails{}}
}

// The crumbs share the list as T3's do (§3.1): the project crumb keeps its
// content width (icon, 6, text up to 160) while the title takes what's
// left; under pressure the title stops at 40 and only then does the
// project give way; a parent folds to "..." once the crumbs overflow.
func TestCrumbLayout(t *testing.T) {
	th := NewTheme()
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	h := &ChatHeader{Project: "atlas", Server: true}
	text := th.textWidth(gtx, crumbFont(P.MutedForeground), "atlas")
	l := th.layoutCrumbs(gtx, h, "Wire up the parser", 1000, false)
	if l.gap != 12 || l.project != 14+6+text || l.title != 1000-2*12-l.sep-l.project {
		t.Errorf("wide: %+v (project text %d)", l, text)
	}
	if n := th.layoutCrumbs(gtx, h, "x", 1000, true); n.gap != 8 {
		t.Errorf("narrow gap %d, want 8", n.gap)
	}
	tight := th.layoutCrumbs(gtx, h, "Wire up the parser", 70, false)
	if tight.title != 40 || tight.project != 70-24-tight.sep-40 {
		t.Errorf("tight: %+v", tight)
	}
	long := th.layoutCrumbs(gtx, &ChatHeader{Project: "a project with a really very long name indeed"}, "t", 2000, false)
	if long.project != 14+6+160 {
		t.Errorf("long project crumb %d, want its text capped at 160", long.project)
	}
	sub := &ChatHeader{Project: "atlas", Parent: "Plan the parser rewrite", Server: true}
	if l := th.layoutCrumbs(gtx, sub, "Child", 1000, false); l.collapsed || l.parent != th.textWidth(gtx, crumbFont(P.MutedForeground), sub.Parent) {
		t.Errorf("parent with room: %+v", l)
	}
	if l := th.layoutCrumbs(gtx, sub, "Child", 200, false); !l.collapsed || l.parent != th.textWidth(gtx, crumbFont(P.MutedForeground), "...") {
		t.Errorf("parent without room: %+v", l)
	}
}

// The header draws its crumbs centred on their caps in the 52dp row: the
// project's monogram tile at y 19, the title's chevron only while the
// title is hovered, and the keyboard's ring around the title it focuses.
func TestHeaderDrawn(t *testing.T) {
	defer SetDark(false)
	SetDark(true)
	s := headerScene()
	l := newLive(image.Pt(900, 52), nil)
	l.draw = func(gtx C) {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		defer op.Offset(image.Pt(20, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(800, 52))
		l.th.chatHeader(g, s, image.Pt(20, 0), false)
	}
	l.frame(0)
	img := l.pixels(t)
	// The tile is amber-400 at 14% over the background; its top edge is
	// clear of the letters.
	wantPixel(t, img, image.Pt(20+7, 19+1), over(alpha(Tailwind["amber-400"], 0x24), P.Background), "monogram tile")
	// Where the chevron goes, past the title's text: nothing at rest.
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	lay := l.th.layoutCrumbs(gtx, s.Header, s.Title, 800, false)
	titleX := 20 + lay.project + lay.gap + lay.sep + lay.gap
	textW := l.th.textWidth(gtx, crumbFont(P.Foreground), s.Title)
	chev := image.Pt(titleX+textW+4+7, 26+1)
	wantPixel(t, img, chev, P.Background, "chevron at rest")
	l.move(float32(titleX+5), 26)
	l.frame(0)
	l.frame(200 * time.Millisecond)
	img = l.pixels(t)
	if got := img.RGBAAt(chev.X, chev.Y); got.R < 0x30 {
		t.Errorf("chevron on hover is #%02x%02x%02x, want it drawn in muted", got.R, got.G, got.B)
	}
	// Tab from the project crumb to the title: the ring is the title's.
	l.move(890, 50)
	l.frame(0)
	l.r.MoveFocus(key.FocusForward)
	l.frame(0)
	l.r.MoveFocus(key.FocusForward)
	l.frame(0)
	img = l.pixels(t)
	wantPixel(t, img, image.Pt(titleX, 26), P.Ring, "title's focus ring")
	wantPixel(t, img, image.Pt(20, 26), over(alpha(Tailwind["amber-400"], 0x24), P.Background), "no ring on the project crumb")
	if acts := func() []Action {
		l.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
		return l.frame(0)
	}(); len(acts) == 0 || acts[0].ID != "header.title" || acts[0].Kind != "click" {
		t.Errorf("Enter on the title: %+v", acts)
	}
}

// A click on the title is a click; the second of a double click asks to
// rename instead, and the menu is placed under the title.
func TestHeaderTitleClicks(t *testing.T) {
	s := headerScene()
	l := newLive(image.Pt(900, 52), nil)
	l.draw = func(gtx C) {
		g := gtx
		g.Constraints = layout.Exact(image.Pt(800, 52))
		l.th.chatHeader(g, s, image.Pt(0, 0), false)
	}
	l.frame(0)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	lay := l.th.layoutCrumbs(gtx, s.Header, s.Title, 800, false)
	x := float32(lay.project + lay.gap + lay.sep + lay.gap + 10)
	l.click(x, 26)
	acts := l.frame(10 * time.Millisecond)
	if len(acts) != 1 || acts[0] != (Action{ID: "header.title", Kind: "click", Clicks: 1}) {
		t.Fatalf("click: %+v", acts)
	}
	l.th.MenuFrom("header.title")
	if want := image.Pt(lay.project+lay.gap+lay.sep+lay.gap, 26-crumbBox(gtx)/2+crumbBox(gtx)+4); l.th.in.ctxAt != want {
		t.Errorf("menu at %v, want %v: under the title, 4dp down", l.th.in.ctxAt, want)
	}
	l.click(x, 26)
	if acts = l.frame(10 * time.Millisecond); len(acts) != 1 || acts[0].Kind != "double" {
		t.Fatalf("double click: %+v", acts)
	}
	// A draft's title is plain text: no menu, no rename.
	s.Header.Server = false
	l.frame(time.Second)
	l.click(x, 26)
	for _, a := range l.frame(10 * time.Millisecond) {
		if a.ID == "header.title" {
			t.Fatalf("a draft's title acted: %+v", a)
		}
	}
}

// Renaming swaps the title for an editor in a ring of --ring at half
// strength.
func TestHeaderRename(t *testing.T) {
	s := headerScene()
	s.Header.Renaming = true
	th := NewTheme()
	th.SetEditorText("header.title", s.Title)
	img := drawn(t, image.Pt(800, 52), 1, func(gtx C) D {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		return th.chatHeader(gtx, s, image.Point{}, false)
	})
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	lay := th.layoutCrumbs(gtx, s.Header, s.Title, 800, false)
	x := lay.project + lay.gap + lay.sep + lay.gap
	wantPixel(t, img, image.Pt(x, 26), over(alpha(P.Ring, 0x80), P.Background), "rename ring")
	wantPixel(t, img, image.Pt(x+lay.title-1, 26), over(alpha(P.Ring, 0x80), P.Background), "the editor spans the title's crumb")
}

// darkest is the lowest red in r: in light mode, how dark the ink there
// gets, 0xff where nothing is drawn.
func darkest(img *image.RGBA, r image.Rectangle) uint8 {
	low := uint8(0xff)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			low = min(low, img.RGBAAt(x, y).R)
		}
	}
	return low
}

// A shared thread's guest link ends the row (§2.2, §3.1): the 14dp user
// icon, its status in 12dp muted and Stop sharing, a ghost-muted xs
// button, 4dp apart, 12 after the breadcrumb, which gives it the room.
// Stop sharing fills with --accent under the pointer, clicks, is the Tab
// stop after the title with T3's ring inside its border, and is off at 64%
// while the close is on its way.
func TestHeaderGuestShare(t *testing.T) {
	s := headerScene()
	s.Title = "Wire up the parser and the lexer and everything that comes after them, all of it"
	g := &GuestLine{Status: "Sam is a guest"}
	s.Header.Guest = g
	l := newLive(image.Pt(900, 52), nil)
	l.draw = func(gtx C) {
		fill(gtx, P.Background, image.Rectangle{Max: gtx.Constraints.Max}, 0)
		defer op.Offset(image.Pt(20, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(800, 52))
		l.th.chatHeader(g, s, image.Pt(20, 0), false)
	}
	l.frame(0)
	img := l.pixels(t)
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Constraints{Max: image.Pt(2000, 52)}}
	x0 := 20 + 800 - l.th.guestLineW(gtx, g)
	textW := l.th.textWidth(gtx, guestFont(), g.Status)
	bx := x0 + 14 + 4 + textW + 4
	if ink := darkest(img, image.Rect(x0, 19, x0+14, 33)); ink > 0xc0 {
		t.Errorf("no user icon in its 14dp box (darkest red %#x)", ink)
	}
	if ink := darkest(img, image.Rect(x0+18, 14, x0+18+textW, 38)); ink < P.MutedForeground.R-3 || ink > P.MutedForeground.R+0x20 {
		t.Errorf("the status's ink reaches %#x, want muted foreground's %#x", ink, P.MutedForeground.R)
	}
	if ink := darkest(img, image.Rect(x0-12+2, 0, x0, 52)); ink != P.Background.R {
		t.Errorf("ink in the 12dp gap before the link (%#x): the title should stop short of it", ink)
	}
	if ink := darkest(img, image.Rect(x0-12-30, 14, x0-12, 38)); ink > 0x60 {
		t.Errorf("the long title should run up to the gap (darkest red there %#x)", ink)
	}
	wantPixel(t, img, image.Pt(bx+3, 26), P.Background, "Stop sharing at rest, no fill")

	l.move(float32(bx+3), 26)
	l.frame(0)
	l.frame(0)
	img = l.pixels(t)
	wantPixel(t, img, image.Pt(bx+3, 26), P.Accent, "Stop sharing under the pointer")
	if ink := darkest(img, image.Rect(bx+8, 14, bx+40, 38)); ink > 0x40 {
		t.Errorf("Stop sharing's label under the pointer is %#x, want foreground", ink)
	}
	l.click(float32(bx+3), 26)
	if acts := l.frame(10 * time.Millisecond); len(acts) != 1 || acts[0] != (Action{ID: "header.unshare", Kind: "click", Clicks: 1}) {
		t.Fatalf("click: %+v", acts)
	}

	// Tab: the project crumb, the title, then Stop sharing.
	l.move(890, 50)
	l.frame(0)
	for range 3 {
		l.r.MoveFocus(key.FocusForward)
		l.frame(0)
	}
	img = l.pixels(t)
	wantPixel(t, img, image.Pt(bx+1, 26), P.Background, "the ring's --background band")
	wantPixel(t, img, image.Pt(bx+2, 26), P.Ring, "the ring, inside the border and the band")
	wantPixel(t, img, image.Pt(bx+3, 26), P.Ring, "the ring's second dp")
	wantPixel(t, img, image.Pt(bx+4, 26), P.Background, "inside the ring")
	l.r.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	if acts := l.frame(0); len(acts) != 1 || acts[0].ID != "header.unshare" {
		t.Fatalf("Enter on Stop sharing: %+v", acts)
	}

	g.Closing = true
	l.frame(time.Second)
	l.click(float32(bx+3), 26)
	if acts := l.frame(10 * time.Millisecond); len(acts) != 0 {
		t.Fatalf("Stop sharing while closing: %+v", acts)
	}
	img = l.pixels(t)
	if ink := darkest(img, image.Rect(bx+8, 14, bx+40, 38)); ink < 0x90 {
		t.Errorf("the closing button's label reaches %#x, want muted at 64%%", ink)
	}
}
