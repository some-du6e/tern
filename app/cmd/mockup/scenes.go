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

// Project icons for the fixtures: monograms in T3's colours.
var (
	ternIcon  = ui.ProjectIcon{Text: "TN", Color: "amber"}
	atlasIcon = ui.ProjectIcon{Text: "AS", Color: "sky"}
	notesIcon = ui.ProjectIcon{Text: "NS", Color: "violet"}
)

// cardItem is a fixture card: in tern unless it says otherwise, on the
// workstation, with Claude, offering its controls; the open thread when
// it's titled selected.
func cardItem(r ui.ThreadRow, selected string) ui.SidebarItem {
	if r.Project == "" {
		r.Project, r.Icon = "tern", ternIcon
	}
	r.ID = r.Title
	r.Active = r.Title == selected
	r.Recede = r.Recede && !r.Active
	r.Providers = []string{"Claude"}
	r.Machine, r.MachineIcon = "workstation", "server"
	r.CanSettle, r.CanSnooze, r.CanUnpin = true, true, true
	return ui.SidebarItem{Row: &r}
}

// slimItem is a fixture row on the shelf: snoozed or settled.
func slimItem(shelf string, r ui.ThreadRow, selected string) ui.SidebarItem {
	it := cardItem(r, selected)
	it.Row.Slim, it.Row.Shelf, it.Row.Recede = true, shelf, true
	return it
}

// sidebar fills the sidebar with a row in each of T3's states: pinned,
// approval, input, a guest's request, working, waiting, failed, done, woke
// and seen; the Working shelf open; Snoozed folded; Settled open on two
// rows with more to show. selected is the open thread.
func sidebar(s *ui.Scene, selected string) {
	s.Envs = envs()
	card := func(r ui.ThreadRow) ui.SidebarItem { return cardItem(r, selected) }
	slim := func(shelf string, r ui.ThreadRow) ui.SidebarItem { return slimItem(shelf, r, selected) }
	s.Sidebar = ui.Sidebar{HasProjects: true, Ticking: true, Items: []ui.SidebarItem{
		card(ui.ThreadRow{Title: "Roadmap", Pinned: true, Branch: "main", Age: "2d", Recede: true}),
		card(ui.ThreadRow{Title: "Nightly backup check", Project: "atlas", Icon: atlasIcon, Pinned: true, Status: ui.StatusDone, Label: "Done", Unread: true, Age: "7h"}),
		card(ui.ThreadRow{Title: "Review the stop capability", Status: ui.StatusApproval, Label: "Approval", Branch: "t3/thread-stop", Worktree: true, Age: "2h"}),
		card(ui.ThreadRow{Title: "Watch pairing flow", Status: ui.StatusInput, Label: "Input", Age: "41m"}),
		card(ui.ThreadRow{Title: "Port the T3 Connect client to Go", Status: ui.StatusWorking, Label: "Working", Timer: "1m", Branch: "t3/go-client", Worktree: true, Recede: true,
			PR: &ui.PRBadge{State: "open", Text: "42"}, Age: "now"}),
		card(ui.ThreadRow{Title: "A guest's question", Project: "notes", Icon: notesIcon, Status: ui.StatusRequest, Label: "Request", Age: "3m"}),
		card(ui.ThreadRow{Title: "Relay status polling", Status: ui.StatusFailed, Label: "Failed", Error: "Error occurred", Age: "5h"}),
		card(ui.ThreadRow{Title: "Sidebar virtualization", Status: ui.StatusWoke, Label: "Woke", Branch: "t3/sidebar-virtual", PR: &ui.PRBadge{State: "merged", Text: "38"}, Age: "18m"}),
		card(ui.ThreadRow{Title: "Outline and sources", Project: "notes", Icon: notesIcon, Recede: true, Age: "1d", Draft: true}),
		{Shelf: &ui.Shelf{ID: "working", Label: "Working", Count: 2, Open: true}},
		card(ui.ThreadRow{Shelf: "working", Title: "Restore test", Project: "atlas", Icon: atlasIcon, Status: ui.StatusWorking, Label: "Working", Timer: "12m", Recede: true, Age: "12m"}),
		card(ui.ThreadRow{Shelf: "working", Title: "Wait for the subagents", Status: ui.StatusWaiting, Label: "Waiting", Timer: "40s", Recede: true, Age: "4m"}),
		{Shelf: &ui.Shelf{ID: "snoozed", Label: "Snoozed", Count: 1}},
		{Shelf: &ui.Shelf{ID: "settled", Label: "Settled", Count: 76, Open: true}},
		slim("settled", ui.ThreadRow{Title: "Bounded snapshot history", Age: "1d"}),
		slim("settled", ui.ThreadRow{Title: "Email-code sign-in", Project: "atlas", Icon: atlasIcon, Age: "3d"}),
		{More: 25},
	}}
	s.Net = "workstation  ·  connected through relay in 612 ms  ·  DPoP session renews in 52m"
	s.Stats = "1,024 threads   ·   2 running   ·   ping 38 ms   ·   frame 0.41 ms"
}

