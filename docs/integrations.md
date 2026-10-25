# T3 Code RPC reference: pull requests, tasks, sharing, apps, previews and the rest

This covers the WebSocket methods that `t3-connect.md`, `orchestration.md`,
`server-settings-models.md` and `projects-files-vcs-terminal.md` leave out. It
comes from upstream `pingdotgg/t3code` at `80455376`. Paths are relative to
that repo. I checked the read-only shapes against a local
`0.1.3-nightly.20261023` server with an `orchestration:read` pairing; where the
live server and the source disagree, I say so.

Framing, tickets, `Ack` and errors are in `t3-connect.md`. Every method here is
a `Request` with a `tag` from `WS_METHODS` in `packages/contracts/src/rpc.ts:387`.
"Call" means one `Exit`. "Stream" means `Chunk`s, which you must `Ack`.

Thread commands such as `thread.pull-request.link` go through
`orchestration.dispatchCommand`, covered in `orchestration.md`. They show up
here only where a feature needs them.

## Contents

1. Pull requests and thread links
2. Scheduled tasks and webhooks
3. Guest shares
4. Secret requests
5. MCP apps
6. Preview, server browser and devices
7. Odds and ends: agent sessions, attachments, assets, cloud, lifecycle streams
8. Auth scopes
9. What a thread view has to handle

## 1. Pull requests and thread links

Contracts: `packages/contracts/src/pullRequest.ts`,
`pullRequestReviewGuide.ts`, `threadPullRequest.ts`. Server:
`apps/server/src/pullRequest/PullRequestService.ts`, handlers in
`apps/server/src/ws.ts:2565`. Web: `apps/web/src/components/pullRequest/`, the
`/pull-requests` route, and the client cache in
`packages/client-runtime/src/state/pullRequests.ts`.

The server reads the host (GitHub, GitLab, Azure and so on) with whatever
credential the environment holds for the project's checkout. The client never
sees a host token. Gate the whole feature on `capabilities.pullRequests` in
`/.well-known/t3/environment`.

### The reference every call takes

Almost every method takes a `PullRequestRef` (`pullRequest.ts:403`):

```ts
{
  projectId: string,           // a T3 project whose repo is on the host
  host?: string,               // "github.com"; send it when you know it
  repository: string,          // "owner/name"
  number: number,
  allowStale?: boolean,        // let the server answer from cache
  expectedAccountId?: string,  // only for cross-environment routing, skip it
}
```

`projectId` is how the server picks the checkout and credential. A
`PullRequestListEntry` gives you all four fields. For a link on a thread, use
the thread's `projectId` plus the link's `host`, `repository` and `number`.
That is what I used for the live check, and it worked.

### Errors

Every PR method fails with one of:

| `_tag` | Fields | Meaning |
| --- | --- | --- |
| `PullRequestUnavailableError` | `reason`: `cli-missing`, `cli-unauthenticated`, `provider-unsupported`; optional `provider`, `requirement` | No credential, a refused credential, or a host T3 can't browse. `requirement` is the host's own wording; show it. |
| `PullRequestOperationError` | `operation`, `detail`, optional `reason: "not-found"` | The host call failed. |
| `EnvironmentAuthorizationError` | `requiredScope`, `requiredPermission` | Scope missing. |

### Reads

All need `orchestration:read`.

| Tag | Input | Output | Web UI use |
| --- | --- | --- | --- |
| `pullRequests.list` | `PullRequestListInput` | `PullRequestListResult` | The pull requests page. |
| `pullRequests.listStats` | `{refs: PullRequestRef[]}` (max 500) | `{stats: [{projectId, repository, number, additions, deletions}]}` | Fills the +/- counts that GitHub listings leave at zero. |
| `pullRequests.filterOptions` | `{kind: "labels"\|"authors", projectId?, projectIds?, host?, query?}` | `{labels, authors, truncated}` | Label and author filter pickers. |
| `pullRequests.summary` | ref | `PullRequestSummary` | Status chips on threads and links, refetched every 60 s. |
| `pullRequests.preview` | ref | `PullRequestPreview` | Hover card for a pasted PR link. |
| `pullRequests.detail` | ref | `PullRequestDetail` | The detail panel header, actions menu and Summary tab. |
| `pullRequests.checks` | ref | `PullRequestChecks \| null` | Checks popover, refetched every 45 s. |
| `pullRequests.activity` | ref | `PullRequestActivity` | Conversation, review threads, commits. |
| `pullRequests.threadComments` | ref + `{threadId, cursor}` | `{comments, nextCursor}` | "More replies" in a long review thread. |
| `pullRequests.diffFileContents` | ref + `{commit?, changeType, oldPath, newPath}` | `{oldContents, newContents}` | Expanding context in the Code tab. |
| `pullRequests.filesViewed` | ref | `{files: [{path, state}], truncated}` | Viewed checkboxes. Missing files are unviewed. |
| `pullRequests.changeBreakdown` | ref | `{code, tests, mocks, generated, docs}` each `{additions, deletions}`, or `null` | Split of the line counts. |
| `pullRequests.stack` | ref | `PullRequestStack \| null` | Stacked PR header. |
| `pullRequests.linkedThreads` | ref | `{threads: [{id, projectId, title, archivedAt}]}` | "Threads on this PR", polled every 10 s. |
| `pullRequests.reviewerCandidates` | ref | `{candidates: [{login, name, avatarUrl, id, kind: "user"\|"team", isRequested}], truncated}` | Reviewer picker. |
| `pullRequests.labelCandidates` | ref | `{candidates: [{name, color, description, isApplied}], truncated}` | Label picker. |
| `pullRequests.reviewGuide` | ref | `PullRequestReviewGuideResult` | Guide tab. |
| `pullRequests.routing`, `pullRequests.routingIdentity` | ref, `{host}` | `{accountId, host, provider, viewer, ...}` | Cross-environment routing. A single-environment client can skip both. |
| `pullRequests.reportState` | `{reference, state, isDraft?}` | void | Tells the PR's home environment what a routed read saw. Skip it. |
| `pullRequests.invalidate` | `{reference?, filesViewedOnly?}` | void | The refresh button. Drops caches and asks link sync to recheck. |
| `pullRequests.subscribeRefreshes` | `{}` | stream of numbers | See below. |

The diff itself is HTTP, not RPC, because patches get big:
`POST {httpBaseUrl}/api/pull-requests/diff` with the same auth headers as the
other environment HTTP routes and a JSON body of ref fields plus `cursor?` and
`commit?`. It returns `{patch, truncated, nextCursor, omittedFileStats?}`.
GitHub refuses diffs past 300 files, so follow `nextCursor`
(`packages/contracts/src/environmentHttp.ts:756`,
`apps/server/src/pullRequest/http.ts`).

