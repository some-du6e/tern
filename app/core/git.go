package core

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The details card's Git: each checkout's status stream and refs, and the
// actions its rows run through the server, as T3's do
// (docs/projects-files-vcs-terminal.md §3).

// watchVcsLocked is the status of e's checkout at cwd, starting its stream
// the first time and after a reconnect. Other checkouts' streams stop
// unless an action still runs in them. Call with the lock held.
func (a *App) watchVcsLocked(e *envState, cwd string) *vcsState {
	key := vcsKey(e.id(), cwd)
	for k, v := range a.vcs {
		if k != key && v.running == nil {
			if v.stop != nil {
				v.stop()
			}
			delete(a.vcs, k)
		}
	}
	v := a.vcs[key]
	if v == nil {
		v = &vcsState{cwd: cwd}
		a.vcs[key] = v
	}
	if conn := e.conn; conn != nil && v.conn != conn && cwd != "" {
		if v.stop != nil {
			v.stop()
		}
		v.conn = conn
		go a.subscribeVcs(conn, key, v)
	}
	return v
}

// subscribeVcs streams v's status and loads its first page of refs, which
// the branch row waits on.
func (a *App) subscribeVcs(conn *t3.Conn, key string, v *vcsState) {
	live := func() bool { return a.vcs[key] == v && v.conn == conn }
	stop := conn.SubscribeVcsStatus(v.cwd, func(ev t3.VcsStatusEvent) {
		a.mu.Lock()
		if live() {
			v.status, v.err = ev.Apply(v.status), ""
		}
		a.mu.Unlock()
		a.changed()
	}, func(err error) {
		a.mu.Lock()
		if live() && err != nil && !errors.Is(err, t3.ErrClosed) {
			v.err = err.Error()
		}
		a.mu.Unlock()
		a.changed()
	})
	a.mu.Lock()
	if live() {
		v.stop = stop
	} else {
		stop()
	}
	a.mu.Unlock()
	a.loadRefs(conn, key, v)
}

// loadRefs reads the checkout's first page of refs.
func (a *App) loadRefs(conn *t3.Conn, key string, v *vcsState) {
	var refs t3.VcsRefs
	err := conn.Call("vcs.listRefs", map[string]any{"cwd": v.cwd, "limit": 100}, &refs)
	a.mu.Lock()
	if a.vcs[key] == v && err == nil {
		v.refs = &refs
	}
	a.mu.Unlock()
	a.changed()
}

// checkout is the open thread's or draft's environment, thread (nil for
// the draft), project and Git state. Call with the lock held.
func (a *App) checkout() (*envState, *t3.Thread, t3.Project, *vcsState) {
	var e *envState
	var t *t3.Thread
	var p t3.Project
	switch {
	case a.draft != "":
		e = a.env(a.draftEnv)
		if e != nil {
			p = e.projects[a.draft]
		}
	case a.openID != "":
		e, t = a.thread(a.openID)
		if t != nil {
			p = e.projects[t.ProjectID]
		}
	}
	if e == nil {
		return nil, nil, p, nil
	}
	return e, t, p, a.vcs[vcsKey(e.id(), gitCwd(t, p))]
}

