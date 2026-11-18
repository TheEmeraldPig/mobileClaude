using System.Collections.Generic;
using System.Linq;
using Il2Cpp;
using Il2CppAssets.Scripts.Models.Towers;
using static BuyableBuffs.Fx;

namespace BuyableBuffs;

public sealed class BuffDef
{
    public string Id = "";
    public string Name = "";
    public string Category = "";
    public double Cost;
    public Effect[] Effects = System.Array.Empty<Effect>();

    /// <summary>Buffs that are on/off (camo, popping types...) can only be bought once.</summary>
    public bool Single;

    /// <summary>Overrides the "max stacks" setting for this buff.</summary>
    public int? MaxStacks;

    public string Description => string.Join(", ", Effects.Select(e => e.Text));

    public void Apply(TowerModel model, int stacks)
    {
        foreach (var e in Effects) e.Apply(model, stacks);
    }
}

/// <summary>
/// The shop's catalogue. First every individual stat a buff can change, then a buyable version of every
/// buff that towers, heroes and powers give in the normal game (made from the same stat effects).
/// </summary>
public static class Buffs
{
    public const string Stats = "Stat Buffs";
    public const string Towers = "Tower Buffs";
    public const string Heroes = "Hero Buffs";
    public const string Powers = "Power & Other Buffs";

    public static readonly string[] Categories = { Stats, Towers, Heroes, Powers };

