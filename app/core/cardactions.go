package core

import (
	"errors"
	"strings"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// handleHeader takes the chat header's and the details card's input. It
// reports whether act was theirs.
func (a *App) handleHeader(th *ui.Theme, act ui.Action) bool {
	id := act.ID
	switch {
	case id == "header.title" && act.Kind == "submit", id == "header.title" && act.Kind == "blur":
		a.renameThread(act.Text)
		return true
	case id == "header.title" && act.Kind == "key":
		// Escape leaves the title as it was.
		a.mu.Lock()
		a.renaming = ""
		a.mu.Unlock()
		return true
	case id == "header.title" && act.Kind == "double":
		a.mu.Lock()
		_, t := a.thread(a.openID)
		if t == nil {
			a.mu.Unlock()
			return true
		}
		a.closeMenusLocked()
		a.renaming = a.openID
		title := t.Title
		a.mu.Unlock()
		th.SetEditorText("header.title", title)
		th.SelectAll("header.title")
		th.Focus("header.title")
		return true
	case strings.HasPrefix(id, "header") && act.Kind == "context":
		// A right-click anywhere in the header opens the thread's menu there.
		a.mu.Lock()
		a.closeMenusLocked()
		if a.openID != "" && a.renaming == "" {
			a.ctxRef = a.openID
		}
		a.mu.Unlock()
		return true
	case strings.HasPrefix(id, "card") && act.Kind == "context":
		// One on the card opens its own menu at the pointer; the machine and
		// branch rows offer to copy their path and branch instead.
		a.mu.Lock()
		a.closeCardMenusLocked()
		a.cardMenu, a.cardAtPointer = map[string]string{"card.run": "copypath", "card.runlocked": "copypath", "card.branch": "copybranch"}[id], true
		if a.cardMenu == "" {
			a.cardMenu = "items"
		}
		a.mu.Unlock()
		return true
	case id == "branch.search" && act.Kind == "change":
		// A new search asks the server afresh, from its first item.
		a.mu.Lock()
		pk := a.branchPicker
		search := pk != nil && pk.query != act.Text
		if search {
			pk.query, pk.hi, pk.found, pk.loading, pk.gen = act.Text, 0, nil, false, pk.gen+1
		}
		a.mu.Unlock()
		if search && newRefName(act.Text) != "" {
			a.findRefs(false)
		}
		return true
	case id == "branch.search" && act.Kind == "key":
		// The arrows move the highlight, round from either end.
		a.mu.Lock()
		if p, _ := a.refPickerNow(); p != nil {
			n := len(p.Refs)
			if p.Create != "" {
				n++
			}
			step := 1
			if act.Text == "↑" {
				step = -1
			}
			if n > 0 {
				a.branchPicker.hi = ((a.branchPicker.hi+step)%n + n) % n
			}
		}
		a.mu.Unlock()
		return true
	case id == "branch.more":
		// The list came near its end: the next page of refs.
		a.findRefs(true)
		return true
	case id == "branch.search" && act.Kind == "submit":
		a.pickHighlighted(th)
		return true
	case strings.HasPrefix(id, "card.pr:") && (act.Kind == "click" || act.Kind == "middle"):
		// A pull request row opens it, as a middle click does.
		a.openPR(atoi(strings.TrimPrefix(id, "card.pr:")))
		return true
	case act.Kind != "click":
		return false
	}
	verb, arg, _ := strings.Cut(id, ":")
	switch verb {
	case "card.prwatch":
		a.stopWatchingPR(atoi(arg))
		return true
	case "card.prs":
		// "Show N more" and "Show less".
		a.mu.Lock()
		a.prsOpen = !a.prsOpen
		a.mu.Unlock()
		return true
	case "card.prpill":
		a.prPill(atoi(arg))
		return true
	case "card.checks":
		a.toggleChecks(atoi(arg))
		return true
	case "card.checksall":
		a.mu.Lock()
		a.checksAll = !a.checksAll
		a.mu.Unlock()
		return true
	case "card.check":
		a.openCheck(atoi(arg))
		return true
	case "alert":
		a.confirmMerge(arg == "ok")
		return true
	}
	if menu, ok := strings.CutPrefix(verb, "card."); ok && arg != "" {
		// A row of one of the card's menus: "card.<menu>:<row>".
		a.cardMenuPick(th, menu, atoi(arg))
		return true
	}
	switch verb {
	case "card.openin", "card.script", "card.git", "card.initgit":
		// A row's own action closes any menu the card has open.
		a.mu.Lock()
		a.closeCardMenusLocked()
		a.mu.Unlock()
	}
	switch verb {
	case "header.project":
		a.mu.Lock()
		e, t := a.thread(a.openID)
		if a.draft != "" || t == nil {
			a.mu.Unlock()
			return true
		}
		envID, project := e.id(), t.ProjectID
		a.mu.Unlock()
		a.switchTo(th, func() { a.draftIn(envID, project) })
	case "header.parent":
		a.mu.Lock()
		e, t := a.thread(a.openID)
		parent := ""
		if t != nil && t.Lineage != nil {
			parent = ref(e.id(), t.Lineage.ParentThreadID)
		}
		a.mu.Unlock()
		if parent != "" {
			a.switchTo(th, func() { a.openThread(parent) })
		}
	case "header.unshare":
		a.stopGuestShare()
	case "header.title":
		// The title drops the thread's menu under itself; a second click
		// closes it.
		a.mu.Lock()
		open := a.ctxRef != a.openID
		a.closeMenusLocked()
		if open && a.openID != "" {
			a.ctxRef = a.openID
			th.MenuFrom("header.title")
		}
		a.mu.Unlock()
	case "card.run", "card.editors", "card.scripts", "card.scripts.more", "card.gitmenu", "card.choose":
		menu := map[string]string{"card.run": "run", "card.editors": "openin", "card.scripts": "scripts", "card.scripts.more": "scripts",
			"card.gitmenu": "git", "card.choose": "items"}[verb]
		a.mu.Lock()
		open := a.cardMenu != menu
		a.closeCardMenusLocked()
		if open {
			a.cardMenu, a.cardAtPointer = menu, false
		}
		a.mu.Unlock()
		if open && menu == "git" {
			a.refreshStatus(false)
		}
	case "card.openin":
		a.mu.Lock()
		e, _, _, _ := a.checkout()
		pref := ""
		if e != nil {
			mode, _ := remoteOpen(e)
			pref = preferredEditor(a.editorChoices(e, mode), a.lastEditor)
		}
		a.mu.Unlock()
		a.openInEditor(pref)
	case "card.script":
		a.runPrimaryScript()
	case "card.branch":
		a.mu.Lock()
		open := a.branchPicker == nil
		a.closeCardMenusLocked()
		if open {
			a.branchPicker = &refPicker{}
		}
		a.mu.Unlock()
		if open {
			th.SetEditorText("branch.search", "")
			th.Focus("branch.search")
			a.refreshStatus(true)
		}
	case "branch.ref":
		a.mu.Lock()
		r, ok := a.pickerRef(atoi(arg))
		a.mu.Unlock()
		if ok {
			a.pickRef(th, r, false)
		}
	case "branch.create":
		a.mu.Lock()
		name := ""
		if a.branchPicker != nil {
			name = newRefName(a.branchPicker.query)
		}
		a.mu.Unlock()
		a.pickRef(th, t3.VcsRef{Name: name}, true)
	case "card.initgit":
		a.initGit()
	case "card.git":
		a.gitQuick()
	case "confirm":
		a.mu.Lock()
		c := a.confirm
		_, _, _, v := a.checkout()
		dirty := v == nil || v.status == nil || v.status.HasWorkingTreeChanges
		if arg == "branch" && featureBranchOff("", dirty) != "" {
			// A clean tree gives the feature branch nothing to be named after.
			a.mu.Unlock()
			return true
		}
		a.confirm = nil
		a.mu.Unlock()
		if c != nil && arg != "abort" {
			a.runGit(c.action, arg == "branch", true)
		}
	default:
		return false
	}
	return true
}

// closeCardMenusLocked closes every menu but the details popover the
// card's own menus may hang inside. Call with the lock held.
func (a *App) closeCardMenusLocked() {
	open := a.detailsOpen
	a.closeMenusLocked()
	a.detailsOpen = open
}

// cardMenuPick runs row i of the card's menu: a draft's workspace, the
// preferred editor, a script, or a Git action.
func (a *App) cardMenuPick(th *ui.Theme, menu string, i int) {
	a.mu.Lock()
	e, t, p, v := a.checkout()
	a.cardMenu = ""
	if e == nil {
		a.mu.Unlock()
		return
	}
	switch menu {
	case "run":
		// Row 0 is the Workspace label.
		if i == 1 || i == 2 {
			a.draftWorktree, a.draftBase = i == 2, ""
		}
		a.mu.Unlock()
		th.Focus("composer")
	case "openin":
		mode, _ := remoteOpen(e)
		choices := a.editorChoices(e, mode)
		if i >= 0 && i < len(choices) {
			a.lastEditor = choices[i]
		}
		editor := a.lastEditor
		a.mu.Unlock()
		_ = a.store.Put("last-editor", editor)
	case "scripts":
		if i < 0 || i >= len(p.Scripts) {
			a.mu.Unlock()
			return
		}
		sc := p.Scripts[i]
		a.mu.Unlock()
		a.runScript(sc, p.ID)
	case "git":
		var items []gitMenuItem
		if v != nil {
			st := v.status
			items = buildGitMenu(st, v.running != nil, st == nil || st.HasPrimaryRemote)
		}
		a.mu.Unlock()
		if i < 0 || i >= len(items) || items[i].disabled {
			return
		}
		// T3's Commit opens its commit dialog first; tern commits every
		// change with a generated message, as its quick action does.
		a.runGit(map[string]string{"commit": "commit", "push": "push", "pr": "create_pr"}[items[i].id], false, false)
	case "items":
		// Row 0 is the label; then the items, a separator and Reset.
		switch {
		case i >= 1 && i <= len(cardItems):
			if id := cardItems[i-1].id; a.itemsHidden[id] {
				delete(a.itemsHidden, id)
			} else {
				a.itemsHidden[id] = true
			}
		case i == len(cardItems)+2:
			a.itemsHidden = map[string]bool{}
		}
		hidden := map[string]bool{}
		for k := range a.itemsHidden {
			hidden[k] = true
		}
		a.mu.Unlock()
		_ = a.store.Put("details-items", hidden)
	case "copypath", "copybranch":
		text := a.workspacePath(t, p)
		if menu == "copybranch" {
			text = a.branchOf(t, v)
		}
		a.mu.Unlock()
		if text != "" {
			th.WriteClipboard(text)
		}
	default:
		a.mu.Unlock()
	}
}

// runPrimaryScript runs the script the scripts split shows.
func (a *App) runPrimaryScript() {
	a.mu.Lock()
	_, t, p, _ := a.checkout()
	ps := primaryScript(p.Scripts, a.lastScripts[p.ID])
	a.mu.Unlock()
	switch {
	case ps == nil:
	case t == nil:
		a.failf(`Failed to run script "`+ps.Name+`"`, errors.New("a new thread has no terminal until its first message"))
	default:
		a.runScript(*ps, p.ID)
	}
}

// refPickerNow is the open branch picker as the card lists it, and the
// checkout it's for. Call with the lock held.
func (a *App) refPickerNow() (*ui.BranchPicker, *vcsState) {
	_, t, p, v := a.checkout()
	if v == nil || a.branchPicker == nil {
		return nil, v
	}
	return a.refPickerScene(v, p.WorkspaceRoot, t == nil && a.draftWorktree), v
}

// pickerRef is the branch picker's ref i, as the scene lists them. Call
// with the lock held.
func (a *App) pickerRef(i int) (t3.VcsRef, bool) {
	p, v := a.refPickerNow()
	if p == nil || i < 0 || i >= len(p.Refs) || p.Refs[i].Disabled {
		return t3.VcsRef{}, false
	}
	for _, r := range a.pickerRefs(v).Refs {
		if r.Name == p.Refs[i].Name {
			return r, true
		}
	}
	return t3.VcsRef{}, false
}

// pickHighlighted picks the branch picker's highlighted item: a ref, or
// the ref to create after them.
func (a *App) pickHighlighted(th *ui.Theme) {
	a.mu.Lock()
	p, _ := a.refPickerNow()
	if p == nil {
		a.mu.Unlock()
		return
	}
	hi := a.branchPicker.hi
	r, create, ok := t3.VcsRef{}, false, false
	switch {
	case hi < len(p.Refs):
		r, ok = a.pickerRef(hi)
	case p.Create != "":
		r, create, ok = t3.VcsRef{Name: newRefName(a.branchPicker.query)}, true, true
	}
	a.mu.Unlock()
	if ok {
		a.pickRef(th, r, create)
	}
}

// stopGuestShare ends the open thread's guest link
// (thread.guest-share.close), Stop sharing off until the server answers.
// The thread's record then comes without the link. T3 only logs a failure,
// so the button just comes back.
func (a *App) stopGuestShare() {
	a.mu.Lock()
	e, t := a.thread(a.openID)
	v := a.view
	if t == nil || e.conn == nil || v == nil || v.env != e || v.id != t.ID || v.unsharing {
		a.mu.Unlock()
		return
	}
	v.unsharing = true
	conn, id := e.conn, t.ID
	a.mu.Unlock()
	a.changed()
	go func() {
		_ = conn.Dispatch(map[string]any{"type": "thread.guest-share.close", "threadId": id})
		a.mu.Lock()
		v.unsharing = false
		a.mu.Unlock()
		a.changed()
	}()
}

// draftIn starts a new thread in project on environment envID, as the
// header's project crumb does.
func (a *App) draftIn(envID, project string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closeThreadLocked()
	a.draftEnv, a.draft = envID, project
	a.draftWorktree, a.draftBase = false, ""
	delete(a.choices, "draft")
}
