package diff

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const patch = `diff --git a/src/parser.ts b/src/parser.ts
index 3b18e51..a9c2f4d 100644
--- a/src/parser.ts
+++ b/src/parser.ts
@@ -37,5 +37,6 @@ export function parse(input: string) {
 const tokens = lex(input);
-const tree = build(tokens);
+const tree = build(tokens, options);
+check(tree);
 return tree;
 }

@@ -90,2 +91,2 @@ function lex(input: string) {
-  return [];
+  return out;
 }
\ No newline at end of file
diff --git a/docs/notes.md b/docs/notes.md
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/docs/notes.md
@@ -0,0 +1,2 @@
+# Notes
+First line.
diff --git a/old.txt b/old.txt
deleted file mode 100644
--- a/old.txt
+++ /dev/null
@@ -1 +0,0 @@
-gone
diff --git a/lib/a.go b/lib/b.go
similarity index 100%
rename from lib/a.go
rename to lib/b.go
diff --git "a/caf\303\251 menu.txt" "b/caf\303\251 menu.txt"
--- "a/caf\303\251 menu.txt"
+++ "b/caf\303\251 menu.txt"
@@ -1 +1 @@
-tea
+coffee
`

// A Git patch reads as Pierre reads it: each file's path and change type,
// its counts, the unchanged lines above each hunk, and its lines with
// their numbers, each change's deletions before its additions.
func TestParse(t *testing.T) {
	files, err := Parse(patch)
	if err != nil {
		t.Fatal(err)
	}
	type summary struct {
		name, prev string
		typ        ChangeType
		add, del   int
		hidden     []int
	}
	var got []summary
	for _, f := range files {
		s := summary{f.Name, f.PrevName, f.Type, f.Additions, f.Deletions, nil}
		for _, h := range f.Hunks {
			s.hidden = append(s.hidden, h.Hidden)
		}
		got = append(got, s)
	}
	want := []summary{
		{"src/parser.ts", "", Change, 3, 2, []int{36, 48}},
		{"docs/notes.md", "", New, 2, 0, []int{0}},
		{"old.txt", "", Deleted, 0, 1, []int{0}},
		{"lib/b.go", "lib/a.go", RenamePure, 0, 0, nil},
		{"café menu.txt", "", Change, 1, 1, []int{0}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files\n got %+v\nwant %+v", got, want)
	}
	var lines []string
	for _, l := range files[0].Hunks[0].Lines {
		lines = append(lines, string(" -+"[l.Kind])+strconv.Itoa(l.Old)+":"+strconv.Itoa(l.New)+" "+l.Text)
	}
	wantLines := []string{
		" 37:37 const tokens = lex(input);",
		"-38:0 const tree = build(tokens);",
		"+0:38 const tree = build(tokens, options);",
		"+0:39 check(tree);",
		" 39:40 return tree;",
		" 40:41 }",
		" 41:42 ",
	}
	if !reflect.DeepEqual(lines, wantLines) {
		t.Fatalf("lines\n got %q\nwant %q", lines, wantLines)
	}
	last := files[0].Hunks[1].Lines
	if !last[len(last)-1].NoNewline || last[0].NoNewline {
		t.Errorf("only the last line has no newline after it: %+v", last)
	}
}

// A file moved and edited is rename-changed; a patch with nothing in it
// is no files.
func TestParseRenameChangedAndEmpty(t *testing.T) {
	files, err := Parse("diff --git a/x.ts b/y.ts\nsimilarity index 80%\nrename from x.ts\nrename to y.ts\n--- a/x.ts\n+++ b/y.ts\n@@ -1 +1 @@\n-a\n+b\n")
	if err != nil || len(files) != 1 || files[0].Type != RenameChanged || files[0].PrevName != "x.ts" || files[0].Name != "y.ts" {
		t.Fatalf("got %+v, %v", files, err)
	}
	if _, err := Parse("not a patch\n"); err != ErrNoFiles {
		t.Fatalf("a text with no files: %v", err)
	}
}

// Paired lines mark what changed between them as word-alt runs: a
// one-character unchanged gap after a change joins it, so "foo(a" is one
// run, and the closing ")" stays plain.
func TestWords(t *testing.T) {
	for _, c := range []struct {
		del, add string
		dr, ar   []Range
	}{
		{"const tree = build(tokens);", "const tree = build(tokens, options);", nil, []Range{{25, 34}}},
		{"a b c", "a x c", []Range{{2, 3}}, []Range{{2, 3}}},
		{"foo(a)", "bar(b)", []Range{{0, 5}}, []Range{{0, 5}}},
		{"same", "same", nil, nil},
	} {
		dr, ar := Words(c.del, c.add)
		if !reflect.DeepEqual(dr, c.dr) || !reflect.DeepEqual(ar, c.ar) {
			t.Errorf("%q → %q: %v, %v; want %v, %v", c.del, c.add, dr, ar, c.dr, c.ar)
		}
	}
	// Parse pairs a change's lines in order.
	files, _ := Parse(patch)
	add := files[0].Hunks[0].Lines[2]
	if !reflect.DeepEqual(add.Words, []Range{{25, 34}}) {
		t.Errorf("the paired addition's words are %v", add.Words)
	}
	if w := files[0].Hunks[0].Lines[3].Words; w != nil {
		t.Errorf("an addition with no deletion to pair has words %v", w)
	}
}

// Lines further apart than the word diff will go change as a whole.
func TestWordsWholeLine(t *testing.T) {
	a := strings.Repeat("a ", 300)
	b := strings.Repeat("b ", 300)
	dr, ar := Words(a, b)
	if len(dr) == 0 || len(ar) == 0 {
		t.Fatalf("got %v, %v", dr, ar)
	}
}

// Tests and test support are tests; generated files never are.
func TestIsTest(t *testing.T) {
	for path, want := range map[string]bool{
		"src/parser.test.ts":        true,
		"pkg/api_test.go":           true,
		"tests/e2e.py":              true,
		"app/FooTest.java":          true,
		"app/Latest.java":           false,
		"src/__mocks__/fs.ts":       true,
		"src/mockServer.ts":         true,
		"fixtures.json":             true,
		"ui/__snapshots__/x.snap":   true,
		"conftest.py":               true,
		"test/generated/schema.ts":  false,
		"go.sum":                    false,
		"src/parser.ts":             false,
		"docs/testing.md":           false,
		"internal/testdata/in.json": true,
	} {
		if got := IsTest(path); got != want {
			t.Errorf("IsTest(%q) = %v", path, got)
		}
	}
}

// The code view sorts its files by full path, numbers by value and case
// aside, so a.txt comes before b/c.ts and c.ts before d/x.ts.
func TestSortPaths(t *testing.T) {
	paths := []string{"d/x.ts", "file10.ts", "c.ts", "B/c.ts", "file9.ts", "a.txt"}
	SortPaths(paths)
	want := []string{"a.txt", "B/c.ts", "c.ts", "d/x.ts", "file9.ts", "file10.ts"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("got %q, want %q", paths, want)
	}
}

// The card puts folders first, each level by name, and folds a chain of
// lone folders into one "a/b/c" row with the deepest one's totals.
func TestCardTree(t *testing.T) {
	nodes := CardTree([]FileStat{
		{"root.md", 1, 0},
		{"a/z.ts", 2, 1},
		{"a/b/c/y.ts", 3, 0},
		{"a/b/c/x.ts", 4, 2},
	})
	var rows []string
	var walk func(ns []CardNode, depth int)
	walk = func(ns []CardNode, depth int) {
		for _, n := range ns {
			rows = append(rows, strings.Repeat("  ", depth)+n.Name+" +"+strconv.Itoa(n.Additions)+" -"+strconv.Itoa(n.Deletions))
			walk(n.Children, depth+1)
		}
	}
	walk(nodes, 0)
	want := []string{
		"a +9 -3",
		"  b/c +7 -2",
		"    x.ts +4 -2",
		"    y.ts +3 -0",
		"  z.ts +2 -1",
		"root.md +1 -0",
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	if nodes[0].Children[0].Path != "a/b/c" {
		t.Errorf("the folded folder's path is %q", nodes[0].Children[0].Path)
	}
}

// The file tree keeps the code view's order, each folder where its first
// file is, flattens lone folder chains, and leaves closed folders' files
// out.
func TestFileTree(t *testing.T) {
	paths := []string{"a.txt", "b/c.ts", "c.ts", "s/com/u/file.ts", "s/com/u/more.ts"}
	types := map[string]ChangeType{"a.txt": New, "b/c.ts": Change, "c.ts": Deleted, "s/com/u/file.ts": Change, "s/com/u/more.ts": RenamePure}
	got := FileTree(paths, types, nil)
	var rows []string
	for _, e := range got {
		rows = append(rows, strings.Repeat("  ", e.Depth)+strings.Join(e.Segments, " / "))
	}
	want := []string{"a.txt", "b", "  c.ts", "c.ts", "s / com / u", "  file.ts", "  more.ts"}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("got %q, want %q", rows, want)
	}
	if got[0].Type != New || got[3].Type != Deleted || !got[4].Dir || got[4].Path != "s/com/u/" {
		t.Errorf("rows %+v", got)
	}
	got = FileTree(paths, types, map[string]bool{"s/com/u/": true})
	if len(got) != 5 || got[4].Open {
		t.Errorf("a closed folder still lists its files: %+v", got)
	}
}
