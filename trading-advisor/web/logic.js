// ==== LOGIC START ====
// Pure trading math: no page, no storage. Shared by the website and the tests.
// Times are US market time (Eastern) wall-clock strings: "YYYY-MM-DDTHH:MM[:SS]".

const MIN_SAMPLE = 5;

const DEFAULT_SETTINGS = {
  account_size: 25000,
  max_risk_pct: 1,
  daily_loss_limit: 500,
  max_trades_per_day: 4,
  min_rr: 2,
  revenge_minutes: 15,
  no_trade_windows: [
    { start: "09:30", end: "09:45", label: "Opening chaos" },
    { start: "11:30", end: "13:30", label: "Lunch chop" },
  ],
  setups: ["Breakout", "Pullback", "Gap and Go", "VWAP Reclaim", "Reversal", "Momentum", "Other"],
  emotions: ["Calm", "Confident", "Excited", "FOMO", "Anxious", "Frustrated", "Bored", "Greedy"],
  mistake_tags: ["Chased entry", "Moved stop", "No plan", "Exited too early", "Oversized", "Ignored rules"],
  // What paper trading must show before moving to real money (see readiness()).
  readiness: { min_trades: 100, min_days: 20, min_profit_factor: 1.5, min_green_days_pct: 60, min_day_ratio: 1.5,
    max_drawdown_pct: 10, max_rule_break_pct: 10 },
};

const TRADE_FIELDS = ["ticker", "direction", "entry_time", "entry_price", "exit_time", "exit_price", "shares",
  "stop_loss", "target", "fees", "account_type", "setup", "reason", "emotion_before", "emotion_after",
  "followed_plan", "mistakes", "notes", "screenshot", "trader"];
const NUMERIC_FIELDS = new Set(["entry_price", "exit_price", "shares", "stop_loss", "target", "fees"]);

const FLAG_INFO = {
  over_risk: ["Risked more than your max", "The distance to your stop times your shares was more than your max risk per trade."],
  no_stop: ["No stop-loss", "No stop-loss was recorded, so the trade's risk was unlimited."],
  bad_stop: ["Stop on the wrong side", "For a long the stop must be below entry; for a short it must be above."],
  low_rr: ["Reward/risk below your minimum", "The planned target didn't pay enough compared with the risk to the stop."],
  revenge: ["Revenge trade", "Opened soon after a losing trade closed."],
  overtrading: ["Overtrading", "This trade was over your max trades per day."],
  after_loss_limit: ["Traded after hitting daily loss limit", "You were already at or past your daily loss limit when you opened this trade."],
  no_trade_window: ["Traded in a no-trade window", "Opened during one of the time windows you said to avoid."],
  held_loser: ["Held a loser too long", "This losing trade was held much longer than your typical winner."],
};

const DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
const TIME_BUCKETS = [
  [0, 570, "Pre-market", "pre-market trades (before 9:30am)"],
  [570, 585, "9:30–9:45", "trades between 9:30 and 9:45am"],
  [585, 600, "9:45–10:00", "trades between 9:45 and 10:00am"],
  [600, 630, "10:00–10:30", "trades between 10:00 and 10:30am"],
  [630, 660, "10:30–11:00", "trades between 10:30 and 11:00am"],
  [660, 720, "11:00–12:00", "trades between 11am and noon"],
  [720, 780, "12:00–1:00", "trades between noon and 1pm"],
  [780, 840, "1:00–2:00", "trades between 1 and 2pm"],
  [840, 900, "2:00–3:00", "trades between 2 and 3pm"],
  [900, 960, "3:00–4:00", "trades in the last hour (3 to 4pm)"],
  [960, 1440, "After hours", "after-hours trades (4pm or later)"],
];
const HOLD_BUCKETS = [[5, "Under 5 min"], [15, "5–15 min"], [60, "15–60 min"], [240, "1–4 hours"], [Infinity, "Over 4 hours"]];

// ---------- small helpers ----------

