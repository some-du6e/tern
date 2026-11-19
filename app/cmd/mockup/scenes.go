package main

import "github.com/some-du6e/tern/app/ui"

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
		StateLabel:  "Running",
		Project:     "tern  ›  t3/go-client",
		Provider:    "Claude",
		Model:       "Claude Opus 5.5",
		Effort:      "High · Fast",
		Runtime:     "Full access",
		Placeholder: "Ask, steer or queue.   /  commands    $  skills    @  files",
		Phase:       -1.1,
		MoreAbove:   true,
	}
	sidebar(s, selected)
	s.Browser = &ui.Browser{Title: "tern", URL: "localhost:8787", Control: "You have control"}
	s.Items = threadItems()
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
	s.MoreAbove = false
	s.Run.Status = "Waiting for you"
	s.Run.Branch = "main"
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
	s.Run = nil
	s.Changes = nil
	s.Panel = ui.PanelBrowser
	s.Browser = &ui.Browser{}
	return s
}

// The gates (docs/ui/11-empty-auth.md). Their fields are editors, so
// their shots type into them first.

func signInScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GateSignIn, Login: &ui.SignIn{}}
}

func signInErrorScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GateSignIn, Login: &ui.SignIn{FieldError: "Couldn't find your account."}}
}

func codeScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GateSignIn, Login: &ui.SignIn{Step: ui.SignInEmailCode, Email: "you@example.com", ResendIn: 24, Other: true}}
}

func passwordScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GateSignIn, Login: &ui.SignIn{Step: ui.SignInPassword, Email: "you@example.com", Other: true, FieldError: "Password is incorrect. Try again, or use another method."}}
}

func pairScene() *ui.Scene { return &ui.Scene{Gate: ui.GatePairing, Pair: &ui.Pairing{}} }

func pairErrorScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GatePairing, Pair: &ui.Pairing{Error: "Invalid pairing token. Check the token and try again."}}
}

func bootScene() *ui.Scene { return &ui.Scene{Gate: ui.GateBooting} }

func bootFailedScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GateBootFailed, Failure: &ui.Failure{Open: true, Report: "open store: permission denied\n\nVersion: 0.1.0\nPlatform: linux amd64\nTime: 2026-11-18T17:28:53.152Z"}}
}

func sessionErrorScene() *ui.Scene {
	return &ui.Scene{Gate: ui.GateSessionError, Failure: &ui.Failure{
		Message: "List environments failed (500) RelayInternalError.",
		Report:  "tern 0.1.0\nTime: 2026-11-18T17:27:22.221Z\n\nList environments failed (500) RelayInternalError",
	}}
}

func phoneScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Title = "Port the T3 Connect client to Go"
	s.Project = "tern"
	s.MoreAbove = false
	items := threadItems()
	s.Items = []ui.Item{items[1], items[2], items[3], items[4], items[5], {Kind: ui.ItemTool, Tool: ui.ToolEdit, Title: "app/t3/rpc.go", Meta: "+48 −6"}, items[7], items[8], items[10]}
	s.Items[7].Open = false // test output wraps badly at phone width
	s.State = ui.ThreadApproval
	s.StateLabel = "Needs approval"
	s.Approval = &ui.Approval{Kind: "command", Detail: "git push -u origin t3/go-client", Cwd: "tern"}
	s.Placeholder = "Steer or queue…"
	return s
}
