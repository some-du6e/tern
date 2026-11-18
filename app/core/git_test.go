package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// liveApp is testApp with the swiss environment on a fake socket that
// streams st as the open thread's Git status.
func liveApp(t *testing.T, st *t3.VcsStatus, refs ...t3.VcsRef) (*App, *fakeEnv) {
	t.Helper()
	a := testApp(t)
	f := newFakeEnv(t)
	e := a.envs[0]
	e.config = &t3.ServerConfig{}
	e.threads["t1"].LatestRunID = "run-1"
	f.on("subscribeVcsStatus", func(f *fakeEnv, r fakeReq) {
		f.chunk(r, map[string]any{"_tag": "snapshot", "local": st.VcsStatusLocal, "remote": st.VcsStatusRemote})
	})
	f.on("vcs.listRefs", func(f *fakeEnv, r fakeReq) {
		f.exit(r, t3.VcsRefs{IsRepo: true, HasPrimaryRemote: true, Refs: refs, TotalCount: len(refs)})
	})
	f.on("vcs.refreshStatus", func(f *fakeEnv, r fakeReq) { f.exit(r, st) })
	f.on("orchestration.subscribeThread", func(f *fakeEnv, r fakeReq) {})
	f.on("orchestration.dispatchCommand", func(f *fakeEnv, r fakeReq) { f.exit(r, map[string]any{"sequence": 1}) })
	e.conn = f.dial(a)
	a.openThread("swiss/t1")
	eventually(t, "the status and refs load", func() bool {
		a.build(time.Now())
		a.mu.Lock()
		defer a.mu.Unlock()
		_, _, _, v := a.checkout()
		return v != nil && v.status != nil && v.refs != nil
	})
	return a, f
}

// On the default branch the quick action asks first, and runs only once
// the dialog says to: git.runStackedAction with the thread, its progress
// on the row, then its result for ten seconds.
func TestQuickActionAsksOnTheDefaultBranch(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.RefName, s.IsDefaultRef, s.HasWorkingTreeChanges = "main", true, true }))
	th := ui.NewTheme()
	if s := a.build(time.Now()); s.Card.Git.Label != "Commit & push" {
		t.Fatalf("quick action %+v", s.Card.Git)
	}
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	s := a.build(time.Now())
	if c := s.Confirm; c == nil || c.Title != "Commit & push to default ref?" || c.Continue != "Commit & push to main" || c.BranchOff != "" {
		t.Fatalf("confirm %+v", s.Confirm)
	}
	if len(f.requests("git.runStackedAction")) != 0 {
		t.Fatal("it ran before the dialog said so")
	}
	a.handle(th, ui.Action{ID: "confirm:continue", Kind: "click"})
	r := f.wait("git.runStackedAction", 1)
	if r.Payload["action"] != "commit_push" || r.Payload["cwd"] != "/root/code/goog" || r.Payload["threadId"] != "t1" || r.Payload["featureBranch"] != nil || r.Payload["actionId"] == "" {
		t.Fatalf("payload %v", r.Payload)
	}
	if s := a.build(time.Now()); s.Confirm != nil {
		t.Fatal("the dialog should close")
	}
	f.chunk(r, map[string]any{"kind": "phase_started", "phase": "commit", "label": "Committing..."})
	f.chunk(r, map[string]any{"kind": "hook_started", "hookName": "pre-commit"})
	f.chunk(r, map[string]any{"kind": "hook_output", "hookName": "pre-commit", "text": "lint\nlint ok\n"})
	eventually(t, "the progress shows", func() bool {
		g := a.build(time.Now()).Card.Git
		return g.Running != nil && g.Running.Status == "Committing..." && g.Running.Output == "lint ok" && strings.HasSuffix(g.Running.Elapsed, "s")
	})
	f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{
		"branch": map[string]any{"status": "skipped_not_requested"}, "pr": map[string]any{"status": "skipped_not_requested"},
		"toast": map[string]any{"title": "Pushed to main", "description": "1 commit"}}})
	f.exit(r, nil)
	eventually(t, "the result shows", func() bool {
		g := a.build(time.Now()).Card.Git
		return g.Running == nil && g.Success != nil && g.Success.Title == "Pushed to main" && g.Success.Description == "1 commit"
	})
	if g := a.build(time.Now().Add(11 * time.Second)).Card.Git; g.Success != nil {
		t.Fatal("the result should go after ten seconds")
	}
}

