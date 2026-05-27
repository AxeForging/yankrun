const rows = document.querySelector("#rows");
const statusEl = document.querySelector("#status");
const notice = document.querySelector("#notice");
const cloneRepo = document.querySelector("#cloneRepo");
const templateSelect = document.querySelector("#templateSelect");
const savedRunsList = document.querySelector("#savedRunsList");
const presetSearch = document.querySelector("#presetSearch");
let summary = { keys: [], counts: {}, values: {} };
let repoType = "ssh";
let activeMode = "local";
let lastRunMeta = { mode: "local" };
let savedRuns = [];
let presetFilter = "all";
let evaluateTimer = 0;

function show(msg, kind) {
  notice.textContent = msg;
  notice.className = "banner show " + kind;
}

function esc(v) {
  return String(v || "").replace(/[&<>"']/g, m => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "\"": "&quot;",
    "'": "&#39;"
  }[m]));
}

function values() {
  const out = {};
  document.querySelectorAll("[data-key]").forEach(i => out[i.dataset.key] = i.value);
  return out;
}

function db() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open("yankrun-workbench", 1);
    req.onupgradeneeded = () => req.result.createObjectStore("runs", { keyPath: "id" });
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function storeRun(run) {
  const database = await db();
  return new Promise((resolve, reject) => {
    const tx = database.transaction("runs", "readwrite");
    tx.objectStore("runs").put(run);
    tx.oncomplete = resolve;
    tx.onerror = () => reject(tx.error);
  });
}

async function allRuns() {
  const database = await db();
  return new Promise((resolve, reject) => {
    const tx = database.transaction("runs", "readonly");
    const req = tx.objectStore("runs").getAll();
    req.onsuccess = () => resolve(req.result || []);
    req.onerror = () => reject(req.error);
  });
}

async function refreshSavedRuns() {
  savedRuns = (await allRuns()).sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  renderSavedRuns();
}

function renderSavedRuns() {
  const targets = new Set(savedRuns.map(runTarget));
  document.querySelector("#savedMeta").textContent = savedRuns.length + " presets · " + targets.size + " targets";
  const query = presetSearch.value.trim().toLowerCase();
  const visible = savedRuns.filter(r => {
    if (presetFilter !== "all" && r.kind !== presetFilter) return false;
    if (!query) return true;
    return [
      r.kind,
      runTarget(r),
      r.payload.branch || "",
      r.payload.outputDir || "",
      Object.keys(r.values || {}).join(" ")
    ].join(" ").toLowerCase().includes(query);
  });
  savedRunsList.innerHTML = visible.length
    ? visible.slice(0, 8).map(renderSavedRun).join("")
    : '<div class="saved-empty">No matching presets.</div>';
  savedRunsList.querySelectorAll("[data-run-id]").forEach(b => b.addEventListener("click", () => loadRun(b.dataset.runId)));
}

async function rememberRun(kind, payload, body) {
  const labelTarget = payload.repo || payload.template || "local";
  const run = {
    id: Date.now().toString(36) + Math.random().toString(36).slice(2),
    label: kind + " · " + labelTarget,
    kind,
    payload,
    values: values(),
    summary: body.summary || summary,
    target: labelTarget,
    placeholders: body.summary && body.summary.keys ? body.summary.keys.length : summary.keys.length,
    matches: body.summary && body.summary.counts ? totalMatches(body.summary.counts) : totalMatches(summary.counts),
    createdAt: new Date().toISOString()
  };
  await storeRun(run);
  await refreshSavedRuns();
}

function runTarget(run) {
  return run.target || run.payload.repo || run.payload.template || "local";
}

function totalMatches(counts) {
  return counts ? Object.values(counts).reduce((n, v) => n + v, 0) : 0;
}

