// inspect.js dumps what T3's web UI actually renders: for each element under
// a selector, its tag, classes, box and the computed styles that differ from
// a blank element's, with colours painted on a canvas and read back as sRGB
// hex (#rrggbb, or #rrggbbaa when not opaque). Run it with inspect.sh.
(() => {
  const query = __QUERY__; // {selector, depth = 3, max = 60, all = false}
  const cvs = document.createElement("canvas");
  cvs.width = cvs.height = 1;
  const ctx = cvs.getContext("2d", { willReadFrequently: true });
  const hex = (c) => {
    ctx.clearRect(0, 0, 1, 1);
    ctx.fillStyle = "#000";
    ctx.fillStyle = c;
    ctx.fillRect(0, 0, 1, 1);
    const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data;
    const h = (x) => x.toString(16).padStart(2, "0");
    return a === 0 ? "transparent" : "#" + h(r) + h(g) + h(b) + (a < 255 ? h(a) : "");
  };
  // Colours inside other values (shadows, gradients, borders) get the same
  // treatment.
  const colorFn = /(rgba?|hsla?|oklch|oklab|lab|lch|color|color-mix)\((?:[^()]|\([^()]*\))*\)/g;
  const norm = (v) => v.replace(colorFn, (m) => hex(m));
  const props = [
    "display", "position", "box-sizing", "width", "height", "min-width", "min-height", "max-width", "max-height",
    "padding-top", "padding-right", "padding-bottom", "padding-left",
    "margin-top", "margin-right", "margin-bottom", "margin-left",
    "gap", "row-gap", "column-gap", "flex-direction", "flex-grow", "flex-shrink", "flex-basis", "align-items", "justify-content",
    "font-family", "font-size", "font-weight", "font-style", "line-height", "letter-spacing", "text-transform",
    "text-align", "text-overflow", "white-space", "font-variant-numeric", "font-feature-settings", "-webkit-font-smoothing",
    "color", "background-color", "background-image", "opacity",
    "border-top-width", "border-right-width", "border-bottom-width", "border-left-width",
    "border-top-color", "border-right-color", "border-bottom-color", "border-left-color",
    "border-top-style", "border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius",
    "outline-width", "outline-color", "outline-offset", "box-shadow", "backdrop-filter", "filter",
    "transform", "transition", "animation", "cursor", "overflow-x", "overflow-y", "z-index", "stroke", "stroke-width", "fill",
  ];
  const blank = document.createElement("div");
  document.body.appendChild(blank);
  const base = getComputedStyle(blank);
  const defaults = Object.fromEntries(props.map((p) => [p, base.getPropertyValue(p)]));
  blank.remove();
  const colorProps = /color$|^fill$|^stroke$/;
  const out = [];
  const visit = (el, depth) => {
    if (out.length >= (query.max ?? 60)) return;
    const cs = getComputedStyle(el);
    const r = el.getBoundingClientRect();
    const style = {};
    for (const p of props) {
      let v = cs.getPropertyValue(p);
      if (v === defaults[p] && !["font-size", "font-family", "font-weight", "line-height", "color"].includes(p)) continue;
      if (colorProps.test(p)) v = hex(v);
      else if (/shadow|image|filter|border/.test(p)) v = norm(v);
      style[p] = v;
    }
    const text = [...el.childNodes].filter((n) => n.nodeType === 3).map((n) => n.textContent.trim()).join(" ").slice(0, 80);
    const attrs = {};
    for (const a of el.attributes) if (/^(data-|aria-|role$|id$|title$|type$)/.test(a.name)) attrs[a.name] = a.value.slice(0, 80);
    out.push({
      depth, tag: el.tagName.toLowerCase(), class: (el.getAttribute("class") || "").slice(0, 600), attrs, text,
      box: { x: +r.x.toFixed(2), y: +r.y.toFixed(2), w: +r.width.toFixed(2), h: +r.height.toFixed(2) }, style,
    });
    if (depth < (query.depth ?? 3)) for (const c of el.children) visit(c, depth + 1);
  };
  const roots = query.all ? [...document.querySelectorAll(query.selector)] : [document.querySelector(query.selector)].filter(Boolean);
  for (const el of roots) visit(el, 0);
  return { url: location.pathname, dark: document.documentElement.classList.contains("dark"), rootClass: document.documentElement.className, count: out.length, nodes: out };
})();
