package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// check is a pull request check by status.
func check(status string) t3.PullRequestCheck {
	return t3.PullRequestCheck{Name: status + " check", Status: status}
}

// mergeable is an open pull request whose checks pass and whose host lets
// the viewer merge or squash it; fn changes it.
func mergeable(fn func(d *t3.PullRequestDetail)) *t3.PullRequestDetail {
	d := &t3.PullRequestDetail{Provider: "github", Number: 12, Title: "Teach the parser about tabs", State: "open", Mergeability: "mergeable",
		Checks: []t3.PullRequestCheck{check("success")}, MergeCapabilities: map[string]bool{"merge": true, "squash": true}}
	d.Capabilities.Actions, d.Capabilities.MergeMethods = []string{"merge", "ready"}, []string{"squash", "merge"}
	d.ViewerPermissions.Actions = []string{"merge", "ready"}
	if fn != nil {
		fn(d)
	}
	return d
}

// A row offers what T3's resolveThreadPanelPullRequestAction does (§4.6),
// first match wins.
func TestPanelAction(t *testing.T) {
	for _, c := range []struct {
		what string
		d    *t3.PullRequestDetail
		want string
	}{
		{"not read", nil, ""},
		{"merged", mergeable(func(d *t3.PullRequestDetail) { d.State = "merged" }), ""},
		{"conflicting", mergeable(func(d *t3.PullRequestDetail) { d.Mergeability = "conflicting" }), "resolve"},
		{"a draft it may mark ready", mergeable(func(d *t3.PullRequestDetail) { d.IsDraft = true }), "ready"},
		{"a draft it may not", mergeable(func(d *t3.PullRequestDetail) { d.IsDraft, d.ViewerPermissions.Actions = true, []string{"merge"} }), ""},
		{"a check failed", mergeable(func(d *t3.PullRequestDetail) { d.Checks = append(d.Checks, check("failure")) }), "fix"},
		{"one was cancelled", mergeable(func(d *t3.PullRequestDetail) { d.Checks = append(d.Checks, check("cancelled")) }), "fix"},
		{"one runs", mergeable(func(d *t3.PullRequestDetail) { d.Checks = append(d.Checks, check("pending")) }), ""},
		{"one waits on someone", mergeable(func(d *t3.PullRequestDetail) { d.Checks = append(d.Checks, check("action-required")) }), ""},
		{"passing", mergeable(nil), "merge"},
		{"no checks", mergeable(func(d *t3.PullRequestDetail) { d.Checks = nil }), "merge"},
		{"skipped ones pass", mergeable(func(d *t3.PullRequestDetail) { d.Checks = []t3.PullRequestCheck{check("skipped"), check("neutral")} }), "merge"},
		{"the host offers no method", mergeable(func(d *t3.PullRequestDetail) { d.MergeCapabilities = map[string]bool{"rebase": true} }), ""},
		{"the viewer may not merge", mergeable(func(d *t3.PullRequestDetail) { d.ViewerPermissions.Actions = []string{"ready"} }), ""},
	} {
		if got := panelAction(c.d); got != c.want {
			t.Errorf("%s: %q, want %q", c.what, got, c.want)
		}
	}
	// Merge uses T3's remembered default, merge, while the host allows it.
	if m := mergeMethod(mergeable(nil)); m != "merge" {
		t.Errorf("method %q", m)
	}
	if m := mergeMethod(mergeable(func(d *t3.PullRequestDetail) { d.MergeCapabilities = map[string]bool{"squash": true} })); m != "squash" {
		t.Errorf("method without merge %q", m)
	}
}

