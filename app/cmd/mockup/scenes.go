package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/ui"
)

func envs() []ui.Env {
	return []ui.Env{{Label: "workstation", Status: ui.EnvOnline}, {Label: "bowen", Status: ui.EnvOnline}, {Label: "macbook", Status: ui.EnvOffline}}
}

func sidebar(s *ui.Scene, selected string) {
	s.Envs = envs()
	s.Filter = "All projects"
	s.Pinned = []ui.ThreadRow{
		{Title: "tern roadmap", Project: "tern", Branch: "main", Age: "2d"},
		{Title: "Fleet: bowen nightly backups", Project: "swiss-fleet", Branch: "main", Env: "bowen", Unread: true, Age: "7h"},
	}
	s.Active = []ui.ThreadRow{
		{Title: "Review: thread.stop capability", Project: "t3code", State: ui.ThreadApproval, Note: "pnpm test --filter server", Age: "2h"},
		{Title: "Watch pairing flow", Project: "tern", State: ui.ThreadQuestion, Note: "Which session should the watch get?", Age: "41m"},
		{Title: "Port the T3 Connect client to Go", Project: "tern", Branch: "t3/go-client", State: ui.ThreadRunning, Timer: "1m 08s", Age: "now"},
		{Title: "Restore test: vaultwarden snapshot", Project: "swiss-fleet", Branch: "main", Env: "bowen", State: ui.ThreadRunning, Timer: "12m", Age: "12m"},
		{Title: "Gio sidebar: virtualize 1k rows", Project: "tern", Branch: "t3/sidebar-virtual", Unread: true, Age: "18m"},
		{Title: "Relay status polling", Project: "tern", State: ui.ThreadError, Note: "RelayEnvironmentEndpointTimedOutError", Age: "5h"},
		{Title: "History essay: outline and sources", Project: "schoolwork-writer", Branch: "main", Age: "1d"},
		{Title: "Bounded snapshot history cursor", Project: "t3code", Branch: "t3/history-cursor", Env: "macbook", Age: "1d"},
	}
	s.Snoozed = []ui.ThreadRow{
		{Title: "Clerk email-code sign-in", Project: "tern", Branch: "t3/clerk", Wakes: "18:00"},
	}
	s.Settled = 76
	for _, list := range [][]ui.ThreadRow{s.Pinned, s.Active, s.Snoozed} {
		for i := range list {
			list[i].Selected = list[i].Title == selected
		}
	}
	s.Net = "workstation  ·  connected through relay in 612 ms  ·  DPoP session renews in 52m"
	s.Stats = "1,024 threads   ·   2 running   ·   ping 38 ms   ·   frame 0.41 ms"
}

// The timeline's fixtures: a made-up project, so nothing real shows.

const retryReply = "Done. `upload.Send` now retries with **exponential backoff**:\n\n" +
	"- the first retry waits 250 ms, doubling up to a 4 s cap;\n" +
	"- it gives up after five attempts and returns the last error;\n" +
	"- `ErrPermanent` stops it at once.\n\n" +
	"The loop is in `upload/send.go:42`; [the tests](upload/send_test.go) cover each case.\n\n" +
	"Run the tests with:\n\n```bash\ngo test ./upload/...\n```\n"

// workspace is the made-up project's root, which its file links name files in.
const workspace = "/home/dev/acme"

