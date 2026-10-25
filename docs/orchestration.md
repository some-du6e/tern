# T3 Code orchestration API reference

This is the full orchestration surface a third-party client can call over the
WebSocket: every `orchestration.dispatchCommand` command, every other
`orchestration.*` RPC, the stream items, the scopes, and the new ID table.
It comes from upstream `pingdotgg/t3code` at `80455376` (2026-10-24), checked
against the local server, which runs `0.1.3-nightly.20261023.3592`. Paths are
relative to the upstream repo.

Read [t3-connect.md](t3-connect.md) first. It covers pairing, DPoP, tickets,
the Effect RPC framing (`Request`/`Chunk`/`Ack`/`Exit`), turn items, bounded
snapshots, history paging, `getTurnItem` and image upload. This document
doesn't repeat those.

The schemas live in:

| File | What's in it |
| --- | --- |
| `packages/contracts/src/orchestrationV2.ts` | Commands (line 2677), RPC names (3240), inputs, stream items, domain events (1677), shells (1884) |
| `packages/contracts/src/orchestrationV2IdTable.ts` | The ID table codec from `80455376` |
| `packages/contracts/src/threadSearch.ts` | `searchThreads` input and result |
| `packages/contracts/src/checkpointDiff.ts` | `getTurnDiff` and `getFullThreadDiff` |
| `packages/contracts/src/rpc.ts:1626` | The `Rpc.make` definitions: payload, success, error, stream or not |
| `packages/contracts/src/providerPolicy.ts` | Runtime modes, interaction modes, approval decisions |
| `packages/client-runtime/src/operations/commands.ts` | How the official clients build each command. The best cross-reference. |
| `apps/server/src/ws.ts:1891` | Server handlers |
| `apps/server/src/auth/RpcAuthorization.ts:42` | Scope per method |

## Conventions

**Command IDs.** Every command carries a `commandId`. The official client uses
a random UUIDv4 (`commands.ts:264`). The server keeps a receipt per
`commandId`. Sending the same command again with the same `commandId` returns
the original `{sequence}` instead of running it twice, so reuse the ID when
you retry after a dropped socket. Reusing a `commandId` for a different thread
fails with "Command ... was already handled for thread ...".

**Entity IDs.** `threadId`, `messageId` and fork `targetThreadId` are minted
by the client, also as UUIDv4 (`apps/web/src/lib/utils.ts:45`). Run IDs,
request IDs, node IDs, plan IDs and checkpoint IDs come from the server; read
them from the projection or shell.

**Provenance.** `thread.create`, `message.dispatch`, `thread.fork`,
`thread.merge_back` and `delegated_task.request` require `createdBy` and
`creationSource`. Send `"createdBy": "user"` and a `creationSource` of `"web"`
or `"mobile"`. The server overwrites `createdBy` with `"user"` for anything
arriving over the socket (`ThreadManagementService.ts:68`). The enums are
`createdBy: "user" | "agent" | "system"` and
`creationSource: "web" | "mobile" | "mcp" | "provider" | "server"`.

**Result.** `dispatchCommand` returns `{"sequence": n}`, the event-store
sequence of the last event it wrote. You don't need to do anything with it.
The effects show up on your shell and thread streams.

**Errors.** A failed command comes back as an `Exit` with `Fail` and
`OrchestrationV2DispatchCommandError`:

```json
{"_tag":"OrchestrationV2DispatchCommandError","commandId":"...","commandType":"queued-run.edit",
 "message":"Answer each question before sending.","detail":"Answer each question before sending."}
```

`detail` is present when the server had a user-facing reason. Show `detail`
if it's there, else `message`. A missing scope comes back as
`EnvironmentAuthorizationError` with `requiredScope`.

**Decoding is lenient on input.** Effect's struct decoding drops unknown keys,
so sending an optional field an old server doesn't know is harmless. The local
nightly ignored `acceptCompactIds` and `acceptDedupedBodies` without error. A
wrong value for a known field fails the whole request.

**Capabilities.** Gate features on what the server advertises, not on your
version. There are two places:

- `server.getConfig` → `environment.capabilities` (also in
  `GET /.well-known/t3/environment`). Relevant flags: `serverResolvedCommandContext`,
  `threadStop`, `threadSnooze`, `threadPinning`, `threadPinReorder`,
  `threadActiveReorder`, `threadAutoSettleOptOut`, `threadTitleRegeneration`,
  `threadVisitedTracking`, `threadSettlement`, `threadPullRequestLinking`,
  `threadPullRequestWatch`, `inlineMessageContext`, `questionAttachments`,
  `commandOutputStreaming`. Schema in `packages/contracts/src/environment.ts:72`.
- `server.getConfig` top level: `shellResumeCompletionMarker`,
  `shellThreadFamilies`, `threadResumeCompletionMarker`,
  `threadSnapshotPagination`, `threadFind`, `threadFindProgressive`
  (`packages/contracts/src/server.ts:582`).

## Shared types

### `modelSelection`

```json
{"instanceId": "claudeAgent", "model": "claude-opus-5",
 "options": [{"id": "effort", "value": "high"}, {"id": "contextWindow", "value": "200k"}]}
```

| Field | Type | Notes |
| --- | --- | --- |
| `instanceId` | string, required | A provider instance from `server.getConfig` → `providers[].instanceId`. Older payloads used `provider`; the server still accepts it on decode. |
| `model` | string, required | A model `slug` from that provider's `models[]`. |
| `options` | array, optional | `[{id, value}]`, `value` a non-empty string or a boolean. The legal ids and values come from `models[].capabilities.optionDescriptors[]` (`type: "select"` with `options[].id`, or `type: "boolean"`). The server also accepts the legacy object form `{"effort": "high"}`. |

That sample is a real thread on the local server.

### Modes

| Name | Values | Default |
| --- | --- | --- |
| `runtimeMode` | `approval-required`, `auto-accept-edits`, `auto`, `full-access` | `full-access` |
| `interactionMode` | `default`, `plan` | `default` |

`plan` is plan mode: the agent proposes a plan (a `proposed_plan` turn item)
instead of editing.

### Attachments

`attachments` is an array of `ChatAttachment`
(`packages/contracts/src/chatAttachment.ts:148`):

```json
{"type": "image", "id": "<attachmentId>", "name": "shot.png", "mimeType": "image/png", "sizeBytes": 12345}
{"type": "file",  "id": "<attachmentId>", "name": "notes.txt", "mimeType": "text/plain", "sizeBytes": 812}
```

Upload first; see t3-connect.md. `id` matches `^[a-z0-9_-]+$`.

### `context`

Optional structured context on a user message: `{"version": 1, "records": [...]}`
(`packages/contracts/src/composerContext.ts:270`). Records are mentions,
skills, terminal snippets, review comments and the like. A small client can
leave it out. Servers without `inlineMessageContext` expect that context as
text in the message instead.

### `workspaceStrategy`

Where a new thread's run happens:

```json
{"type": "root", "branch": "main"}                                      // branch optional
{"type": "existing_worktree", "worktreePath": "/abs/path", "branch": "x"} // branch optional
{"type": "worktree", "baseRef": "main", "branch": "feature/x", "startFromOrigin": true} // branch, startFromOrigin optional
```

### `sourcePoint` (fork and merge)

```json
{"type": "latest_stable"}
{"type": "run", "runId": "<runId>"}
{"type": "checkpoint", "checkpointId": "<checkpointId>"}
```

## Commands for `orchestration.dispatchCommand`

The payload is the command itself, tagged by `type`. All commands need
`orchestration:operate`. In the tables, "req" means required and "opt" means
the key can be left out. `X | null` means the key is required but may be
`null`.

### Thread lifecycle

#### Starting a thread: `orchestration.launchThread` (not a command)

This is how the web app starts a thread. When you type into a new-thread draft
and press send, `startThreadTurn` sees the draft has no server thread yet and
calls `orchestration.launchThread` with the first message
(`commands.ts:672`). One call creates the thread, prepares the workspace and
starts the run. Use this for "new thread with a model".

```json
{
  "commandId": "6b0f...",
  "threadId": "a244...",
  "creationSource": "web",
  "projectId": "<projectId>",
  "title": "Fix the flaky test",
  "generateTitle": true,
  "modelSelection": {"instanceId": "codex", "model": "gpt-5.5", "options": [{"id": "reasoningEffort", "value": "high"}]},
  "runtimeMode": "full-access",
  "interactionMode": "default",
  "workspaceStrategy": {"type": "root"},
  "initialMessage": {"messageId": "9d1c...", "text": "The login test fails on CI", "attachments": []}
}
```

| Field | | Notes |
| --- | --- | --- |
| `commandId` | req | |
| `projectId` | req | From the shell's `projects[].id`. |
| `title` | req | Non-empty. With `generateTitle: true` the server replaces it with a generated title after the first message. The web sends the draft's title seed here. |
| `modelSelection`, `runtimeMode`, `interactionMode`, `workspaceStrategy` | req | See shared types. |
| `threadId` | opt | Pick it yourself so you can subscribe before the call returns. The server makes one if absent. |
| `reuseExistingThread` | opt | `true` when the thread already exists but never started (the web sets this when it only needs a worktree prepared). |
| `creationSource` | opt | Defaults to `"web"`. |
| `generateTitle` | opt | |
| `initialMessage` | opt | `{messageId?, text, context?, attachments}`. `attachments` is required inside it, so send `[]`. Without `initialMessage` the thread is created idle. |

Returns `{threadId, projection, resumed}`. `projection` is a full
`OrchestrationV2ThreadProjection`. `resumed: true` means the same `commandId`
had already launched this thread and you got the existing one back. Errors:
`OrchestrationV2ThreadLaunchError {commandId, projectId, message}`, with the
attachment or disabled-provider reason in `message`.

#### `thread.create`

Creates an idle thread with no message. The web uses it for "Implement plan in
a new thread" (`apps/web/src/components/chat/useProposedPlanSend.ts`), followed
by a `message.dispatch` to the new thread.

```json
{"type": "thread.create", "commandId": "...", "createdBy": "user", "creationSource": "web",
 "threadId": "<new uuid>", "projectId": "...", "title": "Implement: auth refactor",
 "modelSelection": {...}, "runtimeMode": "full-access", "interactionMode": "default",
 "branch": null, "worktreePath": null}
```

All shown fields are required; `branch` and `worktreePath` are `string | null`.
There's also an optional `importedNativeThread` used by provider-session
imports. A client won't need it.

#### Rename and other metadata: `thread.metadata.update`

The sidebar's rename sends `{title}`. "Regenerate title" sends
`{regenerateTitle: true}`. Every field except the three IDs is optional and an
absent field is left alone.

```json
{"type": "thread.metadata.update", "commandId": "...", "threadId": "...", "title": "New name"}
```

| Field | Type | Notes |
| --- | --- | --- |
| `title` | string | |
| `regenerateTitle` | boolean | Gate on `threadTitleRegeneration`. Progress shows in the shell's `titleRegeneration`. |
| `branch`, `worktreePath` | string or null | |
| `expectedBranch`, `expectedWorktreePath` | string or null | Guards: as the names suggest, they make the update conditional on the current value. |
| `expectedEmpty` | boolean | Fails if any message or run exists. |
| `limitRecovery` | object or null | `{runId, resetAt, autoResume?, snooze?}`, at least one of `autoResume`/`snooze`. The "resume when the usage limit resets" toggle. |
| `linkedPullRequest` | object or null | Link or unlink a PR. Prefer the `thread.pull-request.*` commands below. |

#### Archive, unarchive, delete

```json
{"type": "thread.archive",   "commandId": "...", "threadId": "..."}
{"type": "thread.unarchive", "commandId": "...", "threadId": "..."}
{"type": "thread.delete",    "commandId": "...", "threadId": "..."}
```

Sidebar context menu and the archive view. Archiving sends a `thread.removed`
(location `active`) on the shell and a `thread.updated` on the archived shell.

#### Pin, unpin, reorder

```json
{"type": "thread.pin", "commandId": "...", "threadId": "...", "orderKey": "a0"}   // orderKey optional
{"type": "thread.unpin", "commandId": "...", "threadId": "..."}
{"type": "thread.pin.reorder", "commandId": "...", "threadId": "...", "orderKey": "a0V"}
{"type": "thread.active.reorder", "commandId": "...", "threadId": "...", "orderKey": "a1"}
```

The web pins from the context menu and reorders by drag. `orderKey` is a
string that sorts ascending; the client picks a key between its neighbours
(`pinOrderKeyBetween`, `packages/client-runtime/src/state/threadSort.ts:199`).
Read the current keys from the shell's `pinOrderKey` and `activeOrderKey`.
Older servers can't decode `orderKey` on `thread.pin`, so the web sends it only
when `threadPinReorder` is advertised. `thread.active.reorder` is drag order in
the active (unpinned) list and needs `threadActiveReorder`.

#### Settle, snooze, read state

"Settled" is T3's done pile. Snoozing hides a thread until a time.

```json
{"type": "thread.settle",   "commandId": "...", "threadId": "...", "settledAt": "2026-10-24T18:00:00Z"} // settledAt optional
{"type": "thread.unsettle", "commandId": "...", "threadId": "...", "reason": "user"}
{"type": "thread.snooze",   "commandId": "...", "threadId": "...", "snoozedUntil": "2026-10-25T09:00:00.000Z"}
{"type": "thread.unsnooze", "commandId": "...", "threadId": "...", "reason": "user"}
{"type": "thread.auto-settle.set", "commandId": "...", "threadId": "...", "enabled": false}
{"type": "thread.visit",    "commandId": "...", "threadId": "...", "visitedAt": "2026-10-24T18:00:00.000Z", "opened": true, "searchQuery": "login"}
{"type": "thread.mark-unread", "commandId": "...", "threadId": "..."}
```

- `reason` is the literal `"user"`, required.
- `thread.auto-settle.set {enabled: false}` opts one thread out of automatic
  settling (`threadAutoSettleOptOut`).
