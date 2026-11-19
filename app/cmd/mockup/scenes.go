package main

import (
	"image"

	"github.com/some-du6e/tern/app/ui"
)

func envs() []ui.Env {
	return []ui.Env{{Label: "workstation", Status: ui.EnvOnline}, {Label: "buildbox", Status: ui.EnvOnline}, {Label: "macbook", Status: ui.EnvOffline}}
}

func sidebar(s *ui.Scene, selected string) {
	s.Envs = envs()
	s.Filter = "All projects"
	s.Pinned = []ui.ThreadRow{
		{Title: "tern roadmap", Project: "tern", Branch: "main", Age: "2d"},
		{Title: "Fleet: buildbox nightly backups", Project: "home-lab", Branch: "main", Env: "buildbox", Unread: true, Age: "7h"},
	}
	s.Active = []ui.ThreadRow{
		{Title: "Review: thread.stop capability", Project: "t3code", State: ui.ThreadApproval, Note: "pnpm test --filter server", Age: "2h"},
		{Title: "Watch pairing flow", Project: "tern", State: ui.ThreadQuestion, Note: "Which session should the watch get?", Age: "41m"},
		{Title: "Port the T3 Connect client to Go", Project: "tern", Branch: "t3/go-client", State: ui.ThreadRunning, Timer: "1m 08s", Age: "now"},
		{Title: "Restore test: vaultwarden snapshot", Project: "home-lab", Branch: "main", Env: "buildbox", State: ui.ThreadRunning, Timer: "12m", Age: "12m"},
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

const goTestOut = `ok   github.com/some-du6e/tern/app/t3        0.412s
ok   github.com/some-du6e/tern/app/t3/rpc    0.218s
--- replay: relay-exchange.jsonl (41 frames)
    dpop-token     201  31ms
    connect        200  188ms
    oauth/token    200  94ms
    ws ticket      404  -> fallback /api/auth/websocket-ticket  200`

const rpcDiff = `@@ -112,9 +112,22 @@ func (c *Conn) readLoop() {
 	for {
-		_, data, err := c.ws.Read(ctx)
+		ctx, cancel := context.WithTimeout(c.ctx, pongTimeout)
+		_, data, err := c.ws.Read(ctx)
+		cancel()
 		if err != nil {
-			return err
+			// No frame for 15 s: a laptop that slept leaves a socket
+			// that never closes. Drop it and resubscribe.
+			c.closeWith(errNoPong)
+			return`

func threadItems() []ui.Item {
	return []ui.Item{
		{Kind: ui.ItemUser, Text: "Port poc/public/t3.js to Go under app/t3. Keep the DPoP rules from docs/t3-connect.md and add a test that replays a recorded relay exchange."},
		{Kind: ui.ItemFold, Text: "Worked for 4m 12s", Meta: "18 tool calls  ·  6 files"},
		{Kind: ui.ItemAssistant, Text: "Ported. `app/t3` now has **Clerk**, **DPoP**, **Relay** and the RPC socket.\n\n" +
			"- `dpop.go` signs proofs with a P-256 key, and the thumbprint test pins the RFC 7638 member order.\n" +
			"- `relay.go` caches the 30-minute relay token and renews it a minute early.\n" +
			"- `rpc.go` acks every `Chunk` and treats 15 s without a pong as a dead socket.\n\n" +
			"One thing to know: nightly `0.1.3-nightly.20261023` still answers 404 on `/api/auth/websocket-ticket/rpc`, so the client falls back to the bare route."},
		{Kind: ui.ItemUser, Text: "nice. wire it into the UI and keep the socket alive across sleep"},
		{Kind: ui.ItemThought, Text: "A live socket outlives its session, so renewal only matters for tickets and uploads. On resume, resubscribe with afterSequence and wait for synchronized instead of refetching the snapshot."},
		{Kind: ui.ItemTool, Tool: ui.ToolSearch, Title: "rg -n 'onclose|visibilitychange' poc/public/app.js", Meta: "4 matches"},
		{Kind: ui.ItemTool, Tool: ui.ToolEdit, Title: "app/t3/rpc.go", Meta: "+48 −6", Body: rpcDiff},
		{Kind: ui.ItemTool, Tool: ui.ToolEdit, Title: "app/app.go", Meta: "+112 −0"},
		{Kind: ui.ItemTool, Tool: ui.ToolCommand, Title: "go test ./t3/...", Meta: "exit 0  ·  1.9s", Open: true, Body: goTestOut},
		{Kind: ui.ItemTodo, Todos: []ui.Todo{
			{Text: "Reconnect with afterSequence on resume", Done: true},
			{Text: "Renew the DPoP session 3 minutes early", Done: true},
			{Text: "Feed shell and thread streams into the view model", Now: true},
			{Text: "Draft persistence per thread"},
		}},
		{Kind: ui.ItemTool, Tool: ui.ToolCommand, Title: "go build -o bin/tern ./cmd/tern", Live: true, Meta: "3.1s"},
		{Kind: ui.ItemThinking, Live: true, Text: "Gio delivers app.DestroyEvent on Android but only a stage change on desktop, so the resume hook goes on…"},
	}
}

func base(selected string) *ui.Scene {
	s := &ui.Scene{
		Tabs:        []string{"Port the T3 Connect clie…", "Gio sidebar: virtualize…", "Watch pairing flow"},
		Title:       "Port the T3 Connect client to Go",
		State:       ui.ThreadRunning,
		Provider:    "Claude",
		Model:       "Claude Opus 5.5",
		Effort:      "High · Fast",
		Runtime:     "Full access",
		Placeholder: "Ask, steer or queue.   /  commands    $  skills    @  files",
		Phase:       -1.1,
		MoreAbove:   true,
	}
	sidebar(s, selected)
	s.Items = threadItems()
	s.Header = &ui.ChatHeader{Project: "tern", Icon: ui.ProjectIcon{Monogram: "TN", Color: "sky"}, Server: true, CanRename: true}
	s.DetailsCard = true
	s.Card = &ui.ThreadDetails{
		Run:     &ui.RunContext{Machine: "server", Label: "workstation", Tip: "workstation", Workspace: "folder-git-2", Locked: true},
		OpenIn:  &ui.OpenIn{Editor: "vscode", Label: "Open in VS Code"},
		Scripts: &ui.Scripts{Primary: "Test", Icon: "test", Any: true},
		Branch:  &ui.BranchRow{Label: "t3/go-client"},
		Git:     &ui.GitActions{Label: "Commit, push & PR", Icon: "provider:github"},
		// tern has no Diff surface for the row to open yet, so it's off,
		// as in the app.
		Changes: &ui.Changes{Insertions: 2382, Deletions: 14, Base: "vs main", Disabled: true},
	}
	return s
}

// changedFiles are the files the diff window lists.
func changedFiles() []ui.FileChange {
	return []ui.FileChange{
		{Path: "app/t3/rpc.go", Add: 48, Del: 6, Kind: 'M', Selected: true},
		{Path: "app/app.go", Add: 112, Kind: 'A'},
		{Path: "app/t3/relay.go", Add: 9, Del: 3, Kind: 'M'},
		{Path: "app/t3/dpop_test.go", Add: 64, Kind: 'A'},
		{Path: "app/ui/model.go", Add: 21, Del: 2, Kind: 'M'},
		{Path: "poc/README.md", Add: 1, Del: 1, Kind: 'M'},
	}
}

func runningScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Composer = "also make sure it survives the display going to sleep on"
	s.Queue = []string{"then add a tray icon on Windows with the running count"}
	return s
}

func approvalScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.State = ui.ThreadApproval
	s.Active[2].State = ui.ThreadApproval
	s.Active[2].Note = "git push -u origin t3/go-client"
	items := threadItems()
	items = items[:len(items)-2]
	items = append(items, ui.Item{Kind: ui.ItemTool, Tool: ui.ToolCommand, Title: "git push -u origin t3/go-client", Meta: "waiting for you"})
	s.Items = items
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
		Files: changedFiles(),
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
	s.MoreAbove = false
	// A thread on the default branch with work to commit: it pushes there.
	s.Card.Run.Workspace = "folder"
	s.Card.Branch.Label = "main"
	s.Card.Git = &ui.GitActions{Label: "Commit & push", Icon: "cloud-upload"}
	s.Card.Changes = &ui.Changes{Insertions: 31, Deletions: 0, Base: "Uncommitted"}
	s.Items = []ui.Item{
		{Kind: ui.ItemUser, Text: "design how the watch gets a session without ever touching Clerk"},
		{Kind: ui.ItemFold, Text: "Worked for 1m 51s", Meta: "9 tool calls"},
		{Kind: ui.ItemAssistant, Text: "## Two ways in\n\n" +
			"The desktop app is already signed in through T3 Connect, so it can mint a pairing credential with `POST /api/auth/pairing-token` and hand it to the watch over a QR code. The watch trades it once at `/oauth/token` and never sees Clerk, the relay or DPoP.\n\n" +
			"- **30-day bearer session.** One exchange, then the watch reconnects on every wrist raise with one ticket request.\n" +
			"- **Scoped pairing.** Same flow with `orchestration:read` and `orchestration:operate` only, so a lost watch can't open a terminal.\n\n" +
			"Revoking is the same either way: `POST /api/auth/clients/revoke` from the desktop, and close the socket, since a live socket outlives its session."},
		{Kind: ui.ItemThought, Text: "Both work. The scope decision is the user's, not mine."},
	}
	s.Questions = []ui.Question{
		{Header: "Session", Text: "Which session should the watch get?", Options: []string{"30-day, full access", "30-day, read + operate only", "1-hour, renew from phone"}, Picked: 1},
		{Header: "Pairing", Text: "How should the watch receive the credential?", Options: []string{"QR code on desktop", "Typed 6-word code", "Through the iPhone app"}, Picked: 0},
	}
	return s
}

