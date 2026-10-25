# T3 Code projects, files, git and terminal APIs, for a third-party client

This is a reference for the RPCs that deal with a project's files, its git
state, worktrees, diffs and terminals. It comes from reading upstream
`pingdotgg/t3code` at `80455376`. Paths below are relative to that repo.
Connecting, auth and the WebSocket framing are in `t3-connect.md` and are not
repeated here.

None of this is a stable public API. The contracts change between nightlies.
Check `GET /.well-known/t3/environment` capabilities before relying on an
optional method.

## How to read the payloads

- Every method here is an Effect RPC tag, sent as `{"_tag":"Request","id":n,"tag":"...","payload":{...},"headers":[]}`.
  The sections below show only `payload` and the success value.
- A plain call answers with one `Exit`. The success value is what's shown as
  "Result". A method with no success schema (`terminal.write`, `vcs.init`,
  `shell.openInEditor`) returns an `Exit` success with no useful value.
- A stream sends `Chunk` frames. Each element of `values` is one "event" below.
  **Ack every chunk.** The terminal streams let 8 chunks or 64 KB through
  unacked, everything else none.
- Values use the schema's encoded form. Dates are ISO strings. Optional keys
  are absent, not `null`, unless the schema says `NullOr`.
- Errors arrive as `Exit` `Failure` with a `Fail` cause whose `error._tag`
  names the error class. Every method can also fail with
  `EnvironmentAuthorizationError` (see "Required scopes" at the end).
- Contracts live in `packages/contracts/src`. Method tags are `WS_METHODS` in
  `rpc.ts:387`. Handlers are in `apps/server/src/ws.ts`.

## Things that will bite you

- `projects.list`, `projects.add` and `projects.remove` are in `WS_METHODS`
  but have no RPC definition, handler or scope. Calling them fails. Projects
  come from `orchestration.subscribeShell`, are added with `projects.mutate`
  `project.create` and removed with `project.delete`.
- `@` file mentions send **no** `context` records. The mention is a Markdown
  link in `text`. Nothing else rides along. See "The composer @ file mention".
- Every file and git method takes an absolute `cwd` on the **host**. For a
  thread, that's `thread.worktreePath ?? project.workspaceRoot`
  (`apps/web/src/components/ChatView.tsx:1571`).
- `projects.readFile` accepts an absolute or `~/` path and reads it with no
  root check. `filesystem:read` is effectively read access to the whole host
  account. Hand that scope out accordingly.
- `projects.writeFile` is the strict one: `cwd` must be a registered project,
  a managed worktree or a thread's worktree, and `.git` is off limits.
- `ignoreWhitespace` defaults **on** for `orchestration.getTurnDiff` and
  `getFullThreadDiff`, and **off** for `review.getDiffPreview`.
- `terminal.open` on a live terminal with a different `cwd`, `worktreePath` or
  `env` kills the shell and clears its history.
- `vcs.createWorktree` requires the key `path` and it must be `null`.

## 1. Projects

### Where projects come from

The project list is part of the shell stream (`orchestration.subscribeShell`,
`snapshot.projects` plus `project.updated` / `project.removed`), documented in
`t3-connect.md`. A shell project is `OrchestrationProjectShell`
(`packages/contracts/src/orchestrationProject.ts`):

```json
{
  "id": "5c0f...",
  "title": "app",
  "workspaceRoot": "/home/me/code/app",
  "repositoryIdentity": null,
  "defaultModelSelection": null,
  "defaultThreadEnvMode": null,
  "autoPull": false,
  "faviconPath": null,
  "projectIcon": null,
  "scripts": [],
  "createdAt": "2026-10-20T10:00:00.000Z",
  "updatedAt": "2026-10-20T10:00:00.000Z"
}
```

`projects.mutate` returns the fuller `Project`, which adds
`deletedAt: string | null` (`packages/contracts/src/project.ts:151`).

A project script is `{id, name, command, icon, runOnWorktreeCreate, runOnSettle?, async?, previewUrl?, autoOpenPreview?}`.
`icon` is one of `play | test | lint | configure | build | debug`.

### projects.mutate

Scope: `orchestration:operate`, plus `settings:write` if the mutation carries a
`scripts` key (scripts run on the host,
`packages/contracts/src/settings.ts:1597`). Result: `Project`. Error:
`ProjectMutationError {commandId, message}`.

Payload is a tagged union on `type`. `commandId` and `projectId` are yours to
generate (UUIDs work).

```json
{"type":"project.create","commandId":"c1","projectId":"p1","title":"app","workspaceRoot":"/home/me/code/app","createWorkspaceRootIfMissing":true}
```

Optional on create: `defaultModelSelection` (`ModelSelection | null`),
`scripts`.

```json
{"type":"project.update","commandId":"c2","projectId":"p1","title":"app 2"}
```

Every key besides `type`, `commandId` and `projectId` is optional on update:
`title`, `workspaceRoot`, `defaultModelSelection`, `autoPull`, `projectIcon`
(`null` clears), `faviconPath` (`null` clears), `defaultThreadEnvMode`
(`ThreadEnvMode | null`), `scripts` (replaces the whole list).

```json
{"type":"project.delete","commandId":"c3","projectId":"p1","force":false}
```

Deleting a project that still has threads (archived ones count) fails with a `ProjectNotEmptyError`
message unless `force` is true (`apps/server/src/ws.ts:2787`). A delete also
drops any clone tracked for that project.

Create fails with a conflict error if the workspace root already belongs to
another project (`apps/server/src/project/ProjectService.ts:62`).

### projects.ensureScratch

Payload `{}`. Result `{"projectId":"..."}`. Finds or creates the environment's
Scratch project, a managed folder under the server's data directory, for
threads that belong to no repository. Scope `orchestration:operate`. The
project then shows up on the shell stream; the official client waits up to 10
seconds for it (`packages/client-runtime/src/state/projectCommands.ts:179`).

### projects.createNew

Payload `{"name":"my-app"}` (1 to 200 characters). Result:

```json
{"projectId":"...","workspaceRoot":"/home/me/.t3/.../my-app","commitError":"..."}
```

The server claims a folder in its managed projects directory, scaffolds a git
repository, and creates the project. `commitError` is present when the project
and files exist but the first commit failed
(`apps/server/src/project/ManagedProjectFolders.ts:421`). Scope
`orchestration:operate`.

### projects.searchEntries

Fuzzy path search over a workspace index. Scope `filesystem:read`.

```json
{"cwd":"/home/me/code/app","query":"index","limit":80,"kind":"file","imageOnly":false}
```

- `cwd` required. `query` is trimmed, at most 256 characters. A leading run of
  `@`, `.` and `/` is stripped server side.
- `limit` is 1 to 200, required.
- `kind` is `"file"` or `"directory"`. Omit it for a mix of both.
- `imageOnly: true` restricts to files with a favicon-style image extension.

Result:

```json
{"entries":[{"path":"src/index.ts","kind":"file"},{"path":"src","kind":"directory"}],"truncated":false}
```

`path` is relative to `cwd`, forward slashes. `truncated` is also true while
the index is still being built. The index holds up to 25,000 entries and
waits up to 15 seconds to finish its first scan
(`apps/server/src/workspace/WorkspaceSearchIndex.ts:34`). Errors:
`ProjectSearchEntriesError` with `failure` in `workspace_root_not_found`,
`workspace_root_not_directory`, `search_index_scan_timed_out` and friends.

### projects.listEntries

Scope `filesystem:read`.

```json
{"cwd":"/home/me/code/app","directoryPath":"src"}
```

With `directoryPath` (workspace-relative, `""` for the root): one directory
level, files and folders, never `.git`, symlink targets must stay in the root.
Each entry may carry `ignored: true` from `git check-ignore`. `truncated` is
false.

