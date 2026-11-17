package shell

import (
	"errors"
	"os"
	"os/exec"
)

var kinds = map[string]bool{"sh": true, "bash": true}

// run writes code to a .command file, which Terminal opens in a new window.
// The window ends in a login shell, so it stays open to read.
func run(kind, code string) error {
	sh := "/bin/zsh"
	switch kind {
	case "sh":
	case "bash":
		sh = "/bin/bash"
	default:
		return errors.New("can't run that here")
	}
	f, err := os.CreateTemp("", "tern-*.command")
	if err != nil {
		return err
	}
	f.WriteString(script(sh, code))
	f.Close()
	if err := os.Chmod(f.Name(), 0o700); err != nil {
		return err
	}
	cmd := exec.Command("open", "-a", "Terminal", f.Name())
	if err := cmd.Run(); err != nil {
		return errors.New("couldn't open Terminal: " + err.Error())
	}
	return nil
}
