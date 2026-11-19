using System;
using System.Collections.Generic;
using System.Linq;
using BTD_Mod_Helper.Extensions;
using Il2CppAssets.Scripts.Models;
using Il2CppAssets.Scripts.Models.Towers;
using Il2CppAssets.Scripts.Simulation.Towers;
using Il2CppAssets.Scripts.Unity;
using Il2CppAssets.Scripts.Unity.UI_New.InGame;
using Il2CppInterop.Runtime.InteropTypes.Arrays;
using MelonLoader;
using UnityEngine;

namespace BuyableBuffs;

/// <summary>
/// A bought buff is a hidden, unselectable "carrier" tower that sits on the tower it was dropped on and holds the
/// game's own buff behavior (the same one Jungle Drums, a brew, an ability... uses), shrunk so it only reaches that
/// one tower. The game itself applies the buff, so the effect, the buff icon on the tower and the stacking rules
/// are the real ones; the carrier just never goes away.
/// </summary>
internal static class Carrier
{
    /// <summary>How far a carrier's buff reaches. Small enough to only touch the tower it sits on.</summary>
    public const float Reach = 5f;

    private const string Ns = "Assets.Scripts.Models.Towers.Behaviors.";

    private static readonly Dictionary<IntPtr, Tower> TargetOf = new();
    private static readonly List<Tower> Pending = new();
    private static readonly List<(Tower carrier, string message)> ToRefund = new();
    private static readonly List<Tower> ToSell = new();
    private static float nextUpkeep;

    // ------------------------------------------------------------------ building the carrier model

    public static TowerModel BuildModel(BuffInfo? info, string id)
    {
        var gameModel = R.Get(Game.instance, "model");
        var source = info?.SourceTower ?? R.Call(gameModel, "GetTowerFromId", "DartMonkey") as TowerModel;
        if (source == null) throw new Exception("couldn't find a tower model to base the buff on");

        var model = source.Duplicate();
        var behaviors = R.Items(R.Get(model, "behaviors"))
            .Where(b => R.TypeName(b) is "DisplayModel" or "CreateSoundOnTowerPlaceModel")
            .Cast<Model>()
            .ToList();

        if (info != null)
        {
            foreach (var hit in info.Hits)
            {
                switch (hit.Kind)
                {
                    case BuffKind.Support:
                        behaviors.Add(PrepSupport(((Model) hit.Container).Duplicate()));
                        break;
                    case BuffKind.Ability:
                        if (R.TypeName(hit.Buff).Contains("Overclock"))
                            behaviors.AddRange(ConvertOverclock(hit.Buff, info));
                        else
                            behaviors.Add(PrepAbility(((Model) hit.Container).Duplicate()));
                        break;
                    case BuffKind.Attack:
                        var attack = ((Model) hit.Container).Duplicate();
                        R.Set(attack, "range", Reach);
                        behaviors.Add(attack);
                        break;
                    case BuffKind.Geraldo:
                        behaviors.AddRange(ConvertGeraldo(hit.Buff, info));
                        break;
                }
            }
        }

        R.Set(model, "behaviors", new Il2CppReferenceArray<Model>(behaviors.Distinct().ToArray()));
        R.Set(model, "range", Reach);
        R.Set(model, "ignoreTowerForSelection", true);
        R.Set(model, "footprint", R.New(Ns + "CircleFootprintModel", new Dictionary<string, object?>
        {
            ["name"] = "BuyableBuffsFootprint",
            ["radius"] = 0f,
            ["doesntBlockTowerPlacement"] = true,
            ["ignoresPlacementCheck"] = true,
            ["ignoresTowerOverlap"] = true
        }));
        return model;
    }

    private static Model PrepSupport(Model support)
    {
        R.Set(support, "isGlobal", false);
        R.Set(support, "appliesToOwningTower", false);
        R.Set(support, "isCustomRadius", false);
        R.Set(support, "onlyAffectParagon", false);
        R.Set(support, "showBuffIcon", true);
        R.SetEmptyArray(support, "filters");
        return support;
    }

