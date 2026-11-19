package ui

import (
	"image"
	"image/color"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The conversation timeline, ported from T3's MessagesTimeline
// (docs/ui/06-timeline.md): the rows in the chat lane, in a list as wide
// and tall as the canvas whose end clears the composer over it. The list
// follows the end while it grows and lets go when the reader scrolls away
// (§5.10); "Scroll to end" brings it back.

// timelineState is the timeline's state between frames.
type timelineState struct {
	follows map[string]*follow // each thread's list, by Timeline.ID
	order   []string           // the threads, least recently shown first
	hovers  map[string]*gesture.Hover
	docs    map[mdKey]*mdDoc
	layouts map[inlineKey]*inlineLayout
	codes   map[codeKey]*codeLayout
	metrics map[metricKey]fontMetrics
	scrolls map[string]*hscroll
	// unwrapped are code blocks whose wrapping was turned off; full the
	// long user messages shown whole; opened the <details> blocks opened or
	// closed against how they were written; menu the open row menu.
	unwrapped map[string]bool
	full      map[string]bool
	opened    map[string]bool
	menu      string
	reveals   map[string]revealState
	plain     map[plainKey]*inlineText
	previews  map[string]string // thoughts' one-line previews
	menuClose gesture.Click     // the open menu's outside
	// tails are the streaming messages' last blocks as parsed, by owner,
	// and blockSeq numbers the code blocks and tables of the message
	// being drawn.
	tails    map[string]mdKey
	blockSeq int
	// canvasH is the canvas's height, which caps an open group's list
	// (min(18rem, 50dvh)).
	canvasH int
}

// follow is one thread's list and T3's TimelineFollow over it
// (timelineViewport.ts:249-283): following keeps the end in view; the
// reader scrolling away releases it, and coming back within 40dp of the
// end after leaving follows again.
type follow struct {
	list     layout.List
	released bool
	atEnd    bool      // released, and still within 40dp of the end
	leftAt   time.Time // when it left the end, for the pill's 150ms
	first    string    // the row at the top, kept there as rows come and go above it
	heights  map[string]int
	// overflows: the rows are taller than the canvas, so following pins
	// the list's end; until then they start at the top.
	overflows bool
	// reveal is a row a disclosure just opened, to bring into view above
	// the composer once it's laid out, from the top of the row before it
	// when fromPrev; for revealFrames more frames at most.
	reveal       string
	fromPrev     bool
	revealFrames int
	// revealed is the Timeline.Reveal last brought into view.
	revealed string
}

// endBand is how close to the end counts as the end (MessagesTimeline.logic.ts:324-336).
const endBand = 40

// faded is Tailwind's opacity modifier, as in border/60: c at a/255 of its
// own alpha. alpha sets the alpha outright, so dark mode's translucent
// tokens (--border is white at 8%) would come out bright.
func faded(c color.NRGBA, a uint8) color.NRGBA {
	c.A = uint8((int(c.A)*int(a) + 127) / 255)
	return c
}

func newTimelineState() timelineState {
	return timelineState{follows: map[string]*follow{}, hovers: map[string]*gesture.Hover{}, scrolls: map[string]*hscroll{},
		unwrapped: map[string]bool{}, full: map[string]bool{}}
}

func (th *Theme) followFor(id string) *follow {
	tl := &th.tl
	f := tl.follows[id]
	for i, o := range tl.order {
		if o == id {
			tl.order = append(tl.order[:i:i], tl.order[i+1:]...)
			break
		}
	}
	tl.order = append(tl.order, id)
	if f == nil {
		f = &follow{list: layout.List{Axis: layout.Vertical}, heights: map[string]int{}}
		tl.follows[id] = f
		// T3 remembers the last 100 threads' positions.
		if len(tl.order) > 100 {
			delete(tl.follows, tl.order[0])
			tl.order = tl.order[1:]
		}
	}
	return f
}

// timeline draws s's conversation into the canvas: rows in the lane laneX
// px in and laneW wide, the end clearing composer px at the bottom.
func (th *Theme) timeline(gtx C, s *Scene, laneX, laneW, composer int) {
	size := gtx.Constraints.Max
	tl := &s.Timeline
	if tl.Loading {
		return
	}
	if len(tl.Rows) == 0 && !tl.MoreHistory && s.Empty != "" {
		// "Send a message to start the conversation.", 14/20 muted in the
		// middle of what the composer leaves.
		m := op.Record(gtx.Ops)
		d := th.textLine(within(gtx, laneW), Txt{Size: 14, Color: P.MutedForeground}, 20, s.Empty)
		call := m.Stop()
		defer op.Offset(image.Pt(laneX+(laneW-d.Size.X)/2, (size.Y-composer-d.Size.Y)/2)).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
		return
	}
	f := th.followFor(tl.ID)
	l := &f.list
	n := len(tl.Rows) + 2
	ids := func(i int) string {
		switch {
		case i == 0:
			return "header"
		case i <= len(tl.Rows):
			return tl.Rows[i-1].ID
		}
		return "footer"
	}
	if tl.Reveal != f.revealed {
		f.revealed = tl.Reveal
		for i := range tl.Rows {
			if tl.Rows[i].ID == tl.Reveal {
				// The row comes in 24dp under the top, and the end is let go
				// (timelineScrollAnchoring.ts's anchor offset).
				f.released, f.atEnd, f.leftAt = true, false, gtx.Now
				l.Position = layout.Position{First: i + 1, Offset: -dp(gtx, 24), BeforeEnd: true}
				f.first = tl.Rows[i].ID
			}
		}
	}
	if f.released && f.first != "" && ids(l.Position.First) != f.first {
		for i := 0; i < n; i++ {
			if ids(i) == f.first {
				l.Position.First = i
				break
			}
		}
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	g := gtx
	g.Constraints = layout.Exact(size)
	lay := func() op.CallOp {
		m := op.Record(gtx.Ops)
		l.Layout(g, n, func(gtx C, i int) D {
			var d D
			switch {
			case i == 0:
				d = th.timelineHeader(gtx, tl, laneX, laneW)
			case i <= len(tl.Rows):
				d = th.timelineRow(gtx, s, &tl.Rows[i-1], laneX, laneW)
			default:
				d = th.timelineFooter(gtx, s, laneX, laneW, composer)
			}
			f.heights[ids(i)] = d.Size.Y
			return d
		})
		return m.Stop()
	}
	// Following pins the end once the rows overflow the canvas (the list's
	// own ScrollToEnd, which still takes a wheel up); rows that fit start
	// at the top. When that changed, lay out again so the frame is right.
	l.ScrollToEnd = !f.released && f.overflows
	call := lay()
	p := l.Position
	if fits := p.First == 0 && p.Offset <= 0 && p.OffsetLast >= 0; !f.released && fits == f.overflows {
		f.overflows = !fits
		l.ScrollToEnd, l.Position = f.overflows, layout.Position{}
		call = lay()
	}
	call.Add(gtx.Ops)
	f.first = ids(l.Position.First)
	th.revealDisclosure(gtx, f, n, ids, size.Y-composer)
	th.anchorToggles(gtx, f, tl)

	// Where the end is: within the band counts as there.
	dist := 1 << 30
	if p := l.Position; p.First+p.Count >= n {
		dist = max(0, -p.OffsetLast)
	}
	atEnd := dist <= dp(gtx, endBand)
	switch {
	case !f.released && dist > 0:
		// The reader scrolled up: let go of the end (navigated).
		f.released, f.atEnd, f.leftAt = true, atEnd, gtx.Now
	case f.released && atEnd && !f.atEnd:
		// Back at the end after leaving it: follow again, from the end.
		f.released, l.Position.BeforeEnd = false, false
		gtx.Execute(op.InvalidateCmd{})
	case f.released && !atEnd && f.atEnd:
		f.atEnd, f.leftAt = false, gtx.Now
	}
	th.timelineScrollbar(gtx, f, n, ids, size)
	// "Scroll to end", 150ms after the list has left the end.
	if f.released && !f.atEnd {
		if wait := f.leftAt.Add(150 * time.Millisecond).Sub(gtx.Now); wait > 0 {
			gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(wait)})
		} else {
			th.scrollToEnd(gtx, laneX, laneW, size.Y-composer-dp(gtx, 4))
		}
	}
	if th.clicked("timeline.end") {
		f.released, l.Position.BeforeEnd = false, false
		gtx.Execute(op.InvalidateCmd{})
	}
}

