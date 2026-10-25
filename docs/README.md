# T3 Code client docs

Everything an unofficial T3 Code client needs: how to connect, and every RPC
and command the official web app uses. Written from upstream
`pingdotgg/t3code` at `80455376` (2026-10-24) and checked, where noted, against
a local `0.1.3-nightly.20261023` server. None of it is a stable public API.

Read them in this order:

1. [t3-connect.md](t3-connect.md): T3 Connect sign-in, the relay, environment
   sessions with DPoP, the WebSocket RPC framing, turn items, bounded
   snapshots, history paging, image upload. Start here.
   [clerk.md](clerk.md) covers the Clerk sign-in it needs.
2. [orchestration.md](orchestration.md): every `dispatchCommand` command
   (threads, messages, queue, runs, modes, approvals, questions, plans,
   checkpoints) and every other `orchestration.*` RPC. Covers search, diffs,
   the stream events and how to apply them, and the compact ID table.
3. [server-settings-models.md](server-settings-models.md): `server.getConfig`,
   providers, models and option descriptors (the model picker), settings and
   how to save them, keybindings, provider auth and install, usage, diagnostics.
4. [projects-files-vcs-terminal.md](projects-files-vcs-terminal.md): projects,
   `@` file search, files, git and worktrees, diffs, terminals.
5. [integrations.md](integrations.md): pull requests, scheduled tasks and
   webhooks, guest shares, secret requests, MCP apps, previews and devices,
   and every auth scope.
6. [t3-web-features.md](t3-web-features.md): every feature of the web app,
   mapped to the calls behind it, with a checklist for reaching parity.

The browser proof of concept in [`../poc`](../poc) uses these docs. Its
README lists what it covers so far.

## Things that bite

- Every command needs a fresh `commandId`. Retrying with the same one is safe.
- Check `config.environment.capabilities` before using newer behaviour, such
  as `threadStop` or `serverResolvedCommandContext`. Older servers lack them.
- Scope errors put the scope you actually need in `requiredPermission`, not
  in `requiredScope`.
- A T3 Connect or pairing login gets all 16 scopes by default, including
  `terminal:operate`. Treat its tokens like a shell login.
- `server.getConfig` is about 400 KB, mostly Claude's per-folder command and
  skill lists. Fetch it once and apply `subscribeServerConfig` updates.
- Reply text from Claude doesn't stream; only thinking does.
