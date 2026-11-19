// Package ui draws tern's interface with Gio, ported from T3 Code's own web
// client: its tokens (tokens_gen.go), components and behaviour, as specified
// in docs/ui.
package ui

import (
	"image"
	"image/color"
	"math"

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

// over is c composited onto the opaque bg the way browsers do, in sRGB.
// Gio blends in linear light, so a translucent T3 token painted as is comes
// out lighter than in T3: white at 8% on black is #141414 in Chrome but
// about #4f4f4f in Gio. Where the surface under a token is known, paint
// over(token, surface).
func over(c, bg color.NRGBA) color.NRGBA {
	a := float32(c.A) / 255
	mix := func(x, y uint8) uint8 { return uint8(float32(x)*a + float32(y)*(1-a) + 0.5) }
	return color.NRGBA{R: mix(c.R, bg.R), G: mix(c.G, bg.G), B: mix(c.B, bg.B), A: 0xff}
}

// faded is c in a group at opacity k (disabled:opacity-64 is 0xa3) as
// Chrome composites the group onto the opaque bg, in sRGB. Gio's layer
// opacity blends in linear light and would come out paler.
func faded(c, bg color.NRGBA, k uint8) color.NRGBA {
	return over(alpha(c, uint8((int(c.A)*int(k)+127)/255)), bg)
}

// linear is an sRGB channel in linear light, 0 to 1.
func linear(v uint8) float64 {
	s := float64(v) / 255
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// gioAlpha is the alpha at which Gio, blending c in linear light over
// from, paints to, the colour a browser composites over from: for
// translucent paint over an unknown surface that's mostly from. It
// solves the channel where c and from differ most.
func gioAlpha(c, from, to color.NRGBA) uint8 {
	type ch struct{ c, f, t uint8 }
	best, d := ch{}, -1.0
	for _, x := range []ch{{c.R, from.R, to.R}, {c.G, from.G, to.G}, {c.B, from.B, to.B}} {
		if dd := math.Abs(linear(x.f) - linear(x.c)); dd > d {
			best, d = x, dd
		}
	}
	if d < 1e-6 {
		return 0
	}
	a := (linear(best.f) - linear(best.t)) / (linear(best.f) - linear(best.c))
	return uint8(math.Max(0, math.Min(1, a))*255 + 0.5)
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
	ColAccent, ColAccentHi, ColSelected                                      color.NRGBA
	ColGreen, ColAmber, ColRed, ColViolet, ColCyan, ColCode                  color.NRGBA
	ColAddBg, ColAddFg, ColDelBg, ColDelFg, ColHunkFg                        color.NRGBA
)

// legacyColors maps them, composited onto --background where the token is
// translucent (see over), since that's what the old screens mostly draw on.
func legacyColors() {
	bg := P.Background
	on := func(c color.NRGBA) color.NRGBA { return over(c, bg) }
	ColWindow, ColPanel, ColPanelHi = bg, SP.Sidebar, bg
	ColFrame, ColFrameHover, ColFrameActive = on(P.Card), on(P.Accent), on(P.Accent)
	ColBorder, ColBorderHi = on(P.Border), on(P.Input)
	ColText, ColDim, ColFaint = on(P.Foreground), on(P.MutedForeground), on(alpha(P.MutedForeground, 0xb3))
	ColAccent, ColAccentHi = on(P.Primary), on(P.Primary)
	ColSelected = on(alpha(P.Primary, 0x24))
	ColGreen, ColAmber, ColRed = on(P.Success), on(P.Warning), on(P.Error)
	ColViolet, ColCyan, ColCode = Tailwind["violet-500"], on(P.Info), on(P.CodeForeground)
	ColAddBg, ColAddFg = on(alpha(P.DiffAddition, 0x26)), on(P.DiffAdditionForeground)
	ColDelBg, ColDelFg = on(alpha(P.DiffDeletion, 0x26)), on(P.DiffDeletionForeground)
	ColHunkFg = on(P.Info)
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

	// ControlsLeft is where the title bar's fixed controls start, in dp:
	// 12, or 90 on a Mac, where the traffic lights sit in the top bar
	// (T3's --workspace-controls-left). Frameless means tern draws the title
	// bar itself, so its headers drag the window.
	ControlsLeft float32
	Frameless    bool
}

// topBarH is the height of the sidebar's header row, the chat header and
// the right panel's tab bar (--workspace-topbar-height).
const topBarH = 52

// dragArea makes a size-sized area at the current offset move the window.
// Draw it first: what's drawn over it keeps its clicks.
func (th *Theme) dragArea(gtx C, size image.Point) {
	if !th.Frameless {
		return
	}
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
}

func NewTheme() *Theme {
	// The embedded faces come first; the system's fill in the scripts they
	// don't cover, like Chinese and Japanese.
	return &Theme{Shaper: text.NewShaper(text.WithCollection(faces())), ControlsLeft: 12}
}