function parseDT(s) {
  if (!s) return null;
  const m = /^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})(?::(\d{2}))?/.exec(String(s).trim());
  if (!m) throw new Error(`could not read the date/time "${s}"`);
  const ms = Date.UTC(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +(m[6] || 0));
  return { ms, date: `${m[1]}-${m[2]}-${m[3]}`, hm: `${m[4]}:${m[5]}`, mins: +m[4] * 60 + +m[5], dow: (new Date(ms).getUTCDay() + 6) % 7 };
}
const fmtMs = ms => new Date(ms).toISOString().slice(0, 19);
const sum = a => a.reduce((x, y) => x + y, 0);
function median(a) {
  const s = [...a].sort((x, y) => x - y), n = s.length;
  return n % 2 ? s[(n - 1) / 2] : (s[n / 2 - 1] + s[n / 2]) / 2;
}
const money = x => `${x < 0 ? "-" : "+"}$${Math.abs(x).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
const plainMoney = x => `$${Math.abs(x).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
const pctText = x => `${Math.round(x * 100)}%`;
function minutesLabel(m) {
  if (m < 90) return `${Math.round(m)} min`;
  if (m < 1440) return `${(m / 60).toFixed(1)} hours`;
  return `${(m / 1440).toFixed(1)} days`;
}
function timeBucket(dt) {
  for (const [s, e, label, phrase] of TIME_BUCKETS) if (dt.mins >= s && dt.mins < e) return [label, phrase];
  return TIME_BUCKETS[TIME_BUCKETS.length - 1].slice(2);
}
function holdBucket(t) {
  if (t.entry_dt.date !== t.exit_dt.date) return "Overnight or longer";
  for (const [limit, label] of HOLD_BUCKETS) if (t.hold_minutes < limit) return label;
  return HOLD_BUCKETS[HOLD_BUCKETS.length - 1][1];
}
function inWindow(dt, windows) {
  for (const w of windows || []) if (w.start && w.end && w.start <= dt.hm && dt.hm < w.end) return w;
  return null;
}
function journalMistakes(t) {
  return String(t.mistakes || "").split(",").map(s => s.trim()).filter(Boolean);
}

// ---------- market conditions (from the "context" saved for each trade) ----------

const CONTEXT_GROUPS = {
  gap: ["Stock's gap at the open", c => {
    const g = c.daily?.gap_pct;
    if (g === null || g === undefined) return null;
    if (g >= 2) return ["Gapped up 2%+", "trades on days the stock gapped up 2% or more"];
    if (g <= -2) return ["Gapped down 2%+", "trades on days the stock gapped down 2% or more"];
    return ["Small gap (under 2%)", "trades on days the stock had only a small gap"];
  }],
  rel_volume: ["Stock's volume vs normal", c => {
    const v = c.daily?.rel_volume;
    if (v === null || v === undefined) return null;
    if (v >= 2) return ["Heavy (2× normal or more)", "trades on heavy-volume days (2× normal or more)"];
    if (v >= 1) return ["Normal to 2×", "trades on normal-volume days"];
    return ["Below normal", "trades on quieter-than-normal days"];
  }],
  market: ["Overall market (SPY) that day", c => {
    const m = c.market?.spy_change_pct;
    if (m === null || m === undefined) return null;
    if (m >= 0.3) return ["Market up", "trades on days the overall market rose"];
    if (m <= -0.3) return ["Market down", "trades on days the overall market fell"];
    return ["Market flat", "trades on flat market days"];
  }],
  vwap: ["Entry vs VWAP", c => c.entry_vs_vwap === "above" ? ["Above VWAP", "trades entered above VWAP"]
    : c.entry_vs_vwap === "below" ? ["Below VWAP", "trades entered below VWAP"] : null],
  news: ["News that day", c => !Array.isArray(c.news) ? null
    : c.news.length ? ["Had news", "trades on days the stock had news"] : ["No news", "trades on days the stock had no news"]],
};
const CONTEXT_ORDER = {
  gap: ["Gapped up 2%+", "Small gap (under 2%)", "Gapped down 2%+"],
  rel_volume: ["Heavy (2× normal or more)", "Normal to 2×", "Below normal"],
  market: ["Market up", "Market flat", "Market down"],
  vwap: ["Above VWAP", "Below VWAP"],
  news: ["Had news", "No news"],
};
function contextBucket(key, ctx) {
  return ctx ? CONTEXT_GROUPS[key][1](ctx) : null;
}

// ---------- validate one trade ----------

function cleanTrade(data) {
  const t = {};
  for (const f of TRADE_FIELDS) {
    let v = data[f];
    if (typeof v === "string") v = v.trim();
    if (NUMERIC_FIELDS.has(f)) {
      if (v === undefined || v === null || v === "") v = null;
      else {
        v = Number(v);
        if (!isFinite(v)) throw new Error(`${f.replace("_", " ")} must be a number`);
      }
    } else if (v === "" || v === undefined) v = null;
    t[f] = v;
  }
  if (!t.ticker) throw new Error("ticker is required");
  // "NASDAQ:AAPL" (TradingView style) -> "AAPL"
  t.ticker = String(t.ticker).toUpperCase().replace(/^[A-Z]+:/, "");
  t.direction = String(t.direction || "").toLowerCase();
  if (!["long", "short"].includes(t.direction)) throw new Error("direction must be long or short");
  if (!t.entry_time) throw new Error("entry date/time is required");
  parseDT(t.entry_time);
  if (t.entry_price === null || t.entry_price <= 0) throw new Error("entry price must be above 0");
  if (t.shares === null || t.shares <= 0) throw new Error("shares must be above 0");
  if ((t.exit_price === null) !== (t.exit_time === null)) throw new Error("enter both exit date/time and exit price, or neither (open trade)");
  if (t.exit_time) {
    parseDT(t.exit_time);
    if (t.exit_time.replace(" ", "T") < t.entry_time.replace(" ", "T")) throw new Error("exit time is before entry time");
  }
  if (t.stop_loss !== null && t.stop_loss <= 0) t.stop_loss = null;
  if (t.target !== null && t.target <= 0) t.target = null;
  t.fees = t.fees || 0;
  t.account_type = String(t.account_type || "paper").toLowerCase();
  if (!["paper", "real"].includes(t.account_type)) throw new Error("account type must be paper or real");
  const fp = t.followed_plan;
  t.followed_plan = [true, 1, "1", "yes", "true"].includes(fp) ? 1 : [false, 0, "0", "no", "false"].includes(fp) ? 0 : null;
  if (Array.isArray(t.mistakes)) t.mistakes = t.mistakes.filter(Boolean).join(", ");
  // trader: empty for your own trades, otherwise the other person's name
  t.trader = t.trader ? String(t.trader).replace(/\s+/g, " ").slice(0, 60) : null;
  if (t.trader && /^(me|mine|myself|you)$/i.test(t.trader)) t.trader = null;
  return t;
}

// ---------- per-trade numbers ----------

function riskNumbers(direction, entry, stop, target, shares, settings) {
  const out = { risk_per_share: null, risk_dollars: null, risk_pct: null, reward_dollars: null, rr: null, stop_ok: null, target_ok: null };
  const sign = direction === "long" ? 1 : -1;
  if (stop) {
    const per = (entry - stop) * sign;
    out.stop_ok = per > 0;
    if (per > 0) {
      out.risk_per_share = per;
      out.risk_dollars = per * shares;
      const acct = Number(settings.account_size) || 0;
      out.risk_pct = acct ? out.risk_dollars / acct * 100 : null;
    }
  }
  if (target) {
    const reward = (target - entry) * sign;
    out.target_ok = reward > 0;
    if (reward > 0) {
      out.reward_dollars = reward * shares;
      if (out.risk_per_share) out.rr = reward / out.risk_per_share;
    }
  }
  return out;
}

function enrich(trade, settings) {
  const t = { ...trade };
  t.entry_dt = parseDT(t.entry_time);
  t.exit_dt = parseDT(t.exit_time);
  t.closed = !!t.exit_dt && t.exit_price !== null && t.exit_price !== undefined;
  Object.assign(t, riskNumbers(t.direction, t.entry_price, t.stop_loss, t.target, t.shares, settings));
  const sign = t.direction === "long" ? 1 : -1;
  if (t.closed) {
    t.gross_pnl = (t.exit_price - t.entry_price) * sign * t.shares;
    t.pnl = t.gross_pnl - (t.fees || 0);
    t.hold_minutes = (t.exit_dt.ms - t.entry_dt.ms) / 60000;
    t.r_multiple = t.risk_dollars ? t.pnl / t.risk_dollars : null;
    t.outcome = t.pnl > 0 ? "win" : t.pnl < 0 ? "loss" : "breakeven";
  } else {
    t.gross_pnl = t.pnl = t.hold_minutes = t.r_multiple = null;
    t.outcome = "open";
  }
  t.flags = [];
  return t;
}

// ---------- mistake flags ----------

// Each person's paper and real trades are judged separately, so one trader's loss
// never makes another's trade a "revenge trade". Everyone is measured against your rules.
function applyFlags(trades, settings) {
  const groups = new Map();
  for (const t of trades) {
    const k = (t.trader || "") + "|" + t.account_type;
    if (!groups.has(k)) groups.set(k, []);
    groups.get(k).push(t);
  }
  for (const group of groups.values()) {
    group.sort((a, b) => a.entry_dt.ms - b.entry_dt.ms || String(a.id).localeCompare(String(b.id)));
    flagGroup(group, settings);
  }
  return trades;
}

function flagGroup(trades, settings) {
  const maxRisk = Number(settings.account_size) * Number(settings.max_risk_pct) / 100;
  const lossLimit = Number(settings.daily_loss_limit);
  const maxPerDay = Math.floor(Number(settings.max_trades_per_day));
  const minRR = Number(settings.min_rr);
  const revengeMs = Number(settings.revenge_minutes ?? 15) * 60000;
  const winnerHolds = trades.filter(t => t.outcome === "win").map(t => t.hold_minutes);
  const typicalWin = winnerHolds.length >= 3 ? median(winnerHolds) : null;
  const perDay = {};
  for (const t of trades) {
    const f = t.flags;
    if (!t.stop_loss) f.push("no_stop");
    else if (!t.stop_ok) f.push("bad_stop");
    if (t.risk_dollars && t.risk_dollars > maxRisk + 0.005) f.push("over_risk");
    if (t.rr !== null && minRR && t.rr < minRR - 1e-9) f.push("low_rr");
    const day = (perDay[t.entry_dt.date] ||= []);
    day.push(t);
    const earlier = day.filter(p => p !== t && p.closed && p.exit_dt.ms <= t.entry_dt.ms);
    if (earlier.some(p => p.pnl < 0 && t.entry_dt.ms - p.exit_dt.ms <= revengeMs)) f.push("revenge");
    if (maxPerDay > 0 && day.length > maxPerDay) f.push("overtrading");
    if (lossLimit > 0 && earlier.length && sum(earlier.map(p => p.pnl)) <= -lossLimit) f.push("after_loss_limit");
    if (inWindow(t.entry_dt, settings.no_trade_windows)) f.push("no_trade_window");
    if (typicalWin !== null && t.outcome === "loss" && t.hold_minutes > typicalWin * 1.5) f.push("held_loser");
  }
}

// ---------- stats ----------

function summary(closed, settings) {
  const wins = closed.filter(t => t.pnl > 0).map(t => t.pnl);
  const losses = closed.filter(t => t.pnl < 0).map(t => t.pnl);
  const n = closed.length, grossWin = sum(wins), grossLoss = -sum(losses), total = sum(closed.map(t => t.pnl));
  const rs = closed.filter(t => t.r_multiple !== null).map(t => t.r_multiple);
  return {
    trades: n, wins: wins.length, losses: losses.length, breakeven: n - wins.length - losses.length,
    total_pnl: total,
    win_rate: n ? wins.length / n : null,
    avg_win: wins.length ? grossWin / wins.length : null,
    avg_loss: losses.length ? grossLoss / losses.length : null,
    expectancy: n ? total / n : null,
    profit_factor: grossLoss ? grossWin / grossLoss : null,
    largest_win: wins.length ? Math.max(...wins) : null,
    largest_loss: losses.length ? Math.min(...losses) : null,
    avg_r: rs.length ? sum(rs) / rs.length : null,
    total_fees: sum(closed.map(t => t.fees || 0)),
    gross_win: grossWin, gross_loss: grossLoss,
    ...equityAndDrawdown(closed, Number(settings.account_size)),
    ...streaks(closed),
  };
}

const byExit = (a, b) => a.exit_dt.ms - b.exit_dt.ms || String(a.id).localeCompare(String(b.id));

function equityAndDrawdown(closed, start) {
  const ordered = [...closed].sort(byExit);
  let equity = start, peak = start, peakTime = ordered.length ? ordered[0].exit_time : null;
  const curve = [{ time: null, equity: start, pnl: 0, label: "Start" }];
  let dd = { max_drawdown: 0, max_drawdown_pct: 0, drawdown_from: null, drawdown_to: null };
  for (const t of ordered) {
    equity += t.pnl;
    curve.push({ time: t.exit_time, equity: Math.round(equity * 100) / 100, pnl: Math.round(t.pnl * 100) / 100, label: `${t.ticker} ${t.direction}`, id: t.id });
    if (equity > peak) { peak = equity; peakTime = t.exit_time; }
    const drop = peak - equity;
    if (drop > dd.max_drawdown) dd = { max_drawdown: drop, max_drawdown_pct: peak > 0 ? drop / peak * 100 : null, drawdown_from: peakTime, drawdown_to: t.exit_time };
  }
  return { equity_curve: curve, ...dd };
}

function streaks(closed) {
  const best = { win: 0, loss: 0 };
  let kind = null, len = 0;
  for (const t of [...closed].sort(byExit)) {
    const k = t.outcome === "win" || t.outcome === "loss" ? t.outcome : null;
    if (k && k === kind) len++;
    else { kind = k; len = k ? 1 : 0; }
    if (k) best[k] = Math.max(best[k], len);
  }
  return { longest_win_streak: best.win, longest_loss_streak: best.loss, current_streak: { kind, length: len } };
}

function groupStats(label, trades, phrase) {
  const n = trades.length;
  const wins = trades.filter(t => t.pnl > 0).length, losses = trades.filter(t => t.pnl < 0).length;
  const total = sum(trades.map(t => t.pnl));
  return { label, phrase: phrase || label, count: n, wins, losses, win_rate: n ? wins / n : null, loss_rate: n ? losses / n : null,
    total_pnl: total, avg_pnl: n ? total / n : null };
}

function group(closed, keyfunc, { order, byCount } = {}) {
  const groups = new Map(), phrases = {};
  for (const t of closed) {
    const [k, phrase] = keyfunc(t);
    if (!groups.has(k)) groups.set(k, []);
    groups.get(k).push(t);
    phrases[k] = phrase;
  }
  let keys = [...groups.keys()];
  if (order) keys = [...order.filter(k => groups.has(k)), ...keys.filter(k => !order.includes(k))];
  else if (byCount) keys.sort((a, b) => groups.get(b).length - groups.get(a).length || (a < b ? -1 : a > b ? 1 : 0));
  return keys.map(k => groupStats(k, groups.get(k), phrases[k]));
}

function contextBreakdowns(closed) {
  const out = {};
  for (const key of Object.keys(CONTEXT_GROUPS)) {
    const withCtx = closed.filter(t => contextBucket(key, t.ctx));
    if (!withCtx.length) continue;
    out["ctx_" + key] = { title: CONTEXT_GROUPS[key][0], context: true, rows: group(withCtx, t => contextBucket(key, t.ctx), { order: CONTEXT_ORDER[key] }) };
  }
  return out;
}

function breakdowns(closed) {
  const planLabel = { 1: "Followed plan", 0: "Broke plan", null: "Not recorded" };
  const planPhrase = { 1: "trades where you followed your plan", 0: "trades where you broke your plan", null: "trades with no plan answer" };
  return {
    time_of_day: { title: "Time of day (entry)", rows: group(closed, t => timeBucket(t.entry_dt), { order: TIME_BUCKETS.map(b => b[2]) }) },
    day_of_week: { title: "Day of week", rows: group(closed, t => [DAYS[t.entry_dt.dow], `trades on ${DAYS[t.entry_dt.dow]}s`], { order: DAYS }) },
    ticker: { title: "Ticker", rows: group(closed, t => [t.ticker, `trades in ${t.ticker}`], { byCount: true }) },
    setup: { title: "Setup type", rows: group(closed, t => t.setup ? [t.setup, `"${t.setup}" trades`] : ["(no setup)", "trades with no setup recorded"], { byCount: true }) },
    hold_time: { title: "Hold time", rows: group(closed, t => [holdBucket(t), `trades held ${holdBucket(t).toLowerCase()}`], { order: [...HOLD_BUCKETS.map(b => b[1]), "Overnight or longer"] }) },
    plan: { title: "Followed my plan?", rows: group(closed, t => [planLabel[t.followed_plan ?? null], planPhrase[t.followed_plan ?? null]], { order: Object.values(planLabel) }) },
    emotion: { title: "Emotion before entry", rows: group(closed, t => t.emotion_before ? [t.emotion_before, `trades when you felt "${t.emotion_before}"`] : ["(not recorded)", "trades with no emotion recorded"], { byCount: true }) },
    ...contextBreakdowns(closed),
  };
}

// ---------- mistake costs ----------

function mistakeCosts(closed) {
  const buckets = new Map();
  for (const t of closed) {
    const keys = [...t.flags.map(f => ["flag", f]), ...journalMistakes(t).map(m => ["tag", m])];
    for (const [kind, name] of keys) {
      const k = kind + ":" + name;
      if (!buckets.has(k)) buckets.set(k, { key: name, source: kind === "flag" ? "automatic" : "journal",
        label: kind === "flag" ? FLAG_INFO[name][0] : name, explain: kind === "flag" ? FLAG_INFO[name][1] : "You tagged this in your journal.",
        count: 0, total_pnl: 0, money_lost: 0, trade_ids: [] });
      const b = buckets.get(k);
      b.count++; b.total_pnl += t.pnl; b.money_lost += Math.min(t.pnl, 0); b.trade_ids.push(t.id);
    }
  }
  const rows = [...buckets.values()].sort((a, b) => a.total_pnl - b.total_pnl);
  rows.forEach(r => (r.avg_pnl = r.total_pnl / r.count));
  const clean = closed.filter(t => !t.flags.length && !journalMistakes(t).length);
  return { rows, clean: clean.length ? groupStats("Trades with no mistakes", clean) : null };
}

// ---------- plain-English insights ----------

function insights(closed, s, bd, costs) {
  const out = [];
  const add = (tone, text, impact = 0, pinned = false) => out.push({ tone, text, impact: Math.abs(impact), pinned });
  const n = closed.length;
  if (!n) return [{ tone: "info", text: "No closed trades yet. Add a trade or import a CSV to see insights.", impact: 0 }];
  if (n < 30) add("info", `You have ${n} closed trades. Patterns get more trustworthy after about 30 or more trades, so treat these as early hints.`, 0, true);

  if (s.expectancy !== null) {
    if (s.expectancy < 0) add("bad", `On average each trade loses you ${plainMoney(s.expectancy)} after fees. Until that's positive, trade small or paper trade.`, s.total_pnl);
    else add("good", `On average each trade makes you ${plainMoney(s.expectancy)} after fees. Your edge is positive so far.`, s.total_pnl);
  }
  if (s.avg_win && s.avg_loss && s.avg_loss > s.avg_win * 1.1) {
    const need = s.avg_loss / (s.avg_win + s.avg_loss);
    add("bad", `Your average loss (${plainMoney(s.avg_loss)}) is bigger than your average win (${plainMoney(s.avg_win)}). At that size you need to win more than ${pctText(need)} of trades just to break even; you win ${pctText(s.win_rate)}.`,
      (s.avg_loss - s.avg_win) * s.losses);
  }
  for (const key of ["time_of_day", "day_of_week", "ticker", "setup", "hold_time", "emotion", ...Object.keys(bd).filter(k => k.startsWith("ctx_"))]) {
    for (const g of bd[key].rows) {
      if (g.count < MIN_SAMPLE || g.label.startsWith("(")) continue;
      const detail = `(${g.total_pnl < 0 ? g.losses : g.wins} of ${g.count} trades, ${money(g.total_pnl)} total)`;
      const Phrase = g.phrase[0].toUpperCase() + g.phrase.slice(1);
      if (g.loss_rate >= 0.6 && g.total_pnl < 0) add("bad", `Your ${g.phrase} lose ${pctText(g.loss_rate)} of the time ${detail}.`, g.total_pnl);
      else if (g.win_rate >= 0.6 && g.total_pnl > 0) add("good", `Your ${g.phrase} win ${pctText(g.win_rate)} of the time ${detail}. Look for more of these.`, g.total_pnl);
      else if (g.total_pnl < 0 && s.gross_loss && -g.total_pnl >= 0.15 * s.gross_loss)
        add("bad", `${Phrase} have cost you ${plainMoney(g.total_pnl)} overall (average ${money(g.avg_pnl)} per trade, ${g.count} trades).`, g.total_pnl);
    }
  }
  const plan = Object.fromEntries(bd.plan.rows.map(g => [g.label, g]));
  const yes = plan["Followed plan"], no = plan["Broke plan"];
  if (yes && no && yes.count >= 3 && no.count >= 3) {
    add(no.avg_pnl < yes.avg_pnl ? "bad" : "info",
      `When you follow your plan you average ${money(yes.avg_pnl)} per trade (${yes.count} trades). When you break it you average ${money(no.avg_pnl)} (${no.count} trades).`,
      no.total_pnl - no.count * yes.avg_pnl);
  }
  const winH = closed.filter(t => t.outcome === "win").map(t => t.hold_minutes);
  const lossH = closed.filter(t => t.outcome === "loss").map(t => t.hold_minutes);
  if (winH.length >= 3 && lossH.length >= 3) {
    const mw = median(winH), ml = median(lossH);
    if (mw > 0 && ml > mw * 1.3)
      add("bad", `You hold losing trades ${(ml / mw).toFixed(1)}× longer than winners (typical loser ${minutesLabel(ml)}, typical winner ${minutesLabel(mw)}). Losers rarely come back. Respect your stop.`,
        sum(closed.filter(t => t.outcome === "loss").map(t => t.pnl)));
    else if (ml > 0 && mw > ml * 1.3)
      add("good", `You let winners run longer than losers (typical winner ${minutesLabel(mw)}, typical loser ${minutesLabel(ml)}). Keep doing that.`, 0);
  }
  const bad = costs.rows.filter(r => r.total_pnl < 0);
  if (bad.length) add("bad", `Your most expensive mistake is "${bad[0].label}": ${bad[0].count} trades that lost ${plainMoney(bad[0].total_pnl)} in total.`, bad[0].total_pnl);
  if (costs.clean && costs.clean.count >= 3) {
    const c = costs.clean;
    add(c.avg_pnl > 0 ? "good" : "info", `Trades with no mistakes flagged average ${money(c.avg_pnl)} each (${c.count} trades). That's your real edge when you trade clean.`, c.total_pnl);
  }
  if (s.longest_loss_streak >= 4)
    add("info", `Your longest losing streak was ${s.longest_loss_streak} trades in a row. Consider a rule to stop for the day after 2 or 3 losses in a row.`);
  if (s.total_fees > 0 && s.gross_win > 0 && s.total_fees / s.gross_win >= 0.15)
    add("bad", `Fees ate ${pctText(s.total_fees / s.gross_win)} of your winning trades' profits (${plainMoney(s.total_fees)}). Fewer, better trades would keep more.`, s.total_fees);

  // Biggest problems first, but always leave room for what's working.
  const pick = (tone, k) => out.filter(i => i.tone === tone && !i.pinned).sort((a, b) => b.impact - a.impact).slice(0, k);
  return [...out.filter(i => i.pinned), ...pick("bad", 7), ...pick("good", 4), ...pick("info", 2)];
}

// ---------- days ----------

function dailyResults(closed) {
  const days = new Map();
  for (const t of closed) {
    const d = t.exit_dt.date;
    if (!days.has(d)) days.set(d, { date: d, pnl: 0, trades: 0 });
    const x = days.get(d);
    x.pnl += t.pnl; x.trades++;
  }
  return [...days.values()].sort((a, b) => (a.date < b.date ? -1 : 1));
}

function dayStats(days) {
  const green = days.filter(d => d.pnl > 0), red = days.filter(d => d.pnl < 0);
  const avgGreen = green.length ? sum(green.map(d => d.pnl)) / green.length : null;
  const avgRed = red.length ? -sum(red.map(d => d.pnl)) / red.length : null;
  return { days: days.length, green: green.length, red: red.length,
    green_pct: days.length ? green.length / days.length : null,
    avg_day: days.length ? sum(days.map(d => d.pnl)) / days.length : null,
    avg_green: avgGreen, avg_red: avgRed,
    day_ratio: avgGreen !== null && avgRed ? avgGreen / avgRed : avgGreen !== null ? Infinity : null,
    best: days.length ? Math.max(...days.map(d => d.pnl)) : null, worst: days.length ? Math.min(...days.map(d => d.pnl)) : null };
}

// ---------- ready for real money? ----------
// Judged only on YOUR PAPER trades. Every check must pass.

function readiness(trades, settings) {
  const R = { ...DEFAULT_SETTINGS.readiness, ...(settings.readiness || {}) };
  const mine = prepare(trades, settings).filter(t => !t.trader && t.account_type === "paper" && t.closed);
  const s = summary(mine, settings);
  const days = dailyResults(mine);
  const d = dayStats(days);
  const acct = Number(settings.account_size) || 1;
  const ddPct = s.max_drawdown / acct * 100;
  const breaks = mine.length ? mine.filter(t => t.flags.length).length / mine.length * 100 : null;
  const limit = Number(settings.daily_loss_limit) || 0;
  const overLimit = limit ? days.filter(x => x.pnl <= -limit).length : 0;
  const recent = [...mine].sort(byExit).slice(-20);
  const recentExp = recent.length ? sum(recent.map(t => t.pnl)) / recent.length : null;
  const c = (label, value, target, pass, progress, why) => ({ label, value, target, pass: !!pass, progress: Math.max(0, Math.min(1, progress ?? (pass ? 1 : 0))), why });
  const criteria = [
    c("Paper trades", `${s.trades}`, `${R.min_trades} or more`, s.trades >= R.min_trades, s.trades / R.min_trades, "Enough trades for the numbers to mean something."),
    c("Trading days", `${d.days}`, `${R.min_days} or more`, d.days >= R.min_days, d.days / R.min_days, "Shows it works across different market days, not one lucky week."),
    c("Average day", d.avg_day === null ? "–" : money(d.avg_day), "above $0", d.avg_day > 0, null, "You make money per day on average, after fees."),
    c("Green days", d.green_pct === null ? "–" : pctText(d.green_pct), `${R.min_green_days_pct}% or more`, d.green_pct !== null && d.green_pct * 100 >= R.min_green_days_pct,
      d.green_pct === null ? 0 : d.green_pct * 100 / R.min_green_days_pct, "Most days end up, not down."),
    c("Average green day vs average red day", d.day_ratio === null ? "–" : d.day_ratio === Infinity ? "no red days" : `${d.day_ratio.toFixed(2)}×`, `${R.min_day_ratio}× or more`,
      d.day_ratio !== null && d.day_ratio >= R.min_day_ratio, d.day_ratio === null ? 0 : d.day_ratio / R.min_day_ratio, "Good days are clearly bigger than bad days."),
    c("Profit factor", s.profit_factor === null ? (s.wins ? "no losses" : "–") : s.profit_factor.toFixed(2), `${R.min_profit_factor} or more`,
      s.profit_factor === null ? s.wins > 0 && s.trades >= R.min_trades : s.profit_factor >= R.min_profit_factor,
      s.profit_factor === null ? (s.wins ? 1 : 0) : s.profit_factor / R.min_profit_factor, "Total won is well above total lost."),
    c("Worst drawdown", `${ddPct.toFixed(1)}% of account`, `${R.max_drawdown_pct}% or less`, s.trades > 0 && ddPct <= R.max_drawdown_pct, null, "Losing streaks stay small enough to survive."),
    c("Trades that broke a rule", breaks === null ? "–" : `${breaks.toFixed(0)}%`, `${R.max_rule_break_pct}% or less`, breaks !== null && breaks <= R.max_rule_break_pct, null, "You trade your plan; the results aren't luck."),
    c("Days past your daily loss limit", `${overLimit}`, "0", s.trades > 0 && overLimit === 0, null, "You always stop when you hit your limit."),
    c("Last 20 trades", recentExp === null ? "–" : `${money(recentExp)} per trade`, "above $0", recent.length >= 20 && recentExp > 0, null, "You're still profitable lately, not just early on."),
  ];
  return { ready: criteria.every(x => x.pass), passed: criteria.filter(x => x.pass).length, criteria, days, day_stats: d, summary: s };
}

// ---------- top level ----------

function prepare(trades, settings) {
  const enriched = trades.map(t => enrich(t, settings));
  applyFlags(enriched, settings);
  return enriched;
}

// f.who: "me" (default), "others", "all", or "t:<name>" for one other trader.
function whoMatches(t, who = "me") {
  if (who === "all") return true;
  if (who === "others") return !!t.trader;
  if (who.startsWith("t:")) return t.trader === who.slice(2);
  return !t.trader;
}
function whoLabel(who = "me") {
  if (who === "all") return { name: "Everyone", possessive: "Everyone's" };
  if (who === "others") return { name: "Other traders", possessive: "Other traders'" };
  if (who.startsWith("t:")) { const n = who.slice(2); return { name: n, possessive: n.endsWith("s") ? n + "'" : n + "'s" }; }
  return { name: "You", possessive: "Your" };
}
function traderNames(trades) {
  return [...new Set(trades.map(t => t.trader).filter(Boolean))].sort((a, b) => a.localeCompare(b));
}

function filterTrades(trades, f = {}) {
  return trades.filter(t => {
    if (!whoMatches(t, f.who)) return false;
    if ((f.account === "paper" || f.account === "real") && t.account_type !== f.account) return false;
    const d = t.entry_time.slice(0, 10);
    if (f.from && d < f.from) return false;
    if (f.to && d > f.to) return false;
    return true;
  });
}

function fullReport(trades, settings, f = {}, contexts = {}) {
  const enriched = filterTrades(prepare(trades, settings), f);
  for (const t of enriched) t.ctx = contexts[t.id] || null;
  const closed = enriched.filter(t => t.closed);
  const s = summary(closed, settings);
  const bd = breakdowns(closed);
  const costs = mistakeCosts(closed);
  let ins = insights(closed, s, bd, costs);
  if (f.who && f.who !== "me") ins = ins.map(i => ({ ...i, text: aboutSomeoneElse(i.text, whoLabel(f.who)) }));
  const days = dailyResults(closed);
  return { summary: s, breakdowns: bd, mistakes: costs, insights: ins, days, day_stats: dayStats(days),
    with_context: closed.filter(t => t.ctx).length,
    open_trades: enriched.filter(t => !t.closed).length, trades: enriched };
}

// Insights are written to "you"; reword them when looking at someone else's trades.
function aboutSomeoneElse(text, label) {
  return text
    .replace(/Until that's positive, trade small or paper trade\./, "")
    .replace(/\b(loses|makes) you\b/g, "$1 them")
    .replace(/\bYour\b/g, label.possessive).replace(/\byour\b/g, "their")
    .replace(/\bYou\b/g, "They").replace(/\byou\b/g, "they").trim();
}

// One row of headline stats per person, for "You vs other traders".
function traderComparison(trades, settings, f = {}) {
  const enriched = filterTrades(prepare(trades, settings), { ...f, who: "all" }).filter(t => t.closed);
  const people = [null, ...traderNames(enriched)];
  return people.map(name => {
    const closed = enriched.filter(t => (t.trader || null) === name);
    if (!closed.length) return null;
    const s = summary(closed, settings);
    const holds = closed.map(t => t.hold_minutes);
    const best = breakdowns(closed).setup.rows.filter(g => g.count >= 3 && g.label !== "(no setup)").sort((a, b) => b.total_pnl - a.total_pnl)[0];
    return { name: name || "You", who: name ? "t:" + name : "me", trades: s.trades, win_rate: s.win_rate, expectancy: s.expectancy,
      avg_win: s.avg_win, avg_loss: s.avg_loss, profit_factor: s.profit_factor, total_pnl: s.total_pnl,
      typical_hold: median(holds), mistake_rate: closed.filter(t => t.flags.length).length / closed.length,
      best_setup: best ? best.label : null };
  }).filter(Boolean);
}

// ---------- pre-trade check ----------

// plan.conditions (optional): { gap, rel_volume, market, vwap } using the bucket labels in CONTEXT_ORDER.
function preTradeCheck(plan, trades, settings, nowStr, contexts = {}) {
  const direction = (plan.direction || "long").toLowerCase();
  const entry = Number(plan.entry), shares = Number(plan.shares);
  if (!(entry > 0)) throw new Error("Entry price must be a number above 0.");
  if (!(shares > 0)) throw new Error("Shares must be a number above 0.");
  const stop = plan.stop === "" || plan.stop == null ? null : Number(plan.stop);
  const target = plan.target === "" || plan.target == null ? null : Number(plan.target);
  const when = parseDT(plan.time || nowStr);
  const accountType = plan.account_type || "real";
  const setup = plan.setup || null;
  const ticker = String(plan.ticker || "").toUpperCase();

  const acct = Number(settings.account_size), maxRiskPct = Number(settings.max_risk_pct);
  const maxRisk = acct * maxRiskPct / 100, minRR = Number(settings.min_rr);
  const r = riskNumbers(direction, entry, stop, target, shares, settings);
  const positionValue = entry * shares;
  const checks = [];
  const check = (level, title, detail) => checks.push({ level, title, detail });

  if (stop === null) check("red", "No stop-loss", "Decide where you're wrong before you get in. Without a stop your risk has no limit.");
  else if (!r.stop_ok) check("red", "Stop is on the wrong side", `For a ${direction} the stop must be ${direction === "long" ? "below" : "above"} your entry.`);
  else if (r.risk_dollars > maxRisk + 0.005) check("red", "Risk is over your max", `You'd risk ${plainMoney(r.risk_dollars)} (${r.risk_pct.toFixed(2)}% of your account). Your max is ${plainMoney(maxRisk)} (${maxRiskPct}%).`);
  else if (r.risk_dollars > maxRisk * 0.8) check("yellow", "Risk is close to your max", `${plainMoney(r.risk_dollars)} (${r.risk_pct.toFixed(2)}%) of a ${plainMoney(maxRisk)} max.`);
  else check("green", "Risk is within your max", `${plainMoney(r.risk_dollars)} (${r.risk_pct.toFixed(2)}% of your account).`);

  if (target === null) check("yellow", "No target", "Without a target you can't judge if the trade is worth the risk.");
  else if (!r.target_ok) check("red", "Target is on the wrong side", `For a ${direction} the target must be ${direction === "long" ? "above" : "below"} your entry.`);
  else if (r.rr !== null) {
    if (r.rr < minRR - 1e-9) check("red", "Reward/risk below your minimum", `${r.rr.toFixed(2)} : 1, but your rule is at least ${minRR} : 1.`);
    else check("green", "Reward/risk meets your rule", `${r.rr.toFixed(2)} : 1 (minimum ${minRR} : 1).`);
  }
  if (positionValue > acct) check("yellow", "Position is bigger than your account", `${plainMoney(positionValue)} of stock on a ${plainMoney(acct)} account means you'd be using margin (borrowed money).`);

  const w = inWindow(when, settings.no_trade_windows);
  if (w) check("red", "Inside a no-trade window", `${w.start}–${w.end}${w.label ? ` (${w.label})` : ""}`);
  if (when.mins < 570 || when.mins >= 960) check("yellow", "Outside regular market hours", "Pre-market and after-hours have thin volume and wide spreads. Be extra careful.");

  const everyone = prepare(trades, settings);
  const enriched = everyone.filter(t => !t.trader && t.account_type === accountType);
  const today = enriched.filter(t => t.entry_dt.date === when.date && t.entry_dt.ms <= when.ms);
  const doneToday = today.filter(t => t.closed && t.exit_dt.ms <= when.ms);
  const dayPnl = sum(doneToday.map(t => t.pnl));
  const maxTrades = Math.floor(Number(settings.max_trades_per_day)), lossLimit = Number(settings.daily_loss_limit);
  if (maxTrades) {
    if (today.length >= maxTrades) check("red", "Over your max trades for today", `You've already taken ${today.length} ${accountType === "real" ? "real-money" : "paper"} trades today (max ${maxTrades}).`);
    else if (today.length === maxTrades - 1) check("yellow", "Last trade allowed today", `This would be trade ${today.length + 1} of ${maxTrades}.`);
  }
  if (lossLimit) {
    if (dayPnl <= -lossLimit) check("red", "Daily loss limit already hit", `You're at ${money(dayPnl)} today. Your limit is ${plainMoney(lossLimit)}. Stop for the day.`);
    else if (r.risk_dollars && dayPnl - r.risk_dollars < -lossLimit) check("yellow", "A loss here would break your daily limit", `You're at ${money(dayPnl)} today. Losing ${plainMoney(r.risk_dollars)} would put you past -${plainMoney(lossLimit)}.`);
  }
  const revengeMs = Number(settings.revenge_minutes ?? 15) * 60000;
  const recent = doneToday.filter(t => t.pnl < 0 && when.ms - t.exit_dt.ms <= revengeMs);
  if (recent.length) {
    const last = recent.reduce((a, b) => (b.exit_dt.ms > a.exit_dt.ms ? b : a));
    check("yellow", "Possible revenge trade", `Your ${last.ticker} trade lost ${plainMoney(last.pnl)} at ${last.exit_dt.hm}. Take a breath and make sure this is a real setup.`);
  }

  const closed = enriched.filter(t => t.closed);
  for (const t of closed) t.ctx = contexts[t.id] || null;
  const [bucket] = timeBucket(when);
  const history = [];
  const hist = (title, subset) => {
    const g = groupStats(title, subset);
    g.level = g.count < MIN_SAMPLE ? "gray" : g.win_rate < 0.4 || g.total_pnl < 0 ? "yellow" : "green";
    history.push(g);
    return g;
  };
  const judged = [];
  if (setup) {
    judged.push(hist(`Setup "${setup}"`, closed.filter(t => t.setup === setup)));
    hist(`"${setup}" at ${bucket}`, closed.filter(t => t.setup === setup && timeBucket(t.entry_dt)[0] === bucket));
  }
  judged.push(hist(`Any trade at ${bucket}`, closed.filter(t => timeBucket(t.entry_dt)[0] === bucket)));
  if (ticker) hist(`${ticker} trades`, closed.filter(t => t.ticker === ticker));
  hist(`${direction[0].toUpperCase() + direction.slice(1)} trades`, closed.filter(t => t.direction === direction));
  for (const [key, label] of Object.entries(plan.conditions || {})) {
    if (!label || !CONTEXT_GROUPS[key]) continue;
    judged.push(hist(`${CONTEXT_GROUPS[key][0]}: ${label}`, closed.filter(t => contextBucket(key, t.ctx)?.[0] === label)));
  }
  // Other traders: shown for learning, never counted as your own history
  const others = everyone.filter(t => t.trader && t.closed);
  const otherRows = [];
  if (others.length) {
    const oh = (title, subset) => { const g = groupStats(title, subset); g.level = g.count < MIN_SAMPLE ? "gray" : g.win_rate < 0.4 || g.total_pnl < 0 ? "yellow" : "green"; otherRows.push(g); };
    if (setup) oh(`Others: setup "${setup}"`, others.filter(t => t.setup === setup));
    oh(`Others: any trade at ${bucket}`, others.filter(t => timeBucket(t.entry_dt)[0] === bucket));
    if (ticker) oh(`Others: ${ticker} trades`, others.filter(t => t.ticker === ticker));
  }
  for (const g of judged) {
    if (g.count >= MIN_SAMPLE && g.level === "yellow") check("yellow", `Your history is weak here: ${g.label}`, `Won ${pctText(g.win_rate)} of ${g.count} trades, ${money(g.total_pnl)} total.`);
    else if (g.count >= MIN_SAMPLE && g.level === "green") check("green", `Your history is good here: ${g.label}`, `Won ${pctText(g.win_rate)} of ${g.count} trades, ${money(g.total_pnl)} total.`);
  }
  const order = { red: 0, yellow: 1, green: 2 };
  checks.sort((a, b) => order[a.level] - order[b.level]);
  return {
    numbers: { ...r, position_value: positionValue, position_pct: acct ? positionValue / acct * 100 : null, max_risk_dollars: maxRisk,
      suggested_max_shares: r.risk_per_share ? Math.floor(maxRisk / r.risk_per_share) : null, trades_today: today.length, pnl_today: dayPnl },
    checks, history, others_history: otherRows, time_bucket: bucket,
    verdict: checks.some(c => c.level === "red") ? "red" : checks.some(c => c.level === "yellow") ? "yellow" : "green",
  };
}

// ---------- CSV import ----------

function parseCSV(text) {
  text = String(text).replace(/^﻿/, "");
  const rows = [];
  let row = [], field = "", inQ = false;
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (inQ) {
      if (c === '"') { if (text[i + 1] === '"') { field += '"'; i++; } else inQ = false; }
      else field += c;
    } else if (c === '"') inQ = true;
    else if (c === ",") { row.push(field); field = ""; }
    else if (c === "\n" || c === "\r") {
      if (c === "\r" && text[i + 1] === "\n") i++;
      row.push(field); field = "";
      if (row.some(v => v.trim() !== "")) rows.push(row);
      row = [];
    } else field += c;
  }
  row.push(field);
  if (row.some(v => v.trim() !== "")) rows.push(row);
  if (!rows.length) return { headers: [], rows: [] };
  const headers = rows[0].map((h, i) => h.trim() || `Column ${i + 1}`);
  return { headers, rows: rows.slice(1).map(r => Object.fromEntries(headers.map((h, i) => [h, (r[i] ?? "").trim()]))) };
}

function parseNumber(v) {
  if (v === null || v === undefined) return null;
  let s = String(v).trim();
  if (!s || s === "-" || /^(n\/a|na|none|null)$/i.test(s)) return null;
  const neg = s.startsWith("(") && s.endsWith(")");
  s = s.replace(/[,$\s()]/g, "");
  const n = Number(s);
  if (!isFinite(n) || s === "") throw new Error(`"${v}" is not a number`);
  return neg ? -n : n;
}

const MONTHS = { jan: 1, feb: 2, mar: 3, apr: 4, may: 5, jun: 6, jul: 7, aug: 8, sep: 9, oct: 10, nov: 11, dec: 12 };
const TZ_WORDS = /\s*\b(ET|EST|EDT|CT|CST|CDT|PT|PST|PDT|MT|MST|MDT|UTC|GMT)\s*$/i;

// Read a broker date (and optional separate time) as "YYYY-MM-DDTHH:MM:SS".
// offsetHours shifts the time into US Eastern (e.g. +1 for Central).
function parseBrokerDate(dateStr, timeStr, offsetHours = 0) {
  let d = String(dateStr || "").trim().replace(TZ_WORDS, "");
  const tIn = String(timeStr || "").trim().replace(TZ_WORDS, "");
  if (!d) throw new Error("missing date");
  d = d.replace(/(\d)(Z|[+-]\d{2}:?\d{2})$/, "$1"); // drop ISO zone suffix
  const combined = tIn ? `${d} ${tIn}` : d;
  const tm = /(\d{1,2}):(\d{2})(?::(\d{2}))?(?:\.\d+)?\s*([AaPp]\.?[Mm]\.?)?/.exec(combined);
  let datePart = (tm ? combined.replace(tm[0], " ") : combined).replace(/[T,]\s*$/, "").replace(/\s+[T]?\s*$/, "").trim().replace(/,$/, "").trim();
  if (tIn && !tm) throw new Error(`could not read the time "${timeStr}"`);
  let y, mo, day, m;
  if ((m = /^(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})T?$/.exec(datePart))) [y, mo, day] = [+m[1], +m[2], +m[3]];
  else if ((m = /^(\d{1,2})[/-](\d{1,2})[/-](\d{2}|\d{4})$/.exec(datePart))) [mo, day, y] = [+m[1], +m[2], +m[3]];
  else if ((m = /^(\d{4})(\d{2})(\d{2})$/.exec(datePart))) [y, mo, day] = [+m[1], +m[2], +m[3]];
  else if ((m = /^(\d{1,2})[- ]([A-Za-z]{3})[A-Za-z]*\.?[- ,]+(\d{2,4})$/.exec(datePart)) && MONTHS[m[2].toLowerCase()]) [day, mo, y] = [+m[1], MONTHS[m[2].toLowerCase()], +m[3]];
  else if ((m = /^([A-Za-z]{3})[A-Za-z]*\.?\s+(\d{1,2}),?\s+(\d{4})$/.exec(datePart)) && MONTHS[m[1].toLowerCase()]) [mo, day, y] = [MONTHS[m[1].toLowerCase()], +m[2], +m[3]];
  else throw new Error(`could not read the date "${dateStr}"`);
  if (y < 100) y += 2000;
  if (mo < 1 || mo > 12 || day < 1 || day > 31) throw new Error(`could not read the date "${dateStr}"`);
  let hh = 9, mi = 30, ss = 0, warning = null;
  if (tm) {
    hh = +tm[1]; mi = +tm[2]; ss = +(tm[3] || 0);
    const ap = (tm[4] || "").replace(/\./g, "").toUpperCase();
    if (ap === "PM" && hh < 12) hh += 12;
    if (ap === "AM" && hh === 12) hh = 0;
    if (hh > 23 || mi > 59 || ss > 59) throw new Error(`could not read the time in "${combined}"`);
  } else warning = "no time found, used 9:30am";
  const ms = Date.UTC(y, mo - 1, day, hh, mi, ss) + Number(offsetHours || 0) * 3600000;
  return [fmtMs(ms), warning];
}

