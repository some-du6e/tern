package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// What the timeline shows of the derived rows: their words and icons, an
// open tool's body (V2ItemInspector.tsx), the user's bubble.

// timeline fills the scene's timeline from the open thread.
func (a *App) timeline(s *ui.Scene, t *t3.Thread, v *threadView, root string) {
	tl := &s.Timeline
	tl.ID = a.openID
	_, tl.CanRun = v.env.projects[t.ProjectID]
	tl.MoreHistory, tl.LoadingHistory, tl.HistoryError = v.more, v.loading, v.historyErr
	in := timelineInput{working: phaseWorking(t), latest: latestRun(v), running: t.ActiveRunID,
		openRuns: v.unfolded, openGroups: v.groups}
	in.turnEnded = in.latest != nil && turnEnded(in.latest, v.turns)
	tl.Working = in.working
	// The pending approval is the first the composer shows: the typing
	// guard holds one back from the timeline too (07-composer.md §5.13).
	var pending *t3.RuntimeRequest
	if shown := a.shownApprovals(); len(shown) > 0 {
		pending = &t3.RuntimeRequest{ID: shown[0].id, Kind: shown[0].kind}
		in.pending = pending.ID
	}
	if in.working {
		in.activeStarted = parseTime(t.ActivityRunStartedAt)
	}
	// A run the user stops opens its fold; a newer run closes the one
	// before's again (MessagesTimeline.tsx:596-631).
	if r := in.latest; r != nil {
		switch {
		case r.ID == v.seenRun && v.seenStatus == "running" && r.Status == "interrupted":
			v.unfolded[r.ID] = true
		case r.ID != v.seenRun && v.seenRun != "":
			delete(v.unfolded, v.seenRun)
		}
		v.seenRun, v.seenStatus = r.ID, r.Status
	}
	items := v.sorted()
	rows := deriveRows(timelineEntries(items), in)
	if a.cited.thread == a.openID && a.cited.message != "" {
		rows = a.revealCited(tl, v, rows, func() []*row { return deriveRows(timelineEntries(items), in) })
	}
	claimed := false
	for _, r := range rows {
		if ur, ok := a.uiRow(r, v, root, in.pending); ok {
			tl.Rows = append(tl.Rows, ur)
		}
		var shown []*workEntry
		switch r.kind {
		case "work":
			shown = r.entries
		case "work-live":
			shown = []*workEntry{r.live}
		}
		for _, e := range shown {
			claimed = claimed || e.item.Type == "approval_request" && e.item.RequestID == in.pending
		}
	}
	// An approval no row shows goes under the rows.
	if in.pending != "" && !claimed {
		var asked *t3.TurnItem
		for _, it := range items {
			if it.Type == "approval_request" && it.RequestID == in.pending {
				asked = it
			}
		}
		tl.Approvals = append(tl.Approvals, a.uiApproval(pending, asked, v.requestLive(in.pending)))
	}
}

// citedReply is a quoted response's source: its thread's ref and the
// reply's message id.
type citedReply struct{ thread, message string }

// goToCited follows a quoted response's card, arg its escaped
// "<environment>/<thread>/<message>": to its thread, where the timeline
// then shows the reply (assistantCitationNavigation).
func (a *App) goToCited(th *ui.Theme, arg string) {
	parts := strings.Split(arg, "/")
	if len(parts) != 3 {
		return
	}
	for i, p := range parts {
		s, err := url.PathUnescape(p)
		if err != nil || s == "" {
			return
		}
		parts[i] = s
	}
	r := ref(parts[0], parts[1])
	a.mu.Lock()
	a.cited = citedReply{thread: r, message: parts[2]}
	open := a.openID == r
	a.mu.Unlock()
	if !open {
		a.switchTo(th, func() { a.openThread(r) })
	}
	a.changed()
}

// revealCited shows the cited reply once its thread's rows are in: it
// opens the fold that hides it (MessagesTimeline.tsx:664-705) and has the
// timeline bring its row into view. A reply the loaded turns don't hold
// fetches the history before them until it comes, or there's no more.
// derive derives the rows again.
func (a *App) revealCited(tl *ui.Timeline, v *threadView, rows []*row, derive func() []*row) []*row {
	msg := a.cited.message
	for _, r := range rows {
		if r.kind == "turn-fold" && r.hidden[msg] && !v.unfolded[r.runID] {
			v.unfolded[r.runID] = true
			rows = derive()
			break
		}
	}
	for _, r := range rows {
		if r.kind == "message" && r.id == msg {
			tl.Reveal, a.cited = r.id, citedReply{}
			return rows
		}
	}
	switch {
	case a.loadEarlierLocked():
		tl.LoadingHistory = true
	case v.loaded && !v.loading:
		a.cited = citedReply{}
	}
	return rows
}

