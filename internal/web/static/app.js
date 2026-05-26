const rows = document.querySelector("#rows");
const statusEl = document.querySelector("#status");
const notice = document.querySelector("#notice");
const cloneRepo = document.querySelector("#cloneRepo");
const templateSelect = document.querySelector("#templateSelect");
let summary = { keys: [], counts: {}, values: {} };
let repoType = "ssh";

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
    '<div><label>' + esc(k) + '</label>' +
    '<input data-key="' + esc(k) + '" value="' + esc(summary.values[k] || "") + '" autocomplete="off"></div>' +
    '<div class="count">' + (summary.counts[k] || 0) + ' hits</div></div>'
  ).join("");
}

async function apply(dryRun) {
  setBusy(dryRun ? "previewing" : "applying");
  try {
    const body = await postJSON("/api/apply", { values: values(), dryRun });
    reportResult(body, "Applied", "Preview");
    if (body.applied) await scan();
  } catch (err) {
    show(err.message || "Request failed", "err");
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
  if (body.applied) {
    show(appliedVerb + " " + body.totalMatches + " replacements across " + body.placeholders + " placeholders.", "ok");
    return;
  }
  show(previewVerb + ": " + body.totalMatches + " replacements across " + body.placeholders + " placeholders. No files modified.", body.forcedDryRun ? "warn" : "ok");
}

function setMode(mode) {
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
    if (!templates.length) {
      templateSelect.innerHTML = '<option value="">No configured templates</option>';
      return;
    }
    templateSelect.innerHTML = templates.map(t =>
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
  setBusy(dryRun ? "previewing clone" : "cloning");
  try {
    const body = await postJSON("/api/clone", payload);
    reportResult(body, "Cloned and applied", "Clone preview");
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
  setBusy(dryRun ? "previewing generate" : "generating");
  try {
    const body = await postJSON("/api/generate", payload);
    reportResult(body, "Generated and applied", "Generate preview");
    if (body.applied) await scan();
  } catch (err) {
    show(err.message || "Generate failed", "err");
  } finally {
    setReady();
  }
}

document.querySelectorAll("[data-mode]").forEach(b => b.addEventListener("click", () => setMode(b.dataset.mode)));
document.querySelectorAll("[data-repo-type]").forEach(b => b.addEventListener("click", () => setRepoType(b.dataset.repoType)));
document.querySelector("#refresh").addEventListener("click", scan);
document.querySelector("#preview").addEventListener("click", () => apply(true));
document.querySelector("#apply").addEventListener("click", () => apply(false));
document.querySelector("#clonePreview").addEventListener("click", () => cloneApply(true));
document.querySelector("#cloneApply").addEventListener("click", () => cloneApply(false));
document.querySelector("#generatePreview").addEventListener("click", () => generateApply(true));
document.querySelector("#generateApply").addEventListener("click", () => generateApply(false));
loadTemplates();
scan();