func draftScene() *ui.Scene {
	s := base("")
	s.Title = "New thread"
	s.State = ui.ThreadIdle
	// A draft: its title is plain text, its machine row a menu, and it
	// starts in a new worktree off main, so Git keeps quiet.
	s.Header.Server = false
	s.Card = &ui.ThreadDetails{
		Run:     &ui.RunContext{Machine: "server", Label: "workstation", Tip: "workstation", Workspace: "folder-git-2", WorkspaceTip: "New worktree"},
		OpenIn:  &ui.OpenIn{Editor: "vscode", Label: "Open in VS Code"},
		Scripts: &ui.Scripts{},
		Branch:  &ui.BranchRow{Label: "From main"},
	}
	s.Tabs = []string{"New thread", "Port the T3 Connect clie…"}
	s.MoreAbove = false
	s.Items = []ui.Item{{Kind: ui.ItemNotice, Text: "The first message starts the thread."}}
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
		{Kind: ui.MenuRadioRow, Label: "Previous worktree", Icon: "history", Detail: "t3/investigate-release-build-times"},
	}}
	return s
}

// titleMenuScene has the thread's actions menu open under its title, as
// a click on the title opens it; the thread came from an agent on a
// linked machine, so its badge follows the title.
func titleMenuScene() *ui.Scene {
	s := runningScene()
	s.Header.Origin = "laptop"
	s.Menu = &ui.ContextMenu{At: image.Pt(353, 41), Items: []ui.MenuItem{
		{ID: "new-thread-on-branch", Label: "New thread on t3/go-client", Icon: "message-square-plus"},
		{ID: "pin", Label: "Pin thread", Icon: "pin"},
		{ID: "settle", Label: "Settle thread", Icon: "circle-check"},
		{ID: "rename", Label: "Rename thread", Icon: "pencil", Sep: true},
		{ID: "regenerate-title", Label: "Regenerate title", Icon: "refresh-cw"},
		{ID: "mark-unread", Label: "Mark unread", Icon: "mail-open"},
		{ID: "auto-settle", Label: "Auto-settle behavior", Icon: "timer", Children: []ui.MenuItem{
			{ID: "auto-settle:enabled", Label: "Follow settings", Radio: true, Checked: true},
			{ID: "auto-settle:disabled", Label: "Never", Radio: true},
		}},
		{ID: "copy", Label: "Copy", Icon: "copy", Sep: true, Children: []ui.MenuItem{
			{ID: "copy-path", Label: "Path", Icon: "folder"},
			{ID: "copy-branch", Label: "Branch", Icon: "git-branch"},
			{ID: "copy-thread", Label: "Thread", Icon: "message-square"},
			{ID: "copy-thread-id", Label: "Thread ID", Icon: "hash"},
		}},
		{ID: "archive", Label: "Archive thread", Icon: "archive", Off: true, Sep: true},
		{ID: "delete", Label: "Delete", Icon: "trash", Danger: true},
	}}
	return s
}

