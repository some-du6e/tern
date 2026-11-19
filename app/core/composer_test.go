package core

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/clip"
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
// provider's commands only at the prompt's start, /compact only where it
// can go, and skills that are also commands only once.
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
	if got, want := labels(slashItems(p, at("", 0), true, false)), "/model /compact /review /resume /skill:frontend-refactor"; got != want {
		t.Errorf("all: %s, want %s", got, want)
	}
	if got, want := labels(slashItems(p, at("", 0), false, false)), "/model /review /resume /skill:frontend-refactor"; got != want {
		t.Errorf("no compacting now: %s, want %s", got, want)
	}
	if got, want := labels(slashItems(p, at("re", 0), true, false)), "/resume /review /skill:frontend-refactor /model"; got != want {
		t.Errorf(`"re": %s, want %s`, got, want)
	}
	if got, want := labels(slashItems(p, at("", 6), true, false)), "/model /skill:frontend-refactor"; got != want {
		t.Errorf("mid-prompt: %s, want %s", got, want)
	}
	if items := slashItems(p, at("skill", 0), true, false); labels(items) != "/skill:frontend-refactor" || items[0].insert != "$frontend-refactor " || items[0].Badge != "Personal" {
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

// After the name, the trigger shows the reasoning level's word, the
// speed's bolt, Ultracode's icon and the context window
// (describeComposerEffort, buildTraitsTriggerDisplay).
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
	word, speed, ultracode, window := effortSuffix(t3.ModelSelection{}, m)
	if word != "High" || speed != "Fast" || ultracode || window != "1M" {
		t.Fatalf("suffix = %q %q %v %q", word, speed, ultracode, window)
	}
	word, _, _, _ = effortSuffix(t3.ModelSelection{Options: []t3.OptionChoice{{ID: "effort", Value: "low"}}}, m)
	if word != "Low" {
		t.Fatalf("picked effort = %q", word)
	}
	// A Codex service tier names its speed; Ultracode is an on/off option.
	m.Capabilities.OptionDescriptors = []t3.OptionDescriptor{
		sel("serviceTier", "ultrafast", "Standard", "Fast", "Ultrafast"),
		{ID: "ultracode", Type: "boolean", CurrentValue: true},
	}
	if _, speed, ultracode, _ := effortSuffix(t3.ModelSelection{}, m); speed != "Ultrafast" || !ultracode {
		t.Fatalf("tier: speed %q, ultracode %v", speed, ultracode)
	}
}

// The placeholder is T3's, first that applies (ChatComposer.tsx): an
// approval that takes feedback asks for it, one that doesn't says a send
// declines it; then the question's, the provider's, the suggestion.
func TestPlaceholder(t *testing.T) {
	no := false
	for _, c := range []struct {
		approval   *approval
		q          *t3.Question
		none       bool
		suggestion string
		want       string
	}{
		{&approval{live: true, feedback: true}, nil, false, "", "Send feedback to decline this request"},
		{&approval{live: true}, &t3.Question{}, true, "", "Send a message to decline and redirect the agent"},
		{nil, &t3.Question{AllowCustomAnswer: &no}, false, "", "Choose an option above"},
		{nil, &t3.Question{}, false, "", "Type your answer"},
		{nil, &t3.Question{Options: []struct {
			Label       string `json:"label"`
			Value       string `json:"value"`
			Description string `json:"description"`
		}{{Label: "A"}}}, false, "", "Type your own answer, or leave this blank to use the selected option"},
		{nil, nil, true, "go ahead", "Enable a provider in Settings to send a message"},
		{nil, nil, false, "go ahead", "go ahead"},
		{nil, nil, false, "", "Ask anything. @ files, $ skills, / commands"},
	} {
		if got := placeholder(c.approval, c.q, c.none, c.suggestion); got != c.want {
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
// model menu. Escape shuts a trigger of that kind at that place, whatever
// is typed after it, until the caret leaves it (useComposerTriggerState.ts).
func TestCommandMenu(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	th := ui.NewTheme()
	typed := func(s string) *ui.Scene {
		th.SetEditorText("composer", s)
		a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: s})
		return a.build(time.Now())
	}
	s := typed("/mo")
	if len(s.Suggest) != 1 || s.Suggest[0].Label != "/model" || s.SuggestSel != 0 {
		t.Fatalf("menu = %+v", s.Suggest)
	}
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "↓"})
	if s := a.build(time.Now()); s.SuggestSel != 0 {
		t.Fatalf("one row: highlight %d, want it to wrap to 0", s.SuggestSel)
	}
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "⏎"})
	// The prompt's change goes back to the app the way a frame's actions do.
	a.Handle(th, th.Actions())
	if s := a.build(time.Now()); th.EditorText("composer") != "" || s.Picker == nil {
		t.Fatalf("after Enter: %q, picker %v", th.EditorText("composer"), s.Picker != nil)
	}
	typed("/mo")
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "⎋"})
	a.handle(th, ui.Action{ID: "composer", Kind: "caret", Text: "/mo"})
	if s := a.build(time.Now()); len(s.Suggest) != 0 || s.SuggestEmpty != "" {
		t.Fatalf("after Escape the menu stays shut: %+v", s.Suggest)
	}
	if s := typed("/mod"); len(s.Suggest) != 0 {
		t.Fatalf("more typed after Escape opened it again: %+v", s.Suggest)
	}
	// With no trigger at the caret the dismissal ends.
	typed("")
	if s := typed("/zz"); len(s.Suggest) != 0 || s.SuggestEmpty != "No matching command." {
		t.Fatalf("no match: %+v %q", s.Suggest, s.SuggestEmpty)
	}
	if s := typed("/mo"); len(s.Suggest) != 1 {
		t.Fatalf("a new trigger opens it: %+v", s.Suggest)
	}
}

