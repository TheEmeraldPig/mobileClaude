"""Checks for the math and the CSV importer.  Run:  python -m unittest discover tests"""
import csv
import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, ROOT)
os.environ["ADVISOR_DATA_DIR"] = tempfile.mkdtemp(prefix="advisor-test-")

import analysis  # noqa: E402
import db  # noqa: E402
import importer  # noqa: E402

SETTINGS = dict(db.DEFAULT_SETTINGS, account_size=25000, max_risk_pct=1, daily_loss_limit=300,
                max_trades_per_day=3, min_rr=2, revenge_minutes=15,
                no_trade_windows=[{"start": "09:30", "end": "09:45", "label": "open"}])
_next_id = [0]


def T(entry_time, exit_time, entry, exit_, shares=100, direction="long", stop=None, target=None, fees=0, acct="paper", **kw):
    _next_id[0] += 1
    t = db.clean_trade(dict(ticker=kw.pop("ticker", "AAPL"), direction=direction, entry_time=entry_time, exit_time=exit_time,
                            entry_price=entry, exit_price=exit_, shares=shares, stop_loss=stop, target=target,
                            fees=fees, account_type=acct, **kw))
    t["id"] = _next_id[0]
    return t


def flags_of(trades, settings=SETTINGS):
    return {t["id"]: t["flags"] for t in analysis.prepare(trades, settings)}


class PnLTests(unittest.TestCase):
    def test_long_and_short_pnl_with_fees(self):
        long_ = analysis.enrich(T("2026-03-02T10:00", "2026-03-02T10:30", 100, 102, shares=50, fees=2, stop=99), SETTINGS)
        self.assertAlmostEqual(long_["pnl"], 98)
        self.assertAlmostEqual(long_["risk_dollars"], 50)
        self.assertAlmostEqual(long_["r_multiple"], 98 / 50)
        self.assertEqual(long_["hold_minutes"], 30)
        short = analysis.enrich(T("2026-03-02T10:00", "2026-03-02T10:05", 50, 49, direction="short", stop=50.5, target=48), SETTINGS)
        self.assertAlmostEqual(short["pnl"], 100)
        self.assertAlmostEqual(short["rr"], 4)

    def test_summary_numbers(self):
        trades = [T("2026-03-02T10:00", "2026-03-02T10:10", 10, 12),    # +200
                  T("2026-03-02T11:00", "2026-03-02T11:10", 10, 9),     # -100
                  T("2026-03-03T10:00", "2026-03-03T10:10", 10, 9.5),   # -50
                  T("2026-03-03T11:00", "2026-03-03T11:10", 10, 11)]    # +100
        closed = analysis.prepare(trades, SETTINGS)
        s = analysis.summary(closed, SETTINGS)
        self.assertEqual(s["trades"], 4)
        self.assertAlmostEqual(s["total_pnl"], 150)
        self.assertAlmostEqual(s["win_rate"], 0.5)
        self.assertAlmostEqual(s["avg_win"], 150)
        self.assertAlmostEqual(s["avg_loss"], 75)
        self.assertAlmostEqual(s["expectancy"], 37.5)
        self.assertAlmostEqual(s["profit_factor"], 2)
        self.assertAlmostEqual(s["max_drawdown"], 150)  # 25200 peak -> 25050
        self.assertEqual(s["longest_loss_streak"], 2)
        self.assertEqual(s["current_streak"], {"kind": "win", "length": 1})
        self.assertEqual(s["equity_curve"][-1]["equity"], 25150)

    def test_open_trades_are_not_counted(self):
        trades = [T("2026-03-02T10:00", None, 10, None), T("2026-03-02T10:00", "2026-03-02T11:00", 10, 11)]
        r = analysis.full_report(trades, SETTINGS)
        self.assertEqual(r["summary"]["trades"], 1)
        self.assertEqual(r["open_trades"], 1)