// branchPickerScene has the inline card's branch picker open, the
// highlight on its first ref.
func branchPickerScene() *ui.Scene {
	s := runningScene()
	s.Card.Picker = &ui.BranchPicker{Status: "Showing 5 of 31 refs", Refs: []ui.BranchRef{
		{Name: "t3/go-client", Tag: "current"}, {Name: "main", Tag: "default"},
		{Name: "t3/investigate-release-build-times", Tag: "worktree"}, {Name: "t3/sidebar-virtual"}, {Name: "origin/main", Tag: "remote"},
	}}
	return s
}

// noProjectScene is a new thread in No project: its own grey icon on the
// crumb, a plain title, and a card for its folder alone, which has no Git
// and no scripts yet.
func noProjectScene() *ui.Scene {
	s := draftScene()
	s.Suggest, s.Composer, s.Images = nil, "", 0
	s.Header = &ui.ChatHeader{Project: "No project", Icon: ui.ProjectIcon{Lucide: "message-square-dashed", Color: "gray"}}
	s.Card = &ui.ThreadDetails{
		Run:     &ui.RunContext{Machine: "server", Label: "workstation", Tip: "workstation", Workspace: "folder-git-2", WorkspaceTip: "New worktree"},
		OpenIn:  &ui.OpenIn{Label: "Open in editor", Disabled: true},
		Scripts: &ui.Scripts{},
	}
	return s
}