// Where the server takes freshContext and the thread has messages, the /
// menu offers T3's /clear in place of the provider's; picking it, or
// sending /clear alone, arms a fresh session, whose banner shows until its
// dismiss keeps the current session (ChatView.tsx, useComposerSend.ts).
func TestFreshSession(t *testing.T) {
	a := testApp(t)
	cfg := &t3.ServerConfig{Providers: []t3.Provider{{InstanceID: "claudeAgent", Driver: "claudeAgent", SlashCommands: []t3.SlashCommand{{Name: "clear"}}}}}
	cfg.Settings.DefaultModelSelection = &t3.ModelSelection{InstanceID: "claudeAgent", Model: "opus"}
	cfg.Environment.Capabilities.FreshContext = true
	a.envs[0].config = cfg
	v := openWith(t, a, nil, nil)
	th := ui.NewTheme()
	typed := func(s string) *ui.Scene {
		th.SetEditorText("composer", s)
		a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: s})
		return a.build(time.Now())
	}
	menu := func(s *ui.Scene) string {
		var out []string
		for _, it := range s.Suggest {
			out = append(out, it.Label+" "+it.Desc)
		}
		return strings.Join(out, ", ")
	}
	if got := menu(typed("/cl")); got != "/clear Run provider command" {
		t.Fatalf("a thread without messages: %s, want the provider's /clear", got)
	}
	v.messages["m1"] = t3.Message{ID: "m1", Role: "user", Text: "hi"}
	typed("")
	if got := menu(typed("/cl")); got != "/clear Start a fresh session; earlier messages stay visible" {
		t.Fatalf("with messages: %s, want T3's /clear alone", got)
	}
	a.handle(th, ui.Action{ID: "suggest", Kind: "key", Text: "⏎"})
	a.Handle(th, th.Actions())
	if s := a.build(time.Now()); !s.Fresh || th.EditorText("composer") != "" || s.Picker != nil {
		t.Fatalf("after picking /clear: armed %v, prompt %q, picker %v", s.Fresh, th.EditorText("composer"), s.Picker != nil)
	}
	a.handle(th, ui.Action{ID: "fresh.keep", Kind: "click"})
	if s := a.build(time.Now()); s.Fresh {
		t.Fatal("Keep the current session left it armed")
	}
	th.SetEditorText("composer", "/clear ")
	a.handle(th, ui.Action{ID: "composer", Kind: "submit"})
	if s := a.build(time.Now()); !s.Fresh || th.EditorText("composer") != "" {
		t.Fatalf("sending /clear alone: armed %v, prompt %q", s.Fresh, th.EditorText("composer"))
	}
}

// A file mention is T3's serializeComposerFileLink: the basename with
// \ [ ] escaped, the path as encodeURI writes it with ( ) # ? \ escaped.
func TestFileLink(t *testing.T) {
	for _, c := range []struct{ path, want string }{
		{"app/ui/composer.go", "[composer.go](app/ui/composer.go)"},
		{"app/blog/[slug].tsx", `[\[slug\].tsx](app/blog/%5Bslug%5D.tsx)`},
		{"docs/a b(1).md", "[a b(1).md](docs/a%20b%281%29.md)"},
		{"a%20b.md", "[a%20b.md](a%2520b.md)"},
		{"notes/#1?.md", "[#1?.md](notes/%231%3F.md)"},
		{"é.txt", "[é.txt](%C3%A9.txt)"},
	} {
		if got := fileLink(c.path); got != c.want {
			t.Errorf("%q: %s, want %s", c.path, got, c.want)
		}
	}
}

