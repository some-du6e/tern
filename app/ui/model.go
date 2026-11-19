package ui

import (
	"image"
	"time"
)

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
	JumpHints   bool           // a mockup draws the rows' jump keys, as holding mod shows them
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
	Excerpt     *Excerpt // a search result's matching message

	CanSettle bool // Settle (or Un-settle, Wake) is offered
	CanSnooze bool // the snooze clock is offered
	CanUnpin  bool // the pin is a button
	Jump      int  // the n of its jump key, mod+n (thread.jump.n), 1 to 9; 0 for none
	Hover     bool // a mockup draws it hovered
	Focus     bool // a mockup draws it with the keyboard's focus ring
}

// ProjectIcon is a project's monogram (ProjectMonogram.tsx) in one of T3's
// 18 icon colours, or a lucide icon in that colour.
type ProjectIcon struct {
	Text   string // one or two glyphs
	Color  string // "amber", "blue"…
	Lucide string // a lucide icon instead of the monogram
}

// Excerpt is the message a search result was found by, under its title
// (ThreadSearchMatch.tsx): who wrote it, the servers' snippet around the
// match, and the query's words in it, as rune ranges.
type Excerpt struct {
	User    bool // "You:", else "Agent:"
	Snippet string
	Marks   [][2]int
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

// Question is one of an agent's questions (ComposerPendingUserInputPanel.tsx).
type Question struct {
	Header   string
	Text     string
	Options  []string
	Descs    []string // each option's description, "" for none
	Chosen   []bool   // which options are picked
	MultiSel bool
	NoCustom bool // only an option answers it, nothing typed
	// Typed: an answer was typed for it while it was on show.
	Typed bool
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
	Default     bool // the option's default, which the picker's column names
}

// Permissions is the open permissions menu (ComposerPermissionsSection):
// the modes the provider supports, in T3's order, and the thread's.
type Permissions struct {
	Modes []RuntimeMode
	Sel   int
}

// StopPhase is how far a click on Stop has got (ComposerPrimaryActions.tsx).
type StopPhase int

const (
	StopIdle     StopPhase = iota
	StopPending            // clicked: "Stopping…", off
	StopLanded             // the run can't be stopped any more: "Stopped", off
	StopTimedOut           // 10s and it hasn't landed: on again
)

// QueuedMessage is a message queued behind the run (QueuedRunsControl.tsx).
type QueuedMessage struct {
	Text    string
	Editing bool // it's in the composer, being edited
}

// Stashed is a prompt in the composer's stash (ComposerStashMenu.tsx).
type Stashed struct {
	Snippet string  // its text, cut at 90; "(2 images)" without any
	Thumbs  []Thumb // its pictures
	Dropped int     // pictures too big to keep
	Age     string  // "just now", "4m ago"
}

// Suggestion is a row of the composer's @, / and $ menu (ComposerCommandMenu.tsx).
type Suggestion struct {
	Icon   string // the lucide icon before it: "file", "folder"; "" for none
	Prefix string // muted text before the label, like "/skill:"
	Label  string
	Desc   string
	Badge  string // a skill's source: "Personal", "Project", "System"
	// BadgeIcon is the badge's lucide icon: blocks, folder, user-round,
	// settings or package.
	BadgeIcon string
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
	Editor       string // the preferred editor's id, "" for none
	Label        string // "Open in VS Code", "Open in editor"
	Disabled     bool   // nothing to open it with
	MenuDisabled bool
}

// Scripts is the project's scripts (ProjectScriptsControl.tsx): a split
// that runs the primary one, "Actions" when none is, or a row to add one.
type Scripts struct {
	Primary string // the script the split runs, "" for none
	Icon    string // its icon: play, test, lint, configure, build or debug
	Any     bool   // the project has scripts
	CanAdd  bool   // "Add project script" opens the editor for a new one
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
	Access        *Permissions // the permissions menu, open over its chip
	Account       *Traits      // the account menu, open over the sidebar footer
	Projects      *Traits      // the new-thread project menu, one group per machine
	Menu          *ContextMenu // a thread's context menu, at the click or under the header's title
	Details       bool         // the thread details popover is open
	DetailsCard   bool         // the details card is shown inline, where it fits
	SidebarHidden bool         // the sidebar is collapsed
	SidebarW      float32      // the sidebar's width in dp, 0 for T3's default
	Machines      *Traits      // the footer's machines menu, when open
	Term          *TermView    // the open thread's terminal drawer, when open
	Web           *BrowserView // the open thread's browser panel, when open

	Title    string
	State    ThreadState
	Header   *ChatHeader    // the chat header, when a thread or draft is open
	Card     *ThreadDetails // and its details card
	Confirm  *Confirm       // a Git action asking first
	Alert    *Alert         // a pull request action asking first
	Timeline Timeline

	// The composer (docs/ui/07-composer.md). Live, the "composer" editor
	// holds the prompt; mockups put it in Composer.
	Composer    string
	Hero        string // a new thread's project: "What should we build in <Hero>?" over a centered composer
	Scratch     bool   // the draft has no project: "What should we work on?"
	CanScratch  bool   // the hero offers "or start without a project"
	Placeholder string
	Suggestion  string          // the provider's guess at the next prompt, on offer: the placeholder, which Tab takes
	Thumbs      []Thumb         // the composer's pictures, as tiles over the prompt
	Provider    string          // "Claude", "Codex", "Gemini": picks the logo
	Model       string          // the model's short name, on the trigger
	Effort      string          // the trigger's effort word, "" for none
	Speed       string          // the trigger's bolt: "Fast", or "Ultrafast" for its echoed one
	Ultracode   bool            // the trigger's workflow icon, for Ultracode
	NoProvider  bool            // no provider can take the message: the trigger says so instead
	Window      string          // the trigger's context window, like "1M"
	Runtime     string          // the runtime mode: "full-access", "approval-required", …
	RuntimeChip bool            // the footer shows the permissions chip: the mode isn't the default
	Sending     bool            // a send is on its way: the send button spins
	Fresh       bool            // /clear is armed: the next message starts a fresh session
	Stop        StopPhase       // where a click on Stop has got to
	Queue       []QueuedMessage // the messages waiting behind the run, at the foot of the timeline
	// EditingQueued: the composer holds a queued message, and a send updates it.
	EditingQueued bool
	Questions     []Question
	Question      int  // the question on show
	Answering     bool // the answers are on their way
	Dismissible   bool // the questions can be dismissed unanswered
	Suggest       []Suggestion
	SuggestSel    int
	SuggestEmpty  string    // the menu's line while it has no items: still searching, or nothing matched
	Stash         []Stashed // the stashed prompts, newest first
	StashOpen     bool      // the stash's drawer is open
	StashSel      int       // its highlighted row
	Stashed       time.Time // when a prompt was last stashed: the tab pulses for 1.4s

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
