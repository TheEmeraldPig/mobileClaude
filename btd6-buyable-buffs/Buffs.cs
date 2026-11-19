using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using BTD_Mod_Helper.Extensions;
using Il2CppAssets.Scripts.Models;
using Il2CppAssets.Scripts.Unity;
using Il2CppNinjaKiwi.Localization;
using MelonLoader;

namespace BuyableBuffs;

/// <summary>
/// One buyable buff. Its levels follow an upgrade path in the game, so dropping it on the same tower again moves
/// it up: Alchemist Brew level 1 is Acidic Mixture Dip, level 4 is Permanent Brew.
/// </summary>
public sealed class BuffLine
{
    public string Id = "";
    public string Name = "";

    /// <summary>The upgrades (or hero levels) whose buff each level gives, in order.</summary>
    public string[] Upgrades = Array.Empty<string>();

    /// <summary>A Geraldo item instead of an upgrade (the item's behavior class name).</summary>
    public string GeraldoItem = "";

    /// <summary>For one-upgrade buffs that stack in the game (Shinobi Tactics): how many times it stacks.</summary>
    public int Repeat;

    /// <summary>Icon to use if the game's buff icon can't be found.</summary>
    public string FallbackIcon = "";

    internal List<BuffLevel>? levels;
}

public sealed class BuffLevel
{
    public string Name = "";
    public string BuffId = "";
    public string BuffLocs = "";
    public List<Fx> Effects = new();

    /// <summary>Description text, if it isn't just the list of effects.</summary>
    public string Text = "";
}

/// <summary>One stat change, with the text the description shows for it.</summary>
public sealed class Fx
{
    public string Text = "";
    public Action<Model> Apply = _ => { };
}

public static class Buffs
{
    /// <summary>Every level of every buff costs this much.</summary>
    public const int PricePerLevel = 1000;

    public static int Price(int level) => PricePerLevel * level;

    public static readonly List<BuffLine> Lines = new()
    {
        new() { Id = "VillageDrums", Name = "Village Drums", FallbackIcon = "BuffIconVillage2xx",
            Upgrades = new[] { "Jungle Drums", "Primary Training", "Primary Mentoring", "Primary Expertise" } },
        new() { Id = "VillageIntel", Name = "Village Intelligence", FallbackIcon = "BuffIconVillagex2x",
            Upgrades = new[] { "Radar Scanner", "Monkey Intelligence Bureau", "Call to Arms", "Homeland Defense" } },
        new() { Id = "AlchemistBrew", Name = "Alchemist Brew", FallbackIcon = "BuffIconAlchemist3xx",
            Upgrades = new[] { "Acidic Mixture Dip", "Berserker Brew", "Stronger Stimulant", "Permanent Brew" } },
        new() { Id = "Overclock", Name = "Overclock", FallbackIcon = "BuffIconEngineerx4x",
            Upgrades = new[] { "Overclock", "Ultraboost" } },
        new() { Id = "Shinobi", Name = "Shinobi Tactics", FallbackIcon = "BuffIconNinjax3x",
            Upgrades = new[] { "Shinobi Tactics" }, Repeat = 5 },
        new() { Id = "Energizer", Name = "Energizer", FallbackIcon = "BuffIconSub5xx",
            Upgrades = new[] { "Energizer" } },
        new() { Id = "TradeEmpire", Name = "Trade Empire", FallbackIcon = "BuffIconBuccaneerxx4",
            Upgrades = new[] { "Buccaneer-Favored Trades", "Buccaneer-Trade Empire" } },
        new() { Id = "EliteSniper", Name = "Elite Sniper", FallbackIcon = "BuffIconSniperx5x",
            Upgrades = new[] { "Elite Sniper" } },
        new() { Id = "BananaCentral", Name = "Banana Central", FallbackIcon = "BuffIconBananaFarm5xx",
            Upgrades = new[] { "Banana Central" } },
        new() { Id = "RallyingRoar", Name = "Rallying Roar", FallbackIcon = "BuffIconPatFusty",
            Upgrades = new[] { "3 PatFusty roar", "14 PatFusty rawr+" } },
        new() { Id = "HeatItUp", Name = "Heat It Up", FallbackIcon = "BuffIconGwendolin",
            Upgrades = new[] { "gwendolin heat it up", "gwendolin heat it up range", "gwendolin heat it up damage" } },
        new() { Id = "NaturesWard", Name = "Nature's Ward", FallbackIcon = "BuffIconObyn",
            Upgrades = new[]
            {
                "ObynGreenfoot-4-NaturesWard+", "ObynGreenfoot-8-NaturesWard++", "ObynGreenfoot-15-NaturesWard++"
            } },
        new() { Id = "BombExpert", Name = "Bomb Expert", FallbackIcon = "BuffIconStrikerJones",
            Upgrades = new[] { "Bomb Expert", "Artillery Command", "Artillery Command 2" } },
        new() { Id = "SharpeningStone", Name = "Sharpening Stone", FallbackIcon = "BuffIconSharpeningStone",
            GeraldoItem = "SharpeningStoneBehaviorModel" },
        new() { Id = "JarOfPickles", Name = "Jar of Pickles", FallbackIcon = "BuffIconJarOfPickles",
            GeraldoItem = "JarOfPicklesBehaviorModel" },
    };