const LONG_WORDS = new Set(["long", "l", "buy", "b", "bot", "bought", "buy to open", "bto", "buy long"]);
const SHORT_WORDS = new Set(["short", "sell short", "ss", "short sell", "sell to open", "sto", "sshrt", "sht", "s"]);
const BUY_WORDS = new Set(["buy", "b", "bot", "bought", "buy to open", "bto", "buy to cover", "btc", "cover", "long", "buy long"]);
const SELL_WORDS = new Set(["sell", "s", "sld", "sold", "sell short", "ss", "short", "short sell", "sell to open", "sto", "sell to close", "stc", "sshrt", "sht"]);

function parseDirection(v, dflt = "long") {
  const s = String(v || "").trim().toLowerCase();
  if (!s) return dflt;
  if (LONG_WORDS.has(s)) return "long";
  if (SHORT_WORDS.has(s) || s.startsWith("short") || s.startsWith("sell short")) return "short";
  if (s.startsWith("long") || s.startsWith("buy")) return "long";
  throw new Error(`could not tell if "${v}" is long or short`);
}
const rowFees = row => ["fees", "fees2"].reduce((a, k) => a + Math.abs(parseNumber(row[k]) || 0), 0);

function rowsToTrades(rows, opts) {
  const trades = [], errors = [], warnings = [];
  const off = Number(opts.tz_offset_hours || 0);
  rows.forEach((row, i) => {
    try {
      const [entry, w1] = parseBrokerDate(row.entry_date, row.entry_time, off);
      let exit = null, w2 = null;
      if (String(row.exit_date || "").trim()) [exit, w2] = parseBrokerDate(row.exit_date, row.exit_time, off);
      for (const w of [w1, w2]) if (w) warnings.push({ row: i + 1, message: w });
      const shares = parseNumber(row.shares);
      trades.push(cleanTrade({
        ticker: row.ticker, direction: parseDirection(row.direction, opts.default_direction || "long"),
        entry_time: entry, entry_price: parseNumber(row.entry_price), exit_time: exit,
        exit_price: exit ? parseNumber(row.exit_price) : null, shares: shares === null ? null : Math.abs(shares),
        stop_loss: parseNumber(row.stop_loss), target: parseNumber(row.target), fees: rowFees(row),
        account_type: opts.account_type || "paper", setup: row.setup, notes: row.notes, trader: opts.trader || null,
      }));
    } catch (e) { errors.push({ row: i + 1, message: e.message }); }
  });
  return [trades, errors, warnings];
}