### List shapes

```ts
// PullRequestListInput
{
  state: "all" | "open" | "closed" | "merged",
  involvement?: "all" | "reviewing" | "authored",
  filters?: {
    draft?: "only" | "hide",
    review?: "approved" | "changes-requested" | "review-required" | "none",
    checks?: "passing" | "failing",
    labels?: string[][],          // every group must match, any name inside a group
    excludedLabels?: string[],
    author?: string,              // "me" or "@me" means the signed-in host user
  },
  projectId?: string, projectIds?: string[], host?: string,
  limit?: number,                 // 1..500, PER REPOSITORY, default 99
  cursors?: Record<string, string>,
  query?: string,
}

// PullRequestListResult
{
  viewers: Record<host, login>,
  providers: [{host, kind, searchesOnHost, actions?, projectCount, configured, detail}],
  entries: PullRequestListEntry[],
  errors: [{projectId, projectTitle, message}],
  truncated: boolean,
  nextCursors: Record<string, string>,  // pass back as `cursors` for the next page
}
```

`limit` applies to each repository, not the whole result. `limit: 3` gave me 8
entries across several repos. Cursors are keyed per repository too. After you
append a page, re-sort by `updatedAt`, because one repo's older slice can
outrank another's rows (`pullRequest.ts:385`).

A `PullRequestListEntry` has `provider, host, projectId, projectTitle,
repository, number, title, url, author, headBranch, baseBranch, state, isDraft,
mergeability, additions, deletions, createdAt, updatedAt, observedAt?,
viewerReviewRequested, labels, reviewDecision?, checksState?, stack?`.
`additions` and `deletions` are zero until you call `listStats`.

### Detail, checks and activity

`PullRequestDetail` (`pullRequest.ts:555`) is the big one. Beyond the list
fields it has `body`, `changedFiles`, `headSha?`, `headRepositoryNameWithOwner?`,
`mergedAt`, `closedAt`, `reviewers`, `labels`, `checks`, `mergeCapabilities`
(`{merge, squash, rebase}`), `viewer?`, `baseComparison?` (`up-to-date`,
`behind`, `unknown`), `behindBy?`, `autoMergeEnabled?`, `autoMergeMethod?`,
`workflowApprovalsRequired?`, `workspaceRoot`, and two gates:

- `capabilities` says what the host can do at all: `diff`, `comment`,
  `actions[]`, `mergeMethods[]`, `deleteBranch?`, `updateMethods?`, `search`,
  `reactions?`, `viewedFiles?` (`host` or `environment`, meaning where marks are
  stored), `review: {inlineComment, rangeComment?, reply, resolve, verdicts[]}`,
  `reviewers: {request, listCandidates}`, `edit?: {changeRequest, comment}`,
  `stacks?`, `stackActions?`, `labels?`, `checksAlwaysReport?`.
- `viewerPermissions` says what this user may do: `actions[]`, `comment`,
  `resolve`, `verdicts[]`, `requestReviewers`, `updateMethods?`, `labels?`,
  `editChangeRequest?`, `stackRebase?`. Unreported permissions count as granted.

Show a button only when both allow it. Absent optional capabilities mean "no".

A check is `{name, status, description, url, required?}` where `status` is
`pending`, `action-required`, `success`, `failure`, `skipped`, `neutral` or
`cancelled`. `pullRequests.checks` returns `{observedAt?, state, checks}`.

`PullRequestActivity`:

```ts
{
  author?, reviewers?,
  comments: [{id, kind: "issue-comment" | "review-comment" | "review", author, body,
              createdAt, editedAt?, url, path, reviewState, reactions?, canEdit?}],
  commentCount, commentsTruncated, reviewThreadsTruncated?,
  reviewThreads: [{id, path, line, side: "left" | "right", isResolved, isOutdated,
                   comments: [{id, author, body, createdAt, editedAt?, url, reactions?, canEdit?}],
                   commentCount?, nextCommentsCursor?, canResolve?}],
  commits: [{oid, messageHeadline, committedDate, additions?, deletions?, authors?}],
  reactions?,   // on the PR body
}
```

A reaction is `{content, count, actors, viewerHasReacted}`. `content` is one of
`thumbs-up`, `thumbs-down`, `laugh`, `hooray`, `confused`, `heart`, `rocket`,
`eyes`. `actors` leaves out the viewer, `count` includes them.

### Writes

All need `source-control:write` except `generateReviewGuide`, which needs
`orchestration:operate` because it runs a model. All return void.

| Tag | Input (ref fields plus) | Web UI use |
| --- | --- | --- |
| `pullRequests.runAction` | `action`, `mergeMethod?`, `updateMethod?`, `stackNumber?`, `expectedStackHeads?` | Merge, ready, draft, close, reopen, delete-branch, update-branch, enable/disable auto-merge, revert, approve-workflows. |
| `pullRequests.update` | `title?`, `body?` | Inline title and description edits. |
| `pullRequests.comment` | `body` | Conversation comment. |
| `pullRequests.updateComment` | `commentId`, `kind: "issue-comment"\|"review-comment"`, `body` | Edit your comment. |
| `pullRequests.submitReview` | `verdict`, `body`, `comments[]` | Review form. |
| `pullRequests.replyToThread` | `threadId`, `body` | Reply in a review thread. |
| `pullRequests.setThreadResolution` | `threadId`, `resolved` | Resolve or unresolve. |
| `pullRequests.setReaction` | `subjectId?`, `content`, `reacted` | Reaction toggles. No `subjectId` means the PR itself. |
| `pullRequests.setFilesViewed` | `files: [{path, viewed}]` (max 500) | Viewed checkboxes. |
| `pullRequests.requestReviewers` | `reviewers: [{id, kind}]` (1 to 25), `requested` | Reviewer picker. |
| `pullRequests.setLabels` | `labels: string[]` (1 to 25), `applied` | Label picker. |
| `pullRequests.generateReviewGuide` | ref only | "Write a guide". Returns `PullRequestReviewGuideResult`. |

Comment bodies are 1 to 65,536 characters and whitespace is kept. Review drafts:

```ts
{
  path: string,
  oldPath?: string,               // GitLab wants both names for a rename
  position: {kind: "added", newLine}
          | {kind: "deleted", oldLine}
          | {kind: "context", oldLine, newLine, side: "left" | "right"},
  startPosition?: same union,     // only if capabilities.review.rangeComment
  body: string,
}
```

For a merge, the web client first refetches `detail` with `allowStale: false`,
checks the PR is open, not a draft, and that both `capabilities.actions` and
`viewerPermissions.actions` include `merge`. If the host supports stack
actions and the PR is in a stack, it refuses and sends the user to the stack
view (`state/pullRequests.ts:252`). A merge can succeed and only enqueue or
enable auto-merge, so re-read the summary afterwards.

