# Biote

An idle mobile game made with Go and [Ebitengine](https://ebitengine.org), built for Android and Google Play. Tap **SPLIT** to double your biotes until the board is full, **SELL** them for money, and grow stronger through upgrades, rounds, ability cards, rebirths and a skill tree. (It started as "Blob Split"; the folder, package id `com.theemeraldpig.blobsplit` and save folder keep that name so updates install over it.)

## Playing

| | Phone | PC |
|---|---|---|
| Split | SPLIT button | Space |
| Sell | SELL button | S |
| Buy an upgrade | Tap a card in the drawer on the right (hold to keep buying) | Click (hold to keep buying) |
| Open or close the upgrades | The tab on the drawer's edge | Click it |
| Scroll the genome or shop | Drag the list or its scroll bar | Mouse wheel, or drag |
| See a biote's mutations | Tap it | Click it |
| Close a menu | X, or tap outside it | Esc |

- **Mutations**: Gold 1/25 (x3), Shadow 1/60 (x6), Frost 1/150 (x15), Heart 1/400 (x40), Toxic 1/1K (x100), Rainbow 1/3K (x300), Cosmic 1/10K (x1000). Without luck a new biote gains a mutation 1 time in 15 (`mutationOdds`). Mutations stack: a biote can carry several, their multipliers multiply, and the label above it shows the chance of all of them together (`comboOdds`). A child always keeps its parent's mutations and can gain more. Tap a biote to see each mutation's chance and multiplier and its base value. The Catalog only shows mutations you have found.
- **Tutorial** (`extras.go`): new players get six steps (split, fill the board, sell, buy an upgrade, open CHALLENGE, mutations), each pointing at its button with a pulsing ring. SKIP ends it; Settings has REPLAY TUTORIAL. Saves from before it skip it.
- **Upgrades** (money, reset on rebirth): biote value, split speed (3s down to 2s) and capacity (4 up to 64). They sit in a drawer on the right that slides open and shut with the tab on its edge (it remembers which); with it shut the board gets wider, and the tab shows "!" when you can afford an upgrade. The BUY button at the top of the drawer sets x1, x5, x25 or MAX. Selling restarts the split timer. There is no money upgrade for luck: luck comes from skills, cards, the time of day and events.
- **The challenge** (`floors.go`): always running. Each floor has 5 rooms; room 5 is a boss with a twist (`bosses` in `mods.go`: The Parasite, The Drought, The Freeze, The Leech, The Mirror). Each room has a requirement (earn $X within N sells, make one sale worth $X, sell N mutated biotes at once, or create a biote with a given mutation), fixed per floor and room (`reqFor`), and only shown in the CHALLENGE screen (the button at the top centre). Every sale counts. Clearing the boss room adds 1 DNA to your next rebirth; running out of sells sends you back to room 1 of the floor with nothing else lost. Money targets grow 4x each floor. Every 10th floor cleared past your best unlocks an ability card you don't have yet.
- **Ability cards** (`cards.go`): about 70 cards in five rarities (Common, Uncommon, Rare, Epic, Legendary) and ten types (Luck, Value, Speed, Gene, Spread, Time, Storm, Climb, Combo, Idle; the stripe on a card). Every mutation gets a Charm and Lure (chance), Polish and Crown (value) and Spores (spread) card; each type has a ladder of general cards. They are only won from floors, and every unlocked card is always active. The CHALLENGE screen lists them in rarity tabs, locked ones as "?"; tap one to read it. Old card ids are mapped in `oldCards`.
- **Spreading**: when a biote splits, its child can also pick up each mutation of the parent's neighbours on the board (left, right, above, below; `neighbours`, `spreadChance`, 3% base).
- **DNA** (the double helix, `meta.go`): paid by rebirths (1 per floor cleared that run). Spend it in the GENOME screen (the old catalog; tap DNA or the GENOME button) on each found mutation's VALUE (+25% a level), CHANCE (+20%) and SPREAD (+50%).
- **Tokens**: +1 per rebirth for every time your money doubled the rebirth requirement.
- **Shop** (`shop.go`, the token button top right), in four tabs. Every item has a dollar price and a token price (1000 tokens per $1):
  - BOOSTS, forever: Money, Mutations, DNA, Tokens and Skill Points x1.5 ($4.99 each), Golden Biotes (value x2, $4.99), Night Watch (idle x2, $3.99).
  - SKIPS: 1, 4 or 24 hours of income ($0.99, $2.99, $7.99), Instant Rebirth ($0.99) and Skip 10 Floors ($2.49; DNA and any cards for floors past your best).
  - BUNDLES: Starter Pack (2 skill points, a 4h skip, a card; $1.99), Evolution Pack (Money and Mutations x1.5; $7.99), Galaxy Pack (all five x1.5 boosts and No Ads; $19.99).
  - MORE: No Ads ($9.99: every ad button gives its reward at once), a skill point, a card pack.
  - When you can't afford a rebirth, the rebirth screen offers REBIRTH NOW for $0.99; the CHALLENGE screen offers to skip 10 floors.
  - Real money and ads go through the `Purchase` and `ShowAd` hooks in `shop.go`, which the Android app has to set once Google Play Billing and AdMob are added. Until then the dollar buttons say payments aren't connected (items are bought with tokens), and an ad is a 3 second placeholder before its reward.
- **Idle gains**: when you come back, you get 25% of your recent income per second for the time away (up to 8 hours; cards and the Night Watch item raise it), shown on a welcome-back screen with COLLECT or "x2" for watching an ad.
- **Visitors** (`extras.go`): every minute or two a glowing orb drifts across the screen. Tap it for a buff: Double Cash (value x2), Lucky Star (mutation chance x3), Split Rush (splits 2x faster) or Piggy Bank (2 minutes of income now). Half are gifts (take it, or watch an ad to double it); half need an ad. Running buffs show under the CHALLENGE button with their time left, and are saved.
- **Animations**: buttons squash when pressed, SELL throbs when the board is full, affordable upgrades shimmer, confetti falls for cleared rooms, floors, rebirths and purchases, and the board shakes when a floor is cleared.
- **Settings** (the gear, top right): music, sound, version, and RESET SAVE, which asks again before deleting anything.
- **AUTO** split and sell work from the start at half speed; the Fast Auto skills make them full speed.
- **Pacing**: tuned with a bot (`sim_test.go`, `BLOB_SIM=minutes go test -run TestSim -v ./game`) that plays the real rules: the first card (10 floors) takes about an hour, rebirths come roughly every 1.5 hours. Value and its cost grow by about the same factor a level (`valueStep`), so floors take a steady time rather than hitting a wall.
- **Rebirth**: costs $25K, then 10x more each time. It resets money, biotes, upgrades and your floor (back to floor 1), doubles biote value for good and pays 1 skill point, DNA (1 per floor cleared that run) and tokens. Cards, DNA, tokens, your best floor and skills are kept.
- **Skill tree** (`tree` in `mods.go`, its own full screen in `tree.go`): 25 skills on hexagons in a honeycomb flower around Origin, six branches: Luck, Nature, Speed (with Auto Split and Auto Sell), Genes (mutation multipliers, DNA Bank, Purebred), Challenge (Climber, Lucky Draw, Endurance, Bounty) and Wealth. Each skill has axial hex coordinates (`q`, `r`) and sits next to the one it needs. Only skills you can unlock next are shown: at first just Origin, and each skill you buy reveals the next one in its branch. Tap a hexagon to read it on the right, tap again (or UNLOCK) to buy it.
- **Day and night** (`daylight`, a 4-minute cycle): by day Gold and Heart are likelier, by night Shadow and Cosmic. The sky blends day, dusk and night backdrops, the sun and moon cross it, clouds drift, and stars, shooting stars and fireflies come out at night (`world.go`).
- **Events** every few minutes (`events` in `mods.go`): Meteor Shower, Blizzard, Gold Rush, Aurora, Love Bloom, Heatwave and Eclipse, each changing mutation chances and the weather on screen. The box under the left buttons shows the time and the event; tap it for everything in effect and every mutation's chance right now.
- **Music** (`music.go`): spacey synth loops written in code (pads, sub bass, echoing arpeggios, FM bells, sparkles), one for day, one for night and one for each event, each with its own scale, tempo and chords. A track is made in the background the first time it's needed and the game crossfades to it. The note button turns music off.

The save is `save.json` in the app's private storage on Android, or in the user config folder (`BlobSplit/`) on a PC. Blob Split saves are converted when loaded (`loadState`).

## Layout

- `game/`: the whole game.
  - `econ.go`: mutations and masks, upgrade costs and curves, rebirth, round targets, number formatting.
  - `mods.go`: everything that changes the rules, gathered each frame into `mods`: ability cards, the skill tree, day and night, events and bosses.
  - `game.go`: game state, rules (split, sell, rebirth, buying) and input.
  - `floors.go`: the challenge (floors, rooms, requirements, card rewards) and its CHALLENGE screen.
  - `cards.go`: the ability cards. `meta.go`: the DNA lab, idle gains and save reset.
  - `shop.go`: the shop, paid skips, and ads. `extras.go`: visitors and buffs, the tutorial and small animations.
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

## Web version

Biote also runs in a browser (Ebitengine compiles to WebAssembly), which is how it is published as a Claude artifact. `sh web/build.sh [out-dir]` (from `blobsplit/`) writes `index.html` (the loading page, `web/page.html`, with Go's `wasm_exec.js` inlined) and `biote-wasm.txt` (the game, gzipped and base64-encoded, so it fits the artifact's file size limit; the page unpacks it with `DecompressionStream`). Publish `index.html` with `biote-wasm.txt` beside it. In a browser the save goes to `localStorage` (`store_web.go`) instead of a file (`store_file.go`).

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
5. Store listing: use `store/icon-512.png` and `store/feature-1024x500.png`, plus phone screenshots. Fill in the content rating questionnaire, the data safety form, and a privacy policy link. Once AdMob and Play Billing are added (see the shop above), the listing must say the app has ads and in-app purchases, and the data safety form must cover what the ad SDK collects.
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
- `BLOB_PANEL=tree|challenge|index|rebirth|blob|world|shop|away|visitor` opens a screen (`BLOB_SEL=n` picks the biote or card); `BLOB_DO=split,split,sell,buy0,win` runs actions, one every 20 frames (also `visitor`, `ad`, `buff`, `away`, `gift`, `offer`, `tab0`-`tab3`).
- `BLOB_TUT=n` shows tutorial step n (demo and panel runs skip the tutorial otherwise); `BLOB_DRAWER=0` shuts the upgrades drawer; `BLOB_SCROLL=units` scrolls a panel's list.
- `BLOB_CLOCK=0.75` sets the time of day (0.25 midday, 0.75 midnight); `BLOB_EVENT=meteor` starts an event.
- `BLOB_SIZE=WxH` sets the window size (800x360 is a phone in landscape).

`go test ./...` (under `xvfb-run -a` on a headless Linux) checks number formatting, upgrade curves, the 1/15 mutation chance, stacking and inheriting, floors and rooms, cards, the skill tree, rebirth, day/night and events, loading Blob Split saves, updating before the screen size is known, the shop and paid skips, ads, buffs, the tutorial, and the drawer layout at phone, tablet and PC sizes.
