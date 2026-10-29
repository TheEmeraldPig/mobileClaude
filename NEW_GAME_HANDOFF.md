# Handoff: a new game built on Mote's engine

> **Done once:** Desalote (`desalote/`, a multiplayer card game) was built this way. Its README and
> workflow show the renames, the release-tag prefix and the updaters in practice.

You're starting a **new, separate game** in this repository (`TheEmeraldPig/mobileClaude`). It reuses
the engine of **Mote**, a pixel sandbox game in `mote/`. Mote is finished and must keep working
exactly as it is. The owner wants "something more exciting" built on the same lighting and game
systems. **Ask them what kind of game they want before building** (see "First steps").

Read `CLAUDE.md` (repo root) and `mote/README.md` first. They describe Mote's layout and rules.

---

## Ground rules

1. **Don't change `mote/`, `sandbox/`, `render.yaml` or `.github/workflows/mote.yml`.** Every push
   to `mote/` rebuilds and releases Mote to its players.
2. **Build the new game in its own folder** (e.g. `newgame/`, named after the game) by **copying**
   `mote/`, not by importing from it. A shared engine package would mean editing Mote. Copying is
   safe and lets the new game change anything.
3. **Make sure the two games can't collide** (details below): Android app id, save folder, window
   title, release tags, the auto-updaters, and the multiplayer ports.
4. Same targets as Mote: **Windows .exe and Android .apk**, played mostly on a phone in landscape.
   Keep both touch and keyboard/mouse layouts working.

---

## First steps

1. Ask the owner about the game: genre, what makes it exciting, single or multiplayer, phone-first?
   Offer 2-3 concrete ideas that suit this engine. For example:
   - a fast lit action-platformer through dark caves, where light is your weapon and lifeline
   - a co-op survival game against waves at night
   - a dungeon-diving roguelite with seeded runs
   - a physics-sand puzzle game
2. Copy the engine and rename everything (next section). Build and run it unchanged before changing
   gameplay.
3. Strip out what the new game doesn't need (traders, farming, blueprints, etc.) and build the new
   game on what's left.

## Copying and renaming (avoid collisions)

```sh
cp -r mote newgame
cd newgame
# Go module path, used in every import
sed -i 's#github.com/TheEmeraldPig/mobileClaude/mote#github.com/TheEmeraldPig/mobileClaude/newgame#' go.mod
grep -rl 'mobileClaude/mote' --include=*.go . | xargs sed -i 's#mobileClaude/mote/#mobileClaude/newgame/#g'
rm -rf relay   # the optional online relay; keep relaycore/ (multiplayer needs it)
```

Then change each of these. Any one left as Mote's will clash with Mote on the same device:

| What | Where in Mote | Why |
|---|---|---|
| Save/settings folder `"Mote"` | `game/save.go` (`dataPath`) | Otherwise the new game reads and overwrites Mote's worlds |
| Window title `"Mote"` | `cmd/mote/main.go` (rename the folder too) | |
| Android `applicationId`/`namespace` `com.theemeraldpig.mote` | `android/app/build.gradle`, the Java package folder, `-javapkg` in the workflow | Same id = installing one replaces the other |
| App name/icon | `android/app/src/main/AndroidManifest.xml`, `res/`, `cmd/.../winres` | |
| Release tags `build-N` and asset names `Mote.exe`/`Mote.apk` | the workflow | See updaters below |
| **Auto-updaters** | `game/update_windows.go` (`releaseAPI`), `android/.../Updater.java` (`LATEST`) | Both read `/releases/latest` of this repo |
| Multiplayer ports 47469/47470 | `game/netlan.go` | Otherwise each game's world list shows the other's worlds |
| Test env vars `MOTE_SCRIPT`, `MOTE_SIZE`, `MOTE_TOUCH`, `MOTE_NOAUDIO` | `game/testhooks.go`, `game/audio.go` | Optional, but clearer |

**The release and updater rule is critical.** GitHub has one "latest" release per repo, and Mote's
Windows and Android updaters install whatever "latest" is. The new game's workflow must create its
releases with **`gh release create ... --latest=false`** and a different tag prefix
(e.g. `newgame-build-N`). Its own updaters must **list releases**
(`/repos/TheEmeraldPig/mobileClaude/releases`) and pick the newest tag with its own prefix, instead
of using `/releases/latest`. If this is wrong, Mote players get offered the new game as an update.

**CI:** copy `.github/workflows/mote.yml` to a new workflow file. Change:
- the `paths:` filter to `newgame/**`
- the working directories
- the `-ldflags -X` package paths
- the javapkg
- the release step (tag prefix, asset names, `--latest=false`)

The Android signing keystore in `mote/android/` can be reused.

---

## What the engine gives you (in `game/`)

