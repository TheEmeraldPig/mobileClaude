--- Random Joker triggers. Every Joker that gives Chips, Mult, XMult or money while
--- scoring rolls a random trigger the first time it shows up in a run, and its card text
--- is rewritten to say so. Its scoring numbers (and retriggers) then only come on that
--- trigger. Everything else it does (creating cards, growing, Blind effects...) is left
--- exactly as it was.
---
--- To find a Joker's numbers it is run once in the scoring step with events and money
--- switched off and its state copied back afterwards. Nothing else is ever test-run.

local R = Randomizer
local MOD = R.mod

----------------------------------------------------------------------
-- Triggers
----------------------------------------------------------------------

local function id(c) return c and c:get_id() or 0 end
local function has_hand(ctx, name) return ctx.poker_hands and ctx.poker_hands[name] and next(ctx.poker_hands[name]) end

local function any_face(cards)
    for _, c in ipairs(cards or {}) do
        if c:is_face() then return true end
    end
    return false
end

-- Lowest ranked card held in hand (Raised Fist)
local function lowest_held(card)
    local low
    for _, c in ipairs(G.hand and G.hand.cards or {}) do
        if not low or id(c) < id(low) then low = c end
    end
    return low == card
end

-- First face card in the scoring hand (Photograph)
local function first_face(ctx)
    for _, c in ipairs(ctx.scoring_hand or {}) do
        if c:is_face() then return c == ctx.other_card end
    end
    return false
end

local function all_held(suits)
    for _, c in ipairs(G.hand and G.hand.cards or {}) do
        local ok = false
        for _, s in ipairs(suits) do if c:is_suit(s) then ok = true end end
        if not ok then return false end
    end
    return true
end

local function every_suit(cards)
    for _, s in ipairs({ "Hearts", "Diamonds", "Spades", "Clubs" }) do
        local found = false
        for _, c in ipairs(cards or {}) do if c:is_suit(s) then found = true end end
        if not found then return false end
    end
    return true
end

-- Not the most played hand (Obelisk): another hand has been played at least as often
local function not_most_played(name)
    local mine = G.GAME.hands[name] and G.GAME.hands[name].played or 0
    for k, h in pairs(G.GAME.hands) do
        if k ~= name and h.visible and h.played >= mine then return true end
    end
    return false
end

