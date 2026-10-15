# Mote

A falling-sand pixel game. Everything is in `index.html`, with `manifest.webmanifest`, `sw.js` and the icons making it an installable app.

## World generation

The world is seeded and endless left and right. It is built in chunks (`CW` = 128 cells wide, `H` = 600 cells tall) as you travel.

- Only a window of `NCH` chunks around the mote is loaded, held in `grid`. When the mote moves two chunks from the middle, `shiftWindow()` slides the window. Chunks that leave are unloaded. If they changed (dug, built on, bombed, or simply simulated), they are saved compressed in `saved`. Chunks that enter are restored from that save or generated fresh from the seed.
- Everything in the game works in window-local coordinates. `shiftWindow()` moves the player, camera and particles by the same amount, so nothing else needs to know about chunks. `worldX()` converts a local cell x to a world x.
- `createWorld(seed)` starts a world. The same seed text always builds the same world on every device.

### Adding terrain features

Every terrain addition must follow these rules, or seeds stop matching:

1. **No `Math.random()` in generation.** Use `hash01(a, b, salt)`, `noise1(x, scale, salt)` or `chunkRng(chunkIndex, salt)`.
2. **Only the seed and world coordinates.** A chunk's contents must never depend on screen size, load order or what is already loaded.
3. **Add a pass to `GEN_PASSES`.** Each pass fills one chunk buffer. A feature that can cross chunk borders, such as a cave or a tree, must also generate the copies of that feature that start in neighbouring chunks, and only write cells inside the chunk being built. `carveCaves()` is the example.
4. **Give each pass its own salt number** so features don't line up with each other.
5. **Changing a pass, `CW` or `H` changes every seed's world.** That's fine during development, but say so when you do it.

New block types need an id next to `SAND`, `WATER` and `STONE`, a colour in `RGB`, and handling in `step()` if they move.

## Saving and live updates

- The game autosaves to `localStorage` (`mote-save`) every 20 seconds and whenever the app is hidden. The save holds the seed, changed chunks, the mote and the inventory.
- **Installed app (GitHub Pages):** every 20 seconds the page re-downloads itself and compares a fingerprint. When it changes, the game saves, reloads into the new version and drops you back where you were. Requests with `?v=` skip the service worker cache.
- **Claude artifact link:** updates arrive through `window.claude.hot`. `snapshot()` hands the state to the new version, which boots through `boot(data)`.
- **When you add new state** (new items, player stats and so on), add it to `serializeState()` and `restoreState()` so it survives updates and saves.
