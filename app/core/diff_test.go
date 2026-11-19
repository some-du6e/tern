package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// diffView is a thread view with turns 1 to 3 checkpointed, the newest
// first in neither order nor ID.
func diffView() *threadView {
	v := newView(nil, "t1")
	at := time.Now().UTC()
	for i, run := range []string{"r2", "r3", "r1"} {
		n := map[string]int{"r1": 1, "r2": 2, "r3": 3}[run]
		v.checkpoints = append(v.checkpoints, t3.Checkpoint{ID: "cp" + run, RunID: run, AppRunOrdinal: n,
			CapturedAt: at.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano)})
	}
	return v
}

// The scope trigger reads as T3's (§5.2): the newest turn is "Latest
// turn", another "Turn N", a vanished one falls back to the newest; a
// commit is "Commit" until its subject comes; Git scopes by name.
func TestScopeLabel(t *testing.T) {
	ts := turns(diffView())
	for _, c := range []struct {
		sel         diffSel
		src         *t3.DiffSource
		label, want string
	}{
		{diffSel{Kind: "turn", Run: "r3"}, nil, "Latest turn", "latest"},
		{diffSel{Kind: "turn", Run: "r2"}, nil, "Turn 2", "turn:r2"},
		{diffSel{Kind: "turn", Run: "gone"}, nil, "Latest turn", "latest"},
		{diffSel{Kind: "commit", SHA: "abcdef123456"}, nil, "Commit", "commit:abcdef123456"},
		{diffSel{Kind: "commit", SHA: "abcdef123456"}, &t3.DiffSource{HeadRef: "abcdef123456", CommitSubject: "Fix the relay"}, "Fix the relay", "commit:abcdef123456"},
		{diffSel{Kind: "branch"}, nil, "Changes", "branch"},
		{diffSel{Kind: "unstaged"}, nil, "Uncommitted", "unstaged"},
	} {
		ds := &diffState{sel: c.sel, source: c.src}
		p := &ui.DiffPanel{}
		scopeLabel(p, ds, selectedTurn(c.sel, ts), ts)
		if p.Scope != c.label || p.Value != c.want {
			t.Errorf("%+v: %q %q, want %q %q", c.sel, p.Scope, p.Value, c.label, c.want)
		}
	}
	p := &ui.DiffPanel{}
	scopeLabel(p, &diffState{sel: diffSel{Kind: "commit", SHA: "abcdef123456"}}, nil, ts)
	if p.SHA != "abcdef1" {
		t.Errorf("the commit's SHA after the trigger is %q", p.SHA)
	}
}

// The changed files card goes under a run's last reply only, with the
// checkpoint's totals; a run without files has none.
func TestChangedFilesCards(t *testing.T) {
	a := testApp(t)
	a.openID = "swiss/t1"
	v := diffView()
	v.checkpoints[0].Files = []t3.CheckpointFile{{Path: "a/x.go", Additions: 3, Deletions: 1}, {Path: "b.go", Additions: 2}}
	v.checkpoints[0].FilesTotal = &t3.CheckpointTotals{Count: 5, Additions: 40, Deletions: 9}
	for i, it := range []map[string]any{
		{"id": "u", "type": "user_message", "runId": "r2", "text": "go"},
		{"id": "a1", "type": "assistant_message", "runId": "r2", "messageId": "m1", "text": "first"},
		{"id": "a2", "type": "assistant_message", "runId": "r2", "messageId": "m2", "text": "last"},
		{"id": "b", "type": "assistant_message", "runId": "r3", "messageId": "m3", "text": "no files"},
	} {
		it["ordinal"] = i
		raw, _ := json.Marshal(it)
		v.put(raw)
	}
	cards := a.changedFilesLocked(v, v.sorted())
	if len(cards) != 1 || cards["m:m2"] == nil {
		t.Fatalf("cards %v, want one under m2", cards)
	}
	c := cards["m:m2"]
	if c.Run != "r2" || c.Count != 5 || c.Additions != 40 || c.Deletions != 9 || len(c.Nodes) != 2 {
		t.Errorf("card %+v", c)
	}
}

// Opening a card's file selects its turn and unfolds and reveals the
// file; collapse all then folds every file, and again unfolds them.
func TestDiffActions(t *testing.T) {
	a := testApp(t)
	a.openID = "swiss/t1"
	a.view = diffView()
	if !a.diffAction("files.file:r2:a/x.go") {
		t.Fatal("not a Diff target")
	}
	ds := a.diffFor(a.openID)
	if !ds.open || !ds.active || ds.sel.Kind != "turn" || ds.sel.Run != "r2" || ds.revealed != "a/x.go" || ds.folds["turn:r2"]["a/x.go"] {
		t.Fatalf("after opening a file: %+v", ds)
	}
	ds.files = nil
	a.diffAction("diff.scope:unstaged")
	if ds.sel.Kind != "unstaged" || ds.menu {
		t.Errorf("after Uncommitted: %+v", ds.sel)
	}
	a.diffAction("diff.toggle")
	if ds.open {
		t.Error("mod+D on the shown surface left it open")
	}
}
