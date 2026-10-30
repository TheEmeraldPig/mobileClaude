#!/usr/bin/env python3
"""Lens glasses, version 2: a parametric 3D-printable frame with folding hinges.

Parts: the front (lenses and nose pads), two temples (the arms that go over the ears) and two
lids. Each temple joins the front with a printed barrel hinge: two knuckles on the front, one on
the temple, and a 2.85 mm filament offcut as the pin. All the electronics live in the right
temple (camera and recording light in its front face, the XIAO board, battery, button and switch
behind them), so no wires cross the hinge. The left temple has the same shape, with an empty
pocket for a counterweight.

Everything is modelled where it sits when worn (x across the face, y backwards from the front
face, z up), then laid flat for printing. Outputs:
  stl/<part>.stl                  each part on its own, lying the way it prints
  stl/lens_glasses_plate.stl      all parts arranged on a 280 x 280 mm bed (LulzBot TAZ 6)
  preview/model.json              the parts in their worn positions plus the electronics, for blueprint.html

  pip install manifold3d numpy
  python3 frame.py [--head-width 134] [--board-thickness 11]
"""
import argparse
import base64
import json
import math
import struct
from pathlib import Path

import numpy as np
from manifold3d import CrossSection, JoinType, Manifold, set_circular_segments

set_circular_segments(56)
HERE = Path(__file__).resolve().parent
BED = 280.0


def params(a):
    p = dict(
        head_w=a.head_width,                   # between the temples' inner faces at the hinge
        lens_w=a.lens_width, lens_h=a.lens_height, lens_r=11.0, rim=5.0, bridge=a.bridge,
        front_t=5.5, lens_ledge=1.2, lens_rebate=1.8, end_d=8.0,
        # hinge
        knuckle_r=3.2, pin_hole=3.05, gap=0.4, kf_h=6.0, kt_h=7.0,
        # right temple pod: what goes inside, plus room for the wires between the parts
        wall=1.8, lid_t=1.6, lid_lip=1.2, fit=0.25,
        board=(21.0, 17.8, a.board_thickness),   # XIAO ESP32S3 + Sense board, camera unplugged: length, width, stack
        usb_room=1.5, wire_room=2.5,             # USB-C plug clearance below the board; a wire channel along the top
        battery=(5.2, 20.5, 31.0),               # 502030 LiPo with a little room: thickness, height, length
        gap_board_batt=3.0, gap_rear=3.0,         # wiring room around the battery
        front_block=9.0,                         # solid front of the temple: hinge knuckle, camera and light
        cam=9.6, cam_hole=7.4, cam_skin=1.2, cam_depth=7.0, led_hole=3.1,
        rear_wall=4.0,
        # arms
        arm_t=4.5, arm_h=7.0, temple_len=a.temple_length, drop_len=40.0, drop_angle=35.0,
    )
    b_len, b_w, b_t = p["board"]
    p["in_t"] = b_t + 1.2                                              # board stack + wires beside it
    p["body_t"] = p["wall"] + p["in_t"]                                # printed temple thickness (without lid)
    p["T"] = p["body_t"] + p["lid_t"]                                  # temple thickness when assembled
    p["in_h"] = b_len + p["usb_room"] + p["wire_room"]
    p["pod_h"] = p["in_h"] + 2 * p["wall"]
    p["front_h"] = p["lens_h"] + 2 * p["rim"]
    p["pod_top"] = p["front_h"] - 2.0
    p["pod_bot"] = p["pod_top"] - p["pod_h"]
    p["in_l"] = 0.5 + b_w + p["gap_board_batt"] + p["battery"][2] + p["gap_rear"]
    p["pod_l"] = p["front_block"] + p["in_l"] + p["rear_wall"]
    p["lens_cx"] = p["bridge"] / 2 + p["rim"] + p["lens_w"] / 2
    p["lens_cz"] = p["front_h"] / 2
    # hinge axis: vertical, on the temple's inner face, one knuckle radius behind the front
    p["ax"], p["ay"] = p["head_w"] / 2, p["end_d"] + p["knuckle_r"]
    zc = (p["pod_bot"] + p["pod_top"]) / 2
    g, kf, kt = p["gap"], p["kf_h"], p["kt_h"]
    p["k_lo"] = (zc - kt / 2 - g - kf, zc - kt / 2 - g)                 # front knuckle, bottom
    p["k_mid"] = (zc - kt / 2, zc + kt / 2)                             # temple knuckle
    p["k_hi"] = (zc + kt / 2 + g, zc + kt / 2 + g + kf)                 # front knuckle, top
    # camera and light in the temple's front face, outside the frame's end piece
    p["end_x1"] = p["ax"] + p["knuckle_r"] + 1.5                         # outer edge of the front's end piece
    p["cam_x"] = p["ax"] + p["T"] - p["wall"] - p["cam"] / 2 - 0.2
    p["cam_z"] = zc + 4.0
    p["led_z"] = p["cam_z"] - 10.0
    return p


