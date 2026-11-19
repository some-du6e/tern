package t3

import "testing"

func TestImageContext(t *testing.T) {
	r := NewImageRecord("2269711e-e641-4810-99e3-7f8d91698756", "image.png", Attachment{
		Type: "image", ID: "965f22bf-52e8-40f4-bfda-8f2922b8e064-e9ab0eb4-fbae-4f38-a947-828092af2afa",
		Name: "image.png", MimeType: "image/png", SizeBytes: 2650})
	if got, want := r.Token(), "![image.png](t3-context://v1/image/image_2269711e-e641-4810-99e3-7f8d91698756)"; got != want {
		t.Fatalf("token = %s", got)
	}
	text := WithImages("look at this", []ImageRecord{r})
	if want := "look at this\n\n" + r.Token(); text != want {
		t.Fatalf("text = %q", text)
	}
	if got := WithImages("  ", []ImageRecord{r}); got != r.Token() {
		t.Fatalf("image only = %q", got)
	}

	// What the server hands the agent, copied from a real message.
	want := "look at this\n\n[Image: image.png; ref=image_2269711e-e641-4810-99e3-7f8d91698756]\n\n" +
		"<t3_context version=\"1\">\n" +
		"<context kind=\"image\" id=\"image_2269711e-e641-4810-99e3-7f8d91698756\">\n" +
		"name: image.png\nmimeType: image/png\nsizeBytes: 2650\n" +
		"attachmentId: 965f22bf-52e8-40f4-bfda-8f2922b8e064-e9ab0eb4-fbae-4f38-a947-828092af2afa\n" +
		"</context>\n</t3_context>"
	if got := ProjectImages(text, []ImageRecord{r}); got != want {
		t.Fatalf("projected:\n%s\nwant:\n%s", got, want)
	}

	if got := StripContext(text); got != "look at this\n\n[image.png]" {
		t.Fatalf("stripped = %q", got)
	}
	if got := StripContext("see [notes.md](t3-context://v1/mention/m_1) please"); got != "see notes.md please" {
		t.Fatalf("mention = %q", got)
	}
}
