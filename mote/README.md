# Mote (native)

The native version of Mote, built with Go and [Ebitengine](https://ebitengine.org). There's no browser engine inside. It runs on Windows (`Mote.exe`) and Android (`Mote.apk`), and can be shipped on Steam.

## Downloads

Every push that changes `mote/` builds both through `.github/workflows/mote.yml` and publishes a GitHub release named `build-<number>`, containing:

- `Mote.exe`, for Windows. Run it; there's nothing to install. It checks for new releases every minute and restarts itself into the new version, keeping your save.
- `Mote.apk`, for Android. Open it on the phone to install. Each build installs over the last one and keeps the save. From this version on, the app checks for new builds when it starts and every 5 minutes, downloads them itself and opens Android's install screen, so updating is one tap (`android/.../Updater.java`). Android never lets a sideloaded app update silently; the very first time it also asks you to allow Mote to install apps.
- `Mote-steam.exe`, the Steam build, with the built-in updater switched off because Steam does the updating.

Each world is its own save in `worlds/<id>.json`, listed in `worlds.json`, and holds the terrain, your character and inventory. A world's game mode is chosen when it is made: Easy (less damage, keep everything when you die), Normal (drop blocks and materials, keep tools and gear), Hard (more damage, drop everything, respawn with a temporary kit) and Hardcore (like Hard, but dying deletes the world). Saves and settings live in `%AppData%\Mote` on Windows and in the app's private storage on Android. A `save.json` from older versions is turned into a Normal world.

## Playing

| | PC | Phone |
|---|---|---|
| Move / levitate | A, D / W (or arrows, Space) | Joystick; push it up to levitate |
| Use the selected item | Click the world | Touch the world |
| Pick an item | 1-8 or the scroll wheel | Tap a hotbar slot |
| Inventory / crafting | E / C | Bag / hammer buttons |
| Use a station, chest, bed or door | Right-click it | Tap it |
| Build shape, size | B, then - and = | Build bar above the hotbar |
| Rotate / flip a blueprint or station | R / F | Rotate / Flip on the build bar |
| Paint colour | C while holding the paintbrush | Colour on the build bar |
| Undo building | Ctrl+Z | Undo on the build bar |
| Mine background walls | Hold right-click with a pickaxe | Mining: Back walls on the build bar |
| Talk to a trader | Right-click them | Tap them |
| Guide book | Pause menu | Pause menu |

Hand crafting covers planks, torches and the workbench. Everything else is made by opening a station: the workbench (tools, blocks, walls, stations, blueprints), the furnace (bars, glass, bread) and the anvil (metal tools, weapons, armor).

## Layout

- `game/` holds the whole game as a library:
  - `world.go`: seeded chunk world: terrain, caves, caverns, ores and crystals. Follow the rules at the top of the file when adding terrain. The world is `H` = 1600 cells tall; the sky and the land's shape use the top `topH` = 600 rows, so everything below is underground. Saves from before record no height and are deepened on load (`deepenSaved`).
  - `render.go`: the light flood fill (fog of war), the day/night sky and background, and drawing the world at full brightness. Light carries about 85 cells through air and about 40 into solid ground, fading gradually.
  - `lighting.go`: coloured lighting: a blurred RGB light map multiplied over the world (linear filtering for smooth falloff), an emissive layer for things that glow, coloured haze around lights and a vignette.
  - `mining.go`: block hardness, pickaxe tiers and power, crack wear, and mining background walls. With no pickaxe at all, an empty hand digs slowly, and a wooden pickaxe can be crafted by hand from 8 wood, so the mote can never get stuck.
  - `held.go`: draws the item in the mote's hand while it is used (pickaxes swing, swords slash, blocks bob). Swords hit along a slash from the mote to the aim, as far as a pickaxe reaches; better swords only do more damage.
  - `entities.go`: creatures (gloomlings, bats, skitters, wisps, brutes, night shades, the Warden boss) and dropped items. Each kind's size, health, damage, speed and sensing range is in the `kinds` table; every speed is a fraction under 1 of the mote's, so the mote can always outrun them. Spawning is slow (`spawnEvery`, `maxCreatures`).
  - `ai.go`: creature path finding. A breadth-first search over 4x4-cell blocks spreads out from the mote every few frames; creatures step toward the neighbouring block that is closer, so they go around walls and through tunnels. Walkers can only climb where a jump reaches.
  - `blueprints.go`: ready-made structures and the placement hologram. Add a structure by adding a template in `initBlueprints()`, an item and a recipe.
  - `worlds.go`: the world list, game modes (`modeDamage`, death drops in `die()`), respawning and the temporary kit.
  - `stations.go`: workstations (workbench, furnace, anvil). They are background objects kept in `objs` (not blocks): walk past them, tap or right-click to open their crafting menu, hold a pickaxe on one to pick it up. Add one to `stations` with a sprite, plus an item with `station:` set.
  - `structures.go`: big generated structures: sky islands with shrines, underground ruins, desert pyramids and surface villages (whose houses hold traders). They also get background walls through `genWalls`, which re-runs the structure passes into a wall buffer when a chunk without saved walls loads.
  - `traders.go`: village traders (motes in other colours with hats) and their shops: stock and prices in `traderKinds`, what they pay in `sellValue`. Coins drop from creatures and are in chests.
  - `book.go`: the guide book (pause menu): chapters and pages in `book`.
  - `doors.go`: doors (a 3x14 group of `DOOR`/`DOOROPEN` blocks): tap to open and close, creatures can't pass a closed one.
  - `storage.go`: the storage chest screen (a station whose `items` hold 24 stacks).
  - `craft.go`: the crafting screen: category tabs, a row of recipes and a details panel. Recipes live in `recipes` in `items.go`; `at` names the station ("" = by hand).
  - `build.go`: advanced building: shapes (brush, line, box, filled box, circle), size, rotate/flip, paint colours, background walls, the build bar and the holograms.
  - `sim.go`: sand and water.
  - `player.go`: the mote, its physics and sprite.
  - `items.go`: items, inventory, recipes and item use.
  - `menus.go` and `ui.go`: pixel UI and screens.
  - `input.go`: keyboard, mouse, touch and joystick.
  - `audio.go`: synthesised sound effects and music.
  - `save.go`: settings and save game. Add new state to `serializeState`/`restoreState`. Besides blocks, each chunk can save a wall layer and two paint layers (`wall`, `paint`, `wpaint` in `world.go`).
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
