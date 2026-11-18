package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/some-du6e/tern/app/t3"
)

// What the timeline hands to the system: a markdown link to the browser,
// a proposed plan or a sent file to the Downloads folder.

// openExternal opens a web address in the system's browser, as T3's
// desktop app hands links to shell.openExternal. Tests replace it.
var openExternal = func(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// openLink follows a markdown link's click. T3's links open in the system
// browser by default (browserLinkTarget "system"), and its desktop app
// lets only http and https out (parseSafeExternalUrl): anything else
// stays where it is.
func (a *App) openLink(href string) {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return
	}
	if err := openExternal(u.String()); err != nil {
		a.mu.Lock()
		a.fail(err)
		a.mu.Unlock()
		a.changed()
	}
}

// download saves data where a browser's download would land, the
// Downloads folder, under name; a name that's taken gets " (1)", " (2)"
// and on before its extension, as a browser's does.
func (a *App) download(name string, data []byte) {
	name = filepath.Base(name)
	home, err := os.UserHomeDir()
	if err == nil && (name == "." || name == string(filepath.Separator)) {
		err = errors.New("the download has no file name")
	}
	if err == nil {
		dir := filepath.Join(home, "Downloads")
		if err = os.MkdirAll(dir, 0o755); err == nil {
			err = writeNew(dir, name, data)
		}
	}
	if err != nil {
		a.mu.Lock()
		a.fail(err)
		a.mu.Unlock()
		a.changed()
	}
}

// downloadAttachment saves a file sent in the open thread, by attachment
// id, under its own name.
func (a *App) downloadAttachment(id string) {
	a.mu.Lock()
	v := a.view
	name := ""
	var conn *t3.Conn
	if v != nil {
		conn = v.env.conn
		for _, it := range v.items {
			for _, raw := range it.Attachments {
				var at t3.Attachment
				if json.Unmarshal(raw, &at) == nil && at.ID == id {
					name = at.Name
				}
			}
		}
	}
	a.mu.Unlock()
	if conn == nil || name == "" {
		return
	}
	go func() {
		data, err := conn.FetchAttachment(id)
		if err != nil {
			a.mu.Lock()
			a.fail(err)
			a.mu.Unlock()
			a.changed()
			return
		}
		a.download(name, data)
	}()
}

// writeNew writes data to a file of that name in dir that wasn't there.
func writeNew(dir, name string, data []byte) error {
	ext := filepath.Ext(name)
	for i := 0; ; i++ {
		n := name
		if i > 0 {
			n = fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(name, ext), i, ext)
		}
		f, err := os.OpenFile(filepath.Join(dir, n), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	}
}
