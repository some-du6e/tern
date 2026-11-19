package ui

import (
	"image"
	"image/color"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gioui.org/op"
)

// File links (packages/shared/src/markdownLinks.ts, fileLinks.ts, path.ts;
// apps/web/src/markdown-links.ts): a markdown link to a file, or a path in
// backticks, that T3 shows as a file chip. Relative paths need the
// thread's workspace to resolve against.

// fileChip is a file link as T3's chip shows it (MarkdownFileLink.tsx):
// its click target, the path its icon is picked by, its label, the place
// it names with its line and column (its tooltip, "Copy full path") and
// that inside the workspace ("Copy relative path", "" outside it), its
// ink and text size, and the surface under it.
type fileChip struct {
	id, icon, label, target, rel string
	ink, on                      color.NRGBA
	size                         float32
}

// chipInk is a file chip's colour: a document's kind, else a mention's
// (documentChipKindForFileName).
func chipInk(path string) color.NRGBA {
	name, _ := fileIcon(path)
	switch name {
	case "t3-file-icon-pdf":
		return P.ContextChipDocPdf
	case "t3-file-icon-word":
		return P.ContextChipDocWord
	case "t3-file-icon-spreadsheet":
		return P.ContextChipDocSpreadsheet
	case "t3-file-icon-slides":
		return P.ContextChipDocSlides
	case "t3-file-icon-epub":
		return P.ContextChipDocEpub
	}
	return P.ContextChipMention
}

// fileChipArea makes a file chip's box its button: the label underlined
// 0.2em under the baseline while hovered, the path in the code tooltip,
// and a click opening the copy menu T3's chip opens where it can't open
// the file (MarkdownFileLink.tsx:354-381, MarkdownFileLink.logic.ts).
func (th *Theme) fileChipArea(gtx C, lb linkBox) {
	f := lb.file
	c, _ := th.events(gtx, f.id)
	if c.Hovered() {
		y := lb.base + dp(gtx, 0.2*f.size)
		lead := dp(gtx, f.size) + dp(gtx, 0.2*f.size)
		fill(gtx, over(f.ink, f.on), image.Rect(lb.r.Min.X+lead, y, lb.r.Max.X, y+max(1, dp(gtx, 1))), 0)
	}
	off := op.Offset(lb.r.Min).Push(gtx.Ops)
	th.hit(gtx, f.id, lb.r.Size())
	th.codeTipAt(gtx, f.id, f.target, lb.r.Size())
	off.Pop()
	if th.clicked(f.id) {
		th.toggleMenu(f.id)
	}
	choices, values := []menuChoice{{label: "Copy full path"}}, []string{f.target}
	if f.rel != "" {
		choices, values = append([]menuChoice{{label: "Copy relative path"}}, choices...), append([]string{f.rel}, values...)
	}
	if i := th.rowMenu(gtx, f.id, image.Pt(lb.r.Max.X, lb.r.Max.Y+dp(gtx, 4)), false, choices); i >= 0 {
		th.copyNow(gtx, f.id+":copy", values[i])
	}
}

// filePos is a path and the place in the file a link names.
type filePos struct {
	path                  string
	line, column, endLine int
}

