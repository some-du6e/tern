package core

import "testing"

func TestPreviewURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://example.com": "https://example.com",
		"example.com/docs":    "https://example.com/docs",
		"localhost:5173":      "http://localhost:5173",
		":8080":               "http://localhost:8080",
		"3000":                "http://localhost:3000",
		"go generics":         "https://www.google.com/search?q=go+generics",
		"about:blank":         "about:blank",
		"  ":                  "",
	} {
		if got := previewURL(in); got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
}
