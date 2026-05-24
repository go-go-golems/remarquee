#!/usr/bin/env python3
"""Generate a minimal valid PNG for testing image embedding."""
import struct, zlib, sys

def make_png(w, h, r, g, b):
    raw = b''
    for y in range(h):
        raw += b'\x00'  # filter: None
        for x in range(w):
            raw += bytes([r, g, b])
    compressed = zlib.compress(raw, 9)
    sig = b'\x89PNG\r\n\x1a\n'
    def chunk(ctype, data):
        c = ctype + data
        crc = struct.pack('>I', zlib.crc32(c) & 0xffffffff)
        return struct.pack('>I', len(data)) + c + crc
    ihdr = chunk(b'IHDR', struct.pack('>IIBBBBB', w, h, 8, 2, 0, 0, 0))
    idat = chunk(b'IDAT', compressed)
    iend = chunk(b'IEND', b'')
    return sig + ihdr + idat + iend

if __name__ == '__main__':
    out = sys.argv[1] if len(sys.argv) > 1 else '/dev/stdout'
    with open(out, 'wb') as f:
        f.write(make_png(4, 4, 0x87, 0xCE, 0xEB))
