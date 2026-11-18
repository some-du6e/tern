package ui

import "time"

// The conversation timeline's view model, after T3's MessagesTimelineRow
// (MessagesTimeline.logic.ts:494-644; docs/ui/06-timeline.md §2.2). Core
// derives the rows from the thread's turn items with T3's rules (§5.2);
// timeline.go draws them.

// Timeline is the open thread's conversation.
type Timeline struct {
	// ID names the thread: each keeps its own scroll position and follow.
	ID   string
	Rows []TimelineRow
	// The history control over the first row: "Load earlier turns", and
	// why the last page failed.
	MoreHistory, LoadingHistory bool
	HistoryError                string
	// Loading leaves the canvas blank while the thread's snapshot loads.
	Loading bool
	// Approvals are pending approvals no row claimed; they go under the
	// rows (MessagesTimeline.tsx:755-767).
	Approvals []PendingApproval
}

// RowKind is a row's kind (§2.2).
type RowKind int

const (
	RowUser          RowKind = iota // a user's message
	RowAssistant                    // an assistant's message, with its meta row when it ends the response
	RowAssistantMeta                // that meta row, moved below the response's trailing tool groups
	RowTurnFold                     // "Worked for 2m 14s"
	RowWork                         // a standalone tool row or notice, or an open group's list
	RowWorkToggle                   // a group's header: "Ran 3 commands and read 2 files"
	RowWorkLive                     // the running tool, shining
	RowThinking                     // "Thinking", while nothing visible runs
	RowWorking                      // "Working for 12s", the live turn's header
	RowCompaction                   // "Context compacted 180K → 12K tokens"
	RowDivider                      // "Run interrupted", "Forked from conversation"
	RowPlan                         // the proposed plan card
)

// TimelineRow is one row.
type TimelineRow struct {
	// ID is stable for the row's life: its click targets and its state in
	// the list hang off it.
	ID   string
	Kind RowKind
	At   time.Time // when it happened: the timestamp shown on hover
	// Continues is T3's continuesWorkLog: another work-log row follows,
	// so this one drops its trailing space.
	Continues bool

	// RowUser, RowAssistant, RowAssistantMeta.
	Message *Message

	// RowWork: one standalone entry, or an open group's entries (Group
	// set). Label replaces a single tool row's own label.
	Entries []WorkEntry
	// Group is the work group a header, live or thinking row opens, as
	// click target "group:<Group>"; a fold's is "fold:<Group>".
	Group string
	Open  bool
	// Label is a fold's label, a group's summary, a compaction's text, a
	// divider's label, the thinking row's text.
	Label  string
	Detail string // a divider's detail, a failed compaction's reason
	Icon   string // a group's summary icon, a divider's icon
	Failed bool   // the group's latest tool failed; the compaction failed
	// Live: the live or thinking row animates; a compaction is running.
	Live bool
	// Entry is RowWorkLive's running tool; Thought the newest thought over
	// it, one line.
	Entry   *WorkEntry
	Thought string

	// RowWorking: when the run started (zero: "Working..."), and whether a
	// compaction owns the row.
	Since      time.Time
	Compacting bool

	// RowCompaction: token counts before and after, when both are known.
	Before, After string

	// RowPlan: the plan's markdown.
	Plan string
}

// Steer marks a user message that steered a running turn.
type Steer int

const (
	SteerNone     Steer = iota
	SteerSteer          // sent as a steer
	SteerPromoted       // queued, then promoted to a steer
)

// Message is a user's or an assistant's message.
type Message struct {
	ID   string
	Text string // markdown; a user's pictures are ChipText(n) in it
	// At is when a user sent it, or an assistant last updated it (the
	// meta row's time).
	At        time.Time
	Streaming bool

	// A user's message.
	Chips    []string  // the labels of the pictures ChipText(1), ChipText(2), … in Text
	Pictures []Picture // the attached pictures, as they load
	// GroupedNext and GroupedPrev: another of your messages follows or
	// precedes it, so they sit 4dp apart with their joined corners tight.
	GroupedNext, GroupedPrev bool
	Steer                    Steer
	SteerUnread              bool   // a steer the agent hasn't read yet
	Status                   string // "failed" and the like, as a pill under the bubble
	SentByT3                 bool   // "Sent by T3 Code" over it

	// An assistant's message.
	Meta  bool   // the copy and time row under it
	Copy  bool   // the copy button on that row
	Badge string // its status, when not completed: "interrupted"
}

// EntryTone picks a work entry's heading colour.
type EntryTone int

const (
	ToneTool    EntryTone = iota // a tool: --secondary-label
	ToneNotice                   // anything else: foreground at 80%
	ToneWarning                  // runtime warnings: medium, --warning
	ToneDanger                   // severe failures: medium, --destructive, circle-alert
)

// WorkEntry is one row of the work log: a tool call, a thought, a notice.
type WorkEntry struct {
	// ID is the click target "entry:<ID>" when the entry opens.
	ID     string
	At     time.Time
	Icon   string // a lucide icon, or "t3-code", "browser", "pull-request"
	Label  string
	Answer string // an answered question's answer, after the question
	Tone   EntryTone
	Failed bool // the tool failed: its icon is tinted
	// Expands: it opens to Body, or to its thought.
	Expands, Open bool
	// Thought is a reasoning entry's markdown: a one-line preview while
	// closed, the whole trace open. Thinking: still in progress.
	Thought  string
	Thinking bool
	// Error is an error item's own row (TimelineWorkEntryRow.tsx:239-305).
	Error *EntryError
	// Approval is a pending approval's row.
	Approval *PendingApproval
	Body     *EntryBody
	// Answers is an answered question's history, open.
	Answers []QuestionHistory
}

// EntryError is a failed run's error row.
type EntryError struct {
	Label   string
	Message string // the provider's message, under the label
	Warning bool   // a usage limit: in --warning, without the message
}

// QuestionHistory is one answered question.
type QuestionHistory struct {
	Question, Answer string
}

// EntryBody is what an open tool row shows (V2ItemInspector.tsx).
type EntryBody struct {
	Command  string      // a command, highlighted per shell word
	Args     [][2]string // a tool call's arguments: name and value
	ArgsText string      // arguments that aren't an object
	// Output is the tool's output; Pending while it loads, Missing when it
	// came back empty, Err when it couldn't load.
	Output  string
	Pending bool
	Missing bool
	Err     string
	Exit    int // a non-zero exit code: "exit N"
	// A file change: its path, its counts when known, its operations, and
	// a failed edit's error.
	File     string
	Add, Del int
	Counts   bool
	Changes  []string
	Failure  string
	Searches []string    // a search's patterns
	Hits     []SearchHit // a search's results
	Plain    string      // a plain block: read paths, a skill's arguments
	Value    string      // a prompt, a todo list, a notice: muted text
}

// SearchHit is one result of a file or web search.
type SearchHit struct {
	Title string // a file's path and line, or a page's title
	Text  string // the matching line, or the snippet
	Web   bool   // a page: the title in the foreground with its link mark
}

// PendingApproval is an approval waiting on the user, as the timeline
// shows it (TimelinePendingApprovalRow.tsx).
type PendingApproval struct {
	ID     string // the request: click targets "approval:<decision>"
	Kind   string // command, file-change, file-read, permission, mcp-elicitation
	Detail string
	App    string
	Reason string
	Live   bool // the provider can still take an answer
	// Options are its answers; none means T3's defaults.
	Options []ApprovalOption
}

// ApprovalOption is one answer to an approval.
type ApprovalOption struct {
	Decision, Label, Warning string
}
