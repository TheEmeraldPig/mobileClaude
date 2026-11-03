--- JokerDisplay definitions for SimplyJokers.
--- Each definition is attached to its Joker as joker_display_def, which JokerDisplay
--- reads lazily, so this works no matter which mod loads first.

local SJ = SimplyJokers

local function odds_text(card, num, den)
    local n, d = SJ.prob_vars(card, num, den)
    return localize { type = "variable", key = "jdis_odds", vars = { n, d } }
end

local function active_text(active)
    return localize("jdis_" .. (active and "active" or "inactive"))
end

-- Played (or highlighted) cards, left to right
local function full_hand()
    local hand = next(G.play.cards) and G.play.cards or G.hand.highlighted
    local cards = {}
    for _, c in ipairs(hand) do cards[#cards + 1] = c end
    table.sort(cards, function(a, b) return a.T.x < b.T.x end)
    return cards
end

local function triggers(c, scoring_hand)
    return JokerDisplay.calculate_card_triggers(c, scoring_hand)
end

local function in_scoring(c, scoring_hand)
    return JokerDisplay.in_scoring(c, scoring_hand)
end

-- Reusable display pieces
local function plus(ref, colour)
    return {
        text = { { text = "+" }, { ref_table = "card.joker_display_values", ref_value = ref, retrigger_type = "mult" } },
        text_config = { colour = colour },
    }
end

local function xmult(ref)
    return {
        { border_nodes = { { text = "X" }, { ref_table = "card.joker_display_values", ref_value = ref, retrigger_type = "exp" } } },
    }
end

local function dollars(ref)
    return {
        { text = "+$" },
        { ref_table = "card.joker_display_values", ref_value = ref, retrigger_type = "mult" },
    }
end

local function reminder(ref)
    return { { text = "(" }, { ref_table = "card.joker_display_values", ref_value = ref }, { text = ")" } }
end

local function odds_row(ref)
    return { { text = "(" }, { ref_table = "card.joker_display_values", ref_value = ref }, { text = ")" } }
end

local function colour_active(child_index)
    return function(card, text, reminder_text, extra)
        if reminder_text and reminder_text.children and reminder_text.children[child_index] then
            reminder_text.children[child_index].config.colour =
                card.joker_display_values.is_active and G.C.GREEN or G.C.UI.TEXT_INACTIVE
        end
    end
end

local defs = {}

defs.snowball = function()
    local d = plus("chips", G.C.CHIPS)
    d.reminder_text = { { text = "(+" }, { ref_table = "card.joker_display_values", ref_value = "gain" }, { text = " per hand)" } }
    d.calc_function = function(card)
        card.joker_display_values.chips = card.ability.extra.chips
        card.joker_display_values.gain = card.ability.extra.chip_gain * card.config.center:upgrades()
    end
    return d
end

defs.double_down = function()
    return {
        text = xmult("x_mult"),
        calc_function = function(card)
            card.joker_display_values.x_mult = card.ability.extra.xmult
        end,
    }
end

defs.royal_guard = function()
    return { reminder_text = { { text = "(J = Q = K)" } } }
end

defs.piggy_bank = function()
    return {
        text = { { text = "$" }, { ref_table = "card.ability.extra", ref_value = "stored" } },
        text_config = { colour = G.C.GOLD },
        reminder_text = { { text = "(stored)" } },
        extra = { odds_row("odds") },
        extra_config = { colour = G.C.GREEN, scale = 0.3 },
        calc_function = function(card)
            card.joker_display_values.odds = odds_text(card, 1, card.ability.extra.odds)
        end,
    }
end

defs.last_stand = function()
    return {
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            card.joker_display_values.is_active = G.GAME.current_round.hands_left <= 1
            card.joker_display_values.active_text = active_text(card.joker_display_values.is_active)
        end,
        style_function = colour_active(2),
        retrigger_function = function(playing_card, scoring_hand, held_in_hand, joker_card)
            if held_in_hand or G.GAME.current_round.hands_left > 1 then return 0 end
            return in_scoring(playing_card, scoring_hand)
                and joker_card.ability.extra.repetitions * JokerDisplay.calculate_joker_triggers(joker_card) or 0
        end,
    }
end

defs.loose_change = function()
    return {
        text = { { border_nodes = { { text = "X0.5-X5" } } } },
        reminder_text = { { text = "(-$1 to -$10)" } },
    }
end

local function hand_has(name)
    local text, poker_hands = JokerDisplay.evaluate_hand()
    return text ~= "Unknown" and poker_hands and poker_hands[name] and next(poker_hands[name]) and true or false
end

defs.rerun = function()
    return {
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            card.joker_display_values.is_active = hand_has("Straight")
            card.joker_display_values.active_text = active_text(card.joker_display_values.is_active)
        end,
        style_function = colour_active(2),
        retrigger_function = function(playing_card, scoring_hand, held_in_hand, joker_card)
            if held_in_hand or not hand_has("Straight") then return 0 end
            return in_scoring(playing_card, scoring_hand) and JokerDisplay.calculate_joker_triggers(joker_card) or 0
        end,
    }
end

defs.bouquet = function()
    return {
        text = {
            { text = "+", colour = G.C.MULT },
            { ref_table = "card.joker_display_values", ref_value = "mult", colour = G.C.MULT, retrigger_type = "mult" },
            { text = " +", colour = G.C.CHIPS },
            { ref_table = "card.joker_display_values", ref_value = "chips", colour = G.C.CHIPS, retrigger_type = "mult" },
        },
        calc_function = function(card)
            local mult, chips = 0, 0
            local light, dark = false, false
            for _, c in ipairs(full_hand()) do
                light = light or SJ.is_light(c)
                dark = dark or SJ.is_dark(c)
            end
            local text, _, scoring_hand = JokerDisplay.evaluate_hand()
            if light and dark and text ~= "Unknown" then
                for _, c in pairs(scoring_hand) do
                    local t = triggers(c, scoring_hand)
                    if SJ.is_light(c) then mult = mult + card.ability.extra.mult * t end
                    if SJ.is_dark(c) then chips = chips + card.ability.extra.chips * t end
                end
            end
            card.joker_display_values.mult = mult
            card.joker_display_values.chips = chips
        end,
    }
end

defs.odd_jobs = function()
    return {
        text = dollars("dollars"),
        text_config = { colour = G.C.GOLD },
        reminder_text = { { text = "(A,9,7,5,3)" } },
        calc_function = function(card)
            local hand = full_hand()
            local text, _, scoring_hand = JokerDisplay.evaluate_hand()
            local ok = text ~= "Unknown" and #scoring_hand > 0 and not (text == "High Card" and #hand < 5)
            if ok then
                for _, c in pairs(scoring_hand) do
                    if not SJ.is_odd(c) then ok = false end
                end
            end
            card.joker_display_values.dollars = ok and card.ability.extra.dollars or 0
        end,
    }
end

defs.late_bloomer = function()
    local d = plus("mult", G.C.MULT)
    d.calc_function = function(card)
        card.joker_display_values.mult = card.ability.extra.mult * SJ.current_ante()
    end
    return d
end

defs.hot_potato = function()
    return {
        text = xmult("x_mult"),
        extra = { odds_row("odds") },
        extra_config = { colour = G.C.GREEN, scale = 0.3 },
        calc_function = function(card)
            card.joker_display_values.x_mult = card.ability.extra.xmult
            card.joker_display_values.odds = odds_text(card, 1, card.ability.extra.odds)
        end,
    }
end

defs.consolation_prize = function()
    return {
        text = dollars("dollars"),
        text_config = { colour = G.C.GOLD },
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            card.joker_display_values.is_active = not card.ability.extra.bought
            card.joker_display_values.dollars = card.ability.extra.dollars
            card.joker_display_values.active_text = active_text(card.joker_display_values.is_active)
        end,
        style_function = colour_active(2),
    }
end

defs.copycat = function()
    return {
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            local hand = full_hand()
            local first, last = hand[1], hand[#hand]
            card.joker_display_values.is_active = #hand >= 2 and SJ.has_rank(first) and SJ.has_rank(last)
                and first:get_id() == last:get_id()
            card.joker_display_values.active_text = active_text(card.joker_display_values.is_active)
        end,
        style_function = colour_active(2),
        retrigger_function = function(playing_card, scoring_hand, held_in_hand, joker_card)
            if held_in_hand then return 0 end
            local hand = full_hand()
            local first, last = hand[1], hand[#hand]
            if #hand >= 2 and SJ.has_rank(first) and SJ.has_rank(last) and first:get_id() == last:get_id()
                    and (playing_card == first or playing_card == last) and in_scoring(playing_card, scoring_hand) then
                return JokerDisplay.calculate_joker_triggers(joker_card)
            end
            return 0
        end,
    }
end

defs.rust_bucket = function()
    return {
        text = dollars("dollars"),
        text_config = { colour = G.C.GOLD },
        reminder_text = { { text = "(End of round)" } },
        calc_function = function(card)
            local count = 0
            for _, c in ipairs(G.hand and G.hand.cards or {}) do
                if not c.highlighted and not c.debuff and SMODS.has_enhancement(c, "m_steel") then count = count + 1 end
            end
            card.joker_display_values.dollars = count * card.ability.extra.dollars
        end,
    }
end

defs.echo_chamber = function()
    return {
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            card.joker_display_values.is_active = not card.ability.extra.used
            card.joker_display_values.active_text = active_text(card.joker_display_values.is_active)
        end,
        style_function = colour_active(2),
    }
end

defs.card_counter = function()
    return {
        text = xmult("x_mult"),
        calc_function = function(card)
            card.joker_display_values.x_mult = card.config.center:current_xmult(card)
        end,
    }
end

defs.gamblers_fallacy = function()
    return {
        text = { { border_nodes = { { text = "X4" } } } },
        reminder_text = { { text = "(or halve)" } },
        extra = { odds_row("odds") },
        extra_config = { colour = G.C.GREEN, scale = 0.3 },
        calc_function = function(card)
            card.joker_display_values.odds = odds_text(card, 1, card.ability.extra.odds)
        end,
    }
end

defs.monarchy = function()
    return {
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            local text, _, scoring_hand = JokerDisplay.evaluate_hand()
            local active = text ~= "Unknown" and #scoring_hand > 0
            if active then
                for _, c in pairs(scoring_hand) do
                    if not c:is_face() then active = false end
                end
            end
            card.joker_display_values.is_active = active
            card.joker_display_values.active_text = active_text(active)
        end,
        style_function = colour_active(2),
    }
end

defs.collector = function()
    return {
        text = xmult("x_mult"),
        reminder_text = { { text = "(" }, { ref_table = "card.joker_display_values", ref_value = "count" }, { text = " Jokers)" } },
        calc_function = function(card)
            local n = SJ.collected_count()
            card.joker_display_values.count = n
            card.joker_display_values.x_mult = 1 + card.ability.extra.xmult_per * n
        end,
    }
end

defs.ouroboros = function()
    return {
        reminder_text = { { text = "(" }, { ref_table = "card.joker_display_values", ref_value = "count" }, { text = " absorbed)" } },
        calc_function = function(card)
            card.joker_display_values.count = #card.ability.extra.absorbed
        end,
    }
end

defs.spare_parts = function()
    local d = plus("chips", G.C.CHIPS)
    d.calc_function = function(card)
        card.joker_display_values.chips = card.ability.extra.chips
    end
    return d
end

defs.night_owl = function()
    local d = plus("mult", G.C.MULT)
    d.calc_function = function(card)
        local hand = full_hand()
        local ok = #hand > 0
        for _, c in ipairs(hand) do
            if not SJ.is_dark(c) then ok = false end
        end
        card.joker_display_values.mult = ok and card.ability.extra.mult or 0
    end
    return d
end

defs.early_bird = function()
    local d = plus("chips", G.C.CHIPS)
    d.calc_function = function(card)
        local ok = G.GAME.current_round.hands_played == 0 and #full_hand() == 5
        card.joker_display_values.chips = ok and card.ability.extra.chips or 0
    end
    return d
end

defs.middle_child = function()
    return {
        retrigger_function = function(playing_card, scoring_hand, held_in_hand, joker_card)
            if held_in_hand or not in_scoring(playing_card, scoring_hand) then return 0 end
            local hand = full_hand()
            local n = #hand
            local reps = 0
            if (n == 3 or n == 5) and playing_card == hand[(n + 1) / 2] then
                reps = 2
            elseif n == 4 and (playing_card == hand[2] or playing_card == hand[3]) then
                reps = 1
            end
            return reps * JokerDisplay.calculate_joker_triggers(joker_card)
        end,
    }
end

defs.mirror_match = function()
    return {
        text = xmult("x_mult"),
        calc_function = function(card)
            local e = card.ability.extra
            local streak
            if next(G.play.cards) then
                streak = e.streak
            else
                local text = JokerDisplay.evaluate_hand()
                streak = (text ~= "Unknown" and text == e.last) and e.streak + 1 or 0
            end
            card.joker_display_values.x_mult = streak > 0 and e.base + e.gain * (streak - 1) or 1
        end,
    }
end

defs.undertaker = function()
    local d = plus("mult", G.C.MULT)
    d.calc_function = function(card)
        card.joker_display_values.mult = card.ability.extra.mult
    end
    return d
end

defs.wild_west = function()
    return {
        text = xmult("x_mult"),
        extra = { odds_row("odds") },
        extra_config = { colour = G.C.GREEN, scale = 0.3 },
        calc_function = function(card)
            local count = 0
            local text, _, scoring_hand = JokerDisplay.evaluate_hand()
            if text ~= "Unknown" then
                for _, c in pairs(scoring_hand) do
                    if SMODS.has_enhancement(c, "m_wild") then count = count + triggers(c, scoring_hand) end
                end
            end
            card.joker_display_values.x_mult = card.ability.extra.xmult ^ count
            card.joker_display_values.odds = odds_text(card, 1, card.ability.extra.odds)
        end,
    }
end

defs.overtime = function()
    return {
        text = xmult("x_mult"),
        calc_function = function(card)
            card.joker_display_values.x_mult = card.ability.extra.xmult
        end,
    }
end

defs.deep_pockets = function()
    return {
        text = { { text = "+" }, { ref_table = "card.ability.extra", ref_value = "applied" } },
        text_config = { colour = G.C.SECONDARY_SET.Tarot },
        reminder_text = { { text = "(consumable slots)" } },
    }
end

defs.glass_cannon = function()
    return { reminder_text = { { text = "(Glass X+1, always shatters)" } } }
end

local function clockwork_active(card)
    local e = card.ability.extra
    local count = next(G.play.cards) and e.count or e.count + 1
    return count % e.every == 0
end

defs.clockwork = function()
    return {
        reminder_text = reminder("active_text"),
        calc_function = function(card)
            local e = card.ability.extra
            card.joker_display_values.is_active = clockwork_active(card)
            card.joker_display_values.active_text = card.joker_display_values.is_active and active_text(true)
                or ((e.every - (e.count % e.every)) .. " left")
        end,
        style_function = colour_active(2),
        retrigger_function = function(playing_card, scoring_hand, held_in_hand, joker_card)
            if held_in_hand or not clockwork_active(joker_card) or not in_scoring(playing_card, scoring_hand) then
                return 0
            end
            return joker_card.ability.extra.repetitions * JokerDisplay.calculate_joker_triggers(joker_card)
        end,
    }
end

defs.jester_king = function()
    return {
        reminder_text = { { text = "(Jokers trigger twice)" } },
        extra = { odds_row("odds") },
        extra_config = { colour = G.C.GREEN, scale = 0.3 },
        calc_function = function(card)
            card.joker_display_values.odds = odds_text(card, 1, card.ability.extra.odds)
        end,
    }
end

defs.disorder = function()
    return {
        reminder_text = {
            { text = "(" },
            { ref_table = "card.joker_display_values", ref_value = "blueprint_compat", colour = G.C.RED },
            { text = ")" },
        },
        calc_function = function(card)
            local copied_joker, copied_debuff = JokerDisplay.calculate_blueprint_copy(card)
            card.joker_display_values.blueprint_compat = localize("k_incompatible")
            JokerDisplay.copy_display(card, copied_joker, copied_debuff)
        end,
        get_blueprint_joker = function(card)
            return SJ.disorder_target(card)
        end,
    }
end

defs.hungry_hippo = function()
    return {
        reminder_text = { { text = "(on discard)" } },
        extra = {
            { { text = "$1 " }, { text = "(" }, { ref_table = "card.joker_display_values", ref_value = "money_odds" }, { text = ")" } },
            { { text = "Eat " }, { text = "(" }, { ref_table = "card.joker_display_values", ref_value = "eat_odds" }, { text = ")" } },
        },
        extra_config = { colour = G.C.GREEN, scale = 0.3 },
        calc_function = function(card)
            card.joker_display_values.money_odds = odds_text(card, 1, card.ability.extra.money_odds)
            card.joker_display_values.eat_odds = odds_text(card, 1, card.ability.extra.eat_odds)
        end,
    }
end

defs.liquid_joker = function()
    return {
        text = dollars("dollars"),
        text_config = { colour = G.C.GOLD },
        reminder_text = { { text = "(scoring cards)" } },
        calc_function = function(card)
            local count = 0
            local text, _, scoring_hand = JokerDisplay.evaluate_hand()
            if text ~= "Unknown" then
                for _, c in pairs(scoring_hand) do count = count + triggers(c, scoring_hand) end
            end
            card.joker_display_values.dollars = count * card.ability.extra.dollars
        end,
    }
end

defs.fused = function()
    return {
        reminder_text = { { text = "(" }, { ref_table = "card.joker_display_values", ref_value = "count" }, { text = " fused)" } },
        calc_function = function(card)
            card.joker_display_values.count = #card.ability.extra.parts
        end,
    }
end

for key, def in pairs(defs) do
    local center = SMODS.Centers["j_sj_" .. key]
    if center then
        center.joker_display_def = function(JD) return def() end
    end
end