// anchorToggles keeps a row the reader opens or closes where it is: like
// T3's preserveDisclosurePosition it ends following the end, unless the
// turn is working, and the list then stays put by its top row as the row
// grows below. Opening a group or a standalone entry brings what it shows
// into view (MessagesTimeline.tsx:482-594).
func (th *Theme) anchorToggles(gtx C, f *follow, tl *Timeline) {
	for _, a := range th.in.actions {
		verb, arg, _ := strings.Cut(a.ID, ":")
		if a.Kind != "click" || verb != "fold" && verb != "group" && verb != "entry" {
			continue
		}
		if tl.Working && !f.released {
			return
		}
		if !f.released {
			f.released, f.atEnd, f.leftAt = true, true, gtx.Now
		}
		f.reveal = ""
		for _, r := range tl.Rows {
			switch {
			case verb == "group" && r.Group == arg && (r.Kind == RowWorkToggle || r.Kind == RowWorkLive || r.Kind == RowThinking) && !r.Open:
				f.reveal, f.fromPrev, f.revealFrames = arg+":details", true, 10
			case verb == "entry" && r.ID == arg && r.Kind == RowWork && len(r.Entries) == 1 && !r.Entries[0].Open:
				f.reveal, f.fromPrev, f.revealFrames = arg, false, 10
			}
		}
	}
}

// revealDisclosure scrolls a just-opened row's new content into view above
// the composer, bottom px down the canvas, as far as the room above the
// opened row's top allows, 24dp under the canvas's top
// (resolveDisclosureRevealScrollDelta).
func (th *Theme) revealDisclosure(gtx C, f *follow, n int, ids func(int) string, bottom int) {
	if f.reveal == "" {
		return
	}
	if f.revealFrames--; f.revealFrames < 0 {
		f.reveal = ""
		return
	}
	p := f.list.Position
	y := -p.Offset
	prevTop := y
	for i := p.First; i < n && i < p.First+p.Count; i++ {
		h := f.heights[ids(i)]
		if ids(i) == f.reveal {
			top := y
			if f.fromPrev {
				top = prevTop
			}
			hidden, room := y+h-bottom, top-dp(gtx, 24)
			if d := min(hidden, room); d > 0 {
				f.list.Position.Offset += d
				gtx.Execute(op.InvalidateCmd{})
			}
			f.reveal = ""
			return
		}
		prevTop = y
		y += h
	}
}

