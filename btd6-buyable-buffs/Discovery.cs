using System;
using System.Collections.Generic;
using System.Linq;
using Il2CppAssets.Scripts.Models;
using Il2CppAssets.Scripts.Models.Towers;
using Il2CppAssets.Scripts.Unity;
using Il2CppNinjaKiwi.Localization;
using MelonLoader;

namespace BuyableBuffs;

/// <summary>Where a buff comes from in the game and how a carrier gives it out.</summary>
internal enum BuffKind
{
    /// <summary>An aura like Jungle Drums or Radar Scanner (any SupportModel): copied as is.</summary>
    Support,

    /// <summary>An ability that buffs towers around it, like Call to Arms or Rallying Roar.</summary>
    Ability,

    /// <summary>A thrown buff like Alchemist brews.</summary>
    Attack,

    /// <summary>A Geraldo shop item.</summary>
    Geraldo
}

internal sealed class BuffHit
{
    public BuffKind Kind;
    public object Owner = null!; // TowerModel, or GeraldoItemModel
    public object Container = null!; // the behavior to copy (support/ability/attack) or geraldo behavior
    public object Buff = null!; // the model that carries the buff icon
}

/// <summary>Everything the mod knows about one buyable buff.</summary>
internal sealed class BuffInfo
{
    public string IconName = "";
    public string BuffId = "";
    public BuffKind Kind;
    public TowerModel? SourceTower;
    public object? GeraldoItem;
    public List<BuffHit> Hits = new();
    public int Cost;

    // Text is looked up when it's shown, because the game's text isn't loaded yet when buffs are found.
    public Func<string> NameText = () => "";
    public Func<string> DescriptionText = () => "";
    public string Name => NameText();
    public string Description => DescriptionText();
    public bool Stacks;
    public int MaxStacks = 1;
    public List<string> MutatorIds = new();
}

/// <summary>
/// Goes through every tower (all upgrades and hero levels) and every Geraldo item in the game, and finds
/// the behaviors that show a given buff icon on the towers they buff.
/// </summary>
internal static class Discovery
{
    private static Dictionary<string, List<BuffHit>>? hitsByBuffId;
    private static Dictionary<string, List<string>>? buffIdsByGuid;
    private static readonly Dictionary<string, BuffInfo?> Infos = new();

    public static BuffInfo? Get(string iconName, string guid)
    {
        if (Infos.TryGetValue(iconName, out var info)) return info;
        if (Game.instance == null || R.Get(Game.instance, "model") == null) return null; // too early
        try
        {
            Scan();
            info = Build(iconName, guid);
        }
        catch (Exception e)
        {
            MelonLogger.Warning($"[Buyable Buffs] failed to look up {iconName}: {e}");
            info = null;
        }
        Infos[iconName] = info;
        if (info == null) MelonLogger.Msg($"[Buyable Buffs] {iconName}: no tower-to-tower buff uses this icon, skipped");
        else
            MelonLogger.Msg(
                $"[Buyable Buffs] {iconName}: \"{info.Name}\" from {info.SourceTower?.name ?? R.Str(info.GeraldoItem, "name")} ({info.Kind}, ${info.Cost}, stacks: {(info.Stacks ? info.MaxStacks.ToString() : "no")})");
        return info;
    }

    public static string BuffIdOf(object model) => R.Str(model, "buffIconName");

