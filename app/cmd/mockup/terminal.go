package main

import (
	"strings"

	"github.com/charmbracelet/x/vt"

	"github.com/some-du6e/tern/app/ui"
)

// The terminal drawer's states (docs/ui/09-terminal.md §4): one terminal
// under its floating toolbar, two groups in the list with a stacked pair
// showing, no terminals, and the close confirmation.

const prompt = "\x1b[1;34m~/project\x1b[0m \x1b[35mmain\x1b[0m \x1b[32m$\x1b[0m "

// shellSession is a shell after a few commands: T3's ANSI colours, the
// text styles, and a link.
func shellSession(cols, rows int) *vt.SafeEmulator {
	e := vt.NewSafeEmulator(cols, rows)
	var b strings.Builder
	b.WriteString(prompt + "ls\r\n")
	b.WriteString("\x1b[1;34mapp\x1b[0m  \x1b[1;34mdocs\x1b[0m  \x1b[1;32mbuild.sh\x1b[0m  go.mod  go.sum  README.md\r\n")
	b.WriteString(prompt + "go test ./...\r\n")
	b.WriteString("\x1b[32mok\x1b[0m   example.com/app/core   0.412s\r\n")
	b.WriteString("\x1b[32mok\x1b[0m   example.com/app/ui     2.087s\r\n")
	b.WriteString("\x1b[31mFAIL\x1b[0m example.com/app/net    0.318s\r\n")
	b.WriteString(prompt + "colors\r\n")
	for i := 0; i < 8; i++ {
		b.WriteString("\x1b[4" + string(rune('0'+i)) + "m    ")
	}
	b.WriteString("\x1b[0m\r\n")
	for i := 0; i < 8; i++ {
		b.WriteString("\x1b[10" + string(rune('0'+i)) + "m    ")
	}
	b.WriteString("\x1b[0m\r\n")
	b.WriteString("\x1b[1mbold\x1b[0m \x1b[3mitalic\x1b[0m \x1b[4munderline\x1b[0m \x1b[9mstrike\x1b[0m \x1b[2mfaint\x1b[0m \x1b[7minverse\x1b[0m \x1b[31mred\x1b[0m \x1b[91mbright red\x1b[0m\r\n")
	b.WriteString("docs at https://example.com/docs\r\n")
	b.WriteString(prompt)
	e.Write([]byte(b.String()))
	return e
}

// logStream is a dev server's output, long enough to scroll back.
func logStream(cols, rows int) *vt.SafeEmulator {
	e := vt.NewSafeEmulator(cols, rows)
	var b strings.Builder
	b.WriteString(prompt + "npm run dev\r\n")
	for i, path := range []string{"/", "/api/items", "/api/items/3", "/assets/app.js", "/api/health", "/api/items", "/settings", "/api/user"} {
		code := "\x1b[32m200\x1b[0m"
		if i == 6 {
			code = "\x1b[33m304\x1b[0m"
		}
		b.WriteString("\x1b[2m12:0" + string(rune('0'+i)) + "\x1b[0m GET " + path + " " + code + "\r\n")
	}
	e.Write([]byte(b.String()))
	return e
}

// terminalScene is the running thread with its drawer open on one
// terminal.
func terminalScene() *ui.Scene {
	s := runningScene()
	s.Queue = nil
	s.Term = &ui.TermView{Thread: "mockup", Height: 280, Active: "term-1",
		Groups: []ui.TermGroup{{Tabs: []ui.TermTab{{ID: "term-1", Label: "Terminal 1"}}}},
		Panes:  []ui.TermPane{{ID: "term-1", Label: "Terminal 1", Screen: shellSession(150, 15)}}}
	return s
}

// terminalSplitScene has two groups, a lone shell and a stacked pair
// whose second terminal is active, so the list shows and the pair's panes
// do; the drawer is dragged taller.
func terminalSplitScene() *ui.Scene {
	s := runningScene()
	s.Queue = nil
	s.Term = &ui.TermView{Thread: "mockup", Height: 360, Active: "term-3", Shown: 1,
		Groups: []ui.TermGroup{
			{Tabs: []ui.TermTab{{ID: "term-1", Label: "Terminal 1"}}},
			{Vertical: true, Tabs: []ui.TermTab{{ID: "term-2", Label: "Terminal 2"}, {ID: "term-3", Label: "npm run dev"}}},
		},
		Panes: []ui.TermPane{
			{ID: "term-2", Label: "Terminal 2", Screen: shellSession(131, 9)},
			{ID: "term-3", Label: "npm run dev", Screen: logStream(131, 9)},
		}}
	return s
}

// terminalEmptyScene is an open drawer with no terminals in it.
func terminalEmptyScene() *ui.Scene {
	s := runningScene()
	s.Queue = nil
	s.Term = &ui.TermView{Thread: "mockup", Height: 280}
	return s
}

// terminalCloseScene asks before closing the lone terminal.
func terminalCloseScene() *ui.Scene {
	s := terminalScene()
	s.Term.Confirm = &ui.TermConfirm{Title: `Close terminal "Terminal 1"?`, Description: "This stops the running process and clears its history."}
	return s
}
