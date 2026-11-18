package core

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The timeline's rows, derived from the open thread's turn items with
// T3's rules: deriveTimelineEntriesFromVisibleTurnItems
// (apps/web/src/session-logic.ts:490-690) turns items into entries, and
// deriveMessagesTimelineRows (MessagesTimeline.logic.ts:1524-2323) turns
// entries into rows (docs/ui/06-timeline.md §5.2). T3's run attempts,
// HTML renders, MCP apps, worktree setup and subagent groups have no
// rows here yet.

// entryKind is a timeline entry's kind.
type entryKind int

const (
	entMessage entryKind = iota
	entPlan
	entWork
	entEvent
)

// tlEntry is T3's TimelineEntry.
type tlEntry struct {
	id   string
	kind entryKind
	at   time.Time
	msg  *chatMessage
	work *workEntry
	item *t3.TurnItem
}

// chatMessage is T3's ChatMessage.
type chatMessage struct {
	id, role, text, runID string
	streaming             bool
	at, updated           time.Time
	intent                string // inputIntent
	createdBy, source     string // createdBy, creationSource
	idleCompaction        string
	attachments           []json.RawMessage
	item                  *t3.TurnItem
}

func (m *chatMessage) steer() bool {
	return m.intent == "steer" || m.intent == "promoted_queued_to_steer"
}

// own is a message the user sent, not an agent or T3 (resolveUserMessagePresentation).
func (m *chatMessage) own() bool { return m.role == "user" && m.createdBy != "agent" }

// standalone item types are events, not work (session-logic.ts:232-240).
var standalone = map[string]bool{"fork": true, "handoff": true, "run_interrupt_request": true, "run_interrupt_result": true,
	"secret_request": true, "link_request": true, "subagent": true}

// attachmentOnlyPrompt is what T3 sends the agent for a message of only
// attachments; people see no words.
const attachmentOnlyPrompt = "[User attached one or more files without additional text. Respond using the conversation context and the attached files.]"

// attachmentOnly is T3's isAttachmentOnlyBootstrapPrompt: that prompt,
// which Claude's ultrathink effort sends as "Ultrathink:\n<prompt>".
func attachmentOnly(text string) bool {
	return ultrathinkPrefix.ReplaceAllString(text, "") == attachmentOnlyPrompt
}

var ultrathinkPrefix = regexp.MustCompile(`(?i)^Ultrathink:\s*`)

// timelineEntries is T3's deriveTimelineEntriesFromVisibleTurnItems.
func timelineEntries(items []*t3.TurnItem) []*tlEntry {
	var out []*tlEntry
	for _, it := range items {
		switch {
		case it.Type == "checkpoint",
			it.Type == "user_message" && it.InputIntent == "queued_turn" && it.Status == "pending":
			continue
		}
		at := itemAt(it)
		switch {
		case it.Type == "user_message" || it.Type == "assistant_message":
			m := &chatMessage{id: it.MessageID, role: "user", text: it.Text, runID: it.RunID, at: at, updated: parseTime(it.UpdatedAt),
				intent: it.InputIntent, createdBy: it.CreatedBy, source: it.CreationSource, idleCompaction: it.IdleCompaction,
				attachments: it.Attachments, item: it}
			if it.Type == "assistant_message" {
				m.role, m.streaming = "assistant", it.Streaming
			}
			if m.id == "" {
				m.id = it.ID
			}
			out = append(out, &tlEntry{id: m.id, kind: entMessage, at: at, msg: m, item: it})
		case it.Type == "proposed_plan":
			out = append(out, &tlEntry{id: it.ID, kind: entPlan, at: at, item: it})
		case standalone[it.Type]:
			out = append(out, &tlEntry{id: it.ID, kind: entEvent, at: at, item: it})
		default:
			out = append(out, &tlEntry{id: it.ID, kind: entWork, at: at, work: newWorkEntry(it), item: it})
		}
	}
	return out
}

// entry predicates, after MessagesTimeline.logic.ts.

func (en *tlEntry) runID() string {
	switch en.kind {
	case entMessage:
		if en.msg.role == "assistant" {
			return en.msg.runID
		}
		return ""
	case entPlan, entWork:
		return en.item.RunID
	}
	return ""
}

func (en *tlEntry) isUser() bool { return en.kind == entMessage && en.msg.role == "user" }

func (en *tlEntry) isSteer() bool { return en.isUser() && en.msg.steer() }