    private static Model PrepAbility(Model ability)
    {
        R.Set(ability, "isHidden", true);
        R.Set(ability, "cooldown", 0.1f);
        R.Set(ability, "startOffCooldown", true);
        R.Set(ability, "canActivateBetweenRounds", true);
        R.Set(ability, "maxActivationsPerRound", -1);
        R.Set(ability, "maxActivationsPerGame", -1);
        R.Set(ability, "livesCost", 0);
        foreach (var b in R.Items(R.Get(ability, "behaviors")))
        {
            if (Discovery.BuffIdOf(b) == "" && !R.TypeName(b).Contains("Zone") && !R.TypeName(b).Contains("CallToArms"))
                continue;
            // the buff part of the ability: last (almost) forever and only reach this carrier's tower
            R.Set(b, "lifespan", 1_000_000f);
            R.Set(b, "lifespanFrames", 60_000_000);
            R.Set(b, "range", Reach);
            R.Set(b, "useTowerRange", true);
            R.Set(b, "useRadius", false);
            R.Set(b, "canEffectThisTower", false);
            R.Set(b, "canAffectThisTower", false);
            R.SetEmptyArray(b, "filters");
        }
        return ability;
    }

    private static Dictionary<string, object?> BuffArgs(object buff, BuffInfo info, string suffix) => new()
    {
        ["name"] = $"BuyableBuffs_{info.IconName}_{suffix}",
        ["mutatorId"] = R.Str(buff, "mutatorId") is { Length: > 0 } id ? id : $"BuyableBuffs_{info.IconName}",
        ["isUnique"] = !info.Stacks,
        ["isGlobal"] = false,
        ["buffLocsName"] = R.Str(buff, "buffLocsName"),
        ["buffIconName"] = R.Str(buff, "buffIconName"),
    };

    private static IEnumerable<Model> ConvertOverclock(object buff, BuffInfo info)
    {
        // Overclock / Ultraboost target one tower you pick, which a hidden carrier can't do, so their rate
        // boost is given as a permanent rate aura with the same numbers and icon.
        var args = BuffArgs(buff, info, "Rate");
        args["multiplier"] = R.Get(buff, "rateModifier", 1f);
        if (R.New(Ns + "RateSupportModel", args) is Model m) yield return PrepSupport(m);
    }

    public static bool CanConvertGeraldo(List<BuffHit> hits) =>
        hits.Any(h => R.TypeName(h.Buff) is "SharpeningStoneBehaviorModel" or "JarOfPicklesBehaviorModel"
            or "FertilizerBehaviorModel");

    private static IEnumerable<Model> ConvertGeraldo(object buff, BuffInfo info)
    {
        // Geraldo's items are used from his shop, so their numbers are moved onto the matching auras
        var models = new List<object?>();
        switch (R.TypeName(buff))
        {
            case "SharpeningStoneBehaviorModel":
            {
                var args = BuffArgs(buff, info, "Pierce");
                args["pierce"] = R.Get(buff, "pierceIncrease", 0f);
                models.Add(R.New(Ns + "PierceSupportModel", args));
                break;
            }
            case "JarOfPicklesBehaviorModel":
            {
                var dmg = BuffArgs(buff, info, "Damage");
                dmg["increase"] = R.Get(buff, "damageIncrease", 0f);
                var damage = R.New(Ns + "DamageSupportModel", dmg);
                R.Set(damage, "buffLocsName", R.Str(buff, "buffLocsName"));
                R.Set(damage, "buffIconName", R.Str(buff, "buffIconName"));
                models.Add(damage);
                var rate = BuffArgs(buff, info, "Rate");
                rate["mutatorId"] = rate["mutatorId"] + "_Rate";
                rate["multiplier"] = R.Get(buff, "attackSpeedScale", 1f);
                models.Add(R.New(Ns + "RateSupportModel", rate));
                break;
            }
            case "FertilizerBehaviorModel":
            {
                var args = BuffArgs(buff, info, "Cash");
                args["multiplier"] = R.Get(buff, "cashScale", 1f);
                models.Add(R.New(Ns + "BananaCashIncreaseSupportModel", args));
                break;
            }
        }
        foreach (var m in models)
            if (m is Model model)
                yield return PrepSupport(model);
    }