# ---------- helpers ----------
def rrect(w, h, r, cx=0.0, cy=0.0):
    r = min(r, w / 2 - 0.01, h / 2 - 0.01)
    return CrossSection.square((w - 2 * r, h - 2 * r), center=True).offset(r, JoinType.Round).translate((cx, cy))


def box(x0, y0, z0, x1, y1, z1):
    return Manifold.cube((x1 - x0, y1 - y0, z1 - z0)).translate((x0, y0, z0))


def zcyl(x, y, z0, z1, r):
    """Vertical cylinder (along worn z)."""
    return Manifold.cylinder(z1 - z0, r).translate((x, y, z0))


def ycyl(x, z, y0, y1, r):
    """Cylinder along worn y (front to back), e.g. a camera window."""
    return Manifold.cylinder(y1 - y0, r).rotate((-90, 0, 0)).translate((x, y0, z))


def xz_prism(cs, y0, y1):
    """A shape drawn in the x-z plane (looking at the face), given depth from y0 to y1."""
    return Manifold.extrude(cs, y1 - y0).rotate((90, 0, 0)).translate((0, y1, 0))


def yz_prism(cs, x0, x1):
    """A shape drawn in the y-z plane (side view, cs x = worn y, cs y = worn z), given thickness from x0 to x1."""
    return Manifold.extrude(cs, x1 - x0).rotate((90, 0, 90)).translate((x0, 0, 0))


# ---------- hinge ----------
def front_knuckles(p, grow=0.0):
    """The front's two knuckles (right side): round barrels on the axis, joined to the end piece."""
    ax, ay, r = p["ax"], p["ay"], p["knuckle_r"] + grow
    out = Manifold()
    for z0, z1 in (p["k_lo"], p["k_hi"]):
        z0, z1 = z0 - grow, z1 + grow
        barrel = zcyl(ax, ay, z0, z1, r)
        web = box(ax - r, p["end_d"] - 0.5, z0, ax + r, ay, z1)
        out = out + Manifold.batch_hull([barrel, web])
    return out


def temple_knuckle(p):
    ax, ay, r = p["ax"], p["ay"], p["knuckle_r"]
    z0, z1 = p["k_mid"]
    barrel = zcyl(ax, ay, z0, z1, r)
    web = box(ax, ay, z0, ax + r + 1, ay + r + 2, z1)   # into the temple's front block
    return Manifold.batch_hull([barrel, web])


def pin_hole(p):
    return zcyl(p["ax"], p["ay"], p["k_lo"][0] - 1, p["k_hi"][1] + 1, p["pin_hole"] / 2)