    public static object? GameModel => R.Get(Game.instance, "model");

    /// <summary>The levels of a buff, read from the game the first time they're needed.</summary>
    public static List<BuffLevel> Levels(BuffLine line)
    {
        if (line.levels != null) return line.levels;
        if (GameModel == null) return new List<BuffLevel>();
        var levels = new List<BuffLevel>();
        try
        {
            if (line.GeraldoItem != "") levels.AddRange(GeraldoLevels(line));
            else
            {
                foreach (var upgrade in line.Upgrades)
                {
                    var tower = SourceTower(upgrade);
                    if (tower == null)
                    {
                        MelonLogger.Warning($"[Buyable Buffs] {line.Name}: no tower has the upgrade \"{upgrade}\"");
                        break;
                    }
                    var level = new BuffLevel { Name = Loc(upgrade, upgrade) };
                    foreach (var buff in BuffModels(tower)) AddEffects(level, buff);
                    if (level.Effects.Count == 0) continue;
                    levels.Add(level);
                }
                if (line.Repeat > 1 && levels.Count == 1)
                {
                    var one = levels[0];
                    for (var n = 2; n <= line.Repeat; n++)
                        levels.Add(new BuffLevel
                        {
                            Name = $"{one.Name} x{n}", BuffId = one.BuffId, BuffLocs = one.BuffLocs,
                            Effects = Enumerable.Repeat(one.Effects, n).SelectMany(e => e).ToList(),
                            Text = $"{string.Join(", ", one.Effects.Select(e => e.Text))}, {n} times over"
                        });
                }
            }
        }
        catch (Exception e)
        {
            MelonLogger.Warning($"[Buyable Buffs] couldn't read {line.Name} from the game: {e}");
        }

        line.levels = levels;
        MelonLogger.Msg($"[Buyable Buffs] {line.Name}: {levels.Count} levels");
        foreach (var l in levels)
            MelonLogger.Msg($"    {l.Name}: {string.Join(", ", l.Effects.Select(e => e.Text))}");
        return levels;
    }

    /// <summary>The cheapest tower that has this upgrade.</summary>
    private static object? SourceTower(string upgrade)
    {
        object? best = null;
        var bestRank = int.MaxValue;
        foreach (var tower in R.Items(R.Get(GameModel, "towers")))
        {
            if (R.Str(tower, "name").StartsWith("BuyableBuffs-")) continue;
            if (!R.Items(R.Get(tower, "appliedUpgrades")).Any(u => u.ToString() == upgrade)) continue;
            var rank = R.Get(tower, "tier", 0) * 100 + R.Items(R.Get(tower, "tiers")).Sum(Convert.ToInt32);
            if (R.Get(tower, "isParagon", false)) rank += 100_000;
            if (R.Get(tower, "isSubTower", false)) rank += 10_000;
            if (rank >= bestRank) continue;
            bestRank = rank;
            best = tower;
        }
        return best;
    }

