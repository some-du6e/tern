package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// status is a Git status for the rule tables: on branch "feature", with an
// upstream, a remote and nothing to do, then changed by fn.
func status(fn func(st *t3.VcsStatus)) *t3.VcsStatus {
	st := &t3.VcsStatus{}
	st.IsRepo, st.HasPrimaryRemote, st.RefName, st.HasUpstream = true, true, "feature", true
	if fn != nil {
		fn(st)
	}
	return st
}

func intp(n int) *int { return &n }

// The quick action follows T3's table row by row (docs/ui/05-header.md
// §4.5), first match wins.
func TestQuickActionTable(t *testing.T) {
	openPR := &t3.VcsChangeRequest{Number: 7, State: "open"}
	for _, c := range []struct {
		row                 int
		st                  *t3.VcsStatus
		busy, isDefault     bool
		noRemote            bool
		label, kind, act    string
		hint, icon, notIcon string
	}{
		{1, status(nil), true, false, false, "Commit", "show_hint", "", "Git action in progress.", "git-commit-horizontal", ""},
		{2, nil, false, false, false, "Commit", "show_hint", "", "Git status is unavailable.", "git-commit-horizontal", ""},
		{3, status(func(s *t3.VcsStatus) { s.RefName = "" }), false, false, false, "Commit", "show_hint", "", "Create and checkout a ref before pushing or opening a pull request.", "git-commit-horizontal", ""},
		{4, status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges, s.BehindCount = true, 2 }), false, false, false, "Commit", "run_action", "commit", "", "git-commit-horizontal", ""},
		{4, status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges, s.HasUpstream = true, false }), false, false, true, "Commit", "run_action", "commit", "", "git-commit-horizontal", ""},
		{5, status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges = true }), false, true, false, "Commit & push", "run_action", "commit_push", "", "cloud-upload", ""},
		{5, status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges, s.PR = true, openPR }), false, false, false, "Commit & push", "run_action", "commit_push", "", "cloud-upload", ""},
		{6, status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges = true }), false, false, false, "Commit, push & PR", "run_action", "commit_push_pr", "", "provider:", ""},
		{7, status(func(s *t3.VcsStatus) { s.HasUpstream = false }), false, false, true, "Publish repository", "open_publish", "", "", "cloud-upload", ""},
		{8, status(func(s *t3.VcsStatus) { s.HasUpstream, s.PR = false, openPR }), false, false, false, "Commit", "show_hint", "", "Branch is up to date. No action needed.", "git-commit-horizontal", ""},
		{9, status(func(s *t3.VcsStatus) { s.HasUpstream = false }), false, false, false, "Push", "show_hint", "", "No local commits to push.", "cloud-upload", ""},
		{10, status(func(s *t3.VcsStatus) { s.HasUpstream, s.AheadCount = false, 1 }), false, true, false, "Push", "run_action", "commit_push", "", "cloud-upload", ""},
		{10, status(func(s *t3.VcsStatus) { s.HasUpstream, s.AheadCount, s.PR = false, 1, openPR }), false, false, false, "Push", "run_action", "push", "", "cloud-upload", ""},
		{11, status(func(s *t3.VcsStatus) { s.HasUpstream, s.AheadCount = false, 1 }), false, false, false, "Push & create PR", "run_action", "create_pr", "", "provider:", ""},
		{12, status(func(s *t3.VcsStatus) { s.AheadCount, s.BehindCount = 1, 1 }), false, false, false, "Sync ref", "show_hint", "", "Branch has diverged from upstream. Rebase/merge first.", "info", ""},
		{13, status(func(s *t3.VcsStatus) { s.BehindCount = 3 }), false, false, false, "Pull", "run_pull", "", "", "cloud-download", ""},
		{14, status(func(s *t3.VcsStatus) { s.AheadCount = 2 }), false, true, false, "Push", "run_action", "commit_push", "", "cloud-upload", ""},
		{15, status(func(s *t3.VcsStatus) { s.AheadCount = 2 }), false, false, false, "Push & create PR", "run_action", "create_pr", "", "provider:", ""},
		{16, status(func(s *t3.VcsStatus) { s.PR = openPR }), false, false, false, "Commit", "show_hint", "", "Branch is up to date. No action needed.", "git-commit-horizontal", ""},
		{17, status(func(s *t3.VcsStatus) { s.AheadOfDefaultCount = intp(4) }), false, false, false, "Create PR", "run_action", "create_pr", "", "provider:", ""},
		{18, status(func(s *t3.VcsStatus) { s.AheadOfDefaultCount = intp(4) }), false, true, false, "Commit", "show_hint", "", "Branch is up to date. No action needed.", "git-commit-horizontal", ""},
	} {
		q := resolveQuickAction(c.st, c.busy, c.isDefault, !c.noRemote)
		if q.label != c.label || q.kind != c.kind || q.action != c.act || q.hint != c.hint {
			t.Errorf("row %d: %+v, want %q %s %q %q", c.row, q, c.label, c.kind, c.act, c.hint)
		}
		if icon := quickActionIcon(q, ""); icon != c.icon {
			t.Errorf("row %d: icon %q, want %q", c.row, icon, c.icon)
		}
	}
	// GitLab's word for it is a merge request.
	gl := status(func(s *t3.VcsStatus) {
		s.HasWorkingTreeChanges = true
		s.SourceControlProvider = &t3.SourceControlProvider{Kind: "gitlab"}
	})
	if q := resolveQuickAction(gl, false, false, true); q.label != "Commit, push & MR" || quickActionIcon(q, providerKind(gl)) != "provider:gitlab" {
		t.Errorf("gitlab: %+v", q)
	}
}

