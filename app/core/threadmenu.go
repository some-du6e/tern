package core

import (
	"fmt"
	"strings"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// A thread's actions menu (threadActionMenu.logic.ts, useThreadActionMenu.ts;
// docs/ui/05-header.md §2.3, §5.2), which the header's title, a right-click
// on the header and one on a sidebar row open; and the copy menus a
// right-click on the details card's run context and branch rows open. All
// are T3's in-page menu.

// threadMenu is thread t's actions menu (buildThreadActionMenuItems), the
// items tern can act on in T3's order. T3 offers Find in thread, Snooze,
// Share with guest, Turn into project and Project settings only with their
// surfaces, which tern doesn't have yet, so they're left out the way T3
// leaves out what a client or server doesn't offer. Archive waits until
// the turn stops.
func (a *App) threadMenu(e *envState, t *t3.Thread) []ui.MenuItem {
	loaded := e.config != nil
	var m []ui.MenuItem
	if t.Branch != "" {
		m = append(m, ui.MenuItem{ID: "new-thread-on-branch", Label: "New thread on " + t.Branch, Icon: "message-square-plus"})
	}
	if loaded {
		if t.PinnedAt != "" {
			m = append(m, ui.MenuItem{ID: "unpin", Label: "Unpin thread", Icon: "pin-off"})
		} else {
			m = append(m, ui.MenuItem{ID: "pin", Label: "Pin thread", Icon: "pin"})
		}
		if t.SettledOverride == "settled" {
			m = append(m, ui.MenuItem{ID: "unsettle", Label: "Un-settle thread", Icon: "circle-check"})
		} else {
			m = append(m, ui.MenuItem{ID: "settle", Label: "Settle thread", Icon: "circle-check"})
		}
	}
	m = append(m, ui.MenuItem{ID: "rename", Label: "Rename thread", Icon: "pencil", Sep: true})
	if loaded {
		regen := ui.MenuItem{ID: "regenerate-title", Label: "Regenerate title", Icon: "refresh-cw"}
		if t.TitleRegeneration != nil {
			regen.Label, regen.Off = "Regenerating…", true
		}
		m = append(m, regen)
	}
	m = append(m, ui.MenuItem{ID: "mark-unread", Label: "Mark unread", Icon: "mail-open"})
	if loaded && e.config.Environment.Capabilities.ThreadAutoSettleOptOut {
		on := t.AutoSettleDisabledAt == ""
		m = append(m, ui.MenuItem{ID: "auto-settle", Label: "Auto-settle behavior", Icon: "timer", Children: []ui.MenuItem{
			{ID: "auto-settle:enabled", Label: "Follow settings", Radio: true, Checked: on},
			{ID: "auto-settle:disabled", Label: "Never", Radio: true, Checked: !on},
		}})
	}
	copies := []ui.MenuItem{{ID: "copy-path", Label: "Path", Icon: "folder"}}
	if t.Branch != "" {
		copies = append(copies, ui.MenuItem{ID: "copy-branch", Label: "Branch", Icon: "git-branch"})
	}
	copies = append(copies,
		ui.MenuItem{ID: "copy-thread", Label: "Thread", Icon: "message-square"},
		ui.MenuItem{ID: "copy-thread-id", Label: "Thread ID", Icon: "hash"})
	m = append(m, ui.MenuItem{ID: "copy", Label: "Copy", Icon: "copy", Sep: true, Children: copies})
	m = append(m, ui.MenuItem{ID: "archive", Label: "Archive thread", Icon: "archive", Off: turnRunning(t), Sep: true})
	del := ui.MenuItem{ID: "delete", Label: "Delete", Icon: "trash", Danger: true}
	if a.ctxConfirm {
		del.Label = "Click again to delete"
	}
	return append(m, del)
}

// menuScene is the open in-page menu: a thread's actions, or the card's
// "Copy full path" or "Copy branch name", each only with something to
// copy. Call with the lock held.
func (a *App) menuScene(s *ui.Scene) {
	switch a.ctxCopy {
	case "path":
		if _, t, p, _ := a.checkout(); a.workspacePath(t, p) != "" {
			s.Menu = &ui.ContextMenu{Items: []ui.MenuItem{{ID: "copy-full-path", Label: "Copy full path", Icon: "copy"}}}
		}
		return
	case "branch":
		if _, t, _, v := a.checkout(); a.branchOf(t, v) != "" {
			s.Menu = &ui.ContextMenu{Items: []ui.MenuItem{{ID: "copy-branch-name", Label: "Copy branch name", Icon: "copy"}}}
		}
		return
	}
	if e, t := a.thread(a.ctxRef); t != nil {
		s.Menu = &ui.ContextMenu{Items: a.threadMenu(e, t)}
	}
}

// menuPick runs the open menu's item id.
func (a *App) menuPick(th *ui.Theme, id string) {
	a.mu.Lock()
	switch id {
	case "copy-full-path", "copy-branch-name":
		_, t, p, v := a.checkout()
		text := a.workspacePath(t, p)
		if id == "copy-branch-name" {
			text = a.branchOf(t, v)
		}
		a.closeMenusLocked()
		a.mu.Unlock()
		if text != "" {
			th.WriteClipboard(text)
		}
		return
	}
	e, t := a.thread(a.ctxRef)
	r := a.ctxRef
	if t == nil {
		a.closeMenusLocked()
		a.mu.Unlock()
		return
	}
	if id == "delete" && !a.ctxConfirm {
		a.ctxConfirm = true
		a.mu.Unlock()
		return
	}
	root := e.projects[t.ProjectID].WorkspaceRoot
	a.closeMenusLocked()
	conn := e.conn
	a.mu.Unlock()
	verb, arg, _ := strings.Cut(id, ":")
	cmd := map[string]any{"threadId": t.ID}
	switch verb {
	case "new-thread-on-branch":
		// A draft in the thread's project, on its branch, in its worktree
		// when it has one.
		envID, project, worktree, branch := e.id(), t.ProjectID, t.WorktreePath, t.Branch
		a.switchTo(th, func() {
			a.draftIn(envID, project)
			a.mu.Lock()
			a.draftPath, a.draftBranch = worktree, branch
			a.mu.Unlock()
		})
		return
	case "rename":
		a.startRename(th, r)
		return
	case "copy-path":
		path := t.WorktreePath
		if path == "" {
			path = root
		}
		if path == "" {
			a.failf("Path unavailable", fmt.Errorf("this thread does not have a workspace path to copy"))
			return
		}
		th.WriteClipboard(path)
		return
	case "copy-branch":
		th.WriteClipboard(t.Branch)
		return
	case "copy-thread":
		th.WriteClipboard(hostedThreadLink(e.id(), t.ID))
		return
	case "copy-thread-id":
		th.WriteClipboard(t.ID)
		return
	case "pin", "unpin", "settle", "archive", "delete":
		cmd["type"] = "thread." + verb
	case "unsettle":
		cmd["type"], cmd["reason"] = "thread.unsettle", "user"
	case "mark-unread":
		cmd["type"] = "thread.mark-unread"
	case "regenerate-title":
		cmd["type"], cmd["regenerateTitle"] = "thread.metadata.update", true
	case "auto-settle":
		cmd["type"], cmd["enabled"] = "thread.auto-settle.set", arg == "enabled"
	default:
		return
	}
	if conn == nil {
		a.failf("Thread action unavailable", fmt.Errorf("%s isn't connected", e.env.Label))
		return
	}
	if verb == "archive" || verb == "delete" {
		// It's leaving the inbox, so close its tab too.
		a.closeTab(th, r)
	}
	go func() {
		if err := conn.Dispatch(cmd); err != nil {
			a.failf("Thread action failed", err)
		}
	}()
}

// startRename puts thread r's title in the header's editor, all of it
// selected and focused, as the title's double click and "Rename thread" do;
// from a sidebar row's menu the thread opens first.
func (a *App) startRename(th *ui.Theme, r string) {
	a.mu.Lock()
	open := a.openID == r
	a.mu.Unlock()
	if !open {
		a.switchTo(th, func() { a.openThread(r) })
	}
	a.mu.Lock()
	_, t := a.thread(r)
	if t == nil || a.openID != r {
		a.mu.Unlock()
		return
	}
	a.closeMenusLocked()
	a.renaming = r
	title := t.Title
	a.mu.Unlock()
	th.SetEditorText("header.title", title)
	th.SelectAll("header.title")
	th.Focus("header.title")
}

// hostedThreadLink is a link that opens the thread from anywhere, through
// T3's hosted app (threadLinks.ts): its ids escaped as encodeURIComponent
// does, and its punctuation too, so a sentence's full stop never reads as
// part of an id.
func hostedThreadLink(envID, threadID string) string {
	seg := func(id string) string {
		var b strings.Builder
		for _, c := range []byte(id) {
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
				b.WriteByte(c)
			} else {
				fmt.Fprintf(&b, "%%%02X", c)
			}
		}
		return b.String()
	}
	return "https://app.t3.codes/threads/" + seg(envID) + "/" + seg(threadID)
}
