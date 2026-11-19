package t3

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Preview tabs on the server's own headless Chromium, the runtime any
// client can show (docs/integrations.md §6). The page streams as JPEG
// frames; input goes back as JSON.

// PreviewTab is a tab's state.
type PreviewTab struct {
	ThreadID  string `json:"threadId"`
	TabID     string `json:"tabId"`
	NavStatus struct {
		Tag   string `json:"_tag"` // Idle, Loading, Success, LoadFailed
		URL   string `json:"url"`
		Title string `json:"title"`
	} `json:"navStatus"`
	CanGoBack    bool   `json:"canGoBack"`
	CanGoForward bool   `json:"canGoForward"`
	Runtime      string `json:"runtime"`
}

// PreviewTabs lists a thread's tabs.
func (c *Conn) PreviewTabs(threadID string) ([]PreviewTab, error) {
	var out struct {
		Sessions []PreviewTab `json:"sessions"`
	}
	err := c.Call("preview.list", map[string]any{"threadId": threadID}, &out)
	return out.Sessions, err
}

// PreviewOpen opens a server tab on the thread, at url if it's set.
func (c *Conn) PreviewOpen(threadID, url string) (PreviewTab, error) {
	p := map[string]any{"threadId": threadID, "runtime": "server"}
	if url != "" {
		p["url"] = url
	}
	var t PreviewTab
	err := c.Call("preview.open", p, &t)
	return t, err
}

// PreviewNavigate sends a tab to url.
func (c *Conn) PreviewNavigate(threadID, tabID, url string) error {
	return c.Call("preview.navigate", map[string]any{"threadId": threadID, "tabId": tabID, "url": url}, nil)
}

// PreviewClose closes a tab.
func (c *Conn) PreviewClose(threadID, tabID string) error {
	return c.Call("preview.close", map[string]any{"threadId": threadID, "tabId": tabID}, nil)
}

// PreviewEvents streams every preview change on the environment: tabs
// opening, navigating, closing. onEvent gets the thread and tab it was on.
func (c *Conn) PreviewEvents(onEvent func(threadID, tabID string), onEnd func(error)) (cancel func()) {
	return c.Stream("subscribePreviewEvents", map[string]any{}, func(v json.RawMessage) {
		var ev struct {
			ThreadID string `json:"threadId"`
			TabID    string `json:"tabId"`
		}
		if json.Unmarshal(v, &ev) == nil {
			onEvent(ev.ThreadID, ev.TabID)
		}
	}, onEnd)
}

// DevServer is a server listening on the environment's machine.
type DevServer struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	URL         string `json:"url"`
	ProcessName string `json:"processName"`
}

// DevServers streams the dev servers found on the machine: the whole list
// each time it changes.
func (c *Conn) DevServers(onList func([]DevServer), onEnd func(error)) (cancel func()) {
	return c.Stream("subscribeDiscoveredLocalServers", map[string]any{}, func(v json.RawMessage) {
		var ev struct {
			Servers []DevServer `json:"servers"`
		}
		if json.Unmarshal(v, &ev) == nil {
			onList(ev.Servers)
		}
	}, onEnd)
}

// PreviewStream is a tab's frames and input.
type PreviewStream struct {
	ws     *websocket.Conn
	ctx    context.Context
	cancel func()
	mu     sync.Mutex // one writer at a time
}

// PreviewMessage is a JSON message from the stream: viewport, control,
// clipboard, popup and the like.
type PreviewMessage struct {
	Type          string `json:"type"`
	Width, Height int
	Controller    string `json:"controller"`
	Text          string `json:"text"`
}

// OpenPreviewStream starts watching a tab, at most maxW by maxH device
// pixels. onFrame gets each decoded frame, onMessage the rest, and onEnd
// why it stopped.
func (c *Conn) OpenPreviewStream(threadID, tabID string, maxW, maxH int, onFrame func(image.Image), onMessage func(PreviewMessage), onEnd func(error)) (*PreviewStream, error) {
	u, err := MediaSocketURL(c.sess, c.key, "/api/preview-stream/ws", map[string]string{
		"threadId": threadID, "tabId": tabID,
		"maxWidth": strconv.Itoa(max(240, maxW)), "maxHeight": strconv.Itoa(max(240, maxH)), "quality": "80",
	})
	if err != nil {
		return nil, err
	}
	dctx, dcancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer dcancel()
	ws, _, err := websocket.Dial(dctx, u, nil)
	if err != nil {
		return nil, err
	}
	ws.SetReadLimit(64 << 20)
	ctx, cancel := context.WithCancel(context.Background())
	p := &PreviewStream{ws: ws, ctx: ctx, cancel: cancel}
	go func() {
		for {
			typ, data, err := ws.Read(ctx)
			if err != nil {
				onEnd(err)
				return
			}
			if typ == websocket.MessageBinary {
				if img, err := jpeg.Decode(bytes.NewReader(data)); err == nil {
					onFrame(img)
				}
				// The next frame only comes after this one's ack.
				p.Send(map[string]any{"type": "ack"})
				continue
			}
			var m PreviewMessage
			if json.Unmarshal(data, &m) == nil {
				onMessage(m)
			}
		}
	}()
	return p, nil
}

// Send writes one input message: mouse, wheel, key, text, resize,
// navigate, history, reload.
func (p *PreviewStream) Send(msg map[string]any) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	_ = p.ws.Write(ctx, websocket.MessageText, b)
}

// Close stops watching. The tab stays open on the server.
func (p *PreviewStream) Close() {
	p.cancel()
	p.ws.Close(websocket.StatusNormalClosure, "")
}
