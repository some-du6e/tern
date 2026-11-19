package t3

// Pull requests as the details card reads and acts on them
// (packages/contracts/src/pullRequest.ts, docs/integrations.md §1). The
// server reads the host with the environment's own credential; every call
// names the pull request by a PullRequestRef.

// PullRequestRef names a pull request: the project whose checkout and
// credential the server uses, and the pull request's host, repository
// ("owner/name") and number.
type PullRequestRef struct {
	ProjectID  string `json:"projectId"`
	Host       string `json:"host,omitempty"`
	Repository string `json:"repository"`
	Number     int    `json:"number"`
}

// RepositoryIdentity is the repository a project's checkout is of
// (environment.ts), as far as the card needs it: its host's kind, owner and
// name.
type RepositoryIdentity struct {
	Provider string `json:"provider"` // github, gitlab, …
	Owner    string `json:"owner"`
	Name     string `json:"name"`
}

// PullRequestCheck is one of a pull request's checks.
type PullRequestCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pending, action-required, success, failure, skipped, neutral, cancelled
	URL    string `json:"url"`
}

// PullRequestDetail is the part of pullRequests.detail the card reads:
// what the pull request is and what it's waiting on, and which actions the
// host and the viewer allow.
type PullRequestDetail struct {
	Provider     string `json:"provider"`
	Capabilities struct {
		Actions      []string `json:"actions"`
		MergeMethods []string `json:"mergeMethods"` // merge, squash, rebase
	} `json:"capabilities"`
	ViewerPermissions struct {
		Actions []string `json:"actions"`
	} `json:"viewerPermissions"`
	Number            int                `json:"number"`
	Title             string             `json:"title"`
	State             string             `json:"state"` // open, closed, merged
	IsDraft           bool               `json:"isDraft"`
	Mergeability      string             `json:"mergeability"` // mergeable, conflicting, unknown
	Checks            []PullRequestCheck `json:"checks"`
	MergeCapabilities map[string]bool    `json:"mergeCapabilities"`
	ObservedAt        *float64           `json:"observedAt"`
}

// PullRequestChecks is pullRequests.checks: the checks alone, read again
// while they run. Null when the host reports none.
type PullRequestChecks struct {
	State      string             `json:"state"`
	Checks     []PullRequestCheck `json:"checks"`
	ObservedAt *float64           `json:"observedAt"`
}
