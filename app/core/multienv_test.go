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
			e.projects[pid] = t3.Project{ID: pid, Title: title, WorkspaceRoot: "/home/dev/code/" + title}
		}
		for i := range threads {
			th := threads[i]
			e.threads[th.ID] = &th
		}
		return e
	}
	a.envs = []*envState{
		mk("studio", "studio", map[string]string{"p1": "atlas", "p2": "tern"},
			t3.Thread{ID: "t1", Title: "On studio", ProjectID: "p1", LatestUserMessageAt: at(time.Minute)},
			t3.Thread{ID: "same", Title: "Same id on studio", ProjectID: "p2", LatestUserMessageAt: at(3 * time.Minute)}),
		mk("mac", "MacBook", map[string]string{"p9": "nimbus"},
			t3.Thread{ID: "same", Title: "Same id on the Mac", ProjectID: "p9", LatestUserMessageAt: at(2 * time.Minute)}),
	}
	a.booted, a.signedIn = true, true
	return a
}

func titles(rows []ui.ThreadRow) string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Title+"@"+r.Env)
	}
	return strings.Join(out, ", ")
}

func TestInboxMixesEnvironments(t *testing.T) {
	a := testApp(t)
	s := a.build(time.Now())
	if got, want := titles(s.Active), "On studio@studio, Same id on the Mac@MacBook, Same id on studio@studio"; got != want {
		t.Fatalf("active = %s, want %s", got, want)
	}
	if s.Active[1].ID != "mac/same" || s.Active[2].ID != "studio/same" {
		t.Fatalf("refs = %q, %q", s.Active[1].ID, s.Active[2].ID)
	}
	if s.Header == nil || s.Header.Server || s.Card == nil || s.Card.Run.Locked {
		t.Fatal("nothing is open, so the header and card are a new thread's")
	}
}

func TestOpenThreadOnEitherMachine(t *testing.T) {
	a := testApp(t)
	a.openThread("mac/same")
	s := a.build(time.Now())
	if s.Title != "Same id on the Mac" || s.Header.Project != "nimbus" {
		t.Fatalf("open = %q in %q", s.Title, s.Header.Project)
	}
	if !s.Active[1].Selected || s.Active[2].Selected {
		t.Fatal("only the Mac's thread should be selected")
	}
}

func TestDraftProjectMenu(t *testing.T) {
	a := testApp(t)
	a.newDraft()
	s := a.build(time.Now())
	if s.Header.Project != "atlas" || s.Hero != "atlas" {
		t.Fatalf("draft in %q (hero %q)", s.Header.Project, s.Hero)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "project.hero", Kind: "click"})
	s = a.build(time.Now())
	if s.Projects == nil || len(s.Projects.Groups) != 2 {
		t.Fatalf("project menu = %+v", s.Projects)
	}
	g := s.Projects.Groups[1]
	if g.Label != "MacBook" || g.Choices[0].Label != "nimbus" || g.Choices[0].Desc != "~/code/nimbus" {
		t.Fatalf("mac group = %+v", g)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "project:1:0", Kind: "click"})
	s = a.build(time.Now())
	if s.Projects != nil || s.Header.Project != "nimbus" || s.Card.Run.Label != "MacBook" || a.draftEnv != "mac" {
		t.Fatalf("after pick: %q on %q env=%q menu=%v", s.Header.Project, s.Card.Run.Label, a.draftEnv, s.Projects != nil)
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

// menuLabels is a menu's rows as text: each label, "|" for a separator
// drawn before it (none above the first), "(off)" after one that can't be
// picked, "▸" for a submenu.
func menuLabels(items []ui.MenuItem) string {
	var out []string
	for i, it := range items {
		l := it.Label
		if it.Sep && i > 0 {
			l = "| " + l
		}
		if it.Off {
			l += " (off)"
		}
		if len(it.Children) > 0 {
			l += " ▸"
		}
		out = append(out, l)
	}
	return strings.Join(out, ", ")
}

// A right-click on a row opens the thread's actions menu
// (buildThreadActionMenuItems). A server whose config hasn't come leaves
// out what needs it; Delete asks twice.
func TestThreadMenu(t *testing.T) {
	a := testApp(t)
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "thread:mac/same", Kind: "context"})
	s := a.build(time.Now())
	if s.Menu == nil {
		t.Fatal("right-click should open the thread menu")
	}
	if got := menuLabels(s.Menu.Items); got != "Rename thread, Mark unread, | Copy ▸, | Archive thread, Delete" {
		t.Fatalf("menu = %s", got)
	}
	if got := menuLabels(s.Menu.Items[2].Children); got != "Path, Thread, Thread ID" {
		t.Fatalf("copy = %s", got)
	}
	a.handle(th, ui.Action{ID: "ctx:delete", Kind: "click"})
	s = a.build(time.Now())
	if s.Menu == nil || s.Menu.Items[4].Label != "Click again to delete" || !s.Menu.Items[4].Danger {
		t.Fatalf("after one click: %+v", s.Menu)
	}
	// The Mac has no socket here, so the second click reports that.
	a.handle(th, ui.Action{ID: "ctx:delete", Kind: "click"})
	s = a.build(time.Now())
	if s.Menu != nil || !strings.Contains(s.Error, "isn't connected") {
		t.Fatalf("after two clicks: menu=%v error=%q", s.Menu != nil, s.Error)
	}
}