# ---------- parts, worn position (right side; the left is a mirror image) ----------
def front(p):
    lens = rrect(p["lens_w"], p["lens_h"], p["lens_r"])
    rimmed = lens.offset(p["rim"], JoinType.Round)
    shape = CrossSection()
    for s in (-1, 1):
        shape = shape + rimmed.translate((s * p["lens_cx"], p["lens_cz"]))
    top = p["front_h"]
    shape = shape + rrect(p["bridge"] + 8, 12, 3, 0, top - 6)                      # bridge over the nose
    x_rim = p["lens_cx"] + p["lens_w"] / 2
    for s in (-1, 1):                                                               # joins rims to end pieces
        x0, x1 = sorted((s * (x_rim - 2), s * (p["ax"] + 2)))
        shape = shape + rrect(x1 - x0, 18, 3, (x0 + x1) / 2, p["pod_top"] - 10)
    body = xz_prism(shape, 0, p["front_t"])

    for s in (-1, 1):
        cx = s * p["lens_cx"]
        body = body - xz_prism(lens.translate((cx, p["lens_cz"])), -1, p["front_t"] + 1)
        ledge = lens.offset(p["lens_ledge"], JoinType.Round).translate((cx, p["lens_cz"]))
        body = body - xz_prism(ledge, p["front_t"] - p["lens_rebate"], p["front_t"] + 1)  # seat for optional lenses
        pad = box(-1.2, 0, -7, 1.2, 7, 7).rotate((0, 0, s * 28))
        body = body + pad.translate((s * (p["bridge"] / 2 + 1.8), p["front_t"] - 0.5, p["lens_cz"] - 10))

    # right end piece with its knuckles; the left one is the mirror image
    x0 = p["ax"] - p["knuckle_r"] - 1.5
    end = box(x0, 0, p["pod_bot"], p["end_x1"], p["end_d"], p["pod_top"])
    side = end + front_knuckles(p)
    zt0, zt1 = p["k_mid"]
    side = side - zcyl(p["ax"], p["ay"], zt0 - p["gap"], zt1 + p["gap"], p["knuckle_r"] + p["gap"])  # room for the temple's knuckle
    side = side - pin_hole(p)
    return body + side + side.mirror((1, 0, 0))