// An @ word searches for what follows the sign, 256 characters at most.
func TestPathQuery(t *testing.T) {
	long := strings.Repeat("x", 300)
	if q := pathQuery(&trigger{kind: "path", query: " " + long}); len([]rune(q)) != 256 {
		t.Fatalf("query of %d runes, want 256", len([]rune(q)))
	}
}

// A message comes back as typed (recallableComposerPrompt): no Ultrathink
// prefix, review comments or legacy context blocks at its end, no inline
// references; T3's own prompts come back as nothing.
func TestRecallable(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Ultrathink:\nfix it ![shot](t3-context://v1/image/image_x)", "fix it"},
		{"look here\n\n<review_comment file=\"a.go\">nit</review_comment>", "look here"},
		{"run it\n<terminal_context>\n- Terminal 1 lines 1-4:\nls\n</terminal_context>", "run it"},
		{"[User attached one or more files without additional text. Respond using the conversation context and the attached files.]", ""},
		{"PLEASE IMPLEMENT THIS PLAN:\n1. do it", ""},
	} {
		if got := recallable(c.in); got != c.want {
			t.Errorf("%q: %q, want %q", c.in, got, c.want)
		}
	}
}

// The history is what the user typed: not what an automation, another
// agent or a guest sent (buildComposerPromptHistoryEntries).
func TestPromptHistoryIsTheUsers(t *testing.T) {
	v := newView(nil, "t")
	add := func(id, text string, ord float64, m t3.Message) {
		raw, _ := json.Marshal(map[string]any{"id": id, "type": "user_message", "messageId": id, "text": text, "ordinal": ord})
		v.put(raw)
		m.ID, m.Role, m.Text = id, "user", text
		v.messages[id] = m
	}
	add("m1", "mine", 1, t3.Message{CreatedBy: "user"})
	add("m2", "the agent's", 2, t3.Message{CreatedBy: "agent"})
	add("m3", "on a schedule", 3, t3.Message{CreatedBy: "user", ScheduledTaskID: "task"})
	guest := t3.Message{CreatedBy: "user"}
	guest.Context = &struct {
		Guest json.RawMessage `json:"guest"`
	}{Guest: json.RawMessage(`{"name":"A guest","text":"hi"}`)}
	add("m4", "a guest's", 4, guest)
	add("m5", "mine again", 5, t3.Message{CreatedBy: "user"})
	if got := strings.Join(promptHistory(v), ", "); got != "mine, mine again" {
		t.Fatalf("history = %s", got)
	}
}

// queuedView opens swiss/t1 with one message queued behind its run.
func queuedView(t *testing.T, a *App) *threadView {
	t.Helper()
	v := openWith(t, a, nil, nil)
	_, th := a.thread("swiss/t1")
	th.ActiveRunID = "r0"
	v.runs["r0"] = t3.Run{ID: "r0", Status: "running"}
	v.runs["q1"] = t3.Run{ID: "q1", Status: "queued", UserMessageID: "m1"}
	v.messages["m1"] = t3.Message{ID: "m1", Role: "user", Text: "queued text"}
	return v
}