// With its server's config the menu is T3's, as measured on a pinned
// thread with a branch (05-header.md §3.3), less what tern has no surface
// for: the branch's new thread first, Pin or Unpin, Settle, then Rename,
// Regenerate title, Mark unread and Auto-settle behavior, Copy with the
// branch, and Archive, off while the turn runs, and Delete. Each row has
// its icon.
func TestThreadMenuItems(t *testing.T) {
	a := testApp(t)
	e := a.envs[0]
	e.config = &t3.ServerConfig{}
	e.config.Environment.Capabilities.ThreadAutoSettleOptOut = true
	t1 := e.threads["t1"]
	t1.Branch, t1.PinnedAt, t1.ActiveRunID = "main", "2026-01-01T00:00:00Z", "run-1"
	a.handle(ui.NewTheme(), ui.Action{ID: "thread:studio/t1", Kind: "context"})
	m := a.build(time.Now()).Menu
	want := "New thread on main, Unpin thread, Settle thread, | Rename thread, Regenerate title, Mark unread, Auto-settle behavior ▸, " +
		"| Copy ▸, | Archive thread (off), Delete"
	if got := menuLabels(m.Items); got != want {
		t.Fatalf("menu = %s\nwant   %s", got, want)
	}
	icons := []string{"message-square-plus", "pin-off", "circle-check", "pencil", "refresh-cw", "mail-open", "timer", "copy", "archive", "trash"}
	for i, it := range m.Items {
		if it.Icon != icons[i] {
			t.Errorf("%s: icon %q, want %q", it.Label, it.Icon, icons[i])
		}
	}
	if got := menuLabels(m.Items[6].Children); got != "Follow settings, Never" || !m.Items[6].Children[0].Checked || !m.Items[6].Children[0].Radio {
		t.Errorf("auto-settle = %s %+v", got, m.Items[6].Children)
	}
	if got := menuLabels(m.Items[7].Children); got != "Path, Branch, Thread, Thread ID" {
		t.Errorf("copy = %s", got)
	}
}

func TestDetailsPopover(t *testing.T) {
	a := testApp(t)
	a.openThread("studio/t1")
	a.handle(ui.NewTheme(), ui.Action{ID: "details", Kind: "click"})
	s := a.build(time.Now())
	if !s.Details || s.Card == nil || s.Card.Run.Label != "studio" {
		t.Fatalf("details = %v, card = %+v", s.Details, s.Card)
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
	if a.draft != "p1" || a.draftEnv != "studio" || strings.Contains(s.Empty, "Pick a thread") {
		t.Fatalf("draft %q on %q, empty %q", a.draft, a.draftEnv, s.Empty)
	}
	if s.Hero != "atlas" {
		t.Fatal("the draft should offer the project picker")
	}
}
