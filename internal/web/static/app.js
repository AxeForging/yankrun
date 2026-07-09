const rows = document.querySelector("#rows");
const statusEl = document.querySelector("#status");
const notice = document.querySelector("#notice");
const cloneRepo = document.querySelector("#cloneRepo");
const templateSelect = document.querySelector("#templateSelect");
const savedRunsList = document.querySelector("#savedRunsList");
const presetSearch = document.querySelector("#presetSearch");
const delimForm = document.querySelector("#delimForm");
const delimStart = document.querySelector("#delimStart");
const delimEnd = document.querySelector("#delimEnd");
const progressLine = document.querySelector("#progressLine");
const reduceMotion = matchMedia("(prefers-reduced-motion: reduce)").matches;
let summary = { keys: [], counts: {}, values: {} };
let repoType = "ssh";
let activeMode = "local";
let lastRunMeta = { mode: "local" };
let savedRuns = [];
let presetFilter = "all";
let evaluateTimer = 0;
let noticeTimer = 0;
let previewedOnce = false;

// ---- Workflow stepper (Scan -> Fill -> Preview -> Apply) ----
const FLOW = ["scan", "fill", "preview", "apply"];

function setFlow(stage) {
  const idx = stage === "done" ? FLOW.length : FLOW.indexOf(stage);
  document.querySelectorAll(".flow-step").forEach((el, i) => {
    el.classList.toggle("done", i < idx);
    el.classList.toggle("active", i === idx);
  });
}

function show(msg, kind) {
  clearTimeout(noticeTimer);
  const icon = kind === "ok" ? "✓" : kind === "warn" ? "△" : "✕";
  notice.innerHTML = '<span class="banner-icon">' + icon + "</span><span>" + esc(msg) + "</span>";
  notice.className = "banner show " + kind;
  // Success toasts auto-dismiss; warnings and errors stay until addressed.
  if (kind === "ok") noticeTimer = setTimeout(() => { notice.className = "banner"; }, 6000);
}

