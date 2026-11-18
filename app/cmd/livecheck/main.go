// Command livecheck drives the real client headlessly against a paired
// environment: it renders frames, clicks with simulated pointer events, and
// writes a screenshot after each step. It never sends a message.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/some-du6e/tern/app/core"
	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

const scale = 2

type harness struct {
	th     *ui.Theme
	c      *core.App
	win    *headless.Window
	router input.Router
	ops    op.Ops
	size   image.Point
	out    string
	step   int
}

func (h *harness) frame() *ui.Scene {
	h.c.RunUI(h.th)
	s := h.c.Scene()
	h.ops.Reset()
	gtx := layout.Context{
		Ops:         &h.ops,
		Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
		Constraints: layout.Exact(h.size),
		Now:         time.Now(),
		Source:      h.router.Source(),
	}
	if s.Login != nil {
		h.th.LiveSignIn(gtx, s)
	} else {
		h.th.Desktop(gtx, s)
	}
	h.router.Frame(&h.ops)
	if acts := h.th.Actions(); len(acts) > 0 {
		for _, a := range acts {
			fmt.Printf("   action %s %s\n", a.Kind, a.ID)
		}
		h.c.Handle(h.th, acts)
	}
	return s
}

func (h *harness) shot(name string) {
	h.frame()
	if err := h.win.Frame(&h.ops); err != nil {
		log.Fatal(err)
	}
	img := image.NewRGBA(image.Rectangle{Max: h.size})
	if err := h.win.Screenshot(img); err != nil {
		log.Fatal(err)
	}
	h.step++
	f, _ := os.Create(filepath.Join(h.out, fmt.Sprintf("%02d-%s.png", h.step, name)))
	png.Encode(f, img)
	f.Close()
	fmt.Println("shot", name)
}

// click presses and releases at a point given in dp.
func (h *harness) click(x, y float32) {
	p := f32.Pt(x*scale, y*scale)
	h.router.Queue(
		pointer.Event{Kind: pointer.Move, Position: p, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Press, Position: p, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: p, Source: pointer.Mouse},
	)
	h.frame()
	h.frame()
}

func (h *harness) wait(what string, ok func(*ui.Scene) bool) *ui.Scene {
	deadline := time.Now().Add(20 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		t0 := time.Now()
		s := h.frame()
		if s.Net != last {
			fmt.Printf("   %s net=%q frame=%s\n", time.Now().Format("15:04:05.000"), s.Net, time.Since(t0).Round(time.Millisecond))
			last = s.Net
		}
		if ok(s) {
			return s
		}
		time.Sleep(50 * time.Millisecond)
	}
	log.Fatalf("timed out waiting for %s", what)
	return nil
}

