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
		Class   string `json:"class"` // usage_limit, provider_error, …
		Code    string `json:"code"`
		ResetAt string `json:"resetAt"`
	} `json:"failure"`
	Questions   []Question        `json:"questions"`
	Attachments []json.RawMessage `json:"attachments"`

	// What the timeline's rows read (orchestrationV2.ts:1559-1912).
	ParentItemID   string `json:"parentItemId"`
	ProviderTurnID string `json:"providerTurnId"`
	// A user message: turn_start, queued_turn, steer or
	// promoted_queued_to_steer; who sent it.
	InputIntent    string `json:"inputIntent"`
	CreatedBy      string `json:"createdBy"`
	CreationSource string `json:"creationSource"`
	// An assistant message: commentary, partial_answer, final_answer.
	Phase           string             `json:"phase"`
	OutputOmitted   bool               `json:"outputOmitted"`
	ViewedImagePath string             `json:"viewedImagePath"`
	ToolSurface     string             `json:"toolSurface"` // browser, computer
	Changes         []FileChangeDetail `json:"changes"`
	Results         []SearchResult     `json:"results"`
	RequestKind     string             `json:"requestKind"`
	AppName         string             `json:"appName"`
	Options         []ApprovalOption   `json:"options"`
	QuestionAnswer  *QuestionAnswer    `json:"questionAnswer"`
	Retry           *struct {
		Attempt      int  `json:"attempt"`
		MaxAttempts  *int `json:"maxAttempts"`
		RetryDelayMs *int `json:"retryDelayMs"`
	} `json:"retry"`
	Summary          string `json:"summary"`
	BeforeTokenCount *int   `json:"beforeTokenCount"`
	AfterTokenCount  *int   `json:"afterTokenCount"`
	Source           *struct {
		Type     string `json:"type"` // run, node, provider_thread
		ThreadID string `json:"threadId"`
	} `json:"source"`
	TargetThreadID string `json:"targetThreadId"`
}

// FileChangeDetail is one operation of a file change.
type FileChangeDetail struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	OldPath   string `json:"oldPath"`
	FileType  string `json:"fileType"`
	MimeType  string `json:"mimeType"`
}

// SearchResult is a file search's match or a web search's hit.
type SearchResult struct {
	FileName string `json:"fileName"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Preview  string `json:"preview"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
}

// ApprovalOption is one answer an approval offers.
type ApprovalOption struct {
	Decision string `json:"decision"` // accept, acceptForSession, decline, cancel
	Label    string `json:"label"`
	Warning  string `json:"warning"`
}

// QuestionAnswer is what the user answered a question with.
type QuestionAnswer struct {
	RequestID        string                       `json:"requestId"`
	QuestionTextByID map[string]string            `json:"questionTextById"`
	Answers          map[string]json.RawMessage   `json:"answers"`
	Attachments      map[string][]json.RawMessage `json:"attachmentsByQuestionId"`
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
	Ordinal       int    `json:"ordinal"`
	RequestedAt   string `json:"requestedAt"`
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

// ServerConfig is the part of server.getConfig the composer needs.
type ServerConfig struct {
	Providers []Provider `json:"providers"`
	Settings  struct {
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
