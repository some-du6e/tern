package core

import (
	"encoding/json"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The provider's guess at the open thread's next prompt
// (usePromptSuggestion.ts): the server streams it while the thread is
// open, when its settings turn suggestions on, on whichever socket is up,
// so a reconnect subscribes again. The composer offers it as its
// placeholder while the run it came from is the thread's latest and
// completed, until Tab takes it or Escape dismisses it.

// suggestion is the open thread's latest guess and its stream, on conn
// (nil once that stream ended).
type suggestion struct {
	ref, run, text string
	conn           *t3.Conn
	cancel         func()
}

// suggest subscribes to thread r's prompt suggestions on conn when its
// server offers them, unless that stream is already up. Call without the
// lock.
func (a *App) suggest(e *envState, conn *t3.Conn, r string) {
	a.mu.Lock()
	on := e.config != nil && e.config.Settings.PromptSuggestionsEnabled && a.openID == r && (a.suggestion.ref != r || a.suggestion.conn != conn)
	if on {
		if a.suggestion.cancel != nil {
			a.suggestion.cancel()
		}
		if a.suggestion.ref != r {
			a.suggestion = suggestion{ref: r}
		}
		a.suggestion.conn, a.suggestion.cancel = conn, nil
	}
	a.mu.Unlock()
	if !on {
		return
	}
	_, id := splitRef(r)
	cancel := conn.Stream("orchestration.subscribePromptSuggestion", map[string]any{"threadId": id}, func(raw json.RawMessage) {
		var v *struct {
			RunID string `json:"runId"`
			Text  string `json:"text"`
		}
		if json.Unmarshal(raw, &v) != nil {
			return
		}
		a.mu.Lock()
		if a.suggestion.ref == r {
			a.suggestion.run, a.suggestion.text = "", ""
			if v != nil {
				a.suggestion.run, a.suggestion.text = v.RunID, v.Text
			}
		}
		a.mu.Unlock()
		a.changed()
	}, func(error) {
		// The socket went: the next one subscribes again (subscribeThread,
		// loadConfig).
		a.mu.Lock()
		if a.suggestion.ref == r && a.suggestion.conn == conn {
			a.suggestion.conn, a.suggestion.cancel = nil, nil
		}
		a.mu.Unlock()
	})
	a.mu.Lock()
	if a.suggestion.ref == r && a.suggestion.conn == conn {
		a.suggestion.cancel = cancel
	} else {
		cancel()
	}
	a.mu.Unlock()
}

// dropSuggestion ends the open thread's suggestion stream. Call with the
// lock held.
func (a *App) dropSuggestion() {
	if a.suggestion.cancel != nil {
		a.suggestion.cancel()
	}
	a.suggestion = suggestion{}
}

// offeredSuggestion is the suggestion the composer shows, if any: its run
// is the thread's latest to have run, and completed (offeredPromptSuggestion),
// it isn't dismissed, and the composer is idle and empty of everything but
// typing: no pictures, no approval or question, no armed /clear, a
// provider to send to (blocked says otherwise). Call with the lock held.
func (a *App) offeredSuggestion(t *t3.Thread, v *threadView, blocked bool) string {
	sg := a.suggestion
	if blocked || t == nil || v == nil || sg.ref != a.openID || sg.text == "" || a.dismissed[a.openID+"\x00"+sg.run] {
		return ""
	}
	if threadState(t) != ui.ThreadIdle && threadState(t) != ui.ThreadError || a.comp.sending || len(a.images[a.composerKey()]) > 0 {
		return ""
	}
	run := v.runs[t.LatestRunID]
	if run.Status == "cancelled" && run.StartedAt == "" {
		// A message promoted to steer leaves its queued run cancelled
		// before it started: the run before it is the one that ran.
		run = t3.Run{}
		for _, r := range v.runs {
			if r.StartedAt != "" && r.StartedAt > run.StartedAt {
				run = r
			}
		}
	}
	if run.ID != sg.run || run.Status != "completed" {
		return ""
	}
	return sg.text
}

// suggestionKey is Tab or Escape on an empty prompt showing a suggestion:
// Tab puts it in the prompt, Escape dismisses it for its run.
func (a *App) suggestionKey(th *ui.Theme, take bool) {
	a.mu.Lock()
	sg := a.suggestion
	if !take {
		a.dismissed[a.openID+"\x00"+sg.run] = true
	}
	a.mu.Unlock()
	if take && sg.text != "" {
		th.SetEditorText("composer", sg.text)
		th.Focus("composer")
	}
	a.changed()
}
