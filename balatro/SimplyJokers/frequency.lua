--- Settings slider: makes every SimplyJokers item (Jokers, Tarots, Fusion cards and
--- Fusion Packs) up to 20 times more common.

local MOD = SMODS.current_mod

local function boost()
    return math.max(1, math.floor((MOD.config and MOD.config.rarity_boost) or 1))
end

local function is_ours(center)
    return center and (center.original_mod == MOD or center.mod == MOD)
end

-- Cards are picked evenly from a pool list, so adding extra copies of our keys makes
-- them that many times more likely within their pool (a Joker's rarity, the Tarots...)
local get_current_pool_ref = get_current_pool
function get_current_pool(...)
    local pool, pool_key = get_current_pool_ref(...)
    local n = boost()
    if n > 1 and type(pool) == "table" then
        local extra = {}
        for _, key in ipairs(pool) do
            if key ~= "UNAVAILABLE" and is_ours(G.P_CENTERS[key]) then
                for _ = 2, n do extra[#extra + 1] = key end
            end
        end
        for _, key in ipairs(extra) do pool[#pool + 1] = key end
    end
    return pool, pool_key
end

-- The Fusion category's share of shop slots scales too (normally 4, like Tarots)
local FUSION_BASE_RATE = 4
local create_card_for_shop_ref = create_card_for_shop
function create_card_for_shop(...)
    if G.GAME then G.GAME.fusion_rate = FUSION_BASE_RATE * boost() end
    return create_card_for_shop_ref(...)
end

-- Fusion Packs are weighted against the other packs
for _, key in ipairs({ "p_sj_fusion_normal", "p_sj_fusion_jumbo", "p_sj_fusion_mega" }) do
    local center = SMODS.Centers[key]
    if center then
        center.get_weight = function(self) return (self.weight or 1) * boost() end
    end
end

MOD.config_tab = function()
    return {
        n = G.UIT.ROOT,
        config = { align = "cm", padding = 0.2, colour = G.C.BLACK, r = 0.1, minw = 7 },
        nodes = {
            { n = G.UIT.R, config = { align = "cm" }, nodes = {
                { n = G.UIT.T, config = { text = "Item frequency", scale = 0.5, colour = G.C.UI.TEXT_LIGHT } },
            } },
            { n = G.UIT.R, config = { align = "cm" }, nodes = {
                create_slider({
                    label = "Times more common",
                    w = 5, h = 0.4,
                    ref_table = MOD.config, ref_value = "rarity_boost",
                    min = 1, max = 20, decimal_places = 0,
                }),
            } },
            { n = G.UIT.R, config = { align = "cm" }, nodes = {
                { n = G.UIT.T, config = { text = "Jokers, Tarots, Fusion cards and Fusion Packs", scale = 0.32, colour = G.C.UI.TEXT_INACTIVE } },
            } },
            { n = G.UIT.R, config = { align = "cm" }, nodes = {
                { n = G.UIT.T, config = { text = "1 = normal", scale = 0.32, colour = G.C.UI.TEXT_INACTIVE } },
            } },
        },
    }
end
