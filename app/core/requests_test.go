package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// openWith opens thread swiss/t1 with a loaded view holding items and
// runtime requests, the way its snapshot would.
func openWith(t *testing.T, a *App, pending *t3.RuntimeRequest, reqs []t3.Request, items ...map[string]any) *threadView {
	t.Helper()
	a.openThread("swiss/t1")
	e, th := a.thread("swiss/t1")
	th.PendingRuntimeRequest = pending
	v := newView(e, "t1")
	v.loaded = true
	if reqs != nil {
		v.requests = map[string]t3.Request{}
		for _, r := range reqs {
			v.requests[r.ID] = r
		}
	}
	for _, it := range items {
		raw, _ := json.Marshal(it)
		v.put(raw)
	}
	a.view = v
	return v
}

func request(id, kind, status, capability string, feedback bool) t3.Request {
	r := t3.Request{ID: id, Kind: kind, Status: status, CreatedAt: "2026-11-18T10:00:00Z"}
	r.ResponseCapability.Type, r.ResponseCapability.SupportsFeedback = capability, feedback
	return r
}

// The thread's pending approval is a row in its timeline, with the
// request's detail and its answers, at its approval_request item
// (TimelinePendingApprovalRow.tsx); auth refreshes and dynamic tool calls
// are never approvals, and a request the provider can't answer any more
// shows that it's gone.
func TestApprovalRow(t *testing.T) {
	a := testApp(t)
	openWith(t, a, &t3.RuntimeRequest{ID: "r1", Kind: "command"},
		[]t3.Request{
			request("r1", "command", "pending", "live", false),
			request("r2", "auth_refresh", "pending", "live", false),
			request("r3", "dynamic_tool_call", "pending", "live", false),
			request("r4", "file-change", "resolved", "live", false),
		},
		map[string]any{"id": "i1", "type": "assistant_message", "text": "about to run it", "ordinal": 1},
		map[string]any{"id": "i2", "type": "approval_request", "requestId": "r1", "requestKind": "command", "prompt": "rm -rf build", "ordinal": 2},
	)
	s := a.build(time.Now())
	var rows []*ui.PendingApproval
	for _, it := range s.Items {
		if it.Kind == ui.ItemApproval {
			rows = append(rows, it.Approval)
		}
	}
	if len(rows) != 1 || rows[0].ID != "r1" || rows[0].Detail != "rm -rf build" || !rows[0].Live {
		t.Fatalf("approval rows = %+v", rows)
	}
	if s.Placeholder != "Send a message to decline and redirect the agent" {
		t.Fatalf("placeholder = %q", s.Placeholder)
	}

	a = testApp(t)
	openWith(t, a, &t3.RuntimeRequest{ID: "r1", Kind: "command"}, []t3.Request{request("r1", "command", "pending", "not_resumable", false)})
	s = a.build(time.Now())
	if it := s.Items[len(s.Items)-1]; it.Kind != ui.ItemApproval || it.Approval.Live {
		t.Fatalf("a request the provider can't answer: %+v", it)
	}
}

// A send while approvals show declines the ones the provider can still
// answer, then goes as a message; when the first takes feedback, the
// message is its decline instead (useComposerSend.ts).
func TestRedirect(t *testing.T) {
	live, gone, fb := approval{id: "a", live: true}, approval{id: "b"}, approval{id: "c", live: true, feedback: true}
	if f, d := redirect([]approval{live, gone}); f != "" || len(d) != 1 || d[0] != "a" {
		t.Fatalf("live and gone: feedback %q, decline %v", f, d)
	}
	if f, d := redirect([]approval{fb, live}); f != "c" || len(d) != 0 {
		t.Fatalf("feedback first: feedback %q, decline %v", f, d)
	}
	if f, d := redirect(nil); f != "" || len(d) != 0 {
		t.Fatalf("none: %q %v", f, d)
	}
	a := testApp(t)
	openWith(t, a, &t3.RuntimeRequest{ID: "r1", Kind: "command"}, []t3.Request{request("r1", "command", "pending", "live", true)})
	if s := a.build(time.Now()); s.Placeholder != "Send feedback to decline this request" {
		t.Fatalf("placeholder = %q", s.Placeholder)
	}
}