// threadRows is a running turn: a settled one folded above it, its open
// tool group, and the running command with the newest thought.
func threadRows() []ui.TimelineRow {
	now := time.Now()
	return []ui.TimelineRow{
		{ID: "u1", Kind: ui.RowUser, Message: &ui.Message{ID: "u1", Text: "Add retries to the uploader: back off exponentially, give up after five attempts, and keep the existing timeout.",
			PromptLink: "https://app.t3.codes/new?prompt=Add+retries+to+the+uploader"}},
		{ID: "turn-fold:run-1", Kind: ui.RowTurnFold, Group: "run-1", Label: "Worked for 4m 12s"},
		{ID: "a1", Kind: ui.RowAssistant, Message: &ui.Message{ID: "a1", Text: retryReply, Meta: true, Copy: true, Fork: true}},
		{ID: "u2", Kind: ui.RowUser, Message: &ui.Message{ID: "u2", Text: "Looks right. Log each retry, like in " + ui.ChipText(1), Chips: []string{"Image 1"},
			Pictures: []ui.Picture{screenshot(480, 300)}}},
		{ID: "working-indicator-row", Kind: ui.RowWorking, Since: now.Add(-68 * time.Second)},
		{ID: "work-toggle:e1", Kind: ui.RowWorkToggle, Group: "work-group:e1", Open: true, Icon: "hammer", Label: "Ran 2 commands, changed 1 file, and performed 3 other actions", Continues: true},
		{ID: "work-group:e1:details", Kind: ui.RowWork, Group: "work-group:e1", Open: true, Continues: true, Entries: []ui.WorkEntry{
			{ID: "e1", Icon: "eye", Label: "Read upload/send.go", Expands: true},
			{ID: "e2", Icon: "search", Label: "Searched retry in upload"},
			{ID: "e3", Icon: "square-pen", Label: "upload/send.go", Expands: true, Open: true,
				Body: &ui.EntryBody{File: "upload/send.go", Add: 48, Del: 6, Counts: true}},
			{ID: "e4", Icon: "terminal", Label: "go test ./upload/...", Expands: true, Open: true,
				Body: &ui.EntryBody{Command: "go test -run 'TestRetry' -count 3 ./upload/... | tee test.log", Output: "ok  \texample.com/acme/upload\t0.412s"}},
			{ID: "e8", Icon: "globe", Label: "Searched the web for exponential backoff jitter", Expands: true, Open: true,
				Body: &ui.EntryBody{Searches: []string{"exponential backoff jitter"}, Hits: []ui.SearchHit{{Title: "Exponential backoff and jitter",
					Text: "Adding randomness to retries spreads load.", Web: true, URL: "https://example.com/backoff"}}}},
			{ID: "e5", Icon: "terminal", Label: "go vet ./...", Failed: true, Expands: true},
			{ID: "e6", Icon: "brain", Label: "Checking where the logger comes from", Expands: true,
				Thought: "Checking where the logger comes from. `Send` takes a `*slog.Logger`, so each retry can log at debug level."},
		}},
		{ID: "live-activity-row", Kind: ui.RowWorkLive, Live: true, Group: "work-group:e7", Entry: &ui.WorkEntry{ID: "e7", Icon: "terminal", Label: "Running go"},
			Thought: "Each retry logs the attempt and the delay at debug level, so the default level stays quiet."},
	}
}

// screenshot is a stand-in for a pasted picture: a soft gradient.
func screenshot(w, h int) ui.Picture {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(60 + 140*x/w), uint8(110 + 90*y/h), 210, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	p := ui.NewPicture(b.Bytes())
	p.Name = "retry-log.png"
	return p
}

// approvalRow is a command waiting for approval, in the timeline, one of
// its answers risky.
func approvalRow() ui.TimelineRow {
	return ui.TimelineRow{ID: "ap1", Kind: ui.RowWork, Entries: []ui.WorkEntry{{ID: "ap1", Icon: "terminal",
		Approval: &ui.PendingApproval{ID: "req-1", Kind: "command", Detail: "git push -u origin feature/retries", Live: true,
			Options: []ui.ApprovalOption{{Decision: "decline", Label: "Decline"}, {Decision: "acceptForSession", Label: "Always allow this session",
				Warning: "Every later push in this session runs without asking."}, {Decision: "accept", Label: "Approve"}}}}}}
}

// markdownReply has every kind of block T3's markdown draws.
const markdownReply = "## Two ways in\n\n" +
	"The uploader knows how many bytes went out, so it can **report progress** without another goroutine; " +
	"see the [`io.Writer` docs](https://pkg.go.dev/io#Writer) for the shape, and ~~a ticker~~ isn't needed " +
	"([the issue](https://github.com/acme/upload/issues/12) asks for bytes).\n\n" +
	"1. **A callback.** `Send` takes `OnProgress func(sent, total int64)` and calls it after each chunk.\n" +
	"2. **A channel.** `Send` returns a `<-chan Progress` that closes when the upload ends.\n" +
	"   - simpler to cancel\n   - but the caller must drain it\n\n" +
	"| Option | Allocates | Cancels with ctx |\n|:---|:---:|---:|\n| Callback | no | yes |\n| Channel | one per upload | yes |\n\n" +
	"> Both keep *Send's* signature small.\n\n" +
	"> [!TIP]\n> Start with the callback: a channel is easy to build on top of it.\n\n" +
	"★ Insight ─────────────────────────────────────\nA callback runs on the uploader's goroutine,\nso it must not block.\n" +
	"─────────────────────────────────────────────────\n\n" +
	"```go\ntype Progress struct {\n\tSent, Total int64 // bytes\n}\n```\n\n" +
	"- [x] chunked writes\n- [ ] resumable uploads\n\n" +
	"<details open>\n<summary>Why not a ticker</summary>\n\nA ticker fires whether bytes moved or not.\n\n</details>\n\n" +
	"An unset callback costs nothing.[^1]\n\n[^1]: Measured with `go test -bench=Send`.\n"

