"""Copies the icons T3's diff viewer and file trees draw out of its web
client into ui/pierre.go: Pierre's sprites (@pierre/diffs, @pierre/trees,
Apache-2.0, Pierre Computer Company), T3's own file icons
(pierre-icons.ts), and the tables that pick a file's icon by its name.

Pierre's icons are filled shapes on a 16x16 grid, some in layers at an
opacity; T3's are 24x24 or 32x32, filled or stroked, some in fixed
light-dark colours. Every shape becomes SVG path data.

    python3 tools/pierre/gen.py <t3 client assets dir> > ui/pierre.go
"""
import pathlib
import re
import sys
import xml.etree.ElementTree as ET

# The diff viewer's icons tern draws (§2.12): change types, the rename
# arrow, hunk expanders and the gutter's comment button.
DIFFS = ["diffs-icon-symbol-modified", "diffs-icon-symbol-added", "diffs-icon-symbol-deleted",
         "diffs-icon-symbol-moved", "diffs-icon-file-code", "diffs-icon-arrow-right-short",
         "diffs-icon-expand", "diffs-icon-expand-all", "diffs-icon-plus"]

assets = pathlib.Path(sys.argv[1])


def bundle(prefix):
    files = sorted(assets.glob(prefix + "-*.js"))
    if not files:
        raise SystemExit(f"no {prefix} bundle")
    return files[0].read_text(errors="replace")


def unescape(s):
    return s.replace('\\"', '"').replace("\\n", "\n").replace("\\t", "\t")


def symbols(text):
    out = {}
    for m in re.finditer(r'<symbol id="([a-z0-9-]+)"[^>]*viewBox="0 0 (\d+) (\d+)"[^>]*>(.*?)</symbol>',
                         unescape(text), re.S):
        out[m.group(1)] = (int(m.group(2)), m.group(4))
    return out


def colour(v):
    """A fill or stroke value: None for currentColor, else (light, dark)."""
    v = (v or "").strip()
    m = re.fullmatch(r"var\(--[\w-]+,\s*(.*)\)", v)
    if m:
        return colour(m.group(1))
    m = re.fullmatch(r"light-dark\(\s*#([0-9a-fA-F]{6})\s*,\s*#([0-9a-fA-F]{6})\s*\)", v)
    if m:
        return (m.group(1).lower(), m.group(2).lower())
    m = re.fullmatch(r"#([0-9a-fA-F]{6})", v)
    if m:
        return (m.group(1).lower(), m.group(1).lower())
    if v.lower() in ("", "currentcolor"):
        return None
    raise SystemExit(f"unknown colour {v!r}")


def style(el):
    st = dict(el.attrib)
    for decl in el.attrib.get("style", "").split(";"):
        if ":" in decl:
            k, v = decl.split(":", 1)
            st[k.strip()] = v.strip()
    return st


def num(v):
    return float(v)


def path_data(tag, a):
    if tag == "path":
        # A path's first move is absolute even when written "m".
        m = re.match(r"\s*m\s*(-?[\d.]+)[\s,]*(-?[\d.]+)\s*(.*)", a["d"], re.S)
        if not m:
            return a["d"].strip()
        rest = m.group(3)
        if rest and (rest[0].isdigit() or rest[0] in "-."):
            rest = "l" + rest
        return f"M{m.group(1)} {m.group(2)}{rest}"
    if tag == "circle":
        cx, cy, r = num(a["cx"]), num(a["cy"]), num(a["r"])
        return f"M{cx-r:g} {cy:g}a{r:g} {r:g} 0 1 0 {2*r:g} 0a{r:g} {r:g} 0 1 0 {-2*r:g} 0Z"
    if tag == "rect":
        x, y, w, h = num(a.get("x", 0)), num(a.get("y", 0)), num(a["width"]), num(a["height"])
        rx = num(a.get("rx", a.get("ry", 0)))
        ry = num(a.get("ry", rx))
        if rx == 0:
            return f"M{x:g} {y:g}h{w:g}v{h:g}h{-w:g}Z"
        return (f"M{x+rx:g} {y:g}H{x+w-rx:g}A{rx:g} {ry:g} 0 0 1 {x+w:g} {y+ry:g}V{y+h-ry:g}"
                f"A{rx:g} {ry:g} 0 0 1 {x+w-rx:g} {y+h:g}H{x+rx:g}A{rx:g} {ry:g} 0 0 1 {x:g} {y+h-ry:g}"
                f"V{y+ry:g}A{rx:g} {ry:g} 0 0 1 {x+rx:g} {y:g}Z")
    raise SystemExit(f"unknown shape {tag}")


