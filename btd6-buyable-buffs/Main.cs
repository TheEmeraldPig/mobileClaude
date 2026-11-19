using BTD_Mod_Helper;
using BuyableBuffs;
using MelonLoader;

[assembly: MelonInfo(typeof(BuyableBuffsMod), ModHelperData.Name, ModHelperData.Version, ModHelperData.RepoOwner)]
[assembly: MelonGame("Ninja Kiwi", "BloonsTD6")]

namespace BuyableBuffs;

public class BuyableBuffsMod : BloonsTD6Mod
{
    public override void OnUpdate() => Placement.Update();

    public override void OnRestart() => Placement.Reset();

    public override void OnMainMenu() => Placement.Reset();
}
