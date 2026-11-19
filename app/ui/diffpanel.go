package ui

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"

	"github.com/some-du6e/tern/app/diff"
)

// The right panel's Diff surface, ported from T3 Code's DiffPanel.tsx
// (docs/ui/13-diff.md): a header row with the scope menu and the
// toolbar, then the code view and the file tree pane, or a loading,
// empty or error state. Targets: "diff.tab", "diff.close", "diff.scope",
// "diff.scope:<value>", "diff.commit:<sha>", "diff.refresh",
// "diff.collapse", "diff.layout:stacked|split", "diff.wrap",
// "diff.whitespace", "diff.tests", "diff.tree", "diff.file:<path>",
// "diff.folder:<path>", "diff.folders", "diff.base",
// "diff.base:<ref>"; field "diff.base.search".

// DiffPanel is the open thread's Diff surface.
type DiffPanel struct {
	Active bool // its tab is the one the right panel shows
	// Key names what it shows, the thread and scope: the code view keeps a
	// scroll position for each. Rev counts the patches fetched, so a new
	// one lays out afresh.
	Key string
	Rev int

	// The header row: the scope menu's trigger (Scope its label, Value the
	// menu's checked row), a commit's short SHA after it, and the branch
	// scope's head → base row.
	Scope, Value string
	SHA          string
	Menu         *DiffScopeMenu
	Compare      *DiffCompare

	// The toolbar. Stat is the files' totals, nil without files.
	Stat       *DiffStat
	Refresh    bool // a Git scope: the refresh button
	Refreshing bool
	Split      bool
	Wrap       bool
	Whitespace bool // whitespace changes are hidden
	TestFiles  int  // the test files among them, for "Hide test files"
	HideTests  bool
	TreeOpen   bool

	// The body: a panel message, the loading skeleton, or the files, under
	// any notices and a load error.
	Empty   string
	Loading string
	Notices []string
	Error   string
	NoPatch string // a loaded patch with nothing to draw: why
	Raw     *DiffRaw
	Files   []DiffFile
	Tree    []diff.TreeEntry
	// Revealed is the file the tree or a card last revealed: the tree
	// marks it, and each new RevealID scrolls the code view to it.
	Revealed string
	RevealID int
}

// menuOpen reports whether the scope menu or the base ref picker is up.
func (d *DiffPanel) menuOpen() bool {
	return d != nil && (d.Menu != nil || d.Compare != nil && d.Compare.Picker != nil)
}

// DiffStat is a DiffStatLabel's counts.
type DiffStat struct{ Additions, Deletions int }

// DiffRaw is a patch the viewer couldn't read, shown as text.
type DiffRaw struct{ Reason, Text string }

// DiffFile is one file of the code view.
type DiffFile struct {
	diff.File
	Collapsed bool
}

// DiffScopeMenu is the open scope menu, and its open submenu: "turn" or
// "commits".
type DiffScopeMenu struct {
	Sub     string
	Turns   []DiffTurn
	Turn    string // the selected turn's row in the Turn submenu
	Commits []DiffCommit
	// Note is the disabled row under the commits: why there are none, or
	// that only the latest are listed.
	Note string
	// Commit is the selected commit's SHA.
	Commit string
}

// DiffTurn is a row of the Turn submenu: "Turn 16" and when it finished.
type DiffTurn struct {
	Value, Label, Time string
}

// DiffCommit is a row of the Commits submenu.
type DiffCommit struct {
	SHA, Subject, When string
}

// DiffCompare is the branch scope's "head → base" row and its base ref
// picker.
type DiffCompare struct {
	Head, Base string
	Picker     *DiffBasePicker
}

// DiffBasePicker is the open base ref combobox.
type DiffBasePicker struct {
	Automatic bool // the server picks the base: "Automatic" is checked
	Refs      []DiffBaseRef
	Selected  string
}

// DiffBaseRef is a ref the base can be: a local branch, its remote twin,
// or both, when a switch picks between them.
type DiffBaseRef struct {
	Label, Local, Remote string
}

// The surface's lengths (§3.1): its header row, inline in the right
// panel, and the 12dp under it.
const (
	diffHeaderH   = 28
	diffHeaderGap = 12
)

