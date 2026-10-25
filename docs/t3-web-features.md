# T3 Code web features, mapped to the protocol

This is an inventory of what the T3 Code web app shows and does, with the RPC
method or orchestration command behind each feature. It's for building a
third-party client (the Rust app, and the browser PoC in `poc/`) that gets to
parity one feature at a time.

It comes from reading upstream `pingdotgg/t3code` at `80455376`. I didn't run
the upstream app. Paths are relative to that repo, with two short forms:

- `web/` is `apps/web/src/`
- `cr/` is `packages/client-runtime/src/`

Payload shapes live in the sibling docs. This one only names things.

- [orchestration.md](orchestration.md): `orchestration.*` methods, the
  `dispatchCommand` command union, shell and thread streams, turn items.
- [server-settings-models.md](server-settings-models.md): `server.getConfig`,
  settings, providers, models and their option descriptors.
- [projects-files-vcs-terminal.md](projects-files-vcs-terminal.md):
  `projects.*`, `filesystem.*`, `vcs.*`, `git.*`, `review.*`, `terminal.*`.
- [integrations.md](integrations.md): pull requests, preview, devices,
  scheduled tasks, guest share, MCP apps.
- [t3-connect.md](t3-connect.md): getting a session and a socket in the first
  place.

## How to read this

Every feature has a tier:

- **core chat**: any client needs it to be usable as a chat client.
- **power**: a full client wants it, a watch doesn't.
- **desktop-only**: needs Electron, a local editor, OS dialogs or the
  in-app webview. A remote client can't do it, or can only fake it.

"`cmd:X`" means `orchestration.dispatchCommand` with `{type: "X", commandId,
...}`. Everything else in backticks is a WebSocket RPC method name from
`packages/contracts/src/rpc.ts` (`WS_METHODS`) or
`packages/contracts/src/orchestrationV2.ts` (`ORCHESTRATION_V2_WS_METHODS`),
unless it starts with `/`, in which case it's an HTTP path on the environment.

The web UI never calls the protocol directly. It goes component, hook,
`cr/state/*.ts` atom, then `cr/operations/commands.ts`, which builds the
command. That last file is the best single place to see what a button really
sends.

Three habits of the upstream client are worth copying:

1. **Capabilities gate behaviour.** `server.getConfig` returns
   `environment.capabilities`. The client branches on flags such as
   `serverResolvedCommandContext`, `threadStop`, `attachmentUploads`,
   `commandOutputStreaming`, `inlineMessageContext`, `pullRequests`,
   `serverBrowser`, `threadSettlement`, `threadSnooze`, `threadPinning`,
   `requiredWorktreeBootstrap`, plus top-level config flags such as
   `shellThreadFamilies` and `threadFindProgressive`. Older servers lack some,
   and the UI hides the feature or falls back.
2. **Mode and model changes are deferred.** Picking a model, runtime mode or
   plan mode only changes the local draft. The commands go out on the next
   send, just before `message.dispatch`.
3. **Sorting, grouping and status are client-side.** The server stores order
   keys and timestamps. Everything the sidebar shows is derived from the shell
   stream.

Auth scopes matter for which features appear: `orchestration:read`,
`orchestration:operate`, `terminal:read`, `terminal:operate`,
`preview:operate`, `filesystem:read`, `filesystem:write`, `diagnostics:read`.

## Data plumbing

| Feature | What it does | Upstream | Protocol | Tier |
|---|---|---|---|---|
| Thread list data | Every row in the sidebar | `cr/state/shell.ts`, `cr/state/shellSnapshotHttp.ts` | `GET /api/orchestration/shell` first (20 s timeout), falls back to the socket, then `orchestration.subscribeShell` for live `snapshot`, `project.updated`, `project.removed`, `thread.updated`, `thread.removed` | core chat |
| Thread data | The open thread | `cr/state/threads.ts`, `cr/state/boundedThreadSnapshotHttp.ts` | `orchestration.subscribeThread` (bounded snapshot plus events), or `GET /api/orchestration/threads/:id/bounded?compactTurnItems=1&dedupeBodies=1&compactIds=1` | core chat |
| Older history | "Load earlier turns" | `cr/state/threadHistoryHttp.ts` | `GET /api/orchestration/threads/:id/history?cursor=...` | core chat |
| Server config | Providers, models, skills, slash commands, capabilities, keybindings, settings | `cr/state/server.ts` | `server.getConfig`, `subscribeServerConfig` | core chat |
| Client settings | Local preferences | `web/hooks/useSettings.ts` | localStorage (browser) or the desktop bridge. Server keys in the same patch go to `server.updateSettings` | core chat |

## Sidebar and thread list

The default sidebar is not grouped by project. It's one flat list across all
projects and environments, in sections: Pinned, Active, Working (beta),
Snoozed, Settled. Project grouping only feeds the project filter and the
opt-in legacy sidebar (`legacySidebarEnabled`, off by default).

Main files: `web/components/Sidebar.tsx`, `web/components/sidebar/*`,
`Sidebar.logic.ts`, `web/components/threadActionMenu.logic.ts`,
`web/hooks/useThreadActions.ts`, `cr/state/threadSort.ts`,
`cr/state/threadInbox.ts`, `cr/state/threadSettled.ts`.

| Feature | What the user sees | Upstream | Protocol | Tier |
|---|---|---|---|---|
| Sections | Pinned, an "Active" divider, Working shelf, Snoozed shelf, Settled shelf. Shelves collapse; Settled loads N at a time | `Sidebar.tsx`, `Sidebar.logic.ts` | client-only | core chat |
| Sort order | Pinned by `pinOrderKey`. Active by `activeOrderKey`, else anchor time. Snoozed by soonest wake. Settled by `sortSettledThreads` | `cr/state/threadSort.ts`, `threadInbox.ts` | client-only | core chat |
| Legacy sort options | Projects by last user message, created at, or manual drag. Threads by last user message or created at. Preview count 1 to 15 | `web/components/LegacySidebar.tsx` | client-only | power |
| Project grouping | Modes `repository` (merges the same git remote across machines), `repository_path`, `separate`, with per-project overrides | `cr/state/projectGrouping.ts`, `web/sidebarProjectGrouping.ts`, `web/logicalProject.ts` | client-only, uses `repositoryIdentity` from the shell | power |
| Project filter | "All projects" dropdown with search; "Filter by X" in the row menu | `Sidebar.tsx` | client-only | power |
| Multiple environments | Remote threads get a machine icon and label. Search covers every environment | `web/components/ThreadStatusIndicators.tsx` | one socket per environment | power |
| Pin, unpin | Row menu, pin icon on the row, `mod+shift+p`. Unpin can ask to confirm and offers undo. Pinning clears snooze | `useThreadActions.ts` | `cmd:thread.pin` (optional `orderKey`), `cmd:thread.unpin` | core chat |
| Drag to reorder or move between sections | Drag shows a verb badge (Pin, Unpin, Settle, Un-settle, Wake) | `Sidebar.logic.ts`, `Sidebar.drag.ts` | `cmd:thread.pin.reorder`, `cmd:thread.active.reorder`, plus pin, unpin, settle, unsettle, unsnooze | power |
| Sweep | Press a row's settle or wake button and drag across rows to apply it to each | `sidebar/SidebarThreadRow.tsx` | same commands per row | power |
| Settle, un-settle | Hover check button, menu item, `mod+shift+s`. Undo toast | `useThreadActions.ts` | `cmd:thread.settle`, `cmd:thread.unsettle {reason: "user"}` | core chat |
| Settle and advance | `mod+shift+s` in a thread settles it and opens the next one | `web/settleAndAdvance.ts` | `cmd:thread.settle` | power |
| Auto-settle | Per-thread "Auto-settle behavior" Enabled or Disabled. Global: after N days (default 3), on merge (default on), keep threads with PRs | `threadActionMenu.logic.ts` | `cmd:thread.auto-settle.set {enabled}`; globals via `server.updateSettings`. The server itself sends the internal `thread.auto-settle` | power |
| Snooze presets | "In 1 hour", "In 3 hours", "This evening" (18:00), "Tomorrow" (09:00), "Next week" (Monday 09:00), "Custom...". Disabled while the thread waits on you | `cr/state/threadSettled.ts`, `web/components/Sidebar.snooze.ts` | `cmd:thread.snooze {snoozedUntil}` | core chat |
| Custom snooze | Date and time, or a duration in minutes, hours or days | `web/components/CustomSnoozeDialog.tsx` | `cmd:thread.snooze` | power |
| Wake | "Wake thread" item. Threads also wake early on approval, input, guest request or new failure, and show a dismissable "Woke" chip | `threadSettled.ts`, `useThreadActions.ts` | `cmd:thread.unsnooze {reason: "user"}`; dismissing the chip sends `cmd:thread.visit` | core chat |
| Archive, unarchive | Menu item (disabled while running, optional confirm), undo toast. Archived list lives in Settings | `useThreadActions.ts` | `cmd:thread.archive`, `cmd:thread.unarchive`; list via `orchestration.getArchivedShellSnapshot` | core chat |
| Delete | Destructive confirm. Detaches the provider session first, may offer to delete the worktree too | `useThreadActions.ts` | `cmd:provider-session.detach`, `cmd:thread.delete`; worktree check uses `orchestration.getWorktreeThreadIds`, `vcs.removeWorktree`, `vcs.refreshStatus` | core chat |
| Multi-select | Modifier-click rows. Menu: Unpin, Settle (N), Snooze (N), Regenerate titles, Mark unread (N), Delete (N) | `Sidebar.tsx`, `Sidebar.logic.ts` | the same commands, once per thread | power |
| Unread, mark unread | A green "Done" chip when the latest run finished after `lastVisitedAt`. Menu "Mark unread" | `Sidebar.logic.ts`, `useThreadActions.ts` | `cmd:thread.mark-unread` | core chat |
| Visit tracking | Opening a thread records a visit; the watermark advances while you read (throttled to 10 s) | `web/components/chat/ThreadVisitWatermark.tsx` | `cmd:thread.visit {visitedAt, opened?, searchQuery?}` | core chat |
| Rename | Double-click the title, or menu "Rename thread". Empty titles are rejected | `Sidebar.tsx`, `SidebarThreadRow.tsx` | `cmd:thread.metadata.update {title}` | core chat |
| Regenerate title | Menu item, shows "Regenerating..." | `web/hooks/useThreadActionMenu.ts` | `cmd:thread.metadata.update {regenerateTitle: true}` | power |
| Row menu | New thread on {branch}, Pin, Settle, Snooze or Wake, Rename, Regenerate title, Mark unread, Share with guest, Filter by project, Auto-settle behavior, Copy (path, branch, thread ID), Project settings, Archive, Delete. Without operate scope the write items are disabled | `threadActionMenu.logic.ts` | as above; Share uses `guestShare.create` | core chat |
| Draft rows | Unsent local drafts show with a pen icon; "Discard draft" with undo | `sidebar/SidebarDraftBlock.tsx` | client-only | core chat |
| Status chip | One chip per row, in priority order (table below) | `Sidebar.logic.ts`, `SidebarThreadRow.tsx` | client-only, from shell fields | core chat |
| PR, terminal, worktree icons | PR state icon (open emerald, merged violet, closed red, draft), stack badge, pulsing terminal icon, worktree icon | `ThreadStatusIndicators.tsx` | `subscribeVcsStatus`, `pullRequests.*`, `subscribeTerminalMetadata` | power |
| Hover card | Title, project, environment, branch (warns if checked out elsewhere), model, "Handed off from", terminal count, error or usage limit, PR list | `web/components/ThreadHoverCard.tsx` | client-only | power |
| Search | Title matches filter locally. Message matches come from the server, debounced 120 ms, need a 3+ letter word. "In messages" section | `Sidebar.tsx`, `cr/state/threadSearch.ts` | `orchestration.searchThreads {query}` per environment; picking a hit sends `cmd:thread.visit {opened, searchQuery}` | core chat |
| Thread navigation | `mod+shift+[` and `]` for previous and next; `mod+1..9` jump on desktop | `web/keybindings.ts` | client-only | power |
| Drop files on a row | Opens that thread with the files queued in its composer | `Sidebar.tsx` | client-only until send | power |
| Drag a row into the composer | Adds the thread as context | `Sidebar.tsx` | client-only until send | power |
| Undo toast | "Settled", "Snoozed", "Unpinned", "Archived", "Discarded" with Undo for 5 s; `mod+z` | `web/hooks/threadUndo.ts`, `cr/state/threadUndo.ts` | the inverse command (unsettle plus re-pin and re-snooze, unsnooze, unarchive) | core chat |

