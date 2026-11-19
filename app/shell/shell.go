// Package shell opens URLs with the app this computer has for them: a web
// address in the browser, an editor's vscode:// link in the editor.
package shell

// Open hands url to the app registered for its scheme, like an editor's
// vscode:// link.
func Open(url string) error {
	cmd := opener(url)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the opener once it's done, so it isn't left a zombie.
	go func() { _ = cmd.Wait() }()
	return nil
}
