# Handoff note for the next Claude

Two Balatro mods (Lua, Steamodded/SMODS) by Josiah live in `/home/user`:

- `Randomizer/`: the main work in recent sessions, version 1.15.1, prefix `rnd`
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
| `settings.lua` | Config menu: a "Jokers" tab (`config_tab`) and a "Run" tab (`extra_tabs`). Toggles, plus Min/Max sliders and boxes for ranges. |
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

## Fixed in 1.7.0 (2026-11-04)

- The user wants a trigger **replaced**, never added. Any Joker with a "(Currently ...)" note used to keep its old text plus "Scores only when ...". Now every Joker's text has its condition swapped for the new trigger.
- Growing Jokers (`R.grows`: the description says "gain(s) ... Mult/Chips") now **grow** on the random trigger and score their "Currently" amount as normal. Their own growth is undone after each `calculate_joker` (`growth_snapshot`/`undo_growth`), and the amount is learned into `ability.rnd_gain`. Before it's learned, `guess_gain` reads it from `ability.extra` (Runner's `chip_mod`, Hologram's 0.25...). Types and behaviours don't apply to growers, and JokerDisplay keeps their own display.
- "Works once, then stops": the `decays` behaviour went down 0.25 per trigger to nothing. It now stops at normal and resets at end of round.
- `boosted()` was removed: it pumped up `x_mult`, so through the base game's generic XMult code nearly every Joker (Hallucination, Superposition...) looked like it scored and got a trigger.
- The user plays with the Multiplayer mod (seeded, The Order), so keeping seeds untouched matters.

## Fixed in 1.8.0 (2026-11-04)

The user said triggers still added conditions and many Jokers were unaffected. `measure` only gets past suit, rank, face, hand type and chance. Jokers that check anything else (Mystic Summit's 0 discards, Half Joker's card count, Acrobat's final hand, Flower Pot's suits, Card Sharp, Blackboard...) gave nothing on their new trigger, or never got a trigger at all. `base_effect` now reads the amount from the Joker's numbers (`t_mult`, `t_chips`, `x_mult`, `extra.*mult/chips`; a bare `extra` number uses the description's "X#1# Mult" / "+#1# Chips"). `fire`, `R.increment` and `triggerable` fall back to it.

## 1.9.0 (2026-11-04): every Joker gets a trigger

The user wants **all** Jokers affected. `R.category` (saved as `ability.rnd_kind`) sorts each Joker:
- `grow`: grows on the trigger (see 1.7.0).
- `dollar`: end-of-round money (Golden, Cloud 9, Rocket, Satellite, Delayed Gratification, or any center with `calc_dollar_bonus`). `Card:calculate_dollar_bonus` returns nil for them, and the trigger pays `dollar_effect` instead. No type or behaviour.
- `score`: as before.
- `action`: anything else (creating cards, sell value, levelling hands...). `ability.rnd_action` is the step it acts in (`ACTION_KEYS`). Base-game ones are in `KNOWN_ACTIONS`; others learn it the first time they act (events, money, a non-cosmetic result or a changed number). That step is then blocked, and on the trigger `replay` builds a context for that step. It probes quietly to find a rank, or a forced enhancement, that makes the Joker act, then runs it for real (`replaying` forces suit, face and chance). Until an action Joker is known, its card text stays the same (`R.shows_trigger`). Passive Jokers (Four Fingers, Juggler, Showman...) never act, so they never change.
- Known gaps: action Jokers whose condition can't be forced (DNA's 1-card first hand, Faceless's 3 faces, Vagabond's $4, Superposition's Ace) only act when that condition is also true. Invisible Joker, Diet Cola and Luchador act on the trigger instead of when sold.
- A small Lua simulation (lupa) was used to check the action, dollar, grow and score paths. It's in the session scratchpad and not kept.
- 1.9.1: the Jokers whose condition can't be faked now have their effect written out in `OVERRIDES`, at the user's request. On the trigger: DNA copies the card into the deck and hand, Faceless pays $5, Trading Card pays $3 (no destroy, and its text drops "destroy it and"), Vagabond and Superposition create a Tarot.

## 1.10.0 (2026-11-04): ranges in the Config tab

