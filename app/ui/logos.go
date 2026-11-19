package ui

import (
	"image"
	"math"
	"strconv"

	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Provider marks, 24x24 path data from Simple Icons (simpleicons.org, CC0;
// the marks themselves belong to Anthropic, OpenAI and Google).
const (
	logoClaude = "m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"
	logoOpenAI = "M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"
	logoGemini = "M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81"
)

var logoPaths = map[string][]seg{
	"Claude": parseSVGPath(logoClaude),
	"Codex":  parseSVGPath(logoOpenAI),
	"Gemini": parseSVGPath(logoGemini),
}

// ProviderLogo draws a provider's official mark in a sizeDp box. Unknown
// providers get the generic spark.
func ProviderLogo(provider string, sizeDp float32) W {
	return func(gtx C) D {
		segs, ok := logoPaths[provider]
		if !ok {
			return DrawIcon(gtx, IconSpark, ColViolet, sizeDp)
		}
		s := float32(dp(gtx, sizeDp))
		k := s / 24
		var p clip.Path
		p.Begin(gtx.Ops)
		for _, sg := range segs {
			pt := func(i int) f32.Point { return f32.Pt(sg.p[i].X*k, sg.p[i].Y*k) }
			switch sg.op {
			case 'M':
				p.MoveTo(pt(0))
			case 'L':
				p.LineTo(pt(0))
			case 'Q':
				p.QuadTo(pt(0), pt(1))
			case 'C':
				p.CubeTo(pt(0), pt(1), pt(2))
			case 'Z':
				p.Close()
			}
		}
		st := clip.Outline{Path: p.End()}.Op().Push(gtx.Ops)
		switch provider {
		case "Claude":
			paint.ColorOp{Color: rgb(0xd97757)}.Add(gtx.Ops)
		case "Gemini":
			paint.LinearGradientOp{
				Stop1: f32.Pt(s*0.15, s*0.85), Color1: rgb(0x4285f4),
				Stop2: f32.Pt(s*0.85, s*0.15), Color2: rgb(0xa47cf3),
			}.Add(gtx.Ops)
		default:
			paint.ColorOp{Color: ColText}.Add(gtx.Ops)
		}
		paint.PaintOp{}.Add(gtx.Ops)
		st.Pop()
		return D{Size: image.Pt(int(s), int(s))}
	}
}

// seg is one absolute path command: M, L, Q, C or Z.
type seg struct {
	op byte
	p  [3]f32.Point
}

// parseSVGPath turns SVG path data into absolute M/L/Q/C/Z segments. It
// covers the whole command set, arcs included, since icon sets use them.
func parseSVGPath(d string) []seg {
	var out []seg
	i := 0
	skip := func() {
		for i < len(d) && (d[i] == ' ' || d[i] == ',' || d[i] == '\n' || d[i] == '\t') {
			i++
		}
	}
	num := func() float32 {
		skip()
		j := i
		if j < len(d) && (d[j] == '-' || d[j] == '+') {
			j++
		}
		dot, exp := false, false
		for j < len(d) {
			c := d[j]
			switch {
			case c >= '0' && c <= '9':
			case c == '.' && !dot && !exp:
				dot = true
			case (c == 'e' || c == 'E') && !exp:
				exp = true
				if j+1 < len(d) && (d[j+1] == '-' || d[j+1] == '+') {
					j++
				}
			default:
				goto done
			}
			j++
		}
	done:
		v, _ := strconv.ParseFloat(d[i:j], 32)
		i = j
		return float32(v)
	}
	flag := func() bool {
		skip()
		f := d[i] == '1'
		i++
		return f
	}
	more := func() bool {
		skip()
		if i >= len(d) {
			return false
		}
		c := d[i]
		return c == '-' || c == '+' || c == '.' || (c >= '0' && c <= '9')
	}
	var cur, start, lastCtl f32.Point
	var prev byte
	var cmd byte
	for {
		skip()
		if i >= len(d) {
			break
		}
		if c := d[i]; (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			cmd = c
			i++
		}
		rel := cmd >= 'a'
		abs := func(x, y float32) f32.Point {
			if rel {
				return f32.Pt(cur.X+x, cur.Y+y)
			}
			return f32.Pt(x, y)
		}
		up := cmd &^ 0x20
		for first := true; first || more(); first = false {
			if up == 'Z' && !first {
				break
			}
			switch up {
			case 'M':
				cur = abs(num(), num())
				start = cur
				out = append(out, seg{op: 'M', p: [3]f32.Point{cur}})
				up, cmd = 'L', cmd-('M'-'L') // later pairs are implicit lineto
			case 'L':
				cur = abs(num(), num())
				out = append(out, seg{op: 'L', p: [3]f32.Point{cur}})
			case 'H':
				x := num()
				if rel {
					x += cur.X
				}
				cur = f32.Pt(x, cur.Y)
				out = append(out, seg{op: 'L', p: [3]f32.Point{cur}})
			case 'V':
				y := num()
				if rel {
					y += cur.Y
				}
				cur = f32.Pt(cur.X, y)
				out = append(out, seg{op: 'L', p: [3]f32.Point{cur}})
			case 'C':
				c1, c2 := abs(num(), num()), abs(num(), num())
				cur = abs(num(), num())
				lastCtl = c2
				out = append(out, seg{op: 'C', p: [3]f32.Point{c1, c2, cur}})
			case 'S':
				c1 := cur
				if prev == 'C' || prev == 'S' {
					c1 = cur.Mul(2).Sub(lastCtl)
				}
				c2 := abs(num(), num())
				cur = abs(num(), num())
				lastCtl = c2
				out = append(out, seg{op: 'C', p: [3]f32.Point{c1, c2, cur}})
			case 'Q':
				c := abs(num(), num())
				cur = abs(num(), num())
				lastCtl = c
				out = append(out, seg{op: 'Q', p: [3]f32.Point{c, cur}})
			case 'T':
				c := cur
				if prev == 'Q' || prev == 'T' {
					c = cur.Mul(2).Sub(lastCtl)
				}
				cur = abs(num(), num())
				lastCtl = c
				out = append(out, seg{op: 'Q', p: [3]f32.Point{c, cur}})
			case 'A':
				rx, ry, rot := num(), num(), num()
				large, sweep := flag(), flag()
				to := abs(num(), num())
				out = append(out, arcSegs(cur, to, rx, ry, rot, large, sweep)...)
				cur = to
			case 'Z':
				cur = start
				out = append(out, seg{op: 'Z'})
			}
			prev = up
		}
	}
	return out
}

// arcSegs converts an SVG elliptical arc to cubic Béziers, at most a quarter
// turn each (SVG 1.1 appendix F.6).
func arcSegs(from, to f32.Point, rx, ry, rotDeg float32, large, sweep bool) []seg {
	if rx == 0 || ry == 0 {
		return []seg{{op: 'L', p: [3]f32.Point{to}}}
	}
	x1, y1, x2, y2 := float64(from.X), float64(from.Y), float64(to.X), float64(to.Y)
	r1, r2 := math.Abs(float64(rx)), math.Abs(float64(ry))
	phi := float64(rotDeg) * math.Pi / 180
	cp, sp := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p, y1p := cp*dx+sp*dy, -sp*dx+cp*dy
	if l := x1p*x1p/(r1*r1) + y1p*y1p/(r2*r2); l > 1 {
		r1, r2 = r1*math.Sqrt(l), r2*math.Sqrt(l)
	}
	num := r1*r1*r2*r2 - r1*r1*y1p*y1p - r2*r2*x1p*x1p
	den := r1*r1*y1p*y1p + r2*r2*x1p*x1p
	co := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		co = -co
	}
	cxp, cyp := co*r1*y1p/r2, -co*r2*x1p/r1
	cx, cy := cp*cxp-sp*cyp+(x1+x2)/2, sp*cxp+cp*cyp+(y1+y2)/2
	ang := func(ux, uy, vx, vy float64) float64 { return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy) }
	t1 := ang(1, 0, (x1p-cxp)/r1, (y1p-cyp)/r2)
	dt := ang((x1p-cxp)/r1, (y1p-cyp)/r2, (-x1p-cxp)/r1, (-y1p-cyp)/r2)
	if !sweep && dt > 0 {
		dt -= 2 * math.Pi
	} else if sweep && dt < 0 {
		dt += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dt) / (math.Pi / 2)))
	step := dt / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	pt := func(t float64) (float64, float64) {
		x, y := r1*math.Cos(t), r2*math.Sin(t)
		return cp*x - sp*y + cx, sp*x + cp*y + cy
	}
	der := func(t float64) (float64, float64) {
		x, y := -r1*math.Sin(t), r2*math.Cos(t)
		return cp*x - sp*y, sp*x + cp*y
	}
	var out []seg
	for j := 0; j < n; j++ {
		a, b := t1+float64(j)*step, t1+float64(j+1)*step
		ax, ay := pt(a)
		bx, by := pt(b)
		dax, day := der(a)
		dbx, dby := der(b)
		out = append(out, seg{op: 'C', p: [3]f32.Point{
			f32.Pt(float32(ax+k*dax), float32(ay+k*day)),
			f32.Pt(float32(bx-k*dbx), float32(by-k*dby)),
			f32.Pt(float32(bx), float32(by)),
		}})
	}
	if n > 0 {
		out[len(out)-1].p[2] = to
	}
	return out
}
