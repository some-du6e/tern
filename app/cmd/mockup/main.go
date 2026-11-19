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
		{"8-details-git-menu", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, gitMenuScene()) }},
		{"9-details-popover", 1000, 800, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, popoverScene()) }},
		{"10-details-compact-running", 1440, 300, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, gitRunningScene()) }},
		{"11-git-confirm", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, confirmScene()) }},
		{"12-draft-workspace-menu", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, draftMenuScene()) }},
		{"13-header-rename", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			th.SetEditorText("header.title", "Port the T3 Connect client to Go")
			return th.Desktop(gtx, renameScene())
		}},
		{"14-details-pull-requests", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, pullRequestsScene()) }},
		{"15-details-pr-checks", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, checksScene()) }},
		{"16-merge-confirm", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, mergeScene()) }},
		{"17-header-title-menu", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, titleMenuScene()) }},
		{"18-details-branch-picker", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, branchPickerScene()) }},
		{"19-no-project-draft", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, noProjectScene()) }},
		{"20-details-init-git-actions", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, initGitScene()) }},
		{"21-details-items-menu", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, itemsMenuScene()) }},
		{"22-header-subagent-collapsed", 760, 500, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, subagentScene()) }},
		{"23-details-essential", 1440, 238, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, runningScene()) }},
	}
	if *anim != "" {
		if err := renderAnims(ui.NewTheme(), float32(*scale), *anim); err != nil {
			log.Fatalf("anim: %v", err)
		}
	}
	for _, d := range []bool{false, true} {
		ui.SetDark(d)
		list := shots
		if d {
			list = darkShots()
		}
		for _, s := range list {
			start := time.Now()
			path := filepath.Join(*out, s.name+".png")
			if err := render(s, float32(*scale), path); err != nil {
				log.Fatalf("%s: %v", s.name, err)
			}
			fmt.Printf("%-28s %s\n", path, time.Since(start).Round(time.Millisecond))
		}
	}
}

// darkShots are the header and the details card in T3's dark theme.
func darkShots() []shot {
	desktop := func(scene func() *ui.Scene) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
		return func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, scene()) }
	}
	return []shot{
		{"d1-thread-dark", 1440, 900, desktop(runningScene)},
		{"d8-details-git-menu-dark", 1440, 900, desktop(gitMenuScene)},
		{"d9-details-popover-dark", 1000, 800, desktop(popoverScene)},
		{"d12-draft-workspace-menu-dark", 1440, 900, desktop(draftMenuScene)},
		{"d14-details-pull-requests-dark", 1440, 900, desktop(pullRequestsScene)},
		{"d17-header-title-menu-dark", 1440, 900, desktop(titleMenuScene)},
		{"d19-no-project-draft-dark", 1440, 900, desktop(noProjectScene)},
	}
}

// render draws s on a theme of its own, so nothing one shot leaves open
// or on its way carries into the next, and lets its transitions run: the
// frame it writes out comes 300ms after the one that starts them.
func render(s shot, scale float32, path string) error {
	th := ui.NewTheme()
	size := image.Pt(int(float32(s.w)*scale), int(float32(s.h)*scale))
	win, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		return err
	}
	defer win.Release()
	ops := new(op.Ops)
	now := time.Now()
	for _, at := range []time.Time{now, now.Add(300 * time.Millisecond)} {
		ops.Reset()
		s.draw(th, layout.Context{
			Ops:         ops,
			Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
			Constraints: layout.Exact(size),
			Now:         at,
		})
	}
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
