package ui

import (
	"encoding/json"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The work log (WorkLog.tsx, TimelineWorkEntryRow.tsx,
// TimelineWorkGroupRows.tsx, TimelinePendingApprovalRow.tsx;
// docs/ui/06-timeline.md §2.7, §3.7, §3.8): tool calls, thoughts and
// notices on 24dp lines, each with its 16dp icon (stroke 2) in a 24dp slot
// and its label truncating to one line; groups folded under a summary; the
// running tool shining; an open entry's details under it.

// workSpace is WorkLogBlock's space after a row: none when another
// work-log row follows or under an open group's header, 4dp after an open
// group's list, else 8dp.
func workSpace(r *TimelineRow) float32 {
	switch {
	case r.Continues:
		return 0
	case r.Kind == RowWork && r.Open:
		return 4
	case r.Kind != RowWork && r.Open:
		return 0
	}
	return 8
}

// workBlock draws row and the space dp after it, and returns the height.
func (th *Theme) workBlock(gtx C, space float32, row W) int {
	return row(gtx).Size.Y + dp(gtx, space)
}

// workRow draws one work-log row and the space after it.
func (th *Theme) workRow(gtx C, r *TimelineRow, hot bool) int {
	return th.workBlock(gtx, workSpace(r), func(gtx C) D {
		w := gtx.Constraints.Max.X
		h := 0
		switch r.Kind {
		case RowWork:
			if r.Open {
				h = th.groupList(gtx, r)
				break
			}
			for i := range r.Entries {
				off := op.Offset(image.Pt(0, h)).Push(gtx.Ops)
				h += th.entryRow(gtx, &r.Entries[i], false)
				off.Pop()
			}
		case RowWorkToggle:
			h = th.groupHeader(gtx, r, hot)
		case RowWorkLive:
			h = th.liveRow(gtx, r)
		case RowThinking:
			h = th.thinkingRow(gtx, r)
		}
		return D{Size: image.Pt(w, h)}
	})
}

// mutedIcon is a work-log icon's colour: c at 70% and, in light, at 60%
// brightness (ToolActivityIconView's muted: opacity-70 light:brightness-60),
// on the canvas.
func mutedIcon(c color.NRGBA) color.NRGBA {
	if !dark {
		c.R, c.G, c.B = uint8(float32(c.R)*0.6+0.5), uint8(float32(c.G)*0.6+0.5), uint8(float32(c.B)*0.6+0.5)
	}
	return over(alpha(c, uint8(float32(c.A)*0.7+0.5)), P.Background)
}

// failedTint is text-tool-error-icon/40, a failed tool's icon colour.
func failedTint() color.NRGBA { return faded(P.ToolErrorIcon, 0x66) }

// workIcon draws a work-log icon, 16dp at stroke 2: lucide's, or T3's
// own wordmark and browser glyph.
func workIcon(name string, col color.NRGBA) W {
	switch name {
	case "t3-code":
		return t3Wordmark(col, 16)
	case "browser", "computer":
		return browserGlyph(col, 16)
	case "pull-request":
		// PullRequestGlyph.pullRequest (pullRequestIcons.tsx:14).
		name = "git-pull-request-arrow"
	}
	return lucideStroke(name, col, 16, 2)
}

// t3Wordmark is T3Wordmark: the filled "T3", fitted to a size dp box.
func t3Wordmark(col color.NRGBA, size float32) W {
	return func(gtx C) D {
		s := float32(dp(gtx, size))
		const vx, vy, vw, vh = 15.5309, 37, 94.3941, 56.96
		k := s / vw
		segs := cachedPath("t3-code", t3WordmarkPath)
		var p clip.Path
		p.Begin(gtx.Ops)
		at := func(q f32.Point) f32.Point { return f32.Pt((q.X-vx)*k, (q.Y-vy)*k+(s-vh*k)/2) }
		addSegs(&p, segs, at)
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return D{Size: image.Pt(int(s), int(s))}
	}
}

const t3WordmarkPath = `M33.4509 93V47.56H15.5309V37H64.3309V47.56H46.4109V93H33.4509ZM86.7253 93.96C82.832 93.96 78.9653 93.4533 75.1253 92.44C71.2853 91.3733 68.032 89.88 65.3653 87.96L70.4053 78.04C72.5386 79.5867 75.0186 80.8133 77.8453 81.72C80.672 82.6267 83.5253 83.08 86.4053 83.08C89.6586 83.08 92.2186 82.44 94.0853 81.16C95.952 79.88 96.8853 78.12 96.8853 75.88C96.8853 73.7467 96.0586 72.0667 94.4053 70.84C92.752 69.6133 90.0853 69 86.4053 69H80.4853V60.44L96.0853 42.76L97.5253 47.4H68.1653V37H107.365V45.4L91.8453 63.08L85.2853 59.32H89.0453C95.9253 59.32 101.125 60.8667 104.645 63.96C108.165 67.0533 109.925 71.0267 109.925 75.88C109.925 79.0267 109.099 81.9867 107.445 84.76C105.792 87.48 103.259 89.6933 99.8453 91.4C96.432 93.1067 92.0586 93.96 86.7253 93.96Z`

// browserGlyph is T3's BrowserAppIcon: a window with a pointer, stroked at
// 1.9 in its 24 box, the pointer and the dot filled.
func browserGlyph(col color.NRGBA, size float32) W {
	return func(gtx C) D {
		s := float32(dp(gtx, size))
		k := s / 24
		at := func(q f32.Point) f32.Point { return q.Mul(k) }
		var p clip.Path
		p.Begin(gtx.Ops)
		addSegs(&p, cachedPath("browser", "M8.5 19H7.2C4.4 19 3 17.5 3 14.6V7.4C3 4.5 4.5 3 7.4 3h8.2C18.5 3 20 4.5 20 7.4v2.4M11.2 7.2h4.3"), at)
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: 1.9 * k}.Op())
		var f clip.Path
		f.Begin(gtx.Ops)
		addSegs(&f, cachedPath("browser-pointer", "M12.4 11.4L19.9 14L16.5 15.6L15 19.2Z"), at)
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: f.End()}.Op())
		c, r := at(f32.Pt(7.4, 7.2)), 0.75*k
		paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: image.Pt(int(c.X-r+0.5), int(c.Y-r+0.5)), Max: image.Pt(int(c.X+r+0.5), int(c.Y+r+0.5))}.Op(gtx.Ops))
		return D{Size: image.Pt(int(s), int(s))}
	}
}