// The timeline's fixtures: a made-up project, so nothing real shows.

const retryReply = "Done. `upload.Send` now retries with **exponential backoff**:\n\n" +
	"- the first retry waits 250 ms, doubling up to a 4 s cap;\n" +
	"- it gives up after five attempts and returns the last error;\n" +
	"- `ErrPermanent` stops it at once.\n\n" +
	"Run the tests with:\n\n```bash\ngo test ./upload/...\n```\n"

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
	"see the [`io.Writer` docs](https://pkg.go.dev/io#Writer) for the shape, and ~~a ticker~~ isn't needed.\n\n" +
	"1. **A callback.** `Send` takes `OnProgress func(sent, total int64)` and calls it after each chunk.\n" +
	"2. **A channel.** `Send` returns a `<-chan Progress` that closes when the upload ends.\n" +
	"   - simpler to cancel\n   - but the caller must drain it\n\n" +
	"| Option | Allocates | Cancels with ctx |\n|:---|:---:|---:|\n| Callback | no | yes |\n| Channel | one per upload | yes |\n\n" +
	"> Both keep *Send's* signature small.\n\n" +
	"> [!TIP]\n> Start with the callback: a channel is easy to build on top of it.\n\n" +
	"★ Insight ─────────────────────────────────────\nA callback runs on the uploader's goroutine,\nso it must not block.\n" +
	"─────────────────────────────────────────────────\n\n" +
	"```go\ntype Progress struct {\n\tSent, Total int64 // bytes\n}\n```\n\n" +
	"- [x] chunked writes\n- [ ] resumable uploads\n"

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
		Provider:    "Claude",
		Model:       "Opus 5.5",
		Effort:      "High",
		Speed:       "Fast",
		Runtime:     "full-access",
		Placeholder: "Ask anything. @ files, $ skills, / commands",
		Phase:       -1.1,
	}
	sidebar(s, selected)
	s.Browser = &ui.Browser{Title: "tern", URL: "localhost:8787", Control: "You have control"}
	s.Timeline = ui.Timeline{ID: "running", MoreHistory: true, Rows: threadRows()}
	s.Header = &ui.ChatHeader{Project: "tern", Icon: ternIcon, Server: true, CanRename: true}
	s.DetailsCard = true
	s.Card = &ui.ThreadDetails{
		Run:     &ui.RunContext{Machine: "server", Label: "workstation", Tip: "workstation", Workspace: "folder-git-2", Locked: true},
		OpenIn:  &ui.OpenIn{Editor: "vscode", Label: "Open in VS Code"},
		Scripts: &ui.Scripts{Primary: "Test", Icon: "test", Any: true},
		Branch:  &ui.BranchRow{Label: "t3/go-client"},
		Git:     &ui.GitActions{Label: "Commit, push & PR", Icon: "provider:github"},
		Changes: &ui.Changes{Insertions: 2382, Deletions: 14, Base: "vs main"},
	}
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
	s.Queue = []ui.QueuedMessage{{Text: "then add a tray icon on Windows with the running count"}}
	return s
}

func approvalScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Panel = ui.PanelBrowser
	s.State = ui.ThreadApproval
	for _, it := range s.Sidebar.Items {
		if r := it.Row; r != nil && r.Active {
			r.Status, r.Label, r.Timer, r.Recede = ui.StatusApproval, "Approval", "", false
		}
	}
	rows := threadRows()
	s.Timeline = ui.Timeline{ID: "approval", Rows: append(rows[:len(rows)-1], approvalRow())}
	s.Placeholder = "Send a message to decline and redirect the agent"
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

