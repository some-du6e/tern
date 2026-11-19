# 02 · UI primitives (`apps/web/src/components/ui`)

Ground truth for porting T3 Code's primitive components to tern (Go + Gio). It covers the
wrappers T3 builds on Base UI (`@base-ui/react` 1.9.0) plus the small presentational pieces
in `components/ui/`: buttons, menus, popovers, tooltips, form controls, overlays, toasts,
scroll areas and loading/empty states. Feature-level components that *use* these primitives
(sidebar, composer, command palette, settings pages) belong to other docs; they are mentioned
here only where they were the live specimen for a measurement.

**Build measured.** T3 Code `v0.1.10-nightly.20261112.4606`, web client served by the desktop
runtime, Chrome on Linux, viewport 1440×900 CSS px at devicePixelRatio 2, default theme (no
`data-theme-id`), appearance contrast at defaults (`--appearance-contrast-base: 100%`,
boost 0%).

**Value markers used in every table**

| Marker | Meaning |
| --- | --- |
| **M** | Measured live on a real element of the running app (`getComputedStyle`, colours painted on a canvas and read back as sRGB). |
| **P** | Measured live on a *probe*: an element inserted into the running page with the exact class string the component produces. The string was built with the app's own `cn()` (tailwind-merge) and `buttonVariants` imported from the shipped bundle (`/assets/utils-ST3nadSl.js`, exports `t` and `Vr`); other cva configs were evaluated from source with the same merge (validated: 252/252 button variant×size combinations identical to the shipped cva). Chrome computed the result from the shipped CSS, so P values are as real as M values, minus Base UI's runtime attributes, which were set by hand. |
| **S** | Source-derived (TSX, `index.css`, compiled CSS, or the Base UI code inside the shipped bundle). |

**Conventions**

* Colours are sRGB hex as Chrome paints them; 8-digit hex is `#rrggbbaa`. Translucent colours
  composite over whatever is beneath; low-alpha values carry ±1–3 units of rounding in the
  RGB part (the canvas stores premultiplied 8-bit colour), so where a formula (token + alpha)
  is given it is authoritative.
* Tailwind spacing unit is `--spacing: 0.25rem` = 4px; `1rem` = 16px.
* `sm:` = viewport ≥ 640px, `md:` = ≥ 768px. **All measured sizes are the ≥ 640px values**
  (the desktop case). Values below 640px are given as S where they differ.
* `pointer-coarse:` rules (44×44px minimum hit areas via `::after`) only apply on touch
  screens; tern on desktop can ignore them.
* "Light / dark" columns: light first, then dark, in the root token scope (popups are
  portalled to `<body>`, so they always use root tokens even when opened from the sidebar).
