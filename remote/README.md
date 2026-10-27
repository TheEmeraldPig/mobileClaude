# Remote

A pixel MMO built on a copy of Mote's engine (Go + [Ebitengine](https://ebitengine.org)): Mote's lighting, terrain, physics and pixel UI, turned into an online game that mixes [florr.io](https://florr.io) and Legends of IdleOn, with islands to travel between like Starbound.

- **Petals.** Your mote has petals circling it. Hold attack to spread them out and hit creatures, hold defend to pull them in. Petals break when they take hits and grow back. Kinds: Spark, Pebble, Thorn, Leaf (heals), Glow (lights caves), Ember (burns), Frost (slows), Heart (heals you when you're hurt).
- **Rarity.** Creatures and petals come in Common, Unusual, Rare, Epic, Legendary and Mythic. Each tier has three times the damage and health. Creatures drop petals of their own rarity or one lower, and five of the same petal combine into one of the next rarity (for coins).
- **Islands.** Haven is the town everyone shares (no creatures). Greenreach (level 1), Glimmer Hollow (level 8, dark crystal caves) and Cinder Isle (level 18, lava, with a Warden boss every 5 minutes) have creatures. Everyone on an island shares it with everyone else there.
- **Levels.** Kills give XP and coins. Levels add health, and petal slots at levels 8, 16 and 25.
- **Away gains** (like IdleOn). Log off on a combat island and your mote keeps fighting there for up to 12 hours. When you come back, a screen shows what it earned.
- **Emotes.** Players can say Hi, <3, Haha, GG, Help! or Follow me.
- **Play solo.** The game can run the server inside itself, so it works with no internet (solo characters are separate from online ones).

`remote/` is a **copy** of `mote/`, not a shared package, so Remote can change anything without touching Mote. Nothing here may clash with Mote on the same device or in the same repository:

| | Mote | Remote |
|---|---|---|
| Go module | `.../mobileClaude/mote` | `.../mobileClaude/remote` |
| Save and settings folder | `%AppData%\Mote` | `%AppData%\Remote` |
| Android app id | `com.theemeraldpig.mote` | `com.theemeraldpig.remote` |
| Release tags and files | `build-N`, `Mote.exe`/`Mote.apk`, marked latest | `remote-build-N`, `Remote.exe`/`Remote.apk`, **never** marked latest |
| Updaters | read `/releases/latest` | list the releases and pick the newest `remote-build-N` (`game/update_windows.go`, `android/.../Updater.java`) |
| Online server | `mote/relay` (`render.yaml` at the repo root) | `remote/server` (`remote/render.yaml`) |
| Test env vars | `MOTE_SCRIPT`, ... | `REMOTE_SCRIPT`, `REMOTE_SIZE`, `REMOTE_TOUCH`, `REMOTE_NOAUDIO` |
| CI | `.github/workflows/mote.yml` | `.github/workflows/remote.yml` |

## Downloads

Every push that changes `remote/` builds through `.github/workflows/remote.yml` and publishes a GitHub release named `remote-build-<number>` with `Remote.exe`, `Remote-steam.exe` and `Remote.apk`. Each build installs over the last one. Remote's releases are never marked "latest", because Mote's updaters install whatever "latest" is.

## Playing

| | PC | Phone |
|---|---|---|
| Move / levitate | A, D / W (or arrows, Space) | Joystick; push it up to levitate |
| Attack (petals out) | Hold the left mouse button | Hold the sword button |
| Defend (petals in) | Hold the right mouse button | Hold the shield button |
| Petals (equip, combine) | E | Bag button, or tap the petal bar |
| Travel | M | Map button |
| Emotes | 1-6 | Speech button |
| Menu | Esc | Pause button |

## Layout

- `mmo/`: shared by the game and the server, with no graphics:
  - `isles.go`: the islands and their generation. Follow the rules at the top of the file: no `math/rand`, seeded hashes only. `isles_test.go` pins a checksum per island.
  - `defs.go`: rarities, petals, creatures, levels, orbit sizes.
  - `char.go`: characters (level, coins, bag, loadout), combining, and away gains.
  - `proto.go`: the messages (JSON over a WebSocket).
- `srv/`: the game server. It owns characters, runs every island's creatures (AI and physics on the island's cells) and every fight (petals hitting creatures, creatures touching motes, lava), and hands out loot. 20 steps a second, snapshots 10 times a second. Characters are saved as JSON in `DataDir/accounts`. The game also keeps a signed copy (HMAC with `REMOTE_SECRET`), which the server takes back if its disk was wiped.
- `server/`: the server program and its Dockerfile.
- `game/`: the game (Mote's engine):
  - `client.go`: connecting, signing in, the messages, and solo play (the server running inside the game).
  - `isle.go`: loads an island into the grid the renderer draws.
  - `others.go`: other players, creatures from the server, and drawing petals.
  - `hud.go`, `screens.go`, `menus.go`, `ui.go`: the HUD and screens (main menu, name, petals, travel, knocked out, away gains, settings).
  - `render.go`, `lighting.go`: Mote's light flood fill, coloured light map, sky and blocks. Petals and creatures give light (`petalLights`, `entityLights`).
  - `player.go`: the mote's physics and sprite. The game moves its own mote and tells the server where it is; the server won't accept moves faster than a mote can go.
  - `testhooks.go`: scripted runs for tests.

## The server

See [`server/README.md`](server/README.md) for running it and putting it online on Render.

## Building and testing

```sh
cd remote
go test ./mmo ./srv && xvfb-run -a go test ./game
go run ./server                                  # a local server on :8080
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-H windowsgui" -o Remote.exe ./cmd/remote
```

Scripted runs under Xvfb (write `settings.json` with `{"fullscreen":false}` in the fake HOME's `.config/Remote` first):

```sh
HOME=/tmp/h DISPLAY=:99 REMOTE_NOAUDIO=1 REMOTE_SIZE=1280x720 \
  REMOTE_SCRIPT="30:solo,150:travel=meadow,300:tpmob,302:attack,400:shot=/tmp/a.png,420:state,430:quit" ./remote
# phone layout: add REMOTE_TOUCH=1 and REMOTE_SIZE=1200x560
# against a server: 10:server=ws://127.0.0.1:8080,20:online
```

## Steam

`Remote-steam.exe` is the Steam build (no built-in updater). Accounts are a secret key kept on each device for now. Signing in with the Steam profile means validating a Steam session ticket on the server and using the Steam id as the account id. That needs the game's Steamworks app id and publisher key, so it waits until the game has a Steam page.
