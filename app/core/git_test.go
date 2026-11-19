package core

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// liveApp is testApp with the studio environment on a fake socket that
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
	a.openThread("studio/t1")
	eventually(t, "the status and refs load", func() bool {
		built(a, time.Now())
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
	if s := built(a, time.Now()); s.Card.Git.Label != "Commit & push" {
		t.Fatalf("quick action %+v", s.Card.Git)
	}
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	s := built(a, time.Now())
	if c := s.Confirm; c == nil || c.Title != "Commit & push to default ref?" || c.Continue != "Commit & push to main" || c.BranchOff != "" {
		t.Fatalf("confirm %+v", s.Confirm)
	}
	if len(f.requests("git.runStackedAction")) != 0 {
		t.Fatal("it ran before the dialog said so")
	}
	a.handle(th, ui.Action{ID: "confirm:continue", Kind: "click"})
	r := f.wait("git.runStackedAction", 1)
	if r.Payload["action"] != "commit_push" || r.Payload["cwd"] != "/home/dev/code/atlas" || r.Payload["threadId"] != "t1" || r.Payload["featureBranch"] != nil || r.Payload["actionId"] == "" {
		t.Fatalf("payload %v", r.Payload)
	}
	if s := built(a, time.Now()); s.Confirm != nil {
		t.Fatal("the dialog should close")
	}
	f.chunk(r, map[string]any{"kind": "phase_started", "phase": "commit", "label": "Committing..."})
	f.chunk(r, map[string]any{"kind": "hook_started", "hookName": "pre-commit"})
	f.chunk(r, map[string]any{"kind": "hook_output", "hookName": "pre-commit", "text": "lint\nlint ok\n"})
	eventually(t, "the progress shows", func() bool {
		g := built(a, time.Now()).Card.Git
		return g.Running != nil && g.Running.Status == "Committing..." && g.Running.Output == "lint ok" && strings.HasSuffix(g.Running.Elapsed, "s")
	})
	f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{
		"branch": map[string]any{"status": "skipped_not_requested"}, "pr": map[string]any{"status": "skipped_not_requested"},
		"toast": map[string]any{"title": "Pushed to main", "description": "1 commit"}}})
	f.exit(r, nil)
	eventually(t, "the result shows", func() bool {
		g := built(a, time.Now()).Card.Git
		return g.Running == nil && g.Success != nil && g.Success.Title == "Pushed to main" && g.Success.Description == "1 commit"
	})
	if g := built(a, time.Now().Add(11*time.Second)).Card.Git; g.Success != nil {
		t.Fatal("the result should go after ten seconds")
	}
}

// Abort runs nothing; the feature branch needs changes to name it after,
// and then runs the action on a new branch.
func TestConfirmFeatureBranchAndAbort(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.RefName, s.IsDefaultRef, s.AheadCount = "main", true, 2 }))
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	s := built(a, time.Now())
	if c := s.Confirm; c == nil || c.Title != "Push to default ref?" || c.Continue != "Push to main" || !strings.Contains(c.BranchOff, "no changes to commit") {
		t.Fatalf("confirm %+v", s.Confirm)
	}
	a.handle(th, ui.Action{ID: "confirm:branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.git", Kind: "click"})
	a.handle(th, ui.Action{ID: "confirm:abort", Kind: "click"})
	time.Sleep(50 * time.Millisecond)
	if s := built(a, time.Now()); s.Confirm != nil || len(f.requests("git.runStackedAction")) != 0 {
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
	m := built(a, time.Now()).Card.Menu
	if m == nil || m.For != "git" || len(m.Rows) != 3 || !m.Rows[0].Disabled || m.Rows[1].Label != "Push" || m.Rows[2].Label != "Create PR" {
		t.Fatalf("menu %+v", m)
	}
	if r := f.wait("vcs.refreshStatus", 1); r.Payload["cwd"] != "/home/dev/code/atlas" {
		t.Fatalf("opening the menu should refresh the status: %v", r.Payload)
	}
	a.handle(th, ui.Action{ID: "card.git:1", Kind: "click"})
	r := f.wait("git.runStackedAction", 1)
	if r.Payload["action"] != "push" {
		t.Fatalf("push payload %v", r.Payload)
	}
	f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{"pr": map[string]any{"status": "created", "number": 43}, "toast": map[string]any{"title": "Pushed"}}})
	f.exit(r, nil)
	eventually(t, "push finishes", func() bool { g := built(a, time.Now()).Card.Git; return g.Success != nil })
	a.handle(th, ui.Action{ID: "card.gitmenu", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.git:2", Kind: "click"})
	r = f.wait("git.runStackedAction", 2)
	if r.Payload["action"] != "create_pr" {
		t.Fatalf("PR payload %v", r.Payload)
	}
	f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{"pr": map[string]any{"status": "created", "number": 44}, "toast": map[string]any{"title": "Created PR #44"}}})
	f.exit(r, nil)
	eventually(t, "the PR's result", func() bool {
		g := built(a, time.Now()).Card.Git
		return g.Success != nil && g.Success.Title == "Created PR" && g.Success.Description == ""
	})
}

