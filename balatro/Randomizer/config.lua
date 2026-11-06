return {
    -- Each Joker gets a random trigger (off: Jokers work as normal)
    triggers = true,
    -- Each Joker's scoring numbers become a random kind: Chips, Mult, XMult or XChips
    types = true,
    -- Each Joker's scoring numbers keep, grow, reset, stack or decay at random
    behaviours = true,
    -- Each value is multiplied by a random number between min and max when it first appears
    numbers = true, numbers_min = 0.5, numbers_max = 2,
    prices = true, prices_min = 0.5, prices_max = 1.5,
    vouchers = true, vouchers_min = 0.5, vouchers_max = 1.5,
    consumables = true, consumables_min = 0.5, consumables_max = 1.5,
    blinds = true, blinds_min = 0.75, blinds_max = 1.5,
    hands = true, hands_min = 0.5, hands_max = 1.5,
}
