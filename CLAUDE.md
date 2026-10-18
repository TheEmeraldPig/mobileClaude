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
- The game is a roguelite: per-run state lives in `save.json` (deleted when a run ends), permanent upgrades in `meta.json` (`roguelite.go`). New per-run state must go in `serializeState`/`restoreState` in `save.go`.
- Every push to `mote/` builds and releases `Mote.exe`, `Mote-steam.exe` and `Mote.apk` through `.github/workflows/mote.yml`.
- Check layouts by running under Xvfb with the test hooks in `game/testhooks.go` (`MOTE_SCRIPT`, `MOTE_TOUCH=1`, `MOTE_NOAUDIO=1`).

## kitchen/ (the game "Kitchen Clash")

- A portrait idle/tap battler for Android: chefs fight rotten food Clicker Heroes style (one big monster, bosses with timers, recruit boxes, a shop), and the monsters drop ingredients that the kitchen half cooks into buffs and combos for the fight. Go + Ebitengine like `mote/`. See `kitchen/README.md`. The screen is 540 wide; height follows the phone. Every push to `kitchen/` builds `KitchenClash.apk` and `KitchenClash.exe` through `.github/workflows/kitchen.yml`.
- **After every change to `kitchen/`, rebuild the web version and republish it** so the user can play it from the Claude app: run `kitchen/web/build.sh`, then publish `kitchen/web/out/index.html` with the Artifact tool to `url` https://claude.ai/artifact/Kn58gXi11btKxoKsDA1bNQ, with `root` `kitchen/web/out` and `files` `wasm_exec.js` and `kitchen.gz.wasm`. Give the user the link in the reply.
- `xvfb-run -a go test ./game` must pass. Check layouts under Xvfb with a portrait screen (`-screen 0 720x1280x24`), `KITCHEN_SCALE=2` and `KITCHEN_SCRIPT` (see `game/testhooks.go`).
- The style is Clicker Heroes-like hand-inked cartoon: thick near-black outlines, cel shading (one hard shadow tone, hard highlights), bright food colours, goofy grumpy monsters, Lilita One titles with dark outlines. All art is painted once with the signed-distance painter in `game/paint.go` (`sh` shapes, `canvas.part`/`vol`/`volOn`) and cached through `paintArt`; don't draw characters or UI with flat vector shapes. `KITCHEN_PREVIEW=dir go test ./game -run Preview` makes a sheet of every chef, monster and food.
- New characters are painted in layers (body, face states) and animated by transforms in `draw.go` (bobbing, drop-in and squash, hit squish and flash, throws). Keep everything moving: hits flash, coins and ingredients fly to their counters, banners pop, buttons press.
- New recipes, combos and effects go in `food.go`; their rules (effects, cooking, the table) live in `profile.go`. New per-save state goes in `Profile` and must be repaired in `fix()`.
- Drawing goes through `S` (layout units to pixels): `fillRect`, `gradRect`, `clip`, `pimg.draw`. Keep `AntiAlias` off in any `vector.*` call.
