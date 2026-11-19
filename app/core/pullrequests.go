package core

import (
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The details card's pull request rows (ThreadDetailsPrRows.tsx,
// ThreadStatusIndicators.tsx; docs/ui/05-header.md §3.8, §5.10): the pull
// request a thread shows, then its other linked ones, chained and ordered
// as T3's client runtime does (threadPullRequests.ts).

// visiblePRs are a thread's links without the stack-dismissed tombstones,
// which only stop the server's sync adding a stack member back.
func visiblePRs(links []t3.PullRequestLink) []t3.PullRequestLink {
	var out []t3.PullRequestLink
	for _, l := range links {
		if l.Source != "stack-dismissed" {
			out = append(out, l)
		}
	}
	return out
}

// prRepo is a link's host and repository, lower-cased.
func prRepo(l t3.PullRequestLink) string {
	return strings.ToLower(strings.TrimSpace(l.Host)) + "/" + strings.ToLower(strings.TrimSpace(l.Repository))
}

// prKey is a link's identity: its repository and number.
func prKey(l t3.PullRequestLink) string { return prRepo(l) + "#" + strconv.Itoa(l.Number) }

// prOpen reports whether a link is open, or not read yet.
func prOpen(l t3.PullRequestLink) bool { return l.Snapshot == nil || l.Snapshot.State == "open" }

// millis is an ISO time in milliseconds, 0 when it doesn't parse.
func millis(at string) int64 {
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

// prUpdated is when a link last changed: its snapshot's update, else when
// it was linked.
func prUpdated(l t3.PullRequestLink) int64 {
	if l.Snapshot != nil && l.Snapshot.UpdatedAt != "" {
		return millis(l.Snapshot.UpdatedAt)
	}
	return millis(l.LinkedAt)
}

// prChain is pull requests stacked on one another: a host's own stack, or
// one derived from branches (a pull request based on another's head sits
// on it). Depths count up from the base; siblings share one.
type prChain struct {
	layers []t3.PullRequestLink
	depths []int
}

// fromTop is the chain's links from its top down.
func (c prChain) fromTop() []t3.PullRequestLink {
	order := make([]int, len(c.layers))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		i, j := order[a], order[b]
		if c.depths[i] != c.depths[j] {
			return c.depths[i] > c.depths[j]
		}
		return i > j
	})
	out := make([]t3.PullRequestLink, len(order))
	for k, i := range order {
		out[k] = c.layers[i]
	}
	return out
}