// The checks read as T3's presentation reads them: the rollup, the meta
// line's count, the popover's summary and which checks it folds away.
func TestChecksWords(t *testing.T) {
	approval := check("action-required")
	approval.URL = "https://github.com/acme/widgets/actions/runs/123/job/4"
	for _, c := range []struct {
		checks                 []t3.PullRequestCheck
		rollup, count, summary string
	}{
		{nil, "", "", "No checks reported"},
		{[]t3.PullRequestCheck{check("success"), check("success")}, "passing", "", "All checks passed"},
		{[]t3.PullRequestCheck{check("success"), check("skipped")}, "passing", "", "1 of 2 passing"},
		{[]t3.PullRequestCheck{check("skipped")}, "", "", "0 of 1 passing"},
		{[]t3.PullRequestCheck{check("success"), check("pending"), check("pending")}, "pending", "1/3 checks", "2 of 3 running"},
		{[]t3.PullRequestCheck{check("failure"), check("cancelled"), check("success")}, "failing", "2 failed", "2 of 3 failing"},
		{[]t3.PullRequestCheck{check("failure"), check("pending")}, "failing", "1/2 checks", "1 of 2 failing"},
		{[]t3.PullRequestCheck{approval, check("success")}, "pending", "1/2 checks", "1 workflow awaiting approval"},
		{[]t3.PullRequestCheck{approval, approval, check("action-required")}, "pending", "0/3 checks", "2 workflows and 1 check awaiting action"},
		{[]t3.PullRequestCheck{check("action-required")}, "pending", "0/1 checks", "1 check awaiting action"},
	} {
		if r, n, s := checksRollup(c.checks), checksCount(c.checks), checksSummary(c.checks); r != c.rollup || n != c.count || s != c.summary {
			t.Errorf("%v: %q %q %q, want %q %q %q", c.checks, r, n, s, c.rollup, c.count, c.summary)
		}
	}
	checks := []t3.PullRequestCheck{check("success"), check("pending"), check("failure"), check("skipped")}
	names := func(cs []t3.PullRequestCheck) string {
		var out []string
		for _, c := range cs {
			out = append(out, c.Status)
		}
		return strings.Join(out, " ")
	}
	if shown, more := popupChecks(checks, false); names(shown) != "failure pending" || more != "Show all" {
		t.Errorf("folded: %s, %q", names(shown), more)
	}
	if shown, more := popupChecks(checks, true); names(shown) != "failure pending success skipped" || more != "Show less" {
		t.Errorf("unfolded: %s, %q", names(shown), more)
	}
	if shown, more := popupChecks(checks[:1], false); names(shown) != "success" || more != "" {
		t.Errorf("nothing to fold: %s, %q", names(shown), more)
	}
}

// A loaded pull request's tooltip is T3's rich one (ThreadDetailsPrRow.tsx,
// 05-header.md §3.8): its state, its branches, its checks while it's open
// (describePullRequestChecks, not the popover's summary), its files and
// lines, and a draft's conflicts. Its first showing reads the lines by kind
// (pullRequests.changeBreakdown), once, which then lists the kinds with
// lines in T3's order.
func TestPullRequestTip(t *testing.T) {
	for _, c := range []struct {
		checks []t3.PullRequestCheck
		want   string
	}{
		{[]t3.PullRequestCheck{check("failure"), check("pending")}, "1 of 2 running · 1 failed"},
		{[]t3.PullRequestCheck{check("failure"), check("success")}, "1 of 2 failing"},
		{[]t3.PullRequestCheck{check("action-required"), check("success")}, "1 of 2 awaiting action"},
		{[]t3.PullRequestCheck{check("success"), check("skipped")}, "1 of 2 passing"},
		{[]t3.PullRequestCheck{check("success")}, "All checks passed"},
	} {
		if got := describeChecks(c.checks); got != c.want {
			t.Errorf("%v: %q, want %q", c.checks, got, c.want)
		}
	}
	draft := prTip(mergeable(func(d *t3.PullRequestDetail) { d.IsDraft, d.Mergeability, d.BaseBranch = true, "conflicting", "main" }), "draft", nil)
	if draft.StateLabel != "Draft" || draft.Conflicts != "Merge conflicts with main" {
		t.Errorf("a conflicting draft: %+v", draft)
	}
	if merged := prTip(mergeable(func(d *t3.PullRequestDetail) { d.State = "merged" }), "merged", nil); merged.Checks != "" || merged.Conflicts != "" {
		t.Errorf("a merged one has no checks line: %+v", merged)
	}

	a, f := prApp(t, func() *t3.PullRequestDetail {
		return mergeable(func(d *t3.PullRequestDetail) {
			d.BaseBranch, d.HeadBranch = "main", "parser-tabs"
			d.ChangedFiles, d.Additions, d.Deletions = 3, 1200, 45
			d.Checks = []t3.PullRequestCheck{check("success"), check("pending")}
		})
	})
	f.on("pullRequests.changeBreakdown", func(f *fakeEnv, r fakeReq) {
		lines := func(add, del int) map[string]any { return map[string]any{"additions": add, "deletions": del} }
		f.exit(r, map[string]any{"code": lines(1100, 40), "tests": lines(100, 5), "mocks": lines(0, 0), "generated": lines(0, 0)})
	})
	tip := prRow(t, a).Rich
	if tip == nil || tip.Title != "Teach the parser about tabs" || tip.Number != 12 || tip.State != "open" || tip.StateLabel != "Open" ||
		tip.Branches != "main ← parser-tabs" || tip.Checks != "1 of 2 running" || tip.ChecksState != "pending" ||
		tip.Files != 3 || tip.Insertions != 1200 || tip.Deletions != 45 || tip.Conflicts != "" || tip.Kinds != nil {
		t.Fatalf("tip %+v", tip)
	}
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.pr:0", Kind: "tip"})
	a.handle(th, ui.Action{ID: "card.pr:0", Kind: "tip"})
	if b := f.wait("pullRequests.changeBreakdown", 1).Payload; b["number"] != float64(12) || b["repository"] != "acme/widgets" || b["projectId"] != "p1" {
		t.Fatalf("breakdown read %v", b)
	}
	eventually(t, "the lines by kind", func() bool {
		k := built(a, time.Now()).Card.PRs.Rows[0].Rich.Kinds
		return len(k) == 2 && k[0] == (ui.LineKind{Label: "Code", Insertions: 1100, Deletions: 40}) && k[1] == (ui.LineKind{Label: "Tests", Insertions: 100, Deletions: 5})
	})
	if n := len(f.requests("pullRequests.changeBreakdown")); n != 1 {
		t.Errorf("%d breakdown reads, want one", n)
	}
}

