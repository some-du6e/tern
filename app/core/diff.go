package core

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/diff"
	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// T3's Diff surface for the open thread (DiffPanel.tsx, diffPanelStore.ts;
// docs/ui/13-diff.md §5): the scope it shows, the patch it fetched for
// it, how its files are folded; and the changed files cards under the
// replies whose runs changed files.

// diffSel is a thread's selection (DiffPanelSelection), kept across
// restarts as T3 keeps it in t3code:diff-panel-state. BaseRef is the
// branch's base, "" for Automatic; scopes other than Changes and a commit
// keep it too, so Changes returns to it.
type diffSel struct {
	Kind    string `json:"kind"` // branch, unstaged, commit, turn
	BaseRef string `json:"baseRef,omitempty"`
	SHA     string `json:"sha,omitempty"`
	Run     string `json:"turnId,omitempty"`
	File    string `json:"filePath,omitempty"` // a turn's file to reveal
}

// diffState is one thread's Diff surface.
type diffState struct {
	open, active bool // its tab is open, and is the one the panel shows
	sel          diffSel
	menu, picker bool
	reveal       int // counts reveal requests, so the same file can reveal again
	revealed     string
	// The patch shown is for key; fetching is the key on its way.
	key, fetching string
	fetchedAt     time.Time
	files         []diff.File
	raw           *ui.DiffRaw
	empty         bool // the patch was empty
	err, access   string
	notRepo       bool
	source        *t3.DiffSource
	refreshing    bool
	// The Commits submenu's list, and its note: why there are none, or
	// that only the latest came.
	commits     []t3.BranchCommit
	commitsNote string
	// The base ref picker's refs, and what's typed in its search.
	refs     []ui.DiffBaseRef
	refQuery string
	// Each section's folds over the default, and the tree's closed
	// folders.
	folds      map[string]map[string]bool
	treeClosed map[string]bool
	// collapses counts collapse-all presses: each starts the code view at
	// the top. fetches counts the patches fetched.
	collapses, fetches int
}

// diffShared is what every thread's Diff surface shares, and the changed
// files cards' folders.
type diffShared struct {
	threads    map[string]*diffState
	saved      map[string]diffSel // the selections kept, by thread ref
	tree       bool               // the file tree pane is open (t3code.diffFileTreeOpen)
	hideTests  bool               // t3code.diffHideTestFiles
	split      bool               // the diffLayout setting is split
	wrap       bool               // seeded from the wordWrap setting: on
	whitespace bool               // whitespace changes are hidden, the default
	cards      map[string]bool    // a card's folders all open, by thread ref and run
	folders    map[string]bool    // a card's folder flipped from that, by card and path
	// saves are store writes waiting for the lock to go.
	saves []func()
}

// noAccess is T3's message for a connection without filesystem:read.
const noAccess = "This connection cannot read local diffs."

// diffs is the shared Diff state, read from the store on first use. Call
// with the lock held.
func (a *App) diffs() *diffShared {
	if a.diff != nil {
		return a.diff
	}
	d := &diffShared{threads: map[string]*diffState{}, saved: map[string]diffSel{}, wrap: true, whitespace: true,
		cards: map[string]bool{}, folders: map[string]bool{}}
	_, _ = a.store.Get("diff-panel", &d.saved)
	_, _ = a.store.Get("diff-tree", &d.tree)
	_, _ = a.store.Get("diff-hide-tests", &d.hideTests)
	_, _ = a.store.Get("diff-split", &d.split)
	_, _ = a.store.Get("changed-files", &d.cards)
	a.diff = d
	return d
}

// diffFor is the Diff state of thread ref, made on first use with its
// kept selection, else T3's default: Changes against the automatic base.
func (a *App) diffFor(ref string) *diffState {
	d := a.diffs()
	ds := d.threads[ref]
	if ds == nil {
		ds = &diffState{sel: diffSel{Kind: "branch"}, folds: map[string]map[string]bool{}, treeClosed: map[string]bool{}}
		if s, ok := d.saved[ref]; ok && s.Kind != "" {
			ds.sel = s
		}
		d.threads[ref] = ds
	}
	return ds
}

