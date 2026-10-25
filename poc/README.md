# tern PoC

A small browser client for T3 Code that connects through T3 Connect. No
build step, no dependencies. See `../docs/t3-connect.md` for the protocol.

```sh
node poc/server.mjs          # http://127.0.0.1:4777

# On the LAN. WebCrypto needs https off localhost, so use a cert (mkcert works):
cd poc && HOST=0.0.0.0 TLS_CERT=certs/cert.pem TLS_KEY=certs/key.pem node server.mjs
```

Sign in with your T3 Connect email. Clerk emails you a code; password works too
if your account has one. GitHub/Google sign-in isn't available here because
Clerk only redirects to T3's own apps.

What it does, all checked against a real T3 server:

- **Sidebar.**
  - Environments with online status.
  - Threads grouped by project, with a Pinned group on top. Settled and
    snoozed threads fold under "Show N settled".
  - Unread threads show in bold, and the archived threads are behind
    "Archived".
  - The search box filters titles as you type and also searches message
    text on the server (`orchestration.searchThreads`).
- **New threads.** Start one from + (on the sidebar or a project). Pick the
  project, model, plan mode and permissions; the first message creates the
  thread with `orchestration.launchThread`.
- **Thread view.**
  - Messages, thinking, and tool calls; expand a tool row to fetch its
    output.
  - Like T3, a finished run folds into one "Worked for 19s" row above the
    reply.
- **Composer.**
  - The model picker shows ready providers, their models and each model's
    options (effort, fast mode, …).
  - It also has a Plan mode toggle and a permissions mode.
  - `/` lists commands, `$` lists skills and `@` mentions files.
  - Images attach by +, paste or drop.
  - Sending while a run is busy lets the server steer or queue. Queue (or
    Alt+Enter) always queues.
- **Agent requests.**
  - Queued messages can be sent now, edited or cancelled.
  - Approvals get Approve / For session / Decline.
  - Agent questions get their options and a free-text answer.
  - A proposed plan gets "Implement plan".
- **Thread menu (⋯).** Rename (or tap the title), regenerate title, pin,
  settle, snooze, mark unread, fork, diff of the last turn or the whole
  thread, archive, delete. Stop uses `thread.stop`.

## What runs where

- `server.mjs` serves `public/` and forwards `/clerk/v1/*` to
  `https://clerk.t3.codes`. Clerk rejects browser requests from any origin but
  t3.codes, so the proxy drops the Origin header. That's its only job.
- Everything else goes straight from the browser: the relay
  (`relay.t3.codes`, CORS `*`), the environment's tunnel URL, and the
  WebSocket.

## Speed

- The environment session (1 hour) is cached per environment, so a reload
  skips the relay: one ticket request, then the socket. About 0.6 s to a full
  thread list for ~1000 threads in testing.
- Threads open from a bounded snapshot (the recent window) with "Load
  earlier" for the rest. A 2,700-item thread is 16.8 MB as a full snapshot and
  opens in about 0.7 s bounded.
- Rows update in place from `turn-item.updated` events, batched to one render
  per animation frame. Tool output loads only when you expand a row.
- A socket with no pong for 15 s is treated as dead, a tab coming back to the
  foreground reconnects at once, and the 1-hour session renews itself 3
  minutes early.

## Limits

- WebCrypto needs a secure context: `http://localhost`, `http://127.0.0.1`,
  or https. Over plain http on a LAN or tailnet IP it refuses to start.
- Reply text doesn't stream. T3's Claude adapter forwards only thinking
  deltas; reply text arrives as finished blocks, in the official app too.
  Tern shows thinking and running tools, plus a shimmering "Thinking" row.
- The DPoP private key is a non-extractable CryptoKey in IndexedDB. The Clerk
  client token and environment access tokens sit in localStorage, which is
  fine for a PoC and not fine for anything else.
- One sign-in factor only. Accounts with 2FA get an error.
- MCP elicitations and auth refreshes still say to answer them in T3.
- New threads start in the project folder. There's no worktree option yet.
- Not built: settings pages, terminals, git and PR actions, scheduled
  tasks, guest shares, previews and devices. `../docs/` documents all of
  them.
- Claude's file edits carry no diff, so those rows show only the file name.
- Sent images show as "1 attachment"; images aren't displayed in the thread.
- No markdown beyond code spans, code blocks, bold and headings.
