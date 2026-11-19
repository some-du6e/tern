# 06 · Conversation timeline

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
This document covers everything drawn inside the chat lane of a thread: the virtualized list and its header and footer, user and assistant messages, the full markdown renderer (typography, code blocks, tables, links, chips, media, alerts, details), the work log ("Worked for", tool rows, reasoning, live activity), lifecycle dividers, subagent cards, plans, errors, approvals as they appear in the list, and the overlays that sit on the timeline (scroll to end, minimap, the cite toolbar, the image dialog). The canvas, the lane placement rule, the scroller's padding and its top fade are in [03-shell.md](03-shell.md) §2.2, §3.5 and §5.4 and are only referenced here. Tokens are in [01-tokens.md](01-tokens.md), primitives (Button, Tooltip, Menu, Popover, Dialog, ScrollArea, Badge, Skeleton, Spinner, MiddleTruncate) in [02-primitives.md](02-primitives.md).

Conventions, as in 01 to 04:

- Lengths are CSS px at the default 16px root (1 CSS px = 1 Gio dp). Values written in `rem` or `em` scale with the interface font size; markdown block margins are `0.65rem` = 10.4px, so they scale too.
- Colours are sRGB hex as Chrome resolves them, `#rrggbbaa` when translucent (RGB exact, not premultiplied). Where a translucent colour sits on a known surface, the painted result is given as "painted". The timeline is outside `[data-app-sidebar]`, so it always uses the `:root` palette (01-tokens §3.1), never the sidebar scope.
- **M** = measured live (computed style or box, both schemes unless a row says light only), **P** = the real class list evaluated on a probe element inside a live timeline lane or `.chat-markdown` (for elements no thread on the account showed), **S** = source only.
- Coordinates are viewport px at 1440×900 @2x, sidebar expanded, thread details card inline (lane x 380..1116, 736 wide; assistant text column x 384..1112).
- Icons are lucide. T3 renders lucide at stroke 1.75 (01-tokens §3.9), **except** icons that carry Tailwind `stroke-2`: those stay at 2px. Every work-log icon and the folded-message icons are `stroke-2` (measured 2px). Each icon below names its lucide icon, its box size and its stroke.
- Text renders in Inter here; macOS renders SF Pro. Sizes, weights, line heights and letter spacing are font independent; widths of text boxes are not.

**The measured session was read-only.** The pairing scope was `orchestration:read` plus terminal and preview operate, with no filesystem read scope. T3 showed "Image unavailable · ... This connection cannot read host files" for every workspace image (agent-viewed images, markdown images from the workspace), so the loaded image state of those is P/S, while the failure state is M. Nothing was sent, approved, stopped, settled or edited.

## 1. Scope and sources

Paths are relative to the T3 repo; `c/` is `apps/web/src/components`.

| File | Lines | What it gives |
|---|---|---|
| `c/chat/MessagesTimeline.tsx` | 143–162 | per-thread list state, list header (`h-3 sm:h-4` or the 24px fade spacer) |
| | 163–191 | `TimelineListFooter`: composer inset spacer and the 16px end spacer, growth callback |
| | 194–208 | `maintainScrollAtEnd` options (animated only while working) |
| | 428–475, 482–594 | disclosure state per thread (runs, work groups, attempts), reveal-after-expand, toggles |
| | 596–631 | an interrupted run opens its fold, a newer run closes the previous one |
| | 633–707 | tool call visibility, find and citation fold opening |
| | 755–767 | approvals not matched to a row go to the footer |
| | 778–784, 1384–1404 | empty state ("Send a message to start the conversation.") |
| | 857–881 | rows kept mounted (selection, first and last message, cited row, fullscreen app, playing audio) |
| | 895–911, 1423–1468 | anchored end space, LegendList props (estimated size 90, `maintainScrollAtEndThreshold` 1) |
| | 1188–1219 | list header: parent thread divider ("Subagent of", "Context from"), history control |
| | 1336–1355 | row frame: `messages-timeline-row-frame` > `chat-content-lane overflow-x-clip` |
| | 1357–1377 | footer content and the unmatched approval block |
| | 1406–1490 | viewport (`TooltipScrollDismissArea`), selection toolbar, list, minimap |
| | 1493–1523 | `TimelineHistoryControl` ("Load earlier turns") |
| | 1529–1573 | native selections pin their rows |
| | 1581–1587 | item types for size estimates |
| | 1589–1674 | `TimelineRowContent`: per-kind bottom padding, `group/assistant`, data attributes |
| | 1698–1748 | context compaction divider |
| | 1750–1803 | deferred history loading for a fold, `WorkHistorySkeleton` |
| | 1805–1874 | turn fold and superseded-attempt fold rows |
| `c/chat/MessagesTimeline.logic.ts` | 97–280 | single tool label, display label, live label, entry visibility |
| | 281–464 | minimap geometry, follow re-arm (40px), gutters |
| | 494–644 | the 16 row kinds |
| | 680–721 | assistant copy state, terminal assistant message per response |
| | 852–866, 950–977 | unsettled run, active response runs |
| | 1022–1241 | turn folds: what folds, "Worked for", "You stopped after", "Continued for" |
| | 1243–1334 | assistant meta row moved below trailing tool groups |
| | 1444–1490 | tool call visibility filter (`showAll`, `hideSuccessful`, `hideAll`, `zen`) |
| | 1524–2323 | `deriveMessagesTimelineRows`: live tail, working and thinking rows, compaction, work grouping, messages, user grouping |
| `c/chat/WorkLog.tsx` | 14–36, 38–44 | `WorkLogBlock` trailing space, `WorkLogList` |
| | 53–126 | `WorkLogLine`, `WorkLogButton`, `WorkLogRow` |
| | 128–170 | `WorkLogDetails` (`text`, `panel`, `media`), live-edge follow |
| `c/chat/TimelineRowParts.tsx` | 7–55 | sr-only author heading, `TimelineRowTimestamp`, `ActivityShimmerOverlay` |
| `c/chat/TimelineUserMessageRow.tsx` | 97–102 | media edge ring, chip ink inside bubbles |
| | 104–178, 180–250 | file card, video attachment |
| | 252–280 | fresh session and model switch dividers, workflow opening, automation dispatch |
| | 283–410 | automation message, `FoldedMessageHeader` |
| | 412–428 | user message status badge |
| | 430–1063 | the user row: media layout, file rows, actions, bubble |
| | 1065–1128 | action reveal, steer marker, "Edit from here" |
| | 1130–1229 | long message collapse, user markdown body |
| `c/chat/TimelineUserMessageRow.logic.ts` | 5–7 | "Pasted text" label |
| `c/chat/TimelineAssistantRows.tsx` | 34–97 | assistant row |
| | 99–294 | meta row: copy, fork, reply, timestamp, status badge |
| | 296–362 | plan, HTML render and MCP app rows |
| | 366–449 | changed files section |
| `c/chat/TimelineWorkGroupRows.tsx` | 59–119 | `WorkGroupSection` |
| | 121–287 | working timer, expanded group list (max height, fades, follow) |
| | 291–372 | working row, background setup chip, compacting label |
| | 374–505 | thinking row, live activity row and content |
| | 507–654 | live work row, group header |
| `c/chat/TimelineWorkGroupRows.logic.ts` | 1–20 | working timer format |
| `c/chat/TimelineWorkEntryRow.tsx` | 73–145 | thread-read labels, expanded body class, reasoning trace |
| | 155–206 | notification link, viewed images row |
| | 208–608 | work entry row: error and usage-limit branch, icon and heading colours, expansion |
| | 610–670 | answered question history |
| `c/chat/TimelineWorkEntryRow.logic.ts` | 12–193 | summary icons, tone icons, expanded body text, entry icon |
| | 198–260 | thought preview (8192 source chars, 2048 shown) |
| `c/chat/TimelineToolActivityIcon.tsx` | 30–276 | failure tint, icon names, browser glyph, image icons, muted treatment |
| `c/chat/V2ItemInspector.tsx` | 43–416 | expanded tool bodies per item type |
| `c/chat/CommandOutputPanel.tsx` | 12–82 | command output streaming panel |
| `c/chat/ShellCommandBlock.tsx` | 11–63 | highlighted shell command |
| `c/chat/TimelinePendingApprovalRow.tsx` | 15–122 | approval row |
| `c/chat/ComposerPendingApprovalActions.tsx` | 26–135 | inline approval actions |
| `c/chat/TimelineV2EventRow.tsx` | 19–163 | event rows, subagent group |
| `c/chat/TimelineV2EventRow.logic.ts` | 4–34 | subagent group timing |
| `c/chat/TimelineSystemDivider.tsx` | 7–85 | system divider |
| `c/chat/V2LifecycleRow.tsx` | 70–169, 171–183, 215–545, 602–674 | lifecycle rows, status dots, subagent avatar and link, new session, model switch, handoff |
| `c/chat/AgentElapsed.tsx` | 6–50 | elapsed formats |
| `c/chat/ProposedPlanCard.tsx` | 41–256 | plan card |
| `c/chat/ThreadNow.tsx`, `c/chat/ThreadStatusLine.tsx` | 1–103; 1–27 | footer lines |
| `c/chat/ThreadActivityBanners.tsx` | 171–268 | background work line |
| `c/chat/BackgroundWorkDetails.tsx` | 21–97 | background work dialog |
| `c/chat/TimelineOverlays.tsx` | 26–55 | scroll to end button |
| `c/chat/TimelineMinimap.tsx`, `c/chat/timelineMinimapItems.ts` | 32–308; 17–117 | minimap |
| `c/chat/AssistantSelectionToolbar.tsx` | 23–216 | cite toolbar |
| `apps/web/src/lib/selectionActions.ts` | 6–40 | toolbar position |
| `c/chat/AssistantCitationChip.tsx` | 59–176 | quoted response card |
| `c/ChatMarkdown.tsx` | 287–329 | artifact template card |
| | 333–348 | ordered list gutter for wide markers |
| | 350–390 | GitHub alert presentations |
| | 501–993 | markdown state: copy, media, link targets, file links |
| | 1011–1030 | heading level offset, double click selects inline code |
| | 1032–1732 | element overrides (p, blockquote, ol, li, input, a, code, img, table, details, pre) |
| | 1807–1895 | root element, streaming blocks |
| `c/chat/MarkdownCodeBlock.tsx` | 42–127 | code block frame and title |
| | 129–266 | actions (wrap, run, view toggle, copy) |
| | 275–370 | highlighting and its cache |
| `c/chat/MarkdownCodeBlock.logic.ts` | 8–66 | fence language, fence title, closed fence |
| `c/chat/HighlightedCodeLines.tsx` | 1–46 | line-preserving highlighted output |
| `c/chat/MarkdownTable.tsx` | 15–70 | table container and copy menu |
| `c/chat/MarkdownDetails.tsx` | 14–65 | details disclosure |
| `c/chat/MarkdownExternalLink.tsx` | 21–119 | favicon and breakable link text |
| `c/chat/MarkdownFileLink.tsx` | 53–416 | file link chip |
| `c/chat/MarkdownThreadLink.tsx` | 1–71 | thread link chip |
| `c/chat/MarkdownMath.tsx`, `c/chat/MarkdownMermaidCodeBlock.tsx` | 14–79; 14–84 | math and mermaid |
| `c/chat/ChatMarkdownMedia.tsx` | 24–448 | markdown images, video, placeholders, asset images |
| `c/chat/ExpandedImageDialog.tsx` | 34–290 | image dialog |
| `c/chat/MessageCopyButton.tsx` | 12–60 | copy button |
| `c/ContextChip.tsx` | 8–84 | chip variants and label |
| `c/chat/useTimelineFollow.ts` | 59–548 | follow mode, pill debounce, anchoring, scroll to end |
| `c/chat/timelineViewport.ts` | 65–283 | navigation signal, follow transitions |
| `c/chat/pageScrollController.ts` | 1–314 | PageUp/PageDown from the composer |
| `c/chat/timelineScrollAnchoring.ts` | 38–151 | anchor offset 24, overflow, disclosure reveal, remembered positions |
| `c/chat/useTallEnoughToFold.ts` | 5–28 | 2 × 11rem fold threshold |
| `c/composerFooterLayout.ts` | 127–142 | scroll-to-end clearance |
| `c/ChatView.tsx` | 3105–3108, 3502–3508 | scroll-to-end summary and placement |
| `c/ChatView.logic.ts` | 257–279, 1172–1176 | anchor release on tool activity |
| `apps/web/src/index.css` | 551–660 | `live-tool-shine`, `live-activity-focus*` |
| | 952–961 | `.chat-content-lane`, `.messages-timeline-row-frame` |
| | 1792–1801 | code font size scope |
| | 1870–2289 | every `.chat-markdown` rule |
| | 2559–2588 | timeline padding and the scroll-to-end inset |
| `apps/web/src/timestampFormat.ts` | 101–201 | timestamps and their tooltip |
| `apps/web/src/lib/readableSyntaxTheme.ts` | 1–76 | `t3-light`: pierre-light darkened to 4.5:1 |
| `apps/web/src/lib/diffRendering.ts`, `apps/web/src/lib/syntaxHighlighting.ts` | 7–16; 22–44 | highlighter themes and engine ("highlights") |
| `packages/client-runtime/src/work-log/presentation.ts` | 131–693 | tool presentation, failure detection, group actions, summaries, live thought line |
| `packages/client-runtime/src/userMessage.ts` | 21–139 | compaction labels, attribution, grouping |
| `packages/shared/src/orchestrationTiming.ts` | 22–37 | `formatDuration` |
| `packages/shared/src/usageFormat.ts`, `apps/web/src/lib/formatBytes.ts` | 66–79; 1–16 | token and byte formats |
| `packages/shared/src/chatList.ts` | 1–42 | anchored end space |
| `packages/shared/src/t3McpToolPresentation.ts` | 75–100, 373–374 | T3 tool label tuples |
| `packages/shared/src/proposedPlanText.ts` | 1–20 | plan title |
| `packages/contracts/src/settings.ts` | 65–67, 346–347, 527–529, 591, 594 | timestamp format, tool call visibility, word wrap, math defaults |
| shipped `client/assets/pierre-light-Vf__O3ev.js`, `pierre-dark-B2jZWvHG.js` | all | the Zed-format syntax palettes the code highlighter uses |
| `c/chat/MessagesTimeline.logic.ts` | 1005–1020, 1955–1976 | failed runs are itemized: one `work` row per entry, no grouping |
| `c/chat/QueuedRunsControl.tsx` | 57, 241–551 | queued turns at the end of the footer (§2.10, §3.12) |
| `c/chat/SnapShotAttachmentDetails.tsx` | 13–14, 74–168 | window snapshot attachments: overlay, accessibility data button |
| `c/chat/TimelineUserMessageRow.tsx` | 497–527, 680–790 | media classification, corner tightening, snapshot frames, image fallbacks |
| `c/chat/PierreEntryIcon.tsx` | 7–96 | Pierre file icon colours per token, light and dark (§3.6) |
| `packages/shared/src/favicon.ts` | 3–85 | link favicons: Google s2 service for public hosts, GitHub's themed SVG |
| `packages/client-runtime/src/state/subagentDisplay.ts` | 13–50 | subagent group summary and status counts, display title |
| `c/chat/SubagentTooltipContent.tsx`, `c/ThreadHoverCard.tsx` | 60–188; 10–29 | subagent hover card |
| `c/chat/TimelineSystemDivider.tsx` | 7–85 | divider pill with an action (measured on the "Subagent of" header) |
| `c/chat/LinkRequestCard.tsx`, `c/chat/SecretRequestCard.tsx` | 57–160; 40–240 | link and secret request cards |
| `c/chat/ProposedPlanCard.tsx`, `apps/web/src/proposedPlan.ts` | 76–100, 161–230; 3–40 | plan collapse threshold and 10-line preview |
| `c/chat/MarkdownThreadLink.tsx`, `c/contextChipParts.tsx` | 19–71; 343–354 | thread chips and the unresolved chip |

**Live specimens.** Measured in the running app in both schemes: about twenty existing threads of the account, picked for what they contained (work groups, code blocks, tables, headings, blockquotes, images, errors, compaction, subagents, live runs). The words in them never appear here; examples use placeholders. Hover was driven with real pointer moves, disclosures were opened by clicking their triggers (folds, groups, entries, folded messages, the image dialog, the background work dialog) and closed again. Live rows (working, thinking, subagent running) were measured on threads that were running at the time.

**Method notes.** The browser tab reported `document.visibilityState: hidden` (another window covered it), which pauses `prefers-color-scheme` handling, ResizeObserver and the gated animations. I forced frames with screenshots after every scheme switch and turned on CDP focus emulation (`Emulation.setFocusEmulationEnabled`, `Page.setWebLifecycleState active`), after which the tab reported `visible`. One effect of the earlier hidden state is recorded in §5.12: a minimap stuck in hover-only mode. The verification pass showed it is a hidden-tab artefact, not a navigation bug.

## 2. Structure

### 2.1 Timeline frame

```
div[data-workspace-area=conversation]        relative flex min-h-0 flex-1 flex-col bg-background   (03-shell §2.2)
├─ TooltipScrollDismissArea                  relative h-full min-h-0  [data-assistant-citation-viewport]   closes tooltips on scroll
│  ├─ AssistantSelectionToolbar             portal to <body>, only while a selection is citable (§2.9)
│  ├─ LegendList scroll node                messages-timeline-scroll scrollbar-gutter-both h-full min-h-0 overflow-x-hidden
│  │  │                                     overscroll-y-contain [overflow-anchor:none] [topbar-scroll-fade unless a banner tops the timeline]
│  │  └─ div.legend-list-content-container
│  │     ├─ header       24px spacer (h-[--workspace-titlebar-scroll-fade-height]) when the top fade is on, else h-3 sm:h-4
│  │     │               then TimelineHistoryControl; a subagent or side chat drops the spacer and puts its parent divider after the history control
│  │     ├─ rows         absolutely positioned recycled containers, one per MessagesTimelineRow
│  │     │  └─ div.messages-timeline-row-frame                padding-inline 0
│  │     │     └─ div.chat-content-lane.overflow-x-clip [data-timeline-root]   margin-inline auto, width 100%, max-width --chat-content-max-width
│  │     │        └─ TimelineRowContent  [data-timeline-row-id][data-timeline-row-kind][data-message-id][data-message-role]
│  │     │                               bottom padding by kind (§3.1), "group/assistant" on assistant and assistant-meta rows
│  │     └─ footer       TimelineListFooter: [footer content] [unmatched approvals] [spacer = composer inset] [h-3 sm:h-4]
│  └─ TimelineMinimap                       absolute inset-y-0 left-0 z-40 w-18 (§2.9)
└─ ScrollToEndButton                         sibling of the timeline, absolute z-30 (§2.9)
```

- The scroll node's padding and gutters, the 24px top fade and the lane width are in 03-shell §3.5. At 1440 with the details card inline: padding 42 / 242, `scrollbar-gutter: stable both-edges` (6px each side), content container 888 wide, lane 736 at x 380 (M).
- Rows are virtualized. LegendList keeps a draw buffer; recycled containers move between rows. The first and last message rows, rows covered by a text selection, the cited row, a fullscreen app row and rows with playing audio stay mounted (MessagesTimeline.tsx:857–881).
- The footer holds, in order (ChatView.tsx:3109–3145, MessagesTimeline.tsx:1357–1377): a pending context handoff divider, the ThreadNow lines (attention, background work, sync, goal), the thread status line, pending guest messages, queued turns (§2.10), then approvals that no row claimed, then a spacer as tall as the composer overlay, then 16px. The footer content is `null` when none of the first five exists (ChatView.tsx:3111–3116).
- A subagent thread or a side chat has no fade spacer and no `h-3 sm:h-4` header. Its list header is the history control (when shown), then `div.messages-timeline-row-frame > div.chat-content-lane pt-1 sm:pt-2` holding a divider "Subagent of · <parent title>" (`bot`) or "Context from · <parent title>" (`messages-square`), with " on <environment>" appended for a linked machine and the action "Open parent thread" (MessagesTimeline.tsx:1188–1219). Measured: block 50 tall (8 + a 42px divider with its 26px pill). With no history control the first row starts at y 102, where a normal thread starts at 76 (M, dark).

### 2.2 Row kinds

`deriveMessagesTimelineRows` turns timeline entries into these rows (MessagesTimeline.logic.ts:494–644). The renderer picks the component from `kind` (MessagesTimeline.tsx:1589–1674).

