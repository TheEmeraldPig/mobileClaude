# Desalote

A multiplayer poker-hand scoring card game for 1-4 players, in the spirit of Balatro but with its own rules: a locked 13-card deck, hands picked for the whole match, synergy between them, special hands, charms instead of jokers, a live auction shop, a charm market between players, and hexes to curse your rivals. It reuses Mote's engine (Go + [Ebitengine](https://ebitengine.org)): the immediate-mode UI, the synthesised sound, the multiplayer hub and the Android wrapper. Nothing is shared with Mote's code at build time; the parts were copied.

All of Desalote's art, cards, charms and rules are its own. Nothing comes from Balatro's files.

## Downloads

Every push that changes `desalote/` builds through `.github/workflows/desalote.yml` and publishes a GitHub release named `desalote-build-<number>`, with:

- `Desalote.exe` for Windows. It checks for new builds every minute and updates itself between matches.
- `Desalote.apk` for Android. It checks for new builds when it starts and every 5 minutes, then asks to install them.
- `Desalote-steam.exe`, the Steam build (no built-in updater).

This repository also releases Mote, and Mote's updaters install the repository's "latest" release. So Desalote's releases are created with `--latest=false`, and Desalote's own updaters (`game/update_windows.go`, `android/.../Updater.java`) look through the release list for the newest `desalote-build-` tag. Keep it that way, or Mote players would be offered Desalote as an update.

## Rules

### Your 13 cards
- At the start of a match each player is dealt **13 different cards** at random (no two of the same rank and suit). They're locked for the whole match: nothing adds, removes or changes cards. The only exception is the small permanent bonus a special hand gives the cards it uses.
- Each round you hold **3 cards**, with **1 play** and **1 discard**. Charms raise these: Wide Grip (+1 card), Card Sleeve (+1 card, +1 discard), Satchel (+2 cards), Pocket Watch (+1 play), Sieve (+2 discards). When the pile runs out, the played cards are shuffled back in.
- **Flush size** is the average number of cards per suit in your deck, rounded down: 3 clubs + 3 hearts = 3; 2 diamonds + 5 clubs + 3 hearts = 3.33, so 3.
- **Straight size** is the average length of your deck's runs of ranks in a row, rounded up.
- Neither is ever more than 5, since a play is at most 5 cards (`FlushSize`, `StraightSize` in `game/cards.go`). Across random decks the flush size is 3 (95%) or 4; the straight size is usually 2-4, sometimes 5.

### Hands
- Everyone can always score **High Card** and **Pair**. These never count toward synergy.
- Each player is given a **family**:
  - **Same:** Two Pair, Three of a Kind, Full House, Four of a Kind. A Same deck always has at least 3 cards of one rank.
  - **Counting:** Straight, Skip Straight (ranks stepping by 2), Ladder (7 7 8 8).
  - **Suits:** Flush, Two Tone (4+ cards of exactly 2 suits), Rainbow (4 different suits).
