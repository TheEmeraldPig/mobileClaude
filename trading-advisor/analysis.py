"""All the math: P&L, stats, breakdowns, insights, mistake flags and the pre-trade check.

Times are treated as US market time (Eastern) wall-clock times, exactly as entered.
This module only reads data and gives advice. It never places or sends anything.
"""
from collections import OrderedDict
from datetime import datetime, timedelta
from statistics import median

MIN_SAMPLE = 5  # a group needs at least this many trades before we draw conclusions from it

FLAG_INFO = OrderedDict([
    ("over_risk", ("Risked more than your max", "The distance to your stop times your shares was more than your max risk per trade.")),
    ("no_stop", ("No stop-loss", "No stop-loss was recorded, so the trade's risk was unlimited.")),
    ("bad_stop", ("Stop on the wrong side", "For a long the stop must be below entry; for a short it must be above.")),
    ("low_rr", ("Reward/risk below your minimum", "The planned target didn't pay enough compared with the risk to the stop.")),
    ("revenge", ("Revenge trade", "Opened soon after a losing trade closed.")),
    ("overtrading", ("Overtrading", "This trade was over your max trades per day.")),
    ("after_loss_limit", ("Traded after hitting daily loss limit", "You were already at or past your daily loss limit when you opened this trade.")),
    ("no_trade_window", ("Traded in a no-trade window", "Opened during one of the time windows you said to avoid.")),
    ("held_loser", ("Held a loser too long", "This losing trade was held much longer than your typical winner.")),
])

DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"]

TIME_BUCKETS = [  # (start minute of day, end minute, label, phrase used in sentences)
    (0, 570, "Pre-market", "pre-market trades (before 9:30am)"),
    (570, 585, "9:30–9:45", "trades between 9:30 and 9:45am"),
    (585, 600, "9:45–10:00", "trades between 9:45 and 10:00am"),
    (600, 630, "10:00–10:30", "trades between 10:00 and 10:30am"),
    (630, 660, "10:30–11:00", "trades between 10:30 and 11:00am"),
    (660, 720, "11:00–12:00", "trades between 11am and noon"),
    (720, 780, "12:00–1:00", "trades between noon and 1pm"),
    (780, 840, "1:00–2:00", "trades between 1 and 2pm"),
    (840, 900, "2:00–3:00", "trades between 2 and 3pm"),
    (900, 960, "3:00–4:00", "trades in the last hour (3 to 4pm)"),
    (960, 1440, "After hours", "after-hours trades (4pm or later)"),
]

HOLD_BUCKETS = [  # (max minutes, label)
    (5, "Under 5 min"),
    (15, "5–15 min"),
    (60, "15–60 min"),
    (240, "1–4 hours"),
    (float("inf"), "Over 4 hours"),
]


# ---------- small helpers ----------

def parse_dt(s):
    if not s:
        return None
    if isinstance(s, datetime):
        return s
    return datetime.fromisoformat(str(s).strip().replace(" ", "T"))


def money(x):
    sign = "-" if x < 0 else "+"
    return f"{sign}${abs(x):,.2f}"


def plain_money(x):
    return f"${abs(x):,.2f}"


def pct(x):
    return f"{round(x * 100)}%"


def minutes_label(m):
    if m < 90:
        return f"{round(m)} min"
    if m < 60 * 24:
        return f"{m / 60:.1f} hours"
    return f"{m / 1440:.1f} days"


def time_bucket(dt):
    mins = dt.hour * 60 + dt.minute
    for start, end, label, phrase in TIME_BUCKETS:
        if start <= mins < end:
            return label, phrase
    return TIME_BUCKETS[-1][2], TIME_BUCKETS[-1][3]


def hold_bucket(t):
    if t["entry_dt"].date() != t["exit_dt"].date():
        return "Overnight or longer"
    for limit, label in HOLD_BUCKETS:
        if t["hold_minutes"] < limit:
            return label
    return HOLD_BUCKETS[-1][1]


