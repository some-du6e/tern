package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The sidebar's rules, from docs/ui/04-sidebar.md §5, checked on what the
// scene shows: labels, ages, timers, order.

// Ages count whole minutes, hours, then days, and never switch to weeks
// (§5.5).
func TestAge(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 30, 0, 0, time.UTC) // on the minute, as the shared clock is
	for _, c := range []struct {
		ago  time.Duration
		want string
	}{
		{0, "now"}, {50 * time.Second, "now"}, {5 * time.Minute, "5m"}, {59 * time.Minute, "59m"},
		{2*time.Hour + 5*time.Minute, "2h"}, {23 * time.Hour, "23h"}, {3 * 24 * time.Hour, "3d"}, {45 * 24 * time.Hour, "45d"},
	} {
		if got := age(now.Add(-c.ago), now); got != c.want {
			t.Errorf("%v ago: %q, want %q", c.ago, got, c.want)
		}
	}
}

// A run timer is seconds under a minute, minutes under an hour, then hours
// and minutes, with no seconds past the first minute (§5.5).
func TestRunTimer(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0: "0s", 59 * time.Second: "59s", 61 * time.Second: "1m", 59*time.Minute + 59*time.Second: "59m",
		time.Hour: "1h 0m", 4*time.Hour + 12*time.Minute + 30*time.Second: "4h 12m",
	} {
		if got := runTimer(d); got != want {
			t.Errorf("%v: %q, want %q", d, got, want)
		}
	}
}

// A snoozed row's wake label rounds up (§5.5).
func TestWakeLabel(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	for d, want := range map[time.Duration]string{
		-time.Minute: "now", 20 * time.Second: "1m", 59 * time.Minute: "59m",
		61 * time.Minute: "2h", 23 * time.Hour: "23h", 25 * time.Hour: "2d",
	} {
		if got := wakeLabel(now.Add(d), now); got != want {
			t.Errorf("%v: %q, want %q", d, got, want)
		}
	}
}

// sideApp is an App with one machine and the threads given, its server's
// config loaded, so the shelves and menus are T3's full ones.
func sideApp(t *testing.T, threads ...t3.Thread) *App {
	t.Helper()
	a := testApp(t)
	e := newEnv(t3.Environment{EnvironmentID: "m", Label: "workstation"}, false)
	e.status = "online"
	e.projects["p"] = t3.Project{ID: "p", Title: "tern", WorkspaceRoot: "/srv/tern"}
	e.config = &t3.ServerConfig{}
	for i := range threads {
		th := threads[i]
		if th.ProjectID == "" {
			th.ProjectID = "p"
		}
		e.threads[th.ID] = &th
	}
	a.envs = []*envState{e}
	return a
}

// ago is a timestamp d before now.
func ago(now time.Time, d time.Duration) string { return now.Add(-d).UTC().Format(time.RFC3339Nano) }

// The status label follows T3's precedence (resolveSidebarThreadStatus):
// approval, input, a guest's request, waiting, setting up, working, then a
// failure; a finished thread says Done until it's seen.
func TestStatusLabels(t *testing.T) {
	now := time.Now()
	run := func(status string) t3.Thread {
		return t3.Thread{LatestRunID: "r", Status: status, ActivityRunStatus: status, LatestRunStartedAt: ago(now, 3*time.Minute), ActivityRunStartedAt: ago(now, 3*time.Minute)}
	}
	with := func(t t3.Thread, f func(*t3.Thread)) t3.Thread { f(&t); return t }
	cases := []struct {
		t     t3.Thread
		label string
		timer string
	}{
		{with(run("running"), func(t *t3.Thread) { t.PendingRuntimeRequest = &t3.RuntimeRequest{Kind: "command"} }), "Approval", ""},
		{with(run("running"), func(t *t3.Thread) { t.PendingRuntimeRequest = &t3.RuntimeRequest{Kind: "user_input"} }), "Input", ""},
		{with(run("running"), func(t *t3.Thread) { t.PendingGuestMessageCount = 1 }), "Request", ""},
		{with(run("running"), func(t *t3.Thread) { t.BlockedOnThreadWait = true; t.LatestRunCompletedAt = ago(now, time.Minute) }), "Waiting", "1m"},
		{run("preparing"), "Setting up", "3m"},
		{run("running"), "Working", "3m"},
		{with(run("running"), func(t *t3.Thread) { t.Goal = &t3.Goal{Status: "active"} }), "Goal", "3m"},
		{with(t3.Thread{LatestRunID: "r", Status: "failed", LastErrorClass: "usage_limit"}, func(*t3.Thread) {}), "Limited", ""},
		{with(t3.Thread{LatestRunID: "r", Status: "failed"}, func(*t3.Thread) {}), "Failed", ""},
		{t3.Thread{LatestRunID: "r", Status: "completed", LatestRunCompletedAt: ago(now, time.Minute), LastVisitedAt: ago(now, time.Hour)}, "Done", ""},
		// Seen since it finished: the age, no label.
		{t3.Thread{LatestRunID: "r", Status: "completed", LatestRunCompletedAt: ago(now, time.Hour), LastVisitedAt: ago(now, time.Minute)}, "", ""},
		// Never visited: not unread, so no Done (tern used to mark these).
		{t3.Thread{LatestRunID: "r", Status: "completed", LatestRunCompletedAt: ago(now, time.Minute)}, "", ""},
	}
	for i, c := range cases {
		c.t.ID, c.t.Title, c.t.CreatedAt = "t", fmt.Sprint("case ", i), ago(now, time.Hour)
		a := sideApp(t, c.t)
		a.shelfOpen["working"] = true // where the working ones go
		s := a.build(now)
		rs := rows(s)
		if len(rs) != 1 {
			t.Fatalf("case %d: %d rows", i, len(rs))
		}
		if rs[0].Label != c.label || rs[0].Timer != c.timer {
			t.Errorf("case %d: label %q timer %q, want %q %q", i, rs[0].Label, rs[0].Timer, c.label, c.timer)
		}
	}
}

// A working thread, and one done and seen, recede; an unread one doesn't.
func TestRecede(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "seen", Title: "seen", CreatedAt: ago(now, 3*time.Hour), LatestRunID: "r", Status: "completed", LatestRunCompletedAt: ago(now, 2*time.Hour), LastVisitedAt: ago(now, time.Hour)},
		t3.Thread{ID: "unread", Title: "unread", CreatedAt: ago(now, 3*time.Hour), LatestRunID: "r", Status: "completed", LatestRunCompletedAt: ago(now, time.Minute), LastVisitedAt: ago(now, time.Hour)},
	)
	for _, r := range rows(a.build(now)) {
		if r.Recede != (r.Title == "seen") {
			t.Errorf("%s: recede %v", r.Title, r.Recede)
		}
	}
}

// listing is the sidebar's list as text: shelves as "[Label N open]", rows
// by title, "+N" for Show N more.
func listing(s *ui.Scene) string {
	var out []string
	for _, it := range s.Sidebar.Items {
		switch {
		case it.Shelf != nil:
			out = append(out, fmt.Sprintf("[%s %d %v]", it.Shelf.Label, it.Shelf.Count, it.Shelf.Open))
		case it.Row != nil:
			out = append(out, it.Row.Title)
		default:
			out = append(out, fmt.Sprintf("+%d", it.More))
		}
	}
	return strings.Join(out, ", ")
}