// workLine is WorkLogLine: the icon slot, the label after 6dp and the
// trailing marks after 6dp more, on a line at least 24dp tall. The label
// truncates unless wrap; label gets the width left.
func (th *Theme) workLine(gtx C, icon W, label func(gtx C) D, trailing []W, wrap bool) int {
	w := gtx.Constraints.Max.X
	line := dp(gtx, 24)
	gap := dp(gtx, 6)
	var tr []recording
	tw := 0
	for _, t := range trailing {
		if t == nil {
			continue
		}
		if rec := th.recorded(gtx, t); rec.Size.X > 0 {
			tr = append(tr, rec)
			tw += rec.Size.X + gap
		}
	}
	lx := 0
	if icon != nil {
		io := op.Offset(image.Pt(dp(gtx, 4), dp(gtx, 4))).Push(gtx.Ops)
		icon(gtx)
		io.Pop()
		lx = dp(gtx, 24) + gap
	}
	// The label line sits 1.09dp under the icon slot's top (the slot's
	// vertical-align: calc(0.5cap - 0.75rem)).
	lo := op.Offset(image.Pt(lx, dp(gtx, 1.09))).Push(gtx.Ops)
	ld := label(within(gtx, w-lx-tw))
	lo.Pop()
	h := max(line, ld.Size.Y+dp(gtx, 1.09))
	x := w
	for i := len(tr) - 1; i >= 0; i-- {
		x -= tr[i].Size.X
		y := (line - tr[i].Size.Y) / 2
		if !wrap {
			y = (h - tr[i].Size.Y) / 2
		}
		off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		tr[i].call.Add(gtx.Ops)
		off.Pop()
		x -= gap
	}
	return h
}

// headingColor is an entry's label colour (TimelineWorkEntryRow.tsx:420-426).
func headingColor(t EntryTone) (color.NRGBA, font.Weight) {
	switch t {
	case ToneWarning:
		return P.Warning, font.Medium
	case ToneDanger:
		return P.Destructive, font.Medium
	case ToneNotice:
		return over(faded(P.Foreground, 0xcc), P.Background), font.Normal
	}
	return P.SecondaryLabel, font.Normal
}

// entryIconColor is an entry's icon colour before the muted treatment.
func entryIconColor(e *WorkEntry) color.NRGBA {
	switch {
	case e.Tone == ToneWarning:
		return P.Warning
	case e.Tone == ToneDanger:
		return P.Destructive
	case e.Failed:
		return failedTint()
	case e.Tone == ToneNotice && e.Icon == "check":
		return P.MutedForeground
	}
	return P.IconMuted
}

// chevron is the 12dp chevron-right in its 16dp slot, at icon-muted 70%,
// turning to point down over 200ms as the entry opens.
func (th *Theme) chevron(id string, open bool) W {
	return func(gtx C) D {
		s := dp(gtx, 16)
		t := th.reveal(gtx, id+":chevron", open)
		c := f32.Pt(float32(s)/2, float32(s)/2)
		defer op.Affine(f32.AffineId().Rotate(c, t*math.Pi/2)).Push(gtx.Ops).Pop()
		off := op.Offset(image.Pt((s-dp(gtx, 12))/2, (s-dp(gtx, 12))/2)).Push(gtx.Ops)
		LucideW("chevron-right", over(faded(P.IconMuted, 0xb3), P.Background), 12)(gtx)
		off.Pop()
		return D{Size: image.Pt(s, s)}
	}
}

// entryRow is one entry, WorkLogRow: its line, 2dp above and below
// unless in an open group's list, and its details while open. It returns
// its height.
func (th *Theme) entryRow(gtx C, e *WorkEntry, grouped bool) int {
	switch {
	case e.Approval != nil:
		return th.approvalRow(gtx, e.Approval).Size.Y
	case e.Error != nil:
		return th.errorRow(gtx, e)
	}
	w := gtx.Constraints.Max.X
	id := "entry:" + e.ID
	hot := th.rowHot(gtx, id)
	py, px := dp(gtx, 2), dp(gtx, 2)
	if grouped {
		py = 0
	}
	col, weight := headingColor(e.Tone)
	label := func(gtx C) D {
		text := e.Label
		if e.Thought != "" {
			text = th.thoughtPreview(e.Thought, e.Thinking)
			if e.Open {
				text = "Thought"
				if e.Thinking {
					text = "Thinking"
				}
			}
		}
		if e.Answer == "" {
			return th.textLine(gtx, Txt{Size: 14, Font: UI(weight), Color: col}, 22.75, text)
		}
		// A question and its answer share the line, both truncating.
		answerCol := P.MutedForeground
		if !e.Open {
			answerCol = P.Foreground
		}
		q := th.recorded(gtx, func(gtx C) D { return th.textLine(gtx, Txt{Size: 14, Color: col}, 22.75, text) })
		a := th.recorded(gtx, func(gtx C) D { return th.textLine(gtx, Txt{Size: 14, Color: answerCol}, 22.75, e.Answer) })
		avail, gap := gtx.Constraints.Max.X, dp(gtx, 6)
		qw, aw := q.Size.X, a.Size.X
		if qw+gap+aw > avail {
			qw = qw * (avail - gap) / max(1, qw+aw)
			aw = avail - gap - qw
		}
		th.textLine(within(gtx, qw), Txt{Size: 14, Color: col}, 22.75, text)
		off := op.Offset(image.Pt(qw+gap, 0)).Push(gtx.Ops)
		th.textLine(within(gtx, aw), Txt{Size: 14, Color: answerCol}, 22.75, e.Answer)
		off.Pop()
		return D{Size: image.Pt(qw+gap+aw, dp(gtx, 22.75))}
	}
	var trailing []W
	if e.Expands {
		trailing = append(trailing, th.chevron(id, e.Open))
	}
	trailing = append(trailing, th.timestamp(id, e.At, hot))
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	line := th.workLine(within(gtx, w-2*px), workIcon(e.Icon, mutedIcon(entryIconColor(e))), label, trailing, false)
	off.Pop()
	if e.Expands {
		th.hit(gtx, id, image.Pt(w, line+2*py))
	}
	h := py + line
	if e.Open {
		do := op.Offset(image.Pt(px, h)).Push(gtx.Ops)
		h += th.entryDetails(within(gtx, w-2*px), e)
		do.Pop()
	}
	h += py
	th.rowHotArea(gtx, id, image.Pt(w, h))
	return h
}

