package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

var activeStatus = map[string]bool{"preparing": true, "starting": true, "running": true, "waiting": true}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func isSubagent(t *t3.Thread) bool {
	return t.Lineage != nil && t.Lineage.RelationshipToParent == "subagent"
}

func anchor(t *t3.Thread) time.Time {
	for _, s := range []string{t.LatestUserMessageAt, t.UpdatedAt, t.CreatedAt} {
		if s != "" {
			return parseTime(s)
		}
	}
	return time.Time{}
}

// duration is "19s", "2m 5s", "1h 3m", as T3 writes them.
func duration(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		if s%60 == 0 {
			return fmt.Sprintf("%dm", s/60)
		}
		return fmt.Sprintf("%dm %ds", s/60, s%60)
	}
	if (s/60)%60 == 0 {
		return fmt.Sprintf("%dh", s/3600)
	}
	return fmt.Sprintf("%dh %dm", s/3600, (s/60)%60)
}

func threadState(t *t3.Thread) ui.ThreadState {
	switch {
	case t.PendingRuntimeRequest != nil && t.PendingRuntimeRequest.Kind == "user_input":
		return ui.ThreadQuestion
	case t.PendingRuntimeRequest != nil:
		return ui.ThreadApproval
	case t.ActiveRunID != "" || activeStatus[t.ActivityRunStatus]:
		return ui.ThreadRunning
	case t.LastError != "":
		return ui.ThreadError
	}
	return ui.ThreadIdle
}

// machines lists the machines for the footer's menu, with their status.
func (a *App) machines(s *ui.Scene) {
	for _, e := range a.envs {
		st := ui.EnvOffline
		switch e.status {
		case "online":
			st = ui.EnvOnline
		case "busy":
			st = ui.EnvBusy
		}
		s.Envs = append(s.Envs, ui.Env{ID: e.id(), Label: e.env.Label, Status: st})
	}
}

// Scene is the whole window for this frame. Between state changes it
// reuses the last one; only the spinner phase moves.
func (a *App) Scene() *ui.Scene {
	now := time.Now()
	phase := float64(now.UnixMilli()%100000) / 1000 * 5
	ver := a.ver.Load()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.memo != nil && a.memoVer == ver && a.memoSec == now.Unix() {
		s := *a.memo
		s.Phase = phase
		return &s
	}
	s := a.build(now)
	a.memo, a.memoVer, a.memoSec = s, ver, now.Unix()
	cp := *s
	cp.Phase = phase
	return &cp
}