`stackNumber` and `expectedStackHeads` are only for environments that
advertise `capabilities.pullRequestStackActions`.

### Review guides

`pullRequests.reviewGuide` returns:

```ts
{
  guide: null | {
    headSha, generatedAt, modelSelection,
    stops: [{
      title, summary,
      attention: null | {kind: "tradeoff" | "contract" | "product" | "risk" | "accessibility", note},
      excerpts: [{path, side: "new" | "old", start, end, caption}],
      files: [{path, note?}],
      notes: [{path, line, title, body}],
      questions: string[],
    }],
  },
  run: null | {_tag: "Running", headSha, startedAt} | {_tag: "Failed", headSha, failedAt, detail},
  currentHeadSha: string | null,
}
```

A guide is stale when `guide.headSha !== currentHeadSha`. After
`generateReviewGuide`, poll `reviewGuide` while `run._tag === "Running"`.
Guides live on the environment that wrote them. The web client never routes
them to another environment.

### Keeping PR data fresh

`pullRequests.subscribeRefreshes` takes `{}` and streams a growing number. A new
value means "your cached PR reads may be stale, refetch what's on screen". The
server bumps it after a turn finishes in a project, after any PR write, and
after `invalidate` (`PullRequestService.ts:3360`). The value itself means
nothing. The live server sent one number right away.

The web client's cache times are a fair default: list 30 s, checks 45 s,
summary, detail and activity 60 s, files viewed 15 s, filter options 5 min.

### Thread to PR links

Threads carry their PRs in the shell and thread streams
(`orchestrationV2.ts:395`):

| Field | Meaning |
| --- | --- |
| `pullRequests?: ThreadPullRequestLink[]` | Current model. Many links per thread. |
| `linkedPullRequest?: {projectId, repository, number, url} \| null` | Older single link. |
| `branchPullRequest?: same \| null` | The open PR for the thread's branch, found by the server. Not a link. |

On my server 68 of 1,004 threads had `pullRequests`, 513 had a
`branchPullRequest`, and none used `linkedPullRequest`.

`ThreadPullRequestLink` (`threadPullRequest.ts:107`):

```ts
{
  host, repository, number,       // the key; compare all three
  url,
  repositoryWebUrl?,
  source: "manual" | "created" | "agent" | "stack" | "stack-dismissed",
  linkedAt,
  snapshot: null | {state, title, headBranch, baseBranch, isDraft, updatedAt, syncedAt,
                    closedAt?, mergedAt?, author?, additions?, deletions?, changedFiles?,
                    reviewDecision?, checksState?, mergeability?},
  stack: null | {kind: "native", id, number, url, base, layers: [{number, headBranch, state}]},
  watch?: ThreadPullRequestWatch,  // the agent is watching CI and comments
  endedWatch?: ThreadPullRequestWatch,
}
```

Hide links with `source: "stack-dismissed"`. They are tombstones that stop sync
from re-adding a stack member (`packages/shared/src/threadPullRequests.ts:4`).
`snapshot` is the server's last sync, so a thread row can show state, title and
checks with no extra calls. Changes arrive as `thread.pull-request-link-updated`
events carrying one whole link, and as `thread.pull-request-synced` with the
whole thread.

To pick "the" PR for a thread badge, the web client takes the single open link,
otherwise the top of the newest open stack, otherwise the most recently updated
link (`packages/source-control-core/src/shared/threadPullRequests.ts:143`).

Link and unlink through `orchestration.dispatchCommand`
(`orchestration:operate`, `orchestrationV2.ts:2830`):

```json
{"type":"thread.pull-request.link","commandId":"...","threadId":"...",
 "host":"github.com","repository":"owner/name","number":42,
 "url":"https://github.com/owner/name/pull/42","source":"manual"}
{"type":"thread.pull-request.unlink","commandId":"...","threadId":"...",
 "host":"github.com","repository":"owner/name","number":42}
{"type":"thread.pull-request.watch","commandId":"...","threadId":"...",
 "host":"github.com","repository":"owner/name","number":42,"watching":true}
```

Pick the command by environment capability
(`packages/client-runtime/src/threadPullRequestCompatibility.ts`):

- `capabilities.threadPullRequests`: the commands above.
- only `capabilities.threadPullRequestLinking`: `thread.metadata.update` with
  `linkedPullRequest: {projectId, repository, number, url}` or `null`. Needs a
  project whose repo matches.
- neither: no linking.

Watching needs `capabilities.threadPullRequestWatch`. The agent's own MCP tools
(`link_pull_request`, `watch_pull_request`) produce `source: "agent"` links.
The schemas for those tools are in `threadMetadataMcp.ts` and
`orchestratorMcp.ts`. A client never calls them.

## 2. Scheduled tasks and webhooks

Contract: `packages/contracts/src/scheduledTask.ts`. Server:
`apps/server/src/scheduledTasks/ScheduledTaskService.ts`. Web: Settings,
Scheduled tasks (`apps/web/src/routes/settings.scheduled-tasks.tsx`), plus a
badge on user messages a task sent.

A task sends `prompt` on a schedule. With `threadId: null` each run launches a
new thread using the task's model, mode and workspace strategy. With a
`threadId` each run queues the prompt into that thread and follows the thread's
current provider (`ScheduledTaskService.ts:812`). Deleting the thread deletes
its tasks.

### Methods

| Tag | Kind | Scope | Input | Output |
| --- | --- | --- | --- | --- |
| `scheduledTasks.list` | call | `orchestration:read` | `{}` | `{tasks: ScheduledTask[]}` |
| `scheduledTasks.subscribe` | stream | `orchestration:read` | `{}` | `{tasks}` on every change, full list each time |
| `scheduledTasks.upsert` | call | `orchestration:operate` | `ScheduledTaskUpsertInput` | `{task}` |
| `scheduledTasks.setEnabled` | call | `orchestration:operate` | `{id, enabled}` | `{task}` |
| `scheduledTasks.delete` | call | `orchestration:operate` | `{id}` | `{id}` |
| `scheduledTasks.runNow` | call | `orchestration:operate` | `{id}` | `{task}` |
| `scheduledTasks.rotateWebhookToken` | call | `orchestration:operate` | `{id}` | `{task}` with a new URL |
| `scheduledTasks.listWebhookDeliveries` | call | `orchestration:operate` | `{id, acceptAnyOutcome?}` | `{deliveries: Summary[]}` |
| `scheduledTasks.getWebhookDelivery` | call | `orchestration:operate` | `{id, deliveryId, acceptAnyOutcome?}` | `{delivery}` |

Errors are `ScheduledTaskError {message, taskId?}`.

