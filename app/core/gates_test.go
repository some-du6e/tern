package core

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// What the pairing form's Alert says (docs/ui/11-empty-auth.md §4.2,
// §4.3): T3's words for a link without its host or token and for a
// refused token; anything else as it failed, with what to do if the
// token was spent.
func TestPairingErrors(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{t3.ErrPairingLink, "This pairing link is missing its backend host or token."},
		{&t3.Error{What: "Pairing", Status: 401}, "Invalid pairing token. Check the token and try again."},
		{fmt.Errorf("this client requires a newer server. Update T3 Code on studio to connect"),
			"This client requires a newer server. Update T3 Code on studio to connect. If the backend accepted this one-time token, request a new pairing link before retrying."},
	} {
		if got := pairingError(c.err); got != c.want {
			t.Errorf("%v: %q, want %q", c.err, got, c.want)
		}
	}
}

// An empty field is refused on the form, before any request is sent.
func TestEmptyPairingLink(t *testing.T) {
	a := testApp(t)
	a.signedIn, a.pairing = false, true
	a.handle(ui.NewTheme(), ui.Action{ID: "pair.next", Kind: "click"})
	s := a.build(time.Now())
	if s.Gate != ui.GatePairing || s.Pair == nil || s.Pair.Error != "Enter a pairing link to continue." || s.Pair.Submitting {
		t.Fatalf("gate %d, pairing %+v", s.Gate, s.Pair)
	}
}

// The window's gate: the splash until boot decides, the root error view
// for a failed first request, the sign-in or the pairing page.
func TestGates(t *testing.T) {
	a := testApp(t)
	for _, c := range []struct {
		set  func()
		want ui.Gate
	}{
		{func() { a.booted = false }, ui.GateBooting},
		{func() { a.booted, a.failure = true, &ui.Failure{Message: "List environments failed (500)."} }, ui.GateSessionError},
		{func() { a.failure, a.signedIn = nil, false }, ui.GateSignIn},
		{func() { a.pairing = true }, ui.GatePairing},
		{func() { a.signedIn = true }, ui.GateApp},
	} {
		c.set()
		if s := a.build(time.Now()); s.Gate != c.want {
			t.Errorf("gate %d, want %d", s.Gate, c.want)
		}
	}
}

// The boot failure page's report (bootError.ts:96-120) drops hosts and
// tokens and ends with the build, the platform and the time.
func TestBootFailureReport(t *testing.T) {
	err := fmt.Errorf("open store: %w", errors.New("read https://studio.example.com/pair#token=abc123: permission denied"))
	f := BootFailure(err, true, time.Date(2026, 11, 18, 17, 28, 53, 152_000_000, time.UTC))
	for _, bad := range []string{"studio.example.com", "abc123"} {
		if strings.Contains(f.Report, bad) {
			t.Errorf("report names %q:\n%s", bad, f.Report)
		}
	}
	if !strings.HasSuffix(f.Report, "\nTime: 2026-11-18T17:28:53.152Z") || !strings.Contains(f.Report, "\nVersion: ") || !f.Open {
		t.Errorf("report:\n%s", f.Report)
	}
}

// The root error view's report (errorReport, __root.tsx:307-321): tern and
// its version, the time, then the error.
func TestSessionReport(t *testing.T) {
	r := sessionReport(errors.New("List environments failed (500)"), time.Date(2026, 11, 18, 17, 27, 22, 221_000_000, time.UTC))
	lines := strings.Split(r, "\n")
	if !strings.HasPrefix(lines[0], "tern ") || lines[1] != "Time: 2026-11-18T17:27:22.221Z" || lines[2] != "" || lines[3] != "List environments failed (500)" {
		t.Errorf("report:\n%s", r)
	}
}
