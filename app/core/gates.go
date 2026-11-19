package core

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
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
// the gate, the banner over the composer for a computer that's gone, and
// the toasts.

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

// ---------- the connection banner ----------

// reconnectGrace is how long a computer may be reconnecting before the
// banner says so (ENVIRONMENT_RECONNECT_WARNING_GRACE_MS).
const reconnectGrace = 2 * time.Second

// connFailure is how an environment's last connection failed, as T3
// names it (ConnectionAttemptError, errors.ts): the part of the way that
// broke and why, when known, the kind of failure, and whether it's a
// refusal (ConnectionBlockedError), which waiting won't fix.
type connFailure struct {
	hop     string // "device", "relay", "tunnel" or "server"
	cause   string // the relay's endpoint failure, or a cause tern can name
	reason  string // "network", "timeout", "authentication", "permission"…
	blocked bool
}

var errPairingExpired = errors.New("the pairing session expired; pair again")

// failureOf names err, a failed connection to an environment, the way
// T3's relay and environment errors do (errors.ts:40-260). Over T3
// Connect the relay has already reached the host, so a request that
// never arrives broke between this device and the tunnel; a paired
// computer is reached directly.
func failureOf(err error, paired bool) *connFailure {
	unreachable := "tunnel"
	if paired {
		unreachable = "server"
	}
	var te *t3.Error
	if errors.As(err, &te) {
		tag, _ := te.Body["_tag"].(string)
		why, _ := te.Body["reason"].(string)
		switch tag {
		case "RelayAuthInvalidError":
			f := &connFailure{hop: "relay", reason: "authentication", blocked: true}
			if te.Body["dpopFailureReason"] == "time_window" {
				f.cause = "clock_skew"
			}
			return f
		case "RelayEnvironmentLinkProofExpiredError":
			return &connFailure{hop: "relay", reason: "authentication", blocked: true}
		case "RelayEnvironmentConnectNotAuthorizedError":
			switch why {
			case "managed_endpoint_allocation_not_ready":
				return &connFailure{hop: "tunnel", cause: "tunnel_pending", reason: "endpoint-unavailable"}
			case "endpoint_provider_not_managed":
				return &connFailure{hop: "tunnel", cause: "tunnel_not_enabled", reason: "permission", blocked: true}
			case "environment_link_not_found":
				return &connFailure{hop: "relay", cause: "not_linked", reason: "permission", blocked: true}
			}
			return &connFailure{hop: "relay", reason: "permission", blocked: true}
		case "RelayEnvironmentLinkProofInvalidError", "RelayEnvironmentLinkLimitExceededError":
			return &connFailure{hop: "relay", reason: "permission", blocked: true}
		case "RelayEnvironmentEndpointTimedOutError":
			return &connFailure{hop: "tunnel", cause: "timeout", reason: "timeout"}
		case "RelayEnvironmentEndpointUnavailableError":
			cause, _ := te.Body["endpointFailure"].(string)
			if cause == "" && why == "endpoint_response_invalid" {
				cause = "invalid_response"
			}
			hop := "tunnel"
			if cause == "origin_unreachable" || cause == "invalid_response" {
				hop = "server"
			}
			return &connFailure{hop: hop, cause: cause, reason: "endpoint-unavailable"}
		case "RelayEnvironmentLinkUnavailableError":
			return &connFailure{hop: "tunnel", reason: "endpoint-unavailable"}
		case "RelayEnvironmentLinkFailedError", "RelayInternalError":
			return &connFailure{hop: "relay", cause: "relay_error", reason: "relay-unavailable"}
		}
		if strings.HasPrefix(te.What, "Relay") {
			// The relay answered, with nothing tern knows more about.
			return &connFailure{hop: "relay", cause: "relay_error", reason: "relay-unavailable"}
		}
		// The environment's own answer (mapRemoteEnvironmentError).
		switch te.Status {
		case 401:
			return &connFailure{hop: "server", reason: "authentication", blocked: true}
		case 403:
			return &connFailure{hop: "server", reason: "permission", blocked: true}
		case 400, 404:
			return &connFailure{hop: "server", reason: "configuration", blocked: true}
		}
		return &connFailure{hop: "server", reason: "remote-unavailable"}
	}
	if errors.Is(err, errPairingExpired) {
		return &connFailure{hop: "server", reason: "authentication", blocked: true}
	}
	timeout := errors.Is(err, context.DeadlineExceeded)
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		timeout = true
	}
	var ue *url.Error
	if errors.As(err, &ue) && strings.HasPrefix(ue.URL, t3.RelayURL) {
		if timeout {
			return &connFailure{hop: "relay", reason: "timeout"}
		}
		return &connFailure{hop: "relay", reason: "relay-unavailable"}
	}
	if timeout {
		return &connFailure{hop: unreachable, reason: "timeout"}
	}
	return &connFailure{hop: unreachable, reason: "network"}
}

