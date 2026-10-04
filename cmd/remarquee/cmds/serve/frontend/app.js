// remarquee files UI — vanilla ES module, no build step.

const state = {
  cwd: "/",
  entries: [],
  selected: null,
  query: "",
  status: null,
};

const $ = (id) => document.getElementById(id);

async function api(path, opts) {
  const res = await fetch(path, opts);
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = await res.json();
      if (body && body.error && body.error.message) msg = body.error.message;
    } catch (_) { /* ignore */ }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

function toast(msg, isError) {
  const el = $("toast");
  el.textContent = msg;
  el.hidden = false;
  el.classList.toggle("error", !!isError);
  clearTimeout(toast._t);
  toast._t = setTimeout(() => { el.hidden = true; }, 4000);
}

// ---------- URL <-> path routing ----------
// The current directory lives in the URL hash (#/ai/2026/10/04) so links are
// shareable and back/forward navigation works.
function currentHashPath() {
  let h = location.hash.replace(/^#/, "");
  try { h = decodeURIComponent(h); } catch (_) { /* keep raw */ }
  if (!h) return "/";
  return h.startsWith("/") ? h : "/" + h;
}

function setHash(p) {
  if (currentHashPath() === p) return;
  location.hash = p;
}

window.addEventListener("hashchange", () => {
  const p = currentHashPath();
  if (p !== state.cwd) loadFiles(p, true);
});

function fmtTime(s) {
  if (!s) return "—";
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return "—";
  const pad = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function shortType(e) {
  if (e.isDir) return "folder";
  if (e.type === "DocumentType") return "document";
  return e.type || "file";
}

async function loadStatus() {
  try {
    state.status = await api("/api/status");
  } catch (e) {
    setStatus("Could not read status: " + e.message, "error");
    return;
  }
  const s = state.status;
  if (!s.pandocAvailable) {
    setStatus("pandoc was not found on PATH: Markdown uploads will fail. PDF/EPUB uploads still work.", "warn");
  } else {
    setStatus("");
  }
}

function setStatus(text, kind) {
  const el = $("status");
  if (!text) { el.hidden = true; el.textContent = ""; return; }
  el.hidden = false;
  el.textContent = text;
  el.className = "status " + (kind || "");
}

async function loadFiles(dir, fromHistory) {
  state.cwd = dir || "/";
  if (!fromHistory) setHash(state.cwd);
  state.selected = null;
  renderDetail(null);
  try {
    const data = await api("/api/files?dir=" + encodeURIComponent(state.cwd));
    state.entries = data.entries || [];
  } catch (e) {
    toast("List failed: " + e.message, true);
    state.entries = [];
  }
  renderCrumbs();
  renderRows();
}

async function runSearch(q) {
  state.query = q;
  if (!q.trim()) { await loadFiles(state.cwd); return; }
  try {
    const data = await api("/api/search?q=" + encodeURIComponent(q) + "&limit=200");
    state.entries = (data.results || []).map((r) => r.entry);
  } catch (e) {
    toast("Search failed: " + e.message, true);
    state.entries = [];
  }
  renderCrumbs();
  renderRows();
}

function renderCrumbs() {
  const el = $("crumbs");
  el.textContent = "";
  if (state.query.trim()) {
    el.append(elWith("span", null, "search: " + state.query));
    return;
  }
  const parts = state.cwd === "/" ? [] : state.cwd.split("/").filter(Boolean);
  const root = document.createElement("a");
  root.href = "#";
  root.textContent = "/";
  root.onclick = (e) => { e.preventDefault(); loadFiles("/"); };
  el.append(root);
  let acc = "";
  for (const p of parts) {
    acc += "/" + p;
    el.append(document.createTextNode(" / "));
    const a = document.createElement("a");
    a.href = "#";
    a.textContent = p;
    const target = acc;
    a.onclick = (e) => { e.preventDefault(); loadFiles(target); };
    el.append(a);
  }
}

function elWith(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text != null) e.textContent = text;
  return e;
}

function renderRows() {
  const tbody = $("rows");
  tbody.textContent = "";
  $("empty").hidden = state.entries.length > 0;
  $("search-count").textContent = state.query.trim() ? `${state.entries.length} match(es)` : `${state.entries.length} item(s)`;

  for (const e of state.entries) {
    const tr = document.createElement("tr");
    tr.className = "row" + (state.selected && state.selected.id === e.id ? " selected" : "");
    tr.dataset.id = e.id;

    const name = document.createElement("td");
    name.className = "name";
    const label = document.createElement("span");
    label.className = e.isDir ? "folder" : "file";
    label.textContent = e.name;
    name.append(label);
    if (state.query.trim()) {
      const p = elWith("div", "muted", e.path);
      name.append(p);
    }

    const type = elWith("td", "type", shortType(e) + (e.version ? ` v${e.version}` : ""));
    const mod = elWith("td", "mod", fmtTime(e.modifiedTime || e.modifiedClient));

    tr.append(name, type, mod);
    tr.onclick = () => selectEntry(e, tr);
    tr.ondblclick = () => { if (e.isDir) loadFiles(e.path); };
    tbody.append(tr);
  }
}

function selectEntry(entry, tr) {
  state.selected = entry;
  for (const r of document.querySelectorAll("tr.row")) r.classList.toggle("selected", r === tr);
  renderDetail(entry);
}

function renderDetail(entry) {
  const el = $("detail");
  el.textContent = "";
  if (!entry) {
    el.append(elWith("p", "muted", "Select an entry to see details."));
    return;
  }
  const dl = document.createElement("dl");
  const rows = [
    ["NAME", entry.name],
    ["ID", entry.id],
    ["PATH", entry.path],
    ["TYPE", entry.type],
    ["IS DIR", entry.isDir ? "yes" : "no"],
    ["VERSION", String(entry.version)],
    ["MODIFIED", fmtTime(entry.modifiedTime || entry.modifiedClient)],
  ];
  for (const [k, v] of rows) {
    dl.append(elWith("dt", null, k));
    dl.append(elWith("dd", null, v));
  }
  el.append(dl);

  const actions = document.createElement("div");
  actions.className = "actions";
  if (!entry.isDir) {
    const dl2 = elWith("a", "btn", "Download");
    dl2.href = "/api/entries/" + encodeURIComponent(entry.id) + "/download";
    actions.append(dl2);
  }
  actions.append(mkBtn("Rename", () => promptRename(entry)));
  actions.append(mkBtn("Move", () => promptMove(entry)));
  const del = mkBtn("Delete", () => confirmDelete(entry), "danger");
  actions.append(del);
  el.append(actions);
}

function mkBtn(label, onClick, cls) {
  const b = elWith("button", "btn" + (cls ? " " + cls : ""), label);
  b.type = "button";
  b.onclick = onClick;
  return b;
}

// ---------- actions ----------

async function promptRename(entry) {
  const value = await promptDialog("RENAME", "NEW NAME", entry.name);
  if (value == null || value === entry.name) return;
  try {
    await api("/api/entries/" + encodeURIComponent(entry.id), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: value }),
    });
    toast("Renamed to " + value);
    await refreshAfterMutation();
  } catch (e) { toast("Rename failed: " + e.message, true); }
}

