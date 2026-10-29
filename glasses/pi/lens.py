#!/usr/bin/env python3
"""Lens hub: the pocket computer for the Lens glasses.

The glasses send button presses and camera pictures over Wi-Fi. The hub listens
through the microphone, writes down what you said (whisper.cpp), asks Claude Code
(kept running, logged into your own account) and speaks the answer (Piper).

  python3 lens.py              run the hub (what the systemd service does)
  python3 lens.py --console    type questions instead of talking; Enter alone = talk
  python3 lens.py --selftest   check that every part is installed and reachable
"""
import argparse
import array
import base64
import json
import socket
import struct
import zlib
import logging
import logging.handlers
import queue
import re
import shutil
import subprocess
import sys
import threading
import time
import tomllib
import urllib.parse
import urllib.request
import wave
from collections import deque
from datetime import datetime
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

HERE = Path(__file__).resolve().parent
log = logging.getLogger("lens")

DEFAULTS = {
    "data_dir": "~/lens-data",
    "claude": {"bin": "claude", "model": "sonnet", "effort": "low", "web_search": True, "timeout": 90,
               "fresh_after_minutes": 10},
    "ears": {"engine": "whisper", "whisper_bin": "~/whisper.cpp/build/bin/whisper-cli",
             "whisper_model": "~/whisper.cpp/models/ggml-base.en.bin", "language": "en",
             "device": "default", "max_seconds": 15, "silence_seconds": 1.2, "level": 500},
    "mouth": {"engine": "piper", "piper_bin": "~/piper/piper",
              "piper_voice": "~/piper/voices/en_US-lessac-medium.onnx", "device": "default"},
    "eyes": {"source": "glasses", "glasses_url": "http://lens-glasses.local/capture", "send_view": True},
    "hub": {"port": 8765, "token": ""},
}

SYSTEM_PROMPT = """You are Lens, an assistant that lives in a pair of smart glasses. The user is wearing you and talking out loud while going about their day; your answers are read aloud through a small speaker or earbuds.

- When an image is attached it is exactly what the user is looking at right now through the glasses camera. Refer to it naturally ("that sign says...", "the plant on the left..."). Don't mention "the image" or "the photo".
- Answer in one to three short spoken sentences unless the user asks for more. Plain speech only: no markdown, lists, headings or emoji.
- If you can't tell from the view, say what you'd need ("get a bit closer to the label").
- Use web search for anything current or local (opening hours, prices, news, weather).
- When the user asks you to remember something, put the note at the very end of your answer as [[note: short self-contained note]]. The hub saves it and won't read it aloud. Saved notes are listed in each message; use them to answer questions like "where did I park"."""


def load_config(path):
    cfg = json.loads(json.dumps(DEFAULTS))
    if path and Path(path).exists():
        with open(path, "rb") as f:
            for section, values in tomllib.load(f).items():
                if isinstance(values, dict) and isinstance(cfg.get(section), dict):
                    cfg[section].update(values)
                else:
                    cfg[section] = values
    return cfg


def expand(p):
    return Path(p).expanduser()


NOTE_RE = re.compile(r"\[\[note:\s*(.*?)\]\]", re.S | re.I)


class Brain:
    """One Claude Code process kept running, so each question skips the start-up time."""

    def __init__(self, cfg):
        self.cfg = cfg["claude"]
        self.proc = None
        self.events = queue.Queue()
        self.lock = threading.Lock()
        self.cancelled = threading.Event()
        self.last_error = ""
        self.turns = 0
        self.convo_bytes = 0   # size of the conversation so far; every request uploads all of it

    def command(self):
        c = self.cfg
        cmd = [c["bin"], "-p", "--input-format", "stream-json", "--output-format", "stream-json",
               "--verbose", "--include-partial-messages", "--no-session-persistence",
               "--model", c["model"], "--effort", c["effort"], "--system-prompt", SYSTEM_PROMPT]
        tools = ["WebSearch"] if c["web_search"] else []
        cmd += ["--tools", ",".join(tools) if tools else ""]
        if tools:
            cmd += ["--allowedTools", ",".join(tools)]
        return cmd

    def start(self):
        self.stop()
        self.events = queue.Queue()
        self.turns = 0
        self.convo_bytes = 0
        log.info("starting Claude Code (%s)", self.cfg["model"])
        self.proc = subprocess.Popen(self.command(), stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                     stderr=subprocess.PIPE, text=True, bufsize=1)
        events = self.events
        threading.Thread(target=self._read, args=(self.proc, events), daemon=True).start()
        threading.Thread(target=self._read_err, args=(self.proc,), daemon=True).start()

    def _read(self, proc, events):
        for line in proc.stdout:
            try:
                events.put(json.loads(line))
            except json.JSONDecodeError:
                log.debug("claude: %s", line.rstrip())
        events.put({"type": "exited", "code": proc.wait()})

    def _read_err(self, proc):
        for line in proc.stderr:
            line = line.rstrip()
            if line:
                log.warning("claude: %s", line)
                self.last_error = line

    def alive(self):
        return self.proc is not None and self.proc.poll() is None

    def stop(self):
        if self.proc and self.proc.poll() is None:
            self.proc.kill()
            try:
                self.proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                pass
        self.proc = None

    def cancel(self):
        """Stops the current answer. The process is restarted so the next question is ready."""
        if self.lock.locked():
            self.cancelled.set()
            self.stop()

    def reset(self):
        with self.lock:
            self.start()

    def ask(self, text, jpeg=None, on_text=None):
        with self.lock:
            self.cancelled.clear()
            if not self.alive():
                self.start()
            content = []
            if jpeg:
                content.append({"type": "image", "source": {"type": "base64", "media_type": "image/jpeg",
                                                            "data": base64.b64encode(jpeg).decode()}})
            content.append({"type": "text", "text": text})
            msg = {"type": "user", "message": {"role": "user", "content": content}}
            self.turns += 1
            self.convo_bytes += len(json.dumps(msg))
            self.sent_bytes = self.convo_bytes
            try:
                self.proc.stdin.write(json.dumps(msg) + "\n")
                self.proc.stdin.flush()
            except (BrokenPipeError, OSError, AttributeError):
                self.start()
                self.proc.stdin.write(json.dumps(msg) + "\n")
                self.proc.stdin.flush()

            answer, deadline = "", time.time() + self.cfg["timeout"]
            while True:
                try:
                    e = self.events.get(timeout=max(0.1, deadline - time.time()))
                except queue.Empty:
                    self.stop()
                    raise TimeoutError("Claude took too long to answer")
                t = e.get("type")
                if t == "stream_event":
                    ev = e.get("event", {})
                    if ev.get("type") == "content_block_delta" and ev.get("delta", {}).get("type") == "text_delta":
                        answer += ev["delta"]["text"]
                        if on_text:
                            on_text(ev["delta"]["text"])
                elif t == "result":
                    if e.get("is_error") and not answer:
                        raise RuntimeError(e.get("result") or "Claude Code reported an error")
                    return answer or e.get("result") or ""
                elif t == "exited":
                    if self.cancelled.is_set():
                        return answer
                    raise RuntimeError(f"Claude Code stopped (exit {e['code']}): {self.last_error}")