// prChains groups links into chains (resolveThreadPullRequestChains): the
// members of each native stack in the order of the fullest layer list any
// of them carries, then the rest by branch, each base followed depth first
// by what's stacked on it, and anything left over on its own.
func prChains(links []t3.PullRequestLink) []prChain {
	visible := visiblePRs(links)
	var chains []prChain
	placed := map[string]bool{}

	var stackKeys []string
	stacks := map[string][]t3.PullRequestLink{}
	for _, l := range visible {
		if l.Stack == nil {
			continue
		}
		k := prRepo(l) + "#stack:" + l.Stack.ID
		if _, ok := stacks[k]; !ok {
			stackKeys = append(stackKeys, k)
		}
		stacks[k] = append(stacks[k], l)
	}
	for _, k := range stackKeys {
		members := stacks[k]
		numbers := map[int]bool{}
		for _, m := range members {
			numbers[m.Number] = true
		}
		covers := func(layers []t3.PullRequestStackLayer) int {
			n := 0
			for _, layer := range layers {
				if numbers[layer.Number] {
					n++
				}
			}
			return n
		}
		// Saved snapshots can lag behind newer members of the same stack.
		layers := members[0].Stack.Layers
		coverage := covers(layers)
		for _, m := range members[1:] {
			if c := covers(m.Stack.Layers); c > coverage || (c == coverage && len(m.Stack.Layers) > len(layers)) {
				layers, coverage = m.Stack.Layers, c
			}
		}
		at := map[int]int{}
		for i, layer := range layers {
			at[layer.Number] = i
		}
		pos := func(n int) int {
			if i, ok := at[n]; ok {
				return i
			}
			return len(layers)
		}
		sort.SliceStable(members, func(i, j int) bool { return pos(members[i].Number) < pos(members[j].Number) })
		depths := make([]int, len(members))
		for i, m := range members {
			placed[prKey(m)], depths[i] = true, i
		}
		chains = append(chains, prChain{layers: members, depths: depths})
	}

	var remaining []*t3.PullRequestLink
	for i := range visible {
		if !placed[prKey(visible[i])] {
			remaining = append(remaining, &visible[i])
		}
	}
	branch := func(l *t3.PullRequestLink, name string) string { return prRepo(*l) + ":" + name }
	// A head two links share names neither.
	byHead := map[string]*t3.PullRequestLink{}
	for _, l := range remaining {
		if l.Snapshot == nil {
			continue
		}
		k := branch(l, l.Snapshot.HeadBranch)
		if _, ok := byHead[k]; ok {
			byHead[k] = nil
		} else {
			byHead[k] = l
		}
	}
	parentOf := func(l *t3.PullRequestLink) *t3.PullRequestLink {
		if l.Snapshot == nil {
			return nil
		}
		if p := byHead[branch(l, l.Snapshot.BaseBranch)]; p != l {
			return p
		}
		return nil
	}
	children := map[string][]*t3.PullRequestLink{}
	for _, l := range remaining {
		if p := parentOf(l); p != nil {
			children[prKey(*p)] = append(children[prKey(*p)], l)
		}
	}
	type pending struct {
		l     *t3.PullRequestLink
		depth int
	}
	for _, base := range remaining {
		if parentOf(base) != nil {
			continue
		}
		var c prChain
		stack := []pending{{base, 0}}
		for len(stack) > 0 {
			next := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			k := prKey(*next.l)
			// Two links with one identity would share a child list forever.
			if placed[k] {
				continue
			}
			placed[k] = true
			c.layers, c.depths = append(c.layers, *next.l), append(c.depths, next.depth)
			below := children[k]
			for i := len(below) - 1; i >= 0; i-- {
				stack = append(stack, pending{below[i], next.depth + 1})
			}
		}
		if len(c.layers) > 0 {
			chains = append(chains, c)
		}
	}
	for _, l := range remaining {
		if !placed[prKey(*l)] {
			chains = append(chains, prChain{layers: []t3.PullRequestLink{*l}, depths: []int{0}})
		}
	}
	return chains
}

// currentPR is the link a thread shows (resolveThreadCurrentPullRequest):
// its one open link; with several open, the top open layer of the newest
// chain; else the top of its only chain, or the link updated last.
func currentPR(links []t3.PullRequestLink) *t3.PullRequestLink {
	visible := visiblePRs(links)
	if len(visible) == 0 {
		return nil
	}
	var open []t3.PullRequestLink
	for _, l := range visible {
		if prOpen(l) {
			open = append(open, l)
		}
	}
	if len(open) == 1 {
		return &open[0]
	}
	chains := prChains(visible)
	if len(open) > 1 {
		type openChain struct {
			layers []t3.PullRequestLink
			newest int64
		}
		var ocs []openChain
		for _, c := range chains {
			oc := openChain{newest: -1 << 62}
			for _, l := range c.fromTop() {
				if prOpen(l) {
					oc.layers, oc.newest = append(oc.layers, l), max(oc.newest, millis(l.LinkedAt))
				}
			}
			if len(oc.layers) > 0 {
				ocs = append(ocs, oc)
			}
		}
		sort.SliceStable(ocs, func(i, j int) bool { return ocs[i].newest > ocs[j].newest })
		return &ocs[0].layers[0]
	}
	if len(chains) == 1 {
		top := chains[0].fromTop()[0]
		return &top
	}
	latest := visible[0]
	for _, l := range visible[1:] {
		if prUpdated(l) > prUpdated(latest) {
			latest = l
		}
	}
	return &latest
}

