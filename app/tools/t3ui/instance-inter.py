"""instance-inter.py turns the Inter Variable files T3 Code ships into the
static faces tern embeds: weights 400, 500, 600 and 700, the ones T3 uses.

T3 ships Inter as unicode-range subsets, and Chrome shapes each as its own
font. Each weight's subsets are merged into one file here, because Gio
keys in-memory faces by family and weight, so same-weight subsets would
overwrite each other. Merging changes nothing visible: no subset kerns
against another.

    python3 -I tools/t3ui/instance-inter.py <T3 client assets dir> ui/fonts/inter
"""
import glob
import os
import sys
import tempfile

from fontTools.merge import Merger
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

assets, out = sys.argv[1], sys.argv[2]
os.makedirs(out, exist_ok=True)
# Latin first: for a code point in two subsets, the first one's glyph wins.
order = ["latin", "latin-ext", "cyrillic", "cyrillic-ext", "greek", "greek-ext", "vietnamese"]
with tempfile.TemporaryDirectory() as tmp:
    for weight, name in {400: "Regular", 500: "Medium", 600: "SemiBold", 700: "Bold"}.items():
        parts = []
        for subset in order:
            (path,) = glob.glob(os.path.join(assets, f"inter-{subset}-wght-normal-*.woff2"))
            font = TTFont(path)
            font.flavor = None
            part = os.path.join(tmp, f"{subset}.ttf")
            instantiateVariableFont(font, {"wght": weight}, updateFontNames=False).save(part)
            parts.append(part)
        Merger().merge(parts).save(os.path.join(out, f"Inter-{name}.ttf"))
