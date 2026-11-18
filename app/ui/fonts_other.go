//go:build !darwin && !windows

package ui

import "gioui.org/font"

func init() {
	sans = "Inter"
	mono = "Liberation Mono, JetBrains Mono"
}

func systemFaces() []font.FontFace { return nil }
