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
	ArchivedAt            string          `json:"archivedAt"`
}

// ProjectIcon is a project's chosen icon: a monogram in one of T3's icon
// colours, a lucide icon or an emoji. tern draws the monograms.
type ProjectIcon struct {
	Kind  string `json:"kind"` // "monogram", "lucide", "emoji"
	Text  string `json:"text"` // a monogram's glyphs
	Color string `json:"color"`
}

type RuntimeRequest struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

type Lineage struct {
	ParentThreadID       string `json:"parentThreadId"`
	RelationshipToParent string `json:"relationshipToParent"`
	SideChat             bool   `json:"sideChat"`
}

// BackgroundTask is work a run left going: a subagent, a command, a
// monitor. Awaited ones hold the thread at idle until they end.
type BackgroundTask struct {
	Kind    string `json:"kind"`
	Awaited bool   `json:"awaited"`
}

// PullRequestLink is a pull request linked to a thread, with the server's
// last snapshot of it.
type PullRequestLink struct {
	Number   int    `json:"number"`
	URL      string `json:"url"`
	Source   string `json:"source"`
	LinkedAt string `json:"linkedAt"`
	Snapshot *struct {
		State   string `json:"state"` // "open", "closed", "merged"
		IsDraft bool   `json:"isDraft"`
	} `json:"snapshot"`
}

// LinkedPullRequest is the pull request a thread names, by number.
type LinkedPullRequest struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type Thread struct {
	ID                          string             `json:"id"`
	ProjectID                   string             `json:"projectId"`
	Title                       string             `json:"title"`
	CreatedAt                   string             `json:"createdAt"`
	UpdatedAt                   string             `json:"updatedAt"`
	LatestUserMessageAt         string             `json:"latestUserMessageAt"`
	LatestUserAuthoredMessageAt string             `json:"latestUserAuthoredMessageAt"`
	LatestRunID                 string             `json:"latestRunId"`
	LatestRunRequestedAt        string             `json:"latestRunRequestedAt"`
	LatestRunStartedAt          string             `json:"latestRunStartedAt"`
	LatestRunCompletedAt        string             `json:"latestRunCompletedAt"`
	LastVisitedAt               string             `json:"lastVisitedAt"`
	PinnedAt                    string             `json:"pinnedAt"`
	PinOrderKey                 string             `json:"pinOrderKey"`
	ActiveOrderKey              string             `json:"activeOrderKey"`
	SettledOverride             string             `json:"settledOverride"` // "settled", "active" or ""
	SettledAt                   string             `json:"settledAt"`
	UnsettledAt                 string             `json:"unsettledAt"`
	SnoozedUntil                string             `json:"snoozedUntil"`
	SnoozedAt                   string             `json:"snoozedAt"`
	AutoSettleDisabledAt        string             `json:"autoSettleDisabledAt"`
	ArchivedAt                  string             `json:"archivedAt"`
	DeletedAt                   string             `json:"deletedAt"`
	Status                      string             `json:"status"` // "idle" or the latest run's status
	LastError                   string             `json:"lastError"`
	LastErrorClass              string             `json:"lastErrorClass"` // "usage_limit", "provider_error"…
	ActiveRunID                 string             `json:"activeRunId"`
	ActiveProviderThreadID      string             `json:"activeProviderThreadId"`
	ActivityRunStatus           string             `json:"activityRunStatus"`
	ActivityRunStartedAt        string             `json:"activityRunStartedAt"`
	BlockedOnThreadWait         bool               `json:"blockedOnThreadWait"`
	PendingRuntimeRequest       *RuntimeRequest    `json:"pendingRuntimeRequest"`
	PendingGuestMessageCount    int                `json:"pendingGuestMessageCount"`
	HasActionableProposedPlan   bool               `json:"hasActionableProposedPlan"`
	PendingBackgroundTasks      []BackgroundTask   `json:"pendingBackgroundTasks"`
	Goal                        *Goal              `json:"goal"`
	TitleRegeneration           *Regeneration      `json:"titleRegeneration"`
	Lineage                     *Lineage           `json:"lineage"`
	CreationSource              string             `json:"creationSource"` // "provider" for a provider's own subagents
	ModelSelection              *ModelSelection    `json:"modelSelection"`
	ProviderInstanceID          string             `json:"providerInstanceId"`
	ProviderInstanceHistory     []string           `json:"providerInstanceHistory"`
	RuntimeMode                 string             `json:"runtimeMode"`
	InteractionMode             string             `json:"interactionMode"`
	Branch                      string             `json:"branch"`
	WorktreePath                string             `json:"worktreePath"`
	PullRequests                []PullRequestLink  `json:"pullRequests"`
	LinkedPullRequest           *LinkedPullRequest `json:"linkedPullRequest"`
	BranchPullRequest           *LinkedPullRequest `json:"branchPullRequest"`
	EnvironmentID               string             `json:"-"`
}

// Goal is the goal a provider's run works toward.
type Goal struct {
	Status string `json:"status"` // "active", "paused"…
}

// Regeneration is a title regeneration in progress.
type Regeneration struct {
	RequestID string `json:"requestId"`
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

// ServerConfig is the part of server.getConfig the composer and the
// sidebar need.
type ServerConfig struct {
	Providers            []Provider `json:"providers"`
	ScratchWorkspaceRoot string     `json:"scratchWorkspaceRoot"` // where "No project" threads live
	Settings             struct {
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
		SnoozePresets   []SnoozePreset `json:"snoozePresets"`
		EnvironmentIcon string         `json:"environmentIcon"` // the machine's icon, chosen in settings
	} `json:"settings"`
	Environment struct {
		Platform struct {
			Machine string `json:"machine"` // "server", "laptop", "desktop"…
		} `json:"platform"`
		Capabilities struct {
			ServerResolvedCommandContext bool `json:"serverResolvedCommandContext"`
			InlineMessageContext         bool `json:"inlineMessageContext"`
			ThreadStop                   bool `json:"threadStop"`
			ThreadAutoSettleOptOut       bool `json:"threadAutoSettleOptOut"`
			ThreadTitleRegeneration      bool `json:"threadTitleRegeneration"`
		} `json:"capabilities"`
	} `json:"environment"`
}

// SnoozePreset is one of the snooze menu's choices: so many minutes from
// now ("after"), or a local time on a day ("at": "today", "tomorrow", a
// weekday).
type SnoozePreset struct {
	Kind    string `json:"kind"`
	Minutes int    `json:"minutes"`
	Day     string `json:"day"`
	Time    string `json:"time"` // "18:00"
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