// Abort runs nothing; the feature branch needs changes to name it after,
// and then runs the action on a new branch.
func TestConfirmFeatureBranchAndAbort(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.RefName, s.IsDefaultRef, s.AheadCount = "main", true, 2 }))
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	s := a.build(time.Now())
	if c := s.Confirm; c == nil || c.Title != "Push to default ref?" || c.Continue != "Push to main" || !strings.Contains(c.BranchOff, "no changes to commit") {
		t.Fatalf("confirm %+v", s.Confirm)
	}
	a.handle(th, ui.Action{ID: "confirm:branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	a.handle(th, ui.Action{ID: "confirm:abort", Kind: "click"})
	time.Sleep(50 * time.Millisecond)
	if s := a.build(time.Now()); s.Confirm != nil || len(f.requests("git.runStackedAction")) != 0 {
		t.Fatal("a clean tree's feature branch and Abort should run nothing")
	}

	a, f = liveApp(t, status(func(s *t3.VcsStatus) { s.RefName, s.IsDefaultRef, s.HasWorkingTreeChanges = "main", true, true }))
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	a.handle(th, ui.Action{ID: "confirm:branch", Kind: "click"})
	if r := f.wait("git.runStackedAction", 1); r.Payload["action"] != "commit_push" || r.Payload["featureBranch"] != true {
		t.Fatalf("payload %v", r.Payload)
	}
}

// The options menu runs Push and Create PR straight away off the default
// branch, and a created pull request reads "Created PR".
func TestGitMenuActions(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.AheadCount = 1 }))
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.gitmenu", Kind: "click"})
	m := a.build(time.Now()).Card.Menu
	if m == nil || m.For != "git" || len(m.Rows) != 3 || !m.Rows[0].Disabled || m.Rows[1].Label != "Push" || m.Rows[2].Label != "Create PR" {
		t.Fatalf("menu %+v", m)
	}
	if r := f.wait("vcs.refreshStatus", 1); r.Payload["cwd"] != "/root/code/goog" {
		t.Fatalf("opening the menu should refresh the status: %v", r.Payload)
	}
	a.handle(th, ui.Action{ID: "card.git:1", Kind: "click"})
	r := f.wait("git.runStackedAction", 1)
	if r.Payload["action"] != "push" {
		t.Fatalf("push payload %v", r.Payload)
	}
	f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{"pr": map[string]any{"status": "created", "number": 43}, "toast": map[string]any{"title": "Pushed"}}})
	f.exit(r, nil)
	eventually(t, "push finishes", func() bool { g := a.build(time.Now()).Card.Git; return g.Success != nil })
	a.handle(th, ui.Action{ID: "card.gitmenu", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.git:2", Kind: "click"})
	r = f.wait("git.runStackedAction", 2)
	if r.Payload["action"] != "create_pr" {
		t.Fatalf("PR payload %v", r.Payload)
	}
	f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{"pr": map[string]any{"status": "created", "number": 44}, "toast": map[string]any{"title": "Created PR #44"}}})
	f.exit(r, nil)
	eventually(t, "the PR's result", func() bool {
		g := a.build(time.Now()).Card.Git
		return g.Success != nil && g.Success.Title == "Created PR" && g.Success.Description == ""
	})
}

// A failed action says so and leaves the row as it was.
func TestGitActionFails(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges = true }))
	a.handle(ui.NewTheme(), ui.Action{ID: "card.git", Kind: "click"})
	r := f.wait("git.runStackedAction", 1)
	if r.Payload["action"] != "commit_push_pr" {
		t.Fatalf("payload %v", r.Payload)
	}
	f.chunk(r, map[string]any{"kind": "action_failed", "phase": "push", "message": "rejected"})
	f.failWith(r, "rejected")
	eventually(t, "the failure shows", func() bool {
		s := a.build(time.Now())
		return s.Card.Git.Running == nil && s.Error == "Action failed: rejected"
	})
}