Status chips on a default sidebar row, highest priority first:

| Chip | Meaning | Colour |
|---|---|---|
| Approval | A runtime request waits for approve or decline | warning |
| Input | A question waits for an answer | indigo |
| Request | A guest message waits | guest |
| Waiting | Runtime idle, or blocked on a thread wait, with a timer | muted |
| Working (or "Goal") | Preparing, queued, starting, running, waiting, with a duration timer | info |
| Failed | Last run failed | error |
| Limited | `lastErrorClass` is `usage_limit` | warning |
| Woke | Woke from snooze early | warning |
| Done | A run finished since your last visit | success |

The legacy sidebar uses a second, separate set of pills (Pending Approval,
Awaiting Input, Guest Request, Working, Connecting, Waiting, Failed, Plan
Ready, Completed). A new client only needs the first set.

## Thread view

The timeline pipeline is `web/session-logic.ts`
(`deriveTimelineEntriesFromVisibleTurnItems`) then
`web/components/chat/MessagesTimeline.logic.ts` (`deriveMessagesTimelineRows`),
rendered by `MessagesTimeline.tsx` in a virtualised list. The input is
`projection.visibleTurnItems` from `orchestration.subscribeThread`.

Some items never appear as timeline rows: `todo_list` and `checkpoint` show
elsewhere, `run_interrupt_request` only stops folding, and workspace
preparation items become the worktree setup card.

### Folding: "Worked for Xs"

Upstream: `deriveTurnFolds` in `MessagesTimeline.logic.ts`,
`TurnFoldTimelineRow` in `MessagesTimeline.tsx`. Tier: core chat.

- Entries group by run id. A new response starts at a non-steer user message
  or a `notification`.
- In a finished run, everything before the run's last assistant message folds
  into one row: reasoning, tools, intermediate replies.
- These stay visible: the final assistant message, compaction dividers, fork,
  thread_created and secret_request cards, subagent cards while a child is
  live, notifications, answered questions.
- No fold when the run is still active, failed (or has a top-level failed
  `error`), was interrupted mid-request, or still has a streaming message.
- Label is `Worked for {duration}` or `Worked`. An interrupted latest run says
  `You stopped after {duration}` or `You stopped this response`.
- A steer that arrives after output starts a new segment: `Continued for
  {duration}`.
- Duration is run `startedAt` to `completedAt` when known, else prompt to the
  final message's `updatedAt`.
- Collapsed by default, chevron right; open state is per run.
- A run attempt with status `superseded` gets its own fold: `Superseded
  attempt · Cut off by a steer`.

### Tool grouping and live rows

Tier: core chat. Upstream: `MessagesTimeline.logic.ts`,
`cr/work-log/presentation.ts`.

- Consecutive work entries in the same run and attempt form one group. Errors,
  compaction, `system_notice`, notifications and answered questions break a
  group.
- A single tool shows as one line. Several become a toggle row with a summary
  from `summarizeToolGroup`: "Read 3 files", "Changed 2 files", "Ran 4
  commands", "Searched the web 2 times", "Used 3 tools", "Thought (x2)",
  joined with "and Performed N other actions". Failed groups get a red X.
- While a run is working:
  - A `Working for {Ns}` row at the top of the turn (1 s tick), or `Setting up
    worktree...`, `Compacting...`, `Compacting after being idle...`.
  - A `work-live` row for the trailing tool: `Running {program}`, or for
    reasoning `Thinking` plus a one-line preview. It shimmers.
  - A `Thinking` row with a brain icon when there's no live tool row.
- User messages over 8 lines or 600 characters collapse with "Show full
  message".

### Turn item types