// planText is a proposed plan long enough to collapse.
const planText = "# Retry the uploader\n\n## Summary\n\nRetry failed chunks with exponential backoff, and report each retry.\n\n" +
	"## Steps\n\n1. Wrap `Send`'s chunk loop in a retry with a 250 ms base delay.\n2. Double the delay up to 4 s, with jitter.\n" +
	"3. Stop after five attempts, or at once on `ErrPermanent`.\n4. Log each retry at debug level.\n5. Cover it with a fake clock.\n\n" +
	"## Risks\n\n- Retrying a chunk the server already stored duplicates it.\n- A long backoff can outlive the caller's deadline.\n" +
	"- The fake clock must cover the jitter.\n\n## Out of scope\n\n- Resumable uploads.\n- Progress reporting.\n"

// quotedReply quotes the reply's tip, as "Cite" writes it.
const quotedReply = "t3-citation://v1/env/thread/a2?text=Start+with+the+callback&start=812&end=835" +
	"&prefix=TIP%0A&suffix=%3A+a+channel+is+easy+to+build&comment=It+also+keeps+Send+synchronous."

// longPrompt is a message tall enough to collapse.
var longPrompt = strings.Repeat("The uploader drops the connection when a chunk takes longer than the timeout, and the caller sees a bare EOF. ", 3) +
	"\n\n" + strings.Repeat("- keep the timeout per attempt\n", 8) + "\nAnd log every retry."

// settledRows is a finished thread with the timeline's other rows: a fork
// divider, a collapsed prompt, a plan, a compaction, a stopped turn with its
// fold open, an error, and a steer.
func settledRows() []ui.TimelineRow {
	return []ui.TimelineRow{
		{ID: "fork", Kind: ui.RowDivider, Label: "Forked from conversation", Icon: "git-fork", Action: "Open source conversation", Thread: "env/source"},
		{ID: "s1", Kind: ui.RowUser, Message: &ui.Message{ID: "s1", Text: longPrompt}},
		{ID: "turn-fold:run-s1", Kind: ui.RowTurnFold, Group: "run-s1", Label: "Worked for 34m 10s"},
		{ID: "plan", Kind: ui.RowPlan, Plan: planText},
		{ID: "s2", Kind: ui.RowUser, Message: &ui.Message{ID: "s2", Text: "Implement it.", GroupedNext: true}},
		{ID: "s3", Kind: ui.RowUser, Message: &ui.Message{ID: "s3", Text: "Keep the timeout per attempt.", GroupedPrev: true, Steer: ui.SteerSteer}},
		{ID: "compaction", Kind: ui.RowCompaction, Label: "Context compacted", Before: "180K", After: "12K"},
		{ID: "turn-fold:run-s2", Kind: ui.RowTurnFold, Group: "run-s2", Label: "You stopped after 42s", Open: true},
		{ID: "work-toggle:x1", Kind: ui.RowWorkToggle, Group: "work-group:x1", Icon: "terminal", Label: "Ran 3 commands", Failed: true, Continues: true},
		{ID: "x2", Kind: ui.RowWork, Entries: []ui.WorkEntry{{ID: "x2", Icon: "circle-alert",
			Error: &ui.EntryError{Label: "Provider error", Message: "The connection to the model was reset. Try again."}}}},
		{ID: "a2", Kind: ui.RowAssistant, Message: &ui.Message{ID: "a2", Text: markdownReply, Meta: true, Copy: true, Badge: "interrupted"}},
		{ID: "interrupt", Kind: ui.RowDivider, Label: "Run interrupted", Icon: "circle-stop", Wrap: true,
			Detail: "The provider stopped answering while the upload's tests ran, so the run was interrupted before it finished."},
		{ID: "s5", Kind: ui.RowUser, Message: &ui.Message{ID: "s5", Text: "Agreed: [Assistant quote](" + quotedReply + ") Go with the callback."}},
		{ID: "s6", Kind: ui.RowUser, Message: &ui.Message{ID: "s6", Text: "Nightly check: run the upload tests against staging.",
			Fold: "Automation", FoldIcon: "calendar-clock"}},
		{ID: "s7", Kind: ui.RowUser, Message: &ui.Message{ID: "s7", Text: "Thanks, the retries look good from here.", Guest: "Ada"}},
		{ID: "s4", Kind: ui.RowUser, Message: &ui.Message{ID: "s4", Files: []ui.File{{ID: "f1", Name: "upload-trace.pdf", Size: 2516582},
			{ID: "f2", Name: "pasted.txt", Size: 900, Pasted: true}}}},
		{ID: "live-activity-row", Kind: ui.RowThinking, Live: true, Label: "Thinking"},
	}
}