    /// <summary>Every behavior on a tower that buffs other towers.</summary>
    private static IEnumerable<object> BuffModels(object tower)
    {
        foreach (var behavior in R.Items(R.Get(tower, "behaviors")))
        {
            var type = R.TypeName(behavior);
            if (IsSupport(behavior)) yield return behavior;
            else if (type == "AbilityModel")
            {
                foreach (var ab in R.Items(R.Get(behavior, "behaviors")))
                    if (R.Str(ab, "buffIconName") != "" || R.TypeName(ab).Contains("CallToArms") ||
                        R.TypeName(ab).Contains("Overclock"))
                        yield return ab;
            }
            else if (type == "AttackModel" && behavior is Model attack)
            {
                foreach (var d in attack.GetDescendants<Model>().ToSystemList())
                    if (R.TypeName(d).StartsWith("Add") && R.TypeName(d).EndsWith("ToProjectileModel"))
                        yield return d;
            }
        }
    }

    private static bool IsSupport(object behavior)
    {
        for (var t = R.Typed(behavior).GetType(); t != null; t = t.BaseType)
            if (t.Name == "SupportModel")
                return true;
        return false;
    }

    private static IEnumerable<BuffLevel> GeraldoLevels(BuffLine line)
    {
        foreach (var item in R.Items(R.Get(GameModel, "geraldoItemModels")))
        foreach (var bm in R.Items(R.Get(item, "behaviorModels")))
        {
            if (R.TypeName(bm) != line.GeraldoItem) continue;
            var level = new BuffLevel { Name = Loc(R.Str(item, "locsId"), line.Name) };
            AddEffects(level, bm);
            if (level.Effects.Count > 0) yield return level;
            yield break;
        }
    }

    // ------------------------------------------------------------------ turning game models into stat changes

    private static float F(object m, string field, float fallback = float.NaN) => R.Get(m, field, fallback);

    private static string Pct(float fraction) =>
        (fraction * 100).ToString(Math.Abs(fraction * 100 % 1) < 0.05 ? "0" : "0.#", CultureInfo.InvariantCulture) + "%";

    private static string Num(float v) => v.ToString(Math.Abs(v % 1) < 0.001 ? "0" : "0.##", CultureInfo.InvariantCulture);

