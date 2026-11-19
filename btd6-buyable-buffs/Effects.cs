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

internal static class Il2CppListExt
{
    /// <summary>Copies an Il2Cpp IEnumerable (what GetDescendants returns) into a normal C# list,
    /// because C# foreach can't walk Il2Cpp enumerators directly.</summary>
    public static System.Collections.Generic.List<T> ToSystemList<T>(
        this Il2CppSystem.Collections.Generic.IEnumerable<T> source)
    {
        var list = new System.Collections.Generic.List<T>();
        var enumerator = source.GetEnumerator();
        var untyped = enumerator.Cast<Il2CppSystem.Collections.IEnumerator>();
        while (untyped.MoveNext()) list.Add(enumerator.Current);
        return list;
    }
}

/// <summary>The building blocks every buff is made from.</summary>
public static class Fx
{
    // ---- attack speed / range ------------------------------------------------------------

    /// <summary>+pct% attack speed per stack (compounds, like stacking Jungle Drums with Overclock).</summary>
    public static Effect AttackSpeed(float pct) => new($"+{pct:0.#}% attack speed", (m, n) =>
    {
        var mult = MathF.Pow(1 + pct / 100f, n);
        foreach (var w in m.GetDescendants<WeaponModel>().ToSystemList()) w.Rate /= mult;
    });

    public static Effect Range(float pct) => new($"+{pct:0.#}% range", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        m.range *= mult;
        foreach (var a in m.GetDescendants<AttackModel>().ToSystemList()) a.range *= mult;
    });

    // ---- damage / pierce ----------------------------------------------------------------

    public static Effect Damage(float add) => new($"+{add:0.#} damage", (m, n) =>
    {
        foreach (var d in m.GetDescendants<DamageModel>().ToSystemList())
            if (d.damage > 0) d.damage += add * n;
    });

    public static Effect Pierce(float add) => new($"+{add:0.#} pierce", (m, n) =>
    {
        foreach (var p in m.GetDescendants<ProjectileModel>().ToSystemList())
        {
            if (p.pierce <= 0) continue;
            p.pierce += add * n;
            if (p.maxPierce > 0) p.maxPierce += add * n;
        }
    });

    public static Effect PiercePct(float pct) => new($"+{pct:0.#}% pierce", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var p in m.GetDescendants<ProjectileModel>().ToSystemList())
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
            foreach (var p in m.GetDescendants<ProjectileModel>().ToSystemList())
            {
                if (!p.HasBehavior<DamageModel>()) continue;
                p.AddBehavior(new DamageModifierForTagModel($"BuyableBuffs_{tag}_{label}", tag, 1f, add * n,
                    false, false));
                p.hasDamageModifiers = true;
            }
        });

    public static Effect DamageOverTime(float add) => new($"+{add:0.#} burn/poison damage", (m, n) =>
    {
        foreach (var d in m.GetDescendants<DamageOverTimeModel>().ToSystemList()) d.damage += add * n;
    });

    // ---- detection / damage types ---------------------------------------------------------

    public static readonly Effect Camo = new("Camo detection", (m, _) =>
    {
        foreach (var f in m.GetDescendants<FilterInvisibleModel>().ToSystemList()) f.isActive = false;
    });

    public static Effect PopTypes(BloonProperties types, string label) => new($"Can pop {label}", (m, _) =>
    {
        foreach (var d in m.GetDescendants<DamageModel>().ToSystemList()) d.immuneBloonProperties &= ~types;
    });

    public static readonly Effect AllTypes = PopTypes(
        BloonProperties.Lead | BloonProperties.Black | BloonProperties.White | BloonProperties.Purple |
        BloonProperties.Frozen, "every bloon type");

    public static readonly Effect SeeThroughWalls = new("Shoots through walls/obstacles", (m, _) =>
    {
        foreach (var a in m.GetDescendants<AttackModel>().ToSystemList()) a.attackThroughWalls = true;
        foreach (var p in m.GetDescendants<ProjectileModel>().ToSystemList()) p.ignoreBlockers = true;
    });

    // ---- projectiles --------------------------------------------------------------------

    public static Effect ProjectileSpeed(float pct) => new($"+{pct:0.#}% projectile speed", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var t in m.GetDescendants<TravelStraitModel>().ToSystemList()) t.Speed *= mult;
    });

    public static Effect ProjectileLifespan(float pct) => new($"+{pct:0.#}% projectile distance", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var t in m.GetDescendants<TravelStraitModel>().ToSystemList()) t.Lifespan *= mult;
    });

    public static Effect ProjectileSize(float pct) => new($"+{pct:0.#}% projectile size", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var p in m.GetDescendants<ProjectileModel>().ToSystemList()) p.radius *= mult;
    });

    // ---- abilities / income -------------------------------------------------------------

    public static Effect AbilityCooldown(float pct) => new($"-{pct:0.#}% ability cooldown", (m, n) =>
    {
        var mult = MathF.Pow(1 - pct / 100f, n);
        foreach (var a in m.GetDescendants<AbilityModel>().ToSystemList()) a.Cooldown *= mult;
    });

    public static Effect Income(float pct) => new($"+{pct:0.#}% income", (m, n) =>
    {
        var mult = 1 + pct / 100f * n;
        foreach (var c in m.GetDescendants<CashModel>().ToSystemList())
        {
            c.minimum *= mult;
            c.maximum *= mult;
        }
        foreach (var c in m.GetDescendants<PerRoundCashBonusTowerModel>().ToSystemList()) c.cashPerRound *= mult;
    });
}
