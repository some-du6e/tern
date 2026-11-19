package t3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// Conn is the Effect RPC socket to one environment: JSON frames, Ack for
// every Chunk, Ping every 5 s, and dead after 15 s with no frame.
type Conn struct {
	sess *Session
	key  *Key
	ws   *websocket.Conn
	ctx  context.Context
	stop context.CancelFunc

	next    atomic.Int64
	seen    atomic.Int64 // unix ms of the last frame
	mu      sync.Mutex
	pending map[int64]*pending
	closed  bool
	err     error
	done    chan struct{}
	onSlow  func() // the set of slow calls changed

	wmu sync.Mutex // websocket writes are not concurrent-safe
}

type pending struct {
	chunk func([]json.RawMessage)
	done  func(json.RawMessage, error)
	// A call the server should answer soon: when it went out, the timer
	// that marks it slow, and whether it has.
	tag   string
	sent  time.Time
	timer *time.Timer
	slow  bool
}

// SlowRequest is a call the server hasn't answered in its threshold.
type SlowRequest struct {
	Tag       string
	StartedAt time.Time
	Threshold time.Duration
}

// T3 warns about a call with no answer after 15 s, or 2 min for the ones
// that wait on installs and the network, and never about subscriptions or
// the background reports nobody waits on (requestLatencyState.ts).
const (
	slowAfter        = 15 * time.Second
	longRunningAfter = 2 * time.Minute
)

var (
	untrackedCalls   = map[string]bool{"server.getUsageSummary": true, "server.reportClientActivity": true, "sourceControl.signInWait": true}
	longRunningCalls = map[string]bool{"server.updateProvider": true, "server.refreshProviders": true, "server.updateServer": true, "vcs.refreshStatus": true}
)

func slowThreshold(tag string) time.Duration {
	if strings.Contains(tag, "subscribe") || strings.HasPrefix(tag, "pullRequests.") || untrackedCalls[tag] {
		return 0
	}
	if longRunningCalls[tag] {
		return longRunningAfter
	}
	return slowAfter
}

// RPCError is a typed failure from the server, like EnvironmentAuthorizationError.
type RPCError struct {
	Tag     string
	Message string
	Raw     json.RawMessage
}

func (e *RPCError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Tag != "" {
		return e.Tag
	}
	return "RPC failed"
}

// ErrClosed ends every call and stream on a socket that went away.
var ErrClosed = errors.New("socket closed")

// Dial mints a ticket and opens the socket.
func Dial(ctx context.Context, k *Key, s *Session) (*Conn, error) {
	u, err := s.socketURL(k)
	if err != nil {
		return nil, err
	}
	dctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(dctx, u, nil)
	if err != nil {
		return nil, fmt.Errorf("WebSocket: %w", err)
	}
	ws.SetReadLimit(256 << 20) // full thread snapshots run to tens of MB
	c := &Conn{sess: s, key: k, ws: ws, pending: map[int64]*pending{}, done: make(chan struct{})}
	c.ctx, c.stop = context.WithCancel(context.Background())
	c.seen.Store(time.Now().UnixMilli())
	go c.readLoop()
	go c.pingLoop()
	return c, nil
}

func (c *Conn) Session() *Session { return c.sess }

// Done closes when the socket is gone; Err says why.
func (c *Conn) Done() <-chan struct{} { return c.done }

func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *Conn) Close() { c.shutdown(errors.New("closed")) }

func (c *Conn) shutdown(err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed, c.err = true, err
	ps := c.pending
	c.pending = map[int64]*pending{}
	slow := false
	for _, p := range ps {
		slow = c.answered(p) || slow
	}
	c.mu.Unlock()
	c.stop()
	_ = c.ws.Close(websocket.StatusNormalClosure, "")
	close(c.done)
	for _, p := range ps {
		p.done(nil, ErrClosed)
	}
	if slow {
		c.slowChanged()
	}
}

func (c *Conn) send(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, raw); err != nil {
		go c.shutdown(err)
	}
}

func (c *Conn) pingLoop() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-t.C:
			// A laptop that slept leaves a socket that never closes. No frame
			// for 15 s means it's dead; the app reconnects.
			if time.Since(time.UnixMilli(c.seen.Load())) > 15*time.Second {
				c.shutdown(errors.New("no pong for 15 s"))
				return
			}
			c.send(map[string]string{"_tag": "Ping"})
		}
	}
}

type frame struct {
	Tag       string            `json:"_tag"`
	RequestID json.RawMessage   `json:"requestId"`
	Values    []json.RawMessage `json:"values"`
	Exit      *struct {
		Tag   string          `json:"_tag"`
		Value json.RawMessage `json:"value"`
		Cause []struct {
			Tag    string          `json:"_tag"`
			Error  json.RawMessage `json:"error"`
			Defect json.RawMessage `json:"defect"`
		} `json:"cause"`
	} `json:"exit"`
	Defect json.RawMessage `json:"defect"`
}

func (c *Conn) readLoop() {
	for {
		_, data, err := c.ws.Read(c.ctx)
		if err != nil {
			c.shutdown(err)
			return
		}
		c.seen.Store(time.Now().UnixMilli())
		var frames []frame
		if len(data) > 0 && data[0] == '[' {
			err = json.Unmarshal(data, &frames)
		} else {
			var f frame
			err = json.Unmarshal(data, &f)
			frames = []frame{f}
		}
		if err != nil {
			continue
		}
		for _, f := range frames {
			c.handle(f)
		}
	}
}

