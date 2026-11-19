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

func later(s string) bool { return s != "" && parseTime(s).After(time.Now()) }

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

// age is "now", "4m", "2h", "3d", "5w", as the sidebar writes it.
func age(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return fmt.Sprintf("%dw", int(d.Hours()/24/7))
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

func (a *App) unread(r string, t *t3.Thread) bool {
	return r != a.openID && t.LatestRunCompletedAt != "" && (t.LastVisitedAt == "" || t.LastVisitedAt < t.LatestRunCompletedAt)
}

// row is one inbox line. tag adds the machine's name, for a list that mixes
// environments.
func (a *App) row(e *envState, t *t3.Thread, tag bool) ui.ThreadRow {
	id := ref(e.id(), t.ID)
	r := ui.ThreadRow{
		ID:       id,
		Title:    t.Title,
		Project:  e.projects[t.ProjectID].Title,
		Branch:   t.Branch,
		State:    threadState(t),
		Unread:   a.unread(id, t),
		Selected: id == a.openID,
		Age:      age(anchor(t)),
	}
	if tag {
		r.Env = e.env.Label
	}
	if r.Title == "" {
		r.Title = "Untitled"
	}
	if r.Project == "" {
		r.Project = "No project"
	}
	switch r.State {
	case ui.ThreadRunning:
		if st := parseTime(t.ActivityRunStartedAt); !st.IsZero() {
			r.Timer = duration(time.Since(st))
		}
	case ui.ThreadApproval:
		r.Note = "Needs approval: " + strings.ReplaceAll(t.PendingRuntimeRequest.Kind, "-", " ")
	case ui.ThreadQuestion:
		r.Note = "Has a question"
	case ui.ThreadError:
		r.Note = t.LastError
	}
	if later(t.SnoozedUntil) {
		r.Wakes = parseTime(t.SnoozedUntil).Local().Format("Mon 15:04")
		if parseTime(t.SnoozedUntil).Local().YearDay() == time.Now().YearDay() {
			r.Wakes = parseTime(t.SnoozedUntil).Local().Format("15:04")
		}
	}
	return r
}

// envThread is a thread with the environment it lives on.
type envThread struct {
	*t3.Thread
	e *envState
}

// sidebar fills the inbox: one list across projects and machines, in T3's
// sections.
func (a *App) sidebar(s *ui.Scene) {
	q := strings.ToLower(strings.TrimSpace(a.search))
	var pinned, active, snoozed, settled []envThread
	sources := 0
	connected := false
	for _, e := range a.envs {
		connected = connected || e.conn != nil
		if len(e.threads) > 0 {
			sources++
		}
		for _, t := range e.threads {
			if isSubagent(t) || t.ArchivedAt != "" || t.DeletedAt != "" {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(t.Title), q) {
				continue
			}
			a.shelve(envThread{t, e}, &pinned, &active, &snoozed, &settled)
		}
	}
	a.fillSidebar(s, sources > 1, pinned, active, snoozed, settled)
	switch {
	case !connected && len(pinned)+len(active)+len(snoozed)+len(settled) == 0:
		s.SidebarNote = "Connecting…"
	case q != "":
		s.SidebarNote = "No titles match."
	default:
		s.SidebarNote = "Nothing active."
	}
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

func (a *App) shelve(t envThread, pinned, active, snoozed, settled *[]envThread) {
	switch {
	case t.PinnedAt != "":
		*pinned = append(*pinned, t)
	case later(t.SnoozedUntil):
		*snoozed = append(*snoozed, t)
	case t.SettledAt != "":
		*settled = append(*settled, t)
	default:
		*active = append(*active, t)
	}
}

func (a *App) fillSidebar(s *ui.Scene, tag bool, pinned, active, snoozed, settled []envThread) {
	sort.Slice(pinned, func(i, j int) bool {
		ki, kj := pinned[i].PinOrderKey, pinned[j].PinOrderKey
		if ki != kj {
			return ki < kj
		}
		return pinned[i].PinnedAt < pinned[j].PinnedAt
	})
	sort.Slice(active, func(i, j int) bool {
		ki, kj := active[i].ActiveOrderKey, active[j].ActiveOrderKey
		if (ki != "") != (kj != "") {
			return ki != ""
		}
		if ki != kj {
			return ki < kj
		}
		return anchor(active[i].Thread).After(anchor(active[j].Thread))
	})
	sort.Slice(snoozed, func(i, j int) bool { return snoozed[i].SnoozedUntil < snoozed[j].SnoozedUntil })
	sort.Slice(settled, func(i, j int) bool { return settled[i].SettledAt > settled[j].SettledAt })
	for _, t := range pinned {
		s.Pinned = append(s.Pinned, a.row(t.e, t.Thread, tag))
	}
	for _, t := range active {
		s.Active = append(s.Active, a.row(t.e, t.Thread, tag))
	}
	for _, t := range snoozed {
		s.Snoozed = append(s.Snoozed, a.row(t.e, t.Thread, tag))
	}
	s.Settled = len(settled)
	if a.showSettled {
		for i, t := range settled {
			if i == 100 {
				break
			}
			s.SettledRows = append(s.SettledRows, a.row(t.e, t.Thread, tag))
		}
	}
	s.ShowSnoozed, s.ShowSettled = a.showSnoozed, a.showSettled
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
	a.sidebar(s)
	s.Net = a.net
	s.Error = a.errText
	s.Notice = a.notice
	s.Panel = ui.PanelChanges
	s.Details = a.detailsOpen
	s.DetailsCard = a.detailsCard
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
	if _, t := a.thread(a.ctxRef); t != nil {
		g := ui.Trait{Sel: -1}
		for _, it := range a.threadItems(t) {
			g.Choices = append(g.Choices, ui.TraitChoice{Label: it.label, Desc: it.desc, Danger: it.danger, Off: it.off})
		}
		s.Context = &ui.Traits{Groups: []ui.Trait{g}}
	}
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
		a.draftScene(s)
	case a.openID != "":
		a.threadScene(s)
	default:
		s.Title = ""
		s.Empty = "Pick a thread, or press + to start one."
		if !online {
			s.Empty = "Connecting…"
		}
	}
	return s
}

func (a *App) draftScene(s *ui.Scene) {
	e := a.env(a.draftEnv)
	var p t3.Project
	if e != nil {
		p = e.projects[a.draft]
	}
	s.Title = "New thread"
	s.State = ui.ThreadIdle
	s.StateLabel = "Draft"
	s.Project = p.Title
	if len(a.envs) > 1 && e != nil {
		s.Project += "  ·  " + e.env.Label
	}
	s.PickProject = true
	s.Hero = p.Title
	s.Placeholder = "Ask anything…"
	if a.projectsOpen {
		s.ProjectsHero = a.projectsHero
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

func (a *App) threadScene(s *ui.Scene) {
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
	switch s.State {
	case ui.ThreadRunning:
		s.StateLabel = "Working"
		if st := parseTime(t.ActivityRunStartedAt); !st.IsZero() {
			s.StateLabel = "Working · " + duration(time.Since(st))
		}
	case ui.ThreadApproval:
		s.StateLabel = "Needs approval"
	case ui.ThreadQuestion:
		s.StateLabel = "Has a question"
	case ui.ThreadError:
		s.StateLabel = "Failed"
	default:
		s.StateLabel = "Idle"
		if t.SettledAt != "" {
			s.StateLabel = "Settled"
		}
	}
	s.Project = e.projects[t.ProjectID].Title
	if t.Branch != "" {
		s.Project += "  ›  " + t.Branch
	}
	s.Placeholder = "Ask, steer or queue.   /  commands    @  files"
	v := a.view
	if v == nil || !v.loaded {
		s.Empty = "Loading…"
	} else {
		a.items(s, t, v)
		a.pending(s, t, v)
	}
	a.composerScene(s, a.openID)
	s.Run = &ui.RunInfo{Machine: e.env.Label, Provider: s.Provider, Mode: runtimeLabel(t.RuntimeMode), Branch: t.Branch, Worktree: shortPath(t.WorktreePath)}
	switch s.State {
	case ui.ThreadRunning:
		s.Run.Status = s.StateLabel
	case ui.ThreadIdle:
		s.Run.Status = "Idle"
	default:
		s.Run.Status = s.StateLabel
	}
	s.Run.Model = s.Model
	if s.Effort != "" {
		s.Run.Model += " · " + s.Effort
	}
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

// toolOutput reads MCP-style output: a string, or [{type:"text", text}].
func toolOutput(raw json.RawMessage) string {
	var o struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &o) == nil && len(o.Content) > 0 {
		var s string
		if json.Unmarshal(o.Content, &s) == nil {
			return s
		}
		var parts []struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(o.Content, &parts) == nil {
			var b []string
			for _, p := range parts {
				b = append(b, p.Text)
			}
			return strings.Join(b, "\n")
		}
	}
	return rawString(raw)
}

func toolItem(t *t3.TurnItem, full *t3.TurnItem) ui.Item {
	src := t
	if full != nil {
		src = full
	}
	it := ui.Item{Kind: ui.ItemTool, ID: t.ID}
	it.Live = t.Status == "running" || t.Streaming
	it.Failed = t.Status == "failed" || t.OutputIndicatesFailure
	switch t.Type {
	case "command_execution":
		it.Tool = ui.ToolCommand
		it.Title = firstLine(rawString(src.Input))
		if src.ExitCode != nil {
			it.Meta = fmt.Sprintf("exit %d", *src.ExitCode)
			it.Failed = it.Failed || *src.ExitCode != 0
		}
		it.Body = strings.TrimRight(rawString(src.Output), "\n")
	case "file_change":
		it.Tool = ui.ToolEdit
		it.Title = t.FileName
		if t.Additions != nil {
			del := 0
			if t.Deletions != nil {
				del = *t.Deletions
			}
			it.Meta = fmt.Sprintf("+%d −%d", *t.Additions, del)
		}
		it.Body = src.DiffStr
		if it.Body == "" && full != nil {
			it.Body = "No diff from this provider."
		}
	case "file_search":
		it.Tool, it.Title = ui.ToolSearch, "Searched "+t.Pattern
	case "web_search":
		it.Tool, it.Title = ui.ToolWeb, "Searched the web for "+strings.Join(t.Patterns, ", ")
	case "subagent":
		it.Tool = ui.ToolAgent
		it.Title = t.Title
		if it.Title == "" {
			it.Title = firstLine(t.Prompt)
		}
		it.Body = strings.TrimSpace(strings.Join([]string{src.Prompt, src.Result}, "\n\n"))
	case "dynamic_tool":
		it.Tool = ui.ToolMCP
		it.Title = strings.TrimPrefix(t.ToolName, "mcp__") + " " + firstLine(rawString(t.Input))
		it.Body = toolOutput(src.Output)
	default:
		it.Title = strings.ReplaceAll(t.Type, "_", " ")
	}
	if full == nil && it.Body == "" && toolTypes[t.Type] {
		it.Body = "Loading…"
	}
	return it
}

// items builds the rows. Like T3, a finished run folds everything before its
// final reply into one "Worked for 19s" row.
func (a *App) items(s *ui.Scene, t *t3.Thread, v *threadView) {
	list := v.sorted()
	folds, hide := a.foldRuns(t, v, list)
	working := threadState(t) == ui.ThreadRunning
	thinking := false
	for _, it := range list {
		k := rowKey(it)
		if f, ok := folds[k]; ok {
			s.Items = append(s.Items, f)
		}
		if hidden[it.Type] || hide[k] {
			continue
		}
		switch it.Type {
		case "user_message":
			var chips []string
			text := ui.TrimText(t3.ReplaceImages(it.Text, func(label string) string {
				if len(chips) == 9 {
					return "[" + label + "]"
				}
				chips = append(chips, label)
				return ui.ChipText(len(chips))
			}))
			if text == "" && len(it.Attachments) == 0 {
				continue
			}
			s.Items = append(s.Items, ui.Item{ID: k, Kind: ui.ItemUser, Text: text, Attached: len(it.Attachments),
				Chips: chips, Thumbs: a.attachmentThumbs(v.env, it.Attachments)})
		case "assistant_message", "proposed_plan":
			text := strings.TrimSpace(it.Text)
			if text == "" {
				text = strings.TrimSpace(it.Markdown)
			}
			if text == "" {
				continue
			}
			s.Items = append(s.Items, ui.Item{Kind: ui.ItemAssistant, Text: text})
		case "reasoning":
			live := it.Status == "running" || it.Streaming
			thinking = thinking || live
			s.Items = append(s.Items, ui.Item{Kind: ui.ItemThought, ID: it.ID, Text: strings.TrimSpace(it.Text), Live: live, Open: v.open[it.ID]})
		case "todo_list":
			var todos []ui.Todo
			for _, st := range it.Steps {
				text := st.Step
				if text == "" {
					text = st.Text
				}
				todos = append(todos, ui.Todo{Text: text, Done: st.Status == "completed", Now: st.Status == "in_progress"})
			}
			s.Items = append(s.Items, ui.Item{Kind: ui.ItemTodo, Todos: todos})
		case "error":
			msg := "Error"
			if it.Failure != nil && it.Failure.Message != "" {
				msg = it.Failure.Message
			}
			s.Items = append(s.Items, ui.Item{Kind: ui.ItemError, Text: msg})
		case "system_notice", "run_interrupt_result":
			if it.Message != "" {
				s.Items = append(s.Items, ui.Item{Kind: ui.ItemNotice, Text: it.Message})
			}
		case "compaction":
			text := it.Title
			if text == "" {
				text = "Context compacted"
			}
			s.Items = append(s.Items, ui.Item{Kind: ui.ItemNotice, Text: text})
		default:
			ti := toolItem(it, v.full[it.ID])
			ti.Open = v.open[it.ID]
			s.Items = append(s.Items, ti)
		}
	}
	// Claude's reply text arrives in whole blocks, so show that a run is alive.
	if working && !thinking {
		s.Items = append(s.Items, ui.Item{Kind: ui.ItemThinking, Live: true})
	}
	s.MoreAbove = v.more
	if v.loading {
		s.Notice = "Loading earlier turns…"
	}
	if len(s.Items) == 0 && !v.more {
		s.Empty = "No messages yet."
	}
}

func (a *App) foldRuns(t *t3.Thread, v *threadView, list []*t3.TurnItem) (map[string]ui.Item, map[string]bool) {
	folds := map[string]ui.Item{}
	hide := map[string]bool{}
	var segment []*t3.TurnItem
	var boundary *t3.TurnItem
	closeSeg := func() {
		if len(segment) == 0 {
			return
		}
		runID := segment[0].RunID
		run := v.runs[runID]
		if runID == "" || runID == t.ActiveRunID || activeStatus[run.Status] || run.Status == "queued" {
			return
		}
		for _, i := range segment {
			if i.Type == "error" {
				return
			}
		}
		reply := -1
		for i := len(segment) - 1; i >= 0; i-- {
			if segment[i].Type == "assistant_message" && strings.TrimSpace(segment[i].Text) != "" {
				reply = i
				break
			}
		}
		if reply < 0 {
			reply = len(segment)
		}
		var work []*t3.TurnItem
		tools := 0
		for _, i := range segment[:reply] {
			if !hidden[i.Type] {
				work = append(work, i)
				if toolTypes[i.Type] {
					tools++
				}
			}
		}
		onlyText := true
		for _, i := range work {
			if i.Type != "assistant_message" {
				onlyText = false
			}
		}
		if len(work) == 0 || onlyText {
			return
		}
		key := "fold:" + work[0].ID
		start := parseTime(run.StartedAt)
		if boundary != nil && boundary.RunID == runID {
			if st := parseTime(boundary.StartedAt); !st.IsZero() {
				start = st
			} else if st := parseTime(boundary.UpdatedAt); !st.IsZero() {
				start = st
			}
		}
		last := segment[min(reply, len(segment)-1)]
		end := parseTime(last.CompletedAt)
		if end.IsZero() {
			end = parseTime(last.UpdatedAt)
		}
		if reply == len(segment) && run.CompletedAt != "" {
			end = parseTime(run.CompletedAt)
		}
		label := "Worked"
		if !start.IsZero() && !end.IsZero() && end.After(start) {
			label = "Worked for " + duration(end.Sub(start))
		}
		if run.Status == "interrupted" {
			label = "You stopped this response"
		}
		meta := ""
		if tools > 0 {
			meta = fmt.Sprintf("%d tool call", tools)
			if tools > 1 {
				meta += "s"
			}
		}
		open := v.unfolded[key]
		folds[rowKey(work[0])] = ui.Item{Kind: ui.ItemFold, ID: key, Text: label, Meta: meta, Open: open}
		if !open {
			for _, i := range work {
				hide[rowKey(i)] = true
			}
		}
	}
	for _, it := range list {
		if it.Type == "user_message" || (len(segment) > 0 && it.RunID != segment[0].RunID) {
			closeSeg()
			segment = nil
		}
		if it.Type == "user_message" {
			boundary = it
		} else {
			segment = append(segment, it)
		}
	}
	closeSeg()
	return folds, hide
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
