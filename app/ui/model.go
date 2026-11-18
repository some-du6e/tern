package ui

import "image"

// The view model. Every field maps to something the T3 protocol sends (see
// ../docs): shell threads, turn items, runs, runtime requests. The mockups
// fill it with fixtures; the client will fill it from the socket.

type EnvStatus int

const (
	EnvOnline EnvStatus = iota
	EnvBusy
	EnvOffline
)

type Env struct {
	ID     string
	Label  string
	Status EnvStatus
}

type ThreadState int

const (
	ThreadIdle ThreadState = iota
	ThreadRunning
	ThreadApproval
	ThreadQuestion
	ThreadError
)

// ---------- the sidebar (docs/ui/04-sidebar.md) ----------

// Sidebar is the sidebar's content below its header row: the search and
// filter row, the thread list and the footer.
type Sidebar struct {
	Query       string         // a mockup's search; live, the "search" field holds it
	HasProjects bool           // there's a project, so the filter shows
	NoNew       bool           // there's nowhere to start a thread: New thread is disabled
	Scope       []ProjectIcon  // the projects the filter narrows the list to, none for all
	ScopeLabel  string         // the filter's label: "Filter threads by project: A, B"
	ScopeKey    string         // the scope as chosen: a new one starts the list at its top
	Open        string         // the open thread's row, which the list scrolls to each time it changes
	Items       []SidebarItem  // the list, top to bottom
	Empty       *SidebarEmpty  // what the list says when nothing is active
	Results     []ThreadRow    // while searching, the matches in place of the list
	Highlight   int            // the highlighted result
	NoResults   string         // what the list says when nothing matches
	Filter      *ProjectFilter // the filter's popup, while it's open
	Snooze      *SnoozeMenu    // a card's snooze menu, while it's open
	Renaming    string         // the row whose title is being edited
	Ticking     bool           // a row shows a run timer, which moves each second
	Undo        string         // the footer's undo notice: "Settled", "Settled 2"…
	UndoClaim   bool           // mod+Z undoes it even in a text field
}

// SidebarItem is one entry in the thread list: a thread's row, the new
// thread's draft and the rule under it, a shelf's header, or the Settled
// shelf's "Show N more".
type SidebarItem struct {
	Row     *ThreadRow
	Draft   *DraftRow
	Divider bool
	Shelf   *Shelf
	More    int // "Show <More> more"
}

// DraftRow is the new thread's unsent draft (SidebarDraftBlock.tsx): its
// project and the first line written.
type DraftRow struct {
	ID      string // click target "draft:<ID>"; its discard is "discard:<ID>"
	Project string
	Icon    ProjectIcon
	Preview string // the first line written, else "2 attachments"
	Active  bool   // the draft is open
	Hover   bool   // a mockup draws it hovered
}

// Shelf is the header of the Working, Snoozed or Settled shelf.
type Shelf struct {
	ID    string // "working", "snoozed", "settled": click target "shelf:<ID>"
	Label string
	Count int
	Open  bool
}

// RowStatus is the state a card's status label shows (§3.6); StatusNone
// shows the row's age instead.
type RowStatus int

const (
	StatusNone RowStatus = iota
	StatusWorking
	StatusWaiting
	StatusApproval
	StatusInput
	StatusRequest
	StatusLimited
	StatusFailed
	StatusWoke
	StatusDone
)

// ThreadRow is a thread in the sidebar: a card for pinned, active and
// working threads, a slim row on the Snoozed and Settled shelves.
type ThreadRow struct {
	ID    string // "<environment>/<thread>": click target "thread:<ID>"
	Slim  bool   // the one-line row of the Snoozed and Settled shelves
	Shelf string // the shelf it's on, "" for pinned and active
	Title string

	Project string // the project's name
	Icon    ProjectIcon

	Status RowStatus
	Label  string // the status label: "Working", "Goal", "Approval"…
	Timer  string // after a working or waiting label: "4h 12m"
	Age    string // "now", "4m", "2h", "3d"
	Wake   string // a snoozed row's wake label: "3h"

	Recede       bool // quieter: it's working, or done and seen
	Unread       bool // its last run finished since the last visit
	Woke         bool // it woke from a snooze since the last visit
	Active       bool // the open thread
	Selected     bool // in the multi-selection
	Pinned       bool
	Draft        bool // its composer holds unsent text
	Regenerating bool // its title is being regenerated

	Branch      string
	Worktree    bool     // the branch is checked out in a worktree
	Machine     string   // the machine it's on
	MachineIcon string   // that machine's kind: "server", "laptop"…
	Remote      bool     // the row shows its machine's icon
	Scratch     string   // a "No project" thread's machine, for the meta line
	Providers   []string // the provider stack, oldest first: "Codex", "Claude"
	Provider    string   // the current provider, for the hover card's model line
	Model       string   // the hover card's model
	Handoff     string   // "Handed off from Codex"
	Terminals   int      // terminals with a command running
	Error       string   // "Error occurred", "Usage limit reached"
	Limited     bool     // the error is a usage limit
	PR          *PRBadge

	CanSettle bool // Settle (or Un-settle, Wake) is offered
	CanSnooze bool // the snooze clock is offered
	CanUnpin  bool // the pin is a button
	Hover     bool // a mockup draws it hovered
}

