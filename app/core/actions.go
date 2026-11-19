package core

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// Handle applies one frame's input. It runs on the UI goroutine, so it may
// read and write editors; network work goes to goroutines.
func (a *App) Handle(th *ui.Theme, actions []ui.Action) {
	for _, act := range actions {
		a.handle(th, act)
	}
	a.ver.Add(1)
}

func (a *App) handle(th *ui.Theme, act ui.Action) {
	id := act.ID
	if a.handleHeader(th, act) {
		return
	}
	switch {
	case strings.HasPrefix(id, "signin."):
		a.handleSignIn(th, act)
		return
	case a.sidebarAction(th, act):
		return
	case act.Kind == "resize" && id == "sidebar.width":
		// The rail's drag, on release; an empty width resets it.
		w, _ := strconv.ParseFloat(act.Text, 32)
		a.mu.Lock()
		a.sidebarWidth = float32(w)
		a.mu.Unlock()
		_ = a.store.Put("sidebar-width", float32(w))
		return
	case act.Kind == "change" && id == "picker.search":
		a.mu.Lock()
		if a.picker != nil {
			a.picker.query = act.Text
		}
		a.mu.Unlock()
		return
	case act.Kind == "submit" && id == "picker.search":
		a.mu.Lock()
		entries, _ := a.pickerModels()
		a.mu.Unlock()
		if len(entries) > 0 {
			a.pickModel(th, 0)
		}
		return
	case (act.Kind == "change" || act.Kind == "caret") && id == "composer":
		if act.Kind == "change" {
			// The stash's list closes when the prompt changes, and a
			// request that comes in now waits for the typing to stop.
			a.mu.Lock()
			a.comp.stashOpen = false
			a.typed()
			a.mu.Unlock()
		}
		a.composerChanged(th, act.Text)
		return
	case (act.Kind == "focus" || act.Kind == "blur") && id == "composer":
		a.mu.Lock()
		a.guard.focused = act.Kind == "focus"
		if !a.guard.focused {
			a.guard.release()
		}
		a.mu.Unlock()
		a.changed()
		return
	case act.Kind == "key" && id == "stash":
		a.stashMenuKey(th, act.Text)
		return
	case act.Kind == "hover" && id == "stash":
		a.mu.Lock()
		if i := atoi(act.Text); i >= 0 && i < len(a.comp.stash) {
			a.comp.stashSel = i
		}
		a.mu.Unlock()
		return
	case act.Kind == "suggestion" && id == "composer":
		a.suggestionKey(th, act.Text == "take")
		return
	case act.Kind == "history" && id == "composer":
		a.recall(th, act.Text == "back")
		return
	case act.Kind == "paste" && id == "composer":
		a.paste(th)
		return
	case act.Kind == "submit" && id == "composer":
		a.mu.Lock()
		asking := len(a.questionsLocked()) > 0
		draft := a.draft != ""
		a.mu.Unlock()
		switch {
		case asking:
			a.answerAll(th)
		case act.Background && !draft:
			// Mod+Alt+Enter on a thread: send, then a new thread.
			if a.send(th, false, false) {
				a.switchTo(th, a.newDraft)
			}
		case draft:
			// Mod+Enter or Mod+Alt+Enter on a new thread starts it in the
			// background and keeps a new thread here.
			a.send(th, false, act.Alt || act.Background)
		default:
			// Mod+Enter while a run works dispatches the other way.
			a.send(th, act.Alt, false)
		}
		return
	case act.Kind == "key" && id == "suggest":
		a.menuKey(th, act.Text)
		return
	case act.Kind == "hover" && id == "suggest":
		a.mu.Lock()
		if i := atoi(act.Text); i >= 0 && i < len(a.comp.items) {
			a.comp.sel = i
		}
		a.mu.Unlock()
		return
	case act.Kind == "submit" && id == "browser.url":
		a.browserGo(act.Text)
		th.Unfocus()
		return
	case act.Kind == "key" && act.Text == "⎋":
		a.closeMenus()
		return
	case act.Kind != "click":
		return
	}
	verb, arg, _ := strings.Cut(id, ":")
	switch verb {
	case "tab":
		if arg == "draft" {
			return
		}
		a.switchTo(th, func() { a.openThread(arg) })
	case "image.remove":
		a.unpaste(th, arg)
	case "tabclose":
		a.closeTab(th, arg)
	case "new":
		a.switchTo(th, a.newDraft)
	case "account":
		if arg == "" {
			a.mu.Lock()
			open := !a.accountOpen
			a.closeMenusLocked()
			a.accountOpen = open
			a.mu.Unlock()
			return
		}
		a.closeMenus()
		a.signOut()
	case "machines":
		// The footer's machines are there for their status: picking one
		// only closes the menu.
		a.mu.Lock()
		open := arg == "" && !a.machinesOpen
		a.closeMenusLocked()
		a.machinesOpen = open
		a.mu.Unlock()
	case "project", "project.hero":
		if arg == "" {
			// The name in "What should we build in tern?" opens the menu.
			a.mu.Lock()
			open := !a.projectsOpen
			a.closeMenusLocked()
			a.projectsOpen = open
			a.mu.Unlock()
			return
		}
		gi, ci := split2(arg)
		a.mu.Lock()
		if gs := a.projectChoices(); gi >= 0 && gi < len(gs) && ci >= 0 && ci < len(gs[gi].ids) {
			if a.draftEnv != gs[gi].e.id() || a.draft != gs[gi].ids[ci] {
				// The model menu is per environment, so start it afresh,
				// and the new project's workspace with it.
				a.draftEnv, a.draft = gs[gi].e.id(), gs[gi].ids[ci]
				a.draftWorktree, a.draftBase = false, ""
				delete(a.choices, "draft")
			}
		}
		a.projectsOpen = false
		a.mu.Unlock()
		th.Focus("composer")
	case "error":
		a.mu.Lock()
		a.errText = ""
		a.mu.Unlock()
	case "menus.close":
		a.mu.Lock()
		details := a.detailsOpen
		a.closeMenusLocked()
		a.mu.Unlock()
		if details {
			// The details popover hands the focus back to the composer.
			th.Focus("composer")
		}
	case "browser.toggle":
		a.toggleBrowser()
		a.mu.Lock()
		opened := a.browsers[a.openID] != nil && a.browsers[a.openID].open
		a.mu.Unlock()
		if opened {
			th.Focus("browser.url")
		}
	case "browser.newtab":
		a.browserTab("new", 0)
	case "browser.tab", "browser.tabclose":
		i, _ := strconv.Atoi(arg)
		a.browserTab(map[string]string{"browser.tab": "show", "browser.tabclose": "close"}[verb], i)
	case "browser.server":
		i, _ := strconv.Atoi(arg)
		a.mu.Lock()
		v := a.browserView()
		a.mu.Unlock()
		if v != nil && i >= 0 && i < len(v.Servers) {
			a.browserGo(v.Servers[i].URL)
		}
	case "browser.back", "browser.forward", "browser.reload":
		a.mu.Lock()
		v := a.browserView()
		a.mu.Unlock()
		if v != nil {
			switch verb {
			case "browser.back":
				v.Send(map[string]any{"type": "history", "delta": -1})
			case "browser.forward":
				v.Send(map[string]any{"type": "history", "delta": 1})
			default:
				v.Send(map[string]any{"type": "reload"})
			}
		}
	case "terminal.toggle":
		a.toggleTerminal()
	case "terminal.new":
		a.newTerminal()
	case "terminal.tab":
		i, _ := strconv.Atoi(arg)
		a.mu.Lock()
		if d := a.drawers[a.openID]; d != nil && i >= 0 && i < len(d.terms) {
			d.active = i
		}
		a.mu.Unlock()
	case "terminal.close":
		i, _ := strconv.Atoi(arg)
		a.closeTerminal(i)
	case "details":
		// T3's thread details: a card in the canvas when it fits, which
		// stays open until toggled (per thread), else a popover from the
		// header. Opening either sends the focus into it.
		a.mu.Lock()
		if arg == "inline" {
			// The card found room inline: the popover's flag goes, so it
			// doesn't come back by itself when the room does.
			a.detailsOpen = false
			a.mu.Unlock()
			return
		}
		if arg == "card" {
			r := a.openID
			if a.draft != "" {
				r = "draft"
			}
			a.cardHidden[r] = !a.cardHidden[r]
			if !a.cardHidden[r] {
				delete(a.cardHidden, r)
				th.FocusCard()
			}
			hidden := map[string]bool{}
			for k := range a.cardHidden {
				hidden[k] = true
			}
			a.mu.Unlock()
			_ = a.store.Put("details-hidden", hidden)
			return
		}
		open := !a.detailsOpen
		a.closeMenusLocked()
		a.detailsOpen = open
		a.mu.Unlock()
		if open {
			th.FocusCard()
		}
	case "sidebar.toggle":
		a.mu.Lock()
		a.sidebarHidden = !a.sidebarHidden
		hidden := a.sidebarHidden
		a.mu.Unlock()
		_ = a.store.Put("sidebar-hidden", hidden)
	case "stop":
		a.mu.Lock()
		clicked := a.stopClicked()
		a.mu.Unlock()
		if clicked {
			a.stop()
		}
	case "more":
		a.loadEarlier()
	case "fold", "group":
		// A "Worked for" fold by its run, a work group by its id.
		a.mu.Lock()
		if v := a.view; v != nil {
			m := v.unfolded
			if verb == "group" {
				m = v.groups
			}
			m[arg] = !m[arg]
		}
		a.mu.Unlock()
	case "entry":
		// An open entry fetches what the socket left out as it builds.
		a.mu.Lock()
		if a.view != nil {
			a.view.open[arg] = !a.view.open[arg]
		}
		a.mu.Unlock()
	case "terminal.run":
		a.runInTerminal(act.Text)
	case "link":
		a.openLink(arg)
	case "cite":
		a.goToCited(th, arg)
	case "fork":
		a.forkFrom(arg)
	case "plan.download":
		a.download(arg, []byte(act.Text))
	case "file.download":
		a.downloadAttachment(arg)
	case "approval":
		// An answer on an approval's row in the timeline, to its request.
		decision, req, _ := strings.Cut(arg, ":")
		a.respondApproval(req, decision)
	case "q":
		qi, oi := split2(arg)
		a.pickOption(th, qi, oi)
	case "q.prev":
		a.moveQuestion(th, -1)
	case "answer":
		a.answerAll(th)
	case "suggest":
		a.pickItem(th, atoi(arg))
	case "stash":
		a.mu.Lock()
		a.comp.stashOpen, a.comp.stashSel = !a.comp.stashOpen, 0
		a.mu.Unlock()
	case "stash.save":
		a.stashKey(th)
	case "stash.restore":
		a.restoreStash(th, atoi(arg))
	case "stash.delete":
		a.deleteStash(atoi(arg))
	case "attach":
		a.attach()
	case "fresh.keep":
		// The fresh session banner's dismiss: keep the current session.
		a.mu.Lock()
		delete(a.comp.fresh, a.composerKey())
		a.mu.Unlock()
	case "project.scratch":
		// From anywhere, Mod+Alt+N: a new thread, then out of its project.
		a.mu.Lock()
		fresh := a.draft == "" && a.newDraftLocked()
		a.mu.Unlock()
		if fresh {
			a.switchTo(th, func() {})
		}
		a.startScratch(th)
	case "dismiss":
		a.mu.Lock()
		req, _ := a.questions()
		a.mu.Unlock()
		if req != "" {
			a.dispatchOpen("thread.user-input.dismiss", map[string]any{"requestId": req})
		}
	case "queue":
		i, rest, _ := strings.Cut(arg, ":")
		a.queueAction(th, atoi(i), rest)
	case "send":
		a.send(th, act.Alt, false)
	case "model":
		a.mu.Lock()
		open := a.picker == nil
		a.closeMenusLocked()
		if open {
			a.picker = &pickerState{tab: a.currentTab()}
		}
		a.mu.Unlock()
		if open {
			th.SetEditorText("picker.search", "")
			th.Focus("picker.search")
		} else {
			th.Focus("composer")
		}
	case "picker.tab":
		a.mu.Lock()
		if a.picker != nil {
			a.picker.tab = atoi(arg)
		}
		a.mu.Unlock()
	case "picker.model":
		a.pickModel(th, atoi(arg))
	case "trait":
		gi, ci := split2(arg)
		a.setTrait(gi, ci)
	case "access":
		if arg == "" {
			a.mu.Lock()
			open := !a.accessOpen
			a.closeMenusLocked()
			a.accessOpen = open
			a.mu.Unlock()
			return
		}
		_, ci := split2(arg)
		a.mu.Lock()
		c := a.choiceFor(a.composerKey())
		if modes := a.runtimeModes(c); ci >= 0 && ci < len(modes) {
			c.runtime = modes[ci].ID
		}
		a.accessOpen = false
		a.mu.Unlock()
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}

func split2(s string) (int, int) {
	a, b, ok := strings.Cut(s, ":")
	if !ok {
		return atoi(a), -1
	}
	return atoi(a), atoi(b)
}

func (a *App) closeMenus() {
	a.mu.Lock()
	a.closeMenusLocked()
	a.mu.Unlock()
}

func (a *App) closeMenusLocked() {
	a.picker, a.accessOpen, a.projectsOpen, a.accountOpen = nil, false, false, false
	a.detailsOpen, a.machinesOpen = false, false
	a.ctxRef, a.ctxBulk, a.ctxDraft, a.ctxKeyboard, a.ctxConfirm = "", false, false, false, false
	a.filterOpen, a.snoozeRef = false, ""
	a.cardMenu, a.branchPicker, a.checksOpen = "", nil, ""
}

// switchTo keeps each thread's unsent text: it saves the composer, runs the
// switch, then restores the new thread's text and focuses the composer.
func (a *App) switchTo(th *ui.Theme, do func()) {
	a.mu.Lock()
	old := a.composerKey()
	a.mu.Unlock()
	saveDraft(old, th.EditorText("composer"))
	do()
	a.mu.Lock()
	now := a.composerKey()
	a.mu.Unlock()
	th.SetEditorText("composer", drafts[now])
	th.Focus("composer")
}

// drafts is unsent composer text per thread ("draft" for a new thread).
// Like T3 it stays on this device; unlike T3 it doesn't survive a restart yet.
var drafts = map[string]string{}

func saveDraft(key, text string) {
	if key == "" {
		return
	}
	if strings.TrimSpace(text) == "" {
		delete(drafts, key)
		return
	}
	drafts[key] = text
}

func (a *App) closeTab(th *ui.Theme, id string) {
	a.mu.Lock()
	var tabs []string
	idx := -1
	for i, t := range a.tabs {
		if t == id {
			idx = i
			continue
		}
		tabs = append(tabs, t)
	}
	a.tabs = tabs
	wasOpen := a.openID == id || (id == "draft" && a.draft != "")
	next := ""
	if wasOpen && len(tabs) > 0 {
		next = tabs[max(0, min(idx, len(tabs))-1)]
	}
	a.mu.Unlock()
	if !wasOpen {
		return
	}
	a.switchTo(th, func() {
		a.mu.Lock()
		a.closeThreadLocked()
		a.draft = ""
		a.mu.Unlock()
		if next != "" {
			a.openThread(next)
		}
	})
}

// newDraft starts a new thread in the open thread's project, else the most
// recently active one on a connected machine.
func (a *App) newDraft() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.newDraftLocked() {
		a.fail(errors.New("add a project in T3 first"))
	}
}

