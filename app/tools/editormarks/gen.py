"""Rasterises the editor marks T3's Open in editor shows into ui/editormarks.

T3 draws each editor's brand mark as an inline SVG (Icons.tsx,
JetBrainsIcons.tsx): masks, blurred shadows, gradients, an embedded image.
Gio can't draw those, so each becomes a PNG at 16, 24, 32 and 48px, as
librsvg renders it, once per theme where the mark takes the theme's
colour (T3 gives brand marks text-foreground).

    python3 tools/editormarks/gen.py <t3 apps/web/src/components> ui/editormarks
"""
import pathlib
import re
import subprocess
import sys
import tempfile

# Each editor's mark, by T3's editor id (OpenInPicker.tsx); the file
# manager's is the Finder on a Mac and File Explorer on Windows.
MARKS = {
    "cursor": "CursorIcon", "trae": "TraeIcon", "kiro": "KiroIcon", "vscode": "VisualStudioCode",
    "vscode-insiders": "VisualStudioCodeInsiders", "vscodium": "VSCodium", "zed": "Zed",
    "antigravity": "AntigravityIcon", "idea": "IntelliJIdeaIcon", "aqua": "AquaIcon", "clion": "CLionIcon",
    "datagrip": "DataGripIcon", "dataspell": "DataSpellIcon", "goland": "GoLandIcon", "phpstorm": "PhpStormIcon",
    "pycharm": "PyCharmIcon", "rider": "RiderIcon", "rubymine": "RubyMineIcon", "rustrover": "RustRoverIcon",
    "webstorm": "WebStormIcon", "finder": "FinderIcon", "file-explorer": "FileExplorerIcon",
}

# --foreground, which currentColor is in a brand mark, light and dark.
FOREGROUND = {"light": "#18181b", "dark": "#f5f5f5"}

# Marks whose fill comes from a class, per theme (Cursor's
# fill-[#26251E] dark:fill-[#EDECEC]).
CLASS_FILLS = {"CursorIcon": {"light": "#26251E", "dark": "#EDECEC"}}

SIZES = [16, 24, 32, 48]

# JSX's names for SVG's hyphenated attributes.
ATTRS = ["fillRule", "clipRule", "fillOpacity", "stopColor", "stopOpacity", "floodColor", "floodOpacity",
         "colorInterpolationFilters", "strokeWidth", "strokeLinecap", "strokeLinejoin", "strokeMiterlimit",
         "strokeOpacity", "clipPath", "maskType"]


def kebab(name):
    return re.sub(r"[A-Z]", lambda m: "-" + m.group(0).lower(), name)


def component(src, name):
    """The source of export const name, up to the next top-level export."""
    m = re.search(r"^export const " + re.escape(name) + r": Icon = .*?(?=^export |\Z)", src, re.S | re.M)
    if not m:
        raise SystemExit(f"no component {name}")
    return m.group(0)


def strip_braces(s, start):
    """The index after the {...} expression that opens at start."""
    depth = 0
    for i in range(start, len(s)):
        if s[i] == "{":
            depth += 1
        elif s[i] == "}":
            depth -= 1
            if depth == 0:
                return i + 1
    raise SystemExit("unbalanced braces")


def svg_of(src, name, theme):
    body = component(src, name)
    ids = {}
    for var, lit in re.findall(r"const (\w+) = `\$\{[^}]*\}-([\w-]+)`;", body):
        ids[var] = lit
    for vars_, prefix, n in re.findall(r"const \[([^\]]+)\] = useSvgGradientIds\(\"([\w-]+)\", (\d+)\);", body):
        for i, var in enumerate(v.strip() for v in vars_.split(",")):
            ids[var] = f"{prefix}-{chr(97 + i)}"
    consts = dict(re.findall(r"^const (\w+) =\s*\"([^\"]*)\";", src, re.M))
    start = body.index("<svg")
    end = body.rindex("</svg>") + len("</svg>")
    svg = body[start:end]
    # Drop the props, classes and empty expressions the markup carries.
    svg = re.sub(r"\{\.\.\.\w+\}", "", svg).replace("{}", "")
    while (i := svg.find("className={")) >= 0:
        svg = svg[:i] + svg[strip_braces(svg, i + len("className=")):]

    def expr(m):
        e = m.group(1).strip()
        if mm := re.fullmatch(r"`url\(#\$\{(\w+)\}\)`", e):
            return f'="url(#{ids[mm.group(1)]})"'
        if mm := re.fullmatch(r"`#\$\{(\w+)\}`", e):
            return f'="#{ids[mm.group(1)]}"'
        if e in ids:
            return f'="{ids[e]}"'
        if e in consts:
            return f'="{consts[e]}"'
        if mm := re.fullmatch(r"\{\s*mixBlendMode:\s*\"(\w+)\"\s*\}", e):
            return f'="mix-blend-mode:{mm.group(1)}"'
        raise SystemExit(f"{name}: can't read {{{e}}}")

    svg = re.sub(r"=\{((?:[^{}]|\{[^{}]*\})*)\}", expr, svg)
    for a in ATTRS:
        svg = re.sub(r"\b" + a + r"=", kebab(a) + "=", svg)
    svg = svg.replace("<svg", '<svg xmlns="http://www.w3.org/2000/svg"', 1)
    if name in CLASS_FILLS:
        svg = svg.replace("<svg", f'<svg fill="{CLASS_FILLS[name][theme]}"', 1)
    return svg.replace("currentColor", FOREGROUND[theme])


def main():
    components, out = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
    src = (components / "Icons.tsx").read_text() + "\n" + (components / "JetBrainsIcons.tsx").read_text()
    out.mkdir(parents=True, exist_ok=True)
    for old in out.glob("*.png"):
        old.unlink()
    for mark, name in MARKS.items():
        light, dark = svg_of(src, name, "light"), svg_of(src, name, "dark")
        themes = {"": light} if light == dark else {"": light, "-dark": dark}
        for suffix, svg in themes.items():
            with tempfile.NamedTemporaryFile("w", suffix=".svg", delete=False) as f:
                f.write(svg)
            for size in SIZES:
                png = out / f"{mark}{suffix}-{size}.png"
                subprocess.run(["rsvg-convert", "-w", str(size), "-h", str(size), "-o", str(png), f.name], check=True)
            pathlib.Path(f.name).unlink()


main()
