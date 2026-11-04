--- JokerDisplay support. JokerDisplay finds a Joker's display by its key, so while
--- JokerDisplay is working, every randomized Joker's key temporarily points at one shared
--- Randomizer display. That display shows what the Joker's next trigger gives, how many
--- times the current hand would trigger it, and its trigger.

local R = Randomizer

local function plain(text)
    return (text:gsub("{[^}]*}", ""))
end

local KIND_COLOUR = {
    chips = function() return G.C.CHIPS end,
    mult = function() return G.C.MULT end,
    xchips = function() return G.C.CHIPS end,
    xmult = function() return G.C.MULT end,
    dollars = function() return G.C.GOLD end,
}

-- How many times the selected (or played) hand would set off this trigger. nil = can't tell
local function trigger_count(trigger, joker)
    if trigger.key == "coin" then return nil end
    local text, poker_hands, scoring_hand = JokerDisplay.evaluate_hand()
    local full = (G.play and next(G.play.cards) and G.play.cards) or (G.hand and G.hand.highlighted) or {}
    if trigger.kind == "main" then
        if not next(full) then return nil end
        local ctx = { poker_hands = poker_hands, full_hand = full, scoring_hand = scoring_hand, scoring_name = text }
        local ok, hit = pcall(trigger.check, ctx, joker)
        return ok and (hit and 1 or 0) or nil
    elseif trigger.kind == "play" then
        if text == "Unknown" then return nil end
        local n = 0
        for _, c in pairs(scoring_hand or {}) do
            local ok, hit = pcall(trigger.check, { other_card = c, scoring_hand = scoring_hand }, joker)
            if ok and hit then n = n + JokerDisplay.calculate_card_triggers(c, scoring_hand) end
        end
        return n
    elseif trigger.kind == "hand" then
        local n = 0
        for _, c in ipairs(G.hand and G.hand.cards or {}) do
            if not c.highlighted then
                local ok, hit = pcall(trigger.check, { other_card = c }, joker)
                if ok and hit then n = n + JokerDisplay.calculate_card_triggers(c, nil, true) end
            end
        end
        return n
    end
end

local DEFINITION = {
    text = { { ref_table = "card.joker_display_values", ref_value = "rnd_text" } },
    reminder_text = {
        { text = "(" },
        { ref_table = "card.joker_display_values", ref_value = "rnd_label" },
        { text = ")" },
    },
    calc_function = function(card)
        local values = card.joker_display_values
        local trigger = R.trigger_of(card)
        values.rnd_label = trigger and plain(trigger.text) or ""
        local effect = R.next_effect(card)
        local count = trigger and trigger_count(trigger, card)
        values.rnd_kind = effect and R.kind_of(card, effect) or nil
        values.rnd_active = count == nil or count > 0
        if not effect then
            values.rnd_text = ""
        elseif count and count > 1 then
            values.rnd_text = count .. "x " .. R.format_total(effect, values.rnd_kind)
        else
            values.rnd_text = R.format_total(effect, values.rnd_kind)
        end
    end,
    style_function = function(card, text, reminder_text, extra)
        local values = card.joker_display_values
        if text and text.children and text.children[1] then
            local colour = values.rnd_kind and KIND_COLOUR[values.rnd_kind] and KIND_COLOUR[values.rnd_kind]()
            text.children[1].config.colour = values.rnd_active and colour or G.C.UI.TEXT_INACTIVE
        end
        return false
    end,
}

local function pack(...)
    return { n = select("#", ...), ... }
end

-- Point every randomized Joker's key at the Randomizer display while fn runs
local depth = 0
local function with_definitions(fn, ...)
    if depth > 0 or not (JokerDisplay and JokerDisplay.Definitions) then return fn(...) end
    local defs = JokerDisplay.Definitions
    local saved = {}
    for _, area in ipairs({ G.jokers, G.shop_jokers, G.pack_cards }) do
        for _, j in ipairs(area and area.cards or {}) do
            local key = j.config and j.config.center and j.config.center.key
            if key and j.ability and j.ability.rnd_trigger and not R.grows(j) and saved[key] == nil then
                saved[key] = defs[key] or false
                defs[key] = DEFINITION
            end
        end
    end
    depth = depth + 1
    local results = pack(pcall(fn, ...))
    depth = depth - 1
    for key, def in pairs(saved) do defs[key] = def or nil end
    if not results[1] then error(results[2], 0) end
    return unpack(results, 2, results.n)
end

local installed = false

local function install()
    if installed or not JokerDisplay then return end
    installed = true
    for _, name in ipairs({ "initialize_joker_display", "calculate_joker_display", "update_joker_display" }) do
        local ref = Card[name]
        if ref then
            Card[name] = function(...) return with_definitions(ref, ...) end
        end
    end
    for _, name in ipairs({ "calculate_card_triggers", "calculate_joker_triggers", "calculate_joker_modifiers",
        "calculate_blueprint_copy", "copy_display" }) do
        local ref = JokerDisplay[name]
        if ref then
            JokerDisplay[name] = function(...) return with_definitions(ref, ...) end
        end
    end
end

-- JokerDisplay usually loads before this mod, but install on the first frame to be sure
install()
local game_update_ref = Game.update
function Game:update(dt, ...)
    if not installed then install() end
    return game_update_ref(self, dt, ...)
end