func (a *App) build(now time.Time) *ui.Scene {
	s := &ui.Scene{Live: true}
	if !a.booted {
		s.Booting = true
		return s
	}
	if !a.signedIn {
		l := a.login
		s.Login = &l
		return s
	}
	a.machines(s)
	a.sidebar(s, now)
	s.Net = a.net
	s.Error = a.errText
	s.Notice = a.notice
	s.Panel = ui.PanelChanges
	s.Details = a.detailsOpen
	cardKey := a.openID
	if a.draft != "" {
		cardKey = "draft"
	}
	s.DetailsCard = !a.cardHidden[cardKey]
	if c := a.confirm; c != nil {
		var st *t3.VcsStatus
		if _, _, _, v := a.checkout(); v != nil {
			st = v.status
		}
		// Until the status loads the feature branch stays on offer; the
		// server refuses a stale request.
		dirty := st == nil || st.HasWorkingTreeChanges
		title, desc, cont := confirmCopy(c.action, c.branch, c.includesCommit, termsFor(st))
		s.Confirm = &ui.Confirm{Title: title, Description: desc, Continue: cont, BranchOff: featureBranchOff(c.branch, dirty)}
	}
	if m := a.prAlert; m != nil {
		s.Alert = &ui.Alert{Title: "Merge " + m.terms.singular + "?", Description: "This merges #" + strconv.Itoa(m.number) + " using " + m.method + ".", Action: "Merge"}
	}
	s.SidebarHidden, s.SidebarW = a.sidebarHidden, a.sidebarWidth
	s.Term = a.termView()
	s.Web = a.browserView()
	for _, id := range a.tabs {
		title := "Untitled"
		if _, t := a.thread(id); t != nil && t.Title != "" {
			title = t.Title
		}
		s.Tabs = append(s.Tabs, title)
		s.TabIDs = append(s.TabIDs, id)
		if id == a.openID {
			s.TabSel = len(s.Tabs) - 1
		}
	}
	if a.draft != "" {
		s.Tabs = append(s.Tabs, "New thread")
		s.TabIDs = append(s.TabIDs, "draft")
		s.TabSel = len(s.Tabs) - 1
	}
	if len(s.Tabs) == 0 {
		s.TabSel = -1
	}
	running, total, online := 0, 0, false
	for _, e := range a.envs {
		online = online || e.conn != nil
		total += len(e.threads)
		for _, t := range e.threads {
			if !isSubagent(t) && threadState(t) == ui.ThreadRunning {
				running++
			}
		}
	}
	s.Stats = fmt.Sprintf("%s threads   ·   %d running", len3(total), running)
	a.menuScene(s, now)
	if a.accountOpen {
		s.Account = &ui.Traits{Groups: []ui.Trait{{Label: "T3 Connect", Sel: -1, Choices: []ui.TraitChoice{{Label: "Sign out", Desc: "Forgets this device's sessions"}}}}}
	}
	if a.machinesOpen {
		s.Machines = machinesMenu(s.Envs)
	}

	// With nothing open the composer starts a new thread, the way T3 opens
	// on one, rather than asking you to press + first.
	if a.draft == "" && a.openID == "" {
		a.newDraftLocked()
	}
	switch {
	case a.draft != "":
		a.draftScene(s, now)
	case a.openID != "":
		a.threadScene(s, now)
	default:
		s.Title = ""
		s.Empty = "Pick a thread, or press + to start one."
		if !online {
			s.Empty = "Connecting…"
		}
	}
	return s
}

func (a *App) draftScene(s *ui.Scene, now time.Time) {
	e := a.env(a.draftEnv)
	var p t3.Project
	if e != nil {
		p = e.projects[a.draft]
	}
	s.Title = "New thread"
	s.State = ui.ThreadIdle
	if e != nil {
		_, ok := e.projects[a.draft]
		a.headerScene(s, e, nil, p, ok, now)
		a.cardScene(s, e, nil, p, ok, now)
	}
	s.Hero = p.Title
	s.Placeholder = "Ask anything…"
	if a.projectsOpen {
		s.Projects = &ui.Traits{}
		for _, g := range a.projectChoices() {
			s.Projects.Groups = append(s.Projects.Groups, g.trait)
		}
	}
	a.composerScene(s, "draft")
}

// projectGroup is one environment's projects in the new-thread menu.
type projectGroup struct {
	e     *envState
	ids   []string
	trait ui.Trait
}

// projectChoices lists projects to start a thread in, per environment, the
// most recently used first. Call with the lock held.
func (a *App) projectChoices() []projectGroup {
	var out []projectGroup
	n := 0
	for _, e := range a.envs {
		if len(e.projects) > 0 {
			n++
		}
	}
	// A menu, not a browser: the most recent dozen or so are what people
	// pick, split between machines so it fits the window.
	limit := max(5, 12/max(1, n))
	for _, e := range a.envs {
		if len(e.projects) == 0 {
			continue
		}
		last := map[string]time.Time{}
		for _, t := range e.threads {
			if at := anchor(t); at.After(last[t.ProjectID]) {
				last[t.ProjectID] = at
			}
		}
		ids := make([]string, 0, len(e.projects))
		for id := range e.projects {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			if !last[ids[i]].Equal(last[ids[j]]) {
				return last[ids[i]].After(last[ids[j]])
			}
			return e.projects[ids[i]].Title < e.projects[ids[j]].Title
		})
		if len(ids) > limit {
			ids = ids[:limit]
		}
		g := projectGroup{e: e, ids: ids, trait: ui.Trait{Label: e.env.Label, Sel: -1}}
		for i, id := range ids {
			g.trait.Choices = append(g.trait.Choices, ui.TraitChoice{Label: e.projects[id].Title, Desc: homePath(e.projects[id].WorkspaceRoot)})
			if e.id() == a.draftEnv && id == a.draft {
				g.trait.Sel = i
			}
		}
		out = append(out, g)
	}
	return out
}