func base(selected string) *ui.Scene {
	s := &ui.Scene{
		Tabs:        []string{"Port the T3 Connect clie…", "Gio sidebar: virtualize…", "Watch pairing flow"},
		Title:       "Port the T3 Connect client to Go",
		State:       ui.ThreadRunning,
		StateLabel:  "Running",
		Project:     "tern  ›  t3/go-client",
		Provider:    "Claude",
		Model:       "Claude Opus 5.5",
		Effort:      "High · Fast",
		Runtime:     "Full access",
		Placeholder: "Ask, steer or queue.   /  commands    $  skills    @  files",
		Phase:       -1.1,
	}
	sidebar(s, selected)
	s.Browser = &ui.Browser{Title: "tern", URL: "localhost:8787", Control: "You have control"}
	s.Timeline = ui.Timeline{ID: "running", MoreHistory: true, Root: workspace, Rows: threadRows()}
	s.Run = &ui.RunInfo{Status: "Running · 1m 08s", Model: "Opus 5.5 · High", Elapsed: "5m 20s total", Mode: "Full access", Branch: "t3/go-client", Worktree: "~/.t3/worktrees/tern"}
	s.Changes = []ui.FileChange{
		{Path: "app/t3/rpc.go", Add: 48, Del: 6, Kind: 'M', Selected: true},
		{Path: "app/app.go", Add: 112, Kind: 'A'},
		{Path: "app/t3/relay.go", Add: 9, Del: 3, Kind: 'M'},
		{Path: "app/t3/dpop_test.go", Add: 64, Kind: 'A'},
		{Path: "app/ui/model.go", Add: 21, Del: 2, Kind: 'M'},
		{Path: "poc/README.md", Add: 1, Del: 1, Kind: 'M'},
	}
	return s
}

func runningScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Composer = "also make sure it survives the display going to sleep on"
	s.Queue = []string{"then add a tray icon on Windows with the running count"}
	return s
}

func approvalScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Panel = ui.PanelBrowser
	s.State = ui.ThreadApproval
	s.StateLabel = "Needs approval"
	s.Run.Status = "Waiting"
	s.Active[2].State = ui.ThreadApproval
	s.Active[2].Note = "git push -u origin t3/go-client"
	rows := threadRows()
	s.Timeline = ui.Timeline{ID: "approval", Rows: append(rows[:len(rows)-1], approvalRow())}
	s.Approval = &ui.Approval{Kind: "command", Detail: "git push -u origin t3/go-client", Cwd: "~/.t3/worktrees/tern"}
	s.Picker = &ui.ModelPicker{
		Tabs: []ui.PickerTab{{}, {Provider: "Claude"}, {Provider: "Codex"}},
		Tab:  1,
		Models: []ui.PickerModel{
			{Name: "Claude Opus 5.5", Note: "1M", Fav: true, Selected: true},
			{Name: "Claude Sonnet 5.5", Fav: true},
			{Name: "Claude Fable 5.1"},
			{Name: "Claude Haiku 5.5"},
		},
		Legacy: 3,
	}
	return s
}

