using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Text;
using BepInEx;
using BepInEx.Configuration;
using UnityEngine;

namespace MagiciteMod
{
    // An in-game menu for Magicite. It works without knowing the game's class names:
    // it finds the game's live objects by reflection, so their numbers (health, gold,
    // speed, ...) can be read, changed and locked while playing.
    [BepInPlugin("com.theemeraldpig.magicitemod", "Magicite Mod", "1.0.0")]
    public class Plugin : BaseUnityPlugin
    {
        ConfigEntry<KeyboardShortcut> menuKey;
        ConfigEntry<KeyboardShortcut> speedKey;
        ConfigEntry<float> fastSpeed;

        bool open;
        int tab;
        Rect window = new Rect(40, 40, 460, 520);
        Vector2 scroll;
        string status = "";

        float speed = 1f;
        float fps;
        float lastTime;

        // Objects tab
        string filter = "";
        List<TypeGroup> groups = new List<TypeGroup>();
        TypeGroup pickedGroup;
        MonoBehaviour picked;
        readonly Dictionary<FieldInfo, string> edits = new Dictionary<FieldInfo, string>();
        readonly List<Lock> locks = new List<Lock>();

        class TypeGroup
        {
            public Type type;
            public List<MonoBehaviour> objects = new List<MonoBehaviour>();
        }

        class Lock
        {
            public MonoBehaviour obj;
            public FieldInfo field;
            public object value;
        }

        void Awake()
        {
            menuKey = Config.Bind("Keys", "Menu", new KeyboardShortcut(KeyCode.F1), "Opens and closes the mod menu.");
            speedKey = Config.Bind("Keys", "FastForward", new KeyboardShortcut(KeyCode.F2), "Switches between normal speed and FastSpeed.");
            fastSpeed = Config.Bind("Game", "FastSpeed", 3f, "Game speed used by the FastForward key.");
            Logger.LogInfo("Magicite Mod loaded. Press " + menuKey.Value + " in game to open the menu.");
        }

        void Update()
        {
            if (menuKey.Value.IsDown()) open = !open;
            if (speedKey.Value.IsDown()) SetSpeed(Mathf.Approximately(speed, 1f) ? fastSpeed.Value : 1f);

            // realtimeSinceStartup rather than unscaledDeltaTime, which older Unity versions lack.
            float now = Time.realtimeSinceStartup;
            fps = Mathf.Lerp(fps, 1f / Mathf.Max(now - lastTime, 0.0001f), 0.1f);
            lastTime = now;

            // The game may set timeScale itself (pausing, menus); only push ours while it is changed.
            if (!Mathf.Approximately(speed, 1f) && Time.timeScale > 0f) Time.timeScale = speed;

            ApplyLocks();
        }

        // Again after the game's own Update, so a locked value doesn't flicker.
        void LateUpdate()
        {
            ApplyLocks();
        }

        void SetSpeed(float s)
        {
            speed = s;
            Time.timeScale = s;
        }

        void ApplyLocks()
        {
            for (int i = locks.Count - 1; i >= 0; i--)
            {
                Lock l = locks[i];
                if (l.obj == null) { locks.RemoveAt(i); continue; } // the object was destroyed
                try { l.field.SetValue(l.obj, l.value); }
                catch (Exception e) { Logger.LogWarning("Lock on " + l.field.Name + " removed: " + e.Message); locks.RemoveAt(i); }
            }
        }

        void OnGUI()
        {
            if (!open) return;
            window = GUILayout.Window(0x4D41, window, DrawWindow, "Magicite Mod  (" + menuKey.Value + " to close)");
        }

        void DrawWindow(int id)
        {
            tab = GUILayout.Toolbar(tab, new[] { "Game", "Objects", "Locks (" + locks.Count + ")", "Dump" });
            GUILayout.Space(4);
            scroll = GUILayout.BeginScrollView(scroll);
            switch (tab)
            {
                case 0: DrawGame(); break;
                case 1: DrawObjects(); break;
                case 2: DrawLocks(); break;
                case 3: DrawDump(); break;
            }
            GUILayout.EndScrollView();
            if (status != "") GUILayout.Label(status);
            GUI.DragWindow();
        }

        void DrawGame()
        {
            GUILayout.Label("FPS: " + Mathf.RoundToInt(fps));
            GUILayout.Label("Game speed: " + speed.ToString("0.00") + "x");
            float s = GUILayout.HorizontalSlider(speed, 0.1f, 5f);
            if (!Mathf.Approximately(s, speed)) SetSpeed(Mathf.Round(s * 20f) / 20f);
            GUILayout.BeginHorizontal();
            if (GUILayout.Button("0.5x")) SetSpeed(0.5f);
            if (GUILayout.Button("1x")) SetSpeed(1f);
            if (GUILayout.Button("2x")) SetSpeed(2f);
            if (GUILayout.Button("3x")) SetSpeed(3f);
            GUILayout.EndHorizontal();
            GUILayout.Space(8);
            GUILayout.Label("Objects tab: find the player (or anything else) and change its numbers. " +
                            "Lock a value to keep it from changing, for example health.");
        }

