--- Everything else that gets randomized: Joker numbers, shop prices, Blind scores and
--- poker hand values. All rolls use the run's seed, so a seeded run rolls the same way.

local R = Randomizer
local MOD = R.mod

-- Each option is on unless it was turned off in the Config tab
local function on(option)
    return (MOD.config or {})[option] ~= false
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

-- The Min and Max set for an option in the Config tab (either way round)
local function range(option, low, high)
    local config = MOD.config or {}
    local a = tonumber(config[option .. "_min"]) or low
    local b = tonumber(config[option .. "_max"]) or high
    if a > b then a, b = b, a end
    return a, b
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
    -- Any factor works, negative too (+4 Mult with -1 becomes -4 Mult)
    return math.floor(value * factor + 0.5)
end

-- A single number as a card's extra: what its description says it is. "+#1# Mult",
-- "X#1# Mult", "$#1#" are amounts; "#1# in #2#" odds are left alone
local function bare_extra_kind(card)
    local text = R.desc_text and R.desc_text(card.config.center.key, card.config.center.set) or ""
    if text:find("#1#%s*in%s*#") then return nil end
    if text:find("X%s*#1#") then return "x_mult" end
    if text:find("[%+%-]%s*#1#") or text:find("%$%s*#1#") or text:find("#1#%s*Mult") or text:find("#1#%s*Chips") then return "extra" end
end

function R.roll_numbers(joker)
    local ability = joker.ability
    local factor = between("rnd_numbers", range("numbers", 0.5, 2))
    ability.rnd_factor = factor
    for _, key in ipairs({ "mult", "t_mult", "t_chips" }) do
        if type(plain(ability[key])) == "number" then ability[key] = scale(ability[key], key, factor) end
    end
    if type(plain(ability.x_mult)) == "number" and plain(ability.x_mult) > 1 then
        ability.x_mult = scale(ability.x_mult, "x_mult", factor)
    end
    if type(ability.extra) == "table" then
        for k, v in pairs(ability.extra) do
            if type(plain(v)) == "number" and is_effect_key(k) then ability.extra[k] = scale(v, k, factor) end
        end
    elseif type(plain(ability.extra)) == "number" then
        -- Fibonacci's +8 Mult, Acrobat's X3, Golden Joker's $4...
        local kind = bare_extra_kind(joker)
        if kind then ability.extra = scale(ability.extra, kind, factor) end
    end
end

----------------------------------------------------------------------
-- Vouchers and consumables (Tarot, Spectral): their numbers are rolled once
----------------------------------------------------------------------

-- Whole numbers stay whole (slots, hands), others get two decimals
local function scale_plain(v, factor)
    v = plain(v)
    if type(v) ~= "number" or v == 0 then return v end
    if v == math.floor(v) then return math.floor(v * factor + 0.5) end
    return math.floor(v * factor * 100 + 0.5) / 100
end

-- Numbers that must stay at least 1 (cards to select, the "4" in "1 in 4")
local AT_LEAST_ONE = { max_highlighted = true, min_highlighted = true, odds = true }

local function copy_config(t)
    if type(t) ~= "table" then return t end
    local out = {}
    for k, v in pairs(t) do out[k] = copy_config(v) end
    return out
end