def in_window(dt, windows):
    hm = dt.strftime("%H:%M")
    for w in windows or []:
        start, end = w.get("start", ""), w.get("end", "")
        if start and end and start <= hm < end:
            return w
    return None


# ---------- per-trade numbers ----------

def risk_numbers(direction, entry, stop, target, shares, settings):
    """Dollar risk, % of account and reward/risk for a planned or actual trade."""
    out = {"risk_per_share": None, "risk_dollars": None, "risk_pct": None,
           "reward_dollars": None, "rr": None, "stop_ok": None, "target_ok": None}
    sign = 1 if direction == "long" else -1
    if stop:
        per_share = (entry - stop) * sign  # positive when the stop is on the correct side
        out["stop_ok"] = per_share > 0
        if per_share > 0:
            out["risk_per_share"] = per_share
            out["risk_dollars"] = per_share * shares
            acct = float(settings.get("account_size") or 0)
            out["risk_pct"] = out["risk_dollars"] / acct * 100 if acct else None
    if target:
        reward = (target - entry) * sign
        out["target_ok"] = reward > 0
        if reward > 0:
            out["reward_dollars"] = reward * shares
            if out["risk_per_share"]:
                out["rr"] = reward / out["risk_per_share"]
    return out


def enrich(trade, settings):
    t = dict(trade)
    t["entry_dt"] = parse_dt(t["entry_time"])
    t["exit_dt"] = parse_dt(t.get("exit_time"))
    t["closed"] = t["exit_dt"] is not None and t.get("exit_price") is not None
    sign = 1 if t["direction"] == "long" else -1
    t.update(risk_numbers(t["direction"], t["entry_price"], t.get("stop_loss"), t.get("target"), t["shares"], settings))
    fees = t.get("fees") or 0
    if t["closed"]:
        t["gross_pnl"] = (t["exit_price"] - t["entry_price"]) * sign * t["shares"]
        t["pnl"] = t["gross_pnl"] - fees
        t["hold_minutes"] = (t["exit_dt"] - t["entry_dt"]).total_seconds() / 60
        t["r_multiple"] = t["pnl"] / t["risk_dollars"] if t.get("risk_dollars") else None
        t["outcome"] = "win" if t["pnl"] > 0 else "loss" if t["pnl"] < 0 else "breakeven"
    else:
        t["gross_pnl"] = t["pnl"] = t["hold_minutes"] = t["r_multiple"] = None
        t["outcome"] = "open"
    t["flags"] = []
    return t


# ---------- mistake flags ----------

def apply_flags(trades, settings):
    """Add t['flags'] to every enriched trade. Paper and real accounts are judged separately."""
    for acct in ("paper", "real"):
        group = sorted((t for t in trades if t["account_type"] == acct), key=lambda t: (t["entry_dt"], t["id"] or 0))
        _flag_group(group, settings)
    return trades


def _flag_group(trades, settings):
    max_risk = float(settings["account_size"]) * float(settings["max_risk_pct"]) / 100
    loss_limit = float(settings["daily_loss_limit"])
    max_per_day = int(settings["max_trades_per_day"])
    min_rr = float(settings["min_rr"])
    revenge = timedelta(minutes=float(settings.get("revenge_minutes", 15)))
    windows = settings.get("no_trade_windows", [])

    winner_holds = [t["hold_minutes"] for t in trades if t["outcome"] == "win"]
    typical_win_hold = median(winner_holds) if len(winner_holds) >= 3 else None

    per_day = {}
    for t in trades:
        flags = t["flags"]
        if not t.get("stop_loss"):
            flags.append("no_stop")
        elif not t["stop_ok"]:
            flags.append("bad_stop")
        if t.get("risk_dollars") and t["risk_dollars"] > max_risk + 0.005:
            flags.append("over_risk")
        if t.get("rr") is not None and min_rr and t["rr"] < min_rr - 1e-9:
            flags.append("low_rr")

        day = t["entry_dt"].date()
        per_day.setdefault(day, []).append(t)
        earlier = [p for p in per_day[day] if p is not t and p["closed"] and p["exit_dt"] <= t["entry_dt"]]
        if any(p["pnl"] < 0 and t["entry_dt"] - p["exit_dt"] <= revenge for p in earlier):
            flags.append("revenge")
        if len(per_day[day]) > max_per_day > 0:
            flags.append("overtrading")
        if loss_limit > 0 and earlier and sum(p["pnl"] for p in earlier) <= -loss_limit:
            flags.append("after_loss_limit")
        if in_window(t["entry_dt"], windows):
            flags.append("no_trade_window")
        if (typical_win_hold is not None and t["outcome"] == "loss"
                and t["hold_minutes"] > typical_win_hold * 1.5):
            flags.append("held_loser")