// startsResponse: a prompt, or a notification, begins a response.
func (en *tlEntry) startsResponse() bool {
	return en.isUser() && !en.msg.steer() || en.kind == entWork && en.work.itemType == "notification"
}

func (en *tlEntry) isCompaction() bool {
	return en.kind == entWork && en.work.source == "context-compaction"
}

func (en *tlEntry) end() time.Time {
	if en.kind == entMessage {
		return en.msg.updated
	}
	return en.at
}

// failedItem is a top-level error item that failed.
func (en *tlEntry) failedItem() *t3.TurnItem {
	it := en.item
	if en.kind != entWork && en.kind != entEvent || it == nil || it.Type != "error" || it.Status != "failed" || it.ParentItemID != "" {
		return nil
	}
	return it
}

// boundaryRun is a response boundary's run.
func (en *tlEntry) boundaryRun() string {
	switch en.kind {
	case entMessage:
		return en.msg.runID
	case entWork:
		return en.work.runID
	}
	return ""
}

func lastBoundary(entries []*tlEntry) int {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].startsResponse() {
			return i
		}
	}
	return -1
}

// timelineInput is what the rows depend on besides the entries.
type timelineInput struct {
	working       bool    // the thread is working
	latest        *t3.Run // its latest run
	turnEnded     bool    // the latest run's provider turn has ended
	running       string  // the running run
	pending       string  // the pending approval's request
	openRuns      map[string]bool
	openGroups    map[string]bool
	activeStarted time.Time
}

// row is one derived row, T3's MessagesTimelineRow.
type row struct {
	kind      string
	id        string
	at        time.Time
	runID     string
	entry     *tlEntry
	entries   []*workEntry
	expanded  bool // an open group's list (isExpandedToolGroup), or an open header
	label     string
	groupID   string
	active    bool
	live      *workEntry
	thought   *workEntry
	summary   string
	icon      string
	failed    bool
	meta      bool
	copy      bool
	streaming bool
	continues bool
	next      bool
	prev      bool
	steerCont bool
	compact   bool
	title     string
	before    string
	after     string
	failure   string
}

const liveActivityRow = "live-activity-row"

func workGroupID(entryID string) string { return "work-group:" + entryID }

func expandedGroupRow(group string, at time.Time, entries []*workEntry) *row {
	return &row{kind: "work", id: group + ":details", at: at, entries: entries, expanded: true}
}

// terminalAssistants are the last assistant message of each response.
func terminalAssistants(entries []*tlEntry) map[string]bool {
	last := map[string]string{}
	n := 0
	for _, en := range entries {
		if en.kind != entMessage {
			continue
		}
		if en.msg.role == "user" {
			n++
			continue
		}
		key := "turn:" + en.msg.runID
		if en.msg.runID == "" {
			key = "unkeyed:" + strconv.Itoa(n)
		}
		last[key] = en.msg.id
	}
	out := map[string]bool{}
	for _, id := range last {
		out[id] = true
	}
	return out
}

// unsettledRun is T3's deriveUnsettledRunId: the run the thread is still
// answering.
func unsettledRun(latest *t3.Run, running string) string {
	if running != "" {
		return running
	}
	if latest == nil {
		return ""
	}
	settled := latest.CompletedAt != "" && latest.Status != "running" && latest.Status != "starting" && latest.Status != "waiting"
	if settled {
		return ""
	}
	return latest.ID
}

// activeResponseRuns is T3's deriveActiveVisualResponseRunIds.
func activeResponseRuns(entries []*tlEntry, unsettled string, working bool) map[string]bool {
	runs := map[string]bool{}
	if unsettled == "" {
		return runs
	}
	runs[unsettled] = true
	if !working {
		return runs
	}
	for _, en := range entries[lastBoundary(entries)+1:] {
		if r := en.runID(); r != "" {
			runs[r] = true
		}
	}
	return runs
}

// failedRuns are runs that failed: the latest, or any with a failed error.
func failedRuns(entries []*tlEntry, latest *t3.Run) map[string]bool {
	out := map[string]bool{}
	if latest != nil && latest.Status == "failed" {
		out[latest.ID] = true
	}
	for _, en := range entries {
		if it := en.failedItem(); it != nil && it.RunID != "" {
			out[it.RunID] = true
		}
	}
	return out
}

