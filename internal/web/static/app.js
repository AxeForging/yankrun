const rows = document.querySelector("#rows");
const statusEl = document.querySelector("#status");
const notice = document.querySelector("#notice");
let summary = { keys: [], counts: {}, values: {} };

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

async function scan() {
  statusEl.textContent = "scanning";
  const r = await fetch("/api/scan");
  if (!r.ok) {
    show("Scan failed", "err");
    statusEl.textContent = "ready";
    return;
  }
  summary = await r.json();
  render();
  statusEl.textContent = "ready";
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
  statusEl.textContent = dryRun ? "previewing" : "applying";
  const r = await fetch("/api/apply", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ values: values(), dryRun })
  });
  const body = await r.json();
  if (!r.ok) {
    show(body.error || "Request failed", "err");
    statusEl.textContent = "ready";
    return;
  }
  if (body.applied) {
    show("Applied " + body.totalMatches + " replacements across " + body.placeholders + " placeholders.", "ok");
    await scan();
  } else {
    show("Preview: " + body.totalMatches + " replacements across " + body.placeholders + " placeholders. No files modified.", body.forcedDryRun ? "warn" : "ok");
  }
  statusEl.textContent = "ready";
}

document.querySelector("#refresh").addEventListener("click", scan);
document.querySelector("#preview").addEventListener("click", () => apply(true));
document.querySelector("#apply").addEventListener("click", () => apply(false));
scan();
