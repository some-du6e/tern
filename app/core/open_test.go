package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/some-du6e/tern/app/ui"
)

// A markdown link's click opens web addresses in the system browser, T3's
// default link target; T3's desktop app lets nothing else out
// (ElectronShell.ts:52-68).
func TestMarkdownLinkOpens(t *testing.T) {
	a := testApp(t)
	var opened []string
	defer func(f func(string) error) { openExternal = f }(openExternal)
	openExternal = func(u string) error {
		opened = append(opened, u)
		return nil
	}
	for _, href := range []string{"https://pkg.go.dev/io#Writer", "http://localhost:8787/", "file:///etc/hosts", "javascript:alert(1)", "#section", "docs/readme.md"} {
		a.Handle(nil, []ui.Action{{ID: "link:" + href, Kind: "click"}})
	}
	if len(opened) != 2 || opened[0] != "https://pkg.go.dev/io#Writer" || opened[1] != "http://localhost:8787/" {
		t.Fatalf("opened %q", opened)
	}
}

// "Download as markdown" lands in the Downloads folder under the plan's
// name, and a second download of the same plan doesn't overwrite the
// first, as a browser's doesn't.
func TestPlanDownload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	a := testApp(t)
	for _, text := range []string{"# Retry\n", "# Retry again\n"} {
		a.Handle(nil, []ui.Action{{ID: "plan.download:retry.md", Kind: "click", Text: text}})
	}
	for name, want := range map[string]string{"retry.md": "# Retry\n", "retry (1).md": "# Retry again\n"} {
		got, err := os.ReadFile(filepath.Join(home, "Downloads", name))
		if err != nil || string(got) != want {
			t.Errorf("%s: %q %v, want %q", name, got, err, want)
		}
	}
}