// phaseWorking is T3's isWorking for the timeline: the run is running, or
// waiting on the user (derivePhase, session-logic.ts:1074-1084).
func phaseWorking(t *t3.Thread) bool {
	switch t.ActivityRunStatus {
	case "running", "waiting":
		return true
	case "":
		return t.ActiveRunID != ""
	}
	return false
}

// turnEnded is T3's ThreadRunSummary.turnEnded (threadExecution.ts:120-148):
// the run is still active, but the provider's latest turn for its attempt
// has stopped, so only the run's settling is left.
func turnEnded(r *t3.Run, turns map[string]t3.ProviderTurn) bool {
	switch r.Status {
	case "preparing", "starting", "running", "waiting":
	default:
		return false
	}
	var latest t3.ProviderTurn
	found := false
	for _, pt := range turns {
		if r.ActiveAttemptID != "" && pt.RunAttemptID == r.ActiveAttemptID && (!found || pt.Ordinal > latest.Ordinal) {
			latest, found = pt, true
		}
	}
	return found && latest.Status != "pending" && latest.Status != "running"
}

// requestLive is whether the provider can still take request id's answer:
// T3's responseCapability "live" (TimelinePendingApprovalRow.tsx:40). A
// request the projection doesn't hold yet is taken to be.
func (v *threadView) requestLive(id string) bool {
	r, ok := v.requests[id]
	return !ok || r.ResponseCapability.Type == "" || r.ResponseCapability.Type == "live"
}

// latestRun is the thread's newest run that isn't waiting in the queue.
func latestRun(v *threadView) *t3.Run {
	var out *t3.Run
	for id := range v.runs {
		r := v.runs[id]
		if r.Status == "queued" {
			continue
		}
		if out == nil || r.Ordinal > out.Ordinal || r.Ordinal == out.Ordinal && r.RequestedAt > out.RequestedAt {
			out = &r
		}
	}
	return out
}

// uiRow is a derived row as the timeline draws it.
func (a *App) uiRow(r *row, v *threadView, root, pending string) (ui.TimelineRow, bool) {
	out := ui.TimelineRow{ID: r.id, At: r.at, Continues: r.continues, Open: r.expanded, Group: r.groupID}
	switch r.kind {
	case "message":
		m := r.entry.msg
		if m.role == "user" {
			out.Kind = ui.RowUser
			out.Message = a.userMessage(r, v)
			return out, out.Message != nil
		}
		out.Kind = ui.RowAssistant
		out.Message = a.assistantMessage(r, v)
	case "assistant-meta":
		out.Kind = ui.RowAssistantMeta
		out.Message = a.assistantMessage(r, v)
		out.Message.Meta = true
	case "turn-fold":
		out.Kind, out.Label, out.Group = ui.RowTurnFold, r.label, r.runID
	case "work":
		out.Kind = ui.RowWork
		for _, e := range r.entries {
			we := a.uiEntry(e, v, root, pending)
			if r.label != "" && !r.expanded {
				we.Label = r.label
			}
			out.Entries = append(out.Entries, we)
		}
		out.Open = r.expanded
	case "work-toggle":
		out.Kind, out.Label, out.Icon, out.Failed = ui.RowWorkToggle, r.summary, r.icon, r.failed
	case "work-live":
		out.Kind, out.Live = ui.RowWorkLive, r.active
		e := a.uiEntry(r.live, v, root, pending)
		e.Label = r.live.liveLabel(root, r.active)
		e.Failed = r.live.displayFailed()
		if r.thought != nil && !r.expanded {
			out.Thought = liveThoughtLine(truncate(r.thought.detail, 8192))
		}
		// The newest thought over its own row: the row says what it is.
		if out.Thought != "" && r.thought == r.live {
			e.Label, e.Thought = "Thought", ""
			if r.active {
				e.Label = "Thinking"
			}
		}
		out.Entry, out.Failed = &e, e.Failed
	case "thinking":
		out.Kind, out.Live, out.Label = ui.RowThinking, pending == "", "Thinking"
	case "working":
		out.Kind, out.Since, out.Compacting = ui.RowWorking, r.at, r.compact
	case "context-compaction":
		out.Kind, out.Live, out.Failed, out.Detail = ui.RowCompaction, r.active, r.failed, r.failure
		out.Label = r.label
		if r.title != "" && strings.HasPrefix(r.label, "Context compacted") {
			out.Label = r.title + strings.TrimPrefix(r.label, "Context compacted")
		}
		out.Before, out.After = r.before, r.after
		if r.before != "" {
			out.Label = r.title
			if out.Label == "" {
				out.Label = "Context compacted"
			}
		}
	case "event":
		return eventRow(out, r.entry.item, v.env.id())
	case "proposed-plan":
		out.Kind, out.Plan = ui.RowPlan, r.entry.item.Markdown
	default:
		return out, false
	}
	return out, true
}