class FlagTests(unittest.TestCase):
    def test_risk_stop_and_rr_flags(self):
        a = T("2026-03-02T10:00", "2026-03-02T10:10", 100, 99, shares=300, stop=99, target=101)  # risk $300 > $250, rr 1
        b = T("2026-03-02T10:30", "2026-03-02T10:40", 100, 101, shares=10)                        # no stop
        c = T("2026-03-02T10:50", "2026-03-02T10:55", 100, 101, shares=10, stop=101)              # stop above long entry
        f = flags_of([a, b, c])
        self.assertIn("over_risk", f[a["id"]])
        self.assertIn("low_rr", f[a["id"]])
        self.assertIn("no_stop", f[b["id"]])
        self.assertIn("bad_stop", f[c["id"]])

    def test_revenge_overtrading_loss_limit_window(self):
        t1 = T("2026-03-02T09:35", "2026-03-02T09:50", 10, 8, stop=7.9)        # -200, in no-trade window
        t2 = T("2026-03-02T10:00", "2026-03-02T10:10", 10, 8.5, stop=8.4)      # 10 min after loss -> revenge; day now -350
        t3 = T("2026-03-02T11:00", "2026-03-02T11:10", 10, 10.5, stop=9.9)     # after loss limit
        t4 = T("2026-03-02T13:00", "2026-03-02T13:10", 10, 10.5, stop=9.9)     # 4th trade > max 3
        paper_other = T("2026-03-02T10:05", "2026-03-02T10:10", 10, 11, stop=9.9, acct="real")  # other account: not revenge
        f = flags_of([t1, t2, t3, t4, paper_other])
        self.assertIn("no_trade_window", f[t1["id"]])
        self.assertIn("revenge", f[t2["id"]])
        self.assertNotIn("revenge", f[t3["id"]])
        self.assertIn("after_loss_limit", f[t3["id"]])
        self.assertIn("overtrading", f[t4["id"]])
        self.assertNotIn("overtrading", f[t3["id"]])
        self.assertEqual(f[paper_other["id"]], [])

    def test_held_loser(self):
        wins = [T(f"2026-03-0{d}T10:00", f"2026-03-0{d}T10:10", 10, 11, stop=9.9) for d in (2, 3, 4)]
        loser = T("2026-03-05T10:00", "2026-03-05T11:00", 10, 9.9, stop=9.8)
        self.assertIn("held_loser", flags_of(wins + [loser])[loser["id"]])

    def test_mistake_costs_sorted_and_include_journal_tags(self):
        a = T("2026-03-02T10:00", "2026-03-02T10:10", 10, 9, mistakes="Chased entry")
        b = T("2026-03-02T10:20", "2026-03-02T10:30", 10, 10.1, stop=9.95)
        closed = analysis.prepare([a, b], SETTINGS)
        rows = analysis.mistake_costs(closed)["rows"]
        labels = [r["label"] for r in rows]
        self.assertIn("Chased entry", labels)
        self.assertIn("No stop-loss", labels)
        self.assertTrue(all(rows[i]["total_pnl"] <= rows[i + 1]["total_pnl"] for i in range(len(rows) - 1)))


class InsightAndPretradeTests(unittest.TestCase):
    def test_time_of_day_insight(self):
        trades = []
        for d in range(2, 12):
            loss = d < 9  # 7 of 10 early trades lose
            trades.append(T(f"2026-03-{d:02d}T09:35", f"2026-03-{d:02d}T09:40", 10, 9.5 if loss else 10.5, stop=9.4))
        texts = [i["text"] for i in analysis.full_report(trades, SETTINGS)["insights"]]
        self.assertTrue(any("between 9:30 and 9:45am lose 70% of the time" in t for t in texts), texts)

    def test_pretrade_levels(self):
        ok = analysis.pre_trade_check({"direction": "long", "entry": 100, "stop": 99, "target": 103, "shares": 100,
                                       "time": "2026-03-02T10:30", "account_type": "real"}, [], SETTINGS)
        self.assertEqual(ok["verdict"], "green")
        self.assertAlmostEqual(ok["numbers"]["risk_dollars"], 100)
        self.assertAlmostEqual(ok["numbers"]["rr"], 3)
        self.assertEqual(ok["numbers"]["suggested_max_shares"], 250)

        bad = analysis.pre_trade_check({"direction": "short", "entry": 100, "stop": 99, "target": 98, "shares": 100,
                                        "time": "2026-03-02T09:31"}, [], SETTINGS)
        titles = [c["title"] for c in bad["checks"] if c["level"] == "red"]
        self.assertIn("Stop is on the wrong side", titles)
        self.assertIn("Inside a no-trade window", titles)

    def test_pretrade_sees_todays_losses(self):
        t1 = T("2026-03-02T10:00", "2026-03-02T10:10", 10, 6.5, stop=6.4, acct="real")  # -350
        r = analysis.pre_trade_check({"direction": "long", "entry": 100, "stop": 99, "target": 103, "shares": 10,
                                      "time": "2026-03-02T10:20", "account_type": "real"}, [t1], SETTINGS)
        titles = [c["title"] for c in r["checks"]]
        self.assertIn("Daily loss limit already hit", titles)
        self.assertIn("Possible revenge trade", titles)
        self.assertEqual(r["verdict"], "red")


