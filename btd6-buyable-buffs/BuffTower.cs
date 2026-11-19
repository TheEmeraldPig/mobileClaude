using System.Collections.Generic;
using System.Linq;
using BTD_Mod_Helper.Api;
using BTD_Mod_Helper.Api.Towers;
using BTD_Mod_Helper.Extensions;
using Il2CppAssets.Scripts.Models;
using Il2CppAssets.Scripts.Models.Towers;
using Il2CppAssets.Scripts.Models.TowerSets;
using Il2CppAssets.Scripts.Simulation.Towers;
using Il2CppInterop.Runtime.InteropTypes.Arrays;
using Il2CppNinjaKiwi.Common.ResourceUtils;

namespace BuyableBuffs;

/// <summary>
/// A buff in the tower shop (Support section). You buy it like a tower and drop it on the tower to buff; it then
/// disappears and the tower keeps the buff.
/// </summary>
public class BuffTower : ModTower
{
    private readonly BuffLine? line;

    public BuffTower()
    {
    }

    public BuffTower(BuffLine line) => this.line = line;

    public override IEnumerable<ModContent> Load()
    {
        foreach (var l in Buffs.Lines) yield return new BuffTower(l);
    }

    public override string Name => "Buff_" + (line?.Id ?? "");

    internal BuffLine? Line => line;

    private bool Found => line != null && Buffs.Levels(line).Count > 0;

    public override TowerSet TowerSet => TowerSet.Support;

    public override string BaseTower => "DartMonkey";

    public override int Cost => Buffs.Price(1);

    public override bool DontAddToShop => !Found;

    public override bool IncludeInMonkeyTeams => false;

    public override string DisplayName => line?.Name ?? "";

    // Shown in the upgrade/info screen: what every level does, with the numbers.
    public override string Description => line == null ? "" : Buffs.Describe(line);

    public override SpriteReference IconReference => CreateSpriteReference(line == null ? "" : Icons.Guid(line));

    public override SpriteReference PortraitReference => IconReference;

    public override void ModifyBaseTowerModel(TowerModel towerModel)
    {
        // nothing to shoot with; it only needs to be droppable on top of another tower
        var keep = R.Items(R.Get(towerModel, "behaviors"))
            .Where(b => R.TypeName(b) is "DisplayModel" or "CreateSoundOnTowerPlaceModel")
            .Cast<Model>()
            .ToArray();
        R.Set(towerModel, "behaviors", new Il2CppReferenceArray<Model>(keep));
        R.Set(towerModel, "range", 1f);
        R.Set(towerModel, "footprint", R.New("Assets.Scripts.Models.Towers.Behaviors.CircleFootprintModel",
            new Dictionary<string, object?>
            {
                ["name"] = "BuyableBuffsFootprint",
                ["radius"] = 0f,
                ["doesntBlockTowerPlacement"] = true,
                ["ignoresPlacementCheck"] = true,
                ["ignoresTowerOverlap"] = true
            }));
    }

    public override void OnPlaced(Tower tower) => Placement.OnPlaced(tower);
}
