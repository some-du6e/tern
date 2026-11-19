package diff

import (
	"unicode"
	"unicode/utf8"
)

// Words is the word-level change between a deleted line and the added
// line it pairs with, as Pierre's "word-alt" draws it (getDiffFileInput,
// nn/en): a word diff that keeps whitespace as tokens (jsdiff's
// diffWordsWithSpace), each side's changed tokens joined into runs, and a
// one-character unchanged gap after a changed run taken into the run.
func Words(del, add string) (dr, ar []Range) {
	a, b := tokens(del), tokens(add)
	parts := diffTokens(a, b)
	type run struct {
		changed bool
		n       int // bytes
	}
	var ds, as []run
	put := func(side *[]run, changed bool, text string, last bool) {
		s := *side
		if len(s) == 0 || last {
			*side = append(s, run{changed, len(text)})
			return
		}
		top := &s[len(s)-1]
		neutral, topNeutral := !changed, !top.changed
		if neutral == topNeutral || neutral && utf8.RuneCountInString(text) == 1 && !topNeutral {
			top.n += len(text)
			return
		}
		*side = append(s, run{changed, len(text)})
	}
	for i, p := range parts {
		last := i == len(parts)-1
		switch p.op {
		case 0:
			put(&ds, false, p.text, last)
			put(&as, false, p.text, last)
		case -1:
			put(&ds, true, p.text, last)
		case 1:
			put(&as, true, p.text, last)
		}
	}
	ranges := func(rs []run) []Range {
		var out []Range
		at := 0
		for _, r := range rs {
			if r.changed {
				out = append(out, Range{at, at + r.n})
			}
			at += r.n
		}
		return out
	}
	return ranges(ds), ranges(as)
}

// wordChar is jsdiff's extendedWordChars: ASCII letters, digits and _, and
// the Latin letters with diacritics.
func wordChar(r rune) bool {
	switch {
	case r < 0x80:
		return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
	case r >= 0xc0 && r <= 0xff && r != 0xd7 && r != 0xf7,
		r >= 0xf8 && r <= 0x2c6, r >= 0x2c8 && r <= 0x2d7, r >= 0x2de && r <= 0x2ff, r >= 0x1e00 && r <= 0x1eff:
		return true
	}
	return false
}

// tokens splits a line the way diffWordsWithSpace does: runs of word
// characters, runs of other whitespace than newlines, and every other
// character alone.
func tokens(s string) []string {
	var out []string
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		j := i + n
		switch {
		case wordChar(r):
			for j < len(s) {
				r, n := utf8.DecodeRuneInString(s[j:])
				if !wordChar(r) {
					break
				}
				j += n
			}
		case unicode.IsSpace(r) && r != '\n' && r != '\r':
			for j < len(s) {
				r, n := utf8.DecodeRuneInString(s[j:])
				if !unicode.IsSpace(r) || r == '\n' || r == '\r' {
					break
				}
				j += n
			}
		}
		out = append(out, s[i:j])
		i = j
	}
	return out
}

// part is a run of tokens kept (op 0), removed (-1) or added (1).
type part struct {
	op   int
	text string
}

// maxEdits bounds the word diff's work: lines further apart than this
// many tokens change as a whole.
const maxEdits = 400

// diffTokens is the shortest edit from a to b (Myers' O(ND) diff), as
// runs: removals before additions where they meet, like jsdiff.
func diffTokens(a, b []string) []part {
	n, m := len(a), len(b)
	maxD := min(n+m, maxEdits)
	off := maxD + 1
	v := make([]int, 2*maxD+3)
	// trace[d] is v as round d found it, the furthest x on each diagonal
	// after d-1 edits, kept for diagonals -d-1 to d+1.
	var trace [][]int
	done := false
	for d := 0; d <= maxD && !done; d++ {
		trace = append(trace, append([]int(nil), v[off-d-1:off+d+2]...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || k != d && v[off+k-1] < v[off+k+1] {
				x = v[off+k+1]
			} else {
				x = v[off+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[off+k] = x
			if x >= n && y >= m {
				done = true
				break
			}
		}
	}
	if !done {
		var out []part
		if n > 0 {
			out = append(out, part{-1, join(a)})
		}
		if m > 0 {
			out = append(out, part{1, join(b)})
		}
		return out
	}
	// Walk back from the end, collecting single-token edits.
	type edit struct {
		op  int
		tok string
	}
	var rev []edit
	x, y := n, m
	for d := len(trace) - 1; d > 0; d-- {
		at := func(k int) int { return trace[d][k+d+1] }
		k := x - y
		var pk int
		if k == -d || k != d && at(k-1) < at(k+1) {
			pk = k + 1
		} else {
			pk = k - 1
		}
		px := at(pk)
		py := px - pk
		for x > px && y > py {
			x--
			y--
			rev = append(rev, edit{0, a[x]})
		}
		if x == px {
			y--
			rev = append(rev, edit{1, b[y]})
		} else {
			x--
			rev = append(rev, edit{-1, a[x]})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		rev = append(rev, edit{0, a[x]})
	}
	// Join runs; within a stretch of changes, removals go first.
	var out []part
	push := func(op int, tok string) {
		if len(out) > 0 && out[len(out)-1].op == op {
			out[len(out)-1].text += tok
			return
		}
		out = append(out, part{op, tok})
	}
	var dels, adds string
	flush := func() {
		if dels != "" {
			push(-1, dels)
		}
		if adds != "" {
			push(1, adds)
		}
		dels, adds = "", ""
	}
	for i := len(rev) - 1; i >= 0; i-- {
		e := rev[i]
		switch e.op {
		case -1:
			dels += e.tok
		case 1:
			adds += e.tok
		default:
			flush()
			push(0, e.tok)
		}
	}
	flush()
	return out
}

func join(toks []string) string {
	n := 0
	for _, t := range toks {
		n += len(t)
	}
	b := make([]byte, 0, n)
	for _, t := range toks {
		b = append(b, t...)
	}
	return string(b)
}
