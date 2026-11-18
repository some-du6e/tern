// Package ui draws tern's interface with Gio, ported from T3 Code's own web
// client: its tokens (tokens_gen.go), components and behaviour, as specified
// in docs/ui.
package ui

import (
	"image"
	"image/color"

	"gioui.org/io/system"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
)

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

// The theme. P is T3's palette for the light or dark theme the system is in;
// SP and CP are the same theme inside the sidebar and the composer, which
// restyle some tokens (dark mode's sidebar is black, for one). SetDark
// switches them.
var (
	P    = &T3Light
	SP   = &T3LightSidebar
	CP   = &T3LightComposer
	dark bool
)

// SetDark switches to T3's dark theme, or back to light. T3 follows the
// system's setting, and so does tern.
func SetDark(d bool) {
	dark = d
	P, SP, CP = &T3Light, &T3LightSidebar, &T3LightComposer
	if d {
		P, SP, CP = &T3Dark, &T3DarkSidebar, &T3DarkComposer
	}
	legacyColors()
}

// IsDark reports whether the dark theme is on.
func IsDark() bool { return dark }

func init() { SetDark(false) }

// The colours tern's screens were first drawn with, before the port to T3's
// design. Until a screen is ported they map onto T3's tokens, so the whole
// app already follows its light and dark themes.
var (
	ColWindow, ColPanel, ColPanelHi, ColFrame, ColFrameHover, ColFrameActive color.NRGBA
	ColBorder, ColBorderHi, ColText, ColDim, ColFaint                        color.NRGBA
	ColAccent, ColAccentHi, ColAccentSoft, ColSelected                       color.NRGBA
	ColGreen, ColAmber, ColRed, ColViolet, ColCyan, ColCode                  color.NRGBA
	ColAddBg, ColAddFg, ColDelBg, ColDelFg, ColHunkFg                        color.NRGBA
)

func legacyColors() {
	ColWindow, ColPanel, ColPanelHi = P.Background, SP.Sidebar, P.Background
	ColFrame, ColFrameHover, ColFrameActive = P.Card, P.Accent, P.Accent
	ColBorder, ColBorderHi = P.Border, P.Input
	ColText, ColDim, ColFaint = P.Foreground, P.MutedForeground, alpha(P.MutedForeground, 0xb3)
	ColAccent, ColAccentHi = P.Primary, P.Primary
	ColAccentSoft, ColSelected = alpha(P.Primary, 0x2c), alpha(P.Primary, 0x24)
	ColGreen, ColAmber, ColRed = P.Success, P.Warning, P.Error
	ColViolet, ColCyan, ColCode = Tailwind["violet-500"], P.Info, P.CodeForeground
	ColAddBg, ColAddFg = alpha(P.DiffAddition, 0x26), P.DiffAdditionForeground
	ColDelBg, ColDelFg = alpha(P.DiffDeletion, 0x26), P.DiffDeletionForeground
	ColHunkFg = P.Info
	chipColors()
}

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
	// The embedded faces come first; the system's fill in the scripts they
	// don't cover, like Chinese and Japanese.
	return &Theme{Shaper: text.NewShaper(text.WithCollection(faces()))}
}
