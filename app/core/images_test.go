package core

import (
	"bytes"
	"image"
	"image/png"
	"slices"
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

// chipped is the numbers of the pictures the scene shows as chips.
func chipped(s *ui.Scene) []int {
	var out []int
	for _, th := range s.Thumbs {
		if th.N != 0 {
			out = append(out, th.N)
		}
	}
	return out
}

// Pasted pictures are the composer's tiles, named for their files; pasted
// into text they're also chips at the caret, which name the file and its
// size and are sent there (useComposerAttachmentIntake.ts).
func TestPasteImages(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	shot := clip.Image{Name: "pasted.png", Mime: "image/png", Data: pngBytes(t)}
	a.pasted(a.composerKey(), []clip.Image{shot, shot}, nil)
	if s := a.build(time.Now()); len(s.Thumbs) != 2 || s.Thumbs[0].Name != "pasted.png" || s.Thumbs[0].Size != len(shot.Data) {
		t.Fatalf("thumbs = %+v, want two named pasted.png", s.Thumbs)
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
	if s := a.build(time.Now()); len(chipped(s)) != 0 {
		t.Fatalf("tiles numbered as chips: %+v", s.Thumbs)
	}
	// Into text, a picture goes in as a chip at the caret.
	th.SetEditorText("composer", "look ")
	a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	a.RunUI(th)
	if got := th.EditorText("composer"); !strings.HasPrefix(got, "look ") || !slices.Equal(ui.Chips(got), []int{3}) || got != "look "+th.ImageChip(3, "pasted.png", len(shot.Data)) {
		t.Fatalf("composer = %q, want picture 3's chip after the text", got)
	}
	// Removing a tile leaves the text alone; removing a chip's picture
	// takes its chip.
	a.handle(th, ui.Action{ID: "image.remove:0", Kind: "click"})
	if s := a.build(time.Now()); len(s.Thumbs) != 2 || !slices.Equal(ui.Chips(th.EditorText("composer")), []int{3}) {
		t.Fatalf("after removing a tile: %+v, %q", s.Thumbs, th.EditorText("composer"))
	}
	a.handle(th, ui.Action{ID: "image.remove:1", Kind: "click"})
	if s := a.build(time.Now()); len(s.Thumbs) != 1 || th.EditorText("composer") != "look " {
		t.Fatalf("chip left behind: %+v, %q", s.Thumbs, th.EditorText("composer"))
	}
	a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	a.RunUI(th)
	if got := ui.Chips(th.EditorText("composer")); !slices.Equal(got, []int{1}) {
		t.Fatalf("the free number isn't reused: %v", got)
	}

	// Backspacing a chip drops its picture; undo brings it back. The tile
	// stays either way.
	chip := th.ImageChip(1, "pasted.png", len(shot.Data))
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "look "})
	if s := a.build(time.Now()); len(s.Thumbs) != 1 || len(chipped(s)) != 0 {
		t.Fatalf("after deleting the chip: %+v", s.Thumbs)
	}
	a.handle(th, ui.Action{ID: "composer", Kind: "change", Text: "look " + chip})
	if s := a.build(time.Now()); len(s.Thumbs) != 2 || !slices.Equal(chipped(s), []int{1}) {
		t.Fatalf("after undo: %+v", s.Thumbs)
	}

	// Text on the clipboard pastes as text.
	a.uiOps = nil
	a.pasted(a.composerKey(), nil, clip.ErrNone)
	if len(a.uiOps) != 1 {
		t.Fatalf("want a text paste queued, got %d ops", len(a.uiOps))
	}

	// A message takes T3's 100 pictures.
	for i := 0; i < 110; i++ {
		a.pasted(a.composerKey(), []clip.Image{shot}, nil)
	}
	if s := a.build(time.Now()); len(s.Thumbs) != 100 || !strings.Contains(s.Error, "100") {
		t.Fatalf("kept %d, error %q; want 100 and the cap said", len(s.Thumbs), s.Error)
	}
}

// A picture pasted into a question's answer goes with that answer
// (attachmentsByQuestionId): it's the question's, not the thread's.
func TestQuestionPictures(t *testing.T) {
	a := testApp(t)
	openWith(t, a, &t3.RuntimeRequest{ID: "u1", Kind: "user_input"}, nil,
		map[string]any{"id": "i1", "type": "user_input_request", "requestId": "u1", "ordinal": 1,
			"questions": []any{map[string]any{"id": "q1", "header": "Q", "question": "Which?"}}})
	shot := clip.Image{Name: "answer.png", Mime: "image/png", Data: pngBytes(t)}
	a.mu.Lock()
	key := a.imageKey()
	a.mu.Unlock()
	a.pasted(key, []clip.Image{shot}, nil)
	th := ui.NewTheme()
	a.RunUI(th)
	if s := a.build(time.Now()); len(s.Thumbs) != 1 || !slices.Equal(ui.Chips(th.EditorText("composer")), []int{1}) {
		t.Fatalf("question pictures: %+v, %q", s.Thumbs, th.EditorText("composer"))
	}
	// Answered, the request is gone: the thread's composer has no pictures.
	_, tt := a.thread("swiss/t1")
	tt.PendingRuntimeRequest = nil
	if s := a.build(time.Now()); len(s.Thumbs) != 0 {
		t.Fatalf("the question's picture shows in the thread's composer: %+v", s.Thumbs)
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
	imgs := []pastedImage{{id: "abc", n: 1, img: clip.Image{Name: "one.png"}}, {id: "def", n: 2, img: clip.Image{Name: "two.png"}}}
	var recs []t3.ImageRecord
	for _, p := range imgs {
		recs = append(recs, t3.NewImageRecord(p.id, p.img.Name, t3.Attachment{ID: "att", Name: p.img.Name, MimeType: "image/png", SizeBytes: 5}))
	}
	// Picture 1's chip is in the text, so it's sent there; picture 2 has
	// none, so it goes at the end.
	m := message("look "+ui.ChipText(1)+" here", imgs, recs, true)
	if want := "look ![one.png](t3-context://v1/image/image_abc) here\n\n![two.png](t3-context://v1/image/image_def)"; m["text"] != want {
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
	imgs := []pastedImage{{id: "x", n: 2, img: clip.Image{Name: "shot.png"}}}
	recs := []t3.ImageRecord{t3.NewImageRecord("x", "shot.png", t3.Attachment{ID: "a"})}
	if got := message(text, imgs, recs, true)["text"]; got != "do it like this ![shot.png](t3-context://v1/image/image_x)" {
		t.Fatalf("sent %q", got)
	}
}
