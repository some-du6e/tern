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

// A thread's actions menu (threadActionMenu.logic.ts, useThreadActionMenu.ts;
// docs/ui/05-header.md §2.3, §5.2), which the header's title, a right-click
// on the header and one on a sidebar row open; and the copy menus a
// right-click on the details card's run context and branch rows open. All
// are T3's in-page menu.

// threadMenu is thread t's actions menu (buildThreadActionMenuItems), the
// items tern can act on in T3's order. T3 offers Find in thread, Share
// with guest, Turn into project and Project settings only with their
// surfaces, and Snooze's Custom… with its dialog, which tern doesn't have
// yet, so they're left out the way T3 leaves out what a client or server
// doesn't offer. Archive waits until the turn stops.
func (a *App) threadMenu(e *envState, t *t3.Thread, now time.Time) []ui.MenuItem {
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
		if parseTime(t.SnoozedUntil).After(now) {
			m = append(m, ui.MenuItem{ID: "unsnooze", Label: "Wake thread", Icon: "clock"})
		} else {
			var presets []ui.MenuItem
			for _, p := range snoozePresets(now, e.config.Settings.SnoozePresets) {
				presets = append(presets, ui.MenuItem{ID: "snooze:" + p.id, Label: p.label, Detail: p.when})
			}
			m = append(m, ui.MenuItem{ID: "snooze", Label: "Snooze", Icon: "clock", Off: !canSnooze(t), Children: presets})
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
	return append(m, ui.MenuItem{ID: "delete", Label: "Delete", Icon: "trash", Danger: true})
}

// canSnooze reports whether thread t may be snoozed now (canSnooze): not
// while it waits on an approval or another request that isn't a question,
// which would wake it at once.
func canSnooze(t *t3.Thread) bool {
	r := t.PendingRuntimeRequest
	return r == nil || r.Kind == "user_input"
}

// snoozePreset is one of the snooze menu's rows (SnoozePreset): its id,
// its label, and when it wakes the thread, as a time and as the row's
// detail.
type snoozePreset struct {
	id, label, when string
	until           time.Time
}

// defaultSnoozes are T3's snooze times (DEFAULT_SNOOZE_PRESETS): in an
// hour and in three, this evening at 18:00, tomorrow and next Monday at
// 9:00.
var defaultSnoozes = []t3.SnoozePreset{
	{Kind: "after", Minutes: 60}, {Kind: "after", Minutes: 180},
	{Kind: "at", Day: "today", Time: "18:00"}, {Kind: "at", Day: "tomorrow", Time: "09:00"}, {Kind: "at", Day: "monday", Time: "09:00"},
}

// snoozePresets is the snooze menu at now (resolveSnoozePresets): the
// server's times, else T3's, soonest first and one a minute, a time today
// that has passed left out.
func snoozePresets(now time.Time, settings []t3.SnoozePreset) []snoozePreset {
	if settings == nil {
		settings = defaultSnoozes
	}
	var out []snoozePreset
	for _, s := range settings {
		if p, ok := resolveSnooze(s, now); ok {
			out = append(out, p)
		}
	}
	minute := func(p snoozePreset) int64 { return p.until.Unix() / 60 }
	sort.SliceStable(out, func(i, j int) bool { return minute(out[i]) < minute(out[j]) })
	seen := map[int64]bool{}
	kept := out[:0]
	for _, p := range out {
		if !seen[minute(p)] {
			seen[minute(p)] = true
			kept = append(kept, p)
		}
	}
	return kept
}

// resolveSnooze is setting s at now (resolveSnoozePreset): "In 3 hours"
// from now, or "This evening", "Tomorrow", "Next week" or a weekday at its
// time; its detail is the time, with the day before it when a duration
// or Next week wakes on another day than today.
func resolveSnooze(s t3.SnoozePreset, now time.Time) (snoozePreset, bool) {
	var wake time.Time
	days := 0
	label := ""
	switch s.Kind {
	case "after":
		wake = now.Add(time.Duration(s.Minutes) * time.Minute)
		label = "In " + snoozeDuration(s.Minutes)
	case "at":
		var hour, min int
		if _, err := fmt.Sscanf(s.Time, "%d:%d", &hour, &min); err != nil {
			return snoozePreset{}, false
		}
		switch s.Day {
		case "today":
		case "tomorrow":
			days = 1
		default:
			wd, ok := weekdays[s.Day]
			if !ok {
				return snoozePreset{}, false
			}
			if days = (int(wd) - int(now.Weekday()) + 7) % 7; days == 0 {
				days = 7
			}
		}
		y, mo, d := now.Date()
		wake = time.Date(y, mo, d+days, hour, min, 0, 0, now.Location())
		if !wake.After(now) {
			return snoozePreset{}, false
		}
		switch {
		case days == 0 && hour < 12:
			label = "This morning"
		case days == 0 && hour < 17:
			label = "This afternoon"
		case days == 0:
			label = "This evening"
		case days == 1:
			label = "Tomorrow"
		case s.Day == "monday":
			label = "Next week"
		case days == 7:
			label = "Next " + strings.ToUpper(s.Day[:1]) + s.Day[1:]
		default:
			label = strings.ToUpper(s.Day[:1]) + s.Day[1:]
		}
	default:
		return snoozePreset{}, false
	}
	id := "after:" + strconv.Itoa(s.Minutes)
	if s.Kind == "at" {
		id = "at:" + s.Day + ":" + s.Time
	}
	when := wake.Format("3:04 PM")
	namesDay := s.Kind == "at" && label != "Next week"
	if y1, m1, d1 := wake.Date(); !namesDay && (y1 != now.Year() || m1 != now.Month() || d1 != now.Day()) {
		when = wake.Format("Mon") + " " + when
	}
	return snoozePreset{id: id, label: label, when: when, until: wake}, true
}

// weekdays are T3's preset days.
var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
	"thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
}