```json
{"entries":[{"path":"src/index.ts","kind":"file"},{"path":"src/dist","kind":"directory","ignored":true}],"truncated":false}
```

Without `directoryPath` it returns the whole indexed tree (the same index as
search, capped at 25,000 entries, `truncated` says if it was cut). Error:
`ProjectListEntriesError`.

### projects.listIgnoredFiles

Scope `filesystem:read`. Payload `{"cwd":"..."}`. Lists untracked, git-ignored
files (think `.env`), leaving out folders that hold nothing but ignored files
such as `node_modules`.

```json
{"files":[{"path":".env","byteLength":212}],"truncated":false}
```

At most 200 files; `truncated` says more existed. Not a git repo, or git
fails: `{"files":[],"truncated":false}`.

### projects.readFile

Scope `filesystem:read`.

```json
{"cwd":"/home/me/code/app","relativePath":"src/index.ts","allowMissing":false}
```

`relativePath` (max 512 characters) is workspace-relative, or absolute, or
`~/...`. Workspace-relative paths cannot escape the root, symlinks included.
Result:

```json
{"relativePath":"src/index.ts","contents":"...","byteLength":1234,"truncated":false,"exists":true}
```

- Reads at most 1 MiB. `byteLength` is the real file size, `truncated` is
  true past 1 MiB.
- `allowMissing: true` turns a missing file into
  `{"contents":"","byteLength":0,"truncated":false,"exists":false}`. Older
  servers omit `exists`.
- Files with a NUL byte in the first 1 MiB fail with `failure: "binary_file"`.
  Other failures: `workspace_path_outside_root`, `resolved_path_outside_root`,
  `path_not_file`, `operation_failed`. Error class `ProjectReadFileError`.

Source: `apps/server/src/workspace/WorkspaceFileSystem.ts:337`.

### projects.writeFile

Scope `filesystem:write`.

```json
{"cwd":"/home/me/code/app","relativePath":"notes/todo.md","contents":"..."}
```

`relativePath` is workspace-relative only (max 512). Creates parent
directories, truncates and writes UTF-8, refuses `.git` and anything that
resolves outside the real root, and refuses a `cwd` that isn't a known
workspace. Result `{"relativePath":"notes/todo.md"}`. Error:
`ProjectWriteFileError` with the same `failure` values as read (minus
`binary_file`).

### projects.searchContents

Text search over file contents. Scope `filesystem:read`.

```json
{"cwd":"/home/me/code/app","query":"TODO","limit":100,"caseSensitive":false,"wholeWord":false,"useRegex":false}
```

`query` keeps its whitespace, 1 to 256 characters. `limit` 1 to 500. All
three booleans are required. When `caseSensitive` is false and `useRegex` is
false it is smart-case.

```json
{
  "matches": [
    {"path":"src/index.ts","lineNumber":12,"lineContent":"// TODO: fix","matchRanges":[{"start":3,"end":7}]}
  ],
  "truncated": false,
  "regexFallbackError": "..."
}
```

`matchRanges` are offsets into `lineContent`. `regexFallbackError` appears
when your regex didn't compile and the server searched it as plain text. Up to
100 matches per file. Error `ProjectSearchContentsError`.

### The composer @ file mention

The official clients (web, and mobile through the same shared code) do this:

1. **Detect the trigger.** In the composer text, the token before the caret
   starts with `@`. `detectComposerTrigger`
   (`packages/shared/src/composerTrigger.ts:71`) returns
   `{kind:"path", query: token.slice(1), rangeStart, rangeEnd}` where the
   range covers `@` plus the query. `#` starts a pull request search, `$` a
   skill, `/` a slash command.
2. **Search.** After a 120 ms debounce, `projects.searchEntries` with the
   thread's cwd, the query, and a fixed limit of 80
   (`apps/web/src/state/queries.ts:40-41`). Mixed files and directories,
   no `kind`, no `imageOnly`. The web app only searches when the query is
   non-empty. It also lists matching other threads from the shell stream above
   the paths, which is a different feature (a `thread` context record).

   ```json
   {"cwd":"/home/me/code/app","query":"inde","limit":80}
   ```
   ```json
   {"entries":[{"path":"src/index.ts","kind":"file"},{"path":"src/indexer","kind":"directory"}],"truncated":false}
   ```

3. **Insert.** Picking `src/index.ts` replaces `rangeStart..rangeEnd` with a
   Markdown link plus one trailing space
   (`apps/web/src/components/chat/ChatComposer.tsx:1920`):

   ```
   [index.ts](src/index.ts)␠
   ```

   Built by `serializeComposerFileLink`
   (`packages/shared/src/composerTrigger.ts:34`):
   - Label is the basename, with `\`, `[` and `]` backslash-escaped.
   - Destination is `encodeURI(path)` with `(` `)` `#` `?` `\` additionally
     percent-encoded (`%28 %29 %23 %3F %5C`). A space becomes `%20`.
   - Directories are inserted the same way, with the directory path.
   - `docs/My Notes (v2).md` becomes `[My Notes (v2).md](docs/My%20Notes%20%28v2%29.md) `.

4. **Send.** The text goes into `message.dispatch` unchanged. There is no
   `context` and no `records`, because the official clients' `buildMessageContext`
   only emits terminal, review-comment, thread, preview-annotation and
   image/file attachment records (`apps/web/src/lib/composerContextRecords.ts:312`).

   ```json
   {
     "type": "message.dispatch",
     "commandId": "0f6e...",
     "threadId": "t1",
     "messageId": "0f6e...",
     "text": "Look at [index.ts](src/index.ts) and fix the typo",
     "attachments": [],
     "createdBy": "user",
     "creationSource": "web",
     "dispatchMode": {"type": "start_immediately"}
   }
   ```

   Sent as `orchestration.dispatchCommand` (scope `orchestration:operate`),
   result `{"sequence": 1234}`. `context` is omitted entirely when there are no
   records.

How the app turns the text back into a chip: `collectComposerInlineTokens`
(`packages/shared/src/composerInlineTokens.ts:41`) recognises
`[label](dest)` when the decoded destination has no URI scheme and the label
equals the destination's basename, and the older `@path` / `@"path with spaces"`
form when followed by whitespace. The rich editor keeps this as an atom node
`composer-mention` with attrs `{path, source}`, where `source` is the exact
source substring (the link text above). That node never goes on the wire. Only
the text does.

The agent sees the link text as ordinary prose. The server does not read the
file for it. Don't expect the file contents to be attached.

**Typed mention records (optional, unused by the official clients).** The
contract does define one. `MentionContextRecord`
(`packages/contracts/src/composerContext.ts:191`) is
`{version:1, kind:"mention", contextId, label, path}`, and the web app renders
it as a chip (`apps/web/src/components/chat/UserMessageContextChips.tsx:299`).
To use it, put a context reference in the text and the record in `context`:

```json
{
  "type": "message.dispatch",
  "commandId": "...", "threadId": "t1", "messageId": "...",
  "text": "Look at [index.ts](t3-context://v1/mention/mention_a1b2) please",
  "context": {
    "version": 1,
    "records": [
      {"version":1,"kind":"mention","contextId":"mention_a1b2","label":"index.ts","path":"src/index.ts"}
    ]
  },
  "attachments": [], "createdBy":"user", "creationSource":"web",
  "dispatchMode": {"type":"start_immediately"}
}
```

