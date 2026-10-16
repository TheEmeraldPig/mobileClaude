# Orbit Hop

A one-tap space arcade game for Android (Go + Ebitengine, like `mote/`).

- Your ship circles a planet. Tap to fling it off along its path; reach another
  planet's pull to score. Skipping a planet scores both; arriving right on the
  orbit line is a "Perfect" (+1).
- Drifting off screen, crashing into a planet or hitting an asteroid ends the run,
  and the red void creeps in from the left, faster the further you get.
- Stars and planets give coins. **Shop**: upgrades (magnet, gravity well, shields,
  star value, slower void), ships (cosmetic, coins or gems), and a **Store** with
  real-money packs that are free placeholders until Google Play Billing is added.
- After a run you can continue once (5 gems or a placeholder "watch ad").
- Daily reward streak (1-7 gems).

Code: `game/run.go` (one run), `game/shop.go` (save, shop, prices), `game/game.go`
(screens), `game/draw.go` (drawing and colours).

## Run and test here

    go test ./game
    go run ./cmd/hop

Screenshots without a person, under Xvfb:

    HOP_DATA=/tmp/h HOP_SCRIPT="wait 20;press play;wait 30;shot a.png;quit" xvfb-run -a go run ./cmd/hop

Commands are in `game/testhooks.go`.

## Web version

`web/build.sh` builds a WebAssembly version into `web/out` (saves go to the browser's localStorage).
It is published as a Claude artifact at https://claude.ai/artifact/Kn58gXi11btKxoKsDA1bNQ for quick play-testing.

## Builds

Every push to `hop/` runs `.github/workflows/hop.yml`, which publishes `OrbitHop.apk` and
`OrbitHop.exe` as a `hop-build-N` release (not marked latest, so Mote's updater is unaffected).
The app icon is Mote's for now.