// itemizedRuns are failed runs whose work shows item by item: all but a
// usage-limit pause (itemizedFailedRunIds).
func itemizedRuns(entries []*tlEntry, failed map[string]bool) map[string]bool {
	usage, other := map[string]bool{}, map[string]bool{}
	for _, en := range entries {
		if it := en.failedItem(); it != nil && it.RunID != "" {
			if it.Failure != nil && it.Failure.Class == "usage_limit" {
				usage[it.RunID] = true
			} else {
				other[it.RunID] = true
			}
		}
	}
	out := map[string]bool{}
	for r := range failed {
		if !usage[r] || other[r] {
			out[r] = true
		}
	}
	return out
}

// liveSubagents are subagent cards kept visible while one of their batch
// runs.
func liveSubagents(entries []*tlEntry) map[string]bool {
	out := map[string]bool{}
	var batch []*tlEntry
	flush := func() {
		live := false
		for _, en := range batch {
			live = live || activeStatusItem(en.item.Status)
		}
		for _, en := range batch {
			if live {
				out[en.id] = true
			}
		}
		batch = nil
	}
	for _, en := range entries {
		if en.kind != entEvent || en.item.Type != "subagent" {
			flush()
			continue
		}
		if n := len(batch); n > 0 && (batch[n-1].item.RunID != en.item.RunID || batch[n-1].item.ProviderTurnID != en.item.ProviderTurnID) {
			flush()
		}
		batch = append(batch, en)
	}
	flush()
	return out
}

func activeStatusItem(s string) bool { return s == "pending" || s == "running" || s == "waiting" }

// retainedReplies are answers kept visible: final answers.
func retainedReplies(entries []*tlEntry) map[string]bool {
	out := map[string]bool{}
	for _, en := range entries {
		if en.kind == entMessage && en.item != nil && en.item.Type == "assistant_message" && en.item.Phase == "final_answer" {
			out[en.id] = true
		}
	}
	return out
}

// foldRunID is the run an entry folds with, or "".
func foldRunID(en *tlEntry, runless string) string {
	or := func(r string) string {
		if r != "" {
			return r
		}
		return runless
	}
	switch {
	case en.kind == entWork && en.work.itemType == "system_notice":
		return ""
	case en.kind == entMessage && en.msg.role == "assistant":
		return or(en.msg.runID)
	case en.kind == entWork:
		return or(en.work.runID)
	case en.kind == entEvent && (en.item.Type == "fork" || en.item.Type == "thread_created" || en.item.Type == "secret_request" ||
		en.item.Type == "link_request" || en.item.Type == "subagent"):
		return or(en.item.RunID)
	}
	return ""
}

// persistentCard is a resource card a fold never hides.
func persistentCard(en *tlEntry) bool {
	return en.kind == entEvent && (en.item.Type == "fork" || en.item.Type == "thread_created" || en.item.Type == "secret_request" || en.item.Type == "link_request")
}

// fold is T3's TurnFold.
type fold struct {
	runID   string
	anchor  string
	at      time.Time
	hidden  map[string]bool
	members map[string]bool
	label   string
}