func (a *App) threadScene(s *ui.Scene, now time.Time) {
	e, t := a.thread(a.openID)
	if t == nil {
		s.Empty = "Loading…"
		return
	}
	s.Title = t.Title
	if s.Title == "" {
		s.Title = "Untitled"
	}
	s.State = threadState(t)
	p, ok := e.projects[t.ProjectID]
	a.headerScene(s, e, t, p, ok, now)
	a.cardScene(s, e, t, p, ok, now)
	s.Placeholder = "Ask, steer or queue.   /  commands    @  files"
	v := a.view
	if v == nil || !v.loaded {
		s.Timeline.Loading = true
	} else {
		root := t.WorktreePath
		if root == "" {
			root = e.projects[t.ProjectID].WorkspaceRoot
		}
		a.timeline(s, t, v, root)
		if len(s.Timeline.Rows) == 0 && !v.more {
			s.Empty = "Send a message to start the conversation."
		}
		a.pending(s, t, v)
	}
	a.composerScene(s, a.openID)
	if v != nil {
		s.Changes = changes(v)
	}
}

func shortPath(p string) string {
	if i := strings.Index(p, "/.t3/"); i >= 0 {
		return "~" + p[i:]
	}
	return p
}

// homePath writes a workspace under a home directory as "~/…".
func homePath(p string) string {
	for _, home := range []string{"/root/", "/home/", "/Users/"} {
		if rest, ok := strings.CutPrefix(p, home); ok {
			if home != "/root/" {
				_, rest, _ = strings.Cut(rest, "/")
			}
			return "~/" + rest
		}
	}
	return p
}

var runtimeModes = []struct{ id, label, desc string }{
	{"full-access", "Full access", "Runs commands and edits without asking"},
	{"auto", "Auto", "The provider decides what needs approval"},
	{"auto-accept-edits", "Accept edits", "Edits go through, commands ask first"},
	{"approval-required", "Ask every time", "Every command and edit asks first"},
}

func runtimeLabel(id string) string {
	for _, m := range runtimeModes {
		if m.id == id {
			return m.label
		}
	}
	return id
}

