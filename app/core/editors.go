package core

import (
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/some-du6e/tern/app/t3"
)

// Open in editor (OpenInPicker.tsx, remoteOpen.ts, contracts/src/editor.ts).
// An environment on this computer opens its folder with shell.openInEditor;
// one elsewhere opens it here, in an editor that reaches the machine over
// SSH, through the editor's URL scheme.

// editor is one of T3's editors: the commands that find it and, for the
// ones that open remote folders, their URL scheme (JetBrains IDEs share
// Gateway's, with a product code).
type editor struct {
	id, label string
	commands  []string
	scheme    string
	jetbrains string
}

// editors are in the order T3's menu lists them.
var editors = []editor{
	{"cursor", "Cursor", []string{"cursor"}, "cursor", ""},
	{"trae", "Trae", []string{"trae"}, "", ""},
	{"kiro", "Kiro", []string{"kiro"}, "", ""},
	{"vscode", "VS Code", []string{"code"}, "vscode", ""},
	{"vscode-insiders", "VS Code Insiders", []string{"code-insiders"}, "vscode-insiders", ""},
	{"vscodium", "VSCodium", []string{"codium"}, "vscodium", ""},
	{"zed", "Zed", []string{"zed", "zeditor"}, "zed", ""},
	{"antigravity", "Antigravity", []string{"antigravity-ide", "agy-ide"}, "", ""},
	{"idea", "IntelliJ IDEA", []string{"idea"}, "jetbrains", "IU"},
	{"aqua", "Aqua", []string{"aqua"}, "jetbrains", "QA"},
	{"clion", "CLion", []string{"clion"}, "jetbrains", "CL"},
	{"datagrip", "DataGrip", []string{"datagrip"}, "jetbrains", "DB"},
	{"dataspell", "DataSpell", []string{"dataspell"}, "jetbrains", "DS"},
	{"goland", "GoLand", []string{"goland"}, "jetbrains", "GO"},
	{"phpstorm", "PhpStorm", []string{"phpstorm"}, "jetbrains", "PS"},
	{"pycharm", "PyCharm", []string{"pycharm"}, "jetbrains", "PY"},
	{"rider", "Rider", []string{"rider"}, "jetbrains", "RD"},
	{"rubymine", "RubyMine", []string{"rubymine"}, "jetbrains", "RM"},
	{"rustrover", "RustRover", []string{"rustrover"}, "jetbrains", "RR"},
	{"webstorm", "WebStorm", []string{"webstorm"}, "jetbrains", "WS"},
	{"file-manager", "File Manager", nil, "", ""},
}

func editorByID(id string) *editor {
	for i := range editors {
		if editors[i].id == id {
			return &editors[i]
		}
	}
	return nil
}

// editorLabel is what the menu calls an editor; the file manager is this
// computer's own.
func editorLabel(id string) string {
	if id == "file-manager" {
		switch runtime.GOOS {
		case "darwin":
			return "Finder"
		case "windows":
			return "File Explorer"
		}
		return "Files"
	}
	if e := editorByID(id); e != nil {
		return e.label
	}
	return "Editor"
}

// openMode is how an environment's folder opens: local (shell.openInEditor
// on its own machine), remote (a URL to a host it advertises), or not at
// all.
type openMode int

const (
	openLocal openMode = iota
	openRemote
	openNone
)

// remoteOpen is how e's folders open from here, and the host a remote
// editor reaches it at: an environment on this computer (a loopback
// address) opens them itself, else the first host it advertises.
func remoteOpen(e *envState) (openMode, t3.RemoteOpenTarget) {
	if e.conn != nil {
		if u, err := url.Parse(e.conn.Session().HTTPBaseURL); err == nil {
			host := u.Hostname()
			if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
				return openLocal, t3.RemoteOpenTarget{}
			}
		}
	}
	if e.config != nil && len(e.config.RemoteOpenTargets) > 0 {
		return openRemote, e.config.RemoteOpenTargets[0]
	}
	return openNone, t3.RemoteOpenTarget{}
}