// claudeTraits is the options menu for Claude Opus: effort, fast mode and
// the context window, as its option descriptors report them.
func claudeTraits() *ui.Traits {
	return &ui.Traits{Groups: []ui.Trait{
		{Label: "Effort", Sel: 2, Choices: []ui.TraitChoice{{Label: "Low"}, {Label: "Medium"}, {Label: "High"}, {Label: "Extra high"}, {Label: "Max"}, {Label: "Ultrathink"}}},
		{Label: "Fast Mode", Bool: true, On: true},
		{Label: "Context window", Sel: 1, Choices: []ui.TraitChoice{{Label: "200k"}, {Label: "1M", Desc: "Default on Opus"}}},
	}}
}

// traitsScene has the options menu open over the effort button.
func traitsScene() *ui.Scene {
	s := approvalScene()
	s.Picker = nil
	s.Traits = claudeTraits()
	return s
}

// codexScene is the options menu for a Codex model: effort and service tier.
func codexScene() *ui.Scene {
	s := traitsScene()
	s.Provider, s.Model = "Codex", "GPT-5.5"
	s.Traits = &ui.Traits{Groups: []ui.Trait{
		{Label: "Effort", Sel: 2, Choices: []ui.TraitChoice{{Label: "Low"}, {Label: "Medium"}, {Label: "High"}, {Label: "Extra high"}, {Label: "Max"}, {Label: "Ultra"}}},
		{Label: "Service tier", Sel: 1, Choices: []ui.TraitChoice{{Label: "Standard"}, {Label: "Fast", Desc: "2x speed, increased usage"}}},
	}}
	return s
}

func diffScene() *ui.Scene {
	s := runningScene()
	s.Diff = &ui.DiffView{
		Title: "Diff  ·  Turn 4 of 4",
		File:  "app/t3/rpc.go",
		Files: s.Changes,
		Lines: []ui.DiffLine{
			{Kind: '@', Text: "@@ -104,17 +104,37 @@ func (c *Conn) readLoop() error {"},
			{Kind: ' ', Old: 104, New: 104, Text: "func (c *Conn) readLoop() error {"},
			{Kind: ' ', Old: 105, New: 105, Text: "	defer c.shutdown()"},
			{Kind: ' ', Old: 106, New: 106, Text: "	for {"},
			{Kind: '-', Old: 107, Text: "		_, data, err := c.ws.Read(c.ctx)"},
			{Kind: '+', New: 107, Text: "		ctx, cancel := context.WithTimeout(c.ctx, pongTimeout)"},
			{Kind: '+', New: 108, Text: "		_, data, err := c.ws.Read(ctx)"},
			{Kind: '+', New: 109, Text: "		cancel()"},
			{Kind: ' ', Old: 108, New: 110, Text: "		if err != nil {"},
			{Kind: '-', Old: 109, Text: "			return err"},
			{Kind: '+', New: 111, Text: "			// No frame for 15 s: a laptop that slept leaves a socket"},
			{Kind: '+', New: 112, Text: "			// that never closes. Drop it and resubscribe."},
			{Kind: '+', New: 113, Text: "			return c.closeWith(errNoPong)"},
			{Kind: ' ', Old: 110, New: 114, Text: "		}"},
			{Kind: ' ', Old: 111, New: 115, Text: "		msgs, err := decodeFrame(data)"},
			{Kind: ' ', Old: 112, New: 116, Text: "		if err != nil {"},
			{Kind: ' ', Old: 113, New: 117, Text: "			return fmt.Errorf(\"decode frame: %w\", err)"},
			{Kind: ' ', Old: 114, New: 118, Text: "		}"},
			{Kind: ' ', Old: 115, New: 119, Text: "		for _, m := range msgs {"},
			{Kind: '-', Old: 116, Text: "			c.dispatch(m)"},
			{Kind: '+', New: 120, Text: "			if m.Tag == \"Chunk\" {"},
			{Kind: '+', New: 121, Text: "				// The server sends nothing more on this stream until we Ack."},
			{Kind: '+', New: 122, Text: "				c.send(ack{Tag: \"Ack\", RequestID: m.RequestID})"},
			{Kind: '+', New: 123, Text: "			}"},
			{Kind: '+', New: 124, Text: "			c.dispatch(m)"},
			{Kind: ' ', Old: 117, New: 125, Text: "		}"},
			{Kind: ' ', Old: 118, New: 126, Text: "	}"},
			{Kind: ' ', Old: 119, New: 127, Text: "}"},
			{Kind: '@', Text: "@@ -201,6 +209,14 @@ func (c *Conn) Stream(tag string, payload any) *Stream {"},
			{Kind: ' ', Old: 201, New: 209, Text: "	id := c.nextID.Add(1)"},
			{Kind: '+', New: 210, Text: "	st := &Stream{id: id, values: make(chan json.RawMessage, 64)}"},
			{Kind: '+', New: 211, Text: "	c.pending.Store(id, st)"},
		},
	}
	return s
}