// newDraftLocked is newDraft with the lock held. It's false when there's no
// project to start a thread in.
func (a *App) newDraftLocked() bool {
	var env *envState
	project := ""
	if e, t := a.thread(a.openID); t != nil {
		env, project = e, t.ProjectID
	}
	// Machines that are connected first: a draft on an offline one can't send.
	for _, online := range []bool{true, false} {
		if project != "" {
			break
		}
		var best time.Time
		for _, e := range a.envs {
			if online && e.conn == nil {
				continue
			}
			for _, t := range e.threads {
				if _, ok := e.projects[t.ProjectID]; ok && anchor(t).After(best) && !isSubagent(t) {
					best, env, project = anchor(t), e, t.ProjectID
				}
			}
		}
	}
	if project == "" {
		for _, e := range a.envs {
			for id := range e.projects {
				env, project = e, id
				break
			}
			if project != "" {
				break
			}
		}
	}
	if project == "" {
		return false
	}
	a.closeThreadLocked()
	a.draftEnv, a.draft = env.id(), project
	a.draftWorktree, a.draftBase = false, ""
	delete(a.choices, "draft")
	return true
}

func (a *App) signOut() {
	a.mu.Lock()
	var conns []*t3.Conn
	a.signedIn = false
	a.forgetTerminals()
	a.forgetBrowsers()
	for _, e := range a.envs {
		e.gen++
		if e.conn != nil {
			conns = append(conns, e.conn)
			e.conn = nil
		}
		_ = a.store.Delete(shellName(e.id()))
		t3.ForgetSession(a.store, e.id())
	}
	_ = a.store.Delete("envs.json")
	_ = a.store.Delete("env-filter")
	a.envs = nil
	a.closeThreadLocked()
	a.tabs, a.draft, a.draftEnv = nil, "", ""
	a.choices = map[string]*choice{}
	a.images = map[string][]pastedImage{}
	a.trash = map[string][]pastedImage{}
	a.sent = map[string]*sentThumb{}
	a.login = ui.SignIn{}
	envs := t3.Paired(a.store)
	a.mu.Unlock()
	for _, c := range conns {
		c.Close()
	}
	// Signing out drops the Clerk client, cached sessions, cached thread
	// lists and paired machines.
	a.clerk.SignOut()
	for _, e := range envs {
		t3.ForgetSession(a.store, e.EnvironmentID)
	}
	_ = a.store.Delete("paired.json")
	a.changed()
}