| Kind | Component | Row bottom padding | What it is |
|---|---|---|---|
| `message` (user) | `UserMessageRow` | 16 (`pb-4`), 4 (`pb-1`) when another own message follows | a user bubble, or a folded agent/automation message |
| `message` (assistant) | `AssistantTimelineRow` | 16 with its meta row, 8 (`pb-2`) without | assistant markdown |
| `assistant-meta` | `AssistantMetaTimelineRow` | 16 | the copy/fork/reply/time row moved below trailing tool groups |
| `turn-fold` | `TurnFoldTimelineRow` | 6 (`pb-1.5`) | "Worked for 2m 14s" disclosure |
| `attempt-fold` | `AttemptFoldTimelineRow` | 8 | "Superseded attempt" disclosure |
| `work` | `WorkGroupSection` in `WorkLogBlock` | none; the block owns 8 / 4 / 0 | a single tool row, a standalone notice, an expanded group's list, or one entry of a failed run (failed runs are never grouped, §5.2 step 7) |
| `work-toggle` | `WorkGroupToggleTimelineRow` | none (block) | a collapsed group header: "Ran 3 commands and read 2 files" |
| `work-live` | `LiveWorkEntryTimelineRow` | none (block) | the running tool, shimmering |
| `thinking` | `ThinkingTimelineRow` | none (block) | "Thinking" while nothing visible runs |
| `working` | `WorkingTimelineRow` | 6 | "Working for 12s" header of the live turn |
| `viewed-images` | `ViewedImagesTimelineRow` | 8 | images the agent opened in the group above |
| `context-compaction` | `ContextCompactionTimelineRow` | 16 | "Context compacted 180K [arrow-right] 12K tokens" divider |
| `event` | `V2EventTimelineRow` | 8; none for subagent items: a lone card sits flush, a group's `WorkLogBlock` adds 8 (0 when another work-log row follows; measured 46 + 8 = 54) | dividers ("Run interrupted", "Forked from conversation" / "Conversation fork", handoff), subagent cards, created threads, link and secret requests |
| `proposed-plan` | `ProposedPlanTimelineRow` | 16 | plan card |
| `html-render`, `mcp-app` | `HtmlRenderFrame`, `McpAppFrame` | 8 | an HTML artifact or an MCP app frame |
| `worktree-setup` | `WorktreeSetupCard` | 8 | worktree setup progress |

### 2.3 User message

```
div  group @container flex flex-col items-end gap-1  [data-user-message-attribution=agent] when folded
│    (automation messages use their own wrapper with =automation; the "Sent by T3 Code" line carries =t3code)
├─ p "Sent by T3 Code"                    me-1 text-2xs text-muted-foreground/70           (server-sent agent message only)
├─ GuestMessageName                       me-1 text-2xs text-guest-foreground              (guest messages)
├─ UserMessageIntentMarker                me-1 flex items-center justify-end gap-1 text-xs leading-none text-muted-foreground
│                                         [redo-2 12px] "Steer" | "Steer · waiting to be read"   (steers, once per run of steers)
├─ div flex max-w-[80%] justify-end       media above text (images or videos sent with words)
├─ div bubble  relative min-w-0 max-w-[80%] rounded-2xl text-message-foreground bg-message p-3
│  │           + rounded-tr-md (media above, or own message above) + rounded-br-md (own message below)
│  │           guests: bg-guest-surface; media-only, file-cards-only and audio-only messages: no fill, no padding
│  ├─ h3.sr-only "You" | sender thread title | "<name> (guest)"
│  ├─ FoldedMessageHeader                 (agent-attributed messages)
│  ├─ div[dir]                            body; folded: px-3 pb-3, max-h-96 overflow-auto, hidden until expanded
│  │  ├─ media grid                       (media inside the bubble: mb-2 max-w-[210px] grid-cols-2 gap-2)
│  │  ├─ audio previews | file cards | file rows
│  │  └─ CollapsibleUserMessageBody
│  │     ├─ div relative [data-user-message-body][data-user-message-collapsed][-collapsible][-fade]   max-h-44 overflow-hidden + mask when collapsed
│  │     │  └─ ChatMarkdown "text-foreground user-message-markdown", line breaks kept, raw HTML shown as text
│  │     └─ div[data-user-message-footer] mt-1.5 flex justify-end gap-2 > Button xs ghost-muted -ml-1 "Show full message" | "Show less"
│  └─ floating actions                    absolute end-full top-1/2 me-2 (only when grouped with the next message, ≥ 37rem container)
├─ UserMessageProjectedStatus             me-1 > pill "failed" etc. (status other than completed, pending, waiting)
└─ actions row  flex w-full max-w-[80%] justify-end pe-1 items-center text-xs tabular-nums opacity-0 transition-opacity duration-200
   │            group-hover:opacity-100 focus-within:opacity-100 pointer-coarse:opacity-100
   └─ div flex items-center justify-end gap-x-2 gap-y-0.5
      ├─ InlineButton muted "Open thread"   (folded agent messages whose sender thread can open)
      ├─ p timestamp  text-muted-foreground text-xs tabular-nums  (tooltip: long form)
      └─ div flex items-center gap-0.5
         ├─ Button xs ghost [pencil 12px] "Edit from here"   (rollback supported and a checkpoint exists)
         ├─ MessageCopyButton ghost [copy 12px] "Copy message"
         └─ MessageCopyButton ghost [link 12px] "Copy prompt link"  (prompt 1 to 8000 chars)
```

`FoldedMessageHeader` (agent and automation messages, TimelineUserMessageRow.tsx:383–410): `button flex w-full min-w-0 items-center gap-1.5 rounded-2xl px-3 py-2 text-left text-sm` with `[messages-square | calendar-clock, 16px, stroke 2, muted]`, a muted label (`max-w-1/2 shrink-0 truncate`), the first line of the message (`min-w-0 flex-1 truncate`, foreground) and `[chevron-down 14px muted, rotate-180 when open]`.

Window snapshots (image attachments whose `source.kind` is `snap-shot`, TimelineUserMessageRow.tsx:700–772, SnapShotAttachmentDetails.tsx:136–168) never join a collage. Standalone, the frame is `bg-background/70 col-span-2 min-h-16 justify-self-end overflow-hidden rounded-2xl` plus the media ring, and the image is `h-auto max-h-48 w-auto min-w-24 max-w-full object-contain` (the whole capture, at most 192 tall, not cropped). Inside a bubble the frame is `relative h-28 w-52 max-w-full overflow-hidden rounded-lg border border-border/80` (208 × 112). Over the bottom edge sits:

```
div pointer-events-none absolute inset-x-0 bottom-0 flex min-w-0 items-center gap-1.5
    bg-linear-to-t from-black/85 via-black/55 to-transparent px-2.5 pb-2 pt-6
├─ img size-7 rounded-md (app icon) | div size-7 rounded-md bg-white/20 text-3xs font-medium text-white uppercase (first letter of the app)
└─ div min-w-0 flex-1
   ├─ div flex items-center gap-1.5 text-2xs font-medium leading-3.5 text-white > [truncate app name] [SnapShotContentsButton]
   └─ div truncate text-3xs leading-3.5 text-white/70  window title | "Captured window"
```

`SnapShotContentsButton` is a `Button icon-micro overlay` (20×20) with `text` 12px ("View accessibility data") when the capture carries accessibility data, else `image` 12px ("No accessibility data"); it opens a `md` popover titled "Accessibility data" holding the data in a `max-h-64 rounded-md border border-border/70 bg-muted/45 p-2.5 text-2xs leading-4` mono block (JSON highlighted), or a sentence saying none was provided. An image with no preview URL shows its file name instead: `flex min-h-[72px] items-center justify-center px-2 py-3 text-center text-2xs text-muted-foreground/70` (lone: `h-48 w-64`).

The automation message (TimelineUserMessageRow.tsx:283–371) is the same bubble without padding (`min-w-0 max-w-[80%] rounded-2xl bg-message`), label "Automation", summary = the scheduled task title or the first line. Open, it adds a `w-full` bubble, the body `px-3 pb-3 max-h-96 overflow-auto`, and an actions row with "Open automation" (link to the scheduled task), an always-visible timestamp, "Edit from here", copy and prompt link.

### 2.4 Assistant message

```
div  relative min-w-0 px-1 py-0.5
├─ h3.sr-only "T3 Code"
├─ div[data-thread-find-text] > AssistantCitationSource [data-assistant-citation-source=<message id>]
│  └─ ChatMarkdown  (root classes §2.5)
├─ ChangedFilesCard                     mt-4 (when the turn changed files, §3.4)
├─ AssistantStatusBadge "interrupted"   mt-1.5 flex (no meta row and the item was interrupted)
└─ AssistantMessageMeta                 mt-1.5 flex items-center gap-2 text-xs tabular-nums transition-opacity duration-200
   │                                    opacity-0 group-hover/assistant:opacity-100 focus-within:opacity-100 pointer-coarse:opacity-100
   ├─ AssistantStatusBadge              (status other than completed)
   ├─ MessageCopyButton ghost [copy 12px]          "Copy message"
   ├─ Button xs ghost [git-fork 12px]              "Fork from this response"  (provider can fork; pulses while forking)
   ├─ Button xs ghost-muted [reply 12px]           "Reply to this response"
   └─ p timestamp of the last update               text-muted-foreground text-xs tabular-nums
```

The `assistant-meta` row is the same meta row in `px-1`, `mt-0.5`, always visible (TimelineAssistantRows.tsx:146–163). It replaces the inline meta when the response ended with tool groups (MessagesTimeline.logic.ts:1243–1334).

### 2.5 Markdown

Root (ChatMarkdown.tsx:1861–1871): `div.chat-markdown w-full min-w-0 [overflow-wrap:anywhere] [word-break:break-word] text-sm leading-relaxed text-foreground/[calc(90%+var(--appearance-contrast-boost)/10)]`, `data-streaming` while streaming. `inline` renders a `span` without the text classes (thought previews in the work log). User bodies add `text-foreground user-message-markdown` (full foreground, ordered lists always decimal).

| Markdown | Element and override | Source |
|---|---|---|
| paragraph | `p`, skill mentions rendered inline as chips | ChatMarkdown.tsx:1049–1056 |
| headings | `h1`..`h6`, `aria-level` offset by 3 inside messages | 1011–1038 |
| blockquote | `blockquote`; `> [!NOTE]` style alerts become `div[role=note] my-1 border-s-2 ps-3` with a title `p` | 1057–1079, 350–390 |
| lists | `ul`, `ol` (inline `--list-gutter: <digits+2>ch` when a marker has 2+ digits), `li` | 1080–1099, 333–348 |
| task item | `li.task-list-item` + `input[type=checkbox]` read-only (toggle only where the caller supports it) | 1100–1130 |
| link | `a target=_blank rel=noopener noreferrer`; external links get a favicon; file paths become file chips; `#` links scroll in-document; thread links become thread chips; citation links become quoted-response cards; context references render the caller's chip | 1131–1410 |
| inline code | `code[data-inline-code]`; a file path in backticks becomes a file chip; double click selects the whole span | 1411–1447 |
| image | `img` → `ChatMarkdownImage` / `ChatMarkdownAssetImage` / video / audio / fallback | 1448–1642 |
| table | `MarkdownTable` | 1643–1645 |
| details | `MarkdownDetails` | 1646–1648 |
| fenced code | `MarkdownCodeBlock` (mermaid → `MarkdownMermaidCodeBlock`, `math` → display math when math rendering is on) | 1649–1731 |
| artifact template | `div[data-chat-markdown-artifact-template]` card | 287–329 |
| footnotes, strikethrough, autolinks | GFM via `remark-gfm` | packages/shared/src/markdownPipeline.ts:246–275 |
| line breaks | kept for user messages; for assistant messages only when the text has a "★ Insight" block | TimelineAssistantRows.tsx:63, markdownPipeline.ts:739–741 |

### 2.6 Code block

```
div.chat-markdown-codeblock  my-[0.65rem] text-start overflow-hidden rounded-lg border border-border/70 bg-secondary leading-snug
│                            dark:border-transparent dark:bg-input/32   [data-language][data-wrap]  dir=ltr
├─ div.chat-markdown-codeblock-header  flex items-center justify-between gap-2 pt-1.5 pr-1.5 pb-0 pl-3 select-none  [data-thread-find-ignore]
│  ├─ span inline-flex min-w-0 items-center gap-1.5 font-mono text-2xs
│  │  └─ fence title: [Pierre file icon 14px] + name | language with a known icon: [Pierre icon 14px] (tooltip = language) | otherwise the language word
│  └─ span[role=toolbar][aria-label="Code block actions"]  flex items-center gap-0.5
│     ├─ Button icon-xs secondary (wrapped) | ghost-muted (not wrapped)  [text-wrap 12px]  aria-pressed  "Disable line wrap" | "Wrap lines"
│     ├─ Button icon-xs ghost-muted [play 12px] "Run in terminal"   (shell fences, see §5.8)
│     ├─ view toggle  (mermaid: [code 12px] "Show code" / [workflow 12px] "Show diagram")
│     └─ Button icon-xs ghost-muted [copy 12px → check 12px] "Copy code" → "Copied"
└─ div.chat-markdown-shiki > pre.diffs (or .shiki) > code > span.line > span[style=color]
```

The mermaid diagram view drops the frame: `div my-[0.65rem] border border-transparent` with a right-aligned `flex justify-end gap-0.5 pt-1.5 pr-1.5` toolbar (view toggle + copy) above the diagram (MarkdownCodeBlock.tsx:200–210).

Code blocks have **no line numbers**: the highlighter emits `pre.diffs[tabindex=0] > code > span.line` with inline `span.line` elements and no counter or gutter (M). The `pre` carries the theme's inline `color` and `background-color` (the background is overridden to transparent), is focusable for keyboard scrolling, and inherits `tab-size: 4`. The language comes from the fence's `language-*` class: no language gives `text` (the header then shows the word "text"), and `gitignore` maps to `ini` (MarkdownCodeBlock.logic.ts:6–13). A fence title comes from `title=`, `file=` or `filename=` in the fence meta (quoted or bare), else from the first meta word that looks like a file name such as `src/app.ts` (logic.ts:15–24). If a language fails to highlight, the block re-renders as plain `text` (MarkdownCodeBlock.tsx:339–349).

### 2.7 Work log

Every work-log row is drawn by one of three wrappers (WorkLog.tsx:53–126):

```
WorkLogBlock  div  pb-2 (standalone) | pb-1 (expanded group content) | pb-0 (group header, or another work-log row follows)
└─ WorkLogButton <button> | WorkLogRow <div>   group/timeline-row relative w-full min-w-0 rounded-md px-0.5 text-left transition-colors
   │                                           py-0.5 (standalone) | py-0 (inside an expanded group list)
   │                                           + cursor-pointer focus-visible:ring-2 ring-inset ring-ring/70 when clickable
   ├─ WorkLogLine  flex min-h-6 min-w-0 gap-1.5 text-sm leading-relaxed select-none, items-center (items-start when the label wraps)
   │  ├─ div flex min-w-0 flex-1 items-baseline gap-1.5
   │  │  ├─ span relative w-6 shrink-0 leading-0 > span inline-block size-6 overflow-hidden align-[calc(0.5cap_-_0.75rem)]
   │  │  │                                       > span flex size-full items-center justify-center > ICON (16px)
   │  │  └─ div min-w-0 flex-1 text-secondary-label truncate [&_*]:whitespace-nowrap    LABEL
   │  └─ TRAILING  (failure x, chevron, timestamp, "Open chat", "Open subagent")
   └─ children (WorkLogDetails)   ms-7 text | mt-0.5 mb-1.5 pb-2 panel | mt-1 media
```

```
turn fold     div group/timeline-row border-b border-border/60 pb-2 pe-0.5 pt-1
              └─ div relative flex items-center gap-1
                 ├─ button flex cursor-pointer select-none items-center gap-1 rounded-md px-1 text-sm leading-relaxed text-muted-foreground
                 │         tabular-nums transition-colors hover:text-foreground  "Worked for 2m 14s" [chevron-right | chevron-down 14px]
                 └─ TimelineRowTimestamp ms-auto
              + WorkHistorySkeleton while deferred history loads

work-toggle   WorkLogButton: [summary icon 16px] [summary label] [timestamp]
expanded      work row: WorkLogList > LegendList [role=region][aria-label="Tool calls"][data-tool-group-scroll] tabindex=0
              scrollbar-gutter-stable scroll-py-6 overflow-x-hidden rounded-md + virtualized scroll fade
              style max-height: calc(min(18rem, 50dvh) + <expanded entries' extra height>)
              └─ entry rows (24px each, py-0)
entry         WorkLogRow role=button tabindex=0 aria-expanded [data-work-entry-id][data-v2-item-type][data-v2-item-visibility]
              ICON = span flex size-4 items-center justify-center <tone colour> > ToolActivityIconView "block size-4 shrink-0 stroke-2"
              LABEL = div min-w-0 flex-1 overflow-hidden > p flex min-w-0 w-full items-baseline gap-1.5 text-sm leading-relaxed
                      > span min-w-0 flex-1 truncate <heading colour> (+ answer preview span)
              TRAILING = ["Open chat" | "Open subagent"] [x 12px failure mark] [span size-4 > chevron-right 12px] [timestamp]
              expanded: WorkLogDetails media (viewed image) | QuestionAnswerHistory | ReasoningTraceContent | panel (V2ItemInspector or pre)
```

Live rows (TimelineWorkGroupRows.tsx:291–603):

```
working       div border-b border-border/60 pb-2 pt-1
              └─ div flex h-6 min-w-0 items-baseline gap-2 px-1 text-sm leading-relaxed text-muted-foreground tabular-nums
                 ├─ span relative shrink-0 overflow-hidden whitespace-nowrap  "Working for 12s" | "Working..." | "Setting up worktree…" | [minimize-2] "Compacting for 4s"
                 │  └─ ActivityShimmerOverlay (only for worktree setup and compaction)
                 └─ Button micro ghost-muted ml-auto [Spinner 12px] "Background setup" → popover
thinking      LiveActivityRow [brain] "Thinking" (or the provider's starting label, or the zen live answer preview), shimmer
              inside a button "group/live-work flex min-h-6 w-full ..." when it carries a failed group's disclosure
work-live     button group/live-work flex w-full max-w-full cursor-pointer rounded-md text-left (flex-col when a thought line shows)
              ├─ WorkLogRow [brain] thought line in "line-clamp-4" (ChatMarkdown inline), when a newer thought exists
              └─ LiveActivityRow  div relative min-h-6 w-fit max-w-full min-w-0 overflow-hidden rounded-md text-sm leading-relaxed
                 ├─ LiveActivityContent (WorkLogRow: icon text-icon-muted, label span.block.truncate.live-tool-shine while running)
                 └─ ActivityShimmerOverlay > LiveActivityContent highlighted (only for "thinking": the foreground band)
```

### 2.8 Events and cards