// entryDetails is an open entry's WorkLogDetails: a thought's trace or a
// question's answers 28dp in, or the tool's inspector panel.
func (th *Theme) entryDetails(gtx C, e *WorkEntry) int {
	w := gtx.Constraints.Max.X
	in := dp(gtx, 28)
	switch {
	case e.Thought != "":
		return th.traceBox(within(gtx, w-in), "entry:"+e.ID, in, []string{e.Thought}, e.Thinking)
	case len(e.Answers) > 0:
		// QuestionAnswerHistory: 8dp under the line, each question over its
		// answer, 12dp further in.
		y := dp(gtx, 8)
		for i, q := range e.Answers {
			if i > 0 {
				y += dp(gtx, 8)
			}
			st := Txt{Size: 14, Color: P.MutedForeground}
			if q.Question != "" {
				off := op.Offset(image.Pt(in, y)).Push(gtx.Ops)
				y += th.plainText(within(gtx, w-in), q.Question, st, 20)
				off.Pop()
			}
			if q.Answer != "" {
				y += dp(gtx, 4)
				off := op.Offset(image.Pt(in+dp(gtx, 12), y)).Push(gtx.Ops)
				y += th.plainText(within(gtx, w-in-dp(gtx, 12)), q.Answer, st, 20)
				off.Pop()
			}
		}
		return y
	case e.Body != nil:
		// The panel: 2dp under the line, 6dp and 8dp under it.
		off := op.Offset(image.Pt(0, dp(gtx, 2))).Push(gtx.Ops)
		h := th.toolInspector(gtx, e.Body, "entry:"+e.ID)
		off.Pop()
		return dp(gtx, 2) + h + dp(gtx, 6+8)
	}
	return 0
}

// traceBox is ReasoningTraceContent: thoughts as a reply's markdown but in
// the foreground with their line breaks kept, 12dp apart, in px in and
// padded 2 by 4, scrolling past 384dp; while one streams the box keeps
// to its live end, within 4dp (WorkLog.tsx:128-170).
func (th *Theme) traceBox(gtx C, id string, in int, thoughts []string, live bool) int {
	w := gtx.Constraints.Max.X
	st := mdStyle{color: P.Foreground, on: P.Background, breaks: true}
	m := op.Record(gtx.Ops)
	y := dp(gtx, 4)
	for i, t := range thoughts {
		if i > 0 {
			y += dp(gtx, 12)
		}
		off := op.Offset(image.Pt(dp(gtx, 2), y)).Push(gtx.Ops)
		st.owner = id + ":" + strconv.Itoa(i)
		y += th.markdown(within(gtx, w-dp(gtx, 4)), t, st, nil).Size.Y
		off.Pop()
	}
	y += dp(gtx, 4)
	call := m.Stop()
	defer op.Offset(image.Pt(in, 0)).Push(gtx.Ops).Pop()
	follow := float32(0)
	if live {
		follow = 4
	}
	return th.followScroll(gtx, id+":trace", image.Pt(w, min(y, dp(gtx, 384))), y, follow, call)
}

// plainText draws s wrapped in t's font, UI's by default, its line breaks
// kept, lines lineH dp apart.
func (th *Theme) plainText(gtx C, s string, t Txt, lineH float32) int {
	return th.inline(gtx, th.plainPara(s, t, lineH), nil).Size.Y
}

// plainPara is plainText's paragraph.
func (th *Theme) plainPara(s string, t Txt, lineH float32) *inlineText {
	if t.Font.Typeface == "" {
		t.Font = UI(font.Normal)
	}
	p := &inlineText{font: t.Font, size: float32(t.Size), lineH: lineH}
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			p.runs = append(p.runs, inlineRun{brk: true})
		}
		if line != "" {
			p.runs = append(p.runs, inlineRun{text: line, font: t.Font, size: float32(t.Size), color: t.Color})
		}
	}
	return th.keepText(s, t, p)
}

// keepText keeps a one-off paragraph for as long as its text is shown:
// layouts are cached by their paragraph. s, with t's colour, size and
// font, names it.
func (th *Theme) keepText(s string, t Txt, p *inlineText) *inlineText {
	k := plainKey{s, t.Color, float32(t.Size), p.lineH, t.Font}
	if old := th.tl.plain[k]; old != nil {
		return old
	}
	if th.tl.plain == nil || len(th.tl.plain) > 2000 {
		th.tl.plain = map[plainKey]*inlineText{}
	}
	th.tl.plain[k] = p
	return p
}

type plainKey struct {
	s     string
	col   color.NRGBA
	size  float32
	lineH float32
	font  font.Font
}

// followScroll draws a recorded block content px tall in a size box,
// scrolled by the wheel when it's taller. A follow of more than 0 keeps it
// at its end as it grows while it was there, within follow dp.
func (th *Theme) followScroll(gtx C, id string, size image.Point, content int, follow float32, call op.CallOp) int {
	s := th.tl.scrolls[id]
	if s == nil {
		s = &hscroll{}
		th.tl.scrolls[id] = s
		if follow > 0 {
			s.x = content
		}
	}
	most := max(0, content-size.Y)
	if follow > 0 && s.atEnd {
		s.x = most
	}
	d := s.g.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -s.x, Max: most - s.x})
	s.x = max(0, min(most, s.x+d))
	s.atEnd = most-s.x <= dp(gtx, follow)
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if most > 0 {
		s.g.Add(gtx.Ops)
	}
	off := op.Offset(image.Pt(0, -s.x)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return size.Y
}

