package core

import (
	"errors"
	"strings"

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
	case id == "branch.search" && act.Kind == "change":
		a.mu.Lock()
		if a.branchPicker != nil {
			a.branchPicker.query, a.branchPicker.hi = act.Text, 0
		}
		a.mu.Unlock()
		return true
	case id == "branch.search" && act.Kind == "key":
		a.mu.Lock()
		if p := a.branchPicker; p != nil {
			if act.Text == "↑" {
				p.hi = max(0, p.hi-1)
			} else {
				p.hi++
			}
		}
		a.mu.Unlock()
		return true
	case id == "branch.search" && act.Kind == "submit":
		a.pickHighlighted(th)
		return true
	case act.Kind != "click":
		return false
	}
	verb, arg, _ := strings.Cut(id, ":")
	if menu, ok := strings.CutPrefix(verb, "card."); ok && arg != "" {
		// A row of one of the card's menus: "card.<menu>:<row>".
		a.cardMenuPick(th, menu, atoi(arg))
		return true
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
	case "card.run", "card.editors", "card.scripts", "card.scripts.more", "card.gitmenu":
		menu := map[string]string{"card.run": "run", "card.editors": "openin", "card.scripts": "scripts", "card.scripts.more": "scripts", "card.gitmenu": "git"}[verb]
		a.mu.Lock()
		open := a.cardMenu != menu
		a.closeCardMenusLocked()
		if open {
			a.cardMenu = menu
		}
		a.mu.Unlock()
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
		}
	case "branch.ref":
		a.mu.Lock()
		name := a.pickerRef(atoi(arg))
		a.mu.Unlock()
		a.pickRef(th, name, false)
	case "branch.create":
		a.mu.Lock()
		name := ""
		if a.branchPicker != nil {
			name = strings.TrimSpace(a.branchPicker.query)
		}
		a.mu.Unlock()
		a.pickRef(th, name, true)
	case "card.initgit":
		a.initGit()
	case "card.git":
		a.gitQuick()
	case "confirm":
		a.mu.Lock()
		c := a.confirm
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
	e, _, p, v := a.checkout()
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

// pickerRef is the name of the branch picker's ref i, as the scene lists
// them. Call with the lock held.
func (a *App) pickerRef(i int) string {
	_, _, _, v := a.checkout()
	if v == nil || a.branchPicker == nil {
		return ""
	}
	refs := a.refPickerScene(v).Refs
	if i < 0 || i >= len(refs) || refs[i].Disabled {
		return ""
	}
	return refs[i].Name
}

// pickHighlighted picks the branch picker's highlighted item: a ref, or
// the ref to create after them.
func (a *App) pickHighlighted(th *ui.Theme) {
	a.mu.Lock()
	_, _, _, v := a.checkout()
	if v == nil || a.branchPicker == nil {
		a.mu.Unlock()
		return
	}
	p := a.refPickerScene(v)
	hi := min(a.branchPicker.hi, len(p.Refs))
	name, create := "", false
	switch {
	case hi < len(p.Refs):
		name = p.Refs[hi].Name
	case p.Create != "":
		name, create = strings.TrimSpace(a.branchPicker.query), true
	}
	a.mu.Unlock()
	a.pickRef(th, name, create)
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
