package t3

import "encoding/json"

// What the Diff panel and the changed files cards read: the thread's
// checkpoints, the patches between them, and Git's own diffs of a
// checkout (docs/orchestration.md, docs/projects-files-vcs-terminal.md
// §4; packages/contracts/src/orchestrationV2.ts, checkpointDiff.ts,
// review.ts).

// Checkpoint is the snapshot T3 takes after a run: the files the run
// changed. Files may be only the first of them; FilesTotal then counts
// them all.
type Checkpoint struct {
	ID            string            `json:"id"`
	RunID         string            `json:"runId"`
	AppRunOrdinal int               `json:"appRunOrdinal"` // the run's "Turn N"; 0 for none
	Status        string            `json:"status"`        // ready, missing, error, stale
	Files         []CheckpointFile  `json:"files"`
	FilesTotal    *CheckpointTotals `json:"filesTotal"`
	CapturedAt    string            `json:"capturedAt"`
}

// CheckpointFile is one changed file and its line counts.
type CheckpointFile struct {
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// CheckpointTotals counts all of a checkpoint's files.
type CheckpointTotals struct {
	Count     int `json:"count"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

// ThreadCheckpoints reads the checkpoints out of one value of
// orchestration.subscribeThread: a snapshot's, or nil.
func ThreadCheckpoints(raw json.RawMessage) []Checkpoint {
	var v struct {
		Projection *struct {
			Checkpoints []Checkpoint `json:"checkpoints"`
		} `json:"projection"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Projection == nil {
		return nil
	}
	return v.Projection.Checkpoints
}

// TurnDiff is the patch of the thread's turns from to to, by checkpoint
// count: one turn is to-1 → to.
func (c *Conn) TurnDiff(threadID string, from, to int, ignoreWhitespace bool) (string, error) {
	var out struct {
		Diff string `json:"diff"`
	}
	err := c.Call("orchestration.getTurnDiff", map[string]any{
		"threadId": threadID, "fromTurnCount": from, "toTurnCount": to, "ignoreWhitespace": ignoreWhitespace,
	}, &out)
	return out.Diff, err
}

// FullThreadDiff is the patch of the thread from its start to turn to.
func (c *Conn) FullThreadDiff(threadID string, to int, ignoreWhitespace bool) (string, error) {
	var out struct {
		Diff string `json:"diff"`
	}
	err := c.Call("orchestration.getFullThreadDiff", map[string]any{
		"threadId": threadID, "toTurnCount": to, "ignoreWhitespace": ignoreWhitespace,
	}, &out)
	return out.Diff, err
}

// DiffSource is one of a checkout's Git diffs: its working tree against
// HEAD, its branch against the base, or one commit.
type DiffSource struct {
	Kind          string     `json:"kind"` // working-tree, branch-range, commit
	Title         string     `json:"title"`
	CommitSubject string     `json:"commitSubject"`
	BaseRef       string     `json:"baseRef"`
	HeadRef       string     `json:"headRef"`
	Diff          string     `json:"diff"`
	Truncated     bool       `json:"truncated"`
	Files         []DiffStat `json:"files"`
}

// DiffStat is one file's counts in a DiffSource.
type DiffStat struct {
	Path         string `json:"path"`
	PreviousPath string `json:"previousPath"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
}

// DiffPreview is review.getDiffPreview's answer.
type DiffPreview struct {
	Sources    []DiffSource `json:"sources"`
	Incomplete bool         `json:"incomplete"`
}

// DiffPreview reads a checkout's Git diffs at cwd: the branch against
// baseRef ("" for the server's pick), the working tree, or with sha that
// commit. It needs the filesystem:read scope.
func (c *Conn) DiffPreview(cwd, baseRef, sha, sourceKind string, ignoreWhitespace bool) (DiffPreview, error) {
	in := map[string]any{"cwd": cwd, "ignoreWhitespace": ignoreWhitespace}
	if baseRef != "" {
		in["baseRef"] = baseRef
	}
	if sha != "" {
		in["commitSha"] = sha
	} else if sourceKind != "" {
		in["sourceKind"] = sourceKind
	}
	var out DiffPreview
	err := c.Call("review.getDiffPreview", in, &out)
	return out, err
}

// BranchCommit is one commit of a branch, newest first.
type BranchCommit struct {
	SHA         string `json:"sha"`
	Subject     string `json:"subject"`
	CommittedAt string `json:"committedAt"`
}

// BranchCommits lists the commits on cwd's branch since baseRef ("" for
// the server's pick); truncated when only the latest came.
func (c *Conn) BranchCommits(cwd, baseRef string) ([]BranchCommit, bool, error) {
	in := map[string]any{"cwd": cwd}
	if baseRef != "" {
		in["baseRef"] = baseRef
	}
	var out struct {
		Commits   []BranchCommit `json:"commits"`
		Truncated bool           `json:"truncated"`
	}
	err := c.Call("review.getBranchCommits", in, &out)
	return out.Commits, out.Truncated, err
}
