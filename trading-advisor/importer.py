"""Turn rows from any broker's CSV into trades.

The browser reads the CSV and applies your column mapping, then sends rows here
keyed by our own field names. Two shapes are supported:

* "trades": one row is a whole trade (entry and exit on the same row).
* "fills":  one row is a single buy or sell. Fills are paired into trades per
  ticker: a trade opens when the position leaves 0 and closes when it's back at 0.
"""
import re
from datetime import datetime, timedelta

import db

DATE_FORMATS = ["%Y-%m-%d", "%m/%d/%Y", "%m/%d/%y", "%Y/%m/%d", "%d-%b-%Y", "%d-%b-%y", "%b %d %Y", "%b %d, %Y", "%Y%m%d"]
TIME_FORMATS = ["%H:%M:%S", "%H:%M", "%I:%M:%S %p", "%I:%M %p", "%I:%M:%S%p", "%I:%M%p", "%H:%M:%S.%f"]

LONG_WORDS = {"long", "l", "buy", "b", "bot", "bought", "buy to open", "bto", "buy long"}
SHORT_WORDS = {"short", "sell short", "ss", "short sell", "sell to open", "sto", "sshrt", "sht", "s"}
BUY_WORDS = {"buy", "b", "bot", "bought", "buy to open", "bto", "buy to cover", "btc", "cover", "long", "buy long"}
SELL_WORDS = {"sell", "s", "sld", "sold", "sell short", "ss", "short", "short sell", "sell to open", "sto",
              "sell to close", "stc", "sshrt", "sht"}


def parse_number(v):
    if v is None:
        return None
    s = str(v).strip()
    if s == "" or s == "-" or s.lower() in ("n/a", "na", "none", "null"):
        return None
    neg = s.startswith("(") and s.endswith(")")
    s = re.sub(r"[,$\s()]", "", s)
    n = float(s)
    return -n if neg else n


def parse_datetime(date_str, time_str=None, offset_hours=0.0):
    """Parse a broker date (and optional separate time) into 'YYYY-MM-DDTHH:MM:SS'.

    Returns (value, warning). offset_hours shifts the time into US Eastern
    (e.g. +1 if your broker reports Central time).
    """
    d = (date_str or "").strip()
    t = (time_str or "").strip()
    if not d:
        raise ValueError("missing date")
    # drop trailing timezone names like " ET", " EST", "Z"
    d = re.sub(r"\s*(ET|EST|EDT|CT|CST|CDT|PT|PST|PDT|MT|MST|MDT|UTC|GMT)$", "", d, flags=re.I)
    t = re.sub(r"\s*(ET|EST|EDT|CT|CST|CDT|PT|PST|PDT|MT|MST|MDT|UTC|GMT)$", "", t, flags=re.I)
    warning = None
    dt = None
    try:
        dt = datetime.fromisoformat(d.replace("Z", "").replace("/", "-") if re.match(r"^\d{4}[-/]", d) else d)
        if dt.tzinfo:
            dt = dt.replace(tzinfo=None)
    except ValueError:
        pass
    if dt is None:
        for df in DATE_FORMATS:
            for tf in [None] + TIME_FORMATS:
                fmt = df if tf is None else f"{df} {tf}"
                try:
                    dt = datetime.strptime(d.upper() if tf and "%p" in tf else d, fmt)
                    break
                except ValueError:
                    continue
            if dt:
                break
    if dt is None:
        raise ValueError(f'could not read the date "{date_str}"')
    if t:
        tm = None
        for tf in TIME_FORMATS:
            try:
                tm = datetime.strptime(t.upper(), tf)
                break
            except ValueError:
                continue
        if tm is None:
            raise ValueError(f'could not read the time "{time_str}"')
        dt = dt.replace(hour=tm.hour, minute=tm.minute, second=tm.second)
    elif dt.hour == 0 and dt.minute == 0 and dt.second == 0 and not re.search(r"\d:\d", d):
        dt = dt.replace(hour=9, minute=30)
        warning = "no time found, used 9:30am"
    if offset_hours:
        dt += timedelta(hours=float(offset_hours))
    return dt.strftime("%Y-%m-%dT%H:%M:%S"), warning


def parse_direction(v, default="long"):
    s = (v or "").strip().lower()
    if not s:
        return default
    if s in LONG_WORDS:
        return "long"
    if s in SHORT_WORDS or s.startswith("short") or s.startswith("sell short"):
        return "short"
    if s.startswith("long") or s.startswith("buy"):
        return "long"
    raise ValueError(f'could not tell if "{v}" is long or short')


def _fees(row):
    total = 0.0
    for k in ("fees", "fees2"):
        n = parse_number(row.get(k))
        if n:
            total += abs(n)
    return total


