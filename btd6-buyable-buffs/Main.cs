using BTD_Mod_Helper;
using BuyableBuffs;
using Il2CppAssets.Scripts.Simulation.Towers;
using MelonLoader;

[assembly: MelonInfo(typeof(BuyableBuffsMod), ModHelperData.Name, ModHelperData.Version, ModHelperData.RepoOwner)]
[assembly: MelonGame("Ninja Kiwi", "BloonsTD6")]

namespace BuyableBuffs;

public class BuyableBuffsMod : BloonsTD6Mod
{
    public override void OnUpdate() => Carrier.Update();

    public override void OnTowerDestroyed(Tower tower) => Carrier.OnTowerDestroyed(tower);

    public override void OnRestart() => Carrier.Reset();

    public override void OnMainMenu() => Carrier.Reset();
}
