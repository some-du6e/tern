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
}

type RuntimeRequest struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// Request is one of a thread's runtime requests as its projection has it
// (OrchestrationV2RuntimeRequest): what it asks, whether it's still
// pending, and whether the provider can still take an answer.
type Request struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Status             string `json:"status"` // pending, resolved, expired, cancelled
	CreatedAt          string `json:"createdAt"`
	ResponseCapability struct {
		Type             string `json:"type"` // live, message, not_resumable
		SupportsFeedback bool   `json:"supportsFeedback"`
	} `json:"responseCapability"`
}

// ApprovalOption is one answer an approval offers.
type ApprovalOption struct {
	Decision string `json:"decision"` // accept, acceptForSession, decline, cancel
	Label    string `json:"label"`
	Warning  string `json:"warning"`
}

type Lineage struct {
	ParentThreadID       string `json:"parentThreadId"`
	RelationshipToParent string `json:"relationshipToParent"`
}

type Thread struct {
	ID                    string          `json:"id"`
	ProjectID             string          `json:"projectId"`
	Title                 string          `json:"title"`
	CreatedAt             string          `json:"createdAt"`
	UpdatedAt             string          `json:"updatedAt"`
	LatestUserMessageAt   string          `json:"latestUserMessageAt"`
	LatestRunCompletedAt  string          `json:"latestRunCompletedAt"`
	LastVisitedAt         string          `json:"lastVisitedAt"`
	PinnedAt              string          `json:"pinnedAt"`
	PinOrderKey           string          `json:"pinOrderKey"`
	ActiveOrderKey        string          `json:"activeOrderKey"`
	SettledAt             string          `json:"settledAt"`
	SnoozedUntil          string          `json:"snoozedUntil"`
	ArchivedAt            string          `json:"archivedAt"`
	DeletedAt             string          `json:"deletedAt"`
	LastError             string          `json:"lastError"`
	ActiveRunID           string          `json:"activeRunId"`
	LatestRunID           string          `json:"latestRunId"`
	ActivityRunStatus     string          `json:"activityRunStatus"`
	ActivityRunStartedAt  string          `json:"activityRunStartedAt"`
	PendingRuntimeRequest *RuntimeRequest `json:"pendingRuntimeRequest"`
	Lineage               *Lineage        `json:"lineage"`
	ModelSelection        *ModelSelection `json:"modelSelection"`
	RuntimeMode           string          `json:"runtimeMode"`
	InteractionMode       string          `json:"interactionMode"`
	Branch                string          `json:"branch"`
	WorktreePath          string          `json:"worktreePath"`
	EnvironmentID         string          `json:"-"`
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
	// ScheduledTaskID is set on a user message an automation sent.
	ScheduledTaskID string `json:"scheduledTaskId"`

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

	// approval_request: what it asks for (its detail is Prompt) and its answers
	RequestKind string           `json:"requestKind"`
	AppName     string           `json:"appName"`
	Options     []ApprovalOption `json:"options"`

	Steps []struct {
		Step   string `json:"step"`
		Text   string `json:"text"`
		Status string `json:"status"`
	} `json:"steps"`
	Failure *struct {
		Message string `json:"message"`
	} `json:"failure"`
	Questions    []Question        `json:"questions"`
	ResponseMode string            `json:"responseMode"` // "message": the questions can be dismissed
	Attachments  []json.RawMessage `json:"attachments"`
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
	// Who sent it: automations and other agents send user messages too.
	CreatedBy       string `json:"createdBy"` // user, agent, system
	ScheduledTaskID string `json:"scheduledTaskId"`
	Context         *struct {
		Guest json.RawMessage `json:"guest"`
	} `json:"context"`
}

// ThreadItem is one value from orchestration.subscribeThread.
type ThreadItem struct {
	Kind       string `json:"kind"`
	Projection *struct {
		VisibleTurnItems []struct {
			Item json.RawMessage `json:"item"`
		} `json:"visibleTurnItems"`
		Runs            []Run     `json:"runs"`
		Messages        []Message `json:"messages"`
		RuntimeRequests []Request `json:"runtimeRequests"`
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

// ServerConfig is the part of server.getConfig the composer needs.
type ServerConfig struct {
	Providers []Provider `json:"providers"`
	// ScratchWorkspaceRoot is where the scratch project, for threads
	// without one, lives; "" when the server has none.
	ScratchWorkspaceRoot string `json:"scratchWorkspaceRoot"`
	Settings             struct {
		DefaultModelSelection *ModelSelection `json:"defaultModelSelection"`
		DefaultRuntimeMode    string          `json:"defaultRuntimeMode"`
		// PromptSuggestionsEnabled: the server streams the providers'
		// guesses at the next prompt.
		PromptSuggestionsEnabled bool `json:"promptSuggestionsEnabled"`
		Favorites                []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"favorites"`
		ProviderModelPreferences map[string]struct {
			HiddenModels []string `json:"hiddenModels"`
			ModelOrder   []string `json:"modelOrder"`
		} `json:"providerModelPreferences"`
	} `json:"settings"`
	Environment struct {
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
	// The composer's / and $ menus, and the permission modes it offers
	// (all four when it names none).
	SlashCommands         []SlashCommand `json:"slashCommands"`
	Skills                []Skill        `json:"skills"`
	SupportedRuntimeModes []string       `json:"supportedRuntimeModes"`
}

// Entry is a file or folder from projects.searchEntries, its path
// relative to the workspace.
type Entry struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // "file" or "directory"
}

// SlashCommand is a provider's / command.
type SlashCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Input       *struct {
		Hint string `json:"hint"`
	} `json:"input"`
}

// Skill is a provider's skill, for the $ menu.
type Skill struct {
	Name             string `json:"name"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description"`
	ShortDescription string `json:"shortDescription"`
	Path             string `json:"path"`
	Scope            string `json:"scope"`
	Enabled          bool   `json:"enabled"`
	UserInvocable    *bool  `json:"userInvocable"`
}

type Model struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	ShortName    string `json:"shortName"`
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
