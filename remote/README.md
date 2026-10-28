# Remote

A pixel MMO built on a copy of Mote's engine (Go + [Ebitengine](https://ebitengine.org)): Mote's lighting, terrain, physics and pixel UI, turned into an online game that mixes [florr.io](https://florr.io) and Legends of IdleOn, with islands to travel between like Starbound.

- **Characters.** Humanoids with alien touches, like Starbound's: big glossy eyes, human or alien skin tones, and antennae, pointed ears, horns or a head fin. Players pick their skin, eyes, feature, hair, hair colour, shirt and trousers on the Character screen. The look is saved with the character, so everyone sees it.
- **Moving.** Walk, jump, and hold jump in the air for a short alien hover.
- **Tools.** A sword (swings in an arc at what you aim at; damage grows with level) and a pickaxe.
- **Mining.** Haven and the combat islands can't be mined. Mining islands (Stonecrop for now) can: a dig wears a small patch of blocks into worn ones that stay put and can't be mined again, and gives you the resource (stone, dirt, sand, wood, coal, copper, iron, gold, crystal). The island keeps its shape for everyone, and worn blocks grow back after 10 minutes. Each island names its mining rules (`MineRules` in `mmo/mine.go`), so other kinds of island can give different things.
- **Creatures and rarity.** Creatures come in Common, Unusual, Rare, Epic, Legendary and Mythic, each tier much tougher. Kills give XP and coins; levels add health.
- **Islands.** Haven is the town everyone shares (no creatures). Greenreach (level 1), Glimmer Hollow (level 8, dark crystal caves) and Cinder Isle (level 18, lava, with a Warden boss every 5 minutes) have creatures. Stonecrop is for mining. Everyone on an island shares it with everyone else there.
- **Away gains** (like IdleOn). Log off on a combat island and your character keeps fighting there for up to 12 hours. When you come back, a screen shows what it earned.
- **Emotes.** Players can say Hi, <3, Haha, GG, Help! or Follow me.
- **Play solo.** The game can run the server inside itself, so it works with no internet (solo characters are separate from online ones).
- **Petals (switched off).** The florr.io-style petals (orbiting petals with rarities, combining five into one) are still in the code for later: set `mmo.PetalsOn` to bring them back.

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
| Move | A, D (or arrows) | Joystick |
| Jump / hover | W, Up or Space (hold in the air to hover) | Push the joystick up |
| Use the sword or pickaxe | Hold the left mouse button (aims at the mouse) | Hold the big button (aims itself); with the pickaxe, touching the world digs there |
| Switch tool | 1 / 2, Q or the scroll wheel | Small button, or the tool bar |
| Bag (resources) | E | Bag button |
| Travel | M | Map button |
| Emotes | 3-8 | Speech button |
| Menu | Esc | Pause button |

## Layout

- `mmo/`: shared by the game and the server, with no graphics:
  - `isles.go`: the islands and their generation. Follow the rules at the top of the file: no `math/rand`, seeded hashes only. `isles_test.go` pins a checksum per island.
  - `defs.go`: rarities, petals, creatures, levels, orbit sizes.
  - `mine.go`: mining rules per island, resources, the sword, and the `PetalsOn` switch.
  - `look.go`: character looks (skin tones, eyes, hair, features, clothes).
  - `char.go`: characters (level, coins, look, resources, petals), combining, and away gains.
  - `proto.go`: the messages (JSON over a WebSocket).
- `srv/`: the game server. It owns characters, runs every island's creatures (AI and physics on the island's cells) and every fight (sword swings, creatures touching players, lava), checks and applies mining (`srv/mine.go`: reach, digging time, worn blocks, regrowth), and hands out loot. 20 steps a second, snapshots 10 times a second. Characters are saved as JSON in `DataDir/accounts`. The game also keeps a signed copy (HMAC with `REMOTE_SECRET`), which the server takes back if its disk was wiped.
- `server/`: the server program and its Dockerfile.
- `game/`: the game (Mote's engine):
  - `client.go`: connecting, signing in, the messages, and solo play (the server running inside the game).
  - `isle.go`: loads an island into the grid the renderer draws.
  - `body.go`: characters, drawn in layers from their look, with outlines and walk frames, and the tool in hand.
  - `tools.go`: using the sword and pickaxe, aiming, digging progress, and block changes from the server.
  - `others.go`: other players, creatures from the server, and drawing petals (when they're on).
  - `hud.go`, `screens.go`, `menus.go`, `ui.go`: the HUD and screens (main menu, name, character, bag, travel, knocked out, away gains, settings).
  - `render.go`, `lighting.go`: Mote's light flood fill, coloured light map, sky and blocks. Petals and creatures give light (`petalLights`, `entityLights`).
  - `player.go`: the character's physics (walk, jump, hover). The game moves its own character and tells the server where it is; the server won't accept moves faster than a character can go.
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
  REMOTE_SCRIPT="30:solo,40:play,150:travel=quarry,300:tool=1,305:attack,400:shot=/tmp/a.png,420:state,430:quit" ./remote
# phone layout: add REMOTE_TOUCH=1 and REMOTE_SIZE=1200x560
# against a server: 10:server=ws://127.0.0.1:8080,20:online
```

## Steam

`Remote-steam.exe` is the Steam build (no built-in updater). Accounts are a secret key kept on each device for now. Signing in with the Steam profile means validating a Steam session ticket on the server and using the Steam id as the account id. That needs the game's Steamworks app id and publisher key, so it waits until the game has a Steam page.
