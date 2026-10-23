# Blob Split

An idle mobile game made with Go and [Ebitengine](https://ebitengine.org), built for Android and Google Play. Tap **SPLIT** to double your blobs until the board is full, **SELL** them for money, buy upgrades, and **REBIRTH** for a bigger multiplier and gems. Every new blob has a small chance to be a rare one worth much more.

## Playing

| | Phone | PC |
|---|---|---|
| Split | SPLIT button | Space |
| Sell | SELL button | S |
| Buy an upgrade | Tap a card on the right (hold to keep buying) | Click (hold to keep buying) |
| Close a menu | X, or tap outside it | Esc |

- **Upgrades** (reset on rebirth): blob value, split speed (3s down to 2s), capacity (4 up to 64 blobs) and luck.
- **Rare blobs**: Gold 1/100 (x5), Shadow 1/250 (x12), Frost 1/600 (x30), Heart 1/1.5K (x75), Toxic 1/4K (x200), Rainbow 1/12K (x700), Cosmic 1/50K (x3000). Luck divides the odds. The Index shows which ones you've found.
- **Rebirth**: costs $25K, then 10x more each time. It resets money, blobs and upgrades, doubles blob value for good and pays gems.
- **Gem shop** (permanent): Auto Split, Auto Sell, Golden Touch (+50% value per level), Lucky Charm (+50% luck per level) and Quick Hands (-10% split time per level).

The save is `save.json` in the app's private storage on Android, or in the user config folder (`BlobSplit/`) on a PC.

## Layout

- `game/`: the whole game.
  - `econ.go`: rarities, upgrade costs and curves, rebirth, gem upgrades, number formatting. Balance changes go here.
  - `game.go`: game state, rules (split, sell, rebirth, buying) and input.
  - `draw.go`: the screen layout (`lay()`, in units where the screen is 360 tall) and all drawing.
  - `sprites.go`: pixel-art blobs (generated in code, one per rarity) and UI icons.
  - `ui.go`: fonts, colours and the bevelled pixel boxes (same style as Mote).
  - `audio.go`: synthesised chiptune sounds. `save.go`: the save file. `icon.go`: the app icon and store art.
  - `testhooks.go`: screenshots and scripted actions for testing (see below).
- `cmd/blobsplit/`: desktop version, for playing and testing on a PC.
- `cmd/makeart/`: writes the launcher icon and the Play Store icon and feature graphic in `store/` (`xvfb-run -a go run ./cmd/makeart`).
- `mobile/`: the Android entry point, bound into an .aar by `ebitenmobile`.
- `android/`: the Android app project that wraps the .aar (package `com.theemeraldpig.blobsplit`, landscape, targets API 36).

## Builds

Every push that changes `blobsplit/` runs `.github/workflows/blobsplit.yml`, which publishes a GitHub release `blobsplit-<number>` with:

- `BlobSplit.apk`: open it on your phone to install.
- `BlobSplit.aab`: the Android App Bundle to upload to Google Play.

The workflow also runs the APK on an emulator and saves a screenshot as the `blobsplit-emulator` artifact.

## Putting it on Google Play

1. Make a [Google Play developer account](https://play.google.com/console/signup) (one-time $25 fee).
2. Make your private **upload key** once, on your own computer, and keep the file and passwords safe:
   ```sh
   keytool -genkeypair -v -keystore upload.jks -alias upload -keyalg RSA -keysize 2048 -validity 10000
   ```
3. Add it to the GitHub repository under Settings > Secrets and variables > Actions:
   - `BLOBSPLIT_UPLOAD_KEYSTORE`: the file as base64 (`base64 -w0 upload.jks`)
   - `BLOBSPLIT_UPLOAD_KEYSTORE_PASSWORD`, `BLOBSPLIT_UPLOAD_KEY_ALIAS` (`upload`), `BLOBSPLIT_UPLOAD_KEY_PASSWORD`

   Then push or re-run the workflow. The next `BlobSplit.aab` is signed with your upload key. Without these secrets, builds are signed with `android/blobsplit-sideload.keystore`, which is public in this repository: fine for installing on your phone, never for Google Play.
4. In the Play Console, create the app, turn on Play App Signing (the default), and upload `BlobSplit.aab` to a testing track first (internal testing is quickest).
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
- `BLOB_PANEL=shop|index|rebirth` opens a menu; `BLOB_DO=split,split,sell,buy0` runs actions, one every 20 frames.
- `BLOB_SIZE=WxH` sets the window size (800x360 is a phone in landscape).

`go test ./...` (under `xvfb-run -a` on a headless Linux) checks the number formatting and upgrade curves.
