package ui

import (
	"testing"

	"gioui.org/io/key"
	"github.com/charmbracelet/x/vt"
)

func TestTermKeys(t *testing.T) {
	cases := []struct {
		e    key.Event
		mac  bool
		want string
	}{
		{key.Event{Name: "C", Modifiers: key.ModCtrl}, true, "\x03"},
		{key.Event{Name: key.NameReturn}, false, "\r"},
		{key.Event{Name: key.NameUpArrow}, false, "\x1b[A"},
		{key.Event{Name: key.NameDeleteBackward}, true, "\x7f"},
		{key.Event{Name: "B", Modifiers: key.ModAlt}, false, "\x1bb"},
	}
	for _, c := range cases {
		k, ok := termKey(c.e, c.mac)
		if !ok {
			t.Fatalf("%v: not a terminal key", c.e)
		}
		emu := vt.NewEmulator(10, 2)
		go emu.SendKey(k)
		buf := make([]byte, 16)
		n, _ := emu.Read(buf)
		if got := string(buf[:n]); got != c.want {
			t.Errorf("%v = %q, want %q", c.e, got, c.want)
		}
	}
	// Plain letters come as typed text, and a Mac's Option types its own.
	for _, e := range []key.Event{{Name: "A"}, {Name: "E", Modifiers: key.ModAlt}} {
		if _, ok := termKey(e, true); ok {
			t.Errorf("%v should be left to text input", e)
		}
	}
}

func TestTermSelection(t *testing.T) {
	th := NewTheme()
	emu := vt.NewSafeEmulator(20, 3)
	emu.Write([]byte("one\r\ntwo three\r\nfour"))
	sb := emu.ScrollbackLen()
	// From "two" on row 1 to "fo" on row 2.
	th.in.termSel = termSel{from: [2]int{sb + 2, 1}, to: [2]int{sb + 1, 0}}
	if got := th.termSelected(emu); got != "two three\nfo" {
		t.Fatalf("selected %q", got)
	}
	if !th.in.termSel.has(0, sb+1, 5) || th.in.termSel.has(0, sb+2, 2) || th.in.termSel.has(1, sb+1, 0) {
		t.Fatal("has() disagrees with the selection")
	}
}
