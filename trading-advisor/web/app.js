// Day-Trading Advisor page. Trades and settings live in this page's private claude.ai database.
// Opened anywhere else (e.g. as a local file) it runs in preview mode and saves nothing.
"use strict";

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

let currentPage = "dashboard";
let filter = loadFilter();
let editing = null;   // trade being edited, or null for a new one
let csvData = null;   // {headers, rows} from the chosen CSV file

// ---------- storage ----------
// store.trades / store.settings mirror the database; writes go through these methods.

const store = {
  mode: "loading",     // loading | cloud | preview | locked
  trades: [],
  settings: structuredClone(DEFAULT_SETTINGS),
  db: null, assets: null, downloads: null,
  loaded: { trades: false, settings: false },

  async init() {
    if (!window.claude || typeof window.claude.use !== "function") return this.startPreview();
    const [db, user] = await Promise.all([claude.use("db"), claude.use("user")]);
    if (!db) return this.lock("Sign in to claude.ai to open your trading journal.");
    if (user && !(await user.isOwner().catch(() => true))) return this.lock("This trading journal is private to its owner.");
    this.db = db;
    this.mode = "cloud";
    setStatus("ok", "Saved privately");
    claude.use("assets").then(a => { this.assets = a; });
    claude.use("downloads").then(d => { this.downloads = d; });
    db.collection("trades").onSnapshot(snap => {
      this.trades = snap.docs.map(d => ({ ...d.data(), id: d.id }));
      this.loaded.trades = true;
      dataChanged();
    }, e => storageError(e));
    db.doc("settings/main").onSnapshot(snap => {
      this.settings = { ...structuredClone(DEFAULT_SETTINGS), ...(snap.exists ? snap.data() : {}) };
      this.loaded.settings = true;
      dataChanged();
    }, e => storageError(e));
  },
  startPreview() {
    this.mode = "preview";
    this.loaded = { trades: true, settings: true };
    setStatus("warn", "Preview: nothing is saved");
    dataChanged();
  },
  lock(message) {
    this.mode = "locked";
    setStatus("warn", "Not available");
    $("main").innerHTML = `<div class="card empty"><h2>Can't open the journal</h2><p>${esc(message)}</p></div>`;
  },
  ready() { return this.mode === "preview" || (this.mode === "cloud" && this.loaded.trades && this.loaded.settings); },

  async addTrades(list, onProgress) {
    if (this.mode === "preview") {
      list.forEach(t => this.trades.push({ ...t, id: "p" + Math.random().toString(36).slice(2), created_at: new Date().toISOString() }));
      return dataChanged();
    }
    const col = this.db.collection("trades");
    for (let i = 0; i < list.length; i++) {
      await withRetry(() => col.doc().set({ ...list[i], created_at: new Date().toISOString() }));
      onProgress?.(i + 1, list.length);
    }
  },
  async updateTrade(id, t) {
    if (this.mode === "preview") {
      const i = this.trades.findIndex(x => x.id === id);
      this.trades[i] = { ...this.trades[i], ...t };
      return dataChanged();
    }
    const old = this.trades.find(x => x.id === id) || {};
    await withRetry(() => this.db.doc("trades/" + id).set({ ...t, created_at: old.created_at || new Date().toISOString() }));
  },
  async deleteTrade(id) {
    const t = this.trades.find(x => x.id === id);
    if (this.mode === "preview") {
      this.trades = this.trades.filter(x => x.id !== id);
      return dataChanged();
    }
    await withRetry(() => this.db.doc("trades/" + id).delete());
    if (t?.screenshot) await this.deleteShot(t.screenshot);
  },
  async saveSettings(s) {
    if (this.mode === "preview") { this.settings = s; return dataChanged(); }
    await withRetry(() => this.db.doc("settings/main").set(s));
  },
  async uploadShot(file) {
    if (!this.assets) throw new Error("Screenshots can't be saved here. Open the page from your claude.ai account to add one.");
    const res = await this.assets.upload(file);
    return res.id;
  },
  async deleteShot(id) {
    if (this.assets && id) await this.assets.delete(id).catch(() => {});
  },
  shotUrl(id) { return "/_blob/" + encodeURIComponent(id); },
};

async function withRetry(fn) {
  try { return await fn(); }
  catch (e) {
    if (e && (e.code === "unavailable" || e.code === "store_unavailable")) {
      await new Promise(r => setTimeout(r, 600 + Math.random() * 800));
      return fn();
    }
    throw new Error(storageMessage(e));
  }
}
function storageMessage(e) {
  const code = e && e.code;
  if (code === "quota_exceeded") return "Your journal is full. Delete some old trades, then try again.";
  if (code === "resource_exhausted" || code === "rate_limited") return "Too many saves at once. Wait a few seconds and try again.";
  if (code === "invalid_argument") return "Only the owner of this journal can save changes.";
  if (code === "too_large") return "That screenshot is too big (20 MB max).";
  if (code === "unsupported_type") return "Screenshots must be PNG, JPG, GIF or WEBP images.";
  return (e && e.message) || "Something went wrong while saving. Try again.";
}
function storageError(e) {
  setStatus("warn", "Connection lost");
  toast(e && e.code === "revoked" ? "Access to this journal changed. Reload the page." : "Lost the connection to your saved trades. Reload the page.", 8000);
}
function setStatus(kind, text) {
  const s = $("#status");
  if (!s) return;
  s.className = "status " + kind;
  $("#status-text").textContent = text;
}

// Re-render pages that only show data. Forms are left alone so typing isn't wiped.
function dataChanged() {
  if (!store.ready()) return;
  if (["dashboard", "trades", "mistakes"].includes(currentPage)) render();
  else if (currentPage === "rules" && !$("#rules-form").dataset.dirty) renderRules();
}

