package shell

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// kinds are the shells in langs this computer has.
var kinds = map[string]bool{"powershell": true, "cmd": true}

func run(kind, code string) error {
	var cmd *exec.Cmd
	switch kind {
	case "powershell":
		exe := "powershell.exe"
		if p, err := exec.LookPath("pwsh.exe"); err == nil {
			exe = p
		}
		cmd = exec.Command(exe, "-NoExit", "-NoLogo", "-EncodedCommand", encodeCommand(code))
	case "cmd":
		f, err := os.CreateTemp("", "tern-*.cmd")
		if err != nil {
			return err
		}
		f.WriteString("@echo off\r\n" + code + "\r\n")
		f.Close()
		cmd = exec.Command("cmd.exe", "/K", f.Name())
	default:
		return errors.New("can't run that here")
	}
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Dir = home
	}
	// tern has no console, so give the shell its own window.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010} // CREATE_NEW_CONSOLE
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