// The options menu: Commit, Push and Create PR, each disabled when it
// can't run; just Commit without a remote, no Create PR while one's open.
func TestGitMenu(t *testing.T) {
	labels := func(items []gitMenuItem) string {
		var out []string
		for _, it := range items {
			l := it.label
			if it.disabled {
				l += " (off)"
			}
			out = append(out, l)
		}
		return strings.Join(out, ", ")
	}
	dirty := status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges = true })
	ahead := status(func(s *t3.VcsStatus) { s.AheadCount = 1 })
	for _, c := range []struct {
		name     string
		st       *t3.VcsStatus
		noRemote bool
		want     string
	}{
		{"dirty", dirty, false, "Commit, Push (off), Create PR (off)"},
		{"ahead", ahead, false, "Commit (off), Push, Create PR"},
		{"no remote", dirty, true, "Commit"},
		{"open PR", status(func(s *t3.VcsStatus) { s.AheadCount, s.PR = 1, &t3.VcsChangeRequest{State: "open"} }), false, "Commit (off), Push"},
	} {
		if got := labels(buildGitMenu(c.st, false, !c.noRemote)); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	if got := labels(buildGitMenu(ahead, true, true)); got != "Commit (off), Push (off), Create PR (off)" {
		t.Errorf("busy: %s", got)
	}
}

