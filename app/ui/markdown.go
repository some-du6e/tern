package ui

import (
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	gtext "github.com/yuin/goldmark/text"
)

// Markdown as T3's ChatMarkdown renders it (ChatMarkdown.tsx, index.css
// .chat-markdown; docs/ui/06-timeline.md §2.5, §3.5): CommonMark with
// GitHub's tables, strikethrough, task lists and autolinks, parsed by
// goldmark and laid out with .chat-markdown's rules, margins collapsing
// between blocks as CSS's do. GitHub's > [!NOTE] alerts are T3's too.
// Footnotes, <details>, math, Mermaid, images, file and thread chips are
// drawn as their text.

type mdKind int

const (
	mdPara mdKind = iota
	mdHeading
	mdList
	mdItem
	mdQuote
	mdAlert
	mdCode
	mdTable
	mdRule
)

// mdNode is one block.
type mdNode struct {
	kind  mdKind
	tight bool // a tight list item's paragraph, without margins
	level int  // a heading's level; a list's depth, from 1
	// A list: ordered, its first number and its markers' width in digits.
	ordered bool
	start   int
	digits  int
	task    int    // a task item: 1 open, 2 done
	alert   string // an alert's kind: note, tip, important, warning, caution
	// A code block: its language, its title and its code.
	lang, title, code string
	text              *inlineText // a paragraph's or heading's text
	kids              []*mdNode
	// A table: its rows, the first header ones of them, and each column's
	// alignment.
	rows   [][]*inlineText
	header int
}

// mdDoc is a parsed message.
type mdDoc struct {
	blocks []*mdNode
	links  []string
}

// mdStyle is how a message's markdown looks.
type mdStyle struct {
	color color.NRGBA // body text, painted on on
	on    color.NRGBA // the surface under it
	// user is a user's message: full foreground, line breaks kept, raw
	// HTML as written, decimal markers at every depth.
	user bool
	// streaming: a reply still arriving, whose code blocks can't run yet.
	streaming bool
}

// mdBody is an assistant's body text: foreground at 90%, on the canvas.
func mdBody() mdStyle {
	return mdStyle{color: over(faded(P.Foreground, 0xe6), P.Background), on: P.Background}
}

var mdParser = goldmark.New(goldmark.WithExtensions(extension.GFM))

// alertMarker is GitHub's alert syntax (markdownGithubAlerts.ts).
var alertMarker = regexp.MustCompile(`(?i)^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\][ \t]*(?:\r?\n|$)`)

// parsed is src parsed for style st, cached.
func (th *Theme) parsed(src string, st mdStyle, chips []string) *mdDoc {
	key := mdKey{src: src, chips: strings.Join(chips, "\x00"), st: st, dark: dark}
	if d := th.tl.docs[key]; d != nil {
		return d
	}
	if th.tl.docs == nil || len(th.tl.docs) > 2000 {
		// Layouts are keyed by their paragraphs' addresses, so they go too.
		th.tl.docs, th.tl.layouts = map[mdKey]*mdDoc{}, nil
	}
	b := &mdBuilder{src: []byte(src), st: st, chips: chips}
	doc := &mdDoc{blocks: b.blocks(mdParser.Parser().Parse(gtext.NewReader(b.src)), 0)}
	doc.links = b.links
	th.tl.docs[key] = doc
	return doc
}

type mdKey struct {
	src, chips string
	st         mdStyle
	dark       bool
}

// mdBuilder turns goldmark's tree into blocks.
type mdBuilder struct {
	src   []byte
	st    mdStyle
	chips []string
	links []string
}

