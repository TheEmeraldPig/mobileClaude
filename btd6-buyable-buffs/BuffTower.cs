using System.Collections.Generic;
using System.Reflection;
using BTD_Mod_Helper.Api;
using BTD_Mod_Helper.Api.Enums;
using BTD_Mod_Helper.Api.Towers;
using Il2CppAssets.Scripts.Models.Towers;
using Il2CppAssets.Scripts.Models.TowerSets;
using Il2CppAssets.Scripts.Simulation.Towers;
using Il2CppNinjaKiwi.Common.ResourceUtils;

namespace BuyableBuffs;

/// <summary>
/// One shop entry per buff in the game. There's one for every buff icon BTD6 has; the ones that no tower, hero or
/// Geraldo item gives to other towers are left out of the shop. You buy it like a tower and drop it on the tower to
/// buff.
/// </summary>
public class BuffTower : ModTower
{
    public const string Prefix = "BuyableBuffs-Buff_";

    private readonly string iconName;
    private readonly string iconGuid;

    public BuffTower()
    {
        iconName = "";
        iconGuid = "";
    }

    public BuffTower(string iconName, string iconGuid)
    {
        this.iconName = iconName;
        this.iconGuid = iconGuid;
    }

    /// <summary>Makes one BuffTower per "BuffIcon..." sprite in the game.</summary>
    public override IEnumerable<ModContent> Load()
    {
        foreach (var field in typeof(VanillaSprites).GetFields(BindingFlags.Public | BindingFlags.Static))
        {
            if (field.IsLiteral && field.Name.StartsWith("BuffIcon") && field.GetValue(null) is string guid)
                yield return new BuffTower(field.Name, guid);
        }
    }

    public override string Name => "Buff_" + iconName;

    internal BuffInfo? Info => iconName == "" ? null : Discovery.Get(iconName, iconGuid);

    public override TowerSet TowerSet => TowerSet.Support;

    public override string BaseTower => Info?.SourceTower?.baseId is { Length: > 0 } id ? id : "DartMonkey";

    public override int Cost => Info?.Cost ?? 0;

    public override bool DontAddToShop => Info == null;

    public override bool IncludeInMonkeyTeams => false;

    public override string DisplayName => Info?.Name ?? iconName;

    // The real in-game description of the upgrade (or item) that gives this buff.
    // Mod Helper shows it in the tower's upgrade/info screen.
    public override string Description => Info?.Description ?? "";

    public override SpriteReference IconReference => CreateSpriteReference(iconGuid);

    public override SpriteReference PortraitReference => CreateSpriteReference(iconGuid);

    public override TowerModel GetBaseTowerModel(params int[] tiers) => Carrier.BuildModel(Info, Id);

    public override void ModifyBaseTowerModel(TowerModel towerModel)
    {
    }

    public override void OnPlaced(Tower tower) => Carrier.OnPlaced(tower);
}
