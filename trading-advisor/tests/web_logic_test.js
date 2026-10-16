// Checks the website's math (web/logic.js) against the sample files.  Run:  node tests/web_logic_test.js
const assert = require("assert");
const fs = require("fs");
const path = require("path");
const vm = require("vm");

const root = path.join(__dirname, "..");
const ctx = {};
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(path.join(root, "web/logic.js"), "utf8") +
  ";this.L={parseCSV,importRows,fullReport,preTradeCheck,DEFAULT_SETTINGS,parseBrokerDate,cleanTrade};", ctx);
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
assert.throws(() => L.cleanTrade({ ticker: "X", direction: "long", entry_time: "2026-03-02T10:00", entry_price: 0, shares: 1 }));

console.log("web logic: all checks passed");