// putJSON keeps v in the store under name, as it is now, once the lock is
// released (diffAction writes it). Call with the lock held.
func (a *App) putJSON(name string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	a.diffs().saves = append(a.diffs().saves, func() { _ = a.store.Put(name, json.RawMessage(raw)) })
}

// turnSummary is a checkpoint the Turn menu lists: a run and its "Turn N".
type turnSummary struct {
	run string
	n   int
	at  time.Time
	cp  *t3.Checkpoint
}

// turns are the thread's checkpointed runs, newest first: by turn number,
// then by when they finished (DiffPanel.tsx:306-312).
func turns(v *threadView) []turnSummary {
	var out []turnSummary
	for i := range v.checkpoints {
		cp := &v.checkpoints[i]
		if cp.RunID == "" || cp.AppRunOrdinal == 0 {
			continue
		}
		out = append(out, turnSummary{run: cp.RunID, n: cp.AppRunOrdinal, at: parseTime(cp.CapturedAt), cp: cp})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].at.After(out[j].at)
	})
	return out
}

// putCheckpoint merges a checkpoint.captured event in.
func (v *threadView) putCheckpoint(raw json.RawMessage) {
	var cp t3.Checkpoint
	if json.Unmarshal(raw, &cp) != nil || cp.ID == "" {
		return
	}
	for i := range v.checkpoints {
		if v.checkpoints[i].ID == cp.ID {
			v.checkpoints[i] = cp
			return
		}
	}
	v.checkpoints = append(v.checkpoints, cp)
}

// selectedTurn is the turn a turn selection shows: its run's, else the
// newest while that one is missing (DiffPanel.tsx:315-321).
func selectedTurn(sel diffSel, ts []turnSummary) *turnSummary {
	if sel.Kind != "turn" || len(ts) == 0 {
		return nil
	}
	for i := range ts {
		if ts[i].run == sel.Run {
			return &ts[i]
		}
	}
	return &ts[0]
}

// section is the selection's section id, which folds hang off.
func section(sel diffSel, turn *turnSummary) string {
	switch {
	case sel.Kind == "turn" && turn != nil:
		return "turn:" + turn.run
	case sel.Kind == "commit":
		return "commit:" + sel.SHA
	case sel.Kind == "unstaged":
		return "unstaged"
	}
	return "branch"
}

// wantKey names the patch a selection needs, "" for none yet.
func (a *App) wantKey(sel diffSel, turn *turnSummary) string {
	ws := strconv.FormatBool(a.diffs().whitespace)
	switch sel.Kind {
	case "turn":
		if turn == nil {
			return ""
		}
		return "turn:" + turn.run + ":" + strconv.Itoa(turn.n) + ":" + ws
	case "commit":
		return "commit:" + sel.SHA + ":" + sel.BaseRef + ":" + ws
	case "unstaged":
		return "unstaged:" + ws
	}
	return "branch:" + sel.BaseRef + ":" + ws
}

// cwd is where a thread's Git diffs come from: its worktree, else its
// project's folder.
func cwd(e *envState, t *t3.Thread) string {
	if t.WorktreePath != "" {
		return t.WorktreePath
	}
	return e.projects[t.ProjectID].WorkspaceRoot
}

// gitKinds are the preview sources T3's Git scopes read.
var gitKinds = map[string]string{"branch": "branch-range", "unstaged": "working-tree", "commit": "commit"}

