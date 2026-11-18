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
	Timeline   Timeline

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