// otherPRs are the thread's links besides cur, chain by chain, the chain
// changed last first (pullRequestListLines).
func otherPRs(links []t3.PullRequestLink, cur *t3.PullRequestLink) []t3.PullRequestLink {
	chains := prChains(links)
	newest := func(c prChain) int64 {
		n := int64(-1 << 62)
		for _, l := range c.layers {
			n = max(n, prUpdated(l))
		}
		return n
	}
	sort.SliceStable(chains, func(i, j int) bool { return newest(chains[i]) > newest(chains[j]) })
	var out []t3.PullRequestLink
	for _, c := range chains {
		for _, l := range c.layers {
			if prKey(l) != prKey(*cur) {
				out = append(out, l)
			}
		}
	}
	return out
}

// prItem is one of the card's pull request rows: the link it stands for
// (nil for the branch's own pull request), the reference the server's
// fallback names it by, what's known of it, and where it opens.
type prItem struct {
	link         *t3.PullRequestLink
	fallback     *t3.PullRequestRef
	number       int
	url          string
	state, title string // "" until the pull request is read
	draft        bool
	watching     bool
}

// prItems are the thread's pull request rows (useThreadDisplayedPullRequest):
// the current link, read from its snapshot; with none, the one the server
// names for the thread (useLinkedThreadPullRequest's fallback: a server
// without links names its one linked pull request, any server the open one
// on the thread's branch), read from its host; else the open pull request
// Git found on the thread's branch (branchPR); then its other links. Call
// with the lock held.
func prItems(e *envState, t *t3.Thread, branchPR *t3.VcsChangeRequest) []prItem {
	if t == nil {
		return nil
	}
	links := e.config != nil
	var cur *t3.PullRequestLink
	if links {
		cur = currentPR(t.PullRequests)
	}
	fallback := t.BranchPullRequest
	if !links && t.LinkedPullRequest != nil {
		fallback = t.LinkedPullRequest
	}
	if cur == nil && fallback != nil && fallback.Number > 0 && fallback.URL != "" {
		ref := &t3.PullRequestRef{ProjectID: fallback.ProjectID, Repository: fallback.Repository, Number: fallback.Number}
		if u, err := url.Parse(fallback.URL); err == nil {
			ref.Host = u.Hostname()
		}
		return []prItem{{fallback: ref, number: fallback.Number, url: fallback.URL}}
	}
	link := func(l *t3.PullRequestLink) prItem {
		it := prItem{link: l, number: l.Number, url: l.URL, watching: l.Watch != nil}
		if s := l.Snapshot; s != nil {
			it.state, it.title, it.draft = s.State, s.Title, s.IsDraft
		}
		return it
	}
	switch {
	case cur != nil:
		items := []prItem{link(cur)}
		others := otherPRs(t.PullRequests, cur)
		for i := range others {
			items = append(items, link(&others[i]))
		}
		return items
	case branchPR != nil && branchPR.Number > 0 && branchPR.URL != "":
		return []prItem{{number: branchPR.Number, url: branchPR.URL, state: branchPR.State, title: branchPR.Title, draft: branchPR.IsDraft}}
	}
	return nil
}

// cardPRs are the open thread's pull request rows, as the card lists
// them. Call with the lock held.
func (a *App) cardPRs() []prItem {
	e, t, p, v := a.checkout()
	if e == nil || t == nil {
		return nil
	}
	return prItems(e, t, a.branchPR(e, t, p, v))
}

// branchPR is the open pull request Git found for the thread's branch,
// which the rows under the branch row fall back on when the thread links
// none (resolveDisplayedThreadPr): only while the checkout, as the branch
// row names it, is on the thread's own branch. Call with the lock held.
func (a *App) branchPR(e *envState, t *t3.Thread, p t3.Project, v *vcsState) *t3.VcsChangeRequest {
	if v == nil || v.status == nil || !v.status.IsRepo || gitCwd(t.WorktreePath, p) == "" || e.config == nil || isScratch(p, e.config) {
		return nil
	}
	st := v.status
	if t.Branch == "" || t.Branch != a.branchOf(t, v) || st.RefName != t.Branch || st.PR == nil || st.PR.State != "open" {
		return nil
	}
	return st.PR
}

// prState is a pull request's state as T3 presents it: open, draft, closed
// or merged; "" before it's read.
func (it prItem) prState() string {
	if it.state == "open" && it.draft {
		return "draft"
	}
	return it.state
}

