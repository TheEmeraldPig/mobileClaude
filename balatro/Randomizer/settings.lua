--- Config menu, in two tabs: "Jokers" and "Run". Each thing that's randomized has an
--- on/off toggle; each random range has a Min and a Max, each with a slider (0x-10x) and a
--- box you can type any number into (negative too). The slider and the box stay in sync.

local MOD = Randomizer.mod

-- key, label, default min, default max (no min/max: just a toggle)
local JOKER_ROWS = {
    { "triggers", "Random triggers" },
    { "types", "Random type (Chips, Mult, XMult, XChips)" },
    { "behaviours", "Random scaling (grows, resets, stacks, decays)" },
    { "numbers", "Joker values", 0.5, 2 },
}
-- Settings saved by 1.15 still have the old narrow Voucher/consumable ranges
if MOD.config and not MOD.config.rnd_wide_cards then
    MOD.config.rnd_wide_cards = true
    if MOD.config.vouchers_max == 1.5 then MOD.config.vouchers_max = 7 end
    if MOD.config.consumables_max == 1.5 then MOD.config.consumables_max = 4 end
end

local RUN_ROWS = {
    { "prices", "Shop prices", 0.5, 1.5 },
    { "vouchers", "Voucher values", 0.5, 7 },
    { "consumables", "Tarot, Planet & Spectral", 0.5, 4 },
    { "blinds", "Blind scores", 0.75, 1.5 },
    { "hands", "Poker hand values", 0.5, 1.5 },
}
-- The slider covers 0x to 10x; the box takes any number
local SLIDER_MIN, SLIDER_MAX = 0, 10

local function format(v) return string.format("%.2f", v) end
local function on_slider(v) return math.min(SLIDER_MAX, math.max(SLIDER_MIN, v)) end

-- Text typed into a box: what the box's ref holds, and what the slider shows
local inputs = {}

-- Put new text in a box (when the slider moved)
local function set_box(state, text)
    inputs[state.field] = text
    local letters = state.box and state.box.text and state.box.text.letters
    if not letters and G.OVERLAY_MENU then
        -- The box's letters, found through its first letter on screen
        local first = G.OVERLAY_MENU:get_UIE_by_ID(state.box.id .. "_letter_1")
        letters = first and first.config.ref_table
    end
    if letters then
        for i = 1, #letters do letters[i] = text:sub(i, i) end
        if state.box.text then state.box.text.current_position = #text end
    end
end

-- Runs every frame for each slider: moves the slider like normal, then keeps the slider,
-- the value and the box in sync
G.FUNCS.rnd_slider = function(e)
    G.FUNCS.slider(e)
    local bar = e.children[1]
    local rt = bar and bar.config.ref_table
    local state = rt and rt.rnd_state
    if not state then return end
    local value = rt.ref_table[rt.ref_value]
    local text = inputs[state.field]
    if text ~= state.last_text then
        -- Typed into the box
        local n = tonumber(text)
        if n then value = n end
        state.last_text = text
    elseif value ~= state.last_value then
        -- Dragged the slider
        value = math.floor(value * 100 + 0.5) / 100
        local s = format(value)
        set_box(state, s)
        state.last_text = s
    end
    rt.ref_table[rt.ref_value] = value
    state.last_value = value
    rt.text = format(value)
    bar.T.w = (on_slider(value) - rt.min) / (rt.max - rt.min) * rt.w
    bar.config.w = bar.T.w
end

-- Swap the slider's own per-frame function for the one above
local function use_sync(node)
    if type(node) ~= "table" then return end
    if node.config and node.config.func == "slider" then node.config.func = "rnd_slider" end
    for _, child in ipairs(node.nodes or {}) do use_sync(child) end
end

local function number_setting(field, label)
    local value = MOD.config[field]
    MOD.config[field] = type(value) == "number" and value or 1
    inputs[field] = format(MOD.config[field])
    local state = { field = field, last_text = inputs[field], last_value = MOD.config[field] }
    local slider_args = {
        label = label, label_scale = 0.3, text_scale = 0.25, w = 1.6, h = 0.25,
        ref_table = MOD.config, ref_value = field, min = SLIDER_MIN, max = SLIDER_MAX, decimal_places = 2,
        colour = G.C.RED, rnd_state = state,
    }
    -- The slider starts from a value on its own scale; the real value is put back after
    MOD.config[field] = on_slider(MOD.config[field])
    local slider = create_slider(slider_args)
    MOD.config[field] = state.last_value
    use_sync(slider)
    local box_args = {
        id = "rnd_box_" .. field, w = 1.1, h = 0.4, text_scale = 0.3, max_length = 8,
        prompt_text = format(MOD.config[field]), ref_table = inputs, ref_value = field,
        extended_corpus = true, colour = G.C.RED, hooked_colour = darken(G.C.RED, 0.3),
    }
    local box = create_text_input(box_args)
    state.box = box_args
    return { n = G.UIT.C, config = { align = "cm", padding = 0.05 }, nodes = {
        { n = G.UIT.R, config = { align = "cm" }, nodes = { slider } },
        { n = G.UIT.R, config = { align = "cm" }, nodes = { box } },
    } }
end

local function text_node(text, scale, colour)
    return { n = G.UIT.T, config = { text = text, scale = scale, colour = colour or G.C.UI.TEXT_LIGHT } }
end

local function heading(text)
    return { n = G.UIT.R, config = { align = "cm", padding = 0.08 }, nodes = { text_node(text, 0.42) } }
end

local function note(text)
    return { n = G.UIT.R, config = { align = "cm", padding = 0.02 }, nodes = { text_node(text, 0.27, G.C.UI.TEXT_INACTIVE) } }
end

local function row(entry)
    local key = entry[1]
    if MOD.config[key] == nil then MOD.config[key] = true end
    local toggle = create_toggle({ label = entry[2], ref_table = MOD.config, ref_value = key, label_scale = 0.3 })
    if not entry[3] then
        return { n = G.UIT.R, config = { align = "cm", padding = 0.02 }, nodes = { toggle } }
    end
    if MOD.config[key .. "_min"] == nil then MOD.config[key .. "_min"] = entry[3] end
    if MOD.config[key .. "_max"] == nil then MOD.config[key .. "_max"] = entry[4] end
    return { n = G.UIT.R, config = { align = "cm", padding = 0.03 }, nodes = {
        { n = G.UIT.C, config = { align = "cm", minw = 3.6 }, nodes = { toggle } },
        number_setting(key .. "_min", "Min"),
        number_setting(key .. "_max", "Max"),
    } }
end

local function page(title, rows, notes)
    local nodes = { heading(title) }
    for _, entry in ipairs(rows) do nodes[#nodes + 1] = row(entry) end
    for _, text in ipairs(notes) do nodes[#nodes + 1] = note(text) end
    return { n = G.UIT.ROOT, config = { align = "cm", padding = 0.15, colour = G.C.BLACK, r = 0.1, minw = 9 }, nodes = nodes }
end

local NOTES = {
    "Min and Max multiply the normal value (1 = normal; type any number, negative too).",
    "Values roll when a card first appears; poker hands roll at the start of a run.",
    "Tarots and Spectrals roll their numbers, like how many cards they select.",
}

MOD.config_tab = function()
    return page("Jokers", JOKER_ROWS, {
        "Random triggers off: every Joker uses its own trigger again.",
        NOTES[1],
    })
end

MOD.extra_tabs = function()
    return { {
        label = "Run",
        tab_definition_function = function() return page("Run", RUN_ROWS, NOTES) end,
    } }
end
