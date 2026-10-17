#!/usr/bin/env python3
"""Lens glasses, version 1: a parametric 3D-printable frame.

Writes print-ready STL files (each part laid out the way it should sit on the print bed)
to stl/, and the same parts in their assembled positions, plus stand-ins for the
electronics, to preview/model.json for the 3D view in blueprint.html.

  pip install manifold3d numpy
  python3 frame.py                      # default sizes
  python3 frame.py --head-width 138     # wider head
  python3 frame.py --board-thickness 14 # if your board stack measures thicker

Layout: everything electronic lives in the right temple ("pod"): the XIAO board at the
front with its USB-C port facing down, the battery behind it. The camera and recording
light sit in the right end piece of the front, looking forward. The left temple has the
same shape (so the glasses look even) with an empty pocket for a counterweight.
Temples join the front with two pins cut from 1.75 mm filament, plus glue. They don't fold.
"""
import argparse
import base64
import json
import math
import struct
from pathlib import Path

import numpy as np
from manifold3d import CrossSection, JoinType, Manifold, set_circular_segments

set_circular_segments(48)
HERE = Path(__file__).resolve().parent


def params(args):
    p = dict(
        # front
        lens_w=args.lens_width, lens_h=args.lens_height, lens_r=10.0, rim=4.5, bridge=args.bridge,
        front_t=5.0, lens_ledge=1.2, lens_rebate=1.8,
        # where the temples sit: inner faces this far apart
        head_w=args.head_width,
        # right temple pod (electronics)
        wall=1.6, board_l=21.0, board_w=17.8, board_t=args.board_thickness, usb_room=1.5,
        battery=(5.2, 20.5, 31.0),     # thickness, height, length of a 502030 cell, with a little room
        front_block=5.0, rear_wall=4.0, lid_t=1.6, lid_lip=1.2, fit=0.25,
        pod_top=44.0,                   # top of the pod, from the bottom of the front
        # end piece
        end_d=12.0, cam=9.6, cam_hole=7.4, cam_skin=1.2, led_hole=3.1,
        # arms
        arm_t=4.5, arm_h=7.0, temple_len=args.temple_length, drop_len=38.0, drop_angle=35.0,
        pin_d=1.9, pin_depth=6.0,
    )
    p["pod_in_t"] = p["board_t"] + 0.4                         # inside thickness
    p["pod_t"] = p["wall"] + p["pod_in_t"]                     # body thickness (lid adds lid_t)
    p["pod_tt"] = p["pod_t"] + p["lid_t"]
    p["pod_in_h"] = p["board_l"] + p["usb_room"] + 0.3
    p["pod_h"] = p["pod_in_h"] + 2 * p["wall"]
    p["pod_bottom"] = p["pod_top"] - p["pod_h"]
    p["pod_in_l"] = p["board_w"] + 1.0 + 1.5 + p["battery"][2] + 1.5   # board, gap, battery, wiring room
    p["pod_l"] = p["front_block"] + p["pod_in_l"] + p["rear_wall"]
    p["front_h"] = p["lens_h"] + 2 * p["rim"]
    p["lens_cz"] = p["front_h"] / 2
    p["lens_cx"] = p["bridge"] / 2 + p["rim"] + p["lens_w"] / 2
    p["cam_z"] = p["pod_bottom"] + p["pod_h"] / 2 + 3.5        # camera centre height
    p["cam_x"] = p["wall"] + p["pod_in_t"] / 2                 # camera centre, measured from the temple's outer face
    p["led_z"] = p["cam_z"] - 9.0
    p["pins"] = [(p["cam_x"], p["pod_bottom"] + 2.0), (p["cam_x"], p["pod_top"] - 2.0)]
    p["opening"] = (p["pod_bottom"] + 4.0, p["pod_top"] - 3.8)  # wiring opening between end piece and pod
    return p


# ---------- helpers ----------
def rrect(w, h, r, cx=0.0, cy=0.0):
    """Rounded rectangle centred at (cx, cy)."""
    r = min(r, w / 2 - 0.01, h / 2 - 0.01)
    return CrossSection.square((w - 2 * r, h - 2 * r), center=True).offset(r, JoinType.Round).translate((cx, cy))


def rect(x0, y0, x1, y1):
    return CrossSection.square((x1 - x0, y1 - y0)).translate((x0, y0))


def box(x0, y0, z0, x1, y1, z1):
    return Manifold.cube((x1 - x0, y1 - y0, z1 - z0)).translate((x0, y0, z0))


def cyl_z(x, y, z0, z1, d):
    return Manifold.cylinder(z1 - z0, d / 2).translate((x, y, z0))