// turnFolds is T3's deriveTurnFolds: each settled response folds its work
// behind "Worked for <duration>" (§5.2.2).
func turnFolds(entries []*tlEntry, terminal map[string]bool, latest *t3.Run, unfolded, live, retained map[string]bool) map[string]*fold {
	interrupted := map[string]bool{}
	promptAt := map[string]int{}
	for i, en := range entries {
		if en.startsResponse() {
			if r := en.boundaryRun(); r != "" {
				if _, ok := promptAt[r]; !ok {
					promptAt[r] = i
				}
			}
		}
		if en.kind == entEvent && en.item.RunID != "" && (en.item.Type == "run_interrupt_request" || en.item.Type == "run_interrupt_result") {
			interrupted[en.item.RunID] = true
		}
	}
	type segment struct {
		start int
		steer *tlEntry
	}
	type group struct {
		entries   []*tlEntry
		terminal  *tlEntry
		streaming bool
		start     time.Time
		anchor    string
		steers    []segment
	}
	groups := map[string]*group{}
	var order []string
	runlessFailed := map[string]bool{}
	var runless, response string
	var pendingAt time.Time
	pendingAnchor := ""
	for i, en := range entries {
		if en.startsResponse() {
			pendingAnchor, pendingAt = "", time.Time{}
			if i+1 < len(entries) {
				pendingAnchor, pendingAt = entries[i+1].id, en.at
			}
			runless = ""
			if en.boundaryRun() == "" {
				runless = "runless:" + en.id
			}
			response = en.boundaryRun()
			if response == "" {
				response = runless
			}
			continue
		}
		if en.isSteer() {
			r := en.msg.runID
			if r == "" {
				r = response
			}
			if g := groups[r]; g != nil && len(g.entries) > 0 {
				// Back-to-back steers split once, after the last of them.
				if n := len(g.steers); n > 0 && g.steers[n-1].start == len(g.entries) {
					g.steers = g.steers[:n-1]
				}
				g.steers = append(g.steers, segment{len(g.entries), en})
			}
			continue
		}
		own := foldRunID(en, runless)
		if own == "" {
			continue
		}
		run := own
		// Resumed subagents keep their place and fold with their turn.
		if at, ok := promptAt[own]; response != "" && ok && at > i {
			run = response
		}
		if run == runless && en.failedItem() != nil {
			runlessFailed[run] = true
		}
		g := groups[run]
		if g == nil {
			g = &group{start: pendingAt, anchor: pendingAnchor}
			if g.anchor == "" {
				g.anchor = en.id
			}
			pendingAnchor, pendingAt = "", time.Time{}
			groups[run] = g
			order = append(order, run)
		}
		g.entries = append(g.entries, en)
		if en.kind == entMessage {
			if terminal[en.msg.id] {
				g.terminal = en
			}
			g.streaming = g.streaming || en.msg.streaming
		}
	}
	out := map[string]*fold{}
	for _, run := range order {
		g := groups[run]
		if unfolded[run] || interrupted[run] || runlessFailed[run] || g.streaming {
			continue
		}
		hidden := map[string]bool{}
		terminalAt := len(g.entries)
		for i, en := range g.entries {
			if en == g.terminal {
				terminalAt = i
			}
		}
		// Folded dividers lose their place, so only the latest stays.
		visibleCompaction := ""
		for _, en := range g.entries {
			if en.isCompaction() {
				visibleCompaction = en.id
			}
		}
		for i, en := range g.entries {
			if en == g.terminal {
				continue
			}
			trailing := en.kind == entWork && en.work.status != "inProgress" && !en.work.displayFailed()
			if !en.isCompaction() && i > terminalAt && !trailing {
				continue
			}
			if en.id == visibleCompaction || persistentCard(en) || live[en.id] || retained[en.id] {
				continue
			}
			if en.kind == entWork && (en.work.itemType == "notification" || en.work.answer != nil) {
				continue
			}
			hidden[en.id] = true
		}
		hidesWork := func(es []*tlEntry) bool {
			for _, en := range es {
				if hidden[en.id] && !en.isCompaction() {
					return true
				}
			}
			return false
		}
		if !hidesWork(g.entries) {
			continue
		}
		firstEntry, lastEntry := g.entries[0], g.entries[len(g.entries)-1]
		foldAt := g.start
		if foldAt.IsZero() {
			foldAt = firstEntry.at
		}
		start, end := foldAt, lastEntry.end()
		if g.terminal != nil && g.terminal.msg.updated.After(end) {
			end = g.terminal.msg.updated
		}
		var elapsed time.Duration
		known := !start.IsZero() && !end.IsZero()
		if latest != nil && latest.ID == run && latest.StartedAt != "" && latest.CompletedAt != "" {
			start, end, known = parseTime(latest.StartedAt), parseTime(latest.CompletedAt), true
		}
		if known {
			elapsed = max(0, end.Sub(start))
		}
		label := "Worked"
		if known {
			label = "Worked for " + ui.FormatDuration(elapsed)
		}
		if latest != nil && latest.ID == run && latest.Status == "interrupted" {
			label = "You stopped this response"
			if known {
				label = "You stopped after " + ui.FormatDuration(elapsed)
			}
		}
		members := func(es []*tlEntry) (hid, all map[string]bool) {
			hid, all = map[string]bool{}, map[string]bool{}
			for _, en := range es {
				all[en.id] = true
				if hidden[en.id] {
					hid[en.id] = true
				}
			}
			return hid, all
		}
		firstEnd := len(g.entries)
		if len(g.steers) > 0 {
			firstEnd = g.steers[0].start
		}
		hid, all := members(g.entries[:firstEnd])
		out[g.anchor] = &fold{runID: run, anchor: g.anchor, at: foldAt, hidden: hid, members: all, label: label}
		// Work after a steer folds under its own row below that steer.
		for i, s := range g.steers {
			stop := len(g.entries)
			if i+1 < len(g.steers) {
				stop = g.steers[i+1].start
			}
			seg := g.entries[s.start:stop]
			if len(seg) == 0 || !hidesWork(seg) {
				continue
			}
			segEnd := s.steer.at
			for _, en := range seg {
				if en.end().After(segEnd) {
					segEnd = en.end()
				}
			}
			label := "Continued"
			if !s.steer.at.IsZero() {
				label = "Continued for " + ui.FormatDuration(segEnd.Sub(s.steer.at))
			}
			hid, all := members(seg)
			out[seg[0].id] = &fold{runID: run + ":steer:" + s.steer.id, anchor: seg[0].id, at: s.steer.at, hidden: hid, members: all, label: label}
		}
	}
	return out
}

