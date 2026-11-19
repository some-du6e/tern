"""Copies T3's file icons out of its web client into ui/pierre.go.

T3 draws file icons from one SVG sprite: @pierre/trees' built-in set and
T3's own symbols (apps/web/src/pierre-icons.ts), picked by file name and
extension and coloured by token (PierreEntryIcon.tsx). This takes the
shapes, the tables that pick them and the colours from the shipped bundles.

    python3 tools/pierre/gen.py <t3 client assets dir> | gofmt > ui/pierre.go
"""
import pathlib
import re
import sys

assets = pathlib.Path(sys.argv[1])


def bundle(prefix, holding):
    """The bundle named prefix-<hash>.js that holds the text holding."""
    for p in sorted(assets.glob(prefix + "-*.js")):
        js = p.read_text()
        if holding in js:
            return js
    raise SystemExit(f"no {prefix} bundle with {holding}")


icons_js = bundle("pierre-icons", "file-tree-builtin-")
entry_js = bundle("PierreEntryIcon", "astro:[")
utils_js = bundle("utils", "video/x-msvideo")


def table(js, name):
    """A minified object literal of string values: name={k:`v`,"k":`v`}."""
    m = re.search(r"(?:^|[,;{ ])" + re.escape(name) + r"=\{(.*?)\}[,;]", js, re.S)
    if not m:
        raise SystemExit(f"no table {name}")
    return pairs(m.group(1))


def pairs(js):
    """The key:`value` pairs of an object literal's body."""
    return {a or b: v for a, b, v in re.findall(r'(?:"([^"]+)"|([\w$]+)):`([^`]*)`', js)}


def colour_table(js, name):
    m = re.search(re.escape(name) + r"=\{(.*?)\}[,;]", js, re.S)
    if not m:
        raise SystemExit(f"no colour table {name}")
    return {k: (a, b) for k, a, b in re.findall(r"([\w$]+):\[`(#[0-9a-f]{6})`,`(#[0-9a-f]{6})`\]", m.group(1))}


# The colour of each token, light and dark: PierreEntryIcon's first table.
m = re.search(r"([\w$]+)=\{astro:\[`#", entry_js)
if not m:
    raise SystemExit("no icon colours")
colours = colour_table(entry_js, m.group(1))

# The built-in tables: file names, extensions, and the complete set's own.
names_var = re.search(r"([\w$]+)=\{\"\.babelrc\":`babel`", icons_js).group(1)
exts_var = re.search(r"([\w$]+)=\{\"7z\":`zip`", icons_js).group(1)
complete_var = re.search(r"([\w$]+)=\{jsx:`react`", icons_js).group(1)
names, exts, complete = table(icons_js, names_var), table(icons_js, exts_var), table(icons_js, complete_var)

# T3's own: byFileName, byFileExtension (videos first), the language aliases.
t3 = re.search(r"byFileName:\{(.*?)\},byFileExtension:\{(.*?)\}\}", icons_js, re.S)
t3_names = pairs(t3.group(1))
videos = re.findall(r"\[`(\w+)`,`video/[^`]+`\]", re.search(r"new Map\(\[\[`avi`,`video/x-msvideo`\].*?\]\]\)", utils_js).group(0))
t3_exts = {v: "t3-file-icon-video" for v in videos}
t3_exts.update(pairs(t3.group(2)))
aliases_var = re.search(r"([\w$]+)=\{bash:`sh`,csharp:`cs`", icons_js).group(1)
aliases = table(icons_js, aliases_var)
lang_names_var = re.search(r"([\w$]+)=\{dockerfile:`Dockerfile`\}", icons_js).group(1)
lang_names = table(icons_js, lang_names_var)

# The sprites, with T3's document colours put in.
doc_var = re.search(r"([\w$]+)=\{\"t3-file-icon-pdf\":`light-dark", icons_js).group(1)
doc = table(icons_js, doc_var)
sprite = re.sub(r"\$\{" + re.escape(doc_var) + r"\[`([^`]+)`\]\}", lambda m: doc[m.group(1)], icons_js)


def num(v):
    return float(v)


def attrs(s):
    return dict(re.findall(r'([\w-]+)="([^"]*)"', s))


def light_dark(v):
    m = re.search(r"light-dark\((#[0-9a-f]{6}),\s*(#[0-9a-f]{6})\)", v)
    return (m.group(1), m.group(2)) if m else None


def style_of(a, prop):
    m = re.search(prop + r":\s*([^;]+)", a.get("style", ""))
    return m.group(1) if m else None


def shape_d(kind, a):
    if kind == "path":
        return a["d"]
    if kind == "circle":
        cx, cy, r = num(a["cx"]), num(a["cy"]), num(a["r"])
        return f"M{cx-r:g} {cy:g}a{r:g} {r:g} 0 1 0 {2*r:g} 0a{r:g} {r:g} 0 1 0 {-2*r:g} 0Z"
    if kind == "rect":
        x, y, w, h = num(a.get("x", 0)), num(a.get("y", 0)), num(a["width"]), num(a["height"])
        rx = num(a.get("rx", 0))
        if rx == 0:
            return f"M{x:g} {y:g}h{w:g}v{h:g}h{-w:g}Z"
        return (f"M{x+rx:g} {y:g}H{x+w-rx:g}A{rx:g} {rx:g} 0 0 1 {x+w:g} {y+rx:g}V{y+h-rx:g}"
                f"A{rx:g} {rx:g} 0 0 1 {x+w-rx:g} {y+h:g}H{x+rx:g}A{rx:g} {rx:g} 0 0 1 {x:g} {y+h-rx:g}"
                f"V{y+rx:g}A{rx:g} {rx:g} 0 0 1 {x+rx:g} {y:g}Z")
    raise SystemExit(f"unknown shape {kind}")


