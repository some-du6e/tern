# 13 · Diff panel and changed files

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
This document covers every surface that shows changes: the **Diff** surface of the right panel (its header row with the scope menu, the branch/commit comparison controls and the toolbar; the code view with file headers, hunk separators, line numbers, indicator bars, word-level emphasis, selection, the gutter comment button, review-comment annotations and the find panel; the file tree pane), the **changed files card** under an assistant message in the timeline, `DiffStatLabel`, and the loading, empty, error and notice states of all of them. The right panel frame, its resize handle and its sheet form are in [03-shell.md](03-shell.md) §3.7–§3.8; its tab bar (the "Diff" tab with its `file-diff` icon, the "+" launcher row "Diff" with shortcut letter D) is in [05-header.md](05-header.md) §2.8, §3.11 and §3.12, the thread details "Changes" row in §3.8 and the first description of `DiffStatLabel` in §2.9; the changed files card's place in an assistant row is in [06-timeline.md](06-timeline.md) §2.4 and §3.4 and the syntax palettes in §3.6; the review-comment chip in the composer is in [07-composer.md](07-composer.md) §2.12. Tokens are in [01-tokens.md](01-tokens.md) (§3.2 has every `--diff-*` token and the blue-orange variant), primitives (Button, Toggle, ToggleGroup, Menu, Combobox, Switch, Tooltip, Textarea, Skeleton, MiddleTruncate) in [02-primitives.md](02-primitives.md).

Conventions, as in 01 to 11:

- Lengths are CSS px at the default 16px root (1 CSS px = 1 Gio dp). Tailwind classes in `rem` scale with the interface font size. The diff viewer's own sizes (13px code, 20px lines, 32px file header, 24px hunk separator, `ch`-based gutters) are written in px or `ch` inside the viewer and do not follow the root size; the code size follows the monospace size setting (`--diffs-font-size`, 13 by default, written on `<html>` together with `--font-size-code` by appearanceFonts.ts:106–108). T3 sets no `--diffs-line-height`, so code rows stay 20px (Pierre's fallback) at every code size; `ch` widths scale with the size, `1lh` widths (the gutter "+" button) do not.
- Colours are sRGB hex as Chrome paints them, `#rrggbbaa` when translucent. Translucent values read through `inspect.sh` can be a unit or two off in RGB (01-tokens conventions); where the token is known its exact value is given first. The Diff surface and the timeline are outside `[data-app-sidebar]`, so they always use the `:root` palette, never the sidebar scope.
- **M** = measured live (computed style or box, both schemes unless a row says otherwise). **P** = the real class list or CSS rule evaluated on a probe element in the live page, or a value computed from the shipped CSS formula with T3's own inputs (§3.14; the formulas reproduce every measured colour to within 2 units per channel). **S** = source only.
- Icons are lucide (`lucide-react ^0.564.0`) at stroke 1.75 unless a row says otherwise (measured `stroke-width: 1.75px` on every lucide icon here). The diff viewer and the file tree draw their own icons from sprite sheets (Pierre icons, §2.12); those are not lucide.
- Coordinates are viewport px at 1440×900 @2x, sidebar expanded (256), thread details card hidden while the right panel is open, right panel inline at its default 540px (surface area x 901..1440, y 52..900).
- Text renders in Inter and, for code, Liberation Mono (the first installed face of `--font-mono` on this Linux host; 1ch = 0.6em, so 7.80px at 13px, 6.00px at 10px). macOS renders SF Pro and SF Mono. Sizes, weights, line heights and colours are font independent; text and `ch`-based widths are not.

**How it was measured, and what was not touched.** The paired browser had `orchestration:read`, `terminal:read`, `terminal:operate` and `preview:operate`, and no filesystem read scope. That splits this topic in two. T3 gates Git diffs on `useFilesystemReadAccess`, so the Git scopes (Changes, Uncommitted, Commits, the base-ref picker) showed only "This connection cannot read local diffs." Turn (checkpoint) diffs come from `orchestration.getTurnDiff`, so they rendered fully and were measured in both schemes. I opened the Diff surface from the thread details "Changes" row and from a changed files card's "Open diff", collapsed, expanded and revealed files, showed and hid the file tree pane, and flipped the two session-only toggles (line wrapping, whitespace) and set them back. I selected a line and opened a draft comment with the gutter button, then cancelled it without typing. I opened the viewer's find panel with its shortcut and closed it with its close button, again without typing. I did not press the stacked/split toggle, because it writes the `diffLayout` setting, so split view is source-derived (§2.5). No comment was submitted, so the saved-comment annotation is P/S. Reference captures stay in a private folder outside this repo.

## 1. Scope and sources

Paths are relative to the T3 repo; `c/` is `apps/web/src/components`, `w/` is `apps/web/src`. Shipped assets are in `client/assets/` of the pinned runtime; minified bundles are cited by asset name and the selector or function they contain.

| File | Lines | What it gives |
|---|---|---|
| `c/DiffPanel.tsx` | 10–23, 116–117 | lucide icons used; `AUTOMATIC_BASE_REF`, `t3code.diffFileTreeOpen` |
| | 137–186 | `DiffFileCollapseToggle` (file header chevron, lazy-load observer) |
| | 188–211 | `DiffFileHeaderSuffix` (copy-path button, per-file status) |
| | 226–289 | settings read, local wrap state, session whitespace, tree/test-file storage, current selection |
| | 290–371 | scope kinds, turn ordering, reconcile, turn range, `useCheckpointDiff` |
| | 372–469 | Git preview input, commits query, comparison refs, scope and section labels, commit URL |
| | 473–555 | which patch is shown, truncation, lazy sources, per-file stats, totals, file-contents loader |
| | 556–618 | base-ref choices, `useReviewFilePatches` |
| | 622–639 | live refresh on show |
| | 641–797 | file keys, test files, collapse state, reveal, open file, toggles |
| | 799–835 | scope selection handlers |
| | 837–1298 | the header row: scope menu (840–1004), comparison refs and base-ref combobox (1005–1133), toolbar (1135–1296) |
| | 1300–1559 | panel body: empty states, notices, raw content, loading, code view, file tree, raw patch |
| `c/DiffPanelShell.tsx` | 10–47 | shell and header row classes per mode |
| | 49–116 | file-header skeleton, code-line skeleton, `DiffPanelLoadingState` |
| `c/chat/ChangedFilesTree.tsx` | 22–154 | `ChangedFilesCard`: card, sticky header, actions, hidden-file button |
| | 156–275 | `ChangedFilesTree`: folder expansion state, rows |
| | 277–426 | row rendering, virtualization above 250 rows, focus retention, row collection |
| `c/chat/DiffStatLabel.tsx` | 1–48 | `hasNonZeroStat`, `formatCompactDiffCount`, `DiffStatLabel` |
| `c/chat/PierreEntryIcon.tsx` | 7–95 | file icon colours per token, sprite `<use>` icon, lucide fallback |
| `c/chat/TimelineAssistantRows.tsx` | 72–79, 366–449 | where the card mounts; folders default collapsed (412) |
| `c/DiffFilePathCopyButton.tsx` | 1–40 | copy-path button |
| `c/diffs/AnnotatableCodeView.tsx` | 25–324 | diff items, review-comment annotations, draft state, gutter utility |
| `c/diffs/StyledDiffCodeView.tsx` | 15–215, 231–269 | T3's CSS injected into the viewer; item metrics; surface class |
| `c/diffs/DiffCommentAnnotation.tsx` | 47–204 | draft and saved comment UI, keys |
| `c/diffs/commentSubmitShortcut.ts` | 7–19 | mod+Enter rule |
| `c/diffs/DiffFileStatus.tsx`, `DiffFileLoadingBoundary.tsx` | 5–39; 6–34 | per-file error/partial icon; lazy batch boundary |
| `c/diffs/DiffFileTree.tsx` | 36–251 | file tree pane: Pierre tree model, header, reveal, click handling |
| `c/diffs/diffFileTree.logic.ts` | 17–178 | git status per file, file/directory aliasing, ordering, incremental updates |
| `c/diffs/useReviewFilePatches.ts` | 16–220 | file ordering, lazy patches (4 at a time), states, placeholders |
| `c/diffs/useCodeViewFileReveal.ts`, `useCodeViewScrollRestore.ts`, `useHideTestFiles.ts` | 30–181; 1–79; 5–9 | reveal, scroll restore, test-file toggle storage |
| `c/files/FileExplorerPane.tsx` | 8–114 | tree pane width rules and resize keys |
| `c/files/fileTreeExpansion.ts` | 36–60 | expand/collapse all folders |
| `c/ui/refresh-icon.tsx` | 1–38 | refresh icon and its spin |
| `c/chat/useRightPanelActions.ts` | 97–108, 130–140, 291–303, 325 | opening and toggling the Diff surface |
| `c/chat/RightPanelSurfaceContent.tsx` | 31, 108–119 | the only `DiffPanel` mount (`mode="embedded"`, lazy) |
| `c/chat/V2ItemInspector.tsx` | 280–305 | "Open diff" on a file-change tool row (tool-call scope) |
| `c/chat/UserMessageContextChips.tsx` | 560–617 | review-comment chip body renders the commented hunk |
| `w/diffPanelStore.ts` | 16–225 | selection model, persisted per thread; session folds and whitespace |
| `w/rightPanelStore.ts` | 703–712, 1123–1137 | opening the singleton Diff surface; toggle hides an active Diff tab |
| `w/diffFileActions.ts` | 55–112 | title click: workspace-relative path, file surface or editor |
| `w/state/queries.ts` | 415–455 | `useCheckpointDiff`: turn range or whole-thread diff |
| `w/uiStateStore.ts` | 6, 300–320 | `t3code:ui-state:v1`, changed files folder state per run |
| `c/DiffWorkerPoolProvider.tsx` | 52–72 | worker pool render options (no `lineDiffType`) |
| `c/settings/SettingsPanels.tsx` | 147–150, 1444–1491, 1497–1605 | Diffs settings page, diff colour picker |
| `c/pullRequest/PullRequestCodeTab.tsx` | 1636–1639 | contrast: the PR code tab cancels reveals on input, the Diff panel does not |
| `w/lib/diffRendering.ts` | 7–16, 59–198, 269–352 | themes, line stats, whitespace filter, patch parsing, change-icon and surface CSS |
| `w/lib/turnDiffTree.ts`, `w/lib/diffCollapse.ts` | 33–150; 1–13 | card tree building and compaction; collapse-all rules |
| `w/lib/toolCallDiff.ts`, `w/lib/baseRefChoices.ts` | 1–77; 1–61 | tool-call patches; base-ref pairing and filter |
| `w/reviewCommentContext.ts` | 436–506 | range labels, comment record |
| `w/pierre-icons.ts`, `w/pierre-tree-theme.ts` | 26–193, 194–287; 3–27 | T3 icon sprite and overrides, resolver, sprite injection; tree CSS overrides |
| `w/fileContextMenu.ts` | 59–87 | file context menu items |
| `w/index.css` | 945–950, 1279–1301, 1311–1316, 1803–1811, 2284–2289, 2301–2343 | inline subheader rule, change-icon tokens, copy button visibility, viewer fonts, diff tokens, blue-orange hooks, diff surface rules, find panel tokens |
| `c/ChatView.logic.ts`, `w/hooks/useProactivePanels.ts` | 202–231; 145–189 | proactive opening after a turn |
| `packages/shared/src/keybindings.ts` | 77 | `mod+d` → `diff.toggle` when `!terminalFocus` |
| `packages/shared/src/changedFileKind.ts` | 47–87 | which paths count as tests |
| `packages/contracts/src/settings.ts` | 69–71, 337, 404–406, 468–470, 521, 591 | `diffLayout`, `diffColorScheme`, `diffFilesCollapsed`, `diffIgnoreWhitespace`, `proactivePanelsEnabled`, `wordWrap` |
| `packages/contracts/src/checkpointDiff.ts`, `review.ts`, `orchestrationV2.ts` | 7–49; 6–95; 1443–1464, 3718–3719 | turn diff RPCs, Git diff preview RPCs, checkpoint summaries |
| `packages/client-runtime/src/state/review.ts`, `threadCheckpoints.ts` | 34–130; 12–91 | preview (tracked first), per-file patch reads (4 at once), totals |
| `w/timestampFormat.ts` | 3–21, 134–138, 262–285 | turn times, commit relative times |
| `patches/@pierre__diffs@1.7.0-beta.3.patch` | 15, 148, 384–397 | find reveals folded items, focus kept in the viewer, locale-grouped header counts |
| asset `getDiffFileInput-BJvrmjXZ.js` | `be` (`@layer base…` CSS); `ot`/`st`/`ct`; `hn`/`mn`; option defaults | Pierre viewer base CSS; file header builder; hunk separator builder; `collapsedContextThreshold` 1, `expansionLineCount` 100, `hunkSeparators` `line-info`, `overflow` `scroll` |
| asset `diffRendering-DmYelws_.js` | message table | Pierre strings ("# unmodified lines", "Expand all", find panel labels) |
| asset `worker-CPu5BpZb.js` | worker defaults | `lineDiffType: "word-alt"`, `maxLineDiffLength: 1000` |
| asset `FileExplorerPane-B9yFhWXT.js` | `@layer base` CSS | `@pierre/trees` base CSS |
| asset `pierre-icons-q-oSCvGy.js` | `Te`, `Ee`, `De`, `U` | built-in file-name and extension icon tables |
| asset `DiffCommentAnnotation-C54lJ88y.js` | search CSS; `ie`/`ae` | viewer find panel CSS; find/next/previous chord classifier |
| asset `PanelLayoutControls-t6T0i0K_.js` | `handleSearchKeyDown` | viewer key handling for find |
| asset `useNextRenderData-Cm3PGQUD.js` | interaction manager class (`placeUtility`, `placeUtilityFromSelection`, `setHoveredLine`, `handlePointerMove`) | where the gutter "+" sits; why no hover tint (`lineHoverHighlight` defaults to `"disabled"`) |
| asset `FileExplorerPane-B9yFhWXT.js` | `@pierre/trees` CSS (`spacing-item`, `:host(:hover)`), `ps` (active indent guides), tree keydown switch | indent guide opacity rules; tree keyboard |
| `c/ChatView.tsx`, `c/ChatView.logic.ts` | 2146–2177; 1076–1098 | window regains focus → composer refocus rule |
| `c/chat/useChatShortcuts.ts`, `c/chat/composerFocusRedirect.ts` | 56–92, 123–139; 3–94 | window-capture shortcut layer; printable keys redirected to the composer |
| `w/lib/previewFocus.ts`, `w/lib/workspaceFocus.ts` | 1–7; 274–301 | focus inside the right panel counts as `previewFocus`; focus recovery when a focused control unmounts |
| `w/appearanceFonts.ts` | 106–108 | `--font-size-code` and `--diffs-font-size` |

## 2. Structure

### 2.1 Where the changes surfaces appear

| Surface | Mount | Opens |
|---|---|---|
| Diff surface | right panel surface kind `diff`, a singleton tab "Diff" (`file-diff` 16). `DiffPanel` lazy-loaded with `Suspense fallback={null}`, `key` = scoped thread key, `mode="embedded"` (RightPanelSurfaceContent.tsx:31, 108–119). It stays mounted while another tab is active (`visible=false`) | see §5.1 |
| Changed files card | below an assistant message's markdown, `mt-4`, when that run has a checkpoint summary with at least one file and the timeline can open diffs (not inside a side chat) (TimelineAssistantRows.tsx:72–79, 366–390) | always shown when it has files |
| Thread details "Changes" row | [05-header.md](05-header.md) §3.8 | opens the Diff surface on the branch scope |
| File-change tool row | expanded tool row inspector, `Button xs outline` "Open diff" ([06-timeline.md](06-timeline.md) §3.7) | opens the Diff surface on the tool-call scope (V2ItemInspector.tsx:292–304) |

The shell has four modes (`inline`, `sheet`, `sidebar`, `embedded`, DiffPanelShell.tsx:8) but this build only passes `embedded`. So the Electron drag-region header variant (DiffPanelShell.tsx:11–16, listed in 03-shell §5.8) is never reached in this build.

### 2.2 Diff surface, element tree

