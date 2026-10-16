# mobileClaude

## sandbox/ (the game "Mote")

- A single-file browser game in `sandbox/index.html`, played mostly on a phone in landscape. Keep phone (touch) and computer (keyboard and mouse) layouts working; `IS_TOUCH` picks between them.
- **World generation is seeded.** Any terrain or world feature must follow the rules in `sandbox/README.md` under "Adding terrain features". The short version: no `Math.random()` in generation, add features as `GEN_PASSES`, and handle features that cross chunk borders.
- **UI style comes from the mote sprite:** dark outline `--o`, violet body `--b`, light top-left edge `--l`, shaded bottom-right edge `--s`, gold accent `--t`, 3px pixel bevels via `.btn`, `.px` and `.sheet`, and the Pixelify Sans and Silkscreen fonts. New menus and buttons should reuse these classes.
- Test with Playwright using the preinstalled Chromium (`executablePath: '/opt/pw-browsers/chromium'`), in both a desktop viewport and a phone context (`hasTouch: true, isMobile: true`).

## mote/ (native version of Mote: Windows .exe, Android .apk, Steam)

- A Go + Ebitengine port of the game. See `mote/README.md`.
- This native version is now the main one; the web version in `sandbox/` is no longer kept in sync and builds different worlds.
- `game/world_test.go` must keep passing: a seed must always build the same chunks. If you change world generation on purpose, update its reference numbers and say so (every seed's world changes).
- Lighting is a flood fill in `game/render.go` (levels 0-64). New light-giving blocks go in `blockLight()` in `world.go`.
- New block types: add an id, a colour in `blockRGB`, `isSolid`/`blocksLight` handling and a `blockItem` entry in `world.go`, and sim handling in `sim.go` if they move or grow.
- The game is a roguelite: per-run state lives in `save.json` (deleted when a run ends), permanent upgrades in `meta.json` (`roguelite.go`). New per-run state must go in `serializeState`/`restoreState` in `save.go`.
- Every push to `mote/` builds and releases `Mote.exe`, `Mote-steam.exe` and `Mote.apk` through `.github/workflows/mote.yml`.
- Check layouts by running under Xvfb with the test hooks in `game/testhooks.go` (`MOTE_SCRIPT`, `MOTE_TOUCH=1`, `MOTE_NOAUDIO=1`).

## trading-advisor/ (day-trading journal and advisor)

- The main version is the website in `web/`, published as a private claude.ai artifact: https://claude.ai/artifact/VUNk24GVvv5xewtmwJEMHq. Republish it by passing that URL with `file_path` `web/index.html` and `files` for `style.css`, `logic.js`, `app.js` and the three `samples/*.csv`. Omit `capabilities` on redeploys so the stored ones (owner-only db rules, assets, downloads, user) stay.
- The user's trades live in that artifact's database (`trades/<id>`, `settings/main`), readable with the ArtifactData tool. Never copy them into this public repo. A trade with a `trader` name belongs to another person: keep those out of the user's own stats, limits and pre-trade checks.
- It must never connect to a brokerage or place orders. The user is a beginner: keep wording plain and `README.md` steps simple.
- Market context lives in the artifact's `context/<tradeId>` collection. To refresh it: export trades with ArtifactData `list` (`out_dir`), run `python3 trading-advisor/tools/market_context.py <out_dir>/trades <ctx_dir> --only-missing <existing ctx>` (needs `ALPACA_DATA_KEY_ID`/`ALPACA_DATA_SECRET_KEY` and `data.alpaca.markets` allowed), then save the files with ArtifactData `batch` (`file_path`). Never write made-up market data into the user's journal.
- Placing orders with a broker was blocked by Claude Code's safety system in this environment; there is no order-placing code in the repo. Don't add any unless the user has deliberately changed their permission settings to allow it.
- Reading the user's TradingView (watchlist, charts, Paper Trading history) is described in `.claude/skills/tradingview/SKILL.md`. It needs a session with browser access (Claude desktop app or Claude in Chrome) and is read-only.
- All trading math is in `web/logic.js`; run `node tests/web_logic_test.js` after changing it. The page can't use `alert`/`confirm`/`prompt` or plain download links (use the inline confirm and the `downloads` capability).
- The older offline version (`app.py`, Python standard library, SQLite in gitignored `data/`) is no longer kept in sync. Its tests: `python -m unittest discover tests`.
