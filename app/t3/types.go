package t3

import "encoding/json"

// The shapes tern reads from the shell and thread streams. They're partial:
// unknown fields are ignored, and turn items keep their raw JSON too.

type ModelSelection struct {
	InstanceID string         `json:"instanceId"`
	Model      string         `json:"model"`
	Options    []OptionChoice `json:"options,omitempty"`
}

type OptionChoice struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
}

type Project struct {
	ID                    string          `json:"id"`
	Title                 string          `json:"title"`
	WorkspaceRoot         string          `json:"workspaceRoot"`
	DefaultModelSelection *ModelSelection `json:"defaultModelSelection"`
	ProjectIcon           *ProjectIcon    `json:"projectIcon"`
	Scripts               []ProjectScript `json:"scripts"`
}

// ProjectIcon is a project's chosen mark (project.ts): a lucide icon, an
// emoji or a monogram, in one of T3's project colours. Older lucide icons
// carry a monogram in MonogramText or Monogram.
type ProjectIcon struct {
	Kind         string `json:"kind"` // lucide, emoji, monogram
	Name         string `json:"name"`
	Emoji        string `json:"emoji"`
	Text         string `json:"text"`
	Color        string `json:"color"`
	MonogramText string `json:"monogramText"`
	Monogram     string `json:"monogram"`
}

// ProjectScript is one of a project's actions, run in a terminal.
type ProjectScript struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Command             string `json:"command"`
	Icon                string `json:"icon"` // play, test, lint, configure, build, debug
	RunOnWorktreeCreate bool   `json:"runOnWorktreeCreate"`
	RunOnSettle         bool   `json:"runOnSettle"`
	StopOnSettle        bool   `json:"stopOnSettle"`
}

type RuntimeRequest struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type Lineage struct {
	ParentThreadID       string `json:"parentThreadId"`
	RelationshipToParent string `json:"relationshipToParent"`
}

type Thread struct {
	ID                    string             `json:"id"`
	ProjectID             string             `json:"projectId"`
	Title                 string             `json:"title"`
	CreatedAt             string             `json:"createdAt"`
	UpdatedAt             string             `json:"updatedAt"`
	LatestUserMessageAt   string             `json:"latestUserMessageAt"`
	LatestRunCompletedAt  string             `json:"latestRunCompletedAt"`
	LastVisitedAt         string             `json:"lastVisitedAt"`
	PinnedAt              string             `json:"pinnedAt"`
	PinOrderKey           string             `json:"pinOrderKey"`
	ActiveOrderKey        string             `json:"activeOrderKey"`
	SettledAt             string             `json:"settledAt"`
	SnoozedUntil          string             `json:"snoozedUntil"`
	ArchivedAt            string             `json:"archivedAt"`
	DeletedAt             string             `json:"deletedAt"`
	LastError             string             `json:"lastError"`
	ActiveRunID           string             `json:"activeRunId"`
	ActivityRunStatus     string             `json:"activityRunStatus"`
	ActivityRunStartedAt  string             `json:"activityRunStartedAt"`
	PendingRuntimeRequest *RuntimeRequest    `json:"pendingRuntimeRequest"`
	Lineage               *Lineage           `json:"lineage"`
	ModelSelection        *ModelSelection    `json:"modelSelection"`
	RuntimeMode           string             `json:"runtimeMode"`
	InteractionMode       string             `json:"interactionMode"`
	Branch                string             `json:"branch"`
	WorktreePath          string             `json:"worktreePath"`
	LatestRunID           string             `json:"latestRunId"`
	LinkOrigin            *LinkOrigin        `json:"linkOrigin"`
	PullRequests          []PullRequestLink  `json:"pullRequests"`
	LinkedPullRequest     *LinkedPullRequest `json:"linkedPullRequest"`
	BranchPullRequest     *LinkedPullRequest `json:"branchPullRequest"`
	EnvironmentID         string             `json:"-"`
}

// LinkOrigin is the linked environment whose agent started a thread.
type LinkOrigin struct {
	Label string `json:"label"`
}

// GuestShare is a thread's guest link (guestShare.ts): when it expires,
// and the name of the guest who redeemed it, if one has.
type GuestShare struct {
	ExpiresAt string `json:"expiresAt"`
	GuestName string `json:"guestName"`
}

