package main

import (
	"sort"

	"gioui.org/layout"

	"github.com/some-du6e/tern/app/diff"
	"github.com/some-du6e/tern/app/ui"
)

// diffShots are the Diff surface's mockups, light and dark; the tree's
// with two lines selected.
func diffShots() []shot {
	draw := func(scene func() *ui.Scene) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
		return func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, scene()) }
	}
	dark := func(scene func() *ui.Scene) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
		return func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			ui.SetDark(true)
			defer ui.SetDark(false)
			return th.Desktop(gtx, scene())
		}
	}
	selected := func(draw func(th *ui.Theme, gtx layout.Context) layout.Dimensions) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
		return func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			th.SelectDiffLines("app/t3/rpc.go", 4, 6)
			defer th.SelectDiffLines("", 0, 0)
			return draw(th, gtx)
		}
	}
	return []shot{
		{"3-diff", 1440, 900, draw(diffScene)},
		{"3b-diff-tree", 1440, 900, selected(draw(diffTreeScene))},
		{"3c-diff-scope-menu", 1440, 900, draw(diffMenuScene)},
		{"3d-diff-commits", 1440, 900, draw(diffCommitsScene)},
		{"3e-diff-split", 1440, 900, draw(diffSplitScene)},
		{"3f-diff-loading", 1440, 900, draw(diffLoadingScene)},
		{"3g-diff-no-access", 1440, 900, draw(diffAccessScene)},
		{"3h-diff-tree-dark", 1440, 900, selected(dark(diffTreeScene))},
		{"3i-diff-split-dark", 1440, 900, dark(diffSplitScene)},
		{"3j-changed-files", 1440, 900, draw(changedFilesScene)},
		{"3k-changed-files-dark", 1440, 900, dark(changedFilesScene)},
	}
}

// changedFilesScene is a reply with its run's changed files card, the
// Diff surface closed.
func changedFilesScene() *ui.Scene {
	s := runningScene()
	items := threadItems()
	reply := items[2]
	reply.Files = changedFiles()
	s.Items, s.MoreAbove = []ui.Item{items[0], items[1], reply}, false
	return s
}

// The Diff surface's mockups: a turn's patch in the right panel, its
// menus, the file tree, split view, loading and an access error; and the
// changed files card under the reply.

// turnPatch is a made-up turn: an edit with word changes, a new test, a
// rename, a deletion and two hunks with unchanged lines between.
const turnPatch = `diff --git a/app/t3/rpc.go b/app/t3/rpc.go
index 1a2b3c4..5d6e7f8 100644
--- a/app/t3/rpc.go
+++ b/app/t3/rpc.go
@@ -104,16 +104,23 @@ func (c *Conn) readLoop() error {
 func (c *Conn) readLoop() error {
 	defer c.shutdown()
 	for {
-		_, data, err := c.ws.Read(c.ctx)
+		ctx, cancel := context.WithTimeout(c.ctx, pongTimeout)
+		_, data, err := c.ws.Read(ctx)
+		cancel()
 		if err != nil {
-			return err
+			// No frame for 15 s: a laptop that slept leaves a socket
+			// that never closes. Drop it and resubscribe.
+			return c.closeWith(errNoPong)
 		}
 		msgs, err := decodeFrame(data)
 		if err != nil {
 			return fmt.Errorf("decode frame: %w", err)
 		}
 		for _, m := range msgs {
-			c.dispatch(m)
+			if m.Tag == "Chunk" {
+				c.send(ack{Tag: "Ack", RequestID: m.RequestID})
+			}
+			c.dispatch(m)
 		}
 	}
 }
@@ -201,3 +208,5 @@ func (c *Conn) Stream(tag string, payload any) *Stream {
 	id := c.nextID.Add(1)
+	st := &Stream{id: id, values: make(chan json.RawMessage, 64)}
+	c.pending.Store(id, st)
 	return st
 }
diff --git a/app/t3/relay.go b/app/t3/relay.go
--- a/app/t3/relay.go
+++ b/app/t3/relay.go
@@ -40,6 +40,6 @@ func (r *Relay) token() (string, error) {
 	r.mu.Lock()
 	defer r.mu.Unlock()
-	if time.Until(r.expires) > 0 {
+	if time.Until(r.expires) > time.Minute {
 		return r.cached, nil
 	}
 	tok, err := r.fetch()
diff --git a/app/t3/dpop_test.go b/app/t3/dpop_test.go
new file mode 100644
--- /dev/null
+++ b/app/t3/dpop_test.go
@@ -0,0 +1,6 @@
+package t3
+
+import "testing"
+
+// The thumbprint pins RFC 7638's member order.
+func TestThumbprint(t *testing.T) {}
diff --git a/poc/README.md b/docs/poc.md
similarity index 100%
rename from poc/README.md
rename to docs/poc.md
diff --git a/app/old/shim.go b/app/old/shim.go
deleted file mode 100644
--- a/app/old/shim.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package old
-
-func Shim() {}
`

