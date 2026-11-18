package core

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The sidebar's search, its project filter, its menus, and the thread
// actions they send (Sidebar.tsx, threadActionMenu.logic.ts).

// ---------- search ----------

// searchScene lists, in place of the list, every listed thread in scope
// whose title, branch or pull request has all the query's words, in the
// list's order.
func (a *App) searchScene(sb *ui.Sidebar, l sidebarList, q string, now time.Time) {
	words := strings.Fields(strings.ToLower(q))
	match := func(t *t3.Thread) bool {
		hay := strings.ToLower(t.Title + " " + t.Branch)
		for _, pr := range t.PullRequests {
			hay += fmt.Sprintf(" #%d", pr.Number)
		}
		for _, w := range words {
			if !strings.Contains(hay, w) {
				return false
			}
		}
		return true
	}
	a.results = a.results[:0]
	for i, shelf := range [][]sideThread{l.pinned, l.active, l.working, l.snoozed, l.settled} {
		for _, t := range shelf {
			if !match(t.Thread) {
				continue
			}
			r := a.row(t, "", now)
			if i == 4 {
				r.Shelf = "settled"
			}
			sb.Results = append(sb.Results, r)
			a.results = append(a.results, t.ref)
		}
	}
	if len(sb.Results) == 0 {
		sb.NoResults = "No threads found"
		return
	}
	a.searchSel = min(max(0, a.searchSel), len(sb.Results)-1)
	sb.Highlight = a.searchSel
}

// ---------- the project filter ----------

// scopeSet is the projects the list is narrowed to, by "<env>/<project>",
// or nil for all of them. Stored keys whose project is gone or archived
// don't count, and with none left the list shows everything.
func (a *App) scopeSet() map[string]bool {
	var out map[string]bool
	for _, k := range a.scope {
		envID, id := splitRef(k)
		if e := a.env(envID); e != nil {
			if p, ok := e.projects[id]; ok && p.ArchivedAt == "" {
				if out == nil {
					out = map[string]bool{}
				}
				out[k] = true
			}
		}
	}
	return out
}

func (a *App) scopeTitle(k string) string {
	envID, id := splitRef(k)
	if e := a.env(envID); e != nil {
		return e.projects[id].Title
	}
	return ""
}

// scopeIcons is the filter trigger's projects and its label.
func (a *App) scopeIcons(scope map[string]bool) ([]ui.ProjectIcon, string) {
	if len(scope) == 0 {
		return nil, "Filter threads by project"
	}
	var icons []ui.ProjectIcon
	var names []string
	for _, k := range a.scope {
		if !scope[k] {
			continue
		}
		envID, id := splitRef(k)
		p := a.env(envID).projects[id]
		icons = append(icons, projectIcon(p))
		names = append(names, p.Title)
	}
	return icons, "Filter threads by project: " + strings.Join(names, ", ")
}

// filterOrder is every project the filter offers, most recently active
// first. It's fixed when the popup opens, so rows don't move under it.
func (a *App) filterOrder() []string {
	last := map[string]time.Time{}
	var keys []string
	for _, e := range a.envs {
		for _, t := range e.threads {
			k := ref(e.id(), t.ProjectID)
			if at := parseTime(t.LatestUserMessageAt); at.After(last[k]) {
				last[k] = at
			}
		}
		for id, p := range e.projects {
			if p.ArchivedAt == "" {
				keys = append(keys, ref(e.id(), id))
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if !last[keys[i]].Equal(last[keys[j]]) {
			return last[keys[i]].After(last[keys[j]])
		}
		return a.scopeTitle(keys[i]) < a.scopeTitle(keys[j])
	})
	return keys
}

// filterScene is the filter's popup: "All projects" (not while searching
// it), then the projects whose name has the query. One row is
// highlighted: on opening the scope's own, then the first match as the
// query changes, then the arrows' and the pointer's (T3's combobox, with
// autoHighlight).
func (a *App) filterScene(sb *ui.Sidebar, scope map[string]bool) {
	f := &ui.ProjectFilter{}
	q := strings.ToLower(strings.TrimSpace(a.filterQuery))
	a.filterShown = a.filterShown[:0]
	if q == "" {
		f.Items = append(f.Items, ui.FilterItem{Label: "All projects", Selected: len(scope) == 0})
		a.filterShown = append(a.filterShown, "")
	}
	for _, k := range a.filterKeys {
		envID, id := splitRef(k)
		e := a.env(envID)
		if e == nil {
			continue
		}
		p, ok := e.projects[id]
		if !ok || !strings.Contains(strings.ToLower(p.Title), q) {
			continue
		}
		f.Items = append(f.Items, ui.FilterItem{ID: k, Label: p.Title, Icon: projectIcon(p), Selected: scope[k]})
		a.filterShown = append(a.filterShown, k)
	}
	if a.filterHi < 0 {
		// The combobox's value: All projects, or the one project in scope.
		// Several have none, and the first row takes the highlight.
		a.filterHi = 0
		for i, k := range a.filterShown {
			if len(scope) == 1 && scope[k] {
				a.filterHi = i
			}
		}
	}
	if a.filterHi = min(a.filterHi, len(f.Items)-1); a.filterHi >= 0 {
		f.Items[a.filterHi].Highlighted = true
	}
	sb.Filter = f
}

// filterKey is a key in the filter's search: the arrows move the
// highlight, wrapping; Enter picks the highlighted row, and Shift+Enter
// adds or drops its project, keeping the popup open; Escape closes it.
func (a *App) filterKey(act ui.Action) {
	a.mu.Lock()
	n := len(a.filterShown)
	switch {
	case act.Text == "⎋":
		a.filterOpen = false
	case (act.Text == "↓" || act.Text == "↑") && n > 0:
		step := 1
		if act.Text == "↑" {
			step = -1
		}
		a.filterHi = (max(0, a.filterHi) + step + n) % n
	case act.Text == "⏎" && a.filterHi >= 0 && a.filterHi < n:
		k := a.filterShown[a.filterHi]
		a.mu.Unlock()
		if k != "" || !act.Shift {
			a.pickProject(k, act.Shift)
		}
		return
	}
	a.mu.Unlock()
}

// setScope narrows the list to keys and saves it. A new scope starts the
// list afresh: the selection clears and the Settled shelf folds back to its
// first page. Call with the lock held.
func (a *App) setScope(keys []string) {
	a.scope = keys
	a.selected, a.anchor = map[string]bool{}, ""
	a.settledMore = 0
	_ = a.store.Put("sidebar-scope", keys)
}

// pickProject is a click on one of the filter's projects: it scopes the
// list to it, or with Shift adds or drops it and keeps the popup open.
func (a *App) pickProject(k string, shift bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case k == "":
		a.setScope(nil)
	case shift:
		keys := []string{}
		found := false
		for _, s := range a.scope {
			if s == k {
				found = true
				continue
			}
			keys = append(keys, s)
		}
		if !found {
			keys = append(keys, k)
		}
		a.setScope(keys)
		return
	default:
		a.setScope([]string{k})
	}
	a.filterOpen = false
}