```
divider       div flex min-w-0 items-center gap-2 py-2 text-2xs text-muted-foreground (text-destructive for danger)
              ├─ span h-px flex-1 bg-border/70
              ├─ span min-w-0 rounded-full px-2 py-1 (flex flex-wrap items-center justify-center gap-1.5 | text-center when the detail wraps)
              │  │   or, with an action: button rounded-full border border-border/70 bg-background px-2.5 py-1 hover:bg-accent hover:text-foreground
              │  ├─ span inline-flex items-baseline gap-1.5 font-medium [icon 12px self-center] LABEL
              │  └─ span min-w-0 opacity-70 "· detail" (max-w-80 truncate, or ml-1.5 wrap-anywhere whitespace-pre-wrap)
              └─ span h-px flex-1 bg-border/70
subagent      button group/subagent flex w-full min-w-0 items-center gap-2.5 rounded-md px-2 py-1.5 text-left
              hover:bg-accent/20 focus-visible:ring-2 ring-inset ring-ring/70   (a div when it cannot open)
              ├─ avatar span relative inline-flex size-6 rounded-full border border-border/70 bg-muted ring-2 ring-background
              │  ├─ provider icon 14px | [bot 14px muted]
              │  └─ status dot absolute size-2 rounded-full ring-2 ring-background -right-px -bottom-px (top-left when a badge shows)
              ├─ span min-w-0 flex-1
              │  ├─ span flex items-baseline gap-2 > title (truncate text-xs font-medium text-foreground) ["on <machine>" text-xs muted]
              │  └─ span block text-2xs leading-relaxed muted|destructive > status label | detail (truncate, or MiddleTruncate for a path)
              ├─ status label  shrink-0 text-xs muted|destructive   (when a detail shows and the agent is not completed)
              ├─ elapsed | notification time  shrink-0 text-xs muted tabular-nums
              └─ chevron-right 14px text-muted-foreground/60, group-hover:text-foreground
              hover card (200ms delay): ThreadHoverCardPopup with SubagentTooltipContent
subagent group WorkLogBlock > Collapsible[data-subagent-group]: trigger flex w-full items-center gap-2.5 py-2 pr-3.25 transition-opacity hover:opacity-100,
              text-foreground when open or any member is active, else text-muted-foreground opacity-55
              ([avatar ×3 overlapping -space-x-1.5] [+N] ["N subagents" text-xs semibold / status summary text-3xs] [elapsed] [chevron-down 14px, rotate-180 when open])
              aria-label "N subagents", aria-description = the status summary
              panel (CollapsiblePanel animate={false}) mt-1 mb-1 rounded-lg border border-border/60 bg-card/30 p-1 > subagent rows
              a lone subagent renders as a plain subagent card, not a group (TimelineV2EventRow.tsx:19–24)
subagent hover ThreadHoverCard in a Tooltip popup (glass, max-w-80): div flex max-w-80 flex-col gap-2 px-1 py-2
              ├─ title  truncate text-xs leading-tight font-medium text-foreground
              └─ div grid gap-1.5 pl-0.5 text-xs text-muted-foreground
                 ├─ [provider icon 12px grayscale opacity-60 | bot 12px] model label text-foreground/75 (" · <instance>", " · <effort>", speed icon)
                 ├─ flex justify-between gap-4: [status icon 12px + status, capitalize font-medium: info | error | success | muted] [elapsed]
                 ├─ [git-branch | folder 12px] MiddleTruncate value (branch, worktree)
                 └─ [terminal 12px] MiddleTruncate preview of the latest activity
created thread  WorkLogRow [T3 wordmark 16px icon-muted] "Created thread · <title>" + InlineButton "Open chat"
                resource summary (after the final answer): div flex items-center gap-3 rounded-lg border border-border/60 p-3
                [span size-9 rounded-lg bg-muted/60 > message-square 16px secondary-label] [title text-sm font-medium] [Button xs outline "Open chat"]
plan          div rounded-2xl border border-border/80 bg-card/70 p-4 sm:p-5
              ├─ header flex flex-wrap items-center justify-between gap-3: [Badge secondary "Plan"] [h3 truncate text-sm font-medium] [Button icon-xs outline ellipsis 16px menu]
              └─ div mt-4 > clip (relative, max-h-104 overflow-hidden when collapsed) + fade h-24 bg-linear-to-t from-card/95 via-card/80
                 └─ mt-4 flex justify-center > Button sm outline "Expand plan" | "Collapse plan"
attempt fold  button flex w-full items-center gap-2 rounded-md border border-border/60 bg-muted/20 px-2.5 py-2 hover:bg-muted/35
              [chevron 14px muted] [label text-xs font-medium text-foreground/80 "Superseded attempt"] [text-2xs muted "Cut off by a steer"]
compaction    div[role=separator] mx-auto flex w-full max-w-(--chat-content-max-width) items-center gap-3 py-1 text-xs muted|destructive
              [h-px flex-1 bg-border/70 | bg-destructive/30] [span relative min-w-0 overflow-hidden > [minimize-2 | circle-alert 12px] text (+ shimmer)] [rule]
approval      WorkLogRow role=group wrapLabel  [kind icon 16px: terminal | square-pen | eye | wrench | hammer] code block label (max-h-40 overflow-auto mono text-xs leading-5 text-foreground)
              └─ div ms-7.5 flex flex-col gap-1 pb-1 text-xs: [app name] [reason] [[shield 12px] "Waiting for approval" · Decline · Approve · More]
link request, pending   div flex min-w-0 flex-col gap-3 rounded-xl border border-border/60 bg-card p-4
              ├─ div flex flex-col gap-1: p text-sm font-medium text-foreground "Link <machine> to this environment" | "Link a machine to this environment", p text-sm muted reason
              ├─ p[role=alert] text-xs text-destructive (error)
              └─ div flex justify-end gap-2: Button xs ghost-muted "Decline", Button xs "Review link…" | "Choose machine…" (opens a wizard dialog)
                 or p text-xs muted "Only a session that manages access here can answer this."
secret request, pending  form, same frame
              ├─ label text-sm font-medium text-foreground (the request's label), p text-sm muted wrap-anywhere reason
              ├─ [Button sm outline "Open <host>" + external-link] [p font-mono text-xs muted url]   (when the request names a link)
              ├─ div flex items-center gap-2: [source select] [Input type=password, or a mono reference input] [Button "Save securely"]
              ├─ p[role=alert] text-xs text-destructive (error)
              └─ div flex justify-between gap-2: p text-xs muted [shield-check 14px] privacy note, Button xs ghost-muted "Decline"
answered or pending elsewhere   WorkLogRow, icon 14px text-icon-muted: link → link | check | x | minus; secret → lock | check | minus
              label: the outcome; a secret reads "<label> · <outcome>"
```

Thread links render `ThreadContextChip` (kind `thread`, `--context-chip-thread` `#006f6b` / `#009e97`, 01-tokens §3.16) titled with the thread's title, the written label, or "Thread". A link whose thread cannot be found renders `UnresolvedChip`: `circle-dashed` icon, the written label, state `unresolved`, aria-label "Unavailable context, <label>" (MarkdownThreadLink.tsx:58–70, contextChipParts.tsx:343–354).

### 2.9 Overlays on the timeline

```
scroll to end   div.chat-scroll-to-bottom pointer-events-none absolute z-30 flex justify-center py-1.5  style bottom = clearance + 4
                (inset-inline: --chat-lane-inset-start / -end, so it centres on the lane)
                └─ Button xs glass pointer-events-auto [data-scroll-to-end]  [chevron-down 14px] "Scroll to end" [muted "· <summary>"]
minimap         div group/minimap pointer-events-none absolute inset-y-0 left-0 z-40 w-18  (opacity-100 with a ≥48px gutter, else shown on hover, fine pointers only)
                └─ hit strip  absolute top-1/2 left-3 -translate-y-1/2  style height: min(<(n-1)×8>px, max(4rem, 100% - 14.75rem)); width: <0..40>
                   ├─ nav button "Previous turn" (above, bottom-[calc(100%+6px)]) [chevron-up 16px]
                   ├─ button "Jump to message: ..." absolute inset-y-0 left-0 w-full
                   │  ├─ hairline absolute top-0 left-3 h-full w-px bg-border/15
                   │  ├─ strips  absolute left-0 h-0.5 w-6 origin-left -translate-y-1/2 rounded-full, top = i/(n-1)
                   │  └─ preview  absolute left-8 w-80 -translate-y-1/2 > span.dropdown-glass rounded-xl p-3 shadow-xl shadow-black/25
                   └─ nav button "Next turn" (below) [chevron-down 16px]
cite toolbar    Button xs glass fixed z-50 max-w-[calc(100vw-1rem)] [quote 14px] "Cite" | "Shorten selection"   (portal)
image dialog    Dialog variant media: backdrop bg-black/75 z-60; popup 92vw × 92vh grid
                ├─ Button icon-sm media-close absolute right-0 top-0 [x 16px]
                ├─ Button icon media-navigation left/right [chevron-left/right 20px]   (2+ items)
                └─ div relative isolate z-10 flex flex-col items-center
                   ├─ zoomable image  rounded-lg bg-background shadow-2xl ring-1 ring-border/70 | video | failure box
                   └─ caption  mt-2 flex gap-1.5 text-xs text-white/80: name "(2/5)" [snapshot contents button]
```

### 2.10 Queued turns (footer)

Messages waiting to start a turn render at the end of the footer, after guest messages (ChatView.tsx:3143, QueuedRunsControl.tsx:241–551). A queued message wears the attribution of its sent form (agent fold, automation fold, guest name, "Sent by T3 Code"); only the dashed edge and the controls differ.

```
section[data-thread-queue][aria-label="N queued message(s)"]  relative mx-auto flex w-full max-w-(--chat-content-max-width) flex-col gap-2 pb-4
├─ span.sr-only[aria-live=polite] "N queued"
├─ ol flex flex-col items-end gap-3
│  └─ li.group/queued flex w-full flex-col items-end gap-1  aria-label "Queued message 2 of 3, not sent", aria-current when edited
│     ├─ [p "Sent by T3 Code" me-1 text-2xs muted/70] [GuestMessageName]
│     ├─ div flex min-w-0 max-w-[80%] flex-col rounded-2xl border border-dashed text-sm
│     │      border-muted-foreground/40 text-muted-foreground | guests border-guest/50 | editing border-primary/60 text-foreground
│     │      + px-3 py-2 when not folded; w-full when an open fold
│     │  ├─ FoldedMessageHeader (agent: messages-square + sender title | "Agent"; automation: calendar-clock + "Automation" + task title)
│     │  └─ body flex flex-col gap-2 (folded: max-h-96 overflow-auto px-3 pb-2)
│     │     ├─ thumbnails flex flex-wrap gap-1 > span size-8 rounded-md border border-border/70 bg-background > img object-cover | bg-muted/60
│     │     └─ p whitespace-pre-wrap wrap-anywhere  (the full text, context references as their labels)
│     └─ div flex items-center gap-0.5 pe-1 text-xs text-muted-foreground
│        ├─ span me-1 flex gap-2 px-1 opacity-0 → 1 on group/queued hover, focus within, coarse pointer (200ms)
│        │  [InlineButton muted "Open thread"] [InlineButton muted "Open automation"] [timestamp, always visible inside the span]
│        ├─ Button xs ghost-muted [arrow-up] "Send now"   (when the run can be sent; tooltip "Start a turn with this message" |
│        │    "Skip compaction and send this message" | "Send into the current turn", plus the steer shortcut on the first)
│        ├─ span flex gap-1 px-1 [clock-3 12px] "Saving"   (optimistic, not yet on the server)
│        ├─ editing: span px-1 text-foreground "Editing in the composer" + Button xs ghost "Cancel"
│        └─ Button icon-xs ghost-muted [pencil] "Edit queued message" (tooltip "Edit in the composer (<shortcut>)" or the block reason)
│           Button icon-xs ghost-muted [x] "Remove queued message" (tooltip "Remove from queue")
└─ more than 3: the list shows 2, then div self-end pe-1 text-xs > InlineButton muted "Show all N queued" | "Show fewer" [chevron-down 14px, rotate-180 open, 150ms]
```

## 3. Measurements

### 3.1 Frame and vertical rhythm (M unless marked)

