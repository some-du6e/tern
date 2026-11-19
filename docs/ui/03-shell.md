# 03 · Window shell and layout

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
Token values come from [01-tokens.md](01-tokens.md). This document is the frame those tokens are painted into: the app sidebar and its rail, the regions of the chat screen, the right panel, the terminal drawer, the artwork behind the sidebar header, and the Electron window around all of it.

Conventions, as in 01-tokens.md:

- Lengths are CSS px at the default 16px root. One CSS px is one Gio dp. Anything declared in `rem` scales with the interface font size (12 to 20px). Constants written in TypeScript (sidebar 208/256/640, panel 360/540, drawer 180/280) are px and do not scale.
- Colours are sRGB hex as Chrome paints them, `#rrggbbaa` when translucent.
- **M/S column.** M is measured live: a computed style, an element box, or a pixel read from a 2x screenshot. P is a class list evaluated live on a probe element, for states the page could not show safely. S is source-derived only.
- Coordinates are viewport CSS px, origin top-left, measured at 1440×900 @2x in Linux Chrome, which renders Inter for the sans stack.
- Unless a row says otherwise, the measured state is: sidebar expanded at its default width, a thread with messages open, the thread details card shown inline (its default), right panel closed, panel animations off (the default, `panelAnimationDurationMs = 0`), chat width `comfortable`, no theme, environment artwork mode `artwork` on a nightly server, so the nightly sky art is shown.

## 1. Scope and sources

Paths are relative to the T3 repo.

| File | Lines | What it gives |
|---|---|---|
| `apps/web/src/components/AppSidebarLayout.tsx` | 69, 71–127, 175–200, 294–314, 316–408, 410–498 | macOS controls inset, viewport and fullscreen subscriptions, initial sidebar width, F6 commands, the fixed sidebar trigger and its shortcut, the layout root |
| `apps/web/src/components/ui/sidebar.tsx` | 30–34, 88–136, 143–256, 258–423, 425–450, 452–622, 637–674, 676–708 | widths, open store and persistence, provider wrapper, desktop and mobile sidebar, trigger, rail and its drag, inset, footer, content |
| `apps/web/src/components/ui/sidebarState.ts` | 1–9 | expanded/collapsed resolution |
| `apps/web/src/components/threadSidebarWidth.ts` | 1–34 | sidebar min, default, max and the 640px main-content floor |
| `apps/web/src/hooks/useResizeDrag.ts` | 1–141 | the shared pointer-drag engine (sidebar and right panel) |
| `apps/web/src/hooks/useResizableWidth.ts` | 1–118 | right panel width state, clamping, persistence |
| `apps/web/src/hooks/usePreviewPanelInlineSize.ts` | 1–74 | right panel default/min/max |
| `apps/web/src/components/preview/PreviewPanelShell.tsx` | 38–66, 68–215 | right panel host, width transition, focus return, inert |
| `apps/web/src/components/preview/RightPanelResizeHandle.tsx` | 1–33 | the right panel's resize handle |
| `apps/web/src/components/RightPanelSheet.tsx` | 1–31 | right panel as a sheet |
| `apps/web/src/components/chat/ChatRightPanel.tsx` | 19–51 | inline vs sheet switch |
| `apps/web/src/components/RightPanelTabs.tsx` | 1008–1034, 1326–1568 | right panel tab bar row, desktop title bar ownership |
| `apps/web/src/components/ui/sheet.tsx` | 18–127 | sheet backdrop, viewport, popup, slide |
| `apps/web/src/components/SidebarStageBackdrop.tsx` | 9–69, 71–212, 214–393 | stage art selection, container, nightly sky SVG, dev blueprint SVG |
| `apps/web/src/components/sidebar/SidebarChrome.tsx` | 43–121, 123–194 | sidebar header row, brand width probe, history buttons, brand |
| `apps/web/src/components/ChatView.tsx` | 712–733, 1048–1102, 1950–1972, 3251–3354, 3356–3398, 3511–3532, 3825–3828, 3938–3951, 3954–4016 | breakpoints, right panel presence, thread panel mode, header, canvas, conversation area, composer overlay, terminal drawers, right panel mount |
| `apps/web/src/components/chat/PanelLayoutControls.tsx` | 38–231 | panel toggles, maximize control, header and title bar control groups |
| `apps/web/src/workspaceTitlebar.ts` | 1–2 | collapsed-sidebar header inset class |
| `apps/web/src/rightPanelLayout.ts` | 1 | `(max-width: 980px)` |
| `apps/web/src/panelAnimations.ts` | 11–69 | animation gate, navigation suppression, exit presence |
| `apps/web/src/components/chat/ChatCanvas.tsx` | 94–117 | canvas, lane inset variables, width probe |
| `apps/web/src/components/chat/chatCanvasLayout.ts` | 21–55 | the chat lane placement rule |
| `apps/web/src/components/chat/threadDetailsCardLayout.ts` | 4–6, 17–47 | details card slot (280px, 12px gap) |
| `apps/web/src/components/chat/canvasStore.ts` | 82–142 | inline vs popover decision for the details card |
| `apps/web/src/components/chat/ThreadDetailsCard.tsx` | 173–190 | details popover anchor, side, align and offsets |
| `apps/desktop/src/app/DesktopApp.ts` | 279–281 | when the WSL connecting splash shows |
| `apps/web/src/components/chat/MessagesTimeline.tsx` | 1462–1465 | timeline scroller classes, top fade |
| `apps/web/src/components/chat/ChatHeader.tsx` | 247–259 | header content right reserve |
| `apps/web/src/components/chat/ChatTerminalDrawers.tsx` | 41–107 | drawer mounting per thread |
| `apps/web/src/components/PersistentThreadTerminals.tsx` | 116, 237–284 | drawer open/close grid wrapper |
| `apps/web/src/components/ThreadTerminalDrawer.tsx` | 111–154, 1519–1531, 1701–1802, 1804–1845 | drawer height limits, resize handle, frame |
| `apps/web/src/types.ts` | 34 | default drawer height 280 |
| `apps/web/src/components/WorkspacePageHeader.tsx` | 6–27 | header row of non-chat pages |
| `apps/web/src/components/DiffPanelShell.tsx` | 10–40 | diff panel header drag region |
| `apps/web/src/lib/workspaceFocus.ts` | 12–20, 100–299 | F6 areas and focus rules |
| `apps/web/src/lib/accessibilityAnnouncements.ts` | 1–37 | "Sidebar shown/hidden" live region |
| `apps/web/src/lib/windowControlsOverlay.ts` | 7–11, 32–36, 50–100 | `.wco`, `.electron`, `.electron-windows`, native title bar height |
| `apps/web/src/appInit.tsx` | 141–179, 252–258 | window title, hash history in Electron, Electron classes |
| `apps/web/src/branding.logic.ts` | 1–82 | stage label, display name, window title format |
| `apps/web/src/keybindings.ts` | 179–198, 228–250, 385–400 | shortcut resolution, labels, the bold exception |
| `packages/shared/src/keybindings.ts` | 57–68 | default shell shortcuts |
| `packages/contracts/src/settings.ts` | 155–164, 198–208, 339–340 | panel animation range, artwork modes, chat width |
| `apps/web/src/index.css` | 2, 8, 91–165, 469–506, 528–549, 679–712, 1082–1111, 1171–1179, 1234–1278, 1734–1785, 1827–1848, 2291–2299, 2550–2598 | geometry tokens, `.wco`, top fade, stage backdrop, art palette, safe-area and grain utilities, sidebar scope, root sizing, drag regions, chat width, lanes, focus ring |
| `apps/web/index.html` | 5–9, 389–440 | viewport meta, boot splash |
| `apps/web/src/components/CommandPalette.tsx` | 309–345 | the dialog root that wraps the whole shell |
| `apps/web/src/components/ThreadRouteView.tsx`, `apps/web/src/routes/_chat.tsx`, `apps/web/src/routes/__root.tsx` | 227–235; 18–46; 199–206 | where the inset and the layout mount |
| `apps/desktop/src/window/DesktopWindow.ts` | 46, 56–58, 75–88, 152–184, 250–261, 263–270, 320–360, 429–499, 510–613, 787–847, 904, 966–991, 1051–1100 | BrowserWindow options, title bar per platform, traffic lights, bounds persistence, zoom, fullscreen, reveal, splash |
| `apps/desktop/src/settings/DesktopAppSettings.ts` | 50–67 | default and minimum window size |
| `apps/desktop/src/preload.ts` | 46–71, 305–318 | macOS controls inset under zoom, fullscreen bridge |
| `apps/desktop/src/window/DesktopApplicationMenu.ts` | 249–303 | View menu zoom accelerators |
| `apps/desktop/src/preview/Manager.ts` | 1540–1571 | shell shortcuts forwarded from a focused preview guest |
| `apps/web/src/components/chat/useChatTerminalActions.ts` | 143–170 | opening the drawer may launch a shell |
| `apps/web/src/terminalUiStateStore.ts`, `apps/web/src/rightPanelStore.ts`, `apps/web/src/components/ChatView.logic.ts` | 22; 83, 205–214, 1260–1267; 108 | persisted store keys, details card visibility defaults, mounted drawer limit |
| `apps/web/src/components/ui/button.tsx`, `apps/web/src/components/ui/toggle.tsx`, `apps/web/src/components/ui/tooltip.tsx` | 8–73; 6–36; 16, 105–115 | trigger and toggle variants, tooltip layer and its title bar collision padding |
| `apps/web/src/components/Sidebar.tsx`, `apps/web/src/components/chat/ChatBannerLane.tsx`, `apps/web/src/components/chat/TimelineOverlays.tsx` | 2962–2968; 44; 33 | sidebar content order, overlay layers |
| `apps/web/src/components/AppRouteModal.tsx`, `apps/web/src/components/HistorySwipeNavigation.tsx`, `apps/web/src/components/settings/settingsLayout.tsx`, `apps/web/src/components/pullRequest/PullRequestsColumn.tsx` | 72–77; 7–40; 618; 233 | modal drag strip, swipe history, other top-fade users |

Shipped CSS: the minified `main-DcchvbMF.css` (one line, 538,735 bytes) holds every rule cited here. Byte offsets: `.chat-content-lane{` 26101, `.chat-composer-lane{` 30212, `.sidebar-stage-backdrop{` 32347, `.sidebar-stage-backdrop:after` 32547, `.topbar-scroll-fade{` 135214, `.wco{` 484316, `.no-transitions` 485752, `#root{` 513740, `.electron-windows{` 513934, `.drag-region{` 515584, `.drag-region>*` 17123, `:root[data-chat-width=wide]` 525429, `[data-workspace-keyboard-focus]:focus` 530696. Colour-mix rules appear twice, a plain fallback and the `color-mix()` version under `@supports`. Chrome uses the second.

Live page: same build, 1440×900 @2x, light and dark, on a draft view and on two existing threads. Narrow layouts were measured by setting the viewport to 960, 700 and 600 px wide and back. Reference captures stay in a private folder, outside this repo.

## 2. Structure

### 2.1 Element tree, desktop layout (window ≥ 768px)

Measured DOM. Classes are the live class lists, trimmed where they only carry variants for other configurations (floating/inset/icon sidebar, right side).

```
html                                    class "" | "dark"; desktop also "electron", "electron-windows" (Windows), "wco" (title bar overlay visible)
│                                       data-chat-width="comfortable|wide|full"
└─ body                                 inline background-color = chrome colour; background-image = grain in light only (none in dark, §3.1)
   ├─ #root                             width/height 100%, overflow-x clip, overflow-y hidden, padding-right var(--desktop-window-right-resize-inset)
   │  └─ div.contents                   CommandPalette wrapper, inert while the palette is open. It sits inside a Base UI Dialog root,
   │     │                              so every sheet below is a nested dialog
   │     └─ div[data-slot=sidebar-wrapper]
   │        │   class  group/sidebar-wrapper flex min-h-svh w-full max-sm:[--workspace-titlebar-control-size:--spacing(8)] h-dvh! min-h-0!
   │        │   data-sidebar-state="expanded|collapsed"  data-panel-animations="true|false"
   │        │   style  --sidebar-width:<px>; --sidebar-width-icon:3rem; --panel-animation-duration:<n>ms;
   │        │          --workspace-titlebar-content-left:calc(var(--workspace-controls-left) + var(--workspace-titlebar-control-size) + var(--workspace-titlebar-control-gap))
   │        │          [--workspace-controls-left:var(--desktop-window-controls-inset, 90px) on macOS desktop, not fullscreen]
   │        ├─ div[aria-hidden]         brand width probe: pointer-events-none invisible fixed top-0 left-0 flex w-max border-r border-transparent pr-3
   │        │  └─ div                   ml-[var(--workspace-titlebar-content-left)] flex gap-1 > brand mark [+ w-[calc(2*var(--workspace-titlebar-control-size))] spacer in Electron]
   │        ├─ div[data-slot=sidebar]   group peer hidden text-sidebar-foreground md:block
   │        │  │   data-state, data-collapsible ("" | "offcanvas" when collapsed), data-collapsible-mode=offcanvas, data-side=left, data-variant=sidebar; inert when collapsed
   │        │  ├─ div[data-slot=sidebar-gap]   relative w-(--sidebar-width) bg-transparent group-data-[collapsible=offcanvas]:w-0   (reserves the width in the row)
   │        │  └─ div[data-slot=sidebar-container][data-app-sidebar]   role=navigation aria-label="Threads" data-workspace-area=sidebar data-workspace-order=0 tabindex=-1
   │        │     │   class  fixed inset-y-0 z-10 hidden h-svh w-(--sidebar-width) md:flex left-0
   │        │     │          group-data-[collapsible=offcanvas]:left-[calc(var(--sidebar-width)*-1)] group-data-[side=left]:border-r
   │        │     ├─ div[data-slot=sidebar-inner]   flex h-full w-full flex-col bg-sidebar surface-grain
   │        │     │  └─ div                          relative flex min-h-0 flex-1 flex-col   (SidebarFolderDrop)
   │        │     │     ├─ div  sidebar header       relative flex h-[var(--workspace-topbar-height)] shrink-0 flex-row items-center gap-2 pr-3 [drag-region in Electron]
   │        │     │     │  ├─ div.sidebar-stage-backdrop   pointer-events-none absolute inset-x-0 top-0 z-0 h-20 select-none overflow-hidden > svg.h-full.w-full
   │        │     │     │  ├─ div  mobile trigger slot     relative top-[calc(var(--workspace-controls-top)-env(safe-area-inset-top))] z-10
   │        │     │     │  │                              ml-[calc(var(--workspace-controls-left)+1px-env(safe-area-inset-left))] flex shrink-0 items-center md:hidden
   │        │     │     │  └─ div  brand row               relative z-10 flex h-8 min-w-0 flex-1 flex-wrap content-start items-center gap-x-2 overflow-hidden py-0.5
   │        │     │     │     └─ div flex shrink-0 items-center gap-1 > a[aria-label="T3 Code home"] [+ back/forward buttons in Electron] [+ stage pill]
   │        │     │     ├─ div  w-full shrink-0 > div[data-sidebar=group] z-[1]   (search and filter row; sidebar doc)
   │        │     │     ├─ div[role=presentation]  ScrollArea, the thread list (sidebar doc)
   │        │     │     └─ div[data-sidebar=footer]   flex flex-col gap-2 px-[var(--sidebar-content-inset)] pt-1 pb-[var(--sidebar-content-inset)]
   │        │     └─ button[data-slot=sidebar-rail][data-sidebar=rail]   tabindex=-1, see §3.3
   │        ├─ main[data-slot=sidebar-inset]
   │        │      relative flex min-w-0 w-full flex-1 flex-col bg-background surface-grain h-svh min-h-0 overflow-hidden overscroll-y-none md:h-dvh
   │        │      └─ ChatView (§2.2), or the home, settings or other route content
   │        ├─ div.sr-only[aria-live=polite]    background thread announcer
   │        └─ div[data-sidebar-control]        pointer-events-none fixed left-[var(--workspace-controls-left)] top-[var(--workspace-controls-top)] z-50 ml-px
   │                                            flex h-[var(--workspace-topbar-height)] items-center   (hidden while the mobile sheet is mounted)
   │           └─ button[data-sidebar=trigger]  aria-label="Toggle main sidebar" aria-pressed=<visible>, see §3.3
   ├─ toast portals                     (toast doc)
   └─ div#workspace-visibility-alerts.sr-only   two aria-live="polite" aria-atomic="true" children; appended to body by the first sidebar announcement
```

