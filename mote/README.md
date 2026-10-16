# Mote (native)

The native version of Mote, built with Go and [Ebitengine](https://ebitengine.org). There's no browser engine inside. It runs on Windows (`Mote.exe`) and Android (`Mote.apk`), and can be shipped on Steam.

## Downloads

Every push that changes `mote/` builds both through `.github/workflows/mote.yml` and publishes a GitHub release named `build-<number>`, containing:

- `Mote.exe`, for Windows. Run it; there's nothing to install. It checks for new releases every minute and restarts itself into the new version, keeping your save.
- `Mote.apk`, for Android. Open it on the phone to install. Each build installs over the last one and keeps the save.
- `Mote-steam.exe`, the Steam build, with the built-in updater switched off because Steam does the updating.

Runs are roguelite: fainting ends the run and deletes `save.json`, while Essence and upgrades in `meta.json` carry over. Saves and settings live in `%AppData%\Mote` on Windows and in the app's private storage on Android.

## Layout

- `game/` holds the whole game as a library:
  - `world.go`: seeded chunk world: terrain, caves, caverns, ores and crystals. Follow the rules at the top of the file when adding terrain.
  - `render.go`: lighting (fog of war), the day/night sky and background, and drawing the world.
  - `entities.go`: creatures (gloomlings, bats, skitters, wisps, brutes, night shades, the Warden boss) and dropped items. Each kind's size, health, damage, speed and sensing range is in the `kinds` table; every speed is a fraction under 1 of the mote's, so the mote can always outrun them. Spawning is slow (`spawnEvery`, `maxCreatures`).
  - `ai.go`: creature path finding. A breadth-first search over 4x4-cell blocks spreads out from the mote every few frames; creatures step toward the neighbouring block that is closer, so they go around walls and through tunnels. Walkers can only climb where a jump reaches.
  - `blueprints.go`: ready-made structures and the placement hologram. Add a structure by adding a template in `initBlueprints()`, an item and a recipe.
  - `roguelite.go`: runs, Essence, permanent upgrades (`upgrades` list) and the run-over and upgrade screens. Upgrades are stored in `meta.json`.
  - `sim.go`: sand and water.
  - `player.go`: the mote, its physics and sprite.
  - `items.go`: items, inventory, crafting and item use.
  - `menus.go` and `ui.go`: pixel UI and screens.
  - `input.go`: keyboard, mouse, touch and joystick.
  - `audio.go`: synthesised sound effects and music.
  - `save.go`: settings and save game. Add new state to `serializeState`/`restoreState`.
  - `update_windows.go`: the .exe self-updater (left out with `-tags steam`).
- `cmd/mote/` is the desktop entry point, with the Windows icon and manifest in `winres/`.
- `mobile/` is the Android entry point, bound into an .aar by `ebitenmobile`.
- `android/` is the Android app project that wraps the .aar.

## Building by hand

```sh
cd mote
# Windows .exe (from any OS)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-H windowsgui" -o Mote.exe ./cmd/mote
# Steam build
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags steam -ldflags "-H windowsgui" -o Mote.exe ./cmd/mote
# Tests (need a display; use xvfb-run on a headless Linux)
go test ./game
```

## Steam

Upload `Mote-steam.exe` as the game's executable through Steamworks (SteamPipe). Steam handles updates for that build. Achievements and other Steam features can be added later with `github.com/hajimehoshi/go-steamworks`.

## Seeds

`game/world_test.go` checks that seeds always build exactly the same chunks. When world generation changes on purpose, update the reference numbers in that test. This version no longer builds the same worlds as the web version in `sandbox/`.