// fetchDiffLocked starts fetching the patch for ds's selection unless it's
// shown (or force) or on its way. Call with the lock held.
func (a *App) fetchDiffLocked(e *envState, t *t3.Thread, ds *diffState, ts []turnSummary, force bool) {
	turn := selectedTurn(ds.sel, ts)
	key := a.wantKey(ds.sel, turn)
	if key == "" || e == nil || e.conn == nil || ds.fetching == key || ds.key == key && !force {
		return
	}
	ds.fetching = key
	conn, sel, ws, threadID, dir := e.conn, ds.sel, a.diffs().whitespace, t.ID, cwd(e, t)
	n := 0
	if turn != nil {
		n = turn.n
	}
	go func() {
		var patch string
		var src *t3.DiffSource
		var err error
		notRepo := false
		switch {
		case sel.Kind == "turn" && n == 1:
			// The first turn is the thread from its start (queries.ts:425-454).
			patch, err = conn.FullThreadDiff(threadID, 1, ws)
		case sel.Kind == "turn":
			patch, err = conn.TurnDiff(threadID, n-1, n, ws)
		default:
			var p t3.DiffPreview
			p, err = conn.DiffPreview(dir, sel.BaseRef, sel.SHA, gitKinds[sel.Kind], ws)
			notRepo = err == nil && len(p.Sources) == 0
			for i := range p.Sources {
				if p.Sources[i].Kind == gitKinds[sel.Kind] {
					src = &p.Sources[i]
					patch = src.Diff
				}
			}
		}
		a.mu.Lock()
		defer a.changed()
		defer a.mu.Unlock()
		if ds.fetching != key {
			return
		}
		ds.fetching, ds.key, ds.fetchedAt, ds.refreshing = "", key, time.Now(), false
		ds.fetches++
		ds.files, ds.raw, ds.err, ds.access, ds.source, ds.notRepo, ds.empty = nil, nil, "", "", src, notRepo, false
		var rpc *t3.RPCError
		switch {
		case errors.As(err, &rpc) && rpc.Tag == "EnvironmentAuthorizationError":
			ds.access = noAccess
		case err != nil:
			ds.err = err.Error()
		case strings.TrimSpace(patch) == "":
			ds.empty = true
		default:
			files, perr := diff.Parse(patch)
			switch {
			case errors.Is(perr, diff.ErrNoFiles):
				ds.raw = &ui.DiffRaw{Reason: "Unsupported diff format. Showing raw patch.", Text: patch}
			case perr != nil:
				ds.raw = &ui.DiffRaw{Reason: "Failed to parse patch. Showing raw patch.", Text: patch}
			default:
				sort.SliceStable(files, func(i, j int) bool { return diff.Compare(files[i].Path(), files[j].Path()) < 0 })
				ds.files = files
			}
		}
	}()
}

// fetchCommitsLocked lists the branch's commits for the Commits submenu.
// The menu shows the list it had while the new one loads.
func (a *App) fetchCommitsLocked(e *envState, t *t3.Thread, ds *diffState) {
	if e == nil || e.conn == nil {
		return
	}
	if ds.commits == nil {
		ds.commitsNote = "Loading commits..."
	}
	conn, dir, base := e.conn, cwd(e, t), ds.sel.BaseRef
	go func() {
		commits, more, err := conn.BranchCommits(dir, base)
		a.mu.Lock()
		defer a.changed()
		defer a.mu.Unlock()
		var rpc *t3.RPCError
		switch {
		case errors.As(err, &rpc) && rpc.Tag == "EnvironmentAuthorizationError":
			ds.commits, ds.commitsNote = nil, noAccess
		case err != nil:
			ds.commits, ds.commitsNote = nil, err.Error()
		case len(commits) == 0:
			ds.commits, ds.commitsNote = []t3.BranchCommit{}, "No commits on this branch."
		default:
			ds.commits, ds.commitsNote = commits, ""
			if more {
				ds.commitsNote = "Showing latest " + strconv.Itoa(len(commits)) + " commits"
			}
		}
	}()
}

