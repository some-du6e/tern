package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/some-du6e/tern/app/ui"
)

// The composer's states (docs/ui/07-composer.md): a prompt with a picture
// chip and a tile, the @ menu, a run with its queue and banners, the
// questions, and a new thread's hero, with and without a project.

// pictures is n made-up screenshots, as the composer's tiles.
func pictures(n int) []ui.Thumb {
	var out []ui.Thumb
	tints := []color.RGBA{{0x24, 0x42, 0x5f, 0xff}, {0x3b, 0x2b, 0x52, 0xff}, {0x1f, 0x4d, 0x3a, 0xff}}
	for i := 0; i < n; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 160, 100))
		for y := 0; y < 100; y++ {
			for x := 0; x < 160; x++ {
				c := tints[i%len(tints)]
				switch {
				case y < 14:
					c = color.RGBA{0xe8, 0xe8, 0xec, 0xff}
				case x > 10 && x < 60 && y > 24 && y < 90:
					c = color.RGBA{c.R + 0x30, c.G + 0x30, c.B + 0x30, 0xff}
				}
				img.SetRGBA(x, y, c)
			}
		}
		var b bytes.Buffer
		if err := png.Encode(&b, img); err != nil {
			panic(err)
		}
		out = append(out, ui.NewThumb(b.Bytes()))
	}
	return out
}

// composerScene is an idle thread whose prompt holds a picture as a chip,
// with another picture as a tile alone, and a mode that isn't the default.
func composerScene() *ui.Scene {
	s := base("Gio sidebar: virtualize 1k rows")
	s.State, s.StateLabel, s.Run.Status = ui.ThreadIdle, "Idle", "Idle"
	s.Items = s.Items[:4]
	thumbs := pictures(2)
	thumbs[0].N = 1
	s.Thumbs = thumbs
	s.Composer = "compare " + ui.ChipText(1) + " with the header we had before the port, and keep the gaps"
	s.Runtime, s.RuntimeChip = "approval-required", true
	return s
}

// mentionScene is the @ menu over the composer, its second row lit.
func mentionScene() *ui.Scene {
	s := composerScene()
	s.Thumbs, s.RuntimeChip, s.Runtime = nil, false, "full-access"
	s.Composer = "look at @comp"
	s.Suggest = []ui.Suggestion{
		{Icon: "folder", Label: "components", Desc: "app/ui"},
		{Icon: "file", Label: "composer.go", Desc: "app/ui"},
		{Icon: "file", Label: "composerdock.go", Desc: "app/ui"},
		{Icon: "file", Label: "composer_test.go", Desc: "app/ui"},
	}
	s.SuggestSel = 1
	return s
}

// busyScene is a run with nothing typed: Stop, the queue past three, and
// a failed action and a notice attached over the card.
func busyScene() *ui.Scene {
	s := base("Port the T3 Connect client to Go")
	s.Queue = []string{
		"then add a tray icon on Windows with the running count",
		"and a menu item to pause every run",
		"check it on a 125% display",
		"write it up in the README",
	}
	s.Error = "Couldn't upload the picture: the server said 413"
	s.Notice = "Loading earlier turns…"
	return s
}

// answersScene is the second of two questions: Previous, then Submit.
func answersScene() *ui.Scene {
	s := questionScene()
	s.Question = 1
	s.Composer = "pair it from the desktop"
	return s
}

// heroScene is a new thread in a project: the question over the
// composer and its strip, centred in the canvas.
func heroScene() *ui.Scene {
	s := base("")
	s.Title, s.State, s.StateLabel = "New thread", ui.ThreadIdle, "Draft"
	s.Tabs = []string{"New thread"}
	s.Items, s.MoreAbove, s.Run, s.Changes = nil, false, nil, nil
	s.Hero, s.CanScratch = "tern", true
	return s
}

// scratchScene is a new thread without a project.
func scratchScene() *ui.Scene {
	s := heroScene()
	s.Hero, s.CanScratch, s.Scratch = "", false, true
	return s
}
