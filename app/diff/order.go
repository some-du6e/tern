package diff

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Compare orders names the way T3's Intl.Collator(undefined, {numeric:
// true, sensitivity: "base"}) does, near enough for paths: case and
// accents count for nothing, digit runs compare by value, and space and
// punctuation sort before digits, which sort before letters (ICU's root
// order).
func Compare(a, b string) int {
	for a != "" && b != "" {
		if da, db := digits(a), digits(b); da > 0 && db > 0 {
			if c := compareNumbers(a[:da], b[:db]); c != 0 {
				return c
			}
			a, b = a[da:], b[db:]
			continue
		}
		ra, na := utf8.DecodeRuneInString(a)
		rb, nb := utf8.DecodeRuneInString(b)
		if c := collationKey(ra) - collationKey(rb); c != 0 {
			return sign(c)
		}
		a, b = a[na:], b[nb:]
	}
	return sign(len(a) - len(b))
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

func digits(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

func compareNumbers(a, b string) int {
	a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		return sign(len(a) - len(b))
	}
	return strings.Compare(a, b)
}

// punctuation is ICU's root order of the ASCII punctuation and symbols.
const punctuation = "_-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$"

// collationKey is a rune's primary weight: space, then punctuation, then
// digits, then letters without case or accent, then the rest.
func collationKey(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case strings.ContainsRune(punctuation, r):
		return 1 + strings.IndexRune(punctuation, r)
	case r >= '0' && r <= '9':
		return 100 + int(r-'0')
	case unicode.IsLetter(r):
		return 200 + int(baseLetter(unicode.ToLower(r)))
	}
	return 100000 + int(r)
}

// baseLetter strips the Latin-1 letters' accents: é is e.
func baseLetter(r rune) rune {
	const from, to = "àáâãäåçèéêëìíîïñòóôõöøùúûüýÿ", "aaaaaaceeeeiiiinoooooouuuuyy"
	if i := strings.IndexRune(from, r); i >= 0 {
		return []rune(to)[utf8.RuneCountInString(from[:i])]
	}
	return r
}

// SortPaths sorts paths in place as the code view lists files: one flat,
// full-path sort.
func SortPaths(paths []string) {
	sort.SliceStable(paths, func(i, j int) bool { return Compare(paths[i], paths[j]) < 0 })
}

// FileStat is one changed file and its line counts.
type FileStat struct {
	Path                 string
	Additions, Deletions int
}

// CardNode is a row of the changed files card's tree (turnDiffTree.ts): a
// folder, with the totals of everything under it, or a file.
type CardNode struct {
	Dir                  bool
	Name, Path           string
	Additions, Deletions int
	Children             []CardNode
}

// CardTree builds the card's tree: each level's folders first, then its
// files, both by name; a chain of folders that each hold only one folder
// becomes a single "a/b/c" row with the deepest one's totals.
func CardTree(files []FileStat) []CardNode {
	type dir struct {
		node  CardNode
		dirs  map[string]*dir
		order []string
		files []CardNode
	}
	root := &dir{dirs: map[string]*dir{}}
	for _, f := range files {
		var segs []string
		for _, s := range strings.Split(f.Path, "/") {
			if s != "" {
				segs = append(segs, s)
			}
		}
		if len(segs) == 0 {
			continue
		}
		cur := root
		ancestors := []*dir{root}
		for _, s := range segs[:len(segs)-1] {
			next := cur.dirs[s]
			if next == nil {
				path := s
				if cur.node.Path != "" {
					path = cur.node.Path + "/" + s
				}
				next = &dir{node: CardNode{Dir: true, Name: s, Path: path}, dirs: map[string]*dir{}}
				cur.dirs[s] = next
				cur.order = append(cur.order, s)
			}
			cur = next
			ancestors = append(ancestors, cur)
		}
		cur.files = append(cur.files, CardNode{Name: segs[len(segs)-1], Path: strings.Join(segs, "/"), Additions: f.Additions, Deletions: f.Deletions})
		for _, a := range ancestors {
			a.node.Additions += f.Additions
			a.node.Deletions += f.Deletions
		}
	}
	byName := func(nodes []CardNode) {
		sort.SliceStable(nodes, func(i, j int) bool { return Compare(nodes[i].Name, nodes[j].Name) < 0 })
	}
	var build func(d *dir) []CardNode
	build = func(d *dir) []CardNode {
		var subs []CardNode
		for _, name := range d.order {
			sd := d.dirs[name]
			n := sd.node
			n.Children = build(sd)
			subs = append(subs, n)
		}
		byName(subs)
		for i := range subs {
			for len(subs[i].Children) == 1 && subs[i].Children[0].Dir {
				only := subs[i].Children[0]
				subs[i] = CardNode{Dir: true, Name: subs[i].Name + "/" + only.Name, Path: only.Path,
					Additions: only.Additions, Deletions: only.Deletions, Children: only.Children}
			}
		}
		fs := append([]CardNode(nil), d.files...)
		byName(fs)
		return append(subs, fs...)
	}
	return build(root)
}

// TreeEntry is a row of the Diff panel's file tree (@pierre/trees, as
// DiffFileTree.tsx configures it): a file, or a folder whose single-child
// folder chain shows as one row, its Segments "a", "b", "c".
type TreeEntry struct {
	Path     string // a folder's ends in "/"
	Segments []string
	Depth    int
	Dir      bool
	Open     bool // a folder's children follow
	Type     ChangeType
}

// FileTree lays out the file tree's rows for paths in the code view's
// order, each with its change type: a folder sits where its first file
// does in that order (diffFileTree.logic.ts), closed folders' children
// are left out.
func FileTree(paths []string, types map[string]ChangeType, closed map[string]bool) []TreeEntry {
	pos := map[string]int{}
	type node struct {
		path     string
		name     string
		dir      bool
		children map[string]*node
		order    []string
	}
	root := &node{children: map[string]*node{}}
	for i, p := range paths {
		if _, ok := pos[p]; !ok {
			pos[p] = i
		}
		segs := strings.Split(p, "/")
		cur, prefix := root, ""
		for j, s := range segs {
			last := j == len(segs)-1
			key := s
			if !last {
				prefix += s + "/"
				key += "/"
				if _, ok := pos[prefix]; !ok {
					pos[prefix] = i
				}
			}
			next := cur.children[key]
			if next == nil {
				path := p
				if !last {
					path = prefix
				}
				next = &node{path: path, name: s, dir: !last, children: map[string]*node{}}
				cur.children[key] = next
				cur.order = append(cur.order, key)
			}
			cur = next
		}
	}
	var out []TreeEntry
	var walk func(n *node, depth int)
	walk = func(n *node, depth int) {
		kids := make([]*node, 0, len(n.order))
		for _, k := range n.order {
			kids = append(kids, n.children[k])
		}
		sort.SliceStable(kids, func(i, j int) bool {
			if pi, pj := pos[kids[i].path], pos[kids[j].path]; pi != pj {
				return pi < pj
			}
			return kids[i].path < kids[j].path
		})
		for _, k := range kids {
			e := TreeEntry{Path: k.path, Segments: []string{k.name}, Depth: depth, Dir: k.dir}
			for k.dir && len(k.order) == 1 && k.children[k.order[0]].dir {
				k = k.children[k.order[0]]
				e.Path = k.path
				e.Segments = append(e.Segments, k.name)
			}
			if !k.dir {
				e.Type = types[k.path]
			}
			e.Open = k.dir && !closed[e.Path]
			out = append(out, e)
			if e.Open {
				walk(k, depth+1)
			}
		}
	}
	walk(root, 0)
	return out
}
