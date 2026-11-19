// Package diff reads unified Git patches the way T3's diff viewer
// (@pierre/diffs) does: the files, their hunks and lines, and the
// word-level changes between paired lines; and orders and classifies the
// files as T3's Diff panel and changed files card do
// (docs/ui/13-diff.md §5.4, §5.6, §8 note 3).
package diff

import (
	"errors"
	"strconv"
	"strings"
)

// ChangeType is what a file's change is (Pierre's change types).
type ChangeType int

const (
	Change ChangeType = iota
	New
	Deleted
	RenamePure    // moved, contents the same
	RenameChanged // moved and edited
)

// LineKind is a line's part in its hunk.
type LineKind int

const (
	Context LineKind = iota
	Deletion
	Addition
)

// Line is one line of a hunk, with its old and new numbers (0 for the side
// it isn't on) and, for a changed line paired with one on the other side,
// the byte ranges of its text that changed.
type Line struct {
	Kind      LineKind
	Old, New  int
	Text      string
	NoNewline bool // "\ No newline at end of file" follows it
	Words     []Range
}

// Range is the byte range [Start, End) of a line's text.
type Range struct{ Start, End int }

// Hunk is one "@@" block. Its lines are in the unified view's order: each
// change's deletions, then its additions.
type Hunk struct {
	DelStart, DelCount, AddStart, AddCount int
	// Hidden is how many unchanged lines sit between the previous hunk (or
	// the file's start) and this one: the separator above it.
	Hidden int
	Lines  []Line
}

// File is one file of a patch.
type File struct {
	Name, PrevName string // PrevName is the old path of a rename
	Type           ChangeType
	Hunks          []Hunk
	// Additions and Deletions count the changed lines.
	Additions, Deletions int
	Binary               bool
}

// Path is where the file is now, or was when it was deleted.
func (f File) Path() string {
	if f.Name != "" {
		return f.Name
	}
	return f.PrevName
}

// MaxWordLine is the longest line Pierre computes word changes for
// (maxLineDiffLength).
const MaxWordLine = 1000

// ErrNoFiles is a patch with nothing a viewer can draw.
var ErrNoFiles = errors.New("no files in the patch")

// Parse reads a unified patch: Git's (diff --git headers, renames, modes,
// quoted names, binary files) or a plain one (---/+++ only).
func Parse(patch string) ([]File, error) {
	lines := strings.Split(strings.ReplaceAll(patch, "\r\n", "\n"), "\n")
	var files []File
	var f *File
	var oldName, newName string
	flush := func() {
		if f == nil {
			return
		}
		finish(f, oldName, newName)
		files = append(files, *f)
		f = nil
	}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case strings.HasPrefix(l, "diff --git "):
			flush()
			f = &File{}
			oldName, newName = gitNames(l[len("diff --git "):])
		case strings.HasPrefix(l, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ "):
			if f == nil || len(f.Hunks) > 0 {
				flush()
				f = &File{}
			}
			oldName, newName = headerName(l[4:]), headerName(lines[i+1][4:])
			if oldName == "" {
				f.Type = New
			}
			if newName == "" {
				f.Type = Deleted
			}
			i++
		case f == nil:
		case strings.HasPrefix(l, "new file mode"):
			f.Type = New
		case strings.HasPrefix(l, "deleted file mode"):
			f.Type = Deleted
		case strings.HasPrefix(l, "rename from "), strings.HasPrefix(l, "copy from "):
			_, rest, _ := strings.Cut(l, " from ")
			oldName = unquote(rest)
			f.Type = RenamePure
		case strings.HasPrefix(l, "rename to "), strings.HasPrefix(l, "copy to "):
			_, rest, _ := strings.Cut(l, " to ")
			newName = unquote(rest)
			f.Type = RenamePure
		case strings.HasPrefix(l, "Binary files ") || l == "GIT binary patch":
			f.Binary = true
		case strings.HasPrefix(l, "@@ "):
			h, ok := hunkHeader(l)
			if !ok {
				return nil, errors.New("bad hunk header: " + l)
			}
			i = readHunk(&h, lines, i+1) - 1
			f.Hunks = append(f.Hunks, h)
		}
	}
	flush()
	if len(files) == 0 {
		return nil, ErrNoFiles
	}
	return files, nil
}

// finish names a parsed file, settles its type and counts its lines and
// the unchanged lines before each hunk.
func finish(f *File, oldName, newName string) {
	switch f.Type {
	case New:
		f.Name = newName
	case Deleted:
		f.Name = oldName
	default:
		f.Name = newName
		if oldName != "" && oldName != newName {
			f.PrevName = oldName
			f.Type = RenamePure
		}
	}
	if f.Type == RenamePure && len(f.Hunks) > 0 {
		f.Type = RenameChanged
	}
	end := 0
	for i := range f.Hunks {
		h := &f.Hunks[i]
		h.Hidden = max(0, zeroStart(h.AddStart, h.AddCount)-end)
		end = zeroStart(h.AddStart, h.AddCount) + h.AddCount
		for _, l := range h.Lines {
			switch l.Kind {
			case Addition:
				f.Additions++
			case Deletion:
				f.Deletions++
			}
		}
	}
}

// zeroStart is a hunk side's first line from zero: an empty side's start
// names the line before it.
func zeroStart(start, count int) int {
	if count == 0 {
		return start
	}
	return start - 1
}