    // ------------------------------------------------------------------ in game

    public static bool IsCarrier(Tower? t) => t?.towerModel?.GetModTower() is BuffTower;

    private static BuffInfo? InfoOf(Tower carrier) => (carrier.towerModel.GetModTower() as BuffTower)?.Info;

    public static void OnPlaced(Tower carrier) => Pending.Add(carrier);

    public static void OnTowerDestroyed(Tower tower)
    {
        if (IsCarrier(tower))
        {
            TargetOf.Remove(tower.Pointer);
            return;
        }
        foreach (var (carrierPtr, target) in TargetOf.ToList())
        {
            if (target.Pointer != tower.Pointer) continue;
            TargetOf.Remove(carrierPtr);
            var carrier = Carriers().FirstOrDefault(c => c.Pointer == carrierPtr);
            if (carrier != null) ToSell.Add(carrier);
        }
    }

    public static void Reset()
    {
        TargetOf.Clear();
        Pending.Clear();
        ToRefund.Clear();
        ToSell.Clear();
    }

    private static List<Tower> Carriers() =>
        InGame.instance == null ? new List<Tower>() : InGame.instance.GetTowers().Where(IsCarrier).ToList();

    public static void Update()
    {
        if (InGame.instance == null) return;

        foreach (var carrier in Pending.ToList())
        {
            Pending.Remove(carrier);
            try
            {
                Attach(carrier);
            }
            catch (Exception e)
            {
                MelonLogger.Warning($"[Buyable Buffs] attaching a buff failed: {e}");
            }
        }

        foreach (var (carrier, message) in ToRefund.ToList())
        {
            ToRefund.Remove((carrier, message));
            Refund(carrier, message);
        }

        foreach (var carrier in ToSell.ToList())
        {
            ToSell.Remove(carrier);
            try
            {
                carrier.SellTower();
            }
            catch
            {
                // already gone
            }
        }

        if (Time.unscaledTime < nextUpkeep) return;
        nextUpkeep = Time.unscaledTime + 0.25f;
        foreach (var carrier in Carriers())
        {
            try
            {
                Upkeep(carrier);
            }
            catch
            {
                // a single broken carrier shouldn't stop the others
            }
        }
    }

    private static void Attach(Tower carrier)
    {
        var info = InfoOf(carrier);
        if (info == null) return;
        var target = FindTarget(carrier, out var distance);
        if (target == null)
        {
            ToRefund.Add((carrier, "Drop the buff onto one of your towers."));
            return;
        }

        var already = TargetOf.Count(kv => kv.Value.Pointer == target.Pointer &&
                                           Carriers().FirstOrDefault(c => c.Pointer == kv.Key) is { } c &&
                                           c.towerModel.name == carrier.towerModel.name);
        if (already >= info.MaxStacks)
        {
            ToRefund.Add((carrier, info.Stacks
                ? $"That tower already has {info.Name} {info.MaxStacks} times, the most it stacks."
                : $"That tower already has {info.Name}, and it doesn't stack."));
            return;
        }

        if (!Snap(carrier, target) && distance > Reach)
        {
            ToRefund.Add((carrier, "Drop the buff closer to the middle of the tower."));
            return;
        }

        TargetOf[carrier.Pointer] = target;
        Hide(carrier);
        ActivateAbilities(carrier);
    }

    private static void Upkeep(Tower carrier)
    {
        Hide(carrier);
        if (!TargetOf.TryGetValue(carrier.Pointer, out var target))
        {
            // after loading a save: find the tower this carrier sits on again
            target = FindTarget(carrier, out var distance);
            if (target == null || distance > Reach) return;
            TargetOf[carrier.Pointer] = target;
        }

        // ability buffs (Call to Arms, Rallying Roar...) get cast again if their buff ever drops off
        var info = InfoOf(carrier);
        if (info?.Kind != BuffKind.Ability || info.MutatorIds.Count == 0) return;
        if (info.MutatorIds.Any(id => R.Call(target, "GetMutatorById", id) == null))
            ActivateAbilities(carrier);
    }

