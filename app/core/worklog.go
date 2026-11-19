package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The work log's words, icons and failure checks, ported from T3: its
// work entries (apps/web/src/session-logic.ts:68-490), their presentation
// (packages/client-runtime/src/work-log/presentation.ts), tool activity
// (packages/shared/src/toolActivity.ts) and the row labels of
// MessagesTimeline.logic.ts:97-280 and TimelineWorkEntryRow.logic.ts.
// T3's own MCP tools (t3McpToolPresentation.ts) get no special labels
// yet: they read as the dynamic tools they are.

// workEntry is T3's WorkLogEntry: one work-log entry, made from a turn item.
type workEntry struct {
	id, runID string
	at        time.Time
	label     string
	detail    string
	// command is a command execution's input; hasCommand tells "" from none.
	command     string
	hasCommand  bool
	changed     []string // changedFiles
	tone        string   // thinking, tool, info
	toolTitle   string
	toolName    string // a dynamic tool's
	input       any    // a dynamic tool's input, decoded
	output      any
	viewedImage string
	itemType    string
	// status is T3's toolLifecycleStatus: inProgress, completed, failed,
	// stopped or idle.
	status string
	// source is T3's sourceActivityKind: context-compaction,
	// runtime.warning or runtime.error.
	source  string
	surface string // browser, computer
	item    *t3.TurnItem
	answer  *t3.QuestionAnswer

	checked        bool // the cached checks below are set
	failure        bool
	displayFailure bool
	action         string
}

// itemStatus is T3's projectedWorkEntryStatus.
func itemStatus(s string) string {
	switch s {
	case "pending", "running", "waiting":
		return "inProgress"
	case "cancelled", "interrupted":
		return "stopped"
	}
	return s
}

// itemTone is T3's projectedWorkEntryTone.
func itemTone(typ string) string {
	switch typ {
	case "reasoning":
		return "thinking"
	case "command_execution", "file_change", "file_search", "web_search", "dynamic_tool", "subagent",
		"thread_created", "user_input_request", "approval_request":
		return "tool"
	}
	return "info"
}

// decode is a raw JSON value as Go values, or nil.
func decode(raw json.RawMessage) any {
	var v any
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return v
}

// itemAt is when an item started, or its last update.
func itemAt(it *t3.TurnItem) time.Time {
	if t := parseTime(it.StartedAt); !t.IsZero() {
		return t
	}
	return parseTime(it.UpdatedAt)
}