def journal_mistakes(t):
    return [m.strip() for m in (t.get("mistakes") or "").split(",") if m.strip()]


# ---------- stats ----------

def summary(closed, settings):
    wins = [t["pnl"] for t in closed if t["pnl"] > 0]
    losses = [t["pnl"] for t in closed if t["pnl"] < 0]
    n = len(closed)
    gross_win, gross_loss = sum(wins), -sum(losses)
    rs = [t["r_multiple"] for t in closed if t["r_multiple"] is not None]
    s = {
        "trades": n,
        "wins": len(wins),
        "losses": len(losses),
        "breakeven": n - len(wins) - len(losses),
        "total_pnl": sum(t["pnl"] for t in closed),
        "win_rate": len(wins) / n if n else None,
        "avg_win": gross_win / len(wins) if wins else None,
        "avg_loss": gross_loss / len(losses) if losses else None,
        "expectancy": sum(t["pnl"] for t in closed) / n if n else None,
        "profit_factor": gross_win / gross_loss if gross_loss else None,  # None = no losing trades yet
        "largest_win": max(wins) if wins else None,
        "largest_loss": min(losses) if losses else None,
        "avg_r": sum(rs) / len(rs) if rs else None,
        "total_fees": sum(t.get("fees") or 0 for t in closed),
        "gross_win": gross_win,
        "gross_loss": gross_loss,
    }
    s.update(equity_and_drawdown(closed, float(settings["account_size"])))
    s.update(streaks(closed))
    return s


def equity_and_drawdown(closed, start):
    ordered = sorted(closed, key=lambda t: (t["exit_dt"], t["id"] or 0))
    equity, peak = start, start
    peak_time = ordered[0]["exit_time"] if ordered else None
    curve = [{"time": None, "equity": start, "pnl": 0, "label": "Start"}]
    dd = {"max_drawdown": 0.0, "max_drawdown_pct": 0.0, "drawdown_from": None, "drawdown_to": None}
    for t in ordered:
        equity += t["pnl"]
        curve.append({"time": t["exit_time"], "equity": round(equity, 2), "pnl": round(t["pnl"], 2),
                      "label": f'{t["ticker"]} {t["direction"]}', "id": t["id"]})
        if equity > peak:
            peak, peak_time = equity, t["exit_time"]
        drop = peak - equity
        if drop > dd["max_drawdown"]:
            dd = {"max_drawdown": drop, "max_drawdown_pct": drop / peak * 100 if peak > 0 else None,
                  "drawdown_from": peak_time, "drawdown_to": t["exit_time"]}
    return {"equity_curve": curve, **dd}


def streaks(closed):
    ordered = sorted(closed, key=lambda t: (t["exit_dt"], t["id"] or 0))
    best = {"win": 0, "loss": 0}
    cur_kind, cur_len = None, 0
    for t in ordered:
        kind = t["outcome"] if t["outcome"] in ("win", "loss") else None
        if kind and kind == cur_kind:
            cur_len += 1
        else:
            cur_kind, cur_len = kind, (1 if kind else 0)
        if kind:
            best[kind] = max(best[kind], cur_len)
    return {"longest_win_streak": best["win"], "longest_loss_streak": best["loss"],
            "current_streak": {"kind": cur_kind, "length": cur_len}}


