# How T3 Connect works, for a third-party client

This explains how an official T3 Code client finds and connects to an
environment through T3 Connect, and what an unofficial Rust client has to do
to get the same result. It comes from reading upstream `pingdotgg/t3code` at
`453bf01d` (2026-10-25). Paths below are relative to that repo. The upstream
docs worth reading after this one are `docs/internals/t3-connect.md`,
`docs/internals/environment-auth.md` and `docs/internals/remote.md`.

This is the first of several docs; [README.md](README.md) lists the rest
(orchestration commands, settings and models, projects and terminals,
integrations, and the web feature inventory).

None of this is a stable public API. The relay is versioned on its own and
deploys on every push to upstream `main`. Expect to chase changes.

## The short version

T3 Connect is a control plane, not a proxy. The relay at
`https://relay.t3.codes` does three jobs for a client:

1. Lists the environments linked to your T3 Connect (Clerk) account.
2. Tells you each environment's public tunnel URL, a Cloudflare Tunnel
   hostname like `https://prod-<digest>.<tunnel-zone>`.
3. Asks the environment to mint a one-time bootstrap credential for you.

After that the client talks straight to the environment over the tunnel URL.
Every HTTP request and the WebSocket go to the environment, never through the
relay. The environment issues and checks its own sessions. A relay token is
never an environment login.

```
 Rust client                    relay.t3.codes                 environment (t3 serve)
     |                                |                                  |
     |-- Clerk session JWT ---------->|  POST /v1/client/dpop-token      |
     |<- relay DPoP access token -----|                                  |
     |-- GET /v1/environments ------->|                                  |
     |<- [{environmentId, endpoint}] -|                                  |
     |-- POST /v1/environments/:id/connect --->|-- signed mint request -->|
     |<- {endpoint, credential} ------|<- signed, key-bound credential --|
     |                                                                   |
     |-- POST {httpBaseUrl}/oauth/token  (credential + DPoP proof) ----->|
     |<- environment access token (DPoP, 1 hour) ------------------------|
     |-- POST {httpBaseUrl}/api/auth/websocket-ticket/rpc -------------->|
     |<- {ticket, expiresAt} (5 minutes) --------------------------------|
     |== WSS {wsBaseUrl}/ws?wsTicket=...&dpop=... =======================|
```

## Two ways in, and which one I'd pick first

A Rust client can reach an environment's tunnel URL without T3 Connect sign-in
at all. The tunnel only forwards to the environment's loopback HTTP origin, and
the environment accepts any valid session on any route.

**Path A, pairing over the tunnel URL.** Run `t3 pair` (or
`t3 auth pairing create`) on the host. It prints a URL like
`https://<host>/pair#token=<credential>`. Swap the host for the Connect tunnel
hostname if it isn't already. The client exchanges that token once for a
bearer session that lasts 30 days. No Clerk, no DPoP, no relay. The upstream
docs confirm that a paired bearer token works on every route the environment
has, and the tunnel is just another route.

**Path B, full T3 Connect.** Sign in with Clerk, use the relay to discover
environments, and get DPoP-bound sessions that last 1 hour and renew through
the relay. This is what the official apps do and it's the only way to get
automatic discovery of every machine on your account.

Path A is the one to build first. It's a few hundred lines of Rust, it works on
a watch, and it exercises the same environment API and WebSocket that path B
ends at. Path B adds Clerk and DPoP on top. The hard part of path B is Clerk,
covered below.

## Path A: pairing

### 1. Get a pairing token

On the host:

```sh
t3 pair --label "Apple Watch"
# narrower grant:
t3 pair --scope orchestration:read --scope orchestration:operate --label "Apple Watch"
```