// ---------- helpers ----------

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}
function fmtMoney(x, signed = true) {
  if (x === null || x === undefined || isNaN(x)) return "–";
  const s = "$" + Math.abs(x).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  if (!signed) return (x < 0 ? "-" : "") + s;
  return (x > 0 ? "+" : x < 0 ? "-" : "") + s;
}
function pnlSpan(x) {
  if (x === null || x === undefined) return '<span class="muted">open</span>';
  return `<span class="num ${x > 0 ? "pos" : x < 0 ? "neg" : ""}">${fmtMoney(x)}</span>`;
}
const pct = (x, d = 0) => (x === null || x === undefined || isNaN(x)) ? "–" : (x * 100).toFixed(d) + "%";
const num = (x, d = 2) => (x === null || x === undefined || isNaN(x)) ? "–" : Number(x).toFixed(d);
function holdLabel(m) {
  if (m === null || m === undefined) return "–";
  if (m < 90) return Math.round(m) + " min";
  if (m < 1440) return (m / 60).toFixed(1) + " h";
  return (m / 1440).toFixed(1) + " d";
}
function fmtDT(s) {
  if (!s) return "–";
  const [d, t = ""] = String(s).replace(" ", "T").split("T");
  return `${d} ${t.slice(0, 5)}`;
}
// Current US market (Eastern) time as "YYYY-MM-DDTHH:MM", whatever the viewer's own time zone.
function nowET() {
  try {
    return new Date().toLocaleString("sv-SE", { timeZone: "America/New_York" }).replace(" ", "T").slice(0, 16);
  } catch {
    const d = new Date();
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset());
    return d.toISOString().slice(0, 16);
  }
}
function toast(msg, ms = 2600) {
  const t = $("#toast");
  t.textContent = msg;
  t.style.display = "block";
  clearTimeout(toast._t);
  toast._t = setTimeout(() => (t.style.display = "none"), ms);
}
function icon(level) {
  const ch = { green: "✓", good: "✓", yellow: "!", red: "✕", bad: "✕", info: "i", gray: "–" }[level] || "i";
  return `<span class="ico ${level}" aria-hidden="true">${ch}</span>`;
}
function loadFilter() {
  try { return Object.assign({ account: "all", from: "", to: "" }, JSON.parse(localStorage.getItem("advisor-filter") || "{}")); }
  catch { return { account: "all", from: "", to: "" }; }
}
function saveFilter() {
  try { localStorage.setItem("advisor-filter", JSON.stringify(filter)); } catch { /* storage blocked */ }
}
const FLAG_SHORT = {
  over_risk: "Over max risk", no_stop: "No stop", bad_stop: "Bad stop", low_rr: "Low R:R", revenge: "Revenge",
  overtrading: "Overtrading", after_loss_limit: "After loss limit", no_trade_window: "No-trade window", held_loser: "Held loser",
};
const S = () => store.settings;

// Inline yes/no confirmation (the page can't use confirm() dialogs).
function askConfirm(box, message, yesLabel, onYes) {
  box.innerHTML = `<div class="confirm-box"><b>${esc(message)}</b><div class="row">
    <button type="button" class="danger" data-yes>${esc(yesLabel)}</button><button type="button" data-no>Cancel</button></div></div>`;
  $("[data-no]", box).addEventListener("click", () => (box.innerHTML = ""));
  $("[data-yes]", box).addEventListener("click", async () => {
    box.innerHTML = "";
    try { await onYes(); } catch (e) { toast(e.message, 5000); }
  });
}

async function exportCSV() {
  const csv = tradesToCSV([...store.trades].sort((a, b) => (a.entry_time < b.entry_time ? -1 : 1)));
  if (!store.downloads) { toast("Downloads aren't available here. Open the page from your claude.ai account.", 5000); return; }
  try {
    const res = await store.downloads.save({ filename: `my-trades-${nowET().slice(0, 10)}.csv`, data: csv });
    if (res.status === "saved") toast("Backup saved");
  } catch (e) {
    if (e && e.code !== "cancelled" && e.code !== "declined") toast("Couldn't save the file. Try again.", 4000);
  }
}

// ---------- navigation ----------

function go(page, arg) {
  currentPage = page;
  $$(".page").forEach(p => p.classList.toggle("active", p.id === "page-" + page));
  $$("#tabs button").forEach(b => b.classList.toggle("active", b.dataset.page === page));
  $("#tabs button.active")?.scrollIntoView({ block: "nearest", inline: "nearest" });
  window.scrollTo(0, 0);
  render(arg);
}
function render(arg) {
  if (!store.ready()) {
    const slot = { dashboard: "#dash-body", trades: "#trades-body", mistakes: "#mistakes-body" }[currentPage];
    if (slot) $(slot).innerHTML = '<div class="card empty">Loading your trades…</div>';
    return;
  }
  try {
    ({
      dashboard: renderDashboard, trades: renderTrades, edit: () => openEditor(arg),
      import: () => {}, mistakes: renderMistakes, pretrade: preparePretrade, rules: renderRules,
    })[currentPage]();
  } catch (e) {
    console.error(e);
    toast(e.message, 5000);
  }
}

// ---------- filter bar ----------

function setupFilters() {
  $$(".filter-slot").forEach(slot => {
    slot.appendChild($("#filter-tpl").content.cloneNode(true));
    $$("[data-f=account] button", slot).forEach(b => b.addEventListener("click", () => { filter.account = b.dataset.v; filterChanged(); }));
    $("[data-f=from]", slot).addEventListener("change", e => { filter.from = e.target.value; filterChanged(); });
    $("[data-f=to]", slot).addEventListener("change", e => { filter.to = e.target.value; filterChanged(); });
    $("[data-f=clear]", slot).addEventListener("click", () => { filter.from = filter.to = ""; filterChanged(); });
  });
  syncFilters();
}
function syncFilters() {
  $$(".filter-slot").forEach(slot => {
    $$("[data-f=account] button", slot).forEach(b => b.classList.toggle("on", b.dataset.v === filter.account));
    $("[data-f=from]", slot).value = filter.from;
    $("[data-f=to]", slot).value = filter.to;
  });
}
function filterChanged() { saveFilter(); syncFilters(); render(); }

// ---------- dashboard ----------