def temple(p, electronics=True):
    ax, ay, w = p["ax"], p["ay"], p["wall"]
    xi, xo = ax, ax + p["T"]                    # inner face (lid side) and outer face
    xb = ax + p["lid_t"]                        # where the printed body starts (the lid covers the rest)
    y0, L = p["end_d"], p["pod_l"]
    zb, zt = p["pod_bot"], p["pod_top"]
    pod = rrect(L, p["pod_h"], 3.0, y0 + L / 2, zb + p["pod_h"] / 2)
    body = yz_prism(pod, xb, xo)
    body = body + box(xi, y0, zb + 0.6, xb + 0.01, y0 + p["front_block"], zt - 0.6)   # front block reaches the inner face

    # arm: straight back, then down behind the ear; set in from the outer face so it rests on the head
    r = p["arm_h"] / 2
    a_mid = zt - 1.0 - r
    end = (p["temple_len"] + y0, a_mid)
    ang = math.radians(p["drop_angle"])
    tip = (end[0] + p["drop_len"] * math.cos(ang), end[1] - p["drop_len"] * math.sin(ang))
    arm2d = (CrossSection.circle(r).translate((y0 + L - 8, a_mid)) + CrossSection.circle(r).translate(end)).hull()
    arm2d = arm2d + (CrossSection.circle(r).translate(end) + CrossSection.circle(r * 0.8).translate(tip)).hull()
    ax0, ax1 = xb + 3.5, xb + 3.5 + p["arm_t"]
    arm = yz_prism(arm2d, ax0, ax1)
    taper = Manifold.batch_hull([box(ax0, y0 + L - 14, zb + 4, xo, y0 + L - 3, zt - 0.5),
                                 box(ax0, y0 + L + 20, a_mid - r + 0.5, ax1, y0 + L + 21, a_mid + r - 0.5)])
    body = body + arm + taper

    # hinge: the temple's knuckle, minus room for the front's knuckles to turn in
    body = body + temple_knuckle(p)
    body = body - front_knuckles(p, grow=p["gap"])
    body = body - pin_hole(p)

    # the pocket, open on the inner face (closed by the lid)
    c0, c1 = y0 + p["front_block"], y0 + L - p["rear_wall"]
    cavity = rrect(c1 - c0, p["in_h"], 1.5, (c0 + c1) / 2, zb + w + p["in_h"] / 2)
    body = body - yz_prism(cavity, xb - 1, xo - w)
    body = body - box(xb - 1, c1 + p["rear_wall"] / 2 - 0.85, zb + p["pod_h"] / 2 - 0.85,
                      xb + 8, c1 + p["rear_wall"] / 2 + 0.85, zb + p["pod_h"] / 2 + 0.85)   # M2 lid screw (square pilot)

    if electronics:
        cx, c = p["cam_x"], p["cam"] / 2
        # camera: pocket from the inside, round window in the front face
        body = body - box(cx - c, y0 + p["cam_skin"], p["cam_z"] - c, cx + c, c0 + 1, p["cam_z"] + c)
        body = body - ycyl(cx, p["cam_z"], y0 - 1, y0 + p["cam_skin"] + 0.5, p["cam_hole"] / 2)
        # recording light, below the camera, wires back into the pocket
        body = body - ycyl(cx, p["led_z"], y0 - 1, c0 + 1, p["led_hole"] / 2)
        body = body - ycyl(cx, p["led_z"], y0 + 1.6, c0 + 1, 1.9)
        bx0 = c0 + 0.5
        bmid = bx0 + p["board"][1] / 2
        # USB-C port under the board, so it can be charged with the glasses on the table
        body = body - box(xb + 0.5, bmid - 6, zb - 1, xo - w, bmid + 6, zb + w + 0.5)
        batt0 = bx0 + p["board"][1] + p["gap_board_batt"]
        # push button on the outer face, beside the battery; power switch lever through the bottom
        body = body - Manifold.cylinder(w + 2, 1.8).rotate((0, 90, 0)).translate((xo - w - 1, batt0 + 9, zb + p["pod_h"] / 2 + 3))
        sx = batt0 + 22
        body = body - box(xo - w - 4.2, sx - 2.6, zb - 1, xo - w - 1.6, sx + 2.6, zb + w + 0.5)
    return body


def lid(p):
    """Covers the pocket on the temple's inner face; its rim locates it."""
    ax, y0, L, w = p["ax"], p["end_d"], p["pod_l"], p["wall"]
    zb = p["pod_bot"]
    c0, c1 = y0 + p["front_block"], y0 + L - p["rear_wall"]
    cy = zb + p["pod_h"] / 2
    plate = yz_prism(rrect(L - p["front_block"] - 0.4, p["pod_h"] - 0.4, 2.8, c0 + (L - p["front_block"]) / 2, cy),
                     ax, ax + p["lid_t"])
    inner = rrect(c1 - c0 - 2 * p["fit"], p["in_h"] - 2 * p["fit"], 1.3, (c0 + c1) / 2, cy)
    lip = yz_prism(inner, ax + p["lid_t"] - 0.01, ax + p["lid_t"] + p["lid_lip"]) - \
        yz_prism(inner.offset(-1.2, JoinType.Round), ax + p["lid_t"] - 0.1, ax + p["lid_t"] + p["lid_lip"] + 1)
    part = plate + lip
    sy = c1 + p["rear_wall"] / 2
    part = part - Manifold.cylinder(10, 1.15).rotate((0, 90, 0)).translate((ax - 1, sy, cy))
    # the screw hole lands on the rear wall, past the lip
    return part


