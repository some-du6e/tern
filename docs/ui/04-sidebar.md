# 04 · Thread sidebar content

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
This document covers what is drawn inside the sidebar frame: the search and filter row, the thread list with every row variant, the shelves, the empty and search states, the footer, the hover card and the row menus, plus the keyboard, drag and motion behaviour of the list. The frame itself (width, rail, collapse, the header row with the trigger and brand, the stage art) is in [03-shell.md](03-shell.md). Token values come from [01-tokens.md](01-tokens.md); inside the sidebar, dark mode uses the scoped palette of 01-tokens §3.5 (`tokens.extra.json` → `scopes`). Primitives (Button, Tooltip, Menu, Combobox, Kbd, MiddleTruncate, CollapsibleSectionHeader, ScrollArea) are in [02-primitives.md](02-primitives.md) and are referenced, not repeated.

Conventions, as in 01–03:

- Lengths are CSS px at the default 16px root (1 CSS px = 1 Gio dp). Values written as `rem` in source scale with the interface font size.
- Colours are sRGB hex as Chrome resolves them, `#rrggbbaa` when translucent (RGB exact, not premultiplied). Inside the sidebar container, "dark" means the `[data-app-sidebar]` scope: `--sidebar` `#000000`, foreground `#f1f3f7`, muted `#a3a3a3`, secondary label `#8f8f8f`.
- **M** = measured live (computed style or box), **P** = the real class list evaluated on a probe element inside the sidebar container (for states this session could not show), **S** = source only.
- Coordinates are viewport px at 1440×900 @2x, sidebar at its default 256px (surface 0..255, list content 8..247).
- Icons are lucide at stroke 1.75 unless a row says otherwise. Sizes are the rendered box.

**The measured session was read-only** (pairing scope `orchestration:read`). T3 gates everything that writes on the operate scope: rows were not draggable, the snooze clock and the unpin button were not rendered, rename was refused, and menu items that write were disabled. Those states are P or S below. No data was changed.

## 1. Scope and sources

Paths are relative to the T3 repo (`apps/web/src/components` shortened to `c/`).

| File | Lines | What it gives |
|---|---|---|
| `c/Sidebar.tsx` | 330–336 | settled page sizes (10, then +25), shelf expansion keys |
| | 338–347, 734–803, 1866–1878 | optimistic drop record and its release rule |
| | 416–613 | `useSidebarThreads`: shelves, folders, rendered rows, ordered keys |
| | 615–715 | `partitionSidebarThreads`: sections and per-section sort |
| | 806–846, 848–859 | lookups, filter items ("All projects" + pickable groups), selected badge |
| | 861–879, 881–888 | operate check toast; drag distance 6, auto-scroll, drag label height 24, search result ids |
| | 1076–1108, 1134–1164 | project scope resolution, persistence, route reveal widening |
| | 1166–1178 | scratch machine label |
| | 1180–1205, 1207–1225 | filter popup items (order held while open), thread search, draft count |
| | 1227–1306 | project settings navigation, project folder menu |
| | 1363–1434 | search ranking, result keys, highlight |
| | 1442–1451, 2884–2929 | jump labels, window shortcuts (previous/next/jump), jump hints |
| | 1453–1495 | navigation to thread / draft, file drop on a row |
| | 1497–1536 | search keyboard |
| | 1538–1592 | rename, row click (mod, shift, double) |
| | 1594–1695 | settle/unsettle/wake and "advance after park" |
| | 1696–1937, 2027–2236 | drag: modifiers, motion hookup, context drag, sweep, drag start/over/end |
| | 1939–2026 | list items with markers, list motion key |
| | 2237–2289 | snooze |
| | 2291–2540 | multi-selection menu |
| | 2542–2591 | draft menu |
| | 2593–2882 | thread menu and its actions |
| | 2931–2960 | New thread button and its shortcut labels |
| | 2961–3665 | JSX: header, combobox, search results, empty state, list, "Show N more", footer |
| `c/Sidebar.logic.ts` | 57–71, 73, 75–85, 87–110, 112–134, 136–172, 186–188 | park navigation, jump hint delay 200, row a11y label, subscription lease (160px overscan), file drag clear, snooze clock, removed checkout |
| | 192–335 | sections, markers, drop target, drop verb, sweep keys, key at y |
| | 337–528 | reorderable keys, drop plan, drop projection |
| | 584–632 | hidden lineage, row thread for an open child, visibility filter |
| | 634–657 | bulk menu items |
| | 691–787 | shortcut context, modifier hold store, jump hint visibility |
| | 789–849 | last visited, unseen completion/failure, trailing double click, nested link, new-thread rule |
| | 890–1037 | adjacent thread, `SidebarThreadStatus`, recede rule, status resolution, folder dot |
| | 1044–1087 | folder expansion keys, folders |
| | 1100–1203 | timestamps, sort re-exports, search, filter items, filter menu reducer |
| `c/Sidebar.motion.ts` | 1–252 | list FLIP / fade controller |
| `c/Sidebar.pointer.ts` | 1–145 | pointer sensor (activation, cancel paths, click suppression) |
| `c/Sidebar.drag.ts` | 1–251 | label clamp, collision detection, projected sorting strategy |
| `c/Sidebar.snooze.ts` | 1–57 | presets per environment, pick, "time has passed" toast |
| `c/sidebar/SidebarChrome.tsx` | 43–94, 123–194, 196–302 | header row (03-shell), back/forward, brand, footer utility menu |
| `c/sidebar/SidebarThreadHeader.tsx` | 34–170 | search field, filter/add/new buttons, header icon button |
| `c/sidebar/SidebarThreadRow.tsx` | 139–183 | age label, jump hint, working timer, terminal label |
| | 185–231 | provider stack |
| | 233–370 | hover card content |
| | 372–433 | snooze menu button |
| | 440–527 | draft classes, sortable wrapper, drop verb badges |
| | 529–640 | status icons, props, rename input |
| | 642–1617 | thread row (all variants) |
| | 1619–1823 | `latestRunDiff` (always null), search result row |
| `c/sidebar/SidebarListMarkers.tsx` | 13–147 | sortable markers, placeholder, drag boundary, shelf header |
| `c/sidebar/SidebarProjectFolderHeader.tsx` | 14–121 | boring-sidebar folder header |
| `c/sidebar/SidebarDraftBlock.tsx` | 26–235 | draft rows and divider |
| `c/sidebar/SidebarThreadUndoNotice.tsx` | 16–62 | footer undo notice |
| `c/sidebar/SidebarUpdatePill.tsx`, `SidebarUpdateCapsule.tsx`, `DesktopUpdateStatusIcon.tsx`, `SidebarProviderUpdatePill.tsx` | 195–590; 58–183; 1–128; 45–210 | footer update control, update-everything capsule, status icons, provider update pill |
| `c/sidebar/UpdateEverythingDialog.tsx`, `SidebarUpdateReleaseNotes.tsx` | 100–250; 1–163 | "Update N machines?" confirm dialog, release-notes hover popover |
| `c/ui/collapsible-section-header.tsx`, `packages/client-runtime/src/clock.ts` | 5–48; 15–71 | shelf header tones, minute and second ticks (wall-clock aligned) |
| `c/AppSidebarLayout.tsx` | 222–292 | `thread.undo` capture, held-key and text-field rules |
| `c/sidebar/SidebarFolderDrop.tsx`, `routeReveal.ts`, `mainAppLocation.ts` | 29–134; 16–87; 7–45 | folder drop overlay, scroll-to-open-thread, utility pages |
| `c/ThreadHoverCard.tsx` | 1–33 | hover card frame |
| `c/threadActionMenu.logic.ts` | 39–269 | draft and thread menu items, operate gating |
| `c/ThreadStatusIndicators.tsx` | 200–356, 358–448, 450–468, 531–635, 845–924 | PR badge and mini list, PR status, terminal button, worktree icon, search PR number |
| `c/pullRequest/pullRequestIcons.tsx` | 13–54 | PR glyphs and tones |
| `c/ProjectFavicon.tsx`, `c/ProjectMonogram.tsx`, `apps/web/src/projectIdentity.ts`, `apps/web/src/projectIconColors.ts` | 35–202; 7–49; 1–41; 3–125 | project icons, monogram, derived identity, 18 icon colours (the last two live in `src/`, not `components/`) |
| `c/settings/ProjectSettingsPanel.logic.ts` | 9–14 | "No project" icon (`message-square-dashed`, gray) |
| `c/chat/ProviderInstanceIcon.tsx`, `c/Icons.tsx` | 29–194; 405–431 | provider logos and badge |
| `c/EnvironmentMachineIcon.tsx`, `c/ProjectEnvironmentBadge.tsx`, `c/ProjectPinButton.tsx`, `c/ProjectComboboxRow.tsx` | 41–73; 7–44; 28–67; 4–18 | machine icons, environment badge, pin, row actions |
| `c/ThreadSearchMatch.tsx`, `c/ThreadCommandSubtitle.tsx`, `c/ThreadContextChip.tsx` | 9–51; 32–111; 13–61 | search excerpt; palette subtitle and composer chip (not drawn in the sidebar, see §5.20) |
| `c/CustomSnoozeDialog.tsx`, `c/chat/threadContextDrag.ts`, `c/chat/ThreadContextDragGhost.tsx` | 1–254; 34–104; 6–25 | custom snooze dialog, context drag, drag ghost |
| `apps/web/src/contextMenuFallback.ts`, `apps/web/src/localApi.ts` | 169–626; 42–57 | the browser's DOM context menu; native menu in Electron |
| `apps/desktop/src/electron/ElectronMenu.ts` | 63–200 | native menu mapping (detail in parentheses, auto separator, trash icon) |
| `c/ui/sidebar.tsx` | 624–635, 662–818 | `SidebarInput`, footer, content (scroll area), group, menu, menu button |
| `apps/web/src/lib/sidebarTree.ts` | 1–184 | arrow-key tree navigation, focus restore, context-menu key |
| `apps/web/src/threadSelectionStore.ts` | 28–110 | multi-selection, anchor, range |
| `apps/web/src/hooks/showThreadUndoNotice.ts` | 15–169 | undo notice text, grouping, 5s expiry |
| `apps/web/src/timestampFormat.ts` | 134–138, 237–285 | short time, relative age |
| `packages/client-runtime/src/state/threadSort.ts` | 26–69, 268–336 | activity/settled timestamps, settled/pinned/active sorts |
| `packages/client-runtime/src/state/threadInbox.ts` | 17–108, 134–273 | working rule, launching overlay, inbox/working sorts, Working shelf exit grace |
| `packages/client-runtime/src/state/threadSettled.ts` | 39–197, 199–360 | queued grace, snooze rules, woke time, presets, wake label |
| `packages/client-runtime/src/state/models.ts` | 120–141, 370–407 | runtime activity, provider stack (3), working start, timer format |
| `packages/contracts/src/settings.ts` | 129–135, 465–467, 523–555 | default snooze presets, confirm settings, sidebar settings |
| `packages/shared/src/keybindings.ts` | 58–143 | default shortcuts |
| `apps/web/src/index.css` | 10–17, 104–111, 2425–2436 | `any-hover` variant, sidebar geometry tokens, drag label reveal |
| shipped `uiStateStore-BaLUrhpA.js` | `Sn={duration:200,easing:"ease"}` | dnd-kit sortable transition |

Live page: same build, 1440×900 @2x, light and dark, on a draft view and on an existing thread, with all sidebar settings at their defaults (Working shelf on, boring sidebar off, `timestampFormat: locale`). The list held pinned, active, working and settled threads, but no snoozed thread, no draft with content and no row in the approval, request or limited states. The first pass had no input or failed row either. The verification pass (§9) had one of each, a Woke row, Waiting and Working rows on the Working shelf, and a thread with 58 linked pull requests, and measured those live. The tab was repeatedly backgrounded by other windows, which stalls Chrome's animations; every animated measurement was taken after bringing it to the front. Reference captures stay in a private folder outside this repo.

## 2. Structure

### 2.0 Header row (summary)

The 52px row above the search row is specified in 03-shell §2.1 and §3.2–3.3, with its geometry and colours. This is what it holds (SidebarChrome.tsx:43–194), so the port can build it from one list:

- The sidebar trigger sits outside the sidebar container, fixed at the window's top-left: `aria-label="Toggle main sidebar"`, `panel-left-close` / `panel-left` 16, tooltip side bottom with the `sidebar.toggle` shortcut (default Ctrl/Cmd+B). Below 768px a second trigger renders inside the header row (`md:hidden`).
- The brand link `a[aria-label="T3 Code home"]` goes to `/`. It holds the T3 wordmark SVG at `h-[1cap]` and "Code" in `muted-foreground`, 14px 500 `tracking-tight`, 28 tall, radius 6, 2px inset focus ring. On the stage art both turn white, "Code" white/70. Hidden below 768px.
- Desktop app only, from 768px: back and forward buttons after the brand. `Button size=icon-sm` (28×28 from 640px, 32 below), variant `ghost`, or `media-navigation` on the art (white/90, hover bg white/10). Icons `arrow-left` / `arrow-right` 14. Labels "Go back" / "Go forward"; the tooltip (side bottom) appends the `navigation.back` / `navigation.forward` shortcut, default Ctrl/Cmd+[ and Ctrl/Cmd+], as "Go back (Ctrl+[)". A button is disabled while its history direction is empty.
- With environment identification set to `pill`, a `Badge size=sm variant=secondary` reading "Nightly" or "Dev" follows the brand (`ml-1`, 28px slot).

### 2.1 Sidebar content tree

Below the 52px header row (03-shell §2.1). Measured DOM, classes trimmed where they only carry variants for other sidebar configurations.

```
div.relative.flex.min-h-0.flex-1.flex-col                     SidebarFolderDrop (folder drop zone + overlay)
├─ div  sidebar header row                                     03-shell §3.2
├─ div.w-full.shrink-0                                         SidebarContent fixedHeader
│  └─ div[data-sidebar=group]  relative flex w-full min-w-0 flex-col p-(--sidebar-content-inset) z-[1]     0,52 255×48
│     └─ div.flex.items-center.gap-1                                                                         8,60 239×32
│        ├─ div  search field (§2.2)                          flex-1
│        └─ div.flex.shrink-0.items-center                    [filter combobox][add project] (only with projects) [new thread]
├─ div[role=presentation]  ScrollArea root                     hideScrollbars, scrollFade (--fade-size .75rem), no fade padding
│  └─ div[data-slot=scroll-area-viewport][tabindex=-1]        0,100 255×756
│     └─ div[data-sidebar=content]  flex w-full min-w-0 flex-col [overflow-anchor:none] min-h-full
│        └─ div[data-sidebar=group][role=presentation]  relative flex w-full min-w-0 flex-col p-(--sidebar-content-inset) flex-1
│           │                                           (padding 0 8 8: the content's [&>group:first-child]:pt-0 rule removes the top)
│           ├─ ul#sidebar-thread-search-results[role=listbox]   while searching with results (§2.6)
│           ├─ p[role=status]                                   while searching with no results
│           ├─ div  empty state                                 §2.9
│           └─ ul[role=presentation]  relative flex flex-col gap-px [flex-1 when it has items]   8,100 239×…
│              └─ list items (§2.3)
└─ div[data-sidebar=footer]  flex flex-col gap-2 px-2 pt-1 pb-2                                              0,856 255×44
   ├─ provider update pill (§2.11)
   ├─ Apple-silicon warning (desktop)
   └─ ul[data-sidebar=menu]  flex w-full min-w-0 gap-1 flex-row items-center                               8,860 239×32
```

### 2.2 Search and filter row

```
div  flex h-8 min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-sm font-medium text-sidebar-muted-foreground
     hover:bg-sidebar-row-hover hover:text-sidebar-foreground focus-within:bg-sidebar-row-hover focus-within:text-sidebar-foreground
     focus-within:ring-1 focus-within:ring-inset focus-within:ring-ring/70                                   8,60 151×32
├─ svg.lucide-search  size-4 shrink-0 text-(--sidebar-icon-color)                                           16,68 16×16
├─ span[data-slot=input-control]  min-w-0 flex-1  (SidebarInput: unstyled Input, nativeInput)              40,65.5 111×21
│  └─ input[type=search][role=combobox][aria-label="Search threads"][placeholder="Search"]
│        aria-autocomplete=list aria-expanded aria-controls aria-activedescendant   font-medium text-sm leading-normal
└─ [Button size=icon-micro variant=ghost-muted aria-label="Clear thread search"]  > svg.lucide-x size-3     131,66 20×20, only while the query is non-blank
div  flex shrink-0 items-center                                                                              163,62 84×28
├─ ComboboxTrigger render=SidebarHeaderIconButton  label "Filter threads by project[: A, B]"               163,62 28×28
│     content: ListFilter 16 | the scoped project's favicon 16 | two favicons 14 (second: -ml-1 rounded-[25%] bg-sidebar ring-2 ring-sidebar)
├─ SidebarHeaderIconButton "Add project"  > FolderPlus                                                      191,62 28×28
└─ SidebarHeaderIconButton "New thread"   > SquarePen                                                       219,62 28×28
SidebarHeaderIconButton = Tooltip(side top) > SidebarMenuButton size=icon + "relative size-7 shrink-0"
                          + span "pointer-events-none absolute left-1/2 top-1/2 size-[max(100%,3rem)] -translate-1/2 pointer-fine:hidden" (48px touch target)
```

### 2.3 List items, in order

`sidebarListItems` (Sidebar.tsx:1939–1980) plus the draft block and the "more" button (3476–3486, 3631–3654). Markers are zero-height `li`s that dnd-kit sorts like rows.

```
li…  draft rows (§2.7) + li divider                      only drafts with user content, not promoted, in scope
li[data-testid=sidebar-pinned-header]                    marker; label "Pinned" only while dragging
li[data-thread-item]…  pinned rows, variant card
li[data-testid=sidebar-pinned-divider]                   marker; label "Active" only while dragging
li[data-testid=sidebar-active-placeholder]               marker; dashed hint while dragging into an empty Active
li[data-thread-item]…  active rows, variant card         (boring sidebar: folder headers + folder rows instead, §2.10)
li[data-testid=sidebar-working-header]  + working rows   only when ≥ 1 working thread; rows only while expanded
li[data-testid=sidebar-snoozed-header]  + snoozed rows   only when ≥ 1 snoozed thread; rows (slim) only while expanded
li[data-testid=sidebar-settled-header]                   always, once the list has any thread
li[data-testid=sidebar-settled-placeholder]              marker
li[data-thread-item]…  settled rows, variant slim        only while expanded (first 10, then pages of 25)
li > button[data-sidebar-row="settled:more"]             "Show N more" while expanded and rows are hidden
```

