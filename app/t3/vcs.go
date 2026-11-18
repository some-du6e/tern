package t3

import "encoding/json"

// Git, as docs/projects-files-vcs-terminal.md §3 describes it
// (packages/contracts/src/git.ts). Every call takes the checkout's absolute
// cwd on the environment's machine.

// RemoteOpenTarget is an address this computer's editors can reach the
// environment's machine at over SSH (editor.ts).
type RemoteOpenTarget struct {
	Kind     string `json:"kind"` // tailscale, mdns
	Host     string `json:"host"`
	Username string `json:"username"`
}

// SourceControlProvider is the host a repository's remote lives on.
type SourceControlProvider struct {
	Kind    string `json:"kind"` // github, gitlab, …
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

// VcsChangeRequest is the open pull request (or merge request) for a branch.
type VcsChangeRequest struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	BaseRef string `json:"baseRef"`
	HeadRef string `json:"headRef"`
	State   string `json:"state"` // open, closed, merged
	IsDraft bool   `json:"isDraft"`
}

// VcsStatusLocal is the half of a checkout's status git answers locally.
type VcsStatusLocal struct {
	IsRepo                bool                   `json:"isRepo"`
	SourceControlProvider *SourceControlProvider `json:"sourceControlProvider"`
	HasPrimaryRemote      bool                   `json:"hasPrimaryRemote"`
	IsDefaultRef          bool                   `json:"isDefaultRef"`
	RefName               string                 `json:"refName"` // empty on a detached HEAD
	HasWorkingTreeChanges bool                   `json:"hasWorkingTreeChanges"`
	WorkingTree           struct {
		Files []struct {
			Path       string `json:"path"`
			Insertions int    `json:"insertions"`
			Deletions  int    `json:"deletions"`
		} `json:"files"`
		Insertions int `json:"insertions"`
		Deletions  int `json:"deletions"`
	} `json:"workingTree"`
	BranchChanges *struct {
		BaseRef    string `json:"baseRef"`
		Insertions int    `json:"insertions"`
		Deletions  int    `json:"deletions"`
	} `json:"branchChanges"`
}

// VcsStatusRemote is the half that needs the remote: upstream and the pull
// request.
type VcsStatusRemote struct {
	HasUpstream         bool              `json:"hasUpstream"`
	AheadCount          int               `json:"aheadCount"`
	BehindCount         int               `json:"behindCount"`
	AheadOfDefaultCount *int              `json:"aheadOfDefaultCount"`
	PR                  *VcsChangeRequest `json:"pr"`
}

// VcsStatus is both halves, as vcs.refreshStatus returns them.
type VcsStatus struct {
	VcsStatusLocal
	VcsStatusRemote
}

// VcsStatusEvent is one value from subscribeVcsStatus.
type VcsStatusEvent struct {
	Tag    string           `json:"_tag"` // snapshot, localUpdated, remoteUpdated
	Local  *VcsStatusLocal  `json:"local"`
	Remote *VcsStatusRemote `json:"remote"`
}

// Apply merges ev into the status so far, the way applyGitStatusStreamEvent
// does (packages/shared/src/git.ts): a snapshot replaces both halves, an
// update replaces its own. A remote half that is null reads as no upstream.
func (ev VcsStatusEvent) Apply(cur *VcsStatus) *VcsStatus {
	next := VcsStatus{VcsStatusLocal: VcsStatusLocal{IsRepo: true}}
	if cur != nil {
		next = *cur
	}
	switch ev.Tag {
	case "snapshot":
		next.VcsStatusLocal, next.VcsStatusRemote = VcsStatusLocal{}, VcsStatusRemote{}
		if ev.Local != nil {
			next.VcsStatusLocal = *ev.Local
		}
		if ev.Remote != nil {
			next.VcsStatusRemote = *ev.Remote
		}
	case "localUpdated":
		if ev.Local != nil {
			next.VcsStatusLocal = *ev.Local
		}
	case "remoteUpdated":
		next.VcsStatusRemote = VcsStatusRemote{}
		if ev.Remote != nil {
			next.VcsStatusRemote = *ev.Remote
		}
	}
	return &next
}

// SubscribeVcsStatus streams cwd's status. The first event is a snapshot.
func (c *Conn) SubscribeVcsStatus(cwd string, onStatus func(VcsStatusEvent), onEnd func(error)) (cancel func()) {
	return c.Stream("subscribeVcsStatus", map[string]any{"cwd": cwd, "includeRemote": true}, func(v json.RawMessage) {
		var ev VcsStatusEvent
		if json.Unmarshal(v, &ev) == nil {
			onStatus(ev)
		}
	}, onEnd)
}

// VcsRef is a branch or other ref a checkout can switch to.
type VcsRef struct {
	Name         string `json:"name"`
	IsRemote     bool   `json:"isRemote"`
	RemoteName   string `json:"remoteName"`
	Current      bool   `json:"current"`
	IsDefault    bool   `json:"isDefault"`
	WorktreePath string `json:"worktreePath"` // where it is checked out, if anywhere
}

// VcsRefs is one page of vcs.listRefs.
type VcsRefs struct {
	IsRepo           bool     `json:"isRepo"`
	HasPrimaryRemote bool     `json:"hasPrimaryRemote"`
	Refs             []VcsRef `json:"refs"`
	NextCursor       *int     `json:"nextCursor"`
	TotalCount       int      `json:"totalCount"`
}

// GitActionEvent is one value from git.runStackedAction: its progress, then
// its result or failure.
type GitActionEvent struct {
	Kind     string `json:"kind"` // action_started, phase_started, hook_started, hook_output, hook_finished, action_finished, action_failed
	Phase    string `json:"phase"`
	Label    string `json:"label"`
	HookName string `json:"hookName"`
	Text     string `json:"text"`
	Message  string `json:"message"`
	Result   *struct {
		Branch struct {
			Status string `json:"status"`
			Name   string `json:"name"`
		} `json:"branch"`
		PR struct {
			Status string `json:"status"`
			Number int    `json:"number"`
		} `json:"pr"`
		Toast struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"toast"`
	} `json:"result"`
}
