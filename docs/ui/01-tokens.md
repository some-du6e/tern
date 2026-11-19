# 01 · Design tokens

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`, shipped stylesheet `client/assets/main-DcchvbMF.css`.
Machine-readable companions: [`tokens.json`](tokens.json) (the required shape: `tag`, `css`, `fonts`, `themes.light/dark`, `scale`; every custom property resolved in both themes) and [`tokens.extra.json`](tokens.extra.json) (`scopes`: the sidebar and composer sub-palettes; `variants`: chat width, blue-orange diffs, contrast 50–200, glass, Electron-only; `builtInThemes`: 5 themes × 2 modes × 61 roles).

Conventions used throughout:

- **Lengths** are CSS px at the default root size of 16px (`html { font-size: 16px }`, set inline from `settings.fontSizeInterface`). Every `rem` value scales with that setting (12–20px). One CSS px = one Gio dp.
- **Colours** are sRGB hex as Chrome paints them: `#rrggbb`, or `#rrggbbaa` when translucent. They were resolved in the live page: the computed colour goes through `color(from <c> srgb r g b / 1)` onto a 1×1 canvas for the RGB, and through `color(from <c> srgb r g b / alpha)` for the alpha. A translucent colour's RGB is therefore exact rather than premultiplied. (`app/tools/t3ui/inspect.sh` paints translucent colours directly, so its RGB for those can be a few units off.) Out-of-gamut colours are clipped to sRGB. This gives exactly Tailwind v4's published sRGB fallbacks, e.g. red-600 `#e7000b`, emerald-500 `#00bc7d`.
- **M/S column**: **M** = measured live (read with `getComputedStyle` on `<html>`, a scope element, or a probe element carrying the real utility class). **P** = the declared expression evaluated live on a probe element (used for Tailwind `@theme inline` tokens, which are inlined into utilities rather than emitted, and for setting variants that were not switched on). **S** = source-derived only.
- "Light" is the page with no `dark` class. "Dark" is `html.dark`, reached by emulating `prefers-color-scheme: dark` (the default appearance mode is *system*).

## 1. Scope and sources

T3 source, paths relative to the T3 repo:

| File | Lines | What it gives |
|---|---|---|
| `apps/web/src/index.css` | 1–2598 (all) | every token, utility, keyframe and base rule |
| `apps/web/index.html` | 9, 13–388, 389–670 | `theme-color` meta, boot theme script (sets `dark`, `data-theme-id`, html background before CSS loads), boot `<style>` (unlayered body colours) |
| `apps/web/src/main.tsx` | 4–5 | `@fontsource-variable/inter` then `index.css` |
| `apps/web/package.json` | 17, 32, 75 | `@base-ui/react ^1.9.0`, `@fontsource-variable/inter ^5.3.0`, `lucide-react ^0.564.0` |
| `apps/web/src/hooks/useTheme.ts` | 41–50, 244–348, 394–396, 436–487, 518–581 | theme storage keys and defaults, `applyTheme` (toggles `dark`, `no-transitions`), browser-chrome colour sync, system-change listener |
| `apps/web/src/themePalette.ts` | 44–47, 397–415, 680–739, 741–787, 1544–1606, 1612–1653, 1656–1680 | standard painted palette, context-chip solver, `--app-theme-*` role variables, `applyThemePalette`, appearance resolution |
| `apps/web/src/appInit.tsx` | 68–126, 220–224, 262 | `startClientAppearance`: `data-diff-color-scheme`, `data-chat-width`, contrast, glass, font variables |
| `apps/web/src/appearanceContrast.ts` | 1–10 | `--appearance-contrast-*` |
| `apps/web/src/appearanceFonts.ts` | 13–20, 72–137, 139–152 | default stacks, `--font-*`, root font size, `--list-marker-sans`, smoothing |
| `apps/web/src/lib/contextChipAccents.ts` | 1–31 | chip accent seeds |
| `apps/web/src/components/ContextChip.tsx` | 8–35 | how chips consume `--context-chip-*` |
| `apps/web/src/lib/windowControlsOverlay.ts` | 7–11, 50–62, 70–74 | `.wco`, `.electron`, `.electron-windows` |
| `apps/web/src/lib/visibleAnimation.ts` | 10–62 | `--visible-animation-state` gating |
| `apps/web/src/panelAnimations.ts` | 31–39 | panel-animation activation |
| `apps/web/src/components/AppSidebarLayout.tsx` | 69, 446–461 | `--sidebar-width`, `--panel-animation-duration`, macOS controls inset |
| `apps/web/src/components/ui/sidebar.tsx` | 30–32, 243–246, 301 | sidebar widths |
| `apps/web/src/components/chat/ComposerSurface.tsx` | 16–62 | `--chat-composer-*`, themed overrides, which element paints the glass |
| `apps/web/src/components/threadSidebarWidth.ts` | 1–34 | thread sidebar default/min/max width (px) |
| `apps/web/src/components/SidebarStageBackdrop.tsx` | 56 | the only `sidebar-stage-backdrop` use (inside the sidebar) |
| `apps/web/src/components/ThreadTerminalDrawer.tsx` | 600–612 | terminal font family/size resolution |
| `apps/web/src/components/settings/settingsLayout.tsx` | 146–153 | settings search pulse skipped under reduced motion |
| `apps/web/src/components/preview/PreviewChromeRow.tsx`, `components/usage/UsageLimits.tsx` | 299; 95 | inline `var(--color-ring)`, `var(--color-muted)` |
| `apps/web/src/components/diffs/DiffCommentAnnotation.tsx`, `settings/ThemeImportDialog.tsx`, `files/office/WordDocument.tsx` | 83, 126; 121; 156 | the only `selection:` colour utilities |
| `apps/web/src/components/ConnectionStatusDot.tsx`, `preview/AgentBrowserCursor.tsx` | 40; 88 | `animate-status-ping` forms |
| `apps/web/src/components/ui/scroll-area.tsx` | 12–18 | `--virtualized-scroll-fade-mask` |
| `apps/web/src/components/settings/AppearanceSettingsPanel.tsx` | 76–86, 147–480 | appearance options |
| `apps/web/src/components/settings/TypographySettings.tsx` | 47–120, 218–260, 280–358 | typography options |
| `apps/web/src/components/settings/ThemeSettings.tsx` | 57–63, 680–760 | theme library (standard + maintainer themes) |
| `apps/web/src/components/settings/ThemePreviewCircles.tsx` | 58–67 | "T3 Code" standard card |
| `apps/web/src/components/settings/SettingsPanels.tsx` | 1452–1490 | diff colour option |
| `packages/contracts/src/settings.ts` | 137–197, 337–340, 400–500, 594 | ranges and defaults of every appearance setting |
| `packages/shared/src/themePalettes.ts` | 1–234, 236–925 | role list, shared standard palette, the five built-in themes |
| `apps/desktop/src/window/DesktopWindow.ts` | 46, 56–58, 75–88, 164–166, 318–359, 476–489 | window background, title bar overlay, traffic lights |
| `apps/desktop/src/settings/DesktopAppSettings.ts` | 53–67 | window size limits |
| `apps/desktop/src/electron/ElectronTheme.ts` | 38–47 | `nativeTheme.themeSource` |

Shipped build: `client/assets/main-DcchvbMF.css` (538,735 bytes) contains 7 `@font-face` rules, the `@layer theme` block (149 declarations), `base`, `components`, `utilities`, and the unlayered `index.css` rules. No other stylesheet is involved in the default UI. `MathTypeset-*.css` (KaTeX) and `PdfViewer-*.css` load lazily for math and PDF previews.

Live page: the same build at 1440×900 @2x in Linux Chrome, on a draft-thread view, with default appearance settings: no theme selected, chat width `comfortable`, diffs `red-green`, contrast 100, glass 80, interface 16px, smoothing on. Screenshots stay in the private reference folder, not this repo. Method note: a background tab does not dispatch `prefers-color-scheme` change events (no rendering steps run), so after `set media dark|light` the tab must be brought to the front before the app toggles `dark`.

## 2. Structure

### 2.1 Where tokens live in the cascade

- Layer order is declared first: `@layer theme, base, clerk, components, utilities;` (index.css:2). Tailwind's theme variables are in `@layer theme`. Tailwind's preflight and the app's base rules (`* { border-color; outline-color }`, outline offset, html/body background, lucide stroke, drag-region reset) are in `@layer base` (index.css:668–738, 1728–1732, 1836–1840). The maintainer stage palettes and the settings slider are in `@layer components` (740–1077, 2550–2588).
- **The semantic token blocks are unlayered** (`:root` at index.css:91–152, 1113–1229, 1282–1290, 1600–1712, 1768–1771, 1808–1811, 2284–2289; `[data-app-sidebar]` at 1234–1278). Unlayered rules beat every layer, so on the same element (`html`, the sidebar container) a utility cannot override them. Only inline styles or later unlayered rules can. Descendants can still re-declare a variable on themselves, which is how the composer's `[--chat-composer-*:…]` classes work.
- **The boot `<style>` in `index.html` is also unlayered** (index.html:389–670). It sets `body { color: #262626; background: #ffffff }` and `html.dark body { color: #f5f5f5; background: #000000 }` (index.html:398–408). That beats the app's `@layer base` rule `body { color: var(--contrast-foreground) }` (index.css:735–737). **Measured: inherited body text is `#262626` in light** (not `--foreground` `#18181b`) and `#f5f5f5` in dark. Body background is then overwritten inline by the chrome-colour sync (§5.4).
- Lightning CSS rewrote some values in the shipped file:
  - Every `color-mix()` declaration is emitted twice: a fallback first, then the `color-mix()` value under `@supports (color: color-mix(in lab, red, red))`. Chrome uses the second.
  - `light-dark(A, B)` became `var(--lightningcss-light, A) var(--lightningcss-dark, B)`, with `--lightningcss-light: initial; --lightningcss-dark: ;` on `:root`, swapped under `.dark`.
  - Tailwind's `--alpha(var(--color-white) / 3%)` became `color-mix(in oklab, var(--color-white) 3%, transparent)`.
  - `rgb(217 217 217)` became `#d9d9d9`.
- Custom variants: `dark` = `&:is(.dark, .dark *)`, `light` = `&:not(.dark, .dark *)`, `wco` = `&:is(.wco, .wco *)`, `any-hover` = `@media (any-hover: hover) { &:hover }` (index.css:5–17).

### 2.2 Token scopes in the element tree (measured DOM, draft view)

```
html                      class: "" | "dark" (+ "no-transitions" for one frame on theme change; "electron", "electron-windows", "wco" in the desktop app)
│                         data-chat-width="comfortable" data-diff-color-scheme="red-green" [data-theme-id, data-theme-selected when a theme is picked]
│                         style: background-color; font-size:16px; -webkit-font-smoothing:antialiased; --context-chip-* (24);
│                                --appearance-contrast-base/boost/border-boost; --glass-opacity; --list-marker-sans;
│                                --font-size-prompt; --font-size-code; --diffs-font-size  [+ --font-sans/--font-mono/--font-composer, --glass-blur:0px, --app-theme-* when set]
└─ body                   style: background-color (sampled surface). CSS adds the grain tile in light only (§3.13)
   └─ #root
      └─ div[data-slot=sidebar-wrapper][data-panel-animations="false"]
         │   style: --sidebar-width:256px; --sidebar-width-icon:3rem; --workspace-titlebar-content-left; --panel-animation-duration:0ms
         ├─ div[data-slot=sidebar-container][data-app-sidebar]     ← re-declares the palette and recomputes --contrast-* (§3.5)
         │   classes: fixed inset-y-0 z-10 hidden h-svh w-(--sidebar-width) md:flex … group-data-[side=left]:border-r
         │   └─ div[data-slot=sidebar-inner]   classes: flex h-full w-full flex-col bg-sidebar surface-grain …
         └─ main[data-slot=sidebar-inset]      classes: relative flex min-w-0 w-full flex-1 flex-col bg-background surface-grain … h-svh md:h-dvh
            ├─ header[data-chat-header]        classes: relative bg-background … h-[var(--workspace-topbar-height)] pl-(--workspace-gutter-start) pr-(--workspace-gutter-end)
            └─ … div[data-slot=composer-shell] ← declares --chat-composer-* (§3.15)
                classes: … max-w-(--chat-content-max-width) [--chat-composer-drawer-inset:1.375rem] [--chat-composer-glass-surface:var(--card)]
                         [--chat-composer-outline:rgb(0_0_0/8%)] dark:[--chat-composer-glass-surface:var(--surface-raised)]
                         dark:[--chat-composer-highlight:rgb(255_255_255/3%)] dark:[--chat-composer-outline:color-mix(in_srgb,var(--color-white)_5%,transparent)]
                         before:bg-(--chat-composer-glass-surface)/(--glass-opacity) before:backdrop-blur-(--glass-blur) before:backdrop-saturate-(--glass-saturation) …
```

### 2.3 Token families

| Family | Examples | Defined | Purpose |
|---|---|---|---|
| Primitives | Tailwind palette `--color-zinc-200`, `--color-white`, plus `--color-zinc-25` = `oklch(99.2% 0 0)` | `@layer theme` (Tailwind defaults), index.css:222 | raw colours. Only the 80 the app uses are emitted (§3.20) |
| Semantic roles | `--background`, `--foreground`, `--card`, `--primary`, `--border`, `--sidebar-row-hover` … | index.css:1113–1229 | what components mean |
| Contrast-adjusted | `--contrast-foreground`, `--contrast-border` … | index.css:1600–1712 | the role after the Contrast setting. **Utilities read these**: `text-foreground` = `var(--contrast-foreground)`, `border-border` = `var(--contrast-border)` |
| Utility aliases | `--color-foreground: var(--contrast-foreground)`, `--color-background: var(--background)` … | `@theme inline` index.css:177–362 | give Tailwind class names. Inlined into each utility, so `var()` resolves on the element using the class (that is how the sidebar scope works) |
| Geometry | `--radius`, `--control-radius`, `--workspace-*`, `--chat-content-max-width` | index.css:91–152, 1115 | layout lengths |
| Type | `--text-*` (+ `--line-height`), `--font-sans`, `--font-mono` | `@theme` 170–175, `@theme inline` 183–190, Tailwind | type scale |
| Effects | `surface-glass`, `dropdown-glass`, `dialog-glass`, `dialog-backdrop`, `alert-glass`, `topbar-scroll-fade`, `virtualized-scroll-fade`, `sidebar-stage-backdrop`, `live-tool-shine`, `live-activity-focus*`, `surface-grain`, `p*-safe` | `@utility` 364–660, 1082–1102 | composite surfaces |
| Motion | `--ease-drawer`, `--animate-*`, `@keyframes` | 191–232, 295–361, 551–602, 923–931, 2388–2436 | animations |
| Runtime | `--context-chip-*`, `--appearance-contrast-*`, `--glass-opacity`, `--font-size-*`, `--list-marker-sans` | inline on `<html>` | settings and solved colours |
| Scoped | `[data-app-sidebar]` palette; `--chat-composer-*` | index.css:1234–1278; ComposerSurface.tsx:16–20 | sub-palettes |
| Theme roles | `--app-theme-canvas` … (61 roles) | inline on `<html>` when a theme is active; mapped by index.css:1325–1389, 1563–1589 | built-in and custom themes (§5.2) |

## 3. Measurements

### 3.1 Surfaces, text and accents (`:root`)

| Token | Light | Dark | Declared (light) | Declared (dark, if different) | index.css | M/S |
|---|---|---|---|---|---|---|
| `--background` | `#fcfcfc` | `#000000` | `var(--color-zinc-25)` | `#000` | 1116 / 1192 | M |
| `--app-chrome-background` | `#fcfcfc` | `#000000` | `var(--background)` | = | 1117 | M |
| `--foreground` | `#18181b` | `#f5f5f5` | `var(--color-zinc-900)` | `var(--color-neutral-100)` | 1125 / 1194 | M |
| `--card` | `#ffffff` | `#111111` | `var(--color-white)` | `color-mix(in srgb, var(--color-neutral-950) 97%, var(--color-white))` | 1126 / 1195 | M |
| `--card-foreground` | `#18181b` | `#f5f5f5` | `var(--color-zinc-900)` | `var(--color-neutral-100)` | 1127 / 1196 | M |
| `--popover` | `#ffffff` | `#111111` | `var(--color-white)` | `color-mix(in srgb, var(--color-neutral-950) 97%, var(--color-white))` | 1128 / 1197 | M |
| `--popover-foreground` | `#18181b` | `#f5f5f5` | `var(--color-zinc-900)` | `var(--color-neutral-100)` | 1129 / 1198 | M |
| `--surface-raised` | `#ffffff33` | `#111111` | `color-mix(in srgb, var(--card) 20%, transparent)` | `color-mix(in srgb, var(--color-neutral-950) 97%, var(--color-white))` | 1124 / 1193 | M |
| `--primary` | `#1b4ed8` | `#346bf1` | `oklch(0.488 0.217 264)` | `oklch(0.571 0.21 264)` | 1130 / 1199 | M |
| `--primary-foreground` | `#ffffff` | `#ffffff` | `var(--color-white)` | = | 1131 | M |
| `--secondary` | `#fafafa` | `#ffffff08` | `var(--color-zinc-50)` | `--alpha(var(--color-white) / 3%)` | 1132 / 1200 | M |
| `--secondary-foreground` | `#27272a` | `#f5f5f5` | `var(--color-zinc-800)` | `var(--color-neutral-100)` | 1133 / 1201 | M |
| `--muted` | `#fafafa` | `#ffffff08` | `var(--color-zinc-50)` | `--alpha(var(--color-white) / 3%)` | 1134 / 1202 | M |
| `--muted-foreground` | `#71717b` | `#a1a1a1` | `var(--color-zinc-500)` | `var(--color-neutral-400)` | 1135 / 1203 | M |
| `--accent` | `#f4f4f5` | `#ffffff0a` | `var(--color-zinc-100)` | `--alpha(var(--color-white) / 4%)` | 1145 / 1204 | M |
| `--accent-foreground` | `#18181b` | `#f5f5f5` | `var(--color-zinc-900)` | `var(--color-neutral-100)` | 1146 / 1205 | M |
| `--placeholder` | `#71717b` | `#a1a1a1` | `var(--muted-foreground)` | = | 1136 | M |
| `--secondary-label` | `#71717b` | `#a1a1a1` | `var(--muted-foreground)` | = | 1137 | M |
| `--icon-muted` | `#71717b` | `#a1a1a1` | `var(--muted-foreground)` | = | 1138 | M |
| `--border` | `#e4e4e7` | `#ffffff14` | `var(--color-zinc-200)` | `--alpha(var(--color-white) / 8%)` | 1152 / 1212 | M |
| `--input` | `#d4d4d8` | `#ffffff1c` | `var(--color-zinc-300)` | `--alpha(var(--color-white) / 11%)` | 1153 / 1213 | M |
| `--ring` | `#1b4ed8` | `#346bf1` | `var(--primary)` | = | 1154 | M |
| `--message-surface` | `#ebebed` | `#1e1e1e` | `color-mix(in srgb, var(--color-zinc-200) 55%, var(--color-zinc-100))` | `color-mix(in srgb, var(--color-neutral-950) 92%, var(--color-white))` | 1140 / 1207 | M |
| `--message-foreground` | `#18181b` | `#f5f5f5` | `var(--foreground)` | = | 1141 | M |
| `--message-action` | `#1b4ed8` | `#346bf1` | `var(--primary)` | = | 1142 | M |
| `--message-action-foreground` | `#ffffff` | `#ffffff` | `var(--primary-foreground)` | = | 1143 | M |
| `--message-action-hover` | `#3160db` | `#2f60d9` | `color-mix(in srgb, var(--primary) 90%, var(--background))` | = | 1144 | M |
| `--code-background` | `#ffffff` | `#101010` | `color-mix(in srgb, var(--card) 90%, var(--background))` | = | 1180 | M |
| `--code-foreground` | `#18181b` | `#f5f5f5` | `var(--foreground)` | = | 1181 | M |

Notes:

- In light, `--surface-raised` is a 20% white film (`#ffffff33`), not an opaque colour.
- In dark, `--secondary`/`--muted` (white 3%), `--accent` (white 4%), `--border` (white 8%) and `--input` (white 11%) are translucent white, mixed in OKLab (`--alpha()`). Composite them over whatever they sit on.
- `--message-action-hover` mixes the primary 90% with the canvas in sRGB: `#3160db` in light, `#2f60d9` in dark.

