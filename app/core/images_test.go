package core

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/clip"
	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

func pngBytes(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 30, 20))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestPasteImages(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	shot := clip.Image{Name: "pasted.png", Mime: "image/png", Data: pngBytes(t)}
	a.pasted(a.composerKey(), []clip.Image{shot, shot}, nil)
	if s := a.build(time.Now()); len(s.Thumbs) != 2 {
		t.Fatalf("thumbs = %d, want 2", len(s.Thumbs))
	}

	// They belong to that thread's composer, not the next one's.
	a.openThread("mac/same")
	if s := a.build(time.Now()); len(s.Thumbs) != 0 {
		t.Fatalf("other thread shows %d thumbs", len(s.Thumbs))
	}
	a.openThread("swiss/t1")
	// Pasted into an empty prompt, as in T3, they're tiles alone: no chips.
	th := ui.NewTheme()
	a.RunUI(th)
	if got := th.EditorText("composer"); got != "" {
		t.Fatalf("composer = %q, want no chips", got)
	}
	if s := a.build(time.Now()); s.Thumbs[0].N != 0 || s.Thumbs[1].N != 0 {
		t.Fatalf("tiles numbered as chips: %+v", s.Thumbs)
	}
	// Into text, a picture goes in as a chip at the caret.
	th.SetEditorText("composer", "look ")
	a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	a.RunUI(th)
	if got := th.EditorText("composer"); got != "look "+ui.ChipText(3) {
		t.Fatalf("composer = %q, want the chip after the text", got)
	}
	// Removing a tile leaves the text alone; removing a chip's picture
	// takes its chip.
	a.handle(th, ui.Action{ID: "image.remove:0", Kind: "click"})
	if s := a.build(time.Now()); len(s.Thumbs) != 2 || th.EditorText("composer") != "look "+ui.ChipText(3) {
		t.Fatalf("after removing a tile: %+v, %q", s.Thumbs, th.EditorText("composer"))
	}
	a.handle(th, ui.Action{ID: "image.remove:1", Kind: "click"})
	if s := a.build(time.Now()); len(s.Thumbs) != 1 || th.EditorText("composer") != "look " {
		t.Fatalf("chip left behind: %+v, %q", s.Thumbs, th.EditorText("composer"))
	}
	a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	a.RunUI(th)
	if got := th.EditorText("composer"); got != "look "+ui.ChipText(1) {
		t.Fatalf("the free number is reused: %q", got)
	}

	// Backspacing a chip drops its picture; undo brings it back. The tile
	// stays either way.
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "look "})
	if imgs := a.images[a.composerKey()]; len(imgs) != 1 || imgs[0].n != 2 {
		t.Fatalf("after deleting ①: %+v", imgs)
	}
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "look " + ui.ChipText(1)})
	if imgs := a.images[a.composerKey()]; len(imgs) != 2 || imgs[0].n != 1 {
		t.Fatalf("after undo: %d images", len(imgs))
	}

	// Text on the clipboard pastes as text.
	a.uiOps = nil
	a.pasted(a.composerKey(), nil, clip.ErrNone)
	if len(a.uiOps) != 1 {
		t.Fatalf("want a text paste queued, got %d ops", len(a.uiOps))
	}

	// There's a cap.
	for i := 0; i < 12; i++ {
		a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	}
	if n := len(a.images[a.composerKey()]); n != maxImages {
		t.Fatalf("kept %d, want %d", n, maxImages)
	}
}

func TestOpenThreadComposerUsesRef(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	a.choices["swiss/t1"] = &choice{runtime: "approval-required"}
	if s := a.build(time.Now()); s.Runtime != "approval-required" || !s.RuntimeChip {
		t.Fatalf("runtime = %q, chip %v", s.Runtime, s.RuntimeChip)
	}
}

func TestMessageCarriesImages(t *testing.T) {
	imgs := []pastedImage{{id: "abc", n: 1}, {id: "def", n: 2}}
	var recs []t3.ImageRecord
	for _, p := range imgs {
		recs = append(recs, t3.NewImageRecord(p.id, p.label(), t3.Attachment{ID: "att", Name: "pasted.png", MimeType: "image/png", SizeBytes: 5}))
	}
	// Image 1's chip is in the text, so it's sent there; Image 2 has none,
	// so it goes at the end.
	m := message("look "+ui.ChipText(1)+" here", imgs, recs, true)
	if want := "look ![Image 1](t3-context://v1/image/image_abc) here\n\n![Image 2](t3-context://v1/image/image_def)"; m["text"] != want {
		t.Fatalf("text = %q", m["text"])
	}
	rec := recs[0]
	imgs = imgs[:1]
	ctx, _ := m["context"].(map[string]any)
	if recs, _ := ctx["records"].([]t3.ImageRecord); len(recs) != 2 || recs[0].AttachmentID != "att" {
		t.Fatalf("context = %+v", m["context"])
	}
	old := message("hi", imgs, []t3.ImageRecord{rec}, false)
	if _, has := old["context"]; has || !strings.Contains(old["text"].(string), "<t3_context") {
		t.Fatalf("no inline context: %+v", old)
	}
	if plain := message("hi", nil, nil, true); len(plain) != 1 || plain["text"] != "hi" {
		t.Fatalf("no images: %+v", plain)
	}
}

// A chip at the very end of the message survives the send's trim: its fill
// is a space to Go, and trimming it lost the picture's place.
func TestChipAtTheEnd(t *testing.T) {
	text := ui.TrimText("do it like this " + ui.ChipText(2) + "  \n")
	if text != "do it like this "+ui.ChipText(2) {
		t.Fatalf("trimmed to %q", text)
	}
	imgs := []pastedImage{{id: "x", n: 2}}
	recs := []t3.ImageRecord{t3.NewImageRecord("x", imgs[0].label(), t3.Attachment{ID: "a"})}
	if got := message(text, imgs, recs, true)["text"]; got != "do it like this ![Image 2](t3-context://v1/image/image_x)" {
		t.Fatalf("sent %q", got)
	}
}
