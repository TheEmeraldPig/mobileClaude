#!/usr/bin/env python3
"""Fetch what the market looked like for each trade. Read-only: it never places orders. Standard library only.

For each trade it saves:
  - the stock's 5-minute candles for the trade day (9:30am to 4pm ET)
  - the day's open/high/low/close, gap from yesterday, relative volume (vs 20-day average), ATR(14)
  - what SPY (the overall market) did that day
  - VWAP at the moment of entry, and whether the entry was above or below it
  - news headlines about the stock from the evening before through that day

Usage:
  market_context.py TRADES_DIR OUT_DIR [--feed sip|iex] [--only-missing EXISTING_CONTEXT_DIR]

TRADES_DIR is the trades collection exported from the website (one JSON file per trade).
OUT_DIR gets one JSON file per trade, named by trade id, ready to save into the website's "context" collection.

Data comes from Alpaca's market-data API (a free account works). It needs:
  ALPACA_DATA_KEY_ID and ALPACA_DATA_SECRET_KEY set as environment secrets, and
  data.alpaca.markets allowed in the environment's network settings.
"""
import argparse
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import date, datetime, timedelta, timezone
from zoneinfo import ZoneInfo

ET = ZoneInfo("America/New_York")
DATA_BASE = "https://data.alpaca.markets"


class DataError(Exception):
    pass


def get(path, params, opener=urllib.request.urlopen):
    key, secret = os.environ.get("ALPACA_DATA_KEY_ID"), os.environ.get("ALPACA_DATA_SECRET_KEY")
    if not key or not secret:
        raise DataError("ALPACA_DATA_KEY_ID and ALPACA_DATA_SECRET_KEY are not set. Add them as secrets in the environment settings.")
    url = DATA_BASE + path + "?" + urllib.parse.urlencode({k: v for k, v in params.items() if v is not None})
    req = urllib.request.Request(url, headers={"APCA-API-KEY-ID": key, "APCA-API-SECRET-KEY": secret, "Accept": "application/json"})
    try:
        with opener(req, timeout=20) as res:
            return json.loads(res.read())
    except urllib.error.HTTPError as e:
        raise DataError(f"market data said {e.code}: {e.read().decode(errors='replace')[:200]}")
    except urllib.error.URLError as e:
        raise DataError(f"couldn't reach {DATA_BASE} ({e.reason}). Is data.alpaca.markets allowed in the network settings?")


def to_et_str(iso):
    """UTC timestamp -> 'YYYY-MM-DDTHH:MM:SS' in US Eastern."""
    return datetime.fromisoformat(iso.replace("Z", "+00:00")).astimezone(ET).strftime("%Y-%m-%dT%H:%M:%S")


def to_et_date(iso):
    return date.fromisoformat(to_et_str(iso)[:10])


def et_bound(day, hm):
    h, m = map(int, hm.split(":"))
    return datetime(day.year, day.month, day.day, h, m, tzinfo=ET).astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


class Fetcher:
    def __init__(self, feed="sip", opener=urllib.request.urlopen):
        self.feed, self.opener, self.cache = feed, opener, {}

    def get(self, path, params):
        key = (path, json.dumps(params, sort_keys=True))
        if key not in self.cache:
            try:
                self.cache[key] = get(path, params, self.opener)
            except DataError as e:
                # free accounts can't read the newest SIP data; the IEX feed always works
                if params.get("feed") == "sip" and ("403" in str(e) or "subscription" in str(e).lower()):
                    self.feed = "iex"
                    return self.get(path, {**params, "feed": "iex"})
                raise
        return self.cache[key]

    def bars(self, symbol, timeframe, start, end):
        out, token = [], None
        while True:
            res = self.get(f"/v2/stocks/{symbol}/bars", {"timeframe": timeframe, "start": start, "end": end, "limit": 10000,
                                                         "adjustment": "raw", "feed": self.feed, "page_token": token})
            out += res.get("bars") or []
            token = res.get("next_page_token")
            if not token:
                return out

    def daily(self, symbol, day):
        bars = self.bars(symbol, "1Day", (day - timedelta(days=45)).isoformat(), (day + timedelta(days=1)).isoformat())
        return [b for b in bars if to_et_date(b["t"]) <= day]

    def news(self, symbol, day):
        res = self.get("/v1beta1/news", {"symbols": symbol, "start": et_bound(day - timedelta(days=1), "16:00"),
                                         "end": et_bound(day, "16:00"), "limit": 10, "sort": "asc"})
        return [{"time": to_et_str(n["created_at"]), "headline": n.get("headline", ""), "source": n.get("source", "")}
                for n in res.get("news") or []]


