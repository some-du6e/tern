package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/clip"
	"github.com/some-du6e/tern/app/ui"
)

// Mod+S puts the prompt and its pictures aside and clears the composer;
// with nothing in the composer it brings back the only entry, or with
// several opens the list, whose keys pick one (useComposerStash.ts).
func TestStash(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	th := ui.NewTheme()
	stash := func() { a.handle(th, ui.Action{ID: "stash.save", Kind: "click"}) }

	th.SetEditorText("composer", "first   idea,\n  spread out")
	a.pasted(a.composerKey(), []clip.Image{{Name: "a.png", Mime: "image/png", Data: pngBytes(t)}}, nil)
	a.RunUI(th)
	stash()
	s := a.build(time.Now())
	if th.EditorText("composer") != "" || len(s.Thumbs) != 0 {
		t.Fatalf("after stashing: %q, %d pictures", th.EditorText("composer"), len(s.Thumbs))
	}
	if len(s.Stash) != 1 || s.Stash[0].Snippet != "first idea, spread out" || len(s.Stash[0].Thumbs) != 1 || s.Stash[0].Age != "just now" || s.Stashed.IsZero() {
		t.Fatalf("stash = %+v, stashed at %v", s.Stash, s.Stashed)
	}

	// Nothing typed: the only entry comes back, picture and all.
	stash()
	if s := a.build(time.Now()); th.EditorText("composer") != "first   idea,\n  spread out" || len(s.Thumbs) != 1 || len(s.Stash) != 0 {
		t.Fatalf("after restoring: %q, %d pictures, %d stashed", th.EditorText("composer"), len(s.Thumbs), len(s.Stash))
	}

	// Two entries, newest first; Mod+S on an empty composer opens the list.
	stash()
	th.SetEditorText("composer", "second")
	stash()
	stash()
	s = a.build(time.Now())
	if !s.StashOpen || len(s.Stash) != 2 || s.Stash[0].Snippet != "second" {
		t.Fatalf("list: open %v, %+v", s.StashOpen, s.Stash)
	}
	a.handle(th, ui.Action{ID: "stash", Kind: "key", Text: "↓"})
	a.handle(th, ui.Action{ID: "stash", Kind: "key", Text: "⏎"})
	if s := a.build(time.Now()); s.StashOpen || len(s.Stash) != 1 || !strings.HasPrefix(th.EditorText("composer"), "first") {
		t.Fatalf("Enter on the second row: open %v, %+v, %q", s.StashOpen, s.Stash, th.EditorText("composer"))
	}
	// Restoring into a prompt appends after a blank line.
	th.SetEditorText("composer", "now")
	a.handle(th, ui.Action{ID: "stash.restore:0", Kind: "click"})
	if got := th.EditorText("composer"); got != "now\n\nsecond" {
		t.Fatalf("restored into a prompt: %q", got)
	}

	// The stash persists, written off the UI's path.
	th.SetEditorText("composer", "kept")
	stash()
	a.stashSaving.Wait()
	b, err := NewWithStore(a.store, nil)
	if err != nil {
		t.Fatal(err)
	}
	b.envs, b.booted, b.signedIn = a.envs, true, true
	b.openThread("swiss/t1")
	if s := b.build(time.Now()); len(s.Stash) != 1 || s.Stash[0].Snippet != "kept" {
		t.Fatalf("reloaded: %+v", s.Stash)
	}
	a.handle(th, ui.Action{ID: "stash.delete:0", Kind: "click"})
	if s := a.build(time.Now()); len(s.Stash) != 0 {
		t.Fatalf("after delete: %+v", s.Stash)
	}
}

// An entry whose pictures don't fit in the composer stays stashed, and
// says why, rather than come back without some of them.
func TestStashRestoreKeepsPictures(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	th := ui.NewTheme()
	shot := clip.Image{Name: "a.png", Mime: "image/png", Data: pngBytes(t)}
	for i := 0; i < 4; i++ {
		a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	}
	a.handle(th, ui.Action{ID: "stash.save", Kind: "click"})
	for i := 0; i < maxImages-2; i++ {
		a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	}
	a.handle(th, ui.Action{ID: "stash.restore:0", Kind: "click"})
	s := a.build(time.Now())
	if len(s.Stash) != 1 || len(s.Thumbs) != maxImages-2 || !strings.Contains(s.Error, "stays stashed") {
		t.Fatalf("restoring 4 into %d: %d stashed, %d pictures, error %q", maxImages-2, len(s.Stash), len(s.Thumbs), s.Error)
	}
}

// T3's relative times: "just now", then minutes, hours and days ago, and
// never weeks (formatRelativeElapsed).
func TestRelativeAge(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		ago  time.Duration
		want string
	}{
		{10 * time.Second, "just now"}, {4 * time.Minute, "4m ago"}, {90 * time.Minute, "1h ago"},
		{50 * time.Hour, "2d ago"}, {20 * 24 * time.Hour, "20d ago"},
	} {
		if got := relativeAge(now.Add(-c.ago), now); got != c.want {
			t.Errorf("%v ago: %q, want %q", c.ago, got, c.want)
		}
	}
}