// fetchRefsLocked lists the refs the branch base can be, local and remote
// (vcs.listRefs, as DiffPanel.tsx:556-588 asks), the head left out, the
// search's query passed on.
func (a *App) fetchRefsLocked(e *envState, t *t3.Thread, ds *diffState) {
	if e == nil || e.conn == nil {
		return
	}
	conn, dir, head, query := e.conn, cwd(e, t), compareRow(ds, t).Head, strings.TrimSpace(ds.refQuery)
	go func() {
		list := func(kind string) []vcsRef {
			in := map[string]any{"cwd": dir, "refKind": kind, "includeMatchingRemoteRefs": true, "limit": 100}
			if query != "" {
				in["query"] = query
			}
			var out struct {
				Refs []vcsRef `json:"refs"`
			}
			_ = conn.Call("vcs.listRefs", in, &out)
			return out.Refs
		}
		var local []vcsRef
		for _, r := range list("local") {
			if r.Name != head {
				local = append(local, r)
			}
		}
		refs := pairRefs(local, list("remote"))
		a.mu.Lock()
		ds.refs = refs
		a.mu.Unlock()
		a.changed()
	}()
}

// vcsRef is a ref as vcs.listRefs gives it.
type vcsRef struct {
	Name       string `json:"name"`
	RemoteName string `json:"remoteName"`
}

// pairRefs makes the picker's choices (baseRefChoices.ts): each local
// branch with its remote twin, origin's first, then the remote branches
// left alone.
func pairRefs(local, remote []vcsRef) []ui.DiffBaseRef {
	used := map[int]bool{}
	branch := func(r vcsRef) string {
		if r.RemoteName != "" && strings.HasPrefix(r.Name, r.RemoteName+"/") {
			return r.Name[len(r.RemoteName)+1:]
		}
		return r.Name
	}
	var out []ui.DiffBaseRef
	for _, l := range local {
		pick := -1
		for i, r := range remote {
			if used[i] || branch(r) != l.Name {
				continue
			}
			if pick < 0 || r.RemoteName == "origin" && remote[pick].RemoteName != "origin" {
				pick = i
			}
		}
		c := ui.DiffBaseRef{Label: l.Name, Local: l.Name}
		if pick >= 0 {
			used[pick], c.Remote = true, remote[pick].Name
		}
		out = append(out, c)
	}
	for i, r := range remote {
		if !used[i] {
			out = append(out, ui.DiffBaseRef{Label: r.Name, Remote: r.Name})
		}
	}
	return out
}

// shownRefs are the refs matching the search, by label or either name.
func shownRefs(ds *diffState) []ui.DiffBaseRef {
	q := strings.ToLower(strings.TrimSpace(ds.refQuery))
	if q == "" {
		return ds.refs
	}
	var out []ui.DiffBaseRef
	for _, r := range ds.refs {
		if strings.Contains(strings.ToLower(r.Label), q) || r.Local != "" && strings.Contains(strings.ToLower(r.Local), q) ||
			r.Remote != "" && strings.Contains(strings.ToLower(r.Remote), q) {
			out = append(out, r)
		}
	}
	return out
}

// diffRefSearch is a change in the base ref picker's search: the refs
// are read again for it.
func (a *App) diffRefSearch(query string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, t := a.thread(a.openID)
	ds := a.diffs().threads[a.openID]
	if t == nil || ds == nil || !ds.picker {
		return
	}
	ds.refQuery = query
	a.fetchRefsLocked(e, t, ds)
}

