// Checks the website's math (web/logic.js) against the sample files.  Run:  node tests/web_logic_test.js
const assert = require("assert");
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const root = path.join(__dirname, "..");
const ctx = {};
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(path.join(root, "web/logic.js"), "utf8") +
  ";this.L={parseCSV,importRows,fullReport,preTradeCheck,DEFAULT_SETTINGS,parseBrokerDate,cleanTrade,traderComparison,readiness};", ctx);
const L = ctx.L;
const read = f => fs.readFileSync(path.join(root, "samples", f), "utf8");
const map = (csv, m) => L.parseCSV(csv).rows.map(r => Object.fromEntries(Object.entries(m).map(([k, h]) => [k, r[h]])));

const trades = L.importRows(map(read("sample_trades.csv"), {
  ticker: "Symbol", direction: "Side", entry_date: "Open Date", entry_time: "Open Time", entry_price: "Entry Price",
  exit_date: "Close Date", exit_time: "Close Time", exit_price: "Exit Price", shares: "Qty", stop_loss: "Stop",
  target: "Target", fees: "Commission", setup: "Setup" }), { mode: "trades", account_type: "paper" }, []);
assert.strictEqual(trades.errors.length, 0);
assert.strictEqual(trades.trades.length, 32);

const fills = L.importRows(map(read("sample_fills.csv"), {
  ticker: "Symbol", side: "Action", date: "Date/Time", quantity: "Quantity", price: "Price", fees: "Commission" }),
  { mode: "fills", account_type: "real" }, trades.trades);
const aapl = fills.trades.find(t => t.ticker === "AAPL");
assert.strictEqual(aapl.shares, 200);
assert.strictEqual(aapl.entry_price, 176.125);
assert.strictEqual(fills.trades.find(t => t.ticker === "AMD").exit_time, null);

// re-importing the same file adds nothing
assert.strictEqual(L.importRows(map(read("sample_trades.csv"), { ticker: "Symbol", entry_date: "Open Date", entry_time: "Open Time",
  entry_price: "Entry Price", shares: "Qty" }), { account_type: "paper" }, trades.trades).trades.length, 0);

// numbers match the tested Python version for the same data
const all = [...trades.trades, ...fills.trades].map((t, i) => ({ ...t, id: "t" + String(i).padStart(3, "0") }));
const r = L.fullReport(all, L.DEFAULT_SETTINGS, {});
assert.strictEqual(r.summary.total_pnl.toFixed(2), "15.00");
assert.strictEqual(r.summary.profit_factor.toFixed(4), "1.0049");
assert.strictEqual(r.summary.max_drawdown.toFixed(2), "803.00");
assert.ok(r.insights.some(i => i.text.includes("between 9:30 and 9:45am lose 64% of the time")));
assert.strictEqual(JSON.stringify(r.mistakes.rows.slice(0, 2).map(m => [m.label, m.count])), JSON.stringify([["Held a loser too long", 10], ["Traded in a no-trade window", 13]]));

// pre-trade check
const ok = L.preTradeCheck({ direction: "long", entry: 100, stop: 99, target: 103, shares: 100, time: "2026-03-02T10:30", account_type: "real" }, [], L.DEFAULT_SETTINGS);
assert.strictEqual(ok.verdict, "green");
assert.strictEqual(ok.numbers.suggested_max_shares, 250);
const bad = L.preTradeCheck({ direction: "short", entry: 100, stop: 99, target: 98, shares: 100, time: "2026-03-02T09:31" }, [], L.DEFAULT_SETTINGS);
assert.ok(bad.checks.some(c => c.title === "Stop is on the wrong side"));
assert.ok(bad.checks.some(c => c.title === "Inside a no-trade window"));

// broker date formats
const D = (...a) => L.parseBrokerDate(...a)[0];
assert.strictEqual(D("03/02/2026", "01:05:00 PM"), "2026-03-02T13:05:00");
assert.strictEqual(D("3/2/26 9:31 AM"), "2026-03-02T09:31:00");
assert.strictEqual(D("2026-03-02T09:35:00Z"), "2026-03-02T09:35:00");
assert.strictEqual(D("02-Mar-2026 14:05"), "2026-03-02T14:05:00");
assert.strictEqual(D("Mar 2, 2026", "2:05 PM"), "2026-03-02T14:05:00");
assert.strictEqual(D("03/02/2026 08:31", "", 1), "2026-03-02T09:31:00");
assert.throws(() => D("yesterday"));
assert.strictEqual(L.cleanTrade({ ticker: "nasdaq:aapl", direction: "long", entry_time: "2026-03-02T10:00", entry_price: 1, shares: 1 }).ticker, "AAPL");
assert.throws(() => L.cleanTrade({ ticker: "X", direction: "long", entry_time: "2026-03-02T10:00", entry_price: 0, shares: 1 }));

console.log("web logic: all checks passed");

// other traders: kept apart from your stats, flags and limits
const other = L.importRows(map(read("sample_other_trader.csv"), {
  ticker: "Ticker", direction: "Direction", entry_date: "Entry Time", entry_price: "Entry", exit_date: "Exit Time",
  exit_price: "Exit", shares: "Shares", stop_loss: "Stop Loss", target: "Profit Target", fees: "Fees", setup: "Strategy" }),
  { mode: "trades", account_type: "paper", trader: "Sample Trader" }, all);