// The list is pinned, then active, then the Working, Snoozed and Settled
// shelves, each folded until opened, the Settled one 10 at a time (§2.3,
// §5.2, §5.7). Pinned threads with an order key come first; active ones are
// newest first; snoozed ones wake soonest first; settled ones are the
// latest settled first.
func TestShelves(t *testing.T) {
	now := time.Now()
	threads := []t3.Thread{
		{ID: "p1", Title: "pin keyless", PinnedAt: ago(now, time.Hour), CreatedAt: ago(now, time.Hour)},
		{ID: "p2", Title: "pin keyed", PinnedAt: ago(now, time.Hour), PinOrderKey: "a0", CreatedAt: ago(now, 2*time.Hour)},
		{ID: "a1", Title: "older", CreatedAt: ago(now, 2*time.Hour)},
		{ID: "a2", Title: "newer", CreatedAt: ago(now, time.Hour)},
		{ID: "w1", Title: "working", CreatedAt: ago(now, time.Hour), LatestRunID: "r", Status: "running", ActiveRunID: "r", ActivityRunStatus: "running"},
		{ID: "z1", Title: "wakes later", SnoozedUntil: now.Add(3 * time.Hour).UTC().Format(time.RFC3339Nano), CreatedAt: ago(now, time.Hour)},
		{ID: "z2", Title: "wakes sooner", SnoozedUntil: now.Add(time.Hour).UTC().Format(time.RFC3339Nano), CreatedAt: ago(now, time.Hour)},
	}
	for i := range 12 {
		threads = append(threads, t3.Thread{ID: fmt.Sprintf("s%02d", i), Title: fmt.Sprintf("settled %d", i), SettledOverride: "settled",
			SettledAt: ago(now, time.Duration(i+1)*time.Minute), CreatedAt: ago(now, 24*time.Hour)})
	}
	a := sideApp(t, threads...)
	if got, want := listing(a.build(now)), "pin keyed, pin keyless, newer, older, [Working 1 false], [Snoozed 2 false], [Settled 12 false]"; got != want {
		t.Fatalf("folded:\n got %s\nwant %s", got, want)
	}
	th := ui.NewTheme()
	for _, shelf := range []string{"working", "snoozed", "settled"} {
		a.handle(th, ui.Action{ID: "shelf:" + shelf, Kind: "click"})
	}
	s := a.build(now)
	want := "pin keyed, pin keyless, newer, older, [Working 1 true], working, [Snoozed 2 true], wakes sooner, wakes later, [Settled 12 true]"
	for i := range 10 {
		want += fmt.Sprintf(", settled %d", i)
	}
	if got := listing(s); got != want+", +2" {
		t.Fatalf("open:\n got %s\nwant %s, +2", got, want)
	}
	a.handle(th, ui.Action{ID: "settled.more", Kind: "click"})
	if got := listing(a.build(now)); got != want+", settled 10, settled 11" {
		t.Fatalf("after Show 2 more:\n got %s", got)
	}
	// The shelves stay open across a restart.
	b, err := NewWithStore(a.store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !b.shelfOpen["settled"] || !b.shelfOpen["working"] {
		t.Fatal("the shelves' state wasn't kept")
	}
	// Folded, a shelf still shows the open thread.
	a = sideApp(t, threads...)
	a.openID = ref("m", "s05")
	if got := listing(a.build(now)); !strings.HasSuffix(got, "[Settled 12 false], settled 5") {
		t.Fatalf("folded with the open thread: %s", got)
	}
}

// A slim row shows its wake time or its settle age; a card its activity's.
func TestRowAges(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	a := sideApp(t,
		t3.Thread{ID: "z", Title: "z", SnoozedUntil: now.Add(90 * time.Minute).UTC().Format(time.RFC3339Nano), CreatedAt: ago(now, time.Hour)},
		t3.Thread{ID: "s", Title: "s", SettledOverride: "settled", SettledAt: ago(now, 3*time.Hour), LatestUserMessageAt: ago(now, 5*time.Hour)},
		t3.Thread{ID: "c", Title: "c", LatestUserMessageAt: ago(now, 4*time.Minute), CreatedAt: ago(now, time.Hour)},
	)
	a.shelfOpen["snoozed"], a.shelfOpen["settled"] = true, true
	got := map[string]string{}
	for _, r := range rows(a.build(now)) {
		got[r.Title] = r.Age + r.Wake
		if r.Slim != (r.Title != "c") {
			t.Errorf("%s: slim %v", r.Title, r.Slim)
		}
	}
	if got["z"] != "2h" || got["s"] != "3h" || got["c"] != "4m" {
		t.Fatalf("ages %v", got)
	}
}

// A thread whose run ends stays on the Working shelf for 2s, in case the
// next run starts, then moves to Active (WORKING_EXIT_GRACE_MS).
func TestWorkingGrace(t *testing.T) {
	now := time.Now()
	a := sideApp(t, t3.Thread{ID: "w", Title: "w", CreatedAt: ago(now, time.Hour), LatestRunID: "r", Status: "running", ActiveRunID: "r", ActivityRunStatus: "running"})
	if got := listing(a.build(now)); got != "[Working 1 false], [Settled 0 false]" {
		t.Fatalf("running: %s", got)
	}
	w := a.envs[0].threads["w"]
	w.Status, w.ActivityRunStatus, w.ActiveRunID, w.LatestRunCompletedAt = "completed", "", "", now.UTC().Format(time.RFC3339Nano)
	if got := listing(a.build(now.Add(time.Second))); got != "[Working 1 false], [Settled 0 false]" {
		t.Fatalf("1s after its run: %s", got)
	}
	if got := listing(a.build(now.Add(3 * time.Second))); got != "w, [Settled 0 false]" {
		t.Fatalf("3s after its run: %s", got)
	}
}

// With the server's config the row menu is T3's (§5.13), in its order.
func TestFullThreadMenu(t *testing.T) {
	now := time.Now()
	a := sideApp(t, t3.Thread{ID: "t", Title: "t", Branch: "feature/x", CreatedAt: ago(now, time.Hour)})
	a.handle(ui.NewTheme(), ui.Action{ID: "thread:m/t", Kind: "context"})
	s := a.build(now)
	want := "Pin thread, Settle thread, Snooze, Rename thread, Regenerate title, Mark unread, Filter by tern, Copy, Archive thread, Delete"
	if got := menuLabels(s); got != want {
		t.Fatalf("menu:\n got %s\nwant %s", got, want)
	}
	var copyItems []string
	for _, it := range s.Menu.Items {
		if it.ID == "snooze" && (len(it.Children) == 0 || it.Off) {
			t.Fatalf("snooze = %+v", it)
		}
		if it.ID == "copy" {
			for _, c := range it.Children {
				copyItems = append(copyItems, c.Label)
			}
		}
		if (it.ID == "rename" || it.ID == "copy" || it.ID == "archive") != it.Sep {
			t.Errorf("%s: separator %v", it.ID, it.Sep)
		}
	}
	if got := strings.Join(copyItems, ", "); got != "Path, Branch, Thread, Thread ID" {
		t.Fatalf("copy = %s", got)
	}
	// A read-only session can still copy and mark unread, nothing else:
	// the items §5.13 marks † are off, Filter by among them.
	a.envs[0].operate = false
	s = a.build(now)
	open := map[string]bool{"Mark unread": true, "Copy": true}
	for _, it := range s.Menu.Items {
		if it.Off == open[it.Label] {
			t.Errorf("%s read-only: off %v", it.Label, it.Off)
		}
		for _, c := range it.Children {
			if c.Off == open[it.Label] {
				t.Errorf("%s › %s read-only: off %v", it.Label, c.Label, c.Off)
			}
		}
	}
}

// Copy › Thread copies a link that opens the thread from anywhere, through
// T3's hosted app, its ids escaped down to their punctuation.
func TestHostedThreadLink(t *testing.T) {
	if got, want := hostedThreadLink("env-1", "a b.c!"), "https://app.t3.codes/threads/env-1/a%20b%2Ec%21"; got != want {
		t.Fatalf("link %s, want %s", got, want)
	}
}

// Opening a thread the project filter hides widens it to every project, so
// the thread's row shows; one in the scope leaves it (§5.8).
func TestOpenWidensScope(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "in tern", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "in atlas", ProjectID: "g", CreatedAt: ago(now, 2*time.Minute)},
	)
	a.envs[0].projects["g"] = t3.Project{ID: "g", Title: "atlas", WorkspaceRoot: "/srv/atlas"}
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "filter.project:m/p", Kind: "click"})
	a.openThread("m/1")
	if s := a.build(now); titles(s) != "in tern@workstation" {
		t.Fatalf("a thread in the scope changed it: %s", titles(s))
	}
	a.openThread("m/2")
	s := a.build(now)
	if titles(s) != "in tern@workstation, in atlas@workstation" || s.Sidebar.Scope != nil || s.Sidebar.ScopeKey != "" {
		t.Fatalf("after opening one it hides: %s, scope %v %q", titles(s), s.Sidebar.Scope, s.Sidebar.ScopeKey)
	}
}