// ---------- snooze presets ----------

// defaultSnoozePresets are T3's, for servers whose settings have none.
var defaultSnoozePresets = []t3.SnoozePreset{
	{Kind: "after", Minutes: 60}, {Kind: "after", Minutes: 180},
	{Kind: "at", Day: "today", Time: "18:00"}, {Kind: "at", Day: "tomorrow", Time: "09:00"}, {Kind: "at", Day: "monday", Time: "09:00"},
}

var weekdays = map[string]time.Weekday{"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday}

// snoozeChoice is one resolved preset: what the menu says and when it wakes.
type snoozeChoice struct {
	id, label, when string
	until           time.Time
	preset          t3.SnoozePreset
}

func presetID(p t3.SnoozePreset) string {
	if p.Kind == "after" {
		return "after:" + strconv.Itoa(p.Minutes)
	}
	return "at:" + p.Day + ":" + p.Time
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(n) + " " + unit + "s"
}

// clock is a wake time as the snooze menu writes it: "6:00 PM".
func clock(t time.Time) string { return t.Format("3:04 PM") }

// resolvePreset is one preset as of now (resolveSnoozePreset): its label and
// wake time. A time today that has passed is left out.
func resolvePreset(p t3.SnoozePreset, now time.Time) (snoozeChoice, bool) {
	c := snoozeChoice{id: presetID(p), preset: p}
	ahead := 0
	if p.Kind == "after" {
		c.until = now.Add(time.Duration(p.Minutes) * time.Minute)
		switch {
		case p.Minutes%(24*60) == 0:
			c.label = "In " + plural(p.Minutes/(24*60), "day")
		case p.Minutes%60 == 0:
			c.label = "In " + plural(p.Minutes/60, "hour")
		default:
			c.label = "In " + plural(p.Minutes, "minute")
		}
	} else {
		switch p.Day {
		case "today":
		case "tomorrow":
			ahead = 1
		default:
			if ahead = (int(weekdays[p.Day]) - int(now.Weekday()) + 7) % 7; ahead == 0 {
				ahead = 7
			}
		}
		hh, mm, _ := strings.Cut(p.Time, ":")
		h, _ := strconv.Atoi(hh)
		m, _ := strconv.Atoi(mm)
		y, mo, d := now.Date()
		c.until = time.Date(y, mo, d+ahead, h, m, 0, 0, now.Location())
		if !c.until.After(now) {
			return c, false
		}
		switch {
		case ahead == 0 && h < 12:
			c.label = "This morning"
		case ahead == 0 && h < 17:
			c.label = "This afternoon"
		case ahead == 0:
			c.label = "This evening"
		case ahead == 1:
			c.label = "Tomorrow"
		case p.Day == "monday":
			c.label = "Next week"
		case ahead == 7:
			c.label = "Next " + strings.ToUpper(p.Day[:1]) + p.Day[1:]
		default:
			c.label = strings.ToUpper(p.Day[:1]) + p.Day[1:]
		}
	}
	// Durations and "Next week" don't name the day, so a wake on another
	// day says which.
	c.when = clock(c.until)
	namesDay := p.Kind == "at" && c.label != "Next week"
	if y1, m1, d1 := c.until.Date(); !namesDay && (y1 != now.Year() || m1 != now.Month() || d1 != now.Day()) {
		c.when = c.until.Format("Mon") + " " + c.when
	}
	return c, true
}

// snoozeChoices is the snooze menu for an environment: soonest first, one
// row per wake minute (resolveSnoozePresets).
func snoozeChoices(e *envState, now time.Time) []snoozeChoice {
	presets := defaultSnoozePresets
	if e != nil && e.config != nil && len(e.config.Settings.SnoozePresets) > 0 {
		presets = e.config.Settings.SnoozePresets
	}
	var out []snoozeChoice
	for _, p := range presets {
		if c, ok := resolvePreset(p, now); ok {
			out = append(out, c)
		}
	}
	minute := func(c snoozeChoice) int64 { return c.until.Unix() / 60 }
	sort.SliceStable(out, func(i, j int) bool { return minute(out[i]) < minute(out[j]) })
	seen := map[int64]bool{}
	kept := out[:0]
	for _, c := range out {
		if !seen[minute(c)] {
			seen[minute(c)] = true
			kept = append(kept, c)
		}
	}
	return kept
}

// pickSnooze is a menu choice as picked (pickSnoozePreset): a duration
// counts from now, a set time keeps the wake the menu showed while that's
// still ahead.
func pickSnooze(shown []snoozeChoice, id string, now time.Time) (time.Time, error) {
	for _, c := range shown {
		if c.id != id {
			continue
		}
		if c.preset.Kind == "after" {
			return now.Add(time.Duration(c.preset.Minutes) * time.Minute), nil
		}
		if c.until.After(now) {
			return c.until, nil
		}
	}
	return time.Time{}, errors.New("could not snooze thread: that snooze time has passed, choose another time")
}

func snoozeItems(cs []snoozeChoice, prefix string, off bool) []ui.MenuItem {
	var out []ui.MenuItem
	for _, c := range cs {
		out = append(out, ui.MenuItem{ID: prefix + c.id, Label: c.label, Detail: c.when, Off: off})
	}
	return out
}

// ---------- the row menus ----------

// running reports a thread that can't be archived (threadRuntimeCanArchive).
func running(t *t3.Thread) bool {
	switch threadRuntime(t) {
	case "queued":
		return t.ActiveRunID != ""
	case "preparing", "starting", "running":
		return true
	}
	return false
}

// threadMenu is a row's context menu (buildThreadActionMenuItems). Without
// write access, the items that change the thread are disabled.
func (a *App) threadMenu(e *envState, t *t3.Thread, now time.Time) []ui.MenuItem {
	loaded := e.config != nil
	child := childRequests(e.threads)[t.ID]
	var m []ui.MenuItem
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
		if snoozed(t, child, now) {
			m = append(m, ui.MenuItem{ID: "unsnooze", Label: "Wake thread", Icon: "clock"})
		} else {
			m = append(m, ui.MenuItem{ID: "snooze", Label: "Snooze", Icon: "clock", Off: !canSnooze(t, child, now),
				Children: snoozeItems(a.ctxPresets, "snooze:", false)})
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
	if p, ok := e.projects[t.ProjectID]; ok && p.ArchivedAt == "" {
		k := ref(e.id(), t.ProjectID)
		filter := ui.MenuItem{ID: "filter-by-project", Label: "Filter by " + p.Title, Icon: "folder-tree"}
		if a.scopeSet()[k] {
			filter.Label = "Show all projects"
		}
		m = append(m, filter)
	}
	if loaded && e.config.Environment.Capabilities.ThreadAutoSettleOptOut {
		on := t.AutoSettleDisabledAt == ""
		m = append(m, ui.MenuItem{ID: "auto-settle", Label: "Auto-settle behavior", Icon: "timer", Children: []ui.MenuItem{
			{ID: "auto-settle:enabled", Label: "Follow settings", Radio: true, Checked: on},
			{ID: "auto-settle:disabled", Label: "Never", Radio: true, Checked: !on},
		}})
	}
	copyItems := []ui.MenuItem{{ID: "copy-path", Label: "Path", Icon: "folder"}}
	if t.Branch != "" {
		copyItems = append(copyItems, ui.MenuItem{ID: "copy-branch", Label: "Branch", Icon: "git-branch"})
	}
	copyItems = append(copyItems,
		ui.MenuItem{ID: "copy-thread", Label: "Thread", Icon: "message-square"},
		ui.MenuItem{ID: "copy-thread-id", Label: "Thread ID", Icon: "hash"})
	m = append(m, ui.MenuItem{ID: "copy", Label: "Copy", Icon: "copy", Sep: true, Children: copyItems})
	m = append(m, ui.MenuItem{ID: "archive", Label: "Archive thread", Icon: "archive", Off: running(t), Sep: true})
	del := ui.MenuItem{ID: "delete", Label: "Delete", Icon: "trash", Danger: true}
	if a.ctxConfirm {
		del.Label = "Click again to delete"
	}
	m = append(m, del)
	if !e.operate {
		for i := range m {
			if needsOperate(m[i].ID) {
				m[i].Off = true
				for j := range m[i].Children {
					m[i].Children[j].Off = true
				}
			}
		}
	}
	return m
}

// needsOperate reports the menu items that need write access: all but
// marking unread and copying (threadActionRequiresOperate).
func needsOperate(id string) bool {
	switch id {
	case "mark-unread", "copy", "copy-path", "copy-branch", "copy-thread", "copy-thread-id":
		return false
	}
	return true
}

// selection is the multi-selection's threads, in list order.
func (a *App) selection() []sideThread {
	var out []sideThread
	for _, r := range a.ordered {
		if !a.selected[r] {
			continue
		}
		if e, t := a.thread(r); t != nil {
			out = append(out, sideThread{Thread: t, e: e, ref: r, child: childRequests(e.threads)[t.ID]})
		}
	}
	return out
}

// bulkMenu is the menu for a right-click on a selected row: the actions
// that apply to the whole selection, with counts and no icons.
func (a *App) bulkMenu(now time.Time) []ui.MenuItem {
	sel := a.selection()
	n := len(sel)
	operate := true
	var pinned, settling, regen, regenerable, snoozable int
	for _, t := range sel {
		operate = operate && t.e.operate
		loaded := t.e.config != nil
		if loaded && t.PinnedAt != "" {
			pinned++
		}
		if t.SettledOverride != "settled" {
			settling++
		}
		if loaded {
			regen++
			if t.TitleRegeneration == nil {
				regenerable++
			}
		}
		if loaded && canSnooze(t.Thread, t.child, now) {
			snoozable++
		}
	}
	var m []ui.MenuItem
	if pinned > 0 {
		m = append(m, ui.MenuItem{ID: "bulk-unpin", Label: fmt.Sprintf("Unpin (%d)", pinned), Off: !operate})
	}
	m = append(m, ui.MenuItem{ID: "bulk-settle", Label: fmt.Sprintf("Settle (%d)", n), Off: settling == 0 || !operate})
	if snoozable == n {
		m = append(m, ui.MenuItem{ID: "bulk-snooze", Label: fmt.Sprintf("Snooze (%d)", n), Off: !operate,
			Children: snoozeItems(a.ctxPresets, "bulk-snooze:", !operate)})
	}
	switch {
	case regen > 0 && regenerable == 0:
		m = append(m, ui.MenuItem{ID: "bulk-regenerate-title", Label: fmt.Sprintf("Regenerating… (%d)", regen), Off: true})
	case regen > 0:
		m = append(m, ui.MenuItem{ID: "bulk-regenerate-title", Label: fmt.Sprintf("Regenerate titles (%d)", regenerable), Off: !operate})
	}
	m = append(m, ui.MenuItem{ID: "bulk-mark-unread", Label: fmt.Sprintf("Mark unread (%d)", n)})
	del := ui.MenuItem{ID: "bulk-delete", Label: fmt.Sprintf("Delete (%d)", n), Danger: true, Off: !operate}
	if a.ctxConfirm {
		del.Label = "Click again to delete"
	}
	return append(m, del)
}

// menuScene is the open context menu. Call with the lock held.
func (a *App) menuScene(s *ui.Scene, now time.Time) {
	switch {
	case a.ctxDraft:
		s.Menu = &ui.ContextMenu{Items: draftMenu(), Keyboard: a.ctxKeyboard}
	case a.ctxBulk && len(a.selection()) > 0:
		s.Menu = &ui.ContextMenu{Items: a.bulkMenu(now), Keyboard: a.ctxKeyboard}
	case a.ctxRef != "":
		if e, t := a.thread(a.ctxRef); t != nil {
			s.Menu = &ui.ContextMenu{Items: a.threadMenu(e, t, now), Keyboard: a.ctxKeyboard}
		}
	}
}

// openMenu opens a row's context menu, or the selection's when the row is
// part of it. Call with the lock held.
func (a *App) openMenu(r string, keyboard bool) {
	a.closeMenusLocked()
	a.ctxKeyboard = keyboard
	e, t := a.thread(r)
	if t == nil {
		return
	}
	a.ctxPresets = snoozeChoices(e, time.Now())
	if a.selected[r] && len(a.selected) > 0 {
		a.ctxBulk = true
		return
	}
	a.ctxRef = r
}

// menuPick runs the context menu's item id.
func (a *App) menuPick(th *ui.Theme, id string) {
	a.mu.Lock()
	if a.ctxDraft {
		envID, project := splitRef(a.draftHome)
		root := ""
		if e := a.env(envID); e != nil {
			root = e.projects[project].WorkspaceRoot
		}
		a.closeMenusLocked()
		a.mu.Unlock()
		switch id {
		case "copy-path":
			th.WriteClipboard(root)
		case "discard":
			a.discardDraft(th, "draft")
		}
		return
	}
	if a.ctxBulk {
		a.mu.Unlock()
		a.bulkPick(th, id)
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
	presets, operate, root := a.ctxPresets, e.operate, e.projects[t.ProjectID].WorkspaceRoot
	a.closeMenusLocked()
	a.mu.Unlock()
	if !operate && needsOperate(id) {
		a.failf("Thread action unavailable: this connection cannot change one or more selected threads.")
		return
	}
	verb, arg, _ := strings.Cut(id, ":")
	switch verb {
	case "pin", "unpin", "unsettle", "unsnooze", "mark-unread", "archive", "delete":
		a.threadAction(th, r, verb)
	case "settle":
		a.settle([]string{r})
	case "snooze":
		until, err := pickSnooze(presets, arg, time.Now())
		if err != nil {
			a.failf("%v", err)
			return
		}
		a.snooze([]string{r}, until)
	case "rename":
		a.startRename(r)
	case "regenerate-title":
		a.dispatch(e, map[string]any{"type": "thread.metadata.update", "threadId": t.ID, "regenerateTitle": true})
	case "auto-settle":
		a.dispatch(e, map[string]any{"type": "thread.auto-settle.set", "threadId": t.ID, "enabled": arg == "enabled"})
	case "filter-by-project":
		k := ref(e.id(), t.ProjectID)
		a.mu.Lock()
		if a.scopeSet()[k] {
			a.setScope(nil)
		} else {
			a.setScope([]string{k})
		}
		a.mu.Unlock()
	case "copy-path":
		path := t.WorktreePath
		if path == "" {
			path = root
		}
		if path == "" {
			a.failf("Path unavailable: this thread does not have a workspace path to copy.")
			return
		}
		th.WriteClipboard(path)
	case "copy-branch":
		th.WriteClipboard(t.Branch)
	case "copy-thread":
		th.WriteClipboard(hostedThreadLink(e.id(), t.ID))
	case "copy-thread-id":
		th.WriteClipboard(t.ID)
	}
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

// openPullRequest opens the row's pull request in this computer's browser.
func (a *App) openPullRequest(r string) {
	a.mu.Lock()
	url := ""
	if _, t := a.thread(r); t != nil {
		if b := prBadge(t); b != nil {
			url = b.URL
		}
	}
	open := a.OpenURL
	a.mu.Unlock()
	if url == "" || open == nil {
		return
	}
	if err := open(url); err != nil {
		a.failf("Unable to open pull request link: %v", err)
	}
}

// bulkPick runs an item of the selection's menu, then clears the selection.
func (a *App) bulkPick(th *ui.Theme, id string) {
	a.mu.Lock()
	if id == "bulk-delete" && !a.ctxConfirm {
		a.ctxConfirm = true
		a.mu.Unlock()
		return
	}
	sel := a.selection()
	presets := a.ctxPresets
	a.closeMenusLocked()
	a.selected, a.anchor = map[string]bool{}, ""
	a.mu.Unlock()
	verb, arg, _ := strings.Cut(strings.TrimPrefix(id, "bulk-"), ":")
	var refs []string
	for _, t := range sel {
		switch {
		case verb == "unpin" && t.PinnedAt == "",
			verb == "settle" && t.SettledOverride == "settled",
			verb == "regenerate-title" && t.TitleRegeneration != nil:
			continue
		}
		refs = append(refs, t.ref)
	}
	switch verb {
	case "settle":
		a.settle(refs)
	case "snooze":
		until, err := pickSnooze(presets, arg, time.Now())
		if err != nil {
			a.failf("%v", err)
			return
		}
		a.snooze(refs, until)
	case "regenerate-title":
		for _, r := range refs {
			e, t := a.thread(r)
			a.dispatch(e, map[string]any{"type": "thread.metadata.update", "threadId": t.ID, "regenerateTitle": true})
		}
	default:
		for _, r := range refs {
			a.threadAction(th, r, verb)
		}
	}
}

// ---------- thread actions ----------

// failf shows an error banner.
func (a *App) failf(format string, args ...any) {
	a.mu.Lock()
	a.errText = fmt.Sprintf(format, args...)
	a.mu.Unlock()
	a.changed()
}

// dispatch sends a command to the thread's environment, and reports a
// failure in the banner. done, if set, runs after the server accepts it.
func (a *App) dispatch(e *envState, cmd map[string]any, done ...func()) {
	a.mu.Lock()
	conn := e.conn
	a.mu.Unlock()
	if conn == nil {
		a.failf("%s isn't connected", e.env.Label)
		return
	}
	go func() {
		if err := conn.Dispatch(cmd); err != nil {
			a.failf("%v", err)
			return
		}
		for _, f := range done {
			f()
		}
	}()
}

// threadAction sends one of the simple thread commands for ref. Unpinning
// and archiving, once the server has taken them, join the footer's undo
// notice.
func (a *App) threadAction(th *ui.Theme, r, verb string) {
	a.mu.Lock()
	e, t := a.thread(r)
	var undo []map[string]any
	switch {
	case t == nil:
	case verb == "unpin":
		undo = append([]map[string]any{pinStep(t)}, snoozeStep(t)...)
	case verb == "archive":
		undo = []map[string]any{{"type": "thread.unarchive", "threadId": t.ID}}
	}
	a.mu.Unlock()
	if t == nil {
		return
	}
	cmd := map[string]any{"threadId": t.ID}
	switch verb {
	case "pin", "unpin", "archive", "delete":
		cmd["type"] = "thread." + verb
	case "unsettle", "unsnooze":
		cmd["type"], cmd["reason"] = "thread."+verb, "user"
	case "mark-unread":
		cmd["type"] = "thread.mark-unread"
	default:
		return
	}
	if verb == "archive" || verb == "delete" {
		// It's leaving the list, so close its tab too.
		a.closeTab(th, r)
	}
	if undo == nil {
		a.dispatch(e, cmd)
		return
	}
	word := map[string]string{"unpin": "Unpinned", "archive": "Archived"}[verb]
	a.dispatch(e, cmd, func() { a.noteUndo(word, func(*ui.Theme) { a.chain(e, undo) }) })
}

// advance plans where to go when the open thread is parked
// (planForwardNavigation): the next card row after it, wrapping around,
// past the settled and snoozed rows and the others being parked with it;
// with none, a new thread in its project. nil when the open thread isn't
// being parked. Call with the lock held.
func (a *App) advance(parking []string) func(th *ui.Theme) {
	open := a.openID
	i := -1
	skip := map[string]bool{}
	for _, r := range parking {
		skip[r] = true
	}
	for j, r := range a.ordered {
		if r == open {
			i = j
		}
	}
	if i < 0 || !skip[open] {
		return nil
	}
	order := append(append([]string{}, a.ordered[i+1:]...), a.ordered[:i]...)
	for _, r := range order {
		if !a.parked[r] && !skip[r] {
			return func(th *ui.Theme) { a.switchTo(th, func() { a.openThread(r) }) }
		}
	}
	return func(th *ui.Theme) { a.switchTo(th, a.newDraft) }
}

// park settles or snoozes threads with cmd; if the open one is among
// them, the next thread opens once the server has parked it (§5.15). Each
// park the server takes joins the footer's undo notice, and settling a
// woke thread records the visit it was waiting for (settleThread).
func (a *App) park(refs []string, cmd map[string]any) {
	a.mu.Lock()
	next := a.advance(refs)
	open := a.openID
	settling, now := cmd["type"] == "thread.settle", time.Now()
	type target struct {
		e    *envState
		id   string
		r    string
		undo []map[string]any // what puts it back where it was
		woke string           // when it woke, while that's unseen
	}
	var ts []target
	for _, r := range refs {
		e, t := a.thread(r)
		if t == nil {
			continue
		}
		tg := target{e: e, id: t.ID, r: r, undo: undoPark(t, settling)}
		if child := childRequests(e.threads)[t.ID]; settling && woke(t, child, now) {
			tg.woke = wokeAt(t, child, now)
		}
		ts = append(ts, tg)
	}
	a.mu.Unlock()
	word := "Snoozed"
	if settling {
		word = "Settled"
	}
	for _, t := range ts {
		c := map[string]any{"threadId": t.id}
		for k, v := range cmd {
			c[k] = v
		}
		done := []func(){func() { a.noteUndo(word, func(*ui.Theme) { a.chain(t.e, t.undo) }) }}
		if t.woke != "" {
			done = append(done, func() { a.dispatch(t.e, visit(t.id, t.woke)) })
		}
		if t.r == open && next != nil {
			done = append(done, func() { a.onUI(next) })
		}
		a.dispatch(t.e, c, done...)
	}
}

// visit is the command recording that thread id was seen at the time at.
func visit(id, at string) map[string]any {
	return map[string]any{"type": "thread.visit", "threadId": id, "visitedAt": parseTime(at).UTC().Format(time.RFC3339Nano)}
}

func (a *App) settle(refs []string) { a.park(refs, map[string]any{"type": "thread.settle"}) }

func (a *App) snooze(refs []string, until time.Time) {
	a.park(refs, map[string]any{"type": "thread.snooze", "snoozedUntil": until.UTC().Format(time.RFC3339Nano)})
}

// acknowledgeWoke clears a row's Woke label by recording a visit at the
// moment it woke. It doesn't open the thread.
func (a *App) acknowledgeWoke(r string) {
	a.mu.Lock()
	e, t := a.thread(r)
	var at string
	if t != nil {
		at = wokeAt(t, childRequests(e.threads)[t.ID], time.Now())
	}
	a.mu.Unlock()
	if at == "" {
		return
	}
	a.dispatch(e, visit(t.ID, at))
}

// openDraft opens the new thread's draft where it was left.
func (a *App) openDraft() {
	a.mu.Lock()
	defer a.mu.Unlock()
	envID, id := splitRef(a.draftHome)
	if a.env(envID) == nil || a.draft != "" {
		return
	}
	a.closeThreadLocked()
	a.draftEnv, a.draft = envID, id
}

// draftMenu is the draft row's menu (buildDraftActionMenuItems): copying
// its project's path, and discarding it.
func draftMenu() []ui.MenuItem {
	return []ui.MenuItem{
		{ID: "copy", Label: "Copy", Icon: "copy", Children: []ui.MenuItem{{ID: "copy-path", Label: "Path", Icon: "folder"}}},
		{ID: "discard", Label: "Discard draft", Icon: "trash", Danger: true, Sep: true},
	}
}

// ---------- the undo notice ----------

// undoEntry is a park or a discard the footer's notice can still undo
// (showThreadUndoNotice.ts): what it says, and what puts it back.
type undoEntry struct {
	action string // "Settled", "Snoozed", "Unpinned", "Archived", "Discarded"
	undo   func(th *ui.Theme)
}

// undoLife is how long the notice lasts after the last action joins it.
const undoLife = 5 * time.Second

// noteUndo adds an action to the notice and gives it 5s more, after which
// everything in it stands. Until another key or an edit, mod+Z undoes it
// even in a text field (threadUndoClaimsShortcut).
func (a *App) noteUndo(action string, undo func(th *ui.Theme)) {
	a.mu.Lock()
	a.undos = append(a.undos, undoEntry{action, undo})
	a.undoClaim = true
	if a.undoTimer != nil {
		a.undoTimer.Stop()
	}
	a.undoTimer = time.AfterFunc(undoLife, func() {
		a.mu.Lock()
		a.undos, a.undoClaim = nil, false
		a.mu.Unlock()
		a.changed()
	})
	a.mu.Unlock()
	a.changed()
}

// undoGroup is what one undo puts back: the latest action and those like
// it just before it. Call with the lock held.
func (a *App) undoGroup() []undoEntry {
	n := len(a.undos)
	i := n
	for i > 0 && a.undos[i-1].action == a.undos[n-1].action {
		i--
	}
	return a.undos[i:]
}

// undoSummary is the notice's text: the group's action, and its count when
// there are several, "Settled 3". Call with the lock held.
func (a *App) undoSummary() string {
	switch g := a.undoGroup(); len(g) {
	case 0:
		return ""
	case 1:
		return g[0].action
	default:
		return fmt.Sprintf("%s %d", g[0].action, len(g))
	}
}

// undoLast undoes the notice's group (undoLatestThreadAction), and the
// notice moves on to what's left in it.
func (a *App) undoLast(th *ui.Theme) {
	a.mu.Lock()
	g := a.undoGroup()
	a.undos = a.undos[:len(a.undos)-len(g)]
	a.undoClaim = false
	a.mu.Unlock()
	for _, u := range g {
		u.undo(th)
	}
	a.changed()
}

// undoPark is what undoes settling or snoozing t (planThreadUndo): a
// snooze is woken; a settle is un-settled, then pinned again with its
// order kept and snoozed again, as the settle undid both.
func undoPark(t *t3.Thread, settled bool) []map[string]any {
	if !settled {
		return []map[string]any{{"type": "thread.unsnooze", "threadId": t.ID, "reason": "user"}}
	}
	steps := []map[string]any{{"type": "thread.unsettle", "threadId": t.ID, "reason": "user"}}
	if t.PinnedAt != "" {
		steps = append(steps, pinStep(t))
	}
	return append(steps, snoozeStep(t)...)
}

// pinStep pins t again where it was.
func pinStep(t *t3.Thread) map[string]any {
	cmd := map[string]any{"type": "thread.pin", "threadId": t.ID}
	if t.PinOrderKey != "" {
		cmd["orderKey"] = t.PinOrderKey
	}
	return cmd
}

// snoozeStep snoozes t again until it was due to wake, if it was snoozed:
// pinning wakes a thread.
func snoozeStep(t *t3.Thread) []map[string]any {
	if t.SnoozedUntil == "" {
		return nil
	}
	return []map[string]any{{"type": "thread.snooze", "threadId": t.ID, "snoozedUntil": t.SnoozedUntil}}
}

// chain sends steps to e one after another, each once the server has
// taken the one before. A snooze whose time has come by then is left out,
// as the server would refuse it (executeThreadUndo).
func (a *App) chain(e *envState, steps []map[string]any) {
	for len(steps) > 0 && steps[0]["type"] == "thread.snooze" && !parseTime(steps[0]["snoozedUntil"].(string)).After(time.Now()) {
		steps = steps[1:]
	}
	if len(steps) > 0 {
		a.dispatch(e, steps[0], func() { a.chain(e, steps[1:]) })
	}
}

// discardDraft drops what's written and pasted under key, a thread's or
// the new thread's "draft", and offers it back in the undo notice; it
// comes back unless something new has been written there since
// (discardComposerDraft).
func (a *App) discardDraft(th *ui.Theme, key string) {
	a.mu.Lock()
	open := a.composerKey() == key
	imgs := a.images[key]
	delete(a.images, key)
	a.mu.Unlock()
	text := drafts[key]
	if open {
		text = th.EditorText("composer")
		th.SetEditorText("composer", "")
	}
	saveDraft(key, "")
	a.noteUndo("Discarded", func(th *ui.Theme) {
		a.mu.Lock()
		open := a.composerKey() == key
		cur := drafts[key]
		if open {
			cur = th.EditorText("composer")
		}
		fresh := strings.TrimSpace(cur) != "" || len(a.images[key]) > 0
		if !fresh && len(imgs) > 0 {
			a.images[key] = imgs
		}
		a.mu.Unlock()
		switch {
		case fresh:
			a.failf("Failed to restore draft: the draft has new content.")
		case open:
			th.SetEditorText("composer", text)
		default:
			saveDraft(key, text)
		}
	})
}

// startRename puts the row's title in an editor.
func (a *App) startRename(r string) {
	a.mu.Lock()
	e, t := a.thread(r)
	if t == nil || !e.operate {
		a.mu.Unlock()
		return
	}
	a.renamingRow = r
	title := t.Title
	a.mu.Unlock()
	a.onUI(func(th *ui.Theme) {
		th.SetEditorText("rename", title)
		th.Focus("rename")
	})
}

// commitRename sends the edited title. An empty or unchanged one just ends
// the edit.
func (a *App) commitRename(title string) {
	a.mu.Lock()
	r := a.renamingRow
	a.renamingRow = ""
	e, t := a.thread(r)
	a.mu.Unlock()
	title = strings.TrimSpace(title)
	if t == nil || title == "" || title == t.Title {
		return
	}
	a.dispatch(e, map[string]any{"type": "thread.metadata.update", "threadId": t.ID, "title": title})
}

// rowClick is a click on a row (Sidebar.tsx:1573–1592): it opens the thread
// and clears the selection; with the shortcut key it toggles the row's
// selection, with Shift it selects the range from the last row clicked. A
// double click renames instead of opening the thread again.
func (a *App) rowClick(th *ui.Theme, act ui.Action) {
	r := strings.TrimPrefix(act.ID, "thread:")
	a.mu.Lock()
	switch {
	case r == a.renamingRow:
		// The click is in its title's editor.
		a.mu.Unlock()
		return
	case act.Clicks > 1:
		a.mu.Unlock()
		if !act.Short && !act.Shift {
			a.startRename(r)
		}
		return
	case act.Short:
		if a.selected[r] {
			delete(a.selected, r)
		} else {
			a.selected[r] = true
		}
		a.anchor = r
		a.mu.Unlock()
		return
	case act.Shift && a.anchor != "":
		from, to := -1, -1
		for i, k := range a.ordered {
			if k == a.anchor {
				from = i
			}
			if k == r {
				to = i
			}
		}
		if from >= 0 && to >= 0 {
			a.selected = map[string]bool{}
			for _, k := range a.ordered[min(from, to) : max(from, to)+1] {
				a.selected[k] = true
			}
			a.mu.Unlock()
			return
		}
	}
	a.selected, a.anchor = map[string]bool{}, r
	a.mu.Unlock()
	a.switchTo(th, func() { a.openThread(r) })
}

// step opens the thread before or after the open thread's row in the list
// (thread.previous, thread.next), not wrapping; with none open, the last or
// the first.
func (a *App) step(th *ui.Theme, dir int) {
	a.mu.Lock()
	ordered, open := a.ordered, a.rowThread(a.openID)
	a.mu.Unlock()
	if len(ordered) == 0 {
		return
	}
	next := ""
	i := -1
	for j, r := range ordered {
		if r == open {
			i = j
		}
	}
	switch {
	case open == "" && dir < 0:
		next = ordered[len(ordered)-1]
	case open == "":
		next = ordered[0]
	case i >= 0 && i+dir >= 0 && i+dir < len(ordered):
		next = ordered[i+dir]
	}
	if next != "" {
		a.switchTo(th, func() { a.openThread(next) })
	}
}

// jump opens the nth thread in the list (thread.jump.1…9).
func (a *App) jump(th *ui.Theme, n int) {
	a.mu.Lock()
	ordered := a.ordered
	a.mu.Unlock()
	if n >= 1 && n <= len(ordered) {
		r := ordered[n-1]
		a.switchTo(th, func() { a.openThread(r) })
	}
}

// openThreadShortcut runs thread.settle and thread.pin on the open thread:
// settle (and move on) or un-settle, pin or unpin.
func (a *App) openThreadShortcut(th *ui.Theme, what string) {
	a.mu.Lock()
	r := a.openID
	e, t := a.thread(r)
	a.mu.Unlock()
	if t == nil || e.config == nil {
		return
	}
	switch {
	case what == "settle" && t.SettledOverride == "settled":
		a.threadAction(th, r, "unsettle")
	case what == "settle":
		a.settle([]string{r})
	case t.PinnedAt != "":
		a.threadAction(th, r, "unpin")
	default:
		a.threadAction(th, r, "pin")
	}
}

// ---------- input ----------

// sidebarAction handles the sidebar's input, and reports whether act was
// the sidebar's.
func (a *App) sidebarAction(th *ui.Theme, act ui.Action) bool {
	id := act.ID
	verb, arg, _ := strings.Cut(id, ":")
	if act.Kind == "change" || act.Kind == "key" {
		// Any other key or an edit hands mod+Z back to text fields.
		a.mu.Lock()
		a.undoClaim = false
		a.mu.Unlock()
	}
	switch {
	case id == "search":
		a.searchInput(th, act)
		return true
	case id == "rename":
		switch act.Kind {
		case "submit", "blur":
			a.commitRename(th.EditorText("rename"))
		case "key":
			a.mu.Lock()
			a.renamingRow = ""
			a.mu.Unlock()
		}
		return true
	case id == "filter.search":
		switch act.Kind {
		case "change":
			a.mu.Lock()
			a.filterQuery, a.filterHi = act.Text, 0
			a.mu.Unlock()
		case "key":
			a.filterKey(act)
		case "submit":
			a.filterKey(ui.Action{Text: "⏎"})
		}
		return true
	case verb == "filter.project" && act.Kind == "hover":
		a.mu.Lock()
		for i, k := range a.filterShown {
			if k == arg {
				a.filterHi = i
			}
		}
		a.mu.Unlock()
		return true
	case verb == "result" && act.Kind == "hover":
		a.mu.Lock()
		a.searchSel = atoi(arg)
		a.mu.Unlock()
		return true
	case verb == "thread" && act.Kind == "context":
		a.mu.Lock()
		a.openMenu(arg, act.Text == "keyboard")
		a.mu.Unlock()
		return true
	case verb == "draft" && act.Kind == "context":
		a.mu.Lock()
		a.closeMenusLocked()
		a.ctxDraft, a.ctxKeyboard = true, act.Text == "keyboard"
		a.mu.Unlock()
		return true
	case act.Kind != "click":
		return false
	}
	switch verb {
	case "thread":
		a.rowClick(th, act)
	case "result":
		a.openResult(th, atoi(arg))
	case "shelf":
		a.mu.Lock()
		a.shelfOpen[arg] = !a.shelfOpen[arg]
		open := a.shelfOpen
		a.mu.Unlock()
		_ = a.store.Put("sidebar-shelves", open)
	case "settled.more":
		a.mu.Lock()
		a.settledMore++
		a.mu.Unlock()
	case "settle":
		a.settle([]string{arg})
	case "unsettle", "unpin":
		a.threadAction(th, arg, verb)
	case "wake":
		a.threadAction(th, arg, "unsnooze")
	case "woke":
		a.acknowledgeWoke(arg)
	case "pr":
		a.openPullRequest(arg)
	case "discard":
		a.discardDraft(th, arg)
	case "draft":
		a.switchTo(th, a.openDraft)
	case "undo", "thread.undo":
		a.undoLast(th)
	case "terminal":
		a.switchTo(th, func() { a.openThread(arg) })
		a.mu.Lock()
		if d := a.drawers[arg]; d != nil {
			d.open = true
		}
		a.mu.Unlock()
	case "snooze":
		a.mu.Lock()
		open := a.snoozeRef != arg
		a.closeMenusLocked()
		if open {
			e, _ := a.thread(arg)
			a.snoozeRef, a.snoozeMenu = arg, snoozeChoices(e, time.Now())
		}
		a.mu.Unlock()
	case "snooze.pick":
		a.mu.Lock()
		r, shown := a.snoozeRef, a.snoozeMenu
		a.closeMenusLocked()
		a.mu.Unlock()
		until, err := pickSnooze(shown, arg, time.Now())
		if err != nil {
			a.failf("%v", err)
			return true
		}
		a.snooze([]string{r}, until)
	case "ctx":
		a.menuPick(th, arg)
	case "rename":
		a.startRename(arg)
	case "filter":
		a.mu.Lock()
		open := !a.filterOpen
		a.closeMenusLocked()
		a.filterOpen, a.filterQuery, a.filterHi = open, "", -1
		if open {
			a.filterKeys = a.filterOrder()
		}
		a.mu.Unlock()
		if open {
			th.SetEditorText("filter.search", "")
			th.Focus("filter.search")
		}
	case "filter.project":
		a.pickProject(arg, act.Shift)
	case "filter.clear":
		a.mu.Lock()
		a.setScope(nil)
		a.mu.Unlock()
	case "search.clear":
		th.SetEditorText("search", "")
		th.Focus("search")
		a.mu.Lock()
		a.search, a.searchSel = "", 0
		a.mu.Unlock()
	case "thread.prev":
		a.step(th, -1)
	case "thread.next":
		a.step(th, 1)
	case "thread.jump":
		a.jump(th, atoi(arg))
	case "thread.settle", "thread.pin":
		a.openThreadShortcut(th, strings.TrimPrefix(verb, "thread."))
	default:
		return false
	}
	return true
}

// searchInput is the search field's typing and keys: the arrows move the
// highlight through the results, wrapping; Enter opens it; Escape clears
// the search.
func (a *App) searchInput(th *ui.Theme, act ui.Action) {
	a.mu.Lock()
	n := len(a.results)
	switch {
	case act.Kind == "change":
		a.search, a.searchSel = act.Text, 0
	case act.Kind == "key" && act.Text == "⎋":
		a.search, a.searchSel = "", 0
		a.mu.Unlock()
		th.SetEditorText("search", "")
		return
	case act.Kind == "key" && n > 0 && (act.Text == "↓" || act.Text == "↑"):
		step := 1
		if act.Text == "↑" {
			step = -1
		}
		a.searchSel = (a.searchSel + step + n) % n
	case act.Kind == "submit":
		i := a.searchSel
		a.mu.Unlock()
		a.openResult(th, i)
		return
	}
	a.mu.Unlock()
}

// openResult opens search result i.
func (a *App) openResult(th *ui.Theme, i int) {
	a.mu.Lock()
	r := ""
	if i >= 0 && i < len(a.results) {
		r = a.results[i]
	}
	a.mu.Unlock()
	if r != "" {
		a.switchTo(th, func() { a.openThread(r) })
	}
}