// diffPanelLocked is the open thread's Diff surface for the scene, nil
// while its tab is closed. It starts fetching what its scope needs. Call
// with the lock held.
func (a *App) diffPanelLocked(e *envState, t *t3.Thread, v *threadView) *ui.DiffPanel {
	sh := a.diffs()
	ds := sh.threads[a.openID]
	if ds == nil || !ds.open || v == nil {
		return nil
	}
	ts := turns(v)
	turn := selectedTurn(ds.sel, ts)
	p := &ui.DiffPanel{Active: ds.active, Key: a.openID + "\x00" + section(ds.sel, turn) + "\x00" + strconv.Itoa(ds.collapses), Rev: ds.fetches,
		Split: sh.split, Wrap: sh.wrap, Whitespace: sh.whitespace, HideTests: sh.hideTests, TreeOpen: sh.tree,
		Refresh: ds.sel.Kind != "turn", Refreshing: ds.refreshing, Revealed: ds.revealed, RevealID: ds.reveal}
	scopeLabel(p, ds, turn, ts)
	if ds.sel.Kind == "branch" && ds.access == "" && ds.key != "" && !ds.notRepo {
		p.Compare = compareRow(ds, t)
	}
	if ds.menu {
		m := &ui.DiffScopeMenu{Note: ds.commitsNote}
		if ds.sel.Kind == "commit" {
			m.Commit = ds.sel.SHA
		}
		if turn != nil {
			m.Turn = "turn:" + turn.run
		}
		for _, tr := range ts {
			m.Turns = append(m.Turns, ui.DiffTurn{Value: "turn:" + tr.run, Label: "Turn " + strconv.Itoa(tr.n), Time: tr.at.Local().Format("3:04 PM")})
		}
		for _, c := range ds.commits {
			m.Commits = append(m.Commits, ui.DiffCommit{SHA: c.SHA, Subject: c.Subject, When: relativeTime(parseTime(c.CommittedAt))})
		}
		p.Menu = m
	}
	a.fetchDiffLocked(e, t, ds, ts, false)
	want := a.wantKey(ds.sel, turn)
	switch {
	case ds.sel.Kind == "turn" && len(ts) == 0:
		p.Empty = "No completed turns yet."
		return p
	case ds.notRepo && ds.key == want:
		p.Empty = "Turn diffs are unavailable because this project is not a git repository."
		return p
	case ds.access != "" && ds.key == want:
		p.Empty = ds.access
		return p
	case ds.key != want && !ds.refreshing:
		p.Loading = map[string]string{"turn": "Loading checkpoint diff...", "unstaged": "Loading uncommitted changes...",
			"commit": "Loading commit...", "branch": "Loading changes..."}[ds.sel.Kind]
		return p
	}
	p.Error, p.Raw = ds.err, ds.raw
	switch {
	case ds.empty:
		p.NoPatch = "No net changes in this selection."
	case ds.err == "" && ds.raw == nil && len(ds.files) == 0:
		p.NoPatch = "No patch available for this selection."
	}
	if src := ds.source; src != nil && src.Truncated {
		n := "This preview exceeds the size limit. Changes shown are incomplete."
		if len(src.Files) > 0 {
			n += " Totals include all changes."
		}
		p.Notices = append(p.Notices, n)
	}
	diffFiles(p, ds, section(ds.sel, turn), sh.hideTests)
	return p
}

// scopeLabel names the selection on the scope trigger and checks its row
// in the menu (§5.2).
func scopeLabel(p *ui.DiffPanel, ds *diffState, turn *turnSummary, ts []turnSummary) {
	switch ds.sel.Kind {
	case "turn":
		p.Scope, p.Value = "Turn ?", "turn:"+ds.sel.Run
		if turn != nil {
			p.Scope, p.Value = "Turn "+strconv.Itoa(turn.n), "turn:"+turn.run
			if turn.run == ts[0].run {
				p.Scope, p.Value = "Latest turn", "latest"
			}
		}
	case "unstaged":
		p.Scope, p.Value = "Uncommitted", "unstaged"
	case "commit":
		p.Scope, p.Value, p.SHA = "Commit", "commit:"+ds.sel.SHA, shortSHA(ds.sel.SHA)
		if src := ds.source; src != nil && src.HeadRef == ds.sel.SHA && src.CommitSubject != "" {
			p.Scope = src.CommitSubject
		}
	default:
		p.Scope, p.Value = "Changes", "branch"
	}
}

// compareRow is the branch scope's "head → base": the source's refs, else
// the thread's branch and the base picked, else HEAD and Automatic.
func compareRow(ds *diffState, t *t3.Thread) *ui.DiffCompare {
	c := &ui.DiffCompare{Head: "HEAD", Base: "Automatic"}
	if t.Branch != "" {
		c.Head = t.Branch
	}
	if ds.sel.BaseRef != "" {
		c.Base = ds.sel.BaseRef
	}
	if src := ds.source; src != nil {
		if src.HeadRef != "" {
			c.Head = src.HeadRef
		}
		if src.BaseRef != "" {
			c.Base = src.BaseRef
		}
	}
	if ds.picker {
		c.Picker = &ui.DiffBasePicker{Automatic: ds.sel.BaseRef == "", Selected: ds.sel.BaseRef, Refs: shownRefs(ds)}
	}
	return c
}