function shortTarget(target) {
  return target.replace(/^https:\/\/github\.com\//, "").replace(/^git@github\.com:/, "").replace(/\.git$/, "");
}

function renderSavedRun(run) {
  const target = runTarget(run);
  const branch = run.payload.branch ? " · " + run.payload.branch : "";
  const matches = run.matches || (run.summary && run.summary.counts ? totalMatches(run.summary.counts) : 0);
  const placeholders = run.placeholders || (run.summary && run.summary.keys ? run.summary.keys.length : 0);
  return '<button class="saved-run" type="button" data-run-id="' + esc(run.id) + '">' +
    '<div class="saved-main"><span class="saved-kind">' + esc(run.kind) + '</span><strong>' + esc(shortTarget(target)) + '</strong></div>' +
    '<div class="saved-stats"><span>' + placeholders + ' keys</span><span>' + matches + ' hits</span></div>' +
    '<span class="saved-time">' + esc(new Date(run.createdAt).toLocaleString()) + branch + '</span>' +
  '</button>';
}

function setBusy(label) {
  statusEl.textContent = label;
  document.querySelectorAll("button").forEach(b => b.disabled = true);
}

function setReady() {
  statusEl.textContent = "ready";
  document.querySelectorAll("button").forEach(b => b.disabled = false);
  const applyButton = document.querySelector("#apply");
  if (applyButton && applyButton.dataset.forceDry === "true") {
    applyButton.disabled = true;
  }
}

async function readJSON(r) {
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(body.error || "Request failed");
  return body;
}

async function scan() {
  setBusy("scanning");
  try {
    const r = await fetch("/api/scan");
    summary = await readJSON(r);
    render();
  } catch (err) {
    show(err.message || "Scan failed", "err");
  } finally {
    setReady();
  }
}

function render() {
  const total = summary.keys.reduce((n, k) => n + (summary.counts[k] || 0), 0);
  document.querySelector("#metricKeys").textContent = summary.keys.length;
  document.querySelector("#metricMatches").textContent = total;
  if (!summary.keys.length) {
    rows.innerHTML = '<div class="empty">No placeholders found.</div>';
    return;
  }
  rows.innerHTML = summary.keys.map((k, i) =>
    '<div class="row" style="--i:' + i + '">' +
    '<div class="value-cell"><label>' + esc(k) + '</label>' +
    '<input data-key="' + esc(k) + '" value="' + esc(summary.values[k] || "") + '" autocomplete="off">' +
    renderTree(k) + '</div>' +
    '<div class="count">' + (summary.counts[k] || 0) + ' hits</div></div>'
  ).join("");
  document.querySelectorAll("[data-key]").forEach(i => i.addEventListener("input", scheduleEvaluate));
}

function renderTree(key) {
  const files = Array.isArray(summary.files) ? summary.files.filter(f => f.counts && f.counts[key]) : [];
  if (!files.length) return "";
  return '<div class="file-tree" aria-label="Files containing ' + esc(key) + '">' +
    '<div class="tree-root">./</div>' +
    files.map((f, i) => {
      const branch = i === files.length - 1 ? "`--" : "+--";
      return '<div class="tree-file"><span class="branch">' + branch + '</span><span class="path">' + esc(f.path) + '</span><span class="hit-pill">' + f.counts[key] + '</span></div>' +
        renderPreviews(f, key);
    }).join("") +
  '</div>';
}

function renderPreviews(file, key) {
  const previews = Array.isArray(file.previews) ? file.previews.filter(p => p.key === key) : [];
  if (!previews.length) return "";
  return previews.map(p => {
    const value = p.error ? "error: " + p.error : (p.missing ? "missing value" : p.value);
    const cls = p.error ? "bad" : (p.missing ? "missing" : "good");
    return '<div class="eval ' + cls + '"><span>' + esc(p.expression) + '</span><strong>' + esc(value) + '</strong></div>';
  }).join("");
}

async function apply(dryRun) {
  const payload = { values: values(), dryRun };
  setBusy(dryRun ? "previewing" : "applying");
  try {
    const body = await postJSON("/api/apply", payload);
    reportResult(body, "Applied", "Preview");
    await rememberRun("local", payload, body);
    if (body.applied) await scan();
  } catch (err) {
    show(err.message || "Request failed", "err");
  } finally {
    setReady();
  }
}

function scheduleEvaluate() {
  clearTimeout(evaluateTimer);
  statusEl.textContent = "editing";
  evaluateTimer = setTimeout(evaluateCurrent, 2000);
}

async function evaluateCurrent() {
  if (!summary.keys || !summary.keys.length) return;
  const active = document.activeElement && document.activeElement.dataset ? document.activeElement.dataset.key : "";
  const start = document.activeElement && document.activeElement.selectionStart;
  const end = document.activeElement && document.activeElement.selectionEnd;
  statusEl.textContent = "evaluating";
  try {
    summary = await postJSON("/api/evaluate", { summary, values: values() });
    render();
    if (active) {
      const input = document.querySelector('[data-key="' + CSS.escape(active) + '"]');
      if (input) {
        input.focus();
        if (Number.isInteger(start) && Number.isInteger(end)) input.setSelectionRange(start, end);
      }
    }
  } catch (err) {
    show(err.message || "Evaluate failed", "err");
  } finally {
    setReady();
  }
}

async function postJSON(url, payload) {
  const r = await fetch(url, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload)
  });
  return readJSON(r);
}

