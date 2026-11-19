"""Renders the send button's nightly artwork into ui/art.

T3's send button on a nightly build shows the sidebar's nightly sky
(SidebarStageBackdrop.tsx, NightlySkyArt compact): the SVG's viewBox
starts 96 units in and slices to the button, so a 32dp button shows units
96 to 192. This writes that SVG with the default palette (index.css, the
same in light and dark) resolved to sRGB and has rsvg-convert render it at
1x and 2x:

    python3 tools/t3ui/sendart.py ui/art
"""
import math
import pathlib
import subprocess
import sys

# index.css :root, the default nightly palette.
PALETTE = {
    "top": (0.283792, 0.117327, 297.201),
    "mid": (0.227147, 0.086086, 277.99),
    "bottom": (0.200528, 0.055699, 261.216),
    "highlight": (0.707246, 0.157418, 252.091),
    "secondary": (0.600473, 0.182225, 277.296),
    "tertiary": (0.62583, 0.210886, 305.994),
    "line": (0.938794, 0.029114, 273.103),
    "glow-highlight": (0.553749, 0.176543, 271.958),
    "glow-secondary": (0.345571, 0.117466, 273.568),
    "sparkle": (0.880867, 0.057747, 269.011),
}

STARS = [(14, 10, 0.6, 0.85), (38, 22, 0.4, 0.55), (58, 8, 0.5, 0.7), (84, 16, 0.4, 0.5), (104, 7, 0.6, 0.8),
         (126, 20, 0.4, 0.55), (148, 11, 0.5, 0.7), (170, 24, 0.4, 0.5), (192, 9, 0.6, 0.8), (214, 18, 0.4, 0.55),
         (236, 8, 0.5, 0.7), (258, 20, 0.45, 0.6), (278, 11, 0.55, 0.75), (26, 34, 0.4, 0.45), (118, 34, 0.4, 0.45),
         (202, 32, 0.4, 0.5), (268, 34, 0.4, 0.45)]
SPARKLES = [(70, 28), (160, 36), (246, 26)]


def oklch_hex(l, c, h):
    a, b = c * math.cos(math.radians(h)), c * math.sin(math.radians(h))
    l_ = (l + 0.3963377774 * a + 0.2158037573 * b) ** 3
    m_ = (l - 0.1055613458 * a - 0.0638541728 * b) ** 3
    s_ = (l - 0.0894841775 * a - 1.2914855480 * b) ** 3
    rgb = (4.0767416621 * l_ - 3.3077115913 * m_ + 0.2309699292 * s_,
           -1.2684380046 * l_ + 2.6097574011 * m_ - 0.3413193965 * s_,
           -0.0041960863 * l_ - 0.7034186147 * m_ + 1.7076147010 * s_)

    def enc(x):
        x = min(1, max(0, x))
        x = 12.92 * x if x <= 0.0031308 else 1.055 * x ** (1 / 2.4) - 0.055
        return round(x * 255)

    return "#%02x%02x%02x" % tuple(enc(x) for x in rgb)


def svg():
    c = {k: oklch_hex(*v) for k, v in PALETTE.items()}
    stars = "".join(f'<circle cx="{x}" cy="{y}" r="{r}" fill-opacity="{o}"/>' for x, y, r, o in STARS)
    sparkles = "".join(f'<path d="M{x - 1.5} {y}H{x + 1.5}"/><path d="M{x} {y - 1.5}V{y + 1.5}"/>' for x, y in SPARKLES)
    return f'''<svg xmlns="http://www.w3.org/2000/svg" fill="none" preserveAspectRatio="xMinYMin slice" viewBox="96 0 96 96">
<defs>
<linearGradient id="sky" x1="24" y1="0" x2="264" y2="96" gradientUnits="userSpaceOnUse" spreadMethod="reflect">
<stop stop-color="{c['bottom']}"/><stop offset="0.5" stop-color="{c['mid']}"/><stop offset="1" stop-color="{c['top']}"/>
</linearGradient>
<radialGradient id="glow" cx="0" cy="0" r="1" gradientTransform="translate(216 18) rotate(137) scale(120 84)" gradientUnits="userSpaceOnUse">
<stop stop-color="{c['glow-highlight']}" stop-opacity="0.4"/><stop offset="0.5" stop-color="{c['glow-secondary']}" stop-opacity="0.16"/>
<stop offset="1" stop-color="{c['bottom']}" stop-opacity="0"/>
</radialGradient>
<linearGradient id="cloud" x1="0" y1="60" x2="288" y2="96" gradientUnits="userSpaceOnUse">
<stop stop-color="{c['highlight']}" stop-opacity="0.5"/><stop offset="0.52" stop-color="{c['secondary']}" stop-opacity="0.62"/>
<stop offset="1" stop-color="{c['tertiary']}" stop-opacity="0.5"/>
</linearGradient>
<filter id="soft" x="-24" y="-24" width="336" height="144" filterUnits="userSpaceOnUse"><feGaussianBlur stdDeviation="4"/></filter>
<pattern id="stars" width="288" height="96" patternUnits="userSpaceOnUse">
<g fill="{c['line']}">{stars}</g>
<g stroke="{c['sparkle']}" stroke-linecap="round" stroke-opacity="0.7" stroke-width="0.6">{sparkles}</g>
</pattern>
<pattern id="glows" width="640" height="96" patternUnits="userSpaceOnUse"><rect width="640" height="96" fill="url(#glow)"/></pattern>
</defs>
<rect width="8192" height="96" fill="url(#sky)"/>
<rect width="8192" height="96" fill="url(#glows)"/>
<rect width="8192" height="96" fill="url(#stars)"/>
<g filter="url(#soft)"><path d="M-12 88C-12 74 0 63 14 63C18 50 30 41 44 41C58 41 70 49 74 62C79 57 86 54 94 54C110 54 123 66 124 82C132 83 138 88 141 96H-12V88Z" fill="url(#cloud)"/></g>
<g filter="url(#soft)"><path d="M150 96C151 84 161 75 173 75C176 64 186 57 198 57C210 57 220 64 223 75C231 75 238 80 241 87C250 87 257 91 260 96H150Z" fill="url(#cloud)" fill-opacity="0.8"/></g>
</svg>
'''


def main():
    out = pathlib.Path(sys.argv[1])
    src = out / "send-night.svg"
    src.write_text(svg())
    try:
        for scale in (1, 2):
            px = str(32 * scale)
            subprocess.run(["rsvg-convert", "-w", px, "-h", px, "-o", str(out / f"send-night@{scale}x.png"), str(src)], check=True)
    finally:
        src.unlink()


main()