// ProjectIcon is a project's monogram (ProjectMonogram.tsx) in one of T3's
// 18 icon colours, or a lucide icon in that colour.
type ProjectIcon struct {
	Text   string // one or two glyphs
	Color  string // "amber", "blue"…
	Lucide string // a lucide icon instead of the monogram
}

// PRBadge is the meta line's pull request: its state's glyph and its number,
// or "+N" for several linked.
type PRBadge struct {
	State string // "open", "draft", "merged", "closed", "" while unknown
	Text  string
	URL   string
}

// SidebarEmpty is the empty list's message and its one button.
type SidebarEmpty struct {
	Text   string
	Button string // click target "filter.clear"
}

// ProjectFilter is the project filter's popup: "All projects", then each
// project. Its search field is the "filter.search" editor.
type ProjectFilter struct {
	Items []FilterItem
}

// FilterItem is one of the filter's projects; ID "" is "All projects".
type FilterItem struct {
	ID          string // click target "filter.project:<ID>"
	Label       string
	Icon        ProjectIcon
	Selected    bool
	Highlighted bool // where the arrows are, and Enter picks
}

// SnoozeMenu is the menu a card's clock opens: when to wake it.
type SnoozeMenu struct {
	Row     string     // the row it opened on
	Choices []MenuItem // click targets "snooze.pick:<ID>"
}

// ContextMenu is the menu a right-click (or the context menu key) opens on
// a row: the browser's DOM menu in T3 (contextMenuFallback.ts).
type ContextMenu struct {
	Items    []MenuItem
	Keyboard bool        // opened from the keyboard: the first item has focus
	At       image.Point // where a mockup's opens; a live one opens at the click
}

// MenuItem is one row of a menu.
type MenuItem struct {
	ID       string // click target "ctx:<ID>"
	Label    string
	Icon     string // lucide
	Detail   string // a trailing value: "6:00 PM"
	Danger   bool
	Off      bool       // shown, but can't be picked now
	Sep      bool       // a separator above it
	Radio    bool       // a radio row: a check when Checked, else a gap
	Checked  bool       //
	Children []MenuItem // a submenu
}

type ItemKind int

const (
	ItemUser ItemKind = iota
	ItemAssistant
	ItemFold     // "Worked for 2m 14s"
	ItemTool     // command_execution, file_change, …
	ItemThought  // reasoning
	ItemNotice   // system_notice, compaction
	ItemError    // error
	ItemTodo     // todo_list
	ItemThinking // the live "Thinking" row while a run has no reasoning item
)

type ToolKind int

const (
	ToolCommand ToolKind = iota
	ToolEdit
	ToolSearch
	ToolWeb
	ToolAgent
	ToolMCP
)

type Todo struct {
	Text string
	Done bool
	Now  bool
}

type Item struct {
	ID       string // click target "fold:", "tool:" or "thought:" + ID when set
	Kind     ItemKind
	Text     string // message markdown, thought text, fold label
	Tool     ToolKind
	Title    string // tool summary line: "$ go test ./..."
	Body     string // tool output, shown when Open
	Meta     string // "exit 0 · 1.8s", "+12 −3"
	Open     bool
	Live     bool
	Failed   bool
	Todos    []Todo
	Attached int
	Chips    []string // the labels of the pictures ChipText(1), ChipText(2), … in Text
	Thumbs   []Thumb  // the attached pictures, as they load
}

type Approval struct {
	Kind   string // "command", "file-change"
	Detail string
	Cwd    string
	YesNo  bool // false for requests tern can only show, like MCP elicitation
}

type Question struct {
	Header   string
	Text     string
	Options  []string
	Picked   int
	Chosen   []bool // multi-select picks; when set it wins over Picked
	Custom   string
	MultiSel bool
}

