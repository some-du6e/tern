// T3 Connect client: Clerk sign-in, DPoP, relay, environment auth, RPC socket.
// See docs/t3-connect.md for the protocol this follows.

export const RELAY_URL = "https://relay.t3.codes";
const CLERK_TEMPLATE = "t3-relay";
const PROTOCOL = "2";

const enc = new TextEncoder();
const b64url = (bytes) =>
  btoa(String.fromCharCode(...new Uint8Array(bytes)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
const b64urlJson = (value) => b64url(enc.encode(JSON.stringify(value)));
const sha256 = async (text) => b64url(await crypto.subtle.digest("SHA-256", enc.encode(text)));
const form = (fields) => new URLSearchParams(fields).toString();
const jwtExp = (jwt) => JSON.parse(atob(jwt.split(".")[1].replace(/-/g, "+").replace(/_/g, "/"))).exp;

export class T3Error extends Error {
  constructor(message, detail) {
    super(message);
    this.detail = detail;
  }
}

async function readJson(response, what) {
  const text = await response.text();
  let body;
  try {
    body = text ? JSON.parse(text) : {};
  } catch {
    body = { raw: text };
  }
  if (!response.ok) {
    const reason =
      body.errors?.[0]?.long_message ??
      body.errors?.[0]?.message ??
      [body._tag, body.reason, body.dpopFailureReason].filter(Boolean).join(" ") ??
      "";
    throw new T3Error(`${what} failed (${response.status}) ${reason}`.trim(), body);
  }
  return body;
}

// ---------- Clerk (native Frontend API, through the local proxy) ----------

export class Clerk {
  #clientJwt = localStorage.getItem("clerk.client") ?? "";
  #token = null;

  get hasClient() {
    return this.#clientJwt !== "";
  }

  async #fapi(path, fields) {
    const response = await fetch(`/clerk/v1${path}${path.includes("?") ? "&" : "?"}_is_native=1`, {
      method: fields ? "POST" : "GET",
      headers: {
        authorization: this.#clientJwt,
        "content-type": "application/x-www-form-urlencoded",
      },
      ...(fields ? { body: form(fields) } : {}),
    });
    const rotated = response.headers.get("authorization");
    if (rotated) {
      this.#clientJwt = rotated;
      localStorage.setItem("clerk.client", rotated);
    }
    return readJson(response, `Clerk ${path}`);
  }

  async startSignIn(email) {
    const { response } = await this.#fapi("/client/sign_ins", { identifier: email });
    const factors = response.supported_first_factors ?? [];
    const emailCode = factors.find((f) => f.strategy === "email_code");
    return {
      id: response.id,
      hasPassword: factors.some((f) => f.strategy === "password"),
      emailAddressId: emailCode?.email_address_id ?? null,
    };
  }

  async sendEmailCode(signIn) {
    if (!signIn.emailAddressId) throw new T3Error("This account has no email-code sign-in.");
    await this.#fapi(`/client/sign_ins/${signIn.id}/prepare_first_factor`, {
      strategy: "email_code",
      email_address_id: signIn.emailAddressId,
    });
  }

  async finishSignIn(signIn, fields) {
    const { response } = await this.#fapi(`/client/sign_ins/${signIn.id}/attempt_first_factor`, fields);
    if (response.status !== "complete") {
      throw new T3Error(`Sign-in needs another step (${response.status}); this PoC only does one factor.`);
    }
  }

  async #sessionId() {
    const { response } = await this.#fapi("/client");
    const id = response?.last_active_session_id ?? response?.sessions?.[0]?.id;
    if (!id) throw new T3Error("Not signed in to T3 Connect.");
    return id;
  }

  /** A Clerk session JWT with the relay audience. Lives about a minute. */
  async relayToken() {
    if (this.#token && jwtExp(this.#token) - Date.now() / 1000 > 10) return this.#token;
    const sid = await this.#sessionId();
    const { jwt } = await this.#fapi(`/client/sessions/${sid}/tokens/${CLERK_TEMPLATE}`, {});
    this.#token = jwt;
    return jwt;
  }

  signOut() {
    this.#clientJwt = "";
    this.#token = null;
    localStorage.removeItem("clerk.client");
  }
}

// ---------- DPoP key, kept in IndexedDB as a non-extractable CryptoKey ----------

