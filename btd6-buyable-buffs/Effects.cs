using System;
using BTD_Mod_Helper.Extensions;
using Il2Cpp;
using Il2CppAssets.Scripts.Models.Bloons.Behaviors;
using Il2CppAssets.Scripts.Models.Towers;
using Il2CppAssets.Scripts.Models.Towers.Behaviors;
using Il2CppAssets.Scripts.Models.Towers.Behaviors.Abilities;
using Il2CppAssets.Scripts.Models.Towers.Behaviors.Attack;
using Il2CppAssets.Scripts.Models.Towers.Filters;
using Il2CppAssets.Scripts.Models.Towers.Projectiles;
using Il2CppAssets.Scripts.Models.Towers.Projectiles.Behaviors;
using Il2CppAssets.Scripts.Models.Towers.Weapons;

namespace BuyableBuffs;

/// <summary>
/// One stat change. A buff is a list of these. <see cref="Apply"/> gets the number of stacks bought,
/// and is always run on a fresh copy of the tower's clean model, so stacks never get applied twice.
/// </summary>
public sealed record Effect(string Text, Action<TowerModel, int> Apply);

/// <summary>The building blocks every buff is made from.</summary>
public static class Fx
{
    // ---- attack speed / range ------------------------------------------------------------

    /// <summary>+pct% attack speed per stack (compounds, like stacking Jungle Drums with Overclock).</summary>
    public static Effect AttackSpeed(float pct) => new($"+{pct:0.#}% attack speed", (m, n) =>
    {
        var mult = MathF.Pow(1 + pct / 100f, n);
        foreach (var w in m.GetDescendants<WeaponModel>()) w.Rate /= mult;
    });

    public static Effect Range(float pct) => new($"+{pct:0.#}% range", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        m.range *= mult;
        foreach (var a in m.GetDescendants<AttackModel>()) a.range *= mult;
    });

    // ---- damage / pierce ----------------------------------------------------------------

    public static Effect Damage(float add) => new($"+{add:0.#} damage", (m, n) =>
    {
        foreach (var d in m.GetDescendants<DamageModel>())
            if (d.damage > 0) d.damage += add * n;
    });

    public static Effect Pierce(float add) => new($"+{add:0.#} pierce", (m, n) =>
    {
        foreach (var p in m.GetDescendants<ProjectileModel>())
        {
            if (p.pierce <= 0) continue;
            p.pierce += add * n;
            if (p.maxPierce > 0) p.maxPierce += add * n;
        }
    });

    public static Effect PiercePct(float pct) => new($"+{pct:0.#}% pierce", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var p in m.GetDescendants<ProjectileModel>())
        {
            if (p.pierce <= 0) continue;
            p.pierce = MathF.Ceiling(p.pierce * mult);
            if (p.maxPierce > 0) p.maxPierce = MathF.Ceiling(p.maxPierce * mult);
        }
    });

    /// <summary>Bonus damage against bloons with a tag ("Moabs", "Ceramic", "Boss", "Fortified", "Lead"...).</summary>
    public static Effect TagDamage(string tag, string label, float add) => new($"+{add:0.#} damage to {label}",
        (m, n) =>
        {
            foreach (var p in m.GetDescendants<ProjectileModel>())
            {
                if (!p.HasBehavior<DamageModel>()) continue;
                p.AddBehavior(new DamageModifierForTagModel($"BuyableBuffs_{tag}_{label}", tag, 1f, add * n,
                    false, false));
                p.hasDamageModifiers = true;
            }
        });

    public static Effect DamageOverTime(float add) => new($"+{add:0.#} burn/poison damage", (m, n) =>
    {
        foreach (var d in m.GetDescendants<DamageOverTimeModel>()) d.damage += add * n;
    });

    // ---- detection / damage types ---------------------------------------------------------

    public static readonly Effect Camo = new("Camo detection", (m, _) =>
    {
        foreach (var f in m.GetDescendants<FilterInvisibleModel>()) f.isActive = false;
    });

    public static Effect PopTypes(BloonProperties types, string label) => new($"Can pop {label}", (m, _) =>
    {
        foreach (var d in m.GetDescendants<DamageModel>()) d.immuneBloonProperties &= ~types;
    });

    public static readonly Effect AllTypes = PopTypes(
        BloonProperties.Lead | BloonProperties.Black | BloonProperties.White | BloonProperties.Purple |
        BloonProperties.Frozen, "every bloon type");

    public static readonly Effect SeeThroughWalls = new("Shoots through walls/obstacles", (m, _) =>
    {
        foreach (var a in m.GetDescendants<AttackModel>()) a.attackThroughWalls = true;
        foreach (var p in m.GetDescendants<ProjectileModel>()) p.ignoreBlockers = true;
    });

    // ---- projectiles --------------------------------------------------------------------

    public static Effect ProjectileSpeed(float pct) => new($"+{pct:0.#}% projectile speed", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var t in m.GetDescendants<TravelStraitModel>()) t.Speed *= mult;
    });

    public static Effect ProjectileLifespan(float pct) => new($"+{pct:0.#}% projectile distance", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var t in m.GetDescendants<TravelStraitModel>()) t.Lifespan *= mult;
    });

    public static Effect ProjectileSize(float pct) => new($"+{pct:0.#}% projectile size", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var p in m.GetDescendants<ProjectileModel>()) p.radius *= mult;
    });

    // ---- abilities / income -------------------------------------------------------------

    public static Effect AbilityCooldown(float pct) => new($"-{pct:0.#}% ability cooldown", (m, n) =>
    {
        var mult = MathF.Pow(1 - pct / 100f, n);
        foreach (var a in m.GetDescendants<AbilityModel>()) a.Cooldown *= mult;
    });

    public static Effect Income(float pct) => new($"+{pct:0.#}% income", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var c in m.GetDescendants<CashModel>())
        {
            c.minimum *= mult;
            c.maximum *= mult;
        }
        foreach (var c in m.GetDescendants<PerRoundCashBonusTowerModel>()) c.cashPerRound *= mult;
    });
}