// AppThread is the part of a thread's full record (OrchestrationV2AppThread)
// that its stream's projection and thread events carry beyond the shell.
type AppThread struct {
	GuestShare *GuestShare `json:"guestShare"`
}

// PullRequestLink is a pull request linked to a thread, with the server's
// last snapshot of it (ThreadPullRequestLink, threadPullRequest.ts:107).
// Host, Repository and Number are its key.
type PullRequestLink struct {
	Host       string               `json:"host"`
	Repository string               `json:"repository"`
	Number     int                  `json:"number"`
	URL        string               `json:"url"`
	Source     string               `json:"source"` // manual, created, agent, stack, stack-dismissed
	LinkedAt   string               `json:"linkedAt"`
	Snapshot   *PullRequestSnapshot `json:"snapshot"`
	Stack      *PullRequestStack    `json:"stack"`
	Watch      *PullRequestWatch    `json:"watch"` // set while the agent watches its checks and comments
}

// PullRequestWatch is the agent watching a linked pull request: it wakes
// when checks finish, someone comments or the branch conflicts.
type PullRequestWatch struct {
	StartedAt string `json:"startedAt"`
}

// PullRequestSnapshot is what the server last read of a linked pull request.
type PullRequestSnapshot struct {
	State      string `json:"state"` // open, closed, merged
	Title      string `json:"title"`
	HeadBranch string `json:"headBranch"`
	BaseBranch string `json:"baseBranch"`
	IsDraft    bool   `json:"isDraft"`
	UpdatedAt  string `json:"updatedAt"`
}

// PullRequestStack is the host's own stack a linked pull request is in,
// its layers bottom to top.
type PullRequestStack struct {
	ID     string                  `json:"id"`
	Layers []PullRequestStackLayer `json:"layers"`
}

// PullRequestStackLayer is one pull request of a stack.
type PullRequestStackLayer struct {
	Number int `json:"number"`
}

// LinkedPullRequest is the single pull request older servers link a
// thread to, and the open one the server finds for its branch.
type LinkedPullRequest struct {
	ProjectID  string `json:"projectId"`
	Repository string `json:"repository"`
	Number     int    `json:"number"`
	URL        string `json:"url"`
}

// ShellItem is one value from orchestration.subscribeShell.
type ShellItem struct {
	Kind     string `json:"kind"`
	Snapshot *struct {
		Projects []Project `json:"projects"`
		Threads  []Thread  `json:"threads"`
	} `json:"snapshot"`
	ResolvedRoots json.RawMessage `json:"resolvedRepositoryIdentityRoots"`
	Project       *Project        `json:"project"`
	ProjectID     string          `json:"projectId"`
	Thread        *Thread         `json:"thread"`
	ThreadID      string          `json:"threadId"`
	Location      string          `json:"location"`
	Sequence      int64           `json:"sequence"`
}

// TurnItem is one row of a thread, in ordinal order. Fields cover what tern
// draws; Raw keeps the rest.
type TurnItem struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Ordinal     float64 `json:"ordinal"`
	RunID       string  `json:"runId"`
	MessageID   string  `json:"messageId"`
	Status      string  `json:"status"`
	Streaming   bool    `json:"streaming"`
	Text        string  `json:"text"`
	Markdown    string  `json:"markdown"`
	Title       string  `json:"title"`
	Message     string  `json:"message"`
	StartedAt   string  `json:"startedAt"`
	CompletedAt string  `json:"completedAt"`
	UpdatedAt   string  `json:"updatedAt"`
	RequestID   string  `json:"requestId"`
	PlanID      string  `json:"planId"`

	// tools
	Input                  json.RawMessage `json:"input"`
	Output                 json.RawMessage `json:"output"`
	ExitCode               *int            `json:"exitCode"`
	FileName               string          `json:"fileName"`
	Additions              *int            `json:"additions"`
	Deletions              *int            `json:"deletions"`
	DiffStr                string          `json:"diffStr"`
	Pattern                string          `json:"pattern"`
	Patterns               []string        `json:"patterns"`
	ToolName               string          `json:"toolName"`
	Prompt                 string          `json:"prompt"`
	Result                 string          `json:"result"`
	OutputIndicatesFailure bool            `json:"outputIndicatesFailure"`
	Reason                 string          `json:"reason"`

	Steps []struct {
		Step   string `json:"step"`
		Text   string `json:"text"`
		Status string `json:"status"`
	} `json:"steps"`
	Failure *struct {
		Message string `json:"message"`
	} `json:"failure"`
	Questions   []Question        `json:"questions"`
	Attachments []json.RawMessage `json:"attachments"`
}

