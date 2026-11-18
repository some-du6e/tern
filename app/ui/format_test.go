package ui

import (
	"testing"
	"time"
)

// T3's own cases (orchestrationTiming.test.ts), and the doc's examples.
func TestFormatDuration(t *testing.T) {
	for _, c := range []struct {
		ms   int64
		want string
	}{
		{0, "1ms"}, {340, "340ms"}, {3449, "3.4s"}, {3000, "3s"}, {42_000, "42s"},
		{59_000, "59s"}, {59_500, "1m 0s"}, {60_000, "1m 0s"}, {720_000, "12m 0s"}, {736_000, "12m 16s"},
		{3_600_000, "1h"}, {3_660_000, "1h 1m 0s"}, {3_723_000, "1h 2m 3s"}, {2_050_000, "34m 10s"},
	} {
		if got := FormatDuration(time.Duration(c.ms) * time.Millisecond); got != c.want {
			t.Errorf("FormatDuration(%dms) = %q, want %q", c.ms, got, c.want)
		}
	}
	if got := FormatDuration(-time.Second); got != "0ms" {
		t.Errorf("a negative duration is %q, want 0ms", got)
	}
}

// The working timer counts whole seconds, then switches to FormatDuration.
func TestWorkingTimer(t *testing.T) {
	start := time.Date(2026, 11, 17, 16, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		after time.Duration
		want  string
	}{{0, "0s"}, {12_900 * time.Millisecond, "12s"}, {59 * time.Second, "59s"}, {134 * time.Second, "2m 14s"}, {3723 * time.Second, "1h 2m 3s"}} {
		if got := workingTimer(start, start.Add(c.after)); got != c.want {
			t.Errorf("after %v: %q, want %q", c.after, got, c.want)
		}
	}
	if got := workingTimer(time.Time{}, start); got != "0s" {
		t.Errorf("before the start parses: %q, want 0s", got)
	}
}

func TestDayAwareTimestamp(t *testing.T) {
	now := time.Date(2026, 11, 17, 18, 30, 0, 0, time.Local)
	for _, c := range []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 11, 17, 16, 44, 0, 0, time.Local), "4:44 PM"},
		{time.Date(2026, 11, 16, 16, 44, 0, 0, time.Local), "yesterday at 4:44 PM"},
		{time.Date(2026, 11, 3, 9, 5, 0, 0, time.Local), "11/3 9:05 AM"},
		{time.Date(2025, 11, 3, 16, 44, 0, 0, time.Local), "11/3/2025 4:44 PM"},
	} {
		if got := dayAwareTimestamp(c.at, now); got != c.want {
			t.Errorf("%v: %q, want %q", c.at, got, c.want)
		}
	}
	if got := timestampTooltip(time.Date(2026, 11, 17, 16, 44, 0, 0, time.Local)); got != "4:44 PM, 17th November 2026" {
		t.Errorf("tooltip %q", got)
	}
	for day, want := range map[int]string{1: "st", 2: "nd", 3: "rd", 4: "th", 11: "th", 12: "th", 13: "th", 21: "st", 22: "nd", 23: "rd", 31: "st"} {
		if got := ordinal(day); got != want {
			t.Errorf("ordinal(%d) = %q, want %q", day, got, want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	for n, want := range map[int]string{
		512: "512", 2000: "2K", 2500: "2.50K", 11_700: "11.7K", 180_000: "180K", 12_000: "12K",
		1_500_000: "1.50M", 3_000_000_000: "3B", 1234: "1.23K",
	} {
		if got := FormatTokens(n); got != want {
			t.Errorf("FormatTokens(%d) = %q, want %q", n, got, want)
		}
	}
	if got := groupDigits(1234567); got != "1,234,567" {
		t.Errorf("groupDigits = %q", got)
	}
}

// T3's formatBytes (formatBytes.ts): the unit is picked after rounding.
func TestFormatBytes(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{{512, "512 B"}, {1024, "1.00 KB"}, {2516582, "2.40 MB"}, {1048166, "1.00 MB"}, {150 * 1024, "150 KB"}, {15 * 1024 * 1024, "15.0 MB"}} {
		if got := FormatBytes(c.n); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
