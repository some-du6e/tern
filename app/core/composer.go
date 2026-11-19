package core

import (
	"errors"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The composer's behaviour, ported from T3's (docs/ui/07-composer.md §5):
// the placeholder, the trigger's model and effort, the permissions chip,
// the @, / and $ menus, the prompt history and the questions, one at a
// time.

// composerState is the composer's menu, history and questions.
type composerState struct {
	trigger   *trigger       // the @, / or $ word at the caret, while its menu is open
	dismissed *trigger       // the trigger Escape closed: one of its kind at its place stays shut
	items     []menuItem     // the menu's rows
	sel       int            // its highlighted row
	searching string         // the path search on its way, by query
	found     pathSearch     // the last search's results
	seen      [2]string      // the prompt and caret the menu was worked out for
	pos       int            // one past the recalled prompt's place in the thread's prompts; 0 while none is
	request   string         // the request the questions below are for
	question  int            // the question on show
	custom    map[int]string // what was typed for each question
	// responding is the requests whose answers are on their way.
	responding map[string]bool
	// fresh is the threads whose next message starts a fresh session (/clear).
	fresh     map[string]bool
	sending   bool         // a send is on its way
	scratch   string       // the scratch project a draft moved to, until the shell has it
	stash     []stashEntry // the prompts put aside, newest first
	stashOpen bool         // their list is open
	stashSel  int          // its highlighted row
	stashed   time.Time    // when a prompt was last put aside, for the tab's pulse
	// editing is the queued run whose message is in the composer, and
	// editSaved the draft it took the place of.
	editing, editSaved string
}

// pathSearch is what projects.searchEntries found for a query in a
// workspace.
type pathSearch struct {
	cwd, query string
	entries    []t3.Entry
}

// trigger is a word that opens a menu: its kind, what follows the sign,
// and its runes in the prompt.
type trigger struct {
	kind       string // "slash", "skill" or "path"
	query      string
	start, end int
}

func isSpace(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '\r' }

var currencyAmount = regexp.MustCompile(`^\.?[0-9][0-9,._]*(?:[kKmMbBtT]|[eE][0-9]+)?[.,;:!?)]*$`)

// detectTrigger is detectComposerTrigger (composer-logic.ts): the word
// before the caret, when it opens a menu. / at its start opens the
// commands, unless the word is a path with another slash in it; a
// currency sign not followed by an amount opens the skills; @ the files.
// A word against a picture's chip opens nothing, and tern has no pull
// request search, so # is a plain word.
func detectTrigger(text []rune, cursor int) *trigger {
	cursor = max(0, min(cursor, len(text)))
	start := cursor
	for start > 0 && !isSpace(text[start-1]) {
		start--
	}
	word := text[start:cursor]
	if len(word) == 0 {
		return nil
	}
	for _, r := range word {
		if ui.ChipN(r) != 0 || r == ' ' {
			return nil
		}
	}
	switch {
	case word[0] == '/':
		end := cursor
		for end < len(text) && !isSpace(text[end]) {
			end++
		}
		if strings.ContainsRune(string(text[start+1:end]), '/') {
			return nil
		}
		return &trigger{"slash", string(word[1:]), start, cursor}
	case unicode.Is(unicode.Sc, word[0]):
		q := string(word[1:])
		if currencyAmount.MatchString(q) {
			return nil
		}
		return &trigger{"skill", q, start, cursor}
	case word[0] == '@':
		return &trigger{"path", string(word[1:]), start, cursor}
	}
	return nil
}

// menuItem is a row of the composer's menu and what picking it does.
type menuItem struct {
	ui.Suggestion
	insert string // typed in the trigger's place
	model  bool   // /model: it opens the model menu instead
	clear  bool   // /clear: it arms a fresh session instead
	score  int
	tie    string
}

// scoreQuery is scoreQueryMatch (shared/searchRanking.ts) for a value
// and query already trimmed and lowercased: exact, prefix, a match after a
// boundary, anywhere, then as a subsequence, each base plus how far in and
// how much longer the value is. A base below 0 skips that kind of match.
func scoreQuery(value, query string, exact, prefix, boundary, includes, fuzzy int, markers string) (int, bool) {
	if value == "" || query == "" {
		return 0, false
	}
	penalty := min(64, max(0, len(value)-len(query)))
	if value == query {
		return exact, true
	}
	if prefix >= 0 && strings.HasPrefix(value, query) {
		return prefix + penalty, true
	}
	if boundary >= 0 {
		best := -1
		for _, m := range markers {
			if i := strings.Index(value, string(m)+query); i >= 0 && (best < 0 || i+1 < best) {
				best = i + 1
			}
		}
		if best >= 0 {
			return boundary + best*2 + penalty, true
		}
	}
	if includes >= 0 {
		if i := strings.Index(value, query); i >= 0 {
			return includes + i*2 + penalty, true
		}
	}
	if fuzzy >= 0 {
		if s, ok := subsequence(value, query); ok {
			return fuzzy + s, true
		}
	}
	return 0, false
}

// subsequence is scoreSubsequenceMatch: query's letters in order in
// value, scored by where they start, the gaps and the span.
func subsequence(value, query string) (int, bool) {
	qi, first, prev, gaps := 0, -1, -1, 0
	for vi := 0; vi < len(value); vi++ {
		if value[vi] != query[qi] {
			continue
		}
		if first < 0 {
			first = vi
		}
		if prev >= 0 {
			gaps += vi - prev - 1
		}
		prev = vi
		qi++
		if qi == len(query) {
			span := vi - first + 1 - len(query)
			return first*2 + gaps*3 + span + min(64, len(value)-len(query)), true
		}
	}
	return 0, false
}

// best is the lowest of scores that matched.
func best(scores ...[2]int) (int, bool) {
	out, ok := 0, false
	for _, s := range scores {
		if s[1] == 1 && (!ok || s[0] < out) {
			out, ok = s[0], true
		}
	}
	return out, ok
}

func scored(s int, ok bool) [2]int {
	if ok {
		return [2]int{s, 1}
	}
	return [2]int{}
}

// skillLabel is formatProviderSkillDisplayName.
func skillLabel(s t3.Skill) string {
	if d := strings.TrimSpace(s.DisplayName); d != "" {
		return d
	}
	return s.Name
}

// skillSource is resolveProviderSkillSourceKind's label and icon for the
// badge (ComposerCommandMenu.tsx).
func skillSource(s t3.Skill) (label, icon string) {
	p := strings.ReplaceAll(s.Path, "\\", "/")
	if strings.Contains(p, "/.codex/plugins/") || strings.Contains(p, "/.agents/plugins/") {
		return "App", "blocks"
	}
	switch strings.ToLower(strings.TrimSpace(s.Scope)) {
	case "repo", "repository":
		return "Repo", "folder"
	case "project", "workspace", "local":
		return "Project", "folder"
	case "user", "personal":
		return "Personal", "user-round"
	case "system":
		return "System", "settings"
	}
	return "Provider", "package"
}

// scoreSkill is scoreProviderSkill: its name first, then its label,
// short description, description and scope.
func scoreSkill(s t3.Skill, q string) (int, bool) {
	l := strings.ToLower
	return best(
		scored(scoreQuery(l(s.Name), q, 0, 2, 4, 6, 100, "-_/")),
		scored(scoreQuery(l(skillLabel(s)), q, 1, 3, 5, 7, 110, " -_/")),
		scored(scoreQuery(l(s.ShortDescription), q, 20, 22, 24, 26, -1, " -_/")),
		scored(scoreQuery(l(s.Description), q, 30, 32, 34, 36, -1, " -_/")),
		scored(scoreQuery(l(s.Scope), q, 40, 42, -1, 44, -1, "")),
	)
}

// invocable is the provider's skills a user can start, once each by name.
func invocable(skills []t3.Skill) []t3.Skill {
	seen := map[string]bool{}
	var out []t3.Skill
	for _, s := range skills {
		k := strings.ToLower(strings.TrimSpace(s.Name))
		if !s.Enabled || (s.UserInvocable != nil && !*s.UserInvocable) || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

// skillDesc is what a skill's row says about it.
func skillDesc(s t3.Skill, fallback string) string {
	switch {
	case s.ShortDescription != "":
		return s.ShortDescription
	case s.Description != "":
		return s.Description
	case s.Scope != "":
		return s.Scope + " skill"
	}
	return fallback
}

// rank sorts items by score, then by their tie-breakers.
func rank(items []menuItem) []menuItem {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score < items[j].score
		}
		return items[i].tie < items[j].tie
	})
	return items
}

