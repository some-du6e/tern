import { Clerk, DpopKey, Relay, Rpc, cachedSession, forgetSession, openSession } from "./t3.js";

const $ = (id) => document.getElementById(id);
const esc = (s) => String(s ?? "").replace(/[&<>"]/g, (c) => `&#${c.charCodeAt(0)};`);
const show = (el, on) => el.classList.toggle("hidden", !on);
const ACTIVE = new Set(["preparing", "starting", "running", "waiting"]);

const clerk = new Clerk();
let relay;

const state = {
  envs: [],
  status: new Map(), // environmentId -> "online" | "offline" | "busy"
  envId: localStorage.getItem("lastEnv"),
  rpc: null,
  projects: new Map(),
  threads: new Map(),
  threadId: null,
  shownThread: null,
  cancelThread: null,
  retry: 0,
  search: null, // {query, matches} from orchestration.searchThreads
  archived: null, // {threads, projects} while the archive is shown
  openSettled: new Set(), // project ids whose settled threads are expanded
  draft: null, // {projectId} while composing a new thread
  choices: new Map(), // thread id or "draft" -> {modelSelection, runtimeMode, interactionMode}
};

function fail(error) {
  console.error(error);
  $("err").textContent = error?.message ?? String(error);
  show($("err"), true);
}
$("err").onclick = () => show($("err"), false);

function net(text) {
  $("net").textContent = text;
}

// ---------- sign-in ----------

let signIn = null;
let usePassword = false;

function showSignIn() {
  show($("signin"), true);
  show($("app"), false);
  show($("signout"), false);
  $("email").focus();
}

$("usepw").onclick = () => {
  usePassword = true;
  show($("code"), false);
  show($("password"), true);
  show($("usepw"), false);
  $("password").focus();
};

$("signin").onsubmit = async (event) => {
  event.preventDefault();
  const go = $("go");
  go.disabled = true;
  try {
    if (!signIn) {
      signIn = await clerk.startSignIn($("email").value.trim());
      $("email").disabled = true;
      show($("usepw"), signIn.hasPassword);
      if (signIn.emailAddressId) {
        await clerk.sendEmailCode(signIn);
        show($("code"), true);
        $("code").focus();
      } else {
        $("usepw").click();
      }
      go.textContent = "Sign in";
    } else {
      await clerk.finishSignIn(
        signIn,
        usePassword
          ? { strategy: "password", password: $("password").value }
          : { strategy: "email_code", code: $("code").value.trim() },
      );
      show($("signin"), false);
      await start();
    }
  } catch (error) {
    fail(error);
  } finally {
    go.disabled = false;
  }
};

$("signout").onclick = () => {
  clerk.signOut();
  state.rpc?.close();
  location.reload();
};

// ---------- environments ----------

function renderEnvs() {
  $("envs").innerHTML = state.envs.length
    ? state.envs
        .map(
          (env) => `<div class="env${env.environmentId === state.envId ? " on" : ""}" data-id="${esc(env.environmentId)}">
            <span class="dot ${state.status.get(env.environmentId) ?? ""}"></span>${esc(env.label)}</div>`,
        )
        .join("")
    : `<span class="dim" style="padding:6px 8px">No linked environments. Run <code>t3 connect</code> on a host.</span>`;
}

$("envs").onclick = (event) => {
  const id = event.target.closest(".env")?.dataset.id;
  if (id && id !== state.envId) selectEnv(id);
};

async function refreshStatuses() {
  await Promise.all(
    state.envs.map(async (env) => {
      try {
        const result = await relay.status(env.environmentId);
        if (state.status.get(env.environmentId) !== "busy") state.status.set(env.environmentId, result.status);
      } catch {
        state.status.set(env.environmentId, "offline");
      }
    }),
  );
  renderEnvs();
}

async function selectEnv(id) {
  state.envId = id;
  localStorage.setItem("lastEnv", id);
  state.retry = 0;
  closeThread();
  state.projects.clear();
  state.threads.clear();
  renderThreads();
  renderEnvs();
  await connectEnv(id);
}

async function connectEnv(id) {
  const old = state.rpc;
  state.rpc = null;
  if (old) {
    old.onclose = () => {};
    old.close();
  }
  const started = performance.now();
  net("connecting…");
  state.status.set(id, "busy");
  renderEnvs();
  try {
    const cached = cachedSession(id);
    let rpc;
    try {
      // A cached session skips the relay entirely: ticket, then socket.
      rpc = await Rpc.open(relay.key, cached ?? (await openSession(relay, id)));
    } catch (error) {
      if (!cached) throw error;
      forgetSession(id);
      rpc = await Rpc.open(relay.key, await openSession(relay, id));
    }
    if (state.envId !== id) return rpc.close();
    state.rpc = rpc;
    state.config = null;
    state.retry = 0;
    state.status.set(id, "online");
    renderEnvs();
    rpc.onclose = () => {
      if (state.envId !== id || state.rpc !== rpc) return;
      state.rpc = null;
      state.status.set(id, "busy");
      renderEnvs();
      const delay = Math.min(30_000, 1000 * 2 ** state.retry++) * (0.5 + Math.random());
      net(`reconnecting in ${Math.round(delay / 1000)}s`);
      state.reconnect = setTimeout(() => state.envId === id && !state.rpc && connectEnv(id), delay);
    };
    // Uploads and history use the session token over HTTP, so swap in a fresh
    // session before this one expires. The thread view survives the swap.
    const renewIn = rpc.session.expiresAt - Date.now() - 180_000;
    if (renewIn > 0) {
      setTimeout(() => {
        if (state.rpc !== rpc) return;
        forgetSession(id);
        connectEnv(id);
      }, renewIn);
    }
    subscribeShell(rpc, started);
    if (state.threadId) openThread(state.threadId);
  } catch (error) {
    state.status.set(id, "offline");
    renderEnvs();
    net("offline");
    fail(error);
  }
}

// A backgrounded phone tab loses its socket. Reconnect as soon as it's back
// instead of waiting out the backoff.
document.addEventListener("visibilitychange", () => {
  if (document.hidden || state.rpc || !state.envId || !relay) return;
  clearTimeout(state.reconnect);
  state.retry = 0;
  connectEnv(state.envId);
});

// ---------- threads ----------

function subscribeShell(rpc, started) {
  rpc.stream("orchestration.subscribeShell", { requestCompletionMarker: true }, (item) => {
    switch (item.kind) {
      case "snapshot":
        // A snapshot with resolvedRepositoryIdentityRoots only refreshes the
        // projects whose repo identity changed and has no threads. Merge it.
        if (item.resolvedRepositoryIdentityRoots !== undefined) {
          for (const p of item.snapshot.projects) state.projects.set(p.id, p);
          break;
        }
        state.projects = new Map(item.snapshot.projects.map((p) => [p.id, p]));
        state.threads = new Map(item.snapshot.threads.map((t) => [t.id, t]));
        net(`connected in ${Math.round(performance.now() - started)} ms`);
        break;
      case "project.updated":
        state.projects.set(item.project.id, item.project);
        break;
      case "project.removed":
        state.projects.delete(item.projectId);
        break;
      case "thread.updated":
        if (item.location === "active" && !item.thread.deletedAt) state.threads.set(item.thread.id, item.thread);
        else state.threads.delete(item.thread.id);
        break;
      case "thread.removed":
        state.threads.delete(item.threadId);
        break;
      default:
        return;
    }
    scheduleThreadsRender();
    if (state.threadId) renderThreadHeader();
  });
}

let threadsFrame = 0;
function scheduleThreadsRender() {
  threadsFrame ||= requestAnimationFrame(() => {
    threadsFrame = 0;
    renderThreads();
  });
}

// The open thread, which may be an archived one outside the live shell.
const current = () => state.threads.get(state.threadId) ?? state.archived?.threads.find((t) => t.id === state.threadId);

const threadTime = (t) => Date.parse(t.latestUserMessageAt ?? t.updatedAt ?? t.createdAt) || 0;
// Delegated tasks run in child threads that T3 never lists (most threads on a
// busy server). The server pins them, so they'd also flood "Pinned".
const isSubagent = (t) => t.lineage?.relationshipToParent === "subagent";
// A subagent is represented by its nearest listed ancestor, as in T3.
function listedAncestor(id) {
  const seen = new Set();
  let t = state.threads.get(id);
  while (t && isSubagent(t) && t.lineage.parentThreadId && !seen.has(t.id)) {
    seen.add(t.id);
    t = state.threads.get(t.lineage.parentThreadId);
  }
  return t && !isSubagent(t) ? t : null;
}
const later = (iso) => Boolean(iso) && Date.parse(iso) > Date.now();
// Like T3's sidebar: settled and snoozed threads fold under each project.
const isResting = (t) => Boolean(t.settledAt) || later(t.snoozedUntil);
// Unread once a run finished after the user last looked at the thread.
const isUnread = (t) =>
  t.id !== state.threadId && Boolean(t.latestRunCompletedAt) && (!t.lastVisitedAt || t.lastVisitedAt < t.latestRunCompletedAt);

function threadRow(t, extra = "") {
  const dot = t.pendingRuntimeRequest ? "busy" : ACTIVE.has(t.activityRunStatus) || t.activeRunId ? "online" : "";
  const cls = `thread${t.id === state.threadId ? " on" : ""}${isUnread(t) ? " unread" : ""}`;
  const note = later(t.snoozedUntil) ? "snoozed" : t.settledAt ? "" : t.lastError ? esc(t.lastError).slice(0, 120) : "";
  return `<div class="${cls}" data-id="${esc(t.id)}">
    <span class="dot ${dot}"></span><span class="t">${t.pinnedAt ? '<span class="pin">📌</span> ' : ""}${esc(t.title || "Untitled")}</span>
    ${extra || (note ? `<span class="p">${note}</span>` : "")}</div>`;
}

function renderThreads() {
  const filter = $("filter").value.trim().toLowerCase();
  const source = (state.archived ? state.archived.threads : [...state.threads.values()]).filter((t) => !isSubagent(t));
  const projectTitle = (id) => esc((state.archived?.projects.get(id) ?? state.projects.get(id))?.title ?? "No project");
  const matches = (t) => !filter || (t.title ?? "").toLowerCase().includes(filter);
  let html = "";
  if (state.archived) html += `<div class="project"><span>Archived</span></div>`;
  const pinned = state.archived ? [] : source.filter((t) => t.pinnedAt && matches(t));
  if (pinned.length) {
    html += `<div class="project"><span>Pinned</span></div>`;
    html += pinned.sort((a, b) => String(a.pinOrderKey ?? a.pinnedAt).localeCompare(String(b.pinOrderKey ?? b.pinnedAt))).map((t) => threadRow(t)).join("");
  }
  const byProject = new Map();
  for (const t of source) {
    if (!matches(t) || (t.pinnedAt && !state.archived)) continue;
    if (!byProject.has(t.projectId)) byProject.set(t.projectId, []);
    byProject.get(t.projectId).push(t);
  }
  // Projects with no threads still get a header, so you can start one there.
  if (!state.archived && !filter) for (const id of state.projects.keys()) if (!byProject.has(id)) byProject.set(id, []);
  const groups = [...byProject.entries()]
    .map(([projectId, threads]) => [projectId, threads.sort((a, b) => threadTime(b) - threadTime(a))])
    .sort((a, b) => (b[1][0] ? threadTime(b[1][0]) : 0) - (a[1][0] ? threadTime(a[1][0]) : 0));
  for (const [projectId, threads] of groups) {
    const awake = state.archived || filter ? threads : threads.filter((t) => !isResting(t));
    const resting = state.archived || filter ? [] : threads.filter(isResting);
    const open = state.openSettled.has(projectId);
    html += `<div class="project"><span>${projectTitle(projectId)}</span>${
      state.archived || !state.projects.has(projectId) ? "" : `<button data-new="${esc(projectId)}" title="New thread here">+</button>`
    }</div>`;
    html += awake.slice(0, 50).map((t) => threadRow(t)).join("");
    if (resting.length) {
      html += `<div class="more-settled" data-settled="${esc(projectId)}">${open ? "Hide" : "Show"} ${resting.length} settled</div>`;
      if (open) html += resting.slice(0, 100).map((t) => threadRow(t)).join("");
    }
  }
  // Message hits from the server's full-text search, under the title matches.
  if (state.search?.query === filter && state.search.matches.length) {
    html += `<div class="project"><span>In messages</span></div>`;
    const shown = new Set();
    html += state.search.matches
      .map((m) => {
        // A hit inside a subagent's thread points at the thread that started it.
        const t = listedAncestor(m.threadId) ?? state.threads.get(m.threadId) ?? { id: m.threadId, title: "Archived thread" };
        if (shown.has(t.id)) return "";
        shown.add(t.id);
        const snippet = esc(m.snippet).replace(new RegExp(filter.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "gi"), (x) => `<mark>${x}</mark>`);
        return threadRow(t, `<span class="p">${m.source === "user" ? "You: " : ""}${snippet}</span>`).replace('class="thread', 'class="thread match');
      })
      .join("");
  }
  $("threads").innerHTML = html || `<div class="dim" style="padding:12px 8px">${filter ? "No matches." : "No threads."}</div>`;
}

// Title filtering is local and instant; message search asks the server once
// typing pauses. It needs at least two characters.
let searchTimer = 0;
$("filter").oninput = () => {
  scheduleThreadsRender();
  clearTimeout(searchTimer);
  const query = $("filter").value.trim().toLowerCase();
  if (query.length < 2 || !state.rpc) {
    state.search = null;
    return;
  }
  searchTimer = setTimeout(async () => {
    try {
      const result = await state.rpc.call("orchestration.searchThreads", { query: query.slice(0, 200), limit: 30 });
      if ($("filter").value.trim().toLowerCase() !== query) return;
      state.search = { query, matches: result.matches };
      scheduleThreadsRender();
    } catch (error) {
      fail(error);
    }
  }, 250);
};

$("threads").onclick = (event) => {
  const newIn = event.target.closest("[data-new]")?.dataset.new;
  if (newIn) return openDraft(newIn);
  const settled = event.target.closest("[data-settled]")?.dataset.settled;
  if (settled) {
    state.openSettled.has(settled) ? state.openSettled.delete(settled) : state.openSettled.add(settled);
    return renderThreads();
  }
  const id = event.target.closest(".thread")?.dataset.id;
  if (id) openThread(id);
};

$("newthread").onclick = () => {
  const t = current();
  const recent = [...state.threads.values()].sort((a, b) => threadTime(b) - threadTime(a))[0];
  openDraft(t?.projectId ?? recent?.projectId ?? state.projects.keys().next().value);
};

// Archived threads aren't in the live shell; fetch them once per visit.
$("archived").onclick = async () => {
  if (state.archived) {
    state.archived = null;
    $("archived").textContent = "Archived";
    return renderThreads();
  }
  try {
    const snapshot = await state.rpc.call("orchestration.getArchivedShellSnapshot", {});
    state.archived = { threads: snapshot.threads, projects: new Map(snapshot.projects.map((p) => [p.id, p])) };
    $("archived").textContent = "← Active threads";
    renderThreads();
  } catch (error) {
    fail(error);
  }
};

$("back").onclick = () => {
  closeThread();
  renderThreads();
};

// ---------- one thread ----------

// The thread is a list of turn items in `ordinal` order, the same rows the
// official app draws. These carry nothing worth showing on their own.
const HIDDEN = new Set(["checkpoint", "notification", "approval_request", "user_input_request", "run_interrupt_request"]);
const TOOLS = new Set(["command_execution", "file_change", "file_search", "web_search", "dynamic_tool", "subagent"]);

const view = {
  items: new Map(), // row key -> turn item
  rows: new Map(), // row key -> element
  history: { cursor: null, more: false, loading: false },
  full: new Map(), // turn item id -> item from getTurnItem
  runs: new Map(), // run id -> run (status, startedAt, completedAt)
  unfolded: new Set(), // fold keys the user opened
  messages: new Map(), // message id -> message, for queued runs' text
  answers: new Map(), // question id -> {picked: Set, text}, while answering
};
// One row per message even if message.updated beats its turn item here.
const rowKey = (item) => (item.messageId ? `m:${item.messageId}` : item.id);

function putItem(item) {
  if (HIDDEN.has(item.type) && item.type !== "approval_request" && item.type !== "user_input_request") return;
  const key = rowKey(item);
  view.items.set(key, { ...view.items.get(key), ...item });
}

function resetView(text) {
  view.items = new Map();
  view.rows = new Map();
  view.history = { cursor: null, more: false, loading: false };
  view.full = new Map();
  view.runs = new Map();
  view.unfolded = new Set();
  view.messages = new Map();
  view.answers = new Map();
  view.checkpoints = [];
  $("messages").innerHTML = `<div id="empty">${text}</div>`;
}

function closeThread() {
  state.cancelThread?.();
  state.cancelThread = null;
  state.threadId = null;
  state.shownThread = null;
  state.draft = null;
  $("app").classList.remove("thread-open");
  resetView("Pick a thread.");
  for (const id of ["thead", "composer", "approval", "toolbar", "queue", "questions", "planbar", "draftproject"]) show($(id), false);
  closePops();
}

// A new thread is a draft until its first message: then launchThread creates
// it, with the chosen project, model and modes, and starts the run.
function openDraft(projectId) {
  if (!projectId) return fail(new Error("Add a project in T3 first."));
  closeThread();
  state.draft = { projectId };
  state.choices.delete("draft");
  $("app").classList.add("thread-open");
  for (const id of ["thead", "composer", "toolbar", "draftproject"]) show($(id), true);
  show($("stop"), false);
  show($("queuebtn"), false);
  show($("menu"), false);
  $("ttitle").textContent = "New thread in";
  $("tstatus").textContent = "";
  $("draftproject").innerHTML = [...state.projects.values()]
    .sort((a, b) => a.title.localeCompare(b.title))
    .map((p) => `<option value="${esc(p.id)}"${p.id === projectId ? " selected" : ""}>${esc(p.title)}</option>`)
    .join("");
  resetView("Say what to do. The first message starts the thread.");
  renderToolbar();
  scheduleThreadsRender();
  restoreDraft();
  $("input").focus();
}
$("draftproject").onchange = () => {
  // The text you typed moves with the draft to the new project.
  setDraft(draftKey(), "");
  state.draft.projectId = $("draftproject").value;
  saveDraft();
  state.choices.delete("draft");
  renderToolbar();
};

let messagesFrame = 0;
function scheduleMessagesRender() {
  messagesFrame ||= requestAnimationFrame(() => {
    messagesFrame = 0;
    renderMessages();
  });
}

function openThread(id) {
  state.cancelThread?.();
  if (state.draft) closeThread();
  state.threadId = id;
  $("app").classList.add("thread-open");
  for (const el of ["thead", "composer", "toolbar", "menu"]) show($(el), true);
  show($("draftproject"), false);
  if (state.shownThread !== id) {
    state.shownThread = id;
    resetView("Loading…");
  }
  renderThreadHeader();
  scheduleThreadsRender();
  const rpc = state.rpc;
  if (!rpc) return;
  // A bounded snapshot is the recent window plus a cursor for older pages; a
  // long thread's full snapshot can be many MB. Compact mode drops the
  // duplicate turnItems list, so rows come from visibleTurnItems.
  state.cancelThread = rpc.stream(
    "orchestration.subscribeThread",
    { threadId: id, requestCompletionMarker: true, acceptBoundedSnapshot: true, acceptCompactTurnItems: true },
    (item) => {
      if (state.threadId !== id) return;
      if (item.kind === "snapshot") {
        view.items = new Map();
        for (const row of item.projection.visibleTurnItems) putItem(row.item);
        view.runs = new Map(item.projection.runs.map((r) => [r.id, r]));
        view.messages = new Map((item.projection.messages ?? []).map((m) => [m.id, m]));
        view.checkpoints = item.projection.checkpoints ?? [];
        view.history = { cursor: item.historyCursor ?? null, more: item.hasMoreHistory === true, loading: false };
        scheduleMessagesRender();
      } else if (item.kind === "event" && item.event?.type === "turn-item.updated") {
        putItem(item.event.payload);
        scheduleMessagesRender();
      } else if (item.kind === "event" && item.event?.type?.startsWith("run.") && item.event.payload?.id) {
        const r = item.event.payload;
        view.runs.set(r.id, { ...view.runs.get(r.id), ...r });
        scheduleMessagesRender();
      } else if (item.kind === "event" && item.event?.type === "message.updated") {
        // Normally its turn item arrives in the same batch. Keep it in case not.
        const m = item.event.payload;
        view.messages.set(m.id, m);
        if (view.items.has(`m:${m.id}`)) return scheduleMessagesRender();
        const last = Math.max(0, ...[...view.items.values()].map((i) => i.ordinal ?? 0));
        putItem({ ...m, id: `m:${m.id}`, messageId: m.id, type: m.role === "user" ? "user_message" : "assistant_message", ordinal: last + 0.5 });
        scheduleMessagesRender();
      }
    },
    (error) => error && state.rpc === rpc && state.threadId === id && fail(error),
  );
  // Moves T3's read marker, so the thread stops showing as unread everywhere.
  rpc.dispatch({ type: "thread.visit", threadId: id, visitedAt: new Date().toISOString(), opened: true }).catch(() => {});
  renderToolbar();
  restoreDraft();
  $("input").focus();
}

async function loadEarlier() {
  const h = view.history;
  if (!h.more || !h.cursor || h.loading || !state.rpc) return;
  h.loading = true;
  const id = state.threadId;
  renderMessages();
  try {
    const page = await state.rpc.historyPage(id, h.cursor);
    if (state.threadId !== id) return;
    for (const row of page.items) putItem(row.item);
    view.history = { cursor: page.nextCursor, more: page.hasMoreHistory, loading: false };
    const box = $("messages");
    const fromBottom = box.scrollHeight - box.scrollTop;
    renderMessages();
    box.scrollTop = box.scrollHeight - fromBottom;
  } catch (error) {
    h.loading = false;
    renderMessages();
    fail(error);
  }
}

function renderThreadHeader() {
  if (state.draft) return;
  const t = current();
  $("ttitle").textContent = t?.title || "Untitled";
  const running = Boolean(t?.activeRunId);
  $("tstatus").textContent = t?.pendingRuntimeRequest
    ? t.pendingRuntimeRequest.kind === "user_input" ? "has a question" : "needs approval"
    : running ? (t.activityRunStatus ?? "running") : t?.archivedAt ? "archived" : "";
  show($("stop"), running);
  show($("queuebtn"), running);
  if (view.working !== running) {
    view.working = running;
    scheduleMessagesRender();
  }
  renderApproval();
  renderQuestions();
  renderQueue();
  if (!$("modelpop").classList.contains("hidden")) return;
  renderToolbar();
}

// ---------- thread actions ----------

const dispatchFor = (type, extra = {}) => state.rpc.dispatch({ type, threadId: state.threadId, ...extra });

function threadActions(t) {
  const hour = 3600_000;
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  tomorrow.setHours(9, 0, 0, 0);
  const finished = Boolean(t.latestRunCompletedAt) || [...view.runs.values()].some((r) => r.status === "completed");
  if (t.archivedAt) {
    return [
      ["Unarchive", () => dispatchFor("thread.unarchive")],
      ["Delete…", () => confirm(`Delete "${t.title}"? This can't be undone.`) && dispatchFor("thread.delete").then(closeThread), "bad"],
    ];
  }
  return [
    ["Rename…", renameThread],
    ["Regenerate title", () => dispatchFor("thread.metadata.update", { regenerateTitle: true })],
    t.pinnedAt ? ["Unpin", () => dispatchFor("thread.unpin")] : ["Pin", () => dispatchFor("thread.pin")],
    t.settledAt ? ["Unsettle", () => dispatchFor("thread.unsettle", { reason: "user" })] : ["Settle", () => dispatchFor("thread.settle")],
    later(t.snoozedUntil)
      ? ["Unsnooze", () => dispatchFor("thread.unsnooze", { reason: "user" })]
      : ["Snooze 1 hour", () => dispatchFor("thread.snooze", { snoozedUntil: new Date(Date.now() + hour).toISOString() })],
    later(t.snoozedUntil) ? null : ["Snooze until tomorrow 9:00", () => dispatchFor("thread.snooze", { snoozedUntil: tomorrow.toISOString() })],
    // Both need a finished run; the server rejects them before that.
    finished ? ["Mark unread", () => dispatchFor("thread.mark-unread").then(closeThread)] : null,
    finished ? ["Fork", forkThread] : null,
    // Turn counts are checkpoint run ordinals; 0 is the thread's start.
    turns().length ? ["Diff of last turn", () => showDiff(turns().at(-2) ?? 0, turns().at(-1))] : null,
    turns().length ? ["Diff of whole thread", () => showDiff(0, turns().at(-1))] : null,
    ["Archive", () => dispatchFor("thread.archive").then(closeThread)],
    ["Delete…", () => confirm(`Delete "${t.title}"? This can't be undone.`) && dispatchFor("thread.delete").then(closeThread), "bad"],
  ].filter(Boolean);
}

function closePops() {
  for (const id of ["menupop", "modelpop"]) show($(id), false);
}
document.addEventListener("pointerdown", (event) => {
  if (!event.target.closest(".pop, #menu, #model")) closePops();
});

$("menu").onclick = () => {
  const pop = $("menupop");
  if (!pop.classList.contains("hidden")) return closePops();
  const t = current();
  if (!t) return;
  const actions = threadActions(t);
  pop.innerHTML = actions.map(([label, , cls], i) => `<div class="item ${cls ?? ""}" data-i="${i}">${esc(label)}</div>`).join("");
  pop.onclick = async (event) => {
    const i = event.target.closest(".item")?.dataset.i;
    if (i === undefined) return;
    closePops();
    try {
      await actions[i][1]();
    } catch (error) {
      fail(error);
    }
  };
  show(pop, true);
};

const turns = () => [...new Set(view.checkpoints.map((c) => c.appRunOrdinal).filter((n) => Number.isInteger(n)))].sort((a, b) => a - b);

async function showDiff(fromTurnCount, toTurnCount) {
  const { diff } =
    fromTurnCount === 0
      ? await state.rpc.call("orchestration.getFullThreadDiff", { threadId: state.threadId, toTurnCount })
      : await state.rpc.call("orchestration.getTurnDiff", { threadId: state.threadId, fromTurnCount, toTurnCount, ignoreWhitespace: true });
  const lines = String(diff || "No changes.").split("\n");
  $("diffbody").innerHTML = lines
    .map((l) => {
      const cls = l.startsWith("+++") || l.startsWith("---") || l.startsWith("diff ") ? "f" : l.startsWith("+") ? "a" : l.startsWith("-") ? "d" : l.startsWith("@@") ? "h" : "";
      return `<span class="${cls}">${esc(l)}</span>`;
    })
    .join("\n");
  $("difftitle").textContent = fromTurnCount === 0 ? "Whole thread" : `Turn ${toTurnCount}`;
  show($("diffview"), true);
}
$("diffclose").onclick = () => show($("diffview"), false);

async function renameThread() {
  const t = current();
  const title = prompt("Thread title", t?.title ?? "")?.trim();
  if (title && title !== t.title) await dispatchFor("thread.metadata.update", { title });
}
$("ttitle").onclick = () => !state.draft && renameThread().catch(fail);

async function forkThread() {
  const target = crypto.randomUUID();
  await state.rpc.dispatch({
    type: "thread.fork",
    createdBy: "user",
    creationSource: "web",
    sourceThreadId: state.threadId,
    targetThreadId: target,
    sourcePoint: { type: "latest_stable" },
  });
  openThread(target);
}

// ---------- model, plan mode, permissions ----------

// The composer's choices for the open thread or draft. Like T3, they apply
// on the next send: the model rides on message.dispatch, modes are set first.
function choices() {
  const key = state.draft ? "draft" : state.threadId;
  if (!state.choices.has(key)) {
    const t = state.draft ? null : current();
    const project = state.projects.get(state.draft?.projectId ?? t?.projectId);
    const settings = state.config?.settings;
    state.choices.set(key, {
      modelSelection: t?.modelSelection ?? project?.defaultModelSelection ?? settings?.defaultModelSelection ?? null,
      runtimeMode: t?.runtimeMode ?? settings?.defaultRuntimeMode ?? "full-access",
      interactionMode: t?.interactionMode ?? "default",
    });
  }
  return state.choices.get(key);
}

const providerOf = (instanceId) => state.config?.providers?.find((p) => p.instanceId === instanceId);
const modelOf = (selection) => providerOf(selection?.instanceId)?.models.find((m) => m.slug === selection?.model);
const optionValue = (selection, descriptor) =>
  selection?.options?.find((o) => o.id === descriptor.id)?.value ??
  (descriptor.type === "boolean" ? (descriptor.currentValue ?? false) : descriptor.options?.find((o) => o.isDefault)?.id);

async function renderToolbar() {
  if (!state.rpc) return;
  if (!state.config) {
    try {
      state.config = await serverConfig(state.rpc);
    } catch (error) {
      return fail(error);
    }
  }
  const c = choices();
  // A draft opened before the config loaded has no model yet.
  c.modelSelection ??= state.config.settings?.defaultModelSelection ?? null;
  const provider = providerOf(c.modelSelection?.instanceId);
  const model = modelOf(c.modelSelection);
  const effort = model?.capabilities?.optionDescriptors?.find((d) => d.type === "select");
  const effortValue = effort && optionValue(c.modelSelection, effort);
  const effortLabel = effort?.options?.find((o) => o.id === effortValue)?.label ?? effortValue;
  $("model").textContent = model
    ? `${provider.displayName} · ${model.name}${effortLabel ? ` · ${effortLabel}` : ""} ▾`
    : c.modelSelection ? `${c.modelSelection.model} ▾` : "Pick a model ▾";
  show($("plan"), provider?.showInteractionModeToggle !== false);
  $("plan").classList.toggle("on", c.interactionMode === "plan");
  $("runtime").value = c.runtimeMode;
}

$("plan").onclick = () => {
  const c = choices();
  c.interactionMode = c.interactionMode === "plan" ? "default" : "plan";
  renderToolbar();
};
$("runtime").onchange = () => {
  choices().runtimeMode = $("runtime").value;
};

$("model").onclick = () => {
  const pop = $("modelpop");
  if (!pop.classList.contains("hidden")) return closePops();
  renderModelPop();
  show(pop, true);
};

// Providers that are ready, their models, then the chosen model's options
// (effort, fast mode, …) from its optionDescriptors.
function renderModelPop() {
  const c = choices();
  const providers = (state.config?.providers ?? []).filter((p) => p.enabled && p.status === "ready" && p.models.length);
  const model = modelOf(c.modelSelection);
  let html = providers
    .map(
      (p) =>
        `<div class="group">${esc(p.displayName ?? p.instanceId)}</div>` +
        p.models
          .filter((m) => !state.config.settings?.providerModelPreferences?.[p.instanceId]?.hiddenModels?.includes(m.slug))
          .map((m) => {
            const on = c.modelSelection?.instanceId === p.instanceId && c.modelSelection?.model === m.slug;
            return `<div class="item${on ? " on" : ""}" data-provider="${esc(p.instanceId)}" data-model="${esc(m.slug)}">${esc(m.name)}</div>`;
          })
          .join(""),
    )
    .join("");
  const descriptors = model?.capabilities?.optionDescriptors ?? [];
  if (descriptors.length) {
    html += `<div class="opts">${descriptors
      .map((d) =>
        d.type === "boolean"
          ? `<label><input type="checkbox" data-opt="${esc(d.id)}"${optionValue(c.modelSelection, d) ? " checked" : ""}> ${esc(d.label)}</label>`
          : `<label>${esc(d.label)} <select data-opt="${esc(d.id)}">${(d.options ?? [])
              .map((o) => `<option value="${esc(o.id)}"${o.id === optionValue(c.modelSelection, d) ? " selected" : ""}>${esc(o.label ?? o.id)}</option>`)
              .join("")}</select></label>`,
      )
      .join("")}</div>`;
  }
  $("modelpop").innerHTML = html || `<div class="dim" style="padding:8px">No ready providers.</div>`;
}

$("modelpop").onclick = (event) => {
  const pick = event.target.closest("[data-model]");
  if (!pick) return;
  const c = choices();
  const same = c.modelSelection?.instanceId === pick.dataset.provider;
  // Options are per model; keep the ones the new model also has.
  const descriptors = providerOf(pick.dataset.provider)?.models.find((m) => m.slug === pick.dataset.model)?.capabilities?.optionDescriptors ?? [];
  const options = same ? (c.modelSelection.options ?? []).filter((o) => descriptors.some((d) => d.id === o.id)) : [];
  c.modelSelection = { instanceId: pick.dataset.provider, model: pick.dataset.model, options };
  renderModelPop();
  renderToolbar();
};
$("modelpop").onchange = (event) => {
  const id = event.target.dataset.opt;
  if (!id) return;
  const c = choices();
  const value = event.target.type === "checkbox" ? event.target.checked : event.target.value;
  const options = (c.modelSelection.options ?? []).filter((o) => o.id !== id);
  c.modelSelection = { ...c.modelSelection, options: [...options, { id, value }] };
  renderToolbar();
};

// Values like "ultrathink" are prompt words, not settings: T3 prefixes the
// text (except for /commands) and leaves the value out of the selection.
function sendableSelection(selection) {
  const descriptors = modelOf(selection)?.capabilities?.optionDescriptors ?? [];
  const injected = (o) => descriptors.some((d) => d.id === o.id && d.promptInjectedValues?.includes(o.value));
  return { ...selection, options: (selection.options ?? []).filter((o) => !injected(o)) };
}
function withPromptEffort(text, selection) {
  const descriptors = modelOf(selection)?.capabilities?.optionDescriptors ?? [];
  const injected = descriptors.some((d) => d.type === "select" && d.promptInjectedValues?.includes("ultrathink") && optionValue(selection, d) === "ultrathink");
  if (!injected || !text || /^\/[^\s/]+(?:\s|$)/u.test(text) || text.startsWith("Ultrathink:")) return text;
  return `Ultrathink:\n${text}`;
}

// ---------- questions, queue ----------

// Agents ask questions with options (AskUserQuestion and friends). Answers go
// back as {questionId: optionValue | [values] | customText}.
function renderQuestions() {
  const box = $("questions");
  const request = current()?.pendingRuntimeRequest;
  const item = request && [...view.items.values()].find((i) => i.type === "user_input_request" && i.requestId === request.id);
  if (!item?.questions?.length) {
    show(box, false);
    box.dataset.request = "";
    return;
  }
  show(box, true);
  if (box.dataset.request === item.requestId) return;
  box.dataset.request = item.requestId;
  view.answers = new Map(item.questions.map((q) => [q.id, { picked: new Set(), text: q.initialAnswer ?? "" }]));
  box.innerHTML =
    item.questions
      .map(
        (q) => `<div class="question" data-q="${esc(q.id)}">
          <div class="h">${esc(q.header)}</div><div>${esc(q.question)}</div>
          <div class="choices">${q.options
            .map((o) => `<button type="button" data-value="${esc(o.value ?? o.label)}" title="${esc(o.description)}">${esc(o.label)}</button>`)
            .join("")}</div>
          ${q.allowCustomAnswer === false ? "" : `<input placeholder="Or type an answer" value="${esc(q.initialAnswer ?? "")}">`}
        </div>`,
      )
      .join("") +
    `<div class="question row" style="border:0;padding:0"><button type="button" data-dismiss>Dismiss</button><button type="button" class="primary" data-answer>Answer</button></div>`;
  box._questions = item.questions;
}

$("questions").onclick = async (event) => {
  const box = $("questions");
  const requestId = box.dataset.request;
  const choice = event.target.closest("[data-value]");
  if (choice) {
    const q = box._questions.find((x) => x.id === choice.closest("[data-q]").dataset.q);
    const answer = view.answers.get(q.id);
    const value = choice.dataset.value;
    if (q.multiSelect) answer.picked.has(value) ? answer.picked.delete(value) : answer.picked.add(value);
    else answer.picked = answer.picked.has(value) ? new Set() : new Set([value]);
    for (const b of choice.parentElement.children) b.classList.toggle("on", answer.picked.has(b.dataset.value));
    return;
  }
  try {
    if (event.target.closest("[data-dismiss]")) {
      await dispatchFor("thread.user-input.dismiss", { requestId });
    } else if (event.target.closest("[data-answer]")) {
      const answers = {};
      for (const q of box._questions) {
        const a = view.answers.get(q.id);
        const typed = box.querySelector(`[data-q="${CSS.escape(q.id)}"] input`)?.value.trim();
        if (typed) answers[q.id] = typed;
        else if (a.picked.size) answers[q.id] = q.multiSelect ? [...a.picked] : [...a.picked][0];
        else return fail(new Error(`Answer "${q.header}" first.`));
      }
      await dispatchFor("runtime-request.respond", { requestId, answers });
    }
  } catch (error) {
    fail(error);
  }
};

// Messages sent while a run is busy wait in the queue. Each can be sent now
// as a steer into the active run, edited, or cancelled.
function renderQueue() {
  const queued = [...view.runs.values()].filter((r) => r.status === "queued").sort((a, b) => (a.queuePosition ?? 0) - (b.queuePosition ?? 0));
  const box = $("queue");
  show(box, queued.length > 0);
  const held = queued.some((r) => r.queueHeld);
  const html =
    queued
      .map((r) => {
        const text = view.messages.get(r.userMessageId)?.text ?? "Queued message";
        return `<div class="queued" data-run="${esc(r.id)}"><span title="${esc(text)}">⏳ ${esc(text)}</span>
          ${current()?.activeRunId ? `<button data-act="steer">Send now</button>` : ""}
          <button data-act="edit">Edit</button><button data-act="cancel" class="danger">✕</button></div>`;
      })
      .join("") + (held ? `<div class="queued"><span>The queue is paused after a stop.</span><button data-act="resume">Resume</button></div>` : "");
  if (box.sig !== html) {
    box.sig = html;
    box.innerHTML = html;
  }
}

$("queue").onclick = async (event) => {
  const act = event.target.closest("[data-act]")?.dataset.act;
  const runId = event.target.closest("[data-run]")?.dataset.run;
  try {
    if (act === "resume") await dispatchFor("queue.resume");
    else if (act === "cancel") await dispatchFor("queued-run.cancel", { runId });
    else if (act === "steer") await dispatchFor("queued-message.promote-to-steer", { queuedRunId: runId, targetRunId: current().activeRunId });
    else if (act === "edit") {
      const old = view.messages.get(view.runs.get(runId)?.userMessageId)?.text ?? "";
      const text = prompt("Edit queued message", old);
      if (text != null && text !== old) await dispatchFor("queued-run.edit", { runId, text });
    }
  } catch (error) {
    fail(error);
  }
};

// Approve/decline only fits permission prompts. Questions have their own
// panel; MCP elicitations and auth refreshes still need T3.
const YES_NO = new Set(["command", "file-read", "file-change", "permission"]);

function renderApproval() {
  const request = current()?.pendingRuntimeRequest;
  const questions = request?.kind === "user_input" && [...view.items.values()].some((i) => i.requestId === request.id && i.questions);
  show($("approval"), Boolean(request) && !questions);
  if (!request || questions) return;
  const yesNo = YES_NO.has(request.kind);
  for (const button of $("approval").querySelectorAll("button")) show(button, yesNo);
  const asked = [...view.items.values()].find((i) => i.requestId === request.id);
  const kind = request.kind.replace(/[-_]/g, " ");
  $("approvalText").textContent = yesNo
    ? `Agent wants: ${kind}${asked?.prompt || asked?.reason ? ` · ${asked.prompt || asked.reason}` : ""}`
    : `Agent needs ${kind}. Answer it in T3 for now.`;
}

/** Code fences, inline code, bold, headings. Everything else stays plain text. */
function inlineMarkdown(text) {
  return String(text ?? "")
    .split(/```[\w-]*\n?/)
    .map((part, i) =>
      i % 2
        ? `<pre>${esc(part.replace(/\n$/, ""))}</pre>`
        : esc(part)
            .replace(/`([^`\n]+)`/g, "<code>$1</code>")
            .replace(/\*\*([^*\n]+)\*\*/g, "<b>$1</b>")
            .replace(/^#{1,6} (.+)$/gm, "<b>$1</b>"),
    )
    .join("");
}

const json = (v) => (v == null ? "" : typeof v === "string" ? v : JSON.stringify(v, null, 2));
const firstLine = (s) => String(s ?? "").trim().split("\n")[0].slice(0, 160);

// MCP results look like {content: string | [{type:"text", text}], structuredContent}.
function toolOutput(output) {
  const content = output?.content;
  if (typeof content === "string") return content;
  if (Array.isArray(content)) return content.map((c) => c.text ?? json(c)).join("\n");
  return json(output);
}

/** [title, body] for a collapsible row. */
function toolView(t) {
  switch (t.type) {
    case "command_execution":
      return [`$ ${firstLine(t.input)}`, [t.input, t.output, t.exitCode != null && `exit ${t.exitCode}`].filter(Boolean).join("\n\n")];
    case "file_change": {
      const stat = t.additions != null ? ` +${t.additions} −${t.deletions ?? 0}` : "";
      return [`edit ${t.fileName}${stat}`, t.diffStr ?? t.newStr ?? (view.full.has(t.id) ? "No diff from this provider." : "")];
    }
    case "file_search":
      return [`search ${t.pattern ?? ""}`, (t.results ?? []).map(json).join("\n")];
    case "web_search":
      return [`web ${(t.patterns ?? []).join(", ")}`, (t.results ?? []).map(json).join("\n")];
    case "subagent":
      return [`agent ${t.title ?? firstLine(t.prompt)}`, [t.prompt, t.result ?? t.progress].filter(Boolean).join("\n\n")];
    case "dynamic_tool":
      return [`${(t.toolName ?? "tool").replace(/^mcp__/, "")} ${firstLine(JSON.stringify(t.input ?? ""))}`, [json(t.input), t.output !== undefined && toolOutput(t.output)].filter(Boolean).join("\n\n")];
    default:
      return [t.title || t.type.replace(/_/g, " "), json(t)];
  }
}

// The socket strips command output and diffs; getTurnItem has the whole item.
async function loadToolDetail(item) {
  if (!TOOLS.has(item.type) || view.full.get(item.id)?.updatedAt === item.updatedAt || !state.rpc) return;
  const threadId = state.threadId;
  try {
    const { item: full } = await state.rpc.call("orchestration.getTurnItem", { threadId, itemId: item.id });
    if (!full || state.threadId !== threadId) return;
    view.full.set(item.id, { ...full, updatedAt: item.updatedAt });
    renderMessages();
  } catch (error) {
    fail(error);
  }
}

/** [kind, cls, html-or-null, title, body] for one row. */
function rowView(item) {
  const full = view.full.get(item.id);
  const t = full?.updatedAt === item.updatedAt ? full : item;
  switch (t.type) {
    case "user_message":
    case "assistant_message":
    case "proposed_plan": {
      const role = t.type === "user_message" ? "user" : "assistant";
      const n = t.attachments?.length ?? 0;
      if (!n && !String(t.text ?? t.markdown ?? "").trim()) return ["div", "", ""];
      const note = n ? `<div class="dim">${n} attachment${n > 1 ? "s" : ""}</div>` : "";
      return ["div", `msg ${role}${t.streaming ? " streaming" : ""}`, inlineMarkdown(String(t.text ?? t.markdown).trim()) + note];
    }
    case "todo_list":
      return ["div", "msg system", (t.steps ?? []).map((s) => `${s.status === "completed" ? "☑" : "☐"} ${esc(s.step ?? s.text ?? json(s))}`).join("<br>")];
    case "error":
      return ["div", "msg error", esc(t.failure?.message ?? "Error")];
    case "system_notice":
    case "run_interrupt_result":
      return ["div", "msg system", esc(t.message)];
    case "compaction":
      return ["div", "msg system", esc(t.title ?? "Context compacted")];
    // Like T3: "Thinking" while live, "Thought" after, with the text inline. Tap to unclamp.
    case "reasoning": {
      const live = t.status === "running" || t.streaming;
      const text = String(t.text ?? "").trim();
      return ["div", `msg thought${live ? " live" : ""}`, `<b>${live ? "Thinking" : "Thought"}</b>${text ? " " + inlineMarkdown(text) : ""}`];
    }
    default: {
      const failed = t.status === "failed" || t.outputIndicatesFailure;
      const live = t.status === "running" || t.streaming;
      const [title, body] = toolView(t);
      return ["details", `msg tool${live ? " running" : ""}${failed ? " failed" : ""}`, null, title, body];
    }
  }
}

// "19s", "2m 5s", "1h 3m", as T3 writes them.
function duration(ms) {
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m${s % 60 ? ` ${s % 60}s` : ""}`;
  return `${Math.floor(s / 3600)}h${Math.floor(s / 60) % 60 ? ` ${Math.floor(s / 60) % 60}m` : ""}`;
}
const at = (iso) => (iso ? Date.parse(iso) : NaN);

/**
 * Like T3, a finished run folds everything before its final reply into one
 * "Worked for 19s" row. A message sent mid-run starts a new fold. Runs still
 * going, or with an error, stay open. Returns item key -> fold for the first
 * folded item, and the set of folded keys.
 */
function foldRuns(list) {
  const folds = new Map();
  const hidden = new Set();
  const active = current()?.activeRunId;
  let segment = [];
  let boundary = null; // the user message that opened this segment
  const close = () => {
    const runId = segment[0]?.runId;
    const run = view.runs.get(runId);
    if (!runId || runId === active || ACTIVE.has(run?.status) || run?.status === "queued" || segment.some((i) => i.type === "error")) return;
    let reply = segment.findLastIndex((i) => i.type === "assistant_message" && String(i.text ?? "").trim());
    if (reply < 0) reply = segment.length;
    const work = segment.slice(0, reply).filter((i) => !HIDDEN.has(i.type));
    if (!work.some((i) => i.type !== "assistant_message")) return;
    const key = `fold:${work[0].id}`;
    const first = boundary?.runId === runId ? boundary : null;
    const start = first ? at(first.startedAt ?? first.updatedAt) : at(run?.startedAt ?? work[0].startedAt);
    const last = segment[Math.min(reply, segment.length - 1)];
    const end = reply === segment.length && run?.completedAt ? at(run.completedAt) : at(last.completedAt ?? last.updatedAt);
    const took = Number.isFinite(start) && Number.isFinite(end) ? duration(Math.max(0, end - start)) : null;
    const stopped = run?.status === "interrupted" || segment.some((i) => i.type === "run_interrupt_result");
    const label = stopped ? (took ? `You stopped after ${took}` : "You stopped this response") : took ? `Worked for ${took}` : "Worked";
    folds.set(rowKey(work[0]), { key, label, open: view.unfolded.has(key) });
    if (!view.unfolded.has(key)) for (const i of work) hidden.add(rowKey(i));
  };
  for (const item of list) {
    if (item.type === "user_message" || item.runId !== segment[0]?.runId) {
      close();
      segment = [];
    }
    if (item.type === "user_message") boundary = item;
    else segment.push(item);
  }
  close();
  return { folds, hidden };
}

function renderMessages() {
  const box = $("messages");
  const nearBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 80;
  const list = [...view.items.values()].sort((a, b) => (a.ordinal ?? 0) - (b.ordinal ?? 0));
  $("empty")?.remove();

  let previous = null;
  const place = (el) => {
    if (previous ? previous.nextElementSibling !== el : box.firstElementChild !== el) {
      previous ? previous.after(el) : box.prepend(el);
    }
    previous = el;
  };

  if (view.history.more) {
    let more = view.rows.get("more");
    if (!more) {
      more = document.createElement("button");
      more.className = "more";
      more.onclick = loadEarlier;
      view.rows.set("more", more);
    }
    more.textContent = view.history.loading ? "Loading…" : "Load earlier";
    place(more);
  } else {
    view.rows.get("more")?.remove();
    view.rows.delete("more");
  }

  const seen = new Set(["more"]);
  // Only a plan nobody has replied to yet can be implemented.
  const lastUser = list.findLast((i) => i.type === "user_message")?.ordinal ?? -1;
  view.latestPlanId = list.findLast((i) => i.type === "proposed_plan" && (i.ordinal ?? 0) > lastUser)?.planId ?? null;
  const { folds, hidden } = foldRuns(list);
  for (const item of list) {
    const key = rowKey(item);
    const fold = folds.get(key);
    if (fold) {
      let row = view.rows.get(fold.key);
      if (!row) {
        row = document.createElement("div");
        row.className = "msg fold";
        row.onclick = () => {
          view.unfolded.has(fold.key) ? view.unfolded.delete(fold.key) : view.unfolded.add(fold.key);
          renderMessages();
        };
        view.rows.set(fold.key, row);
      }
      const text = `${fold.label} ${fold.open ? "⌄" : "›"}`;
      if (row.textContent !== text) row.textContent = text;
      seen.add(fold.key);
      place(row);
    }
    if (HIDDEN.has(item.type) || hidden.has(key)) continue;
    const [tag, cls, html, title, body] = rowView(item);
    if (html === "" && tag === "div") continue;
    seen.add(key);
    let el = view.rows.get(key);
    if (!el || el.tagName.toLowerCase() !== tag) {
      el?.remove();
      el = document.createElement(tag);
      view.rows.set(key, el);
      if (item.type === "reasoning") el.onclick = () => el.toggleAttribute("data-open");
      if (tag === "details") el.ontoggle = () => el.open && loadToolDetail(el.item);
    }
    el.item = item;
    if (el.className !== cls) el.className = cls;
    if (tag === "div") {
      if (el.sig !== html) {
        el.sig = html;
        el.innerHTML = html;
      }
    } else if (el.sig !== title + "\0" + body) {
      el.sig = title + "\0" + body;
      el.innerHTML = `<summary>${esc(title)}</summary><pre>${esc(body || (TOOLS.has(item.type) ? "Loading…" : ""))}</pre>`;
      // Output that changes while the row is open (a command finishing) is fetched again.
      if (el.open) loadToolDetail(item);
    }
    place(el);
  }
  // Claude's reply text arrives in whole blocks, so show that a run is alive.
  const thinking = list.some((i) => i.type === "reasoning" && (i.status === "running" || i.streaming));
  if (view.working && !thinking) {
    let working = view.rows.get("working");
    if (!working) {
      working = document.createElement("div");
      working.className = "msg thought live working";
      working.innerHTML = "<b>Thinking</b>";
      view.rows.set("working", working);
    }
    seen.add("working");
    place(working);
  }
  for (const [key, el] of view.rows) {
    if (!seen.has(key)) {
      el.remove();
      view.rows.delete(key);
    }
  }
  if (!list.length && !view.history.more && !view.working) box.innerHTML = `<div id="empty">No messages yet.</div>`;
  if (nearBottom) box.scrollTop = box.scrollHeight;
  renderApproval();
  renderQuestions();
  renderQueue();
  renderPlanBar();
}

// ---------- actions ----------

$("input").onkeydown = (event) => {
  if (suggest.range && onSuggestKey(event)) return;
  // Alt+Enter queues behind the running turn instead of steering into it.
  if (event.key === "Enter" && event.altKey) {
    event.preventDefault();
    return send($("input").value.trim(), { queue: true });
  }
  // Phones have no Shift+Enter, so there Enter is a newline and Send sends.
  if (event.key === "Enter" && !event.shiftKey && !event.isComposing && !matchMedia("(pointer: coarse)").matches) {
    event.preventDefault();
    $("composer").requestSubmit();
  }
};
function fitInput() {
  const input = $("input");
  input.style.height = "44px";
  input.style.height = `${Math.min(200, input.scrollHeight)}px`;
}
$("input").oninput = () => {
  fitInput();
  saveDraft();
  updateSuggest();
};

// ---------- drafts ----------

// Like T3, unsent text is kept per thread (and per project for a new thread)
// in localStorage, written 300 ms after typing stops and on leaving the page.
// It stays on this device. Images aren't kept.
const DRAFTS_KEY = "tern.drafts";
const drafts = JSON.parse(localStorage.getItem(DRAFTS_KEY) || "{}");
const draftKey = () => (state.draft ? `new:${state.draft.projectId}` : state.threadId);
let draftTimer = 0;

function setDraft(key, text) {
  if (!key) return;
  if (text.trim()) drafts[key] = { text, at: Date.now() };
  else delete drafts[key];
  clearTimeout(draftTimer);
  draftTimer = setTimeout(flushDrafts, 300);
}
const saveDraft = () => setDraft(draftKey(), $("input").value);

function flushDrafts() {
  clearTimeout(draftTimer);
  // Keep the newest 200 so the storage can't grow without bound.
  const old = Object.entries(drafts).sort((a, b) => b[1].at - a[1].at).slice(200);
  for (const [key] of old) delete drafts[key];
  localStorage.setItem(DRAFTS_KEY, JSON.stringify(drafts));
}
addEventListener("pagehide", flushDrafts);
document.addEventListener("visibilitychange", () => document.hidden && flushDrafts());

function restoreDraft() {
  $("input").value = drafts[draftKey()]?.text ?? "";
  fitInput();
}
$("input").onclick = () => updateSuggest();
$("input").onblur = () => setTimeout(closeSuggest, 150);

// ---------- / commands and $ skills ----------

// Like T3: the word at the cursor picks the menu. "/" lists the provider's
// commands, "$" (or any currency sign) its skills. Both go out as plain text,
// "/name" or "$name", and the provider expands them.
const suggest = { items: [], index: 0, range: null, prefix: "" };

// server.getConfig is ~400 KB (every provider, every workspace), so fetch it
// once per connection, on the first trigger.
function serverConfig(rpc) {
  rpc.config ??= rpc.call("server.getConfig", {}).catch((error) => {
    rpc.config = null;
    throw error;
  });
  return rpc.config;
}

function triggerAt(text, cursor) {
  const start = text.slice(0, cursor).search(/\S*$/);
  const word = text.slice(start, cursor);
  if (/^\/[\w:.-]*$/.test(word)) return { prefix: "/", query: word.slice(1), start };
  const currency = /^\p{Sc}(?!\d)([\w:.-]*)$/u.exec(word); // not "$5"
  if (currency) return { prefix: "$", query: currency[1], start };
  if (word.startsWith("@")) return { prefix: "@", query: word.slice(1), start };
  return null;
}

// "@" searches the thread's folder like T3's composer: projects.searchEntries,
// limit 80, files and directories together.
let fileSearch = 0;
async function searchFiles(cwd, query) {
  const ticket = ++fileSearch;
  await new Promise((r) => setTimeout(r, 120));
  if (ticket !== fileSearch) return null;
  const result = await state.rpc.call("projects.searchEntries", { cwd, query, limit: 80 });
  return ticket === fileSearch ? result.entries.map((e) => ({ name: e.path, description: e.kind, kind: e.kind })) : null;
}

// T3 inserts a file mention as a Markdown link in the text: [name](path).
const mentionLink = (path) =>
  `[${path.split("/").filter(Boolean).at(-1) ?? path}](${encodeURI(path).replace(/[()#?\\]/g, (c) => "%" + c.charCodeAt(0).toString(16).toUpperCase())})`;

async function updateSuggest() {
  const input = $("input");
  const trigger = triggerAt(input.value, input.selectionStart);
  const t = state.draft ? null : current();
  if (!trigger || (!t && !state.draft) || !state.rpc) return closeSuggest();
  let config;
  try {
    config = state.config = await serverConfig(state.rpc);
  } catch (error) {
    closeSuggest();
    return fail(error);
  }
  // The input may have moved on while the config loaded.
  const now = triggerAt(input.value, input.selectionStart);
  if (!now || now.start !== trigger.start || now.prefix !== trigger.prefix) return;
  // The provider the next message goes to, which the model picker may have changed.
  const provider = config.providers?.find((p) => p.instanceId === (choices().modelSelection?.instanceId ?? t?.providerInstanceId));
  const cwd = t?.worktreePath ?? state.projects.get(t?.projectId ?? state.draft.projectId)?.workspaceRoot;
  const workspace = provider?.workspaceSnapshots?.find((w) => w.cwd === cwd);
  let all;
  if (now.prefix === "@") {
    if (!cwd) return closeSuggest();
    try {
      all = await searchFiles(cwd, now.query);
    } catch (error) {
      closeSuggest();
      return fail(error);
    }
    if (!all) return; // a newer keystroke owns the menu
    const latest = triggerAt(input.value, input.selectionStart);
    if (!latest || latest.start !== now.start) return;
  } else {
    all =
      now.prefix === "/"
        ? (workspace?.slashCommands ?? provider?.slashCommands ?? [])
        : (workspace?.skills ?? provider?.skills ?? []).filter((k) => k.enabled !== false);
  }
  const q = now.query.toLowerCase();
  const starts = all.filter((c) => c.name.toLowerCase().startsWith(q));
  const contains = all.filter((c) => !c.name.toLowerCase().startsWith(q) && c.name.toLowerCase().includes(q));
  suggest.items = [...starts, ...contains];
  suggest.index = 0;
  suggest.prefix = now.prefix;
  suggest.range = [now.start, input.selectionStart];
  renderSuggest(all.length ? "No match." : now.prefix === "/" ? "No commands for this provider." : "No skills for this provider.");
}

function renderSuggest(empty) {
  const box = $("suggest");
  show(box, true);
  if (!suggest.items.length) {
    box.innerHTML = `<div class="dim">${empty}</div>`;
    return;
  }
  box.innerHTML = suggest.items
    .map(
      (c, i) =>
        `<div class="pick${i === suggest.index ? " on" : ""}" data-i="${i}"><b>${esc((suggest.prefix === "@" ? "" : suggest.prefix) + c.name)}</b>` +
        `${c.input?.hint ? ` <span class="dim">${esc(c.input.hint)}</span>` : ""}` +
        `<span class="dim">${esc(c.shortDescription ?? c.description ?? "")}</span></div>`,
    )
    .join("");
  box.querySelector(".on")?.scrollIntoView({ block: "nearest" });
}

function closeSuggest() {
  suggest.items = [];
  suggest.range = null;
  show($("suggest"), false);
}

function pickSuggest(i) {
  const item = suggest.items[i];
  if (!item || !suggest.range) return;
  const input = $("input");
  const [start, end] = suggest.range;
  const insert = suggest.prefix === "@" ? `${mentionLink(item.name)} ` : `${suggest.prefix}${item.name} `;
  input.value = input.value.slice(0, start) + insert + input.value.slice(end);
  input.selectionStart = input.selectionEnd = start + insert.length;
  saveDraft();
  closeSuggest();
  input.focus();
}

function onSuggestKey(event) {
  const n = suggest.items.length;
  if (event.key === "Escape") {
    closeSuggest();
  } else if (!n) {
    return false; // "No match." is showing; let Enter send.
  } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
    suggest.index = (suggest.index + (event.key === "ArrowDown" ? 1 : n - 1)) % n;
    renderSuggest();
  } else if ((event.key === "Enter" && !event.shiftKey) || event.key === "Tab") {
    pickSuggest(suggest.index);
  } else {
    return false;
  }
  event.preventDefault();
  return true;
}

// pointerdown, not click, so the pick lands before the input's blur closes the menu.
$("suggest").onpointerdown = (event) => {
  const pick = event.target.closest(".pick");
  if (!pick) return;
  event.preventDefault();
  pickSuggest(Number(pick.dataset.i));
};

// ---------- images ----------

const IMAGE_TYPES = new Set(["image/png", "image/jpeg", "image/gif", "image/webp"]);
let images = [];

function renderPending() {
  show($("pending"), images.length > 0);
  $("pending").innerHTML = images.map((f, i) => `<img data-i="${i}" title="Remove" src="${f.preview}">`).join("");
}

function addImages(files) {
  for (const file of files) {
    if (!IMAGE_TYPES.has(file.type)) {
      fail(new Error(`${file.name || "That file"} isn't PNG, JPEG, GIF or WebP.`));
      continue;
    }
    if (file.size > 10 * 1024 * 1024) {
      fail(new Error(`${file.name} is over 10 MB.`));
      continue;
    }
    file.preview = URL.createObjectURL(file);
    images.push(file);
  }
  renderPending();
}

$("pending").onclick = (event) => {
  const i = event.target.dataset.i;
  if (i === undefined) return;
  URL.revokeObjectURL(images[i].preview);
  images.splice(Number(i), 1);
  renderPending();
};
$("file").onchange = (event) => {
  addImages(event.target.files);
  event.target.value = "";
};
$("input").onpaste = (event) => {
  const files = [...event.clipboardData.files];
  if (files.length) {
    event.preventDefault();
    addImages(files);
  }
};
$("messages").ondragover = (event) => event.preventDefault();
$("messages").ondrop = (event) => {
  event.preventDefault();
  addImages(event.dataTransfer.files);
};

$("composer").onsubmit = (event) => {
  event.preventDefault();
  send($("input").value.trim(), { queue: event.submitter?.dataset.queue === "1" });
};

/**
 * Sends like T3's composer: modes first if they changed, then the message
 * with the chosen model. Busy threads get deliveryIntent "auto", and the
 * server picks steer or queue. Drafts become a thread through launchThread.
 */
async function send(text, { queue = false, plan = null } = {}) {
  const t = state.draft ? null : current();
  if ((!text && !images.length) || (!t && !state.draft) || !state.rpc) return;
  const c = choices();
  if (!c.modelSelection) return fail(new Error("Pick a model first."));
  const files = images;
  const key = draftKey();
  images = [];
  renderPending();
  $("input").value = "";
  fitInput();
  setDraft(key, "");
  const body = withPromptEffort(text, c.modelSelection);
  try {
    const status = $("net").textContent;
    if (files.length) net("uploading…");
    const attachments = await Promise.all(files.map((f) => state.rpc.uploadImage(f)));
    net(status);
    files.forEach((f) => URL.revokeObjectURL(f.preview));
    if (state.draft) {
      const result = await state.rpc.call("orchestration.launchThread", {
        commandId: crypto.randomUUID(),
        creationSource: "web",
        projectId: state.draft.projectId,
        title: firstLine(text).slice(0, 80) || "New thread",
        generateTitle: true,
        modelSelection: sendableSelection(c.modelSelection),
        runtimeMode: c.runtimeMode,
        interactionMode: c.interactionMode,
        workspaceStrategy: { type: "root" },
        initialMessage: { messageId: crypto.randomUUID(), text: body, attachments },
      });
      state.choices.set(result.threadId, c);
      return openThread(result.threadId);
    }
    if (c.runtimeMode !== t.runtimeMode) await dispatchFor("thread.runtime-mode.set", { runtimeMode: c.runtimeMode });
    if (c.interactionMode !== t.interactionMode) await dispatchFor("thread.interaction-mode.set", { interactionMode: c.interactionMode });
    const smart = state.config?.environment?.capabilities?.serverResolvedCommandContext === true;
    await dispatchFor("message.dispatch", {
      messageId: crypto.randomUUID(),
      text: body,
      attachments,
      modelSelection: sendableSelection(c.modelSelection),
      createdBy: "user",
      creationSource: "web",
      ...(plan ? { sourcePlanRef: plan } : {}),
      ...(smart && !queue ? { deliveryIntent: "auto" } : {}),
      dispatchMode: { type: (queue || !smart) && t.activeRunId ? "queue_after_active" : "start_immediately" },
    });
  } catch (error) {
    // Put the text back where it came from, even if another thread is open now.
    setDraft(key, text);
    if (draftKey() === key) {
      $("input").value = text;
      fitInput();
      images = files;
      renderPending();
    }
    fail(error);
  }
}

// Like T3, a plan nobody has answered yet gets its actions by the composer,
// since the plan itself usually sits inside the run's fold.
function renderPlanBar() {
  const plan = !view.working && view.latestPlanId && [...view.items.values()].find((i) => i.type === "proposed_plan" && i.planId === view.latestPlanId);
  const box = $("planbar");
  show(box, Boolean(plan));
  if (!plan || box.dataset.plan === plan.planId) return;
  box.dataset.plan = plan.planId;
  box.innerHTML = `<details><summary>Plan ready</summary><div class="msg">${inlineMarkdown(String(plan.markdown).trim())}</div></details>
    <button class="primary" data-implement="${esc(plan.planId)}">Implement plan</button>`;
}

// "Implement plan" leaves plan mode and sends the plan back, as T3 does.
$("planbar").addEventListener("click", (event) => {
  const planId = event.target.closest("[data-implement]")?.dataset.implement;
  if (!planId) return;
  const plan = [...view.items.values()].find((i) => i.type === "proposed_plan" && i.planId === planId);
  choices().interactionMode = "default";
  renderToolbar();
  send(`PLEASE IMPLEMENT THIS PLAN:\n${String(plan?.markdown ?? "").trim()}`, { plan: { threadId: state.threadId, planId } });
});

// thread.stop also stops commands from earlier runs and delegated tasks;
// older servers only know run.interrupt.
$("stop").onclick = async () => {
  const t = current();
  if (!t?.activeRunId) return;
  try {
    if (state.config?.environment?.capabilities?.threadStop) await dispatchFor("thread.stop");
    else await dispatchFor("run.interrupt", { runId: t.activeRunId, holdQueue: true });
  } catch (error) {
    fail(error);
  }
};

$("approval").onclick = async (event) => {
  const decision = event.target.dataset.decision;
  const t = current();
  if (!decision || !t?.pendingRuntimeRequest) return;
  try {
    await state.rpc.dispatch({
      type: "runtime-request.respond",
      threadId: t.id,
      requestId: t.pendingRuntimeRequest.id,
      decision,
    });
  } catch (error) {
    fail(error);
  }
};

// ---------- boot ----------

async function start() {
  if (!clerk.hasClient) return showSignIn();
  net("loading…");
  try {
    state.envs = await relay.listEnvironments();
  } catch (error) {
    if (/Not signed in|401/.test(error.message)) return showSignIn();
    throw error;
  }
  show($("app"), true);
  show($("signout"), true);
  renderEnvs();
  refreshStatuses();
  setInterval(refreshStatuses, 60_000);
  const target = state.envs.find((e) => e.environmentId === state.envId) ?? state.envs[0];
  if (target) selectEnv(target.environmentId);
  else net("");
}

if (!window.isSecureContext) {
  fail(new Error("Open this over https or http://localhost; WebCrypto needs a secure context."));
} else {
  relay = new Relay(clerk, await DpopKey.load());
  start().catch(fail);
}