function reportResult(body, appliedVerb, previewVerb) {
  if (body.summary && Array.isArray(body.summary.keys)) {
    summary = body.summary;
    render();
  }
  if (body.applied) {
    show(appliedVerb + " " + body.totalMatches + " replacements across " + body.placeholders + " placeholders.", "ok");
    return;
  }
  show(previewVerb + ": " + body.totalMatches + " replacements across " + body.placeholders + " placeholders. No files modified.", body.forcedDryRun ? "warn" : "ok");
}

function setMode(mode) {
  activeMode = mode;
  document.querySelectorAll("[data-mode]").forEach(b => b.classList.toggle("active", b.dataset.mode === mode));
  document.querySelectorAll(".mode").forEach(p => p.classList.toggle("active", p.id === "mode-" + mode));
}

function setRepoType(next) {
  repoType = next;
  document.querySelectorAll("[data-repo-type]").forEach(b => b.classList.toggle("active", b.dataset.repoType === next));
  cloneRepo.placeholder = next === "ssh" ? "git@github.com:org/repo.git" : "https://github.com/org/repo.git";
}

async function loadTemplates() {
  try {
    const r = await fetch("/api/templates");
    const templates = await readJSON(r);
    const options = Array.isArray(templates) ? templates : [];
    if (!options.length) {
      templateSelect.innerHTML = '<option value="">No configured templates</option>';
      return;
    }
    templateSelect.innerHTML = options.map(t =>
      '<option value="' + esc(t.name) + '">' + esc(t.name) + (t.defaultBranch ? " · " + esc(t.defaultBranch) : "") + '</option>'
    ).join("");
  } catch (err) {
    templateSelect.innerHTML = '<option value="">Template lookup failed</option>';
    show(err.message || "Template lookup failed", "err");
  }
}

async function cloneApply(dryRun) {
  const payload = {
    repo: cloneRepo.value.trim(),
    branch: document.querySelector("#cloneBranch").value.trim(),
    outputDir: document.querySelector("#cloneOutput").value.trim(),
    values: values(),
    dryRun
  };
  lastRunMeta = { mode: "clone", repo: payload.repo, branch: payload.branch, outputDir: payload.outputDir };
  setBusy(dryRun ? "previewing clone" : "cloning");
  try {
    const body = await postJSON("/api/clone", payload);
    reportResult(body, "Cloned and applied", "Clone preview");
    await rememberRun("clone", payload, body);
    if (body.applied) await scan();
  } catch (err) {
    show(err.message || "Clone failed", "err");
  } finally {
    setReady();
  }
}