def prism(cs, z0, z1):
    return Manifold.extrude(cs, z1 - z0).translate((0, 0, z0))


# ---------- parts, in print orientation ----------
def front(p):
    """Face down on the bed: x = across the face, y = up, z = backwards from the front face."""
    lens = rrect(p["lens_w"], p["lens_h"], p["lens_r"])
    rimmed = lens.offset(p["rim"], JoinType.Round)
    shape = CrossSection()
    for s in (-1, 1):
        shape = shape + rimmed.translate((s * p["lens_cx"], p["lens_cz"]))
    top = p["front_h"]
    shape = shape + rect(-p["bridge"] / 2 - 3, top - 11, p["bridge"] / 2 + 3, top)          # bridge
    ex0 = p["head_w"] / 2
    for s in (-1, 1):                                                                          # joins to the end pieces
        x0, x1 = sorted((s * (p["lens_cx"] + p["lens_w"] / 2), s * (ex0 + 2)))
        shape = shape + rrect(x1 - x0, 16, 3, (x0 + x1) / 2, p["pod_top"] - 10)
    body = prism(shape, 0, p["front_t"])

    for s in (-1, 1):
        cx = s * p["lens_cx"]
        body = body - prism(lens.translate((cx, p["lens_cz"])), -1, p["front_t"] + 1)           # lens opening
        ledge = lens.offset(p["lens_ledge"], JoinType.Round).translate((cx, p["lens_cz"]))
        body = body - prism(ledge, p["front_t"] - p["lens_rebate"], p["front_t"] + 1)          # optional lens seat
        # nose pad: a slanted tab behind the inner lower rim
        pad = box(-1.2, -7, 0, 1.2, 7, 9).rotate((0, s * 25, 0))
        pad = pad.translate((s * (p["bridge"] / 2 + 1.6), p["lens_cz"] - 9, p["front_t"] - 1))
        body = body + pad

    for side in ("right", "left"):
        s = 1 if side == "right" else -1
        xo = s * (ex0 + p["pod_tt"])     # outer face of the temple
        x0, x1 = sorted((s * ex0, xo))
        block = prism(rrect(x1 - x0, p["pod_h"], 2.5, (x0 + x1) / 2, p["pod_bottom"] + p["pod_h"] / 2), 0, p["end_d"])
        body = body + block
        # temple pins: holes in the back of the end piece
        for (px, pz) in p["pins"]:
            x = xo - s * px
            body = body - cyl_z(x, pz, p["end_d"] - p["pin_depth"], p["end_d"] + 1, p["pin_d"])
        if side == "right":
            cx = xo - s * p["cam_x"]
            c = p["cam"] / 2
            body = body - box(cx - c, p["cam_z"] - c, p["cam_skin"], cx + c, p["cam_z"] + c, p["end_d"] + 1)   # camera pocket
            body = body - cyl_z(cx, p["cam_z"], -1, p["cam_skin"] + 0.5, p["cam_hole"])                      # lens window
            body = body - cyl_z(cx, p["led_z"], -1, p["end_d"] + 1, p["led_hole"])                           # recording light
            body = body - cyl_z(cx, p["led_z"], 1.5, p["end_d"] + 1, 3.6)                                    # LED body
    return body