The token is single-use and expires after 5 minutes by default (`--ttl` changes
that). Without `--scope` it grants everything, including `terminal:operate`
(a shell as the server's user). A watch client that only reads threads and
sends messages should ask for less.

The pairing URL format comes from `apps/server/src/startupAccess.ts:153`. The
path is `/pair` and the token sits in the fragment as `token=<credential>`, so
it never hits a server log.

### 2. Check you're talking to the right environment

`GET {httpBaseUrl}/.well-known/t3/environment` is unauthenticated and returns
the environment descriptor (including its `environmentId`). Clients save it and
check it on reconnect, so a different machine answering on the same address
never receives a credential.

### 3. Exchange the token for a session

`POST {httpBaseUrl}/oauth/token`, form-urlencoded:

```
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
subject_token=<pairing credential>
subject_token_type=urn:t3:params:oauth:token-type:environment-bootstrap
requested_token_type=urn:ietf:params:oauth:token-type:access_token
scope=orchestration:read orchestration:operate      (optional, can only narrow)
client_label=Apple Watch                            (optional)
client_device_type=mobile                           (optional)
client_os=watchos                                   (optional)
```

Response (`packages/contracts/src/auth.ts:234`):

```json
{
  "access_token": "...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 2592000,
  "scope": "orchestration:read orchestration:operate"
}
```

Store `access_token` in the Keychain or Windows Credential Manager. It is the
whole credential.

### 4. Open the WebSocket

1. `POST {httpBaseUrl}/api/auth/websocket-ticket/rpc` with
   `Authorization: Bearer <access_token>`. Response is
   `{"ticket": "...", "expiresAt": "<ISO-8601>"}`. Tickets last 5 minutes, so
   mint one right before each connect. If the server answers 404 it's older
   than ticket audiences; retry at `/api/auth/websocket-ticket`. The
   `0.1.3-nightly.20261023` build does this, so the fallback is needed today.
2. Connect to `wss://<host>/ws?orchestrationProtocol=2&wsTicket=<ticket>`.
   Without `orchestrationProtocol=2` the server answers HTTP 426 before it
   even looks at the ticket.

The official clients also append analytics-ish query parameters
(`clientSurface`, `clientAppVersion`, `clientDeviceType`, `clientOs`,
`connectionMethod`), see `packages/client-runtime/src/authorization/remote.ts:62`.
They're optional.

`GET {httpBaseUrl}/api/auth/session` with the bearer token tells you whether the
session is still valid and what it's allowed to do. Read `permissions` when
present and fall back to `scopes` on older servers.

## Path B: full T3 Connect

### 1. Get a Clerk session JWT

The relay needs a Clerk **session** token minted with the JWT template
`t3-relay` (audience `t3-code-relay`). The official web and mobile apps call
Clerk's `getToken({ template: "t3-relay" })`.

Public production identifiers, from upstream `.env.example`. These are baked
into official builds and aren't secrets:

```
T3CODE_CLERK_PUBLISHABLE_KEY=pk_live_Y2xlcmsudDMuY29kZXMk   # Frontend API: clerk.t3.codes
T3CODE_CLERK_JWT_TEMPLATE=t3-relay
T3CODE_CLERK_CLI_OAUTH_CLIENT_ID=hzxSgY2cH10sDU2r
T3CODE_RELAY_URL=https://relay.t3.codes
```

Here's the trap. The relay has two kinds of authenticated endpoints:

| Endpoints | Accepts |
| --- | --- |
| `GET /v1/environments`, link/unlink endpoints (`Authorization: Bearer`) | Clerk session JWT with the `t3-relay` audience, **or** a Clerk OAuth access token |
| `POST /v1/client/dpop-token` | Clerk session JWT with the `t3-relay` audience **only** |

So the CLI's OAuth app (with its device-code flow, which would be perfect for
a watch) gets you a token that can list your environments but can't be
exchanged for the DPoP token that `connect` needs
(`infra/relay/src/http/Api.ts:1035` calls `verifyClerkBearerToken` with no
OAuth fallback). To connect you need a real Clerk session.

The full sign-in flow is in [clerk.md](clerk.md). Options for a Rust client,
roughly from least to most work:

- **Borrow a session from a platform SDK.** On Apple platforms the Clerk iOS
  SDK can sign in and call `getToken(template:)`. A watchOS app can do the
  sign-in on the paired iPhone and hand the session over. Windows has no Clerk
  SDK.
- **Talk to Clerk's Frontend API directly.** That's what Clerk's native SDKs do
  under the hood: create a sign-in on `https://clerk.t3.codes/v1/client`,
  complete it (email code is the one that needs no redirect URL), then
  `POST /v1/client/sessions/{session_id}/tokens/t3-relay`. OAuth providers
  (GitHub, Google) need a redirect URL on T3's allowlist, which a third-party
  client won't be on. This is undocumented territory for a third party. Clerk
  can change it, and T3 can restrict it.
- **Run your own relay and Clerk instance.** `infra/relay` deploys with Alchemy
  to Cloudflare. That's a lot of infrastructure for a client project.

Clerk session JWTs are short-lived (Clerk's default is 60 seconds), so mint a
fresh one right before each relay call that needs it.

### 2. Make a DPoP key

Generate one P-256 key pair per install and keep it. Every relay and
environment session in path B is bound to it. In Rust, `p256` with the `ecdsa`
and `jwk` features covers it.

A DPoP proof is a compact JWS (`packages/shared/src/dpop.ts`):

```
header  {"typ":"dpop+jwt","alg":"ES256","jwk":{"kty":"EC","crv":"P-256","x":"...","y":"..."}}
payload {"htm":"POST","htu":"https://relay.t3.codes/v1/client/dpop-token",
         "jti":"<random uuid>","iat":<unix seconds>,"ath":"<optional>"}
```

Rules the verifier enforces:

- `htu` is the full request URL with query and fragment stripped.
- `htm` matches the HTTP method.
- `ath` is `base64url(sha256(access_token))`, required when the request also
  carries an access token.
- `iat` must be within 300 seconds of the server clock.
- `jti` is single-use. Make a new proof for every request, retries included.
- The signature is raw 64-byte `r||s`, base64url, over `sha256(header.payload)`.

The thumbprint is `base64url(sha256('{"crv":"P-256","kty":"EC","x":"…","y":"…"}'))`
with keys in exactly that order and no whitespace. That's standard RFC 7638.

### 3. Exchange the Clerk JWT for a relay DPoP token

`POST https://relay.t3.codes/v1/client/dpop-token`, header `dpop: <proof>`,
form-urlencoded body:

```
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
subject_token=<Clerk session JWT, template t3-relay>
subject_token_type=urn:ietf:params:oauth:token-type:jwt
requested_token_type=urn:ietf:params:oauth:token-type:access_token
resource=https://relay.t3.codes
scope=environment:connect environment:status
client_id=t3-web
```

`client_id` must be `t3-web` or `t3-mobile`. `t3-mobile` also allows the
`mobile:registration` scope (push tokens). `resource` must equal the relay
issuer exactly. Response:

```json
{
  "access_token": "...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "DPoP",
  "expires_in": 1800,
  "scope": "environment:connect environment:status"
}
```

It lasts 30 minutes. Check that `scope` matches what you asked for; the
official client treats a mismatch as an error.

### 4. List environments

`GET https://relay.t3.codes/v1/environments` with
`Authorization: Bearer <Clerk token>`:

```json
{
  "environments": [
    {
      "environmentId": "...",
      "label": "workstation",
      "endpoint": {
        "httpBaseUrl": "https://prod-<digest>.<zone>",
        "wsBaseUrl": "wss://prod-<digest>.<zone>",
        "providerKind": "..."
      },
      "linkedAt": "..."
    }
  ]
}
```

`POST /v1/environments/:environmentId/status` (DPoP, scope
`environment:status`) reports `online` or `offline`, plus why: `tunnel_down`
means no `cloudflared` is attached, `origin_unreachable` means the tunnel is up
but T3 Code behind it isn't. Good for a watch complication.

### 5. Connect

`POST https://relay.t3.codes/v1/environments/:environmentId/connect` with:

```
Authorization: DPoP <relay access token>
dpop: <proof, htm=POST, htu=this URL, ath=hash of relay access token>
```

```json
{ "clientProofKeyThumbprint": "<your key's thumbprint>", "deviceId": "optional" }
```

The thumbprint must match the key that signed the proof. The relay asks the
environment to mint a bootstrap credential bound to that thumbprint, checks
the environment's signed answer, and returns:

```json
{
  "environmentId": "...",
  "endpoint": { "httpBaseUrl": "...", "wsBaseUrl": "...", "providerKind": "..." },
  "credential": "<one-time bootstrap credential>",
  "expiresAt": "..."
}
```

The relay never sees your environment session. A stolen `credential` is useless
without your private key.

### 6. Exchange the credential with the environment

Same `/oauth/token` call as path A, sent to `endpoint.httpBaseUrl`, plus a
`dpop` header (proof with `htm=POST`, `htu={httpBaseUrl}/oauth/token`, no
`ath`). The response has `"token_type": "DPoP"` and lasts 1 hour. A DPoP session
from T3 Connect gets the same default scopes as a full pairing.

From here every environment HTTP request carries
`Authorization: DPoP <env token>` and a fresh `dpop` proof whose `ath` hashes
the env token. When it expires, run steps 3, 5 and 6 again. The official client
does this in the background without dropping the WebSocket.

### 7. Open the WebSocket

1. `POST {httpBaseUrl}/api/auth/websocket-ticket/rpc` with
   `Authorization: DPoP <env token>` and a proof for that URL.
2. Make a second proof with `htm=GET`, `htu=https://<host>/ws` (the `wss`
   scheme swapped to `https`, no query), and `ath` hashing **the ticket**, not
   the access token.
3. Connect to `wss://<host>/ws?orchestrationProtocol=2&wsTicket=<ticket>&dpop=<proof>`. The proof goes
   in the query because a WebSocket upgrade can't set custom headers.

See `resolveRemoteDpopWebSocketConnectionUrl` in
`packages/client-runtime/src/authorization/remote.ts:303`.

## Errors

Relay errors are JSON with a `_tag`, a `code` and a `traceId`. The ones worth
handling:

- `RelayAuthInvalidError` (401), `reason` is one of `missing_bearer`,
  `invalid_bearer`, `invalid_dpop`, `not_authorized`. `invalid_dpop` with
  `dpopFailureReason` tells you which proof rule you broke. Clock skew shows up
  as `time_window`.
- `RelayEnvironmentConnectNotAuthorizedError`, e.g.
  `environment_link_not_found` when the environment was unlinked.
- `RelayEnvironmentEndpointUnavailableError` and
  `RelayEnvironmentEndpointTimedOutError` when the environment is offline.

Environment permission errors carry `requiredScope` and `requiredPermission`.

## Lifetimes at a glance

| Thing | Lifetime | Source |
| --- | --- | --- |
| Pairing token | 5 minutes, single use | `apps/server/src/auth/PairingGrantStore.ts:249` |
| Bearer session (pairing) | 30 days | `apps/server/src/auth/SessionStore.ts:432` |
| DPoP session (Connect) | 1 hour | `apps/server/src/auth/EnvironmentAuth.ts:990` |
| WebSocket ticket | 5 minutes | `apps/server/src/auth/SessionStore.ts:433` |
| Relay DPoP access token | 30 minutes | `infra/relay/src/auth/RelayTokens.ts:29` |
| DPoP proof clock skew | ±300 seconds | `packages/shared/src/dpop.ts:18` |

## The WebSocket protocol

The socket speaks Effect's built-in RPC protocol (`effect/rpc`) with JSON
serialization. Each text frame is one JSON message. A frame can also hold a
JSON array of messages, so accept both. Server setup is in
`apps/server/src/ws.ts:3204`; the client is
`packages/client-runtime/src/rpc/session.ts:209`. There's no hello frame. The
connection is ready when your first request gets an answer.

Client to server:

```json
{"_tag":"Request","id":0,"tag":"server.probe","payload":{},"headers":[]}
{"_tag":"Ack","requestId":0}
{"_tag":"Interrupt","requestId":0,"interruptors":[]}
{"_tag":"Ping"}
```

Server to client:

```json
{"_tag":"Chunk","requestId":0,"values":[ ... ]}
{"_tag":"Exit","requestId":0,"exit":{"_tag":"Success","value":{}}}
{"_tag":"Exit","requestId":0,"exit":{"_tag":"Failure","cause":[{"_tag":"Fail","error":{"_tag":"EnvironmentAuthorizationError","message":"...","requiredScope":"orchestration:read"}}]}}
{"_tag":"Defect","defect":"..."}
{"_tag":"Pong"}
```

- `id` is yours to pick. Effect uses an incrementing number. `headers` is a
  list of `[key, value]` pairs; send `[]`. Optional `traceId`, `spanId` and
  `sampled` fields can be left out.
- A plain call answers with one `Exit`. A stream sends `Chunk`s and finishes
  with an `Exit`.
- **Ack every `Chunk`.** The server waits for `{"_tag":"Ack","requestId":n}`
  before sending the next chunk of that stream. Forget it and every
  subscription stalls after its first chunk, which will look like a server bug.
  Terminal streams allow 8 chunks or 64 KB of slack, everything else allows
  none.
- Cancel a call or stream with `Interrupt`.
- A `cause` entry is `Fail` (a typed error), `Die` (a server bug) or
  `Interrupt`. A `Defect` with no `requestId` kills the whole socket.
- Values use the schema's encoded form, so dates arrive as ISO strings.
- Send `Ping` every 5 seconds. The official client drops the socket after 15
  seconds with no frame and reconnects with jittered backoff from 1 second up
  to 5 minutes.

### Methods a small client needs

Tags live in `packages/contracts/src/rpc.ts` and
`packages/contracts/src/orchestrationV2.ts:3185`. Required scopes are in
`apps/server/src/auth/RpcAuthorization.ts`.

| Tag | Kind | Scope | What it's for |
| --- | --- | --- | --- |
| `server.probe` | call | `orchestration:read` | Cheap liveness check. `{}` in, `{}` out. |
| `server.getConfig` | call | `orchestration:read` | Server config. |
| `subscribeServerConfig` | stream | `orchestration:read` | Config snapshot then updates. The official client's first request. |
| `orchestration.subscribeShell` | stream | `orchestration:read` | Projects and threads. This is how you list them; there's no list RPC. |
| `orchestration.subscribeThread` | stream | `orchestration:read` | One thread's messages and events. |
| `orchestration.getThreadProjection` | call | `orchestration:read` | One-shot read of a thread. |
| `orchestration.launchThread` | call | `orchestration:operate` | Start a new thread. |
| `orchestration.dispatchCommand` | call | `orchestration:operate` | Everything else that changes a thread. Returns `{sequence}`. |

`subscribeShell` takes `{afterSequence?, requestCompletionMarker?}` and sends
items with a `kind` of `snapshot` (holding `projects`, `threads`,
`archivedThreads`), `project.updated`, `project.removed`, `thread.updated`,
`thread.removed` or `synchronized`.

`subscribeThread` takes `{threadId, afterSequence?, requestCompletionMarker?}`
and sends `snapshot` (with `snapshotSequence` and `projection`), `event` (with
`sequence` and `event`) and `synchronized`. Unknown events decode as
`unknown-event`, so new server event types won't break an old client if you
keep a catch-all.

Three things the PoC in `poc/` ran into against a real server:

- The shell stream can send a second `snapshot` that carries
  `resolvedRepositoryIdentityRoots` and an empty `threads` array. It's a
  project-metadata refresh, and its `projects` holds only the projects whose
  repo identity changed. Merge those projects in by `id` and keep everything
  else. If you replace your projects or threads with it, your thread list goes
  blank or loses its project names right after connecting.
- `orchestration.getThreadProjection` returns only a recent tail of messages
  (17 of 28 in one test). The `snapshot` from `subscribeThread` has all of
  them. Don't refetch the projection on every event.
- `message.updated` events carry the complete message in `event.payload`.
  Upsert it by `id`. It isn't the live stream, though. While the agent is
  typing, the text comes in `turn-item.updated` events with
  `payload.type: "assistant_message"`, `payload.messageId`, `payload.text`
  (the full text so far) and `payload.streaming`. The official web app
  renders from turn items for this reason. Render the thread from turn items
  in `ordinal` order, not from `messages`.
- With the Claude provider, reply text does not stream. The adapter forwards
  thinking deltas only. Each reply block arrives as one `turn-item.updated`
  with `streaming: false` and the finished text. Show `reasoning` items and
  running tools if you want the user to see progress.
- Full thread snapshots get big: 16.8 MB for a 2,700-item thread. Subscribe
  with `acceptBoundedSnapshot: true` and the snapshot is the recent window
  plus `historyCursor` and `hasMoreHistory`. Add `acceptCompactTurnItems:
  true` to drop the duplicate `turnItems` list and render from
  `visibleTurnItems[].item`. Older pages come from
  `GET /api/orchestration/threads/:threadId/history?cursor=…`, with the same
  DPoP auth as the ticket route plus `x-t3-orchestration-protocol: 2`. It
  returns `{items, nextCursor, hasMoreHistory}`.
- Tool calls are turn items too (`command_execution`, `file_change`,
  `file_search`, `web_search`, `subagent`, `dynamic_tool`). The socket
  strips command output and diffs, and sets `outputOmitted: true`. Fetch the
  full item with `orchestration.getTurnItem {threadId, itemId}` when the user
  opens it.
- Images go up before the message. Call `attachments.createUploadUrl
  {type:"image", name, mimeType, sizeBytes}` (png, jpeg, gif or webp, 10 MB
  max). It returns `{attachmentId, relativeUrl}`. POST the raw bytes to
  `relativeUrl` on the environment's `httpBaseUrl` with the image's
  `Content-Type`. The URL is signed, so the POST carries no auth. Then put
  `{type:"image", id: attachmentId, name, mimeType, sizeBytes}` in
  `message.dispatch`'s `attachments`.

After a reconnect, subscribe again with `afterSequence` set to the last
sequence you saw and `requestCompletionMarker: true`. The server replays what
you missed and sends `synchronized` when you're caught up.

`dispatchCommand` takes a tagged command with `commandId` and `threadId`. The
three a watch needs:

- `message.dispatch` sends a message: `messageId`, `text`, `attachments`, and
  `dispatchMode` (`start_immediately`, `queue_after_active`,
  `steer_active`, `restart_active` or `defer_start`).
- `run.interrupt` stops a run: `runId`, optional `reason` and `holdQueue`.
- `runtime-request.respond` answers an approval or question: `requestId` and a
  `decision` of `accept`, `acceptForSession`, `acceptAlways`, `decline` or
  `cancel`, or `answers` for questions.

The exact schemas are in `orchestrationV2.ts` around lines 2650 to 3106. Don't
hand-copy them into Rust and forget about them. Generate the serde types from
the TypeScript schemas, or at least keep a test that replays recorded frames.

`apps/server/scripts/verify-background-live.ts:250` is a complete minimal
client in about 150 lines: token, ticket, socket, `launchThread`,
`subscribeThread`. Port that first.

The official clients also send `x-t3-orchestration-protocol: 2` on HTTP
requests. `GET /.well-known/t3/environment` returns `serverVersion`,
`orchestrationProtocolVersion` (missing means 1) and `capabilities`. Check
them before using optional features.

As far as I could find, nobody has reimplemented this protocol outside
TypeScript. The Swift and Kotlin code in the mobile app is UI only and goes
through the shared TS runtime. Your Rust client would be the first.

## Sessions, sockets and sign-out

Checked against upstream `e9d61252` and a live nightly server on 2026-10-24.

- **A live socket outlives its session.** The server checks auth only when it
  redeems the WebSocket ticket. I revoked a pairing session with
  `t3 auth session revoke` while its socket was open, and RPCs on that socket
  kept working 13 seconds later. Expiry works the same way. So the 1-hour
  Connect session only matters for HTTP calls (tickets, attachments) and
  reconnects. Renew it before you need one of those, not on a timer that
  tears the socket down.
- **Turn a Connect login into a 30-day one.** `POST /api/auth/pairing-token`
  from an existing session (needs `access:write`, which Connect sessions have)
  mints a pairing credential that exchanges for a 30-day bearer session. A
  phone or desktop app signed in through Connect can mint one for the watch,
  and the watch never touches Clerk, the relay or DPoP.
- **Signing out properly** takes three steps: revoke the environment session
  (`POST /api/auth/clients/revoke` with its `sessionId`, from
  `GET /api/auth/session`), end the Clerk session
  (`POST https://clerk.t3.codes/v1/client/sessions/{id}/end`), then delete the
  DPoP key and cached tokens. Because of the first bullet, also close the
  socket yourself.
- **Clerk sign-in has one step on T3 today.** T3's Clerk instance has no
  second factors turned on, so email code or password goes straight to
  `complete`. [clerk.md](clerk.md) covers the whole flow and what to do if
  that changes.
- **Clock skew.** Proofs with `iat` more than 300 seconds off fail as
  `invalid_dpop` / `time_window`. Watches and VMs drift. Read the `Date`
  header from the failed response, keep the offset, and retry once with a new
  proof.

## Other relay endpoints

`packages/contracts/src/relay.ts` has more than the connect flow:

| Endpoint | Auth | What it's for |
| --- | --- | --- |
| `GET /v1/mobile/agent-activity` | DPoP, scope `mobile:registration` (needs `client_id=t3-mobile` at the token exchange) | `{aggregate: {title, subtitle, activeCount, updatedAt, activities[]} \| null}`, the summary behind T3's Live Activity across all your environments. One HTTP call, no socket, so it suits a watch complication. Environments only publish it when agent activity publishing is turned on. |
| `GET /v2/client/devices` | Clerk bearer | Mobile devices registered for push |
| `POST /v1/client/environment-link-challenges`, `POST /v1/client/environment-links` | Clerk bearer | Link a machine to the account. `t3 connect` on the host does this already, so a client can skip it. |
| `DELETE /v1/client/environment-links/:environmentId` | Clerk bearer | Unlink a machine. Worth a button in settings. |
| `POST /v1/mobile/devices`, `/v1/mobile/live-activities` | DPoP, `mobile:registration` | Push registration. Useless to a third-party app, because the pushes are signed for T3's own app IDs (APNs and FCM). |

## Notes for the Rust side

- One crate for the protocol (`reqwest` + `tokio-tungstenite` + `serde` +
  `p256`), with thin per-platform shells. On Windows it can run as-is. watchOS
  needs a Swift UI layer calling into the Rust core through UniFFI, since
  there's no native Rust UI story there.
- watchOS suspends apps aggressively and limits background networking. Expect
  to reconnect on every wrist raise, so make "mint ticket and open socket"
  fast. That's another point for path A on the watch: no relay round trips
  before the socket opens.
- Push notifications go through the relay (`/v1/mobile/devices` with
  `t3-mobile` scope) and APNs keys for T3's bundle ID, so a third-party watch
  app can't receive T3's pushes. Your app would have to poll or keep its own
  socket.
- Remote servers outlive client releases. Check what the server advertises
  instead of assuming your client's version describes it.