func (b *mdBuilder) blocks(n ast.Node, depth int) []*mdNode {
	var out []*mdNode
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Paragraph:
			out = append(out, &mdNode{kind: mdPara, text: b.paragraph(c, 14, UI(font.Normal), b.st.color, 22.75)})
		case *ast.TextBlock:
			// A tight list's item text, which no <p> wraps: no margins.
			out = append(out, &mdNode{kind: mdPara, tight: true, text: b.paragraph(c, 14, UI(font.Normal), b.st.color, 22.75)})
		case *ast.Heading:
			size := map[int]float32{1: 20, 2: 18, 3: 16}[c.Level]
			if size == 0 {
				size = 14
			}
			col := P.ContrastForeground
			if c.Level == 6 {
				col = P.ContrastMutedForeground
			}
			out = append(out, &mdNode{kind: mdHeading, level: c.Level, text: b.paragraph(c, size, UI(font.SemiBold), over(col, b.st.on), 1.3*size)})
		case *ast.List:
			l := &mdNode{kind: mdList, level: depth + 1, ordered: c.IsOrdered(), start: c.Start}
			n := l.start
			for it := c.FirstChild(); it != nil; it = it.NextSibling() {
				item := &mdNode{kind: mdItem, kids: b.blocks(it, depth+1)}
				if box := taskBox(it); box != nil {
					item.task = 1
					if box.IsChecked {
						item.task = 2
					}
				}
				l.kids = append(l.kids, item)
				l.digits = max(l.digits, len(strconv.Itoa(n)))
				n++
			}
			out = append(out, l)
		case *ast.Blockquote:
			q := &mdNode{kind: mdQuote}
			if kind := b.alert(c); kind != "" {
				q.kind, q.alert = mdAlert, kind
			}
			q.kids = b.blocks(c, depth)
			if q.kind == mdAlert && len(q.kids) > 0 && q.kids[0].kind == mdPara && len(q.kids[0].text.runs) == 0 {
				q.kids = q.kids[1:]
			}
			out = append(out, q)
		case *ast.FencedCodeBlock:
			lang := string(c.Language(b.src))
			info := ""
			if c.Info != nil {
				info = strings.TrimSpace(strings.TrimPrefix(string(c.Info.Segment.Value(b.src)), lang))
			}
			out = append(out, &mdNode{kind: mdCode, lang: fenceLanguage(lang), title: fenceTitle(info), code: b.lines(c)})
		case *ast.CodeBlock:
			out = append(out, &mdNode{kind: mdCode, lang: "text", code: b.lines(c)})
		case *ast.ThematicBreak:
			out = append(out, &mdNode{kind: mdRule})
		case *east.Table:
			t := &mdNode{kind: mdTable}
			for r := c.FirstChild(); r != nil; r = r.NextSibling() {
				var cells []*inlineText
				for cell := r.FirstChild(); cell != nil; cell = cell.NextSibling() {
					w := font.Normal
					if _, head := r.(*east.TableHeader); head {
						w = font.SemiBold
					}
					cells = append(cells, b.paragraph(cell, 14, UI(w), b.st.color, 22.75))
				}
				if _, head := r.(*east.TableHeader); head {
					t.header++
				}
				t.rows = append(t.rows, cells)
			}
			out = append(out, t)
		case *ast.HTMLBlock:
			// T3 parses an assistant's HTML; tern shows its text.
			txt := b.lines(c)
			if !b.st.user {
				txt = htmlTag.ReplaceAllString(txt, "")
			}
			if strings.TrimSpace(txt) != "" {
				p := &inlineText{font: UI(font.Normal), size: 14, lineH: 22.75, codeBg: b.codeBg()}
				b.addText(p, strings.TrimSpace(txt), runStyle{font: UI(font.Normal), size: 14, color: b.st.color, pre: true})
				out = append(out, &mdNode{kind: mdPara, text: p})
			}
		default:
			out = append(out, b.blocks(c, depth)...)
		}
	}
	return out
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// taskBox is a list item's task checkbox, if it has one.
func taskBox(item ast.Node) *east.TaskCheckBox {
	if p := item.FirstChild(); p != nil {
		if box, ok := p.FirstChild().(*east.TaskCheckBox); ok {
			return box
		}
	}
	return nil
}

// alert reads and removes a blockquote's [!NOTE] marker.
func (b *mdBuilder) alert(q *ast.Blockquote) string {
	p, ok := q.FirstChild().(*ast.Paragraph)
	if !ok || p.Lines().Len() == 0 {
		return ""
	}
	seg := p.Lines().At(0)
	m := alertMarker.FindSubmatch(seg.Value(b.src))
	if m == nil {
		return ""
	}
	// Drop the text nodes the marker parsed into.
	end := seg.Start + len(m[0])
	for c := p.FirstChild(); c != nil; {
		next := c.NextSibling()
		t, ok := c.(*ast.Text)
		if !ok || t.Segment.Start >= end {
			break
		}
		if t.Segment.Stop <= end {
			p.RemoveChild(p, c)
		} else {
			t.Segment = t.Segment.WithStart(end)
		}
		c = next
	}
	return strings.ToLower(string(m[1]))
}

