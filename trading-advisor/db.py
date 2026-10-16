"""Local storage: one SQLite file (data/trades.db) plus a screenshots folder.

Nothing here talks to the network. Everything stays on this computer.
"""
import json
import os
import sqlite3
import threading

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DATA_DIR = os.environ.get("ADVISOR_DATA_DIR", os.path.join(BASE_DIR, "data"))
SCREENSHOT_DIR = os.path.join(DATA_DIR, "screenshots")
DB_PATH = os.path.join(DATA_DIR, "trades.db")

# Every column a trade has, in order. Journal fields are optional.
TRADE_FIELDS = [
    "ticker", "direction", "entry_time", "entry_price", "exit_time", "exit_price",
    "shares", "stop_loss", "target", "fees", "account_type",
    "setup", "reason", "emotion_before", "emotion_after", "followed_plan",
    "mistakes", "notes", "screenshot",
]
NUMERIC_FIELDS = {"entry_price", "exit_price", "shares", "stop_loss", "target", "fees"}

DEFAULT_SETTINGS = {
    "account_size": 25000,
    "max_risk_pct": 1.0,
    "daily_loss_limit": 500,
    "max_trades_per_day": 4,
    "min_rr": 2.0,
    "revenge_minutes": 15,
    "no_trade_windows": [
        {"start": "09:30", "end": "09:45", "label": "Opening chaos"},
        {"start": "11:30", "end": "13:30", "label": "Lunch chop"},
    ],
    "setups": ["Breakout", "Pullback", "Gap and Go", "VWAP Reclaim", "Reversal", "Momentum", "Other"],
    "emotions": ["Calm", "Confident", "Excited", "FOMO", "Anxious", "Frustrated", "Bored", "Greedy"],
    "mistake_tags": ["Chased entry", "Moved stop", "No plan", "Exited too early", "Oversized", "Ignored rules"],
}

_lock = threading.Lock()


def connect():
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    return conn


def init():
    with _lock, connect() as conn:
        conn.execute("""
            CREATE TABLE IF NOT EXISTS trades (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                ticker TEXT NOT NULL,
                direction TEXT NOT NULL CHECK (direction IN ('long','short')),
                entry_time TEXT NOT NULL,
                entry_price REAL NOT NULL,
                exit_time TEXT,
                exit_price REAL,
                shares REAL NOT NULL,
                stop_loss REAL,
                target REAL,
                fees REAL DEFAULT 0,
                account_type TEXT NOT NULL DEFAULT 'paper' CHECK (account_type IN ('paper','real')),
                setup TEXT,
                reason TEXT,
                emotion_before TEXT,
                emotion_after TEXT,
                followed_plan INTEGER,
                mistakes TEXT,
                notes TEXT,
                screenshot TEXT,
                created_at TEXT DEFAULT CURRENT_TIMESTAMP
            )""")
        conn.execute("CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)")


# ---------- settings ----------

def get_settings():
    with connect() as conn:
        rows = conn.execute("SELECT key, value FROM settings").fetchall()
    settings = json.loads(json.dumps(DEFAULT_SETTINGS))
    for r in rows:
        if r["key"] in settings:
            settings[r["key"]] = json.loads(r["value"])
    return settings


def save_settings(new):
    clean = {}
    for key, default in DEFAULT_SETTINGS.items():
        if key not in new:
            continue
        val = new[key]
        if isinstance(default, (int, float)):
            val = float(val)
            if val < 0:
                raise ValueError(f"{key} can't be negative")
        elif isinstance(default, list) and not isinstance(val, list):
            raise ValueError(f"{key} must be a list")
        clean[key] = val
    with _lock, connect() as conn:
        for k, v in clean.items():
            conn.execute("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", (k, json.dumps(v)))
    return get_settings()


# ---------- trades ----------

def _num(v):
    if v is None or v == "":
        return None
    return float(v)


