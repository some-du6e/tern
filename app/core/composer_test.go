package core

import (
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The word before the caret opens T3's menus (composer-logic.ts
// detectComposerTrigger): / commands unless it's a path, a currency sign
// skills unless it's an amount, @ files; nothing against a chip.
func TestDetectTrigger(t *testing.T) {
	for _, c := range []struct {
		text       string
		cursor     int
		kind       string
		query      string
		start, end int
	}{
		{"/mo", 3, "slash", "mo", 0, 3},
		{"see /usr/bin", 12, "", "", 0, 0},
		{"/usr", 4, "slash", "usr", 0, 4},
		{"ask $rev", 8, "skill", "rev", 4, 8},
		{"costs $5", 8, "", "", 0, 0},
		{"costs $10k.", 11, "", "", 0, 0},
		{"look @src/ind", 13, "path", "src/ind", 5, 13},
		{"@abc def", 2, "path", "a", 0, 2},
		{"mail a@b", 8, "", "", 0, 0},
		{"#12", 3, "", "", 0, 0},
		{ui.ChipText(1) + "@x", 10, "", "", 0, 0},
		{"two words ", 10, "", "", 0, 0},
	} {
		got := detectTrigger([]rune(c.text), c.cursor)
		switch {
		case c.kind == "" && got != nil:
			t.Errorf("%q at %d: %+v, want no menu", c.text, c.cursor, *got)
		case c.kind != "" && (got == nil || got.kind != c.kind || got.query != c.query || got.start != c.start || got.end != c.end):
			t.Errorf("%q at %d: %+v, want %s %q [%d, %d)", c.text, c.cursor, got, c.kind, c.query, c.start, c.end)
		}
	}
}

// The / menu ranks like T3's (composerSlashCommandSearch.ts): prefixes
// before boundaries before descriptions, ties by kind then name; the
// provider's commands only at the prompt's start, /compact only with
// nothing before it, and skills that are also commands only once.
func TestSlashItems(t *testing.T) {
	yes := true
	p := &t3.Provider{
		SlashCommands: []t3.SlashCommand{{Name: "compact"}, {Name: "review", Description: "Review the diff"}, {Name: "resume"}, {Name: "frontend-refactor"}},
		Skills: []t3.Skill{
			{Name: "frontend-refactor", Scope: "user", Enabled: true, UserInvocable: &yes},
			{Name: "hidden", Enabled: false},
		},
	}
	labels := func(items []menuItem) string {
		var out []string
		for _, it := range items {
			out = append(out, it.Prefix+it.Label)
		}
		return strings.Join(out, " ")
	}
	at := func(q string, start int) *trigger { return &trigger{kind: "slash", query: q, start: start} }
	if got, want := labels(slashItems(p, at("", 0), "")), "/model /compact /review /resume /skill:frontend-refactor"; got != want {
		t.Errorf("all: %s, want %s", got, want)
	}
	if got, want := labels(slashItems(p, at("re", 0), "")), "/resume /review /skill:frontend-refactor /model"; got != want {
		t.Errorf(`"re": %s, want %s`, got, want)
	}
	if got, want := labels(slashItems(p, at("", 6), "later ")), "/model /skill:frontend-refactor"; got != want {
		t.Errorf("mid-prompt: %s, want %s", got, want)
	}
	if items := slashItems(p, at("skill", 0), ""); labels(items) != "/skill:frontend-refactor" || items[0].insert != "$frontend-refactor " || items[0].Badge != "Personal" {
		t.Errorf(`"skill": %+v`, items)
	}
	if got := labels(skillItems(p, &trigger{kind: "skill", query: "front"})); got != "frontend-refactor" {
		t.Errorf("$ menu: %s", got)
	}
}

// Up goes back through the prompts, Down forward; past the newest the
// prompt is empty again (composerPromptHistory.ts).
func TestStepHistory(t *testing.T) {
	h := []string{"a", "b", "c"}
	steps := []struct {
		back  bool
		entry int
		want  string
		ok    bool
	}{
		{true, 2, "c", true}, {true, 1, "b", true}, {true, 0, "a", true}, {true, 0, "", false},
		{false, 1, "b", true}, {false, 2, "c", true}, {false, -1, "", true}, {false, -1, "", false},
	}
	entry := -1
	for i, st := range steps {
		e, p, ok := stepHistory(h, entry, st.back)
		if ok != st.ok || (ok && (e != st.entry || p != st.want)) {
			t.Fatalf("step %d: %d %q %v, want %d %q %v", i, e, p, ok, st.entry, st.want, st.ok)
		}
		if ok {
			entry = e
		}
	}
	if got := recallable("Ultrathink:\nfix it ![shot](t3-context://v1/image/image_x)"); got != "fix it" {
		t.Errorf("recallable = %q", got)
	}
}

// The trigger shows a model's short name (modelShortName.ts): a family's
// prefix goes on its own provider when something's left to name it.
func TestModelShortName(t *testing.T) {
	for _, c := range []struct{ name, driver, want string }{
		{"Claude Opus 5.5", "claudeAgent", "Opus 5.5"},
		{"GPT-5.5", "codex", "GPT-5.5"},
		{"GPT-5.5 Mini", "codex", "5.5 Mini"},
		{"Gemini 3 Pro", "antigravity", "3 Pro"},
		{"Gemini 3 Pro", "cursor", "Gemini 3 Pro"},
		{"OpenAI: GPT-5", "opencode", "GPT-5"},
	} {
		if got := modelShortName(t3.Model{Name: c.name}, c.driver); got != c.want {
			t.Errorf("%q on %s: %q, want %q", c.name, c.driver, got, c.want)
		}
	}
	if got := modelShortName(t3.Model{Name: "Claude Opus 5.5", ShortName: "Opus"}, "claudeAgent"); got != "Opus" {
		t.Errorf("its own short name: %q", got)
	}
}

// After the name, the trigger shows the reasoning level's word, Fast's
// bolt and the context window (describeComposerEffort).
func TestEffortSuffix(t *testing.T) {
	m := &t3.Model{}
	sel := func(id, cur string, opts ...string) t3.OptionDescriptor {
		d := t3.OptionDescriptor{ID: id, Type: "select", CurrentValue: cur}
		for _, o := range opts {
			d.Options = append(d.Options, struct {
				ID          string `json:"id"`
				Label       string `json:"label"`
				Description string `json:"description"`
				IsDefault   bool   `json:"isDefault"`
			}{ID: strings.ToLower(o), Label: o})
		}
		return d
	}
	m.Capabilities.OptionDescriptors = []t3.OptionDescriptor{
		sel("effort", "high", "Low", "Medium", "High"),
		{ID: "fastMode", Type: "boolean", CurrentValue: true},
		sel("contextWindow", "1m", "200k", "1M"),
	}
	word, fast, window := effortSuffix(t3.ModelSelection{}, m)
	if word != "High" || !fast || window != "1M" {
		t.Fatalf("suffix = %q %v %q", word, fast, window)
	}
	word, _, _ = effortSuffix(t3.ModelSelection{Options: []t3.OptionChoice{{ID: "effort", Value: "low"}}}, m)
	if word != "Low" {
		t.Fatalf("picked effort = %q", word)
	}
}

// The placeholder is T3's, first that applies (ChatComposer.tsx).
func TestPlaceholder(t *testing.T) {
	no := false
	for _, c := range []struct {
		approval bool
		q        *t3.Question
		none     bool
		want     string
	}{
		{true, &t3.Question{}, true, "Send a message to decline and redirect the agent"},
		{false, &t3.Question{AllowCustomAnswer: &no}, false, "Choose an option above"},
		{false, &t3.Question{}, false, "Type your answer"},
		{false, &t3.Question{Options: []struct {
			Label       string `json:"label"`
			Value       string `json:"value"`
			Description string `json:"description"`
		}{{Label: "A"}}}, false, "Type your own answer, or leave this blank to use the selected option"},
		{false, nil, true, "Enable a provider in Settings to send a message"},
		{false, nil, false, "Ask anything. @ files, $ skills, / commands"},
	} {
		if got := placeholder(c.approval, c.q, c.none); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

// A send starts a run; while one works it steers it, and Mod+Enter
// queues behind it instead (resolveComposerDispatchMode).
func TestDispatchMode(t *testing.T) {
	if dispatchMode(false, true) != "auto" || dispatchMode(true, false) != "steer" || dispatchMode(true, true) != "queue" {
		t.Fatal("dispatch modes")
	}
}

// Typing / opens the menu over the composer; the arrows move its
// highlight and Enter picks: /model empties the prompt and opens the
// model menu. Escape closes the menu until the word changes.
func TestCommandMenu(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	th := ui.NewTheme()
	th.SetEditorText("composer", "/mo")
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "/mo"})
	s := a.build(time.Now())
	if len(s.Suggest) != 1 || s.Suggest[0].Label != "/model" || s.SuggestSel != 0 {
		t.Fatalf("menu = %+v", s.Suggest)
	}
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "↓"})
	if s := a.build(time.Now()); s.SuggestSel != 0 {
		t.Fatalf("one row: highlight %d, want it to wrap to 0", s.SuggestSel)
	}
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "⏎"})
	if th.EditorText("composer") != "" || a.picker == nil {
		t.Fatalf("after Enter: %q, picker %v", th.EditorText("composer"), a.picker != nil)
	}
	th.SetEditorText("composer", "/mo")
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "/mo"})
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "⎋"})
	a.handle(th, ui.Action{ID: "composer", Kind: "caret", Text: "/mo"})
	if s := a.build(time.Now()); len(s.Suggest) != 0 || s.SuggestEmpty != "" {
		t.Fatalf("after Escape the menu stays shut: %+v", s.Suggest)
	}
	th.SetEditorText("composer", "/mod")
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "/mod"})
	if s := a.build(time.Now()); len(s.Suggest) != 1 {
		t.Fatalf("a changed word opens it again: %+v", s.Suggest)
	}
	th.SetEditorText("composer", "/zz")
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "/zz"})
	if s := a.build(time.Now()); len(s.Suggest) != 0 || s.SuggestEmpty != "No matching command." {
		t.Fatalf("no match: %+v %q", s.Suggest, s.SuggestEmpty)
	}
}