// traitsScene is the model picker with its effort column: the model's
// options beside the models (ProviderModelPicker.tsx).
func traitsScene() *ui.Scene {
	s := approvalScene()
	s.Traits = claudeTraits()
	s.Traits.Groups[0].Choices[1].Default = true
	return s
}

// codexScene is the picker's column for a Codex model: effort and service tier.
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
	// A thread on the default branch with work to commit: it pushes there.
	s.Card.Run.Workspace = "folder"
	s.Card.Branch.Label = "main"
	s.Card.Git = &ui.GitActions{Label: "Commit & push", Icon: "cloud-upload"}
	s.Card.Changes = &ui.Changes{Insertions: 31, Deletions: 0, Base: "Uncommitted"}
	s.Timeline = ui.Timeline{ID: "question", Rows: []ui.TimelineRow{
		{ID: "q1", Kind: ui.RowUser, Message: &ui.Message{ID: "q1", Text: "How should the uploader report progress to the caller?"}},
		{ID: "turn-fold:run-q", Kind: ui.RowTurnFold, Group: "run-q", Label: "Worked for 1m 51s"},
		{ID: "qa", Kind: ui.RowAssistant, Message: &ui.Message{ID: "qa", Text: markdownReply, Meta: true, Copy: true}},
		{ID: "qt", Kind: ui.RowWork, Entries: []ui.WorkEntry{{ID: "qt", Icon: "brain", Label: "Both work. The choice is the caller's.", Expands: true,
			Thought: "Both work. The choice is the caller's."}}},
	}}
	s.Questions = []ui.Question{
		{Header: "Session", Text: "Which session should the watch get?", Options: []string{"30-day, full access", "30-day, read + operate only", "1-hour, renew from phone"},
			Descs: []string{"", "It can't open a terminal", "The phone renews it while it's near"}, Chosen: []bool{false, true, false}},
		{Header: "Pairing", Text: "How should the watch receive the credential?", Options: []string{"QR code on desktop", "Typed 6-word code", "Through the iPhone app"}, Chosen: []bool{true, false, false}},
	}
	s.Placeholder = "Type your own answer, or leave this blank to use the selected option"
	s.Changes = nil
	return s
}

// draftScene is a new thread with the / menu open over its hero's
// composer: T3 shows a draft as the hero while it has no turns.
func draftScene() *ui.Scene {
	s := heroScene()
	// Its machine row is a menu, and it starts in a new worktree off main,
	// so Git keeps quiet.
	s.Card = &ui.ThreadDetails{
		Run:     &ui.RunContext{Machine: "server", Label: "workstation", Tip: "workstation", Workspace: "folder-git-2", WorkspaceTip: "New worktree"},
		OpenIn:  &ui.OpenIn{Editor: "vscode", Label: "Open in VS Code"},
		Scripts: &ui.Scripts{},
		Branch:  &ui.BranchRow{Label: "From main"},
	}
	s.Composer = "/re"
	s.Thumbs = pictures(2)
	s.Suggest = []ui.Suggestion{
		{Label: "/review", Desc: "Review the current diff for bugs"},
		{Label: "/resume", Desc: "Resume an earlier conversation"},
		{Label: "/release-notes", Desc: "Show what changed in this version"},
		{Prefix: "/skill:", Label: "frontend-refactor", Desc: "Refactor a component", Badge: "Personal", BadgeIcon: "user-round"},
		{Label: "/pr-comments", Desc: "Fetch review comments on this branch"},
	}
	return s
}

// gitMenuScene has the details card's Git options menu open: Commit off
// with a clean tree, Push and Create PR on.
func gitMenuScene() *ui.Scene {
	s := runningScene()
	s.Card.Git = &ui.GitActions{Label: "Push & create PR", Icon: "provider:github"}
	s.Card.Changes = &ui.Changes{Base: "vs main"}
	s.Card.Menu = &ui.CardMenu{For: "git", Rows: []ui.MenuRow{
		{Label: "Commit", Icon: "git-commit-horizontal", Disabled: true, Reason: "Worktree is clean. Make changes before committing."},
		{Label: "Push", Icon: "cloud-upload"},
		{Label: "Create PR", Icon: "provider:github"},
	}}
	return s
}

