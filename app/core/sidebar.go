package core

import (
	"cmp"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The thread sidebar (docs/ui/04-sidebar.md): which shelf each thread is
// on and in what order, every row's status, its age or run timer, the
// search, the project filter and the multi-selection. The rules are T3's
// own: Sidebar.logic.ts and Sidebar.tsx in apps/web, threadInbox.ts,
// threadSettled.ts and threadSort.ts in client-runtime.

// sidebarStatus is a thread's state as its row shows it
// (resolveSidebarThreadStatus).
type sidebarStatus int

const (
	stReady sidebarStatus = iota
	stApproval
	stInput
	stRequest
	stPreparing
	stWorking
	stWaiting
	stLimited
	stFailed
)

const (
	settledFirst    = 10 // the Settled shelf's first page, then pages of 25
	settledPage     = 25
	workingGrace    = 2 * time.Second // WORKING_EXIT_GRACE_MS
	staleExit       = time.Minute     // STALE_EXIT_MS
	queuedTurnGrace = 2 * time.Minute // QUEUED_TURN_START_GRACE_MS
)

// threadRuntime is the thread's runtime status as T3's client reads it off
// the shell (shellRuntime, models.ts): "" before its first run, "idle"
// while awaited background work holds it, else the activity run's status
// or the latest run's.
func threadRuntime(t *t3.Thread) string {
	switch {
	case holdsCompletion(t.PendingBackgroundTasks) && t.Status != "failed":
		return "idle"
	case t.LatestRunID == "" && t.ActiveProviderThreadID == "":
		return ""
	case t.ActivityRunStatus != "":
		return t.ActivityRunStatus
	}
	return t.Status
}

// holdsCompletion reports background work the run waits on: subagents and
// background tasks, or commands and monitors the agent said it awaits.
func holdsCompletion(tasks []t3.BackgroundTask) bool {
	for _, k := range tasks {
		if k.Awaited || (k.Kind != "command" && k.Kind != "monitor") {
			return true
		}
	}
	return false
}

func runtimeActive(rt string) bool {
	switch rt {
	case "preparing", "queued", "starting", "running", "waiting":
		return true
	}
	return false
}

// latestRunStatus is the latest run's status, "" with no run.
func latestRunStatus(t *t3.Thread) string {
	if t.LatestRunID == "" {
		return ""
	}
	if t.Status == "idle" {
		return "completed"
	}
	return t.Status
}

// latestRunDone is when the latest run ended, zero while it runs.
func latestRunDone(t *t3.Thread) time.Time {
	if t.LatestRunID == "" {
		return time.Time{}
	}
	if t.LatestRunCompletedAt != "" {
		return parseTime(t.LatestRunCompletedAt)
	}
	switch t.Status {
	case "idle", "completed", "interrupted", "failed", "cancelled", "rolled_back":
		return parseTime(t.UpdatedAt)
	}
	return time.Time{}
}

// pendingApproval is a request the user must answer other than a question.
func pendingApproval(t *t3.Thread) bool {
	r := t.PendingRuntimeRequest
	return r != nil && r.Kind != "user_input" && r.Kind != "auth_refresh"
}

func pendingInput(t *t3.Thread) bool {
	return t.PendingRuntimeRequest != nil && t.PendingRuntimeRequest.Kind == "user_input"
}

// listed reports a thread lists get a row for: not a subagent or a side
// chat, which live inside their parent.
func listed(t *t3.Thread) bool {
	return t.Lineage == nil || (t.Lineage.RelationshipToParent != "subagent" && !t.Lineage.SideChat)
}

// childRequests is what each thread's subagents and side chats ask of the
// user, rolled up to it (pendingSubagentRequestsByThread): "approval" or
// "user_input". A subagent's approval climbs its whole chain; a side chat's
// question or approval reaches its parent.
func childRequests(threads map[string]*t3.Thread) map[string]string {
	out := map[string]string{}
	set := func(id, req string) {
		if out[id] != "approval" {
			out[id] = req
		}
	}
	for _, c := range threads {
		r := c.PendingRuntimeRequest
		if c.Lineage == nil || c.Lineage.ParentThreadID == "" || r == nil || r.Kind == "auth_refresh" {
			continue
		}
		parent := c.Lineage.ParentThreadID
		if c.Lineage.SideChat {
			req := "approval"
			if r.Kind == "user_input" {
				req = "user_input"
			}
			set(parent, req)
			continue
		}
		// Provider-native subagents ask on their parent already, and the
		// coordinator answers a child's questions itself.
		if c.Lineage.RelationshipToParent != "subagent" || c.CreationSource == "provider" || r.Kind == "user_input" || !childRunLive(c) {
			continue
		}
		seen := map[string]bool{c.ID: true}
		for id := parent; id != "" && !seen[id]; {
			seen[id] = true
			set(id, "approval")
			p := threads[id]
			if p == nil || p.Lineage == nil || (p.Lineage.RelationshipToParent != "subagent" && !p.Lineage.SideChat) {
				break
			}
			id = p.Lineage.ParentThreadID
		}
	}
	return out
}

func childRunLive(t *t3.Thread) bool {
	if t.ActivityRunStatus != "" {
		return true
	}
	switch t.Status {
	case "preparing", "starting", "running", "waiting":
		return true
	}
	return false
}

// status is the row's state, first match wins.
func status(t *t3.Thread, child string) sidebarStatus {
	rt := threadRuntime(t)
	switch {
	case pendingApproval(t) || child == "approval":
		return stApproval
	case pendingInput(t) || child == "user_input":
		return stInput
	case t.PendingGuestMessageCount > 0:
		return stRequest
	case rt == "running" && t.BlockedOnThreadWait:
		return stWaiting
	case rt == "preparing":
		return stPreparing
	case rt == "queued" || rt == "starting" || rt == "running" || rt == "waiting":
		return stWorking
	case rt == "idle":
		return stWaiting
	case rt == "failed" && t.LastErrorClass == "usage_limit":
		return stLimited
	case rt == "failed":
		return stFailed
	}
	return stReady
}

// unseen reports a run that finished after the last visit
// (hasUnseenCompletion). A thread never visited is not unread.
func unseen(t *t3.Thread) bool {
	done := latestRunDone(t)
	if done.IsZero() || t.LastVisitedAt == "" {
		return false
	}
	v := parseTime(t.LastVisitedAt)
	return v.IsZero() || done.After(v)
}

// questionAt is when the open question was asked, "" with none.
func questionAt(t *t3.Thread) string {
	if pendingInput(t) {
		return t.PendingRuntimeRequest.CreatedAt
	}
	return ""
}

// failedAt is when the thread failed, and whether it has.
func failedAt(t *t3.Thread) (string, bool) {
	if threadRuntime(t) != "failed" {
		return "", false
	}
	if latestRunStatus(t) == "failed" {
		if d := latestRunDone(t); !d.IsZero() {
			return d.Format(time.RFC3339Nano), true
		}
		return "", true
	}
	return t.UpdatedAt, true
}

// raisedHand reports what ends a snooze early (threadRaisedHandWhileSnoozed):
// an approval or other request, a question asked after snoozing, a child's
// request, a waiting guest, or a failure after snoozing.
func raisedHand(t *t3.Thread, child string) bool {
	r := t.PendingRuntimeRequest
	if pendingApproval(t) || (r != nil && r.Kind != "user_input") {
		return true
	}
	if q := questionAt(t); pendingInput(t) && (t.SnoozedAt == "" || q == "" || parseTime(q).After(parseTime(t.SnoozedAt))) {
		return true
	}
	if child != "" || t.PendingGuestMessageCount > 0 {
		return true
	}
	at, failed := failedAt(t)
	return failed && (t.SnoozedAt == "" || (at != "" && parseTime(at).After(parseTime(t.SnoozedAt))))
}

// snoozed reports a thread on the Snoozed shelf (effectiveSnoozed).
func snoozed(t *t3.Thread, child string, now time.Time) bool {
	wake := parseTime(t.SnoozedUntil)
	return !wake.IsZero() && wake.After(now) && !raisedHand(t, child)
}

// wokeAt is when a snoozed thread woke (threadWokeAt): its wake time once
// passed, or the moment it raised its hand. "" if it hasn't.
func wokeAt(t *t3.Thread, child string, now time.Time) string {
	wake := parseTime(t.SnoozedUntil)
	if wake.IsZero() {
		return ""
	}
	if raisedHand(t, child) {
		at, _ := failedAt(t)
		q := questionAt(t)
		since := parseTime(t.SnoozedAt)
		freshFailure := at != "" && (t.SnoozedAt == "" || parseTime(at).After(since))
		freshQuestion := q != "" && t.SnoozedAt != "" && parseTime(q).After(since)
		switch {
		case freshFailure && (!freshQuestion || !parseTime(at).Before(parseTime(q))):
			return at
		case freshQuestion:
			return q
		case threadRuntime(t) != "":
			return t.UpdatedAt
		}
		return t.SnoozedAt
	}
	if !wake.After(now) {
		return t.SnoozedUntil
	}
	return ""
}

// woke reports a wake the user hasn't seen yet.
func woke(t *t3.Thread, child string, now time.Time) bool {
	at := parseTime(wokeAt(t, child, now))
	if at.IsZero() || t.SettledOverride == "settled" {
		return false
	}
	return t.LastVisitedAt == "" || parseTime(t.LastVisitedAt).Before(at)
}

// queuedTurnStart reports a turn about to start (hasQueuedTurnStart): the
// runtime is getting going, or a message from the last two minutes hasn't
// started a run yet.
func queuedTurnStart(t *t3.Thread, now time.Time) bool {
	switch threadRuntime(t) {
	case "preparing", "queued", "starting":
		return true
	}
	msg := parseTime(t.LatestUserMessageAt)
	if msg.IsZero() {
		return false
	}
	if d := now.Sub(msg); d > queuedTurnGrace || d < -queuedTurnGrace {
		return false
	}
	if t.LatestRunID == "" {
		return true
	}
	for _, s := range []string{t.LatestRunRequestedAt, t.LatestRunStartedAt, t.LatestRunCompletedAt} {
		if s != "" && !parseTime(s).Before(msg) {
			return false
		}
	}
	return true
}

// canSnooze: nothing would wake it at once, and no turn is about to start.
func canSnooze(t *t3.Thread, child string, now time.Time) bool {
	r := t.PendingRuntimeRequest
	return !pendingApproval(t) && (r == nil || r.Kind == "user_input") && child == "" &&
		t.PendingGuestMessageCount == 0 && !queuedTurnStart(t, now)
}

// working reports a thread for the Working shelf (isThreadWorking): its
// runtime is busy or idling on background work, and it isn't waiting on
// the user or stopped with a plan to review.
func working(t *t3.Thread, child string) bool {
	if pendingApproval(t) || pendingInput(t) || child != "" {
		return false
	}
	rt := threadRuntime(t)
	if !runtimeActive(rt) && rt != "idle" {
		return false
	}
	runSettled := t.LatestRunID != "" && !runtimeActive(latestRunStatus(t)) && t.ActiveRunID != t.LatestRunID
	return !(t.InteractionMode == "plan" && t.HasActionableProposedPlan && runSettled)
}

// recede reports a quieter row (shouldRecedeSidebarThread): always while it
// works, and when it's done or asking for approval but neither unread nor
// woke. The open row, a selected one, and one that needs the user never do.
func recede(st sidebarStatus, unread, woke, active, selected bool) bool {
	switch {
	case active || selected || st == stInput || st == stRequest:
		return false
	case st == stPreparing || st == stWorking || st == stWaiting:
		return true
	case st == stReady || st == stApproval:
		return !unread && !woke
	}
	return false
}

// workingState is a thread's place on the Working shelf, kept across
// frames so a thread whose run ends stays there for two seconds: an
// orchestrator's run can end moments before the next starts
// (createInboxReturnTracker, threadInbox.ts).
type workingState struct {
	working   bool
	deadline  time.Time // set while held past the end of its work
	placement string    // where it was filed, to notice it moving
}

// placement is what filing a thread depends on besides its work.
func placement(t *t3.Thread) string {
	return strings.Join([]string{t.ArchivedAt, t.DeletedAt, t.PinnedAt, t.SettledOverride, t.SettledAt, t.SnoozedAt, t.SnoozedUntil, t.ActiveOrderKey}, "|")
}

// onWorkingShelf reports whether ref stays on the Working shelf now, and
// notes when a thread left it for the Active sort. Call with the lock held.
func (a *App) onWorkingShelf(ref string, t *t3.Thread, child string, now time.Time) bool {
	prev, seen := a.working[ref]
	actual := working(t, child)
	pl := ""
	if actual || prev.working {
		pl = placement(t)
	}
	actionable := pendingApproval(t) || pendingInput(t) || child != "" || t.PendingGuestMessageCount > 0 ||
		(t.InteractionMode == "plan" && t.HasActionableProposedPlan) || t.LastError != "" || latestRunStatus(t) == "failed"
	moved := seen && prev.placement != pl
	excluded := t.ArchivedAt != "" || t.DeletedAt != "" || t.PinnedAt != "" || t.SettledOverride == "settled" || snoozed(t, child, now)
	done := latestRunDone(t)
	endedLongAgo := !done.IsZero() && !done.After(now.Add(-staleExit))
	var deadline time.Time
	if !actual && prev.working && !actionable && !moved && !excluded && !endedLongAgo {
		deadline = prev.deadline
		if deadline.IsZero() {
			deadline = now.Add(workingGrace)
		}
	}
	w := actual || deadline.After(now)
	if prev.working && !w {
		a.returned[ref] = now
	}
	if !w {
		deadline = time.Time{}
	}
	a.working[ref] = workingState{working: w, deadline: deadline, placement: pl}
	if !deadline.IsZero() {
		// Look again when the grace runs out.
		a.redrawAt(deadline)
	}
	return w
}

// redrawAt asks for a new scene at t, for state that changes with time:
// one timer, for the soonest time asked for. Call with the lock held.
func (a *App) redrawAt(t time.Time) {
	d := time.Until(t)
	if d <= 0 || (a.timer != nil && !a.timerAt.After(t)) {
		return
	}
	if a.timer != nil {
		a.timer.Stop()
	}
	// A timer stopped as it fires is already waiting for the lock: it
	// finds it's been replaced and leaves the new one be.
	var timer *time.Timer
	timer = time.AfterFunc(d, func() {
		a.mu.Lock()
		if a.timer == timer {
			a.timer = nil
		}
		a.mu.Unlock()
		a.changed()
	})
	a.timer, a.timerAt = timer, t
}

// ---------- order ----------

// sideThread is a listed thread with its environment and what the sidebar
// derived for it this frame.
type sideThread struct {
	*t3.Thread
	e     *envState
	ref   string
	child string // a child's request rolled up to it
}

func ms(s string) int64 {
	if t := parseTime(s); !t.IsZero() {
		return t.UnixMilli()
	}
	return 0
}

// newestFirst orders by key descending, then id and environment.
func newestFirst(ts []sideThread, key func(sideThread) int64) {
	sort.SliceStable(ts, func(i, j int) bool {
		ki, kj := key(ts[i]), key(ts[j])
		if ki != kj {
			return ki > kj
		}
		if ts[i].ID != ts[j].ID {
			return ts[i].ID < ts[j].ID
		}
		return ts[i].e.id() < ts[j].e.id()
	})
}

// sortPinned is sortPinnedThreadsByOrderKey: keyed threads by their key,
// then the rest newest first.
func sortPinned(ts []sideThread) {
	sort.SliceStable(ts, func(i, j int) bool {
		ki, kj := ts[i].PinOrderKey, ts[j].PinOrderKey
		if (ki != "") != (kj != "") {
			return ki != ""
		}
		if ki != kj {
			return ki < kj
		}
		if ki == "" {
			if ci, cj := ms(ts[i].CreatedAt), ms(ts[j].CreatedAt); ci != cj {
				return ci > cj
			}
		}
		if ts[i].ID != ts[j].ID {
			return ts[i].ID < ts[j].ID
		}
		return ts[i].e.id() < ts[j].e.id()
	})
}

// activityTime is the latest of the last message and the latest run's
// request, start and end, else the last update
// (resolveThreadActivityTimestamp): what a row's age counts from.
func activityTime(t *t3.Thread) time.Time {
	var latest time.Time
	for _, s := range []string{t.LatestUserMessageAt, t.LatestRunRequestedAt, t.LatestRunStartedAt} {
		if v := parseTime(s); v.After(latest) {
			latest = v
		}
	}
	if d := latestRunDone(t); d.After(latest) {
		latest = d
	}
	if latest.IsZero() {
		return parseTime(t.UpdatedAt)
	}
	return latest
}

// settledTime is when the thread was settled, else its activity.
func settledTime(t *t3.Thread) time.Time {
	if s := parseTime(t.SettledAt); !s.IsZero() {
		return s
	}
	return activityTime(t)
}

// ---------- formats ----------

// age is a row's age (formatRelativeTimeLabel, compacted): "now" for under
// a minute, then whole minutes, hours and days, counted to now's minute.
func age(at, now time.Time) string {
	if at.IsZero() {
		return ""
	}
	d := max(0, now.Truncate(time.Minute).Sub(at))
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}

// runTimer is a working or waiting label's timer
// (formatWorkingDurationLabel): seconds under a minute, minutes under an
// hour, then hours and minutes.
func runTimer(d time.Duration) string {
	s := max(0, int(d/time.Second))
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %dm", m/60, m%60)
}

// wakeLabel is how long until a snoozed thread wakes, rounded up
// (snoozeWakeLabel): minutes under an hour, hours under a day, then days.
func wakeLabel(until, now time.Time) string {
	left := until.Sub(now)
	switch {
	case until.IsZero() || left <= 0:
		return "now"
	case left < time.Hour:
		return fmt.Sprintf("%dm", max(1, int(math.Ceil(left.Minutes()))))
	case left < 24*time.Hour:
		return fmt.Sprintf("%dh", int(math.Ceil(left.Hours())))
	}
	return fmt.Sprintf("%dd", int(math.Ceil(left.Hours()/24)))
}

// workStart is when the run being timed started (resolveThreadWorkingStartedAt).
func workStart(t *t3.Thread) time.Time {
	if t.ActivityRunStartedAt != "" {
		return parseTime(t.ActivityRunStartedAt)
	}
	if t.LatestRunID != "" && t.LatestRunID == t.ActiveRunID && latestRunDone(t).IsZero() {
		if s := parseTime(t.LatestRunStartedAt); !s.IsZero() {
			return s
		}
		return parseTime(t.LatestRunRequestedAt)
	}
	return time.Time{}
}

// ---------- the list ----------

// sidebarList is the threads the sidebar lists this frame, shelved and
// sorted, with the keyboard's order through them.
type sidebarList struct {
	pinned, active, working, snoozed, settled []sideThread
	ordered                                   []string // refs top to bottom, as the jump keys and ranges count
}

// rowThread is the row thread r shows as: its own, or for a subagent or a
// side chat, which have none, its nearest listed ancestor's
// (resolveSidebarRowThreadId). A broken chain leaves r, which matches no
// row. Call with the lock held.
func (a *App) rowThread(r string) string {
	e, t := a.thread(r)
	seen := map[string]bool{}
	for t != nil && !listed(t) {
		if t.Lineage.ParentThreadID == "" || seen[t.ID] {
			return r
		}
		seen[t.ID] = true
		t = e.threads[t.Lineage.ParentThreadID]
	}
	if t == nil {
		return r
	}
	return ref(e.id(), t.ID)
}

// shelve sorts every listed thread in the project scope onto its shelf.
// Call with the lock held.
func (a *App) shelve(now time.Time, scope map[string]bool) sidebarList {
	var l sidebarList
	a.openRow = a.rowThread(a.openID)
	present := map[string]bool{}
	for _, e := range a.envs {
		children := childRequests(e.threads)
		for _, t := range e.threads {
			if !listed(t) || t.ArchivedAt != "" || t.DeletedAt != "" || (scope != nil && !scope[ref(e.id(), t.ProjectID)]) {
				continue
			}
			st := sideThread{Thread: t, e: e, ref: ref(e.id(), t.ID), child: children[t.ID]}
			present[st.ref] = true
			// Snoozing and settling are the server's settings at work: until
			// its config has loaded, a thread is pinned or active.
			loaded := e.config != nil
			if wake := parseTime(t.SnoozedUntil); loaded && wake.After(now) {
				// A snooze ends when it's due, not on the minute's tick
				// after (useSnoozeWakeClock).
				a.redrawAt(wake)
			}
			// Every thread is watched, so one filed elsewhere loses its hold.
			w := a.onWorkingShelf(st.ref, t, st.child, now)
			switch {
			case loaded && snoozed(t, st.child, now):
				l.snoozed = append(l.snoozed, st)
			case loaded && t.SettledOverride == "settled":
				l.settled = append(l.settled, st)
			case t.PinnedAt != "":
				l.pinned = append(l.pinned, st)
			case w:
				l.working = append(l.working, st)
			default:
				l.active = append(l.active, st)
			}
		}
	}
	for r := range a.working {
		if !present[r] {
			delete(a.working, r)
			delete(a.returned, r)
		}
	}
	sortPinned(l.pinned)
	newestFirst(l.active, func(t sideThread) int64 {
		k := max(ms(t.CreatedAt), ms(t.UnsettledAt), ms(t.LatestRunRequestedAt), latestRunDone(t.Thread).UnixMilli())
		if r, ok := a.returned[t.ref]; ok {
			k = max(k, r.UnixMilli())
		}
		return k
	})
	newestFirst(l.working, func(t sideThread) int64 {
		// A thread the user never wrote in goes by when it began; only an
		// older server, which doesn't send the field, by its run's request.
		sent := t.LatestUserAuthoredMessageAt.At
		if !t.LatestUserAuthoredMessageAt.Sent {
			sent = t.LatestRunRequestedAt
		}
		return max(ms(t.CreatedAt), ms(sent))
	})
	// Soonest to wake first; ties keep one order, as T3's do over its
	// stable shell list, not the map's, which changes with every build.
	sort.SliceStable(l.snoozed, func(i, j int) bool {
		a, b := l.snoozed[i], l.snoozed[j]
		if wa, wb := ms(a.SnoozedUntil), ms(b.SnoozedUntil); wa != wb {
			return wa < wb
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.e.id() < b.e.id()
	})
	sort.SliceStable(l.settled, func(i, j int) bool {
		si, sj := settledTime(l.settled[i].Thread), settledTime(l.settled[j].Thread)
		if !si.Equal(sj) {
			return si.After(sj)
		}
		return l.settled[i].ID < l.settled[j].ID
	})
	for _, s := range [][]sideThread{l.pinned, l.active, a.shown("working", l.working), a.shown("snoozed", l.snoozed), a.shownSettled(l.settled)} {
		for _, t := range s {
			l.ordered = append(l.ordered, t.ref)
		}
	}
	a.parked = map[string]bool{}
	for _, t := range append(append([]sideThread{}, l.snoozed...), l.settled...) {
		a.parked[t.ref] = true
	}
	return l
}

// shown is the part of a shelf the list draws: all of it while it's open,
// else just the open thread's row, if it's there.
func (a *App) shown(shelf string, ts []sideThread) []sideThread {
	if a.shelfOpen[shelf] {
		return ts
	}
	for _, t := range ts {
		if t.ref == a.openRow {
			return []sideThread{t}
		}
	}
	return nil
}

// shownSettled is the Settled shelf's drawn rows: its first pages, plus the
// open thread's row wherever it is.
func (a *App) shownSettled(ts []sideThread) []sideThread {
	vis := ts[:min(len(ts), a.settledShown())]
	for _, t := range ts[len(vis):] {
		if t.ref == a.openRow {
			vis = append(vis[:len(vis):len(vis)], t)
		}
	}
	return a.shown("settled", vis)
}

func (a *App) settledShown() int { return settledFirst + a.settledMore*settledPage }

// ---------- rows ----------

// sidebar fills the scene's sidebar. Call with the lock held.
func (a *App) sidebar(s *ui.Scene, now time.Time) {
	sb := &s.Sidebar
	scope := a.scopeSet()
	l := a.shelve(now, scope)
	a.ordered = l.ordered
	for _, e := range a.envs {
		sb.HasProjects = sb.HasProjects || len(e.projects) > 0
	}
	sb.NoNew = !sb.HasProjects
	sb.Scope, sb.ScopeLabel = a.scopeIcons(scope)
	sb.ScopeKey, sb.Open = strings.Join(a.scope, "\n"), a.openRow
	if a.filterOpen {
		a.filterScene(sb, scope)
	}
	sb.Renaming = a.renamingRow
	sb.Undo, sb.UndoClaim = a.undoSummary(), a.undoClaim
	defer a.snoozeScene(sb)
	if q := strings.TrimSpace(a.search); q != "" {
		a.searchScene(sb, l, q, now)
		return
	}
	// The results go with their query, so Enter in an emptied field opens
	// nothing.
	a.results = a.results[:0]
	draft := a.draftRow(scope)
	if draft != nil {
		sb.Items = append(sb.Items, ui.SidebarItem{Draft: draft}, ui.SidebarItem{Divider: true})
	}
	// mod+1…9 open the list's first nine rows, which show their keys.
	jump := map[string]int{}
	for i, r := range l.ordered[:min(9, len(l.ordered))] {
		jump[r] = i + 1
	}
	add := func(t sideThread, shelf string) {
		r := a.row(t, shelf, now)
		r.Jump = jump[t.ref]
		sb.Items = append(sb.Items, ui.SidebarItem{Row: &r})
		sb.Ticking = sb.Ticking || r.Timer != ""
	}
	for _, t := range l.pinned {
		add(t, "")
	}
	for _, t := range l.active {
		add(t, "")
	}
	shelf := func(id, label string, n int, ts []sideThread) {
		sb.Items = append(sb.Items, ui.SidebarItem{Shelf: &ui.Shelf{ID: id, Label: label, Count: n, Open: a.shelfOpen[id]}})
		for _, t := range ts {
			add(t, id)
		}
	}
	if len(l.working) > 0 {
		shelf("working", "Working", len(l.working), a.shown("working", l.working))
	}
	if len(l.snoozed) > 0 {
		shelf("snoozed", "Snoozed", len(l.snoozed), a.shown("snoozed", l.snoozed))
	}
	listedAny := len(l.pinned)+len(l.active)+len(l.working)+len(l.snoozed)+len(l.settled) > 0
	if listedAny {
		shown := a.shownSettled(l.settled)
		shelf("settled", "Settled", len(l.settled), shown)
		if hidden := len(l.settled) - len(shown); a.shelfOpen["settled"] && hidden > 0 {
			sb.Items = append(sb.Items, ui.SidebarItem{More: min(hidden, settledPage)})
		}
	}
	if draft == nil && len(l.pinned)+len(l.active)+len(l.working)+len(l.snoozed) == 0 && (len(l.settled) == 0 || scope != nil) {
		sb.Empty = a.emptyState(scope, len(l.settled) > 0)
	}
}

// snoozeScene shows the snooze menu while its card is in the list and still
// offers the clock. A card that leaves (onto a folded shelf, or under a
// search's results) or stops offering it (a queued turn) closes its menu,
// so the menu doesn't hang where nothing draws it, nor come back with the
// clock (SidebarThreadRow.tsx). Call with the lock held, once the list is
// built.
func (a *App) snoozeScene(sb *ui.Sidebar) {
	if a.snoozeRef == "" {
		return
	}
	for _, it := range sb.Items {
		if r := it.Row; r != nil && r.ID == a.snoozeRef && !r.Slim && r.CanSnooze {
			sb.Snooze = &ui.SnoozeMenu{Row: a.snoozeRef, Choices: snoozeItems(a.snoozeMenu, "", false)}
			return
		}
	}
	a.snoozeRef, a.snoozeMenu = "", nil
}

// draftRow is the new thread's unsent draft as a row
// (SidebarDraftBlock.tsx), nil without one: something written or pasted,
// for a project in the scope. Its preview is the first line written, else
// how many pictures it holds. Call with the lock held.
func (a *App) draftRow(scope map[string]bool) *ui.DraftRow {
	if a.draft != "" {
		a.draftHome = ref(a.draftEnv, a.draft)
	}
	envID, id := splitRef(a.draftHome)
	e := a.env(envID)
	text := strings.TrimSpace(ui.StripChips(firstLine(drafts["draft"])))
	n := len(a.images["draft"])
	if e == nil || (text == "" && n == 0) || (scope != nil && !scope[a.draftHome]) {
		return nil
	}
	p, ok := e.projects[id]
	if !ok {
		return nil
	}
	if text == "" {
		text = plural(n, "attachment")
	}
	return &ui.DraftRow{ID: "draft", Project: p.Title, Icon: projectIcon(p), Preview: text, Active: a.draft != ""}
}

// emptyState is what the list says with nothing active to show.
func (a *App) emptyState(scope map[string]bool, settled bool) *ui.SidebarEmpty {
	connected, projects := false, false
	for _, e := range a.envs {
		connected = connected || e.conn != nil || len(e.threads) > 0
		projects = projects || len(e.projects) > 0
	}
	switch {
	case !connected:
		return &ui.SidebarEmpty{Text: "Connecting…"}
	case !projects:
		return &ui.SidebarEmpty{Text: "No projects yet"}
	case scope != nil:
		name := "these projects"
		for k := range scope {
			if len(scope) == 1 {
				name = a.scopeTitle(k)
			}
		}
		if settled {
			return &ui.SidebarEmpty{Text: "No active threads in " + name, Button: "Show all projects"}
		}
		return &ui.SidebarEmpty{Text: "No threads in " + name + " yet", Button: "Show all projects"}
	}
	return &ui.SidebarEmpty{Text: "No threads yet"}
}

// row is one thread's row: a card unless it's on the Snoozed or Settled
// shelf.
func (a *App) row(st sideThread, shelf string, now time.Time) ui.ThreadRow {
	t, e := st.Thread, st.e
	active := st.ref == a.openRow
	r := ui.ThreadRow{
		ID:           st.ref,
		Slim:         shelf == "snoozed" || shelf == "settled",
		Shelf:        shelf,
		Title:        t.Title,
		Active:       active,
		Selected:     a.selected[st.ref],
		Pinned:       t.PinnedAt != "",
		Draft:        !active && drafts[st.ref] != "",
		Regenerating: t.TitleRegeneration != nil,
		Branch:       t.Branch,
		Worktree:     strings.TrimSpace(t.WorktreePath),
		Machine:      e.env.Label,
		MachineIcon:  machineIcon(e.config),
		Terminals:    a.busyTerminals(st.ref),
	}
	a.project(e, t.ProjectID, &r)
	if a.scratch(e, t.ProjectID) && t.Branch == "" && a.machineCount() > 1 {
		r.Scratch = e.env.Label
	}
	r.Remote = a.machineCount() > 1 && r.Scratch == ""
	st2 := status(t, st.child)
	// The open thread's watermark moves as it's read, so it's never unread.
	r.Unread = !active && unseen(t)
	r.Woke = woke(t, st.child, now)
	r.Recede = recede(st2, r.Unread, r.Woke, active, r.Selected)
	r.Status, r.Label = rowStatus(t, st2, r.Woke, r.Unread)
	// Timers count to the whole second (useNowSecond): a scene rebuilt
	// mid-second shows the second it's in, not the next.
	sec := now.Truncate(time.Second)
	switch st2 {
	case stPreparing, stWorking:
		if s := workStart(t); !s.IsZero() {
			r.Timer = runTimer(sec.Sub(s))
		}
	case stWaiting:
		if d := latestRunDone(t); !d.IsZero() {
			r.Timer = runTimer(sec.Sub(d))
		}
	}
	switch shelf {
	case "snoozed":
		r.Wake = wakeLabel(parseTime(t.SnoozedUntil), now)
	case "settled":
		r.Age = age(settledTime(t), now)
	default:
		r.Age = age(activityTime(t), now)
	}
	if t.LastError != "" {
		r.Limited = t.LastErrorClass == "usage_limit"
		r.Error = "Error occurred"
		if r.Limited {
			r.Error = "Usage limit reached"
		}
	}
	a.providers(e, t, &r)
	// Settle, Un-settle and Wake need only the server's config, as T3's
	// settlementSupported and snoozeSupported do; without write access a
	// click says why it can't. The snooze clock and Unpin need write access
	// too (SidebarThreadRow.tsx: canOperateThread). With the config come
	// the linked pull requests, in the badge and the hover card's footer.
	loaded := e.config != nil
	r.PR = prBadge(t, loaded)
	if vis := visiblePRs(t.PullRequests); loaded && len(vis) > 0 {
		r.PRs = prLines(vis)
	}
	write := loaded && e.operate
	r.CanSettle, r.CanUnpin = loaded, write
	r.CanSnooze = write && canSnooze(t, st.child, now)
	if write && !r.CanSnooze && queuedTurnStart(t, now) {
		// The clock comes back as the queued turn's grace runs out, which
		// T3 looks at 50ms after.
		if at := parseTime(t.LatestUserMessageAt).Add(queuedTurnGrace + 50*time.Millisecond); at.After(now) {
			a.redrawAt(at)
		}
	}
	return r
}

// rowStatus is a card's status label: its state, or for a thread that's
// done, Woke or Done until it's seen. A seen, finished thread shows its age.
func rowStatus(t *t3.Thread, st sidebarStatus, woke, unread bool) (ui.RowStatus, string) {
	switch st {
	case stPreparing:
		return ui.StatusWorking, "Setting up"
	case stWorking:
		if t.Goal != nil && t.Goal.Status == "active" {
			return ui.StatusWorking, "Goal"
		}
		return ui.StatusWorking, "Working"
	case stWaiting:
		return ui.StatusWaiting, "Waiting"
	case stApproval:
		return ui.StatusApproval, "Approval"
	case stInput:
		return ui.StatusInput, "Input"
	case stRequest:
		return ui.StatusRequest, "Request"
	case stLimited:
		return ui.StatusLimited, "Limited"
	case stFailed:
		return ui.StatusFailed, "Failed"
	case stReady:
		if woke {
			return ui.StatusWoke, "Woke"
		}
		if unread {
			return ui.StatusDone, "Done"
		}
	}
	return ui.StatusNone, ""
}

// project fills in the row's project: its name and icon. "No project"
// threads get T3's dashed message icon.
func (a *App) project(e *envState, id string, r *ui.ThreadRow) {
	p, ok := e.projects[id]
	switch {
	case a.scratch(e, id):
		r.Project = "No project"
		r.Icon = ui.ProjectIcon{Lucide: "message-square-dashed", Color: "gray"}
	case ok:
		r.Project = p.Title
		r.Icon = projectIcon(p)
	}
}

// scratch reports the environment's "No project" project, where threads
// without one live.
func (a *App) scratch(e *envState, id string) bool {
	p, ok := e.projects[id]
	return ok && e.config != nil && e.config.ScratchWorkspaceRoot != "" && strings.TrimRight(p.WorkspaceRoot, "/") == strings.TrimRight(e.config.ScratchWorkspaceRoot, "/")
}

// machineCount is how many machines have threads: past one, rows show
// which machine they're on.
func (a *App) machineCount() int {
	n := 0
	for _, e := range a.envs {
		if len(e.threads) > 0 {
			n++
		}
	}
	return n
}

// machineIcon is the machine's icon (resolveEnvironmentMachineKind), as a
// name the sidebar draws.
func machineIcon(cfg *t3.ServerConfig) string {
	if cfg == nil {
		return "server"
	}
	if k := cfg.Settings.EnvironmentIcon; k != "" {
		return k
	}
	if k := cfg.Environment.Platform.Machine; k != "" {
		return k
	}
	return "server"
}

// busyTerminals counts the thread's terminals with a command running.
func (a *App) busyTerminals(r string) int {
	n := 0
	if d := a.drawers[r]; d != nil {
		for _, tm := range d.terms {
			if tm.busy && tm.status == "running" {
				n++
			}
		}
	}
	return n
}

// providers fills in the provider stack (resolveThreadProviderStack): up to
// two earlier providers, then the current one, and the hover card's model
// and handoff lines.
func (a *App) providers(e *envState, t *t3.Thread, r *ui.ThreadRow) {
	var sel t3.ModelSelection
	if t.ModelSelection != nil {
		sel = *t.ModelSelection
	}
	// The runtime's provider, else the one the thread's model is on.
	cur := sel.InstanceID
	if threadRuntime(t) != "" && t.ProviderInstanceID != "" {
		cur = t.ProviderInstanceID
	}
	byID := func(id string) *t3.Provider {
		if e.config == nil {
			return nil
		}
		for i := range e.config.Providers {
			if e.config.Providers[i].InstanceID == id {
				return &e.config.Providers[i]
			}
		}
		return nil
	}
	var prev []string
	for _, id := range t.ProviderInstanceHistory {
		if id != cur {
			prev = append(prev, id)
		}
	}
	if len(prev) > 0 {
		var names []string
		for _, id := range prev {
			if p := byID(id); p != nil {
				id = p.DisplayName
			}
			names = append(names, id)
		}
		r.Handoff = "Handed off from " + strings.Join(names, ", ")
	}
	p := byID(cur)
	if p == nil {
		// Without the current provider there's no stack at all.
		return
	}
	for _, id := range prev[max(0, len(prev)-2):] {
		if q := byID(id); q != nil {
			r.Providers = append(r.Providers, providerName(q))
		}
	}
	r.Provider = providerName(p)
	r.Providers = append(r.Providers, r.Provider)
	r.Model = sel.Model
	for _, m := range p.Models {
		if m.Slug == sel.Model && m.Name != "" {
			r.Model = modelLabel(m, p.Driver)
		}
	}
	// With several accounts of one driver, or an accent colour, the line
	// says which account.
	if instanceBadge(p, e.config.Providers) {
		r.Model += " · " + cmp.Or(p.DisplayName, providerName(p))
	}
}

// modelFamilies are the model name families each driver's own models are
// of (providerClients.ts): on its own driver a family's name drops to its
// variant, "Claude Opus 5.5" to "Opus 5.5".
var modelFamilies = map[string][]string{
	"codex": {"GPT-", "GPT "}, "claudeAgent": {"Claude "}, "antigravity": {"Gemini "},
	"cursor": {"Grok "}, "grok": {"Grok "}, "muse": {"Muse "},
}

var (
	familyPrefixes = []string{"Claude ", "GPT-", "GPT ", "Gemini ", "Muse ", "Grok "}
	// Words that say which variant of a family, rather than name a model.
	variantWords = map[string]bool{"pro": true, "flash": true, "mini": true, "max": true, "lite": true, "nano": true,
		"ultra": true, "plus": true, "fast": true, "turbo": true, "preview": true, "latest": true, "thinking": true,
		"reasoning": true, "build": true, "code": true, "image": true, "vision": true, "audio": true, "live": true,
		"custom": true, "tools": true, "exp": true, "experimental": true, "chat": true, "tts": true, "stt": true,
		"non": true, "transcribe": true}
	versionWord  = regexp.MustCompile(`(?i)^v?\d+(?:\.\d+)*[a-z]?$`)
	vendorPrefix = regexp.MustCompile(`^[A-Z][\w.&\- ]{0,24}: `)
	inParens     = regexp.MustCompile(`\(.*?\)`)
	wordBreaks   = regexp.MustCompile(`[\s-]+`)
)

// modelLabel is the model as the hover card names it
// (getTriggerDisplayModelName): its short name, or one made from its name
// in its driver's families, without the sub-provider before it.
func modelLabel(m t3.Model, driver string) string {
	name := cmp.Or(m.ShortName, modelShortName(m.Name, modelFamilies[driver]), m.Name)
	q := strings.TrimSpace(m.SubProvider)
	if q == "" {
		return name
	}
	re := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(q) + `(?:\s*[.:/-]\s*|\s+)`)
	if s := strings.TrimSpace(re.ReplaceAllString(name, "")); s != "" {
		return s
	}
	return name
}

// modelShortName is a model's name without its vendor and family where
// what's left still names it (modelShortName.ts): "Opus 5.5", "5.5 Pro"
// on its own driver; "" when that's the name as it is.
func modelShortName(name string, native []string) string {
	rest := name
	if m := vendorPrefix.FindStringIndex(name); m != nil && m[1] < len(name) && !unicode.IsSpace(rune(name[m[1]])) {
		rest = name[m[1]:]
	}
	short, ok := withoutFamily(rest, native)
	switch {
	case ok:
	case namesModel(rest):
		short = rest
	default:
		short = name
	}
	if short == name {
		return ""
	}
	return short
}

// withoutFamily is name without its family's prefix, if what's left still
// names the model: any word but a version on the family's own driver,
// else a word that's no variant's.
func withoutFamily(name string, native []string) (string, bool) {
	prefix := ""
	for _, p := range familyPrefixes {
		if strings.HasPrefix(name, p) {
			prefix = p
			break
		}
	}
	if prefix == "" {
		return "", false
	}
	rest := strings.TrimSpace(name[len(prefix):])
	keep := namesModel(rest)
	if slices.Contains(native, prefix) {
		keep = len(nonVersionWords(rest)) > 0
	}
	if !keep {
		return "", false
	}
	if strings.ContainsFunc(rest, unicode.IsSpace) {
		return rest, true
	}
	return strings.ReplaceAll(rest, "-", " "), true
}

func nonVersionWords(text string) []string {
	var out []string
	for _, w := range wordBreaks.Split(inParens.ReplaceAllString(text, " "), -1) {
		if w != "" && !versionWord.MatchString(w) {
			out = append(out, w)
		}
	}
	return out
}

func namesModel(text string) bool {
	for _, w := range nonVersionWords(text) {
		if !variantWords[strings.ToLower(w)] {
			return true
		}
	}
	return false
}

// instanceBadge reports a provider its rows tell from others
// (shouldShowInstanceBadge): it has an accent colour, or its driver has
// more than one account, counting only enabled ones when it is.
func instanceBadge(p *t3.Provider, all []t3.Provider) bool {
	if c := strings.TrimSpace(p.AccentColor); len(c) == 7 && c[0] == '#' && strings.Trim(c[1:], "0123456789abcdefABCDEF") == "" {
		return true
	}
	n := 0
	for _, q := range all {
		if q.Driver != p.Driver || p.Enabled && !q.Enabled || p.Driver == "acpRegistry" && q.AcpRegistryAgentID != p.AcpRegistryAgentID {
			continue
		}
		if n++; n > 1 {
			return true
		}
	}
	return false
}

// ---------- project icons ----------

// projectIconColors are T3's 18 project icon colours, in the order its
// name hash indexes them (projectIconColors.ts).
var projectIconColors = []string{"gray", "red", "orange", "amber", "yellow", "lime", "green", "emerald", "teal",
	"cyan", "sky", "blue", "indigo", "violet", "purple", "fuchsia", "pink", "rose"}

// projectIcon is the project's chosen icon, a monogram or a lucide icon in
// its colour (a lucide one carrying a monogram from older servers is that
// monogram), else one made from its name (deriveProjectIdentity), which is
// also what an emoji gets: Gio sets no colour emoji. tern loads no
// favicons, so the made one is what T3 draws until a project's favicon
// loads.
func projectIcon(p t3.Project) ui.ProjectIcon {
	if i := p.ProjectIcon; i != nil {
		switch text := cmp.Or(i.Text, i.MonogramText, i.Monogram); {
		case (i.Kind == "monogram" || i.Kind == "lucide") && text != "":
			return ui.ProjectIcon{Text: text, Color: i.Color}
		case i.Kind == "lucide" && i.Name != "":
			return ui.ProjectIcon{Lucide: i.Name, Color: i.Color}
		}
	}
	return ui.ProjectIcon{Text: monogram(p.Title), Color: monogramColor(p.Title)}
}

// monogram is the first letter of the name's first word, then the first
// digit after it, else the first letter of the last word, else the first
// word's last letter, upper-cased: "Tern" is "TN", "app 2" is "A2".
func monogram(name string) string {
	words := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	if len(words) == 0 {
		return "PR"
	}
	first := []rune(words[0])
	second := first[len(first)-1]
	found := false
	for _, g := range first[1:] {
		if unicode.IsNumber(g) {
			second, found = g, true
			break
		}
	}
	if !found && len(words) > 1 {
		second = []rune(words[len(words)-1])[0]
	}
	m := []rune(strings.ToUpper(string(first[0]) + string(second)))
	return string(m[:min(2, len(m))])
}

// monogramColor hashes the lower-cased name into the 18 colours.
func monogramColor(name string) string {
	seed := strings.ToLower(strings.TrimSpace(name))
	if seed == "" {
		seed = "project"
	}
	i := 0
	for _, r := range seed {
		i = (i*31 + int(r)) % len(projectIconColors)
	}
	return projectIconColors[i]
}
