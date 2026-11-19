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
		{"8-sidebar-hover", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarHoverScene()) }},
		{"8b-sidebar-menu", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarMenuScene()) }},
		{"8c-sidebar-filter", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarFilterScene()) }},
		{"8d-sidebar-snooze", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarSnoozeScene()) }},
		{"8e-sidebar-search", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarSearchScene()) }},
		{"8f-sidebar-empty", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarEmptyScene()) }},
		{"8h-sidebar-draft", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarDraftScene()) }},
		{"8i-sidebar-jump", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, sidebarJumpScene()) }},
		{"8j-sidebar-states", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			th.SetEditorText("rename", renamed)
			return th.Desktop(gtx, sidebarStatesScene())
		}},
		{"8k-sidebar-shelves", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			return th.Desktop(gtx, sidebarShelvesScene())
		}},
		{"8l-sidebar-searching", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			return th.Desktop(gtx, sidebarSearchingScene())
		}},
		{"8m-sidebar-no-results", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			return th.Desktop(gtx, sidebarNoResultsScene())
		}},
		{"8n-sidebar-no-projects", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			return th.Desktop(gtx, sidebarNoProjectsScene())
		}},
		{"8g-sidebar-dark", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			ui.SetDark(true)
			defer ui.SetDark(false)
			return th.Desktop(gtx, sidebarHoverScene())
		}},
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
		{"8-timeline", 1440, 2600, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, settledScene()) }},
		{"8b-timeline-dark", 1440, 2600, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			ui.SetDark(true)
			defer ui.SetDark(false)
			return th.Desktop(gtx, settledScene())
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