What each part is for:

- The sidebar is two boxes. `sidebar-gap` is an in-flow spacer that pushes `main` to the right. `sidebar-container` is `position: fixed` and paints the sidebar. Collapsing shrinks the gap to 0 and moves the container to `left: -width`. The container is never removed.
- `[data-app-sidebar]` re-declares the palette for everything inside the container (01-tokens §3.5). The fixed trigger and the brand probe are outside it and use root tokens.
- The trigger is not inside the sidebar. It is a fixed control at the window's top-left, above both the sidebar header and the chat header, so it stays put when the sidebar collapses. On desktop widths the sidebar header leaves room for it with `ml-[var(--workspace-titlebar-content-left)]` on the brand.
- The brand width probe measures how wide the sidebar must be for the brand, plus the back/forward buttons in Electron, to fit beside the trigger. That width is the sidebar's minimum.

### 2.2 Chat screen regions (ChatView layout)

```
div  ChatView root               relative flex min-h-0 min-w-0 flex-1 overflow-hidden bg-background
├─ [title bar controls]          rendered here while an inline right panel is present (fixed position either way);
│                                display: none while the mobile sidebar sheet is mounted (PanelLayoutControls.tsx:180, 203)
├─ div[data-chat-column]         flex min-h-0 min-w-0 flex-col overflow-x-hidden flex-1
│  │                             maximized panel: w-0 flex-none, inert, data-chat-column-maximized-away="true"
│  ├─ header[data-chat-header]   browser:  relative bg-background flex h-[var(--workspace-topbar-height)] min-h-[var(--workspace-topbar-height)] shrink-0 items-center
│  │  │                                    pl-(--workspace-gutter-start) pr-(--workspace-gutter-end)
│  │  │                          Electron: relative bg-background drag-region flex h-[…] min-h-[…] shrink-0 items-center px-3 sm:px-5
│  │  │                                    [wco:pr-(--workspace-native-controls-inset) unless an inline right panel is open]
│  │  │                          both:     [[data-sidebar-state=collapsed]_&]:pl-[var(--workspace-titlebar-content-left)]
│  │  │                                    max-md:[[data-sidebar-state=expanded]_&]:pl-[var(--workspace-titlebar-content-left)]
│  │  ├─ [span]                  Electron with an inline panel present: pointer-events-none fixed top-[controls-top] right-[controls-right] h-[topbar] w-28 [-webkit-app-region:no-drag]
│  │  ├─ [title bar controls]    when no inline panel is present and the controls are not in a sheet
│  │  ├─ [thread details toggle] ThreadPanelHeaderControl, while an inline panel is open
│  │  └─ ChatHeader              flex min-w-0 flex-1 items-center gap-2 sm:gap-3 pr-24 | pr-10 (inline panel open)   (chat header doc)
│  ├─ div                        relative flex min-h-0 min-w-0 flex-1
│  │  └─ div[data-chat-canvas]   relative flex min-h-0 min-w-0 flex-1 flex-col
│  │     │                       style --chat-timeline-gutter:<px>; --chat-lane-inset-start:0px; --chat-lane-inset-end:<px>
│  │     ├─ div[aria-hidden]     width probe: pointer-events-none invisible absolute h-0 w-(--chat-content-max-width) min-w-[40rem] box-content ps-3 sm:ps-12
│  │     ├─ find bar, file drop overlay, banner lane (chat-banner-lane pointer-events-none absolute top-0 z-30 flex flex-col)
│  │     ├─ div[data-workspace-area=conversation]   relative flex min-h-0 flex-1 flex-col bg-background
│  │     │  ├─ timeline scroller  messages-timeline-scroll scrollbar-gutter-both h-full min-h-0 overflow-x-hidden overscroll-y-contain [overflow-anchor:none] [topbar-scroll-fade]
│  │     │  │  └─ rows, each wrapping its content in .chat-content-lane   (timeline doc)
│  │     │  └─ scroll-to-end button  chat-scroll-to-bottom pointer-events-none absolute z-30
│  │     ├─ div[data-chat-composer-overlay][data-workspace-area=composer]
│  │     │  │   thread view: pointer-events-none absolute inset-x-0 bottom-0 z-20 pt-1.5 sm:pt-2
│  │     │  │   draft view:  pointer-events-none absolute inset-0 z-20 flex items-center [@media(max-height:34rem)]:items-end
│  │     │  └─ div.chat-composer-lane.w-full
│  │     │     └─ div[data-chat-composer-stack]  group/composer-stack pointer-events-auto relative z-10 mx-auto w-full max-w-(--chat-content-max-width)
│  │     │        ├─ composer shell and context strip (composer doc)
│  │     │        └─ div[aria-hidden]  h-[calc(env(safe-area-inset-bottom)+1rem)] sm:h-[calc(env(safe-area-inset-bottom)+1.25rem)]
│  │     ├─ preview mini player (preview doc)
│  │     └─ thread details card, absolute z-20 (details doc)
│  └─ terminal drawer wrapper   grid shrink-0 overflow-clip grid-rows-[1fr] | grid-rows-[0fr] (hidden for inactive threads)
│     └─ div min-h-0 overflow-clip > aside[data-thread-terminal-drawer]   §3.10
└─ right panel                  inline: PreviewPanelShell host (§3.7); window ≤ 980px: RightPanelSheet portal (§3.8)
```

The header row is a sibling of the canvas, not an overlay. The timeline starts under it at y = 52. The composer is an overlay at the bottom of the canvas. It does not take flow space, so the timeline reserves room for it with its own end inset (timeline doc).

### 2.3 Region map at 1440×900

```
x      0          255 256                                               1148          1428  1440
  y 0 ┌────────────┬──────────────────────────────────────────────────────────────────────────────┐
      │ [◧] T3 Code│ chat header 256..1440 × 0..52              [▤] [▭] [◫]  at 1335..1427        │
   52 │ art 0..80  ├──────────────────────────────────────────────────────────────────────────────┤
      │ search row │ canvas 256..1440 × 52..900                     ┌ details card ──────┐        │
  100 ├────────────┤                                                │ 1148..1428, y ≥ 64 │        │
      │ thread     │    chat lane 380..1116, 736 wide               └────────────────────┘        │
      │ list       │                                                                              │
  769 │            │    composer stack 380..1116 × 768.5..900                                     │
  856 ├────────────┤                                                                              │
      │ footer     │                                                                              │
  900 └────────────┴──────────────────────────────────────────────────────────────────────────────┘
       sidebar 0..255, its border 255..256, rail hit area 247..263
```

| Region | Box x, y, w, h | Light background | Dark background | Edge | M/S |
|---|---|---|---|---|---|
| window (`body`) | 0, 0, 1440, 900 | `#fcfcfc` | `#000000` | none | M |
| sidebar container | 0, 0, 256, 900 | transparent | transparent | right 1px `#e4e4e7` / `#ffffff14` (paints `#141414` on black) | M |
| sidebar surface (`sidebar-inner`) | 0, 0, 255, 900 | `#fafafa` + grain | `#000000` + grain | none | M |
| sidebar header row | 0, 0, 255, 52 | art | art | none | M |
| stage backdrop | 0, 0, 255, 80 | art fading to `#fafafa` | art fading to `#000000` | none | M |
| brand row | 0, 10, 243, 32 | none | none | none | M |
| brand link | 52, 12, 55.05, 28 | none | none | none | M |
| search row group | 0, 52, 255, 48 | none | none | none | M |
| thread list | 0, 100, 255, 756 | none | none | none | M |
| footer | 0, 856, 255, 44 | none | none | none | M |
| rail (hit area) | 247, 0, 16, 900 | transparent | transparent | hover line 2px at x 255..257 | M |
| sidebar trigger | 13, 12, 28, 28 | transparent | transparent | none | M |
| `main` | 256, 0, 1184, 900 | `#fcfcfc` + grain, fully covered by children | `#000000` | none | M |
| chat header | 256, 0, 1184, 52 | `#fcfcfc` | `#000000` | none, no bottom divider | M |
| header content (ChatHeader) | 276, 14.91, 1144, 22.19 | none | none | right padding 96 | M |
| canvas, conversation area, timeline scroller | 256, 52, 1184, 848 | `#fcfcfc` (conversation area) | `#000000` | top 25px masked | M |
| chat lane | 380, n/a, 736, n/a | none | none | none | M |
| composer overlay | 256, 760.5, 1184, 139.5 | transparent | transparent | none | M |
| composer stack | 380, 768.5, 736, 131.5 | (composer doc) | | | M |
| details card | 1148, 64, 280, content height | (details doc) | | | M |
| title bar controls | 1335, 0, 92, 52 | transparent | transparent | none | M |

Other measured states at 1440×900:

| State | Boxes | M/S |
|---|---|---|
| sidebar collapsed | container −256, 0, 256×900 (off screen); `main` 0, 0, 1440×900; header `padding: 0 20px 0 52px`; lanes `padding-inline: 48px`, composer stack 352, 768.5, 736 wide (centred, the card at 1148 does not reach it); trigger 13, 12 in the ghost variant | M |
| right panel open, inline | chat column and header 256..900 (644); panel host 900..1440 with its border at 900..901; resize handle 901..905; tab bar 901, 0, 539×52; surface area 901, 52, 539×848; details toggle in the header at 860, 0, 28×52; fixed controls 1335..1427 (maximize 1335, terminal 1367, right panel 1399); composer stack 304, 548 wide | M |
| right panel maximized | chat column width 0; panel host 256, 0, 1184×900 | M |
| draft view | composer overlay 256, 52, 1184×848; composer stack 380, 394.25, 736×163.5; details card 1148, 64, 280×231 | M |

Grain: `main` carries `surface-grain`, but its only child (the ChatView root) has an opaque `bg-background`, so no grain shows in the chat area. Measured: chat area pixels are exactly `#fcfcfc` / `#000000`. The sidebar's grain does show. Its average painted value is about `#f9f9f9` on `#fafafa` and `#040404` on black.

### 2.4 Narrow layouts

| Window width | What changes | Source | M/S |
|---|---|---|---|
| ≤ 980px | The right panel opens as a sheet from the right edge instead of inline. A maximized panel stays inline. | rightPanelLayout.ts:1, ChatView.tsx:712, 1055 | M (sheet) / S (maximized) |
| ≤ 760px | Right sheet width becomes `min(88vw, 24rem)` and its 320px minimum is dropped. | RightPanelSheet.tsx:25 | S |
| < 768px | The sidebar becomes a sheet from the left (`Sheet` + `SheetPopup`, kept mounted). The wrapper reports `collapsed` unless the sheet is open. The header always takes the trigger inset. | sidebar.tsx:108–113, 324–371; workspaceTitlebar.ts:1–2 | M |
| < 640px | Title bar control size 32px instead of 28 (trigger 13, 10, 32×32; content left 56). Gutter 12px instead of 20. ChatHeader gap 8. Chat lanes 12px instead of 48; timeline padding 6px (12 − 6px gutter). Composer overlay top padding 6px, bottom spacer 16px. Panel toggles 32×32, group 104 wide. Sidebar rail hidden. Measured at 600: header 0..600 with padding 0 12px 0 56px, toggles at x 483, 519, 555, y 10, composer stack 12, 770.5, 576×129.5, no inline details card | sidebar.tsx:34, 580; index.css:137–143, 2550–2588; button/toggle `sm:` sizes | M |
| < 640px, side effect | The header reserve (12px padding + `pr-24` 96px = 108) is 9px less than the 32px toggle group needs (104 + 12 + 1 = 117), so header content can run under the fixed toggles by up to 9px. T3 does not compensate | ChatHeader.tsx:250–253, PanelLayoutControls.tsx:202 | M (derived from measured boxes) |

Mobile sheet tree:

```
div[data-slot=sheet-viewport]   fixed inset-0 z-(--z-sheet) flex justify-start   (z 46)
└─ div[data-slot=sidebar][data-mobile=true][role=dialog]   SheetPopup
   │   relative flex max-h-full min-h-0 min-w-0 flex-col not-dark:bg-clip-padding shadow-lg/5 border-e
   │   w-(--sidebar-width) max-w-none bg-sidebar surface-grain p-0 text-sidebar-foreground   style --sidebar-width: calc(100vw - var(--spacing) * 3)
   ├─ div[data-slot=sheet-header].sr-only   title "Threads", description "Displays the mobile sidebar."
   └─ div[data-app-sidebar][role=navigation]   flex h-full w-full flex-col pb-safe pt-safe pl-safe max-sm:[--workspace-titlebar-control-size:--spacing(8)]
      ├─ same sidebar content as desktop
      └─ button[data-slot=sidebar-rail]   still rendered here (it is a child of <Sidebar>), see §3.9
```

While this sheet is mounted (open, or closing), both the fixed `div[data-sidebar-control]` and the fixed `div[data-workspace-titlebar-controls]` group are `display: none` (AppSidebarLayout.tsx:383, PanelLayoutControls.tsx:180, 203). Measured at 700 with the sheet open: both report `display: none`.

### 2.5 Desktop app (Electron) structure

- One `BrowserWindow` loads the web build from a file-backed origin with hash history (appInit.tsx:252–253). There is no native toolbar, no vibrancy, no transparency. The web content fills the window, including the title bar area.
- macOS: `titleBarStyle: "hiddenInset"`. The native traffic lights float over the page's top-left. The page reserves their width through `--workspace-controls-left`.
- Windows and Linux: `titleBarStyle: "hidden"` with a `titleBarOverlay`. The native caption buttons float over the page's top-right. Chromium exposes the Window Controls Overlay API, the app adds `.wco` to `<html>`, and CSS reads `env(titlebar-area-*)`.
- `<html>` gains `electron` on every desktop platform and `electron-windows` on Windows (windowControlsOverlay.ts:80–100, appInit.tsx:255–258).
- Draggable areas are CSS `-webkit-app-region: drag` rectangles on the header rows (§5.8).

## 3. Measurements

### 3.1 Window, body and root

| Item | Property | Light | Dark | Source | M/S |
|---|---|---|---|---|---|
| `html` | inline `background-color` | `#fcfcfc` | `#000000` | chrome colour sync, 01-tokens §5.4 | M |
| `html` | `color-scheme` | `light` | `dark` | | M |
| `body` | inline `background-color` | `#fcfcfc` | `#000000` | | M |
| `body` | size, overflow, overscroll | 1440×900, `hidden`, `none` | = | index.css:1734–1743 | M |
| `body` | `background-image` | grain (`--surface-grain`, 256px tile) | `none`: the boot style `html.dark body { background: #000000 }` (specificity 0,1,2) resets the image and beats the app's `body { background-image: var(--surface-grain) }` (0,0,1). Only matters where nothing covers the body, such as the 6px Windows resize lane | index.html:398–408, index.css:1781–1785 | M |
| `body` | `min-height` | `calc(100svh + env(safe-area-inset-top))` = 900 | = | index.css:1736 | M |
| `#root` | size | 1440×900 | = | index.css:1745–1755 | M |
| `#root` | `overflow-x` / `overflow-y` | `clip` / `hidden` | = | | M |
| `#root` | `padding-top` | `max(env(safe-area-inset-top), 0px)` = 0 | = | | M |
| `#root` | `padding-right` | `var(--desktop-window-right-resize-inset)` = 0 (6px on Windows desktop) | = | index.css:1754–1759 | M / S |
| `<meta name=theme-color>` | content | `oklch(0.992 0 0)` (`#fcfcfc`) | `rgb(0, 0, 0)` | useTheme.ts:294–303 | M |
| `<meta name=viewport>` | content | `width=device-width, initial-scale=1.0, viewport-fit=cover, interactive-widget=resizes-content` | = | index.html:5–8 | S |
| boot splash (before React) | layout | `#boot-shell` flex-centred, 96×96 card, 64×64 app icon, on body `#ffffff` | on `#000000` | index.html:389–440 | S |
| sidebar wrapper | box | 0, 0, 1440, 900, `display: flex` | = | | M |
| sidebar wrapper | inline vars | `--sidebar-width: 256px`, `--sidebar-width-icon: 3rem`, `--panel-animation-duration: 0ms`, `--workspace-titlebar-content-left` resolves to 52px | = | AppSidebarLayout.tsx:451–455, sidebar.tsx:241–249 | M |

### 3.2 Sidebar frame

