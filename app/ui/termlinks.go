package ui

import (
	"regexp"
	"strings"
)

// Links in the terminal (terminal-links.ts, docs/ui/09-terminal.md §5.9):
// URLs found in a logical line, its soft-wrapped rows joined. T3 finds
// paths as well, but a path only becomes a link once the server's file
// metadata says it's there, and opens in an editor; tern has neither yet,
// so it finds URLs.

// termURL is T3's URL_PATTERN: http or https up to whitespace, a quote or
// an angle bracket.
var termURL = regexp.MustCompile("(?i)https?://[^\\s\"'`<>]+")

// termMatch is a URL in a line, by byte offsets.
type termMatch struct {
	start, end int
	text       string
}

// termURLs is every URL in line, as extractTerminalLinks finds its url
// kind: trailing .,;!? dropped, and closing brackets that aren't matched.
func termURLs(line string) []termMatch {
	var out []termMatch
	for _, m := range termURL.FindAllStringIndex(line, -1) {
		if s := trimLinkEnd(line[m[0]:m[1]]); s != "" {
			out = append(out, termMatch{start: m[0], end: m[0] + len(s), text: s})
		}
	}
	return out
}

// trimLinkEnd is trimClosingDelimiters for a URL.
func trimLinkEnd(s string) string {
	s = strings.TrimRight(s, ".,;!?")
	for _, p := range [][2]string{{"(", ")"}, {"[", "]"}, {"{", "}"}} {
		for strings.HasSuffix(s, p[1]) && strings.Count(s, p[0]) < strings.Count(s, p[1]) {
			s = s[:len(s)-1]
		}
	}
	return s
}

// isTermURL reports whether a link's text is a web address, which opens
// in the browser; anything else would be a path.
func isTermURL(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}
