# Magicite Mod

A [BepInEx 5](https://github.com/BepInEx/BepInEx) plugin for the game Magicite (SmashGames, 2014). It adds an in-game menu:

- **Game:** change the game speed (0.1x to 5x). F2 switches between normal and fast speed.
- **Objects:** scan the current level for the game's objects (player, enemies, items, ...), open one and change its numbers, true/false values and text. **Lock** keeps a value fixed, for example to keep health full. **x10** multiplies a number.
- **Locks:** everything that is locked, with buttons to unlock.
- **Dump:** writes every class in the game's code, with its fields and methods, to `BepInEx/MagiciteMod-classes.txt`.

The mod finds the game's objects by reflection, so it doesn't depend on Magicite's class names. The class list from **Dump** can be used to add features that hook the game's code directly, such as new items or spells.

## Install

1. Install BepInEx 5 into Magicite: the [BepInExPack for Magicite](https://thunderstore.io/c/magicite/p/BepInEx/BepInExPack_Magicite/) (copy the contents of its `BepInExPack` folder next to `Magicite.exe`), or the x86/x64 `BepInEx_win_*_5.4.x.zip` from BepInEx's releases.
2. Start the game once so BepInEx makes its folders, then close it.
3. Download `MagiciteMod.dll` from the latest `magicite-mod-*` release on GitHub and put it in `Magicite/BepInEx/plugins/`.
4. Start the game and press **F1**.

Keys and the fast speed can be changed in `BepInEx/config/com.theemeraldpig.magicitemod.cfg`.

## Tips

- In a level, open **Objects**, press **Scan scene** and type `player`, `hero`, `stat` or `health` in the filter.
- Some values are worked out again by the game every frame from others (like max health from gear). If a change doesn't stick, look for the value it comes from, or lock it.
- Changing the wrong thing can break a run or a save. Back up your save before trying things.

## Building

```sh
./fetch-libs.sh          # downloads BepInEx.dll and UnityEngine.dll into libs/
dotnet build -c Release  # makes bin/Release/net35/MagiciteMod.dll
```

It targets .NET 3.5 and a single-file `UnityEngine.dll`, so it loads in old Unity versions like the one Magicite uses. Every push to `magicite/` builds the DLL and publishes it as a GitHub release through `.github/workflows/magicite.yml`.