func (a *App) dispatchOpen(kind string, fields map[string]any) { a.dispatchOpenThen(kind, fields, nil) }

// dispatchOpenThen dispatches a command for the open thread and, once the
// server has answered it, runs done with the lock held.
func (a *App) dispatchOpenThen(kind string, fields map[string]any, done func()) {
	a.mu.Lock()
	var conn *t3.Conn
	if e := a.cur(); e != nil && a.draft == "" {
		conn = e.conn
	}
	_, id := splitRef(a.openID)
	if (conn == nil || id == "") && done != nil {
		done()
	}
	a.mu.Unlock()
	if conn == nil || id == "" {
		return
	}
	fields["type"], fields["threadId"] = kind, id
	go func() {
		err := conn.Dispatch(fields)
		a.mu.Lock()
		if err != nil {
			a.fail(err)
		}
		if done != nil {
			done()
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// stop uses thread.stop, which also stops commands from earlier runs and
// delegated tasks; older servers only know run.interrupt.
func (a *App) stop() {
	a.mu.Lock()
	t := a.curThread()
	cfg := a.cfg()
	threadStop := cfg != nil && cfg.Environment.Capabilities.ThreadStop
	a.mu.Unlock()
	if t == nil {
		return
	}
	if threadStop {
		a.dispatchOpen("thread.stop", map[string]any{})
	} else if t.ActiveRunID != "" {
		a.dispatchOpen("run.interrupt", map[string]any{"runId": t.ActiveRunID, "holdQueue": true})
	}
}

// queueAction is queued message i's action: Send now, edit it in the
// composer (or stop editing it), remove it. i -1 is the last.
func (a *App) queueAction(th *ui.Theme, i int, what string) {
	a.mu.Lock()
	runs := a.queuedRuns()
	var active string
	if t := a.curThread(); t != nil {
		active = t.ActiveRunID
	}
	if i == -1 {
		i = len(runs) - 1
	}
	if i < 0 || i >= len(runs) {
		a.mu.Unlock()
		return
	}
	run := runs[i]
	switch what {
	case "edit":
		// The composer takes the message's text in place of the draft,
		// unless a question has it (editBlockedReason).
		if req, _ := a.questions(); req == "" && a.comp.editing == "" {
			a.comp.editing, a.comp.editSaved = run.ID, th.EditorText("composer")
			text := a.view.messages[run.UserMessageID].Text
			a.mu.Unlock()
			th.SetEditorText("composer", text)
			th.Focus("composer")
			return
		}
		a.mu.Unlock()
		return
	case "cancel-edit":
		saved := a.comp.editSaved
		a.comp.editing, a.comp.editSaved = "", ""
		a.mu.Unlock()
		th.SetEditorText("composer", saved)
		return
	}
	a.mu.Unlock()
	switch what {
	case "cancel":
		a.dispatchOpen("queued-run.cancel", map[string]any{"runId": runs[i].ID})
	case "steer":
		if active != "" {
			a.dispatchOpen("queued-message.promote-to-steer", map[string]any{"queuedRunId": runs[i].ID, "targetRunId": active})
		} else {
			a.dispatchOpen("queued-run.start", map[string]any{"runId": runs[i].ID})
		}
	}
}

// asked is the open request and its questions' turn item, unless the
// typing guard holds them back. Call with the lock held.
func (a *App) asked() (string, *t3.TurnItem) {
	t := a.curThread()
	if t == nil || t.PendingRuntimeRequest == nil || t.PendingRuntimeRequest.Kind != "user_input" || a.view == nil || a.guard.held[t.PendingRuntimeRequest.ID] {
		return "", nil
	}
	req := t.PendingRuntimeRequest.ID
	for _, it := range a.view.items {
		if it.RequestID == req && len(it.Questions) > 0 {
			return req, it
		}
	}
	return req, nil
}

// questions is the open request's questions. Call with the lock held.
func (a *App) questions() (string, []t3.Question) {
	req, it := a.asked()
	if it == nil {
		return req, nil
	}
	return req, it.Questions
}

func (a *App) pickAnswer(qi, oi int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	req, qs := a.questions()
	if qi < 0 || qi >= len(qs) || oi < 0 {
		return
	}
	m := a.answers[req]
	if m == nil {
		m = map[int]bool{}
		a.answers[req] = m
	}
	k := qi*100 + oi
	on := !m[k] || !qs[qi].MultiSelect // a single choice stays picked
	if !qs[qi].MultiSelect {
		for j := range qs[qi].Options {
			delete(m, qi*100+j)
		}
	}
	m[k] = on
}

func (a *App) currentTab() int {
	c := a.choiceFor(a.composerKey())
	for i, id := range a.pickerTabs() {
		if id == c.sel.InstanceID {
			return i
		}
	}
	return 0
}

// pickModel switches the composer's model. Options are per model, so only
// the ones the new model also has are kept.
func (a *App) pickModel(th *ui.Theme, i int) {
	a.mu.Lock()
	entries, _ := a.pickerModels()
	if i < 0 || i >= len(entries) {
		a.mu.Unlock()
		return
	}
	e := entries[i]
	c := a.choiceFor(a.composerKey())
	var keep []t3.OptionChoice
	if c.sel.InstanceID == e.instance {
		for _, o := range c.sel.Options {
			for _, d := range e.m.Capabilities.OptionDescriptors {
				if d.ID == o.ID {
					keep = append(keep, o)
				}
			}
		}
	}
	c.sel = t3.ModelSelection{InstanceID: e.instance, Model: e.m.Slug, Options: keep}
	a.picker = nil
	a.mu.Unlock()
	th.Focus("composer")
}

func (a *App) setTrait(gi, ci int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.choiceFor(a.composerKey())
	m := a.model(c.sel)
	if m == nil {
		return
	}
	// Group indexes count only the descriptors traits() shows.
	var shown []t3.OptionDescriptor
	for _, d := range m.Capabilities.OptionDescriptors {
		if d.Type == "boolean" || d.Type == "select" {
			shown = append(shown, d)
		}
	}
	if gi < 0 || gi >= len(shown) {
		return
	}
	d := shown[gi]
	var value any
	if d.Type == "boolean" {
		on, _ := optionValue(c.sel, d).(bool)
		value = !on
	} else {
		if ci < 0 || ci >= len(d.Options) {
			return
		}
		value = d.Options[ci].ID
	}
	var opts []t3.OptionChoice
	for _, o := range c.sel.Options {
		if o.ID != d.ID {
			opts = append(opts, o)
		}
	}
	c.sel.Options = append(opts, t3.OptionChoice{ID: d.ID, Value: value})
}

var slashCommand = regexp.MustCompile(`^/[^\s/]+(\s|$)`)

// sendable drops prompt-injected values like "ultrathink": they're words in
// the prompt, not settings. withPromptEffort adds the prefix T3 uses.
func (a *App) sendable(sel t3.ModelSelection, text string) (t3.ModelSelection, string) {
	m := a.model(sel)
	if m == nil {
		return sel, text
	}
	out := sel
	out.Options = nil
	inject := false
	for _, o := range sel.Options {
		drop := false
		for _, d := range m.Capabilities.OptionDescriptors {
			for _, v := range d.PromptInjectedValues {
				if d.ID == o.ID && o.Value == v {
					drop = true
					if v == "ultrathink" {
						inject = true
					}
				}
			}
		}
		if !drop {
			out.Options = append(out.Options, o)
		}
	}
	if inject && text != "" && !slashCommand.MatchString(text) && !strings.HasPrefix(text, "Ultrathink:") {
		text = "Ultrathink:\n" + text
	}
	return out, text
}

// send works like T3's composer: modes first if they changed, then the
// message with the chosen model. While a run works it steers the run, or
// with alternate (Mod+Enter) queues behind it; servers that resolve the
// dispatch themselves are told which. Drafts become a thread through
// launchThread.
// A new thread's background send starts the thread out of the way and
// keeps a new thread in the composer. It reports whether something went.
func (a *App) send(th *ui.Theme, alternate, background bool) bool {
	text := ui.TrimText(th.EditorText("composer"))
	a.mu.Lock()
	// One send at a time: the button is off while one's on its way.
	if a.comp.sending || (text == "" && len(a.images[a.composerKey()]) == 0) {
		a.mu.Unlock()
		return false
	}
	e := a.cur()
	var conn *t3.Conn
	if e != nil {
		conn = e.conn
	}
	key := a.composerKey()
	if key == "" {
		a.mu.Unlock()
		return false
	}
	if isClear.MatchString(text) && len(a.images[key]) == 0 && a.freshCommand() {
		// /clear alone arms a fresh session rather than going as a message.
		a.comp.fresh[key] = true
		a.mu.Unlock()
		th.SetEditorText("composer", "")
		a.changed()
		return true
	}
	if run := a.comp.editing; run != "" {
		// The composer holds a queued message: the send updates it
		// (queued-run.edit), which keeps its pictures, and the draft it
		// stood in for comes back.
		if len(a.images[key]) > 0 {
			a.fail(errors.New("an edited queued message takes new text only; remove the pictures first"))
			a.mu.Unlock()
			return false
		}
		saved := a.comp.editSaved
		a.comp.editing, a.comp.editSaved = "", ""
		a.mu.Unlock()
		th.SetEditorText("composer", saved)
		a.dispatchOpen("queued-run.edit", map[string]any{"runId": run, "text": text})
		return true
	}
	if conn == nil {
		if e != nil {
			a.fail(errors.New(e.env.Label + " isn't connected yet"))
		}
		a.mu.Unlock()
		return false
	}
	c := a.choiceFor(key)
	if c.sel.Model == "" {
		a.fail(errors.New("pick a model first"))
		a.mu.Unlock()
		return false
	}
	sel, body := a.sendable(c.sel, text)
	runtime := c.runtime
	draft := a.draft
	strategy := a.draftStrategyLocked()
	t := a.curThread()
	smart := e.config != nil && e.config.Environment.Capabilities.ServerResolvedCommandContext
	inline := e.config != nil && e.config.Environment.Capabilities.InlineMessageContext
	imgs := a.images[key]
	approvals := a.shownApprovals()
	// An armed /clear goes with this message, and is spent once it lands.
	fresh := a.comp.fresh[key] && a.freshCommand()
	delete(a.images, key)
	delete(a.trash, key)
	a.comp.sending, a.comp.pos = true, 0
	a.guard.release()
	a.mu.Unlock()
	th.SetEditorText("composer", "")
	saveDraft(key, "")
	a.changed()
	go func() {
		err := a.deliver(e, conn, draft, strategy, t, sel, runtime, body, text, imgs, approvals, alternate, background, smart, inline, fresh)
		a.mu.Lock()
		a.comp.sending = false
		if err == nil && fresh {
			delete(a.comp.fresh, key)
		}
		if err != nil {
			a.fail(err)
			if len(imgs) > 0 && len(a.images[key]) == 0 {
				a.images[key] = imgs
			}
		}
		now := a.composerKey()
		a.mu.Unlock()
		a.changed()
		if err != nil {
			// Put the text back where it came from.
			a.onUI(func(th *ui.Theme) {
				if now == key && th.EditorText("composer") == "" {
					th.SetEditorText("composer", text)
				} else {
					saveDraft(key, text)
				}
			})
		}
	}()
	return true
}

func (a *App) deliver(e *envState, conn *t3.Conn, draft string, strategy map[string]any, t *t3.Thread, sel t3.ModelSelection, runtime, body, text string, imgs []pastedImage, approvals []approval, alternate, background, smart, inline, fresh bool) error {
	feedback, decline := redirect(approvals)
	if feedback != "" {
		// The request takes feedback, so the message is its decline, not a
		// message of its own (sendApprovalFeedback).
		if len(imgs) > 0 {
			return errors.New("Send approval feedback without attachments or added context.")
		}
		return conn.Dispatch(map[string]any{"type": "runtime-request.respond", "threadId": t.ID, "requestId": feedback, "decision": "decline", "feedback": body})
	}
	attachments, recs, err := upload(conn, imgs)
	if err != nil {
		return err
	}
	msg := message(body, imgs, recs, inline)
	if draft != "" {
		title := strings.TrimSpace(ui.StripChips(firstLine(text)))
		if title == "" {
			title = "Image"
		}
		if len(title) > 80 {
			title = title[:80]
		}
		var res struct {
			ThreadID string `json:"threadId"`
		}
		err = conn.Call("orchestration.launchThread", map[string]any{
			"commandId":         t3.NewID(),
			"creationSource":    "web",
			"projectId":         draft,
			"title":             title,
			"generateTitle":     true,
			"modelSelection":    sel,
			"runtimeMode":       runtime,
			"interactionMode":   "default",
			"workspaceStrategy": strategy,
			"initialMessage":    mergeMap(msg, map[string]any{"messageId": t3.NewID(), "attachments": attachments}),
		}, &res)
		if err != nil {
			return err
		}
		r := ref(e.id(), res.ThreadID)
		a.mu.Lock()
		if c := a.choices["draft"]; c != nil {
			cp := *c
			a.choices[r] = &cp
			if !background {
				delete(a.choices, "draft")
			}
		}
		a.mu.Unlock()
		if !background {
			a.openThread(r)
		}
		return nil
	}
	if t == nil {
		return errors.New("that thread is gone")
	}
	if runtime != t.RuntimeMode && runtime != "" {
		if err := conn.Dispatch(map[string]any{"type": "thread.runtime-mode.set", "threadId": t.ID, "runtimeMode": runtime}); err != nil {
			return err
		}
	}
	cmd := map[string]any{
		"type":           "message.dispatch",
		"threadId":       t.ID,
		"messageId":      t3.NewID(),
		"attachments":    attachments,
		"modelSelection": sel,
		"createdBy":      "user",
		"creationSource": "web",
	}
	if fresh {
		// /clear: the agent starts over without the thread's earlier messages.
		cmd["freshContext"] = true
	}
	mergeMap(cmd, msg)
	busy := t.ActiveRunID != ""
	for _, req := range decline {
		// A message sent while approvals wait declines them and redirects
		// the agent (useComposerSend.ts): it steers the waiting turn,
		// T3's follow-up while an approval shows, which is tern's anyway.
		if err := conn.Dispatch(map[string]any{"type": "runtime-request.respond", "threadId": t.ID, "requestId": req, "decision": "decline"}); err != nil {
			return err
		}
	}
	mode := dispatchMode(busy, alternate)
	switch {
	case mode == "queue" || (busy && !smart):
		cmd["dispatchMode"] = map[string]any{"type": "queue_after_active"}
	case smart:
		cmd["dispatchMode"] = map[string]any{"type": "start_immediately"}
		cmd["deliveryIntent"] = mode
	default:
		cmd["dispatchMode"] = map[string]any{"type": "start_immediately"}
	}
	return conn.Dispatch(cmd)
}

// ---------- sign-in ----------

func (a *App) handleSignIn(th *ui.Theme, act ui.Action) {
	if act.Kind == "change" || act.Kind == "key" {
		return
	}
	if act.Kind == "submit" || act.ID == "signin.next" {
		a.signInNext(th)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	switch act.ID {
	case "signin.change":
		a.login = ui.SignIn{Email: a.login.Email}
		a.signIn = nil
		th.Focus("signin.email")
	case "signin.password":
		a.login.Password = !a.login.Password
		th.SetEditorText("signin.code", "")
		th.Focus("signin.code")
	case "signin.pairmode":
		a.login = ui.SignIn{Step: ui.SignInPair}
		th.Focus("signin.pair")
	case "signin.connectmode":
		a.login = ui.SignIn{}
		th.Focus("signin.email")
	}
}

func (a *App) signInNext(th *ui.Theme) {
	a.mu.Lock()
	if a.login.Sending {
		a.mu.Unlock()
		return
	}
	step, si, password := a.login.Step, a.signIn, a.login.Password
	a.login.Sending, a.login.Error = true, ""
	a.mu.Unlock()
	email := strings.TrimSpace(th.EditorText("signin.email"))
	code := strings.TrimSpace(th.EditorText("signin.code"))
	pairURL := strings.TrimSpace(th.EditorText("signin.pair"))
	if step == ui.SignInEmail && email != "" {
		// Clerk answers only once it has sent the email, so show the code
		// field now and send behind it.
		a.mu.Lock()
		a.login = ui.SignIn{Email: email, Step: ui.SignInCode, Sending: true, SendingCode: true}
		a.mu.Unlock()
		th.SetEditorText("signin.code", "")
		th.Focus("signin.code")
		a.changed()
		go a.sendCode(email)
		return
	}
	a.changed()
	go func() {
		var err error
		done := false
		switch step {
		case ui.SignInEmail:
			err = errors.New("enter your T3 Connect email")
		case ui.SignInCode:
			if password {
				err = a.clerk.FinishWithPassword(si, code)
			} else {
				err = a.clerk.FinishWithCode(si, code)
			}
			done = err == nil
		case ui.SignInPair:
			_, err = t3.Pair(a.store, pairURL)
			done = err == nil
		}
		if done {
			// The button keeps spinning until the inbox is there to show.
			a.onUI(func(th *ui.Theme) {
				th.SetEditorText("signin.pair", "")
				th.Focus("composer")
			})
			a.boot()
		}
		a.mu.Lock()
		a.login.Sending = false
		if err != nil {
			a.login.Error = sentence(err.Error())
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// sendCode starts the sign-in and has Clerk email the code, while the code
// step is already up. A failure goes back to the email step with the error.
func (a *App) sendCode(email string) {
	si, err := a.clerk.StartSignIn(email)
	if err == nil && si.EmailAddressID != "" {
		err = a.clerk.SendEmailCode(si)
	}
	a.mu.Lock()
	if a.login.Step != ui.SignInCode || a.login.Email != email || !a.login.SendingCode {
		a.mu.Unlock() // they went back meanwhile
		return
	}
	if err != nil {
		a.login = ui.SignIn{Email: email, Step: ui.SignInEmail, Error: sentence(err.Error())}
	} else {
		a.signIn = si
		a.login.Sending, a.login.SendingCode = false, false
		a.login.HasPassword, a.login.Password = si.HasPassword, si.EmailAddressID == ""
	}
	a.mu.Unlock()
	if err != nil {
		a.onUI(func(th *ui.Theme) { th.Focus("signin.email") })
	}
	a.changed()
}

func sentence(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:] + "."
}