func parseID(raw json.RawMessage) (int64, bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		n, err := strconv.ParseInt(s, 10, 64)
		return n, err == nil
	}
	var n int64
	return n, json.Unmarshal(raw, &n) == nil
}

func (c *Conn) handle(f frame) {
	switch f.Tag {
	case "Pong":
		return
	case "Defect":
		c.shutdown(fmt.Errorf("server defect: %s", f.Defect))
		return
	}
	id, ok := parseID(f.RequestID)
	if !ok {
		return
	}
	c.mu.Lock()
	p := c.pending[id]
	if f.Tag == "Exit" {
		delete(c.pending, id)
	}
	answered := p != nil && c.answered(p)
	c.mu.Unlock()
	if answered {
		c.slowChanged()
	}
	if p == nil {
		return
	}
	switch f.Tag {
	case "Chunk":
		// The server sends nothing more on this stream until we Ack.
		c.send(map[string]any{"_tag": "Ack", "requestId": f.RequestID})
		p.chunk(f.Values)
	case "Exit":
		if f.Exit == nil || f.Exit.Tag == "Success" {
			var v json.RawMessage
			if f.Exit != nil {
				v = f.Exit.Value
			}
			p.done(v, nil)
			return
		}
		e := &RPCError{}
		for i, cause := range f.Exit.Cause {
			if cause.Tag != "Fail" && i > 0 {
				continue
			}
			raw := cause.Error
			if raw == nil {
				raw = cause.Defect
			}
			e.Raw = raw
			var body struct {
				Tag     string `json:"_tag"`
				Message string `json:"message"`
			}
			if json.Unmarshal(raw, &body) == nil {
				e.Tag, e.Message = body.Tag, body.Message
			} else {
				_ = json.Unmarshal(raw, &e.Message)
			}
			if cause.Tag == "Fail" {
				break
			}
		}
		p.done(nil, e)
	}
}

func (c *Conn) request(tag string, payload any, p *pending) (int64, error) {
	if payload == nil {
		payload = struct{}{}
	}
	id := c.next.Add(1) - 1
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return 0, ErrClosed
	}
	c.pending[id] = p
	if after := slowThreshold(tag); after > 0 {
		p.tag, p.sent = tag, time.Now()
		p.timer = time.AfterFunc(after, func() {
			c.mu.Lock()
			slow := c.pending[id] == p && p.timer != nil
			p.slow = slow
			c.mu.Unlock()
			if slow {
				c.slowChanged()
			}
		})
	}
	c.mu.Unlock()
	c.send(map[string]any{"_tag": "Request", "id": strconv.FormatInt(id, 10), "tag": tag, "payload": payload, "headers": []any{}})
	return id, nil
}

// answered stops watching p for slowness, now the server has answered it,
// and reports whether it was slow. Call with c.mu held.
func (c *Conn) answered(p *pending) bool {
	if p.timer == nil {
		return false
	}
	p.timer.Stop()
	p.timer = nil
	slow := p.slow
	p.slow = false
	return slow
}

// WatchSlow calls f, on a goroutine of its own, whenever a call turns slow
// or a slow one is answered.
func (c *Conn) WatchSlow(f func()) {
	c.mu.Lock()
	c.onSlow = f
	c.mu.Unlock()
}

// slowChanged tells the watcher. It never waits for it: the read loop
// calls it, and the watcher may be waiting on this socket.
func (c *Conn) slowChanged() {
	c.mu.Lock()
	f := c.onSlow
	c.mu.Unlock()
	if f != nil {
		go f()
	}
}

// Slow is the calls still waiting past their threshold, oldest first.
func (c *Conn) Slow() []SlowRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []SlowRequest
	for _, p := range c.pending {
		if p.slow {
			out = append(out, SlowRequest{Tag: p.tag, StartedAt: p.sent, Threshold: slowThreshold(p.tag)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}

// Call runs one RPC and decodes its value into out (which may be nil).
func (c *Conn) Call(tag string, payload, out any) error {
	type result struct {
		v   json.RawMessage
		err error
	}
	ch := make(chan result, 1)
	if _, err := c.request(tag, payload, &pending{
		chunk: func([]json.RawMessage) {},
		done:  func(v json.RawMessage, err error) { ch <- result{v, err} },
	}); err != nil {
		return err
	}
	select {
	case r := <-ch:
		if r.err != nil || out == nil || len(r.v) == 0 {
			return r.err
		}
		return json.Unmarshal(r.v, out)
	case <-time.After(60 * time.Second):
		return fmt.Errorf("%s timed out", tag)
	}
}

// Stream calls onValue for each streamed item, then onEnd. The returned
// function cancels it.
func (c *Conn) Stream(tag string, payload any, onValue func(json.RawMessage), onEnd func(error)) (cancel func()) {
	id, err := c.request(tag, payload, &pending{
		chunk: func(vs []json.RawMessage) {
			for _, v := range vs {
				onValue(v)
			}
		},
		done: func(_ json.RawMessage, err error) { onEnd(err) },
	})
	if err != nil {
		onEnd(err)
		return func() {}
	}
	return func() {
		c.mu.Lock()
		_, live := c.pending[id]
		delete(c.pending, id)
		c.mu.Unlock()
		if live {
			c.send(map[string]any{"_tag": "Interrupt", "requestId": strconv.FormatInt(id, 10), "interruptors": []any{}})
		}
	}
}

// Dispatch sends one orchestration command. It adds the commandId.
func (c *Conn) Dispatch(cmd map[string]any) error {
	cmd["commandId"] = newUUID()
	return c.Call("orchestration.dispatchCommand", cmd, nil)
}