// slashItems is the / menu (useComposerMenuItems.ts, composerSlashCommandSearch.ts):
// /model, /clear where a fresh session can start, then the provider's
// commands (only at the prompt's start, /compact only with nothing before
// it, and not its own /clear where T3's is: that would clear a session T3
// still treats as live) and its skills as /skill:name, ranked against the
// query.
func slashItems(p *t3.Provider, t *trigger, before string, fresh bool) []menuItem {
	items := []menuItem{{Suggestion: ui.Suggestion{Label: "/model", Desc: "Switch response model for this thread"}, model: true, tie: "0\x00model"}}
	if fresh && t.start == 0 {
		items = append(items, menuItem{Suggestion: ui.Suggestion{Label: "/clear", Desc: "Start a fresh session; earlier messages stay visible"}, clear: true, tie: "0\x01clear"})
	}
	var skills []t3.Skill
	if p != nil {
		skills = invocable(p.Skills)
		names := map[string]bool{}
		for _, s := range skills {
			names[strings.ToLower(strings.TrimSpace(s.Name))] = true
		}
		for _, c := range p.SlashCommands {
			if t.start != 0 || names[strings.ToLower(strings.TrimSpace(c.Name))] || (c.Name == "compact" && strings.TrimSpace(before) != "") || (c.Name == "clear" && fresh) {
				continue
			}
			desc := c.Description
			if desc == "" && c.Input != nil {
				desc = c.Input.Hint
			}
			if desc == "" {
				desc = "Run provider command"
			}
			items = append(items, menuItem{Suggestion: ui.Suggestion{Label: "/" + c.Name, Desc: desc}, insert: "/" + c.Name + " ", tie: "1\x00" + c.Name})
		}
	}
	q := strings.ToLower(strings.TrimLeft(strings.TrimSpace(t.query), "/"))
	var out []menuItem
	for _, it := range items {
		if q == "" {
			out = append(out, it)
			continue
		}
		name := strings.ToLower(strings.TrimPrefix(it.Label, "/"))
		s, ok := best(
			scored(scoreQuery(name, q, 0, 2, 4, 6, 100, "-_/")),
			scored(scoreQuery(strings.ToLower(it.Desc), q, 20, 22, 24, 26, -1, " -_/")),
		)
		if ok {
			it.score = s
			out = append(out, it)
		}
	}
	for _, s := range skills {
		badge, icon := skillSource(s)
		it := menuItem{Suggestion: ui.Suggestion{Prefix: "/skill:", Label: skillLabel(s), Desc: skillDesc(s, ""), Badge: badge, BadgeIcon: icon}, insert: "$" + s.Name + " ", tie: "2\x00" + s.Name}
		if q != "" {
			sq := strings.TrimPrefix(q, "skill:")
			sc, ok := 0, q == "skill"
			if !ok {
				if sq == "" {
					ok = true
				} else if sc, ok = scoreSkill(s, sq); !ok && strings.HasPrefix("skill", q) {
					sc, ok = 1<<30, true
				}
			}
			if !ok {
				continue
			}
			it.score = sc
		}
		out = append(out, it)
	}
	if q == "" {
		return out
	}
	return rank(out)
}