Rules: `contextId` matches `/^[a-z0-9_-]+$/i`, at most 128 characters, unique
within `records`; the reference is `[label](t3-context://v1/<kind>/<contextId>)`
(`packages/shared/src/composerContextReferences.ts:52`); `label` must not
contain `[ ] \` or newlines; at most 200 records. The server rewrites the
reference for the provider into `[Mention: index.ts; ref=mention_a1b2]` and
appends a `<t3_context version="1">` envelope holding `path: src/index.ts`
(`composerContextReferences.ts:150`, `:276`). Undecodable records are dropped,
not fatal. Unlike the plain link, I could not find anything in the official
UI that produces this, so treat it as lightly exercised.

### projectClone.start, cancel, retry, subscribeProjectClones

Cloning is a tracked server job so a client can show progress and survive a
reconnect.

**projectClone.start**, scope `source-control:write`:

```json
{
  "projectId": "p2",
  "title": "t3code",
  "createdAt": "2026-10-24T12:00:00.000Z",
  "provider": "github",
  "repository": "pingdotgg/t3code",
  "destinationPath": "/home/me/code/t3code",
  "protocol": "auto"
}
```

Give either `provider` + `repository` (looked up through the provider, then
the SSH or HTTPS URL is chosen by `protocol`: `auto | ssh | https`) or a plain
`remoteUrl`. `projectId` and `createdAt` are yours. Result, returned as soon as
the project exists and git has been launched:

```json
{"projectId":"p2","cwd":"/home/me/code/t3code","remoteUrl":"https://github.com/pingdotgg/t3code.git","repository":{"provider":"github","nameWithOwner":"pingdotgg/t3code","url":"...","sshUrl":"..."}}
```

`remoteUrl` has credentials redacted. A second clone to the same destination
or with the same `projectId` fails with "A clone into this destination is
already in progress." The server creates the project (with
`createWorkspaceRootIfMissing`) before cloning, so it appears on the shell
stream while still empty, and re-resolves its repository identity when the
clone finishes (`apps/server/src/ws.ts:2662`,
`apps/server/src/project/ProjectCloneTracker.ts`). Errors:
`SourceControlRepositoryError`, `OrchestrationDispatchCommandError`.

**projectClone.cancel** and **projectClone.retry**, scope
`source-control:write`: payload `{"projectId":"p2"}`, result `{"applied":true}`.
`applied` is false if nothing matched. Cancel removes the half-cloned
directory unless the clone had finished.

**subscribeProjectClones**, scope `orchestration:read`: payload `{}`. Every
event is the **complete** array of tracked clones, not a delta. A slow reader
only gets the newest array (sliding buffer of one). Finished clones stay in
the array for 30 seconds, then vanish.

```json
[
  {
    "projectId": "p2",
    "remoteUrl": "https://github.com/pingdotgg/t3code.git",
    "destinationPath": "/home/me/code/t3code",
    "repository": null,
    "phase": "running",
    "stage": "receiving",
    "percent": 42,
    "detail": "3.1 MiB | 6.2 MiB/s",
    "stalled": false,
    "error": null,
    "startedAt": "2026-10-24T12:00:00.000Z",
    "endedAt": null,
    "sequence": 17
  }
]
```

`phase`: `running | done | failed | cancelled`. `stage`: `connecting | counting
| receiving | resolving | checkout`. `percent` is an int 0 to 100 or null.
`stalled` is set after 20 seconds of git silence, and `detail` is stale then.
`error` is at most 1000 characters. `sequence` increases across all clones.

## 2. filesystem

Both scope `filesystem:read`. These work on arbitrary host paths, not just
projects.

**filesystem.browse** lists **directories only** whose name starts with a
prefix, for a "pick a folder" UI. Source
`apps/server/src/workspace/WorkspaceEntries.ts:201`.

```json
{"partialPath":"~/code/ap","cwd":"/home/me/code/app"}
```
```json
{"parentPath":"/home/me/code","entries":[{"name":"app","fullPath":"/home/me/code/app"},{"name":"apple","fullPath":"/home/me/code/apple"}]}
```

- `partialPath` (max 512): absolute, `~`, `~/...`, or explicitly relative
  (`./x`, `../x`). Explicit-relative paths need `cwd`, else
  `FilesystemBrowseError` with `failure: "current_project_required"`.
- A trailing `/` (or just `~`) means "list inside this directory". Without
  one, the last segment is the prefix and the parent is listed.
- Case-insensitive prefix match, sorted by name. Hidden directories show only
  when the prefix starts with `.` or the path ends in `/`.
- Permission errors give an empty list. Other read errors give
  `read_directory_failed`. A Windows path on a non-Windows host gives
  `windows_path_unsupported`.

**filesystem.getMetadata** stats up to 64 paths at once.

```json
{"paths":["/home/me/code/app/README.md","~/missing","/tmp"]}
```
```json
{"entries":[{"kind":"file","byteLength":812,"mimeType":"text/markdown"},null,{"kind":"directory"}]}
```

Entries follow request order. `null` means missing, unreadable or unsupported.
`kind` is `file | directory | other`. Paths must be absolute or `~/...`; the
contract says relative paths must be anchored by the client. `mimeType` is only
sniffed from the first 512 bytes for files with no extension, so don't expect
it otherwise.

## 3. VCS, git, worktrees, source control

"VCS" methods are the git-backed ones. The driver kind is `git | jj | unknown`
(`packages/contracts/src/vcs.ts`), but today only git implements most of
this. Inputs for all of these live in `packages/contracts/src/git.ts`.

### subscribeVcsStatus

Scope `orchestration:read`. Payload `{"cwd":"...","includeRemote":true}`.
`includeRemote: false` gets cached remote status and does not start the
background remote refresh loop for that cwd (a passive observer).

Events (`_tag` discriminated):

```json
{"_tag":"snapshot","local":{ ...VcsStatusLocalResult },"remote":{ ...VcsStatusRemoteResult } | null}
{"_tag":"localUpdated","local":{ ... }}
{"_tag":"remoteUpdated","remote":{ ... } | null}
```

`local` (`VcsStatusLocalResult`):

```json
{
  "isRepo": true,
  "sourceControlProvider": {"kind":"github","name":"GitHub","baseUrl":"https://github.com"},
  "hasPrimaryRemote": true,
  "isDefaultRef": false,
  "refName": "feature/x",
  "hasWorkingTreeChanges": true,
  "workingTree": {
    "rootPath": "/home/me/code/app",
    "files": [{"path":"src/a.ts","insertions":3,"deletions":1}],
    "insertions": 3,
    "deletions": 1
  },
  "branchChanges": {"baseRef":"main","insertions":10,"deletions":2}
}
```

`sourceControlProvider`, `workingTree.rootPath` and `branchChanges` are
optional. `refName` is null on a detached HEAD. `files[].path` is relative to
`rootPath`.

`remote` (`VcsStatusRemoteResult`):

```json
{
  "hasUpstream": true,
  "aheadCount": 1,
  "behindCount": 0,
  "aheadOfDefaultCount": 3,
  "pr": {"number":42,"title":"Fix","url":"https://github.com/o/r/pull/42","baseRef":"main","headRef":"feature/x","state":"open","isDraft":false,"updatedAt":"2026-10-24T11:00:00Z"}
}
```

`pr` is null with no open or matching change request. The first event is
always a `snapshot` (`remote` may be null while the first remote read is
pending). Merge later events into a `VcsStatusResult` with the rule in
`applyGitStatusStreamEvent` (`packages/shared/src/git.ts:339`): a snapshot
replaces both halves, `localUpdated` keeps the current remote half,
`remoteUpdated` keeps the current local half. Events are only sent when a half
actually changed (the server compares fingerprints). Failed remote refreshes
back off from 30 seconds up to 15 minutes. Source
`apps/server/src/vcs/VcsStatusBroadcaster.ts:722`.

### vcs.refreshStatus

Scope `orchestration:read`. Payload `{"cwd":"..."}`. Forces a refresh and
returns the merged `VcsStatusResult` (the local keys and remote keys in one
object). It also publishes the change to subscribers.

### vcs.listRefs

Scope `orchestration:read`.

```json
{"cwd":"...","query":"feat","cursor":0,"limit":50,"refKind":"all","includeMatchingRemoteRefs":false,"refresh":false}
```

All keys but `cwd` optional. `query` max 256, `limit` 1 to 200, `refKind`
`all | local | remote`, `refresh` forces a fetch-backed refresh.

```json
{
  "isRepo": true,
  "hasPrimaryRemote": true,
  "isAtRepositoryRoot": true,
  "refs": [
    {"name":"main","current":false,"isDefault":true,"worktreePath":null},
    {"name":"origin/feature/x","isRemote":true,"remoteName":"origin","upstreamRef":"refs/remotes/origin/feature/x","current":false,"isDefault":false,"worktreePath":null},
    {"name":"feature/x","current":true,"isDefault":false,"worktreePath":"/home/me/.t3/worktrees/app/feature-x"}
  ],
  "nextCursor": 50,
  "totalCount": 120
}
```

`nextCursor` is null on the last page. `worktreePath` says where a ref is
checked out.

### vcs.pull, vcs.createRef, vcs.switchRef, vcs.init, vcs.removeWorktree

All scope `source-control:write`. Error is `GitCommandError {operation, command, cwd, reason?, detail, ...}`.
`reason` is one of `authentication_failed`, `branch_already_exists`,
`branch_checked_out_in_worktree`, `host_key_unverified`, `not_a_repository`,
`path_already_exists`, `push_rejected_by_remote`,
`push_rejected_non_fast_forward`, `ref_not_found`, `remote_unreachable`,
`tag_would_be_clobbered`; unknown newer values decode as absent. Each success
refreshes the status stream for that cwd.

| Tag | Payload | Result |
| --- | --- | --- |
| `vcs.pull` | `{"cwd"}` | `{"status":"pulled"\|"skipped_up_to_date","refName","upstreamRef":string\|null}`. Fails on detached HEAD or with no upstream. |
| `vcs.createRef` | `{"cwd","refName","switchRef"?:bool}` | `{"refName"}` |
| `vcs.switchRef` | `{"cwd","refName"}` | `{"refName":string\|null}` |
| `vcs.init` | `{"cwd","kind"?:"git"\|"jj"\|"unknown"}` | none |
| `vcs.removeWorktree` | `{"cwd","path","force"?:bool}` | none |

### Worktrees

**vcs.createWorktree**, scope `source-control:write`:

```json
{"cwd":"/home/me/code/app","refName":"main","newRefName":"feature/y","baseRefName":"main","path":null}
```

This runs `git worktree add [-b newRefName] -- <path> refName`. With
`newRefName` the branch is created from `refName`. `baseRefName` records the
branch's merge base for later "changes vs base" views. `path` **must be
present and null**: the server picks its managed worktrees folder (the
contract keeps the key for old servers,
`packages/contracts/src/rpc.ts:1409`). Submodules are initialised. Result:

```json
{"worktree":{"path":"/home/me/.t3/worktrees/app/feature-y","refName":"feature/y"}}
```

Most clients never call this directly. Starting a thread with
`workspaceStrategy: {type:"worktree", baseRef, branch?, startFromOrigin?}` in
`orchestration.launchThread` makes the server create the worktree, run the
project's setup script and start the agent, and reports progress on the
worktree setup stream below. The other strategies are `{type:"root", branch?}`
and `{type:"existing_worktree", worktreePath, branch?}`
(`packages/contracts/src/orchestrationV2.ts:571`).

**subscribeWorktreeSetup**, scope `orchestration:read`. Payload
`{"threadId":"t1"}`. Events are `WorktreeSetupSnapshot | null`
(`packages/contracts/src/worktreeSetup.ts`). The first event is the current
state (or `null` when nothing is tracked). Later events are the whole
snapshot again; a slow reader only gets the newest. A snapshot is kept for 30
seconds after it finishes, after which subscribers see `null`
(`apps/server/src/project/WorktreeSetupTracker.ts:75`). Ignore a snapshot whose
`sequence` is not larger than the one you hold, except after a `null`.

```json
{
  "threadId": "t1",
  "phase": "running",
  "startedAt": "2026-10-24T12:00:00.000Z",
  "endedAt": null,
  "branch": "feature/y",
  "baseRef": "main",
  "worktreePath": "/home/me/.t3/worktrees/app/feature-y",
  "setupScript": {"name":"Install","command":"bun install","terminalId":"setup-..."},
  "stages": [
    {"id":"fetch","status":"done","startedAt":"...","endedAt":"...","percent":null,"detail":null,"tail":[]},
    {"id":"checkout","status":"done","startedAt":"...","endedAt":"...","percent":100,"detail":null,"tail":[]},
    {"id":"setup-script","status":"running","startedAt":"...","endedAt":null,"percent":null,"detail":null,"tail":["added 312 packages"]},
    {"id":"agent","status":"pending","startedAt":null,"endedAt":null,"percent":null,"detail":null,"tail":[]}
  ],
  "error": null,
  "sequence": 9
}
```

`phase`: `running | done | failed | cancelled`. Stage `id`: `fetch | checkout |
submodules | setup-script | agent`, in that order. A fresh worktree thread
tracks `fetch, checkout, setup-script, agent`; reusing an existing worktree
tracks `setup-script, agent`. Stage `status`: `pending | running | done |
skipped | warning | failed`. `tail` holds at most the last 4 lines (400
characters each). `setupScript.terminalId` is a terminal under the same
`threadId`: attach to it with `terminal.observe` to see the script's output.
The finished setup is also recorded as a thread activity of kind
`"worktree-setup"` whose payload is this snapshot
(`worktreeSetup.ts:64`).

**worktreeSetup.cancel**, scope `orchestration:operate`: payload
`{"threadId"}`, result `{"cancelled":true|false}`. Once the agent is about to
start, cancelling is refused and returns false.

### git.runStackedAction

Streaming. Scope `source-control:write`. When `threadId` is set and `action` is
`create_pr` or `commit_push_pr`, `orchestration:read` is required too
(`packages/contracts/src/clientRpcPermissions.ts:70`).

```json
{
  "actionId": "a1",
  "cwd": "/home/me/code/app",
  "action": "commit_push_pr",
  "commitMessage": "Fix typo",
  "featureBranch": false,
  "filePaths": ["src/index.ts"],
  "threadId": "t1",
  "projectId": "p1"
}
```

- `actionId` is yours, echoed in every event.
- `action`: `commit`, `push`, `create_pr`, `commit_push`, `commit_push_pr`.
  - `commit` stages and commits.
  - `push` pushes the current branch (sets upstream when needed).
  - `create_pr` opens a change request, pushing first when the branch has no
    upstream or is ahead.
  - `commit_push` and `commit_push_pr` chain them.
- `commitMessage` (max 10,000): subject, blank line, body. Omit it and the
  server generates one with the configured text-generation model (you will
  see a `phase_started` labelled "Generating commit message...").
- `featureBranch: true` creates a new branch first and commits on it. Only
  valid for actions that commit, else a `GitManagerError`.
- `filePaths` (non-empty if present) limits the commit to those files.
- `threadId` makes the server link a created PR to that thread
  (`apps/server/src/git/linkCreatedPullRequest.ts`).
- Detached HEAD fails for push and PR actions unless `featureBranch` is true.

Events (`GitActionProgressEvent`), each with `actionId`, `cwd`, `action`:

```json
{"kind":"action_started","phases":["commit","push","pr"]}
{"kind":"phase_started","phase":"commit","label":"Committing..."}
{"kind":"hook_started","hookName":"pre-commit"}
{"kind":"hook_output","hookName":"pre-commit","stream":"stderr","text":"lint ok"}
{"kind":"hook_finished","hookName":"pre-commit","exitCode":0,"durationMs":812}
{"kind":"action_finished","result":{
  "action":"commit_push_pr",
  "branch":{"status":"skipped_not_requested"},
  "commit":{"status":"created","commitSha":"abc123","subject":"Fix typo"},
  "push":{"status":"pushed","branch":"feature/x","upstreamBranch":"origin/feature/x","setUpstream":true},
  "pr":{"status":"created","url":"https://github.com/o/r/pull/43","number":43,"baseBranch":"main","headBranch":"feature/x","title":"Fix typo"},
  "toast":{"title":"Created PR #43","cta":{"kind":"open_pr","label":"View PR","url":"https://github.com/o/r/pull/43"}}
}}
{"kind":"action_failed","phase":"push","message":"..."}
```

`phase` values: `branch | commit | push | pr`. Step statuses: branch
`created | skipped_not_requested`; commit `created | skipped_no_changes |
skipped_not_requested`; push `pushed | skipped_not_requested |
skipped_up_to_date`; pr `created | opened_existing | skipped_not_requested`.
`toast.cta.kind` is `none | open_pr | run_action` (the last carries
`{action:{kind: GitStackedAction}}`, a suggested follow-up).

The stream ends after `action_finished`. On failure you get `action_failed`
as a chunk, then the stream fails with a `GitManagerServiceError`
(`GitManagerError`, `GitCommandError`, `SourceControlProviderError`,
`GitPullRequestMaterializationError` or `TextGenerationError`). Source
`apps/server/src/git/GitManager.ts:3018`.

### git.resolvePullRequest

Scope `orchestration:read`.

```json
{"cwd":"/home/me/code/app","reference":"123"}
```

`reference` is a PR number, `#123` or a PR URL. Result:

