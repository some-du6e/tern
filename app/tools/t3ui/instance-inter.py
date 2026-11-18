"""instance-inter.py turns the Inter Variable files T3 Code ships into the
static faces tern embeds: each unicode-range subset at weights 400, 500, 600
and 700, the weights T3 uses. The subsets stay separate fonts, as in Chrome,
which shapes each one on its own.

    python3 -I tools/t3ui/instance-inter.py <T3 client assets dir> ui/fonts/inter
"""
import glob
import os
import sys

from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

assets, out = sys.argv[1], sys.argv[2]
os.makedirs(out, exist_ok=True)
for path in sorted(glob.glob(os.path.join(assets, "inter-*-wght-normal-*.woff2"))):
    subset = os.path.basename(path).split("-wght")[0][len("inter-"):]
    for weight, name in {400: "Regular", 500: "Medium", 600: "SemiBold", 700: "Bold"}.items():
        font = TTFont(path)
        font.flavor = None
        instantiateVariableFont(font, {"wght": weight}, updateFontNames=False).save(
            os.path.join(out, f"Inter-{name}-{subset}.ttf"))
