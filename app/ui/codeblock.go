package ui

import (
	"image"
	"image/color"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// T3's code block (MarkdownCodeBlock.tsx; docs/ui/06-timeline.md §2.6,
// §3.6): a frame with a 30dp header, the language's icon or word or the
// fence's title on the left and the wrap, run and copy buttons on the
// right, over the code
// in mono 13/17.875. T3 highlights with @pierre/diffs's lexers in its
// t3-light and pierre-dark themes; tern lexes with chroma and colours
// chroma's tokens with those themes' scopes, so token edges can differ but
// the colours don't.

// The highlight palettes (§3.6): t3-light, pierre-light darkened to 4.5:1,
// and pierre-dark.
var (
	syntaxLight = map[string]color.NRGBA{
		"fg": rgb(0x0a0a0a), "attribute": rgb(0x0e6944), "type.class": rgb(0x0e6944), "comment": rgb(0x5c5c5c),
		"decorator": rgb(0x0d4c7d), "diff.minus": rgb(0xb3242c), "diff.plus": rgb(0x0e6944), "function": rgb(0x693acf),
		"keyword": rgb(0xb12251), "operator": rgb(0x006781), "number": rgb(0x0b5267), "property_name": rgb(0x00609f),
		"punctuation": rgb(0x4b4b4b), "string": rgb(0x106a2b), "string.escape": rgb(0x0b685b), "string.regex": rgb(0x0d6971),
		"constant": rgb(0x735b02), "tag": rgb(0xa43d23), "type": rgb(0x912ba7), "variable": rgb(0x8f4e17),
		"variable.builtin": rgb(0x7e540e),
	}
	syntaxDark = map[string]color.NRGBA{
		"fg": rgb(0xfafafa), "attribute": rgb(0x60d199), "type.class": rgb(0x60d199), "comment": rgb(0x737373),
		"decorator": rgb(0x69b1ff), "diff.minus": rgb(0xff2e3f), "diff.plus": rgb(0x07c480), "function": rgb(0x9d6afb),
		"keyword": rgb(0xff678d), "operator": rgb(0x08c0ef), "number": rgb(0x68cdf2), "property_name": rgb(0x009fff),
		"punctuation": rgb(0x636363), "string": rgb(0x5ecc71), "string.escape": rgb(0x61d5c0), "string.regex": rgb(0x64d1db),
		"constant": rgb(0xffd452), "tag": rgb(0xff855e), "type": rgb(0xd568ea), "variable": rgb(0xffa359),
		"variable.builtin": rgb(0xffab16),
	}
)

// syntax is the palette of the theme in use.
func syntax() map[string]color.NRGBA {
	if dark {
		return syntaxDark
	}
	return syntaxLight
}

// tokenScope is the scope T3's themes colour a chroma token as.
func tokenScope(t chroma.TokenType, lang string) string {
	switch {
	case t == chroma.KeywordType:
		return "type"
	case t == chroma.KeywordConstant:
		return "number" // true, false, null: boolean
	case t.InCategory(chroma.Keyword):
		return "keyword"
	case t == chroma.NameFunction || t == chroma.NameFunctionMagic || t == chroma.NameBuiltin:
		return "function"
	case t == chroma.NameClass || t == chroma.NameException:
		return "type"
	case t == chroma.NameBuiltinPseudo || t == chroma.NameNamespace || t == chroma.NameLabel:
		return "variable.builtin"
	case t == chroma.NameTag && lang == "json":
		return "property_name"
	case t == chroma.NameTag:
		return "tag"
	case t == chroma.NameAttribute:
		return "attribute"
	case t == chroma.NameDecorator:
		return "decorator"
	case t == chroma.NameConstant || t == chroma.NameEntity:
		return "constant"
	case t.InCategory(chroma.Name):
		return "variable"
	case t == chroma.LiteralStringEscape:
		return "string.escape"
	case t == chroma.LiteralStringRegex:
		return "string.regex"
	case t == chroma.LiteralStringSymbol:
		return "constant"
	case t.InSubCategory(chroma.LiteralString):
		return "string"
	case t.InSubCategory(chroma.LiteralNumber) || t == chroma.LiteralDate:
		return "number"
	case t.InCategory(chroma.Operator):
		return "operator"
	case t.InCategory(chroma.Punctuation):
		return "punctuation"
	case t.InSubCategory(chroma.CommentPreproc):
		return "keyword"
	case t.InCategory(chroma.Comment):
		return "comment"
	case t == chroma.GenericDeleted:
		return "diff.minus"
	case t == chroma.GenericInserted:
		return "diff.plus"
	case t == chroma.GenericHeading || t == chroma.GenericSubheading:
		return "tag"
	case t == chroma.GenericEmph:
		return "keyword"
	case t == chroma.GenericStrong:
		return "constant"
	}
	return ""
}

// plainLanguages ignore token colours and take the text's own
// (index.css: [data-language="text" i] and the like).
var plainLanguages = map[string]bool{"text": true, "txt": true, "plain": true, "plaintext": true}

// codeColor is a span of code in one colour, by rune index.
type codeColor struct {
	end int // the rune it ends before
	col color.NRGBA
}

// highlight colours code in lang: spans in order, or none for a language
// chroma doesn't know.
func highlight(code, lang string) []codeColor {
	lexer := lexers.Get(lang)
	if lexer == nil || plainLanguages[strings.ToLower(lang)] {
		return nil
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, code)
	if err != nil {
		return nil
	}
	pal := syntax()
	var out []codeColor
	at := 0
	total := utf8.RuneCountInString(code)
	for tok := it(); tok != chroma.EOF && at < total; tok = it() {
		col, ok := pal[tokenScope(tok.Type, lang)]
		if !ok {
			col = pal["fg"]
		}
		at = min(total, at+utf8.RuneCountInString(tok.Value))
		if n := len(out); n > 0 && out[n-1].col == col {
			out[n-1].end = at
			continue
		}
		out = append(out, codeColor{at, col})
	}
	return out
}

// codeKey names a laid-out piece of code.
type codeKey struct {
	code  string
	lang  string
	col   color.NRGBA
	size  float32
	lineH float32
	w     int
	scale float32
	dark  bool
}

// codeLayout is laid-out code, recorded once.
type codeLayout struct {
	call op.CallOp
	size image.Point
}

// expandTabs puts spaces up to each tab stop of 8, as a browser's pre does.
func expandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	var sb strings.Builder
	col := 0
	for _, r := range s {
		switch r {
		case '\t':
			n := 8 - col%8
			sb.WriteString(strings.Repeat(" ", n))
			col += n
		case '\n':
			sb.WriteRune(r)
			col = 0
		default:
			sb.WriteRune(r)
			col++
		}
	}
	return sb.String()
}