def rows_to_trades(rows, options):
    """Round-trip mode. Returns (trades, errors, warnings)."""
    trades, errors, warnings = [], [], []
    off = float(options.get("tz_offset_hours") or 0)
    default_dir = options.get("default_direction") or "long"
    for i, row in enumerate(rows, start=1):
        try:
            entry_time, w1 = parse_datetime(row.get("entry_date"), row.get("entry_time"), off)
            exit_time, w2 = (None, None)
            if (row.get("exit_date") or "").strip():
                exit_time, w2 = parse_datetime(row.get("exit_date"), row.get("exit_time"), off)
            for w in (w1, w2):
                if w:
                    warnings.append({"row": i, "message": w})
            shares = parse_number(row.get("shares"))
            trade = db.clean_trade({
                "ticker": row.get("ticker"),
                "direction": parse_direction(row.get("direction"), default_dir),
                "entry_time": entry_time,
                "entry_price": parse_number(row.get("entry_price")),
                "exit_time": exit_time,
                "exit_price": parse_number(row.get("exit_price")) if exit_time else None,
                "shares": abs(shares) if shares is not None else None,
                "stop_loss": parse_number(row.get("stop_loss")),
                "target": parse_number(row.get("target")),
                "fees": _fees(row),
                "account_type": options.get("account_type") or "paper",
                "setup": row.get("setup"),
                "notes": row.get("notes"),
            })
            trades.append(trade)
        except (ValueError, TypeError) as e:
            errors.append({"row": i, "message": str(e)})
    return trades, errors, warnings


def fills_to_trades(rows, options):
    """Fills mode: pair individual buys/sells into round-trip trades (per ticker)."""
    errors, warnings, fills = [], [], []
    off = float(options.get("tz_offset_hours") or 0)
    for i, row in enumerate(rows, start=1):
        try:
            ticker = (row.get("ticker") or "").strip().upper()
            if not ticker:
                raise ValueError("ticker is missing")
            when, w = parse_datetime(row.get("date"), row.get("time"), off)
            if w:
                warnings.append({"row": i, "message": w})
            qty = parse_number(row.get("quantity"))
            price = parse_number(row.get("price"))
            if not qty or not price or price <= 0:
                raise ValueError("quantity and price must be numbers above 0")
            side = (row.get("side") or "").strip().lower()
            if side in BUY_WORDS or side.startswith("buy") or side.startswith("bought"):
                signed = abs(qty)
            elif side in SELL_WORDS or side.startswith("sell") or side.startswith("sold") or side.startswith("short"):
                signed = -abs(qty)
            elif not side:
                signed = qty  # some brokers use negative quantity for sells
            else:
                raise ValueError(f'could not tell if "{row.get("side")}" is a buy or a sell')
            fills.append({"row": i, "ticker": ticker, "time": when, "qty": signed, "price": price, "fees": _fees(row)})
        except (ValueError, TypeError) as e:
            errors.append({"row": i, "message": str(e)})

    fills.sort(key=lambda f: (f["ticker"], f["time"], f["row"]))
    trades = []
    acct = options.get("account_type") or "paper"

    def finish(cur, ticker):
        shares = cur["exit_qty"] if cur["exit_qty"] else cur["entry_qty"]
        t = {"ticker": ticker, "direction": cur["direction"], "entry_time": cur["entry_time"],
             "entry_price": round(cur["entry_val"] / cur["entry_qty"], 4), "shares": shares,
             "fees": round(cur["fees"], 2), "account_type": acct, "exit_time": None, "exit_price": None}
        if cur["exit_qty"]:
            t["exit_time"] = cur["exit_time"]
            t["exit_price"] = round(cur["exit_val"] / cur["exit_qty"], 4)
        trades.append(db.clean_trade(t))

    by_ticker = {}
    for f in fills:
        by_ticker.setdefault(f["ticker"], []).append(f)
    for ticker, tfills in by_ticker.items():
        pos, cur = 0.0, None
        for f in tfills:
            q = f["qty"]
            total_q = abs(q)
            while abs(q) > 1e-9:
                if abs(pos) < 1e-9:
                    cur = {"direction": "long" if q > 0 else "short", "entry_time": f["time"], "entry_qty": 0.0,
                           "entry_val": 0.0, "exit_qty": 0.0, "exit_val": 0.0, "fees": 0.0, "exit_time": None}
                    pos = 0.0
                if pos == 0 or (q > 0) == (pos > 0):
                    part = abs(q)
                    cur["entry_qty"] += part
                    cur["entry_val"] += part * f["price"]
                    pos += q
                    q = 0.0
                else:
                    part = min(abs(q), abs(pos))
                    cur["exit_qty"] += part
                    cur["exit_val"] += part * f["price"]
                    cur["exit_time"] = f["time"]
                    step = part if q > 0 else -part
                    pos += step
                    q -= step
                cur["fees"] += f["fees"] * part / total_q
                if abs(pos) < 1e-9:
                    finish(cur, ticker)
                    cur = None
        if cur is not None:
            if cur["exit_qty"]:
                warnings.append({"row": None, "message": f"{ticker}: {abs(pos):g} shares still open at the end of the file; only the closed {cur['exit_qty']:g} shares were imported as a trade."})
            else:
                warnings.append({"row": None, "message": f"{ticker}: position of {abs(pos):g} shares is still open; imported as an open trade."})
            finish(cur, ticker)
    trades.sort(key=lambda t: t["entry_time"])
    return trades, errors, warnings


def import_rows(rows, options, existing):
    mode = options.get("mode") or "trades"
    trades, errors, warnings = (fills_to_trades if mode == "fills" else rows_to_trades)(rows, options)
    seen = {(t["ticker"], t["entry_time"], round(t["entry_price"], 4), t["shares"], t["account_type"]) for t in existing}
    fresh, dupes = [], 0
    for t in trades:
        key = (t["ticker"], t["entry_time"], round(t["entry_price"], 4), t["shares"], t["account_type"])
        if key in seen:
            dupes += 1
            continue
        seen.add(key)
        fresh.append(t)
    if dupes:
        warnings.append({"row": None, "message": f"Skipped {dupes} trade(s) that are already in your journal."})
    return fresh, errors, warnings