// An open subagent has no row, so its parent's row is the open one, the
// row the list scrolls to and the one the arrows step from.
func TestOpenChildShowsAsItsParent(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "p", Title: "parent", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "c", Title: "child", CreatedAt: ago(now, time.Minute), Lineage: &t3.Lineage{ParentThreadID: "p", RelationshipToParent: "subagent"}},
		t3.Thread{ID: "o", Title: "other", CreatedAt: ago(now, 2*time.Minute)},
	)
	a.openID = "m/c"
	s := a.build(now)
	var open []string
	for _, r := range rows(s) {
		if r.Active {
			open = append(open, r.Title)
		}
	}
	if strings.Join(open, ", ") != "parent" || s.Sidebar.Open != "m/p" {
		t.Fatalf("open rows %v, the list's open row %q", open, s.Sidebar.Open)
	}
}

// A new thread left unsent shows above the list as a draft row, its
// project and its first line, a rule under it; while it's open it's the
// open row. Discarding drafts leaves the footer's undo notice, counting
// them, and undoing puts them back (§2.7, §5.15).
func TestDraftRowAndUndo(t *testing.T) {
	t.Cleanup(func() { delete(drafts, "draft"); delete(drafts, "m/2") })
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "one", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "two", CreatedAt: ago(now, 2*time.Minute)},
	)
	th := ui.NewTheme()
	a.draftEnv, a.draft = "m", "p"
	saveDraft("draft", "\nPlan the release\nwith notes")
	saveDraft("m/2", "a reply in two")
	if d := a.build(now).Sidebar.Items[0].Draft; d == nil || !d.Active {
		t.Fatalf("the open draft's row: %+v", d)
	}
	a.openThread("m/1")
	s := a.build(now)
	d := s.Sidebar.Items[0].Draft
	if d == nil || d.Active || d.Preview != "Plan the release" || d.Project != "tern" || !s.Sidebar.Items[1].Divider {
		t.Fatalf("the draft's row: %+v, then %+v", d, s.Sidebar.Items[1])
	}
	if got := listing(s); !strings.HasSuffix(got, "one, two, [Settled 0 false]") {
		t.Fatalf("the list under it: %s", got)
	}
	a.handle(th, ui.Action{ID: "discard:draft", Kind: "click"})
	a.handle(th, ui.Action{ID: "discard:m/2", Kind: "click"})
	s = a.build(now)
	if s.Sidebar.Items[0].Draft != nil || s.Sidebar.Undo != "Discarded 2" || rows(s)[1].Draft {
		t.Fatalf("after discarding both: draft row %v, notice %q, two's pen %v", s.Sidebar.Items[0].Draft, s.Sidebar.Undo, rows(s)[1].Draft)
	}
	a.handle(th, ui.Action{ID: "undo", Kind: "click"})
	s = a.build(now)
	if s.Sidebar.Items[0].Draft == nil || s.Sidebar.Undo != "" || !rows(s)[1].Draft {
		t.Fatalf("after undo: draft row %v, notice %q, two's pen %v", s.Sidebar.Items[0].Draft, s.Sidebar.Undo, rows(s)[1].Draft)
	}
}

// The list's first nine rows carry their jump keys, mod+1 to mod+9, in the
// order the keys open them (§5.12).
func TestJumpNumbers(t *testing.T) {
	now := time.Now()
	var threads []t3.Thread
	for i := range 11 {
		threads = append(threads, t3.Thread{ID: fmt.Sprint(i), Title: fmt.Sprint("t", i), CreatedAt: ago(now, time.Duration(i+1)*time.Minute)})
	}
	a := sideApp(t, threads...)
	for i, r := range rows(a.build(now)) {
		want := 0
		if i < 9 {
			want = i + 1
		}
		if r.Jump != want {
			t.Errorf("%s: jump %d, want %d", r.Title, r.Jump, want)
		}
	}
}

// Undoing a park puts the thread back as T3 does (planThreadUndo): a
// snooze is woken; a settle is un-settled, then pinned again in its place
// and snoozed again, as settling undid both.
func TestUndoPark(t *testing.T) {
	steps := func(cmds []map[string]any) string {
		var out []string
		for _, c := range cmds {
			s := c["type"].(string)
			if k, ok := c["orderKey"]; ok {
				s += " " + k.(string)
			}
			out = append(out, s)
		}
		return strings.Join(out, ", ")
	}
	pinned := &t3.Thread{ID: "t", PinnedAt: "2026-11-17T10:00:00Z", PinOrderKey: "a1", SnoozedUntil: "2026-11-18T09:00:00Z"}
	if got, want := steps(undoPark(pinned, true)), "thread.unsettle, thread.pin a1, thread.snooze"; got != want {
		t.Fatalf("undoing a settle: %s, want %s", got, want)
	}
	if got, want := steps(undoPark(pinned, false)), "thread.unsnooze"; got != want {
		t.Fatalf("undoing a snooze: %s, want %s", got, want)
	}
	if got, want := steps(undoPark(&t3.Thread{ID: "t"}, true)), "thread.unsettle"; got != want {
		t.Fatalf("undoing a plain settle: %s, want %s", got, want)
	}
}

// The snooze presets read as T3's do: durations from now, then today's
// evening, tomorrow and next week, soonest first; a duration landing on
// another day says which (resolveSnoozePresets).
func TestSnoozePresets(t *testing.T) {
	loc := time.FixedZone("test", 0)
	at := func(wd time.Weekday, h, m int) time.Time {
		d := time.Date(2026, 11, 18, h, m, 0, 0, loc) // a Wednesday
		return d.AddDate(0, 0, int(wd-d.Weekday()))
	}
	got := func(now time.Time) string {
		var out []string
		for _, c := range snoozeChoices(nil, now) {
			out = append(out, c.label+" "+c.when)
		}
		return strings.Join(out, "; ")
	}
	if g, want := got(at(time.Wednesday, 10, 0)), "In 1 hour 11:00 AM; In 3 hours 1:00 PM; This evening 6:00 PM; Tomorrow 9:00 AM; Next week Mon 9:00 AM"; g != want {
		t.Fatalf("Wednesday morning:\n got %s\nwant %s", g, want)
	}
	// Late, the evening has passed and three hours lands tomorrow.
	if g, want := got(at(time.Wednesday, 22, 30)), "In 1 hour 11:30 PM; In 3 hours Thu 1:30 AM; Tomorrow 9:00 AM; Next week Mon 9:00 AM"; g != want {
		t.Fatalf("Wednesday night:\n got %s\nwant %s", g, want)
	}
}

// Ctrl-click (Cmd on a Mac) picks rows, Shift-click a range of them; a
// plain click opens one and drops the selection. A selected row's menu is
// the selection's.
func TestSelection(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "one", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "two", CreatedAt: ago(now, 2*time.Minute)},
		t3.Thread{ID: "3", Title: "three", CreatedAt: ago(now, 3*time.Minute)},
	)
	th := ui.NewTheme()
	selected := func() string {
		var out []string
		for _, r := range rows(a.build(now)) {
			if r.Selected {
				out = append(out, r.Title)
			}
		}
		return strings.Join(out, " ")
	}
	a.build(now)
	a.handle(th, ui.Action{ID: "thread:m/1", Kind: "click", Short: true})
	a.handle(th, ui.Action{ID: "thread:m/3", Kind: "click", Shift: true})
	if got := selected(); got != "one two three" {
		t.Fatalf("after a range: %q", got)
	}
	a.handle(th, ui.Action{ID: "thread:m/2", Kind: "click", Short: true})
	if got := selected(); got != "one three" {
		t.Fatalf("after unpicking two: %q", got)
	}
	a.handle(th, ui.Action{ID: "thread:m/1", Kind: "context"})
	if got, want := menuLabels(a.build(now)), "Settle (2), Snooze (2), Regenerate titles (2), Mark unread (2), Delete (2)"; got != want {
		t.Fatalf("selection's menu:\n got %s\nwant %s", got, want)
	}
	a.handle(th, ui.Action{ID: "menus.close", Kind: "click"})
	a.handle(th, ui.Action{ID: "thread:m/2", Kind: "click"})
	if got := selected(); got != "" {
		t.Fatalf("a plain click kept the selection: %q", got)
	}
}

