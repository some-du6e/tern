//go:build !windows && !darwin

package shell

import "os/exec"

func opener(url string) *exec.Cmd { return exec.Command("xdg-open", url) }