local function keys_of(list, field)
    local out = {}
    for _, v in ipairs(list or {}) do
        local k = field and type(v) == "table" and v[field] or v
        if type(k) == "string" then out[#out + 1] = k end
    end
    return out
end

-- What a card can turn into: enhancements, suits, seals, poker hands
local function enhancements()
    local list = keys_of(G.P_CENTER_POOLS and G.P_CENTER_POOLS.Enhanced, "key")
    return #list > 0 and list or { "m_bonus", "m_mult", "m_wild", "m_glass", "m_steel", "m_stone", "m_gold", "m_lucky" }
end
local function suits()
    local list = keys_of(SMODS.Suit and SMODS.Suit.obj_buffer)
    return #list > 0 and list or { "Spades", "Hearts", "Clubs", "Diamonds" }
end
local function seals()
    local list = keys_of(SMODS.Seal and SMODS.Seal.obj_buffer)
    return #list > 0 and list or { "Gold", "Red", "Blue", "Purple" }
end
local function is_seal(v)
    for _, s in ipairs(seals()) do if s == v then return true end end
    return false
end

-- Every number is multiplied by the rolled factor (whole numbers stay whole)
local function scale_all(t, factor)
    for k, v in pairs(t) do
        if type(v) == "table" and not getmetatable(v) then
            scale_all(v, factor)
        elseif type(plain(v)) == "number" then
            local n = scale_plain(v, factor)
            if AT_LEAST_ONE[k] and plain(v) >= 1 and n < 1 then n = 1 end
            t[k] = n
        end
    end
end

-- A Voucher's or consumable's whole setup rolled: every number (amounts, slots, cards to
-- select, odds), and what it turns cards into (enhancement, suit, seal) or levels up
local function roll_card_numbers(card, option, seed)
    local a = card.ability
    local center = card.config.center
    local factor = between(seed, range(option, 0.5, 4))
    a.rnd_factor = factor
    local cfg = copy_config(center.config or {})
    scale_all(cfg, factor)
    if type(cfg.mod_conv) == "string" and cfg.mod_conv:sub(1, 2) == "m_" then
        cfg.mod_conv = pseudorandom_element(enhancements(), pseudoseed("rnd_enhance"))
    end
    if type(cfg.suit_conv) == "string" then
        cfg.suit_conv = pseudorandom_element(suits(), pseudoseed("rnd_suit"))
    end
    if type(cfg.extra) == "string" and is_seal(cfg.extra) then
        cfg.extra = pseudorandom_element(seals(), pseudoseed("rnd_seal"))
    end
    if type(cfg.hand_type) == "string" and G.handlist and #G.handlist > 0 then
        cfg.hand_type = pseudorandom_element(G.handlist, pseudoseed("rnd_planet"))
    end
    a.rnd_config = cfg
    -- The card's own copies: its extra, and (consumables) the setup the game reads
    a.extra = copy_config(cfg.extra)
    if a.consumeable ~= nil then a.consumeable = cfg end
end

local CONSUMABLE_SETS = { Tarot = true, Spectral = true, Planet = true }

-- While a rolled card is shown or used, its center's numbers are the rolled ones, so the
-- text and the base game's code (which read the center) match what was rolled
function R.with_rolled_center(card, fn, ...)
    local center = card and card.config and card.config.center
    local a = card and card.ability
    local rolled = a and a.rnd_card_rolled and (a.rnd_config or (a.extra ~= nil and { extra = a.extra }))
    if not (center and center.config and rolled) then return fn(...) end
    local saved = center.config
    if a.rnd_config then
        center.config = a.rnd_config
    else
        -- Rolled by an older version: only its extra was rolled
        center.config = copy_config(saved)
        center.config.extra = a.extra
    end
    local results = { n = 0 }
    local function keep(...) results = { n = select("#", ...), ... } end
    keep(pcall(fn, ...))
    center.config = saved
    if not results[1] then error(results[2], 0) end
    return unpack(results, 2, results.n)
end

local use_consumeable_ref = Card.use_consumeable
function Card:use_consumeable(...)
    if self.ability and self.ability.rnd_card_rolled then
        return R.with_rolled_center(self, use_consumeable_ref, self, ...)
    end
    return use_consumeable_ref(self, ...)
end

-- Vouchers whose "+1" is written into the base game's code instead of read from the card:
-- the rest of the rolled amount is added after the Voucher applies
local FIXED_ONE = {
    v_overstock_norm = function(n) change_shop_size(n) end,
    v_overstock_plus = function(n) change_shop_size(n) end,
    v_crystal_ball = function(n) G.consumeables.config.card_limit = G.consumeables.config.card_limit + n end,
    v_antimatter = function(n) G.jokers.config.card_limit = G.jokers.config.card_limit + n end,
    v_paint_brush = function(n) G.hand:change_size(n) end,
    v_palette = function(n) G.hand:change_size(n) end,
}

local apply_to_run_ref = Card.apply_to_run
function Card:apply_to_run(center, ...)
    if self and self.ability and self.ability.rnd_card_rolled then
        local ret = R.with_rolled_center(self, apply_to_run_ref, self, center, ...)
        local fix = FIXED_ONE[self.config.center.key]
        local factor = self.ability.rnd_factor
        if fix and type(factor) == "number" then
            local extra = math.floor(factor + 0.5) - 1
            if extra ~= 0 then
                G.E_MANAGER:add_event(Event({ func = function() pcall(fix, extra) return true end }))
            end
        end
        return ret
    end
    return apply_to_run_ref(self, center, ...)
end

-- The base game works out a card's description numbers here, before the description is
-- built, so the rolled numbers are swapped in for this too
local function replace_plus_one(node, amount, seen)
    if type(node) ~= "table" or seen[node] then return end
    seen[node] = true
    if node.config and type(node.config.text) == "string" then
        node.config.text = node.config.text:gsub("%+1%f[%D]", (amount < 0 and "" or "+") .. tostring(amount))
    end
    for _, child in ipairs(node) do replace_plus_one(child, amount, seen) end
    for _, child in ipairs(node.nodes or {}) do replace_plus_one(child, amount, seen) end
end

local ability_table_ref = Card.generate_UIBox_ability_table
function Card:generate_UIBox_ability_table(...)
    if self.ability and self.ability.rnd_card_rolled then
        local ret = R.with_rolled_center(self, ability_table_ref, self, ...)
        -- Those Vouchers' text says "+1" outright; show the rolled amount
        local factor = self.ability.rnd_factor
        if FIXED_ONE[self.config.center.key] and type(factor) == "number" and type(ret) == "table" then
            pcall(replace_plus_one, ret.main, math.floor(factor + 0.5), {})
        end
        return ret
    end
    return ability_table_ref(self, ...)
end

local function roll_other_cards(card)
    local a = card.ability
    if a.rnd_card_rolled or not G.GAME or G.STAGE ~= G.STAGES.RUN then return end
    if a.set == "Voucher" then
        a.rnd_card_rolled = true
        if on("vouchers") then roll_card_numbers(card, "vouchers", "rnd_voucher") end
    elseif CONSUMABLE_SETS[a.set] then
        a.rnd_card_rolled = true
        if on("consumables") then roll_card_numbers(card, "consumables", "rnd_consumable") end
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
    if self.ability and self.ability.set ~= "Joker" then
        roll_other_cards(self)
        return
    end
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
    -- A trigger from an older version that no longer exists gets rolled again
    if self.ability.rnd_trigger and not R.trigger_of(self) then
        self.ability.rnd_trigger, self.ability.rnd_trigger_tried = nil, nil
    end
    -- Legendary Jokers from before they had two triggers get their second one
    if self.ability.rnd_trigger and not self.ability.rnd_trigger2 and R.is_legendary(self) then R.roll_second(self) end
    if not self.ability.rnd_trigger and self.ability.rnd_trigger_tried ~= round then
        self.ability.rnd_trigger_tried = round
        R.roll_trigger(self)
        if not self.ability.rnd_trigger then
            R.note_passive(self)
        elseif self.joker_display_values and self.update_joker_display then
            -- JokerDisplay already built this Joker's display from its own trigger; rebuild it
            pcall(self.update_joker_display, self, true, true)
        end
    end
end

----------------------------------------------------------------------
-- Shop prices: each card gets its own 50%-150% price factor
----------------------------------------------------------------------

local set_cost_ref = Card.set_cost
function Card:set_cost(...)
    local ret = set_cost_ref(self, ...)
    if on("prices") and G.GAME and G.STAGE == G.STAGES.RUN and self.ability and (self.cost or 0) > 0 then
        self.ability.rnd_price = self.ability.rnd_price or between("rnd_price", range("prices", 0.5, 1.5))
        local cost = plain(self.cost)
        if type(cost) == "number" then
            self.cost = math.floor(cost * self.ability.rnd_price + 0.5)
            -- A free or negative price (the shop pays you) also sells for that, so it
            -- can't be bought and sold for money forever
            if self.cost < 1 then
                self.sell_cost = math.floor(self.cost / 2) + (self.ability.extra_value or 0)
                self.sell_cost_label = self.facing == "back" and "?" or self.sell_cost
            end
        end
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
        G.GAME.rnd_blind_factors[key] = between("rnd_blind" .. key, range("blinds", 0.75, 1.5))
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
            local chips = between("rnd_hand_chips" .. name, range("hands", 0.5, 1.5))
            local mult = between("rnd_hand_mult" .. name, range("hands", 0.5, 1.5))
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
