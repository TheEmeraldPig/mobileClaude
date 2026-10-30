#!/usr/bin/env python3
"""Builds the three OrcaSlicer profiles (printer, print, filament) used to slice the glasses.

It starts from OrcaSlicer's own LulzBot TAZ 6 presets, the ones the owner's prints use, flattens
their "inherits" chains (OrcaSlicer's command line wants complete profiles), applies the owner's
settings from taz6-orca-settings.txt (copied from a G-code file they printed), then a few changes
for these parts.

  python3 orca_profiles.py <OrcaSlicer resources/profiles folder> <output folder>
"""
import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
PRESETS = {   # first one OrcaSlicer has wins (preset names differ between OrcaSlicer versions)
    "machine": ["Lulzbot Taz 6 0.5 nozzle"],
    "process": ["0.25mm Standard @Lulzbot Taz 6"],
    "filament": ["Lulzbot 2.85mm PLA", "Generic PLA @Lulzbot"],   # the owner prints with the first
}
# Plain settings worth carrying over from the owner's own slicing (others stay as OrcaSlicer's preset has them).
FROM_OWNER = """
layer_height initial_layer_print_height wall_loops top_shell_layers bottom_shell_layers sparse_infill_density
sparse_infill_pattern outer_wall_speed inner_wall_speed sparse_infill_speed internal_solid_infill_speed top_surface_speed
initial_layer_speed travel_speed brim_type skirt_loops seam_position line_width enable_support support_type
nozzle_temperature nozzle_temperature_initial_layer hot_plate_temp hot_plate_temp_initial_layer
retraction_length retraction_speed filament_max_volumetric_speed z_offset
filament_diameter filament_density filament_flow_ratio filament_type fan_min_speed fan_max_speed
cool_plate_temp cool_plate_temp_initial_layer textured_plate_temp textured_plate_temp_initial_layer
""".split()
# For the glasses: an extra wall makes the hinge knuckles and thin arms stronger; no supports are needed.
CHANGES = {"wall_loops": "3", "sparse_infill_density": "25%", "enable_support": "0", "brim_type": "auto_brim"}


def owner_settings():
    out = {}
    for line in (HERE / "taz6-orca-settings.txt").read_text().splitlines():
        if line.startswith("; ") and " = " in line:
            k, v = line[2:].split(" = ", 1)
            out[k.strip()] = v.strip()
    return out


def index(profiles):
    """Every preset by name, across vendors (some inherit from shared ones)."""
    by_name = {}
    for f in profiles.rglob("*.json"):
        try:
            d = json.loads(f.read_text())
        except (json.JSONDecodeError, UnicodeDecodeError):
            continue
        if isinstance(d, dict) and "name" in d and "type" in d:
            vendor_first = "Lulzbot" in f.parts
            if d["name"] not in by_name or vendor_first:
                by_name[d["name"]] = d
    return by_name


def flatten(name, by_name, seen=()):
    if name not in by_name:
        close = sorted(n for n in by_name if "ulzbot" in n or "Taz" in n)
        raise SystemExit(f"OrcaSlicer has no preset called {name!r}. LulzBot presets it has: {close}")
    d = by_name[name]
    parent = d.get("inherits")
    base = flatten(parent, by_name, seen + (name,)) if parent and parent not in seen else {}
    merged = {**base, **{k: v for k, v in d.items() if k != "inherits"}}
    return merged


FILAMENT_KEYS = ("filament_", "nozzle_temperature", "hot_plate_temp", "cool_plate_temp", "textured_plate_temp", "fan_")


def apply(profile, settings, kind):
    for k, v in settings.items():
        if k in profile:
            cur = profile[k]
            profile[k] = [v] * len(cur) if isinstance(cur, list) else v
        elif kind == "filament" and k.startswith(FILAMENT_KEYS):
            profile[k] = [v]   # filament settings are per-extruder lists


def main():
    profiles, out = Path(sys.argv[1]), Path(sys.argv[2])
    out.mkdir(parents=True, exist_ok=True)
    by_name = index(profiles)
    printer = None
    owner = {k: v for k, v in owner_settings().items() if k in FROM_OWNER}
    for kind, names in PRESETS.items():
        name = next((n for n in names if n in by_name), names[0])
        prof = flatten(name, by_name)
        apply(prof, owner, kind)
        apply(prof, CHANGES, kind)
        # Keep OrcaSlicer's own preset names: its command line only slices when the print and filament
        # profiles list the printer profile's name as compatible.
        prof.update({"name": name, "from": "User", "inherits": "", "is_custom_defined": "1"})
        if kind == "machine":
            # OrcaSlicer 2.4's command line matches a user printer by the name it inherits from; this one
            # inherits from nothing (it's flattened), so mark it as the system preset it is a copy of.
            prof["from"] = "system"
            printer = name
        else:
            prof["compatible_printers"] = [printer]
            prof["compatible_printers_condition"] = ""
        (out / f"{kind}.json").write_text(json.dumps(prof, indent=1))
        print(f"{kind}: {name} ({len(prof)} settings)")


if __name__ == "__main__":
    main()