- **Lighting** (`render.go` `computeLight`, `lighting.go`):
  - A flood fill on a 0-1023 scale. Light travels about 85 cells through air and about 40 into
    solid blocks.
  - Directional sunlight with soft shadows (`sunLight`, `sunLit`), plus ambient sky light.
  - Each cell remembers which kind of light reached it. That becomes a blurred RGB light map,
    multiplied over the world, with bloom/haze, an emissive layer for glowing things (`emit()`),
    and a vignette.
  - Colours per light type are in `lightTints`, and the falloff curve is `lightCurve`. Light-giving
    blocks go in `blockLight()`/`blockLightSrc()`, and moving lights in `entityLights()`.
  - Keep the test in `light_test.go`: light must not change when the camera moves (that was a
    flicker bug).
- **Look** (`render.go`):
  - Clumpy per-cell textures (`texShade`, in `world.go`), per-material detail (`blockTexture`),
    bevelled edges,
    cave ambient occlusion, grass blades.
  - Kyora-style sky: shaded clouds, misty parallax hills, sun and moon, day/night.
- **World** (`world.go`, `structures.go`):
  - Seeded, endless sideways, 1600 cells tall, loaded in 128-wide chunks.
  - Generation passes in `genPasses`, Kyora-style big trees, caves and caverns, ore clumps, and
    structures (villages, ruins, pyramids, sky islands).
  - Rules: no `math/rand` in generation, features crossing chunk borders must check neighbours,
    and `world_test.go` pins checksums. Read the rules at the top of `world.go`.
- **Simulation** (`sim.go`): falling sand, flowing water and lava, crops.
- **Player** (`player.go`): the mote, with levitation energy, health and squash/stretch.
- **Creatures** (`entities.go`, `ai.go`): several kinds with flow-field pathfinding.
- **Items and combat** (`items.go`, `mining.go`, `held.go`): mining with hardness and tool tiers,
  swords that hit along a slash, bombs, and held-item animations.
- **Building** (`build.go`, `blueprints.go`, `stations.go`, `doors.go`, `storage.go`).
- **UI** (`ui.go`, `menus.go`, `craft.go`): immediate-mode buttons, sheets, labels and text, in the
  pixel-bevel style of the mote sprite, with phone and PC layouts.
- **Saves and modes** (`save.go`, `worlds.go`).
- **Audio** (`audio.go`): synthesised effects and music, with no asset files.
- **Multiplayer**:
  - `net.go`, `netsync.go`, `netmenu.go` and `netlan.go`, plus the message hub in `relaycore/`.
  - Up to 4 players on the same Wi-Fi. The host's game runs the hub and announces its world by UDP
    broadcast, and players can also join by address.
  - Only world changes travel over the network (terrain comes from the seed). Creatures belong to
    the game that spawned them.
- **Performance** (`lighting.go` `checkPerformance`): an automatic lighter mode on slow devices.

## How to test

Run the game under Xvfb and script it with the test hooks in `game/testhooks.go`:

```sh
Xvfb :99 -screen 0 1400x900x24 &
cd newgame && go build -o /tmp/g ./cmd/<name>
HOME=/tmp/h DISPLAY=:99 MOTE_NOAUDIO=1 MOTE_SIZE=1280x720 \
  MOTE_SCRIPT="5:newworld=seed:1,60:shot=/tmp/a.png,90:fps,100:quit" /tmp/g
# phone layout: add MOTE_TOUCH=1 and MOTE_SIZE=1200x560
xvfb-run -a go test ./game
```

Hooks include `newworld`, `give`, `selectid`, `aim`/`unaim`, `shot=PATH`, `state`, `deep`,
`time=0.5`, `foe=KIND`, `tpstruct=...`, `host`/`joinlan`/`net`, and `fps`. Write
`settings.json` with `{"fullscreen":false}` in the fake HOME's config folder first. Always look at
screenshots in both layouts. Check FPS: the test machine has no GPU, so about 10-15 FPS there
is normal for Mote.

## Lessons learned building Mote (avoid repeating them)

- Go's `%` keeps the sign. Wrap indices like `((n%k)+k)%k` (a negative village id crashed the
  game).
- Bounds-check every `grid[...]` neighbour lookup, especially below the world's bottom row
  (`wy < H`).
- Never call noise functions per pixel per frame. Precompute them when a chunk loads, or once per
  frame or column.
- Draw rectangles with the stretched white pixel in `fill` (`ui.go`), not vector paths.
- Light must be computed in world coordinates, every frame. Skipping frames made it stutter.
- When creatures die mid-update, keep anything appended to `ents` during the loop. Loot used to
  vanish this way.
- Android: the minimum API is 23 (e.g. use `getContentLength`, not `getContentLengthLong`). Newer
  Android hides the network interface list (see `localIPs` in `netlan.go`).
- After a deliberate world-generation change, update the `world_test.go` numbers and say so.
- The owner prefers being asked when a design choice is theirs, and otherwise wants polished,
  tested work pushed to the branch and checked in CI (including the Android emulator job's
  screenshots in the logs).