// prApp is liveApp with the open thread's pull request linked and its
// project checked out from its repository, on a server that reads checks
// alone; detail answers pullRequests.detail.
func prApp(t *testing.T, detail func() *t3.PullRequestDetail) (*App, *fakeEnv) {
	t.Helper()
	a, f := liveApp(t, status(nil))
	f.on("pullRequests.detail", func(f *fakeEnv, r fakeReq) { f.exit(r, detail()) })
	f.on("pullRequests.checks", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	a.mu.Lock()
	e := a.envs[0]
	e.config.Environment.Capabilities.PullRequestChecks = true
	p := e.projects["p1"]
	p.RepositoryIdentity = &t3.RepositoryIdentity{Provider: "github", Owner: "acme", Name: "widgets"}
	e.projects["p1"] = p
	e.threads["t1"].PullRequests = []t3.PullRequestLink{{Host: "github.com", Repository: "acme/widgets", Number: 12,
		URL: "https://github.com/acme/widgets/pull/12", Snapshot: &t3.PullRequestSnapshot{State: "open", Title: "Teach the parser about tabs"}}}
	a.mu.Unlock()
	return a, f
}

// prRow is the scene's first pull request row once its detail is in.
func prRow(t *testing.T, a *App) ui.PullRequestRow {
	t.Helper()
	var r ui.PullRequestRow
	eventually(t, "the pull request's detail", func() bool {
		s := built(a, time.Now())
		if s.Card.PRs == nil || len(s.Card.PRs.Rows) == 0 || s.Card.PRs.Rows[0].Detail == nil {
			return false
		}
		r = s.Card.PRs.Rows[0]
		return true
	})
	return r
}

// The row reads its pull request's detail fresh from the host, by the
// link's host, repository and number in the project checked out from it,
// then its checks; it comes back detailed, with Merge. Merge asks in T3's
// alert dialog, and only its Merge runs pullRequests.runAction, the pill
// saying "Merging..." until the host answers; a refusal goes to the error
// banner under T3's title, and the detail is read again.
func TestMergePullRequest(t *testing.T) {
	a, f := prApp(t, func() *t3.PullRequestDetail { return mergeable(nil) })
	r := prRow(t, a)
	d := f.wait("pullRequests.detail", 1).Payload
	if d["projectId"] != "p1" || d["host"] != "github.com" || d["repository"] != "acme/widgets" || d["number"] != float64(12) || d["allowStale"] != false {
		t.Fatalf("detail read %v", d)
	}
	if c := f.wait("pullRequests.checks", 1).Payload; c["number"] != float64(12) || c["allowStale"] != nil {
		t.Fatalf("checks read %v", c)
	}
	if r.Detail.Title != "Teach the parser about tabs" || r.Detail.Number != 12 || r.State != "open" {
		t.Errorf("row %+v %+v", r, r.Detail)
	}
	if c := r.Detail.Checks; c == nil || c.State != "passing" || c.Count != "" {
		t.Errorf("checks %+v", r.Detail.Checks)
	}
	if p := r.Detail.Pill; p == nil || p.Label != "Merge" || p.Tip != "Merge this pull request (merge)" || p.Disabled {
		t.Fatalf("pill %+v", r.Detail.Pill)
	}

	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.prpill:0", Kind: "click"})
	s := built(a, time.Now())
	if al := s.Alert; al == nil || al.Title != "Merge pull request?" || al.Description != "This merges #12 using merge." || al.Action != "Merge" {
		t.Fatalf("alert %+v", s.Alert)
	}
	a.handle(th, ui.Action{ID: "alert:cancel", Kind: "click"})
	if s := built(a, time.Now()); s.Alert != nil {
		t.Fatal("Cancel should close the dialog")
	}
	held := make(chan fakeReq, 1)
	f.on("pullRequests.runAction", func(f *fakeEnv, r fakeReq) { held <- r })
	a.handle(th, ui.Action{ID: "card.prpill:0", Kind: "click"})
	a.handle(th, ui.Action{ID: "alert:ok", Kind: "click"})
	run := <-held
	if p := run.Payload; p["action"] != "merge" || p["mergeMethod"] != "merge" || p["projectId"] != "p1" || p["repository"] != "acme/widgets" || p["number"] != float64(12) {
		t.Fatalf("runAction %v", run.Payload)
	}
	if p := built(a, time.Now()).Card.PRs.Rows[0].Detail.Pill; p.Label != "Merging..." || !p.Disabled {
		t.Errorf("while merging: %+v", p)
	}
	reads := len(f.requests("pullRequests.detail"))
	f.failWith(run, "Pull request is not mergeable")
	eventually(t, "the refusal and a fresh read", func() bool {
		s := built(a, time.Now())
		return strings.HasPrefix(s.Error, "Could not merge this pull request: ") && len(f.requests("pullRequests.detail")) > reads
	})
	if p := prRow(t, a).Detail.Pill; p.Label != "Merge" || p.Disabled {
		t.Errorf("after: %+v", p)
	}
}

// A draft offers Ready, which marks it at once; there's no dialog.
func TestMarkPullRequestReady(t *testing.T) {
	a, f := prApp(t, func() *t3.PullRequestDetail {
		return mergeable(func(d *t3.PullRequestDetail) { d.IsDraft = true })
	})
	r := prRow(t, a)
	if p := r.Detail.Pill; p == nil || p.Label != "Ready" || p.Tip != "Mark this pull request as ready for review" || r.State != "draft" {
		t.Fatalf("draft row %+v, pill %+v", r, r.Detail.Pill)
	}
	if r.Detail.Checks != nil {
		t.Error("a draft's checks stay out of its meta line")
	}
	f.on("pullRequests.runAction", func(f *fakeEnv, r fakeReq) { f.exit(r, nil) })
	a.handle(ui.NewTheme(), ui.Action{ID: "card.prpill:0", Kind: "click"})
	if p := f.wait("pullRequests.runAction", 1).Payload; p["action"] != "ready" || p["mergeMethod"] != nil {
		t.Fatalf("runAction %v", p)
	}
	if s := built(a, time.Now()); s.Alert != nil {
		t.Error("Ready shouldn't ask")
	}
}

// The checks trigger opens their popover: its heading and summary, the
// checks that want attention and the running ones, the finished folded
// behind Show all; a check's Details opens its page.
func TestChecksPopover(t *testing.T) {
	failed := check("failure")
	failed.URL = "https://ci.example.test/runs/7"
	a, _ := prApp(t, func() *t3.PullRequestDetail {
		return mergeable(func(d *t3.PullRequestDetail) {
			d.Checks = []t3.PullRequestCheck{check("success"), failed, check("pending")}
		})
	})
	var opened []string
	a.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	if c := prRow(t, a).Detail.Checks; c.State != "failing" || c.Count != "2/3 checks" || c.Popup != nil {
		t.Fatalf("trigger %+v", c)
	}
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "card.checks:0", Kind: "click"})
	pop := built(a, time.Now()).Card.PRs.Rows[0].Detail.Checks.Popup
	if pop == nil || pop.Heading != "Some checks were not successful" || pop.Summary != "1 of 3 failing" || pop.More != "Show all" || len(pop.Checks) != 2 {
		t.Fatalf("popover %+v", pop)
	}
	if c := pop.Checks[0]; c.Status != "failure" || c.Label != "Failed" || !c.Details {
		t.Errorf("first check %+v", c)
	}
	if c := pop.Checks[1]; c.Status != "pending" || c.Label != "Running" || c.Details {
		t.Errorf("second check %+v", c)
	}
	a.handle(th, ui.Action{ID: "card.checksall", Kind: "click"})
	if pop := built(a, time.Now()).Card.PRs.Rows[0].Detail.Checks.Popup; len(pop.Checks) != 3 || pop.More != "Show less" {
		t.Errorf("unfolded %+v", pop)
	}
	a.handle(th, ui.Action{ID: "card.check:0", Kind: "click"})
	if len(opened) != 1 || opened[0] != failed.URL {
		t.Errorf("opened %v", opened)
	}
	a.handle(th, ui.Action{ID: "menus.close", Kind: "click"})
	if c := built(a, time.Now()).Card.PRs.Rows[0].Detail.Checks; c.Popup != nil {
		t.Error("a click outside should close the popover")
	}
}