    private static void Scan()
    {
        if (hitsByBuffId != null) return;
        hitsByBuffId = new Dictionary<string, List<BuffHit>>();
        buffIdsByGuid = new Dictionary<string, List<string>>();
        var gameModel = R.Get(Game.instance, "model");

        // buff id -> sprite, to match our icon names to the ids the game uses
        var gameData = R.Get(FindStatic("Assets.Scripts.Data.GameData"), "buffIconSprites");
        foreach (var sprite in R.Items(R.Get(gameData, "buffIconSprites")))
        {
            var id = R.Str(sprite, "buffId");
            var guid = R.Str(R.Get(sprite, "icon"), "guidRef");
            if (id == "" || guid == "") continue;
            if (!buffIdsByGuid.TryGetValue(guid, out var list)) buffIdsByGuid[guid] = list = new List<string>();
            list.Add(id);
        }

        foreach (var tower in R.Items(R.Get(gameModel, "towers")))
        {
            if (R.Str(tower, "name").StartsWith(BuffTower.Prefix)) continue;
            foreach (var behavior in R.Items(R.Get(tower, "behaviors")))
            {
                var type = R.TypeName(behavior);
                if (IsSupport(behavior))
                {
                    Add(BuffKind.Support, tower, behavior, behavior);
                }
                else if (type == "AbilityModel")
                {
                    foreach (var ab in R.Items(R.Get(behavior, "behaviors")))
                        Add(BuffKind.Ability, tower, behavior, ab);
                }
                else if (type == "AttackModel")
                {
                    foreach (var d in Descendants(behavior))
                        if (R.TypeName(d).Contains("ToProjectile"))
                            Add(BuffKind.Attack, tower, behavior, d);
                }
            }
        }

        foreach (var item in R.Items(R.Get(gameModel, "geraldoItemModels")))
        foreach (var bm in R.Items(R.Get(item, "behaviorModels")))
            Add(BuffKind.Geraldo, item, bm, bm);
    }

    private static bool IsSupport(object behavior)
    {
        var t = R.Typed(behavior).GetType();
        for (var bt = t; bt != null; bt = bt.BaseType)
            if (bt.Name == "SupportModel")
                return true;
        return false;
    }

    private static IEnumerable<object> Descendants(object model)
    {
        if (model is not Model m) yield break;
        foreach (var d in R.Items(m.GetDescendants<Model>().ToSystemList())) yield return d;
    }

    private static object? FindStatic(string il2CppFullName)
    {
        var t = R.FindType(il2CppFullName);
        return t?.GetProperty("Instance")?.GetValue(null) ?? t?.GetProperty("instance")?.GetValue(null);
    }

    private static void Add(BuffKind kind, object owner, object container, object buff)
    {
        var id = BuffIdOf(buff);
        if (string.IsNullOrEmpty(id)) return;
        if (!hitsByBuffId!.TryGetValue(id, out var list)) hitsByBuffId[id] = list = new List<BuffHit>();
        list.Add(new BuffHit { Kind = kind, Owner = owner, Container = container, Buff = buff });
    }

    private static int Rank(BuffHit hit)
    {
        // prefer the earliest, plainest source: lowest tier, not a sub-tower, not a paragon
        if (hit.Kind == BuffKind.Geraldo) return 0;
        var rank = R.Get(hit.Owner, "tier", 0) * 10 + R.Get(hit.Owner, "tiers", (object?) null) switch
        {
            null => 0,
            var tiers => R.Items(tiers).Sum(t => Convert.ToInt32(t))
        };
        if (R.Get(hit.Owner, "isSubTower", false)) rank += 10_000;
        if (R.Get(hit.Owner, "isParagon", false)) rank += 20_000;
        return rank;
    }