// codeText lays out code in mono size dp on lines lineH dp apart: wrapped
// at maxW px (pre-wrap, breaking anywhere a line must), or not when maxW
// is 0. lang picks its highlighting; col is the text's own colour.
func (th *Theme) codeText(gtx C, code, lang string, col color.NRGBA, size, lineH float32, maxW int) *codeLayout {
	k := codeKey{code, lang, col, size, lineH, maxW, gtx.Metric.PxPerDp, dark}
	if l := th.tl.codes[k]; l != nil {
		return l
	}
	code = expandTabs(code)
	spans := highlight(code, lang)
	px := int(size*gtx.Metric.PxPerDp + 0.5)
	w := maxW
	if w <= 0 {
		w = 1 << 24
	}
	// The metrics first: measuring them shapes text of their own, which
	// would replace the code's.
	m := th.metrics(Mono(), px)
	th.Shaper.LayoutString(text.Parameters{Font: Mono(), PxPerEm: fixed.I(px), MaxWidth: w, WrapPolicy: text.WrapHeuristically}, code)
	lh := lineH * gtx.Metric.PxPerDp
	half := (lh - (m.ascent + m.descent)) / 2
	ops := new(op.Ops)
	g := gtx
	g.Ops = ops
	rec := op.Record(ops)
	var run []text.Glyph
	var runCol color.NRGBA
	line, rune0, width, span := 0, 0, 0, 0
	colorAt := func(r int) color.NRGBA {
		for span < len(spans) && spans[span].end <= r {
			span++
		}
		if span < len(spans) {
			return spans[span].col
		}
		return col
	}
	flush := func() {
		if len(run) > 0 {
			base := int(float32(line)*lh + half + m.ascent + 0.5)
			th.paintGlyphs(g, run, image.Pt(run[0].X.Floor(), base), runCol, false)
			run = run[:0]
		}
	}
	for gl, ok := th.Shaper.NextGlyph(); ok; gl, ok = th.Shaper.NextGlyph() {
		c := colorAt(rune0)
		if c != runCol {
			flush()
			runCol = c
		}
		if gl.Flags&text.FlagParagraphBreak == 0 {
			run = append(run, gl)
			width = max(width, (gl.X + gl.Advance).Ceil())
		}
		rune0 += int(gl.Runes)
		if gl.Flags&text.FlagLineBreak != 0 {
			flush()
			line++
		}
	}
	flush()
	if line == 0 {
		line = 1
	}
	l := &codeLayout{call: rec.Stop(), size: image.Pt(width, int(float32(line)*lh+0.5))}
	if th.tl.codes == nil || len(th.tl.codes) > 1000 {
		th.tl.codes = map[codeKey]*codeLayout{}
	}
	th.tl.codes[k] = l
	return l
}

