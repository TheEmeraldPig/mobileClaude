--- Config tab: an on/off toggle for each thing the mod randomizes.

local MOD = Randomizer.mod

local OPTIONS = {
    { "trigger_only", "Only randomize triggers (Joker numbers, type and scaling stay normal)" },
    { "behaviours", "Random scaling (grows, resets, stacks, decays)" },
    { "types", "Random type (Chips, Mult, XMult, XChips)" },
    { "numbers", "Random amount (x0.5 - x2)" },
    { "prices", "Random shop prices (50% - 150%)" },
    { "blinds", "Random Blind scores (75% - 150%)" },
    { "hands", "Random poker hand values (50% - 150%)" },
}

MOD.config_tab = function()
    local rows = {
        { n = G.UIT.R, config = { align = "cm", padding = 0.05 }, nodes = {
            { n = G.UIT.T, config = { text = "Randomize (Joker triggers are always random)", scale = 0.45, colour = G.C.UI.TEXT_LIGHT } },
        } },
    }
    for _, option in ipairs(OPTIONS) do
        rows[#rows + 1] = { n = G.UIT.R, config = { align = "cm" }, nodes = {
            create_toggle({ label = option[2], ref_table = MOD.config, ref_value = option[1] }),
        } }
    end
    rows[#rows + 1] = { n = G.UIT.R, config = { align = "cm", padding = 0.05 }, nodes = {
        { n = G.UIT.T, config = { text = "Rolls happen as things appear; poker hands roll at the start of a run",
            scale = 0.3, colour = G.C.UI.TEXT_INACTIVE } },
    } }
    return { n = G.UIT.ROOT, config = { align = "cm", padding = 0.2, colour = G.C.BLACK, r = 0.1, minw = 8 }, nodes = rows }
end
