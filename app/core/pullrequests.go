package core

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// A thread's linked pull requests as the sidebar shows them: the meta
// line's badge and the lists of them (threadPullRequests.ts in
// source-control-core, ThreadStatusIndicators.tsx).

// visiblePRs is the links a thread shows (visibleThreadPullRequests): all
// but the stack members the user dismissed.
func visiblePRs(links []t3.PullRequestLink) []t3.PullRequestLink {
	var out []t3.PullRequestLink
	for _, l := range links {
		if l.Source != "stack-dismissed" {
			out = append(out, l)
		}
	}
	return out
}

// prRepo is a link's host and repository, lower-cased: older servers send
// no host, which its URL has.
func prRepo(l t3.PullRequestLink) string {
	host := l.Host
	if host == "" {
		if u, err := url.Parse(l.URL); err == nil {
			host = u.Hostname()
		}
	}
	return strings.ToLower(strings.TrimSpace(host)) + "/" + strings.ToLower(strings.TrimSpace(l.Repository))
}

// prKey is a link's identity: its host, repository and number.
func prKey(l t3.PullRequestLink) string { return prRepo(l) + "#" + strconv.Itoa(l.Number) }

// prOpen reports a link that's open, or not read yet.
func prOpen(l t3.PullRequestLink) bool { return l.Snapshot == nil || l.Snapshot.State == "open" }

// prUpdated is when a link last changed, as the server read it, else when
// it was linked.
func prUpdated(l t3.PullRequestLink) int64 {
	if l.Snapshot != nil && l.Snapshot.UpdatedAt != "" {
		return ms(l.Snapshot.UpdatedAt)
	}
	return ms(l.LinkedAt)
}

// prChain is links that build on each other (ThreadPullRequestChain): a
// host's own stack, its members bottom to top, or a chain derived from
// their branches, each base before what's on it; with each one's depth.
type prChain struct {
	native bool
	links  []t3.PullRequestLink
	depths []int
}

// prChains sorts visible links into chains (resolveThreadPullRequestChains):
// a native stack's members by its layers, as the member that covers most
// of them has them; then the rest by their branches, a link whose base
// branch is another's head going on it, unless two links share that head.
// A link on none of them is a chain of its own.
func prChains(visible []t3.PullRequestLink) []prChain {
	var chains []prChain
	placed := map[string]bool{}
	var stacks []string
	members := map[string][]t3.PullRequestLink{}
	for _, l := range visible {
		if l.Stack == nil {
			continue
		}
		k := prRepo(l) + "#stack:" + l.Stack.ID
		if members[k] == nil {
			stacks = append(stacks, k)
		}
		members[k] = append(members[k], l)
	}
	for _, k := range stacks {
		group := members[k]
		in := map[int]bool{}
		for _, m := range group {
			in[m.Number] = true
		}
		covers := func(m t3.PullRequestLink) int {
			n := 0
			for _, l := range m.Stack.Layers {
				if in[l.Number] {
					n++
				}
			}
			return n
		}
		// The saved stacks can lag behind newer members' own.
		layers, coverage := group[0].Stack, covers(group[0])
		for _, m := range group[1:] {
			if c := covers(m); c > coverage || c == coverage && len(m.Stack.Layers) > len(layers.Layers) {
				layers, coverage = m.Stack, c
			}
		}
		order := map[int]int{}
		for i, l := range layers.Layers {
			order[l.Number] = i
		}
		at := func(m t3.PullRequestLink) int {
			if i, ok := order[m.Number]; ok {
				return i
			}
			return len(layers.Layers)
		}
		sort.SliceStable(group, func(i, j int) bool { return at(group[i]) < at(group[j]) })
		c := prChain{native: true, links: group}
		for i, m := range group {
			placed[prKey(m)] = true
			c.depths = append(c.depths, i)
		}
		chains = append(chains, c)
	}
	var rest []*t3.PullRequestLink
	for i := range visible {
		if !placed[prKey(visible[i])] {
			rest = append(rest, &visible[i])
		}
	}
	branch := func(l *t3.PullRequestLink, b string) string { return prRepo(*l) + ":" + b }
	byHead := map[string]*t3.PullRequestLink{} // nil where two links share a head
	for _, l := range rest {
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
	parent := func(l *t3.PullRequestLink) *t3.PullRequestLink {
		if l.Snapshot == nil {
			return nil
		}
		if p := byHead[branch(l, l.Snapshot.BaseBranch)]; p != l {
			return p
		}
		return nil
	}
	children := map[string][]*t3.PullRequestLink{}
	for _, l := range rest {
		if p := parent(l); p != nil {
			children[prKey(*p)] = append(children[prKey(*p)], l)
		}
	}
	type step struct {
		l     *t3.PullRequestLink
		depth int
	}
	for _, base := range rest {
		if parent(base) != nil {
			continue
		}
		var c prChain
		for todo := []step{{base, 0}}; len(todo) > 0; {
			s := todo[len(todo)-1]
			todo = todo[:len(todo)-1]
			k := prKey(*s.l)
			// Two links of one identity would share their children.
			if placed[k] {
				continue
			}
			placed[k] = true
			c.links, c.depths = append(c.links, *s.l), append(c.depths, s.depth)
			below := children[k]
			for i := len(below) - 1; i >= 0; i-- {
				todo = append(todo, step{below[i], s.depth + 1})
			}
		}
		if len(c.links) > 0 {
			chains = append(chains, c)
		}
	}
	for _, l := range rest {
		if !placed[prKey(*l)] {
			chains = append(chains, prChain{links: []t3.PullRequestLink{*l}, depths: []int{0}})
		}
	}
	return chains
}

// fromTop is a chain's links top first: the deepest, the later of two as
// deep.
func (c prChain) fromTop() []t3.PullRequestLink {
	idx := make([]int, len(c.links))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		if da, db := c.depths[idx[a]], c.depths[idx[b]]; da != db {
			return da > db
		}
		return idx[a] > idx[b]
	})
	out := make([]t3.PullRequestLink, len(idx))
	for i, j := range idx {
		out[i] = c.links[j]
	}
	return out
}

