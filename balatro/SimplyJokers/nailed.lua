--- Nailed: a Tarot card that nails a playing card or Joker in place. A Nailed card
--- can't be debuffed and can't be flipped face down.

SMODS.Atlas { key = "nailed", path = "nailed.png", px = 71, py = 95 }

local NAILED = "sj_nailed"

SMODS.Sticker {
    key = "nailed",
    loc_txt = {
        name = "Nailed",
        text = {
            "Can't be {C:red}debuffed{}",
            "or flipped face down",
        },
        label = "Nailed",
    },
    atlas = "nailed",
    pos = { x = 1, y = 0 },
    badge_colour = HEX("8A8F99"),
    rate = 0,
    -- Only ever added by the Nailed card, never rolled onto shop cards
    needs_enable_flag = true,
    sets = { Joker = true, Default = true, Enhanced = true },
}

local function is_nailed(card)
    return card and card.ability and card.ability[NAILED]
end

-- Can't be debuffed: Steamodded asks each mod before debuffing a card
SMODS.current_mod.set_debuff = function(card)
    if is_nailed(card) then return "prevent_debuff" end
end

local function in_view(card)
    return card.area and (card.area == G.hand or card.area == G.play or card.area == G.jokers)
end

-- Can't be flipped face down while in hand, played or among the Jokers
local flip_ref = Card.flip
function Card:flip(...)
    if is_nailed(self) and self.facing == "front" and in_view(self) then return end
    return flip_ref(self, ...)
end

-- Boss Blinds like The House draw cards face down without flipping them; turn those up
local update_ref = Card.update
function Card:update(dt)
    update_ref(self, dt)
    if is_nailed(self) and self.facing == "back" and in_view(self) and not self.flipping then
        flip_ref(self)
    end
end

local function nail_target()
    local hand = G.hand and G.hand.highlighted or {}
    local jokers = G.jokers and G.jokers.highlighted or {}
    if #hand + #jokers ~= 1 then return end
    local target = hand[1] or jokers[1]
    if is_nailed(target) then return end
    return target
end

SMODS.Consumable {
    key = "nailed",
    set = "Tarot",
    loc_txt = {
        name = "Nailed",
        text = {
            "{C:attention}Nail{} 1 selected playing",
            "card or Joker: it can't be",
            "{C:red}debuffed{} or flipped face down",
        },
    },
    cost = 3,
    atlas = "nailed",
    pos = { x = 0, y = 0 },
    loc_vars = function(self, info_queue, card)
        info_queue[#info_queue + 1] = { set = "Other", key = NAILED }
    end,
    can_use = function(self, card)
        return nail_target() ~= nil
    end,
    use = function(self, card, area, copier)
        local target = nail_target()
        G.E_MANAGER:add_event(Event({
            trigger = "after",
            delay = 0.3,
            func = function()
                play_sound("tarot1")
                SMODS.Stickers[NAILED]:apply(target, true)
                -- Lift any debuff it already has
                if target.debuff then
                    if SMODS.recalc_debuff then SMODS.recalc_debuff(target) else target:set_debuff(false) end
                end
                target:juice_up(0.3, 0.5)
                if target.area == G.hand then G.hand:unhighlight_all() end
                if target.area == G.jokers then G.jokers:unhighlight_all() end
                return true
            end,
        }))
    end,
}