- `thread.visit` moves the read watermark. The web sends it when you open a
  thread (`opened: true`) and as new activity arrives on an open thread. Add
  `searchQuery` (max 200 chars) when the user got there from search; it feeds
  ranking. `opened` and `searchQuery` are optional.
- `thread.auto-settle` (with `snapshotAt`) is what the client's auto-settle
  timer sends. It is rejected if the thread changed since `snapshotAt`. A
  third-party client doesn't need it.

#### Fork and merge back

```json
{"type": "thread.fork", "commandId": "...", "createdBy": "user", "creationSource": "web",
 "sourceThreadId": "<this thread>", "targetThreadId": "<new uuid>",
 "sourcePoint": {"type": "run", "runId": "<runId>"}, "title": "Try the other approach"}
```

`title` and `createdAt` are optional. The web's "Fork from here" on a turn
sends `sourcePoint: {type: "run", runId}`. Subscribe to `targetThreadId` to
see the fork.

```json
{"type": "thread.merge_back", "commandId": "...", "createdBy": "user", "creationSource": "web",
 "sourceThreadId": "<fork>", "targetThreadId": "<parent>",
 "sourcePoint": {"type": "run", "runId": "<runId in the fork>"}}
```

"Merge back" in a fork carries the fork's context into the parent.

#### Stop everything: `thread.stop`

```json
{"type": "thread.stop", "commandId": "...", "threadId": "...", "reason": "optional", "stopDelegatedTasks": true}
```

The Stop button sends this when the server advertises `threadStop` and the
client didn't name a run (`commands.ts:860`). It stops the active run,
background commands and, unless `stopDelegatedTasks: false`, delegated child
tasks. Prefer it over `run.interrupt` when you just want "stop".

### Messages: `message.dispatch`

Sending from the composer.

```json
{
  "type": "message.dispatch",
  "commandId": "...",
  "createdBy": "user",
  "creationSource": "web",
  "threadId": "...",
  "messageId": "<new uuid>",
  "text": "Now add a test",
  "attachments": [],
  "deliveryIntent": "auto",
  "dispatchMode": {"type": "start_immediately"}
}
```

Required: `type`, `commandId`, `createdBy`, `creationSource`, `threadId`,
`messageId`, `text`, `attachments` (send `[]`) and `dispatchMode`.

`dispatchMode` is one of:

| `dispatchMode` | Meaning |
| --- | --- |
| `{"type": "start_immediately"}` | Start a run now. Fails if a run is active and no `deliveryIntent` tells the server how to resolve it. |
| `{"type": "queue_after_active"}` | Queue behind the active run. Shows up in the queue. |
| `{"type": "steer_active", "targetRunId": "<runId>"}` | Inject into the running turn. Only if the provider session has `capabilities.turns.supportsActiveSteering`. |
| `{"type": "restart_active", "targetRunId": "<runId>"}` | Interrupt the running turn and restart it with this message (`supportsSteeringByInterruptRestart`). |
| `{"type": "defer_start", "workspaceStrategy"?: {...}}` | Record the message without starting. Used by the launch flow. |

**Let the server pick.** If `environment.capabilities.serverResolvedCommandContext`
is true (it is on the local nightly), do what the web does
(`commands.ts:761`): send `dispatchMode: {type: "start_immediately"}` plus
`deliveryIntent` of `"auto"`, `"steer"` or `"restart"`, or send
`{type: "queue_after_active"}` with no `deliveryIntent` to queue. The server
then resolves against the live projection (`CommandPolicy.ts:146`): no active
run means start; `steer` and `restart` target the active run; `auto` queues if
the provider turn hasn't started yet, then steers if the provider can, else
queues, else restarts. `/compact` and `/logout` runs can't be steered, so
those follow-ups get queued. This saves you from tracking `targetRunId` and
provider capabilities yourself, which is racy.

Without that capability, read the active run (status `preparing`, `starting`,
`running` or `waiting`) from the projection and pick the mode yourself.

Optional fields worth knowing:

| Field | Notes |
| --- | --- |
| `modelSelection` | Use this model for this run. The web sends the composer's current selection with every send. |
| `titleSeed` | Seed for title generation. The web sends it only on a thread's first message. |
| `context` | See shared types. |
| `sourcePlanRef` | `{threadId, planId}`. Marks the message as implementing a proposed plan; see below. |
| `idleCompaction` | `"beforeCacheExpires"` or `"onReturn"`, only on a `/compact` the client sends on its own. |
| `manualContinuationOfRunId` | "Continue" after a run that stopped early. |
| `guestMessageId` | Send a guest's pending message; the server uses the guest's text and ignores `text`. The web sends `text: ""`, `attachments: []`, `queue_after_active`. Decline with `thread.guest-message.dismiss {guestMessageId}`. |

`notification`, `scheduledTaskId`, `senderThreadId`, `restartContinuationOfRunId`,
`usageLimitContinuationOfRunId`, `usageLimitRecoveryRequestId` and
`delegatedCompletion` are for server and agent-originated messages. Leave them
out.

### The queue

Queued messages are runs with `status: "queued"` in `projection.runs`, ordered
by `queuePosition`. `queueHeld: true` means the queue is paused, which happens
after a Stop (`run.interrupt` with `holdQueue: true`). The text is the run's
user message (`run.userMessageId` → `messages[]`).

```json
{"type": "queued-run.edit", "commandId": "...", "threadId": "...", "runId": "<queued runId>",
 "text": "edited text", "attachments": []}
{"type": "queued-run.cancel",  "commandId": "...", "threadId": "...", "runId": "<queued runId>"}
{"type": "queued-run.reorder", "commandId": "...", "threadId": "...", "runId": "<queued runId>", "beforeRunId": "<other queued runId>"}
{"type": "queued-run.start",   "commandId": "...", "threadId": "...", "runId": "<queued runId>"}
{"type": "queue.resume",       "commandId": "...", "threadId": "..."}
{"type": "queued-message.promote-to-steer", "commandId": "...", "threadId": "...",
 "queuedRunId": "<queued runId>", "targetRunId": "<active runId>"}
```

- `queued-run.edit`: the queue row's edit button. `text` is required.
  `attachments` replaces all attachments; leave it out to keep them. `context`
  is optional.
- `queued-run.reorder`: drag in the queue. `beforeRunId: null` moves it to the
  end.
- `queued-run.start`: "send now" on one queued message.
- `queue.resume`: resume a held queue.
- `queued-message.promote-to-steer`: "steer now" on a queued message; it is
  injected into the active run instead of waiting.

### Runs

```json
{"type": "run.interrupt", "commandId": "...", "threadId": "...", "runId": "<active runId>", "reason": "optional", "holdQueue": true}
```

The web's Stop on a specific run. It always sends `holdQueue: true`, so queued
messages wait instead of starting. `reason` and `holdQueue` are optional.

Related:

```json
{"type": "subagent.stop", "commandId": "...", "threadId": "<parent>", "runId": "<subagent.runId>", "subagentId": "<subagent.id>"}
{"type": "prepared-run.retry", "commandId": "...", "threadId": "...", "runId": "..."}
{"type": "provider-session.detach", "commandId": "...", "threadId": "...", "providerSessionId": "...", "reason": "client-requested"}
```