// A request that comes in while the user types is held back from the
// timeline and the composer until 1.5s pass without a key, the prompt
// loses the focus, or a send (composerTypingGuard.ts).
func TestTypingGuard(t *testing.T) {
	var g typingGuard
	now := time.Now()
	g.requests("k", nil, now)
	g.focused, g.typed = true, now
	g.requests("k", []string{"r1"}, now.Add(time.Second))
	if !g.held["r1"] {
		t.Fatal("a request arriving mid-typing isn't held")
	}
	g.release()
	g.requests("k", []string{"r1"}, now.Add(2*time.Second))
	if g.held["r1"] {
		t.Fatal("released, it's held again")
	}
	g.typed = now.Add(2 * time.Second)
	g.requests("k", []string{"r1", "r2"}, now.Add(5*time.Second))
	if g.held["r2"] {
		t.Fatal("a request after the typing stopped is held")
	}

	a := testApp(t)
	openWith(t, a, &t3.RuntimeRequest{ID: "r1", Kind: "command"}, nil)
	a.build(time.Now())
	th := ui.NewTheme()
	a.handle(th, ui.Action{ID: "composer", Kind: "focus"})
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "ty"})
	_, tt := a.thread("swiss/t1")
	tt.PendingRuntimeRequest = &t3.RuntimeRequest{ID: "r9", Kind: "file-change"}
	s := a.build(time.Now())
	for _, it := range s.Items {
		if it.Kind == ui.ItemApproval {
			t.Fatalf("an approval that came in mid-typing shows: %+v", it.Approval)
		}
	}
	a.handle(th, ui.Action{ID: "composer", Kind: "blur"})
	if s := a.build(time.Now()); s.Items[len(s.Items)-1].Kind != ui.ItemApproval {
		t.Fatalf("after the prompt lost the focus the approval doesn't show: %+v", s.Items)
	}
}

// Stop goes to "Stopping" when clicked, "Stopped" once the run can't be
// stopped, and back once it can, or a new run starts (ComposerPrimaryActions.tsx).
func TestStopPhase(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	_, th := a.thread("swiss/t1")
	th.ActiveRunID = "run1"
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.stopClicked() || a.stopPhase(time.Now()) != ui.StopPending {
		t.Fatal("a click isn't pending")
	}
	if a.stopClicked() {
		t.Fatal("a second click went through while stopping")
	}
	if a.stopPhase(time.Now().Add(11*time.Second)) != ui.StopTimedOut {
		t.Fatal("10s on, it hasn't timed out")
	}
	th.ActiveRunID = ""
	if a.stopPhase(time.Now()) != ui.StopLanded {
		t.Fatal("the run stopped, but Stop hasn't landed")
	}
	th.ActiveRunID = "run2"
	if a.stopPhase(time.Now()) != ui.StopIdle {
		t.Fatal("a new run, but Stop's still stopped")
	}
}

// An answer typed for a question counts once another's on show, so the
// last one's Submit can go (buildPendingUserInputAnswers); a request the
// provider can't answer any more takes none.
func TestTypedAnswers(t *testing.T) {
	a := testApp(t)
	q := func(id string) map[string]any {
		return map[string]any{"id": id, "header": id, "question": id + "?", "options": []map[string]any{{"label": "A"}}}
	}
	openWith(t, a, &t3.RuntimeRequest{ID: "u1", Kind: "user_input"}, []t3.Request{request("u1", "user_input", "pending", "live", false)},
		map[string]any{"id": "i1", "type": "user_input_request", "requestId": "u1", "ordinal": 1, "questions": []any{q("one"), q("two")}})
	th := ui.NewTheme()
	th.SetEditorText("composer", "my own answer")
	a.handle(th, ui.Action{ID: "composer", Kind: "submit", Text: "my own answer"})
	s := a.build(time.Now())
	if s.Question != 1 || !s.Questions[0].Typed || s.Answering {
		t.Fatalf("after the first answer: question %d, typed %v, answering %v", s.Question, s.Questions[0].Typed, s.Answering)
	}
	a.view.requests["u1"] = request("u1", "user_input", "pending", "not_resumable", false)
	if s := a.build(time.Now()); !s.Answering {
		t.Fatal("a request that can't be answered takes answers")
	}
}
