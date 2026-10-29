# Lens: DIY smart glasses

A home-made take on camera + AI glasses: you look at something, ask out loud, and hear an answer. Personal project, not for sale.

The glasses stay as normal-looking as possible by carrying almost nothing: a thumb-sized camera board, a button and a recording light. The thinking happens on a pocket hub that runs Claude Code logged into your own Claude account, so there's no API bill. The hub can be a Raspberry Pi or an old Android phone with a data-only SIM.

```
 glasses (XIAO ESP32S3 Sense)         pocket (Pi or old phone)                        internet
 camera, button, light  ── Wi-Fi ──►  lens.py: listen → whisper.cpp → Claude Code ──► Claude
                                       speak ◄─ Piper ◄─ answer                  ◄──
                                       doctor.py: fixes the hub if it keeps crashing
 earbuds ◄──── Bluetooth ────────────  (phone hotspot gives the Pi and glasses internet)
```

| Folder | What it is |
|---|---|
| `pi/` | The pocket hub: `lens.py`, the repair doctor, `setup.sh` for a Pi, and `android/` for a phone |
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

**3D-printed frame and wiring:** `hardware/` has the printable model (`hardware/stl/`) and a build sheet with a 3D view, the wiring diagram and the build order: [Lens Glasses Blueprint](https://claude.ai/artifact/6CeASikG2KLT9uxdUsBuyB) (also `hardware/blueprint.html`).

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
2. On the Pi: `git clone -b ccr-ce1e995c-dmjvgi https://github.com/TheEmeraldPig/mobileClaude ~/mobileClaude && bash ~/mobileClaude/glasses/pi/setup.sh`
3. Run `claude` once and log in with your Claude account.
4. Pair your earbuds (Bluetooth icon on the desktop, or `bluetoothctl`).
5. `python3 lens.py --selftest --deep`, then `systemctl --user start lens`.

Try it before the glasses exist: set `source = "picam"` or `"usb"` in `config.toml` for a Pi camera or webcam, or run `python3 lens.py --console` to type questions. It also runs on any Linux or Mac computer with Claude Code installed.

How it stays fast: Claude Code is started once and kept running, so a question doesn't wait for it to start up (about 0.5 s to the first word on a follow-up, versus 2 s or more starting fresh). Answers are spoken sentence by sentence while the rest is still being written. "What time is it", "take a photo", "stop" and "new conversation" are handled on the Pi without asking Claude at all.

How it stays running: systemd starts the hub on boot and restarts it if it crashes. If it crashes 5 times in 5 minutes, `doctor.py` runs: it saves the code in git, has Claude Code read the logs and fix the problem inside `glasses/pi/` only, keeps the fix if the self-test passes (otherwise rolls it back), and restarts the hub. It runs at most 3 times a day, and writes what it did to `~/lens-data/doctor-report.md`. Its fixes go on a local `pi-fixes` branch: push that branch so the fix can be checked and merged into the main code.

The hub also answers on the network, for the glasses and for testing: `GET /status`, `POST /ask {"text": "..."}` and `POST /button {"event": "press"|"long"}` on port 8765, with the `X-Lens-Token` header from `config.toml`.

### Pocket hub on an Android phone (instead of a Pi)

An old Android phone (tested design: Galaxy A26) makes a cheaper and smaller hub: battery, mic, speaker and mobile data are built in, so no Pi or power bank. A data-only SIM in it gives the glasses internet anywhere, and its hotspot is the Wi-Fi the glasses join.

**SIM:** any data-only plan the phone's network supports, 1–2 GB a month is enough. **It must include hotspot** (check the plan page). Before buying, put the phone's IMEI (dial `*#06#`) into the carrier's compatibility checker.

**Setup:**
1. Install **F-Droid** (f-droid.org), then from it **Termux**, **Termux:API** and **Termux:Boot**. Open each once. (The Termux on the Play Store is outdated; use F-Droid's.)
2. In Termux: `pkg install git && git clone -b ccr-ce1e995c-dmjvgi https://github.com/TheEmeraldPig/mobileClaude ~/mobileClaude && bash ~/mobileClaude/glasses/pi/android/setup.sh`
3. Log Claude Code into your account: `claude-proot`, open the link, sign in, paste the code back, then `/exit`. (Claude Code has no Android build, so it runs inside a small Ubuntu that setup installs; `claude-proot` is the shortcut into it.)
4. In `~/mobileClaude/glasses/pi/config.toml`, set `token`.
5. Android settings:
   - **Settings → Battery → Background usage limits → Never sleeping apps:** add Termux and Termux:API. Also set both to **Unrestricted** under each app's Battery settings. Otherwise Android pauses the hub.
   - Let **Termux:API** use the microphone (it asks the first time).
   - **Settings → Connections → Mobile Hotspot:** turn it on, set the **band to 2.4 GHz** (the glasses can't see 5 GHz), and note the name and password for the glasses.
   - Pair your earbuds with this phone.
6. `cd ~/mobileClaude/glasses/pi && python3 lens.py --selftest --deep`, then `bash android/run.sh`, or restart the phone and Termux:Boot starts it.

On the phone, speech recognition and the voice are Android's own (through Termux:API), so there's nothing to build. `android/run.sh` does what systemd does on a Pi: restarts the hub when it stops and runs the doctor after 5 crashes in 5 minutes. After 10 quiet minutes the hub starts a fresh conversation, so old pictures aren't re-sent with every question; that keeps mobile data and Max usage down (`fresh_after_minutes` in `config.toml`).

In the glasses' `secrets.h`, use the hub phone's hotspot name and password and `HUB_HOST "gateway"`: the phone making the hotspot is the hub, so the glasses always know where it is, and they tell the hub their own address when they connect.

### Seeing answers: the phone page

Answers are spoken into your earbuds. For anything you'd rather read (lists, steps, translations, your notes, the photos you took), the hub serves a live page:

1. Join your main phone to the hub's Wi-Fi: the hub phone's hotspot (or the same Wi-Fi as the Pi). Joining someone's hotspot is fine on any phone plan.
2. Open `http://<hub address>:8765/?t=<your token>` in its browser. The hub address is the **router/gateway** in that Wi-Fi network's details on your phone, or ask the glasses "what's your address?". The token is remembered after the first visit; bookmark the page or add it to your home screen.

The page shows each question as it's asked, the answer as it's written, and the picture Claude was looking at, plus your history, notes (you can delete them there) and photos. Its Talk, Photo and Stop buttons work like the glasses button, and you can type a question.

The page reads the hub's live event stream (`GET /events`). A display inside the glasses is planned as the next step and will read the same stream.

### Testing without the glasses: your screen as the camera

`tools/screen_glasses.py` stands in for the glasses on your computer: it sends a picture of your screen whenever the hub looks, tells the hub where it is, and the keyboard is the button. Handy for asking "what's this error?" while you're debugging, and for testing the hub and phone page before the glasses are built.

```
pip install mss pillow
python3 glasses/tools/screen_glasses.py --hub http://<hub address>:8765 --token <your token>
```

Enter = button press (talk / stop), `p` = photo, `s` = stop, `q` = quit. The computer and the hub must be on the same Wi-Fi; on Windows, allow Python through the firewall on private networks when it asks. Everything on the screen goes to Claude when you ask, so keep private things off it while it runs. `--monitor 2` uses a second screen.

### The glasses (XIAO ESP32S3 Sense)

1. Arduino IDE → install the **esp32** boards package by Espressif → board **XIAO_ESP32S3**, **Tools → PSRAM → OPI PSRAM**.
2. Copy `firmware/lens_glasses/secrets.example.h` to `secrets.h` and fill in the hotspot name, password and the hub token.
3. Upload. The light blinks twice when it's on Wi-Fi.
4. Wiring: button between **D1** and **GND**; LED (through the 220 Ω resistor) from **D0** to **GND**; battery to the BAT pads on the back.
5. Mount it: camera at the hinge pointing forward, board and battery along one temple arm, button on top of the arm, light next to the camera where people can see it.

With a Pi, the glasses find it as `lens-hub.local`; with a hub phone, they use the hotspot's address (`HUB_HOST "gateway"`). Either way they tell the hub their own address when they connect, so it doesn't matter what addresses the Wi-Fi hands out.

Later upgrades: a small speaker by the ear (MAX98357A amp + 8 Ω speaker), and streaming the board's own microphone so earbuds become optional.

### Be a good camera wearer

Keep the recording light working and visible, don't film people who haven't agreed to it, and check local rules on recording audio: some places need everyone's consent.
