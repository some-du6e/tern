package main

import "C"

// Kept apart from appearance_darwin.go: a file with //export may only
// declare C functions in its preamble, not define them.

//export ternAppearanceChanged
func ternAppearanceChanged(dark C.int) {
	if appearanceChanged != nil {
		appearanceChanged(dark != 0)
	}
}
