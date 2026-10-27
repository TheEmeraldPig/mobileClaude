# Remote

A new game built on a copy of Mote's engine (Go + [Ebitengine](https://ebitengine.org)). It is planned as an online multiplayer game (MMO) that mixes Legends of IdleOn and an io game, with Starbound-style islands to travel between. For now it is still Mote's game under a new name, as the starting point.

`remote/` is a **copy** of `mote/`, not a shared package, so Remote can change anything without touching Mote. Nothing here may clash with Mote on the same device or in the same repository:

| | Mote | Remote |
|---|---|---|
| Go module | `.../mobileClaude/mote` | `.../mobileClaude/remote` |
| Save and settings folder | `%AppData%\Mote` | `%AppData%\Remote` |
| Android app id | `com.theemeraldpig.mote` | `com.theemeraldpig.remote` |
| Release tags and files | `build-N`, `Mote.exe`/`Mote.apk`, marked latest | `remote-build-N`, `Remote.exe`/`Remote.apk`, **never** marked latest |
| Updaters | read `/releases/latest` | list the releases and pick the newest `remote-build-N` (`game/update_windows.go`, `android/.../Updater.java`) |
| Wi-Fi multiplayer ports | 47469/47470 | 47569/47570, and announcements carry `"remote":1` |
| Test env vars | `MOTE_SCRIPT`, ... | `REMOTE_SCRIPT`, `REMOTE_SIZE`, `REMOTE_TOUCH`, `REMOTE_NOAUDIO`, `REMOTE_RESUME` |
| CI | `.github/workflows/mote.yml` | `.github/workflows/remote.yml` |

## Downloads

Every push that changes `remote/` builds through `.github/workflows/remote.yml` and publishes a GitHub release named `remote-build-<number>` with `Remote.exe`, `Remote-steam.exe` and `Remote.apk`. Each build installs over the last one and keeps the saves. Remote's releases are never marked "latest", because Mote's updaters install whatever "latest" is.

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
| Play with friends (same Wi-Fi or hotspot) | Main menu: Multiplayer; or Pause: Open to friends | Same |

Hand crafting covers planks, torches and the workbench. Everything else is made by opening a station: the workbench (tools, blocks, walls, stations, blueprints), the furnace (bars, glass, bread) and the anvil (metal tools, weapons, armor).

## Layout

- `game/` holds the whole game as a library:
  - `world.go`: seeded chunk world: terrain, caves, caverns, ores and crystals. Follow the rules at the top of the file when adding terrain. The world is `H` = 1600 cells tall; the sky and the land's shape use the top `topH` = 600 rows, so everything below is underground. Saves from before record no height and are deepened on load (`deepenSaved`). Trees (`plantTrees`/`growTree`) are big Kyora-style trees with roots, curving branches and ragged leaf clumps; they can spread across chunk borders.
  - Block look (`render.go`): `texShade` gives each cell soft clumpy shading when its chunk loads; `blockTexture` adds per-material detail (pebbles, grass roots, bark, lit leaves), solid blocks get bevelled edges, cave back walls darken next to rock, and grass blades are drawn above grass. Keep per-pixel drawing free of noise calls (it runs for every cell on screen each frame).
  - Performance: the Graphics setting is Auto, Fancy or Fast. On Auto, `checkPerformance` (lighting.go) switches to the lighter mode when the frame rate stays under ~42: the light blur is lighter, clouds update less often, and haze and vignette are skipped. Per-frame constants (sky colours, hill outlines, light colours per level) are worked out once a frame, not per pixel, and `fill` draws rectangles as a stretched white pixel (batched) rather than vector paths.
  - `render.go`: the light flood fill (fog of war), the day/night sky and background, and drawing the world at full brightness. Light carries about 85 cells through air and about 40 into solid ground, fading gradually. Daylight comes from the sun's (or moon's) position: direct light follows slanted rays in world space (`sunLight`; `sunLit` holds how much of the sun reaches each cell, 0-1, so shadow edges are soft; `light_test.go` checks the light doesn't change when the camera moves), so terrain casts shadows that move through the day, while open sky adds dimmer, bluer light straight down.
  - `lighting.go`: coloured lighting in the style of Kyora: a blurred RGB light map multiplied over the world (linear filtering for smooth falloff). Daylight cools to blue in the shade, firelight reddens at the edge of its glow, and darkness is a deep violet-blue rather than black. An emissive layer for things that glow, a wide soft bloom around lights and a vignette. The sky (`render.go`) has shaded, drifting clouds and misty distant hills.
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
  - Multiplayer (`net.go`, `netsync.go`, `netmenu.go`, `netlan.go`): up to 4 players in a host's world, with no server: the hosting game runs the message hub (`relaycore/`) on port 47570 and announces its world by UDP broadcast on port 47569 every 0.7 s; games on the Multiplayer screen list what they hear and connect straight to the host, or players type the host's address (which also works over virtual-LAN apps like Tailscale/ZeroTier). Optional: the same hub can run online for join codes (Mote's `mote/relay/`; CI builds its address in from the `REMOTE_RELAY_URL` repository variable, and the code box only appears when one is set). Terrain comes from the seed, so only changes travel: a joining guest gets the host's changed chunks, then each game sends the cells its own player changed (found by comparing a box around the player before and after their actions each frame). Creatures and dropped items belong to the game that spawned them; hits and pickups are sent to their owner. The host saves guests' characters in its world (`guests` in `save.go`). Falling sand and water run on each game separately.
- `cmd/remote/` is the desktop entry point, with the Windows icon and manifest in `winres/`.
- `mobile/` is the Android entry point, bound into an .aar by `ebitenmobile`.
- `android/` is the Android app project that wraps the .aar.

## Building by hand

```sh
cd remote
# Windows .exe (from any OS)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-H windowsgui" -o Remote.exe ./cmd/remote
# Steam build
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags steam -ldflags "-H windowsgui" -o Remote.exe ./cmd/remote
# Tests (need a display; use xvfb-run on a headless Linux)
go test ./game
```

## Steam

Upload `Remote-steam.exe` as the game's executable through Steamworks (SteamPipe). Steam handles updates for that build. Achievements and other Steam features can be added later with `github.com/hajimehoshi/go-steamworks`.

## Seeds

`game/world_test.go` checks that seeds always build exactly the same chunks. When world generation changes on purpose, update the reference numbers in that test. This version no longer builds the same worlds as the web version in `sandbox/`.