// changes lists the files the thread's edits touched, newest stats per file.
func changes(v *threadView) []ui.FileChange {
	byFile := map[string]*ui.FileChange{}
	var order []string
	for _, it := range v.sorted() {
		if it.Type != "file_change" || it.FileName == "" {
			continue
		}
		c := byFile[it.FileName]
		if c == nil {
			c = &ui.FileChange{Path: it.FileName, Kind: 'M'}
			byFile[it.FileName] = c
			order = append(order, it.FileName)
		}
		if it.Additions != nil {
			c.Add += *it.Additions
		}
		if it.Deletions != nil {
			c.Del += *it.Deletions
		}
	}
	var out []ui.FileChange
	for _, f := range order {
		out = append(out, *byFile[f])
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil {
		for _, k := range []string{"command", "cmd"} {
			if c, ok := obj[k].(string); ok {
				return c
			}
		}
	}
	out, _ := json.MarshalIndent(json.RawMessage(raw), "", "  ")
	return string(out)
}

// yesNo requests get Approve / For session / Decline.
var yesNo = map[string]bool{"command": true, "file-read": true, "file-change": true, "permission": true}

// pending fills the approval card, questions and queue for the open thread.
func (a *App) pending(s *ui.Scene, t *t3.Thread, v *threadView) {
	if req := t.PendingRuntimeRequest; req != nil {
		var asked *t3.TurnItem
		for _, it := range v.items {
			if it.RequestID == req.ID {
				asked = it
			}
		}
		if asked != nil && len(asked.Questions) > 0 {
			picked := a.answers[req.ID]
			for qi, q := range asked.Questions {
				uq := ui.Question{Header: q.Header, Text: q.Question, MultiSel: q.MultiSelect, Picked: -1}
				for oi, o := range q.Options {
					uq.Options = append(uq.Options, o.Label)
					uq.Chosen = append(uq.Chosen, picked[qi*100+oi])
				}
				s.Questions = append(s.Questions, uq)
			}
		} else {
			detail := strings.ReplaceAll(req.Kind, "-", " ")
			if asked != nil {
				for _, d := range []string{asked.Prompt, asked.Reason, asked.Title, rawString(asked.Input)} {
					if d != "" {
						detail = firstLine(d)
						break
					}
				}
			}
			s.Approval = &ui.Approval{Kind: strings.ReplaceAll(req.Kind, "-", " "), Detail: detail, Cwd: shortPath(t.WorktreePath), YesNo: yesNo[req.Kind]}
		}
	}
	var queued []t3.Run
	for _, r := range v.runs {
		if r.Status == "queued" {
			queued = append(queued, r)
		}
	}
	sort.Slice(queued, func(i, j int) bool { return queued[i].QueuePosition < queued[j].QueuePosition })
	for _, r := range queued {
		text := t3.StripContext(v.messages[r.UserMessageID].Text)
		if text == "" {
			text = "Queued message"
		}
		s.Queue = append(s.Queue, firstLine(text))
	}
}

func (a *App) queuedRuns() []t3.Run {
	if a.view == nil {
		return nil
	}
	var queued []t3.Run
	for _, r := range a.view.runs {
		if r.Status == "queued" {
			queued = append(queued, r)
		}
	}
	sort.Slice(queued, func(i, j int) bool { return queued[i].QueuePosition < queued[j].QueuePosition })
	return queued
}

// ---------- model, options, access ----------

func (a *App) provider(instance string) *t3.Provider {
	if a.cfg() == nil {
		return nil
	}
	for i := range a.cfg().Providers {
		if a.cfg().Providers[i].InstanceID == instance {
			return &a.cfg().Providers[i]
		}
	}
	return nil
}

func (a *App) model(sel t3.ModelSelection) *t3.Model {
	p := a.provider(sel.InstanceID)
	if p == nil {
		return nil
	}
	for i := range p.Models {
		if p.Models[i].Slug == sel.Model {
			return &p.Models[i]
		}
	}
	return nil
}

func providerName(p *t3.Provider) string {
	switch p.Driver {
	case "claudeAgent":
		return "Claude"
	case "codex":
		return "Codex"
	}
	switch p.InstanceID {
	case "claudeAgent":
		return "Claude"
	case "codex":
		return "Codex"
	}
	return p.DisplayName
}

// choiceFor is the composer's model and access for a thread or the draft.
func (a *App) choiceFor(key string) *choice {
	if c := a.choices[key]; c != nil {
		return c
	}
	c := &choice{runtime: "full-access"}
	var t *t3.Thread
	if key != "draft" {
		_, t = a.thread(key)
	}
	switch {
	case t != nil && t.ModelSelection != nil:
		c.sel = *t.ModelSelection
	case key == "draft" && a.draftProject().DefaultModelSelection != nil:
		c.sel = *a.draftProject().DefaultModelSelection
	case a.cfg() != nil && a.cfg().Settings.DefaultModelSelection != nil:
		c.sel = *a.cfg().Settings.DefaultModelSelection
	}
	if t != nil && t.RuntimeMode != "" {
		c.runtime = t.RuntimeMode
	} else if a.cfg() != nil && a.cfg().Settings.DefaultRuntimeMode != "" {
		c.runtime = a.cfg().Settings.DefaultRuntimeMode
	}
	if c.sel.Model == "" && a.cfg() == nil {
		return c // not cached: the config may still be loading
	}
	a.choices[key] = c
	return c
}

func (a *App) draftProject() t3.Project {
	if e := a.env(a.draftEnv); e != nil {
		return e.projects[a.draft]
	}
	return t3.Project{}
}

func (a *App) composerKey() string {
	if a.draft != "" {
		return "draft"
	}
	return a.openID
}

func optionValue(sel t3.ModelSelection, d t3.OptionDescriptor) any {
	for _, o := range sel.Options {
		if o.ID == d.ID {
			return o.Value
		}
	}
	if d.Type == "boolean" {
		if b, ok := d.CurrentValue.(bool); ok {
			return b
		}
		return false
	}
	if s, ok := d.CurrentValue.(string); ok && s != "" {
		return s
	}
	for _, o := range d.Options {
		if o.IsDefault {
			return o.ID
		}
	}
	return ""
}

func (a *App) composerScene(s *ui.Scene, key string) {
	for _, p := range a.images[key] {
		s.Thumbs = append(s.Thumbs, p.thumb)
	}
	c := a.choiceFor(key)
	s.Model, s.Provider = "Pick a model", ""
	if p := a.provider(c.sel.InstanceID); p != nil {
		s.Provider = providerName(p)
	}
	m := a.model(c.sel)
	switch {
	case m != nil:
		s.Model = m.Name
	case c.sel.Model != "":
		s.Model = c.sel.Model
	}
	if m != nil {
		// The options button label: each set value, joined with " · ".
		var parts []string
		first := true
		for _, d := range m.Capabilities.OptionDescriptors {
			v := optionValue(c.sel, d)
			switch d.Type {
			case "select":
				def := ""
				for _, o := range d.Options {
					if o.IsDefault {
						def = o.ID
					}
				}
				for _, o := range d.Options {
					if o.ID == v && (first || o.ID != def) {
						parts = append(parts, o.Label)
					}
				}
				first = false
			case "boolean":
				if b, _ := v.(bool); b {
					parts = append(parts, strings.TrimSuffix(d.Label, " Mode"))
				}
			}
		}
		s.Effort = strings.Join(parts, " · ")
	}
	s.Runtime = runtimeLabel(c.runtime)
	if a.picker != nil {
		s.Picker = a.pickerScene(c)
	}
	if a.traitsOpen && m != nil {
		s.Traits = traits(c.sel, m)
	}
	if a.accessOpen {
		g := ui.Trait{Label: "Access"}
		for i, rm := range runtimeModes {
			g.Choices = append(g.Choices, ui.TraitChoice{Label: rm.label, Desc: rm.desc})
			if rm.id == c.runtime {
				g.Sel = i
			}
		}
		s.Access = &ui.Traits{Groups: []ui.Trait{g}}
	}
}

func traits(sel t3.ModelSelection, m *t3.Model) *ui.Traits {
	t := &ui.Traits{}
	for _, d := range m.Capabilities.OptionDescriptors {
		v := optionValue(sel, d)
		switch d.Type {
		case "boolean":
			on, _ := v.(bool)
			t.Groups = append(t.Groups, ui.Trait{Label: d.Label, Bool: true, On: on})
		case "select":
			g := ui.Trait{Label: d.Label, Sel: -1}
			for j, o := range d.Options {
				g.Choices = append(g.Choices, ui.TraitChoice{Label: o.Label, Desc: o.Description})
				if o.ID == v {
					g.Sel = j
				}
			}
			t.Groups = append(t.Groups, g)
		}
	}
	return t
}

// pickerTabs are Favorites (when there are any) and each ready provider.
func (a *App) pickerTabs() []string {
	var tabs []string
	if a.cfg() == nil {
		return nil
	}
	if len(a.cfg().Settings.Favorites) > 0 {
		tabs = append(tabs, "")
	}
	for _, p := range a.cfg().Providers {
		if p.Enabled && p.Status == "ready" && len(p.Models) > 0 {
			tabs = append(tabs, p.InstanceID)
		}
	}
	return tabs
}

func (a *App) isFav(instance, slug string) bool {
	for _, f := range a.cfg().Settings.Favorites {
		if f.Provider == instance && f.Model == slug {
			return true
		}
	}
	return false
}

type pickEntry struct {
	instance string
	m        t3.Model
}

// pickerModels lists what the picker shows for its tab and query.
func (a *App) pickerModels() ([]pickEntry, int) {
	tabs := a.pickerTabs()
	if a.picker == nil || len(tabs) == 0 {
		return nil, 0
	}
	q := strings.ToLower(strings.TrimSpace(a.picker.query))
	tab := tabs[min(a.picker.tab, len(tabs)-1)]
	var out []pickEntry
	legacy := 0
	for _, p := range a.cfg().Providers {
		if !(p.Enabled && p.Status == "ready") {
			continue
		}
		hiddenSet := map[string]bool{}
		for _, h := range a.cfg().Settings.ProviderModelPreferences[p.InstanceID].HiddenModels {
			hiddenSet[h] = true
		}
		for _, m := range p.Models {
			fav := a.isFav(p.InstanceID, m.Slug)
			switch {
			case q != "":
				// Search runs across every provider, like T3's.
				if !strings.Contains(strings.ToLower(m.Name+" "+m.Slug+" "+p.DisplayName), q) {
					continue
				}
			case tab == "":
				if !fav {
					continue
				}
			case p.InstanceID != tab:
				continue
			}
			if hiddenSet[m.Slug] && !m.IsCustom {
				continue
			}
			if m.IsLegacy && !fav && q == "" {
				legacy++
				continue
			}
			out = append(out, pickEntry{p.InstanceID, m})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		fi, fj := a.isFav(out[i].instance, out[i].m.Slug), a.isFav(out[j].instance, out[j].m.Slug)
		return fi && !fj
	})
	return out, legacy
}

func (a *App) pickerScene(c *choice) *ui.ModelPicker {
	p := &ui.ModelPicker{Query: a.picker.query}
	for _, id := range a.pickerTabs() {
		tab := ui.PickerTab{}
		if id != "" {
			if pr := a.provider(id); pr != nil {
				tab.Provider = providerName(pr)
			}
		}
		p.Tabs = append(p.Tabs, tab)
	}
	if len(p.Tabs) == 0 {
		p.Tabs = []ui.PickerTab{{}}
		return p
	}
	p.Tab = min(a.picker.tab, len(p.Tabs)-1)
	entries, legacy := a.pickerModels()
	for i, e := range entries {
		p.Models = append(p.Models, ui.PickerModel{
			ID:       strconv.Itoa(i),
			Name:     e.m.Name,
			Fav:      a.isFav(e.instance, e.m.Slug),
			Selected: c.sel.InstanceID == e.instance && c.sel.Model == e.m.Slug,
		})
	}
	p.Legacy = legacy
	return p
}

// machinesMenu is the footer's list of the account's machines, with their
// status.
func machinesMenu(envs []ui.Env) *ui.Traits {
	g := ui.Trait{Label: "Machines", Sel: -1}
	for _, e := range envs {
		desc := map[ui.EnvStatus]string{ui.EnvOnline: "Online", ui.EnvBusy: "Busy", ui.EnvOffline: "Offline"}[e.Status]
		g.Choices = append(g.Choices, ui.TraitChoice{Label: e.Label, Desc: desc, Off: e.Status == ui.EnvOffline})
	}
	if len(envs) == 0 {
		g.Choices = []ui.TraitChoice{{Label: "No machines yet", Off: true}}
	}
	return &ui.Traits{Groups: []ui.Trait{g}}
}