function idb(mode, fn) {
  return new Promise((resolve, reject) => {
    const open = indexedDB.open("tern", 1);
    open.onupgradeneeded = () => open.result.createObjectStore("keys");
    open.onerror = () => reject(open.error);
    open.onsuccess = () => {
      const tx = open.result.transaction("keys", mode);
      const request = fn(tx.objectStore("keys"));
      tx.oncomplete = () => resolve(request.result);
      tx.onerror = () => reject(tx.error);
    };
  });
}

export class DpopKey {
  static async load() {
    let pair = await idb("readonly", (store) => store.get("dpop"));
    if (!pair) {
      pair = await crypto.subtle.generateKey({ name: "ECDSA", namedCurve: "P-256" }, false, ["sign"]);
      await idb("readwrite", (store) => store.put(pair, "dpop"));
    }
    const { x, y } = await crypto.subtle.exportKey("jwk", pair.publicKey);
    const jwk = { kty: "EC", crv: "P-256", x, y };
    // RFC 7638: members in lexicographic order, no whitespace.
    const thumbprint = await sha256(`{"crv":"P-256","kty":"EC","x":"${x}","y":"${y}"}`);
    return new DpopKey(pair.privateKey, jwk, thumbprint);
  }

  constructor(privateKey, jwk, thumbprint) {
    this.privateKey = privateKey;
    this.jwk = jwk;
    this.thumbprint = thumbprint;
  }

  /** A single-use proof for one request. `boundToken` fills `ath`. */
  async proof(method, url, boundToken) {
    const htu = new URL(url);
    htu.search = "";
    htu.hash = "";
    const payload = {
      htm: method,
      htu: htu.toString(),
      jti: crypto.randomUUID(),
      iat: Math.floor(Date.now() / 1000),
      ...(boundToken ? { ath: await sha256(boundToken) } : {}),
    };
    const input = `${b64urlJson({ typ: "dpop+jwt", alg: "ES256", jwk: this.jwk })}.${b64urlJson(payload)}`;
    const signature = await crypto.subtle.sign({ name: "ECDSA", hash: "SHA-256" }, this.privateKey, enc.encode(input));
    return `${input}.${b64url(signature)}`;
  }
}

// ---------- Relay ----------

export class Relay {
  #access = null;

  constructor(clerk, key) {
    this.clerk = clerk;
    this.key = key;
  }

  async #accessToken() {
    if (this.#access && this.#access.expiresAt - Date.now() > 60_000) return this.#access.token;
    const url = `${RELAY_URL}/v1/client/dpop-token`;
    const body = await readJson(
      await fetch(url, {
        method: "POST",
        headers: {
          dpop: await this.key.proof("POST", url),
          "content-type": "application/x-www-form-urlencoded",
        },
        body: form({
          grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
          subject_token: await this.clerk.relayToken(),
          subject_token_type: "urn:ietf:params:oauth:token-type:jwt",
          requested_token_type: "urn:ietf:params:oauth:token-type:access_token",
          resource: RELAY_URL,
          scope: "environment:connect environment:status",
          client_id: "t3-web",
        }),
      }),
      "Relay token exchange",
    );
    this.#access = { token: body.access_token, expiresAt: Date.now() + body.expires_in * 1000 };
    return body.access_token;
  }

  async #dpopPost(path, json) {
    const url = `${RELAY_URL}${path}`;
    const token = await this.#accessToken();
    return readJson(
      await fetch(url, {
        method: "POST",
        headers: {
          authorization: `DPoP ${token}`,
          dpop: await this.key.proof("POST", url, token),
          ...(json ? { "content-type": "application/json" } : {}),
        },
        ...(json ? { body: JSON.stringify(json) } : {}),
      }),
      `Relay ${path}`,
    );
  }

  async listEnvironments() {
    const body = await readJson(
      await fetch(`${RELAY_URL}/v1/environments`, {
        headers: { authorization: `Bearer ${await this.clerk.relayToken()}` },
      }),
      "List environments",
    );
    return body.environments;
  }

  status(environmentId) {
    return this.#dpopPost(`/v1/environments/${encodeURIComponent(environmentId)}/status`);
  }

  connect(environmentId) {
    return this.#dpopPost(`/v1/environments/${encodeURIComponent(environmentId)}/connect`, {
      clientProofKeyThumbprint: this.key.thumbprint,
    });
  }
}