// popoverScene is a window too narrow for the card inline, so it hangs
// off the header with the branch picker open; the thread is a subagent's,
// so its parent is crumbed before its title.
func popoverScene() *ui.Scene {
	s := runningScene()
	s.Title = "Check the relay's retry budget"
	s.Header.Parent = "Port the T3 Connect client to Go"
	s.Details = true
	s.Card.Picker = &ui.BranchPicker{Highlight: 1, Status: "Showing 4 of 31 refs", Refs: []ui.BranchRef{
		{Name: "t3/go-client", Tag: "current"}, {Name: "main", Tag: "default"},
		{Name: "t3/investigate-release-build-times"}, {Name: "origin/main", Tag: "remote"},
	}}
	return s
}

// gitRunningScene is a short window: the card drops to compact density,
// losing the machine row, but keeps the Git row while its action runs.
func gitRunningScene() *ui.Scene {
	s := runningScene()
	s.Card.Git.Running = &ui.GitProgress{Status: "Running pre-commit hook...", Output: "eslint: 0 problems", Elapsed: "12s"}
	return s
}

// confirmScene asks before pushing to the default branch.
func confirmScene() *ui.Scene {
	s := questionScene()
	s.Confirm = &ui.Confirm{
		Title:       "Commit & push to default ref?",
		Description: `This action will commit and push changes on "main". You can continue on this ref or create a feature ref and run the same action there.`,
		Continue:    "Commit & push to main",
	}
	return s
}

// draftMenuScene is a new thread with its machine row's workspace menu
// open.
func draftMenuScene() *ui.Scene {
	s := draftScene()
	s.Suggest, s.Composer = nil, ""
	s.Card.Menu = &ui.CardMenu{For: "run", Rows: []ui.MenuRow{
		{Kind: ui.MenuGroupLabel, Label: "Workspace"},
		{Kind: ui.MenuRadioRow, Label: "Current checkout", Icon: "folder"},
		{Kind: ui.MenuRadioRow, Label: "New worktree", Icon: "folder-git-2", Checked: true},
	}}
	return s
}

// pullRequestsScene has the thread's pull requests under its branch row,
// unfolded: the open one the agent watches, read in full with its checks
// passed and Merge on offer, then the merged base it stacks on and an
// older closed one. A guest watches the thread through its guest link.
func pullRequestsScene() *ui.Scene {
	s := runningScene()
	s.Header.Guest = &ui.GuestLine{Status: "Sam is a guest"}
	s.Card.PRs = &ui.PullRequests{Others: 2, Expanded: true, Rows: []ui.PullRequestRow{
		{State: "open", Label: "#214: Retry relay reads on a dropped socket", Tip: "PR #214 - Open: Retry relay reads on a dropped socket", Watching: true,
			Detail: &ui.PullRequestDetailRow{Title: "Retry relay reads on a dropped socket", Number: 214,
				Checks: &ui.ChecksMeta{State: "passing"}, Pill: &ui.PullRequestPill{Label: "Merge", Tip: "Merge this pull request (merge)"}}},
		{State: "merged", Label: "#209: Split the relay out of the RPC client", Tip: "PR #209 - Merged: Split the relay out of the RPC client"},
		{State: "closed", Label: "#187: Try a second socket for streams", Tip: "PR #187 - Closed: Try a second socket for streams"},
	}}
	return s
}

// checksScene has the thread's pull request still running its checks,
// their popover open: the failed one first, the running ones, and the
// finished one folded behind Show all.
func checksScene() *ui.Scene {
	s := runningScene()
	s.Card.PRs = &ui.PullRequests{Rows: []ui.PullRequestRow{{State: "open", Label: "#214: Retry relay reads on a dropped socket",
		Detail: &ui.PullRequestDetailRow{Title: "Retry relay reads on a dropped socket", Number: 214,
			Checks: &ui.ChecksMeta{State: "failing", Count: "1/4 checks", Popup: &ui.ChecksPopup{
				Heading: "Some checks were not successful", Summary: "1 of 4 failing", More: "Show all",
				Checks: []ui.CheckRow{
					{Status: "failure", Name: "lint", Label: "Failed", Details: true},
					{Status: "pending", Name: "test (linux)", Label: "Running", Details: true},
					{Status: "pending", Name: "test (windows)", Label: "Running", Details: true},
				},
			}}}}}}
	return s
}