var remoteEditorsOnce sync.Once
var remoteEditorList []string

// probeRemoteEditors finds the editors on this computer that open remote
// folders by their commands, as T3's desktop app probes them; without any,
// VS Code, as T3's browser assumes.
func probeRemoteEditors() []string {
	remoteEditorsOnce.Do(func() {
		for _, e := range editors {
			if e.scheme == "" {
				continue
			}
			for _, c := range e.commands {
				if _, err := exec.LookPath(c); err == nil {
					remoteEditorList = append(remoteEditorList, e.id)
					break
				}
			}
		}
		if len(remoteEditorList) == 0 {
			remoteEditorList = []string{"vscode"}
		}
	})
	return remoteEditorList
}

// editorChoices are the editors e's folders can open in from here, in
// T3's order.
func (a *App) editorChoices(e *envState, mode openMode) []string {
	have := map[string]bool{}
	if mode == openLocal {
		if e.config != nil {
			for _, id := range e.config.AvailableEditors {
				have[id] = true
			}
		}
	} else {
		for _, id := range a.remoteEditors() {
			have[id] = true
		}
	}
	var out []string
	for _, ed := range editors {
		if have[ed.id] {
			out = append(out, ed.id)
		}
	}
	return out
}

// preferredEditor is the editor last picked when it's still there, else
// the first there is.
func preferredEditor(choices []string, last string) string {
	for _, id := range choices {
		if id == last {
			return id
		}
	}
	if len(choices) > 0 {
		return choices[0]
	}
	return ""
}

// encodeURIComponent is JavaScript's: everything but letters, digits and
// -_.!~*'() is percent-encoded as UTF-8.
func encodeURIComponent(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', strings.IndexByte("-_.!~*'()", c) >= 0:
			b.WriteByte(c)
		default:
			b.WriteString("%" + strings.ToUpper(hex.EncodeToString([]byte{c})))
		}
	}
	return b.String()
}

var windowsDrive = regexp.MustCompile(`^[Cc]:[\\/]`)

// remoteOpenURL is the URL that opens path on host in editor id
// (buildRemoteOpenUrl), or "" when it can't open remote folders.
func remoteOpenURL(id, host, user, path string) string {
	ed := editorByID(id)
	if ed == nil || ed.scheme == "" {
		return ""
	}
	if ed.jetbrains != "" {
		q := "h=" + url.QueryEscape(host) + "&launchIde=true&ideHint=" + ed.jetbrains + "&projectHint=" + url.QueryEscape(strings.ReplaceAll(path, `\`, "/"))
		if user != "" {
			q += "&u=" + url.QueryEscape(user)
		}
		return "jetbrains://gateway/ssh/environment?" + q
	}
	posix := strings.ReplaceAll(path, `\`, "/")
	rooted := posix
	if !strings.HasPrefix(rooted, "/") {
		rooted = "/" + rooted
	}
	encode := func(p string) string {
		parts := strings.Split(p, "/")
		for i := range parts {
			parts[i] = encodeURIComponent(parts[i])
		}
		return strings.Join(parts, "/")
	}
	if id == "zed" {
		// Zed roots paths on the system drive, so the drive goes.
		zp := rooted
		if windowsDrive.MatchString(path) {
			zp = rooted[3:]
		}
		dest := encodeURIComponent(host)
		if user != "" {
			dest = encodeURIComponent(user) + "@" + dest
		}
		return "zed://ssh/" + dest + encode(zp)
	}
	dest := host
	if user != "" {
		dest = user + "@" + host
	}
	if strings.Contains(dest, "+") {
		// Remote-SSH splits its authority on "+": such a host goes as hex JSON.
		js, _ := json.Marshal(struct {
			HostName string `json:"hostName"`
			User     string `json:"user,omitempty"`
		}{host, user})
		dest = hex.EncodeToString(js)
	}
	return ed.scheme + "://vscode-remote/ssh-remote+" + encodeURIComponent(dest) + encode(rooted)
}
