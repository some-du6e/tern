package core

import (
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// T3's browser panel: the thread's tabs on its machine, in the server's own
// headless Chromium, the shown one drawn from the frames it streams. The
// tabs are the thread's in T3 too, and stay open on the server when the
// panel closes. A blank tab lists the dev servers running there.

type browser struct {
	open    bool
	tabs    []t3.PreviewTab // the thread's server tabs
	active  string          // the tab shown
	stream  *t3.PreviewStream
	stopEv  func()
	stopSrv func()
	servers []t3.DevServer
	frame   *ui.Frame
	view    image.Point // the page's size in CSS pixels
	size    image.Point // what the panel asked for
	agent   bool        // the agent is driving the tab
	pending string      // an address typed before the stream was up
	err     string
	gen     int // bumps when the stream restarts, so old ones drop out
	opened  int // bumps when the panel opens, for its event streams
}

// toggleBrowser opens or closes the open thread's browser panel.
func (a *App) toggleBrowser() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.openID == "" {
		return
	}
	b := a.browsers[a.openID]
	if b == nil {
		b = &browser{}
		a.browsers[a.openID] = b
	}
	b.open = !b.open
	if !b.open {
		b.shut()
		return
	}
	e, t := a.thread(a.openID)
	if t == nil || e.conn == nil {
		b.err = "The machine isn't connected."
		return
	}
	b.err = ""
	b.opened++
	opened, conn, threadID := b.opened, e.conn, t.ID
	isOpen := func() bool { return b.open && b.opened == opened }
	go func() {
		tabs, err := serverTabs(conn, threadID)
		if err == nil && len(tabs) == 0 {
			var tab t3.PreviewTab
			tab, err = conn.PreviewOpen(threadID, "")
			tabs = []t3.PreviewTab{tab}
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if !isOpen() {
			return
		}
		if err != nil {
			b.err = sentence(err.Error())
			a.changed()
			return
		}
		b.tabs = tabs
		if !b.hasTab(b.active) {
			b.active = tabs[0].TabID
		}
		a.watchLocked(b, conn)
		// Tabs come and go, and navigate, from T3 and the agent too.
		b.stopEv = conn.PreviewEvents(func(tid, _ string) {
			if tid == threadID {
				go a.refreshTabs(b, conn, threadID, opened)
			}
		}, func(error) {})
		b.stopSrv = conn.DevServers(func(list []t3.DevServer) {
			a.mu.Lock()
			if isOpen() {
				b.servers = list
			}
			a.mu.Unlock()
			a.changed()
		}, func(error) {})
	}()
}

// serverTabs is the thread's tabs in the server's browser.
func serverTabs(conn *t3.Conn, threadID string) ([]t3.PreviewTab, error) {
	all, err := conn.PreviewTabs(threadID)
	var tabs []t3.PreviewTab
	for _, t := range all {
		if t.Runtime == "server" {
			tabs = append(tabs, t)
		}
	}
	return tabs, err
}

func (b *browser) hasTab(id string) bool {
	for _, t := range b.tabs {
		if t.TabID == id {
			return true
		}
	}
	return false
}

// refreshTabs reloads the tab list after a preview event. A shown tab that
// closed elsewhere gives way to the first one left.
func (a *App) refreshTabs(b *browser, conn *t3.Conn, threadID string, opened int) {
	tabs, err := serverTabs(conn, threadID)
	if err != nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !b.open || b.opened != opened {
		return
	}
	b.tabs = tabs
	if !b.hasTab(b.active) {
		b.active = ""
		if len(tabs) > 0 {
			b.active = tabs[0].TabID
		}
		a.watchLocked(b, conn)
	}
	a.changed()
}

// watchLocked (re)starts the stream of the shown tab.
func (a *App) watchLocked(b *browser, conn *t3.Conn) {
	b.gen++
	if b.stream != nil {
		go b.stream.Close()
	}
	b.stream, b.frame, b.view, b.agent, b.pending = nil, nil, image.Point{}, false, ""
	gen, size := b.gen, b.size
	var tab t3.PreviewTab
	for _, t := range b.tabs {
		if t.TabID == b.active {
			tab = t
		}
	}
	if tab.TabID == "" {
		return
	}
	if size == (image.Point{}) {
		size = image.Pt(1280, 800)
	}
	live := func() bool { return b.gen == gen && b.open }
	go func() {
		stream, err := conn.OpenPreviewStream(tab.ThreadID, tab.TabID, size.X*2, size.Y*2,
			func(img image.Image) {
				f := ui.NewFrame(img)
				a.mu.Lock()
				if live() {
					b.frame = &f
				}
				a.mu.Unlock()
				a.changed()
			},
			func(m t3.PreviewMessage) {
				if m.Type == "clipboard" {
					// The page copied: ⌘C in the panel.
					text := m.Text
					a.onUI(func(th *ui.Theme) { th.WriteClipboard(text) })
					return
				}
				a.mu.Lock()
				switch m.Type {
				case "viewport":
					b.view = image.Pt(m.Width, m.Height)
				case "control":
					b.agent = m.Controller == "agent"
				}
				a.mu.Unlock()
				a.changed()
			},
			func(err error) {
				a.mu.Lock()
				if live() {
					b.err = "The browser stream ended. Close the panel and open it again."
				}
				a.mu.Unlock()
				a.changed()
			})
		a.mu.Lock()
		defer a.mu.Unlock()
		if err != nil {
			if live() {
				b.err = sentence(err.Error())
			}
			a.changed()
			return
		}
		if !live() {
			go stream.Close()
			return
		}
		b.stream, b.err = stream, ""
		if b.size != (image.Point{}) {
			go stream.Send(map[string]any{"type": "resize", "width": b.size.X, "height": b.size.Y})
		}
		if u := b.pending; u != "" {
			b.pending = ""
			go stream.Send(map[string]any{"type": "navigate", "url": u})
		}
		a.changed()
	}()
}

// browserTab acts on the open thread's tabs: "new", "show:i", "close:i".
func (a *App) browserTab(verb string, i int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	b := a.browsers[a.openID]
	e, t := a.thread(a.openID)
	if b == nil || !b.open || t == nil || e.conn == nil {
		return
	}
	conn, threadID, opened := e.conn, t.ID, b.opened
	switch verb {
	case "new":
		go func() {
			tab, err := conn.PreviewOpen(threadID, "")
			a.mu.Lock()
			defer a.mu.Unlock()
			if err != nil {
				a.fail(err)
				return
			}
			if b.open && b.opened == opened {
				if !b.hasTab(tab.TabID) {
					b.tabs = append(b.tabs, tab)
				}
				b.active = tab.TabID
				a.watchLocked(b, conn)
				a.changed()
			}
		}()
		a.onUIFocus("browser.url")
	case "show":
		if i >= 0 && i < len(b.tabs) && b.tabs[i].TabID != b.active {
			b.active = b.tabs[i].TabID
			a.watchLocked(b, conn)
		}
	case "close":
		if i < 0 || i >= len(b.tabs) {
			return
		}
		id := b.tabs[i].TabID
		b.tabs = append(b.tabs[:i:i], b.tabs[i+1:]...)
		go conn.PreviewClose(threadID, id)
		if len(b.tabs) == 0 {
			b.open = false
			b.shut()
			return
		}
		if id == b.active {
			b.active = b.tabs[min(i, len(b.tabs)-1)].TabID
			a.watchLocked(b, conn)
		}
	}
}

// onUIFocus focuses a field at the next frame. Safe with the lock held.
func (a *App) onUIFocus(id string) {
	a.uiOps = append(a.uiOps, func(th *ui.Theme) { th.Focus(id) })
}

// shut stops watching. Call with the lock held.
func (b *browser) shut() {
	b.gen++
	b.opened++
	if b.stream != nil {
		go b.stream.Close()
	}
	for _, stop := range []func(){b.stopEv, b.stopSrv} {
		if stop != nil {
			stop()
		}
	}
	b.stream, b.stopEv, b.stopSrv, b.frame, b.servers = nil, nil, nil, nil, nil
}

// browserView is the open thread's panel for the scene. Call with the
// lock held.
func (a *App) browserView() *ui.BrowserView {
	b := a.browsers[a.openID]
	if b == nil || !b.open {
		return nil
	}
	v := &ui.BrowserView{Frame: b.frame, Viewport: b.view, Agent: b.agent, Error: b.err, Loading: b.stream == nil && b.err == ""}
	for i, t := range b.tabs {
		title := t.NavStatus.Title
		if title == "" {
			title = hostOf(t.NavStatus.URL)
		}
		v.Tabs = append(v.Tabs, ui.BrowserTab{Title: title, Loading: t.NavStatus.Tag == "Loading"})
		if t.TabID == b.active {
			v.Active = i
			nav := t.NavStatus
			v.URL, v.Title, v.CanBack, v.CanForward = nav.URL, nav.Title, t.CanGoBack, t.CanGoForward
			v.Loading = v.Loading || nav.Tag == "Loading"
			v.Failed = nav.Tag == "LoadFailed"
		}
	}
	if v.URL == "about:blank" {
		v.URL = ""
	}
	for _, s := range b.servers {
		label := s.ProcessName
		if label == "" {
			label = "Port " + fmt.Sprint(s.Port)
		}
		v.Servers = append(v.Servers, ui.DevServer{Label: label, URL: s.URL})
	}
	stream := b.stream
	v.Send = func(msg map[string]any) {
		if stream != nil {
			go stream.Send(msg)
		}
	}
	v.Resize = func(w, h int) {
		a.mu.Lock()
		same := b.size == image.Pt(w, h)
		b.size = image.Pt(w, h)
		a.mu.Unlock()
		if !same && stream != nil {
			go stream.Send(map[string]any{"type": "resize", "width": w, "height": h})
		}
	}
	return v
}

// hostOf is a URL's host, for a tab without a title.
func hostOf(u string) string {
	switch {
	case u == "" || u == "about:blank":
		return "New tab"
	case strings.Contains(u, "://"):
		u = u[strings.Index(u, "://")+3:]
	}
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	return u
}

// browserGo navigates the open thread's tab to what was typed in its
// address bar.
func (a *App) browserGo(typed string) {
	u := previewURL(typed)
	if u == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	b := a.browsers[a.openID]
	e, t := a.thread(a.openID)
	if b == nil || t == nil || e.conn == nil || b.active == "" {
		a.fail(errors.New("the browser isn't connected"))
		a.changed()
		return
	}
	for i := range b.tabs {
		if b.tabs[i].TabID == b.active {
			b.tabs[i].NavStatus.URL, b.tabs[i].NavStatus.Tag = u, "Loading"
		}
	}
	// The stream's viewer has control of the tab, so the address goes that
	// way; a tab whose stream is still starting gets it once it's up.
	if b.stream != nil {
		go b.stream.Send(map[string]any{"type": "navigate", "url": u})
	} else {
		b.pending = u
	}
	a.changed()
}

// previewURL is what an address bar makes of typed text: a URL as is,
// localhost and ports over http, anything else with a dot over https.
func previewURL(s string) string {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return ""
	case strings.Contains(s, "://"), strings.HasPrefix(s, "about:"):
		return s
	case strings.HasPrefix(s, "localhost"), strings.HasPrefix(s, "127."), strings.HasPrefix(s, "0.0.0.0"), strings.HasPrefix(s, ":"):
		if strings.HasPrefix(s, ":") {
			s = "localhost" + s
		}
		return "http://" + s
	case !strings.Contains(s, ".") && !strings.Contains(s, "/"):
		if _, err := parsePort(s); err == nil {
			return "http://localhost:" + s
		}
		return "https://www.google.com/search?q=" + strings.ReplaceAll(s, " ", "+")
	}
	return "https://" + s
}

func parsePort(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a port")
		}
		n = n*10 + int(r-'0')
	}
	if n == 0 || n > 65535 {
		return 0, errors.New("not a port")
	}
	return n, nil
}

// forgetBrowsers stops every panel's stream, on sign-out.
func (a *App) forgetBrowsers() {
	for _, b := range a.browsers {
		b.shut()
	}
	a.browsers = map[string]*browser{}
}