// initGitScene is a thread in a folder that isn't a repository: Initialize
// Git in place of the branch, and the project's scripts as Actions.
func initGitScene() *ui.Scene {
	s := runningScene()
	s.Card.Scripts = &ui.Scripts{Any: true}
	s.Card.Branch, s.Card.Git, s.Card.Changes = nil, nil, nil
	s.Card.InitGit = true
	return s
}

// itemsMenuScene is a card with every row turned off: "Choose what to
// show", with the menu it opens.
func itemsMenuScene() *ui.Scene {
	s := runningScene()
	s.Card = &ui.ThreadDetails{Menu: &ui.CardMenu{For: "items", Rows: []ui.MenuRow{
		{Kind: ui.MenuGroupLabel, Label: "Show in thread details"},
		{Kind: ui.MenuCheckRow, Label: "Environment and worktree"}, {Kind: ui.MenuCheckRow, Label: "Open in editor"},
		{Kind: ui.MenuCheckRow, Label: "Project scripts"}, {Kind: ui.MenuCheckRow, Label: "Branch and pull request"},
		{Kind: ui.MenuCheckRow, Label: "Git actions"}, {Kind: ui.MenuCheckRow, Label: "Changes", Checked: true},
		{Kind: ui.MenuSeparator}, {Label: "Reset to default"},
	}}}
	return s
}

// subagentScene is a subagent's thread in a narrow window: its parent
// folds to "..." to leave the title room.
func subagentScene() *ui.Scene {
	s := runningScene()
	s.SidebarHidden = true
	s.Title = "Check the relay's retry budget against the sleep handler"
	s.Header.Parent = "Port the T3 Connect client to Go"
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
	return &ui.Scene{Login: &ui.SignIn{Email: "dev@example.com", Step: 1, Code: "4821"}}
}

func phoneScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Title = "Port the T3 Connect client to Go"
	s.MoreAbove = false
	items := threadItems()
	s.Items = []ui.Item{items[1], items[2], items[3], items[4], items[5], {Kind: ui.ItemTool, Tool: ui.ToolEdit, Title: "app/t3/rpc.go", Meta: "+48 −6"}, items[7], items[8], items[10]}
	s.Items[7].Open = false // test output wraps badly at phone width
	s.State = ui.ThreadApproval
	s.Approval = &ui.Approval{Kind: "command", Detail: "git push -u origin t3/go-client", Cwd: "tern"}
	s.Placeholder = "Steer or queue…"
	return s
}