function fillsToTrades(rows, opts) {
  const errors = [], warnings = [], fills = [];
  const off = Number(opts.tz_offset_hours || 0);
  rows.forEach((row, i) => {
    try {
      const ticker = String(row.ticker || "").trim().toUpperCase();
      if (!ticker) throw new Error("ticker is missing");
      const [when, w] = parseBrokerDate(row.date, row.time, off);
      if (w) warnings.push({ row: i + 1, message: w });
      const qty = parseNumber(row.quantity), price = parseNumber(row.price);
      if (!qty || !price || price <= 0) throw new Error("quantity and price must be numbers above 0");
      const side = String(row.side || "").trim().toLowerCase();
      let signed;
      if (BUY_WORDS.has(side) || side.startsWith("buy") || side.startsWith("bought")) signed = Math.abs(qty);
      else if (SELL_WORDS.has(side) || side.startsWith("sell") || side.startsWith("sold") || side.startsWith("short")) signed = -Math.abs(qty);
      else if (!side) signed = qty; // some brokers use negative quantity for sells
      else throw new Error(`could not tell if "${row.side}" is a buy or a sell`);
      fills.push({ row: i + 1, ticker, time: when, qty: signed, price, fees: rowFees(row) });
    } catch (e) { errors.push({ row: i + 1, message: e.message }); }
  });
  fills.sort((a, b) => (a.ticker < b.ticker ? -1 : a.ticker > b.ticker ? 1 : 0) || (a.time < b.time ? -1 : a.time > b.time ? 1 : 0) || a.row - b.row);
  const trades = [], acct = opts.account_type || "paper";
  const finish = (cur, ticker) => {
    const t = { ticker, direction: cur.direction, entry_time: cur.entry_time, entry_price: Math.round(cur.entry_val / cur.entry_qty * 10000) / 10000,
      shares: cur.exit_qty || cur.entry_qty, fees: Math.round(cur.fees * 100) / 100, account_type: acct, exit_time: null, exit_price: null,
      trader: opts.trader || null };
    if (cur.exit_qty) { t.exit_time = cur.exit_time; t.exit_price = Math.round(cur.exit_val / cur.exit_qty * 10000) / 10000; }
    trades.push(cleanTrade(t));
  };
  const byTicker = new Map();
  for (const f of fills) { if (!byTicker.has(f.ticker)) byTicker.set(f.ticker, []); byTicker.get(f.ticker).push(f); }
  for (const [ticker, tf] of byTicker) {
    let pos = 0, cur = null;
    for (const f of tf) {
      let q = f.qty;
      const totalQ = Math.abs(q);
      while (Math.abs(q) > 1e-9) {
        if (Math.abs(pos) < 1e-9) {
          cur = { direction: q > 0 ? "long" : "short", entry_time: f.time, entry_qty: 0, entry_val: 0, exit_qty: 0, exit_val: 0, fees: 0, exit_time: null };
          pos = 0;
        }
        let part;
        if (pos === 0 || (q > 0) === (pos > 0)) {
          part = Math.abs(q);
          cur.entry_qty += part; cur.entry_val += part * f.price; pos += q; q = 0;
        } else {
          part = Math.min(Math.abs(q), Math.abs(pos));
          cur.exit_qty += part; cur.exit_val += part * f.price; cur.exit_time = f.time;
          const step = q > 0 ? part : -part;
          pos += step; q -= step;
        }
        cur.fees += f.fees * part / totalQ;
        if (Math.abs(pos) < 1e-9) { finish(cur, ticker); cur = null; }
      }
    }
    if (cur) {
      warnings.push({ row: null, message: cur.exit_qty
        ? `${ticker}: ${Math.abs(pos)} shares still open at the end of the file; only the closed ${cur.exit_qty} shares were imported as a trade.`
        : `${ticker}: position of ${Math.abs(pos)} shares is still open; imported as an open trade.` });
      finish(cur, ticker);
    }
  }
  trades.sort((a, b) => (a.entry_time < b.entry_time ? -1 : a.entry_time > b.entry_time ? 1 : 0));
  return [trades, errors, warnings];
}

function importRows(rows, opts, existing) {
  const [trades, errors, warnings] = (opts.mode === "fills" ? fillsToTrades : rowsToTrades)(rows, opts);
  const key = t => [t.trader || "", t.ticker, String(t.entry_time).slice(0, 19).replace(" ", "T"), Math.round(t.entry_price * 10000), t.shares, t.account_type].join("|");
  const seen = new Set(existing.map(key));
  const fresh = [];
  let dupes = 0;
  for (const t of trades) {
    if (seen.has(key(t))) { dupes++; continue; }
    seen.add(key(t));
    fresh.push(t);
  }
  if (dupes) warnings.push({ row: null, message: `Skipped ${dupes} trade(s) that are already in your journal.` });
  return { trades: fresh, errors, warnings };
}

function tradesToCSV(trades) {
  const cell = v => {
    const s = v === null || v === undefined ? "" : String(v);
    return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  return [["id", ...TRADE_FIELDS].join(","), ...trades.map(t => [t.id, ...TRADE_FIELDS.map(f => t[f])].map(cell).join(","))].join("\n") + "\n";
}
// ==== LOGIC END ====
