package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// testApp is an App with two environments loaded by hand, no network.
func testApp(t *testing.T) *App {
	t.Helper()
	store, err := t3.OpenStoreAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewWithStore(store, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	at := func(d time.Duration) string { return now.Add(-d).Format(time.RFC3339Nano) }
	mk := func(id, label string, projects map[string]string, threads ...t3.Thread) *envState {
		e := newEnv(t3.Environment{EnvironmentID: id, Label: label}, false)
		e.status = "online"
		for pid, title := range projects {
			e.projects[pid] = t3.Project{ID: pid, Title: title, WorkspaceRoot: "/root/code/" + title}
		}
		for i := range threads {
			th := threads[i]
			e.threads[th.ID] = &th
		}
		return e
	}
	a.envs = []*envState{
		mk("swiss", "swiss", map[string]string{"p1": "goog", "p2": "tern"},
			t3.Thread{ID: "t1", Title: "On swiss", ProjectID: "p1", CreatedAt: at(time.Minute), LatestUserMessageAt: at(time.Minute)},
			t3.Thread{ID: "same", Title: "Same id on swiss", ProjectID: "p2", CreatedAt: at(3 * time.Minute), LatestUserMessageAt: at(3 * time.Minute)}),
		mk("mac", "MacBook", map[string]string{"p9": "horizon"},
			t3.Thread{ID: "same", Title: "Same id on the Mac", ProjectID: "p9", CreatedAt: at(2 * time.Minute), LatestUserMessageAt: at(2 * time.Minute)}),
	}
	a.booted, a.signedIn = true, true
	return a
}

// rows is the sidebar's thread rows, top to bottom.
func rows(s *ui.Scene) []*ui.ThreadRow {
	var out []*ui.ThreadRow
	for _, it := range s.Sidebar.Items {
		if it.Row != nil {
			out = append(out, it.Row)
		}
	}
	return out
}

func titles(s *ui.Scene) string {
	var out []string
	for _, r := range rows(s) {
		out = append(out, r.Title+"@"+r.Machine)
	}
	return strings.Join(out, ", ")
}

func TestInboxMixesEnvironments(t *testing.T) {
	a := testApp(t)
	s := a.build(time.Now())
	if got, want := titles(s), "On swiss@swiss, Same id on the Mac@MacBook, Same id on swiss@swiss"; got != want {
		t.Fatalf("active = %s, want %s", got, want)
	}
	rs := rows(s)
	if rs[1].ID != "mac/same" || rs[2].ID != "swiss/same" {
		t.Fatalf("refs = %q, %q", rs[1].ID, rs[2].ID)
	}
	// Two machines have threads, so each row shows its machine's icon.
	if !rs[0].Remote || !rs[1].Remote {
		t.Fatal("rows on a list mixing machines should show their machine")
	}
	if s.Run != nil {
		t.Fatal("nothing is open, so there should be no details panel")
	}
}

func TestOpenThreadOnEitherMachine(t *testing.T) {
	a := testApp(t)
	a.openThread("mac/same")
	s := a.build(time.Now())
	if s.Title != "Same id on the Mac" || s.Project != "horizon" {
		t.Fatalf("open = %q in %q", s.Title, s.Project)
	}
	if rs := rows(s); !rs[1].Active || rs[2].Active {
		t.Fatal("only the Mac's thread should be the open row")
	}
}

func TestDraftProjectMenu(t *testing.T) {
	a := testApp(t)
	a.newDraft()
	s := a.build(time.Now())
	if !s.PickProject || !strings.HasPrefix(s.Project, "goog") {
		t.Fatalf("draft in %q (pick=%v)", s.Project, s.PickProject)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "project", Kind: "click"})
	s = a.build(time.Now())
	if s.Projects == nil || len(s.Projects.Groups) != 2 {
		t.Fatalf("project menu = %+v", s.Projects)
	}
	g := s.Projects.Groups[1]
	if g.Label != "MacBook" || g.Choices[0].Label != "horizon" || g.Choices[0].Desc != "~/code/horizon" {
		t.Fatalf("mac group = %+v", g)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "project:1:0", Kind: "click"})
	s = a.build(time.Now())
	if s.Projects != nil || s.Project != "horizon  ·  MacBook" || a.draftEnv != "mac" {
		t.Fatalf("after pick: %q env=%q menu=%v", s.Project, a.draftEnv, s.Projects != nil)
	}
}

