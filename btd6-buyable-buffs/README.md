# Buyable Buffs (BTD6 mod)

A [BTD6 Mod Helper](https://github.com/gurrenm3/BTD-Mod-Helper) mod that puts every buff in the game in the
tower shop. Buy one like any tower, then drag it onto one of your towers, and that tower gets the real buff
for the rest of the game.

- **Every buff that one thing gives to another tower**: Monkey Village buffs (Jungle Drums, Primary Training,
  Radar Scanner, MIB...), Alchemist brews, Shinobi Tactics, Energizer, Sub Commander, Trade Empire, Elite Sniper,
  Banana Central, Call to Arms, Homeland Defense, Overclock and Ultraboost, hero buffs (Pat Fusty, Obyn, Striker,
  Gwendolin, Brickell, Etienne, Benjamin...), Geraldo's items, and any newer ones. The mod isn't working from a
  list I typed. When the game starts, it goes through each buff icon in the game and finds the upgrade, hero
  level or item that gives that buff, so new buffs from game updates show up too.
- **The shop icon is the buff's own icon**, the one you see on a buffed tower.
- **Price** is the price of the upgrade that gives the buff (for heroes, the hero's price; for Geraldo
  items, the item's price). The usual difficulty price changes apply.
- **The description** is the game's own text for that upgrade or item. It's shown in the upgrade/info screen
  only.
- **Exact buff, kept forever.** The bought buff is the game's own buff, with the same numbers, the buff icon
  on the tower and the same stacking. It stays through upgrades and is saved with the game.
- **Stacking follows the game.** Buffs that stack in the game (like Shinobi Tactics or Ultraboost) can be
  dropped on the same tower again, up to the game's limit. Ones that don't stack are refused, and you get your
  money back.

## How it works

A bought buff is a hidden, unselectable "carrier" that sits on the tower you dropped it on. It holds the
same behavior the real upgrade uses, shrunk so it only reaches that tower. The game's own code applies the
buff. The details for each kind of buff:

- **Auras** (Village buffs, Shinobi, Energizer...) are copied as they are. Tower-type limits are removed, so
  the buff works on whatever tower you choose.
- **Ability buffs** (Call to Arms, Homeland Defense, Rallying Roar...) are cast on that tower again
  whenever they wear off, so they're always active.
- **Brews** are thrown at that tower over and over by the hidden carrier.
- **Overclock / Ultraboost** normally target a tower you pick, so their attack speed boost is given as a
  permanent aura with the same numbers and icon.
- **Geraldo's items** (Sharpening Stone, Jar of Pickles, Fertilizer) have their numbers moved onto the
  matching aura.

If you drop a buff somewhere with no tower, you get your money back. If you sell a buffed tower, its buffs
are sold with it.

When the game starts, the MelonLoader console lists every buff that was found, its source, price and
whether it stacks. It also lists the icons that were skipped because nothing gives that buff to another
tower.

## Installing

1. Install [MelonLoader](https://github.com/LavaGang/MelonLoader/releases) into BTD6, then put
   [`Btd6ModHelper.dll`](https://github.com/gurrenm3/BTD-Mod-Helper/releases) in the game's `Mods` folder.
2. Start BTD6 once, wait for the main menu, and close it.
3. Either copy a prebuilt `BuyableBuffs.dll` into `Mods`, or double-click **`Install.bat`** in this folder. It
   builds the mod against your copy of the game and puts it in `Mods`.

## Building by hand

```bash
dotnet build -p:BloonsTD6="C:\Program Files (x86)\Steam\steamapps\common\BloonsTD6"
```

## Files

- `BuffTower.cs`: one shop entry per buff icon in the game.
- `Discovery.cs`: finds which upgrade, hero level or item gives each buff, and its name, description and price.
- `Carrier.cs`: builds the hidden carrier and attaches it to the tower you drop it on.
- `Reflect.cs`: reads game fields by name, so a renamed field in a game update is skipped instead of crashing.

Only use this in single-player. Using mods in co-op, races, odysseys or other leaderboard modes can get your
account flagged.
