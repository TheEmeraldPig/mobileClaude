# Handoff note for the next Claude

Two Balatro mods (Lua, Steamodded/SMODS) by Josiah live in `/home/user`:

- `Randomizer/`: the main work in recent sessions, version 1.6.0, prefix `rnd`
- `SimplyJokers/`: an older, larger mod, v1.3.1 (zip: `SimplyJokers-v1.3.1.zip`). Recent work there was only `nailed.lua`.

Other files: `tarot_fusions.md`, `spectral_fusions.md`, `enhancement_combos.md` (design lists for SimplyJokers fusions) and `tools/`.

## Latest state (2026-11-03)

- **Stack overflow fixed in 1.5.2, not yet tested in game.** The user reported `stack overflow` (with JokerDisplay 2.0.8, Handy 2.0.6, Amulet 3.6.1). The traceback was `rolls.lua:116` → `R.roll_trigger` → `triggerable` → `measure` (`triggers.lua`), then `copy_table` recursing thousands of times. `measure` deep-copied a Joker's result table, and that table holds a `card` whose `parent`/`children` links loop back. The fix is `copy_chain`, which copies only the `ret` → `ret.extra` chain that `take_scoring` edits. **Never `copy_table` anything that can hold a Card.** 1.5.3 also replaced the remaining `copy_table` calls on Joker state with `safe_copy` (shares tables that have a metatable, such as Cards, and handles loops). The first thing to ask the user is whether 1.5.2 still crashes.
- The user asked for the mods to be put in a GitHub repo (the only one this account can reach is `TheEmeraldPig/mobileClaude`). Adding it was blocked by a permission check, so it isn't done. If you have repo access, copy `Randomizer/`, `SimplyJokers/`, this note and the `.md` design files into it, commit, and push.
- The user prefers things to be **simple**. They pushed back hard on overbuilt card text and display. The `trigger_only` setting must change *only* triggers.

Otherwise, **ask the user what they want next**; don't continue from old requests. The original wording of earlier requests was lost to context compaction, so this note describes what the code does, not what was asked for.

## Randomizer: file map

| File | What it does |
|---|---|
| `main.lua` | Creates the global `Randomizer = { mod = SMODS.current_mod }`, then loads triggers → rolls → settings → joker_display. Logs the version. |
| `config.lua` | Defaults: `trigger_only=false`, `behaviours`, `types`, `numbers`, `prices`, `blinds`, `hands` are all `true`. |
| `triggers.lua` (~40 KB, the core) | `R.TRIGGERS` lists 33 triggers, each the condition of a base-game Joker (the user wants triggers swapped for ones real Jokers have, never made-up conditions). While measuring, `pseudoseed`, `pseudorandom` and `SMODS.pseudorandom_probability` always succeed and use none of the seed, so a Joker's own odds never become an extra condition. `kind` is `"play"` for each scored card, `"hand"` for each held card, or `"main"` for once per hand. It hooks `Card:calculate_joker`: a Joker's scoring numbers, including retriggers, only come on its random trigger, and everything else it does works as normal. A Joker's numbers are found by running it once in the scoring step with events and money switched off, then restoring its state. Each Joker gets a random type (chips, mult, xmult or xchips; converted through `POINTS`/`FROM_POINTS`) and a random behaviour (normal, scales, resets, stacks or decays). It also rewrites card text by hooking `init_localization` and `generate_card_ui`; rewritten text is cached per Joker key and trigger. Public functions: `R.trigger_of`, `R.roll_trigger`, `R.note_passive`, `R.next_effect`, `R.kind_of`, `R.format_total`, `R.format_increase`, `R.increment`, `R.convert`. |
| `rolls.lua` | Joker numbers scale ×0.5–2 (`R.roll_numbers`; only keys that look like mult, chip, dollar or money). Shop prices ×0.5–1.5 per card (`ability.rnd_price`). Blind scores ×0.75–1.5 for each Ante and Blind, cached in `G.GAME.rnd_blind_factors`. Poker hand `s_chips`/`s_mult` ×0.5–1.5 at the start of a new run. The `Card:update` hook rolls numbers once and triggers once; a Joker with no trigger yet retries once per round. |
| `settings.lua` | Config tab with one toggle per option. `trigger_only` turns off numbers, types and behaviours. |
| `joker_display.lua` | JokerDisplay support. While JokerDisplay's functions run, each randomized Joker's key temporarily points at one shared `DEFINITION`, then the originals are restored. A depth counter prevents nesting. The display shows the next effect, how many times the current hand would trigger it, and the trigger text. |

Conventions to keep:
- Every roll uses `pseudorandom`/`pseudoseed`, so seeded runs stay the same.
- Support Talisman big numbers: use `to_number`, and `:floor()` inside a `pcall`.
- Hook a function by saving the original as `local x_ref = Fn` and calling it from the wrapper. Wrappers that change global state use `pcall`, restore the state, then `error(a, 0)`.
- `on(option)` exists in both `rolls.lua` and `triggers.lua`. If the `trigger_only` logic changes, update both copies.
- Per-card state is stored in `card.ability.rnd_*`: `rnd_rolled`, `rnd_factor`, `rnd_trigger`, `rnd_trigger_tried`, `rnd_type`, `rnd_behaviour`, `rnd_total`, `rnd_price`.

## SimplyJokers: `nailed.lua`

- Adds the `sj_nailed` sticker: rate 0, `needs_enable_flag`, so it never shows up on shop cards.
- Adds the "Nailed" Tarot (cost 3). It nails exactly one highlighted hand card or Joker.
- A Nailed card can't be debuffed, through `SMODS.current_mod.set_debuff` returning `"prevent_debuff"`.
- It can't be flipped face down in hand, play or the Joker area, because of the `Card:flip` hook. A `Card:update` hook turns back up cards that were drawn face down, for example by The House.
- Art is `assets/*/nailed.png`, 71×95: the card is at (0,0) and the sticker at (1,0).
- Other SimplyJokers files (`fusion.lua`, `fusion_specials.lua`, `main.lua`) are large and weren't changed in these recent sessions.

## Other notes

- Art was generated with a Python script at `/tmp/claude-0/-home-user/ff149113-ed13-511c-874a-9cf55b95ca00/scratchpad/gen/specials_gen.py`. That scratchpad may be gone in a new session. Read the script before reusing it.
- Nothing has been tested inside the game from this container; there's no Balatro here. Check behaviour by reading the code, and ask the user for `lovely`/Steamodded logs when debugging. Randomizer logs lines like `"<key> -> trigger '<key>'"` (the first 40 Jokers) and `"Run detected..."`.
- `/home/user` isn't a git repo, so nothing is committed or pushed. Files changed here only reach the user through the zips or if they copy them out.

## Fixed in 1.6.0 (2026-11-04)

The user reported that a Joker's new trigger worked like an added condition and made it harder to score. The log showed the own effect *was* being blocked; the real cause was that measuring still rolled the Joker's own odds (Bloodstone's 1 in 2), so the trigger only paid out half the time. The user also wants every trigger to be one a real Joker has: made-up ones (seed, edition, $20, deck size...) were replaced. Jokers saved with a removed trigger roll a new one. 1.5.4's scoring log lines are still there.
