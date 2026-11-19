#!/usr/bin/env python3
"""Generate Safari's small, original sticker and GIF catalog with Pillow."""

import math
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).parent
STICKERS = ROOT / "stickers"
GIFS = ROOT / "gifs"
SIZE = 384
INK = (25, 30, 66, 255)
WHITE = (255, 255, 255, 255)
BLUE = (66, 174, 255, 255)
PURPLE = (135, 80, 255, 255)


def font(size):
    for candidate in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        "/usr/share/fonts/truetype/liberation2/LiberationSans-Bold.ttf",
    ):
        if Path(candidate).exists():
            return ImageFont.truetype(candidate, size)
    return ImageFont.load_default()


def gradient(size, left=BLUE, right=PURPLE):
    image = Image.new("RGBA", (size, size))
    draw = ImageDraw.Draw(image)
    for x in range(size):
        t = x / max(size - 1, 1)
        color = tuple(round(left[i] * (1 - t) + right[i] * t) for i in range(4))
        draw.line((x, 0, x, size), fill=color)
    return image


def heart(draw, cx, cy, radius, color):
    points = []
    for step in range(80):
        t = step * math.tau / 80
        x = 16 * math.sin(t) ** 3
        y = 13 * math.cos(t) - 5 * math.cos(2 * t) - 2 * math.cos(3 * t) - math.cos(4 * t)
        points.append((cx + x * radius / 32, cy - y * radius / 32))
    draw.polygon(points, fill=color)


def planet(expression="happy", size=SIZE, phase=0):
    image = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    scale = size / SIZE
    d = ImageDraw.Draw(image)
    c = lambda n: round(n * scale)

    # A bright, tilted orbit sits behind the ringed planet's face.
    ring = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    rd = ImageDraw.Draw(ring)
    rd.ellipse((c(24), c(146), c(360), c(250)), outline=(47, 183, 255, 255), width=c(17))
    rd.arc((c(24), c(146), c(360), c(250)), 185, 348, fill=(255, 255, 255, 245), width=c(8))
    ring = ring.rotate(-17 + phase, resample=Image.Resampling.BICUBIC)
    image.alpha_composite(ring)
    d = ImageDraw.Draw(image)

    # White sticker edge, then the blue-to-violet mascot.
    d.ellipse((c(48), c(46), c(336), c(338)), fill=WHITE)
    fill = gradient(size)
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).ellipse((c(61), c(59), c(323), c(325)), fill=255)
    image.alpha_composite(Image.composite(fill, Image.new("RGBA", (size, size)), mask))
    d = ImageDraw.Draw(image)
    d.ellipse((c(61), c(59), c(323), c(325)), outline=INK, width=c(7))

    # A small orbital glint is part of every face's Safari identity.
    d.ellipse((c(263), c(93), c(279), c(109)), fill=(255, 255, 255, 245))
    eye_y = 169
    if expression == "laughing":
        d.arc((c(112), c(eye_y - 4), c(168), c(eye_y + 24)), 200, 340, fill=INK, width=c(8))
        d.arc((c(216), c(eye_y - 4), c(272), c(eye_y + 24)), 200, 340, fill=INK, width=c(8))
    elif expression == "sad":
        d.line((c(119), c(eye_y - 3), c(164), c(eye_y + 4)), fill=INK, width=c(8))
        d.line((c(220), c(eye_y + 4), c(265), c(eye_y - 3)), fill=INK, width=c(8))
        d.ellipse((c(251), c(187), c(267), c(218)), fill=(91, 220, 255, 255))
    elif expression == "angry":
        d.line((c(116), c(eye_y - 12), c(163), c(eye_y + 2)), fill=INK, width=c(9))
        d.line((c(222), c(eye_y + 2), c(269), c(eye_y - 12)), fill=INK, width=c(9))
    elif expression == "wow":
        d.ellipse((c(126), c(eye_y - 5), c(151), c(eye_y + 28)), fill=INK)
        d.ellipse((c(231), c(eye_y - 5), c(256), c(eye_y + 28)), fill=INK)
    elif expression == "thinking":
        d.ellipse((c(127), c(eye_y), c(151), c(eye_y + 25)), fill=INK)
        d.ellipse((c(231), c(eye_y - 8), c(255), c(eye_y + 17)), fill=INK)
    elif expression == "love":
        heart(d, c(139), c(eye_y + 10), c(29), (255, 79, 143, 255))
        heart(d, c(243), c(eye_y + 10), c(29), (255, 79, 143, 255))
    else:
        d.ellipse((c(127), c(eye_y), c(151), c(eye_y + 25)), fill=INK)
        d.ellipse((c(231), c(eye_y), c(255), c(eye_y + 25)), fill=INK)

    if expression == "laughing":
        d.ellipse((c(151), c(209), c(233), c(276)), fill=(71, 22, 79, 255))
        d.arc((c(162), c(224), c(222), c(270)), 15, 165, fill=(255, 121, 164, 255), width=c(9))
    elif expression == "wow":
        d.ellipse((c(164), c(213), c(220), c(272)), fill=(59, 29, 89, 255))
        d.ellipse((c(178), c(220), c(202), c(238)), fill=(255, 255, 255, 255))
    elif expression == "thinking":
        d.arc((c(166), c(222), c(217), c(264)), 195, 335, fill=INK, width=c(8))
    elif expression == "sad":
        d.arc((c(157), c(236), c(225), c(282)), 20, 160, fill=INK, width=c(8))
    elif expression == "angry":
        d.line((c(164), c(239), c(220), c(239)), fill=INK, width=c(9))
    else:
        d.arc((c(153), c(208), c(231), c(271)), 15, 165, fill=INK, width=c(9))
        if expression == "love":
            d.ellipse((c(214), c(245), c(229), c(260)), fill=(255, 135, 184, 255))

    if expression == "thinking":
        d.ellipse((c(225), c(253), c(250), c(278)), fill=(255, 187, 111, 255), outline=INK, width=c(4))
        d.ellipse((c(247), c(271), c(260), c(284)), fill=(255, 187, 111, 255), outline=INK, width=c(3))
    if expression == "celebration":
        d.polygon([(c(123), c(76)), (c(244), c(91)), (c(195), c(20))], fill=(255, 192, 68, 255), outline=INK)
        d.ellipse((c(188), c(12), c(204), c(28)), fill=(255, 94, 135, 255))
        for x, y, color in ((72, 131, (255, 98, 137, 255)), (301, 147, (255, 205, 79, 255)), (85, 275, (91, 239, 191, 255))):
            d.ellipse((c(x), c(y), c(x + 13), c(y + 20)), fill=color)
    if expression == "good-night":
        d.text((c(248), c(77)), "Z", font=font(c(49)), fill=WHITE, stroke_width=c(2), stroke_fill=INK)
        d.text((c(287), c(55)), "z", font=font(c(31)), fill=WHITE, stroke_width=c(1), stroke_fill=INK)
    if expression == "thumbs-up":
        d.rounded_rectangle((c(246), c(207), c(283), c(273)), radius=c(14), fill=(255, 199, 110, 255), outline=INK, width=c(5))
        d.rounded_rectangle((c(268), c(169), c(293), c(226)), radius=c(12), fill=(255, 199, 110, 255), outline=INK, width=c(5))
        d.arc((c(255), c(220), c(282), c(261)), 100, 265, fill=INK, width=c(4))
    return image