// deriveRows is T3's deriveMessagesTimelineRows.
func deriveRows(all []*tlEntry, in timelineInput) []*row {
	entries := settleReasoning(all)
	pending := func(e *workEntry) bool {
		return e.item != nil && e.item.Type == "approval_request" && in.pending != "" && e.item.RequestID == in.pending
	}
	unsettled := unsettledRun(in.latest, in.running)
	terminal := terminalAssistants(entries)
	failed := failedRuns(entries, in.latest)
	itemized := itemizedRuns(entries, failed)
	live := liveSubagents(entries)
	retained := retainedReplies(entries)
	activeRuns := activeResponseRuns(entries, unsettled, in.working)
	unfolded := map[string]bool{}
	for r := range activeRuns {
		unfolded[r] = true
	}
	for r := range failed {
		unfolded[r] = true
	}
	folds := turnFolds(entries, terminal, in.latest, unfolded, live, retained)
	collapsed := map[string]bool{}
	for _, f := range folds {
		if !in.openRuns[f.runID] {
			for id := range f.hidden {
				collapsed[id] = true
			}
		}
	}
	for _, en := range entries {
		if en.kind == entWork && pending(en.work) {
			delete(collapsed, en.id)
		}
	}
	inActiveRun := func(e *workEntry) bool {
		return in.working && e.status == "inProgress" && e.runID != "" && e.runID == unsettled
	}
	header := len(entries)
	if in.working {
		header = lastBoundary(entries) + 1
	}

	// The live tail: the active response's trailing work, as one row.
	var tail []*tlEntry
	if in.working && unsettled != "" {
		for i := len(entries) - 1; i >= header; i-- {
			en := entries[i]
			if en.kind != entWork {
				break
			}
			e := en.work
			if e.tone == "error" || e.source == "runtime.error" || e.itemType == "system_notice" || e.itemType == "notification" ||
				e.itemType == "todo_list" || e.answer != nil || pending(e) || e.runID == "" || !activeRuns[e.runID] ||
				e.source == "context-compaction" || collapsed[en.id] || folds[en.id] != nil {
				break
			}
			tail = append([]*tlEntry{en}, tail...)
		}
	}
	var visibleTail []*workEntry
	for _, en := range tail {
		if en.work.visibleInGroup(true) {
			visibleTail = append(visibleTail, en.work)
		}
	}
	var latestVisible, latestRunning, latestThought *workEntry
	if n := len(visibleTail); n > 0 {
		latestVisible = visibleTail[n-1]
	}
	for _, e := range visibleTail {
		if e.activeTurnActivity() {
			latestRunning = e
		}
		if e.itemType == "reasoning" && strings.TrimSpace(e.detail) != "" {
			latestThought = e
		}
	}
	keepsLive := latestRunning != nil || latestVisible != nil && latestVisible.succeeded() && in.pending == ""
	latestFailed := latestRunning == nil && latestVisible != nil && latestVisible.status != "declined" && latestVisible.displayFailed()
	var liveRow *row
	if len(tail) > 0 && latestVisible != nil && !latestFailed {
		anchor := tail[0]
		g := workGroupID(anchor.id)
		id := "work-live:" + anchor.id
		if keepsLive {
			id = liveActivityRow
		}
		shown := latestRunning
		if shown == nil {
			shown = latestVisible
		}
		liveRow = &row{kind: "work-live", id: id, at: anchor.at, live: shown, entries: visibleTail, groupID: g,
			expanded: in.openGroups[g], active: keepsLive, thought: latestThought}
	}
	tailIDs := map[string]bool{}
	if liveRow != nil || latestFailed {
		for _, en := range tail {
			tailIDs[en.id] = true
		}
	}
	placeLive := ""
	if latestVisible != nil {
		for _, en := range tail {
			if en.work == latestVisible {
				placeLive = en.id
			}
		}
	}

	// A bare /compact turn is the compaction itself: its divider stands in
	// for the prompt and the working row.
	compactionRuns := map[string]string{} // run → idle reason, "" for a plain /compact
	compacted := map[string]bool{}
	for _, en := range entries {
		switch {
		case en.isUser() && en.msg.runID != "" && compactionCommand(en.msg):
			compactionRuns[en.msg.runID] = idleCompactionOf(en.msg)
		case en.isCompaction() && en.work.runID != "":
			compacted[en.work.runID] = true
		}
	}
	var activeCompaction *tlEntry
	for _, en := range entries {
		if en.isCompaction() && inActiveRun(en.work) && activeCompaction == nil {
			activeCompaction = en
		}
	}
	_, compactionRun := compactionRuns[unsettled]
	activeIsCompaction := in.working && unsettled != "" && compactionRun
	compactionFinished := activeIsCompaction && compacted[unsettled] && activeCompaction == nil
	compactionOnlyFinished := compactionFinished
	if compactionFinished {
		for _, en := range entries {
			if en.runID() == unsettled && !en.isCompaction() && !(en.isUser() && compactionCommand(en.msg)) {
				compactionOnlyFinished = false
			}
		}
	}
	runningCompaction := ""
	if activeIsCompaction && !compactionFinished {
		runningCompaction = unsettled
	}
	// Compacting on return holds back the message just sent, so the
	// working row says why it waits; any other compaction shows only its
	// divider.
	compactsOnReturn := runningCompaction != "" && compactionRuns[runningCompaction] == "onReturn"
	hasActiveCompaction := activeCompaction != nil || activeIsCompaction && !compactionFinished

	var rows []*row
	working := func() {
		if compactionOnlyFinished || runningCompaction != "" && !compactsOnReturn {
			return
		}
		rows = append(rows, &row{kind: "working", id: "working-indicator-row", at: in.activeStarted, compact: compactsOnReturn})
	}
	hasActivity := false
	liveRows := func() {
		if liveRow == nil {
			return
		}
		rows = append(rows, liveRow)
		hasActivity = hasActivity || liveRow.active
		if liveRow.expanded {
			rows = append(rows, expandedGroupRow(liveRow.groupID, liveRow.at, liveRow.entries))
		}
	}
	for i := 0; i < len(entries); i++ {
		en := entries[i]
		if in.working && i == header {
			working()
		}
		if en.id == placeLive {
			liveRows()
		}
		// A stopped compaction's own divider already says it stopped.
		if en.kind == entEvent && (en.item.Type == "run_interrupt_request" ||
			en.item.Type == "run_interrupt_result" && en.item.RunID != "" && compactionRunOf(compactionRuns, en.item.RunID) && compacted[en.item.RunID]) {
			continue
		}
		if f := folds[en.id]; f != nil {
			rows = append(rows, &row{kind: "turn-fold", id: "turn-fold:" + f.runID, at: f.at, runID: f.runID, label: f.label, expanded: in.openRuns[f.runID]})
		}
		if collapsed[en.id] || tailIDs[en.id] {
			continue
		}
		switch en.kind {
		case entWork:
			e := en.work
			if e.source == "context-compaction" {
				active := inActiveRun(e)
				reason, isRun := compactionRuns[e.runID]
				r := &row{kind: "context-compaction", id: en.id, at: en.at, label: e.label, active: active}
				if isRun && reason != "" {
					r.title = idleCompactionTitles[reason]
					if active {
						r.label = idleCompactingLabels[reason]
					}
				}
				if b, a, ok := compactionTokens(e.item); ok {
					r.before, r.after = b, a
				}
				if e.item.Status == "failed" {
					r.failed, r.failure = true, strings.TrimSpace(e.item.Summary)
				}
				rows = append(rows, r)
				continue
			}
			if pending(e) || e.runID != "" && itemized[e.runID] || e.itemType == "error" && e.status == "failed" || e.tone == "error" ||
				e.source == "runtime.error" || e.itemType == "system_notice" || e.itemType == "notification" || e.itemType == "todo_list" || e.answer != nil {
				rows = append(rows, &row{kind: "work", id: en.id, at: en.at, entries: []*workEntry{e}})
				continue
			}
			grouped := []*workEntry{e}
			j := i + 1
			for ; j < len(entries); j++ {
				nx := entries[j]
				if nx.kind != entWork {
					break
				}
				n := nx.work
				if n.source == "context-compaction" || n.itemType == "error" && n.status == "failed" || n.tone == "error" ||
					n.source == "runtime.error" || n.itemType == "system_notice" || n.itemType == "notification" || n.itemType == "todo_list" ||
					n.answer != nil || pending(n) || tailIDs[nx.id] || collapsed[nx.id] || folds[nx.id] != nil || n.runID != e.runID {
					break
				}
				grouped = append(grouped, n)
			}
			var visible []*workEntry
			for _, g := range grouped {
				if g.visibleInGroup(inActiveRun(g)) {
					visible = append(visible, g)
				}
			}
			i = j - 1
			if len(visible) == 0 {
				continue
			}
			g := workGroupID(en.id)
			var running []*workEntry
			for _, v := range visible {
				if inActiveRun(v) {
					running = append(running, v)
				}
			}
			switch {
			case len(running) > 0:
				rows = append(rows, &row{kind: "work-live", id: "work-live:" + en.id, at: en.at, live: running[len(running)-1],
					entries: visible, groupID: g, expanded: in.openGroups[g], active: true})
				hasActivity = true
				if in.openGroups[g] {
					rows = append(rows, expandedGroupRow(g, en.at, visible))
				}
			case len(visible) == 1 && visible[0].toolLike():
				label := visible[0].singleLabel()
				if visible[0].groupAction() == "edit" {
					label, _ = summarizeGroup(visible)
				}
				rows = append(rows, &row{kind: "work", id: en.id, at: en.at, entries: visible, label: label})
			default:
				summary, _ := summarizeGroup(visible)
				if len(visible) == 1 && !visible[0].toolLike() {
					summary = visible[0].label
				}
				var latestTool *workEntry
				surface := ""
				for _, v := range visible {
					if v.toolLike() {
						latestTool = v
					}
					if v.surface != "" {
						surface = v.surface
					}
				}
				icon := summaryIcon(groupSummaryKind(visible))
				if surface != "" {
					icon = surface
				}
				rows = append(rows, &row{kind: "work-toggle", id: "work-toggle:" + en.id, at: en.at, runID: e.runID, groupID: g,
					expanded: in.openGroups[g], summary: summary, icon: icon, failed: latestTool != nil && latestTool.displayFailed()})
				if in.openGroups[g] {
					rows = append(rows, expandedGroupRow(g, en.at, visible))
				}
			}
		case entPlan:
			rows = append(rows, &row{kind: "proposed-plan", id: en.id, at: en.at, entry: en})
		case entEvent:
			rows = append(rows, &row{kind: "event", id: en.id, at: en.at, entry: en})
		case entMessage:
			m := en.msg
			if m.role == "user" && m.runID != "" && compactionCommand(m) {
				if !compacted[m.runID] && !(compactsOnReturn && m.runID == runningCompaction) {
					reason := compactionRuns[m.runID]
					label := "Context compaction"
					if m.runID == runningCompaction {
						label = "Compacting context"
						if reason != "" {
							label = idleCompactingLabels[reason]
						}
					}
					rows = append(rows, &row{kind: "context-compaction", id: "compaction:" + m.runID, at: en.at, label: label, active: m.runID == runningCompaction})
				}
				continue
			}
			inProgress := m.role == "assistant" && m.runID != "" && activeRuns[m.runID]
			meta := m.role == "assistant" && terminal[m.id] && !inProgress
			rows = append(rows, &row{kind: "message", id: en.id, at: en.at, entry: en, meta: meta, copy: meta, streaming: m.streaming || inProgress})
		}
	}
	hasWorking := false
	for _, r := range rows {
		hasWorking = hasWorking || r.kind == "working"
	}
	if in.working && !hasWorking && header == len(entries) {
		working()
	}
	if in.working && !in.turnEnded && !hasActiveCompaction && !compactionOnlyFinished && (latestFailed || !hasActivity && in.pending == "") {
		r := &row{kind: "thinking", id: liveActivityRow, at: in.activeStarted}
		if latestFailed {
			r.groupID = workGroupID(tail[0].id)
			r.expanded = in.openGroups[r.groupID]
			rows = append(rows, r)
			if r.expanded {
				rows = append(rows, expandedGroupRow(r.groupID, tail[0].at, visibleTail))
			}
		} else {
			rows = append(rows, r)
		}
	}
	rows = attachTrailingMeta(rows)
	for i, r := range rows {
		if workLogRow(r) && i+1 < len(rows) && workLogRow(rows[i+1]) {
			r.continues = true
		}
		if !ownUserRow(r) {
			continue
		}
		r.prev = i > 0 && ownUserRow(rows[i-1])
		r.next = i+1 < len(rows) && ownUserRow(rows[i+1])
		r.steerCont = r.prev && r.entry.msg.steer() && rows[i-1].entry.msg.steer()
	}
	return rows
}

