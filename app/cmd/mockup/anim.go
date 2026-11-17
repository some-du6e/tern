package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/some-du6e/tern/app/ui"
)

// animFrame is one frame of an option animation and how long it shows.
type animFrame struct {
	from, to string
	t        float32
	hold     time.Duration
}

// walk visits each state in turn: a hold on it, then the ~180 ms ease into
// the next at 50 fps.
func walk(states ...string) []animFrame {
	const steps = 9 // 180 ms at 20 ms a frame
	var fs []animFrame
	for i, st := range states {
		if i > 0 {
			for k := 1; k < steps; k++ {
				fs = append(fs, animFrame{from: states[i-1], to: st, t: float32(k) / steps, hold: 20 * time.Millisecond})
			}
		}
		fs = append(fs, animFrame{to: st, t: 1, hold: 700 * time.Millisecond})
	}
	return fs
}

// anims are the menu animations: the Fast Mode switch going on and off.
func anims() map[string]func() ([]animFrame, *ui.Traits, func(t *ui.Traits, f animFrame)) {
	return map[string]func() ([]animFrame, *ui.Traits, func(*ui.Traits, animFrame)){
		"fast-toggle": func() ([]animFrame, *ui.Traits, func(*ui.Traits, animFrame)) {
			return walk("off", "on", "off"), claudeTraits(), func(t *ui.Traits, f animFrame) {
				g := &t.Groups[1]
				g.On, g.Moving, g.T = f.to == "on", f.t < 1, f.t
			}
		},
	}
}

// renderAnims writes each animation's frames as PNGs plus a durations file
// under dir/<name>; tools/gif.py stitches them. Go's own GIF encoder
// dithers dark UI badly, so the palette work happens there.
func renderAnims(th *ui.Theme, scale float32, dir string) error {
	const w, h = 264, 380
	size := image.Pt(int(w*scale), int(h*scale))
	win, err := headless.NewWindow(size.X, size.Y)
	if err != nil {
		return err
	}
	defer win.Release()
	for name, mk := range anims() {
		frames, base, set := mk()
		out := filepath.Join(dir, name)
		if err := os.MkdirAll(out, 0o755); err != nil {
			return err
		}
		durs, err := os.Create(filepath.Join(out, "durations.txt"))
		if err != nil {
			return err
		}
		for i, f := range frames {
			t := *base
			t.Groups = append([]ui.Trait(nil), base.Groups...)
			set(&t, f)
			ops := new(op.Ops)
			gtx := layout.Context{Ops: ops, Metric: unit.Metric{PxPerDp: scale, PxPerSp: scale}, Constraints: layout.Exact(size)}
			ui.Fill(gtx, ui.ColPanelHi, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = image.Point{}
				defer op.Offset(image.Pt(int(14*scale), int(14*scale))).Push(gtx.Ops).Pop()
				return th.TraitsMenu(gtx, &t)
			})
			if err := win.Frame(ops); err != nil {
				return err
			}
			img := image.NewRGBA(image.Rectangle{Max: size})
			if err := win.Screenshot(img); err != nil {
				return err
			}
			if err := writePNG(filepath.Join(out, fmt.Sprintf("%03d.png", i)), img); err != nil {
				return err
			}
			fmt.Fprintln(durs, f.hold.Milliseconds())
		}
		durs.Close()
	}
	return nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
