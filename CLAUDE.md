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
- Lighting is a flood fill in `game/render.go` (levels 0-64) that also tracks which kind of light reached each cell; `game/lighting.go` turns that into a blurred colour light map multiplied over the world, with glowing things on an emissive layer and a soft haze around lights. New light-giving blocks go in `blockLight()` in `world.go` and get a colour through `blockLightSrc()`/`lightTints`.
- New block types: add an id, a colour in `blockRGB`, `isSolid`/`blocksLight` handling and a `blockItem` entry in `world.go`, and sim handling in `sim.go` if they move or grow.
- The game is a sandbox like Terraria/Starbound (not a roguelite). Each world is a save in `worlds/<id>.json` with a game mode (Easy, Normal, Hard, Hardcore; see `worlds.go`). New world or character state must go in `serializeState`/`restoreState` in `save.go`.
- Workstations are background objects (`stations.go`), not blocks: they don't collide, have a dark outline, open their own crafting menu, and give back one item when mined. Recipes say which station they need (`at` in `items.go`).
- Building tools (shapes, walls, paint, rotate/flip, holograms) are in `build.go`; keep the build bar usable on phones.
- Every push to `mote/` builds and releases `Mote.exe`, `Mote-steam.exe` and `Mote.apk` through `.github/workflows/mote.yml`.
- Check layouts by running under Xvfb with the test hooks in `game/testhooks.go` (`MOTE_SCRIPT`, `MOTE_TOUCH=1`, `MOTE_NOAUDIO=1`).

## blobsplit/ (the game "Blob Split": Android / Google Play)

- An idle split-and-sell game in Go + Ebitengine, landscape, made for phones. See `blobsplit/README.md`.
- Balance (rarities, costs, rebirth) is in `game/econ.go`; new saved state goes in `State` in `game/save.go`.
- UI reuses Mote's pixel style: `bevel()` boxes with skins in `game/ui.go`; all text is Titan One (the user disliked Pixelify Sans/Silkscreen here). Layout is in units where the screen is 360 tall (`lay()` in `game/draw.go`); check it at phone (800x360), tablet (1024x768) and PC sizes with the test hooks in `game/testhooks.go` under Xvfb.
- Every push to `blobsplit/` builds `BlobSplit.apk` and `BlobSplit.aab` through `.github/workflows/blobsplit.yml` and releases them as `blobsplit-<n>` (never marked latest, because Mote's updaters read the latest release).