| Item type | What the user sees | Upstream | Protocol behind interactions | Tier |
|---|---|---|---|---|
| `user_message` | Right-aligned bubble, markdown without raw HTML. "Steer" marker for `inputIntent` steer or promoted. Guest messages show "{name} (guest)". Scheduled-task and agent-sent messages fold under a header with "Open automation" or "Open thread". Image grid (click to zoom), file rows with Preview and Download, context chips. Hover: timestamp, "Edit from here", "Copy message", "Copy prompt link" | `web/components/chat/TimelineUserMessageRow.tsx` | attachments via `assets.createUrl`; Edit from here is `cmd:checkpoint.rollback` (below) | core chat |
| `assistant_message` | Markdown, streaming. "(empty response)" if blank. Changed-files card (`N changed files +a -d`, Open diff). Meta row on the final message: status chip, "Copy message", "Fork from this response", timestamp | `chat/TimelineAssistantRows.tsx` | fork is `cmd:thread.fork` (below) | core chat |
| `reasoning` | Brain icon, the thought flattened to one line as the label, `Thinking` or `Thought` as fallback. Expands to markdown | work entry rows | none | core chat |
| `command_execution` | `Ran {program}` or `Running ...`. Expands to the command, output, and a red `exit N` | `chat/V2ItemInspector.tsx`, `CommandOutputPanel.tsx` | `orchestration.getTurnItem` when `outputOmitted`; `orchestration.subscribeCommandOutput {threadId, itemId}` for live output when `capabilities.commandOutputStreaming`, only while the row is open | core chat |
| `file_change` | `Changed {file}` or `Changed N files`. Expands to path, `+a -d`, Open diff, change list. Diff text only when the change failed | `V2ItemInspector.tsx` | `orchestration.getTurnDiff` for Open diff | core chat |
| `file_search`, `web_search` | `Searched files`, `Searched the web` or the pattern. Expands to patterns and results (`file:line:col` with preview, or linked titles with snippets) | `V2ItemInspector.tsx` | none | core chat |
| `dynamic_tool` | Tool title from MCP tool presentation or the tool name. Expands to arguments (JS REPL code highlighted), output text and images. T3 thread tools show a thread link | `V2ItemInspector.tsx`, `cr/t3ToolSummary.ts` | `orchestration.getTurnItem` when input is summarised or output omitted | core chat |
| `dynamic_tool` (html_render) | Inline iframe of an HTML attachment, "Open full screen" | `chat/HtmlRenderFrame.tsx` | `assets.createUrl` | power |
| `dynamic_tool` (MCP app) | Embedded MCP app frame | `chat/McpAppFrame.tsx` | `mcpApps.callTool`, `mcpApps.toolInfo`, `mcpApps.readResource`, `mcpApps.updateModelContext` | power |
| `approval_request` | A row "Approval requested" with the prompt. The live controls are in the composer | work entry rows | `cmd:runtime-request.respond` (composer section) | core chat |
| `user_input_request` | Pending: in the composer. In the timeline: "Input requested", later the question with an answer preview; expands to the Q and A | work entry rows | `cmd:runtime-request.respond` (composer section) | core chat |
| `secret_request` | Card with label, reason, password field "Paste the secret", "Stored securely, never shown to the agent", "Save securely" and "Decline". Settled: one line `{label} · Saved securely and kept private` | `chat/SecretRequestCard.tsx`, `cr/secretRequest.ts` | `secrets.answerRequest {threadId, turnItemId, answer: {type: "save", secret} or {type: "decline"}}` | core chat |
| `proposed_plan` | Card with a "Plan" badge and the first heading as title. Long plans preview 10 lines with "Expand plan". Menu: Copy, Download as markdown, Save to workspace | `chat/ProposedPlanCard.tsx` | Save uses `projects.writeFile`; Implement and Refine are in the composer section | core chat |
| `todo_list` | Not a row. The composer shows a "Tasks n/m" badge that expands to the step list | `chat/ComposerTasksBadge.tsx`, `session-logic.ts` | none (`projection.plans` with `kind: "todo_list"`) | power |
| `notification` | Row with the summary. Icon by source (subagent, command, monitor, background task, system). Child-thread notifications link to the child with "Finished", "Failed", "Stopped", "Updated" | `chat/TimelineWorkEntryRow.tsx` | opening a child is navigation | core chat |
| `error` | Red label and message. `usage_limit`: amber "Usage limit reached. Retry after {time}." Provider retries: "Retrying provider (n/m)...", "Provider recovered". Workspace prep failure: Retry button | `TimelineWorkEntryRow.tsx` | Retry is `cmd:prepared-run.retry {runId}` | core chat |
| `system_notice` | Row with a warning marker | `TimelineWorkEntryRow.tsx` | none | core chat |
| `compaction` | Centered divider: "Context compacted {before} -> {after} tokens", "Compacting context", "Context compaction failed". Idle compaction has its own titles. A bare `/compact` turn collapses into this | `MessagesTimeline.tsx` | none | core chat |
| `run_interrupt_result` | Red divider "Run interrupted · {message}" | `chat/V2LifecycleRow.tsx` | none | core chat |
| `run_interrupt_request` | Not drawn. Prevents the run from folding | | | |
| `checkpoint` | Not a row. Feeds the changed-files card and "Edit from here" | `useThreadCheckpointSummaries` | `cmd:checkpoint.rollback` | core chat |
| `fork` | Divider "Forked from conversation" with "Open source conversation", or "Conversation fork" with "Open fork" | `V2LifecycleRow.tsx` | navigation | core chat |
| `handoff` | Divider "Context handoff" with from and to provider/model icons | `V2LifecycleRow.tsx` | none | power |
| `thread_created` | "Created thread · {title}" with "Open chat"; after the final reply it becomes a summary card | `V2LifecycleRow.tsx` | navigation | power |
| `subagent` | Row per subagent: provider avatar with status dot, title, status (Queued, Running, Waiting, Idle · resumable, Completed, Failed, Stopped), progress preview, elapsed time. Click opens the child thread. Consecutive ones group: "N subagents", "2 working · 1 done" | `V2LifecycleRow.tsx`, `chat/TimelineV2EventRow.tsx`, `cr/state/threadSubagents.ts` | stop is `cmd:subagent.stop {runId, subagentId}`; family via `orchestration.getThreadFamily` when `shellThreadFamilies` | core chat |

Items with `visibility` other than `local` show an "Inherited" or "Synthetic"
chip.

### Thread interactions

| Feature | What the user sees | Upstream | Protocol | Tier |
|---|---|---|---|---|
| Expand a tool row | "Loading output...", "No output.", "Output is no longer available." | `chat/V2ItemInspector.tsx` | `orchestration.getTurnItem {threadId, itemId}`, cached by `updatedAt` | core chat |
| Edit from here | Dialog "Edit from here?" with Cancel, "Revert files too", "Revert and keep changes". The prompt and attachments go back to the composer. Blocked while a run is active | `chat/useTimelineRewind.ts` | `cmd:checkpoint.rollback {scopeId, checkpointId, restoreFiles}`; attachments re-minted with `assets.createUrl` | core chat |
| Fork from this response | New thread from that run, opened when it appears in the shell | `useTimelineRewind.ts`, `cr/operations/commands.ts` | `cmd:thread.fork {sourceThreadId, targetThreadId, sourcePoint: {type: "run", runId}, title}` | power |
| Copy message, copy prompt link | Clipboard | `TimelineAssistantRows.tsx`, `TimelineUserMessageRow.tsx` | client-only | core chat |
| Load earlier | A top divider "Load earlier turns". No auto-load on scroll | `MessagesTimeline.tsx`, `cr/state/threads.ts` | `GET /api/orchestration/threads/:id/history?cursor=...` | core chat |
| Find in thread | `mod+f` bar with "i/N", previous, next. Folded content opens on a match | `chat/ThreadFindBar.tsx`, `useThreadFind.ts` | `orchestration.searchThread`, or `orchestration.searchThreadStream` when `threadFindProgressive` | power |
| Scroll | Follows the bottom within 40 px; "Scroll to end" pill | `chat/useTimelineFollow.ts` | none | core chat |
| Minimap | Rail with "Jump to message", previous and next turn | `chat/TimelineMinimap.tsx` | none | power |
| Citations | Select assistant text, "Cite selection in composer". Renders as a chip that jumps back to the source, loading history if needed | `chat/AssistantSelectionToolbar.tsx`, `AssistantCitationChip.tsx`, `packages/contracts/src/assistantCitations.ts` | text only (a `t3-citation://v1/...` link in the message) | power |
| Activity banners | Child thread waiting ("Open approval", "Open question"), background work with Stop, `/goal resume`, `/goal clear` | `chat/ThreadActivityBanners.tsx` | stop as below | power |
| Usage-limit banner | "Resume at reset", "Cancel auto-resume", "Snooze until reset", "Wake now" | `chat/UsageLimitRecoveryBanner.tsx` | `cmd:thread.metadata.update {limitRecovery}` | power |
| Worktree setup card | "Setting up worktree...", "Worktree ready", failed or cancelled, with cancel, work locally, open terminal | `chat/WorktreeSetupCard.tsx`, `useWorktreeSetup.ts` | `subscribeWorktreeSetup`, `worktreeSetup.cancel`, `cmd:prepared-run.retry` | power |

### Markdown

Upstream: `web/components/ChatMarkdown.tsx`,
`packages/shared/src/markdownPipeline.ts`.

| Feature | Detail | Tier |
|---|---|---|
| GFM | Tables, task lists, sanitised raw HTML (off in user messages), heading ids, RTL | core chat |
| GitHub alerts | Note, Tip, Important, Warning, Caution, each with icon and colour | core chat |
| Code blocks | Shiki highlighting, fence title, language icon, "Copy code", "Wrap lines". Incremental parsing while streaming | core chat |
| Run in terminal | On closed sh, bash, zsh, fish, pwsh fences | desktop-only |
| Tables | "Copy table", expand cells | power |
| Math | `remark-math`, gated by the `mathRenderingEnabled` client setting | power |
| Mermaid | Rendered diagram, "Show code" toggle | power |
| `<details>` | Collapsible, fallback summary "Details" | core chat |
| External links | Favicon; menu with open, copy, "Open in preview" (`preview.open`) | core chat (preview is desktop-only) |
| PR and thread links | PR hover card (`pullRequests.preview`), link to thread | power |
| File links | Basename lookup via `projects.searchEntries` and `filesystem.getMetadata`. Click opens the file panel; menu "Open in {editor}" (`shell.openInEditor`) and reveal | power, editor is desktop-only |
| Images | Direct for http, data and blob URLs; workspace and tool-output images through `assets.createUrl`. Videos, gallery, "Image unavailable" | core chat |
| Codex directives | File citations, follow-ups, artifact templates | power |

## Composer

Main files: `web/components/chat/ChatComposer.tsx`, `useComposerSend.ts`,
`chatViewSend.ts`, `web/components/ChatView.logic.ts`,
`web/composerDraftStore.ts`, `cr/operations/commands.ts`
(`startThreadTurn`), `cr/state/composerDispatch.ts`.

### What a send does

1. Inline images are persisted with `assets.persistChatAttachments` if the
   uploads path wasn't used.
2. Changed thread settings go out first: `cmd:thread.metadata.update`, then
   `cmd:thread.model-selection.set` or `cmd:provider.switch`, then
   `cmd:thread.runtime-mode.set`, then `cmd:thread.interaction-mode.set`.
3. A draft thread's first message is `orchestration.launchThread` with
   `initialMessage` and `workspaceStrategy`. The web never sends `defer_start`.
4. Anything later is `cmd:message.dispatch` with `text`, `attachments`,
   optional `context`, `modelSelection`, `sourcePlanRef`, `dispatchMode` and
   `deliveryIntent`.
5. With `capabilities.serverResolvedCommandContext`, queue means
   `queue_after_active` and everything else is `start_immediately` with
   `deliveryIntent` `auto`, `steer` or `restart`. Older servers make the
   client read `orchestration.getThreadProjection` and pick `steer_active`,
   `restart_active` or `queue_after_active` itself.
6. `message.context` (`packages/contracts/src/composerContext.ts`) is sent
   only with `capabilities.inlineMessageContext`. Otherwise context travels as
   markdown links in the text.