// diffSurface draws the Diff surface in the right panel below its tabs.
func (th *Theme) diffSurface(gtx C, s *Scene) D {
	d := s.Diff
	size := gtx.Constraints.Max
	fill(gtx, P.Background, image.Rectangle{Max: size}, 0)
	// A submenu flips at the panel's right edge, which is the window's.
	th.dv.panelW, th.dv.menuX = size.X, dp(gtx, 8)
	if d.Menu == nil {
		th.dv.sub = ""
	}
	top := dp(gtx, diffHeaderH+diffHeaderGap)
	func() {
		defer op.Offset(image.Pt(0, top)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X, max(0, size.Y-top)))
		th.surfaceBody(g, s, d)
	}()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(size.X, dp(gtx, diffHeaderH)))
	th.diffHeaderRow(g, s, d)
	return D{Size: size}
}

// diffHeaderRow is the scope trigger and its neighbours on the left, the
// toolbar on the right, 8dp in from both sides.
func (th *Theme) diffHeaderRow(gtx C, s *Scene, d *DiffPanel) {
	size := gtx.Constraints.Max
	pad := dp(gtx, 8)
	// The toolbar first: it never shrinks, the left group takes the rest.
	m := op.Record(gtx.Ops)
	tg := gtx
	tg.Constraints.Min = image.Point{}
	tw := th.diffToolbar(tg, d)
	bar := m.Stop()
	func() {
		defer op.Offset(image.Pt(size.X-pad-tw.Size.X, (size.Y-tw.Size.Y)/2)).Push(gtx.Ops).Pop()
		bar.Add(gtx.Ops)
	}()
	left := max(0, size.X-2*pad-tw.Size.X-dp(gtx, 8))
	defer op.Offset(image.Pt(pad, 0)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(left, size.Y))
	th.diffScopeGroup(g, s, d)
}