// diffFiles fills in the code view's files: test files hidden on request
// (a revealed one stays), each folded by the section's folds over the
// default, collapsed (diffFilesCollapsed); and the totals and the tree.
func diffFiles(p *ui.DiffPanel, ds *diffState, sec string, hideTests bool) {
	folds := ds.folds[sec]
	var paths []string
	types := map[string]diff.ChangeType{}
	stat := ui.DiffStat{}
	for _, f := range ds.files {
		if diff.IsTest(f.Path()) {
			p.TestFiles++
			if hideTests && f.Path() != ds.revealed {
				continue
			}
		}
		collapsed, ok := folds[f.Path()]
		p.Files = append(p.Files, ui.DiffFile{File: f, Collapsed: collapsed || !ok})
		paths = append(paths, f.Path())
		types[f.Path()] = f.Type
		stat.Additions += f.Additions
		stat.Deletions += f.Deletions
	}
	if len(p.Files) == 0 {
		return
	}
	// A Git source's own counts cover the files it couldn't send too.
	if src := ds.source; src != nil && len(src.Files) > 0 {
		stat = ui.DiffStat{}
		for _, f := range src.Files {
			stat.Additions += f.Additions
			stat.Deletions += f.Deletions
		}
	}
	p.Stat = &stat
	p.Tree = diff.FileTree(paths, types, ds.treeClosed)
}

func shortSHA(sha string) string { return sha[:min(7, len(sha))] }

// relativeTime is T3's commit age: "just now", "4m ago", "3h ago", "2d
// ago" (timestampFormat.ts:262-285).
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := max(0, time.Since(t))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h ago"
	}
	return strconv.Itoa(int(d.Hours()/24)) + "d ago"
}

// openDiffLocked opens the Diff tab and shows it with its selection, and
// rereads a Git scope read over 10s ago (DiffPanel.tsx:626-639).
func (a *App) openDiffLocked(ds *diffState) {
	ds.open, ds.active = true, true
	if ds.sel.Kind != "turn" && time.Since(ds.fetchedAt) > 10*time.Second {
		ds.key = ""
	}
}

// selectLocked changes the open thread's selection, closing the menus,
// and keeps it. The branch base carries over, so Changes returns to it.
func (a *App) selectLocked(ds *diffState, sel diffSel) {
	if sel.Kind != "branch" && sel.Kind != "commit" {
		sel.BaseRef = ds.sel.BaseRef
	}
	ds.sel = sel
	ds.menu, ds.picker = false, false
	sh := a.diffs()
	sh.saved[a.openID] = sel
	a.putJSON("diff-panel", sh.saved)
}

// selectTurnLocked shows run's turn, revealing file in it: a card's row,
// or its first file.
func (a *App) selectTurnLocked(ds *diffState, run, file string) {
	a.selectLocked(ds, diffSel{Kind: "turn", Run: run, File: file})
	if file != "" {
		sec := "turn:" + run
		if ds.folds[sec] == nil {
			ds.folds[sec] = map[string]bool{}
		}
		ds.folds[sec][file] = false
	}
	revealLocked(ds, file)
}

// revealLocked scrolls the code view to file and selects it in the tree,
// opening the folders it's in (§5.9).
func revealLocked(ds *diffState, file string) {
	ds.reveal++
	ds.revealed = file
	for i := range file {
		if file[i] == '/' {
			delete(ds.treeClosed, file[:i+1])
		}
	}
}

// closeDiffMenusLocked closes the open thread's Diff menus.
func (a *App) closeDiffMenusLocked() {
	if ds := a.diffs().threads[a.openID]; ds != nil {
		ds.menu, ds.picker = false, false
	}
}

