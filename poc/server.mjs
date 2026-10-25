// Serves the PoC and proxies Clerk's Frontend API. Clerk rejects browser
// requests from origins other than t3.codes, so /clerk/* is forwarded here
// without Origin, Referer or cookies. Everything else (relay, environment,
// WebSocket) goes straight from the browser.
import { createServer as createHttp } from "node:http";
import { createServer as createHttps } from "node:https";
import { readFileSync } from "node:fs";
import { readFile } from "node:fs/promises";
import { extname, join } from "node:path";

const HOST = process.env.HOST ?? "127.0.0.1";
const PORT = Number(process.env.PORT ?? 4777);
const CLERK_FAPI = process.env.CLERK_FAPI ?? "https://clerk.t3.codes";
const ROOT = new URL("./public/", import.meta.url).pathname;
// WebCrypto only works in a secure context, so anything off localhost needs TLS.
const TLS = process.env.TLS_CERT && {
  cert: readFileSync(process.env.TLS_CERT),
  key: readFileSync(process.env.TLS_KEY),
};
const TYPES = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css" };

async function proxyClerk(req, res, url) {
  const body = req.method === "GET" ? undefined : await readBody(req);
  const upstream = await fetch(`${CLERK_FAPI}${url.pathname.slice("/clerk".length)}${url.search}`, {
    method: req.method,
    headers: {
      "content-type": req.headers["content-type"] ?? "application/x-www-form-urlencoded",
      ...(req.headers.authorization ? { authorization: req.headers.authorization } : {}),
    },
    body,
  });
  const headers = { "content-type": upstream.headers.get("content-type") ?? "application/json" };
  const clientJwt = upstream.headers.get("authorization");
  if (clientJwt) headers["authorization"] = clientJwt;
  res.writeHead(upstream.status, headers);
  res.end(Buffer.from(await upstream.arrayBuffer()));
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks)));
    req.on("error", reject);
  });
}

async function serveStatic(res, pathname) {
  const file = pathname === "/" ? "index.html" : pathname.slice(1);
  if (file.includes("..")) return void res.writeHead(400).end();
  try {
    const data = await readFile(join(ROOT, file));
    // no-cache revalidates every load, so an edit shows up without a hard refresh.
    res.writeHead(200, { "content-type": TYPES[extname(file)] ?? "application/octet-stream", "cache-control": "no-cache" });
    res.end(data);
  } catch {
    res.writeHead(404).end("not found");
  }
}

(TLS ? (handler) => createHttps(TLS, handler) : createHttp)(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  try {
    if (url.pathname.startsWith("/clerk/v1/")) await proxyClerk(req, res, url);
    else await serveStatic(res, url.pathname);
  } catch (error) {
    console.error(error);
    if (!res.headersSent) res.writeHead(502, { "content-type": "application/json" });
    res.end(JSON.stringify({ errors: [{ message: String(error) }] }));
  }
}).listen(PORT, HOST, () => console.log(`tern poc on ${TLS ? "https" : "http"}://${HOST}:${PORT}`));