class Ears:
    """Records from the microphone until you stop talking, then turns it into text."""

    RATE = 16000

    def __init__(self, cfg, data_dir):
        self.cfg = cfg["ears"]
        self.wav = data_dir / "last-question.wav"
        self.stop_event = threading.Event()

    def record(self):
        c = self.cfg
        self.stop_event.clear()
        proc = subprocess.Popen(["arecord", "-q", "-D", c["device"], "-f", "S16_LE", "-r", str(self.RATE),
                                 "-c", "1", "-t", "raw"], stdout=subprocess.PIPE)
        chunk = self.RATE // 20 * 2   # 50 ms
        frames, heard, quiet, start = [], False, 0.0, time.time()
        try:
            while not self.stop_event.is_set() and time.time() - start < c["max_seconds"]:
                data = proc.stdout.read(chunk)
                if not data:
                    break
                frames.append(data)
                samples = array.array("h", data)
                level = (sum(s * s for s in samples) / max(1, len(samples))) ** 0.5
                if level > c["level"]:
                    heard, quiet = True, 0.0
                elif heard:
                    quiet += 0.05
                    if quiet >= c["silence_seconds"]:
                        break
                elif time.time() - start > 6:
                    break   # nobody said anything
        finally:
            proc.kill()
        if not heard:
            return None
        with wave.open(str(self.wav), "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(2)
            w.setframerate(self.RATE)
            w.writeframes(b"".join(frames))
        return self.wav

    def transcribe(self, wav):
        c = self.cfg
        out = subprocess.run([str(expand(c["whisper_bin"])), "-m", str(expand(c["whisper_model"])), "-f", str(wav),
                              "-l", c["language"], "-nt", "-np"], capture_output=True, text=True, timeout=60)
        text = re.sub(r"\[[A-Z_ ]+\]|\([^)]*\)", " ", out.stdout)   # [BLANK_AUDIO], (wind noise)
        return " ".join(text.split())

    def listen_android(self):
        """Android's own speech recognition through Termux:API; it stops by itself when you pause."""
        self.stop_event.clear()
        proc = subprocess.Popen(["termux-speech-to-text"], stdout=subprocess.PIPE, text=True)
        try:
            while proc.poll() is None:
                if self.stop_event.wait(0.1):
                    proc.kill()
                    return ""
            lines = [l.strip() for l in proc.stdout.read().splitlines() if l.strip()]
        finally:
            if proc.poll() is None:
                proc.kill()
        return lines[-1] if lines else ""   # the last line is the finished sentence

    def listen(self):
        if self.cfg["engine"] == "android":
            return self.listen_android()
        wav = self.record()
        return self.transcribe(wav) if wav else ""


class Mouth:
    """Speaks text sentence by sentence while the answer is still coming in."""

    def __init__(self, cfg):
        self.cfg = cfg["mouth"]
        self.engine = self.cfg["engine"]
        self.lines = queue.Queue()
        self.buf = ""
        self.player = self.voice = None
        self.lock = threading.Lock()
        if self.engine == "piper" and not expand(self.cfg["piper_bin"]).exists():
            log.warning("Piper not found at %s, using espeak-ng", self.cfg["piper_bin"])
            self.engine = "espeak"
        threading.Thread(target=self._run, daemon=True).start()

    def _open(self):
        c = self.cfg
        if self.engine == "piper":
            voice = expand(c["piper_voice"])
            rate = json.loads(Path(str(voice) + ".json").read_text())["audio"]["sample_rate"]
            self.voice = subprocess.Popen([str(expand(c["piper_bin"])), "--model", str(voice), "--output-raw"],
                                          stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True, bufsize=1)
            self.player = subprocess.Popen(["aplay", "-q", "-D", c["device"], "-r", str(rate), "-f", "S16_LE", "-t", "raw", "-"],
                                           stdin=self.voice.stdout)

    def _run(self):
        while True:
            line = self.lines.get()
            with self.lock:
                try:
                    if self.engine == "piper":
                        if not self.voice or self.voice.poll() is not None:
                            self._open()
                        self.voice.stdin.write(line + "\n")
                        self.voice.stdin.flush()
                    elif self.engine == "android":
                        self.voice = subprocess.Popen(["termux-tts-speak", line])
                        self.voice.wait(timeout=60)
                    elif self.engine == "espeak":
                        subprocess.run(["espeak-ng", line], timeout=60)
                    else:
                        print("LENS:", line, flush=True)
                except Exception as e:
                    log.warning("speaking failed: %s", e)
                    self._close()

    def push(self, text):
        """Feed streamed text; whole sentences are spoken as soon as they are complete."""
        self.buf += text
        self.buf = NOTE_RE.sub("", self.buf)
        while True:
            cut = self.buf.find("[[")
            speakable = self.buf if cut < 0 else self.buf[:cut]
            m = re.search(r"[.!?…](\s|$)", speakable)
            if not m or not speakable[:m.end()].strip():
                return
            self.say(speakable[:m.end()])
            self.buf = self.buf[m.end():]

    def flush(self):
        rest = NOTE_RE.sub("", self.buf)
        rest = rest.split("[[")[0]
        self.buf = ""
        if rest.strip():
            self.say(rest)

    def say(self, text):
        text = " ".join(text.split())
        if text:
            self.lines.put(text)

    def _close(self):
        for p in (self.voice, self.player):
            if p and p.poll() is None:
                p.kill()
        self.voice = self.player = None

    def stop(self):
        self.buf = ""
        while not self.lines.empty():
            try:
                self.lines.get_nowait()
            except queue.Empty:
                break
        self._close()   # cuts off what is playing; the next sentence reopens the voice


class Eyes:
    """Gets what the wearer is looking at: from the glasses, a Pi camera, a USB webcam or a file."""

    def __init__(self, cfg):
        self.cfg = cfg["eyes"]

    def look(self):
        c, src = self.cfg, self.cfg["source"]
        try:
            if src == "glasses":
                with urllib.request.urlopen(c["glasses_url"], timeout=4) as r:
                    return r.read()
            if src == "picam":
                return subprocess.run(["rpicam-still", "-n", "-t", "300", "--width", "1280", "--height", "960",
                                       "-e", "jpg", "-o", "-"], capture_output=True, timeout=10).stdout or None
            if src == "usb":
                return subprocess.run(["ffmpeg", "-loglevel", "error", "-f", "v4l2", "-i", "/dev/video0", "-frames:v", "1",
                                       "-vf", "scale=1280:-2", "-f", "image2", "-c:v", "mjpeg", "-"],
                                      capture_output=True, timeout=10).stdout or None
            if src.startswith("file:"):
                return expand(src[5:]).read_bytes()
        except Exception as e:
            log.warning("camera (%s) failed: %s", src, e)
        return None


class Hub:
    HISTORY = 40     # exchanges the phone page shows (and reloads after a restart)
    KEEP_VIEWS = 60  # pictures sent to Claude that are kept for the phone page

    def __init__(self, cfg):
        self.cfg = cfg
        self.data = expand(cfg["data_dir"])
        for d in ("captures", "views"):
            (self.data / d).mkdir(parents=True, exist_ok=True)
        self.notes_file = self.data / "notes.md"
        self.history_file = self.data / "history.jsonl"
        self.brain = Brain(cfg)
        self.ears = Ears(cfg, self.data) if cfg["ears"]["engine"] != "none" else None
        self.mouth = Mouth(cfg)
        self.eyes = Eyes(cfg)
        self.busy = threading.Lock()
        self.state = "ready"
        self.started = time.time()
        self.last_used = time.time()
        self.last = {}
        self.lan_ip = ""            # the hub's own address on the hotspot/Wi-Fi, learned from requests
        self.glasses_seen = 0.0
        self.subscribers = set()    # live displays (phone page now, glasses display later)
        self.sub_lock = threading.Lock()
        self.history = deque(self._load_history(), maxlen=self.HISTORY)
        self.stats = {"questions": 0, "local": 0, "errors": 0, "seconds": 0.0, "sent_bytes": 0}
        self._cache = {}            # slow health checks (internet, battery), refreshed now and then

    # ---------- live events for displays ----------
    def publish(self, event, data):
        with self.sub_lock:
            for q in list(self.subscribers):
                q.put((event, data))

    def subscribe(self):
        q = queue.Queue(maxsize=500)
        with self.sub_lock:
            self.subscribers.add(q)
        return q

    def unsubscribe(self, q):
        with self.sub_lock:
            self.subscribers.discard(q)

    def set_state(self, state):
        self.state = state
        self.publish("state", {"state": state})

    # ---------- notes, captures, history ----------
    def notes(self):
        return self.notes_file.read_text().strip() if self.notes_file.exists() else ""

    def note_list(self):
        return [line[2:] for line in self.notes().splitlines() if line.startswith("- ")]

    def save_note(self, text):
        line = f"({datetime.now():%Y-%m-%d %H:%M}) {' '.join(text.split())}"
        with open(self.notes_file, "a") as f:
            f.write(f"- {line}\n")
        self.publish("note", {"text": line})

    def delete_note(self, index):
        notes = self.note_list()
        if 0 <= index < len(notes):
            del notes[index]
            self.notes_file.write_text("".join(f"- {n}\n" for n in notes))

    def save_capture(self, jpeg):
        path = self.data / "captures" / f"lens-{datetime.now():%Y%m%d-%H%M%S}.jpg"
        path.write_bytes(jpeg)
        self.publish("capture", {"name": path.name})
        return path

    def captures(self):
        return sorted((p.name for p in (self.data / "captures").glob("*.jpg")), reverse=True)

    def save_view(self, jpeg):
        """Keeps the picture Claude was shown, so the phone page can show it next to the answer."""
        name = f"{datetime.now():%Y%m%d-%H%M%S-%f}.jpg"
        (self.data / "views" / name).write_bytes(jpeg)
        for old in sorted((self.data / "views").glob("*.jpg"))[:-self.KEEP_VIEWS]:
            old.unlink(missing_ok=True)
        return name

    def _load_history(self):
        try:
            lines = self.history_file.read_text().splitlines()[-self.HISTORY:]
            return [json.loads(l) for l in lines if l.strip()]
        except (OSError, json.JSONDecodeError):
            return []

    def remember(self, item):
        self.history.append(item)
        with open(self.history_file, "a") as f:
            f.write(json.dumps(item) + "\n")
        if self.history_file.stat().st_size > 2_000_000:
            self.history_file.write_text("".join(json.dumps(i) + "\n" for i in self.history))

    # ---------- answering ----------
    def local_answer(self, q):
        """Things answered on the spot without asking Claude."""
        s = q.lower().strip(" .!?")
        if re.fullmatch(r"(stop|never ?mind|cancel|be quiet|shut up)", s):
            return ""
        if re.search(r"\b(take|snap) (a )?(photo|picture|pic)\b", s):
            jpeg = self.eyes.look()
            return "Saved." if jpeg and self.save_capture(jpeg) else "I couldn't get a picture."
        if re.fullmatch(r"(what('s| is) the time|what time is it)( now)?", s):
            return datetime.now().strftime("It's %-I:%M %p.")
        if re.fullmatch(r"(new conversation|start over|forget (this|our) conversation)", s):
            self.brain.reset()
            return "Okay, fresh start."
        if re.search(r"\b(your|the hub'?s?) (ip )?address\b|\bphone page\b", s):
            if not self.lan_ip:
                return "I don't know my address yet. Look for the router address in your phone's Wi-Fi details."
            return f"Open {self.lan_ip} port {self.cfg['hub']['port']} in your phone's browser."
        return None

    def answer(self, q, jpeg=None, speak=True, use_view=True):
        """Answers a question; returns the text. Only one question runs at a time."""
        local = self.local_answer(q)
        if local is not None:
            if speak and local:
                self.mouth.say(local)
            if local:
                self.stats["local"] += 1
                item = {"at": time.time(), "question": q, "answer": local, "view": None, "seconds": 0}
                self.publish("question", item)
                self.publish("answer", item)
                self.remember(item)
            return local
        if jpeg is None and use_view and self.cfg["eyes"]["send_view"]:
            jpeg = self.eyes.look()
        if jpeg and self.cfg["eyes"].get("keep_views"):
            self.save_capture(jpeg)
        item = {"at": time.time(), "question": q, "answer": "", "view": self.save_view(jpeg) if jpeg else None}
        self.publish("question", item)
        context = [f"Local time: {datetime.now():%A %Y-%m-%d %H:%M}"]
        if self.notes():
            context.append("Saved notes:\n" + self.notes())
        prompt = "<context>\n" + "\n\n".join(context) + "\n</context>\n\n" + q

        def on_text(delta):
            if speak:
                self.mouth.push(delta)
            self.publish("delta", {"text": delta})

        self.set_state("thinking")
        t0 = time.time()
        try:
            text = self.brain.ask(prompt, jpeg, on_text=on_text)
        finally:
            self.set_state("ready")
        if speak:
            self.mouth.flush()
        self.last_used = time.time()
        for note in NOTE_RE.findall(text):
            self.save_note(note)
        spoken = " ".join(NOTE_RE.sub("", text).split())
        item.update(answer=spoken, seconds=round(time.time() - t0, 2))
        self.stats["questions"] += 1
        self.stats["seconds"] += item["seconds"]
        self.stats["sent_bytes"] += getattr(self.brain, "sent_bytes", 0)
        self.publish("answer", item)
        self.remember(item)
        self.last = {"question": q, "answer": spoken, "seconds": item["seconds"], "with_view": bool(jpeg)}
        log.info("Q: %s | A (%.1fs): %s", q, time.time() - t0, spoken)
        return spoken

    def interrupt(self):
        self.mouth.stop()
        self.brain.cancel()
        if self.ears:
            self.ears.stop_event.set()

    def talk(self):
        """A button press: listen, then answer. Pressing again while it works stops it."""
        if not self.busy.acquire(blocking=False):
            self.interrupt()
            return
        try:
            self.mouth.stop()
            if not self.ears:
                self.mouth.say("I have no microphone set up.")
                return
            self.set_state("listening")
            q = self.ears.listen()
            self.set_state("ready")
            if q:
                self.answer(q)
        except Exception as e:
            log.exception("answering failed")
            self.stats["errors"] += 1
            self.mouth.say("Sorry, something went wrong. " + friendly_error(e))
            self.publish("problem", {"text": friendly_error(e)})
        finally:
            self.set_state("ready")
            self.busy.release()

    def photo(self):
        jpeg = self.eyes.look()
        if jpeg:
            log.info("photo saved: %s", self.save_capture(jpeg))
            self.mouth.say("Saved.")
        else:
            self.mouth.say("I couldn't get a picture.")

    def status(self):
        return {"state": self.state, "uptime": round(time.time() - self.started), "claude_running": self.brain.alive(),
                "claude_last_error": self.brain.last_error, "camera": self.cfg["eyes"]["source"],
                "glasses_connected": time.time() - self.glasses_seen < 70, "address": self.lan_ip, "last": self.last}


    # ---------- health, for the control panel ----------
    def cached(self, key, seconds, fn):
        hit = self._cache.get(key)
        if hit and time.time() - hit[0] < seconds:
            return hit[1]
        value = fn()
        self._cache[key] = (time.time(), value)
        return value

    @staticmethod
    def _internet():
        try:
            with socket.create_connection(("api.anthropic.com", 443), timeout=3):
                return True
        except OSError:
            return False

    @staticmethod
    def _battery():
        if not shutil.which("termux-battery-status"):
            return None
        try:
            out = subprocess.run(["termux-battery-status"], capture_output=True, text=True, timeout=6).stdout
            b = json.loads(out)
            return {"percent": b.get("percentage"), "charging": b.get("status") in ("CHARGING", "FULL"),
                    "temp": b.get("temperature")}
        except Exception:
            return None

    def tail(self, name, lines):
        try:
            return (self.data / name).read_text(errors="replace").splitlines()[-lines:]
        except OSError:
            return []

    def health(self):
        c = self.cfg
        ears, mouth, eyes = c["ears"]["engine"], c["mouth"]["engine"], c["eyes"]["source"]
        parts = []

        def part(key, name, status, detail, test=None):
            parts.append({"key": key, "name": name, "status": status, "detail": detail, "test": test})

        err = self.brain.last_error
        if self.brain.alive():
            part("claude", "Claude Code", "ok", f"Running ({c['claude']['model']}, effort {c['claude']['effort']}). "
                 f"{self.brain.turns} questions in this conversation.", "claude")
        elif re.search(r"log ?in|auth|401", err, re.I):
            part("claude", "Claude Code", "fail", "Logged out. Run claude-proot (or claude) and log in again.", "claude")
        else:
            part("claude", "Claude Code", "warn", "Not running right now; the hub restarts it within a few seconds."
                 + (f" Last message: {err[:160]}" if err else ""), "claude")

        if ears == "none":
            part("ears", "Microphone", "off", "No microphone set up (ears engine is none).")
        elif ears == "android":
            ok = bool(shutil.which("termux-speech-to-text"))
            part("ears", "Microphone", "ok" if ok else "fail", "Android speech recognition through Termux:API."
                 if ok else "Termux:API commands missing: pkg install termux-api, and install the Termux:API app.", "listen")
        else:
            ok = expand(c["ears"]["whisper_bin"]).exists() and expand(c["ears"]["whisper_model"]).exists()
            part("ears", "Microphone", "ok" if ok else "fail", "whisper.cpp speech recognition."
                 if ok else "whisper.cpp or its model is missing: run setup.sh.", "listen")

        if mouth == "none":
            part("mouth", "Speaker", "off", "Answers are only printed (mouth engine is none).", "speak")
        else:
            ok = bool(shutil.which("termux-tts-speak")) if mouth == "android" else True
            names = {"android": "Android's text-to-speech", "piper": "Piper voice", "espeak": "espeak-ng voice"}
            part("mouth", "Speaker", "ok" if ok else "fail", f"{names.get(mouth, mouth)}. Plays through paired earbuds if connected."
                 if ok else "Termux:API commands missing.", "speak")

        if eyes == "glasses":
            seen = time.time() - self.glasses_seen
            if seen < 70:
                part("eyes", "Glasses camera", "ok", f"Connected at {c['eyes']['glasses_url'].split('/')[2]}.", "camera")
            else:
                part("eyes", "Glasses camera", "warn", "Glasses not connected. They check in every 30 s once they're on "
                     "the same Wi-Fi; questions still work without a picture.", "camera")
        elif eyes == "none":
            part("eyes", "Camera", "off", "No camera (eyes source is none).")
        else:
            part("eyes", "Camera", "ok", f"Source: {eyes}.", "camera")

        online = self.cached("internet", 30, self._internet)
        part("net", "Internet", "ok" if online else "fail", "Can reach Claude's servers." if online
             else "Can't reach Claude's servers. Check mobile data or Wi-Fi.")

        st = self.stats
        return {
            "now": time.time(), "state": self.state, "uptime": round(time.time() - self.started),
            "address": self.lan_ip, "port": c["hub"]["port"], "parts": parts,
            "battery": self.cached("battery", 60, self._battery),
            "stats": {**st, "average": round(st["seconds"] / st["questions"], 1) if st["questions"] else None,
                      "notes": len(self.note_list()), "photos": len(self.captures())},
            "settings": {"model": c["claude"]["model"], "effort": c["claude"]["effort"],
                         "fresh_after_minutes": c["claude"]["fresh_after_minutes"], "web_search": c["claude"]["web_search"],
                         "ears": ears, "mouth": mouth, "camera": eyes, "send_view": c["eyes"]["send_view"],
                         "data_dir": str(self.data)},
            "log": self.tail("lens.log", 80),
            "doctor": self.tail("doctor-report.md", 30),
        }

    def test(self, what):
        """One-tap checks from the control panel."""
        if what == "speak":
            self.mouth.say("Hi, this is Lens. If you can hear me, the speaker works.")
            return {"ok": True, "text": "Speaking now. You should hear a short sentence."}
        if what == "camera":
            jpeg = self.eyes.look()
            if not jpeg:
                return {"ok": False, "text": "No picture came back from the camera."}
            return {"ok": True, "text": f"Got a {len(jpeg) // 1024} KB picture.", "view": self.save_view(jpeg)}
        if not self.busy.acquire(timeout=5):
            return {"ok": False, "text": "Busy answering a question; try again in a moment."}
        try:
            if what == "listen":
                if not self.ears:
                    return {"ok": False, "text": "No microphone is set up."}
                self.set_state("listening")
                heard = self.ears.listen()
                return {"ok": bool(heard), "text": f"Heard: \u201c{heard}\u201d" if heard else "Didn't hear anything. Speak right after tapping."}
            if what == "claude":
                t0 = time.time()
                reply = self.brain.ask("This is a connection test from the control panel. Reply with exactly: Claude is working.")
                return {"ok": "working" in reply.lower(), "text": f"{reply.strip()} ({time.time() - t0:.1f} s)"}
        except Exception as e:
            return {"ok": False, "text": str(e)[:300]}
        finally:
            self.set_state("ready")
            self.busy.release()
        return {"ok": False, "text": "Unknown test."}


def app_icon(size):
    """The home-screen icon: a lens ring on dark glass, drawn without any image library."""
    bg, ring, glint = (5, 8, 13), (127, 227, 255), (233, 250, 255)
    c, r_out, r_in = size / 2, size * 0.36, size * 0.27
    rows = bytearray()
    for y in range(size):
        rows.append(0)
        for x in range(size):
            d = ((x + 0.5 - c) ** 2 + (y + 0.5 - c) ** 2) ** 0.5
            gx, gy = x - size * 0.40, y - size * 0.38
            if (gx * gx + gy * gy) ** 0.5 < size * 0.05:
                rows += bytes(glint)
            elif r_in <= d <= r_out:
                rows += bytes(ring)
            else:
                rows += bytes(bg)

    def chunk(t, data):
        return struct.pack(">I", len(data)) + t + data + struct.pack(">I", zlib.crc32(t + data) & 0xFFFFFFFF)
    return (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress(bytes(rows), 9)) + chunk(b"IEND", b""))


