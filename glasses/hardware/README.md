# Lens glasses hardware, v2

- **G-code:** every push that changes the model makes GitHub slice it for the owner's LulzBot TAZ 6 + Aerostruder (OrcaSlicer 2.4.2, its TAZ 6 presets plus the owner's settings: 0.5 mm nozzle, 2.85 mm PLA, 220 / 55 C, 0.25 mm layers, 3 walls, 25% infill). It's published as a release named `glasses-print-<number>`: take the newest. About 4 hours and 53 g of PLA, everything on one plate.
- `stl/`: the parts, each lying the way it prints (no supports), and `lens_glasses_plate.stl` with all of them arranged on the 280 mm bed.
  - `front.stl`: face down, hinge knuckles up
  - `temple_right.stl`, `temple_left.stl`: outer side down
  - `lid_right.stl`, `lid_left.stl`: flat, rim up
- `blueprint.html`: the build sheet (3D view, wiring, parts, build order). Published at https://claude.ai/artifact/6CeASikG2KLT9uxdUsBuyB
- `frame.py`: the model. It checks that the hinge parts don't touch, that each temple folds 90 degrees without hitting the front, and that the electronics fit with room for wiring.
- `printer/`: `taz6-orca-settings.txt` is the owner's OrcaSlicer settings (from a G-code file they printed); `orca_profiles.py` turns OrcaSlicer's TAZ 6 presets plus those settings into the profiles `.github/workflows/glasses-print.yml` slices with.

```
pip install manifold3d numpy
python3 frame.py --head-width 136 --board-thickness 12
```

| Setting | Default | What it is |
|---|---|---|
| `--head-width` | 134 | Space between the temples at the hinges, in mm. Measure your head just in front of your ears. |
| `--temple-length` | 100 | From the hinge to the bend behind your ear |
| `--lens-width`, `--lens-height` | 50, 40 | Size of each lens opening |
| `--bridge` | 18 | Gap between the lenses over your nose |
| `--board-thickness` | 11 | XIAO + Sense board stack with the camera unplugged |

**Hinges:** two knuckles on the front, one on each temple, pinned with a 20 mm piece of 2.85 mm filament (fix it at the top only). Each temple folds flat on its own; with one folded, the other folds only part way because the right temple is thick.

Things to check on the first print: the fit on your face, the hinge turning freely (sand the knuckle faces if it's stiff), the board and battery fitting the right temple, and the camera sitting square in its pocket. Change a setting, regenerate and reprint just that part.