// The Git menu's Commit asks first, in T3's dialog (05-header.md §5.11):
// the branch, that it's the default one, and the working tree's files. A
// file left out while editing stays out of filePaths, and the trimmed
// message goes as commitMessage; Commit on new branch commits every file
// on a feature branch; with every file out nothing commits; Cancel, the ✕
// and Escape close it; each opening starts afresh. A file's row opens it
// in the editor, under the status's folder.
func TestCommitDialog(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) {
		s.RefName, s.IsDefaultRef, s.HasWorkingTreeChanges = "main", true, true
		s.WorkingTree.RootPath = "/home/dev/code/atlas"
		s.WorkingTree.Files = []t3.VcsFile{{Path: "src/parser.go", Insertions: 12, Deletions: 3}, {Path: "README.md", Insertions: 1}}
	}))
	a.envs[0].config.AvailableEditors = []string{"vscode"}
	f.on("shell.openInEditor", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	th := ui.NewTheme()
	open := func() *ui.CommitDialog {
		t.Helper()
		a.handle(th, ui.Action{ID: "card.gitmenu", Kind: "click"})
		if m := built(a, time.Now()).Card.Menu; m == nil || m.Rows[0].Label != "Commit" || m.Rows[0].Disabled {
			t.Fatalf("menu %+v", m)
		}
		a.handle(th, ui.Action{ID: "card.git:0", Kind: "click"})
		s := built(a, time.Now())
		if s.Card.Menu != nil || s.Commit == nil {
			t.Fatalf("Commit closes the menu for its dialog: menu %+v, dialog %+v", s.Card.Menu, s.Commit)
		}
		return s.Commit
	}
	finish := func(r fakeReq) {
		t.Helper()
		f.chunk(r, map[string]any{"kind": "action_finished", "result": map[string]any{"toast": map[string]any{"title": "Committed"}}})
		f.exit(r, nil)
		eventually(t, "the commit finishes", func() bool { return built(a, time.Now()).Card.Git.Success != nil })
	}
	d := open()
	if d.Branch != "main" || !d.Default || !d.Opens || d.Editing || len(d.Files) != 2 ||
		d.Files[0] != (ui.CommitFile{Path: "src/parser.go", Insertions: 12, Deletions: 3}) || d.Files[1].Excluded {
		t.Fatalf("dialog %+v", d)
	}
	if n := len(f.requests("git.runStackedAction")); n != 0 {
		t.Fatalf("%d actions ran before the dialog said to", n)
	}
	a.handle(th, ui.Action{ID: "commit:edit", Kind: "click"})
	a.handle(th, ui.Action{ID: "commit.file:1", Kind: "click"})
	if d := built(a, time.Now()).Commit; !d.Editing || d.Files[0].Excluded || !d.Files[1].Excluded {
		t.Fatalf("README left out: %+v", d)
	}
	a.handle(th, ui.Action{ID: "commit.open:0", Kind: "click"})
	if r := f.wait("shell.openInEditor", 1); r.Payload["cwd"] != "/home/dev/code/atlas/src/parser.go" || r.Payload["editor"] != "vscode" {
		t.Fatalf("open payload %v", r.Payload)
	}
	th.SetEditorText("commit.message", "  Fix the parser  ")
	a.handle(th, ui.Action{ID: "commit:ok", Kind: "click"})
	r := f.wait("git.runStackedAction", 1)
	if r.Payload["action"] != "commit" || r.Payload["commitMessage"] != "Fix the parser" || r.Payload["featureBranch"] != nil ||
		fmt.Sprint(r.Payload["filePaths"]) != "[src/parser.go]" {
		t.Fatalf("commit payload %v", r.Payload)
	}
	if s := built(a, time.Now()); s.Commit != nil || s.Confirm != nil || th.EditorText("commit.message") != "" {
		t.Fatal("committing closes the dialog and clears it, without asking on the default branch")
	}
	finish(r)

	if d := open(); d.Editing || d.Files[1].Excluded {
		t.Fatalf("reopened %+v", d)
	}
	a.handle(th, ui.Action{ID: "commit:branch", Kind: "click"})
	r = f.wait("git.runStackedAction", 2)
	if r.Payload["action"] != "commit" || r.Payload["featureBranch"] != true || r.Payload["filePaths"] != nil || r.Payload["commitMessage"] != nil {
		t.Fatalf("new branch payload %v", r.Payload)
	}
	finish(r)

	open()
	a.handle(th, ui.Action{ID: "commit:edit", Kind: "click"})
	a.handle(th, ui.Action{ID: "commit.all", Kind: "click"})
	a.handle(th, ui.Action{ID: "commit:ok", Kind: "click"})
	a.handle(th, ui.Action{ID: "commit:branch", Kind: "click"})
	if d := built(a, time.Now()).Commit; d == nil || !d.Files[0].Excluded || !d.Files[1].Excluded {
		t.Fatalf("every file out, it stays open: %+v", d)
	}
	a.handle(th, ui.Action{ID: "commit.all", Kind: "click"})
	if d := built(a, time.Now()).Commit; d.Files[0].Excluded || d.Files[1].Excluded {
		t.Fatalf("all back in: %+v", d)
	}
	for _, shut := range []ui.Action{{ID: "commit:cancel", Kind: "click"}, {ID: "dialog:close", Kind: "click"}, {ID: "commit.message", Kind: "key", Text: "⎋"}} {
		if shut.ID != "commit:cancel" {
			open()
		}
		a.handle(th, shut)
		if s := built(a, time.Now()); s.Commit != nil {
			t.Fatalf("%+v leaves the dialog open", shut)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(f.requests("git.runStackedAction")); n != 2 {
		t.Fatalf("%d actions ran, want the two commits", n)
	}
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
		s := built(a, time.Now())
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
	if r := f.wait("vcs.pull", 1); r.Payload["cwd"] != "/home/dev/code/atlas" {
		t.Fatalf("pull payload %v", r.Payload)
	}
	eventually(t, "pulled", func() bool {
		g := built(a, time.Now()).Card.Git
		return g.Success != nil && g.Success.Title == "Pulled" && g.Success.Description == "Updated feature from origin/feature"
	})

	a, f = liveApp(t, &t3.VcsStatus{})
	f.on("vcs.init", func(f *fakeEnv, r fakeReq) {})
	if s := built(a, time.Now()); !s.Card.InitGit || s.Card.Initing {
		t.Fatal("no Initialize Git row")
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.initgit", Kind: "click"})
	r := f.wait("vcs.init", 1)
	if r.Payload["cwd"] != "/home/dev/code/atlas" {
		t.Fatalf("init payload %v", r.Payload)
	}
	if !built(a, time.Now()).Card.Initing {
		t.Fatal("the row should say it's initializing while vcs.init runs")
	}
	f.failWith(r, "permission denied")
	eventually(t, "the failure shows, and the row offers to try again", func() bool {
		s := built(a, time.Now())
		return s.Error == "Git initialization failed: permission denied" && !s.Card.Initing
	})
}

// The branch picker lists the checkout's refs; a search asks the server
// for the refs it matches (vcs.listRefs with the query), which the picker
// filters as T3's does. Picking one switches to it with vcs.switchRef, the
// row naming it until the switch lands, and records it on the thread. A
// search no ref is named offers to create it, as a ref name, before its
// refs come: vcs.createRef.
func TestBranchPicker(t *testing.T) {
	a, f := liveApp(t, status(nil),
		t3.VcsRef{Name: "feature", Current: true}, t3.VcsRef{Name: "main", IsDefault: true}, t3.VcsRef{Name: "origin/main", IsRemote: true})
	f.on("vcs.switchRef", func(f *fakeEnv, r fakeReq) {})
	f.on("vcs.createRef", func(f *fakeEnv, r fakeReq) { f.exit(r, map[string]any{"refName": "new-idea"}) })
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	p := built(a, time.Now()).Card.Picker
	if p == nil || len(p.Refs) != 3 || p.Refs[0].Tag != "current" || p.Refs[1].Tag != "default" || p.Refs[2].Tag != "remote" || p.Create != "" || p.Highlight != 0 {
		t.Fatalf("picker %+v", p)
	}
	if r := f.wait("vcs.refreshStatus", 1); r.Payload["localOnly"] != true {
		t.Fatalf("opening the picker should refresh the local status: %v", r.Payload)
	}
	a.handle(th, ui.Action{ID: "branch.search", Kind: "change", Text: "MAI"})
	if r := f.wait("vcs.listRefs", 2); r.Payload["query"] != "MAI" || r.Payload["limit"] != 100.0 || r.Payload["cwd"] != "/home/dev/code/atlas" {
		t.Fatalf("search payload %v", r.Payload)
	}
	eventually(t, "the search's refs", func() bool {
		p := built(a, time.Now()).Card.Picker
		return len(p.Refs) == 2 && p.Refs[0].Name == "main" && p.Refs[1].Name == "origin/main" && p.Create == `Create new ref "MAI"`
	})
	// The arrows go round: up from the first item is the last, the ref to create.
	a.handle(th, ui.Action{ID: "branch.search", Kind: "key", Text: "↑"})
	if p := built(a, time.Now()).Card.Picker; p.Highlight != 2 {
		t.Fatalf("highlight %d, want 2", p.Highlight)
	}
	a.handle(th, ui.Action{ID: "branch.ref:0", Kind: "click"})
	r := f.wait("vcs.switchRef", 1)
	if r.Payload["refName"] != "main" || r.Payload["cwd"] != "/home/dev/code/atlas" {
		t.Fatalf("switch payload %v", r.Payload)
	}
	if s := built(a, time.Now()); s.Card.Picker != nil || s.Card.Branch.Label != "main" || !s.Card.Branch.Disabled {
		t.Fatalf("while it switches the row names the ref and waits: %+v", s.Card.Branch)
	}
	f.exit(r, map[string]any{"refName": "main"})
	if d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 1); d.Payload["branch"] != "main" || d.Payload["threadId"] != "t1" {
		t.Fatalf("branch update %v", d.Payload)
	}
	eventually(t, "the switch lands", func() bool { return !built(a, time.Now()).Card.Branch.Disabled })
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "branch.search", Kind: "change", Text: " new  idea"})
	if p := built(a, time.Now()).Card.Picker; p.Create != `Create new ref "new-idea"` {
		t.Fatalf("create offered %q", p.Create)
	}
	a.handle(th, ui.Action{ID: "branch.search", Kind: "submit"})
	if r := f.wait("vcs.createRef", 1); r.Payload["refName"] != "new-idea" || r.Payload["switchRef"] != true {
		t.Fatalf("create payload %v", r.Payload)
	}
}

