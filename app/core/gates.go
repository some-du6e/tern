package core

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// What tern shows before the app can, and over it (docs/ui/11-empty-auth.md):
// the gate and the toasts.

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

// ---------- toasts ----------

// toastLocked puts t up with an id of its own. Call with the lock held.
func (a *App) toastLocked(t ui.Toast) {
	a.toastN++
	t.ID = strconv.Itoa(a.toastN)
	a.toasts = append(a.toasts, t)
}

// setToastLocked puts t up in place of the toast with its id, keeping
// whether its details are open, or adds it; nil t closes id. Call with
// the lock held.
func (a *App) setToastLocked(id string, t *ui.Toast) {
	for i := range a.toasts {
		if a.toasts[i].ID != id {
			continue
		}
		if t == nil {
			a.toasts = append(a.toasts[:i], a.toasts[i+1:]...)
			return
		}
		t.Open = a.toasts[i].Open
		a.toasts[i] = *t
		return
	}
	if t != nil {
		a.toasts = append(a.toasts, *t)
	}
}

// closeToast closes the toast id, by its close orb or its time running out.
func (a *App) closeToast(id string) {
	a.mu.Lock()
	if id == slowToastID {
		// Closed by hand, it stays closed until no request is slow.
		a.slowClosed = true
	}
	a.setToastLocked(id, nil)
	a.mu.Unlock()
}

// toggleToast opens or folds the toast id's details.
func (a *App) toggleToast(id string) {
	a.mu.Lock()
	for i := range a.toasts {
		if a.toasts[i].ID == id {
			a.toasts[i].Open = !a.toasts[i].Open
		}
	}
	a.mu.Unlock()
}

const slowToastID = "slow"

// slowChanged keeps one toast up while any call waits past its threshold
// (SlowRpcRequestToastCoordinator.tsx): sticky, updated in place as the
// set changes, closed when it empties. Any goroutine may call it.
func (a *App) slowChanged() {
	a.mu.Lock()
	var slow []t3.SlowRequest
	for _, e := range a.envs {
		if e.conn != nil {
			slow = append(slow, e.conn.Slow()...)
		}
	}
	sort.SliceStable(slow, func(i, j int) bool { return slow[i].StartedAt.Before(slow[j].StartedAt) })
	switch {
	case len(slow) == 0:
		a.slowClosed = false
		a.setToastLocked(slowToastID, nil)
	case !a.slowClosed:
		a.setToastLocked(slowToastID, slowToast(slow))
	}
	a.mu.Unlock()
	a.changed()
}

// slowToast is the warning for the calls in slow: how many wait longer
// than the least threshold among them, and each one's tag and when it
// went out, in the details the description folds open.
func slowToast(slow []t3.SlowRequest) *ui.Toast {
	least := slow[0].Threshold
	for _, r := range slow {
		least = min(least, r.Threshold)
	}
	requests := strconv.Itoa(len(slow)) + " requests"
	if len(slow) == 1 {
		requests = "1 request"
	}
	t := &ui.Toast{
		ID:          slowToastID,
		Type:        ui.ToastWarning,
		Title:       "Some requests are slow",
		Description: fmt.Sprintf("%s waiting longer than %ds.", requests, int(math.Round(least.Seconds()))),
		Sticky:      true,
		Folded:      "requests",
	}
	for _, r := range slow {
		t.Details = append(t.Details, ui.ToastDetail{Title: r.Tag, Note: "Started " + r.StartedAt.Format("3:04:05 PM")})
	}
	return t
}

// freezeToast is the warning that host froze (hostFreeze.ts:12-25): for
// how long, and whether its CPU was overloaded or T3 Code too busy.
func freezeToast(host string, st *t3.EventLoopStall) ui.Toast {
	desc := "T3 Code on " + host + " was too busy to answer."
	if st.LoadAverage != nil && *st.LoadAverage > float64(st.CPUCount) {
		desc = fmt.Sprintf("Its CPU is overloaded (load %d on %d cores).", int(math.Round(*st.LoadAverage)), st.CPUCount)
	}
	return ui.Toast{
		Type:        ui.ToastWarning,
		Title:       fmt.Sprintf("%s froze for %ds", host, int(math.Round(float64(st.DurationMs)/1000))),
		Description: desc,
	}
}
