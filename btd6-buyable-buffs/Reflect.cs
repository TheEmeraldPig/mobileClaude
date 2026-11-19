using System;
using System.Collections;
using System.Collections.Generic;
using System.Linq;
using System.Reflection;
using Il2CppInterop.Runtime.InteropTypes;
using MelonLoader;

namespace BuyableBuffs;

/// <summary>
/// Reads and writes game objects by field name. BTD6 has hundreds of buff types with different fields, and a game
/// object is usually handed to us typed as its base class (Model), so this looks up the object's real class and
/// works on that. Missing fields are simply skipped, so a game update that renames one doesn't crash the mod.
/// </summary>
internal static class R
{
    private static readonly Dictionary<string, Type?> TypeCache = new();
    private static Assembly[]? assemblies;

    /// <summary>The game's own class name of an object, e.g. "RateSupportModel".</summary>
    public static string TypeName(object? o) => Il2CppType(o)?.Name ?? o?.GetType().Name ?? "";

    private static Il2CppSystem.Type? Il2CppType(object? o)
    {
        try
        {
            return o is Il2CppObjectBase b ? b.Cast<Il2CppSystem.Object>().GetIl2CppType() : null;
        }
        catch
        {
            return null;
        }
    }

    /// <summary>Finds the C# wrapper class for a game class name.</summary>
    public static Type? FindType(string il2CppFullName)
    {
        if (TypeCache.TryGetValue(il2CppFullName, out var cached)) return cached;
        assemblies ??= AppDomain.CurrentDomain.GetAssemblies();
        var candidates = new[]
        {
            "Il2Cpp" + il2CppFullName, il2CppFullName, "Il2Cpp." + il2CppFullName
        };
        Type? found = null;
        foreach (var name in candidates)
        {
            foreach (var asm in assemblies)
            {
                found = asm.GetType(name, false);
                if (found != null) break;
            }
            if (found != null) break;
        }
        if (found == null)
        {
            // assemblies loaded later
            assemblies = AppDomain.CurrentDomain.GetAssemblies();
        }
        TypeCache[il2CppFullName] = found;
        return found;
    }

    /// <summary>Returns the object re-wrapped as its real class, so all of its fields are visible.</summary>
    public static object Typed(object o)
    {
        if (o is not Il2CppObjectBase b) return o;
        var il2 = Il2CppType(o);
        if (il2 == null) return o;
        var type = FindType(il2.FullName.Replace('/', '+'));
        if (type == null || type.IsInstanceOfType(o)) return o;
        try
        {
            return Activator.CreateInstance(type, b.Pointer) ?? o;
        }
        catch
        {
            return o;
        }
    }

    private const BindingFlags Flags = BindingFlags.Public | BindingFlags.Instance | BindingFlags.FlattenHierarchy;

    private static PropertyInfo? Prop(object typed, string name)
    {
        try
        {
            return typed.GetType().GetProperty(name, Flags);
        }
        catch (AmbiguousMatchException)
        {
            return typed.GetType().GetProperties(Flags).FirstOrDefault(p => p.Name == name);
        }
    }

    private static FieldInfo? Field(object typed, string name) => typed.GetType().GetField(name, Flags);

    public static bool Has(object? o, string name)
    {
        if (o == null) return false;
        var typed = Typed(o);
        return Prop(typed, name) != null || Field(typed, name) != null;
    }

    public static object? Get(object? o, string name)
    {
        if (o == null) return null;
        var typed = Typed(o);
        try
        {
            if (Prop(typed, name) is { } prop && prop.GetIndexParameters().Length == 0) return prop.GetValue(typed);
            return Field(typed, name)?.GetValue(typed);
        }
        catch
        {
            return null;
        }
    }

    public static T Get<T>(object? o, string name, T fallback)
    {
        var v = Get(o, name);
        if (v == null) return fallback;
        if (v is T t) return t;
        try
        {
            return (T) System.Convert.ChangeType(v.ToString(), typeof(T), System.Globalization.CultureInfo.InvariantCulture);
        }
        catch
        {
            return fallback;
        }
    }

    public static string Str(object? o, string name) => Get(o, name)?.ToString() ?? "";

    public static bool Set(object? o, string name, object? value)
    {
        if (o == null) return false;
        var typed = Typed(o);
        var prop = Prop(typed, name);
        try
        {
            if (prop != null && prop.CanWrite)
            {
                prop.SetValue(typed, Convert(value, prop.PropertyType));
                return true;
            }
            var field = Field(typed, name);
            if (field == null || field.IsInitOnly) return false;
            field.SetValue(typed, Convert(value, field.FieldType));
            return true;
        }
        catch (Exception e)
        {
            MelonLogger.Warning($"[Buyable Buffs] couldn't set {TypeName(o)}.{name}: {e.Message}");
            return false;
        }
    }

