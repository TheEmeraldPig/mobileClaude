return {
    -- Jokers only get a random trigger: their numbers, type and scaling stay normal
    trigger_only = false,
    -- Each Joker's scoring numbers keep, grow, reset, stack or decay at random
    behaviours = true,
    -- Each Joker's scoring numbers become a random kind: Chips, Mult, XMult or XChips
    types = true,
    -- Each Joker's amounts are multiplied by a random number between min and max
    numbers = true, numbers_min = 0.5, numbers_max = 2,
    -- Shop prices
    prices = true, prices_min = 0.5, prices_max = 1.5,
    -- Blind score requirements
    blinds = true, blinds_min = 0.75, blinds_max = 1.5,
    -- Poker hands' base Chips and Mult, rolled at the start of each run
    hands = true, hands_min = 0.5, hands_max = 1.5,
}
