package ui

import (
	"time"

	"gioui.org/op"
)

// T3's transitions (docs/ui/01-tokens.md §6): 150ms on Tailwind's
// ease-in-out unless a component says otherwise.

const transitionDur = 150 * time.Millisecond

// easeInOut is Tailwind's ease-in-out, cubic-bezier(0.4, 0, 0.2, 1).
func easeInOut(t float32) float32 { return cubicBezier(0.4, 0, 0.2, 1, t) }

// easeOutCSS is Tailwind's ease-out, cubic-bezier(0, 0, 0.2, 1).
func easeOutCSS(t float32) float32 { return cubicBezier(0, 0, 0.2, 1, t) }

// cubicBezier is the CSS timing function through (x1, y1) and (x2, y2) at
// progress t: it finds the curve's parameter for x = t by bisection, then
// reads y there.
func cubicBezier(x1, y1, x2, y2, t float32) float32 {
	if t <= 0 || t >= 1 {
		return clamp01(t)
	}
	at := func(a, b, s float32) float32 { return 3*a*s*(1-s)*(1-s) + 3*b*s*s*(1-s) + s*s*s }
	lo, hi := float32(0), float32(1)
	for i := 0; i < 24; i++ {
		mid := (lo + hi) / 2
		if at(x1, x2, mid) < t {
			lo = mid
		} else {
			hi = mid
		}
	}
	return at(y1, y2, (lo+hi)/2)
}

// fade is one value on its way to 1 (on) or 0, from where it was at.
type fade struct {
	on   bool
	from float32
	at   time.Time
}

func level(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

// value is where f is at now, a transition of d end to end.
func (f *fade) value(now time.Time, d time.Duration) float32 {
	to := level(f.on)
	span := to - f.from
	if span < 0 {
		span = -span
	}
	if span == 0 {
		return to
	}
	return f.from + (to-f.from)*easeInOut(since(now, f.at, time.Duration(float32(d)*span)))
}

// transition is a CSS transition of a value toward 1 while on and 0 while
// off, over d with T3's ease-in-out. One that reverses part way runs back
// from where it got to, in proportionally less time, as Chrome's do. A
// value starts where it is: nothing animates the first time it's drawn.
func (th *Theme) transition(gtx C, id string, on bool, d time.Duration) float32 {
	if th.in.fades == nil {
		th.in.fades = map[string]*fade{}
	}
	f := th.in.fades[id]
	if f == nil {
		f = &fade{on: on, from: level(on)}
		th.in.fades[id] = f
	}
	if f.on != on {
		f.from, f.on, f.at = f.value(gtx.Now, d), on, gtx.Now
	}
	v := f.value(gtx.Now, d)
	if v != level(on) {
		gtx.Execute(op.InvalidateCmd{})
	}
	return v
}