// codeBlock draws a fenced code block and returns its height.
func (th *Theme) codeBlock(gtx C, n *mdNode, st mdStyle) int {
	w := gtx.Constraints.Max.X
	id := th.blockID(st)
	wrapped := !th.tl.unwrapped[id]
	rad := dp(gtx, 8)
	frame, border := P.Secondary, over(faded(P.Border, 0xb3), st.on)
	if dark {
		frame, border = over(faded(P.Input, 0x52), st.on), color.NRGBA{}
	}
	frame = over(frame, st.on)
	// The 1dp border takes its room in the layout, transparent or not:
	// everything inside sits 1dp in.
	bw := max(1, dp(gtx, 1))
	m := op.Record(gtx.Ops)
	// The header: 6dp in at the top and right, 12dp at the left. It names
	// the code: a fence title after its file's icon, 6dp apart; else the
	// language's icon, its name in a tooltip; else the language, mono 11
	// at 72% (MarkdownCodeBlockTitleContent).
	header := dp(gtx, 30)
	tg := within(gtx, w-2*bw-dp(gtx, 12+6+3*24+8))
	titleText := Txt{Size: 11, Font: Mono(), Color: over(faded(P.ContrastForeground, 0xb8), frame)}
	to := op.Offset(image.Pt(bw+dp(gtx, 12), bw+dp(gtx, 6))).Push(gtx.Ops)
	switch file := languageFile(n.lang); {
	case n.title != "":
		layout.Flex{Alignment: layout.Middle}.Layout(tg,
			layout.Rigid(fileIconW(n.title, 14, frame)),
			layout.Rigid(Spacer(6, 0)),
			layout.Flexed(1, func(gtx C) D { return th.textLine(gtx, titleText, 24, n.title) }),
		)
	case specificIcon(file):
		io := op.Offset(image.Pt(0, dp(gtx, 5))).Push(gtx.Ops)
		d := fileIconW(file, 14, frame)(tg)
		th.tipArea(tg, id+":lang", n.lang, d.Size)
		io.Pop()
	default:
		th.textLine(tg, titleText, 24, n.lang)
	}
	to.Pop()
	// The toolbar: wrap (pressed while wrapped), run, copy; 24dp, 2dp apart.
	type tool struct {
		id, tip string
		b       Button
	}
	wrap := tool{id + ":wrap", "Wrap lines", Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "text-wrap", IconSize: 12, On: frame}}
	if wrapped {
		wrap.tip, wrap.b.Variant = "Disable line wrap", BtnSecondary
	}
	tools := []tool{wrap}
	if st.run && !st.streaming && runnable(n) {
		rid := "run:" + id
		th.runTarget(rid, strings.TrimSpace(n.code))
		tools = append(tools, tool{rid, "Run in terminal", Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "play", IconSize: 12, On: frame}})
	}
	cp := tool{"copy:" + id, "Copy code", Button{Variant: BtnGhostMuted, Size: BtnSizeIconXs, Icon: "copy", IconSize: 12, On: frame}}
	th.copyTarget(cp.id, n.copyText())
	if th.copiedFor(gtx, cp.id, 1200) {
		cp.tip, cp.b.Icon = "Copied", "check"
	}
	tools = append(tools, cp)
	x := w - bw - dp(gtx, 6)
	for i := len(tools) - 1; i >= 0; i-- {
		t := tools[i]
		x -= dp(gtx, 24)
		bo := op.Offset(image.Pt(x, bw+dp(gtx, 6))).Push(gtx.Ops)
		th.T3Button(t.id, t.b)(gtx)
		th.tipAt(gtx, t.id, t.tip, image.Pt(dp(gtx, 24), dp(gtx, 24)), true, 0)
		bo.Pop()
		x -= dp(gtx, 2)
	}
	if th.clicked(wrap.id) {
		th.tl.unwrapped[id] = wrapped
	}
	// The code: padded 12.8 by 14.4, in the highlighter's colours.
	padX, padY := dp(gtx, 14.4), dp(gtx, 12.8)
	inner := w - 2*bw - 2*padX
	codeCol := syntax()["fg"]
	if plainLanguages[strings.ToLower(n.lang)] {
		codeCol = st.color
	}
	maxW := inner
	if !wrapped {
		maxW = 0
	}
	cl := th.codeText(gtx, n.code, n.lang, codeCol, 13, 17.875, maxW)
	body := cl.size.Y + 2*padY
	co := op.Offset(image.Pt(bw, bw+header)).Push(gtx.Ops)
	if wrapped {
		po := op.Offset(image.Pt(padX, padY)).Push(gtx.Ops)
		cl.call.Add(gtx.Ops)
		po.Pop()
	} else {
		th.hscroll(gtx, id, image.Pt(w-2*bw, body), cl.size.X+2*padX, func(gtx C) {
			po := op.Offset(image.Pt(padX, padY)).Push(gtx.Ops)
			cl.call.Add(gtx.Ops)
			po.Pop()
		})
	}
	co.Pop()
	call := m.Stop()
	h := bw + header + body + bw
	r := image.Rect(0, 0, w, h)
	fill(gtx, frame, r, rad)
	if border.A != 0 {
		strokeRect(gtx, border, r, rad, float32(max(1, dp(gtx, 1))))
	}
	func() {
		defer clip.UniformRRect(r, rad).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)
	}()
	return h
}