        // Objects ---------------------------------------------------------------

        void Scan()
        {
            groups.Clear();
            pickedGroup = null;
            picked = null;
            var byType = new Dictionary<Type, TypeGroup>();
            foreach (MonoBehaviour mb in (MonoBehaviour[])FindObjectsOfType(typeof(MonoBehaviour)))
            {
                if (mb == null || !IsGameType(mb.GetType())) continue;
                TypeGroup g;
                if (!byType.TryGetValue(mb.GetType(), out g)) byType[mb.GetType()] = g = new TypeGroup { type = mb.GetType() };
                g.objects.Add(mb);
            }
            groups = byType.Values.OrderBy(g => g.type.Name).ToList();
            status = "Found " + groups.Count + " kinds of game object.";
        }

        static bool IsGameType(Type t)
        {
            string asm = t.Assembly.GetName().Name;
            return asm.StartsWith("Assembly-CSharp") || asm.StartsWith("Assembly-UnityScript");
        }

        void DrawObjects()
        {
            GUILayout.BeginHorizontal();
            if (GUILayout.Button("Scan scene", GUILayout.Width(100))) Scan();
            GUILayout.Label("Filter", GUILayout.Width(36));
            filter = GUILayout.TextField(filter);
            GUILayout.EndHorizontal();

            if (picked != null) { DrawFields(); return; }
            if (pickedGroup != null) { DrawInstances(); return; }

            if (groups.Count == 0) GUILayout.Label("Press Scan scene while in a level. Try filtering for player, hero, stats or health.");
            string f = filter.ToLower();
            foreach (TypeGroup g in groups)
            {
                if (f != "" && !g.type.Name.ToLower().Contains(f)) continue;
                if (GUILayout.Button(g.type.Name + "  (" + g.objects.Count + ")"))
                {
                    if (g.objects.Count == 1) Pick(g.objects[0]);
                    else pickedGroup = g;
                }
            }
        }

        void DrawInstances()
        {
            if (GUILayout.Button("< Back")) { pickedGroup = null; return; }
            GUILayout.Label(pickedGroup.type.Name);
            foreach (MonoBehaviour mb in pickedGroup.objects)
            {
                if (mb == null) continue;
                if (GUILayout.Button(mb.gameObject.name + "  at " + Vector3Text(mb.transform.position))) Pick(mb);
            }
        }

        void Pick(MonoBehaviour mb)
        {
            picked = mb;
            edits.Clear();
            scroll = Vector2.zero;
        }

        static string Vector3Text(Vector3 v)
        {
            return "(" + v.x.ToString("0.0") + ", " + v.y.ToString("0.0") + ")";
        }

        // Fields that can be shown and edited as text: numbers, bools and strings,
        // from the object's class and its parents up to MonoBehaviour.
        static List<FieldInfo> EditableFields(Type t)
        {
            var list = new List<FieldInfo>();
            for (; t != null && t != typeof(MonoBehaviour); t = t.BaseType)
            {
                foreach (FieldInfo fi in t.GetFields(BindingFlags.Instance | BindingFlags.Public | BindingFlags.NonPublic | BindingFlags.DeclaredOnly))
                {
                    if (fi.IsLiteral || fi.IsInitOnly) continue;
                    if (fi.Name.Contains("<")) continue; // compiler-made backing fields
                    if (IsSimple(fi.FieldType)) list.Add(fi);
                }
            }
            return list;
        }

        static bool IsSimple(Type t)
        {
            return t == typeof(int) || t == typeof(float) || t == typeof(bool) || t == typeof(string) ||
                   t == typeof(double) || t == typeof(long) || t == typeof(short) || t == typeof(byte) || t.IsEnum;
        }

        void DrawFields()
        {
            if (picked == null) return;
            if (GUILayout.Button("< Back")) { picked = null; return; }
            GUILayout.Label(picked.GetType().Name + " on " + picked.gameObject.name);

            string f = filter.ToLower();
            foreach (FieldInfo fi in EditableFields(picked.GetType()))
            {
                if (f != "" && !fi.Name.ToLower().Contains(f) && !picked.GetType().Name.ToLower().Contains(f)) continue;
                object value;
                try { value = fi.GetValue(picked); } catch { continue; }
                Lock existing = locks.FirstOrDefault(l => l.obj == picked && l.field == fi);

                GUILayout.BeginHorizontal();
                GUILayout.Label(fi.Name, GUILayout.Width(150));
                if (fi.FieldType == typeof(bool))
                {
                    bool b = GUILayout.Toggle((bool)value, value.ToString());
                    if (b != (bool)value) SetField(fi, b, existing);
                }
                else
                {
                    string text;
                    if (!edits.TryGetValue(fi, out text)) text = value == null ? "" : value.ToString();
                    edits[fi] = GUILayout.TextField(text, GUILayout.Width(110));
                    if (GUILayout.Button("Set", GUILayout.Width(40)))
                    {
                        object parsed;
                        if (TryParse(fi.FieldType, edits[fi], out parsed)) { SetField(fi, parsed, existing); edits.Remove(fi); }
                        else status = "Can't read \"" + edits[fi] + "\" as " + fi.FieldType.Name + ".";
                    }
                    if (fi.FieldType != typeof(string) && !fi.FieldType.IsEnum && GUILayout.Button("x10", GUILayout.Width(36)))
                    {
                        SetField(fi, Multiply(value, 10), existing);
                        edits.Remove(fi);
                    }
                }
                bool locked = GUILayout.Toggle(existing != null, "Lock");
                if (locked && existing == null) locks.Add(new Lock { obj = picked, field = fi, value = fi.GetValue(picked) });
                if (!locked && existing != null) locks.Remove(existing);
                GUILayout.EndHorizontal();
            }
        }