// The filter scopes the list to a project; with nothing active left, the
// list says so and offers all projects back (§2.9, §5.8).
func TestProjectFilter(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "in tern", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "in atlas", ProjectID: "g", CreatedAt: ago(now, 2*time.Minute)},
	)
	a.envs[0].projects["g"] = t3.Project{ID: "g", Title: "atlas", WorkspaceRoot: "/srv/atlas"}
	a.envs[0].projects["e"] = t3.Project{ID: "e", Title: "empty", WorkspaceRoot: "/srv/empty"}
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "filter", Kind: "click"})
	s := a.build(now)
	if s.Sidebar.Filter == nil || s.Sidebar.Filter.Items[0].Label != "All projects" || !s.Sidebar.Filter.Items[0].Selected {
		t.Fatalf("popup = %+v", s.Sidebar.Filter)
	}
	a.handle(th, ui.Action{ID: "filter.project:m/g", Kind: "click"})
	s = a.build(now)
	if s.Sidebar.Filter != nil || titles(s) != "in atlas@workstation" {
		t.Fatalf("scoped to atlas: %s (popup %v)", titles(s), s.Sidebar.Filter != nil)
	}
	if s.Sidebar.ScopeLabel != "Filter threads by project: atlas" || len(s.Sidebar.Scope) != 1 {
		t.Fatalf("trigger = %q %v", s.Sidebar.ScopeLabel, s.Sidebar.Scope)
	}
	a.handle(th, ui.Action{ID: "filter", Kind: "click"})
	a.handle(th, ui.Action{ID: "filter.project:m/e", Kind: "click"})
	s = a.build(now)
	if e := s.Sidebar.Empty; e == nil || e.Text != "No threads in empty yet" || e.Button != "Show all projects" {
		t.Fatalf("empty = %+v", s.Sidebar.Empty)
	}
	a.handle(th, ui.Action{ID: "filter.clear", Kind: "click"})
	if got := titles(a.build(now)); got != "in tern@workstation, in atlas@workstation" {
		t.Fatalf("after Show all projects: %s", got)
	}
}

// The filter's popup takes the keyboard as T3's combobox does: it opens on
// the scope's own row, the arrows move the highlight, Enter picks it,
// Shift+Enter adds a project and keeps the popup open, and typing
// highlights the first match.
func TestProjectFilterKeys(t *testing.T) {
	now := time.Now()
	a := sideApp(t, t3.Thread{ID: "1", Title: "in tern", CreatedAt: ago(now, time.Minute)})
	// atlas changed since tern's thread began, so it comes first.
	a.envs[0].projects["g"] = t3.Project{ID: "g", Title: "atlas", WorkspaceRoot: "/srv/atlas", UpdatedAt: ago(now, time.Second)}
	th := ui.NewTheme()
	lit := func() string {
		s := a.build(now)
		if s.Sidebar.Filter == nil {
			return "closed: " + s.Sidebar.ScopeLabel
		}
		for _, it := range s.Sidebar.Filter.Items {
			if it.Highlighted {
				return it.Label
			}
		}
		return "none"
	}
	key := func(name string, shift bool) {
		a.handle(th, ui.Action{ID: "filter.search", Kind: "key", Text: name, Shift: shift})
	}
	a.handle(th, ui.Action{ID: "filter", Kind: "click"})
	if got := lit(); got != "All projects" {
		t.Fatalf("opened on %q, want All projects", got)
	}
	key("↓", false)
	if got := lit(); got != "atlas" {
		t.Fatalf("ArrowDown to %q, want atlas", got)
	}
	a.handle(th, ui.Action{ID: "filter.search", Kind: "submit"})
	if got := lit(); got != "closed: Filter threads by project: atlas" {
		t.Fatalf("after Enter: %s", got)
	}
	a.handle(th, ui.Action{ID: "filter", Kind: "click"})
	if got := lit(); got != "atlas" {
		t.Fatalf("reopened on %q, want the scope's atlas", got)
	}
	key("↓", false)
	key("⏎", true)
	if got := lit(); got != "tern" {
		t.Fatalf("after Shift+Enter the popup shows %q, want it open on tern", got)
	}
	if got := a.build(now).Sidebar.ScopeLabel; got != "Filter threads by project: atlas, tern" {
		t.Fatalf("scope %q after Shift+Enter", got)
	}
	a.handle(th, ui.Action{ID: "filter.search", Kind: "change", Text: "ter"})
	if got := lit(); got != "tern" {
		t.Fatalf("typing highlights %q, want the first match", got)
	}
}

// The search lists the threads whose title or branch has every word, in
// place of the list; nothing matching says so.
func TestSearch(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "Fix the sidebar", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "Docs", Branch: "t3/sidebar-docs", CreatedAt: ago(now, 2*time.Minute)},
		t3.Thread{ID: "3", Title: "Something else", CreatedAt: ago(now, 3*time.Minute)},
	)
	th := ui.NewTheme()
	// Each frame's scene, built under the lock as Scene does: the search's
	// pause runs on a timer.
	scene := func() *ui.Scene {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.build(now)
	}
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "sidebar"})
	s := scene()
	var got []string
	for _, r := range s.Sidebar.Results {
		got = append(got, r.Title)
	}
	if strings.Join(got, ", ") != "Fix the sidebar, Docs" || len(s.Sidebar.Items) != 0 {
		t.Fatalf("results %v, items %d", got, len(s.Sidebar.Items))
	}
	a.handle(th, ui.Action{ID: "search", Kind: "key", Text: "↓"})
	if s = scene(); s.Sidebar.Highlight != 1 {
		t.Fatalf("highlight %d after ArrowDown", s.Sidebar.Highlight)
	}
	// With no match, the servers are asked once the typing pauses; till
	// they've answered (here, no one, with no socket), the list says
	// they're looking.
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "zebra"})
	if s = scene(); s.Sidebar.NoResults != "Searching thread messages…" {
		t.Fatalf("no results yet: %q", s.Sidebar.NoResults)
	}
	for end := time.Now().Add(5 * time.Second); s.Sidebar.NoResults != "No threads found" && time.Now().Before(end); s = scene() {
		time.Sleep(10 * time.Millisecond)
	}
	if s.Sidebar.NoResults != "No threads found" {
		t.Fatalf("no results: %q", s.Sidebar.NoResults)
	}
	// Enter opens the highlighted result, and the search clears.
	th.SetEditorText("search", "sidebar")
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "sidebar"})
	scene()
	a.handle(th, ui.Action{ID: "search", Kind: "key", Text: "↓"})
	scene()
	a.handle(th, ui.Action{ID: "search", Kind: "submit"})
	s = scene()
	open := ""
	for _, it := range s.Sidebar.Items {
		if it.Row != nil && it.Row.Active {
			open = it.Row.Title
		}
	}
	if len(s.Sidebar.Results) != 0 || open != "Docs" || th.EditorText("search") != "" {
		t.Fatalf("after Enter: %d results, %q open, %q typed", len(s.Sidebar.Results), open, th.EditorText("search"))
	}
}