| Item | Value | Source | M/S |
|---|---|---|---|
| default width | 256px (`16 * 16`) | threadSidebarWidth.ts:2, sidebar.tsx:30 | M |
| minimum width | `max(208, ceil(brand probe width))`. Probe here: 120.05 (52 + 55.05 brand + 12 padding + 1 border), so 208 | threadSidebarWidth.ts:3, 6–8; SidebarChrome.tsx:96–121 | M |
| minimum width, macOS desktop | probe = 130 + 55.05 + 4 gap + 56 back/forward spacer + 12 + 1 = 258.05, so 259. The default 256 is clamped up to 259. Font dependent | same | S |
| minimum width, Windows/Linux desktop | probe = 52 + 55.05 + 4 + 56 + 13 = 180.05, so 208 | same | S |
| maximum width | `max(min, floor(window width) − 640)`. 800 at 1440 | threadSidebarWidth.ts:4, 10–15 | M |
| main content floor | 640px (`40 * 16`). A drag cannot widen the sidebar past `wrapper width − 640` | AppSidebarLayout.tsx:478–480 | M |
| mobile width | `calc(100vw − 12px)`. 688 at 700 | sidebar.tsx:31 | M |
| container | `position: fixed`, `inset-y: 0`, `z-index: 10`, width `--sidebar-width`, `border-right: 1px` | sidebar.tsx:397–411 | M |
| container border colour | `#e4e4e7` / `#ffffff14` (sidebar-scoped `--contrast-border`) | index.css:1243, 1264 | M |
| surface | `bg-sidebar` `#fafafa` / `#000000`, plus `surface-grain` (256px tile at 3.5%) | index.css:1245, 1266, 1098–1102 | M |
| header row | height `--workspace-topbar-height` 52, `padding-right: 12px`, `gap: 8px`, `align-items: center` | SidebarChrome.tsx:60–65 | M |
| brand row | `height: 32px`, `padding-block: 2px`, `column-gap: 8px`, at y 10 | SidebarChrome.tsx:74 | M |
| brand link | `margin-left: var(--workspace-titlebar-content-left)` (52px), `height: 28px`, radius 6px, x 52, y 12, width 55.05 | SidebarChrome.tsx:165–178 | M |
| brand mark | 14px / 20px, weight 500, `letter-spacing: -0.35px` (`tracking-tight`), `gap: 4px`. "T3" wordmark SVG 16.84×10.17 (`h-[1cap]`). "Code" 34.2 wide. Font dependent | SidebarChrome.tsx:180–194 | M |
| brand colour on art | `#ffffff`, "Code" `#ffffffb2` (white/70) | SidebarChrome.tsx:171, 187 | M |
| brand colour without art | `text-foreground`, "Code" `text-muted-foreground` | same | S |
| search row group | y 52..100, `padding: 8px`, `z-index: 1` (paints over the art's tail) | Sidebar.tsx:2968 | M |
| thread list | y 100..856 | sidebar.tsx:676–708 | M |
| footer | y 856..900, padding 4px 8px 8px, `gap: 8px`; its menu row 8, 860, 239, 32 | sidebar.tsx:662–674 | M |

### 3.3 Sidebar trigger and rail

The trigger is a `Button` (`size="icon"`, overridden to `size-[var(--workspace-titlebar-control-size)]!`), wrapped in a fixed `div[data-sidebar-control]` at `left: 12px; margin-left: 1px; top: 0; height: 52px; z-index: 50`.

| Trigger property | Value | M/S |
|---|---|---|
| box | 13, 12, 28×28 (32×32 below 640px, at 13, 10) | M |
| radius, border | 8px (`--control-radius`), 1px transparent | M |
| icon | lucide `PanelLeftClose` while the sidebar is visible, `PanelLeft` while hidden, 16×16, stroke 1.75, at 19, 18 | M |
| label | `aria-label="Toggle main sidebar"`, `aria-pressed` = visible; sr-only text "Toggle Sidebar" | M |
| tooltip | "Toggle main sidebar (<shortcut>)", side bottom; popup measured at 5, 44, 203.75×26.5 | M |
| variant | `media-navigation` while the sidebar is visible and stage art is shown, else `ghost` | S |
| `-webkit-app-region` | `no-drag` | S |
| transition | `box-shadow, scale` 150ms `cubic-bezier(0.4, 0, 0.2, 1)`. Background changes are instant | M |

| Trigger state | Light | Dark | M/S |
|---|---|---|---|
| on art, default | text and icon `#ffffffe6`, background transparent | = | M |
| on art, hover / pressed | background `#ffffff1a`, text `#ffffff` | = | M |
| on art, focus-visible | `box-shadow: inset 0 0 0 1px transparent, inset 0 0 0 3px #ffffff` (a 3px white inner ring) | = | M |
| ghost, default | text `#18181b`, icon `#71717b` (`--contrast-muted-foreground`), background transparent | text `#f5f5f5`, icon `#a1a1a1` | M |
| ghost, hover / `data-pressed` | background `#f4f4f5` | background `#ffffff0a` | M |
| ghost, focus-visible | `inset 0 0 0 1px #fcfcfc, inset 0 0 0 3px #1b4ed8` (1px gap, 2px ring) | `inset 0 0 0 1px #000000, inset 0 0 0 3px #346bf1` | M |
| `:active` | `scale: 0.97` | = | S |

Rail: `button[data-slot=sidebar-rail]`, a child of the fixed container.

| Rail property | Value | M/S |
|---|---|---|
| classes | `-translate-x-1/2 group-data-[side=left]:-right-4 absolute inset-y-0 z-20 hidden w-4 sm:flex after:absolute after:inset-y-0 after:left-1/2 after:w-[2px] hover:after:bg-sidebar-border touch-none` | M |
| hit area | 16px wide, full height, centred on the sidebar's right border: x 247..263, y 0..900 | M |
| hover line | the `::after`, 2px wide at x 255..257 (over the 1px border and 1px into `main`), colour `#e4e4e7` / `#ffffff14` | M |
| cursor | `w-resize` while expanded. While collapsed it moves to `-right-2`, its line to `left: 100%`, cursor `e-resize`, and `pointer-events: none` | M |
| below 640px | `display: none` (`hidden sm:flex`) | S |
| focus | `tabindex=-1`, never keyboard-focusable | M |
| label | `aria-label` "Resize Sidebar" when resizable and open, else "Toggle Sidebar" | M |
| tooltip | "Drag to resize sidebar" (or "Toggle Sidebar"), side right. It anchors to the rail's centre: popup measured at 267, 437, 142.7×26.5, whatever the pointer's y | M |
| during a drag | `body` gets inline `cursor: col-resize; user-select: none`. Gap and container get inline `width` and `transition-duration: 0ms` | M |

### 3.4 Stage backdrop (the art behind the sidebar header)

The art is inline SVG drawn by React. There is no raster asset. The same component, with a shifted viewBox, also paints the send button's art (composer doc).

Container and compositing:

| Item | Value | M/S |
|---|---|---|
| container | `div.sidebar-stage-backdrop`, `position: absolute; left/right/top: 0; height: 80px (h-20); z-index: 0; overflow: hidden; pointer-events: none; user-select: none` | M |
| box | 0, 0, 255, 80. It overflows the 52px header by 28px, behind the search row (`z-index: 1`) | M |
| `mask-image` | `linear-gradient(to bottom, black 0%, black 55%, transparent 92%)`, one layer, whole box | M |
| mask in px | alpha 1 down to y 44, linear to 0 at y 73.6, 0 below | M |
| `::after` | `content: ""; position: absolute; inset: 0;` background `linear-gradient(to bottom, transparent 0%, transparent 28%, F10 40%, F30 52%, F58 64%, F82 75%, F96 85%, var(--stage-fade) 93%)`, where `Fnn` is `color-mix(in srgb, var(--stage-fade) nn%, transparent)`. Measured light: `#fafafa1a`, `#fafafa4d`, `#fafafa94`, `#fafafad1`, `#fafafaf5`, `#fafafa`; dark the same alphas on `#000000` | M |
| `::after` stops in px | 0 at 0..22.4, 0.10 at 32, 0.30 at 41.6, 0.58 at 51.2, 0.82 at 60, 0.96 at 68, 1.0 at 74.4 | M |
| fade colour `--stage-fade` | `var(--sidebar-stage-fade, var(--app-chrome-background))`, which is the sidebar surface: `#fafafa` light, `#000000` dark | M |
| blending | normal source-over. No `mix-blend-mode`. The `::after` is inside the masked box, so the mask applies to it too | S |
| effective art opacity | `eff(y) = mask(y) × (1 − after(y))`, drawn over the sidebar surface of the same colour as the fade. Result: `bg + eff(y) × (art − bg)` | M (checked against pixels) |

`eff(y)`: 1.00 at 0..22.4, 0.98 at 24, 0.90 at 32, 0.73 at 40, 0.63 at 44, 0.44 at 48, 0.29 at 52, 0.17 at 56, 0.08 at 60, 0.04 at 64, 0.01 at 68, 0 from 73.6. Example check, light, x 10, y 40: sky `#07152f`, eff 0.733 over `#fafafa` gives `#485265`. Painted: `#495468`.

SVG framing:

| Item | Value | M/S |
|---|---|---|
| element | `svg.h-full.w-full`, `fill="none"`, `viewBox="0 0 8192 96"`, `preserveAspectRatio="xMinYMin slice"` | M |
| scale | `slice` covers the 255×80 box: scale = max(255/8192, 80/96) = 0.8333. 96 user units become 80px. Anchored top-left | M |
| visible user units | width ÷ 0.8333 = 1.2 × width. 306 units at 255px, 960 at the 800px maximum | M |
| variant `data-stage-art` | `nightly` on this build; the dev art's svg carries `data-stage-art="blueprint"` (not `dev`) | M / S (SidebarStageBackdrop.tsx:113, 228) |
| artwork choice | mode `artwork` (default): `nightly` art when the stage label is Nightly, `dev` art when Dev, nothing otherwise. `nightly-artwork` / `dev-artwork` force one. `pill` shows a "Nightly"/"Dev" badge instead. `none` shows nothing | S |

Nightly sky art (SidebarStageBackdrop.tsx:102–212), drawn in this order. All coordinates are user units, `gradientUnits` and `patternUnits` are `userSpaceOnUse`.

| # | Layer | Geometry | Paint | M/S |
|---|---|---|---|---|
| 1 | sky | rect, width 100% (8192), height 96 | linear gradient (24, 0) → (264, 96), `spreadMethod="reflect"`. Stops 0 `--stage-night-bottom` `#07152f`, 0.5 `--stage-night-mid` `#151443`, 1 `--stage-night-top` `#32155b` | S geometry, M colours |
| 2 | glows | rect 100%×96 filled by a pattern 640×96 that holds one rect 640×96 | radial gradient `cx=0 cy=0 r=1`, `gradientTransform="translate(216 18) rotate(137) scale(120 84)"`: an ellipse centred at (216, 18), radii 120 and 84, rotated 137°. Stops 0 `#5165d8` @0.40, 0.5 `#283075` @0.16, 1 `#07152f` @0. Repeats every 640 units | S, M colours |
| 3 | stars | rect 100%×96 filled by a pattern 288×96 | 17 circles in `#e4eaff` (`--stage-night-line`), each with its own `fill-opacity` (list below). 3 sparkles: two 3-unit strokes each (horizontal `M x−1.5 y H x+1.5`, vertical `M x y−1.5 V y+1.5`), stroke `#c8d7ff` (`--stage-night-sparkle`), width 0.6, opacity 0.7, round caps | S, M colours |
| 4 | cloud A | path `M-12 88C-12 74 0 63 14 63C18 50 30 41 44 41C58 41 70 49 74 62C79 57 86 54 94 54C110 54 123 66 124 82C132 83 138 88 141 96H-12V88Z` | fill linear gradient (0, 60) → (288, 96): stops 0 `#4ea4ff` @0.50, 0.52 `#696fea` @0.62, 1 `#a85bea` @0.50. Filter: Gaussian blur, stdDeviation 4 units (3.33px), filter region x −24, y −24, 336×144 | S, M colours |
| 5 | cloud B | path `M150 96C151 84 161 75 173 75C176 64 186 57 198 57C210 57 220 64 223 75C231 75 238 80 241 87C250 87 257 91 260 96H150Z` | same gradient and blur, `fill-opacity: 0.8` | S |

Stars (cx, cy, r, opacity): (14,10,0.6,0.85) (38,22,0.4,0.55) (58,8,0.5,0.7) (84,16,0.4,0.5) (104,7,0.6,0.8) (126,20,0.4,0.55) (148,11,0.5,0.7) (170,24,0.4,0.5) (192,9,0.6,0.8) (214,18,0.4,0.55) (236,8,0.5,0.7) (258,20,0.45,0.6) (278,11,0.55,0.75) (26,34,0.4,0.45) (118,34,0.4,0.45) (202,32,0.4,0.5) (268,34,0.4,0.45). Sparkles at (70,28), (160,36), (246,26). Source: SidebarStageBackdrop.tsx:71–100.

The clouds are not tiled. They only cover x < 260 units (217px). A wider sidebar shows tiled sky, glows and stars to its right.

Dev blueprint art (SidebarStageBackdrop.tsx:214–393), same framing:

| # | Layer | Geometry | Paint | M/S |
|---|---|---|---|---|
| 1 | paper | rect 100%×96 | linear (60, 0) → (220, 96), reflect. Stops 0 `--stage-art-bottom`, 0.5 `--stage-art-mid`, 1 `--stage-art-top` | S |
| 2 | glows | pattern 768×96 with three rects | radial A `translate(216 14) rotate(137) scale(120 84)`: highlight @0.40, 0.52 secondary @0.16, 1 bottom @0. Radial B `translate(474 44) rotate(166) scale(156 92)`: celeste-highlight @0.34, 0.5 celeste-secondary @0.18, 1 bottom @0. Radial C `translate(704 18) rotate(145) scale(132 88)`: violet-highlight @0.30, 0.52 tertiary @0.14, 1 bottom @0 | S |
| 3 | minor grid | pattern 8×8, path `M8 0H0V8` | stroke `--stage-art-grid-line`, opacity 0.14, width 0.5 | S |
| 4 | major grid | pattern 32×32, path `M32 0H0V32` | stroke grid-line, opacity 0.26, width 0.6 | S |
| 5 | ruler | rect 100%×6 filled by pattern 32×6, path `M4 0V2.5M12 0V2.5M20 0V4M28 0V2.5` | stroke `--stage-art-line`, opacity 0.5, width 0.5 | S |
| 6 | annotations | pattern 768×96: dashed dimension lines with end ticks, small crosses, three dashed circles with cross-hairs | stroke `--stage-art-line`, opacities 0.35 to 0.6, widths 0.4 to 0.7, round caps. Exact coordinates and dash arrays at SidebarStageBackdrop.tsx:326–382 | S |

Palette, resolved live on the backdrop (both themes):

| Token | Light | Dark | M/S |
|---|---|---|---|
| `--stage-night-top` / `-mid` / `-bottom` | `#32155b` / `#151443` / `#07152f` | = | M |
| `--stage-night-highlight` / `-secondary` / `-tertiary` | `#4ea4ff` / `#696fea` / `#a85bea` | = | M |
| `--stage-night-line` / `-sparkle` | `#e4eaff` / `#c8d7ff` | = | M |
| `--stage-night-glow-highlight` / `-glow-secondary` | `#5165d8` / `#283075` | = | M |
| `--stage-art-top` / `-mid` / `-bottom` | `#67c2ff` / `#347ff8` / `#1538d0` | `#3a7ad1` / `#2050ae` / `#101f6e` | M |
| `--stage-art-highlight` / `-secondary` / `-tertiary` / `-line` | `#d4f6ff` / `#65c8ff` / `#7c8bff` / `#ddf7ff` | = | M |
| `--stage-art-celeste-highlight` / `-celeste-secondary` / `-violet-highlight` / `-grid-line` | `#d2ffff` / `#48dcf5` / `#d9d8ff` / `#eaf6ff` | = | M |
| `--stage-fade` | `#fafafa` | `#000000` | M |

The nightly sky has no dark variant. Only the fade colour changes between themes. Built-in themes swap these pigments (index.css:757–921, theme doc).

Painted pixels, nightly art, sidebar at 256px. Read from 2x screenshots at the CSS point shown. Values include grain and can differ by 1 to 3 per channel. Stars and the brand are avoided except where noted.

| y | Light x=10 | Light x=150 | Dark x=10 | Dark x=150 |
|---|---|---|---|---|
| 0 | `#081530` | `#1c194f` | `#081530` | `#1c194f` |
| 12 | `#07152f` | `#1e1d56` | `#07152f` | `#1e1d56` |
| 20 | `#07152f` | `#211d58` | `#07152f` | `#211d58` |
| 28 | `#15233b` | `#2e2b63` | `#07142c` | `#1f1c54` |
| 32 | `#202d44` | `#383469` | `#06132a` | `#1e1a4f` |
| 40 | `#495468` | `#5c5883` | `#051024` | `#18143f` |
| 48 | `#959dad` | `#9d9ab5` | `#081020` | `#100d28` |
| 56 | `#d8dee7` | `#dbd9e7` | `#080f18` | `#0c0a17` |
| 64 | `#f2f4f6` | `#f2f2f6` | `#040507` | `#050407` |
| 72 | `#f8f8f8` | `#f9f9f9` | `#020202` | `#020202` |
| 80 | `#f9faf9` | `#f8f9fa` | `#020302` | `#030405` |

### 3.5 Chat column

| Item | Value | Light | Dark | Source | M/S |
|---|---|---|---|---|---|
| chat column | x 256, width 1184 (window − sidebar), `flex-direction: column`, `overflow-x: hidden` | | | ChatView.tsx:3303–3311 | M |
| header height | `--workspace-topbar-height` 52px (40px under `.wco`) | | | index.css:123, 156 | M / S |
| header padding, browser | 20px / 20px (`--workspace-gutter-start/end`, 12px below 640) | | | ChatView.tsx:3325 | M |
| header padding, collapsed sidebar or < 768px | left `--workspace-titlebar-content-left`: 52px (56 below 640, 130 macOS desktop) | | | workspaceTitlebar.ts:1–2 | M |
| header padding, Electron | `px-3 sm:px-5` (12 / 20), plus right `--workspace-native-controls-inset` under `.wco` | | | ChatView.tsx:3320–3323 | S |
| header background | `bg-background` | `#fcfcfc` | `#000000` | | M |
| header bottom edge | none (border width 0) | | | | M |
| header content | ChatHeader `gap: 12px` (8 below 640), `padding-right: 96px` (`pr-24`, reserves the fixed controls), 40px (`pr-10`) while an inline panel is open | | | ChatHeader.tsx:250–253 | M |
| canvas | 256, 52, 1184×848 | | | ChatCanvas.tsx:96–109 | M |
| conversation area | same box, `bg-background` | `#fcfcfc` | `#000000` | ChatView.tsx:3395–3398 | M |
| lane padding | 48px each side (`3rem`, 12px below 640) plus `--chat-lane-inset-start/end` | | | index.css:2551–2588 | M |
| lane max width | `--chat-content-max-width`: 736px (`46rem`) comfortable, 1152px (`72rem`) wide, `100%` full | | | index.css:131, 2293–2299 | M |
| lane with details card inline | x 380, width 736, `--chat-lane-inset-end: 200px` | | | chatCanvasLayout.ts:43–55 | M |
| lane without details card | x 480, width 736, inset 0 (centred) | | | | M |
| lane, wide or full, details card inline | x 304, width 812 (limited by the card), `--chat-lane-inset-end: 276px`, timeline `padding-right: 318px` | | | | M |
| lane, wide or full, no card | x 304, width 1088 (canvas − 96) | | | | M |
| timeline scroller | full canvas box, `scrollbar-gutter: stable both-edges` (6px each side, so `clientWidth` 1172), `padding-inline: 42px` and `42 + inset-end` (242 with the card) | | | MessagesTimeline.tsx:1462–1465, index.css:2559–2586 | M |
| composer overlay, thread view | absolute, bottom 0, full canvas width, `padding-top: 8px` (6 below 640), z 20. Box 256, 760.5, 1184×139.5 | | | ChatView.tsx:3519–3523 | M |
| composer overlay, draft view | absolute, `inset: 0`, flex `align-items: center` (bottom below 544px window height). Box 256, 52, 1184×848; the stack centres at y 394.25 | | | | M |
| composer lane | same padding rule as the timeline (48 + inset; 248 on the right with the card) | | | | M |
| composer stack | `max-width: --chat-content-max-width`, centred in the lane: 380, 768.5, 736×131.5. Shell 111.5 + bottom spacer 20 (16 below 640, plus safe-area) | | | ChatView.tsx:3525–3529, 3825–3828 | M |
| details card slot | x = canvas width − 280 − 12, y = 12 (+ find bar), width 280 (`--thread-details-panel-width` 17.5rem), height up to canvas height − 24. Measured 1148, 64, 280 | | | threadDetailsCardLayout.ts:4–6, 17–47 | M |

Top fade on the timeline (`topbar-scroll-fade`, index.css:469–506), measured on the scroller:

| Property | Value | M/S |
|---|---|---|
| `mask-image` | layer 1 `linear-gradient(transparent 0%, #0000001a 10%, #0000004d 24%, #00000094 42%, #000000d1 62%, #000000f5 82%, #000000 100%)`, layers 2 and 3 solid black | M |
| `mask-size` | `100% 25px`, `100% calc(100% − 24px)`, `6px 100%` | M |
| `mask-position` | top, bottom, right (`50% 0%`, `50% 100%`, `100% 50%`), no repeat | M |
| effect | content alpha ramps over the top 25px (24px fade height + 1px overlap): 0 at 0, 0.10 at 2.5, 0.30 at 6, 0.58 at 10.5, 0.82 at 15.5, 0.96 at 20.5, 1 at 25. The rightmost 6px (scrollbar lane) is never masked | M |
| when off | while a banner sits at the top of the timeline (`topFadeEnabled = !hasTimelineTopBanner`) | S |
| other users | settings content and pull request list scrollers (settingsLayout.tsx:618, PullRequestsColumn.tsx:233) | S |

### 3.6 Title bar panel controls

`div[data-workspace-titlebar-controls]`: `pointer-events: none; position: fixed; top: var(--workspace-controls-top); right: var(--workspace-controls-right); z-index: 50; margin-right: 1px; height: var(--workspace-topbar-height); display: flex; align-items: center; gap: 4px; -webkit-app-region: no-drag`. Its inner group is `pointer-events-auto`.

| Item | Value | M/S |
|---|---|---|
| group box | 1335, 0, 92×52 (right edge 1427 = 1440 − 12 − 1) | M |
| order | thread details (`SquareMenu`), terminal drawer (`PanelBottom`), right panel (`PanelRight`) | M |
| buttons | x 1335, 1367, 1399; y 12; 28×28 (32×32 below 640) | M |
| maximize control | `Maximize2` / `Minimize2`, kept in the group but `position: absolute; right: 100%; margin-right: 4px; opacity: 0; pointer-events: none; inert` while the right panel is closed (box 1303, 12). It becomes the first in-flow button while the panel is open | M |
| toggle classes | `Toggle variant="ghost" size="sm"`: `h-8 min-w-8 px-[calc(--spacing(1.5)-1px)] sm:h-7 sm:min-w-7`, `rounded-lg` (8px), 1px transparent border, `font-medium text-base sm:text-sm` | M |
| icon | 16×16, stroke 1.75, opacity 0.8, colour = text colour | M |
| default | text `#18181b` / `#f5f5f5`, background transparent | M |
| hover | `bg-accent` `#f4f4f5` / `#ffffff0a`, text unchanged | M |
| pressed (`data-pressed`, `aria-pressed=true`) | `bg-accent` `#f4f4f5` / `#ffffff0a`, text `accent-foreground` | M |
| focus-visible | `inset 0 0 0 1px #fcfcfc, inset 0 0 0 3px #1b4ed8` light; `#000000` and `#346bf1` dark | M |
| disabled | text and icon `#71717b` light, `#a1a1a1` dark (`text-muted-foreground`); icon opacity stays 0.8; `pointer-events: none`; background transparent | P |
| labels | `aria-label` "Toggle thread details panel" (tooltip says "Toggle thread details"), "Toggle terminal drawer", "Toggle right panel" (also `data-right-panel-toggle`), "Maximize panel" / "Restore panel size" | M |
| mobile sidebar sheet mounted | the whole group is `display: none` (so is the header's details toggle) | M |
| tooltips | "Toggle thread details (<key>)", "Toggle terminal drawer (<key>)" or "Terminal drawer is unavailable", "Toggle right panel (<key>)" or "Right panel is unavailable", "Maximize panel" / "Restore panel size". All side bottom | M / S |
| right reserve in the header | ChatHeader `pr-24` (96px); the group needs 92 + 1 | M |

While an inline right panel is open, the thread details toggle leaves the fixed group and moves into the chat header as `ThreadPanelHeaderControl`: `absolute top-[var(--workspace-controls-top)] right-[calc(env(safe-area-inset-right)+0.75rem)] z-50 flex h-[var(--workspace-topbar-height)] items-center`. Measured at 860, 0, 28×52 with the panel open.

At 1440 with the panel open the canvas is only 644px wide, below the 1012px the inline card needs (§5.4), so the details card switches to popover mode. That happens at the default window size whenever the right panel opens; it is not a narrow-window edge case. Measured: no inline card is rendered. The header toggle becomes a Base UI popover trigger (`data-slot=popover-trigger`, `aria-haspopup=dialog`, `aria-pressed=false` while closed). Clicking it opens the card as a popover anchored to the whole chat header (`header[data-chat-header]`): `side="bottom"`, `align="end"`, `alignOffset` 4 (12px gap − 8px panel padding), `sideOffset` = find bar inset (0), collision avoidance `shift`. Popup measured at 600, 52, 296×247, right edge 896 = header right 900 − 4; positioner z 46 (ThreadDetailsCard.tsx:173–190, ChatView.tsx:3314, 3855). Its open state is not persisted (rightPanelStore.ts:1193–1212). The card's own look is in the details doc.

### 3.7 Right panel, inline (window > 980px)

Host: `PreviewPanelShell`, `div[data-preview-panel-mode=inline][data-workspace-area=panel][role=region]`.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| host classes | `relative flex h-full min-h-0 min-w-0 max-w-full flex-col self-stretch bg-background shrink-0 border-l border-border` | | | M |
| host box | 900, 0, 540×900. The chat column shrinks to 256..900 (644) | | | M |
| width | `var(--resizable-width)`, set inline in px | | | M |
| default width | 540px | | | M |
| minimum | 360px | | | M |
| maximum | `max(360, min(floor(0.7 × window width), floor(layout width) − 360))`. Layout width is the ChatView root. 824 at 1440 with the sidebar (1184 − 360) | | | M |
| background | `bg-background` | `#fcfcfc` | `#000000` | M |
| left border | 1px | `#e4e4e7` | `#ffffff14` | M |
| content wrapper | `h-full min-h-0 w-full overflow-clip flex justify-end` > `flex h-full min-h-0 min-w-0 flex-col shrink-0`, width `calc(var(--resizable-width) − 1px)` = 539 | | | M |
| tab bar row | `div[data-right-panel-tabbar]`: 901, 0, 539×52, `gap: 4px`, `padding-left: 8px`, `padding-right: 112px` (`pr-28`, the fixed controls) | | | M |
| surface area | `div[data-right-panel-surface-content]`: 901, 52, 539×848 | | | M |
| empty state | "Open a surface" list, centred, `pb-(--workspace-topbar-height)` (right panel doc) | | | M |

Resize handle (`RightPanelResizeHandle`, `div[role=separator][aria-orientation=vertical]`):

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| classes | `group absolute inset-y-0 left-0 z-40 w-1 cursor-col-resize touch-none select-none any-pointer-coarse:-left-2 any-pointer-coarse:w-4` | | | M |
| hit area | 4px wide at x 901..905 (inside the border), full height, z 40. Coarse pointers: 16px at x 893..909 | | | M / S |
| line | `span`, 1px wide, `left: 0; translate: -50%`: x 900.5..901.5, straddling the border's right half | | | M |
| line default | transparent | | | M |
| line hover (`group-hover:bg-border`, i.e. `--contrast-border`) | | `#e4e4e7` | `#ffffff14` | M dark, P light |
| line active / focus-visible (`bg-primary/60` = `color-mix(in oklab, var(--primary) 60%, transparent)`) | | `#1b4ed899` (`#1b4ed8` at 0.6) | `#346bf199` (`#346bf1` at 0.6) | P (computed value; a live press could not be produced, §7) |
| line transition | `transition-colors 150ms cubic-bezier(0.4, 0, 0.2, 1)` | | | M |
| cursor | `col-resize`; during a drag `body` gets `cursor: col-resize; user-select: none` | | | M |
| focus | not focusable here (no `tabIndex` passed) | | | M |

Maximized (`data-preview-panel-maximized=true`):

| Item | Value | M/S |
|---|---|---|
| chat column | width 0, `flex: none`, `inert`, `data-chat-column-maximized-away="true"` | M |
| host | `flex-1`, width 100%: 256..1440 (1184), no left border, no resize handle | M |
| tab bar | still `padding: 0 112px 0 8px`; while the sidebar is collapsed it takes the collapsed-sidebar left inset | M / S |
| controls | maximize button pressed (`Minimize2`, "Restore panel size"), right panel toggle pressed | M |
| at ≤ 980px | a maximized panel renders inline, not as a sheet | S |

### 3.8 Right panel as a sheet (window ≤ 980px)

Measured at a 960px window.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| viewport | `div[data-slot=sheet-viewport]`: `fixed inset-0 z-(--z-sheet) flex justify-end`, z 46 | | | M |
| backdrop | none. The sheet is nested in the command palette's dialog root and Base UI renders no backdrop for nested dialogs. The page behind stays fully visible and undimmed | | | M |
| popup | `div[data-slot=sheet-popup][role=dialog]` 556.81, 0, 403.19×900 | | | M |
| popup classes | `relative flex max-h-full min-h-0 flex-col bg-popover not-dark:bg-clip-padding text-popover-foreground shadow-lg/5 border-s col-start-2 w-[min(42vw,28rem)] min-w-80 max-w-[28rem]` (+ slide classes) | | | M |
| width | `min(42vw, 28rem)`: 403.19 at 960. Min 320, max 448. ≤ 760px: `min(88vw, 24rem)`, no minimum | | | M / S |
| WCO | `margin-top: --workspace-topbar-height`, `height: calc(100% − topbar)`, so the native caption buttons stay uncovered | | | S |
| popup background (computed) | `bg-popover`; `not-dark:bg-clip-padding` | `#ffffff` | `#111111` | M |
| visible surface | The inner host (below) covers the popup's whole padding box, so `bg-popover` never shows inside the sheet. Pixels inside the sheet: | `#fcfcfc` | `#000000` | M (pixels) |
| left border | 1px `border-s` | `#e4e4e7`; the background is clipped to the padding box, so the border sits over the page: paints `#e4e4e7` | `#ffffff14`; dark is not clipped, so the border sits over the popup's own `#111111`: paints `#242424` | M (computed and pixels) |
| shadow | `0 10px 15px -3px #0000000d, 0 4px 6px -4px #0000000d`; darkens the chat next to the sheet edge to about `#f8f8f8` in light | = | = | M |
| `::before` hairline | `inset: 0`, `box-shadow: 0 1px 0 0 #0000000a` light, `0 -1px 0 0 #ffffff0f` dark; hidden below 640 | | | M |
| inner host | `PreviewPanelShell` mode `sheet`: `w-full`, background `#fcfcfc` / `#000000`, box 557.81, 0, 402.19×900 | | | M |
| mounting | `keepMounted`, `showCloseButton={false}` (RightPanelSheet.tsx:20–25). ChatView still unmounts the whole right panel once it is closed and its exit delay has passed, so a closed sheet is not in the DOM (measured: no `sheet-popup` after closing) | | | S / M |
| tab bar | 557.81, 0, 402.19×52, `padding: 0 12px 0 8px` | | | M |
| in-panel controls | maximize, thread details, terminal, right panel, 28px each, `gap: 4px`, right edge at popup right − 13 (`pr-3` + `mr-px`): x 823, 855, 887, 919 | | | M |
| header controls | not rendered in the chat header while the sheet holds them | | | M |
| open, focus | Opening moves focus into the sheet: a pointer open focused the empty state's "Open a surface" list; a keyboard open (Enter on the focused toggle) left focus on the in-sheet "Toggle right panel" | | | M |
| close | Escape, outside press (Base UI default), the in-panel right panel toggle. If the focused control's tooltip is open (it opens on keyboard focus), the first Escape only closes the tooltip and a second Escape closes the sheet | | | M |
| focus after close | Focus does not return to a toggle. The header toggle that opened the sheet unmounts while the sheet holds the controls, and the sheet host is not the collapsible inline panel, so its focus-return logic does not run. Measured after both an Escape close and a toggle close: focus lands in the composer editor | | | M |

### 3.9 Sidebar as a sheet (window < 768px)

Measured at a 700px window.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| closed | wrapper `data-sidebar-state=collapsed`, fixed trigger at 13, 12 (28×28), header left padding 52 | | | M |
| open | popup 0, 0, 688×900 (`100vw − 12px`), `border-e` 1px, z 46 viewport, no backdrop (nested dialog) | | | M |
| background | `bg-sidebar` + grain, resolved outside `[data-app-sidebar]` (the inner `[data-app-sidebar]` div has no background of its own, so this one shows) | `#fafafa` | `#111111` (root `--sidebar` = `--card`, not the scoped `#000000`); painted mean with grain about `#151515` | M |
| border | `border-e` 1px | `#e4e4e7` | `#ffffff14`, paints about `#292929` over `#111111` | M |
| shadow, `::before` | as the right sheet | | | M |
| trigger | the fixed control hides (`display: none`) while the sheet is mounted. The sheet's own header trigger shows at the same spot, 13, 12, 28×28, `media-navigation` on the art | | | M |
| title bar controls | the fixed top-right panel toggles hide (`display: none`) while the sheet is mounted | | | M |
| brand | hidden below 768 (`hidden md:flex`) | | | M |
| rail | Still rendered, because `SidebarRail` is a child of `<Sidebar>`, but nothing positions it: the `group-data-[side=left]:-right-4` and `in-data-[side=left]:cursor-w-resize` classes need a desktop ancestor that the sheet lacks. Measured at 700: box −8, 0, 16×900 (only x 0..8 is on screen), at the sheet's left edge; `display: flex` (hidden below 640); cursor `default`; `aria-label` and tooltip "Toggle Sidebar"; a click closes the sheet (`toggleSidebar`); hover line 2px at x 0..2 | | | M |
| stage art fade | The backdrop sits inside `[data-app-sidebar]`, so `--stage-fade` resolves to the scoped `#fafafa` / `#000000`. In dark that is not the sheet's `#111111`, so the art's tail ends darker than the surface. Pixels at x 20: y 56 `#0f161f`, y 64 `#0f1012`, y 68 `#111312`, y 72 `#131313`; surface ≈ `#151515`. Light matches (both `#fafafa`) | | | M |
| close | Escape (measured), outside press, the trigger, choosing a utility item (settings, usage, pull requests) | | | M / S |
| animation | `transitionDurationMs: 0` while panel animations are off; the default 200ms otherwise | | | S |

### 3.10 Terminal drawer frame

Not opened live (§7). Geometry from source, colours from probes.

| Item | Value | Light | Dark | M/S |
|---|---|---|---|---|
| place | last child of the chat column, below the canvas row, full chat column width. It pushes the canvas up | | | S |
| wrapper | `grid shrink-0 overflow-clip`, `grid-template-rows: 1fr` visible, `0fr` hidden; drawers of inactive threads `display: none` | | | S |
| frame | `aside[data-thread-terminal-drawer]`: `relative flex min-w-0 flex-col overflow-hidden bg-background z-45 shrink-0 border-t border-border/80` | | | S |
| background | | `#fcfcfc` | `#000000` | P |
| top border | 1px `border-border/80` = `--contrast-border` at 80% | `#e4e4e7cc` (computed `oklab(0.92 0.0011 −0.0038 / 0.8)` = rgb 228 228 231 at 0.8); paints about `#e9e9eb` over the drawer's own `#fcfcfc` | `#ffffff10` (white at 0.064); paints `#101010` over `#000000` | P |
| height | `var(--terminal-drawer-drag-height, <height>px)` | | | S |
| default height | 280px | | | S |
| minimum | 180px | | | S |
| maximum | `min(max(180, floor(0.75 × window height)), max(0, chat column height − header height − composer height))`. 675 at 900 tall | | | S |
| resize strip | `absolute inset-x-0 top-0 z-20 h-1.5 cursor-row-resize touch-none any-pointer-coarse:h-4`: 6px (16px coarse) | | | P |
| z-index | 45, above the timeline minimap, below sheets (46) | | | P |

### 3.11 Z-order

| Layer | z-index | Source | M/S |
|---|---|---|---|
| stage backdrop inside the sidebar | 0 | SidebarStageBackdrop.tsx:56 | M |
| sidebar search row group | 1 | Sidebar.tsx:2968 | M |
| sidebar header content | 10 | SidebarChrome.tsx:67, 74 | M |
| sidebar container (fixed) | 10 | sidebar.tsx:399 | M |
| sidebar rail | 20 | sidebar.tsx:580 | M |
| composer overlay, details card | 20 | ChatView.tsx:3521–3522 | M |
| banner lane, scroll-to-end | 30 | ChatBannerLane.tsx:44, TimelineOverlays.tsx:33 | S |
| right panel resize handle | 40 | RightPanelResizeHandle.tsx:22 | M |
| terminal drawer | 45 | ThreadTerminalDrawer.tsx:1841 | S |
| sheets | 46 (`--z-sheet`) | index.css:118 | M |
| preview mini player | 47–49 | index.css:115–117 | S |
| sidebar trigger, title bar controls | 50 | AppSidebarLayout.tsx:382, PanelLayoutControls.tsx:202 | M |
| dialogs and popovers | 50 and up; tooltips 140 | index.css:115–117, tooltip.tsx:109 | S |

### 3.12 Desktop window chrome (Electron)

| Item | macOS | Windows | Linux | Source | M/S |
|---|---|---|---|---|---|
| `titleBarStyle` | `hiddenInset` | `hidden` | `hidden` | DesktopWindow.ts:320–342 | S |
| native controls | traffic lights at `trafficLightPosition {x: 16, y: 19}` (19 = 52/2 − 7) | overlay caption buttons, `titleBarOverlay {color: "#01000000", height: 40, symbolColor}`. Electron reads 8-digit hex as `#AARRGGBB`, so this is black at alpha 1/255, effectively transparent (fully transparent `#00000000` misbehaves on Linux, per the source comment) | same as Windows | DesktopWindow.ts:46, 56–58, 86–88 | S |
| caption symbol colour | n/a | `#1f2937` light, `#f8fafc` dark | same | DesktopWindow.ts:87–88, 339 | S |
| top bar height | 52px | `max(40px, env(titlebar-area-height))` = 40px | 40px | index.css:123, 156 | S |
| controls left (`--workspace-controls-left`) | `var(--desktop-window-controls-inset, 90px)`, where the preload sets the inset to `90 / zoomFactor` px; 12px in fullscreen | `env(titlebar-area-x) + 12px` = 12px | 12px | AppSidebarLayout.tsx:69, 446–450; preload.ts:49–59; index.css:158 | S |
| controls right (`--workspace-controls-right`) | 12px | `100vw − titlebar-area-width − titlebar-area-x + 12px`: caption buttons' width + 12 | same | index.css:159–161 | S |
| `--workspace-native-controls-inset` | 0 | same formula as controls right | same | index.css:162–164 | S |
| `--workspace-titlebar-content-left` | 90 + 28 + 12 = 130px | 52px | 52px | sidebar.tsx:245–246 | S |
| right resize lane | none | `#root` `padding-right: 6px` (`.electron-windows`) | none | index.css:1754–1759 | S |
| `backgroundColor` | `#ffffff` light, `#000000` dark (`nativeTheme.shouldUseDarkColors`), re-applied on appearance change | same | same | DesktopWindow.ts:164–166, 344–360, 483 | S |
| default size | 1100×780 | same | same | DesktopAppSettings.ts:64–67 | S |
| minimum size | 400×360 | same | same (on Wayland the client-side frame adds 32×42) | DesktopAppSettings.ts:50–56 | S |
| vibrancy, transparency, rounded corners | none set | none | none | DesktopWindow.ts:476–499 | S |
| `autoHideMenuBar` | true | true | true | DesktopWindow.ts:481 | S |
| `disableAutoHideCursor` | true (also `setAutoHideCursor(false)`) | n/a | n/a | DesktopWindow.ts:482, 507–509 | S |
| icon | from the bundle | `.ico` | `.png` | DesktopWindow.ts:152–162 | S |
| first paint | `show: false`; revealed on `ready-to-show` | same | also on `did-finish-load` | DesktopWindow.ts:966–991 | S |
| window title | `title: displayName`; afterwards `document.title` (§5.9) | same | same | DesktopWindow.ts:485, 818–824 | S |
| tooltips near the title bar | Tooltip collision padding at the top is 5px + the native title bar height: 52 on macOS desktop, the overlay rect's bottom (40) under WCO, 0 in a browser. Tooltips never overlap the title bar band | same | same | tooltip.tsx:16, 105–115; windowControlsOverlay.ts:70–78 | S |
| connecting splash | n/a | Shown only in WSL-only mode (`wslOnly && wslBackendEnabled`), before the backend starts: a separate frameless, non-resizable, centred 360×220 window, background `#ffffff` / `#000000`. Its whole body is a drag region, a flex column centred with an 18px gap: a 26px spinner (3px ring in `rgba(31,41,55,0.18)` / `rgba(248,250,252,0.18)`, top segment `#1f2937` / `#f8fafc`, 0.8s linear spin) over a 13px `system-ui` label "Connecting to WSL…" in `#6b7280` / `#9ca3af`. Closed when the main window first reveals | n/a | DesktopWindow.ts:263–270, 985, 1051–1100; DesktopApp.ts:279–281 | S |

## 4. States

| Region | State | Trigger / attribute | Values | M/S |
|---|---|---|---|---|
| sidebar | expanded | `data-sidebar-state=expanded`, `data-state=expanded`, `data-collapsible=""` | gap width = `--sidebar-width`, container `left: 0`, trigger icon `PanelLeftClose`, `aria-pressed=true` | M |
| sidebar | collapsed | `data-state=collapsed`, `data-collapsible=offcanvas`, sidebar root `inert` | gap 0, container `left: -256px` (still rendered, off screen), `main` 0..1440, header `padding-left: 52px`, trigger ghost with `PanelLeft`, rail `pointer-events: none` | M |
| sidebar | resizing | pointer captured on the rail | gap and container inline width (clamped live 208..800 at 1440), `--sidebar-width` unchanged until release, `main` reflows live | M |
| sidebar | mobile closed | < 768, `openMobile=false` | wrapper `collapsed`; the sheet popup stays mounted but closed, so window shortcuts registered by its children keep working | M |
| sidebar | mobile open | sheet `data-open` | §3.9; fixed trigger hidden | M |
| sidebar | welcome page | `showSidebar=false` | no sidebar, no trigger, shortcut inert | S |
| trigger | default / hover / pressed / focus-visible | | §3.3 | M |
| rail | hover | `:hover` | 2px line `#e4e4e7` / `#ffffff14` at x 255..257 | M |
| rail | collapsed | | moved to `right: -8px`, `pointer-events: none`, cursor `e-resize` | M |
| header | collapsed sidebar or < 768 | `[data-sidebar-state=collapsed]` ancestor | `padding-left: var(--workspace-titlebar-content-left)` | M |
| header | inline right panel open | `inlineRightPanelOwnsTitleBar` | header covers only the chat column; ChatHeader `pr-10`; details toggle inside the header | M |
| canvas | details card inline | default, canvas ≥ 1012 wide and ≥ 184 tall | lanes shift left to clear the card (§5.4) | M |
| canvas | details card hidden | toggled off | lanes centred | M |
| canvas | details card popover | canvas too narrow for the inline card (< 1012 wide or < 184 tall), which includes 1440 with the inline right panel open (canvas 644) | no inline card; the header toggle opens it as a popover anchored to the chat header's bottom-right (§3.6); lanes centred | M |
| composer | draft (new thread) | `isDraftHeroState` | overlay `inset: 0`, composer vertically centred; headline above it | M |
| composer | thread | | overlay pinned to the bottom | M |
| timeline | top fade off | top banner shown | no mask | S |
| title bar controls | right panel closed | | maximize control `opacity: 0`, out of flow, inert | M |
| title bar controls | right panel open | | maximize control in flow, `opacity: 1` | M |
| title bar controls | mobile sidebar sheet mounted | < 768, sheet open or closing | whole group and the header details toggle `display: none` | M |
| toggles | pressed / disabled / focus-visible | | §3.6 | M / P |
| right panel | closed | `isOpen=false` | not mounted after the exit delay (`usePanelPresence`) | M |
| right panel | open inline | | §3.7 | M |
| right panel | maximized | `maximized=true` | §3.7 | M |
| right panel | sheet | window ≤ 980, not maximized | §3.8 | M |
| right panel | empty | no surfaces | "Open a surface" list | M |
| resize handle | hover / active | `group-hover`, `group-active` | `#e4e4e7`/`#ffffff14`; `#1b4ed899`/`#346bf199` | M / P |
| terminal drawer | open / closed / empty | per-thread `terminalOpen`; no sessions shows "No terminal sessions for this thread yet." with a button | §3.10 | S |
| workspace area | keyboard focus | `data-workspace-keyboard-focus` on an area that took focus itself | `outline: 2px solid var(--ring)` (`#1b4ed8` / `#346bf1`), `outline-offset: -2px`. Painted on the conversation area: 2px band at x 256..258 and y 52..54 | M |
| window | loading | before React mounts | boot splash: centred 64px icon on `#ffffff` / `#000000` | S |
| window | macOS fullscreen | `enter-full-screen` IPC | controls inset 12px instead of 90px (no traffic lights) | S |
| window | `.wco` | overlay visible (not in fullscreen) | 40px top bar, caption-button reserve on the right | S |
| page | command palette open | `inert` on the shell wrapper | whole shell inert | S |

## 5. Behaviour

### 5.1 Sidebar open state

- State lives in a small store outside React, so shortcuts read and write it at event time (sidebar.tsx:143–187). `open` starts from localStorage key `chat_thread_sidebar_open` (Boolean) or `true`. `openMobile` always starts `false` and is never persisted.
- `toggle()` checks the viewport at call time: below 768px it flips `openMobile`, otherwise `open` (sidebar.tsx:179).
- `setOpen(false)` while open first runs collapse listeners (an active rail drag cancels), then moves focus out of the sidebar if it held focus, to the trigger outside it, because the collapsed sidebar becomes `inert` (sidebar.tsx:125–136, 155–159).
- Every `setOpen` writes `chat_thread_sidebar_open` (measured: `"true"` / `"false"`). On desktop widths a real change announces "Sidebar shown" or "Sidebar hidden" through two alternating `aria-live="polite"` regions in `#workspace-visibility-alerts`. The very first announcement waits 100ms (accessibilityAnnouncements.ts).
- `data-sidebar-state` on the wrapper mirrors the rendered state. Other key handlers read it there (AppSidebarLayout.tsx:298–300).
- Ways to toggle: the trigger button, the `sidebar.toggle` shortcut, the desktop menu action `sidebar.toggle` (a focused preview guest forwards the shortcut that way, preview/Manager.ts:1540–1571), and a click on the rail when the sidebar cannot be resized (mobile).

### 5.2 Sidebar width

- Initial width: stored `chat_thread_sidebar_width` raised to at least 208, or 256 if absent, then capped at `max(208, floor(window width) − 640)` (threadSidebarWidth.ts:25–34).
- Render width: `clamp(width, min, max)` with min from the brand probe and max from the live window width (`resize` listener), so a narrower window shrinks the sidebar without a drag (AppSidebarLayout.tsx:428–455).
- Drag, from the shared engine (useResizeDrag.ts):
  1. Primary button only. `pointerdown` captures the pointer, starts at the container's current width, sets `body` cursor and `user-select`, listens for window `blur`.
  2. `pointermove` stores the x. One `requestAnimationFrame` applies `width = clamp(start + dx, min, max)`; `moved` turns true once |dx| > 2.
  3. The sidebar session writes widths only to the gap and container (inline style, transitions off). It refuses a growing width that would leave `main` under 640px (`shouldAcceptWidth`).
  4. `pointerup` applies the release x and commits. `pointercancel`, `lostpointercapture` and window blur also commit, at the x of the last `pointermove`.
  5. Commit sets `--sidebar-width` on the wrapper, writes `chat_thread_sidebar_width` (even when the pointer did not move) and updates React state. Cleanup removes the inline widths.
  6. Collapsing mid-drag cancels without committing. Unmounting the rail cancels.
- Clicks on the rail: suppressed after a drag that moved; ignored while the sidebar is resizable and open; otherwise they toggle.
- Double-click on the rail resets: removes `chat_thread_sidebar_width` and returns to `min(256, max(208, floor(window width) − 640))` (measured: key removed, width 256).
- Measured clamps at 1440: dragging to x 100 gives 208, to x 1100 gives 800.

### 5.3 Title bar insets

- `--workspace-titlebar-content-left` = controls left + control size + 12 = where content may start to the right of the fixed trigger: 52px in the browser, 56px below 640, 130px on macOS desktop.
- The chat header, page headers (`WorkspacePageHeader`) and a maximized right panel's tab bar take that left padding whenever the sidebar is collapsed, and always below 768px.
- The sidebar brand always starts at that x, inside the sidebar.
- The fixed title bar controls need 92 + 13 px at the right. The chat header reserves it with `pr-24` on its content. The inline right panel's tab bar reserves it with `pr-28`. Under `.wco` the header's own right padding becomes `--workspace-native-controls-inset` while its content keeps `pr-24`. The tab bar's `pr-28` becomes that inset, plus a 96px spacer (`hidden w-24 shrink-0 wco:block`) when the bar has no in-panel controls.

### 5.4 Chat lanes and chat width

The canvas measures itself and a hidden width probe (`w-(--chat-content-max-width) min-w-[40rem] box-content ps-3 sm:ps-12`), then places the lane (chatCanvasLayout.ts:24–55):

```
padding      = probe padding-left               (48, or 12 below 640px)
maxChat      = probe width                      (736 comfortable, 1152 wide, canvas width for full)
minChat      = probe min-width                  (640)
centred      = clamp(container.width − 2·padding, 0, maxChat)
laneRight    = detailsCard ? detailsCard.left − 32 : container.width − padding
width        = max(0, min(centred, laneRight − padding))
left         = max(padding, min((container.width − width) / 2, laneRight − width))
insetEnd     = max(0, container.width − 2·left − width)       → --chat-lane-inset-end
```

The lanes then use `padding-inline: 48px` and `48 + insetEnd`. The timeline subtracts its 6px scrollbar gutter. Rows and the composer stack centre a `max-width: --chat-content-max-width` box in that space. A preview mini player can shrink or move the lane further (preview doc).

The details card is placed at `x = canvas width − 292`, `y = 12 + find bar inset`. It shows inline only if `x − 32 − padding ≥ 640` (canvas ≥ 1012px wide) and its height ≥ 160. Otherwise the header toggle opens it as a popover. The inline card is open by default (`inlineOpen: true`), the popover closed.

`data-chat-width` on `<html>` comes from the `chatWidth` setting (`comfortable` default, `wide`, `full`) and only changes `--chat-content-max-width`.

### 5.5 Right panel

- Open state, active surface, surfaces and `maximized` are per thread in `t3code:right-panel-state:v2` (zustand persist, version 14). Opening with no surfaces shows the empty list. Closing through the toggle leaves no entry behind for that thread.
- Presence: the panel mounts when open. With panel animations active it stays mounted for `durationMs` after closing, so the width can animate to 0 (panelAnimations.ts:41–69).
- Inline vs sheet: sheet when `(max-width: 980px)` matches and the panel is not maximized.
- Width: per thread, key `t3code:preview-panel-width:<environment>:<thread>`, default 540, clamped to [360, max] on every render. Max is recomputed on window resize and when the layout row resizes (once per frame), and a drag in progress re-applies its pointer position when bounds change.
- Drag: same engine, `edge: "left"`, so moving the pointer left widens. During the drag only the host's `--resizable-width` changes and `transition-property` is `none`. Release commits and persists only if the pointer moved more than 2px; otherwise the width is restored. `pointercancel`, `lostpointercapture` and window blur also end the drag as a release, at the last `pointermove` x. Only an abandoned drag (the handle unmounts mid-drag) puts the start width back without committing (useResizeDrag.ts:66–75, 123–125; useResizableWidth.ts:89–96). Measured clamps at 1440 with the sidebar: 360 and 824; re-derived: `max(360, min(floor(0.7 × 1440) = 1008, 1184 − 360 = 824))` = 824.
- Width animation runs only for open and close. Any other width change sets `transition-duration: 0ms` in the same render.
- Focus: entering the panel records where focus came from. Collapsing the inline panel while it holds focus returns focus there if that element is still visible and in view, else to `[data-right-panel-toggle]`. The host gets `inert` and `pointer-events: none` while collapsed. This only applies to the collapsible inline host. The sheet has no such logic, so after a sheet closes, workspace focus recovery puts focus in the composer editor (measured, §3.8).
- Maximize: `rightPanel.toggleMaximized` (no default key) or the maximize button. It moves focus into the panel when maximizing.
- Controls placement: inline panel present means the fixed controls render at the ChatView root and the details toggle moves into the chat header. Sheet open means all four controls render inside the sheet's tab bar and none in the header.

### 5.6 Terminal drawer

- One drawer per thread with open terminals stays mounted (up to 10 hidden threads plus the active one), so switching threads keeps sessions (ChatTerminalDrawers.tsx, ChatView.logic.ts:108).
- Height per thread in `t3code:terminal-state:v1`, default 280. Render height = `clamp(round(height), 180, max)`. When max < 180, max wins.
- Drag on the 6px top strip: height = start + (start y − pointer y), clamped, written to `--terminal-drawer-drag-height` on the frame. Release commits to the store only if the height changed. Window and column resizes during a drag re-clamp it.
- Opening the drawer on a thread with no terminal starts a shell session on the server (useChatTerminalActions.ts:143–170).

### 5.7 Keyboard and focus

Default shell shortcuts (packages/shared/src/keybindings.ts:57–68). `mod` is Cmd on macOS, Ctrl elsewhere. Users can rebind every one through the server's keybindings.

| Command | Default | Condition | Effect |
|---|---|---|---|
| `sidebar.toggle` | mod+B | not inside `[data-keybinding-capture]`; not Mod+B with a non-collapsed selection inside `[data-composer-rich-text="true"]` (that is bold) | toggle sidebar, or the mobile sheet below 768 |
| `workspace.focusNext` / `focusPrevious` | F6 / Shift+F6, also mod+F6 / mod+Shift+F6 | capture phase, so it also leaves the terminal; refused inside a modal | move focus to the next visible workspace area |
| `terminal.toggle` | mod+J | | toggle the drawer |
| `rightPanel.toggle` | mod+Alt+B | | toggle the right panel |
| `threadPanel.toggle` | mod+Shift+B | `!terminalFocus` | toggle the details card |
| `navigation.back` / `forward` | mod+[ / mod+] | `!terminalFocus` | history |
| Escape | | a sheet holds focus | close the sheet. An open tooltip on the focused control takes the first Escape, so a keyboard-focused toggle needs two (measured) |

- The sidebar shortcut handler listens on `window` keydown in the capture phase, ignores `defaultPrevented` events, and calls `preventDefault` + `stopPropagation` when it acts (AppSidebarLayout.tsx:329–369).
- Shortcut labels: macOS uses symbols in the order ⌃⌥⇧⌘ then the key ("⌘B"); other platforms "Ctrl+Alt+Shift+Meta+Key" ("Ctrl+B") (keybindings.ts:228–250). Tooltips append " (<label>)". The measured instance had a user override for these commands, so its labels differed from the defaults.
- F6 areas, in order: `sidebar` 0, `conversation` 1, `composer` 2, `details` 3, `panel` 4, `terminal` 5. Other routes use the same slots: `home` 1 (home, welcome, no-projects hero), `pull-requests` 1, `settings-nav` 0 and `settings` 1 (workspaceAreaProps call sites). Each is `role="region"`, `tabindex="-1"` with an `aria-label` ("Threads", "Conversation", "Message input", "Thread details", "Workspace panel…", "Terminal"). Hidden, inert, collapsed, `data-workspace-unavailable` or maximized-away areas are skipped. The cycle wraps. With focus on the body, F6 continues after the area that last held focus; with no history it starts at the first area (Shift+F6 at the last) (workspaceFocus.ts:229–272). Entering an area restores its last focused element (or, if that element was unmounted, a remounted control with the same id, `aria-label` or button text), else a preferred one (current sidebar row, composer editor, terminal textarea), else the first tabbable control of the right panel's surface, else the first tabbable element, else the area itself (workspaceFocus.ts:136–200). The conversation skips the fallback and only reuses a remembered control that is still scrolled into view. Measured cycle (from the body, last area = conversation): composer editor, first details button, current sidebar row, conversation (the area itself, 2px ring), composer editor.
- Focus that drops to `body` because its element went away returns to a remounted twin, else its area (composer for the conversation), else the main content.

### 5.8 Drag regions and double-click (desktop)

- `.drag-region { -webkit-app-region: drag }`. Its direct children reset to `initial`, which adds no rectangle of its own (Electron 44 inherits the property and `none` would act as `no-drag`). `button, input, textarea, select, a` inside a drag region are `no-drag` (index.css:1827–1848).
- Drag regions, each added only when `isElectron`: sidebar header row (SidebarChrome.tsx:63), chat header (ChatView.tsx:3320), inline right panel tab bar (RightPanelTabs.tsx:1353), page headers (WorkspacePageHeader.tsx:19), diff panel header row (DiffPanelShell.tsx:16). The route modal's title strip below 640px carries `[-webkit-app-region:drag]` unconditionally, which only matters in Electron (AppRouteModal.tsx:77).
- Explicit `no-drag`: the sidebar trigger, history buttons, panel toggles and their groups, the maximize control, the details toggle in the header, right panel tabs and tab scroll buttons, dialogs, menus, popovers, selects, comboboxes, autocompletes.
- App-region hit testing ignores z-index. Fixed controls that overlap a header are not inside that header's DOM, so the header's drag rectangle would swallow them. ChatView and RightPanelTabs therefore add a `pointer-events-none fixed … w-28 [-webkit-app-region:no-drag]` span (112×topbar at the top-right) whenever the fixed controls sit over their row.
- Double-click: T3 handles it only on the sidebar rail (reset width) and on the thread title in the chat header (rename, chat header doc). A double-click elsewhere in a drag region gets the platform's title bar action from Electron (macOS: the "double-click a window's title bar" preference, zoom by default; Windows: maximize/restore; Linux: the window manager's action).

### 5.9 Window lifecycle (desktop)

- Create: restore saved bounds if they fit inside some display (±1px), else 1100×780 centred by the OS. Restore maximized state at first reveal (DesktopWindow.ts:250–261, 983).
- Persist bounds 500ms after `resize`, `move`, `maximize`, `unmaximize`, and immediately on `close`. Maximized or fullscreen windows save their normal bounds, moved onto the window's display if needed. Minimized windows keep the last good state (DesktopWindow.ts:510–613, 825–831).
- Appearance: on theme change the window's background colour and, off macOS, the overlay's symbol colour are re-applied (DesktopWindow.ts:344–360).
- Zoom: View menu Cmd/Ctrl+0, +, −, numpad keys; Ctrl+wheel. Steps of 0.5 zoom level. On macOS each zoom re-centres the traffic lights at `y = round(52 × zoom / 2 − 7)`, and the preload sets `--desktop-window-controls-inset: 90 / zoom px`, so the reserved width stays 90 native points (DesktopWindow.ts:75–84, 429–440; preload.ts:49–59).
- macOS fullscreen: `enter-full-screen` / `leave-full-screen` are sent to the renderer; the 90px inset applies only outside fullscreen. Leaving re-syncs the traffic lights (DesktopWindow.ts:840–846).
- Title: the renderer sets `document.title`; Electron uses it unless it is blank (then the app name). Format: desktop `Project / Thread`, browser `Project / Thread - T3 Code (Nightly)`; with no thread, the app display name. With the screen-reader title setting: `Status: Thread - Project` (branding.logic.ts:59–82).
- Cmd/Ctrl+W reaches the renderer (closes a tab or terminal, not the window); held repeats are swallowed (DesktopWindow.ts:787–803).
- macOS swipe events and, in the renderer, a horizontal trackpad swipe over `main` (120px threshold) navigate history (DesktopWindow.ts:833–839, HistorySwipeNavigation.tsx).

### 5.10 Stage artwork selection

- Stage label: "Nightly" when the primary server version matches `/^[^-+]+-(?:nightly|preview)\.\d{8}\.\d+$/`, else the build's own label (branding.logic.ts:1, 29–37). This server's version matches.
- With art shown, the sidebar trigger (while the sidebar is visible), the history buttons and the brand switch to their on-art colours (white).

## 6. Motion

Panel animations are off by default (`panelAnimationDurationMs = 0`, range 0 to 400). They run only while the wrapper has `data-panel-animations="true"`: duration > 0, no `prefers-reduced-motion: reduce`, and not within two frames of a route change (panelAnimations.ts:11–39). Ease-out is Tailwind's `cubic-bezier(0, 0, 0.2, 1)`.

| Item | Property | Duration | Easing | Gate | Source | M/S |
|---|---|---|---|---|---|---|
| sidebar gap | `width` | `--panel-animation-duration` | ease-out | panel animations | sidebar.tsx:388 | S |
| sidebar container | `left, right, width` | same | ease-out | same | sidebar.tsx:400 | S |
| sidebar rail | `all` | same | ease-out | same | sidebar.tsx:581 | S |
| chat header | `padding-left` | same | ease-out | + `motion-safe` | ChatView.tsx:3317 | S |
| page header | `padding-left, padding-right` | same | ease-out | + `motion-safe` | WorkspacePageHeader.tsx:18 | S |
| right panel host | `width`, from 0 on open (`@starting-style`), to 0 on close, then unmount after the duration | same | ease-out | collapsible inline panel | PreviewPanelShell.tsx:169–185 | S |
| maximize control | `opacity` 0 → 1, on open only (closing hides it at once) | same | ease-out | + `motion-safe`, animations active | PanelLayoutControls.tsx:211–219 | S |
| terminal drawer | `grid-template-rows` 0fr ↔ 1fr, from 0fr on open | same | ease-out | active drawer | PersistentThreadTerminals.tsx:243–249 | S |
| right panel sheet | `opacity` 0 → 1, `translate` 32px → 0 (enter); reverse on exit | `transitionDurationMs` = panel duration when active, else 0 (instant classes) | ease-in-out `cubic-bezier(0.4, 0, 0.2, 1)` | | sheet.tsx:93–105, ChatView.tsx:3959 | M (instant) / S |
| mobile sidebar sheet | `opacity`, `translate` −32px → 0 | 200ms when animations active, else 0 | ease-in-out | | sidebar.tsx:296–301, 337 | S |
| sheet backdrop | `opacity` | 200ms | `transition-all` | never rendered in the shell | sheet.tsx:22 | M |
| right panel resize line | `background-color` | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | always | RightPanelResizeHandle.tsx:29 | M |
| sidebar trigger | `box-shadow`, `scale` (0.97 while pressed) | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | always | button.tsx:9 | M |
| panel toggles | `box-shadow` only; background changes instantly | 150ms | same | always | toggle.tsx:7 | M |
| drags | all of the above transitions disabled while a drag runs | | | | sidebar.tsx:497–499, useResizableWidth.ts:76 | M |
| theme change | every transition and animation 0s for one frame (`.no-transitions`) | | | | index.css:1104–1111 | S |
| top fade, stage art | static | | | | | M |

## 7. Gaps

- **Electron-only behaviour** could not be measured in a browser: `.wco` and its `env(titlebar-area-*)` values, the 40px top bar, traffic-light placement and the 90pt inset, drag regions, the Windows 6px resize lane, fullscreen insets, zoom. All desktop values are source-derived. The caption button width under WCO depends on the OS (on Windows 11 it is three 46px buttons at 40px height; not verified).
- **Double-click on drag regions** is Electron and OS behaviour, not T3 code. The per-platform actions in §5.8 are not verified.
- **Panel animations** are off by default and turning them on means changing a setting, so every non-zero duration in §6 is source-derived. Sheets were measured in their instant form.
- **Terminal drawer** was not opened: on a thread without terminals the toggle starts a shell on the server. Geometry is source-derived; border and background colours come from probes.
- **Dev blueprint art** is not shown on a nightly server. Its geometry is from source; its palette was resolved live.
- **Fonts.** The brand probe, and so the minimum sidebar width, depends on font metrics. Inter gives 120.05 here. On macOS with SF Pro the brand mark measures differently by a few px, which can move the macOS desktop minimum (259 here) by a pixel or two.
- **Keybinding labels.** The measured server has user overrides for the shell shortcuts. Defaults are from source.
- **Hover in screenshots.** Synthetic pointer moves produce computed hover styles, but screenshots did not show them, so hover colours are computed values, not pixels.
- **Touch** variants (`any-pointer-coarse`, `pointer-coarse`, 44px hit targets) were not measured.
- **Safe-area insets** are 0 in this browser, so the `env(safe-area-inset-*)` terms resolve to nothing.
- **Disabled panel toggles** could not be found live (every thread had a project), so the disabled style is a probe (a disabled clone of the live toggle, both themes).
- **Tooltip visuals** and timing belong to the tooltip doc. Only their text and anchor points are recorded here.
- **Popover mode of the details card** is reached at 1440 by opening the inline right panel (canvas 644). Its anchor and box are recorded in §3.6; the card's own styling belongs to the details doc.
- **Background tab during verification.** The verification browser's window was covered by other windows, so Chrome reported `visibilityState: hidden`. rAF, ResizeObserver and CSS transitions only advanced when a screenshot forced a frame. Layout that depends on them (chat lanes, brand probe, right panel max width) was re-measured after forcing frames with screenshots. Synthetic hover and press did not always register, so the light resize-handle hover, the ghost-trigger dark hover and the handle's active colour are probes or computed values, not live states. Drags were not repeated (they need rAF); clamps were re-derived from source.
- **Wide gamut.** The stage night palette and the primary blue are outside sRGB. Values are Chrome's sRGB results; a P3 Mac paints them more saturated (01-tokens §7).

## 8. Port notes for Go/Gio

1. **Root layout.** Lay out a horizontal flex: a spacer of `sidebarWidth` (0 when collapsed) and `main`. Paint the sidebar separately at `x = expanded ? 0 : -width`, width `sidebarWidth`, above `main` (z 10). Its right 1px is the border (`#e4e4e7` / `#ffffff14` over the surface).
2. **Sidebar width.** Default 256. Min `max(208, ceil(contentLeft + brandMarkWidth + 4 + 2 × controlSize + 13))` while the back/forward buttons are shown, as they are in the desktop app; `contentLeft` is 130 on macOS and 52 elsewhere. Max `max(min, floor(windowWidth) − 640)`. Re-clamp on every window resize. Persist the dragged width and the open flag. Double-click on the rail clears the width.
3. **Rail.** A 16px hit area centred on the border's x, full height, cursor `w-resize`. On hover draw a 2px line from the border's x to x + 2. Drag with pointer capture, apply at most once per frame, clamp, refuse growth that leaves `main` under 640, commit on release, cancel on collapse. Hide it below 640 and while collapsed.
4. **Fixed controls.** The sidebar trigger is window-level at (controlsLeft + 1, (topbar − size) / 2), 28px (32 below 640). The panel toggles are window-level at the top-right, right edge `windowWidth − controlsRight − 1`, 4px apart. Draw them after everything else in the shell.
5. **Top bar.** 52px everywhere tern draws its own title bar. On Windows and Linux with native caption buttons in the same band, use the caption height (T3 uses 40) and reserve the caption width plus 12px on the right of every top row.
6. **macOS chrome.** Keep the native traffic lights, hide the title, extend content under the title bar, and place the buttons at (16, 19) pt with the top bar at 52pt; reserve 90pt on the left (scale it with tern's zoom the way T3 divides by the zoom factor). In fullscreen drop the reserve to 12. If the toolkit cannot move the buttons, measure where they land and derive the inset from that.
7. **Windows/Linux chrome.** Frameless window with tern-drawn or native-overlay caption buttons in a 40px band, symbol colour `#1f2937` / `#f8fafc`, transparent background. On Windows leave a 6px strip on the right edge for the resize border; it shows the body, which is `#fcfcfc` with grain in light and flat `#000000` in dark. On every desktop platform, keep tooltips below the title bar band (top collision padding 5 + band height).
8. **Drag regions.** Mark the sidebar header row, chat header, inline right panel tab bar and page headers as window-move areas (Gio `system.ActionMove`), minus every control rectangle. Check whether a double click in a move area reaches the platform's title bar action. If it does not, implement it: zoom or minimize per the macOS preference, toggle maximize on Windows and Linux.
9. **Chat canvas.** Implement §5.4 literally, in that order, with padding 48 (12 below 640), max 736/1152/full, min 640, card clearance 32, card 280 wide at 12px from the top-right. The timeline and composer share the lane. Keep the composer as an overlay at the canvas bottom: 8px top padding, 20px bottom spacer.
10. **Top fade.** Multiply the timeline's alpha by the 25px stepped ramp (0, 0.10, 0.30, 0.58, 0.82, 0.96, 1 at 0, 2.5, 6, 10.5, 15.5, 20.5, 25px), except the rightmost 6px. The area behind the timeline is the conversation area's solid background, so painting that colour over the band with alpha `1 − ramp` gives the same pixels as a mask. Gio's linear gradient takes two stops, so draw the ramp as six bands.
11. **Stage art.** Gio has no SVG, no radial gradient, no pattern fill, and linear gradients with only two stops. Rasterize the art offline (resvg or similar) from the source SVG at 1x, 2x and 3x for 960×96 user units (= 800×80 dp), for the nightly palette and for the dev palette in each theme. Bake the fade in as well: blend each row toward the theme's sidebar colour by `1 − eff(y)` (§3.4), one image per theme. At runtime draw it from the sidebar's top-left, clipped to the sidebar width and 80dp height, under the header and search row.
12. **Right panel.** Inline above 980px window width: default 540, min 360, max `max(360, min(floor(0.7 × window), floor(layoutWidth) − 360))`, persisted per thread, commit on release only after > 2px of movement. 4px handle inside the left border, 1px line on hover (`border`) and drag (`primary` at 60%: `#1b4ed899` / `#346bf199`). Maximize hides the chat column entirely and drops the border. At ≤ 980px show it as a right sheet, `min(42vw, 448)` wide (min 320), no dimming backdrop, shadow `0 10 15 -3 #0000000d` + `0 4 6 -4 #0000000d`, closed by Escape or an outside click. Paint the sheet's surface in the background colour (`#fcfcfc` / `#000000`), not `popover`, and its 1px left edge `#e4e4e7` / `#242424`. After it closes, focus the composer. With the inline panel open at the default 1440 width, the details card must switch to popover mode, anchored to the chat header's bottom-right with a 4px inset.
13. **Mobile sheet.** Below 768px the sidebar is a left sheet `windowWidth − 12` wide. Its dark background is `#111111`, not the desktop sidebar's `#000000`, but its stage art still fades to `#000000`, so the art's lower edge is slightly darker than the surface in dark. While it is mounted, hide both the fixed trigger and the fixed top-right panel toggles. To match T3 exactly, keep the stray rail: an 8px "Toggle Sidebar" strip at the sheet's left window edge (x 0..8, hidden below 640) that closes the sheet on click.
14. **Terminal drawer.** Bottom of the chat column, height clamp [180, min(0.75 × window height, column − header − composer)], default 280 per thread, 6px top resize strip, 1px top border `#e4e4e7cc` / `#ffffff10` (painted `#e9e9eb` / `#101010` over the drawer's own background).
15. **Focus.** Implement the F6 cycle over the six areas in order, skipping hidden ones, with per-area focus memory. From an unfocused window it continues after the last focused area. When an area itself takes keyboard focus draw a 2px inset ring in `--ring`. Collapsing the sidebar while it holds focus moves focus to the trigger. Collapsing the inline right panel returns focus to where it came from if that is still visible and in view, else to the right panel toggle. Closing the right sheet focuses the composer.
16. **Motion.** Ship with panel animations off. If tern offers the setting, animate widths and the drawer's height with ease-out over the chosen duration, and skip animation for two frames after switching threads.

## Verification

I re-checked this document against the same build, assuming it had errors. Browser: 1440×900 @2x Linux Chrome, both themes, plus 960, 700 and 600px wide. Pages: one draft view and one existing thread. I reverted every state change: re-expanded the sidebar and removed its open flag, closed the right panel and both sheets, restored the viewport, theme and `data-chat-width`. A key-by-key comparison of localStorage against a snapshot taken before the run matched exactly. I typed, sent and changed nothing in the account.

### What was checked

Re-measured live, about 160 values, in light and dark unless a line says otherwise:

- Window: `html` and `body` inline backgrounds, `color-scheme`, `theme-color` meta, `body` size, overflow and background image, `#root` padding and overflow.
- Sidebar wrapper inline variables.
- Sidebar: container box, border colour, `left` when collapsed; surface colour and grain tile; gap width; header row, brand row, brand link, brand mark metrics and colours; search group, thread list, footer and footer row boxes.
- Stage backdrop: box, mask, the raw computed `::after` stops, `--stage-fade`, svg viewBox and framing, all 21 palette tokens.
- Stage art pixels: all 44 cells of the §3.4 table re-sampled; every one matched exactly.
- Grain averages: sidebar `#f9f9f9` / `#040404`. Chat area exactly `#fcfcfc` / `#000000`.
- Rail, expanded and collapsed: box, `right`, translate, cursor, label, `pointer-events`, `::after` position; hover line in both themes.
- Sidebar trigger. On the art: default, hover and settled focus ring. Ghost: default and focus ring in both themes; hover live in light, and in dark through the `[data-pressed]` selector that shares the hover rule.
- Chat column; header padding expanded, collapsed and below 640; header content gap and right reserve; canvas, conversation area, timeline padding, gutter, `clientWidth` and mask.
- Composer overlay in thread and draft views, lane padding, stack box, shell and spacer heights; details card box.
- Lanes for comfortable, wide and full, set by writing `data-chat-width` directly, not through settings.
- Title bar controls: group and button boxes; default, pressed, hover and focus-visible colours in both themes; disabled probe in both themes; hidden maximize control.
- Inline right panel: host box, background and border; resize handle and line; tab bar box and padding; surface box; empty-state padding; control positions; composer lane; details popover geometry.
- Right panel sheet at 960: viewport, popup box, background, border, shadow, `::before`, inner host, tab bar, in-sheet controls; edge and surface pixels; open and close focus; Escape behaviour.
- Mobile sidebar sheet at 700: closed and open states, popup box, background, border, shadow, trigger, brand, rail, title bar controls; art pixels in dark.
- 600px layout values; F6 cycle; the sidebar live region text; trigger and rail tooltip boxes.
- Terminal drawer frame classes, on a probe element.

Re-derived from source, more than 30 values: sidebar min, max, initial and reset widths, and the brand probe sums for browser, macOS and Windows/Linux. Rail and handle hit boxes. Lane formula results: 380, 480, 352, 304×548, 304×812, 304×1088. Details card inline thresholds (1012 wide, 184 tall). Right panel max 824. Drawer max 675. Mask and `::after` stops in px and the `eff(y)` table. SVG slice scale. Default shortcuts and label format. WCO variables. Traffic-light position and zoom inset. `titleBarOverlay`, background colours, default and minimum sizes. Persistence keys. Drag-engine commit and cancel paths. Panel animation gating and easing tokens. All 13 shipped-CSS byte offsets. Every z-index citation.

Privacy: searched the document for thread titles, project and branch names, host names, paths, addresses and keybinding overrides. None found.

### What was wrong, and the fix

1. Shipped-CSS offset of `[data-workspace-keyboard-focus]:focus` is 530696, not 530697.
2. Resize-handle active colour `#1b4ed999` / `#346bf299`, and drawer top border `#e3e3e7cc`, were canvas premultiplied-alpha rounding artefacts. Computed values: `#1b4ed899` / `#346bf199` (primary at 0.6) and `#e4e4e7cc`. Fixed in §3.7, §3.10, §4 and port notes 12 and 14, with painted colours added.
3. §3.8 gave the right sheet's background as `#ffffff` / `#111111`. That is only the popup's computed colour; the inner host covers it. Visible pixels are `#fcfcfc` / `#000000`, and the dark 1px edge paints `#242424`, because dark is not `bg-clip-padding`. Fixed, and the sheet border and hairline moved from probe to measured.
4. §3.8 said focus returns to the toggle that opened the sheet. Measured after both an Escape close and a toggle close: focus goes to the composer editor. Fixed in §3.8, §5.5 and port notes 12 and 15.
5. §3.9 said the mobile sidebar sheet has no rail. A rail does render: x −8..8, "Toggle Sidebar", cursor default, a click closes the sheet. Fixed in §2.4, §3.9 and port note 13.
6. §7 said popover mode of the details card was not reached at 1440 and needs a narrower canvas. Opening the inline right panel at 1440 leaves a 644px canvas, so the card is a popover at the default window size. Measured and documented in §3.6 and §4.
7. The structure tree said `body` paints grain. It does in light only; a boot style removes it in dark. Fixed in §2.1 and §3.1.
8. Disabled toggle colour was given for light only. Dark is `#a1a1a1`. Added to §3.6.
9. Toggle hover was marked S, and the dark focus rings of the ghost trigger and the toggles M / S. All three are now measured.
10. §5.5 said a cancelled right-panel drag restores the start width. `pointercancel`, `lostpointercapture` and blur all commit; only an unmounted handle abandons the drag. Fixed.
11. Port note 15 said collapsing a panel moves focus to its toggle first. The inline panel prefers the element focus came from. Fixed.

### What was missing, now added

- The fixed top-right panel toggles, and the header details toggle, are hidden while the mobile sidebar sheet is mounted (§2.2, §2.4, §3.6, §3.9, §4).
- Mobile sheet in dark: the stage art fades to `#000000` over a `#111111` surface, so its tail ends slightly darker, with pixel values (§3.9).
- Details popover anchor, alignment, offsets and box (§3.6).
- Dev art's `data-stage-art="blueprint"` value (§3.4).
- Wide/full lane inset (276) and timeline padding (318) (§3.5).
- Detailed 600px measurements, and the 9px header-reserve shortfall below 640 (§2.4).
- Toggle `aria-label`s (§3.6).
- Escape first closes an open tooltip on the focused control (§3.8, §5.7).
- F6 behaviour from the body, the other routes' area slots, and the remounted-twin restore (§5.7).
- Desktop: `#01000000` is ARGB (alpha 1/255); tooltips keep clear of the title bar; the window title source; the Windows WSL "connecting" splash window, fully specified (§3.12, port note 7).
- A measurement caveat: the verification tab was backgrounded, so frames were forced with screenshots (§7).

### Still open

The items already in §7 stay open: Electron-only chrome and double-click actions, panel animations at non-zero durations, the terminal drawer opened live, the dev art drawn live, touch variants and non-zero safe areas. Two more are new: a live press on the right-panel resize handle, and drag-clamp re-measurement. Both failed in the backgrounded tab, and both values are re-derived from source.
