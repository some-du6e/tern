# Server config, settings, providers and models

This is the reference for the non-orchestration half of the T3 Code WebSocket
API: the server config, providers and their models, model selections and option
controls, server settings, keybindings, provider setup, usage, storage and
diagnostics. It comes from upstream `pingdotgg/t3code` at `80455376`, checked
against a live `0.1.3-nightly.20261023.3592` server with a read-only
(`orchestration:read`) session. Paths are relative to the upstream repo.

Connecting, auth, WebSocket framing and turn items are in
[t3-connect.md](t3-connect.md). Thread commands are in
[orchestration.md](orchestration.md). This doc only points at the commands
that change a thread's model.

Contract files: `packages/contracts/src/server.ts`, `settings.ts`, `model.ts`,
`modelSelection.ts`, `providerInstance.ts`, `providerPolicy.ts`,
`providerSetup.ts`, `providerUsageLimits.ts`, `usage.ts`, `background.ts`,
`resourceTelemetry.ts`, `keybindings.ts`, `acpRegistry.ts`, `rpc.ts`. Server
handlers are in `apps/server/src/ws.ts` around line 2180. Required scopes are
in `apps/server/src/auth/RpcAuthorization.ts`.

## The short version

- Subscribe to `subscribeServerConfig` once per connection. The first item is
  a full `snapshot`. After that you get `providerStatuses`, `settingsUpdated`
  and `keybindingsUpdated` events, and each one replaces its whole slice.
- The config is about 380 KB on the test server, and 305 KB of that is one
  provider's `workspaceSnapshots`. Every `providerStatuses` event resends the
  whole `providers` array.
- A model choice is `{instanceId, model, options?}` with `options` as
  `[{id, value}]`. The available options come from
  `providers[].models[].capabilities.optionDescriptors`, which are `select` or
  `boolean` controls.
- Change a thread's model with `orchestration.dispatchCommand` and
  `{type:"thread.model-selection.set", commandId, threadId, modelSelection}`.
- Server settings are one big struct. Read it with `server.getSettings` (it's
  also in the config) and change it with `server.updateSettings {patch}`.
  Secrets come back redacted.
- "Client settings" such as fonts, chat width and send shortcut never reach the
  server. The official app stores them on the device.

## server.getConfig and subscribeServerConfig

| Tag | Kind | Payload | Returns | Scope |
| --- | --- | --- | --- | --- |
| `server.getConfig` | call | `{}` | `ServerConfig` | `orchestration:read` |
| `subscribeServerConfig` | stream | `{environmentThemes?, usageLimitSources?, usageLimitsCommand?}` | `ServerConfigStreamEvent` | `orchestration:read` |
| `server.probe` | call | `{}` | `{}` | `orchestration:read` |

`server.probe` is the cheap liveness check. The official client uses it when
`environment.capabilities.connectionProbe` is true and falls back to
`server.getConfig` otherwise (`packages/client-runtime/src/rpc/session.ts:394`).

### ServerConfig

`server.ts:582`. Sizes are from the live server. Total was 378,833 bytes of
JSON.

| Field | Type | Bytes | Notes |
| --- | --- | --- | --- |
| `environment` | `ExecutionEnvironmentDescriptor` | 1.5 K | `environmentId`, `label`, `platform {os, arch, machine}`, `serverVersion`, `orchestrationProtocolVersion`, `capabilities {...}`. Same as `/.well-known/t3/environment`. |
| `auth` | `ServerAuthDescriptor` | 250 | `policy`, `bootstrapMethods`, `sessionMethods`, `sessionCookieName`, `serverUpdateScope`. |
| `cwd` | string | | Server's working directory. |
| `keybindingsConfigPath` | string | | Path of the user's keybindings JSON on the host. |
| `keybindings` | `ResolvedKeybindingRule[]` | 15.6 K | Defaults merged with user rules. 82 entries live. |
| `issues` | `ServerConfigIssue[]` | | Keybindings file problems: `{kind:"keybindings.malformed-config", message}` or `{kind:"keybindings.invalid-entry", message, index}`. |
| `providers` | `ServerProvider[]` | 354 K | The heavy part. See below. |
| `availableEditors` | `EditorId[]` | | Editors the host can open. |
| `remoteOpenTargets?` | array | | |
| `directEndpoints?` | `{kind:"lan"\|"tailnet", httpBaseUrl}[]` | | Hints only. Verify the environment id before using one. |
| `observability` | object | | `logsDirectoryPath`, `localTracingEnabled`, and for each of traces, metrics and logs: `otlp*Url?`, `otlp*Source?` (`environment`, `bootstrap` or `settings`), `otlp*Variable?`, `otlp*Enabled`. |
| `settings` | `ServerSettings` | 6.9 K | Redacted. Same value as `server.getSettings`. |
| `shellResumeCompletionMarker?`, `threadResumeCompletionMarker?`, `shellThreadFamilies?`, `threadSnapshotPagination?`, `threadFind?`, `threadFindProgressive?`, `reasoningMessages?`, `shellRevealInFileManager?` | boolean | | Feature flags. Check before using the matching option. |
| `shellRevealInFileManagerKind?` | | | |
| `scratchWorkspaceRoot?`, `newProjectsRoot?` | string | | Host paths for scratch and new projects. |
| `lastEventLoopStall?` | `{detectedAt, durationMs, loadAverage?, cpuCount}` | | Sent for a few minutes after the host froze. Show it, don't cache it. |
| `environmentThemes?` | `EnvironmentTheme[]` | | Never in a snapshot. Arrives as a stream event if you opt in. |
| `usageLimitSources?` | `UsageLimitSourceSnapshot[]` | | Same, stream-only. |

### ServerProvider

One entry per provider instance (`server.ts:205`). `instanceId` is the routing
key. Two instances can share a `driver` (two Codex accounts, for example).

