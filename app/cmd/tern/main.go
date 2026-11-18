// Command tern is the desktop app: a native T3 Code client.
package main

import (
	"errors"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	gioclip "gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/x/explorer"

	"github.com/some-du6e/tern/app/clip"
	"github.com/some-du6e/tern/app/core"
	"github.com/some-du6e/tern/app/shell"
	"github.com/some-du6e/tern/app/ui"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(append([]app.Option{app.Title("tern"), app.Size(unit.Dp(1100), unit.Dp(780)), app.MinSize(unit.Dp(960), unit.Dp(600))}, windowOptions...)...)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

// fadeIn is how long the inbox takes to appear after sign-in or boot.
const fadeIn = 280 * time.Millisecond

func run(w *app.Window) error {
	th := ui.NewTheme()
	th.CanRun, th.Run = shell.Runnable, shell.Run
	th.ControlsLeft, th.Frameless = controlsLeft, frameless
	// T3 follows the system's light or dark setting; so does tern.
	var dark atomic.Bool
	dark.Store(watchDark(func(d bool) {
		dark.Store(d)
		w.Invalidate()
	}))
	shownDark := !dark.Load()
	c, err := core.New(w.Invalidate)
	if err != nil {
		return err
	}
	expl := explorer.NewExplorer(w)
	c.ChooseImages = func() ([]clip.Image, error) { return chooseImages(expl) }
	c.Start()
	th.Focus("signin.email")
	var ops op.Ops
	var shown time.Time // when the desktop first drew after boot or sign-in
	for {
		ev := w.Event()
		expl.ListenEvents(ev)
		platformEvent(ev)
		switch e := ev.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			if d := dark.Load(); d != shownDark {
				shownDark = d
				ui.SetDark(d)
				titleDark(d)
			}
			gtx := app.NewContext(&ops, e)
			c.RunUI(th)
			s := c.Scene()
			switch {
			case s.Booting:
				paint.Fill(gtx.Ops, ui.ColWindow)
				shown = time.Time{}
			case s.Login != nil:
				th.LiveSignIn(gtx, s)
				shown = time.Time{}
			default:
				if shown.IsZero() {
					shown = gtx.Now
				}
				th.Desktop(gtx, s)
				if t := gtx.Now.Sub(shown); t < fadeIn {
					k := 1 - float32(t)/float32(fadeIn)
					col := ui.ColWindow
					col.A = uint8(255 * k * k)
					paint.FillShape(gtx.Ops, col, gioclip.Rect(image.Rectangle{Max: e.Size}).Op())
					gtx.Execute(op.InvalidateCmd{})
				}
			}
			if acts := th.Actions(); len(acts) > 0 {
				c.Handle(th, acts)
				gtx.Execute(op.InvalidateCmd{})
			} else if moving(s) {
				// Spinners and run timers: about 30 fps while anything is live.
				gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(33 * time.Millisecond)})
			}
			e.Frame(gtx.Ops)
		}
	}
}

// chooseImages is the composer's attach button: the system's file dialog
// for the images a message can carry, read in.
func chooseImages(expl *explorer.Explorer) ([]clip.Image, error) {
	files, err := expl.ChooseFiles(".png", ".jpg", ".jpeg", ".gif", ".webp")
	if errors.Is(err, explorer.ErrUserDecline) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []clip.Image
	for _, f := range files {
		data, err := io.ReadAll(io.LimitReader(f, clip.MaxBytes+1))
		name := "image"
		if of, ok := f.(*os.File); ok {
			name = filepath.Base(of.Name())
		}
		f.Close()
		switch {
		case err != nil:
			return out, err
		case len(data) > clip.MaxBytes:
			return out, errors.New(name + " is over 10 MB")
		}
		out = append(out, clip.Image{Name: name, Mime: http.DetectContentType(data), Data: data})
	}
	return out, nil
}

// moving reports whether anything on screen animates. The sign-in screen
// schedules its own frames.
func moving(s *ui.Scene) bool {
	if s.State == ui.ThreadRunning {
		return true
	}
	for _, r := range s.Active {
		if r.State == ui.ThreadRunning {
			return true
		}
	}
	for _, it := range s.Items {
		if it.Live {
			return true
		}
	}
	return false
}
