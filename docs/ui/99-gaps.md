# 99 · What the specs do not cover yet

A completeness audit of `docs/ui/` against T3 Code's web client at tag `v0.1.10-nightly.20261112.4606`, made on 2026-11-19. It lists what a user of the main window can see or do that no doc in this folder specifies. Surfaces are grouped and ranked by how often an everyday user meets them, most visible first.

Each entry gives what the user sees or does, where it is reached from, the source files, and the doc that should own it. Paths are relative to `apps/web/src` unless they start with `packages/` or `apps/`. `c/` is `apps/web/src/components/`. Line numbers are 1-based in the pinned checkout.

Status tags:

- **Absent**: no doc describes it.
- **Named**: a doc names it (in an element tree, a source table or its own Gaps) but specifies none of its structure, values or behaviour.
- **Partial**: a doc covers one stated aspect, and the rest is open.

## 1. Scope and sources

- **Docs read.** 01-tokens, 02-primitives, 03-shell, 04-sidebar, 05-header, 06-timeline, 07-composer, 08-pickers, 09-terminal, 11-empty-auth, 12-keys-palette, 13-diff, tokens.json and tokens.extra.json. 12-keys-palette was written while this audit ran, and was read at its 01:31 and 01:56 revisions. Each doc's §7 Gaps was read. Items a doc already lists as "not measured live" are not repeated here, unless the surface they belong to has no spec at all.
- **Planned but missing.** `10-browser.md` is in the spec workflow's topic list but was never written. Its whole scope is §3.5. The other empty slot, `12-keys-palette.md`, landed during the audit. It closes the command palette and the keybinding map, so this list keeps only what 12 hands on to other docs.
- **Method.**
  1. Every non-test source file under `apps/web/src` (1,416 files) was matched by name against all docs. 505 are named somewhere and 911 never are.
  2. Every never-named file was opened and classed as visible UI, behaviour, or plumbing.
  3. Each visible feature was then searched in the docs by its strings, component names and behaviour, because a doc can describe a feature without naming its file.
  4. Only features still unspecified after step 3 are listed.

  The routes (`routes/*.tsx`), the root hosts (`routes/__root.tsx:176–229`), the right panel's surface switch (`c/chat/RightPanelSurfaceContent.tsx:86–240`) and the default keybindings (`packages/shared/src/keybindings.ts:57–151`) were walked to find every page, modal, panel surface and global dialog.
- **Live checks.** Same build, 1440×900 @2x, light scheme, read-only pairing. Screenshots are kept outside the repo. The command palette was opened, which confirms 12. The Settings modal was opened on General, the Usage modal on Limits, and the Pull requests page in its loading state. Each was closed with Escape or Back, and nothing was changed. No values were measured for this list. It is an inventory, and every entry still needs its own extract and verify pass.

## 2. Summary

| § | Surface | Reached from | Source lines (non-test) | Status | Proposed owner |
|---|---|---|---|---|---|
| 3.1 | Settings modal, 24 pages | sidebar footer gear; palette "Open settings" and "Project settings"; details card "Manage scheduled tasks"; desktop menu `CmdOrCtrl+,`; `/settings/*` links | 57,948 (`c/settings` + 28 route files) | Absent (fragments in 02, 11, 12) | new 14-settings.md, split by nav group |
| 3.2 | Files surface: explorer, file viewer, format viewers, Go to file, Search project contents | file links and chips in messages; composer chips; right panel add menu "Files" (F); mod+P; mod+Shift+F | 17,235 (`c/files`, office viewers included) + 518 (`c/search`) | Absent (13 has the shared tree pane; 12 has the palette frame around the two searches) | new 15-files.md |
| 3.3 | Thread find bar and its highlights | mod+F; header menu "Find in thread"; palette "Find in current thread" | 1,464 | Partial (05: box and card offset; 12: the chord) | 06-timeline, or new 16-find.md |
| 3.4 | Pull requests: page, PR detail surface, thread PR list, link and checkout dialogs, PR link hover card | sidebar footer; palette "Open pull requests"; details card PR rows; right panel P and L; PR links in messages | 23,838 (`c/pullRequest`) | Absent (05: details card rows and the checks popover; 04: sidebar badges; 12: page chords) | new 17-pull-requests.md |
| 3.5 | Browser surface, floating preview, device toolbar, server browser | right panel add menu "Browser" (B); mod+Shift+J; links; terminal links; script preview URLs; agent-opened previews | 4,899 (`c/preview`) + 5,321 (`browser/`) | Absent; 10-browser.md planned, never written | 10-browser.md |
| 3.6 | Usage modal | sidebar footer; mod+U; palette "Open usage" | 4,386 (`c/usage`) | Absent (12: its chords) | new 18-usage.md |
| 3.7 | Chat notices: banner lane, the composer banners 07 leaves undetailed, the file drop overlay, pending guest messages | thread and connection states | about 1,700 | Named (03, 06, 07) | 07-composer and 06-timeline |
| 3.8 | Timeline rows and media named but not specified | thread content | about 4,200 | Named (06) | 06-timeline |
| 3.9 | Composer parts named but not specified | composer | about 1,200 | Named (07) | 07-composer |
| 3.10 | Side chat pane | composer `/side` and `/btw`; add menu "Side chat" (S); details card side chat rows | 340 + the embedded chat view | Named (05, 06, 07 cover only the entry points) | 06-timeline or 05-header |
| 3.11 | Dialogs reached from covered surfaces | header menu, palette, details card, composer, root hosts | about 5,900 | Absent or Named | the doc that owns each entry point |
| 3.12 | Device surface | right panel add menu "Device" (M) | 4,309 (`c/device`) | Absent | 10-browser.md, or new 19-device.md |
| 3.13 | Toasts outside the catalogue | every surface | 692 call sites in 164 files | Partial (11 §5.6 lists 11 sources) | 11-empty-auth §5.6 and each surface doc |
| 3.14 | Primitives with no spec | the surfaces above | 935 | Absent | 02-primitives |
| 3.15 | Interface language | Settings › General › Language | 130 + 9 locale files | Absent | 14-settings.md and 01-tokens §5 |
| 3.16 | Desktop-only main-window pieces | desktop app | about 1,900 | Absent or Partial | 03-shell |
| 3.17 | T3 Connect account pages | the Settings nav avatar, signed in | 968 | Named (11) | 11-empty-auth |
| 3.18 | Prompt-link pages | `/new?prompt=…` links; `?prompt=` on a thread link | 669 | Absent | 11-empty-auth |

## 3. Gaps by surface

### 3.1 Settings modal (Absent)

**What the user sees** (confirmed live on General). A large modal over the dimmed, blurred app. The popup is `size-full` up to `max-w-[80rem] max-h-[60rem]` with `rounded-2xl`, a border and `shadow-2xl/40`. It enters and leaves over 150 ms with opacity and a 0.985 scale. Escape, the X button (tooltip "Close (Esc)") or a press on the backdrop returns to the page that opened it (`c/AppRouteModal.tsx:16–103`). Below 640 px it fills the window.

