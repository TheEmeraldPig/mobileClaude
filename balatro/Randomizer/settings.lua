--- Config tab: on/off toggles, and for each random range a Min and a Max, each with a
--- slider and a box you can type a number into. The slider and the box stay in sync.

local MOD = Randomizer.mod

local TOGGLES = {
    { "trigger_only", "Only randomize triggers" },
    { "behaviours", "Random scaling" },
    { "types", "Random type (Chips, Mult, XMult, XChips)" },
}

-- key, label, default min, default max
local RANGES = {
    { "numbers", "Joker amounts", 0.5, 2 },
    { "prices", "Shop prices", 0.5, 1.5 },
    { "blinds", "Blind scores", 0.75, 1.5 },
    { "hands", "Poker hands", 0.5, 1.5 },
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

MOD.config_tab = function()
    local rows = {
        { n = G.UIT.R, config = { align = "cm", padding = 0.05 }, nodes = {
            text_node("Randomize (Joker triggers are always random)", 0.4),
        } },
    }
    local toggles = {}
    for _, option in ipairs(TOGGLES) do
        toggles[#toggles + 1] = { n = G.UIT.C, config = { align = "cm" }, nodes = {
            create_toggle({ label = option[2], ref_table = MOD.config, ref_value = option[1], label_scale = 0.3 }),
        } }
    end
    rows[#rows + 1] = { n = G.UIT.R, config = { align = "cm" }, nodes = toggles }
    for _, range in ipairs(RANGES) do
        local key = range[1]
        if MOD.config[key .. "_min"] == nil then MOD.config[key .. "_min"] = range[3] end
        if MOD.config[key .. "_max"] == nil then MOD.config[key .. "_max"] = range[4] end
        rows[#rows + 1] = { n = G.UIT.R, config = { align = "cm", padding = 0.03 }, nodes = {
            { n = G.UIT.C, config = { align = "cm", minw = 3.2 }, nodes = {
                create_toggle({ label = range[2], ref_table = MOD.config, ref_value = key, label_scale = 0.3 }),
            } },
            number_setting(key .. "_min", "Min"),
            number_setting(key .. "_max", "Max"),
        } }
    end
    rows[#rows + 1] = { n = G.UIT.R, config = { align = "cm", padding = 0.05 }, nodes = {
        text_node("Min and Max multiply the normal value (1 = normal; type any number). Rolls happen as things appear;", 0.27, G.C.UI.TEXT_INACTIVE),
    } }
    rows[#rows + 1] = { n = G.UIT.R, config = { align = "cm" }, nodes = {
        text_node("poker hands roll at the start of a run", 0.27, G.C.UI.TEXT_INACTIVE),
    } }
    return { n = G.UIT.ROOT, config = { align = "cm", padding = 0.15, colour = G.C.BLACK, r = 0.1, minw = 8 }, nodes = rows }
end