function renderDashboard() {
  const body = $("#dash-body");
  const r = fullReport(store.trades, S(), filter);
  const s = r.summary;
  if (!s.trades) {
    body.innerHTML = `<div class="card empty"><h2>No closed trades yet</h2>
      <p>Add a trade, or import a CSV from your broker. To see how it works first, load the sample trades on the
      <button type="button" class="link-btn" data-page="import">Import CSV</button> page.</p>
      ${r.open_trades ? `<p>You have ${r.open_trades} open trade(s). They're counted once you add an exit.</p>` : ""}</div>`;
    return;
  }
  const pf = s.profit_factor === null ? (s.wins ? "No losses" : "–") : num(s.profit_factor);
  const cs = s.current_streak;
  const cur = cs.kind ? `${cs.length} ${cs.kind === "loss" ? (cs.length > 1 ? "losses" : "loss") : (cs.length > 1 ? "wins" : "win")}` : "–";
  const tile = (k, v, sub = "") => `<div class="tile"><div class="k">${k}</div><div class="v">${v}</div><div class="s">${sub}</div></div>`;
  body.innerHTML = `
    <div class="tiles">
      ${tile("Total P&L", pnlSpan(s.total_pnl), `${s.trades} closed trades${r.open_trades ? ` · ${r.open_trades} open` : ""}`)}
      ${tile("Win rate", pct(s.win_rate), `${s.wins} wins · ${s.losses} losses${s.breakeven ? ` · ${s.breakeven} even` : ""}`)}
      ${tile("Avg win / avg loss", `<span class="pos">${fmtMoney(s.avg_win, false)}</span> / <span class="neg">${fmtMoney(s.avg_loss, false)}</span>`, s.avg_win && s.avg_loss ? `wins are ${num(s.avg_win / s.avg_loss)}× losses` : "")}
      ${tile("Expectancy", pnlSpan(s.expectancy), "average result per trade")}
      ${tile("Profit factor", pf, "gross wins ÷ gross losses (above 1 = profitable)")}
      ${tile("Max drawdown", `<span class="neg">${fmtMoney(-s.max_drawdown)}</span>`, s.max_drawdown ? `${num(s.max_drawdown_pct, 1)}% from peak` : "no drawdown")}
      ${tile("Streaks", `${s.longest_win_streak}W / ${s.longest_loss_streak}L`, `longest win / loss streak · now: ${cur}`)}
      ${tile("Average R", s.avg_r === null ? "–" : num(s.avg_r) + "R", "result ÷ risk to stop (trades with a stop)")}
    </div>
    <div class="card">
      <div class="row"><h2 style="margin:0">Equity curve</h2><span class="spacer"></span>
        <small>Starts at your account size (${fmtMoney(S().account_size, false)}), one point per closed trade</small></div>
      <div class="chart" id="equity-chart"></div>
    </div>
    <div class="card">
      <h2>What the advisor sees</h2>
      <ul class="insights">${r.insights.map(i => `<li class="${i.tone}">${icon(i.tone)}<span>${esc(i.text)}</span></li>`).join("")}</ul>
    </div>
    <h2 style="margin-top:1.5rem">Breakdowns</h2>
    <p class="muted">Blue bars are total profit, red bars are total loss. Groups with fewer than ${MIN_SAMPLE} trades are greyed out because they're too small to trust.</p>
    <div class="grid grid-2">${Object.values(r.breakdowns).map(breakdownCard).join("")}</div>`;
  drawEquity($("#equity-chart"), s.equity_curve);
}

function breakdownCard(b) {
  const max = Math.max(1, ...b.rows.map(r => Math.abs(r.total_pnl)));
  const rows = b.rows.map(r => {
    const w = Math.abs(r.total_pnl) / max * 50;
    return `<tr style="${r.count < MIN_SAMPLE ? "opacity:.55" : ""}" title="${esc(r.label)}: ${r.count} trades, ${pct(r.win_rate)} wins, ${fmtMoney(r.total_pnl)} total">
      <td>${esc(r.label)}</td><td class="r num">${r.count}</td><td class="r num">${pct(r.win_rate)}</td>
      <td class="r">${pnlSpan(r.avg_pnl)}</td><td class="r">${pnlSpan(r.total_pnl)}</td>
      <td><div class="dbar"><div class="axis"></div><div class="b ${r.total_pnl >= 0 ? "p" : "n"}" style="width:${w}%"></div></div></td></tr>`;
  }).join("");
  return `<div class="card"><h3>${esc(b.title)}</h3><div class="table-wrap"><table>
    <thead><tr><th>Group</th><th class="r">Trades</th><th class="r">Win %</th><th class="r">Avg</th><th class="r">Total</th><th></th></tr></thead>
    <tbody>${rows || '<tr><td colspan="6" class="muted">No data</td></tr>'}</tbody></table></div></div>`;
}