def group_stats(label, trades, phrase=None):
    n = len(trades)
    wins = sum(1 for t in trades if t["pnl"] > 0)
    losses = sum(1 for t in trades if t["pnl"] < 0)
    total = sum(t["pnl"] for t in trades)
    return {"label": label, "phrase": phrase or label, "count": n, "wins": wins, "losses": losses,
            "win_rate": wins / n if n else None, "loss_rate": losses / n if n else None,
            "total_pnl": total, "avg_pnl": total / n if n else None}


def _group(closed, keyfunc, order=None, sort_by_count=False):
    groups, phrases = OrderedDict(), {}
    for t in closed:
        key, phrase = keyfunc(t)
        groups.setdefault(key, []).append(t)
        phrases[key] = phrase
    keys = list(groups)
    if order:
        keys = [k for k in order if k in groups] + [k for k in keys if k not in order]
    elif sort_by_count:
        keys.sort(key=lambda k: (-len(groups[k]), k))
    return [group_stats(k, groups[k], phrases[k]) for k in keys]


def breakdowns(closed):
    plan_labels = {1: "Followed plan", 0: "Broke plan", None: "Not recorded"}
    return OrderedDict([
        ("time_of_day", {"title": "Time of day (entry)", "rows": _group(
            closed, lambda t: time_bucket(t["entry_dt"]), order=[b[2] for b in TIME_BUCKETS])}),
        ("day_of_week", {"title": "Day of week", "rows": _group(
            closed, lambda t: (DAYS[t["entry_dt"].weekday()], f'trades on {DAYS[t["entry_dt"].weekday()]}s'), order=DAYS)}),
        ("ticker", {"title": "Ticker", "rows": _group(
            closed, lambda t: (t["ticker"], f'trades in {t["ticker"]}'), sort_by_count=True)}),
        ("setup", {"title": "Setup type", "rows": _group(
            closed, lambda t: (t.get("setup") or "(no setup)", f'"{t.get("setup")}" trades' if t.get("setup") else "trades with no setup recorded"),
            sort_by_count=True)}),
        ("hold_time", {"title": "Hold time", "rows": _group(
            closed, lambda t: (hold_bucket(t), f"trades held {hold_bucket(t).lower()}"),
            order=[b[1] for b in HOLD_BUCKETS] + ["Overnight or longer"])}),
        ("plan", {"title": "Followed my plan?", "rows": _group(
            closed, lambda t: (plan_labels[t.get("followed_plan")], {1: "trades where you followed your plan", 0: "trades where you broke your plan", None: "trades with no plan answer"}[t.get("followed_plan")]),
            order=list(plan_labels.values()))}),
        ("emotion", {"title": "Emotion before entry", "rows": _group(
            closed, lambda t: (t.get("emotion_before") or "(not recorded)", f'trades when you felt "{t.get("emotion_before")}"' if t.get("emotion_before") else "trades with no emotion recorded"),
            sort_by_count=True)}),
    ])


# ---------- mistake costs ----------

