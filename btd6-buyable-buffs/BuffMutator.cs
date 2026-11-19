using System.Collections.Generic;
using BTD_Mod_Helper.Api;
using BTD_Mod_Helper.Api.Towers;
using Il2CppAssets.Scripts.Models;
using Il2CppAssets.Scripts.Simulation.Objects;
using Newtonsoft.Json.Linq;

namespace BuyableBuffs;

/// <summary>
/// The buff itself: a mutator put straight on the tower, the same way Geraldo's items, brews and Village auras
/// buff a tower. It never runs out, it's re-applied whenever the tower's model is rebuilt (upgrades), it's saved
/// with the game, and it shows the buff's own icon on the tower. Its data is the buff level.
/// </summary>
public class BuffMutator : ModMutator
{
    private readonly BuffLine? line;

    public BuffMutator()
    {
    }

    public BuffMutator(BuffLine line) => this.line = line;

    public override IEnumerable<ModContent> Load()
    {
        foreach (var l in Buffs.Lines) yield return new BuffMutator(l);
    }

    public override string Name => "Mutator_" + (line?.Id ?? "");

    internal BuffLine? Line => line;

    public override bool IsUnique => true;

    public override bool Saved => true;

    public static int LevelOf(JToken? data)
    {
        try
        {
            return data == null ? 0 : data.Value<int>();
        }
        catch
        {
            return 0;
        }
    }

    public override BehaviorMutator Create(JToken? data = null)
    {
        var mutator = base.Create(data);
        if (line == null) return mutator;
        var levels = Buffs.Levels(line);
        var level = LevelOf(data);
        if (level >= 1 && level <= levels.Count && Icons.Indicator(levels[level - 1], line) is { } indicator)
            R.Set(mutator, "buffIndicator", indicator);
        return mutator;
    }

    public override bool Mutate(Model baseModel, Model model, JToken data)
    {
        if (line == null || R.TypeName(model) != "TowerModel") return false;
        var levels = Buffs.Levels(line);
        var level = LevelOf(data);
        if (level < 1 || level > levels.Count) return false;
        foreach (var fx in levels[level - 1].Effects)
        {
            try
            {
                fx.Apply(model);
            }
            catch
            {
                // one effect failing shouldn't remove the others
            }
        }
        return true;
    }
}