// A row that's off says why, as T3's getMenuActionDisabledReason does,
// first reason wins; a row that's on says nothing.
func TestGitMenuReasons(t *testing.T) {
	reasons := func(st *t3.VcsStatus, busy, remote bool) string {
		var out []string
		for _, it := range buildGitMenu(st, busy, remote) {
			out = append(out, it.label+": "+gitMenuReason(it, st, busy, remote))
		}
		return strings.Join(out, " | ")
	}
	for _, c := range []struct {
		name   string
		st     *t3.VcsStatus
		busy   bool
		remote bool
		want   string
	}{
		{"clean, nothing ahead", status(nil), false, true,
			"Commit: Worktree is clean. Make changes before committing. | Push: No local commits to push. | Create PR: No local commits to include in a pull request."},
		{"dirty", status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges = true }), false, true,
			"Commit:  | Push: Commit or stash local changes before pushing. | Create PR: No local commits to include in a pull request."},
		{"behind and ahead", status(func(s *t3.VcsStatus) { s.AheadCount, s.BehindCount = 2, 1 }), false, true,
			"Commit: Worktree is clean. Make changes before committing. | Push: Branch is behind upstream. Pull/rebase before pushing. | Create PR: Branch is behind upstream. Pull/rebase before creating a pull request."},
		{"detached on GitLab", status(func(s *t3.VcsStatus) {
			s.RefName, s.AheadCount, s.SourceControlProvider = "", 1, &t3.SourceControlProvider{Kind: "gitlab"}
		}), false, true,
			"Commit: Worktree is clean. Make changes before committing. | Push: Detached HEAD: check out a branch before pushing. | Create MR: Detached HEAD: check out a branch before creating a merge request."},
		{"busy", status(func(s *t3.VcsStatus) { s.AheadCount = 1 }), true, true,
			"Commit: Git action in progress. | Push: Git action in progress. | Create PR: Git action in progress."},
	} {
		if got := reasons(c.st, c.busy, c.remote); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

// The Changes row counts the branch's changes when it has any, else the
// working tree's, and names what it counts against.
func TestChangesTotals(t *testing.T) {
	st := status(nil)
	st.WorkingTree.Insertions, st.WorkingTree.Deletions = 3, 1
	if ins, del, base := changesTotals(st); ins != 3 || del != 1 || base != "" {
		t.Errorf("working tree: +%d -%d %q", ins, del, base)
	}
	st.BranchChanges = &struct {
		BaseRef    string `json:"baseRef"`
		Insertions int    `json:"insertions"`
		Deletions  int    `json:"deletions"`
	}{BaseRef: "main"}
	if ins, del, base := changesTotals(st); ins != 3 || del != 1 || base != "" {
		t.Errorf("an empty branch range keeps the dirty tree: +%d -%d %q", ins, del, base)
	}
	st.BranchChanges.Insertions, st.BranchChanges.Deletions = 2382, 7
	if ins, del, base := changesTotals(st); ins != 2382 || del != 7 || base != "main" {
		t.Errorf("branch: +%d -%d %q", ins, del, base)
	}
	st.WorkingTree.Insertions, st.WorkingTree.Deletions = 0, 0
	st.BranchChanges.Insertions, st.BranchChanges.Deletions = 0, 0
	if _, _, base := changesTotals(st); base != "main" {
		t.Errorf("all clean: base %q, want the branch's", base)
	}
}

// A project without its own icon gets T3's monogram and colour from its
// name (projectIdentity.ts).
func TestProjectIdentity(t *testing.T) {
	for _, c := range []struct{ name, mono string }{
		{"atlas", "AS"}, {"atlas web", "AW"}, {"web2 app", "W2"}, {"x", "XX"}, {"--", "PR"},
	} {
		if mono, color := projectIdentity(c.name); mono != c.mono || color == "" {
			t.Errorf("%q: %q %q, want %q", c.name, mono, color, c.mono)
		}
	}
	if _, a := projectIdentity("atlas"); func() string { _, b := projectIdentity("Atlas"); return b }() != a {
		t.Error("the colour should ignore case")
	}
	ic := projectIcon(t3.Project{Title: "atlas", ProjectIcon: &t3.ProjectIcon{Kind: "lucide", Name: "message-square-dashed", Color: "gray"}})
	if ic.Lucide != "message-square-dashed" || ic.Color != "gray" {
		t.Errorf("lucide icon: %+v", ic)
	}
	if ic := projectIcon(t3.Project{Title: "atlas", ProjectIcon: &t3.ProjectIcon{Kind: "lucide", Name: "box", Color: "red", MonogramText: "AT"}}); ic.Monogram != "AT" || ic.Lucide != "" {
		t.Errorf("legacy monogram: %+v", ic)
	}
}

// Editors open folders on another machine through their own URL schemes,
// as T3's remoteOpen.test.ts expects.
func TestRemoteOpenURL(t *testing.T) {
	for _, c := range []struct{ editor, host, user, path, want string }{
		{"vscode", "sol", "", "/work/project", "vscode://vscode-remote/ssh-remote+sol/work/project"},
		{"zed", "sol", "", "/work/project", "zed://ssh/sol/work/project"},
		{"idea", "sol", "", "/work/project", "jetbrains://gateway/ssh/environment?h=sol&launchIde=true&ideHint=IU&projectHint=%2Fwork%2Fproject"},
		{"vscode", "server", "admin-user", "/work/project", "vscode://vscode-remote/ssh-remote+admin-user%40server/work/project"},
		{"zed", "server", "admin-user", "/work/project", "zed://ssh/admin-user@server/work/project"},
		{"cursor", "server.tail1234.ts.net", "admin-user", "/work/project", "cursor://vscode-remote/ssh-remote+admin-user%40server.tail1234.ts.net/work/project"},
		{"vscode", "sol", "", `C:\Users\theo`, "vscode://vscode-remote/ssh-remote+sol/C%3A/Users/theo"},
		{"zed", "sol", "", `C:\Users\theo`, "zed://ssh/sol/Users/theo"},
		{"zed", "sol", "", "/C:/project", "zed://ssh/sol/C%3A/project"},
		{"trae", "sol", "", "/work/project", ""},
	} {
		if got := remoteOpenURL(c.editor, c.host, c.user, c.path); got != c.want {
			t.Errorf("%s on %s: %q, want %q", c.editor, c.host, got, c.want)
		}
	}
	// A plus in the user goes as hex JSON, which Remote-SSH reads back.
	got := remoteOpenURL("vscode", "ssh.example.com", "vm+demo", "/workspace/project")
	if want := "vscode://vscode-remote/ssh-remote+7b22686f73744e616d65223a227373682e6578616d706c652e636f6d222c2275736572223a22766d2b64656d6f227d/workspace/project"; got != want {
		t.Errorf("plus: %q, want %q", got, want)
	}
}

// cardApp is testApp with the open thread's checkout loaded: a worktree
// thread on a feature branch, or whatever edit makes of it.
func cardApp(t *testing.T, st *t3.VcsStatus, edit func(a *App, e *envState, th *t3.Thread)) (*App, *ui.Scene) {
	t.Helper()
	a := testApp(t)
	e := a.envs[0]
	th := e.threads["t1"]
	th.LatestRunID = "run-1"
	e.config = &t3.ServerConfig{ScratchWorkspaceRoot: "/srv/scratch"}
	if edit != nil {
		edit(a, e, th)
	}
	a.openThread("studio/t1")
	a.mu.Lock()
	v := a.watchVcsLocked(e, gitCwd(th.WorktreePath, e.projects[th.ProjectID]))
	v.status, v.refs = st, &t3.VcsRefs{IsRepo: true}
	a.mu.Unlock()
	return a, built(a, time.Now())
}

// The card's rows follow the thread (§4.4): a started thread's machine row
// is locked; a clean branch offers its quick action and counts nothing.
func TestCardRows(t *testing.T) {
	_, s := cardApp(t, status(nil), nil)
	d := s.Card
	if d == nil || d.Run == nil || d.OpenIn == nil || d.Scripts == nil || d.Branch == nil || d.Git == nil || d.Changes == nil {
		t.Fatalf("rows: %+v", d)
	}
	if !d.Run.Locked || d.Run.Label != "studio" || d.Run.Workspace != "folder" || d.Run.Machine != "server" {
		t.Errorf("run context: %+v", d.Run)
	}
	if d.Branch.Label != "feature" || d.Branch.Disabled {
		t.Errorf("branch: %+v", d.Branch)
	}
	if d.Git.Label != "Commit" || d.Git.Hint != "Branch is up to date. No action needed." {
		t.Errorf("git: %+v", d.Git)
	}
	if d.Changes.Insertions != 0 || d.Changes.Base != "Uncommitted" {
		t.Errorf("changes: %+v", d.Changes)
	}
	if d.Scripts.Primary != "" || d.Scripts.Any {
		t.Errorf("scripts: %+v", d.Scripts)
	}
	if h := s.Header; h == nil || h.Project != "atlas" || !h.Server || s.Title != "On studio" || h.Icon.Monogram == "" {
		t.Errorf("header: %+v %q", h, s.Title)
	}
}

// Dirty on the default branch it's "Commit & push", against the branch's
// base; a worktree thread's machine row names its worktree's folder.
func TestCardRowsWorktree(t *testing.T) {
	st := status(func(s *t3.VcsStatus) { s.HasWorkingTreeChanges, s.IsDefaultRef = true, true })
	st.BranchChanges = &struct {
		BaseRef    string `json:"baseRef"`
		Insertions int    `json:"insertions"`
		Deletions  int    `json:"deletions"`
	}{"main", 12, 3}
	_, s := cardApp(t, st, func(a *App, e *envState, th *t3.Thread) {
		th.WorktreePath, th.Branch = "/srv/worktrees/atlas/fix-login", "fix-login"
		e.projects["p1"] = t3.Project{ID: "p1", Title: "atlas", WorkspaceRoot: "/srv/atlas",
			Scripts: []t3.ProjectScript{{ID: "setup", Name: "Install", RunOnWorktreeCreate: true}, {ID: "test", Name: "Test", Icon: "test"}}}
	})
	d := s.Card
	if d.Git.Label != "Commit & push" || d.Git.Icon != "cloud-upload" || d.Git.Hint != "" {
		t.Errorf("git: %+v", d.Git)
	}
	if d.Changes.Insertions != 12 || d.Changes.Deletions != 3 || d.Changes.Base != "vs main" {
		t.Errorf("changes: %+v", d.Changes)
	}
	if d.Run.Label != "studio" || d.Run.Workspace != "folder-git-2" || d.Run.WorkspaceTip != "/srv/worktrees/atlas/fix-login" {
		t.Errorf("run context: %+v", d.Run)
	}
	if d.Scripts.Primary != "Test" || d.Scripts.Icon != "test" {
		t.Errorf("scripts: %+v", d.Scripts)
	}
}

// A folder that isn't a repository offers Initialize Git and no branch or
// changes; the No project folder offers no Git at all and says "No project".
func TestCardRowsWithoutGit(t *testing.T) {
	notRepo := &t3.VcsStatus{}
	_, s := cardApp(t, notRepo, nil)
	if d := s.Card; !d.InitGit || d.Branch != nil || d.Git != nil || d.Changes != nil {
		t.Errorf("not a repo: %+v", d)
	}
	_, s = cardApp(t, status(nil), func(a *App, e *envState, th *t3.Thread) {
		e.projects["p1"] = t3.Project{ID: "p1", Title: "Scratch", WorkspaceRoot: "/srv/scratch",
			ProjectIcon: &t3.ProjectIcon{Kind: "lucide", Name: "message-square-dashed", Color: "gray"}}
	})
	if d := s.Card; d.InitGit || d.Branch != nil || d.Git != nil || d.Changes != nil || d.OpenIn == nil {
		t.Errorf("no project: %+v", d)
	}
	if h := s.Header; h.Project != "No project" || h.Icon.Lucide != "message-square-dashed" {
		t.Errorf("no project header: %+v", h)
	}
}

// A thread that hasn't run keeps the Git and Changes rows quiet while
// there's nothing to do; a subagent's header crumbs its parent.
func TestCardQuietAndLineage(t *testing.T) {
	_, s := cardApp(t, status(nil), func(a *App, e *envState, th *t3.Thread) {
		th.LatestRunID = ""
		th.Lineage = &t3.Lineage{ParentThreadID: "same", RelationshipToParent: "subagent"}
	})
	if d := s.Card; d.Git != nil || d.Changes != nil || d.Branch == nil {
		t.Errorf("quiet: %+v", d)
	}
	if s.Header.Parent != "Same id on studio" {
		t.Errorf("parent crumb %q", s.Header.Parent)
	}
}

// A draft's machine row opens its workspace menu; picking New worktree
// makes the branch row its base, and the launch asks for a worktree.
func TestDraftWorkspace(t *testing.T) {
	a := testApp(t)
	e := a.envs[0]
	e.config = &t3.ServerConfig{}
	a.newDraft()
	a.mu.Lock()
	v := a.watchVcsLocked(e, e.projects[a.draft].WorkspaceRoot)
	v.status, v.refs = status(func(s *t3.VcsStatus) { s.RefName = "main" }), &t3.VcsRefs{}
	a.mu.Unlock()
	th := ui.NewTheme()
	s := built(a, time.Now())
	if s.Card.Run.Locked || s.Header.Server || s.Title != "New thread" {
		t.Fatalf("draft: run %+v header %+v", s.Card.Run, s.Header)
	}
	a.handle(th, ui.Action{ID: "card.run", Kind: "click"})
	s = built(a, time.Now())
	if s.Card.Menu == nil || s.Card.Menu.For != "run" || len(s.Card.Menu.Rows) != 3 || !s.Card.Menu.Rows[1].Checked {
		t.Fatalf("run menu: %+v", s.Card.Menu)
	}
	a.handle(th, ui.Action{ID: "card.run:2", Kind: "click"})
	s = built(a, time.Now())
	if s.Card.Menu != nil || s.Card.Run.Workspace != "folder-git-2" || s.Card.Run.Label != "studio" || s.Card.Branch.Label != "From main" {
		t.Fatalf("new worktree: run %+v branch %+v", s.Card.Run, s.Card.Branch)
	}
	a.mu.Lock()
	strategy, err := a.draftStrategyLocked()
	a.mu.Unlock()
	if err != nil || strategy["type"] != "worktree" || strategy["baseRef"] != "main" {
		t.Fatalf("launch strategy %v, %v", strategy, err)
	}
	// Picking the new worktree's base offers no ref to create.
	a.handle(th, ui.Action{ID: "card.branch", Kind: "click"})
	a.handle(th, ui.Action{ID: "branch.search", Kind: "change", Text: "spike"})
	if p := built(a, time.Now()).Card.Picker; p == nil || p.Create != "" {
		t.Fatalf("picking a base: %+v", p)
	}
}

// The draft's workspace menu offers the worktree the project's threads
// worked in last, not archived (resolvePreviousWorktreeSeed), its branch
// under the label; taking it makes that worktree the draft's current one,
// which the launch reuses, on its branch.
func TestDraftPreviousWorktree(t *testing.T) {
	a := testApp(t)
	e := a.envs[0]
	e.config = &t3.ServerConfig{}
	at := func(d time.Duration) string { return time.Now().UTC().Add(-d).Format(time.RFC3339Nano) }
	e.threads["w1"] = &t3.Thread{ID: "w1", ProjectID: "p1", WorktreePath: "/work/atlas-wt/old", Branch: "old", UpdatedAt: at(time.Hour)}
	e.threads["w2"] = &t3.Thread{ID: "w2", ProjectID: "p1", WorktreePath: "/work/atlas-wt/lexer", Branch: "fix/lexer", UpdatedAt: at(time.Minute)}
	e.threads["w3"] = &t3.Thread{ID: "w3", ProjectID: "p1", WorktreePath: "/work/atlas-wt/gone", Branch: "gone", UpdatedAt: at(0), ArchivedAt: at(0)}
	a.draftIn("studio", "p1")
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.run", Kind: "click"})
	m := built(a, time.Now()).Card.Menu
	if m == nil || len(m.Rows) != 4 {
		t.Fatalf("run menu %+v", m)
	}
	if r := m.Rows[3]; r.Kind != ui.MenuRadioRow || r.Label != "Previous worktree" || r.Detail != "fix/lexer" || r.Icon != "history" || r.Checked {
		t.Fatalf("previous worktree row %+v", r)
	}
	a.handle(th, ui.Action{ID: "card.run:3", Kind: "click"})
	s := built(a, time.Now())
	if r := s.Card.Run; r.Workspace != "folder-git" || r.WorkspaceTip != "/work/atlas-wt/lexer" {
		t.Fatalf("run context %+v", r)
	}
	a.handle(th, ui.Action{ID: "card.run", Kind: "click"})
	m = built(a, time.Now()).Card.Menu
	if r := m.Rows[1]; r.Label != "Current worktree" || r.Icon != "folder-git" || !r.Checked {
		t.Fatalf("the current row %+v", r)
	}
	if r := m.Rows[3]; r.Detail != "old" {
		t.Fatalf("the previous worktree is now the one before: %+v", r)
	}
	a.mu.Lock()
	strategy, err := a.draftStrategyLocked()
	a.mu.Unlock()
	if err != nil || strategy["type"] != "existing_worktree" || strategy["worktreePath"] != "/work/atlas-wt/lexer" || strategy["branch"] != "fix/lexer" {
		t.Fatalf("launch strategy %v, %v", strategy, err)
	}
}

// The header's project crumb starts a new thread in that project.
func TestHeaderProjectCrumb(t *testing.T) {
	a := testApp(t)
	a.openThread("mac/same")
	a.handle(ui.NewTheme(), ui.Action{ID: "header.project", Kind: "click"})
	if s := built(a, time.Now()); s.Header.Server || s.Title != "New thread" || s.Header.Project != "nimbus" || s.Card.Run.Label != "MacBook" {
		t.Fatalf("header %+v %q, run %+v", s.Header, s.Title, s.Card.Run)
	}
}

// The title opens the open thread's actions menu; the press outside it
// closes it before a second click, which opens it again, as T3's does,
// and Escape closes it. A right-click anywhere in the header opens it too.
func TestHeaderMenu(t *testing.T) {
	a := testApp(t)
	th := ui.NewTheme()
	a.openThread("studio/t1")
	open := func() bool {
		s := built(a, time.Now())
		return s.Menu != nil && len(s.Menu.Items) > 0 && s.Menu.Items[0].Label == "Rename thread"
	}
	a.handle(th, ui.Action{ID: "header.title", Kind: "click"})
	if !open() {
		t.Fatal("the title should open the thread's menu")
	}
	a.handle(th, ui.Action{ID: "menus.close", Kind: "click"})
	a.handle(th, ui.Action{ID: "header.title", Kind: "click"})
	if !open() {
		t.Fatal("a second click should open it again")
	}
	a.handle(th, ui.Action{ID: "menus.escape", Kind: "click"})
	if open() {
		t.Fatal("Escape should close it")
	}
	a.handle(th, ui.Action{ID: "header", Kind: "context"})
	if !open() {
		t.Fatal("a right-click in the header should open it")
	}
}

// The card's own menu, at a right-click, turns its rows off and on and
// resets them, staying open as T3's checkbox items do, and the choice
// persists; the run context and branch rows' right-click offers to copy
// the path or the branch instead, in the in-page menu.
func TestCardItemsMenu(t *testing.T) {
	a, _ := cardApp(t, status(nil), nil)
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card", Kind: "context"})
	m := built(a, time.Now()).Card.Menu
	if m == nil || m.For != "items" || !m.AtPointer || len(m.Rows) != 9 || m.Rows[0].Label != "Show in thread details" ||
		m.Rows[5].Label != "Git actions" || !m.Rows[5].Checked || m.Rows[8].Label != "Reset to default" || !m.Rows[8].Disabled {
		t.Fatalf("menu %+v", m)
	}
	a.handle(th, ui.Action{ID: "card.items:5", Kind: "click"})
	if d := built(a, time.Now()).Card; d.Git != nil || d.Changes == nil || d.Menu == nil || d.Menu.Rows[5].Checked {
		t.Fatalf("Git actions off: %+v", d)
	}
	var stored map[string]bool
	if ok, _ := a.store.Get("details-items", &stored); !ok || !stored["gitActions"] {
		t.Fatalf("stored %v", stored)
	}
	a.handle(th, ui.Action{ID: "card.openin", Kind: "context"})
	if m := built(a, time.Now()).Card.Menu; m.Rows[5].Checked || m.Rows[8].Disabled {
		t.Fatalf("after: %+v", m)
	}
	a.handle(th, ui.Action{ID: "card.items:8", Kind: "click"})
	if d := built(a, time.Now()).Card; d.Git == nil || d.Menu == nil {
		t.Fatal("Reset should bring the row back and leave the menu up")
	}
	for _, c := range []struct{ id, label string }{{"card.branch", "Copy branch name"}, {"card.runlocked", "Copy full path"}, {"card.workspace", "Copy full path"}} {
		a.handle(th, ui.Action{ID: c.id, Kind: "context"})
		s := built(a, time.Now())
		if s.Menu == nil || len(s.Menu.Items) != 1 || s.Menu.Items[0].Label != c.label || s.Menu.Items[0].Icon != "copy" || s.Card.Menu != nil {
			t.Fatalf("%s's menu %+v", c.id, s.Menu)
		}
	}
	a.handle(th, ui.Action{ID: "ctx:copy-full-path", Kind: "click"})
	if s := built(a, time.Now()); s.Menu != nil {
		t.Fatal("copying closes it")
	}
}

// The inline card hides per thread, and the choice persists.
func TestCardToggle(t *testing.T) {
	a := testApp(t)
	th := ui.NewTheme()
	a.openThread("studio/t1")
	if s := built(a, time.Now()); !s.DetailsCard {
		t.Fatal("the card is shown by default")
	}
	a.handle(th, ui.Action{ID: "details:card", Kind: "click"})
	if s := built(a, time.Now()); s.DetailsCard {
		t.Fatal("toggled off")
	}
	a.openThread("mac/same")
	if s := built(a, time.Now()); !s.DetailsCard {
		t.Fatal("another thread keeps its card")
	}
	var hidden map[string]bool
	if ok, _ := a.store.Get("details-hidden", &hidden); !ok || !hidden["studio/t1"] {
		t.Fatalf("stored %v", hidden)
	}
}