// hideDiffLocked closes the open thread's Diff tab, reporting whether it
// was open.
func (a *App) hideDiffLocked() bool {
	ds := a.diffs().threads[a.openID]
	if ds == nil || !ds.open {
		return false
	}
	ds.open, ds.active, ds.menu, ds.picker = false, false, false, false
	return true
}

// diffAction handles the Diff surface's and the cards' click targets; it
// reports whether id was one. Call without the lock.
func (a *App) diffAction(id string) bool {
	verb, arg, item := strings.Cut(id, ":")
	if !strings.HasPrefix(verb, "diff.") && !strings.HasPrefix(verb, "files.") {
		return false
	}
	a.mu.Lock()
	defer func() {
		saves := a.diffs().saves
		a.diffs().saves = nil
		a.mu.Unlock()
		for _, save := range saves {
			save()
		}
	}()
	e, t := a.thread(a.openID)
	if t == nil {
		return true
	}
	sh := a.diffs()
	ds := a.diffFor(a.openID)
	var ts []turnSummary
	if a.view != nil {
		ts = turns(a.view)
	}
	sec := section(ds.sel, selectedTurn(ds.sel, ts))
	folded := func(path string) bool {
		c, ok := ds.folds[sec][path]
		return c || !ok
	}
	fold := func(path string, collapsed bool) {
		if ds.folds[sec] == nil {
			ds.folds[sec] = map[string]bool{}
		}
		ds.folds[sec][path] = collapsed
	}
	switch verb {
	case "diff.toggle":
		// mod+D hides the surface while it shows, else shows it.
		if ds.open && ds.active {
			a.hideDiffLocked()
			break
		}
		a.openDiffLocked(ds)
	case "diff.tab":
		ds.active = true
	case "diff.close":
		a.hideDiffLocked()
	case "diff.scope":
		switch {
		case !item:
			ds.menu, ds.picker = !ds.menu, false
			if ds.menu {
				a.fetchCommitsLocked(e, t, ds)
			}
		case arg == "branch":
			a.selectLocked(ds, diffSel{Kind: "branch", BaseRef: ds.sel.BaseRef})
		case arg == "unstaged":
			a.selectLocked(ds, diffSel{Kind: "unstaged"})
		case arg == "latest" && len(ts) > 0:
			a.selectTurnLocked(ds, ts[0].run, "")
		case strings.HasPrefix(arg, "turn:"):
			a.selectTurnLocked(ds, strings.TrimPrefix(arg, "turn:"), "")
		}
	case "diff.commit":
		// Choosing the shown commit again goes back to Changes.
		if ds.sel.Kind == "commit" && ds.sel.SHA == arg {
			a.selectLocked(ds, diffSel{Kind: "branch", BaseRef: ds.sel.BaseRef})
		} else {
			a.selectLocked(ds, diffSel{Kind: "commit", SHA: arg, BaseRef: ds.sel.BaseRef})
		}
	case "diff.base":
		if !item {
			ds.picker, ds.menu = !ds.picker, false
			if ds.picker {
				ds.refQuery = ""
				a.fetchRefsLocked(e, t, ds)
			}
			break
		}
		a.selectLocked(ds, diffSel{Kind: "branch", BaseRef: arg})
	case "diff.baseswitch":
		// A ref's switch picks its remote copy, or back its local one.
		refs := shownRefs(ds)
		if i, err := strconv.Atoi(arg); err == nil && i >= 0 && i < len(refs) {
			ref := refs[i].Remote
			if ds.sel.BaseRef == ref {
				ref = refs[i].Local
			}
			a.selectLocked(ds, diffSel{Kind: "branch", BaseRef: ref})
		}
	case "diff.refresh":
		ds.refreshing = true
		a.fetchDiffLocked(e, t, ds, ts, true)
	case "diff.collapse":
		// Every file folded unfolds them all; otherwise they all fold.
		all := true
		for _, f := range ds.files {
			all = all && folded(f.Path())
		}
		for _, f := range ds.files {
			fold(f.Path(), !all)
		}
		ds.collapses++
	case "diff.layout":
		sh.split = arg == "split"
		a.putJSON("diff-split", sh.split)
	case "diff.wrap":
		sh.wrap = !sh.wrap
	case "diff.whitespace":
		sh.whitespace = !sh.whitespace
	case "diff.tests":
		sh.hideTests = !sh.hideTests
		a.putJSON("diff-hide-tests", sh.hideTests)
	case "diff.tree":
		sh.tree = !sh.tree
		a.putJSON("diff-tree", sh.tree)
	case "diff.file", "diff.fold":
		fold(arg, !folded(arg))
	case "diff.folder":
		ds.treeClosed[arg] = !ds.treeClosed[arg]
	case "diff.folders":
		// Any folder open closes them all; otherwise they all open.
		var paths []string
		for _, f := range ds.files {
			paths = append(paths, f.Path())
		}
		open := false
		for _, en := range diff.FileTree(paths, nil, ds.treeClosed) {
			open = open || en.Dir && en.Open
		}
		for _, en := range diff.FileTree(paths, nil, nil) {
			if en.Dir {
				ds.treeClosed[en.Path] = open
			}
		}
	case "diff.treefile":
		fold(arg, false)
		revealLocked(ds, arg)
	case "files.open", "files.file":
		run, path, _ := strings.Cut(arg, ":")
		if verb == "files.open" {
			path = ""
			for _, tr := range ts {
				if tr.run == run && len(tr.cp.Files) > 0 {
					path = tr.cp.Files[0].Path
				}
			}
		}
		a.selectTurnLocked(ds, run, path)
		a.openDiffLocked(ds)
	case "files.all":
		k := a.openID + "\x00" + arg
		sh.cards[k] = !sh.cards[k]
		for f := range sh.folders {
			if strings.HasPrefix(f, k+"\x00") {
				delete(sh.folders, f)
			}
		}
		a.putJSON("changed-files", sh.cards)
	case "files.folder":
		run, path, _ := strings.Cut(arg, ":")
		k := a.openID + "\x00" + run + "\x00" + path
		if _, ok := sh.folders[k]; ok {
			delete(sh.folders, k)
		} else {
			sh.folders[k] = !sh.cards[a.openID+"\x00"+run]
		}
	}
	return true
}