- `subagent.stop`: the stop button on a subagent row.
- `prepared-run.retry`: "Retry" after worktree or setup preparation failed
  (the error code is `workspace_preparation_failed`).
- `provider-session.detach`: "Stop session". The web sends one per entry in
  `projection.providerSessions`, with `commandId` suffixed
  `:detach:<sessionId>`. It detaches the thread from its provider sessions
  (`reason` is free text).

### Model, provider, runtime mode, interaction mode

These change the thread's stored settings for the next run. They don't touch a
run in progress.

```json
{"type": "thread.model-selection.set", "commandId": "...", "threadId": "...",
 "modelSelection": {"instanceId": "claudeAgent", "model": "claude-opus-5", "options": [{"id": "effort", "value": "max"}]}}
{"type": "provider.switch", "commandId": "...", "threadId": "...", "modelSelection": {"instanceId": "codex", "model": "gpt-5.5"}}
{"type": "thread.runtime-mode.set", "commandId": "...", "threadId": "...", "runtimeMode": "approval-required"}
{"type": "thread.interaction-mode.set", "commandId": "...", "threadId": "...", "interactionMode": "plan"}
```

- Model and options: the web's model picker and effort menu call
  `updateThreadMetadata({modelSelection})` (`commands.ts:632`). With
  `serverResolvedCommandContext` it always sends `thread.model-selection.set`
  and the server handles a provider change itself. Without it, the client
  sends `thread.model-selection.set` when `instanceId` equals the thread's
  current `providerInstanceId` and `provider.switch` when it differs
  (`packages/shared/src/model.ts:17`). A provider switch hands context over to
  the new provider; a `handoff` turn item records it.
- Send the whole `modelSelection` each time. `options` replaces the old
  options; it doesn't merge.
- Plan mode: `thread.interaction-mode.set` with `"plan"`; back to normal with
  `"default"`.
- The web doesn't send these on every toggle. Before each send it compares
  the composer's runtime and interaction modes with the server thread and
  dispatches `thread.runtime-mode.set` and `thread.interaction-mode.set` only
  for the ones that differ, then the `message.dispatch`
  (`apps/web/src/components/ChatView.tsx:1968`). The model travels on the
  `message.dispatch` itself as `modelSelection`.

### Answering runtime requests

A pending request shows on the shell as `pendingRuntimeRequest: {id, kind,
createdAt, blocking?}` and in the thread as a `runtime-request.updated` event
plus a turn item. `kind` is one of `command`, `file-read`, `file-change`,
`mcp-elicitation`, `permission`, `dynamic_tool_call`, `user_input`,
`auth_refresh`. Check `runtimeRequests[].responseCapability` first:

| `responseCapability.type` | Meaning |
| --- | --- |
| `live` | The provider is waiting on a callback. Answer with `runtime-request.respond`. `supportsFeedback: true` means `feedback` is passed on. |
| `message` | The answer becomes a follow-up user message. Only for `user_input`. |
| `not_resumable` | The provider process is gone. The web shows "Interrupt or restart the run to respond" and disables the buttons. |

#### Approvals, including MCP elicitation

The turn item is `type: "approval_request"` with `requestId`, `requestKind`,
and optional `prompt`, `reason`, `appName`, `fullAccessOverride` and
`options: [{decision, label, warning?}]`. When `options` is present, show
those buttons; otherwise offer the default set.

```json
{"type": "runtime-request.respond", "commandId": "...", "threadId": "...",
 "requestId": "<request id>", "decision": "accept", "feedback": "optional note"}
```

`decision` is one of `accept`, `acceptForSession`, `acceptAlways`, `decline`,
`cancel`.

MCP elicitation is an approval, not a question. Codex's
`mcpServer/elicitation/request` becomes an `approval_request` with
`requestKind: "mcp-elicitation"`, `prompt` set to the server's message, and
`appName` and `options` describing it (`CodexAdapterV2.ts:5336`). The web
labels it "App access approval". Answer it with a `decision`, never `answers`.
Elicitations that need form input are declined by the adapter before they
reach you.

#### Questions (`user_input`)

The turn item is `type: "user_input_request"`:

```json
{"type": "user_input_request", "requestId": "...", "responseMode": "message",
 "questions": [{"id": "q1", "header": "Database", "question": "Which database?",
   "options": [{"label": "Postgres", "description": "...", "value": "pg"}, {"label": "SQLite", "description": "..."}],
   "multiSelect": false, "allowCustomAnswer": true, "required": true}]}
```

`responseMode`, `multiSelect`, `allowCustomAnswer`, `initialAnswer`,
`required`, and each option's `value` and `preview` are optional.

Answer with `answers`, a map from question `id` to the answer:

```json
{"type": "runtime-request.respond", "commandId": "...", "threadId": "...", "requestId": "...",
 "answers": {"q1": "pg", "q2": ["lint", "test"], "q3": "free text the user typed"}}
```

How the web builds each value (`apps/web/src/pendingUserInput.ts:67`):

- A picked option sends the option's `value`, falling back to its `label`.
- A multi-select question sends an array of those strings.
- Custom text wins over a selection when it's non-empty, and is sent as a
  plain string. `allowCustomAnswer: false` disables it.
- Every question must have an answer, or the web won't send.

For `responseMode: "message"` (or `responseCapability.type: "message"`), every
answer must be a non-empty string; arrays are rejected. Questions with
`required: false` can be skipped. The server turns the answers into a user
message of `"<question>\n<answer>"` blocks and queues or steers it
(`Orchestrator.ts:7988`). These questions can also be dismissed:

```json
{"type": "thread.user-input.dismiss", "commandId": "...", "threadId": "...", "requestId": "..."}
```

`attachmentsByQuestionId` (`{questionId: [ChatAttachment, ...]}`) attaches
uploaded files to an answer. Gate it on `questionAttachments`.

### Proposed plans

There are no plan-specific commands. In plan mode the agent emits a
`proposed_plan` turn item (`planId`, `markdown`, `streaming`) and a
`plan.updated` event. The shell's `hasActionableProposedPlan` says a plan is
waiting. The web offers two actions (`useProposedPlanSend.ts`):

1. **Implement here.** Dispatch `thread.interaction-mode.set {interactionMode:
   "default"}`, then `message.dispatch` with the implementation prompt and
   `sourcePlanRef: {threadId, planId}`.
2. **Implement in a new thread.** `thread.create` with `interactionMode:
   "default"` and the same branch and worktree, then `message.dispatch` on the
   new thread with `sourcePlanRef` pointing at the original thread's plan.

To refine the plan instead, send a normal message while still in plan mode.
`todo_list` items are the agent's running checklist and need no action.

### Checkpoints

```json
{"type": "checkpoint.rollback", "commandId": "...", "threadId": "...",
 "scopeId": "<checkpoint.scopeId>", "checkpointId": "<checkpoint.id>", "restoreFiles": true}
```