```json
{"pullRequest":{"number":123,"title":"Fix","url":"https://github.com/o/r/pull/123","baseBranch":"main","headBranch":"fix","state":"open"}}
```

`state`: `open | closed | merged`.

### git.preparePullRequestThread

Scope `source-control:write`. With `mode: "worktree"` and a `threadId`, also
`orchestration:operate`.

```json
{"cwd":"/home/me/code/app","reference":"#123","mode":"worktree","threadId":"t1"}
```

- `mode: "local"` checks the PR branch out in `cwd`. It refuses when there are
  tracked changes.
- `mode: "worktree"` creates (or reuses) a worktree for the PR branch.
- `threadId`, when given, runs the project's worktree setup script for that
  thread in the new worktree. A failing script is logged and ignored.

```json
{
  "pullRequest": {"number":123,"title":"Fix","url":"...","baseBranch":"main","headBranch":"fix","state":"open"},
  "branch": "fix",
  "worktreePath": "/home/me/.t3/worktrees/app/fix",
  "isOnPullRequestHead": true
}
```

`worktreePath` is null in local mode. `isOnPullRequestHead` is false when local
commits or edits stopped a reused checkout from reaching the PR head. Older
servers omit it (decoded as true). Source
`apps/server/src/git/GitManager.ts:2709`.