// eventRow is an event's divider (V2LifecycleRow.tsx:70-169): an
// interrupted run, its message wrapping under it, or a fork, whose button
// opens the conversation on its other side in environment env.
func eventRow(out ui.TimelineRow, it *t3.TurnItem, env string) (ui.TimelineRow, bool) {
	out.Kind = ui.RowDivider
	switch it.Type {
	case "run_interrupt_result":
		out.Label, out.Detail, out.Icon, out.Wrap = "Run interrupted", it.Message, "circle-stop", true
	case "fork":
		out.Label, out.Icon, out.Action = "Conversation fork", "git-fork", "Open fork"
		other := it.TargetThreadID
		if it.Source != nil && it.Source.Type == "run" {
			out.Label, out.Action, other = "Forked from conversation", "Open source conversation", it.Source.ThreadID
		}
		if other == "" {
			out.Action = ""
		} else {
			out.Thread = ref(env, other)
		}
	default:
		return out, false
	}
	return out, true
}

// userMessage is a user's bubble: its text with pictures as chips.
func (a *App) userMessage(r *row, v *threadView) *ui.Message {
	m := r.entry.msg
	text := m.text
	if attachmentOnly(text) {
		text = ""
	}
	var chips []string
	text = ui.TrimText(t3.ReplaceImages(text, func(label string) string {
		if len(chips) == 9 {
			return "[" + label + "]"
		}
		chips = append(chips, label)
		return ui.ChipText(len(chips))
	}))
	pics := a.attachmentPictures(v.env, m.attachments)
	files := attachmentFiles(m.attachments)
	if text == "" && len(pics) == 0 && len(files) == 0 {
		return nil
	}
	out := &ui.Message{ID: m.id, Text: text, At: m.at, Chips: chips, Pictures: pics, Files: files,
		GroupedNext: r.next, GroupedPrev: r.prev, SentByT3: m.createdBy == "agent" && m.source == "server", PromptLink: promptLink(m.text)}
	if !r.steerCont {
		switch m.intent {
		case "steer":
			out.Steer = ui.SteerSteer
		case "promoted_queued_to_steer":
			out.Steer = ui.SteerPromoted
		}
		out.SteerUnread = out.Steer != ui.SteerNone && m.item.Status == "pending"
	}
	switch m.item.Status {
	case "", "completed", "pending", "waiting":
	default:
		out.Status = m.item.Status
	}
	return out
}

// promptLink is T3's promptLinkFor: the prompt, its context references as
// their labels, as a link that starts a thread with it on T3's hosted
// app, or "" when it's empty or over 8000 characters (copyPromptLink.ts,
// promptLinks.ts buildNewThreadLink).
func promptLink(text string) string {
	if attachmentOnly(text) {
		return ""
	}
	p := strings.TrimSpace(t3.ReplaceImages(text, func(label string) string { return label }))
	if n := len(utf16.Encode([]rune(p))); n == 0 || n > 8000 {
		return ""
	}
	return "https://app.t3.codes/new?prompt=" + formEncode(p)
}

// formEncode is application/x-www-form-urlencoded as URLSearchParams
// writes it: spaces as "+", every byte but alphanumerics and *-._ escaped.
func formEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == ' ':
			b.WriteByte('+')
		case c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("*-._", c) >= 0:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// assistantMessage is an assistant's reply, with its meta row's state.
