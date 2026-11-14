#!/usr/bin/env python3
"""Paint out the Flutter DEBUG ribbon from screenshot PNGs.

The ribbon is a dark-red diagonal band in the top-right corner of the
app bar (x>=336, y<=53, x+y in [336,442]) with white anti-aliased
"DEBUG" text on it. It is a debug-build artifact, not app content:
only the ribbon pixels are replaced, with the app bar's own horizontal
gradient (fitted from a clean stretch of the header), and only inside
the app bar (y<56) so the light content below is never touched.
The settings-gear icon (x<344) is preserved.
"""
import sys

import numpy as np
from PIL import Image


def clean(path):
    im = Image.open(path).convert('RGB')
    px = im.load()
    w, h = im.size
    # Fit the header gradient from row y=48 (bottom of the app bar:
    # no title text or icons there), excluding any bright outliers.
    fit_y = 48
    xs, rs, gs, bs = [], [], [], []
    for x in range(70, 221):
        r, g, b = px[x, fit_y]
        if min(r, g, b) > 200:  # stray bright pixel, not gradient
            continue
        xs.append(float(x)); rs.append(float(r)); gs.append(float(g)); bs.append(float(b))
    xs = np.array(xs)
    coefs = [np.polyfit(xs, np.array(ch), 1) for ch in (rs, gs, bs)]

    def grad(x):
        return tuple(int(round(min(255, max(0, np.polyval(cf, x)))))
                     for cf in coefs)

    replaced = 0
    for y in range(min(h, 56)):  # inside the app bar only
        for x in range(w):
            if not (x >= 336 and 336 <= x + y <= 442):
                continue  # outside the ribbon band
            r, g, b = px[x, y]
            # The settings-gear icon (white, x<344) sits just left of the
            # ribbon: never touch it.
            if x < 344 and min(r, g, b) > 200:
                continue
            # Anything that is not the header gradient itself is ribbon
            # (red band, white DEBUG text, or their anti-aliased edges).
            gr, gg, gb = grad(x)
            if abs(r - gr) > 8 or abs(g - gg) > 8 or abs(b - gb) > 8:
                px[x, y] = (gr, gg, gb)
                replaced += 1
    im.save(path)
    print(path, 'replaced', replaced, 'px')


if __name__ == '__main__':
    for p in sys.argv[1:]:
        clean(p)
