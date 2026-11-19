package ui

import (
	"image"
	"image/color"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"gioui.org/font"
	"gioui.org/op"
)

// Quoted responses (packages/shared/src/assistantCitations.ts,
// AssistantCitationChip.tsx; docs/ui/06-timeline.md §3.3, §5.13): a
// message that cites a reply carries it as a link,
// [Assistant quote](t3-citation://v1/<environment>/<thread>/<message>?…),
// and shows it as a card of the passage, which goes to the reply.

// citation is a quoted response: where it's from, the passage and what's
// around it, and the comment sent with it.
type citation struct {
	env, thread, message          string
	text, prefix, suffix, comment string
}

var citeOffset = regexp.MustCompile(`^\d{1,16}$`)

// parseCitation is T3's parseAssistantCitationHref, with the checks of
// its AssistantCitation schema.
func parseCitation(href string) (citation, bool) {
	if !strings.HasPrefix(href, "t3-citation://v1/") {
		return citation{}, false
	}
	u, err := url.Parse(href)
	if err != nil || u.Scheme != "t3-citation" || u.Host != "v1" || u.User != nil || u.Fragment != "" {
		return citation{}, false
	}
	parts := strings.Split(strings.TrimPrefix(u.EscapedPath(), "/"), "/")
	if len(parts) != 3 {
		return citation{}, false
	}
	q := u.Query()
	want, got := 5, 0
	if _, ok := q["comment"]; ok {
		want++
	}
	for _, v := range q {
		got += len(v)
	}
	if got != want {
		return citation{}, false
	}
	for _, k := range []string{"text", "start", "end", "prefix", "suffix"} {
		if len(q[k]) != 1 {
			return citation{}, false
		}
	}
	start, end := q.Get("start"), q.Get("end")
	if !citeOffset.MatchString(start) || !citeOffset.MatchString(end) {
		return citation{}, false
	}
	s, _ := strconv.ParseInt(start, 10, 64)
	e, _ := strconv.ParseInt(end, 10, 64)
	c := citation{text: q.Get("text"), prefix: q.Get("prefix"), suffix: q.Get("suffix"), comment: q.Get("comment")}
	long := func(s string, n int) bool { return len(utf16.Encode([]rune(s))) > n }
	if e <= s || e > 1<<53-1 || strings.TrimSpace(c.text) == "" || long(c.text, 8000) || long(c.comment, 8000) ||
		long(c.prefix, 32) || long(c.suffix, 32) {
		return citation{}, false
	}
	ids := []*string{&c.env, &c.thread, &c.message}
	for i, p := range parts {
		if *ids[i], err = url.PathUnescape(p); err != nil || *ids[i] == "" || long(*ids[i], 4096) {
			return citation{}, false
		}
	}
	return c, true
}

// flat is T3's: white space as single spaces.
func flat(s string) string { return strings.Join(strings.Fields(s), " ") }

// citeID is the card's click target: "cite:<environment>/<thread>/<message>",
// each escaped.
func citeID(c *citation) string {
	return "cite:" + url.PathEscape(c.env) + "/" + url.PathEscape(c.thread) + "/" + url.PathEscape(c.message)
}

// citeText is the card's preview on bg: the whole reply, or its passage
// marked between "…prefix" and "suffix…", in 14/20 muted, clamped to two
// lines.
func (th *Theme) citeText(c *citation, bg color.NRGBA) *inlineText {
	muted := over(P.MutedForeground, bg)
	p := &inlineText{font: UI(font.Normal), size: 14, lineH: 20, maxLines: 2, ellipsis: muted,
		markBg: over(faded(P.Foreground, 0x1f), bg)}
	run := func(s string, mark bool) {
		r := inlineRun{text: s, font: UI(font.Normal), size: 14, color: muted, mark: mark}
		if mark {
			r.color = over(P.Foreground, bg)
		}
		p.runs = append(p.runs, r)
	}
	if c.prefix == "" && c.suffix == "" {
		run(flat(c.text), false)
	} else {
		if c.prefix != "" {
			run("…"+flat(c.prefix), false)
		}
		run(flat(c.text), true)
		if c.suffix != "" {
			run(flat(c.suffix)+"…", false)
		}
	}
	return th.keepText("cite\x00"+citeID(c)+"\x00"+c.prefix+"\x00"+c.text+"\x00"+c.suffix, Txt{Size: 14, Color: bg}, p)
}

// citeFill is the card's black: 10% on the bubble, 14% hovered; 35% and
// 50% in dark.
func citeFill(hover bool) color.NRGBA {
	a := map[[2]bool]uint8{{false, false}: 0x1a, {false, true}: 0x24, {true, false}: 0x59, {true, true}: 0x80}[[2]bool{dark, hover}]
	return alpha(rgb(0), a)
}

// citeCard is AssistantCitationEmbed: the preview on its fill, radius 8,
// padded 10 by 6, as wide as the text; the comment sent with it 6dp under
// it in the text's style. Clicking the card goes to the reply.
func (th *Theme) citeCard(gtx C, c *citation, st mdStyle) int {
	w := gtx.Constraints.Max.X
	id := citeID(c)
	cl, _ := th.events(gtx, id)
	bg := over(citeFill(cl.Hovered()), st.on)
	px, py := dp(gtx, 10), dp(gtx, 6)
	l := th.layoutInline(gtx, th.citeText(c, bg), w-2*px)
	h := l.size.Y + 2*py
	fill(gtx, bg, image.Rect(0, 0, w, h), dp(gtx, 8))
	off := op.Offset(image.Pt(px, py)).Push(gtx.Ops)
	l.call.Add(gtx.Ops)
	off.Pop()
	th.hit(gtx, id, image.Pt(w, h))
	if c.comment == "" {
		return h
	}
	co := op.Offset(image.Pt(0, h+dp(gtx, 6))).Push(gtx.Ops)
	ch := th.plainText(gtx, c.comment, Txt{Size: 14, Color: st.color}, 22.75)
	co.Pop()
	return h + dp(gtx, 6) + ch
}

// citeNatural is how wide the card would like to be, at most maxW: its
// preview on one line, or its comment's longest line.
func (th *Theme) citeNatural(gtx C, c *citation, st mdStyle, maxW int) int {
	px := dp(gtx, 10)
	w := th.layoutInline(gtx, th.citeText(c, over(citeFill(false), st.on)), maxW-2*px).size.X + 2*px
	if c.comment != "" {
		w = max(w, th.layoutInline(gtx, th.plainPara(c.comment, Txt{Size: 14, Color: st.color}, 22.75), maxW).size.X)
	}
	return min(w, maxW)
}