def clean_trade(data):
    """Validate one trade dict and return only known, normalised fields."""
    t = {}
    for f in TRADE_FIELDS:
        v = data.get(f)
        if isinstance(v, str):
            v = v.strip()
        if f in NUMERIC_FIELDS:
            try:
                v = _num(v)
            except (TypeError, ValueError):
                raise ValueError(f"{f.replace('_', ' ')} must be a number")
        elif v == "":
            v = None
        t[f] = v
    if not t["ticker"]:
        raise ValueError("ticker is required")
    t["ticker"] = t["ticker"].upper()
    t["direction"] = (t["direction"] or "").lower()
    if t["direction"] not in ("long", "short"):
        raise ValueError("direction must be long or short")
    if not t["entry_time"]:
        raise ValueError("entry date/time is required")
    if t["entry_price"] is None or t["entry_price"] <= 0:
        raise ValueError("entry price must be above 0")
    if t["shares"] is None or t["shares"] <= 0:
        raise ValueError("shares must be above 0")
    if (t["exit_price"] is None) != (t["exit_time"] is None):
        raise ValueError("enter both exit date/time and exit price, or neither (open trade)")
    if t["exit_time"] and t["exit_time"] < t["entry_time"]:
        raise ValueError("exit time is before entry time")
    if t["stop_loss"] is not None and t["stop_loss"] <= 0:
        t["stop_loss"] = None
    if t["target"] is not None and t["target"] <= 0:
        t["target"] = None
    t["fees"] = t["fees"] or 0.0
    t["account_type"] = (t["account_type"] or "paper").lower()
    if t["account_type"] not in ("paper", "real"):
        raise ValueError("account type must be paper or real")
    fp = t["followed_plan"]
    if fp in (True, 1, "1", "yes", "true"):
        t["followed_plan"] = 1
    elif fp in (False, 0, "0", "no", "false"):
        t["followed_plan"] = 0
    else:
        t["followed_plan"] = None
    if isinstance(t["mistakes"], list):
        t["mistakes"] = ", ".join(m for m in t["mistakes"] if m)
    return t


def list_trades():
    with connect() as conn:
        rows = conn.execute("SELECT * FROM trades ORDER BY entry_time, id").fetchall()
    return [dict(r) for r in rows]


def get_trade(trade_id):
    with connect() as conn:
        r = conn.execute("SELECT * FROM trades WHERE id = ?", (trade_id,)).fetchone()
    return dict(r) if r else None


def insert_trades(trades):
    cols = ", ".join(TRADE_FIELDS)
    marks = ", ".join("?" for _ in TRADE_FIELDS)
    ids = []
    with _lock, connect() as conn:
        for t in trades:
            cur = conn.execute(f"INSERT INTO trades ({cols}) VALUES ({marks})", [t[f] for f in TRADE_FIELDS])
            ids.append(cur.lastrowid)
    return ids


def update_trade(trade_id, t):
    sets = ", ".join(f"{f} = ?" for f in TRADE_FIELDS)
    with _lock, connect() as conn:
        conn.execute(f"UPDATE trades SET {sets} WHERE id = ?", [t[f] for f in TRADE_FIELDS] + [trade_id])


def set_screenshot(trade_id, filename):
    with _lock, connect() as conn:
        conn.execute("UPDATE trades SET screenshot = ? WHERE id = ?", (filename, trade_id))


def _remove_screenshot(name):
    if name:
        path = os.path.join(SCREENSHOT_DIR, os.path.basename(name))
        if os.path.exists(path):
            os.remove(path)


def delete_trade(trade_id):
    t = get_trade(trade_id)
    if t:
        _remove_screenshot(t.get("screenshot"))
    with _lock, connect() as conn:
        conn.execute("DELETE FROM trades WHERE id = ?", (trade_id,))


def delete_all_trades(account_type=None):
    for t in list_trades():
        if account_type in (None, t["account_type"]):
            _remove_screenshot(t.get("screenshot"))
    with _lock, connect() as conn:
        if account_type:
            conn.execute("DELETE FROM trades WHERE account_type = ?", (account_type,))
        else:
            conn.execute("DELETE FROM trades")
