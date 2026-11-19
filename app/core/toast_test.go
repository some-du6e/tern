package core

import (
	"testing"
	"time"

	"github.com/some-du6e/tern/app/t3"
	"github.com/some-du6e/tern/app/ui"
)

// The slow requests' toast (SlowRpcRequestToastCoordinator.tsx): how many
// wait past the least threshold, and each with when it went out.
func TestSlowToast(t *testing.T) {
	at := time.Date(2026, 1, 2, 15, 4, 5, 0, time.Local)
	one := slowToast([]t3.SlowRequest{{Tag: "orchestration.getTurnItem", StartedAt: at, Threshold: 15 * time.Second}})
	if one.Title != "Some requests are slow" || one.Description != "1 request waiting longer than 15s." || !one.Sticky || one.Type != ui.ToastWarning {
		t.Errorf("one: %+v", one)
	}
	two := slowToast([]t3.SlowRequest{
		{Tag: "server.refreshProviders", StartedAt: at, Threshold: 2 * time.Minute},
		{Tag: "server.getConfig", StartedAt: at.Add(time.Second), Threshold: 15 * time.Second},
	})
	if two.Description != "2 requests waiting longer than 15s." || len(two.Details) != 2 || two.Details[0].Note != "Started 3:04:05 PM" || two.Folded != "requests" {
		t.Errorf("two: %+v", two)
	}
}

// The freeze toast (hostFreeze.ts): rounded seconds, and the load only
// when it's over the core count.
func TestFreezeToast(t *testing.T) {
	load := 9.4
	got := freezeToast("studio", &t3.EventLoopStall{DurationMs: 12_400, LoadAverage: &load, CPUCount: 4})
	if got.Title != "studio froze for 12s" || got.Description != "Its CPU is overloaded (load 9 on 4 cores)." {
		t.Errorf("overloaded: %+v", got)
	}
	got = freezeToast("studio", &t3.EventLoopStall{DurationMs: 2_600, CPUCount: 4})
	if got.Title != "studio froze for 3s" || got.Description != "T3 Code on studio was too busy to answer." {
		t.Errorf("busy: %+v", got)
	}
}

// A repeated slow toast stays up as one, and closed by hand it stays
// closed until no request is slow.
func TestSlowToastStaysClosed(t *testing.T) {
	a := testApp(t)
	t0 := slowToast([]t3.SlowRequest{{Tag: "server.getConfig", StartedAt: time.Now(), Threshold: 15 * time.Second}})
	a.setToastLocked(slowToastID, t0)
	a.setToastLocked(slowToastID, t0)
	if s := a.build(time.Now()); len(s.Toasts) != 1 {
		t.Fatalf("%d toasts", len(s.Toasts))
	}
	a.closeToast(slowToastID)
	a.slowChanged()
	if s := a.build(time.Now()); len(s.Toasts) != 0 || a.slowClosed {
		t.Errorf("after closing with none slow: %d toasts, closed %v", len(s.Toasts), a.slowClosed)
	}
}