// groupHeader is a closed or open group's row, WorkLogButton: its summary
// icon and summary, and its time on hover; it opens the group.
func (th *Theme) groupHeader(gtx C, r *TimelineRow, hot bool) int {
	w := gtx.Constraints.Max.X
	py, px := dp(gtx, 2), dp(gtx, 2)
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	line := th.workLine(within(gtx, w-2*px), workIcon(r.Icon, mutedIcon(P.IconMuted)), func(gtx C) D {
		return th.textLine(gtx, Txt{Size: 14, Color: P.SecondaryLabel}, 22.75, r.Label)
	}, []W{th.timestamp(r.ID, r.At, hot)}, false)
	off.Pop()
	h := line + 2*py
	th.hit(gtx, "group:"+r.Group, image.Pt(w, h))
	return h
}

// groupList is an open group's entries, 24dp each, in a list that scrolls
// past min(18rem, 50% of the window) plus what its open entries add. The
// list keeps its scrollbar's 6dp gutter whether or not it scrolls
// (scrollbar-gutter: stable), so its rows end 6dp short of the lane.
func (th *Theme) groupList(gtx C, r *TimelineRow) int {
	w := gtx.Constraints.Max.X
	thoughts, live := true, false
	for _, e := range r.Entries {
		thoughts = thoughts && e.Thought != ""
		live = live || e.Thinking
	}
	if thoughts {
		// A group of only thoughts opens to their trace.
		var ts []string
		for _, e := range r.Entries {
			ts = append(ts, e.Thought)
		}
		return th.traceBox(within(gtx, w-dp(gtx, 28)), r.ID, dp(gtx, 28), ts, live)
	}
	gutter := dp(gtx, Metrics.AppScrollbarWidth)
	m := op.Record(gtx.Ops)
	y, extra := 0, 0
	for i := range r.Entries {
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		h := th.entryRow(within(gtx, w-gutter), &r.Entries[i], true)
		off.Pop()
		y += h
		extra += max(0, h-dp(gtx, 24))
	}
	call := m.Stop()
	limit := dp(gtx, 288)
	if th.tl.canvasH > 0 {
		limit = min(limit, (th.tl.canvasH+dp(gtx, topBarH))/2)
	}
	limit += extra
	if y <= limit {
		call.Add(gtx.Ops)
		return y
	}
	h := th.followScroll(gtx, r.ID, image.Pt(w, limit), y, 0, call)
	// The 6dp bar in the gutter, its thumb in --app-scrollbar-thumb.
	if s := th.tl.scrolls[r.ID]; s != nil {
		length := max(dp(gtx, 16), limit*limit/y)
		ty := (limit - length) * s.x / max(1, y-limit)
		fill(gtx, over(P.AppScrollbarThumb, P.Background), image.Rect(w-gutter, ty, w, ty+length), gutter/2)
	}
	return h
}

// liveRow is the running tool (LiveWorkEntryTimelineRow): the newest
// thought over it when there is one, then the tool's row hugging its label,
// which shines while the tool runs. It opens the group.
func (th *Theme) liveRow(gtx C, r *TimelineRow) int {
	e := r.Entry
	if e.Approval != nil {
		return th.approvalRow(gtx, e.Approval).Size.Y
	}
	w := gtx.Constraints.Max.X
	py, px := dp(gtx, 2), dp(gtx, 2)
	y := 0
	if r.Thought != "" {
		// The thought: up to four lines beside a brain.
		off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
		line := th.workLine(within(gtx, w-2*px), workIcon("brain", mutedIcon(P.IconMuted)), func(gtx C) D {
			return D{Size: image.Pt(gtx.Constraints.Max.X, th.clampedText(gtx, r.Thought, Txt{Size: 14, Color: P.SecondaryLabel}, 22.75, 4))}
		}, nil, true)
		off.Pop()
		y += line + 2*py
	}
	text := e.Label
	if e.Thought != "" {
		text = th.thoughtPreview(e.Thought, e.Thinking)
	}
	iconCol := mutedIcon(P.IconMuted)
	if e.Failed {
		iconCol = mutedIcon(failedTint())
	}
	shiny := r.Live && !e.Failed
	off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
	h := th.activityRow(within(gtx, w), workIcon(e.Icon, iconCol), text, P.SecondaryLabel, func(gtx C, label image.Point) {
		if shiny {
			th.shine(gtx, label, func(col color.NRGBA) W {
				return func(gtx C) D { return th.textLine(gtx, Txt{Size: 14, Color: col}, 22.75, text) }
			})
		}
	})
	off.Pop()
	y += h
	th.hit(gtx, "group:"+r.Group, image.Pt(w, y))
	return y
}

// activityRow is LiveActivityRow's row: the icon and the label on a
// WorkLogRow line, hugging them. over draws on top of the label, its box
// at the current offset.
func (th *Theme) activityRow(gtx C, icon W, label string, col color.NRGBA, over func(gtx C, label image.Point)) int {
	w := gtx.Constraints.Max.X
	py, px := dp(gtx, 2), dp(gtx, 2)
	slot := dp(gtx, 24+6)
	lab := th.recorded(within(gtx, w-2*px-slot), func(gtx C) D { return th.textLine(gtx, Txt{Size: 14, Color: col}, 22.75, label) })
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	line := th.workLine(within(gtx, slot+lab.Size.X), icon, func(gtx C) D {
		lab.call.Add(gtx.Ops)
		return D{Size: lab.Size}
	}, nil, false)
	if over != nil {
		lo := op.Offset(image.Pt(slot, dp(gtx, 1.09))).Push(gtx.Ops)
		over(within(gtx, lab.Size.X), lab.Size)
		lo.Pop()
	}
	off.Pop()
	return line + 2*py
}