func main() {
	storeDir := flag.String("store", "", "store with a paired session")
	out := flag.String("out", "/tmp/livecheck", "screenshot directory")
	bench := flag.String("bench", "", "after opening a thread, profile 300 frames into this file and exit")
	flag.Parse()
	store, err := t3.OpenStoreAt(*storeDir)
	if err != nil {
		log.Fatal(err)
	}
	os.MkdirAll(*out, 0o755)
	h := &harness{th: ui.NewTheme(), size: image.Pt(1440*scale, 900*scale), out: *out}
	h.win, err = headless.NewWindow(h.size.X, h.size.Y)
	if err != nil {
		log.Fatal(err)
	}
	h.c, err = core.NewWithStore(store, func() {})
	if err != nil {
		log.Fatal(err)
	}
	h.c.Start()
	start := time.Now()
	s := h.wait("threads", func(s *ui.Scene) bool { return s.Login == nil && len(s.Active)+len(s.Pinned) > 0 })
	fmt.Printf("inbox: %d pinned, %d active, %d snoozed, %d settled after %s\n", len(s.Pinned), len(s.Active), len(s.Snoozed), s.Settled, time.Since(start).Round(time.Millisecond))
	h.wait("config", func(s *ui.Scene) bool { return true })
	h.shot("inbox")

	// A real pointer click on the first row of the first section.
	first := s.Active[0]
	if len(s.Pinned) > 0 {
		first = s.Pinned[0]
	}
	h.click(130, 30+42+31+23)
	s = h.wait("thread open", func(s *ui.Scene) bool { return len(s.Timeline.Rows) > 0 || s.Empty != "" })
	fmt.Printf("opened %q (expected %q): %d rows, more=%v, model=%q effort=%q\n", s.Title, first.Title, len(s.Timeline.Rows), s.Timeline.MoreHistory, s.Model, s.Effort)
	h.shot("thread")
	if *bench != "" {
		h.benchmark(*bench)
		os.Exit(0)
	}

	// Unfold the first "Worked for" row, if any, through its action.
	for _, r := range s.Timeline.Rows {
		if r.Kind == ui.RowTurnFold {
			h.c.Handle(h.th, []ui.Action{{ID: "fold:" + r.Group, Kind: "click"}})
			break
		}
	}
	s = h.frame()
	h.shot("unfolded")
	// The newest tool row that opens, so it's on screen at the bottom of
	// the list.
	opened := ""
	for i := len(s.Timeline.Rows) - 1; i >= 0 && opened == ""; i-- {
		if r := s.Timeline.Rows[i]; r.Kind == ui.RowWork && len(r.Entries) == 1 && r.Entries[0].Expands {
			opened = r.Entries[0].ID
			h.c.Handle(h.th, []ui.Action{{ID: "entry:" + opened, Kind: "click"}})
		}
	}
	h.wait("tool output", func(s *ui.Scene) bool {
		for _, r := range s.Timeline.Rows {
			for _, e := range r.Entries {
				if e.ID == opened && e.Open {
					return e.Body == nil || !e.Body.Pending
				}
			}
		}
		return opened == ""
	})
	h.shot("tool-open")

	// Type into the composer with key events and press Enter. With a
	// read-only pairing the server must refuse it: that's the send path and
	// the error banner, without posting anything.
	h.th.Focus("composer")
	h.frame()
	h.router.Queue(key.EditEvent{Range: key.Range{Start: 0, End: 0}, Text: "tern livecheck: this must be refused"})
	h.frame()
	fmt.Printf("composer text: %q\n", h.th.EditorText("composer"))
	h.router.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	h.frame()
	s = h.wait("send result", func(s *ui.Scene) bool { return s.Error != "" })
	fmt.Printf("send refused as expected: %q\n", s.Error)
	h.shot("refused")
	h.c.Handle(h.th, []ui.Action{{ID: "error", Kind: "click"}})

	h.c.Handle(h.th, []ui.Action{{ID: "model", Kind: "click"}})
	h.wait("picker", func(s *ui.Scene) bool { return s.Picker != nil && len(s.Picker.Models) > 0 })
	h.shot("model-picker")
	h.c.Handle(h.th, []ui.Action{{ID: "traits", Kind: "click"}})
	h.shot("options")
	h.c.Handle(h.th, []ui.Action{{ID: "menus.close", Kind: "click"}, {ID: "new", Kind: "click"}})
	s = h.frame()
	fmt.Printf("draft in %q\n", s.Project)
	h.shot("draft")
	h.c.Handle(h.th, []ui.Action{{ID: "project", Kind: "click"}})
	s = h.frame()
	if s.Projects == nil || len(s.Projects.Groups) == 0 {
		log.Fatal("no project menu")
	}
	h.shot("draft-projects")
	if len(s.Projects.Groups[0].Choices) > 1 {
		h.c.Handle(h.th, []ui.Action{{ID: "project:0:1", Kind: "click"}})
		s = h.frame()
		fmt.Printf("draft moved to %q\n", s.Project)
		h.shot("draft-moved")
	}
	h.c.Handle(h.th, []ui.Action{{ID: "tabclose:draft", Kind: "click"}, {ID: "tabclose:" + s.TabIDs[0], Kind: "click"}})
	s = h.frame()
	fmt.Printf("nothing open: details panel hidden=%v\n", s.Run == nil)
	// Right-click the third row with the pointer: the thread menu opens there.
	at := f32.Pt(150*scale, (30+42+31+46*2+23)*scale)
	h.router.Queue(
		pointer.Event{Kind: pointer.Move, Position: at, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Press, Position: at, Buttons: pointer.ButtonSecondary, Source: pointer.Mouse},
		pointer.Event{Kind: pointer.Release, Position: at, Source: pointer.Mouse},
	)
	h.frame()
	s = h.frame()
	if s.Context == nil {
		log.Fatal("right-click opened no menu")
	}
	h.shot("thread-menu")
	h.c.Handle(h.th, []ui.Action{{ID: "ctx:0:4", Kind: "click"}})
	h.shot("delete-confirm")
	// Mark unread is harmless; read-only, the server must refuse it.
	h.c.Handle(h.th, []ui.Action{{ID: "ctx:0:2", Kind: "click"}})
	s = h.wait("menu refusal", func(s *ui.Scene) bool { return s.Error != "" })
	fmt.Printf("thread menu refused as expected: %q\n", s.Error)
	h.c.Handle(h.th, []ui.Action{{ID: "error", Kind: "click"}})
	h.c.Handle(h.th, []ui.Action{{ID: "account", Kind: "click"}})
	h.shot("account")
	h.c.Handle(h.th, []ui.Action{{ID: "menus.close", Kind: "click"}})
	// Sign-in, from an empty store.
	empty, _ := t3.OpenStoreAt(filepath.Join(os.TempDir(), "tern-empty"))
	h.c, _ = core.NewWithStore(empty, func() {})
	h.c.Start()
	h.wait("sign-in", func(s *ui.Scene) bool { return s.Login != nil })
	h.th.Focus("signin.email")
	// The entrance, part way and settled.
	time.Sleep(150 * time.Millisecond)
	h.shot("sign-in-entering")
	time.Sleep(700 * time.Millisecond)
	h.shot("sign-in")
	fmt.Println("ok")
	os.Exit(0)
}

// benchmark times frames on an open thread: the scene build and the layout
// separately, with a CPU profile of the whole loop.
func (h *harness) benchmark(path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	const n = 300
	var scene, lay time.Duration
	for i := 0; i < n; i++ {
		t0 := time.Now()
		s := h.c.Scene()
		t1 := time.Now()
		h.ops.Reset()
		gtx := layout.Context{Ops: &h.ops, Metric: unit.Metric{PxPerDp: scale, PxPerSp: scale}, Constraints: layout.Exact(h.size), Now: time.Now(), Source: h.router.Source()}
		h.th.Desktop(gtx, s)
		h.router.Frame(&h.ops)
		h.th.Actions()
		scene += t1.Sub(t0)
		lay += time.Since(t1)
	}
	pprof.StopCPUProfile()
	f.Close()
	fmt.Printf("per frame: scene %s, layout %s\n", (scene / n).Round(time.Microsecond), (lay / n).Round(time.Microsecond))
}