MANIFEST = {"name": "Lens", "short_name": "Lens", "start_url": "/panel", "display": "standalone",
            "background_color": "#05080d", "theme_color": "#05080d",
            "icons": [{"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"},
                      {"src": "/icon-512.png", "sizes": "512x512", "type": "image/png"}]}
ICONS = {}


def friendly_error(e):
    s = str(e).lower()
    if "timeout" in s or "too long" in s:
        return "Claude took too long."
    if "login" in s or "auth" in s or "401" in s:
        return "Claude Code needs logging in again."
    if "limit" in s or "429" in s:
        return "We've hit the usage limit for now."
    return "Try again in a moment."


SAFE_NAME = re.compile(r"[\w.-]+\.jpg")


def make_handler(hub):
    token = hub.cfg["hub"]["token"]

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, fmt, *args):
            log.debug("http: " + fmt, *args)

        def reply(self, code, body):
            self.send_bytes(code, json.dumps(body).encode(), "application/json")

        def send_bytes(self, code, data, ctype, extra=()):
            self.send_response(code)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(data)))
            self.send_header("Cache-Control", "no-store")
            for k, v in extra:
                self.send_header(k, v)
            self.end_headers()
            self.wfile.write(data)

        def authorised(self, query):
            """The glasses send the token in a header; the phone page carries it in a cookie."""
            if not token:
                return True
            cookie = dict(c.strip().split("=", 1) for c in self.headers.get("Cookie", "").split(";") if "=" in c)
            return token in (self.headers.get("X-Lens-Token"), cookie.get("lens"), query.get("t", [None])[0])

        def note_address(self):
            hub.lan_ip = self.connection.getsockname()[0]

        def do_GET(self):
            url = urllib.parse.urlsplit(self.path)
            query = urllib.parse.parse_qs(url.query)
            self.note_address()
            if url.path == "/manifest.webmanifest":
                return self.send_bytes(200, json.dumps(MANIFEST).encode(), "application/manifest+json")
            m = re.fullmatch(r"/icon-(192|512)\.png", url.path)
            if m:
                size = int(m.group(1))
                ICONS.setdefault(size, app_icon(size))
                return self.send_bytes(200, ICONS[size], "image/png")
            if not self.authorised(query):
                if url.path == "/status":
                    return self.reply(200, {"state": hub.state})
                page = b"<!doctype html><meta name=viewport content='width=device-width'><body style='font:18px system-ui;padding:24px;background:#05080d;color:#e8f7ff'>Open this page as <b>http://this-address:port/?t=YOUR_TOKEN</b> (the token from config.toml). It's remembered after that."
                return self.send_bytes(403, page, "text/html; charset=utf-8")
            if url.path == "/status":
                return self.reply(200, hub.status())
            pages = {"/": "phone.html", "/panel": "panel.html"}
            if url.path in pages:
                if "t" in query:   # remember the token and drop it from the address bar
                    cookie = f"lens={token}; Path=/; Max-Age=31536000; SameSite=Strict"
                    return self.send_bytes(303, b"", "text/plain", [("Location", url.path), ("Set-Cookie", cookie)])
                return self.send_bytes(200, (HERE / pages[url.path]).read_bytes(), "text/html; charset=utf-8")
            if url.path == "/api/health":
                return self.reply(200, hub.health())
            if url.path == "/events":
                return self.stream_events()
            if url.path == "/api/state":
                return self.reply(200, {**hub.status(), "history": list(hub.history), "notes": hub.note_list(),
                                        "captures": hub.captures()[:60]})
            m = re.fullmatch(r"/(views|captures)/(.+)", url.path)
            if m and SAFE_NAME.fullmatch(m.group(2)):
                path = hub.data / m.group(1) / m.group(2)
                if path.is_file():
                    return self.send_bytes(200, path.read_bytes(), "image/jpeg")
            self.reply(404, {"error": "not found"})

        def stream_events(self):
            """Server-sent events: the page (and later the glasses display) sees everything as it happens."""
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            q = hub.subscribe()
            try:
                self.wfile.write(b"retry: 2000\n\n")
                self.wfile.flush()
                while True:
                    try:
                        event, data = q.get(timeout=15)
                        self.wfile.write(f"event: {event}\ndata: {json.dumps(data)}\n\n".encode())
                    except queue.Empty:
                        self.wfile.write(b": still here\n\n")
                    self.wfile.flush()
            except (BrokenPipeError, ConnectionResetError, OSError):
                pass
            finally:
                hub.unsubscribe(q)

        def do_POST(self):
            url = urllib.parse.urlsplit(self.path)
            self.note_address()
            if not self.authorised(urllib.parse.parse_qs(url.query)):
                return self.reply(403, {"error": "bad token"})
            try:
                body = json.loads(self.rfile.read(int(self.headers.get("Content-Length") or 0)) or b"{}")
            except json.JSONDecodeError:
                return self.reply(400, {"error": "body must be JSON"})
            path = url.path
            if path == "/hello":
                # The glasses say where they are, so this works on any Wi-Fi without fixed addresses.
                ip = str(body.get("ip") or self.client_address[0])
                if hub.cfg["eyes"]["glasses_url"] != f"http://{ip}/capture":
                    log.info("glasses are at %s", ip)
                hub.cfg["eyes"]["glasses_url"] = f"http://{ip}/capture"
                hub.glasses_seen = time.time()
                hub.publish("glasses", {"connected": True})
                return self.reply(200, {"ok": True})
            if path == "/button":
                event = body.get("event", "press")
                target = {"long": hub.photo, "stop": hub.interrupt}.get(event, hub.talk)
                threading.Thread(target=target, daemon=True).start()
                return self.reply(200, {"ok": True, "event": event})
            m = re.fullmatch(r"/test/(speak|listen|camera|claude)", path)
            if m:
                return self.reply(200, hub.test(m.group(1)))
            if path == "/notes/delete":
                hub.delete_note(int(body.get("index", -1)))
                return self.reply(200, {"notes": hub.note_list()})
            if path == "/ask":
                q = str(body.get("text", "")).strip()
                if not q:
                    return self.reply(400, {"error": "text is required"})
                jpeg = base64.b64decode(body["image"]) if body.get("image") else None
                if not hub.busy.acquire(timeout=30):
                    return self.reply(409, {"error": "busy"})
                try:
                    text = hub.answer(q, jpeg, speak=bool(body.get("speak", True)), use_view=bool(body.get("look", True)))
                    return self.reply(200, {"answer": text, **hub.last})
                except Exception as e:
                    log.exception("ask failed")
                    hub.stats["errors"] += 1
                    hub.publish("problem", {"text": friendly_error(e)})
                    return self.reply(500, {"error": str(e)})
                finally:
                    hub.busy.release()
            self.reply(404, {"error": "not found"})

    return Handler


