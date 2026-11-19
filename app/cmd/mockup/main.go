// Command mockup renders tern's screens with fixture data to PNGs, using the
// same Gio code the app draws with. Run it with EGL_PLATFORM=surfaceless on a
// machine with no display.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/some-du6e/tern/app/ui"
)

type shot struct {
	name string
	w, h int
	draw func(th *ui.Theme, gtx layout.Context) layout.Dimensions
}

func main() {
	out := flag.String("out", "mockups", "output directory")
	scale := flag.Float64("scale", 2, "pixels per dp")
	anim := flag.String("anim", "", "also write the picker animation frames under this directory")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	th := ui.NewTheme()
	shots := []shot{
		{"1-thread", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, runningScene()) }},
		{"2-approval-model-picker", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, approvalScene()) }},
		{"2b-model-options", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, traitsScene()) }},
		{"2c-codex-options", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, codexScene()) }},
		{"3-diff", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, diffScene()) }},
		{"4-question", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, questionScene()) }},
		{"5-new-thread-commands", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, draftScene()) }},
		{"6-sign-in", 1440, 900, gateShot(signInScene, "signin.email", "")},
		{"6b-sign-in-error", 1440, 900, gateShot(signInErrorScene, "signin.email", "someone@example.com")},
		{"6c-sign-in-code", 1440, 900, gateShot(codeScene, "signin.code", "4821")},
		{"6d-sign-in-password", 1440, 900, gateShot(passwordScene, "signin.password", "correct horse")},
		{"6e-pair", 1440, 900, gateShot(pairScene, "pair.link", "")},
		{"6f-pair-error", 1440, 900, gateShot(pairErrorScene, "pair.link", "https://studio.example.com/pair#token=spent")},
		{"6g-boot", 1440, 900, gateShot(bootScene)},
		{"6h-boot-failed", 1440, 900, gateShot(bootFailedScene)},
		{"6i-session-error", 1440, 900, gateShot(sessionErrorScene)},
		{"6j-no-projects", 1440, 900, homeShot(ui.HomeNoProjects)},
		{"6k-no-computer", 1440, 900, homeShot(ui.HomeNoComputer)},
		{"6l-no-thread", 1440, 900, homeShot(ui.HomeNoThread)},
		{"6m-draft-error", 1440, 900, homeShot(ui.HomeDraftError)},
		{"6p-toasts", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, toastsScene()) }},
		{"6q-sign-in-dark", 1440, 900, inDark(gateShot(signInScene, "signin.email", "you@example.com"))},
		{"6r-pair-error-dark", 1440, 900, inDark(gateShot(pairErrorScene, "pair.link", "https://studio.example.com/pair#token=spent"))},
		{"6s-session-error-dark", 1440, 900, inDark(gateShot(sessionErrorScene))},
		{"6t-boot-failed-dark", 1440, 900, inDark(gateShot(bootFailedScene))},
		{"6u-no-computer-dark", 1440, 900, inDark(homeShot(ui.HomeNoComputer))},
		{"6w-toasts-dark", 1440, 900, inDark(func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, toastsScene()) })},
		{"7-phone", 390, 844, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Phone(gtx, phoneScene()) }},
	}
	if *anim != "" {
		if err := renderAnims(th, float32(*scale), *anim); err != nil {
			log.Fatalf("anim: %v", err)
		}
	}
	for _, s := range shots {
		start := time.Now()
		path := filepath.Join(*out, s.name+".png")
		if err := render(th, s, float32(*scale), path); err != nil {
			log.Fatalf("%s: %v", s.name, err)
		}
		fmt.Printf("%-28s %s\n", path, time.Since(start).Round(time.Millisecond))
	}
}

// gateShot draws scene's gate with its fields typed in: field, text, and
// so on.
func gateShot(scene func() *ui.Scene, fields ...string) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
	return func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
		for i := 0; i+1 < len(fields); i += 2 {
			th.SetEditorText(fields[i], fields[i+1])
		}
		return th.Gate(gtx, scene())
	}
}

// homeShot draws the window with the main column's empty state h.
func homeShot(h ui.Home) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
	return func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, homeScene(h)) }
}

// inDark draws draw in the dark theme.
func inDark(draw func(th *ui.Theme, gtx layout.Context) layout.Dimensions) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
	return func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
		ui.SetDark(true)
		defer ui.SetDark(false)
		return draw(th, gtx)
	}
}

func render(th *ui.Theme, s shot, scale float32, path string) error {
	size := image.Pt(int(float32(s.w)*scale), int(float32(s.h)*scale))
	win, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		return err
	}
	defer win.Release()
	ops := new(op.Ops)
	gtx := layout.Context{
		Ops:         ops,
		Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
		Constraints: layout.Exact(size),
		Now:         time.Now(),
	}
	s.draw(th, gtx)
	if err := win.Frame(ops); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rectangle{Max: size})
	if err := win.Screenshot(img); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