// lines is a code or HTML block's text.
func (b *mdBuilder) lines(n ast.Node) string {
	var sb strings.Builder
	ls := n.Lines()
	for i := 0; i < ls.Len(); i++ {
		seg := ls.At(i)
		sb.Write(seg.Value(b.src))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// codeBg is inline code's fill: the foreground at 10% on the surface.
func (b *mdBuilder) codeBg() color.NRGBA { return over(faded(P.ContrastForeground, 0x1a), b.st.on) }

// runStyle is the style inline text is added with.
type runStyle struct {
	font   font.Font
	size   float32
	color  color.NRGBA
	code   bool
	strike bool
	italic bool
	link   int
	pre    bool // keep the text's spaces and line breaks
}

// paragraph is a block's inline text.
func (b *mdBuilder) paragraph(n ast.Node, size float32, f font.Font, col color.NRGBA, lineH float32) *inlineText {
	t := &inlineText{font: f, size: size, lineH: lineH, codeBg: b.codeBg(), chips: b.chips}
	b.inlines(t, n, runStyle{font: f, size: size, color: col})
	return t
}

func (b *mdBuilder) inlines(t *inlineText, n ast.Node, st runStyle) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			b.addText(t, string(c.Segment.Value(b.src)), st)
			switch {
			case c.HardLineBreak():
				t.runs = append(t.runs, inlineRun{brk: true})
			case c.SoftLineBreak() && b.st.user:
				t.runs = append(t.runs, inlineRun{brk: true})
			case c.SoftLineBreak():
				b.addText(t, " ", st)
			}
		case *ast.String:
			b.addText(t, string(c.Value), st)
		case *ast.CodeSpan:
			var sb strings.Builder
			for g := c.FirstChild(); g != nil; g = g.NextSibling() {
				if tx, ok := g.(*ast.Text); ok {
					sb.Write(tx.Segment.Value(b.src))
				}
			}
			cs := st
			cs.code, cs.font, cs.size = true, Mono(), 0.85*st.size
			if st.link == 0 {
				cs.color = over(P.ContrastForeground, b.st.on)
			}
			b.addText(t, strings.ReplaceAll(sb.String(), "\n", " "), cs)
		case *ast.Emphasis:
			es := st
			if c.Level >= 2 {
				es.font.Weight = font.Bold
			} else {
				es.italic = true
			}
			b.inlines(t, c, es)
		case *east.Strikethrough:
			ss := st
			ss.strike = true
			b.inlines(t, c, ss)
		case *ast.Link:
			b.links = append(b.links, string(c.Destination))
			ls := st
			ls.link, ls.color = len(b.links), over(P.InfoForeground, b.st.on)
			b.inlines(t, c, ls)
		case *ast.AutoLink:
			b.links = append(b.links, string(c.URL(b.src)))
			ls := st
			ls.link, ls.color = len(b.links), over(P.InfoForeground, b.st.on)
			b.addText(t, string(c.Label(b.src)), ls)
		case *ast.Image:
			// The image's words, where T3 would show the picture.
			b.inlines(t, c, st)
		case *ast.RawHTML:
			var sb strings.Builder
			for i := 0; i < c.Segments.Len(); i++ {
				seg := c.Segments.At(i)
				sb.Write(seg.Value(b.src))
			}
			switch tag := strings.ToLower(sb.String()); {
			case b.st.user:
				b.addText(t, sb.String(), st)
			case tag == "<br>" || tag == "<br/>" || tag == "<br />":
				t.runs = append(t.runs, inlineRun{brk: true})
			}
		case *east.TaskCheckBox:
		default:
			b.inlines(t, c, st)
		}
	}
}