Without `orchestration:operate`, `list` and `subscribe` drop the `webhook` field
from every task, since the URL alone can start an agent (`ws.ts:1368`). Delivery
logs need operate for the same reason: they hold request bodies. The tasks on my
server were all interval tasks, so the live check didn't exercise the
stripping.

### Task shape

```ts
// ScheduledTask
{
  id, title, prompt, enabled,
  schedule: Schedule,
  projectId, threadId: string | null,
  workspaceStrategy: {type: "root", branch?}
                   | {type: "existing_worktree", worktreePath, branch?}
                   | {type: "worktree", baseRef, branch?, startFromOrigin?},
  modelSelection, runtimeMode, interactionMode,   // see server-settings-models.md
  createdBy: "user" | "agent" | "system",
  creationSource: "web" | "mobile" | "mcp" | "provider" | "server",
  createdAt, updatedAt,
  nextRunAt: string | null, lastRunAt: string | null,
  lastRunStatus: "never" | "running" | "succeeded" | "failed",
  lastRunError: string | null,
  runCount: number,
  webhook?: {path, url: string | null, hasSecret},  // webhook tasks only
}

// Schedule
{type: "interval", everyMs}                          // upsert requires >= 60000
{type: "fixed_time", timeOfDay: "HH:MM", weekdays?}  // 0 = Sunday; omit for daily; host local time
{type: "webhook",
 signature: null | {header, encoding: "hex" | "base64", prefix},
 maxDeliveryAgeMinutes?: number | null,              // 1..1440
 dedupe?: {replayKey?, coalesceKey?, coalesceWindowSeconds?},  // window 10..3600, default 120
 filters?: Filter[]}                                 // up to 16, all must match
```

`ScheduledTaskUpsertInput` is the task's writable fields plus `id?`,
`requireExisting?` (fail if an edited task was deleted meanwhile), `commandId?`
(idempotent create) and optional `createdBy`/`creationSource`. In a webhook
schedule, `signature` can also carry `secret` (omit it to keep the stored one)
or `secretRef` (a one-use ref from a secret request, see section 4).
`filters` omitted keeps the stored filters, `[]` clears them.

Filters (`scheduledTask.ts:120`):

```ts
{path, op: "equals" | "notEquals" | "startsWith", value: string}
{path, op: "in", value: string[]}     // no commas or surrounding spaces in values
{path, op: "exists"}
// path: "headers.x-github-event", "body.action", "query.source",
//       "body.issue.labels[].name" fans out over an array
```

A webhook prompt can use `{{body.path}}`, `{{headers.name}}`, `{{query.name}}`,
`{{body}}` and `{{request}}`. The web default is `"Handle this webhook:\n{{body}}"`.
Dedupe keys are templates made only of placeholders.

### Webhook URLs and deliveries

The environment serves `POST`, `PUT`, `PATCH` and `GET` on
`/api/hooks/:taskId/:token` and answers `202 {"deliveryId": "..."}`
(`environmentHttp.ts:850`). `webhook.path` is that path. `webhook.url` is the
public relay URL, `{relayUrl}/v1/hooks/{endpointKey}/{taskId}/{token}`, and is
`null` when the environment isn't linked to T3 Connect. The relay holds
requests for up to 24 hours while the environment is offline.

When `url` is null, the web app joins `path` onto the environment's
`httpBaseUrl` and warns that only machines that can reach that address can call
it (`packages/client-runtime/src/webhookAddress.ts`). Treat the URL like a
password. Don't log it.

The server takes at most 60 requests per minute per task and queues at most 20.
It keeps the last 50 deliveries, plus up to 20 duplicate or coalesced ones and
20 filtered ones (`ScheduledTaskService.ts:73`).

```ts
// ScheduledTaskWebhookDeliverySummary
{
  id, taskId, receivedAt, method, contentType: string | null, bodyBytes,
  outcome: "accepted" | "dispatch_failed" | "duplicate" | "coalesced" | "rejected_signature"
         | "disabled" | "rate_limited" | "expired" | "filtered" | string,
  signatureVerified, pendingWindow?, missingFields: string[], error: string | null,
}
// ScheduledTaskWebhookDelivery adds:
{ query, headers: Record<string, string>, body, bodyTruncated, renderedPrompt: string | null }
```

Send `acceptAnyOutcome: true`. Without it the server drops rows whose outcome
an old client wouldn't decode. Then keep a catch-all for unknown outcomes.

A user message sent by a task carries `scheduledTaskId`. The web timeline shows
the task's title and links to its settings (`TimelineUserMessageRow.tsx:140`).

## 3. Guest shares

Contracts: `guestShare.ts`, `guestStream.ts`. Server:
`apps/server/src/guest/GuestShareService.ts`, `apps/server/src/guest/http.ts`.
Web: the owner's share button, and the guest page at `/guest`
(`apps/web/src/components/guest/GuestPage.tsx`).

A guest share lets one person without an account watch a thread for 24 hours
and propose messages. The owner sends or dismisses each one. It is a beta
setting and needs a running T3 Connect tunnel.

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `guestShare.create` | `access:write` | `{threadId}` | `{url, share: ThreadGuestShare}` |

Errors: `GuestShareCreateError` with `reason` `disabled` (off in Settings,
Beta), `connect_unavailable`, `thread_unavailable`, `read_failed` or
`changed`. `url` is `https://{tunnel host}/guest#{credential}`. It hands out
thread access, so it needs the same scope as a pairing link. Creating a new
share replaces the old one.

The share lives on the thread as `guestShare` (`orchestrationV2.ts:438`) and
changes arrive as `thread.guest-share-updated`:

```ts
// ThreadGuestShare
{
  id, host, createdAt, expiresAt,
  guestName: string | null,       // set once the guest joins
  messages: [{id, text, status: "pending" | "sent" | "dismissed", submittedAt, attachments?}],
}
```

The owner acts with `dispatchCommand`:

- send: `message.dispatch` with `guestMessageId`. The server uses the guest's
  text and attachments, not yours.
- dismiss: `{"type":"thread.guest-message.dismiss", threadId, guestMessageId}`.
- end the share: `{"type":"thread.guest-share.close", threadId}`.

A guest can have 3 pending messages, 4,000 characters each, up to 4 files and
10 MB per message.

The guest side is plain HTTP under `/api/guest`, cookie-authenticated per
share, served only on the Connect host: `POST /redeem {credential, name}`,
`GET /session`, `GET /stream` (server-sent `GuestStreamEvent`: `view`, `thread`
carrying normal thread stream items, `ended`), `POST /messages`, and
`render`, `item`, `command-output` and `attachment` routes keyed by
`?share=`. A third-party client acting as the owner never needs these. I list
them so you know the guest page isn't your RPC.

## 4. Secret requests