| Item | Value | Source |
|---|---|---|
| list header | 24px when the top fade is on (the default), else 16px; then the history control | MessagesTimeline.tsx:159–162, 1213–1219 |
| history control frame | `chat-content-lane flex flex-col gap-1.5 pb-2`: 36px with only the button | 1493–1523 |
| "Load earlier turns" button | full lane width, 28px tall, `py-1.5`, `gap-3`, 12/16 muted; rules `h-px flex-1 bg-border/70` `#e4e4e7b3` / `#ffffff0e` (painted `#ebebed` / `#0e0e0e`); label "Loading earlier turns…" while loading; an error shows below as `p[role=status] text-center text-xs muted` | M |
| first row top | 52 + 24 + (36 when the history control shows) = y 112; a subagent thread or side chat: 52 + (36) + 50 for its parent divider = y 102 without history (§2.1) | M |
| footer | composer inset = ceil(composer overlay height), measured 140, then 16: footer block 156px | MessagesTimeline.tsx:163–191, M |
| lane | 736 wide at x 380, `overflow-x: clip` | M |
| user message row | bubble (one line 46.75) + 4 gap + 24 actions + 16 = 90.75 | M |
| assistant row, meta inline | 2 + text + 6 + 24 + 2 + 16 | M |
| turn fold row | 4 + 22.75 + 8 + 1 border + 6 = 41.75 | M |
| work row, standalone | 2 + 24 + 2 + 8 = 36 | M |
| work toggle, expanded | 28 (no block padding) | M |
| expanded group list | 24 per entry, + 4 block padding, or 0 when another work-log row follows (measured 72 for three entries before a work row); list max height 288px at 900px window (`min(18rem, 50dvh)`), grows by each expanded entry's extra height (measured 288 → 542 with one entry open) | TimelineWorkGroupRows.tsx:130, 278, M |
| working row | 4 + 24 + 8 + 1 border + 6 = 43 | M |
| thinking row | 28 + 8 = 36 | M |
| compaction row | 24 + 16 = 40 | M |
| event divider | 40 + 8 = 48 | M |
| subagent card | 45.875, no row padding (the next row starts flush) | M |
| context gaps | margins collapse between siblings, so the visible gap is the larger of the two margins: block to block 10.4; above a heading 20 (0 when it is the first child); below a heading 10.4 when a `p`, list, quote, code block or table follows (its 0.65rem top margin beats the heading's 8), 8 only before a block without a top margin, 20 before another heading; list items 4 apart. Measured h2 → p 10.39 and h3 → p 10.39 (light; margins do not change with the scheme) | M |

### 3.2 Typography

All sizes M in both schemes. Colour columns are light / dark; "painted" is the colour on `--background` (`#fcfcfc` / `#000000`).

| Text | Font | Size / line height | Weight | Colour | Notes |
|---|---|---|---|---|---|
| assistant body | sans | 14 / 22.75 | 400 | `#18181be6` / `#f5f5f5e6` (painted `#2e2e31` / `#dddddd`) | `text-foreground/90%`; with contrast above 100 the alpha rises by boost/10 |
| user body | sans | 14 / 22.75 | 400 | `#18181b` / `#f5f5f5` | inside `--message-surface` |
| h1 | sans | 20 / 26 | 600 | `#18181b` / `#f5f5f5` | full foreground, `line-height: 1.3` |
| h2 | sans | 18 / 23.4 | 600 | same | |
| h3 | sans | 16 / 20.8 | 600 | same | |
| h4, h5 | sans | 14 / 18.2 | 600 | same | |
| h6 | sans | 14 / 18.2 | 600 | `#71717b` / `#a1a1a1` | muted |
| strong | inherit | inherit | 700 | inherits (90%) | |
| em | inherit | inherit | 400, synthesized oblique | inherits | Inter ships no italic (01-tokens §3.9) |
| del | inherit | inherit | 400 | inherits | `line-through` |
| inline code | mono | 11.9 (0.85em) / 19.34 | 400 | `#18181b` / `#f5f5f5` | see §3.5 |
| link | sans | inherit | 400 | `#1447e6` / `#50a2ff` | `--info-foreground` |
| file chip | sans | inherit / normal | 500 | per kind, see §3.5 | |
| table | sans | 14 / 22.75 | 400, head 600 | inherits | `font-size: 0.875rem` |
| code block | mono | 13 / 17.875 | 400 | editor fg `#0a0a0a` / `#fafafa` + token colours | `--font-size-code` × `leading-snug` |
| code block language | mono | 11 / 16 | 400 | `#18181bb8` / `#f5f5f5b8` (painted on the frame `#57575a` / `#b1b1b1`) | |
| footnotes | sans | 12 / 19.5 | 400 | `#71717b` / `#a1a1a1` | |
| footnote ref | sans | 11 | 600 | link colour | `inline-flex min-w-4 rounded-sm` |
| work-log label | sans | 14 / 22.75 | 400 | `#71717b` / `#a1a1a1` (`--secondary-label`) | non tool entries `#18181bcc` / `#f5f5f5cc` |
| turn fold | sans | 14 / 22.75 | 400 | `#71717b` / `#a1a1a1`, hover `#18181b` / `#f5f5f5` | tabular nums |
| working timer | sans | 14 / 22.75 | 400 | muted | tabular nums |
| tool output | mono | 13 / 21.125 | 400 | muted; command `#18181bd9` / `#f5f5f5d9` | `text-(length:--font-size-code,var(--text-2xs))` |
| timestamps | sans | 12 / 16 | 400 | `#71717b` / `#a1a1a1` | tabular nums |
| dividers | sans | 11 / 16 | label 500 | muted; detail at opacity .7 | |
| compaction | sans | 12 / 16 | 400 | muted, failed `--destructive` | |
| folded message header | sans | 14 / 20 | 400 | label muted, summary foreground | |
| steer marker | sans | 12 / 12 | 400 | muted | `leading-none` |
| "Sent by T3 Code" | sans | 11 / 16 | 400 | `#71717bb3` / `#a1a1a1b3` (P) | |
| status pills | sans | 10 / 14 | 500 (user) / mono 400 (assistant) | §3.3, §3.4 | |

### 3.3 User message

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| bubble | background | `#ebebed` (`--message-surface`) | `#1e1e1e` | M |
| bubble | radius / padding | 16 / 12 all sides | = | M |
| bubble | width | content width, max 80% of the lane (588.8 at 736) | = | M |
| bubble | text | `#18181b` | `#f5f5f5` | M |
| bubble, grouped | joined corner radius | `rounded-tr-md` / `rounded-br-md` = 6 | = | S |
| actions row | box | max 80% wide, right aligned, `pe-1` (4), 24 tall, 4 below the bubble | = | M |
| actions row | opacity | 0 → 1 on row hover or focus within, 200ms | = | M |
| actions | gap | timestamp to buttons 8; buttons 2 apart; each 24×24 (`Button xs`, ghost-muted for copy/link: icon `#71717b` / `#a1a1a1`) | = | M |
| steer marker | box | 12 tall, `me-1`, gap 4, `redo-2` 12px stroke 1.75 | = | M |
| steer marker | colour | `#71717b` | `#a1a1a1` | M |
| steer tooltip | text | "Steered the active turn" / "Originally queued, then promoted to steer the active turn" / unread: "Waiting for the agent to read this. It reads steers between steps, after a running command finishes." | | S |
| folded header | box | 36 tall, `px-3 py-2`, gap 6, radius 16, full bubble width (80%) | = | M |
| folded header | icon | `messages-square` (agent) / `calendar-clock` (automation), 16px, stroke 2, `#71717b` / `#a1a1a1` | | M |
| folded header | label / summary | label muted, `max-width: 50%`, truncates; summary `#18181b` / `#f5f5f5`, truncates | | M |
| folded header | chevron | `chevron-down` 14px stroke 1.75 muted; `rotate: 180deg` when open, 150ms | | M |
| folded body | box | `px-3 pb-3`, `max-height: 384px`, scrolls | = | M |
| long message, collapsed | clip | `max-height: 176px` (`max-h-44`), mask `linear-gradient(#000 calc(100% - 28px), transparent)` | = | M |
| long message | threshold | collapses only when the rendered body is ≥ 2 × 11rem = 352px | | S |
| "Show full message" | button | `Button xs ghost-muted -ml-1`: 24 tall, 12/16 500, `#71717b` / `#a1a1a1`; footer `mt-1.5` (6), right aligned | | M |
| lone image | frame | `rounded-2xl` 16, `overflow-hidden`, `bg-background/70` `#fcfcfcb3` / `#000000b3`; inset ring 1px `#00000017` / `#ffffff1f` on `::after` (z 10) | | M |
| lone image | size | `h-48` 192 tall, `w-auto`, `min-w-24` 96, `max-w-full`, `object-cover`; cursor zoom-in | | M |
| collage, 2 | grid | `w-80` 320, `grid-cols-2`, rows 12rem | | S |
| collage, 3 | grid | 320 wide, `grid-cols-[3fr_2fr]` (191.39 + 127.59), rows `calc(6rem - 0.5px)` (95.5), first image `row-span-2`, `gap-px` | | M |
| collage, 4+ | grid | 320 wide, 2 columns, rows 6rem; an odd last image spans 2 columns | | S |
| collage | background / ring | gaps show `#00000017` / `#ffffff1f` (`bg-black/9`, `dark:bg-white/12`), same ring as the lone image, radius 16 | | M |
| media inside the bubble | grid | `mb-2 max-w-[210px] grid-cols-2 gap-2`, cells `aspect-[4/3] rounded-lg border border-border/80` | | S |
| media sent with words | layout | media as its own shape (`flex max-w-[80%] justify-end`) 4 above the bubble. The two touching corners tighten to 6: the media's bottom-right (`rounded-br-md`) and the bubble's top-right (`rounded-tr-md`). Grouping tightens the media the same way: top-right when grouped with the previous message, bottom-right when grouped with the next (TimelineUserMessageRow.tsx:524–527) | = | M |
| snapshot, standalone | frame | as the lone image (radius 16, fill `#fcfcfcb3` / `#000000b3`, inset ring `#00000017` / `#ffffff1f`), image `object-contain`, at most 192 tall (measured 334 × 192 for a 1920 × 1103 capture) | | M |
| snapshot | overlay | gradient `linear-gradient(to top, #000000d9, #0000008c 50%, transparent)`, padding 24 top / 10 sides / 8 bottom, gap 6; app icon 28×28 radius 6; app name 11/14 500 `#ffffff`; window title 10/14 white at 70% (`#ffffffb2` as Chrome reads it back); contents button 20×20 radius 4, fill `#000000b3`, icon `image` 12px `#ffffffa6`. Same in both schemes | = | M |
| video | frame | `aspect-[4/3]`, standalone `rounded-2xl` + ring; missing source: black box `text-2xs text-white/70` with the file name | | S |
| file card | box | `w-64` 256 × 52, `rounded-2xl`, `bg-message`, `py-2 ps-2 pe-1`, gap 4 | = | P |
| file card | icon box | 36×36 `rounded-lg bg-background/60` `#fcfcfc99` / `#00000099`, Pierre file icon | | P |
| file card | name / meta | 14/20 500 / 12/16 muted tabular: "PDF · 2.40 MB", "Pasted text" shows only the size, "Sending…" while uploading | | P |
| file card | download | `Button icon-xs ghost-muted` [download 14px], tooltip "Download <name>"; disabled while sending | | S |
| file row (with words) | row | `flex items-center gap-2 rounded-md py-1 text-sm hover:underline` [Pierre icon] [name] [eye 16px] + download button | | S |
| status pill | style | `rounded-full border border-destructive/25 bg-destructive/8 px-1.5 py-0.5 text-3xs font-medium text-destructive`: border `#fb2c3640` / `#fb414a40`, fill `#fb2c3614` / `#fb414a14`, text `#fb2c36` / `#fb414a`, 10/14 500 | | P |
| "Sent by T3 Code" | style | 11/16, `#71717bb3` / `#a1a1a1b3`, `me-1` | | P |
| guest | bubble / name | fill `#ead0ef` / `#391f3d` (`--guest-surface`); pending: dashed 1px `#e12afb80`, no fill; name 11px `#c800de` / `#f4a8ff` | | P |
| quoted response card | box | `rounded-lg` 8, `px-2.5 py-1.5`, 14/20, text `#71717b` / `#a1a1a1`, two-line clamp; fill `bg-black/10` `#0000001a` / `bg-black/35` `#00000059` (on the bubble: `#d4d4d5` / `#141414`); hover black/14 / black/50 | | M |
| quoted text mark | style | `rounded-sm` 4, `px-0.5`, fill `#18181b1f` / `#f5f5f51f`, text `#18181b` / `#f5f5f5` | | M |
| quoted card margins | | 10.4 above unless it starts its block, 10.4 below unless it ends it | | S |

Context chips inside sent messages use `ContextChip` (§3.5) with `--context-chip-ink: var(--context-chip-message, var(--context-chip-accent))`, which equals the accent in the standard look (01-tokens §3.16).

### 3.4 Assistant message and meta row

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| row | padding | 4 left/right, 2 top/bottom; text column 728 at x 384 | = | M |
| meta row | box | `mt-1.5` 6 (inline) / `mt-0.5` 2 (meta row), 24 tall, gap 8 | = | M |
| meta row | opacity | 0, 1 on `group/assistant` hover, focus within, or always for the meta row; 200ms | | M |
| copy, reply | button | 24×24, ghost-muted: icon `#71717b` / `#a1a1a1` 12px stroke 1.75 | | M |
| fork | button | 24×24, ghost: text `#18181b` / `#f5f5f5`, icon muted 12px; `animate-pulse` while forking | | M |
| timestamp | text | 12/16 `#71717b` / `#a1a1a1`, the message's last update | | M |
| status badge | pill | `rounded-full border border-border/70 px-1.5 py-0.5 font-mono text-3xs text-muted-foreground`: border `#e4e4e7b3` / `#ffffff0e`, text muted, mono 10/14 | | P |
| empty text | | "(empty response)" unless streaming or the run produced an HTML render or MCP app | | S |
| changed files card | box | `mt-4` 16, radius 8, fill `#fafafa` (`--secondary`) / `#ffffff06` (`input/20`), full text width | | M |
| changed files header | box | sticky `top: 8px` z 10, `px-3 py-2`, 40 tall, top radius 8; fill `#fafafa` / `#000000` + `linear-gradient(#ffffff06, #ffffff06)` | | M |
| changed files title | text | 12/16 500 `#18181b` / `#f5f5f5`: "N changed files", gap 12; counts `+42` `#007a55` / `#00d492`, `-7` `#c10007` / `#ff6467`, mono 12/16; header buttons 4 apart | | M |
| changed files actions | buttons | [chevrons-down-up / chevrons-up-down 12px] "Collapse all folders" / "Expand all folders" (icon-xs ghost-muted), [file-diff 12px] "Open diff" (xs ghost-muted, the word shows when the card is ≥ 24rem) | | M |
| changed files rows | row | 28 tall, radius 6, gap 8, left padding 8 + 14 per depth, `hover:bg-accent/60`; folder [chevron-right 14px muted/70, rotate 90 when open] [folder / folder-closed 14px muted/75] name mono 11/16 `#71717be6`; file [Pierre icon 14px] name mono 12 `text-foreground/85` with MiddleTruncate; stats mono 10px | | M |

### 3.5 Markdown elements

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| block spacing | `p, ul, ol, blockquote, pre, table container` margin | 10.4 / 10.4 (0 top for the first child, 0 bottom for the last) | = | M |
| headings | margin | 20 top, 8 bottom (top 0 when first child). The 8 rarely shows: the next block's 10.4 top margin collapses over it, so heading → paragraph reads 10.4 (§3.1) | = | M (h2, h3), P (all) |
| `ul` | gutter / marker | `padding-left: 20px`; disc → circle → square by depth; marker colour = text colour, 14px | = | P |
| `ol` | gutter / marker | `padding-left: max(1.25rem, --list-marker-sans)` = 20 at 14px Inter; two-digit markers widen it inline to `<digits + 2>ch`; decimal → lower-alpha → lower-roman (user messages: decimal at every depth); marker `tabular-nums` | = | P, M (decimal) |
| `li + li` | margin-top | 4 | = | M |
| task item | marker / checkbox | no marker; checkbox 13×13, `margin: 0 4.9px 2.1px -20px` (into the gutter), `vertical-align: middle`, native control in the scheme's colours | = | P |
| blockquote | rule / padding | 2px solid `currentColor` at 35%: in assistant text `#18181b50` / `#f5f5f550` (painted `#b4b4b5` / `#4d4d4d`), in user text `#18181b59` / `#f5f5f559`; `padding-left: 12.8px`; text inherits | | M |
| hr | rule | 1px `#e4e4e7` / `#a1a1a1` (dark uses `--contrast-muted-foreground` so it reads on bubbles); margins 0, the blocks around it give 10.4 each side | | M |
| inline code | box | `border: 1px transparent`, radius 4, padding `1.19px 4.165px` (0.1em / 0.35em), `box-decoration-break: clone`, mono 11.9px, line height inherited ratio (19.34) | = | M |
| inline code | colours | fill `#18181b1a` (painted `#e5e5e5`), text `#18181b` | fill `#f5f5f51a` (painted `#191919`), text `#f5f5f5` | M |
| link | colour | `#1447e6` | `#50a2ff` | M |
| link | rest / hover | no underline; hover and focus-visible draw a dotted underline: `radial-gradient(circle, currentColor 0.75px, transparent 1px)` at `0 100%`, `4px 2px`, `repeat-x` (not on file links, chips or image links) | = | M |
| link | tooltip | the full href, side top (Tooltip, 02-primitives §3.7) | = | M |
| link favicon | box | 14×14, `ms-[0.25em] me-[0.2em]` (3.5 / 2.8), `vertical-align: -0.125em`, image `rounded-sm` 4 (natural size 32×32); source-control hosts use their brand glyph (an inline svg); failure falls back to `globe` | = | M |
| link favicon | source | `https://www.google.com/s2/favicons?domain=<host>&sz=32` for public hosts only; private, local and IP hosts get `globe` straight away; a host that failed once stays on `globe` for the session (favicon.ts:74–85, MarkdownExternalLink.tsx:20–45) | | S |
| link text | wrapping | the protocol (or the first character) stays glued to the favicon; words over 16 characters may break anywhere | | S |
| file chip | box | `inline-flex`, gap `0.2em` (2.8), 500, `line-height: normal` (17px box), radius 4, `max-width: 100%` | = | M |
| file chip | colour | `--context-chip-mention` `#006d80`; documents use their own kind (pdf `#b33739`, word `#0066ab`, spreadsheet `#11732f`, slides `#9b4f00`, epub `#6b51bb`) | `#0d9ab4`; pdf `#e46561`, word `#3e92da`, spreadsheet `#479e58`, slides `#cd7833`, epub `#937cea` | M (mention), S (kinds, values 01-tokens §3.16) |
| file chip | icon | Pierre file icon by name at `1em` (14px, the svg's own `size-4` is overridden), in its token colour from §3.6 (measured: TypeScript `#1a85d4`, bash `#199f43`, markdown dark `#5ecc71`, no specific icon `#84848a`) | | M |
| file chip | label | underline 1px, `text-underline-offset: 0.2em` (2.8), colour transparent, becomes the chip colour on hover, 150ms | | M |
| file chip | tooltip | code variant (mono 11px), the workspace path | | S |
| table container | spacing | margins 10.4; ScrollArea with edge fades and chained vertical scroll; horizontal scrollbar always visible (`--app-scrollbar-thumb` muted 60%) when it overflows | | M, P (scrollbar) |
| table | text | 14/22.75, `width: 100%` (the table always spans the text column, 728 at this width), `border-collapse`, `overflow-wrap: normal` and `word-break: normal` on the table, cells top-aligned with `text-wrap: pretty` and `overflow-wrap: break-word`. Cells declare `max-width: 24rem`, but Chrome ignores `max-width` on table cells: a probe column of long prose took 602 of 728px, and a measured header cell was 571 wide. Columns come from the auto table layout (min = longest word, spare width shared by content width). Cells holding math drop the cap | = | M, P (probe) |
| `th` | padding / rule | 8.8 top and bottom, 12 left and right (0 at the table's outer edges), 600, bottom rule 1px `#e4e4e799` / `#ffffff0c` (painted `#eeeeef` / `#0c0c0c`) | | M |
| `td` | padding / rule | 7.2 top and bottom, 12 / 0 at outer edges, bottom rule as `th` | | M |
| table copy | row | `mt-0.5` (2), `-mx-1`, right aligned 24×24 `Button icon-xs ghost-muted` [copy 12px] opening a menu "Copy as Markdown" / "Copy as CSV"; icon turns into check for 1.2s | | M |
| footnotes | section | `margin-top: 20px`, top rule 1px `#e4e4e7` / `#ffffff14`, `padding-top: 12px`, 12/19.5 muted; list margins 0, items 5.6 apart | | P |
| footnote refs | box | `inline-flex min-width: 16px` centred, radius 4, 11px 600, link colour, no underline | | P |
| alert | frame | `my-1` (4), `border-s-2`, `ps-3` (12) | = | P |
| alert colours | border / title | note `#2b7fffb3` / title `#155dfc`; tip `#00bc7db3` / `#009966`; important `#ad46ffb3` / `#9810fa`; warning `#fe9a00b3` / `#e17100`; caution `#fb2c36b3` / `#e7000b`; insight `#2b7fff` / `#2b7fff` | titles: note `#50a2ff`, tip `#00d492`, important `#c27aff`, warning `#fe9a00`, caution `#ff6467`, insight `#2b7fff`; borders unchanged | P |
| alert title | text | `flex items-center gap-1.5 font-medium` 500 with the alert icon 14px stroke 1.75: info, lightbulb, message-square-warning, triangle-alert, octagon-alert, star (insight); labels "Note", "Tip", "Important", "Warning", "Caution", "Insight" | | P, S |
| details | frame | `my-2` (8), top and bottom rule 1px `#e4e4e799` / `#ffffff0c` | | P |
| details trigger | text | `py-2` 36 tall, gap 8, 14/20 500 `#18181b` / `#f5f5f5`; `chevron-right` 16px muted, rotates 90° when open (150ms); summary text, or "Details" | | P |
| details content | box | `pb-3 ps-6` (12 / 24), text at 90% like the body | | P |
| artifact template | card | `my-[0.65rem]`, gap 12, radius 12, border 1px `#e4e4e7b3` / `#ffffff0e`, fill `#ffffff99` / `#11111199` (`card/60`), `px-3 py-2.5`, `shadow-xs`; icon box 36×36 radius 8 `bg-background` with a 1px `border/70` border and `shadow-xs`, holding a 20px muted icon; title 14/20 500; type 12/16 muted; `Button sm outline` "Use template" | | P |
| image | box | inline-block, `max-height: 30rem` (480), `max-width: min(100%, 30rem)`, `object-contain`, cursor zoom-in; framed workspace images `rounded-lg border border-border/40` | | S |
| image | loading slot | standalone: `aspect-video w-full rounded-lg border border-border/40 bg-muted/60`, max width min(100%, 30rem) | | S |
| image | unavailable (standalone) | the slot with `[triangle-alert 14px]` "Image unavailable · <name> · <detail>" centred, 12/16 muted; fill `#fafafa99` / `#ffffff05`, border `#e4e4e766` / `#ffffff08`, radius 8; viewed images cap at 10rem (160) | | M |
| image | unavailable (inline) | `rounded-md border border-border/40 bg-muted/40 px-2 py-1 text-xs text-muted-foreground`: radius 6, fill `#fafafa66` / `#ffffff03` | | P |
| math (setting off by default) | inline / display | inline `inline-block align-middle whitespace-nowrap`; display `block my-[0.65rem]`; viewport `px-0.5 py-0.75`, scrolls sideways; KaTeX, TeX source shown until it loads or when it fails | | S |
| mermaid | diagram | `overflow-x-auto`, centred SVG, click expands it in the image dialog; error: "Unable to render diagram: ..." 12px destructive + source | | S |

### 3.6 Code blocks and syntax colours

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| frame | fill / border | `#fafafa` (`--secondary`), 1px `#e4e4e7b3` (`border/70`) | `#ffffff09` (`input/32`, painted `#090909`), border transparent | M |
| frame | radius / margins | 8; 10.4 above and below | = | M |
| header | box | 30 tall: `padding: 6 6 0 12`, gap 8; text `#18181bb8` / `#f5f5f5b8` | | M |
| language icon | box | 14×14 Pierre icon in its file-type colour (shell `#199f43` / `#5ecc71`) | | M |
| toolbar | buttons | 24×24 each, 2 apart; wrap button `secondary` while wrapped (fill `#fafafa` / `#ffffff08`, icon `#27272a` / `#f5f5f5`), ghost-muted otherwise; copy ghost-muted `#71717b` / `#a1a1a1` | | M |
| `pre` | box | padding 12.8 / 14.4, no border, radius 0, no fill (the highlighter's background is forced transparent) | = | M |
| `pre` | text | mono 13 / 17.875, `#0a0a0a` / `#fafafa` | | M |
| `pre` | wrapping | `white-space: pre-wrap; overflow-wrap: anywhere` when wrapped (the default, setting `wordWrap` true); else `pre` with horizontal scroll | | M |
| `pre` scrollbar | style | `scrollbar-width: thin`, thumb `#e4e4e7c7` / `#ffffff10` (01-tokens §3.13: Chromium ignores the 6px webkit rule here) | | M |
| plain-text fences | colour | `text`, `txt`, `plain`, `plaintext` fences ignore token colours and inherit the body colour | | S |

Token colours. The highlighter is @pierre/diffs with its "highlights" WASM lexers (syntaxHighlighting.ts:22), themed with Zed palettes: `t3-light` in light and `pierre-dark` in dark (diffRendering.ts:7–16). `t3-light` is pierre-light with every syntax colour darkened in 4% steps (`floor(channel × 0.96)`) until it reaches 4.5:1 against a surface of luminance 0.69 (6:1 for `#1ca1c7`, `#636363`, `#1a85d4`) (readableSyntaxTheme.ts:1–76). The table lists every scope of the shipped palettes; `t3-light` was computed with T3's algorithm and every colour that appeared live matched it (M: `#693acf`, `#8f4e17`, `#b12251`, `#006781`, `#106a2b`, `#0b5267`, `#4b4b4b`, `#5c5c5c`; dark `#9d6afb`, `#ffa359`, `#ff678d`, `#08c0ef`, `#5ecc71`, `#68cdf2`, `#636363`, `#737373`).

| Scopes | pierre-light | rendered light (`t3-light`) | rendered dark (`pierre-dark`) |
|---|---|---|---|
| editor foreground | `#0a0a0a` | `#0a0a0a` | `#fafafa` |
| `class`, `class_name`, `selector.class`, `attribute` | `#18a46c` | `#0e6944` | `#60d199` |
| `comment`, `comment.doc`, `markup.strikethrough`, `markup.quote` (italic), `hint` | `#737373` | `#5c5c5c` | `#737373` |
| `decorator`, `attribute.builtin` | `#1a85d4` | `#0d4c7d` | `#69b1ff` |
| `diff.delta` | `#d5a910` | `#735b02` | `#ffca00` |
| `diff.minus` | `#d52c36` | `#b3242c` | `#ff2e3f` |
| `diff.plus` | `#18a46c` | `#0e6944` | `#07c480` |
| `embedded` | `#0a0a0a` | `#0a0a0a` | `#fafafa` |
| `function`, `function.method`, `function.builtin`, `function.special.definition`, `function.call`, `function.method.call`, `method_definition`, `selector.id`, `id_name`, `markup.link.text` | `#693acf` | `#693acf` | `#9d6afb` |
| `keyword`, `keyword.directive`, `keyword.control.at-rule`, `at_keyword`, `keyword.important`, `punctuation.special`, `preproc`, `markup.italic` (italic), `link_uri` | `#d32a61` | `#b12251` | `#ff678d` |
| `keyword.operator`, `operator`, `selector.pseudo`, `pseudo_class_selector`, `pseudo_element_selector` | `#08c0ef` | `#006781` | `#08c0ef` |
| `number`, `boolean`, `value`, `string.plain`, `plain_value`, `unit`, `number.unit` | `#1ca1c7` | `#0b5267` | `#68cdf2` |
| `predictive` (italic) | `#8a8a8a` | `#5c5c5c` | `#636363` |
| `property.css`, `property.definition`, `property_name`, `markup.link.url`, `link_text`, `primary` | `#009fff` | `#00609f` | `#009fff` |
| `punctuation`, `punctuation.bracket`, `punctuation.delimiter`, `punctuation.list_marker` | `#636363` | `#4b4b4b` | `#636363` |
| `string`, `string.template`, `template_string`, `text.literal`, `markup.raw`, `markup.raw.inline`, `markup.raw.block` | `#199f43` | `#106a2b` | `#5ecc71` |
| `string.escape`, `string.special` | `#16a994` | `#0b685b` | `#61d5c0` |
| `string.regex` | `#17a5af` | `#0d6971` | `#64d1db` |
| `string.special.symbol`, `constant`, `constant.css`, `color`, `constant.color`, `markup.bold` (700), `variant` | `#d5a910` | `#735b02` | `#ffd452` |
| `tag`, `tag.css`, `tag_name`, `tag.jsx`, `selector`, `markup.heading` (700), `title` (700), `markup.list`, `markup.list.numbered`, `markup.list.unnumbered` | `#d5512f` | `#a43d23` | `#ff855e` |
| `type`, `type.builtin`, `type.class`, `constructor`, `tag.component`, `enum` | `#a631be` | `#912ba7` | `#d568ea` |
| `variable`, `variable.member`, `variable.scss`, `variable.css`, `property`, `property.custom`, `property.object`, `property_identifier`, `shorthand_property_identifier`, `shorthand_property_identifier_pattern` | `#d47628` | `#8f4e17` | `#ffa359` |
| `variable.builtin`, `variable.special`, `variable.language`, `this`, `self`, `label`, `namespace` | `#d5901c` | `#7e540e` | `#ffab16` |
| `variable.parameter` | `#636363` | `#4b4b4b` | `#a3a3a3` |

The same palettes colour JSON in tool output (`SyntaxTokens`, V2ItemInspector.tsx:43–64) and shell commands in expanded command rows (`ShellCommandBlock`, with embedded-script detection), both measured.

**Pierre file icon colours.** Code headers, file chips, file cards and changed-file rows draw Pierre icons from one shared SVG sprite (`<svg viewBox="0 0 16 16" data-pierre-icon data-icon-token><use href="#<icon name>"/></svg>`, `size-4` unless the caller resizes it). The sprite gives the shape; the colour is an inline `color` picked by the icon's token and the scheme (PierreEntryIcon.tsx:7–60, 76–95). A path with no Pierre icon falls back to lucide `file` or `folder` in `text-icon-muted`. M marks the colours seen live in this pass.

| Light | Dark | Tokens |
|---|---|---|
| `#1a85d4` | `#69b1ff` | typescript (M), python, c, cpp, docker, biome, vscode, webpack |
| `#199f43` | `#5ecc71` | bash (M), markdown (M dark), vue, svgo |
| `#d47628` | `#ffa359` | json, html, rust, svg, swift, zig, zip, claude |
| `#d5a910` | `#ffd452` | javascript, babel, browserslist |
| `#693acf` | `#9d6afb` | css, eslint, bootstrap, terraform, wasm |
| `#a631be` | `#d568ea` | astro, database, vite |
| `#1ca1c7` | `#68cdf2` | go, react, tailwind, oxc |
| `#17a5af` | `#64d1db` | mcp, prettier, table |
| `#d32a61` | `#ff678d` | graphql, image, sass |
| `#d52c36` | `#ff6762` | npm, postcss, ruby, svelte, yml |
| `#ff8c5b` | `#d5512f` (M dark) | git (the only token that is lighter in light than in dark) |
| `#594c5b` | `#79697b` | bun |
| `#84848a` (M) | `#adadb1` (M) | default, text, nextjs, stylelint |

### 3.7 Work log

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| turn fold | row | `pt-1` 4, `pb-2` 8, `pe-0.5` 2, bottom rule 1px `#e4e4e799` / `#ffffff0c` (painted `#eeeeef` / `#0c0c0c`) | | M |
| turn fold | button | 22.75 tall, `px-1` 4, gap 4, radius 6, 14/22.75 muted `#71717b` / `#a1a1a1`, hover `#18181b` / `#f5f5f5` (150ms); width hugs the label | | M |
| turn fold | chevron | `chevron-right` (closed) / `chevron-down` (open), 14px, stroke 1.75, muted | | M |
| turn fold | timestamp | 12/16 muted; hidden (`opacity: 0`, `position: absolute`, no pointer) until the row is hovered or holds focus, then static at the row's right end (`ms-auto`, `me-1`) | | M |
| group header | button | 28 tall (`py-0.5`), `px-0.5` 2, radius 6, transparent at rest and on hover | = | M |
| work icons | box | 24×24 slot, the 16px icon centred; `vertical-align: calc(0.5cap - 0.75rem)` puts the icon slot's top at the row top and the label line 1.09 lower | = | M |
| work icons | colour | `--icon-muted` `#71717b`, `opacity: .7`, `filter: brightness(.6)` (painted `#7a7a7f`, M pixel) | `#a1a1a1`, `opacity: .7`, no filter (painted `#717171`, M pixel) | M |
| work icons | stroke | 2px (`stroke-2`); the browser glyph is a custom 24-unit SVG at stroke 1.9 | | M, S |
| work label | text | 14/22.75 `#71717b` / `#a1a1a1`, one line, ellipsis | | M |
| group list | box | `max-height: calc(min(18rem, 50dvh) + extra)`, `scroll-padding-block: 24px`, `scrollbar-gutter: stable`, radius 6, focus ring inset 2px ring/70 | | M |
| group list | edge fades | `virtualized-scroll-fade` top and/or bottom while scrolled away from that edge | | S |
| entry row | box | 24 tall (`py-0`), `px-0.5`, radius 6, cursor pointer when it can expand | | M |
| entry chevron | icon | `chevron-right` 12px in a 16px slot, `--icon-muted` at `opacity: .7` (painted `#9a9aa1` / `#717171`), `rotate(90deg)` when open, 200ms | | M |
| failed entry | icon tint | `text-tool-error-icon/40`: `#fb2c3666` / `#fca5a566`, then the icon's own `opacity .7` and (light) `brightness(.6)` (painted `#e0bdbf` / `#472e2e`) | | M |
| failed entry | trailing mark | `x` 12px in the same tint, when the tool has its own image icon | | S |
| severe failure | style | icon `circle-alert` in `--destructive`, heading `font-medium text-destructive` | | S |
| warning entry | style | `circle-alert` `text-warning`, heading `font-medium text-warning` (`runtime.warning`) | | S |
| details, text | box | `ms-7` 28, `max-height: 384px`, column gap 12, `px-0.5 py-1`, selectable; reasoning markdown in `text-foreground` 14/22.75 | | M |
| details, panel | box | `mt-0.5` 2, `mb-1.5` 6, `pb-2` 8 | = | M |
| inspector | text | 12/16 base; mono block 13/21.125, `pre-wrap`, selectable | | M |
| inspector args | style | `text-foreground/85` `#18181bd9` / `#f5f5f5d9`; keys muted; values that are JSON, HTML or REPL code highlighted | | M |
| inspector `pre` | box | `max-height: 320px`, scrolls, muted | | M |
| command output | box | `max-height: 320px`, scrolls and follows the end (8px slack), muted mono 13/21.125; ANSI colours from `TERMINAL_PALETTES`; "Earlier output not shown" italic when truncated | | M, S |
| command output | states | "Loading output…" italic muted; "No output." italic muted; "Couldn't load output: <error>" destructive; "exit <n>" destructive for a non-zero exit | | S |
| file change | inspector | path mono 12/16 muted; `+12` `text-diff-addition` / `-3` `text-diff-deletion` when known; `Button xs outline` "Open diff" (24 tall, 12/16 500, fill `#ffffff` / `#ffffff09`, border `#d4d4d8` / `#ffffff1c`, `shadow-xs`) | | M |
| error row | style | `WorkLogRow` wrap: `circle-alert` 16px stroke 1.75 `#fb2c36` / `#fb414a`; title 14/20 500 `--destructive`; timestamp always shown; message `ms-7 py-1` 14/22.75 `#18181bcc` / `#f5f5f5cc`, `pre-wrap` | | M (light), S (dark) |
| usage limit | style | same row in `text-warning`: "Usage limit reached. Retry after tomorrow at 9:00 AM (UTC)." with the viewer's zone name, no message body; `text-warning` is `#fe9a00` in both schemes | | S |
| retry | button | `Button xs outline` [rotate-ccw] "Retry" at `ms-7 pb-1` (failed workspace preparation) | | S |
| notification | style | `info` tone: icon wrapper `text-muted-foreground`, label `#18181bcc` / `#f5f5f5cc` | | M |
| compaction | divider | 24 tall, `py-1`, gap 12, 12/16 muted; rules `#e4e4e7b3` / `#ffffff0e`; `minimize-2` and `arrow-right` 12px; failed: destructive text, rules `bg-destructive/30`, `circle-alert` 12px, "· <reason>" at opacity .8 | | M (normal), S (failed) |
| attempt fold | button | 34 tall, `px-2.5 py-2`, gap 8, radius 6, border 1px `#e4e4e799` / `#ffffff0c`, fill `#fafafa33` / `#ffffff02`, hover `bg-muted/35`; label 12/16 500 `#18181bcc` / `#f5f5f5cc`; sub 11/16 muted | | P |
| history skeleton | rows | `space-y-2 px-1 pb-1 pt-3`, three rows of a 12px square + a 10px pill (40%, 60%, 33% wide) | | S |
| approval row | label | mono 12/20 foreground, `max-height: 160px` scroll, selectable; actions 12/16: `shield` 12px + "Waiting for approval" in `text-warning` (`--warning` `#fe9a00` in both schemes, not `--warning-foreground`), "·", InlineButtons (Approve in foreground, others muted), "More" menu | | S |

### 3.8 Live rows

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| working row | frame | `pt-1` 4, `pb-2` 8, bottom rule 1px `#e4e4e799` / `#ffffff0c`, row `pb-1.5` 6 | | M |
| working row | line | 24 tall, `px-1` 4, gap 8, 14/22.75 muted tabular; "Working for 2m 14s" ticks every second | | M |
| thinking row | box | 28 tall (`min-h-6` + `py-0.5`), width hugs the content, `overflow: hidden`, radius 6 | | M |
| thinking row | base | `brain` 16px icon `#71717b` at .7 + brightness .6, label muted 14/22.75 | | M |
| shimmer band | layer | `live-activity-focus`: `left: -72px`, `width: calc(100% + 9rem)`, mask `linear-gradient(to right, transparent 0, #0000001f 10.8px, #0000008c 25.2px, #000 36px, #0000008c 46.8px, #0000001f 61.2px, transparent 72px, transparent 100%)`, moving by its own width in 2.2s `steps(30)` | = | M |
| shimmer band | content | a copy of the row in full foreground: icon and label `#18181b` / `#f5f5f5`, no opacity or filter | | M |
| live tool label | style | `live-tool-shine`: text clipped to `linear-gradient(to right, transparent, --contrast-foreground, transparent)` 72px wide over `--contrast-secondary-label`; 2.2s `steps(30)` | (01-tokens §3.12) | P |
| play state | variable | `--visible-animation-state: running` only while on screen, visible and without reduced motion; `--visible-animation-will-change: transform` | | M |
| subagent card | box | 45.875 tall, `px-2 py-1.5`, gap 10, radius 6; hover `bg-accent/20` | = | M |
| subagent avatar | box | 24×24 pill, border 1px `#e4e4e7b3` / `#ffffff0e`, fill `#fafafa` / `#ffffff08`, 2px ring in the canvas colour | | M |
| subagent dot | box | 8×8 at `right/bottom: -1px`, 2px ring; status labels and dots: "Queued", "Running", "Waiting" `#2b7fff` (`--info`), "Idle · resumable" muted/50, "Completed" `--success`, "Failed" `--destructive`, "Stopped" (cancelled or interrupted) muted/60 (V2LifecycleRow.tsx:171–183) | | M (running), S |
| subagent text | | title 12/16 500 foreground; detail 11/17.875 muted (destructive when failed); status 12/16 muted; elapsed 12/16 tabular | | M |
| subagent chevron | icon | `chevron-right` 14px, `#71717b99` / `#a1a1a199`, foreground on hover | | M |
| subagent group | trigger | 46 tall (`py-2`), `pr-3.25` (13), gap 10; avatars 24×24 overlapping by 6 (x step 18), border 1px `#e4e4e7b3` / `#ffffff0e`, fill `#fafafa` / `#ffffff08`, ring 2px `#fcfcfc` / `#000000`, provider glyph 14px; `+N` pill 24×24 10/14 500 `#71717b` / `#a1a1a1` on `--muted` with the same ring; title 12/16 600 `#18181b` / `#f5f5f5`; status summary 10/14, `#2b7fff` (`text-info`) while any member is active, destructive when one failed, else muted; elapsed 12/16 muted tabular; `chevron-down` 14px stroke 1.75 muted, `rotate: 180deg` when open (150ms) | | M |
| subagent group | idle | the whole trigger in muted (`#a1a1a1` in dark) at opacity .55, back to 1 on hover (`transition-opacity`, 150ms) | | M (dark), S (hover) |
| subagent group | panel | border `#e4e4e799` / `#ffffff0c`, fill `#ffffff4d` / `#1111114d` (`card/30`), radius 8, `p-1`, `mt-1 mb-1`; opens and closes without a height animation | | M (dark), P (light) |
| subagent hover card | popup | Tooltip popup in the dropdown glass: 320 wide (`max-w-80`), radius 6, fill `#ffffffd5` / `#111111d5`, border `#18181b1a` / `#f5f5f51a`, `blur(12px) saturate(1.14)` / `blur(16px) saturate(1.08)`, shadow `0 20px 25px -5px #00000040, 0 8px 10px -6px #00000040`; content `px-2 py-1` > `px-1 py-2 gap-2` | | M |
| subagent hover card | text | title 12/15 500 foreground; rows 12/16 muted, 6 apart, `pl-0.5`; model `#18181bbf` / `#f5f5f5bf` (foreground/75) after a 12px provider glyph at `grayscale opacity-60`; status capitalised 500 with a 12px icon: `circle-dashed` `--info` while running, `circle-x` `--error` when failed, `check` `--success` `#00bc7d` when completed, else muted; elapsed on the right of the status row; preview row `terminal` 12px + MiddleTruncate | | M |
| ThreadNow line | box | 24 tall, `px-1 pb-2`, gap 6, 12/16 muted, baseline aligned; icon 14px (`terminal` for commands, `circle-dashed` otherwise, `refresh-cw` for sync); description in muted/80 (light) or muted/60 (dark); actions gap 10: elapsed, InlineButton "Details", "Open" (muted), "Stop" | | M |
| ThreadNow actions | text | InlineButton 12px 500 `#18181b` / `#f5f5f5`; muted tone `#71717b` / `#a1a1a1` | | M |
| background work dialog | content | `dialog-glass`, title "Background work" 20/27.5 600; per task: title 14/20 500 + "Waiting" / "Running" 12px muted; command box `rounded-md border border-border bg-muted/20 p-3` mono 12/16 `pre-wrap`, command `text-foreground/85`; live output panel; "Open thread" | | M (light) |

### 3.9 Dividers, plans and cards

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| divider | row | 40 tall (`py-2` + 24 pill), gap 8, 11/16 muted | | M |
| divider | rules | 1px `#e4e4e7b3` / `#ffffff0e` (painted `#ebebed` / `#0e0e0e`) | | M |
| divider | pill | `px-2 py-1` 8 / 4, pill radius, no fill; label 500 gap 6 with a 12px icon (stroke 1.75); detail at opacity .7 | | M |
| divider with action | pill | `px-2.5 py-1` 26 tall, border 1px `#e4e4e7b3` / `#ffffff0e`, fill `#fcfcfc` / `#000000`, hover `bg-accent` + foreground text; tooltip = the action ("Open parent thread", "Open source conversation", "Open fork"). The row is then 42 tall. Label 11/16 500 with a 12px icon (stroke 1.75); detail at opacity .7 in `inline-flex items-center max-w-80 truncate` | | M (dark, "Subagent of"), P (light) |
| divider, danger | colour | `text-destructive` (failed handoff) | | S |
| plan card | box | radius 16, border 1px `#e4e4e7cc` / `#ffffff10`, fill `#ffffffb3` / `#111111b3` (`card/70`), padding 20 (16 below 640) | | P |
| plan header | | Badge secondary "Plan" 18 tall, 12/16 500, `#27272a` on `#fafafa` / `#f5f5f5` on `#ffffff08`, radius 4; title 14/20 500 foreground, truncates; actions `Button icon-xs outline` [ellipsis 16px] → menu "Copy to clipboard" ("Copied!"), "Download as markdown", "Save to workspace" | | P |
| plan body | | `mt-4` 16; collapsed clip `max-height: 416px` (26rem) with a 96px fade `linear-gradient(to top, #fffffff2, #ffffffcc 50%, transparent)` / `#111111f2`, `#111111cc`; `Button sm outline` "Expand plan" / "Collapse plan" centred `mt-4` | | P |
| plan rules | | collapsible only when the markdown is over 900 characters or 20 lines; collapsed, it renders a preview of the body up to its 10th non-empty line, trailing blank lines dropped and a final "..." paragraph added when more follows (`buildCollapsedProposedPlanPreviewMarkdown`), not the clipped full plan, except while find is open. Title = the first markdown heading, else "Proposed plan"; the body drops that heading and a following "Summary" heading; headings inside shift by 3 levels. A failed copy toasts "Could not copy plan" (ProposedPlanCard.tsx:64–100, proposedPlan.ts:3–40, proposedPlanText.ts:1–20) | | S |
| created thread summary | | radius 8, border `#e4e4e799` / `#ffffff0c`, `p-3`, gap 12, 62 tall; icon box 36×36 radius 8 fill `#fafafa99` / `#ffffff05` with `message-square` 16px secondary-label; title 14/20 500 | | P |

### 3.10 Overlays

| Element | Property | Light | Dark | M/P/S |
|---|---|---|---|---|
| scroll to end | placement | wrapper spans the lane (`left 0`, `right` = lane end inset, 200 with the card), `bottom: 144px` = composer clearance 140 + 4, `py-1.5`, z 30 | | M |
| scroll to end | button | `Button xs glass`: 24 tall, 7px side padding (8 minus the 1px border), gap 4, pill, 12/16 500 `#18181b` / `#f5f5f5`; fill `#fcfcfccc` / `#000000cc` with `blur(12px) saturate(1.14)` / `blur(16px) saturate(1.08)`; border 1px `#e4e4e799` / `#ffffff0c`; `shadow-sm`; icon `chevron-down` 14px muted | | M |
| scroll to end | summary | " · 2 queued", " · <child waiting label>" in muted after the label | | S |
| minimap | box | 72 wide (`w-18`) at the canvas's left edge, full height, z 40; visible when the gutter beside the lane is ≥ 48px (224 at 1440), else only on hover with a fine pointer | | M |
| minimap rail | box | starts 12px in, vertically centred; height `min((n-1)×8px, max(4rem, 100% - 14.75rem))` (88 for 12 items); hit strip width `clamp(floor(gutter) - 12, 0, 40)` (40 here) | | M |
| minimap hairline | line | 1px at x + 12, `#e4e4e726` / `#ffffff03` | | M |
| minimap strips | box | 24×2 pill from the rail's left, `transform-origin: left`, `scale-x` 0.333 at rest (8px) | | M |
| minimap strips | colour | `#71717b40` / `#a1a1a140` (muted/25); hovered strip `#71717b99` / `#a1a1a199` at full width | | M |
| minimap fisheye | scale | hovered 1 (24px), ±1 0.667 (16), ±2 0.417 (10), others 0.333 (8); 150ms | | M |
| minimap in view | overlay | `::after` `#18181b8c` / `#f5f5f58c` (foreground/55) at opacity 1 for strips whose row is in view, 150ms | | M |
| minimap preview | card | `dropdown-glass` 320 wide, radius 12, `p-3`, border 1px `#18181b1a`, fill `#ffffffd5` + `blur(12px) saturate(1.14)`; shadow `0 20px 25px -5px #00000040, 0 8px 10px -6px #00000040`; user line 14/20 500 one line `#18181b`; reply 14/20 muted `#71717b`, `max-h-[3.75rem]` with a 3-line clamp, `mt-1` | fill `#111111d5`, border `#f5f5f51a`, `blur(16px) saturate(1.08)`, same shadow; user line `#f5f5f5`, reply `#a1a1a1` | M |
| minimap nav | buttons | `Button icon-micro ghost-muted` 20×20 with a 16px chevron in `text-foreground/90`, 6px beyond the rail ends, opacity 0 until the button itself is hovered or focused (150ms) | | M |
| cite toolbar | button | `Button xs glass`: 24 tall, 7px side padding, gap 4, pill, 12/16 500; fill, blur, border and shadow as the scroll-to-end button; `quote` 14px muted; fixed, z 50 | | M |
| image dialog | backdrop | `#000000bf` (black 75%), no blur, z 60 | = | M |
| image dialog | viewport | grid centred, padding 24 top and bottom, 16 sides | = | M |
| image dialog | popup | 92vw × 92vh (1324.8 × 828), transparent; media width `92vw` (`92vw - 96px` from 640px up, leaving room for the arrows) | = | M |
| image dialog | close | `Button icon-sm media-close` 28×28 at the popup's top-right: fill `#000000a6`, ring `0 0 0 1px #ffffff33` + `shadow-sm`, `x` 16px white; focus-visible adds an inset white ring | = | M |
| image dialog | image frame | radius 8, fill `--background` (`#fcfcfc` / `#000000`), ring 1px `#e4e4e7b3` / `#ffffff0e` (border/70), `shadow-2xl` (`0 25px 50px -12px #00000040`); max height `min((100vh - titlebar) × 0.86, 100vh - titlebar - 160px)` (740 at 900) | | M |
| image dialog | caption | `mt-2` 8, gap 6, 12/16 `#ffffffcc`, "name (2/5)" | = | M |
| image dialog | failure box | `h-48 w-[min(--media-width,32rem)] rounded-lg border border-border/70 bg-black p-6 text-sm text-white` with "This image could not be loaded." or "Image unavailable. The file may have been moved or deleted." | | S |

### 3.11 Icon inventory

All lucide, stroke 1.75 unless noted.

| Where | Icon (lucide name) | Size | Colour |
|---|---|---|---|
| turn fold | `chevron-right`, `chevron-down` | 14 | muted |
| attempt fold | `chevron-right`, `chevron-down` | 14 | muted |
| compaction | `minimize-2`, `arrow-right`, failed `circle-alert` | 12 | text colour |
| work entry, by kind | `terminal` (command), `eye` (read), `square-pen` (edit), `search` (code search), `globe` (web search), `wrench` (other tool, dynamic tool entry), `hammer` (mixed or dynamic-tool group, MCP elicitation approval), `brain` (reasoning), `bot` (subagent, agent tools), `zap` (tone tool, background task notice), `list-todo` (todo list), `message-circle` (questions, approvals), `check` (info tone), `circle-alert` (error tone, failures), `code-xml` imported as `Code2Icon` (JavaScript REPL), `smartphone` (device), custom browser glyph (browser, computer use), T3 wordmark (T3 tools, created threads, system notices), PR glyph (pull request tools) | 16, stroke 2 | §3.7 |
| work entry | `chevron-right` (expand), `x` (failure mark) | 12 | icon-muted .7 / tool-error/40 |
| tool images | website favicon (32px source), themed logo, native app icon; `rounded-xs bg-background`, dark opacity .7, light `brightness(.7)` | 16 (native apps 20) | |
| error row | `circle-alert` | 16, stroke 1.75 | destructive / warning |
| retry | `rotate-ccw` | 16 (button default) | |
| live work | as the entry, `brain` for thoughts | 16, stroke 2 | |
| working row | `minimize-2` (compacting) | 12 | muted |
| user message | `redo-2` (steer) 12; `messages-square`, `calendar-clock` 16 stroke 2; `chevron-down` 14; `pencil` 12; `copy` 12; `link` 12; `check` 12 (copied, `text-primary`); `eye` 16; `download` 14 | | |
| assistant meta | `copy`, `git-fork`, `reply`, `check` | 12 | muted |
| code block | `text-wrap`, `play`, `copy`, `check`, mermaid `code`, `workflow` | 12 | |
| code language | Pierre file icons | 14 | file-type colour |
| table | `copy`, `check` | 12 | muted |
| details | `chevron-right` | 16 | muted |
| alerts | `info`, `lightbulb`, `message-square-warning`, `triangle-alert`, `octagon-alert`, `star` | 14 | alert colour |
| links | favicon image or `globe`; source-control brand glyphs | 14 | |
| media | `triangle-alert` (unavailable) | 14 | muted |
| dividers | `circle-stop` (run interrupted), `git-fork` (fork), `eraser` (new session), `arrow-right-left` and `arrow-right` (handoff), `bot` (subagent of), `messages-square` (context from) | 12 | text colour |
| subagent | `bot` (avatar fallback) 14, `chevron-right` 14, `chevron-down` 14 (group) | | |
| changed files | `chevrons-down-up`, `chevrons-up-down`, `file-diff` 12; `chevron-right`, `folder`, `folder-closed` 14 | | |
| ThreadNow | `terminal`, `circle-dashed`, `refresh-cw` | 14 | muted |
| approvals | kind icon 16 stroke 2; `shield` 12; `triangle-alert` 12 (risky options) | | |
| plan | `ellipsis` | 16 | |
| scroll to end | `chevron-down` | 14 | muted |
| minimap | `chevron-up`, `chevron-down` | 16 | foreground/90 |
| cite | `quote` | 14 | muted |
| image dialog | `x` 16; `chevron-left`, `chevron-right` 20; `image`, `text` 12 | | white |
| snapshot overlay | `image` (no accessibility data), `text` (has it) | 12 | white/65 on black/70 |
| queued turns | `arrow-up` ("Send now"), `pencil`, `x` 14 (button defaults); `clock-3` 12 ("Saving"); `calendar-clock`, `messages-square` 16 stroke 2 (folds); `chevron-down` 14 | | muted |
| subagent hover card | provider glyph, `bot`, `circle-dashed`, `circle-x`, `check`, `git-branch`, `folder`, `terminal` | 12 | status colour or muted |
| link / secret requests | `link`, `check`, `x`, `minus`, `lock` 14 (answered rows); `external-link`, `shield-check` 14 | | icon-muted / muted |
| unresolved thread link | `circle-dashed` | 1em | chip ink |

### 3.12 Queued turns (M, both schemes, an automation message in the queue)

| Element | Property | Light | Dark |
|---|---|---|---|
| section | box | lane width (736), `pb-4` 16, gap 8; list items right-aligned, 12 apart | = |
| item | box | column, gap 4 between the bubble and the actions row | = |
| bubble | frame | `max-w-[80%]`, radius 16, 1px dashed border `#71717b66` (muted/40), no fill, text 14px `#71717b` | border `#a1a1a166`, text `#a1a1a1` |
| folded header | box | 36 tall, `px-3 py-2`, gap 6, radius 16, 14/20; icon 16px stroke 2, label `max-w-1/2` and summary both muted (the queued bubble makes the summary muted too) | = |
| actions row | box | 24 tall, gap 2, `pe-1`, 12px muted; "Send now" `Button xs ghost-muted` 24 tall, 7px side padding, gap 4, 12/16 500; edit and remove `icon-xs` 24×24; icons 14px | colours `#a1a1a1` |
| reveal span | box | `me-1 px-1`, gap 8, opacity 0 until the item is hovered or holds focus (200ms) | = |

## 4. States

| Component | State | Value | M/P/S |
|---|---|---|---|
| turn fold | default / hover / focus-visible | muted text / foreground text (150ms) / inset ring 2px `#1b4ed8b3` / `#346bf1b3` (ring at 70%) | M |
| turn fold | open | chevron-down; `aria-expanded=true`; the hidden rows render below it | M |
| turn fold | loading | `aria-busy`; skeleton below while deferred history loads; failure appends " · Couldn't load work. Click to retry." and the next click retries | S |
| any timeline row | hover / focus within | its timestamp shows (static, opacity 1, takes space at the row end, so labels truncate further) | M |
| work entry | hover | no fill change; timestamp shows | M |
| work entry | focus-visible | inset ring 2px ring/70 | S (class) |
| work entry | expanded | `aria-expanded=true`, chevron rotated 90°, details below; expansion persists per thread for the session | M |
| work entry | failed | tinted icon (or a trailing `x`), `aria-label` ends ", tool call failed" | M |
| work entry | in progress | inside an expanded live group: stays listed even when its status is neutral | S |
| group header | failed | `aria-label` ends ", tool call failed"; visual unchanged | M |
| user actions, assistant meta | hidden / shown | opacity 0 / 1 on row hover, focus within, or always on touch | M |
| copy buttons | copied | icon `check` in `--primary`, button disabled for 1s, anchored toast "Copied!" (tooltip style); error toast "Failed to copy" | S |
| code copy | copied | `check`, label "Copied", 1.2s, no toast | S |
| code wrap | pressed | `secondary` variant and `aria-pressed=true` while wrapped | M |
| fork | busy | disabled, icon `animate-pulse` | S |
| "Edit from here" | disabled | while a checkpoint revert runs or the thread is working (opacity .64) | S |
| folded message | open | chevron rotated 180°, body visible, bubble full width, actions row appears | M |
| long user message | collapsed / expanded | 176px clip with the 28px fade / full height, "Show less" | M |
| file chip | hover | label underlined in the chip colour; context menu on right click (open, open in browser, reveal, copy relative / full path) | M, S |
| link | hover, focus-visible | dotted underline; tooltip with the href | M |
| subagent card | hover | `bg-accent/20` (`#f4f4f533` / `#ffffff02`; Chrome reads the light fill back as `#f5f5f533`), chevron to foreground (`#18181b` measured); hover card after 200ms (V2LifecycleRow.tsx:509) | M (light), S (dark) |
| subagent group | idle / active / open | opacity .55 muted / full foreground / full foreground; open also rotates the chevron 180° and shows the panel without animation, and the open state is kept per thread for the session | M |
| attempt fold | hover | `bg-muted/35` | S |
| changed files row | hover | `bg-accent/60`, folder name to foreground/90, file name to foreground | S |
| divider action | hover | `bg-accent`, foreground text | S |
| plan card | collapsed | 416px clip with fade, "Expand plan" | S |
| live rows | running | shimmer and shine animate while on screen; paused when off screen, when the window is hidden, or under reduced motion | M |
| live rows | reduced motion, forced colours | `live-tool-shine` paints plain text; the focus band hides (opacity 0) | S |
| approval | responding | buttons disabled; provider gone: "Provider process is gone. Interrupt or restart the run to respond." in warning | S |
| media | loading | standalone: empty video-ratio slot; inline: an empty inline-block | S |
| media | failed | "Image unavailable · <name> · <detail>" box | M |
| scroll to end | shown | after 150ms of being away from the end; hidden at once when the end is reached | S |
| minimap | strip in view | `::after` foreground/55 at opacity 1 | M |
| minimap | hover | fisheye scales and the preview card | M |
| cite toolbar | disabled | selection longer than the citation limit: "Shorten selection", disabled | S |
| timeline | empty | "Send a message to start the conversation." 14/20 muted, centred above the composer | S |
| timeline | loading | an empty `bg-background` block (`data-timeline-loading`) while thread detail loads or the draft hero shows | S |
| streaming markdown | new block | each new top-level block fades in over 600ms ease-out (`@starting-style`); finished threads never replay it | S |
| buttons in rows | pressed | `scale: 0.97` while the pointer is down, as every Button (02-primitives §4.1); the wrap toggle keeps its pressed state as the `secondary` variant | M |
| history control | error | the load error shows below the button as a centred 12px muted `p[role=status]`; the button stays enabled | S |
| history control | hover / focus-visible / disabled | text to foreground (`hover:text-foreground`) / 2px `--ring` outline / default cursor while loading (`disabled:cursor-default`) | S (classes), M (rest: 28 tall, 12/16 `#a1a1a1`, rules `#ffffff0e` in dark) |
| queued turn | hover / focus within | the reveal span (sender link, automation link, timestamp) fades in, 200ms | S |
| queued turn | editing | border `primary/60`, text foreground, "Editing in the composer" + "Cancel" replace "Send now", edit and remove | S |
| queued turn | saving / busy | "Saving" with `clock-3` until the server has the message; while any queued action runs, every send, edit and remove button is disabled (opacity .64) | S |
| link / secret request | declining / submitting | buttons disabled; an error shows as a 12px destructive `p[role=alert]` | S |
| mermaid | error | "Unable to render diagram: <message>" 12px destructive, optional `Button xs ghost` "Retry", then the source in mono 12px (MermaidDiagram.tsx:302–316) | S |
| data attributes | state hooks | `data-streaming` (markdown root), `data-user-message-collapsed` / `data-user-message-intent` / `data-user-message-attribution`, `data-wrap` and `data-language` (code block), `data-markdown-details-open`, `data-changed-files-state`, `data-timeline-row-kind` / `data-timeline-row-id` / `data-message-role`, `data-work-entry-id` / `data-v2-item-type` / `data-v2-item-visibility`, `data-tool-group-scroll`, `data-in-view` / `data-persistent-gutter` (minimap), `data-scroll-to-end`, `data-thread-now-line`, `data-timeline-loading`. None of them changes styling except `data-wrap` (pre-wrap), `data-streaming` (block fade-in), `data-in-view` (strip overlay) and `data-language` (plain-text fences) | S |

## 5. Behaviour

### 5.1 Data each row needs

The timeline is built from `TimelineEntry` (apps/web/src/session-logic.ts:105–146): messages (`ChatMessage` with role, text, attachments, run id, input intent, created and updated times, streaming flag, sender thread, attribution fields), proposed plans, HTML renders, MCP apps, work entries and events, each optionally with its `OrchestrationV2ProjectedTurnItem` and the run attempt it belongs to. Work entries carry (session-logic.ts:68–95): `id`, `createdAt`, `runId`, `label`, `detail`, `command`, `rawCommand`, `changedFiles`, `tone` (`thinking`, `tool`, `info`, `error`), `toolTitle`, `toolData`, `viewedImagePath`, `toolSurface`, `toolIcon`, `toolSource`, `sourceActivityKind`, `requestKind`, `itemType`, `toolLifecycleStatus`, `structuredPayload`, `projectedItem`, `questionAnswer`.

Also read by the timeline: the latest run (`status`, `startedAt`, `completedAt`, `turnEnded`), the running run id, whether the thread is working, turn diff summaries (changed files per assistant message), pending approvals, checkpoints (for "Edit from here"), the provider sessions' capabilities (for fork), thread shells of linked threads (titles for thread chips, subagent titles), subagent projections (status, progress, result, times), and the client settings `toolCallVisibility`, `timestampFormat`, `wordWrap`, `mathRenderingEnabled`, `browserLinkTarget`.

The footer also reads the thread's queue projection (queued runs in server order, each with its message, `freshContext`, `editable`, the optimistic messages not yet on the server, and the workflow flags `canSendNow` and `promoteSkipsCompaction` that pick the "Send now" tooltip), the run being edited in the composer, the steer and edit shortcut labels, and the child thread's waiting label for the scroll-to-end summary. Image attachments carry `source` (`kind: "snap-shot"` with `appName`, `appIconDataUrl`, `windowTitle`, `accessibility` or `accessibleText`) for snapshot overlays.

### 5.2 How entries become rows

1. **Visibility filter** (MessagesTimeline.logic.ts:1444–1490). `showAll` (default) keeps everything. `hideSuccessful` drops completed, non-failed commands, file changes, file and web searches and dynamic tools. `hideAll` drops those five item types entirely. `zen` keeps user messages, the cited message, the final answer of each finished response (the response still being answered shows its live answer preview in the thinking row instead), in-flight steer answers, replies kept after a steer, plans, HTML renders, MCP apps, events other than subagent cards, and only these work entries: pending approvals, answered questions, compactions, notifications (they start a response) and errors (MessagesTimeline.logic.ts:1387–1405). With find open, `zen` becomes `hideAll`. Superseded in-progress reasoning is marked completed; delegation rows that duplicate a subagent card are dropped.
2. **Turn folds** (1022–1241). Each response folds, except a run still answering, a failed run, a run with an interrupt request or result in the timeline, a runless response with a failed item or active runless work, and any response with a streaming message (1129–1138, 1517). Everything between the prompt and the response's last assistant message folds under one `turn-fold` row, placed at the first entry after the prompt. The terminal message, persistent resource cards, live subagent cards, kept replies, notifications, answered questions and the latest compaction divider stay visible. Trailing work after the terminal message folds too unless it is in progress or failed. A fold needs at least one hidden non-compaction entry.
   - Label: "Worked for <duration>", or "Worked" without a duration. The latest run when interrupted: "You stopped after <duration>" / "You stopped this response". Each steer that splits a run starts its own fold below the steer: "Continued for <duration>" / "Continued".
   - Duration: the latest run's `startedAt`→`completedAt` when known; otherwise from the prompt's time (or the first entry) to the later of the terminal message's update time and the last entry's end; formatted by `formatDuration` (§5.4).
   - Folds start closed. A run that turns from running to interrupted opens its fold; when a newer run appears, the previous run's fold closes again (MessagesTimeline.tsx:596–631). Find and citations open the fold that hides their match.
3. **Superseded attempts** (797–850): the entries of an attempt cut off by a steer fold under "Superseded attempt · Cut off by a steer", except user messages, handoffs, HTML renders, MCP apps, resource cards, live subagents and system notices. An attempt with a reply that is kept after the steer does not fold, nor does one whose run is unfolded.
4. **Live tail** (1643–1723). While working, the run of trailing work entries of the active response (same attempt, not errors, notices, todos, questions, pending approvals or compactions) becomes one `work-live` row placed at the latest visible entry. Its `entry` is the latest running entry, else the latest visible one; `active` is true while a tool runs or the last one succeeded with no approval pending. It expands into the same list as a group. If the latest entry failed (a declined tool does not count), a `thinking` row with the failed group's disclosure replaces it.
5. **Working row**: a `working` row sits right after the active response's prompt, created at the run's start time ("Working for 12s"); a running compaction can own it ("Compacting for 4s"); worktree setup owns it before the agent starts.
6. **Thinking row**: while working and nothing live is visible (no activity row, no pending approval, the turn has not ended, no compaction), a `thinking` row is appended at the end.
7. **Work grouping** (1955–2106). Consecutive work entries of the same run and attempt group, except errors, failed error items, notices, notifications, todo lists, answered questions, pending approvals, compactions and entries already claimed by a fold or the live tail: those get a `work` row each. A group of one tool-like entry is a single `work` row whose label is its own (an edit shows the summary "Changed 1 file"). Anything else becomes a `work-toggle` with a summary (§5.3), plus a `work` row with the expanded list when open, plus a `viewed-images` row when the group opened images. **Failed runs are itemized**: every work entry of a run that failed, unless the only failures were usage limits, gets its own standalone `work` row with its own display label (an edit shows its file path, a command its command line), with no group and no summary (logic.ts:1005–1020, 1958). Measured on a failed run: 20 consecutive rows, 28 tall each and 36 for the last of a stretch.
8. **Compaction**: a compaction entry becomes a divider: "Compacting context" while active (or "Compacting before the cache expires…" / "Compacting…" for idle compactions), "Context compacted <before> [arrow-right 12px] <after> tokens" when counts are known (the arrow is an icon, not text), "Context compaction stopped", "Context compaction failed · <reason>". Idle compactions title themselves "Context automatically compacted before cache expiry" / "Context compacted before sending"; one answered "compact first?" question folds into its compaction. A bare `/compact` turn shows only the divider.
9. **Assistant meta** (1243–1334): when a response ends with tool groups (or a resource summary), its meta row moves below the last of them as an `assistant-meta` row.
10. **User grouping** (2310–2322): consecutive own messages (no attribution) set `groupedWithNext` / `groupedWithPrevious` (4px gap, tightened corners); a steer following a steer hides its "Steer" marker.
11. **Created threads** (2349–2397): thread-created items of a run repeat as resource summary cards right after its final answer; the inline row inside a closed fold is dropped.
12. `continuesWorkLog`: a work-log row followed by another work-log row drops its block padding, so consecutive rows sit flush.

### 5.3 Labels, summaries and icons

Entry label (`workEntryDisplayLabel`, MessagesTimeline.logic.ts:134–179), first match wins:
1. system notice: its label; reasoning: the thought text collapsed to one line, else the label;
2. a T3 tool: "<verb> <object>" from its label tuple ("Previewing an HTML page", "Previewed an HTML page", "Failed to preview an HTML page", "Declined to ...", "Stopped previewing ..."); pull request tools use "PR #12";
3. a command: the command's display text;
4. code or web search: "Searched <query> in <target>", "Searched files <glob> in <target>", "Searched files <glob>", "Searched <query>", "Searched in <target>" (packages/shared/src/toolActivity.ts:361–384); file searches use their own label;
5. reads: "Read <path>" with " +N more" (toolActivity.ts:386–393);
6. a dynamic tool's title ("Skill: <name>", "Run JavaScript", "Computer use");
7. the detail text (not for retried errors, reads, or multi-line search output);
8. the first changed file, " +N more";
9. "Read file" for reads without a path;
10. the tool title with " complete(d)" stripped, first letter capitalised.

Live labels (`liveWorkEntryLabel`, 234–266): reasoning shows the thought or "Thinking" / "Thought"; commands show "Running <program>", "Ran <program>", "Failed <program>", "Declined <program>", "Stopped <program>" ("command" when no program name parses). An expanded reasoning entry's heading reads "Thinking" or "Thought". An answered question shows the question and the answer preview (foreground while collapsed). A thread-read tool shows "Read thread "<title>"" with the title linked.

Group summaries (`summarizeToolGroup`, presentation.ts:565–652): entries are bucketed by action and counted. Labels: "Ran N command(s)", "Read N file(s)", "Changed N file(s)" (counting distinct files), "Searched code N time(s)", "Searched the web N time(s)", "Used browser N time(s)", "Used device controls N time(s)", "Created N thread(s)", "Linked / Unlinked N pull request(s)", "Watching / Stopped watching N pull request(s)", "Checked linked pull requests" ("Checked linked pull requests N times"), "Used N tool(s)", "Received N update(s)", T3 tool summaries ("tried to preview 2 HTML pages"). Commands, edits and T3 writes outrank reads and searches, which outrank "other" and "update". The two highest buckets show in their original order; tools from an integration lead as "Used <A>, <B>, and <C> integrations"; the rest becomes "Performed N other action(s)". Labels after the first are lower-cased and joined with " and " or ", ... , and ". Reasoning entries in a mixed group are left out of the counts; a group of only reasoning reads "Thought" or "Thought (×N)". The summary also reports whether any counted entry failed (`hasFailure`).

Group icon (`toolGroupSummaryIconName`, TimelineWorkEntryRow.logic.ts:12–53): the group's tool surface, else by kind: pull-request → PR glyph, read → eye, edit → square-pen, command → terminal, thread-create → T3 wordmark, browser → browser glyph, device → smartphone, search → globe, code-search → search, other → wrench, dynamic-tool / update / mixed → hammer, reasoning → brain, agent-tool → bot, tone-tool → zap. An integration's favicon or logo replaces it when the group has one.

Failure (presentation.ts:259–320): a tool failed when its tone is error, its status is failed or declined, a T3 tool reported failure, a command's exit code is non-zero (the exit code outranks output text), the command flagged its output, or its detail matches T3's failure patterns. Reasoning is never scanned.

### 5.4 Formats

| Value | Rule | Example | Source |
|---|---|---|---|
| fold and background durations | `formatDuration`: under 1s "Nms" (min 1); under 10s rounded to a tenth with a trailing ".0" dropped ("3.4s", "3s"); under 60s whole seconds "42s" (the unit is chosen after rounding, so 59.6s reads "1m 0s"); then "1h", "1h 2m 3s", "2m 0s", "1h 5s" (seconds shown whenever minutes are, zero minutes skipped); negative or non-finite input "0ms" | "Worked for 34m 10s" | orchestrationTiming.ts:22–37 |
| working timer | whole seconds under a minute ("12s"), then `formatDuration`; "0s" before the start parses; updates on the shared second tick by writing the DOM text | "Working for 1h 2m 3s" | TimelineWorkGroupRows.logic.ts:3–20 |
| subagent elapsed | "45s", "3m 07s", "2h 05m"; compact form "45s", "3m", "1.5h" (tenths of an hour below 10h), "12h"; ticks every second while the agent is active | "8m 32s" (M) | AgentElapsed.tsx:6–50 |
| subagent group status | counts in the order working (pending, running, waiting), done (completed), failed, stopped (anything else), idle, each "N <word>", joined by " · "; zero counts left out | "6 working · 3 done" (M) | subagentDisplay.ts:26–41 |
| subagent title | "Subagent" when the item has none; a leading "Subagent: " is dropped; a title that is a `/root/...` path shows its last segment with `_` and spaces collapsed and words capitalised | | subagentDisplay.ts:43–50, V2LifecycleRow.tsx:157 |
| timestamps | `formatDayAwareTimestamp`: today "4:44 PM"; yesterday "yesterday at 4:44 PM"; this year "11/3 4:44 PM"; other years "11/3/2025 4:44 PM". Locale from the desktop host (else the browser); setting `timestampFormat` `locale` (default), `12-hour`, `24-hour` | "11/3 4:44 PM" | timestampFormat.ts:184–201 |
| timestamp tooltip | "4:44 PM, 17th November 2026" (English ordinal and month, always) | | timestampFormat.ts:121–132 |
| upcoming (usage reset) | "4:00 PM", "tomorrow at 4:00 PM", "11/20 4:00 PM", plus " (Zone/Name)" | | timestampFormat.ts:204–222 |
| tokens | ≥1e3 "K", ≥1e6 "M", ≥1e9 "B", ≥1e12 "T"; 2 decimals under 10, 1 under 100, none above; only all-zero decimals are dropped ("2K", but "2.50K") | "180K", "11.7K" | usageFormat.ts:66–79 |
| file sizes | "512 B", then KB/MB/GB/TB at base 1024 with 2, 1 or 0 decimals (unit chosen after rounding) | "2.40 MB" | formatBytes.ts:1–16 |
| diff counts | under 1000 as is, then "1.2k", "12k", "1.2m", "1.2b" | "+1.7k" | DiffStatLabel.tsx:8–20 |
| file labels | pasted text shows "Pasted text"; file cards "PDF · 2.40 MB" (extension upper-cased, "File" when none) | | TimelineUserMessageRow.tsx:113–132 |

### 5.5 Truncation and clamping

- One-line ellipsis: work labels, fold labels (they hug their content but cannot exceed the row), folded message label (max 50%) and summary, subagent title and detail (a path-like detail middle-truncates), divider details (`max-w-80`), plan title, file chip labels, minimap user line, changed-file names (MiddleTruncate).
- Clamps: quoted response card 2 lines; minimap reply 3 lines (60px); live thought line 4 lines; thought previews parse at most 8192 source characters and show 2048.
- Height caps: long user messages 176px (when ≥ 352px), folded messages 384px scroll, reasoning details 384px scroll, tool output and command output 320px scroll, approval detail 160px scroll, expanded group list 288px + expanded entries, plan 416px with fade.
- Wrapping: assistant and user text wrap anywhere (`overflow-wrap: anywhere; word-break: break-word`); tables restore word-boundary wrapping so a column is at least its longest word wide, and a word longer than its cell breaks (`overflow-wrap: break-word` on cells; the declared 24rem cell cap has no effect in Chrome, §3.5); code blocks wrap by default.
- Queued turns: more than 3 collapse to the first 2 plus "Show all N queued" (QueuedRunsControl.tsx:57, 238–239); a folded queued message clips its body at 384px and scrolls.

### 5.6 User message rules

- Attribution (packages/client-runtime/src/userMessage.ts:61–117): guest context → guest bubble with the guest's name; a scheduled task (or the legacy "[Triggered by schedule task: ...]" prefix) → automation fold; `createdBy: agent` from the server → "Sent by T3 Code" above a normal bubble; any other agent → folded agent message with the sender thread's title (or "Agent").
- Media placement (TimelineUserMessageRow.tsx:499–527): images or videos without words show as media only; with words they show above the bubble. One image keeps its shape (192 tall, at least 96 wide); several images form a 320-wide collage; snapshots and videos use their own frames. Clicking opens the image dialog with all of the message's images.
- Files without words show as file cards; files referenced by a context chip in the text do not repeat as rows.
- "Copy message" copies the text; when the message has context records it also writes a context fragment (MIME `COMPOSER_CONTEXT_CLIPBOARD_MIME`) so pasting into the composer restores the chips. "Copy prompt link" copies a new-thread link for prompts of 1 to 8000 characters.
- "Edit from here" appears when the provider supports conversation rollback and a checkpoint exists for the turn before; it reverts the thread to that turn.
- Steers show "Steer"; an unread steer shows "Steer · waiting to be read" and explains in its tooltip.
- The fresh-session divider ("New session", `eraser`) and model-switch divider ("<old model> → <new model>") render above a user message that started a new provider session or changed model or effort.

### 5.7 Assistant message rules

- Streaming text comes from the live store so only the streaming row re-renders (TimelineAssistantRows.tsx:36–43). While streaming, the markdown splits into blocks and only the last re-parses (ChatMarkdown.tsx:1821–1827).
- The meta row appears only on a response's last assistant message, and not while that response is still in progress. Copy is offered when the text is non-empty and not streaming; it copies the text with Codex directives rendered.
- Reply puts a "response" context chip (the response's opening and its subagents) into the composer. Fork asks the server to fork the thread from this run.
- Changed files show when the turn has a diff summary and the caller can open diffs; the folder state is remembered per thread and run.

### 5.8 Markdown behaviour

- **Links** open in a new tab, or in the in-app browser per `browserLinkTarget` and modifiers (`resolveLinkTarget`); Alt-click in the desktop app opens externally; media links (image or video URLs) open in the image dialog; pull request links can show a preview hover and open in the PR panel; right click shows a native menu (open in browser, open externally, copy link, link or unlink the PR to the thread). `#fragment` links scroll within the message.
- **File chips** open the file in the right panel at the line, with Cmd-click (macOS) or Ctrl-click opening the preferred editor; PDFs open in the integrated browser; media files open the image dialog (MarkdownFileLink.tsx:112–123, 340–354). Positions parse from `:12`, `:12:3`, `:12-20`, `#L12`, `#L12C3` and `#L12-L20` (packages/shared/src/fileLinks.ts:8–10). The label is `name · parent · L12–20` (or `L12:C3`), where the parent folder suffix appears only when two linked files in the message share a name, as the shortest unique parent path of at least two folders (fileLinks.ts:116–179). Right click lists "Preview media", "Open in <editor>", "Open in integrated browser", "Reveal in Finder" ("Reveal in File Explorer", "Reveal in Files"), "Copy relative path", "Copy full path"; a copy confirms with a toast "Relative path copied".
- **Thread links** resolve the target thread and show a thread chip with its title; unknown targets show an unresolved chip.
- **Inline code** double-click selects the whole span; a path in backticks becomes a file chip.
- **Code blocks**: copy writes the exact code; wrap toggles per block, starting from the `wordWrap` setting (default on). "Run in terminal" shows only for `sh`, `bash`, `zsh`, `fish`, `shell`, `powershell`, `pwsh` fences that are closed, not streaming, end in a newline, are non-empty, do not end in `\`, and contain no control or invisible format characters; it runs the trimmed command in the thread's terminal. Highlighted HTML of finished blocks is cached (500 entries, 50 MB); a block that streamed keeps its line-by-line DOM so selections survive.
- **Tables**: copy as Markdown or CSV from the menu.
- **Details** start closed unless the HTML said `open`; the summary text, or "Details" when there is none, is the trigger; find opens them to reveal matches (MarkdownDetails.tsx:14–65).
- **Selection copy**: copying a selection inside markdown writes text and HTML in which chips, cards and code keep their markdown source (`data-markdown-copy`, `chatMarkdownClipboardPayload`, apps/web/src/markdown-clipboard.ts:442).
- **Math** renders only when the "Math rendering" setting is on (default off); otherwise `$...$` stays text.

### 5.9 Disclosure memory

Expanded folds, attempts, work groups, entries inside groups and their scroll positions are kept per thread for the session, in a map of the last 100 threads together with the scroll position (timelineScrollAnchoring.ts:131–151). Returning to a thread restores them. Expanding near the bottom scrolls just enough to reveal the new content above the composer (`resolveDisclosureRevealScrollDelta`, at most the room above the 24px anchor), smoothly unless reduced motion is on; while the live end is followed, toggles do not anchor.

### 5.10 Scrolling and follow

- **Opening a thread** scrolls to the end, unless the thread was left scrolled up in this session (then the remembered row and offset are restored) or a citation or find target takes over.
- **Follow mode** (timelineViewport.ts:249–283) is `following` or `released` (with `atEnd`). Following keeps the end in view as rows grow (`maintainScrollAtEnd` on data, item layout and layout changes; animated while working unless reduced motion). The end counts as reached within 40px of the bottom (MessagesTimeline.logic.ts:324–336).
- **What releases follow** (timelineViewport.ts:221–241): a wheel up when the content overflows; Up keys (PageUp, Home, ArrowUp) when the content overflows; Down keys (PageDown, End, ArrowDown, Space) or a pointer press when away from the end; a press on the scrollbar lane (the scroll node's trailing `max(16px, scrollbar width)`, timelineViewport.ts:67–97) when the content overflows; touch when away from the end; an upward scroll nobody explains while an iframe is hovered or focused; a focused HTML frame that changes height; any UI command that says so (minimap, citation, find).
- **What re-arms it**: reaching the end after having left it, or "Scroll to end". Navigating while still at the end stays released until the list leaves and comes back.
- **Scroll to end pill**: shown 150ms after the list leaves the end while released (debounced so thread switches do not flash it), hidden immediately at the end. Its pointer-down does not take focus; clicking scrolls to the end (smooth unless reduced motion), releases any anchor and re-arms follow. It sits above the composer at `bottom = ceil(overlay height - (top - contentTop)) + 4`, where `contentTop` is the top of the highest attached banner (or the composer surface) and `top` the highest one that overlaps the button horizontally, so a banner beside the button does not push it up (composerFooterLayout.ts:127–142). It hides while find is open (ChatView.tsx:3502–3508). Its label gains " · <child waiting label> · N queued" when those exist (ChatView.tsx:3105–3108).
- **New turn anchoring**: a sent message becomes the anchor (useTimelineFollow.ts:526–531). Only when it is the first user message in the list (a new thread's first turn) does the list add end space, sized for an anchor offset of 24 (`CHAT_TIMELINE_ANCHOR_OFFSET`, MessagesTimeline.tsx:895–911, packages/shared/src/chatList.ts:12–42), and scroll the message to the top with `viewOffset` 16 (`CHAT_LIST_ANCHOR_OFFSET`), animated unless reduced motion, retrying for up to 12 frames while the list is missing (useTimelineFollow.ts:374–428). The anchor releases, and the list scrolls to the end, as soon as the running turn has reasoning or a work entry with a tool tone, an item type, a request or a command (useTimelineFollow.ts:491–524, ChatView.logic.ts:257–279), and it clears when the anchored run settles (ChatView.tsx:1030–1047). Otherwise sending simply follows the end.
- **Growth below**: when the composer inset grows, or the footer grows (a queued turn, a status line), a following list scrolls to the end again (useTimelineFollow.ts:331–339, 463–467).
- **Page keys from the composer** (pageScrollController.ts): PageUp/PageDown in an editor that cannot scroll further scroll the timeline by `height - 36 - composer overlay height`, animated over 150ms with a cubic-bezier(0.42, 0, 0.58, 1) ease; holding the key for 150ms switches to continuous scrolling whose speed ramps from 1× to 2× of a page per 150ms over 400ms; reduced motion jumps. PageUp releases follow when the content overflows.
- **Keyboard on the timeline** (timelineViewport.ts:104–126): scroll keys typed with focus on the body or inside the timeline are heard app-wide (not in editable fields, not with Alt, Ctrl, Meta or Shift, not while composing text, not while a floating layer is open).

### 5.11 Pointer and keyboard on rows

- Turn folds, group headers, attempt folds and folded message headers are buttons (Enter/Space). Work entries are `div role=button tabindex=0`; Enter or Space toggles only when the row itself has focus, so text inside expanded details stays selectable and nested buttons work (`stopPropagation` on the details).
- Inline buttons inside rows ("Open chat", "Open subagent", thread-read links) stop the row toggle.
- Clicking a subagent card opens its thread (in a linked environment when connected); hovering shows its hover card after 200ms.
- Images: click or Enter/Space on a focused markdown image opens the dialog with every image of that message (gallery order); the dialog moves with ArrowLeft/ArrowRight (wrapping), pans a zoomed image with arrow keys, closes with Escape (an open context menu takes Escape first) or a click on the backdrop, and returns focus to the opener.
- Text selection pins the covered rows so virtualization never recycles them mid-selection.

### 5.12 Minimap

- Items are the user messages of the loaded rows, each with the turn's last assistant text (MessagesTimeline.logic.ts, timelineMinimapItems.ts:17–55). It needs at least 2 items.
- Strips are spaced 8px; the rail caps at `max(4rem, 100% - 14.75rem)` and then compresses. Strip i sits at `i/(n-1)` of the rail.
- `data-in-view` marks strips whose row is in the rendered visible range, refreshed once per frame on scroll.
- Hovering the strip picks the nearest item to the pointer (`round(progress × (n-1))`), scales neighbours (fisheye) and shows the preview: the user text on one line and the reply's plain text on up to 3 lines (markdown stripped, first 1500 characters parsed).
- Clicking jumps to the item (`scrollToIndex`, 24px from the top, smooth unless reduced motion) and releases follow. Previous/Next buttons (tooltips "Previous turn", "Next turn") sit 6px above and below the rail at opacity 0 and appear when they themselves are hovered or focused; Previous is disabled on the first turn, Next on the last turn or at the end of the list. Keyboard: focus the strip, ArrowUp/ArrowDown move the highlight, Home/End jump to the ends, Enter/Space go there. Wheel over the minimap scrolls the timeline.
- Visibility: persistent when the side gutter `(viewport - lane)/2 ≥ 48`, else hover-only (`[@media(pointer:fine)]:block`). The strip takes the pointer when the hit strip is wider than 0, the previous/next buttons only when it is at least 14px (MessagesTimeline.logic.ts:434–466). The gutter is measured when the viewport resizes or when a row's lane reports a width different from the last one seen (MessagesTimeline.tsx:1029–1078). Re-checked in verification: in a visible, focused tab, navigating from a new-thread draft into a thread gave `data-persistent-gutter=true` and a 40px hit strip, and so did a full reload. The hover-only state with a 0px strip appeared only after the page had loaded while the tab was hidden. The earlier finding (hover-only after draft navigation) came from that hidden-tab state: ResizeObserver does not deliver while the tab is hidden, and the lane-width guard then skips the re-measure. With a 0px hit strip inside a `pointer-events-none` container, the hover-only minimap cannot even be hovered, so the stale state stays invisible until the next resize.

### 5.13 Selection and citations

- Selecting text inside an assistant message (not in `user-select: none` parts, visible within its clipping parents) shows the "Cite" pill. It sits at the pointer when the pointer is within 8px of the selected text, else at the selection's last line end, 4px below, clamped 8px inside the window and inside the timeline (selectionActions.ts:6–27). Pointer-down on it keeps the selection; Tab moves focus into it (foreground threads only); Escape dismisses it. Selections over 8000 characters (`ASSISTANT_CITATION_MAX_TEXT_LENGTH`) show "Shorten selection", disabled. The resolved point is the pill's top-left corner, not its centre: a double-click at (748, 196) put the pill's box at exactly x 748, y 196 (M). Its aria-label is "Cite selection in composer", or "Selection is too long to cite" (AssistantSelectionToolbar.tsx:195–212). After a double or triple click the toolbar waits out the 500ms multi-click window before it shows, and after a scroll it comes back 150ms after the scrolling settles (`SELECTION_MULTI_CLICK_INTERVAL_MS`, `SELECTION_SCROLL_SETTLE_MS`, selectionActions.ts:3–4, 130–137, 158).
- Citing inserts a citation chip into the composer and clears the selection.
- A sent citation renders in the user's bubble as the quoted-response card: the passage highlighted (`mark`) between "…prefix" and "suffix…", clamped to 2 lines; clicking it scrolls the timeline to the cited response and flashes the passage (`::highlight(t3-assistant-citation)`, 01-tokens §4). Cards from "Reply to this response" carry a title line (12/16 500 at foreground 80%) above the preview; a comment written with the citation shows below the card (`mt-1.5`, pre-wrap). Focus-visible draws a 2px `--ring` outline (AssistantCitationChip.tsx:59–112).

### 5.14 Accessibility

- Every message has an sr-only `h3` ("You", "T3 Code", "Automation", the sender's title, "<name> (guest)"); markdown headings inside messages get `aria-level` shifted by 3 (capped at 6). The first and last messages stay mounted so heading navigation wraps.
- Work rows expose `aria-expanded` and a full label (with answer preview and failure). Groups are `role=region "Tool calls"`; flat groups `section "Activity"`. Compaction is `role=separator` with its full text. Approvals are `role=group`. Live failure announces "Tool call failed".
- `ThreadActivityAnnouncer` announces turn status changes (working, waiting on the user, idle); the screen reader setting picks the level (`status`, the default, `messages` or `everything`), and only the higher levels read timeline entries. It stays mounted when the timeline is empty (MessagesTimeline.tsx:1170–1186, packages/contracts/src/settings.ts:350, 530–532).

## 6. Motion

| Item | Property | Duration | Easing | Gate | Source | M/P/S |
|---|---|---|---|---|---|---|
| user actions, assistant meta | opacity 0 → 1 | 200ms | `cubic-bezier(0.4, 0, 0.2, 1)` | | TimelineUserMessageRow.tsx:1066; TimelineAssistantRows.tsx:185 | M |
| turn fold button | color | 150ms | same | | MessagesTimeline.tsx:1820 | M |
| work entry chevron | transform rotate 0 → 90° | 200ms | same | | TimelineWorkEntryRow.tsx:542 | M |
| folded message chevron | rotate 0 → 180° | 150ms | same | | TimelineUserMessageRow.tsx:402–407 | M |
| details chevron | rotate 0 → 90° | 150ms | same | | MarkdownDetails.tsx:44–51 | P |
| details panel | height 0 ↔ `--collapsible-panel-height` (Collapsible default) | 200ms | same | none under reduced motion | MarkdownDetails.tsx:54–61; 02-primitives §6 | S |
| subagent group chevron | rotate 0 → 180° | 150ms | same | | TimelineV2EventRow.tsx:135–142 | M |
| subagent group trigger | opacity .55 → 1 on hover | 150ms | same | idle groups | TimelineV2EventRow.tsx:96–103 | M (transition) |
| subagent group panel | none (`animate={false}`, so virtualized rows settle before anchoring resumes) | | | | TimelineV2EventRow.tsx:145–146 | S |
| queued turn reveal span | opacity 0 → 1 | 200ms | same | | QueuedRunsControl.tsx:515 | S |
| "Show all N queued" chevron | rotate 0 → 180° | 150ms | same | none under reduced motion | QueuedRunsControl.tsx:390–397 | S |
| chip label underline | text-decoration-color | 150ms | same | not under reduced motion | ContextChip.tsx:77 | M |
| minimap strips | scale | 150ms | same | | TimelineMinimap.tsx:200–206 | M |
| minimap in-view overlay | opacity | 150ms | same | | same | M |
| minimap (hover-only mode), nav buttons | opacity | 150ms | same | | TimelineMinimap.tsx:107–111, 284–290 | S |
| live tool shine | background-position -72px → 100% + 72px | 2.2s | `steps(30)` | infinite; visible, on screen, no reduced motion, no forced colours | index.css:551–578 | M |
| live activity focus | translateX 0 → 100% (counter -100%) | 2.2s | `steps(30)` | same | index.css:586–660 | M |
| working timer | text | every second | | live only | tickTextContent | M |
| streaming blocks | opacity 0 → 1 (`@starting-style`) | 600ms | ease-out | `[data-streaming]`, no reduced motion | index.css:2204–2220 | S |
| ThreadNow line enter | opacity 0 → 1 (`starting:opacity-0`) | 150ms | ease-out | not under reduced motion | ThreadNow.tsx:18–25 | S |
| follow scroll while working | LegendList animated scroll to end | library default | | working, no reduced motion | MessagesTimeline.tsx:1447–1449 | S |
| scroll to end | smooth scroll | browser smooth | | no reduced motion | useTimelineFollow.ts:318–320 | S |
| disclosure reveal | `scrollBy` smooth | browser smooth | | no reduced motion | MessagesTimeline.tsx:520–522 | S |
| page keys | scrollTop | 150ms per page, hold ramps 1× → 2× over 400ms | cubic-bezier(0.42, 0, 0.58, 1) | reduced motion jumps | pageScrollController.ts:1–7, 127–148 | S |
| minimap jump, anchor positioning | scrollToIndex animated | library default | | no reduced motion | MessagesTimeline.tsx:1477–1483 | S |
| fork busy | `animate-pulse` opacity | 2s | `cubic-bezier(0.4, 0, 0.6, 1)` | infinite while forking | TimelineAssistantRows.tsx:139 | S |
| buttons | press scale 0.97, box-shadow | 150ms | | | 02-primitives §4.1 | M |
| image dialog | popup `scale` 0.98 → 1 and opacity 0 → 1; backdrop opacity | 200ms | ease-in-out | | dialog-styles.ts:1–12 | S |
| scroll-to-end pill | appears after 150ms debounce, hides instantly | | | | useTimelineFollow.ts:65–68 | S |

## 7. Gaps

- **Loaded workspace images** (agent-viewed images, workspace images in markdown, user image chips in text) could not load with this pairing scope; only their failure state was measured. Remote images, collages and lone user images were measured.
- **Running tool row** (`work-live` with `live-tool-shine` on a real label) was not caught live; the thinking row and its shimmer were. The shine itself is measured on a probe in 01-tokens §3.12.
- **Pending approvals and pending questions** were not present on any thread. The approval row is from source; pending questions render in the composer, not the timeline.
- **Grouped consecutive user messages**, the floating actions beside them, guest messages, "Sent by T3 Code", file cards, the plan card, superseded attempts, footnotes, task lists, alerts, details, artifact template cards, h1/h4/h5/h6, math and mermaid were not on any thread; their values are probes (P) with the real class lists, or source.
- **PR link previews** and the right-click menus (native in the desktop app) were not opened. The subagent hover card was opened and measured in both schemes in the verification pass (§3.8).
- **"Edit from here"** never appeared (no rewindable checkpoints on the account's threads). Source only.
- **Dark error row** values are derived from tokens; the row unmounted before the dark pass.
- **Light only**: the background work dialog was measured in light; its dark values come from the tokens it uses (01-tokens §3.12). The minimap preview card and the shimmer band's foreground copy are now measured in both schemes.
- **Pierre file icon shapes** live in a separate SVG sprite inside @pierre/diffs and are not reproduced here. Their colours are now listed per token from source (§3.6), with five checked live; the mapping from file name to icon (`resolvePierreIconForEntry`, apps/web/src/pierre-icons.ts) was not transcribed.
- **Tooltip delays** on timeline triggers follow the primitive defaults (600ms); the timestamp tooltip did not open within the measured hover window.
- **macOS**: SF Pro changes text widths, so hug-width elements (fold buttons, pills, chips) and wrap points differ; the native thin scrollbar of code blocks draws in macOS's style.
- The `zen`, `hideSuccessful` and `hideAll` tool call visibility modes were not switched on (that changes a setting); their behaviour is from source.
- **Link and secret request cards** (pending and answered) were not on any thread; their structure is from source (§2.8).
- **Queued turns** were measured with one automation message in the queue. A plain queued message, the editing and saving states and the "Show all N queued" control are source only.
- **Snapshot attachments** were measured standalone above a bubble only; the in-bubble 208×112 frame and the accessibility data popover are source only.
- **Itemized failed runs** were seen on one failed thread (commands, reads, edits, reasoning). A failed run that only hit a usage limit, which still groups, was not seen.

## 8. Port notes for Go/Gio

**What this replaces in tern.** `app/ui/screens.go` `messages`, `itemGap`, `item`, `foldRow`, `toolIcon`, `toolRow`, `toolMeta`, `toolBody`, `thought`, `queue` (queued turns, §2.10) and the timeline half of `approval`; `app/ui/markdown.go` (`parseMarkdown`, `Markdown`, `CodeBlock`, `codeCard`, `runButton`, `copyButton`); `app/ui/model.go` `Item`, `ItemKind`, `ToolKind`, `Todo`; `app/core/scene.go` `items`, `foldRuns`, `toolItem`, `toolOutput`, `changes`, `duration`. tern's current `toolMeta` prints "exit 0" on every finished command row and "+N −M" on edit rows (core/scene.go:612–630). T3 puts neither on the row: a non-zero exit shows as "exit N" in destructive inside the expanded body, the counts appear in the expanded file change inspector and the changed files card, and no row shows a per-tool duration.

1. **Row model.** Replace `ui.Item` with the 16 row kinds of §2.2 and derive them in core with the rules of §5.2, in that order (visibility, folds, attempts, live tail, working, thinking, grouping, compaction, meta placement, user grouping, created-thread summaries, `continuesWorkLog`). Port `presentation.ts` (labels, summaries, failure detection, group actions) as a Go package with table tests taken from T3's own `presentation.test.ts` and `MessagesTimeline.logic.test.ts` cases.
2. **Data tern's core must add.** Per work entry: `tone`, `toolLifecycleStatus` (including `stopped`, `declined`, `idle`), `requestKind`, `toolTitle`, `toolData` (input and output), `toolSurface`, `toolIcon`, `toolSource` (integration name and icon), `viewedImagePath`, `changedFiles`, `rawCommand`, `questionAnswer`, `sourceActivityKind`, the run attempt and its status, `deferredWork` markers (fold contents loaded on demand). Per message: `inputIntent` (steer, promoted), `createdBy` and `creationSource` (attribution), `scheduledTaskId`, guest context, sender thread, context records (chips), attachments with download state and image dimensions, `updatedAt`, `streaming`. Per run: `startedAt`, `completedAt`, `status`, `turnEnded`; interrupt items; turn diff summaries per assistant message; checkpoints; subagent projections (status, progress, result, times, provider instance); command output streaming (`v2.commandOutput`) and item detail fetches for expanded rows.
3. **Virtualized list.** Gio's `layout.List` is fine; keep per-row measured heights (estimate 90, and the size buckets of MessagesTimeline.tsx:1581–1587 for long messages). Keep the first and last message rows and selected rows laid out. Implement follow with the state machine of `transitionTimelineFollow` and the release rules of §5.10 exactly; most "the timeline jumped" bugs come from getting those wrong.
4. **Bottom space.** End the list with a spacer of `ceil(composer overlay height)` plus 16, and re-scroll to the end when that spacer or the footer grows while following.
5. **Hidden timestamps.** A row's timestamp takes no space until the row is hovered or holds focus; then it is laid out at the row end and the label shrinks. Model it as a trailing widget whose width is 0 when hidden.
6. **Work-log row geometry.** 24px line, 24×24 icon slot with a 16px icon, 6px gap, label 14/22.75 truncating, rows 28 standalone (2px top and bottom) and 24 inside expanded lists, block padding 8 / 4 / 0. Icons draw at stroke 2 here (not 1.75). Paint the icon as `--icon-muted` at 70% after multiplying RGB by 0.6 in light (painted `#7a7a7f`), 70% in dark (`#717171`).
7. **Fold label and duration.** `formatDuration` and the working timer are pure functions; port them with T3's tests (`orchestrationTiming.test.ts`). The working timer should repaint once a second only while live.
8. **Markdown.** tern's own `parseMarkdown` will not match GFM edge cases. Use a CommonMark + GFM parser (goldmark with its GFM extensions) and implement the overrides of §2.5: file-path links and backticked paths become chips, thread links become chips, `> [!NOTE]` alerts, task items, footnotes, `<details>`. Margins are 10.4 between blocks, headings 20 / 8, list gutter 20, `li` gap 4; collapse margins only between siblings as CSS does (a first child has no top margin, a last child no bottom margin).
9. **Text colours with alpha.** Assistant body text is foreground at 90%: paint `#18181be6` / `#f5f5f5e6` over the canvas rather than a pre-mixed grey, so text on code frames and tables blends the same way. Headings and user text are full foreground.
10. **Inline code** needs a rounded background behind a run of text, cloned per line when it wraps (`box-decoration-break: clone`): pad 0.1em / 0.35em, radius 4, at 0.85 of the surrounding size.
11. **Syntax highlighting.** Port the palettes of §3.6 as two `map[scope]Color` tables. Chroma is the practical Go highlighter; map its token types onto these scopes (keyword, string, number, comment, function, type, variable, property, punctuation, operator, tag, attribute, constant). Exact token boundaries will differ from T3's WASM lexers; colours and font (mono 13 / 17.875) must not.
12. **Code block frame.** 30px header with the language word or icon at 11/16 mono in foreground 72%, 24×24 buttons with 2px gaps and 6px right inset, body padding 12.8 / 14.4, frame radius 8 with a 1px `#e4e4e7b3` border in light and no border on `#ffffff09` in dark. Wrap on by default (setting `wordWrap`).
13. **Tables.** The table always fills the text column. Lay out columns like CSS auto table layout: each column's minimum is its longest word, and the spare width goes out in proportion to each column's natural width. Do not cap cells at 384: T3 declares that cap but Chrome ignores `max-width` on cells, so a prose column can take most of the row. Outer cells sit flush with the text, rules are 1px at border/60, and words longer than a cell break. Scroll sideways only when the minimum widths overflow, in a container that then always shows its scrollbar. Put the copy button below the table at the lane's right edge.
14. **Live shimmer.** Draw `live-tool-shine` as the label in secondary-label with a 72px foreground band moving in 30 steps per 2.2s; the thinking row's band is a 72px soft mask (stops at 0, 10.8, 25.2, 36, 46.8, 61.2, 72px with alpha 0, .12, .55, 1, .55, .12, 0) moving across a foreground copy of the row. Stop both when off screen, when the window is hidden, or under reduced motion (01-tokens §5.8).
15. **Glass pills.** The scroll-to-end and cite pills are `Button xs glass`: paint `#fcfcfccc` / `#000000cc` with a 1px border at border/60 and `shadow-sm`; add blur only if affordable (01-tokens port note 10).
16. **Minimap.** A 72px overlay at the canvas's left edge; implement the fisheye scales (1, 0.667, 0.417, 0.333), 8px spacing, rail height cap, persistent-vs-hover rule (gutter ≥ 48), the preview card (dropdown glass, 320 wide) and keyboard navigation. Gio lays out every frame, so derive the gutter from the current frame's constraints instead of caching it; that sidesteps the stale state T3 shows after loading in a hidden tab (§5.12).
17. **Images.** Lone user image: 192px tall, width from the aspect ratio, min 96, max 80% of the lane, radius 16, a 1px inset ring `#00000017` / `#ffffff1f` drawn above the image. Collage: 320 wide, 1px gaps showing the same colour. Markdown images: max 480×480, framed workspace images radius 8 with a border at border/40. The image dialog: black 75% scrim, image at most 92vw × (0.86 × window height) with a ring and the 2xl shadow, 28px close button, 32px side arrow buttons with 20px chevrons (moved to the bottom corners below 640px), white caption at 80%.
18. **Copy feedback.** Message copy shows a small "Copied!" toast anchored to the button for 1s and swaps the icon to a primary check; code and table copy only swap the icon for 1.2s.
19. **Expanded command rows** need the command highlighted per shell word (each word an unbreakable unit that can still wrap anywhere when longer than the line) and a 320px output panel that follows its end within 8px and colours ANSI sequences with T3's terminal palettes.
20. **Dark mode** is the `:root` palette throughout; the sidebar's scoped palette never applies here. hr rules switch to muted foreground in dark (`#a1a1a1`), unlike borders.
21. **Disclosure memory.** Keep expanded folds, groups, entries, list scroll positions and the timeline position per thread in memory (100 threads, least recently used dropped), so switching threads and back restores the view exactly.
22. **Accessibility in Gio.** Give message rows a semantic heading label ("You", "T3 Code"), work rows a button role with their full label and expanded state, and announce turn status changes.
23. **Queued turns.** tern draws the queue as a numbered panel of first lines (`ui/screens.go` `queue`, `core/scene.go` filling `Scene.Queue []string`). T3 draws each queued message as a dashed, unfilled bubble on the right with its full text and its sent-form attribution, an actions row with "Send now", edit and remove, and a hover-revealed timestamp (§2.10, §3.12), and folds the list to 2 rows plus "Show all N queued" past 3. `Scene.Queue` needs, per queued run: run id, message id, full text with context labels resolved, attribution (agent, automation, guest, T3 Code), scheduled task id and title, sender thread, created time, image thumbnails, `editable`, `pending` (optimistic, not yet on the server), whether it can be sent now (fresh-context rules), and which one the composer is editing.
24. **Failed runs.** Do not group the work entries of a failed run; give each its own row with its own label (§5.2 step 7), unless the run only hit a usage limit.
25. **Pierre icons.** Ship the sprite's 16-unit paths as Gio vector icons and colour them from the token table in §3.6. tern has no Pierre icons yet; the lucide `file` / `folder` fallback in `text-icon-muted` covers unknown names.
26. **Link favicons.** T3 loads `https://www.google.com/s2/favicons?domain=<host>&sz=32` for public hosts and remembers failures for the session. A native client that does not want that request can draw `globe` for every link, which is exactly what T3 shows for private hosts and failures.
27. **Snapshots.** A window snapshot attachment is a lone image drawn whole (`object-contain`, at most 192 tall) with the bottom gradient, the app icon (or its initial), the app name and window title of §2.3; core must carry `source.kind`, `appName`, `appIconDataUrl`, `windowTitle` and the accessibility data for the popover.

## Verification

An adversarial pass on 2026-11-18 against the same build (tag 4606) and the live UI at 1440×900 @2x, sidebar expanded, details card inline. The tab was kept visible and focused with CDP focus emulation for the whole pass, after a first attempt showed the hidden-tab effects described in §1. Nothing was sent, edited, approved, stopped or removed; disclosures, the image dialog and a text selection were opened and closed again, and one subagent card was clicked to navigate into its child thread.

### What was checked

**Re-measured live in both schemes (34 items), all matching the doc unless listed under "What was wrong":** assistant paragraph type and colour; h2 type, colour and margins; quote rule colour and padding; inline code size, padding, radius, fill and ink; code frame fill, border, radius and margins; code header height, padding and colour; `pre` type, colour, padding and wrapping; wrap toggle pressed and copy button colours, sizes and spacing; `ul`/`ol` gutters, `li` spacing and marker colour; link colour; file chip colour, weight, gap and underline; table cell padding, rules and width; table copy button; `hr` colours and gaps; the compaction divider; the scroll-to-end pill (placement, size, glass fill, blur, border, shadow, icon); minimap strips, hover fisheye, in-view overlay and preview card; turn fold row, button and chevron; the user bubble; the user actions row; the assistant meta row; timestamps; work-log icons (colour, opacity, light filter, stroke) and labels; the thinking row and its focus band; the subagent group trigger and avatars; queued turns; the image dialog (backdrop, close button, image frame, caption); the snapshot overlay; the cite pill; the working row; the ThreadNow line; the subagent hover card; the lone-image frame and ring; the quoted response card and its highlighted passage.

**Live in one scheme:** h3 and the heading gaps (light); turn fold hover (light); history control, long-message collapse, the "Subagent of" divider, the subagent card inside an open group, the open and idle group states, the changed files card, the expanded command body and the expanded tool group list (dark); a probe table for the cell-width rule.

**Re-derived from source (31):** `formatDuration`, the working timer, `AgentElapsed`, `formatDayAwareTimestamp` and its tooltip, `formatTokens`, `formatBytes`, `formatCompactDiffCount`, per-kind row padding, `WorkLogBlock`/`WorkLogRow`/`WorkLogLine`/`WorkLogDetails`, `MessageCopyButton` and the anchored toast, code block copy, wrap and run rules, every `.chat-markdown` rule (index.css:1866–2289), GitHub alert presentations, the artifact template card, `useTallEnoughToFold`, the collapse mask, `pageScrollController` constants, `selectionActions`, minimap constants, the 40px follow threshold, the 150ms pill debounce, `live-tool-shine` and `live-activity-focus`, `ProposedPlanCard`, `MarkdownDetails`, `MarkdownTable`, the subagent status summary, `PierreEntryIcon` colours, failed-run itemization, the favicon source, `QueuedRunsControl`, and tern's `toolItem` meta strings.

**Privacy:** the doc was searched for host, machine, project, branch and thread names, paths, links, emails and quoted message text. None were found; every quoted string is a UI label from T3's source or a placeholder. This pass added no thread content either: new examples are placeholders or UI labels, and its screenshots stay outside the repo.

### What was wrong, and the fix

| Where | The doc said | Measured or source | Fixed in |
|---|---|---|---|
| heading spacing | 8 below a heading | margins collapse, so heading → paragraph is 10.4 (measured 10.39 after an h2 and an h3); 8 shows only before a block with no top margin | §3.1, §3.5 |
| table cells | max cell width 384 (24rem), port note: cap cells at 384 | T3 declares the cap but Chrome ignores `max-width` on cells: a measured header cell was 571 wide and a probe prose column 602 of 728. The table is `width: 100%`, which the doc never said | §3.5, §5.5, port note 13 |
| minimap finding | draft → thread navigation leaves the minimap hover-only because the gutter is not re-measured | not reproducible in a visible tab (draft → thread and reload both gave persistent mode and a 40px strip); the stuck state appears only after loading in a hidden tab | §1 method notes, §5.12, port note 16 |
| media sent with words | only the bubble's top-right corner tightens | the media's bottom-right corner tightens to 6 as well (measured 16/16/6/16 over a bubble at 16/6/16/16, 4 apart) | §3.3 |
| work grouping | consecutive work entries group into a summary row | entries of a failed run are itemized, one row each with its own label, unless the failures were only usage limits (seen live: 20 rows on a failed run) | §2.2, §5.2 step 7 |
| `formatDuration` | under 10s "one decimal" | a whole value drops ".0" ("3s"), the unit is chosen after rounding (59.6s → "1m 0s"), zero minutes are skipped ("1h 5s") | §5.4 |
| file chip icon colour | "Python `#1a85d4` / `#69b1ff`" | the chip measured was TypeScript; the colour belongs to eight tokens. Replaced with the measured tokens and the full table | §3.5, §3.6 |
| evidence labels | subagent group, divider pill with an action, minimap preview (dark), focus band copy (dark) marked P or light only | measured live in this pass | §3.8, §3.9, §3.10, §7 |

### What was missing, now added

- Queued turns in the footer: structure, states, measurements in both schemes, motion, data and a port note (§2.10, §3.12, §4, §5.1, §5.5, §6, port note 23). tern's numbered queue panel does not match.
- Window snapshot attachments: frames, overlay, contents button and fallback (§2.3, §3.3, §3.11, port note 27).
- Pierre file icon colours for every token in both schemes (§3.6), and the lucide fallback.
- Code blocks: no line numbers, focusable `pre`, `tab-size: 4`, default language "text", `gitignore` → `ini`, fence title parsing, plain-text fallback when highlighting fails (§2.6).
- Link favicons: the Google s2 source, private hosts on `globe`, failure memory (§3.5, port note 26).
- Subagent group details: aria labels, chevron rotation, no panel animation, a lone subagent rendered as a card, the status summary format and the display title rule (§2.8, §3.8, §5.4, §6).
- The subagent hover card, measured (§2.8, §3.8).
- Link and secret request cards, thread chips and the unresolved chip (§2.8).
- The list header of subagent threads and side chats (§2.1).
- Plan card collapse threshold, preview and title rules (§3.9).
- History control hover, focus and disabled states (§4).
- Cite pill placement (top-left at the pointer), aria labels and its multi-click and scroll delays (§5.13).
- The `<details>` panel's 200ms height animation (§6).
- The rule that the footer content is empty unless one of its five parts exists (§2.1).
- Failed-run itemization as a port note (24).

### Still open

The remaining gaps are in §7. The ones that matter most for the port: the running tool row with a live `live-tool-shine` label (only "Thinking" ran during this pass), pending approvals and questions, loaded workspace images, the in-bubble snapshot frame, the Pierre icon shapes and the name-to-icon mapping, and macOS text metrics.