def electronics(p):
    """Stand-ins for the electronics, where they sit inside the right temple (preview only)."""
    ax, y0, w, zb = p["ax"], p["end_d"], p["wall"], p["pod_bot"]
    xb, xo = ax + p["lid_t"], ax + p["T"]
    c0 = y0 + p["front_block"]
    bx0 = c0 + 0.5
    bl, bw, bt = p["board"]
    batt0 = bx0 + bw + p["gap_board_batt"]
    bt_, bh, bl_ = p["battery"]
    c = p["cam"] / 2 - 0.3
    return {
        "board": box(xo - w - bt, bx0, zb + w + p["usb_room"], xo - w - 0.1, bx0 + bw, zb + w + p["usb_room"] + bl),
        "battery": box(xb + p["lid_lip"] + 0.3, batt0, zb + w + 0.5, xb + p["lid_lip"] + 0.3 + bt_ - 0.2, batt0 + bl_ - 1, zb + w + 0.5 + bh - 0.5),
        "button": box(xo - w - 3.5, batt0 + 6, zb + p["pod_h"] / 2, xo - w, batt0 + 12, zb + p["pod_h"] / 2 + 6),
        "switch": box(xo - w - 5.5, batt0 + 22 - 4.3, zb + w, xo - w - 2, batt0 + 22 + 4.3, zb + w + 3.6),
        "camera": box(p["cam_x"] - c, y0 + p["cam_skin"], p["cam_z"] - c, p["cam_x"] + c, y0 + p["cam_skin"] + 5.5, p["cam_z"] + c),
        "led": ycyl(p["cam_x"], p["led_z"], y0 + 0.3, y0 + 5.5, 1.5),
        "pins": pin_hole(p) + pin_hole(p).mirror((1, 0, 0)),
    }


# ---------- laying parts flat for printing ----------
def on_bed(m):
    """Moves a part so it sits on z = 0 with its corner at x = y = 0."""
    b = m.bounding_box()
    return m.translate((-b[0], -b[1], -b[2]))


def to_print(parts):
    face_down = parts["front"].rotate((90, 0, 0))                 # front face on the bed
    return {
        "front": on_bed(face_down),
        # outer face on the bed, pocket up, turned so the length runs across the bed
        "temple_right": on_bed(parts["temple_right"].rotate((0, 90, 0)).rotate((0, 0, 90))),
        "temple_left": on_bed(parts["temple_left"].rotate((0, -90, 0)).rotate((0, 0, 90))),
        "lid_right": on_bed(parts["lid_right"].rotate((0, -90, 0)).rotate((0, 0, 90))),   # flat, rim up
        "lid_left": on_bed(parts["lid_left"].rotate((0, 90, 0)).rotate((0, 0, 90))),
    }


def plate(flat):
    """All parts on one 280 x 280 mm bed, 8 mm apart, centred."""
    def size(m):
        b = m.bounding_box()
        return b[3] - b[0], b[4] - b[1]

    rows = [["front"], ["temple_right"], ["temple_left"], ["lid_right", "lid_left"]]
    placed, y = [], 0.0
    for row in rows:
        x, h = 0.0, 0.0
        for name in row:
            w, d = size(flat[name])
            placed.append(flat[name].translate((x, y, 0)))
            x, h = x + w + 8, max(h, d)
        y += h + 8
    allp = Manifold.compose(placed)
    b = allp.bounding_box()
    w, d = b[3] - b[0], b[4] - b[1]
    if w > BED - 10 or d > BED - 10:
        raise SystemExit(f"plate is {w:.0f} x {d:.0f} mm, too big for the {BED:.0f} mm bed")
    return allp.translate(((BED - w) / 2 - b[0], (BED - d) / 2 - b[1], 0))