// timelineScrollbar is the scroller's 6dp bar at the canvas's right edge
// (01-tokens port note 12), when the rows overflow.
func (th *Theme) timelineScrollbar(gtx C, f *follow, n int, ids func(int) string, size image.Point) {
	total, before := 0, 0
	for i := 0; i < n; i++ {
		h, ok := f.heights[ids(i)]
		if !ok {
			h = dp(gtx, 90) // T3's estimate for a row not laid out yet
		}
		total += h
		if i < f.list.Position.First {
			before += h
		}
	}
	if total <= size.Y {
		return
	}
	top := before + f.list.Position.Offset
	length := max(dp(gtx, 16), size.Y*size.Y/total)
	y := (size.Y - length) * min(top, total-size.Y) / max(1, total-size.Y)
	w := dp(gtx, Metrics.AppScrollbarWidth)
	fill(gtx, over(P.AppScrollbarThumb, P.Background), image.Rect(size.X-w, y, size.X, y+length), w/2)
}

// scrollToEnd is T3's ScrollToEndButton: a glass xs pill centred on the
// lane, its bottom bottom px down the canvas (TimelineOverlays.tsx).
func (th *Theme) scrollToEnd(gtx C, laneX, laneW, bottom int) {
	id := "timeline.end"
	c, _ := th.events(gtx, id)
	m := op.Record(gtx.Ops)
	cg := gtx
	cg.Constraints = layout.Constraints{Max: image.Pt(laneW, dp(gtx, 24))}
	content := layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceEnd}.Layout(cg,
		layout.Rigid(func(gtx C) D {
			return pad(0, -2, 0, -2).Layout(gtx, LucideW("chevron-down", P.ContrastMutedForeground, 14))
		}),
		layout.Rigid(Spacer(4, 0)),
		layout.Rigid(func(gtx C) D {
			return th.textLine(gtx, Txt{Size: 12, Font: UI(font.Medium), Color: P.Foreground}, 16, "Scroll to end")
		}),
	)
	call := m.Stop()
	h := dp(gtx, 24)
	w := content.Size.X + 2*dp(gtx, 7) + 2*max(1, dp(gtx, 1))
	at := image.Pt(laneX+(laneW-w)/2, bottom-dp(gtx, 6)-h)
	defer op.Offset(at).Push(gtx.Ops).Pop()
	r := image.Rect(0, 0, w, h)
	glass, border := faded(P.Background, 0xcc), faded(P.Border, 0x99)
	if c.Hovered() {
		border = P.Border
	}
	shadow(gtx, r, h/2, []boxShadow{{0, 1, 3, 0, alpha(rgb(0), 0x1a)}, {0, 1, 2, -1, alpha(rgb(0), 0x1a)}})
	fill(gtx, over(glass, P.Background), r, h/2)
	strokeRect(gtx, over(border, P.Background), r, h/2, float32(max(1, dp(gtx, 1))))
	off := op.Offset(image.Pt((w-content.Size.X)/2, (h-content.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	th.hit(gtx, id, r.Max)
}

// timelineHeader is the list's head: the 24dp under the top fade, then
// "Load earlier turns" when there's more history.
func (th *Theme) timelineHeader(gtx C, tl *Timeline, laneX, laneW int) D {
	w := gtx.Constraints.Max.X
	h := dp(gtx, Metrics.WorkspaceTitlebarScrollFadeHeight)
	if !tl.MoreHistory && tl.HistoryError == "" {
		return D{Size: image.Pt(w, h)}
	}
	defer op.Offset(image.Pt(laneX, h)).Push(gtx.Ops).Pop()
	y := 0
	if tl.MoreHistory {
		// A 28dp button across the lane: rules either side of the label.
		id := "more"
		c, _ := th.events(gtx, id)
		label, col := "Load earlier turns", P.MutedForeground
		if tl.LoadingHistory {
			label = "Loading earlier turns…"
		} else if c.Hovered() {
			col = P.Foreground
		}
		bh := dp(gtx, 28)
		m := op.Record(gtx.Ops)
		d := th.textLine(gtx, Txt{Size: 12, Color: col}, 16, label)
		call := m.Stop()
		lx := (laneW - d.Size.X) / 2
		rule := over(faded(P.Border, 0xb3), P.Background)
		hline(gtx, rule, bh/2, 0, lx-dp(gtx, 12))
		hline(gtx, rule, bh/2, lx+d.Size.X+dp(gtx, 12), laneW)
		lo := op.Offset(image.Pt(lx, (bh-d.Size.Y)/2)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		lo.Pop()
		if !tl.LoadingHistory {
			th.hit(gtx, id, image.Pt(laneW, bh))
		}
		y = bh
	}
	if tl.HistoryError != "" {
		if y > 0 {
			y += dp(gtx, 6)
		}
		m := op.Record(gtx.Ops)
		d := th.textLine(within(gtx, laneW), Txt{Size: 12, Color: P.MutedForeground}, 16, tl.HistoryError)
		call := m.Stop()
		eo := op.Offset(image.Pt((laneW-d.Size.X)/2, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		eo.Pop()
		y += d.Size.Y
	}
	return D{Size: image.Pt(w, h+y+dp(gtx, 8))}
}

// timelineFooter ends the list: approvals no row shows, then space as tall
// as the composer over it, then 16dp (TimelineListFooter).
func (th *Theme) timelineFooter(gtx C, s *Scene, laneX, laneW, composer int) D {
	y := 0
	for i := range s.Timeline.Approvals {
		off := op.Offset(image.Pt(laneX, y)).Push(gtx.Ops)
		y += th.workBlock(within(gtx, laneW), 8, func(gtx C) D { return th.approvalRow(gtx, &s.Timeline.Approvals[i]) })
		off.Pop()
	}
	return D{Size: image.Pt(gtx.Constraints.Max.X, y+composer+dp(gtx, 16))}
}

// rowPadding is T3's bottom padding by row kind (MessagesTimeline.tsx:
// 1589-1674): work-log rows bring their own.
func rowPadding(r *TimelineRow) float32 {
	switch r.Kind {
	case RowWork, RowWorkLive, RowWorkToggle, RowThinking:
		return 0
	case RowTurnFold, RowWorking:
		return 6
	case RowUser:
		if r.Message.GroupedNext {
			return 4
		}
	case RowAssistant:
		if !r.Message.Meta {
			return 8
		}
	case RowDivider:
		return 8
	}
	return 16
}

// timelineRow draws one row in the lane.
func (th *Theme) timelineRow(gtx C, s *Scene, r *TimelineRow, laneX, laneW int) D {
	w := gtx.Constraints.Max.X
	hot := th.rowHot(gtx, r.ID)
	off := op.Offset(image.Pt(laneX, 0)).Push(gtx.Ops)
	g := within(gtx, laneW)
	var h int
	switch r.Kind {
	case RowUser:
		h = th.userRow(g, r, hot, s.Timeline.Root)
	case RowAssistant, RowAssistantMeta:
		h = th.assistantRow(g, r, hot, s.Timeline.CanRun, s.Timeline.Root)
	case RowTurnFold:
		h = th.turnFold(g, r, hot)
	case RowWork, RowWorkToggle, RowWorkLive, RowThinking:
		h = th.workRow(g, r, hot)
	case RowWorking:
		h = th.workingRow(g, r)
	case RowCompaction:
		h = th.compactionRow(g, r)
	case RowDivider:
		h = th.dividerRow(g, r)
	case RowPlan:
		h = th.planCard(g, r)
	}
	h += dp(gtx, rowPadding(r))
	th.rowHotArea(gtx, r.ID, image.Pt(laneW, h))
	off.Pop()
	return D{Size: image.Pt(w, h)}
}

// rowHot reports whether the pointer is over row id: its timestamps and
// actions show while it is. rowHotArea watches the row's box for it.
func (th *Theme) rowHot(gtx C, id string) bool {
	h := th.tl.hovers[id]
	if h == nil {
		if len(th.tl.hovers) > 5000 {
			th.tl.hovers = map[string]*gesture.Hover{}
		}
		h = new(gesture.Hover)
		th.tl.hovers[id] = h
	}
	return h.Update(gtx.Source)
}

// rowHotArea watches a row's box for the pointer without taking any of its
// events.
func (th *Theme) rowHotArea(gtx C, id string, size image.Point) {
	h := th.tl.hovers[id]
	if h == nil {
		return
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	h.Add(gtx.Ops)
}

// timestamp is TimelineRowTimestamp: the row's time, 12/16 muted, taking
// space at the row's end only while the row is hovered, its long form in a
// tooltip. id names the row.
func (th *Theme) timestamp(id string, at time.Time, show bool) W {
	return func(gtx C) D {
		if !show || at.IsZero() {
			return D{}
		}
		return pad(0, 4, 0, 0).Layout(gtx, th.stampText(id, at, 16))
	}
}

// stampText is a timestamp's text on a lineH dp line, 12dp muted, with
// T3's tooltip over it: "4:44 PM, 17th November 2026"
// (formatChatTimestampTooltip).
func (th *Theme) stampText(id string, at time.Time, lineH float32) W {
	return func(gtx C) D {
		d := th.textLine(gtx, Txt{Size: 12, Color: P.MutedForeground}, lineH, dayAwareTimestamp(at, gtx.Now))
		th.tipArea(gtx, "stamp:"+id, timestampTooltip(at), d.Size)
		return d
	}
}

// tipArea shows tip once the pointer rests on a size-sized area at the
// current offset. The area only watches: its clicks go on to what's
// under it, such as the row that holds it.
func (th *Theme) tipArea(gtx C, id, tip string, size image.Point) {
	if th.in.clicks == nil {
		th.in.clicks = map[string]*gesture.Click{}
	}
	c := th.in.clicks[id]
	if c == nil {
		c = new(gesture.Click)
		th.in.clicks[id] = c
	}
	for {
		if _, ok := c.Update(gtx.Source); !ok {
			break
		}
	}
	func() {
		defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
		defer pointer.PassOp{}.Push(gtx.Ops).Pop()
		c.Add(gtx.Ops)
	}()
	th.tipAt(gtx, id, tip, size, true, 0)
}

// turnFold is "Worked for 2m 14s": a muted 22.75dp button that brightens
// on hover with its chevron, over a rule at border/60
// (MessagesTimeline.tsx:1805-1840).
func (th *Theme) turnFold(gtx C, r *TimelineRow, hot bool) int {
	w := gtx.Constraints.Max.X
	id := "fold:" + r.Group
	c, _ := th.events(gtx, id)
	col := P.MutedForeground
	if c.Hovered() {
		col = P.Foreground
	}
	top, line := dp(gtx, 4), dp(gtx, 22.75)
	icon := "chevron-right"
	if r.Open {
		icon = "chevron-down"
	}
	stamp := th.recorded(gtx, th.timestamp(r.ID, r.At, hot))
	m := op.Record(gtx.Ops)
	d := layout.Flex{Alignment: layout.Middle}.Layout(within(gtx, w-stamp.Size.X-dp(gtx, 12)),
		layout.Rigid(Spacer(4, 0)),
		layout.Rigid(func(gtx C) D { return th.textLine(gtx, Txt{Size: 14, Color: col}, 22.75, r.Label) }),
		layout.Rigid(Spacer(4, 0)),
		layout.Rigid(LucideW(icon, col, 14)),
		layout.Rigid(Spacer(4, 0)),
	)
	call := m.Stop()
	bo := op.Offset(image.Pt(0, top)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	th.hit(gtx, id, image.Pt(d.Size.X, line))
	so := op.Offset(image.Pt(w-dp(gtx, 2)-stamp.Size.X, (line-stamp.Size.Y)/2)).Push(gtx.Ops)
	stamp.call.Add(gtx.Ops)
	so.Pop()
	bo.Pop()
	y := top + line + dp(gtx, 8)
	hline(gtx, over(faded(P.Border, 0x99), P.Background), y, 0, w)
	return y + max(1, dp(gtx, 1))
}

// recording is a widget drawn ahead of where it goes.
type recording struct {
	call op.CallOp
	Size image.Point
}

func (th *Theme) recorded(gtx C, w W) recording {
	m := op.Record(gtx.Ops)
	g := gtx
	g.Constraints.Min = image.Point{}
	d := w(g)
	return recording{m.Stop(), d.Size}
}

// workingRow is "Working for 12s", the live turn's header: muted 14dp on a
// 24dp line, the timer ticking each second, over a rule at border/60
// (TimelineWorkGroupRows.tsx:291-330). A compaction that holds the turn
// back says "Compacting for 4s" after a minimize-2, shimmering.
func (th *Theme) workingRow(gtx C, r *TimelineRow) int {
	w := gtx.Constraints.Max.X
	top, line := dp(gtx, 4), dp(gtx, 24)
	label := "Working..."
	if r.Compacting {
		label = "Compacting…"
	}
	if !r.Since.IsZero() {
		label = "Working for " + workingTimer(r.Since, gtx.Now)
		if r.Compacting {
			label = "Compacting for " + workingTimer(r.Since, gtx.Now)
		}
		next := r.Since.Add(gtx.Now.Sub(r.Since).Truncate(time.Second) + time.Second)
		gtx.Execute(op.InvalidateCmd{At: next})
	}
	draw := func(col color.NRGBA) W {
		return func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					if !r.Compacting {
						return D{}
					}
					return pad(0, 6, 0, 0).Layout(gtx, LucideW("minimize-2", col, 12))
				}),
				layout.Rigid(func(gtx C) D {
					return th.textLine(gtx, Txt{Size: 14, Color: col}, 22.75, label)
				}),
			)
		}
	}
	off := op.Offset(image.Pt(dp(gtx, 4), top+(line-dp(gtx, 22.75))/2)).Push(gtx.Ops)
	d := draw(P.MutedForeground)(within(gtx, w))
	if r.Compacting {
		th.shimmer(gtx, d.Size, draw(P.Foreground))
	}
	off.Pop()
	y := top + line + dp(gtx, 8)
	hline(gtx, over(faded(P.Border, 0x99), P.Background), y, 0, w)
	return y + max(1, dp(gtx, 1))
}

// shimmerBand is the live-activity-focus mask's stops: 72dp of foreground
// fading in and out (index.css:586-660).
var shimmerBand = [][2]float32{{0, 0}, {10.8, 0.12}, {25.2, 0.55}, {36, 1}, {46.8, 0.55}, {61.2, 0.12}, {72, 0}}

// animStep is the step of a 2.2s steps(30) animation, from 0 to 29, and
// when the next one is due.
func animStep(now time.Time) (int, time.Time) {
	const period, steps = 2200 * time.Millisecond, 30
	t := now.UnixNano() % int64(period)
	step := int(t * steps / int64(period))
	next := now.Add(time.Duration(int64(step+1)*int64(period)/steps - t))
	return step, next
}

// shimmer sweeps the band across a size box drawn by w, w drawing the
// highlighted copy: T3's ActivityShimmerOverlay, left to right in 30
// steps every 2.2s.
func (th *Theme) shimmer(gtx C, size image.Point, w W) {
	step, next := animStep(gtx.Now)
	gtx.Execute(op.InvalidateCmd{At: next})
	band := float32(dp(gtx, 72))
	left := -band + (float32(size.X)+2*band)*float32(step)/30
	th.banded(gtx, size, left, shimmerBand, w)
}

// banded draws w through a soft vertical band whose left edge is at left
// px: its alpha follows stops (dp from the band's edge, alpha), in 3dp
// slices.
func (th *Theme) banded(gtx C, size image.Point, left float32, stops [][2]float32, w W) {
	slice := float32(dp(gtx, 3))
	end := stops[len(stops)-1][0] * gtx.Metric.PxPerDp
	g := gtx
	g.Constraints = layout.Exact(size)
	for x := float32(0); x < end; x += slice {
		a := bandAlpha(stops, (x+slice/2)/gtx.Metric.PxPerDp)
		x0, x1 := int(left+x), int(left+x+slice+0.5)
		if a <= 0 || x1 <= 0 || x0 >= size.X {
			continue
		}
		cl := clip.Rect{Min: image.Pt(max(0, x0), 0), Max: image.Pt(min(size.X, x1), size.Y)}.Push(gtx.Ops)
		fade := paint.PushOpacity(gtx.Ops, a)
		w(g)
		fade.Pop()
		cl.Pop()
	}
}

// bandAlpha interpolates the band's alpha at x dp.
func bandAlpha(stops [][2]float32, x float32) float32 {
	for i := 1; i < len(stops); i++ {
		if x <= stops[i][0] {
			a, b := stops[i-1], stops[i]
			return a[1] + (b[1]-a[1])*(x-a[0])/(b[0]-a[0])
		}
	}
	return 0
}

// shine is live-tool-shine (index.css:551-578): the label in the
// secondary label colour with a 72dp foreground gleam crossing it, from
// 72dp before it to its end, in 30 steps every 2.2s.
func (th *Theme) shine(gtx C, size image.Point, w func(col color.NRGBA) W) {
	step, next := animStep(gtx.Now)
	gtx.Execute(op.InvalidateCmd{At: next})
	band := float32(dp(gtx, 72))
	left := -band + (float32(size.X)+band)*float32(step)/30
	th.banded(gtx, size, left, [][2]float32{{0, 0}, {36, 1}, {72, 0}}, w(P.ContrastForeground))
}

// compactionRow is the compaction divider: "Context compacted 180K → 12K
// tokens" between rules at border/70, 12/16 muted (destructive when it
// failed), shimmering while it runs (MessagesTimeline.tsx:1698-1748).
func (th *Theme) compactionRow(gtx C, r *TimelineRow) int {
	w := gtx.Constraints.Max.X
	h := dp(gtx, 24)
	col, rule := P.MutedForeground, over(faded(P.Border, 0xb3), P.Background)
	if r.Failed {
		col, rule = P.Destructive, over(faded(P.Destructive, 0x4d), P.Background)
	}
	draw := func(col color.NRGBA) W {
		return func(gtx C) D {
			icon := "minimize-2"
			if r.Failed {
				icon = "circle-alert"
			}
			kids := []layout.FlexChild{layout.Rigid(LucideW(icon, col, 12)), layout.Rigid(Spacer(6, 0))}
			text := func(s string) layout.FlexChild {
				return layout.Rigid(func(gtx C) D { return th.textLine(gtx, Txt{Size: 12, Color: col}, 16, s) })
			}
			if r.Before != "" {
				kids = append(kids, text(r.Label+" "+r.Before), layout.Rigid(Spacer(6, 0)), layout.Rigid(LucideW("arrow-right", col, 12)),
					layout.Rigid(Spacer(6, 0)), text(r.After+" tokens"))
			} else {
				kids = append(kids, text(r.Label))
			}
			if r.Detail != "" {
				kids = append(kids, layout.Rigid(Spacer(6, 0)), layout.Rigid(func(gtx C) D {
					return th.textLine(gtx, Txt{Size: 12, Color: over(alpha(col, 0xcc), P.Background)}, 16, "· "+r.Detail)
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
		}
	}
	content := th.recorded(within(gtx, w-dp(gtx, 2*24)), draw(col))
	x := (w - content.Size.X) / 2
	hline(gtx, rule, h/2, 0, x-dp(gtx, 12))
	hline(gtx, rule, h/2, x+content.Size.X+dp(gtx, 12), w)
	off := op.Offset(image.Pt(x, (h-content.Size.Y)/2)).Push(gtx.Ops)
	content.call.Add(gtx.Ops)
	if r.Live {
		th.shimmer(gtx, content.Size, draw(P.Foreground))
	}
	off.Pop()
	return h
}

// dividerRow is TimelineSystemDivider: an 11/16 muted pill, padded 8 by
// 4, its label medium after a 12dp icon and its detail at 70%, between
// rules at border/70 8dp from it. A detail that wraps does so under the
// label, each line centred; an action makes the pill a bordered button,
// padded 10, that opens a thread (TimelineSystemDivider.tsx:7-85).
func (th *Theme) dividerRow(gtx C, r *TimelineRow) int {
	w := gtx.Constraints.Max.X
	col := P.MutedForeground
	id := "thread:" + r.Thread
	hovered := false
	if r.Action != "" {
		c, _ := th.events(gtx, id)
		if hovered = c.Hovered(); hovered {
			col = P.Foreground
		}
	}
	padX, bw := dp(gtx, 8), 0
	if r.Action != "" {
		padX, bw = dp(gtx, 10), max(1, dp(gtx, 1))
	}
	gap := dp(gtx, 8)
	avail := w - 2*gap - 2*padX - 2*bw
	var content recording
	if r.Wrap {
		t := th.dividerText(r, col)
		nw := th.layoutInline(gtx, t, avail).size.X
		content = th.recorded(within(gtx, nw), func(gtx C) D { return th.inline(gtx, t, nil) })
	} else {
		content = th.recorded(within(gtx, avail), func(gtx C) D {
			kids := []layout.FlexChild{}
			if r.Icon != "" {
				kids = append(kids, layout.Rigid(LucideW(r.Icon, col, 12)), layout.Rigid(Spacer(6, 0)))
			}
			kids = append(kids, layout.Rigid(func(gtx C) D {
				return th.textLine(gtx, Txt{Size: 11, Font: UI(font.Medium), Color: col}, 16, r.Label)
			}))
			if r.Detail != "" {
				kids = append(kids, layout.Rigid(Spacer(6, 0)), layout.Flexed(1, func(gtx C) D {
					return th.textLine(gtx, Txt{Size: 11, Color: over(faded(col, 0xb3), P.Background)}, 16, "· "+r.Detail)
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
		})
	}
	pill := image.Pt(content.Size.X+2*padX+2*bw, content.Size.Y+2*dp(gtx, 4)+2*bw)
	h := pill.Y + 2*dp(gtx, 8)
	x, y := (w-pill.X)/2, dp(gtx, 8)
	rule := over(faded(P.Border, 0xb3), P.Background)
	hline(gtx, rule, h/2, 0, x-gap)
	hline(gtx, rule, h/2, x+pill.X+gap, w)
	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	if r.Action != "" {
		box := image.Rectangle{Max: pill}
		bg := P.Background
		if hovered {
			bg = over(P.Accent, P.Background)
		}
		fill(gtx, bg, box, pill.Y/2)
		strokeRect(gtx, over(faded(P.Border, 0xb3), P.Background), box, pill.Y/2, float32(bw))
	}
	co := op.Offset(image.Pt(bw+padX, bw+dp(gtx, 4))).Push(gtx.Ops)
	content.call.Add(gtx.Ops)
	co.Pop()
	if r.Action != "" {
		th.hit(gtx, id, pill)
		th.tipAt(gtx, id, r.Action, pill, true, 0)
	}
	return h
}

// dividerText is a divider whose detail wraps, as one centred paragraph:
// the icon and the medium label, 6dp, then "· " and the detail at 70%.
func (th *Theme) dividerText(r *TimelineRow, col color.NRGBA) *inlineText {
	p := &inlineText{font: UI(font.Normal), size: 11, lineH: 16, align: textCenter}
	if r.Icon != "" {
		p.runs = append(p.runs, inlineRun{icon: r.Icon, size: 12, color: col}, inlineRun{gap: 6})
	}
	p.runs = append(p.runs, inlineRun{text: r.Label, font: UI(font.Medium), size: 11, color: col}, inlineRun{gap: 6})
	detail := over(faded(col, 0xb3), P.Background)
	for i, line := range strings.Split("·\u00a0"+r.Detail, "\n") {
		if i > 0 {
			p.runs = append(p.runs, inlineRun{brk: true})
		}
		if line != "" {
			p.runs = append(p.runs, inlineRun{text: line, font: UI(font.Normal), size: 11, color: detail})
		}
	}
	return th.keepText("divider\x00"+r.Icon+"\x00"+r.Label+"\x00"+r.Detail, Txt{Size: 11, Color: col}, p)
}

// planCard is ProposedPlanCard: a card at 70% with a 1dp border at 80%,
// radius 16, padded 20, its "Plan" badge and title over the plan's
// markdown, clipped to 416dp under a fade until expanded.
func (th *Theme) planCard(gtx C, r *TimelineRow) int {
	w := gtx.Constraints.Max.X
	surface := over(faded(P.Card, 0xb3), P.Background)
	p, rad := dp(gtx, 20), dp(gtx, 16)
	title := planTitle(r.Plan)
	body := stripPlanTitle(r.Plan)
	collapsible := len(r.Plan) > 900 || strings.Count(r.Plan, "\n")+1 > 20
	id := r.ID
	open := th.tl.full[id]
	st := mdStyle{color: over(faded(P.Foreground, 0xe6), surface), on: surface, owner: r.ID}
	m := op.Record(gtx.Ops)
	inner := w - 2*p
	// The header: the badge, the title, the actions menu.
	badge := th.recorded(gtx, th.T3Badge(BadgeSecondary, "", "Plan", surface))
	menu := Button{Variant: BtnOutline, Size: BtnSizeIconXs, Icon: "ellipsis", IconSize: 16, On: surface}
	hh := dp(gtx, 24)
	bo := op.Offset(image.Pt(0, (hh-badge.Size.Y)/2)).Push(gtx.Ops)
	badge.call.Add(gtx.Ops)
	bo.Pop()
	to := op.Offset(image.Pt(badge.Size.X+dp(gtx, 8), (hh-dp(gtx, 20))/2)).Push(gtx.Ops)
	th.textLine(within(gtx, inner-badge.Size.X-dp(gtx, 8+12+24)), Txt{Size: 14, Font: UI(font.Medium), Color: P.Foreground}, 20, title)
	to.Pop()
	mo := op.Offset(image.Pt(inner-dp(gtx, 24), 0)).Push(gtx.Ops)
	th.T3Button(id+":menu", menu)(gtx)
	mo.Pop()
	if th.clicked(id + ":menu") {
		th.toggleMenu(id + ":menu")
	}
	// The menu: copy, or download to a file, the plan as T3 exports it.
	// T3's "Save to workspace" isn't here.
	copyLabel := "Copy to clipboard"
	if th.copiedFor(gtx, "copy:"+id, 2000) {
		copyLabel = "Copied!"
	}
	switch th.rowMenu(gtx, id+":menu", image.Pt(inner, hh+dp(gtx, 4)), false, []menuChoice{{label: copyLabel}, {label: "Download as markdown"}}) {
	case 0:
		th.copyNow(gtx, "copy:"+id, planExport(r.Plan))
	case 1:
		th.push(Action{ID: "plan.download:" + planFile(r.Plan), Kind: "click", Text: planExport(r.Plan)})
	}
	y := hh + dp(gtx, 16)
	src := body
	if collapsible && !open {
		src = planPreview(r.Plan)
	}
	bm := op.Record(gtx.Ops)
	bh := th.markdown(within(gtx, inner), src, st, nil).Size.Y
	bcall := bm.Stop()
	shown := bh
	if collapsible && !open {
		shown = min(bh, dp(gtx, 416))
	}
	func() {
		defer op.Offset(image.Pt(0, y)).Push(gtx.Ops).Pop()
		defer clip.Rect{Max: image.Pt(inner, shown)}.Push(gtx.Ops).Pop()
		bcall.Add(gtx.Ops)
		if collapsible && !open {
			// The 96dp fade into the card: card/95 through card/80, over
			// the card's own card/70.
			fadeH := dp(gtx, 96)
			fadeUp(gtx, image.Rect(0, shown-fadeH, inner, shown), P.Card)
		}
	}()
	y += shown
	if collapsible {
		y += dp(gtx, 16)
		label := "Expand plan"
		if open {
			label = "Collapse plan"
		}
		b := th.recorded(gtx, th.T3Button(id+":expand", Button{Variant: BtnOutline, Size: BtnSizeSm, Label: label, On: surface}))
		eo := op.Offset(image.Pt((inner-b.Size.X)/2, y)).Push(gtx.Ops)
		b.call.Add(gtx.Ops)
		eo.Pop()
		y += b.Size.Y
		if th.clicked(id + ":expand") {
			th.tl.full[id] = !open
		}
	}
	call := m.Stop()
	h := y + 2*p
	rr := image.Rect(0, 0, w, h)
	fill(gtx, surface, rr, rad)
	strokeRect(gtx, over(faded(P.Border, 0xcc), P.Background), rr, rad, float32(max(1, dp(gtx, 1))))
	defer op.Offset(image.Pt(p, p)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
	return h
}

// fadeUp paints col over r from transparent at its top to 95% at its
// bottom, through 80% halfway: bg-linear-to-t from-card/95 via-card/80.
func fadeUp(gtx C, r image.Rectangle, col color.NRGBA) {
	steps := max(1, r.Dy()/max(1, dp(gtx, 4)))
	for i := 0; i < steps; i++ {
		t := (float32(i) + 0.5) / float32(steps) // 0 at the top
		a := float32(0.8) * t * 2
		if t > 0.5 {
			a = 0.8 + 0.15*(t-0.5)*2
		}
		y0 := r.Min.Y + r.Dy()*i/steps
		y1 := r.Min.Y + r.Dy()*(i+1)/steps
		fill(gtx, alpha(col, uint8(a*255+0.5)), image.Rect(r.Min.X, y0, r.Max.X, y1), 0)
	}
}

// planTitle is the card's title: the plan's heading, or "Proposed plan".
func planTitle(md string) string {
	if t := planHeadingText(md); t != "" {
		return t
	}
	return "Proposed plan"
}

// planHeadingText is T3's proposedPlanTitle: the plan's first heading, one
// to six hashes and white space before its words, or "".
func planHeadingText(md string) string {
	if h := planHeading.FindStringSubmatch(md); h != nil {
		return strings.TrimSpace(h[1])
	}
	return ""
}

// planFile is T3's buildProposedPlanMarkdownFilename: the heading in
// lower-case words joined by dashes, "plan" without one, and ".md".
func planFile(md string) string {
	s := strings.ToLower(planHeadingText(md))
	s = strings.Trim(planNonWord.ReplaceAllString(planMarks.ReplaceAllString(s, ""), "-"), "-")
	if s == "" {
		s = "plan"
	}
	return s + ".md"
}

var (
	planHeading = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+(.+)$`)
	planMarks   = regexp.MustCompile("[`'\".,!?()\\[\\]{}]+")
	planNonWord = regexp.MustCompile(`[^a-z0-9]+`)
)

// planExport is T3's normalizePlanMarkdownForExport: the plan with one
// line break at its end, as copied and downloaded.
func planExport(md string) string { return strings.TrimRightFunc(md, unicode.IsSpace) + "\n" }

// stripPlanTitle is T3's stripDisplayedPlanMarkdown: the plan without the
// heading the card shows, nor a "Summary" heading after it.
func stripPlanTitle(md string) string {
	lines := strings.Split(strings.TrimRight(md, " \n"), "\n")
	heading := func(l string) (string, bool) {
		t := strings.TrimLeft(l, " ")
		if len(l)-len(t) > 3 || !strings.HasPrefix(t, "#") {
			return "", false
		}
		rest := strings.TrimLeft(t, "#")
		if len(t)-len(rest) > 6 || !strings.HasPrefix(rest, " ") {
			return "", false
		}
		return strings.TrimSpace(rest), true
	}
	start := 0
	if len(lines) > 0 {
		if _, ok := heading(lines[0]); ok {
			start = 1
		}
	}
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	if start < len(lines) {
		if h, ok := heading(lines[start]); ok && strings.EqualFold(h, "summary") {
			start++
			for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
				start++
			}
		}
	}
	return strings.Join(lines[start:], "\n")
}

// planPreview is T3's buildCollapsedProposedPlanPreviewMarkdown: the plan's
// first ten visible lines, and "..." when there's more.
func planPreview(md string) string {
	var out []string
	visible, more := 0, false
	for _, line := range strings.Split(strings.TrimRight(stripPlanTitle(md), " \n"), "\n") {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) != "" && visible >= 10 {
			more = true
			break
		}
		out = append(out, line)
		if strings.TrimSpace(line) != "" {
			visible++
		}
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		if t := planHeadingText(md); t != "" {
			return t
		}
		return "Plan preview unavailable."
	}
	if more {
		out = append(out, "", "...")
	}
	return strings.Join(out, "\n")
}
