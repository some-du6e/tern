package ui

import "gioui.org/font"

func init() {
	// Segoe UI is next in T3's stack: Chrome uses it for what Inter lacks.
	sans = "Inter, Segoe UI"
	mono = "Consolas, JetBrains Mono"
}

// systemFaces has nothing to add: Consolas and Segoe UI come from the
// shaper's system font search.
func systemFaces() []font.FontFace { return nil }