def mistake_costs(closed):
    """Which mistakes cost the most. Includes automatic flags and the tags you journaled."""
    buckets = OrderedDict()
    for t in closed:
        keys = [("flag", f) for f in t["flags"]] + [("tag", m) for m in journal_mistakes(t)]
        for kind, name in keys:
            label = FLAG_INFO[name][0] if kind == "flag" else name
            b = buckets.setdefault((kind, name), {"key": name, "source": "automatic" if kind == "flag" else "journal",
                                                  "label": label, "explain": FLAG_INFO[name][1] if kind == "flag" else "You tagged this in your journal.",
                                                  "count": 0, "total_pnl": 0.0, "money_lost": 0.0, "trade_ids": []})
            b["count"] += 1
            b["total_pnl"] += t["pnl"]
            b["money_lost"] += min(t["pnl"], 0)
            b["trade_ids"].append(t["id"])
    rows = sorted(buckets.values(), key=lambda b: b["total_pnl"])
    for r in rows:
        r["avg_pnl"] = r["total_pnl"] / r["count"]
    clean = [t for t in closed if not t["flags"] and not journal_mistakes(t)]
    return {"rows": rows, "clean": group_stats("Trades with no mistakes", clean) if clean else None}


# ---------- plain-English insights ----------

def insights(closed, s, bd, costs):
    out = []

    def add(tone, text, impact=0.0):
        out.append({"tone": tone, "text": text, "impact": abs(impact)})

    n = len(closed)
    if n == 0:
        return [{"tone": "info", "text": "No closed trades yet. Add a trade or import a CSV to see insights.", "impact": 0}]
    if n < 30:
        add("info", f"You have {n} closed trades. Patterns get more trustworthy after about 30 or more trades, so treat these as early hints.", 1e9)

    if s["expectancy"] is not None:
        if s["expectancy"] < 0:
            add("bad", f"On average each trade loses you {plain_money(s['expectancy'])} after fees. Until that's positive, trade small or paper trade.", s["total_pnl"])
        else:
            add("good", f"On average each trade makes you {plain_money(s['expectancy'])} after fees. Your edge is positive so far.", s["total_pnl"])

    if s["avg_win"] and s["avg_loss"] and s["avg_loss"] > s["avg_win"] * 1.1:
        need = s["avg_loss"] / (s["avg_win"] + s["avg_loss"])
        add("bad", f"Your average loss ({plain_money(s['avg_loss'])}) is bigger than your average win ({plain_money(s['avg_win'])}). "
                   f"At that size you need to win more than {pct(need)} of trades just to break even; you win {pct(s['win_rate'])}.",
            (s["avg_loss"] - s["avg_win"]) * s["losses"])

    for key in ("time_of_day", "day_of_week", "ticker", "setup", "hold_time", "emotion"):
        for g in bd[key]["rows"]:
            if g["count"] < MIN_SAMPLE or g["label"].startswith("("):
                continue
            detail = f"({g['losses'] if g['total_pnl'] < 0 else g['wins']} of {g['count']} trades, {money(g['total_pnl'])} total)"
            phrase = g["phrase"][0].upper() + g["phrase"][1:]
            if g["loss_rate"] >= 0.6 and g["total_pnl"] < 0:
                add("bad", f"Your {g['phrase']} lose {pct(g['loss_rate'])} of the time {detail}.", g["total_pnl"])
            elif g["win_rate"] >= 0.6 and g["total_pnl"] > 0:
                add("good", f"Your {g['phrase']} win {pct(g['win_rate'])} of the time {detail}. Look for more of these.", g["total_pnl"])
            elif g["total_pnl"] < 0 and s["gross_loss"] and -g["total_pnl"] >= 0.15 * s["gross_loss"]:
                add("bad", f"{phrase} have cost you {plain_money(g['total_pnl'])} overall (average {money(g['avg_pnl'])} per trade, {g['count']} trades).", g["total_pnl"])

    plan = {g["label"]: g for g in bd["plan"]["rows"]}
    yes, no = plan.get("Followed plan"), plan.get("Broke plan")
    if yes and no and yes["count"] >= 3 and no["count"] >= 3:
        add("bad" if no["avg_pnl"] < yes["avg_pnl"] else "info",
            f"When you follow your plan you average {money(yes['avg_pnl'])} per trade ({yes['count']} trades). "
            f"When you break it you average {money(no['avg_pnl'])} ({no['count']} trades).",
            no["total_pnl"] - no["count"] * yes["avg_pnl"])

    win_h = [t["hold_minutes"] for t in closed if t["outcome"] == "win"]
    loss_h = [t["hold_minutes"] for t in closed if t["outcome"] == "loss"]
    if len(win_h) >= 3 and len(loss_h) >= 3:
        mw, ml = median(win_h), median(loss_h)
        if mw > 0 and ml > mw * 1.3:
            add("bad", f"You hold losing trades {ml / mw:.1f}× longer than winners (typical loser {minutes_label(ml)}, typical winner {minutes_label(mw)}). "
                       "Losers rarely come back. Respect your stop.", sum(t["pnl"] for t in closed if t["outcome"] == "loss"))
        elif ml > 0 and mw > ml * 1.3:
            add("good", f"You let winners run longer than losers (typical winner {minutes_label(mw)}, typical loser {minutes_label(ml)}). Keep doing that.", 0)

    bad_costs = [r for r in costs["rows"] if r["total_pnl"] < 0]
    if bad_costs:
        top = bad_costs[0]
        add("bad", f'Your most expensive mistake is "{top["label"]}": {top["count"]} trades that lost {plain_money(top["total_pnl"])} in total.', top["total_pnl"])
    if costs["clean"] and costs["clean"]["count"] >= 3:
        c = costs["clean"]
        add("good" if c["avg_pnl"] > 0 else "info",
            f"Trades with no mistakes flagged average {money(c['avg_pnl'])} each ({c['count']} trades). That's your real edge when you trade clean.", c["total_pnl"])

    if s["longest_loss_streak"] >= 4:
        add("info", f"Your longest losing streak was {s['longest_loss_streak']} trades in a row. Consider a rule to stop for the day after 2 or 3 losses in a row.", 0)
    if s["total_fees"] > 0 and s["gross_win"] > 0 and s["total_fees"] / s["gross_win"] >= 0.15:
        add("bad", f"Fees ate {pct(s['total_fees'] / s['gross_win'])} of your winning trades' profits ({plain_money(s['total_fees'])}). Fewer, better trades would keep more.", s["total_fees"])

    # Biggest problems first, but always leave room for what's working.
    pick = lambda tone, n: sorted((i for i in out if i["tone"] == tone), key=lambda i: -i["impact"])[:n]
    infos = pick("info", 3)
    return [i for i in infos if i["impact"] == 1e9] + pick("bad", 7) + pick("good", 4) + [i for i in infos if i["impact"] != 1e9]


