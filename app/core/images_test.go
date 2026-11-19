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
	// Each paste typed its chip at the caret.
	th := ui.NewTheme()
	a.RunUI(th)
	if got := th.EditorText("composer"); got != ui.ChipText(1)+ui.ChipText(2) {
		t.Fatalf("composer = %q", got)
	}
	a.handle(th, ui.Action{ID: "image.remove:0", Kind: "click"})
	if s := a.build(time.Now()); len(s.Thumbs) != 1 || s.Thumbs[0].N != 2 {
		t.Fatalf("after remove: %+v", s.Thumbs)
	}
	if got := th.EditorText("composer"); got != ui.ChipText(2) {
		t.Fatalf("chip left behind: %q", got)
	}
	a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	a.RunUI(th)
	if got := th.EditorText("composer"); got != ui.ChipText(2)+ui.ChipText(1) {
		t.Fatalf("the free number is reused: %q", got)
	}

	// Backspacing a chip drops its picture; undo brings it back.
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "look " + ui.ChipText(1)})
	if imgs := a.images[a.composerKey()]; len(imgs) != 1 || imgs[0].n != 1 {
		t.Fatalf("after deleting ②: %+v", imgs)
	}
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "look " + ui.ChipText(1) + ui.ChipText(2)})
	if imgs := a.images[a.composerKey()]; len(imgs) != 2 || imgs[1].n != 2 {
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
	if s := a.build(time.Now()); s.Runtime != runtimeLabel("approval-required") {
		t.Fatalf("runtime label = %q", s.Runtime)
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

// Sent pictures stay decoded up to sentBudget: past it, the ones shown
// longest ago go, but never one the last scene showed.
func TestSentPicturesBudget(t *testing.T) {
	defer func(b int) { sentBudget = b }(sentBudget)
	a := testApp(t)
	p := ui.NewPicture(pngBytes(t))
	for i, k := range []string{"e/a", "e/b", "e/c"} {
		a.sent[k] = &sentThumb{picture: p, shown: uint64(i)}
		a.sentOrder = append(a.sentOrder, k)
	}
	a.scenes = 2 // the last scene showed e/c
	sentBudget = 2 * p.Bytes()
	a.trimSent()
	if a.sent["e/a"] != nil || a.sent["e/b"] == nil || a.sent["e/c"] == nil {
		t.Fatalf("over the budget by one, kept %v", a.sentOrder)
	}
	sentBudget = 0
	a.trimSent()
	if len(a.sent) != 1 || a.sent["e/c"] == nil {
		t.Errorf("with no budget, kept %v", a.sentOrder)
	}
}
