package core

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The timeline's rows from a thread's turn items, with the cases of T3's
// own tests (MessagesTimeline.logic.test.ts, presentation.test.ts).

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// at is s seconds past t0, as the server writes times.
func at(s int) string { return t0.Add(time.Duration(s) * time.Second).Format(time.RFC3339Nano) }

// turnItem is a turn item as the server sends it, s seconds past t0, with
// more fields as name, value pairs.
func turnItem(id, typ string, ordinal, s int, fields ...any) json.RawMessage {
	m := map[string]any{"id": id, "type": typ, "ordinal": ordinal, "status": "completed", "startedAt": at(s), "updatedAt": at(s)}
	if typ == "user_message" || typ == "assistant_message" {
		m["messageId"] = id
	}
	for i := 0; i+1 < len(fields); i += 2 {
		m[fields[i].(string)] = fields[i+1]
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

// timelineTest is a thread open on its items and runs.
type timelineTest struct {
	t *testing.T
	a *App
}

func newTimelineTest(t *testing.T, runs []t3.Run, items ...json.RawMessage) *timelineTest {
	t.Helper()
	a := testApp(t)
	a.openThread("swiss/t1")
	e, _ := a.thread("swiss/t1")
	v := newView(e, "t1", a.disclosureFor("swiss/t1"))
	for _, it := range items {
		v.put(it)
	}
	for _, r := range runs {
		v.runs[r.ID] = r
	}
	v.loaded = true
	a.view = v
	return &timelineTest{t, a}
}

// working makes the thread's run the one running.
func (tt *timelineTest) working(run string) {
	_, th := tt.a.thread("swiss/t1")
	th.ActiveRunID, th.ActivityRunStatus, th.ActivityRunStartedAt = run, "running", at(0)
}

func (tt *timelineTest) click(id string) {
	tt.a.Handle(nil, []ui.Action{{ID: id, Kind: "click"}})
}

func (tt *timelineTest) rows() []ui.TimelineRow { return tt.a.build(t0.Add(time.Hour)).Timeline.Rows }

// shape is the rows as T3's tests write them: "user:<id>", "fold",
// "work:<entry> + <entry>", or the kind.
func shape(rows []ui.TimelineRow) string {
	var out []string
	for _, r := range rows {
		switch r.Kind {
		case ui.RowUser:
			out = append(out, "user:"+r.Message.ID)
		case ui.RowAssistant:
			out = append(out, "assistant:"+r.Message.ID)
		case ui.RowAssistantMeta:
			out = append(out, "assistant-meta")
		case ui.RowTurnFold:
			out = append(out, "fold")
		case ui.RowWork:
			var ids []string
			for _, e := range r.Entries {
				ids = append(ids, e.ID)
			}
			out = append(out, "work:"+strings.Join(ids, " + "))
		case ui.RowWorkToggle:
			out = append(out, "work-toggle")
		case ui.RowWorkLive:
			out = append(out, "work-live:"+r.Entry.ID)
		case ui.RowThinking:
			out = append(out, "thinking")
		case ui.RowWorking:
			out = append(out, "working")
		case ui.RowCompaction:
			out = append(out, "compaction")
		case ui.RowDivider:
			out = append(out, "divider")
		case ui.RowPlan:
			out = append(out, "plan")
		}
	}
	return strings.Join(out, ", ")
}

func wantShape(t *testing.T, what string, rows []ui.TimelineRow, want string) {
	t.Helper()
	if got := shape(rows); got != want {
		t.Fatalf("%s:\n got  %s\n want %s", what, got, want)
	}
}

// A settled turn folds its commentary and work behind "Worked for", the
// run's own time; only its last reply offers copy (:974, :1100).
func TestSettledTurnFolds(t *testing.T) {
	tt := newTimelineTest(t, []t3.Run{{ID: "run-1", Status: "completed", StartedAt: at(0), CompletedAt: at(134), Ordinal: 1}},
		turnItem("u1", "user_message", 1, 0, "runId", "run-1", "text", "Write a poem", "inputIntent", "turn_start"),
		turnItem("a1", "assistant_message", 2, 10, "runId", "run-1", "text", "I should ground this first.", "updatedAt", at(11), "phase", "commentary"),
		turnItem("c1", "command_execution", 3, 12, "runId", "run-1", "input", "rg poem", "exitCode", 0),
		turnItem("a2", "assistant_message", 4, 20, "runId", "run-1", "text", "Here is the poem.", "updatedAt", at(30), "phase", "final_answer"),
	)
	rows := tt.rows()
	wantShape(t, "folded", rows, "user:u1, fold, assistant:a2")
	if rows[1].Label != "Worked for 2m 14s" {
		t.Errorf("fold label %q", rows[1].Label)
	}
	if m := rows[2].Message; !m.Meta || !m.Copy {
		t.Errorf("the reply has no copy row: %+v", m)
	}
	tt.click("fold:" + rows[1].Group)
	rows = tt.rows()
	wantShape(t, "unfolded", rows, "user:u1, fold, assistant:a1, work:c1, assistant:a2")
	if !rows[1].Open {
		t.Error("the fold isn't open")
	}
	if m := rows[2].Message; m.Meta || m.Copy {
		t.Errorf("the first reply offers copy: %+v", m)
	}
	if l := rows[3].Entries[0].Label; l != "rg poem" {
		t.Errorf("command row label %q", l)
	}
}

// An interrupted latest run reads "You stopped after" its own time
// (:1404).
func TestStoppedTurnFold(t *testing.T) {
	tt := newTimelineTest(t, []t3.Run{{ID: "turn-1", Status: "interrupted", StartedAt: at(0), CompletedAt: at(47), Ordinal: 1}},
		turnItem("work-1", "command_execution", 1, 5, "runId", "turn-1", "input", "make"),
	)
	rows := tt.rows()
	wantShape(t, "rows", rows, "fold")
	if rows[0].Label != "You stopped after 47s" || rows[0].Open {
		t.Errorf("fold %q open=%v", rows[0].Label, rows[0].Open)
	}
}

// A run the user stops opens its fold; the next run closes it again
// (MessagesTimeline.tsx:596-631).
func TestStoppedRunOpensItsFold(t *testing.T) {
	tt := newTimelineTest(t, []t3.Run{{ID: "r1", Status: "running", StartedAt: at(0), Ordinal: 1}},
		turnItem("p", "user_message", 1, 0, "runId", "r1", "text", "Go", "inputIntent", "turn_start"),
		turnItem("c", "command_execution", 2, 1, "runId", "r1", "input", "make"),
	)
	tt.working("r1")
	wantShape(t, "running", tt.rows(), "user:p, working, work-live:c")
	_, th := tt.a.thread("swiss/t1")
	th.ActiveRunID, th.ActivityRunStatus = "", "interrupted"
	tt.a.view.runs["r1"] = t3.Run{ID: "r1", Status: "interrupted", StartedAt: at(0), CompletedAt: at(5), Ordinal: 1}
	rows := tt.rows()
	wantShape(t, "stopped", rows, "user:p, fold, work:c")
	if !rows[1].Open || rows[1].Label != "You stopped after 5s" {
		t.Errorf("fold %q open=%v", rows[1].Label, rows[1].Open)
	}
	tt.a.view.put(turnItem("p2", "user_message", 3, 10, "runId", "r2", "text", "Again", "inputIntent", "turn_start"))
	tt.a.view.runs["r2"] = t3.Run{ID: "r2", Status: "running", StartedAt: at(10), Ordinal: 2}
	tt.working("r2")
	rows = tt.rows()
	wantShape(t, "the next run", rows, "user:p, fold, user:p2, working, thinking")
	if rows[1].Open {
		t.Error("the stopped run's fold stayed open")
	}
}

// Work after a steer folds under its own "Continued for" below the steer,
// and the turn's total stays on top (:1519).
func TestSteerSplitsTheFold(t *testing.T) {
	run := "steered-run"
	tt := newTimelineTest(t, []t3.Run{{ID: run, Status: "completed", StartedAt: at(0), CompletedAt: at(30), Ordinal: 1}},
		turnItem("prompt", "user_message", 1, 0, "runId", run, "text", "prompt", "inputIntent", "turn_start"),
		turnItem("work-before-steer", "command_execution", 2, 4, "runId", run, "input", "ls"),
		turnItem("steer", "user_message", 3, 10, "runId", run, "text", "steer", "inputIntent", "steer"),
		turnItem("work-after-steer", "command_execution", 4, 15, "runId", run, "input", "ls -a"),
		turnItem("final", "assistant_message", 5, 30, "runId", run, "text", "final"),
	)
	rows := tt.rows()
	wantShape(t, "folded", rows, "user:prompt, fold, user:steer, fold, assistant:final")
	if rows[1].Label != "Worked for 30s" || rows[3].Label != "Continued for 20s" || !rows[3].At.Equal(t0.Add(10*time.Second)) {
		t.Errorf("folds %q, %q at %v", rows[1].Label, rows[3].Label, rows[3].At)
	}
	if rows[2].Message.Steer != ui.SteerSteer {
		t.Errorf("the steer isn't marked: %+v", rows[2].Message)
	}
	tt.click("fold:" + rows[3].Group)
	wantShape(t, "the steer's segment open", tt.rows(), "user:prompt, fold, user:steer, fold, work:work-after-steer, assistant:final")
}

// An answered question keeps its own row, outside the groups and the
// fold (:2151).
func TestAnsweredQuestionKeepsItsRow(t *testing.T) {
	run := "answered-question-run"
	answer := map[string]any{"requestId": "scope-question", "questionTextById": map[string]string{"scope": "Which repository?"},
		"answers": map[string]string{"scope": "The private one"}, "attachmentsByQuestionId": map[string]any{}}
	items := func(settled bool) []json.RawMessage {
		last := turnItem("answer", "assistant_message", 4, 5, "runId", run, "text", "Done")
		if !settled {
			last = turnItem("running", "command_execution", 4, 4, "runId", run, "input", "cat running", "status", "running")
		}
		return []json.RawMessage{
			turnItem("prompt", "user_message", 0, 0, "runId", run, "text", "Pick a repository", "inputIntent", "turn_start", "createdBy", "user"),
			turnItem("before", "command_execution", 1, 1, "runId", run, "input", "cat before"),
			turnItem("question", "user_input_request", 2, 2, "runId", run, "requestId", "scope-question", "questions", []any{}, "questionAnswer", answer),
			turnItem("after", "command_execution", 3, 3, "runId", run, "input", "cat after"),
			last,
		}
	}
	settled := newTimelineTest(t, []t3.Run{{ID: run, Status: "completed", StartedAt: at(0), CompletedAt: at(5), Ordinal: 1}}, items(true)...)
	rows := settled.rows()
	wantShape(t, "settled", rows, "user:prompt, fold, work:question, assistant:answer")
	if e := rows[2].Entries[0]; e.Label != "Which repository?" || e.Answer != "The private one" {
		t.Errorf("question row %q → %q", e.Label, e.Answer)
	}
	settled.click("fold:" + run)
	wantShape(t, "settled, unfolded", settled.rows(), "user:prompt, fold, work:before, work:question, work:after, assistant:answer")

	live := newTimelineTest(t, []t3.Run{{ID: run, Status: "running", StartedAt: at(0), Ordinal: 1}}, items(false)...)
	live.working(run)
	rows = live.rows()
	wantShape(t, "running", rows, "user:prompt, working, work:before, work:question, work-live:running")
	live.click("group:" + rows[4].Group)
	wantShape(t, "running, its group open", live.rows(), "user:prompt, working, work:before, work:question, work-live:running, work:after + running")
}

// The live row shows the running tool with the newest thought over it
// (:1867), and a running tool keeps the shared activity row without
// "Thinking" (:1921).
func TestLiveActivityRow(t *testing.T) {
	run := []t3.Run{{ID: "turn-1", Status: "running", StartedAt: at(0), Ordinal: 1}}
	tt := newTimelineTest(t, run,
		turnItem("old-thought", "reasoning", 1, 1, "runId", "turn-1", "text", "First idea."),
		turnItem("new-thought", "reasoning", 2, 2, "runId", "turn-1", "text", "Found the cause."),
		turnItem("running-command", "command_execution", 3, 3, "runId", "turn-1", "input", "rg cause", "status", "running"),
	)
	tt.working("turn-1")
	rows := tt.rows()
	wantShape(t, "thought", rows, "working, work-live:running-command")
	if r := rows[1]; r.Entry.Label != "Running rg" || r.Thought != "Found the cause." || !r.Live {
		t.Errorf("live row %q, thought %q, live %v", r.Entry.Label, r.Thought, r.Live)
	}

	tt = newTimelineTest(t, run,
		turnItem("running-command", "command_execution", 1, 5, "runId", "turn-1", "input", "rg toolCall", "status", "running"),
		turnItem("completed-edit", "file_change", 2, 6, "runId", "turn-1", "fileName", "src/one.ts", "additions", 3, "deletions", 1),
		turnItem("completed-command", "command_execution", 3, 7, "runId", "turn-1", "input", "vp test run", "exitCode", 0),
	)
	tt.working("turn-1")
	rows = tt.rows()
	wantShape(t, "running tool", rows, "working, work-live:running-command")
	tt.click("group:" + rows[1].Group)
	wantShape(t, "its group open", tt.rows(), "working, work-live:running-command, work:running-command + completed-edit + completed-command")
}

// While the run works with nothing live in view, "Thinking" ends the
// timeline; a pending approval takes its place (§5.2.6).
func TestThinkingRow(t *testing.T) {
	run := []t3.Run{{ID: "turn-1", Status: "running", StartedAt: at(0), Ordinal: 1}}
	prompt := turnItem("p", "user_message", 1, 0, "runId", "turn-1", "text", "Go", "inputIntent", "turn_start")
	tt := newTimelineTest(t, run, prompt)
	tt.working("turn-1")
	wantShape(t, "working", tt.rows(), "user:p, working, thinking")

	tt = newTimelineTest(t, run, prompt,
		turnItem("ap", "approval_request", 2, 3, "runId", "turn-1", "requestId", "req-1", "requestKind", "command", "prompt", "git push", "status", "pending"))
	tt.working("turn-1")
	_, th := tt.a.thread("swiss/t1")
	th.ActivityRunStatus, th.PendingRuntimeRequest = "waiting", &t3.RuntimeRequest{ID: "req-1", Kind: "command"}
	rows := tt.rows()
	wantShape(t, "waiting on an approval", rows, "user:p, working, work:ap")
	if a := rows[2].Entries[0].Approval; a == nil || a.Detail != "git push" || a.ID != "req-1" {
		t.Errorf("approval row %+v", a)
	}
}

// A bare /compact turn is its divider alone, with its token counts
// (§5.2.8).
func TestCompactionTurn(t *testing.T) {
	tt := newTimelineTest(t, []t3.Run{{ID: "run-c", Status: "completed", StartedAt: at(0), CompletedAt: at(9), Ordinal: 1}},
		turnItem("p", "user_message", 1, 0, "runId", "run-c", "text", "/compact", "inputIntent", "turn_start"),
		turnItem("c", "compaction", 2, 1, "runId", "run-c", "beforeTokenCount", 180000, "afterTokenCount", 11700),
	)
	rows := tt.rows()
	wantShape(t, "rows", rows, "compaction")
	if r := rows[0]; r.Label != "Context compacted" || r.Before != "180K" || r.After != "11.7K" {
		t.Errorf("compaction %q %q → %q", r.Label, r.Before, r.After)
	}
}

// A tool failed when its exit code says so, or its output does when there
// is no exit code; a zero exit code outranks failure text, and reasoning
// is never scanned (presentation.test.ts).
func TestToolFailure(t *testing.T) {
	for _, c := range []struct {
		name   string
		item   json.RawMessage
		failed bool
	}{
		{"exit code 2", turnItem("w", "command_execution", 2, 1, "runId", "r", "input", `rg "command not found"`, "exitCode", 2), true},
		{"command not found", turnItem("w", "command_execution", 2, 1, "runId", "r", "input", "missing-command", "output", "sh: missing-command: command not found"), true},
		{"output past the scan", turnItem("w", "command_execution", 2, 1, "runId", "r", "input", "cat big", "output", strings.Repeat("x", 32768)+" command not found"), false},
		{"exit code 0 over the text", turnItem("w", "command_execution", 2, 1, "runId", "r", "input", `rg -n "ENOENT|exit code" logs fixtures`,
			"output", "logs/app.log:12: Error: ENOENT: no such file or directory", "exitCode", 0, "outputIndicatesFailure", true), false},
		{"reasoning", turnItem("w", "reasoning", 2, 1, "runId", "r", "text", "That exit code 2 is just from listing the deleted directory, which is expected."), false},
	} {
		tt := newTimelineTest(t, []t3.Run{{ID: "r", Status: "completed", StartedAt: at(0), CompletedAt: at(9), Ordinal: 1}},
			turnItem("p", "user_message", 1, 0, "runId", "r", "text", "Go", "inputIntent", "turn_start"), c.item,
			turnItem("a", "assistant_message", 3, 2, "runId", "r", "text", "Done"))
		tt.click("fold:r")
		rows := tt.rows()
		wantShape(t, c.name, rows, "user:p, fold, work:w, assistant:a")
		if e := rows[2].Entries[0]; e.Failed != c.failed {
			t.Errorf("%s: failed = %v, want %v", c.name, e.Failed, c.failed)
		}
	}
}

// Once the provider's turn has ended, a run still settling has no
// "Thinking" row (MessagesTimeline.logic.ts:2274-2281).
func TestTurnEndedDropsThinking(t *testing.T) {
	run := []t3.Run{{ID: "turn-1", Status: "running", StartedAt: at(0), Ordinal: 1, ActiveAttemptID: "attempt-1"}}
	prompt := turnItem("p", "user_message", 1, 0, "runId", "turn-1", "text", "Go", "inputIntent", "turn_start")
	for _, c := range []struct {
		status, want string
	}{{"running", "user:p, working, thinking"}, {"completed", "user:p, working"}} {
		tt := newTimelineTest(t, run, prompt)
		tt.working("turn-1")
		tt.a.view.turns["pt-1"] = t3.ProviderTurn{ID: "pt-1", RunAttemptID: "attempt-1", Ordinal: 1, Status: c.status}
		wantShape(t, "provider turn "+c.status, tt.rows(), c.want)
	}
}

// T3's own /compact on returning holds the message back, so the working
// row says "Compacting" and no divider shows (MessagesTimeline.logic.ts:
// 1783-1812, 2171-2190).
func TestCompactingOnReturn(t *testing.T) {
	tt := newTimelineTest(t, []t3.Run{{ID: "run-c", Status: "running", StartedAt: at(0), Ordinal: 1}},
		turnItem("p", "user_message", 1, 0, "runId", "run-c", "text", "/compact", "inputIntent", "turn_start",
			"createdBy", "agent", "creationSource", "server", "idleCompaction", "onReturn"))
	tt.working("run-c")
	rows := tt.rows()
	wantShape(t, "compacting", rows, "working")
	if !rows[0].Compacting {
		t.Error("the working row doesn't say it's compacting")
	}
}

// An approval whose provider is gone can't be answered: its row says so
// (TimelinePendingApprovalRow.tsx:40, 95-99).
func TestApprovalProviderGone(t *testing.T) {
	run := []t3.Run{{ID: "turn-1", Status: "running", StartedAt: at(0), Ordinal: 1}}
	for _, c := range []struct {
		capability string
		live       bool
	}{{"live", true}, {"not_resumable", false}} {
		tt := newTimelineTest(t, run, turnItem("p", "user_message", 1, 0, "runId", "turn-1", "text", "Go", "inputIntent", "turn_start"),
			turnItem("ap", "approval_request", 2, 3, "runId", "turn-1", "requestId", "req-1", "requestKind", "command", "prompt", "git push", "status", "pending"))
		tt.working("turn-1")
		_, th := tt.a.thread("swiss/t1")
		th.ActivityRunStatus, th.PendingRuntimeRequest = "waiting", &t3.RuntimeRequest{ID: "req-1", Kind: "command"}
		rr := t3.RuntimeRequestRecord{ID: "req-1", Kind: "command", Status: "pending"}
		rr.ResponseCapability.Type = c.capability
		tt.a.view.requests["req-1"] = rr
		rows := tt.rows()
		wantShape(t, c.capability, rows, "user:p, working, work:ap")
		if a := rows[2].Entries[0].Approval; a == nil || a.Live != c.live {
			t.Errorf("%s: approval %+v", c.capability, a)
		}
	}
}

// openRows is a settled turn of one work item, its fold opened, and the
// rows: the item's entry is rows[2].
func openRows(t *testing.T, item json.RawMessage) (*timelineTest, []ui.TimelineRow) {
	t.Helper()
	tt := newTimelineTest(t, []t3.Run{{ID: "r", Status: "completed", StartedAt: at(0), CompletedAt: at(9), Ordinal: 1}},
		turnItem("p", "user_message", 1, 0, "runId", "r", "text", "Go", "inputIntent", "turn_start"), item,
		turnItem("a", "assistant_message", 3, 2, "runId", "r", "text", "Done"))
	tt.click("fold:r")
	rows := tt.rows()
	wantShape(t, "rows", rows, "user:p, fold, work:w, assistant:a")
	return tt, rows
}

// A tool call's arguments keep the call's own order, a string as it is
// and anything else as compact JSON; html, a JavaScript REPL's code and
// JSON are code (itemDetail.ts:101-130, V2ItemInspector.tsx:197-217).
func TestToolArgsKeepTheirOrder(t *testing.T) {
	tt, _ := openRows(t, turnItem("w", "dynamic_tool", 2, 1, "runId", "r", "toolName", "node_repl.js",
		"input", json.RawMessage(`{"zeta":"last","code":"let a = 1","html":"<p>hi</p>","alpha":{"b": 1, "a": [1, 2]},"empty":""}`)))
	tt.click("entry:w")
	rows := tt.rows()
	b := rows[2].Entries[0].Body
	if b == nil {
		t.Fatal("the open entry has no body")
	}
	want := []ui.ToolArg{{Key: "zeta", Value: "last"}, {Key: "code", Value: "let a = 1", Lang: "typescript"},
		{Key: "html", Value: "<p>hi</p>", Lang: "html"}, {Key: "alpha", Value: `{"b":1,"a":[1,2]}`, Lang: "json"}, {Key: "empty", Value: `""`, Lang: "json"}}
	if len(b.Args) != len(want) {
		t.Fatalf("args %+v", b.Args)
	}
	for i := range want {
		if b.Args[i] != want[i] {
			t.Errorf("arg %d is %+v, want %+v", i, b.Args[i], want[i])
		}
	}
}

// A notification's icon says where it came from, and a failed one is an
// alert (TimelineWorkEntryRow.logic.ts:149-193); its body is its detail.
func TestNotificationIcon(t *testing.T) {
	for _, c := range []struct {
		kind, outcome, icon string
	}{{"command", "completed", "terminal"}, {"subagent", "completed", "bot"}, {"monitor", "updated", "eye"},
		{"system", "completed", "t3-code"}, {"background_task", "completed", "zap"}, {"command", "failed", "circle-alert"}} {
		tt := newTimelineTest(t, []t3.Run{{ID: "r", Status: "completed", StartedAt: at(0), CompletedAt: at(9), Ordinal: 1}},
			turnItem("w", "notification", 1, 0, "runId", "r", "summary", "Build finished", "detail", "All green.",
				"outcome", c.outcome, "source", map[string]any{"kind": c.kind}))
		rows := tt.rows()
		wantShape(t, c.kind, rows, "work:w")
		if e := rows[0].Entries[0]; e.Icon != c.icon || !e.Expands {
			t.Errorf("%s %s: icon %q, expands %v", c.kind, c.outcome, e.Icon, e.Expands)
		}
	}
}

// The usage limit's reset names the zone as Intl does, by its IANA name
// (TimelineWorkEntryRow.tsx:249-255).
func TestUsageLimitZone(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	if got := upcomingTime(t0, t0); !strings.HasSuffix(got, " (Europe/Paris)") {
		t.Errorf("reset at %q", got)
	}
	t.Setenv("TZ", ":/usr/share/zoneinfo/Asia/Tokyo")
	if got := localZone(t0); got != "Asia/Tokyo" {
		t.Errorf("TZ as a path names %q", got)
	}
}

// Claude's ultrathink sends an attachment-only prompt as
// "Ultrathink:\n<prompt>"; nobody sees it (composerContextReferences.ts:
// 181-184).
func TestUltrathinkBootstrapPromptHidden(t *testing.T) {
	tt := newTimelineTest(t, nil, turnItem("p", "user_message", 1, 0, "text", "Ultrathink:\n"+attachmentOnlyPrompt, "inputIntent", "turn_start"))
	for _, r := range tt.rows() {
		if r.Message != nil && strings.Contains(r.Message.Text, "User attached") {
			t.Fatalf("the bubble says %q", r.Message.Text)
		}
	}
}

// An open thought is the whole trace: only its one-line preview is cut
// short (TimelineWorkEntryRow.tsx:121-145).
func TestOpenThoughtIsWhole(t *testing.T) {
	long := strings.Repeat("é", 6000) // 12000 bytes
	_, rows := openRows(t, turnItem("w", "reasoning", 2, 1, "runId", "r", "text", long))
	if got := rows[2].Entries[0].Thought; got != long {
		t.Fatalf("the trace is %d bytes of %d", len(got), len(long))
	}
}

// A web result's title opens its page, when it's a web page's
// (V2ItemInspector.tsx:355-372).
func TestWebSearchHitAddress(t *testing.T) {
	tt, _ := openRows(t, turnItem("w", "web_search", 2, 1, "runId", "r", "patterns", []string{"gio list"},
		"results", []map[string]any{{"title": "Gio", "url": "https://gioui.org/doc"}, {"title": "Local", "url": "file:///etc/hosts"}}))
	tt.click("entry:w")
	b := tt.rows()[2].Entries[0].Body
	if b == nil || len(b.Hits) != 2 || b.Hits[0].URL != "https://gioui.org/doc" || b.Hits[1].URL != "" {
		t.Fatalf("hits %+v", b)
	}
}

// An open entry whose output the socket left out fetches it: when it's
// opened, again when its thread comes back with it still open, and again
// for each new revision of it, as when it finishes (itemDetail.ts:145-158).
func TestOpenEntryFetchesItsOutput(t *testing.T) {
	fetched := make(chan string, 4)
	output := "first"
	defer func(f func(*t3.Conn, string, string) (json.RawMessage, error)) { getTurnItem = f }(getTurnItem)
	getTurnItem = func(_ *t3.Conn, _, itemID string) (json.RawMessage, error) {
		fetched <- itemID
		return json.Marshal(map[string]any{"id": itemID, "type": "command_execution", "status": "completed", "output": output})
	}
	item := func(s int) json.RawMessage {
		return turnItem("w", "command_execution", 2, 1, "runId", "r", "input", "make", "outputOmitted", true, "updatedAt", at(s))
	}
	tt, _ := openRows(t, item(1))
	e, _ := tt.a.thread("swiss/t1")
	e.conn = new(t3.Conn)
	// Fetches land from their own goroutines, so build as Scene does,
	// under the lock.
	locked := func(f func()) {
		tt.a.mu.Lock()
		defer tt.a.mu.Unlock()
		f()
	}
	body := func() (b *ui.EntryBody) {
		locked(func() { b = tt.a.build(t0.Add(time.Hour)).Timeline.Rows[2].Entries[0].Body })
		return b
	}
	wait := func(what, want string) {
		t.Helper()
		select {
		case <-fetched:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: nothing fetched", what)
		}
		// The fetch lands under the lock; a build after it shows it.
		b := body()
		for i := 0; i < 200 && b.Output != want; i++ {
			time.Sleep(10 * time.Millisecond)
			b = body()
		}
		if b.Output != want {
			t.Fatalf("%s: output %q, want %q", what, b.Output, want)
		}
	}
	tt.click("entry:w")
	if b := body(); b == nil || !b.Pending {
		t.Fatalf("opened: %+v", b)
	}
	wait("opened", "first")
	// Back to the thread, the entry still open.
	var v *threadView
	locked(func() {
		v = newView(e, "t1", tt.a.disclosureFor("swiss/t1"))
		for _, raw := range []json.RawMessage{
			turnItem("p", "user_message", 1, 0, "runId", "r", "text", "Go", "inputIntent", "turn_start"), item(1),
			turnItem("a", "assistant_message", 3, 2, "runId", "r", "text", "Done")} {
			v.put(raw)
		}
		v.runs["r"], v.loaded = tt.a.view.runs["r"], true
		tt.a.view = v
	})
	if b := body(); !b.Pending {
		t.Fatalf("restored: %+v", b)
	}
	wait("restored", "first")
	// A new revision of it is fetched again.
	output = "second"
	locked(func() { v.put(item(5)) })
	body()
	wait("updated", "second")
}

// A message of only files shows them, as cards, rather than dropping out
// of the timeline with its bootstrap prompt (TimelineUserMessageRow.tsx:
// 104-178, 529-541).
func TestFileOnlyMessage(t *testing.T) {
	file := map[string]any{"type": "file", "id": "f1", "name": "report.pdf", "mimeType": "application/pdf", "sizeBytes": 2516582}
	pasted := map[string]any{"type": "file", "id": "f2", "name": "pasted.txt", "mimeType": "text/plain", "sizeBytes": 900,
		"source": map[string]any{"_tag": "pasted-text"}}
	tt := newTimelineTest(t, nil, turnItem("p", "user_message", 1, 0, "text", attachmentOnlyPrompt, "inputIntent", "turn_start",
		"attachments", []any{file, pasted}))
	rows := tt.rows()
	wantShape(t, "rows", rows, "user:p")
	m := rows[0].Message
	if m.Text != "" || len(m.Files) != 2 || m.Files[0] != (ui.File{ID: "f1", Name: "report.pdf", Size: 2516582}) || !m.Files[1].Pasted {
		t.Fatalf("message %+v", m)
	}
}

// "Copy prompt link" copies a new-thread link on T3's hosted app, the
// prompt form-encoded as URLSearchParams does, its pictures as their
// labels; empty prompts, the attachment-only prompt and prompts over 8000
// characters have none (copyPromptLink.ts, promptLinks.ts).
func TestPromptLink(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"Fix the *bug* ~now~ & ship it", "https://app.t3.codes/new?prompt=Fix+the+*bug*+%7Enow%7E+%26+ship+it"},
		{"  café\n", "https://app.t3.codes/new?prompt=caf%C3%A9"},
		{"Like ![Image 1](t3-context://v1/image/image_1)", "https://app.t3.codes/new?prompt=Like+Image+1"},
		{"   ", ""},
		{attachmentOnlyPrompt, ""},
		{strings.Repeat("a", 8001), ""},
	} {
		if got := promptLink(c.text); got != c.want {
			t.Errorf("promptLink(%.30q) = %q, want %q", c.text, got, c.want)
		}
	}
	if got := promptLink(strings.Repeat("a", 8000)); got == "" {
		t.Error("an 8000-character prompt has no link")
	}
}