function hideNotice() {
  clearTimeout(noticeTimer);
  notice.className = "banner";
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

async function removeRun(id) {
  const database = await db();
  return new Promise((resolve, reject) => {
    const tx = database.transaction("runs", "readwrite");
    tx.objectStore("runs").delete(id);
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
  savedRunsList.querySelectorAll("[data-del-id]").forEach(b => b.addEventListener("click", e => {
    e.stopPropagation();
    removeRun(b.dataset.delId).then(refreshSavedRuns).then(() => show("Preset deleted.", "ok"));
  }));
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

// timeAgo renders a compact relative timestamp for the preset rail.
function timeAgo(iso) {
  const s = (Date.now() - new Date(iso).getTime()) / 1000;
  if (s < 60) return "just now";
  if (s < 3600) return Math.floor(s / 60) + "m ago";
  if (s < 86400) return Math.floor(s / 3600) + "h ago";
  if (s < 604800) return Math.floor(s / 86400) + "d ago";
  return new Date(iso).toLocaleDateString();
}

function renderSavedRun(run, i) {
  const target = runTarget(run);
  const branch = run.payload.branch ? " · " + run.payload.branch : "";
  const matches = run.matches || (run.summary && run.summary.counts ? totalMatches(run.summary.counts) : 0);
  const placeholders = run.placeholders || (run.summary && run.summary.keys ? run.summary.keys.length : 0);
  return '<div class="saved-run" style="--i:' + i + '">' +
    '<button class="saved-load" type="button" data-run-id="' + esc(run.id) + '" title="Load this preset">' +
      '<div class="saved-main"><span class="saved-kind">' + esc(run.kind) + '</span><strong>' + esc(shortTarget(target)) + '</strong></div>' +
      '<div class="saved-stats"><span>' + placeholders + ' keys</span><span>' + matches + ' hits</span></div>' +
      '<span class="saved-time">' + esc(timeAgo(run.createdAt)) + branch + '</span>' +
    '</button>' +
    '<button class="saved-del" type="button" data-del-id="' + esc(run.id) + '" title="Delete preset" aria-label="Delete preset">✕</button>' +
  '</div>';
}

function setBusy(label) {
  statusEl.textContent = label;
  statusEl.classList.add("busy");
  progressLine.classList.add("on");
  document.querySelectorAll("button").forEach(b => b.disabled = true);
}

function setReady() {
  statusEl.textContent = "ready";
  statusEl.classList.remove("busy");
  progressLine.classList.remove("on");
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

// skeleton renders shimmer placeholders while a scan is in flight.
function skeleton() {
  return Array.from({ length: 3 }, (_, i) =>
    '<div class="skel-row" style="--i:' + i + '"><div class="skel skel-label"></div><div class="skel skel-input"></div></div>'
  ).join("");
}

async function scan() {
  setBusy("scanning");
  setFlow("scan");
  rows.innerHTML = skeleton();
  try {
    const r = await fetch("/api/scan");
    summary = await readJSON(r);
    previewedOnce = false;
    render();
  } catch (err) {
    show(err.message || "Scan failed", "err");
    rows.innerHTML = emptyState("Scan failed.", "Check the server log, then rescan.");
  } finally {
    setReady();
  }
}

function emptyState(title, hint) {
  return '<div class="empty"><div class="empty-glyph">⟦ ⟧</div>' + esc(title) +
    (hint ? "<small>" + esc(hint) + "</small>" : "") + "</div>";
}

// setMetric animates the number toward its new value with a count-up.
function setMetric(sel, target) {
  const el = document.querySelector(sel);
  const from = parseInt(el.textContent, 10) || 0;
  if (reduceMotion || from === target) { el.textContent = target; return; }
  el.classList.remove("bump");
  void el.offsetWidth; // restart the bump animation
  el.classList.add("bump");
  const t0 = performance.now(), dur = 450;
  requestAnimationFrame(function tick(now) {
    const p = Math.min(1, (now - t0) / dur);
    const eased = 1 - Math.pow(1 - p, 3);
    el.textContent = Math.round(from + (target - from) * eased);
    if (p < 1) requestAnimationFrame(tick);
  });
}

function render() {
  const total = summary.keys.reduce((n, k) => n + (summary.counts[k] || 0), 0);
  setMetric("#metricKeys", summary.keys.length);
  setMetric("#metricMatches", total);
  const fillWrap = document.querySelector("#fillWrap");
  if (!summary.keys.length) {
    rows.innerHTML = emptyState("No placeholders found.", "Nothing matches the current delimiter pair — adjust it in the topbar or rescan.");
    fillWrap.hidden = true;
    setFlow("done");
    return;
  }
  fillWrap.hidden = false;
  rows.innerHTML = summary.keys.map((k, i) =>
    '<div class="row" style="--i:' + i + '">' +
    '<div class="value-cell">' + renderField(k) +
    renderTree(k) + '</div>' +
    '<div class="count" data-count-for="' + esc(k) + '">' + (summary.counts[k] || 0) + ' hits</div></div>'
  ).join("");
  document.querySelectorAll("[data-key]").forEach(i => {
    i.addEventListener("input", () => {
      previewedOnce = false; // edits invalidate the last preview
      updateFillProgress();
      scheduleEvaluate();
    });
    if (i.dataset.pattern) {
      i.addEventListener("input", () => validatePattern(i));
      validatePattern(i);
    }
  });
  updateFillProgress();
}

// updateFillProgress drives the n/m meter, per-row state rules, and the
// stepper: all filled -> Preview is next; previewed -> Apply is next.
function updateFillProgress() {
  const inputs = [...document.querySelectorAll("[data-key]")];
  const total = inputs.length;
  const filled = inputs.filter(i => i.value.trim() !== "").length;
  const label = document.querySelector("#fillLabel");
  const fill = document.querySelector("#fillFill");
  if (label) label.textContent = total ? filled + "/" + total + " filled" : "";
  if (fill) {
    fill.style.width = total ? (filled / total * 100) + "%" : "0%";
    fill.classList.toggle("complete", total > 0 && filled === total);
  }
  inputs.forEach(i => {
    const row = i.closest(".row");
    if (!row) return;
    const mv = manifestVar(i.dataset.key);
    const has = i.value.trim() !== "";
    row.classList.toggle("filled", has);
    row.classList.toggle("needs", !!(mv && mv.required) && !has);
    const count = row.querySelector("[data-count-for]");
    if (count) {
      count.innerHTML = (has ? '<span class="tick">✓</span>' : "") +
        (summary.counts[i.dataset.key] || 0) + " hits";
    }
  });
  if (!total) return;
  if (filled === total) setFlow(previewedOnce ? "apply" : "preview");
  else setFlow("fill");
}

// manifestVar returns the yankrun.yaml declaration for a key, if any.
function manifestVar(key) {
  const m = summary.manifest;
  if (!m || !Array.isArray(m.variables)) return null;
  return m.variables.find(v => v.key === key) || null;
}

// renderField draws the label (with a required badge and description from the
// manifest) plus the control — a select for enums, an input otherwise.
function renderField(k) {
  const mv = manifestVar(k);
  const val = summary.values[k] || "";
  const req = mv && mv.required ? '<span class="req-badge">REQUIRED</span>' : "";
  const desc = mv && mv.description ? '<p class="field-desc">' + esc(mv.description) + '</p>' : "";
  let control;
  if (mv && Array.isArray(mv.enum) && mv.enum.length) {
    control = '<select data-key="' + esc(k) + '" aria-label="' + esc(k) + '">' +
      mv.enum.map(o => '<option value="' + esc(o) + '"' + (o === val ? ' selected' : '') + '>' + esc(o) + '</option>').join("") +
      '</select>';
  } else {
    const pat = mv && mv.pattern ? ' data-pattern="' + esc(mv.pattern) + '"' : "";
    control = '<input data-key="' + esc(k) + '" value="' + esc(val) + '" autocomplete="off" aria-label="' + esc(k) + '"' + pat + '>';
  }
  return '<label>' + esc(k) + req + '</label>' + desc + control;
}

// validatePattern flags an input whose value violates its manifest pattern.
function validatePattern(input) {
  try {
    const re = new RegExp(input.dataset.pattern);
    const ok = input.value === "" || re.test(input.value);
    input.setAttribute("aria-invalid", ok ? "false" : "true");
  } catch (_) {
    /* an invalid pattern is the template author's problem, not the user's */
  }
}

// renderDiff shows a file's dry-run unified diff in a collapsible
// terminal-styled block with +/− line stats in the summary bar.
function renderDiff(file) {
  if (!file.diff) return "";
  const lines = file.diff.split("\n");
  let added = 0, removed = 0;
  const body = lines.map(line => {
    let cls = "ctx";
    if (line.startsWith("+") && !line.startsWith("+++")) { cls = "add"; added++; }
    else if (line.startsWith("-") && !line.startsWith("---")) { cls = "del"; removed++; }
    return '<span class="' + cls + '">' + esc(line) + '</span>';
  }).join("\n");
  return '<details class="diff" open><summary class="diff-head">' +
    '<span><span class="chev">▸</span>' + esc(file.path) + '</span>' +
    '<span class="diff-stat"><b class="add">+' + added + '</b><b class="del">−' + removed + '</b></span>' +
    '</summary><pre>' + body + '</pre></details>';
}

function renderTree(key) {
  const files = Array.isArray(summary.files) ? summary.files.filter(f => f.counts && f.counts[key]) : [];
  if (!files.length) return "";
  const hits = files.reduce((n, f) => n + (f.counts[key] || 0), 0);
  const hasDiff = files.some(f => f.diff && Object.keys(f.counts || {}).sort()[0] === key);
  // Keep small trees (or anything carrying a preview diff) open; collapse the rest.
  const open = files.length <= 3 || hasDiff ? " open" : "";
  const label = files.length + (files.length === 1 ? " file" : " files") + " · " + hits + (hits === 1 ? " hit" : " hits");
  return '<details class="file-tree"' + open + ' aria-label="Files containing ' + esc(key) + '">' +
    '<summary><span class="chev">▸</span>' + label + '</summary>' +
    '<div class="tree-body"><div class="tree-root">./</div>' +
    files.map((f, i) => {
      const branch = i === files.length - 1 ? "`--" : "+--";
      // Render the diff once per file, under the first key it contains.
      const fileKeys = Object.keys(f.counts || {}).sort();
      const showDiff = fileKeys.length && fileKeys[0] === key;
      return '<div class="tree-file"><span class="branch">' + branch + '</span><span class="path">' + esc(f.path) + '</span><span class="hit-pill">' + f.counts[key] + '</span></div>' +
        renderPreviews(f, key) + (showDiff ? renderDiff(f) : "");
    }).join("") +
  '</div></details>';
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
    reportResult(body, "Applied", "Preview", dryRun);
    await rememberRun("local", payload, body);
    if (body.applied) await scan();
  } catch (err) {
    show(err.message || "Request failed", "err");
  } finally {
    setReady();
  }
}

async function setDelimiters(e) {
  e.preventDefault();
  const start = delimStart.value;
  const end = delimEnd.value;
  setBusy("updating delimiters");
  rows.innerHTML = skeleton();
  try {
    summary = await postJSON("/api/delimiters", { startDelim: start, endDelim: end });
    previewedOnce = false;
    render();
    show("Delimiters set to " + start.trim() + "KEY" + end.trim() + ". Rescanned with the new pair.", "ok");
  } catch (err) {
    show(err.message || "Failed to set delimiters", "err");
    render();
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
  statusEl.classList.add("busy");
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

function reportResult(body, appliedVerb, previewVerb, wasDryRun) {
  if (body.summary && Array.isArray(body.summary.keys)) {
    summary = body.summary;
    render();
  }
  if (body.applied) {
    setFlow("done");
    show(appliedVerb + " " + body.totalMatches + " replacements across " + body.placeholders + " placeholders.", "ok");
    return;
  }
  if (wasDryRun) {
    previewedOnce = true;
    updateFillProgress();
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
    reportResult(body, "Cloned and applied", "Clone preview", dryRun);
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
    reportResult(body, "Generated and applied", "Generate preview", dryRun);
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
  if (selected.summary) {
    summary = selected.summary;
    previewedOnce = false;
    render();
  }
  Object.entries(selected.values || {}).forEach(([key, value]) => {
    const input = document.querySelector('[data-key="' + CSS.escape(key) + '"]');
    if (input) input.value = value;
  });
  updateFillProgress();
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

// previewActive dry-runs whichever mode is currently selected.
function previewActive() {
  if (activeMode === "clone") cloneApply(true);
  else if (activeMode === "generate") generateApply(true);
  else apply(true);
}

// ---- Keyboard shortcuts: Ctrl+Enter preview · / search · Esc dismiss ----
document.addEventListener("keydown", e => {
  const tag = (e.target.tagName || "").toLowerCase();
  const typing = tag === "input" || tag === "select" || tag === "textarea";
  if (e.key === "Escape") { hideNotice(); return; }
  if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
    e.preventDefault();
    previewActive();
    return;
  }
  if (e.key === "/" && !typing) {
    e.preventDefault();
    presetSearch.focus();
  }
});

document.querySelectorAll("[data-mode]").forEach(b => b.addEventListener("click", () => setMode(b.dataset.mode)));
document.querySelectorAll("[data-repo-type]").forEach(b => b.addEventListener("click", () => setRepoType(b.dataset.repoType)));
document.querySelectorAll("[data-preset-filter]").forEach(b => b.addEventListener("click", () => {
  presetFilter = b.dataset.presetFilter;
  document.querySelectorAll("[data-preset-filter]").forEach(x => x.classList.toggle("active", x.dataset.presetFilter === presetFilter));
  renderSavedRuns();
}));
presetSearch.addEventListener("input", renderSavedRuns);
delimForm.addEventListener("submit", setDelimiters);
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