// A thread in a worktree picks refs as T3's does (resolveBranchSelectionTarget):
// one checked out somewhere moves it there without a switch, the project's
// folder holding the default branch; any other switches its own worktree.
// A move detaches its provider sessions first, so its agent starts again
// in the new checkout, and none can happen while its turn runs: those refs
// are off, and the picker says why.
func TestBranchPickerWorktrees(t *testing.T) {
	a, f := liveApp(t, status(nil),
		t3.VcsRef{Name: "fix/lexer", Current: true, WorktreePath: "/work/widgets-wt/fix"},
		t3.VcsRef{Name: "main", IsDefault: true, WorktreePath: "/work/widgets"},
		t3.VcsRef{Name: "feature/tabs", WorktreePath: "/work/widgets-wt/tabs"},
		t3.VcsRef{Name: "spike"})
	f.on("vcs.switchRef", func(f *fakeEnv, r fakeReq) { f.exit(r, map[string]any{"refName": "spike"}) })
	a.mu.Lock()
	e := a.envs[0]
	e.projects["p1"] = t3.Project{ID: "p1", Title: "widgets", WorkspaceRoot: "/work/widgets"}
	th3 := e.threads["t1"]
	th3.WorktreePath, th3.Branch, th3.ActiveRunID = "/work/widgets-wt/fix", "fix/lexer", "run-2"
	a.mu.Unlock()
	sub := f.wait("orchestration.subscribeThread", 1)
	f.chunk(sub, map[string]any{"kind": "snapshot", "projection": map[string]any{"thread": map[string]any{"id": "t1"}, "providerSessions": []any{map[string]any{"id": "ps-1"}}}})
	// The thread's snapshot, which brings its session, has come once the
	// thread stops loading.
	eventually(t, "the worktree's refs and the thread's session", func() bool {
		s := built(a, time.Now())
		return s.Card.Branch != nil && !s.Card.Branch.Disabled && s.Empty == ""
	})
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	p := built(a, time.Now()).Card.Picker
	if p.Notice != "Stop the current turn to switch worktrees." {
		t.Errorf("notice %q", p.Notice)
	}
	var off []string
	for _, r := range p.Refs {
		if r.Disabled {
			off = append(off, r.Name)
		}
	}
	if strings.Join(off, " ") != "main feature/tabs" {
		t.Fatalf("refs off while the turn runs: %v", off)
	}
	a.handle(th, ui.Action{ID: "branch.ref:2", Kind: "click"})
	time.Sleep(50 * time.Millisecond)
	if n := len(f.requests("orchestration.dispatchCommand:thread.metadata.update")); n != 0 {
		t.Fatalf("a running turn's thread moved: %d updates", n)
	}

	a.mu.Lock()
	th3.ActiveRunID = ""
	a.mu.Unlock()
	if p := built(a, time.Now()).Card.Picker; p == nil {
		t.Fatal("the blocked pick closed the picker")
	} else if p.Notice != "" {
		t.Errorf("notice with no turn running %q", p.Notice)
	}
	a.handle(th, ui.Action{ID: "branch.ref:2", Kind: "click"})
	if d := f.wait("orchestration.dispatchCommand:provider-session.detach", 1).Payload; d["providerSessionId"] != "ps-1" || d["threadId"] != "t1" || d["reason"] != "client-requested" {
		t.Fatalf("detach %v", d)
	}
	if d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 1).Payload; d["branch"] != "feature/tabs" || d["worktreePath"] != "/work/widgets-wt/tabs" {
		t.Fatalf("move to the ref's worktree %v", d)
	}

	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "branch.ref:3", Kind: "click"})
	if r := f.wait("vcs.switchRef", 1).Payload; r["cwd"] != "/work/widgets-wt/fix" || r["refName"] != "spike" {
		t.Fatalf("switch in the thread's worktree %v", r)
	}
	if d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 2).Payload; d["branch"] != "spike" || d["worktreePath"] != "/work/widgets-wt/fix" {
		t.Fatalf("staying put %v", d)
	}

	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "branch.ref:1", Kind: "click"})
	if d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 3).Payload; d["branch"] != "main" || d["worktreePath"] != nil {
		t.Fatalf("back to the project's folder %v", d)
	}
	if n := len(f.requests("orchestration.dispatchCommand:provider-session.detach")); n != 2 {
		t.Errorf("%d detaches, want one for each move", n)
	}
	if n := len(f.requests("vcs.switchRef")); n != 1 {
		t.Errorf("%d switches: refs checked out already move the thread instead", n)
	}
}