def paint(a, group, prop, default=None):
    """A shape's fill or stroke, its own or its group's: None, "current",
    (light, dark), or ("grad", id)."""
    v = style_of(a, prop) or a.get(prop) or style_of(group, prop) or group.get(prop) or default
    if v is None or v == "none":
        return None
    if v.lower() == "currentcolor":
        return "current"
    if v.startswith("url(#"):
        return ("grad", v[5:-1])
    ld = light_dark(v)
    if ld:
        return ld
    if re.fullmatch(r"#[0-9a-f]{6}", v):
        return (v, v)
    raise SystemExit(f"unknown paint {v}")


def go_colour(c):
    return "0x" + c[1:]


def symbol(name, viewbox, body):
    box = num(viewbox.split()[2])
    grads = {}
    for gid, ga, stops in re.findall(r'<linearGradient id="([^"]+)"([^>]*)>(.*?)</linearGradient>', body, re.S):
        g = attrs(ga)
        ops = [num(attrs(s).get("stop-opacity", "1")) for s in re.findall(r"<stop([^>]*)/?>", stops)]
        grads[gid] = (num(g["x1"]), num(g["y1"]), num(g["x2"]), num(g["y2"]), ops[0], ops[-1])
    body = re.sub(r"<defs>.*?</defs>", "", body, flags=re.S)
    shapes = []
    group = {}
    for tag in re.finditer(r"<(/?)(\w+)([^>]*?)(/?)>", body):
        close, kind, rest, selfclose = tag.groups()
        if kind == "g":
            group = {} if close else attrs(rest)
            continue
        if close or kind not in ("path", "circle", "rect"):
            continue
        a = attrs(rest)
        d = shape_d(kind, a)
        # Every shape names its paint; one that doesn't takes the icon's.
        fill, stroke = paint(a, group, "fill", "currentColor"), paint(a, group, "stroke")
        alpha = num(a.get("opacity", "1"))
        even = a.get("fill-rule") == "evenodd"
        for p, width in ((fill, 0), (stroke, num(a.get("stroke-width", group.get("stroke-width", "1"))))):
            if p is None:
                continue
            fields = [f"d: `{d}`"]
            if width:
                fields.append(f"stroke: {width:g}")
            if isinstance(p, tuple) and p[0] == "grad":
                x1, y1, x2, y2, o1, o2 = grads[p[1]]
                fields.append(f"fade: [4]float32{{{x1:g}, {y1:g}, {x2:g}, {y2:g}}}")
            elif isinstance(p, tuple):
                fields.append(f"own: true, light: {go_colour(p[0])}, dark: {go_colour(p[1])}")
            if alpha != 1:
                fields.append(f"alpha: {alpha:g}")
            if even:
                fields.append("evenOdd: true")
            shapes.append("{" + ", ".join(fields) + "}")
    return f'\t"{name}": {{{box:g}, []pierreShape{{{", ".join(shapes)}}}}},'


symbols = {}
for name, viewbox, body in re.findall(r'<symbol id="([^"]+)" viewBox="([^"]+)">(.*?)</symbol>', sprite, re.S):
    if name.startswith("file-tree-builtin-") or name.startswith("t3-file-icon-"):
        symbols[name] = symbol(name, viewbox, body)

# Every icon a table can pick must be there.
wanted = {"file-tree-builtin-" + t for t in list(names.values()) + list(exts.values()) + list(complete.values()) + ["default"]}
wanted |= set(t3_names.values()) | set(t3_exts.values())
missing = wanted - set(symbols)
if missing:
    raise SystemExit(f"missing icons {sorted(missing)}")


def go_map(name, doc_lines, m, value=lambda v: f'"{v}"', typ="string"):
    out = [f"// {line}" if line else "//" for line in doc_lines]
    out.append(f"var {name} = map[string]{typ}{{")
    for k in sorted(m):
        out.append(f'\t"{k}": {value(m[k])},')
    out.append("}")
    out.append("")
    return out


out = ["// Code generated by tools/pierre/gen.py from T3's web client. DO NOT EDIT.", "", "package ui", "",
       "// pierreIcons are the file icons T3 draws, @pierre/trees' and its own:",
       "// each shape's path data in the icon's box, filled or stroked.",
       "var pierreIcons = map[string]pierreIcon{"]
out += [symbols[k] for k in sorted(symbols)]
out += ["}", ""]
out += go_map("pierreNames", ["pierreNames, pierreExtensions and pierreComplete give file names and",
                              "extensions their built-in icon's token (@pierre/trees' complete set)."], names)
out += go_map("pierreExtensions", ["pierreExtensions: see pierreNames."], exts)
out += go_map("pierreComplete", ["pierreComplete: see pierreNames."], complete)
out += go_map("t3IconNames", ["t3IconNames and t3IconExtensions are T3's own picks, by icon name",
                              "(pierre-icons.ts T3_PIERRE_ICONS)."], t3_names)
out += go_map("t3IconExtensions", ["t3IconExtensions: see t3IconNames."], t3_exts)
out += go_map("pierreColors", ["pierreColors are each token's colour, light and dark (PierreEntryIcon.tsx)."],
              colours, lambda v: f"{{{go_colour(v[0])}, {go_colour(v[1])}}}", "[2]uint32")
out += go_map("languageExtensions", ["languageExtensions and languageFileNames name a fence's language as a",
                                     "file, for its icon (syntheticFileNameForLanguageId)."], aliases)
out += go_map("languageFileNames", ["languageFileNames: see languageExtensions."], lang_names)
print("\n".join(out).rstrip("\n"))
