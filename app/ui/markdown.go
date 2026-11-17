package ui

import (
	"hash/fnv"
	"strconv"
	"time"

	"image"
	"regexp"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/x/richtext"
)

// The markdown T3 replies actually use: paragraphs, bullets, headings, fenced
// code, inline code and bold. Anything else renders as plain text.

type mdBlock struct {
	kind byte // 'p', 'h', 'l' (list item), 'c' (code)
	text string
	lang string // a code block's fence language: "powershell", "go"
}

func parseMarkdown(src string) []mdBlock {
	var out []mdBlock
	var para []string
	flush := func() {
		if len(para) > 0 {
			out = append(out, mdBlock{kind: 'p', text: strings.Join(para, " ")})
			para = nil
		}
	}
	lines := strings.Split(strings.TrimSpace(src), "\n")
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "```"):
			flush()
			lang := strings.TrimSpace(strings.TrimPrefix(t, "```"))
			var code []string
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```"); i++ {
				code = append(code, lines[i])
			}
			out = append(out, mdBlock{'c', strings.Join(code, "\n"), lang})
		case t == "":
			flush()
		case strings.HasPrefix(t, "#"):
			flush()
			out = append(out, mdBlock{kind: 'h', text: strings.TrimSpace(strings.TrimLeft(t, "#"))})
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
			flush()
			out = append(out, mdBlock{kind: 'l', text: t[2:]})
		default:
			para = append(para, t)
		}
	}
	flush()
	return out
}

// spans splits inline `code` and **bold** into styled runs.
var mdLink = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+)\)`)

// spans styles one paragraph: links show their text in the accent colour,
// then inline code and bold inside the rest.
func spans(s string, base richtext.SpanStyle) []richtext.SpanStyle {
	m := mdLink.FindStringSubmatchIndex(s)
	if m == nil {
		return plainSpans(s, base)
	}
	link := base
	link.Color = ColAccentHi
	link.Content = s[m[2]:m[3]]
	out := append(plainSpans(s[:m[0]], base), link)
	return append(out, spans(s[m[1]:], base)...)
}

func plainSpans(s string, base richtext.SpanStyle) []richtext.SpanStyle {
	var out []richtext.SpanStyle
	for len(s) > 0 {
		i := nextMark(s)
		if i < 0 {
			break
		}
		if i > 0 {
			sp := base
			sp.Content = s[:i]
			out = append(out, sp)
		}
		if s[i] == '`' {
			end := strings.IndexByte(s[i+1:], '`')
			if end < 0 {
				sp := base
				sp.Content = s[i:]
				out = append(out, sp)
				return out
			}
			sp := base
			sp.Font = Mono()
			sp.Size = base.Size - 1
			sp.Color = ColCode
			sp.Content = s[i+1 : i+1+end]
			out = append(out, sp)
			s = s[i+2+end:]
		} else {
			end := strings.Index(s[i+2:], "**")
			if end < 0 {
				sp := base
				sp.Content = s[i:]
				out = append(out, sp)
				return out
			}
			sp := base
			sp.Font = UI(font.SemiBold)
			sp.Color = rgb(0xf3f5f7)
			sp.Content = s[i+2 : i+2+end]
			out = append(out, sp)
			s = s[i+4+end:]
		}
	}
	if s != "" {
		sp := base
		sp.Content = s
		out = append(out, sp)
	}
	return out
}

// nextMark finds the next "`" or "**".
func nextMark(s string) int {
	i, j := strings.IndexByte(s, '`'), strings.Index(s, "**")
	if i < 0 || (j >= 0 && j < i) {
		return j
	}
	return i
}

// mdCache keeps parsed messages between frames, so an open thread doesn't
// re-parse every message on every frame.
type mdCache struct {
	blocks []mdBlock
	spans  [][]richtext.SpanStyle // per block, styled
}

func (th *Theme) parsed(src string) *mdCache {
	if c := th.md[src]; c != nil {
		return c
	}
	if th.md == nil || len(th.md) > 2000 {
		// The shaped-text cache is keyed by these spans' addresses, so it
		// goes with them.
		th.md = map[string]*mdCache{}
		th.richCache = nil
	}
	base := richtext.SpanStyle{Font: UI(font.Normal), Size: SizeBody, Color: ColText}
	c := &mdCache{blocks: parseMarkdown(src)}
	for _, b := range c.blocks {
		st := base
		if b.kind == 'h' {
			st.Font, st.Size = UI(font.SemiBold), SizeTitle
		}
		var sp []richtext.SpanStyle
		if b.kind != 'c' {
			sp = spans(b.text, st)
		}
		c.spans = append(c.spans, sp)
	}
	th.md[src] = c
	return c
}

func (th *Theme) Markdown(gtx C, src string) D {
	c := th.parsed(src)
	blocks := c.blocks
	h := fnv.New64a()
	h.Write([]byte(src))
	doc := strconv.FormatUint(h.Sum64(), 36)
	var kids []layout.FlexChild
	for i, b := range blocks {
		b := b
		if i > 0 {
			gap := float32(9)
			if b.kind == 'l' && blocks[i-1].kind == 'l' {
				gap = 4
			}
			kids = append(kids, rigid(Spacer(0, gap)))
		}
		switch b.kind {
		case 'p', 'h':
			sp := c.spans[i]
			kids = append(kids, rigid(func(gtx C) D { return th.rich(gtx, sp) }))
		case 'l':
			kids = append(kids, rigid(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					rigid(func(gtx C) D {
						s := dp(gtx, 18)
						circle(gtx, ColFaint, image.Pt(dp(gtx, 6), dp(gtx, 9.5)), dp(gtx, 2))
						return D{Size: image.Pt(s, s)}
					}),
					flexed(func(gtx C) D { return th.rich(gtx, c.spans[i]) }),
				)
			}))
		case 'c':
			id := "code:" + doc + ":" + strconv.Itoa(i)
			kids = append(kids, rigid(func(gtx C) D { return th.codeCard(gtx, id, b.lang, b.text) }))
		}
	}
	return Col(gtx, kids...)
}