// A queued message being edited is its thread's (useQueuedRunEdit.ts): a
// new thread's composer doesn't take it over, so a message sent there
// stays what was typed; back on the thread the edit is there again, and
// Cancel brings the thread's own draft back.
func TestQueuedEditStaysWithItsThread(t *testing.T) {
	drafts = map[string]string{}
	a := testApp(t)
	v := queuedView(t, a)
	th := ui.NewTheme()
	th.SetEditorText("composer", "my draft")
	a.handle(th, ui.Action{ID: "queue:0:edit", Kind: "click"})
	if s := a.build(time.Now()); th.EditorText("composer") != "queued text" || !s.EditingQueued || !s.Queue[0].Editing {
		t.Fatalf("editing: prompt %q, editing %v, queue %+v", th.EditorText("composer"), s.EditingQueued, s.Queue)
	}
	th.SetEditorText("composer", "queued text, edited")
	a.handle(th, ui.Action{ID: "new", Kind: "click"})
	if s := a.build(time.Now()); s.EditingQueued || th.EditorText("composer") != "" {
		t.Fatalf("a new thread: editing %v, prompt %q", s.EditingQueued, th.EditorText("composer"))
	}
	th.SetEditorText("composer", "for the new thread")
	a.handle(th, ui.Action{ID: "composer", Kind: "submit"})
	if got := th.EditorText("composer"); got != "for the new thread" {
		t.Fatalf("sent from the new thread: prompt %q, want what was typed", got)
	}
	a.handle(th, ui.Action{ID: "thread:swiss/t1", Kind: "click"})
	a.view = v // its snapshot, back
	if s := a.build(time.Now()); th.EditorText("composer") != "queued text, edited" || !s.EditingQueued {
		t.Fatalf("back on the thread: prompt %q, editing %v", th.EditorText("composer"), s.EditingQueued)
	}
	a.handle(th, ui.Action{ID: "queue:0:cancel-edit", Kind: "click"})
	if s := a.build(time.Now()); th.EditorText("composer") != "my draft" || s.EditingQueued {
		t.Fatalf("after Cancel: prompt %q, editing %v", th.EditorText("composer"), s.EditingQueued)
	}
}

// When the message being edited leaves the queue, a changed edit stays in
// the composer if the thread's own draft was empty; otherwise that draft
// comes back (recoverQueuedMessageEdit). An edit the server can't take
// stays, and says so.
func TestQueuedEditLeavesTheQueue(t *testing.T) {
	for _, c := range []struct{ draft, want string }{{"my draft", "my draft"}, {"", "queued text, edited"}} {
		drafts = map[string]string{}
		a := testApp(t)
		v := queuedView(t, a)
		th := ui.NewTheme()
		th.SetEditorText("composer", c.draft)
		a.handle(th, ui.Action{ID: "queue:0:edit", Kind: "click"})
		th.SetEditorText("composer", "queued text, edited")
		v.runs["q1"] = t3.Run{ID: "q1", Status: "running", UserMessageID: "m1"}
		s := a.build(time.Now())
		a.RunUI(th)
		if got := th.EditorText("composer"); s.EditingQueued || got != c.want {
			t.Fatalf("draft %q: after the run started, editing %v, prompt %q, want %q", c.draft, s.EditingQueued, got, c.want)
		}
	}
	a := testApp(t)
	queuedView(t, a)
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "queue:0:edit", Kind: "click"})
	th.SetEditorText("composer", "queued text, edited")
	a.handle(th, ui.Action{ID: "composer", Kind: "submit"})
	a.RunUI(th)
	if s := a.build(time.Now()); !s.EditingQueued || th.EditorText("composer") != "queued text, edited" || s.Error != "Could not save the edited queued message." {
		t.Fatalf("a save that didn't go: editing %v, prompt %q, error %q", s.EditingQueued, th.EditorText("composer"), s.Error)
	}
}

// A stashed prompt keeps its pictures' chips: restored, the picture's a
// chip again where it was, numbered for this composer (useComposerStash.ts
// stashes the whole draft).
func TestStashKeepsChips(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	th := ui.NewTheme()
	th.SetEditorText("composer", "look ")
	a.pasted(a.composerKey(), []clip.Image{{Name: "shot.png", Mime: "image/png", Data: pngBytes(t)}}, nil)
	a.RunUI(th)
	th.InsertText("composer", " here")
	a.handle(th, ui.Action{ID: "stash.save", Kind: "click"})
	if s := a.build(time.Now()); len(s.Stash) != 1 || s.Stash[0].Snippet != "look here" {
		t.Fatalf("stashed: %+v", s.Stash)
	}
	a.handle(th, ui.Action{ID: "stash.save", Kind: "click"})
	text := th.EditorText("composer")
	s := a.build(time.Now())
	if chips := ui.Chips(text); len(chips) != 1 || len(s.Thumbs) != 1 || s.Thumbs[0].N != chips[0] {
		t.Fatalf("restored: %q, pictures %+v", text, s.Thumbs)
	}
	if got := ui.StripChips(text); got != "look  here" {
		t.Fatalf("restored text around the chip: %q", got)
	}
}

// A press outside the open stash closes it.
func TestStashCloses(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "stash.save", Kind: "click"})
	if s := a.build(time.Now()); !s.StashOpen {
		t.Fatal("Mod+S on an empty stash didn't open its list")
	}
	a.handle(th, ui.Action{ID: "stash.close", Kind: "click"})
	if s := a.build(time.Now()); s.StashOpen {
		t.Fatal("a press outside left the stash open")
	}
}

