package core

import (
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The main column with no thread (docs/ui/11-empty-auth.md §2.1, §5.4):
// T3's hosted home with no computer on the account; nothing while a
// computer's threads are still coming; NoProjectsHero once they're in and
// there's no project.
func TestHome(t *testing.T) {
	a := testApp(t)
	a.envs = nil
	if s := a.build(time.Now()); s.Home != ui.HomeNoComputer {
		t.Errorf("no computer: home %d", s.Home)
	}
	a.envs = []*envState{newEnv(t3.Environment{EnvironmentID: "e1", Label: "studio"}, false)}
	if s := a.build(time.Now()); s.Home != ui.HomeBlank {
		t.Errorf("threads still loading: home %d", s.Home)
	}
	a.envs[0].loaded = true
	if s := a.build(time.Now()); s.Home != ui.HomeNoProjects {
		t.Errorf("no project: home %d", s.Home)
	}
}

// A thread that's gone once its computer's threads are in: home to a new
// thread when the computer has others, else NoActiveThreadState.
func TestMissingThread(t *testing.T) {
	a := testApp(t)
	e := newEnv(t3.Environment{EnvironmentID: "e1", Label: "studio"}, false)
	e.loaded = true
	e.projects["p"] = t3.Project{ID: "p", Title: "example"}
	e.threads["t1"] = &t3.Thread{ID: "t1", Title: "A thread", ProjectID: "p"}
	a.envs = []*envState{e}
	a.openID = "e1/gone"
	if s := a.build(time.Now()); s.Home != ui.HomeNone || a.draft == "" {
		t.Errorf("gone from a computer with threads: home %d, draft %q", s.Home, a.draft)
	}
	a.draft = ""
	delete(e.threads, "t1")
	a.openID = "e1/gone"
	if s := a.build(time.Now()); s.Home != ui.HomeNoThread {
		t.Errorf("gone from a computer without threads: home %d", s.Home)
	}
}
