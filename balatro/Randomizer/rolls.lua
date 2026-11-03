--- Everything else that gets randomized: Joker numbers, shop prices, Blind scores and
--- poker hand values. All rolls use the run's seed, so a seeded run rolls the same way.

local R = Randomizer
local MOD = R.mod

-- "Only randomize triggers" leaves a Joker's numbers, type and scaling alone; shop prices,
-- Blind scores and poker hands still follow their own settings
local JOKER_OPTIONS = { numbers = true, types = true, behaviours = true }
local function on(option)
    local config = MOD.config or {}
    if config.trigger_only and JOKER_OPTIONS[option] then return false end
    return config[option]
end

-- Talisman (Amulet) turns big numbers into objects; these work on both kinds
local function plain(v)
    if type(v) == "table" and to_number then return to_number(v) end
    return v
end

-- v * factor, rounded, keeping v's kind (plain number or Talisman number)
local function scaled_round(v, factor)
    if type(v) == "number" then return math.floor(v * factor + 0.5) end
    local ok, out = pcall(function()
        local r = v * factor
        if type(r) == "table" and r.floor then r = r:floor() end
        return r
    end)
    return ok and out or v
end

local function between(seed, low, high)
    return low + pseudorandom(seed) * (high - low)
end

----------------------------------------------------------------------
-- Rolling Jokers the first time they show up in a run
----------------------------------------------------------------------

-- Effect numbers worth scaling; anything else (odds, counters, hand size...) is left alone
local SCALE_HINTS = { "mult", "chip", "dollar", "money" }
local SKIP_HINTS = { "odd", "every", "remaining", "size", "limit", "count", "slot" }

local function is_effect_key(key)
    local lk = tostring(key):lower()
    for _, hint in ipairs(SKIP_HINTS) do
        if lk:find(hint) then return false end
    end
    for _, hint in ipairs(SCALE_HINTS) do
        if lk:find(hint) then return true end
    end
    return false
end

local function is_multiplier_key(key)
    local lk = tostring(key):lower()
    return lk:find("^x") ~= nil or lk:find("_x") ~= nil or lk == "x_mult"
end

local function scale(value, key, factor)
    value = plain(value)
    if type(value) ~= "number" or value == 0 then return value end
    if is_multiplier_key(key) then
        if value <= 1 then return value end
        -- Scale the part above X1, to one decimal place
        return math.floor((1 + (value - 1) * factor) * 10 + 0.5) / 10
    end
    local scaled = math.floor(value * factor + 0.5)
    if value > 0 then return math.max(1, scaled) end
    return scaled
end

function R.roll_numbers(joker)
    local ability = joker.ability
    local factor = between("rnd_numbers", 0.5, 2)
    ability.rnd_factor = factor
    for _, key in ipairs({ "mult", "t_mult", "t_chips" }) do
        if type(ability[key]) == "number" then ability[key] = scale(ability[key], key, factor) end
    end
    if type(ability.x_mult) == "number" and ability.x_mult > 1 then
        ability.x_mult = scale(ability.x_mult, "x_mult", factor)
    end
    if type(ability.extra) == "table" then
        for k, v in pairs(ability.extra) do
            if type(v) == "number" and is_effect_key(k) then ability.extra[k] = scale(v, k, factor) end
        end
    end
end

local ROLL_AREAS = function(card)
    return card.area and (card.area == G.jokers or card.area == G.shop_jokers or card.area == G.pack_cards)
end