// ---------- Environment session, cached per environment ----------

const sessionKey = (environmentId) => `env.${environmentId}`;

export function cachedSession(environmentId) {
  const raw = localStorage.getItem(sessionKey(environmentId));
  const session = raw ? JSON.parse(raw) : null;
  return session && session.expiresAt - Date.now() > 120_000 ? session : null;
}

export function forgetSession(environmentId) {
  localStorage.removeItem(sessionKey(environmentId));
}

/** Relay connect, then trade the bootstrap credential with the environment. */
export async function openSession(relay, environmentId) {
  const connected = await relay.connect(environmentId);
  const { httpBaseUrl, wsBaseUrl } = connected.endpoint;
  const url = new URL("/oauth/token", httpBaseUrl).toString();
  const body = await readJson(
    await fetch(url, {
      method: "POST",
      headers: {
        dpop: await relay.key.proof("POST", url),
        "content-type": "application/x-www-form-urlencoded",
      },
      body: form({
        grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
        subject_token: connected.credential,
        subject_token_type: "urn:t3:params:oauth:token-type:environment-bootstrap",
        requested_token_type: "urn:ietf:params:oauth:token-type:access_token",
        client_label: "tern poc",
        client_device_type: "desktop",
      }),
    }),
    "Environment token exchange",
  );
  const session = {
    environmentId,
    httpBaseUrl,
    wsBaseUrl,
    accessToken: body.access_token,
    expiresAt: Date.now() + body.expires_in * 1000,
  };
  localStorage.setItem(sessionKey(environmentId), JSON.stringify(session));
  return session;
}

async function socketUrl(key, session) {
  const mint = async (path) => {
    const url = new URL(path, session.httpBaseUrl).toString();
    return fetch(url, {
      method: "POST",
      headers: {
        authorization: `DPoP ${session.accessToken}`,
        dpop: await key.proof("POST", url, session.accessToken),
        "x-t3-orchestration-protocol": PROTOCOL,
      },
    });
  };
  // Servers older than ticket audiences only have the bare route.
  let response = await mint("/api/auth/websocket-ticket/rpc");
  if (response.status === 404) response = await mint("/api/auth/websocket-ticket");
  const { ticket } = await readJson(response, "WebSocket ticket");
  const ws = new URL(session.wsBaseUrl);
  ws.pathname = ws.pathname === "" || ws.pathname === "/" ? "/ws" : ws.pathname;
  // The upgrade proof names the socket as an HTTP URL and binds the ticket.
  const proofUrl = `${ws.protocol === "wss:" ? "https:" : "http:"}//${ws.host}${ws.pathname}`;
  ws.searchParams.set("orchestrationProtocol", PROTOCOL);
  ws.searchParams.set("wsTicket", ticket);
  ws.searchParams.set("dpop", await key.proof("GET", proofUrl, ticket));
  ws.searchParams.set("clientSurface", "web");
  ws.searchParams.set("connectionMethod", "relay");
  return ws.toString();
}

// ---------- Effect RPC over WebSocket (JSON serialization) ----------

export class Rpc {
  #ws;
  #nextId = 0;
  #pending = new Map();
  #ping;
  #seen = 0;
  #closed = false;
  onclose = () => {};

