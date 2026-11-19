package shell

import "testing"

func TestEncodeCommand(t *testing.T) {
	// The example from PowerShell's own help for -EncodedCommand.
	if got := encodeCommand("dir"); got != "ZABpAHIA" {
		t.Fatalf("dir = %s", got)
	}
}

func TestScript(t *testing.T) {
	got := script("/bin/bash", "echo hi")
	want := "#!/bin/bash\ncd ~\nrm -f \"$0\"\necho hi\nexec \"${SHELL:-/bin/zsh}\" -l\n"
	if got != want {
		t.Fatalf("script = %q", got)
	}
}