Limits: 120,000 input characters; 200 context records, 16 MB serialised.

### Composer features

| Feature | What the user sees | Upstream | Protocol | Tier |
|---|---|---|---|---|
| Interaction mode | Footer chip "Build" or "Plan". Only shown with the client setting `planModeEnabled` (default off) and a provider that allows it. Shift+Tab, `/plan`, `/default`, `mod+shift+a` | `chat/ComposerFooterControls.tsx`, `useComposerSelection.ts` | `cmd:thread.interaction-mode.set` on next send. Modes are `default` and `plan` | core chat (plan is power) |
| Runtime mode | Supervised (`approval-required`), Auto-accept edits (`auto-accept-edits`), Auto (`auto`), Full access (`full-access`, the default). Filtered by the provider's `supportedRuntimeModes` | `chat/runtimeModeConfig.ts` | `cmd:thread.runtime-mode.set` on next send | core chat |
| Model picker | Left rail: Favorites plus each provider instance. Right: models (favorites first, then user order, then catalogue). Legacy models fold. Fuzzy search across instances. Locked to the session's provider unless it supports handoff | `chat/ProviderModelPicker.tsx`, `ModelPickerContent.tsx`, `web/modelOrdering.ts`, `web/providerModels.ts` | data from `server.getConfig`. Same instance: `cmd:thread.model-selection.set`; other instance: `cmd:provider.switch` (always the former with `serverResolvedCommandContext`) | core chat |
| Favorites, hidden models, order | Star, hide, drag order | `ProviderSettingsPanel.tsx` | `server.updateSettings` with `setModelFavorites`, `setModelsHidden`, `setProviderModelOrder` | power |
| Model options | Reasoning effort, thinking, Fast or Normal, Codex service tier, Claude context window. Trigger label joins values with " · ". "Ultrathink" is a prompt prefix, not an option | `chat/TraitsPicker.tsx`, `TraitsSpeed.tsx`, `cr/modelTraits.ts` | rides in `modelSelection.options` | core chat |
| Multi-model send | New git-project drafts only. Each model gets its own worktree | `chat/useMultipleModelSend.ts` | one `orchestration.launchThread` per model; needs `requiredWorktreeBootstrap` | power |
| `/` commands | Built-in: `/model` (opens picker), `/plan`, `/default`. Provider commands from `ServerProvider.slashCommands` at prompt start, inserted as `/name `. `/compact` only on an empty prompt. Client-handled: `/usage-limits` (local banner), `/feedback` (Codex) | `chat/useComposerMenuItems.ts`, `ComposerCommandMenu.tsx` | provider commands are plain text in `message.dispatch`; `/feedback` calls `provider.uploadFeedback` | core chat |
| `$` skills | Inline `$name` token from `ServerProvider.skills` (enabled, user-invocable). Also listed in `/` as `/skill:name` when `showSkillsInSlashMenu` is on | `web/providerSkillSearch.ts`, `cr/providerSkills.ts` | text plus a `skill` context record | power |
| `@` files and folders | Inserts `[basename](path)` | `cr/state/composerPathSearch.ts` | `projects.searchEntries {cwd, query, limit}`, debounced, cached 15 s | core chat |
| `@` threads | Same trigger, ranked by recency from the shell. Blocked if the thread is on another machine | `cr/composerThreadItems.ts` | no extra RPC; a `thread` context record | power |
| `#` pull requests | Number or text, up to 12 hits. Only with `capabilities.pullRequests` and a repo | composer menu | `pullRequests.list {state: "all", projectId, query}`, `pullRequests.detail` for an exact number | power |
| Context chips | Terminal output, diff and file selections (`review-comment`), preview annotations and elements, citations, threads | `web/components/ContextChip.tsx`, `composerContextPresentation.tsx` | ride in `message.context` | power (preview chips desktop-only) |
| Attachments | Paste, drop, file picker. png, jpeg, gif, webp; HEIC converted; PNG and JPEG downscaled to 2000 px. Strip of thumbnails | `chat/useComposerAttachmentIntake.ts`, `ComposerAttachmentStrip.tsx`, `cr/state/attachments.ts` | with `capabilities.attachmentUploads`: `attachments.createUploadUrl`, then POST bytes to the signed URL (10 min), cleanup via `attachments.delete`. Otherwise inline data URLs then `assets.persistChatAttachments` | core chat |
| File attachments | Non-image files, upload path only, up to `min(fileAttachments.maxUploadBytes, 50 MB)` | same | `attachments.createUploadUrl` | power |
| Large paste | 32 KB or more becomes `pasted-text.txt`. `mod+shift+v` pastes plain | `cr/textPaste.ts` | upload | power |
| Approval panel | Heading by kind ("Command approval", "File change approval", "App access approval" for MCP elicitation), `1/N`, prompt and reason. Buttons from `item.options`: Decline and Approve inline, the rest (Cancel, "Always allow this session", accept always) under a menu. "Provider process is gone" when not resumable | `chat/ComposerPendingApprovalPanel.tsx`, `ComposerPendingApprovalActions.tsx`, `cr/state/threadRequests.ts` | `cmd:runtime-request.respond {requestId, decision, feedback?}`. Typing and sending declines with feedback when `supportsFeedback` | core chat |
| Question panel | Header, `i/N`, question markdown, options with 1 to 9 shortcuts, "Select one or more options." for multi-select. Free text in the main composer is the custom answer. Previous, Next question, Submit answers. X to dismiss | `chat/ComposerPendingUserInputPanel.tsx`, `web/pendingUserInput.ts` | `cmd:runtime-request.respond {requestId, answers, attachmentsByQuestionId?}`; dismiss is `cmd:thread.user-input.dismiss {requestId}` | core chat |
| Plan follow-up | "Plan ready {title}" banner. Primary button "Implement" (empty prompt) or "Refine" (has text). Menu "Implement in a new thread" | `chat/ComposerPlanFollowUpBanner.tsx`, `useProposedPlanSend.ts` | Implement: `cmd:thread.interaction-mode.set {default}`, then `message.dispatch` with text `PLEASE IMPLEMENT THIS PLAN:\n{markdown}` and `sourcePlanRef`. Refine: plan mode, `message.dispatch`. New thread: `cmd:thread.create` then `message.dispatch` | core chat |
| Queue vs steer | Client setting `followUpBehavior`, `queue` (default) or `steer`. Enter does the default, `mod+enter` the other while a run is active. Button label shows "Queue message", "Steer message", "Compact and send", "Resume thread" | `cr/state/composerDispatch.ts` | dispatch mode as above | core chat |
| Queued list | "Queued (n)" banner. Drag or arrow reorder, edit (pencil or `alt+up`), "Steer" or "Send now", remove. Held queue shows Resume | `chat/QueuedRunsControl.tsx`, `useQueuedRunEdit.ts` | `cmd:queued-run.reorder {runId, beforeRunId}`, `cmd:queued-run.edit {runId, text, attachments, context}`, `cmd:queued-message.promote-to-steer {queuedRunId, targetRunId}`, `cmd:queued-run.start {runId}`, `cmd:queued-run.cancel`, `cmd:queue.resume` | core chat |
| Resume thread | After an interrupted or usage-limited run, with an empty prompt | `chat/useComposerResume.ts` | `cmd:message.dispatch {text: "Continue where you left off.", manualContinuationOfRunId, dispatchMode: start_immediately}`, then `cmd:queue.resume` if held | core chat |
| Stop | Red square "Stop generation", tooltip "Interrupt". No default key; Escape doesn't stop | `chat/ComposerPrimaryActions.tsx`, `cr/operations/commands.ts` (`interruptThreadTurn`) | subagent: `cmd:subagent.stop`; with `capabilities.threadStop`: `cmd:thread.stop`; else `cmd:run.interrupt {runId, holdQueue: true}` | core chat |
| Drafts | Per thread in localStorage `t3code:composer-drafts:v1`: text, attachments, contexts, model, modes, workspace choice. No server sync | `web/composerDraftStore.ts` | client-only | core chat |
| Prompt stash | `mod+s` stashes the draft (20 entries). Restore appends | `web/promptStashStore.ts`, `chat/ComposerStashMenu.tsx` | client-only | power |
| Prompt history | Up and down step through this thread's own messages when the prompt is empty | `chat/composerPromptHistory.ts` | client-only | power |
| Rich text | Tiptap editor with code blocks, list continuation; serialises to markdown | `web/components/ComposerPromptEditorTiptap.tsx` | client-only | power |
| Placeholder | Default "Ask anything, @tag files/folders, $use skills, or / for commands". Changes for approvals ("Send feedback to decline this request"), questions, plan refine, no project, no provider | `ChatComposer.tsx`, `web/composerPlaceholder.ts` | client-only | core chat |
| Context window meter | Used and max tokens, compaction threshold, "Compact context". Off by default (`contextWindowMeterEnabled`) | `chat/ContextWindowMeter.tsx`, `web/lib/contextWindow.ts` | from the thread projection (`providerTurns`, `providerThread.contextUsage`); Compact sends `/compact` | power |
| Voice input | Not in the web app. Mobile only, on-device transcription | `cr/voice-input/*` | none | power |

Composer keys:

| Key | Action |
|---|---|
| Enter | Send (setting `sendShortcut`: `enter`, `mod-enter-multiline`, `mod-enter`). Newline on soft keyboards |
| Shift+Enter | Newline |
| mod+Enter | Alternate dispatch while a run is active; background send from a draft |
| mod+alt+Enter | Send and open a new thread |
| Esc | Close the trigger menu |
| Up, Down | Menu navigation, or prompt history |
| Enter, Tab | Pick menu item |
| Shift+Tab | Toggle plan mode |
| mod+s | Stash |
| mod+shift+Enter | Steer the next queued message |
| alt+Up | Edit the latest queued message |
| mod+shift+m, h, e, a, x, g, l | Model picker, host, effort, mode, workspace, branch, previous worktree |

## Thread header and actions

Header: `web/components/chat/ChatHeader.tsx`, `web/components/WorkspaceBreadcrumb.tsx`.

| Feature | What the user sees | Upstream | Protocol | Tier |
|---|---|---|---|---|
| Breadcrumb | "Project › Thread title", parent link for forks, guest-share marker | `ChatHeader.tsx` | shell data | core chat |
| Rename | Double-click title | `ChatHeader.tsx` | `cmd:thread.metadata.update {title}` | core chat |
| Action menu | Same items as the sidebar row menu | `threadActionMenu.logic.ts` | as in the sidebar | core chat |
| Stop session | Detaches each provider session (not the same as Stop run) | `cr/operations/commands.ts` (`stopThreadSession`) | `cmd:provider-session.detach` per session | power |
| Restart agent session | Palette item | `CommandPalette.tsx` | `cmd:provider-session.detach`, then `server.refreshProviders` | power |
| Lineage popover | "Lineage · N running": forks, parent, subagents, transfers. "Merge back to source conversation", "Stop subagent" | `chat/ThreadRelationshipsControl.tsx` | `cmd:thread.merge_back {sourcePoint: {type: "run"}}`, `cmd:subagent.stop`, `orchestration.stopWorkflow` | power |
| Thread details panel | "Workspace" and "Version Control" sections, PR rows | `chat/ThreadDetailsPanel.tsx`, `ThreadDetailsPrRows.tsx` | `subscribeVcsStatus`, `pullRequests.detail`, `pullRequests.checks` | power |
| Git actions | One primary button plus menu: "Commit", "Commit & push", "Commit, push & create PR", "Push", "Push & create PR", "Create PR", "Pull", "Publish repository". Commit dialog with file checkboxes, message and feature-branch option | `web/components/GitActionsControl.tsx`, `GitActionsControl.logic.ts`, `cr/state/vcsAction.ts` | `git.runStackedAction {action: commit, push, create_pr, commit_push, commit_push_pr, commitMessage?, featureBranch?, filePaths?}`, `vcs.pull`, `sourceControl.publishRepository`, `subscribeVcsStatus`, `vcs.refreshStatus` | power |
| Branch toolbar | Local vs worktree, branch picker (search, switch, create), base branch for worktrees, machine selector | `web/components/BranchToolbar.tsx`, `BranchPicker.tsx` | `vcs.listRefs`, `vcs.switchRef`, `vcs.createRef`, `cmd:thread.metadata.update {branch, worktreePath}` | power (showing the branch is core chat) |
| Open in editor | Editor picker plus `mod+o` | `chat/OpenInPicker.tsx`, `web/editorPreferences.ts`, `web/remoteOpen.ts` | local: `shell.openInEditor` (launches on the server host). Remote: client-built deep links from `serverConfig.remoteOpenTargets` | desktop-only (remote links are power) |
| Project scripts | Run a script, add and edit scripts with keybindings, auto-open a preview URL | `web/components/ProjectScriptsControl.tsx`, `web/projectScripts.ts` | edit: `server.updateSettings`, `server.upsertKeybinding`, `server.removeKeybinding`; run: `terminal.open`, `terminal.write`; preview: `preview.open` | power |
| Panel toggles | Terminal drawer, right panel, maximise | `chat/PanelLayoutControls.tsx` | client-only | power |
| Right panel tabs | Kinds `diff`, `files`, `file`, `preview`, `device`, `terminal`, `pull-request`, `pull-requests`. "Open a surface" menu lists Browser, Terminal, Files, Diff, Pull request, Linked pull requests, Device | `web/rightPanelStore.ts`, `web/components/RightPanelTabs.tsx` | per tab, below | power |

## New thread flow

| Feature | What the user sees | Upstream | Protocol | Tier |
|---|---|---|---|---|
| New thread | Reuses an empty draft for the project if there is one, else creates one at `/draft/$draftId`. Carries model and mode from the current thread | `web/hooks/useHandleNewThread.ts`, `web/routes/_chat.draft.$draftId.tsx` | client-only until send (may read `.t3` via `projects.readFile`) | core chat |
| First send | The thread appears and runs | `chat/useComposerSend.ts`, `cr/operations/commands.ts` | `orchestration.launchThread {projectId, title, generateTitle, modelSelection, runtimeMode, interactionMode, workspaceStrategy, initialMessage}`. `workspaceStrategy` is `root`, `existing_worktree` or `worktree {baseRef, branch?, startFromOrigin?}` | core chat |
| Project picker | Palette "New thread in...", project filter, draft hero | `CommandPalette.tsx`, `chat/DraftHeroHeadline.tsx` | client-only | core chat |
| Workspace mode | "Current checkout", "New worktree", "Previous worktree ({branch})". Default from project settings, then the `.t3` file | `web/components/BranchToolbarEnvModeSelector.tsx`, `BranchToolbar.logic.ts` | becomes `workspaceStrategy` | core chat |
| Base branch | Branch list and "start from origin" (default from `newWorktreesStartFromOrigin`) | `web/components/WorktreeBaseBranchPicker.tsx` | `vcs.listRefs` | core chat |
| Machine selector | "Run on", "Auto balance", "Add to another machine" | `BranchToolbarEnvironmentSelector.tsx`, `web/hooks/useLoadBalancedEnvironment.ts` | auto balance reads `server.getHostResources` | power |
| Default model | Draft pick, then project default (`projectSettingsOverrides`, server `defaultModelSelection`, project `defaultModelSelection`), then the current thread's model, then a sticky per-provider choice, then `DEFAULT_MODEL_BY_PROVIDER` in `packages/contracts/src/model.ts` | `useHandleNewThread.ts`, `web/composerDraftStore.ts` | server values from `server.getConfig` | core chat |
| Thread without a project | `mod+alt+n` | `web/hooks/useScratchProject.ts` | `projects.ensureScratch`, then wait for the project in the shell | power |
| Add local folder | Palette "Add project", "Local folder" with an in-palette browser | `CommandPalette.tsx` | `filesystem.browse`, then `projects.mutate {type: "project.create"}` (not `dispatchCommand`) | core chat |
| OS folder picker, WSL folder | Native dialog | `CommandPalette.tsx` | desktop bridge, then `projects.mutate` | desktop-only |
| New project | "New project" by name, optional publish | `web/hooks/useNewProject.ts` | `projects.createNew {name}`, `sourceControl.publishRepository` | power |
| Clone | GitHub and other sources, or a URL. Progress toasts | `CommandPalette.tsx`, `ProjectCloneToastCoordinator.tsx` | `server.discoverSourceControl`, `sourceControl.lookupRepository`, `sourceControl.cloneRepository` or `projectClone.start`, `subscribeProjectClones`, `projectClone.cancel`, `projectClone.retry` | power |
| Add project to another machine | Target machine, SSH or HTTPS, destination, copy chosen ignored files | `web/components/AddProjectToMachineDialog.tsx` | `projects.listIgnoredFiles`, `projects.readFile`, `projectClone.start` | power |
| Prompt links | `/new?repo=&prompt=&provider=&model=&pr=` opens a filled draft, cloning if needed, warns about hidden characters | `web/routes/_chat.new.tsx`, `web/components/promptLinks/*` | `projectClone.start` if needed | power |
| Index route | Opens a draft on the most recent project, or an empty state | `web/routes/_chat.index.tsx` | shell data | core chat |

## Command palette and keybindings

The palette (`mod+k`) is `web/components/CommandPalette.tsx` with
`CommandPalette.logic.ts`. Modes are command, files and content. Other code
opens it with the window event `t3code:open-command-palette`. Tier: core chat
for the shell and thread search, power for the rest.

| Item | Protocol |
|---|---|
| Recent threads, Threads, "In messages" | `orchestration.searchThreads`; picking sends `cmd:thread.visit {opened: true, searchQuery}` |
| New thread in {project}, New thread in..., New thread without a project | client-only, scratch uses `projects.ensureScratch` |
| Find in current thread | `orchestration.searchThread` or `searchThreadStream` |
| Copy prompt link, copy thread reference | client-only |
| Link pull request, show linked pull requests | `cmd:thread.pull-request.link`, `pullRequests.*` |
| Share with guest | `guestShare.create` |
| Restart agent session | `cmd:provider-session.detach`, `server.refreshProviders` |
| Go to file (`mod+p`) | `projects.searchEntries` |
| Search project contents (`mod+shift+f`) | `projects.searchContents` |
| Add project, New project, Open WSL folder | see new thread flow |
| Open pull requests, usage, settings, project settings | navigation |
| Theme, appearance, theme editor | client-only |

Keybindings live on the server in a file at `keybindingsConfigPath`. The
settings page (`web/components/settings/KeybindingsSettings.tsx`) reads them
from `server.getConfig`, writes with `server.upsertKeybinding` and
`server.removeKeybinding`, and hears changes on `subscribeServerConfig`. The
command list is `packages/contracts/src/keybindings.ts`; defaults are
`packages/shared/src/keybindings.ts`. A third-party client can read the
user's bindings from the server and honour them.

Default keys (`mod` is Cmd on Mac, Ctrl elsewhere; "!term" means not when the
terminal has focus):