### sourceControl.*

Provider-level repository operations (GitHub, GitLab and so on).
Lookups and publishing go through the provider integration on the
host (usually its CLI, logged in). `server.discoverSourceControl` (scope `orchestration:read`, no payload)
reports what is available and who is authenticated
(`SourceControlDiscoveryResult`).

**sourceControl.lookupRepository**, scope `orchestration:read`:

```json
{"provider":"github","repository":"pingdotgg/t3code","cwd":"/home/me/code"}
```

Result `{"provider":"github","nameWithOwner":"pingdotgg/t3code","url":"https://github.com/pingdotgg/t3code","sshUrl":"git@github.com:pingdotgg/t3code.git"}`.
`provider` is a lowercase slug, unknown values are accepted.

**sourceControl.cloneRepository**, scope `source-control:write`: the blocking
form of `projectClone.start`. Payload
`{"provider"?,"repository"?,"remoteUrl"?,"destinationPath","protocol"?}`.
Result `{"cwd","remoteUrl","repository":{...}|null}`. It has a 120 second
budget and creates no project and gives no progress. Prefer `projectClone.start`.

**sourceControl.publishRepository**, scope `source-control:write`:

```json
{"cwd":"/home/me/code/app","provider":"github","repository":"me/app","visibility":"private","remoteName":"origin","protocol":"ssh"}
```

Creates the remote repository, adds the remote and pushes. Result:

```json
{"repository":{"provider":"github","nameWithOwner":"me/app","url":"...","sshUrl":"..."},"remoteName":"origin","remoteUrl":"git@github.com:me/app.git","branch":"main","upstreamBranch":"origin/main","status":"pushed"}
```

`status`: `pushed | remote_added`. Error: `SourceControlRepositoryError
{provider, operation, detail}`.

The `pullRequests.*` family (list, detail, comments, review) is a separate,
large API and is not covered here.

## 4. Diffs

Two unrelated systems: per-thread diffs between agent turns (checkpoints), and
the live git diff of a working directory (review).

### orchestration.getTurnDiff

Scope `orchestration:read`. Unified git diff of the files an agent changed
between two checkpoints.

```json
{"threadId":"t1","fromTurnCount":2,"toTurnCount":3,"ignoreWhitespace":true}
```

```json
{
  "threadId": "t1",
  "fromTurnCount": 2,
  "toTurnCount": 3,
  "diff": "diff --git a/src/index.ts b/src/index.ts\nindex 83db48f..bf2a0c1 100644\n--- a/src/index.ts\n+++ b/src/index.ts\n@@ -1,3 +1,3 @@\n-const a = 1;\n+const a = 2;\n"
}
```

Semantics, from `apps/server/src/checkpointing/CheckpointDiffQuery.ts:93`:

- Both counts are non-negative ints and `fromTurnCount <= toTurnCount`, or the
  request fails to decode. Equal counts return `diff: ""` without touching
  git.
- A "turn count" is a checkpoint's `appRunOrdinal`: the 1-based number of a
  completed run in the thread. `0` means the thread's starting state (the
  root checkpoint before the first run).
- Where do you get them? From `orchestration.subscribeThread` or
  `orchestration.getThreadProjection`: `projection.checkpoints[]` each have
  `appRunOrdinal` (1-based, or null), `ordinalWithinScope`, `status`
  (`ready | missing | error | stale`), `runId`, `ref`, and `files:
  [{path, kind, additions, deletions}]`
  (`packages/contracts/src/orchestrationV2.ts:1244`). Only `ready` checkpoints
  of `completed` runs count. The web app shows turn N as
  `from = N - 1`, `to = N` (`apps/web/src/components/DiffPanel.tsx:299`).
- `toTurnCount` larger than the newest ready, completed checkpoint fails. So
  does a missing checkpoint ref on either side.
- `ignoreWhitespace` defaults to **true** when omitted.
- Output is capped at 10,000,000 bytes. When cut, it ends with
  `"\n\n[truncated]"`. Bulk `git`-imported paths are dropped from it.
- The diff is run in the scope's own cwd, so a thread in a worktree diffs
  that worktree.
- Error: `OrchestrationGetTurnDiffError {message: "Failed to load turn diff"}`
  with the cause attached. The message is generic.

### orchestration.getFullThreadDiff

Scope `orchestration:read`. Same thing from the thread's start to a turn.

```json
{"threadId":"t1","toTurnCount":3,"ignoreWhitespace":true}
```

Result is identical in shape, with `fromTurnCount` always `0`:

```json
{"threadId":"t1","fromTurnCount":0,"toTurnCount":3,"diff":"diff --git ..."}
```

`toTurnCount: 0` returns an empty diff. Error:
`OrchestrationGetFullThreadDiffError`. It is implemented as `getTurnDiff` from
0.

### review.getDiffPreview

Scope `filesystem:read`. The current git diff of a directory, as the diff
panel shows when no turn is selected. `cwd` must be inside the server's
configured workspace root or a known workspace (registered project, managed
worktree or thread worktree), else a `VcsRepositoryDetectionError`.