// Threads only their messages match follow the title matches, in the order
// the servers' matches rank: whole words and word starts first, then the
// closer match, then the thread that keeps using the words; each shows its
// message with the words marked. Rows already shown keep their places as
// slower servers answer, and the highlight stays on its thread (§5.9).
func TestMessageSearch(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "Parser rewrite", CreatedAt: ago(now, time.Hour)},
		t3.Thread{ID: "2", Title: "Docs", CreatedAt: ago(now, 2*time.Hour)},
		t3.Thread{ID: "3", Title: "Old notes", CreatedAt: ago(now, 90*24*time.Hour)},
		t3.Thread{ID: "4", Title: "Release", CreatedAt: ago(now, 3*time.Hour)},
		t3.Thread{ID: "5", Title: "Tokens", CreatedAt: ago(now, 4*time.Hour)},
		t3.Thread{ID: "6", Title: "Late", CreatedAt: ago(now, 5*time.Hour)},
	)
	found := func(id, source, snippet string, quality float64, hits int) refMatch {
		return refMatch{t3.ThreadSearchMatch{ThreadID: id, Source: source, Snippet: snippet, Quality: &quality, Hits: &hits}, "m/" + id}
	}
	a.mu.Lock()
	a.search = "parser"
	a.msgSearch = msgSearch{query: "parser", waiting: 1, matches: []refMatch{
		found("4", "assistant", "an unparsered blob", 0.3, 1),
		found("2", "user", "Split the parsers out", 0.8, 1),
		found("3", "assistant", "the parser is slow", 1, 1),
		found("5", "assistant", "the parser again", 1, 20),
		found("1", "user", "parser", 1, 1),
		found("9", "user", "a thread this list hasn't", 1, 1),
	}}
	a.mu.Unlock()
	titles := func(s *ui.Scene) string {
		var got []string
		for _, r := range s.Sidebar.Results {
			got = append(got, r.Title)
		}
		return strings.Join(got, ", ")
	}
	s := a.build(now)
	if got := titles(s); got != "Parser rewrite, Tokens, Old notes, Docs, Release" {
		t.Fatalf("results %q", got)
	}
	if r := s.Sidebar.Results[0]; r.Excerpt != nil {
		t.Errorf("a title match shows a message: %+v", r.Excerpt)
	}
	if e := s.Sidebar.Results[2].Excerpt; e == nil || e.User || e.Snippet != "the parser is slow" || fmt.Sprint(e.Marks) != "[[4 10]]" {
		t.Errorf("the agent's message: %+v", e)
	}
	if e := s.Sidebar.Results[3].Excerpt; e == nil || !e.User || fmt.Sprint(e.Marks) != "[[10 16]]" {
		t.Errorf("the user's message: %+v", e)
	}

	// The highlight on Old notes; a slower server's better match goes
	// below the shown rows, and Release's match going leaves it on Old
	// notes.
	a.handle(ui.NewTheme(), ui.Action{ID: "result:2", Kind: "hover"})
	a.mu.Lock()
	a.msgSearch.waiting = 0
	a.msgSearch.matches = append(a.msgSearch.matches[1:], found("6", "user", "parser", 1, 50))
	a.mu.Unlock()
	s = a.build(now)
	if got := titles(s); got != "Parser rewrite, Tokens, Old notes, Docs, Late" || s.Sidebar.Highlight != 2 {
		t.Fatalf("after the late answer: %q, highlight %d", got, s.Sidebar.Highlight)
	}
	a.mu.Lock()
	a.msgSearch.matches = a.msgSearch.matches[:2] // only Docs' and Old notes'
	a.mu.Unlock()
	if s = a.build(now); titles(s) != "Parser rewrite, Old notes, Docs" || s.Sidebar.Highlight != 1 {
		t.Fatalf("after matches went: %q, highlight %d", titles(s), s.Sidebar.Highlight)
	}

	// Typing on shows the last answers while the servers look for the
	// longer query, their words still the ones marked; an unrelated query
	// shows none, and says the servers are looking until they've answered.
	a.mu.Lock()
	a.search = "parsers"
	a.mu.Unlock()
	if s = a.build(now); titles(s) != "Old notes, Docs" || fmt.Sprint(s.Sidebar.Results[0].Excerpt.Marks) != "[[4 10]]" {
		t.Fatalf("refining: %q", titles(s))
	}
	a.mu.Lock()
	a.search = "zebra"
	a.mu.Unlock()
	if s = a.build(now); len(s.Sidebar.Results) != 0 || s.Sidebar.NoResults != "Searching thread messages…" {
		t.Fatalf("looking: %d results, %q", len(s.Sidebar.Results), s.Sidebar.NoResults)
	}
	a.mu.Lock()
	a.msgSearch = msgSearch{query: "zebra"}
	a.mu.Unlock()
	if s = a.build(now); s.Sidebar.NoResults != "No threads found" {
		t.Fatalf("answered: %q", s.Sidebar.NoResults)
	}
	// Two letters search no messages.
	a.mu.Lock()
	a.search = "ze"
	a.mu.Unlock()
	if s = a.build(now); s.Sidebar.NoResults != "No threads found" {
		t.Fatalf("two letters: %q", s.Sidebar.NoResults)
	}
}

// A search finds threads by their pull requests' numbers and titles, and
// by a whole reference to one ("owner/repo#N", its link) or the thread's
// id; a result shows its pull request's number only when the query
// matches its pull requests (§3.11, ThreadSearchPullRequestNumber).
func TestSearchPullRequests(t *testing.T) {
	now := time.Now()
	link := t3.PullRequestLink{Number: 42, Repository: "acme/app", URL: "https://example.test/acme/app/pull/42", LinkedAt: ago(now, time.Hour),
		Snapshot: &t3.PullRequestSnapshot{State: "open", Title: "Faster startup"}}
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "Ship it", CreatedAt: ago(now, time.Minute), PullRequests: []t3.PullRequestLink{link}},
		t3.Thread{ID: "2", Title: "Ship notes", CreatedAt: ago(now, 2*time.Minute)},
		t3.Thread{ID: "thread-0042abc", Title: "Other", CreatedAt: ago(now, 3*time.Minute)},
	)
	for q, want := range map[string]string{
		"#42":         "Ship it 42",
		"startup":     "Ship it 42",
		"ship":        "Ship it, Ship notes",
		"acme/app#42": "Ship it 42",
		"https://example.test/acme/app/pull/42/files": "Ship it",
		"acme/app#4":  "",
		"thread-0042": "Other",
	} {
		a.mu.Lock()
		a.search = q
		s := a.build(now)
		a.mu.Unlock()
		var got []string
		for _, r := range s.Sidebar.Results {
			if r.PR != nil {
				got = append(got, r.Title+" "+r.PR.Text)
			} else {
				got = append(got, r.Title)
			}
		}
		if strings.Join(got, ", ") != want {
			t.Errorf("%q: %q, want %q", q, strings.Join(got, ", "), want)
		}
	}
}

// The servers get the query trimmed and cut to the 200 UTF-16 units they
// take, so a pasted wall of text searches by its start.
func TestMessageQuery(t *testing.T) {
	for q, want := range map[string]string{
		"  fix it  ":                   "fix it",
		strings.Repeat("a", 250):       strings.Repeat("a", 200),
		strings.Repeat("😀", 150):       strings.Repeat("😀", 100),
		strings.Repeat("a", 199) + "😀": strings.Repeat("a", 199),
	} {
		if got := msgQuery(q); got != want {
			t.Errorf("%.20q…: %d bytes, want %d", q, len(got), len(want))
		}
	}
}

// A message search needs a word of three letters, or two of Chinese,
// Japanese or Korean; quotes around the query and its case go, and curly
// apostrophes straighten (messageSearchTokens).
func TestMessageTokens(t *testing.T) {
	for q, want := range map[string]string{
		"fix the UI":      "[fix the]",
		"ab cd":           "[]",
		`"quoted words"`:  "[quoted words]",
		"don’t panic":     "[don't panic]",
		"修复 버그 x":         "[修复 버그]",
		"  spaced   out ": "[spaced out]",
	} {
		if got := fmt.Sprint(msgTokens(q)); got != want {
			t.Errorf("%q: %s, want %s", q, got, want)
		}
	}
}