var (
	relativePrefix  = regexp.MustCompile(`^(~/|\.{1,2}/)`)
	relativeFile    = regexp.MustCompile(`^(?:[\p{L}\p{M}\p{N}._$+@()\[\]-]+(?: +[\p{L}\p{M}\p{N}._$+@()\[\]-]+)*/)+(?:[\p{L}\p{M}\p{N}._$+@()\[\]-]+(?: +[\p{L}\p{M}\p{N}._$+@()\[\]-]+)*(?::\d+(?::\d+|-\d+)?)?)?$`)
	relativeName    = regexp.MustCompile(`^[\p{L}\p{M}\p{N}._$+()\[\]-]+(?: +[\p{L}\p{M}\p{N}._$+()\[\]-]+)*\.[\p{L}\p{M}\p{N}_-]+(?::\d+(?::\d+|-\d+)?)?$`)
	externalScheme  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9+.-]*):(.*)$`)
	positionSuffix  = regexp.MustCompile(`:\d+(?::\d+|-\d+)?$`)
	positionOnly    = regexp.MustCompile(`^\d+(?::\d+|-\d+)?$`)
	positionCapture = regexp.MustCompile(`:(\d+)(?::(\d+)|-(\d+))?$`)
	positionHash    = regexp.MustCompile(`(?i)^#L(\d+)(?:C(\d+)|-L?(\d+))?$`)
	elidedSegment   = regexp.MustCompile(`(?:^|[\\/])(?:\.{3,}|…)(?:[\\/]|$)`)
	fileExtension   = regexp.MustCompile(`\.[A-Za-z0-9_-]+$`)
	anyExtension    = regexp.MustCompile(`\.[^./\\]+$`)
	versionSuffix   = regexp.MustCompile(`\d\.\d[^.]*$`)
	numericDotted   = regexp.MustCompile(`^\d+(?:\.\d+)+$`)
	windowsDrive    = regexp.MustCompile(`^[a-zA-Z]:([/\\]|$)`)
	slashDrive      = regexp.MustCompile(`^/[A-Za-z]:[\\/]`)
)

var posixRoots = []string{"/Users/", "/home/", "/tmp/", "/var/", "/etc/", "/opt/", "/mnt/", "/Volumes/", "/private/", "/root/",
	"/usr/", "/bin/", "/sbin/", "/lib/", "/lib64/", "/srv/", "/dev/", "/proc/", "/sys/", "/run/", "/boot/", "/media/",
	"/workspace/", "/workspaces/"}

var extensionless = map[string]bool{"Makefile": true, "makefile": true, "GNUmakefile": true, "Dockerfile": true, "Containerfile": true,
	"Justfile": true, "justfile": true, "Rakefile": true, "Gemfile": true, "Procfile": true, "Brewfile": true, "Caddyfile": true,
	"Vagrantfile": true, "Jenkinsfile": true, "Podfile": true, "Fastfile": true, "BUILD": true, "WORKSPACE": true, "LICENSE": true,
	"LICENCE": true, "COPYING": true, "NOTICE": true, "AUTHORS": true, "CONTRIBUTORS": true, "CHANGELOG": true, "README": true,
	"CODEOWNERS": true}

var (
	genericTLDs = map[string]bool{"com": true, "net": true, "org": true, "io": true, "dev": true, "app": true, "ai": true, "co": true,
		"edu": true, "gov": true, "mil": true, "info": true, "biz": true, "xyz": true, "me": true, "tv": true, "gg": true, "chat": true,
		"cloud": true, "site": true, "online": true, "tech": true, "store": true, "link": true}
	// .cc is C++ as often as a country: allocator.cc:217 is a file.
	countryTLDs = map[string]bool{"uk": true, "de": true, "fr": true, "nl": true, "se": true, "no": true, "fi": true, "dk": true,
		"pl": true, "ch": true, "at": true, "be": true, "es": true, "it": true, "pt": true, "eu": true, "us": true, "ca": true, "au": true,
		"nz": true, "jp": true, "kr": true, "cn": true, "br": true, "ru": true, "mx": true, "ie": true, "cz": true, "tr": true, "sg": true,
		"hk": true, "cc": true}
)

func isWindowsAbsPath(s string) bool {
	return strings.HasPrefix(s, `\\`) || windowsDrive.MatchString(s)
}
func isAbsPath(s string) bool { return strings.HasPrefix(s, "/") || isWindowsAbsPath(s) }

// isNetworkPath: Windows takes any two leading separators for a share.
func isNetworkPath(s string) bool {
	sep := func(c byte) bool { return c == '/' || c == '\\' }
	return len(s) >= 2 && sep(s[0]) && sep(s[1])
}

func isRelativeFilePath(s string) bool {
	return relativePrefix.MatchString(s) || !strings.HasPrefix(s, "/") && !isWindowsAbsPath(s)
}

// stripSlashDrive drops the "/" a URL puts before "C:/".
func stripSlashDrive(s string) string {
	if slashDrive.MatchString(s) {
		return s[1:]
	}
	return s
}