| Command | Key | When |
|---|---|---|
| `sidebar.toggle` | mod+b | |
| `navigation.back`, `navigation.forward` | mod+[, mod+] | !term |
| `terminal.toggle` | mod+j | |
| `rightPanel.toggle` | mod+alt+b | |
| `threadPanel.toggle` | mod+shift+b | !term |
| `rightPanel.new` | mod+t | right panel open, !term |
| `rightPanel.close` | mod+w | !term |
| `terminal.split`, `terminal.splitVertical` | mod+d, mod+shift+d | terminal focus |
| `terminal.new`, `terminal.close` | mod+n, mod+w | terminal focus |
| `view.reopenClosed` | mod+shift+t | |
| `diff.toggle` | mod+d | !term |
| `preview.toggle` | mod+shift+j | |
| `preview.refresh`, `preview.focusUrl` | mod+r, mod+l | preview focus |
| `preview.zoomIn`, `zoomOut`, `resetZoom` | mod+=, mod+-, mod+0 | preview focus |
| `commandPalette.toggle` | mod+k | !term |
| `filePicker.toggle` | mod+p | !term |
| `projectSearch.toggle` | mod+shift+f | !term |
| `usage.open` | mod+u | !term |
| `theme.select`, `appearance.cycle`, `themeEditor.toggle` | mod+alt+a, mod+alt+shift+a, mod+alt+shift+t | |
| `composer.stash` | mod+s | !term |
| `thread.steerQueuedMessage` | mod+shift+enter | !term |
| `thread.editQueuedMessage` | alt+up | composer focus |
| `composer.sendAlternate` | mod+enter | composer focus, turn running |
| `composer.sendBackground` | mod+enter, mod+alt+enter | composer focus, draft route |
| `composer.sendAndNewThread` | mod+alt+enter | composer focus, not a draft |
| `chat.new` | mod+n, mod+shift+o | !term |
| `chat.newLocal` | mod+shift+n | !term |
| `chat.newWithoutProject` | mod+alt+n | !term |
| `chat.find` | mod+f | !term |
| `modelPicker.toggle` | mod+shift+m | !term |
| `composer.host`, `composer.effort`, `composer.mode` | mod+shift+h, e, a | !term |
| `composer.workspace`, `composer.branch`, `composer.previousWorktree` | mod+shift+x, g, l | !term |
| `pullRequest.copyNumber` | mod+shift+k | !term |
| `modelPicker.previousProvider`, `nextProvider` | mod+shift+up, down | picker open |
| `editor.openFavorite` | mod+o | |
| `thread.previous`, `thread.next` | mod+shift+[, mod+shift+] | |
| `thread.copyReference` | mod+shift+c | !term |
| `thread.settle` | mod+shift+s | !term |
| `thread.pin` | mod+shift+p | !term |
| `thread.undo` | mod+z | !term, not in an editable field |
| `thread.jump.1-9`, `modelPicker.jump.1-9` | mod+1..9 | desktop |
| `usage.cost`, `usage.tokens`, `usage.limits` | c, t, l | usage page |
| `usage.period.day/week/month/quarter` | mod+shift+1..4 | usage page |

No default key: `thread.stop`, `rightPanel.toggleMaximized`,
`composer.cycleHost`, `composer.increaseEffort`, `composer.decreaseEffort`.
Project scripts bind as `script.<id>.run`.

## Notifications

Upstream: `web/components/ThreadNotificationCoordinator.tsx`,
`web/threadNotifications.ts`.

| Feature | What triggers it | Mechanism | Tier |
|---|---|---|---|
| Thread notifications | A thread enters Approval needed, Input needed, Guest request, Thread failed, Usage limit reached, or finishes ("Thread completed", skipped if you stopped it). Setting `notificationMode`: off (default), notifications, sound, both. `inAppNotificationsEnabled` default off | No RPC. Derived from `orchestration.subscribeShell`. Focused window and a different thread: in-app toast with "Open thread". Unfocused: browser `Notification` API (works in a plain browser). Sound via WebAudio | core chat |
| Badge | Count of pending notifications | Browser: canvas favicon with a red circle. Desktop: bridge badge | power (desktop-only on desktop) |
| Screen reader announcer | `aria-live` text for other threads' status changes | `web/components/BackgroundThreadAnnouncer.tsx` | power |
| Slow request toast | "Some requests are slow" | client timing (`SlowRpcRequestToastCoordinator.tsx`) | power |
| Host freeze toast | Why a host dropped | `lastEventLoopStall` from server config (`HostFreezeToastCoordinator.tsx`) | power |
| Clone toasts | Cloning, Cloned, Failed with Retry or Remove | `subscribeProjectClones`, `projectClone.*` | power |
| Desktop update toast | "Update downloaded" | Electron | desktop-only |

`web/browserFaviconStore.ts` is not the unread badge. It caches favicons for
the preview browser.

Mobile push is a different system: the phone registers with the T3 Connect
relay (`POST /v1/mobile/devices`, with preferences such as `notifyOnApproval`,
`notifyOnInput`, `notifyOnCompletion`, `notifyOnFailure`), and the relay
pushes through T3's own APNs credentials to T3's own bundle id. A third-party
watch app can't receive those pushes. It needs its own push path or has to
rely on a live subscription. See `apps/mobile/src/features/agent-awareness/`
and `packages/contracts/src/relay.ts`.

## Diff and review

| Feature | What the user sees | Upstream | Protocol | Tier |
|---|---|---|---|---|
| Turn diff | "Turn N", "Latest turn" | `web/components/DiffPanel.tsx`, `web/diffPanelStore.ts` | `orchestration.getTurnDiff {threadId, fromTurnCount, toTurnCount, ignoreWhitespace}`; `orchestration.getFullThreadDiff {threadId, toTurnCount}` when `fromTurnCount` is 0 | core chat |
| Branch and uncommitted diff | "Changes" (branch vs a base ref, "Remote only"), "Uncommitted" | `DiffPanel.tsx` | `review.getDiffPreview {cwd, baseRef?, ignoreWhitespace}`, `vcs.listRefs`, `subscribeVcsStatus` | power |
| Per-file patch and full context | Lazy per file, 4 at a time; expand to full file | `web/components/diffs/useReviewFilePatches.ts`, `cr/state/review.ts` | `review.getDiffPreview` with a file filter, `review.getDiffFileContents` | power |
| Layout | Stacked or split, ignore whitespace, file tree, collapse | `web/components/DiffPanelShell.tsx`, `diffs/DiffFileTree.tsx` | none | power |
| Review comments | Comment on diff lines; they go into the composer as `review-comment` context | `diffs/AnnotatableCodeView.tsx`, `web/reviewCommentContext.ts` | none until `message.dispatch` | power |
| Open file from diff | Opens in the Files tab, or the editor; copy path | `web/diffFileActions.ts` | `projects.readFile`, `shell.openInEditor` | power |
| Changed-files card | In the chat, under the reply | `chat/ChangedFilesTree.tsx` | checkpoint data | core chat |

## Terminal panel

Needs `terminal:operate` to type and `terminal:read` to watch. Upstream:
`web/components/ThreadTerminalDrawer.tsx`, `PersistentThreadTerminals.tsx`,
`web/components/terminal/*`, `cr/state/terminal.ts`.

| Feature | What the user sees | Protocol | Tier |
|---|---|---|---|
| Drawer and tabs | Several terminals, horizontal and vertical splits, labels | `terminal.open`, `terminal.resize`, `terminal.write` | power |
| Live output | Attached session that survives navigation | `terminal.attach` (stream, retried with `resume: true`); read-only `terminal.observe`; list via `subscribeTerminalMetadata` | power |
| Close | Falls back to typing `exit` | `terminal.close` | power |
| Context menu | "Add to chat" (terminal context chip), Copy, Paste, Select all, "Jump to latest" | none | power |
| Links | URLs open in the preview, file paths in Files or the editor | `preview.open`, `projects.readFile` | power |

The web app never calls `terminal.clear`, `terminal.restart` or
`subscribeTerminalEvents`.

## Pull request views

Gated on `capabilities.pullRequests`. Upstream: `web/routes/_chat.pull-requests.tsx`,
`web/components/pullRequest/*`, `cr/state/pullRequests.ts`. All power.

| Feature | What the user sees | Protocol |
|---|---|---|
| PR list page | Scope (All, Reviewing, Authored), state (All, Open, Closed, Merged), sorts (Merge readiness, Blocked on me, Recently updated, ...), filters (drafts, review, checks, project), stats | `pullRequests.list`, `pullRequests.listStats`, `pullRequests.filterOptions`, `pullRequests.subscribeRefreshes`, `pullRequests.invalidate` |
| Summary tab | Editable title and description, labels, reviewers, reactions | `pullRequests.detail`, `pullRequests.update`, `pullRequests.labelCandidates`, `pullRequests.setLabels`, `pullRequests.reviewerCandidates`, `pullRequests.requestReviewers`, `pullRequests.setReaction`, `pullRequests.updateComment` |
| Timeline tab | Activity, comment form | `pullRequests.activity`, `pullRequests.comment` |
| Code tab | Diff, review threads, reply, resolve, files viewed, change breakdown, submit review | `GET /api/pull-requests/diff`, `pullRequests.diffFileContents`, `pullRequests.threadComments`, `pullRequests.replyToThread`, `pullRequests.setThreadResolution`, `pullRequests.filesViewed`, `pullRequests.setFilesViewed`, `pullRequests.changeBreakdown`, `pullRequests.submitReview` |
| Guide tab | AI review guide and guided tour | `pullRequests.reviewGuide`, `pullRequests.generateReviewGuide` |
| Actions | Merge, Ready, Draft, Close, Reopen, delete or update branch, auto-merge, revert, approve workflows; stacks | `pullRequests.runAction {action}`, `pullRequests.stack` |
| Checks | Popover and pill, adaptive polling | `pullRequests.checks` |
| Link previews | Hover cards, linked threads | `pullRequests.preview`, `pullRequests.linkedThreads` |
| Thread and PR | Link, unlink, watch (the agent follows checks and comments) | `cmd:thread.pull-request.link`, `cmd:thread.pull-request.unlink`, `cmd:thread.pull-request.watch {watching}` |
| Check out a PR as a thread | Local or worktree | `git.resolvePullRequest {cwd, reference}`, `git.preparePullRequestThread {reference, mode, threadId?}` |

