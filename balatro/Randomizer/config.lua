return {
    -- Jokers only get a random trigger: their numbers, type and scaling stay normal
    trigger_only = false,
    -- Each Joker's scoring numbers keep, grow, reset, stack or decay at random
    behaviours = true,
    -- Each Joker's scoring numbers become a random kind: Chips, Mult, XMult or XChips
    types = true,
    -- Each Joker's amounts are rolled between half and double
    numbers = true,
    -- Shop prices are 50%-150% of normal
    prices = true,
    -- Blind score requirements are 75%-150% of normal
    blinds = true,
    -- Poker hands' base Chips and Mult are rolled at the start of each run
    hands = true,
}
