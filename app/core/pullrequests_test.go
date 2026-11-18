package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// prLink is a pull request linked to a thread: number n, state "" for one
// the server hasn't read yet, its head and base branches, and when it was
// linked and last updated (minutes past a fixed hour).
func prLink(n int, state, head, base string, at int) t3.PullRequestLink {
	when := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC).Add(time.Duration(at) * time.Minute).Format(time.RFC3339)
	l := t3.PullRequestLink{Host: "github.com", Repository: "acme/widgets", Number: n,
		URL: fmt.Sprintf("https://github.com/acme/widgets/pull/%d", n), Source: "agent", LinkedAt: when}
	if state != "" {
		l.Snapshot = &t3.PullRequestSnapshot{State: state, Title: fmt.Sprintf("Change %d", n), HeadBranch: head, BaseBranch: base, UpdatedAt: when}
	}
	return l
}

// The pull request a thread shows follows resolveThreadCurrentPullRequest:
// its one open link; the top open layer of a stack; the newest of several
// open chains; else the top of its only chain, or the link updated last.
// Dismissed stack members don't count.
func TestCurrentPR(t *testing.T) {
	native := func(ls ...t3.PullRequestLink) []t3.PullRequestLink {
		for i := range ls {
			ls[i].Stack = &t3.PullRequestStack{ID: "s1", Layers: []t3.PullRequestStackLayer{{Number: 11}, {Number: 12}, {Number: 13}}}
		}
		return ls
	}
	dismissed := prLink(9, "open", "i", "main", 9)
	dismissed.Source = "stack-dismissed"
	for _, c := range []struct {
		name  string
		links []t3.PullRequestLink
		want  int
	}{
		{"none", nil, 0},
		{"the one open", []t3.PullRequestLink{prLink(1, "merged", "a", "main", 1), prLink(2, "open", "b", "main", 0)}, 2},
		{"a stack by branch: its top", []t3.PullRequestLink{prLink(3, "open", "c", "main", 0), prLink(4, "open", "d", "c", 1)}, 4},
		{"two open chains: the one linked last", []t3.PullRequestLink{prLink(5, "open", "e", "main", 2), prLink(6, "open", "f", "main", 1)}, 5},
		{"nothing open, one chain: its top", []t3.PullRequestLink{prLink(7, "merged", "g", "c", 3), prLink(8, "merged", "h", "g", 2)}, 8},
		{"nothing open, two chains: updated last", []t3.PullRequestLink{prLink(7, "merged", "g", "main", 3), prLink(8, "closed", "h", "main", 5)}, 8},
		{"a dismissed member doesn't count", []t3.PullRequestLink{dismissed, prLink(10, "merged", "j", "main", 1)}, 10},
		{"a native stack: its fullest layers' top", native(prLink(12, "open", "x", "main", 1), prLink(13, "open", "y", "x", 0), prLink(11, "open", "w", "main", 2)), 13},
		{"not read yet counts as open", []t3.PullRequestLink{prLink(14, "", "", "", 0), prLink(15, "merged", "k", "main", 1)}, 14},
	} {
		got := 0
		if cur := currentPR(c.links); cur != nil {
			got = cur.Number
		}
		if got != c.want {
			t.Errorf("%s: #%d, want #%d", c.name, got, c.want)
		}
	}
}

// The other links come a chain at a time, the chain updated last first,
// each from its base up (pullRequestListLines).
func TestOtherPRs(t *testing.T) {
	links := []t3.PullRequestLink{prLink(1, "merged", "a", "main", 1), prLink(5, "merged", "e", "main", 2), prLink(2, "open", "b", "a", 3)}
	cur := currentPR(links)
	var got []int
	for _, l := range otherPRs(links, cur) {
		got = append(got, l.Number)
	}
	if cur.Number != 2 || fmt.Sprint(got) != "[1 5]" {
		t.Fatalf("current #%d, others %v; want #2, [1 5]", cur.Number, got)
	}
}

