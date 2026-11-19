package core

import (
	"regexp"
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
	// switching is the ref a switch is moving the checkout to; the branch
	// row names it until the switch lands, as T3's optimistic label does.
	switching string
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

// commitAsk is the Git menu's Commit in its dialog: the files left out of
// the commit, by path, and whether they're being picked.
type commitAsk struct {
	excluded map[string]bool
	editing  bool
}

// gitCommit is what the Commit dialog gives a commit: its message ("" to
// have one generated) and its files, nil for them all.
type gitCommit struct {
	message string
	files   []string
}

// refPicker is the open branch picker: its search and highlight, and the
// refs the server found for the search (vcs.listRefs with the query), a
// page at a time. Each search gets a new generation, so a page for an
// older one is dropped when it lands.
type refPicker struct {
	query   string
	hi      int
	found   *t3.VcsRefs // the pages so far; nil until the first lands
	loading bool
	gen     int
}

func vcsKey(envID, cwd string) string { return envID + "\x00" + cwd }

// gitCwd is where a thread's files and Git live: its worktree, else its
// project's root.
func gitCwd(worktree string, p t3.Project) string {
	if worktree != "" {
		return worktree
	}
	return p.WorkspaceRoot
}

// worktreeOf is the worktree thread t (nil for the draft) works in: its
// own, or the one the draft took over; "" for the project's folder. Call
// with the lock held.
func (a *App) worktreeOf(t *t3.Thread) string {
	if t != nil {
		return t.WorktreePath
	}
	return a.draftPath
}

// previousWorktree is the thread whose worktree the draft's run context
// menu offers to take over (resolvePreviousWorktreeSeed): of project p's
// threads in e not archived, the one updated last in a worktree other
// than the draft's own; nil for none. Call with the lock held.
func (a *App) previousWorktree(e *envState, p t3.Project) *t3.Thread {
	var best *t3.Thread
	for _, t := range e.threads {
		at := millis(t.UpdatedAt)
		if t.ProjectID != p.ID || t.WorktreePath == "" || t.WorktreePath == a.draftPath || t.ArchivedAt != "" || at == 0 {
			continue
		}
		if best == nil || at > millis(best.UpdatedAt) {
			best = t
		}
	}
	return best
}

// keybindings are the shortcuts the shell and the card take and show
// (primaryServerKeybindingsAtom): with no primary environment, the first
// connected one's rules after T3's defaults for the commands they leave
// alone; the defaults until one has its config. Call with the lock held.
func (a *App) keybindings() []ui.Keybinding {
	for _, e := range a.envs {
		if e.conn != nil && e.config != nil {
			return ui.MergeKeys(uiKeys(e.config.Keybindings))
		}
	}
	return ui.MergeKeys(nil)
}

// uiKeys is the server's rules as the ui takes them.
func uiKeys(ks []t3.Keybinding) []ui.Keybinding {
	out := make([]ui.Keybinding, 0, len(ks))
	for _, k := range ks {
		sh := k.Shortcut
		out = append(out, ui.Keybinding{Command: k.Command, Key: sh.Key, Mod: sh.Mod, Meta: sh.Meta, Ctrl: sh.Ctrl,
			Shift: sh.Shift, Alt: sh.Alt, When: uiWhen(k.When)})
	}
	return out
}

// uiWhen is a rule's condition as the ui takes it, not's operand on the left.
func uiWhen(w *t3.KeybindingWhen) *ui.When {
	if w == nil {
		return nil
	}
	if w.Type == "not" {
		return &ui.When{Op: w.Type, Left: uiWhen(w.Node)}
	}
	return &ui.When{Op: w.Type, Name: w.Name, Left: uiWhen(w.Left), Right: uiWhen(w.Right)}
}

// cardItems are the card's rows the "Show in thread details" menu turns
// on and off (threadDetailsItems.ts), in its order; all on by default.
// T3's Servers, Automations, Lineage, Side chats, Usage limits and Context
// window join them with their sections.
var cardItems = []struct{ id, label string }{
	{"environment", "Environment and worktree"}, {"openIn", "Open in editor"}, {"scripts", "Project scripts"},
	{"branch", "Branch and pull request"}, {"gitActions", "Git actions"}, {"changes", "Changes"},
}

// itemsMenu is the card's own menu: its items with their checks, then
// Reset to default, off while everything is at its default.
func (a *App) itemsMenu() []ui.MenuRow {
	rows := []ui.MenuRow{{Kind: ui.MenuGroupLabel, Label: "Show in thread details"}}
	for _, it := range cardItems {
		rows = append(rows, ui.MenuRow{Kind: ui.MenuCheckRow, Label: it.label, Checked: !a.itemsHidden[it.id]})
	}
	return append(rows, ui.MenuRow{Kind: ui.MenuSeparator}, ui.MenuRow{Label: "Reset to default", Disabled: len(a.itemsHidden) == 0})
}

// cardScene fills the details card for thread t (nil for the draft) in
// project p, and makes sure its checkout's status is streaming.
func (a *App) cardScene(s *ui.Scene, e *envState, t *t3.Thread, p t3.Project, hasProject bool, now time.Time) {
	d := &ui.ThreadDetails{}
	s.Card = d
	show := func(item string) bool { return !a.itemsHidden[item] }
	if a.cardMenu == "items" {
		d.Menu = &ui.CardMenu{For: "items", Rows: a.itemsMenu(), AtPointer: a.cardAtPointer}
	}
	if !hasProject {
		return
	}
	worktree := a.worktreeOf(t)
	cwd := gitCwd(worktree, p)
	v := a.watchVcsLocked(e, cwd)
	st := v.status
	isRepo := st == nil || st.IsRepo
	scratch := isScratch(p, e.config)
	offersGit := cwd != "" && e.config != nil && !scratch

	// The run context (resolveEffectiveEnvMode): a thread in a worktree is
	// in worktree mode; a draft is when it's to make a new one, and works in
	// a worktree it took over as its current one.
	worktreeMode := (t != nil && worktree != "") || (t == nil && a.draftWorktree)
	locked := t != nil
	label := workspaceLabel(worktree, worktreeMode, locked)
	icon := "folder"
	switch {
	case worktreeMode:
		icon = "folder-git-2"
	case worktree != "":
		icon = "folder-git"
	}
	path := a.workspacePath(t, p)
	tip := label
	if path != "" {
		tip = path
	}
	if show("environment") {
		d.Run = &ui.RunContext{Machine: machineKind(e.config), Label: e.env.Label, Tip: e.env.Label, Workspace: icon, WorkspaceTip: tip, Locked: locked}
	}
	if !locked && a.cardMenu == "run" {
		d.Menu = &ui.CardMenu{For: "run", Rows: runMenu(worktree, a.draftWorktree, a.previousWorktree(e, p))}
	}

	// Open in editor.
	mode, target := remoteOpen(e)
	choices := a.editorChoices(e, mode)
	pref := preferredEditor(choices, a.lastEditor)
	o := &ui.OpenIn{Editor: pref, Label: "Open in editor", Disabled: pref == "" || cwd == "" || mode == openNone}
	if pref != "" {
		o.Label = "Open in " + editorLabel(pref)
	}
	if show("openIn") {
		d.OpenIn = o
	}
	if a.cardMenu == "openin" {
		d.Menu = &ui.CardMenu{For: "openin", Rows: editorMenu(choices, pref, ui.KeyLabel(s.Keys, "editor.openFavorite"), mode, target, e.env.Label, a.hintSeen, cwd == "")}
	}

	// Project scripts.
	sc := &ui.Scripts{Any: len(p.Scripts) > 0}
	if ps := primaryScript(p.Scripts, a.lastScripts[p.ID]); ps != nil {
		sc.Primary, sc.Icon = ps.Name, ps.Icon
	}
	if show("scripts") {
		d.Scripts = sc
	}
	if a.cardMenu == "scripts" && sc.Any {
		// Each with its shortcut from its environment's own rules
		// (ProjectScriptsControl.tsx).
		var keys []ui.Keybinding
		if e.config != nil {
			keys = uiKeys(e.config.Keybindings)
		}
		var rows []ui.MenuRow
		for _, s := range p.Scripts {
			rows = append(rows, ui.MenuRow{Label: scriptLabel(s), Icon: ui.ScriptIcon(s.Icon), Shortcut: ui.KeyLabel(keys, "script."+s.ID+".run")})
		}
		// With no primary the split is the Actions form, whose menu doesn't
		// light rows on hover (highlightItemOnHover).
		d.Menu = &ui.CardMenu{For: "scripts", Rows: rows, NoHover: sc.Primary == ""}
	}

	// The pull requests go under the branch row, or without one where it
	// would be, with nothing from Git to fall back on. They're read while
	// the card can be seen: shown inline, or its popover open.
	if show("branch") && t != nil {
		visible := !a.cardHidden[a.openID] || a.detailsOpen
		d.PRs = a.prsScene(e, p, prItems(e, t, a.branchPR(e, t, p, v)), termsFor(st), now, visible)
	}
	if !offersGit {
		return
	}
	if isRepo && show("branch") {
		value := a.branchOf(t, v)
		if v.switching != "" {
			value = v.switching
		}
		b := &ui.BranchRow{Label: "Select ref", Disabled: v.refs == nil || v.switching != ""}
		switch {
		case value != "" && worktreeMode && worktree == "":
			b.Label = "From " + value
		case value != "":
			b.Label = value
		}
		d.Branch = b
		if a.branchPicker != nil {
			d.Picker = a.refPickerScene(v, t, p.WorkspaceRoot, t == nil && a.draftWorktree)
		}
	}

	if !isRepo {
		d.InitGit, d.Initing = show("gitActions"), v.initing
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
	if show("gitActions") {
		d.Git = g
	}
	if a.cardMenu == "git" {
		var rows []ui.MenuRow
		for _, it := range items {
			rows = append(rows, ui.MenuRow{Label: it.label, Icon: gitMenuIcon(it.id, providerKind(st)), Disabled: it.disabled,
				Reason: gitMenuReason(it, st, busy, hasRemote)})
		}
		d.Menu = &ui.CardMenu{For: "git", Rows: append(rows, notes...)}
	}
	// tern has no Diff surface to open yet, so the row can't open one.
	c := &ui.Changes{Insertions: ins, Deletions: del, Disabled: true}
	if known {
		c.Base = "Uncommitted"
		if base != "" {
			c.Base = "vs " + base
		}
	}
	if show("changes") {
		d.Changes = c
	}
}

// branchOf is the branch the row shows (resolveBranchToolbarValue): a new
// worktree's base (the default branch until one's picked), else the
// checkout's own branch, else the thread's. Call with the lock held.
func (a *App) branchOf(t *t3.Thread, v *vcsState) string {
	if t == nil && a.draftWorktree {
		if a.draftBase != "" {
			return a.draftBase
		}
		if v != nil && v.refs != nil {
			for _, r := range v.refs.Refs {
				if r.IsDefault && !r.IsRemote {
					return r.Name
				}
			}
		}
	}
	if v != nil && v.status != nil && v.status.RefName != "" {
		return v.status.RefName
	}
	if t != nil {
		return t.Branch
	}
	return ""
}

// workspacePath is where thread t works, its worktree or its project's
// root; "" for a draft's worktree that doesn't exist yet. Call with the
// lock held.
func (a *App) workspacePath(t *t3.Thread, p t3.Project) string {
	if t == nil && a.draftWorktree {
		return ""
	}
	return gitCwd(a.worktreeOf(t), p)
}

// runMenu is a draft's run context menu: its workspace, the current
// checkout (or the worktree it has) or a new worktree, and the previous
// worktree, prev's, with its branch under it, when there's one.
func runMenu(worktree string, newWorktree bool, prev *t3.Thread) []ui.MenuRow {
	current := ui.MenuRow{Kind: ui.MenuRadioRow, Label: workspaceLabel(worktree, false, false), Icon: "folder", Checked: !newWorktree}
	if worktree != "" {
		current.Icon = "folder-git"
	}
	rows := []ui.MenuRow{
		{Kind: ui.MenuGroupLabel, Label: "Workspace"},
		current,
		{Kind: ui.MenuRadioRow, Label: "New worktree", Icon: "folder-git-2", Checked: newWorktree},
	}
	if prev != nil {
		rows = append(rows, ui.MenuRow{Kind: ui.MenuRadioRow, Label: "Previous worktree", Icon: "history", Detail: prev.Branch})
	}
	return rows
}

// editorMenu lists the editors to make the preferred one, the preferred
// with Open in's shortcut (key), or says why there are none.
func editorMenu(choices []string, pref, key string, mode openMode, target t3.RemoteOpenTarget, machine string, hintSeen, noFolder bool) []ui.MenuRow {
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
		r := ui.MenuRow{Label: editorLabel(id), Icon: "editor:" + id, Disabled: noFolder}
		if id == pref {
			r.Shortcut = key
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

// commitScene is the Commit dialog over the checkout's status
// (GitActionsControl.tsx): its branch, "(detached HEAD)" without one, and
// its working tree's files, the ones left out marked, which open while the
// status says where they are. Call with the lock held.
func (a *App) commitScene(c *commitAsk) *ui.CommitDialog {
	d := &ui.CommitDialog{Branch: "(detached HEAD)", Editing: c.editing}
	_, _, _, v := a.checkout()
	if v == nil || v.status == nil {
		return d
	}
	st := v.status
	if st.RefName != "" {
		d.Branch = st.RefName
	}
	d.Default, d.Opens = st.IsDefaultRef, st.WorkingTree.RootPath != ""
	for _, f := range st.WorkingTree.Files {
		d.Files = append(d.Files, ui.CommitFile{Path: f.Path, Insertions: f.Insertions, Deletions: f.Deletions, Excluded: c.excluded[f.Path]})
	}
	return d
}

// pathUnder is path, relative as the status writes it, under folder dir on
// dir's machine (resolvePathLinkTarget): joined with dir's own separator,
// a backslash for a Windows path.
func pathUnder(dir, path string) string {
	sep := "/"
	if len(dir) > 2 && dir[1] == ':' && (dir[2] == '\\' || dir[2] == '/') || strings.HasPrefix(dir, `\\`) || strings.Contains(dir, `:\`) {
		sep = `\`
	}
	return strings.TrimRight(dir, `/\`) + sep + strings.TrimLeft(path, `/\`)
}

// pickerRefs are the refs the open picker lists from: the server's pages
// for its search, or for no search the checkout's first page, which the
// branch row loaded. Call with the lock held.
func (a *App) pickerRefs(v *vcsState) *t3.VcsRefs {
	if pk := a.branchPicker; pk.found != nil || newRefName(pk.query) != "" {
		return pk.found
	}
	return v.refs
}

// refPickerScene is thread t's (nil for the draft's) open branch picker
// (BranchToolbarBranchSelector.tsx): the refs found for its search, each
// with its first tag against the project's folder root and off when
// picking it would move the thread while its turn runs, which a notice
// says; then a ref to create when none is named that (offered while the
// refs load too), except while picking a new worktree's base; and how
// much is loaded.
func (a *App) refPickerScene(v *vcsState, t *t3.Thread, root string, choosingBase bool) *ui.BranchPicker {
	pk := a.branchPicker
	p := &ui.BranchPicker{Highlight: pk.hi}
	found := a.pickerRefs(v)
	q := strings.TrimSpace(pk.query)
	name := newRefName(q)
	exact := false
	if found != nil {
		for _, r := range found.Refs {
			exact = exact || r.Name == name
			blocked := !choosingBase && moveBlocked(t, root, r)
			if blocked {
				p.Notice = worktreeBlocked
			}
			if matchesRef(r.Name, q) {
				p.Refs = append(p.Refs, ui.BranchRef{Name: r.Name, Tag: refTag(r, root), Disabled: blocked})
			}
		}
	}
	if q != "" && !exact && !choosingBase {
		p.Create = `Create new ref "` + name + `"`
	}
	switch {
	case found == nil:
		p.Status, p.Loading = "Loading refs...", true
	case pk.loading:
		p.Status = "Loading more refs..."
	case found.NextCursor != nil:
		p.Status, p.More = "Showing "+len3(len(found.Refs))+" of "+len3(found.TotalCount)+" refs", true
	}
	return p
}

// newRefName is a search as a ref name (sanitizeNewRefName): trimmed, each
// run of spaces, tabs and line breaks a dash.
func newRefName(q string) string { return refSpaces.ReplaceAllString(strings.TrimSpace(q), "-") }

var refSpaces = regexp.MustCompile("[ \t\n\r\f\v]+")

// matchesRef is the picker's filter (shouldIncludeBranchPickerItem): the
// ref's name holds the search, ignoring case, as typed or as a ref name.
func matchesRef(name, q string) bool {
	q = strings.ToLower(q)
	lower := strings.ToLower(name)
	return q == "" || strings.Contains(lower, q) || strings.Contains(lower, newRefName(q))
}

// appendRefs adds a page to the ones before it, a ref named twice keeping
// its first place.
func appendRefs(prev *t3.VcsRefs, page *t3.VcsRefs) *t3.VcsRefs {
	if prev == nil {
		return page
	}
	out := *prev
	out.Refs = append([]t3.VcsRef(nil), prev.Refs...)
	at := map[string]int{}
	for i, r := range out.Refs {
		at[r.Name] = i
	}
	for _, r := range page.Refs {
		if i, ok := at[r.Name]; ok {
			out.Refs[i] = r
			continue
		}
		at[r.Name] = len(out.Refs)
		out.Refs = append(out.Refs, r)
	}
	out.NextCursor, out.TotalCount = page.NextCursor, max(prev.TotalCount, page.TotalCount)
	return &out
}

// refTag is the first of current, worktree, remote and default that fits
// r: "worktree" for a ref checked out somewhere other than the project's
// own folder.
func refTag(r t3.VcsRef, root string) string {
	switch {
	case r.Current:
		return "current"
	case r.WorktreePath != "" && root != "" && r.WorktreePath != root:
		return "worktree"
	case r.IsRemote:
		return "remote"
	case r.IsDefault:
		return "default"
	}
	return ""
}