def daily_stats(dbars, day):
    if not dbars or to_et_date(dbars[-1]["t"]) != day:
        return None
    today, prev = dbars[-1], dbars[:-1]
    out = {"open": today["o"], "high": today["h"], "low": today["l"], "close": today["c"], "volume": today["v"]}
    if prev:
        pc = prev[-1]["c"]
        out.update(prev_close=pc, gap_pct=round((today["o"] - pc) / pc * 100, 2), change_pct=round((today["c"] - pc) / pc * 100, 2))
        vols = [b["v"] for b in prev[-20:]]
        avg = sum(vols) / len(vols)
        out.update(avg_volume_20=round(avg), rel_volume=round(today["v"] / avg, 2) if avg else None)
        trs = [max(prev[i]["h"] - prev[i]["l"], abs(prev[i]["h"] - prev[i - 1]["c"]), abs(prev[i]["l"] - prev[i - 1]["c"]))
               for i in range(max(1, len(prev) - 14), len(prev))]
        if trs:
            out["atr14"] = round(sum(trs) / len(trs), 2)
    return out


def vwap_at(bars, entry_hm):
    """VWAP from the bars that started before the entry time. bars: [hh:mm, o, h, l, c, v]."""
    pv = vol = 0.0
    for b in bars:
        if b[0] >= entry_hm:
            break
        pv += (b[2] + b[3] + b[4]) / 3 * b[5]
        vol += b[5]
    return round(pv / vol, 4) if vol else None


def build_context(trade, f):
    symbol = trade["ticker"]
    entry = str(trade["entry_time"]).replace(" ", "T")
    day = date.fromisoformat(entry[:10])
    raw = f.bars(symbol, "5Min", et_bound(day, "09:30"), et_bound(day, "16:00"))
    bars = [[to_et_str(b["t"])[11:16], b["o"], b["h"], b["l"], b["c"], b["v"]] for b in raw]
    spy = daily_stats(f.daily("SPY", day), day)
    vw = vwap_at(bars, entry[11:16])
    try:
        news = f.news(symbol, day)
    except DataError:
        news = None
    return {"trade_id": trade["id"], "symbol": symbol, "date": day.isoformat(), "bars": bars,
            "daily": daily_stats(f.daily(symbol, day), day),
            "market": {"spy_change_pct": spy.get("change_pct"), "spy_gap_pct": spy.get("gap_pct")} if spy else None,
            "vwap_at_entry": vw, "entry_vs_vwap": None if vw is None else ("above" if trade["entry_price"] >= vw else "below"),
            "news": news, "source": f"Alpaca market data ({f.feed.upper()} feed)",
            "fetched_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")}


def load_trades(path):
    """Every trade file in a directory (yours and other traders'). Id = 'id' field or the file name."""
    trades = []
    for root, _, files in os.walk(path):
        for fn in sorted(files):
            if fn.endswith(".json"):
                with open(os.path.join(root, fn)) as fh:
                    d = json.load(fh)
                doc = d.get("data", d) if isinstance(d, dict) else None
                if isinstance(doc, dict) and "ticker" in doc and doc.get("entry_time"):
                    trades.append({**doc, "id": d.get("id") or doc.get("id") or os.path.splitext(fn)[0]})
    return trades


def main(argv=None, opener=urllib.request.urlopen):
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("trades_dir"); p.add_argument("out_dir")
    p.add_argument("--feed", default="sip", choices=["sip", "iex"])
    p.add_argument("--only-missing", help="directory of context files already saved; those trades are skipped")
    a = p.parse_args(argv)
    trades = load_trades(a.trades_dir)
    have = set()
    if a.only_missing and os.path.isdir(a.only_missing):
        have = {os.path.splitext(fn)[0] for _, _, fs in os.walk(a.only_missing) for fn in fs if fn.endswith(".json")}
    os.makedirs(a.out_dir, exist_ok=True)
    f = Fetcher(a.feed, opener)
    saved, failed = 0, []
    for t in trades:
        if t["id"] in have:
            continue
        try:
            with open(os.path.join(a.out_dir, f"{t['id']}.json"), "w") as fh:
                json.dump(build_context(t, f), fh, separators=(",", ":"))
            saved += 1
        except DataError as e:
            failed.append(f"{t['ticker']} {t['entry_time']}: {e}")
            if "network settings" in str(e) or "not set" in str(e):
                break
    print(json.dumps({"saved": saved, "skipped_existing": len(have & {t["id"] for t in trades}), "failed": failed}, indent=1))
    return 0 if not failed else 1


if __name__ == "__main__":
    sys.exit(main())