// turnFiles is turnPatch read and sorted as the code view lists it.
func turnFiles() []diff.File {
	files, err := diff.Parse(turnPatch)
	if err != nil {
		panic(err)
	}
	sort.SliceStable(files, func(i, j int) bool { return diff.Compare(files[i].Path(), files[j].Path()) < 0 })
	return files
}

// diffPanel is the Diff surface on the latest turn, rpc.go and relay.go
// unfolded.
func diffPanel() *ui.DiffPanel {
	p := &ui.DiffPanel{Active: true, Key: "mock", Scope: "Latest turn", Value: "latest", Wrap: true, Whitespace: true}
	var paths []string
	types := map[string]diff.ChangeType{}
	stat := ui.DiffStat{}
	for _, f := range turnFiles() {
		open := f.Name == "app/t3/rpc.go" || f.Name == "app/t3/relay.go"
		p.Files = append(p.Files, ui.DiffFile{File: f, Collapsed: !open})
		paths = append(paths, f.Path())
		types[f.Path()] = f.Type
		stat.Additions += f.Additions
		stat.Deletions += f.Deletions
		if diff.IsTest(f.Path()) {
			p.TestFiles++
		}
	}
	p.Stat = &stat
	p.Tree = diff.FileTree(paths, types, nil)
	return p
}

// changedFiles is the card under the reply: the turn's files, its folders
// open.
func changedFiles() *ui.ChangedFiles {
	c := &ui.ChangedFiles{Run: "run-4", AllOpen: true}
	var stats []diff.FileStat
	for _, f := range turnFiles() {
		stats = append(stats, diff.FileStat{Path: f.Path(), Additions: f.Additions, Deletions: f.Deletions})
		c.Count++
		c.Additions += f.Additions
		c.Deletions += f.Deletions
	}
	c.Nodes = diff.CardTree(stats)
	return c
}

func diffScene() *ui.Scene {
	s := runningScene()
	s.Diff = diffPanel()
	s.Items[2].Files = changedFiles()
	return s
}

// diffTreeScene has the file tree open beside unwrapped lines.
func diffTreeScene() *ui.Scene {
	s := diffScene()
	s.Diff.TreeOpen, s.Diff.Wrap, s.Diff.Revealed = true, false, "app/t3/rpc.go"
	return s
}

// diffMenuScene has the scope menu open on its Turn submenu.
func diffMenuScene() *ui.Scene {
	s := diffScene()
	s.Diff.Menu = &ui.DiffScopeMenu{Sub: "turn", Turn: "turn:run-4", Turns: []ui.DiffTurn{
		{Value: "turn:run-4", Label: "Turn 4", Time: "2:00 PM"},
		{Value: "turn:run-3", Label: "Turn 3", Time: "1:42 PM"},
		{Value: "turn:run-2", Label: "Turn 2", Time: "1:15 PM"},
		{Value: "turn:run-1", Label: "Turn 1", Time: "12:58 PM"},
	}}
	return s
}

// diffCommitsScene has the Commits submenu open on a connection that
// can't read files.
func diffCommitsScene() *ui.Scene {
	s := diffScene()
	s.Diff.Menu = &ui.DiffScopeMenu{Sub: "commits", Note: "This connection cannot read local diffs."}
	return s
}

// diffSplitScene shows the files side by side.
func diffSplitScene() *ui.Scene {
	s := diffScene()
	s.Diff.Split = true
	return s
}

// diffLoadingScene is a turn's patch on its way.
func diffLoadingScene() *ui.Scene {
	s := diffScene()
	d := &ui.DiffPanel{Active: true, Key: "mock", Scope: "Turn 3", Value: "turn:run-3", Wrap: true, Whitespace: true,
		Loading: "Loading checkpoint diff..."}
	s.Diff = d
	return s
}

// diffAccessScene is Changes on a connection without file access.
func diffAccessScene() *ui.Scene {
	s := diffScene()
	s.Diff = &ui.DiffPanel{Active: true, Key: "mock", Scope: "Changes", Value: "branch", Refresh: true, Wrap: true, Whitespace: true,
		Empty: "This connection cannot read local diffs."}
	return s
}