// The words marked in a message (findSearchTokenRanges): whatever their
// case and accents; one under four letters only where a word starts (at
// an acronym's last capital, not after an apostrophe inside a word) or
// where it opens with punctuation; overlapping ones joined.
func TestMarkRanges(t *testing.T) {
	for _, c := range []struct{ s, q, want string }{
		{"Drafts use the fts index and FTS5", "fts", "[[15 18] [29 32]]"},
		{"Drafts use the fts index", "drafts", "[[0 6]]"},
		{"Café crème", "cafe", "[[0 4]]"},
		{"searching", "searching search", "[[0 9]]"},
		{"open main.ts now", ".ts", "[[9 12]]"},
		{"JSONSchema here", "sch", "[[4 7]]"},
		{"JSONSCHEMA here", "sch", "[]"},
		{"o’neil’sam sam", "sam", "[[11 14]]"},
		{"nothing", "ab", "[]"},
	} {
		if got := fmt.Sprint(markRanges(c.s, c.q)); got != c.want {
			t.Errorf("%q in %q: %s, want %s", c.q, c.s, got, c.want)
		}
	}
}

// A project's chosen icon is its monogram or its lucide icon in its colour;
// a lucide one carrying an older server's monogram is that monogram, and
// an emoji, which Gio can't set, gets the monogram made from the name
// (§3.7).
func TestProjectIcons(t *testing.T) {
	made := ui.ProjectIcon{Text: "TN", Color: "amber"}
	for _, c := range []struct {
		icon *t3.ProjectIcon
		want ui.ProjectIcon
	}{
		{&t3.ProjectIcon{Kind: "monogram", Text: "Q1", Color: "teal"}, ui.ProjectIcon{Text: "Q1", Color: "teal"}},
		{&t3.ProjectIcon{Kind: "lucide", Name: "rocket", Color: "blue"}, ui.ProjectIcon{Lucide: "rocket", Color: "blue"}},
		{&t3.ProjectIcon{Kind: "lucide", Name: "rocket", Color: "blue", MonogramText: "RK"}, ui.ProjectIcon{Text: "RK", Color: "blue"}},
		{&t3.ProjectIcon{Kind: "emoji", Emoji: "🚀"}, made},
		{nil, made},
	} {
		if got := projectIcon(t3.Project{Title: "tern", ProjectIcon: c.icon}); got != c.want {
			t.Errorf("%+v: %+v, want %+v", c.icon, got, c.want)
		}
	}
}

// The filter lists pinned projects first, then the rest, each most
// recently active first: by its threads' newest user message, else by when
// the project itself changed; ties by title, whatever its case (§5.3).
func TestFilterOrder(t *testing.T) {
	now := time.Now()
	a := sideApp(t, t3.Thread{ID: "1", Title: "in tern", CreatedAt: ago(now, 3*time.Hour), LatestUserMessageAt: ago(now, time.Hour)})
	e := a.envs[0]
	e.projects["b"] = t3.Project{ID: "b", Title: "beta", WorkspaceRoot: "/srv/beta", PinnedAt: ago(now, time.Hour), UpdatedAt: ago(now, 48*time.Hour)}
	e.projects["g"] = t3.Project{ID: "g", Title: "gamma", WorkspaceRoot: "/srv/gamma", UpdatedAt: ago(now, 30*time.Minute)}
	e.projects["z"] = t3.Project{ID: "z", Title: "Zeta", WorkspaceRoot: "/srv/zeta", UpdatedAt: ago(now, 5*time.Hour)}
	e.projects["e"] = t3.Project{ID: "e", Title: "eta", WorkspaceRoot: "/srv/eta", UpdatedAt: ago(now, 5*time.Hour)}
	e.projects["d"] = t3.Project{ID: "d", Title: "delta", WorkspaceRoot: "/srv/delta", ArchivedAt: ago(now, time.Hour)}
	a.handle(ui.NewTheme(), ui.Action{ID: "filter", Kind: "click"})
	var got []string
	for _, it := range a.build(now).Sidebar.Filter.Items {
		got = append(got, it.Label)
	}
	if want := "All projects, beta, gamma, tern, eta, Zeta"; strings.Join(got, ", ") != want {
		t.Fatalf("filter order:\n got %s\nwant %s", strings.Join(got, ", "), want)
	}
}

// Monograms take T3's letters and its 18 colours by the name's hash
// (projectIdentity.ts).
func TestMonogram(t *testing.T) {
	for name, want := range map[string]string{"tern": "TN", "north star": "NS", "app2": "A2", "x": "XX", "!!!": "PR"} {
		if got := monogram(name); got != want {
			t.Errorf("%q: %q, want %q", name, got, want)
		}
	}
	if got := monogramColor("tern"); got != "amber" {
		t.Errorf("tern's colour: %q, want amber", got)
	}
}

// The selection's menu works on its threads as they were when it was
// picked: one the servers drop meanwhile doesn't stop the rest, as T3's
// bulk actions run on the selection they were handed.
func TestSelectionMenuOutlivesItsThreads(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "one", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "two", CreatedAt: ago(now, 2*time.Minute)},
	)
	th := ui.NewTheme()
	a.build(now)
	a.handle(th, ui.Action{ID: "thread:m/1", Kind: "click", Short: true})
	a.handle(th, ui.Action{ID: "thread:m/2", Kind: "click", Short: true})
	a.handle(th, ui.Action{ID: "thread:m/1", Kind: "context"})
	if got := menuLabels(a.build(now)); !strings.Contains(got, "Regenerate titles (2)") {
		t.Fatalf("selection's menu: %s", got)
	}
	// The second thread goes as the first one's command goes out, which,
	// with no connection, is when the banner says so.
	a.redraw = func() {
		a.mu.Lock()
		delete(a.envs[0].threads, "2")
		a.mu.Unlock()
	}
	a.handle(th, ui.Action{ID: "ctx:bulk-regenerate-title", Kind: "click"})
	a.redraw = nil
	if got := a.build(now).Error; got != "workstation isn't connected" {
		t.Fatalf("banner %q, want the commands' failure", got)
	}
}

// mod+Shift+S and mod+Shift+P read the open thread's machine under the
// lock, as its config can land from the network at any moment (the race
// detector, which CI runs, holds the test to it).
func TestOpenThreadShortcutsTakeTheLock(t *testing.T) {
	a := sideApp(t, t3.Thread{ID: "1", Title: "one", CreatedAt: ago(time.Now(), time.Minute)})
	a.openID = "m/1"
	th := ui.NewTheme()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 200 {
			a.mu.Lock()
			a.envs[0].config = &t3.ServerConfig{}
			a.mu.Unlock()
		}
	}()
	for range 200 {
		a.handle(th, ui.Action{ID: "thread.pin", Kind: "click"})
	}
	<-done
}

// sceneAt builds the scene under the lock, as Scene does: the timers that
// redraw it take the lock too.
func sceneAt(a *App, now time.Time) *ui.Scene {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.build(now)
}

// redraws is a channel that gets a value each time a asks for a new scene.
func redraws(a *App) chan struct{} {
	c := make(chan struct{}, 16)
	a.redraw = func() {
		select {
		case c <- struct{}{}:
		default:
		}
	}
	return c
}

// waitRedraw waits for a new scene asked for no sooner than at.
func waitRedraw(t *testing.T, c chan struct{}, at time.Time, what string) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Fatalf("no new scene for %s", what)
	}
	if early := time.Until(at); early > 0 {
		t.Fatalf("a new scene for %s %v early", what, early)
	}
}