- **Left column:** the settings nav (`c/settings/SettingsSidebarNav.tsx:122–167`, 285–430) and its "Search" field with a `/` key hint. The page groups are:
  - Preferences (heading hidden): General, Appearance, Chat, Diffs & pull requests, Keybindings, Voice, SnapShots.
  - Agents: Providers, Threads, Browser, Devices, Scheduled tasks.
  - Code: Projects, Source control, Worktrees.
  - Connections: Environments, Sharing & access, Secrets (feature-gated).
  - More (heading hidden): Maintenance, Labs, About, Archive.

  Labels come from `c/settings/settingsSearch.ts:95–119`.
- **Right column:** a 48 px header with the breadcrumb "Settings / <page>", "Restore device defaults" (General only) and the close button (`routes/_chat.settings.tsx:30–43`, 109–144). Under it is the page.
- **Page anatomy:**
  - section headings over cards of rows: title, description, control (Switch, Select, button);
  - per-row reset buttons and policy tooltips for managed settings (`c/settings/settingsLayout.tsx:191–305`, 550–584);
  - "Show where this value comes from" inheritance popovers (`SettingInheritance.tsx`, 472 lines);
  - scope pickers for environment, project and checkout (`SettingsScopeSentence.tsx:141–320`);
  - scope notices, for example "<setting> is not available for the selected target. Choose its owning scope to continue." (`routes/_chat.settings.tsx:45–107`, `SettingsScopeNotice.tsx`).

**What exists elsewhere.**

- 03-shell cites only the modal's drag strip (`AppRouteModal.tsx:72–77`).
- 02-primitives measured Switch and Select inside Settings.
- 11-empty-auth covers the nav's T3 Connect sign-in button and avatar.
- 12-keys-palette covers the Keybindings page's key pills only.

Nothing specifies the modal frame, the nav and its search (results, scrolling to and highlighting the target row: `settingsLayout.tsx:156–186`, 630–638), the header, scope selection, the row anatomy, or any page.

**Pages** (route → panel; the sections are the `SettingsSection` titles in source):

