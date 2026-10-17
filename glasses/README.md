# Lens: DIY smart glasses

A home-made take on camera + AI glasses: you look at something, ask out loud, and hear an answer. Personal project, not for sale.

The glasses stay as normal-looking as possible by carrying almost nothing: a thumb-sized camera board, a button and a recording light. The thinking happens on a Raspberry Pi in your pocket, which runs Claude Code logged into your own Claude account, so there's no API bill.

```
 glasses (XIAO ESP32S3 Sense)         pocket (Raspberry Pi)                          internet
 camera, button, light  ── Wi-Fi ──►  lens.py: listen → whisper.cpp → Claude Code ──► Claude
                                       speak ◄─ Piper ◄─ answer                  ◄──
                                       doctor.py: fixes the hub if it keeps crashing
 earbuds ◄──── Bluetooth ────────────  (phone hotspot gives the Pi and glasses internet)
```

| Folder | What it is |
|---|---|
| `pi/` | The pocket hub: `lens.py`, the repair doctor, services and `setup.sh` |
| `firmware/lens_glasses/` | Arduino sketch for the glasses board |
| `index.html` | The phone-only version (uses an API key). Handy for trying features without any hardware |

## The phone-only version

Open the page on your phone (see "Hosting" below), tap **MENU**, paste a Claude API key from [console.anthropic.com](https://console.anthropic.com), and allow camera and microphone.

| Control | What it does |
|---|---|
| Mic (big button) | Tap and talk; it stops when you pause. Or hold it while you talk. |
| Eye | "What am I looking at?" (or the current mode's question) |
| Camera / video | Take a photo or start/stop a video. Captures are in MENU. |
| Keyboard | Type a question instead of talking |
| Mode chips | **Ask**, **Read** (reads text out), **Translate** (language in MENU), **Identify**, **How do I** (step-by-step help) |
| Tap the answer | Stop talking |

Say "remember that I parked on level 3" and it saves a note (in MENU). Later, "where did I park?" works. It can also search the web for things like opening hours.

On a computer: Space talk, L look, P photo, V video, T type, Esc stop.

Notes:
- The API key, notes and settings are stored only in that browser. Each question costs a little API money; the camera picture is the biggest part. Haiku 5.5 in MENU is the cheapest model.
- Voice input needs Chrome (Android) or Safari (iPhone). Firefox can only type.
- Captures last until you close the page; tap one to save or share it.

### Hosting

The camera and mic only work on an `https://` page. `.github/workflows/glasses.yml` publishes this folder to GitHub Pages on every push that touches `glasses/`. One-time setup: repo **Settings → Pages → Source: GitHub Actions**. Pages only accepts the repo's default branch at first; to publish from other branches too, set **Settings → Environments → github-pages → Deployment branches** to allow them. The page is then at `https://<user>.github.io/<repo>/`.

To try it on a computer without Pages: `npx serve glasses` (or `python3 -m http.server -d glasses`) and open `http://localhost:3000`; localhost counts as secure.

## Building the glasses

### Parts (about $90–130, one time)

| Part | Why | Rough price |
|---|---|---|
| **Raspberry Pi 5 (4 GB)** | The pocket hub. Runs Claude Code, speech-to-text and the voice. A Pi 4 works too, a bit slower. The Pi Zero 2 W is too short on memory for Claude Code plus speech recognition. | $60 |
| microSD card, 32 GB | The Pi's drive | $8 |
| USB-C power bank (5 V 3 A / 15 W or more) | Powers the Pi in your pocket | $15–25 |
| **Seeed Studio XIAO ESP32S3 Sense** | The glasses: camera, Wi-Fi and battery charging on a thumb-sized board | $14–18 |
| 3.7 V LiPo, 250–500 mAh, JST plug | Powers the glasses | $5–8 |
| 6 mm push button | Short press: talk / stop. Long press: take a photo | $1 |
| Small LED + 220 Ω resistor | Recording light, on whenever the camera takes a picture | $1 |
| Cheap sunglasses or safety glasses | The frame | $5–10 |
| Wire, heat-shrink, hot glue | Holding it together | $5 |
| Bluetooth earbuds (or a USB mic headset) | You probably own some. The Pi listens through their mic and talks through them | $0 |

### Pocket hub (Raspberry Pi)

1. Flash **Raspberry Pi OS (64-bit)** with Raspberry Pi Imager. In its settings, add your phone's hotspot as the Wi-Fi network.
2. On the Pi: `git clone https://github.com/TheEmeraldPig/mobileClaude ~/mobileClaude && bash ~/mobileClaude/glasses/pi/setup.sh`
3. Run `claude` once and log in with your Claude account.
4. Pair your earbuds (Bluetooth icon on the desktop, or `bluetoothctl`).
5. `python3 lens.py --selftest --deep`, then `systemctl --user start lens`.

Try it before the glasses exist: set `source = "picam"` or `"usb"` in `config.toml` for a Pi camera or webcam, or run `python3 lens.py --console` to type questions. It also runs on any Linux or Mac computer with Claude Code installed.

How it stays fast: Claude Code is started once and kept running, so a question doesn't wait for it to start up (about 0.5 s to the first word on a follow-up, versus 2 s or more starting fresh). Answers are spoken sentence by sentence while the rest is still being written. "What time is it", "take a photo", "stop" and "new conversation" are handled on the Pi without asking Claude at all.

How it stays running: systemd starts the hub on boot and restarts it if it crashes. If it crashes 5 times in 5 minutes, `doctor.py` runs: it saves the code in git, has Claude Code read the logs and fix the problem inside `glasses/pi/` only, keeps the fix if the self-test passes (otherwise rolls it back), and restarts the hub. It runs at most 3 times a day, and writes what it did to `~/lens-data/doctor-report.md`. Its fixes go on a local `pi-fixes` branch: push that branch so the fix can be checked and merged into the main code.

The hub also answers on the network, for the glasses and for testing: `GET /status`, `POST /ask {"text": "..."}` and `POST /button {"event": "press"|"long"}` on port 8765, with the `X-Lens-Token` header from `config.toml`.

### The glasses (XIAO ESP32S3 Sense)

1. Arduino IDE → install the **esp32** boards package by Espressif → board **XIAO_ESP32S3**, **Tools → PSRAM → OPI PSRAM**.
2. Copy `firmware/lens_glasses/secrets.example.h` to `secrets.h` and fill in the hotspot name, password and the hub token.
3. Upload. The light blinks twice when it's on Wi-Fi.
4. Wiring: button between **D1** and **GND**; LED (through the 220 Ω resistor) from **D0** to **GND**; battery to the BAT pads on the back.
5. Mount it: camera at the hinge pointing forward, board and battery along one temple arm, button on top of the arm, light next to the camera where people can see it.

The glasses find the Pi as `lens-hub.local` and the Pi finds the glasses as `lens-glasses.local`, so it doesn't matter what addresses the hotspot hands out.

Later upgrades: a small speaker by the ear (MAX98357A amp + 8 Ω speaker), and streaming the board's own microphone so earbuds become optional.

### Be a good camera wearer

Keep the recording light working and visible, don't film people who haven't agreed to it, and check local rules on recording audio: some places need everyone's consent.