# ---------- top-level ----------

def filter_trades(trades, account="all", date_from=None, date_to=None):
    res = []
    for t in trades:
        if account in ("paper", "real") and t["account_type"] != account:
            continue
        d = t["entry_time"][:10]
        if date_from and d < date_from:
            continue
        if date_to and d > date_to:
            continue
        res.append(t)
    return res


def prepare(trades, settings):
    enriched = [enrich(t, settings) for t in trades]
    apply_flags(enriched, settings)
    return enriched


def full_report(trades, settings, account="all", date_from=None, date_to=None):
    enriched = filter_trades(prepare(trades, settings), account, date_from, date_to)
    closed = [t for t in enriched if t["closed"]]
    s = summary(closed, settings)
    bd = breakdowns(closed)
    costs = mistake_costs(closed)
    return {
        "summary": s,
        "breakdowns": bd,
        "mistakes": costs,
        "insights": insights(closed, s, bd, costs),
        "open_trades": sum(1 for t in enriched if not t["closed"]),
        "flag_info": {k: {"label": v[0], "explain": v[1]} for k, v in FLAG_INFO.items()},
    }


def trade_view(trades, settings):
    """Trades plus their computed numbers and flags, for the Trades table."""
    keep = ("pnl", "r_multiple", "hold_minutes", "risk_dollars", "risk_pct", "rr", "outcome", "flags")
    return [{**{k: v for k, v in t.items() if not k.endswith("_dt")}, **{k: t[k] for k in keep}}
            for t in prepare(trades, settings)]