func (a *App) assistantMessage(r *row, v *threadView) *ui.Message {
	m := r.entry.msg
	out := &ui.Message{ID: m.id, Text: m.text, At: m.updated, Streaming: m.streaming, Meta: r.meta,
		Copy: r.copy && strings.TrimSpace(m.text) != "" && !r.streaming, Fork: v.canFork(m.item), Forking: a.forking == m.id}
	if out.Text == "" && !m.streaming {
		out.Text = "(empty response)"
	}
	if st := m.item.Status; st != "" && st != "completed" {
		out.Badge = st
	}
	return out
}

// canFork is T3's canForkProjectedAssistantItem: a finished reply of a run
// can be forked from, unless its provider's session is known to fork
// neither natively nor by handing the whole thread over.
func (v *threadView) canFork(it *t3.TurnItem) bool {
	if it.Type != "assistant_message" || it.RunID == "" || it.Status != "completed" {
		return false
	}
	s, ok := v.sessions[v.pthreads[it.ProviderThreadID].ProviderSessionID]
	if !ok {
		// Older rows can outlive their session's record.
		return true
	}
	c := s.Capabilities
	return c.Threads.CanForkThread && c.Threads.CanForkFromTurn && c.Identity.NativeThreadIDs == "strong" || c.Context.SupportsFullThreadHandoff
}

// forkFrom forks the open thread from the run that wrote reply message,
// into "<title> fork", and opens the fork (useTimelineRewind.ts:174-212).
func (a *App) forkFrom(message string) {
	a.mu.Lock()
	v, t := a.view, a.curThread()
	run := ""
	if v != nil {
		for _, it := range v.items {
			if it.Type == "assistant_message" && it.MessageID == message {
				run = it.RunID
			}
		}
	}
	if v == nil || t == nil || v.env.conn == nil || run == "" || a.forking != "" {
		a.mu.Unlock()
		return
	}
	a.forking = message
	conn, env, source, title := v.env.conn, v.env.id(), t.ID, t.Title+" fork"
	a.mu.Unlock()
	a.changed()
	go func() {
		target := t3.NewID()
		err := forkThread(conn, map[string]any{"type": "thread.fork", "createdBy": "user", "creationSource": "web",
			"sourceThreadId": source, "targetThreadId": target, "sourcePoint": map[string]any{"type": "run", "runId": run}, "title": title})
		a.mu.Lock()
		a.forking = ""
		a.fail(err)
		a.mu.Unlock()
		a.changed()
		if err == nil {
			a.onUI(func(th *ui.Theme) { a.switchTo(th, func() { a.openThread(ref(env, target)) }) })
		}
	}()
}

// forkThread dispatches a fork. Tests replace it.
var forkThread = func(conn *t3.Conn, cmd map[string]any) error { return conn.Dispatch(cmd) }