"Revert to this turn". `restoreFiles` is optional. Pick a checkpoint with
`status: "ready"` from `projection.checkpoints`. A failure shows up in the
thread's `rollbackFailure`.

### Workflows

Workflows have no dispatch commands. They are subagents with a `workflow`
field (`phases`, `agents`, `runHandles.scriptPath`). The two RPCs are
`orchestration.stopWorkflow` and `orchestration.getWorkflowScript`, below.

### Pull requests

`{host, repository, number}` identifies a PR in all of these.

```json
{"type": "thread.pull-request.link", "commandId": "...", "threadId": "...", "host": "github.com",
 "repository": "owner/repo", "number": 42, "url": "https://github.com/owner/repo/pull/42",
 "source": "manual", "repositoryWebUrl": "https://github.com/owner/repo"}
{"type": "thread.pull-request.unlink", "commandId": "...", "threadId": "...", "host": "github.com", "repository": "owner/repo", "number": 42}
{"type": "thread.pull-request.watch", "commandId": "...", "threadId": "...", "host": "github.com",
 "repository": "owner/repo", "number": 42, "watching": true}
```

`source` is `manual`, `created`, `agent`, `stack` or `stack-dismissed`.
`repositoryWebUrl` is optional. `watch` takes an optional
`link: {url, source, repositoryWebUrl?}` to link and watch in one go. A watch
can wake the agent with a message when checks change (the internal
`thread.pull-request-watch.sync` carries the wake). The web's Stop also
unwatches every watched PR when there's nothing else to stop.

### Everything else in the union

These pass the schema but belong to the server, the agent or background
reactors. Don't send them from a UI.

| Command | Who sends it |
| --- | --- |
| `thread.guest-share.close` | Ending a guest share link. Fine to send; the open side is server-only. |
| `thread.guest-message.dismiss` | Declining a guest's message (see `message.dispatch`). |
| `thread.auto-settle` | The client's auto-settle timer. |
| `thread.title.regeneration.complete` | The title generator. |
| `thread.pull-request-link.sync`, `thread.pull-request.sync` | The PR status poller. |
| `prepared-run.release`, `prepared-run.progress`, `prepared-run.fail` | The launch pipeline. |
| `notification.delivery.accept` | Notification delivery. |
| `delegated_task.request`, `delegated_task.wake-policy`, `delegated_task.completion-delivery.acknowledge`, `delegated_task.completion-delivery.dispose`, `thread.created.record` | The agent's `delegate_task` and thread-creation tools. They use `parentThreadId` instead of `threadId`. |

`OrchestrationV2InternalCommand` (`orchestrationV2.ts:3144`) is not part of
the RPC schema at all: `thread.guest-share.open`, `thread.guest-share.redeem`,
`thread.guest-message.submit`, `thread.pull-request-watch.sync`,
`checkpoint.rollback.fail`, `runtime-request.delivery.fail`,
`thread.background-work.settle`, `secret_request.record`. The socket rejects
them at decode.

## The other orchestration RPCs

| Tag | Kind | Scope | Input | Output |
| --- | --- | --- | --- | --- |
| `orchestration.dispatchCommand` | call | operate | a command | `{sequence}` |
| `orchestration.launchThread` | call | operate | see above | `{threadId, projection, resumed}` |
| `orchestration.stopWorkflow` | call | operate | `{threadId, subagentId}` | nothing (`Void`) |
| `orchestration.subscribeShell` | stream | read | `{afterSequence?, requestCompletionMarker?, omitIdleSubagents?}` | shell items |
| `orchestration.subscribeArchivedShell` | stream | read | `{}` | archived shell items |
| `orchestration.getArchivedShellSnapshot` | call | read | `{}` | `{schemaVersion, snapshotSequence, projects, threads}` |
| `orchestration.subscribeThread` | stream | read | see below | thread items |
| `orchestration.getThreadProjection` | call | read | `{threadId}` | `OrchestrationV2ThreadProjection` (bounded tail) |
| `orchestration.getTurnItem` | call | read | `{threadId, itemId, revision?}` | `{item: TurnItem \| null}` |
| `orchestration.getTurnDiff` | call | read | `{threadId, fromTurnCount, toTurnCount, ignoreWhitespace?}` | `{threadId, fromTurnCount, toTurnCount, diff}` |
| `orchestration.getFullThreadDiff` | call | read | `{threadId, toTurnCount, ignoreWhitespace?}` | same shape, `fromTurnCount` 0 |
| `orchestration.searchThreads` | call | read | `{query, limit?}` | `{matches}` |
| `orchestration.searchThread` | call | read | see below | find result |
| `orchestration.searchThreadStream` | stream | read | same as `searchThread` | find results |
| `orchestration.getThreadFamily` | call | read | `{threadId}` | `{snapshotSequence, threads}` |
| `orchestration.getWorktreeThreadIds` | call | read | `{worktreePath}` | `{threadIds}` |
| `orchestration.getWorkflowScript` | call | read | `{threadId, scriptPath}` | `{scriptPath, contents, truncated}` |
| `orchestration.subscribeCommandOutput` | stream | read | `{threadId, itemId}` | output frames |

"read" is `orchestration:read`, "operate" is `orchestration:operate`.

### `subscribeShell`

Covered in t3-connect.md. What's new or worth pinning down:

- `omitIdleSubagents: true` leaves idle subagent child threads out of the
  snapshot. Live children, and children that are themselves parents, stay
  (`apps/server/src/orchestration-v2/ShellStream.ts:73`). Send it only when
  `server.getConfig` has `shellThreadFamilies: true`, and read the omitted
  rows with `getThreadFamily`. The local nightly doesn't advertise it.
- `snapshot.archivedThreads` is always `[]` now
  (`ShellStream.ts:54`). Archived threads come from `subscribeArchivedShell`
  or `getArchivedShellSnapshot`.
- On resume with `afterSequence`, the server replays if the gap is small and
  otherwise sends a fresh full snapshot. Either way `synchronized` follows
  when you asked for it.
- The server drops `thread.updated` items whose shell didn't change, but
  resends after 5 seconds so resume cursors keep moving
  (`ShellStream.ts:373`).

Item kinds:

| `kind` | Fields | Apply |
| --- | --- | --- |
| `snapshot` | `snapshot: {schemaVersion, snapshotSequence, projects, threads, archivedThreads}`, `resolvedRepositoryIdentityRoots?` | Without `resolvedRepositoryIdentityRoots`: replace everything. With it: merge `projects` by `id` and touch nothing else (t3-connect.md explains why). |
| `project.updated` | `sequence`, `project` | Upsert by `project.id`. |
| `project.removed` | `sequence`, `projectId` | Delete. |
| `thread.updated` | `sequence`, `location: "active"`, `thread` | Upsert the whole `OrchestrationV2ThreadShell` by `thread.id`. It's a full row, not a patch. |
| `thread.removed` | `sequence`, `location: "active"`, `threadId` | Delete. Sent for archive and delete. |
| `synchronized` | | Caught up. |