    private static void AddEffects(BuffLevel level, object buff)
    {
        if (level.BuffId == "" && R.Str(buff, "buffIconName") is { Length: > 0 } icon)
        {
            level.BuffId = icon;
            level.BuffLocs = R.Str(buff, "buffLocsName");
        }

        var fx = level.Effects;
        var type = R.TypeName(buff);
        switch (type)
        {
            case "RateSupportModel":
            case "RateSupportExplosiveModel":
            case "RateSupportBombExpertModel":
                fx.AddRange(RateFactor(F(buff, "multiplier")));
                break;
            case "ActivateRateSupportZoneModel":
            case "OverclockModel":
            case "OverclockPermanentModel":
                fx.AddRange(RateFactor(F(buff, "rateModifier")));
                break;
            case "CallToArmsModel":
            {
                var m = F(buff, "multiplier");
                if (!float.IsNaN(m) && m > 0)
                {
                    fx.AddRange(RateFactor(1 / m));
                    fx.Add(PierceScale(m));
                }
                break;
            }
            case "SupportShinobiTacticsModel":
            {
                var m = F(buff, "multiplier");
                if (!float.IsNaN(m))
                {
                    var bonus = m < 1 ? 1 - m : m - 1;
                    fx.AddRange(RateFactor(1 / (1 + bonus)));
                    fx.Add(PierceScale(1 + bonus));
                }
                break;
            }
            case "RangeSupportModel":
            case "ActivateRangeSupportZoneModel":
            case "MakeshiftAreaRangeSupportModel":
                fx.AddRange(Range(F(buff, type == "MakeshiftAreaRangeSupportModel" ? "rangeMultiplier" : "multiplier", 0),
                    F(buff, "additive", 0)));
                break;
            case "PierceSupportModel":
                fx.Add(PierceAdd(F(buff, "pierce")));
                break;
            case "ActivatePierceSupportZoneModel":
                fx.Add(PierceAdd(F(buff, "pierceIncrease")));
                break;
            case "PiercePercentageSupportModel":
            {
                var p = F(buff, "percentIncrease");
                fx.Add(PierceScale(1 + (p > 1 ? p / 100 : p)));
                break;
            }
            case "DamageSupportModel":
                fx.Add(DamageAdd(F(buff, "increase")));
                break;
            case "ActivateTowerDamageSupportZoneModel":
                fx.Add(DamageAdd(F(buff, "damageIncrease")));
                break;
            case "SubCommanderSupportModel":
                fx.Add(DamageAdd(F(buff, "damageIncrease")));
                fx.Add(PierceAdd(F(buff, "pierceIncrease")));
                break;
            case "VisibilitySupportModel":
            case "ActivateVisibilitySupportZoneModel":
                fx.Add(Camo());
                break;
            case "DamageTypeSupportModel":
                fx.Add(PopTypes(R.Get(buff, "immuneBloonProperties", 0)));
                break;
            case "ProjectileSpeedSupportModel":
                fx.Add(Scale("projectile speed", F(buff, "multiplier"), "TravelStraitModel", "Speed"));
                break;
            case "ActivateProjectileSpeedSupportZoneModel":
                fx.Add(Scale("projectile speed", 1 + F(buff, "speedIncrease"), "TravelStraitModel", "Speed"));
                break;
            case "ProjectileRadiusSupportModel":
                fx.Add(Scale("projectile size", F(buff, "multiplier"), "ProjectileModel", "radius"));
                break;
            case "ProjectileLifespanSupportModel":
                fx.Add(Scale("projectile distance", F(buff, "multiplier"), "TravelStraitModel", "Lifespan"));
                break;
            case "AbilityCooldownScaleSupportModel":
            {
                var s = F(buff, "abilityCooldownSpeedScale");
                if (!float.IsNaN(s) && s > 0)
                {
                    var factor = s >= 1 ? 1 / s : s;
                    fx.Add(new Fx
                    {
                        Text = $"abilities recharge {Pct(1 / factor - 1)} faster",
                        Apply = m => Each(m, "AbilityModel", a => MulAny(a, factor, "Cooldown", "cooldown"))
                    });
                }
                break;
            }
            case "BananaCashIncreaseSupportModel":
                fx.Add(Income(F(buff, "multiplier")));
                break;
            case "MonkeyCityIncomeSupportModel":
                fx.Add(Income(F(buff, "incomeModifier")));
                break;
            case "AddBerserkerBrewToProjectileModel":
            {
                var dmg = F(buff, "damageUp", 0);
                var pierce = F(buff, "pierceUp", 0);
                var rate = F(buff, "rateUp", 0);
                var range = F(buff, "rangeUp", 0);
                if (dmg != 0) fx.Add(DamageAdd(dmg));
                if (pierce != 0) fx.Add(PierceAdd(pierce));
                if (rate != 0) fx.AddRange(RateFactor(1 / (1 + rate)));
                if (range != 0) fx.AddRange(Range(range, 0));
                fx.AddRange(AddedBehaviors(buff));
                break;
            }
            case "AddAcidicMixtureToProjectileModel":
                fx.AddRange(AddedBehaviors(buff));
                break;
            case "SharpeningStoneBehaviorModel":
                fx.Add(PierceAdd(F(buff, "pierceIncrease")));
                break;
            case "JarOfPicklesBehaviorModel":
            {
                fx.Add(DamageAdd(F(buff, "damageIncrease")));
                var scale = F(buff, "attackSpeedScale");
                if (!float.IsNaN(scale) && scale > 0 && Math.Abs(scale - 1) > 0.001f) fx.AddRange(RateFactor(scale));
                var fort = F(buff, "bonusFortDamage", 0);
                if (fort > 0) fx.Add(TagDamage("Fortified", "fortified bloons", fort));
                break;
            }
        }
        fx.RemoveAll(f => f.Text == "");
    }

    /// <summary>Weapon/projectile/tower behaviors a brew adds (acid that hurts MOABs and ceramics, lead popping...).</summary>
    private static IEnumerable<Fx> AddedBehaviors(object brew)
    {
        foreach (var (field, target) in new[]
                 {
                     ("projBehaviors", "ProjectileModel"), ("weapBehaviors", "WeaponModel"), ("towerBehaviors", "")
                 })
        {
            foreach (var b in R.Items(R.Get(brew, field)))
            {
                if (b is not Model behavior) continue;
                var text = DescribeAdded(behavior);
                if (target == "ProjectileModel" && R.TypeName(behavior) == "DamageModifierForTagModel")
                {
                    yield return new Fx
                    {
                        Text = text,
                        Apply = m => Each(m, "ProjectileModel", p =>
                        {
                            if (!HasDamage(p)) return;
                            R2.Append(p, "behaviors", behavior.Duplicate());
                            R.Set(p, "hasDamageModifiers", true);
                        })
                    };
                    continue;
                }
                yield return new Fx
                {
                    Text = text,
                    Apply = m =>
                    {
                        if (target == "") R2.Append(m, "behaviors", behavior.Duplicate());
                        else Each(m, target, t => R2.Append(t, "behaviors", behavior.Duplicate()));
                    }
                };
            }
        }
    }