func TestSignOutIsBehindAMenu(t *testing.T) {
	a := testApp(t)
	a.handle(ui.NewTheme(), ui.Action{ID: "account", Kind: "click"})
	s := a.build(time.Now())
	if s.Account == nil || !a.signedIn {
		t.Fatal("the account button should open a menu, not sign out")
	}
}

// The footer's machine button lists the machines with their status;
// picking one only closes the menu.
func TestMachinesMenu(t *testing.T) {
	a := testApp(t)
	a.handle(ui.NewTheme(), ui.Action{ID: "machines", Kind: "click"})
	s := a.build(time.Now())
	if s.Machines == nil || len(s.Machines.Groups[0].Choices) != 2 || s.Account != nil {
		t.Fatalf("machines = %+v", s.Machines)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "machines:0:0", Kind: "click"})
	if s = a.build(time.Now()); s.Machines != nil || !a.signedIn {
		t.Fatalf("after a pick: menu=%v signedIn=%v", s.Machines != nil, a.signedIn)
	}
}

// menuLabels is the open context menu's labels.
func menuLabels(s *ui.Scene) string {
	var labels []string
	for _, it := range s.Menu.Items {
		labels = append(labels, it.Label)
	}
	return strings.Join(labels, ", ")
}

func TestThreadMenu(t *testing.T) {
	a := testApp(t)
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "thread:mac/same", Kind: "context"})
	s := a.build(time.Now())
	if s.Menu == nil {
		t.Fatal("right-click should open the thread menu")
	}
	// Without the server's config, the menu has no pin, settle or snooze.
	if got, want := menuLabels(s), "Rename thread, Mark unread, Filter by horizon, Copy, Archive thread, Delete"; got != want {
		t.Fatalf("menu = %s, want %s", got, want)
	}
	// Delete asks twice.
	a.handle(th, ui.Action{ID: "ctx:delete", Kind: "click"})
	s = a.build(time.Now())
	if s.Menu == nil || s.Menu.Items[len(s.Menu.Items)-1].Label != "Click again to delete" {
		t.Fatalf("after one click: %+v", s.Menu)
	}
	// The Mac has no socket here, so the second click reports that.
	a.handle(th, ui.Action{ID: "ctx:delete", Kind: "click"})
	s = a.build(time.Now())
	if s.Menu != nil || !strings.Contains(s.Error, "isn't connected") {
		t.Fatalf("after two clicks: menu=%v error=%q", s.Menu != nil, s.Error)
	}
}

func TestDetailsPopover(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	a.handle(ui.NewTheme(), ui.Action{ID: "details", Kind: "click"})
	s := a.build(time.Now())
	if !s.Details || s.Run == nil || s.Run.Machine != "swiss" {
		t.Fatalf("details = %v, run = %+v", s.Details, s.Run)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "menus.close", Kind: "click"})
	if s = a.build(time.Now()); s.Details {
		t.Fatal("a click outside should close the details")
	}
}

// With nothing open there's no "pick a thread" stop: the composer is a new
// thread in the latest project.
func TestNothingOpenIsANewThread(t *testing.T) {
	a := testApp(t)
	s := a.build(time.Now())
	if a.draft != "p1" || a.draftEnv != "swiss" || strings.Contains(s.Empty, "Pick a thread") {
		t.Fatalf("draft %q on %q, empty %q", a.draft, a.draftEnv, s.Empty)
	}
	if !s.PickProject {
		t.Fatal("the draft should offer the project picker")
	}
}
