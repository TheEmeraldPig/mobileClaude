# Deckbound

A portrait card roguelite for Android (Go + Ebitengine, like `mote/`): Slay the
Spire-style fights, Balatro-style deck customisation, Archero-style menus.

- **Fights**: 3 energy and 5 cards a turn. Cards deal damage, block, poison, draw,
  weaken. The enemy shows what it will do next.
- **Runs**: 10 floors per chapter (elite on 5, boss on 10). After each fight pick one
  of 3 rewards: a new card, an enhancement on one of your cards, a charm, or a heal.
  Beating the boss unlocks the next, harder chapter.
- **Charms** bend the rules for the rest of the run (max 4): +2 attack damage, every
  3rd card doubled, thorns, extra draw...
- **Deck customisation between runs**: put enhancements on any card for good (Foil +3,
  Holo x1.5, Light -1 cost, Gold +6 gold, Echo plays twice), remove cards, get new
  ones from chests. Hero level adds max HP.
- **Money**: energy (5 per run, refills over time), gems, chests, revive for gems or an
  ad, gem packs. Real-money items are free placeholders until Google Play Billing is added.

Code: `game/cards.go` (cards, enhancements, charms, enemies), `game/battle.go` (a run
and its fights), `game/profile.go` (save, energy, shop), `game/game.go` (screens),
`game/ui.go` (drawing). Font: Lilita One, SIL Open Font License (`game/FONT_LICENSE.txt`).

## Run and test here

    go test ./game
    go run ./cmd/deck

Screenshots without a person, under Xvfb (portrait screen):

    DECK_DATA=/tmp/d DECK_SCRIPT="wait 20;press start;wait 30;shot a.png;quit" xvfb-run -a -s "-screen 0 720x1280x24" go run ./cmd/deck

Commands are in `game/testhooks.go`.

## Web version

`web/build.sh` builds a WebAssembly version into `web/out` (saves go to the browser's localStorage).
It is published as a Claude artifact at https://claude.ai/artifact/Kn58gXi11btKxoKsDA1bNQ for quick play-testing.

## Builds

Every push to `deck/` runs `.github/workflows/deck.yml`, which publishes `Deckbound.apk` and
`Deckbound.exe` as a `deck-build-N` release (not marked latest, so Mote's updater is unaffected).
The app icon is Mote's for now.
