"""Checks for tools/market_context.py using a fake data server (no network).  Run:  python -m unittest discover tests"""
import io
import json
import os
import sys
import tempfile
import unittest
import urllib.parse
from datetime import date, datetime, timedelta, timezone

sys.path.insert(0, os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "tools"))
import market_context as mc  # noqa: E402


def utc(day, hm):
    """US Eastern day + HH:MM -> Alpaca-style UTC timestamp."""
    h, m = map(int, hm.split(":"))
    return datetime(day.year, day.month, day.day, h, m, tzinfo=mc.ET).astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


DAY = date(2026, 3, 2)  # a Monday


def fake_daily(symbol):
    # 25 previous weekdays: close 100, volume 1,000,000, range 2. Trade day: opens 103 (gap +3%), volume 3,000,000.
    bars, d = [], DAY - timedelta(days=40)
    while d < DAY:
        if d.weekday() < 5:
            bars.append({"t": utc(d, "00:00").replace("05:00:00", "05:00:00"), "o": 100, "h": 101, "l": 99, "c": 100, "v": 1_000_000})
        d += timedelta(days=1)
    close = 104 if symbol == "AAPL" else 99.5
    bars.append({"t": utc(DAY, "00:00"), "o": 103 if symbol == "AAPL" else 100, "h": 105, "l": 102, "c": close, "v": 3_000_000})
    return bars


class FakeOpener:
    def __init__(self):
        self.calls = []

    def __call__(self, req, timeout=None):
        u = urllib.parse.urlparse(req.full_url)
        q = dict(urllib.parse.parse_qsl(u.query))
        self.calls.append((u.path, q))
        assert req.get_header("Apca-api-key-id") == "k"
        if u.path.endswith("/bars") and q["timeframe"] == "1Day":
            body = {"bars": fake_daily(u.path.split("/")[3])}
        elif u.path.endswith("/bars"):
            # 9:30, 9:35, 9:40 five-minute bars
            body = {"bars": [{"t": utc(DAY, "09:30"), "o": 103, "h": 104, "l": 102, "c": 103.5, "v": 1000},
                             {"t": utc(DAY, "09:35"), "o": 103.5, "h": 105, "l": 103, "c": 104.5, "v": 3000},
                             {"t": utc(DAY, "09:40"), "o": 104.5, "h": 104.8, "l": 104, "c": 104.2, "v": 2000}]}
        elif u.path == "/v1beta1/news":
            body = {"news": [{"created_at": utc(DAY, "07:15"), "headline": "Company beats estimates", "source": "test"}]}
        else:
            raise AssertionError(u.path)
        return io.BytesIO(json.dumps(body).encode())


class MarketContextTests(unittest.TestCase):
    def setUp(self):
        os.environ["ALPACA_DATA_KEY_ID"] = "k"
        os.environ["ALPACA_DATA_SECRET_KEY"] = "s"

    def test_context_numbers(self):
        f = mc.Fetcher("iex", FakeOpener())
        ctx = mc.build_context({"id": "t1", "ticker": "AAPL", "entry_time": "2026-03-02T09:41:00", "entry_price": 104.6}, f)
        self.assertEqual([b[0] for b in ctx["bars"]], ["09:30", "09:35", "09:40"])  # converted to Eastern
        d = ctx["daily"]
        self.assertEqual(d["prev_close"], 100)
        self.assertEqual(d["gap_pct"], 3.0)
        self.assertEqual(d["change_pct"], 4.0)
        self.assertEqual(d["rel_volume"], 3.0)
        self.assertEqual(d["atr14"], 2.0)
        self.assertEqual(ctx["market"]["spy_change_pct"], -0.5)
        # VWAP from the 9:30, 9:35 and 9:40 bars (entry at 9:41)
        tp = [(104 + 102 + 103.5) / 3, (105 + 103 + 104.5) / 3, (104.8 + 104 + 104.2) / 3]
        expected = (tp[0] * 1000 + tp[1] * 3000 + tp[2] * 2000) / 6000
        self.assertAlmostEqual(ctx["vwap_at_entry"], round(expected, 4))
        self.assertEqual(ctx["entry_vs_vwap"], "above")
        self.assertEqual(ctx["news"][0]["time"], "2026-03-02T07:15:00")

    def test_main_writes_one_file_per_trade_and_skips_existing(self):
        with tempfile.TemporaryDirectory() as tmp:
            tdir, out = os.path.join(tmp, "trades"), os.path.join(tmp, "ctx")
            os.makedirs(tdir)
            for tid in ("a1", "b2"):
                with open(os.path.join(tdir, tid + ".json"), "w") as fh:
                    json.dump({"ticker": "AAPL", "entry_time": "2026-03-02T09:41:00", "entry_price": 104, "trader": "Sam" if tid == "b2" else None}, fh)
            opener = FakeOpener()
            self.assertEqual(mc.main([tdir, out, "--feed", "iex"], opener=opener), 0)
            self.assertEqual(sorted(os.listdir(out)), ["a1.json", "b2.json"])
            calls = len(opener.calls)
            self.assertEqual(mc.main([tdir, out, "--feed", "iex", "--only-missing", out], opener=opener), 0)
            self.assertEqual(len(opener.calls), calls)  # nothing fetched again

    def test_missing_keys_is_a_clear_error(self):
        del os.environ["ALPACA_DATA_KEY_ID"]
        with self.assertRaises(mc.DataError) as e:
            mc.get("/v2/stocks/AAPL/bars", {}, FakeOpener())
        self.assertIn("ALPACA_DATA_KEY_ID", str(e.exception))


if __name__ == "__main__":
    unittest.main()
