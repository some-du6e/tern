package core

import (
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The thread details card's rows for the open thread or draft
// (ThreadDetailsPanel.tsx, docs/ui/05-header.md §5.5–5.12).

// vcsState is one checkout's Git status, as subscribeVcsStatus streams
// it, the first page of its refs, and the action running in it.
type vcsState struct {
	cwd     string
	conn    *t3.Conn // the socket its stream is on
	status  *t3.VcsStatus
	err     string // the stream's last failure
	stop    func()
	refs    *t3.VcsRefs
	running *gitRun
	done    *gitDone
	initing bool // vcs.init is running
	busy    bool // a ref is switching
}

// gitRun is a stacked action or pull in progress: its phase's label, the
// last line a hook printed, and when the phase and hook started.
type gitRun struct {
	pull           bool
	label, output  string
	phase, hook    time.Time
	includesCommit bool
}

// gitDone is a finished action's result, shown for ten seconds.
type gitDone struct {
	title, desc string
	until       time.Time
}

// pendingGit is an action on the default branch waiting on the dialog.
type pendingGit struct {
	action, branch string
	includesCommit bool
}

// refPicker is the open branch picker's search and highlight.
type refPicker struct {
	query string
	hi    int
}

func vcsKey(envID, cwd string) string { return envID + "\x00" + cwd }

// gitCwd is where a thread's files and Git live: its worktree, else its
// project's root.
func gitCwd(t *t3.Thread, p t3.Project) string {
	if t != nil && t.WorktreePath != "" {
		return t.WorktreePath
	}
	return p.WorkspaceRoot
}

// cardScene fills the details card for thread t (nil for the draft) in
// project p, and makes sure its checkout's status is streaming.
func (a *App) cardScene(s *ui.Scene, e *envState, t *t3.Thread, p t3.Project, hasProject bool, now time.Time) {
	d := &ui.ThreadDetails{}
	s.Card = d
	if !hasProject {
		return
	}
	cwd := gitCwd(t, p)
	v := a.watchVcsLocked(e, cwd)
	st := v.status
	isRepo := st == nil || st.IsRepo
	scratch := isScratch(p, e.config)
	offersGit := cwd != "" && e.config != nil && !scratch

	// The run context.
	worktree := ""
	if t != nil {
		worktree = t.WorktreePath
	}
	worktreeMode := worktree != "" || (t == nil && a.draftWorktree)
	locked := t != nil
	label := workspaceLabel(worktree, worktreeMode, locked)
	icon := "folder"
	switch {
	case worktreeMode:
		icon = "folder-git-2"
	case worktree != "":
		icon = "folder-git"
	}
	tip := label
	if path := worktree; path != "" || !worktreeMode {
		if path == "" {
			path = p.WorkspaceRoot
		}
		tip = path
	}
	d.Run = &ui.RunContext{Machine: machineKind(e.config), Label: e.env.Label, Tip: e.env.Label, Workspace: icon, WorkspaceTip: tip, Locked: locked}
	if !locked && a.cardMenu == "run" {
		d.Menu = &ui.CardMenu{For: "run", Rows: runMenu(worktree, a.draftWorktree)}
	}

	// Open in editor.
	mode, target := remoteOpen(e)
	choices := a.editorChoices(e, mode)
	pref := preferredEditor(choices, a.lastEditor)
	o := &ui.OpenIn{Editor: pref, Label: "Open in editor", Disabled: pref == "" || cwd == "" || mode == openNone}
	if pref != "" {
		o.Label = "Open in " + editorLabel(pref)
	}
	d.OpenIn = o
	if a.cardMenu == "openin" {
		d.Menu = &ui.CardMenu{For: "openin", Rows: editorMenu(choices, pref, mode, target, e.env.Label, a.hintSeen, cwd == "")}
	}

	// Project scripts.
	sc := &ui.Scripts{Any: len(p.Scripts) > 0}
	if ps := primaryScript(p.Scripts, a.lastScripts[p.ID]); ps != nil {
		sc.Primary, sc.Icon = ps.Name, ps.Icon
	}
	d.Scripts = sc
	if a.cardMenu == "scripts" && sc.Any {
		var rows []ui.MenuRow
		for _, s := range p.Scripts {
			icon := scriptIcon[s.Icon]
			if icon == "" {
				icon = "play"
			}
			rows = append(rows, ui.MenuRow{Label: scriptLabel(s), Icon: icon})
		}
		d.Menu = &ui.CardMenu{For: "scripts", Rows: rows}
	}

	if !offersGit {
		return
	}
	if isRepo {
		value := branchValue(st, t)
		if worktreeMode && worktree == "" && a.draftBase != "" {
			value = a.draftBase
		}
		b := &ui.BranchRow{Label: "Select ref", Disabled: v.refs == nil || v.busy}
		switch {
		case value != "" && worktreeMode && worktree == "":
			b.Label = "From " + value
		case value != "":
			b.Label = value
		}
		d.Branch = b
		if a.branchPicker != nil {
			d.Picker = a.refPickerScene(v)
		}
	}

	if !isRepo {
		d.InitGit = true
		return
	}
	busy := v.running != nil
	isDefault := st != nil && st.IsDefaultRef
	hasRemote := st == nil || st.HasPrimaryRemote
	q := resolveQuickAction(st, busy, isDefault, hasRemote)
	ins, del, base, known := 0, 0, "", st != nil
	if known {
		ins, del, base = changesTotals(st)
	}
	done := v.done
	if done != nil && now.After(done.until) {
		done = nil
	}
	// A fresh draft, or a thread that hasn't run, keeps quiet until there's
	// something to commit (quietEmptyDraft).
	quiet := (t == nil || t.LatestRunID == "") && !busy && v.err == "" && q.disabled && ins == 0 && del == 0
	if quiet {
		return
	}
	g := &ui.GitActions{Label: q.label, Icon: quickActionIcon(q, providerKind(st)), Hint: q.hint}
	items := buildGitMenu(st, busy, hasRemote)
	notes := gitNotes(st, v.err)
	usable := len(notes) > 0
	for _, it := range items {
		usable = usable || !it.disabled
	}
	g.MenuDisabled = busy || !usable
	if r := v.running; r != nil {
		g.Running = gitProgress(r, now)
	} else if done != nil {
		g.Success = &ui.GitSuccess{Title: done.title, Description: done.desc}
	}
	d.Git = g
	if a.cardMenu == "git" {
		var rows []ui.MenuRow
		for _, it := range items {
			rows = append(rows, ui.MenuRow{Label: it.label, Icon: gitMenuIcon(it.id, providerKind(st)), Disabled: it.disabled})
		}
		d.Menu = &ui.CardMenu{For: "git", Rows: append(rows, notes...)}
	}
	c := &ui.Changes{Insertions: ins, Deletions: del, Disabled: true}
	if known {
		c.Base = "Uncommitted"
		if base != "" {
			c.Base = "vs " + base
		}
	}
	d.Changes = c
}

// branchValue is the branch the row shows (resolveBranchToolbarValue): the
// checkout's own, else the thread's.
func branchValue(st *t3.VcsStatus, t *t3.Thread) string {
	if st != nil && st.RefName != "" {
		return st.RefName
	}
	if t != nil {
		return t.Branch
	}
	return ""
}

// scriptIcon names each script icon's lucide icon in the menu.
var scriptIcon = map[string]string{
	"play": "play", "test": "flask-conical", "lint": "list-checks", "configure": "wrench", "build": "hammer", "debug": "bug",
}

// runMenu is a draft's run context menu: its workspace, the current
// checkout (or worktree) or a new worktree.
func runMenu(worktree string, newWorktree bool) []ui.MenuRow {
	current := ui.MenuRow{Kind: ui.MenuRadioRow, Label: workspaceLabel(worktree, false, false), Icon: "folder", Checked: !newWorktree}
	if worktree != "" {
		current.Icon = "folder-git"
	}
	return []ui.MenuRow{
		{Kind: ui.MenuGroupLabel, Label: "Workspace"},
		current,
		{Kind: ui.MenuRadioRow, Label: "New worktree", Icon: "folder-git-2", Checked: newWorktree},
	}
}

// editorMenu lists the editors to make the preferred one, the preferred
// with Open in's shortcut, or says why there are none.
func editorMenu(choices []string, pref string, mode openMode, target t3.RemoteOpenTarget, machine string, hintSeen, noFolder bool) []ui.MenuRow {
	switch {
	case mode == openNone:
		return []ui.MenuRow{{Label: "No SSH route to " + machine, Disabled: true}}
	case mode == openRemote && len(choices) == 0:
		return []ui.MenuRow{{Label: "Open from a computer to edit on " + machine, Disabled: true}}
	case len(choices) == 0:
		return []ui.MenuRow{{Label: "No installed editors found", Disabled: true}}
	}
	var rows []ui.MenuRow
	for _, id := range choices {
		r := ui.MenuRow{Label: editorLabel(id), Icon: "square-arrow-out-up-right", Disabled: noFolder}
		if id == pref {
			r.Shortcut = ui.Shortcut("O")
		}
		rows = append(rows, r)
	}
	if mode == openRemote && !hintSeen {
		rows = append(rows, ui.MenuRow{Label: "Opens over SSH. Needs your key on " + machine, Disabled: true})
	}
	return rows
}

// gitNotes are the Git menu's explanations: a detached HEAD, a clean
// branch behind its upstream, or the status's error.
func gitNotes(st *t3.VcsStatus, err string) []ui.MenuRow {
	var out []ui.MenuRow
	if st != nil && st.RefName == "" {
		out = append(out, ui.MenuRow{Kind: ui.MenuNote, Tone: "warning", Label: "Detached HEAD: create and check out a branch to enable push and " + termsFor(st).singular + " actions."})
	}
	if st != nil && st.RefName != "" && !st.HasWorkingTreeChanges && st.BehindCount > 0 && st.AheadCount == 0 {
		out = append(out, ui.MenuRow{Kind: ui.MenuNote, Tone: "warning", Label: "Behind upstream. Pull/rebase first."})
	}
	if err != "" {
		out = append(out, ui.MenuRow{Kind: ui.MenuNote, Tone: "destructive", Label: err})
	}
	return out
}

// gitProgress is a running action as the row shows it
// (resolveGitActionProgressPresentation).
func gitProgress(r *gitRun, now time.Time) *ui.GitProgress {
	p := &ui.GitProgress{Status: strings.TrimSpace(r.label)}
	switch {
	case r.pull && p.Status == "":
		p.Status = "Pulling latest changes..."
	case p.Status == "" || p.Status == "Running source control action":
		p.Status = "Starting source control action..."
	}
	start := r.phase
	if !r.pull {
		p.Output = strings.TrimSpace(r.output)
		if !r.hook.IsZero() {
			start = r.hook
		}
	}
	if !start.IsZero() {
		p.Elapsed = gitElapsed(now.Sub(start))
	}
	return p
}

// refPickerScene is the open branch picker: the first page of refs that
// match its search, each with its first tag, and a ref to create when
// none matches exactly.
func (a *App) refPickerScene(v *vcsState) *ui.BranchPicker {
	p := &ui.BranchPicker{Highlight: a.branchPicker.hi}
	if v.refs == nil {
		p.Status = "Loading refs..."
		return p
	}
	q := strings.ToLower(strings.TrimSpace(a.branchPicker.query))
	exact := false
	for _, r := range v.refs.Refs {
		if q != "" && !strings.Contains(strings.ToLower(r.Name), q) {
			continue
		}
		exact = exact || strings.EqualFold(r.Name, q)
		p.Refs = append(p.Refs, ui.BranchRef{Name: r.Name, Tag: refTag(r, v.cwd)})
	}
	if q != "" && !exact {
		p.Create = `Create new ref "` + strings.TrimSpace(a.branchPicker.query) + `"`
	}
	if n := len(v.refs.Refs); v.refs.TotalCount > n && q == "" {
		p.Status = "Showing " + len3(n) + " of " + len3(v.refs.TotalCount) + " refs"
	}
	return p
}

// refTag is the first of current, worktree, remote and default that fits r.
func refTag(r t3.VcsRef, cwd string) string {
	switch {
	case r.Current:
		return "current"
	case r.WorktreePath != "" && r.WorktreePath != cwd:
		return "worktree"
	case r.IsRemote:
		return "remote"
	case r.IsDefault:
		return "default"
	}
	return ""
}