// addText adds s in style st: pictures' chips as chip runs, white space
// collapsed as CSS does unless st.pre.
func (b *mdBuilder) addText(t *inlineText, s string, st runStyle) {
	if !st.pre {
		s = collapseSpace(s, t)
	}
	start := 0
	rs := []rune(s)
	flush := func(end int) {
		if end <= start {
			return
		}
		text := string(rs[start:end])
		if st.pre && strings.Contains(text, "\n") {
			for i, line := range strings.Split(text, "\n") {
				if i > 0 {
					t.runs = append(t.runs, inlineRun{brk: true})
				}
				if line != "" {
					t.runs = append(t.runs, b.run(line, st))
				}
			}
			return
		}
		t.runs = append(t.runs, b.run(text, st))
	}
	for i := 0; i < len(rs); i++ {
		if n := ChipN(rs[i]); n > 0 && i+chipLen <= len(rs) && strings.Count(string(rs[i+1:i+chipLen]), string(chipFill)) == chipPad {
			flush(i)
			cs := st
			t.runs = append(t.runs, inlineRun{text: "", font: UI(font.Medium), size: cs.size, color: over(P.ContextChipImage, b.st.on), chip: n})
			i += chipLen - 1
			start = i + 1
		}
	}
	flush(len(rs))
}

func (b *mdBuilder) run(s string, st runStyle) inlineRun {
	return inlineRun{text: s, font: st.font, size: st.size, color: st.color, code: st.code, strike: st.strike, italic: st.italic, link: st.link}
}

// collapseSpace is CSS's white-space: normal: runs of spaces as one, and
// none after a run that already ends in one.
func collapseSpace(s string, t *inlineText) string {
	var sb strings.Builder
	space := false
	if n := len(t.runs); n > 0 && !t.runs[n-1].brk && strings.HasSuffix(t.runs[n-1].text, " ") || n == 0 {
		space = true
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !space {
				sb.WriteByte(' ')
			}
			space = true
			continue
		}
		space = false
		sb.WriteRune(r)
	}
	return sb.String()
}

// thoughtPreview is a thought as one line of text, T3's
// remarkThoughtPreview: its markdown flattened, blocks a space apart, cut
// at 2048 characters; "Thinking" or "Thought" when it says nothing.
// Previews are kept, since rows ask every frame.
func (th *Theme) thoughtPreview(md string, thinking bool) string {
	if s, ok := th.tl.previews[md]; ok {
		if s == "" && thinking {
			return "Thinking"
		} else if s == "" {
			return "Thought"
		}
		return s
	}
	if th.tl.previews == nil || len(th.tl.previews) > 2000 {
		th.tl.previews = map[string]string{}
	}
	s := flattenMarkdown(md)
	th.tl.previews[md] = s
	return th.thoughtPreview(md, thinking)
}

// flattenMarkdown is md's text without its markup.
func flattenMarkdown(md string) string {
	src := []byte(md[:min(len(md), 8192)])
	var sb strings.Builder
	var visit func(n ast.Node)
	visit = func(n ast.Node) {
		if sb.Len() >= 2048 {
			return
		}
		switch n := n.(type) {
		case *ast.Text:
			sb.Write(n.Segment.Value(src))
			if n.SoftLineBreak() || n.HardLineBreak() {
				sb.WriteByte(' ')
			}
			return
		case *ast.String:
			sb.Write(n.Value)
			return
		case *ast.AutoLink:
			sb.Write(n.Label(src))
			return
		case *ast.RawHTML, *ast.HTMLBlock:
			return
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			ls := n.Lines()
			for i := 0; i < ls.Len(); i++ {
				seg := ls.At(i)
				sb.Write(seg.Value(src))
			}
			return
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if c != n.FirstChild() && c.Type() == ast.TypeBlock {
				sb.WriteByte(' ')
			}
			visit(c)
		}
	}
	visit(mdParser.Parser().Parse(gtext.NewReader(src)))
	s := strings.Join(strings.Fields(sb.String()), " ")
	if r := []rune(s); len(r) > 2048 {
		s = string(r[:2048])
	}
	return s
}

// fenceLanguage is T3's extractFenceLanguage: "text" without one.
func fenceLanguage(lang string) string {
	switch lang {
	case "":
		return "text"
	case "gitignore":
		return "ini"
	}
	return lang
}

var (
	fenceTitleAttr  = regexp.MustCompile(`(?i)(?:^|\s)(?:title|file(?:name)?)=(?:"([^"]+)"|'([^']+)'|(\S+))`)
	fenceTitleToken = regexp.MustCompile(`^[\w@][\w@./-]*\.[A-Za-z0-9]+$`)
)