// refreshStatus has the server read the checkout's status again
// (vcs.refreshStatus), as the Git menu does when it opens; the branch
// picker asks for the local half only. The stream carries the result too.
func (a *App) refreshStatus(localOnly bool) {
	a.mu.Lock()
	e, _, _, v := a.checkout()
	if v == nil || e.conn == nil {
		a.mu.Unlock()
		return
	}
	conn := e.conn
	a.mu.Unlock()
	in := map[string]any{"cwd": v.cwd}
	if localOnly {
		in["localOnly"] = true
	}
	go func() {
		var st t3.VcsStatus
		if conn.Call("vcs.refreshStatus", in, &st) != nil {
			return
		}
		a.mu.Lock()
		if v.conn == conn {
			v.status = &st
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// draftStrategyLocked is where the draft's thread will work
// (orchestration.launchThread's workspaceStrategy): the project's checkout,
// or a new worktree from the ref the branch row names. Call with the lock
// held.
func (a *App) draftStrategyLocked() map[string]any {
	_, _, _, v := a.checkout()
	if a.draft == "" || !a.draftWorktree || v == nil {
		return map[string]any{"type": "root"}
	}
	base := a.branchOf(nil, v)
	if base == "" {
		return map[string]any{"type": "root"}
	}
	return map[string]any{"type": "worktree", "baseRef": base}
}

// gitQuick runs the Git row's quick action.
func (a *App) gitQuick() {
	a.mu.Lock()
	_, _, _, v := a.checkout()
	if v == nil {
		a.mu.Unlock()
		return
	}
	st := v.status
	q := resolveQuickAction(st, v.running != nil, st != nil && st.IsDefaultRef, st == nil || st.HasPrimaryRemote)
	a.mu.Unlock()
	switch q.kind {
	case "run_action":
		a.runGit(q.action, false, false)
	case "run_pull":
		a.pull()
	case "open_publish":
		a.failf("Publish repository: tern can't publish a repository yet; publish it in T3")
	}
	// A hint explains itself on hover; a click does nothing more.
}

// runGit runs a stacked action (commit, push, create_pr, commit_push or
// commit_push_pr) through git.runStackedAction, asking first on the
// default branch unless confirmed, and shows its progress on the row.
func (a *App) runGit(action string, featureBranch, confirmed bool) {
	a.mu.Lock()
	e, t, p, v := a.checkout()
	if v == nil || v.running != nil || e.conn == nil {
		a.mu.Unlock()
		return
	}
	st := v.status
	dirty := st != nil && st.HasWorkingTreeChanges
	commits := action == "commit" || action == "commit_push" || action == "commit_push_pr"
	includesCommit := commits && (action == "commit" || dirty || featureBranch)
	isDefault := st != nil && st.IsDefaultRef && !featureBranch
	if !confirmed && requiresConfirm(action, isDefault) && st.RefName != "" {
		a.confirm = &pendingGit{action: action, branch: st.RefName, includesCommit: includesCommit}
		a.mu.Unlock()
		return
	}
	a.confirm, a.cardMenu = nil, ""
	now := time.Now()
	run := &gitRun{phase: now, includesCommit: includesCommit}
	v.running, v.done = run, nil
	payload := map[string]any{"actionId": t3.NewID(), "cwd": v.cwd, "action": action}
	if featureBranch {
		payload["featureBranch"] = true
	}
	if t != nil {
		payload["threadId"] = t.ID
	} else {
		payload["projectId"] = p.ID
	}
	conn, threadID := e.conn, ""
	if t != nil {
		threadID = t.ID
	}
	terms := termsFor(st)
	a.mu.Unlock()
	a.changed()
	failed := false
	conn.Stream("git.runStackedAction", payload, func(raw json.RawMessage) {
		var ev t3.GitActionEvent
		if json.Unmarshal(raw, &ev) != nil {
			return
		}
		a.mu.Lock()
		switch ev.Kind {
		case "phase_started":
			run.label, run.phase, run.hook, run.output = ev.Label, time.Now(), time.Time{}, ""
		case "hook_started":
			run.hook = time.Now()
		case "hook_output":
			if line := lastLine(ev.Text); line != "" {
				run.output = line
			}
		case "action_finished":
			v.running = nil
			done := &gitDone{until: time.Now().Add(10 * time.Second)}
			if r := ev.Result; r != nil {
				done.title, done.desc = r.Toast.Title, r.Toast.Description
				if r.PR.Status == "created" {
					done.title, done.desc = "Created "+terms.short, ""
				}
				if r.Branch.Status == "created" && r.Branch.Name != "" && threadID != "" {
					go a.setThreadBranch(conn, threadID, r.Branch.Name)
				}
			}
			v.done = done
			time.AfterFunc(10*time.Second, a.changed)
		case "action_failed":
			v.running, failed = nil, true
			a.errText = "Action failed: " + ev.Message
		}
		a.mu.Unlock()
		a.changed()
	}, func(err error) {
		a.mu.Lock()
		if v.running == run {
			v.running = nil
		}
		if err != nil && !failed {
			a.errText = "Action failed: " + err.Error()
		}
		a.mu.Unlock()
		a.changed()
	})
}

// lastLine is s's last line with text.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return ""
}

// setThreadBranch records the branch an action moved the thread to.
func (a *App) setThreadBranch(conn *t3.Conn, threadID, branch string) {
	if err := conn.Dispatch(map[string]any{"type": "thread.metadata.update", "threadId": threadID, "branch": branch}); err != nil {
		a.failf("Couldn't record the thread's branch: %v", err)
	}
}

// pull runs vcs.pull and says what it did.
func (a *App) pull() {
	a.mu.Lock()
	e, _, _, v := a.checkout()
	if v == nil || v.running != nil || e.conn == nil {
		a.mu.Unlock()
		return
	}
	run := &gitRun{pull: true, phase: time.Now()}
	v.running, v.done = run, nil
	conn, cwd := e.conn, v.cwd
	a.mu.Unlock()
	a.changed()
	go func() {
		var res struct {
			Status      string `json:"status"`
			RefName     string `json:"refName"`
			UpstreamRef string `json:"upstreamRef"`
		}
		err := conn.Call("vcs.pull", map[string]any{"cwd": cwd}, &res)
		a.mu.Lock()
		if v.running == run {
			v.running = nil
		}
		switch {
		case err != nil:
			a.errText = "Pull failed: " + err.Error()
		case res.Status == "pulled":
			upstream := res.UpstreamRef
			if upstream == "" {
				upstream = "upstream"
			}
			v.done = &gitDone{title: "Pulled", desc: "Updated " + res.RefName + " from " + upstream, until: time.Now().Add(10 * time.Second)}
		default:
			v.done = &gitDone{title: "Already up to date", desc: res.RefName + " is already synchronized.", until: time.Now().Add(10 * time.Second)}
		}
		a.mu.Unlock()
		a.changed()
		if err == nil {
			time.AfterFunc(10*time.Second, a.changed)
		}
	}()
}

// initGit makes the folder a repository with vcs.init.
func (a *App) initGit() {
	a.mu.Lock()
	e, _, _, v := a.checkout()
	if v == nil || v.initing || e.conn == nil {
		a.mu.Unlock()
		return
	}
	v.initing = true
	conn, cwd := e.conn, v.cwd
	a.mu.Unlock()
	a.changed()
	go func() {
		err := conn.Call("vcs.init", map[string]any{"cwd": cwd}, nil)
		a.mu.Lock()
		v.initing = false
		a.mu.Unlock()
		if err != nil {
			a.failf("Git initialization failed: %v", err)
		}
		a.changed()
	}()
}

// findRefs loads a page of the open branch picker's refs for its search
// (vcs.listRefs, 100 at a time, the search as a ref name): the first, or
// with more the one after those it has. It waits while a page is on its
// way, and a page for an older search is dropped.
func (a *App) findRefs(more bool) {
	a.mu.Lock()
	e, _, _, v := a.checkout()
	pk := a.branchPicker
	if pk == nil || v == nil || e.conn == nil || pk.loading {
		a.mu.Unlock()
		return
	}
	in := map[string]any{"cwd": v.cwd, "limit": 100}
	if q := newRefName(pk.query); q != "" {
		in["query"] = q
	}
	var base *t3.VcsRefs
	if more {
		base = a.pickerRefs(v)
		if base == nil || base.NextCursor == nil {
			a.mu.Unlock()
			return
		}
		in["cursor"] = *base.NextCursor
	}
	pk.loading = true
	conn, gen := e.conn, pk.gen
	a.mu.Unlock()
	a.changed()
	go func() {
		var page t3.VcsRefs
		err := conn.Call("vcs.listRefs", in, &page)
		a.mu.Lock()
		if a.branchPicker == pk && pk.gen == gen {
			pk.loading = false
			if err == nil {
				pk.found = appendRefs(base, &page)
			}
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// inside reports whether path is dir or under it, as the server writes
// paths: case aside on Windows.
func inside(path, dir string) bool {
	norm := func(p string) string { return strings.TrimRight(strings.ReplaceAll(p, `\`, "/"), "/") }
	path, dir = norm(path), norm(dir)
	if len(dir) > 1 && dir[1] == ':' {
		path, dir = strings.ToLower(path), strings.ToLower(dir)
	}
	return dir != "" && (path == dir || strings.HasPrefix(path, dir+"/"))
}

// refTarget is where picking a ref takes a thread
// (resolveBranchSelectionTarget): the checkout to switch in, the worktree
// the thread then works in ("" for the project's own folder), and whether
// the ref is checked out there already.
type refTarget struct {
	cwd, worktree string
	reuse         bool
}

// selectionTarget is where picking r takes a thread working in worktree
// ("" for the project's folder, root): a ref checked out somewhere is
// used where it is, the project's folder when that checkout holds it;
// else the default branch takes a worktree's thread back to the folder,
// and any other ref switches the checkout the thread is in.
func selectionTarget(root, worktree string, r t3.VcsRef) refTarget {
	switch {
	case r.WorktreePath != "" && inside(root, r.WorktreePath):
		return refTarget{cwd: r.WorktreePath, reuse: true}
	case r.WorktreePath != "":
		return refTarget{cwd: r.WorktreePath, worktree: r.WorktreePath, reuse: true}
	case worktree != "" && !r.IsDefault:
		return refTarget{cwd: worktree, worktree: worktree}
	}
	return refTarget{cwd: root}
}

// turnRunning reports whether t's turn runs, so it can be stopped
// (deriveCanInterruptRunningThread): T3 won't move a thread to another
// checkout under it.
func turnRunning(t *t3.Thread) bool { return t.ActiveRunID != "" || activeStatus[t.ActivityRunStatus] }

// moveBlocked reports whether picking r would move thread t (nil for the
// draft) to another checkout while its turn runs. Call with the lock held.
func moveBlocked(t *t3.Thread, root string, r t3.VcsRef) bool {
	return t != nil && turnRunning(t) && selectionTarget(root, t.WorktreePath, r).worktree != t.WorktreePath
}

// worktreeBlocked is the branch picker's notice while some ref can't be
// picked for the turn that runs.
const worktreeBlocked = "Stop the current turn to switch worktrees."

// localBranchName is the branch checking out remote ref name makes
// (deriveLocalBranchNameFromRemoteRef): the name past its remote.
func localBranchName(name string) string {
	if i := strings.Index(name, "/"); i > 0 && i < len(name)-1 {
		return name[i+1:]
	}
	return name
}

// pickRef is the branch picker's choice (selectBranch, createRef):
// ref r, or with create a new ref named r.Name. For a draft starting a
// new worktree it's only the base; another draft switches its checkout. A
// thread's pick goes where selectionTarget says, and not at all while its
// turn runs if that would move it: a ref checked out somewhere moves the
// thread there as it is; otherwise that checkout switches with
// vcs.switchRef (a new ref is made in the thread's own, vcs.createRef),
// the row naming it meanwhile (a remote ref by the local branch it
// makes), and the thread records the branch and worktree it landed in.
// The server sends the new status itself; the refs are read again.
func (a *App) pickRef(th *ui.Theme, r t3.VcsRef, create bool) {
	a.mu.Lock()
	e, t, p, v := a.checkout()
	a.branchPicker = nil
	if v == nil || e.conn == nil || r.Name == "" {
		a.mu.Unlock()
		return
	}
	if t == nil && a.draftWorktree {
		a.draftBase = r.Name
		a.mu.Unlock()
		th.Focus("composer")
		return
	}
	target := refTarget{cwd: v.cwd}
	var sessions []string
	if t != nil {
		target.worktree = t.WorktreePath
		if !create {
			target = selectionTarget(p.WorkspaceRoot, t.WorktreePath, r)
		}
		if target.worktree != t.WorktreePath {
			if turnRunning(t) {
				a.mu.Unlock()
				return
			}
			sessions = a.sessionsLocked(t)
		}
	}
	conn, key, threadID := e.conn, vcsKey(e.id(), v.cwd), ""
	if t != nil {
		threadID = t.ID
	}
	if target.reuse {
		a.mu.Unlock()
		th.Focus("composer")
		go a.moveThread(conn, threadID, r.Name, target.worktree, sessions)
		return
	}
	branch := r.Name
	if r.IsRemote {
		branch = localBranchName(r.Name)
	}
	v.switching = branch
	a.mu.Unlock()
	a.changed()
	th.Focus("composer")
	go func() {
		var res struct {
			RefName string `json:"refName"`
		}
		what, err := "Failed to switch ref.", error(nil)
		if create {
			what = "Failed to create and switch ref."
			err = conn.Call("vcs.createRef", map[string]any{"cwd": target.cwd, "refName": r.Name, "switchRef": true}, &res)
		} else {
			err = conn.Call("vcs.switchRef", map[string]any{"cwd": target.cwd, "refName": r.Name}, &res)
		}
		a.mu.Lock()
		v.switching = ""
		a.mu.Unlock()
		if err != nil {
			a.failf("%s: %v", what, err)
			return
		}
		if (create || r.IsRemote) && res.RefName != "" {
			branch = res.RefName
		}
		if threadID != "" {
			a.moveThread(conn, threadID, branch, target.worktree, sessions)
		}
		a.loadRefs(conn, key, v)
	}()
}

// sessionsLocked are the provider sessions attached to the open thread
// t. Call with the lock held.
func (a *App) sessionsLocked(t *t3.Thread) []string {
	v := a.view
	if v == nil || v.id != t.ID {
		return nil
	}
	var out []string
	for id := range v.sessions {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// moveThread records the branch thread id is on and the worktree it works
// in, "" for the project's folder (thread.metadata.update). Leaving its
// worktree it first detaches sessions, so its agent starts again in the
// new one, as T3's setThreadBranch does.
func (a *App) moveThread(conn *t3.Conn, id, branch, worktree string, sessions []string) {
	for _, s := range sessions {
		if err := conn.Dispatch(map[string]any{"type": "provider-session.detach", "threadId": id, "providerSessionId": s, "reason": "client-requested"}); err != nil {
			a.failf("Couldn't stop the thread's session: %v", err)
		}
	}
	cmd := map[string]any{"type": "thread.metadata.update", "threadId": id, "branch": branch, "worktreePath": nil}
	if worktree != "" {
		cmd["worktreePath"] = worktree
	}
	if err := conn.Dispatch(cmd); err != nil {
		a.failf("Couldn't record the thread's branch: %v", err)
	}
}

// openInEditor opens the checkout in editor: on its own machine with
// shell.openInEditor, or here through the editor's remote URL.
func (a *App) openInEditor(editor string) {
	a.mu.Lock()
	e, t, p, _ := a.checkout()
	if e == nil || editor == "" {
		a.mu.Unlock()
		return
	}
	mode, target := remoteOpen(e)
	cwd, conn, open := gitCwd(t, p), e.conn, a.OpenURL
	a.lastEditor = editor
	if mode == openRemote {
		a.hintSeen = true
	}
	a.mu.Unlock()
	_ = a.store.Put("last-editor", editor)
	switch {
	case mode == openLocal && conn != nil:
		go func() {
			if err := conn.Call("shell.openInEditor", map[string]any{"cwd": cwd, "editor": editor}, nil); err != nil {
				a.failf("Couldn't open %s: %v", editorLabel(editor), err)
			}
		}()
	case mode == openRemote && open != nil:
		_ = a.store.Put("remote-open-hint", true)
		if u := remoteOpenURL(editor, target.Host, target.Username, cwd); u != "" {
			if err := open(u); err != nil {
				a.failf("Couldn't open %s: %v", editorLabel(editor), err)
			}
		}
	}
}

// runScript runs one of the project's scripts in a new terminal in the
// thread's drawer, and remembers it as the project's primary.
func (a *App) runScript(sc t3.ProjectScript, projectID string) {
	a.mu.Lock()
	a.lastScripts[projectID] = sc.ID
	last := map[string]string{}
	for k, v := range a.lastScripts {
		last[k] = v
	}
	d := a.drawers[a.openID]
	if d == nil && a.openID != "" {
		d = &drawer{}
		a.drawers[a.openID] = d
	}
	var err error
	if d != nil {
		d.open = true
		// The leading space keeps it out of the shell's history.
		err = a.newTerminalLocked(d, " "+sc.Command+"\r")
	}
	a.mu.Unlock()
	_ = a.store.Put("last-scripts", last)
	if err != nil {
		a.failf(`Failed to run script "%s": %v`, sc.Name, err)
	}
	a.changed()
}

// renameThread commits the header's rename (resolveRenameCommit): an empty
// title is refused, an unchanged one does nothing, else
// thread.metadata.update.
func (a *App) renameThread(title string) {
	a.mu.Lock()
	e, t := a.thread(a.renaming)
	a.renaming = ""
	if t == nil {
		a.mu.Unlock()
		return
	}
	title = strings.TrimSpace(title)
	if title == "" {
		a.errText = "Thread title cannot be empty"
		a.mu.Unlock()
		return
	}
	if title == t.Title || e.conn == nil {
		a.mu.Unlock()
		return
	}
	conn, id := e.conn, t.ID
	a.mu.Unlock()
	go func() {
		if err := conn.Dispatch(map[string]any{"type": "thread.metadata.update", "threadId": id, "title": title}); err != nil {
			a.failf("Failed to rename thread: %v", err)
		}
	}()
}
