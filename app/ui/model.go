package ui

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

// Question is one of an agent's questions (ComposerPendingUserInputPanel.tsx).
type Question struct {
	Header   string
	Text     string
	Options  []string
	Descs    []string // each option's description, "" for none
	Chosen   []bool   // which options are picked
	MultiSel bool
	NoCustom bool // only an option answers it, nothing typed
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

// Stashed is a prompt in the composer's stash (ComposerStashMenu.tsx).
type Stashed struct {
	Snippet string  // its text, cut at 90; "(2 images)" without any
	Thumbs  []Thumb // its pictures
	Dropped int     // pictures too big to keep
	Age     string  // "4m", "2h"
}

// Suggestion is a row of the composer's @, / and $ menu (ComposerCommandMenu.tsx).
type Suggestion struct {
	Icon   string // the lucide icon before it: "file", "folder"; "" for none
	Prefix string // muted text before the label, like "/skill:"
	Label  string
	Desc   string
	Badge  string // a skill's source: "Personal", "Project", "System"
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
	Context                  *Traits      // a thread's right-click menu, at the click
	Details                  bool         // the thread details popover is open
	DetailsCard              bool         // the details card is shown inline, where it fits
	SidebarHidden            bool         // the sidebar is collapsed
	SidebarW                 float32      // the sidebar's width in dp, 0 for T3's default
	Machines                 *Traits      // the footer's machines menu, when open
	Term                     *TermView    // the open thread's terminal drawer, when open
	Web                      *BrowserView // the open thread's browser panel, when open
	PickProject              bool         // the header's project is a dropdown (a draft)
	ShowSnoozed, ShowSettled bool

	Title      string
	State      ThreadState
	StateLabel string
	Project    string
	Items      []Item
	MoreAbove  bool

	// The composer (docs/ui/07-composer.md). Live, the "composer" editor
	// holds the prompt; mockups put it in Composer.
	Composer     string
	Hero         string // a new thread's project: "What should we build in <Hero>?" over a centered composer
	Scratch      bool   // the draft has no project: "What should we work on?"
	CanScratch   bool   // the hero offers "or start without a project"
	ProjectsHero bool   // the Projects menu opens under the heading's project name
	Placeholder  string
	Thumbs       []Thumb // the composer's pictures, as tiles over the prompt
	Provider     string  // "Claude", "Codex", "Gemini": picks the logo
	Model        string  // the model's short name, on the trigger
	Effort       string  // the trigger's effort word, "" for none
	Fast         bool    // the trigger's bolt, for Fast mode
	Window       string  // the trigger's context window, like "1M"
	Runtime      string  // the runtime mode: "full-access", "approval-required", …
	RuntimeChip  bool    // the footer shows the permissions chip: the mode isn't the default
	Sending      bool    // a send is on its way: the send button spins
	Queue        []string
	Approval     *Approval
	Questions    []Question
	Question     int  // the question on show
	Answering    bool // the answers are on their way
	Dismissible  bool // the questions can be dismissed unanswered
	Suggest      []Suggestion
	SuggestSel   int
	SuggestEmpty string    // the menu's line while it has no items: still searching, or nothing matched
	Stash        []Stashed // the stashed prompts, newest first
	StashOpen    bool      // the stash's drawer is open
	StashSel     int       // its highlighted row

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