### 3.2 Status, feedback and diff colours (`:root`)

| Token | Light | Dark | Declared (light) | Declared (dark, if different) | index.css | M/S |
|---|---|---|---|---|---|---|
| `--error` | `#fb2c36` | `#fb414a` | `var(--color-red-500)` | `color-mix(in srgb, var(--color-red-500) 90%, var(--color-white))` | 1147 / 1208 | M |
| `--error-foreground` | `#c10007` | `#ff6467` | `var(--color-red-700)` | `var(--color-red-400)` | 1148 / 1209 | M |
| `--error-surface` | `#fb2c3614` | `#fb414a29` | `color-mix(in srgb, var(--error) 8%, transparent)` | `color-mix(in srgb, var(--error) 16%, transparent)` | 1150 / 1211 | M |
| `--tool-error-icon` | `#fb2c36` | `#fca5a5` | `var(--error)` | `#fca5a5` | 1149 / 1210 | M |
| `--destructive` | `#fb2c36` | `#fb414a` | `var(--error)` | = | 1151 | M |
| `--destructive-foreground` | `#c10007` | `#ff6467` | `var(--error-foreground)` | = | 1155 | M |
| `--warning` | `#fe9a00` | `#fe9a00` | `var(--color-amber-500)` | = | 1164 | M |
| `--warning-foreground` | `#bb4d00` | `#ffb900` | `var(--color-amber-700)` | `var(--color-amber-400)` | 1165 / 1217 | M |
| `--warning-surface` | `#fe9a0014` | `#fe9a0029` | `color-mix(in srgb, var(--warning) 8%, transparent)` | `color-mix(in srgb, var(--warning) 16%, transparent)` | 1166 / 1218 | M |
| `--info` | `#2b7fff` | `#2b7fff` | `var(--color-blue-500)` | = | 1156 | M |
| `--info-foreground` | `#1447e6` | `#50a2ff` | `var(--color-blue-700)` | `var(--color-blue-400)` | 1157 / 1214 | M |
| `--success` | `#00bc7d` | `#00bc7d` | `var(--color-emerald-500)` | = | 1162 | M |
| `--success-foreground` | `#007a55` | `#00d492` | `var(--color-emerald-700)` | `var(--color-emerald-400)` | 1163 / 1216 | M |
| `--guest` | `#e12afb` | `#e12afb` | `var(--color-fuchsia-500)` | = | 1159 | M |
| `--guest-foreground` | `#c800de` | `#f4a8ff` | `var(--color-fuchsia-600)` | `var(--color-fuchsia-300)` | 1160 / 1215 | M |
| `--guest-surface` | `#ead0ef` | `#391f3d` | `color-mix(in srgb, var(--guest) 14%, var(--message-surface))` | = | 1161 | M |
| `--update` | `#1b4ed8` | `#346bf1` | `var(--primary)` | = | 1167 | M |
| `--update-foreground` | `#1b4ed8` | `#50a2ff` | `var(--primary)` | `var(--color-blue-400)` | 1168 / 1219 | M |
| `--diff-addition` | `#00bc7d` | `#00bc7d` | `var(--success)` | = | 2285 | M |
| `--diff-deletion` | `#fb2c36` | `#fb414a` | `var(--destructive)` | = | 2286 | M |
| `--diff-addition-foreground` | `#007a55` | `#00d492` | `light-dark(var(--color-emerald-700), var(--color-emerald-400))` | = | 2287 | M |
| `--diff-deletion-foreground` | `#c10007` | `#ff6467` | `light-dark(var(--color-red-700), var(--color-red-400))` | = | 2288 | M |
| `--diff-modified-icon` | `#425cb1` | `#50a2ffa6` | `color-mix(in srgb, var(--info-foreground) 50%, var(--muted-foreground…` | `color-mix(in srgb, var(--info-foreground) 65%, transparent)` | 1283 / 1287 | M |
| `--diff-added-icon` | (unset) | `#00d492` | `(declared only in dark, index.css:1288)` | `var(--success-foreground)` |  / 1288 | M |

`--diff-added-icon` exists only in dark (index.css:1288). In light, components fall back to their own colour.

With the **Blue & orange** diff setting, `:root[data-diff-color-scheme="blue-orange"]` (index.css:1294–1301, 2301–2314) replaces the diff tokens. Values are the declared expressions evaluated live on probes (P). The setting was not switched.

| Token (blue-orange) | Light | Dark | Declared | M/S |
|---|---|---|---|---|
| `--diff-addition` | `#155dfc` | `#50a2ff` | `light-dark(var(--color-blue-600), var(--color-blue-400))` | P |
| `--diff-deletion` | `#f54a00` | `#ff8904` | `light-dark(var(--color-orange-600), var(--color-orange-400))` | P |
| `--diff-addition-foreground` | `#155dfc` | `#50a2ff` | `var(--diff-addition)` | P |
| `--diff-deletion-foreground` | `#f54a00` | `#ff8904` | `var(--diff-deletion)` | P |
| `--diff-modified-icon` | `#71717b` | `#a1a1a1a6` | `var(--muted-foreground)`; dark `color-mix(in srgb, var(--muted-foreground) 65%, transparent)` | P |
| `--diff-added-icon` | (unset) | (unset) | `initial` | S |

The same rule points the `@pierre/diffs` and file-tree hooks at these colours: `--diffs-addition-color-override`, `--trees-status-added-override`, `--trees-status-untracked-override`, `--trees-git-added-color-override`, `--trees-git-untracked-color-override` take `--diff-addition`, and `--diffs-deletion-color-override`, `--trees-status-deleted-override`, `--trees-git-deleted-color-override` take `--diff-deletion`.

### 3.3 Toolbar, sidebar and terminal roles (`:root` scope)

| Token | Light | Dark | Declared (light) | Declared (dark, if different) | index.css | M/S |
|---|---|---|---|---|---|---|
| `--toolbar-background` | `#fcfcfc` | `#000000` | `var(--app-chrome-background)` | = | 1118 | M |
| `--toolbar-foreground` | `#18181b` | `#f5f5f5` | `var(--foreground)` | = | 1119 | M |
| `--toolbar-border` | `#e4e4e7` | `#ffffff14` | `var(--border)` | = | 1120 | M |
| `--toolbar-control` | `#ffffff` | `#111111` | `var(--popover)` | = | 1121 | M |
| `--toolbar-control-foreground` | `#18181b` | `#f5f5f5` | `var(--foreground)` | = | 1122 | M |
| `--toolbar-control-hover` | `#f4f4f5` | `#ffffff0a` | `var(--accent)` | = | 1123 | M |
| `--sidebar` | `#fafafa` | `#111111` | `var(--color-zinc-50)` | `var(--card)` | 1171 / 1220 | M |
| `--sidebar-foreground` | `#18181b` | `#f5f5f5` | `var(--foreground)` | = | 1172 | M |
| `--sidebar-muted-foreground` | `#71717b` | `#a1a1a1` | `var(--muted-foreground)` | = | 1173 | M |
| `--sidebar-control-surface` | `#f4f4f5` | `#ffffff08` | `var(--color-zinc-100)` | `var(--muted)` | 1174 / 1221 | M |
| `--sidebar-row-hover` | `#fcfcfc` | `#ffffff0a` | `var(--color-zinc-25)` | `var(--accent)` | 1175 / 1222 | M |
| `--sidebar-row-active` | `#ffffff` | `#ffffff0a` | `var(--color-white)` | `var(--accent)` | 1176 / 1223 | M |
| `--sidebar-row-selected` | `#ffffff` | `#ffffff08` | `var(--color-white)` | `var(--muted)` | 1177 / 1224 | M |
| `--sidebar-border` | `#e4e4e7` | `#ffffff14` | `var(--border)` | = | 1178 | M |
| `--sidebar-stage-fade` | `#fafafa` | `#111111` | `var(--sidebar)` | `var(--card)` | 1179 / 1225 | M |
| `--sidebar-icon-color` | `#85868e` | `#8b8b8b` | `color-mix(in srgb, var(--contrast-sidebar-muted-foreground) 85%, var(…` | = | 106 | M |
| `--terminal-background` | `#fcfcfc` | `#000000` | `var(--background)` | = | 1182 | M |
| `--terminal-foreground` | `#18181b` | `#f5f5f5` | `var(--foreground)` | = | 1183 | M |
| `--terminal-cursor` | `#26384e` | `#b4cbff` | `rgb(38 56 78)` | `rgb(180 203 255)` | 1184 / 1226 | M |
| `--terminal-selection-background` | `#253f6333` | `#b4cbff40` | `rgb(37 63 99 / 20%)` | `rgb(180 203 255 / 25%)` | 1185 / 1227 | M |

`--sidebar-icon-color` is declared only on `:root` (index.css:106–110). It is **not** recomputed inside `[data-app-sidebar]`, so the sidebar inherits the root value: `#85868e` light, `#8b8b8b` dark (measured inside the sidebar too).

### 3.4 Contrast-adjusted tokens (`:root`, default contrast 100)

At contrast 100 each `--contrast-*` equals its input role. The formulas (index.css:1600–1712, shipped `@supports` form) are:

- foreground kind: `color-mix(in oklab, color-mix(in oklab, <role> var(--appearance-contrast-base), <surface>), var(--appearance-contrast-target) var(--appearance-contrast-boost))`. `<surface>` is the role's own background: `--background` for foreground, muted-foreground, placeholder, secondary-label and icon-muted. Card, popover, accent, secondary, message-surface, sidebar, toolbar-background and toolbar-control pair with their own foregrounds.
- border kind (`--contrast-border`, `--contrast-input`, `--contrast-sidebar-border`, `--contrast-toolbar-border`): `color-mix(in srgb, color-mix(in srgb, <role> var(--appearance-contrast-base), transparent), <foreground role> var(--appearance-contrast-border-boost))`.
- `--appearance-contrast-target` is `black` in light and `white` in dark (index.css:96, 146).

| Token | Light | Dark | index.css | M/S |
|---|---|---|---|---|
| `--contrast-toolbar-foreground` | `#18181b` | `#f5f5f5` | 1602 | M |
| `--contrast-toolbar-border` | `#e4e4e7` | `#ffffff14` | 1611 | M |
| `--contrast-toolbar-control-foreground` | `#18181b` | `#f5f5f5` | 1616 | M |
| `--contrast-input` | `#d4d4d8` | `#ffffff1c` | 1625 | M |
| `--contrast-border` | `#e4e4e7` | `#ffffff14` | 1630 | M |
| `--contrast-foreground` | `#18181b` | `#f5f5f5` | 1635 | M |
| `--contrast-muted-foreground` | `#71717b` | `#a1a1a1` | 1640 | M |
| `--contrast-placeholder` | `#71717b` | `#a1a1a1` | 1645 | M |
| `--contrast-secondary-label` | `#71717b` | `#a1a1a1` | 1650 | M |
| `--contrast-icon-muted` | `#71717b` | `#a1a1a1` | 1655 | M |
| `--contrast-card-foreground` | `#18181b` | `#f5f5f5` | 1660 | M |
| `--contrast-popover-foreground` | `#18181b` | `#f5f5f5` | 1665 | M |
| `--contrast-accent-foreground` | `#18181b` | `#f5f5f5` | 1670 | M |
| `--contrast-secondary-foreground` | `#27272a` | `#f5f5f5` | 1675 | M |
| `--contrast-message-foreground` | `#18181b` | `#f5f5f5` | 1684 | M |
| `--contrast-sidebar-foreground` | `#18181b` | `#f5f5f5` | 1693 | M |
| `--contrast-sidebar-muted-foreground` | `#71717b` | `#a1a1a1` | 1698 | M |
| `--contrast-sidebar-border` | `#e4e4e7` | `#ffffff14` | 1707 | M |

### 3.5 Sidebar scope (`[data-app-sidebar]`)

The sidebar container re-declares its palette (index.css:1234–1278). The `:root, [data-app-sidebar]` rule (1600–1712) recomputes every `--contrast-*` from that palette, and `@theme inline` utilities resolve on the element. Measured inside `div[data-slot=sidebar-container][data-app-sidebar]`:

| Token (inside `[data-app-sidebar]`) | Light | Dark | :root dark (for comparison) | Declared dark | M/S |
|---|---|---|---|---|---|
| `--background` | `#fcfcfc` | `#000000` | `#000000` | `#000` | M |
| `--foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `#f1f3f7` | M |
| `--card` | `#ffffff` | `#000000` | `#111111` | `#000` | M |
| `--card-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `var(--foreground)` | M |
| `--accent` | `#f4f4f5` | `#191a1d` | `#ffffff0a` | `#191a1d` | M |
| `--accent-foreground` | `#18181b` | `#f7f9ff` | `#f5f5f5` | `#f7f9ff` | M |
| `--muted` | `#fafafa` | `#0a0a0a` | `#ffffff08` | `#0a0a0a` | M |
| `--muted-foreground` | `#71717b` | `#a3a3a3` | `#a1a1a1` | `#a3a3a3` | M |
| `--border` | `#e4e4e7` | `#ffffff14` | `#ffffff14` | `rgb(255 255 255 / 8%)` | M |
| `--input` | `#d4d4d8` | `#ffffff2e` | `#ffffff1c` | `rgb(255 255 255 / 18%)` | M |
| `--sidebar` | `#fafafa` | `#000000` | `#111111` | `var(--card)` | M |
| `--sidebar-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `var(--foreground)` | M |
| `--sidebar-muted-foreground` | `#71717b` | `#a3a3a3` | `#a1a1a1` | `var(--muted-foreground)` | M |
| `--sidebar-control-surface` | `#f4f4f5` | `#0a0a0a` | `#ffffff08` | `var(--muted)` | M |
| `--sidebar-row-hover` | `#fcfcfc` | `#f1f3f714` | `#ffffff0a` | `color-mix(in srgb, var(--contrast-foreground) 8%, transpare…` | M |
| `--sidebar-row-active` | `#ffffff` | `#f1f3f71c` | `#ffffff0a` | `color-mix(in srgb, var(--contrast-foreground) 11%, transpar…` | M |
| `--sidebar-row-selected` | `#ffffff` | `#f1f3f712` | `#ffffff08` | `color-mix(in srgb, var(--contrast-foreground) 7%, transpare…` | M |
| `--sidebar-border` | `#e4e4e7` | `#ffffff14` | `#ffffff14` | `var(--border)` | M |
| `--sidebar-stage-fade` | `#fafafa` | `#000000` | `#111111` | `var(--card)` | M |
| `--secondary-label` | `#71717b` | `#8f8f8f` | `#a1a1a1` | `#8f8f8f` | M |
| `--contrast-toolbar-foreground` | `#18181b` | `#f5f5f5` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--toolbar-foreg…` | M |
| `--contrast-toolbar-border` | `#e4e4e7` | `#ffffff14` | `#ffffff14` | `color-mix(in srgb, color-mix(in srgb, var(--toolbar-border)…` | M |
| `--contrast-toolbar-control-foreground` | `#18181b` | `#f5f5f5` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--toolbar-contr…` | M |
| `--contrast-input` | `#d4d4d8` | `#ffffff2e` | `#ffffff1c` | `color-mix(in srgb, color-mix(in srgb, var(--input) var(--ap…` | M |
| `--contrast-border` | `#e4e4e7` | `#ffffff14` | `#ffffff14` | `color-mix(in srgb, color-mix(in srgb, var(--border) var(--a…` | M |
| `--contrast-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--foreground) v…` | M |
| `--contrast-muted-foreground` | `#71717b` | `#a3a3a3` | `#a1a1a1` | `color-mix(in oklab, color-mix(in oklab, var(--muted-foregro…` | M |
| `--contrast-placeholder` | `#71717b` | `#a1a1a1` | `#a1a1a1` | `color-mix(in oklab, color-mix(in oklab, var(--placeholder) …` | M |
| `--contrast-secondary-label` | `#71717b` | `#8f8f8f` | `#a1a1a1` | `color-mix(in oklab, color-mix(in oklab, var(--secondary-lab…` | M |
| `--contrast-icon-muted` | `#71717b` | `#a1a1a1` | `#a1a1a1` | `color-mix(in oklab, color-mix(in oklab, var(--icon-muted) v…` | M |
| `--contrast-card-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--card-foregrou…` | M |
| `--contrast-popover-foreground` | `#18181b` | `#f5f5f5` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--popover-foreg…` | M |
| `--contrast-accent-foreground` | `#18181b` | `#f7f9ff` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--accent-foregr…` | M |
| `--contrast-secondary-foreground` | `#27272a` | `#f5f5f5` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--secondary-for…` | M |
| `--contrast-message-foreground` | `#18181b` | `#f5f5f5` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--message-foreg…` | M |
| `--contrast-sidebar-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `color-mix(in oklab, color-mix(in oklab, var(--sidebar-foreg…` | M |
| `--contrast-sidebar-muted-foreground` | `#71717b` | `#a3a3a3` | `#a1a1a1` | `color-mix(in oklab, color-mix(in oklab, var(--sidebar-muted…` | M |
| `--contrast-sidebar-border` | `#e4e4e7` | `#ffffff14` | `#ffffff14` | `color-mix(in srgb, color-mix(in srgb, var(--sidebar-border)…` | M |
| `--color-accent` | `#f4f4f5` | `#191a1d` | `#ffffff0a` | `var(--accent)` | P |
| `--color-accent-foreground` | `#18181b` | `#f7f9ff` | `#f5f5f5` | `var(--contrast-accent-foreground)` | P |
| `--color-card` | `#ffffff` | `#000000` | `#111111` | `var(--card)` | P |
| `--color-card-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `var(--contrast-card-foreground)` | P |
| `--color-input` | `#d4d4d8` | `#ffffff2e` | `#ffffff1c` | `var(--contrast-input)` | P |
| `--color-muted-foreground` | `#71717b` | `#a3a3a3` | `#a1a1a1` | `var(--contrast-muted-foreground)` | P |
| `--color-secondary-label` | `#71717b` | `#8f8f8f` | `#a1a1a1` | `var(--contrast-secondary-label)` | P |
| `--color-sidebar` | `#fafafa` | `#000000` | `#111111` | `var(--sidebar)` | P |
| `--color-sidebar-control-surface` | `#f4f4f5` | `#0a0a0a` | `#ffffff08` | `var(--sidebar-control-surface)` | P |
| `--color-sidebar-foreground` | `#18181b` | `#f1f3f7` | `#f5f5f5` | `var(--contrast-sidebar-foreground)` | P |
| `--color-sidebar-muted-foreground` | `#71717b` | `#a3a3a3` | `#a1a1a1` | `var(--contrast-sidebar-muted-foreground)` | P |
| `--color-sidebar-row-active` | `#ffffff` | `#f1f3f71c` | `#ffffff0a` | `var(--sidebar-row-active)` | P |
| `--color-sidebar-row-hover` | `#fcfcfc` | `#f1f3f714` | `#ffffff0a` | `var(--sidebar-row-hover)` | P |
| `--color-sidebar-row-selected` | `#ffffff` | `#f1f3f712` | `#ffffff08` | `var(--sidebar-row-selected)` | P |
| `--sidebar-icon-color` | `#85868e` | `#8b8b8b` | `#8b8b8b` | `inherited from :root (not re-declared in the sidebar) [inde…` | M |

**Light: every value equals the root value** (measured, zero differences). **Dark: the sidebar has its own palette.** Its surface is true black (`--sidebar`/`--card` `#000000`, while the root dark card is `#111111`). Text is `#f1f3f7`, muted `#a3a3a3`, secondary-label `#8f8f8f`. Rows are translucent `#f1f3f7`: hover 8% (`#f1f3f714`), active 11% (`#f1f3f71c`), selected 7% (`#f1f3f712`). Accent is `#191a1d`, control surface `#0a0a0a`, input white 18%.

### 3.6 Utility colour names (`@theme inline`, index.css:233–288)

These are not runtime variables. Tailwind inlines them, so `bg-sidebar-row-hover` compiles to `background-color: var(--sidebar-row-hover)`. Values are at `:root`. Inside the sidebar use §3.5.

Four are also emitted into `@layer theme` on `:root, :host`, because something reads them with `var()`. `--color-primary` and `--color-foreground` are read by the `stash-pulse-*` and `settings-search-target-pulse` keyframes and a few arbitrary-value utilities. `--color-ring` is read by an inline style (PreviewChromeRow.tsx:299) and `--color-muted` by another (UsageLimits.tsx:95). Their `var()` is substituted on `:root` and the result is inherited. So inside `[data-app-sidebar]`, `var(--color-foreground)` still gives the root value (measured: dark `#f5f5f5`, not the sidebar's `#f1f3f7`). Utilities like `text-foreground` are unaffected, because they compile to `var(--contrast-foreground)`, which resolves on the element.

| Utility colour (`bg-*`, `text-*`, `border-*` …) | Maps to | Light | Dark | M/S |
|---|---|---|---|---|
| `ring` (`--color-ring`) | `var(--ring)` | `#1b4ed8` | `#346bf1` | M |
| `muted` (`--color-muted`) | `var(--muted)` | `#fafafa` | `#ffffff08` | M |
| `primary` (`--color-primary`) | `var(--primary)` | `#1b4ed8` | `#346bf1` | M |
| `foreground` (`--color-foreground`) | `var(--contrast-foreground)` | `#18181b` | `#f5f5f5` | M |
| `warning-foreground` (`--color-warning-foreground`) | `var(--warning-foreground)` | `#bb4d00` | `#ffb900` | P |
| `warning` (`--color-warning`) | `var(--warning)` | `#fe9a00` | `#fe9a00` | P |
| `warning-surface` (`--color-warning-surface`) | `var(--warning-surface)` | `#fe9a0014` | `#fe9a0029` | P |
| `error-foreground` (`--color-error-foreground`) | `var(--error-foreground)` | `#c10007` | `#ff6467` | P |
| `tool-error-icon` (`--color-tool-error-icon`) | `var(--tool-error-icon)` | `#fb2c36` | `#fca5a5` | P |
| `error` (`--color-error`) | `var(--error)` | `#fb2c36` | `#fb414a` | P |
| `error-surface` (`--color-error-surface`) | `var(--error-surface)` | `#fb2c3614` | `#fb414a29` | P |
| `update-foreground` (`--color-update-foreground`) | `var(--update-foreground)` | `#1b4ed8` | `#50a2ff` | P |
| `update` (`--color-update`) | `var(--update)` | `#1b4ed8` | `#346bf1` | P |
| `success-foreground` (`--color-success-foreground`) | `var(--success-foreground)` | `#007a55` | `#00d492` | P |
| `success` (`--color-success`) | `var(--success)` | `#00bc7d` | `#00bc7d` | P |
| `diff-addition` (`--color-diff-addition`) | `var(--diff-addition)` | `#00bc7d` | `#00bc7d` | P |
| `diff-deletion` (`--color-diff-deletion`) | `var(--diff-deletion)` | `#fb2c36` | `#fb414a` | P |
| `diff-addition-foreground` (`--color-diff-addition-foreground`) | `var(--diff-addition-foreground)` | `#007a55` | `#00d492` | P |
| `diff-deletion-foreground` (`--color-diff-deletion-foreground`) | `var(--diff-deletion-foreground)` | `#c10007` | `#ff6467` | P |
| `info-foreground` (`--color-info-foreground`) | `var(--info-foreground)` | `#1447e6` | `#50a2ff` | P |
| `info` (`--color-info`) | `var(--info)` | `#2b7fff` | `#2b7fff` | P |
| `guest` (`--color-guest`) | `var(--guest)` | `#e12afb` | `#e12afb` | P |
| `guest-foreground` (`--color-guest-foreground`) | `var(--guest-foreground)` | `#c800de` | `#f4a8ff` | P |
| `guest-surface` (`--color-guest-surface`) | `var(--guest-surface)` | `#ead0ef` | `#391f3d` | P |
| `destructive-foreground` (`--color-destructive-foreground`) | `var(--destructive-foreground)` | `#c10007` | `#ff6467` | P |
| `input` (`--color-input`) | `var(--contrast-input)` | `#d4d4d8` | `#ffffff1c` | P |
| `border` (`--color-border`) | `var(--contrast-border)` | `#e4e4e7` | `#ffffff14` | P |
| `destructive` (`--color-destructive`) | `var(--destructive)` | `#fb2c36` | `#fb414a` | P |
| `accent-foreground` (`--color-accent-foreground`) | `var(--contrast-accent-foreground)` | `#18181b` | `#f5f5f5` | P |
| `accent` (`--color-accent`) | `var(--accent)` | `#f4f4f5` | `#ffffff0a` | P |
| `muted-foreground` (`--color-muted-foreground`) | `var(--contrast-muted-foreground)` | `#71717b` | `#a1a1a1` | P |
| `placeholder` (`--color-placeholder`) | `var(--contrast-placeholder)` | `#71717b` | `#a1a1a1` | P |
| `secondary-label` (`--color-secondary-label`) | `var(--contrast-secondary-label)` | `#71717b` | `#a1a1a1` | P |
| `icon-muted` (`--color-icon-muted`) | `var(--contrast-icon-muted)` | `#71717b` | `#a1a1a1` | P |
| `secondary-foreground` (`--color-secondary-foreground`) | `var(--contrast-secondary-foreground)` | `#27272a` | `#f5f5f5` | P |
| `secondary` (`--color-secondary`) | `var(--secondary)` | `#fafafa` | `#ffffff08` | P |
| `primary-foreground` (`--color-primary-foreground`) | `var(--primary-foreground)` | `#ffffff` | `#ffffff` | P |
| `popover-foreground` (`--color-popover-foreground`) | `var(--contrast-popover-foreground)` | `#18181b` | `#f5f5f5` | P |
| `popover` (`--color-popover`) | `var(--popover)` | `#ffffff` | `#111111` | P |
| `card-foreground` (`--color-card-foreground`) | `var(--contrast-card-foreground)` | `#18181b` | `#f5f5f5` | P |
| `card` (`--color-card`) | `var(--card)` | `#ffffff` | `#111111` | P |
| `code` (`--color-code`) | `var(--code-background)` | `#ffffff` | `#101010` | P |
| `background` (`--color-background`) | `var(--background)` | `#fcfcfc` | `#000000` | P |
| `message` (`--color-message`) | `var(--message-surface)` | `#ebebed` | `#1e1e1e` | P |
| `message-foreground` (`--color-message-foreground`) | `var(--contrast-message-foreground)` | `#18181b` | `#f5f5f5` | P |
| `message-action` (`--color-message-action`) | `var(--message-action)` | `#1b4ed8` | `#346bf1` | P |
| `message-action-foreground` (`--color-message-action-foreground`) | `var(--message-action-foreground)` | `#ffffff` | `#ffffff` | P |
| `message-action-hover` (`--color-message-action-hover`) | `var(--message-action-hover)` | `#3160db` | `#2f60d9` | P |
| `sidebar` (`--color-sidebar`) | `var(--sidebar)` | `#fafafa` | `#111111` | P |
| `sidebar-foreground` (`--color-sidebar-foreground`) | `var(--contrast-sidebar-foreground)` | `#18181b` | `#f5f5f5` | P |
| `sidebar-muted-foreground` (`--color-sidebar-muted-foreground`) | `var(--contrast-sidebar-muted-foreground)` | `#71717b` | `#a1a1a1` | P |
| `sidebar-control-surface` (`--color-sidebar-control-surface`) | `var(--sidebar-control-surface)` | `#f4f4f5` | `#ffffff08` | P |
| `sidebar-row-hover` (`--color-sidebar-row-hover`) | `var(--sidebar-row-hover)` | `#fcfcfc` | `#ffffff0a` | P |
| `sidebar-row-active` (`--color-sidebar-row-active`) | `var(--sidebar-row-active)` | `#ffffff` | `#ffffff0a` | P |
| `sidebar-row-selected` (`--color-sidebar-row-selected`) | `var(--sidebar-row-selected)` | `#ffffff` | `#ffffff08` | P |
| `sidebar-border` (`--color-sidebar-border`) | `var(--contrast-sidebar-border)` | `#e4e4e7` | `#ffffff14` | P |

### 3.7 Chrome inputs: glass, scrollbar, contrast and grain (`:root`)

| Token | Light | Dark | Declared (light) | Declared (dark, if different) | index.css | M/S |
|---|---|---|---|---|---|---|
| `--glass-blur` | `12px` | `16px` | `12px` | `16px` | 119 / 149 | M |
| `--glass-opacity` | `80%` | `80%` | `glassOpacity% (default 80) [html style, appInit.tsx:94]; index.css:12…` | = | 120 | M |
| `--glass-saturation` | `1.14` | `1.08` | `1.14` | `1.08` | 121 / 150 | M |
| `--app-scrollbar-width` | `6px` | `6px` | `6px` | = | 92 | M |
| `--app-scrollbar-thumb` | `#d9d9d9` | `#ffffff14` | `rgb(217 217 217)` | `rgb(255 255 255 / 8%)` | 97 / 147 | M |
| `--app-scrollbar-thumb-hover` | `#bfbfbf` | `#ffffff1f` | `rgb(191 191 191)` | `rgb(255 255 255 / 12%)` | 98 / 148 | M |
| `--appearance-contrast-base` | `100%` | `100%` | `min(appearanceContrast, 100)% [html style, appearanceContrast.ts:4]` | = | 93 | M |
| `--appearance-contrast-boost` | `0%` | `0%` | `max(appearanceContrast - 100, 0)% [html style, appearanceContrast.ts:…` | = | 94 | M |
| `--appearance-contrast-border-boost` | `0%` | `0%` | `max(appearanceContrast - 100, 0) / 4 % [html style, appearanceContras…` | = | 95 | M |
| `--appearance-contrast-target` | `#000000` | `#ffffff` | `black` | `white` | 96 / 146 | M |
| `--surface-grain-size` | `256px 256px` | `256px 256px` | `256px 256px` | = | 1770 | M |
| `--z-sheet` | `46` | `46` | `46` | = | 118 | M |
| `--lightningcss-light` | unset (`initial` = guaranteed-invalid, so `var()` takes its fallback) | empty value (set, so `var()` gives nothing) | `initial` | (empty) | Lightning CSS | M |
| `--lightningcss-dark` | empty value (set, so `var()` gives nothing) | unset (`initial`) | (empty) | `initial` | Lightning CSS | M |

`--surface-grain` (index.css:1769) is `url("data:image/svg+xml,…")`: an SVG `feTurbulence type=fractalNoise baseFrequency=0.9 numOctaves=4 stitchTiles=stitch`, drawn as a rect at `opacity=0.035`, tiled at 256×256px. With `data-theme-id="t3-chat"` it is a 128px tile at opacity `0.02745` (index.css:1776–1779). It paints on any element with `surface-grain` (the sidebar inner and the main inset, in both themes) and on `body` **in light only** (1781–1785). In dark, the boot `<style>`'s `html.dark body { background: #000000 }` (index.html:405–408, unlayered, specificity 0,1,2) beats index.css's `body { background-image: … }` (0,0,1) and its `background` shorthand resets the image. Measured on body: light `background-image: url(…)`, `background-size: 256px 256px`; dark `background-image: none`, `background-size: auto`. In light, the boot `body { background: #ffffff }` has the same specificity as index.css's rule, and index.css loads later, so the grain survives. **Measured effect:** the light sidebar `#fafafa` paints at mean 248.8/255 (range 246–250, sd 0.6). The dark sidebar `#000000` paints at mean 3.8/255 (range 1–7). The main canvas paints flat `#fcfcfc` / `#000000`, because opaque content covers it.

### 3.8 Geometry tokens (`:root`)

| Token | Light (16px root) | Dark | Declared (light) | Declared (dark, if different) | index.css | M/S |
|---|---|---|---|---|---|---|
| `--radius` | `8px` | `8px` | `0.5rem` | = | 1115 | M |
| `--control-radius` | `8px` | `8px` | `0.5rem` | = | 103 | M |
| `--spacing` | `4px` | `4px` | `.25rem` | = | Tailwind | M |
| `--spacing-composer-seam` | `15px` | `15px` | `calc(var(--spacing) * 4 - 1px)` (`@theme inline`: not a runtime variable, and no utility in the shipped CSS uses it) | = | 201 | P |
| `--sidebar-content-inset` | `8px` | `8px` | `0.5rem` | = | 104 | M |
| `--sidebar-control-gap` | `8px` | `8px` | `0.5rem` | = | 105 | M |
| `--sidebar-row-content-inset` | `10px` | `10px` | `0.625rem` | = | 111 | M |
| `--command-shell-inset` | `8px` | `8px` | `0.5rem` | = | 112 | M |
| `--command-content-inset` | `16px` | `16px` | `1rem` | = | 113 | M |
| `--floating-content-inset` | `12px` | `12px` | `0.75rem` | = | 114 | M |
| `--desktop-window-right-resize-inset` | `0px` | `0px` | `0px` | = | 122 | M |
| `--workspace-topbar-height` | `52px` | `52px` | `52px` | = | 123 | M |
| `--workspace-controls-top` | `0px` | `0px` | `0px` | = | 124 | M |
| `--workspace-controls-left` | `12px` | `12px` | `calc(env(safe-area-inset-left) + 0.75rem)` | = | 125 | M |
| `--workspace-controls-right` | `12px` | `12px` | `calc(env(safe-area-inset-right) + 0.75rem)` | = | 126 | M |
| `--workspace-native-controls-inset` | `0px` | `0px` | `0px` | = | 127 | M |
| `--workspace-titlebar-control-size` | `28px` | `28px` | `1.75rem` | = | 128 | M |
| `--workspace-titlebar-control-gap` | `12px` | `12px` | `0.75rem` | = | 129 | M |
| `--workspace-titlebar-scroll-fade-height` | `24px` | `24px` | `1.5rem` | = | 130 | M |
| `--workspace-gutter` | `20px` | `20px` | `1.25rem at >=40rem (sm), .75rem below [index.css:137,141-14…` | = | 137,142 | M |
| `--workspace-gutter-start` | `20px` | `20px` | `calc(env(safe-area-inset-left) + var(--workspace-gutter))` | = | 138 | M |
| `--workspace-gutter-end` | `20px` | `20px` | `calc(env(safe-area-inset-right) + var(--workspace-gutter))` | = | 139 | M |
| `--chat-content-max-width` | `736px` | `736px` | `46rem` | = | 131 | M |
| `--thread-details-panel-width` | `280px` | `280px` | `17.5rem` | = | 132 | M |
| `--font-size-prompt` | `14px` | `14px` | `var(--text-sm), or fontSizePrompt px with separate fonts [h…` | = | html style | M |
| `--font-size-prompt-touch` | `16px` | `16px` | `max(var(--font-size-prompt, 1rem), 16px)` | = | 134 | M |
| `--font-size-code` | `13px` | `13px` | `fontSizeCode px (default 13) [html style, appearanceFonts.t…` | = | html style | M |
| `--diffs-font-size` | `13px` | `13px` | `fontSizeCode px (default 13) [html style, appearanceFonts.t…` | = | html style | M |
| `--list-marker-sans` | `1.175em` | `1.175em` | `canvas width of "0. " in the sans stack at 16px, / 16, in e…` | = | html style | M |

Component-set lengths seen on the live wrapper: `--sidebar-width: 256px`, `--sidebar-width-icon: 3rem`, and `--workspace-titlebar-content-left: calc(var(--workspace-controls-left) + var(--workspace-titlebar-control-size) + var(--workspace-titlebar-control-gap))` = 52px. Below 640px the wrapper sets `--workspace-titlebar-control-size: --spacing(8)` = 32px. Mobile sidebar width is `calc(100vw - var(--spacing) * 3)` (sidebar.tsx:31).

The thread sidebar's `--sidebar-width` is a **px number**, written inline by AppSidebarLayout.tsx:452 from threadSidebarWidth.ts:1–34. It is not `16rem`, so it does **not** scale with the interface font size:

- default `16 * 16` = 256px;
- minimum `max(208px, ceil(brand width))`, where 208px is `13 * 16`;
- maximum `max(minimum, floor(viewport width) − 640px)`, keeping 640px for the main content;
- drag-resizable, persisted in localStorage `chat_thread_sidebar_width`.

sidebar.tsx:30's `SIDEBAR_WIDTH = "16rem"` (sidebar.tsx:243) applies only to sidebars that don't pass their own width.

### 3.9 Typography

**Fonts**

| Token | Declared stack | Renders here (Linux Chrome) | Renders on macOS | Source |
|---|---|---|---|---|
| `--font-sans` | `-apple-system, BlinkMacSystemFont, "Inter Variable", "Segoe UI", system-ui, sans-serif` | **Inter Variable** (bundled web font, latin subset loaded) | SF Pro (system UI font, automatic optical sizes) | index.css:171–172 (M) |
| `--font-mono` | `ui-monospace, "SF Mono", "SFMono-Regular", Menlo, Consolas, "Liberation Mono", monospace` | **Liberation Mono** (`ui-monospace` unsupported on Linux; SF Mono/Menlo/Consolas absent; canvas width equal to Liberation Mono) | SF Mono (`ui-monospace`) | index.css:173–174 (M) |
| `--font-composer` | unset → `font-family: var(--font-composer, var(--font-sans))` | = sans | = sans | appearanceFonts.ts:80 (M: unset) |
| `--diffs-font-family` / `--diffs-header-font-family` | `var(--font-mono)` / `var(--font-sans)` | | | index.css:1808–1811 |

- `@font-face`: 7 rules, family `Inter Variable`, `font-style: normal`, `font-weight: 100 900`, `font-display: swap`, `format("woff2-variations")`. One file per unicode-range subset: cyrillic-ext, cyrillic, greek-ext, greek, vietnamese, latin-ext, latin (shipped CSS, from `@fontsource-variable/inter` 5.x). **No italic face is bundled.** Italics are synthesized: `font-synthesis: weight style small-caps` (measured).
- No mono font and no icon font is bundled for the UI. Icons are inline lucide SVGs. `:where(svg.lucide[stroke-width="2"]) { stroke-width: 1.75 }` (index.css:1728–1732): **measured 99 of 99 icons on screen at `1.75px`**. The terminal ships its own Nerd Font symbols (`terminal/ghostty/fonts/SymbolsNerdFontMono-Regular.woff2`), and KaTeX fonts come with the lazy math CSS.
- `appearanceFonts.ts:16–18` has a different code stack (`"SF Mono", "SFMono-Regular", Menlo, Consolas, "Liberation Mono", monospace`, no `ui-monospace`). It is used **only** as the fallback after a user-chosen code font.
- Inherited text defaults on `html`/`body` (M): `font-size 16px`, `line-height 24px` (preflight 1.5), `font-weight 400`, `letter-spacing normal`, `font-feature-settings normal`, `font-variation-settings normal`, `font-optical-sizing auto`, `font-kerning auto`, `text-rendering auto`, `-webkit-font-smoothing antialiased` (setting on), `tab-size 4`, `-webkit-tap-highlight-color transparent`. Body colour `#262626` (light) / `#f5f5f5` (dark) (§2.1).
- `--list-marker-sans` = `1.175em` here. It is the canvas width of `"0. "` in the sans stack at 16px, divided by 16 (appearanceFonts.ts:91–99), so **it depends on the font** and will differ with SF Pro.

**Type scale** (probe elements with the real utility class; line height is the utility's own `--tw-leading` fallback)

| Utility | Size token | font-size | line-height | Declared | M/S |
|---|---|---|---|---|---|
| `text-5xs` | `--text-5xs` | `7px` | `7px` | `0.4375rem` / `1` | M |
| `text-4xs` | `--text-4xs` | `8px` | `8px` | `0.5rem` / `1` | M |
| `text-3xs` | `--text-3xs` | `10px` | `14px` | `0.625rem` / `calc(0.875 / 0.625)` | M |
| `text-2xs` | `--text-2xs` | `11px` | `16px` | `0.6875rem` / `calc(1 / 0.6875)` | M |
| `text-xs` | `--text-xs` | `12px` | `16px` | `.75rem` / `calc(1 / .75)` | M |
| `text-sm` | `--text-sm` | `14px` | `20px` | `.875rem` / `calc(1.25 / .875)` | M |
| `text-base` | `--text-base` | `16px` | `24px` | `1rem` / `calc(1.5 / 1)` | M |
| `text-lg` | `--text-lg` | `18px` | `28px` | `1.125rem` / `calc(1.75 / 1.125)` | M |
| `text-xl` | `--text-xl` | `20px` | `28px` | `1.25rem` / `calc(1.75 / 1.25)` | M |
| `text-2xl` | `--text-2xl` | `24px` | `32px` | `1.5rem` / `calc(2 / 1.5)` | M |
| `text-3xl` | `--text-3xl` | `30px` | `36px` | `1.875rem` / `calc(2.25 / 1.875)` | M |
| `text-4xl` | `--text-4xl` | `36px` | `40px` | `2.25rem` / `calc(2.5 / 2.25)` | M |

The declared line heights are unitless ratios (e.g. `calc(1 / .75)` = 1.33333). The table's px values are those ratios at the token's own size. `text-2xs`/`3xs` (index.css:183–186) are the dense-UI steps. `4xs`/`5xs` are for provider-icon badges.

**Tracking, leading and weights**

| Token | Value | Resolved at text-sm (14px) | M/S |
|---|---|---|---|
| `--tracking-tighter` (`tracking-tighter`) | `-.05em` | `-0.7px` | M |
| `--tracking-tight` (`tracking-tight`) | `-.025em` | `-0.35px` | M |
| `--tracking-normal` (`tracking-normal`) | `0em` | `normal` | M |
| `--tracking-wide` (`tracking-wide`) | `.025em` | `0.35px` | M |
| `--tracking-wider` (`tracking-wider`) | `.05em` | `0.7px` | M |
| `--tracking-widest` (`tracking-widest`) | `.1em` | `1.4px` | M |
| `--leading-tight` (`leading-tight`) | `1.25` | `17.5px` | M |
| `--leading-snug` (`leading-snug`) | `1.375` | `19.25px` | M |
| `--leading-normal` (`leading-normal`) | `1.5` | `21px` | M |
| `--leading-relaxed` (`leading-relaxed`) | `1.625` | `22.75px` | M |
| `leading-none` | `1` | `14px` | M |
| `--font-weight-normal` (`font-normal`) | `400` | – | M |
| `--font-weight-medium` (`font-medium`) | `500` | – | M |
| `--font-weight-semibold` (`font-semibold`) | `600` | – | M |
| `--font-weight-bold` (`font-bold`) | `700` | – | M |

### 3.10 Radius

| Utility | Declared | Resolved | M/S |
|---|---|---|---|
| `rounded-xs` | `.125rem` | `2px` | M |
| `rounded-sm` | `calc(var(--radius) - 4px)` | `4px` | M |
| `rounded` | `.25rem (Tailwind default)` | `4px` | M |
| `rounded-md` | `calc(var(--radius) - 2px)` | `6px` | M |
| `rounded-lg` | `var(--radius)` | `8px` | M |
| `rounded-xl` | `calc(var(--radius) + 4px)` | `12px` | M |
| `rounded-2xl` | `calc(var(--radius) + 8px)` | `16px` | M |
| `rounded-3xl` | `calc(var(--radius) + 12px)` | `20px` | M |
| `rounded-full` | `calc(infinity * 1px)` | `3.35544e+07px` (Chrome clamps any infinite length to its largest layout value, about 2^25 px; render as a pill, radius = min(w, h) / 2) | M |
| control | `var(--control-radius) = .5rem` | `8px` | M |
| scrollbar-thumb | `3px (index.css:1861)` | `3px` | S |
| markdown-pre | `0.5rem (index.css:2149)` | `8px` | S |
| markdown-inline-code | `0.25rem (index.css:2132)` | `4px` | S |
| composer-chip-selected | `6px (index.css:2486)` | `6px` | S |

`--radius` is `0.5rem`, so every radius scales with the interface size **except the px offsets**. At a 20px root, `rounded-sm` = 10 − 4 = 6px.

### 3.11 Shadows

Every utility's computed `box-shadow` starts with four `transparent 0 0 0 0` ring and inset layers. They are omitted below.

| Utility / token | Resolved box-shadow (offset-x offset-y blur spread colour) | M/S |
|---|---|---|
| `xs` | `0px 1px 2px 0px #0000000d` | M |
| `sm` | `0px 1px 3px 0px #0000001a, 0px 1px 2px -1px #0000001a` | M |
| `DEFAULT` | `0px 1px 3px 0px #0000001a, 0px 1px 2px -1px #0000001a` | M |
| `md` | `0px 4px 6px -1px #0000001a, 0px 2px 4px -2px #0000001a` | M |
| `lg` | `0px 10px 15px -3px #0000001a, 0px 4px 6px -4px #0000001a` | M |
| `xl` | `0px 20px 25px -5px #0000001a, 0px 8px 10px -6px #0000001a` | M |
| `2xl` | `0px 25px 50px -12px #00000040` | M |
| `xs/5` | `0px 1px 2px 0px #0000000d` | M |
| `sm/5` | `0px 1px 3px 0px #0000000d, 0px 1px 2px -1px #0000000d` | M |
| `md/5` | `0px 4px 6px -1px #0000000d, 0px 2px 4px -2px #0000000d` | M |
| `md/10` | `0px 4px 6px -1px #0000001a, 0px 2px 4px -2px #0000001a` | M |
| `lg/5` | `0px 10px 15px -3px #0000000d, 0px 4px 6px -4px #0000000d` | M |
| `lg/20` | `0px 10px 15px -3px #00000033, 0px 4px 6px -4px #00000033` | M |
| `2xl/35` | `0px 25px 50px -12px #00000059` | M |
| `2xl/40 (sm:)` | `0px 25px 50px -12px #00000066` | M |
| `composer` | `0px 1px 2px 0px #0000000d` | M |
| `composer-dark` | `0px 14px 32px -18px #000000bf` | M |
| `inset-2xs` | `inset 0px 1px 0px 0px #0000000d` | M |
| `inset-control-highlight (outer, no inset keyword)` | `0px 1px 0px 0px #ffffff29` | M |
| `inset-control-pressed (outer, no inset keyword)` | `0px 1px 0px 0px #00000014` | S |
| `dialog-glass light` | `0px 24px 64px -24px #000000a6` | M |
| `dialog-glass dark` | `inset 0px 1px 0px 0px #ffffff0a, 0px 24px 72px -20px #000000e6` | M |
| `drop-sm (filter)` | `0 1px 2px #00000026` | M |
| `drop-md (filter)` | `0 3px 3px #0000001f` | M |

- `inset-shadow-control-highlight` and `inset-shadow-control-pressed` are declared without the `inset` keyword (index.css:193–194). **They paint as outer 1px shadows below the element.** Measured: `#ffffff29 0px 1px 0px 0px`, no `inset`.
- `dialog-glass` shadows are from the utility (index.css:430–436), measured on a probe.

### 3.12 Effect utilities (probe elements, both themes)

| Utility | Property | Light | Dark |
|---|---|---|---|
| `surface-glass` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `surface-glass` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `dropdown-glass` | `background-color` | `#ffffffd5` | `#111111d5` |
| `dropdown-glass` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `dropdown-glass` | `border-top-width` | `1px` | = |
| `dropdown-glass` | `border-top-color` | `#18181b1a` | `#f5f5f51a` |
| `dialog-glass border` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `dialog-glass border` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `dialog-glass border` | `border-top-color` | `#18181b1a` | `#ffffff14` |
| `dialog-glass border` | `box-shadow` | `#000000a6 0px 24px 64px -24px` | `#ffffff0a 0px 1px 0px 0px inset, #000000e6 0px 24px 72px -20px` |
| `dialog-backdrop` | `background-color` | `#fcfcfc99` | `#000000a3` |
| `dialog-backdrop` | `backdrop-filter` | `blur(4px)` | = |
| `alert-glass` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `alert-glass` | `background-image` | `linear-gradient(transparent, transparent), none` | = |
| `alert-glass` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `alert-glass[data-variant=error]` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `alert-glass[data-variant=error]` | `background-image` | `linear-gradient(#fb2c360a, #fb2c360a), none` | `linear-gradient(#fb414a0a, #fb414a0a), none` |
| `alert-glass[data-variant=error]` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `alert-glass[data-variant=info]` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `alert-glass[data-variant=info]` | `background-image` | `linear-gradient(#2b7fff0a, #2b7fff0a), none` | = |
| `alert-glass[data-variant=info]` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `alert-glass[data-variant=success]` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `alert-glass[data-variant=success]` | `background-image` | `linear-gradient(#00bc7d0a, #00bc7d0a), none` | = |
| `alert-glass[data-variant=success]` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `alert-glass[data-variant=warning]` | `background-color` | `#fcfcfccc` | `#000000cc` |
| `alert-glass[data-variant=warning]` | `background-image` | `linear-gradient(#fe9a000a, #fe9a000a), none` | = |
| `alert-glass[data-variant=warning]` | `backdrop-filter` | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| `topbar-scroll-fade` | `mask-image` | `linear-gradient(transparent 0%, #0000001a 10%, #0000004d 24%, #00000094 42%, #000000d1 62%, #000000f5 82%, #000000 100%), linear-gradient(#000000, #0…` | = |
| `topbar-scroll-fade` | `mask-size` | `100% 25px, 100% calc(100% - 24px), 6px 100%` | = |
| `topbar-scroll-fade` | `mask-position` | `50% 0%, 50% 100%, 100% 50%` | = |
| `virtualized-scroll-fade` | `mask-size` | `calc(100% - 6px) 100%` | = |
| `virtualized-scroll-fade` | `mask-position` | `0px 50%` | = |
| `sidebar-stage-backdrop` | `mask-image` | `linear-gradient(#000000 0%, #000000 55%, transparent 92%)` | = |
| `sidebar-stage-backdrop` (inside `[data-app-sidebar]`, its only use: SidebarStageBackdrop.tsx:56) | `::after background-image` | `linear-gradient(transparent 0%, transparent 28%, #fafafa1a 40%, #fafafa4d 52%, #fafafa94 64%, #fafafad1 75%, #fafafaf5 85%, #fafafa 93%)` | `linear-gradient(transparent 0%, transparent 28%, #0000001a 40%, #0000004d 52%, #00000094 64%, #000000d1 75%, #000000f5 85%, #000000 93%)` |
| `sidebar-stage-backdrop` (inside `[data-app-sidebar]`) | `--stage-fade` | `#fafafa` | `#000000` (the sidebar's `--sidebar-stage-fade: var(--card)` = `#000`) |
| `sidebar-stage-backdrop` (probe at `:root`, not used there) | `--stage-fade` | `#fafafa` | `#111111` |
| `live-tool-shine` | `color` | `transparent` | = |
| `live-tool-shine` | `background-image` | `linear-gradient(to right, transparent, #18181b, transparent), linear-gradient(#71717b, #71717b)` | `linear-gradient(to right, transparent, #f5f5f5, transparent), linear-gradient(#a1a1a1, #a1a1a1)` |
| `live-tool-shine` | `background-size` | `72px 100%, 100% 100%` | = |
| `live-tool-shine` | `background-position` | `calc(0% - 72px) 0px, 0px 0px` | = |
| `live-tool-shine` | `animation-name` | `live-tool-shine` | = |
| `live-tool-shine` | `animation-duration` | `2.2s` | = |
| `live-tool-shine` | `animation-timing-function` | `steps(30)` | = |
| `live-tool-shine` | `animation-play-state` | `paused` | = |
| `live-activity-focus` | `left` | `-72px` | = |
| `live-activity-focus` | `width` | `calc(100% + 9rem)`: 544px in a 400px probe parent | = |
| `live-activity-focus` | `mask-image` | `linear-gradient(to right, transparent 0px, #0000001f 10.8px, #0000008c 25.2px, #000000 36px, #0000008c 46.8px, #0000001f 61.2px, transparent 72px, tr…` | = |
| `live-activity-focus` | `animation-name` | `live-activity-focus` | = |
| `live-activity-focus` | `animation-duration` | `2.2s` | = |
| `live-activity-focus` | `animation-timing-function` | `steps(30)` | = |
| `live-activity-focus live-activity-focus-aligned` | `width` | `calc(100% - 9rem)`: 256px in a 400px probe parent | = |
| `live-activity-focus live-activity-focus-aligned` | `margin-left` | `72px` | = |
| `surface-grain` | `background-image` | `url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' base…` | = |
| `surface-grain` | `background-size` | `256px 256px` | = |

Formulas (index.css:364–549):

- `surface-glass` = `color-mix(in srgb, var(--background) var(--glass-opacity), transparent)` with `blur(var(--glass-blur)) saturate(var(--glass-saturation))`.
- `dropdown-glass` = `color-mix(in srgb, var(--popover) 18%, color-mix(in srgb, var(--popover) var(--glass-opacity), transparent))`, an alpha of 0.18 + 0.82 × opacity (0.836 → `d5`), plus `border: 1px solid color-mix(in srgb, var(--contrast-foreground) 10%, transparent)`.
- `dialog-glass` is the surface-glass fill plus a border colour (foreground 10% in light, white 8% in dark) and the shadows above. It needs a `border` class for width.
- `dialog-backdrop` = background at 60% (light) / 64% (dark) with `blur(4px)`.
- `alert-glass` stacks `--alert-glass-tint` at 4% over the surface-glass fill. `[data-variant=error|info|success|warning]` picks `--destructive|--info|--success|--warning`.
- `sidebar-stage-backdrop` sets `--stage-fade: var(--sidebar-stage-fade, var(--app-chrome-background))`.
- `virtualized-scroll-fade` needs `--virtualized-scroll-fade-mask`, which scroll-area.tsx:12–18 sets to `linear-gradient(to bottom, transparent, black var(--fade-size), black calc(100% - var(--fade-size)), transparent)` or a one-sided version. The rightmost `--app-scrollbar-width` (6px) lane stays opaque.
- `topbar-scroll-fade` masks the top `--workspace-titlebar-scroll-fade-height + 1px` (25px) with the stepped ramp shown, keeps the rest opaque, and keeps the 6px scrollbar lane opaque.
- Without backdrop-filter support every glass falls back to an opaque `--background` (or `--popover` for dialogs and dropdowns).

### 3.13 Base element defaults (`@layer base` and preflight)

| Element / pseudo | Property | Light | Dark | Source | M/S |
|---|---|---|---|---|---|
| every element `*` | `border-color` | `#e4e4e7` (`--contrast-border`) | `#ffffff14` | index.css:714–716 | M |
| every element `*` | `border-style` / `border-width` | `solid` / `0` | = | preflight | M |
| `::before`, `::after` | `border-color` | **`currentColor`** (the `*` rule does not reach pseudo-elements) | = | preflight | M |
| every element `*` | `outline-color` | `#1b4ed880` (ring at 50%, OKLab) | `#346bf180` | index.css:714–716 | M |
| every element, `::before`, `::after` | `outline-offset` | `-2px` | = | index.css:718–722 | M |
| every element | `outline-style` | `none` (nothing shows until a component sets it) | = | preflight | M |
| menu, select and popover popups | focus outline / ring | removed | = | index.css:723–730 | S |
| `html`, `body` | `background-color` | inline, sampled from `main[data-slot=sidebar-inset]`: `#fcfcfc` | `#000000` | useTheme.ts:279–304 | M |
| `body` | `color` | `#262626` | `#f5f5f5` | index.html:398–408 | M |
| `body` | `background-image` | grain, 256px tile | **`none`**: the boot style's `html.dark body { background: #000000 }` resets it (§3.7) | index.css:1781–1785; index.html:405–408 | M |
| `:root` | `color-scheme` | `light` | `dark` | index.css:1114, 1188 | M |
| `::selection` | colours | **no global app rule**: browser/system default. Linux Chrome paints `#3367d1` over `#fcfcfc` and `#00349e` over `#000000` (consistent with ≈ rgba(0, 65, 198, 0.8)), text `#ffffff`. Exceptions via utilities: `selection:bg-primary selection:text-primary-foreground` (`--primary` on white) in DiffCommentAnnotation.tsx:83, 126 and ThemeImportDialog.tsx:121; `selection:bg-primary/30 selection:text-transparent` (primary 30%, OKLab) in WordDocument.tsx:156 | | (none) | M (default), S (exceptions) |
| `::placeholder` | `color` | `color-mix(in oklab, currentcolor 50%, transparent)` unless a component sets `placeholder:*` | = | preflight | S |
| native scrollbar | width / thumb / radius / track | `6px` / `#d9d9d9` / `3px` / transparent | `6px` / `#ffffff14` (paints `#141414` on black) / `3px` / transparent | index.css:1851–1866 | M (painted) |
| native scrollbar | layout | **takes layout space**: a vertical bar shrinks `clientWidth` by 6px (not an overlay) | = | index.css:1851–1853 | M |
| native horizontal scrollbar | height | **15px**: `::-webkit-scrollbar` sets only `width`, so a horizontal bar keeps Chrome's default 15px thickness (thumb `#d9d9d9`, radius 3px) | 15px, thumb `#ffffff14` | index.css:1851–1853 | M (painted) |
| native scrollbar corner | `::-webkit-scrollbar-corner` | unstyled: paints `#ffffff` where a vertical and a horizontal bar meet | `#ffffff` (a visible 6×15px white block on black) | (none) | M (painted) |
| native scrollbar thumb | `:hover` | `#bfbfbf` | `#ffffff1f` | index.css:1864–1866 | S |
| `.chat-markdown pre` | scrollbar | `scrollbar-width: thin; scrollbar-color: color-mix(in srgb, var(--contrast-border) 78%, transparent) transparent` (thumb `#e4e4e7c7`). Because these standard properties are set, Chromium ≥121 **ignores the element's `::-webkit-scrollbar*` rules** (the declared `height: 6px` and `999px` radius never apply). Linux Chrome drew a standard thin bar: 10px tall, rounded thumb, arrow buttons, transparent track. macOS draws its own thin system scrollbar | thumb `#ffffff10` | index.css:2145–2169 | M |
| `.chat-markdown-table-container` horizontal `[data-slot=scroll-area-scrollbar]` | thumb | `opacity: 1` (always shown); `--app-scrollbar-thumb` muted-foreground 60% `#71717b99`, hover 80% `#71717bcc` | `#a1a1a199`, hover `#a1a1a1cc` | index.css:2276–2282 | P |
| `hr` in `.chat-markdown` / `.composer-tiptap` | `border-top` | 1px (preflight) in `#e4e4e7` (`--contrast-border`, from the `*` rule) | 1px in `--contrast-muted-foreground` `#a1a1a1` | preflight; index.css:2076–2080 | P |
| `svg.lucide[stroke-width="2"]` | `stroke-width` | `1.75` | = | index.css:1728–1732 | M |
| `pre`, `code` | `font-family` | `var(--font-mono)` | = | index.css:1787–1790 | S |
| `#root` | box | `height/width 100%`, `overflow: clip hidden`, `padding-top: max(env(safe-area-inset-top), 0px)`, `padding-right: var(--desktop-window-right-resize-inset)` | = | index.css:1745–1755 | S |
| `html`, `body` | size | `min-height: calc(100svh + env(safe-area-inset-top))`, `overscroll-behavior: none`. Body: `height: 100%; overflow: hidden` | = | index.css:1734–1743 | M |

Measured scrollbar: a 120×200 probe with 800px content painted a 6×50px thumb (length ∝ viewport/content) with rounded ends, flush to the right edge, no arrows and no track paint. A 120×120 probe overflowing both ways measured `offsetWidth − clientWidth = 6` and `offsetHeight − clientHeight = 15`. It painted a 22×15px horizontal thumb with 3px corners and a white 6×15px corner, in both themes.

### 3.14 Other colours declared in index.css rules

| Use | Declared | Light | Dark | Source | M/S |
|---|---|---|---|---|---|
| `::highlight(t3-thread-find)` text | light `oklch(0.216164 0.028283 77.423)`, dark `oklch(0.970151 0 0)` (each behind `var(--app-theme-search-match-foreground, …)`) | `#21180a` | `#f5f5f5` | index.css:407-415 | P |
| `::highlight(t3-thread-find)` background | light `oklch(0.915 0.11 85)`, dark `oklch(0.55 0.1 85)` (behind `var(--app-theme-search-match-background, …)`) | `#ffde8b` | `#8c6c1f` | index.css:409,413 | P |
| `::highlight(t3-thread-find-active)` text | `oklch(0.216164 0.028283 77.423)` | `#21180a` | `#21180a` | index.css:417-423 | P |
| `::highlight(t3-thread-find-active)` background | `oklch(0.774202 0.151946 74.331)` | `#eea52b` | `#eea52b` | index.css:419-422 | P |
| `::highlight(t3-assistant-citation-comment)` | `color-mix(in oklab, var(--primary) 45%, transparent)` | `#1b4ed873` | `#346bf173` | index.css:676-678 | P |
| default `outline-color` of every element | `color-mix(in oklab, var(--ring) 50%, transparent)` | `#1b4ed880` | `#346bf180` | index.css:714-716 | P |
| settings search target pulse ring | `color-mix(in oklab, var(--color-primary) 70%, transparent)` | `#1b4ed8b3` | `#346bf1b3` | index.css:929 | P |
| `.settings-slider` track fill | `var(--primary)` | `#1b4ed8` | `#346bf1` | index.css:978-983 | P |
| `.settings-slider` track rest | `color-mix(in srgb, var(--contrast-muted-foreground) 22%, transparent)` | `#71717b38` | `#a1a1a138` | index.css:981 | P |
| `.settings-slider` track inset ring | `color-mix(in srgb, var(--contrast-border) 55%, transparent)` | `#e4e4e78c` | `#ffffff0b` | index.css:984 | P |
| `.settings-slider` thumb shadow | `color-mix(in srgb, var(--contrast-foreground) 12%, transparent)` | `#18181b1f` | `#f5f5f51f` | index.css:1008 | P |
| `.settings-slider` thumb shadow, hover | `color-mix(in srgb, var(--contrast-foreground) 20%, transparent)` | `#18181b33` | `#f5f5f533` | index.css:1028 | P |
| `.chat-markdown :not(pre) > code` background | `color-mix(in srgb, var(--contrast-foreground) 10%, transparent)` | `#18181b1a` | `#f5f5f51a` | index.css:2135 | P |
| `.chat-markdown-codeblock-header` text | `color-mix(in srgb, var(--contrast-foreground) 72%, transparent)` | `#18181bb8` | `#f5f5f5b8` | index.css:2172 | P |
| `.chat-markdown pre` scrollbar thumb | `color-mix(in srgb, var(--contrast-border) 78%, transparent)` | `#e4e4e7c7` | `#ffffff10` | index.css:2153,2168 | P |
| `.chat-markdown` table row rules | `color-mix(in srgb, var(--contrast-border) 60%, transparent)` | `#e4e4e799` | `#ffffff0c` | index.css:2255,2261 | P |
| `.chat-markdown a` (not file links/chips) | `var(--info-foreground)` | `#1447e6` | `#50a2ff` | index.css:2043-2045 | P |
| blockquote rule (shown with foreground text) | `color-mix(in srgb, currentcolor 35%, transparent)` | `#18181b59` | `#f5f5f559` | index.css:2066 | P |
| `.composer-tiptap :not(pre) > code` background | `color-mix(in oklab, var(--muted-foreground) 12%, transparent)` | `#71717b1f` | `#a1a1a11f` | index.css:2463 | P |
| `.composer-rich-marker` (opacity .75) | `var(--muted-foreground)` | `#71717b` | `#a1a1a1` | index.css:2469-2475 | P |
| `.pele` diagram border | `color-mix(in srgb, var(--foreground) 28%, transparent)` | `#18181b47` | `#f5f5f547` | index.css:1822 | P |
| theme inspector spotlight dimmer | `rgb(2 3 8 / 54%)` | `#0203088a` | `#0203088a` | index.css:1416 | P |
| theme inspector glow fill | `color-mix(in oklab, var(--ring) 9%, transparent)` | `#1b4ed817` | `#346bf117` | index.css:1420 | P |
| theme inspector glow stroke | `color-mix(in oklab, var(--ring) 72%, white)` | `#5885e7` | `#6b98f9` | index.css:1421 | P |
| theme inspector hover box fill | `color-mix(in oklab, var(--ring) 7%, transparent)` | `#1b4ed812` | `#346bf112` | index.css:1431 | P |
| theme inspector hover box border (1px) | `color-mix(in oklab, var(--ring) 70%, white)` | `#5c88e8` | `#6f9bf9` | index.css:1432 | P |
| theme inspector hover box shadow | `0 0 0 3px` ring 16% + `0 0 18px` ring 38% | `#1b4ed829`, `#1b4ed861` | `#346bf129`, `#346bf161` | index.css:1433–1435 | P |
| theme inspector hover label border | `color-mix(in oklab, var(--ring) 48%, var(--contrast-border))` | `#81a1e5` | `#4479f385` | index.css:1444 | P |

### 3.15 Composer scope (`[data-slot=composer-shell]`)

| Variable | Light | Dark | Declared light | Declared dark | M/S |
|---|---|---|---|---|---|
| `--chat-composer-drawer-inset` | `22px` | `22px` | `1.375rem` | `1.375rem` | M |
| `--chat-composer-glass-surface` | `#ffffff` | `#111111` | `var(--card)` | `var(--surface-raised)` | M |
| `--chat-composer-outline` | `#00000014` | `#ffffff0d` | `rgb(0 0 0 / 8%)` | `color-mix(in srgb, var(--color-white) 5%, transparent)` | M |
| `--chat-composer-highlight` | (unset) | `#ffffff08` | `(not declared in light)` | `rgb(255 255 255 / 3%)` | M |
| (glass fill, painted on `::before`; **not a custom property**) | `#ffffffcc` | `#111111cc` | `before:bg-(--chat-composer-glass-surface)/(--glass-opacity)` = `color-mix(in oklab, var(--chat-composer-glass-surface) var(--glass-opacity), transparent)` | same | M |

- The composer's glass is painted on a `::before` (`rounded-2xl`, 16px): `bg-(--chat-composer-glass-surface)/(--glass-opacity)` plus `backdrop-blur-(--glass-blur) backdrop-saturate-(--glass-saturation)` (ComposerSurface.tsx:21). The shell's `::before` paints it normally. With a context row below (`data-context-row`, the measured state), the shell's `::before` is hidden and `[data-slot=composer-host]`'s own `::before` paints the same fill (ComposerSurface.tsx:24–25, 42). With an attached banner (`data-composer-banner-surface=attached`), both are hidden and the main surface paints it (ComposerSurface.tsx:23, 44, 60). Without backdrop-filter support, the fill is the opaque `--chat-composer-glass-surface` (ComposerSurface.tsx:22).
- Themed overrides on the shell (ComposerSurface.tsx:18–20, not measured): under `html[data-theme-id]`, glass surface `var(--app-theme-surface-raised)` and outline `var(--app-theme-toolbar-border)`. Dark themes other than T3 Chat use highlight `color-mix(in srgb, var(--app-theme-input) 12%, transparent)` and outline `color-mix(in srgb, var(--app-theme-input) 30%, var(--background))`. Dark T3 Chat uses highlight `color-mix(in srgb, #432d48 12%, transparent)` and outline `#241e28`.
- The outline is on `[data-slot=composer-host]::after`: `after:border after:border-(--chat-composer-outline)`, 1px, radius inherited (16px). In dark it adds `inset-shadow-2xs` coloured `--chat-composer-highlight`. Measured: light `1px #00000014`, dark `1px #ffffff0d` plus `inset 0 1px #ffffff08` (ComposerSurface.tsx:33–34, 43). The main surface carries the same classes, but its `::after` is hidden unless a banner is attached (ComposerSurface.tsx:58–59).
- Light shadow is `shadow-composer` (`0 1px 2px #0000000d`) on `[data-slot=composer-host]` (measured). Dark uses `shadow-none` on the host (live class list), with `dark:shadow-composer-dark` (`0 14px 32px -18px #000000bf`) on attached surfaces.
- `--chat-composer-attached-*` and `--chat-composer-attachment-overlap` exist only while a banner is attached (ComposerBanner.tsx). They were not present in the measured state, so `--background-image-composer-seam-above` is invalid here.
- `--background-image-composer-seam-below` resolves to `linear-gradient(transparent 0 16px, #0000002e 16px, transparent 26px), linear-gradient(#ffffff03, #ffffff03)` (P).

### 3.16 Context-chip accents (inline on `<html>`)

Solved at runtime per appearance (themePalette.ts:694–739). Seeds are in lib/contextChipAccents.ts:1–25. In light they are solved darker, in dark lighter, to at least 5:1 on canvas, surface and raised surface (§5.5).

| Variable | Seed (contextChipAccents.ts) | Light | Dark | M/S |
|---|---|---|---|---|
| `--context-chip-image` | `oklch(0.62 0.136 237)` | `#006a99` | `#0f96d3` | M |
| `--context-chip-video` | `oklch(0.62 0.16 48)` | `#a54800` | `#dd6e29` | M |
| `--context-chip-file` | `oklch(0.62 0.136 237)` | `#006a99` | `#0f96d3` | M |
| `--context-chip-mention` | `oklch(0.62 0.11 215)` | `#006d80` | `#0d9ab4` | M |
| `--context-chip-terminal` | `oklch(0.62 0.134 163)` | `#00724f` | `#08a170` | M |
| `--context-chip-element` | `oklch(0.62 0.134 70)` | `#8c5800` | `#c27f19` | M |
| `--context-chip-preview-annotation` | `oklch(0.62 0.134 70)` | `#8c5800` | `#c27f19` | M |
| `--context-chip-review-comment` | `oklch(0.62 0.16 292)` | `#6c51ba` | `#957bea` | M |
| `--context-chip-pull-request` | `oklch(0.62 0.16 277)` | `#5358c0` | `#7983ef` | M |
| `--context-chip-pr-open` | `oklch(0.62 0.134 163)` | `#00724f` | `#08a170` | M |
| `--context-chip-pr-draft` | `oklch(0.62 0.02 259)` | `#5d6470` | `#868e99` | M |
| `--context-chip-pr-merged` | `oklch(0.62 0.16 292)` | `#6c51ba` | `#957bea` | M |
| `--context-chip-pr-closed` | `oklch(0.62 0.16 16)` | `#b23649` | `#e46371` | M |
| `--context-chip-skill` | `oklch(0.62 0.16 322)` | `#92439f` | `#bf6ecc` | M |
| `--context-chip-model` | `oklch(0.62 0.13 105)` | `#6c6500` | `#989016` | M |
| `--context-chip-thread` | `oklch(0.62 0.12 190)` | `#006f6b` | `#009e97` | M |
| `--context-chip-environment` | `oklch(0.62 0.11 145)` | `#327037` | `#5c9b5e` | M |
| `--context-chip-citation` | `oklch(0.62 0.16 259)` | `#2361be` | `#4e8cee` | M |
| `--context-chip-doc-pdf` | `oklch(0.62 0.16 24)` | `#b33739` | `#e46561` | M |
| `--context-chip-doc-word` | `oklch(0.62 0.136 248)` | `#0066ab` | `#3e92da` | M |
| `--context-chip-doc-spreadsheet` | `oklch(0.62 0.134 148)` | `#11732f` | `#479e58` | M |
| `--context-chip-doc-slides` | `oklch(0.62 0.134 56)` | `#9b4f00` | `#cd7833` | M |
| `--context-chip-doc-epub` | `oklch(0.62 0.16 291)` | `#6b51bb` | `#937cea` | M |
| `--context-chip-invalid` | `errorForeground role` | `#c10007` | `#ff6467` | M |

Chips paint text in `--context-chip-ink` = `var(--context-chip-accent)` = `var(--context-chip-<kind>)`, with `[&_svg]:size-[1em]` and `gap-[0.2em]` (ContextChip.tsx:9–35). In user messages: `--context-chip-ink: var(--context-chip-message, var(--context-chip-accent))` (TimelineUserMessageRow.tsx:102). `--context-chip-message` is unset for the standard look.

### 3.17 Sidebar stage artwork (`@layer base` and `components`)

| Token | Light | Dark | index.css | M/S |
|---|---|---|---|---|
| `--stage-art-top` | `#67c2ff` | `#3a7ad1` | 682 / 708 | M |
| `--stage-art-mid` | `#347ff8` | `#2050ae` | 683 / 709 | M |
| `--stage-art-bottom` | `#1538d0` | `#101f6e` | 684 / 710 | M |
| `--stage-art-highlight` | `#d4f6ff` | `#d4f6ff` | 685 | M |
| `--stage-art-secondary` | `#65c8ff` | `#65c8ff` | 686 | M |
| `--stage-art-tertiary` | `#7c8bff` | `#7c8bff` | 687 | M |
| `--stage-art-line` | `#ddf7ff` | `#ddf7ff` | 688 | M |
| `--stage-art-celeste-highlight` | `#d2ffff` | `#d2ffff` | 689 | M |
| `--stage-art-celeste-secondary` | `#48dcf5` | `#48dcf5` | 690 | M |
| `--stage-art-violet-highlight` | `#d9d8ff` | `#d9d8ff` | 691 | M |
| `--stage-art-grid-line` | `#eaf6ff` | `#eaf6ff` | 692 | M |
| `--stage-night-base-top` | `#32155b` | `#32155b` | 693 | M |
| `--stage-night-base-mid` | `#151443` | `#151443` | 694 | M |
| `--stage-night-base-bottom` | `#07152f` | `#07152f` | 695 | M |
| `--stage-night-top` | `#32155b` | `#32155b` | 696 | M |
| `--stage-night-mid` | `#151443` | `#151443` | 697 | M |
| `--stage-night-bottom` | `#07152f` | `#07152f` | 698 | M |
| `--stage-night-highlight` | `#4ea4ff` | `#4ea4ff` | 699 | M |
| `--stage-night-secondary` | `#696fea` | `#696fea` | 700 | M |
| `--stage-night-tertiary` | `#a85bea` | `#a85bea` | 701 | M |
| `--stage-night-line` | `#e4eaff` | `#e4eaff` | 702 | M |
| `--stage-night-glow-highlight` | `#5165d8` | `#5165d8` | 703 | M |
| `--stage-night-glow-secondary` | `#283075` | `#283075` | 704 | M |
| `--stage-night-sparkle` | `#c8d7ff` | `#c8d7ff` | 705 | M |

The default (blue/violet "T3 Code" artwork) palette is index.css:679–712. Built-in themes swap these pigments (index.css:757–921): t3-chat rose, grove green, ocean marine, ember copper, iris violet.

### 3.18 index.css component rules (source values)

| Rule | Values | index.css |
|---|---|---|
| `.chat-content-lane` | `margin-inline: auto; width: 100%; min-width: 0; max-width: var(--chat-content-max-width)` (736px) | 952–957 |
| `.chat-composer-lane` | `padding-inline: calc(env(safe-area-inset-*) + 0.75rem + var(--chat-lane-inset-*, 0px))`; ≥640px: `3rem` | 2551–2578 |
| `[data-chat-canvas] .messages-timeline-scroll` | `padding-inline: calc(0.75rem + inset - var(--chat-timeline-gutter, 0px))`; ≥640px: `3rem` | 2559–2586 |
| `.chat-markdown` block spacing | first child `margin-top 0`, last child `margin-bottom 0`. `p, ul, ol, blockquote, pre, table container`: `margin 0.65rem 0` (10.4px) | 1870–1887 |
| headings (chat and composer) | `margin 1.25rem 0 0.5rem; font-weight 600; line-height 1.3; color --contrast-foreground`. h1 20px, h2 18px, h3 16px, h4–h6 14px, h6 muted | 1893–1925 |
| lists | `ul`: gutter/padding 1.25rem, disc → circle → square. `ol`: `padding-inline-start: max(1.25rem, var(--list-marker-sans))`, decimal → lower-alpha → lower-roman (user messages stay decimal), `::marker` tabular-nums. `li + li` margin-top 0.25rem. Task items: no marker, checkbox `margin-inline: -gutter 0.35em` | 1927–2030 |
| composer lists | `ol` gutter `max(1.25rem, 3ch)` rising with `data-marker-width` 3…10 to `11ch`. `ul` `max(1.25rem, 2ch)`. Task lists flex, gap 0.5rem, checkbox 0.95rem, nested indent 1.375rem | 1951–1995, 2507–2548 |
| links | no underline, colour `--info-foreground` (`#1447e6` / `#50a2ff`). Hover/focus: dotted underline `radial-gradient(circle, currentcolor 0.75px, transparent 1px)` at left bottom, `4px 2px`, repeat-x | 2038–2063 |
| blockquote | `border-inline-start: 2px solid color-mix(in srgb, currentcolor 35%, transparent); padding-inline-start: 0.8rem` | 2065–2073 |
| inline code (chat) | `border 1px transparent; radius 0.25rem; background foreground 10%; padding 0.1em 0.35em; font-size 0.85em; colour --contrast-foreground` | 2130–2139 |
| inline code (composer) | mono, `0.92em`, background muted-foreground 12% (OKLab), radius 0.3em, padding 0.05em 0.25em | 2460–2466 |
| `pre` | `border 1px --contrast-border; radius 0.5rem; background var(--muted); padding 0.8rem 0.9rem` (12.8px 14.4px); code block `pre` inside `.chat-markdown-codeblock` has no border, radius or background. Code font size `var(--font-size-code, inherit)` | 1797–1801, 2145–2203 |
| tables | width 100%, collapse, `font-size 0.875rem`. Cells `padding 0.45rem 0.75rem`, top/start, outer cells flush. Head `padding-block 0.55rem`, 600. Row rules `1px color-mix(--contrast-border 60%)`. Max cell width 24rem | 2227–2282 |
| footnotes | `margin-top 1.25rem; border-top 1px --contrast-border; padding-top 0.75rem; muted; 0.75rem`. Refs `inline-flex; min-width 1rem; radius 0.25rem; 0.6875rem/600` | 2102–2128 |
| `strong` | `700` | 2141–2143 |
| composer marker | `.composer-rich-marker`: muted-foreground, opacity 0.75, weight 400 | 2469–2475 |
| composer chip selection | `.ProseMirror-selectednode`: `outline 2px solid var(--ring); offset -2px; radius 6px`. Range selection `::after` `Highlight` at 0.3 | 2483–2501 |
| `.settings-slider` | height 1.5rem. Track 0.375rem pill: primary fill to `progress + offset`, rest muted-foreground 22%, inset 1px ring border 55%. Thumb 1rem, `margin-top -0.3125rem`, `2px solid --primary`, fill `--background`, shadow `0 1px 2px` foreground 12% | 963–1076 |
| settings search target | `animation: settings-search-target-pulse 650ms ease-in-out 2; border-radius 0.75rem` | 923–943 |
| inline preview subheader | `height/min-height calc(var(--spacing) * 7)` = 28px, `margin-bottom` 12px, transparent bottom border | 945–950 |
| ultrathink | `.ultrathink-frame::before`: 2px padding ring masked (`xor`/`exclude`) with `linear-gradient(120deg, #ff6b6b 0%, #f59e0b 18%, #22c55e 36%, #14b8a6 54%, #3b82f6 72%, #ec4899 90%, #ff6b6b 100%)`, `background-size 220% 220%`, `filter: saturate(0.82) brightness(0.92)`. `.ultrathink-chroma { filter: saturate(1.2) }` | 2345–2386 |
| diff surfaces | `diffs-container` background/text `--code-background`/`--code-foreground` (forced). Search panel: `--diffs-search-panel-bg: var(--popover)`, fg popover-foreground, modified `--primary`, top inset 40px | 2316–2343 |
| `.pele` diagrams | `--pele-bg/fg/muted/line/surface/surface-alt/border/accent/critical` (§3.14) | 1815–1825 |
| F6 keyboard focus | `[data-workspace-keyboard-focus]:focus { outline: 2px solid var(--ring); outline-offset: -2px }`. `[data-workspace-area]:focus-visible { outline: none }` | 2590–2598 |
| drag regions | `.drag-region { -webkit-app-region: drag }`. Children `initial`. `button, input, textarea, select, a` → `no-drag` | 1828–1848 |
| diff copy-path button | on hover-capable pointers (`@media (hover: hover)`), `[data-diff-copy-path]` is `opacity: 0` unless its `diffs-container` is hovered or the button holds focus (`:focus-within`) | 1311–1316 |
| provider detail groups | `[data-provider-detail] [data-slot=settings-group]`: border transparent, background transparent, no shadow (flat sections) | 1303–1309 |
| animated height | `[data-slot=animated-height] [data-slot=collapsible-panel]`: `height: auto !important; transition: none !important`; `[data-closed]` → `display: none` | 741–751 |
| timeline row frame | `.messages-timeline-row-frame { padding-inline: 0 }` | 959–961 |
| ligatures | `[data-preview-url-input]`: `font-variant-ligatures: none`. `.composer-tiptap.ProseMirror`: `font-variant-ligatures: normal; font-feature-settings: normal` (undoes Tiptap's injected CSS) | 1722–1724, 2445–2449 |
| composer focus | `.composer-tiptap:focus`, `.composer-tiptap .ProseMirror:focus`: `outline: none` | 2455–2458 |

### 3.19 Desktop shell (Electron) values

| Value | Light | Dark | Source | M/S |
|---|---|---|---|---|
| `BrowserWindow.backgroundColor` (before first paint, and on theme sync) | `#ffffff` | `#000000` | DesktopWindow.ts:164–166, 344–359, 483 | S |
| macOS title bar | `titleBarStyle: "hiddenInset"`, traffic lights at `x: 16, y: 52/2 − 7 = 19` (re-centred for zoom) | = | DesktopWindow.ts:56–58, 75–84, 322–330 | S |
| Windows/Linux title bar | `titleBarStyle: "hidden"`, overlay `color: #01000000`, `height: 40`, `symbolColor: #1f2937` | `symbolColor: #f8fafc` | DesktopWindow.ts:46, 86–88, 333–341 | S |
| Window size | default 1100×780, minimum 400×360 | = | DesktopAppSettings.ts:53–67 | S |
| `nativeTheme.themeSource` | from the web theme (`light`/`dark`/`system`); Linux resolves `system` from the settings portal | | ElectronTheme.ts:38–47, useTheme.ts:350–392 | S |
| macOS controls inset | `--workspace-controls-left: var(--desktop-window-controls-inset, 90px)` when not fullscreen | | AppSidebarLayout.tsx:69, 446–455 | S |
| Windows resize lane | `.electron-windows { --desktop-window-right-resize-inset: 6px }` | | index.css:1757–1759 | S |
| `<meta name="theme-color">` | rewritten to the sampled chrome colour: measured `oklch(0.992 0 0)` (= `#fcfcfc`) | measured `rgb(0, 0, 0)` | useTheme.ts:294–303 | M |

### 3.20 Tailwind palette emitted by the build (`@layer theme`)

All 80 colours that the shipped CSS defines (light and dark are identical). `--color-zinc-25` is T3's own (index.css:222).

| Colour | sRGB | Declared | Colour | sRGB | Declared |
|---|---|---|---|---|---|
| `red-300` | `#ffa2a2` | `oklch(80.8% .114 19.571)` | `red-400` | `#ff6467` | `oklch(70.4% .191 22.216)` |
| `red-500` | `#fb2c36` | `oklch(63.7% .237 25.331)` | `red-600` | `#e7000b` | `oklch(57.7% .245 27.325)` |
| `red-700` | `#c10007` | `oklch(50.5% .213 27.518)` | `orange-400` | `#ff8904` | `oklch(75% .183 55.934)` |
| `orange-500` | `#ff6900` | `oklch(70.5% .213 47.604)` | `orange-600` | `#f54a00` | `oklch(64.6% .222 41.116)` |
| `amber-300` | `#ffd230` | `oklch(87.9% .169 91.605)` | `amber-400` | `#ffb900` | `oklch(82.8% .189 84.429)` |
| `amber-500` | `#fe9a00` | `oklch(76.9% .188 70.08)` | `amber-600` | `#e17100` | `oklch(66.6% .179 58.318)` |
| `amber-700` | `#bb4d00` | `oklch(55.5% .163 48.998)` | `yellow-400` | `#fdc700` | `oklch(85.2% .199 91.936)` |
| `yellow-500` | `#f0b100` | `oklch(79.5% .184 86.047)` | `yellow-600` | `#d08700` | `oklch(68.1% .162 75.834)` |
| `lime-400` | `#9ae600` | `oklch(84.1% .238 128.85)` | `lime-500` | `#7ccf00` | `oklch(76.8% .233 130.85)` |
| `lime-600` | `#5ea500` | `oklch(64.8% .2 131.684)` | `green-400` | `#05df72` | `oklch(79.2% .209 151.711)` |
| `green-500` | `#00c950` | `oklch(72.3% .219 149.579)` | `green-600` | `#00a63e` | `oklch(62.7% .194 149.214)` |
| `emerald-300` | `#5ee9b5` | `oklch(84.5% .143 164.978)` | `emerald-400` | `#00d492` | `oklch(76.5% .177 163.223)` |
| `emerald-500` | `#00bc7d` | `oklch(69.6% .17 162.48)` | `emerald-600` | `#009966` | `oklch(59.6% .145 163.225)` |
| `emerald-700` | `#007a55` | `oklch(50.8% .118 165.612)` | `teal-300` | `#46edd5` | `oklch(85.5% .138 181.071)` |
| `teal-400` | `#00d5be` | `oklch(77.7% .152 181.912)` | `teal-500` | `#00bba7` | `oklch(70.4% .14 182.503)` |
| `teal-600` | `#009689` | `oklch(60% .118 184.704)` | `cyan-400` | `#00d3f3` | `oklch(78.9% .154 211.53)` |
| `cyan-500` | `#00b8db` | `oklch(71.5% .143 215.221)` | `cyan-600` | `#0092b8` | `oklch(60.9% .126 221.723)` |
| `sky-300` | `#74d4ff` | `oklch(82.8% .111 230.318)` | `sky-400` | `#00bcff` | `oklch(74.6% .16 232.661)` |
| `sky-500` | `#00a6f4` | `oklch(68.5% .169 237.323)` | `sky-600` | `#0084d1` | `oklch(58.8% .158 241.966)` |
| `blue-400` | `#50a2ff` | `oklch(70.7% .165 254.624)` | `blue-500` | `#2b7fff` | `oklch(62.3% .214 259.815)` |
| `blue-600` | `#155dfc` | `oklch(54.6% .245 262.881)` | `blue-700` | `#1447e6` | `oklch(48.8% .243 264.376)` |
| `indigo-300` | `#a3b3ff` | `oklch(78.5% .115 274.713)` | `indigo-400` | `#7c86ff` | `oklch(67.3% .182 276.935)` |
| `indigo-500` | `#615fff` | `oklch(58.5% .233 277.117)` | `indigo-600` | `#4f39f6` | `oklch(51.1% .262 276.966)` |
| `violet-300` | `#c4b4ff` | `oklch(81.1% .111 293.571)` | `violet-400` | `#a684ff` | `oklch(70.2% .183 293.541)` |
| `violet-500` | `#8e51ff` | `oklch(60.6% .25 292.717)` | `violet-600` | `#7f22fe` | `oklch(54.1% .281 293.009)` |
| `purple-400` | `#c27aff` | `oklch(71.4% .203 305.504)` | `purple-500` | `#ad46ff` | `oklch(62.7% .265 303.9)` |
| `purple-600` | `#9810fa` | `oklch(55.8% .288 302.321)` | `fuchsia-300` | `#f4a8ff` | `oklch(83.3% .145 321.434)` |
| `fuchsia-400` | `#ed6aff` | `oklch(74% .238 322.16)` | `fuchsia-500` | `#e12afb` | `oklch(66.7% .295 322.15)` |
| `fuchsia-600` | `#c800de` | `oklch(59.1% .293 322.896)` | `pink-400` | `#fb64b6` | `oklch(71.8% .202 349.761)` |
| `pink-500` | `#f6339a` | `oklch(65.6% .241 354.308)` | `pink-600` | `#e60076` | `oklch(59.2% .249 .584)` |
| `rose-400` | `#ff637e` | `oklch(71.2% .194 13.428)` | `rose-500` | `#ff2056` | `oklch(64.5% .246 16.439)` |
| `rose-600` | `#ec003f` | `oklch(58.6% .253 17.585)` | `gray-400` | `#99a1af` | `oklch(70.7% .022 261.325)` |
| `gray-500` | `#6a7282` | `oklch(55.1% .027 264.364)` | `gray-600` | `#4a5565` | `oklch(44.6% .03 256.802)` |
| `zinc-50` | `#fafafa` | `oklch(98.5% 0 none)` | `zinc-100` | `#f4f4f5` | `oklch(96.7% .001 286.375)` |
| `zinc-200` | `#e4e4e7` | `oklch(92% .004 286.32)` | `zinc-300` | `#d4d4d8` | `oklch(87.1% .006 286.286)` |
| `zinc-400` | `#9f9fa9` | `oklch(70.5% .015 286.067)` | `zinc-500` | `#71717b` | `oklch(55.2% .016 285.938)` |
| `zinc-800` | `#27272a` | `oklch(27.4% .006 286.033)` | `zinc-900` | `#18181b` | `oklch(21% .006 285.885)` |
| `neutral-100` | `#f5f5f5` | `oklch(97% 0 none)` | `neutral-400` | `#a1a1a1` | `oklch(70.8% 0 none)` |
| `neutral-950` | `#0a0a0a` | `oklch(14.5% 0 none)` | `black` | `#000000` | `#000` |
| `white` | `#ffffff` | `#fff` | `zinc-25` | `#fcfcfc` | `oklch(99.2% 0 0)` |

Non-colour theme variables emitted (all M): `--spacing: .25rem` (4px); `--container-xs…6xl` = 320, 384, 448, 512, 576, 672, 768, 896, 1024, 1152px; `--blur-xs/sm/md/xl` = 4/8/12/24px; `--drop-shadow-sm: 0 1px 2px #00000026`, `--drop-shadow-md: 0 3px 3px #0000001f`; `--aspect-video: 16 / 9`; `--default-transition-duration: .15s`; `--default-transition-timing-function: cubic-bezier(.4, 0, .2, 1)`; `--default-font-family: var(--font-sans)`; `--default-mono-font-family: var(--font-mono)`.

## 4. States

Token-level states. Component-specific hover and press styling belongs to the component docs.

| State | Tokens / rule | Light | Dark | Source | M/S |
|---|---|---|---|---|---|
| default surface | `--background` / sidebar `--sidebar` | `#fcfcfc` / `#fafafa` | `#000000` / `#000000` (sidebar scope) | §3.1, §3.5 | M |
| hover (rows) | `--sidebar-row-hover` | `#fcfcfc` | sidebar scope `#f1f3f714` (root `#ffffff0a`) | index.css:1175, 1249, 1270 | M |
| active / pressed (rows) | `--sidebar-row-active` | `#ffffff` | sidebar scope `#f1f3f71c` | 1176, 1250, 1271 | M |
| selected (rows) | `--sidebar-row-selected` | `#ffffff` | sidebar scope `#f1f3f712` | 1177, 1251, 1272 | M |
| hover (toolbar control) | `--toolbar-control-hover` | `#f4f4f5` | `#ffffff0a` | 1123 | M |
| hover (message action) | `--message-action-hover` | `#3160db` | `#2f60d9` | 1144 | M |
| hover (generic) | `--accent` | `#f4f4f5` | `#ffffff0a` | 1145, 1204 | M |
| pressed (controls) | `inset-shadow-control-pressed`: outer `0 1px #00000014`. `active:shadow-none` in several places | | | 194 | S |
| raised highlight | `inset-shadow-control-highlight`: outer `0 1px #ffffff29` (enabled, not disabled) | | | 193 | M |
| focus-visible | `--ring` `#1b4ed8` / `#346bf1`. Default outline colour ring/50 `#1b4ed880` / `#346bf180`, offset −2px, style none until set. Keyboard area focus `2px solid --ring`, offset −2px | | | 714–722, 2595–2598 | M |
| focus (slider) | thumb `box-shadow: inset 0 0 0 1px var(--background), inset 0 0 0 3px var(--ring)` | | | 1048–1058 | S |
| hover / active (slider thumb) | `scale(1.08)` + `0 1px 3px` fg 20%; active `scale(0.94)` | | | 1026–1042 | S |
| disabled (themed panel toggles) | `color-mix(in oklab, --contrast-toolbar-foreground 55%, --toolbar-background)` (only with `data-theme-id`) | | | 1494–1504 | S |
| invalid / error | `--destructive` `#fb2c36` / `#fb414a`; `--error-surface` `#fb2c3614` / `#fb414a29`; chip invalid `--context-chip-invalid` `#c10007` / `#ff6467` | | | 1147–1155, 1208–1211 | M |
| warning | `--warning` `#fe9a00`; surface `#fe9a0014` / `#fe9a0029`; text `#bb4d00` / `#ffb900` | | | 1164–1166 | M |
| loading (skeleton) | `motion-safe:animate-skeleton`: opacity 1 ↔ 0.55, stepped | | | 223, 295–313 | M |
| loading (preview bar) | `.preview-loading-progress[data-loading=true]`: opacity 1, `scaleX .04 → .9` over 5.3s | | | 2398–2421 | S |
| live / running | `live-tool-shine` text shimmer: foreground band over secondary-label | `#18181b` over `#71717b` | `#f5f5f5` over `#a1a1a1` | 564–578 | M |
| find match | `::highlight(t3-thread-find)` | bg `#ffde8b`, text `#21180a` | bg `#8c6c1f`, text `#f5f5f5` | 407–415 | P |
| find match (active) | `::highlight(t3-thread-find-active)` | bg `#eea52b`, text `#21180a` | same | 417–423 | P |
| citation | `::highlight(t3-assistant-citation)`: primary × `--assistant-citation-highlight-opacity` (registered `<number>`, initial 0). Comment: primary 45% `#1b4ed873` / `#346bf173` | | | 662–678 | P |
| alert variants | `alert-glass[data-variant=error\|info\|success\|warning]` tint at 4% | `#fb2c360a` `#2b7fff0a` `#00bc7d0a` `#fe9a000a` | `#fb414a0a` `#2b7fff0a` `#00bc7d0a` `#fe9a000a` | 374–404 | M |
| streaming | `.chat-markdown[data-streaming] > *` fade in (600ms) | | | 2210–2220 | S |
| theme change | `html.no-transitions`: all `transition-duration` and `animation-duration` 0s for one frame | | | 1104–1111, useTheme.ts:326–347 | S |
| theme probe | `html[data-theme-token-probe]`: transitions off | | | 1393–1398 | S |
| panel animations | `[data-panel-animations=true]` enables panel width/padding transitions with `--panel-animation-duration` (default `0ms`, live `false`) | | | AppSidebarLayout.tsx:451–461 | M |
| forced colours | `@media (forced-colors: active)`: `.settings-slider` goes back to the native control (`appearance: auto; accent-color: Highlight`, track and thumb `all: revert`). `live-tool-shine` paints plain text. `live-activity-focus` hides (opacity 0, no animation) | | | index.css:1060–1076, 565, 638–642 | S |
| reduced motion | `prefers-reduced-motion: reduce` turns off: `motion-safe:*` animations (skeleton, status-pulse, spinners), `live-tool-shine` (plain text), `live-activity-focus*` (none, opacity 0), the markdown stream fade, the preview loading bar (static at `scaleX(0.9)`), panel animations and visible-animation gating (§5.8). The settings search pulse has no media query. Instead the JS never adds `.settings-search-target-pulse` under reduced motion, so the target's normal focus outline shows (the pulse class suppresses it otherwise) | | | 565, 638–655, 2210, 2411–2421, 939–943; panelAnimations.ts:37; settingsLayout.tsx:146–153 | S |
| hover capability | `@media (hover: hover)` hides the diff copy-path button until its file is hovered or focused. The `any-hover` custom variant (`@media (any-hover: hover) { &:hover }`) lets a secondary hovering pointer reveal sidebar actions on touch-first devices | | | 1311–1316, 10–17 | S |
| focus suppressed | `outline: none` on `[data-workspace-area]:focus-visible` (F6 targets use `[data-workspace-keyboard-focus]` instead), `.composer-tiptap:focus`, `.settings-slider:focus-visible` (thumb ring instead), `.settings-search-target-pulse:focus` | | | 2592–2594, 2455–2458, 1044–1046, 941–943 | S |
| link hover / focus-visible (chat markdown) | dotted underline: `radial-gradient(circle, currentcolor 0.75px, transparent 1px)` at `left bottom`, `4px 2px`, `repeat-x` (not on file links, chips or image links) | `#1447e6` dots | `#50a2ff` dots | 2047–2063 | S |
| empty | no token-level empty state | | | | |

## 5. Behaviour

### 5.1 Appearance options and defaults

| Setting (UI label) | Where | Values | Default | Mechanism | Source |
|---|---|---|---|---|---|
| Theme | Settings → Appearance → Colors & themes | **T3 Code** (standard look, no `data-theme-id`), T3 Chat, Grove, Ocean, Ember, Iris, custom, imported or environment-published themes. A light half and a dark half can come from different themes (the "mix") | T3 Code (`t3code:theme` unset → `"system"`) | localStorage `t3code:theme`, `t3code:theme-halves:v1`, `t3code:themes:v1` | useTheme.ts:41–50; themePalette.ts:44–47; ThemeSettings.tsx:57–63, 725–760 |
| Appearance mode | same page (System / Light / Dark tiles) | `system`, `light`, `dark` | `system` | localStorage `t3code:theme-appearance-mode` (legacy `t3code:theme-follow-system`) | useTheme.ts:146–185; ThemeSettings.tsx:680–700 |
| Contrast | Appearance → Interface | 50–200 (integer %) | 100 | `--appearance-contrast-*` on `<html>` | settings.ts:148–154; appearanceContrast.ts:3–10 |
| Glass opacity | Appearance → Interface | 40–100 | 80 | `--glass-opacity`; at 100 also `--glass-blur: 0px` | settings.ts:137–146; appInit.tsx:93–100 |
| Chat width | Appearance → Interface | Comfortable, Wide, Full | Comfortable | `html[data-chat-width]` → `--chat-content-max-width` 46rem / 72rem / 100% | settings.ts:339–340, 407; index.css:131, 2293–2299 |
| Effort display | Appearance → Interface | Word (`label`), Bars | Word | component only | settings.ts:342–344, 408 |
| Math rendering | Appearance → Interface | on/off | off | lazy KaTeX CSS | settings.ts:594 |
| Panel animations | Appearance → Interface | 0–400ms | 0 (off) | `--panel-animation-duration`, `data-panel-animations` | settings.ts:155–164, 415–417; panelAnimations.ts:31–39 |
| Environment identification | Appearance | artwork (follows the channel: "Nightly artwork" here), nightly-artwork, dev-artwork, pill, none | artwork | sidebar header art (`--stage-*`) | AppearanceSettingsPanel.tsx:40–73 |
| Interface font + size | Appearance → Typography | any installed family; 12–20px | default stack; 16px | `--font-sans` (custom list prepended to the default stack); `html { font-size }` | settings.ts:166–172, 491; appearanceFonts.ts:77–101 |
| Monospace font + size | Typography | monospace families only; 10–18px | default stack; 13px | `--font-mono`, `--font-size-code`, `--diffs-font-size` | settings.ts:182–188, 489; appearanceFonts.ts:106–108 |
| Separate fonts | Typography → Advanced | on/off | off | localStorage `t3code:typography-advanced` | appearanceFonts.ts:20; TypographySettings.tsx:280–297 |
| Prompt font + size | Typography (separate fonts on) | family; 12–20px | interface family; 14px | `--font-composer`; `--font-size-prompt` (else `var(--text-sm)`) | settings.ts:174–180, 490; appearanceFonts.ts:80, 102–105 |
| Terminal font + size | Typography (separate fonts on) | family; 8–20px | **code family and code size (13px)** while separate fonts is off (the default). With it on: its own family (default: the code font) and size (default 12px) | terminal renderer | settings.ts:190–196, 492; appearanceFonts.ts:22–38; ThreadTerminalDrawer.tsx:600–612 |
| Font smoothing | Typography → Advanced | on/off ("thinner grayscale text smoothing instead of the macOS default") | on | `html { -webkit-font-smoothing: antialiased }` | settings.ts:494; appearanceFonts.ts:110–114; TypographySettings.tsx:218–234 |
| Diff colors | Settings (diff section) | Red & green, Blue & orange | Red & green | `html[data-diff-color-scheme]` | settings.ts:337, 404–406; SettingsPanels.tsx:1452–1490 |

### 5.2 Theme resolution

1. **Boot, before any CSS** (index.html:13–388). An inline script reads `t3code:theme`, `t3code:theme-appearance-mode`, `t3code:theme-follow-system`, `t3code:theme-halves:v1`, custom themes in `t3code:themes:v1`, and `matchMedia("(prefers-color-scheme: dark)")`.
   - It toggles `html.dark` (357), sets or deletes `data-theme-id` and `data-theme-selected`, and sets `html.style.backgroundColor`: `#ffffff` / `#000000` for the standard look, or the theme's chrome colour (367).
   - It rewrites every `<meta name="theme-color">`. Only when a theme is selected (`data-theme-selected="true"`) does it also set `--boot-background/foreground/accent` inline (index.html:368–373), which the splash and boot-error styles read under `html[data-theme-selected="true"]` (421–424, 468–471).
   - Standard splash: light `#ffffff` / `#262626` / `#4f46e5`, dark `#000000` / `#f5f5f5` / `#818cf8` (index.html:21–32). Without a selection the splash just inherits the boot `body` colours (`#ffffff`/`#262626`, dark `#000000`/`#f5f5f5`).
   - If storage throws, it follows `prefers-color-scheme`. If `matchMedia` throws too, it falls back to **dark** (`let prefersDark = true`, 375–386).
2. **Runtime.** `useTheme.ts` runs `applyTheme(getStored())` at module load (394–396). Appearance mode comes from storage, falling back to `system`. When following the system, it listens for `change` on the `prefers-color-scheme: dark` MediaQueryList (466–478) and re-applies with transitions suppressed (436–442). Changes in other tabs arrive as `storage` events (444–462).
3. `applyTheme` (306–348):
   - adds `no-transitions` when suppressing;
   - calls `applyThemePalette` (themePalette.ts:1629–1653). For the standard look this deletes `data-theme-id`, removes every `--app-theme-*`, and writes the 24 `--context-chip-*` values solved from `getStandardPaintedThemeColors(appearance)`. For a theme it sets `data-theme-id` and the 61 `--app-theme-*` roles inline, then solves the chips from those roles;
   - toggles `dark` (337), syncs the browser chrome colour (339) and tells the desktop shell (340);
   - forces a reflow and removes `no-transitions` on the next animation frame (341–346).
4. **Which appearance wins** (themePalette.ts:1656–1680): `system` mode uses the system appearance unless the chosen theme lacks that mode, in which case it uses the theme's base appearance. A stored half for the wanted appearance takes priority. All five built-in themes define both modes (light base + dark variant).
5. **Theme mapping.** With `html[data-theme-id]` (non-empty), index.css:1325–1389 maps roles onto the semantic tokens (table below). The same is repeated inside `[data-app-sidebar]` (1563–1589). Extra themed-only overrides cover the chat header controls (1467–1517), code blocks (1522–1539), T3 Chat inline code (1544–1561), the grain tile (1776–1779), the composer's `--chat-composer-*` variables (§3.15) and the sidebar container's own border colour. That border is `color-mix(in srgb, var(--contrast-sidebar-foreground) 10%, transparent)` in light and 8% in dark, except T3 Chat dark, which keeps `--contrast-sidebar-border` (1584–1597).

| Semantic token | Theme role variable | index.css |
|---|---|---|
| `--background` | `var(--app-theme-canvas)` | 1326 |
| `--app-chrome-background` | `var(--app-theme-chrome)` | 1327 |
| `--toolbar-background` | `var(--app-theme-toolbar)` | 1328 |
| `--toolbar-foreground` | `var(--app-theme-toolbar-foreground)` | 1329 |
| `--toolbar-border` | `var(--app-theme-toolbar-border)` | 1330 |
| `--toolbar-control` | `var(--app-theme-toolbar-control)` | 1331 |
| `--toolbar-control-foreground` | `var(--app-theme-toolbar-control-foreground)` | 1332 |
| `--toolbar-control-hover` | `var(--app-theme-toolbar-control-hover)` | 1333 |
| `--surface-raised` | `var(--app-theme-surface-raised)` | 1334 |
| `--foreground` | `var(--app-theme-text)` | 1335 |
| `--card` | `var(--app-theme-surface)` | 1336 |
| `--card-foreground` | `var(--app-theme-text)` | 1337 |
| `--popover` | `var(--app-theme-surface-overlay)` | 1338 |
| `--popover-foreground` | `var(--app-theme-text)` | 1339 |
| `--primary` | `var(--app-theme-message-action)` | 1344 |
| `--primary-foreground` | `var(--app-theme-message-action-foreground)` | 1345 |
| `--secondary` | `var(--app-theme-secondary)` | 1346 |
| `--secondary-foreground` | `var(--app-theme-secondary-foreground)` | 1347 |
| `--muted` | `var(--app-theme-muted)` | 1348 |
| `--muted-foreground` | `var(--app-theme-muted-foreground)` | 1349 |
| `--placeholder` | `var(--app-theme-placeholder)` | 1350 |
| `--secondary-label` | `var(--app-theme-secondary-label)` | 1351 |
| `--icon-muted` | `var(--app-theme-icon-muted)` | 1352 |
| `--accent` | `var(--app-theme-accent-surface)` | 1353 |
| `--accent-foreground` | `var(--app-theme-accent-surface-foreground)` | 1354 |
| `--message-surface` | `var(--app-theme-message-surface)` | 1355 |
| `--message-foreground` | `var(--app-theme-message-foreground)` | 1356 |
| `--message-action` | `var(--app-theme-message-action)` | 1357 |
| `--message-action-foreground` | `var(--app-theme-message-action-foreground)` | 1358 |
| `--message-action-hover` | `var(--app-theme-message-action-hover)` | 1359 |
| `--error` | `var(--app-theme-error)` | 1360 |
| `--error-foreground` | `var(--app-theme-error-foreground)` | 1361 |
| `--tool-error-icon` | `var(--error-foreground)` | 1362 |
| `--error-surface` | `var(--app-theme-error-surface)` | 1363 |
| `--destructive` | `var(--error)` | 1364 |
| `--destructive-foreground` | `var(--error-foreground)` | 1365 |
| `--warning` | `var(--app-theme-warning)` | 1366 |
| `--warning-foreground` | `var(--app-theme-warning-foreground)` | 1367 |
| `--warning-surface` | `var(--app-theme-warning-surface)` | 1368 |
| `--update` | `var(--app-theme-update)` | 1369 |
| `--update-foreground` | `var(--app-theme-update-foreground)` | 1370 |
| `--border` | `var(--app-theme-border)` | 1371 |
| `--input` | `var(--app-theme-input)` | 1372 |
| `--ring` | `var(--app-theme-focus)` | 1373 |
| `--sidebar` | `var(--app-theme-sidebar)` | 1374 |
| `--sidebar-foreground` | `var(--app-theme-sidebar-foreground)` | 1375 |
| `--sidebar-muted-foreground` | `var(--app-theme-sidebar-muted-foreground)` | 1376 |
| `--sidebar-control-surface` | `var(--app-theme-sidebar-control-surface)` | 1377 |
| `--sidebar-row-hover` | `var(--app-theme-sidebar-row-hover)` | 1378 |
| `--sidebar-row-active` | `var(--app-theme-sidebar-row-active)` | 1379 |
| `--sidebar-row-selected` | `var(--app-theme-sidebar-row-selected)` | 1380 |
| `--sidebar-border` | `var(--app-theme-sidebar-border)` | 1381 |
| `--sidebar-stage-fade` | `var(--app-theme-sidebar)` | 1382 |
| `--code-background` | `var(--app-theme-code-background)` | 1383 |
| `--code-foreground` | `var(--app-theme-code-foreground)` | 1384 |
| `--terminal-background` | `var(--app-theme-terminal-background)` | 1385 |
| `--terminal-foreground` | `var(--app-theme-terminal-foreground)` | 1386 |
| `--terminal-cursor` | `var(--app-theme-terminal-cursor)` | 1387 |
| `--terminal-selection-background` | `var(--app-theme-terminal-selection-background)` | 1388 |

Built-in themes, key roles resolved to sRGB (S: source oklch, converted by the live browser). The full 61-role tables for both modes are in `tokens.extra.json` → `builtInThemes`.

| Theme (id) | Mode | canvas | sidebar | text | textMuted | accent | messageAction | border | surface | messageSurface |
|---|---|---|---|---|---|---|---|---|---|---|
| T3 Chat (`t3-chat`) | light | `#fdf7fd` | `#f2e1f4` | `#501854` | `#ac1668` | `#db2777` | `#db2777` | `#eee1ed` | `#faf3fb` | `#f7def2` |
| T3 Chat (`t3-chat`) | dark | `#1f1a24` | `#171018` | `#f9f8fb` | `#e7d0dd` | `#a3004c` | `#a3004c` | `#27242c` | `#29232d` | `#2b2431` |
| Grove (`grove`) | light | `#f3f7f4` | `#e2ede7` | `#241523` | `#746c74` | `#1b7d50` | `#8f6410` | `#cbd5d1` | `#f3f7f4` | `#cce1d7` |
| Grove (`grove`) | dark | `#1b2821` | `#21362b` | `#fffaff` | `#919595` | `#69d69a` | `#e3b34e` | `#415f4f` | `#1b2821` | `#37664d` |
| Ocean (`ocean`) | light | `#f5f7f8` | `#e4ecf2` | `#241523` | `#746c75` | `#2672af` | `#0a6f75` | `#cdd4dc` | `#f5f7f8` | `#d0dfeb` |
| Ocean (`ocean`) | dark | `#17212b` | `#1e2d3b` | `#fffaff` | `#8d8f97` | `#70b9ee` | `#5bd0d6` | `#405567` | `#17212b` | `#375871` |
| Ember (`ember`) | light | `#f9f7f5` | `#f3eae5` | `#241523` | `#766c74` | `#ae552a` | `#b23535` | `#ddd2ce` | `#f9f7f5` | `#ebdad1` |
| Ember (`ember`) | dark | `#291e1a` | `#39281f` | `#fffaff` | `#968e8f` | `#f09a64` | `#f78a7a` | `#664c3f` | `#291e1a` | `#704b34` |
| Iris (`iris`) | light | `#f8f7f9` | `#edeaf4` | `#241523` | `#766c76` | `#7253b9` | `#a82c87` | `#d6d1de` | `#f8f7f9` | `#dfd9ee` |
| Iris (`iris`) | dark | `#1d1929` | `#272139` | `#fffaff` | `#8e8a95` | `#9d7df2` | `#f099d8` | `#4d4366` | `#1d1929` | `#4b3d72` |

The shared "T3 Code" palette in `packages/shared/src/themePalettes.ts:108–234` seeds the theme editor and the mobile app. **It is not what the web paints.** The web standard look comes from index.css. Where they differ:

| Role (shared palette) | Web token | Shared light | Web light | Shared dark | Web dark |
|---|---|---|---|---|---|
| `canvas` | `--background` | `#fcfcfc` | `#fcfcfc` | `#0a0a0a` | `#000000` |
| `text` | `--foreground` | `#27272a` | `#18181b` | `#f5f5f5` | `#f5f5f5` |
| `surface` | `--card` | `#ffffff` | `#ffffff` | `#111111` | `#111111` |
| `surfaceRaised` | `--surface-raised` | `#fcfcfc` | `#ffffff33` | `#111111` | `#111111` |
| `messageSurface` | `--message-surface` | `#f4f4f5` | `#ebebed` | `#141414` | `#1e1e1e` |
| `mutedForeground` | `--muted-foreground` | `#71717b` | `#71717b` | `#818181` | `#a1a1a1` |
| `border` | `--border` | `#e4e4e7` | `#e4e4e7` | `#191919` | `#ffffff14` |
| `input` | `--input` | `#d4d4d8` | `#d4d4d8` | `#1e1e1e` | `#ffffff1c` |
| `sidebar` | `--sidebar` | `#fafafa` | `#fafafa` | `#000000` | `#111111` |
| `accentSurface` | `--accent` | `#f4f4f5` | `#f4f4f5` | `#141414` | `#ffffff0a` |

The web column is the `:root` value. The sidebar itself paints from its own scope (§3.5): dark `--sidebar` is `#000000` there, matching the shared palette.

### 5.3 Contrast and glass application

- Contrast `c` (appearanceContrast.ts:3–10): `--appearance-contrast-base = min(c, 100)%`, `--appearance-contrast-boost = max(c − 100, 0)%`, `--appearance-contrast-border-boost = max(c − 100, 0) / 4 %`.
  - Below 100, text fades toward its surface and borders fade toward transparent.
  - Above 100, text moves toward black (light) or white (dark), and borders are mixed with the foreground.
  - Measured outcomes (P, probe with the per-level variables):

| Token | 50 | 75 | **100 (default)** | 125 | 150 | 200 |
|---|---|---|---|---|---|---|
| light `--contrast-foreground` | `#808082` | `#49494c` | `#18181b` | `#0c0c0f` | `#040405` | `#000000` |
| light `--contrast-muted-foreground` | `#b4b4ba` | `#92929a` | `#71717b` | `#4b4b52` | `#27272c` | `#000000` |
| light `--contrast-border` | `#e4e4e780` | `#e4e4e7bf` | `#e4e4e7` | `#d7d7da` | `#cbcbce` | `#b1b1b4` |
| light `--contrast-input` | `#d4d4d880` | `#d4d4d8bf` | `#d4d4d8` | `#c8c8cd` | `#bcbcc1` | `#a5a5a9` |
| light `--contrast-accent-foreground` | `#7c7c7f` | `#47474a` | `#18181b` | `#0c0c0f` | `#040405` | `#000000` |
| light `--contrast-secondary-foreground` | `#89898b` | `#555558` | `#27272a` | `#171719` | `#08080a` | `#000000` |
| light `--contrast-sidebar-foreground` | `#7f7f81` | `#48484b` | `#18181b` | `#0c0c0f` | `#040405` | `#000000` |
| light `--contrast-sidebar-muted-foreground` | `#b3b3b9` | `#91919a` | `#71717b` | `#4b4b52` | `#27272c` | `#000000` |
| light `--contrast-message-foreground` | `#79797b` | `#454548` | `#18181b` | `#0c0c0f` | `#040405` | `#000000` |
| dark `--contrast-foreground` | `#5f5f5f` | `#a7a7a7` | `#f5f5f5` | `#f7f7f7` | `#fafafa` | `#ffffff` |
| dark `--contrast-muted-foreground` | `#3b3b3b` | `#6c6c6c` | `#a1a1a1` | `#b7b7b8` | `#cfcfcf` | `#ffffff` |
| dark `--contrast-border` | `#ffffff0a` | `#ffffff0f` | `#ffffff14` | `#fafafa23` | `#f9f9f932` | `#f7f7f74f` |
| dark `--contrast-input` | `#ffffff0e` | `#ffffff15` | `#ffffff1c` | `#fbfbfb2a` | `#f9f9f938` | `#f7f7f755` |
| dark `--contrast-accent-foreground` | `#f5f5f585` | `#f5f5f5c2` | `#f5f5f5` | `#f7f7f7` | `#fafafa` | `#ffffff` |
| dark `--contrast-secondary-foreground` | `#f5f5f583` | `#f5f5f5c1` | `#f5f5f5` | `#f7f7f7` | `#fafafa` | `#ffffff` |
| dark `--contrast-sidebar-foreground` | `#797979` | `#b5b5b5` | `#f5f5f5` | `#f7f7f7` | `#fafafa` | `#ffffff` |
| dark `--contrast-sidebar-muted-foreground` | `#535353` | `#797979` | `#a1a1a1` | `#b7b7b8` | `#cfcfcf` | `#ffffff` |
| dark `--contrast-message-foreground` | `#818181` | `#b9b9b9` | `#f5f5f5` | `#f7f7f7` | `#fafafa` | `#ffffff` |

  In dark, below 100, `--contrast-accent-foreground` and `--contrast-secondary-foreground` become translucent, because their surfaces (`--accent`, `--secondary`) are translucent.

  The table is at `:root` scope. Inside `[data-app-sidebar]` the same formulas run on the sidebar's palette (§3.5). In dark that is `#f1f3f7` on `#000000`, so the sidebar's own `--contrast-*` values at a non-default contrast differ from these rows. Re-derive them with the formulas in §3.4.
- Glass (appInit.tsx:93–100): `--glass-opacity: <n>%` inline. At 100 the app writes `--glass-blur: 0px` inline. Otherwise it removes the inline blur, so the CSS value applies (12px light, 16px dark). `--glass-saturation` is fixed (1.14 light, 1.08 dark).

### 5.4 Browser chrome colour

`syncBrowserChromeTheme` (useTheme.ts:279–304) reads the computed background of `main[data-slot='sidebar-inset']` (else `[data-slot='sidebar-inner']`, else `body`). With a theme it uses `--app-chrome-background` instead. It writes the result to `html.style.backgroundColor`, `body.style.backgroundColor` and every `meta[name=theme-color]`. It runs on every theme apply and after each route render or resolve (appInit.tsx:220–224). Measured: light `oklch(0.992 0 0)` (= `#fcfcfc`), dark `rgb(0, 0, 0)`.

### 5.5 Context-chip colour solving

`contextChipVariables` (themePalette.ts:694–730):

- **Surfaces.** The canvas, plus `surface` and `surfaceRaised` composited on the canvas, plus `messageSurface` if it is on the same side of luminance 0.179 as the canvas. The standard painted palette overrides the shared one with light `messageSurface: #ebebed` and dark `canvas: #000000`, `messageSurface: #1e1e1e` (408–415).
- **Direction and target.** Solve darker if black's minimum contrast on those surfaces is at least white's, otherwise lighter. Target = min(5, best achievable).
- **Solving.** Each seed's OKLCH lightness is moved, keeping hue and chroma, until the colour reaches the target on every surface. The result is emitted as hex.
- `--context-chip-invalid` uses the `errorForeground` role.
- `--context-chip-message` is set only when the message surface flips luminance side relative to the canvas.
- Previews (`applyThemeColorPreview`, 1615–1627) run the same solver.

### 5.6 Fonts

`applyAppearanceFontVariables` (appearanceFonts.ts:72–115):

- A custom family list is quoted and prepended: `--font-sans: <custom>, <default stack>`, the same for `--font-mono`. An empty value removes the inline property, so the CSS default applies.
- `--font-composer` is written only when separate fonts is on: `<custom>, var(--font-sans)`.
- Root `font-size` = the interface size clamped to 12–20 and rounded.
- `--font-size-prompt` = the prompt px (separate fonts on) or `var(--text-sm)`. `--font-size-code` = `--diffs-font-size` = the code px.
- `-webkit-font-smoothing: antialiased` when smoothing is on.
- `--list-marker-sans` comes from canvas text metrics.

### 5.7 Data attributes and classes on `<html>`

- `data-diff-color-scheme` and `data-chat-width` are written from client settings, each only when it changes (appInit.tsx:84–89).
- `dark`: index.html:357, 382; useTheme.ts:337; themePalette.ts:1621 (previews).
- `electron`, `electron-windows`, `wco`: lib/windowControlsOverlay.ts:7–9, 50–62, 80–100, run only in Electron (appInit.tsx:255–258).
- `data-theme-id`, `data-theme-selected`: boot script and `applyThemePalette`.
- `data-mobile-composer-route-transition="true"` during the mobile hero-composer view transition (index.css:21–89; components/chat/draftHeroTransition.ts).

### 5.8 Visible-animation gating

Elements registered with `observeVisibleAnimation` get `--visible-animation-state: running` and `--visible-animation-will-change: transform` only while all three hold: they intersect the viewport, `document.visibilityState === "visible"`, and `prefers-reduced-motion` does not match. Otherwise `paused` / `auto` (lib/visibleAnimation.ts:10–27). `live-tool-shine`, `live-activity-focus*` and `visible-animate-spin` read these variables, falling back to `paused`.

## 6. Motion

| Item | Property | Duration | Easing | Delay / iterations | Gate | Source | M/S |
|---|---|---|---|---|---|---|---|
| default `transition` utility | colours, opacity, box-shadow, transform, translate, scale, rotate, filter, backdrop-filter, display, content-visibility, overlay, pointer-events, gradient stops | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | – | – | Tailwind `--default-transition-*` | M |
| `transition-colors` | color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, gradient stops | 150ms | `cubic-bezier(0.4, 0, 0.2, 1)` | – | – | Tailwind | M |
| `transition-opacity` / `-shadow` / `-transform` | opacity / box-shadow / transform, translate, scale, rotate | 150ms | same | – | – | Tailwind | M |
| durations used | `duration-0, 100, 120ms, 140, 150, 180, 200, 250, 300, 2000` | | | | | shipped CSS | M |
| delays used | `delay-50, 150, 300, 2000` (ms) | | | | | shipped CSS | S |
| `ease-out` / `ease-in-out` / `ease-linear` | | | `cubic-bezier(0, 0, 0.2, 1)` / `cubic-bezier(0.4, 0, 0.2, 1)` / `linear` | | | Tailwind | M |
| `ease-drawer` ("surfaces that slide or pop in: composer context, pills") | | | `cubic-bezier(0.32, 0.72, 0, 1)` | | | index.css:191–195 | M |
| panel open/close (sidebar width, header padding, preview width, terminal rows) | width, left/right, padding, grid-template-rows | `--panel-animation-duration` (setting, default **0ms = off**) | `ease-out` | – | `data-panel-animations=true` and no reduced motion | AppSidebarLayout.tsx:451–461; panelAnimations.ts:31–39; sidebar.tsx:388–400 | M |
| theme change | all transitions and animations | 0s for one frame (`.no-transitions`) | – | – | – | index.css:1104–1111 | S |
| `animate-spin` | rotate 0→360° | 1s | linear | infinite | – | Tailwind | M |
| `animate-pulse` | opacity 1→0.5→1 (50%) | 2s | `cubic-bezier(0.4, 0, 0.6, 1)` | infinite | – | Tailwind | M |
| `animate-skeleton` | opacity 1 (0–42%) → 0.55 (50–92%) → 1, `steps(4)` per ramp | 2.4s | ease (keyframe timing steps(4)) | infinite | `motion-safe:` | index.css:223, 295–313 | M |
| `animate-status-pulse` | opacity 1 (0–40%) → 0.5 (50–90%) → 1, `steps(6)` | 2s | steps(6) | infinite | `motion-safe:` | 226, 332–346 | M |
| `animate-status-ping` | opacity 0.9 → 0, scale 0.75 → 2 by 40%, `steps(8)`, then hidden | 2s | steps(8) | infinite | both forms ship: `motion-safe:animate-status-ping` (ConnectionStatusDot.tsx:40) and plain `animate-status-ping motion-reduce:animate-none` (AgentBrowserCursor.tsx:88) | 227, 347–361 | M |
| `animate-stash-pulse-row` / `-alt` | color held at `--color-foreground` to 85.7% (1.2s), eases back over 200ms | 1.4s | ease | 1 | – | 230–231, 314–325 | M |
| `animate-stash-pulse-count` | color held at `--color-primary` to 85.7%, then snaps | 1.4s | `step-start` | 1 | – | 232, 326–331 | M |
| `live-tool-shine` | background-position of a 4.5rem (72px) foreground band from −72px to `100% + 72px`; text clipped | 2.2s | `steps(30)` | infinite; play-state from `--visible-animation-state` | no reduced motion and no forced colours (else plain text) | 551–578 | M |
| `live-activity-focus` / `-counter` | translateX 0→100% / 0→−100% (mask band 4.5rem wide, ramps at 0.675, 1.575, 2.25, 2.925, 3.825rem) | 2.2s | `steps(30)` | infinite | reduced motion or forced colours: none (focus layer opacity 0) | 586–660 | M |
| `visible-animate-spin` | spin | 1s | linear | infinite, gated | `motion-safe:` | 580–584 | M |
| markdown streaming | opacity 0 → 1 on newly inserted blocks (`@starting-style`) | 600ms | ease-out | once | no reduced motion; `[data-streaming]` only | 2210–2220 | S |
| preview loading bar | `scaleX(0.04) → scaleX(0.9)` | 5.3s | `cubic-bezier(0.1, 0.5, 0.2, 1)` | forwards | reduced motion: static at `scaleX(0.9)` | 2388–2421 | S |
| preview bar exit | transform; opacity | 150ms; 150ms | ease-out | opacity delay 220ms | – | 2401–2403 | S |
| settings search target | inset 2px ring 0 → primary 70% → 0 | 650ms | ease-in-out | 2 | (reduced motion: focus outline instead) | 923–943 | S |
| settings slider thumb | transform, box-shadow | 120ms | ease | – | – | 1009–1024 | S |
| sidebar drag label | visibility hidden → visible | 200ms | step-end | once per pickup | – | 2425–2436 | S |
| mobile composer view transition | group `--mobile-composer-transition-duration`; old/new crossfade (hold to 35%, swap by 65%) | runtime | `cubic-bezier(0.4, 0, 0.2, 1)` (group); linear (crossfade) | both | mobile route only | 21–68 | S |
| mobile draft headline exit | opacity 1→0, translateY 0 → −6px | runtime | `cubic-bezier(0.4, 0, 1, 1)` | both | mobile | 70–89 | S |
| `ProseMirror-gapcursor` blink | visibility | 1.1s | `steps(2, start)` | infinite | – | Tiptap injected style (live) | M |

`prefers-reduced-motion` was `no-preference` on the measured page.

## 7. Gaps

- **Wide gamut.** 51 of the 80 palette colours, and the tokens built from them (info, success, warning, error, guest, diff, update), are outside sRGB. On the reference Mac (P3 display), Chrome paints them more saturated than the sRGB hex recorded here. sRGB is the best Gio can do, and the values match Tailwind's own sRGB fallbacks.
- **Fonts on macOS** were not measured. On a Mac, `-apple-system` gives SF Pro and `ui-monospace` gives SF Mono, so advances, line breaking and `--list-marker-sans` (1.175em here, from Inter) will differ. Sizes, weights and line heights are font-independent and are as specified.
- **Selection colour** has no app rule. Only Linux Chrome's default was measured. macOS uses the system highlight colour.
- **Scrollbar `:hover`**: synthetic pointer moves did not trigger the native scrollbar hover. `#bfbfbf` / `#ffffff1f` are source values.
- **Scrollbars on macOS** were not measured. The `::-webkit-scrollbar`-styled bars (6px vertical, 15px horizontal, white corner) are drawn by Chromium itself and should look the same on a Mac. The `.chat-markdown pre` bar is a standard `scrollbar-width: thin` bar, which macOS draws in its own style (overlay or always-on, following the system setting). Only the Linux rendering (10px, arrow buttons) was seen.
- **Electron-only values** (`.wco`, `.electron-windows`, macOS traffic-light inset, window background, title-bar overlay) cannot appear in the browser build. They are source-derived.
- **Themes.** Built-in themes, custom or imported themes, theme halves and the theme editor preview were not switched on: that would change the user's real settings. Built-in role colours are source oklch converted by the live browser. The generator for custom or published themes (`createVividThemeColors`, themePalette.ts:789–924) and its contrast solving are not documented here.
- **Contrast levels and the blue-orange diff scheme** were resolved by evaluating the declared expressions on probes, not by changing the setting.
- **Attached composer banners** (`--chat-composer-attached-*`, `--chat-composer-attachment-overlap`, `--background-image-composer-seam-above`) were not present in the measured state.
- **`--mobile-composer-transition-duration`** is set at runtime by draftHeroTransition.ts (mobile only) and was not measured.
- **Grain** is a procedural SVG noise. Only its effect on painted pixels was measured (§3.7).
- **Pressed and hover states** that need real pointer input (slider thumb, `active:` utilities) are source values.
- Safe-area insets are 0 in this desktop browser, so the `env(safe-area-inset-*)` tokens resolve to their gutters alone.

## 8. Port notes for Go/Gio

1. **Generate, don't transcribe.** `python3 tools/t3ui/gentokens.py ../docs/ui/tokens.json > ui/tokens_gen.go` turns `themes.light/dark` into `T3Light`/`T3Dark` palettes and `Metrics`. Use the resolved values as final. Don't recompute `color-mix()` from rounded or clipped hexes: Chrome mixes unrounded, unclipped floats. For example, dark `--code-background` is `#101010`, while 90% of `#111111` over black would round to `#0f0f0f`.
2. **Keep alpha.** Dark `--muted`/`--secondary` `#ffffff08`, `--accent` `#ffffff0a`, `--border` `#ffffff14`, `--input` `#ffffff1c`, the light `--surface-raised` `#ffffff33`, and every glass fill are translucent. Paint them with alpha over the real parent surface (Gio composites NRGBA). Don't pre-flatten against black.
3. **Default text colour is `#262626` / `#f5f5f5`** (the boot-style quirk, §2.1). Only elements with `text-foreground` (or similar) use `#18181b`.
4. **Sidebar palette.** Inside the sidebar, dark mode uses the scoped palette (`tokens.extra.json` → `scopes["[data-app-sidebar]"]`): surface `#000000`, text `#f1f3f7`, rows `#f1f3f7` at 8/11/7%, and the stage-art fade is `#000000`, not `#111111`. gentokens.py reads only `tokens.json`, so add a `SidebarDark` override (light equals root).
5. **rem scaling.** All `rem` tokens scale with the interface size (12–20, default 16). Keep a `rem` multiplier. Radii mix rem and px (`rounded-sm` = 0.5rem − 4px). Code text is a fixed 13px. Prompt text is `text-sm` (0.875rem) unless separate fonts is on.
6. **Fonts.** On macOS use the system UI font (SF Pro) and SF Mono, loaded from the system at runtime (they can't be bundled). Elsewhere bundle Inter Variable (OFL, as T3 does) and pick the first available mono from the stack. Inter has no italic face here, so synthesize oblique. Weights in use: 400/500/600/700. Grayscale antialiasing matches `-webkit-font-smoothing: antialiased`.
7. **Type pairs.** Use the exact size/line-height pairs: 7/7, 8/8, 10/14, 11/16, 12/16, 14/20, 16/24, 18/28, 20/28, 24/32, 30/36, 36/40. The `html` default line height is 1.5.
8. **Icons.** Draw lucide icons at stroke 1.75 (T3 overrides lucide's 2).
9. **Shadows.** Implement `box-shadow` lists: offset, blur, negative spread. Paint `inset-shadow-control-*` as **outer** 1px lines. Dialogs: `0 24px 64px -24px #000000a6` light; dark adds an inset `0 1px #ffffff0a` and `0 24px 72px -20px #000000e6`.
10. **Glass.** Gio has no cheap backdrop blur. Paint the resolved translucent fill (`#fcfcfccc` / `#000000cc`, dropdown `#ffffffd5` / `#111111d5`, composer `#ffffffcc` / `#111111cc`). Add a blur of the content below only if affordable. At glass opacity 100, fills are opaque and blur is off.
11. **Focus.** Every element's base outline colour is `--ring` at 50% (`#1b4ed880` / `#346bf180`) with `outline-offset: -2px`, and the outline style is `none`. Components opt in with `focus-visible:outline-*` utilities. Workspace keyboard focus draws `2px solid` full `--ring`, inset 2px. Popups never draw one.
12. **Scrollbars.** Vertical: 6px wide, 3px-radius thumb, `#d9d9d9` (hover `#bfbfbf`) light, `#ffffff14` (hover `#ffffff1f`) dark, transparent track, no arrows. It **takes 6px of layout width** from the scroller's content box; it does not overlay. Horizontal native bars are **15px tall**, same thumb colour and radius. Where both meet, the corner is a white `#ffffff` square (6×15px). Code blocks (`.chat-markdown pre`) use the platform's standard thin scrollbar, thumb `#e4e4e7c7` / `#ffffff10`, not the 6px style.
13. **Selection.** T3 sets none. Use the platform highlight (macOS `selectedTextBackgroundColor`), or ≈ rgba(0, 65, 198, 0.8) with white text to match Linux Chrome.
14. **Motion.** Default transitions are 150ms `cubic-bezier(0.4, 0, 0.2, 1)`; drawers use `cubic-bezier(0.32, 0.72, 0, 1)`. Panel animations are off by default. Stepped animations (`steps(4|6|8|30)`) save redraws, so emulate the steps. Pause animations when offscreen, when the window is hidden, or under reduced motion (§5.8). Swap palettes on theme change without animating.
15. **Theme follow.** Default appearance is *system*: watch the OS appearance and switch `T3Light`/`T3Dark` live. The desktop window background is `#ffffff` / `#000000` before the first frame.
16. **Context chips.** Use the solved per-mode hexes (`--context-chip-*`), not the oklch seeds. Under a theme, re-solve with the 5:1 rule (§5.5).
17. **Grain** is optional. Its painted effect is −1.2/255 on the light sidebar and +3.8/255 on the dark one. To emulate it, tile a 256px fractal-noise texture at 3.5% over the sidebar and the main surfaces. Body gets it in light only; in dark the body under the panes is flat `#000000`.
18. **Built-in themes** (`tokens.extra.json` → `builtInThemes`): map roles to semantic tokens with the §5.2 table, then derive the contrast tokens. All five have light and dark modes and sidebar artwork.
19. **Pills.** `rounded-full` computes to `3.35544e+07px`. Treat any radius of at least half the box as a full pill (radius = min(w, h) / 2).
20. **Native controls follow `color-scheme`** (`light` / `dark` on `:root`). Checkboxes, the forced-colours slider fallback and system colours such as `Highlight` (the composer chip range selection) take the light or dark system look.
21. **Sidebar width** is a px value (default 256px, resizable, clamped to [max(208px, brand width), window width − 640px]). It does not scale with the interface font size.

## 9. Verification

An adversarial second pass on 2026-11-17, against the same build and the same live page in both themes. I measured with my own resolver, not the extractor's scripts. It puts probe elements inside the real scope (`<html>`, `[data-app-sidebar]`, `[data-slot=composer-shell]`), reads Chrome's computed colour, and converts it twice: once with Chrome's own canvas, once with independent OKLab → sRGB math. The two agreed on every value. I put the browser back on light, with no probes left, when I finished.

### 9.1 What was checked

Live, in light and dark:

- **tokens.json themes.** Every runtime custom property, at `:root`: 225 colours in light, 226 in dark (`--diff-added-icon` exists only there), and 58 lengths in each. All matched exactly.
- **`@theme inline` tokens.** 71 per theme, at `:root` and inside the sidebar. These aren't runtime variables (Tailwind inlines them), so I evaluated their declared expressions on probes. All matched.
- **Scopes.** 53 sidebar-scope entries per theme and the composer variables. All matched.
- **The doc's own tables.** I parsed every light/dark cell in §3.1–§3.8, §3.15–§3.17 and §3.20, about 600 cells, plus the 48 context-chip cells. All matched apart from the errors listed in §9.2.
- **Effect utilities** on probes with the real classes: surface, dropdown, dialog and alert glass (each variant), dialog backdrop, live-tool-shine, topbar fade, stage backdrop.
- **Scales.** The 12 type sizes, tracking, leading, 10 radii, 11 shadows, inset shadows and drop-shadows.
- **Base and component values.** Base `*` defaults, `::before` border colour, body colour and grain, `color-scheme`, `theme-color`, and every §3.14 row. The composer's `::before`/`::after` fill, border, blur and inset highlight, and the host shadow.
- **Contrast variants.** 35 cells per theme at levels 50, 75, 125, 150 and 200, re-derived from the formulas, against both the doc and tokens.json.
- **Fonts.** The loaded `@font-face`, which family the sans and mono stacks resolve to, `--list-marker-sans` (1.1748 → `1.175em`), italic synthesis, and the lucide stroke (99 of 99 icons at 1.75px).
- **Scrollbar geometry** (layout and painted pixels) and **grain** pixel statistics: sidebar 248.79 light and 3.78 dark, main flat 252 and 0. Both confirm the doc.

From source:

- `appearanceContrast.ts`, the `appInit.tsx` glass, diff and width writes, `appearanceFonts.ts`, the setting ranges and defaults in `packages/contracts/src/settings.ts`, and `DesktopWindow.ts` / `DesktopAppSettings.ts`.
- 18 built-in theme seeds converted from OKLCH with my own math. All matched the §5.2 table.
- `useTheme.ts` apply and listeners, the chip solver in `themePalette.ts:694–739`, `visibleAnimation.ts`, `panelAnimations.ts`, and `index.html` (boot script and style).
- The shipped CSS: 149 theme-layer declarations, 80 palette colours, 7 `@font-face` rules, the compiled form of each utility, and the duration, delay and keyframe lists.

**tokens.json** parses. Its top-level keys are now exactly `tag`, `css`, `fonts`, `themes`, `scale`. Each theme entry is `{raw, value}` with lowercase hex, and the light and dark key sets are identical. All 222 custom properties that index.css declares on `:root` or in `@theme`, and all 149 theme-layer variables, are present. `gentokens.py` still runs on it (output to /tmp only).

**Privacy.** I searched both files for names, hosts, paths, emails, IDs, thread or project text and the server address. Nothing turned up, so nothing needed fixing. The new reference screenshots stay in the private reference folder.

### 9.2 What was wrong, and the fix

| # | Where | Was | Now (evidence) |
|---|---|---|---|
| 1 | §3.10, tokens.json `scale.radius.full` | `rounded-full` = `2147483647px` | `3.35544e+07px`. Chrome serializes every infinite length this way, in both themes |
| 2 | §2.2, §3.7, §3.13, port note 17 | body grain in both themes | light only. In dark, the boot style's `html.dark body { background: #000 }` (specificity 0,1,2) resets the shorthand. Measured `background-image: none` |
| 3 | §3.12 | stage-backdrop fade dark `#111111` | `#000000` in its only real context, inside the sidebar (SidebarStageBackdrop.tsx:56). `#111111` is just the `:root` probe value |
| 4 | port note 12 | scrollbar "overlaying the right 6px" | the vertical bar takes 6px of layout (`offsetWidth − clientWidth = 6`). Horizontal native bars are 15px. The unstyled corner paints white |
| 5 | §3.13 | `.chat-markdown pre` scrollbar 6px, radius 999px | `scrollbar-width`/`scrollbar-color` disable its `::-webkit-scrollbar` rules. Measured a 10px standard thin bar with arrow buttons |
| 6 | §3.6 | `--color-ring`/`--color-muted` "referenced by keyframes" | keyframes read only `--color-primary` and `--color-foreground`. Inline styles read `--color-ring` and `--color-muted`. All four resolve at `:root` (measured inside the sidebar) |
| 7 | §3.8 | `--spacing-composer-seam` marked M | P. It is an `@theme inline` token, not a runtime variable, and no shipped utility uses it |
| 8 | §3.15, tokens.json scope | `--composer-glass-fill` listed as a variable | relabelled as the `::before` fill. In the JSON it moved under `derived` |
| 9 | §3.8 | `--sidebar-width` described as `16rem` | a px value from threadSidebarWidth.ts: 256px default, [max(208px, brand), viewport − 640px]. It does not scale with rem |
| 10 | §3.7 | `--lightningcss-light/dark` both "(unset)" | in each theme one is unset (`initial`) and the other is set to an empty value |
| 11 | §5.1 | terminal default "code font; 12px" | code family and code size (13px) while separate fonts is off (ThreadTerminalDrawer.tsx:600–612) |
| 12 | §5.2 | boot script always sets `--boot-*` | only when a theme is selected. If `matchMedia` also throws, it falls back to dark |
| 13 | §3.12 | live-activity widths `544px` / `256px` as values | the container-relative formulas `calc(100% ± 9rem)` (the px figures came from a 400px probe) |
| 14 | §3.13 | `::selection`: "no app rule" | no global rule. Four components set `selection:` utilities |
| 15 | tokens.json | extra top-level keys (`scopes`, `variants`, `builtInThemes`) broke "exactly this shape" | moved, unchanged, to `tokens.extra.json`. Doc references updated |
| 16 | §3.15 | composer outline "on the main surface's `::after`" | on `[data-slot=composer-host]::after`. The main surface's `::after` shows only with an attached banner (measured, ComposerSurface.tsx:43, 58–59) |

### 9.3 What was missing, and is now filled in

- Blue-orange diff values (§3.2).
- `color-scheme` light/dark (§3.13, port note 20).
- Horizontal scrollbar height, the white corner, the table-scrollbar thumb override and the dark `hr` colour (§3.13).
- Theme-inspector hover box colours (§3.14).
- The composer's themed overrides, which element paints its glass, and its no-backdrop fallback (§3.15).
- The themed sidebar border (§5.2).
- Six index.css rules the §3.18 table skipped.
- States for forced colours, reduced motion, hover capability, suppressed focus and the chat-link underline (§4).
- Delay utilities and both `animate-status-ping` forms (§6).
- The contrast table's scope (§5.3).
- `--assistant-citation-highlight-opacity` in tokens.json.
- Light `theme-color` (§3.19).
- The macOS scrollbar gap (§7).
- Port notes 19–21, and §1 rows for every newly cited file.

### 9.4 Still open

- The selection colour is the browser default. I didn't re-measure it.
- Scrollbar `:hover` is still source-only.
- Themes, contrast levels other than 100, blue-orange, wide/full width, glass 100 and separate fonts were all evaluated from their declared expressions, not switched on. Switching any of them would change the account's settings.
- macOS fonts and scrollbars remain unmeasured (§7).
