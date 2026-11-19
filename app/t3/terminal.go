package t3

import "encoding/json"

// Terminals, as in docs/projects-files-vcs-terminal.md §5. A terminal is
// (threadId, terminalId), both picked by the client. Output is the raw pty
// stream, input is what a terminal sends for key presses.

// TermSnapshot is a terminal's state and replayable scrollback.
type TermSnapshot struct {
	ThreadID     string `json:"threadId"`
	TerminalID   string `json:"terminalId"`
	Cwd          string `json:"cwd"`
	WorktreePath string `json:"worktreePath"`
	Status       string `json:"status"` // starting, running, exited, error
	History      string `json:"history"`
	Label        string `json:"label"`
	ExitCode     *int   `json:"exitCode"`
	UpdatedAt    string `json:"updatedAt"`
	// The PTY grid every viewer draws; 0 from servers that predate it.
	Cols int `json:"cols"`
	Rows int `json:"rows"`
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
	// A change to the shared grid rides on activity.
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// TermSummary is one terminal in subscribeTerminalMetadata's list.
type TermSummary struct {
	ThreadID     string `json:"threadId"`
	TerminalID   string `json:"terminalId"`
	Cwd          string `json:"cwd"`
	WorktreePath string `json:"worktreePath"`
	Status       string `json:"status"`
	Busy         bool   `json:"hasRunningSubprocess"`
	Label        string `json:"label"`
	UpdatedAt    string `json:"updatedAt"`
}

// TermMetaEvent is one event on subscribeTerminalMetadata: the whole list,
// one terminal added or changed, or one removed.
type TermMetaEvent struct {
	Type       string        `json:"type"` // snapshot, upsert, remove
	Terminals  []TermSummary `json:"terminals"`
	Terminal   *TermSummary  `json:"terminal"`
	ThreadID   string        `json:"threadId"`
	TerminalID string        `json:"terminalId"`
}

// TermOpen is where and how big a terminal starts.
type TermOpen struct {
	ThreadID     string            `json:"threadId"`
	TerminalID   string            `json:"terminalId"`
	Cwd          string            `json:"cwd,omitempty"`
	WorktreePath string            `json:"worktreePath,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Cols         int               `json:"cols,omitempty"`
	Rows         int               `json:"rows,omitempty"`
	// Resume reattaches after a drop: it never reopens a terminal that
	// was closed meanwhile.
	Resume bool `json:"resume,omitempty"`
}

// TerminalOpen starts the terminal's shell if it isn't running.
func (c *Conn) TerminalOpen(o TermOpen) error {
	return c.Call("terminal.open", o, nil)
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

// TerminalMetadata streams every terminal on the environment, without
// their output.
func (c *Conn) TerminalMetadata(onEvent func(TermMetaEvent), onEnd func(error)) (cancel func()) {
	return c.Stream("subscribeTerminalMetadata", map[string]any{}, func(v json.RawMessage) {
		var ev TermMetaEvent
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