* Font metrics: T3's sans stack is `-apple-system, BlinkMacSystemFont, "Inter Variable",
  "Segoe UI", system-ui, sans-serif`, so this Linux Chrome rendered **Inter** while macOS
  renders **SF Pro**. Every font size, weight and line height below is set in CSS and
  holds on both; text *widths* (and therefore auto-width boxes such as buttons with labels)
  were measured with Inter and will differ by a few px with SF Pro.

---

## Contents

1. [Scope and sources](#1-scope-and-sources)
2. [Structure](#2-structure)
3. [Measurements](#3-measurements)
4. [States](#4-states)
5. [Behaviour](#5-behaviour)
6. [Motion](#6-motion)
7. [Gaps](#7-gaps)
8. [Port notes for Go/Gio](#8-port-notes-for-gogio)
9. [Verification](#9-verification)

---

## 1. Scope and sources

Paths are relative to the T3 repo (`apps/web/src/...` unless noted). Line ranges are for the
pinned tag.

### 1.1 Primitive sources (`apps/web/src/components/ui/`)

| File | Lines | What was taken |
| --- | --- | --- |
| `button.tsx` | 1–139 | `buttonVariants` base (9), sizes (16–42), variants (43–70), `Button` render/type rule (83–99), `inlineButtonVariants` + `InlineButton` (103–139) |
| `kbd.tsx` | 1–38 | `Kbd` default/outline (5–26), `KbdGroup` (28–36) |
| `badge.tsx` | 1–59 | base (8), sizes (15–23), variants (24–37) |
| `menu.tsx` | 1–414 | `MenuPopup` (21–74), `MenuSearchInput` (77–112), `MenuItem` (118–149), `MenuItemLabel` (151–162), `MenuCheckboxItem` (164–222), `MenuRadioItem` (228–248), `MenuRadioItemIndicator` (250–265), `MenuGroupLabel` (267–285), `MenuSeparator` (287–295), `MenuShortcut` (297–308), `MenuItemDetail` (312–320), `MenuSubTrigger` (326–353), `MenuSubPopup` (355–379) |
| `closing-positioner.tsx` | 1–18 | freezes popup placement while it closes |
| `popover.tsx` | 1–154 | width map (19–24), padding map (26–30), `PopoverPopup` (32–130), `PopoverTitle` (136–144) |
| `tooltip.tsx` | 1–164 | `COLLISION_PADDING` (16), scroll-dismiss tracker (23–63), `Tooltip`/`TooltipTrigger` (65–103), `TooltipPositioner` (105–120), `TooltipPopup` (122–162) |
| `input.tsx` | 1–80 | input classes (21–32), wrapper (59–77) |
| `input-group.tsx` | 1–120 | `inputGroupVariants` (7–29), addon variants (47–82), addon mouse handling (84–114) |
| `textarea.tsx` | 1–60 | wrapper (19–30), textarea + `rows` min-height (31–55) |
| `number-field.tsx` | 1–95 | root (6–25), group (27–41), decrement/increment (43–77), input (79–93) |
| `select.tsx` | 1–218 | trigger variants (12–36), `SelectTrigger` (38–59), `SelectButton` (61–86), `SelectValue` (88–96), `SelectPopup` (98–165), `SelectItem` (167–192), group label (198–206) |
| `combobox.tsx` | 1–287 | `ComboboxInput` (29–105), `ComboboxSearchInput` (107–125), `ComboboxPopup` (135–186), `ComboboxItem` (188–217), `ComboboxEmpty` (219–230), `ComboboxList` (232–242), `ComboboxStatus` (259–270) |
| `autocomplete.tsx` | 1–255 | input (10–74), popup (76–121), item (123–136), group/label (138–169), empty (181–192), list (194–204), clear (206–219) |
| `switch.tsx` | 1–39 | root (12–26), thumb (27–34) |
| `toggle.tsx` | 1–53 | `toggleVariants` (6–36) |
| `toggle-group.tsx` | 1–78 | group (14–50), context-aware item (52–76) |
| `separator.tsx` | 1–19 | all |
| `scroll-area.tsx` | 1–102 | virtualized fade helper (5–20), `ScrollArea` (22–77), `ScrollBar` (79–100) |
| `spinner.tsx` | 1–40 | sizes/tones (7–21), element (23–38) |
| `dialog.tsx` | 1–258 | retained content (16–34), initial focus (44–56), backdrop (68–84), viewport (86–97), popup (99–156), header/footer/title/description/panel (166–237) |
| `dialog-styles.ts` | 1–22 | backdrop, popup, media and mobile-sheet class constants |
| `alert-dialog.tsx` | 1–165 | backdrop (27–36), viewport (38–49), popup (51–88), header (90–101), footer (103–122), title (124–132), description (134–145) |
| `sheet.tsx` | 1–172 | backdrop (18–29), viewport (31–55), popup (57–127), header/title/description (129–160) |
| `toast.tsx` | 1–821 | buffered managers (77–106), icons (110–116), error clamp (118–127), close orb (129–143), copy button (145–167), expandable (169–294), body layout (308–445), visible-time auto dismiss (477–554), `ToastProvider`/`Toasts` (556–712), anchored toasts (714–810) |
| `toast.logic.ts` | 1–115 | action visibility (5–14), collapsed content (16–22), stack layout (40–90), thread scoping (92–115) |
| `toastHelpers.ts` | 1–50 | `stackedThreadToast` |
| `anchoredCopyToast.ts` | 1–30 | "Copied!" (timeout 1000ms) / "Failed to copy" |
| `collapsible.tsx` | 1–43 | trigger (9–17), panel (19–36) |
| `collapsible-section-header.tsx` | 1–54 | tones (5–9), header (11–48), status (50–54) |
| `empty.tsx` | 1–122 | sizes (5–10), root (12–28), header (30–38), media (40–88), title (90–94), description (96–107), content (109–120) |
| `skeleton.tsx` | 1–26 | all |
| `panel-tab-close-button.tsx` | 1–40 | all |
| `middle-truncate.tsx` | 1–134 | component (6–58), `fitHead` (64–90), `splitForMiddleTruncate` (92–134) |
| `middle-truncate.test.ts`, `middle-truncate.browser.test.tsx` | 1–29, 1–26 | expected splits and refit behaviour |
| `draft-input.tsx` | 1–14 | all |
| `label.tsx` | 1–22 | label typography (used by every form row measured) |

### 1.2 Supporting sources

| File | Lines | What was taken |
| --- | --- | --- |
| `apps/web/src/index.css` | 91–152 | `:root` geometry: `--control-radius` (103), `--z-sheet: 46` (118), glass blur/opacity/saturation (119–121) and their dark values (145–151), `--workspace-topbar-height: 52px` (123) |
| `apps/web/src/index.css` | 170–175 | `--font-sans`, `--font-mono` |
| `apps/web/src/index.css` | 177–294 | custom text sizes `2xs`–`5xs` (183–190), `--ease-drawer` (195), `--animate-skeleton` (223), colour utility → variable mapping (233–288), radii (289–294) |
| `apps/web/src/index.css` | 295–313 | `@keyframes skeleton` |
| `apps/web/src/index.css` | 364–372, 425–467 | `surface-glass`, `dialog-glass`, `dialog-backdrop`, `dropdown-glass` utilities |
| `apps/web/src/index.css` | 511–524, 580–584 | `virtualized-scroll-fade`, `visible-animate-spin` |
| `apps/web/src/index.css` | 714–737 | base layer: default border colour, `outline-ring/50`, global `outline-offset: -2px`, popup focus reset (723–730), body text colour |
| `apps/web/src/index.css` | 1105–1111 | `.no-transitions` (theme switch) |
| `apps/web/src/index.css` | 1113–1278 | light tokens (1113–1186), dark tokens (1187–1228), `[data-app-sidebar]` overrides (1234–1278) |
| `apps/web/src/index.css` | 1600–1712 | `--contrast-*` derivations |
| `apps/web/src/index.css` | 1728–1732 | lucide default stroke 1.75 |
| `apps/web/src/index.css` | 1851–1866, 2276–2282 | native `::-webkit-scrollbar`, markdown table scrollbar override |
| `apps/web/index.html` | 390–408 | boot styles: `body { color: #262626 }`, `html.dark body { color: #f5f5f5 }` (unlayered, so they beat Tailwind's base `body` colour) |
| `apps/web/src/hooks/useCommitOnBlur.ts` | 1–36 | DraftInput commit semantics |
| `apps/web/src/lib/visibleAnimation.ts` | 1–63 | Spinner play/pause |
| `apps/web/src/lib/observeResize.ts` | 59–71 | shared ResizeObserver used by MiddleTruncate |
| `apps/web/src/hooks/useTheme.ts` | 42, 306–345, 436–442, 466–480 | system-scheme following (`.dark` class toggle, transitions suppressed during the swap) |
| `apps/web/src/routes/__root.tsx` | 134–138 | `ToastProvider` + `AnchoredToastProvider` mount at the root (no global `TooltipProvider`) |
| `apps/web/package.json`, `pnpm-lock.yaml` | 17; 3265 | `@base-ui/react` 1.9.0 |
| `apps/web/src/components/chat/PanelLayoutControls.tsx` | 93–123 | header panel toggles are `Toggle variant="ghost" size="sm"` |
| `apps/web/src/components/pullRequest/PullRequestListFilters.tsx` | 109 | PR-list search `InputGroup` overrides its input to `**:[input]:h-9 sm:**:[input]:h-8` |
| `apps/web/src/components/ProjectScriptsControl.tsx` | 178–234, 333–363 | script-actions menu: `highlightItemOnHover={false}`, rows that contain a native tabbable edit `Button` |
| `apps/web/src/components/Sidebar.tsx` | 3202–3206, 3324–3328 | sidebar thread list `TooltipProvider delay={150} closeDelay={0} timeout={400}` |

### 1.3 Shipped artefacts (runtime `0.1.10-nightly.20261112.4606/client/assets/`)

| File | Used for |
| --- | --- |
| `main-DcchvbMF.css` | compiled Tailwind theme (`--default-transition-duration: .15s`, `--default-transition-timing-function: cubic-bezier(.4,0,.2,1)`, `--ease-out`, shadow utilities, `.dropdown-glass` etc.) |
| `utils-ST3nadSl.js` | `cn` (export `t`), `buttonVariants` (export `Vr`), Base UI Menu/Tooltip/Dialog focus manager/hover/typeahead code: tooltip delay 600 (`c??C??600`), `closeDelay` 0, `closeOnClick` true; Menu `loopFocus` true, `closeParentOnEsc` false, `highlightItemOnHover` true; item `closeOnClick` true, checkbox/radio item `closeOnClick` false; submenu trigger `delay:100`, `closeDelay:0`, `restMs:100`, `openOnHover:true`, click handler `ignoreMouse:true` + `stickIfOpen:true`, `safePolygon({blockPointerEvents:true})` (40ms stall timer, 0.1px/ms speed floor); list navigation `loopFocus:false` default, Menu passes `disabledIndices: []` and items use `useButton({focusableWhenDisabled:true})`; typeahead hook default `resetMs:750`, **but the Menu root passes `resetMs:500`** (Select keeps 750); focus manager defaults `initialFocus:true`, `returnFocus:true`, `modal:true`, `closeOnFocusOut:true`, but `MenuPopup` passes `modal` only for context menus; modal menus render an inert backdrop with a hole over the trigger and lock scroll unless hover-opened; tooltip trigger `handleClose: safePolygon()` unless `disableHoverablePopup`; `TooltipProvider` `timeout:400` |
| `popover-thRzkULm.js` | Popover root `modal:false`; trigger `openOnHover:false`, `delay:300`, `closeDelay:0` |
| `select-g9ombDFN.js` | Select root `modal:true`, `alignItemWithTrigger:true` (disabled for touch opens); typeahead uses the 750ms hook default and stays enabled while closed (`enabled: !disabled && (open \|\| !readOnly && !multiple)`), where a match sets the value directly |
| `combobox-Bdk1oQ6r.js` | Combobox `modal:false`, `openOnInputClick:true`, `autoHighlight:false`, `loopFocus:true`, `highlightItemOnHover:true`, `fillInputOnItemPress:true` |
| `dialog-DU-F6l5H.js` | Dialog `modal:true`, `disablePointerDismissal:false`; AlertDialog forces `modal:true`, `disablePointerDismissal:true` and `role:"alertdialog"` |
| `toast-BKQ7E-dN.js` | Toast provider `timeout:5e3`, `limit:3`; managers `Fn` (stacked, export `i`) and `In` (anchored, export `r`); `Toast.Positioner` defaults `side:"top"`, `align:"center"`, `sideOffset:0` (T3 passes 4), `collisionPadding:5`; root `role` `dialog` (`alertdialog` for `priority:"high"`), `tabIndex:0`, `aria-modal:false`, limited toasts `inert`; viewport hotkey F6; Escape closes the focused toast; swipe constants 1px (drag start), 40px (dismiss), 10px (reverse cancel), exponent 0.5 (off-axis damping) |
| `scroll-area-CSL1OAMi.js` | `data-scrolling` cleared 500ms after the last scroll; minimum thumb size 16px |
| `main-BMVwsd-_.js` | Switch (role, `useButton` semantics), ToggleGroup `loopFocus:true` / `orientation:'horizontal'`, NumberField `step:1`, `smallStep:.1`, `largeStep:10`, auto-repeat 400ms then every 60ms (touch/pen presses first wait 50ms and must move < 3px; `contextmenu` is suppressed while held) |

### 1.4 Live specimens

Measured in the running app (both schemes, using `set media light|dark`, which the app
follows through `prefers-color-scheme`): header panel toggles (Toggle), the right-panel
"script actions" menu (menu popup, items, keyboard), the pull-request list's filter menu
(sub-triggers, separators, radio group, group label, checkbox items, `MenuSearchInput`,
submenu placement and flipping), its search field and toolbar buttons (InputGroup, outline
Buttons), sidebar header tooltips, the composer model picker (Popover), the composer
workspace select (Select), the composer branch picker (Combobox), the "Add action" dialog
(Dialog, Label, disabled Input/Textarea/Switch, footer Buttons), Settings rows (Switch,
Select trigger, Kbd), the sidebar section headers (CollapsibleSectionHeader), a sidebar
branch label (MiddleTruncate), and toasts raised locally through the shipped toast managers
with placeholder text. Screenshots are kept outside the repo.

---
## 2. Structure

### 2.0 Shared conventions

* **`data-slot`** names every part (`button`, `menu-popup`, `tooltip-trigger`…). When a
  primitive is passed as `render` to a Base UI trigger, the trigger's props are merged last,
  so a Button rendered as a tooltip/menu/popover trigger carries `data-slot="tooltip-trigger"`
  (or `menu-trigger`, `popover-trigger`) instead of `button` (M). Style hooks therefore must not
  depend on `data-slot="button"` alone.
* **Pseudo-element highlight ("lip").** Bordered controls (outline Button, Input, Select
  trigger, Textarea, NumberField group, Toggle outline, tooltips, popovers, Empty icon) add
  `::before { position:absolute; inset:0; border-radius: <outer radius − 1px>; box-shadow:
  0 1px rgb(0 0 0 / 4%) }` in light and `0 -1px rgb(255 255 255 / 6%)` in dark. The
  pseudo-element covers the padding box (inside the border) and has no fill, so only its
  unblurred shadow shows. That shadow is a 1px strip just outside it, lying on the control's **bottom**
  border in light (a slightly darker bottom edge) and on the **top** border in dark (a brighter
  top edge), following the corner radius. Being positioned, it paints over the border. It is
  removed when the control is disabled, focused, invalid or pressed (per component, see §4).
* **Icons** are lucide SVGs. A global base rule turns lucide's default `stroke-width="2"` into
  **1.75** (`index.css:1728–1732`, M on every icon measured). Container rules size direct or
  descendant SVGs that have no `size-*` class of their own and colour those with no `text-*`
  class; an icon with its own `size-*`/`text-*` class opts out.
* **Default border colour** for any element with a `border` but no colour class is the border
  token (`* { border-color: var(--contrast-border) }`, `index.css:715`), and every element
  gets `outline-color: ring/50` and `outline-offset: -2px` (`index.css:715–721`).
* **Inherited text colour.** Elements without a text-colour class inherit the body colour,
  which comes from the unlayered boot style in `apps/web/index.html:398–408`: **#262626**
  light / **#f5f5f5** dark (M), not the foreground token (#18181b). Affected here: Button
  `link` variant, Empty title, popup containers (their items set `text-foreground`).
* **cva + tailwind-merge.** Variant classes are concatenated (base, then each variant key in
  declaration order, then compound variants, then `className`) and merged with
  tailwind-merge, so later classes in the same group replace earlier ones. Modifier-scoped
  classes (`sm:…`) are separate groups, which is why several "small" sizes keep the base's
  `sm:` font/icon sizes on desktop (called out in §3).

### 2.1 Button (`button.tsx`)

```
<button type="button" data-slot="button" class="buttonVariants(variant,size)">   (useRender: `render` swaps the element; then no `type`)
  ::before   absolute inset-0, pointer-events none, radius calc(var(--control-radius) - 1px)  → top-lip highlight (outline/destructive-outline)
  ::after    pointer-coarse only: absolute, size-full, min 44×44 hit area
  <svg>      leading/trailing icons: −2px inline margin each side, size + colour from container rules
  text       label (nowrap)
```

Base classes (line 9): `[--control-icon-color:currentColor] [&_svg]:-mx-0.5 relative inline-flex
shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap
rounded-[var(--control-radius)] border font-medium text-base outline-none
transition-[box-shadow,scale] [&:not([aria-haspopup]):not(:disabled):not([aria-disabled=true])]:will-change-[scale]
[&:active:not([aria-haspopup])]:scale-[0.97] before:pointer-events-none before:absolute
before:inset-0 before:rounded-[calc(var(--control-radius)-1px)] pointer-coarse:after:absolute
pointer-coarse:after:size-full pointer-coarse:after:min-h-11 pointer-coarse:after:min-w-11
focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring focus-visible:ring-offset-1
focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-64
aria-disabled:cursor-not-allowed aria-disabled:opacity-64 sm:text-sm
[&_svg:not([class*='text-'])]:text-[var(--control-icon-color)] [&_svg:not([class*='size-'])]:size-4.5
sm:[&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0`.

`--control-icon-color` is the icon colour knob: `currentColor` by default, the muted-foreground
contrast colour for `ghost`, `glass` and `outline`. Sizes: `compact default icon icon-lg
icon-micro icon-tiny icon-sm icon-sm-pill icon-xl icon-xs lg micro multiline sm sm-pill
sm-multiline xl xs` (16–42). Variants: `default destructive destructive-outline ghost
ghost-muted ghost-destructive glass link media-close media-navigation outline overlay secondary
warning-outline` (43–70). Values in §3.1 and §4.1.

**InlineButton** (103–139): `<button type="button" data-slot="inline-button">` for text-like
actions inside sentences: `inline-flex shrink-0 cursor-pointer items-center gap-0.5 font-medium
underline-offset-2 [text-align:var(--inline-button-text-align,center)]
[white-space:var(--inline-button-white-space,nowrap)] hover:underline focus-visible:outline-2
focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-64` + tone
(`default` text-foreground; `muted` text-muted-foreground hover:text-foreground;
`destructive` text-destructive/80 hover:text-destructive; `picker` gap-1 text-foreground
decoration-from-font underline-offset-4 data-popup-open:underline) + optional
`touchTarget: padded` (coarse only). Font size inherits.

### 2.2 Kbd, KbdGroup (`kbd.tsx`)

```
<kbd data-slot="kbd">   key cap; text or a 12px icon
<kbd data-slot="kbd-group" class="inline-flex items-center gap-1">  wraps several Kbd
```

`Kbd`: `pointer-events-none inline-flex h-5 min-w-5 select-none items-center justify-center gap-1
rounded px-1 font-medium font-sans text-muted-foreground text-xs
[&_svg:not([class*='size-'])]:size-3` + `bg-foreground/[0.07]` (default) or `border border-b-2
border-foreground/15 px-1.5` (outline: a physical key with a 2px bottom edge, used inside running
text). `MenuShortcut` (menu.tsx:297–308) is a different, unboxed `<kbd>`: `ms-auto font-medium
font-sans text-secondary-label text-xs tracking-widest`.

### 2.3 Badge (`badge.tsx`)

`<span data-slot="badge">` (or any element via `render`): base `relative inline-flex shrink-0
items-center justify-center gap-1 whitespace-nowrap rounded-sm border border-transparent
font-medium outline-none transition-shadow focus-visible:ring-2 focus-visible:ring-inset
focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background
disabled:pointer-events-none disabled:opacity-64 [&_svg:not([class*='opacity-'])]:opacity-80
[&_svg:not([class*='size-'])]:size-3.5 sm:[&_svg:not([class*='size-'])]:size-3 [&_svg]:pointer-events-none
[&_svg]:shrink-0 [button&,a&]:cursor-pointer` (+ coarse hit area for button/anchor badges).
Sizes `control default lg sm`; variants `default destructive error info outline secondary
success warning label` (`label` colours come from a `--label` custom property set by the
caller). Hover classes only apply when the badge is rendered as `<button>` or `<a>`.

### 2.4 Menu (`menu.tsx`, Base UI Menu)

```
Menu.Root                                            state only (modal by default)
MenuTrigger  <button data-slot="menu-trigger" aria-haspopup="menu" aria-expanded>   usually render={<Button …/>}
Portal → [modal root menu, not hover-opened] internal backdrop <div role="presentation" data-base-ui-inert
           style="position:fixed; inset:0; user-select:none; clip-path:polygon(<viewport>, hole = trigger rect)">   (M; sits just before the positioner)
       → MenuPrimitive.Positioner  <div role="presentation" data-slot="menu-positioner" class="z-[130]" data-side data-align>
           render = ClosingPositioner (keeps last open placement while closing)
  span[data-base-ui-focus-guard]                    focus trap guards (inside, before/after)
  MenuPrimitive.Popup  <div role="menu" data-slot="menu-popup" data-open|data-closed data-side data-align>
     classes: dropdown-glass relative flex origin-(--transform-origin) rounded-lg
              shadow-[0_16px_40px_-18px_rgb(0_0_0/55%)] dark:shadow-[0_18px_44px_-18px_rgb(0_0_0/80%)]
              outline-none focus:outline-none [-webkit-app-region:no-drag] max-w-[calc(100vw-2rem)]
              + width="default": min-w-[min(10rem,calc(100vw-2rem))]  | width="content": w-max min-w-0
    <div class="max-h-(--available-height) w-full overflow-y-auto p-1">   scroll container
      MenuSearchInput?   (filter field, see below)
      MenuGroup <div role="group" data-slot="menu-group">
        MenuGroupLabel <div data-slot="menu-label" aria-hidden>                px-2 py-1.5 font-medium text-muted-foreground text-xs data-inset:ps-9 sm:data-inset:ps-8
      MenuItem <div role="menuitem" data-slot="menu-item" data-variant data-density data-inset data-highlighted data-disabled>
        <svg>               leading icon (direct child): 16px, opacity .8, muted colour, −2px margin
        MenuItemLabel <span data-slot="menu-item-label" class="min-w-0 truncate">   (touch density trims text box to cap height)
        MenuItemDetail <span data-slot="menu-item-detail">  trailing value, ms-auto ps-4 text-secondary-label tabular-nums
        MenuShortcut <kbd data-slot="menu-shortcut">     trailing key chord
      MenuCheckboxItem <div role="menuitemcheckbox" aria-checked data-checked|data-unchecked data-slot="menu-checkbox-item">  (grid)
        CheckboxItemIndicator <span class="col-start-1"> <svg check 1.75 stroke> </span>   (unmounted while unchecked)
        <span class="col-start-2">label</span>
        + variant "switch": label in col 1, indicator rendered as a mini switch (keepMounted) in col 2
      MenuRadioGroup <div role="group" data-slot="menu-radio-group">
        MenuRadioItem <div role="menuitemradio" aria-checked data-checked data-slot="menu-radio-item">
          <span class="min-w-0 flex-1">children (callers put MenuRadioItemIndicator inside, usually last)</span>
          MenuRadioItemIndicator <span aria-hidden data-slot="menu-radio-item-indicator" class="flex shrink-0"> CheckIcon 14px </span>
      MenuSeparator <div role="separator" data-slot="menu-separator" class="mx-2 my-1 h-px bg-border">
      MenuSub → MenuSubTrigger <div role="menuitem" aria-haspopup="menu" aria-expanded data-slot="menu-sub-trigger" data-popup-open>
                  children + <ChevronRightIcon class="-me-0.5 ms-auto opacity-80">
                MenuSubPopup = MenuPopup with data-slot="menu-sub-content", side inline-end, align start,
                  sideOffset 0, alignOffset −5 (when align ≠ center); popup gets data-nested
```

Item classes (132): `[&>svg]:-mx-0.5 flex min-h-8 cursor-pointer select-none items-center gap-2
rounded-sm px-2 py-1 text-base text-foreground outline-none data-disabled:pointer-events-none
data-disabled:cursor-not-allowed data-highlighted:bg-accent data-inset:ps-8
data-[variant=destructive]:text-destructive-foreground data-highlighted:text-accent-foreground
data-disabled:opacity-64 sm:min-h-7 sm:text-sm [&>svg:not([class*='opacity-'])]:opacity-80
[&>svg:not([class*='size-'])]:size-4.5 sm:[&>svg:not([class*='size-'])]:size-4
[&>svg:not([class*='text-'])]:text-muted-foreground
data-[variant=destructive]:[&>svg:not([class*='text-'])]:text-current [&>svg]:pointer-events-none
[&>svg]:shrink-0`; `variant="ghost"` merges `buttonVariants({variant:"ghost", size:"compact",
className:"h-auto min-h-7 w-full sm:text-xs"})`; `density="touch"` adds `min-h-10 sm:min-h-10`.

Checkbox item (177–181): `grid min-h-8 in-data-[side=none]:min-w-[calc(var(--anchor-width)+1.25rem)]
cursor-pointer items-center gap-2 rounded-sm py-1 ps-2 text-base text-foreground outline-none …
data-highlighted:bg-accent data-highlighted:text-accent-foreground data-disabled:opacity-64
sm:min-h-7 sm:text-sm [&_svg…]:size-4.5 sm:…size-4` + `grid-cols-[1rem_minmax(0,1fr)] pe-4`
(default/icon; icon → `grid-cols-[0.625rem_minmax(0,1fr)]`) or `grid-cols-[1fr_auto] gap-4
pe-1.5` (switch). Check path: `M5.252 12.7 10.2 18.63 18.748 5.37`, `strokeWidth 1.75`,
round caps/joins, viewBox 24.

Radio item (239): `[&_svg]:-mx-0.5 flex min-h-8 … cursor-pointer items-center rounded-sm px-2 py-1
text-base text-foreground outline-none data-checked:bg-foreground/[0.08] … data-highlighted:bg-accent
data-highlighted:text-accent-foreground data-disabled:opacity-64 sm:min-h-7 sm:text-sm …`
(no gap; callers lay out their own row).

Sub-trigger (340): item classes with `data-popup-open:bg-accent data-popup-open:text-accent-foreground`;
the chevron is the last SVG and keeps `ms-auto`.

**MenuSearchInput** (77–112):

```
<div class="min-w-0 shrink-0 px-2 pt-1.5 pb-1">
  <div class="relative -translate-y-px border-b border-border/70 pb-1.5 transition-colors focus-within:border-ring">
    <SearchIcon class="pointer-events-none absolute top-1.5 left-0 size-4 shrink-0 text-muted-foreground/55">   (or Spinner md muted when busy)
    <Input unstyled size="sm" class="rounded-none bg-transparent text-sm [&_input]:h-6.5 [&_input]:ps-5 [&_input]:font-sans [&_input]:leading-6.5">
```

### 2.5 Popover (`popover.tsx`, Base UI Popover)

```
PopoverTrigger <button data-slot="popover-trigger">
Portal
  [touchBackdrop] PopoverPrimitive.Backdrop  fixed inset-0 z-[130] hidden pointer-coarse:block   (eats outside taps on touch)
  Positioner <div data-slot="popover-positioner"> (ClosingPositioner)
     h-(--positioner-height) max-h-(--available-height) w-(--positioner-width) max-w-(--available-width)
     + (not keepMounted) transition-transform data-instant:transition-none
     + variant default: z-[130]   | variant panel: z-(--z-sheet) w-[min(calc(var(--thread-details-panel-width)+1rem),calc(var(--anchor-width)-0.5rem))]
                                     transition-[top] duration-150 ease-out, collisionPadding 0
    Popup <div role="dialog" data-slot="popover-popup">
       relative flex h-(--popup-height,auto) w-(--popup-width,auto) origin-(--transform-origin) rounded-lg
       text-popover-foreground outline-none transition-[width,height,scale,opacity]
       before:(lip, radius lg−1) data-[starting|ending]-style:scale-98 opacity-0
       + dropdown-glass (unless panel) + elevation shadow (unless tooltipStyle/panel)
       + tooltipStyle: w-fit text-balance rounded-md text-xs shadow-md/5 before:rounded-[calc(var(--radius-md)-1px)]
       + width sm|md|lg: max-w-[calc(100vw-2rem)] w-64|w-80|w-96
       + panel: w-full overflow-visible rounded-none border-0 bg-transparent shadow-none before:hidden, no backdrop-filter
       + has-data-[slot=calendar]: rounded-xl (12, lip 11) and the viewport switches to p-2 (8px all round)
      Viewport <div data-slot="popover-viewport"> (Base UI Viewport; a plain div when keepMounted)
         relative size-full max-h-(--available-height) overflow-clip px-(--viewport-inline-padding)
         + padding default: py-4 [--viewport-inline-padding:1rem] | compact: py-2 [..:0.75rem] | none: py-0, inline 0, radius lg−1
         + not tooltipStyle: not-data-transitioning:overflow-y-auto
         + crossfade rules for data-current/data-previous children (handle-based popovers that swap content)
        children
PopoverTitle <h2 data-slot="popover-title" class="font-semibold text-sm leading-none">
PopoverClose <button data-slot="popover-close">
```

### 2.6 Tooltip (`tooltip.tsx`, Base UI Tooltip)

```
[TooltipProvider]  only where a caller wants grouped delays (none at the app root)
[TooltipScrollDismissArea] <div onScrollCapture>  closes the hovered tooltip when its scroller scrolls
Tooltip.Root
  TooltipTrigger  data-slot="tooltip-trigger" data-base-ui-tooltip-trigger   (render={<Button …/>} etc.)
  Portal
    TooltipPositioner <div role="presentation" data-slot="tooltip-positioner">
        pointer-events-none z-[140] max-w-(--available-width) transition-[top,left,right,bottom,transform] data-instant:transition-none
        collisionPadding {top: 5 + native titlebar height, right 5, bottom 5, left 5}
      TooltipPopup <div data-slot="tooltip-popup" data-side data-align data-instant?>
          relative flex origin-(--transform-origin) text-balance rounded-md text-popover-foreground text-xs
          transition-[scale,opacity] before:(lip, radius md−1) data-[starting|ending]-style:scale-98 opacity-0
          data-instant:duration-0
          + default: border bg-popover not-dark:bg-clip-padding shadow-md/5
          + glass:   dropdown-glass shadow-xl shadow-black/25 before:hidden
          + code:    max-w-120 wrap-anywhere text-left font-mono text-[11px] leading-relaxed
          + other:   max-w-80 wrap-anywhere whitespace-normal leading-snug
        <div data-slot="tooltip-viewport" class="min-w-0 px-2 py-1">content</div>
```

There is **no arrow** (no `Tooltip.Arrow` rendered).

### 2.7 Input (`input.tsx`)

```
<span data-slot="input-control" data-size="sm|compact|default|lg|<number>">   the visible box (unless unstyled)
  ::before  lip highlight (radius lg−1)
  <input data-slot="input">  (Base UI Input; or a native <input> with nativeInput)
```

Wrapper: `relative inline-flex w-full rounded-lg border border-input bg-background
not-dark:bg-clip-padding text-base text-foreground shadow-xs/5 ring-ring/24 transition-shadow
before:… not-has-disabled:not-has-focus-visible:not-has-aria-invalid:before:shadow-[0_1px_black/4%]
has-focus-visible:has-aria-invalid:border-destructive/64 has-focus-visible:has-aria-invalid:ring-destructive/16
has-aria-invalid:border-destructive/36 has-focus-visible:border-ring has-autofill:bg-foreground/4
has-disabled:opacity-64 has-[:disabled,:focus-visible,[aria-invalid]]:shadow-none
has-focus-visible:ring-[3px] has-focus-visible:ring-inset sm:text-sm dark:bg-input/32
dark:has-autofill:bg-foreground/8 dark:has-aria-invalid:ring-destructive/24
dark:not-has-…:before:shadow-[0_-1px_white/6%]` (+ `rounded-md` for `compact`, `font-mono
tabular-nums` for `font="mono"`). Input: `h-8.5 w-full min-w-0 rounded-[inherit] px-[calc(--spacing(3)-1px)]
leading-8.5 outline-none placeholder:text-placeholder sm:h-7.5 sm:leading-7.5
[transition:background-color_5000000s_ease-in-out_0s]` + size overrides; `type=search` hides
WebKit's cancel/decoration buttons, `type=number` hides spinners, `type=file` styles the button.
`unstyled` drops all wrapper classes (used inside InputGroup, menu/combobox search fields).

**DraftInput** (`draft-input.tsx`) is `Input` plus `useCommitOnBlur` handlers; identical visuals.

**Label** (`label.tsx`): `<label data-slot="label" class="inline-flex items-center gap-2
text-base/4.5 sm:text-sm/4 font-medium text-foreground">`.

### 2.8 InputGroup (`input-group.tsx`)

```
<div role="group" data-slot="input-group">                      the visible box (variant default|popover|ghost, size default|lg)
  InputGroupAddon <div data-slot="input-group-addon" data-align="inline-start|inline-end|block-start|block-end">
     icons / Kbd / Buttons / Badges; mousedown on non-interactive addon content focuses the input
  InputGroupInput = Input unstyled  (its input-control span becomes display:contents; its ::before hidden)
  (or a Textarea → group grows: h-auto, textarea min-h 82px, py 11px, no resize)
```

Group, default variant: `relative inline-flex w-full min-w-0 items-center rounded-[var(--control-radius)]
border text-base text-foreground ring-ring/24 transition-shadow` + focus/invalid/disabled/autofill
rules keyed on `:has(input…, textarea…)` + `border-input bg-background shadow-xs/5 before:lip`
(dark `bg-input/32`). Popover variant: `dropdown-glass rounded-2xl`, focus = no ring, border
`contrast-foreground 18%`. Ghost: transparent, `hover:bg-muted/40`, focused `bg-background`.
Addon padding/margins adapt to what it holds (button → −8px, trailing kbd → −0.35rem,
trailing badge → −6px). A direct `<kbd>` child gets radius `calc(var(--radius) − 5px)` = 3px (line 48).
Addon `inset="pill"` (lines 53–68) makes every button inside `rounded-full`. Combined with
`inline-end` / `inline-start` it uses padding-end / padding-start 4px (also next to an `sm` input),
gap 2px, and drops the −8px button pull (`has-[>button]:me-0` / `ms-0`). Full strings: lines 8, 21–25, 48–79.

### 2.9 Textarea (`textarea.tsx`)

```
<span data-slot="textarea-control" data-size>   wrapper: same recipe as Input's (radius lg, lip, focus ring, invalid)
  Field.Control → <textarea data-slot="textarea" class="field-sizing-content max-h-64 w-full rounded-[inherit]
                    px-[calc(--spacing(3)-1px)] py-[calc(--spacing(1.5)-1px)] outline-none wrap-anywhere …">
```

`rows` → inline `min-height: calc(<rows>lh + 2 * (var(--spacing) * <1|1.5|2> - 1px))` because
`field-sizing: content` ignores `rows`.

### 2.10 NumberField (`number-field.tsx`, Base UI NumberField)

```
NumberField.Root <div data-slot="number-field" data-size class="flex w-full flex-col items-start gap-2">  (label/description slots stack here)
  NumberFieldGroup <div data-slot="number-field-group">           the visible box (Input recipe, focus-within ring)
    NumberFieldDecrement <button data-slot="number-field-decrement">  MinusIcon, rounded-s-[lg−1], hover:bg-accent
    NumberFieldInput <input data-slot="number-field-input">         centred, tabular-nums, transparent
    NumberFieldIncrement <button data-slot="number-field-increment">  PlusIcon, rounded-e-[lg−1]
```

### 2.11 Select (`select.tsx`, Base UI Select)

```
Select.Root (modal)
SelectTrigger <button role="combobox" data-slot="select-trigger" aria-expanded data-popup-open data-pressed>
   children: SelectValue <span data-slot="select-value" class="flex-1 truncate data-placeholder:text-placeholder">
   SelectPrimitive.Icon <span data-slot="select-icon"> ChevronDownIcon -me-1 size-3.5 opacity-70 </span>
SelectButton: same look as a plain <button> (truncate span + chevron) for non-Select pickers
Portal → Positioner <div data-slot="select-positioner" class="z-[130] select-none" data-side data-align>
  Popup <div data-slot="select-popup" class="origin-(--transform-origin) rounded-lg text-foreground outline-none [-webkit-app-region:no-drag]">
    ScrollUpArrow <div data-slot="select-scroll-up-arrow">   h-6, gradient from popover, ChevronUp 16px (only when the list overflows upward)
    <div class="dropdown-glass relative h-full rounded-lg shadow-[…55%] dark:shadow-[…80%] min-w-(--anchor-width)">
      SelectPrimitive.List <div role="listbox" data-slot="select-list" class="max-h-(--available-height) overflow-y-auto p-1 …">
        SelectGroup <div role="group" data-slot="select-group">
          SelectGroupLabel <div data-slot="select-group-label" aria-hidden class="px-2 py-1.5 font-medium text-muted-foreground text-xs">
        SelectItem <div role="option" aria-selected data-selected data-highlighted data-disabled data-slot="select-item">
          ItemText <div data-slot="select-item-text" class="min-w-0 flex-1 [&_svg:not([class*='text-'])]:text-muted-foreground">
    ScrollDownArrow <div data-slot="select-scroll-down-arrow">
```

Trigger variants (12–36): `default` = Input-like box (`w-full min-w-36 border-input bg-background
… shadow-xs/5 before:lip`, icon `text-icon-muted` at opacity .8 unless the icon sets its own),
`ghost` = transparent, `text-secondary-label`, hover/pressed `bg-accent text-foreground`. Sizes
`compact default lg sm xs`. `SelectItem` has no check indicator. Selection shows only as a
fill.

### 2.12 Combobox (`combobox.tsx`, Base UI Combobox)

```
Combobox.Root (+context: chipsRef, multiple)
ComboboxInput:
  <div class="relative not-has-[>*.w-full]:w-fit w-full text-foreground has-disabled:opacity-64">
    [startAddon] <div data-slot="combobox-start-addon" aria-hidden>  absolute inset-y-0 start-px z-10, ps 11px, opacity .8, icon 16px
    Combobox.Input render=<Input nativeInput size>  → <span data-slot="input-control"> <input data-slot="combobox-input" role="combobox"> 
    [showTrigger] ComboboxTrigger <button data-slot="combobox-trigger">  absolute end-0.5 top-1/2, 28px square (32 <640), ChevronsUpDown 16px, opacity .8 → 1 on hover
    [showClear]   ComboboxClear <button data-slot="combobox-clear">     same box, XIcon; hides the trigger when both are present
ComboboxSearchInput: the menu-style filter field (px-3 pt-2.5 wrapper, border-b rule, search icon) around an unstyled sm ComboboxInput
Portal → Positioner <div data-slot="combobox-positioner" class="z-[130] select-none">
  <span class="dropdown-glass relative flex max-h-full min-w-(--anchor-width) max-w-(--available-width) origin-(--transform-origin)
               rounded-lg shadow-[…55%] transition-[scale,opacity] dark:shadow-[…80%]">
    Combobox.Popup <div role="dialog" data-slot="combobox-popup" class="flex min-w-0 max-h-[min(var(--available-height),23rem)] flex-1 flex-col overflow-hidden text-foreground">
      ComboboxEmpty <div role="status" data-slot="combobox-empty" class="not-empty:p-2 text-center text-base text-muted-foreground sm:text-sm">
      ComboboxList → ScrollArea(scrollbarGutter, scrollFade) → <div role="listbox" data-slot="combobox-list" class="not-empty:scroll-py-1 not-empty:px-1 not-empty:py-1">
        ComboboxItem <div role="option" data-slot="combobox-item" data-selected data-highlighted>
          <div data-slot="combobox-item-content" class="flex min-w-0 flex-1 items-center gap-2 [&_svg:not([class*='text-'])]:text-muted-foreground">
      ComboboxStatus <div data-slot="combobox-status" class="px-3 py-2 font-medium text-muted-foreground text-xs empty:m-0 empty:p-0">
```

`ComboboxListVirtualized` omits the ScrollArea for external virtualisers (the live branch picker
uses one).

### 2.13 Autocomplete (`autocomplete.tsx`, Base UI Autocomplete)

Same input shell as Combobox (`data-slot="autocomplete-input"`, trigger off by default). The
popup frame is **not glass**. Its classes are `relative flex max-h-full min-w-(--anchor-width)
max-w-(--available-width) origin-(--transform-origin) rounded-lg border bg-popover
not-dark:bg-clip-padding shadow-lg/5 transition-[scale,opacity] before:lip`. Items:
`flex min-h-8 cursor-default select-none items-center gap-2 rounded-sm px-2 py-1 text-base
outline-none hover:bg-accent data-selected:bg-accent/50 data-selected:text-foreground
data-highlighted:bg-accent data-highlighted:text-accent-foreground … sm:min-h-7 sm:text-sm`.
Groups stack with `mt-1.5` between them; group labels / list headings use the menu label recipe.

### 2.14 Switch (`switch.tsx`, Base UI Switch)

```
<span role="switch" aria-checked tabindex="0" data-slot="switch" data-size data-checked|data-unchecked data-disabled data-mixed>
   + hidden <input type="checkbox" aria-hidden tabindex=-1> for forms
  <span data-slot="switch-thumb" data-checked|data-unchecked>
```

Root: `inline-flex h-[calc(var(--thumb-size)+2px)] w-[calc(var(--thumb-size)*2-2px)] shrink-0
cursor-pointer items-center rounded-full p-[2px] outline-none transition-[background-color,box-shadow]
duration-200 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring
focus-visible:ring-offset-1 focus-visible:ring-offset-background data-checked:bg-primary
data-unchecked:bg-input data-disabled:cursor-not-allowed data-disabled:opacity-64
data-[mixed]:bg-input` + `--thumb-size` 4 (sm: 3.5) for `size="sm"`, 5 (sm: 4) otherwise. Thumb:
`pointer-events-none block size-[calc(var(--thumb-size)-2px)] shrink-0 origin-left
…:active…:scale-x-110 …:active…:rounded-[var(--thumb-size)/calc(var(--thumb-size)*1.1)]
rounded-(--thumb-size) bg-background shadow-sm/5 dark:data-unchecked:bg-muted-foreground
[transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s]
data-checked:origin-right data-checked:translate-x-[calc(var(--thumb-size)-4px)]`; `mixed` parks
the thumb in the middle at opacity .7 and sets `aria-checked="mixed"`.

### 2.15 Toggle, ToggleGroup (`toggle.tsx`, `toggle-group.tsx`)

`<button data-slot="toggle" aria-pressed data-pressed>`: base `[&_svg]:-mx-0.5 relative inline-flex
shrink-0 cursor-pointer select-none items-center justify-center gap-2 whitespace-nowrap rounded-lg
border font-medium text-base text-foreground outline-none transition-shadow before:… hover:bg-accent
focus-visible:ring-2 … disabled:pointer-events-none disabled:opacity-64 aria-disabled:opacity-64
data-pressed:bg-input/64 data-pressed:text-accent-foreground sm:text-sm
[&_svg:not([class*='opacity-'])]:opacity-80 [&_svg…]:size-4.5 sm:…size-4` + size
(`compact default lg segmented sm xs`) + variant (`default pill ghost outline segmented`).

`ToggleGroup` → `<div role="group" data-slot="toggle-group" data-variant data-size>` with
`flex w-fit *:focus-visible:z-10`; default variant is **segmented**: `gap-0.5 rounded-lg
bg-input/40 p-0.5`, items default to size `segmented`. `default` variant: `gap-0.5`. `outline`
(and others): items are joined, with inner corners and inner borders removed, `::before` lips
offset by ±0.5px, vertical orientation stacks. `Separator`s between joined items light up in
dark when a neighbour is hovered (`input/64`) or pressed (`input`).

### 2.16 Separator (`separator.tsx`)

`<div role="separator" data-slot="separator" data-orientation aria-orientation>`: `shrink-0
bg-border`, horizontal `h-px w-full`, vertical `w-px` + `self-stretch` unless the caller gives a
height class.

### 2.17 ScrollArea (`scroll-area.tsx`, Base UI ScrollArea)

```
ScrollArea.Root <div class="relative size-full min-h-0 overflow-hidden rounded-[inherit]" data-has-overflow-x|y data-overflow-y-start|end …>
  Viewport <div data-slot="scroll-area-viewport" style="overflow:scroll; --scroll-area-overflow-{x,y}-{start,end}: <px>">
     h-full max-h-[inherit] overflow-auto overscroll-contain rounded-[inherit] outline-none transition-shadow focus-visible:ring-2 …
     + scrollFade: mask-t/b/l/r-from-[calc(100% - min(var(--fade-size), var(--scroll-area-overflow-*)))] focus-visible:mask-none [--fade-size:1.5rem]
       + scrollFadePadding (default): scroll-p-[var(--fade-size)]   | else has-focus-visible:mask-none
     + scrollbarGutter: scrollbar-gutter: stable   + chainVerticalScroll: overscroll-y-auto
     + hideScrollbars: scrollbar-width:none, ::-webkit-scrollbar hidden, and no Base UI scrollbars rendered
    children
  ScrollBar vertical   <div data-slot="scroll-area-scrollbar" data-orientation="vertical" data-hovering? data-scrolling?>
    Thumb <div data-slot="scroll-area-thumb" class="relative flex-1 rounded-full bg-[var(--app-scrollbar-thumb)] transition-colors hover:bg-[var(--app-scrollbar-thumb-hover)]">
  ScrollBar horizontal
  Corner <div data-slot="scroll-area-corner">
```

`getVirtualizedScrollFadeClassName({top,bottom})` gives virtualised lists the same 1.5rem fade
through the `virtualized-scroll-fade` utility, keeping a 6px opaque lane for the native scrollbar.

### 2.18 Spinner (`spinner.tsx`)

`<svg class="lucide lucide-loader-circle motion-safe:visible-animate-spin [size] [tone]" role="status"
aria-label="Loading">` with the single path `M21 12a9 9 0 1 1-6.219-8.56`. No default size
(inside a Button the button's SVG rule sizes it).

### 2.19 Dialog (`dialog.tsx`, `dialog-styles.ts`, Base UI Dialog)

```
Dialog.Root (modal); ControlledOpenContext lets RetainedWhileClosed keep the last children during the exit animation
DialogTrigger data-slot="dialog-trigger"
DialogPortal
  DialogBackdrop <div data-slot="dialog-backdrop" class="dialog-backdrop fixed inset-0 z-50 transition-all duration-200 data-[starting|ending]-style:opacity-0">  (forceRender)
     media variant: bg-black/75 backdrop-blur-none (z-60)
  DialogViewport <div data-slot="dialog-viewport" class="fixed inset-0 z-50 grid grid-rows-[1fr_auto_1fr] justify-items-center p-4">
     + bottomStickOnMobile (default): max-sm:grid-rows-[1fr_auto] max-sm:p-0 max-sm:pt-12
     + media: z-[60] grid-rows-1 place-items-center px-4 pb-6 pt-[calc(var(--native-titlebar-height,0px)+1.5rem)]
    DialogPopup <div role="dialog" aria-labelledby aria-describedby data-slot="dialog-popup" data-open data-nested? style="--nested-dialogs: N">
       DIALOG_POPUP_CLASS: dialog-glass + base (see below) + rounded-2xl border; row-start-2 text-popover-foreground; default: max-h-full max-w-lg
       base: -translate-y-[calc(1.25rem*var(--nested-dialogs))] relative flex min-h-0 w-full min-w-0
             scale-[calc(1-0.1*var(--nested-dialogs))] flex-col opacity-[calc(1-0.1*var(--nested-dialogs))] outline-none
             transition-[scale,opacity,translate] duration-200 ease-in-out will-change-transform
             data-nested:data-[starting|ending]-style:translate-y-8 data-nested-dialog-open:origin-top
             data-[starting|ending]-style:scale-98 data-[starting|ending]-style:opacity-0 [-webkit-app-region:no-drag]
       mobile (<640, bottomStickOnMobile): max-w-none rounded-none border-x-0 border-t border-b-0
             opacity-[calc(1-min(var(--nested-dialogs),1))] data-[starting|ending]-style:translate-y-4
      DialogHeader <div data-slot="dialog-header" class="flex flex-col gap-2 p-6 …:pb-3 (when a panel follows) max-sm:pb-4">
        DialogTitle <h2 data-slot="dialog-title" class="wrap-anywhere font-semibold text-xl leading-snug">
        DialogDescription <p data-slot="dialog-description" class="text-muted-foreground text-sm">
      DialogPanel → ScrollArea(scrollFade) → <div data-slot="dialog-panel" class="space-y-4 p-6 …:pt-1 (after header) …:pb-1 (before a bare footer)">
      DialogFooter <div data-slot="dialog-footer" class="flex flex-col-reverse gap-2 px-6 sm:flex-row sm:justify-end sm:rounded-b-[calc(var(--radius-2xl)-1px)]">
         default: border-t bg-muted/72 py-4   | bare: py-4
        DialogAction = Button with data-slot="dialog-action" (non-destructive variants only), receives initial focus
      [showCloseButton] DialogClose aria-label="Close" class="absolute end-2 top-2" render=<Button size="icon" variant="ghost"> <XIcon> 
```

Media variant (`DIALOG_MEDIA_POPUP_CLASS`): no glass, `max-h-[92vh] w-auto max-w-[92vw]
overflow-visible rounded-none border-transparent bg-transparent p-0 shadow-none` over a 75%
black backdrop.

### 2.20 AlertDialog (`alert-dialog.tsx`)

Same backdrop/viewport/popup classes as Dialog (shared constants) with `data-slot="alert-dialog-*"`,
`max-w-lg`, no close button. Base UI renders the popup as `role="alertdialog"` and forces `modal`
and `disablePointerDismissal` on, whatever the props say (S, dialog bundle). Header: `flex min-h-0 flex-col gap-2 p-6 text-center max-sm:pb-4
sm:text-left`. Title adds `shrink-0`. Description: `wrap-anywhere max-h-[min(24rem,60dvh)] min-h-0
overflow-y-auto overscroll-contain text-muted-foreground text-sm select-text`. Footer default
`border-t bg-muted/72 py-4`, bare `pb-6`. Actions are `DialogAction` (= `AlertDialogAction`).
`portalContainer` can retarget the portal.

### 2.21 Sheet (`sheet.tsx`, Base UI Dialog)

```
SheetBackdrop <div data-slot="sheet-backdrop" class="fixed inset-0 z-(--z-sheet) bg-background/60 backdrop-blur-xs transition-all duration-200 data-[starting|ending]-style:opacity-0">
SheetViewport <div data-slot="sheet-viewport" class="fixed inset-0 z-(--z-sheet) grid"> + side: bottom → grid-rows-[1fr_auto] pt-12 | top → grid-rows-[auto_1fr] pb-12 | left → flex justify-start | right → flex justify-end; inset → sm:p-4
  SheetPopup <div role="dialog" data-slot="sheet-popup">
     relative flex max-h-full min-h-0 w-full min-w-0 flex-col bg-popover not-dark:bg-clip-padding text-popover-foreground shadow-lg/5
     transition-[opacity,translate] duration-200 ease-in-out will-change-transform before:lip (no radius) data-[starting|ending]-style:opacity-0 max-sm:before:hidden
     + right: col-start-2 w-[calc(100%-3rem)] max-w-md border-s, start/end translate-x-8
     + left:  w-[calc(100%-3rem)] max-w-md border-e, start/end -translate-x-8
     + top:   border-b, start/end -translate-y-8      + bottom: row-start-2 border-t, start/end translate-y-8
     + inset: before:hidden sm:rounded-2xl sm:border sm:before:rounded-[2xl−1] footer bottom radius
     + transitionDurationMs → inline transition-duration; 0 → everything instant (!important overrides)
    children + [showCloseButton] close Button (icon ghost) absolute end-2 top-2
SheetHeader (p-6, pb-3 before a panel), SheetTitle (font-semibold text-xl leading-snug), SheetDescription (text-muted-foreground text-sm)
```

### 2.22 Toast (`toast.tsx`, Base UI Toast)

Two managers, both buffered until a provider subscribes: `toastManager` (stacked viewport) and
`anchoredToastManager` (toasts positioned against an element).

```
ToastProvider (position "top-right" by default) → Toast.Provider(timeout 5000, limit 3)
  Toast.Portal data-slot="toast-portal"
    Toast.Viewport <div role="region" aria-label="Notifications" aria-live="polite" data-slot="toast-viewport" data-position style="--toast-frontmost-height: <px>">
        fixed z-100 mx-auto flex w-[calc(100%-var(--toast-inset)*2)] max-w-90 [--toast-header-offset:var(--workspace-topbar-height)]
        [--toast-inset:1rem] sm:[--toast-inset:2rem]; top = inset + header offset | bottom = inset; left/right = inset; center: left-1/2 -translate-x-1/2
      Toast.Root <div role="dialog" (role="alertdialog" when priority is "high") tabindex="0" aria-modal="false" data-type data-position data-expanded? data-limited? (+ inert) data-swipe-direction?
                 style="--toast-index:i; --toast-offset-y:<px>; --toast-height:<px>; --toast-swipe-movement-x|y">
          dropdown-glass absolute z-[calc(9999-var(--toast-index))] w-full overflow-visible select-none rounded-lg text-popover-foreground
          shadow-xl shadow-black/25 [transition:transform_.5s_cubic-bezier(.22,1,.36,1),opacity_.5s,height_.15s] + stacking/enter/exit transforms (§6)
          ::after  absolute left-0 w-full h-[gap+1px] below (top positions) / above (bottom): hover bridge between stacked toasts
        ThreadToastVisibleAutoDismiss   (renders nothing; optional visible-time timer)
        <div class="absolute z-20 -top-1.5 -right-1.5"> <button data-slot="toast-close" aria-label="Dismiss notification"> XIcon 12px stroke 2.25
        Toast.Content <div class="pointer-events-auto min-h-0 overflow-y-visible pl-3.5 text-sm transition-opacity duration-250 [overflow-x:clip] data-expanded:opacity-100 …">
          body column (gap 2px)
            title row: [icon box] + Toast.Title <h2 data-slot="toast-title" class="min-w-0 wrap-break-word font-medium">
            Toast.Description <p data-slot="toast-description" class="min-w-0 select-text wrap-break-word text-muted-foreground"> (+ line-clamp-4 for long errors)
            [expandable] "Show details" toggle + panel, or the description itself as the toggle
          trailing controls row (gap 6px): CopyErrorButton, additional actions, secondary action, Toast.Action (Button xs)
AnchoredToastProvider → AnchoredToasts → per toast: Toast.Positioner <div data-slot="toast-positioner" class="z-100 max-w-[min(16rem,var(--available-width))]"> → Toast.Root data-slot="toast-popup" (tooltipStyle: just the title in px-2 py-1)
```

Icons by `type`: `error` CircleAlert, `info` Info, `loading` Spinner, `success` CircleCheck,
`warning` TriangleAlert (`toast.tsx:110–116`). `data.leadingIcon` replaces them.

### 2.23 Collapsible, CollapsibleSectionHeader

Collapsible (Base UI): `Root data-slot="collapsible"`, `Trigger data-slot="collapsible-trigger"
class="cursor-pointer"` (aria-expanded, data-panel-open), `Panel data-slot="collapsible-panel"`
`overflow-hidden` + (animate) `h-(--collapsible-panel-height) transition-[height] duration-200
motion-reduce:transition-none data-[starting|ending]-style:h-0
data-open:data-ending-style:[height:var(--collapsible-panel-height)]`.

`CollapsibleSectionHeader` (not Base UI; a plain `<button type="button" aria-expanded>`):

```
<button class="flex h-8 w-full cursor-pointer items-center gap-2 rounded-md text-left text-xs font-medium
               focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring/70
               + px-2 (default) | ps-2 pe-2.5 (panel) + tone label colour">
  <span class="min-w-0 truncate">label</span>
  <span aria-hidden class="h-px min-w-2 flex-1 <tone line>">        leader rule; `invisible` when an accessory is present
  accessory (e.g. a count, SectionHeaderStatus)
  <ChevronDownIcon aria-hidden class="size-3 shrink-0 transition-transform [-rotate-90 when collapsed]">
```

Tones: `muted` (label `text-sidebar-muted-foreground`, line `bg-sidebar-border/50`),
`emphasized` (`text-sidebar-foreground/80`, `bg-sidebar-foreground/25`), `accent`
(`text-primary`, `bg-primary/50`). `SectionHeaderStatus`: `shrink-0 text-3xs leading-none
text-destructive-foreground`.

### 2.24 Empty (`empty.tsx`)

```
<div data-slot="empty" class="flex min-w-0 flex-1 flex-col items-center justify-center text-balance text-center + size">
  EmptyHeader <div data-slot="empty-header" class="flex max-w-sm flex-col items-center text-center">
    EmptyMedia <div data-slot="empty-media" data-variant class="relative mb-6">
       variant icon: two decorative cards (aria-hidden, absolute bottom-px, scale .84, rotate ∓10°, translate ∓2px,
                     origin bottom-left / bottom-right, no shadow) behind the front card
       front card: relative flex size-9 items-center justify-center rounded-md border bg-card text-foreground shadow-sm/5 before:lip, icon 18px
    EmptyTitle <div data-slot="empty-title" class="font-semibold text-xl">
    EmptyDescription <div data-slot="empty-description" class="text-muted-foreground text-sm … links underlined, hover primary; mt-1 after title/description">
  EmptyContent <div data-slot="empty-content" class="flex w-full min-w-0 max-w-sm flex-col items-center gap-4 text-balance text-sm">
```

### 2.25 Skeleton, PanelTabCloseButton, MiddleTruncate

* **Skeleton**: `<div data-slot="skeleton" class="bg-muted-foreground/15 motion-safe:animate-skeleton
  rounded-sm|rounded-lg|rounded-full">` (shape block/card/pill); callers give the size.
* **PanelTabCloseButton**: inside a tab that carries `group/tab`:
  ```
  <button type="button" aria-label class="cursor-pointer group/close relative flex size-4 shrink-0 items-center justify-center rounded-sm hover:bg-muted">
    <span class="relative flex size-3 items-center justify-center group-hover/tab:hidden group-focus-visible/close:hidden">tab icon/status</span>
    <XIcon class="hidden size-3 group-hover/tab:block group-focus-visible/close:block">
  ```
  wrapped in a Tooltip when `tooltip` is set.
* **MiddleTruncate**:
  ```
  <span title=value? class="inline-flex min-w-0 max-w-full overflow-hidden whitespace-nowrap [cap-aligned extras]">
    split:   <span class="peer min-w-0 overflow-hidden" data-overflow? style="clip-path; ">head</span>
             <span class="shrink-0 peer-data-overflow:before:content-['…']" style="translate">tail</span>
    no split: <span class="min-w-0 truncate">value</span>   (keyed "whole")
  ```

---
## 3. Measurements

### 3.0 Foundation: tokens, surfaces, type, shadows, focus

**Colour tokens, root scope** (`index.css:1113–1228`, utilities map to the `--contrast-*`
variables at 233–288). All **M**, resolved through `color: var(--x)` probes and canvas readback.

| Utility colour | Light | Dark | Notes |
| --- | --- | --- | --- |
| `background` | `#fcfcfc` | `#000000` | zinc-25 / true black |
| `foreground` | `#18181b` | `#f5f5f5` | |
| `card` | `#ffffff` | `#111111` | neutral-950 97% + white |
| `popover` | `#ffffff` | `#111111` | |
| `popover-foreground` | `#18181b` | `#f5f5f5` | |
| `primary` | `#1b4ed8` | `#346bf1` | oklch(.488 .217 264) / oklch(.571 .21 264) |
| `primary-foreground` | `#ffffff` | `#ffffff` | |
| `secondary` | `#fafafa` | `#ffffff08` | dark = white 3% |
| `secondary-foreground` | `#27272a` | `#f5f5f5` | |
| `muted` | `#fafafa` | `#ffffff08` | dark = white 3% |
| `muted-foreground` | `#71717b` | `#a1a1a1` | |
| `accent` | `#f4f4f5` | `#ffffff0a` | dark = white 4% |
| `accent-foreground` | `#18181b` | `#f5f5f5` | |
| `destructive` | `#fb2c36` | `#fb414a` | |
| `destructive-foreground` | `#c10007` | `#ff6467` | |
| `border` | `#e4e4e7` | `#ffffff14` | dark = white 8% |
| `input` | `#d4d4d8` | `#ffffff1c` | dark = white 11% |
| `ring` | `#1b4ed8` | `#346bf1` | = primary |
| `info` / `info-foreground` | `#2b7fff` / `#1447e6` | `#2b7fff` / `#50a2ff` | |
| `success` / `success-foreground` | `#00bc7d` / `#007a55` | `#00bc7d` / `#00d492` | |
| `warning` / `warning-foreground` | `#fe9a00` / `#bb4d00` | `#fe9a00` / `#ffb900` | |
| `warning-surface` | `#ff990014` | `#ff9b0029` | warning 8% / 16% |
| `placeholder`, `secondary-label`, `icon-muted` | `#71717b` | `#a1a1a1` | all = muted-foreground by default |
| `--app-scrollbar-thumb` / `-hover` | `#d9d9d9` / `#bfbfbf` | `#ffffff14` / `#ffffff1f` | white 8% / 12% in dark |
| inherited body text | `#262626` | `#f5f5f5` | `index.html:398–408` |

**Sidebar scope** (`[data-app-sidebar]`, `index.css:1234–1278`), M: light is identical to root.
Dark differs: `card #000000`, `foreground #f1f3f7`, `accent #191a1d`, `accent-foreground #f7f9ff`,
`muted #0a0a0a`, `muted-foreground #a3a3a3`, `input #ffffff2e` (white 18%), `sidebar #000000`,
`sidebar-foreground #f1f3f7`, `sidebar-muted-foreground #a3a3a3`, `secondary-label #8f8f8f`.
Primitives rendered *inside* the sidebar (buttons, section headers, inputs) pick these up;
popups do not.

**Glass parameters** (`index.css:119–121, 145–151`), M: blur **12px** light / **16px** dark,
saturation **1.14** / **1.08**, opacity **80%** both.

| Surface utility | Light | Dark | Marker |
| --- | --- | --- | --- |
| `dropdown-glass` background = `mix(popover 18%, popover @ 80%)` = popover @ 83.6% | `#ffffffd5` | `#111111d5` | M |
| `dropdown-glass` border (1px solid, foreground 10%) | `#14141d1a` | `#f5f5f51a` | M |
| `dropdown-glass` backdrop-filter | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` | M |
| `dialog-glass` background (background @ 80%) | `#fcfcfccc` | `#000000cc` | M |
| `dialog-glass` border | `#14141d1a` (foreground 10%) | `#ffffff14` (white 8%) | M |
| `dialog-glass` shadow | `0 24px 64px -24px #000000a6` | `inset 0 1px #ffffff0a, 0 24px 72px -20px #000000e6` | M |
| `dialog-backdrop` | `#fcfcfc99` (background 60%) + `blur(4px)` | `#000000a3` (64%) + `blur(4px)` | M |
| `surface-glass` (glass Button) | `#fcfcfccc` + glass filter | `#000000cc` + glass filter | P |
| fallback without backdrop-filter support | solid `popover` (dropdown, dialog) or `background` (surface) | same | S |

**Radii** (`index.css:289–294`, `--radius: .5rem`): `sm 4px`, `md 6px`, `lg 8px`, `xl 12px`,
`2xl 16px`, `3xl 20px`; `--control-radius: 8px`; `rounded` = 4px; `rounded-full` computes to
`3.35544e+07px` (Tailwind's `calc(infinity * 1px)`, i.e. a pill: radius = half the short side).
Pseudo-element lips use *outer radius − 1px*.

**Type scale** (compiled theme, S; every use below was M): `3xs 10/14`, `2xs 11/16`, `xs 12/16`,
`sm 14/20`, `base 16/24`, `lg 18/28`, `xl 20/28`, `2xl 24/32`, `3xl 30/36` (px size/line
height). `leading-snug 1.375`, `leading-relaxed 1.625`, `leading-none 1`, `tracking-widest
0.1em`. Weights `400 / 500 / 600`. Mono stack: `ui-monospace, "SF Mono", SFMono-Regular,
Menlo, Consolas, "Liberation Mono", monospace`.

**Shadows** (compiled CSS, S; each M where used):

| Name | Value |
| --- | --- |
| `shadow-xs/5` | `0 1px 2px 0 rgb(0 0 0 / 5%)` (`#0000000d`) |
| `shadow-xs/10` | `0 1px 2px 0 rgb(0 0 0 / 10%)` |
| `shadow-sm` | `0 1px 3px 0 rgb(0 0 0 / 10%), 0 1px 2px -1px rgb(0 0 0 / 10%)` |
| `shadow-sm/5` | same geometry at 5% |
| `shadow-md/5` | `0 4px 6px -1px rgb(0 0 0 / 5%), 0 2px 4px -2px rgb(0 0 0 / 5%)` |
| `shadow-lg/5` | `0 10px 15px -3px rgb(0 0 0 / 5%), 0 4px 6px -4px rgb(0 0 0 / 5%)` |
| `shadow-xl shadow-black/25` | `0 20px 25px -5px rgb(0 0 0 / 25%), 0 8px 10px -6px rgb(0 0 0 / 25%)` |
| popup elevation | light `0 16px 40px -18px rgb(0 0 0 / 55%)`, dark `0 18px 44px -18px rgb(0 0 0 / 80%)` |
| lip (`::before`) | light `0 1px rgb(0 0 0 / 4%)` (`#0000000a`), dark `0 -1px rgb(255 255 255 / 6%)` (`#ffffff0f`) |

**Focus recipes** (M):

* *Control ring* (Button, Badge, Toggle, Switch, ScrollArea viewport, toast close orb):
  `focus-visible:ring-2 ring-inset ring-ring ring-offset-1 ring-offset-background` computes to
  `box-shadow: inset 0 0 0 1px <background>, inset 0 0 0 3px <ring>` with the offset layer on top.
  Painted inside the border: **1px band of background colour** (`#fcfcfc` / `#000000`), then
  **2px of ring** (`#1b4ed8` / `#346bf1`). It replaces nothing else in the shadow list (drop
  shadow and inset highlight stay), and animates over the control's box-shadow transition (150ms).
* *Field ring* (Input, InputGroup, Textarea, NumberField, Select trigger default):
  border → ring colour, `box-shadow: inset 0 0 0 3px ring/24` (`#194fd93d` / `#326df23d`), drop
  shadow and lip removed.
* *Outline* (InlineButton): `outline: 2px solid ring`, offset −2px (global).
* Popups themselves never show a focus ring (`index.css:723–730`).

### 3.1 Button (P unless marked; ≥640px)

Geometry by size (outline variant, 16px icon + "Label"; border 1px is inside the box;
padding-x is `n·4px − 1px` so text sits on the 4px grid once the border is counted):

| Size | Height | Width | Padding x / y | Gap | Radius (lip) | Font size/line | Icon w/o size class | Below 640px (S) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `default` | 32 | auto | 11 / 0 | 8 | 8 (7) | 14/20 | 16 | h 36, text 16/24, icon 18 |
| `sm` | 28 | auto | 9 / 0 | 6 | 8 (7) | 14/20 | 16 | h 32, 16/24, 18 |
| `sm-pill` | 28 | auto | 10 / 0 | 6 | full | 14/20 | 16 | h 32, 16/24, 18 |
| `lg` | 36 | auto | 13 / 0 | 8 | 8 (7) | 14/20 | 16 | h 40, 16/24, 18 |
| `xl` | 40 | auto | 15 / 0 | 8 | 8 (7) | 16/24 | 18 | h 44, 18/28, 20 |
| `xs` | 24 | auto | 7 / 0 | 4 | 8 (7) | 12/16 | 14 | h 28, 14/20, 16 |
| `compact` | 28 | auto | 7 / 0 | 4 | 6 (5) | 14/20 ¹ | 16 ¹ | h 28, 12/16, 14 |
| `micro` | 20 | auto | 5 / 0 | 4 | 4 (3) | 11/16.5 | 12 | same |
| `multiline` | min 32 | auto | 11 / 5 | 8 | 8 (7) | 14/20, wraps | 16 | min 36, 16/24 |
| `sm-multiline` | min 28 | auto | 9 / 3 | 6 | 8 (7) | 14/20, wraps | 16 | min 32, 16/24 |
| `icon` | 32 | 32 | 0 | n/a | 8 (7) | n/a | 16 | 36×36, icon 18 |
| `icon-sm` | 28 | 28 | 0 | n/a | 8 (7) | n/a | 16 | 32×32, 18 |
| `icon-sm-pill` | 28 | 28 | 0 | n/a | full | n/a | 16 | 32×32, 18 |
| `icon-lg` | 36 | 36 | 0 | n/a | 8 (7) | n/a | 16 | 40×40, 18 |
| `icon-xl` | 40 | 40 | 0 | n/a | 8 (7) | n/a | 18 | 44×44, 20 |
| `icon-xs` | 24 | 24 | 0 | n/a | 8 (7) | n/a | 14 (16 inside an InputGroup) | 28×28, 16 |
| `icon-micro` | 20 | 20 | 0 | n/a | 4 (3) | n/a | 16 ¹ | 20×20, 12 |
| `icon-tiny` | 16 | 16 | 0 | n/a | 8 (7) | n/a | 16 ¹ | 16×16, 12 |

¹ The size's own unprefixed icon/text size loses to the base's `sm:` rule on desktop, so
`compact` renders 14px text and `compact`/`icon-micro`/`icon-tiny` render 16px icons unless the
icon carries its own `size-*` class (T3's call sites do: e.g. the toast copy button passes
`size-3`). Port the *measured* value, and honour an explicit icon size.

Common to all sizes: weight **500**, `white-space: nowrap` (except multiline sizes), icon margin
**−2px** left and right, icons `flex-shrink: 0`, cursor pointer. Measured label width for "Label"
in Inter 14px: 36.3px (font-dependent).

Colours by variant at rest (P; the outline row is also M on the PR-list toolbar buttons):

| Variant | Background (light / dark) | Text | Border | Icon | Shadow / lip |
| --- | --- | --- | --- | --- | --- |
| `default` | `#1b4ed8` / `#346bf1` | `#ffffff` | = background | `#ffffff` | `inset 0 1px #ffffff29` + `0 1px 2px` primary 24% (`#194fd93d` / `#326df23d`) |
| `destructive` | `#fb2c36` / `#fb414a` | `#ffffff` | = background | `#ffffff` | `inset 0 1px #ffffff29` + `0 1px 2px` destructive 24% (`#fb2e363d` / `#fb434b3d`) |
| `destructive-outline` | `#ffffff` / `#ffffff09` | `#c10007` / `#ff6467` | `#d4d4d8` / `#ffffff1c` | = text | `0 1px 2px #0000000d`; lip `0 1px #0000000a` / `0 -1px #ffffff0f` |
| `outline` | `#ffffff` / `#ffffff09` | `#18181b` / `#f5f5f5` | `#d4d4d8` / `#ffffff1c` | `#71717b` / `#a1a1a1` | `0 1px 2px #0000000d`; lip as above |
| `secondary` | `#fafafa` / `#ffffff08` | `#27272a` / `#f5f5f5` | transparent | = text | none |
| `ghost` | transparent | `#18181b` / `#f5f5f5` | transparent | `#71717b` / `#a1a1a1` | none |
| `ghost-muted` | transparent | `#71717b` / `#a1a1a1` | transparent | = text | none |
| `ghost-destructive` | transparent | `#71717b` / `#a1a1a1` | transparent | = text | none |
| `glass` | `#fcfcfccc` / `#000000cc` + glass filter | `#18181b` / `#f5f5f5` | `#e4e4e899` / `#ffffff0c` (border 60%) | `#71717b` / `#a1a1a1` | `shadow-sm`; pill radius |
| `link` | transparent | inherited (`#262626` / `#f5f5f5`) | transparent | = text | none |
| `media-close` | `#000000a6` (black 65%) | `#ffffff` | transparent | `#ffffff` | ring `0 0 0 1px #ffffff33` + `shadow-sm` |
| `media-navigation` | transparent | `#ffffffe6` | transparent | = text | none; `position:absolute; top:50%; translate-y:-50%; z-index:20` |
| `overlay` | `#000000b3` (black 70%) | `#ffffffa6` (white 65%) | transparent | = text | none |
| `warning-outline` | `#ff990014` / `#ff9b0029` | `#bb4d00` / `#ffb900` | `#fc980052` (warning 32%) | = text | `0 1px 2px #0000000d` |

The outline background is the *popover* token (white) in light, not `background`
(`#fcfcfc`), so outline buttons read slightly brighter than the inputs next to them.

### 3.2 InlineButton (P)

| Tone | Text (light / dark) | Hover | Other |
| --- | --- | --- | --- |
| `default` | `#18181b` / `#f5f5f5` | underline | |
| `muted` | `#71717b` / `#a1a1a1` | text → foreground, underline | |
| `destructive` | `#fb2c36cc` / `#fb414acc` (80%) | text → `#fb2c36` / `#fb414a`, underline | |
| `picker` | `#18181b` / `#f5f5f5` | underline | gap 4px, underline offset 4px, thickness from font; underlined while its popup is open |

Weight 500, gap 2px, underline offset 2px, font size inherited, `text-align` and `white-space`
overridable via `--inline-button-text-align` / `--inline-button-white-space`.

### 3.3 Kbd, KbdGroup, MenuShortcut, MenuItemDetail

| Part | Box | Padding | Radius | Font | Colours (light / dark) | Marker |
| --- | --- | --- | --- | --- | --- | --- |
| Kbd default | h 20, min-w 20 (one glyph → 20×20) | 0 4px | 4 | 12/16, 500, sans | bg `#1c1c1c12` / `#f1f1f112` (foreground 7%), text `#71717b` / `#a1a1a1` | M (Settings search hint) |
| Kbd outline | h 20 | 0 6px | 4 | 12/16, 500 | transparent; border 1px top/sides + **2px bottom**, `#1b1b1b26` / `#f8f8f826` (foreground 15%) | P |
| Kbd icon | 12×12 | | | | | S |
| KbdGroup | inline-flex | gap 4px | | | | S |
| MenuShortcut | auto (16 tall) | n/a | n/a | 12/16, 500, **letter-spacing 1.2px** (0.1em) | `#71717b` / `#a1a1a1` (sidebar-scope dark `#8f8f8f`) | P |
| MenuItemDetail | auto | start 16px | n/a | inherits row (14/20), tabular numerals | `#71717b` / `#a1a1a1` | P |

### 3.4 Badge (P)

| Size | Height | Min width | Padding x | Radius | Font | Icon | Below 640px (S) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `default` | 18 | 18 | 3 | 4 | 12/16, 500 | 12, opacity .8 | h 22, 14/20, icon 14 |
| `sm` | 16 | 16 | 3 | 4 | 10/10 (`leading-none`) | 12 | h 20, 12/12 |
| `lg` | 22 | 22 | 5 | 4 | 14/20 | 12 | h 26, 16/24 |
| `control` | 24 | 24 | 7 | 8 | 12/16 | 12 | h 28, 14/20 |

Gap 4px; 1px transparent border (except `outline`).

| Variant | Background (light / dark) | Text (light / dark) |
| --- | --- | --- |
| `default` | `#1b4ed8` / `#346bf1` | `#ffffff` |
| `destructive` | `#fb2c36` / `#fb414a` | `#ffffff` |
| `error` | `#ff263314` (destructive 8%) / `#f93e4b29` (16%) | `#c10007` / `#ff6467` |
| `info` | `#2680ff14` (8%) / `#2c7cff29` (16%) | `#1447e6` / `#50a2ff` |
| `success` | `#00bf8014` / `#00bb7c29` | `#007a55` / `#00d492` |
| `warning` | `#ff990014` / `#ff9b0029` | `#bb4d00` / `#ffb900` |
| `outline` | `#fcfcfc` / `#ffffff09`, border `#d4d4d8` / `#ffffff1c` | `#18181b` / `#f5f5f5` |
| `secondary` | `#fafafa` / `#ffffff08` | `#27272a` / `#f5f5f5` |
| `label` (example `--label: #8b5cf6`) | `mix(label 8%)` → `#8c59f214` / `mix(label 12%)` → `#8c5af71f` | `mix(label 30%, foreground)` → `#3a2c5d` / `mix(label 45%, foreground)` → `#c5b0f5` |

### 3.5 Menu

Popup and rows were **M** on two live menus (right-panel script menu; PR-list filter menu with
a submenu); variants not present live are **P**.

| Part | Property | Light | Dark | Marker |
| --- | --- | --- | --- | --- |
| Positioner | z-index / placement | 130, `position:absolute` + `transform: matrix(1,0,0,1,x,y)` | same | M |
| Popup | background / border / filter | dropdown-glass (§3.0) `#ffffffd5`, 1px `#14141d1a`, `blur(12px) saturate(1.14)` | `#111111d5`, 1px `#f5f5f51a`, `blur(16px) saturate(1.08)` | M |
| Popup | shadow | `0 16px 40px -18px #0000008c` | `0 18px 44px -18px #000000cc` | M |
| Popup | radius | 8 | 8 | M |
| Popup | min / max width | min(160, 100vw − 32) / 100vw − 32 (1408 at 1440) | same | M |
| Popup | gap to trigger | 4px (`sideOffset`) | same | M |
| Scroll container | padding / max height | 4 all sides / `--available-height` (viewport − popup top − 5px collision padding) | same | M |
| Item | height / padding / gap | min 28 (32 <640) / 4px 8px / 8 | same | M |
| Item | radius / font | 4 / 14px/20px, 400 (16/24 <640) | same | M |
| Item | text | `#18181b` | `#f5f5f5` | M |
| Item, highlighted | background / text | `#f4f4f5` / `#18181b` | `#ffffff0a` / `#f5f5f5` | M |
| Item icon (direct child svg) | size / opacity / colour / margin | 16 (18 <640) / .8 / `#71717b` / −2px | 16 / .8 / `#a1a1a1` | M |
| Item `destructive` | text + icon | `#c10007` (icon = currentColor) | `#ff6467` | P |
| Item `inset` | padding-start | 32 | 32 | P |
| Item `density="touch"` | min height | 40 | 40 | P |
| Item `variant="ghost"` | h / padding / radius / font / gap | min 28 / **4px 7px** (the item's `py-1` survives the merge; only `px` is replaced) / 6 / 12px/16px, 500 / 4 | same | P (re-probed both schemes) |
| MenuItemLabel | overflow | single line, ellipsis | | S |
| Row with a 24px trailing control | height | 32 (content-driven) | | M |
| Separator | size / margin / colour | 1px × full / 4px 8px / `#e4e4e7` | 1px / same / `#ffffff14` | M / S |
| Group label | height / padding / font | 28 / 6px 8px / 12px/16px, 500, `#71717b` | same, `#a1a1a1` | M |
| Group label `inset` | padding-start | 32 (36 <640) | | P |
| Shortcut / detail | see §3.3 | | | P |
| Sub-trigger | as item + chevron | chevron 16, margin-end −2, opacity .8, `#71717b` | `#a1a1a1` | M |
| Sub-trigger, open | background | `#f4f4f5` (accent) | `#ffffff0a` | M |
| Submenu popup | width / placement | min 160; flush with the trigger row's edge (`sideOffset 0`), top = trigger top − 5 so the first row aligns (`alignOffset −5`) | | M |
| Radio item | height / padding / radius | 28 / 4px 8px / 4 | | M |
| Radio item, checked | background | `#1a1a1a14` (foreground 8%) | `#f2f2f214` | M / P |
| Radio indicator | icon | CheckIcon 14px, stroke 1.75, colour = row text (`currentColor`; radio rows have no muted-icon rule), −2px inline margins (`[&_svg]:-mx-0.5`), at the row end (caller places it last) | `#f5f5f5` | S, screenshot |
| Checkbox item | grid / padding / gap | columns `16px 1fr`, padding 4px 16px 4px 8px, gap 8 → label starts 32px in | | M |
| Checkbox indicator | check svg | 16px, stroke 1.75, colour = row text (`#18181b` / `#f5f5f5`) | | P |
| Checkbox `icon` variant | grid | columns `10px 1fr`, check 14px with −2px margin | | P |
| Checkbox `switch` variant | layout | columns `1fr auto`, gap 16, padding-end 6 | | P |
| ↳ mini switch track | size / pad / colours | 22×14, padding 1, on `#1b4ed8` off `#d4d4d8`, `inset 0 1px #0000000a` | on `#346bf1`, off `#ffffff1c` | P |
| ↳ mini switch thumb | size / colour / travel | 12×12, `#fcfcfc`, `shadow-sm/5`, translate 8px when checked | thumb `#000000` | P |
| MenuSearchInput wrapper | padding | 6px 8px 4px 8px (block 43px tall) | | M |
| ↳ rule | border-bottom / padding-bottom / offset | 1px `#e4e4e7b3` (border 70%) / 6 / translateY −1px | 1px `#ffffff0e` | M / S |
| ↳ rule, focused | border colour | `#1b4ed8` | `#346bf1` | M |
| ↳ search icon | size / position / colour | 16, top 6 left 0, `#71717c8c` (muted 55%) | `#a0a0a08c` | M |
| ↳ input | height / line / padding / font | 26 / 26 / start 20 end 9 / 14px sans | | M |

### 3.6 Popover

| Part | Property | Light | Dark | Marker |
| --- | --- | --- | --- | --- |
| Positioner | z / size vars | 130; `--positioner-width/height`, max `--available-*` | | M |
| Popup (default variant) | surface | dropdown-glass `#ffffffd5`, border `#14141d1a` | `#111111d5`, `#f5f5f51a` | M (model picker) |
| Popup | shadow | `0 16px 40px -18px #0000008c` | `0 18px 44px -18px #000000cc` | M / P |
| Popup | radius / lip | 8 / `0 1px #0000000a` | 8 / `0 -1px #ffffff0f` | M / P |
| Popup | gap to trigger | 4 (default `sideOffset`), `alignOffset` 0 | | M |
| Widths | `sm` / `md` / `lg` / `auto` | 256 / 320 / 384 / content | | P / S |
| Viewport padding | `default` / `compact` / `none` | 16 all sides / 8 block, 12 inline / 0 (viewport radius 7) | | P / M |
| `tooltipStyle` | radius / text / shadow / padding | 6 / 12px, balanced / `shadow-md/5` / 4px 8px | | S |
| PopoverTitle | font | 14px / 14px, 600 | | P |
| `panel` variant | surface | transparent, no border/shadow/filter; z 46; width min(thread-details-panel-width + 16, anchor − 8); viewport 8 / 8 | | S |

### 3.7 Tooltip

| Property | Light | Dark | Marker |
| --- | --- | --- | --- |
| Background | `#ffffff` | `#111111` | M |
| Border | 1px `#e4e4e7` | 1px `#ffffff14` | M |
| Radius / lip radius | 6 / 5 | 6 / 5 | M |
| Lip | `0 1px #0000000a` | `0 -1px #ffffff0f` | M |
| Shadow | `0 4px 6px -1px #0000000d, 0 2px 4px -2px #0000000d` | same | M |
| Text | `#18181b`, 12px / 16.5px (`leading-snug`), 400 | `#f5f5f5` | M |
| Secondary line (callers use `text-muted-foreground`) | `#71717b` | `#a1a1a1` | M |
| Padding | 4px 8px | | M |
| Max width | 320 (`max-w-80`), wraps anywhere, `text-wrap: balance` | | M |
| Gap to trigger | 4 | | M |
| Positioner | z 140, `pointer-events:none`, max width `--available-width` | | M |
| `glass` variant | bg `#ffffffd5` / `#111111d5`, border `#14141d1a` / `#f5f5f51a`, `blur(12px) saturate(1.14)` / `blur(16px) saturate(1.08)`, `shadow-xl` black 25%, no lip | | P |
| `code` variant | mono 11px / 17.875px (`leading-relaxed`), left-aligned, max width 480 | | P |

A two-line live tooltip (title + muted hint) measured 320×61.5 with a 2px column gap set by the
caller.

### 3.8 Input, DraftInput, Label

| Size | Box height | `<input>` height = line height | Padding x | Radius (lip) | Font | Below 640px (S) |
| --- | --- | --- | --- | --- | --- | --- |
| `default` | 32 | 30 | 11 | 8 (7) | 14 | box 36 / input 34, 16px |
| `sm` | 28 | 26 | 9 | 8 (7) | 14 | 32 / 30, 16px |
| `lg` | 36 | 34 | 11 | 8 (7) | 14 | 40 / 38, 16px |
| `compact` | 30 | 28 | 9 | 6 (5) | 12 | same |

P, confirmed M on four live dialog fields (default size).

| Property | Light | Dark | Marker |
| --- | --- | --- | --- |
| Background | `#fcfcfc` (background) | `#ffffff09` (input 32%) | M |
| Border | 1px `#d4d4d8` | 1px `#ffffff1c` | M |
| Shadow | `0 1px 2px #0000000d` | same | P |
| Lip | `0 1px #0000000a` | `0 -1px #ffffff0f` | P |
| Text / caret | `#18181b` | `#f5f5f5` | M |
| Placeholder | `#71717b` | `#a1a1a1` | M |
| `font="mono"` | mono stack, tabular numerals | | P |

Label (M): 14px / 16px, 500, `#18181b` / `#f5f5f5`, inline-flex, gap 8 (16px / 18px below 640).

### 3.9 InputGroup (P; default-variant colours, addon and input padding M on the PR-list search field)

The live PR-list search field is **not** stock geometry. Its caller adds
`**:[input]:h-9 sm:**:[input]:h-8` (`PullRequestListFilters.tsx:109`), so it measures **34** tall
with a 32px `<input>` (line height still 30). The stock group with a default input is 32 tall
(input 30 + 2px border), the P value below. Surface colours, lip, addon and input paddings were M on
that field in both schemes: `#fcfcfc` / `#ffffff09`, border `#d4d4d8` / `#ffffff1c`, addon
padding-start 11, icon 16 at .8, input padding 8 / 11, and `input-control` at `display: contents`.

| Variant | Box | Radius | Surface (light / dark) | Shadow |
| --- | --- | --- | --- | --- |
| `default` | 32 tall with a default input (`lg` size: 36) | 8 | `#fcfcfc` / `#ffffff09`, border `#d4d4d8` / `#ffffff1c`, lip | `0 1px 2px #0000000d` |
| `popover` | 32 | 16 | dropdown-glass `#ffffffd5` / `#111111d5`, border `#14141d1a` / `#f5f5f51a` | none |
| `ghost` | 32 | 8 | transparent, transparent border; hover `muted/40`; **focused: `#fcfcfc` / `#000000` fill plus the full field ring** (border `#1b4ed8` / `#346bf1`, `inset 0 0 0 3px` ring 24%) because the base `has-focus-visible` rules still apply (P, both schemes) | none |

| Part | Value | Marker |
| --- | --- | --- |
| Addon `inline-start` | padding-start 11 (9 next to a `sm` input); icon 16 (18 <640), opacity .8, colour = group text (`#18181b`) | M |
| Input after a start addon | padding-start 8 (6 for `sm` inputs) | M |
| Addon `inline-end` | padding-end 11; trailing `<kbd>` pulls in 5.6px (−0.35rem); a button pulls in 8; a trailing badge 6 | S |
| Addon `block-start` / `block-end` | full-width row, padding 11 (9 next to `sm`), group becomes a column | S |
| Group with a textarea | height auto; textarea min-height 82 (94 <640), padding-block 11, resize none; buttons inside get radius 7 | S |

### 3.10 Textarea (P; disabled state M in the live dialog)

| Size | Textarea min-height | Box height (empty) | Padding x / y | Below 640px (S) |
| --- | --- | --- | --- | --- |
| `default` | 70 | 72 | 11 / 5 | min 82 |
| `sm` | 66 | 68 | 9 / 3 | min 78 |
| `lg` | 74 | 76 | 11 / 7 | min 86 |

Font 14px / 20px (16/24 below 640), max-height 256 then it scrolls, grows with content
(`field-sizing: content`), `resize: vertical` (from Tailwind preflight `textarea { resize: vertical }`, not a T3 class; M), `overflow-wrap: anywhere`.
`rows={n}` → min-height `n × 20 + 2 × padding-y` (rows 3, default → 70, M). Wrapper colours,
shadow, lip and radius equal Input's. **Placeholder** has no `placeholder:text-placeholder`
class, so it falls back to Tailwind preflight `color-mix(currentColor 50%, transparent)`:
`#18181c80` / `#f5f5f580` (M), lighter than an Input's placeholder.

### 3.11 NumberField (P)

| Size | Group height | Input height / line height | Step button width (padding x) | Icon |
| --- | --- | --- | --- | --- |
| `default` | 32 | 30 / 30 | 38 (11) | 16 (18 <640) |
| `sm` | 28 | 26 / **34** ² | 34 (9) | 16 |
| `lg` | 36 | 34 / 34 | 38 (11) | 16 |

² `sm:in-data-[size=sm]:leading-8.5` in source; Chrome still centres the value vertically in
an `<input>`, so it renders centred. Below 640px: default 36/34, sm 32/30, lg 40/38. Group
colours/shadow/lip = Input; buttons have 7px radius on their outer corners only; the value is
centred with tabular numerals.

### 3.12 Select

Trigger (**M**, Settings rows, size `sm`; other sizes **P/S**):

| Size | Height | Padding x | Gap | Radius | Font | Below 640px (S) |
| --- | --- | --- | --- | --- | --- | --- |
| `default` | min 32 | 11 | 8 | 8 | 14/20 | min 36, 16/24 |
| `sm` | min 28 (M) | 9 | 6 | 8 | 14/20 | min 32 |
| `lg` | min 36 | 11 | 8 | 8 | 14/20 | min 40 |
| `compact` | 28 | 7 | 4 | 6 | 14/20 (note ¹, §3.1) | 28, 12/16 |
| `xs` | 24 | 7 | 4 | 6 | 12/16 | 28, 14/20 |

| Property | Light | Dark | Marker |
| --- | --- | --- | --- |
| Default variant surface | `#fcfcfc`, border `#d4d4d8`, `0 1px 2px #0000000d`, lip `0 1px #0000000a` | `#ffffff09`, `#ffffff1c`, lip `0 -1px #ffffff0f` | M |
| Default variant min width | 144 (`min-w-36`), `width:100%` unless overridden | | S |
| Value text | `#18181b`, weight 400; placeholder `#71717b` | `#f5f5f5`; `#a1a1a1` | M / S |
| Chevron | 14×14, margin-end −4, opacity .7, `#71717b` (icon-muted) | `#a1a1a1` | M |
| `ghost` variant | transparent, text `#71717b` (secondary-label) → hover/pressed bg `#f4f4f5`, text foreground | bg `#ffffff0a` | S |

Popup (**M**, composer select, dark; light from tokens):

| Part | Value |
| --- | --- |
| Frame | dropdown-glass, radius 8, popup elevation shadow, `min-width = trigger width` |
| List | padding 4, max height `--available-height`, scrolls |
| Group label | 28 tall, padding 6px 8px, 12px/16px 500, muted |
| Item | min 28, padding 4px 8px, radius 4, 14px/20px, text foreground |
| Item, selected | `#1a1a1a14` / `#f2f2f214` (foreground 8%), **stays when highlighted** |
| Item, highlighted (not selected) | accent `#f4f4f5` / `#ffffff0a` |
| Scroll arrows | 24 tall, chevron 16 (18 <640), cursor default, z 50. The `::before` gradient is `h-[200%]` (48px), inset 1px from the sides and the outer edge, with radius 7 on the outer corners. It is solid `popover` over the arrow's own 24px, then fades to transparent over the next 24px of the list (`from-50%`) (S) |
| Placement | item-aligned over the trigger by default (`data-side="none"`; items then get min-width = trigger + 20); otherwise below, start-aligned, 4px gap (the measured popup: 4px below a 24px trigger) |

### 3.13 Combobox (M, composer branch picker, dark; light colours from tokens)

| Part | Value (light / dark) |
| --- | --- |
| Frame | dropdown-glass, radius 8, popup elevation shadow, min width = anchor, max height min(available, 368) |
| Search block | padding 10px 12px 0; rule 1px border 70% → ring when focused (`#1b4ed8` / `#346bf1`, M); icon 16 muted 55%; input 26 tall, padding-start 20, end 9, 14px |
| List | padding 4 (scroll padding 4 block) inside a ScrollArea with stable gutter and edge fade |
| Item | 28 tall, padding 4px 8px, radius 4, 14/20, content gap 8, cursor pointer |
| Item, hover or highlighted | foreground 9%: `#16161617` / `#f4f4f417` (M) |
| Item, selected (not highlighted) | foreground 6%: `#1111220f` / `#eeeeee0f` (`selectedFill`, default on) |
| Empty | padding 8 when non-empty, centred 14px/20px muted |
| Status | 12/16 500 muted, padding 8px 12px; collapses when empty |
| Standalone input | Input box; trigger/clear button 28×28 (32 <640) at end 2px (0 for `sm`), centred vertically, opacity .8 → 1 on hover, chevrons-up-down / X 16px; input padding-end 28 (26 for `sm`) when a button follows; start addon padding 11, input padding-start 31 (`sm` 27) |

### 3.14 Autocomplete (S, derived from tokens)

Frame: solid `#ffffff` / `#111111` (popover), 1px border `#e4e4e7` / `#ffffff14`, radius 8,
`shadow-lg/5`, lip; items as Combobox geometry but cursor default, hover/highlighted `accent`
(`#f4f4f5` / `#ffffff0a`), selected `accent/50` (`#f3f3f580` / `#ffffff05`); groups separated
by 6px; list padding 4 (scroll padding 4 block) inside a ScrollArea with stable gutter and edge fade; popup max height
min(available, 368) like Combobox; selected + highlighted shows accent (an explicit
`[&[data-highlighted][data-selected]]` rule); clear button 28×28 at end 2px, opacity .8 → 1 on hover.

### 3.15 Switch (M in Settings and the dialog; `sm` P)

| Size | `--thumb-size` | Track (w × h) | Padding | Thumb | Travel when checked | Below 640px (S) |
| --- | --- | --- | --- | --- | --- | --- |
| `default` | 16 | 30 × 18 | 2 | 14 × 14 | 12px | thumb-size 20: track 38×22, thumb 18, travel 16 |
| `sm` | 14 | 26 × 16 | 2 | 12 × 12 | 10px | thumb-size 16: 30×18, thumb 14, travel 12 |

| State | Track (light / dark) | Thumb (light / dark) |
| --- | --- | --- |
| off | `#d4d4d8` / `#ffffff1c` (input) | `#fcfcfc` (background) / `#a1a1a1` (muted-foreground) |
| on | `#1b4ed8` / `#346bf1` (primary) | `#fcfcfc` / **`#000000`** (background) |
| mixed | input colour; thumb centred (travel/2), opacity .7 | |

Thumb shadow `shadow-sm/5` (`0 1px 3px #0000000d, 0 1px 2px -1px #0000000d`), radius = thumb
size (round). Track radius full. Off-state thumb sits at x = 2, on-state at x = 2 + travel;
`transform-origin` is the left edge when off and the right edge when on (for the press stretch).

### 3.16 Toggle (P; header panel toggles M)

| Size | Height | Min width | Padding x | Radius | Font |
| --- | --- | --- | --- | --- | --- |
| `default` | 32 (36 <640) | 32 | 7 | 8 | 14 (16 <640) |
| `sm` | 28 (32) | 28 | 5 | 8 | 14 |
| `lg` | 36 (40) | 36 | 9 | 8 | 14 |
| `xs` | 24 (28) | 24 | 3 | 6 | 14 |
| `compact` | 28 | 28 | 3 | 6 | 14 (note ¹, §3.1; 12 <640) |
| `segmented` | 24 | 0 | 10 | 6 | 14 (note ¹, §3.1; 12 <640) |

Icons 16 (18 <640) at opacity .8, weight 500 (pill: 400), gap 8, 1px border.

| Variant | Off (light / dark) | On = `data-pressed` (light / dark) |
| --- | --- | --- |
| `default` | transparent, text `#18181b` / `#f5f5f5` | bg `#d5d5d8a3` / `#ffffff12` (input 64%) |
| `ghost` | transparent | bg `#f4f4f5` / `#ffffff0a` (accent); disabled keeps opacity 1 but text → muted |
| `outline` | bg `#fcfcfc` / `#ffffff09`, border `#d4d4d8` / `#ffffff1c`, `0 1px 2px #0000000d`, lip | bg `#d5d5d8a3` / `#ffffff1c` (dark uses full `input`), no shadow, no lip |
| `pill` | transparent, text `#71717b` / `#a1a1a1`, padding x 12, full radius, weight 400 | bg input 64%, text foreground |
| `segmented` | transparent, text `#71717b` / `#a1a1a1` | bg `#fcfcfc` / `#ffffff14` (input 72%), text foreground, `0 1px 2px #0000001a` |

The live header panel toggles are `variant="ghost" size="sm"` (`PanelLayoutControls.tsx:93–123`), not a
caller override. They measure 28×28, radius 8, transparent border, 16px icons at .8, `transition: box-shadow`,
and both pressed and hover fill accent `#f4f4f5` / `#ffffff0a` (M, both schemes).

### 3.17 ToggleGroup (P)

Segmented (the default variant): container `#d4d4d766` / `#ffffff0b` (input 40%), padding 2,
gap 2, radius 8 → 28 tall with 24px items; pressed item `#fcfcfc` / `#ffffff14` with
`0 1px 2px #0000001a`; idle items transparent, text muted. `default` variant: gap 2, no
container fill. `outline` (joined): no gap, adjacent borders collapsed, only the outer corners
rounded.

### 3.18 Separator (P)

1px, `#e4e4e7` / `#ffffff14`. Horizontal: full width. Vertical: 1px wide, stretches to the
row height unless given a height class.

### 3.19 ScrollArea (P for the Base UI scrollbar; viewport M)

| Part | Value (light / dark) |
| --- | --- |
| Vertical scrollbar | width 6, inset 4 top/bottom and 1 right |
| Horizontal scrollbar | height 6, inset 4 left/right and 1 bottom |
| Thumb | full radius, `#d9d9d9` / `#ffffff14`; hovered thumb `#bfbfbf` / `#ffffff1f`; min length 16 (S) |
| Scrollbar visibility | opacity 0 at rest; 1 while the pointer is over the scroll area (`data-hovering`) or within 500ms of a scroll (`data-scrolling`) |
| Edge fade (`scrollFade`) | `mask-image` linear gradients on each overflowing edge; fade length = min(24px, distance scrolled from that edge), so it eases in as you leave an edge; off while the viewport has keyboard focus |
| Fade scroll padding | `scroll-padding: 24px` (keyboard/scrollIntoView keeps items clear of the fade) |
| Viewport focus | control ring (§3.0) inside the viewport |
| Native scrollbars elsewhere (`index.css:1851–1866`) | `::-webkit-scrollbar` width 6, transparent track, thumb radius 3 with the same colours |
| Markdown tables | horizontal bar always visible, thumb `muted-foreground` 60% → 80% on hover |

The sidebar thread list and Settings navigation use `hideScrollbars` (no bar at all, M:
`scrollbar-width: none`) with a 0.75rem fade set by their caller.

### 3.20 Spinner (P)

LoaderCircle, stroke 1.75. Sizes `xs` 12, `sm` 14, `md` 16, `lg` 20, none = inherited (16 in a
default Button). Tones: `current` (inherits), `muted` (`#71717b` / `#a1a1a1`). Animation:
`spin 1s linear infinite` (360° per second) when running.

### 3.21 Dialog (M, "Add action" dialog; footer `default` variant and nested values P/S)

| Part | Property | Light | Dark |
| --- | --- | --- | --- |
| Backdrop | fill / filter / z | `#fcfcfc99`, `blur(4px)`, 50 | `#000000a3`, `blur(4px)`, 50 |
| Viewport | layout | fixed, inset 0, z 50, padding 16, grid rows `1fr auto 1fr` (popup vertically centred), items centred horizontally | |
| Popup | width | 100% up to 512 (`max-w-lg`), height ≤ viewport − 32 | |
| Popup | surface | dialog-glass `#fcfcfccc` + `blur(12px) saturate(1.14)`, 1px `#14141d1a` | `#000000cc` + `blur(16px) saturate(1.08)`, 1px `#ffffff14` |
| Popup | radius / shadow | 16 / `0 24px 64px -24px #000000a6` | 16 / `inset 0 1px #ffffff0a, 0 24px 72px -20px #000000e6` |
| Popup | text colour | `#18181b` (popover-foreground) | `#f5f5f5` |
| Header | padding / gap | 24 (bottom 12 when a panel follows; bottom 16 below 640) / 8 | |
| Title | font | 20px / 27.5px (`leading-snug`), 600, wraps anywhere | |
| Description | font / colour | 14px / 20px, `#71717b` | `#a1a1a1` |
| Panel | padding | 24 inline; 4 top after a header; 4 bottom before a bare footer (else 24); 16 between children | |
| Panel scroll | ScrollArea with 24px edge fade | | |
| Footer `bare` | padding / layout | 16 / 24; row, end-aligned, gap 8 (column-reverse below 640) | |
| Footer `default` | adds | top border 1px `#e4e4e7`, fill `#f9f9f9b8` (muted 72%), bottom radius 15 | border `#ffffff14`, fill `#ffffff06` |
| Close button | box / place / icon | ghost `icon` Button 32×32, 8px inside the border from the top and end edges (9px from the outer edge, M); X 16px `#71717b` | X `#a1a1a1` |

Below 640px with `bottomStickOnMobile` (default): viewport rows `1fr auto`, padding 0 with 48px
top, popup full width, square corners, top border only (S).

### 3.22 AlertDialog (S; shares Dialog's measured surface)

Same backdrop, viewport and popup values as §3.21 (`max-w-lg`, no close button). Header 24
padding, gap 8, **centred text below 640px, start-aligned from 640px** (P). Description
14px/20px muted, scrolls past `min(24rem, 60dvh)`, text selectable. Footer `default` as
Dialog's; `bare`: padding 0 / 24 / 24 / 24 (bottom only, P).

### 3.23 Sheet (P/S)

| Part | Value (light / dark) |
| --- | --- |
| Backdrop | `#fcfcfc99` / `#00000099` (background 60%), `blur(4px)`, z 46 (`--z-sheet`) |
| Popup surface | solid popover `#ffffff` / `#111111`, `shadow-lg/5`, lip (hidden below 640), 1px border on the inner edge `#e4e4e7` / `#ffffff14` |
| Right / left sheet | width `100% − 48px`, max 448 (`max-w-md`); border on the start / end edge |
| Top / bottom sheet | full width; 48px gap kept on the far side (viewport padding); border bottom / top |
| `inset` variant | viewport padding 16 (≥640); popup radius 16 with full border, no lip |
| Close button | as Dialog's |

### 3.24 Toast

Stacked viewport (**M**, toasts raised locally with placeholder text, `top-right`):

| Part | Property | Light | Dark |
| --- | --- | --- | --- |
| Viewport | position / width | fixed, top 84 (= 32 inset + 52 top bar), right 32, width min(100% − 64, 360); z 100 | |
| Viewport inset | ≥640 / <640 | 32 / 16 (S) | |
| Toast | surface | dropdown-glass `#ffffffd5`, border `#14141d1a`, `blur(12px) saturate(1.14)` | `#111111d5`, `#f5f5f51a`, `blur(16px) saturate(1.08)` |
| Toast | radius / shadow | 8 / `0 20px 25px -5px #00000040, 0 8px 10px -6px #00000040` | same |
| Content (inline layout) | padding | 12 top/bottom, 14 start, 14 end (24 end when trailing controls exist); row, items centred, space-between, gap 6 | |
| Content (stacked action layout) | padding | 10 block, 14 inline; column, gap 8; body padding-end 20; actions right-aligned full row | |
| Body | gap | 2 between title row and description | |
| Title row | gap / padding-end | 8 / 24 when there are no trailing controls (keeps clear of the close orb) | |
| Type icon box | size | 16 wide × one line (20) tall | |
| Type icon colour | error / info / success / warning | `#fb2c36` / `#2b7fff` / `#00bc7d` / `#fe9a00` | `#fb414a` / `#2b7fff` / `#00bc7d` / `#fe9a00` |
| Loading icon | | Spinner, opacity .8 | |
| Title | font | 14px / 20px, 500, `#18181b` | `#f5f5f5` |
| Description | font | 14px / 20px, `#71717b`; error descriptions ≥ 180 chars clamp to 4 lines | `#a1a1a1` |
| Close orb | box / place | 24×24 circle, `top: −6px; right: −6px`, z 20. The offsets are from the toast's padding box (inside its 1px border), so the orb overhangs the **outer** edge by 5px (M: orb at x = toast right − 19, y = toast top − 5) | |
| Close orb | surface | bg `#ffffffeb` (popover 92%), border `#e4e4e899` (border 60%), `blur(8px)`, `shadow-sm`; X 12px stroke 2.25 `#71717b` | bg `#111111eb`, border `#ffffff0c`, X `#a1a1a1` |
| Copy-error button | box / icon | 20×20 ghost-muted micro button, radius 6, icon 12, colour `#70707acc` (muted 80%); hover keeps a transparent fill (`hover:bg-transparent`) and the icon goes to full `muted-foreground`; once copied the icon becomes a 12px `CheckIcon` in `success` `#00bc7d` and the label/tooltip reads "Copied error" (S, `toast.tsx:145–167`) | muted 80%; copied `#00bc7d` |
| Action buttons | | Button `xs` (24 tall), `default` variant for the primary action, `outline` for secondary/additional | |
| Expand toggle | | 12px 500 muted text, chevron 14px stroke 2.25 opacity .8, gap 4, panel margin-top 8, max height 160 then scrolls | |
| Single-line toast height | | 46 (title only), 68 (title + 1-line description), 88 (2-line description) | |

Collapsed stack (M): newest on top; each older toast *i* (i = 1, 2…) is `scale(1 − 0.1·i)`
from its top edge, moved down `12·i + 0.1·i·H₀` px where H₀ is the newest toast's height
(measured: i=1 → 0.9 and 18.8px for H₀ = 68; i=2 → 0.8 and 37.6px), forced to H₀ tall, its
content at opacity 0; z-index 9999 − i. Expanded (pointer over the viewport, M): scale 1, real
heights, `translateY(sum of newer heights + 12·i)` (measured 80 and 138 for heights 68 and 46).
At most 3 toasts (Base UI `limit`); extras get `data-limited` and fade to opacity 0.

Anchored toasts (**M**, re-measured): positioner z 100, max width min(256, available), placed
**4px above** the anchor. Base UI's `Toast.Positioner` defaults to `side: "top"` and `align: "center"`,
with collision padding 5. T3 only passes `sideOffset: 4`. The toast flips below the anchor (still
4px) when there is no room above, and shifts along the edge to stay on screen. Verified with a
mid-screen probe anchor (`data-side="top"`, the toast's bottom edge 4px above the anchor), then an
anchor at the top edge (flipped to `data-side="bottom"`, 4px below). `tooltipStyle` ("Copied!"):
radius 6, padding 4px 8px, 12px / 16px 400, glass surface, `shadow-xl` black 25%. Regular
(e.g. "Failed to copy"): radius 8, content 12px 14px, 14px text, close orb as above.

### 3.25 Collapsible and CollapsibleSectionHeader

Collapsible panel (P): overflow hidden; animated height (§6). Trigger: cursor pointer only.

CollapsibleSectionHeader (**M**, sidebar, light; dark from sidebar tokens):

| Part | Value (light / dark sidebar scope) |
| --- | --- |
| Button | 32 tall, full width, padding 0 8 (`panel`: start 8, end 10), gap 8, radius 6 |
| Label | 12px / 16px, 500, truncates; tone `muted` `#71717b` / `#a3a3a3` |
| Leader rule | 1px, flex 1, min width 8; `muted` tone `#e3e3e780` / `#ffffff0a` (sidebar border 50%); invisible when an accessory is shown |
| Accessory (caller) | e.g. a count, 12px 400 tabular |
| Chevron | 12×12, muted; rotated −90° when collapsed, 0° when expanded |
| Focus | ring 2px inset `ring/70`, no offset band |
| `emphasized` tone | label foreground 80%, rule foreground 25% |
| `accent` tone | label `#1b4ed8` / `#346bf1`, rule primary 50% |
| SectionHeaderStatus | 10px, line height 1, `#c10007` / `#ff6467` |

### 3.26 Empty (P; ≥768px for padding)

| Size | Padding | Gap | Min height | Title | Description |
| --- | --- | --- | --- | --- | --- |
| `default` | 48 (24 <768) | 24 | n/a | 20px / 28px, 600 | 14px / 20px, muted, margin-top 4 |
| `compact` | 40 (24 <768) | 16 | 256 | 17px / 24px, 600 | 13px / 18px |
| `hero` | 48 (24 <768) | 24 | n/a | 30px / 36px (24/32 <640) | 14px / 20px |

Title colour inherits (`#262626` / `#f5f5f5`), description `#71717b` / `#a1a1a1`. Media `icon`
card: 36×36, radius 6, `#ffffff` / `#111111` (card), border `#e4e4e7` / `#ffffff14`,
`shadow-sm/5`, lip, icon 18; two back cards at scale .84, rotated ±10°, nudged ∓2px, anchored
bottom-left / bottom-right, no shadow; media margin-bottom 24 (0 in `compact`). Content block:
max width 384, gap 16, 14px.

### 3.27 Skeleton (P)

Fill `#72727926` / `#a1a1a126` (muted-foreground 15%). Radius: `block` 4, `card` 8, `pill` full.

### 3.28 PanelTabCloseButton (P)

16×16, radius 4, hover fill `muted` (`#fafafa` / `#ffffff08`). Slot 12×12 shows the tab's own
icon until the tab is hovered (or the button is keyboard-focused), then an X 12px replaces it.

### 3.29 MiddleTruncate (M)

Inherits font and colour. A live overflowing branch label showed the head clipped with
`clip-path: inset(-0.25em 3.83px -0.25em 0)` and the tail moved by `translate: -3.83px` (the
sub-glyph slack), the ellipsis drawn as the tail's `::before` content "…". Non-overflowing
values render without clip or translate.

### 3.30 Layers and placement defaults (S, confirmed M where a popup was opened)

| Surface | z-index | Default side / align / sideOffset / alignOffset |
| --- | --- | --- |
| Sheet backdrop + viewport, panel popover | 46 (`--z-sheet`) | n/a |
| Dialog, AlertDialog backdrop + viewport | 50 (media dialog 60) | centred |
| Toast viewport, anchored toast positioner | 100 | anchored: **top** / center / 4 (flips to bottom when there is no room above, M) |
| Menu, Popover, Select, Combobox, Autocomplete positioners, popover touch backdrop | 130 | menu bottom / center / 4 / 0; submenu inline-end / start / 0 / −5; popover bottom / center / 4 / 0; select bottom / start / 4 / 0 (item-aligned by default); combobox and autocomplete bottom / start / 4 |
| Tooltip positioner | 140 | top / center / 4 |

Collision padding is Base UI's 5px everywhere (tooltips add the native titlebar height on top);
Base UI flips to the opposite side and shifts along the axis to stay inside the viewport (M:
a submenu flipped from inline-end to inline-start next to the right edge).

---
## 4. States

Selectors are Tailwind's; "hover" for Button-like controls means `:hover` **or**
`[data-pressed]` (the attribute Base UI puts on a trigger while its popup is open), because the
classes are written `[:hover,[data-pressed]]:…`.

### 4.1 Button (P, both schemes; live outline buttons M)

| Variant | Hover / `data-pressed` (light / dark) | `:active` (mouse down) | Disabled |
| --- | --- | --- | --- |
| `default` | bg `#1b4ed8e6` / `#346cf1e6` (primary 90%); shadows unchanged on hover | inset highlight → `inset 0 1px #00000014` (black 8%), drop shadow none | opacity .64, no shadows at all |
| `destructive` | bg `#fb2c36e6` / `#fb414ae6` | as default | as default |
| `destructive-outline` | border `#fc2c3552` / `#fc414b52` (destructive 32%), bg `#ff33330a` / `#ff4d4d0a` (4%) | drop shadow none; lip none (light) / `0 -1px #ffffff05` (dark) | opacity .64, no shadow, no lip |
| `outline` | bg `#f3f3f580` (accent 50%) / `#ffffff12` (input 64%) | as destructive-outline | as destructive-outline |
| `secondary` | bg `#f9f9f9e6` / `#ffffff07` (secondary 90%) | `secondary/80` is declared but hover's 90% wins while the pointer is over it (M) | opacity .64 |
| `ghost` | bg `#f4f4f5` / `#ffffff0a` (accent) | n/a | opacity .64 |
| `ghost-muted` | bg accent, text + icon `#18181b` / `#f5f5f5` | n/a | opacity .64 |
| `ghost-destructive` | bg accent, text + icon `#fb2c36` / `#fb414a` | n/a | opacity .64 |
| `glass` | border `#e4e4e7` / `#ffffff14` (full border colour) | n/a | opacity .64 (shadow-sm kept) |
| `link` | underline, offset 4px | n/a | opacity .64 |
| `media-close` | bg `#000000cc` (black 80%) | n/a | |
| `media-navigation` | bg `#ffffff1a` (white 10%), text `#ffffff` | n/a | |
| `overlay` | bg `#000000e6` (black 90%) | n/a | |
| `warning-outline` | border `#fc980066` (warning 40%), bg `#ff9b0029` (16%) / `#ff9b003d` (24%) | drop shadow none | opacity .64, no shadow |

* **Press scale.** Any button **without** `aria-haspopup` scales to **0.97** while `:active`
  (M), with `will-change: scale` while enabled. Menu/popover/select triggers (they carry
  `aria-haspopup`) do not scale; they show the `data-pressed` styles instead.
* **`data-pressed`** (popup open) on `default` = hover colour + the `:active` inset (M);
  on `outline` = hover fill, no drop shadow, no lip (M).
* **Focus-visible.** Control ring (§3.0) in every variant; `media-close` and
  `media-navigation` use a white ring (`media-navigation` with a transparent offset band).
* **`aria-disabled="true"`.** Opacity .64 and `cursor: not-allowed` but pointer events stay on
  (so a tooltip can explain why), shadows stay.
* **Loading.** No built-in state; callers swap the icon for a `Spinner` (it inherits the
  button's icon size and colour).

### 4.2 InlineButton, Badge, Kbd

* InlineButton: hover underline (+ tone colour change, §3.2); focus-visible 2px ring outline
  inset 2px; disabled opacity .64, no pointer events; `picker` underlines while
  `data-popup-open`.
* Badge (only when rendered as `<button>`/`<a>`): hover `default` → `#1b4ed8e6` / `#346cf1e6`,
  `destructive` → destructive 90%, `outline` → `#f3f3f580` / `#ffffff0d` (input 48%),
  `secondary` → secondary 90%; cursor pointer; focus-visible control ring; disabled .64.
* Kbd: static (pointer-events none, not selectable).

### 4.3 Menu

| State | Applies to | Value (light / dark) | Marker |
| --- | --- | --- | --- |
| `data-highlighted` (pointer over the row, or keyboard focus) | item, checkbox, radio, sub-trigger | bg `#f4f4f5` / `#ffffff0a`, text foreground | M |
| `data-highlighted` + destructive | item | bg accent, text stays `#c10007` / `#ff6467` | P |
| `data-disabled` | rows | opacity .64, `cursor: not-allowed`, no pointer events (callers can re-enable pointer events to host a tooltip). Still keyboard-reachable: Base UI keeps disabled rows focusable (`focusableWhenDisabled`), so arrow keys, Home/End and typeahead can highlight one (accent under the .64 opacity), but Enter/Space/click do nothing | P / S |
| `data-checked` | radio | bg `#1a1a1a14` / `#f2f2f214`; highlighted wins over checked (bg accent) | M / P |
| `data-checked` | checkbox | indicator visible (unchecked: unmounted, column stays reserved) | P |
| `data-checked` | checkbox `switch` | track primary, thumb shifted 8px, origin right | P |
| `data-popup-open` | sub-trigger | bg accent, text accent-foreground (kept while the pointer is in the submenu) | M |
| focus inside search | MenuSearchInput | rule → ring colour (150ms colour transition) | M |
| `busy` | MenuSearchInput | search icon replaced by `Spinner md muted` at the same spot | S |
| open | trigger | `aria-expanded="true"`, `data-popup-open`, `data-pressed` | M |
| `data-side`, `data-align` | positioner + popup | actual side after collision (`bottom`, `top`, `inline-start`, `inline-end`, `none`) | M |
| `data-nested` | submenu popup | marks the nested popup | M |

### 4.4 Popover, Tooltip

* Popover popup: `data-open` / `data-closed`; `data-starting-style` and `data-ending-style`
  during the enter/exit transition (scale .98, opacity 0); `data-instant` disables the
  positioner's transform transition; `data-transitioning` on the viewport while content
  cross-fades (overflow clipped then). Trigger: `data-popup-open`, `aria-expanded`.
* Tooltip popup: `data-open`, `data-starting-style` / `data-ending-style` (scale .98, opacity
  0), `data-instant="focus" | "dismiss" | "trigger-change"` (and `delay` inside a provider's
  instant window) → `transition-duration: 0`; positioner `data-instant` → no position
  transition. Trigger has no visual state of its own.

### 4.5 Fields (Input, InputGroup, Textarea, NumberField, Select trigger)

| State | Border (light / dark) | Box shadow | Lip | Other |
| --- | --- | --- | --- | --- |
| rest | `#d4d4d8` / `#ffffff1c` | `0 1px 2px #0000000d` | yes | |
| hover | no hover style | | | (InputGroup `ghost`: bg `#fafafa66` / `#ffffff03`) |
| focus-visible (input focused) | `#1b4ed8` / `#346bf1` | `inset 0 0 0 3px #194fd93d` / `#326df23d` | no | M (P for Textarea/NumberField, M for InputGroup) |
| `aria-invalid` | `#fc2c355c` / `#fc404b5c` (destructive 36%) | none | no | P |
| invalid + focus | `#fa2c37a3` / `#fa424aa3` (64%) | `inset 0 0 0 3px` destructive 16% `#f92c3829` / 24% `#fb434b3d` | no | P; NumberField uses destructive **48%** in light |
| disabled | unchanged | none | no | opacity .64 on the whole box (M in the dialog) |
| autofill | | | | bg foreground 4% `#1a1a1a0a` / 8% `#f2f2f214` |

* InputGroup `popover` variant: focused border = foreground 18% (`#16161c2e` / `#f4f4f42e`),
  no ring. `ghost` variant: focused fill = background (`#fcfcfc` / `#000000`) **and** the normal
  field ring: border `#1b4ed8` / `#346bf1` and `inset 0 0 0 3px #194fd93d` / `#326df23d` (P, both
  schemes). Only `popover` suppresses the ring.
* Select trigger `default` also drops its shadow and lip while `data-pressed`
  (popup open) and dims to .64 with `data-disabled`; `ghost` hover/pressed: bg accent, text
  foreground; ghost focus ring is a plain 2px inset ring (no offset band).
* `SelectValue` with `data-placeholder`: text `#71717b` / `#a1a1a1`.
* NumberField step buttons: hover bg `#f4f4f5` / `#ffffff0a` (150ms colour transition);
  whole group `data-disabled` → opacity .64, no pointer events.
* Combobox/Autocomplete wrapper: `has-disabled` → opacity .64 (the inner input-control resets
  its own opacity so it is not dimmed twice); trigger/clear buttons opacity .8 → 1 on hover.

### 4.6 Select, Combobox and Autocomplete items

| State | Select item | Combobox item | Autocomplete item |
| --- | --- | --- | --- |
| hover | via highlight | foreground 9% (`#16161617` / `#f4f4f417`) | accent |
| `data-highlighted` | accent (`#f4f4f5` / `#ffffff0a`) | foreground 9% | accent |
| `data-selected` | foreground 8% (`#1a1a1a14` / `#f2f2f214`) | foreground 6% (`#1111220f` / `#eeeeee0f`), text foreground | accent 50% (`#f3f3f580` / `#ffffff05`) |
| selected + highlighted | **selected fill wins** (M) | foreground 9% | accent |
| `data-disabled` | opacity .64, not-allowed | opacity .64, not-allowed | opacity .64 |

(Tailwind emits `data-*` variants in alphabetical order, so `data-selected` beats
`data-highlighted` in Select while `data-highlighted` beats `data-checked` in menu radio rows.)

### 4.7 Switch

| State | Value |
| --- | --- |
| `data-unchecked` / `data-checked` | §3.15 colours; thumb at x 2 / 2 + travel |
| `:active` (pointer down on the switch or its label) | thumb `scale-x: 1.1` from its origin side (left when off, right when on) and radius `thumb/(thumb×1.1)` (slightly elliptical), after a 100ms delay |
| focus-visible | control ring |
| `data-disabled` | opacity .64, `cursor: not-allowed` (M) |
| `data-mixed` | track input colour, thumb centred at opacity .7, `aria-checked="mixed"` |
| readOnly | Base UI ignores clicks; no visual change |

### 4.8 Toggle and ToggleGroup

| Variant | Hover (light / dark) | Pressed | Disabled |
| --- | --- | --- | --- |
| `default`, `pill` | bg accent `#f4f4f5` / `#ffffff0a` | §3.16 | opacity .64 |
| `ghost` | bg accent | bg accent | opacity 1, text muted, icons opacity 1 |
| `outline` | bg accent / `#ffffff12` (input 64%) | §3.16, no shadow, no lip (P). `:active` (mouse down, not pressed): drop shadow none; light lip none, dark lip `0 -1px #ffffff05` (white 2%) (S) | opacity .64 |
| `segmented` | bg `#fbfbfb8c` (background 55%) / `#ffffff09` (input 32%), text foreground | §3.16 | opacity .64 |

Focus-visible: control ring; inside a group the focused item is raised to `z-index: 10`.
`aria-pressed` mirrors `data-pressed`.

### 4.9 ScrollArea

`data-hovering` / `data-scrolling` on the scrollbar → opacity 1; thumb `:hover` → hover colour;
viewport `:focus-visible` → ring, edge fade removed; root/viewport `data-has-overflow-x|y` and
`data-overflow-{x,y}-{start,end}` flags reflect scroll position (used for the fade and
`overscroll-behavior-x: contain`).

### 4.10 Dialog, AlertDialog, Sheet

| State | Value |
| --- | --- |
| `data-starting-style` / `data-ending-style` | popup scale .98, opacity 0; backdrop opacity 0; sheet popup opacity 0 + 32px offset toward its edge |
| `--nested-dialogs: N` on a parent popup (a child dialog is open) | translateY −20·N px, scale 1 − 0.1·N, opacity 1 − 0.1·N; `data-nested-dialog-open` → transform origin top |
| `data-nested` child popup enter/exit | also starts/ends 32px lower |
| mobile sheet mode (<640) | enter/exit 16px lower; parent hidden (opacity 0) while a child is open |
| `initialFocus` | first enabled `[data-slot=dialog-action]`, else Base UI's default (first tabbable); touch opens focus the popup itself |

### 4.11 Toast

| State | Value |
| --- | --- |
| `data-type` | error, info, success, warning, loading → icon and colour (§3.24) |
| not expanded, index > 0 | scaled/peeking, content opacity 0 and pointer events none |
| `data-expanded` (viewport hovered or focused) | full layout, content opacity 1 |
| `data-limited` | beyond the 3-toast limit: opacity 0 |
| `data-swipe-direction` + ending | exits along the swipe |
| close orb hover | bg popover `#ffffff` / `#111111`, X → foreground; focus-visible control ring |
| expandable open | chevron up, label "Hide details", panel shown; closed: chevron down, "Show details" |
| description as toggle (hover) | row bg muted 40%, description underlined (offset 2, muted 60% decoration) |

### 4.12 Collapsible pieces, Empty, Skeleton, PanelTabCloseButton, MiddleTruncate

* Collapsible panel: `data-open` / `data-closed`, height transition between 0 and
  `--collapsible-panel-height`; trigger `aria-expanded`, `data-panel-open`.
* CollapsibleSectionHeader: expanded (chevron 0°) / collapsed (−90°); no hover style;
  focus-visible 2px inset `ring/70`.
* Skeleton: always animating (motion-safe).
* PanelTabCloseButton: tab hover → X replaces the slot; button hover → bg muted; focus-visible
  on the button → X shown.
* MiddleTruncate: `data-overflow` on the head span when the head is cut; then the tail shows
  the "…" prefix.

---
## 5. Behaviour

Base UI behaviour below is Base UI 1.9.0 as shipped (defaults read from the bundle, §1.3) and,
where marked **M**, observed live with real key presses and pointer moves.

### 5.1 Button, InlineButton

* Renders `<button type="button">`; with `render` (e.g. a link) the element changes and `type`
  is omitted (`button.tsx:83–99`).
* `disabled` removes pointer events (no hover, no tooltip); `aria-disabled` keeps them and
  shows `not-allowed`. Activation is native button behaviour (Enter on keydown, Space on keyup).
* Press feedback is purely CSS (`:active` scale 0.97), so it follows the native active state:
  it starts on pointer-down on the button and holds until pointer-up (Chrome keeps `:active`
  even if the pointer drifts off while held); keyboard activation (Space held) also matches
  `:active`.

### 5.2 Menu

* **Opening (M).** Enter, Space or ArrowDown on the focused trigger opens with the **first** item
  highlighted; ArrowUp opens with the **last** item highlighted. A pointer press opens the menu
  and Base UI's focus manager moves focus to the **first tabbable element inside the popup**
  (`initialFocus: true`). Rows use a roving `tabindex="-1"`, so normally the popup itself
  (`role="menu"`, `tabindex="-1"`) takes focus and **no row is highlighted**. Measured on two
  menus, with real mouse events, including a reopen after a keyboard open. When the popup contains
  something natively tabbable, that gets focus instead. With a `MenuSearchInput`, the field takes it.
  If a row contains a native `<button>`, that row takes focus and is highlighted even on a mouse open
  (M: the script-actions menu, whose script rows carry an edit button, `ProjectScriptsControl.tsx:178–227`).
  While open the trigger has `aria-expanded="true"`, `data-popup-open` and `data-pressed`.
* **Focus (M).** Real DOM focus moves onto the highlighted row (`document.activeElement` is the
  `role=menuitem*` element); the popup has inside focus guards.
* **Navigation (M).** ArrowDown/ArrowUp move the highlight and **wrap** at both ends
  (`loopFocus: true`); Home/End jump to first/last; pointer hover highlights the row under it
  (`highlightItemOnHover`; callers can turn it off, as the script-actions menu does).
  **Disabled rows are not skipped.** Base UI menu items are `focusableWhenDisabled` and the menu's
  list navigation passes an empty `disabledIndices`, so a disabled row can take the highlight and
  focus from the keyboard but cannot be activated (S, bundle; no disabled row was live).
* **Typeahead (M + S).** Single printable keys (no Ctrl/Meta/Alt) highlight the next row, after
  the current one and wrapping, whose label starts with the typed string, case-insensitively. The
  buffer resets **500ms** after the last key: the Menu root passes `resetMs: 500`, while the hook
  default of 750 applies to Select. Measured live with key events whose timers were due 450, 560,
  620 and 700ms after the first key. At 450ms the buffer still held the first letter. At 560ms and
  later it had reset, which a 750ms timer could not do. When no row matches, the buffer clears and
  the highlight stays. Pressing the same letter again cycles through the rows starting with it
  (when no label starts with a doubled letter). A Space typed while the buffer is non-empty is
  appended to it, and its default action is prevented so it does not activate the row. Earlier
  measurement: "a" jumped to the row starting with "A"; a following "t" (buffer "at") matched
  nothing and left the highlight where it was.
* **Activation (S).** Enter, Space or click on a `MenuItem` runs it and closes the whole menu
  (`closeOnClick: true`); `MenuCheckboxItem` and `MenuRadioItem` toggle/select **without**
  closing (`closeOnClick: false`). Focus returns to the trigger after closing.
* **Escape (M).** Closes only the innermost open menu (`closeParentOnEsc: false`) and returns
  focus to its trigger (the sub-trigger for a submenu, the trigger button for the root menu).
  A second Escape with focus on a trigger is no longer handled by the menu, so it reaches the
  page (in T3 an Escape on a route view can navigate back, so keep this ordering).
* **Outside interaction (M + S).** The root menu is modal by default. Base UI renders a
  transparent, fixed, inset-0 `role="presentation"` backdrop (`user-select: none`) just before the
  positioner. Its `clip-path` polygon cuts a hole exactly over the trigger's rect (M: hole =
  trigger box), so pressing the trigger again still reaches it and toggles the menu closed. Any
  other outside pointer-down lands on the backdrop, closes the menu and does not reach the element
  underneath. While open, the rest of the page is marked `data-base-ui-inert` (M). Scroll is locked
  unless the menu was opened by hover. The focus manager itself is **not** modal for dropdown menus
  (`MenuPopup` passes `modal` only for context menus), so Tab moves focus out of the popup, and focus
  leaving it closes the menu (`closeOnFocusOut`).
* **Submenus (M + S).** Pointer rest/hover on a sub-trigger opens its submenu after **100ms**
  (`delay` 100, `restMs` 100). Leaving closes it immediately (`closeDelay` 0) unless the
  pointer travels inside the **safe area** toward the submenu (`safePolygon`, with pointer
  events on other rows blocked meanwhile). The safe area is a rectangle bridging the gap between
  row and submenu, plus a triangle from the cursor's exit point to the submenu's near edge. Inside
  the triangle the pointer must keep moving: 40ms without a move inside it closes the submenu, and
  so does moving slower than 0.1px/ms. Leaving the shape closes it at once (S, bundle). Per the same
  code, once the pointer has entered the submenu, a later exit that does not return over the
  sub-trigger row also closes it (not exercised live). ArrowRight on a sub-trigger opens the submenu
  and focuses its first row; ArrowLeft or Escape in the submenu closes it and focuses the
  sub-trigger (M). **Mouse presses on a sub-trigger do not toggle it.** The trigger opens on hover
  (`openOnHover: true`), so its click handler ignores mouse input (`ignoreMouse`) and keeps an open
  submenu open (`stickIfOpen`). Two mouse clicks on a hover-opened sub-trigger left it open (M).
  Touch and pen presses toggle it. Sub-triggers never close the menu on click. The submenu flips to
  the other side when it does not fit (M).
* **MenuSearchInput.** Typing goes into the field (its keydown stops propagation except for
  ArrowDown and Escape), so menu typeahead never steals letters; ArrowDown moves into the rows;
  Escape closes the menu. Filtering is done by the caller.
* **Placement.** `ClosingPositioner` freezes the popup at its last open position while the
  close runs, so a trigger that moves or unmounts doesn't drag the closing popup with it.
* **Width.** `width="default"` → at least min(160px, viewport − 32px); `width="content"` →
  shrink-wraps the widest row. Max width viewport − 32px; rows truncate with an ellipsis
  (`MenuItemLabel`).
* Popups opt out of window dragging (`-webkit-app-region: no-drag`) so they stay hoverable over
  the desktop title bar.

### 5.3 Popover

* Opens on trigger click (toggle); `openOnHover` is off by default (when a caller enables it:
  300ms delay, 0ms close delay). **Not modal**: the page stays interactive; an outside press
  or Escape closes it.
* Focus moves into the popup on open (M: the model picker focused its search field) and returns
  to the trigger on close by default (the model picker overrides this and returns focus to the
  composer, M).
* `touchBackdrop` adds a full-screen transparent layer on coarse pointers so the tap that
  closes the popover cannot also activate what is underneath.
* `keepMounted` keeps the subtree alive between openings (drafts survive) and uses a plain div
  instead of Base UI's Viewport (which would re-key the content).
* The Viewport cross-fades between `data-previous` and `data-current` children when one popover
  is re-targeted to another trigger/payload, while the popup animates its width/height.
* `variant="panel"` uses zero collision padding and animates `top` (150ms ease-out) instead of
  transform.

### 5.4 Tooltip

* **Hover open (M).** Opens after the pointer has **rested** on the trigger for **600ms**
  (measured 632ms from pointerenter on a loaded host). Movement restarts the rest timer;
  moves under ~1.4px (`dx² + dy² < 2`) are ignored. Mouse only. Touch does not open on hover.
* **Close.** The tooltip closes immediately on pointer leave (`closeDelay` 0), on click of the trigger
  (`closeOnClick`), on Escape (M), when the trigger is disabled, and on scroll inside a
  `TooltipScrollDismissArea` (unless the trigger has keyboard focus). Base UI also gives tooltips the
  submenu-style safe polygon (`handleClose: safePolygon()` unless `disableHoverablePopup`), so a
  pointer moving steadily from the trigger across the 4px gap keeps the tooltip open on the way.
  T3's positioner is `pointer-events: none`, so the popup never counts as hovered, and the tooltip
  closes once the pointer reaches it (S). A port that closes on leave is off only for that short trip.
* **Keyboard (M).** Focusing a trigger with the keyboard opens its tooltip immediately with no
  animation (`data-instant="focus"`); Escape closes it and focus stays on the trigger.
* The popup is not hoverable (the positioner has `pointer-events: none`).
* **No global provider.** Each trigger uses its own delay (default 600ms, or the caller's
  `delay` prop: T3 uses 0, 150, 200, 250 and 350ms in places). Inside a `TooltipProvider`,
  moving to another trigger within the provider's `timeout` (Base UI default 400ms) opens it
  instantly with `data-instant="delay"`. T3 has providers on the sidebar thread list (delay 150,
  close 0, timeout 400, `Sidebar.tsx:3202–3206, 3324–3328`, so thread-row tooltips open after
  150ms, not 600), the model picker (delay 0), the PR detail panel (delay 150, close 150,
  timeout 400) and theme settings (defaults).
* Placement defaults: side top, align centre, 4px offset, 5px collision padding plus the native
  title-bar height at the top; flips/shifts to stay visible.

### 5.5 Select

* Opens on click, Enter, Space, ArrowDown or ArrowUp on the trigger (`role="combobox"`). On open
  the **selected** item is highlighted and focused (M).
* ArrowDown/ArrowUp move the highlight **without wrapping** (M, re-checked: End then ArrowDown
  stays on the last row, Home then ArrowUp on the first); Home/End jump (M); typeahead matches item
  text (750ms buffer, the hook default); Enter/Space/click selects and closes; Escape closes (M).
  Focus returns to the trigger (the composer's select sends it to the composer instead, M).
* **Typeahead on the closed trigger.** With the trigger focused and the popup closed, typing
  works like a native `<select>`. A matching letter selects the next matching option directly and
  fires `onValueChange` without opening the popup. Base UI's `onMatch` sets the value when closed.
  This is enabled unless the select is disabled, `readOnly` or `multiple` (S, select bundle).
  Port it, because a focused Select trigger is not inert to letter keys.
* **Item-aligned placement (default).** When opened by mouse or keyboard and there is room,
  the popup is positioned so the selected item sits over the trigger (`data-side="none"`,
  items at least trigger width + 20px). Touch opens, lack of space, or
  `alignItemWithTrigger={false}` (many T3 call sites) use side bottom / align start / 4px.
* Modal (outside pointer blocked, scroll locked). After the first open the popup stays mounted:
  the positioner gets the `hidden` attribute and the popup gets `data-closed` (M, re-checked).
* Scroll arrows appear only when the list overflows in that direction; hovering them scrolls.

### 5.6 Combobox and Autocomplete

* Combobox: the input is the `role="combobox"` element; clicking it opens the list
  (`openOnInputClick`); typing filters (callers use `useComboboxFilter`); ArrowDown/ArrowUp move
  the highlight with wrapping (`loopFocus`), pointer hover highlights; Enter selects the
  highlighted item and fills the input (`fillInputOnItemPress`); Escape closes (M). Nothing is
  auto-highlighted while typing (`autoHighlight: false`), but on open the current selection is
  highlighted (M). Not modal. With `multiple`, the popup anchors to the chips container.
* `ComboboxEmpty` renders its text only when no item matches (padding only when non-empty);
  `ComboboxStatus` announces async state (`aria-live`).
* Autocomplete: free-text input with suggestions; same keyboard model; the trigger button is off
  by default; selected suggestion shows the accent-50% fill.

### 5.7 Switch, Toggle, ToggleGroup

* Switch: a focusable `span role="switch"` (tab stop) with button semantics from Base UI
  (`useButton`): click, Space or Enter toggles; clicking its `<label>` toggles; a hidden
  checkbox carries the form value. `mixed` is display-only. Disabled switches are skipped by
  pointer and show `not-allowed`.
* Toggle: native button with `aria-pressed`; click/Enter/Space toggle `data-pressed`.
* ToggleGroup: arrow keys move focus between items (horizontal by default, wrapping); single
  value unless `multiple`; pressing an item sets the group value.

### 5.8 Input, DraftInput, InputGroup, Textarea, NumberField

* Input: Base UI Input (value/`onValueChange`) or a native input with `nativeInput`. Autofill
  background is suppressed with a 5,000,000s background-colour transition (the autofill tint
  never visibly arrives); T3 shows its own autofill tint instead (§4.5).
* **DraftInput** (`useCommitOnBlur`): while focused the field edits a local draft (the prop value
  is ignored until blur); `normalizeDraft(previous, next)` may rewrite every change. On blur it
  commits `next` only if it differs from both the value at focus time and the current prop.
  **Enter** blurs (and therefore commits) unless an IME composition is active (`isComposing` or
  keyCode 229). There is no Escape-to-revert.
* InputGroup: pointer-down (primary button) on a non-interactive part of an addon is prevented
  and focuses the group's input/textarea if none is focused yet; buttons, links, inputs,
  selects and comboboxes inside addons behave normally.
* Textarea: grows with its content from its min height up to 256px, then scrolls.
* NumberField (Base UI): ArrowUp/ArrowDown ±`step` (1), Shift ±`largeStep` (10), Alt
  ±`smallStep` (0.1); PageUp/PageDown ±largeStep; Home/End jump to min/max when set; holding a
  step button repeats after 400ms, then every 60ms. For touch and pen the press first waits 50ms
  and is dropped if the pointer moved 3px or more, which guards against scrolling; the context
  menu is suppressed while a step button is held. Wheel scrubbing is off, and the value is parsed
  and re-formatted on blur/commit.

### 5.9 ScrollArea

* Scrollbars fade in while the pointer is anywhere over the scroll area or while scrolling
  (`data-scrolling` stays for 500ms after the last scroll event) and fade out 300ms later.
  Dragging the thumb scrolls; thumbs never get shorter than 16px.
* Edge fades track the remaining distance to each edge (the `--scroll-area-overflow-*`
  variables), so a list at the very top shows no top fade.
* The viewport is only a tab stop when the caller passes `viewportTabIndex`; then it shows the
  control ring and drops the fade while focused.

### 5.10 Dialog, AlertDialog, Sheet

* **Modal (M).** Focus is trapped; Tab cycles through the popup's tabbables and wraps (Base UI's
  focus guard after the last element receives focus and forwards it to the first, M). Escape
  closes (M). Pointer-down on the backdrop closes Dialog and Sheet
  (`disablePointerDismissal: false`); AlertDialog does not close on outside press (S).
  Page scroll is locked while open.
* **Initial focus.** The first enabled `DialogAction` button if present, otherwise Base UI's
  default (first tabbable element; M: a switch in the measured dialog); on touch opens the
  popup itself is focused so no on-screen keyboard appears.
* **Return focus (M).** To the element that had focus before opening (measured: the menu
  trigger whose item opened the dialog).
* `RetainedWhileClosed` keeps rendering the last open children during the exit transition, so
  the dialog doesn't go blank while fading out.
* Nested dialogs: the parent receives `--nested-dialogs: N` and recedes (§4.10).
* Sheet: same Base UI Dialog semantics; `keepMounted` keeps it alive while closed;
  `transitionDurationMs` overrides the 200ms (0 = instant).

### 5.11 Toast

* Managers are buffered: events raised before a provider subscribes are queued and replayed on
  mount (`toast.tsx:77–103`).
* Stacked toasts: newest first; at most 3 visible (older ones beyond the limit fade out);
  default timeout **5000ms** (Base UI), `timeout: 0` = sticky. Base UI pauses timers while the
  pointer is over the viewport, while a toast has focus, and while the window is unfocused.
* `data.dismissAfterVisibleMs`: an extra timer that only counts time while the document is
  visible **and** focused, and remembers the remaining time across remounts.
* Toasts carrying `data.threadRef` (or `threadId`) only render while that thread is the active
  route; others always render.
* Hovering or focusing the viewport expands the stack (`data-expanded`); the `::after` strip
  between toasts keeps the hover alive while moving between them.
* Swipe to dismiss: `right` + `up` for top-right (`left` + `up` for left positions; only `up`
  or `down` for centre positions). A drag starts after 1px of movement and follows the pointer
  with transitions off. Movement in a direction the toast cannot be swiped is damped to the square
  root of the distance. Releasing more than 40px along an allowed direction dismisses the toast
  along that direction. Pulling back 10px or more toward the start cancels the swipe, and the
  toast springs back on release. This applies when the opposite direction is not also allowed,
  which holds for every T3 position (S, toast bundle).
* Keyboard (S, toast bundle). F6 anywhere moves focus into the stacked viewport, remembers the
  previously focused element and pauses the timers. Each toast root is focusable
  (`tabindex="0"`), and Escape while focus is inside a toast closes that toast. Toasts beyond the
  limit are `inert`, and `priority: "high"` toasts use `role="alertdialog"`.
* The close orb calls `data.onClose` then closes. Error toasts with a string description get a
  copy button (tooltip "Copy error" → "Copied error"); long errors (≥180 chars) clamp to four
  lines. `expandableContent` adds a "Show details" disclosure; with
  `expandableDescriptionTrigger` the description itself is the toggle (Enter/Space toggle).
* Action layout: inline by default; `stackedThreadToast()` sets `actionLayout: "stacked-end"`
  (column layout, actions right-aligned on their own row). Actions are hidden unless
  `actionProps.children` is non-empty.
* Anchored toasts (`anchoredToastManager`): placed against `positionerProps.anchor`, 4px away,
  **above it by default** (Base UI `side: "top"`), flipping below when there is no room (M).
  "Copied!" uses `tooltipStyle` and a 1000ms timeout; "Failed to copy" is an `error` toast with
  the default 5000ms timeout and the regular body.

### 5.12 Collapsible, CollapsibleSectionHeader

* Collapsible: trigger click/Enter/Space toggles; the panel measures its content into
  `--collapsible-panel-height` and animates between 0 and that height (`animate` on by default;
  off inside `AnimatedHeight`, where closed panels are `display: none`).
* CollapsibleSectionHeader is a dumb button: the caller owns `expanded`; it never animates its
  own height. When an `accessory` (e.g. a count) is present the leader rule is hidden so the
  count does not read as a table-of-contents leader.

### 5.13 Spinner, Skeleton, Empty, PanelTabCloseButton, Separator, Kbd

* Spinner (`lib/visibleAnimation.ts`): an IntersectionObserver sets
  `--visible-animation-state` to `running` only while the icon is on screen, the document is
  visible and reduced motion is off; otherwise the rotation is paused (and `will-change` dropped).
  Under `prefers-reduced-motion: reduce` it doesn't animate at all (`motion-safe:`).
* Skeleton pulses only under `motion-safe`.
* PanelTabCloseButton: click runs `onClick` (close the tab); the X appears on tab hover or button
  keyboard focus, otherwise the tab's own status/icon shows in the same 12px slot.
* Separator and Kbd are static.

### 5.14 MiddleTruncate (algorithm)

`splitForMiddleTruncate(value, tail?)` (`middle-truncate.tsx:92–134`), on code points
(`Array.from`, so surrogate pairs are never split):

1. `keep = tail ?? 10`. If no explicit `tail`: find the last `/`; if it is neither first nor last,
   the segment after it is `segment = len − slash − 1`; `keep = segment` when `segment ≤ 16`,
   else 10.
2. If `keep ≤ 0` or `len ≤ keep + 4` → no split (render the whole value with CSS ellipsis).
3. `cut = len − keep`. If no explicit `tail` and the char before `cut` is not a word break
   (`-`, `_`, `.`, `/`): search **forward** from `cut + 1` up to `len − 4` for the first index
   whose previous char is a word break (a shorter tail that starts on a word); if none,
   search **backward** from `cut − 1` while `index ≥ len − 16` and `index > 4`, stopping at the
   **first** word start it meets, the nearest one before `cut`, which gives a longer tail. Use it if
   found (`snapped > 0`). Worked example from the tests: `t3/compare-long-featurenames-ab` has
   len 31 and `keep` 10, so `cut` is 21. The forward search over indices 22–27 finds nothing. The
   backward search stops at 16, after the second `-`, which gives the tail `featurenames-ab`.
4. Return `{head: chars[0:cut], tail: chars[cut:]}`.

Expected results (tests): `feature/ui-polish` → `feature/` + `ui-polish`;
`t3/clone-repo-to-other-machine` → tail `machine`; `t3/investigate-release-build-times` →
`times`; `t3/compare-long-featurenames-ab` → `featurenames-ab`;
`t3/abcdefghijklmnopqrstuvwxyz` → `qrstuvwxyz`; explicit tail 6 on the clone example →
`achine`.

Rendering (`fitHead`, 64–90; runs on every head change and on resize via a shared
ResizeObserver):

* Head is `min-width: 0; overflow: hidden`, tail is `flex-shrink: 0`, so layout gives the tail
  its full width and the head whatever is left.
* If the head overflows: binary-search the largest prefix length `fits` whose rendered width
  (Range rect from the head's left edge) is ≤ the head's client width; step back one if that
  would end on a high surrogate. `slack = clientWidth − width(prefix)`.
* Clip the partial glyph: `clip-path: inset(-0.25em <slack>px -0.25em 0)`; move the tail left by
  the slack (`translate: -<slack>px 0`) so it sits flush against the last whole character; set
  `data-overflow` on the head, which makes the tail render `…` as a `::before` prefix. Neither
  change affects layout, so the next measurement still sees the true available width.
* Not overflowing: clear clip, translate and `data-overflow`.
* `showTitle` (default) puts the full value in `title`. `variant="cap-aligned"` trims the text box
  to cap height and pads 4px below so descenders survive the clip.

---
## 6. Motion

Easings: **std** = Tailwind default `cubic-bezier(0.4, 0, 0.2, 1)` (also what `ease-in-out`
compiles to in this build, M); **ease** = CSS default `cubic-bezier(0.25, 0.1, 0.25, 1)` (used
when a raw `transition:` shorthand omits the timing function); **toast** =
`cubic-bezier(0.22, 1, 0.36, 1)`; `ease-out` = `cubic-bezier(0, 0, 0.2, 1)`. Default duration
150ms (`--default-transition-duration: .15s`). All values M (computed `transition-*`) unless
marked.

| Element | Trigger | Property: from → to | Duration | Easing | Delay |
| --- | --- | --- | --- | --- | --- |
| Button | `:active` (no `aria-haspopup`) | `scale` 1 → 0.97 (and back) | 150ms | std | 0 |
| Button | focus ring, shadow changes | `box-shadow` | 150ms | std | 0 |
| Button | hover / pressed colours | background, border, text | **instant** (not transitioned) | n/a | n/a |
| Badge | focus ring | `box-shadow` | 150ms | std | 0 |
| Toggle (all but segmented) | focus/pressed shadow | `box-shadow` | 150ms | std | 0 |
| Toggle `segmented` | hover/pressed | colours (bg, text, border) | 150ms | std | 0 |
| Input / InputGroup / Textarea / NumberField group | focus, invalid | `box-shadow` (ring). The border colour snaps | 150ms | std | 0 |
| NumberField step buttons | hover | colours | 150ms | std | 0 |
| Select trigger | focus, hover, pressed | `color, box-shadow, background-color` | 150ms | std | 0 |
| MenuSearchInput / ComboboxSearchInput rule | focus-within | border colour | 150ms | std | 0 |
| Menu popup, submenu, Select popup | open / close | **none** (appear and disappear in one frame) | n/a | n/a | n/a |
| Combobox, Autocomplete frame | open / close | `scale, opacity` declared but no start/end values → effectively none (S) | 150ms | std | n/a |
| Combobox trigger / clear button | hover | `opacity` .8 → 1 (`transition-opacity`) (S) | 150ms | std | 0 |
| Autocomplete clear button | hover, focus | `color, background-color, box-shadow, opacity` (.8 → 1) (S) | 150ms | std | 0 |
| Select trigger chevron | n/a | static; `opacity: .7` never changes (S) | n/a | n/a | n/a |
| Popover popup | open / close | `scale` 0.98 → 1, `opacity` 0 → 1 (reverse on close), from `--transform-origin` | 150ms | std | 0 |
| Popover popup | content resize | `width, height` | 150ms | std | 0 |
| Popover positioner | re-anchor / reposition | `transform` (off when `data-instant`) | 150ms | std | 0 |
| Popover viewport children | content swap | `opacity` cross-fade between previous and current | 150ms | std | 0 |
| Popover `panel` positioner | reposition | `top` (none under reduced motion) | 150ms | ease-out | 0 |
| Tooltip popup | open / close (hover) | `scale` 0.98 → 1, `opacity` 0 → 1, from the side facing the trigger | 150ms | std | 0 (open waits for the 600ms rest delay) |
| Tooltip popup | focus open, Escape/outside close, trigger change | instant (`data-instant` → duration 0) | 0 | n/a | n/a |
| Tooltip positioner | moving to a new anchor while open | `top, left, right, bottom, transform` | 150ms | std | 0 |
| Dialog / AlertDialog popup | open / close | `scale` 0.98 → 1, `opacity` 0 → 1 | 200ms | std | 0 |
| Dialog popup, nested child | open / close | also `translate` y 32px → 0 | 200ms | std | 0 |
| Dialog parent while a child opens | stack | `translate` y 0 → −20·N px, `scale` → 1 − 0.1·N, `opacity` → 1 − 0.1·N | 200ms | std | 0 |
| Dialog popup, mobile sheet mode (<640) | open / close | `translate` y 16px → 0 + opacity (S) | 200ms | std | 0 |
| Dialog / AlertDialog / Sheet backdrop | open / close | `opacity` 0 → 1 (`transition: all`) | 200ms | std | 0 |
| Sheet popup | open / close | `opacity` 0 → 1 and `translate` ±32px → 0 on its axis (right: from +32x; left: −32x; top: −32y; bottom: +32y) | 200ms (or `transitionDurationMs`; 0 = instant) | std | 0 |
| Toast (stacked) | enter | `transform` from `translateX(100% + inset) translateY(offset)` (top-right) / `translateY(-100% - inset)` (other top positions) / `translateY(100% + inset)` (bottom) | 500ms | toast | 0 |
| Toast (stacked) | exit (M, top-right) | `transform` → `translateX(100% + 32px)` (other positions: `translateY(100% + inset)`), `opacity` → 0; removed after the transition (measured ~0.63s on a loaded host) | 500ms (opacity 500ms) | transform: toast; opacity: ease | 0 |
| Toast (stacked) | stack/expand/collapse, reorder | `transform` (scale + translate formulas, §3.24) | 500ms | toast | 0 |
| Toast (stacked) | height change | `height` | 150ms | ease | 0 |
| Toast content | expand / collapse | `opacity` (hidden content of back toasts) | 250ms | std | 0 |
| Toast swipe exit | swipe | `transform` continues along the swipe by `100% + inset` | 500ms | toast | 0 |
| Anchored toast | open / close | `scale` 0.98 → 1, `opacity` 0 → 1 | 150ms | std | 0 |
| Toast close orb | hover/focus | `color, background-color, box-shadow` | 150ms | std | 0 |
| Toast "Show details" / description toggle | hover | colours | 150ms | std | 0 |
| Switch track | check / focus | `background-color, box-shadow` | 200ms | std | 0 |
| Switch thumb | check | `translate` 0 → travel, `transform-origin` left → right | 150ms | ease | 0 |
| Switch thumb | press (`:active`) | `scale` (x 1 → 1.1) and `border-radius` → elliptical | scale 100ms, radius 150ms | ease | scale 100ms |
| Menu checkbox `switch` indicator | as Switch | same values (`[transition:translate_.15s,border-radius_.15s,scale_.1s_.1s,transform-origin_.15s]`, track 200ms) | | | |
| Collapsible panel | open / close | `height` 0 ↔ `--collapsible-panel-height` (none under reduced motion or `animate={false}`) | 200ms | std | 0 |
| CollapsibleSectionHeader chevron | expand / collapse | `rotate` −90° ↔ 0° | 150ms | std | 0 |
| ScrollArea scrollbar | pointer enters / scroll starts | `opacity` 0 → 1 | 100ms | std | 0 |
| ScrollArea scrollbar | pointer leaves / 500ms after last scroll | `opacity` 1 → 0 | 150ms | std | **300ms** |
| ScrollArea thumb | hover | `background-color` | 150ms | std | 0 |
| ScrollArea viewport | focus ring | `box-shadow` | 150ms | std | 0 |
| Spinner | while visible | `rotate` 360° per turn (`spin`) | 1s, infinite | linear | n/a |
| Skeleton | always (motion-safe) | `opacity`: hold 1 for 0–42%, step down to .55 in 4 steps over 42–50%, hold to 92%, step back up in 4 steps to 100% | 2.4s, infinite | `steps(4)` | n/a |
| Input autofill | autofill | `background-color` 5,000,000s (suppresses the UA autofill flash) | n/a | ease-in-out | 0 |

Skeleton opacity sequence with `steps(4)` (jump-end): 1 → 0.8875 → 0.775 → 0.6625 → 0.55 over
0.192s starting at 1.008s; back up 0.55 → 0.6625 → 0.775 → 0.8875 → 1 over 2.208–2.4s.

**Theme switch.** While the app swaps light/dark (including when it follows a system scheme
change, `useTheme.ts:436–442`) it adds `.no-transitions` to `<html>` (`index.css:1105–1111`:
all transition and animation durations 0) and removes it on the next frame
(`useTheme.ts:327–345`), so colour changes never animate.

**Reduced motion.** Only Spinner, Skeleton (`motion-safe:`), Collapsible panels and the panel
popover's `top` transition (`motion-reduce:`) honour it. Popover, tooltip, dialog, sheet and
toast transitions run regardless.

---
## 7. Gaps

What could not be measured live, and why. Everything listed still has a source-derived (S) or
probe (P) value above.

* **Not reachable without changing data or settings.** Checked state of a live
  `MenuCheckboxItem` and selecting radio/select/combobox items (would change saved filters or
  composer settings); toggling a live Switch; AlertDialog (only opened by destructive flows such
  as delete/archive confirmations). Checked/selected/pressed values are P.
* **No live instance on the screens visited.** Autocomplete, NumberField, enabled Textarea (the
  only live one was disabled), Sheet (used for narrow/mobile layouts), media Dialog, Popover
  with default padding (the model picker uses `padding="none"`), Popover `panel` variant,
  Base UI ScrollArea scrollbars (the lists visited hide them or use native/virtualised
  scrolling), Empty, Skeleton, Spinner in motion, Select in item-aligned mode (the select
  opened fell back to below-trigger placement).
* **Timing.** The host ran at load ~23 on 3 cores during measurement; timers and frames in the
  page lagged up to ~2×. All durations and delays are therefore taken from source and the
  shipped Base UI code; live timings were sanity checks only (tooltip open 632ms vs 600ms
  configured; a submenu opened 528ms after pointerenter vs the configured 100ms, attributed to
  event-loop lag). Exit transitions were confirmed by attribute sequence, not frame timing. The
  verification pass got around the lag for the menu typeahead reset by comparing timer order
  rather than wall-clock gaps. The second key's timer was scheduled at the same instant as Base
  UI's reset timer, so whichever was due first ran first (§9).
* **Behaviours read from Base UI code but not exercised.** Menu press-drag-release selection,
  Tab closing an open menu, scroll lock under modal menus/selects/dialogs, Select scroll-arrow
  auto-scroll, NumberField auto-repeat and keyboard steps, toast swipe gestures and timer pausing
  on window blur, ToggleGroup roving focus, AlertDialog ignoring outside presses, the toast F6
  viewport hotkey and Escape-to-close (both now read from the bundle, §5.11), keyboard highlight
  of disabled menu rows (no disabled row was live), Select typeahead on a closed trigger (it would
  change a live setting), the safe polygon's 40ms stall and speed floor, and the tooltip's
  safe-polygon trip across the gap.
* **Fonts.** Measured with Inter (Linux); macOS renders SF Pro. All sizes/line heights are
  CSS-fixed; widths of auto-width controls (buttons, badges, menu min-content) will differ.
* **Breakpoints.** Below-640px (and below-768px for Empty) values are source-derived; the
  viewport was 1440px.
* **Themes and contrast.** Only the default palette at default appearance contrast was
  measured. Custom themes (`html[data-theme-id]`, `index.css:1325–1389`) replace most tokens and
  the contrast settings re-derive every `--contrast-*` colour (`index.css:1600–1712`); port the
  formulas, not just the hex values, if tern supports themes.
* **Desktop shell specifics.** Measured in Chrome against the web client. In Electron with
  window-controls overlay, `--native-titlebar-height` feeds the tooltip collision padding and
  media dialog top padding, and Windows adds a 6px right resize inset; not measured.
* **Coarse pointers.** The 44×44 touch hit areas and touch-only behaviours (popover touch
  backdrop, Select disabling item alignment on touch, dialog initial focus on the popup) were
  not exercised.
* **Translucent colour precision.** Canvas readback of low-alpha colours carries ±1–3 units of
  RGB rounding (for example `destructive/4` reads `#ff33330a`); use the formula (token + alpha)
  rather than the literal hex for alphas below ~10%.

---

## 8. Port notes for Go/Gio

1. **Tokens first.** Implement a theme struct with the §3.0 light and dark values, plus a
   sidebar-scope override set (dark only differs). Treat Tailwind `token/NN` as *the token with
   alpha × NN%* (the oklab mix with transparent preserves the colour); the glass background is
   popover at **83.6%** alpha. Inherited body text is `#262626` / `#f5f5f5`, not the foreground
   token.
2. **Desktop sizes.** Use the ≥640px column everywhere. Box sizes include the 1px border and
   horizontal padding is `4n − 1` px; keep both or text will sit 1px off T3's grid. If tern ever
   lays out narrower than 640 logical px, switch to the <640 column (larger controls and text).
3. **Glass without backdrop-filter.** Gio has no backdrop blur. Either render the content behind
   a popup into an offscreen op, blur it (12px light / 16px dark, Gaussian; saturate 1.14 / 1.08)
   and draw the 83.6% popover tint over it, or use T3's own no-support fallback: **solid
   `popover`** for dropdown-glass and dialog-glass (`#ffffff` / `#111111`) and solid
   `background` for surface-glass. The fallback is what T3 itself paints without
   backdrop-filter, so it is a faithful second choice.
4. **Shadows.** Implement layered box shadows (offset, blur, spread, colour) as cached
   blurred rounded-rect textures or nine-patches; popups need the asymmetric elevation
   (`0 16px 40px -18px` @55% light, `0 18px 44px -18px` @80% dark), toasts `shadow-xl` at 25%
   black, dialogs the dialog-glass shadow. Draw the **lip** (`::before`) as a 1px stroke over the
   control's border: along the bottom edge in light (black 4%), along the top edge in dark
   (white 6%), curving with the corners (it is the inner rounded rect, radius − 1, offset 1px
   down/up, visible only outside that rect).
5. **Focus.** Control ring = inside the border, 1px of the background colour then 2px of the ring
   colour, fading in over 150ms. Field ring = border switches to ring colour instantly plus a 3px
   inner ring at 24% alpha (animated 150ms). Only show on keyboard focus (`:focus-visible`
   semantics); text inputs show it on any focus.
6. **Icons.** Lucide paths on a 24-unit viewBox with stroke **1.75 units**, scaled with the
   icon: 16px → 1.167px stroke, 14px → 1.02px, 12px → 0.875px. Containers set the size (16px in
   most controls on desktop) unless the call site gives one; Button icons have −2px side margins.
7. **Text.** Enforce CSS line boxes (e.g. 14px font in a 20px line, 12px in 16px, 11px in 16.5px)
   and centre the shaped line in them; weights 400/500/600. Use the platform UI font on macOS
   (SF Pro) and Inter elsewhere, matching T3's stack.
8. **Overlay layer.** One overlay stack with T3's order: sheet 46 < dialog 50 < media dialog 60 <
   toast 100 < popups 130 < tooltip 140. Popups are positioned relative to their anchor with
   side/align/offsets from §3.30, flip to the opposite side and shift along the axis to stay 5px
   inside the window; expose the resolved side to drive transform origins.
9. **Menus.** No open/close animation. Keyboard: focus moves into rows; arrows wrap, Home/End,
   **500ms** typeahead (same letter again cycles), Enter/Space activate, Escape closes one level and
   restores focus, ArrowRight/ArrowLeft for submenus. Disabled rows stay in the keyboard order but
   do nothing. Submenus open after 100ms of hover and use a safe-triangle grace area that needs the
   pointer to keep moving (40ms stall, 0.1px/ms floor). Mouse clicks on a sub-trigger are ignored;
   only touch toggles it. Place submenus flush with the trigger row, 5px above its top, flipping
   sides when needed. Opening by keyboard highlights the first (ArrowUp: last) row. Opening by
   click focuses the first natively focusable thing in the popup (search field, or a row holding a
   button), otherwise the popup itself, with no row highlighted. Outside presses close the menu
   and are swallowed, except on the trigger, which toggles.
10. **Tooltips.** A 600ms *rest* timer that restarts on pointer movement over ~1.4px; immediate
    open on keyboard focus without animation; close on leave/click/Escape/scroll; 150ms
    fade + scale from 0.98 anchored on the side facing the trigger; 4px gap, top by default;
    pointer-transparent; max width 320 with balanced wrapping. Sidebar thread rows sit in a
    provider with a 150ms delay and a 400ms instant-reopen window; give tern a delay group with
    the same numbers.
11. **Fields.** Inputs have no hover state (the ghost InputGroup is the exception, with
    `muted/40`). A focused ghost InputGroup still draws the full field ring; only the popover
    variant drops it. Placeholder colours differ (Input muted vs Textarea 50% current colour). DraftInput: keep a local draft while focused, commit on blur or Enter
    (not during IME composition), only when changed.
12. **Select vs menus.** Select does not wrap with arrow keys and keeps the selected-fill on the
    highlighted row; menus wrap and the highlight replaces the checked fill. Select typeahead
    resets after 750ms against the menu's 500ms. On a focused, closed Select trigger, typing
    selects the matching option directly, like a native `<select>`. Port these differences rather
    than sharing one list widget configuration.
13. **Dialogs.** Centre in the window with 16px margin, max width 512, radius 16; trap focus with
    wrap-around; Escape and backdrop click close (not for alert dialogs); restore focus to the
    opener; animate scale 0.98 → 1 and opacity over 200ms, backdrop fade 200ms; keep rendering the
    last content during the exit.
14. **Toasts.** Top-right stack 32px from the right edge and 84px from the top, 360px wide.
    Implement the collapsed/expanded transforms exactly as §3.24 (scale `1 − 0.1i`, peek 12px,
    gap 12px, back toasts forced to the front toast's height with hidden content), 500ms toast
    easing, at most 3, 5s default timeout paused while hovered/focused/unfocused window, and the
    24px close orb set −6px from the padding box, so it overhangs the outer border edge by 5px.
    F6 focuses the stack and Escape closes the focused toast. Anchored toasts (for example "Copied!")
    sit 4px **above** their anchor and flip below only when there is no room.
15. **Switch.** Track 30×18, thumb 14 (desktop), 12px travel over 150ms with the press stretch
    (x 1.1 after 100ms); dark checked thumb is black (background), dark unchecked thumb is the
    muted-foreground grey.
16. **ScrollArea.** Overlay scrollbars 6px wide inset 4/1px, visible only while hovered or within
    500ms of scrolling (fade in 100ms, out 150ms after 300ms), min thumb 16px; edge fades of up
    to 24px that grow with scroll distance. Native-style 6px scrollbars elsewhere use the same
    thumb colours with radius 3.
17. **MiddleTruncate.** Port `splitForMiddleTruncate` exactly (code points, 10/16/4 constants,
    word-break chars `- _ . /`) and lay out head + tail with the tail never shrinking; when the
    head overflows, cut it after its last whole glyph and draw "…" flush before the tail.
18. **Motion helper.** Most transitions are 150ms on `cubic-bezier(0.4, 0, 0.2, 1)`; overlays use
    200ms; toasts 500ms on `cubic-bezier(0.22, 1, 0.36, 1)`. Disable all transitions for the frame
    in which the theme flips.

---

## 9. Verification

An adversarial second pass on 2026-11-17 against the same build
(`v0.1.10-nightly.20261112.4606`), the same browser session and viewport (1440×900 at 2x), and both
schemes (`set media light|dark`, left on light). It re-measured values on live elements and
probes, re-derived values from the TSX, `index.css` and the shipped Base UI code, and hunted for
missing states and behaviours. Fixes are already folded into §1–§8. This section records what was
checked and what changed. Raw measurements and scripts are kept outside the repo.

### 9.1 Re-measured live (both schemes unless noted)

About 1,000 comparisons in total: about 600 on real elements and about 450 on probes. Probes were
built with the shipped `cn`/`buttonVariants` or the exact source class strings.

| Area | Specimen | Values compared | Result |
| --- | --- | --- | --- |
| Colour tokens | `var(--x)` probes | 29 tokens + inherited body text, ×2; glass blur/opacity/saturation; radius, `--z-sheet`, top-bar height, default duration and easing, `--ease-out`, `--ease-in-out` | all match §3.0 |
| Tooltip | sidebar "New thread" trigger | surface, border, radius 6 / lip 5, lip and drop shadows, 12/16.5 400, padding 4/8, max width 320, wrap + balance, z 140, `pointer-events`, popup and positioner transitions, 4px gap, 320×61.5 box | all match §3.7 |
| Tooltip, keyboard | Shift+Tab onto a header button | opens at once, `transition-duration: 0s`, `data-instant` set | matches §5.4 |
| Menu | right-panel script-actions menu | glass bg/border/blur, elevation shadow, radius 8, min 160 / max 1408, z 130, 4px gap, no transition, max height = viewport − top − 5, padding 4, rows 28 with 4/8 padding and gap 8, radius 4, 14/20 400, highlight fill, 16px icon at .8 with −2px margins and stroke 1.75 | all match §3.5 |
| Menu, radio rows + label | run-context menu | label 28 tall, 6/8 padding, 12/16 500 muted; rows 28, 4/8, radius 4, no gap; checked `#1a1a1a14` / `#f2f2f214`; checked + highlighted shows accent | all match |
| Sub-trigger + submenu | PR-list filter menu | open fill accent, chevron 16 / .8 / −2px end / muted; submenu glass, `data-nested`, flush (`sideOffset` 0), −5 align offset, min 160, flipped to inline-start near the edge | all match |
| Popover | composer model picker | glass, radius 8 (lip 7), elevation shadow, 150ms `width,height,scale,opacity`, z 130, `padding="none"` viewport, 4px gap, focus to its search field, focus back to the composer on close | all match |
| Search field in a popup | model picker | wrapper 10/12/0, 1px rule → ring colour on focus with a 150ms colour transition, icon 16 at muted 55%, input 26/26 with 20/9 padding; selected + highlighted row at foreground 9% | all match §3.13 |
| Select | composer workspace select | below / start / 4px, z 130, no transition, glass frame min width = trigger, list padding 4, max height = available height; fills for highlighted, selected, and selected + highlighted (selected wins); no arrow wrap (End/Down, Home/Up); mounted-but-hidden after close | match. The `hidden` attribute sits on the positioner (fixed in §5.5) |
| Toast, stacked | probe toast with placeholder text | viewport top 84 / right 32 / width 360 / z 100; 68px root; glass, `shadow-xl` black 25%, transition string, z 9999; 13px `::after`; content 12/14, gap 6, 250ms opacity; title and description type; 16×20 icon; orb 24px with glass, blur 8, `shadow-sm`, 12px X at 2.25; orb hover | all match. Orb overhang clarified to 5px |
| Toast, anchored | probe anchor mid-screen, then at the top edge | side, gap 4, z 100, max width 256, glass, radius 6, 12/16, padding 4/8, 150ms `scale,opacity`; flip | **side was wrong** (fixed) |
| Toggle | header panel toggles | 28×28, radius 8, pressed and hover fill accent, 16px icon at .8, `box-shadow` transition | values match; variant attribution was wrong (fixed) |
| Outline Button | PR-list "Filters" and "Refresh" | 32 tall, 11 padding, radius 8, popover / `input/32` fill, `input` border, `0 1px 2px #0000000d`, lip in both schemes, muted icon colour, `will-change: scale` only without `aria-haspopup` | all match |
| Control focus ring | keyboard focus on the refresh button | `inset 0 0 0 1px` background band + `inset 0 0 0 3px` ring; drop shadow and lip kept | matches §3.0 |
| InputGroup | PR-list search field | colours, lip, addon padding 11, icon 16 at .8, input padding 8/11, `display: contents` wrapper | colours match; the live field is 34 tall because of a caller override (fixed in §3.9) |
| CollapsibleSectionHeader | sidebar section headers | 32 tall, 8/8, gap 8, radius 6, 12/16 500, tone colours, rule colour + hidden with a count, 12px chevron at −90° with 150ms rotate, 12px tabular count | all match §3.25 |
| Probes | buttons (`default`, `destructive-outline`, `outline` sm/compact/micro, `ghost` icon-xs, `secondary` xs, `glass` sm), Kbd default + outline, ghost MenuItem, Toggle outline/segmented rest + pressed, ghost InputGroup focused, Switch default/sm × off/on, Badge default/sm, Textarea | heights, paddings, gaps, radii, type, fills, borders, shadows, lips, icon sizes and colours, thumb travel and origin | match, except the ghost MenuItem padding and the ghost InputGroup focus ring (fixed) |

Behaviour exercised live: mouse opens of three menus with focus and highlight checks, including a
reopen after a keyboard open. Also: ArrowDown open, Escape focus return (four times), the modal
backdrop and its trigger hole, the `data-base-ui-inert` marking, hover-open plus two mouse presses
on a sub-trigger, menu typeahead timing (ten valid trials, five of them timer-order controlled),
Select arrow keys, and the anchored toast flip.

### 9.2 Re-derived from source and the shipped bundle

More than 300 values. Every size and variant row in §3.1 against `button.tsx:16–70`. The menu
item, checkbox, radio, sub-trigger, search-input, label, shortcut and detail classes against
`menu.tsx`. The popover width/padding maps (`popover.tsx:19–30, 95–120`). Tooltip collision
padding and variants (`tooltip.tsx:16, 109–155`). Input, Textarea and NumberField sizes
(`input.tsx:22–25`, `textarea.tsx:35–47`, `number-field.tsx:86`). Select trigger sizes
(`select.tsx:26–33`). Combobox paddings and fills (`combobox.tsx:52–98, 202–203`). Switch
geometry (`switch.tsx:14–31`). Toggle sizes and variants (`toggle.tsx`). The toast stack
transforms (`toast.tsx:614–650`) and layout (`toast.logic.ts:40–90`). ScrollArea bar timings
(`scroll-area.tsx:87`). The dialog, alert-dialog and sheet classes. Empty sizes (`empty.tsx:5–10`).
The skeleton keyframes (`index.css:295–313`). The glass utilities (`index.css:364–467`).
`useCommitOnBlur`, `visibleAnimation` and the theme-switch suppression. MiddleTruncate was traced
by hand against all five test cases.

In the bundle: menu typeahead `resetMs:500` against the hook default 750, and the typeahead
rules. Submenu trigger `openOnHover`/`delay`/`closeDelay` and its `ignoreMouse`/`stickIfOpen`
click handling. The safePolygon constants. Tooltip 600ms rest, the movement threshold and the
safe-polygon close. TooltipProvider `timeout:400`. Menu `modal` default and the backdrop
cut-out. The focus manager's `modal` only for context menus. Menu items `focusableWhenDisabled`.
Select closed-trigger typeahead. Combobox, ToggleGroup and NumberField defaults, including the
400/60ms repeat and the touch guard. AlertDialog role and forced flags. Toast positioner
defaults, roles, F6, Escape and swipe constants.

### 9.3 Errors found and fixed

1. **Menu typeahead reset** was 750ms; it is **500ms**. The Menu root passes `resetMs:500`, and
   750 is only the hook default, which Select uses. Confirmed live (§5.2, §1.3, §8.9).
2. **Sub-trigger mouse press** was said to toggle the submenu. With a mouse it does nothing:
   `ignoreMouse` and `stickIfOpen` apply because of `openOnHover`, and two clicks left it open.
   Only touch and pen toggle (§5.2, §8.9).
3. **Disabled menu rows** were said to be skipped by arrow keys. Base UI keeps them focusable
   (`focusableWhenDisabled`, empty `disabledIndices`), so the keyboard can highlight them, but they
   cannot be activated (§5.2, §4.3, §8.9).
4. **Anchored toasts** were placed below the anchor (`side` bottom). Base UI's default is `top`,
   and they flip below only without room. Confirmed live (§3.24, §3.30, §5.11, §8.14).
5. **Ghost MenuItem padding** was `0 7px`. It is `4px 7px`, at 12/16 with gap 4 (§3.5).
6. **Header panel toggles** were described as `sm` toggles with a caller override of the pressed
   fill. They are `variant="ghost" size="sm"`, and the accent fill comes from the variant (§3.16).
7. **InputGroup height "M on the PR-list search field"**. That field overrides its input height
   and measures 34, not 32. The 32px stock value comes from the probe (§3.9).
8. **"Click opens with no item highlighted"** was too strong. The focus manager focuses the first
   tabbable element in the popup. That is usually the popup itself, but it is the search field when
   there is one, or a row that holds a native button, which then shows as highlighted (§5.2, §8.9).
9. **MiddleTruncate backward search** "taking the last word start found" misread the loop. It
   stops at the first, nearest word start before `cut`. A worked example was added (§5.14).
10. **Select after close**: the `hidden` attribute is on the positioner, and the popup gets
    `data-closed` (§5.5).
11. **Toast close-orb overhang** was given as 6px. The −6px offsets are measured from the padding
    box, so the overhang past the outer border edge is 5px (§3.24, §8.14).
12. **Textarea `resize: vertical`** was attributed to the browser default. It comes from
    Tailwind preflight (§3.10).

### 9.4 Omissions filled

Select typeahead on a closed trigger (§5.5, §8.12). Typeahead rules: modifier keys ignored,
same-letter cycling, Space inside a buffer, clearing on no match (§5.2). Safe-area geometry and
its 40ms / 0.1px/ms rules (§5.2). The tooltip safe polygon against T3's pointer-transparent
positioner (§5.4). The sidebar thread-list tooltip provider (150 / 0 / 400) and the provider
default timeout (§5.4, §8.10). The modal menu backdrop with its trigger hole, inert marking,
non-modal focus manager and hover-open scroll exception (§2.4, §5.2). The AlertDialog role and
forced flags (§2.20). Toast roles, `tabindex`, `inert`, F6, Escape and swipe thresholds (§2.22,
§5.11). Copy-error button hover and copied states (§3.24). InputGroup addon `inset="pill"` and the
3px kbd radius (§2.8). The focus ring on a ghost InputGroup (§3.9, §4.5, §8.11). Popover calendar
radius and padding (§2.5). Combobox and Autocomplete button transitions (§6). Autocomplete max
height and its selected + highlighted fill (§3.14). The dark `:active` lip on an outline Toggle
(§4.8). The NumberField touch/pen guard (§5.8). Select scroll-arrow gradient geometry (§3.12).
Radio indicator colour and margins (§3.5). Source rows for the caller files cited (§1.2).

### 9.5 Privacy

The doc was scanned for project, thread, branch, host and account names seen in the live app, for
local paths and for the server address. None were found, and none were added. The MiddleTruncate
strings come from T3's public test file. Probe toasts used placeholder text and were closed.

### 9.6 Still unverified after this pass

Read from code but not exercised live: keyboard highlight of a disabled menu row (none was live),
Select typeahead on a closed trigger (it would change a setting), toast F6, Escape and swipe, the
safe polygon's 40ms stall and speed floor, and submenu exit after entering. Dialog, AlertDialog and
Sheet were not reopened in this pass; their glass values were re-derived from `index.css` only.
MiddleTruncate was not re-measured, because no truncated label was on screen. The §7 gaps on
fonts, sub-640px layouts, themes and contrast, the Electron overlay and coarse pointers are
unchanged.

One measurement hazard turned up. Near the end the browser tab went `visibilityState: hidden` on
its own. That paused animation frames, so a menu's exit sat at `data-ending-style` until the tab
was brought to the front, and then it finished normally. Base UI completes exits on an animation
frame even when no CSS transition runs. Values taken earlier came from a visible tab, because the
theme helper confirms visibility on every switch. Check `document.visibilityState` before trusting
any open or close sequence recorded on this host.
