package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The detailed pull request row (ThreadDetailsPrRow.tsx,
// pullRequestDetail.logic.ts, pullRequestPresentation.tsx,
// docs/ui/05-header.md §3.8, §4.6): each pull request row reads its detail
// and checks from the host through the server, and once there's something
// to say under the title, says it, with the action the pull request is
// ready for. Merging and marking ready go through pullRequests.runAction,
// as T3's do.

// prLoad is what the card has read of one pull request: its detail and its
// checks, when each last came, and the action on its way.
type prLoad struct {
	detail      *t3.PullRequestDetail
	checks      *t3.PullRequestChecks
	checksNewer bool // the checks came after the detail
	noChecks    bool // the host reports none: no more asking
	loading     bool
	checking    bool
	loadedAt    time.Time
	checkedAt   time.Time
	acting      string // merge or ready
}

// prAlert is the merge waiting on its dialog.
type prAlert struct {
	envID  string
	key    string
	ref    t3.PullRequestRef
	number int
	method string
	terms  changeTerms
}

// prKeyOf is a pull request's key in the card's reads: its environment,
// host, repository and number.
func prKeyOf(envID string, ref t3.PullRequestRef) string {
	return envID + "\x00" + strings.ToLower(ref.Host) + "\x00" + strings.ToLower(ref.Repository) + "#" + strconv.Itoa(ref.Number)
}

// repositorySelector is how a host names a project's repository
// (sourceControlClients.repositorySelector): "owner/name", Azure DevOps'
// by its name alone.
func repositorySelector(id *t3.RepositoryIdentity) string {
	switch {
	case id == nil || id.Name == "":
		return ""
	case id.Provider == "azure-devops":
		return id.Name
	case id.Owner == "":
		return ""
	}
	return id.Owner + "/" + id.Name
}

// refInput is ref as the pullRequests.* calls take it, with more fields;
// the branch's own pull request has no host to send.
func refInput(ref t3.PullRequestRef, more map[string]any) map[string]any {
	in := map[string]any{"projectId": ref.ProjectID, "repository": ref.Repository, "number": ref.Number}
	if ref.Host != "" {
		in["host"] = ref.Host
	}
	for k, v := range more {
		in[k] = v
	}
	return in
}

// hostKinds are the hosts T3 knows by their public domain.
var hostKinds = map[string]string{"github.com": "github", "gitlab.com": "gitlab", "bitbucket.org": "bitbucket", "dev.azure.com": "azure-devops"}

// projectFor is the project of e whose checkout is of a link's repository
// (findProjectForChangeRequest), or with lend, any on its host
// (findProjectOnChangeRequestHost); nil for none. They're tried in id
// order. Call with the lock held.
func projectFor(e *envState, l *t3.PullRequestLink, lend bool) *t3.Project {
	ids := make([]string, 0, len(e.projects))
	for id := range e.projects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, pass := range []bool{false, true} {
		if pass && !lend {
			break
		}
		for _, id := range ids {
			p := e.projects[id]
			ident := p.RepositoryIdentity
			switch {
			case ident == nil || ident.Provider == "":
			case !pass && strings.EqualFold(repositorySelector(ident), l.Repository):
				return &p
			case pass && hostKinds[strings.ToLower(l.Host)] == ident.Provider:
				return &p
			}
		}
	}
	return nil
}

// prRefOf is the reference row it of thread t's card reads its pull
// request by (ThreadDetailsPrRow.tsx), nil without one: a link's own host,
// repository and number, in the project checked out from that repository,
// for the shown row else the thread's own, for the others any on its
// host; the branch's pull request by number in the thread's project's
// repository. Call with the lock held.
func prRefOf(e *envState, p t3.Project, it prItem, shown bool) *t3.PullRequestRef {
	if e.config == nil {
		return nil
	}
	if l := it.link; l != nil {
		q := projectFor(e, l, !shown)
		switch {
		case shown && (q == nil || strings.EqualFold(repositorySelector(p.RepositoryIdentity), l.Repository)):
			q = &p
		case q == nil:
			return nil
		}
		return &t3.PullRequestRef{ProjectID: q.ID, Host: l.Host, Repository: l.Repository, Number: l.Number}
	}
	if repo := repositorySelector(p.RepositoryIdentity); repo != "" {
		return &t3.PullRequestRef{ProjectID: p.ID, Repository: repo, Number: it.number}
	}
	return nil
}

// checksInterval is how often running checks are read again
// (pullRequestChecksRefreshIntervalMs): 45s while any waits, or none are
// reported yet, else a minute.
func checksInterval(checks []t3.PullRequestCheck) time.Duration {
	if len(checks) == 0 {
		return 45 * time.Second
	}
	for _, c := range checks {
		if c.Status == "pending" || c.Status == "action-required" {
			return 45 * time.Second
		}
	}
	return time.Minute
}

// prDetailEvery is how often an open pull request's detail is read again.
const prDetailEvery = 10 * time.Minute

// loadPRLocked has the card's read of the pull request at ref under key
// up to date, as T3's queries keep it: the detail at once, again every ten
// minutes while open; then the checks, again while they run, when the
// server can read them alone. Each answer asks for a frame when the next
// read is due. Call with the lock held.
func (a *App) loadPRLocked(e *envState, key string, ref t3.PullRequestRef, now time.Time) *prLoad {
	l := a.prLoads[key]
	if l == nil {
		l = &prLoad{}
		a.prLoads[key] = l
	}
	conn := e.conn
	if conn == nil {
		return l
	}
	open := l.detail == nil || l.detail.State == "open"
	if !l.loading && (l.loadedAt.IsZero() || (open && now.Sub(l.loadedAt) >= prDetailEvery)) {
		l.loading = true
		go a.readPRDetail(conn, l, ref)
	}
	checks := e.config != nil && e.config.Environment.Capabilities.PullRequestChecks
	if d := withNewerChecks(l); checks && d != nil && !l.loading && !l.checking &&
		(l.checkedAt.IsZero() || (d.State == "open" && !l.noChecks && now.Sub(l.checkedAt) >= checksInterval(withNewerChecks(l).Checks))) {
		l.checking = true
		go a.readPRChecks(conn, l, ref)
	}
	return l
}

// readPRDetail reads a pull request's detail (pullRequests.detail, fresh
// from the host). A failure keeps what was read before, and the row its
// simple form until the next read.
func (a *App) readPRDetail(conn *t3.Conn, l *prLoad, ref t3.PullRequestRef) {
	var d t3.PullRequestDetail
	err := conn.Call("pullRequests.detail", refInput(ref, map[string]any{"allowStale": false}), &d)
	a.mu.Lock()
	l.loading, l.loadedAt = false, time.Now()
	if err == nil {
		l.detail, l.checksNewer = &d, false
	}
	open := l.detail == nil || l.detail.State == "open"
	a.mu.Unlock()
	if open {
		time.AfterFunc(prDetailEvery, a.changed)
	}
	a.changed()
}

// readPRChecks reads a pull request's checks alone (pullRequests.checks);
// null means the host reports none.
func (a *App) readPRChecks(conn *t3.Conn, l *prLoad, ref t3.PullRequestRef) {
	var raw json.RawMessage
	err := conn.Call("pullRequests.checks", refInput(ref, nil), &raw)
	var c *t3.PullRequestChecks
	if err == nil && len(raw) > 0 {
		err = json.Unmarshal(raw, &c)
	}
	a.mu.Lock()
	l.checking, l.checkedAt = false, time.Now()
	switch {
	case err != nil:
	case c == nil:
		l.noChecks = true
	default:
		l.checks, l.checksNewer = c, true
	}
	again := checksInterval(withNewerChecks(l).Checks)
	a.mu.Unlock()
	time.AfterFunc(again, a.changed)
	a.changed()
}

// withNewerChecks is l's detail with whichever checks were read last
// (resolvePullRequestDetailChecks): by the host's own times when both
// have one, else by which came after. Nil before the detail.
func withNewerChecks(l *prLoad) *t3.PullRequestDetail {
	d, c := l.detail, l.checks
	if d == nil || c == nil {
		return d
	}
	newer := l.checksNewer
	if d.ObservedAt != nil && c.ObservedAt != nil {
		newer = *c.ObservedAt >= *d.ObservedAt
	}
	if !newer {
		return d
	}
	out := *d
	out.Checks = c.Checks
	if d.State != "merged" {
		out.State = c.State
	}
	return &out
}

// ---------- T3's rules for checks and actions ----------

// checkFailed reports a check that failed or was cancelled.
func checkFailed(c t3.PullRequestCheck) bool {
	return c.Status == "failure" || c.Status == "cancelled"
}

// checkWaiting reports a check still running or waiting on someone.
func checkWaiting(c t3.PullRequestCheck) bool {
	return c.Status == "pending" || c.Status == "action-required"
}

// countChecks is how many checks f picks.
func countChecks(checks []t3.PullRequestCheck, f func(t3.PullRequestCheck) bool) int {
	n := 0
	for _, c := range checks {
		if f(c) {
			n++
		}
	}
	return n
}

// checksRollup is what a pull request's checks come to
// (pullRequestChecksState): failing, pending, passing, or "" for none to
// speak of.
func checksRollup(checks []t3.PullRequestCheck) string {
	switch {
	case len(checks) == 0:
		return ""
	case countChecks(checks, checkFailed) > 0:
		return "failing"
	case countChecks(checks, checkWaiting) > 0:
		return "pending"
	case countChecks(checks, func(c t3.PullRequestCheck) bool { return c.Status == "success" }) > 0:
		return "passing"
	}
	return ""
}

// classifyChecks is classifyPullRequestChecks: none, failing, pending or
// passing, where skipped and neutral checks pass.
func classifyChecks(checks []t3.PullRequestCheck) string {
	switch {
	case len(checks) == 0:
		return "none"
	case countChecks(checks, checkFailed) > 0:
		return "failing"
	case countChecks(checks, checkWaiting) > 0:
		return "pending"
	}
	return "passing"
}

// workflowApproval reports whether an action-required check is a workflow
// run waiting on approval (isWorkflowApprovalCheck).
func workflowApproval(c t3.PullRequestCheck) bool {
	if c.Status != "action-required" {
		return false
	}
	_, rest, ok := strings.Cut(c.URL, "/actions/runs/")
	if !ok || rest == "" || rest[0] < '0' || rest[0] > '9' {
		return false
	}
	end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	return end < 0 || rest[end] == '/'
}

// checksSummary is the popover's line under its heading
// (summarizePullRequestChecks).
func checksSummary(checks []t3.PullRequestCheck) string {
	n := len(checks)
	if n == 0 {
		return "No checks reported"
	}
	approvals := countChecks(checks, workflowApproval)
	others := countChecks(checks, func(c t3.PullRequestCheck) bool { return c.Status == "action-required" }) - approvals
	switch failed, pending, passed := countChecks(checks, checkFailed), countChecks(checks, func(c t3.PullRequestCheck) bool { return c.Status == "pending" }),
		countChecks(checks, func(c t3.PullRequestCheck) bool { return c.Status == "success" }); {
	case failed > 0:
		return fmt.Sprintf("%d of %d failing", failed, n)
	case approvals > 0 && others > 0:
		return fmt.Sprintf("%d %s and %d %s awaiting action", approvals, plural(approvals, "workflow", "workflows"), others, plural(others, "check", "checks"))
	case approvals > 0:
		return fmt.Sprintf("%d %s awaiting approval", approvals, plural(approvals, "workflow", "workflows"))
	case others > 0:
		return fmt.Sprintf("%d %s awaiting action", others, plural(others, "check", "checks"))
	case pending > 0:
		return fmt.Sprintf("%d of %d running", pending, n)
	case passed == n:
		return "All checks passed"
	default:
		return fmt.Sprintf("%d of %d passing", passed, n)
	}
}

// checksCount is the meta line's count: how many have finished while
// some haven't, else how many failed, else nothing.
func checksCount(checks []t3.PullRequestCheck) string {
	waiting, failed := countChecks(checks, checkWaiting), countChecks(checks, checkFailed)
	switch {
	case waiting > 0:
		return fmt.Sprintf("%d/%d checks", len(checks)-waiting, len(checks))
	case failed > 0:
		return fmt.Sprintf("%d failed", failed)
	}
	return ""
}

// checkLabels are the checks' statuses as the popover names them.
var checkLabels = map[string]string{"pending": "Running", "action-required": "Awaiting action", "success": "Passed",
	"failure": "Failed", "cancelled": "Cancelled", "skipped": "Skipped", "neutral": "Neutral"}

// checksHeadings are the popover's headings by rollup.
var checksHeadings = map[string]string{"passing": "All checks have passed", "failing": "Some checks were not successful",
	"pending": "Some checks haven't completed yet"}

// popupChecks are the checks the popover lists (ChecksBody): those that
// want attention, then the running ones, then the finished, which fold
// behind "Show all" while any of the first two are there.
func popupChecks(checks []t3.PullRequestCheck, all bool) (shown []t3.PullRequestCheck, more string) {
	var attention, running, done []t3.PullRequestCheck
	for _, c := range checks {
		switch c.Status {
		case "failure", "cancelled", "action-required":
			attention = append(attention, c)
		case "pending":
			running = append(running, c)
		default:
			done = append(done, c)
		}
	}
	fold := len(attention)+len(running) > 0 && len(done) > 0
	shown = append(attention, running...)
	if all || !fold {
		shown = append(shown, done...)
	}
	switch {
	case fold && all:
		more = "Show less"
	case fold:
		more = "Show all"
	}
	return shown, more
}

// canAct reports whether the host offers action and the viewer may take it.
func canAct(d *t3.PullRequestDetail, action string) bool {
	has := func(list []string) bool {
		for _, x := range list {
			if x == action {
				return true
			}
		}
		return false
	}
	return has(d.Capabilities.Actions) && has(d.ViewerPermissions.Actions)
}

// mergeMethods are the ways the host would merge it, in its order
// (allowedPullRequestMergeMethods).
func mergeMethods(d *t3.PullRequestDetail) []string {
	var out []string
	for _, m := range d.Capabilities.MergeMethods {
		if d.MergeCapabilities[m] {
			out = append(out, m)
		}
	}
	return out
}

// mergeMethod is the method Merge uses (resolvePullRequestMergeMethod):
// tern has no merge picker, so T3's remembered choice is its default,
// merge, when the host allows it, else the host's first.
func mergeMethod(d *t3.PullRequestDetail) string {
	ms := mergeMethods(d)
	for _, m := range ms {
		if m == "merge" {
			return m
		}
	}
	if len(ms) > 0 {
		return ms[0]
	}
	return "merge"
}

// panelAction is the action a row offers (resolveThreadPanelPullRequestAction):
// resolve a conflict, mark a draft ready, fix failing checks, or merge
// once checks pass and the host allows a method; "" for none.
func panelAction(d *t3.PullRequestDetail) string {
	switch {
	case d == nil || d.State != "open":
		return ""
	case d.Mergeability == "conflicting":
		return "resolve"
	case d.IsDraft && canAct(d, "ready"):
		return "ready"
	case d.IsDraft:
		return ""
	}
	switch classifyChecks(d.Checks) {
	case "failing":
		return "fix"
	case "pending":
		return ""
	}
	if canAct(d, "merge") && len(mergeMethods(d)) > 0 {
		return "merge"
	}
	return ""
}

// termsOf is a host's words for a change request by its kind.
func termsOf(kind string) changeTerms {
	if kind == "gitlab" {
		return changeTerms{"MR", "merge request"}
	}
	return changeTerms{"PR", "pull request"}
}

// detailRow is a loaded pull request's detailed form, nil when there's
// nothing to say under its title: no checks to show, no action and no
// watch. Fix and Resolve hand the work to a thread; tern doesn't start
// those yet, so it offers Merge and Ready only. Call with the lock held.
func (a *App) detailRow(key string, l *prLoad, number int, watching bool) *ui.PullRequestDetailRow {
	d := withNewerChecks(l)
	if d == nil {
		return nil
	}
	terms := termsOf(d.Provider)
	conflicting := d.State == "open" && d.Mergeability == "conflicting"
	rollup := checksRollup(d.Checks)
	row := &ui.PullRequestDetailRow{Title: d.Title, Number: number}
	if rollup != "" && !conflicting && !d.IsDraft {
		row.Checks = &ui.ChecksMeta{State: rollup, Count: checksCount(d.Checks)}
		if a.checksOpen == key {
			shown, more := popupChecks(d.Checks, a.checksAll)
			pop := &ui.ChecksPopup{Heading: checksHeadings[rollup], Summary: checksSummary(d.Checks), More: more}
			for _, c := range shown {
				label := checkLabels[c.Status]
				if workflowApproval(c) {
					label = "Awaiting approval"
				}
				pop.Checks = append(pop.Checks, ui.CheckRow{Status: c.Status, Name: c.Name, Label: label, Details: c.URL != ""})
			}
			row.Checks.Popup = pop
		}
	}
	switch panelAction(d) {
	case "merge":
		row.Pill = &ui.PullRequestPill{Label: "Merge", Tip: "Merge this " + terms.singular + " (" + mergeMethod(d) + ")"}
		if l.acting == "merge" {
			row.Pill.Label = "Merging..."
		}
	case "ready":
		row.Pill = &ui.PullRequestPill{Label: "Ready", Tip: "Mark this " + terms.singular + " as ready for review"}
		if l.acting == "ready" {
			row.Pill.Label = "Marking..."
		}
	}
	if row.Pill != nil {
		row.Pill.Disabled = l.acting != ""
	}
	if row.Checks == nil && row.Pill == nil && !(watching && d.State == "open") {
		return nil
	}
	return row
}

// shownPR is the card's pull request row i with its reference and read:
// row 0 the one the thread shows, then while unfolded its other links.
// Call with the lock held.
func (a *App) shownPR(i int) (e *envState, it prItem, ref *t3.PullRequestRef, l *prLoad) {
	e, t, p, _ := a.checkout()
	items := a.cardPRs()
	if t == nil || i < 0 || i >= len(items) || (i > 0 && !a.prsOpen) {
		return nil, prItem{}, nil, nil
	}
	ref = prRefOf(e, p, items[i], i == 0)
	if ref == nil {
		return nil, prItem{}, nil, nil
	}
	return e, items[i], ref, a.prLoads[prKeyOf(e.id(), *ref)]
}

// prPill runs row i's action: Merge asks first in T3's dialog, Ready goes.
func (a *App) prPill(i int) {
	a.mu.Lock()
	e, it, ref, l := a.shownPR(i)
	if l == nil || l.detail == nil || l.acting != "" {
		a.mu.Unlock()
		return
	}
	d := withNewerChecks(l)
	key, terms := prKeyOf(e.id(), *ref), termsOf(d.Provider)
	switch panelAction(d) {
	case "merge":
		a.closeCardMenusLocked()
		a.prAlert = &prAlert{envID: e.id(), key: key, ref: *ref, number: it.number, method: mergeMethod(d), terms: terms}
		a.mu.Unlock()
	case "ready":
		a.mu.Unlock()
		a.runPRAction(e, key, *ref, "ready", "", terms)
	default:
		a.mu.Unlock()
	}
}

// confirmMerge answers the merge dialog: ok merges with its method.
func (a *App) confirmMerge(ok bool) {
	a.mu.Lock()
	m := a.prAlert
	a.prAlert = nil
	var e *envState
	if m != nil {
		e = a.env(m.envID)
	}
	a.mu.Unlock()
	if ok && m != nil && e != nil {
		a.runPRAction(e, m.key, m.ref, "merge", m.method, m.terms)
	}
}

// prFailure is T3's title for a refused action.
func prFailure(action string, terms changeTerms) string {
	if action == "merge" {
		return "Could not merge this " + terms.singular
	}
	return "Could not mark this ready for review"
}

// runPRAction asks the host to merge or mark ready (pullRequests.runAction),
// the pill saying so meanwhile. A refusal goes to the error banner with
// T3's title; either way the detail and checks are read again.
func (a *App) runPRAction(e *envState, key string, ref t3.PullRequestRef, action, method string, terms changeTerms) {
	a.mu.Lock()
	l := a.prLoads[key]
	conn := e.conn
	if l == nil || l.acting != "" || conn == nil {
		a.mu.Unlock()
		return
	}
	l.acting = action
	a.mu.Unlock()
	a.changed()
	in := refInput(ref, map[string]any{"action": action})
	if method != "" {
		in["mergeMethod"] = method
	}
	go func() {
		err := conn.Call("pullRequests.runAction", in, nil)
		a.mu.Lock()
		l.acting, l.loadedAt, l.checkedAt = "", time.Time{}, time.Time{}
		if err != nil && !errors.Is(err, t3.ErrClosed) {
			a.errText = prFailure(action, terms) + ": " + err.Error()
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// toggleChecks opens row i's checks popover, or closes it.
func (a *App) toggleChecks(i int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e, _, ref, _ := a.shownPR(i)
	if ref == nil {
		return
	}
	key := prKeyOf(e.id(), *ref)
	open := a.checksOpen != key
	a.closeCardMenusLocked()
	if open {
		a.checksOpen, a.checksAll = key, false
	}
}

// openCheck opens the open popover's check i in the browser.
func (a *App) openCheck(i int) {
	a.mu.Lock()
	var url string
	if l := a.prLoads[a.checksOpen]; l != nil && l.detail != nil {
		if shown, _ := popupChecks(withNewerChecks(l).Checks, a.checksAll); i >= 0 && i < len(shown) {
			url = shown[i].URL
		}
	}
	open := a.OpenURL
	a.mu.Unlock()
	if url != "" && open != nil {
		if err := open(url); err != nil {
			a.failf("Unable to open check details: %v", err)
		}
	}
}
