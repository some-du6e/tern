package ui

import (
	"bytes"
	"os"

	"gioui.org/font"
	fontapi "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

func init() {
	sans = "SF Pro, Inter"
	mono = "SF Mono, Menlo, JetBrains Mono"
}

// systemFaces loads the Mac's SF Pro (upright and italic) and SF Mono, the
// fonts T3 renders here. They can't be bundled; macOS has had them as
// variable fonts in /System/Library/Fonts since 10.15.
func systemFaces() []font.FontFace {
	var out []font.FontFace
	for _, f := range []struct {
		path   string
		family font.Typeface
		style  font.Style
	}{
		{"/System/Library/Fonts/SFNS.ttf", "SF Pro", font.Regular},
		{"/System/Library/Fonts/SFNSItalic.ttf", "SF Pro", font.Italic},
		{"/System/Library/Fonts/SFNSMono.ttf", "SF Mono", font.Regular},
	} {
		if ft := loadFont(f.path); ft != nil {
			out = append(out, weightFaces(ft, f.family, f.style)...)
			if f.family == "SF Pro" && f.style == font.Regular && ft.Upem() > 0 {
				capEm = fontapi.NewFace(ft).LineMetric(fontapi.CapHeight) / float32(ft.Upem())
			}
		}
	}
	return out
}

// loadFont parses the font file at path, or returns nil.
func loadFont(path string) *fontapi.Font {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lds, err := opentype.NewLoaders(bytes.NewReader(data))
	if err != nil || len(lds) == 0 {
		return nil
	}
	ft, err := fontapi.NewFont(lds[0])
	if err != nil {
		return nil
	}
	return ft
}
