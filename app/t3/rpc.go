package t3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

	wmu sync.Mutex // websocket writes are not concurrent-safe
}

type pending struct {
	chunk func([]json.RawMessage)
	done  func(json.RawMessage, error)
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
	c.mu.Unlock()
	c.stop()
	_ = c.ws.Close(websocket.StatusNormalClosure, "")
	close(c.done)
	for _, p := range ps {
		p.done(nil, ErrClosed)
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
	c.mu.Unlock()
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
	c.mu.Unlock()
	c.send(map[string]any{"_tag": "Request", "id": strconv.FormatInt(id, 10), "tag": tag, "payload": payload, "headers": []any{}})
	return id, nil
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
