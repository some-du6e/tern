package core

import (
	"sort"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The open thread's runtime requests as the composer and the timeline
// answer them (packages/client-runtime/src/state/threadRequests.ts): its
// pending approvals, and the typing guard that holds back a request which
// arrives while the user types (composerTypingGuard.ts).

// approval is a pending approval as T3 derives it
// (derivePendingThreadRequests): its request, whether the provider can
// still take an answer and whether it takes feedback with a decline. The
// timeline shows it at its approval_request row.
type approval struct {
	id, kind, created string
	live, feedback    bool
}

// notApprovals are the request kinds that are never an approval.
var notApprovals = map[string]bool{"user_input": true, "auth_refresh": true, "dynamic_tool_call": true}

// pendingApprovals is t's approvals waiting on the user, oldest first:
// the projection's pending requests once the view has them, else the
// shell's one pending request. Call with the lock held.
func pendingApprovals(t *t3.Thread, v *threadView) []approval {
	var out []approval
	switch {
	case v != nil && v.requests != nil:
		for _, r := range v.requests {
			if r.Status != "pending" || notApprovals[r.Kind] {
				continue
			}
			live := r.ResponseCapability.Type == "live"
			out = append(out, approval{id: r.ID, kind: r.Kind, created: r.CreatedAt, live: live, feedback: live && r.ResponseCapability.SupportsFeedback})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].created != out[j].created {
				return out[i].created < out[j].created
			}
			return out[i].id < out[j].id
		})
	case t != nil && t.PendingRuntimeRequest != nil && !notApprovals[t.PendingRuntimeRequest.Kind]:
		out = append(out, approval{id: t.PendingRuntimeRequest.ID, kind: t.PendingRuntimeRequest.Kind, live: true})
	}
	return out
}

// shownApprovals is the open thread's pending approvals the user can see:
// those the typing guard isn't holding back. Call with the lock held.
func (a *App) shownApprovals() []approval {
	t := a.curThread()
	if a.draft != "" || t == nil {
		return nil
	}
	var out []approval
	for _, ap := range pendingApprovals(t, a.view) {
		if !a.guard.held[ap.id] {
			out = append(out, ap)
		}
	}
	return out
}

// redirect is what a send does to the approvals it shows
// (useComposerSend.ts): when the first takes feedback, the message goes
// to it as its decline; else the send declines each one the provider can
// still answer and goes out as a message.
func redirect(shown []approval) (feedback string, decline []string) {
	if len(shown) > 0 && shown[0].feedback {
		return shown[0].id, nil
	}
	for _, ap := range shown {
		if ap.live {
			decline = append(decline, ap.id)
		}
	}
	return "", decline
}

// respondApproval answers approval req with decision, if it's shown and
// the provider can still take an answer (onRespondToApproval). Its row's
// answers are off until the command lands.
func (a *App) respondApproval(req, decision string) {
	a.mu.Lock()
	ok := false
	for _, ap := range a.shownApprovals() {
		ok = ok || (ap.id == req && ap.live && !a.comp.responding[req])
	}
	if ok {
		a.comp.responding[req] = true
	}
	a.mu.Unlock()
	if !ok {
		return
	}
	a.dispatchOpenThen("runtime-request.respond", map[string]any{"requestId": req, "decision": decision}, func() {
		delete(a.comp.responding, req)
	})
}

// stopWait is how long Stop stays off waiting for the run to stop
// (STOP_PENDING_MAX_MS).
const stopWait = 10 * time.Second

// stopState is where a click on Stop has got, for one thread's run.
type stopState struct {
	thread, run string
	phase       ui.StopPhase
	at          time.Time
}

// stopClicked notes a click on Stop: it's off, "Stopping…", until the run
// stops or 10s pass. Call with the lock held.
func (a *App) stopClicked() bool {
	t := a.curThread()
	if t == nil || a.stopping.phase == ui.StopPending || a.stopping.phase == ui.StopLanded {
		return false
	}
	a.stopping = stopState{thread: a.openID, run: t.ActiveRunID, phase: ui.StopPending, at: time.Now()}
	time.AfterFunc(stopWait, a.changed)
	return true
}

// stopPhase moves the open thread's stop along (ComposerPrimaryActions.tsx):
// pending or timed out becomes landed once the run can't be stopped; landed
// goes back to idle once it can again, as does a new run; pending times out
// after 10s. Call with the lock held.
func (a *App) stopPhase(now time.Time) ui.StopPhase {
	t := a.curThread()
	if t == nil || a.stopping.thread != a.openID {
		return ui.StopIdle
	}
	running := threadState(t) == ui.ThreadRunning
	switch st := &a.stopping; {
	case t.ActiveRunID != "" && st.run != "" && t.ActiveRunID != st.run:
		*st = stopState{}
	case (st.phase == ui.StopPending || st.phase == ui.StopTimedOut) && !running:
		st.phase = ui.StopLanded
	case st.phase == ui.StopLanded && running:
		*st = stopState{}
	case st.phase == ui.StopPending && now.Sub(st.at) >= stopWait:
		st.phase = ui.StopTimedOut
	}
	return a.stopping.phase
}

// typingIdle is how long after the last keystroke the user counts as
// done typing (COMPOSER_TYPING_IDLE_MS).
const typingIdle = 1500 * time.Millisecond

// typingGuard is composerTypingGuard.ts: a request that arrives while the
// prompt has the focus and a key went in under 1.5s ago is held back from
// the timeline and the composer until the user stops typing, leaves the
// prompt or sends, so it's never answered by text typed for something
// else.
type typingGuard struct {
	key        string // the composer it guards
	seen, held map[string]bool
	focused    bool
	typed      time.Time // the last keystroke
}

// requests notes the requests pending in composer key now: one that's
// new while the user types is held.
func (g *typingGuard) requests(key string, ids []string, now time.Time) {
	if g.seen == nil || g.key != key {
		g.key, g.seen, g.held, g.typed = key, map[string]bool{}, map[string]bool{}, time.Time{}
	}
	pending := map[string]bool{}
	for _, id := range ids {
		pending[id] = true
	}
	for id := range g.held {
		if !pending[id] {
			delete(g.held, id)
		}
	}
	for _, id := range ids {
		if !g.seen[id] && g.focused && now.Sub(g.typed) < typingIdle {
			g.held[id] = true
		}
		g.seen[id] = true
	}
}

// release shows what's held: the user stopped, left the prompt or sent.
func (g *typingGuard) release() {
	clear(g.held)
	g.typed = time.Time{}
}

// typed is a keystroke in the prompt: once 1.5s pass without another, the
// guard lets go. Call with the lock held.
func (a *App) typed() {
	now := time.Now()
	a.guard.typed = now
	time.AfterFunc(typingIdle, func() {
		a.mu.Lock()
		idle := a.guard.typed.Equal(now)
		if idle {
			a.guard.release()
		}
		a.mu.Unlock()
		if idle {
			a.changed()
		}
	})
}

// guardRequests runs the typing guard over the open thread's pending
// requests. Call with the lock held, before anything reads what it holds.
func (a *App) guardRequests(now time.Time) {
	var ids []string
	if t := a.curThread(); a.draft == "" && t != nil {
		for _, ap := range pendingApprovals(t, a.view) {
			ids = append(ids, ap.id)
		}
		if req := t.PendingRuntimeRequest; req != nil && req.Kind == "user_input" {
			ids = append(ids, req.ID)
		}
	}
	a.guard.requests(a.composerKey(), ids, now)
}
