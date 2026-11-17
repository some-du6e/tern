package t3

import "encoding/json"

// Terminals, as in docs/projects-files-vcs-terminal.md §5. A terminal is
// (threadId, terminalId), both picked by the client. Output is the raw pty
// stream, input is what a terminal sends for key presses.

// TermSnapshot is a terminal's state and replayable scrollback.
type TermSnapshot struct {
	ThreadID   string `json:"threadId"`
	TerminalID string `json:"terminalId"`
	Cwd        string `json:"cwd"`
	Status     string `json:"status"` // starting, running, exited, error
	History    string `json:"history"`
	Label      string `json:"label"`
	ExitCode   *int   `json:"exitCode"`
}

// TermEvent is one event on a terminal.attach stream.
type TermEvent struct {
	Type     string        `json:"type"` // snapshot, output, exited, restarted, cleared, closed, error, activity
	Snapshot *TermSnapshot `json:"snapshot"`
	Data     string        `json:"data"`
	ExitCode *int          `json:"exitCode"`
	Message  string        `json:"message"`
	Label    string        `json:"label"`
	Busy     bool          `json:"hasRunningSubprocess"`
}

// TermOpen is where and how big a terminal starts.
type TermOpen struct {
	ThreadID   string `json:"threadId"`
	TerminalID string `json:"terminalId"`
	Cwd        string `json:"cwd,omitempty"`
	Cols       int    `json:"cols,omitempty"`
	Rows       int    `json:"rows,omitempty"`
	// Restart a shell that exited; Resume reattaches after a drop without
	// ever restarting.
	Restart bool `json:"restartIfNotRunning,omitempty"`
	Resume  bool `json:"resume,omitempty"`
}

// TerminalAttach opens the terminal if it needs to and streams its events.
// The first is always a snapshot.
func (c *Conn) TerminalAttach(o TermOpen, onEvent func(TermEvent), onEnd func(error)) (cancel func()) {
	return c.Stream("terminal.attach", o, func(v json.RawMessage) {
		var ev TermEvent
		if json.Unmarshal(v, &ev) == nil {
			onEvent(ev)
		}
	}, onEnd)
}

// TerminalWrite sends key presses or pasted text.
func (c *Conn) TerminalWrite(threadID, terminalID, data string) error {
	return c.Call("terminal.write", map[string]any{"threadId": threadID, "terminalId": terminalID, "data": data}, nil)
}

// TerminalResize resizes the pty.
func (c *Conn) TerminalResize(threadID, terminalID string, cols, rows int) error {
	return c.Call("terminal.resize", map[string]any{"threadId": threadID, "terminalId": terminalID, "cols": cols, "rows": rows}, nil)
}

// TerminalClose ends the shell and forgets its scrollback.
func (c *Conn) TerminalClose(threadID, terminalID string) error {
	return c.Call("terminal.close", map[string]any{"threadId": threadID, "terminalId": terminalID, "deleteHistory": true}, nil)
}