// Behind its upstream the quick action pulls; a folder without Git
// initialises one.
func TestPullAndInit(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.BehindCount = 2 }))
	f.on("vcs.pull", func(f *fakeEnv, r fakeReq) {
		f.exit(r, map[string]any{"status": "pulled", "refName": "feature", "upstreamRef": "origin/feature"})
	})
	a.handle(ui.NewTheme(), ui.Action{ID: "card.git", Kind: "click"})
	if r := f.wait("vcs.pull", 1); r.Payload["cwd"] != "/root/code/goog" {
		t.Fatalf("pull payload %v", r.Payload)
	}
	eventually(t, "pulled", func() bool {
		g := a.build(time.Now()).Card.Git
		return g.Success != nil && g.Success.Title == "Pulled" && g.Success.Description == "Updated feature from origin/feature"
	})

	a, f = liveApp(t, &t3.VcsStatus{})
	f.on("vcs.init", func(f *fakeEnv, r fakeReq) { f.failWith(r, "permission denied") })
	if s := a.build(time.Now()); !s.Card.InitGit {
		t.Fatal("no Initialize Git row")
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.initgit", Kind: "click"})
	if r := f.wait("vcs.init", 1); r.Payload["cwd"] != "/root/code/goog" {
		t.Fatalf("init payload %v", r.Payload)
	}
	eventually(t, "the failure shows", func() bool {
		return a.build(time.Now()).Error == "Git initialization failed: permission denied"
	})
}

// The branch picker lists the refs, filters them as you type, switches to
// one with vcs.switchRef and records it on the thread, and creates one that
// doesn't exist with vcs.createRef.
func TestBranchPicker(t *testing.T) {
	a, f := liveApp(t, status(nil),
		t3.VcsRef{Name: "feature", Current: true}, t3.VcsRef{Name: "main", IsDefault: true}, t3.VcsRef{Name: "origin/main", IsRemote: true})
	f.on("vcs.switchRef", func(f *fakeEnv, r fakeReq) { f.exit(r, map[string]any{"refName": "main"}) })
	f.on("vcs.createRef", func(f *fakeEnv, r fakeReq) { f.exit(r, map[string]any{"refName": "spike"}) })
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	p := a.build(time.Now()).Card.Picker
	if p == nil || len(p.Refs) != 3 || p.Refs[0].Tag != "current" || p.Refs[1].Tag != "default" || p.Refs[2].Tag != "remote" || p.Create != "" {
		t.Fatalf("picker %+v", p)
	}
	a.handle(th, ui.Action{ID: "branch.search", Kind: "change", Text: "MAI"})
	p = a.build(time.Now()).Card.Picker
	if len(p.Refs) != 2 || p.Refs[0].Name != "main" || p.Create != `Create new ref "MAI"` {
		t.Fatalf("filtered %+v", p)
	}
	a.handle(th, ui.Action{ID: "branch.ref:0", Kind: "click"})
	if r := f.wait("vcs.switchRef", 1); r.Payload["refName"] != "main" || r.Payload["cwd"] != "/root/code/goog" {
		t.Fatalf("switch payload %v", r.Payload)
	}
	if d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 1); d.Payload["branch"] != "main" || d.Payload["threadId"] != "t1" {
		t.Fatalf("branch update %v", d.Payload)
	}
	if s := a.build(time.Now()); s.Card.Picker != nil {
		t.Fatal("picking closes the picker")
	}
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "branch.search", Kind: "change", Text: "spike"})
	a.handle(th, ui.Action{ID: "branch.search", Kind: "submit"})
	if r := f.wait("vcs.createRef", 1); r.Payload["refName"] != "spike" || r.Payload["switchRef"] != true {
		t.Fatalf("create payload %v", r.Payload)
	}
}