// changedFilesLocked is the changed files card of each reply that ends a
// run with changed files, by row key (TimelineAssistantRows.tsx:366-390,
// threadCheckpoints.ts): the run's last reply carries it. Call with the
// lock held.
func (a *App) changedFilesLocked(v *threadView, list []*t3.TurnItem) map[string]*ui.ChangedFiles {
	last := map[string]string{}
	for _, it := range list {
		if it.Type == "assistant_message" && it.RunID != "" {
			last[it.RunID] = rowKey(it)
		}
	}
	sh := a.diffs()
	out := map[string]*ui.ChangedFiles{}
	for _, tr := range turns(v) {
		k, ok := last[tr.run]
		if !ok || len(tr.cp.Files) == 0 {
			continue
		}
		c := &ui.ChangedFiles{Run: tr.run, Count: len(tr.cp.Files), Open: map[string]bool{}}
		var stats []diff.FileStat
		for _, f := range tr.cp.Files {
			stats = append(stats, diff.FileStat{Path: f.Path, Additions: f.Additions, Deletions: f.Deletions})
			c.Additions += f.Additions
			c.Deletions += f.Deletions
		}
		if tot := tr.cp.FilesTotal; tot != nil {
			c.Count, c.Additions, c.Deletions = tot.Count, tot.Additions, tot.Deletions
		}
		c.Nodes = diff.CardTree(stats)
		card := a.openID + "\x00" + tr.run
		c.AllOpen = sh.cards[card]
		for f, open := range sh.folders {
			if path, ok := strings.CutPrefix(f, card+"\x00"); ok {
				c.Open[path] = open
			}
		}
		out[k] = c
	}
	return out
}