    private static void ActivateAbilities(Tower carrier)
    {
        foreach (var behavior in R.Items(R.Get(carrier, "towerBehaviors")))
            if (R.TypeName(behavior) == "Ability")
                R.Call(behavior, "Activate");
    }

    private static void Hide(Tower carrier)
    {
        var node = carrier.GetUnityDisplayNode();
        if (node != null && node.gameObject.activeSelf) node.gameObject.SetActive(false);
    }

    private static bool TryPos(object tower, out float x, out float y)
    {
        var p = R.Get(tower, "Position");
        x = R.Get(p, "X", float.NaN);
        if (float.IsNaN(x)) x = R.Get(p, "x", float.NaN);
        y = R.Get(p, "Y", float.NaN);
        if (float.IsNaN(y)) y = R.Get(p, "y", float.NaN);
        if (!float.IsNaN(x) && !float.IsNaN(y)) return true;

        // fall back to the position in the Unity scene (x and z are the map's two axes there)
        var unity = R.Call(p, "ToUnity");
        x = R.Get(unity, "x", float.NaN);
        y = R.Get(unity, "z", float.NaN);
        return !float.IsNaN(x) && !float.IsNaN(y);
    }

    private static Tower? FindTarget(Tower carrier, out float distance)
    {
        distance = float.MaxValue;
        if (!TryPos(carrier, out var cx, out var cy)) return null;
        Tower? best = null;
        foreach (var tower in InGame.instance.GetTowers())
        {
            if (tower.Pointer == carrier.Pointer || IsCarrier(tower)) continue;
            if (!TryPos(tower, out var x, out var y)) continue;
            var d = MathF.Sqrt((x - cx) * (x - cx) + (y - cy) * (y - cy));
            if (d >= distance) continue;
            distance = d;
            best = tower;
        }
        if (best == null) return null;
        var radius = Math.Max(Reach, R.Get(best.towerModel, "radius", 6f) + 2f);
        return distance <= radius ? best : null;
    }

    /// <summary>Moves the carrier onto the exact center of its tower.</summary>
    private static bool Snap(Tower carrier, Tower target)
    {
        if (!TryPos(target, out var x, out var y)) return false;
        try
        {
            var typed = R.Typed(carrier);
            var method = typed.GetType().GetMethods()
                .FirstOrDefault(m => m.Name == "PositionTower" && m.GetParameters().Length >= 1);
            if (method == null) return false;
            var ps = method.GetParameters();
            var vec = Activator.CreateInstance(ps[0].ParameterType, x, y);
            var args = new object?[ps.Length];
            args[0] = vec;
            for (var i = 1; i < ps.Length; i++)
                args[i] = ps[i].HasDefaultValue ? ps[i].DefaultValue : ps[i].ParameterType.IsValueType
                    ? Activator.CreateInstance(ps[i].ParameterType)
                    : null;
            method.Invoke(typed, args);
            return TryPos(carrier, out var nx, out var ny) && MathF.Abs(nx - x) < 0.5f && MathF.Abs(ny - y) < 0.5f;
        }
        catch (Exception e)
        {
            MelonLogger.Warning($"[Buyable Buffs] couldn't move a buff onto its tower: {e.Message}");
            return false;
        }
    }

    private static void Refund(Tower carrier, string message)
    {
        try
        {
            var paid = R.Get(carrier, "worth", (float) carrier.towerModel.cost);
            var before = InGame.instance.GetCash();
            carrier.SellTower();
            var gained = InGame.instance.GetCash() - before;
            if (paid > gained) InGame.instance.AddCash(paid - gained);
        }
        catch (Exception e)
        {
            MelonLogger.Warning($"[Buyable Buffs] refund failed: {e.Message}");
        }
        try
        {
            Game.instance.ShowMessage(message, "Buyable Buffs");
        }
        catch
        {
            MelonLogger.Msg($"[Buyable Buffs] {message}");
        }
    }
}
