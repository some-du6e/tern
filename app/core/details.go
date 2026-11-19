package core

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// T3's chat header and thread details card, built from the open thread or
// draft (docs/ui/05-header.md §4.4, §5). The Git rules are ports of
// GitActionsControl.logic.ts and its client-runtime half; the run context
// labels are BranchToolbar.logic.ts's.

// projectColors are T3's project colours, in PROJECT_ICON_COLORS order.
var projectColors = []string{"gray", "red", "orange", "amber", "yellow", "lime", "green", "emerald", "teal",
	"cyan", "sky", "blue", "indigo", "violet", "purple", "fuchsia", "pink", "rose"}

// projectIdentity is the monogram and colour T3 derives from a project's
// name when it has no icon of its own (projectIdentity.ts): the first
// word's first letter and either its first digit, the last word's first
// letter or the first word's last letter; a colour hashed from the name.
func projectIdentity(name string) (monogram, color string) {
	name = strings.TrimSpace(name)
	words := strings.FieldsFunc(name, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	monogram = "PR"
	if len(words) > 0 {
		glyphs := []rune(words[0])
		second := glyphs[len(glyphs)-1]
		if len(words) > 1 {
			second = []rune(words[len(words)-1])[0]
		}
		for _, g := range glyphs[1:] {
			if unicode.IsNumber(g) {
				second = g
				break
			}
		}
		monogram = strings.ToUpper(string([]rune{glyphs[0], second}))
		if r := []rune(monogram); len(r) > 2 {
			monogram = string(r[:2])
		}
	}
	seed := strings.ToLower(name)
	if seed == "" {
		seed = "project"
	}
	i := 0
	for _, g := range seed {
		i = (i*31 + int(g)) % len(projectColors)
	}
	return monogram, projectColors[i]
}

// projectIcon is a project's mark: the icon it chose (a legacy lucide
// icon with a monogram is that monogram), else its derived monogram.
// tern shows no favicons and no emoji, so those fall back to the monogram.
func projectIcon(p t3.Project) ui.ProjectIcon {
	mono, color := projectIdentity(p.Title)
	if ic := p.ProjectIcon; ic != nil {
		text := ic.Text
		if ic.Kind == "lucide" {
			text = ic.MonogramText
			if text == "" {
				text = ic.Monogram
			}
			if text == "" {
				return ui.ProjectIcon{Lucide: ic.Name, Color: ic.Color}
			}
		}
		if text != "" {
			return ui.ProjectIcon{Monogram: text, Color: ic.Color}
		}
	}
	return ui.ProjectIcon{Monogram: mono, Color: color}
}

// isScratch reports whether p is the environment's No project folder.
func isScratch(p t3.Project, cfg *t3.ServerConfig) bool {
	return cfg != nil && cfg.ScratchWorkspaceRoot != "" && strings.TrimRight(p.WorkspaceRoot, "/\\") == strings.TrimRight(cfg.ScratchWorkspaceRoot, "/\\")
}

// headerScene fills the chat header for thread t (nil for a draft) in
// project p.
func (a *App) headerScene(s *ui.Scene, e *envState, t *t3.Thread, p t3.Project, hasProject bool, now time.Time) {
	h := &ui.ChatHeader{Server: t != nil}
	if hasProject {
		h.Project, h.Icon = p.Title, projectIcon(p)
		if isScratch(p, e.config) {
			h.Project = "No project"
		}
	}
	if t != nil {
		if isSubagent(t) {
			h.Parent = "Parent thread"
			if parent := e.threads[t.Lineage.ParentThreadID]; parent != nil && parent.Title != "" {
				h.Parent = parent.Title
			}
		}
		if t.LinkOrigin != nil {
			h.Origin = t.LinkOrigin.Label
		}
		h.CanRename = true
		h.Renaming = a.renaming == a.openID
		if v := a.view; v != nil && v.env == e && v.id == t.ID && v.guestShare != nil {
			h.Guest = &ui.GuestLine{Status: guestStatus(v.guestShare, now), Closing: v.unsharing}
		}
	}
	s.Header = h
}

// guestStatus is what the header says of a guest link (GuestShareStatus):
// expired once T3's minute clock reaches the minute it expires in, else
// who redeemed it, else that it's waiting for someone to.
func guestStatus(gs *t3.GuestShare, now time.Time) string {
	exp := gs.ExpiresAt
	if len(exp) > 16 {
		exp = exp[:16]
	}
	switch {
	case exp <= now.UTC().Format("2006-01-02T15:04"):
		return "Guest link expired"
	case gs.GuestName != "":
		return gs.GuestName + " is a guest"
	}
	return "Guest link active"
}

// ---------- Git, as GitActionsControl.logic.ts has it ----------

// changeTerms are a host's words for a change request: GitLab's merge
// request, everyone else's pull request.
type changeTerms struct{ short, singular string }

func termsFor(st *t3.VcsStatus) changeTerms {
	if st != nil && st.SourceControlProvider != nil && st.SourceControlProvider.Kind == "gitlab" {
		return changeTerms{"MR", "merge request"}
	}
	return changeTerms{"PR", "pull request"}
}

// providerKind is the status's host for its mark.
func providerKind(st *t3.VcsStatus) string {
	if st != nil && st.SourceControlProvider != nil {
		return st.SourceControlProvider.Kind
	}
	return ""
}

func aheadOfDefault(st *t3.VcsStatus) int {
	if st.AheadOfDefaultCount != nil {
		return *st.AheadOfDefaultCount
	}
	return st.AheadCount
}

// quickAction is what the Git row's primary does: run a stacked action,
// pull, publish, or explain why there's nothing to do.
type quickAction struct {
	label    string
	kind     string // run_action, run_pull, open_publish, show_hint
	action   string // commit, push, create_pr, commit_push, commit_push_pr
	hint     string
	disabled bool
}

// resolveQuickAction picks the quick action for a status, first match
// wins (docs/ui/05-header.md §4.5).
func resolveQuickAction(st *t3.VcsStatus, busy, isDefault, hasPrimaryRemote bool) quickAction {
	hint := func(label, why string) quickAction {
		return quickAction{label: label, kind: "show_hint", hint: why, disabled: true}
	}
	run := func(label, action string) quickAction {
		return quickAction{label: label, kind: "run_action", action: action}
	}
	if busy {
		return hint("Commit", "Git action in progress.")
	}
	if st == nil {
		return hint("Commit", "Git status is unavailable.")
	}
	terms := termsFor(st)
	openPR := st.PR != nil && st.PR.State == "open"
	ahead, behind := st.AheadCount > 0, st.BehindCount > 0
	pushAction := "push"
	if isDefault {
		pushAction = "commit_push"
	}
	switch {
	case st.RefName == "":
		return hint("Commit", "Create and checkout a ref before pushing or opening a "+terms.singular+".")
	case st.HasWorkingTreeChanges && (behind || (!st.HasUpstream && !hasPrimaryRemote)):
		return run("Commit", "commit")
	case st.HasWorkingTreeChanges && (openPR || isDefault):
		return run("Commit & push", "commit_push")
	case st.HasWorkingTreeChanges:
		return run("Commit, push & "+terms.short, "commit_push_pr")
	case !st.HasUpstream && !hasPrimaryRemote:
		return quickAction{label: "Publish repository", kind: "open_publish"}
	case !st.HasUpstream && !ahead && openPR:
		return hint("Commit", "Branch is up to date. No action needed.")
	case !st.HasUpstream && !ahead:
		return hint("Push", "No local commits to push.")
	case !st.HasUpstream && (openPR || isDefault):
		return run("Push", pushAction)
	case !st.HasUpstream:
		return run("Push & create "+terms.short, "create_pr")
	case ahead && behind:
		return quickAction{label: "Sync ref", kind: "show_hint", hint: "Branch has diverged from upstream. Rebase/merge first.", disabled: true}
	case behind:
		return quickAction{label: "Pull", kind: "run_pull"}
	case ahead && (openPR || isDefault):
		return run("Push", pushAction)
	case ahead:
		return run("Push & create "+terms.short, "create_pr")
	case openPR:
		return hint("Commit", "Branch is up to date. No action needed.")
	case aheadOfDefault(st) > 0 && !isDefault:
		return run("Create "+terms.short, "create_pr")
	}
	return hint("Commit", "Branch is up to date. No action needed.")
}

// quickActionIcon is the icon for a quick action: cloud-upload to publish
// or push, cloud-download to pull, a commit for commits, the host's mark
// for pull requests, info for the rest.
func quickActionIcon(q quickAction, provider string) string {
	switch {
	case q.kind == "open_publish":
		return "cloud-upload"
	case q.kind == "run_pull":
		return "cloud-download"
	case q.kind == "run_action" && q.action == "commit":
		return "git-commit-horizontal"
	case q.kind == "run_action" && (q.action == "push" || q.action == "commit_push"):
		return "cloud-upload"
	case q.kind == "run_action":
		return "provider:" + provider
	case q.label == "Commit":
		return "git-commit-horizontal"
	case q.label == "Push":
		return "cloud-upload"
	}
	return "info"
}

// gitMenuItem is a row of the Git options menu.
type gitMenuItem struct {
	id, label string
	disabled  bool
}

// buildGitMenu is the options menu: Commit, then Push and Create PR when
// there's a remote, dropping Create PR while one is open.
func buildGitMenu(st *t3.VcsStatus, busy, hasPrimaryRemote bool) []gitMenuItem {
	if st == nil {
		return nil
	}
	terms := termsFor(st)
	hasBranch := st.RefName != ""
	openPR := st.PR != nil && st.PR.State == "open"
	behind := st.BehindCount > 0
	pushable := st.HasUpstream || (hasPrimaryRemote && !st.HasUpstream)
	commit := gitMenuItem{"commit", "Commit", busy || !st.HasWorkingTreeChanges}
	if !hasPrimaryRemote {
		return []gitMenuItem{commit}
	}
	push := gitMenuItem{"push", "Push", busy || !hasBranch || behind || st.AheadCount == 0 || !pushable}
	if openPR {
		return []gitMenuItem{commit, push}
	}
	pr := gitMenuItem{"pr", "Create " + terms.short, busy || !hasBranch || openPR || aheadOfDefault(st) == 0 || behind || !pushable}
	return []gitMenuItem{commit, push, pr}
}

// gitMenuReason is why a Git menu row is off (getMenuActionDisabledReason),
// "" while it's on.
func gitMenuReason(it gitMenuItem, st *t3.VcsStatus, busy, hasPrimaryRemote bool) string {
	switch {
	case !it.disabled:
		return ""
	case busy:
		return "Git action in progress."
	case st == nil:
		return "Git status is unavailable."
	}
	terms := termsFor(st)
	detached, dirty, behind := st.RefName == "", st.HasWorkingTreeChanges, st.BehindCount > 0
	noRemote := !st.HasUpstream && !hasPrimaryRemote
	switch it.id {
	case "commit":
		if !dirty {
			return "Worktree is clean. Make changes before committing."
		}
		return "Commit is currently unavailable."
	case "push":
		switch {
		case detached:
			return "Detached HEAD: check out a branch before pushing."
		case dirty:
			return "Commit or stash local changes before pushing."
		case behind:
			return "Branch is behind upstream. Pull/rebase before pushing."
		case noRemote:
			return `Add an "origin" remote before pushing.`
		case st.AheadCount == 0:
			return "No local commits to push."
		}
		return "Push is currently unavailable."
	}
	switch {
	case detached:
		return "Detached HEAD: check out a branch before creating a " + terms.singular + "."
	case noRemote:
		return `Add an "origin" remote before creating a ` + terms.singular + "."
	case aheadOfDefault(st) <= 0:
		return "No local commits to include in a " + terms.singular + "."
	case behind:
		return "Branch is behind upstream. Pull/rebase before creating a " + terms.singular + "."
	}
	return "Create " + terms.singular + " is currently unavailable."
}

// gitMenuIcon is a menu row's icon: the host's mark for the pull request.
func gitMenuIcon(id, provider string) string {
	switch id {
	case "commit":
		return "git-commit-horizontal"
	case "push":
		return "cloud-upload"
	}
	return "provider:" + provider
}

// requiresConfirm reports whether action on the default branch asks first.
func requiresConfirm(action string, isDefault bool) bool {
	return isDefault && (action == "push" || action == "create_pr" || action == "commit_push" || action == "commit_push_pr")
}

// confirmCopy is the default-branch dialog's question.
func confirmCopy(action, branch string, includesCommit bool, terms changeTerms) (title, desc, cont string) {
	suffix := ` on "` + branch + `". You can continue on this ref or create a feature ref and run the same action there.`
	switch {
	case (action == "push" || action == "commit_push") && includesCommit:
		return "Commit & push to default ref?", "This action will commit and push changes" + suffix, "Commit & push to " + branch
	case action == "push" || action == "commit_push":
		return "Push to default ref?", "This action will push local commits" + suffix, "Push to " + branch
	case includesCommit:
		return "Commit, push & create " + terms.short + " from default ref?", "This action will commit, push, and create a " + terms.singular + suffix, "Commit, push & create " + terms.short
	}
	return "Push & create " + terms.short + " from default ref?", "This action will push local commits and create a " + terms.singular + suffix, "Push & create " + terms.short
}

// featureBranchOff is why the dialog can't move the work to a feature
// branch: a clean tree gives the server nothing to name one after.
func featureBranchOff(branch string, dirty bool) string {
	if dirty {
		return ""
	}
	return "There are no changes to commit, so a feature branch can't be named. Create a branch yourself or continue on " + branch + "."
}

// changesTotals is what the Changes row counts (resolveChangesTotals): the
// Diff panel's comparison of the branch when it has file counts, else the
// branch's changes when there are any, else the working tree's, else
// whichever exists. Base is "" for the working tree.
func changesTotals(st *t3.VcsStatus, p *changesPreview) (ins, del int, base string) {
	if p != nil && p.files {
		return p.ins, p.del, p.base
	}
	tree := st.WorkingTree
	if b := st.BranchChanges; b != nil && (b.Insertions > 0 || b.Deletions > 0) {
		return b.Insertions, b.Deletions, b.BaseRef
	}
	if tree.Insertions > 0 || tree.Deletions > 0 || st.BranchChanges == nil {
		return tree.Insertions, tree.Deletions, ""
	}
	b := st.BranchChanges
	return b.Insertions, b.Deletions, b.BaseRef
}

// gitElapsed is how long an action has run: "12s", "1m 5s".
func gitElapsed(d time.Duration) string {
	s := max(0, int(d/time.Second))
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm %ds", s/60, s%60)
}

// ---------- the run context, as BranchToolbar.logic.ts has it ----------

// workspaceLabel names where a thread works: a locked thread's worktree by
// its folder, "Local checkout" or "New worktree"; otherwise "Current
// worktree", "Current checkout" or "New worktree".
func workspaceLabel(worktree string, worktreeMode, locked bool) string {
	switch {
	case locked && worktree != "":
		parts := strings.FieldsFunc(worktree, func(r rune) bool { return r == '/' || r == '\\' })
		if len(parts) == 0 {
			return "Worktree"
		}
		return parts[len(parts)-1]
	case worktreeMode:
		return "New worktree"
	case locked:
		return "Local checkout"
	case worktree != "":
		return "Current worktree"
	}
	return "Current checkout"
}

// machineKind is the environment's machine for its mark
// (resolveEnvironmentMachineKind): the icon its settings chose, else what
// its platform reports, else a server.
func machineKind(cfg *t3.ServerConfig) string {
	switch {
	case cfg != nil && cfg.Settings.EnvironmentIcon != "":
		return cfg.Settings.EnvironmentIcon
	case cfg != nil && cfg.Environment.Platform.Machine != "":
		return cfg.Environment.Platform.Machine
	}
	return "server"
}

// primaryScript is the script the scripts split runs: the one last run in
// this project, else the first that's neither setup nor settle, else the
// first that isn't settle.
func primaryScript(scripts []t3.ProjectScript, last string) *t3.ProjectScript {
	for i := range scripts {
		if scripts[i].ID == last {
			return &scripts[i]
		}
	}
	for i := range scripts {
		if !scripts[i].RunOnWorktreeCreate && !scripts[i].RunOnSettle {
			return &scripts[i]
		}
	}
	for i := range scripts {
		if !scripts[i].RunOnSettle {
			return &scripts[i]
		}
	}
	return nil
}

// scriptLabel is a script's menu label with its roles: "Install (setup)".
func scriptLabel(sc t3.ProjectScript) string {
	var roles []string
	if sc.RunOnWorktreeCreate {
		roles = append(roles, "setup")
	}
	if sc.RunOnSettle {
		roles = append(roles, "on settle")
	}
	if len(roles) == 0 {
		return sc.Name
	}
	return sc.Name + " (" + strings.Join(roles, ", ") + ")"
}
