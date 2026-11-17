// Package ui draws tern's interface with Gio. The look is a tightened take on
// Dear ImGui's dark style: dense rows, flat frames, one blue accent.
package ui

import (
	"embed"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/io/system"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
)

//go:embed fonts/*.otf fonts/*.ttf
var fontFiles embed.FS

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

// Palette. Surfaces step up in lightness from the window to active frames.
var (
	ColWindow      = rgb(0x0c0e11)
	ColPanel       = rgb(0x121519)
	ColPanelHi     = rgb(0x161a1f)
	ColFrame       = rgb(0x1a1e24)
	ColFrameHover  = rgb(0x222730)
	ColFrameActive = rgb(0x2a303a)
	ColBorder      = rgb(0x242930)
	ColBorderHi    = rgb(0x333a45)

	ColText  = rgb(0xe2e5e9)
	ColDim   = rgb(0x8d95a1)
	ColFaint = rgb(0x5b626e)

	ColAccent     = rgb(0x4c8dff)
	ColAccentHi   = rgb(0x6aa2ff)
	ColAccentSoft = alpha(rgb(0x4c8dff), 0x2c)
	ColSelected   = alpha(rgb(0x4c8dff), 0x24)

	ColGreen  = rgb(0x3dd68c)
	ColAmber  = rgb(0xf2b544)
	ColRed    = rgb(0xff5f5f)
	ColViolet = rgb(0xb08cff)
	ColCyan   = rgb(0x5cc8e6)
	ColCode   = rgb(0xe8b77a)

	ColAddBg  = rgb(0x15291f)
	ColAddFg  = rgb(0x7fe0a7)
	ColDelBg  = rgb(0x2e171b)
	ColDelFg  = rgb(0xff8e8e)
	ColHunkFg = rgb(0x7aa7ff)
)

// Type scale, in sp.
const (
	SizeSmall unit.Sp = 11
	SizeUI    unit.Sp = 13
	SizeBody  unit.Sp = 13.5
	SizeTitle unit.Sp = 15
	SizeMono  unit.Sp = 12
)

type Theme struct {
	Shaper *text.Shaper
	in     input
	md     map[string]*mdCache

	richCache map[richKey]*richOps
	signIn    signInAnim

	// CanRun and Run give PowerShell and cmd code blocks their ▷: whether
	// a block in a language can run on this PC, and running it.
	CanRun func(lang string) bool
	Run    func(lang, code string) error

	// TitleInset is the room, in dp, the window's own buttons take at the top
	// left: the Mac's traffic lights, drawn over the app. With it the top of
	// the sidebar and the thread drags the window.
	TitleInset float32
}

// topBarH is the height of the sidebar's top row and the thread header,
// which the Mac's traffic lights sit centred in.
const topBarH = 48

// dragArea makes a size-sized area at the current offset move the window.
// Draw it first: what's drawn over it keeps its clicks.
func (th *Theme) dragArea(gtx C, size image.Point) {
	if th.TitleInset == 0 {
		return
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
}

func NewTheme() *Theme {
	var faces []font.FontFace
	entries, _ := fontFiles.ReadDir("fonts")
	for _, e := range entries {
		data, err := fontFiles.ReadFile("fonts/" + e.Name())
		if err != nil {
			panic(err)
		}
		parsed, err := opentype.ParseCollection(data)
		if err != nil {
			panic(err)
		}
		faces = append(faces, parsed...)
	}
	// The embedded faces come first; the system's fill in the scripts they
	// don't cover, like Chinese and Japanese.
	return &Theme{Shaper: text.NewShaper(text.WithCollection(faces))}
}

func UI(w font.Weight) font.Font { return font.Font{Typeface: "Inter", Weight: w} }
func Mono() font.Font            { return font.Font{Typeface: "JetBrains Mono"} }
