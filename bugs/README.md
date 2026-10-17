# Bug Brigade

A portrait idle/tap battler for Android (Go + Ebitengine, like `mote/`), in the
spirit of Best Fiends Forever: a team of cute bugs fights off waves of slugs.

- **Fighting**: your bugs attack on their own; tap the battlefield to hit too (every
  finger counts). Each stage has 10 slugs and then a boss with a 30 second timer. Lose
  to the boss and you keep farming the stage until you press "Fight Boss!".
- **Bugs**: 8 bugs in 4 rarities (Common, Rare, Epic, Legendary), each with its own
  attack speed and a skill that charges up (tap its portrait when it says TAP!): big
  hits, hitting every slug, team frenzy, crit taps, extra boss time, triple coins.
- **Growth**: coins level bugs and tap power (damage doubles at levels 10, 25, 50 and
  every 50 after). Team slots open at stages 2, 5, 10 and 18. Every 5th boss drops an
  egg; a new area starts every 5 stages (6 areas, then they repeat).
- **Money**: gems from bosses buy eggs (copies add stars: +50% damage each, up to 5),
  a free egg every 4 hours, a coin sack, a starter pack and gem packs. Real-money items
  are free test purchases until Google Play Billing is added. Coins keep coming while
  the game is closed (half rate, up to 8 hours), doubled for 30 gems.

## Art

Everything is painted in code, once, at the screen's real resolution (`paint.go`):
shapes are signed distance fields, so edges are smooth and outlines exact, and each
shape is shaded as a soft rounded volume (wrap-around light, a bounce light on the
shaded side, gloss). Bugs and slugs are built from these shapes in layers (wings,
body, face) so they can blink, flap, squash and stretch (`bugs.go`, `slugs.go`,
`draw.go`). Backdrops are in `scene.go`, buttons, panels and icons in `ui.go`.

Code: `game/profile.go` (save and all the numbers), `game/game.go` (input, battle,
effects), `game/draw.go` (battlefield and HUD), `game/screens.go` (menus and
overlays), `game/art.go` (scale, image cache, text). Fonts: Lilita One and Fredoka,
SIL Open Font License (`game/FONT_LICENSE.txt`).

## Run and test here

    xvfb-run -a go test ./game
    go run ./cmd/bugs

Screenshots without a person, under Xvfb (portrait screen):

    BUGS_SCALE=2 BUGS_DATA=/tmp/d BUGS_SCRIPT="wait 120;press tab:2;wait 30;shot a.png;quit" xvfb-run -a -s "-screen 0 720x1280x24" go run ./cmd/bugs

`BUGS_TIMING=1` prints how long drawing takes. Commands are in `game/testhooks.go`.
`BUGS_PREVIEW=dir go test ./game -run Preview` paints every bug and slug into one sheet;
`BUGS_ICON=file go test ./game -run Icon` paints the app icon.

## Web version

`web/build.sh` builds a WebAssembly version into `web/out` (saves go to the browser's localStorage).

## Builds

Every push to `bugs/` runs `.github/workflows/bugs.yml`, which builds `BugBrigade.apk`
and `BugBrigade.exe` and publishes them as a `bugs-build-N` release (not marked latest,
since Mote's updater reads the latest release).