```
[data-right-panel-surface-content]                       (right panel, 03-shell §3.7)
└─ div  flex h-full min-w-0 flex-col bg-background w-full           DiffPanelShell (embedded)
   ├─ div[data-surface-subheader]  flex items-center justify-between gap-2 px-2
   │     flex h-10 min-h-10 shrink-0 items-center border-b border-border/60 bg-background
   │     in-data-[preview-panel-mode=inline]:mb-3 :h-7 :min-h-7 :border-b-transparent      (DiffPanelShell.tsx:13–17)
   │  ├─ div  @container/diff-refs flex min-w-0 flex-1 items-center gap-3 [-webkit-app-region:no-drag]   left group
   │  │  ├─ Menu trigger = Button xs secondary  max-w-full [min-w-0 shrink when a commit is shown]
   │  │  │     aria-label "Diff scope: <label>"  data-workspace-focus-default
   │  │  │     [span.truncate <label>] [chevron-down 14, shrink-0 opacity-70]
   │  │  ├─ (commit scope) a | span  shrink-0 font-mono text-xs text-muted-foreground [underline underline-offset-4]  <sha 7>
   │  │  ├─ Menu popup (align start)  → §2.3
   │  │  └─ (branch scope, file access) div  flex min-w-0 max-w-full items-center gap-2 overflow-hidden text-xs text-muted-foreground
   │  │        aria-label "Comparing <head> against <base>"
   │  │        ├─ span  flex min-w-[min(3rem,calc(100%-2.75rem))] items-center gap-2 overflow-hidden @max-[20rem]/diff-refs:hidden
   │  │        │     [span.min-w-0.max-w-48.truncate <head>] [arrow-right 14 shrink-0 opacity-70]   tooltip "<head> → <base>"
   │  │        ├─ span[aria-hidden]  hidden @max-[20rem]/diff-refs:inline  "vs"                    tooltip "<head> → <base>"
   │  │        └─ Combobox (base ref) → §2.3
   │  └─ div  flex shrink-0 items-center gap-1 [-webkit-app-region:no-drag]  [gap-0 when a commit is shown]   toolbar
   │     ├─ DiffStatLabel layout=inline  mr-1 text-2xs                (when there are files)
   │     ├─ Button icon-sm ghost  "Refresh diff" | "Refreshing diff"  [RefreshIcon sm]        (Git scopes only)
   │     ├─ Button icon-sm ghost  "Collapse all files" | "Expand all files"  [chevrons-down-up | chevrons-up-down 14]   (when files)
   │     ├─ ToggleGroup segmented "Diff layout"  [Toggle "Stacked diff view" rows-3 14][Toggle "Split diff view" columns-2 14]   (no tooltips)
   │     ├─ Toggle ghost sm  "Disable diff line wrapping" | "Enable diff line wrapping"  [text-wrap 14]
   │     │     tooltip "Disable line wrapping" | "Enable line wrapping" (shorter than the aria-label)
   │     ├─ Toggle ghost sm  "Hide whitespace changes" | "Show whitespace changes"  [pilcrow 14]
   │     ├─ Toggle ghost sm  "Hide test files"  [flask-conical-off 14]                          (when test files exist)
   │     └─ Toggle ghost sm  "Show file tree" | "Hide file tree"  [folder-tree 14]               (when files)
   └─ body (one of)
      ├─ div  flex flex-1 items-center justify-center px-5 text-center text-xs text-muted-foreground/70   panel empty state
      ├─ DiffPanelLoadingState "Checking file access..."
      └─ div  flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background
         ├─ p (0..3 notices)  shrink-0 border-b border-border/70 bg-muted/40 px-3 py-1.5 text-2xs text-muted-foreground
         ├─ div.px-3 > p.mb-2.text-2xs.text-error/80                         load error, when nothing renders
         └─ one of
            ├─ div  min-h-0 flex-1 space-y-3 overflow-auto p-3 text-xs        recorded tool content without a diff
            ├─ DiffPanelLoadingState <label>                                    first load
            ├─ div  flex h-full items-center justify-center px-3 py-2 text-xs text-muted-foreground/70 > p   no changes
            ├─ div  flex min-h-0 flex-1 overflow-hidden                         files
            │  ├─ div  min-h-0 min-w-0 flex-1  (click and context-menu capture, §5.7)
            │  │  └─ div.diff-render-surface [--code-background:var(--background)] outline-none h-full min-h-0 overflow-auto
            │  │        tabindex=-1, style "overflow-anchor: none"      Pierre CodeView root (scroll container)
            │  │     ├─ div  style "contain: layout style; height: <total>px"         virtual content
            │  │     │  ├─ div  style "contain: size layout; height: <offset>px"       spacer above the window
            │  │     │  └─ div  style "position: sticky; width: 100%; contain: inline-size layout style; isolation: isolate; display: flex"
            │  │     │     └─ diffs-container ×N  (one per rendered file, open shadow root) → §2.4
            │  │     ├─ div[data-diffs-code-view-footer]  style "display: flow-root"  → DiffFileLoadingBoundary (lazy sources)
            │  │     └─ div[data-search-panel]  (shadow root, only while find is open) → §2.11
            │  └─ FileExplorerPane (when the tree is open) → §2.7
            └─ div  min-h-0 flex-1 overflow-auto p-2 > div.space-y-2                raw patch (unparseable)
               ├─ p  text-2xs text-muted-foreground/75   <reason>
               └─ pre  max-h-[72vh] rounded-md border border-border/70 bg-background/70 p-3 font-mono text-2xs leading-relaxed text-muted-foreground/90 [whitespace-pre-wrap wrap-break-word when wrapping]
```

