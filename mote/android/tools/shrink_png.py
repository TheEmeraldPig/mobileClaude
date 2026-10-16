# Prints a PNG screenshot as small raw RGB (base64) so CI logs can show what the
# Android emulator displayed. Usage: python3 shrink_png.py shot.png [width]
import base64, struct, sys, zlib

def read_png(path):
    data = open(path, 'rb').read()
    pos, idat, w = 8, b'', 0
    while pos < len(data):
        n, kind = struct.unpack('>I4s', data[pos:pos + 8])
        body = data[pos + 8:pos + 8 + n]
        if kind == b'IHDR':
            w, h, depth, ctype = struct.unpack('>IIBB', body[:10])
        elif kind == b'IDAT':
            idat += body
        pos += 12 + n
    bpp = {2: 3, 6: 4}[ctype]
    raw, stride = zlib.decompress(idat), w * bpp
    rows, prev, i = [], bytearray(stride), 0
    for _ in range(h):
        f, line = raw[i], bytearray(raw[i + 1:i + 1 + stride])
        i += 1 + stride
        for x in range(stride):
            a = line[x - bpp] if x >= bpp else 0
            b, c = prev[x], prev[x - bpp] if x >= bpp else 0
            if f == 1: line[x] = (line[x] + a) & 255
            elif f == 2: line[x] = (line[x] + b) & 255
            elif f == 3: line[x] = (line[x] + (a + b) // 2) & 255
            elif f == 4:
                p = a + b - c
                pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
                line[x] = (line[x] + (a if pa <= pb and pa <= pc else b if pb <= pc else c)) & 255
        rows.append(line)
        prev = line
    return w, h, bpp, rows

w, h, bpp, rows = read_png(sys.argv[1])
ow = int(sys.argv[2]) if len(sys.argv) > 2 else 160
oh = max(1, ow * h // w)
out = bytearray()
for y in range(oh):
    r = rows[y * h // oh]
    for x in range(ow):
        o = (x * w // ow) * bpp
        out += r[o:o + 3]
print(f"{ow}x{oh} (from {w}x{h})")
print(base64.b64encode(out).decode())