type FileChange struct {
	Path     string
	Add, Del int
	Kind     byte // 'M', 'A', 'D'
	Selected bool
}

type DiffLine struct {
	Kind     byte // ' ', '+', '-', '@'
	Old, New int
	Text     string
}

type DiffView struct {
	Title string
	File  string
	Files []FileChange
	Lines []DiffLine
}

// ModelPicker is T3's model menu. Tabs are Favorites (Provider "") and each
// ready provider; Models are the open tab's, favourites first.
type ModelPicker struct {
	Query  string
	Tabs   []PickerTab
	Tab    int
	Models []PickerModel
	Legacy int // older models, folded under "Legacy models"
}

type PickerTab struct {
	Provider string
}

type PickerModel struct {
	ID       string // click target "picker.model:<ID>" when set
	Name     string
	Note     string // "1M" and the like
	Fav      bool
	Selected bool
}

// Traits is the model options menu behind the effort button, one group per
// option descriptor the model reports.
type Traits struct {
	Groups []Trait
}

type Trait struct {
	Label   string
	Choices []TraitChoice // a select descriptor
	Sel     int
	Bool    bool // a boolean descriptor, drawn as a switch
	On      bool
	Moving  bool    // the switch is mid-change
	T       float32 // 0 to 1 through that change
}

type TraitChoice struct {
	Label, Desc string
	Danger      bool // red, for things like Delete
	Off         bool // shown but can't be picked now
}

type Suggestion struct {
	Name, Hint, Desc string
}

type SignIn struct {
	Email       string
	Step        int // 0 email, 1 code, 2 pairing URL
	Code        string
	Sending     bool
	SendingCode bool // the code step is up while Clerk sends the email
	Error       string
	Password    bool // typing a password instead of the code
	HasPassword bool
}

const (
	SignInEmail = iota
	SignInCode
	SignInPair
)

// Browser is the preview surface. An empty URL is the new-tab page.
type Browser struct {
	Title   string
	URL     string
	Control string // "You have control", "Agent has control", "Watching"
}

type RunInfo struct {
	Machine  string
	Provider string // picks the model's logo
	Status   string
	Model    string
	Elapsed  string
	Mode     string
	Branch   string
	Worktree string
}

type Scene struct {
	Envs    []Env
	Sidebar Sidebar

	Tabs   []string
	TabIDs []string // click targets "tab:<ID>" and "tabclose:<ID>"
	TabSel int

	// Live marks a scene from the real client: fields become editors and
	// rows become click targets. Mockups leave it off.
	Live          bool
	Booting       bool         // live: still reading the saved session; draw a blank window
	Error         string       // shown as a dismissable banner
	Notice        string       // a quiet status line over the composer, like "Sending…"
	Empty         string       // shown instead of messages when no thread is open
	Access        *Traits      // the access-mode menu, open over the lock
	Account       *Traits      // the account menu, open over the sidebar footer
	Projects      *Traits      // the new-thread project menu, one group per machine
	Menu          *ContextMenu // a sidebar row's context menu, at the click
	Details       bool         // the thread details popover is open
	DetailsCard   bool         // the details card is shown inline, where it fits
	SidebarHidden bool         // the sidebar is collapsed
	SidebarW      float32      // the sidebar's width in dp, 0 for T3's default
	Machines      *Traits      // the footer's machines menu, when open
	Term          *TermView    // the open thread's terminal drawer, when open
	Web           *BrowserView // the open thread's browser panel, when open
	PickProject   bool         // the header's project is a dropdown (a draft)

	Title      string
	State      ThreadState
	StateLabel string
	Project    string
	Items      []Item
	MoreAbove  bool

	Composer     string
	Hero         string // a new thread's project: "What should we build in <Hero>?" over a centered composer
	ProjectsHero bool   // the Projects menu opens under the heading's project name
	Placeholder  string
	Images       int     // mockup placeholders
	Thumbs       []Thumb // the composer's pasted images, live
	Provider     string  // "Claude", "Codex", "Gemini": picks the logo
	Model        string
	Effort       string // the options button label, values joined with " · "
	Runtime      string
	Queue        []string
	Approval     *Approval
	Questions    []Question
	Suggest      []Suggestion
	SuggestSel   int
	SuggestHead  string

	Changes []FileChange
	Run     *RunInfo
	Panel   int // which right panel tab is open
	Browser *Browser

	Picker *ModelPicker
	Traits *Traits
	Diff   *DiffView
	Login  *SignIn

	Phase float64 // spinner angle, so a frame can show motion

	Net       string
	Stats     string
	FrameTime string
}