```json
{"cwd":"/home/me/code/app","baseRef":"main","ignoreWhitespace":false}
```

`baseRef` and `ignoreWhitespace` are optional. Result:

```json
{
  "cwd": "/home/me/code/app",
  "generatedAt": "2026-10-24T12:00:00.000Z",
  "sources": [
    {
      "id": "working-tree", "kind": "working-tree", "title": "Uncommitted",
      "baseRef": "HEAD", "headRef": null,
      "diff": "diff --git ...", "diffHash": "9f2c...",
      "truncated": false,
      "files": [{"path":"src/a.ts","previousPath":null,"additions":3,"deletions":1}]
    },
    {
      "id": "branch-range", "kind": "branch-range", "title": "Changes vs main",
      "baseRef": "main", "headRef": "feature/x",
      "diff": "diff --git ...", "diffHash": "41ab...",
      "truncated": false,
      "files": [ ... ]
    }
  ]
}
```

- Two sources, always in that order when the cwd is a repo: working tree vs
  `HEAD` (includes untracked files), and branch range vs the merge base with
  `baseRef` (or the detected default branch).
- Not a repo: `sources: []`.
- Each `diff` is capped at 120,000 bytes and ends with a truncation marker
  when cut, with `truncated: true`. `files` is omitted when the stats could
  not be read completely.
- `diffHash` is a SHA-256 of the diff plus stats. Use it to skip re-renders.
- To get the full patch of one file, repeat the call with
  `"file":{"path":"src/a.ts","previousPath":null,"sourceKind":"working-tree"}`.
  The per-file cap is 1 MiB.
- Source `apps/server/src/vcs/GitVcsDriverCore.ts:3034`.

### review.getDiffFileContents

Scope `filesystem:read`. Both sides of a file, for expanding the unchanged
lines around a hunk.

```json
{
  "cwd": "/home/me/code/app",
  "sourceKind": "branch-range",
  "changeType": "change",
  "baseRef": "main",
  "headRef": "feature/x",
  "oldPath": "src/a.ts",
  "newPath": "src/a.ts"
}
```
```json
{"oldContents":"...","newContents":"..."}
```

`changeType`: `change | rename-pure | rename-changed | new | deleted`. For
`new` the old side is `""`, for `deleted` the new side is `""`. The old side
comes from `git show <base>:<oldPath>`, where base is `baseRef ?? HEAD` for
`working-tree` and the merge base for `branch-range`. The new side is read
from the **working tree**, not from `headRef`. Binary files, and files over 1
MiB, fail with a `GitCommandError`. Git repositories only.

## 5. Terminals

Source: `packages/contracts/src/terminal.ts`,
`apps/server/src/terminal/Manager.ts`,
`apps/server/src/terminal/OutputProtocol.ts`.

### Identity and encoding

- A terminal is `(threadId, terminalId)`. **The client picks both.** The
  server never allocates a terminal id. `threadId` is any non-empty string;
  official clients use the thread's id. `terminalId` is 1 to 128 characters,
  conventionally `term-1`, `term-2`, ... (`DEFAULT_TERMINAL_ID = "term-1"`).
- All terminal data is **plain UTF-8 text inside JSON strings**. No base64.
  Output (`data`, `history`) is the raw pty stream, so it contains ANSI escape
  sequences and `\r\n`. Input (`terminal.write` `data`) is what a terminal
  emulator would send for key presses: `"ls\r"` for Enter, `"\u0003"` for
  Ctrl+C, `"\u001b[A"` for the up arrow, pasted text as is. Feed output straight
  into an emulator (xterm.js, `vte` or similar).
- Dimensions: `cols` 1 to 1000, `rows` 1 to 500. Defaults 120 by 30.
- `env`: up to 128 entries, key `^[A-Za-z_][A-Za-z0-9_]*$` (max 128), value max
  8,192. The server also sets `COLORTERM=truecolor` by default and drops
  `TMUX`, `STY`, `ZELLIJ*`, `PORT` and similar from the inherited
  environment.
- `providerInstanceId` (optional) launches the shell with that provider
  instance's environment variables.

### terminal.open

Scope `terminal:operate`. Creates the session if it doesn't exist, starts the
shell, and returns a snapshot. It does not stream.

```json
{"threadId":"t1","terminalId":"term-1","cwd":"/home/me/code/app","worktreePath":null,"cols":120,"rows":30,"env":{"FOO":"bar"}}
```

Required: `threadId`, `terminalId`, `cwd`. Result `TerminalSessionSnapshot`:

```json
{
  "threadId": "t1",
  "terminalId": "term-1",
  "cwd": "/home/me/code/app",
  "worktreePath": null,
  "status": "running",
  "pid": 48213,
  "history": "",
  "exitCode": null,
  "exitSignal": null,
  "label": "zsh",
  "updatedAt": "2026-10-24T12:00:00.000Z",
  "sequence": 0
}
```

`status`: `starting | running | exited | error`. `history` is the replayable
scrollback (bounded; the server keeps up to 5,000 lines or 8 MiB, official
clients keep 512 KiB). `sequence` is the terminal's event counter.

Behaviour when the session already exists:

- Same `cwd`, `worktreePath` and `env`, process alive: nothing restarts. It
  resizes if you gave different `cols` / `rows` and returns the snapshot.
- Different `cwd`, `worktreePath` or `env`: the shell is stopped, history is
  cleared, and a new shell starts.
- Process exited: history is cleared and a new shell starts.

`cwd` must be an existing directory: `TerminalCwdNotFoundError`,
`TerminalCwdNotDirectoryError`, `TerminalCwdStatError`. `worktreePath`, when
set, is recorded and used as the lease for the workspace.

### terminal.attach (stream)

Scope `terminal:operate`. This is what the official UI uses instead of `open`:
it opens (when needed) and then streams.

```json
{"threadId":"t1","terminalId":"term-1","cwd":"/home/me/code/app","cols":120,"rows":30}
```

All but `threadId` and `terminalId` are optional: `cwd`, `worktreePath`,
`cols`, `rows`, `env`, `providerInstanceId`, `restartIfNotRunning`, `resume`.

- Session unknown and `cwd` given: starts a shell like `open`.
- Session unknown, no `cwd`: fails `TerminalSessionLookupError`.
- Session exists with a dead process, `restartIfNotRunning: true`, `cwd`
  given, and `resume` not true: restarts.
- `resume: true` is for resubscribing after a drop. It never restarts a
  stopped terminal, and if the terminal was closed it answers with a `closed`
  event. The official client sends the original payload for the first attach and
  adds `resume: true` for every retry after it got any reply
  (`packages/client-runtime/src/state/terminal.ts:52`).
- Differing `cols` / `rows` resize the pty.

Events (`TerminalAttachStreamEvent`), every one has `threadId`, `terminalId`
and optional `sequence`, except `snapshot`:

```json
{"type":"snapshot","snapshot":{ ...TerminalSessionSnapshot }}
{"type":"output","threadId":"t1","terminalId":"term-1","sequence":5,"data":"\u001b[32mok\u001b[0m\r\n"}
{"type":"exited","threadId":"t1","terminalId":"term-1","sequence":6,"exitCode":0,"exitSignal":null}
{"type":"restarted","threadId":"t1","terminalId":"term-1","sequence":7,"snapshot":{ ... }}
{"type":"cleared","threadId":"t1","terminalId":"term-1","sequence":8}
{"type":"closed","threadId":"t1","terminalId":"term-1","sequence":9}
{"type":"error","threadId":"t1","terminalId":"term-1","message":"..."}
{"type":"activity","threadId":"t1","terminalId":"term-1","hasRunningSubprocess":true,"label":"npm run dev"}
```