    /// <summary>Sets an array field (like "filters") to an empty array.</summary>
    public static void SetEmptyArray(object? o, string name)
    {
        if (o == null) return;
        var typed = Typed(o);
        var prop = Prop(typed, name);
        if (prop == null || !prop.CanWrite) return;
        try
        {
            prop.SetValue(typed, EmptyArray(prop.PropertyType));
        }
        catch
        {
            // leave it
        }
    }

    private static object? EmptyArray(Type t)
    {
        if (t.IsArray) return Array.CreateInstance(t.GetElementType()!, 0);
        return Activator.CreateInstance(t, 0L);
    }

    private static object? Convert(object? value, Type to)
    {
        if (value == null) return null;
        if (to.IsInstanceOfType(value)) return value;
        if (to.IsPrimitive || to == typeof(string) || to.IsEnum)
            return to.IsEnum
                ? Enum.ToObject(to, value)
                : System.Convert.ChangeType(value, to, System.Globalization.CultureInfo.InvariantCulture);
        return value;
    }

    /// <summary>Calls a method by name with the given arguments.</summary>
    public static object? Call(object? o, string name, params object?[] args)
    {
        if (o == null) return null;
        var typed = Typed(o);
        var method = typed.GetType()
            .GetMethods(BindingFlags.Public | BindingFlags.Instance | BindingFlags.FlattenHierarchy)
            .FirstOrDefault(m => m.Name == name && m.GetParameters().Length == args.Length);
        if (method == null) return null;
        var ps = method.GetParameters();
        var converted = args.Select((a, i) => Convert(a, ps[i].ParameterType)).ToArray();
        return method.Invoke(typed, converted);
    }

    /// <summary>Every item of a game list or array (Il2Cpp or normal).</summary>
    public static List<object> Items(object? list)
    {
        var result = new List<object>();
        if (list == null) return result;
        if (list is string) return result;
        if (list is IEnumerable e and not Il2CppObjectBase)
        {
            foreach (var item in e)
                if (item != null)
                    result.Add(item);
            return result;
        }

        var type = list.GetType();
        var count = type.GetProperty("Count")?.GetValue(list) ?? type.GetProperty("Length")?.GetValue(list);
        var indexer = type.GetProperties().FirstOrDefault(p => p.GetIndexParameters().Length == 1);
        if (count == null || indexer == null) return result;
        var n = System.Convert.ToInt32(count);
        var indexType = indexer.GetIndexParameters()[0].ParameterType;
        for (var i = 0; i < n; i++)
        {
            try
            {
                var item = indexer.GetValue(list, new[] { System.Convert.ChangeType(i, indexType) });
                if (item != null) result.Add(item);
            }
            catch
            {
                // skip
            }
        }
        return result;
    }

    /// <summary>
    /// Creates a game object of the named class, filling constructor arguments by parameter name
    /// (anything not given gets an empty/zero value).
    /// </summary>
    public static object? New(string il2CppFullName, Dictionary<string, object?> args)
    {
        var type = FindType(il2CppFullName);
        if (type == null)
        {
            MelonLogger.Warning($"[Buyable Buffs] game class {il2CppFullName} not found");
            return null;
        }
        var ctor = type.GetConstructors()
            .Where(c => c.GetParameters().Length > 1)
            .OrderByDescending(c => c.GetParameters().Count(p => args.ContainsKey(p.Name ?? "")))
            .FirstOrDefault();
        if (ctor == null) return null;
        var values = ctor.GetParameters().Select(p =>
        {
            if (p.Name != null && args.TryGetValue(p.Name, out var v)) return Convert(v, p.ParameterType);
            return DefaultFor(p.ParameterType);
        }).ToArray();
        try
        {
            return ctor.Invoke(values);
        }
        catch (Exception e)
        {
            MelonLogger.Warning($"[Buyable Buffs] couldn't create {il2CppFullName}: {e.InnerException?.Message ?? e.Message}");
            return null;
        }
    }

    private static object? DefaultFor(Type t)
    {
        if (t == typeof(string)) return "";
        if (t.IsValueType) return Activator.CreateInstance(t);
        if (t.IsArray || t.Name.StartsWith("Il2CppReferenceArray") || t.Name.StartsWith("Il2CppStructArray") ||
            t.Name.StartsWith("Il2CppStringArray"))
        {
            try
            {
                return EmptyArray(t);
            }
            catch
            {
                return null;
            }
        }
        return null;
    }
}

internal static class Il2CppListExt
{
    /// <summary>Copies an Il2Cpp IEnumerable (what GetDescendants returns) into a normal C# list,
    /// because C# foreach can't walk Il2Cpp enumerators directly.</summary>
    public static List<T> ToSystemList<T>(this Il2CppSystem.Collections.Generic.IEnumerable<T> source)
    {
        var list = new List<T>();
        var enumerator = source.GetEnumerator();
        var untyped = enumerator.Cast<Il2CppSystem.Collections.IEnumerator>();
        while (untyped.MoveNext()) list.Add(enumerator.Current);
        return list;
    }
}
