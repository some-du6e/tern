package ui

import "testing"

// SF Pro (upright and italic) and SF Mono load from /System at T3's four
// weights.
func TestMacSystemFonts(t *testing.T) {
	n := map[string]int{}
	for _, f := range systemFaces() {
		n[string(f.Font.Typeface)]++
	}
	if n["SF Pro"] != 8 || n["SF Mono"] != 4 {
		t.Fatalf("system faces = %v, want 8 SF Pro and 4 SF Mono", n)
	}
}