# ---------- output ----------
def stl_bytes(m):
    mesh = m.to_mesh()
    v = np.asarray(mesh.vert_properties)[:, :3].astype(np.float32)
    t = np.asarray(mesh.tri_verts)
    tri = v[t]
    n = np.cross(tri[:, 1] - tri[:, 0], tri[:, 2] - tri[:, 0])
    n /= np.maximum(np.linalg.norm(n, axis=1, keepdims=True), 1e-12)
    data = np.zeros(len(t), dtype=[("n", "<f4", 3), ("v", "<f4", (3, 3)), ("a", "<u2")])
    data["n"], data["v"] = n, tri
    return b"Lens glasses v2".ljust(80, b" ") + struct.pack("<I", len(t)) + data.tobytes()


def check(p, parts):
    """Fit checks: the hinge parts must not touch, and the temple must swing 90 degrees inward."""
    problems = []
    fr, tr = parts["front"], parts["temple_right"]
    if (fr ^ tr).volume() > 0.5:
        problems.append(f"front and right temple overlap by {(fr ^ tr).volume():.2f} mm3 when open")
    for deg in (30, 60, 90):
        moved = tr.translate((-p["ax"], -p["ay"], 0)).rotate((0, 0, deg)).translate((p["ax"], p["ay"], 0))
        clash = (fr ^ moved).volume()
        if clash > 0.5:
            problems.append(f"right temple hits the front when folded {deg} degrees ({clash:.1f} mm3)")
    e = electronics(p)
    inside = tr + parts["lid_right"]
    for name in ("board", "battery", "camera", "button", "switch"):
        v = (inside ^ e[name]).volume()
        if v > 0.5:
            problems.append(f"{name} hits the temple ({v:.1f} mm3)")
    return problems


def main():
    ap = argparse.ArgumentParser(description="Generate the Lens glasses frame (v2, folding)")
    ap.add_argument("--head-width", type=float, default=134.0, help="between the temples' inner faces at the hinge (mm)")
    ap.add_argument("--lens-width", type=float, default=50.0)
    ap.add_argument("--lens-height", type=float, default=40.0)
    ap.add_argument("--bridge", type=float, default=18.0)
    ap.add_argument("--temple-length", type=float, default=100.0, help="from the hinge to the bend behind the ear (mm)")
    ap.add_argument("--board-thickness", type=float, default=11.0, help="XIAO + Sense board stack with the camera unplugged (mm)")
    ap.add_argument("--out", default=str(HERE))
    p = params(ap.parse_args())

    worn = {"front": front(p), "temple_right": temple(p, True), "lid_right": lid(p)}
    worn["temple_left"] = temple(p, False).mirror((1, 0, 0))
    worn["lid_left"] = worn["lid_right"].mirror((1, 0, 0))
    for name, m in worn.items():
        if m.status().name != "NoError" or m.is_empty():
            raise SystemExit(f"{name}: geometry error {m.status()}")

    problems = check(p, worn)
    for msg in problems:
        print("FIT PROBLEM:", msg)

    out = Path(__file__).resolve().parent if not ap.parse_args().out else Path(ap.parse_args().out)
    (out / "stl").mkdir(parents=True, exist_ok=True)
    (out / "preview").mkdir(parents=True, exist_ok=True)
    for old in (out / "stl").glob("*.stl"):
        old.unlink()
    flat = to_print(worn)
    for name, m in flat.items():
        (out / "stl" / f"{name}.stl").write_bytes(stl_bytes(m))
        b = m.bounding_box()
        print(f"{name:13s} {b[3]-b[0]:6.1f} x {b[4]-b[1]:6.1f} x {b[5]-b[2]:5.1f} mm  {m.volume()/1000:5.1f} cm3")
    pl = plate(flat)
    (out / "stl" / "lens_glasses_plate.stl").write_bytes(stl_bytes(pl))
    b = pl.bounding_box()
    print(f"plate         {b[3]-b[0]:6.1f} x {b[4]-b[1]:6.1f} mm on the {BED:.0f} mm bed")

    shown = {**worn, **electronics(p)}
    bundle = {name: base64.b64encode(stl_bytes(m)).decode() for name, m in shown.items()}
    (out / "preview" / "model.json").write_text(json.dumps(bundle))
    if problems:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