// newWorkEntry is T3's projectedWorkEntry: a turn item as a work entry.
func newWorkEntry(it *t3.TurnItem) *workEntry {
	title := strings.TrimSpace(it.Title)
	e := &workEntry{id: it.ID, runID: it.RunID, at: itemAt(it), tone: itemTone(it.Type), itemType: it.Type,
		status: itemStatus(it.Status), item: it, surface: it.ToolSurface}
	if p := strings.TrimSpace(it.ViewedImagePath); p != "" && !strings.ContainsAny(p, "\r\n") && isImagePath(p) {
		e.viewedImage = p
	}
	or := func(s, fallback string) string {
		if s != "" {
			return s
		}
		return fallback
	}
	switch it.Type {
	case "thread_created":
		e.label = "Created thread"
	case "compaction":
		e.label, e.source, e.detail = compactionLabel(it), "context-compaction", it.Summary
	case "reasoning":
		e.label, e.detail = or(title, "Thinking"), it.Text
	case "command_execution":
		e.label, e.toolTitle = or(title, "Ran command"), or(title, "Command")
		e.command, e.hasCommand = rawString(it.Input), true
	case "file_change":
		label := "Changed " + it.FileName
		if len(it.Changes) > 1 {
			label = fmt.Sprintf("Changed %d files", len(it.Changes))
		}
		e.label, e.toolTitle = or(title, label), or(title, "File change")
		for _, c := range it.Changes {
			e.changed = append(e.changed, c.Path)
		}
		if it.Changes == nil {
			e.changed = []string{it.FileName}
		}
	case "file_search":
		e.label = or(title, or(searchLabel(map[string]any{"pattern": it.Pattern}), "Searched files"))
		e.detail, e.toolTitle = it.Pattern, or(title, "File search")
	case "web_search":
		e.label, e.detail, e.toolTitle = or(title, "Searched the web"), strings.Join(it.Patterns, ", "), or(title, "Web search")
	case "system_notice":
		e.label, e.source = it.Message, "runtime.warning"
	case "error":
		e.label, e.detail = providerError(it)
		switch {
		case it.Failure != nil && it.Failure.Class == "usage_limit" && it.Status != "completed":
			e.source = "runtime.warning"
		case it.Retry == nil:
			e.source = "runtime.error"
		}
	case "dynamic_tool":
		e.toolName, e.input, e.output = it.ToolName, decode(it.Input), decode(it.Output)
		label := or(it.ToolName, "Tool call")
		switch classifyTool("dynamic_tool_call", "", map[string]any{"toolName": it.ToolName, "input": e.input}) {
		case "read":
			paths := toolFilePaths(map[string]any{"input": e.input})
			label = readLabel(first(paths), 0)
		case "search":
			label = or(searchLabel(map[string]any{"input": e.input}), label)
		}
		e.label, e.toolTitle = or(title, label), or(title, or(it.ToolName, "Tool"))
	case "todo_list":
		done, current := 0, ""
		for _, s := range it.Steps {
			if s.Status == "completed" {
				done++
			}
		}
		for _, want := range []string{"running", "pending"} {
			for _, s := range it.Steps {
				if current == "" && s.Status == want {
					current = or(s.Text, s.Step)
				}
			}
		}
		e.label = fmt.Sprintf("Tasks %d/%d", done, len(it.Steps))
		if current != "" {
			e.label += " · " + current
		}
	case "approval_request":
		e.label, e.detail = or(title, "Approval requested"), or(it.Prompt, it.RequestKind)
	case "user_input_request":
		e.answer = it.QuestionAnswer
		e.label = or(title, "Input requested")
		if e.answer != nil {
			e.label = or(title, "Answered questions")
		}
	case "notification":
		e.label = or(it.Summary, title)
	default:
		e.label = or(title, strings.ReplaceAll(it.Type, "_", " "))
	}
	return e
}

func first(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return xs[0]
}

// providerError is T3's providerErrorPresentation: an error's label and
// detail, with the retry's progress.
func providerError(it *t3.TurnItem) (label, detail string) {
	msg, usage := "", false
	if it.Failure != nil {
		msg, usage = it.Failure.Message, it.Failure.Class == "usage_limit"
	}
	kind := "Provider error"
	if usage {
		kind = "Usage limit reached"
	}
	r := it.Retry
	if r == nil {
		if title := strings.TrimSpace(it.Title); title != "" && !usage {
			kind = title
		}
		return kind, msg
	}
	progress := fmt.Sprint(r.Attempt)
	if r.MaxAttempts != nil {
		progress = fmt.Sprintf("%d/%d", r.Attempt, *r.MaxAttempts)
	}
	switch it.Status {
	case "running":
		label = "Retrying provider (" + progress + ")"
		if d := r.RetryDelayMs; d != nil && *d > 0 {
			if *d < 1000 {
				msg += fmt.Sprintf(" Retrying in %dms.", *d)
			} else {
				msg += " Retrying in " + strings.TrimSuffix(fmt.Sprintf("%.1f", float64(*d)/1000), ".0") + "s."
			}
		}
	case "completed":
		label = "Provider recovered (" + progress + " retries)"
	case "failed":
		label = kind + " after " + progress + " retries"
	default:
		label = "Provider retry stopped (" + progress + ")"
	}
	return label, msg
}

// compactionLabel is T3's contextCompactionLabel.
func compactionLabel(it *t3.TurnItem) string {
	switch it.Status {
	case "failed":
		return "Context compaction failed"
	case "cancelled", "interrupted":
		return "Context compaction stopped"
	case "pending", "running", "waiting":
		return "Compacting context"
	}
	if before, after, ok := compactionTokens(it); ok {
		return "Context compacted " + before + " → " + after + " tokens"
	}
	return "Context compacted"
}

// compactionTokens is T3's contextCompactionTokenCounts.
func compactionTokens(it *t3.TurnItem) (before, after string, ok bool) {
	if it.Status == "running" || it.BeforeTokenCount == nil || it.AfterTokenCount == nil {
		return "", "", false
	}
	return ui.FormatTokens(*it.BeforeTokenCount), ui.FormatTokens(*it.AfterTokenCount), true
}

// ---------- what an entry is ----------