class ImporterTests(unittest.TestCase):
    def test_dates(self):
        self.assertEqual(importer.parse_datetime("03/02/2026", "01:05:00 PM")[0], "2026-03-02T13:05:00")
        self.assertEqual(importer.parse_datetime("2026-03-02 09:31:15")[0], "2026-03-02T09:31:15")
        self.assertEqual(importer.parse_datetime("3/2/26 9:31 AM")[0], "2026-03-02T09:31:00")
        self.assertEqual(importer.parse_datetime("03/02/2026 08:31", offset_hours=1)[0], "2026-03-02T09:31:00")
        v, warn = importer.parse_datetime("2026-03-02")
        self.assertEqual(v, "2026-03-02T09:30:00")
        self.assertTrue(warn)
        with self.assertRaises(ValueError):
            importer.parse_datetime("yesterday")

    def test_numbers_and_sides(self):
        self.assertEqual(importer.parse_number("$1,234.50"), 1234.5)
        self.assertEqual(importer.parse_number("(12.00)"), -12)
        self.assertEqual(importer.parse_direction("Sell Short"), "short")
        self.assertEqual(importer.parse_direction("BUY"), "long")

    def _csv(self, name, mapping):
        with open(os.path.join(ROOT, "samples", name), newline="") as f:
            return [{k: r[v] for k, v in mapping.items()} for r in csv.DictReader(f)]

    def test_sample_trades_file(self):
        rows = self._csv("sample_trades.csv", {"ticker": "Symbol", "direction": "Side", "entry_date": "Open Date",
                                               "entry_time": "Open Time", "entry_price": "Entry Price", "exit_date": "Close Date",
                                               "exit_time": "Close Time", "exit_price": "Exit Price", "shares": "Qty",
                                               "stop_loss": "Stop", "target": "Target", "fees": "Commission", "setup": "Setup"})
        trades, errors, _ = importer.import_rows(rows, {"mode": "trades", "account_type": "paper"}, [])
        self.assertEqual(errors, [])
        self.assertGreaterEqual(len(trades), 30)
        again, _, warnings = importer.import_rows(rows, {"mode": "trades", "account_type": "paper"}, trades)
        self.assertEqual(again, [])  # duplicates are skipped
        self.assertTrue(warnings)

    def test_fills_are_paired(self):
        rows = self._csv("sample_fills.csv", {"ticker": "Symbol", "side": "Action", "date": "Date/Time",
                                              "quantity": "Quantity", "price": "Price", "fees": "Commission"})
        trades, errors, warnings = importer.import_rows(rows, {"mode": "fills", "account_type": "real"}, [])
        self.assertEqual(errors, [])
        by = {t["ticker"]: t for t in trades}
        self.assertEqual(by["AAPL"]["shares"], 200)
        self.assertAlmostEqual(by["AAPL"]["entry_price"], 176.125)
        self.assertAlmostEqual(by["AAPL"]["exit_price"], 177.275)
        self.assertEqual(by["TSLA"]["direction"], "short")
        self.assertIsNone(by["AMD"]["exit_time"])  # still open
        self.assertTrue(any("AMD" in w["message"] for w in warnings))

    def test_flip_through_zero(self):
        rows = [{"ticker": "X", "side": "Buy", "date": "2026-03-02 10:00", "quantity": "100", "price": "10"},
                {"ticker": "X", "side": "Sell", "date": "2026-03-02 10:05", "quantity": "150", "price": "11"},
                {"ticker": "X", "side": "Buy", "date": "2026-03-02 10:10", "quantity": "50", "price": "10.5"}]
        trades, _, _ = importer.fills_to_trades(rows, {})
        self.assertEqual([(t["direction"], t["shares"]) for t in trades], [("long", 100), ("short", 50)])

    def test_bad_rows_reported(self):
        rows = [{"ticker": "", "entry_date": "2026-03-02", "entry_price": "10", "shares": "5"},
                {"ticker": "AAPL", "entry_date": "2026-03-02 10:00", "entry_price": "abc", "shares": "5"}]
        trades, errors, _ = importer.rows_to_trades(rows, {})
        self.assertEqual(trades, [])
        self.assertEqual(len(errors), 2)


if __name__ == "__main__":
    unittest.main()