// thinkingRow is "Thinking" (ThinkingTimelineRow): a brain and the label,
// with the foreground band sweeping across them while it's live.
func (th *Theme) thinkingRow(gtx C, r *TimelineRow) int {
	w := gtx.Constraints.Max.X
	draw := func(icon, text color.NRGBA) func(gtx C) int {
		return func(gtx C) int { return th.activityRow(gtx, workIcon("brain", icon), r.Label, text, nil) }
	}
	slot := dp(gtx, 2+24+6+2)
	lab := th.recorded(within(gtx, w), func(gtx C) D { return th.textLine(gtx, Txt{Size: 14, Color: P.MutedForeground}, 22.75, r.Label) })
	h := draw(mutedIcon(P.IconMuted), P.MutedForeground)(within(gtx, w))
	if r.Live {
		th.shimmer(gtx, image.Pt(slot+lab.Size.X, h), func(gtx C) D {
			return D{Size: image.Pt(gtx.Constraints.Max.X, draw(P.Foreground, P.Foreground)(gtx))}
		})
	}
	if r.Group != "" {
		th.hit(gtx, "group:"+r.Group, image.Pt(w, h))
	}
	return h
}

// clampedText is s wrapped to at most lines lines lineH dp apart, the last
// ending in "…" when it's cut (line-clamp).
func (th *Theme) clampedText(gtx C, s string, t Txt, lineH float32, lines int) int {
	t.MaxLines, t.LineH = lines, lineH/float32(t.Size)
	g := gtx
	g.Constraints.Min = image.Point{}
	m := op.Record(gtx.Ops)
	d := th.Text(g, t, s)
	call := m.Stop()
	if t.Font.Typeface == "" {
		t.Font = UI(font.Normal)
	}
	mt := th.metrics(t.Font, gtx.Sp(t.Size))
	content := int(mt.ascent + mt.descent + 0.5)
	pitch := dp(gtx, lineH)
	n := 1 + max(0, (d.Size.Y-content+pitch/2)/pitch)
	off := op.Offset(image.Pt(0, (pitch-content)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return n * pitch
}

// jsonValid is whether s is JSON, or JSON documents a blank line apart.
func jsonValid(s string) bool {
	if json.Valid([]byte(s)) {
		return true
	}
	docs := blankLines.Split(s, -1)
	if len(docs) < 2 {
		return false
	}
	for _, d := range docs {
		if !json.Valid([]byte(d)) {
			return false
		}
	}
	return true
}

var blankLines = regexp.MustCompile(`\n\s*\n`)

// errorRow is a failed run's error (TimelineWorkEntryRow.tsx:239-305): a
// circle-alert and the medium label in --destructive (a usage limit in
// --warning), its time always shown, the provider's message under it.
func (th *Theme) errorRow(gtx C, e *WorkEntry) int {
	w := gtx.Constraints.Max.X
	py, px := dp(gtx, 2), dp(gtx, 2)
	col := P.Destructive
	if e.Error.Warning {
		col = P.Warning
	}
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	line := th.workLine(within(gtx, w-2*px), LucideW("circle-alert", col, 16), func(gtx C) D {
		p := th.keepText(e.Error.Label, Txt{Size: 14, Font: UI(font.Medium), Color: col}, &inlineText{font: UI(font.Medium), size: 14, lineH: 20,
			runs: []inlineRun{{text: e.Error.Label, font: UI(font.Medium), size: 14, color: col}}})
		return th.inline(gtx, p, nil)
	}, []W{th.timestamp("entry:"+e.ID, e.At, true)}, true)
	off.Pop()
	h := py + line
	if e.Error.Message != "" {
		mo := op.Offset(image.Pt(px+dp(gtx, 28), h+dp(gtx, 4))).Push(gtx.Ops)
		h += dp(gtx, 8) + th.plainText(within(gtx, w-2*px-dp(gtx, 28)), e.Error.Message, Txt{Size: 14, Color: over(faded(P.Foreground, 0xcc), P.Background)}, 22.75)
		mo.Pop()
	}
	return h + py
}

// approvalIcons are the approval row's icons by request kind
// (TimelinePendingApprovalRow.tsx:15-31).
var approvalIcons = map[string]string{"command": "terminal", "file-change": "square-pen", "file-read": "eye", "permission": "wrench", "mcp-elicitation": "hammer"}

// approvalLabels name what an approval asks for when it says nothing.
var approvalLabels = map[string]string{"command": "Command approval", "file-change": "File change approval", "file-read": "File read approval",
	"permission": "App permission approval", "mcp-elicitation": "App access approval"}

// approvalRow is a pending approval in the timeline: the request's
// detail in mono 12/20, then "Waiting for approval" and its answers,
// 30dp in.
func (th *Theme) approvalRow(gtx C, a *PendingApproval) D {
	w := gtx.Constraints.Max.X
	py, px := dp(gtx, 2), dp(gtx, 2)
	icon := approvalIcons[a.Kind]
	if icon == "" {
		icon = "wrench"
	}
	detail := a.Detail
	if detail == "" {
		detail = approvalLabels[a.Kind]
	}
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	line := th.workLine(within(gtx, w-2*px), workIcon(icon, mutedIcon(P.IconMuted)), func(gtx C) D {
		cl := th.codeText(gtx, detail, "", P.Foreground, 12, 20, gtx.Constraints.Max.X)
		h := min(cl.size.Y, dp(gtx, 160))
		th.followScroll(gtx, "approval:"+a.ID, image.Pt(gtx.Constraints.Max.X, h), cl.size.Y, 0, cl.call)
		return D{Size: image.Pt(cl.size.X, h)}
	}, nil, true)
	off.Pop()
	y := py + line
	in := px + dp(gtx, 30)
	text := func(s string, col color.NRGBA) {
		y += dp(gtx, 4)
		o := op.Offset(image.Pt(in, y)).Push(gtx.Ops)
		y += th.plainText(within(gtx, w-in), s, Txt{Size: 12, Color: col}, 16)
		o.Pop()
	}
	if a.App != "" {
		text(a.App, P.MutedForeground)
	}
	if a.Reason != "" {
		text(a.Reason, P.MutedForeground)
	}
	y += dp(gtx, 4)
	ao := op.Offset(image.Pt(in, y)).Push(gtx.Ops)
	actions := th.recorded(within(gtx, w-in), func(gtx C) D { return th.approvalActions(gtx, a) })
	actions.call.Add(gtx.Ops)
	ao.Pop()
	y += actions.Size.Y + dp(gtx, 4) + py
	return D{Size: image.Pt(w, y)}
}

// defaultApprovalOptions are T3's answers when the request offers none.
var defaultApprovalOptions = []ApprovalOption{{Decision: "cancel", Label: "Cancel"}, {Decision: "decline", Label: "Decline"},
	{Decision: "acceptForSession", Label: "Always allow this session"}, {Decision: "accept", Label: "Approve"}}

// approvalActions is the approval's 12/16 line: the warning shield and
// "Waiting for approval", then Decline and Approve as inline buttons and
// the rest under More, a menu above it (ComposerPendingApprovalActions,
// inline). Their click targets are "approval:<decision>:<ID>", and while
// an answer is on its way they're off. A risky answer has a
// triangle-alert, its warning in a tooltip.
func (th *Theme) approvalActions(gtx C, a *PendingApproval) D {
	if !a.Live {
		return th.textLine(gtx, Txt{Size: 12, Color: P.Warning}, 16, "Provider process is gone. Interrupt or restart the run to respond.")
	}
	opts := a.Options
	if len(opts) == 0 {
		opts = defaultApprovalOptions
	}
	kids := []layout.FlexChild{
		layout.Rigid(LucideW("shield", P.Warning, 12)),
		layout.Rigid(Spacer(4, 0)),
		layout.Rigid(func(gtx C) D { return th.textLine(gtx, Txt{Size: 12, Color: P.Warning}, 16, "Waiting for approval") }),
		layout.Rigid(Spacer(8, 0)),
		layout.Rigid(func(gtx C) D { return th.textLine(gtx, Txt{Size: 12, Color: P.MutedForeground}, 16, "·") }),
	}
	var more []menuChoice
	var moreIDs []string
	for _, o := range opts {
		id := ""
		if !a.Responding {
			id = "approval:" + o.Decision + ":" + a.ID
		}
		if o.Decision != "decline" && o.Decision != "accept" {
			more, moreIDs = append(more, menuChoice{o.Label, o.Warning}), append(moreIDs, id)
			continue
		}
		o := o
		kids = append(kids, layout.Rigid(Spacer(8, 0)), layout.Rigid(func(gtx C) D {
			d := th.approvalButton(id, o.Label, o.Decision != "accept", o.Warning != "")(gtx)
			th.tipAt(gtx, id, o.Warning, d.Size, true, 0)
			return d
		}))
	}
	menuID := ""
	if !a.Responding {
		menuID = "approval.more:" + a.ID
	}
	if len(more) > 0 {
		kids = append(kids, layout.Rigid(Spacer(8, 0)), layout.Rigid(func(gtx C) D {
			d := th.approvalButton(menuID, "More", true, false)(gtx)
			if menuID == "" {
				return d
			}
			if th.clicked(menuID) {
				th.toggleMenu(menuID)
			}
			// side=top align=end, 4dp over the trigger.
			if i := th.rowMenu(gtx, menuID, image.Pt(d.Size.X, -dp(gtx, 4)), true, more); i >= 0 {
				th.push(Action{ID: moreIDs[i], Kind: "click"})
			}
			return d
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
}

// approvalButton is an approval's answer, an InlineButton: medium 12/16
// text, underlined on hover; the muted tone brightens to the foreground.
// warn puts a 12dp triangle-alert 2dp before the label. Without an id
// it's off: it draws at rest and takes no clicks.
func (th *Theme) approvalButton(id, label string, muted, warn bool) W {
	return func(gtx C) D {
		hovered := false
		if id != "" {
			c, _ := th.events(gtx, id)
			hovered = c.Hovered()
		}
		col := P.Foreground
		if muted && !hovered {
			col = P.MutedForeground
		}
		x := 0
		if warn {
			io := op.Offset(image.Pt(0, (dp(gtx, 16)-dp(gtx, 12))/2)).Push(gtx.Ops)
			LucideW("triangle-alert", col, 12)(gtx)
			io.Pop()
			x = dp(gtx, 12+2)
		}
		off := op.Offset(image.Pt(x, 0)).Push(gtx.Ops)
		d := th.textLine(gtx, Txt{Size: 12, Font: UI(font.Medium), Color: col}, 16, label)
		if hovered {
			// underline-offset-2 under the baseline.
			y := d.Size.Y - d.Baseline + dp(gtx, 2)
			hline(gtx, col, y, 0, d.Size.X)
		}
		off.Pop()
		d.Size.X += x
		if id != "" {
			th.hit(gtx, id, d.Size)
		}
		return d
	}
}

// toolInspector is V2ItemInspector (V2ItemInspector.tsx:197-416): an open
// tool's call and its result, 12/16 with mono 13/21.125 blocks. Its
// sections are 8dp apart; a call's command, arguments and output 6.
func (th *Theme) toolInspector(gtx C, b *EntryBody, id string) int {
	w := gtx.Constraints.Max.X
	mono := func(gtx C, s string, col color.NRGBA, lang string, key string, tall bool) int {
		cl := th.codeText(gtx, s, lang, col, 13, 21.125, gtx.Constraints.Max.X)
		if !tall {
			cl.call.Add(gtx.Ops)
			return cl.size.Y
		}
		// StructuredValue and the command output: 320dp, then it scrolls;
		// the output follows its end within 8dp.
		follow := float32(0)
		if key == "output" {
			follow = 8
		}
		return th.followScroll(gtx, id+":"+key, image.Pt(gtx.Constraints.Max.X, min(cl.size.Y, dp(gtx, 320))), cl.size.Y, follow, cl.call)
	}
	fg85 := over(faded(P.Foreground, 0xd9), P.Background)
	type part struct {
		gap  float32 // dp above it
		draw func(gtx C) int
	}
	var parts []part
	add := func(gap float32, f func(gtx C) int) { parts = append(parts, part{gap, f}) }
	status := func(s string, col color.NRGBA, italic bool) {
		add(6, func(gtx C) int {
			p := th.keepText(s, Txt{Size: 13, Font: Mono(), Color: col}, &inlineText{font: Mono(), size: 13, lineH: 21.125,
				runs: []inlineRun{{text: s, font: Mono(), size: 13, color: col, italic: italic}}})
			return th.inline(gtx, p, nil).Size.Y
		})
	}
	if b.Command != "" {
		add(6, func(gtx C) int { return th.inline(gtx, th.shellCommand(b.Command, fg85), nil).Size.Y })
	}
	for i, arg := range b.Args {
		arg := arg
		gap := float32(6)
		if i > 0 {
			gap = 0 // one block, a line each
		}
		add(gap, func(gtx C) int {
			if arg.Lang == "" {
				// The value runs on after "key ", wrapping (pre-wrap).
				p := &inlineText{font: Mono(), size: 13, lineH: 21.125}
				p.runs = append(p.runs, inlineRun{text: arg.Key + " ", font: Mono(), size: 13, color: P.MutedForeground})
				for i, line := range strings.Split(expandTabs(arg.Value), "\n") {
					if i > 0 {
						p.runs = append(p.runs, inlineRun{brk: true})
					}
					if line != "" {
						p.runs = append(p.runs, inlineRun{text: line, font: Mono(), size: 13, color: fg85})
					}
				}
				return th.inline(gtx, th.keepText("arg\x00"+arg.Key+"\x00"+arg.Value, Txt{Size: 13, Font: Mono(), Color: fg85}, p), nil).Size.Y
			}
			// Code goes under its key, highlighted (StructuredValue).
			key := th.codeText(gtx, arg.Key+" ", "", P.MutedForeground, 13, 21.125, 0)
			key.call.Add(gtx.Ops)
			off := op.Offset(image.Pt(0, key.size.Y)).Push(gtx.Ops)
			h := mono(gtx, arg.Value, P.MutedForeground, arg.Lang, "arg:"+arg.Key, true)
			off.Pop()
			return key.size.Y + h
		})
	}
	if b.ArgsText != "" {
		add(6, func(gtx C) int { return mono(gtx, b.ArgsText, P.MutedForeground, jsonLang(b.ArgsText), "args", true) })
	}
	switch {
	case b.Output != "":
		add(6, func(gtx C) int { return mono(gtx, b.Output, P.MutedForeground, jsonLang(b.Output), "output", true) })
	case b.Pending:
		status("Loading output…", P.MutedForeground, true)
	case b.Err != "":
		status("Couldn't load output: "+b.Err, P.Destructive, false)
	case b.Missing:
		status("No output.", P.MutedForeground, true)
	}
	if b.Exit != 0 {
		status("exit "+strconv.Itoa(b.Exit), P.Destructive, false)
	}
	if b.File != "" {
		add(8, func(gtx C) int {
			kids := []layout.FlexChild{layout.Rigid(func(gtx C) D {
				return th.textLine(gtx, Txt{Size: 12, Font: Mono(), Color: P.MutedForeground}, 16, b.File)
			})}
			if b.Counts {
				kids = append(kids, layout.Rigid(Spacer(8, 0)),
					layout.Rigid(func(gtx C) D {
						return th.textLine(gtx, Txt{Size: 12, Color: P.DiffAddition}, 16, "+"+groupDigits(b.Add))
					}),
					layout.Rigid(Spacer(4, 0)),
					layout.Rigid(func(gtx C) D {
						return th.textLine(gtx, Txt{Size: 12, Color: P.DiffDeletion}, 16, "-"+groupDigits(b.Del))
					}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...).Size.Y
		})
	}
	if b.Failure != "" {
		add(8, func(gtx C) int { return mono(gtx, b.Failure, P.MutedForeground, "", "failure", true) })
	}
	// The operations: mono 12/16 muted, 4dp apart.
	for i, c := range b.Changes {
		c := c
		gap := float32(8)
		if i > 0 {
			gap = 4
		}
		add(gap, func(gtx C) int {
			return th.plainText(gtx, c, Txt{Size: 12, Font: Mono(), Color: P.MutedForeground}, 16)
		})
	}
	if len(b.Searches) > 0 {
		patterns := strings.Join(b.Searches, "\n")
		add(8, func(gtx C) int { return mono(gtx, patterns, fg85, "", "search", false) })
	}
	for i, hit := range b.Hits {
		hit := hit
		// File hits are 4dp apart, a page's 6.
		gap := float32(8)
		switch {
		case i > 0 && hit.Web:
			gap = 6
		case i > 0:
			gap = 4
		}
		add(gap, func(gtx C) int {
			var h int
			if hit.Web {
				h = th.webHit(gtx, hit)
			} else {
				h = th.plainText(gtx, hit.Title, Txt{Size: 12, Font: Mono(), Color: over(faded(P.Foreground, 0xcc), P.Background)}, 16)
			}
			if hit.Text != "" {
				// A file's matching line 2dp under it; a page's snippet
				// right under.
				if !hit.Web {
					h += dp(gtx, 2)
				}
				off := op.Offset(image.Pt(0, h)).Push(gtx.Ops)
				h += th.plainText(gtx, hit.Text, Txt{Size: 12, Color: P.MutedForeground}, 16)
				off.Pop()
			}
			return h
		})
	}
	if b.Plain != "" {
		add(8, func(gtx C) int { return mono(gtx, b.Plain, P.SecondaryLabel, "", "plain", true) })
	}
	if b.Value != "" {
		add(8, func(gtx C) int { return mono(gtx, b.Value, P.MutedForeground, "", "value", true) })
	}
	y := 0
	for i, p := range parts {
		if i > 0 {
			y += dp(gtx, p.gap)
		}
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		y += p.draw(within(gtx, w))
		off.Pop()
	}
	return y
}

// webHit is a web search result's title: medium 12/16 in the foreground
// with a 12dp external-link 4dp after it. With a web address it opens the
// page, underlined on hover (V2ItemInspector.tsx:355-372).
func (th *Theme) webHit(gtx C, hit SearchHit) int {
	id := "link:" + hit.URL
	hovered := false
	if hit.URL != "" {
		c, _ := th.events(gtx, id)
		hovered = c.Hovered()
	}
	key := "web\x00" + hit.Title
	if hovered {
		key = "web.hover\x00" + hit.Title
	}
	t := Txt{Size: 12, Font: UI(font.Medium), Color: P.Foreground}
	p := &inlineText{font: t.Font, size: 12, lineH: 16}
	p.runs = append(p.runs, inlineRun{text: hit.Title, font: t.Font, size: 12, color: t.Color, underline: hovered})
	if hit.URL != "" {
		p.runs = append(p.runs, inlineRun{gap: 4}, inlineRun{icon: "external-link", size: 12, color: t.Color})
	}
	d := th.inline(gtx, th.keepText(key, t, p), nil)
	if hit.URL != "" {
		th.hit(gtx, id, d.Size)
	}
	return d.Size.Y
}

// shellCommand is an expanded command row's command, highlighted per shell
// word as T3's shellscript grammar colours it (ShellCommandBlock.tsx):
// each command's name a function, its arguments variables, numbers,
// quoted strings, operators and comments their own, the rest in the
// editor's colour; control characters shown as their symbols. The block
// itself is foreground at 85%.
func (th *Theme) shellCommand(cmd string, col color.NRGBA) *inlineText {
	cmd = visibleControls(strings.TrimSpace(cmd))
	p := &inlineText{font: Mono(), size: 13, lineH: 21.125}
	pal := syntax()
	for _, tok := range shellTokens(cmd) {
		if tok.brk {
			p.runs = append(p.runs, inlineRun{brk: true})
			continue
		}
		c, ok := pal[tok.scope]
		if !ok {
			c = pal["fg"]
		}
		p.runs = append(p.runs, inlineRun{text: tok.text, font: Mono(), size: 13, color: c})
	}
	return th.keepText("shell\x00"+cmd, Txt{Size: 13, Font: Mono(), Color: col}, p)
}

// shellToken is a stretch of a command in one scope, or a line break.
type shellToken struct {
	text, scope string
	brk         bool
}

// shellTokens splits a command into the words and marks the shellscript
// grammar tells apart: a command's first word is a function, the words
// after it are arguments; a quoted string is a string, a word of digits a
// number, | || && ; & and redirections operators, # to the line's end a
// comment.
func shellTokens(cmd string) []shellToken {
	var out []shellToken
	first := true // the next word names a command
	rs := []rune(cmd)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '\n':
			out = append(out, shellToken{brk: true})
			first = true
			i++
		case r == ' ' || r == '\t':
			j := i
			for j < len(rs) && (rs[j] == ' ' || rs[j] == '\t') {
				j++
			}
			out = append(out, shellToken{text: string(rs[i:j])})
			i = j
		case r == '#' && (i == 0 || rs[i-1] == ' ' || rs[i-1] == '\t'):
			j := i
			for j < len(rs) && rs[j] != '\n' {
				j++
			}
			out = append(out, shellToken{text: string(rs[i:j]), scope: "comment"})
			i = j
		case strings.ContainsRune("|&;<>()", r):
			j := i
			for j < len(rs) && strings.ContainsRune("|&;<>()", rs[j]) {
				j++
			}
			op := string(rs[i:j])
			out = append(out, shellToken{text: op, scope: "operator"})
			// After a pipe, a list or a subshell another command starts;
			// a redirection's target is still an argument.
			first = !strings.ContainsAny(op, "<>")
			i = j
		case r == ',':
			out = append(out, shellToken{text: ","})
			i++
		default:
			j := i
			quote := rune(0)
			for j < len(rs) {
				c := rs[j]
				if quote != 0 {
					if c == '\\' && quote == '"' && j+1 < len(rs) {
						j += 2
						continue
					}
					if c == quote {
						quote = 0
					}
					j++
					continue
				}
				if c == '\'' || c == '"' {
					quote = c
					j++
					continue
				}
				if c == '\\' && j+1 < len(rs) {
					j += 2
					continue
				}
				if c == ' ' || c == '\t' || c == '\n' || c == ',' || strings.ContainsRune("|&;<>()", c) {
					break
				}
				j++
			}
			word := string(rs[i:j])
			scope := "variable"
			switch {
			case word[0] == '\'' || word[0] == '"':
				scope = "string"
			case first:
				scope = "function"
			case strings.Trim(word, "0123456789") == "":
				scope = "number"
			}
			// A leading NAME=value is an assignment, and a command follows.
			if first && strings.Contains(word, "=") && !strings.ContainsAny(word[:strings.Index(word, "=")], "'\"$/") {
				scope = "variable"
			} else {
				first = false
			}
			out = append(out, shellToken{text: word, scope: scope})
			i = j
		}
	}
	return out
}

// visibleControls is T3's withVisibleControlCharacters: control characters
// other than tabs and line breaks drawn as their Control Pictures.
func visibleControls(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == 0x7f:
			return '␡'
		case r < 0x20 && r != '\t' && r != '\n' && r != '\r':
			return 0x2400 + r
		}
		return r
	}, s)
}

// JSONText is T3's isJsonToolText: JSON, or JSON documents a blank line
// apart.
func JSONText(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" || !strings.ContainsAny(t[:1], "{[\"") && t != "true" && t != "false" && t != "null" {
		return false
	}
	return jsonValid(t)
}

func jsonLang(s string) string {
	if JSONText(s) {
		return "json"
	}
	return ""
}
