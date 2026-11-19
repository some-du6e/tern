package core

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// T3's terminal drawer state (terminalUiStateStore.ts), one per thread: open
// or not, its height, its terminals and how they group into split panes,
// and which is active. Every change goes through these, which keep T3's
// rules: a group holds four at most, a closed terminal hands over to the
// next one in its group, and a drawer back at its defaults is forgotten.

const (
	termDefaultHeight = 280 // DEFAULT_THREAD_TERMINAL_HEIGHT
	termDefaultID     = "term-1"
	termGroupMax      = 4 // MAX_TERMINALS_PER_GROUP
)

// termGroup is a set of split panes, side by side or, when Vertical,
// stacked.
type termGroup struct {
	ID       string   `json:"id"`
	IDs      []string `json:"terminalIds"`
	Vertical bool     `json:"vertical,omitempty"`
}

// termState is a thread's drawer.
type termState struct {
	Open        bool        `json:"open"`
	Height      float32     `json:"height"`
	IDs         []string    `json:"terminalIds"`
	Active      string      `json:"activeTerminalId"`
	Groups      []termGroup `json:"terminalGroups"`
	ActiveGroup string      `json:"activeTerminalGroupId"`
}

// normalizeIDs trims ids and drops empty and repeated ones.
func normalizeIDs(ids []string) []string {
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// uniqueGroupID is base, or base-2, base-3 and on if that's taken.
func uniqueGroupID(base string, used map[string]bool) string {
	id := base
	for i := 2; used[id]; i++ {
		id = base + "-" + strconv.Itoa(i)
	}
	used[id] = true
	return id
}

// normalizeGroups keeps each of ids in one group: unknown and repeated ids
// leave their groups, empty groups go, and ids no group holds get one of
// their own.
func normalizeGroups(groups []termGroup, ids []string) []termGroup {
	if len(ids) == 0 {
		return nil
	}
	assigned, used := map[string]bool{}, map[string]bool{}
	var out []termGroup
	for _, g := range groups {
		var gids []string
		for _, id := range normalizeIDs(g.IDs) {
			if slices.Contains(ids, id) && !assigned[id] {
				gids = append(gids, id)
			}
		}
		if len(gids) == 0 {
			continue
		}
		for _, id := range gids {
			assigned[id] = true
		}
		base := strings.TrimSpace(g.ID)
		if base == "" {
			base = "group-" + gids[0]
		}
		out = append(out, termGroup{ID: uniqueGroupID(base, used), IDs: gids, Vertical: g.Vertical})
	}
	for _, id := range ids {
		if !assigned[id] {
			out = append(out, termGroup{ID: uniqueGroupID("group-"+id, used), IDs: []string{id}})
		}
	}
	return out
}

// groupOf is the index of the group holding id, or -1.
func groupOf(groups []termGroup, id string) int {
	return slices.IndexFunc(groups, func(g termGroup) bool { return slices.Contains(g.IDs, id) })
}

func groupIndex(groups []termGroup, id string) int {
	return slices.IndexFunc(groups, func(g termGroup) bool { return g.ID == id })
}

func cloneGroups(groups []termGroup) []termGroup {
	out := make([]termGroup, len(groups))
	for i, g := range groups {
		out[i] = termGroup{ID: g.ID, IDs: slices.Clone(g.IDs), Vertical: g.Vertical}
	}
	return out
}

func validHeight(h float32) bool { return h > 0 && !math.IsInf(float64(h), 0) }

// normalized is s with its ids, groups, active terminal and height made
// consistent.
func (s termState) normalized() termState {
	ids := normalizeIDs(s.IDs)
	active := s.Active
	if !slices.Contains(ids, active) {
		active = ""
		if len(ids) > 0 {
			active = ids[0]
		}
	}
	groups := normalizeGroups(s.Groups, ids)
	activeGroup := ""
	if groupIndex(groups, s.ActiveGroup) >= 0 {
		activeGroup = s.ActiveGroup
	} else if g := groupOf(groups, active); g >= 0 {
		activeGroup = groups[g].ID
	} else if len(groups) > 0 {
		activeGroup = groups[0].ID
	}
	h := s.Height
	if !validHeight(h) {
		h = termDefaultHeight
	}
	return termState{Open: s.Open, Height: h, IDs: ids, Active: active, Groups: groups, ActiveGroup: activeGroup}
}

// isDefault reports whether s is a closed, 280-tall drawer with nothing in
// it, which T3 doesn't keep.
func (s termState) isDefault() bool {
	n := s.normalized()
	return !n.Open && n.Height == termDefaultHeight && len(n.IDs) == 0 && n.ActiveGroup == ""
}

// add puts terminal id into the drawer and makes it active: in a group of
// its own, or split into the active group, after the active terminal,
// side by side or stacked. A full group refuses a fifth; splitting an
// empty drawer is the same as adding.
func (s termState) add(id string, split, vertical bool) termState {
	n := s.normalized()
	if strings.TrimSpace(id) == "" {
		return n
	}
	split = split && len(n.IDs) > 0
	isNew := !slices.Contains(n.IDs, id)
	ids := n.IDs
	if isNew {
		ids = append(slices.Clone(n.IDs), id)
	}
	groups := cloneGroups(n.Groups)
	if g := groupOf(groups, id); g >= 0 {
		groups[g].IDs = slices.DeleteFunc(groups[g].IDs, func(x string) bool { return x == id })
		if len(groups[g].IDs) == 0 {
			groups = slices.Delete(groups, g, g+1)
		}
	}
	used := map[string]bool{}
	for _, g := range groups {
		used[g.ID] = true
	}
	if !split {
		gid := uniqueGroupID("group-"+id, used)
		groups = append(groups, termGroup{ID: gid, IDs: []string{id}})
		return termState{Open: true, Height: n.Height, IDs: ids, Active: id, Groups: groups, ActiveGroup: gid}.normalized()
	}
	at := groupIndex(groups, n.ActiveGroup)
	if at < 0 {
		at = groupOf(groups, n.Active)
	}
	if at < 0 {
		groups = append(groups, termGroup{ID: uniqueGroupID("group-"+n.Active, used), IDs: []string{n.Active}})
		at = len(groups) - 1
	}
	dest := &groups[at]
	if !slices.Contains(dest.IDs, id) {
		if isNew && len(dest.IDs) >= termGroupMax {
			return n
		}
		if i := slices.Index(dest.IDs, n.Active); i >= 0 {
			dest.IDs = slices.Insert(dest.IDs, i+1, id)
		} else {
			dest.IDs = append(dest.IDs, id)
		}
	}
	dest.Vertical = vertical
	return termState{Open: true, Height: n.Height, IDs: ids, Active: id, Groups: groups, ActiveGroup: dest.ID}.normalized()
}

// setOpen opens or closes the drawer. Opening an empty one adds term-1.
func (s termState) setOpen(open bool) termState {
	n := s.normalized()
	if open && len(n.IDs) == 0 {
		return n.add(termDefaultID, false, false)
	}
	n.Open = open
	return n
}

// setHeight keeps a dragged height; nonsense heights are ignored.
func (s termState) setHeight(h float32) termState {
	n := s.normalized()
	if validHeight(h) {
		n.Height = h
	}
	return n
}

// activate makes id the active terminal, and its group the shown one.
func (s termState) activate(id string) termState {
	n := s.normalized()
	if !slices.Contains(n.IDs, id) {
		return n
	}
	if g := groupOf(n.Groups, id); g >= 0 {
		n.ActiveGroup = n.Groups[g].ID
	}
	n.Active = id
	return n
}

// ensure adds id if it isn't there, makes it active, and opens the drawer
// when open is set.
func (s termState) ensure(id string, open bool) termState {
	n := s
	if !slices.Contains(n.normalized().IDs, id) {
		n = n.add(id, false, false)
	}
	n = n.activate(id)
	if open {
		n = n.setOpen(true)
	}
	return n.normalized()
}

// close takes id out. The next terminal in its group takes over, else the
// one before it, else the one now where it was. Closing the last one
// resets the drawer, dragged height and all.
func (s termState) close(id string) termState {
	n := s.normalized()
	at := slices.Index(n.IDs, id)
	if at < 0 {
		return n
	}
	rest := slices.Delete(slices.Clone(n.IDs), at, at+1)
	if len(rest) == 0 {
		return termState{Height: termDefaultHeight}
	}
	next := n.Active
	if next == id {
		var gids []string
		if g := groupOf(n.Groups, id); g >= 0 {
			gids = n.Groups[g].IDs
		}
		i := slices.Index(gids, id)
		switch {
		case i+1 < len(gids):
			next = gids[i+1]
		case i > 0:
			next = gids[i-1]
		default:
			next = rest[min(at, len(rest)-1)]
		}
	}
	var groups []termGroup
	for _, g := range n.Groups {
		if ids := slices.DeleteFunc(slices.Clone(g.IDs), func(x string) bool { return x == id }); len(ids) > 0 {
			groups = append(groups, termGroup{ID: g.ID, IDs: ids, Vertical: g.Vertical})
		}
	}
	activeGroup := "group-" + next
	if g := groupOf(groups, next); g >= 0 {
		activeGroup = groups[g].ID
	} else if len(groups) > 0 {
		activeGroup = groups[0].ID
	}
	return termState{Open: n.Open, Height: n.Height, IDs: rest, Active: next, Groups: groups, ActiveGroup: activeGroup}.normalized()
}

// reconcile takes the server's list of the thread's terminals as the
// drawer's, keeping the groups and the active terminal where it can.
func (s termState) reconcile(ids []string) termState {
	n := s.normalized()
	if slices.Equal(n.IDs, ids) {
		return n
	}
	active := n.Active
	if !slices.Contains(ids, active) {
		active = ""
		if len(ids) > 0 {
			active = ids[0]
		}
	}
	groups := normalizeGroups(n.Groups, ids)
	activeGroup := ""
	if g := groupOf(groups, active); g >= 0 {
		activeGroup = groups[g].ID
	} else if len(groups) > 0 {
		activeGroup = groups[0].ID
	}
	return termState{Open: n.Open, Height: n.Height, IDs: ids, Active: active, Groups: groups, ActiveGroup: activeGroup}.normalized()
}

// shownGroups is the drawer's groups in the order the list draws them: by
// where their first terminal is in the drawer.
func (s termState) shownGroups() []termGroup {
	groups := cloneGroups(s.Groups)
	rank := func(g termGroup) int {
		r := len(s.IDs)
		for _, id := range g.IDs {
			if i := slices.Index(s.IDs, id); i >= 0 {
				r = min(r, i)
			}
		}
		return r
	}
	sort.SliceStable(groups, func(i, j int) bool { return rank(groups[i]) < rank(groups[j]) })
	return groups
}

// termNumber matches T3's terminal ids: term-N, terminal-N, or term-N
// with a UUID after it.
var termNumber = regexp.MustCompile(`(?i)^term(?:inal)?-(\d+)(?:-[\da-f]{8}(?:-[\da-f]{4}){3}-[\da-f]{12})?$`)

// termLabel is T3's getTerminalLabel: "Terminal N" for its ids, else the id.
func termLabel(id string) string {
	if m := termNumber.FindStringSubmatch(id); m != nil {
		return "Terminal " + m[1]
	}
	return id
}

// nextTermID is the lowest term-N none of ids has taken.
func nextTermID(ids []string) string {
	used := map[string]bool{}
	for _, id := range ids {
		if m := termNumber.FindStringSubmatch(id); m != nil {
			used[m[1]] = true
		}
	}
	n := 1
	for used[strconv.Itoa(n)] {
		n++
	}
	return "term-" + strconv.Itoa(n)
}

// naturalLess orders ids the way Intl.Collator with numeric does: runs of
// digits by their value, so term-2 comes before term-10.
func naturalLess(a, b string) bool {
	for a != "" && b != "" {
		da, db := unicode.IsDigit(rune(a[0])), unicode.IsDigit(rune(b[0]))
		if da && db {
			na, ra := leadingDigits(a)
			nb, rb := leadingDigits(b)
			if na != nb {
				return na < nb
			}
			a, b = ra, rb
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func leadingDigits(s string) (int, string) {
	i := 0
	for i < len(s) && unicode.IsDigit(rune(s[i])) {
		i++
	}
	n, _ := strconv.Atoi(s[:i])
	return n, s[i:]
}