// runShells are the fences T3 offers "Run in terminal" for.
var runShells = map[string]bool{"sh": true, "bash": true, "zsh": true, "fish": true, "shell": true, "powershell": true, "pwsh": true}

// runnable is T3's canRun (MarkdownCodeBlock.tsx:166-176,
// ChatMarkdown.tsx:1721-1723): a shell fence that was closed, so a reply
// cut off mid-command never runs, of one line that isn't empty or
// continued by a trailing \, with no control or invisible format
// characters that would make what shows differ from what runs. A line
// break is a control character, so a second line rules it out.
func runnable(n *mdNode) bool {
	cmd := strings.TrimSpace(n.code)
	if !n.closed || !runShells[n.lang] || cmd == "" || strings.HasSuffix(cmd, `\`) {
		return false
	}
	for _, r := range n.code {
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

// copyText is what the copy button copies: the code as T3's code element
// holds it, its last line ended by a line break.
func (n *mdNode) copyText() string {
	if n.code == "" {
		return ""
	}
	return n.code + "\n"
}

// hscroll draws content wide px across a size box, scrolled sideways by
// the wheel or a drag when it doesn't fit.
func (th *Theme) hscroll(gtx C, id string, size image.Point, wide int, content func(gtx C)) {
	s := th.tl.scrolls[id]
	if s == nil {
		s = &hscroll{}
		th.tl.scrolls[id] = s
	}
	most := max(0, wide-size.X)
	d := s.g.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Horizontal, pointer.ScrollRange{Min: -s.x, Max: most - s.x}, pointer.ScrollRange{})
	s.x = max(0, min(most, s.x+d))
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if most > 0 {
		s.g.Add(gtx.Ops)
	}
	off := op.Offset(image.Pt(-s.x, 0)).Push(gtx.Ops)
	content(gtx)
	off.Pop()
	if most > 0 {
		// Chromium's thin scrollbar under the code.
		h := dp(gtx, 6)
		tw := max(dp(gtx, 16), size.X*size.X/wide)
		tx := (size.X - tw) * s.x / most
		thumb := faded(P.ContrastBorder, 0xc7)
		fill(gtx, over(thumb, P.Background), image.Rect(tx, size.Y-h-dp(gtx, 1), tx+tw, size.Y-dp(gtx, 1)), h/2)
	}
}

// hscroll is a scroller's state: its gesture, how far it has scrolled, and
// whether it was at its end (for one that follows its end).
type hscroll struct {
	g     gesture.Scroll
	x     int
	atEnd bool
}

// copiedFor reports whether copy button id was clicked in the last ms
// milliseconds, asking for a frame when its check should go.
func (th *Theme) copiedFor(gtx C, id string, ms int) bool {
	at, ok := th.in.copied[id]
	if !ok {
		return false
	}
	left := at.Add(time.Duration(ms) * time.Millisecond).Sub(gtx.Now)
	if left <= 0 {
		delete(th.in.copied, id)
		return false
	}
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
	return true
}

// clicked reports a click on target id this frame, taking it out of the
// frame's actions: the timeline's own toggles never reach core.
func (th *Theme) clicked(id string) bool {
	hit := false
	for i := 0; i < len(th.in.actions); i++ {
		if a := th.in.actions[i]; a.ID == id && a.Kind == "click" {
			th.in.actions = append(th.in.actions[:i], th.in.actions[i+1:]...)
			i--
			hit = true
		}
	}
	return hit
}

// copyTarget makes click target id copy s.
func (th *Theme) copyTarget(id, s string) {
	if th.in.copies == nil {
		th.in.copies, th.in.copied = map[string]string{}, map[string]time.Time{}
	}
	th.in.copies[id] = s
}

// runTarget makes click target id run command in the thread's terminal:
// it sends action "terminal.run" with the command.
func (th *Theme) runTarget(id, command string) {
	if th.in.runs == nil {
		th.in.runs = map[string]string{}
	}
	th.in.runs[id] = command
}