    private static BuffInfo? Build(string iconName, string guid)
    {
        var ids = new List<string> { iconName };
        if (buffIdsByGuid!.TryGetValue(guid, out var more)) ids.AddRange(more);

        var hits = ids.Distinct().SelectMany(id => hitsByBuffId!.TryGetValue(id, out var l) ? l : new List<BuffHit>())
            .ToList();
        if (hits.Count == 0) return null;

        // all the behaviors for this buff on the single best source
        var best = hits.OrderBy(Rank).First();
        var fromBest = hits.Where(h => ReferenceEquals(h.Owner, best.Owner) || SamePointer(h.Owner, best.Owner))
            .GroupBy(h => PointerOf(h.Container)).Select(g => g.First()).ToList();

        var info = new BuffInfo
        {
            IconName = iconName,
            BuffId = BuffIdOf(best.Buff),
            Kind = best.Kind,
            Hits = fromBest,
        };

        if (best.Kind == BuffKind.Geraldo)
        {
            if (!Carrier.CanConvertGeraldo(fromBest)) return null;
            info.GeraldoItem = best.Owner;
            info.Cost = (int) Math.Round(R.Get(best.Owner, "cost", 0f));
            var locs = R.Str(best.Owner, "locsId");
            var buffLocs = R.Str(best.Buff, "buffLocsName");
            info.NameText = () => Loc(locs, Loc(buffLocs, iconName));
            info.DescriptionText = () => Loc(locs + " Description", "");
        }
        else
        {
            info.SourceTower = (best.Owner as TowerModel) ?? new TowerModel(((Il2CppInterop.Runtime.InteropTypes.Il2CppObjectBase) best.Owner).Pointer);
            FillTowerText(info, best);
        }

        // Stacking: the game's own rules
        foreach (var hit in fromBest)
        {
            var buff = hit.Buff;
            var maxStacks = Math.Max(R.Get(buff, "maxStacks", 0), R.Get(buff, "maxStackSize", 0));
            var unique = R.Get(buff, "isUnique", true);
            if (maxStacks > 1 || (!unique && hit.Kind == BuffKind.Support))
            {
                info.Stacks = true;
                info.MaxStacks = Math.Max(info.MaxStacks, maxStacks > 1 ? maxStacks : 99);
            }
            var mutatorId = R.Str(buff, "mutatorId");
            if (mutatorId == "") mutatorId = R.Str(buff, "mutationId");
            if (mutatorId != "") info.MutatorIds.Add(mutatorId);
        }
        if (!info.Stacks) info.MaxStacks = 1;

        if (info.Cost <= 0) info.Cost = 500;
        return info;
    }

    private static void FillTowerText(BuffInfo info, BuffHit best)
    {
        var tower = info.SourceTower!;
        var gameModel = R.Get(Game.instance, "model");
        var baseId = R.Str(tower, "baseId");
        var isHero = R.Items(R.Get(tower, "behaviors")).Any(b => R.TypeName(b) == "HeroModel");

        // the upgrade that added this buff = the highest tier upgrade the source has
        object? upgrade = null;
        foreach (var name in R.Items(R.Get(tower, "appliedUpgrades")))
        {
            var u = R.Call(gameModel, "GetUpgrade", name.ToString());
            if (u == null) continue;
            if (upgrade == null || R.Get(u, "tier", 0) > R.Get(upgrade, "tier", 0)) upgrade = u;
        }

        var buffLocs = R.Str(best.Buff, "buffLocsName");
        var level = R.Get(tower, "tier", 0);
        var upgradeName = upgrade != null ? R.Str(upgrade, "name") : "";
        if (isHero)
        {
            info.NameText = () =>
            {
                var hero = Loc(baseId, baseId);
                var buffName = Loc(buffLocs, "");
                return buffName != "" ? $"{buffName} ({hero})" : $"{hero} Level {level}";
            };
            info.Cost = (int) Math.Round(R.Get(tower, "cost", 0f));
            info.DescriptionText = () => Loc(upgradeName + " Description", Loc(baseId + " Description", ""));
        }
        else if (upgrade != null)
        {
            info.NameText = () => Loc(upgradeName, upgradeName);
            info.Cost = (int) Math.Round(R.Get(upgrade, "cost", 0f));
            info.DescriptionText = () => Loc(upgradeName + " Description", "");
        }
        else
        {
            info.NameText = () => Loc(buffLocs, "") is { Length: > 0 } n ? n : Loc(baseId, baseId);
            info.Cost = (int) Math.Round(R.Get(tower, "cost", 0f));
            info.DescriptionText = () => Loc(baseId + " Description", "");
        }
    }

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
                if (R.Call(table, "ContainsKey", key) is true)
                {
                    var text = R.Call(table, "get_Item", key)?.ToString();
                    if (!string.IsNullOrEmpty(text)) return text!;
                }
            }
        }
        catch
        {
            // fall through
        }
        return fallback;
    }

    private static IntPtr PointerOf(object o) =>
        o is Il2CppInterop.Runtime.InteropTypes.Il2CppObjectBase b ? b.Pointer : IntPtr.Zero;

    private static bool SamePointer(object a, object b) => PointerOf(a) != IntPtr.Zero && PointerOf(a) == PointerOf(b);
}
