# Buyable Buffs (BTD6 mod)

A [BTD6 Mod Helper](https://github.com/gurrenm3/BTD-Mod-Helper) mod that puts the game's tower buffs in the
tower shop (Support section). Buy one like a tower and drop it on one of your towers, and that tower keeps the
buff for the rest of the game.

## Buffs and levels

Each buff follows an upgrade path. Dropping the same buff on the same tower again raises it one level:

| Buff | Levels |
| --- | --- |
| Village Drums | Jungle Drums, Primary Training, Primary Mentoring, Primary Expertise |
| Village Intelligence | Radar Scanner, Monkey Intelligence Bureau, Call to Arms, Homeland Defense |
| Alchemist Brew | Acidic Mixture Dip, Berserker Brew, Stronger Stimulant, Permanent Brew |
| Overclock | Overclock, Ultraboost |
| Shinobi Tactics | 1 to 5 stacks |
| Energizer | Energizer |
| Trade Empire | Favored Trades, Trade Empire |
| Elite Sniper | Elite Sniper |
| Banana Central | Banana Central |
| Rallying Roar (Pat Fusty) | Rallying Roar, Rawr+ |
| Heat It Up (Gwendolin) | Heat It Up, +range, +damage |
| Nature's Ward (Obyn) | levels 4, 8 and 15 |
| Bomb Expert (Striker Jones) | Bomb Expert, Artillery Command, Artillery Command 2 |
| Sharpening Stone (Geraldo) | 1 level |
| Jar of Pickles (Geraldo) | 1 level |

Every level costs the same: level 1 is $1,000, level 2 is $2,000, and so on. The shop price covers level 1, and
dropping it again charges the rest.

The numbers come from the game itself. The tower's info screen lists each level with its exact percentages,
for example "+15% attack speed, +10% range". To change the buffs or prices, edit the list at the top of
`Buffs.cs`.

## How the buff is given

The buff is a mutator put directly on the tower, the same way Geraldo's items, brews and Village auras buff a
tower. It shows the buff's icon on the tower, is re-applied after every upgrade, never runs out, and is saved
with the game. The item you dropped disappears.

When the game starts, the MelonLoader console lists every buff and level it found, with their effects.

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

- `Buffs.cs`: the list of buffs and their levels, and how the game's numbers become stat changes.
- `BuffTower.cs`: the shop item.
- `BuffMutator.cs`: the buff on the tower.
- `Placement.cs`: what happens when you drop a buff on a tower.
- `Reflect.cs`: reads game fields by name, so a renamed field in a game update is skipped instead of crashing.

Only use this in single-player. Using mods in co-op, races, odysseys or other leaderboard modes can get your
account flagged.
