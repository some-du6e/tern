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

// markdownClicks draws src in style st 600dp wide at 1x, and clicks at a
// point, returning that frame's actions.
func markdownClicks(src string, st mdStyle) (click func(p image.Point) []Action, r *gioinput.Router) {
	th := NewTheme()
	r = new(gioinput.Router)
	var ops op.Ops
	frame := func() []Action {
		ops.Reset()
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Constraints: layout.Exact(image.Pt(600, 400)), Source: r.Source(), Now: time.Now()}
		th.markdown(gtx, src, st, nil)
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

// runnableBody is a reply in a thread whose terminal can run its blocks.
func runnableBody() mdStyle {
	st := mdBody()
	st.run = true
	return st
}

// A code block's copy button is the header's last, 24dp square, 6dp in
// from the frame's top right inside its 1dp border (MarkdownCodeBlock.tsx);
// it copies the code as T3's code element holds it, its last line ended.
func TestCodeBlockCopy(t *testing.T) {
	click, r := markdownClicks("Run this:\n\n```go\nfunc main() {}\n```", mdBody())
	// The paragraph's 22.75dp line, 10.4dp, then the frame.
	top := 23 + 10
	click(image.Pt(600-1-6-12, top+1+6+12))
	if mime, data, ok := r.WriteClipboard(); !ok || mime != "application/text" || string(data) != "func main() {}\n" {
		t.Fatalf("clipboard = %q %q %v", mime, data, ok)
	}
}

// A code block is 1 + 30 + its code + 1 tall: the border takes its room in
// light, where it shows, and in dark, where it's transparent (§3.6).
func TestCodeBlockBorderBox(t *testing.T) {
	defer SetDark(false)
	for _, d := range []bool{false, true} {
		SetDark(d)
		th := NewTheme()
		var ops op.Ops
		gtx := layout.Context{Ops: &ops, Metric: unit.Metric{PxPerDp: 4, PxPerSp: 4}, Constraints: layout.Constraints{Max: image.Pt(4*600, 4*400)}, Now: time.Now()}
		// 12.8 + 17.875 + 12.8 of code at 4x: 51 + 72 + 51 px.
		if h := th.markdown(gtx, "```go\nfunc main() {}\n```", mdBody(), nil).Size.Y; h != 4*(1+30+1)+51+72+51 {
			t.Errorf("dark %v: the block is %.2fdp tall", d, float32(h)/4)
		}
	}
}

// Shell fences get "Run in terminal" just before the copy button, 2dp
// apart: it sends the trimmed command to the thread's terminal. T3's own
// cases decide which (ChatMarkdown.browser.test.tsx:332-377): only a
// closed fence of one line runs, so a reply cut off mid-command never
// does. Other languages, streaming replies and threads without a project
// have none; there the spot is the wrap toggle, which stays in the
// timeline.
func TestCodeBlockRun(t *testing.T) {
	run := func(top int) image.Point { return image.Pt(600-1-6-24-2-12, top+1+6+12) }
	for _, c := range []struct {
		src, cmd string
		top      int
	}{
		{"```bash\necho hello\n```", "echo hello", 0},
		{"~~~bash\necho tilde\n~~~", "echo tilde", 0},
		{"> ```bash\n> echo quote\n> ```", "echo quote", 0},
		{"````bash\necho four\n````", "echo four", 0},
		// The item's paragraph, 22.75dp, then 10.4 to the block.
		{"- Command\n\n  ```bash\n  echo list\n  ```", "echo list", 23 + 10},
		{"```bash\r\necho crlf\r\n```\r", "echo crlf", 0},
	} {
		click, _ := markdownClicks(c.src, runnableBody())
		if acts := click(run(c.top)); len(acts) != 1 || acts[0] != (Action{ID: "terminal.run", Kind: "click", Text: c.cmd}) {
			t.Errorf("%q: run click sent %+v", c.src, acts)
		}
	}
	for _, src := range []string{
		"```bash\necho one\necho two\n```",
		"```typescript\necho hello\n```",
		"```bash\n\n```",
		"```bash\necho hello\n\n```",
		"```bash\necho hello\\\n```",
		"```bash\necho safe \u202e#\n```",
		"```bash\necho incomplete",
		"~~~bash\necho incomplete",
		"````bash\necho incomplete\n```",
		`<pre><code class="language-bash">echo html</code></pre>`,
	} {
		click, _ := markdownClicks(src, runnableBody())
		if acts := click(run(0)); len(acts) != 0 {
			t.Errorf("%q: the toolbar sent %+v", src, acts)
		}
	}
	streaming := runnableBody()
	streaming.streaming = true
	for what, st := range map[string]mdStyle{"no project": mdBody(), "streaming": streaming} {
		click, _ := markdownClicks("```bash\nls -la\n```", st)
		if acts := click(run(0)); len(acts) != 0 {
			t.Errorf("%s: the toolbar sent %+v", what, acts)
		}
	}
}