Keep the highest `sequence` (or `snapshot.snapshotSequence`) as your resume
cursor. `location` is always `"active"` on this stream today; the schema
allows `"archive"`.

Thread shell fields a client usually renders: `title`, `status` (`idle` or a
run status), `activityRunStatus`, `activeRunId`, `latestRunId`,
`pendingRuntimeRequest`, `latestVisibleMessage`, `hasActionableProposedPlan`,
`modelSelection`, `runtimeMode`, `interactionMode`, `lineage`, `pinnedAt`,
`pinOrderKey`, `activeOrderKey`, `settledAt`, `settledOverride`,
`snoozedUntil`, `lastVisitedAt`, `latestUserMessageAt`, `pendingBackgroundTasks`,
`lastError`, `lastErrorClass`, `usageLimitResetAt`, `goal`, `itemCount`,
`visibleItemCount`. Full list at `orchestrationV2.ts:1884`.

### `subscribeArchivedShell` and `getArchivedShellSnapshot`

Both take `{}`. The stream sends one `snapshot` with
`{schemaVersion, snapshotSequence, projects, threads}` (here `threads` is the
archived list), then `thread.updated {sequence, thread}` when an archived
thread changes or gets archived, and `thread.removed {sequence, threadId}`
when one is unarchived or deleted. No `location` field, no `synchronized`, no
resume. Open it only while the archive view is on screen. The call returns
the same snapshot once. The local server had 39 archived threads.

### `subscribeThread`

```json
{
  "threadId": "...",
  "afterSequence": 1234,
  "requestCompletionMarker": true,
  "acceptBoundedSnapshot": true,
  "acceptCompactTurnItems": true,
  "acceptDedupedBodies": true,
  "acceptCompactIds": true
}
```

Everything except `threadId` is optional.

| Option | Effect |
| --- | --- |
| `afterSequence` | Resume. The server replays events after it if there are at most 128 of them and they fit in 1 MB; otherwise it sends a fresh snapshot (`apps/server/src/orchestration-v2/ThreadStream.ts:94`). A value past the server's high water also gets a snapshot. |
| `requestCompletionMarker` | Send `{kind: "synchronized"}` after the snapshot or replay. Needs `threadResumeCompletionMarker`. |
| `acceptBoundedSnapshot` | The snapshot is the recent window, with `historyCursor`, `hasMoreHistory` and `latestLocalTurnOrdinal`. Without it you get the whole thread. Needs `threadSnapshotPagination`. |
| `acceptCompactTurnItems` | With a bounded snapshot: `projection.turnItems` leaves out the items already in `visibleTurnItems` (local rows) and the snapshot says `turnItemsOmitLocalVisible: true`. Rebuild with `boundedSnapshotProjection` in `packages/shared/src/orchestrationV2BoundedSnapshot.ts`, or render only from `visibleTurnItems`. |
| `acceptDedupedBodies` | Bodies repeated elsewhere on the stream are blanked and marked. See "Deduped bodies" below. Applies to the snapshot, the replay and the live tail. |
| `acceptCompactIds` | New in `80455376`. Bounded snapshots send each long ID once. See "The ID table". Only affects bounded snapshots. |

The official client sends `acceptBoundedSnapshot`, `acceptCompactTurnItems`
and `acceptCompactIds` together, plus `acceptDedupedBodies` always
(`packages/client-runtime/src/state/threads.ts:960`). It sends them without
checking server version; old servers ignore the keys.

Item kinds (`orchestrationV2.ts:3505`):

| `kind` | Fields |
| --- | --- |
| `snapshot` | `snapshotSequence`, `projection`, and with bounded snapshots `historyCursor`, `hasMoreHistory`, `latestLocalTurnOrdinal`, `turnItemsOmitLocalVisible?`, `ids?` |
| `event` | `sequence`, `event: {id, threadId, type, payload, occurredAt, runId?, nodeId?, driver?, providerInstanceId?, rawEventId?}` |
| `synchronized` | |

After a `snapshot`, resume events with `afterSequence = snapshotSequence`.
Events can overlap the snapshot; drop any event whose `sequence` you've
already applied.

#### Applying thread events

Every event payload is a complete record, never a patch. Upsert by the
record's key and you're done:

| `event.type` | `payload` | Upsert into `projection.` |
| --- | --- | --- |
| `thread.created`, `thread.archived`, `thread.unarchived`, `thread.deleted`, `thread.settled`, `thread.unsettled`, `thread.snoozed`, `thread.unsnoozed`, `thread.pinned`, `thread.unpinned`, `thread.pin-reordered`, `thread.active-reordered`, `thread.auto-settle-set`, `thread.visited`, `thread.marked-unread`, `thread.metadata-updated`, `thread.pull-request-synced`, `thread.runtime-mode-updated`, `thread.interaction-mode-updated`, `thread.model-selection-updated`, `thread.provider-switched`, `thread.guest-share-updated` | `OrchestrationV2AppThread` | Replace `thread`. |
| `thread.pull-request-link-updated` | `ThreadPullRequestLink` | `thread.pullRequests` by `{host, repository, number}`. |
| `run.created`, `run.updated` | `OrchestrationV2Run` | `runs` by `id`. |
| `run.background-work-cancelled` | `{runId, restartCancelledBackgroundWork}` | Set that field on the run. |
| `run-attempt.created`, `run-attempt.updated` | `OrchestrationV2RunAttempt` | `attempts` by `id`. |
| `node.updated` | `OrchestrationV2ExecutionNode` | `nodes` by `id`. |
| `subagent.updated` | `OrchestrationV2Subagent` | `subagents` by `id`. |
| `provider-session.attached`, `provider-session.updated` | `OrchestrationV2ProviderSession` | `providerSessions` by `id`. |
| `provider-session.detached` | `{providerSessionId, detachedAt, reason?}` | Remove or mark that session. |
| `provider-thread.updated` | `OrchestrationV2ProviderThread` | `providerThreads` by `id`. |
| `provider-turn.updated` | `OrchestrationV2ProviderTurn` | `providerTurns` by `id`. Carries token usage. |
| `runtime-request.updated` | `OrchestrationV2RuntimeRequest` | `runtimeRequests` by `id`. `status` leaves `pending` when answered. |
| `message.updated` | `OrchestrationV2ConversationMessage` | `messages` by `id`. |
| `turn-item.updated` | a turn item | `turnItems` by `id`, and the matching `visibleTurnItems[].item` (match on `item.id`, or `sourceItemId`). A new local item gets a new visible row; order by `item.ordinal`. |
| `plan.updated` | `OrchestrationV2PlanArtifact` (`kind: "proposed_plan"` or `"todo_list"`) | `plans` by `id`. |
| `checkpoint-scope.created` | `OrchestrationV2CheckpointScope` | `checkpointScopes` by `id`. |
| `checkpoint.captured` | `OrchestrationV2Checkpoint` | `checkpoints` by `id`. |
| `checkpoint.rollback-requested` | `{scopeId, checkpointId, requestedAt}` | Show "reverting". |
| `context-handoff.updated` | `OrchestrationV2ContextHandoff` | `contextHandoffs` by `id`. |
| `context-transfer.created`, `context-transfer.updated` | `OrchestrationV2ContextTransfer` | `contextTransfers` by `id`. |