async function promptMove(entry) {
  const value = await promptDialog("MOVE", "DESTINATION REMOTE PATH", "/");
  if (value == null) return;
  try {
    await api("/api/entries/" + encodeURIComponent(entry.id) + "/move", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ destDir: value }),
    });
    toast("Moved to " + value);
    await refreshAfterMutation();
  } catch (e) { toast("Move failed: " + e.message, true); }
}

async function confirmDelete(entry) {
  const ok = await confirmDialog(`Delete ${entry.name}? This cannot be undone.`);
  if (!ok) return;
  try {
    await api("/api/entries", {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ids: [entry.id], confirm: "DELETE", recursive: true }),
    });
    toast("Deleted " + entry.name);
    await refreshAfterMutation();
  } catch (e) {
    // The cloud may have committed some deletions before returning an
    // error. Refresh the visible state without replacing the error toast
    // with a success message or implying the operation was rolled back.
    toast("Delete failed: " + e.message, true);
    try { await refreshAfterMutation(); }
    catch (refreshError) {
      toast("Delete failed: " + e.message + "; refresh failed: " + refreshError.message, true);
    }
  }
}

async function refreshAfterMutation() {
  if (state.query.trim()) await runSearch(state.query);
  else await loadFiles(state.cwd);
}

// ---------- dialogs ----------