    private static string DescribeAdded(Model behavior)
    {
        var type = R.TypeName(behavior);
        if (type == "DamageModifierForTagModel")
        {
            var tag = R.Str(behavior, "tag");
            var add = F(behavior, "damageAddative", 0);
            var mult = F(behavior, "damageMultiplier", 1);
            var parts = new List<string>();
            if (add != 0) parts.Add($"+{Num(add)} damage to {TagName(tag)}");
            if (Math.Abs(mult - 1) > 0.001f) parts.Add($"{Pct(mult - 1)} more damage to {TagName(tag)}");
            return string.Join(", ", parts);
        }
        if (type.Contains("Acid")) return "acid that eats away at bloons (including lead)";
        return "";
    }

    private static string TagName(string tag) => tag switch
    {
        "Moabs" => "MOAB-class bloons",
        "Ceramic" => "ceramic bloons",
        "Fortified" => "fortified bloons",
        "Lead" => "lead bloons",
        "Boss" => "bosses",
        _ => tag
    };

    // ------------------------------------------------------------------ effects on a tower model

    private static void Each(Model tower, string typeName, Action<Model> action)
    {
        if (typeName == "TowerModel")
        {
            action(tower);
            return;
        }
        foreach (var d in tower.GetDescendants<Model>().ToSystemList())
            if (R.TypeName(d) == typeName)
                action(d);
    }

    private static void MulAny(object o, float factor, params string[] fields)
    {
        foreach (var f in fields)
        {
            var v = R.Get(o, f, float.NaN);
            if (float.IsNaN(v)) continue;
            R.Set(o, f, v * factor);
            return;
        }
    }

    private static bool HasDamage(Model projectile) =>
        R.Items(R.Get(projectile, "behaviors")).Any(b => R.TypeName(b) == "DamageModel");

    /// <summary>rate is time between attacks, so a factor below 1 means faster attacks.</summary>
    private static IEnumerable<Fx> RateFactor(float factor)
    {
        if (float.IsNaN(factor) || factor <= 0 || Math.Abs(factor - 1) < 0.0001f) yield break;
        var speed = 1 / factor - 1;
        yield return new Fx
        {
            Text = speed > 0 ? $"+{Pct(speed)} attack speed" : $"{Pct(speed)} attack speed",
            Apply = m => Each(m, "WeaponModel", w => MulAny(w, factor, "Rate", "rate"))
        };
    }

    private static IEnumerable<Fx> Range(float multiplier, float additive)
    {
        if (float.IsNaN(multiplier)) multiplier = 0;
        if (float.IsNaN(additive)) additive = 0;
        // the game stores +10% as 0.1 (or occasionally 1.1)
        var fraction = multiplier > 1 ? multiplier - 1 : multiplier;
        if (fraction == 0 && additive == 0) yield break;
        var parts = new List<string>();
        if (fraction != 0) parts.Add($"+{Pct(fraction)} range");
        if (additive != 0) parts.Add($"+{Num(additive)} range");
        yield return new Fx
        {
            Text = string.Join(", ", parts),
            Apply = m =>
            {
                R.Set(m, "range", R.Get(m, "range", 0f) * (1 + fraction) + additive);
                Each(m, "AttackModel", a => R.Set(a, "range", R.Get(a, "range", 0f) * (1 + fraction) + additive));
            }
        };
    }

    private static Fx PierceAdd(float add) => float.IsNaN(add) || add == 0
        ? new Fx()
        : new Fx
        {
            Text = $"+{Num(add)} pierce",
            Apply = m => Each(m, "ProjectileModel", p =>
            {
                var pierce = R.Get(p, "pierce", 0f);
                if (pierce > 0) R.Set(p, "pierce", pierce + add);
            })
        };