// fenceTitle is T3's extractFenceTitle: title="…" in the fence's meta, or
// a file name in it.
func fenceTitle(meta string) string {
	if m := fenceTitleAttr.FindStringSubmatch(meta); m != nil {
		for _, s := range m[1:] {
			if s != "" {
				return s
			}
		}
	}
	for _, f := range strings.Fields(meta) {
		if fenceTitleToken.MatchString(f) {
			return f
		}
	}
	return ""
}

// ---------- layout ----------

// markdown draws src in style st, as wide as it may be.
func (th *Theme) markdown(gtx C, src string, st mdStyle, chips []string) D {
	doc := th.parsed(src, st, chips)
	return D{Size: image.Pt(gtx.Constraints.Max.X, th.mdStack(gtx, doc, doc.blocks, st))}
}

// margins are a block's collapsed top and bottom margins, in dp: its own,
// or a container's first and last child's, whichever is larger, since
// nothing separates them (CSS margin collapsing).
func (n *mdNode) margins() (top, bottom float32) {
	switch {
	case n.kind == mdHeading:
		return 20, 8
	case n.kind == mdRule || n.tight:
		return 0, 0
	case n.kind == mdItem:
		top, bottom = 0, 0
	case n.kind == mdAlert:
		top, bottom = 4, 4
	default:
		top, bottom = 10.4, 10.4
	}
	if len(n.kids) > 0 && (n.kind == mdList || n.kind == mdItem || n.kind == mdQuote || n.kind == mdAlert) {
		first := n.kids[0]
		if n.kind == mdAlert {
			// The title is a paragraph of its own.
			first = &mdNode{kind: mdPara}
		}
		ft, _ := first.margins()
		_, lb := n.kids[len(n.kids)-1].margins()
		top, bottom = max(top, ft), max(bottom, lb)
	}
	return top, bottom
}

// mdStack lays out blocks top to bottom at the current offset and returns
// their height. The gap between two blocks is the larger of their facing
// margins; the stack's outer margins are its parent's to apply, and the
// message's own are dropped (.chat-markdown > :first-child, :last-child).
func (th *Theme) mdStack(gtx C, doc *mdDoc, nodes []*mdNode, st mdStyle) int {
	y := 0
	var prev float32
	for i, n := range nodes {
		top, bottom := n.margins()
		if i > 0 {
			y += dp(gtx, max(prev, top))
		}
		off := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		y += th.mdBlock(gtx, doc, n, st)
		off.Pop()
		prev = bottom
	}
	return y
}

// mdNatural is how wide blocks would be in a box that shrinks to fit them,
// up to maxW: paragraphs as wide as their longest line, lists and quotes
// as their items plus their gutters, and code, tables and rules the
// whole width.
func (th *Theme) mdNatural(gtx C, doc *mdDoc, nodes []*mdNode, maxW int) int {
	w := 0
	for _, n := range nodes {
		switch n.kind {
		case mdPara, mdHeading:
			w = max(w, th.layoutInline(gtx, n.text, maxW).size.X)
		case mdList:
			gutter := dp(gtx, 20)
			for _, item := range n.kids {
				w = max(w, gutter+th.mdNatural(gtx, doc, item.kids, maxW-gutter))
			}
		case mdQuote:
			in := dp(gtx, 2+12.8)
			w = max(w, in+th.mdNatural(gtx, doc, n.kids, maxW-in))
		default:
			return maxW
		}
	}
	return min(w, maxW)
}

// within is gtx narrowed to w px wide, from no minimum.
func within(gtx C, w int) C {
	gtx.Constraints = layout.Constraints{Max: image.Pt(max(0, w), gtx.Constraints.Max.Y)}
	return gtx
}