// mergeScene asks before merging the thread's pull request.
func mergeScene() *ui.Scene {
	s := pullRequestsScene()
	s.Alert = &ui.Alert{Title: "Merge pull request?", Description: "This merges #214 using merge.", Action: "Merge"}
	return s
}

// renameScene renames the thread in place in the header; the thread came
// from an agent on a linked machine.
func renameScene() *ui.Scene {
	s := runningScene()
	s.Header.Origin = "laptop"
	s.Header.Renaming = true
	return s
}

func signInScene() *ui.Scene {
	return &ui.Scene{Login: &ui.SignIn{Email: "karim@example.com", Step: 1, Code: "4821"}}
}

func phoneScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Title = "Port the T3 Connect client to Go"
	rows := threadRows()
	// The group closed: a phone shows its summary.
	rows[5].Open, rows[5].Continues = false, true
	s.Timeline = ui.Timeline{ID: "phone", Rows: append(append(rows[1:5:5], rows[5]), approvalRow())}
	s.State = ui.ThreadApproval
	s.Placeholder = "Send a message to decline and redirect the agent"
	return s
}

// row is the fixture sidebar's row titled title.
func row(s *ui.Scene, title string) *ui.ThreadRow {
	for _, it := range s.Sidebar.Items {
		if it.Row != nil && it.Row.Title == title {
			return it.Row
		}
	}
	panic("no row " + title)
}

// sidebarHoverScene has a card hovered, its controls in place of its
// status, and its hover card out beside it.
func sidebarHoverScene() *ui.Scene {
	s := runningScene()
	r := row(s, "Review the stop capability")
	r.Hover, r.Provider, r.Model, r.Handoff, r.Terminals = true, "Claude", "Claude Opus 5.5", "Handed off from Codex", 1
	row(s, "Email-code sign-in").Hover = true
	return s
}

// sidebarMenuScene has a row's context menu open from the keyboard.
func sidebarMenuScene() *ui.Scene {
	s := runningScene()
	sub := func(labels ...string) []ui.MenuItem {
		var out []ui.MenuItem
		for _, l := range labels {
			out = append(out, ui.MenuItem{ID: l, Label: l})
		}
		return out
	}
	snooze := sub("In 1 hour", "In 3 hours", "This evening", "Tomorrow", "Next week")
	s.Menu = &ui.ContextMenu{At: image.Pt(120, 262), Keyboard: true, Items: []ui.MenuItem{
		{ID: "pin", Label: "Pin thread", Icon: "pin"},
		{ID: "settle", Label: "Settle thread", Icon: "circle-check"},
		{ID: "snooze", Label: "Snooze", Icon: "clock", Children: snooze},
		{ID: "rename", Label: "Rename thread", Icon: "pencil", Sep: true},
		{ID: "regenerate-title", Label: "Regenerating…", Icon: "refresh-cw", Off: true},
		{ID: "mark-unread", Label: "Mark unread", Icon: "mail-open"},
		{ID: "filter-by-project", Label: "Filter by tern", Icon: "folder-tree"},
		{ID: "copy", Label: "Copy", Icon: "copy", Sep: true, Children: sub("Path", "Branch", "Thread", "Thread ID")},
		{ID: "archive", Label: "Archive thread", Icon: "archive", Sep: true},
		{ID: "delete", Label: "Delete", Icon: "trash", Danger: true},
	}}
	return s
}

// sidebarFilterScene has the project filter open.
func sidebarFilterScene() *ui.Scene {
	s := runningScene()
	s.Sidebar.Filter = &ui.ProjectFilter{Items: []ui.FilterItem{
		{Label: "All projects", Selected: true},
		{ID: "tern", Label: "tern", Icon: ternIcon},
		{ID: "atlas", Label: "atlas", Icon: atlasIcon},
		{ID: "notes", Label: "notes", Icon: notesIcon},
	}}
	return s
}