type Question struct {
	ID                string `json:"id"`
	Header            string `json:"header"`
	Question          string `json:"question"`
	MultiSelect       bool   `json:"multiSelect"`
	AllowCustomAnswer *bool  `json:"allowCustomAnswer"`
	Options           []struct {
		Label       string `json:"label"`
		Value       string `json:"value"`
		Description string `json:"description"`
	} `json:"options"`
}

type Run struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	StartedAt     string `json:"startedAt"`
	CompletedAt   string `json:"completedAt"`
	QueuePosition int    `json:"queuePosition"`
	QueueHeld     bool   `json:"queueHeld"`
	UserMessageID string `json:"userMessageId"`
}

type Message struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Text string `json:"text"`
}

// ThreadItem is one value from orchestration.subscribeThread.
type ThreadItem struct {
	Kind       string `json:"kind"`
	Projection *struct {
		Thread           *AppThread `json:"thread"`
		VisibleTurnItems []struct {
			Item json.RawMessage `json:"item"`
		} `json:"visibleTurnItems"`
		Runs     []Run     `json:"runs"`
		Messages []Message `json:"messages"`
	} `json:"projection"`
	HistoryCursor  string `json:"historyCursor"`
	HasMoreHistory bool   `json:"hasMoreHistory"`
	Event          *struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	} `json:"event"`
	Sequence int64 `json:"sequence"`
}

type HistoryPage struct {
	Items []struct {
		Item json.RawMessage `json:"item"`
	} `json:"items"`
	NextCursor     string `json:"nextCursor"`
	HasMoreHistory bool   `json:"hasMoreHistory"`
}

type Attachment struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	MimeType  string `json:"mimeType"`
	SizeBytes int    `json:"sizeBytes"`
}

// ServerConfig is the part of server.getConfig the composer and the thread
// details card need.
type ServerConfig struct {
	Providers            []Provider         `json:"providers"`
	AvailableEditors     []string           `json:"availableEditors"`
	RemoteOpenTargets    []RemoteOpenTarget `json:"remoteOpenTargets"`
	ScratchWorkspaceRoot string             `json:"scratchWorkspaceRoot"`
	Settings             struct {
		EnvironmentIcon       string          `json:"environmentIcon"`
		DefaultModelSelection *ModelSelection `json:"defaultModelSelection"`
		DefaultRuntimeMode    string          `json:"defaultRuntimeMode"`
		Favorites             []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"favorites"`
		ProviderModelPreferences map[string]struct {
			HiddenModels []string `json:"hiddenModels"`
			ModelOrder   []string `json:"modelOrder"`
		} `json:"providerModelPreferences"`
	} `json:"settings"`
	Environment struct {
		Platform struct {
			Machine string `json:"machine"`
		} `json:"platform"`
		Capabilities struct {
			ServerResolvedCommandContext bool `json:"serverResolvedCommandContext"`
			InlineMessageContext         bool `json:"inlineMessageContext"`
			ThreadStop                   bool `json:"threadStop"`
		} `json:"capabilities"`
	} `json:"environment"`
}

type Provider struct {
	InstanceID  string  `json:"instanceId"`
	Driver      string  `json:"driver"`
	DisplayName string  `json:"displayName"`
	Enabled     bool    `json:"enabled"`
	Status      string  `json:"status"`
	Models      []Model `json:"models"`
}

type Model struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	IsCustom     bool   `json:"isCustom"`
	IsLegacy     bool   `json:"isLegacy"`
	Capabilities struct {
		OptionDescriptors []OptionDescriptor `json:"optionDescriptors"`
	} `json:"capabilities"`
}

type OptionDescriptor struct {
	ID                   string   `json:"id"`
	Label                string   `json:"label"`
	Description          string   `json:"description"`
	Type                 string   `json:"type"`
	CurrentValue         any      `json:"currentValue"`
	PromptInjectedValues []string `json:"promptInjectedValues"`
	Options              []struct {
		ID          string `json:"id"`
		Label       string `json:"label"`
		Description string `json:"description"`
		IsDefault   bool   `json:"isDefault"`
	} `json:"options"`
}