// toolLike is T3's workLogEntryIsToolLike (session-logic.ts).
func (e *workEntry) toolLike() bool {
	return e.tone == "tool" || e.tone == "thinking" || e.tone == "error" || e.hasCommand
}

// lifecycleToolLike is presentation.ts's own workLogEntryIsToolLike, which
// also counts tool lifecycle item types.
func (e *workEntry) lifecycleToolLike() bool {
	if e.tone == "tool" || e.tone == "thinking" || e.tone == "error" || strings.TrimSpace(e.command) != "" {
		return true
	}
	switch e.itemType {
	case "command_execution", "file_change", "web_search":
		return true
	}
	return false
}

// check works out an entry's failure and group action once, for the many
// rules that ask.
func (e *workEntry) check() {
	if e.checked {
		return
	}
	e.checked = true
	e.failure = e.failedFromOutput(true)
	e.displayFailure = e.failedFromOutput(false)
	e.action = e.computeAction()
}

// failed is T3's workEntryIndicatesToolFailure; displayFailed its
// workEntryDisplayIndicatesToolFailure, which leaves the command out.
func (e *workEntry) failed() bool        { e.check(); return e.failure }
func (e *workEntry) displayFailed() bool { e.check(); return e.displayFailure }

func (e *workEntry) failedFromOutput(withCommand bool) bool {
	if e.tone == "error" || e.status == "failed" || e.status == "declined" {
		return true
	}
	// A thought is prose, not output: "exit code 2 is expected" isn't a
	// failed tool.
	if !e.lifecycleToolLike() || e.itemType == "reasoning" {
		return false
	}
	if it := e.item; it != nil && it.Type == "command_execution" {
		// The exit code outranks text in the output or the command.
		if it.ExitCode != nil {
			return *it.ExitCode != 0
		}
		if it.OutputIndicatesFailure {
			return true
		}
		if out := rawString(it.Output); out != "" && outputIndicatesFailure(truncate(out, 32_768)) {
			return true
		}
	}
	out := e.detail
	if withCommand && e.command != "" {
		out = strings.TrimPrefix(out+"\n"+e.command, "\n")
	}
	return out != "" && outputIndicatesFailure(out)
}