// sidebarSnoozeScene has a card's snooze menu open.
func sidebarSnoozeScene() *ui.Scene {
	s := runningScene()
	r := row(s, "Watch pairing flow")
	r.Hover = true
	var cs []ui.MenuItem
	for _, c := range [][2]string{{"In 1 hour", "11:00 AM"}, {"In 3 hours", "1:00 PM"}, {"This evening", "6:00 PM"}, {"Tomorrow", "9:00 AM"}, {"Next week", "Mon 9:00 AM"}} {
		cs = append(cs, ui.MenuItem{ID: c[0], Label: c[0], Detail: c[1]})
	}
	s.Sidebar.Snooze = &ui.SnoozeMenu{Row: r.ID, Choices: cs}
	return s
}

// sidebarSearchScene is the sidebar searching: its results in place of
// the list, the first highlighted, a settled one quieter with the number
// of the pull request the query matched, and two found by their messages,
// one cut short.
func sidebarSearchScene() *ui.Scene {
	s := runningScene()
	s.Sidebar.Query = "pairing"
	s.Sidebar.Results = []ui.ThreadRow{
		{ID: "1", Title: "Watch pairing flow", Project: "tern", Icon: ternIcon, Age: "41m"},
		{ID: "2", Title: "Pairing URL parser", Project: "tern", Icon: ternIcon, Age: "3d", Shelf: "settled",
			PR: &ui.PRBadge{State: "merged", Text: "128"}},
		{ID: "3", Title: "Relay handshake", Project: "tern", Icon: ternIcon, Age: "2h",
			Excerpt: &ui.Excerpt{User: true, Snippet: "…retry the pairing once the code expires", Marks: [][2]int{{11, 18}}}},
		{ID: "4", Title: "Notes app sync", Project: "notes", Icon: notesIcon, Age: "5d",
			Excerpt: &ui.Excerpt{Snippet: "The pairing screen now shows the code and waits for the other device to confirm it", Marks: [][2]int{{4, 11}}}},
	}
	s.Sidebar.Items = nil
	return s
}

// sidebarDraftScene has the new thread's unsent draft atop the list,
// hovered, and the footer's undo notice after two settles.
func sidebarDraftScene() *ui.Scene {
	s := runningScene()
	s.Sidebar.Items = append([]ui.SidebarItem{
		{Draft: &ui.DraftRow{ID: "draft", Project: "notes", Icon: notesIcon, Preview: "Sketch the onboarding copy", Hover: true}},
		{Divider: true},
	}, s.Sidebar.Items...)
	s.Sidebar.Undo = "Settled 2"
	return s
}

// sidebarJumpScene has mod held: the first nine rows show their jump keys.
func sidebarJumpScene() *ui.Scene {
	s := runningScene()
	n := 0
	for _, it := range s.Sidebar.Items {
		if it.Row != nil && n < 9 {
			n++
			it.Row.Jump = n
		}
	}
	s.Sidebar.JumpHints = true
	return s
}

// sidebarEmptyScene is the list scoped to a project with nothing in it.
func sidebarEmptyScene() *ui.Scene {
	s := runningScene()
	s.Sidebar.Items = nil
	s.Sidebar.Scope = []ui.ProjectIcon{notesIcon}
	s.Sidebar.ScopeLabel = "Filter threads by project: notes"
	s.Sidebar.Empty = &ui.SidebarEmpty{Text: "No threads in notes yet", Button: "Show all projects"}
	return s
}

// renamed is the row sidebarStatesScene renames, and its title in the
// field.
const renamed = "Choose the snapshot format"