A collapsed shelf still renders the open thread's row under its header (`routeThreadOnly`, Sidebar.tsx:406–413). If the list holds no thread at all (and no folder), it has no items, not even the markers.

### 2.4 Card row (pinned, active, working)

SidebarThreadRow.tsx:1417–1616. `{…}` marks optional parts.

```
li[data-thread-item=<key>]  list-none py-0.5 [content-visibility:auto] [contain-intrinsic-size:auto_78px | auto_58px when no meta line]
│                           + relative z-20 while dragged; carries dnd-kit transform/transition and file-drop handlers
└─ Tooltip(disabled while dragging or the snooze menu is open)
   └─ div[role=button][tabindex=0][data-sidebar-row=<key>][data-sidebar-parent?][data-testid=sidebar-row-card]
      │  aria-label="<title>, <status label>, <project>"  aria-current=page (open thread)  aria-busy (title regenerating)
      │  class = rowSurface (§3.5): group/sidebar-row relative w-full cursor-pointer overflow-hidden rounded-md text-left outline-none
      │          select-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring + state classes
      ├─ span.sr-only  <title>
      └─ div  relative z-10 h-[4.875rem] px-(--sidebar-row-content-inset) py-(--sidebar-content-inset) [h-auto when no meta line]
         ├─ div  line 1: flex h-5 min-w-0 items-center gap-1.5
         │  ├─ {draft pen: span[role=img][aria-label="Unsent draft"] > SquarePen size-3 text-warning-foreground, tooltip "Unsent draft" (top)}
         │  ├─ {ProjectFavicon size-4 shrink-0}
         │  ├─ {pin: button "Unpin thread" (Pin ↔ PinOff on hover/focus) | svg Pin [role=img][aria-label=Pinned]}
         │  ├─ span  min-w-0 flex-1 truncate font-normal text-secondary-label text-xs   <project display name>   (or an empty flex-1 span)
         │  └─ span.group/sidebar-status-slot  relative ml-auto flex h-5 min-w-8 shrink-0 items-stretch justify-end text-xs
         │     │   (while dragged: the drop-verb badge instead; hidden during an action sweep; jump hint: other children invisible+absolute)
         │     ├─ span  status-or-age: flex items-center self-center tabular-nums text-secondary-label
         │     │        + (not Woke) pointer-events-none and, on row hover or a focus-visible control in the slot: absolute right-0 opacity-0
         │     │   └─ status label (§3.6) | Woke button | RelativeTimeLabel
         │     ├─ {span controls: pointer-events-none absolute inset-y-0 right-0 flex items-stretch opacity-0
         │     │        → static, opacity-100, pointer events on row hover or when a control inside has focus-visible; stays while the snooze menu is open
         │     │   ├─ {Discard draft: button px-1.5 > X size-3.5, tooltip "Discard draft" (top)}
         │     │   ├─ {Snooze: Menu > button "Snooze thread" h-full px-1.5 gap-0.5 > Clock size-3, tooltip "Snooze thread"}
         │     │   └─ {Settle: button "Settle thread" -mr-1 px-1.5 gap-1 > Check size-3.5 + "Settle", tooltip "Settle thread"}}
         │     └─ {Kbd[data-jump-hint]  jump hint, desktop only}
         ├─ div  line 2: mt-1 flex min-w-0 items-center gap-1.5
         │  ├─ title span (§3.5) | rename input (§3.17)
         │  ├─ {span.sr-only[role=status] "Regenerating title"}
         │  └─ {title-line icons when the meta line is empty: span inline-flex shrink-0 items-center gap-1 > [remote machine][provider stack]}
         └─ {div  line 3 (meta): mt-0.5 flex min-w-0 items-center gap-1.5 text-secondary-label text-xs
            ├─ branch:  ThreadWorktreeIndicator? + span.flex.min-w-0.flex-1 > MiddleTruncate(branch, no title)
            │   │  ThreadWorktreeIndicator (only with a worktree path): span[role=img][aria-label="Worktree: <path> (<branch>)"]
            │   │     > FolderGit2 size-3.5 text-icon-muted, tooltip with the same text (top)
            │   | scratch machine: EnvironmentMachineIcon size-3 + span.min-w-0.flex-1.truncate <machine label>
            │   | span.flex-1
            ├─ {terminal button: button[aria-label="1 terminal process running. Open terminal" | "N terminal processes running. Open terminal"]
            │     inline-flex rounded-sm text-teal-600 dark:text-teal-300/90 hover:opacity-80 focus-visible:ring-2 ring-inset
            │     > Terminal size-3.5 motion-safe:animate-status-pulse, tooltip with the aria-label text (top)}
            ├─ {PR badge: InlineButton rendered as a[href][target=_blank] (one PR) or button (stack or several PRs: opens the list)
            │     aria-label "PR #<n> - <State>: <title>" ("PR #<n>, status pending" before details load) | "…, and <k> more linked; overall <state>"
            │       | "Stack of <n> pull requests, <state>"
            │     ("PR" and "pull requests" are the host's terms: "MR" and "merge requests" on GitLab)
            │     > span.contents.font-normal.text-xs.tabular-nums.<tone> > icon size-3 + span.text-secondary-label "<n>" | "+<k+1>" | "<layers>"
            │     + its own glass popover listing the PRs (§2.12, §3.15)}
            ├─ {diff "+N −N": never rendered in this build, §7}
            └─ span  pointer-events-none ml-auto inline-flex shrink-0 items-center gap-1 > [remote machine icon][provider stack]}
```

The meta line is dropped when it would only hold trailing icons: no branch, no scratch machine label, no PR badge, no diff and no running terminal (1161–1166). The row then shrinks to 60px and the icons move to the end of the title line.

### 2.5 Slim and folder rows (settled, snoozed; boring-sidebar folders)

SidebarThreadRow.tsx:1218–1415. Variant `slim` for the snoozed and settled shelves; `folder` and `folder-multiline` inside boring-sidebar folders.

```
li[data-thread-item]  list-none [content-visibility:auto] [contain-intrinsic-size:auto_36px] (folder-multiline with meta: auto_54px)
└─ Tooltip > div[role=button][data-testid=sidebar-row-slim | sidebar-row-folder]  rowSurface
   │  slim/folder: flex h-9 items-center gap-2.5 px-2.5     multiline: flex flex-col justify-center px-2.5 py-1.5     folders: ps-9
   │  settled rows, not active: [&:not(:hover):not(:focus-within)_*]:text-secondary-label
   └─ div  contents (one-line) | flex h-6 min-w-0 items-center gap-2.5 (multiline)
      ├─ span.sr-only <title>
      ├─ {favicon wrapper (not in folders): flex shrink-0 items-center transition-opacity
      │      + (not active, or settled) opacity-40 grayscale → group-hover / group-focus-within: opacity-100 grayscale-0 > ProjectFavicon size-4}
      ├─ {draft pen}  title span  {multiline: title-line icons}  {pin}  {terminal button (not multiline)}
      ├─ {span.sr-only[role=status] "Regenerating title"}
      ├─ span  trailing slot: relative ml-auto flex h-6 min-w-8 shrink-0 items-center justify-end  (hidden during a sweep; jump-hint rule)
      │  ├─ span  inline-flex justify-end tabular-nums text-secondary-label transition-opacity + (not Woke) group-any-hover:opacity-0
      │  │  └─ snoozed: AlarmClock size-3 + wake label (text-xs text-info-foreground)
      │  │     | Woke button (row isWoke; folders: woke status) | folders: status label size-3 icon | span.text-xs > age
      │  ├─ action button (absent when unsupported):
      │  │     snoozed: "Wake thread now" > AlarmClockOff size-3 mb-px  (no tooltip)
      │  │     settled: "Un-settle thread" > Undo2 size-3.5 mb-px, tooltip (top)
      │  │     folder rows: "Settle thread" > Check size-3 (px-2, no -mr-1), tooltip (top)
      │  │     class: pointer-events-none absolute inset-y-0 right-0 [-mr-1] inline-flex cursor-pointer items-center gap-1 rounded-md
      │  │            bg-transparent px-1.5 text-xs text-muted-foreground opacity-0 transition-opacity hover:text-foreground
      │  │            focus-visible:pointer-events-auto focus-visible:opacity-100 group-any-hover:pointer-events-auto group-any-hover:opacity-100
      │  │            + (Woke) group-any-hover:static
      │  └─ {jump hint Kbd}
      └─ {multiline: meta line, as the card}
```

### 2.6 Search results

Sidebar.tsx:3199–3277, SidebarThreadRow.tsx:1633–1823.

```
TooltipProvider(delay 150, closeDelay 0, timeout 400) > DndContext(autoScroll off; drag out as context only)
└─ ul#sidebar-thread-search-results[role=listbox][aria-label="Thread search results"]  flex flex-col gap-px
   └─ li[role=presentation].list-none   (ref scrolls the highlighted one into view, block nearest)
      └─ button#sidebar-thread-search-result-<i>[role=option][tabindex=-1][aria-selected=<highlighted>][aria-current?]
         │  aria-describedby=<id>-pr  aria-label="<title>, <project>"
         │  group/sidebar-row flex min-h-9 w-full cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-1 text-left text-sm outline-none
         │  + highlighted or open: bg-sidebar-row-active text-sidebar-foreground
         │  + else: text-sidebar-muted-foreground/75 hover:bg-sidebar-row-hover hover:text-sidebar-foreground
         ├─ {ProjectFavicon size-4 shrink-0 transition-opacity + settled & not highlighted/open: opacity-40 grayscale, hover restores}
         └─ span.flex.min-w-0.flex-1.flex-col
            ├─ span.flex.min-w-0.items-center.gap-2.5
            │  ├─ span  min-w-0 flex-1 truncate  font-medium text-foreground | settled: font-normal text-secondary-label/70 group-hover:text-foreground
            │  ├─ span#<id>-pr.contents > {PR number (only when the query matches a PR)}
            │  └─ span  w-8 shrink-0 text-right text-xs text-muted-foreground/55 tabular-nums > age
            └─ {excerpt: span.truncate.text-xs.text-muted-foreground/85 > "You:" | "Agent:" + snippet with <mark> matches}
         + the row hover card of §2.12 (same content as a card row's)
p[role=status]  px-2 py-6 text-center text-xs text-sidebar-muted-foreground   "Searching thread messages…" | "No threads found"
```