// uiEntry is a work entry as its row draws it (TimelineWorkEntryRow.tsx).
func (a *App) uiEntry(e *workEntry, v *threadView, root, pending string) ui.WorkEntry {
	out := ui.WorkEntry{ID: e.id, At: e.at, Icon: e.icon(), Label: e.displayLabel(root), Open: v.open[e.id]}
	it := e.item
	if it.Type == "approval_request" && pending != "" && it.RequestID == pending {
		p := a.uiApproval(&t3.RuntimeRequest{ID: it.RequestID, Kind: it.RequestKind}, it, v.requestLive(it.RequestID))
		out.Approval = &p
		return out
	}
	if it.Type == "error" && it.Status == "failed" {
		ee := &ui.EntryError{Label: e.label}
		if it.Failure != nil {
			ee.Message = it.Failure.Message
			if it.Failure.Class == "usage_limit" {
				ee.Warning, ee.Message = true, ""
				ee.Label = "Usage limit reached."
				if at := parseTime(it.Failure.ResetAt); !at.IsZero() {
					ee.Label += " Retry after " + upcomingTime(at, time.Now()) + "."
				}
			}
		}
		out.Error = ee
		return out
	}
	warning := e.source == "runtime.warning"
	failed := !warning && e.displayFailed()
	danger := !warning && failed && (e.itemType == "error" || !e.toolLike())
	switch {
	case warning:
		out.Tone, out.Icon = ui.ToneWarning, "circle-alert"
	case danger:
		out.Tone, out.Icon = ui.ToneDanger, "circle-alert"
	case !e.toolLike():
		out.Tone = ui.ToneNotice
	}
	out.Failed = failed && !danger
	reasoning := e.itemType == "reasoning"
	if reasoning {
		// The whole thought: only its one-line preview is cut short.
		out.Thought, out.Thinking = e.detail, e.status == "inProgress"
		out.Expands = strings.TrimSpace(e.detail) != ""
		return out
	}
	if q := e.answer; q != nil {
		out.Label, out.Answer, out.Answers = questionPreview(q), answerPreview(q), questionHistory(q)
		out.Expands = true
		return out
	}
	if out.Open && needsDetailFetch(it) {
		a.fetchTool(v, it)
	}
	full, fetchErr := v.fetched(it)
	plain, hasPlain := "", false
	switch {
	case e.groupAction() == "read":
		plain, hasPlain = readOutput(e, root), true
	case e.itemType == "dynamic_tool" && e.toolName == "Skill":
		input, _ := e.input.(map[string]any)
		plain, hasPlain = strings.TrimSpace(str(input["args"])), true
	}
	if hasPlain {
		out.Expands = plain != "" || e.viewedImage != "" || needsDetailFetch(it)
		if out.Open {
			out.Body = &ui.EntryBody{Plain: plain}
			if needsDetailFetch(it) {
				fillOutput(out.Body, it, full, fetchErr)
			}
		}
		return out
	}
	out.Expands = e.viewedImage != "" || hasDetail(it)
	if out.Open {
		out.Body = inspect(it, full, fetchErr, root)
	}
	return out
}

// uiApproval is an approval request as its row shows it; live: the
// provider can still take its answer. Its answers are off while one is on
// its way.
func (a *App) uiApproval(req *t3.RuntimeRequest, it *t3.TurnItem, live bool) ui.PendingApproval {
	p := ui.PendingApproval{ID: req.ID, Kind: req.Kind, Live: live, Responding: a.comp.responding[req.ID]}
	if it != nil {
		p.Detail, p.App, p.Reason = it.Prompt, it.AppName, it.Reason
		for _, o := range it.Options {
			p.Options = append(p.Options, ui.ApprovalOption{Decision: o.Decision, Label: o.Label, Warning: o.Warning})
		}
	}
	return p
}

// upcomingTime is T3's formatUpcomingTimestamp and the zone it names:
// "4:00 PM", "tomorrow at 4:00 PM", "11/20 4:00 PM", then " (<zone>)",
// the zone's IANA name (Intl's resolvedOptions().timeZone).
func upcomingTime(t, now time.Time) string {
	t, now = t.Local(), now.Local()
	days := func(x time.Time) int64 {
		return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, time.UTC).Unix() / 86400
	}
	clock := t.Format("3:04 PM")
	var s string
	switch d := days(t) - days(now); {
	case d == 0:
		s = clock
	case d == 1:
		s = "tomorrow at " + clock
	case t.Year() == now.Year():
		s = fmt.Sprintf("%d/%d %s", t.Month(), t.Day(), clock)
	default:
		s = fmt.Sprintf("%d/%d/%d %s", t.Month(), t.Day(), t.Year(), clock)
	}
	return s + " (" + localZone(t) + ")"
}

// localZone is the system's IANA zone name: TZ's, else the one
// /etc/localtime points into the zone database at. A system that says
// neither, as Windows doesn't, gets the zone's abbreviation.
func localZone(t time.Time) string {
	tz := strings.TrimPrefix(os.Getenv("TZ"), ":")
	if _, name, ok := strings.Cut(tz, "zoneinfo/"); ok {
		return name
	}
	if tz != "" && !strings.HasPrefix(tz, "/") {
		return tz
	}
	if p, err := os.Readlink("/etc/localtime"); err == nil {
		if _, name, ok := strings.Cut(p, "zoneinfo/"); ok {
			return name
		}
	}
	return t.Format("MST")
}

