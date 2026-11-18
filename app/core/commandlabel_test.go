package core

import (
	"strings"
	"testing"
)

// T3's own cases (commandLabel.test.ts) for the shapes tern parses.
func TestCommandProgramName(t *testing.T) {
	for _, c := range []struct{ command, want string }{
		{"/bin/bash --rcfile /tmp/config -c 'git status'", "git"},
		{"zsh -lc 'CI=1 env -u DEBUG sudo -u root vp test run'", "vp"},
		{`"C:\Program Files\Git\bin\bash.exe" -lc "git status"`, "git"},
		{"sudo -u root pnpm test", "pnpm"},
		{"zsh script.sh -c 'git status'", "zsh"},
		{"my-shell -c 'git status'", "my-shell"},
		{"$HOME/.bun/bin/bun test", "bun"},
		{`rg -n "if|for|while" src`, "rg"},
		{"vp test && git status", "vp"},
		{"echo `git status`", "echo"},
		{"cat <(rg needle src)", "cat"},
		{"cd packages/client-runtime && vp test run", "vp"},
		{"PATH+=:/tools npm test", "npm"},
		{"cd $(find . -type d | head -1) && git status", "git"},
		{"cd /tmp && # use the selected workspace\nnpm test", "npm"},
		{"cd ${ROOT:-path;with;semicolons} && bun test", "bun"},
		{"source ~/.nvm/nvm.sh && nvm use", "nvm"},
		{"unset DEBUG; node app.js", "node"},
		{"/bin/zsh -lc 'source ~/.nvm/nvm.sh && nvm use'", "nvm"},
		{"timeout 10s python3 script.py", "python3"},
		{"hash --help", "hash"},
		{"Exec node app.js", "Exec"},
		{"CD /tmp && npm test", "CD"},
		{"# first comment\n  # second comment\ngit status", "git"},
		{">/tmp/log && npm test", "npm"},
		{"cd /tmp && 2>& 1 node app.js", "node"},
		{"cd /tmp && {output}>/tmp/log deno test", "deno"},
	} {
		if got := commandProgramName(c.command); got != c.want {
			t.Errorf("commandProgramName(%q) = %q, want %q", c.command, got, c.want)
		}
	}
	for _, command := range []string{
		"zsh -lc", "zsh -lc 'git status", "zsh -lc 'env'",
		"if test -f package.json; then vp test; fi", "[[ -f package.json ]]", "test -f package.json",
		`for file in *; do echo "$file"; done`, "check() { vp test; }", "{ vp test; }", "(( count += 1 ))",
		". ./script.sh", "export NODE_ENV=test", "repeat 3 echo ok", "return 1", "true",
		"command -v git", "exec > output.log", "env CI=1 cd /tmp && npm test", "$PY scripts/check.py",
		"& { Get-Process }", "coproc npm test",
	} {
		if got := commandProgramName(command); got != "" {
			t.Errorf("commandProgramName(%q) = %q, want none", command, got)
		}
	}
	// Unwrapping and setup segments are bounded.
	cmd := "git status"
	for range 9 {
		cmd = "sh -c '" + strings.ReplaceAll(cmd, "'", `'\''`) + "'"
	}
	if got := commandProgramName(cmd); got != "" {
		t.Errorf("nine nested shells gave %q", got)
	}
	var setup []string
	for range 2000 {
		setup = append(setup, "export VALUE=configured")
	}
	if got := commandProgramName(strings.Join(append(setup, "npm test"), ";")); got != "" {
		t.Errorf("2000 setup segments gave %q", got)
	}
}

// A command's row shows its script without the shell around it, and its
// control characters as pictures.
func TestCommandDisplayText(t *testing.T) {
	for _, c := range []struct{ command, want string }{
		{"/bin/zsh -lc 'git status'", "git status"},
		{"bash -c 'ls' extra", "bash -c 'ls' extra"},
		{"zsh -lc 'a' && b", "zsh -lc 'a' && b"},
		{"printf '\x1b[31m'", "printf '␛[31m'"},
		{"  go test ./...  ", "go test ./..."},
	} {
		if got := commandDisplayText(c.command); got != c.want {
			t.Errorf("commandDisplayText(%q) = %q, want %q", c.command, got, c.want)
		}
	}
}