// prStateLabels are the states' names (PULL_REQUEST_STATE_PRESENTATION).
var prStateLabels = map[string]string{"open": "Open", "draft": "Draft", "closed": "Closed", "merged": "Merged"}

// prsScene is the card's pull request rows in project p: the shown one,
// then, while expanded, the others "Show N more" folds. Each reads its
// pull request's detail while the card can be seen, which gives it its
// state and, with something to say under its title, its detailed form;
// what the card no longer shows is forgotten. Call with the lock held.
func (a *App) prsScene(e *envState, p t3.Project, items []prItem, terms changeTerms, now time.Time, visible bool) *ui.PullRequests {
	shown := map[string]bool{}
	defer func() {
		for k := range a.prLoads {
			if !shown[k] {
				delete(a.prLoads, k)
			}
		}
	}()
	if len(items) == 0 {
		return nil
	}
	rows := &ui.PullRequests{Others: len(items) - 1, Expanded: a.prsOpen}
	for i, it := range items {
		if i > 0 && !a.prsOpen {
			break
		}
		var detail *ui.PullRequestDetailRow
		if ref := prRefOf(e, p, it, i == 0); ref != nil {
			key := prKeyOf(e.id(), *ref)
			shown[key] = true
			if l := a.loadPRLocked(e, key, *ref, now, visible); l.detail != nil {
				d := withNewerChecks(l)
				it.state, it.draft = d.State, d.IsDraft
				if it.title == "" {
					it.title = d.Title
				}
				detail = a.detailRow(key, l, it.number, it.watching)
			}
		}
		r := ui.PullRequestRow{State: it.prState(), Label: "#" + strconv.Itoa(it.number), Watching: it.watching && (it.state == "" || it.state == "open"), Detail: detail}
		if title := strings.TrimSpace(it.title); title != "" {
			r.Label += ": " + it.title
		}
		// The tooltip leads with the host's word for it and its state.
		r.Tip = strings.ToUpper(terms.singular[:1]) + terms.singular[1:] + " #" + strconv.Itoa(it.number)
		if label := prStateLabels[r.State]; label != "" {
			r.Tip = terms.short + " #" + strconv.Itoa(it.number) + " - " + label + ": " + it.title
		}
		rows.Rows = append(rows.Rows, r)
	}
	return rows
}

// openPR opens the card's pull request row i in the browser, as T3 does
// with no in-app pull request view to open it in.
func (a *App) openPR(i int) {
	a.mu.Lock()
	items := a.cardPRs()
	a.mu.Unlock()
	if i >= 0 && i < len(items) {
		a.openWeb("Couldn't open "+items[i].url, items[i].url)
	}
}

// openWeb opens a link the server handed on, a pull request's or a check's,
// in this computer's browser, only when it's http or https with a host:
// whoever made it chose it, and the OS would open a file, a share or an
// option-like "-a…" too (T3's parseSafeExternalUrl). what heads a failure.
func (a *App) openWeb(what, link string) {
	a.mu.Lock()
	open := a.OpenURL
	a.mu.Unlock()
	if open == nil {
		return
	}
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		a.failf(what, errors.New("only web links open"))
		return
	}
	if err := open(u.String()); err != nil {
		a.failf(what, err)
	}
}

// stopWatchingPR has the agent stop watching row i's pull request
// (thread.pull-request.watch, watching false).
func (a *App) stopWatchingPR(i int) {
	a.mu.Lock()
	e, t, _, _ := a.checkout()
	items := a.cardPRs()
	if t == nil || e.conn == nil || i < 0 || i >= len(items) || items[i].link == nil {
		a.mu.Unlock()
		return
	}
	l, conn, id := items[i].link, e.conn, t.ID
	a.mu.Unlock()
	go func() {
		err := conn.Dispatch(map[string]any{"type": "thread.pull-request.watch", "threadId": id,
			"host": l.Host, "repository": l.Repository, "number": l.Number, "watching": false})
		if err != nil {
			a.failf("Couldn't stop watching #"+strconv.Itoa(l.Number), err)
		}
	}()
}
