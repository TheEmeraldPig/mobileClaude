# Kitchen Clash

A portrait idle/tap battler for Android (Go + Ebitengine, like `mote/`) with a cooking
half. Your crew of chefs fights rotten food in Clicker Heroes style; the monsters drop
ingredients; the kitchen cooks them into dishes that power up the fight.

## The fight

- One big monster at a time. Your chefs throw food at it on their own; tap the monster
  to hit too (every finger counts). 10 monsters per stage, then a boss with a 30 second
  timer. Lose to the boss and you keep fighting the stage until you press "Fight Boss!".
- 8 chefs in 4 rarities, each with a skill that charges up (tap the portrait when it
  says TAP!): big hits, flurries, crew frenzy, crit taps, extra boss time, triple coins.
- Coins level chefs and tap power (damage doubles at levels 10, 25, 50, then every 50).
  Crew spots open at stages 2, 5, 10 and 18. Every 5th boss gives a free recruit.
- 6 areas of 5 stages, each with its own monster family and ingredients: The Pantry
  (flour, eggs, cheese), Veggie Garden, BBQ Yard, Candy Shop, The Fridge, Dumpster Alley.

## The kitchen

- **Stations**: Counter (free), Stove, Oven, Grill, Freezer. Build them with coins and
  upgrade them: each level cooks 15% faster and makes dishes 20% stronger.
- **Cooking**: pick a recipe, then stop the swinging needle in the green (Perfect = the
  dish is 50% stronger). It cooks in real time, also while the game is closed.
- **Serving**: 3 plates on the table. A served dish gives its buff for 4 minutes of
  play: chef damage, tap damage, crits, coins, boss time, skill charge, ingredient
  drops, boss damage or all damage.
- **Combos**: a starter + main + dessert is a full course (every dish x1.5); pairs like
  Omelette + Pancakes (Breakfast Club) or Taco + Chili (Fiesta!) add extra bonuses; and
  each chef has a favourite dish that doubles their damage while it is on the table.

## Money

Gems from bosses buy Recruit Boxes and Star Recruits (copies add stars: +50% damage each,
up to 5), coin sacks and grocery crates. A free recruit every 4 hours, a starter pack and
gem packs. Real-money items are free test purchases until Google Play Billing is added.
Coins and ingredients keep coming while the game is closed (half rate, up to 8 hours).

## Art

Everything is painted in code, once, at the screen's real resolution (`paint.go`): shapes
are signed distance fields, inked with thick dark outlines and cel shaded (one hard
shadow tone, a hard highlight), like a hand-drawn cartoon. Chefs are busts in two layers
(body, face states) in `chefs.go`; monsters (body, angry and hurt faces) in
`monsters.go`; food icons in `food.go`; backdrops, the kitchen and its stations in
`scene.go`; buttons, panels and icons in `ui.go`.

Code: `game/profile.go` (save and all the numbers, the kitchen rules), `game/food.go`
(ingredients, recipes, combos), `game/game.go` (input, fight, cooking minigame),
`game/draw.go` (fight, HUD, crew bar, kitchen view), `game/screens.go` (menus and
overlays), `game/art.go` (scale, image cache, text). Fonts: Lilita One and Fredoka, SIL
Open Font License (`game/FONT_LICENSE.txt`).

## Run and test here

    xvfb-run -a go test ./game
    go run ./cmd/kitchen

Screenshots without a person, under Xvfb (portrait screen):

    KITCHEN_SCALE=2 KITCHEN_DATA=/tmp/d KITCHEN_SCRIPT="wait 120;press tab:2;wait 30;shot a.png;quit" xvfb-run -a -s "-screen 0 720x1280x24" go run ./cmd/kitchen

`KITCHEN_TIMING=1` prints how long drawing takes. Commands are in `game/testhooks.go`.
`KITCHEN_PREVIEW=dir go test ./game -run Preview` paints every chef, monster and food
into one sheet; `KITCHEN_ICON=file go test ./game -run Icon` paints the app icon.

## Web version

`web/build.sh` builds a WebAssembly version into `web/out` (saves go to the browser's localStorage).

## Builds

Every push to `kitchen/` runs `.github/workflows/kitchen.yml`, which builds
`KitchenClash.apk` and `KitchenClash.exe` and publishes them as a `kitchen-build-N`
release (not marked latest, since Mote's updater reads the latest release).