How to render: the first event is always `snapshot`. Write `snapshot.history`
into your emulator, then append each `output.data`. On `restarted`, reset the
emulator and write the new snapshot's `history`. On `cleared`, reset it. On
`closed` stop. A `snapshot` mid-stream means the server re-synced you: reset
and replay it. That happens if you fall behind by more than 8 MiB of queued
output; the stream then restarts from a fresh snapshot instead of dropping
data. Events that raced the snapshot and are already in it are filtered out.
`activity` is informational (a foreground child process is running, with its
command as `label`).

### terminal.observe (stream)

Scope `terminal:read`. Payload `{"threadId":"t1","terminalId":"term-1"}`.
Same events as attach, but read-only: it never creates, restarts or resizes
anything, and fails with `TerminalSessionLookupError` when the session doesn't
exist. Use it for a worktree setup script's terminal.

### terminal.write

Scope `terminal:operate`.

```json
{"threadId":"t1","terminalId":"term-1","data":"ls -la\r"}
```

`data` is 1 to 65,536 characters. No result. Writing to an unknown terminal
fails `TerminalSessionLookupError`. Writing to a session that isn't running
fails `TerminalNotRunningError`, except an `exited` session, which is silently
ignored. Pty failures are `TerminalWriteError {terminalPid, cause}`. The
echoed output comes back on the attach stream, not as the result.

### terminal.resize

Scope `terminal:operate`.

```json
{"threadId":"t1","terminalId":"term-1","cols":100,"rows":40}
```

No result. A missing or non-running session is a quiet no-op.

### terminal.clear

Scope `terminal:operate`. Payload `{"threadId","terminalId"}`. No result.
Clears the stored history and emits `cleared`. It does not clear the screen of
the running shell.

### terminal.restart

Scope `terminal:operate`.

```json
{"threadId":"t1","terminalId":"term-1","cwd":"/home/me/code/app","cols":120,"rows":30}
```

`cwd`, `cols` and `rows` are required here. Kills the shell, clears history,
starts a new one. Result is a `TerminalSessionSnapshot`. Subscribers see a
`restarted` event.

### terminal.close

Scope `terminal:operate`.

```json
{"threadId":"t1","terminalId":"term-1","deleteHistory":true}
```

Omit `terminalId` to close all of the thread's terminals. `deleteHistory`
also removes the persisted scrollback. No result. Subscribers see `closed`.

### subscribeTerminalMetadata

Scope `terminal:read`. Payload `{}`. Lightweight list of every terminal in the
environment, across all threads, without output.

```json
{"type":"snapshot","terminals":[{"threadId":"t1","terminalId":"term-1","cwd":"/home/me/code/app","worktreePath":null,"status":"running","pid":48213,"exitCode":null,"exitSignal":null,"hasRunningSubprocess":false,"label":"zsh","updatedAt":"2026-10-24T12:00:00.000Z"}]}
{"type":"upsert","terminal":{ ...TerminalSummary }}
{"type":"remove","threadId":"t1","terminalId":"term-1"}
```

The first event is `snapshot` (newest first). Replace your list on
`snapshot`, upsert on `upsert`, delete on `remove`.

### subscribeTerminalEvents

Scope `terminal:read`. Payload `{}`. The older firehose: every event of every
terminal in the environment (`started`, `output`, `exited`, `closed`, `error`,
`cleared`, `restarted`, `activity`, same shapes as above plus `started`
`{snapshot}`). There is no initial snapshot, and if the reader falls 8 MiB
behind the stream ends instead of re-syncing. Prefer attach or observe for
output and metadata for status.

### A minimal terminal flow

```js
// PoC rpc object from poc/public/t3.js
const ids = { threadId: "t1", terminalId: "term-1" };
await rpc.call("terminal.open", { ...ids, cwd: "/home/me/code/app", cols: 100, rows: 30 });
const stop = rpc.stream("terminal.attach", { ...ids, resume: true }, (event) => {
  if (event.type === "snapshot") term.reset(), term.write(event.snapshot.history);
  else if (event.type === "output") term.write(event.data);
  else if (event.type === "restarted") term.reset(), term.write(event.snapshot.history);
  else if (event.type === "cleared") term.reset();
});
await rpc.call("terminal.write", { ...ids, data: "echo hi\r" });
await rpc.call("terminal.resize", { ...ids, cols: 120, rows: 40 });
await rpc.call("terminal.close", { ...ids });
stop();
```

## 6. shell.openInEditor and assets

### shell.openInEditor

Scope `orchestration:operate`. Launches an editor **on the server host**, which
is only useful if someone sits at that machine. No result.

```json
{"cwd":"/home/me/code/app/src/index.ts:42:7","editor":"vscode"}
```

- `editor` is one of: `cursor`, `trae`, `kiro`, `vscode`, `vscode-insiders`,
  `vscodium`, `zed`, `antigravity`, `idea`, `aqua`, `clion`, `datagrip`,
  `dataspell`, `goland`, `phpstorm`, `pycharm`, `rider`, `rubymine`,
  `rustrover`, `webstorm`, `file-manager`
  (`packages/contracts/src/editor.ts:14`).
- Despite its name, `cwd` can be a file path. A trailing `:line` or
  `:line:column` is parsed and passed in the editor's own style (`--goto` for
  VS Code family, `--line/--column` for JetBrains, plain path for Zed).
- `file-manager` with `reveal: true` reveals the item in Finder, Explorer or
  the Linux file manager. The server advertises
  `shellRevealInFileManager` as a capability.
- Errors: `ExternalLauncherUnknownEditorError`,
  `ExternalLauncherUnsupportedEditorError`,
  `ExternalLauncherCommandNotFoundError {editor, command}`,
  `ExternalLauncherUnsupportedTargetError` (paths with line breaks or double
  quotes).

For a remote user the useful part is `buildRemoteOpenUrl` in the same file: it
builds `vscode://vscode-remote/ssh-remote+user@host/path` style URLs for your
own machine's editor.

### assets.createUrl

Mints a short-lived signed URL for a file the server can read, so a client can
use it in an `<img>` or download it without sending auth headers. Scope is
`filesystem:read` for `workspace-file`, `media-file` and
`draft-workspace-file`, and `orchestration:read` for every other resource kind.
Payload is `{"resource": <one of>}` with `_tag`:

| `_tag` | Fields |
| --- | --- |
| `workspace-file` | `threadId`, `path` (max 1024). Resolved under the thread's worktree or project root. |
| `media-file` | `threadId`, `path`. Absolute allowed. Images, video, audio, HTML, PDF only. |
| `draft-workspace-file` | `cwd`, `path`. For a thread that doesn't exist yet. |
| `attachment` | `attachmentId`, `fileName?`, `mimeType?`, `disposition?` (`inline \| attachment`), `renderIntent?` (`mcp-app`). |
| `tool-output-image` | `threadId`, `itemId`, `index`. |
| `project-favicon` | `cwd`, `path?`. |
| `native-app-icon` | `app`. |
| `github-media` | `cwd`, `url` (max 2048). Credentialed fetch of a PR body's media. |

```json
{"resource":{"_tag":"workspace-file","threadId":"t1","path":"docs/diagram.png"}}
```
```json
{"relativeUrl":"/api/assets/<token>/diagram.png","expiresAt":1793000000000,"sourcePath":"docs/diagram.png","imageDimensions":{"width":800,"height":600},"fileRevision":"..."}
```

Fetch `GET {httpBaseUrl}{relativeUrl}`. The token is the credential, so no
auth header is needed. The default lifetime is one hour (`expiresAt` is epoch
milliseconds). The response sets `private, max-age=3600` and
`x-content-type-options: nosniff`, and files support `Range` and `HEAD`
(`apps/server/src/http.ts:397`, `apps/server/src/assets/AssetAccess.ts:72`).
`imageDimensions` and `sourcePath` are present when known. `fileRevision`
changes only when the file does, unlike the re-signed URL. Errors are the
`Asset*Error` family (`not found`, `must be relative to the project root`,
`only images, videos, audio, HTML, and PDF files can be previewed`, and so on).