def save_stickers():
    STICKERS.mkdir(parents=True, exist_ok=True)
    expressions = (
        ("safari-planet", "happy"),
        ("happy", "happy"),
        ("laughing", "laughing"),
        ("love", "love"),
        ("sad", "sad"),
        ("angry", "angry"),
        ("wow", "wow"),
        ("thinking", "thinking"),
        ("thumbs-up", "thumbs-up"),
        ("good-night", "good-night"),
        ("celebration", "celebration"),
    )
    for name, expression in expressions:
        planet(expression).save(STICKERS / f"{name}.webp", "WEBP", quality=84, method=6)


def gif_frames(name, count=12):
    frames = []
    for frame in range(count):
        t = frame / count
        image = Image.new("RGBA", (256, 256), (0, 0, 0, 0))
        d = ImageDraw.Draw(image)
        if name == "planet-spin":
            image = planet("happy", 256, phase=frame * 30)
        elif name == "heart-bounce":
            y = 116 + round(18 * math.sin(t * math.tau))
            heart(d, 128, y, 132, (255, 75, 135, 255))
            d.line((cxy(100), cxy(192), cxy(157), cxy(192)), fill=WHITE, width=4)
        elif name == "waving-planet":
            image = planet("happy", 256, phase=frame * 10)
            d = ImageDraw.Draw(image)
            swing = round(12 * math.sin(t * math.tau))
            d.line((194, 162, 222 + swing, 133), fill=INK, width=13)
            d.ellipse((216 + swing, 120, 239 + swing, 144), fill=(255, 199, 110, 255), outline=INK, width=4)
        elif name == "pulsing-hi":
            scale = 1 + 0.08 * math.sin(t * math.tau)
            radius = round(87 * scale)
            d.ellipse((128 - radius, 128 - radius, 128 + radius, 128 + radius), fill=(64, 174, 255, 255), outline=INK, width=7)
            label = "HI!"
            f = font(round(72 * scale))
            box = d.textbbox((0, 0), label, font=f)
            d.text((128 - (box[2] - box[0]) / 2, 128 - (box[3] - box[1]) / 2 - box[1]), label, font=f, fill=WHITE, stroke_width=2, stroke_fill=INK)
        elif name == "laughing-face":
            image = planet("laughing" if frame % 4 != 2 else "happy", 256)
        else:
            y = 176 - round(frame * 11)
            d.polygon([(119, y + 42), (154, y + 42), (179, y - 31), (152, y - 31)], fill=(108, 116, 255, 255), outline=INK)
            d.polygon([(153, y - 31), (179, y - 31), (166, y - 75)], fill=(255, 103, 133, 255), outline=INK)
            d.ellipse((126, y - 11, 132, y - 5), fill=WHITE)
            flame = 18 + (frame % 3) * 7
            d.polygon([(126, y + 40), (147, y + 40), (136, y + 40 + flame)], fill=(255, 178, 69, 255), outline=(251, 100, 71, 255))
            d.polygon([(130, y + 40), (143, y + 40), (136, y + 40 + flame // 2)], fill=(255, 235, 144, 255))
            d.ellipse((25, 42, 42, 59), fill=(255, 207, 84, 220))
            d.ellipse((209, 71, 221, 83), fill=(255, 116, 143, 220))
        frames.append(image.convert("P", palette=Image.Palette.ADAPTIVE, colors=128))
    return frames


def cxy(value):
    return value


def save_gifs():
    GIFS.mkdir(parents=True, exist_ok=True)
    for name in ("planet-spin", "heart-bounce", "waving-planet", "pulsing-hi", "laughing-face", "rocket-launch"):
        frames = gif_frames(name)
        frames[0].save(
            GIFS / f"{name}.gif",
            save_all=True,
            append_images=frames[1:],
            duration=100,
            loop=0,
            optimize=True,
            disposal=2,
            transparency=0,
        )


if __name__ == "__main__":
    save_stickers()
    save_gifs()