// mdBlock lays out one block and returns its height.
func (th *Theme) mdBlock(gtx C, doc *mdDoc, n *mdNode, st mdStyle) int {
	w := gtx.Constraints.Max.X
	switch n.kind {
	case mdPara, mdHeading:
		return th.inline(within(gtx, w), n.text, doc.links).Size.Y
	case mdRule:
		col := P.Border
		if dark {
			col = P.ContrastMutedForeground
		}
		hline(gtx, over(col, st.on), 0, 0, w)
		return max(1, dp(gtx, 1))
	case mdList:
		return th.mdList(gtx, doc, n, st)
	case mdQuote:
		// 2dp of the text colour at 35%, then 12.8dp of padding.
		rule := dp(gtx, 2)
		off := op.Offset(image.Pt(rule+dp(gtx, 12.8), 0)).Push(gtx.Ops)
		h := th.mdStack(within(gtx, w-rule-dp(gtx, 12.8)), doc, n.kids, st)
		off.Pop()
		fill(gtx, over(alpha(st.color, 0x59), st.on), image.Rect(0, 0, rule, h), 0)
		return h
	case mdAlert:
		return th.mdAlert(gtx, doc, n, st)
	case mdCode:
		return th.codeBlock(gtx, n, st)
	case mdTable:
		return th.mdTable(gtx, doc, n, st)
	}
	return 0
}

// alertLooks are T3's GitHub alert presentations (ChatMarkdown.tsx:350-390):
// label, icon, rule and title colours.
var alertLooks = map[string]struct {
	label, icon          string
	rule, title, titleDk string
}{
	"note":      {"Note", "info", "blue-500", "blue-600", "blue-400"},
	"tip":       {"Tip", "lightbulb", "emerald-500", "emerald-600", "emerald-400"},
	"important": {"Important", "message-square-warning", "purple-500", "purple-600", "purple-400"},
	"warning":   {"Warning", "triangle-alert", "amber-500", "amber-600", "amber-500"},
	"caution":   {"Caution", "octagon-alert", "red-500", "red-600", "red-400"},
}

// mdAlert is a GitHub alert: a 2dp rule in its colour at 70%, padded 12,
// a medium title with its 14dp icon, then the quote.
func (th *Theme) mdAlert(gtx C, doc *mdDoc, n *mdNode, st mdStyle) int {
	look := alertLooks[n.alert]
	rule := dp(gtx, 2)
	w := gtx.Constraints.Max.X
	titleCol := Tailwind[look.title]
	if dark {
		titleCol = Tailwind[look.titleDk]
	}
	g := within(gtx, w-rule-dp(gtx, 12))
	off := op.Offset(image.Pt(rule+dp(gtx, 12), 0)).Push(gtx.Ops)
	// The title: a 22.75 line, the icon 6dp before the medium label.
	line := dp(gtx, 22.75)
	ic := dp(gtx, 14)
	io := op.Offset(image.Pt(0, (line-ic)/2)).Push(gtx.Ops)
	LucideW(look.icon, over(titleCol, st.on), 14)(g)
	io.Pop()
	lo := op.Offset(image.Pt(ic+dp(gtx, 6), 0)).Push(gtx.Ops)
	th.textLine(g, Txt{Size: 14, Font: UI(font.Medium), Color: over(titleCol, st.on)}, 22.75, look.label)
	lo.Pop()
	h := line
	if len(n.kids) > 0 {
		ft, _ := n.kids[0].margins()
		h += dp(gtx, max(10.4, ft))
		ko := op.Offset(image.Pt(0, h)).Push(gtx.Ops)
		h += th.mdStack(g, doc, n.kids, st)
		ko.Pop()
	}
	off.Pop()
	fill(gtx, over(alpha(Tailwind[look.rule], 0xb3), st.on), image.Rect(0, 0, rule, h), 0)
	return h
}

// mdList lays out a list's items with their markers in the 20dp gutter:
// discs, circles and squares by depth, or numbers widened for two digits.
func (th *Theme) mdList(gtx C, doc *mdDoc, n *mdNode, st mdStyle) int {
	w := gtx.Constraints.Max.X
	gutter := dp(gtx, 20)
	if n.ordered && n.digits > 1 {
		gutter = max(gutter, (n.digits+2)*th.chWidth(gtx))
	}
	g := within(gtx, w-gutter)
	y := 0
	var prev float32
	for i, item := range n.kids {
		top, bottom := item.margins()
		if i > 0 {
			y += dp(gtx, max(prev, top, 4)) // li + li
		}
		off := op.Offset(image.Pt(gutter, y)).Push(gtx.Ops)
		h := th.mdStack(g, doc, item.kids, st)
		off.Pop()
		mo := op.Offset(image.Pt(0, y)).Push(gtx.Ops)
		th.listMarker(gtx, n, item, i, gutter, st)
		mo.Pop()
		y += h
		prev = bottom
	}
	return y
}

