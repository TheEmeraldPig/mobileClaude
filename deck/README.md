# Deckbound

A portrait card roguelite for Android (Go + Ebitengine, like `mote/`): Slay the
Spire-style fights, Balatro-style deck customisation, Archero-style menus.

- **Your collection and starting deck**: pick up to 5 cards from your collection to
  start each run with. The budget: 1 Legendary, or 2 Epics (no Uncommons), or 3 Rares
  plus 1 Uncommon, plus any Commons; or any mix of Commons and Uncommons. Enhance
  collection cards for good (Foil +3, Holo x1.5, Light -1 cost, Gold +6 coins, Echo
  plays twice). Chests add cards. Rarities: Common, Uncommon, Rare, Epic, Legendary.
- **Runs**: a branching map of 10 rows plus the boss. Choose your path through fights,
  elites, the merchant (cards, charms, remove a card, forge an enhancement, heal),
  campfires (heal or forge), mystery events and treasure. After a fight, add 1 of 3
  cards to your deck (or skip). Coins are spent in the run; gold is paid out after it.
- **Fights**: 3 energy and 5 cards a turn; enemies show what they will do next.
- **Charms** bend the rules for the rest of the run: +2 attack damage, every 3rd card
  doubled, block that stays, extra draw...
- **Money**: energy (5 per run, refills over time), gems, chests, revive for gems or an
  ad, gem packs. Real-money items are free placeholders until Google Play Billing is added.
- A run in progress is saved, so closing the app doesn't lose it.

All art is drawn in code (shaded shapes, gradients, glows) at the screen's real pixel
resolution: `S` in `art.go` maps the 540-wide layout to pixels.

Code: `game/cards.go` (cards, rules, charms, enemies, chapters), `game/run.go` (map,
run, merchant, events), `game/battle.go` (fights), `game/profile.go` (save, collection,
energy, shop), `game/game.go` (input, flow, effects), `game/screens_*.go` (screens),
`game/art.go` (drawing kit), `game/characters.go`, `game/cardart.go`, `game/scenes.go`. Font: Lilita One, SIL Open Font License (`game/FONT_LICENSE.txt`).

## Run and test here

    go test ./game
    go run ./cmd/deck

Screenshots without a person, under Xvfb (portrait screen):

    DECK_SCALE=2 DECK_DATA=/tmp/d DECK_SCRIPT="wait 20;press start;goto shop;wait 30;shot a.png;quit" xvfb-run -a -s "-screen 0 720x1280x24" go run ./cmd/deck

`DECK_TIMING=1` prints how long drawing takes.

Commands are in `game/testhooks.go`.

## Web version

`web/build.sh` builds a WebAssembly version into `web/out` (saves go to the browser's localStorage).
It is published as a Claude artifact at https://claude.ai/artifact/Kn58gXi11btKxoKsDA1bNQ for quick play-testing.

## Builds

Every push to `deck/` runs `.github/workflows/deck.yml`, which publishes `Deckbound.apk` and
`Deckbound.exe` as a `deck-build-N` release (not marked latest, so Mote's updater is unaffected).
The app icon is Mote's for now.
