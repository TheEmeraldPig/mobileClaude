# mobileClaude

## sandbox/ (the game "Mote")

- A single-file browser game in `sandbox/index.html`, played mostly on a phone in landscape. Keep phone (touch) and computer (keyboard and mouse) layouts working; `IS_TOUCH` picks between them.
- **World generation is seeded.** Any terrain or world feature must follow the rules in `sandbox/README.md` under "Adding terrain features". The short version: no `Math.random()` in generation, add features as `GEN_PASSES`, and handle features that cross chunk borders.
- **UI style comes from the mote sprite:** dark outline `--o`, violet body `--b`, light top-left edge `--l`, shaded bottom-right edge `--s`, gold accent `--t`, 3px pixel bevels via `.btn`, `.px` and `.sheet`, and the Pixelify Sans and Silkscreen fonts. New menus and buttons should reuse these classes.
- Test with Playwright using the preinstalled Chromium (`executablePath: '/opt/pw-browsers/chromium'`), in both a desktop viewport and a phone context (`hasTouch: true, isMobile: true`).

## mote/ (native version of Mote: Windows .exe, Android .apk, Steam)

- A Go + Ebitengine port of the game. See `mote/README.md`.
- This native version is now the main one; the web version in `sandbox/` is no longer kept in sync and builds different worlds.
- `game/world_test.go` must keep passing: a seed must always build the same chunks. If you change world generation on purpose, update its reference numbers and say so (every seed's world changes).
- Lighting is a flood fill in `game/render.go` (levels 0-64). New light-giving blocks go in `blockLight()` in `world.go`.
- New block types: add an id, a colour in `blockRGB`, `isSolid`/`blocksLight` handling and a `blockItem` entry in `world.go`, and sim handling in `sim.go` if they move or grow.
- The game is a sandbox like Terraria/Starbound (not a roguelite). Each world is a save in `worlds/<id>.json` with a game mode (Easy, Normal, Hard, Hardcore; see `worlds.go`). New world or character state must go in `serializeState`/`restoreState` in `save.go`.
- Workstations are background objects (`stations.go`), not blocks: they don't collide, have a dark outline, open their own crafting menu, and give back one item when mined. Recipes say which station they need (`at` in `items.go`).
- Building tools (shapes, walls, paint, rotate/flip, holograms) are in `build.go`; keep the build bar usable on phones.
- Every push to `mote/` builds and releases `Mote.exe`, `Mote-steam.exe` and `Mote.apk` through `.github/workflows/mote.yml`.
- Check layouts by running under Xvfb with the test hooks in `game/testhooks.go` (`MOTE_SCRIPT`, `MOTE_TOUCH=1`, `MOTE_NOAUDIO=1`).

## glasses/ (Lens: DIY smart glasses)

- The main design: glasses with only a XIAO ESP32S3 Sense (camera, button, recording LED; `firmware/lens_glasses/`) talk over Wi-Fi to a Raspberry Pi hub in the pocket (`pi/`). See `glasses/README.md`.
- `pi/lens.py` runs Claude Code as one long-lived `claude -p --input-format stream-json` process logged into the owner's own account (no API key), with whisper.cpp for speech-to-text and Piper for speech. Keep it to the Python standard library. Notes are saved from `[[note: ...]]` tags in answers.
- The hub also runs on an old Android phone in Termux (`pi/android/`): Claude Code runs inside proot-distro Ubuntu through the `claude-proot` wrapper (Termux home bound at `/termux-home`; never over `/root`, which holds Claude Code), speech in and out use Termux:API (`engine = "android"`), `android/run.sh` replaces systemd, and the glasses join the phone's hotspot and find the hub as the Wi-Fi gateway. Keep both the Pi and Android paths working.
- `pi/phone.html` is the live phone page the hub serves at `/` (token in a cookie). Displays follow the hub's server-sent events at `/events` (`state`, `question`, `delta`, `answer`, `note`, `capture`, `glasses`, `problem`); the planned in-glasses display should read the same stream. Test the page with Playwright at phone and desktop sizes.
- `pi/panel.html` is the control panel at `/panel`, fed by `GET /api/health` (`Hub.health()`) and `POST /test/<speak|listen|camera|claude>`, plus the event stream. `app/` is the Android app (plain Java WebView, no AndroidX, like `mote/android`) that shows it, with an offline page that can start `run.sh` through Termux's RUN_COMMAND. `.github/workflows/lens-app.yml` builds `Lens.apk`, checks it on an emulator against a real hub, and releases it as `lens-app-<n>`.
- `pi/doctor.py` is the on-device repair agent: systemd runs it after repeated crashes, it lets Claude Code fix only `glasses/pi/`, keeps a fix only if `lens.py --selftest` passes, and commits it to a local `pi-fixes` branch. When the owner pushes `pi-fixes`, review it and merge what's right.
- Test the hub anywhere with Claude Code installed: a `config.toml` with `[ears] engine = "none"`, `[mouth] engine = "none"`, `[eyes] source = "file:<some.jpg>"`, then `lens.py --console` or `POST /ask`. `lens.py --selftest` must pass.
- `tools/screen_glasses.py` is a stand-in for the glasses (screen capture as `/capture`, `/hello`, keyboard as the button) for testing on a computer; keep it speaking the same HTTP API as the firmware.
- `hardware/frame.py` is the parametric 3D model (manifold3d): it writes print-oriented STLs to `hardware/stl/` and `hardware/preview/model.json` for the 3D view in `hardware/blueprint.html` (published as an artifact). Regenerate both after changing it, keep every part printable without supports, and keep the blueprint's wiring in step with the firmware pins (LED on D0, button on D1).
- `index.html` is the phone-only version: it calls the Claude API from the browser with the user's own API key through the Anthropic SDK from jsDelivr, pinned to an exact version. Keep phone (touch, portrait and landscape) and computer layouts working; test with Playwright with Chromium's fake camera flags (`--use-fake-device-for-media-stream --use-fake-ui-for-media-stream`) and the API mocked with `page.route`. `.github/workflows/glasses.yml` publishes it to GitHub Pages.