Each random range (Joker amounts, shop prices, Blind scores, poker hands) has `<key>_min`/`<key>_max` in `config.lua`, read by `range()` in `rolls.lua` (missing values fall back to the old defaults; a min above the max is swapped). `settings.lua` gives each Min/Max a slider (0-10) and a text box that takes any number, with no limit (1.10.1, at the user's request). `G.FUNCS.rnd_slider` replaces the slider's per-frame `slider` function: it keeps the slider, the value and the box in sync. It writes the box's letters through `args.text.letters`, or finds them through the box's `<id>_letter_1` element. The Balatro UI functions (`create_slider`, `create_text_input`, `G.FUNCS.slider`) were written from memory of the base game, not read, since its source isn't available here. If the tab errors, check those first.
- 1.11.0: everything can go negative (user request). Joker amounts aren't kept at a minimum of 1 any more; type conversion only clamps positive amounts; `base_effect`, `guess_gain` and `dollar_effect` take negative amounts; shop prices can be 0 or negative, and then the sell value follows the price so buying and selling can't make money forever. The base game's own card text can still read "+-4 Mult".
- 1.11.1: fixed a crash when hovering a Joker (from 1.9.0): `trigger` in the `generate_card_ui` hook had become `true` instead of the trigger.
- 1.12.0: the user wants every Joker to fire each time its trigger hits. Blueprint and Brainstorm (`COPIERS`, kind `copy`) used to run their normal once-per-hand copying when the trigger hit. Now their own copying steps are blocked, and each hit gives the copied Joker's effect (`copied_effect`: its numbers, its action, its override or its retriggers). Retriggers: `repetition_hit` also lets held cards be retriggered (Mime) on held-card or whole-hand triggers, and `capped` limits retriggers from a card trigger to 1 per card, as the user asked.
- 1.12.1: the user wants card retriggers to work as normal (the 1.12.0 cap is gone; `capped` now does nothing). Instead, a Joker itself can be retriggered (a `retrigger_joker` pass) at most once per step and card (`retriggered_already`, reset at `before`).
- 1.12.2: what the user actually wants: any Joker can be retriggered without limit (the 1.12.1 limit is gone). A retrigger Joker can fire any number of times, but never because of what it retriggered (`retriggered_by_itself`: its own `retrigger_joker` pass, or a repetition check on an already-retriggered card).
- 1.13.0: retrigger Jokers now retrigger the cards they normally work on (every played card for Hack, every held card for Mime) their normal amount × `hits_in_hand`. That counts matching scoring cards for played-card triggers (retriggers don't count), matching held cards for held-card triggers, and 1 for a whole-hand trigger. Text: action Jokers also lose "for each..."/"per..." (Mail), and a "#n# in #m# chance this card is destroyed" downside clause is dropped from the text (Gros Michel still has the downside).
- 1.13.1: Photograph text kept "First played face card" (new SUBJECTS patterns). Played-card measuring and `measure_repetitions` now look at the card on its own (`scoring_hand = {card}`), so "first card" checks (Photograph, Hanging Chad) pass for any card. Retrigger Jokers whose own condition is game state (Dusk's final hand) fall back to 1 retrigger (`is_retrigger_joker`: the description starts with "Retrigger"), and are never sorted as `action`.
- 1.13.2: the user wants every effect to do exactly what the rewritten card says. Retrigger Jokers (description starts with "Retrigger") no longer run their own code: in `retriggers` every played card, or every held card if the text says "held in hand", gets the "#1# additional times" amount (`ability.extra`, or 1 if the text has no such phrase) × trigger hits. Their text reads "Retrigger all played cards [#1# additional times] when ...".
- 1.14.0: Legendary Jokers (rarity 4) get `ability.rnd_trigger2` (`R.roll_second`, never `first_hand`/`final_hand`, never the same as the first; Legendaries in an existing run get one in `Card:update`). `triggers_of`/`hit_count` count every trigger that hits, and each hit runs the effect once (`repeated`, chained with `SMODS.merge_effects`). Retriggers add both triggers' hits. Text: "... when A, and also when B".
- 1.15.0: the config menu has a "Jokers" tab (random triggers, type, scaling, Joker values) and a "Run" tab (shop prices, Voucher values, Tarot & Spectral values, Blind scores, poker hands). `trigger_only` is gone; `triggers = false` makes every hook pass straight through (`R.triggers_on`). `on()` now means "not set to false". Joker values: a bare `extra` number is now scaled when the description shows it's an amount (`bare_extra_kind`: "+#1#", "X#1#", "$#1#"; never "#1# in #2#" odds), and Talisman numbers are read through `plain()`; both were skipped before. Vouchers and Tarot/Spectral cards scale their `ability.extra` once (`roll_card_numbers`; whole numbers stay whole, odds are skipped). While they're shown, applied (`apply_to_run`) or used (`use_consumeable`), `R.with_rolled_center` swaps their center's `config.extra` for the rolled value, because the base game reads the center.
- 1.15.1: crash with Multiplayer + Amulet once money passed $999. Multiplayer keeps `PVP/MP.GAME.real_money` as text and reads it with `tonumber`; an Amulet big number made it "1,952", `tonumber` gave nil, and `functions/common_events.lua:100` crashed. Fix: `tonumber` also reads numbers written with thousands commas (only strings that would otherwise give nil); `ease_dollars` turns big-number amounts into plain numbers and strips commas from `real_money`.
