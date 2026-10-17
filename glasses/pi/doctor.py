#!/usr/bin/env python3
"""Lens doctor: when the hub keeps crashing, Claude Code reads the logs and fixes it.

systemd runs this when lens.service gives up after repeated crashes. It:
  1. saves the current code in git (so any fix can be undone),
  2. asks Claude Code to find the cause in the logs and fix it, only inside this folder,
  3. keeps the fix if `lens.py --selftest` passes, otherwise rolls it back,
  4. writes what happened to <data_dir>/doctor-report.md and restarts the hub.

Fixes are committed on the local branch `pi-fixes`. Push that branch to GitHub so the
same fix lands in the main code too.

  python3 doctor.py            run once (what systemd does)
  python3 doctor.py --dry-run  show the logs and prompt it would use, change nothing
"""
import argparse
import json
import shutil
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
from lens import expand, load_config  # noqa: E402

MAX_RUNS_PER_DAY = 3

PROMPT = """You are the on-device repair agent for Lens, a smart-glasses hub that runs on a Raspberry Pi in the owner's pocket. The hub service crashed repeatedly and systemd stopped restarting it. Find the cause and fix it so the hub runs again.

Rules:
- Only edit files in this folder (the hub's code and config). Don't install packages, touch other folders, or change system settings.
- Make the smallest fix that gets the hub running. Don't refactor or add features.
- If the cause is outside the code (Claude Code logged out, no internet, a missing microphone, a missing program), don't change code to hide it. Explain what the owner needs to do instead.
- Check your fix with `python3 lens.py --selftest` (add `--config <file>` if the logs show a custom config).
- Finish with a two-line summary: what broke, and what you changed or what the owner must do.
{env_note}
Recent hub log:
{log}

systemd journal for lens.service:
{journal}

Self-test output:
{selftest}
"""


def run(cmd, **kw):
    return subprocess.run(cmd, capture_output=True, text=True, **kw)


def tail(path, lines=150):
    try:
        return "\n".join(Path(path).read_text(errors="replace").splitlines()[-lines:])
    except OSError:
        return "(no log file)"


def git(*args):
    return run(["git", "-C", str(HERE), *args])


def ensure_branch():
    """Puts the code on the pi-fixes branch with everything committed; returns the commit to roll back to."""
    if git("rev-parse", "--is-inside-work-tree").returncode != 0:
        git("init", "-q")
    if git("rev-parse", "--abbrev-ref", "HEAD").stdout.strip() != "pi-fixes":
        git("checkout", "-q", "-B", "pi-fixes")
    git("add", "-A", ".")
    git("-c", "user.name=Lens doctor", "-c", "user.email=lens@localhost", "commit", "-q", "-m", "doctor: snapshot before repair")
    return git("rev-parse", "HEAD").stdout.strip()


def runs_today(state_file):
    try:
        runs = json.loads(state_file.read_text())
    except (OSError, json.JSONDecodeError):
        runs = []
    return [t for t in runs if time.time() - t < 86400]


def restart_hub():
    if shutil.which("systemctl"):
        run(["systemctl", "--user", "reset-failed", "lens.service"])
        run(["systemctl", "--user", "restart", "lens.service"])


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--config", default=str(HERE / "config.toml"))
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--model", default="opus", help="Claude Code model for repairs")
    args = ap.parse_args()

    cfg = load_config(args.config)
    data = expand(cfg["data_dir"])
    data.mkdir(parents=True, exist_ok=True)
    report = data / "doctor-report.md"
    state = data / "doctor-runs.json"

    journal = "(journalctl not available)"
    if shutil.which("journalctl"):
        journal = run(["journalctl", "--user", "-u", "lens.service", "-n", "80", "--no-pager"]).stdout or "(empty)"
    selftest = run([sys.executable, str(HERE / "lens.py"), "--config", args.config, "--selftest"], timeout=120)
    env_note = ""
    if cfg["claude"]["bin"] == "claude-proot":
        env_note = ("- This hub runs on an Android phone in Termux; you run inside its Ubuntu (proot). When you run the\n"
                    "  self-test there, failures about Termux:API commands are expected: the doctor re-runs it in Termux afterwards.\n")
    prompt = PROMPT.format(log=tail(data / "lens.log"), journal=journal, selftest=selftest.stdout + selftest.stderr,
                           env_note=env_note)

    if args.dry_run:
        print(prompt)
        return

    runs = runs_today(state)
    stamp = f"\n## {datetime.now():%Y-%m-%d %H:%M}\n"
    if len(runs) >= MAX_RUNS_PER_DAY:
        with open(report, "a") as f:
            f.write(stamp + f"Skipped: already tried {MAX_RUNS_PER_DAY} repairs today. Needs a human look.\n")
        return
    state.write_text(json.dumps(runs + [time.time()]))

    before = ensure_branch()
    allowed = ["Read", "Grep", "Glob", "Edit",
               "Bash(python3 lens.py --selftest:*)", "Bash(python3 lens.py --config:*)", "Bash(git diff:*)", "Bash(git status:*)"]
    r = run([cfg["claude"]["bin"], "-p", prompt, "--model", args.model, "--permission-mode", "acceptEdits",
             "--tools", "Read,Grep,Glob,Edit,Bash", "--allowedTools", *allowed, "--no-session-persistence"],
            cwd=HERE, timeout=900)
    summary = (r.stdout.strip() or r.stderr.strip() or "(no answer)")[-2000:]

    changed = bool(git("status", "--porcelain").stdout.strip())
    check = run([sys.executable, str(HERE / "lens.py"), "--config", args.config, "--selftest"], timeout=120)
    if changed and check.returncode == 0:
        git("add", "-A", ".")
        git("-c", "user.name=Lens doctor", "-c", "user.email=lens@localhost", "commit", "-q",
            "-m", "doctor: automatic repair", "-m", summary)
        outcome = "Fixed and kept (branch pi-fixes). Push it to GitHub so the fix gets reviewed."
    elif changed:
        git("reset", "-q", "--hard", before)
        outcome = "Tried a fix but the self-test still failed, so it was rolled back."
    else:
        outcome = "No code changed."

    with open(report, "a") as f:
        f.write(stamp + outcome + "\n\n" + summary + "\n")
    print(outcome, "\n", summary, sep="")
    restart_hub()


if __name__ == "__main__":
    main()
