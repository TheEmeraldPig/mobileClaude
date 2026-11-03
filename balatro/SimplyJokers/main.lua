--- SimplyJokers: 35 new Jokers for Balatro (requires Steamodded 1.0+)

SMODS.Atlas {
    key = "jokers",
    path = "jokers.png",
    px = 71,
    py = 95,
}

-- Snowball: scaling chips every hand played; its upgrade retriggers once per ante
SMODS.Joker {
    key = "snowball",
    loc_txt = {
        name = "Snowball",
        text = {
            "Gains {C:chips}+#2#{} Chips for every hand",
            "played, {C:attention}retriggered 1{} time per {C:attention}Ante",
            "{C:inactive}(Currently {C:chips}+#1#{C:inactive} Chips, {C:chips}+#3#{C:inactive} per hand)",
        },
    },
    config = { extra = { chips = 0, chip_gain = 4 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 0, y = 0 },
    blueprint_compat = true,
    perishable_compat = false,
    -- One upgrade per hand plus one extra per ante (never fewer than one)
    upgrades = function(self)
        local ante = G.GAME and G.GAME.round_resets and G.GAME.round_resets.ante or 1
        return 1 + math.max(ante, 0)
    end,
    loc_vars = function(self, info_queue, card)
        local e = card.ability.extra
        return { vars = { e.chips, e.chip_gain, e.chip_gain * self:upgrades() } }
    end,
    calculate = function(self, card, context)
        if context.before and not context.blueprint then
            local gain = card.ability.extra.chip_gain * self:upgrades()
            card.ability.extra.chips = card.ability.extra.chips + gain
            return { message = "+" .. gain, colour = G.C.CHIPS, card = card }
        end
        if context.joker_main and card.ability.extra.chips > 0 then
            return { chips = card.ability.extra.chips }
        end
    end,
}

-- Double Down: scaling XMult for every Pair played
SMODS.Joker {
    key = "double_down",
    loc_txt = {
        name = "Double Down",
        text = {
            "Gains {X:mult,C:white} X#2# {} Mult each",
            "time a {C:attention}Pair{} is played",
            "{C:inactive}(Currently {X:mult,C:white} X#1# {C:inactive} Mult)",
        },
    },
    config = { extra = { xmult = 1, xmult_gain = 0.1 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 1, y = 0 },
    blueprint_compat = true,
    perishable_compat = false,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.xmult, card.ability.extra.xmult_gain } }
    end,
    calculate = function(self, card, context)
        if context.before and context.scoring_name == "Pair" and not context.blueprint then
            card.ability.extra.xmult = card.ability.extra.xmult + card.ability.extra.xmult_gain
            return { message = localize("k_upgrade_ex"), colour = G.C.MULT, card = card }
        end
        if context.joker_main and card.ability.extra.xmult > 1 then
            return { xmult = card.ability.extra.xmult }
        end
    end,
}

-- Royal Guard: every face card counts as a Jack, Queen and King for other Jokers.
-- The effect lives in the Card:calculate_joker hook below.
SMODS.Joker {
    key = "royal_guard",
    loc_txt = {
        name = "Royal Guard",
        text = {
            "Every {C:attention}face{} card counts as",
            "a {C:attention}Jack{}, {C:attention}Queen{}, and {C:attention}King{}",
            "for other Jokers",
        },
    },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 2, y = 0 },
    blueprint_compat = false,
}

local FACE_IDS = { 11, 12, 13 }

-- When a Joker checks a face card, try it as its real rank first, then as each
-- other face rank, and use the first rank that makes the Joker trigger.
local calculate_joker_ref = Card.calculate_joker
function Card:calculate_joker(context)
    local other = context and context.other_card
    if not (other and other.ability and other.ability.set ~= "Joker"
            and self.ability.set == "Joker" and next(SMODS.find_card("j_sj_royal_guard"))) then
        return calculate_joker_ref(self, context)
    end
    local id = other:get_id()
    if type(id) ~= "number" or not other:is_face() then
        return calculate_joker_ref(self, context)
    end

    local ret, post = calculate_joker_ref(self, context)
    if ret then return ret, post end

    local own_get_id = rawget(other, "get_id")
    for _, alt in ipairs(FACE_IDS) do
        if alt ~= id then
            other.get_id = function() return alt end
            local ok, alt_ret, alt_post = pcall(calculate_joker_ref, self, context)
            other.get_id = own_get_id
            if not ok then error(alt_ret) end
            if alt_ret then return alt_ret, alt_post end
        end
    end
    return ret, post
end

-- Piggy Bank: swallows half your money each round, may break for a 1.5x payout
SMODS.Joker {
    key = "piggy_bank",
    loc_txt = {
        name = "Piggy Bank",
        text = {
            "Takes {C:attention}half{} your money at",
            "end of round. {C:green}#1# in #2#{} chance",
            "to break and pay out {C:money}X#3#{}",
            "of everything inside",
            "{C:inactive}(Currently holding {C:money}$#4#{C:inactive})",
        },
    },
    config = { extra = { stored = 0, odds = 4, payout = 1.5 } },
    rarity = 1,
    cost = 5,
    atlas = "jokers",
    pos = { x = 3, y = 0 },
    blueprint_compat = false,
    eternal_compat = false,
    loc_vars = function(self, info_queue, card)
        local e = card.ability.extra
        return { vars = { G.GAME.probabilities.normal or 1, e.odds, e.payout, e.stored } }
    end,
    calculate = function(self, card, context)
        if not (context.end_of_round and context.main_eval and not context.blueprint) then return end
        local e = card.ability.extra

        local take = math.floor(math.max(G.GAME.dollars, 0) / 2)
        if take > 0 then
            e.stored = e.stored + take
            ease_dollars(-take)
            card_eval_status_text(card, "extra", nil, nil, nil,
                { message = "-$" .. take, colour = G.C.MONEY })
        end

        if pseudorandom("sj_piggy_bank") < G.GAME.probabilities.normal / e.odds then
            local payout = math.floor(e.stored * e.payout)
            ease_dollars(payout)
            G.E_MANAGER:add_event(Event({
                func = function()
                    play_sound("tarot1")
                    card.T.r = -0.2
                    card:juice_up(0.3, 0.4)
                    card.states.drag.is = true
                    card.children.center.pinch.x = true
                    G.E_MANAGER:add_event(Event({
                        trigger = "after", delay = 0.3, blockable = false,
                        func = function()
                            G.jokers:remove_card(card)
                            card:remove()
                            return true
                        end,
                    }))
                    return true
                end,
            }))
            return { message = "Smashed! +$" .. payout, colour = G.C.MONEY }
        end
    end,
}