// A long list of refs comes a page at a time: the picker says how many it
// has and, asked near its end, loads the next page from the cursor the
// last one gave; a remote ref it switches to is recorded by the local
// branch it makes.
func TestBranchPickerPages(t *testing.T) {
	a, f := liveApp(t, status(nil))
	refs := func(from, to int) []t3.VcsRef {
		var out []t3.VcsRef
		for i := from; i < to; i++ {
			out = append(out, t3.VcsRef{Name: fmt.Sprintf("origin/ref-%03d", i), IsRemote: true})
		}
		return out
	}
	f.on("vcs.listRefs", func(f *fakeEnv, r fakeReq) {
		if r.Payload["cursor"] == nil {
			f.exit(r, map[string]any{"isRepo": true, "refs": refs(0, 100), "nextCursor": 100, "totalCount": 150})
			return
		}
		f.exit(r, map[string]any{"isRepo": true, "refs": refs(100, 150), "nextCursor": nil, "totalCount": 150})
	})
	f.on("vcs.switchRef", func(f *fakeEnv, r fakeReq) { f.exit(r, map[string]any{"refName": "ref-120"}) })
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "branch.search", Kind: "change", Text: "ref"})
	eventually(t, "the first page", func() bool {
		p := built(a, time.Now()).Card.Picker
		return len(p.Refs) == 100 && p.More && p.Status == "Showing 100 of 150 refs"
	})
	a.handle(th, ui.Action{ID: "branch.more", Kind: "click"})
	if r := f.wait("vcs.listRefs", 3); r.Payload["cursor"] != 100.0 || r.Payload["query"] != "ref" {
		t.Fatalf("next page payload %v", r.Payload)
	}
	eventually(t, "the second page", func() bool {
		p := built(a, time.Now()).Card.Picker
		return len(p.Refs) == 150 && !p.More && p.Status == ""
	})
	a.handle(th, ui.Action{ID: "branch.ref:120", Kind: "click"})
	if r := f.wait("vcs.switchRef", 1); r.Payload["refName"] != "origin/ref-120" {
		t.Fatalf("switch payload %v", r.Payload)
	}
	if d := f.wait("orchestration.dispatchCommand:thread.metadata.update", 1); d.Payload["branch"] != "ref-120" {
		t.Fatalf("branch update %v", d.Payload)
	}
}