// readOutput is T3's workEntryReadOutput: the files a read read, in full.
func readOutput(e *workEntry, root string) string {
	raw := e.changed
	if len(raw) == 0 && e.itemType == "dynamic_tool" {
		raw = toolFilePaths(map[string]any{"input": e.input})
	}
	var out []string
	seen := map[string]bool{}
	for _, p := range raw {
		p = strings.ReplaceAll(strings.TrimSpace(p), `\`, "/")
		if root != "" && !strings.HasPrefix(p, "/") {
			p = strings.TrimRight(root, "/") + "/" + strings.TrimLeft(strings.TrimPrefix(p, "./"), "/")
		}
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n")
}

// needsDetailFetch is T3's turnItemNeedsDetailFetch (itemDetail.ts:
// 150-159): the socket left the output out, or summarized a tool's input.
func needsDetailFetch(it *t3.TurnItem) bool {
	switch it.Type {
	case "command_execution":
		return it.OutputOmitted
	case "dynamic_tool":
		return it.OutputOmitted || summarized(decode(it.Input))
	}
	return false
}

// summarized is T3's isSummarizedValue: an input the wire cut down to a
// summary.
func summarized(v any) bool {
	obj, ok := v.(map[string]any)
	if !ok || obj["truncated"] != true {
		return false
	}
	_, ok = obj["summary"].(string)
	return ok
}

// detailRevision is T3's turnItemDetailRevision: one key while the item
// runs and another once it's done, so an open row fetches once while it
// streams and again when it finishes.
func detailRevision(it *t3.TurnItem) string {
	switch it.Status {
	case "idle", "pending", "running", "waiting":
		return "live"
	}
	return it.UpdatedAt
}

// hasDetail is T3's turnItemHasDetail: an open row would show something.
func hasDetail(it *t3.TurnItem) bool {
	switch it.Type {
	case "command_execution":
		return strings.TrimSpace(rawString(it.Input)) != "" || it.OutputOmitted || strings.TrimSpace(rawString(it.Output)) != "" ||
			it.ExitCode != nil && *it.ExitCode != 0
	case "file_change", "checkpoint", "fork", "handoff":
		return true
	case "file_search":
		return len(it.Results) > 0 || strings.TrimSpace(it.Pattern) != ""
	case "web_search":
		return len(it.Results) > 0 || len(it.Patterns) > 0
	case "dynamic_tool":
		args, argsText := toolArgs(it.Input, it.ToolName)
		return it.OutputOmitted || len(args) > 0 || argsText != ""
	case "approval_request":
		return strings.TrimSpace(it.Prompt) != ""
	case "user_input_request":
		return len(it.Questions) > 0
	case "notification":
		return strings.TrimSpace(it.Detail) != ""
	case "system_notice":
		return strings.TrimSpace(it.Message) != ""
	case "error":
		return it.Failure != nil && strings.TrimSpace(it.Failure.Message) != ""
	case "todo_list":
		return len(it.Steps) > 0
	}
	return false
}

// inspect is T3's V2ItemInspector: an open tool's call, output and result.
func inspect(it, full *t3.TurnItem, fetchErr, root string) *ui.EntryBody {
	b := &ui.EntryBody{}
	switch it.Type {
	case "command_execution":
		b.Command = strings.TrimSpace(rawString(it.Input))
		fillOutput(b, it, full, fetchErr)
		if it.ExitCode != nil && *it.ExitCode != 0 {
			b.Exit = *it.ExitCode
		}
	case "dynamic_tool":
		src := it
		if full != nil && summarized(decode(it.Input)) {
			src = full
		}
		b.Args, b.ArgsText = toolArgs(src.Input, it.ToolName)
		fillOutput(b, it, full, fetchErr)
	case "file_change":
		src := it
		if full != nil {
			src = full
		}
		b.File = workspacePath(it.FileName, root)
		if it.Additions != nil || it.Deletions != nil {
			b.Counts = true
			if it.Additions != nil {
				b.Add = *it.Additions
			}
			if it.Deletions != nil {
				b.Del = *it.Deletions
			}
		}
		if it.Status == "failed" {
			b.Failure = strings.TrimSpace(src.DiffStr)
		}
		for _, c := range it.Changes {
			s := c.Operation + " "
			if c.OldPath != "" {
				s += c.OldPath + " → "
			}
			s += workspacePath(c.Path, root)
			if kinds := strings.Trim(strings.Join([]string{c.FileType, c.MimeType}, ", "), ", "); kinds != "" {
				s += " (" + kinds + ")"
			}
			b.Changes = append(b.Changes, s)
		}
	case "file_search":
		if p := strings.TrimSpace(it.Pattern); p != "" {
			b.Searches = []string{it.Pattern}
		}
		for _, r := range it.Results {
			title := workspacePath(r.FileName, root)
			if r.Line > 0 {
				title += ":" + strconv.Itoa(r.Line)
			}
			if r.Column > 0 {
				title += ":" + strconv.Itoa(r.Column)
			}
			b.Hits = append(b.Hits, ui.SearchHit{Title: title, Text: r.Preview})
		}
	case "web_search":
		b.Searches = it.Patterns
		for _, r := range it.Results {
			title := r.Title
			if title == "" {
				title = r.URL
			}
			if title == "" {
				title = "Search result"
			}
			b.Hits = append(b.Hits, ui.SearchHit{Title: title, Text: r.Snippet, Web: true, URL: webAddress(r.URL)})
		}
	case "approval_request":
		b.Value = it.Prompt
	case "user_input_request":
		var qs []string
		for _, q := range it.Questions {
			lines := []string{q.Question}
			for _, o := range q.Options {
				lines = append(lines, "• "+o.Label)
			}
			qs = append(qs, strings.Join(lines, "\n"))
		}
		b.Value = strings.Join(qs, "\n\n")
	case "notification":
		b.Value = it.Detail
	case "system_notice":
		b.Value = it.Message
	case "error":
		if it.Failure != nil {
			b.Value = it.Failure.Message
		}
	case "todo_list":
		var lines []string
		for _, s := range it.Steps {
			mark := "○"
			switch s.Status {
			case "completed":
				mark = "✓"
			case "running":
				mark = "◉"
			}
			text := s.Text
			if text == "" {
				text = s.Step
			}
			lines = append(lines, mark+" "+text)
		}
		b.Value = strings.Join(lines, "\n")
	}
	return b
}

// fillOutput puts a tool's output in its body: from the item, or from the
// full item fetched for it (FetchedToolOutput).
func fillOutput(b *ui.EntryBody, it, full *t3.TurnItem, fetchErr string) {
	src := it
	if needsDetailFetch(it) {
		if full == nil {
			b.Pending, b.Err = fetchErr == "", fetchErr
			return
		}
		src = full
	}
	b.Output = outputText(src)
	b.Missing = b.Output == "" && needsDetailFetch(it)
}

// outputText is T3's turnItemOutputText.
func outputText(it *t3.TurnItem) string {
	switch it.Type {
	case "command_execution":
		return commandOutputText(rawString(it.Output))
	case "dynamic_tool":
		return toolValue(decode(it.Output))
	}
	return ""
}

// commandOutputText unwraps Claude's {"stdout", "stderr", "interrupted"}
// output.
func commandOutputText(out string) string {
	if strings.TrimSpace(out) == "" {
		return ""
	}
	if !strings.HasPrefix(strings.TrimLeft(out, " \t\r\n"), `{"stdout"`) {
		return out
	}
	var o struct {
		Stdout      *string `json:"stdout"`
		Stderr      *string `json:"stderr"`
		Interrupted *bool   `json:"interrupted"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(out)), &o) != nil || o.Stdout == nil || o.Stderr == nil || o.Interrupted == nil {
		return out
	}
	var parts []string
	for _, p := range []string{*o.Stdout, *o.Stderr} {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "\n")
}

// toolArgs is T3's toolCallLines for a tool's arguments (itemDetail.ts:
// 101-130): an object's fields in the call's own order, a string as it is
// and anything else as compact JSON; an input that isn't an object, or
// was summarized, as text. An "html" value is HTML, a JavaScript REPL's
// "code" TypeScript, and JSON JSON (V2ItemInspector.tsx:197-217).
func toolArgs(raw json.RawMessage, tool string) (fields []ui.ToolArg, text string) {
	args := decode(raw)
	if _, ok := args.(map[string]any); !ok || summarized(args) {
		return nil, toolValue(args)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, toolValue(args)
	}
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		key, _ := t.(string)
		var v json.RawMessage
		if dec.Decode(&v) != nil {
			break
		}
		var s string
		if json.Unmarshal(v, &s) != nil || s == "" {
			var b bytes.Buffer
			if json.Compact(&b, v) != nil {
				b.Reset()
				b.Write(v)
			}
			s = b.String()
		}
		lang := ""
		switch {
		case key == "html":
			lang = "html"
		case key == "code" && isJavaScriptRepl(tool):
			lang = "typescript"
		case ui.JSONText(s):
			lang = "json"
		}
		fields = append(fields, ui.ToolArg{Key: key, Value: s, Lang: lang})
	}
	return fields, ""
}

