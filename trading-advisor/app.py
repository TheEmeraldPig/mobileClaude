"""Day-Trading Advisor: a local web dashboard.

Run:  python app.py      then open http://127.0.0.1:8765

It only listens on your own computer (127.0.0.1), stores everything in the
data/ folder next to this file, and never connects to a broker or places orders.
"""
import base64
import csv
import io
import json
import mimetypes
import os
import re
import sys
import threading
import time
import webbrowser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

import analysis
import db
import importer

HOST = "127.0.0.1"
PORT = int(os.environ.get("ADVISOR_PORT", "8765"))
STATIC_DIR = os.path.join(db.BASE_DIR, "static")
MAX_BODY = 20 * 1024 * 1024  # 20 MB, enough for a screenshot or a big CSV
IMAGE_TYPES = {"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}


class ApiError(Exception):
    def __init__(self, status, message):
        super().__init__(message)
        self.status = status


class Handler(BaseHTTPRequestHandler):
    server_version = "TradingAdvisor/1.0"

    def log_request(self, code="-", size="-"):  # keep the terminal quiet; errors still print
        pass

    # ----- plumbing -----

    def _host_ok(self):
        # Only answer requests addressed to this computer (blocks DNS-rebinding tricks).
        host = (self.headers.get("Host") or "").split(":")[0]
        return host in ("127.0.0.1", "localhost")

    def _send(self, status, body, ctype="application/json; charset=utf-8", extra=None):
        if isinstance(body, (dict, list)):
            body = json.dumps(body, default=_json_default).encode()
        elif isinstance(body, str):
            body = body.encode()
        self.send_response(status)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        for k, v in (extra or {}).items():
            self.send_header(k, v)
        self.end_headers()
        self.wfile.write(body)

    def _json_body(self):
        # Requiring JSON means other websites can't quietly post to this server.
        if not (self.headers.get("Content-Type") or "").startswith("application/json"):
            raise ApiError(415, "expected JSON")
        length = int(self.headers.get("Content-Length") or 0)
        if length > MAX_BODY:
            raise ApiError(413, "upload too large (20 MB max)")
        raw = self.rfile.read(length) if length else b"{}"
        try:
            return json.loads(raw or b"{}")
        except json.JSONDecodeError:
            raise ApiError(400, "invalid JSON")

    def _dispatch(self, method):
        if not self._host_ok():
            return self._send(403, {"error": "forbidden"})
        url = urlparse(self.path)
        try:
            if url.path.startswith("/api/"):
                return self._api(method, url.path, parse_qs(url.query))
            if method != "GET":
                raise ApiError(405, "method not allowed")
            return self._static(url.path)
        except ApiError as e:
            self._send(e.status, {"error": str(e)})
        except ValueError as e:
            self._send(400, {"error": str(e)})
        except Exception as e:  # pragma: no cover - last resort so the page shows something
            import traceback
            traceback.print_exc()
            self._send(500, {"error": f"server error: {e}"})

    def do_GET(self):
        self._dispatch("GET")

    def do_POST(self):
        self._dispatch("POST")

    def do_PUT(self):
        self._dispatch("PUT")

    def do_DELETE(self):
        self._dispatch("DELETE")

    # ----- files -----

    def _static(self, path):
        if path in ("/", "/index.html"):
            return self._file(os.path.join(STATIC_DIR, "index.html"))
        if path.startswith("/static/"):
            return self._file(os.path.join(STATIC_DIR, os.path.basename(path)))
        if path.startswith("/samples/"):
            return self._file(os.path.join(db.BASE_DIR, "samples", os.path.basename(path)),
                              extra={"Content-Disposition": f'attachment; filename="{os.path.basename(path)}"'})
        if path.startswith("/screenshots/"):
            return self._file(os.path.join(db.SCREENSHOT_DIR, os.path.basename(path)))
        raise ApiError(404, "not found")

    def _file(self, full, extra=None):
        if not os.path.isfile(full):
            raise ApiError(404, "not found")
        ctype = mimetypes.guess_type(full)[0] or "application/octet-stream"
        if ctype.startswith("text/") or ctype in ("application/javascript",):
            ctype += "; charset=utf-8"
        with open(full, "rb") as f:
            self._send(200, f.read(), ctype, extra)

    # ----- API -----

    def _api(self, method, path, query):
        q = {k: v[0] for k, v in query.items()}
        settings = db.get_settings()

        if path == "/api/settings":
            if method == "GET":
                return self._send(200, settings)
            if method == "PUT":
                return self._send(200, db.save_settings(self._json_body()))

        if path == "/api/trades":
            if method == "GET":
                return self._send(200, analysis.trade_view(db.list_trades(), settings))
            if method == "POST":
                trade = db.clean_trade(self._json_body())
                new_id = db.insert_trades([trade])[0]
                return self._send(201, {"id": new_id})

        if path == "/api/trades/delete-all" and method == "POST":
            body = self._json_body()
            if body.get("confirm") != "DELETE":
                raise ApiError(400, 'type DELETE to confirm')
            acct = body.get("account_type") if body.get("account_type") in ("paper", "real") else None
            db.delete_all_trades(acct)
            return self._send(200, {"ok": True})

        m = re.fullmatch(r"/api/trades/(\d+)(/screenshot)?", path)
        if m:
            trade_id = int(m.group(1))
            existing = db.get_trade(trade_id)
            if not existing:
                raise ApiError(404, "trade not found")
            if m.group(2):
                return self._screenshot(method, existing)
            if method == "GET":
                return self._send(200, existing)
            if method == "PUT":
                body = self._json_body()
                body["screenshot"] = existing.get("screenshot")
                db.update_trade(trade_id, db.clean_trade(body))
                return self._send(200, {"id": trade_id})
            if method == "DELETE":
                db.delete_trade(trade_id)
                return self._send(200, {"ok": True})

        if path == "/api/report" and method == "GET":
            return self._send(200, analysis.full_report(
                db.list_trades(), settings, q.get("account", "all"), q.get("from") or None, q.get("to") or None))

        if path == "/api/pretrade" and method == "POST":
            plan = self._json_body()
            for f in ("entry", "shares"):
                try:
                    if float(plan.get(f) or 0) <= 0:
                        raise ValueError
                except (TypeError, ValueError):
                    raise ApiError(400, f"{f} must be a number above 0")
            return self._send(200, analysis.pre_trade_check(plan, db.list_trades(), settings))

        if path == "/api/import" and method == "POST":
            body = self._json_body()
            rows = body.get("rows") or []
            if not isinstance(rows, list):
                raise ApiError(400, "rows must be a list")
            trades, errors, warnings = importer.import_rows(rows, body.get("options") or {}, db.list_trades())
            if body.get("commit"):
                db.insert_trades(trades)
            return self._send(200, {"trades": trades, "errors": errors, "warnings": warnings,
                                    "committed": bool(body.get("commit"))})

        if path == "/api/export.csv" and method == "GET":
            out = io.StringIO()
            w = csv.writer(out)
            w.writerow(["id"] + db.TRADE_FIELDS)
            for t in db.list_trades():
                w.writerow([t["id"]] + [t[f] if t[f] is not None else "" for f in db.TRADE_FIELDS])
            return self._send(200, out.getvalue(), "text/csv; charset=utf-8",
                              {"Content-Disposition": 'attachment; filename="my-trades-backup.csv"'})

        raise ApiError(404, "unknown API route")

    def _screenshot(self, method, trade):
        if method == "DELETE":
            db.set_screenshot(trade["id"], None)
            db._remove_screenshot(trade.get("screenshot"))
            return self._send(200, {"ok": True})
        if method != "POST":
            raise ApiError(405, "method not allowed")
        data_url = self._json_body().get("data_url") or ""
        m = re.fullmatch(r"data:(image/[a-z]+);base64,(.+)", data_url, re.S)
        if not m or m.group(1) not in IMAGE_TYPES:
            raise ApiError(400, "screenshot must be a PNG, JPG, GIF or WEBP image")
        try:
            raw = base64.b64decode(m.group(2), validate=True)
        except ValueError:
            raise ApiError(400, "could not read that image")
        name = f"trade{trade['id']}_{int(time.time())}{IMAGE_TYPES[m.group(1)]}"
        with open(os.path.join(db.SCREENSHOT_DIR, name), "wb") as f:
            f.write(raw)
        db._remove_screenshot(trade.get("screenshot"))
        db.set_screenshot(trade["id"], name)
        return self._send(200, {"screenshot": name})


def _json_default(o):
    if hasattr(o, "isoformat"):
        return o.isoformat()
    raise TypeError(type(o))


def main():
    db.init()
    try:
        server = ThreadingHTTPServer((HOST, PORT), Handler)
    except OSError:
        print(f"\nPort {PORT} is busy. Is the advisor already running? Try http://{HOST}:{PORT}\n")
        sys.exit(1)
    url = f"http://{HOST}:{PORT}"
    print("=" * 60)
    print("  Day-Trading Advisor is running (analysis only, no broker).")
    print(f"  Open this in your browser:  {url}")
    print(f"  Your data is saved in:      {db.DATA_DIR}")
    print("  To stop it: click this window and press Ctrl+C")
    print("=" * 60)
    if "--no-browser" not in sys.argv:
        threading.Timer(1.0, lambda: webbrowser.open(url)).start()
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nStopped. Your data is saved.")


if __name__ == "__main__":
    main()
