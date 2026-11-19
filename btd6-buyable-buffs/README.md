# Buyable Buffs (BTD6 mod)

A [BTD6 Mod Helper](https://github.com/gurrenm3/BTD-Mod-Helper) mod. Select any placed tower and a
**BUFFS** button appears at the top of the screen. It opens a shop where you spend in-game cash on buffs
for that tower. Bought buffs:

- last for the rest of the game,
- stay on after upgrades, hero level-ups and paragon degree changes (they're put back on the new model),
- are saved with the tower, so they come back when you continue a saved game,
- stack with the normal game buffs (Village, Alchemist and so on), which still work on top.

## What you can buy

**Stat buffs** cover every stat that a buff in the game changes: attack speed, range, damage, pierce,
pierce %, MOAB/ceramic/boss/fortified/lead bonus damage, burn/poison damage, projectile speed, distance and
size, ability cooldown, income, camo detection, popping lead/black/white/frozen/purple, and ignoring
line of sight.

**Tower, hero and power buffs** are buyable versions of the buffs other things give in the normal game.
They're built from the same stat effects:

- Towers: Village Range, Jungle Drums, Primary Training, Primary Mentoring, Radar Scanner, MIB,
  Homeland Defense, Call To Arms, Acidic Mixture Dip, Berserker Brew, Stronger Stimulant, Overclock,
  Ultraboost, Shinobi Tactics, Trade Empire, Banana Central, Druidic Reach, Embrittlement
- Heroes: Rallying Roar, Heat It Up, Nature's Ward, Artillery Command, Bank Hack, UAV Uplink,
  Blood Sacrifice, Naval Tactics, Sharpening Stone, Jar of Pickles, See-Through Glasses, Spirit Brew
- Powers and other: Monkey Boost, Paragon Power Totem, and some Monkey Knowledge-style buffs

The numbers are close to the game's but not exact copies, and buffs that only last a while in the
normal game (Homeland Defense, Overclock, Monkey Boost...) are permanent here, so they cost more. Each
row in the shop lists exactly what the buff does. To add a buff, or change its price or effects, edit
`Buffs.cs`. The effects themselves are in `Effects.cs`.

On/off buffs (camo, popping a bloon type) can be bought once. Stacking buffs can be bought up to
**Max Stacks** times (default 5; Ultraboost goes to 10 and Shinobi Tactics to 20). The mod's settings in
Mod Helper also have **Price Multiplier** and **Prices Rise With Stacks** (on by default: the nth stack
costs n × the base price).

## Installing

1. Install [MelonLoader](https://github.com/LavaGang/MelonLoader/releases) into BTD6, then put
   [`Btd6ModHelper.dll`](https://github.com/gurrenm3/BTD-Mod-Helper/releases) in the game's `Mods` folder.
2. Start BTD6 once, wait for the main menu, and close it.
3. Double-click **`Install.bat`** in this folder. It finds the game, downloads the .NET SDK if you don't
   have it, builds the mod and puts `BuyableBuffs.dll` in `Mods`.

## Building by hand

You need BTD6 with MelonLoader and BTD6 Mod Helper installed, plus the .NET 6 SDK or newer.

```bash
dotnet build -p:BloonsTD6="C:\Program Files (x86)\Steam\steamapps\common\BloonsTD6"
```

If you leave out `-p:BloonsTD6=...`, the default Steam path above is used. The build copies
`BuyableBuffs.dll` into the game's `Mods` folder. You can also make an empty mod with Mod Helper's
in-game **Mod Sources → Create Mod** and drop these `.cs` files into it.

## Notes

- Run the game at least once with MelonLoader first, so that `MelonLoader/Il2CppAssemblies` exists.
- `ModHelperData.WorksOnVersion` is set to 50.0. Ninja Kiwi renames internals now and then, so if a
  game update breaks the build, the usual culprits are field names in `Effects.cs`, such as
  `AbilityModel.Cooldown` or the `DamageModifierForTagModel` constructor.
- Only use this in single-player. Don't use it in co-op, races, odysseys or other leaderboard modes, where
  modding can get your account flagged.