        void SetField(FieldInfo fi, object value, Lock existing)
        {
            try
            {
                fi.SetValue(picked, value);
                if (existing != null) existing.value = value;
                status = fi.Name + " = " + value;
            }
            catch (Exception e) { status = "Couldn't set " + fi.Name + ": " + e.Message; }
        }

        static object Multiply(object v, int by)
        {
            if (v is int) return (int)v * by;
            if (v is float) return (float)v * by;
            if (v is double) return (double)v * by;
            if (v is long) return (long)v * by;
            if (v is short) return (short)((short)v * by);
            if (v is byte) return (byte)Math.Min(255, (byte)v * by);
            return v;
        }

        static bool TryParse(Type t, string s, out object result)
        {
            result = null;
            try
            {
                if (t == typeof(string)) result = s;
                else if (t.IsEnum) result = Enum.Parse(t, s, true);
                else result = Convert.ChangeType(s.Trim(), t, System.Globalization.CultureInfo.InvariantCulture);
                return true;
            }
            catch { return false; }
        }

        void DrawLocks()
        {
            if (locks.Count == 0) GUILayout.Label("Nothing is locked. Tick Lock next to a value in the Objects tab.");
            for (int i = 0; i < locks.Count; i++)
            {
                Lock l = locks[i];
                if (l.obj == null) continue;
                GUILayout.BeginHorizontal();
                GUILayout.Label(l.obj.GetType().Name + "." + l.field.Name + " = " + l.value);
                if (GUILayout.Button("Unlock", GUILayout.Width(60))) { locks.RemoveAt(i); i--; }
                GUILayout.EndHorizontal();
            }
            if (locks.Count > 0 && GUILayout.Button("Unlock all")) locks.Clear();
        }

        // Dump ------------------------------------------------------------------

        void DrawDump()
        {
            GUILayout.Label("Writes every class in the game's code, with its fields and methods, to a text file. " +
                            "Useful for making the mod do more: send the file along and new features can target the game directly.");
            if (GUILayout.Button("Write class list")) DumpTypes();
        }

        void DumpTypes()
        {
            string path = Path.Combine(Paths.BepInExRootPath, "MagiciteMod-classes.txt");
            var sb = new StringBuilder();
            const BindingFlags all = BindingFlags.Instance | BindingFlags.Static | BindingFlags.Public | BindingFlags.NonPublic | BindingFlags.DeclaredOnly;
            int count = 0;
            foreach (Assembly asm in AppDomain.CurrentDomain.GetAssemblies())
            {
                if (!asm.GetName().Name.StartsWith("Assembly-")) continue;
                Type[] types;
                try { types = asm.GetTypes(); }
                catch (ReflectionTypeLoadException e) { types = e.Types.Where(t => t != null).ToArray(); }
                sb.AppendLine("=== " + asm.GetName().Name);
                foreach (Type t in types.OrderBy(t => t.FullName))
                {
                    count++;
                    sb.AppendLine();
                    sb.AppendLine("class " + t.FullName + (t.BaseType != null ? " : " + t.BaseType.Name : ""));
                    foreach (FieldInfo fi in t.GetFields(all))
                        sb.AppendLine("    " + (fi.IsStatic ? "static " : "") + fi.FieldType.Name + " " + fi.Name);
                    foreach (MethodInfo mi in t.GetMethods(all))
                    {
                        if (mi.IsSpecialName) continue;
                        string args = string.Join(", ", mi.GetParameters().Select(p => p.ParameterType.Name + " " + p.Name).ToArray());
                        sb.AppendLine("    " + (mi.IsStatic ? "static " : "") + mi.ReturnType.Name + " " + mi.Name + "(" + args + ")");
                    }
                }
            }
            try
            {
                File.WriteAllText(path, sb.ToString());
                status = "Wrote " + count + " classes to " + path;
                Logger.LogInfo(status);
            }
            catch (Exception e) { status = "Couldn't write the file: " + e.Message; }
        }
    }
}