Contract: `secretRequest.ts`; item type in `orchestrationV2.ts:1386`. Web:
`TimelineV2EventRow.tsx`, helpers in
`packages/client-runtime/src/secretRequest.ts`.

An agent calls the `request_secret` MCP tool when it needs an API key or
similar. The thread gets a turn item:

```ts
{
  ...turnItemBase,          // id, threadId, runId, ordinal, status, ...
  type: "secret_request",
  label: string,            // what's being asked for
  reason: string,
  placeholder?: string,
  secretStatus: "pending" | "saved" | "declined" | "cancelled",
}
```

The user answers with:

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `secrets.answerRequest` | `orchestration:operate` | `{threadId, turnItemId, answer: {type: "save", secret} \| {type: "decline"}}` | void |

`threadId` and `turnItemId` are the item's own `threadId` and `id`. Trim the
secret, and don't send an empty one. The server stores the value under a
one-use `SecretRef` and gives the agent only the ref. The thread only ever
learns the status, so the value never appears in any stream. The agent can
pass the ref on, for example as a webhook signing secret.

Errors are `SecretRequestError {reason}`. The ones a user can hit are
`not_found`, `already_answered` and `agent_stopped` (the run that asked has
ended, so the secret can't be used). The web shows the server's message for
`SecretRequestError` and `EnvironmentAuthorizationError` and a generic message
for anything else, so a value never leaks through an error string.

How the web renders it, which I'd copy:

| `secretStatus` | Row visibility | Show |
| --- | --- | --- |
| `pending` | `local` | Password field, Save, Decline. Note: "Stored securely, never shown to the agent". |
| `pending` | `inherited` or `synthetic` | "Waiting for an answer in the original thread". A forked thread can't answer. |
| `saved` | any | "Saved securely and kept private" |
| `declined` | any | "Declined" |
| `cancelled` | any | "Request ended" |

Visibility comes from the projected turn item wrapper (`orchestrationV2.ts:1642`).
Never echo the typed value into logs, crash reports or the transcript.

## 5. MCP apps

Contract: `mcpApps.ts`; reference parsing in `packages/shared/src/mcpApp.ts`
and `packages/shared/src/toolOutput.ts:186`. Server:
`apps/server/src/mcpApps/McpAppRequests.ts`. Web:
`apps/web/src/components/chat/McpAppFrame.tsx`, bridge in
`packages/client-runtime/src/mcpApps/host.ts`.

MCP apps are the MCP Apps extension (`io.modelcontextprotocol/ui`, protocol
`2026-01-26`). A tool on an MCP server returns a `ui://` HTML resource, and the
host renders it in a sandboxed frame inline in the thread. The app can call
back into its server's tools.

### Finding one in a thread

It's a `dynamic_tool` turn item whose `output.t3McpApp` holds a reference:

```ts
{
  attachmentId: string,   // the stored HTML
  server: string,
  tool: string,
  resourceUri: string,    // must start with "ui://"
  csp?: {connectDomains?, resourceDomains?, frameDomains?, baseUriDomains?},
  permissions?: {camera?, microphone?, geolocation?, clipboardWrite?},
  prefersBorder?: boolean,
}
```

Accept it only if `item.toolName === `${server}.${tool}``, so one tool can't
pose as another server's app. The socket keeps `t3McpApp` in the compact output
even when it sets `outputOmitted`. Fetch the full item with
`orchestration.getTurnItem` to get `input` and `output.result`, which the app
needs as its tool input and result.

### Loading the HTML

Call `assets.createUrl` with:

```json
{"resource":{"_tag":"attachment","attachmentId":"<attachmentId>","fileName":"app.html",
 "mimeType":"text/html","disposition":"inline","renderIntent":"mcp-app"}}
```

You get `{relativeUrl, expiresAt}`. Load `{httpBaseUrl}{relativeUrl}` in an
iframe with `sandbox="allow-scripts allow-forms"`. The URL is signed and lasts
about an hour, and `expiresAt` is epoch milliseconds. Mint a new one near
expiry rather than reloading a running app. A Rust client needs a webview for
this. The browser PoC can use a plain iframe.

### Bridge calls

The app speaks JSON-RPC over `postMessage`. The host forwards these to the
server. In every call, `threadId` is the item's own thread (`sourceThreadId`
for an inherited row) and `itemId` is the item id.

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `mcpApps.toolInfo` | `orchestration:read` | `{threadId, itemId, name}` | `{callable, readOnly, title?, tool?}` |
| `mcpApps.callTool` | `orchestration:operate` | `{threadId, itemId, name, arguments: object}` | `{content: unknown[], structuredContent?, isError?, _meta?}` |
| `mcpApps.readResource` | `orchestration:read` | `{threadId, itemId, uri}` | `{contents: unknown[]}` |
| `mcpApps.updateModelContext` | `orchestration:operate` | `{threadId, itemId, conversationThreadId, content?, structuredContent?}` | void |

The server routes every call to the server named in the reference, so an app
only reaches its own MCP server. It refuses tools the app may not call. Check
`toolInfo.callable` before showing a control.

`updateModelContext` replaces what this app tells the agent on its next turn.
No content clears it. `conversationThreadId` is the thread the user is looking
at. It must be the item's thread or a fork of it. Only text blocks and
`structuredContent` are accepted.