local announced = false
local update_ref = Card.update
function Card:update(dt)
    if not announced and G.STAGE == G.STAGES.RUN and R.log then
        announced = true
        R.log("Run detected, Jokers will roll triggers as they appear")
    end
    update_ref(self, dt)
    if not (self.ability and self.ability.set == "Joker" and G.GAME and G.STAGE == G.STAGES.RUN and ROLL_AREAS(self)) then
        return
    end
    -- Numbers are rolled once
    if not self.ability.rnd_rolled then
        self.ability.rnd_rolled = true
        if on("numbers") then R.roll_numbers(self) end
    end
    -- Every Joker gets a random trigger. One that has none yet (passive, or a check that
    -- failed) tries again once a round, so it can still pick one up later
    local round = G.GAME.round or 0
    if not self.ability.rnd_trigger and self.ability.rnd_trigger_tried ~= round then
        self.ability.rnd_trigger_tried = round
        R.roll_trigger(self)
        if not self.ability.rnd_trigger then R.note_passive(self) end
    end
end

----------------------------------------------------------------------
-- Shop prices: each card gets its own 50%-150% price factor
----------------------------------------------------------------------

local set_cost_ref = Card.set_cost
function Card:set_cost(...)
    local ret = set_cost_ref(self, ...)
    if on("prices") and G.GAME and G.STAGE == G.STAGES.RUN and self.ability and (self.cost or 0) > 0 then
        self.ability.rnd_price = self.ability.rnd_price or between("rnd_price", 0.5, 1.5)
        local cost = plain(self.cost)
        if type(cost) == "number" then self.cost = math.max(1, math.floor(cost * self.ability.rnd_price + 0.5)) end
    end
    return ret
end

----------------------------------------------------------------------
-- Blind scores: each Blind of each Ante gets its own 75%-150% factor
----------------------------------------------------------------------

local blind_type = nil

local function blind_factor(ante, kind)
    G.GAME.rnd_blind_factors = G.GAME.rnd_blind_factors or {}
    local key = tostring(ante) .. "_" .. kind
    if not G.GAME.rnd_blind_factors[key] then
        G.GAME.rnd_blind_factors[key] = between("rnd_blind" .. key, 0.75, 1.5)
    end
    return G.GAME.rnd_blind_factors[key]
end

local get_blind_amount_ref = get_blind_amount
function get_blind_amount(ante, ...)
    local amount = get_blind_amount_ref(ante, ...)
    if blind_type and on("blinds") and G.GAME then
        amount = scaled_round(amount, blind_factor(ante, blind_type))
    end
    return amount
end

-- The blind select screen and the blind itself both ask get_blind_amount; tell it which Blind
local blind_choice_ref = create_UIBox_blind_choice
function create_UIBox_blind_choice(kind, ...)
    local previous = blind_type
    blind_type = kind
    local ok, a, b = pcall(blind_choice_ref, kind, ...)
    blind_type = previous
    if not ok then error(a, 0) end
    return a, b
end

local set_blind_ref = Blind.set_blind
function Blind:set_blind(blind, reset, silent, ...)
    local previous = blind_type
    blind_type = G.GAME and G.GAME.blind_on_deck or nil
    local ok, a, b = pcall(set_blind_ref, self, blind, reset, silent, ...)
    blind_type = previous
    if not ok then error(a, 0) end
    return a, b
end

----------------------------------------------------------------------
-- Poker hand values: rolled once at the start of a new run
----------------------------------------------------------------------

local start_run_ref = Game.start_run
function Game:start_run(args, ...)
    local ret = start_run_ref(self, args, ...)
    local loading = args and args.savetext
    if on("hands") and not loading and G.GAME and G.GAME.hands then
        for name, hand in pairs(G.GAME.hands) do
            local chips = between("rnd_hand_chips" .. name, 0.5, 1.5)
            local mult = between("rnd_hand_mult" .. name, 0.5, 1.5)
            hand.s_chips = scaled_round(hand.s_chips, chips)
            hand.s_mult = scaled_round(hand.s_mult, mult)
            local ok = pcall(function()
                local level = plain(hand.level) or 1
                hand.chips = hand.s_chips + (hand.l_chips or 0) * (level - 1)
                hand.mult = hand.s_mult + (hand.l_mult or 0) * (level - 1)
            end)
            if not ok then hand.chips, hand.mult = hand.s_chips, hand.s_mult end
        end
    end
    return ret
end
