package core

import (
	"encoding/json"
	"errors"
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

// draftStrategyLocked is where the draft's thread will work
// (orchestration.launchThread's workspaceStrategy): the project's checkout,
// or a new worktree from the ref the branch row names, else the checkout's
// branch, else the default one. Call with the lock held.
func (a *App) draftStrategyLocked() map[string]any {
	_, _, _, v := a.checkout()
	if a.draft == "" || !a.draftWorktree || v == nil {
		return map[string]any{"type": "root"}
	}
	base := a.draftBase
	if base == "" && v.status != nil {
		base = v.status.RefName
	}
	if base == "" && v.refs != nil {
		for _, r := range v.refs.Refs {
			if r.IsDefault && !r.IsRemote {
				base = r.Name
			}
		}
	}
	if base == "" {
		return map[string]any{"type": "root"}
	}
	return map[string]any{"type": "worktree", "baseRef": base}
}

// failf shows "what: err" in the error banner, from any goroutine.
func (a *App) failf(what string, err error) {
	a.mu.Lock()
	a.errText = what + ": " + err.Error()
	a.mu.Unlock()
	a.changed()
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
		a.failf("Publish repository", errors.New("tern can't publish a repository yet; publish it in T3"))
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
		a.failf("Couldn't record the thread's branch", err)
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
			a.failf("Git initialization failed", err)
		}
		a.changed()
	}()
}

// pickRef is the branch picker's choice: a new worktree's base for a
// draft starting one, else the checkout switches to it (or creates it
// first) with vcs.switchRef or vcs.createRef, and a thread records it.
func (a *App) pickRef(th *ui.Theme, name string, create bool) {
	a.mu.Lock()
	e, t, _, v := a.checkout()
	a.branchPicker = nil
	if v == nil || e.conn == nil || name == "" {
		a.mu.Unlock()
		return
	}
	if t == nil && a.draftWorktree {
		a.draftBase = name
		a.mu.Unlock()
		th.Focus("composer")
		return
	}
	v.busy = true
	conn, cwd, threadID := e.conn, v.cwd, ""
	if t != nil {
		threadID = t.ID
	}
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
			err = conn.Call("vcs.createRef", map[string]any{"cwd": cwd, "refName": name, "switchRef": true}, &res)
		} else {
			err = conn.Call("vcs.switchRef", map[string]any{"cwd": cwd, "refName": name}, &res)
		}
		a.mu.Lock()
		v.busy = false
		a.mu.Unlock()
		if err != nil {
			a.failf(what, err)
			return
		}
		if threadID != "" && res.RefName != "" {
			a.setThreadBranch(conn, threadID, res.RefName)
		}
		a.changed()
	}()
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
				a.failf("Couldn't open "+editorLabel(editor), err)
			}
		}()
	case mode == openRemote && open != nil:
		_ = a.store.Put("remote-open-hint", true)
		if u := remoteOpenURL(editor, target.Host, target.Username, cwd); u != "" {
			if err := open(u); err != nil {
				a.failf("Couldn't open "+editorLabel(editor), err)
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
		a.failf(`Failed to run script "`+sc.Name+`"`, err)
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
			a.failf("Failed to rename thread", err)
		}
	}()
}