// bannerLocked is the composer's banner for the open thread's or draft's
// computer while it isn't connected (useSystemComposerBanners.tsx:150-216):
// once it has been trying for 2s, reconnecting, with Retry now after a
// failure (none on the first try); at once, offline in the error colours
// with Reconnect, when it was refused. Call with the lock held.
func (a *App) bannerLocked(now time.Time) *ui.Banner {
	e := a.cur()
	if e == nil || e.conn != nil {
		return nil
	}
	host := e.env.Label
	if f := e.failed; f != nil && f.blocked {
		return &ui.Banner{Error: true, Title: host + " is offline", Detail: diagnosis(e), Action: "Reconnect"}
	}
	if e.tryingAt.IsZero() || now.Sub(e.tryingAt) < reconnectGrace {
		return nil
	}
	b := &ui.Banner{Title: host + " is reconnecting", Detail: diagnosis(e)}
	if e.failed != nil {
		b.Action = "Retry now"
	}
	return b
}

// trying marks e as trying to connect, from connected or from a refusal,
// and redraws when the banner's grace is up. Call with the lock held.
func (a *App) trying(e *envState, now time.Time) {
	if e.tryingAt.IsZero() {
		e.tryingAt = now
		time.AfterFunc(reconnectGrace, a.changed)
	}
}

// retryNow is the banner's Retry now and Reconnect: a fresh attempt now,
// the backoff started over.
func (a *App) retryNow() {
	a.mu.Lock()
	e := a.cur()
	if e != nil {
		e.retry = 0
	}
	a.mu.Unlock()
	if e != nil {
		a.connect(e)
	}
}

// diagnosis is the banner's description, the headline for where e's
// connection broke (diagnosis.ts:65-157): from its last failure, or the
// relay's last word on it.
func diagnosis(e *envState) string {
	host := e.env.Label
	var hop, cause, reason string
	switch f := e.failed; {
	case f != nil:
		hop, cause, reason = f.hop, f.cause, f.reason
	case e.relay.Status == "offline":
		hop, cause = "tunnel", e.relay.EndpointFailure
		switch {
		case e.relay.OfflineReason == "tunnel_released":
			cause = "released"
		case cause == "origin_unreachable" || cause == "invalid_response":
			hop = "server"
		}
	default:
		return ""
	}
	switch cause {
	case "clock_skew":
		return "This device's clock is off"
	case "not_linked":
		return host + " is not linked to your T3 Connect account"
	case "relay_error":
		return "T3 Connect returned an error"
	case "tunnel_pending":
		return host + "'s tunnel is still being set up"
	case "tunnel_not_enabled":
		return "T3 Connect is not turned on for " + host
	case "released", "tunnel_down":
		return host + "'s tunnel is offline"
	case "origin_unreachable":
		return "Tunnel works, but T3 Code on " + host + " did not answer"
	case "invalid_response":
		return "T3 Code on " + host + " sent an unexpected answer"
	case "dns":
		return host + "'s tunnel address did not resolve"
	case "tls":
		return "Secure connection to " + host + "'s tunnel failed"
	case "refused":
		return host + "'s tunnel refused the connection"
	case "timeout":
		return host + " did not answer in time"
	}
	unreachable := reason == "network" || reason == "transport" || reason == "timeout"
	switch hop {
	case "device":
		return "This device could not prepare the connection"
	case "relay":
		if reason == "authentication" {
			return "T3 Connect did not accept this device's sign-in"
		}
		return "This device can't reach T3 Connect"
	case "tunnel":
		if unreachable {
			return "This device can't reach " + host + "'s tunnel"
		}
		return "T3 Connect could not reach " + host + "'s tunnel"
	}
	if reason == "authentication" || reason == "permission" {
		return "T3 Code on " + host + " did not accept this device"
	}
	return "T3 Code on " + host + " did not answer"
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
