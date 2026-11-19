# 07 · Composer

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
Foundations: [01-tokens.md](01-tokens.md), [02-primitives.md](02-primitives.md), [03-shell.md](03-shell.md). Neighbour: [05-header.md](05-header.md) owns the BranchToolbar logic (run context, workspace, branch menus) that the composer's context strip reuses.

This document covers the chat composer: the glass shell and its outline, the prompt editor, the footer (attach, model and effort trigger, permissions chip, overflow menu, voice input, send / stop / queue / steer / resume, compact toggle), attachments and inline context chips, the context strip under a new thread's composer, the notice banners attached above it, the agent-question drawer, approval cards, the prompt stash, the `@` `/` `$` `#` menus, keyboard handling, drafts, the draft hero ("What should we build in …?") and the collapsed phone composer. The model picker popup itself is a separate document; here it appears only as its trigger, its open state and the two pieces `ComposerModelEffort.tsx` defines (the effort column and the picker footer's permissions menu).

Conventions, as in 05-header.md:

- Lengths are CSS px at the 16px root. One CSS px is one Gio dp. `rem` values scale with the interface font size.
- Colours are sRGB hex as Chrome paints them, `#rrggbbaa` when translucent, resolved as exact RGB plus alpha (01-tokens.md conventions). "Light / dark" cells read light first.
- **M** is measured live on the real composer. **P** is a probe: either a clone of the live composer stack (same classes and attributes, React handlers stripped, placed in a fixed overlay at the composer's lane) or new elements whose class strings were built from source with the app's own `cn` and `buttonVariants`, imported from the shipped bundle (`/assets/utils-ST3nadSl.js`, exports `t` and `Vr`). Chrome computed P values from the shipped CSS, so they are as real as M values minus runtime behaviour. **S** is source only.
- Coordinates are viewport px at 1440×900 @2x in Linux Chrome (Inter). Unless a row says otherwise: sidebar expanded at 256, details card inline at 1148, right panel closed. "Shell-relative" boxes put the composer shell's top-left at 0, 0.
- The composer sits outside `[data-app-sidebar]`, so it reads root tokens. Menus, popovers and the composer's drawer layer portal to `<body>`: they read root tokens and **do not see the shell's `--chat-composer-*` variables** (§3.9 shows where that changes a colour).
- **Read-only client.** The measuring browser was paired without the operate scope. Live consequences: the send button is disabled with the status "This connection cannot change threads." (ChatView.tsx:3631–3633), the mic is disabled (ChatComposer.tsx:1801–1807), and Stop never shows because `canInterrupt` requires `canOperateThread` (ChatView.tsx:1826–1827). Enabled looks are P.
- **Nothing was typed.** The brief forbids typing into the composer, so the menus, chips, attachments, stash, multi-line growth, the question drawer and the voice pills were measured on probes. Every probe class string is the source string (cited) passed through the app's `cn`.
- Text widths are Inter widths; SF Pro on macOS differs by a few px.

## 1. Scope and sources

Paths are relative to the T3 repo (`apps/web/src/` unless noted).

| File | Lines | What it gives |
|---|---|---|
| `components/chat/ChatComposer.tsx` | 1–3769 (all) | props (348–493), draft and attachment state (604–811), upload start (814–861), provider/model resolution (863–972), cursor, trigger and menu state (974–1153), approval / follow-up state (1154–1174), pending primary action, resume, phone collapse (1013–1018, 1209–1237), banner items (2464–2527), send pipeline (2108–2231), paste and drop (2529–2878), focus/collapse checks (2883–2934), imperative handle (2948–3221), JSX tree (3272–3769) |
| `components/chat/ComposerSurface.tsx` | 1–85 | Shell (5–31), outline classes (33–34), Host (36–50), Main (52–69), ContextStrip (71–83) |
| `components/ComposerPromptEditorTiptap.tsx` | 1–1973 | node views and chip wrapper (247–587), input rules (589–703), chip range selection (709–742), extensions (807–911), editor attributes (1077–1101), key handling (1112–1341), surround (1342–1383), paste (1384–1480), controlled value (1569–1653), focus API (1667–1790), copy/cut (1792–1818), render and placeholder (1820–1889) |
| `composerPlaceholder.ts` | 1 | default placeholder |
| `components/chat/ComposerFooterControls.tsx` | 1–331 | permissions chip (24–97), `ComposerPermissionsSection` (105–186), primary-actions wrapper (188–257), model picker wrapper and multi-model toggling (264–331) |
| `components/chat/ComposerControlsBar.tsx` | 1–312 | trigger states (40), ownership and sizes (134–164), mode block (165–182), model picker props (184–247), unavailable control (248–262), blocks and overflow (264–311) |
| `components/chat/ComposerControl.tsx` | 1–81 | `composerControlClassName` sm/xs (10–20), `ComposerControl` (26–41), `ComposerControlIcon` (43–65) |
| `components/chat/ComposerPrimaryActions.tsx` | 1–405 | pending labels (70–86), stop timeout (88), pill class (90–91), stop state (126–145), queue detection (151–158), stop button (163–192), pending actions (202–255), stop-only (257–268), labels / status / tooltip (270–305), send button (307–360), compact toggle (361–404) |
| `components/chat/ComposerModelEffort.tsx` | 1–677 | trigger suffix `describeComposerEffort` (84–177), effort column rows (179–523), footer segments (525–615), picker permissions menu (617–657), picker footer (660–677) |
| `components/chat/EffortGlyph.tsx` | 1–107 | bar glyph (6–89), Fast bolt (92–107) |
| `components/chat/ProviderModelPicker.tsx` | 140–491 | trigger labels (170–177, 232–278), trigger (337–437), popup placement (438–458), docked layout (459–482) |
| `components/chat/runtimeModeConfig.ts`, `composerControlsOwnership.ts`, `CompactComposerControlsMenu.tsx`, `composerEventScope.ts`, `useComposerMenuState.ts` | 1–64; 1–31; 1–54; 1–93; 1–10 | modes, where permissions show, overflow menu, floating-layer focus rules |
| `components/ComposerContextPermissions.tsx`, `ComposerContextLabel.tsx` | 19–67; 1–36 | the strip's permissions chip and label wrapper |
| `components/BranchToolbar.tsx`, `BranchToolbar.logic.ts` | 165–260, 434–593, 739–832; 66–92 | strip run-context trigger, label compaction and motion, strip layout, strip visibility |
| `components/chat/ComposerSpeechButton.tsx`, `VoiceInputPill.tsx`, `VoiceInputErrorPill.tsx` | 1–222; 1–76; 1–118 | mic button, recording / status pill, waveform, error pill |
| `components/chat/ComposerAttachmentStrip.tsx`, `ComposerImageThumbnail.tsx`, `composerAttachmentFiles.ts` | 1–748; 1–81; 1–210 | media row, file list, preview dialog, resting thumbnails, queued attachments, attach button, thumbnails, file classification and limits |
| `components/chat/useComposerAttachmentIntake.ts`, `useComposerImageGroups.ts` | 1–453; 1–40 | intake, compression, when images become chips, large-paste folding |
| `packages/client-runtime/src/textPaste.ts`, `packages/contracts/src/chatAttachment.ts` | 1–82; 13–17 | paste threshold and limits |
| `components/composerContextPresentation.tsx`, `contextChipParts.tsx`, `ContextChip.tsx`, `composerInlineChip.ts`, `ThreadContextChip.tsx`, `chat/FileTagChip.tsx`, `chat/ComposerPendingTerminalContexts.tsx`, `chat/TerminalContextInlineChip.tsx`, `contextPresentationRegistry.ts`, `ComposerContextLabel.tsx` | 1–496; 1–354; 1–102; 1–13; 1–61; 1–27; 1–32; 1–66; 1–62; 1–36 | chip kinds, states, details modes |
| `components/chat/ComposerBanner.tsx`, `ComposerBannerStack.tsx`, `ComposerServerUpdateStatus.tsx`, `ComposerUsageLimits.tsx`, `ComposerFreshSession.tsx`, `ComposerPromptLengthValidation.tsx`, `useSystemComposerBanners.tsx` | 1–324; 1–224; 1–14; 1–169; 1–16; 1–13; 150–263 | banner parts, notice cards, the banners ChatView feeds |
| `components/chat/ComposerPendingApprovalActions.tsx`, `ChildThreadApprovals.tsx`, `TimelinePendingApprovalRow.tsx` | 1–141; 1–95; 34–122 | approval buttons and cards |
| `components/chat/ComposerPendingUserInputPanel.tsx` | 1–363 | agent-question drawer |
| `components/chat/ComposerStashBadge.tsx`, `ComposerStashMenu.tsx`, `useComposerStash.ts`, `promptStashStore.ts` | 1–62; 1–242; 161–247, 461–730; 11–15 | stash tab, stash drawer, stash actions |
| `components/chat/ComposerCommandMenu.tsx`, `ComposerCommandMenuLayer.tsx`, `composerSlashCommandSearch.ts`, `useComposerMenuItems.ts`, `useComposerTriggerState.ts`, `composerMenuHighlight.ts`, `components/ui/command.tsx`, `components/ui/autocomplete.tsx` | 1–319; 1–95; 1–116; 1–421; 1–38; 1–20; 128–209; 123–204 | menus, placement, sources, ranking, highlight |
| `components/chat/composerShortcuts.ts`, `composerCommandKey.ts`, `composerPromptHistory.ts`, `composer-logic.ts`, `packages/client-runtime/src/state/composerDispatch.ts`, `packages/shared/src/keybindings.ts` | 1–38; 1–284; 1–161; 49–93, 251–331; 1–51; 57–123 | keys, history, dispatch modes, default bindings |
| `composerDraftStore.ts`, `components/chat/composerFocusRedirect.ts`, `useChatShortcuts.ts`, `composerTypingGuard.ts` | 118–232, 463–505, 1478–1483; 1–102; 72–80; 1–60 | drafts, type-to-focus, typing guard |
| `components/chat/DraftHeroHeadline.tsx`, `DraftHeroProjectHeading.tsx`, `DraftProjectName.tsx`, `draftHeroTransition.ts`, `useDraftHeroLayoutTransition.ts`, `ChatView.logic.ts` | 1–438; 1–63; 1–16; 1–93; 1–70; 303–323, 720–748 | draft hero, transitions, hero state, send state |
| `components/chat/CollapsedComposerPromptPreview.tsx`, `components/composerFooterLayout.ts`, `useComposerRestingTransition.ts`, `useComposerMultilinePrompt.ts` | 1–128; 1–142; 1–460; 1–53 | phone collapse, footer thresholds, resting transition |
| `components/ChatView.tsx` | 1735, 1759–1767, 1822–1858, 2572–2663, 3511–3832 | mount, strip visibility, focus helpers, banner items, overlay tree |
| `components/SidebarStageBackdrop.tsx` | 14–25, 67–69, 102–393 | send-button artwork |
| `index.css` | 191–232, 314–331, 2345–2386, 2436–2548 | motion tokens, stash pulses, ultrathink frame, composer editor rules |
| `packages/contracts/src/settings.ts` | 155–164, 408, 499–522, 590 | panel animation range, effort display, strip persistence, rich text, send shortcut, follow-up behaviour, skills in slash menu, SnapShot animations |

Live page: same build, 1440×900 @2x, light and dark. I visited a draft view with the strip and no prompt suggestion, a settled server thread showing a prompt suggestion, a running server thread whose trigger shows the Fast bolt, and the running thread at 600 px wide (collapsed phone composer). I opened and closed (Escape) the strip's permissions menu, the model picker and the hero project picker; hovered and Tab-focused every control. No message was sent, nothing was typed, no setting was changed. The sidebar's Working shelf I expanded was collapsed again. Screenshots stayed in a private folder.

## 2. Structure

### 2.1 Placement in the chat canvas

The overlay, lane and stack are specified in 03-shell.md §2.2 and §3.5. Recap of what the composer adds (ChatView.tsx:3511–3832):

```
div[data-chat-composer-overlay][data-workspace-area=composer][data-workspace-order=2]
│   thread: pointer-events-none absolute inset-x-0 bottom-0 z-20 pt-1.5 sm:pt-2
│   draft:  pointer-events-none absolute inset-0 z-20 flex items-center [@media(max-height:34rem)]:items-end
│   inert while a checkpoint revert runs
└─ div.chat-composer-lane.w-full
   └─ div[data-chat-composer-stack]   group/composer-stack pointer-events-auto relative z-10 mx-auto w-full max-w-(--chat-content-max-width)
      ├─ [draft hero] div.absolute.inset-x-0.bottom-full
      │     └─ div  pb-4 group-has-data-[composer-shoulder-tab]/composer-stack:pb-0  > DraftHeroHeadline   §2.16
      ├─ div.relative.z-10                       FLIP anchor of the hero transition
      │  ├─ ChildThreadApprovals                 §2.15 (child-thread approvals, above the card)
      │  ├─ div.relative
      │  │  ├─ ComposerSurface.Shell             §2.3
      │  │  │  ├─ ComposerSurface.Host           inert + aria-busy while a queued-message edit saves
      │  │  │  │  └─ div.relative.z-10
      │  │  │  │     ├─ [child threads] ProviderSubagentBar (the composer is not mounted then; ChatView.tsx:1735)
      │  │  │  │     └─ ChatComposer form       §2.2
      │  │  │  └─ div.min-h-0 > div.relative.z-0[data-terminal-open] > [strip] div.pointer-events-auto > BranchToolbar   §2.8
      │  │  └─ [draft hero with notices] div  absolute inset-x-0 top-full flex flex-col gap-1 pt-4 [@media(max-height:34rem)]:static
      │  │        └─ ComposerBannerStack detached   §2.9
      │  └─ div[aria-hidden]  h-[calc(env(safe-area-inset-bottom)+1rem)] sm:h-[calc(env(safe-area-inset-bottom)+1.25rem)]
```

- The composer is one persistent component across threads. Its primary actions are keyed by `environment:thread` so stop state belongs to one thread (ChatComposer.tsx:3658–3660). Its editor is keyed by draft target and text mode, so switching threads remounts the editor and restores focus if it had it (ComposerPromptEditorTiptap.tsx:774–788, 1667–1679).
- The strip shows only with a project and either the draft hero, or the setting `persistComposerContextStrip` (default off) in a Git repo or on a machine that should be named (BranchToolbar.logic.ts:66–79, ChatView.tsx:1759–1767).

### 2.2 ChatComposer tree

```
form[data-chat-composer-form][data-thread-context-drop]   mx-auto w-full min-w-0 max-w-(--chat-content-max-width)      ChatComposer.tsx:3273–3324
├─ ComposerBanner.Dock  div[data-slot=composer-banner-attachment]   hidden unless it holds an attached surface         3325–3359
│  ├─ ComposerBanner.Column
│  │  ├─ ComposerBannerStack → one Attachment + NoticeCard per notice                                                  §2.9
│  │  └─ [agent question] Attachment > Root[data-chat-composer-top-drawer][variant=info] > ComposerPendingUserInputPanel §2.10
│  └─ ComposerStashBadge (shoulder tab)                                                                                 §2.10
└─ div.relative
   └─ ComposerSurface.Main  div[data-chat-composer-main-surface][data-inline-resting-controls] (+ ultrathink-frame)      3360–3365
      └─ div[data-chat-composer-surface][data-chat-composer-mobile-collapsed=true|false]                                  3366–3377
         │   rounded-2xl transition-[background-color] duration-200
         │   in-data-[thread-context-over]:bg-accent/45 in-data-[thread-context-over]:ring-1 in-data-[thread-context-over]:ring-inset in-data-[thread-context-over]:ring-primary/70
         │   + (drag over) bg-accent/45 ring-1 ring-inset ring-primary/70   + (project selection required) opacity-75
         ├─ [phone, collapsed] prompt row                                                                                 §2.17
         ├─ div[data-chat-composer-body]   relative px-4 pb-2 pt-3.5 sm:pt-4   (hidden while collapsed)                   3437–3452
         │  ├─ [stash open, no menu] ComposerCommandMenuLayer > ComposerStashMenu                                        §2.11
         │  ├─ div[role=status][aria-atomic=true].sr-only  menu loading / empty announcement                              3472–3482
         │  ├─ [trigger active] ComposerCommandMenuLayer > ComposerCommandMenu                                            §2.11
         │  ├─ [editing a queued message] ComposerQueuedAttachmentRow                                                     §2.13
         │  ├─ ComposerAttachmentMediaRow   (images, SnapShots, videos, preparing images)                                §2.13
         │  ├─ ComposerAttachmentFileList   (other files without an inline chip)                                          §2.13
         │  └─ div.relative
         │     ├─ [file preview] ComposerFilePreviewDialog
         │     ├─ ComposerPromptEditorTiptap                                                                              §2.4
         │     └─ [phone, question] div[data-chat-composer-mobile-pending-actions]  absolute bottom-0 right-0 flex items-center justify-end gap-1
         ├─ ComposerPromptLengthValidation  p[role=alert][data-chat-composer-validation=prompt-length]   px-3 pb-2 text-xs text-destructive sm:px-4
         ├─ [dictation] div[aria-label="Live transcription"]  mx-3 mb-3 max-h-32 overflow-y-auto rounded-lg border border-border/60 bg-muted/30 px-3 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap sm:mx-4
         ├─ div[data-chat-composer-footer]   flex min-w-0 flex-nowrap items-center overflow-visible px-2 pb-2 (+ pt-2 with a question; hidden sm:flex on phones with the inline answer actions)   3625–3742
         │  ├─ [attach allowed] div.me-2 > ComposerAttachButton                                                           §2.13
         │  └─ div[data-chat-composer-actions=right][data-chat-composer-transition-actions][data-chat-composer-primary-actions-compact]
         │     │   flex min-w-0 flex-1 flex-nowrap items-center justify-end gap-2
         │     ├─ div[data-chat-composer-controls=footer][data-chat-composer-footer-controls]  relative flex min-w-0 flex-1 items-center justify-end gap-0.5
         │     │     └─ ComposerControlsBar                                                                               §2.5
         │     └─ ComposerFooterPrimaryActions > ComposerPrimaryActions                                                   §3.5, §5.3
         │           children (always just before the primary button): ComposerSpeechRecordingPill, ComposerSpeechButton, VoiceInputSetup
         └─ [phone, collapsed] div.h-8 > div.absolute.bottom-2.inset-x-4.min-w-0 > div[data-chat-composer-resting-controls]   §2.17
```

### 2.3 Shell, host and main surface: who paints the glass

ComposerSurface.tsx splits the card into three layers so the glass is painted exactly once:

| Configuration | Shell `::before` | Host `::before` | Host `::after` (outline) | Host shadow | Main surface | Main `::after` |
|---|---|---|---|---|---|---|
| plain (thread view) | **glass** | none | **outline** | `shadow-composer` (light) | transparent | hidden |
| context row: `Shell[data-context-row]` (draft with strip) | hidden | **glass** (`-z-1`) | **outline** | `shadow-composer` (light) | transparent | hidden |
| attached banner: the shell `has-[data-composer-banner-surface=attached]` | hidden | hidden | hidden | none | **glass** + `shadow-composer` (light) | **outline**, `z-20` |

Class strings:

- Shell (ComposerSurface.tsx:11–25), `div[data-slot=composer-shell][data-context-row?]`: `@container/composer-surface group/composer-surface relative isolate mx-auto w-full max-w-(--chat-content-max-width)`, the variable block (01-tokens §3.15), then `before:pointer-events-none before:absolute before:inset-0 before:z-0 before:rounded-2xl before:bg-(--chat-composer-glass-surface)/(--glass-opacity) before:backdrop-blur-(--glass-blur) before:backdrop-saturate-(--glass-saturation)`, `not-supports-[…backdrop-filter…]:before:bg-(--chat-composer-glass-surface)`, `has-data-[composer-banner-surface=attached]:before:hidden`, `data-context-row:before:hidden`.
- Outline (33–34): `after:pointer-events-none after:absolute after:inset-0 after:rounded-[inherit] after:border after:border-(--chat-composer-outline) dark:after:inset-shadow-2xs dark:after:inset-shadow-(color:--chat-composer-highlight)`.
- Host (39–45), `div[data-slot=composer-host]`: `relative z-10 w-full rounded-2xl shadow-composer after:z-1 dark:shadow-none`, the context-row `before:` glass (`-z-1`), the outline, and `group-has-data-[composer-banner-surface=attached]/composer-surface:{shadow-none, before:hidden, after:hidden}`.
- Main (56–64): `group relative z-10 rounded-2xl p-px`, the outline with `after:z-20 after:hidden`, and under an attached banner `after:block`, the glass fill and blur, `shadow-composer dark:shadow-none`, the no-backdrop fallback, and `**:data-[chat-composer-mobile-collapsed=true]:min-h-[calc(1rem+1px)]`.
- ContextStrip (76–79): `group/composer-context surface-glass relative flex w-full items-center gap-2 overflow-x-clip px-2 pt-2 text-muted-foreground`; BranchToolbar adds `gap-1 text-xs font-normal text-foreground/70 dark:text-muted-foreground` (BranchToolbar.tsx:743).

Why: the strip under a new thread is its own page-coloured glass, so the card paints only its own rect; with attached banners the main surface paints over the banners' tucked-under bottoms.

### 2.4 Prompt editor

```
div   relative flow-root font-(family-name:--font-composer,var(--font-sans)) text-(length:--font-size-prompt,var(--text-sm))
│     max-sm:pointer-coarse:text-(length:--font-size-prompt-touch)                                   ComposerPromptEditorTiptap.tsx:1824–1829
├─ EditorContent div
│  └─ div.tiptap.ProseMirror.composer-tiptap[role=textbox][data-testid=composer-editor][contenteditable]   1077–1101
│        -m-1 block max-h-52 min-h-[calc(2lh+--spacing(2))] overflow-y-auto p-1 whitespace-pre-wrap wrap-break-word
│        bg-transparent leading-relaxed text-foreground focus:outline-none   (+ max-sm:pb-12 with phone answer actions)
│        aria-multiline=true, aria-label "Message <thread title>" ("Message" without a thread), aria-placeholder,
│        aria-autocomplete=list aria-haspopup=listbox, aria-controls + aria-activedescendant while a menu item is active,
│        aria-readonly when disabled, data-composer-rich-text=true|false
│     └─ p | ul/ol/li | li[data-checked] task items | blockquote | h1–h6 | hr | code block node view | inline atoms (chips)
└─ [empty and no context records] div   pointer-events-none absolute inset-0 leading-relaxed text-placeholder        1873–1883
      placeholder text, then the suffix: span.ml-2.inline-flex.align-[1px] > Kbd variant=outline "⇥ Tab"  (ChatComposer.tsx:3571–3577)
```

- The `-m-1 p-1` pair gives the text 4px of breathing room for chip outlines without moving the text: the content box is 702 wide while the element is 710 (§3.1).
- Chips are inline atom nodes (`composer-mention`, `composer-skill`, `composer-model`, `composer-citation`, `composer-context-reference`, 250–587) rendered through `NodeViewWrapper as=span` with `relative inline-flex min-w-0 max-w-full select-none items-baseline` plus an unused `data-[composer-chip-selected]` highlight hook (247–248; no code sets that attribute in this build).
- Rich text (setting `composerRichTextEnabled`, default true; contracts settings.ts:514) adds markdown marks, lists, task lists, blockquote, headings, rules and highlighted code blocks (807–911). The stored prompt is always markdown text.

### 2.5 Footer controls bar

```
ComposerControlsBar                                                                                 ComposerControlsBar.tsx:248–311
├─ [no provider] ComposerControl[data-chat-provider-unavailable] shrink-0  > circle-alert 16, "Open provider settings" | "No provider available"
└─ else
   ├─ model + effort trigger                                                                        §2.6
   ├─ [mode block shown] div[data-resting-block=mode][data-composer-block-icon-only=true|false]   flex w-max min-w-max shrink-0 items-center gap-1
   │     hidden when it overflows: pointer-events-none invisible absolute (+ aria-hidden, inert)
   │     icon-only: labels → invisible absolute w-max; compact icons → relative
   │  └─ ComposerFooterModeControls: Menu > Tooltip > MenuTrigger > ComposerControl.ps-1.5[data-composer-shortcut=composer.mode]
   │        aria-label "Permissions: <label>"; ComposerControlIcon (16, 12 at xs; text-current when it differs from the default) + span[data-composer-control-label]
   │        shown only while the mode differs from the default AND no new-thread strip names it; in the collapsed phone row the label is sr-only (icon only)
   │        (composerControlsOwnership.ts:13–18; ComposerControlsBar.tsx:134–146; ComposerFooterControls.tsx:77–80)
   └─ div[data-resting-controls-overflow]  min-w-0 shrink-0 (pointer-events-none invisible absolute + aria-hidden + inert when empty)
      └─ CompactComposerControlsMenu: ComposerControl.shrink-0 aria-label "More composer controls" > ellipsis 16 (12 at xs)
            data-composer-shortcut=composer.mode only while it holds the permissions
```

`ComposerControl` (ComposerControl.tsx:10–20) is the composer's own control class, `sm` in the footer, `xs` in the strip and the phone resting row:

- base: `relative inline-flex shrink-0 cursor-pointer items-center justify-center whitespace-nowrap rounded-(--control-radius) border border-transparent text-base outline-none hover:bg-accent data-pressed:bg-accent focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64 data-disabled:pointer-events-none data-disabled:opacity-64 pointer-coarse:after:absolute pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11 [&:active:not([aria-haspopup])]:scale-[0.97] [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg]:-mx-0.5 [&_svg[data-composer-control-icon]]:mx-0` (on a coarse pointer an `::after` grows the hit area to at least 44×44 without changing the look), and labels (`[data-composer-control-label]`, `[data-chat-provider-model-picker-label]`) get `[text-box:trim-both_cap_alphabetic]` plus `py-1.5`, so a label box is cap height + 12 (22.19 at 14 px Inter, M). `supports-[text-box:trim-both_cap_alphabetic]` gates that padding (ComposerControl.tsx:13). Strip labels are `[data-composer-label]` and get no padding, so their box is the bare cap height (8.73 at 12 px, M).
- `sm`: `h-7 gap-1.5 px-2 font-medium text-foreground/80 [&_svg:not([class*='text-'])]:text-muted-foreground hover:text-foreground sm:text-sm [&_svg:not([class*='size-'])]:size-4.5 sm:[&_svg:not([class*='size-'])]:size-4`.
- `xs`: `h-7 gap-1 has-[[data-composer-label]]:gap-1.5 px-1.75 font-normal text-foreground/70 text-sm hover:text-foreground dark:text-muted-foreground sm:h-6 sm:text-xs [&_svg:not([class*='size-'])]:size-4 sm:[&_svg:not([class*='size-'])]:size-3.5 [&_svg]:mx-0 …chevron margins`.
- both: `aria-pressed:bg-accent aria-pressed:text-accent-foreground aria-pressed:hover:bg-accent/80`.

### 2.6 Model + effort trigger

```
button[data-chat-provider-model-picker][data-slot=popover-trigger][aria-haspopup=dialog]                     ProviderModelPicker.tsx:337–437
│  data-composer-shortcut="composer.effort" or "composer.effort composer.mode" (when it owns the permissions shortcut)
│  aria-label "<full model name>, <traits label>"   (no tooltip: the composer owns it)
│  ComposerControl sm + "min-w-0 shrink justify-between whitespace-nowrap" + "group/model-trigger data-popup-open:bg-accent data-popup-open:text-foreground min-w-13"
│     (+ "shrink text-xs!" when collapsed; ComposerControlsBar.tsx:214–218)
└─ span  flex min-w-0 flex-1 items-center gap-1.5 (gap-1 at xs)
   ├─ span[data-chat-provider-model-picker-icon].contents > ProviderInstanceIcon size-4 (instance badge h-3 min-w-3 px-0.5 text-5xs at −2/−2 when several instances share a driver)
   │     multi-model: span.flex.-space-x-1 with ≤3 icons size-4 rounded-full ring-2 ring-(--chat-composer-glass-surface) and "+N" size-4 text-3xs
   ├─ span[data-chat-provider-model-picker-label]  min-w-0 flex-1 overflow-hidden truncate max-w-56     the model's short name
   ├─ suffix: span  flex shrink-0 items-center gap-1.5 text-muted-foreground -ms-0.5 (sm)                 ComposerModelEffort.tsx:145–175
   │     group-hover / group-data-pressed / group-data-popup-open → text-foreground
   │  ├─ span[data-composer-control-label].max-w-24.truncate  effort word        | svg[data-composer-effort-glyph] bars (§3.6)
   │  ├─ [Fast | Ultrafast] FastBolt
   │  ├─ [Ultracode] workflow 12
   │  └─ [context-window option] span[data-composer-control-label][data-composer-secondary-trait].max-w-24.truncate.text-xs
   └─ [model unavailable] Badge variant=outline size=sm "Unavailable"
```

The popup opens docked: `side=top`, `align=end` (start when collapsed), `collisionAvoidance: shift` (ProviderModelPicker.tsx:449–454, ComposerControlsBar.tsx:191). Its body is the model list beside a 192 px effort column (`w-48`, `border-l border-border/70 bg-muted/40`) in a fixed 346 px tall row (`h-86.5`), then the footer (459–482). Measured popup: 554×393 at 477, 444, right edge = trigger right, bottom 4 px above the trigger (M).

### 2.7 Permissions controls

Modes (runtimeModeConfig.ts:14–39), in this order; a provider's `supportedRuntimeModes` filters them (44–53):

| Mode id | Label | Icon (lucide) | Description |
|---|---|---|---|
| `approval-required` | Supervised | `lock` | Ask before commands and file changes. |
| `auto-accept-edits` | Auto-accept edits | `pen-line` | Auto-approve edits, ask before other actions. |
| `auto` | Auto | `shield-check` | Supported providers approve routine actions; others still ask. |
| `full-access` | Full access | `shield-off` | Allow commands and edits without prompts. |

A managed policy can disable modes above its ceiling; they stay listed with the reason (55–64).

The same `ComposerPermissionsSection` appears in four places:

1. Footer chip (ComposerFooterControls.tsx:24–97): `sm` ComposerControl with `ps-1.5`, tooltip side top "Permissions: <label>. <description>", menu `align=start`.
2. Strip chip on a new thread (ComposerContextPermissions.tsx:37–66): `xs` ComposerControl `ms-auto min-w-0 shrink-0 data-composer-context-control`, icon 12, `ComposerContextLabel`, tooltip "Permissions · <shortcut>", menu `side=top align=end`.
3. Overflow menu "More composer controls", when the chip does not fit (CompactComposerControlsMenu.tsx:10–54).
4. The model picker footer's `PermissionsMenu` button (ComposerModelEffort.tsx:617–657): `ms-auto flex h-7 cursor-pointer items-center gap-1.5 rounded-md px-2 text-muted-foreground text-xs … hover:bg-foreground/[0.09] hover:text-foreground data-popup-open:…`, icon 14, label, `chevron-up` 12; menu `side=top align=end`.

```
div[data-composer-permissions]   grid w-72 gap-1.5 px-2 py-1.5                                   ComposerFooterControls.tsx:118–184
├─ div  font-medium text-muted-foreground text-xs   "Permissions"
├─ div[role=group][aria-label=Permissions]  flex rounded-lg bg-input/40 p-0.5   (Menu.RadioGroup)
│  └─ div[role=menuitemradio][aria-label][aria-describedby] × modes    flex h-6 flex-1 cursor-pointer items-center justify-center text-muted-foreground outline-none
│        first:rounded-l-md last:rounded-r-md data-disabled:cursor-not-allowed data-disabled:opacity-40 data-checked:bg-input/72 data-checked:text-foreground
│        data-highlighted:text-foreground not-data-checked:data-highlighted:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset
│     └─ icon size-3.5
├─ span.sr-only × modes   description or disabled reason (aria-describedby targets)
└─ div[aria-hidden].grid.gap-0.5
   ├─ span  text-sm, text-foreground (the current mode) | text-muted-foreground (previewing another)
   └─ span  min-h-8 text-muted-foreground text-xs leading-4   description (two lines reserved)
```

In the section (ComposerFooterControls.tsx:110–160), pointer-enter or focus on a segment previews that mode's label and description (pointer-leave or blur restores the current one); ArrowLeft / ArrowRight move focus between segments (the menu itself only navigates with ArrowUp / ArrowDown); each segment is a `Menu.RadioItem` with `closeOnClick`, so a pick closes the menu, and the change callback runs only when the picked mode differs. A disabled segment's `aria-describedby` text is its policy reason instead of the description.

### 2.8 Context strip (new threads)

BranchToolbar's composer layout (BranchToolbar.tsx:739–832), inside the shell under the host:

```
div[data-slot=composer-context-strip][data-compact?]  ContextStrip + gap-1 text-xs font-normal text-foreground/70 dark:text-muted-foreground
├─ [Git] div.contents @xl/composer-surface:hidden > RunContextSelector (one menu: Run on / Workspace)   (narrow composers, < 576)
├─ [Git or machine indicator] div  min-h-7 min-w-10 items-center gap-1 sm:min-h-6 flex-1 (hidden @xl:flex when Git)
│  ├─ [machine indicator] BranchToolbarEnvironmentSelector + Separator vertical mx-0.5 h-3.5
│  └─ [Git] BranchToolbarEnvModeSelector  (Select: folder | folder-git | folder-git-2 icon + "Current checkout" / "Current worktree" / "New worktree")
├─ [Git] BranchToolbarBranchSelector  min-w-0 flex-initial justify-end @xl:ml-auto   (combobox: git-branch 12 + branch)
└─ trailing: ComposerContextPermissions   (drafts only; server threads show no strip chip)
```

Labels use `ComposerContextLabel` (ComposerContextLabel.tsx:14–33): outer `span[data-composer-label] min-w-0 max-w-[240px] group-data-[compact]/composer-context:max-w-0`, inner `span[data-composer-label-motion] block w-full min-w-0 truncate [text-box:trim-both_cap_alphabetic] overflow-x-clip overflow-y-visible max-w-[240px] transition-opacity duration-180 ease-drawer group-data-[compact]/composer-context:opacity-0`. Menus and their item logic are in 05-header.md §5.6–5.9.

### 2.9 Banners

`ComposerBanner` parts (ComposerBanner.tsx):

| Part | Element / classes | Lines |
|---|---|---|
| Surface | `div[data-composer-banner-surface=attached\|detached][data-variant]`: surface colours, `relative isolate border-0 bg-transparent shadow-none [--chat-composer-attached-tint:transparent]`, variant colours; attached: `[--chat-composer-attachment-overlap:calc(1rem+1px)] before:rounded-t-2xl before:mask-t-from-transparent before:mask-t-from-4 before:mask-t-to-black before:mask-t-to-4`; detached: `[--chat-composer-attachment-overlap:0px] before:rounded-2xl`; `before:` glass with `before:border before:border-(--chat-composer-attached-outline)`, `before:bg-linear-to-b` tint, blur, `before:shadow-composer dark:before:shadow-composer-dark`, dark attached `before:bg-composer-seam-above` | 12–72 |
| Attachment | `div[data-slot=composer-banner-attachment] mx-auto -mb-[calc(1rem+1px)] w-[calc(100%-2*var(--chat-composer-drawer-inset))]`; a following attachment's surface gets `before:rounded-none before:border-t-0 [--composer-banner-corner-top:var(--control-radius)]` | 74–87 |
| Dock | Attachment + `flex items-end gap-1 not-has-data-[composer-banner-surface=attached]:hidden` | 89–99 |
| Column | `flex min-w-0 flex-1 flex-col empty:hidden [&>attachment]:w-full [&>attachment:last-child]:mb-0` | 101–112 |
| Root | Surface + `min-w-0 px-1 py-(--composer-banner-padding-block) after:block after:h-(--chat-composer-attachment-overlap) text-xs/4 [--composer-banner-corner-top:var(--radius-xl)] [--composer-banner-icon-column:--spacing(7)] [--composer-banner-padding-block:--spacing(1)] sm:[--composer-banner-icon-column:--spacing(6)]`; density comfortable `py 1.25`, spacious `px-3 py 3`; width fill `@container`, content `w-fit max-w-full flex-none`; corner rules make the first row's last action button `rounded-se-(--composer-banner-corner-top)` and a detached single row's `rounded-ee-xl` | 114–149 |
| Row | `div[data-composer-banner-row][data-composer-banner-layout=inline\|wrap-actions\|approval] group/banner-row grid min-h-(--composer-banner-icon-column) w-full min-w-0 grid-cols-[var(--composer-banner-icon-column)_minmax(0,1fr)_auto] items-center gap-x-1 text-start`; 2 columns without actions; as a button: `cursor-pointer rounded-xl hover:bg-foreground/5 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring`; wrap-actions: `flex flex-wrap`, content `max-w-[calc(100%-icon column-4px)] grow`; approval: `items-start gap-x-2 gap-y-3` | 151–177 |
| Icon | `span[data-slot=composer-banner-icon][aria-hidden] col-start-1 row-start-1 flex w-(icon column) items-center justify-center text-muted-foreground [&>svg]:size-3`; approval layout: `pt-0.5 text-warning [&>svg]:size-4` | 179–192 |
| Content | `span[data-slot=composer-banner-content] col-start-2 row-start-1 flex min-w-0 items-center gap-1`; spans columns 1–2 with `ps-2 sm:ps-1.5` when there is no icon; `pe-2 sm:pe-1.5` with neither icon nor actions | 194–208 |
| Separator | `span[data-slot=composer-banner-separator].mx-1.inline-block.flex-none.text-muted-foreground/40` "·"; inside Content its margin is 0 (`*:data-[slot=composer-banner-separator]:mx-0`), so only the content's 4 px gap spaces it | 210–220, 199 |
| Actions | `span[data-slot=composer-banner-actions] col-start-3 row-start-1 flex flex-wrap items-center justify-end gap-1` (wrap-actions: `ms-auto`) | 222–235 |
| Children | `grid gap-px [&_[data-composer-banner-row]]:min-h-5` | 237–246 |
| Scroll | ScrollArea `radius=none scrollFade h-auto max-h-[min(24rem,40dvh)]`, inner `[[data-has-overflow-y]>&]:pe-2` | 248–260 |
| Count | `inline-flex min-w-(--composer-banner-icon-column,1em) flex-none justify-center font-medium text-muted-foreground tabular-nums` | 262–272 |
| Body | `min-w-0 ps-8 sm:ps-7` | 274–276 |
| Dot | `span size-1.5 flex-none rounded-full bg-current` (6 px status dot in the text colour) | 278–282 |
| ToggleIcon | Button ghost icon-xs rendered as `span[aria-hidden]`, `pointer-events-none`, `chevron-down` 14, `rotate-180` when collapsed | 284–296 |
| Dismiss | Button ghost icon-xs, `x` 14 | 298–304 |

Notice card (ComposerBannerStack.tsx:166–224):

```
Attachment[data-composer-banner-stack]  + "transition-opacity duration-150 ease-out starting:opacity-0 motion-reduce:transition-none"
└─ Root[role=alert for error|warning][data-chat-composer-collapsed-controls] density=comfortable
   ├─ Row layout=wrap-actions
   │  ├─ Icon + "h-(--composer-banner-icon-column) self-start"  > item icon
   │  ├─ Content.whitespace-nowrap
   │  │  ├─ NoticeText title        min-w-0 truncate font-medium leading-7 sm:leading-6
   │  │  └─ [description] NoticeText  w-0 min-w-0 grow truncate text-muted-foreground @max-[400px]:sr-only
   │  │       (each NoticeText adds an "info" Button icon-xs ghost-muted when its text overflows; hover opens a tooltip-styled popover side top with the full text,
   │  │        aria-label "Notice details", max-w-[min(30rem,calc(100vw-2rem))], content in Banner.Scroll. Button labels: "Show full title: <title>" /
   │  │        "Show details: <title>" ("Show title details" / "Show notice details" for non-text titles). Once shown, the button stays until the banner
   │  │        grows wider than it was when the button appeared, so a wrap caused by the button cannot flap it; a text change resets it. ComposerBannerStack.tsx:64–164)
   │  └─ [actions or dismiss] Actions.min-w-14.sm:min-w-12  > actions, Dismiss (aria-label = dismissLabel ?? "Dismiss warning")
   └─ [children] Children
```

Groups and order: `environment` first, then `attention` (default), each in input order (ComposerBannerStack.tsx:28, 45–47). Detached mode (draft hero) renders plain `div[data-composer-banner-stack]` wrappers with `detached` cards.

Banners ChatView feeds, in input order (ChatView.tsx:2572–2663 assembles feedback, fallback, limit recovery, extension status, usage limits, project clone, fresh session, the system banners, branch mismatch; ChatComposer.tsx:2464–2521 appends reply-to and the managed ceiling). The stack then pulls the `environment` group to the top:

| Banner | Variant, group | Icon | Title / description | Actions |
|---|---|---|---|---|
| feedback submissions (one per in-flight feedback report) | per submission | not detailed here (ChatView.tsx:2540–2550) | | |
| fallback model (usage limit) | info | `gauge` | "Using <model>" / "<provider> is out of usage · resets <time>" | ghost xs "Use <default model>"; dismiss "Dismiss fallback model notice" (ComposerUsageLimits.tsx:72–103) |
| limit recovery, provider extension status | per item | not detailed here (ChatView.tsx:2551–2571, 2619–2622, ThreadActivityBanners.tsx:36+) | | |
| usage limits | info | `gauge` | "Usage limits" / account summary or "N accounts"; children: limit windows, notices, "Manage usage" link, reset credits | dismiss "Dismiss usage limits" (47–70) |
| project clone | per item | not detailed here (ChatView.tsx:2624) | | |
| fresh session armed (`/clear`) | info | `eraser` | "Your next message starts a fresh session" / "The agent won't see earlier messages in this thread." | dismiss "Keep the current session" (ComposerFreshSession.tsx:6–16) |
| machine unavailable | warning, or error when the connection errored; group environment | `wifi-off` | "<machine> is reconnecting" / "… is offline" + diagnosis | ghost xs "Reconnect" / "Retry now", "Disconnect server" (useSystemComposerBanners.tsx:186–216) |
| server update running / failed | default / error; group environment | `download` / `circle-alert` (`text-error`) | "Updating <server>" / "Could not update <server>" + stage or message | "Disconnect server" / "Retry"; dismiss on failure (219–261; ComposerServerUpdateStatus.tsx:5–14) |
| local checkout on another branch than the thread's | per item | not detailed here (ThreadActivityBanners.tsx:272+) | | restore branch, dismiss for the session |
| reply-to | default | `reply` | "Replying to [<subagent titles>]" / citation label as a button (`cursor-pointer hover:text-foreground`, opens the cited message) | dismiss "Cancel reply" (ChatComposer.tsx:2466–2495) |
| managed runtime ceiling | warning | `building` | the policy denial text | ghost xs "Switch to <mode>" (2496–2521) |

While an agent question is open only environment-group error/warning banners stay (2522–2527). On the draft hero ChatView passes no banners to the composer and renders them detached below it instead (ChatView.tsx:3647–3651, 3816–3823).

### 2.10 Dock: agent question and stash tab

Agent question (ChatComposer.tsx:3328–3350, ComposerPendingUserInputPanel.tsx:194–345):

```
Attachment > Root[data-chat-composer-top-drawer][variant=info] (density default)
└─ Collapsible[dir]
   ├─ CollapsibleTrigger = Row as button  px-4 py-1.5   data-pending-user-input-toggle=expanded|collapsed, title "Show/Hide the question and its options"
   │  ├─ Content: header span (min-w-0 truncate text-muted-foreground; max-w-1/2 when collapsed)
   │  │           [collapsed] Separator + question span (min-w-0 flex-1 truncate text-secondary-label)
   │  └─ Actions.gap-0: [several questions] Count "i/n", ToggleIcon, [dismissible] Dismiss as span[role=button][tabindex=0] "Dismiss question without answering"
   └─ CollapsiblePanel (height animation)
      └─ Scroll > Body.ps-6.pe-4.pb-1.wrap-anywhere.sm:ps-5.5
         ├─ ChatMarkdown question   pt-0.5 text-foreground/85 (+ text-right in RTL)
         └─ div.-ms-2.mt-1.5.sm:-ms-1.5
            └─ per option: div > button[aria-pressed]
                  group flex w-full items-start gap-2.5 rounded-md bg-transparent py-1.5 ps-2 sm:ps-1.5 outline-none transition-colors duration-150
                  focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-primary/25; selected text-foreground; else text-foreground/70 hover:bg-muted/30
                  disabled/responding: opacity-50 cursor-not-allowed
               ├─ span.pointer-events-none.flex.h-5.shrink-0.items-center > OptionRadio | CheckboxMark variant=contrast (multi-select)
               ├─ div.min-w-0.flex-1.flex.flex-col.gap-0.5 > label span (text-sm font-medium) + [description, only when it differs from the label] span (text-secondary-label text-2xs)
               └─ [index < 9] span.flex.h-5.w-7.shrink-0.items-center.justify-center.sm:w-6 > Kbd (default) "1"…"9"
               + [option.preview] ChatMarkdown lineBreaks  ps-9 pe-2 pb-1.5 text-xs text-foreground/70 sm:ps-8
```

`OptionRadio` (349–363): `relative inline-flex size-4.5 shrink-0 items-center justify-center rounded-full border border-foreground/55 bg-background not-dark:bg-clip-padding sm:size-4 dark:bg-input/32`, `shadow-xs/5` unchecked; checked adds `span.-inset-px.absolute.rounded-full.bg-primary` with a `before:size-2 sm:before:size-1.5 before:rounded-full before:bg-primary-foreground` dot.

Stash tab (ComposerStashBadge.tsx:30–60): Root density comfortable, width content, `data-composer-shoulder-tab`, `ml-auto`; Row as `button[data-prompt-stash-badge][aria-expanded]` (`aria-label "Stashed prompts: N. Open stash."`) with Icon `bookmark`, Content "Stash", Actions > Count. It sits right of the notice column, bottom-aligned, and is hidden at zero entries.

### 2.11 Drawer layer: command menu and stash

`ComposerCommandMenuLayer` (ComposerCommandMenuLayer.tsx:21–95) portals `div[data-composer-drawer-layer].pointer-events-auto.fixed.z-40.flex.flex-col` to `<body>`, positioned from the main surface's rect on every resize and scroll:

- `left = surface.left + inset`, `width = surface.width − 2 × inset`, inset = `--chat-composer-drawer-inset` (1.375 rem = 22);
- `bottom = innerHeight − surface.top − (rootFont + 1)` so the drawer tucks 17 px under the card;
- `maxHeight = max(96, surface.top − 24 + rootFont + 1)`.

Command menu (ComposerCommandMenu.tsx:107–186, 188–278):

```
Command (Autocomplete, inline, open, autoHighlight=false, mode=none)
└─ Surface[data-composer-command-drawer]  attached; flex min-h-0 w-full flex-col overflow-hidden pb-(--chat-composer-attachment-overlap)
   │   **:data-[slot=command-item]:rounded-lg **:data-[slot=scroll-area-scrollbar]:data-[orientation=vertical]:my-4
   ├─ [items] CommandList#listId[aria-label = Files and folders | Pull requests | Commands | Skills]  max-h-72 min-h-0 scroll-pb-6, list p-2
   │  └─ CommandGroup > CommandItem[role=option][id][data-composer-item-id][aria-selected]
   │        AutocompleteItem + CommandItem classes; active: bg-accent! text-accent-foreground!; hover never paints (highlight follows the mouse instead)
   │     ├─ lead: PierreEntryIcon (path) | messages-square 16 text-secondary-label (thread) | machine icon 16 | provider icon 16 | PR state icon 16 in its tone
   │     └─ span.flex.min-w-0.flex-1.items-center.gap-2
   │        ├─ span  min-w-0 max-w-[45%] shrink-0 truncate font-sans text-xs font-medium   label ("/skill:" prefix in text-secondary-label for skills under "/")
   │        ├─ span  min-w-0 flex-1 truncate text-left text-secondary-label text-xs       description
   │        └─ [skill] Badge secondary ms-auto: source icon + "App|Repo|Project|Personal|System|Provider" (+ " Skill" under "$")
   └─ [no items] div.px-5.pt-3.5.pb-7 > p.text-secondary-label.text-xs   loading or empty text
```

Stash drawer (ComposerStashMenu.tsx:127–240): Root `flex min-h-0 flex-col after:shrink-0` (default density), header Row as button "Close stash" (`bookmark`, Content muted "Stash", Count, ToggleIcon expanded), then Scroll > Children as `ul[role=list][aria-label="Stashed prompts"]` of Rows as `li` (`relative rounded-sm`, highlighted `bg-accent text-accent-foreground`): Icon `file-text`, Content > restore button (`button[data-stash-restore]`, aria-label "Restore stashed prompt: <snippet>", `min-w-0 flex-1 cursor-pointer truncate text-left text-foreground/80 outline-none before:absolute before:inset-0 before:rounded-sm focus-visible:before:ring-2 focus-visible:before:ring-inset focus-visible:before:ring-ring`; the `::before` makes the whole row its hit area), Actions: status ("saving N images…" muted | "N images dropped" `text-warning-foreground`), up to 3 image thumbs (`size-4 rounded border border-border/70 object-cover`, `-space-x-1.5`), file count (`file` 12 + N), `time[datetime]` (RelativeTime: "just now" under a minute, then "Nm ago", "Nh ago", "Nd ago", floored, re-rendered when the label would change; timestampFormat.ts:262–271, useRelativeTimeState.ts; `tabular-nums max-sm:hidden`), Dismiss "Delete stashed prompt" (`z-10` so it sits above the row-wide restore hit area). Empty: one Row "Nothing stashed yet. Press <shortcut> with a prompt in the composer to stash it.".

### 2.12 Inline context chips

`ContextChip` (ContextChip.tsx:8–71): `span[data-slot=context-chip][data-state]` (or the `render` element: button, link) with `inline-flex max-w-full items-center gap-[0.2em] font-medium leading-[normal] text-(--context-chip-ink) [--context-chip-ink:var(--context-chip-accent)] [--document-icon-stroke:var(--context-chip-ink)] [&_svg]:block [&_svg]:size-[1em] [&_svg]:shrink-0 [&_svg]:self-center [button&,a&,[data-popup-open]&]:cursor-pointer focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-current rounded-sm disabled:cursor-default` and a kind class `[--context-chip-accent:var(--context-chip-<kind>)]`. States: `unresolved` → accent `currentColor`; `invalid` → `--context-chip-invalid` (`--destructive-foreground`). `ContextChipLabel` (73–84): `block min-w-0 self-baseline overflow-x-clip text-ellipsis whitespace-nowrap underline decoration-transparent decoration-1 underline-offset-[0.2em] transition-colors [:is(button:enabled,a):hover>&]:decoration-current`.

| Reference | Chip | Icon | Click / details | Source |
|---|---|---|---|---|
| `[name](path)` file mention | kind `mention`, or a document kind (`doc-pdf`, `doc-word`, `doc-spreadsheet`, `doc-slides`, `doc-epub`) by extension; `dir=ltr` button | workspace entry icon (file-type icon or `file`/`folder`) | opens the file in the right panel; tooltip side top with the entry path | ComposerPromptEditorTiptap.tsx:273–310 |
| `$skill` (known, user-invocable) | `skill` | 24×24 package-cube SVG (stroke 1.85) | popover: label, description, "View instructions" | 391–414, composerInlineChip.ts:1 |
| `model:<instance>/<model>` | `ModelMentionChip` (neutral) | provider icon | popover: label, provider, source | 336–373 |
| citation | `citation` | `quote` | comment editor | 440–550 |
| `![label](t3-context://v1/image/<id>)` | `image` button | the image itself, 1em, `rounded-sm object-cover` | expands the image; tooltip "name\nsize[\n\nfailure]" | composerContextPresentation.tsx:186–210, contextChipParts.tsx:149–192 |
| `[label](t3-context://v1/file/<id>)` | `file`, `video` or a document kind; `invalid` on upload failure, `unresolved` when it must be re-attached | file-type icon, `film`, or `circle-dashed` | opens the preview dialog / video; tooltip | 212–255, 194–341 |
| terminal | `terminal`; `invalid` when expired | `terminal` | popover with the excerpt | TerminalContextInlineChip.tsx:16–66 |
| review comment / PR | `review-comment`, `pull-request`, `pr-open`, `pr-draft`, `pr-merged`, `pr-closed` | `message-circle` / PR state icon | popover / link preview | composerContextPresentation.tsx:412–440, contextChipParts.tsx:102–147 |
| preview annotation | `preview-annotation` | `mouse-pointer-click` | popover with the crop and targets | 441–462 |
| thread | `thread` link | project favicon or `messages-square` | opens the thread; tooltip "Open thread" or the unreadable reason (`invalid`) | ThreadContextChip.tsx:13–60 |
| machine | `environment` | machine icon | tooltip "T3 Code environment" / "Machine not in this app" | EnvironmentContextChip.tsx:12–40 |
| anything unknown or gone | neutral `unresolved` | `circle-dashed` | tooltip "This context is no longer available. Remove it or attach it again." | composerContextPresentation.tsx:351–358 |

Image and file chips put the name (`min-w-0 max-w-72 overflow-x-clip text-ellipsis whitespace-nowrap`, middle-truncated to 36 characters; a longer name keeps its first 21 and last 14 code points around "…", so the extension survives, `middleTruncateAttachmentName` in composerInlineChip.ts:3–13) and then the size and status in `text-3xs text-current` inside a `flex items-baseline gap-[0.33em]` label (contextChipParts.tsx:183–189, 332–338; composerInlineChip.ts:3–13).

### 2.13 Attachments

- Media row (ComposerAttachmentStrip.tsx:136–427): `div.mb-3.flex.max-w-full.gap-2`, `flex-wrap`, or with SnapShots a horizontal snap scroller (`snap-x snap-proximity overflow-x-auto overscroll-x-contain pb-1`, a thin 10 px scrollbar with a `foreground/18` rounded thumb). Image tile: `group/attachment shrink-0 snap-start bg-background relative h-16 w-16 overflow-hidden rounded-lg border border-border/80` > button (`h-full w-full cursor-zoom-in`, "Preview <name>") > thumbnail `object-cover`; overlays: non-persisted warning (top-left), "Preparing" / progress strip (bottom), failed-upload retry (Button `overlay` icon-xs, bottom-left), remove (Button `media-close` icon-xs, top-right; SnapShots show it on hover / focus-within only). SnapShot frame: `relative h-28 w-52 max-w-full overflow-hidden rounded-lg border border-border/80` (SnapShotAttachmentDetails.tsx:13–14). Video tile: same 64 box on `bg-black`, gradient `from-black/70 via-black/10 to-black/10` (to top), `play` 16 filled white with `drop-shadow-md`.
- File list (430–532): `div.mb-3.flex.flex-col.gap-1`; rows `flex min-w-0 items-center gap-2 py-1 text-sm text-foreground`: file-type icon 16, name button (`min-w-0 flex-1 truncate text-left hover:underline focus-visible:outline-2`), status `shrink-0 text-xs text-secondary-label` (size, "NN%", "Attach again", "Remove to send"), retry (ghost icon-xs, on failure), remove (ghost icon-xs). Files that have an inline chip are not listed (ChatComposer.tsx:673–689).
- Queued-message attachments (663–700): `mb-3 flex flex-wrap gap-2` of 64 tiles with `media-close`.
- Attach button (702–748): hidden `input[type=file][multiple]` and Button ghost icon-sm "Attach files" with `paperclip` (`translate-y-px`), tooltip "Attach files". Pointer-down never moves focus.
- Phone resting thumbnails (584–660): `size-7` tiles `rounded-md border border-border/70 bg-muted/60`, at most 3, then "+N" (`text-xs font-medium text-secondary-label tabular-nums`).

### 2.14 Voice input

- Mic (ComposerSpeechButton.tsx:91–120): Button ghost icon-sm (icon-xs compact) "Start voice input", `mic`; tooltip side top "Start voice input (<shortcut>)"; only in the idle phase; pointer-down keeps focus.
- Pill (122–222, VoiceInputPill.tsx:50–76): `div[data-slot=voice-input-pill] flex min-w-0 items-center gap-1.5 ring-1 ring-inset h-9 max-w-fit grow-[1.5] basis-0 rounded-full p-0.5 sm:h-8 bg-muted/48 ring-border` (error tone `bg-destructive/6 text-destructive ring-destructive/24`):
  - cancel Button ghost icon-sm-pill `x` (tooltip "Cancel voice input (Esc)");
  - status `div[role=status].flex.min-w-0.flex-1.items-center.gap-2.overflow-hidden` (`pe-2.5` without a trailing button): recording = dot `size-2 rounded-full bg-destructive`, elapsed `text-xs text-muted-foreground tabular-nums` ("m:ss", " / m:ss" with a limit), waveform; other phases = Spinner sm muted + `text-sm text-muted-foreground` status;
  - recording: Button default icon-sm-pill `check` "Finish voice input"; post-processing: Button ghost sm-pill "Skip".
- Waveform (VoiceInputPill.tsx:9–44): `div.flex.h-4.min-w-0.flex-1.items-center.justify-end.gap-0.5.overflow-hidden` in `me-1.5 flex w-16 min-w-0 shrink grow`, 24 bars `h-full w-0.5 shrink-0 origin-center rounded-full bg-foreground opacity-35 transition-[transform,opacity] duration-100 ease-out`. It scrolls right to left. Each new input level shifts the 24-entry history one bar left and lands on the rightmost bar (`copyWithin`, 16–24), so bar `i` shows level `i` of the history (opacity `0.3 + 0.55 × level`, `scaleY(max(0.2, level))`).
- Mic disabled (ChatComposer.tsx:1801–1807): no operate rights, machine unavailable, project selection required, an approval waiting, an answer submitting, or a choice-only question.
- Error pill (VoiceInputErrorPill.tsx:34–117): dismiss (ghost icon-sm-pill), focusable status (`circle-alert` 14 + short label, `text-sm font-medium`, tooltip = full message), action Button outline sm-pill ("Settings" | "Retry" | "Record again") or a 6 px spacer.
- Live transcription box: §2.2 tree; committed text plus a muted tentative span (ChatComposer.tsx:3615–3623).

### 2.15 Approvals

The composer itself shows no approval card in this build. A pending approval of the thread appears in the timeline (`TimelinePendingApprovalRow`, timeline doc) and changes the composer's placeholder and send meaning (§5.13). Approvals from child threads render above the shell (ChildThreadApprovals.tsx:15–95):

```
div  mb-2 flex max-h-80 flex-col gap-2 overflow-auto
└─ section[aria-label="Approval from <title>"]  rounded-lg border border-border bg-background p-3 text-xs
   ├─ p.mb-2.font-medium  "Approval needed from <title>"
   ├─ pre  max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono   detail or request kind
   ├─ [app name] p.mt-2.text-muted-foreground; [not live] p.mt-2.text-warning "Provider process is gone. Interrupt or restart the child thread to respond."
   ├─ [reason] p.mt-2.text-muted-foreground; [full-access override] p.mt-2.text-muted-foreground
   ├─ div.mt-2.flex.flex-wrap.gap-2 > ComposerPendingApprovalActions presentation=inline
   └─ [error] p[role=alert].mt-2.text-destructive
```

`ComposerPendingApprovalActions` (ComposerPendingApprovalActions.tsx:33–141): primary options are decline and accept; the rest (default: Cancel, "Always allow this session") go into a "More" menu (`side=top align=end width=content`). Inline presentation: `InlineButton` tone default for accept, muted otherwise, label `whitespace-normal`. Composer presentation (available, unused here): Button xs `default` for accept, `outline` otherwise, label `max-w-40 truncate`, More = Button outline icon-xs `ellipsis`. An option with a warning shows `triangle-alert` 12 and a tooltip.

### 2.16 Draft hero

```
DraftHeroHeadline   div.mx-auto.flex.w-full.max-w-5xl.flex-col.items-center                              DraftHeroHeadline.tsx:414–437
├─ [project resolved] DraftHeroProjectHeading                                                             DraftHeroProjectHeading.tsx:32–62
│  ├─ div.group/draft-hero.relative.w-full[data-inline=true|false]
│  │  ├─ span[aria-hidden] invisible measuring copy: flex w-max items-baseline gap-2 whitespace-nowrap font-normal text-3xl tracking-tight
│  │  └─ h1  flex w-full flex-col items-center justify-center gap-2 text-center font-normal text-2xl text-foreground tracking-tight sm:text-3xl
│  │        lg:group-data-[inline=true]/draft-hero:flex-row
│  │     ├─ span  max-w-full lg:inline:shrink-0 [@media(max-height:28rem)]:sr-only   "What should we build" + "?" (stacked) | " in" (inline)
│  │     └─ span  flex min-w-0 max-w-full items-baseline justify-center text-base tracking-normal lg:inline:text-3xl lg:inline:tracking-tight
│  │        ├─ span.min-w-0 > project selector
│  │        └─ span  hidden shrink-0 lg:inline:inline [@media(max-height:28rem)]:hidden!   "?"
├─ [scratch draft] h1 "What should we work on?"; [no project, can choose] h1 "<selector> to start"; [no projects] h1 "Add a project to start"
└─ [scratch workspace available] p.mt-2.flex.h-6.w-full.items-center.justify-center.text-sm
      > scratch: the project selector ("No project"); otherwise InlineButton muted "or start without a project" (tooltip side bottom: shortcut)
```

Triage hero (S; DraftHeroHeadline.tsx:89–91, 402–412; TriageDraftHero.tsx:11–59; useStartTriage.ts:19, 30). When a scratch ("No project") draft's prompt links the triage playbook file, the whole headline becomes this:

```
div.mx-auto.flex.w-full.max-w-5xl.flex-col.items-center
├─ h1  w-full text-center font-normal text-2xl text-foreground tracking-tight sm:text-3xl   "What went wrong?"
└─ p   mt-2 flex h-6 w-full items-center justify-center gap-1.5 text-sm
   ├─ span[tabindex=0]  min-w-0 truncate font-medium text-foreground   machine label ("This machine" without one) + span.ml-1.5.font-normal.text-muted-foreground "v<server version>"
   │     tooltip side bottom: "The agent reads T3 Code's logs and source here, and asks before it applies a fix or posts anything."
   ├─ span[aria-hidden].text-muted-foreground  "·"
   └─ InlineButton tone=muted  "or run" + code.ml-1.font-mono.text-xs "npx t3 triage"; click copies the command and reads "Copied" for 1.5 s; tooltip side bottom "Copy command, for a terminal"
```

Project selector (263–366): Combobox; trigger = `ComboboxTrigger render=InlineButton tone=picker`, `data-draft-project-trigger`, `pointer-events-auto min-w-0 max-w-full align-baseline` > `DraftProjectName singleLine` (`min-w-0 block truncate text-left`), tooltip side top = project title. Popup `align=center w-72 overflow-hidden`: search "Search projects...", empty "No matching projects.", items `hideIndicator`: project favicon 16 (or `folder-plus` for the last item "Add project"), name, pin button, machine badge. Without any projects the selector is a dotted-underline button "Add a project" (367–375).

### 2.17 Collapsed phone composer

Below 640 px (`max-sm`), when the composer is not focused, the prompt is one line, no question is open, and the hero does not force it open (ChatComposer.tsx:1013–1018):

```
div.flex.items-center.justify-between.gap-2.px-3.py-2                                                         3378–3435
├─ button[data-chat-composer-transition-prompt] aria-label "Expand composer"   min-w-0 flex-1 truncate bg-transparent p-0 text-left text-sm focus:outline-none
│     text-foreground with a prompt (CollapsedComposerPromptPreview: inert chips) | text-placeholder: "Enable a provider in Settings" | "Ask anything..."
├─ [images] ComposerRestingImagePreviews
└─ button[data-chat-composer-transition-actions]  flex size-8 shrink-0 items-center justify-center rounded-full bg-message-action text-message-action-foreground hover:bg-message-action-hover disabled:opacity-64
      aria-label "Resume thread" | "Open composer to compact and send" | "Send message"; play 16 filled | 16 px arrow (stroke 2)
+ resting controls row: div.h-8 > div.absolute.bottom-2.inset-x-4 > div[data-chat-composer-resting-controls]
      relative flex w-max min-w-0 max-w-full items-center gap-1 text-muted-foreground/70 [&_button]:text-xs!  (invisible + inert when they do not fit)
```

- While collapsed the body is `hidden` and the regular footer is not rendered at all unless voice input is active (ChatComposer.tsx:3446–3451, 3625); the controls render once, in the resting row, at size `xs`.
- The round send is disabled while the thread runs, a send is busy or blocked, no provider is available, a project must be chosen, the machine is unavailable, or there is neither sendable content nor a resume (ChatComposer.tsx:1223–1230). Its click resumes, expands the composer when a compaction is pending and not kept, or sends.
- Both buttons prevent focus on pointer down. The prompt button then expands the card and focuses the end of the prompt a frame later (`expandMobileComposer`, ChatComposer.tsx:2256–2277); the send button leaves focus where it was.

## 3. Measurements

### 3.1 Geometry at 1440×900 (M)

| Part | Thread view box (x, y, w, h) | Draft view box | Notes |
|---|---|---|---|
| overlay | 256, 760.5, 1184, 139.5 | 256, 52, 1184, 848 | thread `padding-top` 8 (6 below 640) |
| stack | 380, 768.5, 736, 131.5 | 380, 394.25, 736, 163.5 | draft: centred, 52 + (848 − 163.5) / 2 |
| hero wrapper | – | 380, 310.25, 736, 84 | outer div `bottom: 100%`; its inner div carries `padding-bottom` 16 (0 while the stash tab shows) |
| hero h1 | – | 380, 310.25, 736, 36 | inline layout |
| hero subline `p` | – | 380, 354.25, 736, 24 | `margin-top` 8 |
| shell | 380, 768.5, 736, 111.5 | 380, 394.25, 736, 143.5 | draft = host 111.5 + strip 32 |
| bottom spacer | 380, 880, 736, 20 | 380, 537.75, 736, 20 | 16 below 640 |

Shell-relative (identical in both views):

| Part | Box | Notes |
|---|---|---|
| host = form = main surface | 0, 0, 736, 111.5 | main `padding` 1 |
| surface | 1, 1, 734, 109.5 | radius 16 |
| body | 1, 1, 734, 69.5 | `padding` 16 16 8 (top 14 below 640) |
| editor content box | 17, 17, 702, 45.5 | |
| editor element | 13, 13, 710, 53.5 | margin −4, padding 4; min height `2lh + 8` = 53.5; max 208 |
| footer | 1, 70.5, 734, 40 | `padding` 0 8 8; content 32 tall |
| attach button | 9, 72.5, 28, 28 | wrapper `margin-right` 8 |
| actions group | 45, 70.5, 682, 32 | gap 8, `justify-content: flex-end` |
| controls | 45, 72.5, 606, 28 | gap 2, `justify-content: flex-end` |
| model trigger | right edge 651 | width follows content (measured 157.75 and 152.77 for two models) |
| mic | 659, 72.5, 28, 28 | |
| send | 695, 70.5, 32, 32 | right edge 9 inside the shell (1 + 8) |
| strip | 0, 111.5, 736, 32 | `padding` 8 8 0, gap 4 |
| strip controls | y 119.5, height 24 | first control at x 8; permissions ends at 728 |

Growth (P, clone with 1, 3 and 14 paragraphs): editor 53.5 / 76.25 / 208 (capped; `scrollHeight` 327); body 69.5 / 92.25 / 224; 3 lines = 3 × 22.75 + 8. The capped editor shows the 6 px app scrollbar (`clientWidth` 704 of 710).

### 3.2 Card surfaces

| Part | Property | Light | Dark | Token / rule | M/P |
|---|---|---|---|---|---|
| card glass (shell / host / main `::before` or fill) | background | `#ffffffcc` | `#111111cc` | `--chat-composer-glass-surface` (`--card` / `--surface-raised`) at `--glass-opacity` 80% | M |
| | backdrop-filter | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` | `--glass-blur`, `--glass-saturation` | M |
| | radius | 16 | 16 | `rounded-2xl` | M |
| | painted over the canvas | `#ffffff` | `#0e0e0e` | sampled | M |
| outline (`::after`) | border | 1px `#00000014` | 1px `#ffffff0d` | `--chat-composer-outline` | M |
| | inset highlight | none | `inset 0 1px 0 #ffffff08` | `--chat-composer-highlight` (`inset-shadow-2xs`) | M |
| | painted | `#ebebeb` | `#1a1a1a` | sampled | M |
| | highlight painted (the row under the top border) | – | `#151515` | sampled, verification pass | M |
| host shadow | box-shadow | `0 1px 2px #0000000d` | none | `shadow-composer`; dark `shadow-none` | M |
| | painted below the card (over the `#fcfcfc` page) | `#f6f6f6` at 1 px; the 2x device rows read `#f3f3f3`, `#f5f5f5`, `#f7f7f7`, `#f9f9f9`, `#fbfbfb` over the first 2.5 px | – (dark stays `#000000`) | sampled | M |
| strip | background | `#fcfcfccc` | `#000000cc` | `surface-glass` (`--background` at 80%) | M |
| | backdrop-filter | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` | | M |
| | painted | `#fdfdfd` | `#000000` | | M |
| main surface under attached banners | background, shadow | `#ffffffcc`, `0 1px 2px #0000000d` | `#111111cc`, none | | P |
| | `::after` | 1px `#00000014`, z 20 | 1px `#ffffff0d` + `inset 0 1px #ffffff08` | | P |
| no backdrop-filter support | fill | `#ffffff` | `#111111` | opaque glass surface | S |
| drag over / thread dragged over | surface background | `#f4f4f573` | `#ffffff05` | `--accent` at 45% | P |
| | ring | `inset 0 0 0 1px #1b4ed8b3` | `#346bf1b3` | `--primary` at 70% | P |
| project selection required | surface opacity | 0.75 | 0.75 | `opacity-75` | S |
| ultrathink prompt | main `::before` | 2 px ring: `linear-gradient(120deg, #ff6b6b 0%, #f59e0b 18%, #22c55e 36%, #14b8a6 54%, #3b82f6 72%, #ec4899 90%, #ff6b6b 100%)` at `220% 220%`, `saturate(0.82) brightness(0.92)`, masked to the 2 px padding ring, radius inherited | same | `.ultrathink-frame` (index.css:2345–2382); surface adds `shadow-[0_0_0_1px_rgba(255,255,255,0.07)_inset]`, provider icon `saturate(1.2)` | P |

Themed overrides under `html[data-theme-id]` are in 01-tokens §3.15.

### 3.3 Editor, placeholder, rich text

| Property | Light | Dark | Source | M/P |
|---|---|---|---|---|
| font | `--font-composer` (unset) → `--font-sans`: Inter here, SF Pro on macOS | = | | M |
| size / line height | 14 / 22.75 (`leading-relaxed`); 16 on phones with a coarse pointer | = | `--font-size-prompt`, `--font-size-prompt-touch` | M |
| text, caret | `#18181b` | `#f5f5f5` | `--contrast-foreground`; caret is `currentColor` | M |
| placeholder | `#71717b` | `#a1a1a1` | `--contrast-placeholder`, same font and line | M |
| white space | `break-spaces`, `overflow-wrap: break-word`; paragraphs margin 0 | = | `break-spaces` comes from Tiptap's injected (unlayered) ProseMirror CSS, which beats the layered `whitespace-pre-wrap` utility; index.css:2451–2453 only zeroes the paragraph margin | M |
| ligatures | `font-variant-ligatures: normal`, `font-feature-settings: normal` (Tiptap's base CSS turns ligatures off; T3 turns them back on to match sent messages) | = | index.css:2445–2448 | S |
| inline code | mono, 0.92em (12.88), background `#71717b1f`, radius 0.3em, padding 0.05em 0.25em | background `#a1a1a11f` | index.css:2460–2466 | P |
| bold | 700 | = | | P |
| markdown markers | `#71717b` at opacity 0.75, weight 400, no style/decoration | `#a1a1a1` | `.composer-rich-marker` (2469–2475) | P |
| node-selected chip | `outline: 2px solid #1b4ed8`, offset −2, radius 6 | `#346bf1` | `.ProseMirror-selectednode` (2483–2487) | P |
| chips in a text selection (focused editor) | `::after` `Highlight` at opacity 0.3, radius 6 (Linux Chrome `Highlight` = `#0041c6cc`) | = | `.composer-chip-range-selected` (2489–2501) | P |
| task item | `li[data-checked]` flex gap 8; checkbox 0.95rem in a 1lh label; nested lists indent 1.375rem | = | 2507–2548 | S |
| lists, headings, quotes, rules | shared with chat markdown (01-tokens §3.18); composer `ol` gutter grows with marker width | = | 1884–2101 | S |
| prompt-suggestion key hint | Kbd outline: 20 tall, min 20 wide, padding 0 6, radius 4, border 1px + bottom 2px `#18181b26` (foreground 15%), text 12/16 500 `#71717b`, gap 4, 8 px after the text, raised 1px | border `#f5f5f526`, text `#a1a1a1` | kbd.tsx; ChatComposer.tsx:3571–3577 | M (both themes, verification pass) |

### 3.4 Footer controls

| Part | Property | Light | Dark | M/P |
|---|---|---|---|---|
| attach (Button ghost icon-sm) | box | 28×28 (32 below 640), radius 8, 1px transparent border | = | M |
| | icon | `paperclip` 16 (18 below 640), `#71717b` (`--contrast-muted-foreground`), shifted down 1px | `#a1a1a1` | M |
| | hover | background `#f4f4f5` (`--accent`), text `#18181b`; icon stays muted | `#ffffff0a`, `#f5f5f5` | M |
| model trigger (`sm`) | box | 28 tall, padding 0 8, gap 6, radius 8, min width 52 | = | M |
| | text | 14/20 weight 500, `#18181bcc` (foreground 80%) | `#f5f5f5cc` | M |
| | label | short model name, cap-trimmed box 22.19 tall, max 224 | = | M |
| | suffix | `#71717b`, −2px start margin; effort word max 96 | `#a1a1a1` | M |
| | provider icon | 16, its own fill (Claude `#d97757`), −2px side margins | = | M |
| | hover / open | background `#f4f4f5`; label and suffix `#18181b` | `#ffffff0a`; `#f5f5f5` | M |
| | Fast bolt | `zap` 12, filled `currentColor` (suffix colour), 6 px after the word | = | M |
| overflow "…" (`sm`) | box | 34×28, `ellipsis` 16 `#71717b` | `#a1a1a1` | M |
| mic (Button ghost icon-sm) | box | 28×28, `mic` 16 `#71717b` | `#a1a1a1` | M |
| | disabled | opacity 0.64 | = | M |
| send / stop / compact | | §3.5 | | |

### 3.5 Send, Stop, compact toggle and pending actions (P)

| Element | Property | Light | Dark |
|---|---|---|---|
| send, plain variant (no stage art) | box | 32×32 round (36 below 640) | = |
| | fill / icon | `#1b4ed8` (`--message-action`) / `#ffffff` | `#346bf1` / `#ffffff` |
| | shadow | `0 1px 0 #ffffff29` (control highlight, outer) + `0 1px 2px #1b4ed83d` | `#ffffff29` + `#346bf13d` |
| | hover | `#3160db` (`--message-action-hover`), `scale: 1.05` | `#2f60d9`, 1.05 |
| | pressed (S) | shadow `0 1px 0 #00000014`, no drop shadow | = |
| | disabled | fill `#18181b1f` (foreground 12%), icon `#71717b`, no shadow, no hover scale | `#f5f5f51f`, `#a1a1a1` |
| send, stage-art variant (nightly / dev builds) | fill | the stage artwork (03-shell §3.4) in a `-z-10` span; compact viewBox `96 0 8192 96` (nightly) or `64 0 8192 96` (dev) with `preserveAspectRatio="xMinYMin slice"`, so the 96-unit-tall art is scaled ×⅓ to the 32 px button and shows viewBox x 96–192 (nightly) or 64–160 (dev) (SidebarStageBackdrop.tsx:113–118, 232; measured art rect 2730.67 wide, starting 32 px left of the button) | = |
| | shadow | `0 1px 0 #ffffff29` + `0 1px 2px #0000003d` | = |
| | hover | `scale: 1.05`, `filter: brightness(1.1)` | = |
| | disabled | opacity 0.64, no shadow (M) | = |
| send icons | idle | 14×14 arrow: path `M7 11.5V2.5M7 2.5L3 6.5M7 2.5L11 6.5`, stroke 1.8, round caps and joins | = |
| | running, steer | `corner-up-right` 16 | = |
| | running, queue (default queue, or ⌘/Ctrl held) | `list-plus` 16 | = |
| | resume | `play` 16 filled | = |
| | editing a queued message | `check` 16 | = |
| | busy | `loader-circle` 14 spinning (1 s linear, paused while off screen) | = |
| Stop | box | 32×32 round; 28 inside the pending-answer row at ≥640 | = |
| | fill | `#fb2c36e6` (`--destructive` 90%) | `#fb414ae6` |
| | shadow | `0 1px 0 #ffffff29` + `0 1px 2px #fb2c363d` | `#ffffff29` + `#fb414a3d` |
| | hover | `#fb2c36` | `#fb414a` |
| | icon | 12×12 SVG: rect 2, 2, 8×8, rx 1.5, white | = |
| | disabled (stopping, stopped) | opacity 0.64, `cursor: default`, no hover change | = |
| compact toggle (aria-pressed = compacting) | box | 28 tall, padding 0 8, gap 6, radius 8, 14/500 tabular (16 below 640), `minimize-2` 16 `#71717b`; focus 2 px inset `--ring`; aria-label "Compact <N> tokens of history before sending"; it and Send sit in `div[data-chat-composer-compact-send].flex.items-center.gap-2` (ComposerPrimaryActions.tsx:371–381) | = |
| | text | pressed `#18181b` on `#f4f4f599` (`--accent` 60%); not pressed `#18181bcc` | `#f5f5f5` on `#ffffff06`; `#f5f5f5cc` |
| | hover | `#f4f4f5`, `#18181b` | `#ffffff0a`, `#f5f5f5` |
| | label | "Compact" / "Keep" (hidden below a 480 px composer), then the token count `#71717b` ("128k") | `#a1a1a1` |
| pending answer: Previous | box | Button outline sm: 28 tall, padding 0 9, 14/500, radius 8, `#ffffff` fill, border `#d4d4d8`, `0 1px 2px #0000000d` | fill `#ffffff09`, border `#ffffff1c` |
| pending answer: row | layout | `flex items-center justify-end`, gap 8 (6 compact) (ComposerPrimaryActions.tsx:204) | = |
| pending answer: Submit pill | box | 28 tall (32 below 640), padding 0 16 (12 compact), round, gap 6, 14/500 (16/500 below 640: `text-base sm:text-sm`); focus 2 px inset ring with 1 px offset (ComposerPrimaryActions.tsx:90–91, 232) | = |
| | fill / text | `#1b4ed8` / `#ffffff`, shadow `0 1px 2px #1b4ed83d` | `#346bf1` |
| | disabled | opacity 0.64, no shadow | = |

### 3.6 Effort suffix glyphs (S, with measured word and bolt)

- Word (default, setting `effortDisplay: "label"`; contracts settings.ts:408): the current level's label, e.g. "Medium" (M). Named levels (max, ultracode, prompt-injected ones such as Ultrathink) always show their word (EffortGlyph.tsx:15, 30–33; ComposerModelEffort.tsx:119–131). Without a drawable glyph the word falls back by control type (ComposerModelEffort.tsx:119–131): a boolean reasoning option reads "<label> On" / "<label> Off"; an Ultrathink prompt on the primary control reads "Ultrathink"; otherwise the option's current label, or the descriptor label for a `thinking` control. With nothing to show (no word, glyph, Fast, Ultracode or context window) the trigger has no suffix (141–142).
- Bars (`effortDisplay: "bars"`; EffortGlyph.tsx:55–89): an SVG `4 × count − 2` wide and 14 tall (12 at `xs`). `maxBar = height − 2`; `step = max(1, round(maxBar / 2 / max(count − 1, 1)))`; bar `i` is 2 wide at `x = 4i`, `height = maxBar − (count − 1 − i) × step`, `y = maxBar − height` (bottoms 2 px above the box, on the text baseline); filled bars opacity 1, the rest 0.4; colour `currentColor`. Levels: the descriptor's options minus named levels; 2–7 levels or no glyph; a leading `off`/`none` level is not drawn and counts as zero filled. Example: 4 levels at `sm` → heights 6, 8, 10, 12, width 14.
- Fast: `zap` 12 filled. Ultrafast: a 15×12 box with an outlined `zap` at left 3 behind a filled one whose 4 px stroke uses the glass surface colour, so the echo is cut out (EffortGlyph.tsx:92–107).
- Ultracode: `workflow` 12. A context-window option adds its label in `text-xs`.

### 3.7 Context strip (M)

| Part | Property | Light | Dark |
|---|---|---|---|
| controls (`xs`) | box | 24 tall (28 below 640), padding 0 7, gap 6 with a label, radius 8 | = |
| | text | 12/16 weight 400 (14 below 640), `#18181bb3` (foreground 70%) | `#a1a1a1` (`--muted-foreground`) |
| | icons | 12, `currentColor` | = |
| | label | cap-trimmed box 8.73 tall, max 240 | = |
| | hover | background `#f4f4f5`, text `#18181b` | background `#ffffff0a`; text stays `#a1a1a1` (`dark:text-muted-foreground` wins) |
| | open | `data-pressed` → background `#f4f4f5` | `#ffffff0a` |
| layout at 736 wide | boxes | workspace select 8, 119.5, 132.61; branch 535.86, 91.06 wide; permissions 630.92, 97.08 wide, ends at 728. A second draft gave the same workspace and permissions boxes and a branch at 565.67, 61.25 wide, for a shorter branch name; the branch control is content-sized and ends 4 px before the permissions chip | = |
| tooltip | permissions | "Permissions · <shortcut>", side top | = |

### 3.8 Permissions menu (M; popup frame per 02-primitives §3.5)

| Part | Property | Light | Dark |
|---|---|---|---|
| popup | frame | `dropdown-glass`: `#ffffffd5`, border 1px `#18181b1a`, `blur(12px) saturate(1.14)`, radius 8, shadow `0 16px 40px -18px #0000008c` | `#111111d5`, `#f5f5f51a`, `blur(16px) saturate(1.08)`, `0 18px 44px -18px #000000cc` |
| | placement (strip chip) | side top, align end, 4 px gap: 810, 378, 298×132 for a chip ending at 1108 | = |
| | padding | viewport `padding` 4 | = |
| section | box | 288 wide, `padding` 6 8, row gap 6 | = |
| heading | text | "Permissions" 12/16 500 `#71717b` | `#a1a1a1` |
| segmented group | box | 28 tall, radius 8, `padding` 2, background `#d4d4d866` (`--input` 40%) | `#ffffff0b` |
| segment | box | 24 tall, equal widths (67 each for four), outer corners radius 6, icon 14 | = |
| | default | icon `#71717b` | `#a1a1a1` |
| | checked | background `#d4d4d8b8` (`--input` 72%), icon `#18181b` | `#ffffff14`, `#f5f5f5` |
| | highlighted, not checked | background `#f4f4f5`, icon `#18181b` | `#ffffff0a`, `#f5f5f5` |
| | disabled | opacity 0.4, `not-allowed` | = |
| preview label | text | 14/20, `#18181b` for the current mode; `#71717b` while previewing another | `#f5f5f5`; `#a1a1a1` |
| preview description | text | 12/16 `#71717b`, at least 32 tall | `#a1a1a1` |

### 3.9 Banners (P)

| Part | Property | Light | Dark |
|---|---|---|---|
| attached card | box | width = shell − 44 (inset 22 each side; 692 on a 736 shell), next element overlaps 17 (`-mb-[calc(1rem+1px)]`) | = |
| | padding, type | 5 4 (comfortable), 12/16 | = |
| | `::before` | glass `#ffffffcc` + blur, border 1px, first card radius 16 16 0 0, joined cards radius 0 and no top border, bottom 16 px masked out (`linear-gradient(to top, transparent 16px, black 16px)`) | glass `#111111cc` |
| | `::after` | block 17 tall (room under the tuck) | = |
| | outline (default, info, success) | `#00000014` inside the shell; `#18181b14` when portaled (foreground 8% fallback) | `#ffffff0d` |
| | warning | tint `#fe9a0014`, outline `#fe9a0047` | `#fe9a0014`, `#fe9a0047` |
| | error | tint `#fb2c3614`, outline `#fb2c3652` | `#fb414a14`, `#fb414a52` |
| | shadow | `0 1px 2px #0000000d` | `0 14px 32px -18px #000000bf` |
| | dark seam | – | `linear-gradient(to top, transparent 0 17px, #0000002e 17px, transparent 27px)` under the tint |
| | visible height | 34 (51 − 17); consecutive cards step 34 | = |
| row | layout | `min-height` 24, columns 24 \| 1fr \| auto (icon column 28 below 640), gap 4; wraps actions | = |
| icon | glyph | 12, `#71717b`, top-aligned 24 box | `#a1a1a1` |
| title | text | 12/24 weight 500, inherits `#262626` (28 line below 640) | `#f5f5f5` |
| description | text | 12/16 `#71717b`, grows, `sr-only` when the card is under 400 | `#a1a1a1` |
| actions | box | min width 48 (56 below 640), gap 4; ghost xs button 24 tall, padding 0 7, 12/500 | = |
| dismiss | box | Button ghost icon-xs 24×24, `x` 14 `#71717b`; first card's top-right radius 12, joined cards 8 | = |
| | hover | `#f4f4f5`, `#18181b` | `#ffffff0a`, `#f5f5f5` |
| detached card (hero) | shape | radius 16 all round, no mask, no overlap; 34 tall for one row; stack gap 4, 16 below the card | = |
| | dismiss of a one-row card | radius 12 on top-right and bottom-right | = |

### 3.10 Stash tab and stash drawer (P)

| Part | Property | Light | Dark |
|---|---|---|---|
| tab | box | width to content (95.86 for "Stash" + one digit), 51 tall (34 visible), surface radius 16 16 0 0 | = |
| | row button | 24 tall, radius 12, columns 24 \| auto \| 24; text `#71717b` | `#a1a1a1` |
| | hover | text `#18181b`, background `#18181b0d` (foreground 5%) | `#f5f5f5`, `#f5f5f50d` |
| | open | text `#18181b`, `pointer-events: none` | `#f5f5f5` |
| | count | min width 24, 500, tabular, `#71717b` | `#a1a1a1` |
| drawer | box | padding 4, `::after` 17; header row 24 | = |
| | rows | 24 tall (min 20), radius 4, gap 1; highlighted `#f4f4f5` with `#18181b` | `#ffffff0a`, `#f5f5f5` |
| | restore text | 12/16 `#18181bcc` (foreground 80%), truncated | `#f5f5f5cc` |
| | dropped images | `#bb4d00` (`--warning-foreground`) | `#ffb900` |
| | thumbnails | 16×16, radius 4, border `#e4e4e7b3`, overlap −6 | `#ffffff0e` |
| | time, file count | `#71717b`, tabular | `#a1a1a1` |

### 3.11 Command menu (P)

| Part | Property | Light | Dark |
|---|---|---|---|
| layer | box | 692 wide on a 736 card, tucked 17 under it; measured 175 tall for five items | = |
| surface | `::before` | glass `#ffffffcc`, border `#18181b14` (portaled fallback), radius 16 16 0 0, bottom 16 masked; `padding-bottom` 17 | `#111111cc`, `#ffffff0d`, seam gradient |
| list | box | padding 8, max height 288 | = |
| item | box | 28 tall (32 below 640), padding 6 8, gap 8, radius 8, 14/20 | = |
| | text | inherits `#262626` | `#f5f5f5` |
| | active | background `#f4f4f5` (`--accent`), text `#18181b` | `#ffffff0a`, `#f5f5f5` |
| | lead icon | 16; generic file/folder `#71717b` (`--icon-muted`), file-type icons in their colour | `#a1a1a1` |
| | label | 12/16 500, max 45% | = |
| | description | 12/16 `#71717b` (`--secondary-label`) | `#a1a1a1` |
| | skill badge | 18 tall, padding 0 3, 12/500, radius 4, `#fafafa` / `#27272a`, icon 12 at 80% | `#ffffff08` / `#f5f5f5` |
| empty / loading | text | padding 14 20 28; 12/16 `#71717b` | `#a1a1a1` |

### 3.12 Agent-question drawer (P)

| Part | Property | Light | Dark |
|---|---|---|---|
| card | surface | attached, neutral outline (info has no tint), padding 4 4 | as banners |
| header row | box | 36 tall, padding 6 16, radius 12; hover `#18181b0d` | `#f5f5f50d` |
| | header text | 12/16 `#71717b` | `#a1a1a1` |
| | count | "1/3" min 24, 500, tabular, `#71717b` | `#a1a1a1` |
| body | box | scroll max `min(384, 40% of the viewport)` (360 at 900 tall); padding 0 16 4 22 (24 below 640) | = |
| question | text | 14/22.75, `#18181bd9` (foreground 85%), padding-top 2 | `#f5f5f5d9` |
| options | box | start margin −6, top margin 6; rows padding 6 0 6 6 (start 8 below 640), gap 10, radius 6 | = |
| | selected / other | `#18181b` / `#18181bb3` | `#f5f5f5` / `#f5f5f5b3` |
| | hover (not selected) | `#fafafa4d` (`--muted` 30%) | `#ffffff02` |
| | label / description | 14/20 500 / 11/16 `#71717b` | – / `#a1a1a1` |
| radio | box | 16 (18 below 640), border `#18181b8c` (foreground 55%), fill `#fcfcfc`, unchecked shadow `0 1px 2px #0000000d` | border `#f5f5f58c`, fill `#ffffff09` |
| | checked | fill `#1b4ed8` (`--primary`) inset −1, white dot 6 (8 below 640) | `#346bf1` |
| key hint | box | Kbd 20×20, radius 4, `#18181b12`, 12/500 `#71717b` | `#f5f5f512`, `#a1a1a1` |

### 3.13 Chip colours (P)

All chips: inherit the editor size (14), weight 500, `line-height: normal` (17 tall), gap 0.2em (2.8), icons 1em (14), radius 4, label underline 1px at 0.2em offset, transparent until hover. Size and status text 10/14. Colours are `--context-chip-*` (01-tokens §3.16):

| Kind | Light | Dark | M/P |
|---|---|---|---|
| mention | `#006d80` | `#0d9ab4` | P |
| skill | `#92439f` | `#bf6ecc` | P |
| image, file | `#006a99` | `#0f96d3` | P |
| terminal, pr-open | `#00724f` | `#08a170` | P (terminal) |
| thread | `#006f6b` | `#009e97` | P |
| unresolved | text colour `#18181b` | `#f5f5f5` | P |
| invalid (failed upload, expired terminal, unreadable thread) | `#c10007` | `#ff6467` | P |
| video | `#a54800` | `#dd6e29` | tokens |
| environment | `#327037` | `#5c9b5e` | tokens |
| review-comment, pr-merged | `#6c51ba` | `#957bea` | tokens |
| preview-annotation, element | `#8c5800` | `#c27f19` | tokens |
| pull-request | `#5358c0` | `#7983ef` | tokens |
| pr-draft | `#5d6470` | `#868e99` | tokens |
| pr-closed | `#b23649` | `#e46371` | tokens |
| model | `#6c6500` | `#989016` | tokens |
| citation | `#2361be` | `#4e8cee` | tokens |
| doc-pdf / doc-word / doc-spreadsheet / doc-slides / doc-epub | `#b33739` / `#0066ab` / `#11732f` / `#9b4f00` / `#6b51bb` | `#e46561` / `#3e92da` / `#479e58` / `#cd7833` / `#937cea` | tokens |

### 3.14 Attachments (P)

| Part | Property | Light | Dark |
|---|---|---|---|
| media row | layout | margin-bottom 12, gap 8 | = |
| tile | box | 64×64, radius 8, border 1px `#e4e4e7cc` (border 80%), background `#fcfcfc` | `#ffffff10`, `#000000` |
| remove (`media-close` icon-xs) | box | 24×24 at 4/4 from the top-right, radius 8, `#000000a6`, white `x` 14, `0 0 0 1px #ffffff33` ring + shadow-sm; hover `#000000cc` | = |
| fallback | icon | `image` 16 `#71717b` | `#a1a1a1` |
| progress strip | box | bottom, 14 tall, padding 0 4, 10/14, `#fcfcfcd9` (background 85%), `#18181b`; "Preparing" or "NN%" | `#000000d9`, `#f5f5f5` |
| non-persisted badge | box | 16×16 at 4/4 top-left, radius 4, padding 2, `#fcfcfcd9`, `circle-alert` 12 `#bb4d00` | `#000000d9`, `#ffb900` |
| retry (`overlay` icon-xs) | box | 24×24 at bottom-left 4/4, `#000000b3`, icon `#ffffffa6`; hover `#000000e6` | = |
| video tile | box | 64 on `#000000`, gradient `#000000b3 → #0000001a`, `play` 16 white with `drop-shadow(0 3px 3px #0000001f)` | = |
| file row | box | 32 tall (padding 4 0), gap 8, 14/20 `#18181b`; status 12 `#71717b`; remove ghost icon-xs 24 | `#f5f5f5`, `#a1a1a1` |

### 3.15 Voice input (P)

| Part | Property | Light | Dark |
|---|---|---|---|
| pill | box | 32 tall (36 below 640), round, padding 2, gap 6, flex grow 1.5 from a 0 basis, never wider than its content (`max-w-fit`) | = |
| | fill / ring | `#fafafa7a` (`--muted` 48%) / inset 1px `#e4e4e7` | `#ffffff04` / `#ffffff14` |
| | cancel, finish | 28 round buttons; finish fill `#1b4ed8` with `inset 0 1px #ffffff29` | `#346bf1` |
| | recording dot | 8, `#fb2c36` | `#fb414a` |
| | elapsed | 12/16 `#71717b`, tabular | `#a1a1a1` |
| | waveform | 64 wide, 16 tall, 24 bars 2 wide gap 2, `#18181b` | `#f5f5f5` |
| | status | 14/20 `#71717b`, spinner 14 | `#a1a1a1` |
| error pill | colours | fill `#fb2c360f`, ring `#fb2c363d`, text `#fb2c36`; label 14/500 | `#fb414a0f`, `#fb414a3d`, `#fb414a` |
| | action | outline sm-pill 28 tall, padding 0 10 | fill `#ffffff09`, border `#ffffff1c` |
| live transcription | box | margin 0 16 12 (12 below 640), max 128 tall, radius 8, border `#e4e4e799`, fill `#fafafa4d`, padding 8 12, 14/22.75 `#262626`; tentative `#71717b` | `#ffffff0c`, `#ffffff02`, `#f5f5f5` / `#a1a1a1` |
| validation | text | 12/16 `#fb2c36` (`--destructive`), padding 0 16 8 (12 below 640) | `#fb414a` |

### 3.16 Approval card (P)

| Part | Light | Dark |
|---|---|---|
| card | radius 8, border 1px `#e4e4e7`, fill `#fcfcfc`, padding 12, 12/16 `#262626` | `#ffffff14`, `#000000`, `#f5f5f5` |
| detail | mono 12, max 160 tall, pre-wrap | = |
| secondary lines | `#71717b` | `#a1a1a1` |
| provider gone | `#fe9a00` (`--warning`) | `#fe9a00` |
| error | `#fb2c36` | `#fb414a` |
| inline actions | 12/500; Approve `#18181b`; others `#71717b` → hover `#18181b` + underline (offset 2) | `#f5f5f5`; `#a1a1a1` → `#f5f5f5` |
| composer presentation | outline xs 24 tall (`#ffffff`, `#d4d4d8`); accept default xs (`#1b4ed8`); More outline icon-xs 24 | per 02-primitives |

### 3.17 Draft hero (M)

| Part | Property | Light | Dark |
|---|---|---|---|
| h1 | type | 30/36 (24/32 below 640), weight 400, letter-spacing −0.75 (`tracking-tight`), `#18181b` | `#f5f5f5` |
| | inline layout | row, gap 8: "What should we build" + " in", the project name at 30/500, "?" | = |
| | stacked layout (S) | column, gap 8: "What should we build?" then the project name at 16 with normal tracking | = |
| project trigger | hover / open | underline offset 4, thickness from-font, colour `#18181b` | `#f5f5f5` |
| subline | type | 24 tall row, "or start without a project" 14/20 500 `#71717b` | `#a1a1a1` |
| | hover | `#18181b`, underline offset 2 | `#f5f5f5` |
| project picker popup | box | 288 wide (`w-72`), align center, side bottom, 5 below the name; items 28 tall; Combobox frame per 02-primitives §3.13 | = |

### 3.18 Collapsed phone composer (M at 600×900 in light; dark P)

| Part | Property | Light | Dark |
|---|---|---|---|
| surface | box | 574×80 (main 576×82) | = |
| prompt row | box | 48 tall, padding 8 12, gap 8 | = |
| | prompt | 14/20; `#71717b` placeholder "Ask anything..." | `#a1a1a1` |
| | send | 32 round `#1b4ed8`, plain variant even on nightly builds; disabled opacity 0.64 | `#346bf1` |
| resting controls | box | band 32 tall; controls absolute at bottom 8, inset 16; `xs` controls 28 tall, text forced to 12/16; wrapper colour `#71717bb3`, trigger text `#18181bb3` | `#a1a1a1b3`, `#a1a1a1` |

### 3.19 Focus indicators (M)

| Element | Indicator |
|---|---|
| composer controls, footer and strip buttons, banner action buttons | inset ring: `inset 0 0 0 1px` background (`#fcfcfc` / `#000000`) over `inset 0 0 0 3px` `--ring` (`#1b4ed8` / `#346bf1`): a 2 px ring 1 px inside the edge |
| editor | none (`outline: none`; the card shows no focus state) |
| chips | 2 px outline in the chip colour, offset −2 |
| banner rows that are buttons (stash tab, question header) | 2 px outline `--ring`, offset −2 |
| stash restore | `::before` 2 px inset ring |
| question options | 1 px inset ring `--primary` at 25% |
| permissions segments | 2 px inset ring `--ring` over a foreground 9% fill (`#18181b17` / `#f5f5f517`) |

Measured Tab order on a draft (read-only client): editor → attach → model trigger → (mic and send disabled, skipped) → strip workspace → strip branch → strip permissions → details card. An enabled client adds mic and send after the trigger.

### 3.20 Tooltips

All use the 02-primitives tooltip (12/16.5, side top unless noted, 4 px gap).

| Trigger | Text | Source |
|---|---|---|
| attach | "Attach files" | ComposerAttachmentStrip.tsx:744 |
| model trigger | none (aria-label only) | ProviderModelPicker.tsx:422–423 |
| footer permissions chip | "Permissions: <label>. <description>" | ComposerFooterControls.tsx:45, 84 |
| strip permissions | "Permissions · <shortcut>" ("Permissions" when `composer.mode` is unbound; the label follows the server's resolved keybindings, so a user override shows its own chord. The live client showed a customised chord, not the Mod+Shift+A default) | ComposerContextPermissions.tsx:25–28, 56 |
| mic | "Start voice input (<shortcut>)" | ComposerSpeechButton.tsx:75–88 |
| send (on a wrapper span, so a disabled button still explains itself) | status, else "Summarize N tokens of history, then send", else while running with steering possible and no queued message being edited "Click to <steer\|queue>, ⌘-click or <shortcut> to <queue\|steer>" (Ctrl on Windows/Linux), else the label | ComposerPrimaryActions.tsx:299–305, 355–360 |
| Stop | "Interrupt" / "Stopping…" / "Stopped" (aria-label "Stop generation" / "Stopping" / "Stopped") | 138–139, 188–190 |
| compact toggle | "Next send compacts N tokens first." + "Click to keep full history" (or "keeps all N tokens" + "Click to compact first") | 392–399 |
| effort column rows with a description | side right, max 224 | ComposerModelEffort.tsx:222–227 |
| hero project | the project title | DraftHeroHeadline.tsx:308–310 |
| "or start without a project" | the shortcut, side bottom | 398 |

### 3.21 Icons (lucide, stroke 1.75 unless noted)

| Icon | Size | Where |
|---|---|---|
| `paperclip` | 16 (18 below 640) | attach |
| `mic` | 16 | voice |
| `ellipsis` | 16 (12 at xs) | overflow menu "More composer controls" (ComposerControlIcon) |
| `ellipsis` | 14 (16 below 640) | approval "More approval options" in the composer presentation (Button outline icon-xs default icon size); the inline presentation that child approvals use shows the text "More" instead (ComposerPendingApprovalActions.tsx:99–111) |
| `corner-up-right`, `list-plus`, `check`, `play` (filled) | 16 | send states |
| `loader-circle` | 14 | busy send, voice status |
| `minimize-2` | 16 | compact toggle |
| `zap` (filled), `workflow` | 12 | effort suffix |
| `lock`, `pen-line`, `shield-check`, `shield-off` | 16 chip / 12 strip / 14 segments | permissions |
| `chevron-up` | 12 | picker footer permissions |
| `chevron-down`, `x` | 14 | banner toggle and dismiss |
| `bookmark`, `file-text`, `file` | 12 | stash |
| `reply`, `building`, `eraser`, `gauge`, `wifi-off`, `download`, `circle-alert`, `info` | 12 | banners, notice details |
| `messages-square` | 16 | thread items and chips |
| `image`, `film`, `terminal`, `quote`, `message-circle`, `mouse-pointer-click`, `circle-dashed`, `monitor` | 1em | chips |
| `image` | 16 | tile fallback |
| `refresh-cw`-style `RefreshIcon` | 14 | upload retry |
| `circle-alert` | 12 / 14 | tile warning / voice error |
| `folder`, `folder-git`, `folder-git-2`, `git-branch` | 12 | strip |
| `folder-plus` | 16 | hero picker "Add project" |
| `triangle-alert` | 12 | risky approval options |
| `blocks`, `folder`, `user-round`, `settings`, `package` | 12 | skill source badge |
| `chevron-left` | 14 | phone "Previous question" |
| custom arrow (stroke 1.8, 14 box) | 14 | idle send |
| custom stop square | 12 | Stop |
| package cube (stroke 1.85) | 1em | skill chip |

## 4. States

| Element | State | Look | Source / M-P |
|---|---|---|---|
| card | default | §3.2 | M |
| | focus inside | no change | M |
| | drag over (files or mentions, or a thread dragged in) | surface `--accent` 45% + inset 1px `--primary` 70% ring, 200 ms background transition | ChatComposer.tsx:3370–3376, P |
| | project selection required | surface opacity 0.75, editor read-only, placeholder "Choose a project above to start a thread" | 3374, 3565–3566, 3579 |
| | ultrathink in the prompt | rainbow 2 px frame (§3.2) | composerProviderState.tsx:118–134 |
| | queued-message edit saving | host `inert`, `aria-busy=true` | ChatView.tsx:3560–3563 |
| | checkpoint revert | overlay `inert`, send "Rewinding conversation" | ChatView.tsx:3517, 3637 |
| editor | empty | placeholder (§5.2); hidden as soon as any context record exists | ComposerPromptEditorTiptap.tsx:1873 |
| | disabled | `aria-readonly`, not editable: project selection required, choice-only question, answer submitting, dictation freezing the editor | ChatComposer.tsx:3578–3583 |
| | scrolling | past 208 tall; caret kept in view | §3.1 |
| footer controls | hover / pressed / open | `--accent` background; `sm` text → foreground; `xs` text → foreground in light only | §3.4, §3.7 |
| | disabled | opacity 0.64, no pointer events (model picker while the provider catalog loads or a send is busy) | ComposerControlsBar.tsx:192 |
| | icon-only | the mode block's labels hide when the measured width runs short | 274–284 |
| | overflowed | the mode block moves into "…" | 268–308 |
| permissions chip | matches the user's default, or the new-thread strip names the permissions | not shown (the picker footer answers ⌘⇧A; the picker trigger then carries `data-composer-shortcut="composer.effort composer.mode"`, M) | composerControlsOwnership.ts:6–31; ComposerControlsBar.tsx:143 |
| | differs from the default | shown, icon in the label colour | ComposerControlsBar.tsx:176 |
| | collapsed phone row | icon only (`accessDisplay: "icon"`, label `sr-only`) | composerControlsOwnership.ts:13–17 |
| send | §5.3 decision table | §3.5 | P |
| Stop | idle → pending ("Stopping", disabled) → landed ("Stopped", disabled) or timed out after 10 s (enabled again); while any stop phase is set the Stop stays rendered after Send, so typing cannot move the clicked target | §5.3 | ComposerPrimaryActions.tsx:126–145, 194–200 |
| model trigger | provider catalog pending | disabled, shows the thread's last model | ComposerControlsBar.tsx:192–207 |
| | no model | "Choose model"; multi-model "Choose models" / "A, B, N more" | ProviderModelPicker.tsx:170–177, 251–258 |
| | model unavailable | "(Unavailable)" in labels + outline Badge | 431–435 |
| context strip | labels overflow | `data-compact`: labels width 0 and opacity 0, icons only (16 px hysteresis) | BranchToolbar.tsx:486–593, BranchToolbar.logic.ts:81–92 |
| | narrow composer (< 576) | one run-context menu instead of the separate selectors | BranchToolbar.tsx:745–775 |
| | locked (started thread / locked mode) | plain span with the label, no menu | 262–281 |
| banners | entering | opacity 0 → 1 over 150 ms | ComposerBannerStack.tsx:30–31 |
| | text overflow | "info" button opens the full text | 64–164 |
| | card narrower than 400 / 560 | notice description `sr-only` / approval actions wrap under the text | ComposerBannerStack.tsx:203; ComposerBanner.tsx:200, 228 |
| question | expanded / collapsed | header-only row with "·" and the question | ComposerPendingUserInputPanel.tsx:195–228 |
| | responding | options disabled at opacity 0.5; Submit reads "Submitting..." | 280, ComposerPrimaryActions.tsx:76–78 |
| | not resumable | options disabled | ComposerPendingUserInputPanel.tsx:87–88 |
| stash tab | 0 entries | hidden | ComposerStashBadge.tsx:14 |
| | new stash | 1.4 s colour pulse (row foreground, count primary) | 16–27, 42–50 |
| | drawer open | tab text foreground, no pointer events | 43–45 |
| menus | loading | "Searching workspace files..." / "Searching workspace skills..." / "Finding pull request..." | ComposerCommandMenu.tsx:166–181 |
| | empty | "No matching files or folders." / "No matching command." / "No skills found. Try / to browse provider commands." / PR texts | useComposerMenuItems.ts:397–418 |
| chips | hover | label underline in the chip colour | ContextChip.tsx:73–84 |
| | node selected | 2 px `--ring` outline | index.css:2483 |
| | in a selection | `Highlight` at 30% | 2493–2501 |
| | unresolved / invalid | `circle-dashed` and text colour / `--context-chip-invalid` | §2.12 |
| attachments | preparing | fallback icon + "Preparing" strip, button disabled | ComposerAttachmentStrip.tsx:220–289 |
| | uploading | "NN%" strip after 150 ms; chips append " NN%" | useAttachmentUploadProgress.ts:7–18 |
| | upload failed | retry button + tooltip reason; chips `invalid` + "upload failed" | ComposerAttachmentStrip.tsx:290–314 |
| | not saved with the draft | warning badge on the tile | 263–280 |
| | needs re-attach | file chip `unresolved` "attach again"; list status "Attach again" / "Remove to send"; send blocked | composerContextPresentation.tsx:218–252 |
| voice | idle / preparing / recording / transcribing / post-processing / error | mic / pill variants (§2.14); status "Starting…" or "Downloading model N%", "Recording", "Transcribing…", "Cleaning up…", error text or "Voice input failed" | ComposerSpeechButton.tsx:17–40 |
| phone | collapsed / expanded | §2.17 | ChatComposer.tsx:1013–1018 |

## 5. Behaviour

### 5.1 What the composer shows and where it comes from

- The draft for the current target (a server thread ref or a draft id): prompt markdown, images, files, terminal contexts, preview annotations, review comments, thread and machine contexts, reply-to, per-provider model selection, active provider, runtime mode, interaction mode (composerDraftStore.ts:463–482).
- Model and effort: from the draft's selection, else the thread's, else the project default, else sticky provider state; the trigger shows the model's short name (`getTriggerDisplayModelName`) and the traits label (ChatComposer.tsx:863–972; providerIconUtils.ts:30–44).
- Permissions: the thread's runtime mode limited to the provider's `supportedRuntimeModes` (runtimeModeConfig.ts:44–53); the default mode comes from server settings (`defaultRuntimeMode`).
- Send state: sendable content = prompt text other than inline references, or any image/file, or a queued message's existing attachments, or terminal contexts with text, or annotations/review comments (ChatView.logic.ts:720–748).
- Send block reasons, first wins (ChatView.tsx:3631–3645, ChatComposer.tsx:924–932, 765–786): "This connection cannot change threads.", "Preparing machine", "Rewinding conversation", "Sending feedback", "Messages loading", "Preparing worktree", project clone reason, "Select at least one model.", attachment reasons ("Update this server to send files with question answers", file capability reasons, "Attach the interrupted file(s) again or remove it/them", upload waits), the managed-policy denial, provider reasons (Antigravity install / sign-in). While voice input prepares: "Preparing voice input" (1848–1850).

### 5.2 Placeholder

In priority order (ChatComposer.tsx:3554–3577; composerPlaceholder.ts:1):

1. Approval waiting: "Send feedback to decline this request" (provider takes feedback) or "Send a message to decline and redirect the agent".
2. Agent question: "Choose an option above" (choice only), "Type your answer" (no options), else "Type your own answer, or leave this blank to use the selected option".
3. "Choose a project above to start a thread".
4. "Enable a provider in Settings to send a message".
5. The prompt suggestion text, followed by the outline Kbd "⇥ Tab". Offered only when idle, not busy, no fresh session armed, no dictation freeze, no context records, no approval or question, a project and a provider (2280–2300).
6. "Ask anything. @ files, $ skills, / commands".

### 5.3 Primary actions

`ComposerPrimaryActions` picks one layout (ComposerPrimaryActions.tsx:202–404):

1. **Question open**: [Previous (outline sm; compact: outline icon-sm `chevron-left`) when past the first question] [speech] [Submit pill: "Next question" / "Submit answer" / "Submit answers" (compact "Next" / "Submit"; responding "Submitting...")] [Stop when interruptible]. The pill is disabled while responding, without operate rights, with the environment unavailable or a blocked send, or until the answer is complete / can advance (voice input overrides that last rule).
2. **Interruptible run (or a stop pending or timed out) and nothing typed** (and not editing a queued message): [speech] [Stop] (257–268).
3. **Otherwise**: [compact toggle when a compaction is pending and not hidden] [speech] [Send] [+ Stop while any stop phase is set (pending, timed out or landed), so the clicked target stays at the right edge] (194–200).

Send label and icon:

| Condition (first match) | aria-label / tooltip label | Icon |
|---|---|---|
| resume available and no content | "Resume thread" (`type=button`) | `play` |
| editing a queued message | "Update queued message" | `check` |
| compaction pending and not kept | "Compact and send" | arrow |
| running and dispatch = queue | "Queue message" | `list-plus` |
| running | "Steer message" | `corner-up-right` |
| else | "Submit message" | arrow |
| busy (any) | status "Submitting message" / "Updating queued message" | spinner |

A status replaces the aria-label: "Environment disconnected", the block reason, "Preparing worktree", or the busy text (289–298). Disabled when busy, blocked, disconnected, or without content and resume (318–323). The stage-art variant applies when the server's stage label is "nightly" or "dev" (or the environment identification setting forces artwork) (SidebarStageBackdrop.tsx:14–25).

Stop: click → "Stopping" (disabled); `canInterrupt` turning false → "Stopped"; turning true again or a new run id resets; a pending stop re-enables after 10 s (ComposerPrimaryActions.tsx:126–145).

### 5.4 Dispatch while a turn runs

- `followUpBehavior` setting: `steer` (default) or `queue` (contracts settings.ts:518–520). An approval shown in the thread forces `steer`, so a send declines it and redirects the agent (ChatComposer.tsx:1157–1161).
- `resolveComposerDispatchMode`: not running → `auto`; `canSteer` false (an explicit `/compact` or compaction that cannot take steers) → `queue`; the alternate modifier swaps steer ↔ queue (composerDispatch.ts:4–16).
- Alternate modifier: ⌘ (Ctrl) held while clicking Send, or the `composer.sendAlternate` chord (Mod+Enter) while running (ChatComposer.tsx:2207–2218). The Send icon follows the live modifier state (`useShortcutModifierState`), so holding ⌘ swaps `corner-up-right` ↔ `list-plus` before the click (ComposerPrimaryActions.tsx:150–158).

### 5.5 Keyboard

Inside the editor, in this order (ComposerPromptEditorTiptap.tsx:1127–1341, composerCommandKey.ts:202–281, composer-logic.ts:59–93):

| Key | Effect |
|---|---|
| any key while composing (IME, `isComposing` or keyCode 229) | never sends, never navigates; Enter is swallowed for the IME |
| Enter inside a rich-text code block | closing fence exits, trailing blank lines exit, else an indented newline |
| ```` ``` ```` + Enter | converts the fence into a code block |
| Enter / Tab with a menu open | inserts the highlighted (or first) item |
| ArrowDown / ArrowUp with a menu open | moves the highlight, wrapping |
| Escape with a menu open | dismisses that trigger (the same `@`/`/`/`$`/`#` at that position stays dismissed until it changes) |
| Escape, empty prompt, suggestion offered | dismisses the suggestion |
| Tab, empty prompt, suggestion offered | fills the suggestion |
| ArrowUp / ArrowDown, no menu | prompt history (§5.7) |
| Enter (setting `sendShortcut: "enter"`, default) | sends (foreground); not with Shift or Alt, not on key repeat, not with a software keyboard open |
| Shift+Enter | new paragraph; in a list item splits or lifts it; in an empty quote line lifts out |
| Mod+Enter | running: alternate dispatch; draft: background send (the thread starts, a new draft opens in the same project); otherwise a plain send. When the setting requires a modifier for this prompt (`mod-enter`, or `mod-enter-multiline` with a newline), Mod+Enter is the plain send and the alternate and background chords become Mod+Shift+Enter (apps/web/src/keybindings.ts:113–130) |
| Mod+Alt+Enter | draft: background send; server thread: send and open a new thread |
| `sendShortcut: "mod-enter-multiline"` | Enter sends a one-line prompt; once the prompt has a newline, Mod+Enter is needed |
| Tab / Shift+Tab | code block indent/outdent; plain-text list indent/outdent; Shift+Tab after a citation focuses its comment button |
| ArrowLeft / ArrowRight next to a chip | steps over the whole chip; at a styled edge visits the marker stop first |
| macOS Home/End, ⌘←/→, ⌘↑/↓ | line / document boundary via `Selection.modify` (with Shift extends) |
| PageUp / PageDown when the editor cannot scroll that way | scrolls the timeline |
| a delimiter (`(` `[` `{` `'` `"` `“` `` ` `` `<` `«` `*` `_`) over a plain selection | wraps the selection (marks for `*`, `_`, `` ` ``) |
| Mod+B / Mod+I / Mod+E (no Shift or Alt) | bold / italic / inline code, unless a composer key handler or an app command bound to the chord in this context takes it first (Mod+B with no selection toggles the sidebar, 03-shell §5.7) (ComposerPromptEditorTiptap.tsx:1114–1126; apps/web/src/keybindings.ts:403–407) |

Global composer chords (packages/shared/src/keybindings.ts:72, 93–96, 117–123; composerShortcuts.ts:14–38), all ignored on key repeat:

| Chord | Command | Effect |
|---|---|---|
| Mod+Shift+M | `modelPicker.toggle` | open / close the model picker |
| Mod+Shift+E | `composer.effort` | open the picker on the effort column |
| Mod+Shift+A | `composer.mode` | open the permissions menu wherever it lives (chip, overflow, picker footer) and focus the checked mode |
| Mod+Shift+H / X / G / L | `composer.host` / `composer.workspace` / `composer.branch` / `composer.previousWorktree` | strip controls |
| Mod+Shift+D | `composer.dictation` | start / finish voice input |
| Mod+S | `composer.stash` | stash (§5.11); always claimed so the browser's save dialog never opens |
| Alt+ArrowUp | `thread.editQueuedMessage` | edit the last queued message |
| Mod+Shift+Enter | `thread.steerQueuedMessage` | steer the queued message now |
| (unbound) | `composer.increaseEffort` / `composer.decreaseEffort` | step the reasoning level |

`openControl` focuses the first visible, enabled `button[data-composer-shortcut~=command]` in the shell and clicks it (ChatComposer.tsx:3033–3053).

### 5.6 Triggers and menus

Detection on the word before the caret (composer-logic.ts:251–293): `/` at its start (unless the word is a path with a second slash) → commands; `#` alone or `#` + a letter or digit then letters/digits/`_`/`-` (`/^#([\p{L}\p{N}][\p{L}\p{N}_-]*)?$/u`, so a bare `#` opens the menu with an empty query) → pull requests; a currency symbol (`\p{Sc}`, normally `$`) not followed by an amount → skills; `@` → files, folders, threads, machines and models. No trigger next to a chip, in a code block or in a literal answer.

Items (useComposerMenuItems.ts):

- `@`: matching threads (by recency), machines (when linked machines are offered and not answering a question), models, then workspace entries (`label` = basename, `description` = parent path) (198–231).
- `/`: `/model` "Switch response model for this thread", `/side` and `/btw` (side chat), `/clear` "Start a fresh session; earlier messages stay visible", provider slash commands (not `compact` unless allowed, not `clear` when T3's `/clear` exists), and skills as `/skill:<name>` when `showSkillsInSlashMenu` (default true). `/side`, `/btw`, `/clear` and provider commands appear only at the start of the prompt (232–309; composerSlashCommandSearch.ts:16–31).
- `$`: skills ranked by `searchProviderSkills` (310–322).
- `#`: pull requests of the project's repository, up to 12 shown (from a list of up to 99), linked ones first; text and number queries settle after 180 ms, and a number missing from the list fetches that exact pull request (useComposerMenuItems.ts:55–56, 108–190, 323–383).
- Ranking for `/`: command name exact 0, prefix 2, boundary 4, includes 6, fuzzy 100+; description 20–26; ties by built-in, provider command, skill (composerSlashCommandSearch.ts:33–116).
- Highlight: the first item by default; a mouse move highlights; the highlight sticks while the trigger kind and query stay the same (composerMenuHighlight.ts:1–20). The active item's DOM node scrolls into view; the editor exposes it with `aria-activedescendant`.

Selecting (ChatComposer.tsx:1877–2062) replaces the trigger text and swallows one following space:

| Item | Inserted |
|---|---|
| path | `[basename](encoded path) ` |
| `/model` | nothing; opens the model picker |
| `/clear` | nothing; arms a fresh session (banner) |
| `/side`, `/btw` | `/side ` / `/btw ` |
| provider command | `/<name> ` (the usage-limits command opens the usage panel instead) |
| skill | `$<name> ` |
| pull request | a review-comment context reference |
| thread / machine | a context reference + its record |
| model | `model:<instance>/<model> ` |

### 5.7 Prompt history

ArrowUp recalls earlier prompts when there are no attachments or contexts, no question, no modifiers, the caret is on the first visual line, and the prompt is empty or still shows a recalled entry (composerCommandKey.ts:140–181). Entries are the thread's user messages typed by the user (not automation or agent messages), stripped of the Ultrathink prefix, trailing review comments, legacy context blocks and inline references; consecutive duplicates collapse (composerPromptHistory.ts:73–138). ArrowDown on the last visual line walks forward; past the newest it clears the prompt (140–161). Typing anything ends browsing (ChatComposer.tsx:1586–1588).

### 5.8 Prompt text formats

The stored prompt is markdown:

- file mention `[name](path)` (packages/shared/src/composerTrigger.ts:34–37);
- context reference `[label](t3-context://v1/<kind>/<id>)`, images prefixed `!` (packages/shared/src/composerContextReferences.ts:16–17, 56–58, 138–146);
- skill `$name`; model `model:<instance>/<model>` (packages/client-runtime/src/composerModelMentions.ts:48–50);
- citations serialized by `serializeAssistantCitation`.

Copy and cut write `text/plain` and, when references are selected, a context fragment (`COMPOSER_CONTEXT_CLIPBOARD_MIME` + HTML) so pasting into another composer brings the records (ComposerPromptEditorTiptap.tsx:1792–1818).

### 5.9 Attachments and paste

- Classification (composerAttachmentFiles.ts:106–123): GIF, HEIC/HEIF, JPEG, PNG, WebP → image (a `.gif` `.jpg` `.jpeg` `.png` `.webp` name is trusted when the MIME type is empty or `application/octet-stream`); SVG and DNG → file; any other non-image MIME → file; other `image/*` previewable in the workspace → file, else rejected ("'<name>' is not a supported image type. Attach GIF, HEIC, HEIF, JPEG, PNG, or WebP images.", useComposerAttachmentIntake.ts:186). A paste is taken over by the attachment path when it holds an image, an unsupported image, SVG or DNG, or files without text (composerAttachmentFiles.ts:187–210).
- Limits (packages/contracts/src/chatAttachment.ts:13–17): 100 attachments per message, 10 MB per image (images are compressed to fit), 80 MB of images, 50 MB per file before the server reports its own limit, 120,000 prompt characters.
- **Pasted images become chips only when the prompt already has text** (other than inline references), or the paste replaces a selection, or a question is open, or a project must be chosen; otherwise the image is a tile only. Every standalone image is a tile either way (useComposerAttachmentIntake.ts:143–148, useComposerImageGroups.ts:38). Files always get a chip at the caret and leave the file list while the chip exists (ChatComposer.tsx:673–689).
- Images show as "Preparing" tiles while they compress; a likely duplicate prepares without a tile; failures set a thread error ("'<name>' could not be read as an image." / "'<name>' is too large to attach, even after compression.") (useComposerAttachmentIntake.ts:247–340, 281–282).
- Removing an image referenced in the text asks first: "Remove <name> from the message? It is referenced in your text; removing it also removes every reference." (343–369).
- A paste of 32 KiB or more (characters or UTF-8 bytes), or one that would exceed the character limit, becomes `pasted-text.txt` (then `pasted-text-2.txt`, `-3` …) with the info toast "Large paste attached as <name>" / "<size> · Use ⌘⇧V to keep a large paste inline." (Ctrl+Shift+V off macOS) (useComposerAttachmentIntake.ts:233–242, 371–446; textPaste.ts:1–58). When it cannot be attached and would overflow the limit, the paste is refused with "Pasted text is too large for this message" / "Remove some text or an attachment, then paste again." or "Pasted text is too large to attach" / "Reduce the clipboard contents or save a smaller excerpt as a file." (409–439). ⌘⇧V / Ctrl+Shift+V arms a one-second window that pastes text inline and pastes images without chips (ChatComposer.tsx:1041–1087, 2686–2712).
- Folders dropped or pasted insert `@`-style file links at the end, local servers only ("Folders can only be referenced on a server running on this computer") (2645–2684). Dragged threads become thread chips; threads from unlinked machines are imported as transcript files with a loading toast (2811–2859).
- Send waits: "Still compressing a pasted image." / "Send again once its thumbnail appears." and "Still bringing attached context into this message." / "Send again once its chip resolves." (2125–2145).
- Sizes read "N KB" (ceiling, at least 1) or "N.N MB"; progress "NN%" (packages/client-runtime/src/state/attachments.ts:196–200; lib/attachmentUploadState.ts:48–51).

### 5.10 Drafts

- One draft per target key: the scoped thread key, or the draft id (composerDraftStore.ts:1478–1483). Drafts survive thread switches and reloads: localStorage `t3code:composer-drafts:v1`, schema version 9, writes debounced 300 ms and merged with other tabs' changes (118–232). Images persist as data URLs; ones that do not fit are flagged and get the warning badge (ChatComposer.tsx:1512–1556).
- A draft has user content when any of prompt text, images, files, persisted attachments, contexts or reply-to exists (composerDraftStore.ts:483–505); that decides whether picking another project in the hero moves the draft or opens that project's draft.

### 5.11 Stash

- Mod+S with content stashes the whole draft (prompt, images, files, contexts, reply-to) and clears the composer; the tab pulses. With no prompt text, images or files (contexts alone do not count) it restores the only entry (unless that entry is still saving images), else it toggles the drawer. That happens with several entries and also with none, when the drawer shows the single "Nothing stashed yet…" row although the tab itself is hidden (useComposerStash.ts:495–504). With a question open it toggles the drawer; it does nothing while the command palette is open, a checkpoint revert runs or a project must be chosen. Files must be uploaded and present first ("Wait for file uploads before stashing this prompt", "Attach dropped files again or remove them before stashing"), and context still arriving refuses with "Still bringing attached context into this message." / "Stash again once its chip resolves." (useComposerStash.ts:461–520, 696–709).
- Up to 20 entries, newest first, in localStorage `t3code:prompt-stash:v2` (promptStashStore.ts:11–15).
- Restore appends to the current prompt after a blank line and re-imports records; entries from another environment refuse with a toast (useComposerStash.ts:161–247).
- Drawer keys (window capture): ArrowUp/Down move the highlight (wrapping), Enter restores, Mod+Backspace deletes, Escape closes; a pointer down outside closes; it also closes when the prompt changes or a menu opens (ComposerStashMenu.tsx:59–112, useComposerStash.ts:685–692). Snippets collapse whitespace and cut at 90 characters with "…"; entries without text read "(N images)" / "(N files)" / "(N attachments)" (singular for 1) or "(empty)" (ComposerStashMenu.tsx:16–29). The drawer opens with the newest entry highlighted; a mouse move or focus on a row highlights it; arrow keys scroll the row into view and move focus only if focus is already inside the drawer; Enter on the row's delete button deletes instead of restoring (59–112). The drawer only takes keys whose target is the page body, the stash tab, the drawer or the composer editor. Row times use the relative format in §2.11.

### 5.12 Agent questions

- One question at a time; Previous / Next in the footer; "i/n" in the header (ComposerPendingUserInputPanel.tsx:229–234).
- Single-select: a click selects and auto-advances after 200 ms; multi-select toggles. Digits 1–9 pick options (document keydown) only when focus is not in an input, textarea or contenteditable, no Ctrl/⌘/Alt is held, the question is expanded and answerable, and the thread is not a background or side-chat pane (130–177). Since focus normally sits in the composer editor, the digits work after focus moves out (for example onto an option).
- The editor holds the custom answer; a selected option is used when the answer is left blank. Literal questions (`initialAnswer`) edit raw text without chips or triggers (ChatComposer.tsx:614, 974–986).
- The header row collapses the question to one line; dismissible questions show a dismiss control (195–256).
- Attachments in answers need server support; terminal output cannot be added ("Terminal output can't be added to a question answer.") (ChatComposer.tsx:3085–3096).

### 5.13 Approvals and the typing guard

- A pending approval turns the next send into feedback that declines it (§5.2, §5.4).
- Typing guard: a request that arrives while the user is typing (editor focused, a keystroke within 1.5 s) is held back from the timeline and the composer until the user stops, blurs or sends, so an approval is never answered by a message typed for something else (composerTypingGuard.ts:1–60, ChatComposer.tsx:589–613).

### 5.14 Footer layout

- The form width is measured on every resize. Under 620 the footer is compact: while voice input is active the attach button, the controls and the compact toggle hide and the voice pill takes the row beside Send. With a question open, under 780 the primary actions go compact ("Next" / "Submit", icon-only Previous) (composerFooterLayout.ts:1–42; ChatComposer.tsx:1174, 1494–1510, 1838, 3636–3698).
- The controls row measures natural widths and, as space runs out, first makes the mode block icon-only, then hides it into "…"; it promotes back only with 1 px slack to avoid flapping (composerFooterLayout.ts:88–125).

### 5.15 Draft hero

- Shown for a local draft with no timeline entries, not working, not docked, and no worktree setup card; a pending background submission keeps it (ChatView.logic.ts:303–323).
- The heading reads inline ("What should we build in <project>?") at ≥1024 px when its measured one-line width fits, else stacked (DraftHeroProjectHeading.tsx:17–30, 48–58). Windows under 28 rem tall drop the first part.
- Picking a project with an empty draft opens that project's draft; with content it moves the draft and applies the project's default model unless one was chosen (DraftHeroHeadline.tsx:196–246). "Add project" opens the command palette's add-project flow. "or start without a project" opens the scratch project and focuses the project trigger (377–400).

### 5.16 Focus

- Menus and popovers opened from the composer return focus to the editor's end when they close (`finalFocus`, composerEventScope.ts:12–28); measured after Escape from the permissions menu and the model picker (M). The hero project picker returns focus to its trigger (M).
- Footer buttons prevent focus on pointer down where noted (attach, mic, stash rows, phone buttons), so the caret stays in the editor.
- Type to focus: a printable key pressed outside any editable or interactive element, with no open dialog, menu or popover, is inserted at the end of the prompt and focuses it; pasting works the same (composerFocusRedirect.ts:72–102, useChatShortcuts.ts:72–80).
- F6 area `composer` (03-shell §5.7); thread switches land focus in the editor (its aria-label names the thread).

### 5.17 Phone composer

The collapsed row expands on tap (focusing the end of the prompt) and collapses again a frame after focus leaves the composer and its floating layers (ChatComposer.tsx:2256–2278, 2883–2911). Its send button resumes, expands for compaction, or sends (3412–3418).

## 6. Motion

| What | Property | Duration | Easing | Notes / source |
|---|---|---|---|---|
| surface drag highlight | background-color | 200 ms | `cubic-bezier(0.4, 0, 0.2, 1)` | ChatComposer.tsx:3371 |
| send button | all (scale 1.05, shadows, fill, brightness) | 150 ms | same | ComposerPrimaryActions.tsx:311 |
| Stop | colours | 150 ms | same | 170 |
| Buttons (attach, mic, banner actions) | box-shadow, scale (press 0.97) | 150 ms | same | 02-primitives §6 |
| ComposerControl press | scale 0.97 (not on popup triggers) | immediate | – | ComposerControl.tsx:12 |
| banner / notice card entering | opacity 0 → 1 (`@starting-style`) | 150 ms | `cubic-bezier(0, 0, 0.2, 1)` | ComposerBannerStack.tsx:30–31; none with reduced motion |
| stash tab hover / open | text colour | 200 ms | `cubic-bezier(0.4, 0, 0.2, 1)` | ComposerStashBadge.tsx:43 (`transition-colors duration-200`) |
| stash tab after a stash | row colour = foreground, held 1200 ms, eases back over 200 ms; count = primary 1200 ms then snaps (`step-start`); count enters from `translateY(2px)` opacity 0 over 180 ms ease-out | 1400 ms | – | index.css:230–232, 314–331; ComposerStashBadge.tsx:20–24 |
| strip labels compacting | width and max-width (WAAPI, `fill: backwards`) + opacity | 180 ms | `cubic-bezier(0.32, 0.72, 0, 1)` | BranchToolbar.tsx:434–483; ComposerContextLabel.tsx:28; skipped with reduced motion |
| question panel collapse | height | 200 ms | default | collapsible.tsx:29 |
| question options | colours | 150 ms | default | ComposerPendingUserInputPanel.tsx:278 |
| chip label underline | text-decoration-color | 150 ms | default | ContextChip.tsx:76 |
| waveform bars | transform (scaleY ≥ 0.2), opacity (0.3 + 0.55 × level) | 100 ms | ease-out | VoiceInputPill.tsx:22–38; static 0.4 with reduced motion |
| SnapShot tile arriving | opacity 0 → 1, scale 0.95 → 1 | 300 ms | `cubic-bezier(0.32, 0.72, 0, 1)` | ComposerAttachmentStrip.tsx:125–128 |
| spinner | rotation | 1 s linear infinite | – | paused while not visible |
| draft hero → thread | FLIP translate of the lane | panel animation duration (default 0 ms = off; up to 400) | `cubic-bezier(0.4, 0, 0.2, 1)` | useDraftHeroLayoutTransition.ts:46–55; contracts settings.ts:155–164 |
| phone collapse / expand | card height, prompt and action translate, controls fade with a 4 px drift (arrivals delayed half the duration when expanding) | panel animation duration | `cubic-bezier(0.32, 0.72, 0, 1)` | useComposerRestingTransition.ts:5–7, 196–330 |
| phone route change | view transition (`t3-mobile-composer`, `t3-mobile-draft-headline`) | panel animation duration | – | draftHeroTransition.ts:38–93 |
| tooltips, menus, popovers | per 02-primitives §6 | | | |

## 7. Gaps

- **No live typing.** Menus, chips, rich text, attachments, the stash, the question drawer and multi-line growth are P. Runtime behaviour (debounce timing of searches, highlight order, chip parsing) is S.
- **No live Stop, steer or queue.** The read-only client never sees Stop and cannot send; the running thread showed the steer icon only on a clone. Enabled send, hover and pressed looks are P.
- **No live approval or question.** None was pending during the session. The timeline's approval row belongs to the timeline document.
- **Voice input** states are P; the setup dialog (`VoiceInputSetup`) is not covered.
- **Model picker popup** content is out of scope; only its placement and the trigger's open state were measured.
- **SnapShot captures**, the composer code-block node view and language picker, citation comment editor, model and PR chip popovers, skill popovers and file preview dialog were not opened (S only).
- **Notice details popover** (overflowing banner text) and **ProviderSubagentBar** (child threads) are S only.
- **Phone transitions** and the hero FLIP are off by default (panel animation 0 ms) and were not exercised.
- **Themes** (`data-theme-id`) and contrast settings were not switched; formulas are in 01-tokens.
- **macOS** renders SF Pro: the cap-trimmed label boxes, the Kbd glyphs and the hero's inline-fit check depend on font metrics.
- The data-attribute chip highlight (`data-composer-chip-selected`) exists in the class string but nothing sets it in this build.
- **Triage hero** ("What went wrong?", §2.16) needs a triage session; S only.
- **Hero project picker popup.** The verification pass did not re-measure its geometry (§3.17) or the strip's `data-pressed` open look. It ran under a load average of 30–90 and the browser wedged once.

## 8. Port notes for Go/Gio

What this replaces in tern:

- `app/ui/screens.go`: `hero` (481–550, doc comment included; the func starts at 484), `composerArea` (1071–1124), `approval` (1125–1165), `questions` (1166–1215), `queue` (1216–1248), `composer` (1249–1388), `errorBanner` (1389–1402), `footPick` / `footSep` (1403–1429), `thumb` (1430–1446). The current composer is a 10-radius bordered box with a 220 dp editor; T3's is a 16-radius glass card with an 8% outline, a 208 px editor cap and a separate footer row.
- `app/ui/chips.go` (chip runes and drawing) and `app/ui/thumb.go` (pasted thumbnails): replace with §2.12 chips and §2.13 tiles.
- `app/ui/input.go` `editor` (369–~520): the prompt editor needs rich text, atoms, IME-safe Enter and the key table in §5.5.
- `app/ui/model.go` Scene fields `Composer` … `SuggestHead` (259–278) and `Access`, `Projects`, `Notice`, `Error`: grow into a composer view model (below).
- `app/core/scene.go` `composerScene` (1024–1086), `traits` (1087+), `runtimeModes` (505–518): **tern's mode labels are wrong** ("Ask every time", "Accept edits", different descriptions) and its list runs in reverse (full-access first); use T3's four labels, icons, descriptions and order (Supervised, Auto-accept edits, Auto, Full access) from §2.7 verbatim.
- The new-thread hero has a third heading tern lacks: the triage hero "What went wrong?" (§2.16).

Drawing:

1. Paint the card in three layers like §2.3: glass fill (opaque fallback `#ffffff` / `#111111` is what Chrome paints over the canvas anyway: `#ffffff`, `#0e0e0e`), then content, then the 1 px outline on top (`#00000014` / `#ffffff0d`, painted `#ebebeb` / `#1a1a1a`) with the dark 1 px top highlight. Light adds `0 1 2 #0000000d` below. Gio has no backdrop blur; over the timeline, approximate with the 80% fill (the blur is visible only where text scrolls under the card).
2. The strip is a separate `#fcfcfccc` / `#000000cc` band under the card, inside the same 736 lane.
3. Attached surfaces (banners, question, stash drawer, command drawer) are 44 px narrower, tuck 17 px under the card, have 16 px top radii only on the first one and mask their bottom 16 px. Draw them before the card so the card covers the tuck; paint the dark seam gradient (§3.9) above the card's top edge.
4. Text-box trimming: T3 trims labels to cap height and pads 6 px top and bottom. In Gio, centre on the cap height (Inter cap 0.727 em; use the font's real cap height for SF Pro) instead of the line box.
5. Focus: draw the 2 px inset ring 1 px inside the control's edge (1 px background, then 2 px `--ring`).
6. Send artwork: reuse the stage-art renderer from 03-shell §3.4 with the viewBox shifted (96 nightly, 64 dev), scaled ×⅓ (`xMinYMin slice` of the 96-unit-tall art onto 32 px, so the button shows art x 96–192 or 64–160), clipped to a 32 px circle; fall back to `--message-action`.
8. Strip labels are bare cap-height boxes (no 6 px padding, §2.5); only the footer's control labels get the padding.
7. Effort bars: implement §3.6 with whole-pixel bars; they are designed for 1x crispness.

Behaviour to port as pure functions (test them like T3's own tests): `detectComposerTrigger`, `composerSubmissionIntentForKey`, `resolveComposerDispatchMode`, `resolveComposerControlsOwnership`, `resolveRestingComposerControlsLayout`, `shouldUseCompactComposerFooter`, `stepComposerPromptHistory` / `recallableComposerPrompt`, `searchSlashCommandItems`, `resolveEffortGlyph` / `effortBarHeights`, `pastedTextDisposition`, `classifyComposerAttachmentFile`, `middleTruncateAttachmentName`, `formatAttachmentSize`, `formatContextWindowTokens`, `formatRelativeElapsed`, `getComposerPromptLengthValidationMessage`, the placeholder priority (§5.2) and the primary-action table (§5.3).

Data tern's core must provide (several are missing today; `t3/types.go` `Provider` has only id, driver, name, enabled, status and models):

- Per provider: `skills[]` (name, displayName, description, shortDescription, path, scope, enabled, userInvocable, userInvocationOnly), `slashCommands[]` (name, description, input hint), `supportedRuntimeModes`, install/auth status, per-cwd workspace skills; per model `shortName`, `subProvider`, option descriptors with `promptInjectedValues` (packages/contracts/src/server.ts:79–130, 217–272).
- Client settings: `sendShortcut`, `followUpBehavior`, `effortDisplay`, `composerRichTextEnabled`, `persistComposerContextStrip`, `showSkillsInSlashMenu`, `snapShotAnimations`, `panelAnimationDurationMs`, font sizes; server settings `defaultRuntimeMode`; resolved keybindings (for labels and chords); managed policy ceilings.
- Environment capabilities: attachment uploads and `maxFileAttachmentBytes`, question attachments, model-catalog refresh, connection phase (for the machine banner), server update state.
- Thread state: phase, active run id, `canInterrupt`, `canSteer` (compaction), resumable runs, pending approvals (kind, detail, options with warnings, feedback support, response capability), pending user inputs (questions with header, text, options with label/value/description/preview, multi-select, custom-answer rules, dismissible), child-thread approvals, queued messages (for editing), the prompt suggestion, the compaction token count, usage-limit reports and fallback model state.
- Searches over RPC: workspace path search for `@`, thread search, pull-request list and detail for `#`, machines and linked environments.
- Local stores tern must own: drafts per target (persisted, multi-window safe), the prompt stash (20 entries), attachment upload queue with progress, prompt history built from the thread's user messages, the typing guard.
- Voice input (optional): transcription environment, model download state, levels.

## 9. Verification

I checked this document adversarially on 2026-11-18 against the same build (`v0.1.10-nightly.20261112.4606`) and the same source tree. The browser had a fresh read-only pairing at 1440×900 @2x (600×900 for the phone row), Linux Chrome with Inter, in both colour schemes. I typed and sent nothing and changed no setting. I opened the permissions menu and the model picker and closed them with Escape, hovered and Tab-focused controls, and moved only between existing threads. I removed the probe buttons after use and left the browser on light. The machine ran at a load average of 30–90 and the tab wedged once and had to be re-paired, so I dropped a few planned checks (§7).

### 9.1 Re-measured live in both themes (M)

Each row was measured in light and in dark, and each matched the document unless the Result column says otherwise.

| # | Value | Light / dark | Result |
|---|---|---|---|
| 1–5 | host `::before` glass fill, backdrop filter, `::after` outline, dark inset highlight, host shadow (draft with strip) | `#ffffffcc` / `#111111cc`; `blur(12px) saturate(1.14)` / `blur(16px) saturate(1.08)`; `#00000014` / `#ffffff0d`; none / `inset 0 1px 0 #ffffff08`; `0 1px 2px #0000000d` / none | ✓ |
| 6–8 | strip fill, backdrop, text | `#fcfcfccc` / `#000000cc`; 12 / 16 blur; `#18181bb3` / `#a1a1a1` | ✓ |
| 9–11 | editor text, caret, placeholder | `#18181b` / `#f5f5f5` (caret the same); `#71717b` / `#a1a1a1`, 14/22.75 | ✓ |
| 12–14 | attach icon, hover background, hover text | `#71717b` / `#a1a1a1`; `#f4f4f5` / `#ffffff0a`; `#18181b` / `#f5f5f5` (icon stays muted) | ✓ |
| 15–18 | model trigger text, suffix, hover background, hover label and suffix | `#18181bcc` / `#f5f5f5cc`; `#71717b` / `#a1a1a1`; `#f4f4f5` / `#ffffff0a`; `#18181b` / `#f5f5f5` | ✓ |
| 19 | mic icon, disabled opacity | `#71717b` / `#a1a1a1`, 0.64 | ✓ |
| 20–21 | strip permissions hover background and text | `#f4f4f5`, `#18181b` / `#ffffff0a`, text stays `#a1a1a1` | ✓ |
| 22–25 | permissions popup fill, border, backdrop, shadow | `#ffffffd5` / `#111111d5`; `#18181b1a` / `#f5f5f51a`; 12 / 16 blur; `0 16px 40px -18px #0000008c` / `0 18px 44px -18px #000000cc` | ✓ |
| 26–32 | permissions heading, group fill, segment default, checked, highlighted, preview label, description | `#71717b` / `#a1a1a1`; `#d4d4d866` / `#ffffff0b`; `#71717b` / `#a1a1a1`; `#d4d4d8b8` + `#18181b` / `#ffffff14` + `#f5f5f5`; `#f4f4f5` + `#18181b` / `#ffffff0a` + `#f5f5f5`; current `#18181b` / `#f5f5f5`, previewing `#71717b`; `#71717b` / `#a1a1a1` | ✓ |
| 33–36 | hero h1, project trigger, its hover underline, subline and its hover | 30/36 400, −0.75, `#18181b` / `#f5f5f5`; trigger 30/500; underline offset 4 in the text colour; subline `#71717b` / `#a1a1a1` → `#18181b` / `#f5f5f5`, underline offset 2 | ✓ |
| 37 | focus ring (attach, model trigger, strip chip) | `inset 0 0 0 1px #fcfcfc` + `inset 0 0 0 3px #1b4ed8` / `#000000` + `#346bf1` | ✓ |
| 38–39 | suggestion Kbd border and text | `#18181b26` / `#f5f5f526`; `#71717b` / `#a1a1a1`; 20 tall, padding 0 6, 1 + 2 px border, radius 4, 12/16 500, `margin-left` 8, `vertical-align` 1px | ✓; dark was P, now M |
| 40–43 | phone (600 px): placeholder, send fill, resting-row colour, resting trigger text | `#71717b` / `#a1a1a1`; `#1b4ed8` / `#346bf1` at 0.64; `#71717bb3` / `#a1a1a1b3`; `#18181bb3` / `#a1a1a1` | ✓; dark was P, now M |
| 44–45 | painted card interior and outline (screenshot pixels) | `#ffffff` / `#0e0e0e`; `#ebebeb` / `#1a1a1a` | ✓; added the dark highlight row `#151515` and the light shadow ramp |

Theme-independent geometry re-measured (M): overlay, stack and shell boxes in both views; draft centring; body padding 16 16 8; editor content box 702 × 45.5 inside a 710 × 53.5 element (min `2lh + 8`, max 208); footer padding 0 8 8; attach 28 at x 9; model trigger 157.75 wide ending at 651 (`min-width` 52, gap 6, label 22.19, suffix −2 px, effort word max 96, provider icon 16 with −2 px margins); hidden overflow wrapper 34 × 28; mic at 659; send 32 at 695; strip 32 tall, padding 8 8 0, gap 4, controls 24 tall with 7 px padding and 12 px icons, labels 8.73; permissions popup 810, 378, 298 × 132 (section 288, segments 67 × 24, group 28, 4 px gap above the chip); model picker popup 554 × 393 at 477, 444 with its right edge on the trigger's and 4 px above it; tooltip 12/16.5 about 4 px above the chip; phone surface 574 × 80 in a 576 × 82 main, prompt row 48 (padding 8 12, gap 8), resting band 32 with controls 28 tall at bottom 8, inset 16, 12/16 text; overlay `padding-top` 8 (6 at 600 px). Tab order on a draft: editor → attach → model trigger (disabled mic and send skipped) → strip workspace → two more strip stops (branch, permissions) → the details card's first control. Escape from the permissions menu and the model picker returns focus to the editor.

Probe buttons (P) carried the source class strings in a fixed overlay and were hovered with the real pointer in both themes. Plain send `#1b4ed8` / `#346bf1`, hover `#3160db` / `#2f60d9` at `scale: 1.05`, the outer `0 1px 0 #ffffff29` highlight (the custom `--inset-shadow-control-highlight` value has no `inset` keyword, so the "outer" in §3.5 is right), disabled `#18181b1f` / `#f5f5f51f` with a muted icon and no shadow; stage-art send shadow `#0000003d` and hover `brightness(1.1)`; Stop `#fb2c36e6` / `#fb414ae6`, hover `#fb2c36` / `#fb414a`. All ✓.

### 9.2 Re-derived from source (S)

All matched unless listed under 9.3: the §1 line ranges for ComposerSurface, ComposerControl, ComposerPrimaryActions, EffortGlyph, ComposerFooterControls, ComposerBanner (every part) and ComposerBannerStack; Stop sizes (`size-8`, 28 at ≥640 only inside the answer row) and its 10 s re-enable; pending labels; send label and status order and the steer/queue tooltip; the stage-art viewBox shifts; the ownership rules (`resolveComposerControlsOwnership`); `ComposerControl` sm/xs classes; effort-bar math (4 levels → 6, 8, 10, 12, width 14); the 620 / 780 footer thresholds; the 400 / 560 banner container breakpoints; the icon column (28, 24 at ≥640) and the notice title's 28/24 leading; the question drawer's paddings, radio and option rules; the stash snippet rules, 20-entry store and keys; the drawer layer's position formula; the trigger detection; `composerSubmissionIntentForKey` and the Mod+Enter → Mod+Shift+Enter rewrite; prompt-history guards; draft storage key, version 9 and 300 ms debounce; the 1.5 s typing guard; the 32 KiB paste threshold; attachment limits and classification; the attachment size format; the placeholder priority; the prompt-suggestion conditions; the strip visibility rule and its 16 px hysteresis; the 180 ms strip-label motion; the collapsible's 200 ms height; the SnapShot arrival motion; the waveform formulas; the stash pulse keyframes; tern's function locations in `ui/screens.go`, `ui/input.go`, `ui/model.go` and `core/scene.go`.

### 9.3 Errors found and fixed

1. In §3.21 the approval "More" ellipsis was listed at 16 / 12. It is Button outline `icon-xs`, so 14 at ≥640 and 16 below, and the inline presentation that child approvals use shows the text "More" with no icon.
2. In §3.3 `white-space: break-spaces` was attributed to index.css:2451–2453. It comes from Tiptap's injected ProseMirror CSS beating the `whitespace-pre-wrap` utility. Those lines only zero the paragraph margin.
3. In §2.9 the banner feed order put project clone before usage limits. The source runs limit recovery and extension status, then usage limits, then project clone.
4. In §4 the footer permissions chip hid only when the mode matched the default. It also hides while the new-thread strip names the permissions (`matchesDefault: accessMatchesDefault || permissionsShownInStrip`), and it is icon-only in the collapsed phone row.
5. In §3.5 the Submit pill was 14/500 at every width. It is 16/500 below 640 (`text-base sm:text-sm`), with a 6 px gap.
6. In §5.3 Stop trailed Send only while a stop was "pending or landed". Any stop phase keeps it there, timed out included, and the Stop-only layout also covers a pending or timed-out stop.
7. In §5.6 the `#` trigger needed a letter after `#`. A bare `#` opens the menu with an empty query, and the first character after `#` must be a letter or digit.
8. In §3.1 the hero wrapper carried the 16 px bottom padding. Its inner div does, and the padding drops to 0 while the stash tab shows.
9. In §8 tern's `hero` range read "481–~560". It is 481–550, with the doc comment from 481 and the func from 484.

Smaller corrections that changed no value: the Shell citation now starts at the element (ComposerSurface.tsx:11, was 13), and the new citations added in this pass were each checked against the source line they name.

### 9.4 Omissions filled

`data-slot=composer-shell` / `composer-host` selectors (§2.3); the label padding's `supports-[text-box]` gate and the unpadded strip labels (§2.5); the permissions section's preview, Left/Right keys and close-on-pick (§2.7); the banner `Dot` part and the separator's zero margin inside Content (§2.9); the notice-details button labels, popover width and anti-flap rule (§2.9); the option description only showing when it differs from the label (§2.10); the stash restore and delete details and the relative-time format (§2.11, §5.11), plus Mod+S opening an empty drawer; the 21 + "…" + 14 middle truncation (§2.12); the waveform's right-to-left history and the mic's disabled conditions (§2.14); the triage hero "What went wrong?" (§2.16); the collapsed phone row's missing footer, send-disabled conditions and pointer-down focus rule (§2.17); the ligature reset (§3.3); the stage-art slice scaling (§3.5, §8); the compact toggle's aria-label and wrapper, and the answer row's gap (§3.5); the effort word fallbacks "<label> On/Off", "Ultrathink" and `thinking` (§3.6); the branch control's content-sized width (§3.7); the tooltip chord following the resolved keybindings (§3.20); the digit-key conditions (§5.12); the stash tab's 200 ms colour transition (§6); the dark highlight pixel and light shadow ramp (§3.2); tern's reversed mode order and the missing triage hero (§8).

### 9.5 Privacy

A scan of the document for thread titles, project, branch and machine names, paths, emails, IDs and tokens found none. This pass describes the identifying values it saw live generically: the hero's project name, the details card's machine name, the branch name and a customised keybinding. Screenshots stayed in a private folder outside the repo.