let promptResolve = null;
function promptDialog(title, label, initial) {
  $("prompt-title").textContent = title;
  $("prompt-label").textContent = label;
  $("prompt-input").value = initial || "";
  const dlg = $("prompt-dialog");
  dlg.showModal();
  $("prompt-input").focus();
  $("prompt-input").select();
  return new Promise((resolve) => { promptResolve = resolve; });
}
$("prompt-form").addEventListener("submit", (e) => {
  e.preventDefault();
  const v = $("prompt-input").value;
  $("prompt-dialog").close();
  if (promptResolve) { promptResolve(v); promptResolve = null; }
});
$("prompt-cancel").addEventListener("click", () => {
  $("prompt-dialog").close();
  if (promptResolve) { promptResolve(null); promptResolve = null; }
});

let confirmResolve = null;
function confirmDialog(text) {
  $("confirm-text").textContent = text;
  $("confirm-input").value = "";
  const dlg = $("confirm-dialog");
  dlg.showModal();
  $("confirm-input").focus();
  return new Promise((resolve) => { confirmResolve = resolve; });
}
$("confirm-form").addEventListener("submit", (e) => {
  e.preventDefault();
  const typed = $("confirm-input").value;
  if (typed !== "DELETE") { toast("Type DELETE to confirm", true); return; }
  $("confirm-dialog").close();
  if (confirmResolve) { confirmResolve(true); confirmResolve = null; }
});
$("confirm-cancel").addEventListener("click", () => {
  $("confirm-dialog").close();
  if (confirmResolve) { confirmResolve(false); confirmResolve = null; }
});

// ---------- upload ----------

let staged = []; // { file, path }

function openUploadDialog(destDir) {
  $("upload-dest").value = destDir || state.cwd;
  $("job-progress").textContent = "";
  renderStaged();
  $("upload-dialog").showModal();
}

function setStaged(entries) {
  staged = entries;
  renderStaged();
}

function renderStaged() {
  const box = $("staged");
  if (staged.length === 0) { box.hidden = true; box.textContent = ""; return; }
  box.hidden = false;
  box.textContent = "";
  box.append(elWith("div", null, `${staged.length} file(s) staged:`));
  for (const s of staged.slice(0, 200)) box.append(elWith("div", null, s.path));
  if (staged.length > 200) box.append(elWith("div", null, `… and ${staged.length - 200} more`));
}

function fileListToStaged(fileList, useRelative) {
  const out = [];
  for (const f of fileList) {
    const rel = useRelative && f.webkitRelativePath ? f.webkitRelativePath : f.name;
    out.push({ file: f, path: rel });
  }
  return out;
}

$("upload-btn").addEventListener("click", () => { setStaged([]); openUploadDialog(state.cwd); });
$("uploadfolder-btn").addEventListener("click", () => {
  setStaged([]);
  openUploadDialog(state.cwd);
  $("upload-dir").click();
});
$("upload-cancel").addEventListener("click", () => $("upload-dialog").close());

$("upload-files").addEventListener("change", (e) => setStaged(fileListToStaged(e.target.files, false)));
$("upload-dir").addEventListener("change", (e) => setStaged(fileListToStaged(e.target.files, true)));

$("upload-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  if (staged.length === 0) { toast("Choose files or a folder, or drag them onto the page", true); return; }
  const fd = new FormData();
  fd.append("destDir", $("upload-dest").value || "/");
  for (const s of staged) {
    fd.append("files", s.file, s.file.name);
    fd.append("paths", s.path);
  }
  $("upload-submit").disabled = true;
  try {
    const res = await api("/api/uploads", { method: "POST", body: fd });
    await pollJob(res.jobId);
    await refreshAfterMutation();
  } catch (err) {
    toast("Upload failed: " + err.message, true);
  } finally {
    $("upload-submit").disabled = false;
  }
});

// ---------- drag & drop ----------

let dragDepth = 0;
function hasFiles(e) {
  return e.dataTransfer && Array.from(e.dataTransfer.types || []).includes("Files");
}

