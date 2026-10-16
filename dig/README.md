# Deep Dig

An idle mining game for Android (Go + Ebitengine, like `mote/`).

- Tap the rock for coins. Buy a better pickaxe and diggers that mine on their own.
- **Descend** restarts the mine for gems; each gem kept gives +10% coins.
- **Gem Shop**: boosts, time warps, auto tapper, golden pickaxe, longer offline earnings.
- **Store**: real-money gem packs. These are free placeholders until Google Play Billing is added.
- Daily reward streak (1-7 gems) and offline earnings bring players back.

Economy numbers are in `game/economy.go`.

## Run and test here

    go test ./game
    go run ./cmd/dig

Screenshots without a person, under Xvfb:

    DIG_DATA=/tmp/d DIG_SCRIPT="wait 20;coins 5000;tab 1;shot a.png;quit" xvfb-run -a go run ./cmd/dig

Commands are in `game/testhooks.go`.

## Builds

Every push to `dig/` runs `.github/workflows/dig.yml`, which publishes `DeepDig.apk` and
`DeepDig.exe` as a `dig-build-N` release. The app icon is Mote's for now.