Only rows listed for a message match carry the excerpt. Rows matched on title, branch or pull request have none (the palette's `SearchMatchHintLine` is not used here).

### 2.7 Draft rows

SidebarDraftBlock.tsx:26–137, 145–235. One per unsent new-thread draft, newest first, above the pinned marker.

```
li.list-none.py-0.5
└─ div[role=button][tabindex=0][data-sidebar-row="draft:<id>"][data-testid=sidebar-draft-row]  aria-label="<preview>, Unsent draft, <project>"
   │  group/sidebar-row relative w-full cursor-pointer overflow-hidden rounded-md text-left text-sidebar-foreground outline-none select-none
   │  focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring + (open) bg-sidebar-row-active | bg-warning/4 hover:bg-warning/8
   ├─ span.sr-only <preview>
   └─ div  relative z-10 h-[4.875rem] px-(--sidebar-row-content-inset) py-(--sidebar-content-inset)
      ├─ div  flex h-5 min-w-0 items-center gap-1.5
      │  ├─ SquarePen size-3 shrink-0 text-warning-foreground
      │  ├─ {ProjectFavicon size-4 shrink-0}
      │  ├─ span  min-w-0 flex-1 truncate text-xs font-medium text-secondary-label   <project>
      │  └─ span  ml-auto flex h-5 min-w-5 shrink-0 items-center justify-end
      │     └─ button "Discard draft"  pointer-events-none inline-flex items-center rounded-md px-1 text-muted-foreground opacity-0
      │           transition-opacity hover:text-foreground focus-visible:pointer-events-auto focus-visible:opacity-100
      │           group-hover:pointer-events-auto group-hover:opacity-100 > X size-3, tooltip "Discard draft" (top)
      └─ div[aria-hidden]  mt-0.5 truncate text-sm font-medium text-foreground/90   <preview>
li[aria-hidden][data-testid=sidebar-draft-divider]  mx-2.5 my-1.5 h-px list-none bg-sidebar-border/60
```

### 2.8 Shelf header and drag markers

SidebarListMarkers.tsx:13–147.

```
li[data-thread-selection-safe][data-testid=sidebar-<shelf>-header]  list-none relative mx-0.5 h-8 [mt-auto]   (dnd-kit sortable, never draggable)
└─ CollapsibleSectionHeader  button[data-sidebar-row="shelf:<working|snoozed|settled>"][aria-expanded][data-testid=sidebar-<shelf>-shelf-toggle]
   │  flex h-8 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left text-xs font-medium
   │  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring/70 + tone colour
   ├─ span.min-w-0.truncate  "Working" | "Snoozed" | "Settled"
   ├─ span[aria-hidden]  h-px min-w-2 flex-1 + tone rule + invisible (an accessory is always present)
   ├─ span.font-normal.tabular-nums  <count>
   └─ ChevronDown size-3 shrink-0 transition-transform + -rotate-90 when collapsed
li  pinned-header / pinned-divider: pointer-events-none relative mx-0.5 -mb-px h-0
└─ {while dragging: div.sidebar-drag-boundary-label  absolute inset-x-2 top-1 flex h-4 items-center gap-2
      > span shrink-0 text-xs font-medium (sidebar-foreground/80 | primary)  "Pinned" | "Active"
      > span[aria-hidden] h-px flex-1 (sidebar-foreground/25 | primary/50)}
li  active-placeholder / settled-placeholder: relative mx-0.5 -mb-px h-0  (transition none)
└─ {hint: div absolute inset-x-0 top-0 flex h-9 items-center justify-center rounded-md border border-dashed
      border-sidebar-foreground/25 text-xs text-sidebar-foreground/80 [target: border-primary/40 bg-primary/5 text-primary]  "Active" | "Settled"}
```

### 2.9 Empty state

Sidebar.tsx:3278–3322, shown when not searching, no folders, no draft rows, no pinned/active/working/snoozed rows, and either no settled thread or an active project filter.

```
div  flex flex-col items-center gap-2 px-2 py-6 text-center text-xs text-muted-foreground/60
├─ no projects:   span "No projects yet"  + button [Plus size-3 -mx-0.5] "Add project"
├─ filtered:      span "No active threads in <Project>" | "No threads in <Project> yet" ("these projects" for several) + button "Show all projects"
└─ otherwise:     "No threads yet"
button  inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-sidebar-border px-2.5 py-1 text-2xs font-medium
        text-sidebar-muted-foreground transition-colors hover:bg-sidebar-row-hover hover:text-sidebar-foreground
```

With a filter that leaves only settled threads, the message renders above the list, which still shows the Settled header.

### 2.10 Boring sidebar folders (setting `boringSidebarEnabled`, default off)

SidebarProjectFolderHeader.tsx:38–121, Sidebar.tsx:3494–3518. Active threads are grouped under one folder per project group (scope groups in sidebar order, archived projects last); pinned, working (no Working shelf in this mode), snoozed and settled stay as above.

```
li[data-testid=sidebar-project-folder]  group/project-folder relative mt-2 list-none first:mt-0
├─ button[data-sidebar-row="folder:<projectKey>"][aria-expanded]
│     flex h-8 w-full cursor-pointer items-center gap-2.5 rounded-(--control-radius) px-(--sidebar-row-content-inset) text-left outline-none
│     hover:bg-sidebar-row-hover focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring
│  ├─ ProjectFavicon size-4
│  ├─ span  min-w-0 flex-1 truncate text-sm font-medium text-sidebar-foreground/90   <project>
│  ├─ {collapsed: dot size-1.5 rounded-full + sr-only ", needs you" | ", failed" | ", done" | ", working"}
│  ├─ {collapsed with hidden rows: span text-xs tabular-nums text-sidebar-muted-foreground <n>  invisible on folder hover / action focus / coarse pointers}
│  └─ ChevronDown size-3 text-sidebar-muted-foreground transition-transform + -rotate-90 when collapsed
└─ span  absolute inset-y-0 right-6 flex items-center opacity-0 focus-within:opacity-100 group-hover:opacity-100 pointer-coarse:opacity-100
   └─ Button size=icon-xs ghost-muted [data-sidebar-action-for=folder:<key>] "New thread in <Project>" > SquarePen, tooltip same text
```

Folder rows: `variant=folder` (one line) or `folder-multiline` (setting `boringSidebarMultilineRows`), `data-sidebar-parent="folder:<key>"`, start padding 36px so titles line up with the folder name.

### 2.11 Footer

SidebarChrome.tsx:196–302, SidebarThreadUndoNotice.tsx, SidebarUpdatePill.tsx, SidebarProviderUpdatePill.tsx.

```
div[data-sidebar=footer]
├─ {provider update pill: div.group/provider-update relative flex min-h-7 w-full shrink-0 items-center overflow-hidden rounded-lg text-2xs leading-4 font-medium
│     transform-gpu transition-all duration-180 ease-drawer + tone + (exiting) pointer-events-none translate-y-1.5 opacity-0
│   ├─ {dismiss progress: absolute inset-y-0 left-0 w-full origin-left scale-x-0 transition-transform ease-linear starting:scale-x-100
│   │     border-r border-current/15 inset-shadow-2xs inset-shadow-white/8 + tone fill, duration = dismissAfterVisibleMs}
│   ├─ button[data-provider-update-main]  relative z-[1] flex min-w-0 flex-1 items-center gap-2 px-2 py-1.5 text-left
│   │     > Spinner sm | CircleCheck | TriangleAlert | Download (size-3.5) + span.wrap-break-word <title>; tooltip <description> (top)
│   └─ {button "Dismiss provider update notice"  mr-1 flex size-5 items-center justify-center rounded-md opacity-70 hover:opacity-100 > X size-3.5}}
├─ {Alert variant=warning "Intel build on Apple Silicon" (desktop)}
└─ ul[data-sidebar=menu].flex-row.items-center
   ├─ main pages:  li.shrink-0 > Tooltip(top) > SidebarMenuButton size=icon "Settings" > Settings
   │               li.shrink-0 > … "Pull requests" > GitPullRequestArrow   (when any environment supports pull requests)
   │               li.shrink-0 > … "Usage" > ChartNoAxesColumn
   │  utility pages (/projects/*, /pull-requests):  li.min-w-0.flex-1 > SidebarMenuButton > ArrowLeft + span "Back"
   ├─ {undo notice: li.@container.flex.min-w-0.flex-1 (aria-hidden + inert while fading out)
   │     > div  flex h-6 min-w-0 flex-1 flex-row-reverse flex-wrap gap-x-1 overflow-hidden text-2xs leading-6 whitespace-nowrap
   │            text-sidebar-muted-foreground transition-opacity duration-200 ease-out starting:opacity-0 [+ opacity-0 when gone]
   │       ├─ InlineButton.min-w-0.shrink > span flex h-6 flex-wrap justify-end overflow-hidden > Undo2 (-ml-4 h-6 w-4) + "<shortcut> to undo"
   │       └─ span  "<Settled|Snoozed|Unpinned|Archived|Discarded>[ <n>],"}
   └─ {update control: li.ml-auto.shrink-0 > Popover > Tooltip > button size-8 rounded-full (§3.14) | li.ml-auto.flex.flex-1 > capsule}
```

### 2.12 Hover card, context menu, snooze menu

```
ThreadHoverCardPopup = TooltipPopup variant=glass side=right align=start sideOffset=4  class max-w-80 text-left whitespace-normal
└─ div.flex.min-w-0.max-w-80.flex-col.gap-2.px-1.py-2
   ├─ div.min-w-0.truncate.text-xs.leading-tight.font-medium.text-foreground   <title>
   ├─ div.grid.gap-1.5.pl-0.5.text-xs.text-muted-foreground   lines: div.flex.min-w-0.items-center.gap-2 > icon size-3 + div.min-w-0.truncate.text-foreground/75
   │     in this order, each only when its data exists (SidebarThreadRow.tsx:282–368):
   │     · project: ProjectFavicon size-3 + display name
   │     · machine: EnvironmentMachineIcon stroke-muted-foreground + environment label (shown for the local machine too)
   │     · branch: GitBranch stroke-muted-foreground + MiddleTruncate (native title on); its text has no colour class, so it
   │       stays muted-foreground, not foreground/75
   │     · branch mismatch: items-start, text-warning, CircleAlert mt-0.5, wrapping text "You're currently checked out on another branch."
   │     · provider: ProviderInstanceIcon size-3 grayscale opacity-60 + "<model>" or "<model> · <instance name>" when the instance badge shows
   │     · handoff: ArrowRightLeft + "Handed off from <A>, <B>" (earlier providers' display names)
   │     · terminal: Terminal (teal tone) + "1 terminal process running" | "N terminal processes running"
   │     · error: CircleAlert stroke-current, whole line text-warning "Usage limit reached" (usage_limit) or text-error "Error occurred"
   └─ {footer, only where the host supports several PR links and the thread has some:
       div.border-t.border-border/60.pt-2.pl-0.5.text-xs.text-muted-foreground > ThreadPullRequestsMiniList (no click handler, so rows are plain divs)}

PR badge popover (ThreadStatusIndicators.tsx:319–354, 358–448): TooltipPopup variant=glass side=top sideOffset=0, class pointer-events-auto w-80 max-w-[calc(100vw-2rem)]
└─ div.-mx-1.max-h-[calc(var(--available-height)-0.5rem-2px)].overflow-y-auto.overscroll-contain
   └─ ul.flex.flex-col.gap-1 > li (padding-left min(depth, 3) × 12px)
      └─ a[href][target=_blank]  flex min-w-0 items-center gap-2 rounded-sm px-1 py-1 hover:bg-accent focus-visible:bg-accent
           focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring
         > PR icon size-3 <tone> + span.font-mono.tabular-nums "#<n>" + span.truncate.text-foreground/75 <title> [+ span.ml-auto.pl-1.text-3xs "stack · N" | "chain · N"]

DOM context menu (browser; contextMenuFallback.ts:379–604), appended to <body>, one div per level:
div[data-context-menu][data-level][role=menu][tabindex=-1]  dropdown-glass fixed z-[10000] min-w-32 max-w-sm overflow-hidden rounded-lg bg-clip-padding
└─ div  max-h-[calc(100vh-0.5rem)] overflow-y-auto overflow-x-hidden p-1
   ├─ div[role=separator]  mx-2 my-1 h-px bg-border                  before items with separatorBefore
   └─ button[role=menuitem|menuitemradio][tabindex=-1][aria-disabled][aria-haspopup=menu?][aria-expanded?]
        flex w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-left outline-none transition-colors
        sm:min-h-7 sm:text-sm min-h-8 text-base + text-foreground hover:bg-accent | destructive | disabled (opacity-64)
      ├─ {svg icon (stroke 2, not 1.75)  size-4.5 sm:size-4 text-muted-foreground | check / 16px spacer for radio items}
      ├─ span.min-w-0.flex-1.truncate  <label>
      ├─ {span.ms-auto.shrink-0.ps-4.text-muted-foreground.text-xs  <detail>}
      └─ {ChevronRight  -me-0.5 ms-auto size-4.5 sm:size-4 text-muted-foreground opacity-80}

Snooze menu (card rows with operate access): Menu > MenuPopup side=bottom align=end
   MenuItem "<preset label>" + MenuItemDetail "<when>" … MenuSeparator, MenuItem "Custom…"     (02-primitives §2.4)
```

## 3. Measurements

### 3.1 Search and filter row

| Item | Property | Light | Dark | M/S |
|---|---|---|---|---|
| row group | box, padding, z | 0, 52, 255×48; padding 8; `z-index: 1` | = | M |
| inner row | gap | 4 | = | M |
| search field | box | 8, 60, 151×32 (flex 1) | = | M |
| search field | padding / gap / radius | 6 8 / 8 / 6 | = | M |
| search field | text (default) | 14/20 500 `--sidebar-muted-foreground` `#71717b` | `#a3a3a3` | M |
| search field | hover | bg `--sidebar-row-hover` `#fcfcfc`, text `#18181b` (icon stays `#85868e`) | bg `#f1f3f714`, text `#f1f3f7` (icon stays `#8b8b8b`) | M |
| search field | focus-within | bg `#fcfcfc` + `inset 0 0 0 1px #1b4ed8b3` (ring 70%) | bg `#f1f3f714` + `inset 0 0 0 1px #346bf1b3` | M |
| search icon | lucide `search`, 16, at 16, 68 | `--sidebar-icon-color` `#85868e` (stays on hover and focus) | `#8b8b8b` | M |
| input | box | 40, 65.5, 111×21 (line height 1.5 × 14) | = | M |
| input | text / placeholder | 14px 500 `#18181b`; "Search" `#71717b` | `#f1f3f7`; `#a3a3a3` | M |
| input | native search decorations | cancel button, decoration, results button: `appearance: none` | = | M |
| clear button | box | 131, 66, 20×20, radius 4 (`icon-micro`, `ghost-muted`) | = | M |
| clear button | icon | lucide `x`, 12, `#71717b` | `#a3a3a3` | M |
| clear button | hover | bg `--accent` `#f4f4f5`, icon `--foreground` `#18181b` | bg `#191a1d` (sidebar-scoped accent), icon `#f1f3f7` (`ghost-muted` hovers to `text-foreground`, not `accent-foreground`) | M |
| header icon buttons | boxes | filter 163, add 191, new 219; y 62; 28×28; radius 8 | = | M |
| header icon buttons | text / icon | text `#71717bcc` (muted 80%), icon 16 `#85868e` | `#a3a3a3cc`, icon `#8b8b8b` | M |
| header icon buttons | hover | bg `#fcfcfc`, icon and text `#18181b` | bg `#f1f3f714`, `#f1f3f7` | M |
| header icon buttons | pressed (`:active`) | bg `--sidebar-row-active` `#ffffff` | `#f1f3f71c` | S |
| header icon buttons | focus-visible | `inset 0 0 0 2px #1b4ed8` | `#346bf1` | S (ring classes; same ring measured on rows) |
| header icon buttons | disabled ("New thread" with no project) | opacity 0.64, no pointer events | = | S |
| header icon buttons | transition | `width, height, padding` 150ms `cubic-bezier(0.4, 0, 0.2, 1)`; colours change instantly | = | M |
| filter icon | no scope / one project / several | `list-filter` 16 / favicon 16 / two favicons 14, the second shifted −4px with a 2px `--sidebar` ring (`#fafafa` / `#000000`) | | S |
| tooltips | side, delay | "top", flips to bottom at the top of the window (measured: New thread tooltip at 73, 94, 320×61.5, side bottom); default 600ms rest delay (a footer tooltip measured 609ms from pointerover to popup) | = | M |
| New thread tooltip | content | "New thread (Ctrl+N)" 12/16.5 `#18181b`; second line `#71717b` "New thread in current project: Shift+click (Ctrl+Shift+N)", 2px gap; default tooltip surface (02 §3.7) | dark tooltip | M (labels shown are the defaults; the measured server had overrides) |

### 3.2 Project filter popup

| Item | Light | Dark | M/S |
|---|---|---|---|
| positioner | z 130; side bottom, align start, 4px below the trigger (popup at 163, 94); flips/shifts on collision | = | M |
| frame | 288 wide (`min(18rem, --available-width)`), max height `min(--available-height, 23rem)` + 2 border (370 measured), radius 8, `overflow: hidden` | = | M |
| surface | dropdown-glass `#ffffffd5`, 1px `#18181b1a`, `blur(12px) saturate(1.14)` | `#111111d5`, 1px `#f5f5f51a`, `blur(16px) saturate(1.08)` | M |
| shadow | `0 16px 40px -18px #0000008c` | `0 18px 44px -18px #000000cc` | M |
| search block | padding 10 12 0; inner bottom border 1px `border/70` `#e4e4e7b3`, `#1b4ed8` while focused; padding-bottom 6; nudged −1px | border `#ffffff0e`, focused `#346bf1` | M (focused), S (rest) |
| search icon | `search` 16, absolute top 6 left 0, `muted/55` `#71717b8c` | `#a1a1a18c` | M |
| search input | 262×26, padding 0 9 0 20, 14px, line 26; placeholder "Search projects..."; `aria-label="Search projects"` | | M |
| empty | "No matching projects." padding 8, centred, 14/20 `#71717b` | `#a1a1a1` | M (classes), S (text) |
| list | padding 4 (scroll area viewport) | = | M |
| item | 278×32 min, padding 4 8, radius 4, text 14 `#18181b` | `#f5f5f5` | M |
| item highlighted (hover or keyboard) | bg `foreground/9%` `#18181b17` | `#f5f5f517` | M |
| item content | gap 8 | = | M |
| "All projects" icon | `folder` 16 `#71717b` | `#a1a1a1` | M |
| selected badge | 12×12 circle at right/bottom −4px, `--primary` `#1b4ed8`, ring 2px `--popover` `#ffffff`; `check` 8px stroke 4 white | `#346bf1`, ring `#111111` | M |
| label | 14 truncate | | M |
| settings button | 24×24 `icon-xs` `ghost-muted`, radius 8, `settings` 14 `#71717b`; `visibility: hidden` unless the row is highlighted (fine pointers). Out of the tab order, `aria-hidden`, native `title` "Project settings for <Project>" | `#a1a1a1` | M |
| pin slot | 24×24; the pin button (`icon-xs` `ghost-muted`, `pin` 14, `pin-off` on hover when pinned) shows when pinned or highlighted (always on coarse pointers), native `title` "Pin <Project> (<thread.pin shortcut>)" or "Unpin …", out of the tab order; without pin rights an empty 24px spacer | | M (spacer), S (button) |
| "All projects" end spacer | 24×24 (keeps the row 32 tall) | | M |
| environment badge | machine icon 14 `muted-foreground`, tooltip "On A" / "Also on A, B" (top) | | S |

### 3.3 List geometry

| Item | Value | M/S |
|---|---|---|
| list viewport | 0, 100, 255×756 at 900 tall (between the search row and the 44px footer) | M |
| scrollbars | none (`scrollbar-width: none`) | M |
| edge fade | `mask-image` on the viewport, 12px (`--fade-size: .75rem`), each edge fades over `min(12px, distance scrolled from that edge)`; no scroll padding | M |
| content padding | 0 8 8 (group `p-(--sidebar-content-inset)`, top removed by the content's first-child rule) | M |
| list | 8, 100, width 239; `gap: 1px`; `flex: 1` when it has items, so the first shelf header's `margin-top: auto` pins the shelves to the bottom of a short list | M (gap, width), S (pinning) |
| card `li` | padding 2 0; 82 tall with the meta line, 64 without; row pitch 83 / 65 | M |
| card surface | 239×78 (or 60), radius 6, `overflow: hidden` | M |
| marker `li` | 0 tall, margin 0 2 −1 (cancels the list gap) | M |
| shelf header `li` | 235×32 at x 10 (margin 0 2) | M |
| slim row | 239×36, no `li` padding; pitch 37 | M |
| "Show N more" | 239×36 | M |
| search result | 239×36 (44 with an excerpt) | M |
| `content-visibility` | `auto`, intrinsic size `auto 78px` (card), `auto 58px` (card without meta), `auto 36px` (slim), `auto 54px` (multiline folder row with meta) | M (card), S (others) |
| during a drag | `ul::after` 1px line at `bottom: -120px` keeps the scroll range from shrinking | S |

### 3.4 Card row geometry

Measured on a 256px sidebar. x positions are for a row at x 8.

| Part | Value | M/S |
|---|---|---|
| inner box | padding 8 10 (`--sidebar-content-inset`, `--sidebar-row-content-inset`), 239×78 / 60, z 10 | M |
| line 1 | 219×20 at y + 8, gap 6, items centred | M |
| project icon | 16×16 at x 18 | M |
| pin | 12×12, between icon and project name (x 40) | M |
| project name | 12/16 400, starts x 40 (58 with a pin), flex 1, ellipsis | M |
| status slot | min 32 wide, 20 tall, right edge 237 (x 205 with "2h") | M |
| line 2 | mt 4, 219×20 | M |
| title | 14/20, flex 1, `white-space: nowrap; text-overflow: ellipsis`, `dir` from the title's first strong character | M |
| line 3 | mt 2, 219×16, gap 6, 12/16 | M |
| worktree icon | 14×14 at x 18 | M |
| branch | starts x 18 (38 after a worktree icon), flex 1, MiddleTruncate (head clipped, tail kept, "…" between, 02 §5.14) | M |
| terminal button | 14×14, radius 4 | M |
| PR badge | gap 2, icon 12, number 12 tabular; measured 21.78×16 for "2" | M |
| provider stack | right edge 237, current logo 14×14 | M |
| hover controls | Settle 62.45×20 (padding 0 6, gap 4, margin-right −4, so it occupies 58.45); Snooze 24×20; Discard draft 26×20 | M (Settle), S (others) |
| slot on hover | Woke label stays and Settle sits beside it: slot 110.63 wide (52.17 + 58.45), re-measured in verification. On an ordinary row the slot grows from 32 to 58.45 and the project name gives up the difference (159 → 132.55 wide) | M |
| age label | right-aligned, `tabular-nums`, 12/16 | M |

### 3.5 Card row colours by state

| State | Surface | Text / title | Light | Dark | M/S |
|---|---|---|---|---|---|
| default (not receded) | transparent | row `--sidebar-foreground`; title 500 `foreground` | `#18181b` / `#18181b` | `#f1f3f7` / `#f1f3f7` | M |
| receded | transparent | row `sidebar-muted/75`; title 400 `foreground/75` | `#71717bbf` / `#18181bbf` | `#a3a3a3bf` / `#f1f3f7bf` | M |
| receded, title on row hover or row focus-visible | | title `foreground` | `#18181b` | `#f1f3f7` | S |
| receded working / setting up | + `opacity: 0.7`, 1 on hover or focus-within (150ms) | as receded | | | M |
| failed, seen | | row `sidebar-foreground` (failed never recedes); title 500 `foreground/95` | `#18181b` / `#18181bf2` | `#f1f3f7` / `#f1f3f7f2` | M |
| input (and request) | | never recede: row `sidebar-foreground`, title 500 `foreground` | `#18181b` / `#18181b` | `#f1f3f7` / `#f1f3f7` | M (input), S (request) |
| hover | `--sidebar-row-hover` | row `sidebar-foreground` | `#fcfcfc` | `#f1f3f714` | M |
| open thread (active) | `--sidebar-row-active` + light only `inset 0 0 0 1px` `sidebar-border/70` | title 500 foreground | `#ffffff`, ring `#e4e4e7b3` | `#f1f3f71c`, no ring | M |
| selected, or swept by an action | `--sidebar-row-selected` + light ring | | `#ffffff`, ring `#e4e4e7b3` | `#f1f3f712`, no ring | M |
| unsent draft (not the open thread) | `warning/4`, hover `warning/8` | | `#fe9a000a` / `#fe9a0014` | same | P |
| focus-visible | `inset 0 0 0 2px --ring` | | `#1b4ed8` | `#346bf1` | M |
| file dragged over | `inset 0 0 0 1px primary/70` + hover surface unless active/selected | | `#1b4ed8b3`, `#fcfcfc` | `#346bf1b3`, `#f1f3f714` | P |
| being dragged | `--sidebar` + a `--sidebar-row-active` gradient layer, `shadow-lg`, opacity 1, z 20 | sidebar-foreground | `#fafafa` + `#ffffff` = `#ffffff`; `0 10px 15px -3px #0000001a, 0 4px 6px -4px #0000001a` | `#000000` + `#f1f3f71c` ≈ `#1a1b1b`; same shadow | P |
| title regenerating | | title opacity 0.55 (150ms) | | | S |

Other card parts:

| Part | Light | Dark | M/S |
|---|---|---|---|
| project name, age, meta line, branch | `--secondary-label` `#71717b` | `#8f8f8f` | M |
| pin (static or unpin button) | `pin` 12, `muted-foreground/65` `#71717ba6`; unpin hover `#18181b` | `#a3a3a3a6`; hover `#f1f3f7` | M (static) / P |
| hover-control text (Settle, Snooze, Discard) | `muted-foreground` `#71717b`, hover `foreground` `#18181b` | `#a3a3a3`, hover `#f1f3f7` | M |
| draft pen | `warning-foreground` `#bb4d00` | `#ffb900` | P |
| worktree icon | `folder-git-2` 14, `icon-muted` `#71717b` | `#a1a1a1` | M |
| terminal button | `terminal` 14, `teal-600` `#009689`, `status-pulse` | `teal-300/90` `#46edd5e6` | M |
| remote machine icon | 14, `sidebar-muted/70` `#71717bb3` | `#a3a3a3b3` | P |
| PR badge number | `secondary-label` `#71717b` | `#8f8f8f` | M |
| PR badge icon: open (`git-pull-request-arrow`) | `emerald-600` `#009966` | `emerald-300/90` `#5ee9b5e6` | M |
| PR badge icon: draft (`git-pull-request-draft`) | `zinc-500` `#71717b` | `zinc-400/80` `#9f9fa9cc` | M |
| PR badge icon: merged (`git-merge`) | `violet-600` `#7f22fe` | `violet-300/90` `#c4b4ffe6` | M |
| PR badge icon: closed (`git-pull-request-closed`) | `red-600` `#e7000b` | `red-300/90` `#ffa2a2e6` | P |
| PR badge: stack | `layers` + layer count, aggregate tone | | S |
| PR badge: several linked | aggregate icon + "+N" (N = others + 1) | | S |
| PR badge: status pending | `git-pull-request-arrow`, `muted-foreground` | | S |
| provider logo (current) | 14, opacity 0.6; Claude fill `#d97757`, OpenAI/Codex `fill: currentColor` (the meta line's `#71717b`) | Claude `#d97757`, Codex `#8f8f8f` | M (Claude), S (Codex) |
| provider logos (previous, up to 2) | 12, opacity 0.35, `grayscale`, 6px apart, current on top (z 10) | | S |
| instance badge (several accounts of one driver, or an accent colour) | 12 min, `text-5xs` (7px) 600 initials, at right/bottom −3px, accent fill or `--card` with muted text, border `--card` | | S |

### 3.6 Status labels

Card rows show the status in line 1 (`inline-flex gap-1 font-medium`, icon 16, 12/16). Folder rows show it in the time slot with a 12px icon. Precedence in §5.4.

| Status | Label | Icon (lucide) | Colour token | Light | Dark | Timer | M/S |
|---|---|---|---|---|---|---|---|
| preparing | "Setting up" | `circle-dashed` | `--info-foreground` | `#1447e6` | `#50a2ff` | since the work started | P |
| working | "Working" ("Goal" while a provider goal is active) | `circle-dashed` | `--info-foreground` | `#1447e6` | `#50a2ff` | since the work started (measured "4h 12m") | M (both themes) |
| waiting | "Waiting" | `circle-dot-dashed` | `--muted-foreground` | `#71717b` | `#a3a3a3` | since the last run completed (seen live in both the "Ns" and the "Hh Mm" form) | M |
| approval | "Approval" | `shield-question-mark` | `--warning-foreground` | `#bb4d00` | `#ffb900` | none | P |
| input | "Input" | `message-circle-question-mark` | `indigo-600` / `indigo-300` | `#4f39f6` | `#a3b3ff` | none | M |
| request (guest) | "Request" | `user-round` | `--guest-foreground` | `#c800de` | `#f4a8ff` | none | P |
| limited | "Limited" | `circle-alert` | `--warning-foreground` | `#bb4d00` | `#ffb900` | none | P |
| failed | "Failed" | `circle-alert` | `--error-foreground` | `#c10007` | `#ff6467` | none | M |
| woke | "Woke" (button "Dismiss Woke notification", underline on hover, `role=status` on the text) | `alarm-clock` | `--warning-foreground` | `#bb4d00` | `#ffb900` | none | M |
| done (unseen completion) | "Done" | `circle-check` | `--success-foreground` | `#007a55` | `#00d492` | none | P |
| ready, seen | relative age | none | `--secondary-label` | `#71717b` | `#8f8f8f` | none | M |

Timer span: `tabular-nums`, 4px after the label, `aria-hidden`. Measured Working label 116.34 wide: icon 16 + 4 + "Working" 47.9 + 4 + "4h 12m" 44.44.

Icon names are the canonical names in the shipped lucide-react 0.564 (the rendered class is `lucide-<name>`). `ShieldQuestion` and `MessageCircleQuestion` are old aliases that render `shield-question-mark` and `message-circle-question-mark` (measured class `lucide-message-circle-question-mark`; both names found in the shipped chunks).

### 3.7 Project icons

| Kind | Rendering | Light | Dark | M/S |
|---|---|---|---|---|
| favicon image | `img` 16×16 (`object-fit: contain`, radius 25%) | | | M |
| monogram (override, or the fallback while no favicon) | 16×16 SVG `viewBox 0 0 16 16`, radius 25%, tile tint from the SVG's CSS `background-color: color-mix(in srgb, currentColor 14%, transparent)` (no `rect`), text centred at (8, 10.8), `fill: currentColor`, mono (`ui-monospace, "SF Mono"…`) 8.25px 700, `textLength` 6 (one glyph) or 12 (two), `lengthAdjust: spacingAndGlyphs` | red-600 `#e7000b` on `#e7000b24`; yellow-600 `#d08700` on `#d0870024`; amber-600 `#e17100` on `#e1710024` | red-400 `#ff6467` on `#ff646724`; yellow-400 `#fdc700` on `#fdc70024`; amber-400 `#ffb900` on `#ffb90024` (P) | M |
| lucide override | 16 (`size-4` over `size-3.5`) in the icon's colour. All lucide overrides come from one lazy chunk; until it loads, a `folder-code` icon (`size-full`, inherits the colour) stands in (seen live on first paint) | gray `#4a5565` | `#99a1af` | M |
| emoji override | 16 box, glyph at 80% of the box height | | | S |
| "No project" (scratch) | lucide `message-square-dashed`, gray | `#4a5565` | `#99a1af` | M |
| no title and no image | `folder-code` 16, `--icon-muted` | `#71717b` | `#a1a1a1` | S |

Monogram from a name (apps/web/src/projectIdentity.ts:13–26). The name is NFKC-normalised and trimmed, then split into words, which are runs of letters and digits (`[\p{L}\p{N}]+`). The first glyph is the first character of the first word, letter or digit. The second is the first digit later in that word, else the first character of the last word when there are several words, else the last character of the first word. So "web3 site" gives "W3", "alpha beta" gives "AB", "tern" gives "TN", and a one-character name doubles ("A" gives "AA"). The result is upper-cased and cut to 2 glyphs; a name with no letters or digits gives "PR". Colour: hash `index = (index × 31 + codePoint) mod 18` over the lower-cased name, into the 18 colours of projectIconColors.ts (gray, red, orange, amber, yellow, lime, green, emerald, teal, cyan, sky, blue, indigo, violet, purple, fuchsia, pink, rose). Each colour is Tailwind `<name>-600` in light and `<name>-400` in dark.

### 3.8 Slim rows (settled, snoozed)

| Part | Light | Dark | M/S |
|---|---|---|---|
| surface | 239×36, padding 0 10, gap 10, radius 6 (folders: padding-left 36) | = | M |
| favicon | 16, opacity 0.4, `grayscale(1)`; hover / focus-within: 1, `grayscale(0)` (150ms) | = | M |
| title | 14/20; 500 (400 when receded); see colour rule below | | M |
| title, settled, at rest | all descendants forced to `--secondary-label` `#71717b` | `#8f8f8f` | M |
| title, settled, hover | `foreground` `#18181b` | `#f1f3f7` | M |
| title colour rule (non-settled slim) | receded `secondary-label`; open/woke/input/request `foreground`; unread `muted-foreground`; else `secondary-label/70` `#71717bb3` | else `#8f8f8fb3` | S / P |
| hover surface | `#fcfcfc` | `#f1f3f714` | M |
| trailing slot | min 32 wide, 24 tall; age 12/16 `secondary-label`, fades out on hover (150ms) | | M |
| settled age | from `settledAt` | | M |
| snoozed wake label | `alarm-clock` 12 + "3h", 12/16 `--info-foreground` `#1447e6`, tabular | `#50a2ff` | P |
| Un-settle button | 26×24 at the slot's right edge (absolute), `undo-2` 14 with 1px bottom margin; opacity 0 → 1 on row hover (150ms). On row hover `muted-foreground` `#71717b`, on its own hover `#18181b`. At rest the settled rule forces `secondary-label` on it, which is invisible anyway | `#a3a3a3`, hover `#f1f3f7`; at rest `#8f8f8f` | M |
| Wake button | `alarm-clock-off` 12, same box rules | | S |
| "Show N more" | 239×36, 14/20 400 `#71717b`, `chevron-down` 16, gap 10, padding 0 10, radius 6 | `#a3a3a3` | M |
| "Show N more" hover | bg `#fcfcfc`, text `#18181b` | `#f1f3f714`, `#f1f3f7` | M |

### 3.9 Shelf headers

| Part | Light | Dark | M/S |
|---|---|---|---|
| button | 235×32, padding 0 8, gap 8, radius 6, 12/16 500 | = | M |
| label, count, chevron (tone `muted`) | `--sidebar-muted-foreground` `#71717b`; count 400 tabular | `#a3a3a3` | M |
| leader rule | 1px `sidebar-border/50` `#e4e4e780`, `visibility: hidden` (a count is always shown) | `#ffffff0a` | M |
| chevron | `chevron-down` 12; −90° collapsed, 0° expanded; `transition: transform, translate, scale, rotate` 150ms `cubic-bezier(0.4, 0, 0.2, 1)` | | M |
| hover | none | | M |
| focus-visible | `inset 0 0 0 2px` ring/70 `#1b4ed8b3` | `#346bf1b3` | M light, S dark |
| tone `emphasized` (Settled while dragging) | label `sidebar-foreground/80` `#18181bcc`, rule `#18181b40` | `#f1f3f7cc`, `#f1f3f740` | P |
| tone `accent` (Settled as the drop target) | `--primary` `#1b4ed8`, rule `#1b4ed880` | `#346bf1`, `#346bf180` | P |
| spacing | headers follow each other 1px apart (list gap); the first shelf has `margin-top: auto` | | M |

### 3.10 Drag markers and badges (P)

| Part | Light | Dark |
|---|---|---|
| boundary label box | absolute, inset-x 8, top 4, 16 tall, gap 8 | = |
| boundary label text | 12/16 500 `sidebar-foreground/80` `#18181bcc` | `#f1f3f7cc` |
| boundary rule | 1px flex 1 `sidebar-foreground/25` `#18181b40` | `#f1f3f740` |
| boundary label, drop target | text `#1b4ed8`, rule `primary/50` `#1b4ed880` | `#346bf1`, `#346bf180` |
| projected label height | 24 (`SIDEBAR_DRAG_LABEL_HEIGHT`, scaled by row height / 36) | = |
| placeholder hint | 36 tall, full width, radius 6, 1px dashed `#18181b40`, text 12 `#18181bcc`, centred | `#f1f3f740`, `#f1f3f7cc` |
| placeholder, drop target | border `primary/40` `#1b4ed866`, bg `primary/5` `#1b4ed80d`, text `#1b4ed8` | `#346bf166`, `#346bf10d`, `#346bf1` |
| drop verb badge | 20 tall, padding 0 6, gap 4, radius 4, 1px `primary/40` `#1b4ed866`, bg `primary/10` `#1b4ed81a`, 11/16 500 `#1b4ed8`; "Pin" 30.45 wide | `#346bf166`, `#346bf11a`, `#346bf1` |
| drop verb icons (12) | Pin `pin`, Unpin `pin-off`, Settle `circle-check`, Un-settle `undo-2`, Wake `alarm-clock-off` | = |
| context drag ghost | fixed at pointer + 12, z 100, max 288 wide, radius 8, 1px `--border`, `--popover`, padding 8 12, gap 8, 14px, `shadow-lg`; `messages-square` 16 `secondary-label`; "+N" pill (`--muted`, 12px, padding 0 6) for several threads (S) | |

### 3.11 Search results

| Part | Light | Dark | M/S |
|---|---|---|---|
| row | min 36, padding 4 10, gap 10, radius 6, 14/20 | = | M |
| row text (default) | `sidebar-muted/75` `#71717bbf` | `#a3a3a3bf` | M |
| row highlighted (keyboard or pointer) or open | `--sidebar-row-active` `#ffffff`, no ring | `#f1f3f71c` | M |
| row hover | `#fcfcfc` | `#f1f3f714` | S |
| title | 14/20 500 `#18181b`, ellipsis | `#f1f3f7` | M |
| settled title | 400 `secondary-label/70` `#71717bb3`, hover / highlighted `#18181b` (stays 400) | `#8f8f8fb3`, highlighted `#f1f3f7` | M |
| settled favicon | opacity 0.4, grayscale, unless highlighted/open or hovered | = | M |
| age | 32 wide, right, 12/16 `muted/55` `#71717b8c` | `#a3a3a38c` | M |
| PR number | gap 2, icon 12, 12px tabular, tone colour (settled: `secondary-label`, tone on hover) | | S |
| excerpt line | 193 wide, 12/16 `muted/85` `#71717bd9`, ellipsis; row grows to 44 | `#a3a3a3d9` | M |
| "You:" / "Agent:" | `--info-foreground` `#1447e6` / `--success-foreground` `#007a55` | `#50a2ff` / `#00d492` | M |
| match | `<mark>` transparent, 600, `foreground` `#18181b` | `#f1f3f7` | M |
| status paragraph | 239×64, padding 24 8, centred 12/16 `#71717b` | `#a3a3a3` | M |

### 3.12 Draft rows (P)

| Part | Light | Dark |
|---|---|---|
| surface | `warning/4` `#fe9a000a`, hover `warning/8` `#fe9a0014`; open draft `#ffffff` | same; open `#f1f3f71c` |
| size | 239×78 (fixed height, two lines), `li` padding 2 0 | = |
| pen | `square-pen` 12 `#bb4d00` | `#ffb900` |
| project name | 12/16 500 `#71717b` | `#8f8f8f` |
| preview | 14/20 500 `foreground/90`, ellipsis, mt 2 | |
| discard | `x` 12, padding 0 4, `#71717b` → `#18181b`; opacity 0 → 1 on row hover | `#a3a3a3` → `#f1f3f7` |
| divider | 1px, margin 6 10, `sidebar-border/60` `#e4e4e799` | `#ffffff0c` |

### 3.13 Empty state (P)

| Part | Light | Dark |
|---|---|---|
| block | padding 24 8, gap 8, centred, 12/16 `muted/60` `#71717b99` | `#a3a3a399` |
| button | 26 tall ("Add project" 83×26), padding 4 10, gap 6, radius 6, 1px `--sidebar-border` `#e4e4e7`, 11/16 500 `#71717b` | border `#ffffff14`, text `#a3a3a3` |
| button hover | bg `#fcfcfc`, text `#18181b` (150ms colours) | bg `#f1f3f714`, `#f1f3f7` |
| plus icon | 12, margin 0 −2 | = |

### 3.14 Footer

| Part | Light | Dark | M/S |
|---|---|---|---|
| footer | 0, 856, 255×44, padding 4 8 8, column gap 8 | = | M |
| utility row | 8, 860, 239×32, row gap 4 | = | M |
| icon buttons | 32×32 at x 8, 44, 80; radius 8; text `#71717bcc`; icon 16 `#85868e` | `#a3a3a3cc`; `#8b8b8b` | M |
| icon button hover | bg `#fcfcfc`, icon `#18181b` | `#f1f3f714`, `#f1f3f7` | M |
| icon button pressed | bg `#ffffff` | `#f1f3f71c` | S |
| icon button focus-visible | `inset 0 0 0 2px` ring | | S |
| tooltips | side top, 4px gap; "Settings" 64.25×26.5 at 5, 829.5 (5px collision padding); default 600ms delay, measured 609ms from pointerover to popup on "Usage"; dark surface `#111111`, 1px `#ffffff14` | | M |
| "Back" (utility pages) | 239×32, padding 6 10, gap 8, 14/20 500, `#71717bcc`, `arrow-left` 16 `#85868e`; hover bg `#fcfcfc`, text `#18181b` | | M |
| undo notice | 24 tall, 11/24 `#71717b`; fades 200ms `ease-out` in and out | `#a3a3a3` | P |
| update control, idle | 32×32 round, `refresh-cw` 16 `--sidebar-icon-color` `#85868e`; hover bg `#fcfcfc`; disabled opacity 0.6, `cursor: not-allowed` | `#8b8b8b` | P |
| update control, update pending | bg `--sidebar-control-surface` `#f4f4f5`, text `#18181b`, hover `#fcfcfc` | `#0a0a0a`, `#f1f3f7` | P |
| update icons | available: `download` 16 + 6px `currentColor` dot at top/right −2, 2px ring control surface; downloading: `download` 16 in a 32px ring (r 14, stroke 1.5, track `currentColor/22`, progress dash offset, round cap, `-rotate-90`, 300ms ease-out); downloaded: `rotate-cw` 16 + 10px foreground badge at −4/−4 with `check` 8 stroke 3 and a 2px background ring; everything: `circle-arrow-up` 16; checking: `refresh-cw` spinning | | S |
| update capsule | 32 tall, flex 1, full radius, `#f4f4f5`; label 12 truncate, padding-left 12; action 12 underlined "Later" / "Retry" / "Dismiss" (padding 0 12 0 6); slow: ring 1px `warning/40` `#fe9a0066`; failed: `destructive/40` `#fb2c3666` | `#0a0a0a`; `#fb414a66` | P |
| provider pill | min 28 tall, full width, radius 8, 11/16 500; loading/success `#f4f4f5` / `#18181b`; warning `warning/12` `#fe9a001f` / `#fe9a00`; error `destructive/12` `#fb2c361f` / `#fb2c36` | `#0a0a0a` / `#f1f3f7`; warning same; error `#fb414a1f` / `#fb414a` | P |
| provider pill progress | success `foreground/8` `#18181b14`, right border `current/15` | `#f1f3f714` | P |
| update confirm dialog | AlertDialog (02 primitives). Title "Update N machine(s)?" unless the caller passes one; description "T3 Code `<version>`" (mono 12) and/or "N provider update(s)", plus " · Checking for more…" while checking; a Machine / T3 Code / Providers grid (12px muted header row, rows with `border-border/50` dividers, machine names MiddleTruncate at most 128 wide, "restarts last" under this app, "N running" in `text-warning`); an "N skipped" collapsible; footer note "N running thread(s) will be interrupted…" and buttons Cancel (outline) and Update (UpdateEverythingDialog.tsx:100–250) | | S |

### 3.15 Hover card

Measured on a four-line card (project, machine, branch, model): popup 251, 439, 234.17×131 for a row at 8, 439, 239×78.

| Part | Light | Dark | M/S |
|---|---|---|---|
| placement | right of the row, top-aligned, 4px gap (x = row right + 4) | = | M |
| positioner | z 140, `pointer-events: none`, max width `--available-width`; moves with `top, left, right, bottom, transform` 150ms (`data-instant`: none) | = | M |
| surface | glass `#ffffffd5`, 1px `#18181b1a`, radius 6, `blur(12px) saturate(1.14)` | `#111111d5`, `#f5f5f51a`, `blur(16px) saturate(1.08)` | M |
| shadow | `0 20px 25px -5px #00000040, 0 8px 10px -6px #00000040` | same | M |
| width | content, max 320, text left, wraps | = | M |
| padding | viewport 4 8; card 8 4; gap 8 | = | M |
| title | 12/15 500 `#18181b`, ellipsis | `#f5f5f5` | M |
| detail grid | 12/16 `#71717b`, row gap 6, padding-left 2 | `#a1a1a1` | M |
| line | gap 8, icon 12, text `foreground/75` `#18181bbf`, ellipsis. The branch line is the exception: its MiddleTruncate has no colour class, so it keeps the grid's `muted-foreground` `#71717b` | `#f5f5f5bf`; branch `#a1a1a1` | M |
| line icons | favicon 12; machine and `git-branch` and `arrow-right-left` with stroke `muted-foreground`; provider logo grayscale 0.6; `terminal` teal; `circle-alert` warning `#fe9a00` / error `#fb2c36` | `#fb414a` | M / P |
| footer | border-top 1px `border/60` `#e4e4e799`, padding 8 0 0 2, 12/16 `#71717b`; PR mini list. Only where the host supports several PR links and the thread has at least one | `#ffffff0c` | M light, P dark |
| PR mini list in the card | `ul` gap 4; each item is a plain `div` (no link, no padding, no hover), 16 tall: icon 12 in the state tone, "#123" mono tabular, title `foreground/75` with ellipsis, gap 8; nested chain indent 12px per level (max 3); "stack · N" / "chain · N" 10px. The card's popup has `pointer-events: none`, so nothing in it is clickable | | M |
| height | no max height and no scroller. A thread with 58 linked PRs gave a 320×1304 card: Base UI clamped its top to 5px and it ran about 400px past the bottom of the 900px window | = | M |
| open timing | 150ms rest inside the list's provider (measured 163ms from pointerover); moving to the next row within 400ms swaps the card at once (`data-instant="delay"`, measured on the first frame); closes on leave with no delay | | M |

PR badge popover (hovering the meta line's PR badge; structure in §2.12):

| Part | Light | Dark | M/S |
|---|---|---|---|
| surface | the hover card's glass: `#ffffffd5`, 1px `#18181b1a`, radius 6, same blur and shadow; `pointer-events: auto` so its links work | `#111111d5`, `#f5f5f51a` | M light, S dark |
| size | 320 wide (`w-80`, at most viewport − 32), as tall as the list up to the available height. Measured 320×890 at 217, 5 for a badge at x 179.66–217: side top did not fit, so it flipped to the right with no offset | = | M |
| scroller | viewport padding 4 8; inner `-mx-1` scroller, max height `--available-height` − 10, `overscroll-behavior: contain` | = | M |
| item | link 24.5 tall (12px text at the popup's 16.5 line height plus padding 4), radius 4, gap 8; hover and focus-visible bg `--accent`, focus-visible 2px `--ring` outline inset 2 | = | M (box), S (states) |
| item parts | icon 12 in the state tone (open `#009966`, merged `#7f22fe`, closed and draft as §3.5); "#N" mono tabular `#18181b`; title `foreground/75` `#18181bbf` with ellipsis | as §3.5; `#f5f5f5`, `#f5f5f5bf` | M light |
| single PR without a list | one item built from the badge's own number, URL and title | = | S |

### 3.16 Context menu

Browser (DOM fallback). In the desktop app the same items go to a native OS menu (§5.13).

| Part | Light | Dark | M/S |
|---|---|---|---|
| position | top-left at the pointer (keyboard: the row's bottom-left), clamped 4px inside the window after the first frame | = | M |
| frame | min 128, max 384; radius 8; dropdown-glass `#ffffffd5`, 1px `#18181b1a`, `blur(12px) saturate(1.14)`; **no shadow** | `#111111d5`, `#f5f5f51a`, `blur(16px) saturate(1.08)` | M |
| padding | 4 (inner scroller, max height viewport − 8) | = | M |
| row | 28 tall, padding 4 8, gap 8, radius 4, 14/20, at every width. The row's inline style (`min-height: 1.75rem; font-size: 0.875rem; line-height: 1.25rem`) beats the `min-h-8 text-base` classes meant for narrow screens, so only the icon grows below 640 | = | M (≥640), S (narrow) |
| row text | `--contrast-foreground` `#18181b` | `#f5f5f5` (root scope; the menu lives in `<body>`) | M |
| icon | 16 (18 below 640: the icon has classes only, no inline size), **stroke 2**, round caps, `muted-foreground` `#71717b`; drawn from the menu's own path table (contextMenuFallback.ts:5–167), not lucide classes | `#a1a1a1` | M |
| highlighted (hover or focus) | bg `--accent` `#f4f4f5`, text `accent-foreground` `#18181b`, icon stays `#71717b`; colours fade over 150ms `cubic-bezier(0.4, 0, 0.2, 1)` (`transition-colors`) | `#ffffff0a`, `#f5f5f5`, icon `#a1a1a1` | M |
| destructive leaf | text and icon `destructive-foreground` `#c10007`; highlighted bg `destructive/10` | `#ff6467` | S |
| disabled | `muted-foreground`, opacity 0.64, no pointer events | `#a1a1a1`, 0.64 | M |
| separator | 1px `--contrast-border` `#e4e4e7`, margin 4 8 | `#ffffff14` | M |
| detail | margin-start auto, padding-start 16, 12px muted | | S |
| submenu chevron | `chevron-right` 16, muted, opacity 0.8, margin-end −2 | | M |
| submenu | at row right + 4, row top; flipped to row left − width − 4 when it would overflow | | M |
| measured thread menu | 207.02×401: 13 rows and 3 separators, 13 × 28 + 3 × 9 + 8 padding + 2 border. The width follows the longest label: "New thread on <branch>" with a long branch gave 335.94 | | M |
| measured Copy submenu | 128×122 (min width, 4 rows) at the Copy row's right + 4 and top, opened by hovering the row | | M |

### 3.17 Rename input (P)

| Part | Light | Dark |
|---|---|---|
| box | replaces the title, flex 1, 22 tall, padding 0 4, radius 4, 1px border | = |
| colours | text `card-foreground` `#18181b`, bg `--card` `#ffffff`, border `--input` `#d4d4d8`; focused border `foreground` `#18181b` (S) | `#f1f3f7`, `#000000`, `#ffffff2e`; focused `#f1f3f7` (S) |
| font | 14px 500 | = |

### 3.18 Folder header (boring sidebar, P / S)

| Part | Light | Dark |
|---|---|---|
| button | full width × 32, padding 0 10, gap 10, radius 8; hover `#fcfcfc` | `#f1f3f714` |
| name | 14/20 500 `sidebar-foreground/90` `#18181be6` | `#f1f3f7e6` |
| dot | 6×6 circle: attention `--warning` `#fe9a00`, failed `--error` `#fb2c36`, done `--success` `#00bc7d`, working `--info` `#2b7fff` | failed `#fb414a`, others same |
| count | 12/16 tabular `#71717b` | `#a3a3a3` |
| chevron | 12 `#71717b`, −90° when collapsed | `#a3a3a3` |
| new-thread button | 24×24 `icon-xs` `ghost-muted`, `square-pen` 14, 24px from the right edge | |
| spacing | 8px above each folder except the first | |

### 3.19 Focus indicators

| Element | Indicator | Light | Dark | M/S |
|---|---|---|---|---|
| rows, draft rows, folder headers, the Woke, Unpin and terminal buttons | `inset 0 0 0 2px --ring` | `#1b4ed8` | `#346bf1` | M (rows in both themes, Woke) |
| shelf headers | 2px inset ring at 70% | `#1b4ed8b3` | `#346bf1b3` | M light, S dark |
| search field | 1px inset ring at 70% (focus-within) | `#1b4ed8b3` | `#346bf1b3` | M |
| header and footer buttons | 2px inset ring | | | S |
| "Show N more" | no ring or outline classes: Chrome's `outline: auto`, `outline-color` ring/50, offset −2 | `#1b4ed880` | `#346bf180` | M dark |
| Settle / Snooze / Discard / Un-settle / Wake buttons | no ring classes: Chrome's `outline: auto` in `outline-color` ring/50, offset −2: a rounded 2px ring (`#1b4ed880`, painted `#8da6eb` on white) with a 1px white halo | | | M |
| PR badge (InlineButton) | `outline: 2px solid --ring` | | | S |
| list viewport | never focused (`tabindex=-1`) | | | M |

### 3.20 Icon inventory

| Where | lucide name (size) |
|---|---|
| search row | `search` (16), `x` (12), `list-filter` (16), `folder-plus` (16), `square-pen` (16) |
| filter popup | `search` (16), `folder` (16), `check` (8, stroke 4), `settings` (14), `pin` / `pin-off` (14) |
| card line 1 | `square-pen` (12), `pin` / `pin-off` (12), status icons (16; 12 in folder rows): `circle-dashed`, `circle-dot-dashed`, `shield-question-mark`, `message-circle-question-mark`, `user-round`, `circle-alert`, `circle-check`, `alarm-clock` |
| card controls | `check` (14), `clock` (12), `x` (14) |
| meta line | `folder-git-2` (14), `terminal` (14), PR: `git-pull-request-arrow`, `git-pull-request-draft`, `git-pull-request-closed`, `git-merge`, `layers` (12); machines: `server`, `cloud`, `monitor`, `laptop` + custom Linux / mini PC / workstation glyphs (14 remote, 12 scratch) |
| slim rows | `undo-2` (14), `alarm-clock-off` (12), `alarm-clock` (12), `check` (12, folder rows) |
| shelves, more | `chevron-down` (12 / 16) |
| drag | `pin`, `pin-off`, `circle-check`, `undo-2`, `alarm-clock-off` (12); ghost `messages-square` (16) |
| hover card | `git-branch`, `circle-alert`, `arrow-right-left`, `terminal` (12) |
| empty state, folder drop | `plus` (12), `folder-plus` (20) |
| footer | `settings`, `git-pull-request-arrow`, `chart-no-axes-column`, `arrow-left`, `undo-2` (16); update: `refresh-cw`, `download`, `rotate-cw`, `circle-arrow-up` (16), `check` (8); provider pill: `circle-check`, `triangle-alert`, `download`, `x` (14) |
| context menu (stroke 2) | `message-square-plus`, `pin`, `pin-off`, `circle-check`, `clock`, `pencil`, `refresh-cw`, `mail-open`, `user-plus`, `folder-tree`, `folder-plus`, `timer`, `copy`, `folder`, `git-branch`, `message-square`, `hash`, `settings`, `archive`, `trash`, `chevron-right`, `check` (16) |
| header (03-shell) | `panel-left-close` / `panel-left` (16); desktop back/forward `arrow-left` / `arrow-right` (14) |
| project icon | any lucide name the user picks (16 in rows, 12 in the hover card); `message-square-dashed` for "No project"; `folder-code` while the lucide chunk loads and for a project with no title and no image |
| not lucide | Claude and OpenAI logos (Icons.tsx:405–431, 14 / 12); project favicons; monograms |

## 4. States

| Element | State | Trigger | Look | M/S |
|---|---|---|---|---|
| search field | default / hover / focus-within | pointer, focus | §3.1 | M |
| search field | searching | trimmed query non-empty | clear button appears; results replace the list; `aria-expanded` true when results exist | M |
| search field | composing (IME) | `isComposing` / keyCode 229 | keys ignored | S |
| filter trigger | scoped | 1 or 2+ projects chosen | favicon(s) instead of `list-filter`; label lists the projects | S |
| filter popup | open / highlighted / selected / empty | | §3.2 | M |
| new thread | disabled | no project at all | opacity 0.64 | S |
| card row | default / receded / hover / active / selected / focus-visible / draft / dragging / file over / regenerating | §5.4, §5.11 | §3.5 | M / P |
| card row | status | §5.4 | §3.6 | M / P |
| card row | woke | snooze woke after the last visit | Woke button stays on hover, Settle appears beside it | M |
| card row | hover (fine pointer) | `@media (any-hover: hover)` + `:hover` | label/age hidden, controls shown, project name shrinks | M |
| card row | control focused by keyboard | `has-[:focus-visible]` in the slot | controls shown, label hidden, without hover | M |
| card row | snooze menu open | | controls pinned visible, label hidden, hover card off | S |
| card row | jump hints | desktop, jump modifier held 200ms | slot shows only the Kbd | S |
| card row | read-only environment | no operate scope | no drag, no snooze button, pin is static, rename refused; menu writes disabled | M |
| card row | removed checkout | live git status says not a repo | branch and worktree hidden (meta line may vanish) | S |
| card row | branch mismatch | local mode, checkout on another branch | hover card warning line | S |
| card row | renaming | F2 or a double-click, with write access | the rename input of §3.17 replaces the title; Enter commits, Escape cancels, blur commits (§5.11) | S |
| card row | project icon loading | a lucide override before its chunk arrives | `folder-code` in the override's colour | M |
| PR badge | hover | 150ms rest (list provider) | glass popover with the PR list (§3.15) | M |
| slim row | settled at rest / hover | | §3.8 | M |
| slim row | snoozed | | wake label, Wake button | S |
| shelf header | collapsed / expanded | click, ArrowLeft/Right | chevron −90° / 0°; rows hidden except the open thread | M |
| shelf header | dragging / drop target | | tones emphasized / accent | P |
| "Show N more" | idle / loading / failed | older settled pages | "Show N more" / "Loading" (`aria-disabled`, cursor default) / "Could not load older threads. Retry" | M (idle) |
| list | empty | §2.9 | empty block | S |
| list | search pending / no results | | status paragraph | M |
| list | drag in progress | | markers visible, list motion suspended, other rows shift (§6) | S |
| list | action sweep | pointer held on Settle/Un-settle/Wake and moved | swept rows get the selected surface and a verb badge; all list children `pointer-events: none` | S |
| list | file drag over the sidebar (not a row) | | overlay "Drop folders to add projects" (desktop) / "Dropping folders needs the desktop app" | P |
| folder | collapsed with hidden threads | | count + status dot | S |
| footer | utility page | `/projects/*`, `/pull-requests` | single "Back" button | M |
| footer | undo notice | an undoable park in the last 5s | text + undo button | S |
| footer | update states | desktop updater | §3.14 | S |
| hover card | open / closed | 150ms rest; instant hand-off to the next row within 400ms | §3.15 | M |
| hover card | many linked PRs | footer list longer than the window | the card overflows the window bottom; T3 does not cap it | M |
| context menu | open / submenu / highlighted / disabled | | §3.16 | M |

## 5. Behaviour

### 5.1 Data each row needs

From `EnvironmentThreadShell` (client-runtime models.ts:143–208) and the stores the row reads:

- identity: `environmentId`, `id`, `projectId`, `title`, `titleRegeneration`, `lineage` (parent, relationship), the set of tasks delegated to listed parents;
- placement: `archivedAt`, `deletedAt`, `pinnedAt`, `pinOrderKey`, `activeOrderKey`, `settledOverride`, `settledAt`, `unsettledAt`, `snoozedUntil`, `snoozedAt`, `autoSettleDisabledAt`, `createdAt`, `updatedAt`;
- activity: `runtime` (`status`, `activeRunId`, `activityStartedAt`, `providerInstanceId`, `lastError`, `lastErrorClass`), `latestRun` (`runId`, `status`, `requestedAt`, `startedAt`, `completedAt`), `latestUserMessageAt`, `latestUserAuthoredMessageAt`, `blockedOnThreadWait`, `pendingBackgroundTasks`, `goal.status`;
- attention: `hasPendingApprovals`, `hasPendingUserInput`, `pendingUserInputAt`, `pendingSubagentRequest`, `pendingGuestMessageCount`, `interactionMode`, `hasActionableProposedPlan`, `source.pendingRuntimeRequest`;
- visits: server `lastVisitedAt` (undefined on old servers), else a local watermark per thread;
- checkout: `branch`, `worktreePath`, plus a live git status for the row's cwd (`refName`, `isRepo`);
- providers: `providerInstanceHistory`, `modelSelection` (`instanceId`, `model`), the environment's provider entries (driver, display name, accent colour, enabled, models);
- pull requests: `pullRequests` (links with snapshots), `linkedPullRequest`, `branchPullRequest`, and whether the environment supports multiple links;
- elsewhere: running terminal ids for the thread, whether the thread has an unsent composer draft, the project (display name, favicon, icon override, workspace root, scratch), the environment (label, machine kind, primary or remote, connected, config loaded, capabilities, snooze presets), multi-selection membership.

Rows near the viewport (within 160px of the scroller, or the open row) lease live subscriptions for PR details and git status; others render from the shell alone (Sidebar.logic.ts:87–110).

### 5.2 Sections

For each visible thread (Sidebar.tsx:615–715):

1. Visible: not archived, not a child shown under its parent (subagents, side chats, delegated tasks), and inside the project scope if one is set (Sidebar.logic.ts:612–632).
2. Section, first match wins (Sidebar.logic.ts:194–203): **snoozed** (`snoozedUntil` in the future and the thread has not raised its hand, config loaded), **settled** (`settledOverride == "settled"`, config loaded), **pinned** (`pinnedAt` set), else **active**.
3. With the Working shelf on (default, and not in boring mode), an active thread goes to **working** while it is working: an active runtime (preparing, queued, starting, running, waiting) or an idle runtime with background work, and no pending approval, input or subagent request, and not a plan-ready stop (threadInbox.ts:17–31). A thread that stops working stays on the shelf for 2s (`WORKING_EXIT_GRACE_MS`) unless it became actionable, moved, was parked, or its run ended more than 60s ago (threadInbox.ts:134–273).
4. "Raised hand" ends a snooze early: a pending approval or non-input request, a new question asked after snoozing, a subagent request, a waiting guest message, or a failure after snoozing (threadSettled.ts:95–117).
5. A thread its own client is launching reads as `preparing` until its first run lands (threadInbox.ts:37–50), so it does not appear twice.
6. An open child thread (subagent, side chat) has no row; its nearest listed ancestor shows as the open row (`aria-current`), but not as the "route thread" for drafts and PR clicks (Sidebar.logic.ts:588–606).

### 5.3 Sorting

| Section | Order | Source |
|---|---|---|
| pinned | keyed rows by `pinOrderKey` (lexicographic, base-26 digits), then keyless rows by `createdAt` newest first; ties by id then environment | threadSort.ts:268–301 |
| active, Working shelf on | newest of `createdAt`, `unsettledAt`, `latestRun.requestedAt`, `latestRun.completedAt` and the time it left the Working shelf | threadInbox.ts:57–74 |
| active, Working shelf off | keyless rows first by `max(createdAt, unsettledAt)` newest first, then keyed rows by `activeOrderKey` | threadSort.ts:303–336 |
| active, boring folders | newest of `createdAt` and the latest user-authored message (`latestRun.requestedAt` on old servers) | threadInbox.ts:81–96 |
| working | same as boring folders | Sidebar.tsx:705 |
| snoozed | `snoozedUntil` soonest first | Sidebar.tsx:706–710 |
| settled | `settledAt` (else the activity time) newest first, ties by id ascending | threadSort.ts:54–69 |
| draft rows | session `createdAt` newest first | SidebarDraftBlock.tsx:204 |
| filter popup | pinned projects first, then the rest, both in the sidebar project order (default `updated_at`: the newest user message among the project's visible threads, else the project's own update time; ties by title) | projectGrouping.ts:251–256; Sidebar.logic.ts:1350–1428; threadSort.ts:82–119 |

`orderedThreadKeys` (keyboard traversal, jump numbers, ranges, "advance") = pinned, active (or folder rows), visible working, visible snoozed, rendered settled (Sidebar.tsx:546–554).

### 5.4 Status, unread, woke and recede

`resolveSidebarThreadStatus` (Sidebar.logic.ts:962–992), first match wins: approval (pending approval or subagent approval) → input (pending input or subagent input) → request (guest messages waiting) → waiting (running but blocked on a child-thread wait) → preparing → working (queued, starting, running, waiting runtime) → waiting (idle runtime) → limited (failed with `usage_limit`) / failed → ready.

The label (SidebarThreadRow.tsx:745–805) follows the status; for `ready` it is "Woke" when the thread woke from a snooze after the last visit and is not settled, else "Done" when unread, else no label (the age shows).

- A thread is unread (`hasUnseenCompletion`) when its latest run's `completedAt` is after `lastVisitedAt`. A thread never visited is not unread (Sidebar.logic.ts:799–808). tern currently marks those unread.
- A thread is woke while `lastVisitedAt` is older than `threadWokeAt`, which is the scheduled wake once it has passed, or the moment the thread raised its hand (threadSettled.ts:168–197; SidebarThreadRow.tsx:733–737). Clicking "Woke" acknowledges it by recording a visit at the wake time. It does not open the thread.
- A row recedes (`shouldRecedeSidebarThread`, Sidebar.logic.ts:928–950) always when preparing, working or waiting, and when ready or approval but neither unread nor woke. It never recedes when it is the open row, a selected row, input, request, failed or limited.

### 5.5 Formats

| Value | Rule | Source |
|---|---|---|
| row age | elapsed floored: < 60s "now", < 60m "Nm", < 24h "Nh", else "Nd" (no weeks, no cap). The clock is not `Date.now()` but the start of the current UTC minute from the shared minute tick (`YYYY-MM-DDTHH:mm`, wall-clock aligned), and a negative elapsed counts as 0. So a thread active a moment ago reads "now" for one to two minutes, and every age on screen changes in the same frame at a minute boundary | timestampFormat.ts:262–285; SidebarThreadRow.tsx:139–153; clock.ts:68–71 |
| age source, card and search rows | latest of `latestUserMessageAt`, `latestRun.requestedAt`, `startedAt`, `completedAt`; else `updatedAt` | threadSort.ts:33–52 |
| age source, settled rows | `settledAt`, else the activity time | threadSort.ts:26–31 |
| working timer | elapsed floored: < 60s "Ns", < 60m "Nm", else "Hh Mm" (e.g. "4h 12m"); hours never roll into days ("26h 3m"). The clock is the second tick (epoch ms floored to the second, wall-clock aligned), re-rendering only the timer span | models.ts:401–407; SidebarThreadRow.tsx:166–179; clock.ts:62–65 |
| working start | `runtime.activityStartedAt` whenever the server sends the field, even as null (then no timer). Only an older server that omits it falls back to the latest run's `startedAt`, else `requestedAt`, and only while that run is open and is the runtime's active run | models.ts:384–399 |
| waiting timer | since `latestRun.completedAt` | SidebarThreadRow.tsx:1529–1532 |
| snoozed wake label | remaining, rounded up: < 1h "Nm" (min 1), < 24h "Nh", else "Nd"; "now" when due | threadSettled.ts:351–360 |
| snooze presets (default) | in 1 hour, in 3 hours, today 18:00, tomorrow 09:00, Monday 09:00; per environment from server settings | contracts settings.ts:129–135 |
| preset labels | "In <duration>" in the largest unit that divides evenly ("In 1 hour", "In 3 hours", "In 90 minutes", "In 2 days"), "This morning" / "This afternoon" / "This evening" (today, by hour < 12 / < 17), "Tomorrow", "Next week" (Monday), "Next <Day>" (7 days) or "<Day>"; a today preset whose time has passed is dropped; sorted soonest first, one per wake minute | threadSettled.ts:251–329 |
| preset detail | the wake time in the user's timestamp format (locale, 12h, 24h); day-name prefix ("Mon 9:00 AM") when a duration or "Next week" lands on another day | threadSettled.ts:294–306; Sidebar.snooze.ts:49–57 |
| counts | shelf counts are plain integers; Settled includes older settled threads the client has not loaded | Sidebar.tsx:502–503 |
| title | as stored; `dir` from its first strong character | SidebarThreadRow.tsx:1067 |
| draft preview | the prompt's first line with context references replaced by their labels and citations as plain text, else "N attachment(s)" | SidebarDraftBlock.tsx:37–54 |
| undo notice | "<Action>" for one, "<Action> N" for several of the same action; the button reads "<shortcut> to undo" or "Undo" | SidebarThreadUndoNotice.tsx:20–25; showThreadUndoNotice.ts:48–51 |
| folder dot | most urgent hidden thread: attention (input, approval, request, limited) > failed > done (ready and unread) > working (working, preparing); waiting stays quiet | Sidebar.logic.ts:1017–1037 |

### 5.6 Truncation

Titles, project names, folder names, search titles, excerpts and hover card lines end in an ellipsis (`text-overflow: ellipsis`). Branches use MiddleTruncate: keep the end, clip the start of the tail, draw "…" at the cut (02 §5.14); the hover card's branch keeps the native title, the row's does not. Search excerpts are cut by the server around the match and then truncated at the end. The project name in line 1 gives way first when the status slot grows on hover.

### 5.7 Shelves

- Working, Snoozed and Settled each persist their expansion in localStorage (`t3code:sidebar:working-expanded`, `…:snoozed-expanded`, `…:settled-expanded`, Boolean, default false) (Sidebar.tsx:332–334, 452–466).
- Collapsed: only the header, plus the open thread's row if it lives in that shelf.
- Settled shows the first 10 rows when expanded, then "Show N more" adds 25 (N = min(hidden, 25)). The count resets when the project scope changes. Older settled threads load in pages from the server while the shelf is expanded; the button shows "Loading" or the retry text meanwhile (Sidebar.tsx:470–516, 3631–3654; olderSettledShelf.ts:24–82).
- Toggling a shelf does not animate rows (§6).

### 5.8 Project filter

- The scope is a list of project-group keys, persisted in the UI state store (`sidebarProjectScopeKeys`). Empty means all projects.
- Clicking a project scopes the list to it. Shift+click adds or removes it and keeps the popup open. "All projects" clears the scope. Shift+Enter in the search field toggles the highlighted project (Sidebar.tsx:2988–3006, 3075–3080).
- Changing the scope clears the multi-selection and scrolls the list to the top (1078–1082).
- A scope whose project is missing reads as all projects once every snapshot has loaded, but the stored keys stay; an archived project's scope reads as all projects (1083–1095).
- Opening a thread the scope hides (history, palette, links) widens the scope to all projects and stores that (1134–1163).
- The popup's search filters labels by substring (the Base UI combobox filter); "All projects" is hidden while a query is typed (Sidebar.logic.ts:1171–1179). Items keep their order while the popup is open, so pinning does not move rows (Sidebar.tsx:1180–1184).
- In the popup: the thread pin shortcut (default Ctrl/Cmd+Shift+P) pins the highlighted project; ContextMenu or Shift+F10, or a right-click on an item, opens that project's settings and closes the popup (3053–3088, 3111–3113).
- The popup returns focus to the trigger only when closed from the keyboard (`finalFocus`, 3047).
- The empty-state button "Show all projects" clears the scope.

### 5.9 Search

- Typing sets the query and resets the highlight. A non-blank query replaces the list with results; Escape clears it (the field keeps focus); the clear button clears and refocuses (Sidebar.tsx:1497–1536; SidebarThreadHeader.tsx:83–97).
- Results list the threads whose title, branch or pull request match first, ranked as the command palette ranks them (no "picks"). Threads that match only in their messages follow, in the order the server ranked those messages across environments. Rows already shown keep their places while slower environments answer (Sidebar.tsx:1363–1417; Sidebar.logic.ts:1127–1159). The search covers pinned, active, working, snoozed and loaded settled threads in the current scope.
- ArrowDown and ArrowUp move the highlight and wrap around. Moving the pointer over a row highlights it. The highlight follows its thread when late results arrive, and a new query starts at the top. Enter (or a click) clears the query first, then opens the thread; for a message match it also opens find-in-thread on that match (1419–1434, 1501–1511). Verified live: ArrowUp from the first of 50 results wraps to the last, and Escape empties the field and keeps focus in it.
- The field is the combobox. Its `aria-activedescendant` points at the highlighted result's id.
- With no results the status line reads "Searching thread messages…" while the server search runs, then "No threads found". On the loaded host that took about 9s.
- Result rows can be dragged out as composer context (no reordering), and accept file drops like rows. Each result shows the card row's hover card (§2.12) on the same 150ms rule. Its age uses the activity time even for settled threads, unlike the settled slim row.
- The searchable set grows while older settled pages load (expanding Settled starts that), so a query can return more title matches a moment later.

### 5.10 Header buttons

- New thread (`square-pen`) starts a thread in the current context when Shift is held, when exactly one open project exists, or when there are no project groups. The context is the open thread's or draft's project, else the default project. Otherwise it opens the command palette's "New thread in" picker, and it opens that picker anyway if starting finds no project (Sidebar.tsx:2931–2953; Sidebar.logic.ts:842–849). It is disabled with no project. Its tooltip shows the `chat.new` shortcut (default Ctrl/Cmd+N, and Ctrl/Cmd+Shift+O also binds it), plus a second line with `chat.newLocal` (default Ctrl/Cmd+Shift+N) when a click would open the picker.
- Add project (`folder-plus`) opens the command palette in add-project mode. It is hidden, with the filter, when there is no project.
- Below 768px (mobile sheet) every navigation from the sidebar also closes the sheet.

### 5.11 Pointer on rows

- A click opens the thread, clears any multi-selection and sets the range anchor. Ctrl+click (Cmd on macOS) toggles the row's selection. Shift+click selects the range from the anchor over `orderedThreadKeys`. The second click of a double click does not navigate again, and clicks on nested links are ignored (Sidebar.tsx:1573–1592).
- A double-click without modifiers, outside buttons, links and inputs, starts a rename when the user can write (SidebarThreadRow.tsx:877–891).
- Rename (SidebarThreadRow.tsx:601–640; Sidebar.tsx:1538–1571). The input (`aria-label="Thread title"`, `dir="auto"`) mounts focused with its text selected. Enter commits, Escape cancels, and a blur commits unless Enter or Escape already did. Keys stop at the input, and IME composition (`isComposing`, keyCode 229) is ignored. Commit trims the text. An empty title cancels and raises the warning toast "Thread title cannot be empty"; an unchanged title sends nothing; a server failure raises "Failed to rename thread" with the error text.
- A right-click opens the row menu, or the multi-selection menu when the row is selected (§5.13).
- Hover, with a fine pointer, swaps the status or age for the controls and shows the hover card after 150ms.
- A click on Settle, Un-settle or Wake applies it. Holding the button and moving across rows sweeps the same action over every row of the same section between the start and the pointer (§5.16).
- "Woke" acknowledges the wake. The unpin icon unpins, after a confirmation if `confirmThreadUnpin` is on. Discard draft drops the thread's unsent draft.
- The PR badge opens the PR. While a thread is open the PR opens in the right panel, and if the row is another thread, that thread opens too. A stack, or several linked PRs, open the thread's pull-request panel instead (SidebarThreadRow.tsx:962–970, 1077–1093).
- The terminal button reveals that thread's first running terminal and opens the thread.
- Files dropped on a row attach to that thread's composer after navigating to it; folders dropped anywhere on the sidebar add projects (desktop app) (Sidebar.tsx:1467–1487; SidebarFolderDrop.tsx).

### 5.12 Keyboard

| Key | Where | Effect | Source |
|---|---|---|---|
| Tab / Shift+Tab | everywhere | rows are in the tab order (`tabindex=0`) and their inner buttons after them; the list viewport is not | SidebarThreadRow.tsx:1238, ui/sidebar.tsx:692 |
| ArrowDown / ArrowUp | a row or header (`[data-sidebar-row]`) | focus the next / previous visible row, clamped (no wrap); scrolls it into view (block nearest) | sidebarTree.ts:113–118 |
| Home / End | same | first / last row | 119–124 |
| ArrowRight | collapsed shelf or folder | expand; otherwise focus its first child row | 125–131 |
| ArrowLeft | expanded shelf or folder | collapse; otherwise focus the row's parent header | 132–138 |
| Enter / Space | row | open the thread (draft rows: open the draft) | SidebarThreadRow.tsx:873–875 |
| F2 | row, with write access | rename | 865–871 |
| ContextMenu / Shift+F10 | row | menu on key release, at the row's bottom-left; a keyboard-opened menu focuses its first enabled item | sidebarTree.ts:83–97, 147–165 |
| modifiers held | rows | tree keys ignore Alt/Ctrl/Meta/Shift | 98–106 |
| `thread.previous` / `thread.next` | window | previous / next in `orderedThreadKeys`, no wrap; with no open row: last / first. Default Ctrl/Cmd+Shift+[ / ] | Sidebar.tsx:2884–2913 |
| `thread.jump.1…9` | window, desktop app only | open the Nth thread in `orderedThreadKeys`. Default Ctrl/Cmd+1…9; hints (Kbd with the shortcut label) appear after the modifier is held 200ms and vanish on release; not while the terminal has focus | 2914–2916; Sidebar.logic.ts:711–787 |
| `chat.new`, `chat.newLocal`, `chat.newWithoutProject` | window, not while the terminal has focus | as the New thread button; new local thread; new thread without a project. Defaults Ctrl/Cmd+N (and Ctrl/Cmd+Shift+O), Ctrl/Cmd+Shift+N, Ctrl/Cmd+Alt+N | chatRouteCommands.ts:105–145; keybindings.ts:112–115 |
| `thread.settle` / `thread.pin` / `thread.undo` | window (open thread), not while the terminal has focus | settle (and advance) or un-settle; pin or unpin (unpin asks first when `confirmThreadUnpin` is on); undo the last park while the notice shows. Defaults Ctrl/Cmd+Shift+S, Ctrl/Cmd+Shift+P, Ctrl/Cmd+Z. Undo also fires inside text fields while the sidebar is visible and the notice holds the shortcut, until a non-modifier key or an input event releases it; a held key undoes once. In the open filter popup, Ctrl/Cmd+Shift+P pins the highlighted project instead | useThreadLifecycle.tsx:663–689; AppSidebarLayout.tsx:226–292; keybindings.ts:131–133 |
| Escape | search field | clear the search | Sidebar.tsx:1514–1519 |
| Escape | during a drag | cancel | Sidebar.pointer.ts:106–108 |

Window shortcuts skip events already handled, key repeats, an open command palette and an open model picker. Focus: a row that moves to another shelf remounts; the tree refocuses the same row id, else its parent header, else the row at the old index, without scrolling (sidebarTree.ts:43–61).

### 5.13 Row menus

Item lists and conditions (threadActionMenu.logic.ts:116–269; Sidebar.tsx:2593–2882). `[sep]` is `separatorBefore`. Items marked † need write access and are disabled without it (the measured state).

**Thread menu**

| # | Item | Icon | Condition | Action |
|---|---|---|---|---|
| 1 | "New thread on <branch>" | `message-square-plus` | thread has a branch | new thread on that branch and worktree |
| 2 | "Pin thread" / "Unpin thread" † | `pin` / `pin-off` | environment config loaded | pin / unpin |
| 3 | "Settle thread" / "Un-settle thread" † | `circle-check` | config loaded | settle (open thread advances, §5.15) / un-settle |
| 4 | "Snooze" › presets + [sep] "Custom…" †, or "Wake thread" † | `clock` | config loaded; Snooze disabled when the thread cannot be snoozed now (pending approval or non-input request, subagent request, guest message, or a queued start within 2 min) | snooze / wake |
| 5 | [sep] "Rename thread" † | `pencil` | always | inline rename |
| 6 | "Regenerate title" / "Regenerating…" (disabled) † | `refresh-cw` | config loaded | regenerate, toast on result |
| 7 | "Mark unread" | `mail-open` | always | rewind the visit watermark |
| 8 | "Share with guest" | `user-plus` | guest sharing granted | open the share dialog |
| 9 | "Filter by <Project>" / "Show all projects" † | `folder-tree` | project not archived | set the scope to that one project, or clear it when it is already the scope. It only changes local state, yet `threadActionRequiresOperate` does not exempt it, so it is disabled without write access (measured disabled in the read-only session) |
| 10 | "Turn into project" † | `folder-plus` | a "No project" thread with its own folder, on a capable server; disabled while running | promote; toast "Project created" |
| 11 | "Auto-settle behavior" › "Follow settings" / "Never" (radio) † | `timer` | server supports the opt-out | set auto-settle |
| 12 | [sep] "Copy" › "Path", "Branch" (with a branch), "Thread", "Thread ID" | `copy` › `folder`, `git-branch`, `message-square`, `hash` | always | copy; toast "Path copied" / "Branch copied" / "Thread ID copied" with the value; a thread with no path toasts "Path unavailable" / "This thread does not have a workspace path to copy."; "Thread" copies a thread reference (link) |
| 13 | "Project settings" | `settings` | always | open project settings |
| 14 | [sep] "Archive thread" † | `archive` | disabled while running | archive (confirm "Archive thread "<title>"?" if `confirmThreadArchive`) |
| 15 | "Delete" † (destructive) | `trash` | always | confirm "Delete thread "<title>"?\nThis permanently clears conversation history for this thread." (destructive, default on) |

The builder also knows "Find in thread" (`text-search`), but the sidebar never enables it; it appears in other callers.

**Multi-selection menu.** A right-click on a selected row opens it; it has no icons. Items: "Unpin (N)" when any selected thread is pinned, N = the pinned ones; "Settle (N)", disabled if none is unsettled; "Snooze (N)" › presets + "Custom…", present only when every selected thread has its config loaded and can be snoozed now; "Regenerate titles (N)" with N = the threads not already regenerating, or "Regenerating… (N)" (disabled) with N = the threads that support it; "Mark unread (N)"; "Delete (N)" (destructive; confirm "Delete N threads?\nThis permanently clears conversation history for these threads."). Settle, Snooze, Mark unread and Delete count the whole selection. Every item but Mark unread is disabled without write access. Snooze presets come from the first selected thread's environment. Actions clear the selection (Sidebar.tsx:2291–2540; Sidebar.logic.ts:634–657).

**Draft menu.** "Copy" › "Path", "Branch" (disabled with neither), "Project settings" (with a project), [sep] "Discard draft" (destructive, `trash`) (threadActionMenu.logic.ts:39–68).

**Folder menu** (boring sidebar). "New thread", "Copy path", "Pin project" / "Unpin project", "Project settings" (Sidebar.tsx:1266–1306).

The browser draws the DOM menu of §3.16 (contextMenuFallback.ts). The desktop app shows a native menu instead: detail text is appended in parentheses ("Tomorrow (9:00 AM)"), radio items become checkboxes, a separator is inserted before the first destructive item if none precedes it, and on macOS destructive leaves get a 12px template trash icon; other icons are dropped (ElectronMenu.ts:63–200). DOM menu keys: Arrow Up/Down move with wrap over enabled items, Home/End, ArrowRight opens a submenu (focusing its first item), ArrowLeft/Escape close the submenu (Escape on the root closes all), Enter/Space activate, Tab closes. Hovering an item focuses and highlights it and opens its submenu (hovering a leaf closes deeper submenus); a pointer-opened menu focuses the menu itself, a keyboard-opened one its first enabled item. Clicking a row that has a submenu opens it and focuses its first enabled item. Outside pointer down, another context menu, window blur, or focus leaving closes it. Closing restores focus to the element that had it (verified: Escape with focus on a root item closed both levels and focus went back to the row). The menu also swallows the mouse's Back and Forward thumb buttons, as a native menu does (contextMenuFallback.ts:300–345, 520–604).

Failed actions raise stacked error toasts ("Failed to settle thread", "Failed to pin thread", "Failed to wake thread", "Failed to delete thread"…). A write without access raises "Thread action unavailable" / "This connection cannot change one or more selected threads." (Sidebar.tsx:869–879).

### 5.14 Snooze

- Card rows show the clock button when the user can write, the server config is loaded, and `canSnooze` holds; it hides again (closing its menu) while a queued start is pending (2 min grace after the last user message) (SidebarThreadRow.tsx:930–961).
- The menu (bottom, end-aligned) lists the presets resolved when it opens, then "Custom…". A duration preset re-resolves at the moment it is picked; a set time keeps the shown wake; a passed time raises the warning "Could not snooze thread" / "That snooze time has passed. Choose another time." (Sidebar.snooze.ts:27–47).
- "Custom…" opens the Custom snooze dialog (CustomSnoozeDialog.tsx): title "Custom snooze", description "Choose when snoozed threads return to your inbox.", a toggle group "Date and time" / "Duration"; date (calendar popover from today) + time (default now + 1h), or amount (default 2) + unit (Minutes/Hours/Days, default Hours); footer "Edit snooze options" (ghost-muted, opens thread settings), "Cancel", "Snooze"; errors "Choose a valid date and time in the future." / "Enter a positive duration.".
- Snoozing the open thread advances (§5.15).

### 5.15 Parking, advancing and undo

- Settling or snoozing the open thread moves to the next card row after it in `orderedThreadKeys` (wrapping, skipping settled, snoozed and co-parked rows); if none, a new thread in the same project; if no project, home (Sidebar.tsx:1594–1617). The move happens only if the server confirms the park.
- Parks register with the undo notice (5s, grouped by action) in the footer and `thread.undo`.

### 5.16 Drag and drop

- A drag starts with a primary-button press on a row followed by more than 6px of movement (Euclidean). A release before that is a normal click. After a drag T3 swallows the next click, blocks the context menu and clears any text selection (Sidebar.pointer.ts).
- Escape, window blur, `pagehide`, a window resize, the document hiding, `pointercancel` and the row leaving the list all cancel the drag.
- Rows don't drag while being renamed, without write access, while their environment is disconnected or its config is not loaded, and while an earlier drop awaits the server.
- The row moves vertically only, never above the "Pinned" label line, and stays inside the scroller. Auto-scroll starts only within 5% of the scroller's height from an edge, at acceleration 3, and is off for context drags.
- Each section takes drops differently (Sidebar.logic.ts:363–491). Pinned pins or reorders, with fractional `pinOrderKey`s, rewriting spread keys when needed. Active unpins, un-settles or wakes, and reorders only when the Working shelf is off; with the shelf on, Active is time-ordered and a drop only changes the section. Settled settles, which also unpins and wakes. Working and Snoozed take no drops, Working rows only drag out as context, and Settled drops need settlement support.
- While dragging, the "Pinned" and "Active" boundary labels appear after a 200ms reveal, empty sections show a dashed 36px hint, the Settled header turns emphasized (accent when targeted), and the dragged row shows the destination verb ("Pin", "Unpin", "Settle", "Un-settle", "Wake"). Other rows slide to their projected places with dnd-kit's `transform 200ms ease`.
- On drop the new placement shows at once and holds until the server's state matches. A failure puts the row back and raises a toast.
- Moving the pointer left or right out of the list turns the drag into a thread-context drag. Rows stop moving, the scroll position is restored and a ghost follows the pointer. Dropping it on a composer adds the thread, or the whole selection if the row is selected, as context chips (Sidebar.tsx:1750–1793; threadContextDrag.ts).
- The action sweep works the same way from Settle, Un-settle or Wake. Past 6px of movement it selects every row of the same section between the start row and the row under the pointer, giving them the selected surface and the verb badge. Release applies the action to all of them (Sidebar.tsx:1813–1865).

### 5.17 Route reveal and scroll

On mount and on each route change the open row is scrolled into view once (block nearest), unless it is under the pointer (it was just clicked). A row in a hidden sheet waits until it has a box. Search results hold the reveal until the list returns (routeReveal.ts:35–87).

### 5.18 Footer

- Settings, Pull requests (only when an environment supports them) and Usage navigate to their pages (Settings and Usage open as modals over the last main view). On the pull-requests and project pages the row shows "Back" to the last main-app page (mainAppLocation.ts:7–45).
- The undo notice fills the free space right of the icons and never moves the list (§5.15). Its row is `flex-row-reverse`, so the undo button sits at the right end and the summary to its left: "Settled, Ctrl+Z to undo" (the summary keeps a trailing comma). A summary too long to fit beside the button wraps onto a second line that the 24px box clips, rather than truncating mid-word, and the `undo-2` icon shows only once the label has wrapped away. Screen readers hear a separate full sentence such as "Settled 2 threads".
- The Pull requests button navigates with the saved list preferences as its search params; every footer navigation closes the mobile sheet first.
- The update control and capsule appear only in the desktop app, or in a browser when servers or providers have updates ("Update everything"). States and tooltips: "Check for updates", "Checking for updates…", "Update <v> ready to download", "Downloading update (N%)", "Update <v> downloaded. Click to restart and install.", retry texts; release notes open as a hover popover when present (side top, opens on hover, 150ms close delay, also on keyboard focus). Starting "Update everything" asks first with the confirm dialog of §3.14 (SidebarUpdatePill.tsx:415–590; UpdateEverythingDialog.tsx; desktopUpdate.logic.ts:89–166).
- The provider pill shows provider update progress and results ("Updating Codex", "N providers updated", "… still needs an update", failures); success dismisses itself after 3s with a draining bar; warnings and failures have a dismiss button ("Dismiss until provider status changes"); click opens provider settings (SidebarProviderUpdatePill.tsx; ProviderUpdateLaunchNotification.logic.ts:324–451).

### 5.19 Accessibility

- Row: `role=button`, `aria-label` "<title>, <status label>, <project>" (empty parts dropped), `aria-current=page` for the open row, `aria-busy` while the title regenerates; the visible title is `aria-hidden` and an sr-only copy is read (Sidebar.logic.ts:75–85).
- The thread list is `role=presentation` (flattens VoiceOver's list boundary); results are a `listbox` of `option`s driven by the search combobox.
- Status labels are not live regions; background announcements cover them.

### 5.20 Related components listed in scope

- `ThreadCommandSubtitle` is the palette's thread subtitle (project favicon 12 + name · machine · workspace icon (`folder-git-2` for worktrees, `folder` otherwise) + branch · provider logo · "Current thread", 12px `muted/70`, dots `muted/50`). It is not drawn in the sidebar.
- `ThreadContextChip` is the composer chip a sidebar context drag produces: a thread `ContextChip` linking to the thread, with the project favicon (else `messages-square`) and the title; tooltip "Open thread", or the unreadable-machine message (composer doc).
- `ThreadRowLeadingStatus` / `ThreadRowTrailingStatus` and `resolveThreadStatusPill` (dot + "Working", "Pending Approval"…) belong to other thread lists; the sidebar rows use §3.6 instead.

## 6. Motion

| Item | Property | Duration | Easing | Condition | Source | M/S |
|---|---|---|---|---|---|---|
| list: row moved | `transform: translateY(old − new)` → 0 (WAAPI) | 150ms | `ease-out` keyword = cubic-bezier(0, 0, 0.58, 1), not Tailwind's | after a commit that changes order, sections, folder sizes or drafts; not during a drag, not when a shelf was toggled in the same commit, not under reduced motion, not on the first pass or while hidden, not when more than 40 rows enter or leave at once | Sidebar.motion.ts:1–4, 101–118, 151–155; Sidebar.tsx:2003–2026 | S |
| list: row entering | `opacity 0→1`, `translateY(−min(height, 40))` → 0, or the shift of the nearest moved row above | 150ms | `ease-out` | same | Sidebar.motion.ts:195–217 | S |
| list: row leaving | an inert clone at the old place: `opacity` (current entry progress) → 0, `translateY(0)` → `+min(height, 40)` or the riding shift; removed on finish | 150ms | `ease-out` | same | Sidebar.motion.ts:40–89, 179–183 | M (a stalled clone was seen while the tab was hidden) |
| list: interrupted motion | remaining offsets carry into the next pass | | | | Sidebar.motion.ts:26–32 | S |
| list: after a drop | rows animate from their released position | 150ms | `ease-out` | no reduced motion | Sidebar.motion.ts:226–245 | S |
| drag: peers | `transform` to the projected slot | 200ms | `ease` (cubic-bezier(0.25, 0.1, 0.25, 1)) | while sorting | shipped dnd-kit `Sn`; Sidebar.logic.ts:186–188 | S |
| drag: context drag | `transform 150ms ease-out` | 150ms | `ease-out` | context drag | SidebarThreadRow.tsx:477 | S |
| drag: boundary labels | visibility hidden → visible | 200ms | `step-end` | once per pickup | index.css:2425–2436 | S |
| receded working row | `opacity` 0.7 ↔ 1 | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | hover / focus-within; none under reduced motion | SidebarThreadRow.tsx:986–988 | M |
| title | `opacity` (regenerating 0.55) | 150ms | same | | 1044 | M |
| slim favicon, age, action buttons | `opacity` | 150ms | same | hover | 1266–1300 | M |
| card status ↔ controls | none (instant swap) | | | hover | 1483–1547 | M |
| shelf / folder chevron | `rotate` 0 ↔ −90° | 150ms | same | toggle | collapsible-section-header | M |
| shelf rows | none | | | toggle | Sidebar.tsx:2016–2026 | S |
| header and footer buttons | `width, height, padding` only; colours instant | 150ms | same | | ui/sidebar.tsx:747 | M |
| empty-state button, update button | colours | 150ms | same | hover | | S |
| terminal icon | `status-pulse`: opacity 1 (0–40%) → 0.5 (50–90%) → 1, `steps(6)`, all rows in phase | 2s | steps | `motion-safe` | ThreadStatusIndicators.tsx:544–600; 01 §6 | M |
| hover card | `scale 0.98 → 1`, `opacity 0 → 1` in, reverse out. A hand-off to the next row within 400ms carries `data-instant="delay"`, which sets the popup's duration to 0 and the positioner's transition to none, so the card jumps to the new row without animating | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | open after the rest delay, close on leave | 02 §6; tooltip.tsx:109, 143 | M |
| filter popup | `scale`, `opacity` | 150ms | same | | 02 §6 | M |
| DOM context menu | no open or close animation; item highlight `color, background-color, border-color` fade (`transition-colors`) | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | hover / focus | contextMenuFallback.ts:435–447 | M |
| undo notice | `opacity` 0 → 1 on mount, 1 → 0 then unmount | 200ms | `cubic-bezier(0, 0, 0.2, 1)` (Tailwind `ease-out`) | | SidebarThreadUndoNotice.tsx:46–49 | S |
| provider pill | exit `translateY(6px)` + `opacity 0`; progress `scaleX(1 → 0)` over the visible time | 180ms; 3000ms | `cubic-bezier(0.32, 0.72, 0, 1)`; linear | | SidebarProviderUpdatePill.tsx:15–163 | S |
| update download ring | `stroke-dashoffset` | 300ms | `ease-out` | | DesktopUpdateStatusIcon.tsx:84 | S |
| update check | `refresh-cw` spin, latched to finish its turn | 1s | linear | `motion-safe` | DesktopUpdateStatusIcon.tsx:21–41 | S |
| timers | text changes each second (timer) and each minute (age) | | | | | M |

## 7. Gaps

- **Write-gated UI.** The read-only session could not show dragging, the snooze clock and its menu, the unpin button, rename, the action sweep, or enabled write items in the menus (Pin, Settle, Snooze, Rename, Regenerate, Archive, Delete) and their destructive highlight. Their colours are probes with the real classes and their geometry comes from source.
- **Row states missing from the data.** No row was in the approval, request or limited state, no card showed Done (one did, inside a stalled fade-out clone), and there were no snoozed rows, draft rows, rows with an unsent draft, remote-machine icons, provider instance badges, stacked PR badges, Codex logos, or branch-mismatch and handoff lines. All of these are P or S. Input and Failed rows, the several-PRs "+N" badge and its popover appeared during verification and are now M.
- **Boring sidebar folders.** The setting was off, so folders are source and probes only.
- **Desktop-only parts.** Native context menus, back/forward buttons, jump hints, the update control and capsule, the Apple-silicon warning and folder drops that add projects are all source only.
- **List motion.** Nothing reordered during the session, so the FLIP values come from source. dnd-kit's 200ms `ease` is read from the shipped bundle.
- **Tooltip timing.** Timed once each with the tab in front: the row hover card opened 163ms after pointerover (150ms delay) and a footer tooltip 609ms after (600ms). The header tooltips share the footer's default and were not timed separately.
- **Filter popup focus.** Resolved. With the tab in front, Escape closed the popup and focus went back to the trigger, as `finalFocus` says. The first pass saw focus land in the composer only because the tab was in the background.
- **Dark values not read live.** Resolved for the search field's hover, footer button hover, the DOM menu's highlight and the clear button's hover, all measured in dark during verification. Still derived in dark: the shelf header focus ring, the hover card's PR footer border, the PR badge popover.
- **Diff stats.** They are dead code in this build: `latestRunDiff` returns null, so the `+N −N` spans with `diff-addition/deletion-foreground` never render.
- **Keybinding labels.** The measured server overrides several shortcuts, so the labels here are the defaults.
- **Fonts.** Widths such as "Settle" 62.45 and "Working" 47.9 are Inter. SF Pro differs by a few px.
- **Narrow windows.** Below 768px the same content sits in the sheet (03-shell §3.9). I didn't re-measure it here.
- **Hover pixels.** Hover and focus colours are computed values. Only the Settle focus ring was checked in pixels.
- **Background tab.** Chrome stalls an unfocused tab: the colour scheme switch did not apply (no `.dark` class) and popups closed on window blur until the tab was brought to the front. Every value here was read with the tab in front; a reading taken in the background is wrong, not a T3 state.

## 8. Port notes for Go/Gio

Replaces in tern: the sidebar parts of `app/ui/screens.go` (`sidebar`, `shelf`, `statusChip`, `threadRow`, from line 106), `app/ui/details.go` `sidebarFooter`, the sidebar fields of `app/ui/model.go` (`ThreadRow`, `Scene.Pinned/Active/Snoozed/Settled/SettledRows/SidebarNote/Search/Filter/ShowSnoozed/ShowSettled`), and in `app/core/scene.go` `age`, `duration`, `threadState`, `unread`, `row`, `sidebar`, `shelve`, `fillSidebar` (lines 38–240).

1. **Model in core, drawing in ui.** Build a `SidebarModel` in core each frame: sections (pinned, active or folders, working, snoozed, settled), `orderedKeys`, shelf states, settled visible count, search state and a per-row view model: `Variant` (card, slim, folder, folderMultiline), `Status` (§5.4, with label, icon, colour token, timer start), `Recede`, `Unread`, `Woke`, `Active`, `Selected`, `HasDraft`, `ProjectName`, `ProjectIcon`, `Pinned`, `Age` source time, `Branch`, `Worktree`, `ScratchMachine`, `PR` (number, state, stack/others), `Terminals`, `Remote`, `ProviderStack` (≤ 3), `CanWrite`, `CanSnooze`, `SettleSupported`. Keep the formatting rules of §5.5 in core and test them.
2. **Fix tern's current rules.** `age()` must not switch to weeks ("Nd" forever); the working timer is "Ns" / "Nm" / "Hh Mm" (no seconds after a minute); never-visited threads are not unread; there are no "PINNED"/"ACTIVE" headings at rest; shelves are "Working", "Snoozed", "Settled" in sentence case with a right-aligned count and chevron; the search placeholder is "Search"; New thread is `square-pen`, not `plus`; rows are 78/60 (card) and 36 (slim), not 46.
3. **Data tern must add.** `t3.Thread` (`app/t3/types.go`) already carries the pin and active order keys, `settledAt`, `snoozedUntil`, `lastVisitedAt`, `latestRunCompletedAt`, the activity run status and start, `pendingRuntimeRequest`, `lineage`, `modelSelection`, `interactionMode`, `branch` and `worktreePath`. For §5.1 it still lacks the settle override (`settledOverride`), `unsettledAt`, `snoozedAt`, `autoSettleDisabledAt`, the thread status and `lastErrorClass` (failed versus usage-limited), the rest of the latest run (`runId`, `status`, `requestedAt`, `startedAt`), `latestUserAuthoredMessageAt`, `pendingUserInputAt`, `pendingSubagentRequest`, `pendingGuestMessageCount`, `blockedOnThreadWait`, `pendingBackgroundTasks`, `hasActionableProposedPlan`, `goal`, the provider instance id and history, `pullRequests` with snapshots (plus the linked and branch PR references) and `titleRegeneration`. Outside the thread it needs per-environment provider entries, the operate scope, server capabilities and snooze presets, running terminal ids per thread, unsent drafts, project icon overrides and favicon bytes, the server thread search with snippets, older settled pages, and live git status for rows within 160px of the viewport. For the PR badge and its popover it also needs each link's host kind (GitHub "PR" versus GitLab "MR" wording), chain and stack structure, and the snapshot's title and state.
4. **List layout.** One scrolling `layout.List` over the item sequence of §2.3 with a 1px gap, inset 8 left/right/bottom. Markers are zero-height items. Make the list at least as tall as the viewport and add the remaining space above the first shelf header so short lists keep the shelves at the bottom. Paint a 12px alpha ramp at the top and bottom edges, each scaled by min(12, distance scrolled from that edge), and no scrollbar.
5. **Card row.** Draw the surface (radius 6, colours §3.5) inside a 2px vertical inset; content at 10/8 padding: line 1 at y 8 (20 tall), line 2 at y 32, line 3 at y 54 (16 tall); 60px rows drop line 3 and put the trailing icons after the title. Use one text style per part (§3.4). Truncate with an ellipsis; branches with MiddleTruncate.
6. **Status slot.** A right-aligned slot, min 32 wide, 20 tall. At rest: status label or age. On hover (only with a hovering pointer) or when one of its controls has keyboard focus: hide the label (keep its width out of layout) and show the controls [Discard][Snooze][Settle] laid out from the right, Settle with −4 right margin; "Woke" stays and the controls sit to its left. The project name absorbs the width change. No animation.
7. **Recede and faded states** are colour and opacity only: receded rows use `sidebar-muted/75` text and a 400-weight `foreground/75` title; working/preparing receded rows draw the whole row at 0.7 opacity, animated to 1 over 150ms on hover.
8. **Live ticks.** Drive timers from one 1s ticker that runs only while a working/waiting/preparing row is on screen, and ages from one minute ticker aligned to wall-clock minutes. Invalidate only the affected text.
9. **Hover card.** Use tern's tooltip machinery with a list-wide provider: 150ms rest, instant re-open within 400ms, close on leave, placement right/start with 4px gap, flip within the window. Glass surface: fill `#ffffffd5` / `#111111d5` over a blurred backdrop. Gio has no backdrop blur; the card sits over the chat's flat background, so the fill alone comes within a few units of T3's pixels except where it covers message text. Shadow `0 20 25 −5` + `0 8 10 −6` at 25% black. T3 gives the card no max height, so a thread with dozens of linked PRs produces a card taller than the window (§3.15). Copy that only if pixel parity matters more than usability; otherwise cap the footer list and say so in tern's notes.
10. **Context menu.** Reproduce the browser's DOM menu (§3.16) in tern, not the dropdown-glass Menu primitive: no shadow, 28px rows, 16px icons drawn at **stroke 2**, 4px clamps to the window, submenus at row right + 4, keyboard rules of §5.13. On macOS consider a native NSMenu to match the desktop app; if so map details into "Label (detail)" and drop icons, as Electron does.
11. **Keyboard tree.** Implement §5.12 over the visible `[data-sidebar-row]` sequence (thread rows, draft rows, shelf headers, folder headers, "Show N more"), with parent ids (`shelf:<name>`, `folder:<key>`) for ArrowLeft/Right, and refocus by id after a row moves.
12. **Selection.** Ctrl/Cmd toggles, Shift ranges over `orderedKeys` from the anchor; navigation clears. Selected rows use `sidebar-row-selected` plus the light ring.
13. **Drag (later phase).** If implemented: 6px activation, vertical constraint, projected positions with 200ms ease transforms, labels and hints of §3.10, optimistic placement until the server echoes. Leaving the list horizontally becomes a context drag with the §3.10 ghost. Without it, offer the same moves through the menu (pin, settle, wake) and keep the list stable.
14. **Motion.** Implement the FLIP controller of §6 in ui: remember each item's y from the last frame by key; on a layout-key change animate moved items from their old offset (150ms, CSS `ease-out` = cubic-bezier(0, 0, 0.58, 1)), fade new items in from −min(h, 40), and keep a ghost of removed items fading down by min(h, 40). Skip it when shelves toggle, under reduced motion, on the first frame and when more than 40 items enter or leave.
15. **Menus that write** must check the environment's operate scope first and toast "Thread action unavailable" when missing; disable items as `buildThreadActionMenuItems` does.
16. **Footer.** Three 32px icon buttons with 4px gap and 600ms tooltips above them; "Back" on utility pages; the undo notice to their right with a 5s lifetime; the update control at the far right (desktop). The provider pill sits above the row.
17. **Persistence.** Store shelf expansion (three booleans, default false) and the project scope (list of group keys) per profile, as T3 does in localStorage and its UI store.
18. **PR badge popover.** The meta line's PR badge has its own interactive glass popover (§2.12, §3.15): 320 wide, scrollable to the available height, one link row per PR. It is separate from the hover card, whose PR footer is a plain list. Hovering the badge swaps the row's hover card for this popover inside the same tooltip group.
19. **Clocks.** Compute ages against the start of the current UTC minute and timers against the current whole second, both from wall-clock-aligned tickers (§5.5). Ages computed from `time.Now()` will flip one minute early compared with T3, and "now" will show for a shorter time.
20. **Rename.** Implement the input rules of §5.11: select all on focus, Enter commits, Escape cancels, blur commits, trimmed, empty is refused with a warning toast, unchanged sends nothing.
21. **Icon names.** Use the canonical lucide names of §3.20 when mapping to tern's icon set: `shield-question-mark` and `message-circle-question-mark`, not the old aliases.

## 9. Verification

A second pass on 2026-11-18 checked this document against the live UI and the pinned source. It used its own read-only browser session (same build, 1440×900 @2x, every sidebar setting at its default) and brought the tab to the front before each reading. I re-measured the values below in both themes, re-derived the rules after them from source, and changed the sections above wherever a value was wrong or missing. Live results that matched are now marked M where they used to be P or S. Shelves toggled for the measurements were collapsed again and the theme was left on light. No thread data was changed.

### 9.1 Re-measured live, light and dark

| # | Value | Light | Dark | Result |
|---|---|---|---|---|
| 1 | search field box, padding, radius, type | 8, 60, 151×32; 6 8; 6; 14/20 500 | = | match |
| 2 | search field text, icon | `#71717b`, `#85868e` (stroke 1.75) | `#a3a3a3`, `#8b8b8b` | match |
| 3 | search input text, placeholder | `#18181b`, `#71717b` (14/21) | `#f1f3f7`, `#a3a3a3` | match |
| 4 | search field hover | `#fcfcfc`, `#18181b`, icon unchanged | `#f1f3f714`, `#f1f3f7` | match; was S, now M |
| 5 | clear button box and hover | 131, 66, 20×20, radius 4, `x` 12 `#71717b` | hover bg `#191a1d`, icon `#f1f3f7` | **wrong**: dark hover icon was `#f7f9ff`; fixed in §3.1 |
| 6 | header icon buttons | 163 / 191 / 219, 28×28, radius 8, text `#71717bcc`, icon `#85868e`, width/height/padding 150ms | `#a3a3a3cc`, `#8b8b8b` | match |
| 7 | filter popup frame | 163, 94, 288×370, `#ffffffd5`, 1px `#18181b1a`, blur 12 saturate 1.14, `0 16px 40px -18px #0000008c`, z 130 | `#111111d5`, `#f5f5f51a`, blur 16 saturate 1.08, `0 18px 44px -18px #000000cc` | match |
| 8 | filter items and search | 278×32, padding 4 8, radius 4, highlight `#18181b17`; search 262×26, padding 0 9 0 20, focused border `#1b4ed8` | highlight `#f5f5f517`, text `#f5f5f5`, folder icon `#a1a1a1`, badge `#346bf1` ring `#111111`, focused border `#346bf1` | match |
| 9 | filter popup Escape | focus returns to the trigger | = | match with the source; the gap in §7 is closed |
| 10 | card title, default | 14/20 500 `#18181b` | `#f1f3f7` | match |
| 11 | receded row and title | `#71717bbf`; 400 `#18181bbf` | `#a3a3a3bf`; `#f1f3f7bf` | match |
| 12 | failed row title, seen | 500 `#18181bf2` | `#f1f3f7f2` | match; was P, now M |
| 13 | status Input | `#4f39f6`, 12/16 500, gap 4 | `#a3b3ff` | colour match, was P, now M; **icon name wrong**: renders `message-circle-question-mark` |
| 14 | status Failed | `#c10007`, `circle-alert` | `#ff6467` | match; was P, now M |
| 15 | status Woke | `#bb4d00`, underline on hover, tooltip "Dismiss Woke notification"; slot 110.63 on hover | `#ffb900` | match |
| 16 | status Working, Waiting | `#1447e6`, `#71717b`; working receded row opacity 0.7 with a 150ms transition | `#50a2ff`, `#a3a3a3` | match |
| 17 | project name, meta line | 12/16 400 `#71717b`; gap 6 | `#8f8f8f` | match |
| 18 | worktree icon | `folder-git-2` 14 `#71717b` | `#a1a1a1` | match |
| 19 | PR badge icon and number | open `#009966`, number `#71717b` | `#5ee9b5e6`, `#8f8f8f` | match |
| 20 | provider logo | 14×14, opacity 0.6, fill `#d97757` | same | match |
| 21 | pin icon | 12×12 at x 40, `#71717ba6` | `#a3a3a3a6` | match |
| 22 | Settle control | 62.45×20, padding 0 6, margin-right −4, gap 4, radius 6, `#71717b` | `#a3a3a3`, hover `#f1f3f7` | match |
| 23 | row hover | `#fcfcfc`, text `#18181b`; slot 32 → 58.45; name 159 → 132.55 | `#f1f3f714`, `#f1f3f7` | match |
| 24 | open row | `#ffffff`, `inset 0 0 0 1px #e4e4e7b3` | `#f1f3f71c`, no ring | match |
| 25 | selected row (Ctrl+click) | `#ffffff` + the same ring | `#f1f3f712`, no ring | match |
| 26 | row focus-visible | `inset 0 0 0 2px #1b4ed8` | `#346bf1` | match |
| 27 | shelf header | 235×32, 12/16 500 `#71717b`, rule `#e4e4e780` hidden, chevron −90° 150ms, focus ring `#1b4ed8b3` | `#a3a3a3`, rule `#ffffff0a` | match |
| 28 | settled slim row | 239×36, padding 0 10, gap 10; title `#71717b` at rest, `#18181b` on hover; favicon 0.4 grayscale → 1; Un-settle 26×24 | `#8f8f8f`, `#f1f3f7`; Un-settle `#a3a3a3` → `#f1f3f7` | match |
| 29 | "Show N more" | 239×36, 14/20 400 `#71717b`, hover `#fcfcfc` / `#18181b` | `#a3a3a3`, hover `#f1f3f714` / `#f1f3f7`; focus outline auto `#346bf180` | match; hover was S; focus style was missing |
| 30 | footer | 0, 856, 255×44, padding 4 8 8, gap 8; buttons 32×32 at 8 / 44 / 80, radius 8, `#71717bcc`, icon `#85868e`, hover `#fcfcfc` / `#18181b` | `#a3a3a3cc`, `#8b8b8b`, hover `#f1f3f714` / `#f1f3f7` | match; dark hover was S |
| 31 | footer tooltip | "Settings" 64.25×26.5 at 5, 829.5; 609ms delay | `#111111`, 1px `#ffffff14` | match |
| 32 | hover card frame | 4px right of the row, top-aligned; `#ffffffd5`, `#18181b1a`, blur 12 saturate 1.14; shadow `0 20px 25px -5px #00000040, 0 8px 10px -6px #00000040`; padding 4 8 / 8 4, gap 8; z 140; 163ms delay | `#111111d5`, `#f5f5f51a`, blur 16 saturate 1.08 | match |
| 33 | hover card lines | title 12/15 500 `#18181b`; grid 12/16 `#71717b`, gap 6; text `#18181bbf` | `#f5f5f5`, `#a1a1a1`, `#f5f5f5bf` | **wrong** for the branch line, which is muted (`#71717b` / `#a1a1a1`); fixed in §3.15 |
| 34 | hover card with many PRs | 320×1304 at 251, 5 in a 900px window | = | **missing**: no max height; added to §3.15 and §4 |
| 35 | PR badge popover | 320×890 glass, flipped to the right, link rows 24.5 tall | = | **missing**; added to §2.12 and §3.15 |
| 36 | project icons | monogram red-600 `#e7000b` on `#e7000b24`, yellow-600 `#d08700`; lucide override gray `#4a5565` | red-400 `#ff6467`, yellow-400 `#fdc700`; gray `#99a1af` | match; the tint is a CSS background, not a fill (wording fixed) |
| 37 | context menu | 207.02×401, `#ffffffd5`, `#18181b1a`, no shadow, separator `#e4e4e7`, disabled `#71717b` at 0.64, icons 16 stroke 2 | `#111111d5`, `#f5f5f51a`, text `#f5f5f5`, disabled `#a1a1a1` 0.64, separator `#ffffff14`, highlight `#ffffff0a` with a 150ms colour fade | match; dark highlight was S; **§6 said no motion**, fixed |
| 38 | Copy submenu | 128×122 at the row's right + 4, row top | = | match |
| 39 | search results | 239×36 (44 with an excerpt), padding 4 10, gap 10; highlight `#ffffff`; default `#71717bbf`; settled title `#71717bb3` 400; age `#71717b8c`; excerpt 193 wide `#71717bd9`; "You:" `#1447e6`, "Agent:" `#007a55`; mark 600 `#18181b` | `#f1f3f71c`, `#a3a3a3bf`, `#8f8f8fb3`, `#a3a3a38c`, `#a3a3a3d9`, `#50a2ff`, `#00d492`, `#f1f3f7`; "No threads found" 239×64 `#a3a3a3` | match; dark settled title and "You:" were P |
| 40 | list geometry | viewport 0, 100, 255×756, no scrollbar, 12px mask; `ul` gap 1, flex 1; card pitch 83 / 65; lines at +8, +32, +54; first shelf header `mt-auto` | = | match |

Behaviour checked live: ArrowDown, Home and End move focus between rows and clamp at both ends; ArrowRight expands a collapsed shelf and leaves focus on it; ArrowLeft goes from a child to its shelf header and then collapses it. Search arrows wrap (from the first of 50 results ArrowUp lands on the last), Escape empties the field and keeps focus, and the clear button refocuses the field. A right-click menu opens at the pointer, hovering Copy opens its submenu, Escape with focus on a root item closes both levels, and focus returns to the row. "Filter by <Project>" was disabled in the read-only session. Moving from one row to the next swapped the hover card within the first frame.

### 9.2 Re-derived from source

| # | Rule | Source | Result |
|---|---|---|---|
| 1 | age format and its clock | timestampFormat.ts:262–285; SidebarThreadRow.tsx:139–153; clock.ts:68–71 | format matches; **missing**: the clock is the floored UTC minute, so "now" lasts one to two minutes. Added to §5.5 |
| 2 | working timer | models.ts:401–407; clock.ts:62–65 | matches; added that hours never roll into days |
| 3 | working start | models.ts:384–399 | **imprecise**: a null `activityStartedAt` means no timer, and the fallback applies only to older servers. Fixed in §5.5 |
| 4 | status precedence and labels | Sidebar.logic.ts:962–992; SidebarThreadRow.tsx:745–805 | match |
| 5 | recede and unread rules | Sidebar.logic.ts:799–808, 928–950 | match |
| 6 | monogram and colour hash | projectIdentity.ts:13–35 | **imprecise**: the first glyph can be a digit and one-character names double. Fixed in §3.7. The file path in §1 was wrong (`src/`, not `components/`); fixed |
| 7 | section sorts and Working shelf grace | threadSort.ts:26–69, 268–336; threadInbox.ts:17–31, 57–96, 134–141 | match |
| 8 | snooze rules, presets, labels, wake label | threadSettled.ts:39–197, 251–360; settings.ts:129–135 | match; added the duration label rule ("In 90 minutes") |
| 9 | thread menu gating | threadActionMenu.logic.ts:100–112, 116–269; Sidebar.tsx:2593–2700 | **wrong**: "Filter by <Project>" needs write access; §5.13 now marks it † |
| 10 | multi-selection counts | Sidebar.tsx:2300–2385; Sidebar.logic.ts:634–657 | **wrong**: Unpin counts the pinned threads and Regenerate counts the regenerable ones, not the selection. Fixed in §5.13 |
| 11 | DOM menu keys, focus and colours | contextMenuFallback.ts:284–345, 379–604 | match; added click-to-open submenus and the thumb-button swallow |
| 12 | default shortcuts | keybindings.ts:112–133; AppSidebarLayout.tsx:226–292 | **missing** defaults for settle, pin, undo and new-without-project; added to §5.12 |
| 13 | hover card content | SidebarThreadRow.tsx:233–368 | **missing** line texts and conditions; added to §2.12 |
| 14 | PR badge, worktree and terminal controls | ThreadStatusIndicators.tsx:200–356, 358–448, 554–635 | **missing** labels, tooltips and the badge popover; added to §2.4, §2.12, §3.15 |
| 15 | rename input | SidebarThreadRow.tsx:601–640; Sidebar.tsx:1538–1571 | **missing** keys and toasts; added to §5.11 |
| 16 | DOM menu row size on narrow screens | contextMenuFallback.ts:432–447 | **wrong**: §3.16 said 32 tall and 16/24 text below 640px; the inline style keeps 28 and 14/20 at every width. Fixed |
| 17 | lucide names | shipped chunks; lucide-react 0.564 | **wrong**: `shield-question` and `message-circle-question` are `shield-question-mark` and `message-circle-question-mark`. Fixed in §3.6 and §3.20 |
| 18 | list motion and drag cancel paths | Sidebar.motion.ts:1–252; Sidebar.pointer.ts:1–145 | match |
| 19 | header row | SidebarChrome.tsx:43–194 | **missing** from this doc; summary added as §2.0 |
| 20 | update control and dialog | SidebarUpdatePill.tsx:415–590; UpdateEverythingDialog.tsx:100–250 | dialog **missing**; added to §3.14 and §5.18 |
| 21 | search selection | Sidebar.tsx:1497–1536; SidebarThreadRow.tsx:1633–1823 | **missing**: selecting a result clears the query first, and results carry the hover card. Added to §2.6 and §5.9 |

### 9.3 What else changed

- New: §2.0 (header row), the meta-line control labels and tooltips in §2.4, the shelf toggle's test id in §2.8, the PR badge popover in §2.12 and §3.15, the update dialog row in §3.14, the "Show N more" focus style in §3.19, the lucide loading stand-in in §3.7 and §3.20, four rows in §4, the undo notice layout in §5.18, and port notes 18–21.
- Motion: the DOM menu's item highlight fades over 150ms; a hover card hand-off has no animation at all (`data-instant` zeroes both transitions).
- Gaps: the filter focus, tooltip timing and dark-value gaps are closed; a background-tab note was added.
- Privacy: I searched the document for project, thread, branch, machine and account names seen in the live data and for paths and ids, and found none. The new examples ("web3 site", "alpha beta") are invented. Reference captures stay outside the repo.
- Still open: the write-gated UI, the approval, request and limited states, Done on a card, snoozed and draft rows, boring folders, Codex logos and the desktop-only parts are still P or S, because the read-only session and today's data could not show them.