`pullRequests.summary`, `routing`, `routingIdentity` and `reportState` are
internal to `cr/state/pullRequestRouting.ts`. No component calls them.

## Preview browser

Two backends. The Electron webview (desktop-only) is used when
`window.desktopBridge.preview` exists. The server-side browser is used when
the server reports `capabilities.serverBrowser`, and streams over the
WebSocket `/api/preview-stream`, so it works in a plain browser. Needs
`preview:operate`. Upstream: `web/components/preview/*`,
`cr/preview/serverBrowserStream.ts`, `web/browser/ServerBrowserSurface.tsx`.
All power unless marked.

| Feature | What the user sees | Protocol |
|---|---|---|
| Browser tab | URL bar, back, forward, reload, zoom, menu. Control states "You have control", "Agent has control", "Read-only", "Watching" | `preview.open`, `preview.close`, `preview.adjust`, `preview.resize`; navigation goes over the stream or bridge |
| Sessions and events | | `preview.list`, `subscribePreviewEvents` |
| Empty state | Discovered local dev servers, recent URLs | `subscribeDiscoveredLocalServers`, `preview.open` |
| Agent-opened previews | Agent cursor overlay | `subscribePreviewEvents` |
| Mini player | Picture in picture | stream or bridge |
| Profiles and status | Desktop reports profiles and status; "Clear profile" | `preview.reportStatus`, `preview.reportProfiles`, `preview.clearProfile` (desktop-only) |

`preview.navigate` and `preview.refresh` exist but the web app doesn't call
them.

## Devices

iOS simulator and Android emulator streaming, through a device hub on the
server host. Upstream: `web/components/device/*`, `cr/state/device.ts`. All
power.

| Feature | Protocol |
|---|---|
| Pick host and device, boot, close, power off | `device.list`, `device.open`, `device.close {shutdown?}`, `subscribeDeviceState` |
| Live stream with touch | HTTP and WebSocket to `/api/device-hub/vendor/serve-sim` or `serve-emu` |
| Controls (appearance, text size, orientation, location, permissions, open URL, launch app, shake, push) | `device.detail`, `device.action {type}` |
| Host setup and tests | `device.configure`, `device.testHost`, `device.easAccount` |

## Files

Needs `filesystem:read` (and `filesystem:write` to save). Upstream:
`web/components/files/*`. All power.

| Feature | Protocol |
|---|---|
| File tree, filter, drag a file into the composer | `projects.listEntries`, `projects.searchEntries` |
| Viewer and editor (source, markdown, CSV, audio, images, PDF, HTML), save | `projects.readFile`, `projects.writeFile`, `assets.createUrl`, `shell.openInEditor`, `preview.open` |
| File metadata | `filesystem.getMetadata` |
| Project content search | `projects.searchContents` |

## Settings pages

Settings split by key. Keys in `ServerSettings` (`packages/contracts/src/settings.ts`)
go to `server.updateSettings {patch}`. Everything else is a client setting in
localStorage. A few server keys are pushed to every connected environment:
`continueThreadsAfterServerUpdate`, `sidebarAutoSettleAfterDays`,
`sidebarAutoSettleOnMerge`, `autoResumeLimitedThreads`,
`snoozeLimitedThreads`, `newWorktreesStartFromOrigin`,
`sourceControlWritingStyle`, `textGenerationModelSelection`
(`cr/state/sharedSettings.ts`).

The settings that change chat behaviour are:

- Server: `defaultModelSelection`, `defaultRuntimeMode`,
  `defaultThreadEnvMode`, `responseStreamingMode`,
  `textGenerationModelSelection`, auto-settle keys, `autoResumeLimitedThreads`,
  `snoozeLimitedThreads`, `idleCompaction`, `favorites`,
  `providerModelPreferences`.
- Client: `notificationMode`, `inAppNotificationsEnabled`, `sendShortcut`,
  `followUpBehavior`, `planModeEnabled`, `showSkillsInSlashMenu`,
  `timestampFormat`, the confirm toggles.

| Page | What's on it | Protocol | Tier |
|---|---|---|---|
| General | Project defaults (model, workspace, permissions, submodules, auto pull, merge method). Organisation (grouping, sorting, auto-settle, auto-resume, idle compaction). Behaviour (notifications, timestamps, diff options, streaming mode, send shortcut, follow-up behaviour). Text generation model and title instructions. About and updates | `server.updateSettings`; `server.updateServer`, `server.updateServerWithProgress`, `server.commitDesktopUpdate` | core chat for model, permissions, workspace, notifications, follow-ups; rest power |
| Appearance | Themes, contrast, fonts, chat width, math, diff colours, motion | mostly client; `defaultTheme`, `environmentIcon` are server keys | power |
| Archived | Archived threads per environment, Unarchive, Delete | `orchestration.getArchivedShellSnapshot`, `cmd:thread.unarchive`, `cmd:thread.delete` | core chat |
| Beta | Working shelf, guest links (`guestSharingEnabled`) | `server.updateSettings` | power |
| Connections | Saved environments, authorised clients, pairing links, WSL, Tailscale, network exposure, load balancing | `subscribeAuthAccess`; HTTP `/api/auth/clients`, `/api/auth/clients/revoke`, `/api/auth/clients/revoke-others`, `/api/auth/pairing-links`, `/api/auth/pairing-links/revoke`, `/api/auth/pairing-token` | pairing is core chat; WSL, Tailscale, exposure desktop-only |
| Diagnostics | Processes, resource history, traces, OTLP export | `server.getProcessDiagnostics`, `server.getProcessResourceHistory`, `server.getTraceDiagnostics`, `server.signalProcess`, `subscribeResourceTelemetry`, `server.getResourceTelemetryHistory`, `server.retryResourceTelemetry`, `server.getHostResources`, `server.checkOtlpEndpoint` | power |
| Integrations | Preview browser defaults and profiles, device hub, agent browser and device access | `device.configure`, `device.list`, `device.testHost`, `device.easAccount`, `subscribeDeviceState`, `preview.clearProfile`, `server.updateSettings` | power, partly desktop-only |
| Keybindings | Editor | `server.upsertKeybinding`, `server.removeKeybinding` | desktop-only in practice |
| Projects | Name, icon, checkouts, project defaults, scripts, delete | `projects.mutate {type: "project.update" or "project.delete"}`, `server.updateSettings`, keybinding RPCs | power |
| Providers | Instances, install, sign-in, models, ACP registry, usage sources | `server.refreshProviders`, `server.updateProvider`, `server.updateSettings`; `provider.auth.start`, `respond`, `complete`, `cancel`, `logout`, `subscribe`; `provider.chatgpt.*`; `provider.install.start`, `cancel`, `subscribe`, `remove`; `server.*AcpRegistry*` | status and model list core chat; setup power |
| Scheduled tasks | Next section | | power |
| SnapShot | Screenshot shortcut, sound, flash | client-only | desktop-only |
| Source control | gh and glab auth status, hosts, tokens, writing style, branch naming, fetch interval | `server.discoverSourceControl`, `server.updateSettings` | power |
| Storage | Worktree directory, cleanup rules, "Delete now", last run | `server.runStorageCleanup`, `server.getStorageCleanupReport`, `server.getDatabaseStorage`, `server.updateSettings` | power |
| Open source licenses | Static | none | |

Other routes: `/pair` (pairing, HTTP only, core chat), `/connect` (Clerk sign-in
for the CLI), `/connect-agent` (MCP OAuth approve or deny), `/welcome`
(first-run wizard using `agentSessions.scan` and `agentSessions.import`,
power), `/usage` (below), `/guest` (below). `ConnectionStatusDot.tsx` colours a
dot by connection phase with no RPC; it's cheap to copy and worth it.

## Scheduled tasks

Upstream: `web/components/settings/ScheduledTasksSettings.tsx`,
`packages/contracts/src/scheduledTask.ts`. Tier: power.

The list shows each task's title, schedule, "Next run", last run status and
error, an enable switch, and a menu with Edit, Deliveries (webhooks only), Run
now and Delete. The editor has name, project, workspace, prompt, model,
machine, enabled, and a schedule that's one of a fixed time with weekdays, an
interval (60 s minimum), or a webhook (URL with rotate, optional signature,
dedupe, coalesce, up to 16 filters, `{{body.x}}` templating).

