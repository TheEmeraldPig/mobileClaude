--- Randomizer: randomizes Joker triggers, Joker numbers, shop prices, Blind scores and
--- poker hand values. Each part can be turned off in the mod's Config tab.

Randomizer = { mod = SMODS.current_mod }

assert(SMODS.load_file("triggers.lua"))()
assert(SMODS.load_file("rolls.lua"))()
assert(SMODS.load_file("settings.lua"))()
-- JokerDisplay support (only does anything when JokerDisplay is installed)
assert(SMODS.load_file("joker_display.lua"))()

-- Written to the log so it's easy to confirm which version is running
local function log(msg) if sendInfoMessage then sendInfoMessage(msg, "Randomizer") end end
Randomizer.log = log
log(("Randomizer %s loaded (Steamodded %s)"):format(tostring(SMODS.current_mod.version), tostring(SMODS.version or MODDED_VERSION)))
