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
		{"6-sign-in", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.SignInScreen(gtx, signInScene()) }},
		{"7-phone", 390, 844, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Phone(gtx, phoneScene()) }},
		{"c1-composer-chips", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, composerScene()) }},
		{"c2-composer-mentions", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, mentionScene()) }},
		{"c3-composer-busy", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, busyScene()) }},
		{"c4-composer-answers", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, answersScene()) }},
		{"c4b-composer-stash", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, stashScene()) }},
		{"c3b-composer-queue-edit", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, queueEditScene()) }},
		{"c9-composer-permissions", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, permissionsScene()) }},
		{"c10-composer-suggestion", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, suggestionScene()) }},
		{"c11-composer-ultrafast", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, ultrafastScene()) }},
		{"c12-composer-no-provider", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, noProviderScene()) }},
		{"c13-composer-fresh-session", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, freshScene()) }},
		{"c5b-new-thread-stash", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, heroStashScene()) }},
		{"c5-new-thread-hero", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, heroScene()) }},
		{"c6-new-thread-scratch", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, scratchScene()) }},
		{"c7-composer-dark", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			ui.SetDark(true)
			defer ui.SetDark(false)
			return th.Desktop(gtx, busyScene())
		}},
		{"c8-new-thread-hero-dark", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			ui.SetDark(true)
			defer ui.SetDark(false)
			return th.Desktop(gtx, heroScene())
		}},
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
