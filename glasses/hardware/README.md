# Lens glasses hardware, v1

- `stl/`: the parts to print, each already lying the way it goes on the bed. No supports needed.
  - `front.stl`: face down
  - `temple_right.stl`, `temple_left.stl`: outer side down
  - `lid_right.stl`, `lid_left.stl`: flat, rim up
- `blueprint.html`: the build sheet: a 3D view of where every part goes, the wiring diagram, the parts list and the build order. Published at https://claude.ai/artifact/6CeASikG2KLT9uxdUsBuyB
- `frame.py`: the model itself. Every size is a setting, so you can make it fit you:

```
pip install manifold3d numpy
python3 frame.py --head-width 136 --board-thickness 12.5
```

| Setting | Default | What it is |
|---|---|---|
| `--head-width` | 132 | Space between the temples at the front, in mm. Measure your head just in front of your ears. |
| `--temple-length` | 100 | From the front to the bend behind your ear |
| `--lens-width`, `--lens-height` | 50, 38 | Size of each lens opening |
| `--bridge` | 18 | Gap between the lenses over your nose |
| `--board-thickness` | 12 | XIAO + Sense board stack with the camera unplugged |

Print in PETG (or PLA for a first try): 0.2 mm layers, 3 walls, 25% infill. The front is 163 mm wide, so it needs a bed at least 170 mm wide.

Things to check on the first print: the lens openings and nose pads against your face, the board and battery fitting the right temple, and the camera sitting square in the end piece. Change a setting, regenerate and reprint just that part.