// A snooze ends when it's due, not at the next minute's tick: the list asks
// for a new scene at the wake (useSnoozeWakeClock), and the thread is back
// in it.
func TestSnoozeWakesOnTime(t *testing.T) {
	now := time.Now()
	wake := now.Add(300 * time.Millisecond)
	a := sideApp(t, t3.Thread{ID: "1", Title: "snoozed", CreatedAt: ago(now, time.Hour), SnoozedAt: ago(now, time.Minute), SnoozedUntil: wake.UTC().Format(time.RFC3339Nano)})
	a.shelfOpen["snoozed"] = true
	redrawn := redraws(a)
	if got, want := listing(sceneAt(a, time.Now())), "[Snoozed 1 true], snoozed, [Settled 0 false]"; got != want {
		t.Fatalf("snoozed: %s, want %s", got, want)
	}
	waitRedraw(t, redrawn, wake, "the wake")
	if got, want := listing(sceneAt(a, time.Now())), "snoozed, [Settled 0 false]"; got != want {
		t.Fatalf("woken: %s, want %s", got, want)
	}
}

// The snooze clock, held back while a message from the last two minutes
// hasn't started its turn, comes back as that grace runs out, not at the
// next minute's tick (SidebarThreadRow.tsx's snooze gate).
func TestSnoozeClockComesBackOnTime(t *testing.T) {
	now := time.Now()
	sent := now.Add(-queuedTurnGrace + 300*time.Millisecond)
	a := sideApp(t, t3.Thread{ID: "1", Title: "one", CreatedAt: ago(now, time.Hour), LatestUserMessageAt: sent.UTC().Format(time.RFC3339Nano)})
	redrawn := redraws(a)
	if rows(sceneAt(a, time.Now()))[0].CanSnooze {
		t.Fatal("the clock shows with a turn about to start")
	}
	waitRedraw(t, redrawn, sent.Add(queuedTurnGrace), "the grace's end")
	if !rows(sceneAt(a, time.Now()))[0].CanSnooze {
		t.Fatal("the clock isn't back once the grace is up")
	}
}

// A connection that can only read still gets Settle, Un-settle and Wake,
// as T3 offers them wherever the server's config has loaded, and a click
// on one says why it can't; the snooze clock and Unpin need write access
// (SidebarThreadRow.tsx: canOperateThread).
func TestReadOnlyRows(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "c", Title: "card", PinnedAt: ago(now, time.Hour), CreatedAt: ago(now, time.Hour)},
		t3.Thread{ID: "z", Title: "snoozed", SnoozedUntil: now.Add(time.Hour).UTC().Format(time.RFC3339Nano), CreatedAt: ago(now, time.Hour)},
		t3.Thread{ID: "s", Title: "settled", SettledOverride: "settled", SettledAt: ago(now, time.Hour), CreatedAt: ago(now, time.Hour)},
	)
	a.envs[0].operate = false
	a.shelfOpen["snoozed"], a.shelfOpen["settled"] = true, true
	rs := rows(a.build(now))
	if len(rs) != 3 {
		t.Fatalf("%d rows, want 3", len(rs))
	}
	for _, r := range rs {
		if !r.CanSettle || r.CanSnooze || r.CanUnpin {
			t.Errorf("%s: settle %v, snooze %v, unpin %v; want Settle (or Un-settle, Wake) alone", r.Title, r.CanSettle, r.CanSnooze, r.CanUnpin)
		}
	}
	for id, want := range map[string]string{
		"settle:m/c":   "Failed to settle thread",
		"unsettle:m/s": "Failed to un-settle thread",
		"wake:m/z":     "Failed to wake thread",
	} {
		a.handle(ui.NewTheme(), ui.Action{ID: id, Kind: "click"})
		if got := a.build(now).Error; got != want+": this connection cannot change threads or projects." {
			t.Errorf("%s: banner %q", id, got)
		}
	}
}

// A card's snooze menu closes when the card stops offering the clock (a
// queued turn) or leaves the list (onto the folded Working shelf), and
// stays closed when the clock comes back (SidebarThreadRow.tsx).
func TestSnoozeMenuClosesWithItsCard(t *testing.T) {
	now := time.Now()
	a := sideApp(t, t3.Thread{ID: "1", Title: "one", CreatedAt: ago(now, time.Hour)})
	th := ui.NewTheme()
	a.build(now)
	a.handle(th, ui.Action{ID: "snooze:m/1", Kind: "click"})
	if a.build(now).Sidebar.Snooze == nil {
		t.Fatal("the menu didn't open")
	}
	// A message queues a turn: the clock goes for two minutes.
	one := a.envs[0].threads["1"]
	one.LatestUserMessageAt = ago(now, 0)
	if a.build(now).Sidebar.Snooze != nil {
		t.Fatal("the menu stayed with the clock gone")
	}
	later := now.Add(3 * time.Minute)
	if s := a.build(later); s.Sidebar.Snooze != nil || !rows(s)[0].CanSnooze {
		t.Fatalf("with the clock back: menu %v, clock %v; want the clock and no menu", s.Sidebar.Snooze, rows(s)[0].CanSnooze)
	}
	// Open again, then the run starts and the card goes to the Working
	// shelf, which is folded.
	a.handle(th, ui.Action{ID: "snooze:m/1", Kind: "click"})
	if a.build(later).Sidebar.Snooze == nil {
		t.Fatal("the menu didn't open again")
	}
	one.LatestRunID, one.Status, one.ActiveRunID, one.ActivityRunStatus = "r", "running", "r", "running"
	if s := a.build(later); s.Sidebar.Snooze != nil {
		t.Fatalf("the menu stayed with its card on the folded shelf: %s", listing(s))
	}
}

// An undo runs what it took off the notice, even when parks land while it
// runs: they join the notice in the space the undone ones left, and don't
// take their place in the undo (showThreadUndoNotice.ts).
func TestUndoKeepsItsGroup(t *testing.T) {
	a := sideApp(t)
	var ran []string
	undo := func(name string) func(*ui.Theme) { return func(*ui.Theme) { ran = append(ran, name) } }
	a.noteUndo("Settled", func(*ui.Theme) {
		ran = append(ran, "one")
		// Two snoozes the servers take as the undo runs.
		a.noteUndo("Snoozed", undo("three"))
		a.noteUndo("Snoozed", undo("four"))
	})
	a.noteUndo("Settled", undo("two"))
	if got := a.build(time.Now()).Sidebar.Undo; got != "Settled 2" {
		t.Fatalf("notice %q, want Settled 2", got)
	}
	a.undoLast(ui.NewTheme())
	if got := strings.Join(ran, " "); got != "one two" {
		t.Fatalf("the undo ran %q, want one two", got)
	}
	if got := a.build(time.Now()).Sidebar.Undo; got != "Snoozed 2" {
		t.Fatalf("notice %q after the undo, want Snoozed 2", got)
	}
}

// The notice lasts 5s from the last park in it: the earlier one's timer,
// caught already firing as the later park arrives, doesn't end it early.
func TestUndoNoticeLasts(t *testing.T) {
	a := sideApp(t)
	a.noteUndo("Settled", func(*ui.Theme) {})
	a.noteUndo("Settled", func(*ui.Theme) {})
	a.endUndo(1) // the first park's timer, running late
	if got := a.build(time.Now()).Sidebar.Undo; got != "Settled 2" {
		t.Fatalf("notice %q after the first park's timer, want Settled 2", got)
	}
	a.endUndo(2)
	if got := a.build(time.Now()).Sidebar.Undo; got != "" {
		t.Fatalf("notice %q after the last park's timer, want none", got)
	}
}

// Enter in a search field that Escape emptied, or that opening a result
// emptied, opens nothing: the results were the last query's, and with
// nothing typed there are none (Sidebar.tsx, §5.9).
func TestEnterInAnEmptySearch(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "1", Title: "Fix the sidebar", CreatedAt: ago(now, time.Minute)},
		t3.Thread{ID: "2", Title: "Docs", CreatedAt: ago(now, 2*time.Minute)},
	)
	th := ui.NewTheme()
	open := func() string {
		for _, r := range rows(sceneAt(a, now)) {
			if r.Active {
				return r.Title
			}
		}
		return ""
	}
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "sidebar"})
	sceneAt(a, now)
	// Escape and Enter in one frame, before a scene is built…
	a.handle(th, ui.Action{ID: "search", Kind: "key", Text: "⎋"})
	a.handle(th, ui.Action{ID: "search", Kind: "submit"})
	if got := open(); got != "" {
		t.Fatalf("Enter after Escape opened %q", got)
	}
	// …and in frames of their own.
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "sidebar"})
	sceneAt(a, now)
	a.handle(th, ui.Action{ID: "search", Kind: "key", Text: "⎋"})
	sceneAt(a, now)
	a.handle(th, ui.Action{ID: "search", Kind: "submit"})
	if got := open(); got != "" {
		t.Fatalf("Enter in the emptied field opened %q", got)
	}
}

