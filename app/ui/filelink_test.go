package ui

import (
	"image"
	"testing"
)

// Backticked paths that name files, after T3's inlineCodeFilePathCandidate
// and its markdownLinks.test.ts cases.
func TestInlineCodePath(t *testing.T) {
	for code, want := range map[string]string{
		"src/send.go":      "src/send.go",
		"send.go:12":       "send.go:12",
		"./run.sh":         "./run.sh",
		"~/notes/todo.md":  "~/notes/todo.md",
		`src\win\a.ts`:     "src/win/a.ts",
		"/etc/hosts":       "/etc/hosts",
		"example.com/path": "",
		"v1.2.3":           "",
		"go test ./...":    "",
		"upload/v2":        "",
		"I/W217:220":       "",
		"src/.../send.go":  "",
		"lib/bar-1.2.0":    "",
	} {
		if got := inlineCodePath(code); got != want {
			t.Errorf("inlineCodePath(%q) = %q, want %q", code, got, want)
		}
	}
}

// File links resolve in the workspace and keep their place in the file;
// web links, fragments and relative paths without a workspace aren't
// files (parseMarkdownFileLink, resolveMarkdownFileLinkTarget).
func TestFileLinkTarget(t *testing.T) {
	for _, c := range []struct {
		href, cwd string
		want      filePos
		ok        bool
	}{
		{"src/send.go#L12", "/w", filePos{path: "/w/src/send.go", line: 12}, true},
		{"src/send.go:12-20", "/w/", filePos{path: "/w/src/send.go", line: 12, endLine: 20}, true},
		{"file:///home/me/a%20b.go", "", filePos{path: "/home/me/a b.go"}, true},
		{"/api/users.json", "", filePos{path: "/api/users.json"}, true},
		{"~/x/y.txt", "/home/me/w", filePos{path: "/home/me/x/y.txt"}, true},
		{"src/send.go", "", filePos{}, false},
		{"https://example.com/a.go", "/w", filePos{}, false},
		{"#heading", "/w", filePos{}, false},
		{"//host/share/a.go", "/w", filePos{}, false},
	} {
		got, ok := fileLinkTarget(c.href, c.cwd)
		if ok != c.ok || ok && got != c.want {
			t.Errorf("fileLinkTarget(%q, %q) = %+v %v, want %+v %v", c.href, c.cwd, got, ok, c.want, c.ok)
		}
	}
}

// A chip's label is the file's name, the parents that tell it from
// another of the message's files of that name (two at least), and its
// lines (fileLinkLabel, buildFileLinkParentSuffixByPath).
func TestFileLabel(t *testing.T) {
	suffixes := fileLabelSuffixes([]string{"/w/a/x/send.go", "/w/b/x/send.go", "/w/c/recv.go"})
	for _, c := range []struct {
		p    filePos
		want string
	}{
		{filePos{path: "/w/a/x/send.go", line: 12, endLine: 20}, "send.go · a/x · L12–20"},
		{filePos{path: "/w/b/x/send.go", line: 3, column: 4}, "send.go · b/x · L3:C4"},
		{filePos{path: "/w/c/recv.go"}, "recv.go"},
	} {
		if got := fileLinkLabel(c.p, suffixes); got != c.want {
			t.Errorf("fileLinkLabel(%+v) = %q, want %q", c.p, got, c.want)
		}
	}
	for _, c := range []struct {
		label, href string
		want        bool
	}{{"src/send.go", "src/send.go", true}, {"send.go", "src/send.go", true}, {"the sender", "src/send.go", false},
		{"send.go:12", "src/send.go:13", false}} {
		if got := isFileLabel(c.label, c.href); got != c.want {
			t.Errorf("isFileLabel(%q, %q) = %v", c.label, c.href, got)
		}
	}
}

// A path in backticks is a file chip, not code: medium, in
// --context-chip-mention (#006d80 on the light canvas), after its file's
// icon, without the code's #e5e5e5 box (§3.5).
func TestFileChip(t *testing.T) {
	st := mdBody()
	st.cwd = "/w"
	img := mdDrawn(t, "See `src/send.go:12` now", st, 400, 30, 2)
	mention, code := rgb(0x006d80), rgb(0xe5e5e5)
	chip, box := false, false
	for y := 0; y < 2*23; y++ {
		for x := 0; x < 2*400; x++ {
			c := img.RGBAAt(x, y)
			chip = chip || near(c.R, mention.R, 8) && near(c.G, mention.G, 8) && near(c.B, mention.B, 8)
			box = box || near(c.R, code.R, 2) && near(c.G, code.G, 2) && near(c.B, code.B, 2)
		}
	}
	if !chip || box {
		t.Errorf("a chip's colour drawn %v, a code box drawn %v", chip, box)
	}
}

// Clicking a chip opens the menu T3's chip opens where it can't open the
// file: "Copy relative path", then "Copy full path".
func TestFileChipMenu(t *testing.T) {
	s := &Scene{Timeline: Timeline{ID: "chip", Root: "/w", Rows: []TimelineRow{{ID: "a", Kind: RowAssistant,
		Message: &Message{ID: "a", Text: "`src/send.go:12`"}}}}}
	click, r := timelineRouter(NewTheme(), s)
	// The paragraph starts 2dp into the row, under the 24dp header; its
	// chip's box is 2 to 19dp down its line, the menu 4dp under that,
	// padded 4, its rows 28dp.
	for i, want := range []string{"src/send.go:12", "/w/src/send.go:12"} {
		click(image.Pt(32+4+8, 24+2+10))
		if acts := click(image.Pt(32+40, 24+2+19+4+4+28*i+14)); len(acts) != 0 {
			t.Fatalf("a copy reached core: %+v", acts)
		}
		if _, data, ok := r.WriteClipboard(); !ok || string(data) != want {
			t.Errorf("row %d copied %q %v, want %q", i, data, ok, want)
		}
	}
}