| Page | Sections and notable content | Sources |
|---|---|---|
| General | Notifications (thread notifications select, in-app notifications); Sidebar (project grouping, project order, Working section); Confirmations (unpin, archive, delete, quit shortcut: direct, hold or press twice); Display (language, timestamp format) | `SettingsPanels.tsx:774–1135` (sections at 789, 809, 918, 1041), `NotificationSettings.tsx` |
| Appearance | Colors & themes (theme gallery and search, preview circles, wireframe, theme import from VS Code / Open VSX, custom themes); Interface (typography, UI and code fonts with previews, panel animations preview) | `AppearanceSettingsPanel.tsx` (486), `ThemeSettings.tsx` (982), `ThemeSearchSection.tsx` (491), `ThemePreviewCircles.tsx`, `ThemeWireframe.tsx`, `ThemeImportDialog.tsx` (543), `TypographySettings.tsx` (543), `FontFamilyPicker.tsx`, `SettingsFontPreviews.tsx`, `PanelAnimationsPreview.tsx`, `openVsxThemes.ts`, `vscodeThemeImport.ts` |
| Chat | Composer; Follow-ups; Activity; Screen reader | `SettingsPanels.tsx:1136–1496` |
| Diffs & pull requests | Diffs; Pull requests (routing of PR links) | `SettingsPanels.tsx:1497–1640`, `PullRequestRoutingSettings.tsx` |
| Keybindings | the command list, recording a chord, conflicts, when-clauses, reset (12 has the key pills only) | `KeybindingsSettings.tsx` (1546), `KeybindingsSettings.logic.ts` (471), `CaptureShortcutConfig.tsx` (342) |
| Voice | Input (microphone, its order and test, dictation shortcut); Transcription (engine, language, translate to English, silence filtering, remove filler words); Dictionary; Cleanup (model, instructions, correction cue); model downloads and hardware | `VoiceSettingsPanel.tsx` (1697), `MicrophoneTest.tsx`, `MicrophoneOrderList.tsx`, `SpeechLanguageSelect.tsx`, `SpeechModelComparison.tsx`, `CloudVoiceConfiguration.tsx` (472) |
| SnapShots | SnapShots, setup dialog, shortcut recorder | `SnapShotSettings.tsx` (585), `SnapShotSetupDialog.tsx` (473), `useSnapShotShortcutRecorder.tsx` |
| Providers | provider instance cards (Setup, Runtime, Environment, Models), authentication, custom models, add-instance wizard, Codex setup and accounts, ACP registry, load balancing, Advanced | `ProviderSettingsPanel.tsx` (1448), `ProviderInstanceCard.tsx` (1425), `ProviderModelsSection.tsx`, `CustomModelEditor.tsx`, `ProviderSettingsForm.tsx`, `ProviderSetupSection.tsx`, `ProviderAuthenticationSection.tsx`, `CodexSetupSection.tsx` (1215), `AddProviderInstanceDialog.tsx` (768), `AddCodexAccountDialog.tsx`, `AcpRegistrySearchStep.tsx`, `AcpSessionManagementSection.tsx`, `LoadBalancingSettings.tsx`, `ProviderAccentColorPicker.tsx` |
| Threads | New projects; Snooze (presets editor, used by 04's snooze menu); Usage limits (sources); Auto-settle; Text generation (generators, title instructions) | `SettingsPanels.tsx:1641–2126`, `SnoozePresetsSettings.tsx` (470), `UsageProviderSettings.tsx`, `AddUsageLimitSourceDialog.tsx`, `TextGeneratorsSettings.tsx` (751), `ThreadTitleInstructionsSettings.tsx` |
| Browser | Agent access; Browsing (profiles, link target, auto-show the floating preview, search engine; Advanced: default viewport, zoom, appearance); Recordings (frame rate, input). Desktop-only rows sit in a group marked "Only available in the desktop app."; browser import wizard | `IntegrationsSettings.tsx:1581–1620` (whole file 1628), `BrowserImportWizard.tsx` (583) |
| Devices | Devices (integration, `IntegrationsSettings.tsx:820`); device sources and host editors (local, EAS, pooled) | `IntegrationsSettings.tsx:1622–1628`, `DeviceSourcesSettings.tsx` (815), `DeviceHostEditor.tsx`, `EasDeviceHostEditor.tsx`, `SimlockDeviceHostEditor.tsx` |
| Scheduled tasks | the task list and editor, schedules and intervals, worktree base branch | `ScheduledTasksSettings.tsx` (2145), `scheduledTasksSettings.logic.ts`, `IntervalSecondsField.tsx`, `c/WorktreeBaseBranchPicker.tsx` |
| Projects | per project: Checkouts; Project (name, icon, pin, archive); defaults (model, workspace, automatic pull); actions (`t3.json`); Danger | `ProjectsSettings.tsx`, `ProjectSettingsPanel.tsx` (715), `ProjectDefaultsSettings.tsx` (580), `ProjectActionsSettings.tsx`, `ProjectIconPickerDialog.tsx`, `ProjectFaviconPickerDialog.tsx` |
| Source control | Server environment; Version Control; accounts, app sign-in, hosts, tokens; Commits & pull requests (writing); branch naming | `SourceControlSettings.tsx` (574), `SourceControlAccountSettings.tsx`, `SourceControlAppSignInSettings.tsx`, `SourceControlHostSettings.tsx`, `SourceControlTokenSettings.tsx`, `SourceControlWritingSettings.tsx` (618), `BranchNamingSettings.tsx` |
| Worktrees | Creation; Branches; cleanup rules | `WorktreesSettings.tsx`, `StorageSettings.tsx` (892) |
| Environments | This app; Environments (saved, linked, T3 Connect rows, add environment dialog, routes) | `ConnectionsSettings.tsx` (1656), `SavedEnvironmentRows.tsx` (805), `LinkedEnvironmentsSettings.tsx`, `AddEnvironmentDialog.tsx` (669), `EnvironmentRoutesList.tsx`, `LocalEnvironmentSetting.tsx` |
| Sharing & access | per-environment pairing links (create dialog with scopes, copy, show link or code, QR), authorized clients with their scopes and "Revoke others", reachable addresses | `ConnectionsSettings.tsx:155–230`, `AuthorizedClientsSettings.tsx` (1008), `MobileAppSettingsRow.tsx` |
| Secrets | secret providers and sources | `SecretsSettings.tsx`, `SecretProviderDialog.tsx`, `SecretSourceInput.tsx` |
| Maintenance | Updates (update track switch dialog); Background activity; Diagnostics (entry to the Diagnostics page); Cleanup and Database | `SettingsPanels.tsx:2127–2236`, `UpdateTrackSwitchDialog.tsx`, `BackgroundActivitySettings.tsx`, `StorageSettings.tsx`, `DatabaseStorageNotice.tsx` |
| Diagnostics (not in the nav) | Live Processes, Resource History, Trace Diagnostics, Latest Failures, Most Common Failures, Slowest Spans, Span Logs, Top Span Names; OpenTelemetry export | `DiagnosticsSettings.tsx` (1331), `ResourceTelemetryDiagnostics.tsx` (1295), `TelemetryExportSettings.tsx` |
| Labs | Beta: boring sidebar (with its multiline option; 04 §2.10 covers the sidebar it produces), guest links (with sign-in when T3 Connect applies), linked environments, prompt suggestions, pooled device hosts, fast worktree checkout | `LabsSettingsPanel.tsx:20–150` |
| About | App (version, update track); Legal (link to Open source licenses) | `SettingsPanels.tsx:2237–2323`, `AboutVersionTitle.tsx`; `OpenSourceLicenses.tsx` (Third-party notices page) |
| Archive | archived threads, unarchive and delete | `ArchivedThreadsPanel.tsx` (275) |

**Behaviour to specify.**

- `/settings` redirects to General. Renamed paths (`beta`, `integrations`, `storage`) and moved anchors redirect (`routes/_chat.settings.tsx:174–183`, `settingsSearch.ts:1403–1412`).
- The scope is kept in the URL (`project`, `machine`, `checkout`).
- Providers and Secrets are single-environment pages (`routes/_chat.settings.tsx:146–172`).
- Closing the modal blurs the focused field first, so commit-on-blur rows save (`AppRouteModal.tsx:34–41`).
- Settings render over the thread that opened them, so the chat view never remounts (`routes/_chat.tsx:25–43`).

**Data.** `packages/contracts/src/settings.ts` (2,090 lines), server config, provider instances (`providerInstance.ts`), scheduled tasks (`scheduledTask.ts`), source control (`sourceControl.ts`), devices (`device.ts`), relay and remote access (`relay.ts`, `remoteAccess.ts`), secrets (`secretReference.ts`), keybinding upsert and remove (12 §1).

### 3.2 Files surface, Go to file and Search project contents (Absent)

**Reached from:**

- clicking a file link or chip in a message (`c/ChatMarkdown.tsx:835–840`, 925; `c/chat/UserMessageContextChips.tsx:379`) or in the composer;
- the right panel's add menu "Files" (F);
- palette "Go to file" (mod+P) and "Search project contents" (mod+Shift+F);
- "Open in file viewer" in the media context menu (§3.8).

The right panel renders kinds `files` and `file` through `c/files/FilePreviewPanel.tsx` (`c/chat/RightPanelSurfaceContent.tsx:203–240`). 05-header specifies the tab (title, Pierre icon, "Copy path" menu item), 13-diff §2.7 the shared tree pane (`FileExplorerPane.tsx`), and 12-keys-palette §2.4 and §5.19 the palette frame around the two searches. 12 states that "the searches themselves are specified with the files surfaces", and no such doc exists.

- **Explorer** (`FileBrowserPanel.tsx`, 621 lines):
  - "Search files" field (104–105);
  - "Refresh files" with "Refreshing…" (73–80);
  - "Expand all folders" / "Collapse all folders" (558–574);
  - a loading row (307);
  - a context menu on entries, "Copy mention" and "Add to chat", with toasts (215–259);
  - dragging an entry into the composer as a mention (`fileTreeDragMention.ts`);
  - tree expansion memory (`fileTreeExpansion.ts`).
- **File viewer** (`FilePreviewPanel.tsx`, 1,804 lines):
  - breadcrumbs with folder popovers, "Browse <folder>" and "Retry loading folder" (`FileBreadcrumbs.tsx:135–302`);
  - toolbar: word wrap, "Open file in preview browser", "Show/Hide file explorer", the rendered/source toggle ("Show markdown source", "Show rendered diagram", "Show table", "Show rendered page") (1018–1021, 1408–1422);
  - a "Full file" bar with copy and "Save file" (1748–1762);
  - read errors with their messages (`filePreviewMode.ts:9–26`);
  - the read-only source view with line comments (`ReadOnlySourcePreview.tsx` through Pierre `File`; `fileCommentAnnotations.ts`);
  - line reveal from links with a line (`fileLineReveal.ts`).
- **Format viewers:**
  - images (`PreviewImageContent.tsx`) and video;
  - audio (`AudioPreview.tsx`, `InlineAudioPreview.tsx`);
  - PDF: pages, zoom, fit width, "Find in PDF" (`PdfViewer.tsx:126–429`);
  - fonts (`FontViewer.tsx`);
  - notebooks (`NotebookPreview.tsx`);
  - Mermaid (`MermaidViewer.tsx`), markdown (`FileMarkdownPreview.tsx`) and HTML (`BrowserDocumentFrame.tsx`);
  - CSV and TSV tables (`VirtualTablePreview.tsx`, `DelimitedTablePreview.tsx`);
  - large text (`LargeTextViewer.tsx`);
  - zip and tar archives and SQLite tables with paging (`BinaryViewer.tsx:198–353`);
  - Word, Excel and PowerPoint files and ODF documents (`OfficeViewer.tsx`, `OpenXmlOfficeViewer.tsx`, `office/*` with its own paint code, `extendedDocumentPreview.ts`).
- **Attachment preview** (`AttachmentFilePreview.tsx`, 491 lines): the same viewers for a draft or sent attachment, with "No preview for this file", "Copy preview", "Save file" and "Remove from draft" (366–471). It is used by the file surface for attachments, the composer's file preview dialog (`ComposerAttachmentStrip.tsx`), and the file chips of sent messages and composer contexts (`UserMessageContextChips.tsx`, `composerContextPresentation.tsx`).
- **Go to file** (`ProjectFilePicker.tsx`): file rows with Pierre icons, highlighted matches and the footer action "Open file".
- **Search project contents** (`c/search/ProjectContentSearchDialog.tsx`, 330 lines): toggles "Match case", "Match whole word" and "Use regular expression" (174–190), placeholder "Search in <project>", results per file with line snippets (`HighlightedSearchLine.tsx`), and "This connection cannot search host files." without file scope (12 §5.19).
- **Data:** project entries search and file reads (`packages/contracts/src/project.ts`, `filesystem.ts`), assets (`assets.ts`), workspace mutation ids for refresh.

### 3.3 Thread find bar (Partial)

**Reached from** mod+F (`chat.find`, when the server has `threadFind`), the header menu "Find in thread" (05 §2.3) and the palette (12 §5.11 row 5).

**Covered:** the bar's top offset and 36 px height, as an input to the details card's placement (05 §3.5), and the chord (12 §5.7).

**Open:**

- the bar itself (`c/chat/ThreadFindBar.tsx:1–129`): absolute `top-3 right-3 z-40`, width `min(--thread-details-panel-width, 100% − 1.5rem)`, 150 ms fade-in; an input "Find in thread"; the count label "i/n", "1/…" while counting, "Searching…" and "Search failed"; "Previous match" and "Next match" (disabled without matches or while counting); "Close find";
- Enter for next and Shift+Enter for previous, with IME composition ignored (52–60);
- match and current-match highlights in the timeline, including folded rows that open to reveal a match (`threadFindHighlights.ts`, `useThreadFindNavigation.ts`, `useThreadFind.ts`, `threadFindStore.ts`, `useThreadFindPositionReader.ts`, `markdownFindContext.ts`);
- server-side search over history not yet loaded (`searchThread` and `searchThreadStream`, `packages/contracts/src/rpc.ts:1803–1815`);
- find state kept per route (`routes/_chat.tsx:9`, 60–73).

### 3.4 Pull requests (Absent)

**Pull requests page** (`/pull-requests`, a utility page in the main column; the sidebar footer turns into "Back", per 04): confirmed live.

- A title row "Pull requests" with the right panel toggle.
- A toolbar: search "Search pull requests, or label:bug", Sort, Filters, provider "All", refresh (`c/pullRequest/PullRequestsColumn.tsx:63–280`, `PullRequestListFilters.tsx:130–546`).
- Scope crumbs: state (All, Open, Closed, Merged), involvement (All, Reviewing, Authored), host and provider (`pullRequestListOptions.ts:20–39`).
- Filters: drafts (All, Drafts only, Hide drafts), reviews (Approved, Changes requested, Review required, No reviews), checks (Passing, Failing), author search, label search, project (`PullRequestListFilters.tsx:148–164`, 301–399, 546).
- Sorts: Merge readiness, Blocked on me, Recently updated, Newest shown, Oldest shown, Largest shown, Smallest shown.
- Grouped rows (`PullRequestListGroups.tsx`, `PullRequestListRow.tsx`, `PullRequestRow.tsx`) and skeleton rows (`PullRequestGhosts.tsx`).
- Empty and error states: "No pull requests", "Nothing under these filters", "Check again", "Load more pull requests", "Pull requests unavailable" (`PullRequestListEmptyState.tsx:119–145`, `PullRequestsPage.tsx:1189–1252`).
- The selected PR opens in the right panel. Until one is selected, the right panel toggle is unavailable, with the tooltip "Select a pull request first" (`PullRequestsPage.tsx:1145–1160`, `c/chat/PanelLayoutControls.tsx:52`, 132).
- Search parameters and their limits are in `routes/_chat.pull-requests.tsx:30–79`.

**PR detail surface** (right panel kind `pull-request`: `RightPanelSurfaceContent.tsx:120–173`, `PullRequestDetailPanel.tsx`, 2,014 lines). Opened from the details card's PR rows when links open in the app (05 §5.10), from the page, and from PR links in messages.

- **Header** (1537–1818): crumbs with "Back to this thread's pull requests" and "Open #n on host" (`PullRequestDetailCrumbs.tsx`); the base ← head branches; changed-file count; copy checkout command and branch name; checks; comment and commit counts.
- **Actions** (`PullRequestDetailActions.tsx:108–451`): Check out, Resolve conflicts, Ready for review, auto-merge badge, merge, "Approve workflows to run"; their failure messages at `PullRequestDetailPanel.tsx:172–181` and handoff toasts at 899–1062.
- **Tabs** Summary, Guide, Timeline, Code (209–212):
  - *Summary* (`PullRequestSummaryTab.tsx`, 1,141 lines): an editable description; comments with a filter and "Newest first" / "Oldest first"; reviews ("Resolved", "Review dismissed"); checks; reactions; "Fix in a thread".
  - *Timeline* (`PullRequestTimelineTab.tsx`, 918 lines): the activity feed with "Add to chat".
  - *Code* (`PullRequestCodeTab.tsx`, 1,721 lines): diff scope by commit; per-file viewed ticks; stacked or split; whitespace; wrap; collapse all; hide test files; file tree; line comments ("Add to review" or "Add to agent"); notices for partial diffs; "Conversations not on the current diff"; "Load more files". Its diff is a separate use of Pierre, so 13-diff does not cover it.
  - *Guide* (`PullRequestGuideTab.tsx`, `guide/*`: tour, rail, stops, notes, prose, excerpts, file diffs): an AI review guide.
- **Editing pieces:** review form, comment form and PR composer, markdown editor, textarea and renderer, reviewer and label pickers, title editor, resolve-thread button, "More replies".
- **Stacked PRs:** stack menu, popover, header and layers.
- **Other:** the PR actions menu, speed actions, the source-control app access notice (`c/SourceControlAppAccessNotice.tsx`).

**Thread PR list** (kind `pull-requests`, `ThreadPullRequestsPanel.tsx`, 492 lines; add menu "Linked pull requests" (L), palette "Show linked pull requests"): the thread's linked PRs, the candidate picker and the thread links (`PullRequestThreadLinks.tsx`, `PullRequestCandidatePicker.tsx`).

**Dialogs and hovers:**

- "Link pull request" (`LinkPullRequestDialog.tsx`, 178 lines; placeholder "Pull request URL or #42"), from the palette and from link menus.
- "Checkout pull request" (`c/PullRequestThreadDialog.tsx:205–227`): "Resolve a <provider> pull request, then create the draft thread in the main repo or in a dedicated worktree." Reached from the branch picker's "Checkout <pull request>" item (08 §5.8) through `usePullRequestDraftThread.ts:62` and `ChatOverlayHosts.tsx:65–100`.
- The PR link hover card on PR links in messages (`PullRequestLinkPreview.tsx`, 254 lines, built on `ui/preview-card.tsx`; 06 §5.8 says only that it exists) and the link context menu (`pullRequestLinkContextMenu.ts`).

**What exists elsewhere.** 05 covers the details card's PR rows, state icons and the checks popover; 04 covers sidebar PR badges; 12 covers the page's chords.

**Data:** `packages/contracts/src/pullRequest.ts` (1,013 lines), `threadPullRequest.ts`, `pullRequestReviewGuide.ts`, `review.ts`.

### 3.5 Browser surface and floating preview (Absent; planned 10-browser.md)

tern already draws a browser panel (`app/ui/browser.go`, `app/core/preview.go`) with no spec behind it. 03-shell covers the right panel frame. 05-header covers the browser tab's favicon, title and mute button, and the add menu's "Browser" row with its profiles submenu. Nothing covers the surface.

- **Chrome row** (`c/preview/PreviewChromeRow.tsx:97–289`):
  - Back, Forward, Refresh / Stop (tooltip "Loading…");
  - the address field "Search or enter URL" (Enter submits; Escape restores the current URL and blurs; IME composition is ignored, 170–180);
  - "Open in system browser";
  - "Annotate preview" / "Cancel annotation";
  - "Capture screenshot", where Shift-click records and "Stop recording" ends it;
  - "Float preview over chat" / "Close floating preview".
- **More menu** (`PreviewMoreMenu.tsx:26–167`, tooltip "More"):
  - Find (with its shortcut), Hard reload, Open DevTools (desktop);
  - open or close a separate preview window, a move item;
  - show or hide the device toolbar;
  - an Appearance submenu for the page's colour scheme (System, Light, Dark);
  - Zoom out, Reset zoom, Zoom in.

  A zoom indicator appears when the zoom changes (`ZoomIndicator.tsx`).
- **Find on page** (`PreviewFindBar.tsx:65–132`): "Find in page", previous, next, close. It is the target of `preview.find`.
- **Empty page** (`PreviewEmptyState.tsx`, `previewEmptyStateLogic.ts`): discovered local servers marked "Listening" (`PreviewLocalServerCard.tsx`, `useDiscoveredLocalServers.ts`, `ui/discovery-list.tsx`) and recent URLs, each with "Remove <url> from history" (`PreviewRecentUrlCard.tsx`).
- **Unreachable page** with "Details" / "Hide details" (`PreviewUnreachable.tsx`).
- **Agent cursor** over the page while an agent drives it (`AgentBrowserCursor.tsx`).
- **Annotations with voice input** (`PreviewAnnotationSpeech.tsx`, `lib/previewAnnotation.ts`).
- **Device toolbar** (`browser/BrowserDeviceToolbar.tsx:31–330`): "Responsive" and device presets, width and height fields, aspect lock, "Rotate viewport", close. Viewport resize handles on the left, right and bottom edges respond to the arrow keys (`browser/BrowserViewportResizeHandles.tsx:60–136`).
- **Server browser** (the server's own Chromium, which is what tern streams), `browser/ServerBrowserSurface.tsx`:
  - "Take control" / "Release control" (767);
  - a file chooser (829);
  - page dialogs with a response field (860–867);
  - the toast "The page copied text" (113).

  Also: crash recovery (`browser/webviewCrashRecovery.ts`), recording (`browser/browserRecording.ts`, `recordingCompositor.ts`) and picture in picture (`browser/serverPictureInPicture.ts`).
- **Floating preview** (mini player) over the chat, for a browser or device: "Pop into separate window" / "Close separate window", "Open preview in right panel", "Close floating preview", a recording indicator (`ThreadPreviewMiniPlayer.tsx:255–611`), and its placement and snapping (`previewMiniPlayerLayout.ts`, 327 lines). 03-shell §2.2 leaves it to "the preview doc", and 05 uses only its frame for the details card.
- **Opening rules:**
  - links (`browser/browserLinkTarget.ts`, 06 §5.8);
  - terminal links (`openTerminalLinkInPreview.ts`, 09 §5.9);
  - files ("Open file in preview browser", `browser/openFileInPreview.ts`);
  - agent-opened previews (`agentOpenedPreviews.ts`);
  - discovered ports (`openDiscoveredPort.ts`).
- **Data:** `packages/contracts/src/preview.ts` (406 lines), `desktopBrowser.ts`, `browserProfile.ts`, `previewCapture.ts`, `previewAutomation.ts`.

### 3.6 Usage modal (Absent)

Confirmed live on Limits. The modal frame is the same as Settings (§3.1). `routes/_chat.usage.tsx`, `c/usage/UsagePage.tsx` (1,377 lines).

- **Header** (361–495):
  - breadcrumb "Usage / <environments ▾> / <providers ▾>" (environment and provider pickers, 1137–1260);
  - a segmented period toggle (Past 24h, 7 days, 30 days, 90 days; `usageShortcuts.ts:12–17`), hidden on Limits, which "are always current" (UsagePage.tsx:393–411);
  - the segmented metric Cost, Tokens, Limits;
  - "Refresh usage" / "Refresh limits";
  - close.
- **Cost and Tokens views:**
  - a provider chart over time with its "Total" (`UsageProviderChart.tsx`);
  - Totals: Processed tokens, Cached input, Uncached input, Output, Cache savings (785–840, 1341–1343);
  - share bars: Tokens by type, Cost by type, Cost by speed (`UsageShareBar.tsx`);
  - a Breakdown table by Model or by Day / Hour, with columns Model, Cost, Share, Tokens (847–970);
  - unpriced usage details (681);
  - a per-model dialog with cost, tokens, per 1M tokens, cache hit and "Set price" (`UsageModelDialog.tsx:61–143`);
  - the "Custom model prices" dialog with an editable table (`UsagePriceOverrides.tsx:71–456`, `ui/table.tsx`).
- **Limits view:**
  - one card per provider and window (live: "Monthly", "<n>% left" with a pace icon);
  - pace states Ahead of pace, On pace, Under pace (`UsageLimits.tsx:47–67`);
  - per-account bars with initials avatars and reset countdowns, a reveal-email toggle, "Even pace" and remaining percentages (`UsageLimitsPooled.tsx:64–364`);
  - a confirm dialog (`UsageLimits.tsx:300`).
- **Keys:** c, t and l switch the metric; mod+Shift+1 to 4 switch the period. 12 §5.7 covers these.
- **Data:** `packages/contracts/src/usage.ts`, `providerUsageLimits.ts`, `usageLimitSourceId.ts`; `packages/shared/src/usageMerge.ts`.

### 3.7 Chat notices (Named)

- **Banner lane** at the top of the chat canvas (`c/chat/ChatBannerLane.tsx:44–63`). 03-shell §3.11 gives only its z-index (30).
  - The provider status banner (`ProviderStatusBanner.tsx`, 192 lines) is a `ui/alert` in the warning or error variant (145–151). It shows the selected provider's warning or error status, sign-in required, and versions that are unsupported or "known to be broken" (19–92, 131–140). It is dismissible per status key (54–57). 08 cites lines 60–93 only for the "Update required" logic.
  - The thread error banner (`ThreadErrorBanner.tsx`) shows the message mapping (29–40), a copyable admin request (17), and dismissal.
- **Composer banners 07 lists as "not detailed here"** (07 §2.9):
  - the feedback notice (`ComposerFeedback.tsx`: "Could not copy thread ID", "Dismiss feedback notice");
  - usage-limit recovery (`UsageLimitRecoveryBanner.tsx:72–151`: "Usage limit reached" or "ChatGPT usage limit reached", "Resume at reset" / "Cancel auto-resume", "Snooze until reset" / "Wake now");
  - provider extension status (`ThreadActivityBanners.tsx:36–50`);
  - project clone progress (`useProjectCloneBanner.tsx:35–60`: "Cloning repository", "Repository not cloned", "Copying files");
  - branch mismatch with "Restore branch" (`ThreadActivityBanners.tsx:272–310`) and its confirm "Switch to <branch>?" (`ChatOverlayHosts.tsx:139–190`).
- **Banners 07 does not list:**
  - child threads waiting, "Subagent waiting" / "Side chat waiting" with "Open approval" / "Open question" (`ThreadActivityBanners.tsx:60–90`);
  - stopping, "Stopping" / "Agent not responding to Stop" with "Force stop" (140–165; `useForceStopThread.ts`);
  - the auto-balance update notice, "Update available for…" / "Updating…" / "Could not update…" with "View machines" and "Update all" (`useAutoBalanceUpdateBanner.tsx:28–140`).
- **File drop overlay** while files are dragged over the chat (`c/chat/TimelineOverlays.tsx:5–20`): a dashed `primary/60` frame inset 8 px, and a pill with a paperclip and "Drop files to attach". Drag tracking is in `workspaceFileDrop.ts` and `folderDrop.ts`. 03-shell §2.2 names it only.
- **Pending guest messages** in the timeline footer (`c/chat/GuestShareControls.tsx:220–326`, mounted at `ChatView.tsx:3137`): "<n> guest messages waiting", each with Dismiss and Send. 06 §2.1 names the slot only.

### 3.8 Timeline rows and media named but not specified (Named)

06-timeline lists these row kinds or mentions them, without structure, values or behaviour.

- **HTML render frame** (`c/chat/HtmlRenderFrame.tsx`, 126 lines):
  - a frame whose height comes from the server's measurement and the frame's reported content height at the current column width (`htmlRenderFrameHeight`, 28–38);
  - a hover toolbar at the top end with "Open full screen" (150 ms fade, visible on focus and on coarse pointers, 81–107);
  - the toast "Page unavailable" and long-lived asset URLs (49–56);
  - theming (`hooks/useHtmlRenderTheme.ts`);
  - chords replayed out of the frame (12 §1, `BrowserDocumentFrame.tsx`).
- **HTML render full-screen dialog** (`HtmlRenderDialog.tsx`, 108 lines): `w-[min(92vw,1176px)]`, max height `min(86vh, 100vh − 160px)`, a close button above, and the title with Zoom out, Reset zoom and Zoom in below.
- **MCP app frame** (`McpAppFrame.tsx`, 494 lines):
  - a sandboxed frame (`allow-scripts allow-forms`), inline with an optional rounded border;
  - full screen as a fixed top-layer popover with a 40 px header holding the server name and "Exit full screen" (434–466). The inline row keeps its height so the timeline does not jump;
  - the states "The <server> app was closed" with "Show app", "… left its page and was stopped", and "Unable to load the <server> app" (416–489).

  Also `c/mcp/mcpApps.ts`, `mcpGate.ts`, `state/mcpApps.ts`, `packages/contracts/src/mcpApps.ts`.
- **Worktree setup card** (`WorktreeSetupCard.tsx`, 394 lines):
  - `section "Worktree setup"` with `data-worktree-setup-phase`;
  - the header "Worktree ready", "Worktree setup failed" or "Worktree setup cancelled", with a spinner and the live-activity focus band;
  - step rows with tabular times;
  - a `dl` of Branch, Base, Path and Setup in mono with middle truncation;
  - the log tail.
- **Workflow opening block** (`WorkflowOpeningBlock.tsx`, 242 lines): "Workflow started: <title>", phases with member chats ("Open <member>"), and the script with "Show full script" / "Show less". The details card's workflow row (`ThreadLineageWorkflowRow.tsx`: members, "Stop workflow", phase expansion) is outside 05's scope, which records only the lineage row look.
- **Image zoom and pan** inside 06's image dialog (`ZoomableImage.tsx`, 261 lines): up to 8× zoom (11, 119), pinch through Ctrl+wheel, plain wheel scrolling, and mouse-drag panning when zoomed (137–226). Also `ExpandedImagePreview.tsx` (item model, asset URLs).
- **Video player** (`c/media/MediaVideoPlayer.tsx`): poster, "Play <name>", loading, "Video preview unavailable", "Retry video". The media context menu (`MediaActions.tsx:133–191`) has Copy full path, Copy relative path, Open in file viewer, Copy image, Save and Copy URL, with their toasts.
- **Provider subagent bar** (`ProviderSubagentBar.tsx`, 78 lines), which replaces the composer on provider-native child threads (07 names it only), and the "Subagent work" dialog (`NativeChildWorkDialog.tsx`: "Read only. These subagents cannot take messages.", one section per child; `ChatView.tsx:3578–3590`).
- **Link request wizard:** "Review link…" / "Choose machine…" opens a wizard dialog (06 §2.8 names it; `LinkRequestCard.tsx` with `ui/wizard.tsx`).
- **Revert confirm** "Edit from here?" with its text and actions (`ChatOverlayHosts.tsx:192–235`). 06 covers the button only.
- **External link context menu** in the browser build (`externalLinkContextMenu.ts`). 06 §5.8 lists its items as a native menu.

### 3.9 Composer parts named but not specified (Named)

- **Code blocks in the rich-text composer:** the node view with its language label and picker, and its highlighting (`c/chat/ComposerCodeBlockNodeView.tsx`, `ComposerCodeBlockLanguagePicker.tsx`, `c/composerCodeBlockHighlight.ts`, `composer-code-block.ts`, `composer-code-languages.ts`). 07 §5.5 has the keys inside a code block; §2.4 and §7 only name the node view and picker.
- **Chip popovers:** the model chip (`ModelMentionChip.tsx`), the PR chip, and the skill chip text (`SkillInlineText.tsx`). 07 §2.12 gives the skill popover's contents in one line, and the rest are named in §7.
- **Voice input setup dialog** (`VoiceInputSetup.tsx`, 277 lines, a wizard). 07 §7 says it is not covered.
- **File preview dialog contents:** `ComposerFilePreviewDialog` is in 07's tree; its body is the attachment preview of §3.2.

### 3.10 Side chat pane (Named)

Right panel kind `side-chat` (`c/chat/SideChatPane.tsx`, 183 lines).

- An `h-9` header with the title (12 px, 500, truncated) and the buttons:
  - "Bring back", with the tooltip "Adds this side chat to the main thread's next message." or the reason it is blocked, and the toast "Brought back";
  - Delete, confirming `Delete side chat "<title>"?`;
  - Close.
- The deleted state reads "This side chat was deleted."
- Below the header is the side chat's own chat view in embedded mode. Embedded mode drops the chat header, details panel, branch restore, terminal drawers, mini player, right panel and link dialog (`ChatView.tsx:3295–3312`, 3834–3954, 4023).
- Side chats are offered when the server has `sideChats`, the view is not embedded, the user can operate, and the thread is neither a side chat nor a subagent (`ChatView.tsx:2785–2800`). Also `useSideChat.ts` and `lib/sideChatPane.ts`.
- **Covered elsewhere:** the `/side` and `/btw` commands (07 §5.6), the tab icon (05 §3.11), the details card rows (05 §3.10), and the embedded timeline's header divider (06 §2.1).

### 3.11 Dialogs reached from covered surfaces (Absent or Named)

The AlertDialog and Dialog frames are in 02-primitives. Each row below still needs its content, fields, validation, actions and toasts.

| Dialog | Reached from | Sources |
|---|---|---|
| Share with guest: "Guest link", "Create link" / "Creating...", copy (the header indicator shown afterwards is in 05) | header menu (05 §2.3), palette, sidebar row menu (04 §5.13) | `c/chat/GuestShareControls.tsx:48–175` |
| Add / Edit action: Name, "Choose icon", Keybinding ("Press shortcut"), Command, "Preview URL (optional)", "Save action" / "Save changes" (02 §3.21 measured the dialog frame, not its fields) | details card "Add project script" and the gear on a script (05 §5.8) | `c/projectScriptEditor.tsx:417–568` (573) |
| Add to another machine: source and target machine, clone protocol | run-context menu (05 §3.7) | `c/AddProjectToMachineDialog.tsx:227–303` (447), `lib/addProjectToMachine.ts` |
| Use T3 Code in another app (MCP): wizard with app logos | palette (12 row 23), Sharing & access | `c/mcp/UseInAnotherAppDialog.tsx`, `c/mcp/logos/*.png` |
| "Your ChatGPT plan is connected": shown once per newly connected Codex/ChatGPT profile, "Manage usage" and "Continue" | automatic, root host | `c/settings/ChatGptWelcomeCoordinator.tsx:57–83` |
| Relay client install | T3 Connect flows, root host | `c/cloud/RelayClientInstallDialog.tsx` |
| Theme editor: a floating, minimizable "Create theme" / "Edit theme" panel with name, appearance, "Filter colors", "Use advanced theme colors", colour roles (Raised surface, Muted text, Subtle surface, Highlight surface, Message surface, Code surface, Search matches, Current match, Sidebar background, Sidebar controls, Sidebar selection, Terminal background, …), "Inspect app colors" (pick a colour from the app), "Save changes" | mod+Alt+Shift+T, palette "Toggle theme editor", Appearance settings | `c/settings/ThemeEditorHost.tsx`, `ThemeEditorPanel.tsx` (1258), `themeInspector.ts` (474), `ThemeColorPicker.tsx`, `themeEditorStore.ts` |
| Set up devices: wizard "Device hub", "Simulators", "Agent access" | device surface, `ChatView.tsx:3296` | `c/device/DeviceSetup.tsx:141–336`, `ChatOverlayHosts.tsx:103–137` |
| Custom snooze: 04 §5.14 has its copy and fields, but not the calendar popover's look | snooze menu "Custom…" | `c/CustomSnoozeDialog.tsx`, `ui/calendar.tsx` |
| Confirm dialogs raised through `requestConfirmDialog`: texts and button variants per use, beyond 09's terminal close and 04's update confirm | many | `c/ConfirmDialogHost.tsx`, `confirmDialog.ts` |

### 3.12 Device surface (Absent)

Right panel kind `device` (`c/device/DevicePanel.tsx` and 23 more files, 4,309 lines). It is reached from the add menu "Device" (M); 05 covers the tab's rename, icon and tooltip.

- **Device list** per host: Open or Start, discovery retry, details, warnings and dismissals (`DevicePanel.tsx:255–474`), with host availability ("iOS available", "Android available") and updates.
- **Live stream** in a 3D device frame (`DeviceStreamView.tsx`, `DevicePhoneViewport.tsx`, `deviceFrameLayout.ts`, `models/*.glb`), including dual-screen and foldable controls.
- **Controls rail** (`DeviceControlsRail.tsx:59–307`): Rotate, Text size, Tools, Save screenshot, More (Power off or Return device to the pool), Flat view.
- **Tools panel** (`DeviceToolsPanel.tsx:28–558`):
  - host diagnostics;
  - open a URL or deep link, launch an app;
  - appearance, text size, Liquid Glass, colour filter, orientation;
  - overlay element frames;
  - push notification;
  - location presets;
  - permissions.
- **Trackpad gestures** (`phoneTrackpad.ts`) and the loading view.
- **Data:** `packages/contracts/src/device.ts` (736 lines).

### 3.13 Toasts outside the catalogue (Partial)

11-empty-auth §5.6 calls itself the catalogue and lists 11 sources. The client has 692 `toastManager.add` / `stackedThreadToast` call sites in 164 files. By directory, 37 of those files are in `c/settings`, 34 in `c/chat`, 27 in `c/` and 20 in `c/pullRequest`. Surface docs list some of their own toasts (04, 05, 12), but none of the following is specified anywhere:

- **"Enjoying T3 Code?"** / "Give us a star on GitHub to let us know!" with "Open GitHub" → "Starring…" → "Thanks for the star!" and a confetti burst. It is raised after sends, once the window is idle (`c/GitHubStarPrompt.tsx:31–168`, `c/chat/useComposerSend.ts`).
- **Project clone progress** (`c/ProjectCloneToastCoordinator.tsx`, 229 lines).
- **Thread title regeneration** (`c/threadTitleRegenerationToast.ts`).
- **Source-control app access** (`c/sourceControlAppAccessToast.ts`).
- **Media, files, side chat, PR, SnapShot and preview toasts**, cited in §3.2, §3.4, §3.5, §3.8, §3.10 and §3.16.

The catalogue should either become complete or point to the owning docs.

### 3.14 Primitives with no spec (Absent)

02-primitives cites 36 of the 52 files in `components/ui/`. Six others are covered by other docs: `alert` (11), `command` (07, 12), `refresh-icon` (13), `sidebar` and `sidebarState` (03, 04), and `standalone-page` (11). The remaining 10 are covered nowhere, and the surfaces above need them.

| Primitive | Used by |
|---|---|
| `checkbox.tsx` (109) | Welcome page, Connect onboarding, add to another machine, the palette's new-project options, the PR Code tab, Git actions, authorized clients |
| `radio-group.tsx` (34) | MCP dialog, connect-agent page, ChatGPT account picker, add provider instance, link request card, Git actions |
| `group.tsx` (72) | project scripts and Open in split buttons (05 measured the live elements, but not the primitive) |
| `calendar.tsx` (110) | custom snooze |
| `wizard.tsx` (155) | MCP dialog, Connect onboarding, provider wizard, text generators, Codex account, SnapShot setup, voice setup, link request, Git actions |
| `color-picker.tsx` (226) | theme editor, provider accent colour |
| `discovery-list.tsx` (37) | browser empty page, device list |
| `preview-card.tsx` (44) | PR link hover card |
| `table.tsx` (67) | custom model prices |
| `qr-code.tsx` (81) | Welcome phone panel, mobile app row, pairing links |

### 3.15 Interface language (Absent)

Settings › General › Language offers "System default" ("System default follows your browser or OS language.") and eight locales: de, es, fr, ja, ko, pt-BR, ru, zh (`apps/web/gt.config.json`, `i18n/locales.ts`, `SettingsPanels.tsx:783–785`, 1043–1046).

About 70 strings are translated today, mostly the Settings nav and General page (`i18n/translations/*.json`, 74 lines each, through gt-react in 7 files). Rendering waits for the locale (11 §2.1 mentions only the wait).

Not specified:

- which strings change;
- the locale resolution (`i18n/I18nProvider.tsx`, `initializeI18n.ts`, `runtime.ts`, `packages/client-runtime` locale names);
- how translated widths affect the layouts specified in English.

### 3.16 Desktop-only main-window pieces (Absent or Partial)

- **Application menu structure** (`apps/desktop/src/window/DesktopApplicationMenu.ts:160–315`):
  - app menu: About, Check for Updates..., Settings..., Services, Hide, Hide Others, Show All, Quit;
  - File: Settings... off macOS, Close Window or Quit;
  - Edit: Undo, Redo, Cut, Copy, Paste, Paste as Text, Delete, Select All, Speech;
  - View: Reload, Force Reload, Toggle Developer Tools, Actual Size, Zoom In, Zoom Out, Toggle Full Screen;
  - Window.

  12 §5.7 lists the accelerators only. tern draws its own menu bar (`app/ui/screens.go:35`).
- **Quit hint:** a pill "Hold <shortcut> or press twice to quit" at 22% from the top, `z-100`, `rounded-full bg-foreground/95 px-8 py-4 text-2xl font-bold`, driven by the Quit shortcut setting (Direct, Hold, Press twice) (`c/QuitHoldOverlay.tsx:51–70`).
- **SnapShot capture:** a global shortcut captures a window into a draft, with a fly-in animation and a sound, and the toasts "Snapshot failed" and "Snapshot taken, but no project is available" (`c/desktop/SnapShotCoordinator.tsx:377–487`, `lib/snapShotAnimation.ts`, `lib/snapShotSound.ts`, `lib/snapShotShortcut.ts`, `state/desktopSnapShot.ts`). 07 covers SnapShot attachments only.
- **"SSH verification required"** password prompt for SSH hosts (`c/desktop/SshPasswordPromptDialog.tsx:166–228`).
- **Desktop deep links:** the confirm "Add the backend at <origin>?", then "Backend added" / "Could not add backend" / "Could not open the link" (`c/desktop/DesktopWebLinkCoordinator.tsx:108–145`).
- **Editor brand icons** in the details card's Open in picker: VS Code, Insiders, VSCodium, Cursor, Zed, Kiro, Trae, Antigravity, the JetBrains IDEs, Finder, File Explorer and the Linux file manager (`c/Icons.tsx`, `c/JetBrainsIcons.tsx`). 05 saw no enabled editor, so only the fallback icon is specified.

### 3.17 T3 Connect account pages (Named)

When the user is signed in, the Settings nav's avatar opens Clerk's profile with three T3 pages (`c/clerk/T3ConnectAccountPages.tsx:12–26`). 11 §2.6 names them only.

- **Mobile clients:** push notification and Live Activities badges per device, and a loading state (`MobileClientsUserProfilePage.tsx`).
- **T3 Connect:** linked environments, "Managed tunnel" / "Activity publishing only", deregistering with a confirm, and the toasts "Server deregistered" / "Could not deregister server" (`T3ConnectUserProfilePage.tsx`, `RemoveT3ConnectEnvironmentDialog.tsx`, `c/cloud/TunnelUsage.tsx`).
- **T3 Chat:** the link state, either the email or "Not linked" (`T3ChatUserProfilePage.tsx`).

### 3.18 Prompt-link pages (Absent)

- **`/new?…`** (`routes/_chat.new.tsx:1–28`, `c/promptLinks/NewThreadLinkView.tsx`):
  - a parse error shows the empty state "Could not open the link" with the reason;
  - otherwise it resolves the repository and project, may clone it ("Cloning <owner>/<name>", "Clone failed"), offers "Check out", or shows "This link is open" or an error page (`NewThreadLinkView.tsx:222–453`).
- **`?prompt=` on a thread link** places the prompt in the composer and strips it from the URL, with the error toast "Could not open the linked prompt" (`c/promptLinks/ThreadLinkPrompt.tsx`).
- 12 covers the palette's "Copy prompt link" rows that make these links (`copyPromptLink.ts`, `linkedPrompt.ts`).

## 4. Checked and left out

- **`/guest`**, the guest page (`c/guest/*`). 11-empty-auth marks it out of scope: guests see it, not the main window's user.
- **`DevReplayHost`** renders only in dev builds (`routes/__root.tsx:196`).
- **`HostedPromptLinkLanding`** and the other hosted-app-only states beyond what 11 covers.
- **Redirect-only routes:** `/settings/beta`, `/settings/integrations`, `/settings/storage`, `/projects/$projectKey` (to Settings › Projects) and `/threads/$environmentId/$threadId` (to the thread).
- **Non-visual hosts and helpers:** `ThreadVisitWatermark` (renders null), `RunningThreadKeepAlive`, `DesktopAppActivationCoordinator`, `RenderErrorBoundary` (shows the caller's fallback), `AnimatedHeight` / `AnimatedWidth` (motion helpers used by covered surfaces), and the state, RPC and query modules.
- **The terminal renderer, keybinding model, palette, sidebar, header, details card, timeline, composer, pickers, diff and the empty and auth surfaces.** Their docs exist, and their own §7 Gaps are not repeated here.

## 5. Limits of this audit

- **Coverage was judged by name, string and behaviour.** A file a doc cites can still hold unspecified parts. This list catches those only where a doc names a part without describing it.
- **Docs change while this runs.** 12-keys-palette changed during the audit, and 10-browser.md may land after it. Re-run the name match (§1, step 1) before acting on §3.5.
- **No values were measured.** Line counts are non-test source lines, a rough size signal only.
- **Privacy.** All docs were scanned for personal names, machine names and project names. An earlier revision of 12-keys-palette.md §7 named a person. Its 01:56 revision says "a product decision" instead. No other hit.

## 6. Port notes for tern

- **Model.** `app/ui/model.go` `Scene` has no fields for the palette, Settings, toasts, pull requests, usage, files, find, side chats, devices, the banner lane or the theme editor. Errors and notices are single strings (`Error`, `Notice`). Each surface above needs its own view model, filled by `app/core/scene.go` from the protocol.
- **Files.** Today `app/ui/screens.go` (2,881 lines) draws the whole window, `app/ui/browser.go` and `app/core/preview.go` the browser panel, and `app/ui/details.go` the details card. Natural new files: `ui/settings.go` (modal frame and nav; one file per page group), `ui/files.go`, `ui/find.go`, `ui/pullrequests.go`, `ui/usage.go`, `ui/toast.go`, `ui/devices.go`, plus a route-modal primitive in `ui/primitives.go` shared by Settings and Usage.
- **Data tern's core must provide:**

  | Surface | Data |
  |---|---|
  | Settings | client and server settings with their scopes (`packages/contracts/src/settings.ts`) |
  | Files | project entry search and file reads (`project.ts`, `filesystem.ts`) and asset URLs (`assets.ts`) |
  | Find | `searchThread` / `searchThreadStream` (`rpc.ts:1803–1815`) and the server's `threadFind` capability |
  | Pull requests | pull request list, detail, review, guide and thread links (`pullRequest.ts`, `threadPullRequest.ts`, `pullRequestReviewGuide.ts`, `review.ts`) |
  | Usage | usage and limits (`usage.ts`, `providerUsageLimits.ts`) |
  | Browser | sessions, frames, control and capture (`preview.ts`, `previewCapture.ts`, `previewAutomation.ts`, `browserProfile.ts`) |
  | Devices | devices (`device.ts`) |
  | Guest sharing | guest shares (`guestShare.ts`) |
  | MCP apps | MCP apps (`mcpApps.ts`) |
  | Scheduled tasks | scheduled tasks (`scheduledTask.ts`) |
  | Worktrees | worktree setup snapshots (`worktreeSetup.ts`) |

- **Order.** The browser panel is already drawn in tern without a spec. Write 10-browser.md first, then Settings (the largest surface, reached from the always-visible footer gear), then Files (opened by every file chip in a message).