  static async open(key, session) {
    const rpc = new Rpc();
    rpc.session = session;
    rpc.key = key;
    const url = await socketUrl(key, session);
    await new Promise((resolve, reject) => {
      const ws = new WebSocket(url);
      const timer = setTimeout(() => reject(new T3Error("WebSocket open timed out")), 15_000);
      ws.onopen = () => {
        clearTimeout(timer);
        resolve();
      };
      ws.onerror = () => reject(new T3Error("WebSocket failed to open"));
      rpc.#ws = ws;
    });
    rpc.#seen = Date.now();
    rpc.#ws.onmessage = (event) => {
      rpc.#seen = Date.now();
      rpc.#receive(event.data);
    };
    rpc.#ws.onclose = (event) => rpc.#shutdown(event);
    rpc.#ping = setInterval(() => {
      // A phone that slept or changed networks leaves a socket that never
      // closes. Pongs stop, so give up on it and let the app reconnect.
      if (Date.now() - rpc.#seen > 15_000) return rpc.#shutdown({ code: 4000, reason: "no pong" });
      rpc.#send({ _tag: "Ping" });
    }, 5_000);
    return rpc;
  }

  #shutdown(event) {
    if (this.#closed) return;
    this.#closed = true;
    clearInterval(this.#ping);
    this.#ws.onclose = this.#ws.onmessage = null;
    if (this.#ws.readyState <= WebSocket.OPEN) this.#ws.close();
    // onclose first, so the app knows it's reconnecting before streams end.
    this.onclose(event);
    for (const pending of this.#pending.values()) pending.fail(new T3Error("Socket closed"));
    this.#pending.clear();
  }

  #send(message) {
    if (this.#ws.readyState === WebSocket.OPEN) this.#ws.send(JSON.stringify(message));
  }

  #receive(data) {
    const parsed = JSON.parse(data);
    for (const message of Array.isArray(parsed) ? parsed : [parsed]) {
      if (message._tag === "Pong") continue;
      if (message._tag === "Defect") {
        console.error("socket defect", message.defect);
        this.close();
        continue;
      }
      const pending = this.#pending.get(String(message.requestId));
      if (!pending) continue;
      if (message._tag === "Chunk") {
        // The server sends nothing more on this stream until we Ack.
        this.#send({ _tag: "Ack", requestId: message.requestId });
        pending.chunk(message.values);
      } else if (message._tag === "Exit") {
        this.#pending.delete(String(message.requestId));
        const exit = message.exit;
        if (exit._tag === "Success") pending.done(exit.value);
        else {
          const cause = exit.cause?.find((c) => c._tag === "Fail") ?? exit.cause?.[0];
          const error = cause?.error ?? cause?.defect ?? cause;
          pending.fail(new T3Error(error?.message ?? error?._tag ?? "RPC failed", error));
        }
      }
    }
  }

  #request(tag, payload, handlers) {
    const id = this.#nextId++;
    this.#pending.set(String(id), handlers);
    this.#send({ _tag: "Request", id, tag, payload, headers: [] });
    return id;
  }

  call(tag, payload = {}) {
    return new Promise((done, fail) => this.#request(tag, payload, { chunk() {}, done, fail }));
  }

  /** Calls onValue for each streamed item. Returns a function that cancels. */
  stream(tag, payload, onValue, onEnd = () => {}) {
    const id = this.#request(tag, payload, {
      chunk: (values) => values.forEach(onValue),
      done: () => onEnd(null),
      fail: (error) => onEnd(error),
    });
    return () => {
      if (this.#pending.delete(String(id))) this.#send({ _tag: "Interrupt", requestId: id, interruptors: [] });
    };
  }

  dispatch(command) {
    return this.call("orchestration.dispatchCommand", { commandId: crypto.randomUUID(), ...command });
  }

  /** One page of older turn items, from a bounded snapshot's historyCursor. */
  async historyPage(threadId, cursor) {
    const url = new URL(`/api/orchestration/threads/${encodeURIComponent(threadId)}/history`, this.session.httpBaseUrl);
    url.searchParams.set("cursor", cursor);
    const token = this.session.accessToken;
    return readJson(
      await fetch(url, {
        headers: {
          authorization: `DPoP ${token}`,
          dpop: await this.key.proof("GET", url.toString(), token),
          "x-t3-orchestration-protocol": PROTOCOL,
        },
      }),
      "Thread history",
    );
  }

  /** Uploads an image to the environment and returns it as a message attachment. */
  async uploadImage(file) {
    const meta = { type: "image", name: file.name || "image.png", mimeType: file.type, sizeBytes: file.size };
    const { attachmentId, relativeUrl } = await this.call("attachments.createUploadUrl", meta);
    // The URL is signed, so the upload itself carries no auth.
    const response = await fetch(new URL(relativeUrl, this.session.httpBaseUrl), {
      method: "POST",
      headers: { "content-type": file.type },
      body: file,
    });
    if (!response.ok) throw new T3Error(`Image upload failed (${response.status})`);
    return { ...meta, id: attachmentId };
  }

  close() {
    if (this.#ws) this.#shutdown({ code: 1000, reason: "closed" });
  }
}
