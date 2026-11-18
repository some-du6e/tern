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
	switch {
	case strings.HasPrefix(id, "signin."):
		a.handleSignIn(th, act)
		return
	case act.Kind == "resize" && id == "sidebar.width":
		// The rail's drag, on release; an empty width resets it.
		w, _ := strconv.ParseFloat(act.Text, 32)
		a.mu.Lock()
		a.sidebarWidth = float32(w)
		a.mu.Unlock()
		_ = a.store.Put("sidebar-width", float32(w))
		return
	case act.Kind == "change" && id == "search":
		a.mu.Lock()
		a.search = act.Text
		a.mu.Unlock()
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
	case act.Kind == "change" && id == "composer":
		a.syncImages(act.Text)
		return
	case act.Kind == "paste" && id == "composer":
		a.paste(th)
		return
	case act.Kind == "submit" && id == "composer":
		a.send(th, act.Alt)
		return
	case act.Kind == "submit" && id == "browser.url":
		a.browserGo(act.Text)
		th.Unfocus()
		return
	case act.Kind == "submit" && id == "answer":
		a.answer(th)
		return
	case act.Kind == "key" && act.Text == "⎋":
		a.closeMenus()
		return
	case act.Kind == "middle":
		return
	case act.Kind == "context":
		a.mu.Lock()
		a.closeMenusLocked()
		if r, ok := strings.CutPrefix(id, "thread:"); ok {
			a.ctxRef = r
		}
		a.mu.Unlock()
		return
	case act.Kind != "click":
		return
	}
	verb, arg, _ := strings.Cut(id, ":")
	switch verb {
	case "thread":
		a.switchTo(th, func() { a.openThread(arg) })
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
	case "shelf":
		a.mu.Lock()
		if arg == "snoozed" {
			a.showSnoozed = !a.showSnoozed
		} else {
			a.showSettled = !a.showSettled
		}
		a.mu.Unlock()
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
			// The menu opens under whichever was clicked: the header's
			// dropdown or the name in "What should we build in tern?".
			a.mu.Lock()
			open := !a.projectsOpen || a.projectsHero != (verb == "project.hero")
			a.closeMenusLocked()
			a.projectsOpen, a.projectsHero = open, verb == "project.hero"
			a.mu.Unlock()
			return
		}
		gi, ci := split2(arg)
		a.mu.Lock()
		if gs := a.projectChoices(); gi >= 0 && gi < len(gs) && ci >= 0 && ci < len(gs[gi].ids) {
			if a.draftEnv != gs[gi].e.id() || a.draft != gs[gi].ids[ci] {
				// The model menu is per environment, so start it afresh.
				a.draftEnv, a.draft = gs[gi].e.id(), gs[gi].ids[ci]
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
		a.closeMenus()
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
		// stays open until toggled, else a popover from the header.
		a.mu.Lock()
		if arg == "card" {
			a.detailsCard = !a.detailsCard
			card := a.detailsCard
			a.mu.Unlock()
			_ = a.store.Put("details-card", card)
			return
		}
		open := !a.detailsOpen
		a.closeMenusLocked()
		a.detailsOpen = open
		a.mu.Unlock()
	case "sidebar.toggle":
		a.mu.Lock()
		a.sidebarHidden = !a.sidebarHidden
		hidden := a.sidebarHidden
		a.mu.Unlock()
		_ = a.store.Put("sidebar-hidden", hidden)
	case "ctx":
		_, j := split2(arg)
		a.threadMenu(th, j)
	case "stop":
		a.stop()
	case "more":
		a.loadEarlier()
	case "fold":
		a.mu.Lock()
		if a.view != nil {
			k := "fold:" + arg
			a.view.unfolded[k] = !a.view.unfolded[k]
		}
		a.mu.Unlock()
	case "tool":
		a.mu.Lock()
		var item *t3.TurnItem
		if a.view != nil {
			a.view.open[arg] = !a.view.open[arg]
			if a.view.open[arg] {
				for _, it := range a.view.items {
					if it.ID == arg {
						item = it
					}
				}
			}
		}
		a.mu.Unlock()
		if item != nil {
			a.loadTool(item)
		}
	case "thought":
		a.mu.Lock()
		if a.view != nil {
			a.view.open[arg] = !a.view.open[arg]
		}
		a.mu.Unlock()
	case "approval":
		a.respond(map[string]any{"decision": arg})
	case "q":
		qi, oi := split2(arg)
		a.pickAnswer(qi, oi)
	case "answer":
		a.answer(th)
	case "dismiss":
		a.dispatchOpen("thread.user-input.dismiss", map[string]any{"requestId": a.pendingID()})
	case "queue":
		i, rest, _ := strings.Cut(arg, ":")
		a.queueAction(atoi(i), rest)
	case "send":
		a.send(th, false)
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
	case "traits":
		a.mu.Lock()
		open := !a.traitsOpen
		a.closeMenusLocked()
		a.traitsOpen = open
		a.mu.Unlock()
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
		if ci >= 0 && ci < len(runtimeModes) {
			a.choiceFor(a.composerKey()).runtime = runtimeModes[ci].id
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
	a.picker, a.traitsOpen, a.accessOpen, a.projectsOpen, a.accountOpen = nil, false, false, false, false
	a.detailsOpen, a.machinesOpen = false, false
	a.ctxRef, a.ctxConfirm = "", false
}

// ctxItem is one entry in a thread's right-click menu.
type ctxItem struct {
	verb, label, desc string
	danger, off       bool
}

// threadItems is the right-click menu for t, like the web sidebar's. Archive
// and Delete wait until a run stops, as T3's do.
func (a *App) threadItems(t *t3.Thread) []ctxItem {
	var out []ctxItem
	if t.PinnedAt != "" {
		out = append(out, ctxItem{verb: "unpin", label: "Unpin"})
	} else {
		out = append(out, ctxItem{verb: "pin", label: "Pin"})
	}
	switch {
	case later(t.SnoozedUntil):
		out = append(out, ctxItem{verb: "unsnooze", label: "Wake up"})
	case t.SettledAt != "":
		out = append(out, ctxItem{verb: "unsettle", label: "Unsettle"})
	default:
		out = append(out, ctxItem{verb: "settle", label: "Settle", desc: "Move it to the done pile"})
	}
	out = append(out, ctxItem{verb: "unread", label: "Mark unread"})
	busy := threadState(t) == ui.ThreadRunning
	why := ""
	if busy {
		why = "Stop the run first"
	}
	out = append(out, ctxItem{verb: "archive", label: "Archive", desc: why, off: busy})
	del := ctxItem{verb: "delete", label: "Delete", desc: why, danger: true, off: busy}
	if a.ctxConfirm && !busy {
		del.label, del.desc = "Click again to delete", "The thread and its history go for good"
	}
	return append(out, del)
}

// threadMenu runs entry j of the open right-click menu.
func (a *App) threadMenu(th *ui.Theme, j int) {
	a.mu.Lock()
	e, t := a.thread(a.ctxRef)
	if t == nil {
		a.closeMenusLocked()
		a.mu.Unlock()
		return
	}
	items := a.threadItems(t)
	if j < 0 || j >= len(items) || items[j].off {
		a.mu.Unlock()
		return
	}
	verb, r := items[j].verb, a.ctxRef
	if verb == "delete" && !a.ctxConfirm {
		a.ctxConfirm = true
		a.mu.Unlock()
		return
	}
	a.closeMenusLocked()
	conn := e.conn
	a.mu.Unlock()
	if conn == nil {
		a.mu.Lock()
		a.fail(errors.New(e.env.Label + " isn't connected"))
		a.mu.Unlock()
		return
	}
	cmd := map[string]any{"threadId": t.ID}
	switch verb {
	case "pin", "unpin", "settle", "archive", "delete":
		cmd["type"] = "thread." + verb
	case "unsettle", "unsnooze":
		cmd["type"], cmd["reason"] = "thread."+verb, "user"
	case "unread":
		cmd["type"] = "thread.mark-unread"
	}
	if verb == "archive" || verb == "delete" {
		// It's leaving the inbox, so close its tab too.
		a.closeTab(th, r)
	}
	go func() {
		if err := conn.Dispatch(cmd); err != nil {
			a.mu.Lock()
			a.fail(err)
			a.mu.Unlock()
			a.changed()
		}
	}()
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

func (a *App) dispatchOpen(kind string, fields map[string]any) {
	a.mu.Lock()
	var conn *t3.Conn
	if e := a.cur(); e != nil && a.draft == "" {
		conn = e.conn
	}
	_, id := splitRef(a.openID)
	a.mu.Unlock()
	if conn == nil || id == "" {
		return
	}
	fields["type"], fields["threadId"] = kind, id
	go func() {
		if err := conn.Dispatch(fields); err != nil {
			a.mu.Lock()
			a.fail(err)
			a.mu.Unlock()
			a.changed()
		}
	}()
}

func (a *App) pendingID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if t := a.curThread(); t != nil && t.PendingRuntimeRequest != nil {
		return t.PendingRuntimeRequest.ID
	}
	return ""
}

func (a *App) respond(extra map[string]any) {
	req := a.pendingID()
	if req == "" {
		return
	}
	extra["requestId"] = req
	a.dispatchOpen("runtime-request.respond", extra)
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

func (a *App) queueAction(i int, what string) {
	a.mu.Lock()
	runs := a.queuedRuns()
	var active string
	if t := a.curThread(); t != nil {
		active = t.ActiveRunID
	}
	a.mu.Unlock()
	if i < 0 || i >= len(runs) {
		return
	}
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

// questions is the open request's questions, from its turn item.
func (a *App) questions() (string, []t3.Question) {
	t := a.curThread()
	if t == nil || t.PendingRuntimeRequest == nil || a.view == nil {
		return "", nil
	}
	req := t.PendingRuntimeRequest.ID
	for _, it := range a.view.items {
		if it.RequestID == req && len(it.Questions) > 0 {
			return req, it.Questions
		}
	}
	return req, nil
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
	on := !m[k]
	if !qs[qi].MultiSelect {
		for j := range qs[qi].Options {
			delete(m, qi*100+j)
		}
	}
	m[k] = on
}

// answer sends {questionId: value | [values] | typed text}.
func (a *App) answer(th *ui.Theme) {
	typed := strings.TrimSpace(th.EditorText("answer"))
	a.mu.Lock()
	req, qs := a.questions()
	picked := a.answers[req]
	answers := map[string]any{}
	for qi, q := range qs {
		var vals []string
		for oi, o := range q.Options {
			if picked[qi*100+oi] {
				v := o.Value
				if v == "" {
					v = o.Label
				}
				vals = append(vals, v)
			}
		}
		switch {
		case typed != "" && (len(vals) == 0 || len(qs) == 1):
			answers[q.ID] = typed
		case len(vals) == 0:
			a.fail(errors.New("answer \"" + q.Header + "\" first"))
			a.mu.Unlock()
			return
		case q.MultiSelect:
			answers[q.ID] = vals
		default:
			answers[q.ID] = vals[0]
		}
	}
	a.mu.Unlock()
	if req == "" || len(answers) == 0 {
		return
	}
	th.SetEditorText("answer", "")
	a.respond(map[string]any{"answers": answers})
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
// message with the chosen model. Busy threads get deliveryIntent "auto" and
// the server picks steer or queue; queue forces the queue. Drafts become a
// thread through launchThread.
func (a *App) send(th *ui.Theme, queue bool) {
	text := ui.TrimText(th.EditorText("composer"))
	a.mu.Lock()
	if text == "" && len(a.images[a.composerKey()]) == 0 {
		a.mu.Unlock()
		return
	}
	e := a.cur()
	var conn *t3.Conn
	if e != nil {
		conn = e.conn
	}
	key := a.composerKey()
	if key == "" {
		a.mu.Unlock()
		return
	}
	if conn == nil {
		if e != nil {
			a.fail(errors.New(e.env.Label + " isn't connected yet"))
		}
		a.mu.Unlock()
		return
	}
	c := a.choiceFor(key)
	if c.sel.Model == "" {
		a.fail(errors.New("pick a model first"))
		a.mu.Unlock()
		return
	}
	sel, body := a.sendable(c.sel, text)
	runtime := c.runtime
	draft := a.draft
	t := a.curThread()
	smart := e.config != nil && e.config.Environment.Capabilities.ServerResolvedCommandContext
	inline := e.config != nil && e.config.Environment.Capabilities.InlineMessageContext
	imgs := a.images[key]
	delete(a.images, key)
	delete(a.trash, key)
	a.notice = "Sending…"
	a.mu.Unlock()
	th.SetEditorText("composer", "")
	saveDraft(key, "")
	a.changed()
	go func() {
		err := a.deliver(e, conn, draft, t, sel, runtime, body, text, imgs, queue, smart, inline)
		a.mu.Lock()
		a.notice = ""
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
}

func (a *App) deliver(e *envState, conn *t3.Conn, draft string, t *t3.Thread, sel t3.ModelSelection, runtime, body, text string, imgs []pastedImage, queue, smart, inline bool) error {
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
			"workspaceStrategy": map[string]any{"type": "root"},
			"initialMessage":    mergeMap(msg, map[string]any{"messageId": t3.NewID(), "attachments": attachments}),
		}, &res)
		if err != nil {
			return err
		}
		r := ref(e.id(), res.ThreadID)
		a.mu.Lock()
		if c := a.choices["draft"]; c != nil {
			a.choices[r] = c
			delete(a.choices, "draft")
		}
		a.mu.Unlock()
		a.openThread(r)
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
	mergeMap(cmd, msg)
	busy := t.ActiveRunID != ""
	switch {
	case queue && busy:
		cmd["dispatchMode"] = map[string]any{"type": "queue_after_active"}
	case smart:
		cmd["dispatchMode"] = map[string]any{"type": "start_immediately"}
		cmd["deliveryIntent"] = "auto"
	case busy:
		cmd["dispatchMode"] = map[string]any{"type": "queue_after_active"}
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
