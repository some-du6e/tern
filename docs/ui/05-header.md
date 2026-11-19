# 05 · Chat header and thread details card

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
Foundations: [01-tokens.md](01-tokens.md), [02-primitives.md](02-primitives.md), [03-shell.md](03-shell.md). This document covers the title bar row of the chat column: the header, its breadcrumb, the thread actions menu behind the title, and where the panel toggles go in each layout. It also covers the thread details card, inline and as a popover, with every row it can show; the right panel's tab bar and its empty launcher; `DiffStatLabel`; `PullRequestContextDetails`; and the top fade under the header.

Conventions, as in 03-shell.md:

- Lengths are CSS px at the 16px root. One CSS px is one Gio dp. `rem` values scale with the interface font size.
- Colours are sRGB hex as Chrome paints them, `#rrggbbaa` when translucent. Two-theme cells read "light / dark". Translucent colours were resolved the way 01-tokens.md describes (exact RGB plus alpha), not read back from a premultiplied canvas.
- **M** is measured live. **P** is a class list evaluated live on a probe element. **S** is source only.
- Coordinates are viewport px at 1440×900 @2x in Linux Chrome, which renders Inter. Unless a row says otherwise the measured state is: sidebar expanded at 256, a server thread with messages, the details card inline at 1148, 64, right panel closed, panel animations off.
- Everything here sits outside `[data-app-sidebar]`, so it reads root tokens. Popups portal to `<body>` and read root tokens too.
- **Read-only client.** The measuring browser was paired with the `orchestration:read`, `terminal:read`, `terminal:operate` and `preview:operate` scopes only. Rename, every Git write, automation toggles and editor opens therefore rendered disabled or did not appear. Script controls stay enabled: the card never disables them (§3.7). That is a real state, the one a read-only client sees, and it is recorded as one. The enabled looks come from the same class lists (P) or from source (S).
- Text widths are Inter widths. SF Pro on macOS gives other widths and a slightly smaller cap height (§3.1).

## 1. Scope and sources

Paths are relative to the T3 repo.

| File | Lines | What it gives |
|---|---|---|
| `apps/web/src/components/chat/ChatHeader.tsx` | 46–69, 85–118, 120–158, 160–246, 247–398 | props, rename commit rule, rename state, parent-title collapse, rename / menu / context menu / key handlers, the breadcrumb tree |
| `apps/web/src/components/WorkspaceBreadcrumb.tsx` | 11–73 | `nav`/`ol`, crumb text, crumb item, separator |
| `apps/web/src/components/chat/PanelLayoutControls.tsx` | 38–138, 140–171, 173–187, 189–231 | the three toggles and their tooltips, maximize, the header-owned details toggle, the fixed group |
| `apps/web/src/components/ChatView.tsx` | 715, 817, 843–853, 936–942, 1055, 1089–1103, 1119–1129, 1201–1214, 1239–1243, 1275–1280, 1651–1669, 1719, 1732, 1945–1972, 2193–2230, 2707, 3251–3353, 3854–3923, 3954–3973 | everything the header and the card are fed, the three control placements, the card mount |
| `apps/web/src/components/chat/useRightPanelActions.ts` | 129–140, 212–243, 315–326 | opening Changes, the toggle handlers, the shortcut map |
| `apps/web/src/hooks/useThreadActionMenu.ts`, `apps/web/src/components/threadActionMenu.logic.ts` | 61–288; 1–269 | the title menu: items, order, what a read-only client may still use |
| `apps/web/src/contextMenuFallback.ts`, `apps/web/src/localApi.ts` | 169–211, 236–626; 42–57 | the in-page menu used in browsers (Electron shows a native menu) |
| `apps/web/src/components/Sidebar.logic.ts`, `packages/shared/src/textDirection.ts` | 821–823; 1–44 | trailing double-click test, title `dir` |
| `apps/web/src/components/ProjectFavicon.tsx`, `ProjectMonogram.tsx` | 35–202; 7–49 | the project icon in the crumb |
| `apps/web/src/components/chat/GuestShareControls.tsx` | 177–214 | guest share indicator at the header's end |
| `apps/web/src/components/chat/ThreadDetailsCard.tsx` | 22–199 | inline vs popover, density measurement, the inline `aside`, the popover popup |
| `apps/web/src/components/chat/threadDetailsCardLayout.ts` | 4–47 | 12px gap, 280px width, density rule, placement rule |
| `apps/web/src/components/chat/ThreadDetailsPanel.tsx` | 96–337 | sections and their show rules, the card's context menu, the empty fallback |
| `apps/web/src/components/chat/ThreadDetailsSection.tsx` | 1–49 | section frame, heading row, divider rule |
| `apps/web/src/components/chat/threadDetailsPanelStyles.ts` | 1–47 | every row class string |
| `apps/web/src/components/chat/ThreadDetailsControl.tsx` | 20–106 | row parts and tones |
| `apps/web/src/components/chat/threadDetailsItems.ts`, `ThreadDetailsItemsMenu.tsx` | 4–64; 16–42 | which items show, defaults, the "Show in thread details" menu |
| `apps/web/src/components/BranchToolbar.tsx` | 106–140, 165–432, 595–737 | "Copy full path", the run-context selector, the panel layout |
| `apps/web/src/components/BranchToolbar.logic.ts` | 16–64, 94–114, 121–233 | labels, environment indicator, branch trigger label rules |
| `apps/web/src/components/ComposerContextLabel.tsx`, `EnvironmentMachineIcon.tsx`, `PreviousWorktreeItemContent.tsx` | 6–36; 1–73; 5–19 | row label wrapper, machine icons, the previous-worktree item |
| `apps/web/src/components/BranchToolbarBranchSelector.tsx` | 97–792 | branch row, picker items, switch/create/checkout flows |
| `apps/web/src/components/BranchPicker.tsx` | 22–239 | the combobox popup and ref items |
| `apps/web/src/components/chat/OpenInPicker.tsx`, `OpenInPicker.logic.ts`, `OpenInPickerShortcut.ts` | 85–464; 5–20; 18–111 | editor list, split control, visibility, `editor.openFavorite` |
| `apps/web/src/editorPreferences.ts`, `editorLabels.ts`, `remoteOpen.ts` | 21, 47–56; 7–19; 31–60, 147–174 | preferred editor, labels, local vs remote opening |
| `apps/web/src/components/ProjectScriptsControl.tsx`, `apps/web/src/projectScripts.ts`, `packages/shared/src/projectScripts.ts`, `apps/web/src/components/projectScriptEditor.tsx` | 78–470; 70–73, 105–108; 11–21, 59–65; 57–79, 408, 568 | scripts split, primary script, menu labels, script icons, dialog titles |
| `apps/web/src/components/chat/ThreadDetailsPrRows.tsx`, `ThreadDetailsPrRow.tsx` | 39–216; 89–553 | pull request rows, row actions, rich tooltip, merge confirm |
| `apps/web/src/components/pullRequest/pullRequestIcons.tsx`, `pullRequestPresentation.tsx`, `pullRequestDetail.logic.ts`, `PullRequestChecksPopover.tsx` | 13–54; 118–124, 199–330, 497–515; 311–379; 130–217 | state icons and tones, checks presentation, row action choice, checks popover |
| `apps/web/src/components/ThreadStatusIndicators.tsx`, `apps/web/src/lib/openPullRequestLink.ts` | 62–163, 450–512; 43–48, 243–270 | which pull request a thread shows, opening it |
| `apps/web/src/components/GitActionsControl.tsx` | 130–189, 229–445, 951–1225, 1225–1503, 1504–1567, 1572–1781, 1783–2019 | Git actions row, menu, progress and success, Changes row, commit and default-branch dialogs |
| `apps/web/src/components/GitActionsControl.logic.ts`, `packages/client-runtime/src/state/gitActions.ts`, `packages/client-runtime/src/diffCount.ts` | 51–107, 109–341, 343–389, 443–481; 46–50, 316–327; 1–5 | quick action label rules, menu items, totals, default-branch prompt, count format |
| `packages/source-control-*/src/shared/definition.ts` | `changeRequest` blocks | "PR" / "pull request" (GitHub, Bitbucket, Azure DevOps, Forgejo, GitCafe, Cursor Origin), "MR" / "merge request" (GitLab) |
| `apps/web/src/components/chat/DiffStatLabel.tsx`, `apps/web/src/components/PullRequestContextDetails.tsx`, `contextChipParts.tsx` | 1–48; 12–35; 102–147 | compact diff stat, pull request chip tooltip |
| `apps/web/src/components/chat/ThreadAutomationsPanel.tsx`, `ThreadServersPanel.tsx`, `ThreadDetailsUsage.tsx`, `ThreadRelationshipsControl.tsx` | 34–254; 37–207; 43–260; 100–200, 560–600, 700–1033 | the card's lower sections (structure, headings, row variants; the lineage graph ordering is out of scope) |
| `apps/web/src/components/chat/canvasStore.ts`, `ChatCanvas.tsx`, `chatCanvasLayout.ts` | 82–157; 33–69; 22–29 | card placement inputs: find bar edge, lane padding probe, 32px clearance, preview overlap |
| `apps/web/src/components/ui/tooltip.tsx`, `apps/web/src/components/chat/useChatTerminalActions.ts` | 105–126; 201–212 | tooltip defaults (side top, align center, offset 4); the script run guard |
| `apps/web/src/components/RightPanelTabs.tsx` | 109–171, 176–210, 265–315, 317–570, 572–698, 701–826, 828–1030, 1032–1568 | tab bar, tabs, icons, titles, add menu, launcher, context menu, shortcuts |
| `apps/web/src/components/ui/panel-tab-close-button.tsx`, `RightPanelSheet.tsx`, `chat/ChatRightPanel.tsx` | 12–40; 5–31; 19–51 | close button, sheet wrapper, inline vs sheet |
| `apps/web/src/rightPanelStore.ts` | 196–213, 322–342, 972–991, 1023–1029, 1139–1160, 1180–1212 | card visibility state, closing surfaces, persistence |
| `apps/web/src/components/ui/scroll-area.tsx`, `ui/popover.tsx` | 22–77; 19–130 | scroll fade, popover `panel` variant |
| `apps/web/src/index.css`, `apps/web/src/components/chat/MessagesTimeline.tsx`, `ThreadFindBar.tsx` | 469–506; 159–162, 1188–1195, 1462–1465; 24 | top fade, its list spacer, find bar inset |
| `packages/shared/src/keybindings.ts`, `packages/contracts/src/git.ts`, `packages/contracts/src/rpc.ts` | 57–133; 83–93, 212–271; 427, 460–475, 551–575, 614–615 | default shortcuts, `VcsRef` and `VcsStatusResult`, RPC names |

Live page: same build, 1440×900 @2x, light and dark. I visited a draft view; server threads in Git repositories (clean, dirty on the default branch, a worktree on a feature branch with a merged pull request); a folder that is not a repository; and two No project threads, one with linked pull requests. Density was measured at window heights 500, 330, 250, 238 and 220, truncation at widths 520 and 420. Screenshots stayed in a private folder. Nothing was sent, renamed, committed or toggled. Every menu and popover I opened was closed with Escape. The Files and Diff surfaces I added to the right panel were closed again, and the panel's stored state ended as it began.

## 2. Structure

### 2.1 Title bar row placements

The chat header is the 52px row at the top of the chat column (03-shell.md §2.2, §3.5). What sits in it depends on the right panel (ChatView.tsx:1089–1102, 3266–3283, 3302, 3329–3336, 3965–3973):

| Placement | When | Where the controls go |
|---|---|---|
| no right panel | `rightPanelPresent` false | `WorkspaceTitlebarControls` is rendered inside `header[data-chat-header]` but is `position: fixed` at the window's top right: hidden maximize slot, details, terminal, right panel. ChatHeader reserves 96px (`pr-24`) |
| inline right panel | panel present, window > 980px or maximized | the fixed group renders at the ChatView root with maximize, terminal, right panel. The details toggle moves into the header as `ThreadPanelHeaderControl` (absolute, right 12px). ChatHeader reserves 40px (`pr-10`). Electron adds a 112px `no-drag` span over the fixed group |
| right panel sheet | panel open, window ≤ 980px, not maximized | no controls in the header. The sheet's tab bar ends with `mr-px flex items-center gap-1`: maximize, details, terminal, right panel |

### 2.2 ChatHeader

```
div  flex min-w-0 flex-1 items-center gap-2 sm:gap-3 pr-24 | pr-10                 onContextMenu → thread menu at the pointer
├─ nav[aria-label="Thread breadcrumb"]  min-w-0 flex-1 overflow-clip [overflow-clip-margin:2px]
│  └─ ol  m-0 flex min-w-0 list-none items-center gap-2 p-0 text-sm sm:gap-3
│     ├─ [has project] li  flex min-w-0 items-center font-medium shrink text-muted-foreground
│     │  └─ button[aria-label="New thread in <project>"]  tooltip trigger, side top
│     │        inline-flex min-w-0 max-w-full cursor-pointer items-center gap-1.5 rounded-sm text-muted-foreground transition-colors
│     │        hover:text-foreground focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring
│     │     ├─ ProjectFavicon  size-3.5   (monogram tile, favicon image, lucide icon or emoji; No project uses its own icon)
│     │     └─ span[data-slot=workspace-breadcrumb-text]  block min-w-0 truncate [text-box:trim-both_cap_alphabetic]
│     │                                                   supports-[text-box:trim-both_cap_alphabetic]:py-1.5 max-w-40
│     ├─ li[aria-hidden]  flex shrink-0 items-center text-icon-muted  > span[data-slot=workspace-breadcrumb-text] "/"
│     ├─ [parent link] li  flex min-w-0 items-center font-medium shrink-0 text-muted-foreground
│     │  └─ button[aria-label="Open parent thread: <title>"]  same classes as the project button, no gap
│     │     ├─ span[data-slot=workspace-breadcrumb-text][aria-hidden]  (w-0 while collapsed)
│     │     └─ [collapsed] span[data-parent-breadcrumb-ellipsis][aria-hidden] "..."
│     ├─ [parent link] li[aria-hidden] "/"
│     └─ li[aria-current=page]  flex min-w-0 items-center font-medium text-foreground min-w-10 flex-1
│        ├─ (renaming) input[aria-label="Thread title"][dir=auto][autofocus]
│        │      min-w-0 flex-1 rounded-sm bg-transparent text-sm font-medium text-foreground outline-none ring-1 ring-inset ring-ring/50 focus:ring-ring
│        ├─ (server thread) button[aria-haspopup=menu][aria-label="Thread actions for <title>"]  tooltip trigger, side top
│        │      group/thread-title inline-flex min-w-0 max-w-full cursor-pointer items-center gap-1 rounded-sm text-left focus-visible:…
│        │   ├─ h2.min-w-0[dir=ltr|rtl]  > span[data-slot=workspace-breadcrumb-text] <title>   (no dir when the title has no letters)
│        │   └─ svg.lucide-chevron-down[data-thread-title-chevron][aria-hidden]  size-3.5 shrink-0 text-muted-foreground opacity-0
│        │          transition-opacity group-hover/thread-title:opacity-100 group-focus-visible/thread-title:opacity-100
│        ├─ (draft) h2[aria-label=<title>].min-w-0.flex-1[dir]  tooltip trigger  > span[workspace-breadcrumb-text] "New thread"
│        └─ (linked origin, not renaming) Badge variant=info size=sm "From <origin>"  tooltip: "Started by an agent on <origin> through a linked environment."
└─ [server thread with a guest share] div  flex shrink-0 items-center gap-1 text-muted-foreground text-xs
       UserIcon size-3.5, "<name> is a guest" | "Guest link active" | "Guest link expired", Button xs ghost-muted "Stop sharing"
```