// Open in editor opens the folder with shell.openInEditor on a machine
// that's this computer, in the preferred editor, which the menu picks.
func TestOpenInEditorHere(t *testing.T) {
	a, f := liveApp(t, status(nil))
	a.envs[0].config.AvailableEditors = []string{"zed", "vscode"}
	f.on("shell.openInEditor", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	th := ui.NewTheme()
	if o := built(a, time.Now()).Card.OpenIn; o.Label != "Open in VS Code" || o.Disabled {
		t.Fatalf("open in %+v", o)
	}
	a.handle(th, ui.Action{ID: "card.openin", Kind: "click"})
	if r := f.wait("shell.openInEditor", 1); r.Payload["editor"] != "vscode" || r.Payload["cwd"] != "/home/dev/code/atlas" {
		t.Fatalf("payload %v", r.Payload)
	}
	a.handle(th, ui.Action{ID: "card.editors", Kind: "click"})
	m := built(a, time.Now()).Card.Menu
	if m == nil || len(m.Rows) != 2 || m.Rows[0].Label != "VS Code" || m.Rows[0].Shortcut == "" || m.Rows[1].Label != "Zed" {
		t.Fatalf("editors %+v", m)
	}
	a.handle(th, ui.Action{ID: "card.openin:1", Kind: "click"})
	if o := built(a, time.Now()).Card.OpenIn; o.Label != "Open in Zed" || len(f.requests("shell.openInEditor")) != 1 {
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
	a.openThread("studio/t1")
	s := built(a, time.Now())
	if o := s.Card.OpenIn; o.Label != "Open in Cursor" || o.Disabled {
		t.Fatalf("open in %+v", o)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.editors", Kind: "click"})
	if m := built(a, time.Now()).Card.Menu; len(m.Rows) != 2 || m.Rows[1].Label != "Opens over SSH. Needs your key on studio" {
		t.Fatalf("menu %+v", m)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.openin", Kind: "click"})
	if len(opened) != 1 || opened[0] != "cursor://vscode-remote/ssh-remote+dev%40box.example.ts.net/home/dev/code/atlas" {
		t.Fatalf("opened %v", opened)
	}
	e.config.RemoteOpenTargets = nil
	a.handle(ui.NewTheme(), ui.Action{ID: "card.editors", Kind: "click"})
	s = built(a, time.Now())
	if o := s.Card.OpenIn; !o.Disabled || s.Card.Menu.Rows[0].Label != "No SSH route to studio" {
		t.Fatalf("no route: %+v %+v", o, s.Card.Menu)
	}
}

// Renaming commits the trimmed title with thread.metadata.update, refuses
// an empty one and leaves an unchanged one alone; Escape cancels, and so
// does the focus moving to a button.
func TestRenameThread(t *testing.T) {
	a, f := liveApp(t, status(nil))
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	if s := built(a, time.Now()); !s.Header.Renaming || th.EditorText("header.title") != "On studio" {
		t.Fatalf("renaming %v, %q", s.Header.Renaming, th.EditorText("header.title"))
	}
	a.handle(th, ui.Action{ID: "header.title", Kind: "submit", Text: "   "})
	if s := built(a, time.Now()); s.Header.Renaming || s.Error != "Thread title cannot be empty" {
		t.Fatalf("empty: renaming %v, error %q", s.Header.Renaming, s.Error)
	}
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "key", Text: "⎋"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "cancel", Text: "Something else"})
	if s := built(a, time.Now()); s.Header.Renaming {
		t.Fatal("a press on a button should end the rename")
	}
	a.handle(th, ui.Action{ID: "header.title", Kind: "double"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "blur", Text: "On studio"})
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
// drawer, typed in once the shell's there; one run from the menu becomes
// the split's. A draft has no terminal to run one in yet, and says so
// without making it the split's.
func TestRunScript(t *testing.T) {
	a, f := liveApp(t, status(nil))
	a.envs[0].projects["p1"] = t3.Project{ID: "p1", Title: "atlas", WorkspaceRoot: "/home/dev/code/atlas",
		Scripts: []t3.ProjectScript{{ID: "dev", Name: "Dev", Command: "npm run dev", Icon: "play"}, {ID: "test", Name: "Test", Command: "npm test", Icon: "test"}}}
	f.on("terminal.attach", func(f *fakeEnv, r fakeReq) {
		f.chunk(r, map[string]any{"type": "snapshot", "snapshot": map[string]any{"status": "running", "history": ""}})
	})
	f.on("terminal.write", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	if sc := built(a, time.Now()).Card.Scripts; sc.Primary != "Dev" || sc.Icon != "play" {
		t.Fatalf("scripts %+v", sc)
	}
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.script", Kind: "click"})
	if r := f.wait("terminal.attach", 1); r.Payload["cwd"] != "/home/dev/code/atlas" || r.Payload["threadId"] != "t1" {
		t.Fatalf("attach %v", r.Payload)
	}
	if w := f.wait("terminal.write", 1); w.Payload["data"] != " npm run dev\r" {
		t.Fatalf("write %v", w.Payload)
	}
	if s := built(a, time.Now()); s.Term == nil {
		t.Fatal("the drawer should be open")
	}
	a.handle(th, ui.Action{ID: "card.scripts", Kind: "click"})
	if m := built(a, time.Now()).Card.Menu; m == nil || len(m.Rows) != 2 || m.Rows[1].Label != "Test" || m.Rows[1].Icon != "flask-conical" {
		t.Fatalf("scripts menu %+v", m)
	}
	a.handle(th, ui.Action{ID: "card.scripts:1", Kind: "click"})
	if w := f.wait("terminal.write", 2); w.Payload["data"] != " npm test\r" {
		t.Fatalf("write %v", w.Payload)
	}
	if sc := built(a, time.Now()).Card.Scripts; sc.Primary != "Test" || sc.Icon != "test" {
		t.Fatalf("after Test: scripts %+v", sc)
	}
	a.handle(th, ui.Action{ID: "header.project", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.scripts", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.scripts:0", Kind: "click"})
	s := built(a, time.Now())
	if s.Header.Server || s.Error != `Failed to run script "Dev": a new thread has no terminal until its first message` || s.Card.Scripts.Primary != "Test" {
		t.Fatalf("on a draft: error %q, scripts %+v", s.Error, s.Card.Scripts)
	}
}

// A new worktree needs a ref to start from: with none the draft says so
// and keeps its text, rather than launch in the project's own checkout
// (planComposerTurn).
func TestDraftNewWorktreeNeedsABase(t *testing.T) {
	a, f := liveApp(t, status(func(s *t3.VcsStatus) { s.RefName = "" }))
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "header.project", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.run", Kind: "click"})
	a.handle(th, ui.Action{ID: "card.run:2", Kind: "click"})
	if s := built(a, time.Now()); s.Card.Run.Workspace != "folder-git-2" || s.Card.Branch == nil || s.Card.Branch.Label != "Select ref" {
		t.Fatalf("run %+v, branch %+v", s.Card.Run, s.Card.Branch)
	}
	a.mu.Lock()
	a.choices["draft"] = &choice{sel: t3.ModelSelection{InstanceID: "claude", Model: "claude-opus"}}
	a.mu.Unlock()
	th.SetEditorText("composer", "Start on the parser")
	a.handle(th, ui.Action{ID: "composer", Kind: "submit", Text: "Start on the parser"})
	if s := built(a, time.Now()); s.Error != "Select a base branch before sending in New worktree mode." || th.EditorText("composer") != "Start on the parser" {
		t.Fatalf("error %q, composer %q", s.Error, th.EditorText("composer"))
	}
	time.Sleep(50 * time.Millisecond)
	if n := len(f.requests("orchestration.launchThread")); n != 0 {
		t.Fatalf("%d launches", n)
	}
}

// A shared thread's header says who its guest is, or that the link waits
// or has expired, by the minute as T3's clock reads it. Stop sharing sends
// thread.guest-share.close, off until the server answers, and the record
// that follows without a link takes the line away.
func TestGuestShare(t *testing.T) {
	a, f := liveApp(t, status(nil))
	pending := make(chan fakeReq, 2)
	f.on("orchestration.dispatchCommand", func(f *fakeEnv, r fakeReq) {
		if r.Payload["type"] != "thread.guest-share.close" {
			f.exit(r, map[string]any{"sequence": 1})
			return
		}
		pending <- r
	})
	sub := f.wait("orchestration.subscribeThread", 1)
	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Minute).Add(30 * time.Second)
	share := map[string]any{"id": "share-1", "host": "example.test", "createdAt": expires.Add(-24 * time.Hour).Format(time.RFC3339Nano),
		"expiresAt": expires.Format(time.RFC3339Nano), "guestName": nil, "messages": []any{}}
	f.chunk(sub, map[string]any{"kind": "snapshot", "projection": map[string]any{"thread": map[string]any{"id": "t1", "guestShare": share}}})
	guest := func(now time.Time) *ui.GuestLine { return built(a, now).Header.Guest }
	eventually(t, "the link shows", func() bool { return guest(time.Now()) != nil })
	if g := guest(time.Now()); g.Status != "Guest link active" || g.Closing {
		t.Fatalf("waiting: %+v", g)
	}
	// Its minute has come though its second hasn't.
	if g := guest(expires.Add(-30 * time.Second)); g.Status != "Guest link expired" {
		t.Errorf("at the minute it expires in: %q", g.Status)
	}
	if g := guest(expires.Add(-31 * time.Second)); g.Status != "Guest link active" {
		t.Errorf("the second before that minute: %q", g.Status)
	}
	share["guestName"] = "Sam"
	f.chunk(sub, map[string]any{"kind": "event", "event": map[string]any{"type": "thread.guest-share-updated", "payload": map[string]any{"id": "t1", "guestShare": share}}})
	eventually(t, "the guest's name", func() bool { return guest(time.Now()).Status == "Sam is a guest" })

	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "header.unshare", Kind: "click"})
	r := <-pending
	if r.Payload["type"] != "thread.guest-share.close" || r.Payload["threadId"] != "t1" {
		t.Fatalf("close %v", r.Payload)
	}
	if g := guest(time.Now()); !g.Closing {
		t.Error("Stop sharing should be off while the close is on its way")
	}
	a.handle(th, ui.Action{ID: "header.unshare", Kind: "click"})
	f.exit(r, map[string]any{"sequence": 2})
	eventually(t, "Stop sharing comes back", func() bool { return !guest(time.Now()).Closing })
	select {
	case r := <-pending:
		t.Fatalf("a second close while the first was on its way: %v", r.Payload)
	default:
	}
	f.chunk(sub, map[string]any{"kind": "event", "event": map[string]any{"type": "thread.guest-share-updated", "payload": map[string]any{"id": "t1", "guestShare": nil}}})
	eventually(t, "the line goes with the link", func() bool { return guest(time.Now()) == nil })
}
