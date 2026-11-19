using System;
using System.Collections.Generic;
using System.Linq;
using BTD_Mod_Helper.Api;
using BTD_Mod_Helper.Extensions;
using Il2CppAssets.Scripts.Simulation.Towers;
using Il2CppAssets.Scripts.Unity;
using Il2CppAssets.Scripts.Unity.UI_New.InGame;
using MelonLoader;
using Newtonsoft.Json.Linq;

namespace BuyableBuffs;

/// <summary>Turns a dropped buff into a buff on the tower under it.</summary>
internal static class Placement
{
    private static readonly List<Tower> Pending = new();

    public static void OnPlaced(Tower dropped) => Pending.Add(dropped);

    public static void Reset() => Pending.Clear();

    public static void Update()
    {
        if (Pending.Count == 0 || InGame.instance == null) return;
        foreach (var dropped in Pending.ToList())
        {
            Pending.Remove(dropped);
            try
            {
                Apply(dropped);
            }
            catch (Exception e)
            {
                MelonLogger.Warning($"[Buyable Buffs] couldn't apply a buff: {e}");
            }
        }
    }

    private static void Apply(Tower dropped)
    {
        if (dropped.towerModel.GetModTower() is not BuffTower { Line: { } line }) return;
        var mutator = ModContent.GetContent<BuffMutator>().FirstOrDefault(m => m.Line == line);
        var levels = Buffs.Levels(line);
        var paid = R.Get(dropped, "worth", (float) dropped.towerModel.cost);

        // the dropped buff itself goes away without giving any money back
        Remove(dropped);

        var target = FindTarget(dropped);
        if (target == null || mutator == null || levels.Count == 0)
        {
            InGame.instance.AddCash(paid);
            Message("Drop the buff onto one of your towers.");
            return;
        }

        var timed = R.Call(target, "GetMutatorById", mutator.MutatorId);
        var current = timed == null ? 0 : BuffMutator.LevelOf(Data(R.Get(timed, "mutator")));
        var level = current + 1;
        if (level > levels.Count)
        {
            InGame.instance.AddCash(paid);
            Message($"That tower already has {line.Name} at its highest level.");
            return;
        }

        // the shop price covers level 1; higher levels cost the difference
        var extra = (double) Buffs.Price(level) - paid;
        if (extra > 0)
        {
            if (InGame.instance.GetCash() < extra)
            {
                InGame.instance.AddCash(paid);
                Message($"Level {level} of {line.Name} costs ${Buffs.Price(level):N0}.");
                return;
            }
            InGame.instance.AddCash(-extra);
        }

        R.Call(target, "RemoveMutatorsById", mutator.MutatorId);
        R2.CallNamed(target, "AddMutator", mutator.Create(new JValue(level)), new Dictionary<string, object?>
        {
            ["time"] = -1,
            ["roundsRemaining"] = -1,
            ["updateDuration"] = true,
            ["applyMutation"] = true,
            ["useRoundTime"] = true,
            ["includeSubTowers"] = true,
        });
        Message($"{levels[level - 1].Name} (level {level}/{levels.Count})");
    }

    private static JToken? Data(object? mutator)
    {
        var json = R.Str(mutator, "bloonTag");
        try
        {
            return json == "" ? null : JToken.Parse(json);
        }
        catch
        {
            return null;
        }
    }

    private static void Remove(Tower dropped)
    {
        var before = InGame.instance.GetCash();
        dropped.SellTower();
        var gained = InGame.instance.GetCash() - before;
        if (gained > 0) InGame.instance.AddCash(-gained);
    }

    private static bool TryPos(object tower, out float x, out float y)
    {
        var p = R.Get(tower, "Position");
        x = R.Get(p, "X", float.NaN);
        if (float.IsNaN(x)) x = R.Get(p, "x", float.NaN);
        y = R.Get(p, "Y", float.NaN);
        if (float.IsNaN(y)) y = R.Get(p, "y", float.NaN);
        if (!float.IsNaN(x) && !float.IsNaN(y)) return true;
        var unity = R.Call(p, "ToUnity");
        x = R.Get(unity, "x", float.NaN);
        y = R.Get(unity, "z", float.NaN);
        return !float.IsNaN(x) && !float.IsNaN(y);
    }

    private static Tower? FindTarget(Tower dropped)
    {
        if (!TryPos(dropped, out var cx, out var cy)) return null;
        Tower? best = null;
        var bestDistance = float.MaxValue;
        foreach (var tower in InGame.instance.GetTowers())
        {
            if (tower.Pointer == dropped.Pointer || tower.towerModel.GetModTower() is BuffTower) continue;
            if (!TryPos(tower, out var x, out var y)) continue;
            var d = MathF.Sqrt((x - cx) * (x - cx) + (y - cy) * (y - cy));
            if (d >= bestDistance) continue;
            bestDistance = d;
            best = tower;
        }
        if (best == null) return null;
        var radius = Math.Max(6f, R.Get(best.towerModel, "radius", 6f) + 3f);
        return bestDistance <= radius ? best : null;
    }

    private static void Message(string text)
    {
        try
        {
            Game.instance.ShowMessage(text, "Buyable Buffs");
        }
        catch
        {
            MelonLogger.Msg($"[Buyable Buffs] {text}");
        }
    }
}