// decodeOr is s percent-decoded, or s when it doesn't decode.
func decodeOr(s string) string {
	if d, err := url.PathUnescape(s); err == nil {
		return d
	}
	return s
}

func hasPositionSuffix(s string) bool    { return positionSuffix.MatchString(s) }
func trimPositionSuffix(s string) string { return positionSuffix.ReplaceAllString(s, "") }

// fileBasename is a path's last segment.
func fileBasename(p string) string {
	t := strings.TrimRight(p, `/\`)
	if t == "" {
		return p
	}
	return t[max(strings.LastIndex(t, "/"), strings.LastIndex(t, `\`))+1:]
}

// splitFilePos takes a ":12", ":12:4" or ":12-20" off path, or reads a
// "#L12" hash.
func splitFilePos(path, hash string) filePos {
	m := positionCapture.FindStringSubmatch(path)
	suffix := m != nil
	if !suffix {
		m = positionHash.FindStringSubmatch(hash)
	}
	if m == nil {
		return filePos{path: path}
	}
	out := filePos{path: path}
	if suffix {
		out.path = path[:len(path)-len(m[0])]
	}
	line, _ := strconv.Atoi(m[1])
	col, _ := strconv.Atoi(m[2])
	end, _ := strconv.Atoi(m[3])
	if line > 0 {
		out.line = line
		if end > line {
			out.endLine = end
		}
	}
	if col > 0 {
		out.column = col
	}
	return out
}

// String is the path with its line and column, as editors open it.
func (p filePos) String() string {
	if p.line == 0 {
		return p.path
	}
	s := p.path + ":" + strconv.Itoa(p.line)
	if p.column > 0 {
		s += ":" + strconv.Itoa(p.column)
	}
	return s
}

// looksLikeHostname: the first segment of "example.com/x" is a host.
func looksLikeHostname(seg string, position bool) bool {
	if strings.HasPrefix(seg, ".") {
		return false
	}
	l := strings.ToLower(seg)
	if l == "localhost" || numericDotted.MatchString(seg) {
		return true
	}
	labels := strings.Split(l, ".")
	if len(labels) < 2 {
		return false
	}
	last := labels[len(labels)-1]
	return genericTLDs[last] || !position && countryTLDs[last]
}

// inlineCodePath is T3's inlineCodeFilePathCandidate: code that names a
// file, or "".
func inlineCodePath(code string) string {
	t := strings.TrimSpace(code)
	if t == "" || strings.ContainsAny(t, " \t\n\r\f\v`") {
		return ""
	}
	c := t
	if !isWindowsAbsPath(t) {
		c = strings.ReplaceAll(t, `\`, "/")
	}
	position := hasPositionSuffix(c)
	if !position && !strings.ContainsAny(c, `\/`) || elidedSegment.MatchString(trimPositionSuffix(c)) {
		return ""
	}
	if !relativePrefix.MatchString(c) && !strings.HasPrefix(c, "/") && !isWindowsAbsPath(c) {
		bare := trimPositionSuffix(c)
		// A trailing slash names a folder; inline code wants a file.
		if strings.HasSuffix(bare, "/") || looksLikeHostname(strings.Split(bare, "/")[0], position) {
			return ""
		}
		segs := strings.FieldsFunc(strings.TrimRight(bare, `/\`), func(r rune) bool { return r == '/' || r == '\\' })
		base := ""
		if len(segs) > 0 {
			base = segs[len(segs)-1]
		}
		if versionSuffix.MatchString(base) {
			return ""
		}
		// A ":line" alone doesn't make I/W217:220 a file.
		if !fileExtension.MatchString(base) && (!position || strings.Contains(bare, "/") && !anyExtension.MatchString(base) && !extensionless[base]) {
			return ""
		}
	}
	return c
}

func looksLikeFilePath(path, authored string) bool {
	switch {
	case isWindowsAbsPath(path) || relativePrefix.MatchString(path):
		return true
	case strings.HasPrefix(path, "/"):
		if hasPositionSuffix(authored) {
			return true
		}
		for _, r := range posixRoots {
			if strings.HasPrefix(authored, r) {
				return true
			}
		}
		base := authored[strings.LastIndex(authored, "/")+1:]
		return extensionless[base] || fileExtension.MatchString(base)
	case extensionless[path]:
		return true
	}
	return relativeFile.MatchString(authored) || relativeName.MatchString(path)
}

func hasExternalScheme(path string) bool {
	if isWindowsAbsPath(path) {
		return false
	}
	m := externalScheme.FindStringSubmatch(path)
	return m != nil && (strings.HasPrefix(m[2], "//") || !positionOnly.MatchString(m[2]))
}

// parseFileURL is a file: URL's path, still escaped, and its hash.
func parseFileURL(href string) (path, hash string, ok bool) {
	u, err := url.Parse(href)
	if err != nil || !strings.EqualFold(u.Scheme, "file") {
		return "", "", false
	}
	path = u.EscapedPath()
	if host := u.Hostname(); host != "" && !strings.EqualFold(host, "localhost") {
		path = `\\` + host + strings.ReplaceAll(path, "/", `\`)
	}
	if path == "" {
		return "", "", false
	}
	if u.Fragment != "" {
		hash = "#" + u.EscapedFragment()
	}
	return stripSlashDrive(path), hash, true
}

// parseFileLink is T3's parseMarkdownFileLink: the file a link's
// destination names, if it names one.
func parseFileLink(href string) (filePos, bool) {
	n := strings.TrimSpace(href)
	if strings.HasPrefix(n, "<") && strings.HasSuffix(n, ">") {
		n = n[1 : len(n)-1]
	}
	if n == "" || strings.HasPrefix(n, "#") || strings.HasPrefix(n, "//") {
		return filePos{}, false
	}
	path, hash, ok := "", "", false
	if strings.HasPrefix(strings.ToLower(n), "file:") {
		path, hash, ok = parseFileURL(n)
	}
	if !ok {
		path, hash = n, ""
		if i := strings.IndexByte(path, '#'); i >= 0 {
			path, hash = path[:i], path[i:]
		}
		if i := strings.IndexByte(path, '?'); i >= 0 {
			path = path[:i]
		}
	}
	path = stripSlashDrive(decodeOr(strings.TrimSpace(path)))
	hash = decodeOr(strings.TrimSpace(hash))
	if path == "" || hasExternalScheme(path) {
		return filePos{}, false
	}
	parse := func(path string) (filePos, bool) {
		p := splitFilePos(path, hash)
		return p, !elidedSegment.MatchString(p.path) && looksLikeFilePath(p.path, path)
	}
	if p, ok := parse(path); ok || strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) || isWindowsAbsPath(path) {
		return p, ok
	}
	if strings.Contains(path, `\`) {
		return parse(strings.ReplaceAll(path, `\`, "/"))
	}
	return filePos{}, false
}

// workspaceRelative is path inside the workspace root, or "" outside it.
func workspaceRelative(path, root string) (string, bool) {
	if root == "" {
		return "", false
	}
	r := strings.TrimRight(root, "/")
	if strings.TrimRight(path, "/") == r {
		return ".", true
	}
	if !strings.HasPrefix(path, r+"/") {
		return "", false
	}
	rel := path[len(r)+1:]
	depth := 0
	for _, s := range strings.Split(rel, "/") {
		switch {
		case s == "..":
			depth--
		case s != "" && s != ".":
			depth++
		}
		if depth < 0 {
			return "", false
		}
	}
	return rel, true
}

// fileLinkTarget is T3's resolveMarkdownFileLinkTarget: the file a link
// names, a relative one resolved in cwd; ok false when it names none.
func fileLinkTarget(href, cwd string) (filePos, bool) {
	p, ok := parseFileLink(href)
	if !ok {
		return filePos{}, false
	}
	if !isRelativeFilePath(p.String()) {
		// A network path not in the workspace signs Windows in to its host.
		if _, in := workspaceRelative(p.path, cwd); isNetworkPath(p.path) && !in {
			return filePos{}, false
		}
		return p, true
	}
	if cwd == "" {
		return filePos{}, false
	}
	// T3's resolvePathLinkTarget: "~/" from the home cwd is in, else cwd,
	// joined in their own style.
	winStyle := func(v string) bool { return isWindowsAbsPath(v) || windowsSep.MatchString(v) }
	join := func(base, next string) string {
		if winStyle(base) {
			return strings.TrimRight(base, `/\`) + `\` + strings.ReplaceAll(next, "/", `\`)
		}
		return strings.TrimRight(base, `/\`) + "/" + strings.TrimLeft(next, "/")
	}
	switch {
	case strings.HasPrefix(p.path, "~/") || winStyle(cwd) && strings.HasPrefix(p.path, `~\`):
		if home := homeIn.FindString(cwd); home != "" {
			p.path = join(home, p.path[2:])
		}
	case !isAbsPath(p.path):
		p.path = join(cwd, p.path)
	}
	return p, true
}

var (
	homeIn     = regexp.MustCompile(`^(?:/Users/[^/]+|/home/[^/]+|(?i:[A-Z]:[\\/]Users[\\/][^\\/]+))`)
	windowsSep = regexp.MustCompile(`[A-Za-z]:\\`)
)

// fileLabelSuffixes is T3's buildFileLinkParentSuffixByPath: for files of
// one message that share a name, the fewest parent folders, at least two,
// that tell them apart.
func fileLabelSuffixes(paths []string) map[string]string {
	groups := map[string][]string{}
	seen := map[string]bool{}
	for _, p := range paths {
		n := strings.ReplaceAll(p, `\`, "/")
		segs := strings.FieldsFunc(n, func(r rune) bool { return r == '/' })
		if len(segs) == 0 || seen[n] {
			continue
		}
		seen[n] = true
		groups[segs[len(segs)-1]] = append(groups[segs[len(segs)-1]], n)
	}
	out := map[string]string{}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		parents := map[string][]string{}
		counts := map[string]int{}
		for _, p := range g {
			segs := strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
			parents[p] = segs[:len(segs)-1]
			for d := 1; d <= len(parents[p]); d++ {
				counts[strings.Join(parents[p][len(parents[p])-d:], "/")]++
			}
		}
		for _, p := range g {
			segs := parents[p]
			if len(segs) == 0 {
				continue
			}
			depth := len(segs)
			for d := 1; d <= len(segs); d++ {
				if counts[strings.Join(segs[len(segs)-d:], "/")] == 1 {
					depth = d
					break
				}
			}
			depth = min(len(segs), max(depth, 2))
			out[p] = strings.Join(segs[len(segs)-depth:], "/")
		}
	}
	return out
}

// fileLinkLabel is T3's fileLinkLabel: "send.go · upload/v2 · L12–20".
func fileLinkLabel(p filePos, suffixes map[string]string) string {
	parts := []string{fileBasename(p.path)}
	if s := suffixes[strings.ReplaceAll(p.path, `\`, "/")]; s != "" {
		parts = append(parts, s)
	}
	switch {
	case p.line > 0 && p.endLine > 0:
		parts = append(parts, "L"+strconv.Itoa(p.line)+"–"+strconv.Itoa(p.endLine))
	case p.line > 0 && p.column > 0:
		parts = append(parts, "L"+strconv.Itoa(p.line)+":C"+strconv.Itoa(p.column))
	case p.line > 0:
		parts = append(parts, "L"+strconv.Itoa(p.line))
	}
	return strings.Join(parts, " · ")
}

// isFileLabel is T3's isMarkdownFileLinkLabel: a link's words that only
// repeat its file, so the chip stands alone.
func isFileLabel(label, href string) bool {
	d, ok := parseFileLink(href)
	if !ok {
		return false
	}
	norm := func(p string) string {
		return strings.TrimRight(strings.TrimPrefix(strings.ReplaceAll(p, `\`, "/"), "./"), "/")
	}
	l := splitFilePos(strings.TrimSpace(label), "")
	if l.line != 0 && l.line != d.line || l.column != 0 && l.column != d.column || l.endLine != 0 && l.endLine != d.endLine {
		return false
	}
	lp, dp := norm(l.path), norm(d.path)
	if lp == "" {
		return true
	}
	if isWindowsAbsPath(d.path) {
		lp, dp = strings.ToLower(lp), strings.ToLower(dp)
	}
	return dp == lp || strings.HasSuffix(dp, "/"+lp)
}
