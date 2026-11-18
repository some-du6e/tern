package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	gioinput "gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// markdownClicks draws src as an assistant's reply 600dp wide at 1x, and
// clicks at a point, returning that frame's actions.
func markdownClicks(src string) (click func(p image.Point) []Action, r *gioinput.Router) {
	th := NewTheme()
	r = new(gioinput.Router)
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400)), Source: r.Source(), Now: time.Now()}
		th.markdown(gtx, src, mdBody(), nil)
		r.Frame(&ops)
		return th.Actions()
	}
	frame()
	return func(p image.Point) []Action {
		fp := f32.Pt(float32(p.X), float32(p.Y))
		r.Queue(pointer.Event{Kind: pointer.Move, Position: fp, Source: pointer.Mouse})
		frame()
		r.Queue(pointer.Event{Kind: pointer.Press, Position: fp, Buttons: pointer.ButtonPrimary, Source: pointer.Mouse},
			pointer.Event{Kind: pointer.Release, Position: fp, Source: pointer.Mouse})
		return frame()
	}, r
}

// A code block's copy button is the header's last, 24dp square, 6dp in
// from the frame's top right (MarkdownCodeBlock.tsx); it copies the code
// as written.
func TestCodeBlockCopy(t *testing.T) {
	click, r := markdownClicks("Run this:\n\n```go\nfunc main() {}\n```")
	// The paragraph's 22.75dp line, 10.4dp, then the frame.
	top := 23 + 10
	click(image.Pt(600-6-12, top+6+12))
	if mime, data, ok := r.WriteClipboard(); !ok || mime != "application/text" || string(data) != "func main() {}" {
		t.Fatalf("clipboard = %q %q %v", mime, data, ok)
	}
}

// Shell fences get "Run in terminal" just before the copy button, 2dp
// apart: it sends the trimmed command to the thread's terminal. Other
// languages have none.
func TestCodeBlockRun(t *testing.T) {
	click, _ := markdownClicks("```bash\nls -la\n```\n\n```go\nfunc main() {}\n```")
	run := image.Pt(600-6-24-2-12, 6+12)
	if acts := click(run); len(acts) != 1 || acts[0] != (Action{ID: "terminal.run", Kind: "click", Text: "ls -la"}) {
		t.Fatalf("run click: %+v", acts)
	}
	// The first frame is 30 + 12.8 + 17.875 + 12.8 tall and the second
	// 10.4 under it; there the same spot is the wrap toggle, which stays in
	// the timeline.
	second := 30 + 13 + 18 + 13 + 10
	if acts := click(run.Add(image.Pt(0, second))); len(acts) != 0 {
		t.Fatalf("a go fence's toolbar sent %+v", acts)
	}
}