function drawEquity(el, curve) {
  const W = 900, H = 300, L = 64, R = 14, T = 14, B = 30;
  const vals = curve.map(p => p.equity);
  let lo = Math.min(...vals), hi = Math.max(...vals);
  const pad = (hi - lo) * 0.08 || Math.max(1, hi * 0.01);
  lo -= pad; hi += pad;
  const x = i => L + (curve.length === 1 ? 0 : i / (curve.length - 1)) * (W - L - R);
  const y = v => T + (hi - v) / (hi - lo) * (H - T - B);
  let grid = "";
  for (let k = 0; k <= 5; k++) {
    const v = lo + (hi - lo) * k / 5;
    grid += `<line class="gridline" x1="${L}" x2="${W - R}" y1="${y(v)}" y2="${y(v)}"/>
      <text class="axis-label" x="${L - 8}" y="${y(v) + 4}" text-anchor="end">${"$" + Math.round(v).toLocaleString("en-US")}</text>`;
  }
  const xticks = [0, Math.floor((curve.length - 1) / 2), curve.length - 1].filter((v, i, a) => a.indexOf(v) === i);
  for (const i of xticks) {
    grid += `<text class="axis-label" x="${x(i)}" y="${H - 8}" text-anchor="${i === 0 ? "start" : i === curve.length - 1 ? "end" : "middle"}">${i === 0 ? "start" : "trade " + i}</text>`;
  }
  const path = curve.map((p, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(p.equity).toFixed(1)}`).join("");
  el.innerHTML = `<svg viewBox="0 0 ${W} ${H}" role="img" aria-label="Equity curve">
      ${grid}
      <line class="baseline" x1="${L}" x2="${W - R}" y1="${y(curve[0].equity)}" y2="${y(curve[0].equity)}"/>
      <path class="line" d="${path}"/>
      <line class="hover-line" x1="0" x2="0" y1="${T}" y2="${H - B}" visibility="hidden"/>
      <circle class="hover-dot" r="5" visibility="hidden"/>
      <rect x="${L}" y="0" width="${W - L - R}" height="${H}" fill="transparent"/>
    </svg><div class="tooltip"></div>`;
  const svg = $("svg", el), tip = $(".tooltip", el), hl = $(".hover-line", el), dot = $(".hover-dot", el);
  const move = ev => {
    const rect = svg.getBoundingClientRect();
    const px = (ev.clientX - rect.left) / rect.width * W;
    const i = Math.max(0, Math.min(curve.length - 1, Math.round((px - L) / (W - L - R) * (curve.length - 1))));
    const p = curve[i];
    hl.setAttribute("x1", x(i)); hl.setAttribute("x2", x(i)); hl.setAttribute("visibility", "visible");
    dot.setAttribute("cx", x(i)); dot.setAttribute("cy", y(p.equity)); dot.setAttribute("visibility", "visible");
    tip.innerHTML = i === 0 ? `<b>Start</b><br>${fmtMoney(p.equity, false)}`
      : `<b>${esc(p.label)}</b> · ${fmtDT(p.time)}<br>Trade: ${pnlSpan(p.pnl)}<br>Equity: <b>${fmtMoney(p.equity, false)}</b>`;
    tip.style.display = "block";
    const sx = x(i) / W * rect.width, sy = y(p.equity) / H * rect.height, tw = tip.offsetWidth;
    tip.style.left = Math.max(0, Math.min(rect.width - tw, sx - tw / 2)) + "px";
    tip.style.top = Math.max(0, sy - tip.offsetHeight - 12) + "px";
  };
  const leave = () => { tip.style.display = "none"; hl.setAttribute("visibility", "hidden"); dot.setAttribute("visibility", "hidden"); };
  svg.addEventListener("pointermove", move);
  svg.addEventListener("pointerdown", move);
  svg.addEventListener("pointerleave", leave);
}

// ---------- trades list ----------

function renderTrades() {
  const body = $("#trades-body");
  const trades = filterTrades(prepare(store.trades, S()), filter)
    .sort((a, b) => b.entry_dt.ms - a.entry_dt.ms);
  if (!trades.length) {
    body.innerHTML = `<div class="empty">No trades match. <button type="button" class="link-btn" data-page="edit">Add one</button>
      or <button type="button" class="link-btn" data-page="import">import a CSV</button>.</div>`;
    return;
  }
  body.innerHTML = `<div class="table-wrap"><table>
    <thead><tr><th>Entry</th><th>Ticker</th><th>Side</th><th>Acct</th><th class="r">Shares</th><th class="r">Entry</th><th class="r">Exit</th>
      <th class="r">P&amp;L</th><th class="r">R</th><th class="r">Hold</th><th>Setup</th><th>Plan</th><th>Flags</th></tr></thead>
    <tbody>${trades.map(t => `<tr class="clickable" data-id="${esc(t.id)}">
      <td class="num">${fmtDT(t.entry_time)}</td><td><b>${esc(t.ticker)}</b></td><td>${t.direction}</td>
      <td><span class="badge ${t.account_type}">${t.account_type === "real" ? "real" : "paper"}</span></td>
      <td class="r num">${t.shares}</td><td class="r num">${num(t.entry_price)}</td><td class="r num">${t.exit_price === null ? "–" : num(t.exit_price)}</td>
      <td class="r">${pnlSpan(t.pnl)}</td><td class="r num">${t.r_multiple === null ? "–" : num(t.r_multiple, 1)}</td>
      <td class="r num">${holdLabel(t.hold_minutes)}</td><td>${esc(t.setup || "")}</td>
      <td>${t.followed_plan === 1 ? "✓ yes" : t.followed_plan === 0 ? "✕ no" : ""}</td>
      <td>${t.flags.map(f => `<span class="flag">${FLAG_SHORT[f]}</span>`).join("")}${t.screenshot ? ' <span title="Has a screenshot">📷</span>' : ""}</td></tr>`).join("")}
    </tbody></table></div>`;
  $$("tr[data-id]", body).forEach(tr => tr.addEventListener("click", () => go("edit", tr.dataset.id)));
}

// ---------- add / edit trade ----------

function fillSelect(sel, items, current, blank = "—") {
  const list = [...items];
  if (current && !list.includes(current)) list.push(current);
  sel.innerHTML = `<option value="">${blank}</option>` + list.map(i => `<option>${esc(i)}</option>`).join("");
  sel.value = current || "";
}

function renderMistakeChips(selected) {
  const box = $("#mistake-chips");
  box.innerHTML = S().mistake_tags.map(m =>
    `<button type="button" class="chip-toggle ${selected.includes(m) ? "on" : ""}" data-m="${esc(m)}">${esc(m)}</button>`).join("");
  $$("button", box).forEach(b => b.addEventListener("click", () => b.classList.toggle("on")));
  $("[name=mistakes_extra]").value = selected.filter(m => !S().mistake_tags.includes(m)).join(", ");
}

function openEditor(id) {
  const f = $("#trade-form");
  f.reset();
  $("#shot-input").value = "";
  $("#delete-confirm").innerHTML = "";
  editing = id ? store.trades.find(t => t.id === id) || null : null;
  const t = editing || { direction: "long", account_type: filter.account === "real" ? "real" : "paper", fees: 0 };
  $("#edit-title").textContent = editing ? `Edit trade: ${editing.ticker}` : "Add a trade";
  $("#delete-trade").hidden = !editing;
  for (const k of ["ticker", "direction", "account_type", "shares", "entry_price", "exit_price", "stop_loss", "target", "fees", "reason", "notes"]) {
    f.elements[k].value = t[k] ?? "";
  }
  f.elements.entry_time.value = (t.entry_time || nowET()).replace(" ", "T").slice(0, 16);
  f.elements.exit_time.value = (t.exit_time || "").replace(" ", "T").slice(0, 16);
  fillSelect(f.elements.setup, S().setups, t.setup);
  fillSelect(f.elements.emotion_before, S().emotions, t.emotion_before);
  fillSelect(f.elements.emotion_after, S().emotions, t.emotion_after);
  const fp = t.followed_plan === 1 ? "1" : t.followed_plan === 0 ? "0" : "";
  $$("[name=followed_plan]", f).forEach(r => (r.checked = r.value === fp));
  renderMistakeChips(String(t.mistakes || "").split(",").map(s => s.trim()).filter(Boolean));
  showScreenshot(t.screenshot);
  liveCalc();
}

function showScreenshot(id) {
  const box = $("#shot-preview");
  if (!id) { box.innerHTML = ""; return; }
  box.innerHTML = `<a href="${store.shotUrl(id)}" target="_blank" rel="noopener"><img class="shot" src="${store.shotUrl(id)}" alt="Trade screenshot"></a>
    <button type="button" class="small danger" id="shot-remove" style="margin-top:.4rem">Remove screenshot</button>`;
  $("#shot-remove").addEventListener("click", async () => {
    try {
      const { id: _drop, ...rest } = editing;
      await store.updateTrade(editing.id, { ...stripComputed(rest), screenshot: null });
      await store.deleteShot(id);
      editing = { ...editing, screenshot: null };
      showScreenshot(null);
      toast("Screenshot removed");
    } catch (e) { toast(e.message, 5000); }
  });
}
function stripComputed(t) {
  const out = {};
  for (const k of TRADE_FIELDS) out[k] = t[k] ?? null;
  return out;
}

function liveCalc() {
  const f = $("#trade-form").elements;
  const v = k => parseFloat(f[k].value);
  const dir = f.direction.value === "long" ? 1 : -1;
  const entry = v("entry_price"), shares = v("shares"), stop = v("stop_loss"), target = v("target"), exit = v("exit_price"), fees = v("fees") || 0;
  const parts = [];
  if (entry && shares) parts.push(`Position: <b>${fmtMoney(entry * shares, false)}</b>`);
  if (entry && shares && stop) {
    const rps = (entry - stop) * dir;
    if (rps > 0) {
      const risk = rps * shares, p = risk / S().account_size * 100, max = S().account_size * S().max_risk_pct / 100;
      parts.push(`Risk: <b class="${risk > max ? "neg" : ""}">${fmtMoney(risk, false)}</b> (${p.toFixed(2)}% of account${risk > max ? ", over your max!" : ""})`);
      if (target) {
        const rr = (target - entry) * dir / rps;
        parts.push(`Reward/risk: <b class="${rr < S().min_rr ? "neg" : ""}">${rr > 0 ? rr.toFixed(2) + " : 1" : "target on wrong side"}</b>`);
      }
    } else parts.push('<b class="neg">Stop is on the wrong side of entry</b>');
  }
  if (entry && shares && exit) parts.push(`P&L after fees: ${pnlSpan((exit - entry) * dir * shares - fees)}`);
  $("#live-calc").innerHTML = parts.length ? parts.map(p => `<span>${p}</span>`).join("") : '<span class="muted">Fill in prices and shares to see risk and P&amp;L.</span>';
}

async function saveTrade(ev) {
  ev.preventDefault();
  const f = ev.target.elements;
  const mistakes = [...$$("#mistake-chips .on").map(b => b.dataset.m), ...f.mistakes_extra.value.split(",").map(s => s.trim()).filter(Boolean)];
  const data = {};
  for (const k of ["ticker", "direction", "account_type", "shares", "entry_price", "exit_price", "stop_loss", "target", "fees",
    "entry_time", "exit_time", "setup", "emotion_before", "emotion_after", "reason", "notes"]) data[k] = f[k].value;
  data.followed_plan = $$("[name=followed_plan]").find(r => r.checked)?.value ?? "";
  data.mistakes = mistakes.join(", ");
  data.screenshot = editing?.screenshot || null;
  const btn = $("#save-btn");
  btn.disabled = true;
  try {
    const trade = cleanTrade(data);
    const file = $("#shot-input").files[0];
    let oldShot = null;
    if (file) {
      if (file.size > 20 * 1024 * 1024) throw new Error("That screenshot is too big (20 MB max).");
      btn.textContent = "Uploading screenshot…";
      oldShot = trade.screenshot;
      trade.screenshot = await store.uploadShot(file).catch(e => { throw new Error(storageMessage(e)); });
    }
    btn.textContent = "Saving…";
    if (editing) await store.updateTrade(editing.id, trade);
    else await store.addTrades([trade]);
    if (oldShot) await store.deleteShot(oldShot);
    toast(editing ? "Trade updated" : "Trade saved");
    go("trades");
  } catch (e) {
    toast(e.message, 5000);
  } finally {
    btn.disabled = false;
    btn.textContent = "Save trade";
  }
}

// ---------- CSV import ----------

const IMPORT_FIELDS = {
  trades: [
    ["ticker", "Ticker / symbol", true, [/^(symbol|ticker)$/i, /symbol|ticker|stock|instrument|security/i]],
    ["direction", "Long / short (side)", false, [/^(side|direction)$/i, /side|direction|long.?short|position.?type|^type$|action/i]],
    ["entry_date", "Entry date (or date+time)", true, [/(open|entry|buy).*date/i, /^(open(ed)?|entry)( time)?$/i, /^date/i, /date/i]],
    ["entry_time", "Entry time (if separate)", false, [/(open|entry|buy).*time$/i, /^time$/i]],
    ["entry_price", "Entry price", true, [/(open|entry|buy).*price/i, /avg.*(open|entry|buy)/i, /^(entry|open)$/i]],
    ["exit_date", "Exit date (or date+time)", false, [/(close|exit|sell).*date/i, /^(close[d]?|exit)( time)?$/i]],
    ["exit_time", "Exit time (if separate)", false, [/(close|exit|sell).*time$/i]],
    ["exit_price", "Exit price", false, [/(close|exit|sell).*price/i, /avg.*(close|exit|sell)/i, /^(exit|close)$/i]],
    ["shares", "Shares / quantity", true, [/^(qty|quantity|shares|size)$/i, /qty|quantity|shares|size|volume/i]],
    ["stop_loss", "Stop-loss", false, [/stop/i]],
    ["target", "Target", false, [/target|take.?profit|^tp$/i]],
    ["fees", "Fees / commission", false, [/commission|comm\b|^fees?$/i, /fee/i]],
    ["fees2", "Other fees", false, [/fee|reg|sec|taf|ecn/i]],
    ["setup", "Setup", false, [/setup|strategy|tag/i]],
    ["notes", "Notes", false, [/note|comment|memo/i]],
  ],
  fills: [
    ["ticker", "Ticker / symbol", true, [/^(symbol|ticker)$/i, /symbol|ticker|stock|instrument|security/i]],
    ["side", "Buy / sell (side)", false, [/^(side|action)$/i, /side|action|buy.?sell|b\/s|transaction|type/i]],
    ["date", "Date (or date+time)", true, [/date.?time|time.?stamp|exec.*time|fill.*time/i, /date/i, /time/i]],
    ["time", "Time (if separate)", false, [/^time$/i, /exec.*time$/i]],
    ["quantity", "Quantity", true, [/^(qty|quantity|shares|size)$/i, /qty|quantity|shares|size|filled/i]],
    ["price", "Price", true, [/^(price|fill price|exec price|avg price)$/i, /price/i]],
    ["fees", "Fees / commission", false, [/commission|comm\b|^fees?$/i, /fee/i]],
    ["fees2", "Other fees", false, [/fee|reg|sec|taf|ecn/i]],
  ],
};
const importMode = () => $("[name=imp-mode]:checked").value;

function guessMapping(headers, fields) {
  const used = new Set(), map = {};
  for (const [key, , , patterns] of fields) {
    for (const re of patterns) {
      const h = headers.find(h => !used.has(h) && re.test(h));
      if (h) { map[key] = h; used.add(h); break; }
    }
  }
  return map;
}

function renderMapping() {
  if (!csvData) return;
  const fields = IMPORT_FIELDS[importMode()];
  const guess = guessMapping(csvData.headers, fields);
  $("#imp-map").innerHTML = fields.map(([key, label, req]) => `
    <label class="field"><span>${label}${req ? " *" : ""}</span>
      <select data-key="${key}"><option value="">(none)</option>${csvData.headers.map(h => `<option ${guess[key] === h ? "selected" : ""}>${esc(h)}</option>`).join("")}</select>
    </label>`).join("");
  $("#imp-sample").innerHTML = `<small>First rows of ${esc(csvData.name)} (${csvData.rows.length} rows total):</small><table><thead><tr>${csvData.headers.map(h => `<th>${esc(h)}</th>`).join("")}</tr></thead>
    <tbody>${csvData.rows.slice(0, 3).map(r => `<tr>${csvData.headers.map(h => `<td>${esc(r[h])}</td>`).join("")}</tr>`).join("")}</tbody></table>`;
  $("#imp-map-card").hidden = false;
  $("#imp-preview-card").hidden = true;
}

function mappedRows() {
  const map = {};
  $$("#imp-map select").forEach(s => { if (s.value) map[s.dataset.key] = s.value; });
  const missing = IMPORT_FIELDS[importMode()].filter(([k, , req]) => req && !map[k]).map(([, label]) => label);
  if (missing.length) throw new Error("Please choose a column for: " + missing.join(", "));
  return csvData.rows.map(r => Object.fromEntries(Object.entries(map).map(([k, h]) => [k, r[h]])));
}

function importOptions() {
  return { mode: importMode(), account_type: $("#imp-account").value, tz_offset_hours: $("#imp-tz").value, default_direction: $("#imp-default-dir").value };
}

function previewImport() {
  let r;
  try { r = importRows(mappedRows(), importOptions(), store.trades); } catch (e) { toast(e.message, 5000); return; }
  const list = (items, title) => items.length ? `<details ${items.length < 6 ? "open" : ""}><summary>${title} (${items.length})</summary><ul class="err-list">${
    items.map(e => `<li>${e.row ? `Row ${e.row}: ` : ""}${esc(e.message)}</li>`).join("")}</ul></details>` : "";
  $("#imp-preview").innerHTML = `
    <p><b>${r.trades.length}</b> trade(s) ready to import as <b>${importOptions().account_type === "real" ? "real-money" : "paper"}</b> trades${r.errors.length ? `, <b class="neg">${r.errors.length}</b> row(s) couldn't be read and will be skipped` : ""}.</p>
    ${list(r.errors, "Rows with problems")}${list(r.warnings, "Notes")}
    <div class="table-wrap" style="margin:.8rem 0"><table><thead><tr><th>Ticker</th><th>Side</th><th>Entry</th><th class="r">Entry $</th><th>Exit</th><th class="r">Exit $</th><th class="r">Shares</th><th class="r">Stop</th><th class="r">Fees</th></tr></thead>
    <tbody>${r.trades.slice(0, 50).map(t => `<tr><td>${esc(t.ticker)}</td><td>${t.direction}</td><td>${fmtDT(t.entry_time)}</td><td class="r num">${num(t.entry_price)}</td>
      <td>${fmtDT(t.exit_time)}</td><td class="r num">${t.exit_price === null ? "–" : num(t.exit_price)}</td><td class="r num">${t.shares}</td>
      <td class="r num">${t.stop_loss === null ? "–" : num(t.stop_loss)}</td><td class="r num">${num(t.fees)}</td></tr>`).join("")}</tbody></table></div>
    ${r.trades.length > 50 ? `<p class="muted">Showing the first 50.</p>` : ""}
    ${r.trades.some(t => t.stop_loss === null) ? `<p class="notice">Some trades have no stop-loss in the file, so they'll be flagged "No stop". You can add stops later by editing each trade.</p>` : ""}
    <button type="button" class="primary" id="imp-commit" ${r.trades.length ? "" : "disabled"}>Import ${r.trades.length} trade(s)</button>`;
  $("#imp-commit").addEventListener("click", async e => {
    const btn = e.target;
    btn.disabled = true;
    try {
      await store.addTrades(r.trades, (i, n) => (btn.textContent = `Saving ${i} of ${n}…`));
      toast(`Imported ${r.trades.length} trades`);
      csvData = null;
      $("#csv-input").value = "";
      $("#imp-map-card").hidden = $("#imp-preview-card").hidden = true;
      go("dashboard");
    } catch (err) {
      toast(err.message + " Trades saved before the problem were kept; import the file again to add the rest.", 8000);
      btn.disabled = false;
    }
  });
  $("#imp-preview-card").hidden = false;
  $("#imp-preview-card").scrollIntoView({ behavior: "smooth" });
}

async function loadSample(name, mode) {
  try {
    const res = await fetch(name);
    if (!res.ok) throw new Error();
    $(`[name=imp-mode][value=${mode}]`).checked = true;
    $("#imp-account").value = "paper";
    $("#imp-tz").value = "0";
    csvData = { ...parseCSV(await res.text()), name };
    renderMapping();
  } catch {
    toast("Couldn't load the sample file.", 4000);
  }
}

// ---------- mistakes ----------

function renderMistakes() {
  const body = $("#mistakes-body");
  const r = fullReport(store.trades, S(), filter);
  const m = r.mistakes;
  const byId = Object.fromEntries(r.trades.map(t => [t.id, t]));
  if (!m.rows.length) {
    body.innerHTML = `<div class="card empty"><h2>No mistakes found</h2><p>${r.summary.trades ? "Nice: none of your closed trades broke a rule." : "Add some trades first."}</p></div>`;
    return;
  }
  const worst = Math.max(1, ...m.rows.map(x => Math.abs(x.total_pnl)));
  const c = m.clean;
  body.innerHTML = `
    <div class="card">
      <h2>Which mistakes cost you the most</h2>
      <p class="muted">"Net result" is the total profit or loss of every trade that had this mistake. A trade with two mistakes counts in both rows.
        ${c ? `For comparison, your <b>${c.count}</b> trades with no mistakes averaged ${pnlSpan(c.avg_pnl)} each.` : ""}</p>
      <div class="table-wrap"><table>
        <thead><tr><th>Mistake</th><th>Found by</th><th class="r">Trades</th><th class="r">Avg result</th><th class="r">Net result</th><th></th></tr></thead>
        <tbody>${m.rows.map(x => `<tr title="${esc(x.explain)}"><td><b>${esc(x.label)}</b><br><small>${esc(x.explain)}</small></td>
          <td>${x.source === "automatic" ? "Advisor" : "Your journal"}</td><td class="r num">${x.count}</td>
          <td class="r">${pnlSpan(x.avg_pnl)}</td><td class="r">${pnlSpan(x.total_pnl)}</td>
          <td style="width:170px"><div class="hbar ${x.total_pnl >= 0 ? "p" : ""}" style="width:${Math.abs(x.total_pnl) / worst * 100}%"></div></td></tr>`).join("")}</tbody>
      </table></div>
    </div>
    <div class="grid grid-2">${m.rows.filter(x => x.source === "automatic").map(x => `
      <div class="card"><h3>${esc(x.label)} <small>(${x.count})</small></h3>
        <div class="table-wrap"><table><tbody>${x.trade_ids.slice().reverse().map(id => byId[id]).filter(Boolean).map(t =>
          `<tr class="clickable" data-id="${esc(t.id)}"><td class="num">${fmtDT(t.entry_time)}</td><td><b>${esc(t.ticker)}</b> ${t.direction}</td>
           <td class="r num">${t.risk_dollars ? "risk " + fmtMoney(t.risk_dollars, false) : ""}</td><td class="r">${pnlSpan(t.pnl)}</td></tr>`).join("")}</tbody></table></div>
      </div>`).join("")}</div>`;
  $$("tr[data-id]", body).forEach(tr => tr.addEventListener("click", () => go("edit", tr.dataset.id)));
}

// ---------- pre-trade check ----------

function preparePretrade() {
  const f = $("#pt-form").elements;
  fillSelect(f.setup, S().setups, f.setup.value, "(any)");
  if (!f.time.value) f.time.value = nowET();
}

function runPretrade(ev) {
  ev.preventDefault();
  const f = ev.target.elements;
  const plan = Object.fromEntries(["ticker", "direction", "entry", "stop", "target", "shares", "setup", "account_type", "time"].map(k => [k, f[k].value]));
  let r;
  try { r = preTradeCheck(plan, store.trades, S(), nowET()); } catch (e) { toast(e.message, 5000); return; }
  const n = r.numbers;
  const verdictText = {
    red: "Stop. This trade breaks one or more of your rules.",
    yellow: "Caution. Nothing is broken outright, but check the warnings.",
    green: "This trade fits your rules.",
  }[r.verdict];
  const tile = (k, v, s = "") => `<div class="tile"><div class="k">${k}</div><div class="v">${v}</div><div class="s">${s}</div></div>`;
  $("#pt-result").innerHTML = `
    <div class="verdict ${r.verdict}">${icon(r.verdict)}<span>${verdictText}</span></div>
    <div class="tiles">
      ${tile("Dollar risk", n.risk_dollars ? fmtMoney(n.risk_dollars, false) : "–", n.risk_dollars ? `if the stop is hit · max ${fmtMoney(n.max_risk_dollars, false)}` : "needs a valid stop")}
      ${tile("% of account", n.risk_pct === null ? "–" : n.risk_pct.toFixed(2) + "%", `your max is ${S().max_risk_pct}%`)}
      ${tile("Reward / risk", n.rr === null ? "–" : n.rr.toFixed(2) + " : 1", `your minimum is ${S().min_rr} : 1`)}
      ${tile("Potential reward", n.reward_dollars ? fmtMoney(n.reward_dollars, false) : "–", "if the target is hit")}
      ${tile("Position size", fmtMoney(n.position_value, false), n.position_pct === null ? "" : `${n.position_pct.toFixed(0)}% of your account`)}
      ${tile("Max shares for your risk", n.suggested_max_shares ?? "–", "shares you can buy within your max risk")}
    </div>
    <div class="card"><h3>Rule checks</h3>${r.checks.map(c => `<div class="check ${c.level}">${icon(c.level)}<div><b>${esc(c.title)}</b><span>${esc(c.detail)}</span></div></div>`).join("")}
      <small>Today so far (${plan.account_type === "real" ? "real money" : "paper"}): ${n.trades_today} trade(s), ${fmtMoney(n.pnl_today)}</small></div>
    <div class="card"><h3>How similar past trades went</h3>
      <div class="table-wrap"><table><thead><tr><th></th><th>Your past trades</th><th class="r">Trades</th><th class="r">Win %</th><th class="r">Avg</th><th class="r">Total</th></tr></thead>
      <tbody>${r.history.map(h => `<tr><td>${icon(h.level)}</td><td>${esc(h.label)}${h.count && h.count < MIN_SAMPLE ? ' <small class="muted">(too few to judge)</small>' : ""}</td>
        <td class="r num">${h.count}</td><td class="r num">${pct(h.win_rate)}</td><td class="r">${h.count ? pnlSpan(h.avg_pnl) : "–"}</td><td class="r">${h.count ? pnlSpan(h.total_pnl) : "–"}</td></tr>`).join("")}</tbody></table></div>
    </div>
    <p class="muted">Advice only, based on your own rules and history. It is not a prediction and not financial advice.</p>`;
}

// ---------- rules ----------

function renderRules() {
  const form = $("#rules-form");
  const f = form.elements;
  delete form.dataset.dirty;
  for (const k of ["account_size", "max_risk_pct", "daily_loss_limit", "max_trades_per_day", "min_rr", "revenge_minutes"]) f[k].value = S()[k];
  for (const k of ["setups", "emotions", "mistake_tags"]) f[k].value = S()[k].join("\n");
  $("#windows").innerHTML = "";
  S().no_trade_windows.forEach(addWindowRow);
  updateRiskHint();
}
function addWindowRow(w = { start: "", end: "", label: "" }) {
  const row = document.createElement("div");
  row.className = "windows-row";
  row.innerHTML = `<input type="time" class="w-start" value="${esc(w.start)}" aria-label="From"><input type="time" class="w-end" value="${esc(w.end)}" aria-label="To">
    <input class="w-label" value="${esc(w.label || "")}" placeholder="Why (e.g. lunch chop)" aria-label="Why"><button type="button" class="small danger">Remove</button>`;
  $("button", row).addEventListener("click", () => { row.remove(); $("#rules-form").dataset.dirty = "1"; });
  $("#windows").appendChild(row);
}
function updateRiskHint() {
  const f = $("#rules-form").elements;
  const d = parseFloat(f.account_size.value) * parseFloat(f.max_risk_pct.value) / 100;
  $("#risk-dollars").textContent = isNaN(d) ? "" : `= ${fmtMoney(d, false)} per trade`;
}
async function saveRules(ev) {
  ev.preventDefault();
  const f = ev.target.elements;
  const data = {};
  for (const k of ["account_size", "max_risk_pct", "daily_loss_limit", "max_trades_per_day", "min_rr", "revenge_minutes"]) {
    const v = Number(f[k].value);
    if (!isFinite(v) || v < 0) { toast("Rules must be numbers of 0 or more."); return; }
    data[k] = v;
  }
  for (const k of ["setups", "emotions", "mistake_tags"]) data[k] = [...new Set(f[k].value.split("\n").map(s => s.trim()).filter(Boolean))];
  data.no_trade_windows = $$("#windows .windows-row").map(r => ({ start: $(".w-start", r).value, end: $(".w-end", r).value, label: $(".w-label", r).value.trim() }))
    .filter(w => w.start && w.end);
  if (data.no_trade_windows.some(w => w.end <= w.start)) { toast("Each no-trade window must end after it starts."); return; }
  try {
    await store.saveSettings(data);
    store.settings = { ...structuredClone(DEFAULT_SETTINGS), ...data };
    renderRules();
    toast("Rules saved");
  } catch (e) { toast(e.message, 5000); }
}

function deleteAll() {
  const scope = $("#del-scope").value;
  const what = { paper: "all paper trades", real: "all real-money trades", all: "ALL trades" }[scope];
  const targets = store.trades.filter(t => scope === "all" || t.account_type === scope);
  if (!targets.length) { toast("There are no trades to delete."); return; }
  askConfirm($("#del-confirm"), `Permanently delete ${what} (${targets.length})? This can't be undone.`, `Delete ${targets.length} trades`, async () => {
    for (const t of targets) await store.deleteTrade(t.id);
    toast(`Deleted ${targets.length} trades`);
  });
}