def shapes(name, body):
    root = ET.fromstring(f'<g xmlns="http://www.w3.org/2000/svg">{body}</g>')
    out = []

    def walk(el, inherited):
        st = {**inherited, **{k: v for k, v in style(el).items() if k in ("fill", "stroke", "stroke-width", "opacity", "fill-rule")}}
        tag = el.tag.split("}")[-1]
        if tag == "defs":
            return
        if tag == "g":
            for c in el:
                walk(c, st)
            return
        if st.get("fill", "").startswith("url("):
            # Next.js's one gradient, current colour fading out: tern paints
            # its middle, current colour at half.
            st["fill"], st["opacity"] = "currentColor", 0.5
        stroked = st.get("fill") == "none"
        col = colour(st.get("stroke") if stroked else st.get("fill"))
        out.append({"d": path_data(tag, el.attrib), "opacity": float(st.get("opacity", 1)),
                    "col": col, "stroke": float(st.get("stroke-width", 1)) if stroked else 0,
                    "evenodd": st.get("fill-rule") == "evenodd"})

    for c in root:
        walk(c, {})
    return out


def go_icon(name, size, body):
    parts = []
    for s in shapes(name, body):
        f = [f"d: `{s['d']}`"]
        if s["opacity"] != 1:
            f.append(f"opacity: {s['opacity']:g}")
        if s["col"]:
            f.append(f"light: 0x{s['col'][0]}, dark: 0x{s['col'][1]}, own: true")
        if s["stroke"]:
            f.append(f"stroke: {s['stroke']:g}")
        if s["evenodd"]:
            f.append("evenOdd: true")
        parts.append("{" + ", ".join(f) + "}")
    return f'\t"{name}": {{{size}, []pierreShape{{{", ".join(parts)}}}}},'


PAIR = r'(?:"([^"]+)"|([\w$-]+)):`([^`]*)`'


def pairs(text):
    """The entries of a minified object literal of strings: k:`v`,"k.x":`v`."""
    return {(a or b): v for a, b, v in re.findall(PAIR, text)}


def js_object(text, var):
    m = re.search(r"[,;\s]" + re.escape(var) + r"=\{([^}]*)\}", text)
    if not m:
        raise SystemExit(f"no table {var}")
    return pairs(m.group(1))


def go_table(name, doc, table):
    out = doc + [f"var {name} = map[string]string{{"]
    for k in sorted(table):
        out.append(f'\t"{k}": "{table[k]}",')
    return out + ["}", ""]


diffs = symbols(bundle("getDiffFileInput"))
icons_js = bundle("pierre-icons")
# T3's document icons take their colour from DOCUMENT_ICON_COLORS, spliced
# into the sprite's template as ${Y[`name`]}.
doc_colours = re.search(r'[,;\s]Y=\{("t3-file-icon-pdf":.*?)\}', icons_js)
if not doc_colours:
    raise SystemExit("no document icon colours")
doc_colours = pairs(doc_colours.group(1))
trees = symbols(re.sub(r"\$\{Y\[`([\w-]+)`\]\}", lambda m: doc_colours[m.group(1)], icons_js))

# @pierre/trees' tables: Te by file name, Ee by extension, De the
# extensions the complete set maps first.
tail = icons_js[icons_js.rfind("</svg>"):]
names = js_object(icons_js, "Te")
exts = js_object(icons_js, "Ee")
complete = js_object(icons_js, "De")
# T3's overrides, in its icon set Z: byFileName and byFileExtension, the
# video extensions spread in first.
z = re.search(r"byFileName:\{([^}]*)\},byFileExtension:\{(.*?)\}\}", tail, re.S)
if not z:
    raise SystemExit("no T3 icon overrides")
t3_names, t3_exts = pairs(z.group(1)), pairs(z.group(2))
videos = re.search(r"\[`avi`,`video/x-msvideo`\]((?:,\[`\w+`,`video/[\w.-]+`\])*)\]",
                   "\n".join(p.read_text(errors="replace") for p in assets.glob("utils-*.js")))
if not videos:
    raise SystemExit("no video extensions")
for ext in ["avi"] + re.findall(r"\[`(\w+)`,", videos.group(1)):
    t3_exts.setdefault(ext, "t3-file-icon-video")

out = ["// Code generated by tools/pierre/gen.py from T3's web client. DO NOT EDIT.", "",
       "package ui", "",
       "// pierreIcons are the icons T3's diff viewer and file trees draw: Pierre's",
       "// sprites (@pierre/diffs, @pierre/trees; Apache-2.0, Pierre Computer",
       "// Company) and T3's own file icons (pierre-icons.ts), each its viewBox",
       "// side and its shapes as SVG path data.",
       "var pierreIcons = map[string]pierreIcon{"]
for name in DIFFS:
    if name not in diffs:
        raise SystemExit(f"no icon {name}")
    out.append(go_icon(name, *diffs[name]))
for name in sorted(trees):
    out.append(go_icon(name, *trees[name]))
out += ["}", ""]
out += go_table("pierreFileNames", ["// pierreFileNames is @pierre/trees' icon token by lowercase file name."], names)
out += go_table("pierreExtensions", ["// pierreExtensions is its icon token by extension suffix."], exts)
out += go_table("pierreCompleteExtensions", ["// pierreCompleteExtensions are the suffixes its complete set maps",
                                             "// before pierreExtensions."], complete)
out += go_table("t3FileNames", ["// t3FileNames and t3Extensions are T3's own icons by file name and",
                                "// extension, which win over Pierre's tables."], t3_names)
out += go_table("t3Extensions", [], t3_exts)
print("\n".join(out).rstrip() + "\n")
