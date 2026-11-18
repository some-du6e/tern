package ui

import (
	"embed"
	"io/fs"

	"gioui.org/font"
	"gioui.org/font/opentype"
	fontapi "github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

// T3 Code's type, per docs/ui/01-tokens.md §3.9: its sans stack is
// -apple-system, BlinkMacSystemFont, "Inter Variable", "Segoe UI", so a Mac
// renders SF Pro and everything else the Inter Variable it bundles. tern
// embeds that same Inter (4.001), instanced at the four weights T3 uses by
// tools/t3ui/instance-inter.py, one face per unicode-range subset as Chrome
// loads it. Code is ui-monospace, "SF Mono", Menlo, Consolas, "Liberation
// Mono": SF Mono on a Mac, Consolas on Windows. JetBrains Mono is only the
// last resort when none of those is installed.

//go:embed fonts/inter/*.ttf fonts/*.ttf
var fontFiles embed.FS

// sans and mono are the font families text asks for, best first; the
// platform files set them.
var sans, mono font.Typeface

// faces is every embedded face, then the platform's own (systemFaces).
func faces() []font.FontFace {
	var out []font.FontFace
	fs.WalkDir(fontFiles, "fonts", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fontFiles.ReadFile(path)
		if err != nil {
			panic(err)
		}
		parsed, err := opentype.ParseCollection(data)
		if err != nil {
			panic(err)
		}
		out = append(out, parsed...)
		return nil
	})
	return append(out, systemFaces()...)
}

// cssWeights are the weights T3 uses, as Gio's weights and CSS's.
var cssWeights = []struct {
	gio font.Weight
	css float32
}{{font.Normal, 400}, {font.Medium, 500}, {font.SemiBold, 600}, {font.Bold, 700}}

// varFace is one instance of a variable font. Gio has no font variations,
// so each weight is a face of its own, made with fontapi's.
type varFace struct {
	font *fontapi.Font
	vars []fontapi.Variation
}

func (v varFace) Face() *fontapi.Face {
	f := fontapi.NewFace(v.font)
	f.SetVariations(v.vars)
	return f
}

// weightFaces makes a face per T3 weight of the variable font ft under
// family. The shaper tells faces apart by their *fontapi.Font, so each gets
// a shallow copy: its own pointer over the same parsed, read-only tables.
func weightFaces(ft *fontapi.Font, family font.Typeface, style font.Style) []font.FontFace {
	var out []font.FontFace
	for _, w := range cssWeights {
		cp := *ft
		out = append(out, font.FontFace{
			Font: font.Font{Typeface: family, Weight: w.gio, Style: style},
			Face: varFace{&cp, []fontapi.Variation{{Tag: ot.MustNewTag("wght"), Value: w.css}}},
		})
	}
	return out
}

func UI(w font.Weight) font.Font { return font.Font{Typeface: sans, Weight: w} }
func Mono() font.Font            { return font.Font{Typeface: mono} }