    public static readonly List<BuffDef> All = new()
    {
        // ================= every individual stat =================
        B("attack_speed", "Attack Speed", Stats, 400, AttackSpeed(10)),
        B("range", "Range", Stats, 300, Range(10)),
        B("damage", "Damage", Stats, 750, Damage(1)),
        B("pierce", "Pierce", Stats, 500, Pierce(2)),
        B("pierce_pct", "Pierce %", Stats, 600, PiercePct(15)),
        B("moab_damage", "MOAB Damage", Stats, 700, TagDamage("Moabs", "MOAB-class", 2)),
        B("ceramic_damage", "Ceramic Damage", Stats, 500, TagDamage("Ceramic", "ceramic", 2)),
        B("boss_damage", "Boss Damage", Stats, 900, TagDamage("Boss", "bosses", 5)),
        B("fortified_damage", "Fortified Damage", Stats, 600, TagDamage("Fortified", "fortified", 2)),
        B("lead_damage", "Lead Damage", Stats, 300, TagDamage("Lead", "lead", 2)),
        B("dot_damage", "Damage Over Time", Stats, 450, DamageOverTime(1)),
        B("projectile_speed", "Projectile Speed", Stats, 250, ProjectileSpeed(20)),
        B("projectile_distance", "Projectile Distance", Stats, 250, ProjectileLifespan(20)),
        B("projectile_size", "Projectile Size", Stats, 400, ProjectileSize(15)),
        B("ability_cooldown", "Ability Cooldown", Stats, 800, AbilityCooldown(10)),
        B("income", "Income", Stats, 900, Income(10)),
        S("camo", "Camo Detection", Stats, 400, Camo),
        S("pop_lead", "Pop Lead", Stats, 450, PopTypes(BloonProperties.Lead, "lead")),
        S("pop_black", "Pop Black", Stats, 300, PopTypes(BloonProperties.Black, "black")),
        S("pop_white", "Pop White", Stats, 300, PopTypes(BloonProperties.White, "white")),
        S("pop_frozen", "Pop Frozen", Stats, 300, PopTypes(BloonProperties.Frozen, "frozen")),
        S("pop_purple", "Pop Purple", Stats, 600, PopTypes(BloonProperties.Purple, "purple")),
        S("see_through_walls", "Ignore Line of Sight", Stats, 500, SeeThroughWalls),

        // ================= tower buffs =================
        B("village_range", "Village Range", Towers, 350, Range(10)),
        B("jungle_drums", "Jungle Drums", Towers, 1200, AttackSpeed(15)),
        B("primary_training", "Primary Training", Towers, 600, Pierce(1), Range(10),
            ProjectileSpeed(10)),
        B("primary_mentoring", "Primary Mentoring", Towers, 1500, Range(5), AbilityCooldown(15)),
        S("radar_scanner", "Radar Scanner", Towers, 1000, Camo),
        S("mib", "Monkey Intelligence Bureau", Towers, 4000, AllTypes),
        B("homeland_defense", "Homeland Defense", Towers, 12000, AttackSpeed(100)),
        B("call_to_arms", "Call To Arms", Towers, 6000, AttackSpeed(50), PiercePct(50)),
        B("acidic_mixture_dip", "Acidic Mixture Dip", Towers, 1000, TagDamage("Moabs", "MOAB-class", 1),
            TagDamage("Ceramic", "ceramic", 1), PopTypes(BloonProperties.Lead, "lead")),
        B("berserker_brew", "Berserker Brew", Towers, 1600, Damage(1), Pierce(2), AttackSpeed(10),
            Range(10)),
        B("stronger_stimulant", "Stronger Stimulant", Towers, 3200, Damage(1), Pierce(3), AttackSpeed(15),
            Range(15)),
        B("overclock", "Overclock", Towers, 5000, AttackSpeed(40)),
        B("ultraboost", "Ultraboost", Towers, 1800, AttackSpeed(6)).Max(10),
        B("shinobi_tactics", "Shinobi Tactics", Towers, 800, AttackSpeed(8), PiercePct(8)).Max(20),
        B("trade_empire", "Trade Empire", Towers, 2500, Income(20), Damage(1), Pierce(1)),
        B("banana_central", "Banana Central", Towers, 4000, Income(25)),
        B("spirit_of_the_forest", "Druidic Reach", Towers, 900, Range(15), Pierce(1)),
        B("embrittlement", "Embrittlement Expertise", Towers, 1400, TagDamage("Moabs", "MOAB-class", 1),
            PopTypes(BloonProperties.Lead | BloonProperties.Frozen, "lead and frozen")),

        // ================= hero buffs =================
        B("rallying_roar", "Rallying Roar (Pat Fusty)", Heroes, 1500, Damage(1)),
        B("heat_it_up", "Heat It Up (Gwendolin)", Heroes, 1200, Damage(1), Pierce(1)),
        B("natures_ward", "Nature's Ward (Obyn)", Heroes, 1100, Range(10), Pierce(1)),
        B("artillery_command", "Artillery Command (Striker Jones)", Heroes, 1400, AbilityCooldown(20),
            TagDamage("Fortified", "fortified", 1)),
        B("bank_hack", "Bank Hack (Benjamin)", Heroes, 1500, Income(12)),
        S("drone_uplink", "UAV Uplink (Etienne)", Heroes, 1200, Camo),
        B("blood_sacrifice", "Blood Sacrifice (Adora)", Heroes, 2000, AttackSpeed(10), Range(10)),
        B("naval_tactics", "Naval Tactics (Brickell)", Heroes, 3000, AttackSpeed(50)),
        B("sharpening_stone", "Sharpening Stone (Geraldo)", Heroes, 800, Pierce(2),
            TagDamage("Ceramic", "ceramic", 1)),
        B("jar_of_pickles", "Jar of Pickles (Geraldo)", Heroes, 900, Damage(1),
            TagDamage("Ceramic", "ceramic", 1)),
        S("see_through_glasses", "See-Through Glasses (Geraldo)", Heroes, 700, Camo),
        B("spirit_brew", "Spirit Brew (Ezili)", Heroes, 1300, DamageOverTime(2), Range(5)),

        // ================= powers / other =================
        B("monkey_boost", "Monkey Boost", Powers, 8000, AttackSpeed(100)).Max(3),
        B("paragon_power", "Paragon Power Totem", Powers, 10000, Damage(2), Pierce(5),
            TagDamage("Boss", "bosses", 10)),
        B("knowledge_sharpshooting", "Monkey Knowledge: Sharp Shots", Powers, 500, Pierce(1),
            ProjectileSpeed(10)),
        B("knowledge_bigger_bombs", "Monkey Knowledge: Bigger Bombs", Powers, 500, ProjectileSize(20)),
        B("knowledge_quick_hands", "Monkey Knowledge: Quick Hands", Powers, 700, AttackSpeed(5),
            AbilityCooldown(5)),
    };

    public static readonly Dictionary<string, BuffDef> ById = All.ToDictionary(b => b.Id);

    private static BuffDef B(string id, string name, string category, double cost, params Effect[] fx) =>
        new() { Id = id, Name = name, Category = category, Cost = cost, Effects = fx };

    private static BuffDef S(string id, string name, string category, double cost, params Effect[] fx) =>
        new() { Id = id, Name = name, Category = category, Cost = cost, Effects = fx, Single = true };

    private static BuffDef Max(this BuffDef def, int max)
    {
        def.MaxStacks = max;
        return def;
    }
}