var (
	idleCompactingLabels = map[string]string{"beforeCacheExpires": "Compacting before the cache expires…", "onReturn": "Compacting…"}
	idleCompactionTitles = map[string]string{"beforeCacheExpires": "Context automatically compacted before cache expiry", "onReturn": "Context compacted before sending"}
)

// idleCompactionOf is T3's idleCompactionOf: why T3 sent a /compact on
// its own, or "" for one someone asked for. Servers from before
// idleCompaction marked their own as the agent's.
func idleCompactionOf(m *chatMessage) string {
	if m.idleCompaction != "" {
		return m.idleCompaction
	}
	if m.createdBy == "agent" && m.source == "server" {
		return "beforeCacheExpires"
	}
	return ""
}

func compactionRunOf(runs map[string]string, run string) bool {
	_, ok := runs[run]
	return ok
}

// compactionCommand is a user's bare "/compact".
func compactionCommand(m *chatMessage) bool {
	return m.role == "user" && strings.ToLower(strings.TrimSpace(m.text)) == "/compact" && len(m.attachments) == 0
}

// settleReasoning marks in-progress reasoning that newer entries follow
// as done.
func settleReasoning(entries []*tlEntry) []*tlEntry {
	out := make([]*tlEntry, len(entries))
	for i, en := range entries {
		out[i] = en
		if i < len(entries)-1 && en.kind == entWork && en.work.itemType == "reasoning" && en.work.status == "inProgress" {
			w := *en.work
			w.status = "completed"
			cp := *en
			cp.work = &w
			out[i] = &cp
		}
	}
	return out
}