// skillItems is the $ menu (searchProviderSkills): the skills, ranked.
func skillItems(p *t3.Provider, t *trigger) []menuItem {
	if p == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(strings.TrimLeftFunc(t.query, func(r rune) bool { return unicode.Is(unicode.Sc, r) })))
	var out []menuItem
	for _, s := range invocable(p.Skills) {
		badge, icon := skillSource(s)
		it := menuItem{Suggestion: ui.Suggestion{Label: skillLabel(s), Desc: skillDesc(s, "Run provider skill"), Badge: badge + " Skill", BadgeIcon: icon}, insert: "$" + s.Name + " ", tie: s.Name}
		if q != "" {
			sc, ok := scoreSkill(s, q)
			if !ok {
				continue
			}
			it.score = sc
		}
		out = append(out, it)
	}
	if q == "" {
		return out
	}
	return rank(out)
}

// pathItems is the @ menu's rows for a search's entries: the file or
// folder's name, then where it is; picking one types a Markdown link to it.
func pathItems(entries []t3.Entry) []menuItem {
	var out []menuItem
	for _, e := range entries {
		icon := "file"
		if e.Kind == "directory" {
			icon = "folder"
		}
		name, dir := path.Base(e.Path), path.Dir(e.Path)
		if dir == "." {
			dir = ""
		}
		out = append(out, menuItem{Suggestion: ui.Suggestion{Icon: icon, Label: name, Desc: dir}, insert: fileLink(e.Path) + " "})
	}
	return out
}