document.addEventListener("dragenter", (e) => {
  if (!hasFiles(e)) return;
  e.preventDefault();
  dragDepth++;
  $("drop-overlay").hidden = false;
});
document.addEventListener("dragover", (e) => {
  if (!hasFiles(e)) return;
  e.preventDefault();
  if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
});
document.addEventListener("dragleave", (e) => {
  if (!hasFiles(e)) return;
  dragDepth--;
  if (dragDepth <= 0) { dragDepth = 0; $("drop-overlay").hidden = true; }
});
document.addEventListener("drop", async (e) => {
  if (!hasFiles(e)) return;
  e.preventDefault();
  dragDepth = 0;
  $("drop-overlay").hidden = true;
  const entries = await collectDropped(e.dataTransfer);
  if (entries.length === 0) { toast("Nothing to upload", true); return; }
  setStaged(entries);
  openUploadDialog(state.cwd);
});

async function collectDropped(dt) {
  const out = [];
  // Capture entries synchronously: the DataTransfer becomes invalid after the
  // drop handler yields to the event loop.
  const roots = [];
  for (const it of dt.items ? Array.from(dt.items) : []) {
    if (it.kind === "file" && typeof it.webkitGetAsEntry === "function") {
      const entry = it.webkitGetAsEntry();
      if (entry) roots.push(entry);
    }
  }
  if (roots.length > 0) {
    for (const entry of roots) await traverseEntry(entry, "", out);
    return out;
  }
  for (const f of dt.files || []) out.push({ file: f, path: f.name });
  return out;
}

async function traverseEntry(entry, prefix, out) {
  if (entry.isFile) {
    const file = await new Promise((resolve, reject) => entry.file(resolve, reject));
    out.push({ file, path: prefix + entry.name });
    return;
  }
  if (entry.isDirectory) {
    const reader = entry.createReader();
    for (;;) {
      const batch = await new Promise((resolve, reject) => reader.readEntries(resolve, reject));
      if (!batch.length) break;
      for (const child of batch) await traverseEntry(child, prefix + entry.name + "/", out);
    }
  }
}

async function pollJob(jobId) {
  const box = $("job-progress");
  for (;;) {
    let job;
    try { job = await api("/api/uploads/" + encodeURIComponent(jobId)); }
    catch (e) { box.textContent = "job lost: " + e.message; return; }
    box.textContent = "";
    const div = elWith("div", "job");
    div.append(elWith("div", null, `job ${job.state} — ${job.done}/${job.total} → ${job.destDir}`));
    const ul = document.createElement("ul");
    for (const it of job.items) {
      const li = elWith("li", "state-" + it.state, `${it.name}: ${it.state}${it.error ? " — " + it.error : ""}`);
      ul.append(li);
    }
    div.append(ul);
    box.append(div);
    if (job.state === "done" || job.state === "failed") return;
    await new Promise((r) => setTimeout(r, 700));
  }
}

// ---------- new folder ----------

$("newfolder-btn").addEventListener("click", () => {
  $("newfolder-parent").value = state.cwd;
  $("newfolder-name").value = "";
  $("newfolder-dialog").showModal();
  $("newfolder-name").focus();
});
$("newfolder-cancel").addEventListener("click", () => $("newfolder-dialog").close());
$("newfolder-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const parentPath = $("newfolder-parent").value || "/";
  const name = $("newfolder-name").value.trim();
  if (!name) { toast("Enter a folder name", true); return; }
  try {
    await api("/api/folders", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ parentPath, name }),
    });
    $("newfolder-dialog").close();
    toast("Created " + name);
    await refreshAfterMutation();
  } catch (err) { toast("Create failed: " + err.message, true); }
});

// ---------- refresh ----------

$("refresh-btn").addEventListener("click", async () => {
  const btn = $("refresh-btn");
  btn.disabled = true;
  setStatus("Refreshing cloud tree…", "warn");
  try {
    await api("/api/refresh", { method: "POST" });
    setStatus("");
    toast("Cloud tree refreshed");
    await refreshAfterMutation();
  } catch (e) {
    setStatus("Refresh failed: " + e.message, "error");
  } finally {
    btn.disabled = false;
  }
});

// ---------- search ----------

let searchTimer = null;
$("search").addEventListener("input", (e) => {
  const q = e.target.value;
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => runSearch(q), 180);
});

// ---------- boot ----------

(async function boot() {
  await loadStatus();
  await loadFiles(currentHashPath(), false);
})();