-- Every trigger is the condition of a base-game Joker (named after each one).
-- kind: "play" = each scored card, "hand" = each card held in hand, "main" = once per hand
R.TRIGGERS = {
    -- Greedy, Lusty, Wrathful and Gluttonous Joker
    { key = "hearts", kind = "play", text = "a {C:hearts}Heart{} is scored", check = function(ctx) return ctx.other_card:is_suit("Hearts") end },
    { key = "diamonds", kind = "play", text = "a {C:diamonds}Diamond{} is scored", check = function(ctx) return ctx.other_card:is_suit("Diamonds") end },
    { key = "spades", kind = "play", text = "a {C:spades}Spade{} is scored", check = function(ctx) return ctx.other_card:is_suit("Spades") end },
    { key = "clubs", kind = "play", text = "a {C:clubs}Club{} is scored", check = function(ctx) return ctx.other_card:is_suit("Clubs") end },
    -- Scary Face / Smiley Face
    { key = "face", kind = "play", text = "a {C:attention}face{} card is scored", check = function(ctx) return ctx.other_card:is_face() end },
    -- Photograph
    { key = "first_face", kind = "play", text = "the first {C:attention}face{} card is scored", check = first_face },
    -- Scholar
    { key = "ace", kind = "play", text = "an {C:attention}Ace{} is scored", check = function(ctx) return id(ctx.other_card) == 14 end },
    -- Even Steven / Odd Todd
    { key = "even", kind = "play", text = "a {C:attention}10, 8, 6, 4 or 2{} is scored", check = function(ctx) local i = id(ctx.other_card) return i >= 2 and i <= 10 and i % 2 == 0 end },
    { key = "odd", kind = "play", text = "an {C:attention}A, 9, 7, 5 or 3{} is scored", check = function(ctx) local i = id(ctx.other_card) return i == 14 or (i >= 3 and i <= 9 and i % 2 == 1) end },
    -- Fibonacci
    { key = "fibonacci", kind = "play", text = "an {C:attention}A, 2, 3, 5 or 8{} is scored", check = function(ctx) local i = id(ctx.other_card) return i == 14 or i == 2 or i == 3 or i == 5 or i == 8 end },
    -- Walkie Talkie
    { key = "ten_four", kind = "play", text = "a {C:attention}10 or 4{} is scored", check = function(ctx) local i = id(ctx.other_card) return i == 10 or i == 4 end },
    -- Hack
    { key = "low", kind = "play", text = "a {C:attention}2, 3, 4 or 5{} is scored", check = function(ctx) local i = id(ctx.other_card) return i >= 2 and i <= 5 end },
    -- Triboulet
    { key = "king_queen", kind = "play", text = "a {C:attention}King or Queen{} is scored", check = function(ctx) local i = id(ctx.other_card) return i == 13 or i == 12 end },

    -- Baron / Shoot the Moon / Reserved Parking / Raised Fist
    { key = "held_king", kind = "hand", text = "a {C:attention}King{} is held in hand", check = function(ctx) return id(ctx.other_card) == 13 end },
    { key = "held_queen", kind = "hand", text = "a {C:attention}Queen{} is held in hand", check = function(ctx) return id(ctx.other_card) == 12 end },
    { key = "held_face", kind = "hand", text = "a {C:attention}face{} card is held in hand", check = function(ctx) return ctx.other_card:is_face() end },
    { key = "held_lowest", kind = "hand", text = "a card held in hand is the {C:attention}lowest{} rank", check = function(ctx) return lowest_held(ctx.other_card) end },

    -- Jolly, Mad, Zany, Crazy, Droll Joker and The Family
    { key = "pair", kind = "main", text = "the hand contains a {C:attention}Pair{}", check = function(ctx) return has_hand(ctx, "Pair") end },
    { key = "two_pair", kind = "main", text = "the hand contains {C:attention}Two Pair{}", check = function(ctx) return has_hand(ctx, "Two Pair") end },
    { key = "three", kind = "main", text = "the hand contains {C:attention}Three of a Kind{}", check = function(ctx) return has_hand(ctx, "Three of a Kind") end },
    { key = "straight", kind = "main", text = "the hand contains a {C:attention}Straight{}", check = function(ctx) return has_hand(ctx, "Straight") end },
    { key = "flush", kind = "main", text = "the hand contains a {C:attention}Flush{}", check = function(ctx) return has_hand(ctx, "Flush") end },
    { key = "four", kind = "main", text = "the hand contains {C:attention}Four of a Kind{}", check = function(ctx) return has_hand(ctx, "Four of a Kind") end },
    -- Half Joker / Square Joker
    { key = "few_cards", kind = "main", text = "{C:attention}3{} or fewer cards are played", check = function(ctx) return #(ctx.full_hand or {}) <= 3 end },
    { key = "four_cards", kind = "main", text = "exactly {C:attention}4{} cards are played", check = function(ctx) return #(ctx.full_hand or {}) == 4 end },
    -- DNA / Dusk / Acrobat
    { key = "first_hand", kind = "main", text = "it's the {C:attention}first hand{} of the round", check = function() return G.GAME.current_round.hands_played == 0 end },
    { key = "final_hand", kind = "main", text = "it's the {C:attention}final hand{} of the round", check = function() return G.GAME.current_round.hands_left == 0 end },
    -- Mystic Summit / Delayed Gratification
    { key = "no_discards_left", kind = "main", text = "you have {C:attention}0{} discards left", check = function() return G.GAME.current_round.discards_left == 0 end },
    { key = "no_discards", kind = "main", text = "no {C:red}discards{} were used this round", check = function() return G.GAME.current_round.discards_used == 0 end },
    -- Bloodstone / Space Joker
    { key = "coin", kind = "main", text = "a {C:green}1 in 2{} chance hits", check = function(ctx, joker) return SMODS.pseudorandom_probability(joker, "rnd_coin", 1, 2) end },
    -- Ride the Bus
    { key = "no_face", kind = "main", text = "the hand has no {C:attention}face{} cards", check = function(ctx) return not any_face(ctx.scoring_hand) end },
    -- Card Sharp
    { key = "repeat_hand", kind = "main", text = "the hand was already played this round", check = function(ctx) local h = ctx.scoring_name and G.GAME.hands[ctx.scoring_name] return h and h.played_this_round > 1 end },
    -- Obelisk
    { key = "not_most_played", kind = "main", text = "it's not your {C:attention}most played{} hand", check = function(ctx) return ctx.scoring_name and not_most_played(ctx.scoring_name) end },
    -- Blackboard
    { key = "dark_held", kind = "main", text = "all held cards are {C:spades}Spades{} or {C:clubs}Clubs{}", check = function() return all_held({ "Spades", "Clubs" }) end },
    -- Flower Pot
    { key = "all_suits", kind = "main", text = "the hand has a {C:diamonds}Diamond{}, {C:clubs}Club{}, {C:hearts}Heart{} and {C:spades}Spade{}", check = function(ctx) return every_suit(ctx.scoring_hand) end },
}

local BY_KEY = {}
for _, t in ipairs(R.TRIGGERS) do BY_KEY[t.key] = t end

function R.trigger_of(joker)
    return joker and joker.ability and joker.ability.rnd_trigger and BY_KEY[joker.ability.rnd_trigger]
end

-- The three scoring steps a trigger can belong to
local function context_kind(context)
    if context.joker_main then return "main" end
    if context.individual and context.cardarea == G.play then return "play" end
    if context.individual and context.cardarea == G.hand and not context.end_of_round then return "hand" end
end

-- Contexts that only ask a question; they are always answered normally
local PASS_THROUGH = {
    mod_probability = true, fix_probability = true, check_enhancement = true,
    retrigger_joker_check = true, modify_scoring_hand = true, evaluate_poker_hand = true,
    debuff_hand = true, debuff_card = true, check_eternal = true, selling_self = true,
}

local function passes_through(context)
    for k in pairs(PASS_THROUGH) do
        if context[k] then return true end
    end
    return false
end

----------------------------------------------------------------------
-- Measuring a Joker: run it in a scoring step with nothing sticking
----------------------------------------------------------------------

local measuring = 0
local measure_card, measure_money = nil, 0
local MEASURE_IDS = { 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2 }
-- replaying: a Joker's own action is being run for its random trigger; its card checks
-- pass like when measuring, but what it does is real
local replaying = 0
-- force_enhanced: the card being looked at also counts as having every enhancement
local force_enhanced = false
-- Counts of events and money, to tell whether a Joker did something
local events_seen, money_seen = 0, 0

local add_event_ref = EventManager.add_event
function EventManager:add_event(...)
    events_seen = events_seen + 1
    if measuring > 0 then return end
    return add_event_ref(self, ...)
end

local ease_dollars_ref = ease_dollars
function ease_dollars(mod, ...)
    money_seen = money_seen + 1
    if measuring > 0 then
        if type(mod) == "number" then measure_money = measure_money + mod end
        return
    end
    return ease_dollars_ref(mod, ...)
end

if SMODS.has_enhancement then
    local has_enhancement_ref = SMODS.has_enhancement
    function SMODS.has_enhancement(card, key, ...)
        if force_enhanced and card == measure_card and (measuring > 0 or replaying > 0) then return true end
        return has_enhancement_ref(card, key, ...)
    end
end

-- While measuring, chance rolls always succeed and don't use up the run's seed, so a
-- Joker's own odds (Bloodstone's 1 in 2...) don't become an extra condition on the trigger
local pseudoseed_ref = pseudoseed
function pseudoseed(key, ...)
    if measuring > 0 then return 0.5 end
    return pseudoseed_ref(key, ...)
end

local pseudorandom_ref = pseudorandom
function pseudorandom(seed, min, max, ...)
    if measuring > 0 then
        if min and max then return max end
        return 0
    end
    return pseudorandom_ref(seed, min, max, ...)
end

if SMODS.pseudorandom_probability then
    local probability_ref = SMODS.pseudorandom_probability
    function SMODS.pseudorandom_probability(...)
        if measuring > 0 or replaying > 0 then return true end
        return probability_ref(...)
    end
end

local get_id_ref = Card.get_id
function Card:get_id(...)
    if self.rnd_force_id then return self.rnd_force_id end
    return get_id_ref(self, ...)
end

-- While measuring, the card being looked at counts as every suit and as a face card
local is_suit_ref = Card.is_suit
function Card:is_suit(...)
    if (measuring > 0 or replaying > 0) and measure_card == self then return true end
    return is_suit_ref(self, ...)
end

local is_face_ref = Card.is_face
function Card:is_face(...)
    if (measuring > 0 or replaying > 0) and measure_card == self then return true end
    return is_face_ref(self, ...)
end

local calculate_joker_ref = Card.calculate_joker

local COSMETIC = { card = true, colour = true, message_card = true, juice_card = true, focus = true, sound = true, delay = true }

-- Whether a result does anything beyond messages and colours
local function has_effect(ret)
    local node = ret
    while type(node) == "table" do
        for k in pairs(node) do
            if not COSMETIC[k] and k ~= "extra" then return true end
        end
        node = node.extra
    end
    return false
end

local function report(err)
    if sendWarnMessage then sendWarnMessage(tostring(err), "Randomizer") end
end

-- Copy a Joker's saved state without following links to Cards or other objects
-- (anything with a metatable is shared, not copied) and without looping on cycles.
-- Never use copy_table here: a Card's parent/children links loop back and overflow the stack.
local function safe_copy(t, seen)
    if type(t) ~= "table" or getmetatable(t) then return t end
    seen = seen or {}
    if seen[t] then return seen[t] end
    local out = {}
    seen[t] = out
    for k, v in pairs(t) do out[k] = safe_copy(v, seen) end
    return out
end

-- Run once with everything copied back afterwards; returns its result and any money it
-- tried to pay directly
local last_quiet_ability, last_quiet_other
local function run_quietly(joker, ctx, rank)
    local other = ctx.other_card
    local joker_ability, other_ability = joker.ability, other and other.ability
    joker.ability = safe_copy(joker_ability)
    if other then
        other.ability = safe_copy(other_ability)
        other.rnd_force_id = rank or nil
    end
    local buffers = G.GAME and { G.GAME.consumeable_buffer, G.GAME.joker_buffer, G.GAME.dollar_buffer }
    measuring, measure_card, measure_money = measuring + 1, other, 0
    local ok, ret = pcall(calculate_joker_ref, joker, ctx)
    local money = measure_money
    measuring, measure_card = measuring - 1, nil
    if other then
        other.rnd_force_id = nil
        other.ability = other_ability
    end
    last_quiet_ability, last_quiet_other = joker.ability, other and other.ability
    joker.ability = joker_ability
    if buffers then G.GAME.consumeable_buffer, G.GAME.joker_buffer, G.GAME.dollar_buffer = buffers[1], buffers[2], buffers[3] end
    if not ok then
        report(ret)
        return nil, 0
    end
    return ret, money
end

local function every_hand(poker_hands, played)
    local out = {}
    for k, v in pairs(poker_hands or {}) do out[k] = v end
    for _, name in ipairs(G.handlist or {}) do
        if not (out[name] and next(out[name])) then out[name] = { played } end
    end
    return out
end

local function spare_card()
    return (G.play and G.play.cards[1]) or (G.hand and G.hand.cards[1]) or (G.playing_cards or {})[1]
end

-- The scoring steps to measure in, built from what's happening now
local function scoring_contexts(context)
    local played = context.scoring_hand or (G.play and next(G.play.cards) and G.play.cards) or {}
    local base = {
        full_hand = context.full_hand or played,
        scoring_hand = played,
        scoring_name = context.scoring_name or "High Card",
        poker_hands = every_hand(context.poker_hands, played),
    }
    local function make(t)
        for k, v in pairs(base) do
            if t[k] == nil then t[k] = v end
        end
        return t
    end
    local card = context.other_card or spare_card()
    local list = { make({ joker_main = true, cardarea = G.jokers }) }
    if card then
        list[#list + 1] = make({ individual = true, cardarea = G.play, other_card = card })
        list[#list + 1] = make({ individual = true, cardarea = G.hand, other_card = card })
    end
    return list
end

----------------------------------------------------------------------
-- Random type and behaviour of a Joker's scoring numbers
--   type:      its Chips/Mult/XMult/XChips all turn into one random kind
--   behaviour: normal, scales (keeps growing), resets (grows, resets each round),
--              stacks (more each time in the same hand) or decays (shrinks each time)
----------------------------------------------------------------------

-- "Only randomize triggers" leaves a Joker's numbers, type and scaling alone
local JOKER_OPTIONS = { numbers = true, types = true, behaviours = true }
local function on(option)
    local config = MOD.config or {}
    if config.trigger_only and JOKER_OPTIONS[option] then return false end
    return config[option]
end

local SCORE_KEYS = {
    chips = "chips", chip_mod = "chips", h_chips = "chips",
    mult = "mult", mult_mod = "mult", h_mult = "mult",
    xmult = "xmult", x_mult = "xmult", Xmult = "xmult", Xmult_mod = "xmult", x_mult_mod = "xmult",
    xchips = "xchips", x_chips = "xchips", Xchip_mod = "xchips",
    dollars = "dollars", p_dollars = "dollars", h_dollars = "dollars",
}
local MULTIPLIERS = { xmult = true, xchips = true }
R.TYPES = { "chips", "mult", "xmult", "xchips" }
R.BEHAVIOURS = { "normal", "scales", "resets", "stacks", "decays" }

-- Every kind of number has a value in "points" so it can be turned into another kind
local POINTS = {
    mult = function(v) return v end,
    chips = function(v) return v / 4 end,
    xmult = function(v) return (v - 1) * 20 end,
    xchips = function(v) return (v - 1) * 10 end,
}
local FROM_POINTS = {
    -- A positive amount stays at least +1 / X1.1; a negative one can go as low as it goes
    mult = function(p) local v = math.floor(p + 0.5) return p > 0 and math.max(1, v) or v end,
    chips = function(p) local v = math.floor(p * 4 + 0.5) return p > 0 and math.max(1, v) or v end,
    xmult = function(p) local v = math.floor((1 + p / 20) * 10 + 0.5) / 10 return p > 0 and math.max(1.1, v) or v end,
    xchips = function(p) local v = math.floor((1 + p / 10) * 10 + 0.5) / 10 return p > 0 and math.max(1.1, v) or v end,
}

function R.convert(kind, value, to)
    if kind == to or not POINTS[kind] or not FROM_POINTS[to] then return value end
    return FROM_POINTS[to](POINTS[kind](value))
end

-- Pull the scoring numbers out of a result (following its extra chain); the result keeps
-- everything else it does. Returns the numbers and what's left of the result.
local function plain(v)
    if type(v) == "table" and to_number then
        local ok, n = pcall(to_number, v)
        if ok then return n end
    end
    return v
end

-- Copy just the ret -> ret.extra chain take_scoring edits. A deep copy would follow the
-- card references in a result (card.parent/children loop) and overflow the stack
local function copy_chain(ret)
    local out = {}
    for k, v in pairs(ret) do out[k] = v end
    if type(out.extra) == "table" then out.extra = copy_chain(out.extra) end
    return out
end

local function take_scoring(ret)
    local found
    local node = ret
    while type(node) == "table" do
        local removed = false
        for k, raw in pairs(node) do
            local norm = SCORE_KEYS[k]
            local v = plain(raw)
            if norm and type(v) == "number" then
                found = found or {}
                if MULTIPLIERS[norm] then found[norm] = (found[norm] or 1) * v else found[norm] = (found[norm] or 0) + v end
                node[k] = nil
                removed = true
            end
        end
        if removed then node.message = nil end
        node = node.extra
    end
    return found, has_effect(ret) and ret or nil
end

local function blank(effect)
    for k, v in pairs(effect) do
        if MULTIPLIERS[k] and v ~= 1 then return false end
        if not MULTIPLIERS[k] and v ~= 0 then return false end
    end
    return true
end

-- Multiply an effect by n (additive values times n, multipliers' bonus part times n)
local function times(effect, n)
    local out = {}
    for k, v in pairs(effect) do
        if MULTIPLIERS[k] then out[k] = 1 + (v - 1) * n else out[k] = v * n end
    end
    return out
end

local function add(a, b)
    local out = {}
    for k, v in pairs(a or {}) do out[k] = v end
    for k, v in pairs(b or {}) do
        if MULTIPLIERS[k] then out[k] = (out[k] or 1) + (v - 1) else out[k] = (out[k] or 0) + v end
    end
    return out
end

local function round_effect(effect)
    for k, v in pairs(effect) do
        if MULTIPLIERS[k] then effect[k] = math.floor(v * 100 + 0.5) / 100 else effect[k] = math.floor(v + 0.5) end
    end
    return effect
end

-- Apply the Joker's random type
local function retype(joker, effect)
    local to = on("types") and joker.ability.rnd_type
    if not to then return effect end
    local points = 0
    local out = { dollars = effect.dollars }
    for k, v in pairs(effect) do
        if POINTS[k] then points = points + POINTS[k](v) end
    end
    if points ~= 0 then out[to] = FROM_POINTS[to](points) end
    return out
end

-- Apply the Joker's random behaviour (this is where its state moves on)
local function behave(joker, effect)
    local mode = on("behaviours") and joker.ability.rnd_kind ~= "dollar" and joker.ability.rnd_behaviour or "normal"
    local a = joker.ability
    if mode == "scales" or mode == "resets" or mode == "stacks" then
        -- All three build up a running total; they differ only in when it resets
        a.rnd_total = add(a.rnd_total, effect)
        return safe_copy(a.rnd_total)
    elseif mode == "decays" then
        a.rnd_decay = a.rnd_decay or 2
        local out = times(effect, a.rnd_decay)
        a.rnd_decay = math.max(1, a.rnd_decay - 0.25)
        return out
    end
    return effect
end

local function number_text(v)
    local str = string.format("%.2f", v):gsub("0+$", ""):gsub("%.$", "")
    return str
end

-- "+4" / "-4"; money as "$4" / "-$4"
local function signed(v) return (v < 0 and "-" or "+") .. number_text(math.abs(v)) end
local function money(v) return (v < 0 and "-$" or "$") .. number_text(math.abs(v)) end

local UNIT_OF = { chips = " Chips", mult = " Mult", xchips = " Chips", xmult = " Mult" }
local EFFECT_ORDER = { "chips", "mult", "xchips", "xmult", "dollars" }

-- "+12 Mult, X2.2 Chips"; an empty total shows as "+0 Mult" / "X1 Mult" of its kind
function R.format_total(effect, kind)
    local parts = {}
    for _, k in ipairs(EFFECT_ORDER) do
        local v = effect and effect[k]
        if v then
            if k == "dollars" then parts[#parts + 1] = money(v)
            elseif MULTIPLIERS[k] then parts[#parts + 1] = "X" .. number_text(v) .. UNIT_OF[k]
            else parts[#parts + 1] = signed(v) .. UNIT_OF[k] end
        end
    end
    if #parts > 0 then return table.concat(parts, ", ") end
    kind = kind or "mult"
    if kind == "dollars" then return "$0" end
    return (MULTIPLIERS[kind] and "X1" or "+0") .. UNIT_OF[kind]
end

-- "+4 Mult" / "X0.4 Mult": how much one trigger adds
function R.format_increase(effect, kind)
    local parts = {}
    for _, k in ipairs(EFFECT_ORDER) do
        local v = effect and effect[k]
        if v then
            if k == "dollars" then parts[#parts + 1] = money(v)
            elseif MULTIPLIERS[k] then parts[#parts + 1] = "X" .. number_text(v - 1) .. UNIT_OF[k]
            else parts[#parts + 1] = signed(v) .. UNIT_OF[k] end
        end
    end
    if #parts > 0 then return table.concat(parts, ", ") end
    return R.format_total(nil, kind)
end

-- The kind a Joker's numbers come out as (its random type, or its own first kind)
function R.kind_of(joker, effect)
    if on("types") and joker.ability.rnd_type then return joker.ability.rnd_type end
    for _, k in ipairs(EFFECT_ORDER) do
        if effect and effect[k] then return k end
    end
    return "mult"
end

local function deliver(joker, effect)
    if not effect or blank(effect) then return nil end
    effect = round_effect(behave(joker, retype(joker, effect)))
    if blank(effect) then return nil end
    effect.card = joker
    return effect
end

-- What this Joker gives in the scoring step right now: its Chips/Mult/XMult/XChips and
-- money, before any random type or behaviour. nil if it gives nothing.
local function measure(joker, context)
    for _, ctx in ipairs(scoring_contexts(context or {})) do
        -- Hand-level Jokers that want a specific poker hand (Jolly, To Do List...) get to
        -- see every hand; card-level ones get to see every rank
        local tries = {}
        if ctx.other_card then
            for _, r in ipairs(MEASURE_IDS) do tries[#tries + 1] = { rank = r } end
        else
            tries[1] = { name = ctx.scoring_name }
            for _, n in ipairs(G.handlist or {}) do tries[#tries + 1] = { name = n } end
        end
        for _, t in ipairs(tries) do
            if t.name then ctx.scoring_name = t.name end
            local ret, money = run_quietly(joker, ctx, t.rank)
            local numbers = type(ret) == "table" and take_scoring(copy_chain(ret)) or nil
            if money and money ~= 0 then
                numbers = numbers or {}
                numbers.dollars = (numbers.dollars or 0) + money
            end
            if numbers and not blank(numbers) then return numbers end
        end
    end
end

-- Retriggers (Hack, Dusk, Sock and Buskin...): how many it would give this card
local function measure_repetitions(joker, context)
    for _, r in ipairs(MEASURE_IDS) do
        local ret = run_quietly(joker, context, r)
        if type(ret) == "table" and (ret.repetitions or 0) > 0 then return ret end
    end
end

-- The random trigger happened: give the Joker's scoring numbers
-- What the Joker gives when its own condition is met, read from its numbers (filled in
-- below). Used when running it can't make its condition pass (Mystic Summit needs 0
-- discards, Half Joker 3 cards, Flower Pot every suit...), so the new trigger always pays
local base_effect, dollar_effect

local function fire(joker, context)
    local numbers = measure(joker, context) or base_effect(joker)
        or (joker.ability.rnd_kind == "dollar" and dollar_effect(joker) or nil)
    local effect = numbers and deliver(joker, numbers)
    return effect
end

-- Cached for half a second so card text and JokerDisplay don't re-measure every frame
function R.increment(joker)
    if not (G.GAME and G.jokers and joker.ability) then return nil end
    local now = G.TIMERS and G.TIMERS.REAL or 0
    local cached = joker.rnd_measure
    if cached and now - cached.at < 0.5 and cached.type == joker.ability.rnd_type then return cached.value end
    local numbers = measure(joker, {}) or base_effect(joker)
        or (joker.ability.rnd_kind == "dollar" and dollar_effect(joker) or nil)
    local value = numbers and round_effect(retype(joker, numbers)) or nil
    joker.rnd_measure = { at = now, value = value, type = joker.ability.rnd_type }
    return value
end

-- What the next trigger would give, after its behaviour
function R.next_effect(joker)
    local inc = R.increment(joker)
    if not inc then return nil end
    local mode = on("behaviours") and joker.ability.rnd_behaviour or "normal"
    if mode == "scales" or mode == "resets" or mode == "stacks" then
        return round_effect(add(joker.ability.rnd_total, inc))
    elseif mode == "decays" then
        return round_effect(times(inc, joker.ability.rnd_decay or 2))
    end
    return inc
end

----------------------------------------------------------------------
-- Rolling a trigger
----------------------------------------------------------------------

local logged = 0

----------------------------------------------------------------------
-- Jokers that grow (Runner, Ride the Bus, Hologram...): they keep scoring their
-- "Currently" amount as normal, but they only grow on the random trigger. Their own
-- growth is undone, and the amount it grew by is remembered for the trigger to use.
----------------------------------------------------------------------

local function desc_text(key)
    local d = G.localization and G.localization.descriptions.Joker[key]
    if not (d and d.text) then return "" end
    local parts = {}
    local function walk(t)
        for _, l in ipairs(t) do
            if type(l) == "table" then walk(l) else parts[#parts + 1] = tostring(l) end
        end
    end
    walk(d.text)
    return (table.concat(parts, " "):gsub("{[^}]*}", ""))
end

base_effect = function(joker)
    local a = joker.ability
    local out = {}
    local function put(kind, v)
        v = plain(v)
        if type(v) ~= "number" then return end
        if kind == "xmult" then
            -- Below X1 only counts when the amount was rolled down (a negative or small factor)
            if v > 1 or (v < 1 and (a.rnd_factor or 1) < 1) then out.xmult = v end
        elseif v ~= 0 then
            out[kind] = out[kind] or v
        end
    end
    put("mult", a.t_mult)
    put("chips", a.t_chips)
    put("mult", a.mult)
    put("xmult", a.x_mult)
    local ex = a.extra
    if type(ex) == "table" then
        for k, v in pairs(ex) do
            local lk = tostring(k):lower()
            if not (lk:find("mod") or lk:find("gain") or lk:find("add") or lk:find("sub")) then
                if lk:find("x_?mult") or lk:find("xmult") then put("xmult", v)
                elseif lk:find("mult") then put("mult", v)
                elseif lk:find("chip") then put("chips", v) end
            end
        end
    elseif type(plain(ex)) == "number" then
        -- A bare number: its description says what it is ("X#1# Mult", "+#1# Chips")
        local text = desc_text(joker.config.center.key)
        if text:find("X#1#%s*Mult") or text:find("X%d") then put("xmult", ex)
        elseif text:find("#1#%s*Chips") then put("chips", ex)
        elseif text:find("#1#%s*Mult") then put("mult", ex) end
    end
    return next(out) and out or nil
end

local function grows_text(text)
    return (text:find("[Gg]ains?[^%.]-Mult") or text:find("[Gg]ains?[^%.]-Chips")) and true or false
end

function R.grows(joker)
    local a = joker.ability
    if a.rnd_grows == nil then a.rnd_grows = grows_text(desc_text(joker.config.center.key)) end
    return a.rnd_grows
end

local GROWTH_HINTS = { "mult", "chip" }
local GROWTH_SKIP = { "mod", "gain", "add", "sub", "inc", "odds", "every", "remaining", "size", "limit" }
local function growth_key(k)
    local lk = tostring(k):lower()
    if lk:sub(1, 4) == "rnd_" then return false end
    for _, w in ipairs(GROWTH_SKIP) do if lk:find(w, 1, true) then return false end end
    for _, h in ipairs(GROWTH_HINTS) do if lk:find(h, 1, true) then return true end end
    return false
end

-- The numbers a Joker grows: its Mult, XMult and the Mult/Chips in its extra table
local function growth_snapshot(a)
    local snap = {}
    for _, k in ipairs({ "mult", "x_mult" }) do
        if type(plain(a[k])) == "number" then snap[#snap + 1] = { a, k, a[k] } end
    end
    if type(a.extra) == "table" then
        for k, v in pairs(a.extra) do
            if growth_key(k) and type(plain(v)) == "number" then snap[#snap + 1] = { a.extra, k, v } end
        end
    end
    return snap
end

-- Undo whatever the Joker grew by on its own, and remember how much it grew
local function undo_growth(joker, snap)
    local learned
    for _, entry in ipairs(snap) do
        local t, k, old = entry[1], entry[2], entry[3]
        local now_n, old_n = plain(t[k]), plain(old)
        if now_n ~= old_n then
            t[k] = old
            if type(now_n) == "number" and type(old_n) == "number" and now_n > old_n then
                learned = learned or {}
                learned[#learned + 1] = { where = t == joker.ability and "top" or "extra", key = k, amount = now_n - old_n }
            end
        end
    end
    if learned then joker.ability.rnd_gain = learned end
end

-- Before a Joker has grown once on its own, guess the amount from its settings
-- (Runner's chip_mod grows its chips, Hologram's 0.25 grows its XMult...)
local function guess_gain(joker)
    local a = joker.ability
    local ex = a.extra
    if type(plain(ex)) == "number" and plain(ex) ~= 0 then
        local n = plain(ex)
        return { { where = "top", key = n < 1 and "x_mult" or "mult", amount = n } }
    end
    if type(ex) ~= "table" then return nil end
    for k, v in pairs(ex) do
        local lk, n = tostring(k):lower(), plain(v)
        if type(n) == "number" and n ~= 0 and (lk:find("mod") or lk:find("gain") or lk:find("add") or lk:find("inc")) then
            local base = lk:gsub("_?mod$", ""):gsub("_?gain$", ""):gsub("_?add$", ""):sub(1, 4)
            for k2, v2 in pairs(ex) do
                if k2 ~= k and growth_key(k2) and type(plain(v2)) == "number" and tostring(k2):lower():find(base, 1, true) then
                    return { { where = "extra", key = k2, amount = n } }
                end
            end
            return { { where = "top", key = lk:find("x") and "x_mult" or "mult", amount = n } }
        end
    end
end

local function apply_gain(joker)
    local a = joker.ability
    local gains = a.rnd_gain or guess_gain(joker)
    local applied = false
    for _, g in ipairs(gains or {}) do
        local t = g.where == "extra" and a.extra or a
        if type(t) == "table" and t[g.key] ~= nil then
            local ok, v = pcall(function() return t[g.key] + g.amount end)
            if ok then t[g.key], applied = v, true end
        end
    end
    return applied
end

----------------------------------------------------------------------
-- Jokers that pay money at end of round (Golden Joker, Cloud 9, Rocket...): they pay
-- on the random trigger instead
----------------------------------------------------------------------

local DOLLAR_JOKERS = { j_golden = true, j_cloud_9 = true, j_rocket = true, j_satellite = true, j_delayed_grat = true }
local function pays_at_end(joker)
    local center = joker.config.center
    return DOLLAR_JOKERS[center.key] or type(center.calc_dollar_bonus) == "function"
end

local dollar_bonus_ref = Card.calculate_dollar_bonus
function Card:calculate_dollar_bonus(...)
    if measuring == 0 and self.ability and self.ability.rnd_kind == "dollar" and R.trigger_of(self) then return nil end
    return dollar_bonus_ref(self, ...)
end

dollar_effect = function(joker)
    local ok, n = pcall(dollar_bonus_ref, joker)
    n = ok and plain(n) or nil
    if type(n) ~= "number" or n <= 0 then
        local ex = joker.ability.extra
        n = plain(type(ex) == "table" and ex.dollars or ex)
    end
    if type(n) == "number" and n ~= 0 then return { dollars = n } end
end

----------------------------------------------------------------------
-- Jokers that do something other than score (create cards, gain sell value, level up
-- hands...): what they do now happens on the random trigger instead of their own
-- condition. Where a Joker acts is known for base-game Jokers and learned for others the
-- first time it acts.
----------------------------------------------------------------------

-- The step a context belongs to, most specific first
local ACTION_KEYS = {
    "setting_blind", "end_of_round", "open_booster", "skipping_booster", "buying_card", "selling_card",
    "selling_self", "reroll_shop", "ending_shop", "first_hand_drawn", "hand_drawn", "pre_discard", "discard",
    "skip_blind", "using_consumeable", "playing_card_added", "remove_playing_cards", "before", "after",
    "joker_main", "individual",
}
local ASKING = {}
for k in pairs(PASS_THROUGH) do ASKING[k] = true end
ASKING.selling_self = nil
local function action_key(context)
    if context.repetition or context.destroying_card or context.game_over then return nil end
    for k in pairs(ASKING) do
        if context[k] then return nil end
    end
    for _, k in ipairs(ACTION_KEYS) do
        if context[k] then return k end
    end
end

local KNOWN_ACTIONS = {
    j_hallucination = "open_booster", j_cartomancer = "setting_blind", j_marble = "setting_blind",
    j_riff_raff = "setting_blind", j_burglar = "setting_blind", j_chicot = "setting_blind",
    j_certificate = "first_hand_drawn", j_8_ball = "individual", j_seance = "joker_main",
    j_superposition = "joker_main", j_vagabond = "joker_main", j_space = "before", j_dna = "before",
    j_midas_mask = "before", j_gift = "end_of_round", j_egg = "end_of_round", j_turtle_bean = "end_of_round",
    j_invisible = "selling_self", j_luchador = "selling_self", j_diet_cola = "selling_self",
    j_perkeo = "ending_shop", j_mail = "discard", j_faceless = "discard", j_trading = "discard",
    j_burnt = "pre_discard", j_hiker = "individual", j_ticket = "individual",
}

-- Numbers in a Joker's state, to see whether running it changed anything (Egg's sell value)
local function numbers_of(a)
    local out = {}
    for k, v in pairs(a or {}) do
        if type(plain(v)) == "number" and tostring(k):sub(1, 4) ~= "rnd_" then out[k] = plain(v) end
    end
    if a and type(a.extra) == "table" then
        for k, v in pairs(a.extra) do
            if type(plain(v)) == "number" then out["extra." .. tostring(k)] = plain(v) end
        end
    end
    return out
end
local function changed(before, after)
    for k, v in pairs(after) do if before[k] ~= v then return true end end
    for k in pairs(before) do if after[k] == nil then return true end end
    return false
end

local function acted(ret, events, money, before, after)
    return events > 0 or money or (type(ret) == "table" and has_effect(ret)) or changed(before, after)
end

local function flags_of(context)
    local out = {}
    for k, v in pairs(context or {}) do
        if type(v) ~= "table" and type(v) ~= "function" and type(v) ~= "userdata" then out[k] = v end
    end
    return out
end

-- A context for the Joker's own step, built from what's happening now
local function action_context(joker, act, context)
    local ctx = flags_of(joker.rnd_flags)
    ctx[act] = ctx[act] or true
    local played = context.scoring_hand or (G.play and next(G.play.cards) and G.play.cards) or {}
    ctx.cardarea = (act == "individual" and (context.cardarea == G.hand and G.hand or G.play)) or G.jokers
    ctx.full_hand = context.full_hand or played
    ctx.scoring_hand = played
    ctx.scoring_name = context.scoring_name or "High Card"
    ctx.poker_hands = every_hand(context.poker_hands, played)
    ctx.other_card = context.other_card or spare_card()
    ctx.blind = G.GAME and G.GAME.blind
    ctx.card = joker
    ctx.cards, ctx.removed = {}, {}
    if act == "end_of_round" then ctx.main_eval, ctx.game_over = true, false end
    return ctx
end

-- Find a way to make the Joker act (any rank, then also counting as enhanced) without it
-- sticking, then do it for real. Returns the Joker's result, or nil if it can't act.
local DROP = { remove = true, saved = true, debuff = true, prevent_debuff = true }
local function replay(joker, act, context)
    local ctx = action_context(joker, act, context)
    local other = ctx.other_card
    local found
    local ranks = { false }
    if other then for _, r in ipairs(MEASURE_IDS) do ranks[#ranks + 1] = r end end
    for _, enhanced in ipairs({ false, true }) do
        force_enhanced = enhanced
        for _, rank in ipairs(ranks) do
            local before = numbers_of(joker.ability)
            local before_other = other and numbers_of(other.ability) or {}
            local e = events_seen
            local ret, money = run_quietly(joker, ctx, rank or nil)
            -- Hiker changes the card, not itself
            local other_changed = other and changed(before_other, numbers_of(last_quiet_other))
            if other_changed or acted(ret, events_seen - e, money ~= 0, before, numbers_of(last_quiet_ability)) then
                found = { rank = rank, enhanced = enhanced }
                break
            end
        end
        if found then break end
    end
    force_enhanced = false
    if not found then return nil end

    force_enhanced = found.enhanced
    measure_card = other
    if other and found.rank then other.rnd_force_id = found.rank end
    replaying = replaying + 1
    local ok, ret = pcall(calculate_joker_ref, joker, ctx)
    replaying = replaying - 1
    if other then other.rnd_force_id = nil end
    measure_card, force_enhanced = nil, false
    if not ok then
        report(ret)
        return nil
    end
    if type(ret) ~= "table" then return { message = localize("k_active_ex"), colour = G.C.FILTER, card = joker } end
    for k in pairs(DROP) do ret[k] = nil end
    ret.card = joker
    return ret
end

-- Jokers whose own condition can't be faked: on the trigger they just do their effect
local function make_tarot(joker, seed)
    if not (G.consumeables and #G.consumeables.cards + G.GAME.consumeable_buffer < G.consumeables.config.card_limit) then return nil end
    G.GAME.consumeable_buffer = G.GAME.consumeable_buffer + 1
    G.E_MANAGER:add_event(Event({ func = function()
        local card = create_card("Tarot", G.consumeables, nil, nil, nil, nil, nil, seed)
        card:add_to_deck()
        G.consumeables:emplace(card)
        G.GAME.consumeable_buffer = 0
        return true
    end }))
    return { message = localize("k_plus_tarot"), colour = G.C.PURPLE, card = joker }
end

local function extra_number(joker, key, default)
    local ex = joker.ability.extra
    local v = plain(type(ex) == "table" and ex[key] or ex)
    return type(v) == "number" and v or default
end

local OVERRIDES = {
    -- DNA: add a permanent copy of the card to the deck and draw it to hand
    j_dna = function(joker, context)
        local source = context.other_card or (context.full_hand or {})[1]
        if not (source and G.hand and G.deck) then return nil end
        G.playing_card = (G.playing_card and G.playing_card + 1) or 1
        local copy = copy_card(source, nil, nil, G.playing_card)
        copy:add_to_deck()
        G.deck.config.card_limit = G.deck.config.card_limit + 1
        table.insert(G.playing_cards, copy)
        G.hand:emplace(copy)
        copy.states.visible = nil
        G.E_MANAGER:add_event(Event({ func = function() copy:start_materialize() return true end }))
        if SMODS.calculate_context then SMODS.calculate_context({ playing_card_added = true, cards = { copy } }) end
        return { message = localize("k_copied_ex"), colour = G.C.CHIPS, card = joker }
    end,
    -- Faceless Joker: earn $5
    j_faceless = function(joker) return { dollars = extra_number(joker, "dollars", 5), card = joker } end,
    -- Trading Card: earn $3
    j_trading = function(joker) return { dollars = extra_number(joker, "dollars", 3), card = joker } end,
    -- Vagabond and Superposition: create a Tarot card
    j_vagabond = function(joker) return make_tarot(joker, "vag") end,
    j_superposition = function(joker) return make_tarot(joker, "sup") end,
}

----------------------------------------------------------------------
-- Blueprint and Brainstorm: every time their trigger hits, they give the copied
-- Joker's effect (what that Joker gives on its own trigger)
----------------------------------------------------------------------

local COPIERS = {
    j_blueprint = function(joker)
        for i, j in ipairs(G.jokers and G.jokers.cards or {}) do
            if j == joker then return G.jokers.cards[i + 1] end
        end
    end,
    j_brainstorm = function() return G.jokers and G.jokers.cards[1] end,
}

local function copy_target(copier)
    local find = COPIERS[copier.config.center.key]
    local target = find and find(copier)
    if not target or target == copier or target.debuff then return nil end
    if target.config.center.blueprint_compat == false then return nil end
    return target
end

local copying = 0
local function copied_effect(copier, context)
    local target = copy_target(copier)
    if not target or copying > 3 then return nil end
    copying = copying + 1
    local ok, out = pcall(function()
        local kind = R.category(target)
        local effect
        if kind == "copy" then
            effect = copied_effect(target, context)
        elseif kind == "action" then
            local own = OVERRIDES[target.config.center.key]
            if own then effect = own(copier, context)
            elseif target.ability.rnd_action then effect = replay(target, target.ability.rnd_action, context) end
        else
            local numbers = measure(target, context) or (kind ~= "grow" and base_effect(target) or nil)
                or (kind == "dollar" and dollar_effect(target) or nil)
            if numbers and not blank(numbers) then effect = round_effect(numbers) end
        end
        if effect then effect.card = copier end
        return effect
    end)
    copying = copying - 1
    if not ok then
        report(out)
        return nil
    end
    return out
end

-- What kind of Joker this is: "grow", "dollar", "score", "action" or "copy"
local scores
function R.category(joker)
    local a = joker.ability
    if COPIERS[joker.config.center.key] then a.rnd_kind = "copy" end
    if not a.rnd_kind then
        local key = joker.config.center.key
        if R.grows(joker) then a.rnd_kind = "grow"
        elseif pays_at_end(joker) then a.rnd_kind = "dollar"
        elseif KNOWN_ACTIONS[key] then a.rnd_kind, a.rnd_action = "action", KNOWN_ACTIONS[key]
        elseif scores(joker) then a.rnd_kind = "score"
        else a.rnd_kind = "action" end
    end
    return a.rnd_kind
end

-- Whether the card text shows the new trigger (a Joker that hasn't acted yet, and isn't
-- a known one, still shows its own text; one that never acts is passive)
function R.shows_trigger(joker)
    return R.category(joker) ~= "action" or joker.ability.rnd_action ~= nil
end

R.action_key, R.replay, R.acted, R.numbers_of, R.flags_of = action_key, replay, acted, numbers_of, flags_of

-- Jokers that give Chips, Mult, money or retriggers while scoring
scores = function(joker)
    if measure(joker, {}) or base_effect(joker) then return true end

    local card = spare_card()
    if card then
        local ctx = { repetition = true, cardarea = G.play, other_card = card, full_hand = { card }, scoring_hand = { card } }
        if measure_repetitions(joker, ctx) then return true end
    end
    return false
end

function R.note_passive(joker)
    if sendInfoMessage then
        sendInfoMessage(("%s gives nothing while scoring, so it keeps its own trigger"):format(
            tostring(joker.config and joker.config.center and joker.config.center.key)), "Randomizer")
    end
end

-- Every Joker gets a trigger
function R.roll_trigger(joker)
    local trigger = pseudorandom_element(R.TRIGGERS, pseudoseed("rnd_trigger"))
    joker.ability.rnd_trigger = trigger.key
    R.category(joker)
    -- Type and behaviour are always rolled; the settings decide whether they apply
    joker.ability.rnd_type = pseudorandom_element(R.TYPES, pseudoseed("rnd_type"))
    joker.ability.rnd_behaviour = pseudorandom_element(R.BEHAVIOURS, pseudoseed("rnd_behaviour"))
    if logged < 40 and R.log then
        logged = logged + 1
        R.log(("%s -> trigger '%s' (%s)"):format(tostring(joker.config.center.key), trigger.key, tostring(joker.ability.rnd_kind)))
    end
end

----------------------------------------------------------------------
-- The hook: a Joker's scoring numbers only come on its random trigger
----------------------------------------------------------------------

-- Logs the first scoring events, so a log shows whether a Joker's own effect was blocked
-- and when its random trigger fired
local scoring_logged = 0
local function describe(numbers)
    local parts = {}
    for k, v in pairs(numbers or {}) do parts[#parts + 1] = k .. " " .. tostring(v) end
    table.sort(parts)
    return table.concat(parts, ", ")
end
-- How many times a trigger hits in this hand, counting each card once (retriggers of a
-- card don't count): played-card triggers count the scoring cards that match, held-card
-- triggers the held cards that match, whole-hand triggers 1 if they hit
local function hits_in_hand(trigger, context, joker)
    local function count(cards, extra)
        local n = 0
        for _, c in ipairs(cards or {}) do
            local ctx = { other_card = c, scoring_hand = context.scoring_hand, full_hand = context.full_hand }
            for k, v in pairs(extra or {}) do ctx[k] = v end
            local ok, hit = pcall(trigger.check, ctx, joker)
            if ok and hit then n = n + 1 end
        end
        return n
    end
    if trigger.kind == "play" then return count(context.scoring_hand) end
    if trigger.kind == "hand" then return count(G.hand and G.hand.cards) end
    local ok, hit = pcall(trigger.check, context, joker)
    return (ok and hit) and 1 or 0
end

-- A retrigger Joker (Hack, Mime...) retriggers the cards it normally works on (every
-- played card for Hack, every held card for Mime) its normal amount, once for each time
-- its trigger hits in the hand: Hack on "a Heart is scored" with 3 Hearts played
-- retriggers each card 3 times
local function retriggers(joker, context, trigger)
    local base = measure_repetitions(joker, context)
    if not base then return nil end
    local hits = hits_in_hand(trigger, context, joker)
    if hits <= 0 then return nil end
    base.repetitions = (base.repetitions or 1) * hits
    return base
end

-- Any Joker can be retriggered as many times as it's retriggered. A retrigger Joker is
-- just never set off by what it retriggered: not by its own Joker retrigger, and not by
-- a card's retriggered scoring (cards ask for retriggers on their first scoring only)
local function retriggered_by_itself(joker, context)
    if context.retrigger_joker and context.retrigger_joker == joker then return true end
    if context.repetition and context.other_card and context.other_card.repetition_trigger then return true end
    return false
end

local function note_scoring(joker, kind, trigger, blocked, effect)
    if scoring_logged >= 60 or not (R.log and (blocked or effect)) then return end
    scoring_logged = scoring_logged + 1
    local key = tostring(joker.config.center.key)
    if blocked then R.log(("%s [%s]: own effect blocked (%s)"):format(key, kind, describe(blocked))) end
    if effect then
        local shown = {}
        for k, v in pairs(effect) do if k ~= "card" then shown[k] = v end end
        R.log(("%s [%s]: trigger '%s' fired (%s)"):format(key, kind, trigger.key, describe(shown)))
    end
end

function Card:calculate_joker(context, ...)
    local trigger = self.ability and self.ability.set == "Joker" and self.ability.rnd_trigger and BY_KEY[self.ability.rnd_trigger]
    -- Not randomized, being measured, or copied by Blueprint/Brainstorm (the copier's own
    -- trigger decides when it copies)
    if not trigger or measuring > 0 or context.blueprint then
        return calculate_joker_ref(self, context, ...)
    end

    if retriggered_by_itself(self, context) then return nil end
    local category = R.category(self)

    -- Blueprint/Brainstorm: their own copying is replaced by giving the copied Joker's
    -- effect each time their trigger hits
    if category == "copy" then
        if context.repetition then
            local target = copy_target(self)
            local reps = target and retriggers(target, context, trigger)
            if reps then reps.card = self end
            return reps
        end
        local kind = context_kind(context)
        if kind then
            if not (kind == trigger.kind and trigger.check(context, self)) then return nil end
            local effect = copied_effect(self, context)
            if effect then note_scoring(self, kind, trigger, nil, effect) end
            return effect
        end
        if action_key(context) then return nil end
        return calculate_joker_ref(self, context, ...)
    end

    -- Jokers that act (create cards, gain sell value...) act on the random trigger
    -- instead of in their own step
    if category == "action" then
        local a = self.ability
        local step = action_key(context)
        local kind = context_kind(context)
        local hit = a.rnd_action and kind and kind == trigger.kind and trigger.check(context, self)
        local ret, post
        if step and step == a.rnd_action then
            -- Its own step: blocked (but remembered, to copy its details on the trigger)
            self.rnd_flags = flags_of(context)
        elseif step and not a.rnd_action then
            -- Not known yet where it acts: let it act this once and learn the step
            local before, e, m = numbers_of(a), events_seen, money_seen
            ret, post = calculate_joker_ref(self, context, ...)
            if acted(ret, events_seen - e, money_seen ~= m, before, numbers_of(a)) then
                a.rnd_action, self.rnd_flags = step, flags_of(context)
                if R.log then R.log(("%s acts in '%s', now on trigger '%s'"):format(tostring(self.config.center.key), step, trigger.key)) end
            end
        else
            ret, post = calculate_joker_ref(self, context, ...)
        end
        if hit then
            local own_effect = OVERRIDES[self.config.center.key]
            local extra
            if own_effect then
                local ok, res = pcall(own_effect, self, context)
                if ok then extra = res else report(res) end
            else
                extra = replay(self, a.rnd_action, context)
            end
            if extra then
                if R.log and scoring_logged < 60 then
                    scoring_logged = scoring_logged + 1
                    R.log(("%s [%s]: trigger '%s' fired (acted)"):format(tostring(self.config.center.key), kind, trigger.key))
                end
                if type(ret) == "table" then return SMODS.merge_effects({ ret, extra }), post end
                return extra, post
            end
        end
        return ret, post
    end

    -- Growing Jokers grow on the random trigger instead of their own condition
    if category == "grow" then
        local kind = context_kind(context)
        local hit = kind and kind == trigger.kind and trigger.check(context, self)
        local grew = hit and apply_gain(self)
        local snap = growth_snapshot(self.ability)
        local ret, post = calculate_joker_ref(self, context, ...)
        undo_growth(self, snap)
        if grew then
            if R.log and scoring_logged < 60 then
                scoring_logged = scoring_logged + 1
                R.log(("%s [%s]: trigger '%s' fired (grew)"):format(tostring(self.config.center.key), kind, trigger.key))
            end
            local msg = { message = localize("k_upgrade_ex"), colour = G.C.FILTER, card = self }
            if type(ret) == "table" then return SMODS.merge_effects({ msg, ret }), post end
            return msg, post
        end
        return ret, post
    end

    if context.before and self.ability.rnd_behaviour == "stacks" then self.ability.rnd_total = nil end
    if context.end_of_round and context.main_eval then
        if self.ability.rnd_behaviour == "resets" then self.ability.rnd_total = nil end
        if self.ability.rnd_behaviour == "decays" then self.ability.rnd_decay = nil end
    end

    -- Retriggers happen on the random trigger instead of their own condition
    if context.repetition then
        return retriggers(self, context, trigger)
    end

    local kind = context_kind(context)
    if not kind then
        -- Everything outside the scoring steps works exactly as normal
        return calculate_joker_ref(self, context, ...)
    end

    -- In the scoring steps its own result is replaced: its scoring numbers come only on
    -- the random trigger, anything else it does here still happens
    local own = calculate_joker_ref(self, context, ...)
    local blocked, rest
    if type(own) == "table" then blocked, rest = take_scoring(own) end
    local effect = (kind == trigger.kind and trigger.check(context, self)) and fire(self, context) or nil
    note_scoring(self, kind, trigger, blocked, effect)
    if rest and effect then return SMODS.merge_effects({ rest, effect }) end
    return rest or effect
end

----------------------------------------------------------------------
-- Card text: the Joker's own description with its condition taken out and the new
-- trigger written in. Works on the raw localization text, so it applies to Jokers from
-- any mod. Rewritten descriptions are cached per Joker key and trigger.
----------------------------------------------------------------------

local function trim(str)
    return (str:gsub("^%s+", ""):gsub("%s+$", ""))
end

-- Visible length of a line, without {formatting} tags
local function visible_len(str)
    return #(str:gsub("{[^}]*}", ""))
end

-- Condition clauses at the start of a description: "If ...,", "When ...,", "At end of round,"
local LEADING = {
    "^[Ii]f [^,]-, ", "^[Ww]hen [^,]-, ", "^[Ee]ach time [^,]-, ", "^[Ee]very time [^,]-, ",
    "^[Aa]t end of [^,]-, ", "^[Aa]t the end of [^,]-, ", "^[Aa]fter [^,]-, ", "^[Bb]efore [^,]-, ",
    "^[Oo]n [^,]-, ", "^[Dd]uring [^,]-, ", "^[Uu]pon [^,]-, ", "^[Ee]very [^,]-hands?[^,]-, ",
    "^[Ff]or the [^,]-, ", "^[Oo]nce per [^,]-, ",
    -- "1 in 2 chance to create...", "1 in 2 chance for played Hearts to give..."
    "^[^,]-[Cc]hance to ", "^[^,]-[Cc]hance for [^,]- to ",
    -- Trading Card only pays on its trigger
    "^[Dd]estroy it and ",
}

-- "Played cards with X suit give ...", "Each King held in hand gives ..." -> "Gives ..."
local SUBJECTS = {
    "^[Pp]layed cards with .- give ", "^[Pp]layed .- cards give ", "^[Pp]layed .- give ",
    "^[Ee]ach played .- gives? ", "^[Ee]very played .- gives? ", "^[Ee]ach .- held in hand gives? ",
    "^[Ee]ach .- in hand gives? ", "^[Ee]ach .- gives? ", "^[Ee]very .- gives? ",
    "^[Cc]ards held in hand give ", "^[Ss]cored .- give ",
}

-- Condition clauses at the end: "... if played hand contains a Pair", "... when scored"
local TRAILING = {
    " if .*$", " when .*$", " whenever .*$", " each time .*$", " every time .*$",
    " at end of round.*$", " at the end of .*$", " on the first hand.*$", " on first hand.*$",
    " on the final hand.*$", " on final hand.*$", " on the last hand.*$", " during the .*$",
    " after each .*$", " per hand played.*$", " per [^%s]*consecutive.*$", " without a .*$",
}

-- Capitalise the first visible letter, skipping {formatting} tags
local function capitalise(s)
    local i = 1
    while i <= #s do
        local c = s:sub(i, i)
        if c == "{" then
            local close = s:find("}", i, true)
            if not close then break end
            i = close + 1
        elseif c:match("%a") then
            return s:sub(1, i - 1) .. c:upper() .. s:sub(i + 1)
        else
            i = i + 1
        end
    end
    return s
end

-- Growing Jokers also say how often they grow: "per reroll", "for each card sold"...
local GROW_TRAILING = { " per .*$", " for each .*$", " for every .*$", " every .*$" }

local function strip_condition(sentence, grows)
    local s = trim(sentence)
    local again = true
    while again do
        again = false
        for _, pat in ipairs(LEADING) do
            local out = s:gsub(pat, "", 1)
            if out ~= s then s, again = trim(out), true end
        end
    end
    for _, pat in ipairs(SUBJECTS) do
        local out = s:gsub(pat, "Gives ", 1)
        if out ~= s then
            s = out
            break
        end
    end
    for _, pat in ipairs(TRAILING) do
        local out = s:gsub(pat, "", 1)
        if out ~= s then s = trim(out) end
    end
    -- A chance for the Joker to destroy itself is its own downside, not when it scores
    s = s:gsub("%s*{[^}]*}#%d+# in #%d+#{}%s*chance this.*$", "")
    for _, pat in ipairs(grows and GROW_TRAILING or {}) do
        local out = s:gsub(pat, "", 1)
        if out ~= s then s = trim(out) end
    end
    -- Retrigger Jokers retrigger whichever card set them off
    if s:match("^[Rr]etrigger") then s = "Retrigger played cards" end
    s = s:gsub("[%.,]+$", "")
    return capitalise(s)
end

local function wrap(text, width)
    local lines, line = {}, ""
    for word in text:gmatch("%S+") do
        local candidate = line == "" and word or (line .. " " .. word)
        if visible_len(candidate) > width and line ~= "" then
            lines[#lines + 1] = line
            line = word
        else
            line = candidate
        end
    end
    if line ~= "" then lines[#lines + 1] = line end
    return lines
end

-- Lines in brackets ("(Currently +4 Mult)") are kept as they are
local function is_note(line)
    local plain = trim(line:gsub("{[^}]*}", ""))
    return plain:sub(1, 1) == "("
end

-- Retyping card text: "{C:mult}+#1#{} Mult" becomes "{C:chips}+#41#{} Chips", where var
-- 41 is filled in with the converted amount when the text is shown
local SLOT_START = 40
local TAG = { chips = "{C:chips}", mult = "{C:mult}", xmult = "{X:mult,C:white}", xchips = "{X:chips,C:white}" }
local UNIT = { chips = "Chips", mult = "Mult", xmult = "Mult", xchips = "Chips" }

local function retype_text(text, to, conversions)
    local out, pos = "", 1
    while true do
        local a, b, num = text:find("#(%d+)#", pos)
        if not a then break end
        local rest = text:sub(b + 1)
        local plain = rest
        repeat
            local stripped = plain:gsub("^%s*{[^}]*}", "", 1)
            local changed_tag = stripped ~= plain
            plain = stripped
        until not changed_tag
        local unit = plain:match("^%s*(%a+)")
        if unit == "Mult" or unit == "Chips" then
            local prefix = text:sub(a - 1, a - 1)
            local from = (prefix == "X" and "x" or "") .. unit:lower()
            local slot = SLOT_START + #conversions + 1
            conversions[#conversions + 1] = { from_var = tonumber(num), from = from, slot = slot }
            local before = text:sub(pos, a - 1)
            if prefix == "X" or prefix == "+" then before = before:sub(1, -2) end
            before = before:gsub("{[^}]*}(%s*)$", TAG[to] .. "%1", 1)
            local u1, u2 = rest:find(unit, 1, true)
            -- XMult/XChips numbers sit in a box, which needs a space either side
            local number = MULTIPLIERS[to] and (" X#" .. slot .. "# ") or ("+#" .. slot .. "#")
            out = out .. before .. number .. rest:sub(1, u1 - 1) .. UNIT[to]
            pos = b + u2 + 1
        else
            out = out .. text:sub(pos, b)
            pos = b + 1
        end
    end
    return out .. text:sub(pos)
end

-- #61# = what one trigger adds, #60# = the current total
local BEHAVIOUR_LINES = {
    scales = { "{C:attention}Increases{} by {C:attention}#61#{}", "every trigger", "{C:inactive}(Currently #60#)" },
    resets = { "{C:attention}Increases{} by {C:attention}#61#{}", "every trigger, resets at", "end of round", "{C:inactive}(Currently #60#)" },
    stacks = { "{C:attention}Increases{} by {C:attention}#61#{}", "every trigger, resets", "after each hand", "{C:inactive}(Currently #60#)" },
    decays = { "{C:red}Decreases{} by {C:attention}#61#{}", "every trigger, down to normal,", "resets at end of round", "{C:inactive}(Currently #60#)" },
}

local function rewrite_lines(lines, trigger, joker, conversions)
    local body, notes = {}, {}
    for _, line in ipairs(lines) do
        if type(line) == "string" then
            if is_note(line) then notes[#notes + 1] = line else body[#body + 1] = line end
        end
    end
    local joined = table.concat(body, " ")
    -- Growing Jokers keep their own scoring, so type and behaviour don't apply to them
    local grows = R.grows(joker)
    -- Only Jokers that score get a random type and behaviour
    local own = R.category(joker) ~= "score"
    local to = not own and on("types") and joker.ability.rnd_type
    -- Jokers that act also lose "for each discarded Jack...", like growing ones
    local effect = strip_condition(joined, grows or R.category(joker) == "action")
    if effect == "" then effect = "Activates" end
    if to then effect = retype_text(effect, to, conversions) end
    local out = wrap(effect .. " when " .. trigger.text, 30)
    local mode = not own and on("behaviours") and BEHAVIOUR_LINES[joker.ability.rnd_behaviour]
    for _, line in ipairs(mode or {}) do out[#out + 1] = line end
    for _, note in ipairs(notes) do
        if to then note = retype_text(note, to, conversions) end
        out[#out + 1] = note
    end
    return out
end

local cache = {}

local function rewritten_key(key, trigger, joker)
    local descs = G.localization.descriptions.Joker
    local original = descs and descs[key]
    if not (original and original.text) then return nil end
    local own = R.category(joker) ~= "score"
    local to = not own and on("types") and joker.ability.rnd_type or "same"
    local mode = not own and on("behaviours") and joker.ability.rnd_behaviour or "normal"
    local new_key = "rnd_" .. key .. "__" .. trigger.key .. "__" .. to .. "__" .. mode
    if cache[new_key] and descs[new_key] then return new_key, cache[new_key] end

    local text
    local conversions = {}
    if type(original.text[1]) == "table" then
        -- Multi-box description: rewrite the first box, keep the others
        text = { rewrite_lines(original.text[1], trigger, joker, conversions) }
        for i = 2, #original.text do text[i] = original.text[i] end
    else
        text = rewrite_lines(original.text, trigger, joker, conversions)
    end

    local parsed = {}
    for i, line in ipairs(text) do
        if type(line) == "table" then
            parsed[i] = {}
            for j, sub in ipairs(line) do parsed[i][j] = loc_parse_string(sub) end
        else
            parsed[i] = loc_parse_string(line)
        end
    end
    descs[new_key] = { name = original.name, text = text, text_parsed = parsed, name_parsed = original.name_parsed }
    cache[new_key] = conversions
    return new_key, conversions
end


-- Clear the cache when the language changes (localization is rebuilt)
local init_localization_ref = init_localization
function init_localization(...)
    cache = {}
    return init_localization_ref(...)
end

local generate_card_ui_ref = generate_card_ui
function generate_card_ui(_c, full_UI_table, specific_vars, card_type, badges, hide_desc, main_start, main_end, card, ...)
    local ret = generate_card_ui_ref(_c, full_UI_table, specific_vars, card_type, badges, hide_desc, main_start, main_end, card, ...)
    local trigger = card and not full_UI_table and not hide_desc and _c and _c.set == "Joker" and _c == card.config.center
        and card.ability and card.ability.rnd_trigger and BY_KEY[card.ability.rnd_trigger]
    if not (trigger and ret and ret.main and R.shows_trigger(card)) then return ret end

    -- The same key and #vars# the Joker's own description uses
    local key, vars = _c.key, specific_vars
    if type(_c.loc_vars) == "function" then
        local ok, res = pcall(_c.loc_vars, _c, {}, card)
        if ok and type(res) == "table" then
            vars = res.vars or vars
            key = res.key or key
        end
    end
    local new_key, conversions = rewritten_key(key, trigger, card)
    if not new_key then return ret end

    -- Fill in the converted amounts and the behaviour's current value
    local all = {}
    for i, v in pairs(vars or {}) do all[i] = v end
    for _, c in ipairs(conversions or {}) do
        local v = all[c.from_var]
        all[c.slot] = type(v) == "number" and R.convert(c.from, v, card.ability.rnd_type) or v
    end
    local inc = R.increment(card)
    local kind = R.kind_of(card, inc)
    if card.ability.rnd_behaviour == "decays" then
        -- Real amounts: how much it drops by, and what it gives now
        local step = inc and round_effect(times(inc, 0.25))
        all[61] = R.format_increase(step, kind)
        all[60] = R.format_total(inc and round_effect(times(inc, card.ability.rnd_decay or 2)), kind)
    else
        all[61] = R.format_increase(inc, kind)
        all[60] = R.format_total(card.ability.rnd_total, kind)
    end
    vars = all

    -- Keep any extra rows other code put before/after the description (Misprint...)
    local first = main_start and ret.main[1] or nil
    local last = main_end and ret.main[#ret.main] or nil
    for i = #ret.main, 1, -1 do ret.main[i] = nil end
    if first then ret.main[1] = first end
    localize({ type = "descriptions", set = "Joker", key = new_key, nodes = ret.main, vars = vars or {} })
    if last then ret.main[#ret.main + 1] = last end
    return ret
end