// fileLink is serializeComposerFileLink (shared/composerTrigger.ts): a
// Markdown link labelled with the path's basename, \ [ and ] escaped, to
// the path as encodeURI writes it with ( ) # ? and \ encoded too.
func fileLink(p string) string {
	base := p[strings.LastIndexAny(p, `/\`)+1:]
	label := strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`).Replace(base)
	return "[" + label + "](" + encodeURI(p) + ")"
}

// encodeURI is JavaScript's encodeURI followed by T3's own escapes: every
// UTF-8 byte but letters, digits and ;,/:@&=+$-_.!~*' as %XX.
func encodeURI(s string) string {
	const keep = ";,/:@&=+$-_.!~*'"
	var b strings.Builder
	for _, c := range []byte(s) {
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte(keep, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteString("%" + strings.ToUpper(strconv.FormatInt(int64(c)|0x100, 16)[1:]))
	}
	return b.String()
}

// ---------- the prompt history ----------

var (
	reviewComment = regexp.MustCompile(`<review_comment\b[^>]*>[\s\S]*?</review_comment>`)
	// legacyContext is a context block at the end; RE2 has no backreference,
	// so recallable checks the closing tag names the opening one.
	legacyContext  = regexp.MustCompile(`\n*<(terminal_context|element_context|preview_annotation)>\n([\s\S]*?)\n</(terminal_context|element_context|preview_annotation)>\s*$`)
	legacyHeader   = regexp.MustCompile(`(?m)^- (.+):$`)
	previewOpening = regexp.MustCompile(`(?m)^<preview_annotation>\s*$`)
	previewFields  = regexp.MustCompile(`(?m)^(?:Id|Page|Comment|Targets): `)
	terminalLines  = regexp.MustCompile(`^(.+?) lines? (\d+(?:-\d+)?)$`)
	automationNote = regexp.MustCompile(`^\[Triggered by schedule task: [^\r\n]+\]\r?\n\r?\n`)
	automationID   = regexp.MustCompile(`^scheduled-task-message:(.+):\d+:(?:scheduled|manual)$`)
)

// T3's own prompts, which no one typed: the one that starts a turn for
// attachments alone (composerContextReferences.ts) and a plan's go-ahead
// (proposedPlan.ts).
const (
	bootstrapPrompt = "[User attached one or more files without additional text. Respond using the conversation context and the attached files.]"
	planPrompt      = "PLEASE IMPLEMENT THIS PLAN:\n"
)

// recallable is recallableComposerPrompt (composerPromptHistory.ts): a
// sent message as the composer would bring it back, without the
// Ultrathink prefix, the review comments and legacy context blocks at its
// end or its inline references, whose records are gone; T3's own prompts
// come back as nothing.
func recallable(text string) string {
	p := strings.TrimSpace(text)
	p = strings.TrimPrefix(p, "Ultrathink:\n")
	for p != "" {
		if q := trimReviewComments(p); q != p {
			p = q
			continue
		}
		m := legacyContext.FindStringSubmatchIndex(p)
		if m == nil || p[m[2]:m[3]] != p[m[6]:m[7]] {
			break
		}
		kind, body := p[m[2]:m[3]], p[m[4]:m[5]]
		if kind == "preview_annotation" && previewOpening.MatchString(body) && !previewFields.MatchString(body) {
			break
		}
		var headers []string
		for _, h := range legacyHeader.FindAllStringSubmatch(body, -1) {
			headers = append(headers, h[1])
		}
		if kind != "preview_annotation" && len(headers) == 0 {
			break
		}
		p = strings.TrimRight(p[:m[0]], " \t\r\n")
		if kind == "terminal_context" {
			p = stripTerminalLabels(p, headers)
		}
	}
	refs := t3.ContextRefs(p)
	for i := len(refs) - 1; i >= 0; i-- {
		start, end := refs[i][0], refs[i][1]
		switch {
		case end < len(p) && p[end] == ' ':
			end++
		case start > 0 && p[start-1] == ' ':
			start--
		}
		p = p[:start] + p[end:]
	}
	p = strings.TrimSpace(p)
	if p == bootstrapPrompt || strings.HasPrefix(p, planPrompt) {
		return ""
	}
	return p
}

// trimReviewComments drops the review comment blocks that end p.
func trimReviewComments(p string) string {
	cut := len(p)
	blocks := reviewComment.FindAllStringIndex(p, -1)
	for i := len(blocks) - 1; i >= 0; i-- {
		if strings.TrimSpace(p[blocks[i][1]:cut]) != "" {
			break
		}
		cut = blocks[i][0]
	}
	if cut == len(p) {
		return p
	}
	return strings.TrimRight(p[:cut], " \t\r\n")
}

// stripTerminalLabels drops the inline @terminal:lines labels a legacy
// terminal block's headers name, with a space beside each.
func stripTerminalLabels(p string, headers []string) string {
	word := func(c byte) bool {
		return c == '_' || c == '@' || c == '.' || c == '-' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
	}
	for _, h := range headers {
		m := terminalLines.FindStringSubmatch(h)
		if m == nil {
			continue
		}
		label := "@" + strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(m[1]))), "-") + ":" + m[2]
		for from := 0; from < len(p); {
			i := strings.Index(p[from:], label)
			if i < 0 {
				break
			}
			start, end := from+i, from+i+len(label)
			if (start > 0 && word(p[start-1])) || (end < len(p) && (p[end] == '-' || '0' <= p[end] && p[end] <= '9')) {
				from = start + 1
				continue
			}
			switch {
			case end < len(p) && p[end] == ' ':
				end++
			case start > 0 && p[start-1] == ' ':
				start--
			}
			p = p[:start] + p[end:]
			break
		}
	}
	return p
}

// typedByUser reports whether a user message is the user's own: not an
// automation's, another agent's or a guest's (resolveUserMessagePresentation).
func typedByUser(m t3.Message, text string) bool {
	if m.Context != nil && len(m.Context.Guest) > 0 && string(m.Context.Guest) != "null" {
		return false
	}
	if m.ScheduledTaskID != "" || m.CreatedBy == "agent" {
		return false
	}
	return !(automationNote.MatchString(text) && automationID.MatchString(m.ID))
}