# ---------- pre-trade check ----------

def pre_trade_check(plan, trades, settings, now=None):
    """Advice for a planned trade. Returns numbers, green/yellow/red checks and history."""
    direction = (plan.get("direction") or "long").lower()
    entry = float(plan["entry"])
    shares = float(plan["shares"])
    stop = float(plan["stop"]) if plan.get("stop") not in (None, "") else None
    target = float(plan["target"]) if plan.get("target") not in (None, "") else None
    when = parse_dt(plan.get("time")) or now or datetime.now()
    account_type = plan.get("account_type") or "real"
    setup = plan.get("setup") or None
    ticker = (plan.get("ticker") or "").upper()

    acct = float(settings["account_size"])
    max_risk_pct = float(settings["max_risk_pct"])
    max_risk = acct * max_risk_pct / 100
    min_rr = float(settings["min_rr"])
    r = risk_numbers(direction, entry, stop, target, shares, settings)
    position_value = entry * shares

    checks = []

    def check(level, title, detail):
        checks.append({"level": level, "title": title, "detail": detail})

    # Stop-loss and risk
    if stop is None:
        check("red", "No stop-loss", "Decide where you're wrong before you get in. Without a stop your risk has no limit.")
    elif not r["stop_ok"]:
        check("red", "Stop is on the wrong side", f"For a {direction} the stop must be {'below' if direction == 'long' else 'above'} your entry.")
    else:
        if r["risk_dollars"] > max_risk + 0.005:
            check("red", "Risk is over your max", f"You'd risk {plain_money(r['risk_dollars'])} ({r['risk_pct']:.2f}% of your account). Your max is {plain_money(max_risk)} ({max_risk_pct:g}%).")
        elif r["risk_dollars"] > max_risk * 0.8:
            check("yellow", "Risk is close to your max", f"{plain_money(r['risk_dollars'])} ({r['risk_pct']:.2f}%) of a {plain_money(max_risk)} max.")
        else:
            check("green", "Risk is within your max", f"{plain_money(r['risk_dollars'])} ({r['risk_pct']:.2f}% of your account).")

    # Reward / risk
    if target is None:
        check("yellow", "No target", "Without a target you can't judge if the trade is worth the risk.")
    elif not r["target_ok"]:
        check("red", "Target is on the wrong side", f"For a {direction} the target must be {'above' if direction == 'long' else 'below'} your entry.")
    elif r["rr"] is not None:
        if r["rr"] < min_rr - 1e-9:
            check("red", "Reward/risk below your minimum", f"{r['rr']:.2f} : 1, but your rule is at least {min_rr:g} : 1.")
        else:
            check("green", "Reward/risk meets your rule", f"{r['rr']:.2f} : 1 (minimum {min_rr:g} : 1).")

    if position_value > acct:
        check("yellow", "Position is bigger than your account", f"{plain_money(position_value)} of stock on a {plain_money(acct)} account means you'd be using margin (borrowed money).")

    # Time rules
    w = in_window(when, settings.get("no_trade_windows"))
    if w:
        check("red", "Inside a no-trade window", f"{w['start']}–{w['end']} {('(' + w['label'] + ')') if w.get('label') else ''}".strip())
    mins = when.hour * 60 + when.minute
    if mins < 570 or mins >= 960:
        check("yellow", "Outside regular market hours", "Pre-market and after-hours have thin volume and wide spreads. Be extra careful.")

    # Today's state, from the trades you've logged in this account
    enriched = [t for t in prepare(trades, settings) if t["account_type"] == account_type]
    today = [t for t in enriched if t["entry_dt"].date() == when.date() and t["entry_dt"] <= when]
    done_today = [t for t in today if t["closed"] and t["exit_dt"] <= when]
    day_pnl = sum(t["pnl"] for t in done_today)
    max_trades = int(settings["max_trades_per_day"])
    loss_limit = float(settings["daily_loss_limit"])
    if max_trades:
        if len(today) >= max_trades:
            check("red", "Over your max trades for today", f"You've already taken {len(today)} {account_type} trades today (max {max_trades}).")
        elif len(today) == max_trades - 1:
            check("yellow", "Last trade allowed today", f"This would be trade {len(today) + 1} of {max_trades}.")
    if loss_limit:
        if day_pnl <= -loss_limit:
            check("red", "Daily loss limit already hit", f"You're at {money(day_pnl)} today. Your limit is {plain_money(loss_limit)}. Stop for the day.")
        elif r["risk_dollars"] and day_pnl - r["risk_dollars"] < -loss_limit:
            check("yellow", "A loss here would break your daily limit", f"You're at {money(day_pnl)} today. Losing {plain_money(r['risk_dollars'])} would put you past -{plain_money(loss_limit)}.")
    revenge = timedelta(minutes=float(settings.get("revenge_minutes", 15)))
    recent_loss = [t for t in done_today if t["pnl"] < 0 and when - t["exit_dt"] <= revenge]
    if recent_loss:
        last = max(recent_loss, key=lambda t: t["exit_dt"])
        check("yellow", "Possible revenge trade", f"Your {last['ticker']} trade lost {plain_money(last['pnl'])} at {last['exit_dt'].strftime('%H:%M')}. Take a breath and make sure this is a real setup.")

    # How similar past trades went (closed trades in this account)
    closed = [t for t in enriched if t["closed"]]
    bucket, _ = time_bucket(when)
    history = []

    def hist(title, subset):
        g = group_stats(title, subset)
        if g["count"] == 0:
            level = "gray"
        elif g["count"] < MIN_SAMPLE:
            level = "gray"
        elif g["win_rate"] < 0.4 or g["total_pnl"] < 0:
            level = "yellow"
        else:
            level = "green"
        g["level"] = level
        history.append(g)
        return g

    if setup:
        hs = hist(f'Setup "{setup}"', [t for t in closed if t.get("setup") == setup])
        hist(f'"{setup}" at {bucket}', [t for t in closed if t.get("setup") == setup and time_bucket(t["entry_dt"])[0] == bucket])
    hb = hist(f"Any trade at {bucket}", [t for t in closed if time_bucket(t["entry_dt"])[0] == bucket])
    if ticker:
        hist(f"{ticker} trades", [t for t in closed if t["ticker"] == ticker])
    hist(f"{direction.capitalize()} trades", [t for t in closed if t["direction"] == direction])

    for g in ([hs] if setup else []) + [hb]:
        if g["count"] >= MIN_SAMPLE and g["level"] == "yellow":
            check("yellow", f"Your history is weak here: {g['label']}",
                  f"Won {pct(g['win_rate'])} of {g['count']} trades, {money(g['total_pnl'])} total.")
        elif g["count"] >= MIN_SAMPLE and g["level"] == "green":
            check("green", f"Your history is good here: {g['label']}",
                  f"Won {pct(g['win_rate'])} of {g['count']} trades, {money(g['total_pnl'])} total.")

    suggested = None
    if r["risk_per_share"]:
        suggested = int(max_risk // r["risk_per_share"])

    order = {"red": 0, "yellow": 1, "green": 2}
    checks.sort(key=lambda c: order[c["level"]])
    verdict = "red" if any(c["level"] == "red" for c in checks) else "yellow" if any(c["level"] == "yellow" for c in checks) else "green"
    return {
        "numbers": {**r, "position_value": position_value, "position_pct": position_value / acct * 100 if acct else None,
                    "max_risk_dollars": max_risk, "suggested_max_shares": suggested,
                    "trades_today": len(today), "pnl_today": day_pnl},
        "checks": checks,
        "history": history,
        "verdict": verdict,
        "time_bucket": bucket,
    }
