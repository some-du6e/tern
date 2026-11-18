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