def temple(p, side):
    """Lying on its outer face: x = along the temple (0 = joint), y = up, z = from the outer face inwards."""
    w, L = p["wall"], p["pod_l"]
    yb, yt = p["pod_bottom"], p["pod_top"]
    pod_out = rrect(L, p["pod_h"], 3, L / 2, yb + p["pod_h"] / 2)
    body = prism(pod_out, 0, p["pod_t"])

    # arm: straight back, then down behind the ear
    a_top = yt - 0.5
    a_mid = a_top - p["arm_h"] / 2
    r = p["arm_h"] / 2
    end = (p["temple_len"], a_mid)
    ang = math.radians(p["drop_angle"])
    tip = (end[0] + p["drop_len"] * math.cos(ang), end[1] - p["drop_len"] * math.sin(ang))
    arm2d = (CrossSection.circle(r).translate((L - 8, a_mid)) + CrossSection.circle(r).translate(end)).hull()
    arm2d = arm2d + (CrossSection.circle(r).translate(end) + CrossSection.circle(r * 0.8).translate(tip)).hull()
    arm = prism(arm2d, 0, p["arm_t"])
    # smooth taper from the pod into the arm
    taper = Manifold.batch_hull([box(L - 14, yb + 3, 0, L - 3, yt - 0.5, p["arm_t"] + 4),
                                 box(L + 18, a_mid - r + 0.5, 0, L + 19, a_mid + r - 0.5, p["arm_t"])])
    body = body + arm + taper

    # hollow inside, open on the inner face (closed by the lid)
    x0, x1 = p["front_block"], L - p["rear_wall"]
    cavity = prism(rrect(x1 - x0, p["pod_in_h"], 1.5, (x0 + x1) / 2, yb + w + p["pod_in_h"] / 2), w, p["pod_t"] + 1)
    body = body - cavity
    # lid screw (M2) into the rear wall
    body = body - cyl_z(L - p["rear_wall"] / 2 - 0.2, yb + p["pod_h"] / 2, p["pod_t"] - 8, p["pod_t"] + 1, 1.7)
    # pins into the front block
    for (pz_x, pz_y) in p["pins"]:
        body = body - Manifold.cylinder(p["pin_depth"] + 1, p["pin_d"] / 2).rotate((0, 90, 0)).translate((-1, pz_y, pz_x))

    if side == "right":
        o0, o1 = p["opening"]
        # wiring and camera opening from the end piece into the pod
        body = body - box(-1, o0, w + 0.4, p["front_block"] + 1, o1, p["pod_t"] - 0.6)
        bx0 = p["front_block"] + 0.5
        bx1 = bx0 + p["board_w"]
        # USB-C port under the board
        body = body - box(bx0 + p["board_w"] / 2 - 6, yb - 1, w - 0.01, bx0 + p["board_w"] / 2 + 6, yb + w + 0.5, p["pod_t"] - 0.01)
        batt0 = bx1 + 2.5
        # push button on the outer face, beside the battery
        body = body - cyl_z(batt0 + 9, yb + p["pod_h"] / 2 + 3, -1, w + 1, 3.6)
        # power switch lever (SS12D00) through the bottom
        sx = batt0 + 22
        body = body - box(sx - 2.5, yb - 1, w + 2.0, sx + 2.5, yb + w + 0.5, w + 4.6)
    return body


def lid(p):
    """Flat on the bed, lip up."""
    L, w = p["pod_l"], p["wall"]
    x0, x1 = p["front_block"], L - p["rear_wall"]
    cy = p["pod_bottom"] + p["pod_h"] / 2
    plate = prism(rrect(L - 0.4, p["pod_h"] - 0.4, 2.8, L / 2, cy), 0, p["lid_t"])
    inner = rrect(x1 - x0 - 2 * p["fit"], p["pod_in_h"] - 2 * p["fit"], 1.3, (x0 + x1) / 2, cy)
    lip = prism(inner, p["lid_t"], p["lid_t"] + p["lid_lip"]) - prism(inner.offset(-1.2, JoinType.Round), p["lid_t"] - 0.01, p["lid_t"] + p["lid_lip"] + 1)
    part = plate + lip
    part = part - cyl_z(L - p["rear_wall"] / 2 - 0.2, cy, -1, p["lid_t"] + p["lid_lip"] + 1, 2.3)
    return part


# ---------- placing parts on the head (for the preview) ----------
def place_front(m):
    """Print orientation -> worn: x across, y backwards, z up."""
    return m.transform(np.array([[1, 0, 0, 0], [0, 0, 1, 0], [0, 1, 0, 0]], dtype=float))


def place_temple(m, p, side, lid_part=False):
    s = 1 if side == "right" else -1
    xo = s * (p["head_w"] / 2 + p["pod_tt"])
    # local x (along) -> worn y; local y (up) -> z; local z (inwards) -> towards the centre
    M = np.array([[0, 0, -s, xo], [1, 0, 0, p["end_d"]], [0, 1, 0, 0]], dtype=float)
    if lid_part:   # the lid sits on the inner face, lip pointing into the pod
        m = m.mirror((0, 0, 1)).translate((0, 0, p["pod_t"] + p["lid_t"]))
    return m.transform(M)