// "Fork from this response" is on a finished reply of a run unless its
// provider's session can't fork; clicking it forks the thread from that
// run into "<title> fork", the button pulsing until it's done, then opens
// the fork (useTimelineRewind.ts:174-212).
func TestForkFromReply(t *testing.T) {
	sent := make(chan map[string]any, 1)
	done := make(chan error)
	defer func(f func(*t3.Conn, map[string]any) error) { forkThread = f }(forkThread)
	forkThread = func(_ *t3.Conn, cmd map[string]any) error {
		sent <- cmd
		return <-done
	}
	tt := newTimelineTest(t, []t3.Run{{ID: "run-1", Status: "completed", StartedAt: at(0), CompletedAt: at(30), Ordinal: 1}},
		turnItem("u1", "user_message", 1, 0, "runId", "run-1", "text", "Write a poem", "inputIntent", "turn_start"),
		turnItem("a1", "assistant_message", 2, 20, "runId", "run-1", "text", "Here is the poem.", "providerThreadId", "pt"),
	)
	reply := func() *ui.Message {
		tt.a.mu.Lock()
		defer tt.a.mu.Unlock()
		return tt.a.build(t0.Add(time.Hour)).Timeline.Rows[1].Message
	}
	if m := reply(); !m.Fork || m.Forking {
		t.Fatalf("the reply: fork %v, forking %v", m.Fork, m.Forking)
	}
	v := tt.a.view
	v.pthreads["pt"] = t3.ProviderThread{ID: "pt", ProviderSessionID: "ps"}
	v.sessions["ps"] = t3.ProviderSession{ID: "ps"}
	if reply().Fork {
		t.Error("a session that can't fork offers fork")
	}
	s := v.sessions["ps"]
	s.Capabilities.Context.SupportsFullThreadHandoff = true
	v.sessions["ps"] = s
	if !reply().Fork {
		t.Error("a session that hands threads over offers no fork")
	}
	e, th := tt.a.thread("swiss/t1")
	e.conn, th.Title = new(t3.Conn), "Poems"
	tt.click("fork:a1")
	cmd := <-sent
	point, _ := cmd["sourcePoint"].(map[string]any)
	if cmd["type"] != "thread.fork" || cmd["sourceThreadId"] != "t1" || point["runId"] != "run-1" || cmd["title"] != "Poems fork" {
		t.Fatalf("fork command %+v", cmd)
	}
	if !reply().Forking {
		t.Error("the button doesn't say the fork is on its way")
	}
	// The fork opens without a connection to stream it from.
	tt.a.mu.Lock()
	e.conn = nil
	tt.a.mu.Unlock()
	done <- nil
	for i := 0; i < 200 && reply().Forking; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	tt.a.RunUI(ui.NewTheme())
	if want := "swiss/" + cmd["targetThreadId"].(string); tt.a.openID != want {
		t.Errorf("open thread %q, want the fork %q", tt.a.openID, want)
	}
}