// While /clear is armed the provider's suggestion isn't offered: the fresh
// session couldn't see what it refers to (ChatComposer.tsx).
func TestNoSuggestionWhileFresh(t *testing.T) {
	a := testApp(t)
	cfg := &t3.ServerConfig{Providers: []t3.Provider{{InstanceID: "claudeAgent", Driver: "claudeAgent", Enabled: true, Status: "ready", Models: []t3.Model{{Slug: "opus", Name: "Claude Opus 5.5"}}}}}
	cfg.Environment.Capabilities.FreshContext = true
	a.envs[0].config = cfg
	v := openWith(t, a, nil, nil)
	_, th := a.thread("swiss/t1")
	th.LatestRunID = "r1"
	v.runs["r1"] = t3.Run{ID: "r1", Status: "completed", StartedAt: "2026-11-18T10:00:00Z"}
	v.messages["m1"] = t3.Message{ID: "m1", Role: "user", Text: "hi"}
	a.suggestion = suggestion{ref: "swiss/t1", run: "r1", text: "go ahead"}
	if s := a.build(time.Now()); s.Suggestion != "go ahead" || s.Placeholder != "go ahead" {
		t.Fatalf("idle: suggestion %q, placeholder %q", s.Suggestion, s.Placeholder)
	}
	a.comp.fresh["swiss/t1"] = true
	if s := a.build(time.Now()); s.Suggestion != "" || s.Placeholder != "Ask anything. @ files, $ skills, / commands" {
		t.Fatalf("with /clear armed: suggestion %q, placeholder %q", s.Suggestion, s.Placeholder)
	}
}

// Mod+Alt+N works from a thread too: it opens a new thread, keeping the
// thread's prompt for when it's back.
func TestScratchFromThread(t *testing.T) {
	drafts = map[string]string{}
	a := testApp(t)
	a.envs[0].config = &t3.ServerConfig{ScratchWorkspaceRoot: "/srv/scratch"}
	a.openThread("swiss/t1")
	if s := a.build(time.Now()); !s.CanScratch {
		t.Fatal("a thread on a machine with a scratch workspace can't start one without a project")
	}
	th := ui.NewTheme()
	th.SetEditorText("composer", "unsent")
	a.handle(th, ui.Action{ID: "project.scratch", Kind: "click"})
	if a.build(time.Now()).Title != "New thread" || th.EditorText("composer") != "" {
		t.Fatalf("after Mod+Alt+N: prompt %q", th.EditorText("composer"))
	}
	a.handle(th, ui.Action{ID: "thread:swiss/t1", Kind: "click"})
	if got := th.EditorText("composer"); got != "unsent" {
		t.Fatalf("back on the thread: prompt %q, want its own", got)
	}
}

// The provider's /compact is on offer only alone in the prompt, on a
// thread that can compact now: not while it runs (compactSlashCommandAvailable).
func TestCompactCommand(t *testing.T) {
	a := testApp(t)
	cfg := &t3.ServerConfig{Providers: []t3.Provider{{InstanceID: "claudeAgent", Driver: "claudeAgent", SlashCommands: []t3.SlashCommand{{Name: "compact"}}}}}
	cfg.Settings.DefaultModelSelection = &t3.ModelSelection{InstanceID: "claudeAgent", Model: "opus"}
	a.envs[0].config = cfg
	v := openWith(t, a, nil, nil)
	v.messages["m1"] = t3.Message{ID: "m1", Role: "user", Text: "hi"}
	// Connected: a thread compacts on a machine it can reach.
	a.envs[0].conn = new(t3.Conn)
	t.Cleanup(func() { a.envs[0].conn = nil })
	th := ui.NewTheme()
	menu := func(text string, caret int) string {
		th.SetEditorText("composer", text)
		th.Editor("composer").SetCaret(caret, caret)
		a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: text})
		var out []string
		for _, it := range a.build(time.Now()).Suggest {
			out = append(out, it.Label)
		}
		return strings.Join(out, " ")
	}
	if got := menu("/com", 4); got != "/compact" {
		t.Fatalf("alone: %q", got)
	}
	if got := menu("/com later", 4); strings.Contains(got, "/compact") {
		t.Fatalf("with text after it: %q", got)
	}
	_, t1 := a.thread("swiss/t1")
	t1.ActiveRunID = "r1"
	if got := menu("/comp", 5); strings.Contains(got, "/compact") {
		t.Fatalf("while it runs: %q", got)
	}
}
