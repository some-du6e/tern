package core

import (
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
	// A read-only session can still copy and mark unread, nothing else.
	a.envs[0].operate = false
	s = a.build(now)
	for _, it := range s.Menu.Items {
		if it.Off != needsOperate(it.ID) {
			t.Errorf("%s read-only: off %v", it.ID, it.Off)
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
	a.envs[0].projects["g"] = t3.Project{ID: "g", Title: "atlas", WorkspaceRoot: "/srv/atlas"}
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
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "sidebar"})
	s := a.build(now)
	var got []string
	for _, r := range s.Sidebar.Results {
		got = append(got, r.Title)
	}
	if strings.Join(got, ", ") != "Fix the sidebar, Docs" || len(s.Sidebar.Items) != 0 {
		t.Fatalf("results %v, items %d", got, len(s.Sidebar.Items))
	}
	a.handle(th, ui.Action{ID: "search", Kind: "key", Text: "↓"})
	if s = a.build(now); s.Sidebar.Highlight != 1 {
		t.Fatalf("highlight %d after ArrowDown", s.Sidebar.Highlight)
	}
	a.handle(th, ui.Action{ID: "search", Kind: "change", Text: "zebra"})
	if s = a.build(now); s.Sidebar.NoResults != "No threads found" {
		t.Fatalf("no results: %q", s.Sidebar.NoResults)
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
