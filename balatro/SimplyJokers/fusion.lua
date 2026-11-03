--- Fusion: a consumable type whose one card, Fuse, merges two Jokers, two consumables
--- of the same set, or two playing cards into one.
---
--- Fused Jokers and consumables store their parts as { key, ability }. To run a part,
--- the fused card temporarily takes on that part's center and ability. Events queued
--- while a part runs are wrapped so they also run as that part, because vanilla code
--- often reads self.ability from inside its events.

local SJ = SimplyJokers

-- Lets a playing card count as several enhancements at once (Wild + Steel...)
SMODS.current_mod.optional_features.quantum_enhancements = true

----------------------------------------------------------------------
-- Modes: state that is entered while a part runs and re-entered by its events
----------------------------------------------------------------------

local active_modes = {}

local function run_in_modes(modes, fn, ...)
    local restores = {}
    for i, mode in ipairs(modes) do
        restores[i] = mode.enter()
        active_modes[#active_modes + 1] = mode
    end
    local ok, a, b = pcall(fn, ...)
    for i = #modes, 1, -1 do
        table.remove(active_modes)
        restores[i]()
    end
    if not ok then error(a, 0) end
    return a, b
end

local add_event_ref = EventManager.add_event
function EventManager:add_event(event, ...)
    if #active_modes > 0 and type(event) == "table" and type(event.func) == "function" then
        local modes = {}
        for i, mode in ipairs(active_modes) do modes[i] = mode end
        local func = event.func
        event.func = function(...) return run_in_modes(modes, func, ...) end
    end
    return add_event_ref(self, event, ...)
end

local function swap_mode(card, entry)
    return {
        enter = function()
            local ability, center, center_key = card.ability, card.config.center, card.config.center_key
            card.ability, card.config.center, card.config.center_key = entry.ability, G.P_CENTERS[entry.key], entry.key
            return function()
                card.ability, card.config.center, card.config.center_key = ability, center, center_key
            end
        end,
    }
end

-- While a fused Joker's part runs, money can't be lost
local no_loss = 0
local no_loss_mode = {
    enter = function()
        no_loss = no_loss + 1
        return function() no_loss = no_loss - 1 end
    end,
}

local ease_dollars_ref = ease_dollars
function ease_dollars(mod, ...)
    if no_loss > 0 and type(mod) == "number" and mod < 0 then return end
    return ease_dollars_ref(mod, ...)
end

local function flatten_parts(card, fused_key)
    local parts = {}
    if card.config.center.key == fused_key then
        for _, p in ipairs(card.ability.extra.parts) do parts[#parts + 1] = copy_table(p) end
    else
        local ability = copy_table(card.ability)
        -- Stickers are bad parts; slot changes belong to the fused card's own edition
        ability.eternal, ability.perishable, ability.perish_tally, ability.rental = nil, nil, nil, nil
        if ability.card_limit then ability.card_limit = 0 end
        if ability.extra_slots_used then ability.extra_slots_used = 0 end
        parts[1] = { key = card.config.center.key, ability = ability }
    end
    return parts
end

local function part_names(parts)
    local names = {}
    for _, p in ipairs(parts) do
        local center = G.P_CENTERS[p.key]
        local ok, name = pcall(localize, { type = "name_text", set = center and center.set or "Joker", key = p.key })
        name = (ok and name and name ~= "ERROR") and name or p.key
        names[#names + 1] = (p.copies or 1) > 1 and (name .. " x" .. p.copies) or name
    end
    return table.concat(names, " + ")
end

local function add_part_tooltips(info_queue, parts)
    for _, p in ipairs(parts) do
        if G.P_CENTERS[p.key] then info_queue[#info_queue + 1] = G.P_CENTERS[p.key] end
    end
end

local function extras(card, field)
    card.ability[field] = card.ability[field] or {}
    return card.ability[field]
end

local function add_extra_edition(card, key)
    if not G.P_CENTERS[key] then return end
    local list = extras(card, "sj_extra_editions")
    list[#list + 1] = key
    local limit = G.P_CENTERS[key].config and G.P_CENTERS[key].config.card_limit
    if limit then card.ability.card_limit = (card.ability.card_limit or 0) + limit end
end

local function clear_extra_editions(card)
    for _, key in ipairs(card.ability.sj_extra_editions or {}) do
        local limit = G.P_CENTERS[key] and G.P_CENTERS[key].config and G.P_CENTERS[key].config.card_limit
        if limit then card.ability.card_limit = (card.ability.card_limit or 0) - limit end
    end
    card.ability.sj_extra_editions = nil
end

-- Every edition on a card: its real one first, then the stacked extras
local function all_editions(card)
    local list = {}
    if card.edition and card.edition.key then list[1] = card.edition.key end
    for _, key in ipairs(card.ability.sj_extra_editions or {}) do list[#list + 1] = key end
    return list
end

-- Editions always mix when fusing: the first becomes the card's real edition and
-- every other one (duplicates included) stacks on as an extra
local function apply_editions(card, list)
    clear_extra_editions(card)
    card:set_edition(list[1], true, true)
    for i = 2, #list do add_extra_edition(card, list[i]) end
end

local function mixed_editions(a, b)
    local list = all_editions(a)
    for _, key in ipairs(all_editions(b)) do list[#list + 1] = key end
    return list
end

----------------------------------------------------------------------
-- Fused Joker
----------------------------------------------------------------------

local FUSED_JOKER = "j_sj_fused"

-- Effect keys that count as "the effect" of a scoring trigger, normalised
local SCORE_KEYS = {
    chips = "chips", chip_mod = "chips", h_chips = "chips",
    mult = "mult", mult_mod = "mult", h_mult = "mult",
    xmult = "xmult", x_mult = "xmult", Xmult = "xmult", Xmult_mod = "xmult", x_mult_mod = "xmult",
    xchips = "xchips", x_chips = "xchips", Xchip_mod = "xchips",
    dollars = "dollars", p_dollars = "dollars", h_dollars = "dollars",
}
local MULTIPLIERS = { xmult = true, xchips = true }

local function scoring_kind(context)
    if context.joker_main then return "main" end
    if context.individual and context.cardarea == G.play then return "play" end
    if context.individual and context.cardarea == G.hand and not context.end_of_round then return "hand" end
end

-- The good scoring numbers in a result (following its extra chain)
local function scoring_effect(ret)
    local found
    while type(ret) == "table" do
        for k, v in pairs(ret) do
            local norm = SCORE_KEYS[k]
            if norm and type(v) == "number" and (MULTIPLIERS[norm] and v > 1 or not MULTIPLIERS[norm] and v > 0) then
                found = found or {}
                found[norm] = MULTIPLIERS[norm] and (found[norm] or 1) * v or (found[norm] or 0) + v
            end
        end
        ret = ret.extra
    end
    return found
end

-- Remove bad parts from a result: money loss, negative bonuses, multipliers below 1
local function strip_bad(ret)
    local node = ret
    while type(node) == "table" do
        for k, v in pairs(node) do
            local norm = SCORE_KEYS[k]
            if norm and type(v) == "number" and (MULTIPLIERS[norm] and v < 1 or not MULTIPLIERS[norm] and v < 0) then
                node[k] = nil
            end
        end
        node = node.extra
    end
    return ret
end

-- Contexts that only ask a question (probabilities, enhancement checks...) must not be
-- run once per fused copy. Older Steamodded versions don't have this check built in.
local NO_RETRIGGER = {
    mod_probability = true, fix_probability = true, check_enhancement = true,
    retrigger_joker_check = true, retrigger_joker = true, modify_scoring_hand = true,
    modify_weights = true, evaluate_poker_hand = true, debuff_hand = true,
}

local function can_context_retrigger(context)
    if SMODS.can_context_retrigger then return SMODS.can_context_retrigger(context) end
    for key in pairs(NO_RETRIGGER) do
        if context[key] then return false end
    end
    return true
end

local function run_joker_part(card, part, fn)
    return run_in_modes({ swap_mode(card, part), no_loss_mode, SJ.protect_mode }, fn)
end

-- Probing: to share a trigger, a half that didn't trigger is test-run in a sandbox where
-- the card counts as every suit, face and rank and every poker hand is present. Its
-- state, the card's state and the shop/consumable buffers are copied and restored,
-- events are dropped and no money changes hands, so a probe has no side effects.
local probing, probe_card = 0, nil
local PROBE_IDS = { 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2 }

local probe_add_event_ref = EventManager.add_event
function EventManager:add_event(event, ...)
    if probing > 0 then return end
    return probe_add_event_ref(self, event, ...)
end

local probe_ease_dollars_ref = ease_dollars
function ease_dollars(...)
    if probing > 0 then return end
    return probe_ease_dollars_ref(...)
end

local probe_is_suit_ref = Card.is_suit
function Card:is_suit(...)
    if probing > 0 and probe_card == self then return true end
    return probe_is_suit_ref(self, ...)
end

local probe_is_face_ref = Card.is_face
function Card:is_face(...)
    if probing > 0 and probe_card == self then return true end
    return probe_is_face_ref(self, ...)
end

local function every_hand(poker_hands, played)
    local out = {}
    for k, v in pairs(poker_hands or {}) do out[k] = v end
    for _, name in ipairs(G.handlist or {}) do
        if not (out[name] and next(out[name])) then out[name] = { played } end
    end
    return out
end

-- Contexts to probe in: the same kind of trigger first, then the other scoring kinds
local function probe_contexts(context, kind)
    local played = context.scoring_hand or (G.play and G.play.cards) or {}
    local base = {
        full_hand = context.full_hand or played,
        scoring_hand = played,
        scoring_name = context.scoring_name,
        poker_hands = every_hand(context.poker_hands, played),
    }
    local play_card = (kind == "play" and context.other_card) or played[1]
    local hand_card = (kind == "hand" and context.other_card) or (G.hand and G.hand.cards[1])
    local function make(t)
        for k, v in pairs(base) do
            if t[k] == nil then t[k] = v end
        end
        return t
    end
    local build = {
        main = function() return make({ joker_main = true, cardarea = G.jokers }) end,
        play = function() return play_card and make({ individual = true, cardarea = G.play, other_card = play_card }) end,
        hand = function() return hand_card and make({ individual = true, cardarea = G.hand, other_card = hand_card }) end,
    }
    local list = {}
    for _, k in ipairs({ kind, "main", "play", "hand" }) do
        local ctx = build[k] and build[k]()
        if ctx and not list[k] then
            list[k] = true
            list[#list + 1] = ctx
        end
    end
    return list
end

local function probe_effect(card, part, ctx)
    local entry = { key = part.key, ability = copy_table(part.ability) }
    local other = ctx.other_card
    local other_ability = other and other.ability
    if other then other.ability = copy_table(other_ability) end
    local buffers = { G.GAME.consumeable_buffer, G.GAME.joker_buffer }
    probing, probe_card = probing + 1, other
    local found
    local ok, err = pcall(function()
        for _, id in ipairs(other and PROBE_IDS or { false }) do
            if other then other.sj_force_id = id end
            local ret = run_joker_part(card, entry, function() return card:calculate_joker(ctx) end)
            found = scoring_effect(type(ret) == "table" and strip_bad(ret) or nil)
            if found then break end
        end
    end)
    if other then
        other.sj_force_id = nil
        other.ability = other_ability
    end
    probing, probe_card = probing - 1, nil
    G.GAME.consumeable_buffer, G.GAME.joker_buffer = buffers[1], buffers[2]
    if not ok then error(err, 0) end
    return found
end

-- Runs a part's add_to_deck / remove_from_deck passives without touching the fused
-- card's own edition or added_to_deck state
local function part_deck_change(card, part, adding, from_debuff)
    run_joker_part(card, part, function()
        local edition, added = card.edition, card.added_to_deck
        card.edition = nil
        if adding then
            card.added_to_deck = false
            Card.add_to_deck(card, from_debuff)
        else
            card.added_to_deck = true
            Card.remove_from_deck(card, from_debuff)
        end
        card.edition, card.added_to_deck = edition, added
    end)
end

SMODS.Joker {
    key = "fused",
    loc_txt = {
        name = "Fused Joker",
        text = {
            "{C:attention}#1#{}",
            "Has every ability below, and when",
            "one triggers while scoring, the",
            "others trigger with it",
            "{C:inactive}(No downsides, copies retrigger)",
        },
    },
    config = { extra = { parts = {} } },
    rarity = 4,
    cost = 10,
    atlas = "jokers",
    pos = { x = 0, y = 7 },
    blueprint_compat = true,
    no_collection = true,
    discovered = true,
    unlocked = true,
    in_pool = function() return false end,
    loc_vars = function(self, info_queue, card)
        local parts = card.ability.extra.parts or {}
        add_part_tooltips(info_queue, parts)
        return { vars = { #parts > 0 and part_names(parts) or "Nothing" } }
    end,
    add_to_deck = function(self, card, from_debuff)
        for _, part in ipairs(card.ability.extra.parts) do part_deck_change(card, part, true, from_debuff) end
    end,
    remove_from_deck = function(self, card, from_debuff)
        for _, part in ipairs(card.ability.extra.parts) do part_deck_change(card, part, false, from_debuff) end
    end,
    update = function(self, card, dt)
        if not card.added_to_deck then return end
        for _, part in ipairs(card.ability.extra.parts) do
            local center = G.P_CENTERS[part.key]
            if center and center.update then
                run_joker_part(card, part, function() center:update(card, dt) end)
            end
        end
    end,
    calculate = function(self, card, context)
        local parts = card.ability.extra.parts
        local results = {}
        local can_retrigger = can_context_retrigger(context)
        for i, part in ipairs(parts) do
            -- Fused copies of the same Joker retrigger it once per extra copy
            local runs = {}
            for copy = 1, can_retrigger and (part.copies or 1) or 1 do
                local ret = run_joker_part(card, part, function() return card:calculate_joker(context) end)
                if type(ret) ~= "table" then break end
                runs[#runs + 1] = strip_bad(ret)
            end
            if #runs == 1 then
                results[i] = runs[1]
            elseif #runs > 1 then
                local reps = 0
                for _, r in ipairs(runs) do reps = reps + (r.repetitions or 0) end
                results[i] = reps > 0 and { repetitions = reps } or SMODS.merge_effects(runs)
            end
        end

        -- Shared triggers: when any half triggers while scoring, every other half gives
        -- its effect too. The effect is found by probing that half right now; if a probe
        -- can't make it trigger, the last effect it gave on its own is used instead.
        local kind = probing == 0 and scoring_kind(context)
        if kind then
            local triggered = false
            for i, part in ipairs(parts) do
                local effect = scoring_effect(results[i])
                if effect then
                    part.learned = part.learned or {}
                    if not context.blueprint then part.learned[kind] = effect end
                    triggered = true
                end
            end
            if triggered then
                for i, part in ipairs(parts) do
                    if not scoring_effect(results[i]) then
                        local effect
                        for _, probe_ctx in ipairs(probe_contexts(context, kind)) do
                            effect = probe_effect(card, part, probe_ctx)
                            if effect then break end
                        end
                        effect = effect or (part.learned and part.learned[kind] and copy_table(part.learned[kind]))
                        if effect then
                            results[i] = results[i] and SMODS.merge_effects({ results[i], effect }) or effect
                        end
                    end
                end
            end
        end

        local effects, repetitions = {}, 0
        for i = 1, #parts do
            local ret = results[i]
            if ret and next(ret) then
                if ret.repetitions then
                    repetitions = repetitions + ret.repetitions
                else
                    effects[#effects + 1] = ret
                end
            end
        end
        if repetitions > 0 then
            return { message = localize("k_again_ex"), repetitions = repetitions, card = card }
        end
        if #effects > 0 then return SMODS.merge_effects(effects) end
    end,
    calc_dollar_bonus = function(self, card)
        local total = 0
        for _, part in ipairs(card.ability.extra.parts) do
            local bonus = run_joker_part(card, part, function() return card:calculate_dollar_bonus() end)
            if type(bonus) == "number" and bonus > 0 then total = total + bonus * (part.copies or 1) end
        end
        if total > 0 then return total end
    end,
}

-- Same Joker fused more than once becomes one part with a copy count
local function merge_copies(parts)
    local merged, by_key = {}, {}
    for _, p in ipairs(parts) do
        local existing = by_key[p.key]
        if existing then
            existing.copies = (existing.copies or 1) + (p.copies or 1)
        else
            p.copies = p.copies or 1
            by_key[p.key] = p
            merged[#merged + 1] = p
        end
    end
    return merged
end

local function is_invisible(j)
    return j.config.center.key == "j_invisible"
end

local function fuse_jokers(a, b)
    -- Invisible Joker + anything else: Invisible is used up to copy the other Joker
    if is_invisible(a) ~= is_invisible(b) then
        local invisible, other = a, b
        if is_invisible(b) then invisible, other = b, a end
        local editions = mixed_editions(other, invisible)
        SMODS.destroy_cards({ invisible }, { bypass_eternal = true })
        local copy = copy_card(other)
        apply_editions(copy, editions)
        copy:add_to_deck()
        G.jokers:emplace(copy)
        copy:start_materialize()
        return
    end

    local parts = flatten_parts(a, FUSED_JOKER)
    for _, p in ipairs(flatten_parts(b, FUSED_JOKER)) do parts[#parts + 1] = p end
    parts = merge_copies(parts)
    local editions = mixed_editions(a, b)
    local sell = (a.sell_cost or 0) + (b.sell_cost or 0)

    SMODS.destroy_cards({ a, b }, { bypass_eternal = true })

    local fused = create_card("Joker", G.jokers, nil, nil, nil, nil, FUSED_JOKER, "sj_fuse")
    fused.ability.extra.parts = parts
    -- Also replaces create_card's random edition roll
    apply_editions(fused, editions)
    fused:add_to_deck()
    G.jokers:emplace(fused)
    fused:start_materialize()
    fused.ability.extra_value = math.max(sell - fused.sell_cost, 0)
    fused:set_cost()
end

----------------------------------------------------------------------
-- The Fusion consumable type (Fuse and every fused consumable)
----------------------------------------------------------------------

SMODS.Atlas { key = "fusion", path = "fusion.png", px = 71, py = 95 }

SMODS.ConsumableType {
    key = "Fusion",
    primary_colour = HEX("E8743B"),
    secondary_colour = HEX("C85A2A"),
    collection_rows = { 4, 4 },
    -- Same shop weight as Tarots and Planets (4 each; Jokers are 20)
    shop_rate = 4,
    -- If the pool is ever empty (e.g. a Fuse is already in the shop), still offer a Fuse
    default = "c_sj_fuse",
    loc_txt = {
        name = "Fusion",
        collection = "Fusion Cards",
        undiscovered = {
            name = "Not Discovered",
            text = {
                "Purchase or use",
                "this card in an",
                "unseeded run to",
                "learn what it does",
            },
        },
    },
}

----------------------------------------------------------------------
-- Fused consumables (same set only). They belong to the Fusion category.
----------------------------------------------------------------------

-- Maps a vanilla set to the fused card made from it. Fused cards are Fusion cards,
-- but remember which set their parts came from so they only fuse with that set.
local FUSED_CONSUMABLE = {}

local function base_set(c)
    local extra = c.ability.extra
    return type(extra) == "table" and extra.part_set or c.ability.set
end

-- Playing cards touched by the fused consumable currently being used. The first
-- part to set an enhancement / seal / edition on a card does it normally, later
-- parts add theirs on top instead of replacing it.
local current_touch = nil
local hand_unhighlight_pending = false

local function touch_mode(touch)
    return {
        enter = function()
            local previous = current_touch
            current_touch = touch
            return function() current_touch = previous end
        end,
    }
end


local function edition_key(edition)
    if type(edition) == "string" then return edition end
    if type(edition) == "table" then
        if edition.key then return edition.key end
        if edition.type then return "e_" .. edition.type end
        for k, v in pairs(edition) do
            if v then return "e_" .. k end
        end
    end
end


local function add_extra_seal(card, seal)
    if not seal or card.seal == seal then return end
    local list = extras(card, "sj_extra_seals")
    for _, s in ipairs(list) do
        if s == seal then return end
    end
    list[#list + 1] = seal
end

local function add_extra_enhancement(card, key)
    if not key or key == "c_base" or card.config.center.key == key then return end
    extras(card, "sj_extra_enh")[key] = true
    SMODS.enh_cache:write(card, nil)
end

local set_ability_ref = Card.set_ability
function Card:set_ability(center, initial, delay_sprites)
    if current_touch and self.playing_card and type(center) == "table" and center.set == "Enhanced"
            and delay_sprites ~= "quantum" then
        local t = current_touch[self] or {}
        current_touch[self] = t
        if t.enhancement and self.config.center.set == "Enhanced" then
            add_extra_enhancement(self, center.key)
            return
        end
        t.enhancement = true
    end
    return set_ability_ref(self, center, initial, delay_sprites)
end

local set_seal_ref = Card.set_seal
function Card:set_seal(seal, silent, immediate)
    if current_touch and self.playing_card and seal then
        local t = current_touch[self] or {}
        current_touch[self] = t
        if t.seal and self.seal then
            add_extra_seal(self, seal)
            return
        end
        t.seal = true
    end
    return set_seal_ref(self, seal, silent, immediate)
end

local set_edition_ref = Card.set_edition
function Card:set_edition(edition, immediate, silent, delay)
    local key = edition_key(edition)
    if current_touch and self.playing_card and key and key ~= "e_base" then
        local t = current_touch[self] or {}
        current_touch[self] = t
        if t.edition and self.edition then
            add_extra_edition(self, key)
            return
        end
        t.edition = true
    end
    return set_edition_ref(self, edition, immediate, silent, delay)
end

-- Vanilla Tarots unhighlight the hand inside their own events, which would break the
-- second part's events. Hold it until the fused card is finished.
local unhighlight_all_ref = CardArea.unhighlight_all
function CardArea:unhighlight_all(...)
    if current_touch and self == G.hand then
        hand_unhighlight_pending = true
        return
    end
    return unhighlight_all_ref(self, ...)
end

----------------------------------------------------------------------
-- Special fusions: specific pairs of Tarot or Spectral cards do something
-- new instead of just both effects. The pairs live in fusion_specials.lua and are
-- written with the building blocks below (F.*). Each block queues its work as an
-- event, so a special plays out step by step in the order it is written.
----------------------------------------------------------------------

local F = {}
SimplyJokers.fusion = F

local SPECIALS = {}
F.loc = {}

local function pair_key(a, b)
    if a > b then a, b = b, a end
    return a .. "+" .. b
end

function F.register(a, b, special)
    SPECIALS[pair_key(a, b)] = special
    if special.lines then
        local loc_key = "sj_special_" .. a .. "_" .. b
        special.loc_key = loc_key
        F.loc[loc_key] = { name = special.title or "Special Fusion", text = special.lines }
    end
end

-- Blocks used when a duplicate runs its effect a second time (downsides only once)
local no_hand_loss, no_card_destroy = 0, 0
local function counter_mode(get, set)
    return {
        enter = function()
            set(get() + 1)
            return function() set(get() - 1) end
        end,
    }
end
local BLOCK_MODES = {
    money = no_loss_mode,
    jokers = SJ.protect_mode,
    hand_size = counter_mode(function() return no_hand_loss end, function(v) no_hand_loss = v end),
    cards = counter_mode(function() return no_card_destroy end, function(v) no_card_destroy = v end),
}

local change_size_ref = CardArea.change_size
function CardArea:change_size(delta, ...)
    if no_hand_loss > 0 and self == G.hand and type(delta) == "number" and delta < 0 then return end
    return change_size_ref(self, delta, ...)
end

for _, name in ipairs({ "start_dissolve", "shatter" }) do
    local ref = Card[name]
    Card[name] = function(self, ...)
        if no_card_destroy > 0 and self.playing_card then
            self.getting_sliced = nil
            return
        end
        return ref(self, ...)
    end
end

local destroy_cards_ref = SMODS.destroy_cards
function SMODS.destroy_cards(cards, ...)
    if no_card_destroy > 0 then
        local list = cards and cards[1] and cards or { cards }
        local kept = {}
        for _, c in ipairs(list) do
            if not c.playing_card then kept[#kept + 1] = c end
        end
        if #kept == 0 then return end
        cards = kept
    end
    return destroy_cards_ref(cards, ...)
end

local function step(fn, delay)
    G.E_MANAGER:add_event(Event({
        trigger = "after",
        delay = delay or 0.15,
        func = function()
            fn()
            return true
        end,
    }))
end
F.step = step

local function resolve(list)
    if type(list) == "function" then list = list() end
    local out = {}
    for _, c in ipairs(list or {}) do
        if c and not c.removed and not c.getting_sliced then out[#out + 1] = c end
    end
    return out
end

local function shuffle_pick(list, n, seed)
    local pool, out = {}, {}
    for _, c in ipairs(list) do pool[#pool + 1] = c end
    for _ = 1, math.min(n, #pool) do
        local c, i = pseudorandom_element(pool, pseudoseed(seed))
        out[#out + 1] = c
        table.remove(pool, i)
    end
    return out
end

local SEALS = { "Gold", "Red", "Blue", "Purple" }
local SUITS = { "Spades", "Hearts", "Clubs", "Diamonds" }
local FACE = { "Jack", "Queen", "King" }
local NUMBERED = { "2", "3", "4", "5", "6", "7", "8", "9", "10" }
local ANY_RANK = { "2", "3", "4", "5", "6", "7", "8", "9", "10", "Jack", "Queen", "King", "Ace" }

local function random_enhancement(seed)
    local pool = {}
    for _, v in pairs(G.P_CENTER_POOLS.Enhanced) do
        if v.key ~= "m_stone" and not v.overrides_base_rank then pool[#pool + 1] = v end
    end
    return pseudorandom_element(pool, pseudoseed(seed)).key
end

local function random_edition_key(seed, no_negative)
    return edition_key(poll_edition(seed, nil, no_negative ~= false, true)) or "e_foil"
end

-- Stacking setters: a card that already has one keeps it and gains the new one as an extra
local function give_enhancement(c, key, stack)
    if stack and c.config.center.set == "Enhanced" and c.config.center.key ~= key then
        add_extra_enhancement(c, key)
    else
        set_ability_ref(c, G.P_CENTERS[key])
    end
end

local function give_seal(c, seal)
    if c.seal and c.seal ~= seal then add_extra_seal(c, seal) else set_seal_ref(c, seal, true, true) end
end

local function give_edition(c, key)
    if c.edition then add_extra_edition(c, key) else set_edition_ref(c, key, true) end
end

F.give_enhancement, F.give_seal, F.give_edition = give_enhancement, give_seal, give_edition

function F.sel(ctx) return ctx.selected end

-- Flip the cards, change them, flip them back
function F.apply(cards, opts)
    local list
    step(function()
        list = resolve(cards)
        for _, c in ipairs(list) do
            c:flip()
            play_sound("card1")
            c:juice_up(0.3, 0.3)
        end
    end)
    step(function()
        for _, c in ipairs(list) do
            if opts.suit or opts.rank then SMODS.change_base(c, opts.suit, opts.rank) end
            if opts.rank_up then SMODS.modify_rank(c, opts.rank_up) end
            if opts.enhancement then give_enhancement(c, opts.enhancement, opts.stack) end
            if opts.also then add_extra_enhancement(c, opts.also) end
            if opts.seal then give_seal(c, opts.seal) end
            if opts.edition then give_edition(c, opts.edition == "random" and random_edition_key("sj_fs_ed") or opts.edition) end
            if opts.fn then opts.fn(c) end
            SMODS.enh_cache:write(c, nil)
        end
    end, 0.3)
    step(function()
        for _, c in ipairs(list) do
            c:flip()
            play_sound("tarot2", 1, 0.6)
        end
    end, 0.3)
end

function F.both(cards, a, b)
    F.apply(cards, {
        fn = function(c)
            set_ability_ref(c, G.P_CENTERS[a])
            add_extra_enhancement(c, b)
        end,
    })
end

function F.seal(cards, seal) F.apply(cards, { seal = seal }) end
function F.edition(cards, key) F.apply(cards, { edition = key }) end

-- Each card has a 1 in `odds` chance to get a random edition
function F.random_editions(ctx, cards, odds)
    F.apply(cards, {
        fn = function(c)
            if SMODS.pseudorandom_probability(ctx.card, "sj_fs_reds", 1, odds) then
                give_edition(c, random_edition_key("sj_fs_red"))
            end
        end,
    })
end

local function pick_suit(ctx, suit)
    if suit == "random" then
        ctx.random_suit = ctx.random_suit or pseudorandom_element(SUITS, pseudoseed("sj_fs_suit"))
        return ctx.random_suit
    elseif suit == "selected" then
        return ctx.selected[1] and ctx.selected[1].base.suit
    elseif suit == "destroyed" then
        return ctx.destroyed[1] and ctx.destroyed[1].base.suit or pick_suit(ctx, "random")
    end
    return suit
end

local function pick_rank(ctx, rank, pool)
    if rank == "random" then
        ctx.random_rank = ctx.random_rank or pseudorandom_element(pool or ANY_RANK, pseudoseed("sj_fs_rank"))
        return ctx.random_rank
    elseif rank == "selected" then
        return ctx.selected[1] and ctx.selected[1].base.value
    elseif rank == "made" then
        return ctx.made[1] and ctx.made[1].base.value or pick_rank(ctx, "random", pool)
    end
    return rank
end

function F.convert_hand(ctx, suit, rank)
    local opts = {}
    step(function()
        opts.suit = suit and pick_suit(ctx, suit) or nil
        opts.rank = rank and pick_rank(ctx, rank) or nil
    end)
    F.apply(function() return G.hand.cards end, opts)
end

function F.destroy_random(ctx, n, keep_selected)
    step(function()
        local pool = {}
        for _, c in ipairs(G.hand.cards) do
            local selected = false
            for _, s in ipairs(ctx.selected) do
                if s == c then selected = true end
            end
            if not (keep_selected and selected) and not c.getting_sliced then pool[#pool + 1] = c end
        end
        local list = shuffle_pick(pool, n, "sj_fs_destroy")
        for _, c in ipairs(list) do ctx.destroyed[#ctx.destroyed + 1] = c end
        if #list > 0 then
            play_sound("tarot1")
            SMODS.destroy_cards(list)
        end
    end, 0.4)
end

function F.destroy_selected(ctx, cards)
    step(function()
        local list = resolve(cards or ctx.selected)
        for _, c in ipairs(list) do ctx.destroyed[#ctx.destroyed + 1] = c end
        if #list > 0 then SMODS.destroy_cards(list) end
    end, 0.3)
end

-- Add new playing cards. opts: kind (face/ace/numbered/any), suit, rank, enhanced,
-- enhancement, seal ("random" ok), edition ("random" ok), negative_first, like (a card
-- to copy the enhancement/seal/edition of), area ("deck" or the hand)
function F.add_cards(ctx, n, opts)
    opts = opts or {}
    step(function()
        local made = {}
        local pool = opts.kind == "face" and FACE or opts.kind == "ace" and { "Ace" }
            or opts.kind == "numbered" and NUMBERED or ANY_RANK
        for i = 1, n do
            local enhancement = opts.enhancement
            if opts.like and opts.like.config.center.set == "Enhanced" then enhancement = opts.like.config.center.key end
            if not enhancement and opts.enhanced then enhancement = random_enhancement("sj_fs_enh") end
            local rank = opts.rank and pick_rank(ctx, opts.rank, pool) or pseudorandom_element(pool, pseudoseed("sj_fs_r"))
            local suit = opts.suit and pick_suit(ctx, opts.suit) or pseudorandom_element(SUITS, pseudoseed("sj_fs_s"))
            local c = SMODS.add_card({
                set = enhancement and "Enhanced" or "Base",
                enhancement = enhancement,
                rank = rank,
                suit = suit,
                area = opts.area == "deck" and G.deck or G.hand,
                key_append = "sj_fs",
                no_edition = true,
            })
            local seal = opts.seal == "random" and pseudorandom_element(SEALS, pseudoseed("sj_fs_seal")) or opts.seal
            if opts.like and opts.like.seal then seal = opts.like.seal end
            if seal then c:set_seal(seal, true, true) end
            local edition = opts.edition == "random" and random_edition_key("sj_fs_ed") or opts.edition
            if opts.negative_first and i == 1 then edition = "e_negative" end
            if opts.like and opts.like.edition then edition = opts.like.edition.key end
            if edition then c:set_edition(edition, true) end
            made[#made + 1] = c
            ctx.made[#ctx.made + 1] = c
        end
        if playing_card_joker_effects then playing_card_joker_effects(made) end
        play_sound("card1")
    end, 0.5)
end

-- Copies of a selected card (index picks which selected card, default the first)
function F.copy_selected(ctx, n, opts)
    opts = opts or {}
    step(function()
        local base = ctx.selected[opts.index or 1]
        if not base or base.removed then return end
        local made = {}
        for _ = 1, n do
            G.playing_card = (G.playing_card and G.playing_card + 1) or 1
            local c = copy_card(base, nil, nil, G.playing_card)
            c:add_to_deck()
            G.deck.config.card_limit = G.deck.config.card_limit + 1
            table.insert(G.playing_cards, c)
            G.hand:emplace(c)
            c:start_materialize()
            if opts.seal then give_seal(c, opts.seal) end
            if opts.edition then give_edition(c, opts.edition == "random" and random_edition_key("sj_fs_ed") or opts.edition) end
            made[#made + 1] = c
            ctx.made[#ctx.made + 1] = c
        end
        if playing_card_joker_effects then playing_card_joker_effects(made) end
    end, 0.4)
end

-- Death: the left selected card becomes a copy of the right one
function F.death(ctx)
    F.apply(function() return { ctx.selected[1] } end, {
        fn = function(c)
            local right = ctx.selected[2]
            if right and right ~= c then copy_card(right, c) end
        end,
    })
end

function F.hand_size(n) step(function() G.hand:change_size(n) end) end

function F.ectoplasm_hand(ctx)
    step(function()
        G.GAME.ecto_minus = G.GAME.ecto_minus or 1
        G.hand:change_size(-G.GAME.ecto_minus)
        G.GAME.ecto_minus = G.GAME.ecto_minus + 1
    end)
end

function F.money(n) step(function() ease_dollars(type(n) == "function" and n() or n) end, 0.3) end
function F.set_money(v) step(function() if G.GAME.dollars ~= v then ease_dollars(v - G.GAME.dollars, true) end end, 0.3) end
function F.half_money() step(function() ease_dollars(-math.floor(math.max(G.GAME.dollars, 0) / 2), true) end, 0.3) end

function F.double_money(max, odds, ctx)
    step(function()
        if odds and not SMODS.pseudorandom_probability(ctx.card, "sj_fs_hermit", 1, odds) then return end
        ease_dollars(math.max(0, math.min(G.GAME.dollars, max)))
    end, 0.3)
end

function F.sell_value(max, extra)
    step(function()
        local total = 0
        for _, j in ipairs(G.jokers.cards) do total = total + (j.sell_cost or 0) end
        ease_dollars(math.min(total, max) + (type(extra) == "function" and extra() or extra or 0))
    end, 0.3)
end

function F.count_suit(suit)
    local n = 0
    for _, c in ipairs(G.playing_cards) do
        if c:is_suit(suit) then n = n + 1 end
    end
    return n
end

local function joker_room(n)
    return #G.jokers.cards + (G.GAME.joker_buffer or 0) + (n or 1) <= G.jokers.config.card_limit
end

-- Create a Joker. opts: rarity, legendary, key, edition ("random" ok), edition_odds,
-- count (or a function returning one)
function F.joker(ctx, opts)
    opts = opts or {}
    step(function()
        local count = type(opts.count) == "function" and opts.count() or opts.count or 1
        for _ = 1, count do
            local edition = opts.edition == "random" and random_edition_key("sj_fs_jed", false) or opts.edition
            if opts.edition_odds and SMODS.pseudorandom_probability(ctx.card, "sj_fs_jodds", 1, opts.edition_odds) then
                edition = random_edition_key("sj_fs_jed")
            end
            if edition ~= "e_negative" and not joker_room() then break end
            local j = SMODS.add_card({
                set = "Joker",
                rarity = opts.rarity,
                legendary = opts.legendary,
                key = opts.key,
                key_append = "sj_fs",
                no_edition = edition == nil,
            })
            if edition then j:set_edition(edition, true) end
            ctx.made_jokers[#ctx.made_jokers + 1] = j
        end
        play_sound("timpani")
    end, 0.4)
end

-- A copy of the last Joker this special created
function F.copy_made_joker(ctx)
    step(function()
        local src = ctx.made_jokers[#ctx.made_jokers]
        if not src or not joker_room() then return end
        local j = copy_card(src)
        j:add_to_deck()
        G.jokers:emplace(j)
        j:start_materialize()
        ctx.made_jokers[#ctx.made_jokers + 1] = j
    end, 0.4)
end

local function other_jokers(ctx)
    local list = {}
    for _, j in ipairs(G.jokers.cards) do
        local mine = j == ctx.target or j == ctx.chosen
        for _, m in ipairs(ctx.made_jokers) do
            if m == j then mine = true end
        end
        if not mine and not j.getting_sliced and not SMODS.is_eternal(j) then list[#list + 1] = j end
    end
    return list
end

-- Destroy the other Jokers: all of them, or `count` random ones
function F.destroy_jokers(ctx, count)
    step(function()
        local list = other_jokers(ctx)
        if count then list = shuffle_pick(list, count, "sj_fs_jd") end
        if #list > 0 then SMODS.destroy_cards(list) end
    end, 0.3)
end

-- Ankh-style: copy a random Joker. opts: destroy ("all", a number, or 0), edition,
-- keep_negative
function F.ankh(ctx, opts)
    opts = opts or {}
    step(function()
        local pool = {}
        for _, j in ipairs(G.jokers.cards) do
            if not j.getting_sliced then pool[#pool + 1] = j end
        end
        ctx.chosen = pseudorandom_element(pool, pseudoseed("sj_fs_ankh"))
    end)
    if opts.destroy and opts.destroy ~= 0 then F.destroy_jokers(ctx, opts.destroy ~= "all" and opts.destroy or nil) end
    step(function()
        local src = ctx.chosen
        if not src then return end
        local edition = opts.edition == "random" and random_edition_key("sj_fs_jed") or opts.edition
        if edition ~= "e_negative" and not joker_room() then return end
        local j = copy_card(src, nil, nil, nil, not opts.keep_negative and src.edition and src.edition.negative)
        if edition then give_edition(j, edition) end
        j:add_to_deck()
        G.jokers:emplace(j)
        j:start_materialize()
        ctx.made_jokers[#ctx.made_jokers + 1] = j
    end, 0.4)
end

-- Give a random Joker an edition. same = reuse the Joker picked by the last call
function F.joker_edition(ctx, key, same)
    step(function()
        if not (same and ctx.target) then
            local pool = {}
            for _, j in ipairs(G.jokers.cards) do
                if not j.edition and not j.getting_sliced then pool[#pool + 1] = j end
            end
            ctx.target = pseudorandom_element(pool, pseudoseed("sj_fs_jedit"))
        end
        if ctx.target then
            give_edition(ctx.target, key == "random" and random_edition_key("sj_fs_jed") or key)
            ctx.target:juice_up(0.3, 0.5)
        end
    end, 0.4)
end

-- Wheel of Fortune: 1 in `odds` chance for a random Joker to get a random edition
function F.wheel(ctx, odds, times)
    step(function()
        local n = type(times) == "function" and times() or times or 1
        for _ = 1, n do
            local pool = {}
            for _, j in ipairs(G.jokers.cards) do
                if not j.edition then pool[#pool + 1] = j end
            end
            if #pool > 0 and SMODS.pseudorandom_probability(ctx.card, "wheel_of_fortune", 1, odds) then
                local j = pseudorandom_element(pool, pseudoseed("sj_fs_wheel"))
                j:set_edition(random_edition_key("wheel_of_fortune"), true)
            end
        end
    end, 0.4)
end

local HAND_GROUPS = {
    flush = { "Flush", "Straight Flush", "Flush House", "Flush Five" },
    kind = { "Pair", "Two Pair", "Three of a Kind", "Four of a Kind", "Five of a Kind", "Full House", "Flush House", "Flush Five" },
}

function F.most_played()
    local best, played = "High Card", -1
    for _, name in ipairs(G.handlist) do
        local h = G.GAME.hands[name]
        if h and h.played > played and SMODS.is_poker_hand_visible(name) then best, played = name, h.played end
    end
    return best
end

function F.planet_for(hand)
    for _, v in pairs(G.P_CENTER_POOLS.Planet) do
        if v.config and v.config.hand_type == hand then return v.key end
    end
end

-- Level up hands: "all", a group name ("flush", "kind"), "most_played" or a list
function F.level(ctx, which, n)
    step(function()
        local names = {}
        if which == "all" then
            for _, name in ipairs(G.handlist) do names[#names + 1] = name end
        elseif which == "most_played" then
            names = { F.most_played() }
        elseif HAND_GROUPS[which] then
            names = HAND_GROUPS[which]
        else
            names = which
        end
        for _, name in ipairs(names) do
            if G.GAME.hands[name] then level_up_hand(ctx.card, name, true, n) end
        end
        play_sound("tarot1")
        update_hand_text({ sound = "button", volume = 0.7, pitch = 0.9, delay = 0 },
            { handname = which == "all" and localize("k_all_hands") or names[1], chips = "...", mult = "...", level = "+" .. n })
        delay(0.6)
        update_hand_text({ sound = "button", volume = 0.7, pitch = 1.1, delay = 0 },
            { mult = 0, chips = 0, handname = "", level = "" })
    end, 0.4)
end

local function consumable_room()
    return #G.consumeables.cards + G.GAME.consumeable_buffer < G.consumeables.config.card_limit
end

-- Create consumables. what: { key = ... }, { set = ... }, "last_used" or "most_played_planet".
-- opts: count (number or function), edition, negative_odds
function F.consumable(ctx, what, opts)
    opts = opts or {}
    step(function()
        local key, set
        if what == "last_used" then
            key = G.GAME.last_tarot_planet
            if not key or key == "c_fool" then return end
        elseif what == "most_played_planet" then
            key = F.planet_for(F.most_played())
        else
            key, set = what.key, what.set
        end
        local count = type(opts.count) == "function" and opts.count() or opts.count or 1
        for _ = 1, count do
            local edition = opts.edition
            if opts.negative_odds and SMODS.pseudorandom_probability(ctx.card, "sj_fs_neg", 1, opts.negative_odds) then
                edition = "e_negative"
            end
            if edition ~= "e_negative" and not consumable_room() then break end
            local c = SMODS.add_card({ key = key, set = key and nil or set, area = G.consumeables, key_append = "sj_fs", no_edition = edition == nil })
            if edition then c:set_edition(edition, true) end
        end
    end, 0.4)
end

-- Wraith + The Soul: pick any Joker from the Joker Collection
local picking = nil

function F.choose_joker(ctx)
    step(function()
        picking = true
        G.FUNCS.your_collection_jokers()
    end, 0.3)
end

local card_click_ref = Card.click
function Card:click(...)
    if picking and self.area and self.area.config and self.area.config.collection
            and self.config.center and self.config.center.set == "Joker" then
        local key = self.config.center.key
        picking = nil
        G.FUNCS.exit_overlay_menu()
        G.E_MANAGER:add_event(Event({
            func = function()
                SMODS.add_card({ set = "Joker", key = key, key_append = "sj_pick" })
                play_sound("timpani")
                return true
            end,
        }))
        return
    end
    return card_click_ref(self, ...)
end

local exit_overlay_ref = G.FUNCS.exit_overlay_menu
G.FUNCS.exit_overlay_menu = function(...)
    local was_picking = picking
    picking = nil
    local ret = exit_overlay_ref(...)
    -- Closed without choosing: get a random Rare Joker instead
    if was_picking then
        G.E_MANAGER:add_event(Event({
            func = function()
                SMODS.add_card({ set = "Joker", rarity = "Rare", key_append = "sj_pick" })
                return true
            end,
        }))
    end
    return ret
end

-- Requirements a special checks before it can be used
local function needs_ok(needs)
    if not needs then return true end
    if needs.hand and not (G.hand and #G.hand.cards > 0) then return false end
    if needs.select then
        local n = G.hand and #G.hand.highlighted or 0
        if n < needs.select[1] or n > needs.select[2] then return false end
    end
    if needs.jokers and not joker_room(needs.jokers) then return false end
    if needs.owned_jokers and #G.jokers.cards < needs.owned_jokers then return false end
    if needs.edition_target then
        local found = false
        for _, j in ipairs(G.jokers.cards) do
            if not j.edition then found = true end
        end
        if not found then return false end
    end
    return true
end

-- Pairs up a fused card's parts: special pairs and duplicates first, leftovers run normally
local function plan(parts)
    local used, out = {}, {}
    for i, a in ipairs(parts) do
        if not used[i] then
            for j = i + 1, #parts do
                local b = parts[j]
                if not used[j] then
                    local special = SPECIALS[pair_key(a.key, b.key)]
                    if a.key == b.key then
                        out[#out + 1] = { dup = { a, b } }
                    elseif special then
                        out[#out + 1] = { special = special, parts = { a, b } }
                    end
                    if a.key == b.key or special then
                        used[i], used[j] = true, true
                        break
                    end
                end
            end
            if not used[i] then
                used[i] = true
                out[#out + 1] = { normal = a }
            end
        end
    end
    return out
end
F.plan = plan

local function part_can_use(card, part)
    return run_in_modes({ swap_mode(card, part) }, function() return card:can_use_consumeable() end)
end

function F.can_use(card)
    local parts = card.ability.extra.parts
    if #parts == 0 then return false end
    for _, item in ipairs(plan(parts)) do
        if item.special then
            if not needs_ok(item.special.needs) then return false end
        else
            if not part_can_use(card, item.normal or item.dup[1]) then return false end
        end
    end
    return true
end

local function run_normal(card, part, area, copier, touch, blocks)
    local modes = { swap_mode(card, part), touch_mode(touch) }
    for _, b in ipairs(blocks or {}) do modes[#modes + 1] = BLOCK_MODES[b] end
    run_in_modes(modes, function() card:use_consumeable(area, copier) end)
end

function F.use(card, area, copier)
    local touch = {}
    hand_unhighlight_pending = false
    local selected = {}
    for _, c in ipairs(G.hand and G.hand.highlighted or {}) do selected[#selected + 1] = c end
    table.sort(selected, function(a, b) return a.T.x < b.T.x end)

    for _, item in ipairs(plan(card.ability.extra.parts)) do
        if item.special then
            local ctx = { card = card, selected = selected, made = {}, made_jokers = {}, destroyed = {} }
            step(function()
                play_sound("tarot1")
                card:juice_up(0.3, 0.5)
            end, 0.3)
            item.special.run(ctx, F)
        elseif item.dup then
            -- Same card twice: its effect happens twice, its downside only once
            run_normal(card, item.dup[1], area, copier, touch)
            run_normal(card, item.dup[2], area, copier, touch, { "money", "jokers", "hand_size", "cards" })
        else
            run_normal(card, item.normal, area, copier, touch)
        end
    end
    G.E_MANAGER:add_event(Event({
        func = function()
            hand_unhighlight_pending = false
            if G.hand then G.hand:unhighlight_all() end
            return true
        end,
    }))
end

-- The special that applies to a fused card, for its tooltip
function F.specials_for(parts)
    local list = {}
    for _, item in ipairs(plan(parts)) do
        if item.special and item.special.loc_key then list[#list + 1] = item.special.loc_key end
    end
    return list
end

local function fused_consumable(part_set, key, label, pos)
    FUSED_CONSUMABLE[part_set] = "c_sj_" .. key
    SMODS.Consumable {
        key = key,
        set = "Fusion",
        loc_txt = {
            name = label,
            text = {
                "Does all of:",
                "{C:attention}#1#{}",
            },
        },
        config = { extra = { parts = {}, part_set = part_set } },
        cost = 6,
        atlas = "fusion",
        pos = pos,
        discovered = true,
        unlocked = true,
        in_pool = function() return false end,
        loc_vars = function(self, info_queue, card)
            local parts = card.ability.extra and card.ability.extra.parts or {}
            add_part_tooltips(info_queue, parts)
            for _, loc_key in ipairs(F.specials_for(parts)) do
                info_queue[#info_queue + 1] = { set = "Other", key = loc_key }
            end
            -- Shown in the Collection before anything has been fused into it
            return { vars = { #parts > 0 and part_names(parts) or ("2 fused " .. part_set .. " cards") } }
        end,
        can_use = function(self, card)
            return F.can_use(card)
        end,
        use = function(self, card, area, copier)
            F.use(card, area, copier)
        end,
    }
end

fused_consumable("Tarot", "fused_tarot", "Fused Tarot", { x = 1, y = 0 })
fused_consumable("Planet", "fused_planet", "Fused Planet", { x = 2, y = 0 })
fused_consumable("Spectral", "fused_spectral", "Fused Spectral", { x = 3, y = 0 })

local function fuse_consumables(a, b)
    local set = base_set(a)
    local fused_key = FUSED_CONSUMABLE[set]
    local parts = flatten_parts(a, fused_key)
    for _, p in ipairs(flatten_parts(b, fused_key)) do parts[#parts + 1] = p end
    local editions = mixed_editions(a, b)

    SMODS.destroy_cards({ a, b }, { bypass_eternal = true })

    local fused = create_card("Fusion", G.consumeables, nil, nil, nil, nil, fused_key, "sj_fuse")
    fused.ability.extra = { parts = parts, part_set = set }
    apply_editions(fused, editions)
    fused:add_to_deck()
    G.consumeables:emplace(fused)
    fused:start_materialize()
end

----------------------------------------------------------------------
-- Extra enhancements, seals and editions on playing cards
----------------------------------------------------------------------

local get_enhancements_ref = SMODS.get_enhancements
function SMODS.get_enhancements(card, extra_only)
    local ret = get_enhancements_ref(card, extra_only)
    local extra = card and card.ability and card.ability.sj_extra_enh
    if extra and SMODS.optional_features.quantum_enhancements then
        for key in pairs(extra) do
            if G.P_CENTERS[key] and key ~= card.config.center.key then ret[key] = true end
        end
    end
    return ret
end

-- Combine an extra result into the main one; repetitions add together
local function combine(main, extra)
    if not extra then return main end
    if not main then return extra end
    if extra.repetitions then
        main.repetitions = (main.repetitions or 0) + extra.repetitions
        extra.repetitions = nil
        main.message = main.message or localize("k_again_ex")
    end
    if next(extra) then
        local node = main
        while node.extra do node = node.extra end
        node.extra = extra
    end
    return main
end

local function edition_table(key)
    local center = G.P_CENTERS[key]
    local kind = key:sub(3)
    local t = { [kind] = true, type = kind, key = key }
    for k, v in pairs(center and center.config or {}) do
        t[k] = type(v) == "table" and copy_table(v) or v
    end
    return t
end

local calculate_seal_ref = Card.calculate_seal
function Card:calculate_seal(context, ...)
    local ret, post = calculate_seal_ref(self, context, ...)
    local list = self.ability and self.ability.sj_extra_seals
    if list and #list > 0 then
        local own = self.seal
        for _, seal in ipairs(list) do
            self.seal = seal
            local ok, extra = pcall(calculate_seal_ref, self, context, ...)
            self.seal = own
            if not ok then error(extra, 0) end
            ret = combine(ret, extra)
        end
    end
    return ret, post
end

local calculate_edition_ref = Card.calculate_edition
function Card:calculate_edition(context, ...)
    local ret = calculate_edition_ref(self, context, ...)
    local list = self.ability and self.ability.sj_extra_editions
    if list and #list > 0 and self.edition then
        local own = self.edition
        for _, key in ipairs(list) do
            if G.P_CENTERS[key] then
                self.edition = edition_table(key)
                local ok, extra = pcall(calculate_edition_ref, self, context, ...)
                self.edition = own
                if not ok then error(extra, 0) end
                ret = combine(ret, extra)
            end
        end
    end
    return ret
end

----------------------------------------------------------------------
-- Enhancement combos: a playing card with both enhancements of a pair gets an
-- extra effect on top of both normal ones. "scored" runs when the card scores,
-- "held" while it's held in hand during scoring.
----------------------------------------------------------------------

local function chance(c, seed, odds)
    return SMODS.pseudorandom_probability(c, seed, 1, odds)
end

local function suits_in(cards)
    local seen, n = {}, 0
    for _, c in ipairs(cards or {}) do
        if not SMODS.has_no_suit(c) and not seen[c.base.suit] then
            seen[c.base.suit] = true
            n = n + 1
        end
    end
    return n
end

local function later(fn)
    G.E_MANAGER:add_event(Event({
        func = function()
            fn()
            return true
        end,
    }))
end

local COMBOS = {
    { "m_bonus", "m_mult", held = function() return { chips = 30, mult = 4 } end },
    { "m_bonus", "m_wild", scored = function(c, ctx) return { chips = 15 * suits_in(ctx.full_hand) } end },
    { "m_bonus", "m_glass", scored = function(c)
        c.ability.perma_bonus = (c.ability.perma_bonus or 0) + 10
        return { message = localize("k_upgrade_ex"), colour = G.C.CHIPS }
    end },
    { "m_bonus", "m_steel", held = function() return { chips = 30 } end },
    { "m_bonus", "m_stone", keeps_rank = true, keeps_suit = true },
    { "m_bonus", "m_gold", scored = function() return { dollars = 1 } end },
    { "m_bonus", "m_lucky", scored = function(c) if chance(c, "sj_combo_bl", 5) then return { chips = 100 } end end },
    { "m_mult", "m_wild", scored = function(c, ctx) return { mult = 2 * suits_in(ctx.full_hand) } end },
    { "m_mult", "m_glass", shatter = function(c)
        later(function()
            local pool = {}
            for _, h in ipairs(G.hand.cards) do
                if h ~= c and not h.getting_sliced then pool[#pool + 1] = h end
            end
            local target = pseudorandom_element(pool, pseudoseed("sj_combo_mg"))
            if target then
                set_ability_ref(target, G.P_CENTERS.m_mult)
                target:juice_up(0.3, 0.5)
            end
        end)
    end },
    { "m_mult", "m_steel", held = function() return { mult = 4 } end },
    { "m_mult", "m_stone", keeps_rank = true, keeps_suit = true },
    { "m_mult", "m_gold", scored = function() return { dollars = 1 } end },
    { "m_mult", "m_lucky", scored = function(c) if chance(c, "sj_combo_ml", 5) then return { mult = 20 } end end },
    { "m_wild", "m_glass", glass_safe = function() return (G.GAME.last_hand_played or ""):find("Flush") ~= nil end },
    { "m_wild", "m_steel", scored = function() return { xmult = 1.5 } end },
    { "m_wild", "m_stone", keeps_rank = true },
    { "m_wild", "m_gold", scored = function() return { dollars = 3 } end },
    { "m_wild", "m_lucky", scored = function(c) if chance(c, "sj_combo_wl", 15) then return { dollars = 20 } end end },
    { "m_glass", "m_steel", glass_safe = function() return true end },
    { "m_glass", "m_stone", shatter = function()
        later(function()
            SMODS.add_card({ set = "Enhanced", enhancement = "m_stone", area = G.deck, key_append = "sj_combo", no_edition = true })
        end)
    end },
    { "m_glass", "m_gold", shatter = function() later(function() ease_dollars(10) end) end },
    { "m_glass", "m_lucky", glass_safe = function(c) return chance(c, "sj_combo_gl", 2) end },
    { "m_steel", "m_stone", scored = function() return { xmult = 1.5 } end },
    { "m_steel", "m_gold", held = function() return { dollars = 1 } end },
    { "m_steel", "m_lucky", held = function(c)
        local ret = {}
        if chance(c, "sj_combo_sl_m", 5) then ret.mult = 20 end
        if chance(c, "sj_combo_sl_d", 15) then ret.dollars = 20 end
        if next(ret) then return ret end
    end },
    { "m_stone", "m_gold", scored = function() return { dollars = 2 } end },
    { "m_stone", "m_lucky", scored = function(c) if chance(c, "sj_combo_stl", 5) then return { chips = 50 } end end },
    { "m_gold", "m_lucky", scored = function(c) if chance(c, "sj_combo_gol", 5) then return { dollars = 3 } end end },
}

local function combos_of(c)
    local list = {}
    if not (c and c.playing_card and c.ability and c.ability.sj_extra_enh) then return list end
    for _, combo in ipairs(COMBOS) do
        if SMODS.has_enhancement(c, combo[1]) and SMODS.has_enhancement(c, combo[2]) then list[#list + 1] = combo end
    end
    return list
end

local eval_card_ref = eval_card
function eval_card(card, context, ...)
    local ret, post = eval_card_ref(card, context, ...)
    if card and card.playing_card and not card.debuff and context.main_scoring and not context.extra_enhancement
            and (context.cardarea == G.play or context.cardarea == G.hand) then
        local field = context.cardarea == G.play and "scored" or "held"
        for _, combo in ipairs(combos_of(card)) do
            local eff = combo[field] and combo[field](card, context)
            if eff then
                eff.card = eff.card or card
                ret = ret or {}
                ret.enhancement = combine(ret.enhancement, eff)
            end
        end
    end
    return ret, post
end

local pseudorandom_probability_ref = SMODS.pseudorandom_probability
function SMODS.pseudorandom_probability(trigger_obj, seed, ...)
    local result = pseudorandom_probability_ref(trigger_obj, seed, ...)
    if result and seed == "glass" and type(trigger_obj) == "table" then
        for _, combo in ipairs(combos_of(trigger_obj)) do
            if combo.glass_safe and combo.glass_safe(trigger_obj) then return false end
        end
    end
    return result
end

local combo_shatter_ref = Card.shatter
function Card:shatter(...)
    if self.playing_card and no_card_destroy == 0 and not self.sj_shattered then
        self.sj_shattered = true
        for _, combo in ipairs(combos_of(self)) do
            if combo.shatter then combo.shatter(self) end
        end
    end
    return combo_shatter_ref(self, ...)
end

local has_no_rank_ref = SMODS.has_no_rank
function SMODS.has_no_rank(card)
    if not has_no_rank_ref(card) then return false end
    for _, combo in ipairs(combos_of(card)) do
        if combo.keeps_rank then return false end
    end
    return true
end

local has_no_suit_ref = SMODS.has_no_suit
function SMODS.has_no_suit(card)
    if not has_no_suit_ref(card) then return false end
    for _, combo in ipairs(combos_of(card)) do
        if combo.keeps_suit then return false end
    end
    return true
end

-- Gold and Blue Seals are checked directly by vanilla rather than through
-- calculate_seal, so extra copies of them are handled here
local function extra_seal_count(card, seal)
    local n = 0
    for _, s in ipairs(card.ability and card.ability.sj_extra_seals or {}) do
        if s == seal then n = n + 1 end
    end
    return n
end

local get_p_dollars_ref = Card.get_p_dollars
function Card:get_p_dollars(...)
    local ret = get_p_dollars_ref(self, ...)
    if not self.debuff and type(ret) == "number" then
        ret = ret + 3 * extra_seal_count(self, "Gold")
    end
    return ret
end

local get_end_of_round_effect_ref = Card.get_end_of_round_effect
function Card:get_end_of_round_effect(context, ...)
    local ret = get_end_of_round_effect_ref(self, context, ...)
    if self.debuff or self.ability.extra_enhancement then return ret end
    for _ = 1, extra_seal_count(self, "Blue") do
        if #G.consumeables.cards + G.GAME.consumeable_buffer < G.consumeables.config.card_limit then
            G.GAME.consumeable_buffer = G.GAME.consumeable_buffer + 1
            G.E_MANAGER:add_event(Event({
                trigger = "before",
                delay = 0.0,
                func = function()
                    if G.GAME.last_hand_played then
                        local planet
                        for _, v in pairs(G.P_CENTER_POOLS.Planet) do
                            if v.config.hand_type == G.GAME.last_hand_played then planet = v.key end
                        end
                        local c = create_card("Planet", G.consumeables, nil, nil, nil, nil, planet, "blusl")
                        c:add_to_deck()
                        G.consumeables:emplace(c)
                    end
                    G.GAME.consumeable_buffer = 0
                    return true
                end,
            }))
            card_eval_status_text(self, "extra", nil, nil, nil,
                { message = localize("k_plus_planet"), colour = G.C.SECONDARY_SET.Planet })
        end
    end
    return ret
end

----------------------------------------------------------------------
-- Fused rank: playing cards whose number is the chips of the two fused cards added up
----------------------------------------------------------------------

SMODS.Atlas { key = "fused_rank", path = "fused_rank.png", px = 71, py = 95 }

local FUSED_RANK = SMODS.Rank {
    key = "fused",
    card_key = "SJF",
    loc_txt = { name = "Fused" },
    pos = { x = 0 },
    nominal = 0,
    shorthand = "F",
    lc_atlas = "sj_fused_rank",
    hc_atlas = "sj_fused_rank",
    suit_map = { Hearts = 0, Clubs = 1, Diamonds = 2, Spades = 3 },
    next = {},
    strength_effect = { fixed = 0, random = false, ignore = true },
    in_pool = function() return false end,
}

local function apply_fused_number(card)
    local n = card.ability and card.ability.sj_number
    if n and card.base and card.base.value == FUSED_RANK.key then
        card.base.nominal = n
        -- Some vanilla code (Raised Fist, rank sorting) reads base.id directly instead of
        -- get_id(), so it must be the card's main rank, not the placeholder Fused rank
        local ranks = card.ability.sj_ranks
        local main = ranks and ranks[1] and SMODS.Ranks[ranks[1]]
        if main then card.base.id = main.id end
    end
end

-- A fused playing card counts as every rank and suit of the cards fused into it.
-- sj_ranks is a list of rank keys (first one is its main rank), sj_suits a set.
local function fused_ranks(card)
    local ranks = card.ability and card.ability.sj_ranks
    if ranks and card.base and card.base.value == FUSED_RANK.key then return ranks end
end

local get_id_ref = Card.get_id
function Card:get_id(...)
    if self.sj_force_id then return self.sj_force_id end
    local ranks = fused_ranks(self)
    if ranks and ranks[1] and SMODS.Ranks[ranks[1]] and not SMODS.has_no_rank(self) then
        return SMODS.Ranks[ranks[1]].id
    end
    return get_id_ref(self, ...)
end

local is_face_ref = Card.is_face
function Card:is_face(from_boss, ...)
    if is_face_ref(self, from_boss, ...) then return true end
    if self.debuff and not from_boss then return false end
    for _, key in ipairs(fused_ranks(self) or {}) do
        if SMODS.Ranks[key] and SMODS.Ranks[key].face then return true end
    end
    return false
end

local is_suit_ref = Card.is_suit
function Card:is_suit(suit, bypass_debuff, flush_calc, ...)
    if is_suit_ref(self, suit, bypass_debuff, flush_calc, ...) then return true end
    local suits = fused_ranks(self) and self.ability.sj_suits
    if not (suits and suits[suit]) or SMODS.has_no_suit(self) then return false end
    return flush_calc or bypass_debuff or not self.debuff
end

-- Try every combination of ranks for the multi-rank cards in a hand and keep the best result
local function with_rank_choices(hand, fn, score)
    local multi = {}
    for _, c in ipairs(hand or {}) do
        local ranks = fused_ranks(c)
        if ranks and #ranks > 1 and not SMODS.has_no_rank(c) then multi[#multi + 1] = c end
    end
    if #multi == 0 then return fn() end
    local best, best_score
    local function try(i)
        if i > #multi then
            local result = fn()
            local s = score(result)
            if not best_score or s > best_score then best, best_score = result, s end
            return
        end
        for _, key in ipairs(fused_ranks(multi[i])) do
            if SMODS.Ranks[key] then
                multi[i].sj_force_id = SMODS.Ranks[key].id
                try(i + 1)
            end
        end
        multi[i].sj_force_id = nil
    end
    local ok, err = pcall(try, 1)
    for _, c in ipairs(multi) do c.sj_force_id = nil end
    if not ok then error(err, 0) end
    return best
end

local function cards_in(groups)
    local n = 0
    for _, g in ipairs(groups or {}) do n = n + #g end
    return n * 10 + #(groups or {})
end

local get_X_same_ref = get_X_same
function get_X_same(num, hand, or_more, ...)
    local rest = { ... }
    return with_rank_choices(hand, function() return get_X_same_ref(num, hand, or_more, unpack(rest)) end, cards_in)
end

local get_straight_ref = get_straight
function get_straight(hand, ...)
    local rest = { ... }
    return with_rank_choices(hand, function() return get_straight_ref(hand, unpack(rest)) end, cards_in)
end

-- Jokers that check a rank (8 Ball, Scholar, Baron...) see each of the card's ranks
local fused_calculate_joker_ref = Card.calculate_joker
function Card:calculate_joker(context, ...)
    local other = context and context.other_card
    local ranks = other and other.playing_card and fused_ranks(other)
    if not (ranks and #ranks > 1) or other.sj_force_id then
        return fused_calculate_joker_ref(self, context, ...)
    end
    local ret, post = fused_calculate_joker_ref(self, context, ...)
    if ret then return ret, post end
    for i = 2, #ranks do
        local rank = SMODS.Ranks[ranks[i]]
        if rank then
            other.sj_force_id = rank.id
            local ok, alt, alt_post = pcall(fused_calculate_joker_ref, self, context, ...)
            other.sj_force_id = nil
            if not ok then error(alt, 0) end
            if alt then return alt, alt_post end
        end
    end
    return ret, post
end

local set_base_ref = Card.set_base
function Card:set_base(card, initial, ...)
    local ret = set_base_ref(self, card, initial, ...)
    apply_fused_number(self)
    return ret
end

-- Keep the number correct (also after loading a save) and draw it on the card
local card_update_ref = Card.update
function Card:update(dt)
    card_update_ref(self, dt)
    local n = self.ability and self.ability.sj_number
    if n and self.base and self.base.value == FUSED_RANK.key then
        apply_fused_number(self)
        local label = self.children.sj_number
        if not label or label.sj_n ~= n then
            if label then label:remove() end
            local colour = G.C.SUITS[self.base.suit] or G.C.BLACK
            local text = tostring(n)
            label = UIBox {
                definition = { n = G.UIT.ROOT, config = { align = "cm", colour = G.C.CLEAR }, nodes = {
                    { n = G.UIT.T, config = { text = text, scale = #text > 3 and 0.45 or 0.6, colour = colour, shadow = true } },
                } },
                config = { align = "cm", offset = { x = 0, y = 0 }, parent = self, bond = "Strong" },
            }
            -- Never block hovering or clicking the card underneath
            label.states.collide.can = false
            label.sj_n = n
            self.children.sj_number = label
        end
        label.states.visible = self.sprite_facing == "front"
    elseif self.children.sj_number then
        self.children.sj_number:remove()
        self.children.sj_number = nil
    end
end

-- The chips a card gives on its own: its number, its enhancement's chips (Bonus, Stone,
-- stacked extras) and any permanent bonus chips (Hiker...)
local function card_chips(card)
    local chips = 0
    if card.base.value == FUSED_RANK.key and card.ability.sj_number then
        chips = card.ability.sj_number
    elseif not SMODS.has_no_rank(card) then
        chips = card.base.nominal or 0
    end
    chips = chips + (card.ability.bonus or 0) + (card.ability.perma_bonus or 0)
    for key in pairs(card.ability.sj_extra_enh or {}) do
        local config = G.P_CENTERS[key] and G.P_CENTERS[key].config
        chips = chips + (config and config.bonus or 0)
    end
    return chips
end

local function card_ranks(card)
    if fused_ranks(card) then return card.ability.sj_ranks end
    if SMODS.has_no_rank(card) then return {} end
    return { card.base.value }
end

local function card_suits(card)
    if fused_ranks(card) and card.ability.sj_suits then return card.ability.sj_suits end
    if SMODS.has_no_suit(card) then return {} end
    return { [card.base.suit] = true }
end

local function fuse_playing_cards(keep, other)
    local n = card_chips(keep) + card_chips(other)
    local ranks, seen = {}, {}
    for _, c in ipairs({ keep, other }) do
        for _, key in ipairs(card_ranks(c)) do
            if not seen[key] then
                seen[key] = true
                ranks[#ranks + 1] = key
            end
        end
    end
    local suits = {}
    for _, c in ipairs({ keep, other }) do
        for suit in pairs(card_suits(c)) do suits[suit] = true end
    end

    -- Collect everything from both cards
    local enhancements, seals, editions = {}, {}, {}
    local function gather(c)
        if c.config.center.key ~= "c_base" then enhancements[#enhancements + 1] = c.config.center.key end
        for key in pairs(c.ability.sj_extra_enh or {}) do enhancements[#enhancements + 1] = key end
        if c.seal then seals[#seals + 1] = c.seal end
        for _, s in ipairs(c.ability.sj_extra_seals or {}) do seals[#seals + 1] = s end
        if c.edition and c.edition.key then editions[#editions + 1] = c.edition.key end
        for _, e in ipairs(c.ability.sj_extra_editions or {}) do editions[#editions + 1] = e end
    end
    gather(keep)
    gather(other)

    SMODS.destroy_cards({ other })

    G.E_MANAGER:add_event(Event({
        trigger = "after",
        delay = 0.2,
        func = function()
            keep:flip()
            play_sound("tarot1")
            return true
        end,
    }))
    G.E_MANAGER:add_event(Event({
        trigger = "after",
        delay = 0.3,
        func = function()
            -- Main enhancement / seal / edition first, the rest stack on as extras
            if enhancements[1] and keep.config.center.key ~= enhancements[1] then
                set_ability_ref(keep, G.P_CENTERS[enhancements[1]])
            end
            for i = 2, #enhancements do add_extra_enhancement(keep, enhancements[i]) end
            if seals[1] and keep.seal ~= seals[1] then set_seal_ref(keep, seals[1], true, true) end
            for i = 2, #seals do add_extra_seal(keep, seals[i]) end
            if editions[1] and not (keep.edition and keep.edition.key == editions[1]) then
                set_edition_ref(keep, editions[1], true, true)
            end
            for i = 2, #editions do add_extra_edition(keep, editions[i]) end
            -- Permanent bonus chips are now part of the number
            keep.ability.perma_bonus = 0
            keep.ability.sj_number = n
            keep.ability.sj_ranks = ranks
            keep.ability.sj_suits = suits
            SMODS.change_base(keep, nil, FUSED_RANK.key)
            apply_fused_number(keep)
            SMODS.enh_cache:write(keep, nil)
            keep:flip()
            keep:juice_up(0.3, 0.5)
            return true
        end,
    }))
end

----------------------------------------------------------------------
-- The Fusion consumable type and the Fuse card
----------------------------------------------------------------------

local FUSE_KEY = "c_sj_fuse"

-- Works out what the player has selected to fuse: returns kind, a, b or nil
local function fuse_selection(fuse_card)
    local jokers = G.jokers and G.jokers.highlighted or {}
    local hand = G.hand and G.hand.highlighted or {}
    local consumables = {}
    for _, c in ipairs(G.consumeables and G.consumeables.highlighted or {}) do
        if c ~= fuse_card then consumables[#consumables + 1] = c end
    end

    local groups = (#jokers > 0 and 1 or 0) + (#hand > 0 and 1 or 0) + (#consumables > 0 and 1 or 0)
    if groups ~= 1 then return end

    if #jokers == 2 then
        -- A Joker can only be fused once
        if jokers[1].config.center.key == FUSED_JOKER or jokers[2].config.center.key == FUSED_JOKER then return end
        return "joker", jokers[1], jokers[2]
    end
    if #hand == 2 then
        local a, b = hand[1], hand[2]
        if b.T.x < a.T.x then a, b = b, a end
        return "card", a, b
    end
    if #consumables == 2 then
        local a, b = consumables[1], consumables[2]
        if base_set(a) ~= base_set(b) or not FUSED_CONSUMABLE[base_set(a)] then return end
        return "consumable", a, b
    end
end

SMODS.Consumable {
    key = "fuse",
    set = "Fusion",
    loc_txt = {
        name = "Fuse",
        text = {
            "Fuse {C:attention}2{} selected Jokers,",
            "{C:attention}2{} consumables of the same type,",
            "or {C:attention}2{} playing cards into one",
        },
    },
    cost = 4,
    atlas = "fusion",
    pos = { x = 0, y = 0 },
    loc_vars = function(self, info_queue, card)
        info_queue[#info_queue + 1] = {
            set = "Other",
            key = "sj_fuse_rules",
        }
    end,
    can_use = function(self, card)
        return fuse_selection(card) ~= nil
    end,
    use = function(self, card, area, copier)
        local kind, a, b = fuse_selection(card)
        if not kind then return end
        G.E_MANAGER:add_event(Event({
            trigger = "after",
            delay = 0.3,
            func = function()
                play_sound("tarot1")
                card:juice_up(0.3, 0.5)
                if kind == "joker" then
                    fuse_jokers(a, b)
                    G.jokers:unhighlight_all()
                elseif kind == "consumable" then
                    fuse_consumables(a, b)
                    G.consumeables:unhighlight_all()
                else
                    fuse_playing_cards(a, b)
                    G.E_MANAGER:add_event(Event({
                        trigger = "after",
                        delay = 0.2,
                        func = function()
                            G.hand:unhighlight_all()
                            return true
                        end,
                    }))
                end
                return true
            end,
        }))
    end,
}

SMODS.current_mod.process_loc_text = function()
    G.localization.misc.dictionary.k_sj_fusion_pack = "Fusion Pack"
    for key, entry in pairs(SimplyJokers.fusion.loc) do
        SMODS.process_loc_text(G.localization.descriptions.Other, key, entry)
    end
    SMODS.process_loc_text(G.localization.descriptions.Other, "sj_fuse_rules", {
        name = "Fusing",
        text = {
            "{C:attention}Jokers:{} one Joker with both",
            "abilities that trigger together,",
            "without their downsides. Copies of",
            "the same Joker retrigger it.",
            "A Joker can only be fused once.",
            "{C:attention}Invisible Joker:{} copies the other Joker",
            "{C:attention}Consumables:{} one card that does both.",
            "Tarot and Spectral pairs get a special",
            "effect instead (shown on the card)",
            "{C:attention}Playing cards:{} keeps every enhancement,",
            "seal and edition. Its number is the",
            "chips both cards gave, added up, and",
            "it counts as every rank and suit of both",
        },
    })
end

-- While a Fuse is held, allow selecting 2 Jokers and 2 consumables at once
local raised = nil

local game_update_ref = Game.update
function Game:update(dt)
    game_update_ref(self, dt)
    if not (G.jokers and G.consumeables) then return end
    local has_fuse = false
    for _, c in ipairs(G.consumeables.cards) do
        if c.config.center.key == FUSE_KEY then has_fuse = true end
    end
    if raised and raised.jokers ~= G.jokers then raised = nil end
    if has_fuse and not raised then
        raised = {
            jokers = G.jokers,
            joker_limit = G.jokers.config.highlighted_limit,
            consumable_limit = G.consumeables.config.highlighted_limit,
        }
        G.jokers.config.highlighted_limit = math.max(raised.joker_limit or 1, 2)
        G.consumeables.config.highlighted_limit = math.max(raised.consumable_limit or 1, 3)
    elseif not has_fuse and raised then
        G.jokers.config.highlighted_limit = raised.joker_limit
        G.consumeables.config.highlighted_limit = raised.consumable_limit
        if #G.jokers.highlighted > (raised.joker_limit or 1) then G.jokers:unhighlight_all() end
        if #G.consumeables.highlighted > (raised.consumable_limit or 1) then G.consumeables:unhighlight_all() end
        raised = nil
    end
end

----------------------------------------------------------------------
-- Fusion Packs: a mix of Fuse cards and ready-made fused Tarots, Planets and Spectrals
----------------------------------------------------------------------

SMODS.Atlas { key = "fusion_packs", path = "fusion_packs.png", px = 71, py = 95 }

-- A random consumable key from a set, skipping hidden cards (The Soul, Black Hole)
local function random_part_key(set, seed)
    local pool = {}
    for _, center in ipairs(G.P_CENTER_POOLS[set] or {}) do
        if not center.hidden and center.key ~= FUSED_CONSUMABLE[set] then pool[#pool + 1] = center.key end
    end
    return pseudorandom_element(pool, pseudoseed(seed))
end

-- The exact ability a real card of this key would start with
local function fresh_ability(key)
    local tmp = Card(0, 0, G.CARD_W, G.CARD_H, nil, G.P_CENTERS[key])
    local ability = copy_table(tmp.ability)
    tmp:remove()
    return ability
end

local PACK_SETS = { { "Tarot", 5 }, { "Planet", 3 }, { "Spectral", 2 } }

local function pack_card(self, pack, i)
    -- 1 in 3 slots is a Fuse (at most one per pack), the rest are pre-fused pairs
    if not pack.sj_fuse_given and pseudorandom("sj_fusion_pack_kind") < 1 / 3 then
        pack.sj_fuse_given = true
        return { set = "Fusion", key = FUSE_KEY, area = G.pack_cards, skip_materialize = true, key_append = "sj_fpk" }
    end
    local roll, set = pseudorandom("sj_fusion_pack_set") * 10, "Tarot"
    for _, entry in ipairs(PACK_SETS) do
        if roll < entry[2] then set = entry[1] break end
        roll = roll - entry[2]
    end
    local parts = {}
    for k = 1, 2 do
        local key = random_part_key(set, "sj_fusion_pack_part" .. k)
        parts[k] = { key = key, ability = fresh_ability(key), copies = 1 }
    end
    local card = SMODS.create_card({
        set = "Fusion", key = FUSED_CONSUMABLE[set], area = G.pack_cards,
        skip_materialize = true, key_append = "sj_fpk", no_edition = true,
    })
    card.ability.extra = { parts = parts, part_set = set }
    return card
end

local FUSION_COLOUR = HEX("E8743B")

local function fusion_pack(key, label, size, choose, cost, weight, pos)
    SMODS.Booster {
        key = key,
        kind = "Fusion",
        group_key = "k_sj_fusion_pack",
        loc_txt = {
            name = label,
            text = {
                "Choose {C:attention}#1#{} of up to",
                "{C:attention}#2#{} {C:attention}Fusion{} cards to",
                "be used immediately",
            },
            group_name = "Fusion Pack",
        },
        config = { extra = size, choose = choose },
        cost = cost,
        weight = weight,
        atlas = "fusion_packs",
        pos = pos,
        draw_hand = true,
        discovered = true,
        unlocked = true,
        create_card = pack_card,
        ease_background_colour = function(self)
            ease_colour(G.C.DYN_UI.MAIN, FUSION_COLOUR)
            ease_background_colour({ new_colour = FUSION_COLOUR, special_colour = G.C.BLACK, contrast = 2 })
        end,
    }
end

fusion_pack("fusion_normal", "Fusion Pack", 3, 1, 4, 0.6, { x = 0, y = 0 })
fusion_pack("fusion_jumbo", "Jumbo Fusion Pack", 5, 1, 6, 0.6, { x = 1, y = 0 })
fusion_pack("fusion_mega", "Mega Fusion Pack", 5, 2, 8, 0.15, { x = 2, y = 0 })

assert(SMODS.load_file("fusion_specials.lua"))()
