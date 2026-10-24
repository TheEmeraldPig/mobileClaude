# Biote

An idle mobile game made with Go and [Ebitengine](https://ebitengine.org), built for Android and Google Play. Tap **SPLIT** to double your biotes until the board is full, **SELL** them for money, and grow stronger through upgrades, rounds, ability cards, rebirths and a skill tree. (It started as "Blob Split"; the folder, package id `com.theemeraldpig.blobsplit` and save folder keep that name so updates install over it.)

## Playing

| | Phone | PC |
|---|---|---|
| Split | SPLIT button | Space |
| Sell | SELL button | S |
| Buy an upgrade | Tap a card on the right (hold to keep buying) | Click (hold to keep buying) |
| See a biote's mutations | Tap it | Click it |
| Close a menu | X, or tap outside it | Esc |

- **Mutations**: Gold 1/25 (x3), Shadow 1/60 (x6), Frost 1/150 (x15), Heart 1/400 (x40), Toxic 1/1K (x100), Rainbow 1/3K (x300), Cosmic 1/10K (x1000). Without luck a new biote gains a mutation 1 time in 15 (`mutationOdds`). Mutations stack: a biote can carry several, their multipliers multiply, and the label above it shows the chance of all of them together (`comboOdds`). A child always keeps its parent's mutations and can gain more. Tap a biote to see each mutation's chance and multiplier and its base value. The Catalog only shows mutations you have found.
- **Upgrades** (money, reset on rebirth): biote value, split speed (3s down to 2s) and capacity (4 up to 64). The BUY button sets x1, x5, x25 or MAX. Selling restarts the split timer. There is no money upgrade for luck: luck comes from skills, cards, the time of day and events.
- **Rounds**, like Balatro's blinds (`econ.go`, `game.go`): each ante has **Petri**, **Colony** and **Outbreak** (the boss, with a twist from `bosses` in `mods.go`: The Parasite, The Drought, The Freeze, The Leech, The Mirror). Earn the target within 4 sells; running out restarts the round. Clearing a round pays DNA and opens the **Market**, which sells three ability cards for DNA (reroll for 1 DNA). Beating the Outbreak also gives a free card of a random rarity and moves to the next ante.
- **Ability cards** (`cards` in `mods.go`): 20 cards in four rarities (Common, Uncommon, Rare, Legendary) that change luck, one mutation's chance or value, value by day, luck by night, split time, sells per round, jackpots and more. You hold 3 (up to 5 with skills); tap one to see it or sell it for half its price.
- **Rebirth**: costs $25K, then 10x more each time. It resets money, biotes, upgrades and rounds, doubles biote value for good and pays skill points: 1, plus 1 for every Outbreak beaten that run. Cards, DNA and skills are kept.
- **Skill tree** (`tree` in `mods.go`): 21 skills in five branches below Origin: Luck, Speed (including Auto Split and Auto Sell), Wealth, Cards (slots, better market, free reroll) and Nature (night and day bonuses, more and longer events). Tap a skill to read it, tap again to unlock it.
- **Day and night** (`daylight`, a 4-minute cycle): by day Gold and Heart are likelier, by night Shadow and Cosmic. The sky blends day, dusk and night backdrops, the sun and moon cross it, clouds drift, and stars, shooting stars and fireflies come out at night (`world.go`).
- **Events** every few minutes (`events` in `mods.go`): Meteor Shower, Blizzard, Gold Rush, Aurora, Love Bloom, Heatwave and Eclipse, each changing mutation chances and the weather on screen. The box under the left buttons shows the time and the event; tap it for everything in effect and every mutation's chance right now.
- Music: a chiptune loop written in code (`makeMusic` in `audio.go`); the note button turns it off.

The save is `save.json` in the app's private storage on Android, or in the user config folder (`BlobSplit/`) on a PC. Blob Split saves are converted when loaded (`loadState`).

## Layout

- `game/`: the whole game.
  - `econ.go`: mutations and masks, upgrade costs and curves, rebirth, round targets, number formatting.
  - `mods.go`: everything that changes the rules, gathered each frame into `mods`: ability cards, the skill tree, day and night, events and bosses.
  - `game.go`: game state, rules (split, sell, rounds, market, rebirth, buying) and input.
  - `draw.go`: the screen layout (`lay()`, in units where the screen is 360 tall) and the main screen. `panels.go`: the pop-up screens.
  - `world.go`: the sky, sun and moon, clouds, stars, fireflies and event weather.
  - `sprites.go`: pixel-art biotes (generated in code, one per mutation) and UI icons.
  - `ui.go`: the pixel font (Jersey 15, sizes snapped to whole pixels), colours and the bevelled pixel boxes (same style as Mote).
  - `audio.go`: synthesised chiptune sounds and the music loop. `save.go`: the save file. `icon.go`: the app icon and store art.
  - `testhooks.go`: screenshots and scripted actions for testing (see below).
- `cmd/blobsplit/`: desktop version, for playing and testing on a PC.
- `cmd/makeart/`: writes the launcher icon and the Play Store icon and feature graphic in `store/` (`xvfb-run -a go run ./cmd/makeart`).
- `mobile/`: the Android entry point, bound into an .aar by `ebitenmobile`.
- `android/`: the Android app project that wraps the .aar (package `com.theemeraldpig.blobsplit`, landscape, targets API 36).

## Builds

Every push that changes `blobsplit/` runs `.github/workflows/blobsplit.yml`, which publishes a GitHub release `biote-<number>` with:

- `Biote.apk`: open it on your phone to install.
- `Biote.aab`: the Android App Bundle to upload to Google Play.

The workflow also runs the APK on an emulator and saves a screenshot as the `biote-emulator` artifact.

## Putting it on Google Play

1. Make a [Google Play developer account](https://play.google.com/console/signup) (one-time $25 fee).
2. Make your private **upload key** once, on your own computer, and keep the file and passwords safe:
   ```sh
   keytool -genkeypair -v -keystore upload.jks -alias upload -keyalg RSA -keysize 2048 -validity 10000
   ```
3. Add it to the GitHub repository under Settings > Secrets and variables > Actions:
   - `BLOBSPLIT_UPLOAD_KEYSTORE`: the file as base64 (`base64 -w0 upload.jks`)
   - `BLOBSPLIT_UPLOAD_KEYSTORE_PASSWORD`, `BLOBSPLIT_UPLOAD_KEY_ALIAS` (`upload`), `BLOBSPLIT_UPLOAD_KEY_PASSWORD`

   Then push or re-run the workflow. The next `Biote.aab` is signed with your upload key. Without these secrets, builds are signed with `android/blobsplit-sideload.keystore`, which is public in this repository: fine for installing on your phone, never for Google Play.
4. In the Play Console, create the app, turn on Play App Signing (the default), and upload `Biote.aab` to a testing track first (internal testing is quickest).
5. Store listing: use `store/icon-512.png` and `store/feature-1024x500.png`, plus phone screenshots. Fill in the content rating questionnaire, the data safety form (the game collects no data and has no ads), and a privacy policy link.
6. New personal developer accounts must run a closed test with testers for 14 days before going to production.

Each build's version code is the workflow run number, so it always goes up, as Play requires.

## Testing without a phone

```sh
cd blobsplit
go build -o bs ./cmd/blobsplit
BLOB_NOAUDIO=1 BLOB_DEMO=1 BLOB_SHOT=shot.png BLOB_SIZE=800x360 xvfb-run -a ./bs
```

- `BLOB_SHOT=file.png` saves a screenshot after `BLOB_SHOT_AT` frames (default 90) and quits; screenshot runs never touch the save.
- `BLOB_DEMO=1` starts with a late-game board; `BLOB_FRESH=1` starts a new game.
- `BLOB_PANEL=tree|index|rebirth|market|blob|card|world|round` opens a menu (`BLOB_SEL=n` picks the biote or card); `BLOB_DO=split,split,sell,buy0,win` runs actions, one every 20 frames.
- `BLOB_CLOCK=0.75` sets the time of day (0.25 midday, 0.75 midnight); `BLOB_EVENT=meteor` starts an event.
- `BLOB_SIZE=WxH` sets the window size (800x360 is a phone in landscape).

`go test ./...` (under `xvfb-run -a` on a headless Linux) checks number formatting, upgrade curves, the 1/15 mutation chance, stacking and inheriting, rounds, the market, the skill tree, rebirth, day/night and events, loading Blob Split saves, and updating before the screen size is known.
