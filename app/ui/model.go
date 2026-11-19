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

// ThreadRow is one line of the inbox. The sidebar mixes every project and
// machine into one list, so each row carries its own project and machine.
type ThreadRow struct {
	ID       string // click target "thread:<ID>" when set
	Title    string
	Project  string
	Branch   string
	Env      string // machine label for threads on another environment, "" for this one
	State    ThreadState
	Unread   bool // a run finished since the last visit: the "Done" chip
	Selected bool
	Age      string // "4m", "2h", "3d"
	Timer    string // run duration for the Working chip
	Note     string // what the thread waits on, or its error
	Wakes    string // snoozed until, "18:00"
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

// ContextMenu is the menu a thread's title, a right-click on the header or
// on a sidebar row opens: the browser's DOM menu in T3
// (contextMenuFallback.ts).
type ContextMenu struct {
	Items    []MenuItem
	Keyboard bool        // opened from the keyboard: the first item has focus
	At       image.Point // where a mockup's opens; a live one opens at the click
}

// MenuItem is one row of a menu.
type MenuItem struct {
	ID       string // click target "ctx:<ID>"
	Label    string
	Icon     string // the menu's own icon (menuIcons)
	Detail   string // a trailing value: "6:00 PM"
	Danger   bool
	Off      bool       // shown, but can't be picked now
	Sep      bool       // a separator above it
	Radio    bool       // a radio row: a check when Checked, else a gap
	Checked  bool       //
	Children []MenuItem // a submenu
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

// ChatHeader is the breadcrumb in the chat column's title row
// (ChatHeader.tsx): the project, a subagent's parent, then the thread's
// title (Scene.Title).
type ChatHeader struct {
	Project   string      // the project crumb, "No project" for the scratch project; "" for none
	Icon      ProjectIcon // the crumb's mark
	Parent    string      // a subagent's parent thread, crumbed before the title
	Server    bool        // a thread on the server: its title opens the thread's menu; a draft's is plain text
	Origin    string      // the linked environment whose agent started it: "From <Origin>"
	CanRename bool        // a double-click on the title renames it
	Renaming  bool        // the title is the "header.title" editor
	Guest     *GuestLine  // the thread's guest link, after the breadcrumb
}

// GuestLine is the header's guest link indicator (GuestShareIndicator):
// who the thread is shared with, and "Stop sharing" ("header.unshare").
type GuestLine struct {
	Status  string // "<name> is a guest", "Guest link active" or "Guest link expired"
	Closing bool   // the close is on its way: Stop sharing is off
}

// ProjectIcon is a project's mark (ProjectFavicon.tsx): a monogram on a
// tile, or a lucide icon, in one of T3's project colours.
type ProjectIcon struct {
	Monogram string // one or two letters
	Lucide   string // the icon, when the project chose one
	Color    string // gray, red, orange, … rose (projectIconColors.ts)
}

// ThreadDetails is the thread details card (ThreadDetailsPanel.tsx): the
// thread's workspace, then its version control. Rows it doesn't offer are
// nil.
type ThreadDetails struct {
	Run     *RunContext
	OpenIn  *OpenIn
	Scripts *Scripts
	Branch  *BranchRow
	PRs     *PullRequests // under the branch row, or in its place without one
	Git     *GitActions
	InitGit bool // "Initialize Git", for a folder that isn't a repository
	Initing bool // vcs.init is running: "Initializing...", and it can't be pressed again
	Changes *Changes

	Menu   *CardMenu     // the menu open off a row
	Picker *BranchPicker // the branch picker, when open
}

// RunContext is the machine and workspace row (BranchToolbar.tsx).
type RunContext struct {
	Machine      string // the machine's kind for its icon: server, cloud, linux, desktop, laptop, mac-mini, mac-studio; "" leads with the workspace icon
	Label        string
	Tip          string // the machine icon's tooltip
	Workspace    string // its icon: folder, folder-git or folder-git-2
	WorkspaceTip string // the workspace icon's tooltip, its path
	Locked       bool   // a started thread: a plain row without a menu
}

// OpenIn is the Open in editor split (OpenInPicker.tsx).
type OpenIn struct {
	Editor   string // the preferred editor's id, "" for none
	Label    string // "Open in VS Code", "Open in editor"
	Disabled bool   // nothing to open it with
}

// Scripts is the project's scripts (ProjectScriptsControl.tsx): a split
// that runs the primary one, "Actions" when none is, or a row to add one.
type Scripts struct {
	Primary string // the script the split runs, "" for none
	Icon    string // its icon: play, test, lint, configure, build or debug
	Any     bool   // the project has scripts
}

// BranchRow is the branch picker's trigger (BranchToolbarBranchSelector.tsx).
type BranchRow struct {
	Label    string // the branch, "From <branch>" for a new worktree's base, "Select ref"
	Disabled bool   // until the refs load, and while a branch action runs
}

// PullRequests are the thread's pull request rows (ThreadDetailsPrRows.tsx):
// the one it shows, then "Show N more" folding its other linked ones,
// which Rows holds too while Expanded. Row i's target is "card.pr:<i>",
// its eye "card.prwatch:<i>", the fold "card.prs".
type PullRequests struct {
	Rows     []PullRequestRow
	Others   int
	Expanded bool
}

// PullRequestRow is one pull request: its state's icon in its tone and
// "#N: <title>". With Detail it's the detailed form instead.
type PullRequestRow struct {
	State    string // open, draft, closed or merged; "" until it's read
	Label    string
	Tip      string // "PR #12 - Open: <title>"
	Watching bool   // the agent watches it: an eye after the row stops that
	Detail   *PullRequestDetailRow
}

// PullRequestDetailRow is a pull request whose detail has loaded with
// something to say under its title (ThreadDetailsPrRow.tsx): the title on
// up to two lines, then a meta line of "#N", its checks, the eye while the
// agent watches it and the action it's ready for ("card.prpill:<i>").
type PullRequestDetailRow struct {
	Title  string
	Number int
	Checks *ChecksMeta
	Pill   *PullRequestPill
}

// ChecksMeta is a pull request's checks in its meta line, a trigger for
// their popover ("card.checks:<i>", PullRequestChecksPopover.tsx): the
// rollup's icon and tone, and a count when there's one to give.
type ChecksMeta struct {
	State string // passing, failing or pending
	Count string // "3/5 checks", "2 failed" or ""
	Popup *ChecksPopup
}

// ChecksPopup is the open checks popover: what the checks come to, then
// each check with its status, and whether a "Show all" ("card.checksall")
// unfolds the finished ones.
type ChecksPopup struct {
	Heading string // "Some checks were not successful"
	Summary string // "2 of 5 failing"
	Checks  []CheckRow
	More    string // "Show all", "Show less" or ""
}

// CheckRow is a check in the popover: its status's icon and label, and a
// "Details" link to its page ("card.check:<i>") when it has one.
type CheckRow struct {
	Status, Name, Label string
	Details             bool
}

// PullRequestPill is the action a pull request is ready for at the end of
// its meta line, in the primary tone: Merge or Ready.
type PullRequestPill struct {
	Label    string // "Merge", "Merging...", "Ready", "Marking..."
	Tip      string
	Disabled bool // an action is on its way
}

// Alert is T3's AlertDialog asking before an action: its question, then
// Cancel ("alert:cancel") and the action ("alert:ok").
type Alert struct {
	Title, Description, Action string
}

// GitActions is the quick action split (GitActionsControl.tsx).
type GitActions struct {
	Label        string // the quick action
	Icon         string // a lucide icon, or "provider:<kind>" for the host's mark
	Hint         string // why it can't run now: it explains instead of acting
	MenuDisabled bool
	Running      *GitProgress // an action is running
	Success      *GitSuccess  // one just finished
}

// GitProgress is a running Git action: its phase, its last output line and
// how long it has run.
type GitProgress struct {
	Status, Output, Elapsed string
}

// GitSuccess is the line a finished action leaves for ten seconds.
type GitSuccess struct {
	Title, Description string
}

// Changes is the change totals row.
type Changes struct {
	Insertions, Deletions int
	Base                  string // "Uncommitted", "vs <base>"; "" until the status loads
	Disabled              bool   // there's no diff to open here
}

// CardMenu is a T3 menu open off one of the card's rows: "run", "openin",
// "scripts" or "git"; or the card's own "items" menu, from "Choose what to
// show" or at the pointer after a right-click.
type CardMenu struct {
	For       string
	Rows      []MenuRow
	AtPointer bool
	NoHover   bool // the pointer doesn't light its rows (highlightItemOnHover off)
}

// MenuRowKind is what a MenuRow draws (menu.tsx).
type MenuRowKind int

const (
	MenuItemRow MenuRowKind = iota
	MenuRadioRow
	MenuCheckRow // a checkbox row: the check, when Checked, before the label
	MenuGroupLabel
	MenuSeparator
	MenuNote // a 12dp line of explanation, in Tone
)

// MenuRow is one row of a T3 menu. Its click target is "<menu>:<index>".
type MenuRow struct {
	Kind     MenuRowKind
	Label    string
	Icon     string // a lucide icon, 16dp at 80% before the label (12dp in radio rows); "provider:<kind>" or "editor:<id>" for a mark
	Detail   string // a radio row's second line in 12px muted: the previous worktree's branch
	Shortcut string
	Checked  bool
	Disabled bool
	Reason   string // why a disabled row is off, in a popover to its left on hover
	Tone     string // a note's colour: "warning" or "destructive"
}

// BranchPicker is the branch combobox (BranchPicker.tsx): a search field
// over the refs it lists ("branch.search"). Near the end of a list with
// More it asks for the next page ("branch.more").
type BranchPicker struct {
	Refs      []BranchRef
	Create    string // "Create new ref "<name>"" when no ref matches exactly
	Notice    string // why some refs are off: "Stop the current turn to switch worktrees."
	Status    string // "Loading refs...", "Showing 50 of 120 refs"
	Highlight int
	More      bool // the server has more refs for the search
	Loading   bool // the search's first page hasn't come yet
}

// BranchRef is one ref in the picker, with its first tag: current,
// worktree, remote or default.
type BranchRef struct {
	Name, Tag string
	Disabled  bool
}

// Confirm is T3's alert dialog for a Git action on the default branch
// (GitActionsControl.tsx): its question, then Abort, Continue and a
// feature branch. Its targets are "confirm:abort", "confirm:continue" and
// "confirm:branch".
type Confirm struct {
	Title, Description string
	Continue           string // the continue button's label
	BranchOff          string // why the feature branch button is disabled, "" when it isn't
}

type Scene struct {
	Envs        []Env
	Search      string
	Filter      string // project filter, "All projects"
	Pinned      []ThreadRow
	Active      []ThreadRow
	Snoozed     []ThreadRow
	Settled     int         // folded, loaded N at a time
	SettledRows []ThreadRow // live: the settled threads, once unfolded
	SidebarNote string      // live: why the inbox is empty ("Connecting…")

	Tabs   []string
	TabIDs []string // click targets "tab:<ID>" and "tabclose:<ID>"
	TabSel int

	// Live marks a scene from the real client: fields become editors and
	// rows become click targets. Mockups leave it off.
	Live                     bool
	Booting                  bool         // live: still reading the saved session; draw a blank window
	Error                    string       // shown as a dismissable banner
	Notice                   string       // a quiet status line over the composer, like "Sending…"
	Empty                    string       // shown instead of messages when no thread is open
	Access                   *Traits      // the access-mode menu, open over the lock
	Account                  *Traits      // the account menu, open over the sidebar footer
	Projects                 *Traits      // the new-thread project menu, one group per machine
	Menu                     *ContextMenu // a thread's actions menu, at the click or under the header's title
	Details                  bool         // the thread details popover is open
	DetailsCard              bool         // the details card is shown inline, where it fits
	SidebarHidden            bool         // the sidebar is collapsed
	SidebarW                 float32      // the sidebar's width in dp, 0 for T3's default
	Machines                 *Traits      // the footer's machines menu, when open
	Term                     *TermView    // the open thread's terminal drawer, when open
	Web                      *BrowserView // the open thread's browser panel, when open
	ShowSnoozed, ShowSettled bool

	Title     string
	State     ThreadState
	Header    *ChatHeader    // the chat header, when a thread or draft is open
	Card      *ThreadDetails // and its details card
	Confirm   *Confirm       // a Git action asking first
	Alert     *Alert         // a pull request action asking first
	Items     []Item
	MoreAbove bool

	Composer    string
	Hero        string // a new thread's project: "What should we build in <Hero>?" over a centered composer
	Placeholder string
	Images      int     // mockup placeholders
	Thumbs      []Thumb // the composer's pasted images, live
	Provider    string  // "Claude", "Codex", "Gemini": picks the logo
	Model       string
	Effort      string // the options button label, values joined with " · "
	Runtime     string
	Queue       []string
	Approval    *Approval
	Questions   []Question
	Suggest     []Suggestion
	SuggestSel  int
	SuggestHead string

	Picker *ModelPicker
	Traits *Traits
	Diff   *DiffView
	Login  *SignIn

	Phase float64 // spinner angle, so a frame can show motion

	Net       string
	Stats     string
	FrameTime string
}
