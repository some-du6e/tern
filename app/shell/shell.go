// Package shell runs a reply's code block in a console window on this
// computer, for the ▷ on PowerShell and cmd blocks on Windows and shell
// blocks on a Mac.
package shell

import (
	"encoding/base64"
	"strings"
	"unicode/utf16"
)

// Runnable reports whether a block in lang can run here.
func Runnable(lang string) bool {
	return kinds[langs[strings.ToLower(lang)]]
}

// langs maps fence languages to the shell that runs them.
var langs = map[string]string{
	"powershell": "powershell", "ps1": "powershell", "pwsh": "powershell", "ps": "powershell",
	"cmd": "cmd", "bat": "cmd", "batch": "cmd",
	"sh": "sh", "shell": "sh", "bash": "bash", "zsh": "sh", "console": "sh", "shellscript": "sh",
}

// Run starts code in a new console window that stays open, so its output can
// be read.
func Run(lang, code string) error { return run(langs[strings.ToLower(lang)], code) }

// encodeCommand is code for PowerShell's -EncodedCommand: UTF-16LE in
// base64, so quotes and newlines arrive as written.
func encodeCommand(code string) string {
	u := utf16.Encode([]rune(code))
	b := make([]byte, 2*len(u))
	for i, c := range u {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// script is a .command file for sh that runs code from the home folder and
// then hands over to the user's own shell.
func script(sh, code string) string {
	return "#!" + sh + "\ncd ~\nrm -f \"$0\"\n" + code + "\nexec \"${SHELL:-/bin/zsh}\" -l\n"
}