-- Last Stand: final hand retriggers every played card twice, then destroys them
SMODS.Joker {
    key = "last_stand",
    loc_txt = {
        name = "Last Stand",
        text = {
            "On the {C:attention}final hand{} of the round,",
            "retrigger all played cards {C:attention}#1#{} times,",
            "then {C:red}destroy{} all played cards",
        },
    },
    config = { extra = { repetitions = 2 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 4, y = 0 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.repetitions } }
    end,
    calculate = function(self, card, context)
        if G.GAME.current_round.hands_left ~= 0 then return end
        if context.repetition and context.cardarea == G.play then
            return { message = localize("k_again_ex"), repetitions = card.ability.extra.repetitions, card = card }
        end
        if context.destroy_card and not context.blueprint
                and (context.cardarea == G.play or context.cardarea == "unscored") then
            return { remove = true }
        end
    end,
}

----------------------------------------------------------------------
-- Second batch
----------------------------------------------------------------------

-- Probability helpers: use Steamodded's probability API when present so Oops!
-- All 6s and similar effects apply, falling back to the classic formula.
local function roll(card, seed, num, den)
    if SMODS.pseudorandom_probability then
        return SMODS.pseudorandom_probability(card, seed, num, den)
    end
    return pseudorandom(seed) < G.GAME.probabilities.normal * num / den
end

local function prob_vars(card, num, den)
    if SMODS.get_probability_vars then
        return SMODS.get_probability_vars(card, num, den)
    end
    return (G.GAME and G.GAME.probabilities.normal or 1) * num, den
end

local function current_ante()
    return math.max(G.GAME and G.GAME.round_resets and G.GAME.round_resets.ante or 1, 0)
end

local function has_rank(c)
    return not (SMODS.has_no_rank and SMODS.has_no_rank(c)) and (c:get_id() or 0) > 0
end

local function joker_name(key)
    local ok, name = pcall(localize, { type = "name_text", set = "Joker", key = key })
    return (ok and name and name ~= "ERROR") and name or key
end

-- Loose Change: lose $1-10 each hand, gain XMult equal to half the loss
SMODS.Joker {
    key = "loose_change",
    loc_txt = {
        name = "Loose Change",
        text = {
            "Each hand, randomly lose {C:money}$#1#-#2#{}",
            "and gain {X:mult,C:white} XMult {} equal to",
            "half the amount lost",
        },
    },
    config = { extra = { min = 1, max = 10 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 0, y = 1 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.min, card.ability.extra.max } }
    end,
    calculate = function(self, card, context)
        if context.joker_main then
            local lost = pseudorandom("sj_loose_change", card.ability.extra.min, card.ability.extra.max)
            return { dollars = -lost, extra = { xmult = lost / 2 } }
        end
    end,
}

-- Rerun: retrigger every played card when the hand contains a Straight
SMODS.Joker {
    key = "rerun",
    loc_txt = {
        name = "Rerun",
        text = {
            "Retrigger all played cards",
            "if played hand contains",
            "a {C:attention}Straight{}",
        },
    },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 1, y = 1 },
    blueprint_compat = true,
    calculate = function(self, card, context)
        if context.repetition and context.cardarea == G.play and next(context.poker_hands["Straight"]) then
            return { message = localize("k_again_ex"), repetitions = 1, card = card }
        end
    end,
}

local function is_light(c) return c:is_suit("Hearts") or c:is_suit("Diamonds") end
local function is_dark(c) return c:is_suit("Spades") or c:is_suit("Clubs") end

-- Bouquet: light and dark suits both pay out when a hand mixes them
SMODS.Joker {
    key = "bouquet",
    loc_txt = {
        name = "Bouquet",
        text = {
            "If played hand contains both",
            "{C:hearts}light{} and {C:spades}dark{} suit cards,",
            "scored {C:hearts}Hearts{} and {C:diamonds}Diamonds{} give {C:mult}+#1#{} Mult",
            "and scored {C:spades}Spades{} and {C:clubs}Clubs{} give {C:chips}+#2#{} Chips",
        },
    },
    config = { extra = { mult = 4, chips = 10 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 2, y = 1 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.mult, card.ability.extra.chips } }
    end,
    calculate = function(self, card, context)
        if context.individual and context.cardarea == G.play then
            local light, dark = false, false
            for _, c in ipairs(context.full_hand) do
                light = light or is_light(c)
                dark = dark or is_dark(c)
            end
            if not (light and dark) then return end
            local c = context.other_card
            local ret = {}
            if is_light(c) then ret.mult = card.ability.extra.mult end
            if is_dark(c) then ret.chips = card.ability.extra.chips end
            if next(ret) then return ret end
        end
    end,
}

local function is_odd(c)
    if not has_rank(c) then return false end
    local id = c:get_id()
    return id == 14 or (id <= 10 and id % 2 == 1)
end

-- Odd Jobs: $5 when every scoring card is odd (High Card needs 5 played cards)
SMODS.Joker {
    key = "odd_jobs",
    loc_txt = {
        name = "Odd Jobs",
        text = {
            "Earn {C:money}$#1#{} if every scoring",
            "card is {C:attention}odd{}",
            "{C:inactive}(High Card needs 5 played cards)",
        },
    },
    config = { extra = { dollars = 5 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 3, y = 1 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.dollars } }
    end,
    calculate = function(self, card, context)
        if context.joker_main then
            if context.scoring_name == "High Card" and #context.full_hand < 5 then return end
            for _, c in ipairs(context.scoring_hand) do
                if not is_odd(c) then return end
            end
            return { dollars = card.ability.extra.dollars }
        end
    end,
}