// webAddress is T3's resolveExternalWebLinkHref: a result's address when
// it's a web page's, "//" taken as https.
func webAddress(s string) string {
	if strings.HasPrefix(s, "//") {
		s = "https:" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return u.String()
}

// toolValue is T3's formatToolValue: text blocks as their text, anything
// else as indented JSON.
func toolValue(v any) string {
	if v == nil {
		return ""
	}
	if t, ok := textBlocks(v, 0); ok {
		if strings.TrimSpace(t) == "" {
			return ""
		}
		return prettyJSONText(t)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil || string(b) == "{}" || string(b) == "[]" {
		return ""
	}
	return string(b)
}

// textBlocks is T3's textFromBlocks: MCP content blocks as text.
func textBlocks(v any, depth int) (string, bool) {
	if depth > 4 {
		return "", false
	}
	switch v := v.(type) {
	case string:
		return v, true
	case []any:
		var parts []string
		for _, x := range v {
			t, ok := textBlocks(x, depth+1)
			if !ok {
				return "", false
			}
			if t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n"), true
	case map[string]any:
		switch v["type"] {
		case "text":
			if t, ok := v["text"].(string); ok {
				return t, true
			}
		case "image":
			return "[image]", true
		case "resource_link":
			if u, ok := v["uri"].(string); ok {
				return u, true
			}
		}
		keys := 0
		for k := range v {
			if k != "isError" && k != "is_error" {
				keys++
			}
		}
		if c, ok := v["content"]; ok && keys == 1 {
			return textBlocks(c, depth+1)
		}
	}
	return "", false
}

// prettyJSONText indents text that is JSON, or JSON lines.
func prettyJSONText(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "{") && !strings.HasPrefix(t, "[") {
		return s
	}
	var whole any
	if json.Unmarshal([]byte(t), &whole) == nil {
		b, _ := json.MarshalIndent(whole, "", "  ")
		return string(b)
	}
	var docs []string
	for _, line := range strings.Split(t, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var d any
		if json.Unmarshal([]byte(line), &d) != nil {
			return s
		}
		b, _ := json.MarshalIndent(d, "", "  ")
		docs = append(docs, string(b))
	}
	return strings.Join(docs, "\n\n")
}

// questionPreview, answerPreview and questionHistory are T3's question
// and answer helpers (work-log/userInput.ts).
func questionPreview(q *t3.QuestionAnswer) string {
	var out []string
	for _, id := range questionIDs(q) {
		if t := oneLine(q.QuestionTextByID[id]); t != "" {
			out = append(out, t)
		}
	}
	return strings.Join(out, " · ")
}

func answerPreview(q *t3.QuestionAnswer) string {
	var out []string
	for _, id := range questionIDs(q) {
		if t := answerText(decode(q.Answers[id])); t != "" {
			out = append(out, t)
		}
	}
	return oneLine(strings.Join(out, " · "))
}

func questionHistory(q *t3.QuestionAnswer) []ui.QuestionHistory {
	var out []ui.QuestionHistory
	for _, id := range questionIDs(q) {
		out = append(out, ui.QuestionHistory{Question: q.QuestionTextByID[id], Answer: answerText(decode(q.Answers[id]))})
	}
	return out
}

func questionIDs(q *t3.QuestionAnswer) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, m := range []map[string]string{q.QuestionTextByID} {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			add(k)
		}
	}
	keys := make([]string, 0, len(q.Answers))
	for k := range q.Answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		add(k)
	}
	return ids
}

// answerText is T3's getQuestionAnswerText.
func answerText(v any) string {
	switch v := v.(type) {
	case string:
		return v
	case []any:
		var out []string
		for _, x := range v {
			if t := answerText(x); t != "" {
				out = append(out, t)
			}
		}
		return strings.Join(out, ", ")
	case map[string]any:
		return answerText(v["answers"])
	}
	return ""
}