// currentPR is the link a thread's badge stands for
// (resolveThreadCurrentPullRequest): the one open link; with several open,
// the top of the newest open chain; else the top of the one chain, or the
// latest to change.
func currentPR(visible []t3.PullRequestLink) t3.PullRequestLink {
	var open []t3.PullRequestLink
	for _, l := range visible {
		if prOpen(l) {
			open = append(open, l)
		}
	}
	if len(open) == 1 {
		return open[0]
	}
	chains := prChains(visible)
	if len(open) > 1 {
		type openChain struct {
			links  []t3.PullRequestLink
			newest int64
		}
		var oc []openChain
		for _, c := range chains {
			var ls []t3.PullRequestLink
			var newest int64
			for _, l := range c.fromTop() {
				if prOpen(l) {
					ls, newest = append(ls, l), max(newest, ms(l.LinkedAt))
				}
			}
			if len(ls) > 0 {
				oc = append(oc, openChain{ls, newest})
			}
		}
		sort.SliceStable(oc, func(i, j int) bool { return oc[i].newest > oc[j].newest })
		return oc[0].links[0]
	}
	if len(chains) == 1 {
		return chains[0].fromTop()[0]
	}
	latest := visible[0]
	for _, l := range visible[1:] {
		if prUpdated(l) > prUpdated(latest) {
			latest = l
		}
	}
	return latest
}

// prState is a link's state as a badge shows it: a draft is its own, and
// one not read yet is "" (status pending).
func prState(l t3.PullRequestLink) string {
	switch {
	case l.Snapshot == nil:
		return ""
	case l.Snapshot.State == "open" && l.Snapshot.IsDraft:
		return "draft"
	}
	return l.Snapshot.State
}

// prOverall is several links' state together (resolveThreadPullRequestBadge):
// draft if every one is a draft, open if any is open (or not read yet),
// merged if every one is merged, else closed.
func prOverall(visible []t3.PullRequestLink) string {
	drafts, open, merged := true, false, true
	for _, l := range visible {
		s := "open"
		if l.Snapshot != nil {
			s = l.Snapshot.State
		}
		drafts = drafts && l.Snapshot != nil && s == "open" && l.Snapshot.IsDraft
		open = open || s == "open"
		merged = merged && s == "merged"
	}
	switch {
	case drafts:
		return "draft"
	case open:
		return "open"
	case merged:
		return "merged"
	}
	return "closed"
}

// prBadge is the meta line's pull request (the row's badge): the current
// link's state and number; with several linked, their overall state, as
// the stack's layer count when they make one chain, else "+N". Without
// the server's config there are no links to go by (only a server that
// sends it can hold several), and the badge is the pull request the
// thread names, or its branch's.
func prBadge(t *t3.Thread, loaded bool) *ui.PRBadge {
	if vis := visiblePRs(t.PullRequests); loaded && len(vis) > 0 {
		cur := currentPR(vis)
		b := &ui.PRBadge{State: prState(cur), Text: strconv.Itoa(cur.Number), URL: cur.URL}
		switch {
		case len(vis) > 1 && len(prChains(vis)) == 1:
			b.State, b.Stack, b.Text = prOverall(vis), true, strconv.Itoa(len(vis))
		case len(vis) > 1:
			b.State, b.Text = prOverall(vis), fmt.Sprintf("+%d", len(vis))
		}
		return b
	}
	for _, l := range []*t3.LinkedPullRequest{t.LinkedPullRequest, t.BranchPullRequest} {
		if l != nil {
			return &ui.PRBadge{Text: strconv.Itoa(l.Number), URL: l.URL}
		}
	}
	return nil
}

// prLines is a thread's links as a list shows them (pullRequestListLines):
// chain by chain, the latest to change first, each in its own order and
// indented by its depth, a chain's first line saying what it is and how
// many it holds.
func prLines(visible []t3.PullRequestLink) []ui.PRLine {
	chains := prChains(visible)
	newest := func(c prChain) int64 {
		var n int64
		for _, l := range c.links {
			n = max(n, prUpdated(l))
		}
		return n
	}
	sort.SliceStable(chains, func(i, j int) bool { return newest(chains[i]) > newest(chains[j]) })
	var out []ui.PRLine
	for _, c := range chains {
		for i, l := range c.links {
			line := ui.PRLine{State: prState(l), Number: l.Number, Title: l.Repository, URL: l.URL, Depth: c.depths[i]}
			if l.Snapshot != nil {
				line.Title = l.Snapshot.Title
			}
			if i == 0 && len(c.links) > 1 {
				kind := "chain"
				if c.native {
					kind = "stack"
				}
				line.Chain = fmt.Sprintf("%s · %d", kind, len(c.links))
			}
			out = append(out, line)
		}
	}
	return out
}