def setup_logging(data_dir, verbose=False):
    data_dir.mkdir(parents=True, exist_ok=True)
    fmt = logging.Formatter("%(asctime)s %(levelname)s %(message)s")
    file = logging.handlers.RotatingFileHandler(data_dir / "lens.log", maxBytes=1_000_000, backupCount=2)
    file.setFormatter(fmt)
    console = logging.StreamHandler()
    console.setFormatter(fmt)
    log.handlers[:] = [file, console]
    log.setLevel(logging.DEBUG if verbose else logging.INFO)


def selftest(cfg, deep=False):
    """Prints a check list; returns the number of problems."""
    problems = 0

    def check(ok, what, fix=""):
        nonlocal problems
        print(("  ok   " if ok else "  FAIL ") + what + ("" if ok or not fix else f"  ->  {fix}"))
        problems += not ok

    print("Lens self-test")
    claude = shutil.which(cfg["claude"]["bin"])
    check(bool(claude), "Claude Code installed", "run setup.sh, or install from https://claude.ai/install.sh")
    if claude and deep:
        r = subprocess.run([claude, "-p", "Reply with just OK", "--model", "haiku", "--no-session-persistence"],
                           capture_output=True, text=True, timeout=90)
        check(r.returncode == 0 and "OK" in r.stdout.upper(), "Claude Code answers (logged in)",
              "run `claude` once and log in with your account: " + (r.stderr.strip() or r.stdout.strip())[:200])
    if cfg["ears"]["engine"] == "whisper":
        check(expand(cfg["ears"]["whisper_bin"]).exists(), "whisper.cpp built", "run setup.sh")
        check(expand(cfg["ears"]["whisper_model"]).exists(), "speech model downloaded", "run setup.sh")
        check(bool(shutil.which("arecord")), "arecord (microphone)", "sudo apt install alsa-utils")
    if "android" in (cfg["ears"]["engine"], cfg["mouth"]["engine"]):
        check(bool(shutil.which("termux-tts-speak")), "Termux:API commands",
              "pkg install termux-api, and install the Termux:API app from F-Droid")
    if cfg["mouth"]["engine"] == "piper":
        ok = expand(cfg["mouth"]["piper_bin"]).exists() and expand(cfg["mouth"]["piper_voice"]).exists()
        check(ok or bool(shutil.which("espeak-ng")), "a voice (Piper or espeak-ng)", "run setup.sh")
        check(bool(shutil.which("aplay")), "aplay (speaker)", "sudo apt install alsa-utils")
    src = cfg["eyes"]["source"]
    if src == "glasses":
        try:
            with urllib.request.urlopen(cfg["eyes"]["glasses_url"].replace("/capture", "/status"), timeout=4) as r:
                check(r.status == 200, "glasses reachable")
        except Exception as e:
            print(f"  warn glasses not reachable ({e}); fine if they're switched off")
    elif src.startswith("file:"):
        check(expand(src[5:]).exists(), f"test picture {src[5:]}")
    try:
        quiet = json.loads(json.dumps(cfg))
        quiet["ears"]["engine"] = quiet["mouth"]["engine"] = "none"
        Hub(quiet)
        check(True, "hub starts up")
    except Exception as e:
        check(False, f"hub starts up ({type(e).__name__}: {e})")
    print("all good" if not problems else f"{problems} problem(s)")
    return problems