assert.strictEqual(other.errors.length, 0);
assert.strictEqual(other.trades.length, 22);
assert.ok(other.trades.every(t => t.trader === "Sample Trader"));
const withOthers = [...all, ...other.trades.map((t, i) => ({ ...t, id: "o" + String(i).padStart(3, "0") }))];
const mine = L.fullReport(withOthers, L.DEFAULT_SETTINGS, {});
assert.strictEqual(mine.summary.total_pnl.toFixed(2), "15.00"); // unchanged by their trades
const theirs = L.fullReport(withOthers, L.DEFAULT_SETTINGS, { who: "t:Sample Trader" });
assert.strictEqual(theirs.summary.trades, 22);
assert.ok(theirs.insights.every(i => !/\bYour\b|\byou\b/.test(i.text)), JSON.stringify(theirs.insights));
const cmp = L.traderComparison(withOthers, L.DEFAULT_SETTINGS, {});
assert.strictEqual(JSON.stringify(cmp.map(c => [c.name, c.trades])), JSON.stringify([["You", 36], ["Sample Trader", 22]]));
// the same trades for a different person are not duplicates
assert.strictEqual(L.importRows(map(read("sample_trades.csv"), { ticker: "Symbol", entry_date: "Open Date", entry_time: "Open Time",
  entry_price: "Entry Price", shares: "Qty" }), { account_type: "paper", trader: "Someone" }, all).trades.length, 32);
// pre-trade: their history shown separately; their trades don't count toward your daily limits
const pt = L.preTradeCheck({ direction: "long", entry: 100, stop: 99, target: 103, shares: 100, setup: "Pullback",
  time: "2026-03-09T10:45", account_type: "paper" }, withOthers, L.DEFAULT_SETTINGS);
assert.ok(pt.others_history.some(h => h.label === 'Others: setup "Pullback"' && h.count === 9));
assert.strictEqual(pt.numbers.trades_today, 2);

console.log("other traders: all checks passed");

// market conditions: breakdowns, insights and pre-trade history use the saved context
const ctxFor = (gap, rv, spy, vwap) => ({ daily: { gap_pct: gap, rel_volume: rv }, market: { spy_change_pct: spy }, entry_vs_vwap: vwap, news: [] });
const contexts = {};
all.forEach((t, i) => { contexts[t.id] = i % 2 ? ctxFor(3, 2.5, 0.8, "above") : ctxFor(0.5, 0.7, -0.6, "below"); });
const withCtx = L.fullReport(all, L.DEFAULT_SETTINGS, {}, contexts);
assert.ok(withCtx.breakdowns.ctx_gap && withCtx.breakdowns.ctx_rel_volume && withCtx.breakdowns.ctx_market && withCtx.breakdowns.ctx_vwap);
assert.strictEqual(withCtx.with_context, 36);
assert.strictEqual(withCtx.breakdowns.ctx_gap.rows.reduce((a, r) => a + r.count, 0), 36);
assert.ok(!L.fullReport(all, L.DEFAULT_SETTINGS, {}).breakdowns.ctx_gap); // no context, no section
const ptc = L.preTradeCheck({ direction: "long", entry: 100, stop: 99, target: 103, shares: 100, time: "2026-03-02T10:30", account_type: "paper",
  conditions: { gap: "Gapped up 2%+", vwap: "Above VWAP" } }, all, L.DEFAULT_SETTINGS, null, contexts);
assert.ok(ptc.history.some(h => h.label === "Stock's gap at the open: Gapped up 2%+" && h.count > 0));

// readiness: only your paper trades count, and a short good run isn't enough
const rd = L.readiness(withOthers, L.DEFAULT_SETTINGS);
assert.strictEqual(rd.summary.trades, 32);            // sample paper trades only (fills were real, other trader excluded)
assert.strictEqual(rd.ready, false);
assert.ok(rd.criteria.find(c => c.label === "Paper trades").pass === false);
// a long, steady, rule-following paper record passes every check
const good = [];
let day = new Date(Date.UTC(2026, 0, 5));
for (let d = 0; d < 25; d++) {
  while ([0, 6].includes(day.getUTCDay())) day = new Date(day.getTime() + 864e5);
  const ds = day.toISOString().slice(0, 10);
  for (let k = 0; k < 4; k++) {
    const win = !(d % 3 === 0 && k < 2) && k !== 3; // most days 3 wins / 1 loss, every third day 1 win / 3 losses
    good.push({ id: `g${d}-${k}`, ticker: "AAPL", direction: "long", account_type: "paper", fees: 0,
      entry_time: `${ds}T1${k}:00:00`, exit_time: `${ds}T1${k}:20:00`, entry_price: 100, stop_loss: 99.5, target: 101.5,
      exit_price: win ? 101.2 : 99.6, shares: 100 });
  }
  day = new Date(day.getTime() + 864e5);
}
const settingsNoWindows = { ...L.DEFAULT_SETTINGS, no_trade_windows: [] };
const rg = L.readiness(good, settingsNoWindows);
assert.strictEqual(rg.ready, true, JSON.stringify(rg.criteria.filter(c => !c.pass)));
assert.strictEqual(rg.criteria.length, 10);

console.log("market conditions and readiness: all checks passed");
