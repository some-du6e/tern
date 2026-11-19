package t3

import (
	"testing"
	"time"
)

// T3 warns about a call with no answer after 15s, after 2min for the ones
// that wait on installs and the network, and never about subscriptions,
// pull requests or the reports nobody waits on (requestLatencyState.ts).
func TestSlowThresholds(t *testing.T) {
	for tag, want := range map[string]time.Duration{
		"server.getConfig":               15 * time.Second,
		"orchestration.getTurnItem":      15 * time.Second,
		"server.refreshProviders":        2 * time.Minute,
		"vcs.refreshStatus":              2 * time.Minute,
		"orchestration.subscribeThread":  0,
		"pullRequests.list":              0,
		"server.reportClientActivity":    0,
		"sourceControl.signInWait":       0,
		"server.getUsageSummary":         0,
		"terminal.subscribeTerminalData": 0,
	} {
		if got := slowThreshold(tag); got != want {
			t.Errorf("%s: %v, want %v", tag, got, want)
		}
	}
}