`McpAppRequestError {threadId, reason}` reasons: `not-an-app`,
`provider-unsupported`, `session-stopped` ("send a message in the thread that
created it to use the app again"), `tool-not-callable`, `unsupported-content`,
`request-failed`.

The web host also handles `openLink` (only from a click in the frame),
`sendMessage` (asks the user first, then sends as a normal message), display
mode changes, downloads, and resize. Strip `null` fields from tool results
before you pass them to the app. Codex sends `_meta: null` and the MCP Apps SDK
rejects it. Default frame height is 320 px, max 2000, max HTML 5 MB.

Agent HTML from the `html_render` tool is a separate thing. It uses
`output.htmlRender` and the same asset route, and it has no tool bridge.

## 6. Preview, server browser and devices

Brief on purpose. Most of this is built for the desktop app.

### Preview tabs

Contract: `preview.ts`. Server: `apps/server/src/preview/Manager.ts`,
`ServerBrowser*.ts`. Web: `apps/web/src/components/preview/`.

A preview tab is an embedded browser attached to a thread. There are two
runtimes, in `PreviewSessionSnapshot.runtime`:

- `desktop`: an Electron webview inside the desktop app. Only the desktop app
  can render it. A third-party client can list these tabs but can't show them.
- `server`: a headless Chromium on the environment host, streamed as JPEG
  frames. Any client can show it. It needs `capabilities.serverBrowser`.

| Tag | Scope | Input | Output | Notes |
| --- | --- | --- | --- | --- |
| `preview.list` | `orchestration:read` | `{threadId}` | `{sessions: Snapshot[], serverEpoch, revision}` | |
| `subscribePreviewEvents` | `orchestration:read` | `{}` | stream of `opened`, `navigated`, `resized`, `failed`, `closed` | Each has `threadId, tabId, createdAt, serverEpoch, revision`. |
| `preview.open` | `preview:operate` | `{threadId, url?, viewport?, profileId?, runtime?, reveal?}` | Snapshot | Pass `runtime: "server"` from a non-desktop client. |
| `preview.navigate` | `preview:operate` | `{threadId, tabId, url, resolvedTitle?}` | Snapshot | |
| `preview.resize` | `preview:operate` | `{threadId, tabId, viewport}` | Snapshot | `{_tag:"fill"}`, `freeform` or `preset`. 240 to 3840 px. |
| `preview.adjust` | `preview:operate` | `{threadId, tabId, colorScheme?, zoomFactor?, hardReload?, clear?}` | Snapshot | |
| `preview.refresh` | `preview:operate` | `{threadId, tabId}` | void | |
| `preview.close` | `preview:operate` | `{threadId, tabId?}` | void | No `tabId` closes all of the thread's tabs. |
| `preview.reportStatus` | `preview:operate` | `{threadId, tabId, navStatus, canGoBack, canGoForward}` | void | Desktop only. The webview host reports its state. |
| `preview.reportProfiles` | `preview:operate` | `{profiles, defaultProfileId, recordingFrameRate?}` | void | The web reports the user's browser profiles so server tabs can use them. |
| `preview.clearProfile` | `preview:operate` | `{profileId}` | void | Deletes a profile's stored data. |
| `subscribeDiscoveredLocalServers` | `orchestration:read` | `{configuredUrls?}` | stream of `{servers: [{host, port, url, processName, pid, terminal}], scannedAt}` | Dev servers listening on the host, for "open in preview". |

Snapshot fields: `threadId, tabId, navStatus` (`Idle`, `Loading`, `Success`
or `LoadFailed`, each with `url`, `title`), `canGoBack, canGoForward, viewport?,
colorScheme?, zoomFactor?, profileId?, runtime?, automationOwner?, reveal?,
updatedAt`. Errors: `PreviewSessionLookupError`, `PreviewInvalidUrlError`,
`PreviewControlRequiredError` (another party owns the server tab). Use
`serverEpoch` plus `revision` to drop stale events after a server restart.

To watch a server tab, mint a media ticket
(`POST /api/auth/websocket-ticket/media`) and open
`wss://{host}/api/preview-stream/ws?threadId=&tabId=&wsTicket=` with
`maxWidth`, `maxHeight` and `quality` if you like. Binary messages are JPEG
frames. Send `{"type":"ack"}` after each frame or no more arrive. Text
messages are JSON events (`viewport`, `control`, `popup`, `download`, ...).
Viewing needs `orchestration:read`. Input needs `preview:operate` and no
`interactive=false`. Close code 4404 means the tab is gone, 4503 carries a JSON
`{need, command}` for missing host libraries
(`apps/server/src/preview/ServerBrowserStream.ts`). The input message format
lives in `ServerBrowser.ts`. I didn't document it.

Agent-driven browsing (`preview_click`, `preview_snapshot` and so on) goes over
a private desktop channel (`previewAutomation.ts`,
`apps/server/src/preview/DesktopBrowserChannel.ts`), or straight to the server
browser. It isn't client RPC. `browserImport.ts` (import cookies from Chrome,
Safari and others) and `previewCapture.ts` are desktop IPC only.

### Devices

Contract: `device.ts`. Server: `apps/server/src/device/`. Web:
`apps/web/src/components/device/`.

iOS simulators and Android emulators on the environment host, on an SSH host,
or on Expo EAS cloud devices, shown in a thread's side panel. The devices run
where the server is, not on the client. A Mac host is needed for iOS. The
viewer works from any client.

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `subscribeDeviceState` | `orchestration:read` | `{}` | stream of `DeviceServiceState` |
| `device.list` | `orchestration:read`, or `orchestration:operate` with `retryHostId`/`updateTool` | `{updateTool?, inspectOnly?, retryHostId?}` | `DeviceServiceState` |
| `device.detail` | `orchestration:read` | `{hostId?, deviceId}` | `{hostId, deviceId, settings, foregroundApp, readAt}` |
| `device.open` | `orchestration:operate` | `{threadId, hostId?, deviceId, platform, boot?}` | `{threadId, hostId, deviceId, platform, openedAt}` |
| `device.close` | `orchestration:operate` | `{threadId, hostId?, deviceId?, shutdown?}` | void |
| `device.shutdown` | `orchestration:operate` | `{hostId?, deviceId, platform}` | void |
| `device.action` | `orchestration:operate` | tagged by `type` | `DeviceDetail` |
| `device.configure` | `settings:write` | `{enabled?, agentAccessEnabled?, onboardingCompleted?, ...}` | `DeviceServiceState` |
| `device.testHost` | `settings:write` | SSH host config | host summary |
| `device.easAccount` | `settings:write` | `{hostId?, accessToken?}` | `{signedInAs, credentialSource, apps}` |

`device.action` types: `setAppearance`, `setTextSize`, `setToggle`,
`setLiquidGlass`, `setColorFilter`, `setOrientation`, `setLocation`,
`clearLocation`, `setPermission`, `openUrl`, `launchApp`, `terminateApp`,
`shake`, `sendPush`.

`DeviceServiceState` has `hostStatus`, `hostStatuses`, `hosts`, `easHosts?`,
`devices`, `sessions`, `onboardingCompleted`, `agentAccessEnabled`,
`hubBasePath` (`/api/device-hub` live) and `revision`. The screen stream goes
through `{hubBasePath}/...` on the environment with `wsTicket` from a media
ticket. Reading needs `orchestration:read`. Touch input and stream settings
need `orchestration:operate` (`apps/server/src/device/DeviceHubProxy.ts:174`).
The stream protocol belongs to Expo's device hub and lives in
`packages/client-runtime/src/device/stream.ts`.

## 7. Odds and ends

### Agent session import

Contract: `agentSessions.ts`. Web: onboarding and Settings, "import from Claude
Code / Codex".

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `agentSessions.scan` | `orchestration:read` | `{}` | `{candidates, scannedAt, truncated?}` |
| `agentSessions.import` | `orchestration:operate` | `{projectId, expectedWorkspaceRoot?}` | `{importedCount, skippedCount}` |