// The card shows the current link as "#N: <title>" with its state, the
// tooltip naming the host's term; the others fold under "Show N more",
// and a row opens its pull request in the browser. Without links, the
// open pull request Git found shows, but only on the thread's branch.
func TestCardPRRows(t *testing.T) {
	a, s := cardApp(t, status(nil), func(a *App, e *envState, th *t3.Thread) {
		th.Branch = "feature"
		th.PullRequests = []t3.PullRequestLink{prLink(1, "merged", "a", "main", 1), prLink(2, "open", "b", "a", 3), prLink(5, "closed", "e", "main", 2)}
	})
	p := s.Card.PRs
	if p == nil || len(p.Rows) != 1 || p.Others != 2 || p.Expanded {
		t.Fatalf("rows %+v", p)
	}
	if r := p.Rows[0]; r.Label != "#2: Change 2" || r.State != "open" || r.Tip != "PR #2 - Open: Change 2" || r.Watching {
		t.Errorf("current row %+v", r)
	}
	var opened []string
	a.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.prs", Kind: "click"})
	p = built(a, time.Now()).Card.PRs
	if len(p.Rows) != 3 || !p.Expanded || p.Rows[1].Label != "#1: Change 1" || p.Rows[2].State != "closed" {
		t.Fatalf("expanded %+v", p)
	}
	a.handle(th, ui.Action{ID: "card.pr:2", Kind: "middle"})
	if len(opened) != 1 || opened[0] != "https://github.com/acme/widgets/pull/5" {
		t.Fatalf("opened %v", opened)
	}

	branchPR := status(func(st *t3.VcsStatus) {
		st.PR = &t3.VcsChangeRequest{Number: 7, Title: "Fix the parser", URL: "https://github.com/acme/widgets/pull/7", State: "open"}
	})
	_, s = cardApp(t, branchPR, func(a *App, e *envState, th *t3.Thread) { th.Branch = "feature" })
	if p := s.Card.PRs; p == nil || len(p.Rows) != 1 || p.Rows[0].Label != "#7: Fix the parser" || p.Others != 0 {
		t.Fatalf("the branch's pull request: %+v", p)
	}
	_, s = cardApp(t, branchPR, func(a *App, e *envState, th *t3.Thread) { th.Branch = "other" })
	if s.Card.PRs != nil {
		t.Fatalf("off the thread's branch: %+v", s.Card.PRs)
	}
	// A folder that isn't a repository still lists the thread's links.
	_, s = cardApp(t, &t3.VcsStatus{}, func(a *App, e *envState, th *t3.Thread) {
		th.PullRequests = []t3.PullRequestLink{prLink(3, "", "", "", 0)}
	})
	if p := s.Card.PRs; p == nil || p.Rows[0].Label != "#3" || p.Rows[0].Tip != "Pull request #3" || !s.Card.InitGit {
		t.Fatalf("not a repository: %+v", p)
	}
}

// While the agent watches the thread's pull request its row ends in an
// eye, which stops that (thread.pull-request.watch, watching false).
func TestStopWatchingPR(t *testing.T) {
	a, f := liveApp(t, status(nil))
	a.mu.Lock()
	l := prLink(4, "open", "b", "main", 0)
	l.Watch = &t3.PullRequestWatch{StartedAt: l.LinkedAt}
	a.envs[0].threads["t1"].PullRequests = []t3.PullRequestLink{l}
	a.mu.Unlock()
	if r := built(a, time.Now()).Card.PRs.Rows[0]; !r.Watching {
		t.Fatalf("row %+v", r)
	}
	a.handle(ui.NewTheme(), ui.Action{ID: "card.prwatch:0", Kind: "click"})
	d := f.wait("orchestration.dispatchCommand:thread.pull-request.watch", 1)
	if d.Payload["threadId"] != "t1" || d.Payload["number"] != 4.0 || d.Payload["repository"] != "acme/widgets" || d.Payload["host"] != "github.com" || d.Payload["watching"] != false {
		t.Fatalf("watch payload %v", d.Payload)
	}
}