// richKey names one shaped paragraph: the cached spans and the width and
// scale they were shaped at.
type richKey struct {
	spans *richtext.SpanStyle
	n     int
	w     int
	scale float32
}

type richOps struct {
	ops  op.Ops
	call op.CallOp
	dims D
}

// rich lays out styled text. Shaping is the costly part of a frame, so a
// paragraph from the markdown cache is recorded once per width and replayed
// after that.
func (th *Theme) rich(gtx C, s []richtext.SpanStyle) D {
	if len(s) == 0 {
		return D{}
	}
	k := richKey{&s[0], len(s), gtx.Constraints.Max.X, gtx.Metric.PxPerDp}
	if c := th.richCache[k]; c != nil {
		c.call.Add(gtx.Ops)
		return c.dims
	}
	if th.richCache == nil || len(th.richCache) > 4000 {
		th.richCache = map[richKey]*richOps{}
	}
	c := &richOps{}
	rgtx := gtx
	rgtx.Ops = &c.ops
	m := op.Record(&c.ops)
	t := richtext.Text(new(richtext.InteractiveText), th.Shaper, s...)
	t.LineHeightScale = 1.42
	c.dims = t.Layout(rgtx)
	c.call = m.Stop()
	th.richCache[k] = c
	c.call.Add(gtx.Ops)
	return c.dims
}

// CodeBlock is monospace text on the deepest surface. maxLines 0 shows all.
func (th *Theme) CodeBlock(gtx C, code string, maxLines int) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: ColWindow, Border: ColBorder, Radius: 5, Pad: padXY(12, 9)}.Layout(gtx, func(gtx C) D {
		return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: rgb(0xc9cfd8), MaxLines: maxLines, LineH: 1.45}, code)
	})
}

// shellLangs get the terminal icon in a code block's header, like T3.
var shellLangs = map[string]bool{"sh": true, "bash": true, "zsh": true, "shell": true, "console": true,
	"powershell": true, "ps1": true, "pwsh": true, "cmd": true, "bat": true, "fish": true}

// codeCard is a reply's code block the way T3 draws it: a header with the
// language and a copy button over selectable code.
func (th *Theme) codeCard(gtx C, id, lang, code string) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return Box{Bg: ColWindow, Border: ColBorder, Radius: 8}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return Col(gtx,
			rigid(func(gtx C) D {
				return padXY(10, 5).Layout(gtx, func(gtx C) D {
					ic := IconFile
					if shellLangs[strings.ToLower(lang)] {
						ic = IconTerminal
					}
					return Row(gtx,
						rigid(Middle(IconW(ic, ColDim, 14))),
						rigid(Spacer(7, 0)),
						rigid(Middle(func(gtx C) D { return th.Small(gtx, ColDim, lang) })),
						flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
						rigid(th.runButton(id, lang, code)),
						rigid(th.copyButton(id, code)),
					)
				})
			}),
			rigid(func(gtx C) D {
				hline(gtx, ColBorder, 0, 0, gtx.Constraints.Max.X)
				return D{Size: image.Pt(gtx.Constraints.Max.X, max(1, dp(gtx, 1)))}
			}),
			rigid(func(gtx C) D {
				return padXY(12, 9).Layout(gtx, func(gtx C) D {
					return th.selectable(gtx, id, code, Mono(), SizeMono, rgb(0xc9cfd8), 1.45)
				})
			}),
		)
	})
}

// runButton runs a PowerShell or cmd block in a new console window on this
// PC, with a check for a moment after (red ✕ if it couldn't start).
func (th *Theme) runButton(id, lang, code string) W {
	return func(gtx C) D {
		if th.CanRun == nil || th.Run == nil || !th.CanRun(lang) {
			return D{}
		}
		rid := "run:" + id
		if th.in.runs == nil {
			th.in.runs = map[string][2]string{}
		}
		th.in.runs[rid] = [2]string{lang, code}
		ic, col := IconPlay, ColDim
		if at, ok := th.in.copied[rid]; ok {
			if left := 1500*time.Millisecond - gtx.Now.Sub(at); left > 0 {
				ic, col = IconCheck, ColGreen
				if th.in.runFailed[rid] {
					ic, col = IconClose, ColRed
				}
				gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
			} else {
				delete(th.in.copied, rid)
			}
		}
		return th.Click(rid, func(gtx C) D {
			return pad(4, 4, 4, 4).Layout(gtx, IconW(ic, col, 14))
		})(gtx)
	}
}

// copyButton puts text on the clipboard when clicked, and shows a check
// for a moment after.
func (th *Theme) copyButton(id, text string) W {
	return func(gtx C) D {
		cid := "copy:" + id
		if th.in.copies == nil {
			th.in.copies = map[string]string{}
		}
		if th.in.copied == nil {
			th.in.copied = map[string]time.Time{}
		}
		th.in.copies[cid] = text
		ic, col := IconCopy, ColDim
		if at, ok := th.in.copied[cid]; ok {
			if left := 1500*time.Millisecond - gtx.Now.Sub(at); left > 0 {
				ic, col = IconCheck, ColGreen
				gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
			} else {
				delete(th.in.copied, cid)
			}
		}
		return th.Click(cid, func(gtx C) D {
			return pad(4, 4, 4, 4).Layout(gtx, IconW(ic, col, 14))
		})(gtx)
	}
}