func questionScene() *ui.Scene {
	s := base("Watch pairing flow")
	s.Tabs = []string{"Watch pairing flow", "Port the T3 Connect clie…"}
	s.Title = "Watch pairing flow"
	s.State = ui.ThreadQuestion
	s.StateLabel = "Has a question"
	s.Project = "tern  ›  main"
	s.Run.Status = "Waiting for you"
	s.Run.Branch = "main"
	s.Timeline = ui.Timeline{ID: "question", Rows: []ui.TimelineRow{
		{ID: "q1", Kind: ui.RowUser, Message: &ui.Message{ID: "q1", Text: "How should the uploader report progress to the caller?"}},
		{ID: "turn-fold:run-q", Kind: ui.RowTurnFold, Group: "run-q", Label: "Worked for 1m 51s"},
		{ID: "qa", Kind: ui.RowAssistant, Message: &ui.Message{ID: "qa", Text: markdownReply, Meta: true, Copy: true}},
		{ID: "qt", Kind: ui.RowWork, Entries: []ui.WorkEntry{{ID: "qt", Icon: "brain", Label: "Both work. The choice is the caller's.", Expands: true,
			Thought: "Both work. The choice is the caller's."}}},
	}}
	s.Questions = []ui.Question{
		{Header: "Session", Text: "Which session should the watch get?", Options: []string{"30-day, full access", "30-day, read + operate only", "1-hour, renew from phone"}, Picked: 1},
		{Header: "Pairing", Text: "How should the watch receive the credential?", Options: []string{"QR code on desktop", "Typed 6-word code", "Through the iPhone app"}, Picked: 0},
	}
	s.Changes = nil
	return s
}

func draftScene() *ui.Scene {
	s := base("")
	s.Title = "New thread"
	s.State = ui.ThreadIdle
	s.StateLabel = "Draft"
	s.Project = "in tern  ›  new worktree"
	s.Tabs = []string{"New thread", "Port the T3 Connect clie…"}
	s.Timeline = ui.Timeline{}
	s.Composer = "/re"
	s.Images = 2
	s.SuggestHead = "COMMANDS  ·  CLAUDE"
	s.Suggest = []ui.Suggestion{
		{Name: "/review", Hint: "[pr]", Desc: "Review the current diff for bugs"},
		{Name: "/resume", Desc: "Resume an earlier conversation"},
		{Name: "/release-notes", Desc: "Show what changed in this version"},
		{Name: "/frontend:refactor", Hint: "<path>", Desc: "Refactor a component"},
		{Name: "/pr-comments", Desc: "Fetch review comments on this branch"},
	}
	s.Run = nil
	s.Changes = nil
	s.Panel = ui.PanelBrowser
	s.Browser = &ui.Browser{}
	return s
}

func signInScene() *ui.Scene {
	return &ui.Scene{Login: &ui.SignIn{Email: "karim@example.com", Step: 1, Code: "4821"}}
}

func phoneScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Title = "Port the T3 Connect client to Go"
	s.Project = "tern"
	rows := threadRows()
	// The group closed: a phone shows its summary.
	rows[5].Open, rows[5].Continues = false, true
	s.Timeline = ui.Timeline{ID: "phone", Rows: append(append(rows[1:5:5], rows[5]), approvalRow())}
	s.State = ui.ThreadApproval
	s.StateLabel = "Needs approval"
	s.Approval = &ui.Approval{Kind: "command", Detail: "git push -u origin t3/go-client", Cwd: "tern"}
	s.Placeholder = "Steer or queue…"
	return s
}

// settledScene is a finished thread with the timeline's other rows.
func settledScene() *ui.Scene {
	s := base("Gio sidebar: virtualize 1k rows")
	s.Title = "Gio sidebar: virtualize 1k rows"
	s.State, s.StateLabel = ui.ThreadIdle, "Idle"
	s.Run.Status = "Idle"
	s.Timeline = ui.Timeline{ID: "settled", MoreHistory: true, Root: workspace, Rows: settledRows()}
	s.Changes = nil
	return s
}
