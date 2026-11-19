//go:build !windows && !darwin

package shell

import (
	"errors"
	"os/exec"
)

var kinds = map[string]bool{}

func run(kind, code string) error { return errors.New("running code needs Windows or a Mac") }

func opener(url string) *exec.Cmd { return exec.Command("xdg-open", url) }