| Field | Type | Meaning |
| --- | --- | --- |
| `instanceId` | string | Stable id. Default instances use the driver name: `codex`, `claudeAgent`, `cursor`, `grok`, `opencode`, `antigravity`, `muse`, `pi`. |
| `driver` | string | Driver kind. `acpRegistry` for ACP registry agents. |
| `displayName?`, `accentColor?`, `iconUrl?`, `badgeLabel?` | string | Presentation. |
| `enabled` | boolean | Turned on in settings. |
| `installed` | boolean | The CLI exists. |
| `version`, `installedVersion?` | string or null | |
| `status` | `ready` \| `warning` \| `error` \| `disabled` | Health. |
| `availability?` | `available` \| `unavailable` | Missing means available. Unavailable implies `installed` and `enabled` are false. |
| `unavailableReason?`, `message?` | string | Human text, for example "Cursor is disabled in T3 Code settings." |
| `auth` | `{status, type?, label?, email?, action?, canLogout?, subscriptionSharing?, profileId?, workspaceId?}` | `status` is `authenticated`, `unauthenticated` or `unknown`. `action` is an ACP URL-auth prompt (see ACP registry). |
| `checkedAt` | ISO time | Last health check. |
| `models` | `ServerProviderModel[]` | See [Models](#models). |
| `updateRequiredModels?` | `{slug, name, badge?, minVersion}[]` | Models that need a newer CLI. Kept out of `models`; show as a notice. |
| `slashCommands` | `{name, description?, input?: {hint}}[]` | Provider-wide `/` commands. |
| `skills` | `{name, description?, path, scope?, enabled, displayName?, shortDescription?, userInvocationOnly?, userInvocable?}[]` | Provider-wide skills (`$` menu). |
| `workspaceSnapshots?` | `{cwd, checkedAt, slashCommands, slashCommandsPending?, skills, optionDescriptors?}[]` | Per-directory commands and skills. |
| `usageLimits?` | `ServerProviderUsageLimits` | Rate-limit bars. See [Usage](#usage-and-usage-limits). |
| `versionAdvisory?` | `{status, currentVersion, latestVersion, updateCommand, canUpdate, canInstallVersion?, checkedAt, message}` | `status` is `unknown`, `current` or `behind_latest`. |
| `compatibilityAdvisory?` | `{status, latestVersionStatus?, message, recommendedVersion, recommendedRange}` | `status` is `unknown`, `supported`, `graceful`, `unsupported` or `broken`. |
| `updateState?` | `{status, startedAt, finishedAt, message, output}` | Progress of `server.updateProvider`. `status` is `idle`, `queued`, `running`, `succeeded`, `failed` or `unchanged`. |
| `setup?` | `{canAuthenticate, canInstall, documentationUrl?}` | Whether `provider.auth.*` and `provider.install.*` apply. |
| `nativeSessions?` | `{canList, canLoad, canResume, canDelete?}` | Importing the provider's own sessions. |
| `continuation?` | `{groupKey}` | Instances with the same key share history, e.g. `claude:home:/root/.claude`. A started thread can only move between instances with its key. |
| `showInteractionModeToggle?` | boolean | Show the plan/default toggle. |
| `supportedRuntimeModes?` | `RuntimeMode[]` | Missing means all four. Grok reports `approval-required`, `auto`, `full-access`. |
| `reportsContextWindow?` | boolean | Provider reports context usage. |
| `requiresNewThreadForModelChange?` | boolean | Model can't change after the first turn. |
| `supportsConversationRollback?`, `supportsTextGeneration?`, `configurableProviders?` | boolean | `supportsTextGeneration` missing means true. |
| `runtimePaths?` | `{homePath, shadowHomePath}` | |

The live server had eight providers. Codex and Claude were `ready`, the other
six were `disabled` with empty or near-empty `models`.

### What makes it heavy

Live numbers for the Claude instance: `workspaceSnapshots` 305 KB (16
directories, each with 64 slash commands and 13 skills, every description
repeated), `slashCommands` 15 KB, `models` 8.4 KB. Codex `skills` was 8 KB.
Keybindings 15.6 KB. Everything else is small.

For a small client:

- Read `providers[].models` for the picker and `settings` for defaults.
- For the `/` and `$` menus, prefer the matching `workspaceSnapshots[]` entry
  where `cwd` equals the thread's worktree or project root, and fall back to the
  provider-wide `slashCommands` and `skills`. That's
  `packages/client-runtime/src/providerSkills.ts:118`.
- A snapshot is fresh for 5 minutes (`PROVIDER_WORKSPACE_SNAPSHOT_TTL_MS`).
  When the composer opens on a directory with no fresh snapshot, the web app
  calls `server.refreshProviders {instanceId, cwd}`, then reads the new entry
  from the result or the next `providerStatuses` event. `slashCommandsPending:
  true` means the scan is still running
  (`apps/web/src/components/chat/useProviderWorkspaceSnapshotRefresh.ts`).
- Don't refetch the config on a timer. Stay on the stream.

### Stream events

`server.ts:688`. Every event has `version: 1` and a `type`:

| `type` | Payload | Apply by |
| --- | --- | --- |
| `snapshot` | `{config: ServerConfig}` | Replacing everything. Sent first, and again on every reconnect. |
| `providerStatuses` | `{providers: ServerProvider[]}` | Replacing `config.providers` with the full list. |
| `settingsUpdated` | `{settings: ServerSettings}` | Replacing `config.settings`. Already redacted. |
| `keybindingsUpdated` | `{keybindings, issues}` | Replacing both. |
| `environmentThemesUpdated` | `{themes: EnvironmentTheme[]}` | Only with `environmentThemes: true`. |
| `usageLimitSourcesUpdated` | `{sources: UsageLimitSourceSnapshot[]}` | Only with `usageLimitSources: true`. |

Server behaviour (`apps/server/src/ws.ts:3064`):

- `providerStatuses` is debounced by 200 ms and only sent when the JSON
  differs from the last one. It still carries every provider, workspace
  snapshots included. Expect a few hundred KB per event. That happens on
  health refreshes (every `providerHealthRefreshInterval`, 5 minutes by
  default), after `server.refreshProviders`, and while a provider updates.
- The opt-in flags exist because older clients reject unknown event types and
  lose the whole subscription. Only send `environmentThemes: true` or
  `usageLimitSources: true` if you handle those types. `usageLimitsCommand:
  true` makes the server add a `/usage-limits` slash command, which a client
  must handle itself or it gets sent as a prompt.
- Ack every chunk (see t3-connect.md). The server buffers changes until the
  snapshot is acked.

The official client keeps one projection and applies each event as above
(`packages/client-runtime/src/state/serverConfigProjection.ts`). It reuses
unchanged sub-objects to avoid re-rendering, which a Rust client can skip.

## Models

### Model entries

`ServerProviderModel` (`server.ts:74`):

```ts
{
  slug: string,             // what you put in modelSelection.model
  name: string,             // "Claude Opus 5.5"
  shortName?: string,
  subProvider?: string,     // e.g. upstream vendor for OpenCode models
  aliases?: string[],       // ["opus", "opus-5.5", "claude-opus-5.5"]
  badge?: "new",
  isCustom: boolean,        // user-added in settings
  isDefault?: boolean,      // provider's default model
  isLegacy?: boolean,       // goes in the collapsed "Legacy models" group
  capabilities: { optionDescriptors?: ProviderOptionDescriptor[] } | null
}
```

Live Codex list: `gpt-6.1-sol gpt-6-astra(default) gpt-6-luna`, five legacy
models, and three custom ones. Live Claude list: 13 built-in models, 9 of them
legacy, plus 6 custom ones.

### Model selection

`modelSelection.ts`:

```json
{ "instanceId": "claudeAgent", "model": "claude-opus-5-5",
  "options": [ { "id": "effort", "value": "high" }, { "id": "fastMode", "value": false } ] }
```

- `options` is optional. Leave it out when nothing is set.
- `value` is a non-empty string for `select` options and a boolean for
  `boolean` options.
- The decoder also accepts the legacy shapes `{provider, model}` (as
  `instanceId`) and `options` as an object `{"effort":"high"}`. Always send the
  array form.

### Option descriptors

`model.ts:7`. Two kinds:

```ts
// select
{ id, label, description?, type: "select",
  options: [{ id, label, description?, isDefault? }],
  currentValue?: string,
  promptInjectedValues?: string[] }
// boolean
{ id, label, description?, type: "boolean", currentValue?: boolean }
```

Render a `select` as a dropdown of `options[].label` and a `boolean` as a
toggle. The value to show when the selection has no entry for that id is
`currentValue`, else the option with `isDefault`
(`packages/shared/src/model.ts:240`, `getProviderOptionCurrentValue`). A
`boolean` with no `currentValue` and no selection is off.

Descriptors are per model. When the user switches model, keep only the
option ids the new model also has. The web app re-normalizes the selection
against the new model's descriptors: a stored value that isn't one of the
choices falls back to `currentValue` or the default.

Seen on the live server:

| Driver | `id` | Type | Choices |
| --- | --- | --- | --- |
| codex | `reasoningEffort` | select | `low medium high xhigh max ultra`. The default varies by model (`low` on gpt-6.1-sol, `medium` on others). Each descriptor also carries `currentValue`. |
| codex | `serviceTier` | select | `default` ("Standard"), `priority` ("Fast", "2x speed, increased usage") |
| claudeAgent | `effort` | select | `low medium high xhigh max ultracode ultrathink`, varies by model. `promptInjectedValues: ["ultrathink"]`. |
| claudeAgent | `fastMode` | boolean | "Fast Mode" |
| claudeAgent | `contextWindow` | select | `200k`, `1m` (default on claude-opus-4-6) |
| claudeAgent | `thinking` | boolean | haiku-4-5, `currentValue: true` |

From driver source:

| Driver | `id` | Source |
| --- | --- | --- |
| opencode | `variant` (select, reasoning levels from the model) and `agent` (select, e.g. `build`, `plan`) | `packages/provider-opencode/src/server/status.ts:204` |
| pi | `thinking` (select of thinking levels) | `packages/provider-pi/src/server/thinkingCapabilities.ts:34` |
| grok | `reasoningEffort` | `packages/provider-grok/src/server/status.ts:203` |
| muse | `reasoningEffort` | `packages/provider-muse/src/server/modelCatalog.ts:59` |
| cursor | ids from the Cursor SDK model parameters | `packages/provider-cursor/src/server/sdkModel.ts` |
| acpRegistry | whatever the agent's ACP `configOptions` returns | `packages/provider-acp-registry/src/server/driver.ts:97` |

Treat ids as opaque and render from the descriptor. Two ids get special
handling in the web app:

- `fastMode`: if the model has a boolean `fastMode` and the selection doesn't
  mention it, the web app adds `{id:"fastMode", value:false}`. Cursor can
  default Fast to on, and T3 wants it off unless the user chooses it
  (`apps/web/src/components/chat/composerProviderState.tsx:74`).
- `agent` with a `plan` choice: hidden when the client setting
  `planModeEnabled` is off (`apps/web/src/providerModels.ts:66`).

The composer's "effort" control is the first `select` descriptor. The
reasoning-like ids are `reasoningEffort`, `effort`, `reasoning`, `variant`,
`reasoning_effort` and `thinking`
(`packages/client-runtime/src/state/threadExecution.ts:148`,
`modelTraits.ts:17`). The `composer.effort`, `composer.increaseEffort` and
`composer.decreaseEffort` keybindings step through that descriptor's choices.

### Prompt-injected values (Ultrathink)

`promptInjectedValues` lists choices that aren't settings. They're words in
the prompt. For Claude, picking "Ultrathink" makes the web app edit the prompt
to start with `Ultrathink:\n`, and the picker shows "Ultrathink" while the
prompt contains the word. The value stored in `options` stays a real setting:
when normalizing, a prompt-injected value resolves to the default choice
(`packages/shared/src/model.ts:144`). The prefix is skipped for slash commands
(`applyClaudePromptEffortPrefix`, `model.ts:508`).

So, don't send `{"id":"effort","value":"ultrathink"}`. Send the real effort,
or leave it out, and prefix the text. The PoC sends the value and prefixes,
which is wrong on the first half.

### What gets sent

The web app sends only options the user set explicitly, normalized against the
descriptors, plus the implicit `fastMode: false` (`composerProviderState.tsx:149`,
`buildExplicitProviderOptionSelectionsFromDescriptors`). Defaults are left out.
That keeps a thread's choice stable if the provider's default changes later.

### Workspace option descriptors

`workspaceSnapshots[].optionDescriptors` exists in the contract and the server
merges it per directory (`apps/server/src/provider/ProviderRegistry.ts:131`),
but the web app at this commit doesn't read it. The picker uses model
capabilities only. None of the live snapshots carried it. Ignore it for now.

### How the web picker builds its list

`apps/web/src/components/chat/ModelPickerContent.tsx`, with helpers in
`apps/web/src/modelSelection.ts`, `modelOrdering.ts` and `providerInstances.ts`.

1. **Providers.** One sidebar entry per provider with `enabled: true`, plus a
   "Favorites" entry first. Default instances come before custom instances of
   the same driver. A provider is selectable when `enabled`,
   `availability !== "unavailable"` and `status === "ready"`
   (`isProviderInstancePickerReady`). Enabled providers that aren't ready show
   a status message and a setup button when `setup` allows it, the CLI isn't
   installed, or `auth.status` is `unauthenticated`.
2. **Models per provider** (`getAppModelOptionsForInstance`):
   - start with `models` where `isCustom` is false;
   - append custom models from
     `settings.providerInstances[instanceId].config.customModels`, rebuilt from
     settings because the provider snapshot can lag removals. Each entry is a
     slug string or `{slug, name?, capabilities?}`. Built-in slugs win over a
     custom one with the same slug;
   - drop `providerModelPreferences[instanceId].hiddenModels`. Custom models
     are never hidden;
   - sort by `providerModelPreferences[instanceId].modelOrder` (listed slugs
     first, in that order), then by server order.
3. **Favorites.** `settings.favorites` is `[{provider: instanceId, model: slug}]`.
   Inside a provider tab favorites sort to the top. The Favorites tab shows
   only favorites, ordered by provider then the above. The picker opens on
   Favorites when there are any.
4. **Legacy.** Inside a provider tab, models with `isLegacy` that aren't
   favorites go into a collapsed "Legacy models" group at the bottom.
5. **Update notices.** `updateRequiredModels` render as a notice like "update
   the CLI to use X".
6. **Search** ranks across all providers by `name`, `shortName`,
   `subProvider`, driver and provider display name, with favorites boosted
   (`modelPickerSearch.ts`).
7. **Locked provider.** Once a thread has started, the picker only allows
   instances with the same `driver` and, if set, the same
   `continuation.groupKey`. The exception is a thread whose provider session
   reports `supportsProviderSwitchingViaHandoff`, which can switch freely
   (`apps/web/src/components/ChatView.tsx:1178`,
   `packages/client-runtime/src/state/threadWorkflows.ts`).
8. **Selection** resolves the typed or stored model by exact slug, then
   case-insensitive name, then `aliases`, then the driver alias table
   `MODEL_SLUG_ALIASES_BY_PROVIDER` in `model.ts`
   (`resolveSelectableModel`, `packages/shared/src/model.ts:428`).

If you only want a simple picker: enabled and ready providers, built-in models
minus hidden ones plus custom ones, `modelOrder` sort, favorites on top, legacy
collapsed.

### Defaults

For a new thread the web app uses, in order
(`apps/web/src/hooks/useHandleNewThread.ts:131`):

1. the model the user carried over from the current draft or thread;
2. `settings.projectSettingsOverrides[projectId].defaultModelSelection`;
3. `settings.defaultModelSelection`;
4. if those are null, the first enabled provider's `isDefault` model, else its
   first model, else `DEFAULT_MODEL_BY_PROVIDER[driver]` from `model.ts:136`.

A stored default that the provider no longer lists falls back to that
provider's default model (`resolveEffectiveDefaultModelSelection`).
`defaultRuntimeMode` (default `full-access`) and the project's override work
the same way.

Live values: `defaultModelSelection` was `{"instanceId":"claudeAgent","model":"claude-opus-5-5"}`.

### Changing a thread's model

Send `orchestration.dispatchCommand` (scope `orchestration:operate`) with one of
these (`packages/contracts/src/orchestrationV2.ts:2904`, `:3135`):

```json
{"type":"thread.model-selection.set","commandId":"...","threadId":"...",
 "modelSelection":{"instanceId":"claudeAgent","model":"claude-sonnet-5-5","options":[{"id":"effort","value":"high"}]}}

{"type":"provider.switch","commandId":"...","threadId":"...","modelSelection":{...}}
```

If `environment.capabilities.serverResolvedCommandContext` is true, always
send `thread.model-selection.set` and the server works out whether it's a
provider switch. On older servers, send `thread.model-selection.set` when the
instance stays the same and `provider.switch` when it changes
(`packages/client-runtime/src/operations/commands.ts:633`,
`packages/shared/src/model.ts:17`).

Related commands, same shape without `modelSelection`:

```json
{"type":"thread.runtime-mode.set","commandId":"...","threadId":"...","runtimeMode":"auto-accept-edits"}
{"type":"thread.interaction-mode.set","commandId":"...","threadId":"...","interactionMode":"plan"}
```

`runtimeMode` is `approval-required`, `auto-accept-edits`, `auto` or
`full-access`. Limit the choices to `supportedRuntimeModes` when the provider
reports it. `interactionMode` is `default` or `plan`. Only offer it when
`showInteractionModeToggle` is true.

`message.dispatch` also takes an optional `modelSelection`, and `thread.create`
and `launchThread` take `modelSelection`, `runtimeMode` and `interactionMode`.
See orchestration.md for those.

Before changing the model on a started thread, the web app blocks the change
and asks for a new chat when the provider changes and the session doesn't
support handoff, or when either provider has
`requiresNewThreadForModelChange: true`
(`apps/web/src/components/ChatView.logic.ts:1007`).

## Settings

### Two kinds of settings

| | Server settings | Client settings |
| --- | --- | --- |
| Schema | `ServerSettings`, `settings.ts:1103` | `ClientSettingsSchema`, `settings.ts:309` |
| Stored | `settings.json` on the host, secrets in a separate store | On the device, through the desktop bridge or browser storage |
| Read | `server.getSettings`, `config.settings`, `settingsUpdated` | Local only |
| Write | `server.updateSettings {patch}` | Local only |

The web app merges them into `UnifiedSettings` and splits each patch by key:
keys that exist in `ServerSettings` go to the server, the rest stay local
(`apps/web/src/hooks/useSettings.ts:242`). `favorites` and
`providerModelPreferences` exist in both. The server copy is null until the
client migrates its local copy with `migrateModelPreferences`. Once it's set,
the server copy is authoritative, and an empty array means "no favorites",
not "unmigrated".

A third-party client should treat client settings as its own preferences and
not try to sync them.

### Reading and writing

| Tag | Payload | Returns | Scope |
| --- | --- | --- | --- |
| `server.getSettings` | `{}` | `ServerSettings` (redacted) | `orchestration:read` |
| `server.updateSettings` | `{patch: ServerSettingsPatch, providerInstanceMutation?}` | `ServerSettings` (redacted) | see below |

The patch (`settings.ts:1424`) is partial. Each key you send replaces that
setting. The exceptions:

| Key | Merge rule |
| --- | --- |
| `setModelFavorites` | `[{provider, model, favorite: boolean}]` adds or removes favorites. Use this rather than writing `favorites`. |
| `setModelsHidden` | `[{provider, model, hidden: boolean}]` |
| `setProviderModelOrder` | `{provider, modelOrder: string[]}` replaces one instance's order. |
| `migrateModelPreferences` | `{favorites?, providerModelPreferences?}`, applied only while the server values are null. |
| `textGenerationModelSelection` | Partial `{instanceId?, model?, options?}`. |
| `worktreeCleanup`, `storageCleanup`, `backgroundActivity`, `observability`, `sourceControlWritingStyle` | Nested fields are optional and merge. |
| `projectSettingsOverrides` | Per project id. A value replaces that project's override set, `null` removes it. Omitted `providerInstances` keeps its stored value. |
| `projectAgentBrowserAccessOverrides`, `projectScriptOverrides`, `projectAutoPullOverrides` | Per project id, `null` removes. |
| `usageLimitSources`, `usagePriceOverrides`, `usageModelAliases` | Per key, `null` removes. |
| `sourceControlHosts` | Per host. Patched fields replace whole. Redaction markers keep the secret, empty removes it. |
| `providerInstances` | Replaces the whole map. Prefer `providerInstanceMutation`. |

`providerInstanceMutation` (`providerInstance.ts:62`) edits one instance:

```json
{"operation":"create","instanceId":"codex_work","instance":{"driver":"codex","enabled":true,"displayName":"Codex (work)","config":{"setupMode":"existing"}}}
{"operation":"upsert","instanceId":"claudeAgent","instance":{ ...full ProviderInstanceConfig... }}
{"operation":"remove","instanceId":"codex_work"}
```

`create` fails if the id exists. Removing a default instance such as `codex`
resets it to defaults. Instance ids match `^[a-zA-Z][a-zA-Z0-9_-]*$`, 64
characters max.

`ProviderInstanceConfig` is `{driver, displayName?, accentColor?, environment?,
enabled?, config?}`. `config` is driver-specific and unknown drivers round-trip
untouched. `environment` is `[{name, value, sensitive, valueRedacted?}]`.

### Secrets and redaction

Every settings value the server sends goes through
`redactServerSettingsForClient` (`apps/server/src/serverSettings.ts:130`):

- Secret config fields (annotated `settingsSecret`, or form control
  `password`) come back as `"••••••"` (six U+2022 bullets,
  `apps/server/src/settingsSecrets.ts:23`) when set.
- Sensitive environment variables come back as `value: ""` with
  `valueRedacted: true`.

To keep a secret when saving an instance, send the bullet string or
`valueRedacted: true` back unchanged. To clear it, send an empty value. Never
log settings with real values in them. Known secret fields: Antigravity
`apiKey`, usage source `managementKey`, OpenCode `serverPassword`, EAS device
host tokens, source control host tokens.

### Scopes for updateSettings

`requiredScopesForServerSettingsPatch` (`settings.ts:1577`) and
`RpcAuthorization.ts:236`:

- any key other than `providerInstances` and `usageLimitSources` needs
  `settings:write`;
- `providerInstances`, `usageLimitSources` or a `providerInstanceMutation`
  needs `providers:manage`;
- a patch with both needs both.

### ServerSettings by settings page

The web settings pages are listed in
`apps/web/src/components/settings/settingsSearch.ts`. The tables below map
each server setting to the page that shows it. Defaults are from the schema.
Pages that only use client settings get a one-line summary.

**General** (`/settings/general`, `SettingsPanels.tsx`)

| Key | Type, default | Meaning |
| --- | --- | --- |
| `defaultModelSelection` | `ModelSelection \| null`, null | Default model for new threads. |
| `defaultRuntimeMode` | `RuntimeMode`, `full-access` | "Permissions" for new threads. |
| `textGenerationModelSelection` | `ModelSelection`, `{codex, gpt-6-luna, reasoningEffort: low}` | Model for titles, commit messages and other side jobs. |
| `threadTitleInstructions` | string, "" | Extra instructions for title generation. |
| `responseStreamingMode` | `turn` \| `paragraph`, `paragraph` | How assistant text is released to clients. |
| `snoozeLimitedThreads` | bool, false | Snooze threads that hit a usage limit. |
| `autoResumeLimitedThreads` | bool, false | Resume them when the limit resets. |
| `sidebarAutoSettleAfterDays` | 1 to 90 or null, 3 | Settle inactive threads after N days. Null is off. |
| `sidebarAutoSettleOnMerge` | bool, true | Settle a thread when its PR merges. |
| `sidebarAutoSettleKeepPullRequests` | bool, false | Don't auto-settle threads with open PRs. |
| `idleCompaction` | `onReturn` \| `beforeCacheExpires` \| `off`, optional | When to compact idle threads over 200k tokens. |
| `compactIdleThreads` | bool or mode, false | Legacy form of the same. Use `idleCompactionPatch` logic in `settings.ts:1354`: write `idleCompaction` if the server has it, else this. |
| `enableProviderUpdateChecks` | bool, true | Check for provider CLI updates. |
| `continueThreadsAfterServerUpdate` | bool, true | Resume running threads after a server restart or update. |
| `backgroundActivity` | `{schemaVersion:1, profile, baseProfile?, overrides}` | Background work profile. `profile` is `balanced`, `performance`, `battery-saver` or `custom`. `overrides` holds durations in ms (`automaticGitFetchInterval`, `providerHealthRefreshInterval`, `hostPowerMonitorActiveInterval`, `hostPowerMonitorIdleInterval`, `idleClientTtl`) and `pauseWhenHostLocked`, `pauseWhenHostLowPower`, `pauseWhenClientLowPower`, `pauseWhenOnBattery`. |
| `backgroundActivityProfile` | profile, `balanced` | Legacy single field. |
| `defaultThreadEnvMode` | `ThreadEnvMode` or omitted | "New threads": local checkout or worktree. Omitted inherits `t3.json`, then local. |
| `worktreeSubmodules` | value or null | Submodule handling in new worktrees. |
| `newWorktreesStartFromOrigin` | bool, true | "Start from remote". |
| `addProjectBaseDirectory` | string, "" | Where the add-project dialog starts. |
| `projectSettingsOverrides` | per project | "Project defaults and overrides". Keys are those in `PROJECT_SCOPED_SERVER_SETTING_KEYS` (`settings.ts:955`) plus `providerInstances: {[id]: {enabled?}}`. |

Client-only on this page: notification mode, in-app notifications, time
format, diff whitespace and layout, proactive panels, skills in slash menu,
rich text composer, collapse on scroll, `sendShortcut`, `followUpBehavior`
(`queue` or `steer`), confirmations, quit shortcut, legacy toggles
(`planModeEnabled`, `contextWindowMeterEnabled`, `legacySidebarEnabled`).

**Providers** (`/settings/providers`, `ProviderSettingsPanel.tsx`,
`ProviderInstanceCard.tsx`)

| Key | Meaning |
| --- | --- |
| `providerInstances` | Instances and their driver config. Edited with `providerInstanceMutation`. |
| `favorites`, `providerModelPreferences` | Model list editor: hide, reorder, favorite. |
| `providerHealthRefreshInterval` | ms, default 300000. "Health check interval". |
| `usageLimitSources` | `{[id]: {kind:"cliproxy", label?, url, managementKey (secret), enabled}}`. "Usage providers". |
| `cursorKeychainUsageEnabled` | bool, false. Read Cursor usage from the keychain. |
| `usagePriceOverrides` | `{[model]: {inputCostPerMillionTokens, outputCostPerMillionTokens, cacheReadCostPerMillionTokens?, cacheWriteCostPerMillionTokens?}}` |
| `usageModelAliases` | `{[model]: model}` for usage pricing. |

Driver config fields with form annotations (`settings.ts:580` onward and each
`packages/provider-*/src/*client*`):

| Driver | Fields |
| --- | --- |
| codex | `binaryPath` ("codex"), `homePath` (CODEX_HOME), `shadowHomePath`, `launchArgs`, hidden `customModels`, `setupMode` (`managed`/`existing`), `enabled` |
| claudeAgent | `binaryPath` ("claude"), `homePath` (CLAUDE_CONFIG_DIR), `autoCompactWindow` ("" or 100000 to 1000000), `idleCompaction` (`default` or a mode), `launchArgs`, hidden `customModels`, `enabled` |
| antigravity | `authMethod` (`oauth-personal`, `oauth-business`, `gemini-api-key`, `agent-platform`), `apiKey` (secret), `gcpProject`, `gcpLocation`, `binaryPath`, `customModels` |

Each field schema carries `title`, `description` and a `providerSettingsForm`
annotation (`control`: `text`, `password`, `textarea`, `switch` or `select`;
`placeholder`; `hidden`; `clearWhenEmpty`; `options`). A generic form renderer
can be built from these, but they're only reachable from TypeScript. A Rust
client will need its own per-driver form.

Custom model entries in `config.customModels` are a slug string or
`{slug, name?, capabilities?: {optionDescriptors}}`.

**Source control** (`/settings/source-control`)

| Key | Meaning |
| --- | --- |
| `defaultAutoPull` | bool, false. Pull before work starts. |
| `removeAgentCreditsOnMerge` | bool, false. Strip "Co-authored-by" agent lines. |
| `pullRequestMergeMethod` | merge method or null. |
| `automaticGitFetchInterval` | ms, default 30000. |
| `branchNamingMode` | `static`, `semantic`, `custom`. Default `static`. |
| `branchNamePrefix` | default `t3`. |
| `branchNameInstructions` | string. |
| `sourceControlWritingStyle` | `{mode: repo_conventions\|conventional_commits\|custom, customInstructions, followChangeRequestTemplates}` |
| `sourceControlWriterModelSelection` | model or null. |
| `reviewGuidesEnabled`, `reviewGuideModelSelection`, `reviewGuideInstructions` | Review guides (beta). |
| `sourceControlHosts` | Per-host accounts and tokens, secrets redacted. `server.discoverSourceControl {}` lists what the host has. |

**Integrations** (`/settings/integrations`)

`enableAgentBrowserAccess` (true), `projectAgentBrowserAccessOverrides`,
`enableAgentDeviceAccess` (false), `enableDeviceSupport` (false),
`deviceOnboardingCompleted`, `deviceHosts`, `easDeviceHosts`. The browser
profile, viewport, zoom, recording, link target and search engine settings on
this page are client settings.

**Storage** (`/settings/storage`, `StorageSettings.tsx`)

| Key | Meaning |
| --- | --- |
| `storageCleanup` | `{worktreeKeepWhen: any-local-files\|uncommitted-changes\|tracked-changes, worktreeAfterDays: 1..3650\|null, worktreeSettledAfterDays: 0..3650\|null, worktreeOnMerge, worktreeOnDelete, worktreeUnchanged, browserArtifactsAfterDays, logsAfterDays}` |
| `worktreeCleanup` | Older `null \| {mode:"off"} \| {mode:"custom", rules}`. Also per project. |
| `worktreesDirectory` | "" means default. `previousWorktreesDirectories` is server-maintained. |

**Projects** (`/settings/projects`): `defaultProjectScripts`,
`projectScriptOverrides`, `projectAutoPullOverrides` and the per-project
`projectSettingsOverrides`. Scripts run on the host, so setting them through
`projects.mutate` needs `settings:write` as well as `orchestration:operate`.

**Connections** (`/settings/connections`): `environmentIcon` (machine kind or
null). The rest of that page is pairing, sessions and endpoints, covered by
t3-connect.md.

**Appearance**: client settings except `defaultTheme` and `defaultThemeSetAt`,
which `t3 theme set` writes and RPC patches can't. **Beta**:
`guestSharingEnabled` (false). **Diagnostics**: `observability` with
`otlpTracesUrl`, `otlpMetricsUrl`, `otlpLogsUrl` ("" is off).
**Keybindings**, **SnapShots**, **Scheduled tasks**, **Archive**: see their own
sections or other docs.

### Keybindings

| Tag | Payload | Returns | Scope |
| --- | --- | --- | --- |
| `server.upsertKeybinding` | `{key, command, when?, replace?: {key, command, when?}}` | `{keybindings, issues}` | `settings:write` |
| `server.removeKeybinding` | `{key, command, when?}` | `{keybindings, issues}` | `settings:write` |

`key` is a shortcut string such as `mod+shift+k`, 64 characters max. `when` is
an expression like `!terminalFocus`, 256 max. `command` is one of
`STATIC_KEYBINDING_COMMANDS` (`keybindings.ts:59`) or `script.<id>.run`.
`replace` names an existing rule to swap out, which is how the editor changes
a shortcut. The server writes the file at `keybindingsConfigPath`, and other
clients get `keybindingsUpdated`.

Resolved rules in the config look like:

```json
{"command":"appearance.cycle",
 "shortcut":{"key":"a","metaKey":false,"ctrlKey":false,"shiftKey":true,"altKey":true,"modKey":true},
 "whenAst":{"type":"not","node":{"type":"identifier","name":"terminalFocus"}}}
```

`modKey` means Cmd on macOS and Ctrl elsewhere. `whenAst` nodes are
`identifier`, `not`, `and`, `or`. Defaults are `DEFAULT_KEYBINDINGS` in
`packages/shared/src/keybindings.ts`.

## Provider maintenance and setup

### Refresh and update

| Tag | Payload | Returns | Scope |
| --- | --- | --- | --- |
| `server.refreshProviders` | `{instanceId?, cwd?, fresh?, refreshModels?}` | `{providers}` | `orchestration:read` |
| `server.updateProvider` | `{provider: driver, instanceId?, targetVersion?}` | `{providers}` | `providers:manage` |

`server.refreshProviders` forms (`ws.ts:2283`):

- `{}` rechecks every provider and the usage-limit sources;
- `{instanceId}` rechecks one;
- `{instanceId, cwd, fresh?}` rescans that directory's slash commands and
  skills;
- `{refreshModels: true}` also drops model caches and re-lists models. It may
  start provider sessions. The settings page refresh button sends exactly
  `{refreshModels: true}`.

It only needs read scope, so a read-only client can use it, but it costs the
host real work. Don't call it on a timer.

`server.updateProvider` installs a CLI update. Show the button when
`versionAdvisory.canUpdate` is true, and pass `targetVersion` only when
`canInstallVersion` is true. Watch `updateState` in `providerStatuses`.
Failure is `ServerProviderUpdateError {provider, reason}`.

### Auth flow

All `provider.auth.*` methods need `providers:manage`. They return or stream
`ProviderAuthState` (`providerSetup.ts:121`):

```ts
{ instanceId, phase: "idle"|"starting"|"waiting"|"verifying"|"succeeded"|"failed"|"cancelled",
  flowId: string|null, authorizationUrl: string|null, expiresAt: string|null, message: string|null,
  methods?: [{id, name, description, type: "agent"|"terminal"|"credentials", accountEmail?}],
  interaction?: null
    | {type:"browser", id, url, requiresConsent, acceptsCallback?}
    | {type:"deviceCode", id, url, userCode}
    | {type:"terminal", id, output, outputOffset?}
    | {type:"credentials", id, fields:[{name, label, secret}]},
  credentialOwner?: "provider"|"t3" }
```

| Tag | Payload |
| --- | --- |
| `provider.auth.subscribe` (stream) | `{instanceId}` |
| `provider.auth.start` | `{instanceId, methodId?, returnUrl?, callbackMode?: "server"\|"client"}` |
| `provider.auth.respond` | `{instanceId, flowId, interactionId, response}` |
| `provider.auth.complete` | `{instanceId, flowId, callbackUrl}` |
| `provider.auth.cancel` | `{instanceId, flowId}` |
| `provider.auth.logout` | `{instanceId}` |

`response` is `{type:"browser", action:"accept"|"decline"}`,
`{type:"terminal", data, size?:{cols, rows}}` or
`{type:"credentials", values:{[fieldName]: string}}`.

The loop: subscribe, call `start`, render `interaction` (open `url`, show
`userCode`, show terminal `output` and send keystrokes, or show a form), call
`respond` per interaction, stop at `succeeded`, `failed` or `cancelled`. For a
browser flow that redirects back with a code, pass the redirect URL to
`complete`. Then wait for `providerStatuses` to show the new `auth.status`.

Codex has ChatGPT-specific variants (`provider.chatgpt.reconnect-profile`,
`provider.chatgpt.import-profile`, `provider.chatgpt.handoff.subscribe`,
`provider.codex.auth-callback.subscribe`). They move OAuth tokens between
machines. Skip them unless you're building Codex account management.

`provider.consumeResetCredit {instanceId}` or `{sourceId, accountId, creditId}`
spends a banked rate-limit reset. It returns `{outcome: reset|nothingToReset|noCredit|alreadyRedeemed, warning?}`.

### Install flow

For drivers with `setup.canInstall` (Antigravity on the live server).

| Tag | Payload | Scope |
| --- | --- | --- |
| `provider.install.subscribe` (stream) | `{instanceId}` | `orchestration:read` |
| `provider.install.start` | `{instanceId}` | `providers:manage` |
| `provider.install.cancel` | `{instanceId, operationId}` | `providers:manage` |
| `provider.install.remove` | `{instanceId}` | `providers:manage` |

All return `ProviderInstallState`: `{driver, operationId, phase:
idle|downloading|extracting|verifying|succeeded|failed|cancelled,
downloadedBytes, totalBytes, version, installedVersion, executablePath?,
source?: managed|local, canRemove, message}`.

## Usage and usage limits

### Rate-limit bars

`providers[].usageLimits` (`providerUsageLimits.ts:32`):

```json
{"checkedAt":"2026-10-25T03:25:45.799Z",
 "windows":[{"id":"primary","kind":"monthly","label":"Monthly","usedPercent":78,
             "resetsAt":"2026-11-17T19:34:20.000Z","windowDurationMins":43200}],
 "resetCredits":{"availableCount":0}}
```

`kind` is `session`, `weekly`, `monthly` or `other`. Optional
`credentialFingerprint`, `resetCredits.nextExpiresAt`, `resetCredits.nextCreditId`,
`externalUsage {label, url}`, and `unavailable {reason: unsupported|probeFailed,
message?}`. Claude reported `unavailable: {reason: "unsupported"}`. After a
failed probe the server keeps the last good bars.

CLIProxy sources from `settings.usageLimitSources` show up as
`usageLimitSources` events if you subscribe with `usageLimitSources: true`:
`{id, kind:"cliproxy", label, checkedAt, accounts:[{id, driver, email?, plan?,
workspaceId?, usageLimits}], error?}`.

### Token and cost summary

| Tag | Payload | Returns | Scope |
| --- | --- | --- | --- |
| `server.getUsageSummary` | `{sinceDay, untilDay, timeZone, resolution?: day\|hour, sinceTime?, untilTime?, awaitRefresh?}` | `UsageSummary` | `diagnostics:read` |
| `server.refreshUsageRates` | `{}` | `UsagePricing` | `diagnostics:read` |

Days are `YYYY-MM-DD`. `timeZone` must be an IANA name, since offsets break
across DST. The summary (`usage.ts:127`) is `{contractVersion (6), readAt,
timeZone, sinceDay, untilDay, buckets, sources, pricing, scanDurationMs}`.
Each bucket is `{day, hourStart?, provider, model, totals:{uncachedInputTokens,
cachedInputTokens, cacheCreationTokens, outputTokens, reasoningTokens},
costUsd, cacheSavingsUsd, costSource: providerReported|modelPriced|unpriced,
records, unpricedRecords, sessions, ...}`. Reasoning tokens are already in
`outputTokens`. Sum `distinctSessions` from `sources`, not bucket `sessions`.

The scan reads transcripts from disk and can take seconds. The web app calls
once, shows the result, and if any `sources[].refreshing` is true calls again
with `awaitRefresh: true` (`packages/client-runtime/src/state/server.ts:1118`).
It keeps the result for 60 seconds.

A read-only session gets `EnvironmentAuthorizationError` with
`requiredPermission: "diagnostics:read"` and `requiredScope:
"orchestration:read"`. Check `requiredPermission`. `requiredScope` is the
legacy parent for old clients.

## Storage, database and background policy

| Tag | Kind | Payload | Returns | Scope |
| --- | --- | --- | --- | --- |
| `server.getStorageCleanupReport` | stream | `{}` | `StorageCleanupReport \| null`, then updates | `orchestration:read` |
| `server.runStorageCleanup` | call | `{}` | `StorageCleanupReport` | `settings:write` |
| `server.getDatabaseStorage` | stream | `{}` | `DatabaseStorageStatus \| null` | `orchestration:read` |
| `server.getBackgroundPolicy` | call | `{}` | `BackgroundPolicySnapshot` | `orchestration:read` |
| `subscribeBackgroundPolicy` | stream | `{}` | `BackgroundPolicySnapshot` | `orchestration:read` |
| `server.reportClientActivity` | call | `ClientActivityReportInput` | nothing | `orchestration:read` |
| `server.reportHostPowerState` | call | `HostPowerSnapshot` | nothing | `environment:maintain` |

Despite the names, the two `get` storage methods are streams.

`StorageCleanupReport` (`settings.ts:1074`) is `{trigger: automatic|manual,
startedAt, finishedAt, entries, counts:{removed, kept, failed}, omittedCount,
bytesFreed}`. Entries are `{kind: worktree|browser-artifacts|logs, outcome:
removed|kept|failed, reason, path, projectPath, threadId, threadTitle, bytes,
files}`. `runStorageCleanup` deletes files right away.

`DatabaseStorageStatus` is `{databaseBytes, reclaimableBytes,
incrementalVacuumEnabled, compactCommand}`. Live: 2.0 GB with nothing to
reclaim. Suggest running `compactCommand` offline when
`incrementalVacuumEnabled` is false and `reclaimableBytes` is at least 1 GiB
(`shouldRecommendDatabaseCompaction`).

Background policy decides when the server does optional work such as git
fetches and provider health checks. Clients hold leases by calling
`server.reportClientActivity` (`background.ts:68`):

```json
{"environmentId":"...","clientId":"<stable uuid per install>","clientKind":"web",
 "visible":true,"focused":true,"recentlyInteracted":true,"appState":"active",
 "scopes":[{"type":"provider-status"},{"type":"thread","threadId":"..."}],
 "ttlMs":45000,"observedAt":"2026-10-24T12:00:00.000Z"}
```

`clientKind` is `web`, `desktop-renderer`, `mobile` or `unknown`. Scopes are
`server-config`, `provider-status {instanceId?}`, `vcs-status {cwd}`,
`git-refs {cwd}`, `diagnostics` and `thread {threadId}`. The web app reports
every 25 seconds with a 45-second TTL, always including `provider-status`,
plus a scope for each open subscription
(`apps/web/src/lib/backgroundActivityReporter.ts`). Mobile can also send
`lowPowerMode`, `batteryState` and `networkType`. A client that never reports
still works. The server then treats the host as idle and does less background
work. `BackgroundPolicySnapshot` is `{hostPower, leases, activeForegroundLeaseCount,
activeScopeKeys, shouldRunOpportunisticWork, updatedAt}`.

## Diagnostics

All need `diagnostics:read` unless noted. The Diagnostics page
(`DiagnosticsSettings.tsx`, `ResourceTelemetryDiagnostics.tsx`) uses them.

| Tag | Payload | Returns |
| --- | --- | --- |
| `server.getTraceDiagnostics` | `{}` | Span counts, slowest spans, failures and recent warning logs from the local trace file (`server.ts:372`). |
| `server.getProcessDiagnostics` | `{}` | `{serverPid, readAt, processCount, totalRssBytes, totalCpuPercent, processes:[{pid, startTimeMs, ppid, pgid, status, cpuPercent, rssBytes, elapsed, command, depth, childPids}], error}` |
| `server.getProcessResourceHistory` | `{windowMs, bucketMs}` | Buckets and top processes. |
| `subscribeResourceTelemetry` (stream) | `{}` | `ResourceTelemetrySnapshot {readAt, sampleIntervalMs, processes, groups, power, speedLimitPercent, attribution, health}` |
| `server.getResourceTelemetryHistory` | `{windowMs, bucketMs}` | `{buckets:[{startedAt, endedAt, avgCpuPercent, maxCpuPercent, maxRssBytes, ioReadBytes, ioWriteBytes, maxProcessCount}], topProcesses, health, ...}` |
| `server.retryResourceTelemetry` | `{}` | `{accepted, snapshot}`. Needs `environment:maintain` and `diagnostics:read`. |
| `server.getHostResources` | `{}` | `{sampledAt, cpuUtilization (0..1 or null), cpuCount, availableMemoryBytes, totalMemoryBytes}`. Scope `orchestration:read`, used for load balancing. |
| `server.signalProcess` | `{pid, startTimeMs, signal: SIGINT\|SIGKILL}` | `{pid, signal, signaled, message}`. Scope `environment:maintain`. Only descendants of the server. |
| `server.checkOtlpEndpoint` | `{signal: traces\|metrics\|logs, url}` | `{_tag:"Accepted", latencyMs}`, `{_tag:"Rejected", status}` or `{_tag:"Unreachable", timedOut}`. Scope `settings:write`. |

Effect `Option` fields such as `firstSpanAt` and `error` encode as
`{"_tag":"None"}` or `{"_tag":"Some","value":...}`. `DateTimeUtc` fields are
ISO strings.

## ACP registry

ACP registry agents are providers with `driver: "acpRegistry"`. Adding one is
a search, a prepare, then a `create` mutation with the agent id in `config`.

| Tag | Payload | Returns | Scope |
| --- | --- | --- | --- |
| `server.searchAcpRegistry` | `{query, limit?}` | `{agents:[{id, name, ...}]}` | `orchestration:read` |
| `server.prepareAcpRegistryAgent` | `{agentId}` | `{agentId, version, distribution, prepared: true}` (downloads it) | `providers:manage` |
| `server.uninstallAcpRegistryManagedBinary` | `{agentId}` | `{agentId, removed}` | `providers:manage` |
| `server.acceptAcpRegistryUrlAuth` | `{instanceId, elicitationId}` | `{accepted}` | `providers:manage` |
| `server.listAcpRegistrySessions` | `{instanceId, projectId, cursor?}` | `{sessions:[{sessionId, cwd, additionalDirectories, title, ...}], nextCursor, canLoad, canResume, canDelete}` | `orchestration:read` |
| `server.importAcpRegistrySession` | `{instanceId, projectId, sessionId, title?, updatedAt?}` | `{threadId, imported}` | `orchestration:operate` |
| `server.deleteAcpRegistrySession` | `{instanceId, projectId, sessionId}` | `{deleted: true}` | `orchestration:operate` |
| `server.listAcpRegistryProviders` | `{instanceId, projectId}` | `{providers:[{providerId, supported, required, current}]}` | `orchestration:read` |
| `server.setAcpRegistryProvider` | `{instanceId, projectId, providerId, apiType, baseUrl, headers?}` | `{configured: true}` | `providers:manage` |
| `server.disableAcpRegistryProvider` | `{instanceId, projectId, providerId}` | `{disabled: true}` | `providers:manage` |
| `server.logoutAcpRegistry` | `{instanceId}` | `{loggedOut: true}`, fails with `logout_unsupported` unless `auth.canLogout` | `providers:manage` |

When an agent wants the user to open a URL, its provider's `auth.action` is
`{elicitationId, url, message, createdAt?, expiresAt?}`. Open the URL, then call
`server.acceptAcpRegistryUrlAuth`. `headers` in `setAcpRegistryProvider` are
write-only secrets and never come back. Errors are `AcpRegistryOperationError
{reason, message}`.

## Scopes at a glance

From `apps/server/src/auth/RpcAuthorization.ts` and
`packages/contracts/src/clientRpcPermissions.ts`. Denials carry
`requiredPermission` (the real scope) and `requiredScope` (a legacy parent such
as `orchestration:read` for `diagnostics:read`, or `orchestration:operate` for
`settings:write`, `providers:manage` and `environment:maintain`).

| Scope | Methods in this doc |
| --- | --- |
| `orchestration:read` | `server.probe`, `server.getConfig`, `subscribeServerConfig`, `subscribeServerLifecycle`, `server.getSettings`, `server.refreshProviders`, `provider.install.subscribe`, `server.getStorageCleanupReport`, `server.getDatabaseStorage`, `server.getBackgroundPolicy`, `subscribeBackgroundPolicy`, `server.reportClientActivity`, `server.getHostResources`, `server.discoverSourceControl`, `server.searchAcpRegistry`, `server.listAcpRegistrySessions`, `server.listAcpRegistryProviders` |
| `orchestration:operate` | `orchestration.dispatchCommand` (model, runtime and interaction mode changes), `server.importAcpRegistrySession`, `server.deleteAcpRegistrySession` |
| `settings:write` | `server.updateSettings` (non-provider keys), `server.upsertKeybinding`, `server.removeKeybinding`, `server.runStorageCleanup`, `server.checkOtlpEndpoint` |
| `providers:manage` | `server.updateSettings` with `providerInstances`, `usageLimitSources` or `providerInstanceMutation`; `server.updateProvider`; `provider.auth.*` including the ChatGPT and Codex callback variants; `provider.install.start`, `.cancel`, `.remove`; `provider.consumeResetCredit`; `server.prepareAcpRegistryAgent`, `server.uninstallAcpRegistryManagedBinary`, `server.acceptAcpRegistryUrlAuth`, `server.setAcpRegistryProvider`, `server.disableAcpRegistryProvider`, `server.logoutAcpRegistry` |
| `diagnostics:read` | `server.getUsageSummary`, `server.refreshUsageRates`, `server.getTraceDiagnostics`, `server.getProcessDiagnostics`, `server.getProcessResourceHistory`, `server.getResourceTelemetryHistory`, `subscribeResourceTelemetry`, `server.retryResourceTelemetry` (plus `environment:maintain`) |
| `environment:maintain` | `server.updateServer`, `server.updateServerWithProgress`, `server.commitDesktopUpdate`, `server.signalProcess`, `server.reportHostPowerState` |

A read-only pairing (`orchestration:read`) is enough for the model picker,
reading settings and the `/` and `$` menus. It isn't enough to change a
thread's model (`orchestration:operate`), save settings (`settings:write`) or
see usage (`diagnostics:read`). Read `permissions` from
`GET /api/auth/session` and hide what the session can't do. The web app does
the same per method (`CLIENT_GUARDED_RPC_SCOPES`).

## What I checked against the live server

With an `orchestration:read` session on `0.1.3-nightly.20261023.3592`:

- `server.getConfig` returned 378,833 bytes. The first `subscribeServerConfig`
  item was a 378,874-byte `snapshot`.
- `server.getSettings` returned the same 6,885-byte settings as the config.
  OpenCode's `serverPassword` came back empty because it wasn't set.
- `server.probe` returned `{}`.
- `server.getBackgroundPolicy`, `server.getStorageCleanupReport` and
  `server.getDatabaseStorage` worked with read scope.
- `server.getUsageSummary` failed with `requiredPermission: "diagnostics:read"`.
- The option descriptors and model lists quoted above are from that config.

Nothing that writes was called.