// ---------- start ----------

function init() {
  setupFilters();
  document.addEventListener("click", e => {
    const nav = e.target.closest("[data-page]");
    if (nav) go(nav.dataset.page);
    const sample = e.target.closest("[data-sample]");
    if (sample) loadSample(sample.dataset.sample, sample.dataset.mode);
    if (e.target.closest(".export-btn")) exportCSV();
  });
  $("#trade-form").addEventListener("submit", saveTrade);
  $("#trade-form").addEventListener("input", liveCalc);
  $("#cancel-edit").addEventListener("click", () => go("trades"));
  $("#delete-trade").addEventListener("click", () => {
    if (!editing) return;
    askConfirm($("#delete-confirm"), `Delete this ${editing.ticker} trade? This can't be undone.`, "Delete trade", async () => {
      await store.deleteTrade(editing.id);
      toast("Trade deleted");
      go("trades");
    });
  });
  $("#csv-input").addEventListener("change", async e => {
    const file = e.target.files[0];
    if (!file) return;
    csvData = { ...parseCSV(await file.text()), name: file.name };
    if (!csvData.rows.length) { toast("That file has no rows."); csvData = null; return; }
    renderMapping();
  });
  $$("[name=imp-mode]").forEach(r => r.addEventListener("change", renderMapping));
  $("#imp-preview-btn").addEventListener("click", previewImport);
  $("#pt-form").addEventListener("submit", runPretrade);
  $("#rules-form").addEventListener("submit", saveRules);
  $("#rules-form").addEventListener("input", () => { $("#rules-form").dataset.dirty = "1"; updateRiskHint(); });
  $("#add-window").addEventListener("click", () => { addWindowRow(); $("#rules-form").dataset.dirty = "1"; });
  $("#del-all").addEventListener("click", deleteAll);
  go("dashboard");
  store.init().catch(e => { console.error(e); store.lock("Something went wrong while opening your journal. Reload the page."); });
}

if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
else init();