A candidate is `{path, title, projectId?, sources: ("claudeAgent"|"codex")[],
threadCount, lastActiveAt, alreadyImported, git?: {remoteKey, repository}}`.
Scan reads the host's `~/.claude` and `~/.codex` session files and is
read-only. My server returned 19. Import only takes projects that already
exist. Create the project first (see `projects-files-vcs-terminal.md`). Errors:
`AgentSessionScanError`, `AgentSessionImportProjectNotFoundError`,
`AgentSessionImportProjectChangedError` (scan again).

### Attachments

`attachments.createUploadUrl` and the upload POST are in `t3-connect.md`. Files
work the same way as images with `type: "file"`, any `mimeType` and up to 50 MB,
when `capabilities.fileAttachments` is present.

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `attachments.delete` | `orchestration:operate` | `{attachmentId}` | void |
| `assets.persistChatAttachments` | `orchestration:operate` | `{threadId, messageId, attachments: [{type:"image", name, mimeType, sizeBytes, dataUrl, id?}]}` | `{attachments: ChatAttachment[]}` |

`attachments.delete` only removes an upload that hasn't been sent yet, for
when the user removes it from the composer. For anything else it quietly does
nothing (`apps/server/src/assets/AttachmentUpload.ts:224`). Pending uploads
also expire. A send that fails with `AssetAttachmentNotFoundError` means
"upload again". `persistChatAttachments` is the older data-URL path. Prefer
signed uploads.

### Asset URLs

| Tag | Scope | Input | Output |
| --- | --- | --- | --- |
| `assets.createUrl` | `filesystem:read` for `workspace-file`, `media-file`, `draft-workspace-file`; otherwise `orchestration:read` | `{resource}` | `{relativeUrl, expiresAt, sourcePath?, imageDimensions?, fileRevision?}` |

`resource` is tagged by `_tag`:

| `_tag` | Fields | For |
| --- | --- | --- |
| `attachment` | `attachmentId, fileName?, mimeType?, disposition?, renderIntent?` | Images and files on messages, MCP app HTML |
| `tool-output-image` | `threadId, itemId, index` | Screenshots a tool returned |
| `workspace-file` | `threadId, path` | A project file to preview |
| `media-file` | `threadId, path` | Images, video, audio, HTML, PDF the agent referenced |
| `draft-workspace-file` | `cwd, path` | Same, before a thread exists |
| `project-favicon` | `cwd, path?` | Project icon |
| `native-app-icon` | `app` | Icons for desktop apps in tool rows |
| `github-media` | `cwd, url` | Private images in a PR body, fetched with the server's credential |

`relativeUrl` is `/api/assets/{token}/{fileName}`. The token carries the auth,
so the GET needs no header, which suits `<img>` tags. `expiresAt` is epoch ms,
about an hour out. `fileRevision` only changes when the content does, so cache
by it rather than by URL.

### T3 Connect relay client

| Tag | Kind | Scope | Output |
| --- | --- | --- | --- |
| `cloud.getRelayClientStatus` | call | `relay:read` | `{status:"available", executablePath, source, version}`, `{status:"missing", version}` or `{status:"unsupported", platform, arch, version}` |
| `cloud.installRelayClient` | stream | `relay:write` | `{type:"progress", stage}` then `{type:"complete", status}` |

This is the `cloudflared` binary the environment needs for a Connect tunnel. It
belongs in Settings, Connections. Linking and unlinking are HTTP under
`/api/connect/*` with `relay:read`/`relay:write` (`apps/server/src/cloud/http.ts`).

### Lifecycle and policy streams

| Tag | Scope | Values | Use |
| --- | --- | --- | --- |
| `subscribeServerLifecycle` | `orchestration:read` | `{version:1, sequence, type, payload}` | `welcome` (`environment, cwd, projectName, bootstrap*`), then `ready` (`at, environment, updateOutcome?`), sometimes `legacyThreadMigration` (`status, totalThreadCount`). Live server sent `welcome` then `ready`. A `ready` with `updateOutcome` follows a self-update. |
| `subscribeAuthAccess` | `access:read` | `{version:1, revision, type, payload}` | `snapshot` (`pairingLinks, clientSessions`), then `pairingLinkUpserted`, `pairingLinkRemoved`, `clientUpserted`, `clientRemoved`. The Connections settings list. |
| `subscribeBackgroundPolicy` | `orchestration:read` | `{hostPower, leases, activeForegroundLeaseCount, activeScopeKeys, shouldRunOpportunisticWork, updatedAt}` | Whether the server should do background refreshes. |
| `subscribeResourceTelemetry` | `diagnostics:read` | `{readAt, sampleIntervalMs, processes, groups, power, speedLimitPercent, attribution, health}` | Diagnostics page. |

`subscribeBackgroundPolicy` reflects leases that clients report with
`server.reportClientActivity` (`orchestration:read`, schema in
`background.ts`). A client that tells the server it's visible and focused gets
fresher provider and VCS status. A watch app in the background should report
`visible: false` or nothing.

`remoteAccess.ts` (`AdvertisedEndpoint`) and `applicationEvent.ts` (stored
project events) have no RPC of their own. The first is desktop IPC. The second
is the server's event store.

## 8. Auth scopes

Scopes live in `packages/contracts/src/auth.ts:30`. The server's map from
method to scope is `apps/server/src/auth/RpcAuthorization.ts`.
`packages/contracts/src/clientRpcPermissions.ts` is a subset the official
clients check before calling, to hide buttons. The server checks every call
regardless, so the client list is for UI only.

### Every scope

| Scope | Grants | Legacy parent |
| --- | --- | --- |
| `orchestration:read` | Read threads, projects, config, PRs, tasks, previews, devices. Most subscriptions. | |
| `orchestration:operate` | Dispatch commands, launch threads, uploads, secret answers, task writes and webhook logs, MCP app calls, device control, agent session import, review guide generation, `stopWorkflow`. | |
| `settings:write` | `server.updateSettings` (depends on the patch), keybindings, device host config, storage cleanup, OTLP check. | operate |
| `providers:manage` | Provider sign-in, install, update, ACP registry changes. | operate |
| `environment:maintain` | Server self-update, desktop update commit, `signalProcess`, host power reports. | operate |
| `preview:operate` | Drive preview tabs and the server browser. | operate |
| `diagnostics:read` | Traces, process diagnostics, resource telemetry, usage summary. | read |
| `terminal:read` | Observe terminals. | `terminal:operate` |
| `terminal:operate` | Open and type into terminals, a shell as the server's user. | |
| `source-control:write` | PR writes, clone, publish, pull, worktrees, refs, stacked git actions. | operate |
| `filesystem:read` | Read and search project files, browse the filesystem, diff previews, workspace asset URLs. | read |
| `filesystem:write` | `projects.writeFile`. | operate |
| `review:write` | Nothing. Kept so old credentials still decode. Can't be granted. | |
| `access:read` | `subscribeAuthAccess`, list pairing links and clients over HTTP. | |
| `access:write` | Mint pairing tokens, revoke links and clients, `guestShare.create`. | |
| `relay:read` | Relay client status, Connect link state, reachability. | |
| `relay:write` | Install the relay client, link, unlink, relay config. | |

