#!/usr/bin/env python3
"""Stand-in glasses for testing: your computer screen becomes the glasses camera.

It behaves like the glasses firmware: it serves /capture (a picture of your screen) and
/status, tells the hub where it is with /hello, and the keyboard is the glasses button.
So the hub, the phone page and Claude can all be tested before the glasses exist.

  pip install mss pillow
  python3 screen_glasses.py --hub http://192.168.1.20:8765 --token YOUR_TOKEN

Keys (type, then Enter):  Enter = button press (talk / stop)   p = long press (photo)
                           s = stop   q = quit
Whatever is on the chosen screen is sent to Claude each time you ask, so keep private
things off it while this runs.
"""
import argparse
import io
import json
import socket
import threading
import time
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import mss
from PIL import Image


def grab(monitor, max_side, quality):
    with getattr(mss, "MSS", mss.mss)() as sct:
        mons = sct.monitors   # [0] is every screen together, [1] the first screen, ...
        shot = sct.grab(mons[monitor if monitor < len(mons) else 1])
    img = Image.frombytes("RGB", shot.size, shot.bgra, "raw", "BGRX")
    img.thumbnail((max_side, max_side))
    out = io.BytesIO()
    img.save(out, "JPEG", quality=quality)
    return out.getvalue()


def local_ip(hub_url):
    """The address the hub can reach us at: the one this computer uses to talk to it."""
    host = urllib.parse.urlsplit(hub_url).hostname
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
        s.connect((host, 9))   # no packet is sent; this only picks the right network
        return s.getsockname()[0]


def post(args, path, body):
    req = urllib.request.Request(args.hub.rstrip("/") + path, data=json.dumps(body).encode(), method="POST",
                                 headers={"Content-Type": "application/json", "X-Lens-Token": args.token})
    with urllib.request.urlopen(req, timeout=5) as r:
        return json.loads(r.read() or b"{}")


def main():
    ap = argparse.ArgumentParser(description="Use your screen as the Lens glasses camera, for testing")
    ap.add_argument("--hub", required=True, help="the hub's address, e.g. http://192.168.1.20:8765")
    ap.add_argument("--token", default="", help="the token from the hub's config.toml")
    ap.add_argument("--port", type=int, default=8801, help="port this program listens on")
    ap.add_argument("--monitor", type=int, default=1, help="1 = first screen, 2 = second, 0 = all screens")
    ap.add_argument("--max-side", type=int, default=1280, help="shrink the picture to at most this many pixels")
    ap.add_argument("--quality", type=int, default=80, help="JPEG quality, 1-95")
    args = ap.parse_args()

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *a):
            pass

        def do_GET(self):
            if self.path.startswith("/capture"):
                data = grab(args.monitor, args.max_side, args.quality)
                print(f"\r[screen sent to the hub, {len(data) // 1024} KB]")
                ctype = "image/jpeg"
            elif self.path.startswith("/status"):
                data, ctype = json.dumps({"ok": True, "screen": True}).encode(), "application/json"
            else:
                self.send_response(404)
                self.end_headers()
                return
            self.send_response(200)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)

    grab(args.monitor, args.max_side, args.quality)   # fail now, not mid-question, if screen capture doesn't work
    server = ThreadingHTTPServer(("0.0.0.0", args.port), Handler)
    server.daemon_threads = True
    threading.Thread(target=server.serve_forever, daemon=True).start()
    me = f"{local_ip(args.hub)}:{args.port}"

    def hello_loop():
        said = False
        while True:
            try:
                post(args, "/hello", {"ip": me})
                if not said:
                    print(f"Connected to the hub. It fetches your screen from http://{me}/capture")
                    said = True
            except Exception as e:
                print(f"\rCan't reach the hub at {args.hub} ({e}). Same Wi-Fi? Right token? Retrying...")
                said = False
            time.sleep(30)

    threading.Thread(target=hello_loop, daemon=True).start()
    print("Keys: Enter = button press (talk / stop), p = photo, s = stop, q = quit")
    keys = {"": "press", "p": "long", "s": "stop"}
    try:
        while True:
            k = input().strip().lower()
            if k == "q":
                break
            if k in keys:
                try:
                    post(args, "/button", {"event": keys[k]})
                    print({"press": "Talk now (the hub is listening)...", "long": "Photo taken.", "stop": "Stopped."}[keys[k]])
                except Exception as e:
                    print(f"The hub didn't take the button press: {e}")
    except (KeyboardInterrupt, EOFError):
        pass


if __name__ == "__main__":
    main()