// Snoozed threads that wake at the same moment, as a bulk snooze leaves
// them, keep one order from build to build (§5.3), and so do the keys and
// jump numbers that go by it.
func TestSnoozedTiesKeepTheirOrder(t *testing.T) {
	now := time.Now()
	wake := now.Add(time.Hour).UTC().Format(time.RFC3339Nano)
	var threads []t3.Thread
	for _, id := range []string{"c", "a", "e", "b", "d"} {
		threads = append(threads, t3.Thread{ID: id, Title: id, SnoozedUntil: wake, CreatedAt: ago(now, time.Hour)})
	}
	a := sideApp(t, threads...)
	a.shelfOpen["snoozed"] = true
	for range 20 {
		if got, want := listing(a.build(now)), "[Snoozed 5 true], a, b, c, d, e, [Settled 0 false]"; got != want {
			t.Fatalf("snoozed:\n got %s\nwant %s", got, want)
		}
	}
}

// Until a machine's server config has loaded, its threads aren't snoozed
// or settled in the list: they're pinned or active, as T3 files them
// (Sidebar.tsx, §5.2 step 2).
func TestShelvesWaitForConfig(t *testing.T) {
	now := time.Now()
	a := sideApp(t,
		t3.Thread{ID: "z", Title: "snoozed", SnoozedUntil: now.Add(time.Hour).UTC().Format(time.RFC3339Nano), CreatedAt: ago(now, time.Hour)},
		t3.Thread{ID: "s", Title: "settled", SettledOverride: "settled", SettledAt: ago(now, time.Minute), CreatedAt: ago(now, 2*time.Hour)},
	)
	a.envs[0].config = nil
	s := a.build(now)
	if got, want := listing(s), "snoozed, settled, [Settled 0 false]"; got != want {
		t.Fatalf("before the config: %s, want %s", got, want)
	}
	for _, r := range rows(s) {
		if r.Slim {
			t.Errorf("%s is a slim row before the config loaded", r.Title)
		}
	}
	a.envs[0].config = &t3.ServerConfig{}
	if got, want := listing(a.build(now)), "[Snoozed 1 false], [Settled 1 false]"; got != want {
		t.Fatalf("with the config: %s, want %s", got, want)
	}
}

// The Working shelf orders by the user's last message, else by when the
// thread began: a thread the user never wrote in, which current servers
// send as null, goes by its creation, and only one from an older server,
// which leaves the field out, by its run's request
// (sortWorkingThreadsBySend).
func TestWorkingOrderNeverWritten(t *testing.T) {
	now := time.Now()
	decode := func(id string, created, requested time.Duration, authored string) t3.Thread {
		raw := fmt.Sprintf(`{"id":%q,"title":%q,"createdAt":%q,"latestRunId":"r","status":"running","activeRunId":"r",`+
			`"activityRunStatus":"running","latestRunRequestedAt":%q%s}`, id, id, ago(now, created), ago(now, requested), authored)
		var th t3.Thread
		if err := json.Unmarshal([]byte(raw), &th); err != nil {
			t.Fatal(err)
		}
		return th
	}
	a := sideApp(t,
		decode("never wrote", 2*time.Hour, time.Minute, `,"latestUserAuthoredMessageAt":null`),
		decode("wrote", time.Hour, time.Hour, fmt.Sprintf(`,"latestUserAuthoredMessageAt":%q`, ago(now, 30*time.Minute))),
		decode("older server", 3*time.Hour, 5*time.Minute, ""),
	)
	a.shelfOpen["working"] = true
	if got, want := listing(a.build(now)), "[Working 3 true], older server, wrote, never wrote, [Settled 0 false]"; got != want {
		t.Fatalf("working:\n got %s\nwant %s", got, want)
	}
}

// A snooze preset tern can't read, as a newer server's kind or day would
// be, is left out of the menu, and the rest still show
// (resolveSnoozePresets).
func TestSnoozePresetsTernCantRead(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC) // a Wednesday
	e := &envState{config: &t3.ServerConfig{}}
	e.config.Settings.SnoozePresets = []t3.SnoozePreset{
		{Kind: "until-idle"}, {Kind: "at", Time: "09:00"}, {Kind: "at", Day: "someday", Time: "09:00"},
		{Kind: "at", Day: "friday", Time: "late"}, {Kind: "after", Minutes: -5}, {Kind: "at", Day: "friday", Time: "09:00"},
	}
	var got []string
	for _, c := range snoozeChoices(e, now) {
		got = append(got, c.label+" "+c.when)
	}
	if want := "Friday 9:00 AM"; strings.Join(got, "; ") != want {
		t.Fatalf("menu %q, want %q", strings.Join(got, "; "), want)
	}
}

// A run timer counts to the whole second, as T3's second tick does: a
// scene built late in a second shows that second, not the next (§5.5).
func TestRunTimerCountsWholeSeconds(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, int(900*time.Millisecond), time.UTC)
	start := time.Date(2026, 11, 17, 11, 59, 0, int(500*time.Millisecond), time.UTC)
	a := sideApp(t, t3.Thread{ID: "w", Title: "w", CreatedAt: ago(now, time.Hour), LatestRunID: "r", Status: "running", ActiveRunID: "r",
		ActivityRunStatus: "running", ActivityRunStartedAt: start.Format(time.RFC3339Nano)})
	a.shelfOpen["working"] = true
	if r := rows(a.build(now)); len(r) != 1 || r[0].Timer != "59s" {
		t.Fatalf("rows %+v, want one timing 59s", r)
	}
}

// The open thread's read marker moves as T3's tracker moves it while the
// thread is shown: the open goes with the first visit; each update after
// moves it to the thread's updatedAt, at most every 10s while a run
// streams; and a run that ends unseen moves it at once, so leaving right
// after doesn't leave the row saying Done (threadVisits.ts).
func TestVisits(t *testing.T) {
	t0 := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) string { return t0.Add(d).Format(time.RFC3339Nano) }
	th := &t3.Thread{ID: "t", UpdatedAt: at(0), LatestRunID: "r", Status: "running", LastVisitedAt: at(-time.Hour)}
	v := visits{ref: "m/t", open: true}
	step := func(now time.Duration, want string, wantOpen bool, wantRetry time.Duration) {
		t.Helper()
		got, opened, retry := v.next(th, t0.Add(now))
		var r time.Duration
		if !retry.IsZero() {
			r = retry.Sub(t0)
		}
		if got != want || opened != wantOpen || r != wantRetry {
			t.Fatalf("at %v: visit %q open %v retry %v, want %q %v %v", now, got, opened, r, want, wantOpen, wantRetry)
		}
	}
	step(0, at(0), true, 0)
	step(time.Second, "", false, 0) // nothing new
	th.LastVisitedAt = at(0)        // the server's echo
	th.UpdatedAt = at(2 * time.Second)
	step(2*time.Second, "", false, 10*time.Second) // streaming: held back
	th.UpdatedAt = at(5 * time.Second)
	step(5*time.Second, "", false, 10*time.Second)
	step(10*time.Second, at(5*time.Second), false, 0)
	th.LastVisitedAt = at(5 * time.Second)
	// The run ends 2s on: that goes at once.
	th.Status, th.LatestRunCompletedAt, th.UpdatedAt = "completed", at(12*time.Second), at(12*time.Second)
	step(12*time.Second, at(12*time.Second), false, 0)
	// Marked unread while it's open: the marker stays where the user put it.
	th.LastVisitedAt = at(-time.Hour)
	step(20*time.Second, "", false, 0)
}