Old servers only know the legacy scopes (`orchestration:*`, `terminal:operate`,
`access:*`, `relay:*`). Newer servers send both `scopes`, filtered to the legacy
set, and `permissions`, the real list. Read `permissions`. Only if it's missing,
treat a fine scope as granted when its legacy parent is in `scopes`
(`sessionGrantsScope`, `auth.ts:128`). Denials carry both: `requiredScope` is
the legacy parent and `requiredPermission` is the real one. Live,
`subscribeResourceTelemetry` with a read-only token failed with
`requiredScope: "orchestration:read"`, `requiredPermission: "diagnostics:read"`.
Key your UI on `requiredPermission`.

### Per-method requirements for this document

| Methods | Scope |
| --- | --- |
| All PR reads in section 1, `subscribeRefreshes`, `invalidate`, `reportState`, `routing*`, HTTP diff | `orchestration:read` |
| `pullRequests.runAction`, `update`, `comment`, `updateComment`, `submitReview`, `replyToThread`, `setThreadResolution`, `setReaction`, `setFilesViewed`, `requestReviewers`, `setLabels` | `source-control:write` |
| `pullRequests.generateReviewGuide`, PR link commands via `dispatchCommand` | `orchestration:operate` |
| `scheduledTasks.list`, `subscribe` | `orchestration:read` (no `webhook` field without operate) |
| `scheduledTasks.upsert`, `setEnabled`, `delete`, `runNow`, `rotateWebhookToken`, `listWebhookDeliveries`, `getWebhookDelivery` | `orchestration:operate` |
| `guestShare.create` | `access:write` |
| `secrets.answerRequest` | `orchestration:operate` |
| `mcpApps.toolInfo`, `readResource` | `orchestration:read` |
| `mcpApps.callTool`, `updateModelContext` | `orchestration:operate` |
| `preview.list`, `subscribePreviewEvents`, `subscribeDiscoveredLocalServers`, preview stream viewing and downloads | `orchestration:read` |
| Other `preview.*`, preview stream input and uploads | `preview:operate` |
| `device.list` (plain), `device.detail`, `subscribeDeviceState`, device hub viewing | `orchestration:read` |
| `device.list` with `retryHostId` or `updateTool`, `device.open`, `close`, `shutdown`, `action`, device hub control | `orchestration:operate` |
| `device.configure`, `testHost`, `easAccount` | `settings:write` |
| `agentSessions.scan` | `orchestration:read` |
| `agentSessions.import`, `attachments.createUploadUrl`, `attachments.delete`, `assets.persistChatAttachments` | `orchestration:operate` |
| `assets.createUrl` | `filesystem:read` or `orchestration:read`, by resource |
| `cloud.getRelayClientStatus` / `cloud.installRelayClient` | `relay:read` / `relay:write` |
| `subscribeAuthAccess` | `access:read` |
| `subscribeServerLifecycle`, `subscribeBackgroundPolicy` | `orchestration:read` |
| `subscribeResourceTelemetry` | `diagnostics:read` |

A few methods outside this document need two scopes or depend on the payload.
`git.preparePullRequestThread` in worktree mode with a `threadId` needs
`source-control:write` and `orchestration:operate`. `git.runStackedAction` with
`create_pr` or `commit_push_pr` and a `threadId` needs `source-control:write`
and `orchestration:read`. `server.retryResourceTelemetry` needs
`environment:maintain` and `diagnostics:read`. `server.updateSettings` and
`projects.mutate` depend on what they change (see the other docs).

### What a client gets by default

| How the session was made | Scopes |
| --- | --- |
| `t3 pair` or `t3 auth pairing create` with no `--scope` | `AuthStandardClientScopes`: all 16 grantable scopes, everything except `review:write` |
| T3 Connect relay connect (`CloudLink.ts:1597`) | Same full set, on a 2-minute key-bound credential |
| Desktop bootstrap, startup pairing link | Same full set |
| `POST /api/auth/pairing-token` from an existing session | Defaults to the full set, but each scope must be one the caller holds, and the caller needs `access:write` (`apps/server/src/auth/http.ts:449`) |
| MCP client OAuth | `read-only`: read. `full-access`: read, operate, settings, source-control. Others: read and operate. These tokens only work on `/mcp`. |

The `scope` field on `/oauth/token` can only narrow what the pairing granted.
So by default a paired or Connect client can do everything, including
`terminal:operate` and `access:write`, which means it can open a shell and mint
more pairing links. For a watch or a phone I'd pair with
`--scope orchestration:read --scope orchestration:operate` and add
`source-control:write` only if you build PR actions.

## 9. What a thread view has to handle

Things that show up inside a thread and need more than rendering messages. The
item and event shapes are in `orchestration.md`.

- **PR links.** `thread.pullRequests` (hide `stack-dismissed`), the legacy
  `linkedPullRequest`, and `branchPullRequest`. Each link's `snapshot` is
  enough for a badge. `link.watch` means the agent is watching CI. Apply
  `thread.pull-request-link-updated` events by key. Refetch live PR data when
  `pullRequests.subscribeRefreshes` ticks.
- **Secret request rows.** `secret_request` turn items. Answerable only when
  `secretStatus === "pending"` and the row is `local`. Answer with
  `secrets.answerRequest`. Keep the value out of every log.
- **MCP app rows.** `dynamic_tool` items with a valid `output.t3McpApp`. Load
  the HTML through `assets.createUrl`, fetch the full item for the tool input
  and result, and forward the app's calls to `mcpApps.*` using the item's own
  `threadId` and the viewed thread as `conversationThreadId`. Without a webview,
  show the tool row and a "open in T3 Code" note.
- **HTML renders.** `dynamic_tool` items with `output.htmlRender`. Same asset
  route, no bridge.
- **Guest messages.** `thread.guestShare.messages` with `status: "pending"`
  need Send (`message.dispatch` with `guestMessageId`) and Dismiss. Show
  `guestName` and `expiresAt` somewhere.
- **Scheduled task messages.** User messages with `scheduledTaskId`. Look up
  the title in `scheduledTasks.list` and label the message.
- **Preview and device sessions.** `preview.list {threadId}` and
  `DeviceServiceState.sessions` filtered by `threadId` tell you a thread has a
  browser tab or phone open. Only `runtime: "server"` tabs can be shown outside
  the desktop app.
- **Attachment images.** `attachment` and `tool-output-image` assets via
  `assets.createUrl`. Cache by `fileRevision`.