// hunkHeader reads "@@ -a,b +c,d @@ context".
func hunkHeader(l string) (Hunk, bool) {
	body, _, ok := strings.Cut(l[3:], " @@")
	if !ok {
		return Hunk{}, false
	}
	del, add, ok := strings.Cut(body, " ")
	if !ok || !strings.HasPrefix(del, "-") || !strings.HasPrefix(add, "+") {
		return Hunk{}, false
	}
	span := func(s string) (int, int, bool) {
		a, b, two := strings.Cut(s, ",")
		start, err := strconv.Atoi(a)
		if err != nil {
			return 0, 0, false
		}
		if !two {
			return start, 1, true
		}
		n, err := strconv.Atoi(b)
		return start, n, err == nil
	}
	h := Hunk{}
	var ok1, ok2 bool
	h.DelStart, h.DelCount, ok1 = span(del[1:])
	h.AddStart, h.AddCount, ok2 = span(add[1:])
	return h, ok1 && ok2
}

// readHunk reads h's lines from lines[i:] until its counts are met, and
// returns the index after them. Each run of changed lines is regrouped as
// its deletions, then its additions, and its pairs get their word changes.
func readHunk(h *Hunk, lines []string, i int) int {
	old, nw := h.DelStart, h.AddStart
	dels, adds := h.DelCount, h.AddCount
	var dl, al []Line
	flush := func() {
		pairWords(dl, al)
		h.Lines = append(append(h.Lines, dl...), al...)
		dl, al = nil, nil
	}
	last := func() *Line {
		switch {
		case len(al) > 0:
			return &al[len(al)-1]
		case len(dl) > 0:
			return &dl[len(dl)-1]
		case len(h.Lines) > 0:
			return &h.Lines[len(h.Lines)-1]
		}
		return nil
	}
	for ; i < len(lines); i++ {
		l := lines[i]
		if strings.HasPrefix(l, `\`) {
			if p := last(); p != nil {
				p.NoNewline = true
			}
			continue
		}
		if dels == 0 && adds == 0 {
			break
		}
		switch {
		case strings.HasPrefix(l, "-") && dels > 0:
			if len(al) > 0 {
				flush()
			}
			dl = append(dl, Line{Kind: Deletion, Old: old, Text: l[1:]})
			old++
			dels--
		case strings.HasPrefix(l, "+") && adds > 0:
			al = append(al, Line{Kind: Addition, New: nw, Text: l[1:]})
			nw++
			adds--
		case (l == "" || strings.HasPrefix(l, " ")) && dels > 0 && adds > 0:
			flush()
			text := ""
			if l != "" {
				text = l[1:]
			}
			h.Lines = append(h.Lines, Line{Kind: Context, Old: old, New: nw, Text: text})
			old++
			nw++
			dels--
			adds--
		default:
			// A truncated or malformed hunk: keep what it had.
			flush()
			return i
		}
	}
	flush()
	return i
}

// pairWords gives each deletion and the addition at the same place in its
// change its word changes, as Pierre pairs them.
func pairWords(dl, al []Line) {
	for i := 0; i < len(dl) && i < len(al); i++ {
		if len(dl[i].Text) > MaxWordLine || len(al[i].Text) > MaxWordLine {
			continue
		}
		dl[i].Words, al[i].Words = Words(dl[i].Text, al[i].Text)
	}
}

// gitNames reads "a/x b/y" from a diff --git line. Names with spaces are
// ambiguous there; the ---/+++ and rename lines that follow settle them.
func gitNames(s string) (string, string) {
	if strings.HasPrefix(s, `"`) {
		end := quotedEnd(s)
		a := unquote(s[:end])
		b := unquote(strings.TrimSpace(s[end:]))
		return strip(a), strip(b)
	}
	if i := strings.Index(s, " b/"); i >= 0 {
		return strip(s[:i]), strip(s[i+1:])
	}
	if i := strings.LastIndex(s, ` "`); i >= 0 {
		return strip(s[:i]), strip(unquote(s[i+1:]))
	}
	a, b, _ := strings.Cut(s, " ")
	return strip(a), strip(b)
}

// headerName reads a ---/+++ name, "" for /dev/null.
func headerName(s string) string {
	if i := strings.IndexByte(s, '\t'); i >= 0 && !strings.HasPrefix(s, `"`) {
		s = s[:i]
	}
	s = unquote(strings.TrimRight(s, " "))
	if s == "/dev/null" {
		return ""
	}
	return strip(s)
}

func strip(s string) string {
	for _, p := range []string{"a/", "b/", "i/", "w/", "c/", "o/"} {
		if rest, ok := strings.CutPrefix(s, p); ok {
			return rest
		}
	}
	return s
}

// quotedEnd is the index after a C-quoted name at the start of s.
func quotedEnd(s string) int {
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return len(s)
}

// unquote decodes Git's C-style quoting of names with special bytes:
// "a/caf\303\251.txt" is a/café.txt.
func unquote(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	s = s[1 : len(s)-1]
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 == len(s) {
			b = append(b, c)
			continue
		}
		i++
		switch e := s[i]; e {
		case 'a':
			b = append(b, '\a')
		case 'b':
			b = append(b, '\b')
		case 't':
			b = append(b, '\t')
		case 'n':
			b = append(b, '\n')
		case 'v':
			b = append(b, '\v')
		case 'f':
			b = append(b, '\f')
		case 'r':
			b = append(b, '\r')
		case '0', '1', '2', '3':
			if i+2 < len(s) {
				if v, err := strconv.ParseUint(s[i:i+3], 8, 8); err == nil {
					b = append(b, byte(v))
					i += 2
					continue
				}
			}
			b = append(b, e)
		default:
			b = append(b, e)
		}
	}
	return string(b)
}