- Your three hands are **dealt at random** (`dealHands` in `game/match.go`). First the synergy is rolled: all three from your family, two, or one. Hands from outside your family start a level up, so a low roll isn't a worse deal. Any other hand only counts as a Pair or High Card. Some hands need more cards than you start with (Two Pair needs 4); charms that enlarge your hand make them possible.
- **Synergy:** when two picks share a family, all three score x1.2. When all three do, x1.5.
- **Special hand:** Five of a Kind, Straight Flush or Flush House. Once a round, the cards in your hand (up to 5; you don't choose) briefly turn into that hand. It scores big and doesn't use a play. Afterwards the cards go back to normal with +3 points each, for good. Five of a Kind always uses the deck's most common rank (ties at random). That makes it a little predictable, so it scores less than the other two.
- A hand scores (hand base + the scoring cards' values) x the hand's Mult, then charms, hexes and synergy apply (`Run.score` in `game/run.go`).

### Modes (picked by the host)
- **Duel:** 1 v 1. Small blinds are against the house (miss the target, lose a life). Big and Boss blinds are showdowns: the lower score loses a life.
- **Last Standing:** 2-4 players. Miss a blind's target and lose a life. Each Boss blind is a showdown where the lowest score loses a life. Last one standing wins, otherwise the most lives (then points) after the last ante.
- **Co-op Heist:** everyone's scores add up against a bigger team target, and the team shares its lives.
- **Race:** everyone plays at their own pace with a personal shop (set prices). First to beat the final Boss blind wins. Hexes still work.
- Length: Short (3 antes), Standard (5) or Long (8). Each ante has a Small, Big and Boss blind. Lives: 1-5.

### Between rounds: the shop
- **Auction:** three charms and a hex (four charms in co-op) go to the highest bidder. A late bid adds a few seconds, and you can't promise more money than you have. The shop closes when everyone presses Done or the timer runs out.
- **Tomes:** level up one of your picked hands at a set price (once per hand per shop).
- **Trade:** sell a charm to the bank for half its price, or list it for other players at your price and buy theirs.
- **Hexes:** cast on a rival to curse their next round: Fog, Rust, Squeeze, Muddle, Jinx, Freeze, Shackle, or Pickpocket (which happens straight away). A Mirror Ward bounces one back, and a Thorn Ward makes the caster pay.
- There are 28 **charms** (`charms` in `game/charms.go`). Many play off this game's own systems: your family, synergy, special hands, lives, hexes and the other players.

### Handy controls
Quicker ways to play, after the Balatro mod Handy. Each can be switched off in Settings > Handy controls (`handySettings` in `game/menus.go`, keys in `tableKeys` in `game/game.go`):
- **Sweep to pick:** press a card and drag across the others to pick (or drop) them all; works on touch screens. On a computer, Shift + pointing picks cards, and right-click (or C) drops them all.
- **Sorting:** by rank or by suit (the Sort button, R and T); remembered.
- **Game speed** x1, x2 or x4 (F), or **instant scoring**.
- **Quick actions:** Ctrl+click a charm to sell it, or a shop item to bid on (or buy) it.
- **Keys:** Enter/Space play, D discard, S special, 1-8 pick cards, Tab your deck, H hexes; in the shop 1-4 bid or buy, Q tomes, W trade, Enter done.

### Layout
The table is kept open: the players are pills along the top (tap one for details), the score card sits at the top left, your charms are coins at the top right (tap for details and selling), and hexes, tomes, trading and your deck (tap the deck pile) open as menus (`game/layout.go`).

## Playing together

- **Same Wi-Fi or hotspot:** Host a table. Friends pick Join a table and see it listed, or type the host's address. No server is needed: the host's game runs the message hub (`relaycore`) and announces the table on UDP port 47479. The hub listens on 47480 and up, different ports from Mote's so the two games never list each other.
- **Online code:** when the build has a relay address, the host can switch to a 5-letter code. Desalote uses Mote's relay (the `MOTE_RELAY_URL` repository variable). The relay only passes messages, so one server serves both games.
- Bots can fill any empty seat, and Play with bots starts a match against bots alone.

## Layout

- `game/cards.go`: cards, dealing the 13-card deck, flush and straight sizes.
- `game/hands.go`: hand types, families, detecting hands, levels, synergy.
- `game/charms.go`: charms, hexes.
- `game/run.go`: one player's run: drawing, playing, discarding, the special hand, scoring, and the bot's brain.
- `game/match.go`: the match, run by the host: modes, blinds and targets (`blindTarget`), lives, results, the auction, tomes, the market, hexes, Race mode, and the bots.
- `game/client.go`: what each game does for its own player: actions sent to the host, keeping its run in step with the match.
- `game/net.go`, `game/netlan.go`, `relaycore/`: the connection, messages, and LAN discovery.
- `game/table.go`, `game/shop.go`, `game/menus.go`: the screens. `game/cardart.go`: cards (rounded, drawn once into a cached image so they can tilt and scale), charms, hexes, tomes and hearts. `game/ui.go`: the widgets: warm palette, soft rounded panels and buttons (drawn as nine-slices of a cached disc, so they batch), and the Jersey 10/15 fonts (OFL, `game/assets/OFL-Jersey.txt`).
- `game/fx.go`: the swirling warm background (a Kage shader), cards moving on springs (`placeCard`: dealt from the deck, fanned and swaying in the hand, lifted when picked, flung away when played or discarded), floating numbers and screen shake.
- `game/audio.go`: synthesised sound effects and the music, "Ember Lounge": a swung 8-bar loop in A minor with electric piano, walking bass, brushed drums and a plucked melody that changes each time round.
- `android/`: the Android app (`com.theemeraldpig.desalote`). `mobile/` is the Go side of it, and `cmd/desalote` is the desktop entry point.

The host sends the whole `Match` to everyone whenever it changes. Guests send actions (`action` in `client.go`). Each game deals and plays its own player's cards from the seed the host gave it, and reports scores.

## Testing

```sh
xvfb-run -a go test ./game   # rules, scoring, hexes, auction/market, and whole bot matches in every mode
```

`TestBotMatchesFinish` plays every mode and length with bots and checks they all end. When changing scoring or `blindTarget`, check that bots still pass most early blinds and about half of the last ones.

To look at screens, run under Xvfb with the test hooks in `game/testhooks.go`:

```sh
go build -o /tmp/d ./cmd/desalote
HOME=/tmp/h DISPLAY=:99 DESALOTE_NOAUDIO=1 DESALOTE_SIZE=1280x720 \
  DESALOTE_SCRIPT="10:solo,11:bots=2,20:start,40:autopick,150:botmove,200:shot=/tmp/a.png,210:quit" /tmp/d
# phone layout: DESALOTE_TOUCH=1 DESALOTE_SIZE=873x393
```

Hooks include `solo`, `host`, `joinlan=ADDR`, `mode=N`, `bots=N`, `start`, `autopick`, `select=0:2`, `botmove`, `tap=X:Y`, `shop`, `tab=N`, `ov=N`, `givecharm=ID`, `givehex=ID`, `money=N`, `end`, `state`, `shot=PATH` and `quit`. Leave about 10 frames between a change and a screenshot.