// snoozeDuration is a preset's minutes in the largest unit that divides
// them: "3 hours", "90 minutes", "2 days".
func snoozeDuration(minutes int) string {
	n, unit := minutes, "minute"
	switch {
	case minutes%(24*60) == 0:
		n, unit = minutes/(24*60), "day"
	case minutes%60 == 0:
		n, unit = minutes/60, "hour"
	}
	if n != 1 {
		unit += "s"
	}
	return strconv.Itoa(n) + " " + unit
}

// menuScene is the open in-page menu: a thread's actions, or the card's
// "Copy full path" or "Copy branch name", each only with something to
// copy. Call with the lock held.
func (a *App) menuScene(s *ui.Scene, now time.Time) {
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
		s.Menu = &ui.ContextMenu{Items: a.threadMenu(e, t, now)}
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
	a.closeMenusLocked()
	if t == nil {
		a.mu.Unlock()
		return
	}
	if id == "delete" {
		// The menu closes, and the app's confirm dialog asks first.
		a.deleteAsk = r
		a.mu.Unlock()
		return
	}
	root := e.projects[t.ProjectID].WorkspaceRoot
	conn := e.conn
	var snoozes []t3.SnoozePreset
	if e.config != nil {
		snoozes = e.config.Settings.SnoozePresets
	}
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
	case "pin", "unpin", "settle", "archive":
		cmd["type"] = "thread." + verb
	case "unsettle", "unsnooze":
		cmd["type"], cmd["reason"] = "thread."+verb, "user"
	case "snooze":
		// The time as it is when it's picked (pickSnoozePreset): a
		// duration from now.
		var until time.Time
		for _, p := range snoozePresets(time.Now(), snoozes) {
			if p.id == arg {
				until = p.until
			}
		}
		if until.IsZero() {
			a.failf("Could not snooze thread", errors.New("That snooze time has passed. Choose another time."))
			return
		}
		cmd["type"], cmd["snoozedUntil"] = "thread.snooze", until.UTC().Format(time.RFC3339Nano)
	case "mark-unread":
		cmd["type"] = "thread.mark-unread"
	case "regenerate-title":
		cmd["type"], cmd["regenerateTitle"] = "thread.metadata.update", true
	case "auto-settle":
		cmd["type"], cmd["enabled"] = "thread.auto-settle.set", arg == "enabled"
	default:
		return
	}
	a.dispatchThread(th, e, r, conn, cmd)
}

// dispatchThread sends cmd for thread r in e, closing its tab first when
// it leaves the inbox (archive, delete).
func (a *App) dispatchThread(th *ui.Theme, e *envState, r string, conn *t3.Conn, cmd map[string]any) {
	if conn == nil {
		a.failf("Thread action unavailable", fmt.Errorf("%s isn't connected", e.env.Label))
		return
	}
	if cmd["type"] == "thread.archive" || cmd["type"] == "thread.delete" {
		a.closeTab(th, r)
	}
	go func() {
		if err := conn.Dispatch(cmd); err != nil {
			a.failf("Thread action failed", err)
		}
	}()
}

// deleteAlert is the confirm dialog Delete opens (confirmThreadDelete,
// on by default; ConfirmDialogHost.tsx): the question as its title, the
// rest of T3's message under it, and a destructive Confirm. Call with the
// lock held.
func (a *App) deleteAlert() *ui.Alert {
	_, t := a.thread(a.deleteAsk)
	if t == nil {
		return nil
	}
	return &ui.Alert{Title: `Delete thread "` + t.Title + `"?`, Description: "This permanently clears conversation history for this thread.",
		Action: "Confirm", Variant: ui.BtnDestructive}
}

// confirmDelete answers the delete dialog: ok deletes the thread.
func (a *App) confirmDelete(th *ui.Theme, ok bool) {
	a.mu.Lock()
	r := a.deleteAsk
	a.deleteAsk = ""
	e, t := a.thread(r)
	var conn *t3.Conn
	if e != nil {
		conn = e.conn
	}
	a.mu.Unlock()
	if ok && t != nil {
		a.dispatchThread(th, e, r, conn, map[string]any{"type": "thread.delete", "threadId": t.ID})
	}
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
