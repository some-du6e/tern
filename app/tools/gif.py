"""Stitch cmd/mockup's animation frames into a GIF: gif.py FRAMES_DIR OUT.gif"""
import pathlib
import sys

from PIL import Image

src, out = pathlib.Path(sys.argv[1]), sys.argv[2]
durations = [int(x) for x in (src / "durations.txt").read_text().split()]
frames = [Image.open(p).convert("RGB") for p in sorted(src.glob("*.png"))]
# One adaptive palette from the first lit frame keeps the dark panel flat.
palette = frames[len(frames) // 2].quantize(colors=255, method=Image.Quantize.MEDIANCUT)
frames = [f.quantize(palette=palette, dither=Image.Dither.NONE) for f in frames]
frames[0].save(out, save_all=True, append_images=frames[1:], duration=durations, loop=0, optimize=False, disposal=1)