The code view is virtualized per file (Pierre CodeView): only files near the viewport are mounted as `diffs-container`; a collapsed file is exactly its 32px header. Item metrics are fixed by T3 so the virtualizer and the painted rows agree: `diffHeaderHeight 32`, `hunkSeparatorHeight 24`, `spacing 0`, `paddingTop 0`, `paddingBottom 8` (Pierre's 8px gap fallback), outer layout padding and gap 0 (StyledDiffCodeView.tsx:253–264).

### 2.3 Scope menu, commits and base-ref picker

```
Menu popup (02-primitives §2.4; align start, side bottom; dropdown-glass)
├─ RadioGroup value=<scope value>
│  ├─ RadioItem "Changes"      value branch     closeOnClick
│  ├─ RadioItem "Uncommitted"  value unstaged   closeOnClick
│  └─ RadioItem "Latest turn"  value latest     closeOnClick
├─ SubTrigger "Turn"  [chevron-right 16, menu primitive]
│  └─ SubContent > RadioGroup value turn:<runId>
│        RadioItem ×n  closeOnClick:  span.flex.items-center.gap-2 [span "Turn <N>"] [span.ml-auto.text-xs.tabular-nums.text-muted-foreground <time>]
└─ SubTrigger "Commits"   (opening it snapshots the current list and refreshes it in the background)
   └─ SubContent  max-h-80 max-w-96 > RadioGroup value commit:<sha>
         per commit: div  flex items-center rounded-sm hover:bg-accent has-data-highlighted:bg-accent has-focus-visible:bg-accent
                     [&_[data-checked]]:bg-transparent [&_[data-highlighted]]:bg-transparent  [+ bg-foreground/[0.08] when selected]
           ├─ RadioItem  min-w-0 flex-1  closeOnClick  (Tooltip side right, max-w-80: the subject)
           │     span.flex.items-center.gap-3 [span.min-w-0.flex-1.truncate <subject>] [span.shrink-0.text-xs.tabular-nums.text-muted-foreground <relative time>]
           └─ MenuItem shrink-0 rendered as <a target=_blank> aria-label "Open commit <sha7>"
                 span.font-mono.text-xs.text-muted-foreground.underline.underline-offset-4 <sha7>
              | span.shrink-0.px-2.font-mono.text-xs.text-muted-foreground <sha7>     (no code-host URL)
         then at most one disabled MenuItem: <error> | "Checking file access..." | <access error> | "This connection cannot read local diffs."
                                              | "Loading commits..." | "No commits on this branch." | "Showing latest <n> commits"
```

Base-ref combobox (DiffPanel.tsx:1035–1131; Combobox primitive 02-primitives §2.12):

```
ComboboxTrigger = Button ghost-muted xs  min-w-9 max-w-48 shrink-0   aria-label "Change comparison target. Currently <base>"
   [span.min-w-0.truncate <base | "Automatic">] [chevron-down 14 shrink-0 opacity-70]
ComboboxPopup align start  w-72 min-w-0 max-w-[calc(100vw-1rem)] overflow-hidden
├─ ComboboxSearchInput placeholder "Search refs..."
├─ div  grid shrink-0 grid-cols-[1rem_minmax(0,1fr)] items-center gap-2 border-b border-border/70 ps-3 pe-6.5 pt-2 pb-1.5
│        font-medium text-3xs text-muted-foreground uppercase tracking-wide
│     [span aria-hidden][div grid min-w-0 grid-cols-[minmax(0,1fr)_2rem] items-center: "Branch" | "Remote" (text-right)]
├─ ComboboxEmpty "No matching refs."
└─ ComboboxList  max-h-64 min-w-0 overflow-x-hidden
   ├─ ComboboxItem  w-full min-w-0 grid-cols-[1rem_minmax(0,1fr)]  value __automatic_base_ref__  "Automatic"  (hidden while a query is typed)
   └─ per choice: ComboboxItem  same classes
        div grid w-full min-w-0 grid-cols-[minmax(0,1fr)_2rem] items-center overflow-hidden
        ├─ span.block.min-w-0.truncate.pe-2 <local name, or remote name for remote-only>
        └─ local+remote: Switch [--thumb-size:--spacing(3)] aria-label "Use remote version of <name>" (clicks do not reach the item)
           remote only:  span.flex.justify-end.text-muted-foreground > check 12 role=img aria-label "Remote only" (tooltip "Remote only")
```

### 2.4 One file in the code view (Pierre `diffs-container` shadow tree)

T3 renders through `@pierre/diffs` 1.7.0-beta.3 (patched). Each file is a `diffs-container` custom element whose shadow root holds Pierre's CSS layers (`base`, `theme`, `rendered`, `unsafe`) plus T3's `unsafeCSS` (StyledDiffCodeView.tsx:15–215 and diffRendering.ts:269–352). Light-DOM children are slotted into the header.

```
diffs-container  (light DOM; T3 sets bg/text to --code-background/--code-foreground, border 0, overflow clip: index.css:2316–2343)
│  slotted: div[slot=header-prefix] > DiffFileCollapseToggle      div[slot=header-filename-suffix] > DiffFileHeaderSuffix
│           (lazy sources) div[slot=<metadata>] > DiffStatLabel; annotations: div[slot=annotation-<side>-<line>] > comment UI
└─ #shadow-root
   ├─ div[data-diffs-header=default][data-change-type=<change|new|deleted|rename-pure|rename-changed>][data-sticky]   32px, sticky top 0
   │  ├─ div[data-header-content]
   │  │  ├─ slot (header prefix: collapse chevron)
   │  │  ├─ svg[data-change-icon=<type>] 16×16 (Pierre sprite: modified / added / deleted / moved)
   │  │  ├─ [div[data-prev-name] > bdi <old path>]  [svg[data-rename-icon] diffs-icon-arrow-right-short]     renames only
   │  │  ├─ div[data-title] > bdi <path>
   │  │  └─ slot (filename suffix: copy path, status)
   │  └─ div[data-metadata]
   │     ├─ span[data-deletions-count] "-<n>"   when deletions > 0 or additions = 0
   │     ├─ span[data-additions-count] "+<n>"   when additions > 0 or deletions = 0
   │     │     (no counts at all for an added file with no characters; counts locale-grouped, patch :384–397)
   │     └─ slot (metadata: DiffStatLabel for lazy sources, whose own counts are then hidden)
   └─ pre[data-diff][data-diff-type=single|split][data-overflow=wrap|scroll][data-indicators=bars][data-background]
        [data-custom-styles][data-interactive-line-numbers]   style "--diffs-min-number-column-width-default: <digits>ch"
      └─ code[data-code][data-unified][data-container-size]   grid: <number column> <content>   (unified; style "grid-row: span <rows>")
         ├─ div[data-gutter]   (subgrid; sticky left 0 in scroll mode; z 3)
         │  ├─ div[data-separator=line-info][data-separator-first?][data-separator-last?][data-expand-index?]  24px
         │  │  └─ div[data-separator-wrapper]  (absolute, full width)
         │  │     ├─ [div[data-expand-button] role=button [data-expand-up|down|both] > svg diffs-icon-expand | diffs-icon-expand-all]  (Git scopes)
         │  │     └─ div[data-separator-content] > span[data-unmodified-lines] "<n> unmodified lines"
         │  ├─ div[data-column-number=<n>][data-line-type=context|change-addition|change-deletion][data-line-index=<a>,<b>]
         │  │  ├─ span[data-line-number-content] <n>
         │  │  └─ div[data-gutter-utility-slot] > button[data-utility-button] > svg diffs-icon-plus 16
         │  │        (one per viewer, moved between rows: the bottom row of the selection, else the mouse-hovered row; §3.8)
         │  ├─ div[data-gutter-buffer=annotation]       (rows that hold an annotation)
         │  └─ div[data-gutter-buffer=buffer]            (split view, the missing side)
         └─ div[data-content]
            ├─ div[data-separator=…]  (same row; its wrapper is hidden on this side)
            ├─ div[data-line=<n>][data-alt-line?][data-line-type=…][data-line-index=…] > span[data-char=<col>][style="color:<token>"] …
            │     word-level change runs carry [data-diff-span]
            ├─ div[data-line-annotation=<i>,<line>] > div[data-annotation-content] > slot   (comment UI)
            ├─ div[data-no-newline] > span "No newline at end of file"
            └─ div[data-content-buffer]              (split view, hatched empty side)
```

### 2.5 Split view (S: the toggle writes a setting and was not pressed)

With `diffLayout = "split"` the same file uses `data-diff-type="split"` and two code columns: deletions (old file) left, additions (new file) right. Each change block pairs deleted and added lines row by row; context rows appear on both sides with their own numbers; a side with no counterpart row is filled by `[data-content-buffer]` (hatched) and `[data-gutter-buffer=buffer]`.

| Overflow | Layout (Pierre base CSS, StyledDiffCodeView.tsx:19–21) |
|---|---|
| wrap (default) | `pre` is one grid `repeat(2, var(--diffs-code-grid))`, T3 forcing `--diffs-code-grid: var(--diffs-grid-number-column-width) minmax(0, 1fr)` so both code columns stay equal; `code` elements are `display: contents`; deletions gutter column 1, deletions content column 2 with `border-right: 1px solid var(--diffs-bg)`; additions gutter column 3 with `border-left: 1px solid var(--diffs-bg)`, content column 4; `padding-block: 8px` (top 0 under a header) |
| scroll | `pre` is a grid `1fr 1fr`; each side is its own `code[data-code]` with its own horizontal scroll and sticky gutter; `[data-deletions]` has `border-right: 1px solid var(--diffs-bg)`, `[data-additions]` `border-left: 1px solid var(--diffs-bg)` |
| hunk separators | the deletions side draws the separator: expand buttons, then the pill, left corners rounded 6. The additions side draws only a plain bar in `--diffs-bg-separator` with its right corners rounded 6 and no text or buttons: in the additions gutter, sized `100cqi − 8px` in scroll mode, and in wrap mode also across the additions content cell with an 8px right margin |
| empty side | `[data-content-buffer]`: `background-image: repeating-linear-gradient(-45deg, transparent, transparent 4.24px, var(--diffs-bg-buffer) 4.24px, var(--diffs-bg-buffer) 5.66px)`, `background-size: 8px 8px`, `background-position: 5px 0`, `background-origin: border-box`, spanning the missing rows (`grid-row: span n; min-height: n × 1lh`); `[data-gutter-buffer=buffer]` paints `--diffs-bg-context-gutter` |

### 2.6 Review comments in the diff

`renderAnnotation` (AnnotatableCodeView.tsx:293–321) mounts the comment UI into the line annotation row after the line where the range ends, on the deletions side when the range ends on a deleted line, else on the additions side (41–43). Several entries on the same row share one annotation.

```
div  py-1                                                         (row holds the draft)
   | divide-y divide-border/30 border-y border-border/30          (row holds saved comments only)
└─ DiffCommentAnnotation ×n
   draft:  div[data-diff-comment-annotation]  px-3 py-2 font-sans text-foreground caret-foreground selection:bg-primary selection:text-primary-foreground
           contentEditable=false, pointerdown does not reach the viewer
           ├─ PullRequestMarkdownTextarea size sm  placeholder "Add a comment…"  aria-label "Comment on lines <range>"  autoFocus
           └─ div  mt-1.5 flex flex-wrap items-center gap-1
              ├─ span  mr-auto text-3xs text-muted-foreground/70  "⌘/Ctrl Enter to send"
              ├─ [TextFieldSpeechControls]   (dictation, when the environment supports it; not shown on the measured host)
              ├─ Button ghost-muted xs "Cancel"
              └─ Button xs (default) "Comment"   disabled while empty, pending or dictation is preparing
   saved:  div[data-diff-comment-annotation].group/comment  flex min-w-0 items-start gap-2.5 border-s-2 border-primary/55
           bg-primary/[0.045] px-3 py-2.5 font-sans text-foreground selection:bg-primary selection:text-primary-foreground
           ├─ message-circle 14 (size-3.5 h-5) shrink-0 text-primary/70
           ├─ p  min-w-0 flex-1 whitespace-pre-wrap text-sm leading-5  dir=auto-detected   <text>
           └─ span  -my-1 -mr-1 flex shrink-0 opacity-0 transition-opacity group-hover/comment:opacity-100 focus-within:opacity-100 max-sm:opacity-100
              └─ Button ghost-muted icon-xs "Delete comment" > trash-2 12
```

While a draft exists the gutter utility and line selection are disabled (AnnotatableCodeView.tsx:280–281).

### 2.7 File tree pane

```
aside[aria-label="File explorer"]  relative flex min-h-0 min-w-0 shrink-0 bg-background border-l border-border/60
    style "--resizable-width: <w>px; width: var(--resizable-width)"            (FileExplorerPane, resizable)
├─ RightPanelResizeHandle  label "Resize Changes files"  tabIndex 0  role=separator aria-valuenow/min/max  (03-shell §3.7 styles)
└─ div  flex min-h-0 flex-1 flex-col bg-background                                  DiffFileTree
   ├─ div[data-surface-subheader]  flex h-10 min-h-10 shrink-0 items-center gap-1 border-b border-border/60 bg-background px-2
   │     text-xs text-muted-foreground  (inline right panel: h-7 mb-3 transparent border)
   │  ├─ span  px-1 font-medium text-foreground  "Files"
   │  ├─ span  ml-auto tabular-nums  <file count>
   │  └─ Button icon-xs ghost  "Collapse all folders" | "Expand all folders"  [chevrons-down-up | chevrons-up-down 14]  (when folders exist)
   └─ file-tree-container[aria-label="<section title> files"][data-file-tree-virtualized]  min-h-0 flex-1 overflow-hidden
         style "color-scheme: <light|dark>; --trees-fg-override: var(--contrast-foreground)"
      └─ #shadow-root  (@pierre/trees 1.0.0-beta.6, density compact, search off, flattened empty directories, initially open)
         └─ div[data-file-tree-virtualized-wrapper][data-file-tree-colored-icons]
            └─ div[role=tree][data-file-tree-virtualized-root][data-file-tree-has-git-lane]
               └─ div[data-file-tree-virtualized-scroll] > div[data-file-tree-virtualized-list] > div[data-file-tree-virtualized-sticky]
                  └─ button[data-type=item][role=treeitem][data-item-type=file|folder][data-item-path][aria-level][aria-selected]
                        [data-item-git-status=<added|deleted|modified|renamed>] | [data-item-contains-git-change]
                     ├─ div[data-item-section=spacing] > div[data-item-section=spacing-item] ×depth      indent guides
                     ├─ div[data-item-section=icon] > svg (file icon, or file-tree-icon-chevron for folders)
                     ├─ div[data-item-section=content]  name (middle truncation), or flattened folders "a / b / c"
                     ├─ div[data-item-section=decoration]
                     └─ div[data-item-section=git] > span[title="Git status: <status>"] "M"|"A"|"D"|"R"
                           | dot with title "Contains git status items" (folders holding a change)
```

### 2.8 Changed files card (timeline)

```
div[data-changed-files-state=tree]  @container/changed-files mt-4 rounded-lg bg-secondary dark:bg-input/20
├─ div[data-changed-files-header]  sticky top-2 z-10 flex items-center justify-between gap-2 rounded-t-lg bg-secondary px-3 py-2
│     dark:bg-background dark:bg-linear-to-b dark:from-input/20 dark:to-input/20
│  ├─ div  flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs font-medium text-foreground
│  │  ├─ span  "<count> changed file[s]"   (count with toLocaleString)
│  │  └─ DiffStatLabel layout=inline  text-xs leading-4     (when additions or deletions > 0)
│  └─ div  flex shrink-0 items-center gap-1
│     ├─ Button icon-xs ghost-muted  "Collapse all folders" | "Expand all folders"  [chevrons-down-up | chevrons-up-down 12]
│     │     data-scroll-anchor-ignore   (only when a path has a folder)
│     └─ Button xs ghost-muted  aria-label "Open diff"  [file-diff 12][span hidden @[24rem]/changed-files:inline "Open diff"]
│           tooltip "Open the full diff"
├─ ChangedFilesTree  (key run:<expanded>)
│  └─ div.p-2 > div.relative  [style height = rows × 28 when virtualized]
│     └─ div[data-changed-file-row=<path>]  [absolute inset-x-0, top = index × 28 when virtualized]
│        ├─ folder: button  group flex h-7 w-full cursor-pointer items-center gap-2 rounded-md py-1.5 pr-2 text-left transition-colors
│        │     hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring
│        │     focus-visible:ring-offset-1 focus-visible:ring-offset-background   style padding-left = 8 + 14 × depth
│        │     aria-expanded  data-scroll-anchor-ignore
│        │  ├─ chevron-right 14  shrink-0 text-muted-foreground/70 transition-transform group-hover:text-foreground/80 [rotate-90 open]
│        │  ├─ folder (open) | folder-closed 14  shrink-0 text-muted-foreground/75
│        │  ├─ span  truncate font-mono text-2xs text-muted-foreground/90 group-hover:text-foreground/90   <name, compacted "a/b/c">
│        │  └─ span  ml-auto shrink-0 font-mono text-3xs tabular-nums > DiffStatLabel (aligned)   (when the folder's stat is non-zero)
│        └─ file: button  same classes (no aria-expanded)   onContextMenu → file context menu
│           ├─ span aria-hidden size-3.5 shrink-0          (spacer, when the tree has any folder or depth > 0)
│           ├─ PierreEntryIcon  size-3.5 text-muted-foreground/70   (sprite icon in its token colour)
│           ├─ span  flex min-w-0 font-mono text-xs text-foreground/85 group-hover:text-foreground > MiddleTruncate <file name>
│           └─ span  ml-auto shrink-0 font-mono text-3xs tabular-nums > DiffStatLabel (aligned)   (when the file has stats)
└─ (more files than listed) div.px-3.pb-2 > Button xs ghost-muted  "<n> more file[s] in the full diff"
```

### 2.9 DiffStatLabel

`span[role=group][aria-label="<a> additions, <d> deletions"]` with two `span[aria-hidden].font-mono` children, `text-diff-addition-foreground` "+<a>" and `text-diff-deletion-foreground` "-<d>", both always shown (DiffStatLabel.tsx:22–48). Layout `aligned` (default): `inline-grid grid-cols-[4ch_4ch] gap-2 text-right tabular-nums align-middle`; layout `inline`: `inline-flex items-center gap-1 tabular-nums align-middle`. Numbers use `formatCompactDiffCount` (§5.11). Used in: the Diff toolbar (inline, `text-2xs`), the changed files card header (inline, `text-xs leading-4`), card rows (aligned, `text-3xs` from the wrapper), the per-file header for lazy Git sources (aligned, 12px from the header), and the thread details card (05-header).

### 2.10 Skeletons

`DiffPanelLoadingState` (DiffPanelShell.tsx:92–116), `div[role=status][aria-live=polite][aria-label=<label>]  min-h-0 flex-1 overflow-hidden bg-background`:

```
DiffFileHeaderSkeleton (medium)      div flex h-8 items-center gap-2 px-2 pr-3
│  [div flex size-5 shrink-0 items-center justify-center > Skeleton size-2.5]  [Skeleton size-5 shrink-0]
│  [Skeleton pill h-3 w-1/2 max-w-64]  [div ml-auto flex shrink-0 items-center gap-2 > Skeleton pill h-3 w-5 ×2]
div flex h-6 items-center gap-2 px-2 pr-3:  [div h-px flex-1 bg-border/40] [Skeleton pill h-2.5 w-24] [div h-px flex-1 bg-border/40]
div space-y-2 px-3 py-2:  3 × [div flex items-center gap-3 > Skeleton pill h-2.5 w-5 shrink-0 + Skeleton pill h-2.5 w-2/3 | w-4/5 | w-3/5]
DiffFileHeaderSkeleton (short: w-2/5 max-w-52)
DiffFileHeaderSkeleton (long: w-3/5 max-w-72)
span.sr-only <label>
```

`DiffFileLoadingBoundary` (lazy Git sources, in the code view footer): `div[role=status][aria-label="Loading diff…"]` with `min(remaining, 4)` × `div.border-b.border-border/40 > DiffFileHeaderSkeleton medium`.

### 2.11 Find panel (Pierre)

Opened from the keyboard while the code view has focus (§5.12). A sticky overlay host `div[data-search-panel][data-search-panel-overlay]` with a shadow root sits as a sibling of the virtual content; T3 feeds it popover tokens and a 40px top inset that clears the sticky file header (index.css:2324–2332). The panel is an inline-size container (`container: search-panel`) and switches layout at 400px (asset `DiffCommentAnnotation-C54lJ88y.js`, search CSS): from 400px one row, below it (for example with the file tree open) two rows and 10px padding.

```
div[data-search-panel]  (column, right-aligned)
└─ div[data-editor-widget]  min-width 260, max-width 100%, radius 9, padding 4 (10 below 400px), gap 8
   ├─ div[data-search-grid][data-mode=find]   columns auto×4, gap 4px 6px
   │     ≥ 400px: areas "toggle find matches nav"            (input box 200 wide)
   │     < 400px: areas "toggle find find find" / ". nav matches matches"  (columns auto auto 1fr auto, input box auto)
   │  ├─ button[data-replace-toggle] "Toggle Replace"   (hidden in find mode)
   │  ├─ div[data-input-box][data-find] > input[data-search] placeholder "Search"
   │  │     + div[data-search-toggles] > button "Match Case" | "Whole Word" | "Regexp"  (20×20 each, 12px icons)
   │  ├─ div[data-matches][data-no-matches?] "No results" | "<n> result[s]" | "<i> of <n>"
   │  ├─ div[data-search-nav] > button "Previous" | "Next"   (24×24; disabled without matches)
   │  └─ [data-replace-cell] with "Replace" / "Replace All"   (display: none in find mode)
   └─ button[data-search-close] "Close"   20×20 circle at the widget's top-right corner (translate 10px, -10px)
```

All icon buttons share one rule (`[data-search-icon]`): colour `mix(lab, fg 65%, bg)`, hover full fg, `aria-pressed=true` (an active Match Case / Whole Word / Regexp) in the modified colour, which T3 sets to `--primary`, focus-visible 1px outline in the same colour, disabled opacity .25; colour and background transition 100ms ease-in-out. The close button inverts on hover (bg the widget fg, glyph the widget bg). Colours are in §3.13.

### 2.12 Icons

| Where | Icon | Size | Colour |
|---|---|---|---|
| scope trigger, base-ref trigger | lucide `chevron-down` | 14 | current, opacity .7 |
| head → base | lucide `arrow-right` | 14 | muted, opacity .7 |
| remote-only ref | lucide `check` | 12 | muted |
| refresh | lucide `refresh-cw` (RefreshIcon `sm`) | 14 | muted (`--control-icon-color`) |
| collapse/expand all files | lucide `chevrons-down-up` / `chevrons-up-down` | 14 | muted |
| layout | lucide `rows-3` (stacked) / `columns-2` (split) | 14 | toggle colour, opacity .8 |
| wrap, whitespace, tests, tree | lucide `text-wrap`, `pilcrow`, `flask-conical-off`, `folder-tree` | 14 | toggle colour, opacity .8 |
| menu sub-trigger | lucide `chevron-right` (menu primitive) | 16 | `#71717b` / `#a1a1a1`, opacity .8 |
| file collapse toggle | lucide `chevron-down` (open) / `chevron-right` (collapsed) | 16 | `text-muted-foreground/80` |
| copy path | lucide `copy`, copied `check` | 12 | muted; copied `text-success` |
| per-file status | lucide `rotate-cw` (error) / `info` (partial) | 12 | muted |
| saved comment | lucide `message-circle` | 14 (box 14×20) | `text-primary/70` |
| delete comment | lucide `trash-2` | 12 | muted |
| changed files card | lucide `chevrons-down-up` / `chevrons-up-down`, `file-diff` | 12 | muted |
| card folder rows | lucide `chevron-right`, `folder` / `folder-closed` | 14 | muted/70, muted/75 |
| card file rows | Pierre sprite file icon (`PierreEntryIcon`); fallback lucide `file` (`size-4 text-icon-muted`, which the card's `size-3.5 text-muted-foreground/70` overrides) | 14 | token colour (§3.11) |
| file header change | Pierre `diffs-icon-symbol-modified` (change), `-added` (new), `-deleted`, `-moved` (renames); `diffs-icon-file-code` for plain files | 16 | §3.4 |
| rename arrow | Pierre `diffs-icon-arrow-right-short` | 16 | current |
| hunk expand | Pierre `diffs-icon-expand` (single direction; flipped vertically for `data-expand-down`), `diffs-icon-expand-all` (both) | 16 | separator text colour |
| gutter comment button | Pierre `diffs-icon-plus` | 16 | `--diffs-bg` on `--diffs-modified-base` |
| tree folder | Pierre `file-tree-icon-chevron` (rotated −90° when collapsed) | 16 | `--trees-fg-muted` |
| tree folder change dot | Pierre `file-tree-icon-dot` | 6 viewBox | modified colour at 50% |
| tree files | Pierre `file-tree-builtin-<token>` or T3 `t3-file-icon-*` | 16 | token colour, or git-status colour when the icon has no token |
| find panel | Pierre sprite glyphs | 12 | §3.13 |

The Pierre and tree sprites are injected once into the document (`#t3code-pierre-file-icon-sprite`, pierre-icons.ts:270–287: Pierre's "complete" set, 58 symbols, plus T3's 23; 81 measured). Symbol ids: `file-tree-icon-{chevron,dot,file,lock,ellipsis}`, `file-tree-builtin-{bash,c,cpp,css,database,default,font,git,go,html,image,javascript,json,markdown,mcp,python,ruby,rust,swift,table,text,typescript,zip,astro,babel,biome,bootstrap,browserslist,bun,claude,docker,eslint,graphql,nextjs,npm,oxc,postcss,prettier,react,sass,stylelint,svelte,svg,svgo,tailwind,terraform,vite,vscode,vue,wasm,webpack,yml,zig}`, and `t3-file-icon-{video,agents,pnpm,diff,xml,pdf,word,spreadsheet,slides,epub,csharp,dart,elixir,haskell,java,kotlin,lua,php,powershell,r,scala,toml}` plus `t3-tree-icon-loading`. Built-ins use a 16×16 viewBox; T3's use 24×24 (lucide-derived outlines at stroke 2, or filled Simple Icons logos) or 32×32 (agents, pnpm).

## 3. Measurements

### 3.1 Diff surface frame and header row

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| shell | 901, 52, 539×848, `bg-background` | `#fcfcfc` | `#000000` | M |
| header row (inline right panel) | 901, 52, 539×28; `margin-bottom: 12px`; `padding: 0 8px`; `gap: 8px`; bottom border 1px transparent (index.css:945–950 and the `in-data-[preview-panel-mode=inline]` classes) | bg `#fcfcfc` | `#000000` | M |
| header row (sheet / not inline) | 40 tall, `padding: 0 8px`, no bottom margin, `border-bottom: 1px border/60`. Measured with the viewport resized live to 960 (right panel became a 402px sheet); the file tree's "Files" header is then 40 tall too, directly below | `#e4e4e799` | `#ffffff0c` (white 4.8%) | M |
| content starts | y 92 (52 + 28 + 12) | | | M |
| left group | x 909, 24 tall, `gap: 12px`, flex 1 | | | M |
| scope trigger | Button xs secondary: 24 tall, `padding: 0 7px` (8 − 1 border), radius 8, `gap: 4px`, 12/16 500. "Changes" 81 wide, "Turn 16" 71.9 | bg `--secondary` `#fafafa`, text `--secondary-foreground` `#27272a` | bg `#ffffff08`, text `#f5f5f5` | M |
| scope chevron | 14×14, opacity .7, `margin: 0 -2px` | `#27272a` | `#f5f5f5` | M |
| toolbar | right-aligned, `gap: 4px` (`gap-0` with a commit), 28 tall, y 51.5 (centred on the 28px row) | | | M |
| DiffStatLabel (toolbar) | inline, `gap: 4px`, `margin-right: 4px`, mono 11/16 400 tabular. "+95 -3" (ASCII hyphen-minus, as in every count here): 37 wide; compact counts apply ("+1.5k -0", 50.2 wide) | `#007a55` / `#c10007` | `#00d492` / `#ff6467` | M |
| refresh button | Button icon-sm ghost 28×28, radius 8; icon 14 | icon `#71717b` | `#a1a1a1` | M |
| collapse-all button | Button icon-sm ghost 28×28; icon 14 | icon `#71717b` | `#a1a1a1` | M |
| layout group | ToggleGroup segmented 70×28, `padding: 2px`, `gap: 2px`, radius 8, bg `bg-input/40` | `#d4d4d866` | `#ffffff0b` | M |
| layout toggles | 32×24, radius 6, `padding: 0 10px`; icon 14 at opacity .8 | pressed bg `#fcfcfc`, text `#18181b`, shadow `0 1px 2px #0000001a`; unpressed text `#71717b` | pressed bg `#ffffff14`, text `#f5f5f5`, same shadow; unpressed `#a1a1a1` | M |
| wrap / whitespace / tests / tree toggles | Toggle ghost sm 28×28, radius 8, `padding: 0 5px`; icon 14 opacity .8 | pressed bg `--accent` `#f4f4f5`, text `#18181b`; unpressed transparent, `#18181b` | pressed `#ffffff0a`, `#f5f5f5`; unpressed `#f5f5f5` | M |
| toolbar order and x (turn scope, 539 wide) | DiffStatLabel (its left edge moves with its width: 1188.97 for "+95 -3", 1175.77 for "+1.5k -0"); collapse-all 1234; layout 1266..1336; wrap 1340; whitespace 1372; tree 1404..1432 (8px right padding). A selection with no files keeps only layout, wrap and whitespace (plus refresh in Git scopes) | | | M |
| tooltips | Tooltip primitive, side top, 4px gap ("Collapse all files": 1193, 21, 110×26.5 above the button) | | | M |

### 3.2 Scope menu and submenus

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| popup | Menu primitive (`dropdown-glass`, radius 8, 1px border, `p-1`). Measured 909, 81.5, 160×150 (min width 160), 4px below the trigger | bg `#ffffffd5`, border `#18181b1a`, shadow `0 16px 40px -18px #0000008c`, blur 12 saturate 1.14 | bg `#111111d5`, border `#f5f5f51a`, shadow `0 18px 44px -18px #000000cc`, blur 16 saturate 1.08 | M |
| rows | 150×28, `padding: 4px 8px`, radius 4, 14/20 400 `text-foreground` | `#18181b` | `#f5f5f5` | M |
| checked radio row | no check mark; `data-checked:bg-foreground/[0.08]`. With the newest turn selected two rows are checked: "Latest turn" in the menu and that turn's "Turn N" row in the Turn submenu, whose group value is always `turn:<runId>` (DiffPanel.tsx:811–819, 889) | `#18181b14` (painted `#1a1a1a14`) | `#f5f5f514` (painted `#f2f2f214`) | M |
| highlighted / open sub-trigger | `--accent` | `#f4f4f5` | `#ffffff0a` | M |
| sub-trigger chevron | 16, `ms-auto`, opacity .8 | `#71717b` | `#a1a1a1` | M |
| submenu position | right of its trigger, overlapping the parent by 5px: parent 909..1069, sub at x 1064; top 5px above the trigger row (Turn trigger y 170.5 → sub y 165.5) | | | M |
| Turn rows | "Turn 22" … "Turn 11": label 14/20; time `text-xs tabular-nums text-muted-foreground` 12/16 right-aligned (`ml-auto`), 8px min gap; "2:00 PM" 48.3 wide | time `#71717b` | `#a1a1a1` | M |
| empty Turn submenu | 160×10 (border + `p-1`, no rows) | | | M |
| Commits submenu | `max-h-80` (320) `max-w-96` (384). With no file access: one disabled row "This connection cannot read local diffs.", 276.9×28, opacity .64, `cursor: not-allowed` | | | M |
| commit rows | subject truncates (`min-w-0 flex-1`), relative time 12/16 tabular muted, 12px gap; SHA cell mono 12/16 muted underlined, offset 4; selected row `bg-foreground/[0.08]` | | | S |
| base-ref popup | 288 wide (`w-72`); header row 10/14 500 uppercase `tracking-wide` (0.025em) muted, `padding: 8px 26px 6px 12px`, bottom border `border/70`; list `max-h-64` (256); switch thumb 12px | border `#e4e4e7b3` | `#ffffff0e` | S (colours P) |

### 3.3 Code view surface and file header

`--code-background` is `var(--background)` inside the Diff surface (StyledDiffCodeView.tsx:245–246), not the `:root` `--code-background` (`#ffffff` / `#101010`). `--code-foreground` is `--foreground`.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| surface | `.diff-render-surface`, x 901, y 92, 539×808 (323.4 wide with the tree at its default width); scrolls vertically; app scrollbar styling (01-tokens §3.7). Its own background is transparent: the colour comes from the body wrapper (`bg-background`) and each `diffs-container` | `#fcfcfc` | `#000000` | M |
| `diffs-container` | width of the surface (533 while its vertical scrollbar shows), `font-size 13px`, `line-height 20px`, `overflow: clip`, bg code-background, text code-foreground | `#fcfcfc` / `#18181b` | `#000000` / `#f5f5f5` | M |
| file header | 32 tall, `display: flex`, `justify-content: space-between`, `align-items: center`, `gap: 8px`, `padding: 0 12px 0 8px`, `font: 12px/1 var(--font-sans)`, `position: sticky; top: 0; z-index: 4`, bottom border transparent | bg `#fcfcfc` | `#000000` | M |
| header hover | bg unchanged; `box-shadow: inset 3px 0 <code-foreground 24%>` (a 3px left edge) | `#18181b3d` | `#f5f5f53d` | M |
| collapse toggle | slot box 18×20 at x 909; Button icon-micro ghost 20×20, radius 4, `margin-left: -2px`; chevron 16 | icon `#71717bcc`; hover bg `#f4f4f5` | icon `#a1a1a1cc`; hover `#ffffff0a` | M (dark hover S) |
| change icon | 16×16, x 935 (toggle + 8) | §3.4 | | M |
| title | x 959, 12/16 sans 400, `cursor: pointer`; clipped with a leading ellipsis (`direction: rtl`, so the end of the path stays visible: "…/lib/parser.ts") | `#18181b` | `#f5f5f5` | M |
| title hover | `color-mix(in srgb, code-foreground 84%, --primary)`; `transition: color 120ms ease` | `#182139` | `#d6dff4` | M |
| copy-path button | slot after the title + 8; Button icon-micro ghost-muted 20×20, radius 4; `copy` 12; `aria-label="Copy file path"`, tooltip "Copy path" (default side) then "Copied" (DiffFilePathCopyButton.tsx:24–37). Hidden (`opacity: 0`) on hover-capable pointers until the file (`diffs-container`) is hovered or the button holds focus (index.css:1311–1316) | icon `#71717b`; hover bg `#f4f4f5`, icon `#18181b` | `#a1a1a1`; hover `#ffffff0a`, `#f5f5f5` | M (dark hover P) |
| copied state | `check` 12 `text-success`; tooltip "Copied"; anchored copy toast (02-primitives §3.24) | `#00bc7d` | `#00bc7d` | S (colour P) |
| header counts | right edge at surface − 12; `[data-metadata]` `gap: 1ch` of 12px sans (7.57), tabular; counts mono 11/11 (`font-size 11px`, `line-height 1`); deletions first: "-1 +2" | `+` `#18a46c`, `-` `#d52c36` | `+` `#07c480`, `-` `#ff2e3f` | M |
| header counts with blue-orange | `--diffs-addition-color-override` / `--diffs-deletion-color-override` = `--diff-addition` / `--diff-deletion` | `#155dfc` / `#f54a00` | `#50a2ff` / `#ff8904` | P |
| lazy-source counts | Pierre counts hidden (`[data-additions-count], [data-deletions-count] { display: none }`), replaced by DiffStatLabel aligned (two 4ch columns at 12px mono, 8 gap, right-aligned) in `diff-*-foreground` | `#007a55` / `#c10007` | `#00d492` / `#ff6467` | S |
| renamed file | `[data-prev-name]` at opacity .7, then `diffs-icon-arrow-right-short` 16, then the new path; both names ellipsize from the start | | | S |

The header colours of the counts (and of line numbers and bars below) come from the syntax theme, not from T3's tokens: the highlighter theme sets Pierre's `--diffs-light-addition-color` etc. from its git decoration colours (`t3-light` keeps pierre-light's `#18a46c` / `#d52c36`; `pierre-dark` gives `#07c480` / `#ff2e3f`). Only the blue-orange setting overrides them.

### 3.4 Change icon colours

| Type | Light | Dark | Source | M/P |
|---|---|---|---|---|
| change, rename-pure, rename-changed | `--diff-modified-icon` `#425cb1` | `#50a2ffa6` (`--info-foreground` at 65%) | diffRendering.ts:344–351, index.css:1282–1288 | M |
| new | `--diff-added-icon` unset in light → `--diff-addition-foreground` `#007a55` | `--diff-added-icon` `#00d492` | diffRendering.ts:334–339 | M |
| deleted | `--diff-deletion-foreground` `#c10007` | Pierre deletion base `#ff2e3f` | diffRendering.ts:340–342 | P |
| plain file (`file`) | opacity .6 | | Pierre base CSS | S |
| blue-orange: change | `--muted-foreground` `#71717b` | `#a1a1a1a6` | index.css:1294–1301 | P |
| blue-orange: new | `#155dfc` | `#50a2ff` | `--diff-added-icon: initial` | P |
| blue-orange: deleted | `#f54a00` | `#ff8904` | | P |

### 3.5 Lines, gutter and indicators (red & green, the default)

Code font: `--font-mono` (`ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace`), 13px, `line-height: 20px`, `tab-size: 2`, colours from the syntax theme (`t3-light` / `pierre-dark`, 06-timeline §3.6). Plain code text is the theme foreground.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| `pre` / `code` | grid columns `<number column> <content>`: `41px 498px` for two-digit numbers in a 539 surface; `padding-top: 0` under the header; `padding-bottom: 2px` plus the 6px horizontal scrollbar gutter (8 below the last row) | bg code-background | | M |
| number column | per row `div[data-column-number]`: `padding-left: 2ch` (15.60), `padding-right: 1ch` (7.80), `border-right: 2px solid <code-background>`, right-aligned, content width = `<max digits>ch` (inline `--diffs-min-number-column-width-default: 2ch`; 1ch for single-digit files), `box-sizing: content-box`; `cursor: pointer` (interactive numbers) | | | M |
| number colour, context | `color-mix(in lab, theme-fg 65%, code-background)` | `#565656` | `#9b9b9b` | M |
| content row | `div[data-line]` `padding: 0 1ch` (7.80); context rows have no tint | text `#0a0a0a`, bg `#fcfcfc` | text `#fafafa`, bg `#000000` | M |
| addition row | content bg | `#ecf9f4` | `#02110b` | M |
| addition number | bg; text = addition base | `#eef9f5`; `#18a46c` | `#04110b`; `#07c480` | M |
| deletion row | content bg | `#fef0f0` | `#170705` | M |
| deletion number | bg; text = deletion base | `#fef1f0`; `#d52c36` | `#170806`; `#ff2e3f` | M |
| addition bar | number cell `::before`, absolute left 0, 4px wide, full row height, solid addition base | `#18a46c` | `#07c480` | M |
| deletion bar | same box, `linear-gradient(0deg, <bg-deletion> 50%, <deletion base> 50%)` at `background-size: 2px 2px` (`calc(1lh / round(1lh / 2px))`): a dashed bar, 1px dashes every 2px | `#fde6e2` / `#d52c36` | `#331512` / `#ff2e3f` | M |
| wrapped lines | `white-space: pre-wrap; word-break: break-word`; a 3-row wrapped line is 60 tall and its number cell stretches with it | | | M |
| scroll mode (wrap off) | `white-space: pre`, rows 20 tall, content column as wide as the longest line (990.86 measured), gutter `position: sticky; left: 0`, own horizontal scroll per file (6px scrollbar, thumb transparent, `--diffs-bg-context` while the file is hovered) | | | M |
| "No newline at end of file" | `[data-no-newline]` row, text at opacity .6, not selectable | | | S |
| line hover tint | none. Pierre sets `data-hovered` only when its `lineHoverHighlight` option is not `"disabled"`, which is the default, and T3 never passes that option (asset `useNextRenderData-Cm3PGQUD.js` `setHoveredLine`; no `lineHoverHighlight` anywhere in apps/web). Pierre's hover CSS (97% / 91% mix toward the hover target) is therefore dead in the Diff panel | | | M (absence) + S |

### 3.6 Hunk separators and expand controls

Hunk header lines (`@@ -a,b +c,d @@` and the function context after them) are never drawn. Each gap between hunks, and the gap above the first hunk, is a separator row that names the hidden line count instead (`hunkSeparators: "line-info"`, Pierre's default; the `metadata` type that would print the hunk spec is not used).

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| separator row | `[data-separator=line-info]` 24 tall (`height: 24px !important; margin-block: 0`), bg `--diffs-bg-separator` = `color-mix(in srgb, code-bg 97%, code-fg)` | `#f5f5f6` | `#070707` | M |
| wrapper | absolute, full surface width (`100cqi`), `padding: 0 8px`, bg code-background | `#fcfcfc` | `#000000` | M |
| content pill | flex 1 (523 wide in a 539 surface), radius 6, `padding: 0 1ch` (6.94 at 11px Inter), `font: 11px var(--font-sans)`, line height 20, no underline, ellipsis, not selectable | bg `#f5f5f6`, text `color-mix(in srgb, code-fg 50%, code-bg)` `#8a8a8c` | `#070707`, `#7a7a7a` | M |
| text | "<n> unmodified line[s]"; unknown count "More unchanged context may be available" | | | M / S |
| first separator | above the first hunk when it does not start at line 1 ("36 unmodified lines" above line 37) | | | M |
| expand button (Git scopes, file contents loadable) | own cell before the pill: 32 wide (34 on fine pointers), full row height, `border-right: 2px solid <bg>`, colour as the pill text; the pill's left corners square off | | | S |
| expand hover | button and pill: `color: code-fg`, bg `color-mix(in srgb, code-bg 92%, code-fg)`; Pierre's own `underline` on hover is disabled | `#eaeaea` (P) | `#141414` (P) | S |
| which buttons | middle gap ≤ 100 lines: one "both" button (`diffs-icon-expand-all`); gap > 100: an up and a down button, on fine pointers stacked in two half-height rows of one 34px cell (`grid-template-rows: 50% 50%`, 1px divider), on coarse pointers side by side (`grid-template-columns: 32px 32px auto`); before the first hunk: one `data-expand-down` (icon flipped vertically); after the last hunk: one `data-expand-up`. A hidden "Expand all" button exists but is `display: none` | | | S |
| expansion step | 100 lines per click (`expansionLineCount` default); gaps of 1 line are never collapsed (`collapsedContextThreshold` 1) | | | S |
| turn / tool-call diffs | partial patches without file contents: the pill only, no buttons, no trailing separator (measured) | | | M |

### 3.7 Word-level emphasis

T3 passes `lineDiffType: "none"` to the viewer (DiffPanel.tsx:1508), but highlighting runs in the worker pool, whose render options never set `lineDiffType`, so the worker's default `"word-alt"` applies (DiffWorkerPoolProvider.tsx:66–71; worker-CPu5BpZb.js). **Word-level spans render** (measured).

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| span | `[data-diff-span]` inline, `border-radius: 3px`, `box-decoration-break: clone` (each wrapped fragment rounds) | | | M |
| addition emphasis | Pierre default: addition base at 15% (light) / 20% (dark). T3's `--diffs-bg-addition-emphasis-override` is set on `[data-diff]`, below the `:host` where Pierre resolves `--diffs-bg-addition-emphasis`, so it has no effect | `#18a46c26` | `#07c48033` | M |
| deletion emphasis | same rule | `#d52c3626` | `#ff2e3f33` | M |
| text | unchanged token colours | | | M |
| limits | lines over 1,000 characters get no word diff (`maxLineDiffLength`); tokenizing stops at 1,000 characters per line (`tokenizeMaxLineLength`) | | | S |

### 3.8 Selection, gutter comment button and comments

| Item | Value | Light | Dark | M/P |
|---|---|---|---|---|
| selected content row | `--diffs-line-bg: light-dark(mix(lab, code-bg 88%, mix(srgb, code-bg 50%, modified)), mix(lab, code-bg 80%, mix(srgb, code-bg 70%, modified)))` (StyledDiffCodeView.tsx:34–53), modified base `#009fff` | `#eff7fd` | `#060e15` | M |
| selected number cell | `91% / 35%` and `85% / 60%` mixes (55–68) | `#f0f7fd` | `#080e15` | M |
| selected number text | `--diffs-selection-number-fg`: modified 65% + black / 75% + white in lab | `#20649e` | `#72b6ff` | M |
| selection bar | number cell `::before` 4px, `#009fff` (replaces the change bar on selected rows) | `#009fff` | `#009fff` | M |
| selected annotation row | the draft's row is part of the selection and takes the same colours: for a one-line draft the number cell stays `data-selected-line=single`, the line becomes `first`, the gutter buffer and the annotation `last` | `#eff7fd` / gutter `#f0f7fd` | `#060e15` / gutter `#080e15` | M |
| gutter comment button | one `button[data-utility-button]` per viewer, moved between rows: while lines are selected it sits on the **bottom row of the selection** and stays there whatever the pointer hovers; with no selection it follows the row under a **mouse** pointer (gutter or code; touch and pen moves are ignored) and disappears when the pointer leaves (asset `useNextRenderData-Cm3PGQUD.js` `placeUtility` / `placeUtilityFromSelection`, `handlePointerMove`). 20×20 (1lh), radius 4, bg `--diffs-modified-base`, icon `diffs-icon-plus` 16 in `--diffs-bg`; pinned to the number cell's right padding edge and overhanging into the code by `1lh − 1ch` (`margin-right: -12.2px`); top of the row; `z-index 4`; `cursor: pointer`. Removed while a draft is open | bg `#009fff`, icon `#fcfcfc` | bg `#009fff`, icon `#000000` | M |
| draft annotation | `px-3 py-2` (12/8), 13px base; width = content column | text `#18181b` | `#f5f5f5` | M |
| draft textarea | Textarea sm (02-primitives §3.10): wrapper as wide as the annotation minus `px-3` (474×68 in a 498 column, 252×68 in a 276 one); `min-height: 66px` (`min-h-16.5`, 78 below 640px), `field-sizing: content`, `max-h-64`; `padding: 3px 9px`; 14/20; radius 8. Unfocused: 1px `--input` border, `shadow-xs/5` (`0 1px 2px #0000000d`); focused: border `--primary` and `inset 0 0 0 3px primary/24`. Focus styling needs the window focused (a background window shows the unfocused look even though the textarea is the active element) | focused: bg `#fcfcfc`, border `#1b4ed8`, ring `#1b4ed83d` | unfocused: bg `#ffffff09`, border `#ffffff1c`; focused: border `#346bf1`, ring `#346bf13d` | M |
| draft placeholder | "Add a comment…", current colour at 50% | `#18181b80` | `#f5f5f580` | M |
| draft footer | `margin-top: 6px`, `gap: 4px`, 24 tall | | | M |
| hint | "⌘/Ctrl Enter to send", 10/14, `muted-foreground/70`, pushes the buttons right | `#71717bb3` | `#a1a1a1b3` | M |
| Cancel | Button ghost-muted xs, 55.8×24, `padding: 0 7px`, 12/16 500 | `#71717b` | `#a1a1a1` | M |
| Comment | Button xs default, 71.7×24, primary; disabled (opacity .64, no pointer events) while the text is empty | bg `#1b4ed8`, text `#ffffff` | `#346bf1`, `#ffffff` | M |
| saved comment | `border-s-2 border-primary/55`, bg `primary/[0.045]`, `padding: 10px 12px`, `gap: 10px`; icon box 14×20; text 14/20 | border `#1b4ed88c`, bg `#1b4ed80b`, icon `#1b4ed8b3` | `#346bf18c`, `#346bf10b`, `#346bf1b3` | P |
| saved comments group | `divide-y` and `border-y`, `border/30` | `#e4e4e74d` | `#ffffff06` | P |
| annotation background (no selection) | Pierre `--diffs-bg-context` (T3's override is ineffective at this level): `mix(lab, code-bg 98.5%, black)` / `mix(lab, code-bg 92.5%, white)`; gutter part `--diffs-bg-context-gutter` | `#f8f8f8`, gutter `#f8f8f8` | `#171717`, gutter `#0c0c0c` | P (formula) |

### 3.9 Blue & orange diff colours

Set by `:root[data-diff-color-scheme="blue-orange"]` (index.css:2301–2314, 1294–1301). Token values are in 01-tokens §3.2; the viewer colours below are computed with T3's formulas (§3.14). The default red-green values in §3.5 were reproduced by the same formulas to within 2 units per channel.

| Item | Light | Dark | M/P |
|---|---|---|---|
| addition base (numbers, bar, header `+`) | `#155dfc` | `#50a2ff` | P |
| deletion base (numbers, dashes, header `-`) | `#f54a00` | `#ff8904` | P |
| addition row / number bg | `#eff2fc` / `#f0f2fc` | `#0a0e15` / `#0a0e15` | P |
| deletion row / number bg | `#fdf1ec` / `#fef2ed` | `#170c01` / `#170d02` | P |
| deletion dash background | `#ffe8de` | `#321f0d` | P |
| word emphasis | `#155dfc26` / `#f54a0026` | `#50a2ff33` / `#ff890433` | P |
| DiffStatLabel and card counts (`--diff-*-foreground`) | `#155dfc` / `#f54a00` | `#50a2ff` / `#ff8904` | P |
| tree: added, untracked / deleted | `#155dfc` / `#f54a00` | `#50a2ff` / `#ff8904` | P |
| tree: modified, renamed | unchanged (`#1ca1c7`, `#d5a910`) | (`#08c0ef`, `#ffd452`) | P |

### 3.10 File tree pane

| Item | Value | Light | Dark | M/P |
|---|---|---|---|---|
| pane width | default `initial = max(160, min(256, 0.4 × row width))` = 215.6 in a 539 row; max `max(160, min(560, max(row − 256, initial)))` = 283 (the responsive default never shrinks to make room for the 256px content reserve, so between rows of 400 and ~427 the max is the default itself); min 160; re-measured on every row resize; a dragged width is persisted at `t3code:diff-file-tree-width` (FileExplorerPane.tsx:8–10, 45–63) | | | M |
| pane border | `border-l border-border/60` | `#e4e4e799` | `#ffffff0c` | M |
| resize handle | 03-shell §3.7 (4px hit area, 1px line on hover/focus), `aria-valuemin 160 / valuemax 283` | | | M |
| pane header | 28 tall inline (+12 margin), `padding: 0 8px`, `gap: 4px`, 12/16 muted; "Files" `padding: 0 4px` 500 foreground; count tabular; collapse-all Button icon-xs ghost 24×24, radius 8, icon 14 muted | "Files" `#18181b`, count `#71717b` | `#f5f5f5`, `#a1a1a1` | M |
| tree host | `color: --contrast-foreground` (`--trees-fg-override`), font `--font-sans` 12px (`0.75rem`), `line-height: 24px` rows; `--trees-bg` transparent | `#18181b` | `#f5f5f5` | M |
| scroll padding | `padding: 0 8.4px 0 14.4px` (16 − 1.6 margin; right also − 6 scrollbar gutter) | | | M |
| row | `button[data-type=item]`: 24 tall (compact density 0.8 × 30), `padding: 0 6.4px`, `margin: 0 1.6px`, `gap: 4.8px`, radius 6 (`--radius-md`) | | | M |
| row hover | Pierre `--trees-bg-muted`: accent `#009fff` at 8% / 10% over transparent (T3's `--trees-hover-bg-override` is not a variable this version reads) | `#009fff14` | `#009fff1a` | M |
| selected row | `--trees-selected-bg-override: color-mix(in srgb, currentColor 12%, transparent)`; adjacent selected rows join (inner corners square) | `#18181b1f` | `#f5f5f51f` | M |
| focus ring | `::before` inset 0, `outline: 1px solid #009fff`, `outline-offset: -1px`, radius 6; `z-index 2` | `#009fff` | `#009fff` | M |
| file name colour | git status colour on icon (when the icon has no token) and name: modified `#1ca1c7`, added / untracked `#16a994`, deleted `#ff2e3f`, renamed `#d5a910`, ignored `#adadb1` (icon at 50%) | as listed | `#08c0ef`, `#00cab1`, `#ff6762`, `#ffd452`, `#4a4a4e` | M (modified, added), S (others) |
| git lane | 12 wide at the row's right; letter 12px 600 in the status colour ("M", "A"…; `title="Git status: <status>"`) | | | M |
| folder | chevron `file-tree-icon-chevron` 16 in `--trees-fg-muted`, rotated −90° when collapsed; name in the tree foreground; lane shows a 6px dot at opacity .5 in the modified colour when the folder contains changes (`title="Contains git status items"`) | chevron `#84848a`, dot `#1ca1c7` | `#84848a`, dot `#08c0ef` | M |
| flattened folders | single-child chains render as one row "a / b / c" (`[data-item-flattened-subitems]` `gap: 2px`; each segment underlines on hover) | | | M |
| indent guides | `[data-item-section=spacing-item]` 1px left border `color-mix(in lab, fg-muted 25%, transparent)`; opacity 0, 0.75 while the tree is hovered (`:host(:hover)`), and **1 for the guides of the folder that holds the tree's focused row** (the roving `tabindex=0` row, by default the selected file): the tree injects `[data-item-section="spacing-item"][data-ancestor-path="<folder path>"] { opacity: 1 }` (asset `FileExplorerPane-B9yFhWXT.js`, `ps`). Measured: the revealed file's sibling guides at opacity 1 with and without hover. `transition: opacity .15s`; the indent step is 18.69 (7.5 + 6.39 + gap) | `#84848a40` | `#84848a40` | M |
| file icons | 16×16, token colours as the card (§3.11) except `package.json`-style overrides with no token, which take the status colour | e.g. git `#ff8c5b`, TypeScript `#1a85d4`, Markdown `#199f43` | `#d5512f`, `#69b1ff`, `#5ecc71` | M |
| truncation | names middle-truncate (`[data-truncate-group-container=middle]`): the name is split before its extension and the start segment is cut; the start of the name and its extension stay visible ("compo… json"); flattened folder segments each keep their start ("s. / com… / u."). T3 hides Pierre's fading truncation marker (`[data-truncate-marker] { display: none }`) and gives the visible segment a plain end ellipsis (`text-overflow: ellipsis`, pierre-tree-theme.ts:12–13), so the "…" is an ordinary ellipsis glyph | | | M |
| loading icon | `t3-tree-icon-loading` at opacity .6, spinning (§6) | | | S |

### 3.11 Changed files card

| Item | Value | Light | Dark | M/P |
|---|---|---|---|---|
| card | lane width (728 measured), `margin-top: 16px`, radius 8 | bg `--secondary` `#fafafa` | `bg-input/20` `#ffffff06` | M |
| header | 40 tall, `padding: 8px 12px`, `gap: 8px`, sticky `top: 8px`, z 10, top corners 8 | bg `#fafafa` | `#000000` under `linear-gradient(#ffffff06, #ffffff06)` (opaque while sticky) | M |
| title | 12/16 500 `text-foreground`; title-to-counts gap 12; wraps to a second line (`gap-y-1`) when narrow | `#18181b` | `#f5f5f5` | M |
| title counts | inline DiffStatLabel, mono 12/16, weight 500 (inherited from the title row's `font-medium`; the toolbar's and the rows' are 400), gap 4 ("+95 -3" 40 wide, "+1.5k -0" 54.4) | `#007a55` / `#c10007` | `#00d492` / `#ff6467` | M |
| header actions | gap 4; "Expand all folders" icon-xs ghost-muted 24×24 radius 8, icon 12; "Open diff" xs ghost-muted 81.2×24, `padding: 0 7px`, gap 4, 12/16 500, icon 12 (`margin-left: -2px`); the label hides below a 24rem (384px) card | `#71717b` | `#a1a1a1` | M |
| rows container | `padding: 8px` | | | M |
| row | 28 tall, radius 6, `padding: 6px 8px 6px <8 + 14 × depth>`, gap 8, `transition: color, background-color… 150ms cubic-bezier(0.4, 0, 0.2, 1)` | | | M |
| row hover | `bg-accent/60` | `#f4f4f599` (read `#f3f3f599`) | `#ffffff06` | M |
| row focus-visible | `box-shadow: inset 0 0 0 1px <background>, inset 0 0 0 3px <ring>` | `#fcfcfc`, `#1b4ed8` | `#000000`, `#346bf1` | M |
| folder chevron | 14, `muted-foreground/70`; hover `foreground/80`; open `rotate: 90deg`, `transition: transform 150ms` | `#71717bb3` → `#18181bcc` | `#a1a1a1b3` → `#f5f5f5cc` | M |
| folder icon | `folder-closed` / `folder` 14, `muted-foreground/75` | `#71717bbf` | `#a1a1a1bf` | M |
| folder name | mono 11/16 (`text-2xs`), `muted-foreground/90`; hover `foreground/90` | `#71717be6` → `#18181be6` | `#a1a1a1e6` → `#f5f5f5e6` | M |
| file spacer | 14×14 empty box (aligns files with folder names) when any folder exists | | | M |
| file icon | Pierre sprite: an `svg` 14×14 (`size-3.5`) with `viewBox="0 0 16 16"`, so the 16-unit artwork draws at 0.875 scale, `<use href="#<symbol>">`; `data-pierre-icon` and `data-icon-token` attributes. Without a resolvable icon the lucide `file` fallback takes the row's classes after class merging (14px, `muted-foreground/70`). A built-in image icon with no token is given the `image` token (pierre-icons.ts:236–250). Colour by token, light / dark (PierreEntryIcon.tsx:7–60): default, text, nextjs, stylelint `#84848a`/`#adadb1`; markdown, bash, vue, svgo `#199f43`/`#5ecc71`; typescript, python, c, cpp, biome, docker, vscode, webpack `#1a85d4`/`#69b1ff`; javascript, babel, browserslist `#d5a910`/`#ffd452`; json, html, rust, swift, svg, zig, zip, claude `#d47628`/`#ffa359`; go, react, tailwind, oxc `#1ca1c7`/`#68cdf2`; css, eslint, bootstrap, terraform, wasm `#693acf`/`#9d6afb`; astro, vite, database `#a631be`/`#d568ea`; graphql, image, sass `#d32a61`/`#ff678d`; npm, postcss, ruby, svelte, yml `#d52c36`/`#ff6762`; git `#ff8c5b`/`#d5512f`; mcp, prettier, table `#17a5af`/`#64d1db`; bun `#594c5b`/`#79697b`. An icon resolved through a T3 file-name override has no token and takes the default colour | measured: git `#ff8c5b`, default `#84848a`, markdown `#199f43` | `#d5512f`, `#adadb1`, `#5ecc71`, typescript `#69b1ff` | M |
| file name | mono 12/16 (`text-xs`), `foreground/85`, MiddleTruncate (`title` = full name); hover full foreground | `#18181bd9` → `#18181b` | `#f5f5f5d9` → `#f5f5f5` | M |
| row stats | aligned DiffStatLabel in a mono 10/14 tabular wrapper: two 4ch (24px) right-aligned columns, gap 8 (56 wide), right edge at row − 8 | `#007a55` / `#c10007` | `#00d492` / `#ff6467` | M |
| more files | Button xs ghost-muted in `px-3 pb-2` | `#71717b` | `#a1a1a1` | S |

### 3.12 Loading, empty, error and notice states

| Item | Value | Light | Dark | M/P |
|---|---|---|---|---|
| panel empty / access text | `px-5` (20), centred in the body, 12/16, `muted-foreground/70` | `#71717bb3` | `#a1a1a1b3` | M |
| skeleton header | 32 tall, `padding: 0 12px 0 8px`, gap 8; boxes 10 (in a 20 cell), 20, title 12 tall pill (256 / 207.6 / 288 measured in 539: the fractions apply to the 519px content box, so the short title is 0.4 × 519 = 207.6, under its 208 cap), two 20×12 pills 8 apart at the right | bars `bg-muted-foreground/15` `#71717b26` | `#a1a1a126` | M |
| skeleton separator | 24 tall, two 1px `bg-border/40` lines either side of a 96×10 pill | lines `#e4e4e766` | `#ffffff08` | M |
| skeleton code lines | `padding: 8px 12px`, rows 10 tall 8 apart; 20×10 number pill, 12 gap, then 2/3, 4/5, 3/5 width pills | | | M |
| skeleton shapes | radius 4 (`rounded-sm`) for squares, full for pills; `animation: skeleton 2.4s ease infinite` (motion-safe; 02-primitives §3.27) | | | M |
| notices | `padding: 6px 12px`, 11/16, bottom border `border/70`, bg `bg-muted/40` | text `#71717b`, bg `#fafafa66`, border `#e4e4e7b3` | `#a1a1a1`, `#ffffff03`, `#ffffff0e` | P |
| load error line | 11/16 `text-error/80`, `margin-bottom: 8px`, in `px-3` | `#fb2c36cc` | `#fb414acc` | P |
| raw patch block | `pre` radius 6, 1px `border/70`, bg `background/70`, `padding: 12px`, mono 11/17.875, `muted-foreground/90`, `max-height: 72vh`; reason line 11/16 `muted-foreground/75` | bg `#fcfcfcb3`, text `#71717be6`, reason `#71717bbf` | `#000000b3`, `#a1a1a1e6`, `#a1a1a1bf` | P |
| no-changes text | `padding: 8px 12px`, 12/16, `muted-foreground/70`, centred in the full body (539×808) | `#71717bb3` | `#a1a1a1b3` | M (dark), P (light) |
| recorded tool content | `padding: 12px`, `space-y-3`, 12/16; labels muted; `pre` mono 11 | | | S |

### 3.13 Find panel

| Item | Value | Light | Dark | M |
|---|---|---|---|---|
| position | sticky in the surface, 40px below its top (y 132), right inset 16 | | | M |
| size, one-row layout (panel ≥ 400px) | 334.3×38 in a 539 surface (x 1089.7..1424): padding 4, input box 200×28, "No results" 58.3 wide, Previous / Next at x 1371 / 1395 | | | M |
| size, two-row layout (panel < 400px) | 284×78 in a 323 surface (file tree open): padding 10, input 256×28 across the first row, Previous / Next then the matches label on the second | | | M |
| widget | radius 9, `padding: 4px` (10 below a 400px container), 1px border `color-mix(in lab, fg 20%, transparent)`, shadow `inset 0 0 0 1px <bg>, 0 4px 8px #00000013, 0 6px 18px #00000013`, bg `color-mix(in lab, fg 4%, <popover>)`, 14px sans | bg `#f4f4f5`, border `#18181b33`, inset `#ffffff` | bg `#191919`, border `#f5f5f533`, inset `#111111` | M |
| input | 28 tall (24 line + 2 padding + 1px borders), `padding-inline: 6px`, end padding 72 for the toggles, radius 6, 1px border `color-mix(in lab, fg 12%, bg)`, bg popover, 13px text; focus-visible `outline: 1px solid <primary>` | bg `#ffffff`, border `#dfdfe0`, text `#18181b`, focus outline `#1b4ed8` | `#111111`, `#282828`, `#f5f5f5` | M (focus outline light) |
| toggles | three 20×20 icon buttons inside the input's right end, 1px apart, radius 4; colour `mix(lab, fg 65%, bg)`; hover fg; pressed (`aria-pressed=true`) primary | `#5f5f62` | `#9d9d9d` | M (hover, pressed S) |
| matches | 12/20 500 tabular, min width 50; `mix(lab, fg 50%, bg)` with results, `mix(lab, fg 35%, bg)` with `[data-no-matches]` | "No results" `#a5a5a7`; with results `#818183` (P) | `#575757`; `#797979` (P) | M |
| previous / next | 24×24, radius 4, same colour rule as the toggles; disabled (opacity .25) when there are no matches | `#5f5f62` | `#9d9d9d` | M |
| close | 20×20 circle, top-right corner offset (10, −10); bg `mix(lab, widget fg 50%, panel bg)`, glyph the widget bg; hover: bg the widget fg, glyph the widget bg | bg `#818183`, glyph `#f4f4f5` | bg `#797979`, glyph `#191919` | M (hover S) |
| match highlights | `::highlight(diffs-search-match)`: bg `light-dark(#ff963288, #ff963266)`, text keeps its colour; current match: `color-mix(in lab, <modified base> 45%, transparent)` | `#ff963288`, current `#009fff73` | `#ff963266` | S (values from StyledDiffCodeView.tsx:24–32) |

### 3.14 Colour derivation (for the port)

Every viewer colour is a mix of the code background, the code foreground and one base colour. Formulas, with `bg` = `--code-background` (= `--background` here), `fg` = `--code-foreground` (= `--foreground`), theme fg = the syntax theme's foreground (`#0a0a0a` light, `#fafafa` dark), and bases = the syntax theme's git colours unless blue-orange is on (addition `#18a46c`/`#07c480`, deletion `#d52c36`/`#ff2e3f`, modified `#009fff`). `mix(srgb|lab, A p%, B)` is CSS `color-mix()`; `lab` is CIE Lab (D50).

| Colour | Light | Dark |
|---|---|---|
| change row (content) | `mix(lab, bg 88%, mix(srgb, bg 50%, X))` | `mix(lab, bg 80%, mix(srgb, bg 70%, X))` |
| change number cell | `mix(lab, bg 91%, mix(srgb, bg 35%, X))` | `mix(lab, bg 85%, mix(srgb, bg 60%, X))` |
| X for rows | additions `--diff-addition` (`--success` `#00bc7d`), deletions `--diff-deletion` (`--destructive`) | same tokens, dark values |
| selected rows | the two lines above with X = modified base | |
| deletion dash background | `mix(lab, bg 88%, deletion base)` | `mix(lab, bg 80%, deletion base)` |
| line number text | `mix(lab, theme fg 65%, bg)`; change rows use the base | same |
| selected number text | `mix(lab, modified 65%, black)` | `mix(lab, modified 75%, white)` |
| separator bg / text | `mix(srgb, bg 97%, fg)` / `mix(srgb, fg 50%, bg)` | same |
| expand hover bg | `mix(srgb, bg 92%, fg)` | same |
| title hover | `mix(srgb, fg 84%, --primary)` | same |
| header hover edge | fg at 24% alpha | same |
| word emphasis | base at 15% alpha | base at 20% alpha |
| annotation bg | `mix(lab, bg 98.5%, black)` | `mix(lab, bg 92.5%, white)` |
| split empty-side hatch | `mix(lab, bg 92%, black)` | `mix(lab, bg 92%, white)` |

The 1–2 unit differences between formula and paint come from Chrome mixing the true wide-gamut palette colours (01-tokens §7); port the measured hex for the default palette and the formulas for themes.

## 4. States

| Element | State | Appearance / rule | M/S |
|---|---|---|---|
| scope trigger | default / hover / focus-visible / open | Button xs secondary (02-primitives §4.1); `aria-expanded` while open | M (default, open) |
| scope menu rows | checked | bg `foreground/8%`, no indicator | M |
| | highlighted | `--accent` | M |
| | disabled (Commits status rows) | opacity .64, `not-allowed` | M |
| toolbar toggles | pressed (`data-pressed`, `aria-pressed=true`) | accent bg (ghost) or raised white/`#ffffff14` segment (segmented) | M |
| refresh | refreshing | aria "Refreshing diff", tooltip "Refreshing diff…", icon `motion-safe:visible-animate-spin` (spins while in view, tab visible, no reduced motion) | S |
| file header | hover | 3px inset left edge; copy button fades to opacity 1 | M |
| | collapsed | header only (32px); chevron right; `aria-expanded=false`, label "Expand <path>"; tooltip "Expand diff" | M |
| | expanded | chevron down; "Collapse <path>"; tooltip "Collapse diff"; body fades in (§6) | M |
| | pending (lazy patch not loaded) | rendered collapsed whatever the fold state; a pending file meant to be open starts loading when its header comes within 600px of the viewport, a folded one when it is expanded | S |
| | error (lazy) | `rotate-cw` 12 button "Retry loading diff" after the copy button | S |
| | truncated (lazy) | `info` 12 button "Partial diff preview", tooltip "This file is too large to show in full. Counts include all changes." | S |
| title | hover | primary-tinted colour, 120ms | M |
| copy button | hidden / shown | opacity 0 until file hover or focus within (hover-capable pointers only; touch shows it always) | M |
| | copied | check icon `text-success`, tooltip "Copied", anchored toast; error toast on a denied clipboard | S |
| line | selected | §3.8 colours and blue bar; `data-selected-line` = `single`, `first` or `last` | M |
| | commentable | gutter "+" button on the mouse-hovered row, or pinned to the bottom row of the selection while one exists; none while a draft is open | M |
| | hovered | no tint (`lineHoverHighlight` is off, §3.5) | M |
| draft comment | empty | Comment disabled | M |
| | dictating / pending | textarea read-only while dictation freezes it; Comment and secondary disabled while preparing or pending | S |
| saved comment | hover / focus within | delete button fades in (always visible below 640px) | S |
| tree row | hover / selected / focus-visible | §3.10 | M |
| tree indent guides | idle / tree hovered / in the focused row's folder | opacity 0 / .75 / 1 (§3.10) | M |
| card row | hover / focus-visible / open folder | §3.11 | M |
| panel | no thread | "Select a thread to inspect turn diffs." | S |
| | not a Git repository (any scope but tool call) | "Turn diffs are unavailable because this project is not a git repository." | S |
| | turn scope, no checkpoints | "No completed turns yet." | S |
| | Git scope, file access pending | `DiffPanelLoadingState` "Checking file access..." | S |
| | Git scope, no file access | the access error, else "This connection cannot read local diffs." | M |
| | first load | skeleton with label "Loading tool call diff..." / "Loading checkpoint diff..." / "Loading uncommitted changes..." / "Loading commit..." / "Loading changes..." | M (checkpoint) |
| | loaded, empty | "No net changes in this selection." (an empty, complete patch) or "No patch available for this selection." | S |
| | error | `text-error/80` message above whatever renders | S |
| | unparseable patch | raw `pre` with "Unsupported diff format. Showing raw patch." or "Failed to parse patch. Showing raw patch." | S |
| | tool call with recorded content only | "The provider recorded file content without a diff." + "Recorded content before" / "Recorded content after" blocks ("Empty content." for an empty one) | S |
| notices | untracked still loading / failed | "Showing tracked changes. Checking untracked files..." / "Showing tracked changes. Untracked changes could not be loaded. Refresh to retry." | S |
| | truncated preview | "This preview exceeds the size limit. Changes shown are incomplete." (+ " Totals include all changes." for complete Git previews with file stats) | S |
| | tool call missing files | "This tool call changed <n> files. The recorded patch omits <m> of them." | S |
| lazy list | more files to load | footer with up to 4 skeleton headers, `role=status` "Loading diff…" | S |
| find panel | no matches / matches / current | "No results" (35% mix), "<n> results" / "<i> of <n>" (50% mix); nav buttons disabled at .25 opacity without matches | M (no matches) |
| | layout | one row from 400px panel width, two rows and 10px padding below (§2.11) | M |
| | icon button hover / pressed / focus-visible | colour → fg; pressed toggle → primary; 1px primary outline; close hover inverts | S |
| card | > 250 rows | virtualized rows (absolute, 28px pitch, 8 rows overscan); the focused row stays mounted | S |
| | more files than listed | "<n> more file[s] in the full diff" button | S |
| data attributes | | `data-changed-files-state="tree"`, `data-changed-file-row=<path>`, `data-surface-subheader`, `data-diff-comment-annotation`, `data-diff-copy-path`, `data-workspace-focus-default` (scope trigger), `data-scroll-anchor-ignore` (card controls, so toggling does not anchor-scroll the timeline). Pierre: `data-change-type`, `data-line-type`, `data-selected-line`, `data-overflow`, `data-diff-type`, `data-indicators`, `data-expand-*`, `data-item-*` (tree) | M |

## 5. Behaviour

### 5.1 Opening and toggling the Diff surface

- **Right panel "+" → Diff**, and the thread details **Changes** row: select the **branch** scope ("Changes", keeping the remembered base ref) and open or activate the Diff tab (useRightPanelActions.ts:130–140). Only server threads in Git repositories.
- **`mod+D`** (`diff.toggle`, not while a terminal has focus; Ctrl+D off macOS, ⌘D on macOS): if the panel is open with the Diff tab active, hide the panel; otherwise open or activate the Diff tab with its last selection (useRightPanelActions.ts:97–108; rightPanelStore.ts:1123–1137). Server threads only.
- **Changed files card**: "Open diff" and "<n> more files…" select the card's run with its first file; a file row selects the run with that file (§5.5 reveal). The turn's `revealRequestId` increments on every request, so the same file can be revealed again (diffPanelStore.ts:98–113).
- **File-change tool row → Open diff**: selects the tool-call scope `{threadId, itemId, revision}` (V2ItemInspector.tsx:292–304).
- **Proactive** (setting "Open pull requests and changes panels proactively", off by default; settings.ts:521): when a run completes (status `completed`, no run active) and no linked pull request takes precedence, its checkpoint is `ready` with files in a Git repo, and it touched at least 3 files or at least 50 changed lines, the Diff surface opens on the **branch** scope, unless the user opened or closed a panel since the run started (ChatView.logic.ts:202–231; useProactivePanels.ts:145–189).

### 5.2 Scopes and the selection model

The selection is one of (diffPanelStore.ts:16–21), persisted per thread in localStorage `t3code:diff-panel-state:v1` (default `{kind: "branch", baseRef: null}`):

| Kind | Menu value | Trigger label | Section id (comments, folds) | Source |
|---|---|---|---|---|
| `branch` `{baseRef}` | `branch` | "Changes" | `branch` | `review.getDiffPreview`, `sourceKind: branch-range` |
| `unstaged` | `unstaged` | "Uncommitted" | `unstaged` | same, `sourceKind: working-tree` |
| `commit` `{sha, baseRef}` | `commit:<sha>` | the commit subject, else "Commit" until the subject arrives; the section title (comments, tree label) falls back to "Commit <sha7>" | `commit:<sha>` | same with `commitSha` |
| `turn` `{turnId, filePath, revealRequestId}` | `latest` when it is the newest checkpoint, else `turn:<runId>` | "Latest turn" or "Turn <N>" ("Turn ?" without a number) | `turn:<runId>` | `orchestration.getTurnDiff` |
| `tool-call` `{threadId, itemId, revision}` | none checked | "Tool call" | `tool-call:<thread>:<item>` | the item's recorded patch |

- The menu radio group shows the scope; choosing the already-selected commit again goes back to "Changes" (DiffPanel.tsx:820–835).
- Turns are the thread's checkpoint summaries, ordered by `checkpointTurnCount` descending, then `completedAt` descending (306–312). "Turn N" is the checkpoint's app run ordinal; checkpoints with no ordinal or no run id are dropped (client-runtime threadCheckpoints.ts:36–37), so the submenu can skip numbers or hold a single "Turn 3" (measured). Runs whose checkpoint has no net change still list and show "No net changes in this selection." (measured). A selected turn that disappears is rewritten to the newest one (315–321); while missing, the newest is shown.
- A turn diff is the range `N−1 → N`; for `N = 1` the whole-thread diff `orchestration.getFullThreadDiff {toTurnCount: 1}` is used instead (queries.ts:425–454). There is no "whole thread" scope in the menu; "Changes" (branch against its base) plays that role.
- Choosing "Uncommitted" remembers the branch base ref so "Changes" returns to it (diffPanelStore.ts:42–75).
- Base ref: "Automatic" (`baseRef: null`, the server picks) or a ref from the picker. The comparison shows `<head> → <base>`; head is the source's head ref, else the Git status ref, else "HEAD"; base is the source's base ref, else the selected one, else "Automatic" (DiffPanel.tsx:430–431).

### 5.3 Data and queries

| Data | Request | Notes |
|---|---|---|
| Git status | `vcs.status {cwd}` | `isRepo` (default true while unknown), current ref name |
| file access | `useFilesystemReadAccess(environment)` | Git scopes need it; turn and tool-call scopes do not |
| Git preview | `review.getDiffPreview {cwd, baseRef?, commitSha?, ignoreWhitespace, metadataOnly: true, sourceKind?}` | First asks with `trackedOnly: true` and shows that (`incomplete`) while the full preview (with untracked files) loads (client-runtime review.ts:34–61). Result: `{cwd, generatedAt, sources[{id, kind, title, commitSubject?, baseRef, headRef, diff, diffHash, truncated, patchesDeferred?, files?[{path, previousPath, additions, deletions}]}], incomplete?}`. Revisioned by the workspace mutation id, so every mutation refetches |
| per-file patch (lazy) | `review.getDiffPreview` with `file: {path, previousPath, sourceKind}` | at most 4 reads at once; stale after 5 min or a newer preview |
| file contents (context expansion) | `review.getDiffFileContents {cwd, sourceKind, changeType, baseRef, headRef, oldPath, newPath}` | single-flight per file |
| commits | `review.getBranchCommits {cwd, baseRef?}` → `{commits[{sha, subject, committedAt}], truncated}` | only while the surface is visible |
| refs | `vcs.listRefs {cwd, refKind, includeMatchingRemoteRefs: true, query?, limit: 100}` with `refKind` `local` and `remote` | two queries; local refs exclude the head; pairs a local with its remote twin, preferring `origin` (baseRefChoices.ts:16–46); the query filters label, local and remote names case-insensitively |
| checkpoints | the thread projection's `checkpoints[{id, scopeId, runId, appRunOrdinal, ref, status, files[{path, kind, additions, deletions}], filesTotal?{count, additions, deletions}, capturedAt}]` | `files` may be a prefix of the changed files when `filesTotal` is set (threadCheckpoints.ts:12–29) |
| turn diff | `orchestration.getTurnDiff {threadId, fromTurnCount, toTurnCount, ignoreWhitespace}` / `orchestration.getFullThreadDiff {threadId, toTurnCount, ignoreWhitespace}` → `{threadId, fromTurnCount, toTurnCount, diff}` | diff is a unified Git patch |
| tool call | `orchestration.getTurnItem` (file_change item: `diffStr`, `fileName`, `changes[{operation, path, oldPath}]`, `oldStr`, `newStr`) | revision-checked against the live item |

- **Live refresh** (Git scopes only): showing the surface revalidates the preview unless it was read in the last 10 seconds; no timer (DiffPanel.tsx:626–639). The refresh button refetches explicitly.
- **Tool-call patches** (toolCallDiff.ts): complete Git patches pass through; bare hunks or whole-file contents recorded for add/delete operations are wrapped into a patch with a synthetic `@@ -0,0 +1,n @@` / `@@ -1,n +0,0 @@` header and `a/` / `b/` names; a "… output truncated for transport" suffix marks the content truncated. Whitespace hiding for tool calls is done client-side (diffRendering.ts:79–130); Git and turn diffs ask the server.

### 5.4 Files: parsing, order, totals

- Patches are parsed with Pierre's `parsePatchFiles` (quoted and escaped Git names decoded; diffRendering.ts:157–189). A parse that yields no files, or throws, falls back to the raw text with its reason.
- **Order in the code view**: Git sources with file stats list files sorted by path with `Intl.Collator(undefined, {numeric: true, sensitivity: "base"})`; parsed patches are sorted the same way (useReviewFilePatches.ts:16–19, 133–134, 219). It is a flat full-path sort, so a root file such as `a.txt` sorts before `b/c.ts` and `c.ts` before `d/x.ts`.
- **Order in the file tree** follows the same list: each directory sits where its first file appears, then depth, then `localeCompare` (diffFileTree.logic.ts:88–143). Folders are therefore not forced first. Single-child folder chains are flattened.
- **Order in the changed files card** is different: per level, folders first, then files, each by name with the numeric, base-sensitivity collator; single-folder chains are compacted into one "a/b/c" folder row whose stat is the deepest folder's (turnDiffTree.ts:33–89).
- **Totals**: the toolbar DiffStatLabel sums the Git source's file stats when present, else the hunks' added and deleted lines of the rendered files (DiffPanel.tsx:529–538, 694). The card uses `filesTotal` when present, else sums its files. File stats in the tree and card come from the checkpoint summary.
- **Lazy Git sources** (`truncated` or `patchesDeferred` with file stats): files render as placeholders (no hunks, `isPartial`, cache key ending `:pending`, a `rename-changed` placeholder type when a previous path exists). The first 4 files load at once; when the footer boundary comes within 600px of the viewport the next 4 load (useReviewFilePatches.ts:20, 52–103; DiffFileLoadingBoundary.tsx). Expanding, revealing or finding a pending file loads it.
- A path that appears as both an added and a deleted entry (file ↔ symlink) shows as one "modified" tree entry; both patches render (diffFileTree.logic.ts:31–42).

### 5.5 Folding, revealing, scroll

- **Default fold**: every file collapsed when the "Default diff file state" setting is Collapsed (the default; settings.ts:468), else expanded. Per-scope overrides (`<env>:<thread>:<section id>`) live in a session store, so they survive tab switches but not a reload (diffPanelStore.ts:171–198).
- **Toggle one file**: its chevron, or a click on the header anywhere that is not a button, link or the title (DiffPanel.tsx:1398–1426). A pending file always expands on the first click.
- **Collapse / expand all files**: if every shown file is collapsed, expand all; otherwise collapse all; resets the saved scroll and remounts the viewer (diffCollapse.ts; DiffPanel.tsx:793–797).
- **Reveal** (tree selection, card rows, "Open diff"): expands the file, loads it if lazy, and scrolls the viewer so the file's header sits at the top (`align: start`). A requested range is centred (`align: center`); for patch-only diffs, context is expanded first so the line exists, with 10 lines of margin (useCodeViewFileReveal.ts:30–55). An external request keeps retrying until the file is present, then never repeats for the same request id.
- **Scroll restore**: each scope remembers its scroll top for the session; a remounted viewer returns there, loading lazy batches until the position is reachable, then clamps (useCodeViewScrollRestore.ts). A reveal or collapse-all forgets it.
- **Hidden test files** stay in the lazy-loading index; a requested file stays visible even when it is a test.

### 5.6 Toolbar behaviour

| Control | Effect | Persistence |
|---|---|---|
| Stacked / Split | `updateClientSettings({diffLayout})` → unified (`stacked`) or split view | client setting `diffLayout`, default `stacked` (settings.ts:69–71, 470) |
| Wrap | wrap long lines (`overflow: wrap`) or scroll them (`overflow: scroll`) | component state, seeded from the `wordWrap` setting (default true) on mount |
| Whitespace | pressed = hide whitespace-only changes (refetches) | session store, `null` follows the "Hide whitespace changes by default" setting (default true) |
| Hide test files | hides paths that `classifyChangedFile` calls tests or mocks (changedFileKind.ts:47–87). Test names: `*.test.*`, `*.spec.*`, `*.e2e.*`, `*.eval.*`, `*_test.*`, `*_spec.*`, `test_*.py`, `conftest.py`, `vitest.setup.*`, `jest.setup.*`, and case-sensitive class names such as `FooTest.java`, `FooTests.kt`, `FooSpec.swift`. Test folders: `__tests__`, `__test__`, `test`, `tests`, `spec`, `specs`, `e2e`, `evals`, `cypress`. Mocks: `*.snap`, `*.mock.*`, `*.fixture.*`, `*.stub.*`, `*.fake.*` (and plurals), names that start with `mock(s)`, `fixture(s)`, `stub(s)` or `fake(s)` followed by `-`, `_`, `.`, a capital or nothing (`mockServer.ts`, `fixtures.json`), and the folders `__mocks__`, `__fixtures__`, `__snapshots__`, `mock`, `mocks`, `fixture`, `fixtures`, `fakes`, `stubs`, `testdata`. Generated files (lockfiles, `*.min.js`, `generated/`…) are checked first and never count. Folder names compare lower-cased; the test-name pattern ignores case, the class-name rule and the mock name patterns do not (changedFileKind.ts:31–32, 46–49). Shown only when such files exist; tooltip "Show <n> hidden test file[s]" while hiding | localStorage `t3code.diffHideTestFiles` |
| File tree | shows or hides the tree pane | localStorage `t3code.diffFileTreeOpen`, default hidden |
| Refresh | refetches the Git preview | none |

Settings → Diffs (SettingsPanels.tsx:1497–1605): "Diff layout" (Stacked, Split), "Hide whitespace changes" switch, "Default diff file state" (Expanded, Collapsed), "Diff colors" (Red & green (default), Blue & orange; the trigger shows two 8px dots, deletion then addition, reversed for blue-orange).

### 5.7 Mouse handling in the code view

- Click on a file **title** opens the file: inside a thread, a "file" surface in the right panel at the workspace-relative path; without a thread ref, the preferred editor (DiffPanel.tsx:744–768; diffFileActions.ts:55–112). Absolute paths (some providers' tool-call diffs) are made workspace-relative; paths outside the workspace do nothing.
- Click elsewhere on a **header** toggles the fold (buttons and links keep their own clicks).
- **Right-click** a title: the file context menu (native in the desktop app, DOM fallback in the browser) with "Open", "Reveal in <file manager>", "Open with" ▸ editors, only for paths that resolve to an absolute path (fileContextMenu.ts:59–87).
- Click a **line number**: selects that line; drag or shift-click extends the selection (Pierre line selection). Click the **"+"** on the hovered row to start a comment on the selection or that line.
- Pointer-down inside a comment does not reach the viewer, so selection does not change while editing.

### 5.8 Review comments

- A draft gets an id, a range and a label; the textarea focuses on the next frame, caret at the end. **Escape** cancels (or cancels dictation first); **⌘Enter / Ctrl+Enter** submits when the trimmed text is non-empty (commentSubmitShortcut.ts). Submitting adds a review comment to the **composer draft** of the thread (`addReviewComment`) and clears the draft and selection; the composer shows it as a chip (07-composer). Deleting a saved comment removes it from the composer draft.
- The record (reviewCommentContext.ts:456–506): `{id, sectionId, sectionTitle ("Turn <N>", "Changes", "Uncommitted", the commit subject or "Commit <sha7>", "Tool call"), filePath, startIndex, endIndex, rangeLabel, text, diff: "@@ -o,n +o,n @@\n<marked lines>", fenceLanguage: "diff", selection}`. For full files the line indices count expanded context.
- **Range label**: the number of the first and last selected lines, new number preferred, old for deleted lines; prefixed "+" or "-" when every line in the range is an addition or every one a deletion; "<a> to <b>" for ranges; "line" or "<n> lines" when no number exists (436–453). Examples: "18", "+19", "-7", "+19 to +20", "16 to 20".
- Saved comments are shown only in their own section (scope) and file, restored onto the current diff when the range still maps (`restoreDiffReviewCommentRange`); comments whose lines vanished are not drawn.
- A draft survives a refresh and a lazy placeholder; it is dropped when its file leaves the diff.

### 5.9 File tree

- Built from the shown files' paths with their Git status (`new` → added, `deleted` → deleted, renames → renamed, `change` → modified). A file whose path is also a directory gets an invisible zero-width-space suffix internally (diffFileTree.logic.ts:46–73).
- Initially every folder is open. "Collapse all folders" / "Expand all folders" in the pane header act on every folder (fileTreeExpansion.ts).
- Selecting a file (click or keyboard) reveals it in the code view (§5.5); clicking the already-selected file reveals it again. A reveal from elsewhere expands its ancestors, selects it and scrolls it into view (`nearest`).
- Path updates are applied incrementally (removes before adds, deepest folders removed first) so selection and focus survive paging; a reshuffle that turns a file into a folder resets the model but carries selection, focus and collapsed folders across.

### 5.10 Changed files card

- Shown under an assistant message whose run has a checkpoint summary with at least one file (TimelineAssistantRows.tsx:366–390). The file list is the summary's (possibly a prefix); the title counts `totals.count`.
- Folders start **collapsed** (`threadChangedFilesExpandedById[thread][run] ?? false`, persisted in `t3code:ui-state:v1`). The header button flips all folders for that run; a folder row flips only itself (component state, reset when "all" changes). Each toggle asks the timeline list to re-measure before paint.
- Rows are buttons in tab order (Enter/Space activate); file rows open the turn diff at that file; right-click opens the file context menu.
- More than 250 rows: rows are absolutely positioned at 28px and only the visible window plus 8 rows each side is mounted, computed against the nearest scrolling ancestor (ChangedFilesTree.tsx:23–24, 286–353).

### 5.11 Formats

| Value | Rule | Example | Source |
|---|---|---|---|
| DiffStatLabel counts | below 1,000 as is; then thousands with one decimal below 10k, trailing ".0" dropped, else rounded; same for m below 999,500,000, then b | "+12", "+1.2k", "+12k", "+1m" | DiffStatLabel.tsx:8–20 |
| Pierre header counts | `toLocaleString()` | "+2,382" | patch :391, :397 |
| card title | `count.toLocaleString()` + " changed file" / " changed files" | "6 changed files" | ChangedFilesTree.tsx:69 |
| more files | "<n> more file[s] in the full diff" | | 147–148 |
| Turn time | `Intl.DateTimeFormat` hour numeric, minute 2-digit; 12- or 24-hour per the timestamp setting ("locale" leaves it to the locale) | "2:00 PM" | timestampFormat.ts:3–21, 134–138 |
| commit time | relative: "just now" under a minute, "<n>m ago", "<n>h ago", "<n>d ago" | "3h ago" | timestampFormat.ts:262–285 |
| short SHA | first 7 characters | | DiffPanel.tsx:864 |
| hunk gap | "<n> unmodified line" / "<n> unmodified lines" | "36 unmodified lines" | Pierre messages |
| tooltip counts | "Show 1 hidden test file" / "Show <n> hidden test files" | | DiffPanel.tsx:1270–1272 |

### 5.12 Keyboard and focus

| Keys | Where | Effect | Source |
|---|---|---|---|
| mod+D | anywhere but a terminal | toggle the Diff surface (§5.1) | keybindings.ts:77 |
| F6 workspace cycling | into the right panel | restores the control last focused in this Diff surface, else focuses the scope trigger (`data-workspace-focus-default`) | DiffPanel.tsx:845; lib/workspaceFocus.ts:136–160 |
| menu keys | scope menu, submenus, combobox | Base UI menu and combobox keys (02-primitives §5.2, §5.6); Escape closes one level and returns focus to the trigger (measured) | |
| mod+F | code view focused (`.diff-render-surface`, tabindex −1, takes focus on click; measured) | open the find panel (find mode) with focus in its input; mod+Alt+F asks for replace, which a read-only diff ignores. T3's own mod+F bindings run first (window capture, useChatShortcuts.ts:123–139) but do not take the chord here: focus anywhere under `[data-preview-panel-mode]` counts as `previewFocus` (previewFocus.ts:1–7), so `chat.find` (`!terminalFocus && !previewFocus`, keybindings.ts:116) does not match and the viewer's own capture listener on its root opens the panel (measured) | asset `ie`; asset `PanelLayoutControls` `handleSearchKeyDown` |
| mod+G / mod+Shift+G | code view or find input | next / previous match | asset `ae` |
| Enter / Shift+Enter | find input | next / previous match | asset |
| Escape | find panel | close; focus stays in the viewer rather than dropping to the page (patch :148). Find searches collapsed files and collapsed context too, unfolding a file when it navigates to a match in it (patch :15; DiffPanel.tsx:1489) | |
| Escape, mod+Enter | draft comment | cancel, submit | §5.8 |
| Tab, Enter, Space | toolbar, card rows, header buttons | standard buttons | |
| printable keys | code view or any other non-editable, non-button target | typed into the composer: T3's window-capture layer redirects a single printable key with no Ctrl/Alt/⌘ to the composer's end unless the event path holds an input, textarea, contenteditable, textbox, button, link or menu/option/tab role, an open popup or dialog, or a surface-launcher letter (composerFocusRedirect.ts:85–94; useChatShortcuts.ts:72–81). Not exercised live (it would type into the composer) | S |
| (window focus) | window regains focus | after two animation frames the composer is focused when `document.activeElement` is the body or any element that is not a text field and not inside a dialog, menu, listbox, popup, terminal or `[data-workspace-keyboard-focus]` region (ChatView.tsx:2146–2177, ChatView.logic.ts:1076–1098). The code view (a `div`) and the find panel (its input sits in a shadow root, so `activeElement` is the host `div`) both qualify, so switching windows away and back moves focus from them to the composer (measured several times) | M |
| mod+W | anywhere but a terminal | closes the active right-panel tab, the Diff tab included ([05-header.md](05-header.md) §5.3) | keybindings.ts:75 |
| ↑ ↓, ← →, Home, End | file tree (roving focus) | ↑/↓ move focus (with Shift they extend the selection); → expands a closed folder, else moves down; ← collapses an open folder, else moves to the parent; Home/End first/last row; Enter/Space activate the focused row (selects it, which reveals the file); Ctrl/⌘+Space toggles the focused row's selection, Ctrl/⌘+A selects every visible row (the last selected file is revealed). No type-ahead: T3 turns the tree's search off | asset `FileExplorerPane` keydown switch; DiffFileTree.tsx:82–92 |
| ←, →, Home, End | tree resize handle (focused) | +16 (wider), −16, minimum, maximum; with Alt, Ctrl or ⌘ held the keys do nothing (Shift is allowed) | FileExplorerPane.tsx:86–108 |

### 5.13 Persistence summary

| Key | Store | Content |
|---|---|---|
| `t3code:diff-panel-state:v1` (v2) | localStorage | per thread: selection, branch base ref |
| session | memory | folds per scope, whitespace override, scroll per scope |
| `t3code.diffFileTreeOpen`, `t3code.diffHideTestFiles`, `t3code:diff-file-tree-width` | localStorage | tree open, hide tests, tree width |
| `t3code:ui-state:v1` | localStorage | changed files card folder state per thread and run |
| client settings | `diffLayout`, `diffIgnoreWhitespace`, `diffFilesCollapsed`, `diffColorScheme`, `wordWrap`, `timestampFormat`, `proactivePanelsEnabled` | |
| right panel store | localStorage (03-shell) | whether the Diff tab exists and is active |

## 6. Motion

| What | Property | Duration / easing | Notes | M/S |
|---|---|---|---|---|
| file body after expanding | `opacity` 0 → 1 via `@starting-style` | 200ms ease-out | `[data-diff]`, `[data-file]`; collapsing cuts instantly; none under reduced motion; height never animates | M |
| file title hover | `color` | 120ms ease | | M |
| file header hover edge | `box-shadow` | instant | | M |
| header buttons, toolbar | primitives (`box-shadow`, `scale` 150ms cubic-bezier(0.4, 0, 0.2, 1)) | | 02-primitives §6 | M |
| card rows | colours and background | 150ms cubic-bezier(0.4, 0, 0.2, 1) | | M |
| card folder chevron | `rotate` 0 → 90° | 150ms cubic-bezier(0.4, 0, 0.2, 1) | | M |
| saved comment delete button | `opacity` | 150ms (Tailwind `transition-opacity`) | | S |
| refresh icon | rotate 360° | 1s linear infinite | `visible-animate-spin`, gated on visibility and reduced motion (01-tokens §6) | S |
| skeleton | `skeleton` keyframes | 2.4s ease infinite | motion-safe | M |
| tree loading icon | rotate 360° | 1s linear infinite | only without reduced motion (pierre-tree-theme.ts:15–19) | S |
| tree indent guides | `opacity` 0 → .75 on tree hover; the focused row's folder sits at 1 | 150ms (CSS default `ease`) | | M |
| tree truncation marker | none: Pierre's marker would fade in over 100ms ease-in-out, but T3 hides it (`display: none`, pierre-tree-theme.ts:13) | | | S |
| find panel icon buttons | `color`, `background-color` | 100ms ease-in-out | toggles, Previous / Next, Close (search CSS `[data-search-icon]`) | S |
| menus, tooltips, combobox | primitives | | 02-primitives §6 | |
| programmatic scroll (reveal, find) | Pierre CodeView spring scroll (`omega 0.015`, settle at 0.5px / 0.05 velocity) | | the Diff panel never cancels a pending reveal on user input (the pull request Code tab does, PullRequestCodeTab.tsx:1636–1639) | S |

## 7. Gaps

1. **Split view** was not measured: its toggle writes the `diffLayout` setting. §2.5 is from Pierre's and T3's CSS.
2. **Git scopes** (Changes, Uncommitted, Commits, the base-ref picker, the head → base row, lazy file loading, per-file status buttons, truncation notices, expand-context buttons, the commit SHA link) were not reachable: the session had no filesystem read scope. Only the access message and the disabled Commits row were measured.
3. **Saved review comments** were not created (that edits the composer draft); their colours are probes.
4. **Line hover tint** (resolved in verification): no tint, by source as well as by measurement. Pierre writes `data-hovered` only when `lineHoverHighlight` is enabled, the option defaults to `"disabled"` and T3 never sets it (§3.5).
5. **Word-level emphasis** renders although T3 asks for none (worker default, §3.7), and T3's emphasis, buffer and context colour overrides are ineffective at the level Pierre resolves them. The spec follows what renders; a later T3 build may change either.
6. **Sheet-mode header** (resolved in verification): a live resize to 960 did turn the right panel into a sheet in the verification session, and the 40px bordered header was measured in both schemes (§3.1). Only the rest of the sheet-mode Diff surface (toolbar wrapping at 402px) was not looked at.
7. **Blue & orange** values are computed (§3.9), not measured. The verification recomputed the row, number and dash colours with an independent sRGB + CIE Lab (D50) implementation and matched them to within 1 unit per channel.
8. **Find panel** was measured with no query; match highlight colours and the hover / pressed / focus-visible looks of its buttons are source. The "focus ended in the composer" event is explained (§5.12): when the window regains focus T3 moves focus from any non-text element, the find panel's host `div` included, to the composer. Typed-query behaviour (result counts, current-match colour) was not exercised because it would need typing.
9. **Pierre icon artwork** (sprite paths) is not reproduced here; §2.12 lists the symbol ids, the mapping is in §8.
10. **Fonts**: `ch`-based gutters and widths follow Liberation Mono here (1ch = 0.6em); SF Mono on macOS has a similar advance but is not identical.
11. **The `data-hovered` / `data-editor-*` / merge-conflict styles** of Pierre's CSS are unused by T3's diff panel and not specified.
12. **Draft comment focus ring in dark** (resolved in verification): re-measured with the window focused, border `#346bf1`, ring `#346bf13d` painted `#326df23d`, bg `#ffffff09`.
13. **Printable-key redirect** from the code view to the composer (§5.12) is source only: testing it would type into the real composer.
14. **Expand-context buttons, lazy Git files and per-file status buttons** remain unmeasured (gap 2): every reachable diff was a turn diff, which never carries file contents.

## 8. Port notes for Go/Gio

1. **What this replaces in tern.** `app/ui/screens.go` `diffWindow`, `diffBody`, `diffLine` (2029–2157: the 920×560 modal diff window with a 230px file list), `fileRow` (1969–2007) and the "CHANGES" block of the details panel with its "Last turn / Whole thread" `segmented` control (1911–1968), plus the `PanelChanges` surface wiring in `rightPanel` (2334–2471); `app/ui/shell.go:88–90` (the modal's mount). T3 has no modal and no such segmented control: changes live in a right-panel **Diff** surface with the scope menu of §2.3, and in the timeline's changed files card. `app/ui/model.go` `FileChange`, `DiffLine`, `DiffView` are too thin; `app/core/scene.go` `changes()` (522–545), which sums `file_change` tool items, must give way to checkpoint summaries, and `s.Panel = ui.PanelChanges` (274) to the right panel surface list.
2. **Data tern's core must add.** Checkpoints from the thread projection (§5.3) with `filesTotal`; RPCs `orchestration.getTurnDiff`, `orchestration.getFullThreadDiff`, `review.getDiffPreview` (tracked-first, metadata-only, per-file), `review.getDiffFileContents`, `review.getBranchCommits`, `vcs.status`, `vcs.listRefs`; the filesystem-read capability; the code host commit URL rule (`projectCommitUrl`); client settings `diffLayout`, `diffIgnoreWhitespace`, `diffFilesCollapsed`, `diffColorScheme`, `wordWrap`, `timestampFormat`, `proactivePanelsEnabled`; the workspace mutation id (to refetch); review comments on the composer draft. tern currently calls only `orchestration.getTurnItem` among these.
3. **Diff model.** Parse unified Git patches into files `{name, prevName, type: change|new|deleted|rename-pure|rename-changed, isPartial, hunks[{deletionStart, deletionCount, additionStart, additionCount, collapsedBefore, content: [context n | change (deletions d, additions a)]}], additionLines, deletionLines}`, decoding Git's quoted names and `\ No newline at end of file`. Compute word-level ranges per paired change line the way Pierre's "word-alt" does: a word diff that keeps whitespace tokens (jsdiff `diffWordsWithSpace`), consecutive changed tokens joined into one run, and a one-character unchanged gap after a changed run absorbed into it; skip the pair when either line is over 1,000 characters (asset `getDiffFileInput-BJvrmjXZ.js`, `nn`/`en`). Hide whitespace server-side for Git and turn diffs; client-side only for tool calls (diffRendering.ts:79–130 is the algorithm).
4. **Layout constants.** File header 32; hunk separator 24 (pill inset 8, radius 6); code line 20 at every code font size (13px default; wrap: measured height in 20px rows); number cell `2ch + digits·ch + 1ch + 2px border`; content padding 1ch; 4px change bar; 8px below a file's last row; gutter comment button 20px square at the number cell's right edge overhanging `20 − 1ch`; sticky header per file inside a virtualized list with fixed metrics (§2.2). In split view mirror the number+content pair on each half with 1px gaps in the background colour.
5. **Colours.** Precompute the measured tables for the default theme (§3.3–§3.8) and implement `color-mix` in sRGB and CIE Lab (D50) for themed code backgrounds and the blue-orange scheme (§3.14); keep the syntax-theme git colours (`#18a46c`/`#d52c36`, `#07c480`/`#ff2e3f`) separate from T3's `--diff-*` tokens: the counts in file headers, line numbers and bars use the former, DiffStatLabel and the card use the latter. The deletion bar is dashed (2px period), the addition bar solid. Word emphasis is the base colour at 15% (light) / 20% (dark).
6. **Syntax highlighting.** Reuse the `t3-light` / `pierre-dark` palettes from 06-timeline §3.6 (Chroma lexers mapped onto those scopes); plain code is `#0a0a0a` / `#fafafa`.
7. **Icons.** Lucide icons as listed (§2.12) at stroke 1.75. Pierre's change icons and file icons need their SVG paths vendored from the sprite (check `@pierre/diffs` / `@pierre/trees` licensing first). File icon resolution order (pierre-icons.ts and the bundled tables): T3 file-name overrides (`package.json` → npm, `tsconfig.json` → typescript, `agents.md`, `pnpm-lock.yaml`, `pnpm-workspace.yaml`), T3 extension overrides (video, pdf, doc/docx/odt/rtf, xls/xlsx/ods/csv/tsv, ppt/pptx/odp, epub, cs/csx, dart, diff/patch, ex/exs, hs/lhs, heic/heif → image, java, kt/kts, lua/luau, php, ps1/psm1/psd1, r, scala, toml, xml/xsd/xsl/xslt), then Pierre's file-name table (`.gitignore`/`.gitattributes`/`.gitmodules`/`.gitkeep` → git, `dockerfile`/compose files → docker, `readme.md` → markdown, `claude.md` → claude, lint/format/bundler config names → their tool), then per extension suffix (longest first: `a.b.c` tries `b.c`, then `c`) `jsx`/`tsx` → react and `sass`/`scss` → sass before the extension table (ts/mts/cts/tsx → typescript, js/mjs/cjs/jsx → javascript, json/json5/jsonc/jsonl → json, md/mdx/markdown → markdown, py/pyi/pyw/pyx → python, rs → rust, go, rb/erb/rake/gemspec → ruby, swift, c/h → c, cc/cpp/cxx/hh/hpp/hxx/inl/mm → cpp, css/less/postcss/scss/sass/styl → css, html/htm/xhtml, sh/bash/zsh/fish/ksh/csh → bash, yaml/yml → yml, sql/db/sqlite → database, csv/tsv/xls/xlsx/ods → table, images (png, jpg, jpeg, gif, webp, avif, bmp, ico, icns, tif, tiff) → image, svg, fonts (ttf, otf, woff, woff2, eot), archives (zip, tar, tgz, gz, bz2, xz, 7z, rar, jar, war) → zip, wasm/wat/wast, tf/tfvars/tfstate → terraform, vue, svelte, astro, zig, txt/log/cfg/conf/ini/env/rst/rtf/LICENSE/CHANGELOG/AUTHORS/CONTRIBUTORS → text, graphql/gql, mcp, code-workspace → vscode), else `default`.
8. **File tree.** Gio has no tree widget: build one from the paths with the ordering of §5.4, flattened single-child folders, 24px rows, a 12px status lane and the git status colours of §3.10. It is a separate visual system from the card (sans 12px, status-coloured names, `#009fff` hover/focus accents), so do not share styles with the card. Names truncate as "start… ext" with a plain ellipsis (no fade marker); indent guides are invisible except while the pointer is over the tree (.75) and, always, for the folder of the focused row (1).
9. **Interaction.** Implement: per-scope fold sets defaulting to the setting; header click toggles; title click opens the file surface; reveal scrolls the header to the top; line selection by number click and drag; one "+" utility per viewer that sits on the bottom row of the selection when there is one and otherwise follows the mouse-hovered row (none for touch); no line hover tint; a draft editor with Escape/mod+Enter; find (mod+F when the viewer has focus) over collapsed content too, with the one-row / two-row layout switch at 400px; mod+D toggling the surface; lazy batches of 4 for large Git sources.
10. **Focus and keys around the viewer.** tern's key routing must reproduce two app-wide rules that reach into the Diff surface (§5.12): a printable key pressed while focus is on a non-editable, non-button target (the code view) goes to the end of the composer, and when the window regains focus while a non-text element has focus, focus moves to the composer. Keep the find input and comment textarea as text fields so neither rule fires for them, but note that T3 treats the find panel as a non-text element on window refocus (shadow-root host).
11. **Tests worth writing.** `formatCompactDiffCount` boundaries (999, 1000, 9950, 9999 → "10k", 999,499, 999,500 → "1m"); range labels (§5.8 examples); card tree building and compaction; tree vs code-view ordering on a mixed path set; Pierre header count visibility rules (added empty file, deletions-only, both zero); the colour formulas against the measured tables; test-file classification; scope label rules (§5.2) including "Latest turn" vs "Turn N" and the commit fallback.

## 9. Verification

### 9.1 Extraction pass

- **Live (M)**: Diff surface in a Git worktree thread without file access (both schemes: access message, scope menu, Turn and Commits submenus, toolbar); a turn diff with 6 files (both schemes: headers, counts, change icons, separators, lines, numbers, bars, word spans, wrap and scroll modes, sticky headers, selection, gutter button, draft comment, file tree pane, find panel, loading skeleton caught with a mutation observer); a changed files card with folders (both schemes: header, rows, hover, focus-visible, expanded folder, all expanded).
- **Formula check**: the §3.14 formulas reproduce every measured row, selection, number, separator, title-hover and dash colour within 2 units per channel in both schemes.
- **Restored**: wrap and whitespace toggles, folder states, file tree hidden again, light scheme, viewport 1440×900 @2x; the browser session was closed and its pairing revoked.

### 9.2 Adversarial verification (2026-11-18): what was checked

A second paired session (same build, 1440×900 @2x, schemes switched with the emulated `prefers-color-scheme`) re-measured the surfaces on different data from the extraction: a Git-scope Diff surface without file access, three threads' turn diffs (a new-files-only turn, a whole-file modification with word spans, a partial modification with a leading "<n> unmodified lines" separator), turns with no net change, and three changed files cards. The browser window loses focus and visibility whenever another window covers it, so I brought the tab forward and reset the viewport before each measurement and threw away values read while the document had no focus.

- **Re-measured live, both schemes** (about 75 values, all matching the doc unless listed in 9.3): shell and body colours; inline header row (28 + 12, padding, gap, transparent border); sheet header row (40, border, at a 960 viewport); scope trigger (size, padding, radius, 12/16 500, colours) and its chevron; toolbar positions, gap, DiffStatLabel colours and font, layout group and segments, ghost toggles pressed and unpressed, collapse-all icon; menu popup glass, border, shadow, blur; rows, checked row, open sub-trigger, sub-trigger chevron, submenu offset (1064, 165.5), empty Turn submenu (160×10), Turn row time (12/16, "8:18 PM" 48.3); Commits disabled row (276.9×28, .64, not-allowed); `diffs-container` (13/20, clip, colours); file header (32, padding 8/12, 12px/1, sticky, z 4); header hover edge; collapse toggle (20×20 at 907, radius 4, chevron colours); change icons new and change; title colour, hover colour, rtl, 120ms; copy button (20×20, x = title end + 8, shown on hover); counts colours and order; number column (2ch / 1ch padding, 2px border, 41 / 48.8 columns for 2 / 3 digits); context, addition and deletion rows, numbers and bars, the dashed deletion bar; word spans (radius 3, clone, 15% / 20% alpha); wrap and scroll modes; separator row, wrapper and pill (24, radius 6, 11px, colours); selection row, number, number text and bar; gutter button size, colours, offset and placement; draft annotation, textarea wrapper, placeholder, hint, Cancel, Comment; annotation row selection colours; file tree pane width and resize values (215.6 / 160 / 283), header, host colour, scroll padding, rows, hover, selected, icons, chevron, lane letters and folder dot, status-coloured names, indent guides, truncation; changed files card (bg, header, sticky top 8, title, counts, actions, rows, hover, folder chevron, folder icon and name, file spacer, icon colour, name, aligned stats); find panel in both layouts and schemes; loading skeleton (dark, caught with a mutation observer); "No net changes" state (dark); sprite symbol count (81 = 58 + 23) and change-icon symbol ids; code view root, `pre`, `code` and header attributes.
- **Behaviour checked live**: header-background click toggles the fold; Escape in a menu closes one level and returns focus to the trigger; clicking a code line focuses `.diff-render-surface`; mod+F there opens Pierre's find with focus in its input; Escape closes it and leaves focus on the surface; a draft opens on the selection with `aria-label "Comment on lines 2"` and Cancel removes it and the selection; the "+" stays on the selected row while the pointer hovers other rows; no `data-hovered` ever appears; regaining window focus moved focus from the code view and from the find input to the composer.
- **Re-derived from source** (35 checks, all consistent unless listed in 9.3): DiffPanelShell header and skeleton classes; `formatCompactDiffCount`; ChangedFilesTree classes and virtualization constants; the card's mount and default-collapsed folders; keybindings 75, 77, 116; the six settings defaults; FileExplorerPane width rules and keys; timestamp formats; StyledDiffCodeView CSS and item metrics; diffRendering surface and change-icon CSS; Pierre's base CSS formulas (rows, numbers, dashes, emphasis, separators, expand buttons, utility button); worker defaults (`word-alt`, 1,000-character limits); Pierre's interaction manager; patch lines 15, 148, 384–397 and the header-count rule; range labels; test-file classification; card tree building; the 52 icon colour tokens; the icon resolver; `useCheckpointDiff`; right-panel open and toggle; selection store and persistence; opening actions; proactive thresholds (3 files or 50 lines); per-file status, copy and loading-boundary components; RefreshIcon; the settings colour dots; the file context menu; title-click resolution; the tree CSS and T3's tree overrides; the find panel CSS; the window-focus, type-to-composer and preview-focus rules; checkpoint summary derivation; `--diffs-font-size`; the tern files named in §8.
- **Formulas**: an independent sRGB + CIE Lab (D50) implementation reproduced the red-green rows, numbers, dashes, numbers' text and selection within 2 units per channel, every blue-orange row, number and dash value in §3.9 (to within 1 unit), and the find panel's 35%, 50% and 65% mixes.
- **Privacy**: searched the doc for names, paths, hosts, accounts and copied file names; nothing personal was found. The examples ("…/lib/parser.ts", "compo… json", "s. / com… / u.") are generic. Real file names seen during verification were not written down.

### 9.3 What was wrong, and the fix

1. **Gutter "+" placement** (§3.8, §2.4, §4, §8). The doc said it appears on the hovered row. Pierre keeps one button per viewer and, while lines are selected, pins it to the bottom row of the selection whatever the pointer hovers; only without a selection does it follow the mouse (not touch). Measured and confirmed in the bundle.
2. **Find panel size** (§2.11, §3.13). "284×78" was given as the panel's size. That is the two-row layout used when the panel is narrower than 400px (the file tree was open). In a 539 surface the widget is one 334×38 row with 4px padding. Both layouts are now specified.
3. **Indent guides** (§3.10, §4, §6). Opacity was given as 0, .75 on hover. The guides of the focused row's folder are always at 1 (an injected rule); measured at 1 with and without hover.
4. **Tree truncation marker motion** (§6). A 100ms fade was listed. T3 hides that marker (`display: none`) and uses a plain end ellipsis on the visible segment, so there is no fade.
5. **Code view surface colour** (§3.3). The doc gave the surface `#fcfcfc` / `#000000`. Its own background is transparent; the colour comes from the body wrapper and the containers.
6. **File tree maximum width** (§3.10). `max(160, min(560, row − 256))` is a simplification. With a responsive default the reserve never shrinks the default, so the max is `max(160, min(560, max(row − 256, default)))`, which differs for rows of about 400 to 427px.
7. **Card file icon geometry** (§3.11, §2.12). "`<use>` 12.25 inside" was wrong. The icon is a 14px `svg` with a 16-unit viewBox; the lucide fallback in the card ends up 14px and `muted-foreground/70`, not 16px `text-icon-muted`.
8. **Resize-handle modifiers** (§5.12). The doc's "modifiers ignored" read as "modifiers make no difference". With Alt, Ctrl or ⌘ held the keys do nothing; Shift is allowed.
9. **Classification case rules** (§5.6). Only the class-name rule was called case-sensitive. The mock name patterns are case-sensitive too, while folder names compare lower-cased.
10. **Draft textarea geometry** (§3.8). "252×68 in a 276 column" described one layout. It is the annotation width minus 24, and the focused look needs the window focused.
11. **Status labels** upgraded from S / P to M where now measured: sheet-mode header, dark skeleton, dark "No net changes", dark highlighted menu row, dark selected annotation row, the dark unfocused and focused draft textarea.
12. **Gaps 4, 6 and 8** were open or unexplained. All three are resolved in §7: no hover tint by source, the sheet header measured, the composer focus explained.
13. **tern line ranges** (§8). The modal mount is `app/ui/shell.go:88–90` and `diffLine` ends at screens.go:2157 (the doc had 88–89 and 2029–2160). The other tern references check out.

### 9.4 What was missing, now added

- Wrap toggle tooltips differ from its aria-labels; the layout toggles have no tooltips; the copy button's `aria-label "Copy file path"` and "Copy path" tooltip.
- DiffStatLabel weights (500 in the card header, 400 elsewhere) and compact counts in the toolbar.
- Both "Latest turn" and the newest "Turn N" row show as checked; checkpoints without an ordinal or run id are dropped, so turn numbers can skip.
- Folder lane dot title "Contains git status items"; tree truncation splits off the extension; dot colours.
- Find panel: the 400px layout switch, button colours and their hover / pressed / focus-visible / disabled states, the 100ms ease-in-out transition, matches colours with and without results, the input's primary focus outline, the hidden replace cell.
- Keyboard and focus: printable keys typed in the code view go to the composer; regaining window focus moves focus from the code view or find panel to the composer; why mod+F reaches Pierre rather than T3's thread find; mod+W; the full tree key map.
- Code rows stay 20px at every code font size (`--diffs-font-size` only).
- Coarse-pointer expand buttons sit side by side; the sticky container is a flex box; `code` carries `grid-row: span <rows>`; how `data-selected-line` is spread over a one-line draft.
- Port notes 8–10: tree truncation and guides, the "+" placement, no hover tint, the find layouts, and the focus and key rules tern must reproduce.

### 9.5 Still open

Split view (gap 1), every Git-scope surface (gaps 2 and 14), the saved comment (gap 3), measured blue-orange colours (gap 7), typed find queries (gap 8) and the printable-key redirect (gap 13). The verification session's toggles were restored (wrap on, file tree hidden, light scheme, 1440×900 @2x). The browser dropped its pairing once mid-session and was paired again; the unpair script then closed it and revoked every session paired under this label.