    private static Fx PierceScale(float factor) => float.IsNaN(factor) || Math.Abs(factor - 1) < 0.0001f
        ? new Fx()
        : new Fx
        {
            Text = $"+{Pct(factor - 1)} pierce",
            Apply = m => Each(m, "ProjectileModel", p =>
            {
                var pierce = R.Get(p, "pierce", 0f);
                if (pierce > 0) R.Set(p, "pierce", MathF.Ceiling(pierce * factor));
            })
        };

    private static Fx DamageAdd(float add) => float.IsNaN(add) || add == 0
        ? new Fx()
        : new Fx
        {
            Text = $"+{Num(add)} damage",
            Apply = m => Each(m, "DamageModel", d =>
            {
                var damage = R.Get(d, "damage", 0f);
                if (damage > 0) R.Set(d, "damage", damage + add);
            })
        };

    private static Fx TagDamage(string tag, string label, float add) => new()
    {
        Text = $"+{Num(add)} damage to {label}",
        Apply = m => Each(m, "ProjectileModel", p =>
        {
            if (!HasDamage(p)) return;
            var modifier = R.New("Assets.Scripts.Models.Towers.Projectiles.Behaviors.DamageModifierForTagModel",
                new Dictionary<string, object?>
                {
                    ["name"] = "BuyableBuffs_" + tag, ["tag"] = tag, ["damageMultiplier"] = 1f,
                    ["damageAddative"] = add
                });
            if (modifier == null) return;
            R2.Append(p, "behaviors", modifier);
            R.Set(p, "hasDamageModifiers", true);
        })
    };

    private static Fx Camo() => new()
    {
        Text = "can see and pop camo bloons",
        Apply = m => Each(m, "FilterInvisibleModel", f => R.Set(f, "isActive", false))
    };

    private static Fx PopTypes(int immune) => new()
    {
        Text = immune == 0 ? "can pop every bloon type" : "can pop more bloon types",
        Apply = m => Each(m, "DamageModel", d =>
        {
            var current = R.Get(d, "immuneBloonProperties", 0);
            R.Set(d, "immuneBloonProperties", current & immune);
        })
    };

    private static Fx Scale(string what, float factor, string typeName, string field) =>
        float.IsNaN(factor) || factor <= 0 || Math.Abs(factor - 1) < 0.0001f
            ? new Fx()
            : new Fx
            {
                Text = $"{(factor > 1 ? "+" : "")}{Pct(factor - 1)} {what}",
                Apply = m => Each(m, typeName, x => MulAny(x, factor, field))
            };

    private static Fx Income(float m)
    {
        if (float.IsNaN(m) || m == 0) return new Fx();
        var factor = m < 1 ? 1 + m : m;
        return new Fx
        {
            Text = $"+{Pct(factor - 1)} income",
            Apply = t => Each(t, "CashModel", c =>
            {
                MulAny(c, factor, "minimum");
                MulAny(c, factor, "maximum");
            })
        };
    }

    // ------------------------------------------------------------------ text

    public static string Loc(string key, string fallback)
    {
        if (string.IsNullOrEmpty(key)) return fallback;
        try
        {
            var lm = LocalizationManager.Instance;
            if (lm == null) return fallback;
            foreach (var tableName in new[] { "textTable", "defaultTable" })
            {
                var table = R.Get(lm, tableName);
                if (table == null) continue;
                if (R.Call(table, "ContainsKey", key) is true &&
                    R.Call(table, "get_Item", key)?.ToString() is { Length: > 0 } text)
                    return text;
            }
        }
        catch
        {
            // fall through
        }
        return fallback;
    }

    public static string Describe(BuffLine line)
    {
        var levels = Levels(line);
        if (levels.Count == 0) return "";
        var lines = new List<string>
        {
            "Drop it on a tower. Dropping it on the same tower again raises it a level."
        };
        for (var i = 0; i < levels.Count; i++)
        {
            var l = levels[i];
            var text = l.Text != "" ? l.Text : string.Join(", ", l.Effects.Select(e => e.Text));
            lines.Add($"Lv {i + 1}: {l.Name} (${Price(i + 1):N0}): {text}");
        }
        return string.Join("\n", lines);
    }
}
