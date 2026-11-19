package shell

import "os/exec"

func opener(url string) *exec.Cmd { return exec.Command("open", url) }