Unknown event types, and `turn-item.updated` with an unknown item `type`,
decode as `{kind: "unknown-event", sequence, eventType}` in the official
client. Do the same: skip it but advance your cursor.

Run statuses are `preparing`, `queued`, `starting`, `running`, `waiting`,
`completed`, `interrupted`, `failed`, `cancelled`, `rolled_back`. A run is
active in `preparing`, `starting`, `running` and `waiting`; that's the set
the web uses to find `targetRunId`.

### `getThreadProjection`

`{threadId}` in, a projection out. It's bounded to the recent window like a
bounded snapshot, but without the window fields, so you can't page from it.
Fine for a one-off peek; for anything you'll keep, subscribe.

### `getTurnItem`

`{threadId, itemId, revision?}` returns `{item}` with output and diffs
restored, or `{item: null}`. The server ignores `revision` at this commit;
the official client uses it to key its own cache. See t3-connect.md.

### `getTurnDiff` and `getFullThreadDiff`

```json
{"threadId": "...", "fromTurnCount": 2, "toTurnCount": 3, "ignoreWhitespace": true}
{"threadId": "...", "toTurnCount": 3}
```

Turn counts are checkpoint `appRunOrdinal`s (0 is before the first run).
`fromTurnCount` must be `<= toTurnCount`. Both return
`{threadId, fromTurnCount, toTurnCount, diff}`, where `diff` is a unified diff
string. `ignoreWhitespace` defaults to `true` on the server. Use the first
for "changes in this turn" and the second for "all changes so far" (the web's
`DiffPanel.tsx` shows both). Errors are
`OrchestrationGetTurnDiffError` and `OrchestrationGetFullThreadDiffError`.

### `searchThreads`: search across threads

```json
{"query": "login test", "limit": 20}
```

`query` is trimmed, 2 to 200 characters; `limit` is 1 to 50. Returns:

```json
{"matches": [{"threadId": "...", "projectId": "...", "source": "user", "snippet": "...the login test fails...",
  "messageCreatedAt": "2026-10-24T17:01:02.000Z", "quality": 1, "hits": 3, "messageId": "..."}]}
```

`source` is `user` or `assistant`. `snippet` is at most 240 characters.
`quality` (1 whole word, 0.8 prefix, 0.3 infix, plus bonuses), `hits` and
`messageId` are new; the local nightly omits all three. It searches message
text only, not tool output.

### `searchThread` and `searchThreadStream`: find in one thread

The Ctrl+F bar inside a thread. Gate on `threadFind`; use the stream version
when `threadFindProgressive` is set.

```json
{"threadId": "...", "query": "TODO", "index": 0}
{"threadId": "...", "query": "TODO", "start": {"entryId": "...", "occurrence": 0}, "offset": 1}
```

| Field | | Notes |
| --- | --- | --- |
| `threadId`, `query` | req | `query` 1 to 200 characters. |
| `index` | opt | Jump to the nth match overall. |
| `start` | opt | `{entryId, occurrence}`, the match you're on. |
| `offset` | opt | Step from `start`: `1` next, `-1` previous. Wraps around. |
| `skills` | opt | `[{name, displayName?}]` so skill mentions match by their display names. |

Result:

```json
{"complete": true, "snapshotSequence": 5678, "totalMatches": 15, "activeIndex": 3,
 "match": {"entryId": "<messageId or turn item id>", "runId": "...", "occurrence": 1},
 "navigation": [{"entryId": "...", "runId": "...", "startIndex": 0, "count": 2}]}
```

`entryId` is the `messageId` for user and assistant messages and the turn item
`id` for everything else. `occurrence` is which hit inside that entry.
`navigation` lists up to 17 entries with matches around the active one, for
the scrollbar ticks. `match` is `null` when nothing matches.

With no `index` and no `offset`, `searchThreadStream` first sends a quick
`{complete: false, totalMatches: 1, activeIndex: 0, match}` as soon as it
finds the first hit at or after `start`, then the full result. Otherwise it
sends one item. `searchThread` returns only the last item. On the local
nightly the stream sent one item and no `complete` field.

### `getThreadFamily` and `getWorktreeThreadIds`

`{threadId}` returns every active thread that shares the thread's
`lineage.rootThreadId`: the whole subagent tree, as thread shells. An unknown
thread has an empty family. `{worktreePath}` returns the IDs of active threads
on that worktree, subagents included. Both exist for clients that subscribe with `omitIdleSubagents`. The
local nightly doesn't have them (calls fail), matching its missing
`shellThreadFamilies` flag.

### `stopWorkflow` and `getWorkflowScript`

`{threadId, subagentId}` stops a provider-native workflow run. It needs
`orchestration:operate` even though it isn't a dispatch command. Errors are
`OrchestrationV2StopWorkflowError` with `reason` of `not-running`,
`unsupported`, `unavailable` or `stop-failed`. The web shows Stop on a
workflow agent row in `ThreadRelationshipsControl.tsx`.

`{threadId, scriptPath}` returns `{scriptPath, contents, truncated}`. Take
`scriptPath` from `subagent.workflow.runHandles.scriptPath`. It must be an
absolute `.js` path inside the server's workflow scripts root, or you get
`OrchestrationGetWorkflowScriptError` with `reason` `invalid-path`,
`root-unavailable`, `not-found`, `outside-root`, `not-js`, `not-regular-file`,
`changed-during-read` or `read-failed`.

### `subscribeCommandOutput`

`{threadId, itemId}` for a `command_execution` item. Live output for an
expanded command row, never persisted. Gate on `commandOutputStreaming`.

```json
{"kind": "replace", "text": "...", "truncated": false, "running": true}
{"kind": "append",  "text": "...", "truncated": false, "running": true}
{"kind": "append",  "text": "", "truncated": false, "running": false}
```

`replace` resets your buffer to `text`; `append` adds to it. `truncated`
means older output was dropped to keep the tail bounded. `running: false` is
the last frame. The text has terminal escapes; feed it to a terminal emulator
or strip them. Unsubscribe with `Interrupt` when the row collapses.

## The ID table (commit `80455376`)

"Thread snapshots send each long ID once." A bounded snapshot repeats the
same UUIDs many times: every turn item carries `threadId`, `runId`,
`nodeId`, `providerThreadId`, `providerTurnId` and more. The commit adds an
opt-in that sends each distinct ID once in a top-level `ids` array and
replaces every occurrence with its index.

**It is opt-in.** Over the socket, send `acceptCompactIds: true` on
`subscribeThread`, together with `acceptBoundedSnapshot: true`; a full
snapshot never gets an ID table. Over HTTP, add `compactIds=1` to
`GET /api/orchestration/threads/:threadId/bounded` and
`GET /api/orchestration/threads/:threadId/history`. Without the flag the wire
is unchanged. The commit makes no other change to the wire.

**What changes.** Only `snapshot` items (and history pages) change. Events,
replay and `synchronized` are unchanged. A snapshot with the table looks like
this:

```json
{"kind": "snapshot", "snapshotSequence": 5678, "historyCursor": "...", "hasMoreHistory": true,
 "latestLocalTurnOrdinal": 12, "turnItemsOmitLocalVisible": true,
 "projection": {
   "thread": {"id": 0, "projectId": 1, "providerInstanceId": 2, "activeProviderThreadId": 3, "lineage": {"parentThreadId": null, "relationshipToParent": null, "rootThreadId": "a244..."}, ...},
   "runs": [{"id": 4, "threadId": 0, "userMessageId": 5, ...}],
   "visibleTurnItems": [{"position": 0, "visibility": "local", "sourceThreadId": 0, "sourceItemId": 6,
                         "item": {"id": 6, "threadId": 0, "runId": 4, "messageId": 5, "type": "user_message", ...}}],
   ...},
 "ids": ["a244...", "proj-...", "claudeAgent", "pt-...", "run-...", "msg-...", "ti-..."]}
```

The rule (`packages/contracts/src/orchestrationV2IdTable.ts`), applied to
`projection` in a snapshot or to the top level of a history page:

1. A "record" is each object element of a top-level array (`runs[i]`,
   `turnItems[i]`, `visibleTurnItems[i]`, `items[i]` in a history page),
   each top-level object value (`thread`), and the `item` object inside a
   record. Nothing deeper.
2. In a record, a key named `id` or ending in `Id` whose value is a number
   is an index: replace it with `ids[n]`.
3. A key ending in `Ids` whose value is an array: replace each number in it
   with `ids[n]` (`handoffIds` on provider threads, `fromProviderThreadIds`
   and `fromProviderInstanceIds` on handoff items).
4. `null` stays `null`. Nested objects other than `item` keep their strings:
   `thread.lineage.rootThreadId`, `run.sourcePlanRef.planId` and
   `modelSelection.instanceId` arrive as plain strings.
5. Then drop the `ids` key.

Every string or number under such a key gets interned, so after expansion
they are back to their original values. Keys are matched case-sensitively.
A Rust decoder
can do this on `serde_json::Value` before deserializing into typed structs:

```rust
fn expand(record: &mut Map<String, Value>, ids: &[Value]) {
    for (key, value) in record.iter_mut() {
        if key == "id" || key.ends_with("Id") {
            if let Some(n) = value.as_u64() { *value = ids[n as usize].clone(); }
        } else if key.ends_with("Ids") {
            if let Some(list) = value.as_array_mut() {
                for v in list { if let Some(n) = v.as_u64() { *v = ids[n as usize].clone(); } }
            }
        } else if key == "item" {
            if let Some(obj) = value.as_object_mut() { expand(obj, ids); }
        }
    }
}
// For a snapshot: take `ids` off the item, then for each value in `projection`:
// arrays -> expand each object element; objects -> expand the object.
```

Expand only when `ids` is an array. The rule is fixed for this opt-in; the
comment in the source says a change would need a new flag (`compactIds=2` or a
new `accept*` key).

**Old vs new server.** I subscribed to a 30-item thread on the local
`0.1.3-nightly.20261023.3592` with all four `accept*` flags. It returned a
normal bounded snapshot with string IDs, no `ids` key and no error. So:

| Server | Sends with `acceptCompactIds: true` |
| --- | --- |
| Older than `80455376`, including the local nightly | Plain snapshot, string IDs. The flag is ignored. |
| `80455376` and later | `ids` array plus integer indexes in records, for bounded snapshots and `compactIds=1` HTTP pages. |

There's no capability flag for this. A client that always sends the flag and
always checks for `ids` works against both, which is what the official client
does.

The local nightly also sent `payloadBudgetExceeded` on the bounded snapshot.
That key isn't in the upstream schema any more; ignore it.

## Deduped bodies

`acceptDedupedBodies: true` (HTTP: `dedupeBodies=1`) blanks a body the
client can copy from a record it already has, and sets a marker. Restore
before rendering (`packages/shared/src/orchestrationV2DedupedBodies.ts`):

| Marker | Blanked to | Restore from |
| --- | --- | --- |
| `user_message.textOmitted`, `assistant_message.textOmitted` | `text: ""` | `messages[]` entry with `id == item.messageId` |
| `checkpoint.filesOmitted` | `files: []` | `checkpoints[]` entry with `id == item.checkpointId` |
| subagent item `promptOmitted`, `resultOmitted` | `prompt: ""`, `result: null` | `subagents[]` entry with `id == item.subagentId` |
| `subagent.updated` payload `promptOmitted`, `resultOmitted` | same | The previous record for that subagent on this stream |

Never store or forward a marked value. The deduper sees the snapshot, replay
and live tail as one stream, so keep its state per subscription. On the local
nightly a small thread came back with no markers at all, so I couldn't tell
whether it supports this. If you don't want to write the restore logic, leave
the flag off.

## Auth scopes

Scopes come from the session (`GET /api/auth/session` → `permissions`, or
`scopes` on older servers). The server checks each request against
`RPC_REQUIRED_SCOPES` in `apps/server/src/auth/RpcAuthorization.ts:42`.
`clientRpcPermissions.ts` is the client's copy for hiding buttons; the server
enforces regardless.

| Scope | Methods |
| --- | --- |
| `orchestration:read` | `subscribeShell`, `subscribeArchivedShell`, `getArchivedShellSnapshot`, `subscribeThread`, `getThreadProjection`, `getTurnItem`, `getTurnDiff`, `getFullThreadDiff`, `searchThreads`, `searchThread`, `searchThreadStream`, `getThreadFamily`, `getWorktreeThreadIds`, `getWorkflowScript`, `subscribeCommandOutput`. Also `server.probe`, `server.getConfig`, `subscribeServerConfig`. |
| `orchestration:operate` | `dispatchCommand` (every command type), `launchThread`, `stopWorkflow`. |

There's no per-command check: an `operate` session can send any command in
the public union. The HTTP history and bounded snapshot routes need
`orchestration:read`. Attachment upload URLs (`attachments.createUploadUrl`)
are separate; see t3-connect.md.

A watch-style client wants exactly
`--scope orchestration:read --scope orchestration:operate`.

## What the local nightly does differently

Checked on 2026-10-24 with a read-only session against
`0.1.3-nightly.20261023.3592`:

- Advertises `serverResolvedCommandContext`, `threadStop`, `threadFind`,
  `threadFindProgressive`, `threadSnapshotPagination`,
  `commandOutputStreaming`, `questionAttachments`. Doesn't advertise
  `shellThreadFamilies`.
- `getThreadFamily` and `getWorktreeThreadIds` fail.
- `searchThreads` matches have no `quality`, `hits` or `messageId`.
- `searchThread` results have no `complete`.
- `acceptCompactIds` is ignored. No ID table.
- Bounded snapshots carry an extra `payloadBudgetExceeded` key.
- `snapshot.archivedThreads` is `[]`, as upstream.

I didn't send any commands to it, so the command payloads above come from the
schemas and the official client, not a live round trip.
