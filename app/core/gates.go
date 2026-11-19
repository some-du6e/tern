package core

import (
	"errors"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// What tern shows before the app can (docs/ui/11-empty-auth.md): the
// splash, the boot failure page, the root error view, the sign-in and the
// pairing page.

// gateLocked is the window's gate. Call with the lock held.
func (a *App) gateLocked() ui.Gate {
	switch {
	case !a.booted:
		return ui.GateBooting
	case a.failure != nil:
		return ui.GateSessionError
	case !a.signedIn && a.pairing:
		return ui.GatePairing
	case !a.signedIn:
		return ui.GateSignIn
	}
	return ui.GateApp
}

// version is tern's build, as its module or its commit names it.
func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	v := bi.Main.Version
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 && (v == "" || v == "(devel)") {
			v = s.Value[:7]
		}
	}
	return v
}

// iso is t as JavaScript's toISOString writes it, which T3's reports use.
func iso(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// causes is err's text, then each error it wraps whose text isn't already
// in the one before, at most n of them: Go's errors repeat what they wrap.
func causes(err error, n int) []string {
	out := []string{err.Error()}
	for c := errors.Unwrap(err); c != nil && len(out) <= n; c = errors.Unwrap(c) {
		if !strings.Contains(out[len(out)-1], c.Error()) {
			out = append(out, c.Error())
		}
	}
	return out
}

// sessionReport is the root error view's report (errorReport,
// __root.tsx:307-321): tern and its version, the time, then the error and
// up to 5 causes. tern has no route to name, so it has no Path line.
func sessionReport(err error, now time.Time) string {
	lines := []string{"tern " + version(), "Time: " + iso(now), ""}
	for i, c := range causes(err, 5) {
		if i > 0 {
			lines = append(lines, "", "Caused by:")
		}
		lines = append(lines, c)
	}
	return strings.Join(lines, "\n")
}

var (
	reportHost  = regexp.MustCompile(`(?i)\b[a-z][\w+.-]*://[^\s/?#'"()]+`)
	reportQuery = regexp.MustCompile(`[?#][^\s'"()]*=[^\s'"()]*`)
	lineCol     = regexp.MustCompile(`(?::\d+){1,2}$`)
)

// redact drops what can name a private machine or carry a token from a
// report: hosts in URLs, and query strings and fragments with values
// (bootError.ts:79-85), keeping a trailing line:column.
func redact(s string) string {
	s = reportHost.ReplaceAllString(s, "")
	return reportQuery.ReplaceAllStringFunc(s, func(m string) string { return lineCol.FindString(m) })
}

// BootFailure is the boot failure page for err (bootError.ts:96-120): the
// error and up to 3 causes, redacted and cut at 6,000 characters, then the
// build, the platform and the time. repeat opens its "More options", as
// T3 does for a second failure within 5 minutes. tern has no channels,
// surfaces or user agent to report.
func BootFailure(err error, repeat bool, now time.Time) *ui.Failure {
	details := redact(strings.Join(causes(err, 3), "\nCaused by: "))
	if len(details) > 6000 {
		details = details[:6000] + "\n[truncated]"
	}
	report := strings.Join([]string{
		details,
		"",
		"Version: " + version(),
		"Platform: " + runtime.GOOS + " " + runtime.GOARCH,
		"Time: " + iso(now),
	}, "\n")
	return &ui.Failure{Report: report, Open: repeat}
}

// ---------- pairing ----------

// pairNext pairs with the link in the field (T3's /pair Continue). An
// empty field is refused here, before any request; on success the app
// comes up, where T3 reloads.
func (a *App) pairNext(th *ui.Theme) {
	a.mu.Lock()
	if a.pair.Submitting {
		a.mu.Unlock()
		return
	}
	link := strings.TrimSpace(th.EditorText("pair.link"))
	a.pair = ui.Pairing{Submitting: link != ""}
	if link == "" {
		a.pair.Error = "Enter a pairing link to continue."
	}
	a.mu.Unlock()
	a.changed()
	if link == "" {
		return
	}
	go func() {
		_, err := t3.Pair(a.store, link)
		if err == nil {
			a.onUI(func(th *ui.Theme) {
				th.SetEditorText("pair.link", "")
				th.Focus("composer")
			})
			a.boot()
		}
		a.mu.Lock()
		a.pair = ui.Pairing{}
		if err != nil {
			a.pair.Error = pairingError(err)
		} else {
			a.pairing = false
		}
		a.mu.Unlock()
		a.changed()
	}()
}

// pairingError is the form's Alert for a pairing that failed: T3's words
// for a link without its host or token and for a refused token
// (PairingRouteSurface.tsx:145-190, auth.ts:62-102), else the failure
// and, as T3's hosted page says it, what to do if the token was spent.
func pairingError(err error) string {
	var te *t3.Error
	switch {
	case errors.Is(err, t3.ErrPairingLink):
		return "This pairing link is missing its backend host or token."
	case errors.As(err, &te) && te.What == "Pairing" && te.Status >= 400 && te.Status < 500:
		return "Invalid pairing token. Check the token and try again."
	}
	return sentence(err.Error()) + " If the backend accepted this one-time token, request a new pairing link before retrying."
}