// truncate is s cut to at most n bytes, at a character's edge.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// failurePatterns are T3's toolOutputIndicatesFailure (toolOutput.ts:236):
// output from a tool that reported success but describes a failure.
var (
	failureText    = regexp.MustCompile(`(?i)file not found|no files found|enoent|no such file|commandnotfoundexception|command not found|is not recognized as the name of a cmdlet|a parameter cannot be found that matches parameter name`)
	cannotFindPath = regexp.MustCompile(`(?i)cannot find path`)
	doesNotExist   = regexp.MustCompile(`(?i)because it does not exist`)
	notRecognized  = regexp.MustCompile(`(?i)is not recognized`)
	theTerm        = regexp.MustCompile(`(?i)the term '`)
	exitCodes      = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<exited with exit code\s+[1-9]\d*\s*>`),
		regexp.MustCompile(`(?i)exit(?:ed)? with exit code\s+[1-9]\d*`),
		regexp.MustCompile(`(?i)exit code\s*[:\s]\s*[1-9]\d*\b`),
	}
)

func outputIndicatesFailure(s string) bool {
	if failureText.MatchString(s) || cannotFindPath.MatchString(s) && doesNotExist.MatchString(s) ||
		notRecognized.MatchString(s) && theTerm.MatchString(s) {
		return true
	}
	for _, re := range exitCodes {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// succeeded is T3's workEntryIndicatesToolSuccess.
func (e *workEntry) succeeded() bool {
	if !e.toolLike() || e.failed() || (e.tone == "thinking" && e.itemType != "reasoning") {
		return false
	}
	switch e.status {
	case "failed", "declined", "inProgress", "stopped", "idle":
		return false
	}
	return true
}

// neutral is T3's workEntryIndicatesToolNeutralStatus: a tool still going
// or not reported.
func (e *workEntry) neutral() bool { return e.toolLike() && !e.failed() && !e.succeeded() }

// visibleInGroup is T3's workEntryIsVisibleInGroup.
func (e *workEntry) visibleInGroup(expanded bool) bool {
	if e.itemType == "reasoning" {
		return strings.TrimSpace(e.detail) != ""
	}
	return expanded && e.status == "inProgress" || e.status == "stopped" || !e.neutral()
}

// activeTurnActivity is T3's workEntryIsActiveTurnActivity.
func (e *workEntry) activeTurnActivity() bool {
	return e.status == "inProgress" || e.status == "" && e.toolLike()
}

// ---------- group actions and summaries ----------

// groupAction is T3's toolGroupAction: what kind of work an entry is, for
// its group's summary and icon.
func (e *workEntry) groupAction() string { e.check(); return e.action }

func (e *workEntry) computeAction() string {
	switch {
	case e.itemType == "thread_created":
		return "thread-create"
	case e.viewedImage != "":
		return "read"
	case e.itemType == "approval_request" || e.itemType == "user_input_request":
		if e.lifecycleToolLike() {
			return "other"
		}
		return "update"
	}
	itemType := ""
	switch e.itemType {
	case "command_execution", "file_change", "web_search":
		itemType = e.itemType
	case "dynamic_tool":
		itemType = "dynamic_tool_call"
	}
	toolName := e.toolName
	if e.itemType != "dynamic_tool" {
		toolName = e.toolTitle
	}
	title := e.toolTitle
	if title == "" {
		title = e.label
	}
	switch classifyTool(itemType, title, map[string]any{"toolName": toolName}) {
	case "read":
		return "read"
	case "file_change":
		return "edit"
	case "command":
		return "command"
	case "search":
		if e.itemType == "web_search" && !e.localCodeSearch() {
			return "search"
		}
		return "code-search"
	}
	switch {
	case e.itemType == "file_change":
		return "edit"
	case e.itemType == "command_execution" || e.hasCommand:
		return "command"
	case e.localCodeSearch():
		return "code-search"
	case e.itemType == "web_search":
		return "search"
	case len(e.changed) > 0:
		return "edit"
	case e.lifecycleToolLike():
		return "other"
	}
	return "update"
}

var grepWord = regexp.MustCompile(`(?i)\bgrep\b`)

func (e *workEntry) localCodeSearch() bool {
	title := e.toolTitle
	if title == "" {
		title = e.label
	}
	return e.itemType == "file_search" || e.itemType == "web_search" && grepWord.MatchString(compactToolLabel(title))
}

// classifyTool is T3's classifyToolActivity.
func classifyTool(itemType, title string, data map[string]any) string {
	kind := strings.ToLower(strings.TrimSpace(str(data["kind"])))
	name := toolNameToken(str(data["toolName"]))
	switch {
	case itemType == "command_execution":
		return "command"
	case itemType == "image_view":
		return "read"
	case itemType == "file_change":
		return "file_change"
	case itemType == "web_search":
		return "search"
	case kind == "execute":
		return "command"
	case kind == "edit" || kind == "move" || kind == "delete" || kind == "write":
		return "file_change"
	case kind == "search" || name == "find" || name == "grep" || name == "glob" || name == "rg" || name == "ls":
		return "search"
	case kind == "read":
		return "read"
	case name == "terminal" || name == "bash" || name == "shell":
		return "command"
	case name == "read" || name == "readfile":
		return "read"
	}
	return "other"
}

// toolNameToken is a plain tool name folded for matching; MCP names, with
// their __ or dots, have none.
func toolNameToken(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "__") || strings.ContainsAny(name, "./") {
		return ""
	}
	return strings.ToLower(strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name))
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// compactToolLabel is T3's normalizeCompactToolLabel: without a trailing
// "complete" or "completed".
var trailingComplete = regexp.MustCompile(`(?i)\s+(?:complete|completed)\s*$`)

func compactToolLabel(s string) string {
	return strings.TrimSpace(trailingComplete.ReplaceAllString(s, ""))
}

// groupActionCount is how many a summary counts: distinct files for edits.
func groupActionCount(action string, entries []*workEntry) int {
	if action != "edit" {
		return len(entries)
	}
	files := map[string]bool{}
	bare := 0
	for _, e := range entries {
		if len(e.changed) == 0 {
			bare++
		}
		for _, f := range e.changed {
			files[f] = true
		}
	}
	return len(files) + bare
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// groupActionLabel is T3's toolGroupActionLabel.
func groupActionLabel(action string, n int) string {
	switch action {
	case "read":
		return fmt.Sprintf("Read %d %s", n, plural(n, "file", "files"))
	case "edit":
		return fmt.Sprintf("Changed %d %s", n, plural(n, "file", "files"))
	case "command":
		return fmt.Sprintf("Ran %d %s", n, plural(n, "command", "commands"))
	case "thread-create":
		return fmt.Sprintf("Created %d %s", n, plural(n, "thread", "threads"))
	case "device":
		return fmt.Sprintf("Used device controls %d %s", n, plural(n, "time", "times"))
	case "browser":
		return fmt.Sprintf("Used browser %d %s", n, plural(n, "time", "times"))
	case "search":
		return fmt.Sprintf("Searched the web %d %s", n, plural(n, "time", "times"))
	case "code-search":
		return fmt.Sprintf("Searched code %d %s", n, plural(n, "time", "times"))
	case "other":
		return fmt.Sprintf("Used %d %s", n, plural(n, "tool", "tools"))
	}
	return fmt.Sprintf("Received %d %s", n, plural(n, "update", "updates"))
}

// actionPriority ranks a summary's parts: changes first, then reads and
// searches, then the rest.
func actionPriority(action string) int {
	switch action {
	case "command", "edit", "thread-create":
		return 0
	case "other", "update":
		return 2
	}
	return 1
}

// summarizeGroup is T3's summarizeToolGroup: "Ran 3 commands and read 2
// files", the two biggest kinds of work in their order, the rest counted.
func summarizeGroup(entries []*workEntry) (summary string, failed bool) {
	var tools []*workEntry
	for _, e := range entries {
		if e.itemType != "reasoning" {
			tools = append(tools, e)
		}
	}
	if len(entries) > 0 && len(tools) == 0 {
		if len(entries) == 1 {
			return "Thought", false
		}
		return fmt.Sprintf("Thought (×%d)", len(entries)), false
	}
	type group struct {
		action  string
		entries []*workEntry
	}
	var groups []*group
	byAction := map[string]*group{}
	for _, e := range tools {
		a := e.groupAction()
		g := byAction[a]
		if g == nil {
			g = &group{action: a}
			byAction[a] = g
			groups = append(groups, g)
		}
		g.entries = append(g.entries, e)
	}
	type part struct {
		index, count, priority int
		label                  string
	}
	var parts []part
	for i, g := range groups {
		parts = append(parts, part{i, len(g.entries), actionPriority(g.action), groupActionLabel(g.action, groupActionCount(g.action, g.entries))})
		for _, e := range g.entries {
			failed = failed || e.displayFailed()
		}
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].priority < parts[j].priority })
	if len(parts) > 2 {
		parts = parts[:2]
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].index < parts[j].index })
	var labels []string
	counted := 0
	for _, p := range parts {
		labels = append(labels, p.label)
		counted += p.count
	}
	if rest := len(tools) - counted; rest > 0 {
		labels = append(labels, fmt.Sprintf("Performed %d other %s", rest, plural(rest, "action", "actions")))
	}
	for i := 1; i < len(labels); i++ {
		r, n := utf8.DecodeRuneInString(labels[i])
		labels[i] = string(unicode.ToLower(r)) + labels[i][n:]
	}
	if len(labels) < 3 {
		return strings.Join(labels, " and "), failed
	}
	return strings.Join(labels[:len(labels)-1], ", ") + ", and " + labels[len(labels)-1], failed
}

// groupSummaryKind is T3's toolGroupSummaryKind.
func groupSummaryKind(entries []*workEntry) string {
	var tools []*workEntry
	for _, e := range entries {
		if e.itemType != "reasoning" {
			tools = append(tools, e)
		}
	}
	if len(entries) > 0 && len(tools) == 0 {
		return "reasoning"
	}
	actions := map[string]bool{}
	for _, e := range tools {
		actions[e.groupAction()] = true
	}
	if len(actions) != 1 {
		return "mixed"
	}
	var action string
	for a := range actions {
		action = a
	}
	if action != "other" {
		return action
	}
	kinds := map[string]bool{}
	for _, e := range tools {
		switch {
		case e.itemType == "dynamic_tool":
			kinds["dynamic-tool"] = true
		case e.itemType == "subagent" || e.tone == "thinking":
			kinds["agent-tool"] = true
		case e.tone == "tool":
			kinds["tone-tool"] = true
		default:
			kinds["other"] = true
		}
	}
	if len(kinds) == 1 {
		for k := range kinds {
			return k
		}
	}
	return "mixed"
}

// summaryIcon is T3's toolGroupSummaryIconName.
func summaryIcon(kind string) string {
	switch kind {
	case "pull-request", "link-pr", "unlink-pr", "list-prs", "watch-pr", "unwatch-pr":
		return "pull-request"
	case "read":
		return "eye"
	case "edit":
		return "square-pen"
	case "command":
		return "terminal"
	case "thread-create":
		return "t3-code"
	case "browser":
		return "browser"
	case "device":
		return "smartphone"
	case "search":
		return "globe"
	case "code-search":
		return "search"
	case "other":
		return "wrench"
	case "reasoning":
		return "brain"
	case "agent-tool":
		return "bot"
	case "tone-tool":
		return "zap"
	}
	return "hammer" // dynamic-tool, update, mixed
}

// toneIcon is T3's workToneIcon.
func toneIcon(tone string) string {
	switch tone {
	case "error":
		return "circle-alert"
	case "thinking":
		return "brain"
	case "info":
		return "check"
	}
	return "zap"
}

// entryIcon is T3's workEntryIconName (TimelineWorkEntryRow.logic.ts:
// 149-193): a notification's says where it came from.
func (e *workEntry) icon() string {
	if it := e.item; e.itemType == "notification" && it != nil {
		if it.Outcome == "failed" {
			return "circle-alert"
		}
		kind := ""
		if it.Source != nil {
			kind = it.Source.Kind
		}
		switch kind {
		case "subagent", "delegated_task":
			return "bot"
		case "command":
			return "terminal"
		case "monitor":
			return "eye"
		case "system":
			return "t3-code"
		}
		return "zap"
	}
	if e.itemType == "user_input_request" || e.itemType == "approval_request" {
		return "message-circle"
	}
	if e.itemType == "todo_list" {
		return "list-todo"
	}
	if e.surface != "" {
		return e.surface
	}
	if e.itemType == "dynamic_tool" && isJavaScriptRepl(e.toolName) {
		return "code-xml"
	}
	if a := e.groupAction(); a != "other" {
		return summaryIcon(a)
	}
	switch e.itemType {
	case "dynamic_tool":
		return "wrench"
	case "subagent":
		return "bot"
	}
	return toneIcon(e.tone)
}

func isJavaScriptRepl(tool string) bool {
	return tool == "js" || tool == "cua_repl.js" || tool == "node_repl.js"
}

// ---------- labels ----------

var spaces = regexp.MustCompile(`\s+`)

func oneLine(s string) string { return strings.TrimSpace(spaces.ReplaceAllString(s, " ")) }

// displayLabel is T3's workEntryDisplayLabel: what an entry's row says.
func (e *workEntry) displayLabel(root string) string {
	if e.itemType == "system_notice" {
		return e.label
	}
	if e.itemType == "reasoning" || e.tone == "thinking" {
		if t := oneLine(e.detail); t != "" {
			return t
		}
		return e.label
	}
	if strings.TrimSpace(e.command) != "" {
		return commandDisplayText(e.command)
	}
	action := e.groupAction()
	if action == "code-search" || action == "search" {
		l := ""
		if e.itemType == "file_search" {
			l = e.label
		} else {
			l = searchLabel(e.toolData())
		}
		if l != "" {
			return l
		}
	}
	if action == "read" {
		if paths := e.readPaths(root); len(paths) > 0 {
			return readLabel(paths[0], len(paths)-1)
		}
	}
	retry := e.item != nil && e.item.Type == "error" && e.item.Retry != nil
	if title := e.dynamicTitle(); title != "" {
		return title
	}
	detail := strings.TrimSpace(e.detail)
	searchOutput := (action == "code-search" || action == "search") && strings.ContainsAny(detail, "\r\n")
	if detail != "" && !retry && action != "read" && !searchOutput {
		return detail
	}
	if len(e.changed) > 0 {
		p := workspacePath(e.changed[0], root)
		if len(e.changed) == 1 {
			return p
		}
		return fmt.Sprintf("%s +%d more", p, len(e.changed)-1)
	}
	if action == "read" && e.viewedImage == "" {
		return "Read file"
	}
	h := e.toolTitle
	if h == "" {
		h = e.label
	}
	h = compactToolLabel(h)
	r, n := utf8.DecodeRuneInString(h)
	if n == 0 {
		return h
	}
	return string(unicode.ToUpper(r)) + h[n:]
}

// singleLabel is T3's singleToolCallLabel: a lone tool row's label.
func (e *workEntry) singleLabel() string {
	if e.itemType == "reasoning" {
		if t := oneLine(e.detail); t != "" {
			return t
		}
		return "Thought"
	}
	if title := e.dynamicTitle(); title != "" {
		return title
	}
	if e.itemType == "web_search" {
		if e.toolTitle != "" {
			return e.toolTitle
		}
		return "Web search"
	}
	return e.displayLabel("")
}

// liveLabel is T3's liveWorkEntryLabel: "Running git" while it runs,
// "Ran git" after.
func (e *workEntry) liveLabel(root string, active bool) string {
	status := liveStatus(e.status, active)
	if e.itemType == "reasoning" {
		if t := oneLine(e.detail); t != "" {
			return t
		}
		if status == "inProgress" {
			return "Thinking"
		}
		return "Thought"
	}
	if c := strings.TrimSpace(e.command); c != "" {
		verb := map[string]string{"inProgress": "Running", "failed": "Failed", "declined": "Declined", "stopped": "Stopped"}[status]
		if verb == "" {
			verb = "Ran"
		}
		program := commandProgramName(c)
		if program == "" {
			program = "command"
		}
		return verb + " " + program
	}
	return e.displayLabel(root)
}

// liveStatus is T3's liveActivityToolStatus.
func liveStatus(status string, present bool) string {
	switch {
	case status == "failed" || status == "declined" || status == "stopped" || status == "idle":
		return status
	case present || status == "inProgress":
		return "inProgress"
	}
	return "completed"
}

// dynamicTitle is T3's dynamicToolTitle: a JavaScript REPL's title, a
// Claude skill's name.
func (e *workEntry) dynamicTitle() string {
	if e.itemType != "dynamic_tool" {
		return ""
	}
	input, _ := e.input.(map[string]any)
	if isJavaScriptRepl(e.toolName) {
		if t := strings.TrimSpace(str(input["title"])); t != "" {
			return t
		}
		if t := strings.TrimSpace(e.item.Title); t != "" && !strings.EqualFold(t, "js") && !strings.EqualFold(t, e.toolName) {
			return t
		}
		return "Run JavaScript"
	}
	if e.toolName == "Skill" {
		if name := strings.TrimSpace(str(input["skill"])); name != "" {
			return "Skill: " + name
		}
	}
	return ""
}

// toolData is what T3 keeps as an entry's toolData: a dynamic tool's input
// and output, otherwise the item.
func (e *workEntry) toolData() map[string]any {
	if e.itemType == "dynamic_tool" {
		return map[string]any{"input": e.input, "output": e.output}
	}
	return nil
}

// readPaths are the files a read entry read, as the workspace shows them.
func (e *workEntry) readPaths(root string) []string {
	raw := e.changed
	if len(raw) == 0 && e.itemType == "dynamic_tool" {
		raw = toolFilePaths(map[string]any{"input": e.input})
	}
	var out []string
	for _, p := range raw {
		out = append(out, workspacePath(p, root))
	}
	return out
}

// workspacePath is T3's formatWorkspaceRelativePath: a path under the
// workspace as "<workspace>/<rest>".
func workspacePath(p, root string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	if root == "" {
		return p
	}
	root = strings.TrimRight(strings.ReplaceAll(root, `\`, "/"), "/")
	label := root[strings.LastIndexByte(root, '/')+1:]
	switch {
	case p == root:
		return label
	case strings.HasPrefix(p, root+"/"):
		return label + "/" + p[len(root)+1:]
	case !strings.HasPrefix(p, "/"):
		if strings.HasPrefix(p, label+"/") {
			return p
		}
		return label + "/" + strings.TrimLeft(strings.TrimPrefix(p, "./"), "/")
	}
	return p
}

// readLabel is T3's formatReadToolLabel.
func readLabel(p string, more int) string {
	suffix := ""
	if more > 0 {
		suffix = fmt.Sprintf(" +%d more", more)
	}
	if p = strings.TrimSpace(p); p == "" {
		return "Read file" + suffix
	}
	return "Read " + p + suffix
}

var (
	searchQueryKeys  = []string{"pattern", "query", "searchTerm", "regex", "grep", "needle"}
	searchGlobKeys   = []string{"glob", "globPattern", "glob_pattern", "include", "filePattern", "file_pattern"}
	searchTargetKeys = []string{"path", "target_directory", "targetDirectory", "directory", "cwd", "root"}
)

// searchLabel is T3's formatSearchToolLabel: "Searched <query> in
// <target>" and its shorter forms.
func searchLabel(data map[string]any) string {
	input := data
	for _, k := range []string{"rawInput", "input"} {
		if r, ok := data[k].(map[string]any); ok && len(r) > 0 {
			input = r
			break
		}
	}
	firstOf := func(keys []string) string {
		for _, k := range keys {
			if s := strings.TrimSpace(str(input[k])); s != "" {
				return s
			}
		}
		return ""
	}
	query, glob := firstOf(searchQueryKeys), firstOf(searchGlobKeys)
	target := ""
	for _, part := range strings.FieldsFunc(firstOf(searchTargetKeys), func(r rune) bool { return r == '/' || r == '\\' }) {
		if part != "." {
			target = part
		}
	}
	switch {
	case query != "" && target != "":
		return "Searched " + query + " in " + target
	case glob != "" && target != "":
		return "Searched files " + glob + " in " + target
	case glob != "":
		return "Searched files " + glob
	case query != "":
		return "Searched " + query
	case target != "":
		return "Searched in " + target
	}
	return ""
}

var pathKeys = []string{"path", "filePath", "file_path", "relativePath", "filename", "fileName", "newPath", "oldPath"}

// toolFilePaths is T3's collectToolFilePaths: up to 8 file paths named in
// a tool's data.
func toolFilePaths(data any) []string {
	var paths []string
	seen := map[string]bool{}
	var walk func(v any, depth int)
	walk = func(v any, depth int) {
		if depth > 4 || len(paths) >= 8 {
			return
		}
		switch v := v.(type) {
		case []any:
			for _, x := range v {
				walk(x, depth+1)
			}
		case map[string]any:
			for _, k := range pathKeys {
				if s := strings.TrimSpace(str(v[k])); s != "" && !seen[s] && len(paths) < 8 {
					seen[s] = true
					paths = append(paths, s)
				}
			}
			for _, k := range []string{"locations", "item", "input", "result", "rawInput", "data", "changes"} {
				if x, ok := v[k]; ok {
					walk(x, depth+1)
				}
			}
		}
	}
	walk(data, 0)
	return paths
}

var imageExt = regexp.MustCompile(`(?i)\.(?:png|jpe?g|gif|webp|bmp|avif|svg|ico|heic)$`)

// isImagePath is whether a path names a picture the timeline could show.
func isImagePath(p string) bool { return imageExt.MatchString(p) }

// liveThoughtLine is T3's liveThoughtLine: a thought's bold heading, or
// its first sentence, as plain text.
var (
	thoughtHeading = regexp.MustCompile(`^\s*\*\*([^*\r\n]+)\*\*[ \t]*\r?(?:\n|$)`)
	mdLinkText     = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	mdLineMarker   = regexp.MustCompile(`(?m)^[ \t]*(?:#{1,6}|[-*+]|\d+\.)[ \t]+`)
	mdInlineMarks  = regexp.MustCompile("`+|\\*\\*|~~")
	mdEmphasis     = regexp.MustCompile(`(^|[^\w*])[*_]([^*_\n]+)[*_]([^\w*]|$)`)
	sentenceEnd    = regexp.MustCompile(`[.?!]["'”’)]?\s`)
)

func liveThoughtLine(md string) string {
	src, heading := md, false
	if m := thoughtHeading.FindStringSubmatch(md); m != nil {
		src, heading = m[1], true
	}
	s := mdLinkText.ReplaceAllString(src, "$1")
	s = mdLineMarker.ReplaceAllString(s, "")
	s = mdInlineMarks.ReplaceAllString(s, "")
	s = mdEmphasis.ReplaceAllString(s, "$1$2$3")
	s = oneLine(s)
	if heading {
		return s
	}
	if loc := sentenceEnd.FindStringIndex(s + " "); loc != nil && loc[1]-1 <= len(s) {
		return strings.TrimSpace(s[:loc[1]-1])
	}
	return s
}
