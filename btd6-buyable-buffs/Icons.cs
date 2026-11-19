using System.Linq;
using System.Reflection;
using BTD_Mod_Helper.Api.Enums;

namespace BuyableBuffs;

/// <summary>Finds the game's own buff icons.</summary>
internal static class Icons
{
    /// <summary>Sprite id of the icon a tower shows for this buff.</summary>
    public static string Guid(BuffLine line)
    {
        var levels = Buffs.Levels(line);
        var buffId = levels.Select(l => l.BuffId).FirstOrDefault(id => id != "") ?? "";
        if (buffId != "")
        {
            var gameData = R.Get(R.FindType("Assets.Scripts.Data.GameData")?.GetProperty("Instance")?.GetValue(null),
                "buffIconSprites");
            foreach (var sprite in R.Items(R.Get(gameData, "buffIconSprites")))
            {
                if (R.Str(sprite, "buffId") != buffId) continue;
                var guid = R.Str(R.Get(sprite, "icon"), "guidRef");
                if (guid != "") return guid;
            }
        }
        return typeof(VanillaSprites).GetField(line.FallbackIcon, BindingFlags.Public | BindingFlags.Static)
            ?.GetValue(null) as string ?? "";
    }

    /// <summary>The buff indicator (icon on the buffed tower) the game uses for this buff level.</summary>
    public static object? Indicator(BuffLevel level, BuffLine line)
    {
        if (level.BuffId == "") return null;
        foreach (var indicator in R.Items(R.Get(Buffs.GameModel, "buffIndicatorModels")))
            if (R.Str(indicator, "iconName") == level.BuffId)
                return indicator;
        return R.New("Assets.Scripts.Models.GenericBehaviors.BuffIndicatorModel", new()
        {
            ["name"] = "BuffIndicatorModel_BuyableBuffs_" + line.Id,
            ["buffName"] = level.BuffLocs,
            ["iconName"] = level.BuffId,
        });
    }
}
