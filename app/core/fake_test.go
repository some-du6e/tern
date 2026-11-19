package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// fakeEnv is an environment's RPC socket for tests: a local server that
// speaks T3's Effect RPC framing, records every request and answers each
// as the test says. Nothing reaches a real server.
type fakeEnv struct {
	t    *testing.T
	srv  *httptest.Server
	mu   sync.Mutex
	reqs []fakeReq
	ws   *websocket.Conn
	// answer replies to a request by tag; requests without one stay open.
	answer map[string]func(f *fakeEnv, r fakeReq)
}

type fakeReq struct {
	ID, Tag string
	Payload map[string]any
}

func newFakeEnv(t *testing.T) *fakeEnv {
	t.Helper()
	f := &fakeEnv{t: t, answer: map[string]func(*fakeEnv, fakeReq){}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/websocket-ticket/rpc", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ticket":"test"}`))
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		f.mu.Lock()
		f.ws = ws
		f.mu.Unlock()
		for {
			_, data, err := ws.Read(context.Background())
			if err != nil {
				return
			}
			var fr struct {
				Tag     string          `json:"_tag"`
				ID      string          `json:"id"`
				RPC     string          `json:"tag"`
				Payload json.RawMessage `json:"payload"`
			}
			if json.Unmarshal(data, &fr) != nil || fr.Tag != "Request" {
				continue
			}
			req := fakeReq{ID: fr.ID, Tag: fr.RPC}
			_ = json.Unmarshal(fr.Payload, &req.Payload)
			f.mu.Lock()
			f.reqs = append(f.reqs, req)
			answer := f.answer[req.Tag]
			f.mu.Unlock()
			if answer != nil {
				answer(f, req)
			}
		}
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// dial connects a's key to the fake as a paired (bearer) session.
func (f *fakeEnv) dial(a *App) *t3.Conn {
	f.t.Helper()
	sess := &t3.Session{EnvironmentID: "swiss", HTTPBaseURL: f.srv.URL, WSBaseURL: "ws" + strings.TrimPrefix(f.srv.URL, "http"),
		AccessToken: "test", ExpiresAt: time.Now().Add(time.Hour), Bearer: true}
	conn, err := t3.Dial(context.Background(), a.key, sess)
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(conn.Close)
	return conn
}

func (f *fakeEnv) send(v any) {
	raw, _ := json.Marshal(v)
	f.mu.Lock()
	ws := f.ws
	f.mu.Unlock()
	_ = ws.Write(context.Background(), websocket.MessageText, raw)
}

// chunk streams values on request r.
func (f *fakeEnv) chunk(r fakeReq, values ...any) {
	f.send(map[string]any{"_tag": "Chunk", "requestId": r.ID, "values": values})
}

// exit ends request r with value.
func (f *fakeEnv) exit(r fakeReq, value any) {
	f.send(map[string]any{"_tag": "Exit", "requestId": r.ID, "exit": map[string]any{"_tag": "Success", "value": value}})
}

// failWith ends request r with a server error.
func (f *fakeEnv) failWith(r fakeReq, msg string) {
	f.send(map[string]any{"_tag": "Exit", "requestId": r.ID, "exit": map[string]any{"_tag": "Failure",
		"cause": []any{map[string]any{"_tag": "Fail", "error": map[string]any{"_tag": "GitCommandError", "message": msg}}}}})
}

// on answers every request for tag with do.
func (f *fakeEnv) on(tag string, do func(f *fakeEnv, r fakeReq)) {
	f.mu.Lock()
	f.answer[tag] = do
	f.mu.Unlock()
}

// requests are the requests made for tag so far. A tag of
// "orchestration.dispatchCommand:<type>" picks the commands of that type.
func (f *fakeEnv) requests(tag string) []fakeReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	tag, kind, _ := strings.Cut(tag, ":")
	var out []fakeReq
	for _, r := range f.reqs {
		if r.Tag == tag && (kind == "" || r.Payload["type"] == kind) {
			out = append(out, r)
		}
	}
	return out
}

// wait is the n-th request for tag, waiting up to five seconds for it.
func (f *fakeEnv) wait(tag string, n int) fakeReq {
	f.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if rs := f.requests(tag); len(rs) >= n {
			return rs[n-1]
		}
		time.Sleep(5 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t.Fatalf("no request %d for %s; got %+v", n, tag, f.reqs)
	return fakeReq{}
}

// built is the scene for now, built with the lock held as Scene builds it:
// the fake's streams change the app from their own goroutine.
func built(a *App, now time.Time) *ui.Scene {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.build(now)
}

// eventually fails unless ok turns true within five seconds.
func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("never: %s", what)
}