func workLogRow(r *row) bool {
	return r.kind == "work" || r.kind == "work-toggle" || r.kind == "work-live" || r.kind == "thinking"
}

func ownUserRow(r *row) bool {
	return r.kind == "message" && r.entry.msg.own()
}

// attachTrailingMeta is T3's attachTrailingToolGroupsToAssistant: when a
// response ends with tool groups, its copy and time row moves below them.
func attachTrailingMeta(rows []*row) []*row {
	after := map[int]*row{}
	for mi, r := range rows {
		if r.kind != "message" || r.entry.msg.role != "assistant" || !r.meta || r.entry.msg.runID == "" {
			continue
		}
		run := r.entry.msg.runID
		last, tools := -1, false
		for i := mi + 1; i < len(rows); i++ {
			c := rows[i]
			if c.kind == "message" {
				break
			}
			if c.kind == "work-toggle" && c.runID == run {
				tools, last = true, i
				continue
			}
			if c.kind == "work" {
				ours := false
				for _, e := range c.entries {
					ours = ours || e.runID == run
				}
				if !ours {
					continue
				}
				if !c.expanded {
					for _, e := range c.entries {
						tools = tools || e.toolLike() || e.item != nil && e.item.Type == "error" && e.item.Status == "failed"
					}
				}
				if tools {
					last = i
				}
			}
		}
		if last < 0 {
			continue
		}
		r.meta, r.copy = false, false
		after[last] = &row{kind: "assistant-meta", id: "assistant-meta:" + r.entry.msg.id, at: rows[last].at, entry: r.entry, copy: true, streaming: r.streaming}
	}
	if len(after) == 0 {
		return rows
	}
	var out []*row
	for i, r := range rows {
		out = append(out, r)
		if m := after[i]; m != nil {
			out = append(out, m)
		}
	}
	return out
}
