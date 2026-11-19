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

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/some-du6e/tern/app/ui"
)

type shot struct {
	name  string
	w, h  int
	draw  func(th *ui.Theme, gtx layout.Context) layout.Dimensions
	hover image.Point // where the pointer rests, in dp; nowhere at 0, 0
}

// desktop draws scene's window.
func desktop(scene func() *ui.Scene) func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
	return func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Desktop(gtx, scene()) }
}

func main() {
	out := flag.String("out", "mockups", "output directory")
	scale := flag.Float64("scale", 2, "pixels per dp")
	anim := flag.String("anim", "", "also write the picker animation frames under this directory")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	none := image.Point{}
	shots := []shot{
		{"1-thread", 1440, 900, desktop(runningScene), none},
		{"2-approval-model-picker", 1440, 900, desktop(approvalScene), none},
		{"2b-model-options", 1440, 900, desktop(traitsScene), none},
		{"2c-codex-options", 1440, 900, desktop(codexScene), none},
		{"3-diff", 1440, 900, desktop(diffScene), none},
		{"4-question", 1440, 900, desktop(questionScene), none},
		{"5-new-thread-commands", 1440, 900, desktop(draftScene), none},
		{"6-sign-in", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.SignInScreen(gtx, signInScene()) }, none},
		{"7-phone", 390, 844, func(th *ui.Theme, gtx layout.Context) layout.Dimensions { return th.Phone(gtx, phoneScene()) }, none},
		{"8-details-git-menu", 1440, 900, desktop(gitMenuScene), none},
		{"9-details-popover", 1000, 800, desktop(popoverScene), none},
		{"10-details-compact-running", 1440, 300, desktop(gitRunningScene), none},
		{"11-git-confirm", 1440, 900, desktop(confirmScene), none},
		{"12-draft-workspace-menu", 1440, 900, desktop(draftMenuScene), none},
		{"13-header-rename", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			th.SetEditorText("header.title", "Port the T3 Connect client to Go")
			return th.Desktop(gtx, renameScene())
		}, none},
		{"14-details-pull-requests", 1440, 900, desktop(pullRequestsScene), none},
		{"15-details-pr-checks", 1440, 900, desktop(checksScene), none},
		{"16-merge-confirm", 1440, 900, desktop(mergeScene), none},
		{"17-header-title-menu", 1440, 900, desktop(titleMenuScene), none},
		{"18-details-branch-picker", 1440, 900, desktop(branchPickerScene), none},
		{"19-no-project-draft", 1440, 900, desktop(noProjectScene), none},
		{"20-details-init-git-actions", 1440, 900, desktop(initGitScene), none},
		{"21-details-items-menu", 1440, 900, desktop(itemsMenuScene), none},
		{"22-header-subagent-collapsed", 760, 500, desktop(subagentScene), none},
		{"23-details-essential", 1440, 238, desktop(runningScene), none},
		// The pointer on the title: its chevron, and its tooltip under it.
		{"24-header-title-hover", 1440, 900, desktop(runningScene), image.Pt(393, 26)},
		// The pointer on a quick action that can't run: its reason; and a
		// project without scripts, whose row adds one.
		{"25-details-hint-reason", 1440, 900, desktop(hintScene), image.Pt(1257, 236)},
		// The pointer on the thread menu's Snooze: its submenu.
		{"26-thread-menu-snooze", 1440, 900, desktop(titleMenuScene), image.Pt(393, 41+5+3*28+14)},
		{"27-delete-confirm", 1440, 900, desktop(deleteScene), none},
		{"28-git-commit", 1440, 900, func(th *ui.Theme, gtx layout.Context) layout.Dimensions {
			th.SetEditorText("commit.message", "Pair the watch with a scoped session")
			return th.Desktop(gtx, commitScene())
		}, none},
		{"29-git-commit-editing", 1440, 900, desktop(commitEditScene), none},
		// The pointer on the pull request: T3's rich tooltip under it.
		{"30-details-pr-tooltip", 1440, 900, desktop(prTipScene), image.Pt(1300, 236)},
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
	none := image.Point{}
	return []shot{
		{"d1-thread-dark", 1440, 900, desktop(runningScene), none},
		{"d8-details-git-menu-dark", 1440, 900, desktop(gitMenuScene), none},
		{"d9-details-popover-dark", 1000, 800, desktop(popoverScene), none},
		{"d12-draft-workspace-menu-dark", 1440, 900, desktop(draftMenuScene), none},
		{"d14-details-pull-requests-dark", 1440, 900, desktop(pullRequestsScene), none},
		{"d17-header-title-menu-dark", 1440, 900, desktop(titleMenuScene), none},
		{"d19-no-project-draft-dark", 1440, 900, desktop(noProjectScene), none},
		{"d25-details-hint-reason-dark", 1440, 900, desktop(hintScene), image.Pt(1257, 236)},
		{"d27-delete-confirm-dark", 1440, 900, desktop(deleteScene), none},
		{"d29-git-commit-editing-dark", 1440, 900, desktop(commitEditScene), none},
		{"d30-details-pr-tooltip-dark", 1440, 900, desktop(prTipScene), image.Pt(1300, 236)},
	}
}

// render draws s on a theme of its own, so nothing one shot leaves open
// or on its way carries into the next, and lets its transitions run: the
// frame it writes out comes 300ms after the one that starts them. With the
// pointer resting on it, the pointer comes after the first frame, once
// there's something under it to tell, and the last frame comes 800ms on,
// past a tooltip's 600ms wait and its fade.
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
	frames := []time.Duration{0, 300 * time.Millisecond}
	var router input.Router
	var src input.Source
	hover := s.hover != (image.Point{})
	if hover {
		frames = []time.Duration{0, 10 * time.Millisecond, 800 * time.Millisecond}
		src = router.Source()
	}
	for i, d := range frames {
		ops.Reset()
		s.draw(th, layout.Context{
			Ops:         ops,
			Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
			Constraints: layout.Exact(size),
			Now:         now.Add(d),
			Source:      src,
		})
		router.Frame(ops)
		if hover && i == 0 {
			router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(s.hover.X)*scale, float32(s.hover.Y)*scale)})
		}
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