// A quoted response's card goes to its reply: the fold hiding it opens,
// and the timeline brings its row into view once (MessagesTimeline.tsx:
// 664-705). A reply the thread doesn't hold reveals nothing.
func TestCitedReplyIsRevealed(t *testing.T) {
	tt := newTimelineTest(t, []t3.Run{{ID: "run-1", Status: "completed", StartedAt: at(0), CompletedAt: at(134), Ordinal: 1}},
		turnItem("u1", "user_message", 1, 0, "runId", "run-1", "text", "Write a poem", "inputIntent", "turn_start"),
		turnItem("a1", "assistant_message", 2, 10, "runId", "run-1", "text", "I should ground this first.", "phase", "commentary"),
		turnItem("c1", "command_execution", 3, 12, "runId", "run-1", "input", "rg poem", "exitCode", 0),
		turnItem("a2", "assistant_message", 4, 20, "runId", "run-1", "text", "Here is the poem.", "phase", "final_answer"),
	)
	wantShape(t, "folded", tt.rows(), "user:u1, fold, assistant:a2")
	tt.click("cite:swiss/t1/a1")
	s := tt.a.build(t0.Add(time.Hour))
	wantShape(t, "revealed", s.Timeline.Rows, "user:u1, fold, assistant:a1, work:c1, assistant:a2")
	if s.Timeline.Reveal != "a1" {
		t.Errorf("reveal %q, want a1", s.Timeline.Reveal)
	}
	if r := tt.a.build(t0.Add(time.Hour)).Timeline.Reveal; r != "" {
		t.Errorf("the reply is revealed again: %q", r)
	}
	tt.click("cite:swiss/t1/gone")
	if r := tt.a.build(t0.Add(time.Hour)).Timeline.Reveal; r != "" || tt.a.cited != (citedReply{}) {
		t.Errorf("a reply that isn't there: reveal %q, still waiting for %+v", r, tt.a.cited)
	}
}