// sidebarStatesScene has two rows picked out of the list, a title being
// renamed, a row with the keyboard's focus ring, and the labels the main
// list doesn't show: Setting up, Goal and Limited.
func sidebarStatesScene() *ui.Scene {
	s := runningScene()
	card := func(r ui.ThreadRow) ui.SidebarItem { return cardItem(r, s.Title) }
	s.Sidebar.Items = []ui.SidebarItem{
		card(ui.ThreadRow{Title: "Settings page copy", Project: "notes", Icon: notesIcon, Selected: true, Age: "3h"}),
		card(ui.ThreadRow{Title: "Bounded snapshot history", Selected: true, Branch: "t3/snapshots", Age: "5h"}),
		card(ui.ThreadRow{Title: renamed, Age: "1d"}),
		card(ui.ThreadRow{Title: "Pairing URL parser", Focus: true, Age: "2d"}),
		card(ui.ThreadRow{Title: "Bootstrap the worktree", Status: ui.StatusWorking, Label: "Setting up", Timer: "8s", Recede: true, Age: "now"}),
		card(ui.ThreadRow{Title: "Ship the release notes", Project: "atlas", Icon: atlasIcon, Status: ui.StatusWorking, Label: "Goal", Timer: "4h 12m", Recede: true, Age: "4h"}),
		card(ui.ThreadRow{Title: "Long refactor of the relay", Status: ui.StatusLimited, Label: "Limited", Error: "Usage limit reached", Limited: true, Age: "20m"}),
	}
	s.Sidebar.Renaming = renamed
	return s
}

// sidebarShelvesScene has the Snoozed and Settled shelves open: snoozed
// rows with their wake times, one hovered for its Wake button, and settled
// ones, one hovered for Un-settle, with more to show.
func sidebarShelvesScene() *ui.Scene {
	s := runningScene()
	card := func(r ui.ThreadRow) ui.SidebarItem { return cardItem(r, s.Title) }
	slim := func(shelf string, r ui.ThreadRow) ui.SidebarItem { return slimItem(shelf, r, s.Title) }
	s.Sidebar.Items = []ui.SidebarItem{
		card(ui.ThreadRow{Title: "Port the T3 Connect client to Go", Status: ui.StatusWorking, Label: "Working", Timer: "1m", Branch: "t3/go-client", Worktree: true, Age: "now"}),
		card(ui.ThreadRow{Title: "Watch pairing flow", Status: ui.StatusInput, Label: "Input", Age: "41m"}),
		{Shelf: &ui.Shelf{ID: "working", Label: "Working", Count: 2}},
		{Shelf: &ui.Shelf{ID: "snoozed", Label: "Snoozed", Count: 2, Open: true}},
		slim("snoozed", ui.ThreadRow{Title: "Follow up on the flaky test", Wake: "3h", Age: "2d", Hover: true}),
		slim("snoozed", ui.ThreadRow{Title: "Revisit the cache keys", Project: "atlas", Icon: atlasIcon, Wake: "2d", Age: "5d"}),
		{Shelf: &ui.Shelf{ID: "settled", Label: "Settled", Count: 40, Open: true}},
		slim("settled", ui.ThreadRow{Title: "Bounded snapshot history", Age: "1d", Hover: true}),
		slim("settled", ui.ThreadRow{Title: "Email-code sign-in", Project: "atlas", Icon: atlasIcon, Age: "3d"}),
		{More: 25},
	}
	return s
}

// sidebarSearchingScene is a search no title matches while the servers
// are still looking through the messages.
func sidebarSearchingScene() *ui.Scene {
	s := runningScene()
	s.Sidebar.Query, s.Sidebar.Items, s.Sidebar.NoResults = "keychain", nil, "Searching thread messages…"
	return s
}

// sidebarNoResultsScene is a search nothing matches.
func sidebarNoResultsScene() *ui.Scene {
	s := sidebarSearchingScene()
	s.Sidebar.NoResults = "No threads found"
	return s
}

// sidebarNoProjectsScene is a machine with no projects yet: no filter, New
// thread off, and the list saying so.
func sidebarNoProjectsScene() *ui.Scene {
	s := draftScene()
	s.Sidebar = ui.Sidebar{NoNew: true, Empty: &ui.SidebarEmpty{Text: "No projects yet"}}
	return s
}

// settledScene is a finished thread with the timeline's other rows.
func settledScene() *ui.Scene {
	s := base("Gio sidebar: virtualize 1k rows")
	s.Title = "Gio sidebar: virtualize 1k rows"
	s.State = ui.ThreadIdle
	s.Timeline = ui.Timeline{ID: "settled", MoreHistory: true, Rows: settledRows()}
	s.Changes = nil
	return s
}