- The project crumb starts a new thread in that project. It shrinks (`shrink` replaces the item's `shrink-0` through tailwind-merge) and its text caps at 160px.
- The parent crumb only appears when the thread's lineage says it is a subagent of another thread (`lineage.relationshipToParent === "subagent"`); it links within the same environment (ChatView.tsx:843–853, 936–942). Forks and side chats get no parent crumb. Its text never shrinks. Instead, once the whole list is wider than the `ol`, the text collapses to width 0 and "..." shows (§5.1).
- The current crumb is the title. On a server thread it is a menu button with a chevron that only shows on hover or keyboard focus. On a draft it is a plain `h2`. While renaming it is an input.
- `WorkspaceBreadcrumbText` trims the line box to the cap height and adds 6px above and below, so every crumb is `12 + cap height` tall and the row centres on the letters, not on the line box.

### 2.3 Thread actions menu

Clicking the title, or right-clicking anywhere in ChatHeader, opens the thread actions menu (ChatHeader.tsx:199–236, useThreadActionMenu.ts). In Electron it is the native OS menu (`desktopBridge.showContextMenu`). In a browser it is the in-page menu from contextMenuFallback.ts, appended to `<body>`:

```
div[role=menu][data-context-menu=true][data-level=0][aria-label="Context menu"][tabindex=-1]     (a submenu: data-level=1…, aria-label = its parent item's text)
    dropdown-glass fixed z-[10000] min-w-32 max-w-sm overflow-hidden rounded-lg bg-clip-padding text-popover-foreground outline-none
└─ div  max-h-[calc(100vh-0.5rem)] min-w-0 max-w-sm overflow-y-auto overflow-x-hidden p-1
   ├─ div[role=separator][data-context-menu-separator]  mx-2 my-1 h-px bg-border        (before an item with separatorBefore)
   ├─ div  px-2 py-1.5 font-medium text-muted-foreground text-xs                         (header items; none in this menu)
   └─ button[role=menuitem|menuitemradio][tabindex=-1][aria-disabled][aria-haspopup=menu when it has children]
         flex w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-left outline-none transition-colors sm:min-h-7 sm:text-sm min-h-8 text-base
         + inline style min-height:1.75rem; font-size:.875rem; line-height:1.25rem; padding:.25rem .5rem; color:var(--contrast-foreground)
           (inline wins, so every row is 28 tall with 14px text at any width; contextMenuFallback.ts:437–453)
         ├─ svg  viewBox 24, stroke 2, size-4.5 sm:size-4 (18 below 640, 16 from 640), text-muted-foreground (destructive: current colour)
         │       | radio: check icon (same sizes) or a spacer pinned to 16×16 by inline style
         ├─ span.min-w-0.flex-1.truncate  label
         ├─ [detail] span  ms-auto shrink-0 ps-4 text-muted-foreground text-xs
         └─ [children] svg chevron-right  -me-0.5 ms-auto size-4.5 shrink-0 text-muted-foreground opacity-80 sm:size-4
```

Items in order (threadActionMenu.logic.ts:117–269). Brackets give the condition:

1. Find in thread, `text-search` [thread find enabled]
2. New thread on `<branch>`, `message-square-plus` [the thread has a branch]
3. Pin thread or Unpin thread, `pin` / `pin-off` [server supports pinning]
4. Settle thread or Un-settle thread, `circle-check` [settlement]
5. Snooze ▸ (presets with their time as detail, separator, "Custom…") or Wake thread, `clock` [snooze; disabled when it cannot snooze now]
6. separator, Rename thread, `pencil`
7. Regenerate title or "Regenerating…", `refresh-cw` [title regeneration]
8. Mark unread, `mail-open`
9. Share with guest, `user-plus` [guest grant]
10. Filter by `<project>` or Show all projects, `folder-tree` [project filter]
11. Turn into project, `folder-plus` [No project thread with its own folder; disabled while running]
12. Auto-settle behavior ▸ Follow settings / Never (radio checks), `timer` [auto-settle opt-out]
13. separator, Copy ▸ Path (`folder`), Branch (`git-branch`, with a branch), Thread (`message-square`), Thread ID (`hash`)
14. Project settings, `settings`
15. separator, Archive thread, `archive` [disabled while running]
16. Delete, `trash`, destructive

A client without the operate scope keeps Find, New thread on branch, Project settings, Mark unread, Share with guest and the Copy items; every other item renders disabled, children included (threadActionMenu.logic.ts:101–115, 254–268). That is what was measured.

The sidebar's row menu is the same component with the same items. [04-sidebar.md](04-sidebar.md) §3.16 and §5.13 also describe how Electron's native menu renders them (details in parentheses, radio items as checkboxes, icons dropped).

### 2.4 Panel toggles

03-shell.md §3.6 has their geometry and colours. The tree (PanelLayoutControls.tsx:57–137):

```
div[data-panel-layout-controls]  flex h-full shrink-0 items-center gap-1 [-webkit-app-region:no-drag]
├─ span.flex.shrink-0  tooltip trigger "Toggle thread details (<key>)", side bottom
│  └─ Toggle[aria-label="Toggle thread details panel"] variant ghost size sm, pressed = card open      > SquareMenu size-4
│        (popover mode: rendered by PopoverTrigger with the shared handle, so it is data-slot=popover-trigger, aria-haspopup=dialog)
├─ span  tooltip "Toggle terminal drawer (<key>)" or "Terminal drawer is unavailable"
│  └─ Toggle[aria-label="Toggle terminal drawer"] disabled without a project                         > PanelBottom size-4
└─ span  tooltip "Toggle right panel (<key>)" or "Right panel is unavailable"
   └─ Toggle[aria-label="Toggle right panel"][data-right-panel-toggle] disabled without a project    > PanelRight size-4
```

### 2.5 Details card frame

Inline (ThreadDetailsCard.tsx:115–169):

```
aside[data-workspace-area=details][data-workspace-order=3][role=region][aria-label="Thread details"][tabindex=-1]
     [data-workspace-focus-key="details:<env>:<thread>"][data-thread-details-panel=inline][data-density=full|compact|essential]
     absolute z-20   (+ transition-[top] duration-150 ease-out motion-reduce:transition-none while the find bar is open)
     style: right 12px; top 12 + find bar bottom edge (12, or 60 with the find bar); width 280px; max-height <slot height>
└─ div[data-thread-details-card]  dropdown-glass isolate contain-paint grid max-h-full grid-rows-[minmax(0,1fr)] overflow-hidden rounded-2xl
   └─ ScrollArea scrollFade: div[role=presentation] relative size-full min-h-0 overflow-hidden rounded-[inherit]
      ├─ div[data-slot=scroll-area-viewport]  h-full max-h-[inherit] overflow-auto overscroll-contain rounded-[inherit] outline-none transition-shadow
      │     focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background
      │     + edge masks, focus-visible:mask-none, [--fade-size:1.5rem], scroll-p-[var(--fade-size)]
      │  └─ div (measured for density) > div.group/details-items > sections (§2.7)
      ├─ Scrollbar vertical  flex w-1.5 my-1 mr-px opacity-0 transition-opacity delay-300; data-hovering / data-scrolling: opacity-100, delay-0, duration-100
      │  └─ Thumb  relative flex-1 rounded-full bg-(--app-scrollbar-thumb) hover:bg-(--app-scrollbar-thumb-hover)
      └─ Scrollbar horizontal (h-1.5 mx-1 mb-px) and Corner, unused here   (scroll-area.tsx:22–100)
```

Popover (ThreadDetailsCard.tsx:132–197, popover.tsx:79–125):

```
Popover (handle shared with the header toggle; open = mode popover and popoverOpen)
└─ div[data-slot=popover-positioner]  z-(--z-sheet) (46)
      w-[min(calc(var(--thread-details-panel-width)+--spacing(4)),calc(var(--anchor-width)---spacing(2)))] transition-[top] duration-150 ease-out
      anchor header[data-chat-header], side bottom, align end, alignOffset 4, sideOffset = find bar inset, collisionPadding 0,
      collisionAvoidance { side: shift, align: shift, fallbackAxisSide: none }
   └─ div[data-slot=popover-popup][role=dialog][aria-label="Thread details"][data-workspace-area=details]
         variant panel: w-full overflow-visible rounded-none border-0 bg-transparent shadow-none before:hidden, no backdrop filter
      └─ div[data-slot=popover-viewport]  overflow-visible py-2 [--viewport-inline-padding:--spacing(2)]   (8px all round)
         └─ div[data-density][data-thread-details-panel=popover]
            └─ the same card div + max-h-[min(calc(100dvh-6.5rem),calc(var(--available-height,100dvh)-1rem))]
```

The popup element exists in both modes. Base UI keeps a popup mounted if it unmounts mid-open, so ThreadDetailsCard renders it always and only fills it in popover mode (ThreadDetailsCard.tsx:170–196).

### 2.6 Sections and row parts

`ThreadDetailsSection` (ThreadDetailsSection.tsx:19–48):

```
section[aria-labelledby=<heading id>] (or aria-label=<title> when the heading is hidden)
        px-2 pt-2 pb-2.5 has-[>div:last-child:empty]:hidden
        + separated: border-border/65 [section:not(:has(>div:last-child:empty))~&]:border-t
├─ div  mb-1 flex min-h-8 min-w-0 items-center justify-between gap-2 px-2.5      (class "hidden" when showHeading is false)
│  ├─ h3[id]  min-w-0 truncate text-2xs font-medium text-muted-foreground select-none
│  └─ [actions] div  flex shrink-0 items-center gap-1
└─ content, the last child: an empty div hides the whole section
```

Every control in the card is a `ThreadDetailsControl` (ThreadDetailsControl.tsx:46–91). Base classes (ThreadDetailsControl.tsx:53): `relative inline-flex shrink-0 cursor-pointer items-center whitespace-nowrap border outline-none transition-[box-shadow,scale] [&:not([aria-haspopup]):not(:disabled):not([aria-disabled=true])]:will-change-[scale] [&:active:not([aria-haspopup])]:scale-[0.97] focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 [&_svg]:shrink-0 disabled:pointer-events-none disabled:opacity-64 aria-disabled:cursor-not-allowed aria-disabled:opacity-64`. On a coarse pointer every control gets a transparent `::after` hit area of at least 44×44 (`min-h-11 min-w-11`), so the 24px icon controls stay tappable. Plus one part:

| Part | Added classes (threadDetailsPanelStyles.ts) | Box |
|---|---|---|
| `row` | `h-8 min-w-0 rounded-lg border-transparent gap-2.5 px-2.5 text-left text-[13px] font-medium text-foreground/80 w-full justify-start` + row surface | 262×32 |
| `select` | `row` + `gap-0 px-0`. Children are wrapped in `span[data-slot=select-label]` (`flex h-full min-w-0 flex-1 items-center rounded-s-lg gap-2.5 px-2.5 text-left` + half hover) and `span[data-slot=select-icon]` (`rounded-e-lg` + half hover, 32 wide, chevron 16, `::before` hairline) | 262×32 = 229×30 + 32×30 |
| `primary` | control classes + `flex-1 justify-start rounded-e-none` + split button surface | 229×32 |
| `secondary` | control classes + `w-8 justify-center rounded-s-none px-0` + split button surface | 32×32 |
| `icon` | `size-6 justify-center rounded-md border-transparent bg-transparent p-0` + row surface | 24×24 |
| `meta` | `h-6 gap-1 rounded-md border-transparent px-1.5 text-xs font-medium text-muted-foreground` + row surface | 24 tall |

Modifiers: `multiline` adds `h-auto min-h-8 py-1.5 disabled:opacity-100`, so a disabled multiline control (the Git progress and success rows) is not faded. Tone `muted` adds `text-muted-foreground hover:text-foreground/80`. Tone `primary` adds `bg-primary text-primary-foreground hover:text-primary-foreground hover:!bg-primary/85 data-pressed:!bg-primary/85` (and the same under `dark:`). Tone `destructive` adds `text-destructive hover:text-destructive data-pressed:text-destructive`; on a `meta` part also `bg-destructive/10 hover:!bg-destructive/16 data-pressed:!bg-destructive/16`. A `meta` part with a primary or destructive tone is `h-5` with a 1px ring and no offset; the primary one draws the ring in `--background`.

Surfaces: row surface = `bg-transparent shadow-none hover:!bg-black/[0.035] data-pressed:!bg-black/[0.035] dark:hover:!bg-white/[0.08] dark:data-pressed:!bg-white/[0.08]`. Split buttons add `data-popup-open:` with the same fill. A split group is `group/details-split flex w-full items-center rounded-lg` with the same hover fill plus `has-[[data-popup-open]]:` fill. Its separator is `span[aria-hidden] h-4 w-px shrink-0 bg-transparent group-hover/details-split:bg-border/65 group-has-[[data-popup-open]]/details-split:bg-border/65`.

When `panel` is false the same component renders a plain `Button` (used by the composer toolbar, not by this card).

### 2.7 Card content, in order

```
div.group/details-items
├─ section[aria-label=Workspace]  separated=false, heading hidden
│  └─ div.flex.flex-col
│     ├─ div[data-thread-panel-run-context].flex.w-full.flex-col          environment row (BranchToolbar panelSection=workspace)
│     │  └─ Menu trigger ThreadDetailsControl part=select  |  locked span
│     ├─ div[role=group][aria-label="Open in editor"]  split: primary "Open in <Editor>" | hairline | secondary "Choose editor"
│     └─ scripts: div[role=group][aria-label="Project scripts"] split [script] | chevron
│                 or div[role=group][aria-label="Project actions"] split [wrench "Actions"] | chevron
│                 or row [+ "Add project script"]
├─ section[aria-label="Version Control"]  heading hidden, separated in full density only
│  └─ div.flex.flex-col
│     ├─ [Git repo] div[data-thread-panel-run-context] > div.flex.min-w-0.flex-col.items-stretch.w-full
│     │     ├─ span.flex.min-w-0 > ComboboxTrigger part=select (branch)
│     │     └─ ThreadDetailsThreadPrRows (the branch's pull request and the thread's linked ones)
│     ├─ [not a Git repo or No project] ThreadDetailsThreadPrRows (linked pull requests only)
│     └─ GitActionsControl: row "Initialize Git"  |  div[role=group][aria-label="Git actions"] split;  then button "Changes"
├─ section "Servers"          ThreadServersPanel          (full density, server thread, item on)
├─ section "Automations"      ThreadAutomationsPanel      (full density, server thread, item on)
├─ section "Lineage" (or "Lineage · N running"), "Side chats" (or "Side chats · N running")
│                             ThreadRelationshipsPanel (full density, server thread, either item on; §3.10)
├─ section "Usage limits"     ThreadDetailsUsage          (item on, off by default)
├─ section "Context window"   ThreadDetailsContext        (item on and a context window known, off by default)
└─ div.p-2  group-has-[>section>:last-child:not(:empty)]/details-items:hidden
   └─ row [sliders-horizontal "Choose what to show"]     shown only when every section above is empty
+ Menu at the pointer: "Show in thread details" (§3.11)
```

### 2.8 Right panel tab bar

The panel host and the sheet are in 03-shell.md §3.7 and §3.8. The bar (RightPanelTabs.tsx:1349–1545):

```
div[data-right-panel-tabbar]  flex h-[var(--workspace-topbar-height)] min-h-[…] shrink-0 items-center gap-1 pl-2
      pr-28 (inline, no layout controls) | pr-3 (sheet)   [Electron inline: drag-region wco:pr-(--workspace-native-controls-inset)]
      [inline and maximized: the collapsed-sidebar left inset]
├─ ScrollArea[data-right-panel-tab-list]  radius none, hideScrollbars, scrollFade, min-w-0 flex-1
│  └─ div  flex h-full w-max min-w-full items-center gap-1
│     ├─ RightPanelTab × n   (dnd-kit sortable)
│     │  div[data-active-tab=true|false]  cursor-pointer group/tab flex h-6 max-w-36 shrink-0 items-center gap-0.5 rounded-md pr-2 pl-1.5 text-xs
│     │       active: bg-accent text-foreground   inactive: text-muted-foreground hover:bg-accent/60 hover:text-foreground   dragging: relative z-10
│     │  ├─ button[aria-label="Close <title>"]  group/close relative flex size-4 shrink-0 items-center justify-center rounded-sm hover:bg-muted
│     │  │  ├─ span.relative.flex.size-3  group-hover/tab:hidden group-focus-visible/close:hidden  > surface icon 12 [+ pending dot]
│     │  │  └─ X  hidden size-3 group-hover/tab:block group-focus-visible/close:block
│     │  ├─ [audible browser tab] button size-4 rounded-sm hover:bg-muted  > Volume2 or VolumeOff size-3
│     │  └─ button.cursor-pointer.flex.min-w-0.items-center (tooltip trigger) > span.truncate <title>    | device rename input
│     └─ [panel open] Menu trigger Button[aria-label="Add panel surface"] size icon-xs variant ghost-muted  > Plus size-3.5
├─ [tabs overflow] div[role=group][aria-label="Scroll panel tabs"]  flex shrink-0 items-center gap-0.5 [-webkit-app-region:no-drag]
│     Button icon-xs ghost "Scroll tabs left" (ChevronLeft), "Scroll tabs right" (ChevronRight), each disabled at its end
├─ [sheet] layout controls
├─ [Electron, no layout controls] span  hidden w-24 shrink-0 wco:block
└─ [Electron inline] span  pointer-events-none fixed top right h-[topbar] w-28 [-webkit-app-region:no-drag]
div[data-right-panel-surface-content]  flex min-h-0 flex-1 flex-col  > the active surface, or the launcher (§3.16)
```

### 2.9 DiffStatLabel and PullRequestContextDetails

Neither renders in the card. They belong to this topic because the card's numbers and pull request rows share their rules.

- `DiffStatLabel` (DiffStatLabel.tsx:22–48) is the `+N -N` pair in the changed-files tree and the diff panel: `span[role=group][aria-label="N additions, N deletions"]`, layout `aligned` = `inline-grid grid-cols-[4ch_4ch] gap-2 text-right tabular-nums align-middle`, layout `inline` = `inline-flex items-center gap-1 tabular-nums align-middle`. Two `span[aria-hidden].font-mono`, `text-diff-addition-foreground` and `text-diff-deletion-foreground`, with `formatCompactDiffCount` (§5.12).
- `PullRequestContextDetails` (PullRequestContextDetails.tsx:12–35) is the tooltip body of a pull request context chip (composer, messages): `div.max-w-80.space-y-1.overflow-hidden.py-0.5.text-left`, then a 12px weight 500 line (state icon 14 in the state tone, "Pull request #N" in `--foreground`, the state label in the tone), the title (`wrap-break-word`, foreground), and a 10px `--secondary-label` line `<head> → <base>` in `code` with `arrow-right` 12. Chips without an in-app preview show it in a plain tooltip, side top (contextChipParts.tsx:139–146).

### 2.10 Top fade

The timeline scroller under the header carries `topbar-scroll-fade` (index.css:469–506) unless a banner sits at the top of the timeline (`topFadeEnabled = !hasTimelineTopBanner`, ChatView.tsx:1719, 3495). The mask is fixed to the scroller, not to the scroll position: content alpha ramps from 0 to 1 over the top 25px (24px fade height + 1), except the 6px scrollbar lane. Values and the stepped ramp are in 03-shell.md §3.5. With the fade on, the list starts with a 24px spacer (`h-[var(--workspace-titlebar-scroll-fade-height)]`) so the first row is not born faded; with it off the spacer is 12px (16px from 640 up) (MessagesTimeline.tsx:159–162, 1188–1195). A thread with a parent link gets neither spacer: its list opens with the parent divider row (`chat-content-lane pt-1 sm:pt-2`, MessagesTimeline.tsx:1195–1198). The header itself is opaque `--background` with no bottom border, so the fade is the only separation between header and timeline.

## 3. Measurements

### 3.1 Header row and breadcrumb

Geometry, light and dark identical.

| Item | Value | M/S |
|---|---|---|
| header | 256, 0, 1184×52, padding 0 20, `--background`, no bottom border (03-shell.md §3.5) | M |
| ChatHeader root | 276, 14.91, 1144×22.19. `gap: 12px` (8 below 640). `padding-right: 96px` (`pr-24`), 40px (`pr-10`) while the inline right panel owns the title bar | M |
| `nav` | 276..1324, 1048 wide. `overflow: clip`, `overflow-clip-margin: 2px` | M |
| `ol` | `column-gap: 12px` (8 below 640). Font 14px / 20px | M |
| project crumb `li` | content width (40.22 for a four-letter name), `flex-shrink: 1`, `min-width: 0` | M |
| project button | `gap: 6px`, radius 4 | M |
| project icon | 14×14 at y 19 | M |
| project text | x 296 (icon 14 + gap 6). `max-width: 160px`, one line, ellipsis. Padding 6 0. 14px / 20px, weight 500 | M |
| separator `li` | "/" at 14px / 20px, weight 400, 5.05 wide in Inter. `flex-shrink: 0` | M |
| current `li` | `flex: 1 1 0%`, `min-width: 40px` | M |
| title button | `gap: 4px`, radius 4. Width = text width + 4 + 14 | M |
| title text | 14px / 20px, weight 500, one line, ellipsis. `dir` from `textDirection(title)` | M |
| title chevron | 14×14, 4 after the text, y 19. Opacity 0 at rest | M |
| draft title `h2` | `flex: 1`, `aria-label` = title, no chevron, not focusable | M |
| crumb box height | 22.19 = 6 + cap height + 6, centred in the 52px row (top 14.91) | M |
| rename input | 20 tall (line height, no padding), `flex: 1`, radius 4, transparent | P |
| below 640px | header padding 0 12 0 56, root gap 8, `ol` gap 8 (measured at 520 and 420 wide) | M |

Cap height. `text-box: trim-both cap alphabetic` sets the crumb box to cap height plus the 6px paddings. Inter's cap height is 0.727em, 10.19px at 14px, so the box is 22.19. SF Pro's is about 0.705em, which gives about 21.9 on a Mac. In Gio, centre the cap band, not the line: the baseline sits at `26 + capHeight / 2` (31.09 with Inter).

Truncation, measured at 420px wide: the title `li` took the 194.73 left over and its text truncated at 176.73. The project crumb kept its 40.22. The title item grows from 0 and stops shrinking at 40, so the project crumb starts shrinking only once the title is down to 40.

Colours.

| Element | Token | Light | Dark | M/S |
|---|---|---|---|---|
| header background | `--background` | `#fcfcfc` | `#000000` | M |
| project crumb text and icon slot | `--muted-foreground` | `#71717b` | `#a1a1a1` | M |
| project crumb, hover | `--foreground` | `#18181b` | `#f5f5f5` | M |
| separator | `--icon-muted` | `#71717b` | `#a1a1a1` | M |
| title | `--foreground` | `#18181b` | `#f5f5f5` | M |
| title chevron | `--muted-foreground` | `#71717b` | `#a1a1a1` | M |
| focus ring, crumbs and title | `--ring`, `box-shadow: inset 0 0 0 2px` over the 22.19px box, radius 4, no offset | `#1b4ed8` | `#346bf1` | M |
| rename input ring, at rest | `--ring` 50%, `inset 0 0 0 1px` | `#1b4ed880` | `#346bf180` | P |
| rename input ring, focused | `--ring`, `inset 0 0 0 1px` | `#1b4ed8` | `#346bf1` | P |
| rename input text | `--foreground`, 14px / 20px, 500 | `#18181b` | `#f5f5f5` | P |
| "From <origin>" | Badge `info` `sm` (02-primitives.md §3.4): 16 tall, 10px text | bg `#2680ff14`, text `#1447e6` | bg `#2c7cff29`, text `#50a2ff` | S |
| guest share line | `--muted-foreground`, 12px / 16px | `#71717b` | `#a1a1a1` | S |
| monogram project icon | the project colour, tile `currentColor` at 14%, radius 25%, mono 8.25px bold | e.g. amber-600 `#e17100` | e.g. amber-400 | M |
| No project icon | lucide `message-square-dashed`, `text-gray-600 dark:text-gray-400` | `#4a5565` | `#99a1af` | M / P |

### 3.2 Header tooltips

Tooltip styling is 02-primitives.md §3.7 (`--popover` `#ffffff` / `#111111`, 1px `--border`, radius 6, 12px / 16.5px, padding 4 8, 600ms delay). The header has no room above it, so the tooltips declared `side="top"` all flip below. In the desktop app the top collision padding is 5 + title bar height, which forces the same flip.

| Trigger | Text | Declared | Measured (light) | M/S |
|---|---|---|---|---|
| project crumb | "New thread in <project>" | top | below, centred on the crumb: 240, 41, 146.7×26.5 for an eight-letter name. Top 41 = crumb bottom (37.1) + 4. A box read before the open transition ends is at scale 0.98 (top 40.92, 25.97 tall) | M |
| title (server thread) | the full title | top | below, centred: 362, 41, 206.34×26.5 for a 221.7px title; 399.5, 41, 233.45×26.5 for a 272.4px title button | M |
| draft title | the title | top | below | S |
| parent crumb | the parent title | top | below | S |
| "From <origin>" badge | "Started by an agent on <origin> through a linked environment." | top | below | S |
| details toggle | "Toggle thread details (<key>)" | bottom | 1216.5, 44, 218.73×26.5: right edge held 5px from the window | M |
| terminal toggle | "Toggle terminal drawer (<key>)", or "Terminal drawer is unavailable" | bottom | 1217, 44 | M |
| right panel toggle | "Toggle right panel (<key>)", or "Right panel is unavailable" | bottom | 1210.5, 44 | M |
| maximize | "Maximize panel" / "Restore panel size" | bottom | | S |

Keyboard focus also opens these tooltips (Base UI). A focused title therefore shows its tooltip under the menu button.

### 3.3 Thread actions menu (browser fallback)

| Property | Light | Dark | M/S |
|---|---|---|---|
| position from the title | left = title button left (345.27), top = button bottom + 4 (41.09) | = | M |
| position from a right-click | the pointer's x, y (right-click at 900, 26 put the menu at 900, 26) | = | M |
| clamping | one frame after opening, moved to stay 4px inside the window | = | S |
| size | 207.02×401 for 13 items and 3 separators; width follows the content, min 128, max 384 (335.94 wide when "New thread on <branch>" carried a long branch name) | = | M |
| surface | `dropdown-glass` `#ffffffd5`, `blur(12px) saturate(1.14)` | `#111111d5`, `blur(16px) saturate(1.08)` | M |
| border | 1px `--contrast-foreground` 10%: `#18181b1a` | `#f5f5f51a` | M |
| radius, clip | 8 (`--radius-lg`), `background-clip: padding-box` | = | M |
| shadow | none (Base UI menus have one; this menu does not) | none | M |
| inner padding | 4 | = | M |
| item | min height 28, padding 4 8, gap 8, radius 4 (`--radius-sm`), 14px / 20px sans, 400, `cursor: default`, at every width: the inline style beats the `min-h-8 text-base` classes, so rows stay 28 tall and 14px below 640 (measured at 520 wide) | = | M |
| item text | `--contrast-foreground` `#18181b` | `#f5f5f5` | M |
| item icon | 16×16 from 640 up, 18×18 below (`size-4.5 sm:size-4`, no inline size; measured at 520), `--muted-foreground` `#71717b`. Stroke width **2**: these SVGs have no `lucide` class, so the 1.75 rule (01-tokens.md §3.9) does not reach them | `#a1a1a1` | M |
| highlighted (hover or focus) | bg `--accent` `#f4f4f5`, text `--contrast-accent-foreground` `#18181b` | `#ffffff0a`, `#f5f5f5` | M |
| destructive (Delete) | text and icon `--destructive-foreground` `#c10007`; highlighted bg `--destructive` 10% `#fb2c361a` | `#ff6467`, `#fb414a1a` | S |
| disabled | text `--contrast-muted-foreground` `#71717b`, opacity .64, `pointer-events: none` | `#a1a1a1` | M |
| separator | 1px `--contrast-border` `#e4e4e7`, margin 4 8 (measured 181 wide) | `#ffffff14` | M |
| submenu chevron | `chevron-right` 16 (18 below 640), muted, opacity .8, margin-right −2 | = | M |
| item detail | 12px muted, padding-left 16, pushed right | = | S |
| submenu placement | left = parent item right + 4, top = parent item top. If that overflows the window: parent item left − width − 4. Measured: Copy ▸ (item right 711.13, top 316.09) opened its submenu at 715.13, 316.09, 128×122 (four items, min width 128) | = | M |
| z-index | 10000 | = | M |

Measured order on a pinned server thread on `main`, read-only client: Find in thread · New thread on main · Unpin thread (off) · Settle thread (off) · Snooze ▸ (off) · separator · Rename thread (off) · Regenerate title (off) · Mark unread · Auto-settle behavior ▸ (off) · separator · Copy ▸ · Project settings · separator · Archive thread (off) · Delete (off). Separators measured at y 190, 311 and 376.

### 3.4 Panel toggles, additions to 03-shell.md §3.6

| Item | Value | M/S |
|---|---|---|
| shortcuts in the tooltips | defaults `threadPanel.toggle` mod+Shift+B, `terminal.toggle` mod+J, `rightPanel.toggle` mod+Alt+B. Labels "Ctrl+Shift+B" off macOS, "⇧⌘B" on macOS (03-shell.md §5.7). The measured server had user overrides, so its labels differed | S |
| details toggle pressed | inline mode: the card is visible. Popover mode: the popover is open. Pressed paints `--accent` `#f4f4f5` / `#ffffff0a` (measured while the popover was open) | M |
| popover-mode trigger | the Toggle is rendered by `PopoverTrigger`: `data-slot=popover-trigger`, `aria-haspopup=dialog`, `aria-pressed=false` while closed; the Toggle's own `onPressedChange` is unset | M |
| header-owned details toggle | `ThreadPanelHeaderControl`: absolute, top 0, right 12 inside the header, 28×52 box; 860, 0 with a 644px chat column | M |
| maximize slot while the panel is closed | `position: absolute; right: 100%; margin-right: 4px; opacity: 0; inert`, box 1303, 12 | M |
| disabled | terminal and right panel toggles without a project; muted text, `pointer-events: none` (03-shell.md §3.6) | S / P |

### 3.5 Details card frame

| Item | Value | M/S |
|---|---|---|
| inline slot | x = canvas width − 292 (right 12), y = 12 + the find bar's bottom edge in canvas coordinates while the bar is open. The bar sits at top 12 and is 36 tall, so y = 60 (measured, 112 in the viewport). `THREAD_FIND_BAR_RESERVED_HEIGHT` (36 + 12) only switches that measurement on (ChatCanvas.tsx:37–51, canvasStore.ts:98). Width 280 | M |
| inline max height | canvas height − y − 12: 824 at 900 tall, 776 with the find bar | M |
| inline rule | inline only if x − 32 − lane padding ≥ 640 (canvas ≥ 1012 wide) and the max height ≥ 160 (canvas ≥ 184 tall, 232 with the find bar). Otherwise popover mode. Measured edges with the 256 sidebar: window 1268 wide inline, 1267 popover; window 236 tall inline (max height 160, essential density), 235 popover | M |
| preview mini player | when the mini player's frame overlaps the card's column (frame.x + frame.width > x − 12 and frame.x < x + 292), the max height becomes min(canvas height − y − 12, frame.y − y − 12); under 160 the card goes to popover mode (threadDetailsCardLayout.ts:35–39) | S |
| inline height | content height up to the max, then the content scrolls | M |
| card | radius 16 (`rounded-2xl`), 1px border, no shadow, `isolation: isolate`, `contain: paint`, one grid row `minmax(0, 1fr)` | M |
| scroll area | 278 wide inside the border. Edge fades from ScrollArea `scrollFade`: each overflowing edge fades over min(24px, overflow on that edge). When the viewport itself has keyboard focus (Chrome puts a scroller in the tab order only while it has no focusable child, so this is rare in the card) the masks drop and it draws `inset 0 0 0 1px --background, inset 0 0 0 3px --ring`. Overlay scrollbar: 6px wide, 4px from top and bottom, 1px from the right, rounded thumb in `--app-scrollbar-thumb`; opacity 0, shown over 100ms while hovered or scrolling, hidden again after a 300ms delay | M (box) / S |
| popover positioner | width min(280 + 16, header width − 8) = 296. z 46. Measured 600, 52, 296×437 with a 644px chat column | M |
| popover placement | right edge = header right − 4 (`alignOffset` 12 − 8); top = header bottom + find bar inset | M |
| popover padding | 8 all round, so the card lands 12 in from the header's right edge and 8 below it (measured 608, 60) | M |
| popover card max height | min(window height − 104, available height − 16): 796 at 900 tall | M |
| popover density height | canvas height − 52 | S |

| Element | Token | Light | Dark | M/S |
|---|---|---|---|---|
| card fill | `dropdown-glass`: `--popover` at 83.6% | `#ffffffd5` | `#111111d5` | M |
| backdrop | `--glass-blur`, `--glass-saturation` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` | M |
| border | `--contrast-foreground` 10% | `#18181b1a` | `#f5f5f51a` | M |
| fill as painted over the chat canvas | | `#ffffff` | `#0e0e0e` | M (pixels) |
| border as painted | | `#e7e7e8` | `#252525` | M (pixels) |
| section divider | `--border` 65% | `#e4e4e7a6`, paints `#ededef` | `#ffffff0d`, paints `#1a1a1a` | M |
| section heading | `--muted-foreground`, 11px / 16px, 500 | `#71717b` | `#a1a1a1` | M |

### 3.6 Sections and rows

| Item | Value | M/S |
|---|---|---|
| section padding | 8 8 10 8, content width 262 | M |
| divider | 1px top border on a separated section that follows a section with content. Version Control is separated only in full density. Servers, Automations, Lineage, Side chats, Usage limits and Context window are always separated. Workspace never is | M |
| heading row | min height 32, margin-bottom 4, padding 0 10, gap 8. Actions at the right, gap 4 | M |
| row | 262×32, radius 8, 1px transparent border, padding 0 10, gap 10. 13px / 19.5px, weight 500, left aligned, no wrap | M |
| row icon and label columns | a 16px icon at row x + 11, the label at row x + 37 | M |
| 14px icons (`plus`, `git-branch-plus`) | at row x + 11. The label gets `margin-left: 2px`, so it still starts at x + 37 | M |
| label | `text-box: trim-both cap alphabetic`, `overflow-x: clip`, `overflow-y: visible`, ellipsis. Box = cap height (9.45 at 13px in Inter), centred 11.27 from the row top | M |
| select row | button 262×32, gap 0, padding 0. Label half 229×30 at x + 1, padding 0 10, gap 10, `rounded-s-lg`. Icon half 32×30 with `margin-right: -1px`, `rounded-e-lg`, chevron 16 centred at x + 238 | M |
| select hairline | the icon half's `::before`: 1×16, `left: -1px`, centred vertically, transparent at rest | M |
| split group | 262×32, `rounded-lg`. Primary 229×32 (`rounded-e-none`), hairline span 1×16 at x + 229, secondary 32×32 (`rounded-s-none`, padding 0) | M |
| locked row | a `span`, 262×32, radius 8, 1px transparent border, padding 0 10, gap 10. Label flexes, trailing 16px icon | M |
| icon control | 24×24, radius 6 | M |
| meta control | 24 tall, padding 0 6, gap 4, radius 6, 12px / 16px, 500 | M |
| meta pill (primary, destructive) | 20 tall, padding 0 6, gap 4, radius 6, 12px / 16px, 500 | M |
| multiline row | min height 32, padding 6 10, `align-items: flex-start`, icons `margin-top: 1px` | M |

Row colours and states. All rows share these unless a later table says otherwise.

| State | Token or formula | Light | Dark | M/S |
|---|---|---|---|---|
| text | `--foreground` 80% | `#18181bcc` | `#f5f5f5cc` | M |
| locked run-context text | `text-foreground/80`, `dark:text-muted-foreground` | `#18181bcc` | `#a1a1a1` | M |
| 16px icons | `--muted-foreground` | `#71717b` | `#a1a1a1` | M |
| 14px icons (`plus`, `git-branch-plus`) | the text colour | `#18181bcc` | `#f5f5f5cc` | M |
| rest | transparent, no shadow | | | M |
| hover, pressed (`data-pressed`) | `!important` black 3.5% / white 8% | `#00000009`, paints `#f6f6f6` | `#ffffff14`, paints `#202020` over `#0e0e0e` | M |
| select or split, the hovered half | a second layer of the same fill on the hovered half; the other half keeps the group's single layer | hovered half paints `#eeeeee`, other half `#f6f6f6` | hovered half `#313131`, other half `#202020` | M |
| hairline while hovered or open | `--border` 65% | `#e4e4e7a6`, paints `#eaeaec` | `#ffffff0d`, paints `#2c2c2c` | M |
| menu open (`data-popup-open`) | the hover fill stays on, the hairline stays visible | | | S |
| press | `scale: 0.97` on rows without `aria-haspopup`. "Show N more" and the branch trigger opt out with `active:scale-100` | | | S |
| focus-visible | `box-shadow: inset 0 0 0 1px --background, inset 0 0 0 3px --ring` | `#fcfcfc`, `#1b4ed8` | `#000000`, `#346bf1` | M |
| disabled | opacity .64. `:disabled` also drops pointer events. `aria-disabled` keeps hover (for reason popovers) and shows `cursor: not-allowed` | | | M |
| muted tone ("Show N more") | `--muted-foreground`, hover `--foreground` 80% | `#71717b` | `#a1a1a1` | M |

### 3.7 Workspace section

Run context (environment row).

| Item | Value | M/S |
|---|---|---|
| editable (a draft, or a server thread with no messages) | select row. `aria-label` "Run context: <machine or auto label>, <workspace label>" (the first part is dropped when there is none). With the environment indicator (§5.6): machine icon 18×18 (`size-4.5`, −1px side margins) at x + 10, the machine label, a trailing workspace icon 16, the chevron half. Without it: the workspace icon 16 leads, the label is the workspace label, no trailing icon | M |
| locked (a started thread) | the `span` row, same metrics, no chevron half. Label 188 wide at 1440 with the trailing icon | M |
| machine icon | `EnvironmentMachineIcon` by kind: `server`, `cloud`, `linux` (a Linux mark), `desktop` (`monitor`), `laptop`, custom `mac-mini` and `mac-studio` drawings (24 viewBox, stroke 1.75); `server` when the machine is unknown. `scale` 16 while auto-balancing | M / S |
| workspace icon | `folder-git-2` whenever the effective mode is worktree: a draft set to New worktree, and every server thread that has a worktree or is preparing one. `folder-git` for a draft attached to an existing worktree (local mode, "Current worktree"). `folder` otherwise ("Current checkout", "Local checkout") (BranchToolbar.tsx:190–195, BranchToolbar.logic.ts:158–172). Measured: `folder-git-2` on a worktree thread and on a No project thread, `folder` on a draft in the checkout | M |
| tooltips | machine icon: the environment label, "Run on" if none, the auto label while auto-balancing. Workspace icon: the workspace path, the label if there is no path, or "Each model starts in its own worktree." when every model gets its own | S |
| label | with the indicator: the auto-balance label, else the machine label, else "Run on". Without it: the workspace label (BranchToolbar.tsx:249–256, §5.6) | M |
| menu | Base UI menu below, `align start`, width = row width (262), offset 4: 1157, 109, 262×138 | M |
| menu content | "Run on" radio group (12px machine icons) plus an "Add to another machine" submenu at opacity .55 until highlighted, then a separator, only with more than one machine or something to add. "Workspace" radio group: Current checkout or Current worktree (`folder` / `folder-git` 12), New worktree (`folder-git-2` 12), Previous worktree (`history` 12, two lines: label, then the branch in 12px muted) | M (Workspace) / S (Run on) |
| menu colours | 02-primitives.md §3.5. Checked radio row `--foreground` 8%: `#18181b14` / `#f5f5f514` | M |

Open in editor.

| Item | Value | M/S |
|---|---|---|
| split | primary "Open in <Editor>" with that editor's icon 16, or "Open in editor" with `square-arrow-out-up-right` 16 muted when there is no preferred editor. Secondary "Choose editor" chevron | M (disabled) |
| editor icons | brand marks in full colour (`text-foreground opacity-100` for brand kinds); the generic file manager icon on Linux (`folder-closed`) is muted | S |
| disabled | primary: no preferred editor, no folder, or no way to open. Secondary: the host refuses editor opens for this client | M |
| tooltips | "Open in <Editor>" on the primary, "Choose editor" on the secondary, side top | S |
| menu | `align end`, anchored to the whole split group, width = group width. One row per available editor (icon 16 + label, the preferred one shows the `editor.openFavorite` shortcut). Disabled informational rows: "No installed editors found", "No SSH route to <machine>", "Open from a computer to edit on <machine>", "Opens over SSH. Needs your key on <machine>" | S |

Project scripts.

| State | Look | M/S |
|---|---|---|
| no scripts | row: `plus` 14 at x + 11 + "Add project script" (label `ml-0.5`, so it starts at x + 37); tooltip "Add project script", side top | M |
| a primary script | `div[role=group][aria-label="Project scripts"]` split: `ScriptIcon` 16 + script name (`aria-label` "Run <name>") / chevron "Script actions". Tooltips "Run <name>" and "Script actions", side top. The card always passes a run handler, so the primary is never `aria-disabled` and the menu rows are never disabled; a client without terminal write access gets a click that does nothing (useChatTerminalActions.ts:212). The "Pair this client again with permission to run terminal commands." tooltip only appears where no handler is passed (ProjectScriptsControl.tsx:298, 316–320) | M (enabled split) / S |
| no primary (every script is a settle script, or there are only t3.json scripts to import) | `div[role=group][aria-label="Project actions"]` split: `wrench` 16 + "Actions" (`aria-label` "Project actions") / chevron "Choose project action"; both halves open the menu | S |
| menu | `align end`, anchored to the split, width = split width, `highlightItemOnHover` off for the Actions form. Rows: `ScriptIcon` 16 + name with roles ("<name> (setup, on settle)") + its shortcut; on hover or focus the shortcut fades out and a 24×24 ghost gear (`settings` 14, "Edit <name>") fades in. Then "From t3.json" group with `download` 14 import marks, then `plus` 16 "Add project script" | S |
| script icons | `play` (default), `flask-conical` (test), `list-checks` (lint), `wrench` (configure), `hammer` (build), `bug` (debug) | S |

### 3.8 Version Control section

Branch row.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| trigger | `button[role=combobox][data-slot=combobox-trigger][aria-haspopup=dialog]` as a select row: `git-branch` 16 muted, the label in `MiddleTruncate` (cap-aligned variant, 02-primitives.md §5.14), chevron half. The label span carries `title=<ref>`, so only the browser's native tooltip shows the full name | `#18181bcc` | `#f5f5f5cc` | M |
| loading | disabled (opacity .64) until the first page of refs loads, and while a branch action runs | | | M |
| popup | Combobox popup (02-primitives.md §3.13), `align start`, side bottom, offset 4, width = trigger width: 1157, 224, 262×349 with a full list. Frame `dropdown-glass`, radius 8, positioner z 130 | | | M |
| search | "Search refs..." 14px, rule 1px `--border` 70% turning `--ring` while focused, block padding 10 12 0 | `#1b4ed8` focused | `#346bf1` | M |
| list | `max-height: 224px` (`max-h-56`), padding 8 0 8 4, stable gutter, edge fades while scrolled | | | M / S |
| ref item | 28 tall, padding 4 8, radius 4, 14px, name in `MiddleTruncate`, trailing tag 10px / 14px (`text-3xs`) `--muted-foreground` 45%: "current", "worktree" (checked out in another worktree), "remote", "default" (first that applies) | tag `#71717b73` | `#a1a1a173` | M |
| highlighted item | `--foreground` about 9% (the first item is highlighted on open, `autoHighlight`) | `#18181b17` | `#f5f5f517` | M |
| status lines | "Loading refs...", "Loading more refs...", "Showing N of M refs" (measured "Showing 100 of N refs" for a large repository), the notice "Stop the current turn to switch worktrees.", and the empty state "No refs found." (BranchPicker.tsx:129, 186–187) | | | M / S |
| start from remote | footer `label`: `refresh` icon 12 + "Start from remote" (12px 500 muted) + Switch `sm` (`aria-label` "Start worktree from remote"), padding 8 12, gap 12, top border `--border` 60%; only while choosing a new worktree's base. Tooltip, side top, max width 288: "Uses the latest commit of the base branch’s tracked remote branch, even if its name differs. Otherwise uses the same-named branch on origin. Uses your local branch if that remote branch is missing." (BranchPicker.tsx:157–185) | | | S |

Pull request rows. A thread can show its current pull request with details and actions, plus a "Show N more" list of its other linked ones.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| detailed row wrapper | padding-bottom 6 | | | M |
| open button | multiline row, 262 wide, 50 tall for a two-line title, padding 6 10, `align-items: flex-start`, state icon 16 at `margin-top: 1px`. Its `aria-label` is the pull request's URL (every open row, detailed or simple) | | | M |
| title | 13px / 18px (`leading-4.5`), 500, two lines then ellipsis (`line-clamp: 2`), `white-space: normal` | `#18181bcc` | `#f5f5f5cc` | M |
| meta line | `margin-top: 6px`, 20 tall, padding 0 10 0 36, gap 2, 12px / 16px `--muted-foreground` | `#71717b` | `#a1a1a1` | M |
| number | "#N", tabular figures, `margin-right: 4px` | | | M |
| checks trigger | meta control: 24 tall, padding 0 6, gap 4; the checks icon 14 in its tone; text "N failed" or "N/M checks" when there is something to say. `aria-label` "Open checks: <summary>" | | | M |
| Merge or Ready pill | 20 tall, `--primary`, white text, 1px inset ring in `--background` when focused | `#1b4ed8`, hover `#1b4ed8d9` | `#346bf1`, hover `#346bf1d9` | M |
| Resolve or Fix pill | 20 tall, padding 0 6, gap 4, radius 6, 12px / 16px 500, `--destructive` 10% fill, `--destructive` text, hover 16%; Resolve and Fix end with `arrow-up-right` 12 when they open a new thread; Fix shows a 12px spinner while preparing. Measured on a conflicting pull request under the branch row: "Resolve" + arrow, 75.2×20, pushed to the meta line's end, no checks trigger; tooltip "Check the branch out and resolve the conflicts in a new thread", side top (fits above the row), 320 wide, two lines | `#fb2c361a` / `#fb2c36`, hover `#fb2c3629` | `#fb414a1a` / `#fb414a`, hover `#fb414a29` | M (Resolve) / S (Fix) |
| simple row | a 32px row: state icon + "#N: <title>" (or "#N") on one line. Tooltip, side top: the status indicator's tooltip, else "Pull request #N" (capitalised singular) | | | S |
| watching (simple) | a split: open row as primary, `eye` 16 as secondary (`aria-label` "Stop watching #N"), `eye-off` while hovered. In the detailed form an `eye` 14 icon control sits in the meta line. Tooltip, side top: "Watching: the agent wakes when checks finish, someone comments, or the branch conflicts. Click to stop." | | | S |
| Show N more / Show less | muted row, `chevron-down` / `chevron-up` 16 | `#71717b` | `#a1a1a1` | M |
| expanded list | `div[data-thread-details-pr-list]`, each linked pull request as its own row, same layout | | | M |

State icons, tones and labels (pullRequestIcons.tsx:33–54; the labels "Open", "Draft", "Closed", "Merged" appear in tooltips and in `PullRequestContextDetails`), and checks tones (pullRequestPresentation.tsx:199–300):

| State | Icon (16 in rows, 12 in tabs) | Light | Dark | M/S |
|---|---|---|---|---|
| open | `git-pull-request-arrow` | emerald-600 `#009966` | emerald-300 90% `#5ee9b5e6` | M |
| draft | `git-pull-request-draft` | zinc-500 `#71717b` | zinc-400 80% `#9f9fa9cc` | P |
| closed | `git-pull-request-closed` | red-600 `#e7000b` | red-300 90% `#ffa2a2e6` | P |
| merged | `git-merge` | violet-600 `#7f22fe` | violet-300 90% `#c4b4ffe6` | M / P |
| unknown state | `git-pull-request-arrow` | `--muted-foreground` | | S |
| checks passing | `circle-check` 14 | `#009966` | `#5ee9b5e6` | M |
| checks failing | `circle-x` 14, `--destructive` | `#fb2c36` | `#fb414a` | M |
| checks pending | `circle-dot` 14, amber-600 / amber-400 90% | `#e17100` | `#ffb900e6` | P |

Rich tooltip on a detailed row (ThreadDetailsPrRow.tsx:268–342), measured: tooltip `variant="glass"` (`#ffffffd5`, 1px `#18181b1a`, `0 20px 25px -5px #00000040, 0 8px 10px -6px #00000040`), max width 320, declared side top / align start / offset 4. At the card's position it flips to below and end-aligned: 1099, 242, 320×214. Content, padding 8 4, gap 8: a 12px line with the title (500, `--foreground`, truncated) and "#N" muted; then rows with gap 6, 12px muted, 12px icons: state icon + state label (`--foreground` 75%), `git-branch` + "<base> ← <head>", checks icon + summary (open pull requests with checks), `triangle-alert` + "Merge conflicts with <base>" in `--destructive` (drafts with conflicts), `file-diff` + "N files" and `+N` / `-N`, then the per-category breakdown (Code, Tests, Generated, Docs) indented 20.

Checks popover (PullRequestChecksPopover.tsx:169–216), measured: Base UI popover `width="md"` (320), glass with the elevation shadow (`0 16px 40px -18px #0000008c`), side bottom, declared align start, flipped to end: 922.39, 270, 320×162. Content: the checks state label 14px 500, a 12px muted summary, then rows (status icon 14, name truncated, status label muted, "Details" in `--primary`).

Git actions.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| split group | 262×32. While an action runs the group fills black 3.5% / white 5.5% and the secondary is replaced by the elapsed time | `#00000009` | `#ffffff0e` | M (rest) / P |
| quick action | primary: icon 16 in `--muted-foreground` (the provider mark too, measured on "Commit, push & PR") + label (§5.11). When it can run, its tooltip is the label, side top (GitActionsControl.tsx:1654–1683) | | | M |
| unavailable quick action | `aria-disabled`, opacity .64, hover opens a reason popover after 300ms: tooltip-style popover (glass, radius 6, 12px / 16px, padding 4 8, `shadow-md/5`), side bottom, declared align start, flipped to end: 1100.94, 256, 285×26 | | | M |
| options | secondary "Git action options" chevron, tooltip side top; disabled while an action runs or when the menu would be empty. Menu `align end`, anchored to the split, width = split width | | | M / S |
| menu rows | Commit (`git-commit-horizontal`), Push (`cloud-upload`), Create PR (provider icon), each 16. A disabled row explains itself in a hover popover, side left. Then "Publish repository..." (`cloud-upload`), then notices: 12px `--warning` "Detached HEAD: create and check out a branch to enable push and pull request actions.", "Behind upstream. Pull/rebase first.", or the status error in `--destructive`, padding 6 8 | | | S |
| progress | grid `auto 1fr`, column gap 12, row 18: spinner, status (13px, cap-trimmed); the last output line 11px muted below it (row grows over 200ms); elapsed "Ns" or "Nm Ns" 11px tabular muted, 32 tall, padding-right 10 | | | S |
| success | `check` 14 in `--success` `#00bc7d`, the title, an optional 11px muted description; for 10s | | | S |
| Initialize Git | row: `git-branch-plus` 14 + "Initialize Git" (label `ml-0.5`), "Initializing..." while it runs; disabled without the source-control write scope. It is checked before the compact rule, so it still shows in compact and essential density (GitActionsControl.tsx:1574–1588) | | | M |

Changes row.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| row | 262×32 button, `file-diff` 16 muted. Disabled (opacity .64) where it cannot open the diff (drafts, folders without Git) | | | M |
| inner | `span.flex.min-w-0.flex-1.items-baseline.gap-1.5`, 20.5 tall | | | M |
| "Changes" | 13px, 500, `--foreground` 80% | `#18181bcc` | `#f5f5f5cc` | M |
| base label | "Uncommitted" or "vs <baseRef>", 13px 400 `--muted-foreground`, truncates; tooltip with the same text (max width 384), side top | `#71717b` | `#a1a1a1` | M |
| counts | pushed right, padding-left 8, gap 8. Mono 13px / 19.5px, 500, tabular figures | | | M |
| insertions above 0 | `--diff-addition-foreground` | `#007a55` | `#00d492` | M |
| deletions above 0 | `--diff-deletion-foreground` | `#c10007` | `#ff6467` | M |
| a zero count | `--muted-foreground` | `#71717b` | `#a1a1a1` | M |
| format | "+N" and "-N", N through `Intl.NumberFormat()` (en: "2,382") | | | M |

### 3.9 The card's own menu

Right-clicking the card (outside fields, links and selected text) opens a Base UI menu at the pointer, side bottom, `align start`, offset 4. The same menu opens from "Choose what to show", anchored to that row's bottom-left. Measured: 1153, 73, 232.02×411 (02-primitives.md §3.5 styling): group label "Show in thread details", 12 checkbox items with a check for each visible item (Environment and worktree, Open in editor, Project scripts, Branch and pull request, Git actions, Changes, Servers, Automations, Lineage, Side chats, Usage limits, Context window), a separator, "Reset to default" (disabled while everything is at its default).

### 3.10 Lower sections

| Section | Structure and values | M/S |
|---|---|---|
| Automations | heading "Automations" (11px / 16px 500 muted, 66.4 wide) with an icon control "Manage scheduled tasks" (`settings-2` 14, opens Settings › Scheduled tasks for this environment; tooltip "Manage scheduled tasks"). Only tasks bound to this thread are listed; the section is absent when there are none and no load error. List `max-height: 216px`, scrolls. Each task: `li` padding 6 10, gap 10, radius 8, 48 tall, no hover fill of its own; `calendar-clock` 16 muted with a 6px status dot at −4/−4 top right (never run `--muted-foreground` 40%, running sky-500 `#00a6f4`, succeeded emerald-500 `#00bc7d`, failed `--destructive`); title 14px / 20px 500 `--foreground` 80% and a schedule line 11px / 16px muted, both truncated, tooltip side left (title in 500, then the schedule line muted). Schedule line: the schedule's label, then " · <policy>" when a policy blocks it, " · next <relative time>" when enabled with a next run, nothing more when enabled without one, " · paused" when off. "Edit <task>" (`pencil` 14, tooltip "Edit automation", opens the task in settings) and "Run <task> now" (`play` 14, tooltip "Run now", absent for webhook tasks, disabled while the task runs, while another toggle or run is pending, or without permission) icon controls. The icon controls set no text colour, so their glyphs inherit the card's `--foreground` (`#18181b` / `#f5f5f5` measured), not the 80% row colour. A default Switch (30×18, `--primary` when on) labelled "Pause <task>" / "Resume <task>", disabled under the same rules. A load failure adds "Could not load automations: <error>" (11px `--destructive`, padding 6 10); failed toggles and runs toast "Could not update automation" / "Could not run automation" | M |
| Lineage | heading "Lineage", or "Lineage · N running" while subagents run (measured "Lineage · 3 running"). The heading is hidden when the only content is the "Previous agents" row. With a detachable agent session the heading carries an icon control "More thread actions" (`ellipsis` 14) whose menu (`align end`, min width 240) has one item, `unplug` 14 "Disconnect agent session". A thread delegated from a linked environment first shows a "delegated from" row. Then rows come in three groups (ThreadRelationshipsControl.tsx:545–577): related threads (the parent, forks, context transfers), active subagents (including tasks running on a linked environment), and "Previous agents" (finished ones: completed, failed, error, cancelled, interrupted, idle). The last is a collapsible row: `bot` 16, "Previous agents", "N failed" (10px, line height 1, `--destructive-foreground`) when any failed, the count (12px 400 tabular, muted, hidden while expanded) and `chevron-down` 16 turned −90° while collapsed (`transition-transform`); collapsed by default. Each list shows 6 rows, then a "Show N more" button (`plus` 16, up to 12 per click; 14px 500 `--muted-foreground` 70%, hover black 5.5% / white 7.5% and `--foreground` 80%). List `max-height: 216px`. Row `li` 32 tall, radius 8; the row button is a 32px row. Icon 16 muted: `corner-left-up` (parent), `messages-square` (side chat), `git-fork` (fork), and for a subagent its provider's icon (`bot` when unknown). An 8px status dot with a 2px `--card` border sits at −4/−4 bottom right (top left when the provider icon carries an instance badge): running, in progress, pending or waiting `--info` `#2b7fff`; failed or error `--destructive`; completed `--success`; otherwise `--muted-foreground` 45% (ThreadRelationshipIcon.tsx:81–95). Title 14px / 16px 500 `--foreground` 85%, truncated. Rows without an agent end with `arrow-right` 12 muted at opacity 0 until the row is hovered. Agent rows end with the elapsed time (11px 400 tabular muted, "5s", "11s") or, when finished, the status label (11px 500, `--destructive` when failed, muted otherwise). A stoppable agent adds a destructive icon control "Stop subagent <title>" (`square` 12 filled, a spinner while stopping; tooltip "Stop subagent", side left) at right 4: it appears on row hover or focus-within (always on coarse pointers and no-hover devices) while the trailing time hides. Row tooltips open after 200ms, side left: the full title plus "Open <relationship> in this chat", or a hover card with the agent's model, status and result for subagents. A fork that can merge back is a split: the open row, then a 32px secondary "Merge back to <parent>" (`git-merge` 12, spinner while merging; tooltip "Merge this conversation back into <parent>", or "Complete a run in this fork before merging it back"), then its status label | M / S |
| Side chats | the same rows under the heading "Side chats", or "Side chats · N running" | S |
| Servers | heading "Servers". Each server is a multiline row: `globe` 16, "localhost:<port>" 14px / 16px 500 `--foreground` 85%, a second line "Agent" or "Terminal" and the process name joined by " · ", 11px muted. Click (or middle-click) opens it in the browser panel; tooltip after 200ms, side left: "Open in the browser panel. <modifier>-click for your browser." A destructive icon control "Stop localhost:<port>" (`square` 12 filled, `loader-circle` 12 spinning while stopping; tooltip "Stop server") appears at right 4 on hover or focus-within, when the server reports stop support and a pid. Four per page, then "Show 4 more" / "Show less" (`active:scale-100`) | S |
| Usage limits, Context window | `grid grid-cols-[auto_minmax(3rem,1fr)_auto] gap-x-3 text-xs`. Each row 24 tall, padding 0 10, radius 8, hover fill as rows; label muted (max 112), a 3px bar (`--muted-foreground` 30%, 20% in dark) with its fill and a 1×9 marker in `--foreground` 60%, then the value. Off by default | S |
| empty fallback | `div.p-2` > row `sliders-horizontal` 16 + "Choose what to show" (label `ml-0.5`) | S |

### 3.11 Right panel tab bar

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| bar | 901, 0, 539×52 inline. `gap: 4px`, padding 0 112 0 8 (`pl-2 pr-28`); the sheet uses padding-right 12 and ends with its controls. No bottom border | | | M |
| tab list | 909..1328 (419 wide) inline; horizontal ScrollArea, scrollbars hidden, edge fades | | | M |
| tab | 24 tall, `max-width: 144px`, padding 0 8 0 6, gap 2, radius 6, 12px / 16px, 400. y 14 | | | M |
| inactive tab | `--muted-foreground`, transparent | `#71717b` | `#a1a1a1` | M |
| inactive tab, hover | `--accent` 60%, text `--foreground` | `#f4f4f599`, `#18181b` | `#ffffff06`, `#f5f5f5` | M |
| active tab | `--accent`, text `--foreground` | `#f4f4f5`, `#18181b` | `#ffffff0a`, `#f5f5f5` | M |
| close button | 16×16, radius 4. Shows the surface icon 12 at rest, `x` 12 while the tab is hovered or the button has keyboard focus. Hover fill `--muted` | `#fafafa` | `#ffffff08` | M |
| pending dot | 6px, current colour, at −2/−2 bottom right of the icon slot (a file still loading) | | | S |
| audio button | 16×16, radius 4, hover `--muted`; `volume-2` or `volume-off` 12, tooltip "Mute tab" / "Unmute tab" | | | S |
| title | one line, ellipsis. Tooltip = the title; device tabs add "<host> · <version>" in muted. The tooltip is declared with the default side top and flips below: measured centred on the title, top 38 (title bottom 34 + 4) | | | M / S |
| add button | Button `icon-xs` `ghost-muted`, 24×24, radius 8, `plus` 14 `--muted-foreground`; hover `--accent` and `--foreground`. Keyboard focus ring `inset 0 0 0 1px #fcfcfc, inset 0 0 0 3px #1b4ed8` | `#71717b` | `#a1a1a1` | M |
| add menu | Base UI menu, `align start`, side bottom, offset 6: 909, 44, 194.63×206 (re-measured). Rows: Browser (`earth`; whenever the browser is available this row is a submenu trigger listing the browser profiles, max width 224, even with a single profile), Terminal (`square-terminal`), Files (`files`), Diff (`file-diff`), Pull request (`git-pull-request-arrow`), Linked pull requests (`link-2`), Device (`smartphone`), Side chat (`messages-square`, where offered). Icons 16, shortcut letters B T F D P L M S as `MenuShortcut` (12px muted, `letter-spacing: 1.2px`). Unavailable rows are disabled (opacity .64) with a reason tooltip | | | M |
| scroll buttons | only while the tabs overflow: Button `icon-xs` `ghost`, 24×24, `chevron-left` / `chevron-right` 14, group gap 2, disabled at their end, tooltips "Scroll tabs left" / "Scroll tabs right" | | | S |

Surface tab icons and titles (RightPanelTabs.tsx:572–698):

| Surface | Icon, 12px | Title |
|---|---|---|
| browser | the page favicon (captured from the page when same-origin, else a public favicon service), fallback `earth` | page title, else the URL's host, else "Browser" |
| terminal | `square-terminal` | the terminal's label ("Terminal 1" style) |
| files | `files` | "Files" |
| file | the file-type icon for its path | the file name |
| diff | `file-diff` | "Diff" |
| pull request | the state icon in its tone (§3.8), muted `git-pull-request-arrow` until known | "#N" |
| linked pull requests | `link-2` | "Pull requests" |
| side chat | `messages-square` with a 6px dot at −2/−2 top right: `--muted-foreground` while running, `--info` when it needs approval or input | the side chat's title |
| device | Apple or Android mark by platform, else `smartphone` | the device title or name, else "Device" |

### 3.12 Right panel launcher ("Open a surface")

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| container | fills the surface area (901, 52, 539×848), centred, padding 0 24 52 24 so the list centres on the window rather than the area, focusable (`tabindex=0`, `aria-label="Open a surface"`, `data-workspace-focus-default`), `scrollbar-gutter: stable both-edges`. `data-surface-launcher-keys` lists the letters of the rows that are available (measured "BTFDPLM": no side chat row) | | | M |
| inner | 320 wide (`max-w-xs`), padding 24 0 | | | M |
| heading | "Open a surface", 14px / 20px 500 `--foreground`, centred, margin-bottom 12 | `#18181b` | `#f5f5f5` | M |
| rows | 320×32, gap 2, padding 0 10, gap 10, radius 8, 14px / 20px 400 in the inherited body colour; icon 16; label flexes; `Kbd` 20×20 letter at the end | `#262626` | `#f5f5f5` | M |
| row hover or arrow highlight | `--accent` 60% (`group-hover:bg-accent/60`, or `bg-accent/60` on the highlighted row), `transition-colors` 150ms | `#f4f4f599` | `#ffffff06` | M (light hover) / S |
| unavailable row | opacity .5, not a button (`aria-disabled`, `tabindex=0`), tooltip with the reason, side top | | | S |
| Kbd | 20×20, radius 4, `--foreground` 7%, 12px 500 muted | `#18181b12` | `#f5f5f512` | M |
| browser profiles | with more than one profile the Browser row's label gets `pr-7` (28px) and a 24×24 `ghost-muted` chevron button ("Open browser in a profile", `chevron-down` 14) sits at right 32, which opens a menu of profiles (`align end`, offset 6, max width 224) | | | M |

### 3.13 Icon inventory

All lucide at stroke 1.75 unless noted.

| Where | Icon | Size |
|---|---|---|
| title menu button | `chevron-down` | 14 |
| project crumb | project icon (monogram, favicon, lucide or emoji); No project: `message-square-dashed` | 14 |
| panel toggles | `square-menu`, `panel-bottom`, `panel-right`, `maximize-2` / `minimize-2` | 16 |
| guest share | `user` | 14 |
| title menu (browser fallback) | `text-search`, `message-square-plus`, `pin`, `pin-off`, `circle-check`, `clock`, `pencil`, `refresh-cw`, `mail-open`, `user-plus`, `folder-tree`, `folder-plus`, `timer`, `copy`, `folder`, `git-branch`, `message-square`, `hash`, `settings`, `archive`, `trash`, `chevron-right`, `check` | 16, **stroke 2** |
| run context | machine icon (`server`, `cloud`, `monitor`, `laptop`, custom); `scale`; `folder`, `folder-git`, `folder-git-2`; `chevron-down` | 18 (machine), 16 |
| run context menu | machine icons, `scale`, `plus`, `folder`, `folder-git`, `folder-git-2`, `history` | 12 |
| Open in | editor brand icons, `square-arrow-out-up-right`, `folder-closed` (Linux file manager), `chevron-down` | 16 |
| scripts | `plus` (14), script icons, `wrench`, `chevron-down`; menu `settings` 14, `download` 14, `plus` 16 | 16 |
| branch | `git-branch`, `chevron-down`; picker `search` 16, `refresh` 12 | 16 |
| pull requests | state icons 16; checks 14; `eye` / `eye-off` 16 or 14; `arrow-up-right` 12; `chevron-down` / `chevron-up` 16; tooltip icons 12 (`git-branch`, `triangle-alert`, `file-diff`) | as listed |
| Git actions | `git-commit-horizontal`, `cloud-upload`, `cloud-download`, provider mark (GitHub and others), `info`, `chevron-down` 16; `check` 14; `git-branch-plus` 14; spinner | as listed |
| Changes | `file-diff` | 16 |
| Automations | `settings-2` 14, `calendar-clock` 16, `pencil` 14, `play` 14 | as listed |
| Lineage | `corner-left-up`, `bot`, `messages-square`, `git-fork` 16; `arrow-right` 12 | as listed |
| Servers | `globe` 16, `square` 12 (filled), `loader-circle` 12, `chevron-down` / `chevron-up` 16 | as listed |
| empty fallback | `sliders-horizontal` | 16 |
| tab bar | surface icons 12, `x` 12, `volume-2` / `volume-off` 12, `plus` 14, `chevron-left` / `chevron-right` 14 | as listed |
| launcher | `earth`, `square-terminal`, `files`, `file-diff`, `git-pull-request-arrow`, `link-2`, `smartphone`, `messages-square` 16; `chevron-down` 14 | as listed |

## 4. States

### 4.1 Header

| State | Trigger | What changes | M/S |
|---|---|---|---|
| server thread | `serverThread !== null` | title is the menu button with the hover chevron | M |
| draft | local draft route | title is a plain `h2` "New thread": no chevron, no menu, no rename, not focusable | M |
| No project thread | project is the scratch project | project crumb reads "No project" with that project's own icon (`message-square-dashed`, gray) and still starts a new thread in it | M |
| no project at all | `activeProject` null | no project crumb and no separator, the title alone | S |
| subagent child | lineage `relationshipToParent === "subagent"` | parent crumb and a second "/" between project and title; collapses to "..." when the list overflows | S |
| linked origin | thread shell `linkOrigin` | "From <origin>" badge after the title | S |
| guest share | thread has a guest share | indicator with "Stop sharing" after the breadcrumb | S |
| renaming | double-click or "Rename thread" | input in place of the title, badge hidden, header right-click ignored | P |
| title hover | pointer over the title button | chevron opacity 1 | M |
| title focus-visible | keyboard focus | 2px inset ring, chevron visible, tooltip open | M |
| crumb hover | pointer over the project or parent crumb | text `--foreground` | M |
| inline right panel open | panel inline | ChatHeader `padding-right` 40; details toggle at the header's right | M |
| collapsed sidebar, or window < 768 | 03-shell.md §5.3 | header `padding-left` 52 (56 below 640) | M |
| read-only client | no operate scope | double-click does nothing; operate items in the menu are disabled | M |

### 4.2 Panel toggles

| Toggle | State | Look or behaviour | M/S |
|---|---|---|---|
| details | inline, card visible | pressed (`--accent`) | M |
| details | inline, card hidden | not pressed | M |
| details | popover mode, closed | not pressed, `aria-haspopup=dialog` | M |
| details | popover mode, open | pressed while the popover shows | M |
| details | inline right panel open | lives in the header at right 12 | M |
| terminal, right panel | no project | disabled, "… is unavailable" tooltip | S |
| all | mobile sidebar sheet mounted | `display: none` (03-shell.md §3.6) | M |

### 4.3 Card mode and density

| State | Condition | Result | M/S |
|---|---|---|---|
| inline, full | slot exists, full content fits | everything visible, Version Control has its divider | M |
| inline, compact | full content taller than the slot, compact content fits | no environment row, no Git actions row (unless one is running or just succeeded; "Initialize Git" stays), no Servers / Automations / Lineage / Side chats, no Version Control divider | M (1440×250) |
| inline, essential | compact content taller too | compact minus the Open in editor row | M (1440×238) |
| popover, closed | no slot: canvas < 1012 wide, or slot height < 160 (canvas < 184 tall, < 232 with the find bar), or a preview mini player cutting the slot under 160 | nothing on the canvas; header toggle is a popover trigger | M (1440×235, 1267×900, and 1440 with the inline panel) |
| popover, open | toggle clicked or shortcut | card under the header's bottom-right, density judged against canvas height − 52 | M |
| hidden | inline and toggled off | no card; chat lanes centre (03-shell.md §5.4) | M |
| find bar open | thread find active | inline card moves down 48 to canvas y 60 (animated over 150ms ease-out) and its max height drops by 48; closing the bar snaps it back to 12 without a transition (the class is only present while the inset is non-zero). The popover gets a 48px side offset | M (inline) / S (popover) |
| content scrolls | content taller than the card | ScrollArea scrolls, edges fade | S |

### 4.4 Card content by thread kind

| Thread | Workspace | Version Control | Lower sections | M/S |
|---|---|---|---|---|
| draft in a Git project | editable run context (select row), Open in, scripts | branch picker; Git actions and Changes hidden while the draft has no changes (`quietEmptyDraft`) | none (drafts get none) | M |
| server thread with messages, Git repo, clean | locked run context, Open in, scripts | branch, the pull request rows if any, Git actions, "Changes +0 -0" | Servers, Automations, Lineage, Side chats when they have content | M |
| server thread, dirty on the default branch | as above | quick action "Commit & push"; Changes "vs <base>" with counts | as above | M |
| server thread, worktree on a feature branch, merged pull request | locked run context with `folder-git-2` | branch, a detailed pull request row (merged icon, "#N", checks), quick action "Commit, push & PR" when dirty, Changes | as above | M |
| server thread with no runs yet | run context may still change workspace mode | Git actions and Changes hidden while there is nothing to do | as above | S |
| folder that is not a Git repository | as above | linked pull request rows if any, then "Initialize Git". No branch row, no Changes | as above | M |
| No project thread | run context (`folder-git-2`), Open in and scripts for its own folder | no branch, Git actions or Changes. Linked pull requests still show (detailed row, Merge pill, "Show N more") | as above | M |
| thread without a project | nothing from Open in or scripts; the run-context row returns nothing | nothing | as above | S |
| loading (thread shell or Git status not loaded yet) | run context shows a chevron until the shell says the thread started | branch trigger disabled; quick action "Commit" disabled ("Git status is unavailable."); Changes "+0 -0", both muted, no base label | appears when loaded | M |
| read-only client | Open in disabled (both halves: the host refuses editor opens without the operate scope); the script split stays enabled (a run without terminal write access does nothing) | quick action `aria-disabled` with "This connection cannot change source control."; options chevron disabled; Initialize Git disabled; branch items that need a checkout disabled | automation run and switch disabled | M |

### 4.5 Git status → quick action

`resolveQuickAction` (GitActionsControl.logic.ts:191–341), first match wins. `PR` is the provider's short label ("PR", or "MR" on GitLab), `pull request` its singular.

| # | Condition | Label | Kind / action | Icon |
|---|---|---|---|---|
| 1 | a Git action is running | Commit | hint "Git action in progress." | `git-commit-horizontal` |
| 2 | no status | Commit | hint "Git status is unavailable." | `git-commit-horizontal` |
| 3 | detached HEAD | Commit | hint "Create and checkout a ref before pushing or opening a pull request." | `git-commit-horizontal` |
| 4 | dirty, and behind or (no upstream and no primary remote) | Commit | run `commit` | `git-commit-horizontal` |
| 5 | dirty, open pull request or default branch | Commit & push | run `commit_push` | `cloud-upload` |
| 6 | dirty otherwise | Commit, push & PR | run `commit_push_pr` | provider mark |
| 7 | clean, no upstream, no primary remote | Publish repository | open the publish dialog | `cloud-upload` |
| 8 | clean, no upstream, not ahead, open pull request | Commit | hint "Branch is up to date. No action needed." | `git-commit-horizontal` |
| 9 | clean, no upstream, not ahead | Push | hint "No local commits to push." | `cloud-upload` |
| 10 | clean, no upstream, ahead, open pull request or default branch | Push | run `commit_push` on the default branch, else `push` | `cloud-upload` |
| 11 | clean, no upstream, ahead | Push & create PR | run `create_pr` | provider mark |
| 12 | ahead and behind | Sync ref | hint "Branch has diverged from upstream. Rebase/merge first." | `info` |
| 13 | behind | Pull | run pull | `cloud-download` |
| 14 | ahead, open pull request or default branch | Push | run `commit_push` (default branch) or `push` | `cloud-upload` |
| 15 | ahead | Push & create PR | run `create_pr` | provider mark |
| 16 | open pull request with an upstream | Commit | hint "Branch is up to date. No action needed." | `git-commit-horizontal` |
| 17 | ahead of the default branch, not on it | Create PR | run `create_pr` | provider mark |
| 18 | otherwise | Commit | hint "Branch is up to date. No action needed." | `git-commit-horizontal` |

The reason shown on hover is "This connection cannot change source control." for a client without the source-control write scope, else the hint of a disabled action (GitActionsControl.tsx:1187–1191). Hints are disabled actions: the primary is `aria-disabled` and opens the reason popover instead of acting.

### 4.6 Pull request row actions

`resolveThreadPanelPullRequestAction` (pullRequestDetail.logic.ts:364–379) on the loaded detail:

| Detail | Pill | Pending label | Tooltip | M/S |
|---|---|---|---|---|
| not open, or no detail | none | | | M (merged) |
| open with merge conflicts | Resolve, destructive, `arrow-up-right` when it opens a new thread; the checks trigger is dropped | "Preparing..." | "Check the branch out and resolve the conflicts in a new thread" or "Resolve the conflicts in this thread" | M (under the branch row) |
| open draft that may be marked ready | Ready, primary | "Marking..." | "Mark this pull request as ready for review" | S |
| open, a check failed or was cancelled | Fix, destructive, `arrow-up-right` or a spinner while preparing | "Fix" | "Fix the failing checks in a new thread" or "… in this thread" | S |
| open, checks pending or awaiting action | none | | | S |
| open, passing or no checks, merge allowed with at least one method | Merge, primary | "Merging..." | "Merge this pull request (<method>)" | M |

Detailed layout (title row plus meta line) is used when the detail is loaded and there are checks to show, a pill, or a watch control. Checks show unless the pull request is a draft or conflicting. Pills are disabled while any action or hand-off runs. Merge first asks: AlertDialog "Merge pull request?", "This merges #N using <method>.", buttons "Cancel" (outline sm) and "Merge" (default sm).

### 4.7 Right panel tabs

| State | Look | M/S |
|---|---|---|
| inactive | transparent, `--muted-foreground` | M |
| inactive, hover | `--accent` 60%, `--foreground`; the icon slot shows `x` | M |
| active | `--accent`, `--foreground` | M |
| active, hover | the icon slot shows `x` | M |
| close button focus-visible | the slot shows `x`. The close and title buttons have no focus style of their own, so Chrome's `:focus-visible` ring shows (`outline: auto 1px`, author colour `--ring` 50% from the `*` rule, `outline-offset: -2px`) | S |
| dragging | `position: relative; z-index: 10`, follows the pointer on the x axis. Pointer only: the one sensor is a `PointerSensor` (6px), so there is no keyboard reordering even though dnd-kit's screen-reader instructions are in the DOM (RightPanelTabs.tsx:833) | S |
| pending | 6px dot under the icon | S |
| audible browser tab | speaker button after the close button | S |
| renaming a device tab | 96px input (`w-24`), `--background` fill, 1px `--ring` inset ring | S |
| overflow | scroll buttons appear; edge fades; the active tab scrolls into view | S |
| panel closed | the tab bar unmounts after the exit delay (03-shell.md §5.5) | M |

## 5. Behaviour

### 5.1 Breadcrumb data

- Project name: `activeProject.isScratch ? "No project" : activeProject.title` (ChatHeader.tsx:85).
- Project icon: the project's explicit icon if it has one (monogram, emoji or lucide name, in a project colour); otherwise its favicon file through the asset server; otherwise a monogram from the name with a colour derived from the name (ProjectFavicon.tsx:35–116, projectIdentity.ts). The sidebar uses the same component.
- Title: the thread title, "New thread" for a draft. `dir` follows `textDirection(title)`: each sentence takes the direction of most of its words, the text takes the direction of most sentences (textDirection.ts:32–44). A title with no right-to-left letter is `ltr`, and one with no letters at all gets no `dir`. A sentence tie takes its first word's direction; a tie between sentences takes the word majority of the whole title, then its first word.
- Parent: shown for subagent children only (`lineage.relationshipToParent === "subagent"`). Title from the parent's shell, else "Parent thread".
- Parent collapse (ChatHeader.tsx:120–158): `needed = Σ(width of each li's first child) + Σ(hidden overflow of crumb text without a max-width) − ellipsis width + gap × (items − 1)`. Collapse when `needed > ol.clientWidth`. It re-measures on `ol` resize, on the next frame, after font loads, and when the project name, title, parent, interface font or rename state changes.
- Linked origin: the shell's `linkOrigin.label`.

### 5.2 Title, rename and the header menu

- Click opens the thread actions menu at (title button left, title button bottom + 4). The second click of a double-click (`event.detail > 1`) does not open it again (ChatHeader.tsx:206–210).
- Double-click starts renaming, when the client may operate the thread, no modifier is held and the click was not on the chevron. It closes the menu first. In Electron the native menu takes the second click, so only browsers see this path.
- Rename input: autofocus, selects everything on focus. Enter commits. Escape cancels. Blur commits, except when focus moves to a button, which cancels. Keys during IME composition (`isComposing`, keyCode 229) are ignored.
- Commit rule (ChatHeader.tsx:61–69, 173–198): trim. Empty gives a warning toast "Thread title cannot be empty". Unchanged does nothing. Otherwise `thread.updateMetadata` with the new title; a failure gives an error toast "Failed to rename thread" with the error message. Renaming ends by itself when the thread changes or the operate scope goes away.
- "Rename thread" in the menu runs the same rename.
- Right-click anywhere in the header content, unless renaming: on a server thread the same menu opens at the pointer; on a draft with a project a one-item menu "Project settings" (icon `settings`) opens there; otherwise the browser's own menu shows.
- Project crumb click starts a new thread in that project. Parent crumb click navigates to the parent.
- Menu keyboard (contextMenuFallback.ts:284–343): ArrowDown and ArrowUp move with wrap-around over enabled items, Home and End jump, ArrowRight opens a submenu, ArrowLeft closes one, Enter and Space activate, Tab closes the menu (focus continues from the element that opened it). Escape acts on the level that holds focus: in a submenu it closes that submenu and focuses its parent item; at level 0 it closes the whole menu. A submenu opened by hover leaves focus on its parent item in level 0, so one Escape then closes everything (measured). Initial focus: the first enabled item when the element focused before opening matched `:focus-visible` and is not a text field, else the menu element (contextMenuFallback.ts:611–618). Measured: a click on the title while it still showed its keyboard ring started on the first item; a right-click started on the menu. Pointer-down or right-click outside closes it once a frame has passed; focus leaving or the window blurring closes it too, without restoring focus. Closing returns focus to where it was if focus was inside the menu. Measured: Escape returned focus to the title button.
- Menu pointer (contextMenuFallback.ts:506–595): hovering an enabled item focuses and highlights it. Hovering a parent item opens its submenu at once; clicking it opens the submenu and focuses the submenu's first enabled item. Hovering a leaf item, or the pointer entering a menu, closes the levels below it. A click on a leaf counts only once a frame has passed since the menu opened, so the opening gesture cannot pick an item; Enter and Space always count. Mouse back and forward buttons are swallowed inside the menu.

### 5.3 Panel toggles and shortcuts

- Details toggle, inline mode: toggles the per-thread inline flag. Opening sends focus into the card (`requestWorkspaceFocus({ area: "details" })`, measured: the first tabbable row in a fresh card, the last focused control once one has had focus). Closing while focus is inside the card moves focus to the content area (useRightPanelActions.ts:222–237).
- Details toggle, popover mode: the shared popover handle opens and closes the popover. Opening runs `requestWorkspaceFocus({ area: "details" })` (ThreadDetailsCard.tsx:177–180): the card's last focused control, else its first tabbable one (measured: it returned to the branch row focused earlier). Escape, an outside press or the toggle closes it; a press inside the find bar does not (ThreadDetailsCard.tsx:135–147). Measured: closing with Escape put focus back on the header's details toggle, both when the popover was opened by clicking the toggle and when it was opened with mod+Shift+B from an unfocused page. An outside press leaves focus where the press put it. When the focused row's tooltip is open, the first Escape only closes the tooltip.
- If the card switches to inline while the popover flag is set, the flag is cleared so the popover does not reopen later (ThreadDetailsCard.tsx:66–69).
- Persistence: the inline flag is stored per thread only while hidden (`t3code:right-panel-state:v2`, `threadPanelVisibilityByThreadKey`, measured `{inlineOpen: false, popoverOpen: false}`; showing the card again removes the entry). The popover flag is never stored. Default: inline open.
- Terminal toggle opens or closes the drawer (03-shell.md §5.6). The shortcut does nothing for a client that can neither read nor write terminals.
- Right panel toggle: opening restores the panel and focuses its active terminal if any. Closing turns an active browser surface into the mini player, floats an active device surface, un-maximizes and closes (useRightPanelActions.ts:164–221).
- Maximize toggles the maximized flag and moves focus into the panel when maximizing.
- Default shortcuts (packages/shared/src/keybindings.ts:57–133): mod+Shift+B `threadPanel.toggle` (not in a terminal), mod+J `terminal.toggle`, mod+Alt+B `rightPanel.toggle`, `rightPanel.toggleMaximized` (no default), mod+T `rightPanel.new` (panel open, not in a terminal: focuses "+" and opens its menu), mod+W `rightPanel.close` (closes the active tab), mod+O `editor.openFavorite`, mod+D `diff.toggle` (not in a terminal), mod+F `chat.find` (the title menu's "Find in thread"; not in a terminal or a preview). The card's run-context row carries `data-composer-shortcut="composer.host composer.workspace"`, but the composer looks for those targets only inside its own shell, so mod+Shift+H and mod+Shift+X never reach the card.

### 5.4 Card mode and density

- Mode: inline when the canvas gives a placement (threadDetailsCardLayout.ts:17–47), else popover (ThreadDetailsCard.tsx:40–42).
- Density (threadDetailsCardLayout.ts:8–15): `full` when the full content height is unknown (0) or fits; else `compact` when the compact height is unknown or fits; else `essential`. Heights are measured per density as it renders, keyed by environment, thread, card width (or "popup") and the visible item set. Scrollable lists (lineage, side chats, the pull request list, automations, server overflow rows) do not count, so a long list scrolls instead of dropping sections.
- The available height is the slot height inline, and canvas height − 52 in the popover.
- Mode inputs come from the canvas measurements (ChatCanvas.tsx:33–69, canvasStore.ts:82–157): canvas client size, the lane probe's padding (48 from 640 up, 12 below) and `min-width` (640), the find bar's bottom edge, and the preview mini player's frame. The probe and the canvas are watched by a ResizeObserver, so the mode follows a resize after the observer fires, not synchronously.
- In full density the card reports its full content height to the canvas, which keeps the preview mini player clear of it.

### 5.5 Which rows show

ThreadDetailsPanel.tsx:106–179:

- `offersGit` = the thread has a Git cwd (worktree path or project root), the server config and the project are loaded, and the project is not the scratch project.
- Environment row: full density and item on.
- Open in editor: density not essential, item on, and `showOpenInPicker` (a project exists, and the thread is on the primary or host environment, or remote opening is possible for it; OpenInPicker.logic.ts:5–20).
- Scripts: item on and the thread has a project.
- Branch row: `offersGit`, a Git repository, item "Branch and pull request" on.
- Pull request rows without a branch row: not (`offersGit` and Git repository), item "Branch and pull request" on.
- Git actions and Changes: `offersGit`, a project name, and (item "Git actions" on, or item "Changes" on in a Git repository). Each of the two rows also has its own item.
- Servers, Automations, Lineage, Side chats: full density, a server thread, item on, and something to show.
- Usage limits: item on. Context window: item on and a context window known.
- A Git repository means the live status's `isRepo`, else the value remembered this session for that cwd, else true.
- Items (threadDetailsItems.ts:4–17) default on, except Usage limits and Context window, which follow the retired `threadDetailsUsageEnabled` flag. Client settings store only the choices that differ from the default (`threadDetailsItems`). "Reset to default" clears them and the flag.

### 5.6 Run context row

- Label: the auto-balance label while auto-balancing. Otherwise it depends on the environment indicator (`shouldShowEnvironmentIndicator`: the project has more than one machine, or the thread's machine is not the primary one; BranchToolbar.logic.ts:58–64). With the indicator the row leads with the machine icon and shows the machine's label ("Run on" when the machine is not listed), with the workspace icon trailing; without it the row leads with the workspace icon and shows the workspace label (BranchToolbar.tsx:249–259). Workspace label: "New worktree" when every model gets its own worktree; when the mode is locked, the worktree folder's name, else "New worktree" or "Local checkout"; in worktree mode "New worktree"; else "Current worktree" with a worktree path, or "Current checkout" (BranchToolbar.tsx:196–202, BranchToolbar.logic.ts:94–114).
- Environment label: a primary environment with a generic name ("local", "local environment") reads "This device" (BranchToolbar.logic.ts:38–56).
- Locked when the environment is locked (the thread has messages or a runtime), or when the mode is locked (a server thread with a worktree) and no environment picker applies.
- The menu changes the environment (unsent drafts only), the workspace mode (Current checkout, New worktree, Previous worktree; drafts and empty server threads) and opens "Add to another machine". Picking "Previous worktree" seeds the draft with the newest other worktree of the project that is not archived.
- Right-click: "Copy full path" copies the workspace path and toasts "Path copied" with the path, or "Failed to copy path". A ctrl-click or a non-primary button does not open the menu.

### 5.7 Open in editor

- Editors: on the host's machine, the editors the server found (`availableEditors`); over a remote link, the editors this computer can open by URL (probed by the desktop app; a browser assumes VS Code, phones and tablets get none). They are shown in a fixed order: Cursor, Trae, Kiro, VS Code, VS Code Insiders, VSCodium, Zed, Antigravity, IntelliJ IDEA, Aqua, CLion, DataGrip, DataSpell, GoLand, PhpStorm, PyCharm, Rider, RubyMine, RustRover, WebStorm, then the file manager ("Finder", "File Explorer", "Files").
- Preferred editor: the last one picked (`localStorage` `t3code:last-editor`) when available, else the first available in the contracts' `EDITORS` order.
- Primary click opens the thread's Git cwd in the preferred editor (`shell.openInEditor`, or the editor's remote URL), then announces "Opened in <Editor>" to screen readers.
- Picking an editor in the menu only makes it the preferred one. It does not open anything.
- mod+O clicks the last `[data-open-favorite-editor]` button on the page (capture phase, so a focused terminal does not swallow it). Without one, a window-level fallback opens the editor directly. It is off while the right panel shows a file, whose viewer has its own Open in.

### 5.8 Project scripts

- Scripts: the project's scripts after settings overrides (packages/shared/src/projectScripts.ts:11–21).
- Primary: the script last run in this project; else the first that is neither a setup nor a settle script; else the first that is not a settle script (projectScripts.ts:105–108).
- Primary click runs that script (the server runs it in a terminal). Menu rows run theirs. The gear edits one ("Edit Action" dialog). "Add project script" opens the "Add Action" dialog (02-primitives.md §3.21 measured it). t3.json rows import a script that the project lacks by command and by name; a failed import reopens the dialog with the error.
- Row labels add roles: "<name> (setup)", "<name> (on settle)", "<name> (setup, on settle)".

### 5.9 Branch row

- Value: in worktree mode without a worktree, the thread's branch, else the current Git branch; otherwise the current Git branch, else the thread's branch (BranchToolbar.logic.ts:189–200).
- Label: "Select ref" when there is none; when choosing a new worktree's base, "From <branch>", plus " (remote)" when starting from origin with a local branch; otherwise the branch.
- Opening refreshes the local status (`vcs.refreshStatus`, `localOnly`). Closing clears the query.
- Items: a "Checkout pull request" item first when the query parses as a pull request reference (drafts only, two lines: "Checkout pull request" and the reference); the refs, paged (load more at half a page from the end); "Create new ref "<name>"" last when the query has no exact match (not while choosing a worktree base). Matching is a case-insensitive substring, also against the query sanitised into a ref name.
- Picking a ref: as a worktree base it only sets the thread's branch. A ref checked out in another worktree switches the thread to that worktree. Otherwise `vcs.switchRef` runs in the right checkout with the label updated optimistically; failure restores it and toasts "Failed to switch ref.". Changing worktree while a turn runs is refused with a warning toast "Stop the current turn to switch worktrees.". Creating runs `vcs.createRef` with a switch ("Failed to create and switch ref." on failure). Picking moves focus to the composer.
- Enter in the search box picks the highlighted item. Right-click on the row or an item offers "Copy branch name" ("Branch name copied").
- A new worktree thread with no branch gets the default branch, or the current one, once refs load.

### 5.10 Pull request rows

- Which pull request (ThreadStatusIndicators.tsx:121–163, 492–512): the thread's current linked pull request when the server supports several; else the legacy linked one; else the branch's pull request from Git status, only when the status is on the thread's branch and the pull request is open.
- Detail refreshes every 10 minutes while open; checks refresh while open until settled.
- Click and middle-click open it (lib/openPullRequestLink.ts:243–270): in the app when "open pull request links in app" is on and no modifier is held, else in the browser. Cmd or Ctrl or the middle button forces the browser.
- "Show N more" lists the thread's other visible linked pull requests in chain order. "Show less" folds them.
- The eye control stops watching (`watching: false`). Fix and Resolve hand the work off. Under the branch row (no composer target) they check the branch out in a new thread, so their pills end with `arrow-up-right`. In the rows that stand in for the branch row (No project, non-repo folders) the card passes the composer as the target: the task text goes into this thread's composer with a success toast "Added to the composer", "The task is in the composer. Read it over, then send.", and the pill has no arrow (ThreadDetailsPanel.tsx:244–250, ThreadDetailsPrRow.tsx:200–211, 344–347).

### 5.11 Git actions

- Quick action: §4.5. A click runs it: a hint shows an info toast (title = label, description = hint); Pull runs `vcs.pull` and shows "Pulled" ("Updated <ref> from <upstream>") or "Already up to date" inline; Publish opens the publish dialog; the rest run `git.runStackedAction`.
- On the default branch, push and pull request actions first ask (GitActionsControl.logic.ts:343–380, GitActionsControl.tsx:1967–2016). Titles: "Push to default ref?", "Commit & push to default ref?", "Push & create PR from default ref?", "Commit, push & create PR from default ref?". The description ends `on "<branch>". You can continue on this ref or create a feature ref and run the same action there.` Buttons: "Abort" (outline), a continue button labelled "Push to <branch>", "Commit & push to <branch>", "Push & create PR" or "Commit, push & create PR" (outline), and "Check out feature branch & continue" (default). The last is disabled with a clean tree, which also shows "There are no changes to commit, so a feature branch can't be named. Create a branch yourself or continue on <branch>."
- The menu refreshes the status when it opens. When the menu would be empty, hovering or focusing the group refreshes it instead. Menu items (GitActionsControl.logic.ts:109–176): Commit opens the commit dialog; Push and Create PR run directly. With no primary remote only Commit is listed; with an open pull request, Commit and Push. Disabled items explain themselves: "Worktree is clean. Make changes before committing.", "Detached HEAD: check out a branch before pushing.", "Commit or stash local changes before pushing.", "Branch is behind upstream. Pull/rebase before pushing.", "Add an "origin" remote before pushing.", "No local commits to push.", and the pull request versions of these.
- Commit dialog: "Commit changes", "Review and confirm your commit. Leave the message blank to auto-generate one.", a branch line ("Default branch" in `--warning` on the default branch), the file list with per-file `+N / -N` and an Edit mode with checkboxes, an optional message ("Leave empty to auto-generate"), buttons "Cancel", "Commit on new branch", "Commit".
- While running: progress content (§3.8). After success: the result inline for 10s ("Created PR" for a new pull request). Failures toast "Action failed" or "Pull failed" with the message and no timeout.
- `quietEmptyDraft` (GitActionsControl.tsx:1194–1200): in a fresh draft, or a server thread with no runs, with no progress, no status error, a disabled quick action and zero changes, both the Git actions row and the Changes row hide. In compact density the Git actions row also hides unless something is running or just succeeded.

### 5.12 Change totals and number formats

- Totals (GitActionsControl.logic.ts:454–481): the Diff panel's branch comparison when it has file stats (sum of the files); else the status's branch changes when non-zero; else the working tree when non-zero; else the branch changes if present, else the working tree. The comparison is the same query the Diff panel makes (`review.getDiffPreview`, metadata only, branch range, the remembered base, the whitespace setting), revisioned by the status totals. It only runs while the Changes item is on, the folder is a repository and the client may read files (`useFilesystemReadAccess`); otherwise the totals come from the status alone (GitActionsControl.tsx:1083–1123).
- Base label: "Uncommitted" for working-tree totals, "vs <baseRef>" for branch totals, none before the status loads.
- Card counts: `formatDiffCount` = `new Intl.NumberFormat().format(n)`, so the locale's grouping ("2,382" in English).
- `DiffStatLabel` counts: `formatCompactDiffCount`. Unlike the card's Changes row, both spans keep the addition and deletion colours at 0. Below 1000, the plain number. Below 999,500: thousands with one decimal under 10 and a trailing ".0" dropped, else rounded ("1.2k", "12k"). The same pattern for "m" below 999,500,000 and "b" above.
- Clicking Changes opens the Diff surface in the right panel with the branch scope (useRightPanelActions.ts:129–140). Drafts and folders without Git cannot, so the row is disabled there.

### 5.13 Right panel tabs

- Order is the panel's surface list. Dragging reorders: 6px activation distance, x axis only, clamped to the tab list (RightPanelTabs.tsx:828–863).
- Click activates (a terminal surface also gets focus; a browser surface becomes the active preview tab). Middle-click closes. The tab cancels the middle button's mouse-down and mouse-up everywhere, because Chromium on Linux only skips the primary-selection paste when the mouse-up is cancelled. The close button closes.
- Closing the active tab activates the one now at its index, or the last. Closing the last surface closes the panel (rightPanelStore.ts:972–991). Measured: closing the active second of two tabs activated the first; closing that one closed the panel, the inline details card came back, and focus landed on the right panel toggle.
- Right-click (native menu in Electron, in-page menu in a browser): Rename (device tabs), Copy path (file tabs), Mute tab or Unmute tab (browser tabs, disabled until the desktop state arrives), Close, Close others (disabled with one tab), Close to the right (disabled on the last tab), Close all.
- A vertical wheel scrolls the tab list sideways (line deltas × 16, page deltas × list width) and is consumed only if the list moved. Horizontal input scrolls natively, which keeps swipe navigation working.
- The scroll buttons move by max(120, 0.75 × list width), smooth unless reduced motion. The active tab scrolls into view when it changes and when the tabs start to overflow.
- Add menu: while it is open, a bare letter key runs that row (B, T, F, D, P, L, M, S; measured: F added Files, D added Diff, each closing the menu). mod+T opens it. Whenever the browser is available, Browser is a submenu of profiles; any click on the row that is not a touch (mouse, pen, keyboard activation) opens the default profile, a touch opens the submenu (RightPanelTabs.tsx:172–174, 1431–1463). Unavailable add-menu rows carry their own, longer reasons (RightPanelTabs.tsx:176–185): "Browser previews are only available in the T3 Code desktop app.", "Terminal surfaces are only available from a project thread.", "Files are only available when a project is open.", "Diff is only available for server threads in Git repositories.", "This thread's branch has no pull request yet.", "No linked pull requests are available for this thread.", "Devices are only available from a thread.", "Side chats start after the first message."
- Launcher: the same letters work anywhere on the page (capture phase) unless the target is a text field or an open popup, menu, dialog, select, combobox or autocomplete would take them. Arrow keys move a highlight with wrap-around; Enter runs it when the launcher itself has focus; hovering highlights. Unavailable rows explain why in a tooltip (RightPanelTabs.tsx:201–210: "Only available in the desktop app.", "Available when a project is open.", "Available for Git repositories.", "No pull request on this branch yet.", "No linked pull requests available.", "Available from a thread.", "Available after the first message.").
- Device tabs rename on double-click: Enter or blur commits, Escape restores.

### 5.14 Focus order

- Tab order in the header: project crumb, parent crumb, title, guest "Stop sharing". The draft title is not focusable.
- The card is F6 area `details` (order 3, 03-shell.md §5.7). Entering it focuses its last focused control, else its first tabbable one. Inside, Tab follows the rows top to bottom; a split group is two stops, a select row one.
- Menus and popovers from the card return focus to their trigger on Escape (measured on the branch picker, with the focus-visible ring).

## 6. Motion

Ease-in-out is Tailwind's `cubic-bezier(0.4, 0, 0.2, 1)`, ease-out `cubic-bezier(0, 0, 0.2, 1)`.

| What | Property | Duration | Easing | Source | M/S |
|---|---|---|---|---|---|
| project and parent crumb | `color` (and the other `transition-colors` properties) | 150ms | ease-in-out | ChatHeader.tsx:270, 297 | M |
| title chevron | `opacity` 0 ↔ 1 | 150ms | ease-in-out | ChatHeader.tsx:363 | M |
| header left padding | `padding-left` | panel animation duration | ease-out | 03-shell.md §6 | S |
| header and toggle tooltips | 02-primitives.md §6 | | | | S |
| thread actions menu (browser) | none: it appears at once and is clamped one frame later | | | contextMenuFallback.ts:599–603 | S |
| card rows | `box-shadow`, `scale` (focus ring, 0.97 press) | 150ms | ease-in-out | ThreadDetailsControl.tsx:53 | M |
| inline card moving for the find bar | `top`, only on the way down: the class exists only while the inset is non-zero, so closing the bar snaps the card back to 12 | 150ms | ease-out; none with reduced motion | ThreadDetailsCard.tsx:153–157 | M |
| inline card show and hide | none, it mounts and unmounts | | | | M |
| popover positioner | `top` | 150ms | ease-out | popover.tsx:85 | M |
| popover popup | `width`, `height`, `scale`, `opacity`; opens from and closes to scale 0.98, opacity 0 | 150ms | ease-in-out | popover.tsx:95 | M |
| card scrollbar | `opacity` 0 → 1 while hovered or scrolling, back to 0 after a 300ms delay | 100ms in | ease-in-out | scroll-area.tsx:87 | S |
| "Previous agents" chevron | `transform`, rotate −90° (collapsed) ↔ 0 | 150ms | ease-in-out | ThreadRelationshipsControl.tsx:180–187 | S |
| lineage and server row tooltips | open delay 200ms instead of the usual 600ms | | | ThreadRelationshipsControl.tsx:820, ThreadServersPanel.tsx:126 | S |
| reason popovers (disabled quick action, disabled Git menu rows) | open on hover after Base UI's 300ms trigger delay, then the popup's 150ms scale and opacity | | | GitActionsControl.tsx:1515–1529, 1622–1652 | S |
| density change | none | | | | M |
| Git progress output and success description | `grid-template-rows` 0fr ↔ 1fr, `opacity` | 200ms | ease-out; none with reduced motion | GitActionsControl.tsx:375, 416 | S |
| automation switch | 02-primitives.md §6 (Switch) | | | | S |
| lineage row arrow | `opacity` 0 → 1 on row hover | 150ms | ease-in-out | ThreadRelationshipsControl.tsx:796 | M |
| scripts menu: shortcut out, gear in | `opacity` | 150ms | ease-in-out | ProjectScriptsControl.tsx:200–205 | S |
| right panel tabs | none on fill or colour (hover is instant) | | | RightPanelTabs.tsx:1019 | M |
| tab drag | dnd-kit `transform`, sortable default transition | 250ms | ease | dnd-kit default | S |
| tab list scroll buttons | smooth scroll, instant with reduced motion | browser | | RightPanelTabs.tsx:1089–1097 | S |
| launcher rows | `transition-colors` | 150ms | ease-in-out | RightPanelTabs.tsx:506 | M |
| maximize control fade-in | 03-shell.md §6 | | | | S |

## 7. Gaps

- **Read-only pairing.** Every enabled source-control state, editor opens, rename and automation toggles were not reachable. Script runs were reachable but not exercised, because a click starts a terminal command. Their looks are the same class lists on enabled elements (P) or source (S). Hover on an enabled quick action uses the split-button fill measured on enabled splits.
- **Git states not present live.** Ahead, behind, diverged, no upstream, no primary remote, detached HEAD, a running action, the success line and the default-branch dialog come from source. Clean, dirty on the default branch, dirty on a feature branch, non-repo and loading were measured.
- **Pull request states.** Open (with Merge), open with conflicts (with Resolve) and merged were measured. Draft, closed, failing (Fix), pending and watching are source with probed colours.
- **Rename input** could not be opened (no operate scope); it was measured as a probe in the header.
- **Parent crumb, "From <origin>" badge, guest share indicator**: no live thread had them.
- **Open in editor, enabled**: the measured server gave this client no usable editor, so only the disabled split and its fallback icon were seen.
- **Run on group, Add to another machine, auto-balance**: single machine live.
- **Servers, Side chats, Usage limits, Context window** sections were not present (no discovered servers, no side chats, items off by default). Lineage logic (graph ordering, merge back, workflows) is out of scope; only its row look is recorded.
- **Electron**: the thread actions menu is the native OS menu there; drag regions and title bar insets are 03-shell.md's.
- **Hover in screenshots.** agent-browser's own screenshot drops a synthetic hover. The verification pass captured hovered states through CDP `Page.captureScreenshot` instead (1× pixels), so light and dark hover fills are now both read from pixels.
- **Hidden windows.** A browser window covered by another reports `visibilityState: hidden`; timers throttle and frames stop, so Base UI tooltips and popovers stall at their starting style (scale 0.98, opacity 0) and colour-scheme switches wait. Boxes read in that state are off by a fraction of a pixel. The verification pass moved its own window partly on screen before measuring.
- **Backdrop blur.** Card and menu fills were sampled over a plain canvas, where the blur changes nothing. Over busy content T3 shows a frosted blur; the painted colours in this document are the flat case.
- **SF Pro.** Widths and the crumb's 22.19 box use Inter metrics.

## 8. Port notes for Go/Gio

1. **What this replaces in tern.** `threadHeader` and `compactHeader` in `app/ui/screens.go` become the T3 header: breadcrumb only, no state chip, no Stop button, no "Project › branch" line. T3 has no separate compact header; below 768px it keeps the same row with the 52/56px left inset. `details()` and `detailsRow()` in `app/ui/details.go` become the details card: drop the status, model, elapsed and mode rows and the click-to-copy rows, which T3's card does not have (T3 copies through right-click menus). `rightPanel`, `panelHeader` and `panelTabs` in `app/ui/screens.go` become the tab bar of §2.8 and the launcher of §3.12. `titlebarControls` in `app/ui/shell.go` keeps its geometry but takes its tooltip labels from the keybindings, gains the popover-trigger form and moves the details toggle into the header while the inline panel is open (§2.1). `popIn` in `app/ui/details.go` should start from scale 0.98, not 0.95.
2. **Header layout.** Inside the 52dp header: a row with gap 12 (8 below 640) and an end reserve of 96 (40 with the inline panel). The `ol` is a row with gap 12: the project crumb (14dp icon, gap 6, text capped at 160 with an ellipsis), "/", then the title item, which grows from 0 with a 40dp minimum. Give the title all spare width first; shrink the project crumb only after the title reaches 40. Vertically centre each crumb's cap band at y 26: baseline = 26 + capHeight / 2, capHeight from the font's OS/2 table (Inter 0.727 em, SF Pro about 0.705 em).
3. **Title control.** A clickable region over text + 4 + 14dp chevron. Draw the chevron only while hovered or keyboard-focused, fading over 150ms. Single click opens the menu at (left, bottom + 4); ignore the second click of a double click; double click (operate scope, no modifiers) starts the rename editor: a 20dp single-line editor with a 1dp inset ring in `--ring` and the commit rules of §5.2. Tooltips sit below the header, centred, 4dp gap.
4. **Thread actions menu.** Gio has no native menu, so draw the browser fallback of §3.3. Fill: composite `#ffffffd5` / `#111111d5` over what is behind (Gio has no backdrop blur; over the plain canvas this paints `#ffffff` / `#0e0e0e`). Border 1dp `--contrast-foreground` 10%, radius 8, no shadow, padding 4, 28dp rows with 14dp text at every window width, 16dp icons (18dp below 640) drawn at stroke 2, separators 1dp inset 8. Hover focuses the item; hovering a parent opens its submenu at once; ignore leaf clicks in the first frame after opening. Clamp 4dp inside the window. Implement the keyboard map of §5.2. The item list and the read-only disabling come from threadActionMenu.logic.ts.
5. **Card frame.** Inline: 280 wide at (canvas.w − 292, 12 + find inset, where the inset is the find bar's bottom edge, 48, while it is open; animate only the move down), max height canvas.h − top − 12 (or the mini player's top − top − 12 when it overlaps the card column), radius 16, 1dp border, no shadow, the glass fill. Popover: same card, 8dp inside a 296-wide box whose right edge is header.right − 4 and top is header.bottom. Use 03-shell.md §5.4 to choose. Clip content to the card and scroll it, with 24dp edge fades on the overflowing side.
6. **Density.** Lay out the card's content at full density off screen (or measure as you build), subtract the heights of the scrollable lists, and choose full, compact or essential with §5.4. Cache by (thread, width, visible items). Drop rows per §4.3.
7. **Rows.** One control type with parts row, select, primary, secondary, icon and meta (§2.6 and §3.6). Icon column at x + 11, label at x + 37. On touch input give every control a hit area of at least 44×44dp around its box. Hover: fill black 3.5% (white 8% dark), and for select and split controls a second fill on the hovered half plus the 1×16 hairline in `--border` 65%. Press scales 0.97 around the centre over 150ms, except controls that open menus. Focus: 1dp `--background` then 2dp `--ring`, inset, radius 8. Disabled: 64% opacity.
8. **Labels.** 13dp, weight 500, `--foreground` at 80%; cap-band centred like the header. Branch labels use the MiddleTruncate algorithm of 02-primitives.md §5.14. The Changes counts use the mono face at 13dp with tabular figures and locale grouping.
9. **Data tern's core must provide.** Today `core/scene.go` fills `RunInfo{Machine, Mode, Branch, Worktree}` and a "Project › branch" string. The card needs:
   - Thread shell: title, branch, worktree path, environment and project ids, lineage (parent id, relationship, side chat), link origin label, linked pull requests (number, URL, title, state, draft, head and base, sync time, watch), the legacy linked and branch pull requests, latest run (or none), runtime (running or not), message count, guest share.
   - Project: title, scratch flag (from the server's `scratchWorkspaceRoot`), workspace root, icon or favicon, scripts after settings overrides, repository provider.
   - Environments for the project: label, machine kind, primary flag; the primary and host environment ids; the remote-open mode.
   - Server config: `availableEditors`, keybindings, capabilities (`pullRequestChecks`, `localServerStop`, `threadFind`), settings (`newWorktreesStartFromOrigin`, offered source control providers), usage limit sources.
   - Git: a `subscribeVcsStatus` stream per Git cwd (snapshot, `localUpdated`, `remoteUpdated` → `VcsStatusResult`: `isRepo`, provider, `hasPrimaryRemote`, `isDefaultRef`, `refName`, `hasWorkingTreeChanges`, working tree files and totals, `branchChanges`, `hasUpstream`, `aheadCount`, `behindCount`, `aheadOfDefaultCount`, `pr`); `vcs.listRefs` with query and paging; `vcs.refreshStatus`, `vcs.switchRef`, `vcs.createRef`, `vcs.init`, `vcs.pull`; `git.runStackedAction` with its progress (operation, current label, last output line, start times).
   - `review.getDiffPreview` (metadata only, branch range) for the Changes totals.
   - `pullRequests.detail` and `pullRequests.checks`, and the merge, ready and watch actions.
   - `scheduledTasks.subscribe` (plus `setEnabled`, `runNow`); `subscribeDiscoveredLocalServers` (tern already streams it); `shell.openInEditor`.
   - Client settings: `threadDetailsItems`, `threadDetailsUsageEnabled`, `pullRequestLinksOpenInApp`, `diffIgnoreWhitespace`. Local state: last editor, last-run script per project, the per-thread hidden-card flag, right panel surfaces.
   - Session scopes per environment: orchestration operate (rename, editor opens on the host, automations, branch metadata), source-control write (Git rows), terminal write (script runs), filesystem read (the Changes comparison), settings write (script keybindings).
   - Lineage: running subagent count, per-row status, elapsed time, stop support, the "Previous agents" group, merge-back availability and detach support.
10. **Right panel tab bar.** 52dp row, no bottom rule, padding 8 left and 112 right inline (12 right in the sheet), gap 4. Tabs 24dp pills (radius 6, padding 6/8, gap 2, 12dp text, max 144) with `--accent` when active and 60% `--accent` on hover. The 16dp close slot shows the 12dp surface icon and swaps to `x` while the tab is hovered. "+" is a 24dp ghost-muted button opening the add menu (offset 6). Horizontal scrolling with hidden scrollbars, 24dp edge fades, wheel-to-horizontal, scroll buttons on overflow, drag-to-reorder after 6dp. Closing the last tab closes the panel.
11. **Launcher.** Centred 320dp column, heading 14dp 500, 32dp rows with 16dp icons and a 20dp letter key cap, letter shortcuts that work page-wide while the launcher is shown, arrow-key highlight.
12. **Tests worth writing.** The quick action table (§4.5) row by row; `resolveChangesTotals`; `formatCompactDiffCount` boundaries (999, 1000, 9950, 999499, 999500); density selection; the inline/popover edges (canvas 1012 vs 1011 wide, 184 vs 183 tall, plus 48 with the find bar); the show rules of §5.5 for each thread kind of §4.4; the workspace icon and label rules (§3.7, §5.6) for drafts and server threads with and without a worktree; the breadcrumb collapse rule; tooltip flip below the header; tab close fallback order; Escape in the thread menu with a hover-opened submenu.

## 9. Verification

I re-checked this document against the same build and assumed it had errors. Browser: 1440×900 @2x Linux Chrome in both themes. I also measured at 520 wide, at the 1267/1268 and 235/236 edges of the card's inline rule, and with the right panel open. Pages: a draft, server threads in Git repositories, a No project thread, and threads with automations and running subagents. One repository thread sat on the default branch. Another lived in a worktree with an open pull request, which picked up merge conflicts during the pass. I opened and closed menus, popovers, the branch picker, the run-context menu, the find bar and the right panel. I added Files and Diff surfaces and closed them again, and I hid and re-showed the inline card. The stored right panel state ended identical to the snapshot I took first, and I left the theme on light. I sent, renamed, ran, switched and committed nothing. Hovered pixels come from CDP captures, because agent-browser's screenshot drops a synthetic hover. Before measuring I moved my browser window partly on screen, because a covered window stalls Base UI's transitions (§7).

### What was checked

Re-measured live, about 130 values, in light and dark unless noted:

- Header. Box, padding, background and the missing bottom border. ChatHeader box, gap and end reserve. The `nav` clip, the `ol` gap and font. The project crumb's width, gap, radius, transition and 14px icon, seen as a favicon image, as a monogram tile with its 14% fill and 8.25px bold mono letters, and as the No project `message-square-dashed` in both themes. Separator width, weight and colour. Title button gap and width rule. Chevron size, position, opacity and transition. Crumb, title and chevron colours, crumb hover, the chevron on hover and on keyboard focus, and the title focus ring in both themes. The below-640 padding at 520 wide.
- Header tooltips. The title and project crumb boxes, and all three panel toggle tooltips with their side and 5px window margin. The toggle's pressed fill in both themes, the fixed group, the hidden maximize slot, and the header-owned toggle at 860, 0 with the panel open.
- Thread actions menu. Position from a click and from a right-click, size, and the surface, border, blur and missing shadow in both themes. Item box, text, icon size and stroke 2. Highlight, disabled and separator colours in both themes, separator positions, submenu placement and size, Escape and focus return, rows at 520 wide, and the draft's one-item menu.
- Card. Aside box, max height and attributes. Fill, border and blur in both themes, plus painted fill, border and divider pixels in both themes. Section padding and headings. Row box, font and colours, and the locked run-context colour, which is 80% foreground in light and muted in dark. Select halves and chevron, the label cap box, and the 14px icon offsets. Hover fills in both themes as computed values and as pixels, split halves and hairline pixels, the dark focus ring, and disabled opacity.
- Version Control. The branch trigger and picker, covering frame, search, list, item, highlight, tags, status line and focus return on Escape. The PR row: open button, title, meta line, number, checks trigger, the Merge pill in both themes, the Resolve pill and its hover in both themes, and the Resolve tooltip. Then "Show N more", the Git actions split and its reason popover, and the Changes label, base label, mono counts and colours in both themes.
- Lower sections. The Automations heading, task row, status dot, icon controls and switch. The Lineage heading, rows, status dots, elapsed times, stop controls and the "Previous agents" row.
- Card modes. Popover box, padding, card max height, z-index and transitions. The toggle's pressed and expanded states. Focus on open, and on Escape after opening by click and by mod+Shift+B. Inline toggle persistence, the find bar offset and transition, the inline and popover edges by width and height, essential density at 236 tall, and the card's own menu.
- Right panel. Tab bar box and padding. Launcher box, padding, heading, rows, Kbd and hover. The add button and add menu, the F and D letter keys, tabs at rest and hovered in dark and active in light, close button hover, the tab tooltip, closing order, and focus after the last tab closes.

Re-derived from source, about 60 rules: rename commit and blur rules; the header context menu; control placements and the 980px sheet query; toggle labels and tooltips; card layout thresholds, density and the find bar inset; the show rules in ThreadDetailsPanel; section, style and control class strings; all 18 rows of `resolveQuickAction`, `buildMenuItems` and the menu's reason strings; `quietEmptyDraft`; `resolveChangesTotals` and its query gate; both count formats; quick action icons; progress, success and default-branch copy; workspace icon, label and environment indicator rules; OpenInPicker visibility, preference and disabled rules; the scripts control and its run guard; PR row actions, tooltips, watch control and labels; PR state presentation; PullRequestContextDetails; DiffStatLabel; the fallback menu's inline styles, focus, keyboard and pointer handling; the title menu's item order; default keybindings; details items; the automations, lineage and servers panels; right panel tabs, sensors, add menu and both reason tables; tooltip, popover and ScrollArea defaults; `textDirection`; the double-click guard; the top fade and its spacer; BranchPicker; machine icon kinds; the right panel action handlers.

Privacy: I searched the document for the project, thread, branch, machine and account names, paths, URLs and keybinding overrides I saw during the pass. I found none and added none.

### What was wrong, and the fix

1. §2.3, §3.3, port note 4. Menu rows were "28 (32 below 640)". Inline styles beat the `min-h-8 text-base` classes, so rows stay 28 tall with 14px text at every width. Only the icons and the submenu chevron grow, to 18 below 640. Measured at 520 wide.
2. §3.7. The workspace icon rule gave `folder-git` for "the thread has a worktree". The icon is `folder-git-2` whenever the effective mode is worktree, and that covers every server thread living in a worktree. `folder-git` only appears on a draft attached to an existing worktree (BranchToolbar.tsx:190–195, BranchToolbar.logic.ts:158–172). Measured on a worktree thread, a No project thread and a draft.
3. §3.7, §5.6. The run-context label rule ignored the environment indicator. With the indicator the row shows the machine icon and label plus a trailing workspace icon. Without it the row shows the workspace icon and label, with no trailing icon.
4. Conventions, §3.7, §4.4. Script runs were said to render `aria-disabled` for a client without the run permission. The card always passes a run handler, so the script split is never disabled, and the permission check sits inside the handler (useChatTerminalActions.ts:212). Measured: an enabled split.
5. §5.3. "After closing, focus landed in the composer." Escape put focus back on the header's details toggle, whether the popover was opened from the toggle or with the shortcut.
6. §4.3. Popover mode was given for "canvas < 160 tall". The rule is slot height < 160, which means a canvas under 184 tall, or under 232 with the find bar. Measured at 236, inline in essential density, and at 235, popover. A preview mini player that overlaps the card's column can also cut the slot below 160.
7. §2.5, §3.5. The find bar inset was a constant +48 from `THREAD_FIND_BAR_RESERVED_HEIGHT`. It is the find bar's measured bottom edge, which is 48 because the bar sits at 12 and is 36 tall. The constant only switches the measurement on. Measured: card at canvas y 60, max height 776.
8. §3.6. Dark hover was "about `#212121`" and the dark hovered split half "about `#333333`". The pixels are `#202020` and `#313131`, with the other half at `#202020`. I added the hairline's painted values, `#eaeaec` and `#2c2c2c`.
9. §3.2. The project crumb tooltip top was 40.92, a reading taken mid-transition at scale 0.98. Settled, it is 41.
10. §5.2. Escape "closes the innermost level". It acts on the level that holds focus. A hover-opened submenu leaves focus on its parent, so one Escape closes the whole menu. Measured.
11. §5.13, §3.11. The add menu's Browser row was a submenu only "with several browser profiles". It is a submenu whenever the browser is available, and any click that is not a touch opens the default profile.
12. §3.12. In the launcher, `pr-7` was put on the Browser row. It sits on the row's label span.
13. §6. The find bar move read as animated both ways. Only the move down animates, and the card snaps back. The popover popup transition is now measured. Corrected citations: popover.tsx:85 and :95, RightPanelTabs.tsx:506, ThreadDetailsCard.tsx:153–157.
14. §3.10. The Lineage heading was given as "Lineage". It reads "Lineage · N running" while subagents run, and it hides when the only content is the "Previous agents" row.

### What was missing, now added

- Control classes. `will-change: scale`, the 44×44 coarse-pointer hit area, the full tone class lists, and the fact that a disabled multiline control is not faded. See §2.6 and port note 7.
- ScrollArea. The viewport's own focus ring, the overlay scrollbar and its timing, and `data-workspace-order=3` on the card. See §2.5, §3.5 and §6.
- Menu. A submenu's `aria-label`, hover focusing, hover-opened submenus, the one-frame guard on leaf clicks, swallowed back and forward buttons, and the exact initial-focus test. See §2.3 and §5.2.
- Card placement. The preview mini player overlap rule, the measured width and height edges, and the ResizeObserver lag. See §3.5 and §5.4.
- Lineage. The "More thread actions" menu with "Disconnect agent session", the related, active and "Previous agents" groups, paging of 6 rows then 12 more per click, elapsed times, the Stop subagent control, merge-back splits, the 200ms tooltip delay, hover cards, and "Side chats · N running". See §3.10.
- Automations and Servers. Tooltips, switch labels, the schedule line format, the error line, toasts, disabled rules and the full-foreground icon colour for Automations. The line join, tooltips and stop conditions for Servers. See §3.10.
- PR rows. The URL `aria-label`, the plain tooltip on simple rows, the watch tooltip and label, state labels, and the measured Resolve state. See §3.8 and §4.6.
- Git actions. The quick action's tooltip, and Initialize Git surviving compact density. See §3.8 and §4.3.
- Branch picker. The native `title` tooltip, the popup frame, "No refs found.", and the start-from-remote label and tooltip. See §3.8.
- Scripts. Group labels and the Add project script tooltip. See §2.7 and §3.7.
- Right panel. The add menu's eight reason strings, the measured letter keys, focus after the last tab closes, pointer-only drag, and the launcher's key list attribute. See §3.12, §4.7 and §5.13.
- Data rules. `textDirection` ties and the no-`dir` case, the file-read gate on the Changes comparison, DiffStatLabel colouring zeros, mod+F `chat.find`, and the parent-link header that replaces the timeline spacer. See §2.10, §5.1, §5.3 and §5.12.
- Port notes. The session scopes and lineage data tern needs, and new tests for the inline edges, the workspace icon rule and menu Escape.

### Still open

The gaps in §7 remain. Enabled Git states and dialogs, draft, closed, failing, pending and watching pull requests, the rename input, the parent crumb, the origin badge and guest share, an enabled editor menu, multiple machines, the Servers, Side chats, Usage and Context sections, Electron's native menu, blur over busy content and SF Pro metrics are still source-only or unmeasured. I did not time the reason popover's 300ms delay or the 200ms lineage tooltip delay live. I never saw the lineage hover card, a merge-back split or tab overflow live.
