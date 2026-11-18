package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// The timeline's formats (docs/ui/06-timeline.md §5.4). Timestamps follow
// T3's "locale" setting as an en-US browser renders it: 12-hour clock,
// numeric month and day.

// FormatDuration is T3's formatDuration (orchestrationTiming.ts:22-37):
// "340ms", "3.4s", "42s", then "2m 0s", "1h", "1h 2m 3s", with seconds
// shown whenever minutes are.
func FormatDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	if ms < 0 {
		return "0ms"
	}
	if ms < 1000 {
		return strconv.Itoa(max(1, int(math.Round(ms)))) + "ms"
	}
	if ms < 10_000 {
		return strconv.FormatFloat(math.Round(ms/100)/10, 'f', -1, 64) + "s"
	}
	// The unit is chosen after rounding, so 59.5 seconds is never "60s".
	total := int(math.Round(ms / 1000))
	if total < 60 {
		return strconv.Itoa(total) + "s"
	}
	h, m, s := total/3600, total%3600/60, total%60
	var parts []string
	if h > 0 {
		parts = append(parts, strconv.Itoa(h)+"h")
	}
	if m > 0 {
		parts = append(parts, strconv.Itoa(m)+"m")
	}
	if s > 0 || m > 0 {
		parts = append(parts, strconv.Itoa(s)+"s")
	}
	return strings.Join(parts, " ")
}

// workingTimer is the working row's clock (TimelineWorkGroupRows.logic.ts):
// whole seconds under a minute, then FormatDuration of whole seconds.
func workingTimer(since, now time.Time) string {
	if since.IsZero() {
		return "0s"
	}
	s := max(0, int(now.Sub(since)/time.Second))
	if s < 60 {
		return strconv.Itoa(s) + "s"
	}
	return FormatDuration(time.Duration(s) * time.Second)
}

// clockTime is "4:44 PM".
func clockTime(t time.Time) string { return t.Format("3:04 PM") }

// dayAwareTimestamp is T3's formatDayAwareTimestamp (timestampFormat.ts:
// 184-201): "4:44 PM" today, "yesterday at 4:44 PM", "11/3 4:44 PM" this
// year, "11/3/2025 4:44 PM" before.
func dayAwareTimestamp(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	t, now = t.Local(), now.Local()
	days := calendarDays(now) - calendarDays(t)
	switch {
	case days <= 0:
		return clockTime(t)
	case days == 1:
		return "yesterday at " + clockTime(t)
	case t.Year() == now.Year():
		return fmt.Sprintf("%d/%d %s", t.Month(), t.Day(), clockTime(t))
	}
	return fmt.Sprintf("%d/%d/%d %s", t.Month(), t.Day(), t.Year(), clockTime(t))
}

// calendarDays numbers t's calendar day, so differences are whole days
// across daylight saving shifts.
func calendarDays(t time.Time) int {
	return int(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// timestampTooltip is T3's formatChatTimestampTooltip: "4:44 PM, 17th
// November 2026", English ordinals and months always.
func timestampTooltip(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	t = t.Local()
	return fmt.Sprintf("%s, %d%s %s %d", clockTime(t), t.Day(), ordinal(t.Day()), t.Month(), t.Year())
}

func ordinal(day int) string {
	if n := day % 100; n >= 11 && n <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	}
	return "th"
}

// FormatTokens is T3's formatTokens (usageFormat.ts:66-79): "180K",
// "11.7K", "2.50K", with two decimals under 10, one under 100 and only
// all-zero decimals dropped.
func FormatTokens(n int) string {
	v := float64(n)
	for _, u := range []struct {
		at   float64
		unit string
	}{{1e12, "T"}, {1e9, "B"}, {1e6, "M"}, {1e3, "K"}} {
		if math.Abs(v) >= u.at {
			x := v / u.at
			digits := 2
			switch {
			case math.Abs(x) >= 100:
				digits = 0
			case math.Abs(x) >= 10:
				digits = 1
			}
			s := strconv.FormatFloat(x, 'f', digits, 64)
			if i := strings.IndexByte(s, '.'); i >= 0 && strings.Trim(s[i+1:], "0") == "" {
				s = s[:i]
			}
			return s + u.unit
		}
	}
	return groupDigits(n)
}

// FormatBytes is T3's formatBytes (formatBytes.ts): "512 B", then KB, MB,
// GB and TB at base 1024 with 2, 1 or no decimals, the unit picked after
// rounding.
func FormatBytes(n int) string {
	if n < 1024 {
		return strconv.Itoa(n) + " B"
	}
	scaled := func(v float64) string {
		digits := 2
		switch {
		case v >= 100:
			digits = 0
		case v >= 10:
			digits = 1
		}
		return strconv.FormatFloat(v, 'f', digits, 64)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	v, u := float64(n)/1024, 0
	for u < len(units)-1 {
		if r, _ := strconv.ParseFloat(scaled(v), 64); r < 1024 {
			break
		}
		v /= 1024
		u++
	}
	return scaled(v) + " " + units[u]
}

// groupDigits is n with thousands separators, as en-US writes it.
func groupDigits(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
