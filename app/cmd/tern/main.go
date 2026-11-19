// Command tern is the desktop app: a native T3 Code client.
package main

import (
	"log"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"

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

// issueURL is where the boot failure page files a bug report.
var issueURL = "https://github.com/some-du6e/tern/issues/new?" + url.Values{"title": {"tern failed to start"}}.Encode()

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
	// A client that can't start (its store or key unreadable) is T3's boot
	// failure page, and Reload tries again; failing twice in 5 minutes
	// opens its "More options".
	var c *core.App
	var failure *ui.Failure
	var failedAt time.Time
	start := func() {
		var err error
		if c, err = core.New(w.Invalidate); err != nil {
			now := time.Now()
			failure = core.BootFailure(err, !failedAt.IsZero() && now.Sub(failedAt) < 5*time.Minute, now)
			failedAt = now
			return
		}
		c.Start()
	}
	start()
	th.Focus("signin.email")
	var ops op.Ops
	for {
		ev := w.Event()
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
			if c == nil {
				th.Gate(gtx, &ui.Scene{Gate: ui.GateBootFailed, Failure: failure})
				for _, act := range th.Actions() {
					switch act.ID {
					case "boot.reload":
						start()
					case "boot.more":
						failure.Open = !failure.Open
					case "boot.issue":
						go func() { _ = shell.Open(issueURL) }()
					}
					gtx.Execute(op.InvalidateCmd{})
				}
				e.Frame(gtx.Ops)
				continue
			}
			c.RunUI(th)
			s := c.Scene()
			if s.Gate == ui.GateApp {
				th.Desktop(gtx, s)
			} else {
				th.Gate(gtx, s)
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

// moving reports whether anything on screen animates. The gates and the
// toasts schedule their own frames.
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
