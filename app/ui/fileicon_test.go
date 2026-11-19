package ui

import (
	"image"
	"testing"
)

// Files get T3's icons by name, then by each extension the name could
// have, T3's own picks first (pierre-icons.ts, @pierre/trees).
func TestFileIcon(t *testing.T) {
	for _, c := range []struct{ path, name, token string }{
		{"main.go", "file-tree-builtin-go", "go"},
		{"web/src/App.tsx", "file-tree-builtin-react", "react"},
		{"lib/a.test.ts", "file-tree-builtin-typescript", "typescript"},
		{"docs/README.md", "file-tree-builtin-markdown", "markdown"},
		{".gitignore", "file-tree-builtin-git", "git"},
		{"package.json", "file-tree-builtin-npm", ""},
		{"report.pdf", "t3-file-icon-pdf", ""},
		{"photo.heic", "file-tree-builtin-image", "image"},
		{"Makefile", "file-tree-builtin-default", "default"},
	} {
		if name, token := fileIcon(c.path); name != c.name || token != c.token {
			t.Errorf("fileIcon(%q) = %s, %q; want %s, %q", c.path, name, token, c.name, c.token)
		}
	}
	for _, c := range []struct {
		lang, file string
		icon       bool
	}{{"bash", "file.sh", true}, {"shell", "file.sh", true}, {"go", "file.go", true}, {"dockerfile", "Dockerfile", true},
		{"text", "file.text", false}, {"elm", "file.elm", false}} {
		if f := languageFile(c.lang); f != c.file || specificIcon(f) != c.icon {
			t.Errorf("language %q: %q, its own icon %v; want %q, %v", c.lang, f, specificIcon(f), c.file, c.icon)
		}
	}
}

// A shell block's header shows bash's 14dp icon, not the word: its
// squircle is #199f43 at 20% on the frame, 13dp in and 12dp down, and
// its mark is in the full colour (§3.6). Go's "O" has its middle cut out
// by the evenodd rule: there the frame shows, and #1ca1c7 above it.
func TestCodeHeaderLanguageIcon(t *testing.T) {
	img := mdDrawn(t, "```bash\necho hi\n```", mdBody(), 400, 80, 2)
	frame := over(P.Secondary, P.Background)
	// (3, 8) of the icon's 16: on the squircle, clear of the "$".
	wantPixel(t, img, image.Pt(2*13+5, 2*12+14), over(alpha(rgb(0x199f43), 0x33), frame), "bash's squircle")
	mark := false
	for y := 2 * 12; y < 2*26; y++ {
		for x := 2 * 13; x < 2*27; x++ {
			c := img.RGBAAt(x, y)
			mark = mark || near(c.R, 0x19, 6) && near(c.G, 0x9f, 6) && near(c.B, 0x43, 6)
		}
	}
	if !mark {
		t.Error("no pixel of the icon is bash's #199f43")
	}
	img = mdDrawn(t, "```go\npackage main\n```", mdBody(), 400, 80, 2)
	// The O's middle, (11.4, 7.4) of 16, and its ring above, (11.4, 5.1).
	wantPixel(t, img, image.Pt(2*13+20, 2*12+13), frame, "the middle of Go's O")
	wantPixel(t, img, image.Pt(2*13+20, 2*12+9), rgb(0x1ca1c7), "Go's O")
}