def electronics(p):
    """Stand-ins showing where each electronic part goes (preview only)."""
    s, xo = 1, p["head_w"] / 2 + p["pod_tt"]
    w, yb = p["wall"], p["pod_bottom"]
    def pod_box(x0, y0, z0, x1, y1, z1):   # pod-local coordinates -> worn
        return place_temple(box(x0, y0, z0, x1, y1, z1), p, "right")
    bx0 = p["front_block"] + 0.5
    parts = {
        "board": pod_box(bx0, yb + w + p["usb_room"], w + 0.2, bx0 + p["board_w"], yb + w + p["usb_room"] + p["board_l"], w + p["board_t"]),
        "battery": pod_box(bx0 + p["board_w"] + 2.5, yb + w + 1, p["pod_t"] - p["battery"][0] - 0.2,
                           bx0 + p["board_w"] + 2.5 + p["battery"][2] - 1, yb + w + 1 + p["battery"][1] - 0.5, p["pod_t"] - 0.2),
        "button": pod_box(bx0 + p["board_w"] + 2.5 + 6, yb + p["pod_h"] / 2, w, bx0 + p["board_w"] + 2.5 + 12, yb + p["pod_h"] / 2 + 6, w + 3.5),
        "switch": pod_box(bx0 + p["board_w"] + 2.5 + 22 - 4.3, yb + w, w + 2, bx0 + p["board_w"] + 2.5 + 22 + 4.3, yb + w + 3.6, w + 5.5),
    }
    cx = xo - p["cam_x"]
    c = p["cam"] / 2 - 0.3
    parts["camera"] = box(cx - c, p["cam_skin"], p["cam_z"] - c, cx + c, p["cam_skin"] + 4.5, p["cam_z"] + c)
    parts["led"] = Manifold.cylinder(5, 1.5).rotate((-90, 0, 0)).translate((cx, 0.2, p["led_z"]))
    return parts


# ---------- output ----------
def stl_bytes(m):
    """Binary STL of a manifold."""
    mesh = m.to_mesh()
    v = np.asarray(mesh.vert_properties)[:, :3].astype(np.float32)
    t = np.asarray(mesh.tri_verts)
    tri = v[t]
    n = np.cross(tri[:, 1] - tri[:, 0], tri[:, 2] - tri[:, 0])
    n /= np.maximum(np.linalg.norm(n, axis=1, keepdims=True), 1e-12)
    data = np.zeros(len(t), dtype=[("n", "<f4", 3), ("v", "<f4", (3, 3)), ("a", "<u2")])
    data["n"], data["v"] = n, tri
    return b"Lens glasses v1".ljust(80, b" ") + struct.pack("<I", len(t)) + data.tobytes()


def main():
    ap = argparse.ArgumentParser(description="Generate the Lens glasses frame")
    ap.add_argument("--head-width", type=float, default=132.0, help="distance between the inner faces of the temples at the front (mm)")
    ap.add_argument("--lens-width", type=float, default=50.0)
    ap.add_argument("--lens-height", type=float, default=38.0)
    ap.add_argument("--bridge", type=float, default=18.0)
    ap.add_argument("--temple-length", type=float, default=100.0, help="from the joint to the bend behind the ear (mm)")
    ap.add_argument("--board-thickness", type=float, default=12.0, help="XIAO + Sense board stack, camera removed (mm)")
    ap.add_argument("--out", default=str(HERE))
    args = ap.parse_args()
    p = params(args)

    out = Path(args.out)
    (out / "stl").mkdir(parents=True, exist_ok=True)
    (out / "preview").mkdir(parents=True, exist_ok=True)
    parts = {
        "front": front(p),
        # Modelled as a left-hand part; the right-hand parts are its mirror image.
        "temple_right": temple(p, "right").mirror((1, 0, 0)),
        "temple_left": temple(p, "left"),
        "lid_right": lid(p).mirror((1, 0, 0)),
        "lid_left": lid(p),
    }
    for name, m in parts.items():
        if m.status().name != "NoError" or m.is_empty():
            raise SystemExit(f"{name}: geometry error {m.status()}")
        # sit each part on the bed at z = 0
        lo = m.bounding_box()[2]
        (out / "stl" / f"{name}.stl").write_bytes(stl_bytes(m.translate((0, 0, -lo))))
        bb = m.bounding_box()
        print(f"{name:13s} {bb[3]-bb[0]:6.1f} x {bb[4]-bb[1]:6.1f} x {bb[5]-bb[2]:5.1f} mm  {m.volume()/1000:5.1f} cm3")

    placed = {
        "front": place_front(parts["front"]),
        "temple_right": place_temple(temple(p, "right"), p, "right"),
        "temple_left": place_temple(temple(p, "left"), p, "left"),
        "lid_right": place_temple(lid(p), p, "right", lid_part=True),
        "lid_left": place_temple(lid(p), p, "left", lid_part=True),
        **electronics(p),
    }
    # every part in place, as base64 STL in one file, for the blueprint page's 3D view
    bundle = {name: base64.b64encode(stl_bytes(m)).decode() for name, m in placed.items()}
    (out / "preview" / "model.json").write_text(json.dumps(bundle))
    print("wrote", out / "stl", "and", out / "preview" / "model.json")


if __name__ == "__main__":
    main()
