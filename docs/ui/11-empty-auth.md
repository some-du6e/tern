# 11 · Empty, onboarding and sign-in surfaces, notices

T3 Code web/desktop UI, tag `v0.1.10-nightly.20261112.4606`. Read [01-tokens](01-tokens.md), [02-primitives](02-primitives.md) and [03-shell](03-shell.md) first. This document uses their tokens, Button/Input/Toast/Empty specs and shell regions without repeating them.

This covers everything the window shows when there is no thread to show, when it cannot start, when it is not signed in or paired, when it loses its server, and the notices and notifications it raises on its own:

- startup: the boot splash, the boot failure page, the root error page, the first-run recovery screen;
- sign-in and pairing: the standalone page shell, the `/pair` pairing form, the hosted pairing link page, the T3 Connect sign-in modal (Clerk) and its entry points, the agent and CLI authorization pages, the Alert primitive they use;
- main-window empty states: no active thread, no projects, draft start failure, the hosted "no computer" home, page not found, blank routes;
- onboarding: the Welcome page (`/welcome`) and the post-sign-in Connect wizard (summary);
- connection loss: the reconnecting/offline composer banner, connection diagnosis headlines, the host-freeze and slow-request toasts;
- the app-level toast catalogue, thread notifications (toasts, desktop notifications, sounds, badge) and the background screen-reader announcer.

How tern maps onto this: tern's T3 Connect email-code sign-in corresponds to the Clerk `SignIn` modal that T3 opens with `clerk.openSignIn()` (§2.6, §3.6). tern's pairing-URL step corresponds to T3's pairing surfaces: visually the `/pair` form (§2.3), and by data (host plus token) the hosted pairing page (§2.4). §8 says which tern files change.

Markers in the M/S columns:

- **M**: measured live in the pinned build served by the measured server, 1440×900 @2x, Linux Chrome, Inter.
- **M\***: measured live in the pinned build with my own browser's network altered to reach the state: the session request mocked as signed-out or aborted, the credential POST aborted, the main JS chunk aborted, or offline emulation. Nothing reached the server. The rendering is the real code.
- **H**: measured on the hosted app `https://app.t3.codes` (the source's `DEFAULT_HOSTED_APP_URL`, connectAuth.ts:10). It is a different build (`main-D3w5ftSO.js`, `main-DfFVnlRj.css`) loading Clerk JS 6.44.0, while the pinned tag catalogs `@clerk/clerk-js` 6.42.3 (pnpm-workspace.yaml:38). Only the Clerk modal and the hosted home were measured there, signed out, nothing submitted.
- **P**: probe. The component's exact class strings (merged with the shipped `cn`, `buttonVariants` from `utils-ST3nadSl.js` exports `t` and `Vr`) rendered into the live page and measured. Real CSS, real tokens, hand-built DOM.
- **S**: source only.

Paths below are relative to `apps/web/src` unless they start with `apps/`, `packages/` or `pnpm-`. Colours are `light / dark`. Placeholder text stands in for real names.

## 1. Scope and sources

| File | Lines | What it gives |
|---|---|---|
| `apps/web/index.html` | 389-670, 680-688 | boot `<style>`: body colours, `#boot-shell` splash, every `#boot-error` rule; splash markup |
| `bootstrap.ts` | 1-4 | `import("./main")` failure → `showBootError` |
| `main.tsx` | 14, 30-56 | services start, lazy Clerk shell (`clerkPublishableKey && hasCloudPublicConfig()`), render after `router.load()` |
| `lib/bootError.ts` | 3-12, 26-52, 96-120, 144-152, 154-159, 188-199, 201-255, 257-395 | boot failure page: copy, triage command, update button, error report, repeat-failure open state |
| `routes/__root.tsx` | 62-113, 115-128, 130-140, 142-232, 234-321 | auth gate states, not-found view, toast providers above every branch, branch selection, root error view and report |
| `environments/primary/auth.ts` | 62-102, 128-140, 173-188, 263-302, 304-395 | gate state type, error messages, transient retry, bootstrap, credential submit |
| `routes/pair.tsx` | 9-31, 33-57, 59-61 | `/pair` gate, `PairingRouteSurface` / `HostedPairingRouteSurface` / pending |
| `routes/_chat.tsx` | 48-57 | non-authenticated → `/pair` |
| `routes/_chat.index.tsx` | 24-34, 36-91, 93-118 | index: hosted empty state, auto draft, `NoProjectsHero`, `DraftStartError` |
| `routes/welcome.tsx` | 14-22, 24-62 | `/welcome` gate and fallback to `NoProjectsHero` |
| `components/auth/PairingRouteSurface.tsx` | 18-27, 29-137, 139-255, 257-290 | pairing pending, form, hosted pairing page, copy helpers |
| `components/ui/standalone-page.tsx` | 5-36 | `StandalonePage`, `StandalonePageHeader` |
| `components/T3Wordmark.tsx` | 3-12 | "T3" glyph, `viewBox="15.5309 37 94.3941 56.96"` |
| `components/ui/alert.tsx` | 7-29, 50-143 | Alert variants and slots |
| `components/NoActiveThreadState.tsx` | 6-35 | no active thread |
| `components/NoProjectsHero.tsx` | 13-59 | no projects hero |
| `components/HostedEmptyState.tsx` | 21-64, 66-116, 118-131, 133-163 | hosted home, steps |
| `components/InstallT3Commands.tsx`, `CommandBlock.tsx` | 9-31; 8-43 | install commands |
| `components/cloud/CloudEnvironmentConnectList.tsx` | 160-500 | T3 Connect computer rows (summary) |
| `components/cloud/cloudEnvironmentConnectionPresentation.ts` | 12-59 | button and status labels per phase |
| `components/ui/empty.tsx` | 5-122 | Empty sizes and slots |
| `components/WorkspacePageHeader.tsx`, `workspaceTitlebar.ts` | 6-27; 1-2 | top bar used by the empty states |
| `components/ChatView.tsx`, `ThreadRouteView.tsx` | 3195-3197; 143-155, 216-225 | when `NoActiveThreadState` and blank mains appear |
| `components/onboarding/FirstRunGate.tsx`, `onboarding/firstRun.logic.ts` | 31, 67-181, 183-221; 58-71, 109-140 | first-run gate and recovery screen |
| `components/onboarding/WelcomePage.tsx` | 124-324, 326-344, 346-368, 405-720, 722-744, 746-907, 909-1035, 1104-1192, 1194-1350, 1767-1884 | Welcome page |
| `components/clerk/*` | BrowserManagedAuthShell 8-20; ElectronManagedAuthShell 10-22; clerkAppearance 4-48; useT3ConnectAuthPrompt 6-11; authRedirect 6-20; T3ConnectSidebarSignIn 9-66; T3ConnectAccountPages 12-26 | Clerk provider, appearance, sign-in entry points, account pages |
| `cloud/publicConfig.ts` | 79-106 | when T3 Connect is on in a browser |
| `hostedPairing.ts`, `packages/shared/src/remote.ts` | 34-69; 146-188 | hosted origin, pairing request parsing, token strip |
| `connection/onboarding.ts`, `packages/client-runtime/src/connection/onboarding.ts` | 13-24; 77-88, 101-169 | `connectPairing` (single-flight command) → `registerPairing`: descriptor fetch, known-address and protocol checks, bearer exchange, registration |
| `lib/chunkReloadGuard.ts` | 1-26 | one automatic reload per tab when a lazy chunk fails to preload |
| `hooks/useTheme.ts` | 271-300, 466-487 | `syncBrowserChromeTheme` (html/body background and `theme-color`), media-query listener lifetime |
| `hooks/useEnvironmentDisconnectDelay.ts`, `rpc/requestLatencyState.ts`, `hooks/useCopyToClipboard.ts` | 7-10; 6-97; 195-226 | 20 s before "Disconnect server"; slow-request thresholds; copy feedback reset |
| `components/settings/SettingsSidebarNav.tsx` | 77-86, 432-439 | "Sign in to T3 Connect" in the Settings sidebar footer |
| `components/auth/ConnectAgentSurface.tsx`, `ApproveAgentSurface.tsx`, `cloud/ConnectCliAuthSurface.tsx` | 35-46, 133-446; 100-175; 14-89 | agent and CLI authorization pages |
| `components/cloud/ConnectOnboardingDialog.tsx` | 43-60, 136-200, 233-267 | post-sign-in "Set up T3 Connect" wizard |
| `components/chat/useSystemComposerBanners.tsx`, `ChatView.logic.ts` | 82-89, 156-216; 110 | reconnecting/offline banner |
| `packages/client-runtime/src/connection/diagnosis.ts`, `presentation.ts` | 65-157, 159-200; 67-95 | diagnosis headlines and steps, status text |
| `components/ConnectionStatusDot.tsx` | 6-75 | status dot |
| `appInit.tsx` | 141-205, 245-306 | document title, app services that raise toasts |
| `components/LegacyThreadMigrationToast.ts`, `SlowRpcRequestToastCoordinator.tsx`, `HostFreezeToastCoordinator.ts`, `PermissionUpdateNotice.ts`, `KeybindingsUpdateToast.tsx`, `desktopUpdate.toast.tsx` | all | app-level toasts |
| `packages/client-runtime/src/connection/hostFreeze.ts` | 12-25 | freeze toast copy |
| `components/ThreadNotificationCoordinator.tsx`, `.logic.ts`, `threadNotifications.ts` | 62-447; 16-112; 8-135 | notifications, sounds, badge |
| `components/BackgroundThreadAnnouncer.tsx`, `backgroundThreadAnnouncement.ts`, `hooks/useLiveRegion.ts`, `packages/client-runtime/src/state/threadActivityAnnouncement.ts` | 19-91; 24-88; 6-41; 7-14, 50-60 | screen-reader announcements |
| `components/ui/toast.tsx` | 38-71, 110-116, 332-410, 692 | toast data options and layouts used here |
| `packages/contracts/src/settings.ts` | 400-403 | `notificationMode` default `off`, `inAppNotificationsEnabled` default `false` |

`AuthSurfaceShell` named in the brief does not exist in this tag (no match in `apps/web/src`). The shared shell of every auth page is `StandalonePage` (§2.2).

Live specimens: `/welcome`, `/zzz` (not found), `/x/y/z` (blank thread route), the draft hero offline, `/pair` signed-out (M\*), `/pair#token=…` with the POST aborted (M\*), `/pair?host=…#token=…` against `http://192.0.2.1:9` (a reserved documentation address) (M\*), `/` with the session request aborted (M\*), `/` with the main chunk aborted (M\*), toasts raised locally through the shipped `toastManager` (`toast-BKQ7E-dN.js` export `i`) with placeholder text. Screenshots are in the private reference folder.

## 2. Structure

### 2.1 Which surface the window shows

The root route decides the auth gate before anything renders (`routes/__root.tsx:62-113`):

| Gate status | When | Result |
|---|---|---|
| `guest` | path `/guest` | guest page only (out of scope) |
| `hosted-pairing` | path `/pair` and the URL has `host` and `token` (`hasHostedPairingRequest`, hostedPairing.ts:51-69) | `HostedPairingRouteSurface` (§2.4) |
| `hosted-static` | local environment disabled (desktop setting) or origin is the hosted app (hostedPairing.ts:34-49) | app shell; index shows `HostedEmptyState` when the catalog has no environment (§2.11) |
| `authenticated` | `GET /api/auth/session` says authenticated, or a URL/desktop bootstrap credential was exchanged | app shell |
| `requires-auth` | otherwise | every app route redirects to `/pair` (`_chat.tsx:49-54`, `welcome.tsx:15-20`) → `PairingRouteSurface` (§2.3). When an automatic exchange failed (a URL token outside `/pair`, or the desktop bootstrap credential), the state carries `errorMessage`, and `/pair` hands it to the form as its initial error (auth.ts:318-327, pair.tsx:54). A `requires-auth` result is not cached, so the gate resolves again on the redirect. Only the desktop credential, which is retried on every resolve, reliably arrives at `/pair` with its error; a browser URL token's error is most likely lost (S) |
| (throws) | the session request failed (any status except 502/503/504, which retry every 500ms for 15s) | `RootRouteErrorView` (§2.5) |

`/pair` itself redirects to `/` (replace) when the gate is `authenticated`, `hosted-static` or `guest` (pair.tsx:18-24). `/pair`, `/connect`, `/connect-agent` and `/approve-agent` render their route alone, without the sidebar shell or the first-run gate, whatever the gate status (`__root.tsx:157-164`).

Before React renders, `index.html` shows the boot splash (§2.7). React renders only after `router.load()` and the locale resolve (main.tsx:38-56), so the splash stays up during the session request. `bootstrap.ts:4` sends any rejection of `import("./main")` or of its `startup` promise (main chunk, the lazy Clerk shell chunk, router load) to the boot failure page (§2.8). `main.tsx` registers a `vite:preloadError` listener, so lazy chunks requested after main has loaded (the Clerk shell, route chunks) first reload the page once per tab (sessionStorage `t3code:chunk-load-reloaded`, cleared after a successful start; main.tsx:20-26, chunkReloadGuard.ts:1-26). A second Clerk-shell failure rejects `startup` and reaches the boot failure page; a route chunk failure surfaces through the router's error view instead (main.tsx:44). The main bundle itself fails before that listener exists and goes straight to the boot failure page (M\*: aborting `main-*.js` showed it with no reload).

Inside the authenticated shell, `FirstRunGate` (FirstRunGate.tsx:67-181) renders nothing while its first-run decision is pending, the "Still connecting" recovery screen once the decision has stayed pending for 4,000 ms after settings hydrated (§2.9), the "Could not read settings" screen immediately whenever settings hydration fails or retries, redirects `/` to `/welcome` when the workspace is fresh, and otherwise renders the app. Toast providers sit above every branch (`__root.tsx:130-140`), so toasts survive branch switches.

Main-column empty states (inside the shell, sidebar visible):

| Route state | Shows | Source |
|---|---|---|
| `/`, a non-archived project exists | nothing; a new draft opens in the most recently updated project (`replace`) | _chat.index.tsx:36-88 |
| `/`, draft start threw | `DraftStartError` (§2.12) | _chat.index.tsx:77-87 |
| `/`, no projects, all shells bootstrapped | `NoProjectsHero` (§2.10) | _chat.index.tsx:89-90 |
| `/`, no projects, still bootstrapping | nothing | _chat.index.tsx:90 |
| `/` on the hosted app, catalog ready, 0 environments | `HostedEmptyState` (§2.11) | _chat.index.tsx:28-31 |
| `/welcome` without a primary environment, or after Done | `NoProjectsHero` | welcome.tsx:45 |
| thread route whose `ChatView` has no thread or draft | `NoActiveThreadState` (§2.13) | ChatView.tsx:3195-3197 |
| thread confirmed missing | navigate to `/` when its environment has any thread; otherwise a blank main (M: `/x/y` and `/x/y/z` both rendered an empty `main[data-slot=sidebar-inset]`, 1184 wide) | ThreadRouteView.tsx:143-155, 216-225 |
| `/draft/<id>` with no draft session and no promotion redirect, or a thread hidden by the tag filter | `<Navigate to="/" replace>` | ThreadRouteView.tsx:159-212 |
| no route matched | `RootRouteNotFoundView` (§2.14) | __root.tsx:115-128 |

### 2.2 StandalonePage and StandalonePageHeader

Used by the pairing pages, the root error view, the agent and CLI authorization pages and the hosted prompt-link landing (standalone-page.tsx:5-36).

```
div  flex h-full items-center-safe justify-center overflow-y-auto scrollbar-gutter-both bg-background px-4 py-10 text-foreground sm:px-6
└─ section  dialog-glass w-full max-w-xl rounded-2xl border p-6                 the card
   ├─ div  flex items-baseline justify-between gap-3 text-muted-foreground       header row
   │  ├─ span role=img aria-label="T3 Code"  flex items-baseline gap-1.5
   │  │  ├─ svg T3Wordmark aria-hidden  h-3 w-auto shrink-0 text-foreground      "T3" glyph, fill currentColor
   │  │  └─ span  text-lg font-medium tracking-tight                              "Code"
   │  └─ [context] span  text-sm                                                  e.g. "Agent sign-in"
   ├─ h1  mt-6 text-xl leading-tight font-semibold                                title
   ├─ p   mt-2 text-sm text-muted-foreground                                      description
   └─ children
```

The page div owns scrolling because `#root` cannot scroll. `items-center-safe` keeps an oversized card top-aligned instead of clipping it.

### 2.3 Pairing form (`/pair`, `PairingRouteSurface`)

PairingRouteSurface.tsx:85-136:

```
StandalonePage
├─ StandalonePageHeader  title "Pair with this environment"  description describeAuthGate(bootstrapMethods)
├─ form  mt-6 space-y-4   (submit → submitServerAuthCredential)
│  ├─ div  space-y-2
│  │  ├─ label for=pairing-token  text-sm font-medium                  "Pairing token"
│  │  └─ Input id=pairing-token nativeInput  autoCapitalize=none autoComplete=off autoCorrect=off spellCheck=false
│  │        placeholder "Paste a one-time token or pairing secret"  disabled while submitting
│  ├─ [error] Alert variant=error > AlertDescription  message   (initial value: the gate's errorMessage, PairingRouteSurface.tsx:40)
│  └─ div  flex flex-wrap gap-2
│     ├─ Button size=sm type=submit                "Continue" | "Pairing..."   disabled while submitting
│     └─ Button size=sm variant=outline            "Reload app" (window.location.reload)   disabled while submitting
└─ div  mt-6 rounded-lg border border-border/70 bg-background/55 px-3 py-3 text-xs leading-relaxed text-muted-foreground
         describeSupportedMethods(bootstrapMethods)
```

`PairingPendingSurface` (18-27) is the route's `pendingComponent`: the header only, title "Pairing with this environment", description "Validating the pairing link and preparing your session.".

### 2.4 Hosted pairing link page (`HostedPairingRouteSurface`)

PairingRouteSurface.tsx:209-254. Reached with `/pair?host=<backend>&label=<name>#token=<one-time code>` (`buildHostedPairingUrl`, hostedPairing.ts:71+):

```
StandalonePage
├─ StandalonePageHeader  title "Pairing backend" | "Pairing failed" | "Backend paired"   description = message
├─ [request] div  mt-5 rounded-lg border border-border/70 bg-background/55 px-3 py-3 text-xs leading-relaxed text-muted-foreground
│     "Host: " + span.font-mono.text-foreground/80  host
├─ [error] Alert variant=error className=mt-5 > AlertDescription
│     "Verify the backend is reachable from this browser, supports CORS for hosted clients, and is served over HTTPS when opening this page from HTTPS."
└─ div  mt-6 flex flex-wrap gap-2
   ├─ pairing:   Button size=sm disabled "Pairing..."
   ├─ can retry: Button size=sm "Try again"
   └─ paired:    Button size=sm variant=outline "Open app"   (location.href = "/")
```

### 2.5 Root error view

__root.tsx:234-274, shown for any error thrown while loading routes (most often the session request):

```
StandalonePage
├─ StandalonePageHeader  title "Something went wrong."  description errorMessage(error)
├─ div  mt-5 flex flex-wrap gap-2
│  ├─ Button size=sm "Try again"                  router.invalidate()
│  ├─ Button size=sm variant=outline "Reload app"  window.location.reload()
│  └─ CopyErrorButton: Button size=sm variant=outline  [CopyIcon] "Copy error" | [CheckIcon.text-success] "Copied"
└─ div  mt-5 overflow-hidden rounded-lg border border-border/70 bg-background/55
   ├─ p    px-3 py-1.5 text-xs font-medium text-muted-foreground          "Error report"
   └─ pre  max-h-64 overflow-auto border-t border-border/70 bg-background/80 px-3 py-2 text-xs whitespace-pre-wrap text-foreground/85
```

The report (`errorReport`, 307-321) is `"<app display name> <version>"`, `"Path: <router pathname>"` (no query, so tokens stay off the clipboard), `"Time: <ISO>"`, a blank line, the stack (or message, or JSON), then up to 5 `"Caused by:"` blocks.

### 2.6 T3 Connect sign-in (Clerk) and its entry points

T3 does not draw its own sign-in form. It loads Clerk lazily and opens Clerk's prebuilt `SignIn` modal, themed with `clerkAppearance`:

- Provider: `BrowserManagedAuthShell` (`ClerkProvider` + `ManagedRelayAuthProvider`) in browsers, `ElectronManagedAuthShell` (adds `passkeys` from `@clerk/electron/passkeys`) in the desktop app. Mounted only when a publishable key is built in and `hasCloudPublicConfig()` holds (main.tsx:31-36).
- `hasCloudPublicConfig()` (publicConfig.ts:79-106) needs the Clerk key, JWT template and relay URL. In a browser with a `pk_live_` key it is true only on `https:`, no port, at the Clerk production domain or a subdomain. **So on `http://127.0.0.1:3773` T3 Connect is off**: no Clerk, no sign-in button (M: `window.Clerk` undefined, `BrowserManagedAuthShell` chunk never loaded). It is on in the desktop app and on `app.t3.codes`.
- Opening: `useT3ConnectAuthPrompt().openAuthPrompt()` = `clerk.openSignIn(resolveClerkSignInProps(location.href, isElectron))` (useT3ConnectAuthPrompt.tsx:6-11). In the browser both `forceRedirectUrl` and `signUpForceRedirectUrl` are the current URL. In Electron they are the current URL with pathname `/` and no search, so the hash route survives (authRedirect.ts:6-20).

Entry points (all call the same prompt):

| Where | Element | Source |
|---|---|---|
| Settings sidebar footer, desktop widths (`max-md:hidden`) | `SidebarMenu > SidebarMenuItem > SidebarMenuButton` with `LogInIcon` + "Sign in to T3 Connect"; hidden until Clerk loads and while signed in | T3ConnectSidebarSignIn.tsx:50-66, SettingsSidebarNav.tsx:432-439 |
| same place, signed in | Clerk `UserButton`, `avatarBox: size-7`, `userButtonTrigger: mb-1! rounded-lg! p-1! hover:bg-sidebar-row-hover!`, plus account pages "Mobile clients", "T3 Connect", "T3 Chat" (icons `smartphone`, `server`, `message-square`, 16) | T3ConnectSidebarSignIn.tsx:21-48, T3ConnectAccountPages.tsx:12-26 |
| hosted home, step 1 | Button size=sm "Sign in", disabled until Clerk is loaded | HostedEmptyState.tsx:85-87 |
| Welcome page, Phone step 2 | Button size=xs outline "Sign in to T3 Connect" | WelcomePage.tsx:1027-1034 |
| Use in another app (MCP) dialog, Labs settings, saved environment rows | their own buttons | UseInAnotherAppDialog.tsx:364, LabsSettingsPanel.tsx:81, SavedEnvironmentRows.tsx:541 |
| `/approve-agent`, `/connect` | "Sign in" Button (default size) and auto-open on load | ApproveAgentSurface.tsx:131-144, ConnectCliAuthSurface.tsx:28-50, 80-86 |

Clerk's modal tree with the element keys T3 styles (measured on H):

```
div.cl-modalBackdrop + "dialog-backdrop z-50"          fixed, flex, align-items flex-start, justify center
└─ div.cl-modalContent                                 margin 64px auto
   └─ div.cl-cardBox + "dialog-glass rounded-2xl border"
      ├─ div.cl-card + "border-0 bg-transparent shadow-none"     padding 32/40, column, gap 32
      │  ├─ button.cl-modalCloseButton aria-label="Close modal"
      │  ├─ div.cl-header + "items-start text-start"
      │  │  ├─ h1.cl-headerTitle + "text-xl font-semibold"        "Sign in to <application name>"
      │  │  └─ p.cl-headerSubtitle + "text-sm text-muted-foreground"  "Welcome back! Please sign in to continue"
      │  ├─ div.cl-main  (gap 24)
      │  │  ├─ div.cl-socialButtonsRoot > div.cl-socialButtons (grid) > button.cl-socialButtonsIconButton ×N
      │  │  ├─ div.cl-dividerRow: div.cl-dividerLine, p.cl-dividerText "or", div.cl-dividerLine
      │  │  └─ form.cl-form
      │  │     ├─ div.cl-formFieldRow > div.cl-formField: label.cl-formFieldLabel "Email address", input.cl-formFieldInput#identifier-field
      │  │     ├─ div.cl-formFieldRow (password, hidden: opacity 0, height 0, margin-top -16 until Clerk needs it)
      │  │     └─ button.cl-formButtonPrimary + authFlowElements   "Continue"  (svg.cl-buttonArrowIcon + "hidden")
      │  └─ div.cl-footerAction > a "Use passkey instead"
      └─ div.cl-footer + "bg-transparent bg-none"
         ├─ div.cl-footerAction: span.cl-footerActionText "Don’t have an account?" + a.cl-footerActionLink "Sign up"
         └─ div.cl-footerItem: "Secured by" + Clerk logo
```

`clerkAppearance` (clerkAppearance.ts:4-48), verbatim in effect:

| Key | Value |
|---|---|
| `cssLayerName` | `clerk`: Clerk's CSS goes in the `clerk` cascade layer, below `components` and `utilities` (index.css:2), so the element classes win |
| `options.shimmer` | `false` |
| `variables.borderRadius` | `0.5rem` |
| `colorPrimary` | `var(--update-foreground)`: `#1b4ed8 / #50a2ff`. Clerk uses it for filled buttons and bare links |
| `colorPrimaryForeground` | `var(--card)`: `#ffffff / #111111` |
| `colorDanger`, `colorSuccess`, `colorWarning` | `var(--error)` `#fb2c36 / #fb414a`; `var(--success)` `#00bc7d`; `var(--warning)` `#fe9a00` |
| `colorNeutral`, `colorForeground`, `colorInputForeground` | `var(--contrast-foreground)`: `#18181b / #f5f5f5` |
| `colorMuted` | `color-mix(in srgb, var(--card) 98%, var(--contrast-foreground))`: ≈ `#fafafa / #161616` (S, computed) |
| `colorMutedForeground` | `var(--contrast-muted-foreground)`: `#71717b / #a1a1a1` |
| `colorBackground` | `var(--card)` |
| `colorInput` | `var(--secondary)`: `#fafafa / #ffffff08` |
| `colorRing` | `var(--ring)`: `#1b4ed8 / #346bf1` |
| `elements.modalBackdrop` | `dialog-backdrop z-50` |
| `elements.cardBox` | `dialog-glass rounded-2xl border` |
| `elements.card` | `border-0 bg-transparent shadow-none` |
| `elements.footer` | `bg-transparent bg-none` |
| `elements.header` | `items-start text-start` |
| `elements.headerTitle` | `text-xl font-semibold` |
| `elements.headerSubtitle` | `text-sm text-muted-foreground` |
| `elements.buttonArrowIcon` | `hidden` |
| `formFieldErrorText`, `otpCodeFieldErrorText` | `color: var(--error-foreground)` `#c10007 / #ff6467` |
| `formFieldWarningText` | `color: var(--warning-foreground)` `#bb4d00 / #ffb900` |
| `formFieldSuccessText`, `otpCodeFieldSuccessText` | `color: var(--success-foreground)` `#007a55 / #00d492` |
| `signIn.elements.formButtonPrimary`, `signUp.elements.formButtonPrimary` | `border-primary bg-primary bg-none text-primary-foreground shadow-xs shadow-primary/24 hover:bg-primary/90 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background` (auth flows only; UserProfile keeps Clerk's own primary for destructive confirmations) |

The later steps of the flow are Clerk's, themed by the same appearance: after Continue with an email, Clerk shows the email-code step (six OTP boxes, a resend link with a countdown, "Use another method") or the password step ("Forgot password?"), with field errors under the field in `--error-foreground`. They were not reached (§7).

### 2.7 Boot splash (`index.html`)

```
div#root
└─ div#boot-shell                      flex, centred, min-height 100%, background/colour inherit (theme: --boot-background/--boot-foreground)
   └─ div#boot-shell-card aria-label="T3 Code splash screen"     96×96, centred, z-index 1
      └─ img#boot-shell-logo src=/apple-touch-icon.png alt="T3 Code"   64×64, object-fit contain
```

React replaces it on first render.

### 2.8 Boot failure page (`showBootError`)

lib/bootError.ts:257-395 builds plain DOM into `#boot-shell` (styles index.html:443-668):

```
div#boot-error
├─ div role=alert
│  ├─ h1  "T3 Code failed to start"
│  └─ p   "We are not sure why. Try a reload first." | offline (web only, navigator.onLine false): "This browser appears to be offline. Check the connection, then reload."
├─ div#boot-error-actions
│  ├─ button#boot-error-reload  "Reload"
│  └─ button#boot-error-update hidden   desktop only: "Download update" | "Downloading update (N%)" (aria-disabled) | "Restart to update"
├─ p#boot-error-update-note hidden       "The update download failed. Try Download update again." | "The update install failed. Try Restart to update again."
├─ details (open when the last failure in this tab was < 5 min ago)
│  ├─ summary  "More options if it keeps failing"
│  ├─ [desktop] a  "Download and reinstall T3 Code" | nightly: "Download and reinstall the latest nightly build"
│  ├─ div
│  │  ├─ p  "Run this on the computer that runs T3 Code. Your agent can check the local logs and help you file a bug report."
│  │  └─ div#boot-error-command > code "npx t3 triage" (desktop: "t3 triage" or "<launcher> triage") + button.boot-error-icon title/aria "Copy command" (14px copy icon)
│  └─ div
│     ├─ pre#boot-error-details tabindex=0   the report
│     ├─ p  "Details may include local file paths and names. Check before sharing."
│     └─ div#boot-error-report > button "Copy error details" + a "File a bug report on GitHub"
└─ span.boot-error-sr aria-live=polite     announcements
```

Report (buildErrorReport, 96-120): the error and up to 3 causes (`Caused by: …`), URLs and query/hash tokens redacted, truncated at 6,000 chars with `[truncated]`, then `Version`, `Channel`, `Surface` (desktop/web), `Platform`, `User agent`, `Time`.

Copy feedback (bootError.ts:188-199 `copyText`, 269-274 `announce`, 297-320, 330-343; S). Copying uses the Clipboard API only in a secure context. Otherwise it selects the text and runs `execCommand("copy")`. None of these feedback states resets.

- When the command copies, its icon becomes a 14px `check` (stroke 2), `title` and `aria-label` become "Copied", and the page announces "Copied to clipboard.".
- When the copy is blocked, the button's content becomes the hint "Press ⌘C to copy" (Mac/iPhone/iPad) or "Press Ctrl+C to copy", drawn in the button's 12px / 400 style (which is why `.boot-error-icon` sets a font, `min-width` and side padding). The command text stays selected, and the page announces "Copy was blocked. The text is selected so you can copy it manually.".
- "Copy error details" becomes "Copied", or shows the same hint.
- Each announcement clears the `aria-live` span and writes the text 100 ms later, so a repeat is spoken.

"File a bug report on GitHub" opens a new tab (`target=_blank rel=noreferrer`) at `https://github.com/pingdotgg/t3code/issues/new` with `template=bug_report.yml`, `title=T3 Code failed to start`, `area=apps/web`, `version=<version> (<channel>)`, `environment=<surface>, <platform>` (bootError.ts:345-352). The desktop reinstall link goes to `https://t3.codes/download` (nightly: `?channel=nightly`).

### 2.9 First-run recovery

FirstRunGate.tsx:183-221, full window (no shell):

```
main  flex h-dvh min-h-0 items-center justify-center bg-background px-6 text-foreground
└─ div  flex max-w-sm flex-col items-center text-center
   ├─ h1  text-lg font-semibold                  "Still connecting" | "Could not read settings"
   ├─ p   mt-2 text-sm text-muted-foreground     "T3 Code could not confirm this workspace." | "Your saved settings could not be loaded."
   └─ Button size=sm variant=outline mt-5  [RefreshIcon refreshing={retrying}] "Reload" (location.reload) | "Retry" (re-read settings; disabled while retrying)
```

### 2.10 No projects hero (`NoProjectsHero`)

NoProjectsHero.tsx:13-59:

```
SidebarInset role=main data-workspace-area=home data-workspace-order=1 aria-label="Home" tabindex=-1  h-dvh min-h-0 overflow-hidden overscroll-y-none
└─ div  flex min-h-0 min-w-0 flex-1 flex-col overflow-x-hidden bg-background
   ├─ [desktop] WorkspacePageHeader electron      empty drag strip (the window moves only where CSS opts in)
   └─ Empty size=hero  flex-1
      └─ div  w-full max-w-lg px-8 py-12
         └─ EmptyHeader  max-w-none
            ├─ EmptyTitle          "What should we work on?"
            ├─ EmptyDescription    "Add a project, or start without one." | no scratch target: "Add a project to start your first thread."
            └─ div  mt-6 flex justify-center gap-2
               ├─ Button size=sm  [plus 16] "Add project"                         → command palette "add-project"
               └─ [scratch target] Button size=sm variant=outline [message-square-dashed 16] "Start without a project"
```

### 2.11 Hosted home (`HostedEmptyState`)

HostedEmptyState.tsx:21-163. Only on the hosted app (or with the local environment switched off) when no computer is in the catalog:

```
SidebarInset  h-dvh min-h-0 overflow-hidden overscroll-y-none
└─ div  flex min-h-0 min-w-0 flex-1 flex-col overflow-x-hidden bg-background
   ├─ WorkspacePageHeader  border-b border-border  > div.flex.items-center.gap-2 > span.text-sm.font-medium.text-foreground.md:text-muted-foreground/60  <app display name>
   └─ Empty (default)  flex-1
      ├─ EmptyHeader
      │  ├─ EmptyTitle        "No computer connected" | local env off: "This computer is off"
      │  └─ EmptyDescription  cloud on: "Your agents run on your computer." | cloud off: "Your agents run on your computer. Start T3 Code there first." | off: "Turn it back on, or connect a different computer."
      └─ EmptyContent
         ├─ local env off: Button size=sm (Link /settings/connections) "Open Environments"
         └─ ol  w-full space-y-4 text-left
            SetupStep = li.flex.gap-3 > span.w-2.shrink-0.pt-1.text-muted-foreground <n> + div.min-w-0.flex-1
            cloud on (T3ConnectSteps):
              1  signed in: p.flex.items-center.gap-1.5.text-muted-foreground [check 16] "Signed in"
                 signed out: Button size=sm "Sign in" (disabled until Clerk loaded)
              2  RunOnComputer("t3 connect")
                 + signed in: div.mt-4.space-y-1.5 > CloudEnvironmentConnectRows (refreshWhileEmpty, auto-select)
                              empty: p.text-xs.text-muted-foreground "Waiting for your computer…"
            cloud off (PairingSteps):
              1  RunOnComputer("t3 pair")
              2  Button size=sm (Link /settings/connections) "Add pairing link"
RunOnComputer = p "Run on your computer:" + InstallT3Commands(command, mt-2) + a.mt-2.inline-block.text-xs.text-muted-foreground.underline.underline-offset-2.hover:text-foreground "Or download the desktop app" (https://t3.codes/download, new tab)
InstallT3Commands = div.space-y-2 > p.text-xs.text-muted-foreground "macOS, Linux", CommandBlock "curl -fsSL https://t3.codes/install.sh | sh",
                    p "Windows (PowerShell)", CommandBlock "irm https://t3.codes/install.ps1 | iex", p "Then", CommandBlock <command>,
                    p "Skip the install if you already have `t3`. If `t3` is not found after installing, add the folder the installer names to your PATH."
CommandBlock = div.flex.items-center.justify-between.gap-3.rounded-lg.border.border-border.bg-muted/50.font-mono.px-3.py-2.5.text-sm
               > span.min-w-0.truncate (span.mr-2.text-muted-foreground "$" + command) + Button size=icon-xs variant=ghost aria-label "Copy command" [copy 14 | check 14 for 1,500 ms]
```

`CloudEnvironmentConnectRows` in selection mode renders, per discovered computer, `label.flex.cursor-pointer.items-center.gap-3.rounded-lg.border.border-border.bg-background.px-3.py-2.5` with a Checkbox, the machine icon (16, muted), the label (`text-sm font-medium` truncating, optional `text-xs` muted detail) and a right status (`max-w-[45%] text-xs`, muted or destructive): "Connecting…", "Connected", "Reconnecting…", "Client not supported", "Connection failed", "Offline", "Not connected", or for undiscovered states "Available", "Offline", "Unavailable", "Checking…" (CloudEnvironmentConnectList.tsx:368-420; cloudEnvironmentConnectionPresentation.ts:12-59). A discovery problem shows "Could not load T3 Connect environments" (`text-sm font-medium text-destructive`) + reason + outline sm "Try again" (284-307).

### 2.12 Draft start error (`DraftStartError`)

_chat.index.tsx:93-118:

```
SidebarInset role=main data-workspace-area=home data-workspace-order=1 aria-label="Home" tabindex=-1  h-dvh min-h-0 overflow-hidden overscroll-y-none
├─ [desktop] WorkspacePageHeader electron
└─ Empty (default) flex-1
   └─ EmptyHeader  max-w-md
      ├─ EmptyTitle  "Couldn’t start a new thread"
      ├─ EmptyDescription  "The project is still available. Try opening the draft again."
      └─ div  mt-5 flex justify-center > Button size=sm [RefreshIcon size=md (refresh-cw 16)] "Try again"
```

### 2.13 No active thread (`NoActiveThreadState`)

NoActiveThreadState.tsx:6-35:

```
SidebarInset  h-dvh min-h-0 overflow-hidden overscroll-y-none
└─ div  flex min-h-0 min-w-0 flex-1 flex-col overflow-x-hidden bg-background
   ├─ WorkspacePageHeader electron={isElectron}  border-b border-border
   │  ├─ desktop: span.text-xs.text-muted-foreground/50  "No active thread"
   │  └─ web:     div.flex.items-center.gap-2 > span.text-sm.font-medium.text-foreground.md:text-muted-foreground/60  "No active thread"
   └─ Empty (default) flex-1
      └─ div  w-full max-w-lg px-8 py-12
         └─ EmptyHeader  max-w-none
            ├─ EmptyTitle        "Pick a thread to continue"
            └─ EmptyDescription  "Select an existing thread or create a new one to get started."
```

`WorkspacePageHeader` (WorkspacePageHeader.tsx:16-24): `flex h/min-h/max-h-(--workspace-topbar-height)! shrink-0 items-center gap-3 pl-(--workspace-gutter-start) pr-(--workspace-gutter-end)`, padding transition with the panel animations, `drag-region` + `wco:pr-(--workspace-native-controls-inset)` on desktop, and `pl-(--workspace-titlebar-content-left)` when the sidebar is collapsed (or below 768px).

### 2.14 Page not found

__root.tsx:115-128, rendered in the shell's outlet next to the sidebar, with no inset surface of its own:

```
main  flex min-h-0 min-w-0 flex-1 items-center justify-center p-6
└─ div  flex max-w-sm flex-col items-center gap-4 text-center
   ├─ h1  text-lg font-medium text-foreground     "Page not found"
   ├─ p   text-sm text-muted-foreground           "This link doesn't point to a page in <app display name>. Go home to choose a project or start a thread."
   └─ Button (default size) render=Link to="/" replace   "Go home"
```

### 2.15 Welcome page (`/welcome`)

WelcomePage.tsx:124-324. The shell renders without the sidebar (`AppSidebarLayout showSidebar={!isWelcome}`, __root.tsx:202):

```
SidebarInset role=main aria-label="Welcome" data-workspace-area=home  h-dvh min-h-0 overflow-hidden overscroll-y-none
└─ div  flex min-h-0 min-w-0 flex-1 flex-col bg-background
   ├─ [desktop] WorkspacePageHeader electron
   └─ div  min-h-0 flex-1 overflow-y-auto scrollbar-gutter-both
      └─ div  mx-auto flex min-h-full w-full max-w-3xl flex-col px-6 pt-12 pb-8 sm:pt-20
         ├─ h1  text-2xl font-semibold tracking-tight text-foreground       "Welcome to T3 Code"
         ├─ p   mt-1.5 text-sm text-muted-foreground                        "Setting up " + span.text-foreground <computer label | "this computer"> + "."
         ├─ div  mt-10 border-t border-border/60
         │  ├─ WelcomeSection "Agents"    (AgentsSection)
         │  ├─ WelcomeSection "Projects"  (ProjectsSection; hidden when the scan finds nothing)
         │  └─ [phone offered] WelcomeSection "Phone"
         ├─ div  mt-6 flex flex-wrap items-center justify-between gap-3
         │  ├─ agent ready: span (spacer) | else Button variant=ghost-muted "Start without an agent"
         │  └─ div  flex items-center gap-3
         │     ├─ [no agent] span.text-xs.text-muted-foreground  "Set up one agent to start."
         │     └─ Button (default) disabled until an agent is ready or while importing:
         │        "Start" | "Import N project(s) and start" | "Import threads and start" | "Importing…" | "Starting…"
         └─ p  mt-auto pt-12 text-2xs leading-relaxed text-muted-foreground/70   telemetry notice + a "privacy policy"
WelcomeSection = section.grid.gap-x-10.border-b.border-border/60.py-4.sm:grid-cols-[6rem_minmax(0,1fr)] > h2.text-sm.leading-10.text-muted-foreground + div.min-w-0
RowIcon        = span.flex.size-5.shrink-0.items-center.justify-center.text-muted-foreground
```

AgentsSection rows (1129-1192): `div.flex.items-center.gap-3` (`min-h-10 py-1`, compact `min-h-9`) > RowIcon(provider icon 18, compact 16) + `div.min-w-0.flex-1` (name `block truncate text-sm text-foreground` + `font-medium` when not compact; detail `p.line-clamp-2.text-xs.leading-relaxed.break-words.text-muted-foreground`) + `div.shrink-0` (ready: `span.inline-flex.items-center.gap-1.5.text-xs.font-medium.text-success-foreground` [check 14] "Ready"; checking: `span.text-xs.text-muted-foreground` "Checking…"; else one action Button size=xs outline: "Turn on", "Update", "Set up", "[terminal 14] Install", "[terminal 14] Sign in", "Sign in", "Details"). While checking with nothing found: `p.text-sm.leading-10.text-muted-foreground` "Checking <label>…". Permission notes: `p.pl-8.text-xs.text-muted-foreground` "This connection cannot control terminals." / "This connection cannot manage agents.". Then a Collapsible: trigger `group flex min-h-10 w-full items-center gap-3 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset` > RowIcon[plus 16] + `span.text-sm` "Add more agents" (muted, foreground on hover) or "Set up an agent" (foreground) + closed: logo stack `span.flex.items-center.gap-2.opacity-60.grayscale` (≤6 provider icons 14 + ACP icon) + `chevron-down 16 ml-auto text-muted-foreground group-data-panel-open:rotate-180`; panel: `grid grid-cols-1 gap-x-8 sm:grid-cols-2` of compact rows, then "More agents in the ACP Registry" row with outline xs "Browse". An install/sign-in terminal card can open below (`mt-4 overflow-hidden rounded-lg border border-border/70 bg-background`, 256px terminal, see 09-terminal).

ProjectsSection (746-907): pending `p.text-sm.leading-10.text-muted-foreground` "Looking for projects in your Claude Code, Codex, and Pi history…"; error row with ghost sm "Retry"; otherwise `div.flex.min-h-10.items-center.gap-3.py-1` > label (RowIcon[Checkbox] + title `block text-sm font-medium text-foreground` "Import projects and their threads" / "Import N projects and their threads" + `block text-xs text-muted-foreground` "From your Claude Code, Codex, and Pi history") + ghost-muted xs "Review all N" / "Hide" [chevron-down 14, rotated when open]. Review list: counts row, ghost xs "Select all"/"Select none", scroll area `max-h-80` of candidate rows (`flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 hover:bg-muted/40`, nested `pl-8`) with a meta grid `grid-cols-[1rem_1rem_1rem_2.5rem_2.25rem] gap-x-1 text-xs text-muted-foreground tabular-nums` (source icons 12, thread count, relative age). Without operate scope: `p.pt-1.pl-8.text-sm.text-destructive` "This connection cannot import projects or thread history.".

PhoneSection (909-960): row `flex min-h-10 items-center gap-3 py-1` > RowIcon[smartphone 16] + `span.min-w-0.flex-1.text-sm.font-medium.text-foreground` "Use T3 Code on your phone" + ghost-muted xs "Set up"/"Hide" [chevron-down 14]; panel `flex items-center gap-8 pt-2 pb-3 pl-8`: two steps (`li.flex.gap-3`, number `w-3 text-sm text-muted-foreground tabular-nums`) and a QR code (`rounded-lg bg-white p-1.5`, 124px, level M, margin 2, encodes `https://t3.codes/mobile`). Step 2 states: "Checking T3 Connect…", "Could not check T3 Connect." + outline xs "Check T3 Connect", "[check 16 success] Your phone can reach this computer. Sign in to the app with the same account.", "T3 Connect is on for this computer. …", "Signed in. Share this computer so the app can reach it." + "Set up T3 Connect", or signed out "Sign in to T3 Connect here, then in the app with the same account." + outline xs "Sign in to T3 Connect" (962-1035).

### 2.16 Agent and CLI authorization pages

All are `StandalonePage`s (header context in the right of the header row):

| Route | States (title / description) | Body | Source |
|---|---|---|---|
| `/connect-agent` | "Checking the sign-in request" / "One moment while this environment verifies the agent's request." + Spinner lg muted (20, `mt-6`); "This sign-in cannot continue" / reason + p `mt-4 text-sm text-muted-foreground` "Close this page and start the sign-in again from your agent."; ready "Connect <client>" / "This agent wants to use the threads in every project on **<host>**." | `dl` `mt-4 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 text-sm` (App · name set by the app; Access goes to <host> + "on the computer that opened this page" or "· whoever runs it gets this access"); note `mt-2 text-xs` "Only approve a sign-in you just started."; form `mt-6 space-y-5`: "Sign-in code" Input (placeholder "Paste a code", `autoComplete=one-time-code`) + help `text-xs`; "What it may do" RadioGroup (`flex flex-col gap-3`) of AccessOption cards; error Alert; buttons (default size) "Approve <level>" / "Choose what it may do" / "Approving…" and outline "Deny" / "Denying…" | ConnectAgentSurface.tsx:133-392 |
| `/approve-agent` | "This sign-in link is incomplete" / "Close this page and start the sign-in again from your agent."; "Open this page in its own window" / "For your safety, agent sign-ins cannot be approved inside another site." (framed); "Connecting to T3 Connect" / "One moment while your account loads." + Spinner; "Sign in to approve" / "Sign in to T3 Connect with the account this environment is linked to." + Button "Sign in", then the consent body with link buttons "Use a sign-in code instead" and a sign-out link | context "Agent sign-in" (also on every `/connect-agent` state, ConnectAgentSurface.tsx:392) | ApproveAgentSurface.tsx:100-195 |
| `/connect` | "This connect link is incomplete" / "The link is missing its authorization request. Re-run `t3 connect` in your terminal and open the freshly printed URL." (context "Authorization request"); "Connecting your terminal" / "Redirecting to authorize T3 Connect for your CLI…" or "Sign in to continue authorizing T3 Connect for your CLI." + Button "Sign in" | context "Browser authorization" | ConnectCliAuthSurface.tsx:14-89 |

AccessOption (ConnectAgentSurface.tsx:406-446): `RadioPrimitive.Root` `flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2.5 text-left`, selected `border-primary bg-background ring-2 ring-inset ring-primary/25` (dark `border-transparent bg-primary/10 ring-1 ring-primary/30`), unselected `border-border bg-background hover:bg-muted/50` (dark `border-transparent bg-white/[0.035] hover:bg-accent`); icon 16 `mt-0.5` primary or muted (Read only = `eye`; the others from `runtimeModeConfig`); label `text-sm font-medium`, detail `text-xs text-muted-foreground`. Approve stays disabled for 700 ms after the page shows or regains focus (`ARM_DELAY_MS`, 47, 182-200).

### 2.17 Alert (`ui/alert.tsx`)

Not in 02-primitives; specified here because the pairing pages use it.

```
div role=alert data-slot=alert data-variant  relative rounded-xl border px-3.5 py-3 text-card-foreground text-sm + surface + variant
└─ div  flex gap-2 items-center   (controlAlignment "first-line": items-start, and min-h-7 pt-1 sm:min-h-6 sm:pt-0.5 with an action)
   ├─ [icon children] div  flex shrink-0 items-center justify-center size-4 [&>svg]:size-full   (first-line: h-lh w-4 [&>svg]:size-4)
   ├─ div  flex min-w-0 flex-1 flex-col gap-0.5
   │  ├─ AlertTitle  div data-slot=alert-title  font-medium
   │  └─ AlertDescription  div data-slot=alert-description  flex flex-col gap-2.5 text-muted-foreground
   └─ [AlertAction] div  flex shrink-0 items-center self-center (first-line: h-lh self-start) > div[data-slot=alert-action].flex.gap-1
```

Children are sorted by slot (alert.tsx:31-78): `AlertTitle`/`AlertDescription` and bare text go to the content column, `AlertAction` to the action column, any other element (an icon) to the icon box.

Variants (alert.tsx:7-29): `default` `bg-transparent dark:bg-input/32 [&_svg]:text-muted-foreground`; `error` `border-error/32 bg-error-surface text-error-foreground`, description `text-error-foreground/80`, svg `text-error`; `info` `border-info/32 bg-info/4`, svg `text-info`; `success` `border-success/32 bg-success/4`, svg `text-success`; `warning` `border-warning/32 bg-warning-surface text-warning-foreground`, description `/80`, svg `text-warning`; `sidebar` `rounded-lg border-sidebar-border bg-sidebar-control-surface px-2 py-1.5 text-[11px] leading-4`, description block `text-sidebar-muted-foreground`. Surface `glass` adds `alert-glass` (01-tokens §3.12).

### 2.18 Reconnecting / offline banner

The connection-loss indicator in the main window is a composer banner, specified structurally in [07-composer §2.9](07-composer.md). On the draft hero it renders detached below the composer:

```
div[data-composer-banner-stack]  transition-opacity duration-150 ease-out starting:opacity-0 motion-reduce:transition-none
└─ div role=alert data-slot=composer-banner data-variant=warning|error data-composer-banner-surface=detached
   ::before  glass card: rounded-2xl border (variant 28%) fill attached surface at glass opacity, tint gradient (variant 8%), blur/saturate, shadow-composer
   └─ div[data-composer-banner-row] data-composer-banner-layout=wrap-actions   icon column | content | actions
      ├─ span[data-slot=composer-banner-icon]  [wifi-off, 12]
      ├─ span[data-slot=composer-banner-content]
      │  ├─ span  min-w-0 truncate font-medium leading-7 sm:leading-6   "<machine> is reconnecting" | "<machine> is offline"
      │  └─ span  w-0 min-w-0 grow truncate text-muted-foreground @max-[400px]:sr-only   diagnosis headline (§5.9)
      └─ span[data-slot=composer-banner-actions]
         ├─ [phase ≠ connecting] Button size=xs variant=ghost  "Retry now" (reconnecting) | "Reconnect" (offline/error)
         └─ [remote, after the disconnect delay] Button size=xs variant=ghost  "Disconnect server"  title "Hide this server's threads. Switch it on again in Environments."
```

Actions (useSystemComposerBanners.tsx:86-136, 160-215): "Retry now"/"Reconnect" calls the catalog's `retryNow`, or switches the environment back on when it is disabled (`entry.enabled === false`). "Disconnect server" appears once the environment has been unavailable for 20,000 ms (`DISCONNECT_DELAY_MS`, useEnvironmentDisconnectDelay.ts:7) and only for remote targets (not the primary or a desktop-local one). It is disabled while it runs, then switches the environment off and navigates to `/` (replace). Failures raise the stacked error toasts in §5.6. The title reads "is reconnecting" for both `connecting` and `reconnecting`, and "is offline" otherwise.

## 3. Measurements

### 3.1 StandalonePage (M on `/pair`, the root error view and the hosted pairing page; values identical across them)

| Part | Property | Light | Dark |
|---|---|---|---|
| page | box, padding | 1440×900, padding 40 (top/bottom) × 24 (sides, ≥640; 16 below) | = |
| page | background / text | `--background` `#fcfcfc` / `--foreground` `#18181b` | `#000000` / `#f5f5f5` |
| page | scrollbar gutter | `scrollbar-gutter: stable both-edges`: 6px reserved on each side even when nothing scrolls (M on `/welcome`, same class: `clientWidth` 1428 of 1440 with no overflow). The symmetric gutters leave the centred card at x 432 | = |
| card | width / x | 576 (`max-w-xl`) centred: x 432 at 1440 | = |
| card | padding / radius / border | 24 / 16 / 1px | = |
| card | fill | `dialog-glass`: `#fcfcfccc` | `#000000cc` |
| card | border colour | `#18181b1a` (foreground 10%) | `#ffffff14` |
| card | shadow | `0 24px 64px -24px #000000a6` | `inset 0 1px 0 #ffffff0a, 0 24px 72px -20px #000000e6` |
| card | backdrop | `blur(12px) saturate(1.14)` | `blur(16px) saturate(1.08)` |
| header row | box | 526×28, baseline aligned, gap 12 | = |
| wordmark | "T3" glyph | 19.88×12 (`h-3`, viewBox 94.394×56.96), fill `--foreground` `#18181b` | `#f5f5f5` |
| wordmark | gap glyph → "Code" | 6 (`gap-1.5`), baselines aligned | = |
| wordmark | "Code" | 18px / 28px, 500, letter-spacing −0.45px, `--muted-foreground` `#71717b` | `#a1a1a1` |
| context | text | 14px / 20px, 400, `#71717b`, right-aligned on the header baseline | `#a1a1a1` |
| h1 | type | 20px / 25px (`leading-tight`), 600, `#18181b`, margin-top 24 | `#f5f5f5` |
| p | type | 14px / 20px, `#71717b`, margin-top 8 | `#a1a1a1` |
| offsets | from card top | header 25, h1 77, description 110 | = |

### 3.2 Pairing form (M\*)

| Part | Property | Light | Dark |
|---|---|---|---|
| form | margin-top | 24 | = |
| label row | geometry | label 14px / 20px, 500, `#18181b`; it is inline, so its line box is the block's 24px and the `space-y-2` 8px margin does not apply: label line at form top 0-24, input at 24 | `#f5f5f5` |
| input box | size | 526×32 (`sm:h-7.5` input 30 + 2 border), radius 8 | = |
| input box | fill / border | `--background` `#fcfcfc` / `--input` `#d4d4d8` | `--input` 32% `#ffffff09` / `#ffffff1c` |
| input box | shadow | `0 1px 2px #0000000d` + lip `::before` `0 1px 0 #0000000a` (02-primitives §2.7) | `0 1px 2px #0000000d` + lip `0 -1px 0 #ffffff0f` |
| input | text / padding | 14px, line-height 30, padding-x 11; placeholder `--placeholder` `#71717b` | placeholder `#a1a1a1` |
| input | focus-visible (M\*, reached with Tab / Shift+Tab) | border `--ring` `#1b4ed8`, `inset 0 0 0 3px #194fd93d` (ring 24%), drop shadow and lip removed | border `#346bf1`, `inset 0 0 0 3px #326df23d` (M\*) |
| error Alert | box | 526×46 for a one-line message (M\*: "Enter a pairing token to continue."), 526×66 for two lines; margin-bottom 16 (`space-y-4`) | = |
| buttons row | gap / height | 8 / 28 | = |
| "Continue" | Button default sm | 80.52×28, padding-x 9, gap 6, 14px / 20px 500, radius 8, fill `--primary` `#1b4ed8`, text `#ffffff`, shadow `inset 0 1px #ffffff29` + `0 1px 2px #194fd93d` (primary 24%) | fill `#346bf1`, shadow `inset 0 1px #ffffff29` + `0 1px 2px #326df23d` |
| "Reload app" | Button outline sm | 94.83×28, fill `#ffffff`, border `#d4d4d8`, text `#18181b`, shadow `0 1px 2px #0000000d` | fill `#ffffff09`, border `#ffffff1c`, text `#f5f5f5` |
| info box | box | margin-top 24, radius 8, padding 12, 526×65 for two lines | = |
| info box | fill / border | `--background` 55% `#fcfcfc8c` / `--border` 70% `#e4e4e7b3` | `#0000008c` / `#ffffff0e` |
| info box | text | 12px / 19.5px (`leading-relaxed`), `#71717b` | `#a1a1a1` |
| card | height | 368 without an error (M\*); 430 with a one-line error (M\*: 368 + 46 + 16); 450 with a two-line error (derived) | = |

### 3.3 Hosted pairing page (M\*)

| Part | Light (dark from the same tokens as §3.1-3.2) |
|---|---|
| pending | title "Pairing backend", description "Connecting to this backend.", host box margin-top 20, 526×47.5, "Host:" 12px / 19.5px muted + host `font-mono` 12px `--foreground` 80% `#18181bcc` (`#f5f5f5cc`); button row margin-top 24 with "Pairing..." default sm 79.3×28 at opacity 0.64; card 274.5 tall |
| failed | description = the error, then " If the backend accepted this one-time token, request a new pairing link before retrying." (four lines, 80 tall); host box; Alert error margin-top 20, 526×86 (three lines), fill `#fb2c3614`, border `#fb2c3652`, description `#c10007cc`; "Try again" default sm 82.3×28; card 440.5 tall |

### 3.4 Root error view (M\*)

| Part | Property | Light | Dark |
|---|---|---|---|
| buttons row | margin-top / gap | 20 / 8 | = |
| "Try again" | default sm | 82.3×28 (§3.2 colours) | |
| "Reload app" | outline sm | 94.83×28 | |
| "Copy error" | outline sm | 109.73×28, `copy` icon 16 in `--muted-foreground` `#71717b`, −2px side margins | icon `#a1a1a1` |
| copied | icon / label | `check` 16 `--success` `#00bc7d`, "Copied", back to "Copy error" after 2,000 ms (`useCopyToClipboard` default timeout, useCopyToClipboard.ts:196) (S) | = |
| report box | box | margin-top 20, 526×286, radius 8, border `#e4e4e7b3`, fill `#fcfcfc8c` | `#ffffff0e`, `#0000008c` |
| "Error report" | type | 12px / 16px, 500, padding 6 × 12 (28 tall), `#71717b` | `#a1a1a1` |
| pre | box | max-height 256, scrolls, border-top 1px `#e4e4e7b3`, padding 8 × 12, fill `--background` 80% `#fcfcfccc` | `#ffffff0e`, `#000000cc` |
| pre | text | 12px / 16px `--font-mono` (Liberation Mono here, SF Mono on macOS), `--foreground` 85% `#18181bd9`, `pre-wrap` | `#f5f5f5d9` |
| card | height | 509 with a long stack | = |

### 3.5 Alert variants (P; error also M\*)

Box: radius 12, border 1px, padding 12 × 14, 14px / 20px, gap 8 icon → content, content gap 2, icon box 16. Heights: 66 for a two-line description alone, 68 with a title + one-line description.

| Variant | Fill | Border | Text | Description | Icon |
|---|---|---|---|---|---|
| default | transparent / `#ffffff09` | `--border` `#e4e4e7` / `#ffffff14` | `--card-foreground` `#18181b` / `#f5f5f5` | `#71717b` / `#a1a1a1` | `#71717b` / `#a1a1a1` |
| error | `--error-surface` `#fb2c3614` / `#fb414a29` | error 32% `#fb2c3652` / `#fb414a52` | `--error-foreground` `#c10007` / `#ff6467` | 80% `#c10007cc` / `#ff6467cc` | `--error` `#fb2c36` / `#fb414a` |
| info | info 4% `#2b7fff0a` | info 32% `#2b7fff52` | `#18181b` / `#f5f5f5` | muted | `--info` `#2b7fff` |
| success | success 4% `#00bc7d0a` | success 32% `#00bc7d52` | `#18181b` / `#f5f5f5` | muted | `--success` `#00bc7d` |
| warning | `--warning-surface` `#fe9a0014` / `#fe9a0029` | warning 32% `#fe9a0052` | `--warning-foreground` `#bb4d00` / `#ffb900` | 80% `#bb4d00cc` / `#ffb900cc` | `--warning` `#fe9a00` |
| sidebar | `--sidebar-control-surface` `#f4f4f5` / `#0a0a0a` (sidebar scope) | `--sidebar-border` `#e4e4e7` / `#ffffff14` | 11px / 16px, `--card-foreground` | `--sidebar-muted-foreground` `#71717b` / `#a3a3a3` (sidebar scope; description is `block`) | (S) |

Measured fills read back slightly off the token mix at these low alphas, for example error `#ff263314` / `#f93e4b29` and border `#fc2c3552` / `#fc414b52` (M\*). Port the token values above.

### 3.6 T3 Connect sign-in modal (H; start step, signed out)

Re-measured on 2026-11-18, when the hosted app was loading Clerk JS 6.45.0. These rows matched again: card box, card padding and gap, backdrop, close button, title, subtitle, field label, input (rest and focus), primary button (rest and hover), social button boxes, link colours, divider and footer fill, in both themes where a row gives two values. That pass corrected the primary button's shadow colour and added the measured dark focus ring and hover fill.

| Part | Property | Light | Dark |
|---|---|---|---|
| backdrop | fill / blur | `dialog-backdrop`: `--background` 60% `#fcfcfc99` (01-tokens §3.12) | `#000000a3` (M on H) |
| backdrop | layout | fixed, flex, `align-items: flex-start`, centred horizontally, z 50 | = |
| modal content | placement | margin 64 top and bottom: card top at y 64 | = |
| cardBox | box | 400 wide (Clerk default), 517.89 tall in this step | = |
| cardBox | surface | same as the standalone card: `#fcfcfccc`, border `#18181b1a`, radius 16, `0 24px 64px -24px #000000a6`, `blur(12px) saturate(1.14)` | `#000000cc`, `#ffffff14`, `inset 0 1px #ffffff0a, 0 24px 72px -20px #000000e6`, `blur(16px) saturate(1.08)` |
| card | padding / gap | 32 top/bottom, 40 sides; column gap 32 (header → main → passkey link) | = |
| close button | box | 40×40 at 8 from the top and right edges, padding 12, radius 8; X 16 (Clerk icon, stroke 1.5) `--muted-foreground` `#71717b` | `#a1a1a1` |
| header | title | `h1` 20px / 28px, 600, `#18181b`, left-aligned | `#f5f5f5` |
| header | subtitle | 14px / 20px, `#71717b`, 4 below the title | `#a1a1a1` |
| main | gap | 24 (social → divider → form) | = |
| social buttons | grid | one row of N equal buttons, gap 8; with 4 providers each 74×32 | = |
| social button | box | radius 8, padding 7 × 12, 13px / 18px 500, colour foreground 62% `#18181b9e` | `#f5f5f59e` |
| social button | outline | `0 0 0 1px #1c1c1c12, 0 2px 3px -1px #00000014, 0 1px 0 #00000005` | `0 0 0 1px #f1f1f112, …` |
| social icons | size | 16; monochrome marks (Apple, GitHub) painted in `#18181b`, multicolour marks (Google, Microsoft) as-is | `#f5f5f5` |
| divider | line / text | 1px `#1c1c1c12` each side; "or" 13px / 18px `#71717b`, 16 margins; row 18 tall | `#f1f1f112`; `#a1a1a1` |
| form | gap | 32 (field → button) | = |
| field label | type | 13px / 18px, 500, `#18181b`; 8 above the input | `#f5f5f5` |
| input | box | 320×32, radius 8, padding 7 × 12, 13px / 18px; fill `--secondary` `#fafafa`; outline drawn as `0 0 0 1px #1b1b1b1c, 0 0 1px #1c1c1c12`; placeholder "Enter your email address" | fill `#ffffff08`; `0 0 0 1px #f6f6f61c, 0 0 1px #f1f1f112` |
| input | focus | `0 0 0 1px #19191d47, 0 0 1px #1b1b1b1c, 0 0 0 4px #1b4ed8` (a solid 4px ring in `--ring`, because T3 sets `colorRing` opaque) | `0 0 0 1px #f4f4f447, 0 0 1px #f6f6f61c, 0 0 0 4px #346bf1` (M on H) |
| primary button | box | 320×32, radius 8, padding 7 × 12, 13px / 18px 500, border width 0; fill `--primary` `#1b4ed8`, text `#ffffff`, shadow `0 1px 2px #194fd93d` (`shadow-xs shadow-primary/24`, no inset highlight); hover `bg-primary/90` `#1b4ed8e6` (M on H); arrow hidden | `#346bf1`, `0 1px 2px #326df23d` |
| "Use passkey instead" | link | 13px / 18px, 500, `colorPrimary` `#1b4ed8`, centred | `#50a2ff` |
| footer action | box / type | padding 16 × 32; "Don’t have an account?" 13px / 18px `#71717b` + 4 + "Sign up" 13px / 18px 500 `#1b4ed8` | `#a1a1a1`, `#50a2ff` |
| footer item | box | border-top 1px `#1c1c1c12`, padding 16 × 32; "Secured by" 12px / 16px 500 `#71717b` + Clerk logo 48×14 muted | border `#f1f1f112` |
| footer | fill | transparent (T3 override) | = |

### 3.7 Boot splash (P, plus a live screenshot during loading)

| Part | Value |
|---|---|
| background | `#boot-shell` is transparent (`background: inherit` from `#root`), so the body shows: `#ffffff` (light) / `#000000` (dark) from the boot style (index.html:398-408), and the boot script also sets `html`'s inline `background-color` and the `theme-color` meta to the same chrome colour (index.html:358-374; error fallback 383-385); `--boot-background` when a theme is selected |
| text | body colour `#262626` / `#f5f5f5`; everything the app renders without its own colour class inherits this (it is why `EmptyTitle` and the banner title read `#262626`, not `--foreground`) |
| card | 96×96, centred in the window |
| logo | 64×64, `apple-touch-icon.png` (180×180 source), `object-fit: contain` |
| theme-color meta | at boot the script writes the chrome colour: `#ffffff` / `#000000` with no theme selected (M\* with the main chunk aborted, both themes). Once the app runs, `syncBrowserChromeTheme` (useTheme.ts:271-300) copies the background of `main[data-slot=sidebar-inset]`, else the sidebar's, else the body's, into `html`, `body` and the meta: `oklch(0.992 0 0)` = `#fcfcfc` on a thread page, `oklch(0.985 0 none)` = `#fafafa` on page-not-found (M, light); `#000000` in dark |

### 3.8 Boot failure page (M\*)

Colours: `--boot-error-fg` `#262626` / `#f5f5f5`; `--boot-error-bg` `#ffffff` / `#000000` (the body); line = `currentColor` 12% `#2626261f` / `#f5f5f51f`.

| Part | Property | Value |
|---|---|---|
| column | box | width `min(400px, 100% − 48px)` = 400, x centred (520 at 1440), top-aligned, padding-top `clamp(24px, 22vh, 220px)` = 198 at 900 tall, padding-bottom 24 |
| column | type | 13px / 20px; family is the body stack `-apple-system, BlinkMacSystemFont, "Inter Variable", "Segoe UI", system-ui, sans-serif` |
| h1 | type | 15px / 22px, 600 |
| p | type | 13px / 20px, opacity 0.7, margin-top 4, `overflow-wrap: anywhere`, `text-wrap: pretty` |
| actions | box | margin-top 16, gap 8 |
| "Reload" | box | 72.56×32, padding 0 × 14, radius 6, 13px 500, border 1px fg; fill fg, text bg: light fill `#262626` with `#ffffff` text, dark fill `#f5f5f5` with `#000000` text (M\*, both themes); hover opacity 0.9 |
| other buttons | box | 32 tall (report row: 28, padding 0 × 10), radius 6, border 1px line, transparent, text fg, 500; `aria-disabled` opacity 0.6, default cursor |
| update note | type | 12px / 18px, margin-top 8, opacity 0.7 |
| details | box | margin-top 32, full width |
| summary | box | inline-flex, gap 8, radius 4, opacity 0.7 (1 on hover or open), 20 tall; marker: a 5×5 box with right and bottom borders in currentColor (declared 1.5px, index.html:573-574; Chrome computes 1px here), rotated −45° closed (points right) and 45° open (points down) |
| details blocks | box | each a grid with gap 8, margin-top 20; links block-level, `fit-content` |
| details p | type | 12px / 18px, opacity 0.7 |
| command row | box | flex, gap 8, padding 4 4 4 10, border 1px line, radius 6, 34 tall; `code` flex 1, 12px / 20px mono (`ui-monospace, "SF Mono", Menlo, Consolas, monospace`), opacity 0.8 |
| copy icon button | box | `min-width` 24, 24 tall, padding 0 × 4, no border, radius 4, opacity 0.7; hover opacity 1 + fill currentColor 8%; icon 14 stroke 2 |
| report pre | box | max-height 200, padding 10 × 12, border 1px line, radius 6, opacity 0.8, `pre-wrap`; type 12px / 16px mono (`#boot-error pre` beats `#boot-error-details`'s 11px, measured 12) |
| report row | box | flex, gap 16; link underline offset 3, decoration currentColor 40% (100% on hover) |
| focus-visible | outline | 2px solid fg, offset 2 (buttons, links, summary, pre) |
| page | scroll | `#boot-shell` becomes full height and scrolls itself (`overflow-y: auto`) once the error is in |

### 3.9 Main-window empty states (P; hosted home also H)

Main column at 1440 with the sidebar open: x 256, 1184×900. Top bar where present: 52 tall (`--workspace-topbar-height`; 40 under window-controls overlay), padding-x 20 (`--workspace-gutter`, 12 below 640), border-bottom 1px `--border` `#e4e4e7` / `#ffffff14`, title span 14px / 20px 500 in `--muted-foreground` 60% `#71717b99` (≥768px; `--foreground` below) / `#a1a1a199`. H confirmed 52 tall, padding-left 20 and `#e4e4e7` on the hosted home.

| Surface | Geometry | Type and colour (light / dark) |
|---|---|---|
| No active thread | Empty padding 48 (24 <768); inner `max-w-lg` 512 with padding 48 × 32; header block 448×52, vertically centred in the area below the bar | title "Pick a thread to continue" 20px / 28px 600, inherits `#262626` / `#f5f5f5`; description 14px / 20px `#71717b` / `#a1a1a1`, margin-top 4 |
| No projects hero | no bar on web; inner 512 wide; header block 448×112, centred in 900 | title 30px / 36px 600 (24px / 32px below 640); description margin-top 4; button row margin-top 24, gap 8, buttons 28 tall: "Add project" 115.72 wide (default sm, plus 16 white), "Start without a project" 186.22 wide (outline sm, icon 16 `--muted-foreground` `#71717b` / `#a1a1a1`) |
| Draft start error | header `max-w-md` (448); 369×100 | title 20px / 28px 600; description margin-top 4; button row margin-top 20; "Try again" default sm 100.3×28 with refresh-cw 16 white |
| Hosted home | header `max-w-sm` (384); EmptyContent 384 wide, gap 24 below the header (Empty gap) | title 20px / 28px 600; description margin-top 4 (two lines when cloud is off) |
| hosted steps | `ol` `space-y-4` (16); step number column 8 wide + gap 12, number 14px / 20px muted with padding-top 4 | step text 14px / 20px `#262626` / `#f5f5f5` |
| install commands | `space-y-2` (8); captions 12px / 16px muted; CommandBlock 364×46, padding 10 × 12, radius 8, border `#e4e4e7` / `#ffffff14`, fill `--muted` 50% `#fafafa80` / `#ffffff04`, text 14px / 20px mono, "$" muted with 8 after; copy button 24×24 ghost (icon 14 muted); footnote 12px / 16px muted (33 tall for two lines with inline `code`) | |
| download link | 12px / 16px muted, underline offset 2, margin-top 8; hover `--foreground` | |
| hosted heights | step 1 "Sign in" row 28 (button 65.36×28); step with RunOnComputer 320 | (H matches P; re-measured 2026-11-18: title `#262626`, description `#71717b`, content 384, steps 28 and 320, CommandBlock 364×46 padding 10 × 12, border `#e4e4e7`, 14px) |
| Page not found (M, both themes re-measured) | `main` padding 24, no own fill (shows the body, whose inline colour `syncBrowserChromeTheme` copies from `[data-slot=sidebar-inner]` because there is no `main[data-slot=sidebar-inset]`: sidebar-scope `--sidebar` `#fafafa` / `#000000`); column 384 wide, gap 16; `main` text inherits `#262626` / `#f5f5f5` | h1 18px / 28px 500 `#18181b` / `#f5f5f5`; p 14px / 20px `#71717b` / `#a1a1a1`, two lines; "Go home" default size 84.2×32, padding-x 11, `#1b4ed8` / `#346bf1` with `inset 0 1px #ffffff29` + `0 1px 2px #194fd93d` / `#326df23d`; it is an `a href="/"` |
| First-run recovery | full window, padding-x 24; column 286×104 (connection) | h1 18px / 28px 600 `--foreground` `#18181b` / `#f5f5f5`; p margin-top 8; "Reload" outline sm 83.83×28 margin-top 20, icon 16 muted |

### 3.10 Welcome page (M)

| Part | Property | Light | Dark |
|---|---|---|---|
| column | box | 768 wide (`max-w-3xl`), centred inside 6px scrollbar gutters (x 336), padding 80 top (48 below 640), 24 sides, 32 bottom; content 720 | = |
| background | main | `--background` `#fcfcfc` + grain | `#000000` |
| h1 | type | 24px / 32px, 600, letter-spacing −0.6px, `#18181b` | `#f5f5f5` |
| subtitle | type | 14px / 20px, margin-top 6, `#71717b`; label `#18181b` | `#a1a1a1`; `#f5f5f5` |
| sections block | box | margin-top 40, border-top 1px `--border` 60% `#e4e4e799` (reads back `#e4e4e899`) | `#ffffff0c` |
| section | box | grid 96px + 1fr (≥640), column gap 40, padding 16 × 0, border-bottom 1px `#e4e4e799` | `#ffffff0c` |
| section label | type | 14px, line-height 40, `#71717b` | `#a1a1a1` |
| agent row | box | min-height 40, padding 4 × 0, gap 12; icon column 20 (icon 18) | = |
| agent name | type | 14px / 20px, 500, `#18181b` | `#f5f5f5` |
| "Ready" | type | 12px / 16px, 500, `--success-foreground` `#007a55`, check 14, gap 6 | `#00d492` |
| note | type | 12px / 16px, `#71717b`, padding-left 32 | `#a1a1a1` |
| add-agents trigger | box | 584×40, gap 12, radius 6, no fill in any state; label 14px / 20px `#71717b`, hover `--foreground` `#18181b` (M; only the label changes); logo stack opacity 0.6, grayscale, gap 8, icons 14; chevron 16 `#71717b`, stroke 1.75 | `#a1a1a1`; hover `#f5f5f5` |
| projects row | box | 584×44 (min 40, padding 4), gap 12 | = |
| checkbox | box | 16×16 (≥640; 18 below), radius 4, border `--input` `#d4d4d8`, fill `--background` `#fcfcfc`, disabled opacity 0.64 | fill `--input` 32%, border `#ffffff1c` |
| projects title / sub | type | 14px / 20px 500 `#18181b`; 12px / 16px `#71717b` | `#f5f5f5`; `#a1a1a1` |
| permission line | type | 14px / 20px, padding 4 top, 32 left, `--destructive` `#fb2c36` | `#fb414a` |
| footer row | box | margin-top 24, space-between, gap 12; "Start" default size 56.25×32, padding-x 11 | = |
| telemetry notice | type | 11px / 17.875px, `--muted-foreground` 70% `#71717bb3`, padding-top 48, pushed to the bottom (`mt-auto`); copy "T3 Code collects anonymous usage data to improve the app. To opt out, see the [privacy policy]." (managed policy off: "<policy message> See the [privacy policy]."); link underline, decoration 40%, offset 2, hover full `--muted-foreground`, opens in a new tab (WelcomePage.tsx:326-342) | `#a1a1a1b3` |

### 3.11 Reconnecting banner (M\*, draft hero, offline emulation)

| Part | Property | Light | Dark |
|---|---|---|---|
| banner | box | 736×34 (composer width), padding 5 × 4 | = |
| surface (`::before`) | shape | radius 16, border 1px warning 28% `#fe9a0047` (error variant uses `--destructive` 28%) | = |
| surface | fill | attached surface (`--card`) at `--glass-opacity` 80%: `#ffffffcc`, plus `linear-gradient` of warning 8% | `#111111cc` + same tint |
| surface | backdrop / shadow | `blur(12px) saturate(1.14)` / `shadow-composer` `0 1px 2px #0000000d` | `blur(16px) saturate(1.08)` / `0 14px 32px -18px #000000bf` |
| row | box | 728×24, gap 4; icon column 24 (28 below 640) | = |
| icon | wifi-off | 12, stroke 1.75, `--muted-foreground` `#71717b` | `#a1a1a1` |
| title | type | 12px, 500, line-height 24, inherits `#262626` | `#f5f5f5` |
| description | type | 12px / 16px, `#71717b`, truncates; hidden below a 400px container | `#a1a1a1` |
| action | "Retry now" | ghost xs: 73.89×24, padding-x 7, 12px / 16px 500, `#18181b`, transparent; corner radius 8 except the outer end corners 12 (`rounded-se-xl`/`rounded-ee-xl`) | `#f5f5f5` |

Re-measured in both themes (M\*): box, surface, row, icon, title, description colour and the action. The responsive rows (icon column 28 below 640, description hidden below 400) are source-derived.

### 3.12 App toasts (M, raised locally)

Toast geometry and colours are in 02-primitives §3.24. The root is `div.dropdown-glass` with `role=dialog`, `tabindex=0`, `aria-modal=false`, `data-position=top-right` and `aria-labelledby`/`aria-describedby`. `data-type` is present only when the toast has a type, so the untyped permission notice has none (M). Re-measured colours (M, placeholder toasts): fill `#ffffffd5` / `#111111d5`, title `#18181b` / `#f5f5f5`, description `#71717b` / `#a1a1a1`, close orb `#ffffffeb` / `#111111eb` with an x 12 in `#71717b` / `#a1a1a1`. Variants specific to this document:

| Toast | Size | Layout details |
|---|---|---|
| legacy migration (loading) | 360×88 | title row: spinner `loader-circle` 16×20 (`[&>svg]:h-lh w-4`), opacity 0.8, `--foreground`; title 14px / 20px 500; description two lines 14px / 20px muted; content padding 12 × 14, title row padding-right 24 (clear of the close orb) |
| thread notification ("Thread completed") | 360×68 | inline layout: leading icon in `flex h-lh w-4 shrink-0 items-center justify-center` (16×20), `circle-check` 16 `--success-foreground` `#007a55` / `#00d492`; content padding-right 24 because trailing controls exist; action "Open thread" Button xs default 87.22×24, 12px / 16px 500, padding-x 7; no copy button |
| permission notice (stacked-end) | 360×176 | content column, gap 8, padding 10 × 14; body padding-right 20; title two lines (40); description four lines (80); action row full width, right-aligned, gap 6: "Dismiss" ghost xs (60.77×24, transparent, `#18181b`) then "Open Sharing & access" default xs (149.17×24) |
| slow requests (warning, expandable description) | 360×68 closed | description row is `role=button` (radius 4, hover fill `--muted` 40%), description underlines on hover (offset 2, decoration muted 60%), chevron-down 14×20 opacity 0.8 stroke 2.25 → chevron-up when open; panel margin-top 8, max-height 160 (scrolls), padding-right 2; items 12px / 16px: tag 500 `--foreground`, "Started <time>" 10px / 14px opacity 0.75 margin-top 2; items separated by 1px `--border` 50% `#e4e4e780` with padding-bottom 8 and gap 10 |

## 4. States

### 4.1 Gate and startup

| State | Surface | Notes |
|---|---|---|
| loading bundle | boot splash | until the router has resolved and React renders |
| bundle failed | boot failure page | details open by default on a repeat failure within 5 min (sessionStorage `t3code:boot-failed-at`, bootError.ts:6-7, 144-152) |
| session request failed | root error view | "Primary environment request failed during fetch-session-state (HTTP <status>)." (auth.ts:62-64). Network failures count as HTTP 500 and are not retried; 502/503/504 retry for 15 s |
| not paired | `/pair` form | description by bootstrap methods: desktop-bootstrap → "This environment expects a trusted pairing credential before the app can connect."; otherwise "Enter a pairing token to start a session with this environment." (269-275). Info box: both → "Desktop-managed pairing and one-time pairing tokens are both accepted for this environment."; desktop only → "This environment is desktop-managed. Open it from the desktop app or paste a bootstrap credential if one was issued explicitly."; else "This environment accepts one-time pairing tokens. Pairing links can open this page directly, or you can paste the token here." (277-290) |
| first-run pending > 4 s | "Still connecting" | the 4 s timer starts once settings have hydrated (FirstRunGate.tsx:152-159). Separately, while settings hydration has failed or is retrying, "Could not read settings" replaces everything at once, with no timer (171-173); Retry is disabled and its icon spins while retrying |
| lazy chunk failed, first time in this tab | one automatic reload | only for chunks loaded after main (§2.1); the main bundle has no reload and shows the boot failure page at once |

### 4.2 Pairing form

| State | Visual |
|---|---|
| default | §3.2; input empty or prefilled from `#token=` |
| opened with an error | the gate's `errorMessage` (in practice a failed desktop bootstrap credential, §2.1) is shown in the Alert from the first frame (S) |
| focus-visible (input) | ring border + 3px inset ring (M\*, both themes) |
| submitting | input and both buttons disabled (opacity 0.64), primary label "Pairing..." (S) |
| empty submit | "Enter a pairing token to continue." in the Alert, thrown before any request (no POST was sent, M\*); focus stays on Continue; the Alert is `role=alert`, so it is announced |
| error | Alert error between the field and the buttons. Messages: empty → "Enter a pairing token to continue."; rejected → "Invalid pairing token. Check the token and try again."; slow session → "Timed out waiting for authenticated session after bootstrap."; transport → "Primary environment request failed during exchange-bootstrap-credential (HTTP <status>)." (M\*); unknown → "Authentication failed." (auth.ts:62-102, PairingRouteSurface.tsx:257-267) |
| success | no visual; history replaced with `/`, then a full reload |
| pending route | `PairingPendingSurface` (S) |

### 4.3 Hosted pairing page

| State | Title | Description | Buttons |
|---|---|---|---|
| pairing | "Pairing backend" | "Connecting to this backend." | disabled "Pairing..." |
| no host or token | "Pairing failed" | "This pairing link is missing its backend host or token." | none |
| resubmit | "Pairing failed" | "This one-time pairing token was already submitted. Request a new pairing link." | none |
| failed | "Pairing failed" | "<error> If the backend accepted this one-time token, request a new pairing link before retrying." | "Try again" |
| paired | "Backend paired" | "<label, or "The environment"> is saved in this browser." | outline "Open app" |

Failed shows the CORS/HTTPS Alert. The host box shows whenever the URL parsed.

### 4.4 T3 Connect sign-in

| State | Visual |
|---|---|
| Clerk loading | entry buttons disabled ("Sign in" on the hosted home, "Sign in to T3 Connect" on the Welcome page); the Settings sidebar button is not rendered until loaded |
| signed in | Settings footer shows the avatar `UserButton`; hosted home step 1 reads "[check] Signed in" |
| modal: start | §3.6 |
| modal: input focus | 4px solid ring (H, both themes) |
| modal: primary hover | `bg-primary/90`: `#1b4ed8e6` (H) |
| modal: code / password / error steps | Clerk's (not reached) |

### 4.5 Empty states, Welcome, banner

| Surface | State | Visual |
|---|---|---|
| No projects hero | no scratch target | one button, description "Add a project to start your first thread." |
| Hosted home | local environment off | title "This computer is off", single "Open Environments" button |
| Hosted home | cloud on, signed out → in | step 1 button → "Signed in"; step 2 gains the computer list or "Waiting for your computer…" |
| Welcome | agents checking | "Checking <label>…" row; agent rows "Checking…" |
| Welcome | no agent found after checks | the add-agents fold opens by itself once (WelcomePage.tsx:437) and reads "Set up an agent" |
| Welcome | no agent ready | footer shows "Start without an agent" and "Set up one agent to start."; Start disabled |
| Welcome | read-only connection | notes "This connection cannot manage agents." / "… cannot control terminals."; checkbox disabled; destructive line (M) |
| Welcome | importing | Start reads "Importing…" (with a selection) or "Starting…"; page leave blocked, a try shows info toast "Finishing setup" / "One moment." |
| banner | connecting or reconnecting, first 2,000 ms of each attempt | hidden (grace; a later attempt to the same environment gets a new grace) |
| banner | reconnecting | warning, "<machine> is reconnecting", "Retry now" (M\*) |
| banner | offline | warning, "<machine> is offline", "Reconnect"; no grace |
| banner | error phase | error variant (destructive tint), "<machine> is offline", "Reconnect"; no grace |
| banner | connecting (first connect) | after the grace: warning, "<machine> is reconnecting", no button |
| banner | environment switched off | "Reconnect" switches it back on |
| banner | unavailable ≥ 20 s, remote target | adds "Disconnect server" (disabled while it runs) |
| banner | server update running | hidden while reconnecting |

### 4.6 Data attributes and roles

- Alert: `role=alert`, `data-slot=alert`, `data-variant`; boot error message `role=alert`; composer banner `role=alert`, `data-variant`, `data-composer-banner-surface`.
- Toast roots carry `data-type` when typed (02-primitives §4.11), plus `role=dialog`, `aria-modal=false` and `data-position=top-right` (§3.12).
- `NoProjectsHero`, `DraftStartError` and the Welcome page put `workspaceAreaProps("home", 1, …)` on their inset: `data-workspace-area=home`, `data-workspace-order=1`, `aria-label` "Home" (Welcome: "Welcome") and `tabindex=-1`; the explicit `role=main` overrides the helper's `role=region` (workspaceFocus.ts:12-20; focus regions, 03-shell §5.7). `NoActiveThreadState` and `HostedEmptyState` have none of these.
- Boot failure page: the copy-command button's `title`/`aria-label` switches from "Copy command" to "Copied" or to the copy hint (§2.8).

## 5. Behaviour

### 5.1 Startup order

1. `index.html` paints the splash and the boot theme (01-tokens §5.2).
2. `bootstrap.ts` imports `main` and awaits its `startup` promise; a rejection of either → boot failure page (bootstrap.ts:4). Lazy chunks loaded after main first get one automatic reload per tab (§2.1).
3. `main.tsx` starts app services (toasts, notifications, title), loads Clerk only when T3 Connect applies, waits for `router.load()` and the locale, then renders.
4. The root `beforeLoad` resolves the gate (§2.1). A URL token is taken and stripped first; a desktop bootstrap token is exchanged automatically.

### 5.2 Pairing

- Submit trims the credential, POSTs `/api/auth/browser-session`, then polls `GET /api/auth/session` every 100 ms for up to 2 s until authenticated (auth.ts:139-140, 242-261, 331-346).
- A `#token=` (or `?token=`) on `/pair` prefills the field and submits once on mount, after stripping the token from the URL (PairingRouteSurface.tsx:38-83). Stripping goes through `history.replaceState`, which TanStack Router's history patches to notify subscribers (`Match-CtCj-_Kc.js`), so the root gate re-runs; `explicitPairingRequested` keeps the form on screen through that reload (auth.ts:348-358).
- The hosted page strips the token the same way and submits once on mount through `connectPairing({ host, pairingCode })`, a single-flight command (`connection/onboarding.ts:13-24`) that runs `registerPairing` (packages/client-runtime/src/connection/onboarding.ts:101-169) in this order:
  1. Fetch `<host>/.well-known/t3/environment`.
  2. Run the registry's known-address check (`assertKnownAddress`) and the orchestration-protocol check. A server that is outdated but can be updated still passes, so it can be updated from this client. Both checks run before the code is redeemed, so a wrong link does not spend it.
  3. Exchange the code for a bearer session, register the environment in the browser and switch it on.
- An unparsable host fails as "The pairing details are invalid." or the parser's own message (77-88). "Try again" after a failure re-submits the same code. "This one-time pairing token was already submitted. Request a new pairing link." is defensive: it needs a second submit while the code is in flight or after it succeeded, and the UI offers no button in either state (PairingRouteSurface.tsx:164-192).
- Measured with a mocked signed-out session: "Pairing backend" appeared 6.6 s after load, "Pairing failed" 37 ms later, and then the gate's next state replaced the page (§7).
- Reload app: `window.location.reload()`. Continue is a form submit, so Enter in the field submits.
- After a successful form pairing, `/pair`'s `onAuthenticated` replaces history with `/`, flushes, and reloads the page so the WebSocket and cached scopes pick up the new cookie (pair.tsx:48-53).
- While the gate is `authenticated`, `hosted-static` or `guest`, `/pair` is never shown: it redirects to `/` (pair.tsx:18-24).

### 5.3 T3 Connect sign-in

- Only where T3 Connect is on (§2.6). The modal is Clerk's: Escape or the close button dismisses it (H: Escape closed it), the backdrop is Clerk's modal backdrop.
- After sign-in Clerk redirects to the URL it was opened from (browser) or to `/` with the hash route kept (desktop).
- Signed in but no computer: the hosted home lists computers discovered through T3 Connect and auto-adds compatible ones once discovery settles (every discovered computer has left `checking`); while the list is empty and the device is online it re-runs discovery every 5,000 ms while the page is visible, and also on window focus and on becoming visible (`EMPTY_DISCOVERY_REFRESH_INTERVAL_MS`, HostedEmptyState.tsx:66-116, CloudEnvironmentConnectList.tsx:41, 229-270). The first added computer unmounts the empty state.
- The post-sign-in wizard (`ConnectOnboardingDialog`) titled "Set up T3 Connect" offers "Publish this environment" ("Make this environment available to your other devices through T3 Connect.") and "Publish agent activity" ("Send activity from this environment to your mobile clients for push notifications and Live Activities.") steps, then "Connect devices"; it does not open on the hosted empty home (ConnectOnboardingDialog.tsx:63-65, 127, 136-200, 233-267).

### 5.4 Index and empty states

- `/` never shows a picker when a project exists: it opens a new draft in the most recently updated project (sorted like the sidebar, `updated_at`), and retries when the draft did not open (`null`) (_chat.index.tsx:47-75).
- "Add project" opens the command palette on its add-project page; "Start without a project" starts a scratch thread on the primary environment's scratch target (NoProjectsHero.tsx:37-51).
- "Try again" on the draft error bumps the retry counter and tries again.
- Missing threads: home when the environment has threads, else stay blank (§2.1).
- Header text "No active thread" is muted at ≥768px and `--foreground` below (the header is the only title on mobile).

### 5.5 Welcome page

- Opened by the first-run gate from `/` when the workspace is fresh (no projects/threads beyond the server's own bootstrap pair), or from Settings. Direct links are never redirected into setup.
- On mount it refreshes the computer's provider list (`refreshProviders`) and scans agent history for projects.
- Start: imports the selected projects (create project, then import its agent sessions), completes onboarding (`onboardingCompletedAt`), lands in the first project with imported history (or `/`), then toasts "Imported N thread(s)" or, on a partial import, the sticky warning "Some history was not imported". Failure to save: error toast "Could not finish setup" / "Your settings could not be saved. Try again." (WelcomePage.tsx:179-235).
- Selection persists in localStorage per computer; by default recent new projects are picked.
- The telemetry notice changes when a managed policy turns telemetry off ("See the privacy policy").
- The page follows a live system theme change, unlike the standalone pages (M with forced frames, §7.9).

### 5.6 Toasts the window raises (catalogue)

Defaults: stacked viewport top-right, 5,000 ms, max 3 (02-primitives §5.11). `timeout: 0` is sticky.

| Source | Type | Title / description | Timeout, actions |
|---|---|---|---|
| legacy thread migration | loading | "Restoring your threads…" / "Migrating <N, locale-formatted> thread(s) from the previous version. You can keep working while this finishes." | sticky; closed when the server stops reporting `running` (LegacyThreadMigrationToast.ts:8-39) |
| slow RPC acks | warning | "Some requests are slow" / "<N> request(s) waiting longer than <min threshold>s." (threshold rounded to whole seconds) | sticky, updated in place, closed when none remain; expandable list via the description ("Show requests" / "Hide requests"), each item the request tag and "Started <`toLocaleTimeString()`>". A request counts as slow when its ack takes longer than 15,000 ms, or 120,000 ms for provider update/refresh, server update and VCS refresh. Subscriptions, `pullRequests.*`, the usage summary, client-activity reports and the source-control sign-in wait are never tracked; at most 256 are tracked (requestLatencyState.ts:6-97) |
| host freeze (reported when the server config arrives, including on connect) | warning | "<host> froze for <round(durationMs/1000)>s" / "Its CPU is overloaded (load <round(loadAverage)> on <cpuCount> cores)." when load > cores, else "T3 Code on <host> was too busy to answer." | default; once per environment and stall (`detectedAt`) (hostFreeze.ts:12-25, HostFreezeToastCoordinator.ts:9-33) |
| legacy permissions | (none) | "Permissions have changed for <label>" / "This connection still uses the old permissions, so some actions may no longer be available. Pair again using a new link with the permissions you need." | sticky; stacked-end: "Open Sharing & access" (default) → `/settings/access`, "Dismiss" (ghost); dismissal stored in localStorage `t3code:permission-update:v1:<environment>`, also when it is closed with the close orb (`onClose`, PermissionUpdateNotice.ts:35-61); shown once per environment per app run |
| keybindings reload | success / warning | "Keybindings updated" / "Keybindings configuration reloaded successfully."; "Invalid keybindings configuration" + a warning body with the config path | default; warning is stacked |
| desktop update downloaded | success | "Update downloaded" / "Restart the app from the update button to install it." + "Read more" button (muted, dotted underline offset 4, arrow-right 12 rotated −45°, stroke 2.25) | default; "Unable to open release notes" error if the link fails |
| reconnect / disconnect failure | error (stacked) | "Could not reconnect environment" / message or "Failed to reconnect."; "Could not disconnect server" / message or "Failed to disconnect." (not raised for an interrupted command) | default |
| T3 Connect add | error | "Could not connect environment" / message, or "Could not connect the T3 Connect environment." | secondary "Copy trace ID" when a trace id is known (CloudEnvironmentConnectList.tsx:168-186) |
| Welcome page | info / error / warning / success | "Finishing setup" / "One moment."; "Could not finish setup"; "Some history was not imported" (sticky); "Imported N thread(s)"; "Could not turn on <agent>"; "Could not save Codex setup" | |
| thread notifications | success / warning / error | §5.7 | default; "Open thread" |
| PR review guide | success / error | "Review guide ready" / "Review guide failed" + the PR title, or "<repository> #<number>" | skipped when that guide is already on screen in a focused window (PullRequestGuideRunCoordinator.tsx:85-96) |

### 5.7 Thread notifications, sounds and the badge

`startThreadNotifications` (ThreadNotificationCoordinator.tsx:62-447) watches every connected environment's live thread shell:

- Kinds: `input` (a thread needs attention: status input, request, approval, failed, limited) and `completion` (idle with its latest run completed).
- Which threads: every thread in the live shell except those shown under a parent (delegated subagent threads, whose pending requests surface on the parent instead) and those hidden by the tag filter (Coordinator 245-254, 136-141). A `ready` thread whose latest run failed counts as `failed`.
- When `notificationMode` is `off` and in-app notifications are off, no environment is watched at all and open notification toasts close (Coordinator 358-372). Changing the mode clears pending desktop notifications.
- Gate (logic.ts:19-112): the first sighting of a thread and archived threads never notify. A run the user asked to stop (`latestRunStopRequested`) suppresses only its completion; it can still raise an input notification. A request with an id notifies once per status and id until the thread stops needing attention; without an id, once per run and status. A completion that answers a background-work wake waits 5,000 ms (`RELEASED_COMPLETION_GRACE_MS`) so a follow-up wake can supersede it. Grace timers pause while the snapshot is stale (reconnecting).
- Titles: completion "Thread completed"; approval "Approval needed"; limited "Usage limit reached"; failed "Thread failed"; request "Guest request"; else "Input needed". Body: the thread title.
- Settings (defaults: `notificationMode` `off`, `inAppNotificationsEnabled` `false`; mode labels "Off", "Notifications only", "Sound only", "Notifications with sound", threadNotifications.ts:8-13). The sound plays in sound modes whatever else happens. When the in-app toast shows, it takes the place of the desktop notification for that event (Coordinator 161-200).
  - sound modes (`sound`, `notifications-and-sound`): `notification-completion.mp3` or `notification-input.mp3` through an `AudioContext` unlocked by the first pointer or key press; a sound does not restart while it is still playing.
  - in-app toast when enabled, the window is visible and focused, and the thread is not the open one: type success (completion), error (failed) or warning; leading icon 16: `circle-check` `--success-foreground`, `shield-question` `--warning-foreground` (approval), `circle-alert` `--warning-foreground` (limited) or `--destructive-foreground` (failed), `message-circle-question` `--info-foreground` (`#1447e6` / `#50a2ff`, input and request); action "Open thread"; no copy button. Opening the thread or hiding it closes its toasts.
  - desktop notification modes (`notifications`, `notifications-and-sound`), when the window is hidden or unfocused and permission is granted: `new Notification(title, { body, tag, silent: true, requireInteraction })`, `requireInteraction` only for completions on Windows. Tag: 64-bit FNV-1a of `"<environment>:<thread>"` as 16 hex digits (threadNotifications.ts:23-36); a new notification for the same thread closes and replaces the old one. Click: close, focus, `desktopBridge.revealWindow()`, open the thread. Pending notifications of a removed environment, or of a thread that becomes hidden, close.
- Badge = number of desktop notifications still pending. Web: the favicon is swapped for a 64×64 PNG: a `#e5484d` circle of radius 28 at (32,32) with white `600 40px "Segoe UI", sans-serif` digits, centred and middle-baselined at (32,34), or `600 30px` "9+" above 9; the original `link[rel=icon]` is removed and put back at 0. Desktop: `desktopBridge.setNotificationBadge({ count, image })`, the image only on Windows. The badge clears on window focus and on the bridge's badge-clear event (threadNotifications.ts:41-81, Coordinator 386-415).

### 5.8 Background announcer and window title

- `BackgroundThreadAnnouncer` is an `sr-only` `div aria-live="polite"` (no `role=status`, which Orca drops) mounted in `AppSidebarLayout` outside the sidebar so a collapsed, inert sidebar does not silence it (BackgroundThreadAnnouncer.tsx:84-91, AppSidebarLayout.tsx:492).
- It speaks, for listed threads in live environments other than the open one and not snoozed: "Needs approval: <title>", "Needs your input: <title>" on a new request, and "Done: <title>", "Failed: <title>", "Stopped: <title>" when a run settles (not while another run is still working). Several in one update join with ". ". First sightings are a silent baseline, so connecting does not read every thread (backgroundThreadAnnouncement.ts:53-88; threadActivityAnnouncement.ts:7-14, 50-60).
- The text clears after 7,000 ms; repeating the same text appends a no-break space so it counts as new (useLiveRegion.ts:6, 27-34).
- The window title names the open thread and project (with its status when the screen-reader setting is on) or the app (appInit.tsx:141-205).

### 5.9 Connection diagnosis headline (banner description)

`ConnectionDiagnosisHeadline` text from `diagnosis.ts:98-157` (`<host>` = the computer label):

| Cause | Headline |
|---|---|
| this device offline: the device network is offline, or the environment's phase is `offline` (diagnosis.ts:65-68) | "This device is offline" |
| T3 Connect sign-in still loading | "This device is still loading its T3 Connect sign-in" |
| outdated server | "<host> runs an older T3 Code" |
| clock skew | "This device's clock is off" |
| not linked | "<host> is not linked to your T3 Connect account" |
| relay error | "T3 Connect returned an error" |
| tunnel pending / not enabled / down | "<host>'s tunnel is still being set up" / "T3 Connect is not turned on for <host>" / "<host>'s tunnel is offline" |
| origin unreachable / invalid response | "Tunnel works, but T3 Code on <host> did not answer" / "T3 Code on <host> sent an unexpected answer" |
| dns / tls / refused / timeout | "<host>'s tunnel address did not resolve" / "Secure connection to <host>'s tunnel failed" / "<host>'s tunnel refused the connection" / "<host> did not answer in time" |
| by hop, no cause | device: "This device could not prepare the connection"; relay: "T3 Connect did not accept this device's sign-in" or "This device can't reach T3 Connect"; tunnel: "This device can't reach <host>'s tunnel" or "T3 Connect could not reach <host>'s tunnel"; server: "T3 Code on <host> did not accept this device" or "T3 Code on <host> did not answer" (M) |

Status text elsewhere (`connectionStatusText`, presentation.ts:67-88): "Available", "Offline", "Connecting...", "Reconnecting..." or "Failed to connect. Reconnecting... Reason: <e>", "Connected", "Client not supported", "Connection failed[. Reason: <e>]". `ConnectionStatusDot` (ConnectionStatusDot.tsx:6-75): 8px dot in a 12px box; connected `--success`, connecting/reconnecting `--warning` with a 60% warning ping (`animate-status-ping`, 2,000 ms), error `--destructive`, otherwise `--muted-foreground` 40%; with a tooltip it is a `button` (`cursor-help`) and the tooltip opens on top.

### 5.10 Keyboard and focus

- Pairing form: the token field is the first tab stop; Enter submits; Tab → Continue → Reload app.
- Boot failure page: plain DOM; `details` toggles with Enter/Space on the summary; `pre` is focusable (`tabindex=0`) for scrolling.
- Clerk modal: Clerk traps focus; Escape closes (H).
- Toasts: F6 focuses the viewport, Escape closes the focused toast (02-primitives §5.11).
- Consent page: Approve is armed 700 ms after load or refocus, so a stray click from another window cannot approve.

## 6. Motion

| What | Property | Duration, easing | Source |
|---|---|---|---|
| boot splash → app | none (replaced on first render) | | main.tsx |
| boot error | none; the summary marker rotates without transition | | index.html:569-580 |
| standalone card | none (static) | | |
| buttons | `box-shadow, scale` 150 ms; press scale 0.97 | `cubic-bezier(.4,0,.2,1)` | 02-primitives §6 |
| reconnecting banner | opacity 0 → 1 on mount | 150 ms `ease-out` = `cubic-bezier(0, 0, 0.2, 1)` (M: computed `opacity 0.15s cubic-bezier(0, 0, 0.2, 1)`), from `@starting-style` (`starting:opacity-0`); none with reduced motion. It runs only on rendered frames: in a hidden window it stays at opacity 0 until the next paint | composer banner stack |
| reconnecting banner | appears after a grace | 2,000 ms per connecting/reconnecting attempt (`ENVIRONMENT_RECONNECT_WARNING_GRACE_MS`) | ChatView.logic.ts:110, useSystemComposerBanners.tsx:82-85 |
| "Disconnect server" | appears after the environment stays unavailable | 20,000 ms (`DISCONNECT_DELAY_MS`) | useEnvironmentDisconnectDelay.ts:7 |
| toasts | enter, stack, exit | 02-primitives §6 (transform 0.5 s `cubic-bezier(.22,1,.36,1)`, opacity 0.5 s, height 0.15 s) | toast.tsx |
| loading spinners (toast, consent, recovery) | rotate | 1 s linear infinite, paused when not visible (`visible-animate-spin`) | 01-tokens §5.8 |
| connection dot ping | `motion-safe:animate-status-ping` in `bg-warning/60`: opacity 0.9 → 0 and scale 0.75 → 2 by 40% in `steps(8)`, then hidden to 100% (01-tokens §6) | 2,000 ms, infinite | ConnectionStatusDot.tsx:20-22, 40; index.css:227, 347-361 |
| hosted home discovery | re-poll while no computer is listed | every 5,000 ms while visible, plus on focus / visibility | CloudEnvironmentConnectList.tsx:41 |
| Clerk backdrop | Clerk keyframes | 150 ms (H) | Clerk |
| Clerk modal content | Clerk keyframes | 180 ms `ease-out` (H) | Clerk |
| Welcome collapsibles | panel height | 02-primitives Collapsible (200 ms); chevrons rotate without transition | WelcomePage.tsx:647, 847, 923 |
| copy feedback | icon swap | 1,500 ms (CommandBlock); 2,000 ms ("Copied" on the root error view, hook default); never resets on the boot failure page | CommandBlock.tsx:17-20, useCopyToClipboard.ts:196, bootError.ts:297-343 |
| live-region text | cleared after speaking | 7,000 ms; boot page announcements are written 100 ms after clearing | useLiveRegion.ts:6, bootError.ts:269-274 |

## 7. Gaps

1. **Clerk's later steps are unmeasured.** The email-code step (OTP boxes, resend countdown), the password step, field errors, "Use another method", second factors and sign-up were not reached, because reaching them means submitting an email. Their copy and layout are Clerk's (en-US localization) under T3's appearance. Measure them in the desktop app with a test account.
2. **Clerk was measured on the hosted app**, a different T3 build. It loaded Clerk 6.44.0 at extraction and 6.45.0 at verification (2026-11-18), with identical results. The pinned tag catalogs 6.42.3, and the pinned build served over plain http has T3 Connect off (§2.6). Clerk's own spacing may differ slightly between those versions; T3's overrides are the same source.
3. **Hosted pairing page lifetime.** After stripping the token, the router re-ran the gate and replaced the page (observed with my mocked session). On the real hosted origin the gate becomes `hosted-static` and `/pair` redirects to `/`, so the page likely vanishes as soon as it mounts while the pairing continues in the background. "Backend paired" was never seen. This needs a check on the hosted app with a real link.
4. Not observed live: `PairingPendingSurface`, the "Pairing..." submitting state of the form, the boot error's update button (desktop only), desktop (Electron) header variants (drag strip, window-controls inset), the Settings sidebar sign-in button and avatar (T3 Connect off here; geometry per 04-sidebar `SidebarMenuButton`).
5. Probes, not the components: no active thread, no projects hero, draft start error, the first-run recovery, the consent page, the hosted home variants other than "signed out" (which H confirmed), Alert variants other than error.
6. Welcome page: measured only in a read-only connection with two ready agents. The install/sign-in terminal card, the Codex setup card, the project review list, the phone panel (needs T3 Connect) and the expanded add-agents grid are from source.
7. Notifications: desktop notifications, sounds, the favicon badge and in-app notification toasts were not triggered (both settings default off; nothing finished during the session). The toast layout was raised locally.
8. The banner's "Disconnect server" action (remote environments only) and the error-phase variant were not seen; offline emulation produced the warning "reconnecting" variant with the "server did not answer" headline.
9. Standalone pages do not follow a live system theme change: nothing subscribes to the media query outside the app shell (`useTheme.ts:466-487` attaches the listener on first `useTheme()` subscriber). M: switching the scheme on `/pair` changed nothing until reload, even with frames forced. The Welcome page and the app shell do follow live. A port may follow live; that is a deliberate deviation if chosen. Measuring note: the measurement window on the server is hidden (`document.visibilityState` "hidden"), so media-query change events, `@starting-style` transitions and rAF work only run when a screenshot forces a frame. Re-check "does not follow" or "stuck at opacity 0" findings with forced frames before trusting them.
10. The `.cl-*` Clerk elements are styled by Clerk's runtime CSS-in-JS, not by `main-DcchvbMF.css`; values in §3.6 are computed results, not declarations.
11. Not exercised: the boot page's copy feedback ("Copied" / "Press Ctrl+C to copy"), the root error view's "Copied" state, and the `/pair` form opened with a gate error (all S). Clicking a copy button would have written to the shared desktop clipboard.

## 8. Port notes for Go/Gio

1. **New primitives.** Add `StandalonePage` (page + 576dp glass card + wordmark header, §2.2/§3.1), `Alert` (§2.17/§3.5) and reuse `Empty`, `Button`, `Input`, `Toast` from 02-primitives. The card is `dialog-glass`: without real backdrop blur in Gio, paint the opaque fallback (01-tokens §3.12) or the tint over the page colour; on a flat page they look the same.
2. **Wordmark.** Port the T3 glyph path from `T3Wordmark.tsx:5-8` as a vector (viewBox 15.5309 37 94.3941 56.96), drawn 12dp tall in `--foreground`, then "Code" 18/28 500 −0.45 tracking in muted, baseline-aligned, 6dp apart. Do not substitute tern's logo on these surfaces if the port is 1:1.
3. **Sign-in (replaces `app/ui/signin_live.go` `LiveSignIn`, the dot-grid `signInGrid`, `flapping`, and the mockup `SignInScreen` in `app/ui/screens.go:2161-2298`).** T3 has no custom sign-in card: the email step is Clerk's start step in a 400dp `dialog-glass` card over a `dialog-backdrop`, 64dp from the top (§3.6). Build tern's email step from those measurements: header title 20/28 600 "Sign in to T3 Code", subtitle 14/20, label "Email address" 13/18 500, input 320×32 on `--secondary` with a 1px shadow outline and a 4dp `--ring` focus ring, primary 320×32 13/18 500 (no inset highlight, unlike T3's own Button; hover 90% fill), footer with a top hairline. tern cannot do social or passkey sign-in, so either omit those rows (and the "Don’t have an account?" footer) or show them disabled; this is a product decision to settle before porting. The code step and password step follow Clerk's flow (gap 1); keep tern's `codeField` six-box editor but restyle it to Clerk's OTP look once measured. Field errors go under the field in `--error-foreground` (`#c10007` / `#ff6467`), 13dp; drop the card shake and dot-grid light, which T3 does not have. Clerk's backdrop needs a window behind it: in tern the sign-in is the whole window, so paint `--background` with the backdrop tint or the empty shell behind.
4. **Pairing URL.** For tern's "Pair with a URL" step use the `/pair` form layout (§2.3/§3.2): title "Pair with this environment", label + 32dp input, sm buttons 28dp, info box. Copy: tern takes a full pairing URL (host + token) like the hosted page; keep T3's errors for the token exchange (§4.2) and the hosted messages for host problems (§4.3). Like `registerPairing`, check the descriptor (`/.well-known/t3/environment`) and protocol compatibility before redeeming the one-time code, so a wrong or incompatible host never spends it (§5.2). After success T3 reloads; tern should go straight to the app. The empty-field error is client-side and must not send a request.
5. **Boot.** Replace the blank `s.Booting` fill in `app/cmd/tern/main.go:72-74` with the splash: a `#ffffff` / `#000000` window (the boot style's own colours, not `--background` `#fcfcfc`), the 64dp app icon centred in a 96dp box. Add a boot failure screen (§2.8/§3.8) for "could not start" (for example, a corrupt store), and the root error view (§2.5/§3.4) for a failed first session request, with Try again / Reload / Copy error and the report block. The boot page's colours are its own fixed pair (`#262626` on `#ffffff`, `#f5f5f5` on `#000000`), not the app tokens.
6. **Empty states (replace the centred `s.Empty` label at `app/ui/screens.go:717-719` and the strings at `app/core/scene.go:336-338, 428, 461, 743`).** tern already opens a draft when nothing is open, matching T3's index (§5.4). Map: no projects anywhere → No projects hero (§2.10); no environment on the Connect account → hosted home "No computer connected" with T3 Connect steps (§2.11; tern is signed in, so step 1 reads "Signed in" and step 2 shows the install commands and "Waiting for your computer…"); a thread that failed to open → No active thread (§2.13); draft creation failed → Draft start error (§2.12). "Connecting…" maps to nothing in T3: T3 shows nothing until bootstrapped, then the empty state. Use a blank main rather than a "Connecting…" label for 1:1.
7. **Connection loss.** Remove the text status (`s.Net`, `core/app.go:428-453`) from any visible surface used as a connection indicator; T3's indicator is the composer banner (§2.18/§3.11) after a 2s grace per connecting/reconnecting attempt (offline and error show at once), with the diagnosis headline (§5.9). Core must expose per environment: phase (`connecting`, `reconnecting`, `offline`, `error`, `connected`), the time the phase started (for the grace and the 20 s "Disconnect server" delay), whether the environment is switched off (then "Reconnect" switches it on), whether it is the primary/local target (no disconnect), whether the device network is offline, the last failure's hop and cause, the label, and retry / enable / disable actions. tern's red dismissable `errorBanner` (`app/ui/screens.go:1388-1399`) has no T3 counterpart for failed actions: T3 raises error toasts (stacked) instead.
8. **Toasts.** tern has no toast system yet. Implement the 02-primitives toast (top-right, 360dp, 3 max, 5s default, sticky option, stacked-end layout, leading icon, expandable description) and raise at least: host freeze, slow requests (core must track pending RPCs with tag, start time and threshold: 15 s, 120 s for provider update/refresh, server update and VCS refresh; skip subscriptions and background reports, §5.6), legacy migration (core must surface the server's migration status and total count), permission update (legacy scopes), reconnect failures and thread notifications.
9. **Thread notifications.** Core must deliver per-thread transitions with: status (input / request / approval / failed / limited / ready), latest run id, status and completion time, the pending request id, background-wake flag, stop-requested flag, archived flag. Port the gate in `ThreadNotificationCoordinator.logic.ts` exactly (first sighting silent, archived silent, stop-requested runs silent for completion only, once per status and request id, 5s wake grace) and the coordinator's filters (skip threads shown under a parent and threads hidden by the tag filter; watch nothing while both settings are off). Use native notifications (macOS `UNUserNotificationCenter`, Windows toast) for the desktop mode with the same tag hashing, the dock/taskbar badge count, and the two sounds. Settings needed: `notificationMode`, `inAppNotificationsEnabled`.
10. **Accessibility.** Gio has no live regions; skip the announcer or route the announcement strings (§5.8) to the platform's accessibility notification API if tern gains one.
11. **Welcome page.** Only needed if tern runs its own first-run flow; tern connects to existing machines, so it can skip it. If ported, its data needs provider readiness per computer, a project-history scan and the import commands.
12. **Fonts.** Inter here, SF Pro on macOS; the boot failure page uses the system stack before the app CSS loads, so on macOS it is SF Pro, and its mono stack resolves to SF Mono.
13. **Model changes in `app/ui/model.go`.** `SignIn` needs Clerk's step names and per-field errors rather than one `Error`; `Scene` needs an auth-gate enum (booting, boot failed, session error, sign-in, pairing, app), an empty-state enum with its parameters (scratch available, environment count, signed in), a banner list per environment, and a toast queue.

## Verification

An adversarial pass on 2026-11-18 against the pinned build at 1440×900 @2x, in a throwaway paired session. As in the extraction, I only changed what my own browser saw. I mocked the session request as signed-out, aborted it, aborted `main-BMVwsd-_.js`, used offline emulation, and raised placeholder toasts through the shipped `toastManager`. The only form action was an empty-token submit on `/pair`, which throws before any request (the request log showed no new `browser-session` POST). The Clerk modal was re-opened on the hosted app signed out and dismissed with Escape, with nothing typed or submitted. Screenshots are in the private reference folder, never in the repo.

### Live re-measurements (both themes unless noted)

| Surface | Values checked | Result |
|---|---|---|
| `/pair` StandalonePage (M\*) | page fill and text; card box 576×368 at x 432, padding, radius, fill, border, shadow, backdrop; header row; wordmark size and fill; "Code" type and colour; h1 and description type, colours and offsets 25/77/110 | match |
| `/pair` form (M\*) | label; input box, fill, border, drop shadow, lip, placeholder; focus-visible ring; Continue and Reload app boxes, fills, borders, shadows; info box fill, border, text; Tab order | input ring and Continue shadow wrong (fixed); lip added |
| `/pair` error (M\*) | Alert box (one line 526×46), fill, border, text, description; card 430; focus after submit | match, new values added |
| Root error view (M\*) | button widths; copy icon colour and margins; report box; "Error report" label; `pre` type, fill, border, colour; card 509 | match |
| Boot failure page (M\*) | column geometry; h1, p, Reload and secondary button styles; command row; copy icon button; `pre`; summary and marker in both states; repeat-failure open state; fg/bg/line colours; link decoration; `theme-color` | boot `theme-color` wrong (fixed) |
| Page not found (M) | main box and padding; column; h1 and p type and colour; Go home box, fill, shadow; body fill | match; Go home fill and shadow added |
| Welcome (M) | column geometry and padding; h1, subtitle, label colours; section borders and grid; agent rows, "Ready", note; add-agents trigger at rest and hover; projects row, checkbox (dark), permission line (dark); Start; telemetry notice; scrollbar gutter | match; hover and gutter behaviour added |
| Reconnecting banner (M\*) | box, padding, row, icon, title, description, action and its corner radii; `::before` border, fill, tint, blur, shadow; opacity transition | match |
| Toasts (M) | loading and stacked-end layouts and sizes; fill, title, description, close orb, action colours | match; colours and root attributes added |
| Clerk start step (H) | card box, card padding and gap, backdrop, close button, title, subtitle, label, input at rest and focus, primary at rest and hover, social boxes, links, divider, footer fill, Escape | primary shadow colour wrong (fixed); dark focus ring and hover measured |
| Hosted home (H, light) | title, description, content width, step heights, CommandBlock, header | match |

In total this is well over 25 values in each theme.

### Re-derived from source

`standalone-page.tsx:5-36`, `PairingRouteSurface.tsx:18-290`, `pair.tsx:9-61`, `__root.tsx:62-321`, `environments/primary/auth.ts:62-395`, `hostedPairing.ts:34-85`, `connection/onboarding.ts:13-24` with `packages/client-runtime/src/connection/onboarding.ts:77-169`, `clerkAppearance.ts:4-48`, `authRedirect.ts:6-20`, `publicConfig.ts:79-108`, `index.html:1-388, 389-688`, `bootstrap.ts:4`, `main.tsx:14-60`, `chunkReloadGuard.ts`, `bootError.ts:1-395`, `FirstRunGate.tsx:31-221`, `empty.tsx`, `NoActiveThreadState.tsx`, `NoProjectsHero.tsx`, `_chat.index.tsx`, `HostedEmptyState.tsx`, `InstallT3Commands.tsx`, `CommandBlock.tsx`, `ThreadRouteView.tsx:95-236`, `alert.tsx:7-145`, `useSystemComposerBanners.tsx:60-230`, `diagnosis.ts:60-200`, `presentation.ts:60-95`, `ConnectionStatusDot.tsx`, `hostFreeze.ts`, `HostFreezeToastCoordinator.ts`, `SlowRpcRequestToastCoordinator.tsx`, `requestLatencyState.ts`, `PermissionUpdateNotice.ts`, `LegacyThreadMigrationToast.ts`, `KeybindingsUpdateToast.tsx`, `desktopUpdate.toast.tsx`, `ThreadNotificationCoordinator.tsx`, `.logic.ts`, `threadNotifications.ts`, `backgroundThreadAnnouncement.ts`, `threadActivityAnnouncement.ts`, `useLiveRegion.ts`, `useTheme.ts:236-300, 455-495`, `WelcomePage.tsx:124-346, 425-445`, the three agent and CLI authorization surfaces, `T3ConnectSidebarSignIn.tsx`, `SettingsSidebarNav.tsx:425-442`, and the tern files named in §8. The string tables in §2, §4, §5.6, §5.7, §5.8 and §5.9 were checked against these, plus `CloudEnvironmentConnectList.tsx:168-186` and `PullRequestGuideRunCoordinator.tsx:85-96` for the last catalogue rows. The Welcome page's own toasts were spot-checked (`WelcomePage.tsx:166-235, 520, 1080`).

### What was wrong, now fixed

1. The primary Button shadow was given as `#1b4ed83d` / `#346bf13d`, with no inset highlight in dark. Measured: `inset 0 1px #ffffff29` in both themes plus `0 1px 2px #194fd93d` / `#326df23d`, as in 02-primitives (§3.2, §3.9). Clerk's primary has the drop shadow only, `#194fd93d` / `#326df23d` (§3.6).
2. The input focus ring was `#1b4ed83d`, with a probe value `#346bf13d` for dark. Measured: `#194fd93d` / `#326df23d` (§3.2).
3. The boot `theme-color` was "`oklch(0.992 0 0)` in light at boot". At boot it is `#ffffff` / `#000000`. `oklch(0.992 0 0)` is what `syncBrowserChromeTheme` writes later on a thread page; page-not-found gets `oklch(0.985 0 none)` (§3.7).
4. The standalone scrollbar gutter was described as "6px each side when the page scrolls". It is `stable both-edges`, reserved even with nothing to scroll (§3.1).
5. `connectPairing` was cited in the runtime package. It is the web command in `connection/onboarding.ts:13-24`, wrapping `registerPairing` (101-169). That also runs a known-address check and a protocol check before redeeming the code (§5.2).
6. The notification gate said runs stopped by the user never notify. Stop-requested runs only lose their completion notification; input notifications still fire (§5.7).
7. The banner grace was described for `reconnecting` only. It applies to each `connecting` and `reconnecting` attempt, and the title says "is reconnecting" for both (§2.18, §4.5).
8. FirstRunGate was summarised as "the recovery screen after 4,000 ms". Only "Still connecting" waits, and only after settings hydrate; "Could not read settings" shows at once (§2.1, §4.1).
9. The boot failure page was tied to the main bundle failing to import. Any rejection of the `startup` promise also reaches it. Lazy chunks that fail after main has loaded first reload the page once per tab; the main bundle has no such retry (§2.1, §5.1).
10. Line citations: `_chat.tsx:50-55` became 49-54; the coordinator's badge range 374-405 became 386-415.

### Omissions filled

- Gate: `/pair` opening with the gate's error, `/pair` redirecting to `/` when already signed in, the four routes that skip the shell, unknown drafts redirecting to `/`, and the chunk reload guard.
- Boot failure page: copy feedback and hints, the announcement timing, the GitHub issue link and its parameters, and the marker's computed 1px border.
- Structure: `workspaceAreaProps` attributes on the hero and the draft error, the descriptions of the incomplete and framed `/approve-agent` states, `/connect-agent`'s header context, and Alert's slot sorting and first-line action box.
- Banner: re-enabling a switched-off environment, the 20 s "Disconnect server" delay and what it does, and the failure toast copy.
- Toasts: root attributes and colours, slow-request thresholds and exclusions, host-freeze rounding, the permission notice's close-orb dismissal, fallback copy for the T3 Connect add and PR guide toasts, and the 2 s "Copied" reset on the root error view.
- Notifications: which threads are skipped, nothing watched while both settings are off, mode labels, sound independent of the visual path, same-tag replacement, and badge text alignment.
- Hosted home discovery polling every 5 s, the telemetry copy and link hover, the Welcome trigger hover, the Clerk hover and dark focus ring, sidebar Alert hexes, the one-line error Alert height, and the diagnosis "offline" condition.
- A measuring note in §7.9. The measurement window is hidden, so theme switches and `@starting-style` transitions only run on forced frames. With frames forced, the Welcome page and app shell follow a live theme change and `/pair` still does not, which confirms the extractor's quirk for standalone pages only.

### Privacy

I searched the doc for machine, project, thread, branch and account names seen during the session, plus emails, private hosts and UUIDs. None appear. The only hosts are `127.0.0.1:3773`, the reserved `192.0.2.1`, and public T3 URLs. Placeholder text stands in everywhere else. No edit was needed.

### Still open

§7 lists what remains unmeasured. The Clerk steps after the email, the hosted pairing page's lifetime on the real origin, live notifications, the banner's error variant and Disconnect action, the boot page's copy feedback, and the probe-only empty states still need a check with a test account or a desktop build.