async function generateApply(dryRun) {
  const payload = {
    template: templateSelect.value,
    branch: document.querySelector("#generateBranch").value.trim(),
    outputDir: document.querySelector("#generateOutput").value.trim(),
    values: values(),
    dryRun
  };
  lastRunMeta = { mode: "generate", template: payload.template, branch: payload.branch, outputDir: payload.outputDir };
  setBusy(dryRun ? "previewing generate" : "generating");
  try {
    const body = await postJSON("/api/generate", payload);
    reportResult(body, "Generated and applied", "Generate preview");
    await rememberRun("generate", payload, body);
    if (body.applied) await scan();
  } catch (err) {
    show(err.message || "Generate failed", "err");
  } finally {
    setReady();
  }
}

function loadRun(id) {
  const selected = savedRuns.find(r => r.id === id);
  if (!selected) return;
  Object.entries(selected.values || {}).forEach(([key, value]) => {
    const input = document.querySelector('[data-key="' + CSS.escape(key) + '"]');
    if (input) input.value = value;
  });
  if (selected.summary) {
    summary = selected.summary;
    render();
  }
  if (selected.kind === "clone") {
    setMode("clone");
    cloneRepo.value = selected.payload.repo || "";
    document.querySelector("#cloneBranch").value = selected.payload.branch || "";
    document.querySelector("#cloneOutput").value = selected.payload.outputDir || "";
  }
  if (selected.kind === "generate") {
    setMode("generate");
    templateSelect.value = selected.payload.template || "";
    document.querySelector("#generateBranch").value = selected.payload.branch || "";
    document.querySelector("#generateOutput").value = selected.payload.outputDir || "";
  }
  if (selected.kind === "local") setMode("local");
  show("Loaded saved run.", "ok");
}

function exportRuns() {
  const blob = new Blob([JSON.stringify({ version: 1, runs: savedRuns }, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "yankrun-workbench-runs.json";
  a.click();
  URL.revokeObjectURL(url);
}

async function importRuns(file) {
  const parsed = JSON.parse(await file.text());
  const runs = Array.isArray(parsed.runs) ? parsed.runs : [];
  for (const run of runs) {
    if (run && run.id) await storeRun(run);
  }
  await refreshSavedRuns();
  show("Imported " + runs.length + " saved runs.", "ok");
}

document.querySelectorAll("[data-mode]").forEach(b => b.addEventListener("click", () => setMode(b.dataset.mode)));
document.querySelectorAll("[data-repo-type]").forEach(b => b.addEventListener("click", () => setRepoType(b.dataset.repoType)));
document.querySelectorAll("[data-preset-filter]").forEach(b => b.addEventListener("click", () => {
  presetFilter = b.dataset.presetFilter;
  document.querySelectorAll("[data-preset-filter]").forEach(x => x.classList.toggle("active", x.dataset.presetFilter === presetFilter));
  renderSavedRuns();
}));
presetSearch.addEventListener("input", renderSavedRuns);
document.querySelector("#refresh").addEventListener("click", scan);
document.querySelector("#preview").addEventListener("click", () => apply(true));
document.querySelector("#apply").addEventListener("click", () => apply(false));
document.querySelector("#clonePreview").addEventListener("click", () => cloneApply(true));
document.querySelector("#cloneApply").addEventListener("click", () => cloneApply(false));
document.querySelector("#generatePreview").addEventListener("click", () => generateApply(true));
document.querySelector("#generateApply").addEventListener("click", () => generateApply(false));
document.querySelector("#exportRuns").addEventListener("click", exportRuns);
document.querySelector("#importRuns").addEventListener("click", () => document.querySelector("#importFile").click());
document.querySelector("#importFile").addEventListener("change", e => {
  const file = e.target.files && e.target.files[0];
  if (file) importRuns(file).catch(err => show(err.message || "Import failed", "err"));
  e.target.value = "";
});
loadTemplates();
refreshSavedRuns().catch(() => show("Saved runs unavailable in this browser.", "warn"));
scan();
