using System;
using System.Collections.Generic;
using System.Linq;
using BTD_Mod_Helper;
using BTD_Mod_Helper.Api.ModOptions;
using BTD_Mod_Helper.Extensions;
using BuyableBuffs;
using Il2CppAssets.Scripts.Models.Profile;
using Il2CppAssets.Scripts.Models.Towers;
using Il2CppAssets.Scripts.Simulation.Towers;
using Il2CppAssets.Scripts.Unity.UI_New.InGame;
using MelonLoader;

[assembly: MelonInfo(typeof(BuyableBuffsMod), ModHelperData.Name, ModHelperData.Version, ModHelperData.RepoOwner)]
[assembly: MelonGame("Ninja Kiwi", "BloonsTD6")]

namespace BuyableBuffs;

/// <summary>What one tower has bought.</summary>
internal sealed class TowerBuffs
{
    public Tower Tower = null!;
    public readonly Dictionary<string, int> Stacks = new();

    /// <summary>The tower's model without our buffs (what the game gave it last).</summary>
    public TowerModel? BaseModel;

    /// <summary>The buffed model we last gave the tower. If the tower's model is ever something else
    /// (upgrade, hero level up, paragon degree...) the buffs get put back on top of the new one.</summary>
    public IntPtr AppliedModel;
}

public class BuyableBuffsMod : BloonsTD6Mod
{
    public static readonly ModSettingDouble PriceMultiplier = new(1.0)
    {
        displayName = "Price Multiplier",
        description = "Multiplies the price of every buff.",
        min = 0,
        max = 10,
        slider = true,
    };

    public static readonly ModSettingInt MaxStacks = new(5)
    {
        displayName = "Max Stacks",
        description = "How many times each stacking buff can be bought for one tower.",
        min = 1,
        max = 100,
        slider = true,
    };

    public static readonly ModSettingBool PricesRiseWithStacks = new(true)
    {
        displayName = "Prices Rise With Stacks",
        description = "Each extra stack of a buff costs more (base x stack number).",
    };

    private const string SaveKey = "BuyableBuffs";
    private static readonly Dictionary<IntPtr, TowerBuffs> State = new();

    // ---------------------------------------------------------------- queries used by the UI

    internal static int StacksOf(Tower tower, BuffDef def) =>
        State.TryGetValue(tower.Pointer, out var s) && s.Stacks.TryGetValue(def.Id, out var n) ? n : 0;

    internal static int MaxOf(BuffDef def) => def.Single ? 1 : def.MaxStacks ?? (int)(long)MaxStacks;

    internal static double PriceOf(BuffDef def, int owned)
    {
        var price = def.Cost * PriceMultiplier;
        if (PricesRiseWithStacks) price *= owned + 1;
        return Math.Round(price);
    }

    internal static double Cash => InGame.instance != null ? InGame.instance.GetCash() : 0;

    internal static bool TryBuy(Tower tower, BuffDef def)
    {
        if (InGame.instance == null) return false;
        var owned = StacksOf(tower, def);
        if (owned >= MaxOf(def)) return false;
        var price = PriceOf(def, owned);
        if (Cash < price) return false;

        InGame.instance.AddCash(-price);
        var s = GetOrCreate(tower);
        s.Stacks[def.Id] = owned + 1;
        Apply(s);
        return true;
    }

    // ---------------------------------------------------------------- applying buffs

    private static TowerBuffs GetOrCreate(Tower tower)
    {
        if (!State.TryGetValue(tower.Pointer, out var s))
            State[tower.Pointer] = s = new TowerBuffs { Tower = tower };
        return s;
    }

    private static void Apply(TowerBuffs s)
    {
        var root = s.Tower.rootModel.Cast<TowerModel>();
        // If the tower no longer has our model, its current one is the new clean base.
        if (s.BaseModel == null || root.Pointer != s.AppliedModel) s.BaseModel = root;

        var model = s.BaseModel.Duplicate();
        foreach (var (id, stacks) in s.Stacks)
            if (stacks > 0 && Buffs.ById.TryGetValue(id, out var def))
                def.Apply(model, stacks);

        s.Tower.UpdateRootModel(model);
        s.AppliedModel = model.Pointer;
    }

    public override void OnUpdate()
    {
        if (InGame.instance == null || InGame.instance.bridge == null || State.Count == 0)
        {
            BuyableBuffsUI.Tick();
            return;
        }

        foreach (var (key, s) in State.ToList())
        {
            try
            {
                if (s.Tower.rootModel.Pointer != s.AppliedModel) Apply(s);
            }
            catch (Exception e)
            {
                MelonLogger.Warning($"Dropping buffs for a tower that is gone: {e.Message}");
                State.Remove(key);
            }
        }

        BuyableBuffsUI.Tick();
    }

    // ---------------------------------------------------------------- game hooks

    public override void OnTowerSelected(Tower tower) => BuyableBuffsUI.Show(tower);

    public override void OnTowerDeselected(Tower tower) => BuyableBuffsUI.Hide();

    public override void OnTowerDestroyed(Tower tower)
    {
        State.Remove(tower.Pointer);
        BuyableBuffsUI.TowerGone(tower);
    }

    public override void OnRestart() => Reset();
    public override void OnMainMenu() => Reset();

    private static void Reset()
    {
        State.Clear();
        BuyableBuffsUI.Destroy();
    }

    // Bought buffs are written into the tower's save data so they come back when a game is continued.
    public override void OnTowerSaved(Tower tower, TowerSaveDataModel saveData)
    {
        if (!State.TryGetValue(tower.Pointer, out var s) || saveData.metaData == null) return;
        var text = string.Join(";", s.Stacks.Where(kv => kv.Value > 0).Select(kv => $"{kv.Key}={kv.Value}"));
        if (text.Length > 0) saveData.metaData[SaveKey] = text;
    }

    public override void OnTowerLoaded(Tower tower, TowerSaveDataModel saveData)
    {
        if (saveData.metaData == null || !saveData.metaData.ContainsKey(SaveKey)) return;
        var s = GetOrCreate(tower);
        foreach (var part in saveData.metaData[SaveKey].Split(';', StringSplitOptions.RemoveEmptyEntries))
        {
            var kv = part.Split('=');
            if (kv.Length == 2 && Buffs.ById.ContainsKey(kv[0]) && int.TryParse(kv[1], out var n))
                s.Stacks[kv[0]] = n;
        }
        s.AppliedModel = IntPtr.Zero; // OnUpdate puts the buffs on next frame
    }
}