def console(hub):
    print("Type a question and press Enter. Enter on its own = talk with the microphone. Ctrl+C quits.")
    for line in sys.stdin:
        q = line.strip()
        try:
            if not q:
                hub.talk()
            else:
                with hub.busy:
                    hub.answer(q)
        except Exception as e:
            print("error:", e)


def main():
    ap = argparse.ArgumentParser(description="Lens hub for the DIY smart glasses")
    ap.add_argument("--config", default=str(HERE / "config.toml"))
    ap.add_argument("--console", action="store_true", help="type questions in the terminal")
    ap.add_argument("--selftest", action="store_true", help="check the setup and exit")
    ap.add_argument("--deep", action="store_true", help="with --selftest: also ask Claude a test question")
    ap.add_argument("-v", "--verbose", action="store_true")
    args = ap.parse_args()

    cfg = load_config(args.config)
    if args.selftest:
        sys.exit(1 if selftest(cfg, args.deep) else 0)
    setup_logging(expand(cfg["data_dir"]), args.verbose)
    try:
        run_hub(cfg, args)
    except Exception:
        log.exception("hub crashed")
        raise


def run_hub(cfg, args):
    hub = Hub(cfg)
    hub.brain.start()   # warm up now so the first question is quick
    server = ThreadingHTTPServer(("0.0.0.0", cfg["hub"]["port"]), make_handler(hub))
    server.daemon_threads = True   # open phone pages mustn't stop the hub from exiting
    threading.Thread(target=server.serve_forever, daemon=True).start()
    log.info("Lens hub ready on port %s (camera: %s)", cfg["hub"]["port"], cfg["eyes"]["source"])
    try:
        if args.console:
            console(hub)
        else:
            fresh_after = cfg["claude"]["fresh_after_minutes"] * 60
            while True:
                time.sleep(5)
                idle = not hub.brain.lock.locked() and not hub.busy.locked()
                if not hub.brain.alive() and idle:
                    log.warning("Claude Code process is gone, restarting it")
                    hub.brain.start()
                elif fresh_after and idle and hub.brain.turns and time.time() - hub.last_used > fresh_after:
                    # A new conversation stops old pictures being re-sent with every question,
                    # which saves mobile data and usage limits.
                    log.info("quiet for a while: starting a fresh conversation")
                    hub.brain.start()
    except KeyboardInterrupt:
        pass
    finally:
        hub.brain.stop()
        hub.mouth.stop()


if __name__ == "__main__":
    main()
