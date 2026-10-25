# Signing in to T3 Connect through Clerk

T3 Connect accounts live in Clerk. To use the relay (path B in
[t3-connect.md](t3-connect.md)), a client needs a Clerk session and a
short-lived session JWT from the `t3-relay` template. This doc covers getting
both from Rust without a Clerk SDK, by calling Clerk's Frontend API (FAPI)
the way Clerk's own iOS and Android SDKs do.

None of this is a public API for third parties. It's the API Clerk's native
SDKs use, against T3's production instance. Clerk or T3 can change or block
it. The browser PoC (`poc/public/t3.js`, class `Clerk`) does all of it and
works today.

## What T3's instance allows

`GET https://clerk.t3.codes/v1/environment?_is_native=1` is public and
describes the instance. As of 2026-10-25:

| Setting | Value | What it means for us |
| --- | --- | --- |
| `native_settings.api_enabled` | `true` | Native (non-browser) FAPI calls are allowed |
| First factors | `email_code`, `password`, `passkey`, `oauth_github`, `oauth_google`, `oauth_apple`, `oauth_microsoft`, `oauth_token_apple`, `google_one_tap`, `ticket` | Only `email_code` and `password` work for a third-party client (see below) |
| Second factors | none | No two-step login. Every sign-in completes after the first factor. |
| `single_session_mode` | `true` | One signed-in account per Clerk client |
| Sign-up | public, captcha required | Send new users to the official app to make an account |
| `reverification` | `true` | Sensitive account changes may ask you to sign in again. Signing in doesn't. |

Check this endpoint again before relying on the table. If `second_factors`
stops being empty, the client has to handle `needs_second_factor` (below).

### Which sign-in methods a third-party client can use

- **Email code.** Works for every account with a verified email, including
  accounts made with GitHub or Google. This is the one to build.
- **Password.** Works if the account has one. Accounts made through GitHub or
  Google usually don't.
- **GitHub, Google, Microsoft, Apple (web OAuth).** No. Clerk only redirects
  to URLs on T3's allowlist, and our app's URL isn't on it.
- **Native Sign in with Apple (`oauth_token_apple`).** No. Clerk checks that
  Apple's identity token was issued for T3's own app ID.
- **Passkeys.** Not tried. A passkey is tied to T3's domain, and Apple
  platforms only let an app use it if T3's domain names that app. Windows
  might allow it through the system WebAuthn API.
- **Google One Tap, tickets.** No. One Tap needs T3's Google client ID on a
  T3 origin, and only T3's backend can mint tickets.

## How FAPI calls work

Base URL `https://clerk.t3.codes/v1`. Every request:

- adds `_is_native=1` to the query string,
- sends `Authorization: <client JWT>` (the raw value, no `Bearer`), or nothing
  on the very first call,
- sends POST bodies as `application/x-www-form-urlencoded`.

Every response may carry a new `Authorization` header. That's the client JWT,
and Clerk rotates it. **Save it to disk every time it changes**, before you do
anything else with the response. Lose it and the user is signed out. It's the
long-lived credential, so keep it in the platform keychain (Keychain on Apple,
Credential Manager on Windows), not a plain file.

Successful bodies look like `{"response": <object>, "client": <client>}`.
Errors are `{"errors": [{"code", "message", "long_message", "meta"}]}` with a
4xx status. Show `long_message` to the user.

A browser can't make these calls: Clerk rejects any `Origin` other than T3's.
That's why the PoC routes them through `server.mjs`, which drops the header.
A native Rust client sends no `Origin`, so it calls Clerk directly.

## Sign-in

```
1. POST /client/sign_ins                     identifier=<email>
   -> response: sign_in {id, status:"needs_first_factor", supported_first_factors[]}

   supported_first_factors entries look like
   {strategy:"email_code", email_address_id:"idn_...", safe_identifier:"k***@..."}
   {strategy:"password"}

2a. Email code:
   POST /client/sign_ins/{id}/prepare_first_factor
        strategy=email_code&email_address_id=<from step 1>
   (Clerk emails a 6-digit code)
   POST /client/sign_ins/{id}/attempt_first_factor
        strategy=email_code&code=<code>

2b. Password:
   POST /client/sign_ins/{id}/attempt_first_factor
        strategy=password&password=<password>

   -> response: sign_in {status:"complete", created_session_id:"sess_..."}
```

Drive the UI off `status`:

| `status` | Show |
| --- | --- |
| `needs_identifier` | Email box |
| `needs_first_factor` | Code box (and a "use password" link if `password` is in `supported_first_factors`) |
| `needs_second_factor` | Can't happen on T3 today. If it does: `POST .../prepare_second_factor` and `.../attempt_second_factor` with a strategy from `supported_second_factors` (`totp`, `phone_code`, `backup_code`). |
| `needs_new_password` | Send the user to the official app |
| `complete` | Signed in. The session id is `created_session_id`. |

Errors you'll see: `form_code_incorrect`, `verification_expired` (send a new
code), `form_password_incorrect`, `form_identifier_not_found` (no account:
send them to the official app to sign up), and 429 rate limiting.

## Getting a relay token

```
POST /client/sessions/{session_id}/tokens/t3-relay        (empty body)
-> {"jwt": "<session JWT, audience t3-code-relay>"}
```

The JWT lives about 60 seconds (read `exp` from its payload). Cache it and
mint a new one when fewer than 10 seconds remain. Only the relay calls need
it: `GET /v1/environments`, the link endpoints, and the
`/v1/client/dpop-token` exchange. After that exchange the relay's own
30-minute DPoP token takes over, so a busy client asks Clerk for a JWT about
once every half hour.

To find the session id after a restart, `GET /client` and use
`response.last_active_session_id`. If it's null, or the token call returns
401 or 404, the Clerk session is gone (expired or ended elsewhere). Show the
sign-in screen again. T3 doesn't publish its session lifetime, so don't
assume one.

## Sign-out

```
POST /client/sessions/{session_id}/end
```

Then delete the client JWT from the keychain. Do the rest of the sign-out from
[t3-connect.md](t3-connect.md#sessions-sockets-and-sign-out) too: revoke the
environment sessions and drop the DPoP key.

## Rust sketch

`reqwest` + `serde`. About 200 lines with real error types.

```rust
const FAPI: &str = "https://clerk.t3.codes/v1";

pub struct Clerk {
    http: reqwest::Client,
    client_jwt: Option<String>,        // loaded from / saved to the keychain
    relay_jwt: Option<(String, i64)>,  // (jwt, exp)
}

impl Clerk {
    async fn fapi(&mut self, method: Method, path: &str, form: &[(&str, &str)]) -> Result<Value> {
        let mut req = self.http.request(method, format!("{FAPI}{path}?_is_native=1"));
        if let Some(jwt) = &self.client_jwt {
            req = req.header(AUTHORIZATION, jwt);
        }
        let res = req.form(form).send().await?;
        if let Some(new) = res.headers().get(AUTHORIZATION) {
            self.client_jwt = Some(new.to_str()?.to_owned());
            keychain::save("clerk.client", self.client_jwt.as_deref().unwrap())?;
        }
        let status = res.status();
        let body: Value = res.json().await?;
        if !status.is_success() {
            return Err(ClerkError::from_body(status, body));
        }
        Ok(body)
    }

    pub async fn start(&mut self, email: &str) -> Result<SignIn>;          // step 1
    pub async fn send_code(&mut self, s: &SignIn) -> Result<()>;           // step 2a, prepare
    pub async fn attempt(&mut self, s: &SignIn, f: Factor) -> Result<SignIn>; // code or password
    pub async fn relay_token(&mut self) -> Result<String>;                 // cached ~60 s
    pub async fn sign_out(&mut self) -> Result<()>;
}
```

## Where this goes on each platform

- **Windows and desktop:** the Rust client runs this flow itself.
- **iPhone:** same Rust code through UniFFI, or Clerk's iOS SDK. The SDK makes
  the same calls and has the same limits (no OAuth), so it buys little.
- **Watch:** no Clerk at all. The phone or desktop app, already signed in,
  mints a pairing credential for the watch (`POST /api/auth/pairing-token`,
  see [t3-connect.md](t3-connect.md#sessions-sockets-and-sign-out)).
  The watch gets a 30-day bearer session and never sees Clerk or the relay.

## The better long-term fix

The `t3 connect` CLI signs in with Clerk OAuth device codes (client
`hzxSgY2cH10sDU2r`): the user enters a short code in any browser and can use
GitHub, Google or anything else. That would suit a watch or Windows app
well. The relay already accepts those OAuth tokens for `GET /v1/environments`,
but `POST /v1/client/dpop-token` doesn't
(`infra/relay/src/http/Api.ts`, `verifyClerkBearerToken` with no OAuth
fallback). Asking T3 to accept them there, or to give third-party clients their own
OAuth client, would remove every limit in this doc.
