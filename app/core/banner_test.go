package core

import (
	"errors"
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
)

// The open thread's computer's banner (docs/ui/11-empty-auth.md §2.18,
// §4.5): nothing for the first 2s of trying, then "is reconnecting", with
// Retry now once a try has failed; a refusal says "is offline" at once,
// in the error colours, with Reconnect.
func TestBanner(t *testing.T) {
	a := testApp(t)
	a.openThread("swiss/t1")
	e := a.env("swiss")
	now := time.Now()
	e.tryingAt = now.Add(-time.Second)
	if b := a.build(now).Banner; b != nil {
		t.Errorf("within the grace: %+v", b)
	}
	e.tryingAt = now.Add(-3 * time.Second)
	if b := a.build(now).Banner; b == nil || b.Title != "swiss is reconnecting" || b.Action != "" || b.Error {
		t.Errorf("first try: %+v", b)
	}
	e.failed = failureOf(errors.New("dial tcp: connection reset"), false)
	if b := a.build(now).Banner; b == nil || b.Action != "Retry now" || b.Detail != "This device can't reach swiss's tunnel" {
		t.Errorf("after a failure: %+v", b)
	}
	e.tryingAt = time.Time{}
	e.failed = failureOf(&t3.Error{What: "Relay /v1/environments/swiss/connect", Status: 403,
		Body: map[string]any{"_tag": "RelayEnvironmentConnectNotAuthorizedError", "reason": "environment_link_not_found"}}, false)
	b := a.build(now).Banner
	if b == nil || !b.Error || b.Title != "swiss is offline" || b.Action != "Reconnect" || b.Detail != "swiss is not linked to your T3 Connect account" {
		t.Errorf("refused: %+v", b)
	}
}

// The banner's description names where the connection broke
// (diagnosis.ts:98-157), from the last failure or, with none yet, the
// relay's last word on it.
func TestDiagnosis(t *testing.T) {
	relay := func(tag string, extra map[string]any) error {
		body := map[string]any{"_tag": tag}
		for k, v := range extra {
			body[k] = v
		}
		return &t3.Error{What: "Relay /v1/environments/e1/connect", Status: 503, Body: body}
	}
	for _, c := range []struct {
		err    error
		paired bool
		want   string
	}{
		{relay("RelayEnvironmentEndpointUnavailableError", map[string]any{"endpointFailure": "origin_unreachable"}), false, "Tunnel works, but T3 Code on studio did not answer"},
		{relay("RelayEnvironmentEndpointUnavailableError", map[string]any{"endpointFailure": "tunnel_down"}), false, "studio's tunnel is offline"},
		{relay("RelayEnvironmentEndpointTimedOutError", nil), false, "studio did not answer in time"},
		{relay("RelayInternalError", nil), false, "T3 Connect returned an error"},
		{relay("RelayAuthInvalidError", map[string]any{"dpopFailureReason": "time_window"}), false, "This device's clock is off"},
		{&t3.Error{What: "WebSocket ticket", Status: 401}, true, "T3 Code on studio did not accept this device"},
		{errors.New("dial tcp: connection refused"), true, "T3 Code on studio did not answer"},
	} {
		e := newEnv(t3.Environment{EnvironmentID: "e1", Label: "studio"}, c.paired)
		e.failed = failureOf(c.err, c.paired)
		if got := diagnosis(e); got != c.want {
			t.Errorf("%v: %q, want %q", c.err, got, c.want)
		}
	}
	e := newEnv(t3.Environment{EnvironmentID: "e1", Label: "studio"}, false)
	e.relay = t3.RelayStatus{Status: "offline", OfflineReason: "tunnel_released"}
	if got := diagnosis(e); got != "studio's tunnel is offline" {
		t.Errorf("released tunnel: %q", got)
	}
}