### attachments.createUploadUrl and attachments.delete

Scope `orchestration:operate` for both. Covered for images in `t3-connect.md`.
The full input is a union:

```json
{"type":"image","name":"shot.png","mimeType":"image/png","sizeBytes":123456}
{"type":"file","name":"report.pdf","mimeType":"application/pdf","sizeBytes":2048000}
```

- Images: `png | jpeg | gif | webp`, 1 byte to 10 MiB. `type` may be omitted.
- Files: any MIME type up to 100 characters, 1 byte to 50 MiB.
- Name max 255.

Result `{"attachmentId":"...","relativeUrl":"/api/attachments/upload/<payload>.<sig>","expiresAt":...}`.
`POST` the raw bytes to `{httpBaseUrl}{relativeUrl}` with a `Content-Length`
that equals `sizeBytes` exactly (else 400). The URL is signed and valid for 10
minutes; success is `204`. Then reference it in `message.dispatch`:

```json
{"attachments":[{"type":"file","id":"<attachmentId>","name":"report.pdf","mimeType":"application/pdf","sizeBytes":2048000}]}
```

At most 100 attachments per message, images 80 MiB in total. To record the
attachment as context too, add a `file` or `image` record
(`{version:1, kind:"file", contextId, label, attachmentId, name, mimeType, sizeBytes}`)
and a `[label](t3-context://v1/file/<contextId>)` reference in the text, which
is what the web app does. `attachments.delete` with `{"attachmentId"}` discards
an upload that was never sent. No result.

### assets.persistChatAttachments

Scope `orchestration:operate`. The older path: send base64 images inline.
Payload:

```json
{"threadId":"t1","messageId":"m1","attachments":[{"type":"image","name":"a.png","mimeType":"image/png","sizeBytes":1234,"dataUrl":"data:image/png;base64,..."}]}
```

Result `{"attachments":[ChatAttachment...]}`. `dataUrl` is limited to 14,000,000
characters. Prefer the upload URL, which avoids base64 in a WebSocket frame.

## 7. Required scopes

The server decides in `apps/server/src/auth/RpcAuthorization.ts`. The
client-side subset is `CLIENT_GUARDED_RPC_SCOPES` in
`packages/contracts/src/clientRpcPermissions.ts`, and the server enforces those
too. A missing scope fails the call with:

```json
{"_tag":"EnvironmentAuthorizationError","message":"The authenticated token is missing required scope: terminal:read.","requiredScope":"terminal:operate","requiredPermission":"terminal:read"}
```

`requiredPermission` is the real scope. `requiredScope` is a legacy parent for
old clients (`filesystem:read` reports `orchestration:read`, `terminal:read`
reports `terminal:operate`, `source-control:write` and `filesystem:write`
report `orchestration:operate`; `packages/contracts/src/auth.ts:100`). A
session granted only legacy `scopes` (no `permissions` list) is treated by the
client as holding the children of any parent scope it has. A server never
expands grants.

When creating a pairing token, ask for exactly these:

| Method | Scope |
| --- | --- |
| `projects.listEntries`, `projects.listIgnoredFiles`, `projects.readFile`, `projects.searchContents`, `projects.searchEntries` | `filesystem:read` |
| `projects.writeFile` | `filesystem:write` |
| `projects.mutate` | `orchestration:operate`, plus `settings:write` if a `scripts` key is present (create or update) |
| `projects.ensureScratch`, `projects.createNew` | `orchestration:operate` |
| `projects.list`, `projects.add`, `projects.remove` | not callable |
| `projectClone.start`, `.cancel`, `.retry` | `source-control:write` |
| `subscribeProjectClones` | `orchestration:read` |
| `filesystem.browse`, `filesystem.getMetadata` | `filesystem:read` |
| `subscribeVcsStatus`, `vcs.refreshStatus`, `vcs.listRefs` | `orchestration:read` |
| `vcs.pull`, `vcs.createWorktree`, `vcs.removeWorktree`, `vcs.createRef`, `vcs.switchRef`, `vcs.init` | `source-control:write` |
| `git.runStackedAction` | `source-control:write`; plus `orchestration:read` when `threadId` is set and `action` is `create_pr` or `commit_push_pr` |
| `git.resolvePullRequest` | `orchestration:read` |
| `git.preparePullRequestThread` | `source-control:write`; plus `orchestration:operate` when `mode` is `worktree` and `threadId` is set |
| `subscribeWorktreeSetup` | `orchestration:read` |
| `worktreeSetup.cancel` | `orchestration:operate` |
| `sourceControl.lookupRepository`, `server.discoverSourceControl` | `orchestration:read` |
| `sourceControl.cloneRepository`, `sourceControl.publishRepository` | `source-control:write` |
| `orchestration.getTurnDiff`, `orchestration.getFullThreadDiff` | `orchestration:read` |
| `review.getDiffPreview`, `review.getDiffFileContents` | `filesystem:read` |
| `orchestration.dispatchCommand` (`message.dispatch`) | `orchestration:operate` |
| `terminal.open`, `.attach`, `.write`, `.resize`, `.clear`, `.restart`, `.close` | `terminal:operate` |
| `terminal.observe`, `subscribeTerminalEvents`, `subscribeTerminalMetadata` | `terminal:read` |
| `shell.openInEditor` | `orchestration:operate` |
| `assets.createUrl` | `filesystem:read` for `workspace-file`, `media-file`, `draft-workspace-file`; else `orchestration:read` |
| `assets.persistChatAttachments`, `attachments.createUploadUrl`, `attachments.delete` | `orchestration:operate` |

A client that only browses and chats needs `orchestration:read`,
`orchestration:operate` and `filesystem:read`. Add `terminal:operate` (or just
`terminal:read`) for a terminal, `source-control:write` for git actions,
`filesystem:write` for file edits. Scope names are listed in
`packages/contracts/src/auth.ts:30`.

## 8. Using it from the PoC

`poc/public/t3.js` has `rpc.call(tag, payload)` and
`rpc.stream(tag, payload, onValue, onEnd)` (returns a cancel function). Both
already speak the framing, including the Ack. Everything above is a one-liner:

```js
const hits = await rpc.call("projects.searchEntries", { cwd, query: "inde", limit: 80 });
const mention = `[${base}](${encodeURI(path).replace(/[()#?\\]/g, c => "%" + c.charCodeAt(0).toString(16).toUpperCase())}) `;
await rpc.dispatch({ type: "message.dispatch", threadId, messageId: crypto.randomUUID(), text: `Look at ${mention}`,
  attachments: [], createdBy: "user", creationSource: "web", dispatchMode: { type: "start_immediately" } });
const diff = await rpc.call("orchestration.getTurnDiff", { threadId, fromTurnCount: 0, toTurnCount: 1 });
rpc.stream("subscribeVcsStatus", { cwd, includeRemote: true }, (e) => { /* snapshot | localUpdated | remoteUpdated */ });
```

`rpc.dispatch` adds `commandId` for you. The label escaping for `\ [ ]` is left
out of the snippet; add it if file names can contain those.

## What I did not verify

- I read the server code, not a running server (the brief said not to contact
  one). Example values in the payloads are illustrative, though every key and
  enum is from the schemas.
- I did not trace how `projects.mutate`'s optional `scripts` interacts with
  `runOnWorktreeCreate` beyond the scope rule.
- Whether providers other than the ones reaching
  `ProviderTurnStartService.ts` and `ProviderTurnControlService.ts` expand
  context references. I only confirmed those two call
  `projectComposerContextForProvider`.
- `jj` is a declared VCS kind. I only read the git paths.
