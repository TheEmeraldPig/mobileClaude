#!/bin/sh
# Downloads the BepInEx and UnityEngine libraries the mod compiles against into libs/.
# They are not committed: they belong to BepInEx and Unity.
set -e
cd "$(dirname "$0")"
mkdir -p libs .tmp
if [ ! -f libs/BepInEx.dll ]; then
  curl -sSL -o .tmp/bepinex.zip https://github.com/BepInEx/BepInEx/releases/download/v5.4.23.2/BepInEx_win_x64_5.4.23.2.zip
  unzip -o -q -j .tmp/bepinex.zip BepInEx/core/BepInEx.dll -d libs
fi
if [ ! -f libs/UnityEngine.dll ]; then
  # A single-file UnityEngine.dll (not the split modules), so the mod loads in old and new Unity versions.
  curl -sSL -o .tmp/unity.nupkg https://api.nuget.org/v3-flatcontainer/unity3d.unityengine/2018.1.6-f1/unity3d.unityengine.2018.1.6-f1.nupkg
  unzip -o -q -j .tmp/unity.nupkg lib/UnityEngine.dll -d libs
fi
rm -rf .tmp