// diffScopeGroup is the scope menu's trigger, a commit's SHA and the
// comparison row, 12dp apart.
func (th *Theme) diffScopeGroup(gtx C, s *Scene, d *DiffPanel) {
	size := gtx.Constraints.Max
	x := 0
	// The trigger may shrink only when a SHA follows it.
	shaW := 0
	if d.SHA != "" {
		shaW = th.lineWidth(gtx, Txt{Size: 12, Font: Mono()}, d.SHA) + dp(gtx, 12)
	}
	tw := th.menuTrigger(gtx, "diff.scope", d.Scope, BtnSecondary, d.Menu != nil, size.X-shaW, image.Pt(x, (size.Y-dp(gtx, 24))/2))
	// The menu hangs 4dp under the trigger, its left edge on the trigger's.
	if d.Menu != nil {
		th.deferBelow(gtx, image.Pt(x, (size.Y+dp(gtx, 24))/2+dp(gtx, 4)), func(gtx C) D {
			return th.diffScopeMenu(gtx, s, d)
		})
	}
	x += tw + dp(gtx, 12)
	if d.SHA != "" {
		func() {
			defer op.Offset(image.Pt(x, (size.Y-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			th.Text(gtx, Txt{Size: 12, LineH: 16.0 / 12, Font: Mono(), Color: P.MutedForeground, MaxLines: 1}, d.SHA)
		}()
		x += shaW
	}
	if c := d.Compare; c != nil && x < size.X {
		defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X-x, size.Y))
		th.diffCompareRow(g, s, c)
	}
}

// diffCompareRow is "<head> → <base>", the base a ghost-muted trigger for
// the ref picker; too narrow for both refs (under 20rem), "vs <base>".
func (th *Theme) diffCompareRow(gtx C, s *Scene, c *DiffCompare) {
	size := gtx.Constraints.Max
	muted := Txt{Size: 12, LineH: 16.0 / 12, Color: P.MutedForeground, MaxLines: 1}
	baseLabel := c.Base
	x := 0
	if size.X >= dp(gtx, 320) {
		hw := min(th.lineWidth(gtx, muted, c.Head), dp(gtx, 192))
		func() {
			defer op.Offset(image.Pt(0, (size.Y-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(hw, dp(gtx, 16))}
			th.Text(g, muted, c.Head)
		}()
		x = hw + dp(gtx, 8)
		func() {
			defer op.Offset(image.Pt(x, (size.Y-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
			LucideW("arrow-right", over(alpha(P.MutedForeground, 0xb3), P.Background), 14)(gtx)
			th.tipHover(gtx, "diff.refs", image.Pt(dp(gtx, 14), dp(gtx, 14)))
			th.tipAt(gtx, "diff.refs", c.Head+" → "+c.Base, image.Pt(dp(gtx, 14), dp(gtx, 14)), true, 0)
		}()
		x += dp(gtx, 14+8)
	} else {
		func() {
			defer op.Offset(image.Pt(0, (size.Y-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
			x = th.Text(gtx, muted, "vs").Size.X + dp(gtx, 8)
		}()
	}
	th.menuTrigger(gtx, "diff.base", baseLabel, BtnGhostMuted, c.Picker != nil, min(size.X-x, dp(gtx, 192)), image.Pt(x, (size.Y-dp(gtx, 24))/2))
	if c.Picker != nil {
		th.deferBelow(gtx, image.Pt(x, (size.Y+dp(gtx, 24))/2+dp(gtx, 4)), func(gtx C) D {
			return th.diffBasePicker(gtx, s, c)
		})
	}
}

// menuTrigger is a menu's xs Button with its label then a 14dp chevron at
// 70% (DiffPanel.tsx's scope and base triggers): 24 tall, padded 7, at
// most maxW wide, its label cut short to fit. It returns its width.
func (th *Theme) menuTrigger(gtx C, id, label string, v ButtonVariant, open bool, maxW int, at image.Point) int {
	defer op.Offset(at).Push(gtx.Ops).Pop()
	b := Button{Variant: v}
	cl := b.colors()
	c, _ := th.events(gtx, id)
	hot := c.Hovered() || open
	bg, fg := cl.bg, cl.text
	if hot {
		if cl.bgHover.A != 0 {
			bg = cl.bgHover
		}
		if cl.textHover.A != 0 {
			fg = cl.textHover
		}
	}
	t := Txt{Size: 12, LineH: 16.0 / 12, Font: UI(font.Medium), Color: fg, MaxLines: 1}
	px := max(1, dp(gtx, 1))
	chrome := 2*dp(gtx, 7) + 2*px + dp(gtx, 4+10)
	lw := min(th.lineWidth(gtx, t, label), max(0, maxW-chrome))
	w, h := lw+chrome, dp(gtx, 24)
	r := image.Rectangle{Max: image.Pt(w, h)}
	if bg.A != 0 {
		fill(gtx, over(bg, P.Background), r, dp(gtx, 8))
	}
	func() {
		defer op.Offset(image.Pt(px+dp(gtx, 7), (h-dp(gtx, 16))/2)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(lw, dp(gtx, 16))}
		th.Text(g, t, label)
	}()
	func() {
		defer op.Offset(image.Pt(px+dp(gtx, 7)+lw+dp(gtx, 4-2), (h-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
		LucideW("chevron-down", over(alpha(fg, 0xb3), over(bg, P.Background)), 14)(gtx)
	}()
	th.hit(gtx, id, r.Max)
	return w
}

// lineWidth is how wide t sets s on one line, in px.
func (th *Theme) lineWidth(gtx C, t Txt, s string) int {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
	t.MaxLines = 1
	w := th.Text(g, t, s).Size.X
	m.Stop()
	return w
}

// tipHover makes a size-sized area at the current offset report hovering as
// target id, for a tooltip, without a pointer cursor. Its clicks go to no
// one.
func (th *Theme) tipHover(gtx C, id string, size image.Point) {
	c, _ := th.events(gtx, id)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	c.Add(gtx.Ops)
}

// diffTool is one toolbar control: a T3 Button or Toggle w by h, its
// lucide icon centred.
type diffTool struct {
	id, tip, icon string
	w, h, radius  float32
	bg, hot       color.NRGBA // translucent fills over the surface
	col           color.NRGBA
	alpha         float32 // the icon's opacity, 0 for 1
	shadow        bool    // a pressed segment's 0 1px 2px
	tipAlign      int
}

func (th *Theme) diffToolW(t diffTool, on color.NRGBA) W {
	return func(gtx C) D {
		c, _ := th.events(gtx, t.id)
		w, h := dp(gtx, t.w), dp(gtx, t.h)
		r := image.Rectangle{Max: image.Pt(w, h)}
		rad := dp(gtx, t.radius)
		bg := t.bg
		if c.Hovered() && t.hot.A != 0 {
			bg = t.hot
		}
		if t.shadow {
			shadow(gtx, r, rad, []boxShadow{{0, 1, 2, 0, alpha(rgb(0), 0x1a)}})
		}
		if bg.A != 0 {
			fill(gtx, over(bg, on), r, rad)
		}
		col := t.col
		if t.alpha > 0 {
			col.A = uint8(float32(col.A)*t.alpha + 0.5)
		}
		func() {
			defer op.Offset(image.Pt((w-dp(gtx, 14))/2, (h-dp(gtx, 14))/2)).Push(gtx.Ops).Pop()
			if t.icon == "refreshing" {
				// It turns once a second while the refresh runs.
				c := float32(dp(gtx, 14)) / 2
				turn := float32(gtx.Now.UnixMilli()%1000) / 1000 * 2 * math.Pi
				defer op.Affine(f32.Affine2D{}.Rotate(f32.Pt(c, c), turn)).Push(gtx.Ops).Pop()
				gtx.Execute(op.InvalidateCmd{})
				LucideW("refresh-cw", over(col, over(bg, on)), 14)(gtx)
				return
			}
			LucideW(t.icon, over(col, over(bg, on)), 14)(gtx)
		}()
		th.hit(gtx, t.id, r.Max)
		th.tipAt(gtx, t.id, t.tip, r.Max, true, t.tipAlign)
		return D{Size: r.Max}
	}
}

// diffToolbar is the header row's right side (§3.1): the totals, then
// refresh, collapse all, the stacked/split pair and the wrap, whitespace,
// test file and file tree toggles, 4dp apart.
func (th *Theme) diffToolbar(gtx C, d *DiffPanel) D {
	bg := P.Background
	ghost := func(id, tip, icon string) diffTool {
		return diffTool{id: id, tip: tip, icon: icon, w: 28, h: 28, radius: 8, hot: P.Accent, col: P.MutedForeground}
	}
	toggle := func(id, tip, icon string, on bool) diffTool {
		t := diffTool{id: id, tip: tip, icon: icon, w: 28, h: 28, radius: 8, hot: P.Accent, col: P.Foreground, alpha: 0.8}
		if on {
			t.bg = P.Accent
		}
		return t
	}
	var kids []layout.FlexChild
	gap := func() { kids = append(kids, rigid(Spacer(4, 0))) }
	if d.SHA != "" {
		gap = func() {}
	}
	add := func(w W) {
		if len(kids) > 0 {
			gap()
		}
		kids = append(kids, rigid(w))
	}
	if st := d.Stat; st != nil {
		add(func(gtx C) D {
			return pad(0, 4, 0, 0).Layout(gtx, func(gtx C) D { return th.diffStat(gtx, st.Additions, st.Deletions, false, 11, 16) })
		})
	}
	if d.Refresh {
		t := ghost("diff.refresh", "Refresh diff", "refresh-cw")
		if d.Refreshing {
			t.tip, t.icon = "Refreshing diff…", "refreshing"
		}
		add(th.diffToolW(t, bg))
	}
	if len(d.Files) > 0 {
		collapsed := true
		for _, f := range d.Files {
			collapsed = collapsed && f.Collapsed
		}
		if collapsed {
			add(th.diffToolW(ghost("diff.collapse", "Expand all files", "chevrons-up-down"), bg))
		} else {
			add(th.diffToolW(ghost("diff.collapse", "Collapse all files", "chevrons-down-up"), bg))
		}
	}
	add(th.diffLayoutGroup(d))
	wrap := toggle("diff.wrap", "Enable line wrapping", "text-wrap", d.Wrap)
	if d.Wrap {
		wrap.tip = "Disable line wrapping"
	}
	add(th.diffToolW(wrap, bg))
	ws := toggle("diff.whitespace", "Hide whitespace changes", "pilcrow", d.Whitespace)
	if d.Whitespace {
		ws.tip = "Show whitespace changes"
	}
	add(th.diffToolW(ws, bg))
	if d.TestFiles > 0 {
		tests := toggle("diff.tests", "Hide test files", "flask-conical-off", d.HideTests)
		if d.HideTests {
			tests.tip = "Show " + plural(d.TestFiles, "hidden test file")
		}
		add(th.diffToolW(tests, bg))
	}
	if len(d.Files) > 0 {
		tree := toggle("diff.tree", "Show file tree", "folder-tree", d.TreeOpen)
		if d.TreeOpen {
			tree.tip = "Hide file tree"
		}
		tree.tipAlign = 1
		add(th.diffToolW(tree, bg))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
}

// diffLayoutGroup is the stacked/split ToggleGroup, segmented: input at
// 40% behind two 32x24 segments, the pressed one raised.
func (th *Theme) diffLayoutGroup(d *DiffPanel) W {
	return func(gtx C) D {
		w, h := dp(gtx, 70), dp(gtx, 28)
		bg := over(twAlpha(P.Input, 40), P.Background)
		fill(gtx, bg, image.Rect(0, 0, w, h), dp(gtx, 8))
		// Its segments have no tooltips, unlike the toggles after it.
		seg := func(id, icon string, on bool, x float32) {
			t := diffTool{id: id, icon: icon, w: 32, h: 24, radius: 6, col: P.MutedForeground, alpha: 0.8,
				hot: alpha(P.Background, 0x8c)}
			if dark {
				t.hot = alpha(rgb(0xffffff), 0x09)
			}
			if on {
				t.bg, t.hot, t.col, t.shadow = P.Background, color.NRGBA{}, P.Foreground, true
				if dark {
					t.bg = alpha(rgb(0xffffff), 0x14)
				}
			}
			defer op.Offset(image.Pt(dp(gtx, x), dp(gtx, 2))).Push(gtx.Ops).Pop()
			th.diffToolW(t, bg)(gtx)
		}
		seg("diff.layout:stacked", "rows-3", !d.Split, 2)
		seg("diff.layout:split", "columns-2", d.Split, 36)
		return D{Size: image.Pt(w, h)}
	}
}

// surfaceBody is the surface under the header row: a panel message, the
// loading skeleton, or the notices and error over the files.
func (th *Theme) surfaceBody(gtx C, s *Scene, d *DiffPanel) {
	size := gtx.Constraints.Max
	faint := Txt{Size: 12, LineH: 16.0 / 12, Color: over(alpha(P.MutedForeground, 0xb3), P.Background), Align: text.Middle}
	switch {
	case d.Empty != "":
		th.centredNote(gtx, faint, d.Empty, 20)
		return
	case d.Loading != "":
		th.diffSkeleton(gtx)
		return
	}
	y := 0
	for _, n := range d.Notices {
		y += th.diffNotice(gtx, n, y)
	}
	if d.Error != "" && len(d.Files) == 0 && d.Raw == nil {
		func() {
			defer op.Offset(image.Pt(dp(gtx, 12), y)).Push(gtx.Ops).Pop()
			g := gtx
			g.Constraints = layout.Constraints{Max: image.Pt(size.X-dp(gtx, 24), size.Y)}
			dd := th.Text(g, Txt{Size: 11, LineH: 16.0 / 11, Color: over(alpha(P.Error, 0xcc), P.Background)}, d.Error)
			y += dd.Size.Y + dp(gtx, 8)
		}()
	}
	defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(image.Pt(size.X, max(0, size.Y-y)))
	switch {
	case d.Raw != nil:
		th.diffRaw(g, d)
	case len(d.Files) == 0:
		if d.NoPatch != "" {
			th.centredNote(g, faint, d.NoPatch, 12)
		}
	default:
		th.diffFiles(g, s, d)
	}
}

// centredNote is a muted message in the middle of the area, padded x on
// both sides.
func (th *Theme) centredNote(gtx C, t Txt, msg string, x float32) {
	size := gtx.Constraints.Max
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Min: image.Pt(size.X-dp(gtx, 2*x), 0), Max: image.Pt(size.X-dp(gtx, 2*x), size.Y)}
	dd := th.Text(g, t, msg)
	call := m.Stop()
	defer op.Offset(image.Pt(dp(gtx, x), (size.Y-dd.Size.Y)/2)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// diffNotice is a notice strip over the files at y: 11/16 muted, padded
// 6 by 12, on muted at 40% over a border at 70%. It returns its height.
func (th *Theme) diffNotice(gtx C, msg string, y int) int {
	w := gtx.Constraints.Max.X
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(w-dp(gtx, 24), 1<<20)}
	dd := th.Text(g, Txt{Size: 11, LineH: 16.0 / 11, Color: P.MutedForeground}, msg)
	call := m.Stop()
	h := dd.Size.Y + dp(gtx, 12) + max(1, dp(gtx, 1))
	defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
	fill(gtx, over(twAlpha(P.Muted, 40), P.Background), image.Rect(0, 0, w, h), 0)
	hline(gtx, over(twAlpha(P.Border, 70), P.Background), h-max(1, dp(gtx, 1)), 0, w)
	defer op.Offset(image.Pt(dp(gtx, 12), dp(gtx, 6))).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return h
}

// diffRaw is a patch the viewer couldn't parse: why, then the text in a
// bordered block (§3.12).
func (th *Theme) diffRaw(gtx C, d *DiffPanel) {
	size := gtx.Constraints.Max
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer op.Offset(image.Pt(dp(gtx, 8), dp(gtx, 8))).Push(gtx.Ops).Pop()
	w := size.X - dp(gtx, 16)
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(w, size.Y)}
	rd := th.Text(g, Txt{Size: 11, LineH: 16.0 / 11, Color: over(alpha(P.MutedForeground, 0xbf), P.Background)}, d.Raw.Reason)
	y := rd.Size.Y + dp(gtx, 8)
	defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
	m := op.Record(gtx.Ops)
	g.Constraints = layout.Constraints{Max: image.Pt(w-dp(gtx, 24), int(0.72*float32(size.Y)))}
	t := Txt{Size: 11, LineH: 17.875 / 11, Font: Mono(), Color: over(alpha(P.MutedForeground, 0xe6), P.Background)}
	if !d.Wrap {
		t.MaxLines = 0
	}
	td := th.Text(g, t, d.Raw.Text)
	call := m.Stop()
	r := image.Rect(0, 0, w, td.Size.Y+dp(gtx, 24))
	rad := dp(gtx, 6)
	fill(gtx, over(alpha(P.Background, 0xb3), P.Background), r, rad)
	strokeRect(gtx, over(twAlpha(P.Border, 70), P.Background), r, rad, float32(max(1, dp(gtx, 1))))
	defer op.Offset(image.Pt(dp(gtx, 12), dp(gtx, 12))).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// diffSkeleton is T3's DiffPanelLoadingState (DiffPanelShell.tsx:92-116;
// §2.10, §3.12): a medium file header, a separator, three code lines, then
// a short and a long header, its bars breathing together.
func (th *Theme) diffSkeleton(gtx C) {
	size := gtx.Constraints.Max
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	a := skeletonAlpha(gtx)
	bar := over(alpha(P.MutedForeground, uint8(float32(0x26)*a+0.5)), P.Background)
	line := over(twAlpha(P.Border, 40), P.Background)
	// Rows are padded 8 on the left and 12 on the right.
	left, right := dp(gtx, 8), size.X-dp(gtx, 12)
	y := 0
	header := func(frac float32, maxW float32) {
		cy := y + dp(gtx, 16)
		x := left
		// The chevron's 20dp cell holds a 10dp square, then the icon's 20.
		fill(gtx, bar, image.Rect(x+dp(gtx, 5), cy-dp(gtx, 5), x+dp(gtx, 15), cy+dp(gtx, 5)), dp(gtx, 4))
		x += dp(gtx, 20+8)
		fill(gtx, bar, image.Rect(x, cy-dp(gtx, 10), x+dp(gtx, 20), cy+dp(gtx, 10)), dp(gtx, 4))
		x += dp(gtx, 20+8)
		tw := min(int(frac*float32(right-left)), dp(gtx, maxW))
		fill(gtx, bar, image.Rect(x, cy-dp(gtx, 6), x+tw, cy+dp(gtx, 6)), dp(gtx, 6))
		fill(gtx, bar, image.Rect(right-dp(gtx, 48), cy-dp(gtx, 6), right-dp(gtx, 28), cy+dp(gtx, 6)), dp(gtx, 6))
		fill(gtx, bar, image.Rect(right-dp(gtx, 20), cy-dp(gtx, 6), right, cy+dp(gtx, 6)), dp(gtx, 6))
		y += dp(gtx, 32)
	}
	header(1.0/2, 256)
	// The separator: a 96x10 pill between two hairlines, 8dp either side.
	cy := y + dp(gtx, 12)
	pw := dp(gtx, 96)
	px := left + (right-left-pw)/2
	hline(gtx, line, cy, left, px-dp(gtx, 8))
	hline(gtx, line, cy, px+pw+dp(gtx, 8), right)
	fill(gtx, bar, image.Rect(px, cy-dp(gtx, 5), px+pw, cy+dp(gtx, 5)), dp(gtx, 5))
	y += dp(gtx, 24)
	// Three code lines padded 8 by 12: a number pill, then the code's at a
	// fraction of the row.
	y += dp(gtx, 8)
	for _, frac := range []float32{2.0 / 3, 4.0 / 5, 3.0 / 5} {
		x, row := dp(gtx, 12), size.X-dp(gtx, 24)
		fill(gtx, bar, image.Rect(x, y, x+dp(gtx, 20), y+dp(gtx, 10)), dp(gtx, 5))
		x += dp(gtx, 20+12)
		fill(gtx, bar, image.Rect(x, y, x+int(frac*float32(row)), y+dp(gtx, 10)), dp(gtx, 5))
		y += dp(gtx, 10+8)
	}
	header(2.0/5, 208)
	header(3.0/5, 288)
}

// skeletonAlpha is T3's skeleton breath (index.css @keyframes skeleton,
// 2.4s): held at 1, down to .55 in four steps, held, and back.
func skeletonAlpha(gtx C) float32 {
	const period = 2400 * time.Millisecond
	t := float32(gtx.Now.UnixMilli()%period.Milliseconds()) / float32(period.Milliseconds())
	step := func(from, to, at float32) float32 { return from + (to-from)*float32(int(at*4))/4 }
	var a float32
	switch {
	case t < 0.42:
		a = 1
	case t < 0.5:
		a = step(1, 0.55, (t-0.42)/0.08)
	case t < 0.92:
		a = 0.55
	default:
		a = step(0.55, 1, (t-0.92)/0.08)
	}
	// The next step lands on a 2.4s grid of 20ms or longer holds.
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(20 * time.Millisecond)})
	return a
}

// diffFiles is the code view with the file tree pane beside it when open.
func (th *Theme) diffFiles(gtx C, s *Scene, d *DiffPanel) {
	size := gtx.Constraints.Max
	tw := 0
	if d.TreeOpen {
		tw = th.treePanePx(gtx, size.X)
	}
	func() {
		g := gtx
		g.Constraints = layout.Exact(image.Pt(size.X-tw, size.Y))
		th.codeView(g, s, d)
	}()
	if tw > 0 {
		defer op.Offset(image.Pt(size.X-tw, 0)).Push(gtx.Ops).Pop()
		g := gtx
		g.Constraints = layout.Exact(image.Pt(tw, size.Y))
		th.treePane(g, d)
	}
}

// diffTab is the Diff surface's tab in the right panel's tab row, in the
// browser tabs' style: its file-diff icon, "Diff" and a close button.
func (th *Theme) diffTab(gtx C, d *DiffPanel) D {
	bg, col := color.NRGBA{}, ColDim
	if d.Active {
		bg, col = ColFrameActive, ColText
	}
	return Middle(th.Click("diff.tab", func(gtx C) D {
		return Box{Bg: bg, Radius: 6, Pad: pad(5, 4, 5, 8)}.Layout(gtx, func(gtx C) D {
			return Row(gtx,
				rigid(Middle(LucideW("file-diff", col, 13))),
				rigid(Spacer(6, 0)),
				rigid(Middle(func(gtx C) D {
					return th.Text(gtx, Txt{Size: SizeSmall, Font: UI(font.Medium), Color: col, MaxLines: 1}, "Diff")
				})),
				rigid(Spacer(6, 0)),
				rigid(Middle(th.Click("diff.close", func(gtx C) D {
					return pad(2, 2, 2, 2).Layout(gtx, LucideW("x", ColFaint, 12))
				}))),
			)
		})
	}))(gtx)
}
