# Mote (native)

The native version of Mote, built with Go and [Ebitengine](https://ebitengine.org). There's no browser engine inside. It runs on Windows (`Mote.exe`) and Android (`Mote.apk`), and can be shipped on Steam.

## Downloads

Every push that changes `mote/` builds both through `.github/workflows/mote.yml` and publishes a GitHub release named `build-<number>`, containing:

- `Mote.exe`, for Windows. Run it; there's nothing to install. It checks for new releases every minute and restarts itself into the new version, keeping your save.
- `Mote.apk`, for Android. Open it on the phone to install. Each build installs over the last one and keeps the save.
- `Mote-steam.exe`, the Steam build, with the built-in updater switched off because Steam does the updating.

Saves and settings live in `%AppData%\Mote` on Windows and in the app's private storage on Android.

## Layout

- `game/` holds the whole game as a library:
  - `world.go`: seeded chunk world. Follow the rules at the top of the file when adding terrain.
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

## Same worlds as the web version

`game/world_test.go` checks that seeds build exactly the same chunks as `sandbox/index.html`.