| Action | Protocol |
|---|---|
| Live list | `scheduledTasks.subscribe` (the web doesn't call `scheduledTasks.list`) |
| Save | `scheduledTasks.upsert` |
| Enable | `scheduledTasks.setEnabled` |
| Delete | `scheduledTasks.delete` |
| Run now | `scheduledTasks.runNow` |
| Rotate webhook URL | `scheduledTasks.rotateWebhookToken` |
| Deliveries | `scheduledTasks.listWebhookDeliveries`, `scheduledTasks.getWebhookDelivery` |

Webhooks arrive at `/api/hooks/:hookId/:token`.

## Usage page

`web/routes/usage.tsx`, `web/components/usage/*`. Tier: power.

Per-provider and per-model tokens and cost over a time window, with price
overrides and pooled rate limits. Uses `server.getUsageSummary` (needs
`diagnostics:read`), `server.refreshProviders`, `server.updateSettings` for
price overrides, and `provider.consumeResetCredit` for the reset-credit
button.

## Guest share

Tier: power. Gated on `guestSharingEnabled` and a running T3 Connect tunnel.

| Feature | Protocol |
|---|---|
| "Share with guest" link | `guestShare.create {threadId}`; the server then issues `thread.guest-share.open` itself |
| Stop sharing | `cmd:thread.guest-share.close` |
| Pending guest messages: Send or Dismiss | Send: `cmd:message.dispatch {guestMessageId, text: "", dispatchMode: {type: "queue_after_active"}}`. Dismiss: `cmd:thread.guest-message.dismiss` |
| Guest page `/guest?share=...` | HTTP only under `/api/guest` (`/session`, `/item`, `/command-output` as SSE, `/render`) with a per-share cookie |

## What the mobile app treats as essential

`apps/mobile` (Expo) is close to a full client, not a thin one. That makes it
less useful as a "minimum" guide than I hoped, but what it renders first and
what it skips still says something.

Mobile has, as core:

- Home thread list from `orchestration.subscribeShell`, search via
  `orchestration.searchThreads`, swipe actions for archive, pin, settle,
  snooze (`features/home/`).
- Thread view from `orchestration.subscribeThread` with every item type,
  `orchestration.getTurnItem` to expand, live command output via
  `orchestration.subscribeCommandOutput` (`features/threads/ThreadFeed.tsx`,
  `src/lib/threadActivity.ts`). Rows: message, activity, activity group, work
  toggle, run fold, thinking.
- Approvals as "Allow once", "Allow session", "Decline"
  (`PendingApprovalCard.tsx`), questions (`PendingUserInputCard.tsx`), secret
  requests (`SecretRequestCard`), all through `cmd:runtime-request.respond`
  or `secrets.answerRequest`.
- Composer with an outbox (`src/state/use-thread-outbox-drain.ts`): send,
  queue, steer, stop, attachments via `attachments.createUploadUrl`, `/`, `$`
  and `@` menus, dictation, queue editing.
- New task via `orchestration.launchThread`, with branch pickers
  (`vcs.listRefs`).
- A thread settings sheet for model, runtime mode and plan mode.
- Pairing by QR or host plus code.

Mobile skips: the keybindings editor, SnapShot, browser profiles and device
host setup, trace and process diagnostics, storage cleanup, the ACP registry
and adding provider instances, source-control token and host settings,
authorised-client management, WSL and Tailscale, the theme editor, the welcome
wizard, and the webhook deliveries viewer. It has no UI for
`defaultModelSelection`; it reads it from server config.

For a watchOS client, my cut is smaller than mobile's:

1. Pairing over HTTP, then the socket.
2. `orchestration.subscribeShell` for the list, with status chips (Approval,
   Input, Working, Failed, Done) and `cmd:thread.visit`.
3. `orchestration.subscribeThread`, showing user and assistant messages, the
   folded "Worked for" row, and the live "Working for" row. No tool detail.
4. `cmd:runtime-request.respond` for approve and decline, and for questions
   with options. `secrets.answerRequest` decline only (nobody pastes a secret
   on a watch).
5. `cmd:message.dispatch` for dictated replies, and Stop.
6. Settle and snooze from the list.
7. Optional: a new thread through `orchestration.launchThread` with the
   server's default model and runtime mode.

Push is the gap. The official push path goes through T3's relay and APNs
credentials, so a watch client has to keep a subscription alive or bring its
own notifier.

## PoC checklist

Ordered by what I'd build next. Ticked items exist in `poc/public/app.js` and
`t3.js` today.

### Core chat, done

- [x] Environment list with online status, connect, reconnect on tab focus
- [x] Thread list from `orchestration.subscribeShell`: pinned section, project
      groups sorted by latest user message, settled and snoozed folded per
      project, unread marker, running and waiting dots, last error note
- [x] Title filter plus message search via `orchestration.searchThreads`
- [x] Archived view via `orchestration.getArchivedShellSnapshot`, with
      unarchive and delete
- [x] Thread menu: rename, regenerate title, pin, unpin, settle, unsettle,
      snooze 1 hour and tomorrow 9:00, unsnooze, mark unread, archive, delete
- [x] `cmd:thread.visit` on open
- [x] Thread view from a bounded `orchestration.subscribeThread` snapshot,
      updated in place from `turn-item.updated`, `run.*` and
      `message.updated`
- [x] History paging through `/api/orchestration/threads/:id/history`
- [x] "Worked for Xs" folding, "You stopped after Xs", errors keep a run open
- [x] Thinking rows, live "Thinking" and "Thought"
- [x] Tool rows that fetch detail with `orchestration.getTurnItem` on expand
- [x] Image send via `attachments.createUploadUrl` and a signed POST (paste,
      drop, picker)
- [x] Stop via `cmd:thread.stop` or `cmd:run.interrupt`
- [x] Yes and no approvals via `cmd:runtime-request.respond` (Approve, For
      session, Decline)
- [x] Questions with options, free text, multi-select, dismiss
- [x] `/` commands and `$` skills from `server.getConfig`, `@` files from
      `projects.searchEntries`
- [x] Send with deferred mode changes, `deliveryIntent: "auto"` when the
      server resolves context, Alt+Enter to queue
- [x] Queue panel: send now (promote to steer), edit, cancel, resume
- [x] New thread from a draft via `orchestration.launchThread` (project root
      only)
- [x] Model picker with providers, hidden models, option descriptors,
      ultrathink as a prompt prefix; plan mode toggle; runtime mode select
- [x] Plan bar with "Implement plan" (`sourcePlanRef`)
- [x] Fork (`sourcePoint: latest_stable`), diff of last turn and whole thread

### Core chat, next

1. [ ] Approval buttons from `item.options` instead of a fixed set, so MCP
       elicitations, `acceptAlways` and option warnings work. Add the
       per-kind heading and the "Provider process is gone" state.
2. [ ] `secret_request` card with `secrets.answerRequest` (save and decline).
       It blocks a run until answered, like approvals.
3. [ ] Real markdown: GFM tables and lists, links, code highlighting with
       copy, GitHub alerts, `<details>`. The PoC renders fences, inline code,
       bold and headings only.
4. [ ] Images in messages: render user attachments and markdown images, minting
       URLs with `assets.createUrl`.
5. [ ] Live command output via `orchestration.subscribeCommandOutput` while a
       command row is open (when `commandOutputStreaming`).
6. [ ] Live "Working for Ns" row with a ticking timer, and the `work-live`
       row for the trailing tool.
7. [ ] Tool group summaries ("Ran 4 commands, Read 3 files") instead of one
       row per tool inside an open fold.
8. [ ] Rows the PoC hides or flattens: `notification` (shown upstream),
       `fork` and `run_interrupt_result` dividers, compaction token counts,
       `subagent` cards with status and a link to the child thread.
9. [ ] "Resume thread" after an interrupted or usage-limited run
       (`manualContinuationOfRunId`, then `queue.resume`).
10. [ ] "Edit from here" via `cmd:checkpoint.rollback`, restoring the prompt.
11. [ ] Status chips with the upstream priority (Approval, Input, Working,
        Failed, Limited, Done) instead of two dot colours.
12. [ ] Snooze presets matching upstream ("In 3 hours", "This evening",
        "Next week", custom).
13. [ ] Notifications from the shell stream with the browser `Notification`
        API, gated on a setting.
14. [ ] Workspace choice on new threads: new worktree with a base branch
        (`vcs.listRefs`), existing worktree.
15. [ ] Composer drafts that survive reloads (per thread, localStorage).
16. [ ] Error rows with retry info and the `prepared-run.retry` button.
17. [ ] Plan follow-ups: Refine, and Implement in a new thread.

### Power, later

- [ ] Queue reorder (`queued-run.reorder`), `queued-run.start` "Send now"
- [ ] Fork from a specific response (`sourcePoint: {type: "run", runId}`)
- [ ] Find in thread (`orchestration.searchThread`)
- [ ] Undo toasts for settle, snooze, unpin, archive
- [ ] Auto-settle per thread (`thread.auto-settle.set`)
- [ ] Todo list as a composer badge rather than an inline row
- [ ] Context window meter from `providerThread.contextUsage`
- [ ] `#` PR mentions, `@` thread mentions
- [ ] Diff panel with per-file view, branch and uncommitted scopes
      (`review.getDiffPreview`)
- [ ] Git actions (`git.runStackedAction`, `vcs.pull`) and branch picker
- [ ] Terminal (`terminal.attach`, `write`, `resize`), needs `terminal:operate`
- [ ] PR list and detail (`pullRequests.*`)
- [ ] Scheduled tasks list with enable and run now
- [ ] Settings: default model, runtime mode and workspace, follow-up behaviour
- [ ] Files tab, server-side preview browser, usage page, guest share

### Skip

- Editor launch, OS folder picker, WSL, Tailscale, SnapShot, desktop update
  toasts, the Electron preview webview, keybinding editing. These need the
  desktop app or the server host's screen.
