//go:build !darwin && !windows

package main

// watchDark has no system setting to follow here; tern starts light, as T3
// does in a browser without a colour-scheme preference.
func watchDark(func(dark bool)) bool { return false }
