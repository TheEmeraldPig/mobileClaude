using System.Collections.Generic;
using BTD_Mod_Helper.Api.Components;
using BTD_Mod_Helper.Api.Enums;
using Il2CppAssets.Scripts.Simulation.Towers;
using Il2CppAssets.Scripts.Unity.UI_New.InGame;
using Il2CppTMPro;
using UnityEngine;
using UnityEngine.UI;

namespace BuyableBuffs;

/// <summary>
/// A "BUFFS" button appears at the top of the screen while a tower is selected. It opens a scrolling
/// shop listing every buff, how many stacks the selected tower owns, and a buy button.
/// </summary>
internal static class BuyableBuffsUI
{
    private sealed class Row
    {
        public BuffDef Def = null!;
        public ModHelperText Owned = null!;
        public ModHelperButton Buy = null!;
        public ModHelperText BuyText = null!;
    }

    private static ModHelperButton? toggle;
    private static ModHelperPanel? shop;
    private static ModHelperText? header;
    private static readonly List<Row> Rows = new();
    private static Tower? tower;
    private static float nextRefresh;

    private static RectTransform? Root => InGame.instance != null ? InGame.instance.uiRect : null;

    public static void Show(Tower selected)
    {
        tower = selected;
        if (Root == null) return;
        if (toggle == null)
        {
            toggle = ModHelperButton.Create(new Info("BuyableBuffsToggle", 0, -190, 520, 170, new Vector2(0.5f, 1f)),
                VanillaSprites.GreenBtnLong, new System.Action(ToggleShop));
            toggle.AddText(new Info("Text", InfoPreset.FillParent), "BUFFS", 70);
            toggle.transform.SetParent(Root, false);
        }
        toggle.gameObject.SetActive(true);
        Refresh();
    }

    public static void Hide()
    {
        tower = null;
        if (toggle != null) toggle.gameObject.SetActive(false);
        if (shop != null) shop.gameObject.SetActive(false);
    }

    public static void TowerGone(Tower gone)
    {
        if (tower != null && tower.Pointer == gone.Pointer) Hide();
    }

    public static void Destroy()
    {
        if (toggle != null) Object.Destroy(toggle.gameObject);
        if (shop != null) Object.Destroy(shop.gameObject);
        toggle = null;
        shop = null;
        header = null;
        Rows.Clear();
        tower = null;
    }

    /// <summary>Keeps prices/affordability current while the shop is open (cash changes every pop).</summary>
    public static void Tick()
    {
        if (shop == null || !shop.gameObject.activeSelf || Time.unscaledTime < nextRefresh) return;
        nextRefresh = Time.unscaledTime + 0.25f;
        Refresh();
    }

    private static void ToggleShop()
    {
        if (tower == null) return;
        if (shop == null) Build();
        if (shop == null) return;
        shop.gameObject.SetActive(!shop.gameObject.activeSelf);
        Refresh();
    }

    private static void Build()
    {
        if (Root == null) return;
        Rows.Clear();

        shop = ModHelperPanel.Create(new Info("BuyableBuffsShop", 0, -60, 1500, 1150, new Vector2(0.5f, 0.5f)),
            VanillaSprites.MainBGPanelBlue);
        shop.transform.SetParent(Root, false);

        header = shop.AddText(new Info("Header", 0, 500, 1300, 110), "Buffs", 64);
        var close = shop.AddButton(new Info("Close", 650, 500, 140, 110), VanillaSprites.RedBtnLong,
            new System.Action(() => shop!.gameObject.SetActive(false)));
        close.AddText(new Info("Text", InfoPreset.FillParent), "X", 64);

        var scroll = shop.AddScrollPanel(new Info("Scroll", 0, -60, 1420, 960), RectTransform.Axis.Vertical,
            VanillaSprites.BlueInsertPanelRound, 18, 24);

        foreach (var category in Buffs.Categories)
        {
            scroll.AddScrollContent(ModHelperText.Create(new Info(category, 0, 0, 1340, 90), category, 60));
            foreach (var def in Buffs.All)
                if (def.Category == category)
                    scroll.AddScrollContent(BuildRow(def));
        }
    }

    private static ModHelperPanel BuildRow(BuffDef def)
    {
        var panel = ModHelperPanel.Create(new Info(def.Id, 0, 0, 1340, 190), VanillaSprites.MainBGPanelBlue);

        var name = panel.AddText(new Info("Name", -290, 42, 700, 80), def.Name, 50, TextAlignmentOptions.Left);
        name.Text.enableAutoSizing = true;
        name.Text.fontSizeMax = 50;

        var desc = panel.AddText(new Info("Desc", -290, -40, 700, 90), def.Description, 34,
            TextAlignmentOptions.Left);
        desc.Text.enableAutoSizing = true;
        desc.Text.fontSizeMax = 34;
        desc.Text.fontSizeMin = 18;

        var owned = panel.AddText(new Info("Owned", 210, 0, 230, 150), "", 46);

        var row = new Row { Def = def, Owned = owned };
        row.Buy = panel.AddButton(new Info("Buy", 500, 0, 300, 140), VanillaSprites.GreenBtnLong,
            new System.Action(() =>
            {
                if (tower != null) BuyableBuffsMod.TryBuy(tower, def);
                Refresh();
            }));
        row.BuyText = row.Buy.AddText(new Info("Text", InfoPreset.FillParent), "", 44);
        Rows.Add(row);
        return panel;
    }

    private static void Refresh()
    {
        if (shop == null || tower == null) return;
        var cash = BuyableBuffsMod.Cash;
        if (header != null)
            header.SetText($"Buffs for {tower.towerModel.name}   (${cash:N0})");

        foreach (var row in Rows)
        {
            var owned = BuyableBuffsMod.StacksOf(tower, row.Def);
            var max = BuyableBuffsMod.MaxOf(row.Def);
            row.Owned.SetText(row.Def.Single ? (owned > 0 ? "Owned" : "") : $"{owned}/{max}");

            var image = row.Buy.GetComponent<Image>();
            if (owned >= max)
            {
                row.BuyText.SetText("MAX");
                if (image != null) image.color = new Color(0.6f, 0.6f, 0.6f, 1f);
            }
            else
            {
                var price = BuyableBuffsMod.PriceOf(row.Def, owned);
                row.BuyText.SetText($"${price:N0}");
                if (image != null) image.color = cash >= price ? Color.white : new Color(1f, 0.45f, 0.45f, 1f);
            }
        }
    }
}