-- Late Bloomer: +5 Mult per Ante, applied before any card scores
SMODS.Joker {
    key = "late_bloomer",
    loc_txt = {
        name = "Late Bloomer",
        text = {
            "{C:mult}+#1#{} Mult per {C:attention}Ante{},",
            "added {C:attention}before{} any cards score",
            "{C:inactive}(Currently {C:mult}+#2#{C:inactive} Mult)",
        },
    },
    config = { extra = { mult = 5 } },
    rarity = 1,
    cost = 5,
    atlas = "jokers",
    pos = { x = 4, y = 1 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.mult, card.ability.extra.mult * current_ante() } }
    end,
    calculate = function(self, card, context)
        if context.initial_scoring_step then
            local mult = card.ability.extra.mult * current_ante()
            if mult > 0 then return { mult = mult } end
        end
    end,
}

-- Hot Potato: X3 Mult, but may burn a random Joker when a Blind is selected
SMODS.Joker {
    key = "hot_potato",
    loc_txt = {
        name = "Hot Potato",
        text = {
            "{X:mult,C:white} X#1# {} Mult",
            "{C:green}#2# in #3#{} chance to {C:red}destroy{} a",
            "random Joker when {C:attention}Blind{} is selected",
            "{C:inactive}(Can destroy itself)",
        },
    },
    config = { extra = { xmult = 3, odds = 4 } },
    rarity = 2,
    cost = 7,
    atlas = "jokers",
    pos = { x = 0, y = 2 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local num, den = prob_vars(card, 1, card.ability.extra.odds)
        return { vars = { card.ability.extra.xmult, num, den } }
    end,
    calculate = function(self, card, context)
        if context.setting_blind and not context.blueprint and roll(card, "sj_hot_potato", 1, card.ability.extra.odds) then
            local targets = {}
            for _, j in ipairs(G.jokers.cards) do
                if not j.getting_sliced and not (j.ability and j.ability.eternal) then
                    targets[#targets + 1] = j
                end
            end
            local victim = pseudorandom_element(targets, pseudoseed("sj_hot_potato_target"))
            if victim then
                victim.getting_sliced = true
                G.E_MANAGER:add_event(Event({
                    func = function()
                        victim:start_dissolve({ G.C.RED }, nil, 1.6)
                        return true
                    end,
                }))
                return { message = "Burned!", colour = G.C.RED }
            end
        end
        if context.joker_main then
            return { xmult = card.ability.extra.xmult }
        end
    end,
}

-- Consolation Prize: $10 for leaving the shop without buying anything
SMODS.Joker {
    key = "consolation_prize",
    loc_txt = {
        name = "Consolation Prize",
        text = {
            "Earn {C:money}$#1#{} when leaving the",
            "shop without buying anything",
        },
    },
    config = { extra = { dollars = 10, bought = false } },
    rarity = 2,
    cost = 5,
    atlas = "jokers",
    pos = { x = 1, y = 2 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.dollars } }
    end,
    calculate = function(self, card, context)
        if (context.buying_card or context.open_booster) and not context.blueprint then
            card.ability.extra.bought = true
        end
        if context.ending_shop then
            local bought = card.ability.extra.bought
            if not context.blueprint then card.ability.extra.bought = false end
            if not bought then
                return { dollars = card.ability.extra.dollars }
            end
        end
    end,
}

-- Copycat: retrigger the first and last played cards if they share a rank
SMODS.Joker {
    key = "copycat",
    loc_txt = {
        name = "Copycat",
        text = {
            "If the {C:attention}first{} and {C:attention}last{} played",
            "cards share a rank,",
            "retrigger them both",
        },
    },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 2, y = 2 },
    blueprint_compat = true,
    calculate = function(self, card, context)
        if context.repetition and context.cardarea == G.play then
            local hand = context.full_hand
            local first, last = hand[1], hand[#hand]
            if #hand >= 2 and has_rank(first) and has_rank(last) and first:get_id() == last:get_id()
                    and (context.other_card == first or context.other_card == last) then
                return { message = localize("k_again_ex"), repetitions = 1, card = card }
            end
        end
    end,
}

-- Rust Bucket: Steel cards stop giving XMult and pay $4 each at end of round
SMODS.Joker {
    key = "rust_bucket",
    loc_txt = {
        name = "Rust Bucket",
        text = {
            "{C:attention}Steel{} cards lose their {X:mult,C:white} XMult {},",
            "but give {C:money}$#1#{} each if held",
            "in hand at end of round",
        },
    },
    config = { extra = { dollars = 4 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 3, y = 2 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        info_queue[#info_queue + 1] = G.P_CENTERS.m_steel
        return { vars = { card.ability.extra.dollars } }
    end,
    calculate = function(self, card, context)
        if context.individual and context.end_of_round and context.cardarea == G.hand
                and SMODS.has_enhancement(context.other_card, "m_steel") and not context.other_card.debuff then
            return { dollars = card.ability.extra.dollars }
        end
    end,
}

local get_chip_h_x_mult_ref = Card.get_chip_h_x_mult
function Card:get_chip_h_x_mult()
    if self.config.center.key == "m_steel" and G.jokers and next(SMODS.find_card("j_sj_rust_bucket")) then
        return 0
    end
    return get_chip_h_x_mult_ref(self)
end

-- Echo Chamber: once per round, copy the first card of a hand that is all one rank
SMODS.Joker {
    key = "echo_chamber",
    loc_txt = {
        name = "Echo Chamber",
        text = {
            "If every played card has the",
            "{C:attention}same rank{}, add a copy of the",
            "first card to your hand",
            "{C:inactive}(Once per round)",
        },
    },
    config = { extra = { used = false } },
    rarity = 2,
    cost = 7,
    atlas = "jokers",
    pos = { x = 4, y = 2 },
    blueprint_compat = true,
    calculate = function(self, card, context)
        if context.end_of_round and context.main_eval and not context.blueprint then
            card.ability.extra.used = false
            return
        end
        if context.before and not card.ability.extra.used then
            local hand = context.full_hand
            if not has_rank(hand[1]) then return end
            for _, c in ipairs(hand) do
                if not has_rank(c) or c:get_id() ~= hand[1]:get_id() then return end
            end
            if not context.blueprint then card.ability.extra.used = true end
            G.playing_card = (G.playing_card and G.playing_card + 1) or 1
            local copy = copy_card(hand[1], nil, nil, G.playing_card)
            copy:add_to_deck()
            G.deck.config.card_limit = G.deck.config.card_limit + 1
            table.insert(G.playing_cards, copy)
            G.hand:emplace(copy)
            copy.states.visible = nil
            G.E_MANAGER:add_event(Event({
                func = function()
                    copy:start_materialize()
                    return true
                end,
            }))
            if playing_card_joker_effects then playing_card_joker_effects({ copy }) end
            return { message = localize("k_copied_ex"), colour = G.C.CHIPS, card = card }
        end
    end,
}

-- Card Counter: XMult grows as the deck runs low
SMODS.Joker {
    key = "card_counter",
    loc_txt = {
        name = "Card Counter",
        text = {
            "{X:mult,C:white} X#1# {} Mult for every card",
            "remaining in deck below {C:attention}#2#{}",
            "{C:inactive}(Currently {X:mult,C:white} X#3# {C:inactive} Mult)",
        },
    },
    config = { extra = { xmult_per = 0.1, threshold = 40 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 0, y = 3 },
    blueprint_compat = true,
    current_xmult = function(self, card)
        local e = card.ability.extra
        local left = G.deck and #G.deck.cards or e.threshold
        return 1 + e.xmult_per * math.max(e.threshold - left, 0)
    end,
    loc_vars = function(self, info_queue, card)
        local e = card.ability.extra
        return { vars = { e.xmult_per, e.threshold, self:current_xmult(card) } }
    end,
    calculate = function(self, card, context)
        if context.joker_main then
            local xmult = self:current_xmult(card)
            if xmult > 1 then return { xmult = xmult } end
        end
    end,
}

-- Gambler's Fallacy: coin flip between X4 Mult and halving Chips and Mult
SMODS.Joker {
    key = "gamblers_fallacy",
    loc_txt = {
        name = "Gambler's Fallacy",
        text = {
            "{C:green}#1# in #2#{} chance for {X:mult,C:white} X#3# {} Mult,",
            "otherwise {C:attention}halve{} Chips and Mult",
        },
    },
    config = { extra = { xmult = 4, odds = 2 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 1, y = 3 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local num, den = prob_vars(card, 1, card.ability.extra.odds)
        return { vars = { num, den, card.ability.extra.xmult } }
    end,
    calculate = function(self, card, context)
        if context.joker_main then
            -- Only the X4 outcome is a probability roll, so Oops! only helps the good side
            if roll(card, "sj_gamblers_fallacy", 1, card.ability.extra.odds) then
                return { xmult = card.ability.extra.xmult }
            end
            return { xchips = 0.5, xmult = 0.5, message = "Busted!", colour = G.C.RED }
        end
    end,
}

-- Monarchy: all-face-card hands level up their hand type
SMODS.Joker {
    key = "monarchy",
    loc_txt = {
        name = "Monarchy",
        text = {
            "If every scoring card is a",
            "{C:attention}face{} card, upgrade the",
            "played {C:attention}poker hand{}",
        },
    },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 2, y = 3 },
    blueprint_compat = true,
    calculate = function(self, card, context)
        if context.before then
            for _, c in ipairs(context.scoring_hand) do
                if not c:is_face() then return end
            end
            return { level_up = true, message = localize("k_level_up_ex") }
        end
    end,
}

-- The Collector: XMult for every different Joker owned this run.
-- Ownership is recorded in G.GAME so it is saved with the run.
local add_to_deck_ref = Card.add_to_deck
function Card:add_to_deck(from_debuff)
    if G.GAME and self.ability and self.ability.set == "Joker" then
        G.GAME.sj_collected = G.GAME.sj_collected or {}
        G.GAME.sj_collected[self.config.center.key] = true
    end
    return add_to_deck_ref(self, from_debuff)
end

local function collected_count()
    local n = 0
    for _ in pairs(G.GAME and G.GAME.sj_collected or {}) do n = n + 1 end
    return n
end

SMODS.Joker {
    key = "collector",
    loc_txt = {
        name = "The Collector",
        text = {
            "{X:mult,C:white} X#1# {} Mult for every different",
            "Joker you have owned this run",
            "{C:inactive}(Currently {X:mult,C:white} X#2# {C:inactive} Mult)",
        },
    },
    config = { extra = { xmult_per = 0.25 } },
    rarity = 4,
    cost = 20,
    atlas = "jokers",
    pos = { x = 3, y = 3 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local per = card.ability.extra.xmult_per
        return { vars = { per, 1 + per * collected_count() } }
    end,
    calculate = function(self, card, context)
        if context.joker_main then
            return { xmult = 1 + card.ability.extra.xmult_per * collected_count() }
        end
    end,
}

-- Ouroboros: eats the Joker to its right after each Boss Blind and keeps its effect.
-- Absorbed Jokers are stored as {key, ability}. To run one, Ouroboros briefly takes
-- on that Joker's center and ability, calculates, then switches back.
-- While an absorbed effect runs (and inside any events it queues), Jokers cannot be
-- destroyed. This keeps copied self-destructs (Gros Michel, Ice Cream...) and
-- Joker-destroying effects (Madness, Ceremonial Dagger...) from doing anything.
local protect_jokers = 0

local function protected(fn)
    return function(...)
        protect_jokers = protect_jokers + 1
        local ok, a, b = pcall(fn, ...)
        protect_jokers = protect_jokers - 1
        if not ok then error(a, 0) end
        return a, b
    end
end

local add_event_ref = EventManager.add_event
function EventManager:add_event(event, ...)
    if protect_jokers > 0 and type(event) == "table" and type(event.func) == "function" then
        event.func = protected(event.func)
    end
    return add_event_ref(self, event, ...)
end

local function is_protected(c)
    return protect_jokers > 0 and c and c.ability and c.ability.set == "Joker"
end

for _, name in ipairs({ "start_dissolve", "shatter", "remove" }) do
    local ref = Card[name]
    Card[name] = function(self, ...)
        if is_protected(self) then
            self.getting_sliced = nil
            return
        end
        return ref(self, ...)
    end
end

local remove_card_ref = CardArea.remove_card
function CardArea:remove_card(card, ...)
    if is_protected(card) then return end
    return remove_card_ref(self, card, ...)
end

local function as_absorbed(card, entry, fn)
    local center = G.P_CENTERS[entry.key]
    if not center then return end
    local sliced = {}
    for _, j in ipairs(G.jokers and G.jokers.cards or {}) do sliced[j] = j.getting_sliced end
    local own_ability, own_center = card.ability, card.config.center
    card.ability, card.config.center = entry.ability, center
    local ok, a, b = pcall(protected(fn))
    card.ability, card.config.center = own_ability, own_center
    -- Undo any "about to be destroyed" marks the copied effect placed on Jokers
    for _, j in ipairs(G.jokers and G.jokers.cards or {}) do
        if j.getting_sliced and not sliced[j] then j.getting_sliced = nil end
    end
    if not ok then error(a, 0) end
    return a, b
end

SMODS.Joker {
    key = "ouroboros",
    loc_txt = {
        name = "Ouroboros",
        text = {
            "After each {C:attention}Boss Blind{}, {C:red}destroy{} the",
            "Joker to the right and permanently",
            "gain its effect {C:inactive}(except editions",
            "{C:inactive}and destroying Jokers)",
            "{C:inactive}(Absorbed: #1#)",
        },
    },
    config = { extra = { absorbed = {} } },
    rarity = 4,
    cost = 20,
    atlas = "jokers",
    pos = { x = 4, y = 3 },
    blueprint_compat = false,
    loc_vars = function(self, info_queue, card)
        local names = {}
        for _, entry in ipairs(card.ability.extra.absorbed) do
            names[#names + 1] = joker_name(entry.key)
        end
        return { vars = { #names > 0 and table.concat(names, ", ") or "none" } }
    end,
    calculate = function(self, card, context)
        local absorbed = card.ability.extra.absorbed

        -- Run every absorbed effect and combine the results
        local effects, repetitions = {}, 0
        for _, entry in ipairs(absorbed) do
            local ret = as_absorbed(card, entry, function() return card:calculate_joker(context) end)
            if type(ret) == "table" then
                if ret.repetitions then
                    repetitions = repetitions + ret.repetitions
                else
                    effects[#effects + 1] = ret
                end
            end
        end

        -- An Ouroboros that was itself absorbed doesn't eat anything
        if context.end_of_round and context.main_eval and context.beat_boss and not context.blueprint
                and protect_jokers == 0 then
            local index
            for i, j in ipairs(G.jokers.cards) do
                if j == card then index = i end
            end
            local victim = index and G.jokers.cards[index + 1]
            if victim and not victim.getting_sliced and not (victim.ability and victim.ability.eternal) then
                -- Strip the edition first so a Negative's extra slot isn't kept
                if victim.edition then victim:set_edition(nil, true, true) end
                absorbed[#absorbed + 1] = { key = victim.config.center.key, ability = copy_table(victim.ability) }
                -- Skip remove_from_deck so passive effects (hand size, slots...) stay forever
                victim.added_to_deck = false
                victim.getting_sliced = true
                G.E_MANAGER:add_event(Event({
                    func = function()
                        victim:start_dissolve({ G.C.PURPLE }, nil, 1.6)
                        return true
                    end,
                }))
                effects[#effects + 1] = { message = "Devoured!", colour = G.C.PURPLE }
            end
        end

        if repetitions > 0 then
            return { message = localize("k_again_ex"), repetitions = repetitions, card = card }
        end
        if #effects > 0 then
            return SMODS.merge_effects(effects)
        end
    end,
    calc_dollar_bonus = function(self, card)
        local total = 0
        for _, entry in ipairs(card.ability.extra.absorbed) do
            total = total + (as_absorbed(card, entry, function() return card:calculate_dollar_bonus() end) or 0)
        end
        if total > 0 then return total end
    end,
}

----------------------------------------------------------------------
-- Third batch
----------------------------------------------------------------------

-- Lets Jester King retrigger other Jokers
SMODS.current_mod.optional_features = { retrigger_joker = true }

local function enhancement(c)
    return c.config and c.config.center and c.config.center.key
end

-- Copies a playing card into the hand (same approach as DNA)
local function copy_to_hand(source)
    G.playing_card = (G.playing_card and G.playing_card + 1) or 1
    local copy = copy_card(source, nil, nil, G.playing_card)
    copy:add_to_deck()
    G.deck.config.card_limit = G.deck.config.card_limit + 1
    table.insert(G.playing_cards, copy)
    G.hand:emplace(copy)
    copy.states.visible = nil
    G.E_MANAGER:add_event(Event({
        func = function()
            copy:start_materialize()
            return true
        end,
    }))
    if playing_card_joker_effects then playing_card_joker_effects({ copy }) end
    return copy
end

local SPARE_PARTS = { m_steel = true, m_gold = true, m_glass = true }

local function has_any_enhancement(c, keys)
    for key in pairs(keys) do
        if SMODS.has_enhancement(c, key) then return true end
    end
    return false
end

-- Spare Parts: discarding Steel, Gold or Glass cards adds permanent Chips
SMODS.Joker {
    key = "spare_parts",
    loc_txt = {
        name = "Spare Parts",
        text = {
            "Gains {C:chips}+#2#{} Chips when a {C:attention}Steel{},",
            "{C:attention}Gold{} or {C:attention}Glass{} card is discarded",
            "{C:inactive}(Currently {C:chips}+#1#{C:inactive} Chips)",
        },
    },
    config = { extra = { chips = 0, chip_gain = 3 } },
    rarity = 1,
    cost = 5,
    atlas = "jokers",
    pos = { x = 0, y = 4 },
    blueprint_compat = true,
    perishable_compat = false,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.chips, card.ability.extra.chip_gain } }
    end,
    calculate = function(self, card, context)
        if context.discard and not context.blueprint and has_any_enhancement(context.other_card, SPARE_PARTS)
                and not context.other_card.debuff then
            card.ability.extra.chips = card.ability.extra.chips + card.ability.extra.chip_gain
            return { message = localize("k_upgrade_ex"), colour = G.C.CHIPS, card = card }
        end
        if context.joker_main and card.ability.extra.chips > 0 then
            return { chips = card.ability.extra.chips }
        end
    end,
}

-- Night Owl: +15 Mult for hands made only of Spades and Clubs
SMODS.Joker {
    key = "night_owl",
    loc_txt = {
        name = "Night Owl",
        text = {
            "{C:mult}+#1#{} Mult if played hand",
            "contains only {C:spades}Spades{}",
            "and {C:clubs}Clubs{}",
        },
    },
    config = { extra = { mult = 15 } },
    rarity = 1,
    cost = 5,
    atlas = "jokers",
    pos = { x = 1, y = 4 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.mult } }
    end,
    calculate = function(self, card, context)
        if context.joker_main then
            for _, c in ipairs(context.full_hand) do
                if not is_dark(c) then return end
            end
            return { mult = card.ability.extra.mult }
        end
    end,
}

-- Early Bird: +60 Chips on the first hand of the round if it uses 5 cards
SMODS.Joker {
    key = "early_bird",
    loc_txt = {
        name = "Early Bird",
        text = {
            "{C:chips}+#1#{} Chips if the {C:attention}first hand{}",
            "of the round has {C:attention}5{} cards",
        },
    },
    config = { extra = { chips = 60 } },
    rarity = 1,
    cost = 5,
    atlas = "jokers",
    pos = { x = 2, y = 4 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.chips } }
    end,
    calculate = function(self, card, context)
        if context.joker_main and G.GAME.current_round.hands_played == 0 and #context.full_hand == 5 then
            return { chips = card.ability.extra.chips }
        end
    end,
}

-- Middle Child: retrigger the middle card twice, or the two middle cards once
SMODS.Joker {
    key = "middle_child",
    loc_txt = {
        name = "Middle Child",
        text = {
            "Retrigger the {C:attention}middle{} card of a",
            "{C:attention}3{} or {C:attention}5{} card hand {C:attention}2{} times, or the",
            "{C:attention}2{} middle cards of a {C:attention}4{} card hand {C:attention}1{} time",
        },
    },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 3, y = 4 },
    blueprint_compat = true,
    calculate = function(self, card, context)
        if context.repetition and context.cardarea == G.play then
            local hand, other = context.full_hand, context.other_card
            local n = #hand
            local reps = 0
            if (n == 3 or n == 5) and other == hand[(n + 1) / 2] then
                reps = 2
            elseif n == 4 and (other == hand[2] or other == hand[3]) then
                reps = 1
            end
            if reps > 0 then
                return { message = localize("k_again_ex"), repetitions = reps, card = card }
            end
        end
    end,
}

-- Mirror Match: repeated hand types build a growing XMult streak
SMODS.Joker {
    key = "mirror_match",
    loc_txt = {
        name = "Mirror Match",
        text = {
            "{X:mult,C:white} X#1# {} Mult if played hand is the",
            "same type as the last hand played,",
            "plus {X:mult,C:white} X#2# {} for each repeat in a row",
            "{C:inactive}(Last hand: #3#, streak #4#)",
        },
    },
    config = { extra = { base = 1.5, gain = 0.25, last = nil, streak = 0 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 4, y = 4 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local e = card.ability.extra
        local last = e.last and localize(e.last, "poker_hands") or "none"
        return { vars = { e.base, e.gain, last, e.streak } }
    end,
    calculate = function(self, card, context)
        local e = card.ability.extra
        if context.before and not context.blueprint then
            e.streak = (context.scoring_name == e.last) and e.streak + 1 or 0
            e.last = context.scoring_name
        end
        if context.joker_main and e.streak > 0 then
            return { xmult = e.base + e.gain * (e.streak - 1) }
        end
    end,
}

-- Undertaker: permanent Mult for every playing card destroyed
SMODS.Joker {
    key = "undertaker",
    loc_txt = {
        name = "Undertaker",
        text = {
            "Gains {C:mult}+#2#{} Mult whenever a",
            "playing card is {C:red}destroyed{}",
            "{C:inactive}(Currently {C:mult}+#1#{C:inactive} Mult)",
        },
    },
    config = { extra = { mult = 0, mult_gain = 4 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 0, y = 5 },
    blueprint_compat = true,
    perishable_compat = false,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.mult, card.ability.extra.mult_gain } }
    end,
    calculate = function(self, card, context)
        if context.remove_playing_cards and not context.blueprint and context.removed and #context.removed > 0 then
            card.ability.extra.mult = card.ability.extra.mult + card.ability.extra.mult_gain * #context.removed
            return { message = localize("k_upgrade_ex"), colour = G.C.MULT, card = card }
        end
        if context.joker_main and card.ability.extra.mult > 0 then
            return { mult = card.ability.extra.mult }
        end
    end,
}

-- Wild West: Wild cards give XMult and may duplicate themselves into hand
SMODS.Joker {
    key = "wild_west",
    loc_txt = {
        name = "Wild West",
        text = {
            "{C:attention}Wild{} cards give {X:mult,C:white} X#1# {} Mult",
            "when scored. Each played Wild card",
            "has a {C:green}#2# in #3#{} chance to copy",
            "itself into your hand",
        },
    },
    config = { extra = { xmult = 1.5, odds = 4 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 1, y = 5 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        info_queue[#info_queue + 1] = G.P_CENTERS.m_wild
        local num, den = prob_vars(card, 1, card.ability.extra.odds)
        return { vars = { card.ability.extra.xmult, num, den } }
    end,
    calculate = function(self, card, context)
        if context.before then
            local copied = false
            for _, c in ipairs(context.full_hand) do
                if SMODS.has_enhancement(c, "m_wild") and not c.debuff
                        and roll(card, "sj_wild_west", 1, card.ability.extra.odds) then
                    copy_to_hand(c)
                    copied = true
                end
            end
            if copied then
                return { message = localize("k_copied_ex"), colour = G.C.CHIPS, card = card }
            end
        end
        if context.individual and context.cardarea == G.play and SMODS.has_enhancement(context.other_card, "m_wild") then
            return { xmult = card.ability.extra.xmult }
        end
    end,
}

-- Overtime: winning on the last hand permanently adds XMult
SMODS.Joker {
    key = "overtime",
    loc_txt = {
        name = "Overtime",
        text = {
            "Gains {X:mult,C:white} X#2# {} Mult when a Blind",
            "is beaten on the {C:attention}final hand{}",
            "{C:inactive}(Currently {X:mult,C:white} X#1# {C:inactive} Mult)",
        },
    },
    config = { extra = { xmult = 1, xmult_gain = 0.2 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 2, y = 5 },
    blueprint_compat = true,
    perishable_compat = false,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.xmult, card.ability.extra.xmult_gain } }
    end,
    calculate = function(self, card, context)
        if context.end_of_round and context.main_eval and not context.game_over and not context.blueprint
                and G.GAME.current_round.hands_left == 0 then
            card.ability.extra.xmult = card.ability.extra.xmult + card.ability.extra.xmult_gain
            return { message = localize("k_upgrade_ex"), colour = G.C.MULT, card = card }
        end
        if context.joker_main and card.ability.extra.xmult > 1 then
            return { xmult = card.ability.extra.xmult }
        end
    end,
}

-- Deep Pockets: +1 consumable slot, plus 1 more for every $100 held.
-- extra.applied tracks how many slots this card has currently added.
local function deep_pockets_target(card)
    return 1 + math.floor(math.max(G.GAME.dollars or 0, 0) / card.ability.extra.per)
end

SMODS.Joker {
    key = "deep_pockets",
    loc_txt = {
        name = "Deep Pockets",
        text = {
            "{C:attention}+1{} consumable slot, plus",
            "{C:attention}1{} more for every {C:money}$#1#{} you have",
            "{C:inactive}(Currently {C:attention}+#2#{C:inactive} slots)",
        },
    },
    config = { extra = { per = 100, applied = 0 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 3, y = 5 },
    blueprint_compat = false,
    loc_vars = function(self, info_queue, card)
        local slots = G.GAME and G.GAME.dollars and deep_pockets_target(card) or 1
        return { vars = { card.ability.extra.per, slots } }
    end,
    add_to_deck = function(self, card, from_debuff)
        local target = deep_pockets_target(card)
        G.consumeables:change_size(target)
        card.ability.extra.applied = target
    end,
    remove_from_deck = function(self, card, from_debuff)
        G.consumeables:change_size(-card.ability.extra.applied)
        card.ability.extra.applied = 0
    end,
    update = function(self, card, dt)
        if not (card.added_to_deck and G.consumeables and G.GAME) or card.debuff then return end
        local diff = deep_pockets_target(card) - card.ability.extra.applied
        if diff ~= 0 then
            G.consumeables:change_size(diff)
            card.ability.extra.applied = card.ability.extra.applied + diff
        end
    end,
}

-- Glass Cannon: Glass cards give X3 Mult instead of X2, but always shatter
SMODS.Joker {
    key = "glass_cannon",
    loc_txt = {
        name = "Glass Cannon",
        text = {
            "{C:attention}Glass{} cards give {X:mult,C:white} X1 {} more",
            "Mult, but always {C:red}shatter{}",
            "when scored",
        },
    },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 4, y = 5 },
    blueprint_compat = false,
    loc_vars = function(self, info_queue, card)
        info_queue[#info_queue + 1] = G.P_CENTERS.m_glass
    end,
    calculate = function(self, card, context)
        if context.destroy_card and context.cardarea == G.play and not context.blueprint
                and SMODS.has_enhancement(context.destroy_card, "m_glass") then
            return { remove = true }
        end
    end,
}

local get_chip_x_mult_ref = Card.get_chip_x_mult
function Card:get_chip_x_mult(...)
    local x = get_chip_x_mult_ref(self, ...)
    if self.config.center.key == "m_glass" and G.jokers and next(SMODS.find_card("j_sj_glass_cannon")) and x > 0 then
        return x + 1
    end
    return x
end

-- Clockwork: every 3rd hand, retrigger all played cards twice
SMODS.Joker {
    key = "clockwork",
    loc_txt = {
        name = "Clockwork",
        text = {
            "Every {C:attention}#1#rd{} hand played,",
            "retrigger all played cards {C:attention}#2#{} times",
            "{C:inactive}(#3# remaining)",
        },
    },
    config = { extra = { every = 3, repetitions = 2, count = 0 } },
    rarity = 3,
    cost = 8,
    atlas = "jokers",
    pos = { x = 0, y = 6 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local e = card.ability.extra
        return { vars = { e.every, e.repetitions, e.every - (e.count % e.every) } }
    end,
    calculate = function(self, card, context)
        local e = card.ability.extra
        if context.before and not context.blueprint then
            e.count = e.count + 1
        end
        if context.repetition and context.cardarea == G.play and e.count % e.every == 0 then
            return { message = localize("k_again_ex"), repetitions = e.repetitions, card = card }
        end
    end,
}

-- Jester King: each other Joker has a 1 in 4 chance to trigger twice
-- Jester Kings never retrigger each other (including an Ouroboros that absorbed one)
local function is_jester_king(j)
    if not (j and j.config and j.config.center) then return false end
    if j.config.center.key == "j_sj_jester_king" then return true end
    for _, entry in ipairs(j.config.center.key == "j_sj_ouroboros" and j.ability.extra.absorbed or {}) do
        if entry.key == "j_sj_jester_king" then return true end
    end
    return false
end

SMODS.Joker {
    key = "jester_king",
    loc_txt = {
        name = "Jester King",
        text = {
            "Every other Joker has a",
            "{C:green}#1# in #2#{} chance to",
            "{C:attention}trigger twice{}",
            "{C:inactive}(Except other Jester Kings)",
        },
    },
    config = { extra = { odds = 4 } },
    rarity = 4,
    cost = 20,
    atlas = "jokers",
    pos = { x = 1, y = 6 },
    blueprint_compat = false,
    loc_vars = function(self, info_queue, card)
        local num, den = prob_vars(card, 1, card.ability.extra.odds)
        return { vars = { num, den } }
    end,
    calculate = function(self, card, context)
        if context.retrigger_joker_check and not context.retrigger_joker and not is_jester_king(context.other_card)
                and roll(card, "sj_jester_king", 1, card.ability.extra.odds) then
            return { message = localize("k_again_ex"), repetitions = 1, card = card }
        end
    end,
}

-- Disorder: copies a random Joker, re-rolled at end of round or when a Joker is sold.
-- The target is stored by sort_id so it survives the Jokers being reordered.
local function disorder_pick(card, exclude)
    local options = {}
    for _, j in ipairs(G.jokers and G.jokers.cards or {}) do
        if j ~= card and j ~= exclude and j.config.center.blueprint_compat
                and j.config.center.key ~= "j_sj_disorder" then
            options[#options + 1] = j
        end
    end
    local target = pseudorandom_element(options, pseudoseed("sj_disorder"))
    card.ability.extra.target = target and target.sort_id or nil
end

local function disorder_target(card)
    for _, j in ipairs(G.jokers and G.jokers.cards or {}) do
        if j.sort_id == card.ability.extra.target and j ~= card then return j end
    end
end

SMODS.Joker {
    key = "disorder",
    loc_txt = {
        name = "Disorder",
        text = {
            "Copies the ability of a",
            "{C:attention}random{} Joker you own",
            "{C:inactive}(Changes at end of round or",
            "{C:inactive}when a Joker is sold)",
            "{C:inactive}(Copying: #1#)",
        },
    },
    config = { extra = { target = nil } },
    rarity = 2,
    cost = 7,
    atlas = "jokers",
    pos = { x = 2, y = 6 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local target = disorder_target(card)
        return { vars = { target and joker_name(target.config.center.key) or "nothing" } }
    end,
    add_to_deck = function(self, card, from_debuff)
        disorder_pick(card)
    end,
    calculate = function(self, card, context)
        if not context.blueprint then
            if context.selling_card and context.card and context.card.ability.set == "Joker" then
                disorder_pick(card, context.card)
                return
            end
            if not disorder_target(card) then disorder_pick(card) end
        end

        local ret = SMODS.blueprint_effect(card, disorder_target(card), context)

        if context.end_of_round and context.main_eval and not context.blueprint then
            disorder_pick(card)
            local swap = { message = "Shuffled!", colour = G.C.PURPLE }
            return ret and SMODS.merge_effects({ ret, swap }) or swap
        end
        return ret
    end,
}

-- Hungry Hippo: discarded cards may pay $1 or get eaten
SMODS.Joker {
    key = "hungry_hippo",
    loc_txt = {
        name = "Hungry Hippo",
        text = {
            "Each discarded card has a",
            "{C:green}#1# in #2#{} chance to give {C:money}$#3#{}",
            "and a {C:green}#1# in #4#{} chance to be {C:red}destroyed{}",
        },
    },
    config = { extra = { money_odds = 5, eat_odds = 15, dollars = 1 } },
    rarity = 2,
    cost = 6,
    atlas = "jokers",
    pos = { x = 3, y = 6 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        local e = card.ability.extra
        local num, money_den = prob_vars(card, 1, e.money_odds)
        local _, eat_den = prob_vars(card, 1, e.eat_odds)
        return { vars = { num, money_den, e.dollars, eat_den } }
    end,
    calculate = function(self, card, context)
        if context.discard then
            local e = card.ability.extra
            local ret = {}
            if roll(card, "sj_hippo_money", 1, e.money_odds) then ret.dollars = e.dollars end
            if roll(card, "sj_hippo_eat", 1, e.eat_odds) then ret.remove = true end
            if next(ret) then return ret end
        end
    end,
}

-- Liquid Joker: earn $1 every time any playing card triggers, anywhere, any time
-- (scoring, held in hand, end of round...). A card counts as triggered when
-- Steamodded actually applies effects for it. Each trigger fires a custom
-- context so Blueprint and Brainstorm can copy the payout.
local in_liquid_context = false

local trigger_effects_ref = SMODS.trigger_effects
SMODS.trigger_effects = function(effects, card)
    local flags = trigger_effects_ref(effects, card)
    if not in_liquid_context and flags and flags.calculated and card and card.playing_card
            and G.jokers and next(SMODS.find_card("j_sj_liquid_joker")) then
        in_liquid_context = true
        local ok, err = pcall(SMODS.calculate_context, { sj_card_triggered = true, other_card = card })
        in_liquid_context = false
        if not ok then error(err, 0) end
    end
    return flags
end

SMODS.Joker {
    key = "liquid_joker",
    loc_txt = {
        name = "Liquid Joker",
        text = {
            "Earn {C:money}$#1#{} every time",
            "any card is {C:attention}triggered{}",
        },
    },
    config = { extra = { dollars = 1 } },
    rarity = 4,
    cost = 20,
    atlas = "jokers",
    pos = { x = 4, y = 6 },
    blueprint_compat = true,
    loc_vars = function(self, info_queue, card)
        return { vars = { card.ability.extra.dollars } }
    end,
    calculate = function(self, card, context)
        if context.sj_card_triggered then
            return { dollars = card.ability.extra.dollars }
        end
    end,
}

-- Shared helpers for joker_display.lua
SimplyJokers = {
    prob_vars = prob_vars,
    current_ante = current_ante,
    has_rank = has_rank,
    is_light = is_light,
    is_dark = is_dark,
    is_odd = is_odd,
    enhancement = enhancement,
    collected_count = collected_count,
    disorder_target = disorder_target,
    -- Blocks Joker destruction while active (used by fused Jokers)
    protect_mode = {
        enter = function()
            protect_jokers = protect_jokers + 1
            return function() protect_jokers = protect_jokers - 1 end
        end,
    },
}

-- The Fusion consumable type (Fuse) and fused cards
assert(SMODS.load_file("fusion.lua"))()

-- Nailed: Tarot card that stops a card being debuffed or flipped
assert(SMODS.load_file("nailed.lua"))()

-- Settings slider to make this mod's items more common
assert(SMODS.load_file("frequency.lua"))()

-- JokerDisplay support (only used when the JokerDisplay mod is installed)
assert(SMODS.load_file("joker_display.lua"))()