// Open in editor opens the folder with shell.openInEditor on a machine
// that's this computer, in the preferred editor, which the menu picks.
func TestOpenInEditorHere(t *testing.T) {
	a, f := liveApp(t, status(nil))
	a.envs[0].config.AvailableEditors = []string{"zed", "vscode"}
	f.on("shell.openInEditor", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	th := ui.NewTheme()
	if o := a.build(time.Now()).Card.OpenIn; o.Label != "Open in VS Code" || o.Disabled {
		t.Fatalf("open in %+v", o)
	}
	a.handle(th, ui.Action{ID: "card.openin", Kind: "click"})
	if r := f.wait("shell.openInEditor", 1); r.Payload["editor"] != "vscode" || r.Payload["cwd"] != "/root/code/goog" {
		t.Fatalf("payload %v", r.Payload)
	}
	a.handle(th, ui.Action{ID: "card.editors", Kind: "click"})
	m := a.build(time.Now()).Card.Menu
	if m == nil || len(m.Rows) != 2 || m.Rows[0].Label != "VS Code" || m.Rows[0].Shortcut == "" || m.Rows[1].Label != "Zed" {
		t.Fatalf("editors %+v", m)
	}
	a.handle(th, ui.Action{ID: "card.openin:1", Kind: "click"})
	if o := a.build(time.Now()).Card.OpenIn; o.Label != "Open in Zed" || len(f.requests("shell.openInEditor")) != 1 {
		t.Fatalf("picking only prefers it: %+v", o)
	}
}

// A machine elsewhere opens its folder here, through the editor's remote
// URL to the host it advertises; without one there's no way to it.
func TestOpenInEditorElsewhere(t *testing.T) {
	a := testApp(t)
	a.remoteEditors = func() []string { return []string{"cursor"} }
	var opened []string
	a.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	e := a.envs[0]
	e.config = &t3.ServerConfig{RemoteOpenTargets: []t3.RemoteOpenTarget{{Kind: "tailscale", Host: "box.example.ts.net", Username: "dev"}}}
	a.openThread("swiss/t1")
	s := a.build(time.Now())
	if o := s.Card.OpenIn; o.Label != "Open in Cursor" || o.Disabled {
		t.Fatalf("open in %+v", o)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.editors", Kind: "click"})
	if m := a.build(time.Now()).Card.Menu; len(m.Rows) != 2 || m.Rows[1].Label != "Opens over SSH. Needs your key on swiss" {
		t.Fatalf("menu %+v", m)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.openin", Kind: "click"})
	if len(opened) != 1 || opened[0] != "cursor://vscode-remote/ssh-remote+dev%40box.example.ts.net/root/code/goog" {
		t.Fatalf("opened %v", opened)
	}
	e.config.RemoteOpenTargets = nil
	a.handle(ui.NewTheme(), ui.Action{ID: "card.editors", Kind: "click"})
	s = a.build(time.Now())
	if o := s.Card.OpenIn; !o.Disabled || s.Card.Menu.Rows[0].Label != "No SSH route to swiss" {
		t.Fatalf("no route: %+v %+v", o, s.Card.Menu)
	}
}

// Renaming commits the trimmed title with thread.metadata.update, refuses
// an empty one and leaves an unchanged one alone; Escape cancels.
func TestRenameThread(t *testing.T) {
	a, f := liveApp(t, status(nil))
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	if s := a.build(time.Now()); !s.Header.Renaming || th.EditorText("header.title") != "On swiss" {
		t.Fatalf("renaming %v, %q", s.Header.Renaming, th.EditorText("header.title"))
	}
	a.handle(th, ui.Action{ID: "header.title", Kind: "submit", Text: "   "})
	if s := a.build(time.Now()); s.Header.Renaming || s.Error != "Thread title cannot be empty" {
		t.Fatalf("empty: renaming %v, error %q", s.Header.Renaming, s.Error)
	}
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "key", Text: "⎋"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "blur", Text: "On swiss"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "submit", Text: "  Fix the login page "})
	d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 1)
	if d.Payload["title"] != "Fix the login page" || d.Payload["threadId"] != "t1" {
		t.Fatalf("rename %v", d.Payload)
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(f.requests("orchestration.dispatchCommand:thread.metadata.update")); n != 1 {
		t.Fatalf("%d renames: Escape and an unchanged title should send none", n)
	}
}

// The scripts split runs its script in a new terminal in the thread's
// drawer, typed in once the shell's there, and remembers it.
func TestRunScript(t *testing.T) {
	a, f := liveApp(t, status(nil))
	a.envs[0].projects["p1"] = t3.Project{ID: "p1", Title: "goog", WorkspaceRoot: "/root/code/goog",
		Scripts: []t3.ProjectScript{{ID: "dev", Name: "Dev", Command: "npm run dev", Icon: "play"}}}
	f.on("terminal.attach", func(f *fakeEnv, r fakeReq) {
		f.chunk(r, map[string]any{"type": "snapshot", "snapshot": map[string]any{"status": "running", "history": ""}})
	})
	f.on("terminal.write", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	if sc := a.build(time.Now()).Card.Scripts; sc.Primary != "Dev" || sc.Icon != "play" {
		t.Fatalf("scripts %+v", sc)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.script", Kind: "click"})
	if r := f.wait("terminal.attach", 1); r.Payload["cwd"] != "/root/code/goog" || r.Payload["threadId"] != "t1" {
		t.Fatalf("attach %v", r.Payload)
	}
	if w := f.wait("terminal.write", 1); w.Payload["data"] != " npm run dev\r" {
		t.Fatalf("write %v", w.Payload)
	}
	if s := a.build(time.Now()); s.Term == nil || a.lastScripts["p1"] != "dev" {
		t.Fatalf("drawer %v, last %q", s.Term != nil, a.lastScripts["p1"])
	}
}