// promptHistory is what the thread's user typed, oldest first, a prompt
// sent twice in a row once (buildComposerPromptHistoryEntries).
func promptHistory(v *threadView) []string {
	var out []string
	for _, it := range v.sorted() {
		if it.Type != "user_message" {
			continue
		}
		m := v.messages[it.MessageID]
		text := it.Text
		if text == "" {
			text = m.Text
		}
		if it.ScheduledTaskID != "" || !typedByUser(m, text) {
			continue
		}
		p := recallable(text)
		if p == "" || (len(out) > 0 && out[len(out)-1] == p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// stepHistory is stepComposerPromptHistory: back from entry (-1 for the
// prompt being typed) to the one before, or forward to the one after,
// which past the newest is an empty prompt again. ok is false when there's
// nowhere to go.
func stepHistory(entries []string, entry int, back bool) (int, string, bool) {
	if back {
		next := entry - 1
		if entry < 0 {
			next = len(entries) - 1
		}
		if next < 0 {
			return entry, "", false
		}
		return next, entries[next], true
	}
	if entry < 0 {
		return entry, "", false
	}
	if entry+1 >= len(entries) {
		return -1, "", true
	}
	return entry + 1, entries[entry+1], true
}

// ---------- the model trigger ----------

// modelFamilies are the name prefixes a driver's own models share
// (providerClients.ts), which their trigger drops.
var modelFamilies = map[string][]string{
	"codex":       {"GPT-", "GPT "},
	"claudeAgent": {"Claude "},
	"antigravity": {"Gemini "},
}

var (
	familyPrefixes = []string{"Claude ", "GPT-", "GPT ", "Gemini ", "Muse ", "Grok "}
	variantWords   = map[string]bool{"pro": true, "flash": true, "mini": true, "max": true, "lite": true, "nano": true, "ultra": true,
		"plus": true, "fast": true, "turbo": true, "preview": true, "latest": true, "thinking": true, "reasoning": true, "build": true,
		"code": true, "image": true, "vision": true, "audio": true, "live": true, "custom": true, "tools": true, "exp": true,
		"experimental": true, "chat": true, "tts": true, "stt": true, "non": true, "transcribe": true}
	versionWord   = regexp.MustCompile(`(?i)^v?\d+(?:\.\d+)*[a-z]?$`)
	vendorPrefix  = regexp.MustCompile(`^[A-Z][\w.&\- ]{0,24}: \S`)
	parenthesised = regexp.MustCompile(`\(.*?\)`)
	wordBreaks    = regexp.MustCompile(`[\s-]+`)
)

// nonVersionWords is name's words that aren't version numbers.
func nonVersionWords(s string) []string {
	var out []string
	for _, w := range wordBreaks.Split(parenthesised.ReplaceAllString(s, " "), -1) {
		if w != "" && !versionWord.MatchString(w) {
			out = append(out, w)
		}
	}
	return out
}

// namesModel reports whether s has a word that names a model, not just a
// variant of one.
func namesModel(s string) bool {
	for _, w := range nonVersionWords(s) {
		if !variantWords[strings.ToLower(w)] {
			return true
		}
	}
	return false
}

// modelShortName is the trigger's name for a model (modelShortName.ts,
// providerIconUtils.ts): its own short name, or its name without a vendor
// prefix and, on its family's own provider, without the family: Claude
// Opus 5.5 is Opus 5.5 on Claude, GPT-5.5 stays GPT-5.5 on Codex.
func modelShortName(m t3.Model, driver string) string {
	if m.ShortName != "" {
		return m.ShortName
	}
	name := m.Name
	if loc := vendorPrefix.FindStringIndex(name); loc != nil {
		name = name[strings.Index(name, ": ")+2:]
	}
	for _, f := range familyPrefixes {
		if !strings.HasPrefix(name, f) {
			continue
		}
		rest := strings.TrimSpace(name[len(f):])
		native := false
		for _, nf := range modelFamilies[driver] {
			native = native || nf == f
		}
		keep := namesModel(rest)
		if native {
			keep = len(nonVersionWords(rest)) > 0
		}
		if keep {
			if strings.ContainsAny(rest, " \t") {
				return rest
			}
			return strings.ReplaceAll(rest, "-", " ")
		}
		break
	}
	if namesModel(name) {
		return name
	}
	return m.Name
}

// reasoningIDs are the option ids T3 reads as the reasoning level.
var reasoningIDs = map[string]bool{"reasoningEffort": true, "effort": true, "reasoning": true, "variant": true,
	"reasoning_effort": true, "thinking_level": true, "thinking": true}

// effortSuffix is what the trigger shows after the model's name
// (describeComposerEffort, buildTraitsTriggerDisplay): the reasoning
// level's label, the speed its bolt shows ("Fast" or "Ultrafast"),
// whether Ultracode is on, for its workflow icon, and the context window's
// label.
func effortSuffix(sel t3.ModelSelection, m *t3.Model) (word, speed string, ultracode bool, window string) {
	label := func(d t3.OptionDescriptor) string {
		v, _ := optionValue(sel, d).(string)
		for _, o := range d.Options {
			if o.ID == v {
				return o.Label
			}
		}
		return ""
	}
	for _, d := range m.Capabilities.OptionDescriptors {
		switch {
		case d.Type == "boolean" && d.ID == "fastMode":
			if on, _ := optionValue(sel, d).(bool); on {
				speed = "Fast"
			}
		case d.Type == "boolean" && d.ID == "ultracode":
			on, _ := optionValue(sel, d).(bool)
			ultracode = ultracode || on
		case d.Type == "select" && d.ID == "serviceTier":
			if l := label(d); l == "Fast" || l == "Ultrafast" {
				speed = l
			}
		case d.Type == "select" && d.ID == "contextWindow":
			window = label(d)
		case word == "" && reasoningIDs[d.ID] && d.Type == "select":
			word = label(d)
		case word == "" && reasoningIDs[d.ID] && d.Type == "boolean":
			on, _ := optionValue(sel, d).(bool)
			word = d.Label + map[bool]string{true: " On", false: " Off"}[on]
		}
	}
	return word, speed, ultracode, window
}

// ---------- the placeholder ----------

// placeholder is T3's, first that applies (ChatComposer.tsx, §5.2): an
// approval waiting turns a send into its decline, or its feedback where
// it takes some; a question asks for an answer; without a provider
// there's nothing to send to; else the provider's guess at the next
// prompt, when there's one on offer (the composer adds its Tab key).
func placeholder(approval *approval, q *t3.Question, noProvider bool, suggestion string) string {
	switch {
	case approval != nil && approval.feedback:
		return "Send feedback to decline this request"
	case approval != nil:
		return "Send a message to decline and redirect the agent"
	case q != nil && q.AllowCustomAnswer != nil && !*q.AllowCustomAnswer:
		return "Choose an option above"
	case q != nil && len(q.Options) == 0:
		return "Type your answer"
	case q != nil:
		return "Type your own answer, or leave this blank to use the selected option"
	case noProvider:
		return "Enable a provider in Settings to send a message"
	case suggestion != "":
		return suggestion
	}
	return "Ask anything. @ files, $ skills, / commands"
}

// ---------- the menu, live ----------

// composerChanged follows the prompt: typing ends a walk through the
// history, and the word at the caret opens, updates or closes the menu.
func (a *App) composerChanged(th *ui.Theme, text string) {
	caret := th.EditorCaret("composer")
	a.mu.Lock()
	defer a.mu.Unlock()
	c := &a.comp
	// A keystroke reports both its change and its caret: work it out once.
	seen := [2]string{text, strconv.Itoa(caret)}
	if seen == c.seen {
		return
	}
	c.seen = seen
	a.syncImages(text)
	runes := []rune(text)
	if c.pos > 0 {
		if h := a.history(); c.pos > len(h) || h[c.pos-1] != text {
			c.pos = 0
		}
	}
	t := detectTrigger(runes, caret)
	// Escape shuts a trigger's kind at its place until the caret leaves it
	// (useComposerTriggerState.ts).
	if d := c.dismissed; t != nil && d != nil && t.kind == d.kind && t.start == d.start {
		t = nil
	} else {
		c.dismissed = nil
	}
	if t == nil || c.trigger == nil || t.kind != c.trigger.kind || t.query != c.trigger.query {
		c.sel = 0
	}
	c.trigger = t
	if t == nil {
		c.items, c.found = nil, pathSearch{}
		return
	}
	p := a.provider(a.choiceFor(a.composerKey()).sel.InstanceID)
	switch t.kind {
	case "slash":
		c.items = slashItems(p, t, string(runes[:t.start]), a.freshCommand())
	case "skill":
		c.items = skillItems(p, t)
	case "path":
		c.items = nil
		switch q := pathQuery(t); {
		case q == "":
		case c.found.query == q && c.found.cwd == a.workspace():
			c.items = pathItems(c.found.entries)
		case c.searching != q:
			a.searchPaths(q)
		}
	}
	c.sel = min(c.sel, max(0, len(c.items)-1))
}

// freshSession is whether the open thread's next message can start a
// fresh session (ChatView.tsx freshSessionAvailable): it has messages and
// its server takes freshContext. Call with the lock held.
func (a *App) freshSession() bool {
	cfg := a.cfg()
	return a.draft == "" && a.view != nil && len(a.view.messages) > 0 && cfg != nil && cfg.Environment.Capabilities.FreshContext
}

// freshCommand is whether /clear is on offer: a fresh session can start,
// and no approval or question has the composer, whose text answers it.
// Call with the lock held.
func (a *App) freshCommand() bool {
	return a.freshSession() && len(a.shownApprovals()) == 0 && len(a.questionsLocked()) == 0
}

// isClear is isClearComposerCommand: the prompt is /clear alone.
var isClear = regexp.MustCompile(`(?i)^/clear\s*$`)

// pathQuery is what an @ word searches for: trimmed, and 256 characters at
// most, which is all the server takes (contracts project.ts).
func pathQuery(t *trigger) string {
	q := strings.TrimSpace(t.query)
	if r := []rune(q); len(r) > 256 {
		q = string(r[:256])
	}
	return q
}

// menuEmpty is the menu's line while it has no rows: still looking, or
// that nothing matched (ComposerCommandMenu.tsx).
func (a *App) menuEmpty() string {
	t := a.comp.trigger
	switch {
	case t == nil || len(a.comp.items) > 0:
		return ""
	case t.kind == "path" && strings.TrimSpace(t.query) == "":
		return ""
	case t.kind == "path" && a.comp.searching != "":
		return "Searching workspace files..."
	case t.kind == "path":
		return "No matching files or folders."
	case t.kind == "skill":
		return "No skills found. Try / to browse provider commands."
	}
	return "No matching command."
}

// workspace is where the composer's thread works: its worktree, else its
// project's folder.
func (a *App) workspace() string {
	if a.draft != "" {
		return a.draftProject().WorkspaceRoot
	}
	e, t := a.thread(a.openID)
	if t == nil {
		return ""
	}
	if t.WorktreePath != "" {
		return t.WorktreePath
	}
	return e.projects[t.ProjectID].WorkspaceRoot
}

// searchPaths asks the environment for the paths matching q once the word
// has settled for T3's 120ms (queries.ts: 80 at most), and fills the menu
// if the word's still the same when they come. Call with the lock held.
func (a *App) searchPaths(q string) {
	cwd := a.workspace()
	e := a.cur()
	if cwd == "" || e == nil || e.conn == nil {
		return
	}
	a.comp.searching = q
	conn := e.conn
	go func() {
		time.Sleep(120 * time.Millisecond)
		a.mu.Lock()
		still := a.comp.searching == q
		a.mu.Unlock()
		if !still {
			return
		}
		var res struct {
			Entries []t3.Entry `json:"entries"`
		}
		err := conn.Call("projects.searchEntries", map[string]any{"cwd": cwd, "query": q, "limit": 80}, &res)
		a.mu.Lock()
		if a.comp.searching == q {
			a.comp.searching = ""
		}
		if t := a.comp.trigger; err == nil && t != nil && t.kind == "path" && pathQuery(t) == q && a.workspace() == cwd {
			a.comp.found = pathSearch{cwd: cwd, query: q, entries: res.Entries}
			a.comp.items = pathItems(res.Entries)
			a.comp.sel = 0
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// menuKey is the menu's keys: the arrows move the highlight around,
// Enter and Tab pick it, Escape closes the menu while the caret stays in
// that word.
func (a *App) menuKey(th *ui.Theme, name string) {
	a.mu.Lock()
	c := &a.comp
	n := len(c.items)
	switch name {
	case "↑", "↓":
		if n > 0 {
			step := 1
			if name == "↑" {
				step = n - 1
			}
			c.sel = (c.sel + step) % n
		}
		a.mu.Unlock()
	case "⎋":
		c.dismissed, c.trigger, c.items, c.found = c.trigger, nil, nil, pathSearch{}
		a.mu.Unlock()
	default:
		sel := c.sel
		a.mu.Unlock()
		a.pickItem(th, sel)
	}
}

// pickItem puts row i in the trigger's place, taking one space after it
// so none doubles; /model opens the model menu instead, and /clear arms a
// fresh session.
func (a *App) pickItem(th *ui.Theme, i int) {
	a.mu.Lock()
	c := &a.comp
	if c.trigger == nil || i < 0 || i >= len(c.items) {
		a.mu.Unlock()
		return
	}
	it, t := c.items[i], *c.trigger
	c.trigger, c.items = nil, nil
	a.mu.Unlock()
	text := []rune(th.EditorText("composer"))
	end := t.end
	if strings.HasSuffix(it.insert, " ") && end < len(text) && text[end] == ' ' {
		end++
	}
	th.ReplaceText("composer", t.start, end, it.insert)
	switch {
	case it.model:
		a.handle(th, ui.Action{ID: "model", Kind: "click"})
	case it.clear:
		a.mu.Lock()
		a.comp.fresh[a.composerKey()] = true
		a.mu.Unlock()
	}
}

// history is the open thread's prompts. Call with the lock held.
func (a *App) history() []string {
	if a.draft != "" || a.view == nil {
		return nil
	}
	return promptHistory(a.view)
}

// recall walks the prompt history back or forward.
func (a *App) recall(th *ui.Theme, back bool) {
	a.mu.Lock()
	entry, p, ok := stepHistory(a.history(), a.comp.pos-1, back)
	if ok {
		a.comp.pos = entry + 1
	}
	a.mu.Unlock()
	if ok {
		th.RecallPrompt(p)
	}
}

// dispatchMode is resolveComposerDispatchMode with T3's default follow-up
// (steer): a send starts a run, or while one works steers it, or with the
// alternate key queues behind it.
func dispatchMode(running, alternate bool) string {
	switch {
	case !running:
		return "auto"
	case alternate:
		return "queue"
	}
	return "steer"
}

// attach is the attach button: the images picked put on the message the
// way pasted ones are.
func (a *App) attach() {
	if a.ChooseImages == nil {
		return
	}
	a.mu.Lock()
	key := a.imageKey()
	a.mu.Unlock()
	go func() {
		imgs, err := a.ChooseImages()
		if len(imgs) == 0 && err == nil {
			return
		}
		a.pasted(key, imgs, err)
	}()
}

// startScratch is "or start without a project": the draft moves to the
// environment's scratch project, which the server makes the first time
// (projects.ensureScratch).
func (a *App) startScratch(th *ui.Theme) {
	a.mu.Lock()
	e := a.env(a.draftEnv)
	a.mu.Unlock()
	if e == nil || e.conn == nil {
		return
	}
	conn := e.conn
	go func() {
		var res struct {
			ProjectID string `json:"projectId"`
		}
		err := conn.Call("projects.ensureScratch", map[string]any{}, &res)
		a.mu.Lock()
		if err != nil {
			a.fail(err)
		} else if a.draftEnv == e.id() && a.draft != "" {
			a.draft, a.comp.scratch = res.ProjectID, res.ProjectID
			delete(a.choices, "draft")
		}
		a.mu.Unlock()
		a.changed()
		a.onUI(func(th *ui.Theme) { th.Focus("composer") })
	}()
}

// ---------- questions ----------

// questionsLocked is the open request's questions. Call with the lock held.
func (a *App) questionsLocked() []t3.Question {
	_, qs := a.questions()
	return qs
}

// syncQuestions starts the question flow afresh for a new request. Call
// with the lock held.
func (a *App) syncQuestions(req string) {
	if a.comp.request != req {
		a.comp.request, a.comp.question, a.comp.custom = req, 0, map[int]string{}
	}
}

// moveQuestion leaves the question on show for the one step after it
// (or before), keeping what was typed for it and bringing back what was
// typed for the next.
func (a *App) moveQuestion(th *ui.Theme, step int) {
	a.mu.Lock()
	req, qs := a.questions()
	a.syncQuestions(req)
	next := a.comp.question + step
	if next < 0 || next >= len(qs) {
		a.mu.Unlock()
		return
	}
	a.comp.custom[a.comp.question] = th.EditorText("composer")
	a.comp.question = next
	typed := a.comp.custom[next]
	a.mu.Unlock()
	th.SetEditorText("composer", typed)
}

// answerAll is Submit (and Enter): once the question on show has its
// answer, on to the next; at the last, the answers, each what was typed
// for it or else its picked options' values (buildPendingUserInputAnswers),
// with each question's pictures (attachmentsByQuestionId). The options
// and the prompt are off until the server has them.
func (a *App) answerAll(th *ui.Theme) {
	a.mu.Lock()
	req, qs := a.questions()
	a.syncQuestions(req)
	if req == "" || len(qs) == 0 || a.answering(req) {
		a.mu.Unlock()
		return
	}
	a.comp.custom[a.comp.question] = th.EditorText("composer")
	answers := map[string]any{}
	for qi, q := range qs {
		if v, ok := a.answerFor(req, qi, q); ok {
			answers[q.ID] = v
		} else if qi <= a.comp.question {
			a.mu.Unlock()
			return
		}
	}
	if a.comp.question < len(qs)-1 {
		a.mu.Unlock()
		a.moveQuestion(th, 1)
		return
	}
	pics := map[string][]pastedImage{}
	for _, q := range qs {
		if imgs := a.images[a.questionKey(req, q.ID)]; len(imgs) > 0 {
			pics[q.ID] = imgs
		}
	}
	var conn *t3.Conn
	if e := a.cur(); e != nil {
		conn = e.conn
	}
	_, thread := splitRef(a.openID)
	typed := a.comp.custom[a.comp.question]
	a.comp.responding[req] = true
	a.guard.release()
	a.mu.Unlock()
	th.SetEditorText("composer", "")
	a.changed()
	go func() {
		err := respondAnswers(conn, thread, req, answers, pics)
		a.mu.Lock()
		delete(a.comp.responding, req)
		if err != nil {
			a.fail(err)
		} else {
			for _, q := range qs {
				delete(a.images, a.questionKey(req, q.ID))
			}
		}
		a.mu.Unlock()
		a.changed()
		if err != nil {
			// Put the last answer back where it was typed.
			a.onUI(func(th *ui.Theme) {
				a.mu.Lock()
				now, _ := a.questions()
				a.mu.Unlock()
				if now == req && th.EditorText("composer") == "" {
					th.SetEditorText("composer", typed)
				}
			})
		}
	}()
}

// respondAnswers uploads each question's pictures, puts their tokens where
// the answer typed for it has their chips, and sends the answers.
func respondAnswers(conn *t3.Conn, thread, req string, answers map[string]any, pics map[string][]pastedImage) error {
	if conn == nil || thread == "" {
		return errors.New("not connected")
	}
	cmd := map[string]any{"type": "runtime-request.respond", "threadId": thread, "requestId": req, "answers": answers}
	byQuestion := map[string]any{}
	for id, imgs := range pics {
		attachments, recs, err := upload(conn, imgs)
		if err != nil {
			return err
		}
		byQuestion[id] = attachments
		if s, ok := answers[id].(string); ok {
			answers[id], _ = withTokens(s, imgs, recs)
		}
	}
	if len(byQuestion) > 0 {
		cmd["attachmentsByQuestionId"] = byQuestion
	}
	return conn.Dispatch(cmd)
}

// answering reports whether req can't take answers now: they're on their
// way, or the provider can no longer take them. Call with the lock held.
func (a *App) answering(req string) bool {
	if a.comp.responding[req] {
		return true
	}
	if a.view != nil {
		if r, ok := a.view.requests[req]; ok {
			return r.ResponseCapability.Type == "not_resumable"
		}
	}
	return false
}

// questionKey is where the pictures for question id of req go
// (questionAttachmentDraftId). Call with the lock held.
func (a *App) questionKey(req, id string) string {
	return a.composerKey() + "\x00" + req + "\x00" + id
}

// imageKey is where the composer's pictures go: its own, or with a
// question on show, that question's. Call with the lock held.
func (a *App) imageKey() string {
	req, qs := a.questions()
	if req == "" || len(qs) == 0 {
		return a.composerKey()
	}
	qi := a.comp.question
	if a.comp.request != req {
		qi = 0
	}
	return a.questionKey(req, qs[min(qi, len(qs)-1)].ID)
}

// answerFor is question qi's answer so far: what was typed for it, where
// it takes typed answers, else its picked options' values (the first, for
// a single choice). Call with the lock held.
func (a *App) answerFor(req string, qi int, q t3.Question) (any, bool) {
	if typed := ui.TrimText(a.comp.custom[qi]); typed != "" && (q.AllowCustomAnswer == nil || *q.AllowCustomAnswer) {
		return typed, true
	}
	var vals []string
	for oi, o := range q.Options {
		if a.answers[req][qi*100+oi] {
			v := o.Value
			if v == "" {
				v = o.Label
			}
			vals = append(vals, v)
		}
	}
	switch {
	case len(vals) == 0:
		return nil, false
	case q.MultiSelect:
		return vals, true
	}
	return vals[0], true
}

// pickOption picks option oi of question qi; a single choice moves on to
// the next question after T3's 200ms.
func (a *App) pickOption(th *ui.Theme, qi, oi int) {
	a.pickAnswer(qi, oi)
	a.mu.Lock()
	req, qs := a.questions()
	a.syncQuestions(req)
	advance := qi == a.comp.question && qi < len(qs) && !qs[qi].MultiSelect && qi < len(qs)-1
	a.mu.Unlock()
	if !advance {
		return
	}
	time.AfterFunc(200*time.Millisecond, func() {
		a.onUI(func(th *ui.Theme) {
			a.mu.Lock()
			now, _ := a.questions()
			same := now == req && a.comp.question == qi
			a.mu.Unlock()
			if same {
				a.moveQuestion(th, 1)
			}
		})
	})
}