// listMarker draws an item's marker on its first line: a 13dp checkbox
// for a task, else the bullet or number, ending before the gutter's edge.
func (th *Theme) listMarker(gtx C, n, item *mdNode, i, gutter int, st mdStyle) {
	line := dp(gtx, 22.75)
	switch {
	case item.task > 0:
		s := dp(gtx, 13)
		r := image.Rect(0, (line-s)/2, s, (line-s)/2+s)
		checkbox(gtx, r, item.task == 2)
	case n.ordered:
		style := "decimal"
		if !st.user {
			style = []string{"decimal", "lower-alpha", "lower-roman"}[min(2, (n.level-1)%3)]
		}
		// The number and its ". ", tabular, ending where the text starts.
		m := op.Record(gtx.Ops)
		d := th.textLine(within(gtx, gutter), Txt{Size: 14, Color: st.color}, 22.75, listNumber(n.start+i, style)+". ")
		call := m.Stop()
		off := op.Offset(image.Pt(gutter-d.Size.X, 0)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	default:
		// Chrome's symbol markers: 5dp at 14dp text, ending 7dp before the
		// text, on the x-height's middle; discs, then circles, then squares.
		d := dp(gtx, 5)
		x0, y0 := gutter-dp(gtx, 7)-d, (line-d)/2+dp(gtx, 1)
		r := image.Rect(x0, y0, x0+d, y0+d)
		switch (n.level - 1) % 3 {
		case 0:
			paint.FillShape(gtx.Ops, st.color, clip.Ellipse(r).Op(gtx.Ops))
		case 1:
			px := max(1, dp(gtx, 1))
			ring := clip.Ellipse(r.Inset(px / 2)).Path(gtx.Ops)
			paint.FillShape(gtx.Ops, st.color, clip.Stroke{Path: ring, Width: float32(px)}.Op())
		default:
			fill(gtx, st.color, r, 0)
		}
	}
}

// checkbox is a task's read-only native checkbox, as Chrome draws one in
// the page's color-scheme.
func checkbox(gtx C, r image.Rectangle, on bool) {
	rad := dp(gtx, 2)
	px := float32(max(1, dp(gtx, 1)))
	switch {
	case on && dark:
		fill(gtx, rgb(0x99c8ff), r, rad)
		polylineIn(gtx, r, rgb(0x3b3b3b))
	case on:
		fill(gtx, rgb(0x0075ff), r, rad)
		polylineIn(gtx, r, rgb(0xffffff))
	case dark:
		fill(gtx, rgb(0x3b3b3b), r, rad)
		strokeRect(gtx, rgb(0x858585), r, rad, px)
	default:
		fill(gtx, rgb(0xffffff), r, rad)
		strokeRect(gtx, rgb(0x767676), r, rad, px)
	}
}

// polylineIn draws a checkbox's tick in r.
func polylineIn(gtx C, r image.Rectangle, col color.NRGBA) {
	u := float32(r.Dx()) / 13
	pt := func(x, y float32) f32.Point { return f32.Pt(float32(r.Min.X)+x*u, float32(r.Min.Y)+y*u) }
	polyline(gtx, col, 1.6*u, pt(3, 6.8), pt(5.5, 9.3), pt(10, 4.2))
}

// listNumber writes n as an ordered list's style does.
func listNumber(n int, style string) string {
	switch style {
	case "lower-alpha":
		var s []byte
		for n > 0 {
			n--
			s = append([]byte{byte('a' + n%26)}, s...)
			n /= 26
		}
		return string(s)
	case "lower-roman":
		return strings.ToLower(roman(n))
	}
	return strconv.Itoa(n)
}

func roman(n int) string {
	if n <= 0 || n >= 4000 {
		return strconv.Itoa(n)
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var sb strings.Builder
	for i, v := range vals {
		for n >= v {
			sb.WriteString(syms[i])
			n -= v
		}
	}
	return sb.String()
}

// chWidth is CSS's ch at 14dp: the width of "0".
func (th *Theme) chWidth(gtx C) int {
	_, _, w := th.shapeLine(UI(font.Normal), dp(gtx, 14), "0", 1<<20, false)
	return w
}
