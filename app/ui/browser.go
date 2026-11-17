package ui

import (
	"image"
	"image/color"
	"image/draw"
	"io"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
)

// The browser panel, T3's right panel: a tab in the thread machine's own
// browser, drawn from the frames it streams, with T3's address bar over
// it. Targets: "browser.toggle", "browser.back", "browser.forward",
// "browser.reload"; field "browser.url".

// BrowserView is the open thread's panel.
type BrowserView struct {
	URL, Title string
	Loading    bool
	Failed     bool
	CanBack    bool
	CanForward bool
	Frame      *Frame
	Viewport   image.Point // the page's size in CSS pixels
	Agent      bool        // the agent is driving the tab
	Error      string
	// Send is one input message to the page; Resize sizes it to the panel
	// in CSS pixels.
	Send   func(map[string]any)
	Resize func(w, h int)
	// The thread's tabs, and the dev servers a blank one lists.
	Tabs    []BrowserTab
	Active  int
	Servers []DevServer
}

type BrowserTab struct {
	Title   string
	Loading bool
}

type DevServer struct{ Label, URL string }

// Frame is one picture of the page, ready to draw.
type Frame struct {
	op   paint.ImageOp
	size image.Point
}

// NewFrame converts a decoded frame once, off the UI goroutine.
func NewFrame(img image.Image) Frame {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	}
	return Frame{op: paint.NewImageOp(rgba), size: rgba.Bounds().Size()}
}

// Unfocus takes the keyboard focus off every field at the next frame.
func (th *Theme) Unfocus() { th.in.blur = true }

// WriteClipboard puts text on the clipboard at the next frame.
func (th *Theme) WriteClipboard(text string) { th.in.clipOut = &text }

// browserW is the panel's width: about two fifths of the window until its
// edge is dragged, and never so wide the thread can't be read.
func (th *Theme) browserW(win int, gtx C) int {
	w := max(dp(gtx, 420), win*2/5)
	if th.in.webW > 0 {
		w = dp(gtx, th.in.webW)
	}
	return max(dp(gtx, 320), min(w, win-dp(gtx, sidebarW+420)))
}

func (th *Theme) browserPanel(gtx C, s *Scene) D {
	v := s.Web
	// The left edge drags, left for wider. The strip goes over the page,
	// so it's last, and the drag shows at the next frame.
	defer func() {
		if d := th.dragEdge(gtx, &th.in.webEdge, image.Pt(dp(gtx, 5), gtx.Constraints.Max.Y), true); d != 0 {
			th.in.webW = float32(gtx.Constraints.Max.X)/gtx.Metric.PxPerDp - d/gtx.Metric.PxPerDp
			gtx.Execute(op.InvalidateCmd{})
		}
	}()
	return Fill(gtx, ColPanelHi, func(gtx C) D {
		return Col(gtx,
			rigid(func(gtx C) D { return th.browserTabs(gtx, v) }),
			rigid(func(gtx C) D { return th.browserBar(gtx, v) }),
			flexed(func(gtx C) D { return th.browserPage(gtx, v) }),
		)
	})
}

// browserTabs is the panel's top row, as tall as the thread header: the
// thread's tabs, +, and close.
func (th *Theme) browserTabs(gtx C, v *BrowserView) D {
	return FixedH(gtx, topBarH, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		th.dragArea(gtx, size)
		hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
		gtx.Constraints.Min.X = size.X
		return pad(0, 10, 0, 10).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			var kids []layout.FlexChild
			for i, t := range v.Tabs {
				i, t := i, t
				on := i == v.Active
				kids = append(kids, layout.Flexed(1, func(gtx C) D {
					// Tabs share the row, up to 200dp each.
					gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, dp(gtx, 200))
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return Middle(th.Click("browser.tab:"+strconv.Itoa(i), func(gtx C) D {
						bg, col := color.NRGBA{}, ColDim
						if on {
							bg, col = ColFrameActive, ColText
						}
						return Box{Bg: bg, Radius: 6, Pad: pad(5, 4, 5, 8)}.Layout(gtx, func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return Row(gtx,
								rigid(Middle(func(gtx C) D {
									if t.Loading {
										return Spinner(ColAccentHi, 13, float64(gtx.Now.UnixMilli())/200)(gtx)
									}
									return LucideW("globe", col, 13)(gtx)
								})),
								rigid(Spacer(6, 0)),
								flexed(Middle(func(gtx C) D {
									return th.Text(gtx, Txt{Size: SizeSmall, Font: UI(font.Medium), Color: col, MaxLines: 1}, t.Title)
								})),
								rigid(Middle(th.Click("browser.tabclose:"+strconv.Itoa(i), func(gtx C) D {
									return pad(2, 2, 2, 2).Layout(gtx, LucideW("x", ColFaint, 12))
								}))),
							)
						})
					}))(gtx)
				}), rigid(Spacer(4, 0)))
			}
			kids = append(kids,
				rigid(Middle(th.Tip("browser.newtab", "New tab", false, 0, func(gtx C) D {
					return pad(5, 5, 5, 5).Layout(gtx, LucideW("plus", ColDim, 15))
				}))),
				flexed(func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
				rigid(Middle(th.Tip("browser.toggle", "Close browser", false, 1, func(gtx C) D {
					return pad(5, 5, 5, 5).Layout(gtx, LucideW("x", ColDim, 15))
				}))),
			)
			return Row(gtx, kids...)
		})
	})
}

// browserBar is T3's address row: back, forward, reload, the address.
func (th *Theme) browserBar(gtx C, v *BrowserView) D {
	return FixedH(gtx, 40, func(gtx C) D {
		size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Min.Y)
		hline(gtx, ColBorder, size.Y-max(1, dp(gtx, 1)), 0, size.X)
		gtx.Constraints.Min.X = size.X
		// The address follows the page while it isn't being typed in.
		if ed := th.Editor("browser.url"); !gtx.Focused(ed) && th.in.browserURL != v.URL {
			th.in.browserURL = v.URL
			ed.SetText(v.URL)
		}
		nav := func(id, icon string, on bool) layout.FlexChild {
			return rigid(Middle(func(gtx C) D {
				col := ColDim
				if !on {
					col = ColFaint
					id = ""
				}
				return th.Click(id, func(gtx C) D { return pad(5, 5, 5, 5).Layout(gtx, LucideW(icon, col, 15)) })(gtx)
			}))
		}
		return pad(0, 10, 0, 10).Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return Row(gtx,
				nav("browser.back", "arrow-left", v.CanBack),
				nav("browser.forward", "arrow-right", v.CanForward),
				nav("browser.reload", "rotate-cw", true),
				rigid(Spacer(6, 0)),
				flexed(Middle(th.Field("browser.url", IconGlobe, true, "Search or enter a URL", false))),
			)
		})
	})
}

// browserPage draws the page and passes the pointer and keyboard to it.
func (th *Theme) browserPage(gtx C, v *BrowserView) D {
	size := gtx.Constraints.Max
	// The page is as big as the panel, in CSS pixels at 1 per dp.
	css := image.Pt(int(float32(size.X)/gtx.Metric.PxPerDp), int(float32(size.Y)/gtx.Metric.PxPerDp))
	if v.Resize != nil && css.X >= 100 && css.Y >= 100 {
		v.Resize(css.X, css.Y)
	}
	tag := &th.in.browser
	view := v.Viewport
	if view.X == 0 {
		view = css
	}
	scale := float32(size.X) / float32(max(1, view.X)) // device px per CSS px
	th.browserInput(gtx, v, tag, scale)

	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	if v.URL == "" && v.Error == "" {
		// A new tab: T3's empty page, with the machine's dev servers.
		fill(gtx, ColPanelHi, image.Rectangle{Max: size}, 0)
		layout.Center.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, dp(gtx, 360))
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			kids := []layout.FlexChild{
				rigid(func(gtx C) D { return layout.Center.Layout(gtx, LucideW("globe", ColFaint, 28)) }),
				rigid(Spacer(0, 10)),
				rigid(func(gtx C) D {
					return layout.Center.Layout(gtx, func(gtx C) D {
						return th.Text(gtx, Txt{Color: ColDim, Align: text.Middle}, "Type a URL above, or a port like 3000\nfor a dev server on the thread's machine.")
					})
				}),
			}
			if len(v.Servers) > 0 {
				kids = append(kids, rigid(Spacer(0, 18)), rigid(th.SectionLabel("RUNNING ON THIS MACHINE", nil)))
				for i, sv := range v.Servers {
					i, sv := i, sv
					kids = append(kids, rigid(th.Click("browser.server:"+strconv.Itoa(i), func(gtx C) D {
						return FixedH(gtx, 34, func(gtx C) D {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return pad(0, 10, 0, 10).Layout(gtx, func(gtx C) D {
								gtx.Constraints.Min.X = gtx.Constraints.Max.X
								return Row(gtx,
									rigid(Middle(LucideW("server", ColDim, 14))),
									rigid(Spacer(10, 0)),
									flexed(Middle(func(gtx C) D { return th.Label(gtx, ColText, font.Medium, sv.Label) })),
									rigid(Middle(func(gtx C) D {
										return th.Text(gtx, Txt{Size: SizeMono, Font: Mono(), Color: ColFaint, MaxLines: 1}, sv.URL)
									})),
								)
							})
						})
					})))
				}
			}
			return Col(gtx, kids...)
		})
	} else if f := v.Frame; f != nil {
		// The frame covers the page's viewport, whatever its pixel size.
		k := float32(view.X) * scale / float32(f.size.X)
		st := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(k, k))).Push(gtx.Ops)
		f.op.Filter = paint.FilterLinear
		f.op.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		st.Pop()
	} else {
		fill(gtx, rgb(0xffffff), image.Rectangle{Max: size}, 0)
	}
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	area.Pop()

	note := v.Error
	switch {
	case note != "":
	case v.Agent:
		note = "The agent is using this tab."
	case v.Frame == nil && v.Loading:
		note = "Starting the browser…"
	}
	if note != "" {
		m := op.Record(gtx.Ops)
		d := Box{Bg: alpha(ColWindow, 0xe8), Border: ColBorderHi, Radius: 6, Pad: padXY(10, 6)}.Layout(gtx, func(gtx C) D {
			return th.Text(gtx, Txt{Size: SizeSmall, Font: UI(font.Medium), Color: ColText, MaxLines: 2}, note)
		})
		call := m.Stop()
		off := op.Offset(image.Pt((size.X-d.Size.X)/2, dp(gtx, 14))).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}
	if v.Loading && v.Frame != nil {
		// T3's thin progress line under the address bar.
		w := int(float32(size.X) * float32(0.3+0.6*float64(gtx.Now.UnixMilli()%1200)/1200))
		fill(gtx, ColAccent, image.Rect(0, 0, w, max(2, dp(gtx, 2))), 0)
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(33 * time.Millisecond)})
	}
	return D{Size: size}
}

// cdpMods is the modifiers as Chrome counts them: Alt 1, Ctrl 2, Meta 4,
// Shift 8.
func cdpMods(m key.Modifiers) int {
	n := 0
	if m.Contain(key.ModAlt) {
		n |= 1
	}
	if m.Contain(key.ModCtrl) {
		n |= 2
	}
	if m.Contain(key.ModCommand) {
		n |= 4
	}
	if m.Contain(key.ModShift) {
		n |= 8
	}
	return n
}

// domKeys is the DOM key, code and key code for the keys a page needs
// as key presses rather than typed text.
var domKeys = map[key.Name][3]any{
	key.NameReturn: {"Enter", "Enter", 13}, key.NameEnter: {"Enter", "NumpadEnter", 13},
	key.NameDeleteBackward: {"Backspace", "Backspace", 8}, key.NameDeleteForward: {"Delete", "Delete", 46},
	key.NameTab: {"Tab", "Tab", 9}, key.NameEscape: {"Escape", "Escape", 27},
	key.NameUpArrow: {"ArrowUp", "ArrowUp", 38}, key.NameDownArrow: {"ArrowDown", "ArrowDown", 40},
	key.NameLeftArrow: {"ArrowLeft", "ArrowLeft", 37}, key.NameRightArrow: {"ArrowRight", "ArrowRight", 39},
	key.NameHome: {"Home", "Home", 36}, key.NameEnd: {"End", "End", 35},
	key.NamePageUp: {"PageUp", "PageUp", 33}, key.NamePageDown: {"PageDown", "PageDown", 34},
}

func (th *Theme) browserInput(gtx C, v *BrowserView, tag event.Tag, scale float32) {
	send := func(m map[string]any) {
		if v.Send != nil {
			v.Send(m)
		}
	}
	at := func(p f32.Point) (float32, float32) { return p.X / scale, p.Y / scale }
	mods := key.ModCtrl | key.ModShift | key.ModAlt | key.ModCommand
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: tag},
			key.Filter{Focus: tag, Name: "", Optional: mods},
			key.Filter{Focus: tag, Name: key.NameTab, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: "V", Required: key.ModShortcut},
			transfer.TargetFilter{Target: tag, Type: "application/text"},
			pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Release | pointer.Move | pointer.Drag | pointer.Scroll,
				ScrollX: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}, ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}},
		)
		if !ok {
			return
		}
		switch e := ev.(type) {
		case pointer.Event:
			x, y := at(e.Position)
			btn := "none"
			switch {
			case e.Buttons.Contain(pointer.ButtonPrimary):
				btn = "left"
			case e.Buttons.Contain(pointer.ButtonSecondary):
				btn = "right"
			case e.Buttons.Contain(pointer.ButtonTertiary):
				btn = "middle"
			}
			bits := 0
			if e.Buttons.Contain(pointer.ButtonPrimary) {
				bits |= 1
			}
			if e.Buttons.Contain(pointer.ButtonSecondary) {
				bits |= 2
			}
			if e.Buttons.Contain(pointer.ButtonTertiary) {
				bits |= 4
			}
			m := map[string]any{"type": "mouse", "x": x, "y": y, "buttons": bits, "modifiers": cdpMods(e.Modifiers)}
			switch e.Kind {
			case pointer.Press:
				gtx.Execute(key.FocusCmd{Tag: tag})
				// Two presses close together are a double click.
				clicks := 1
				if gtx.Now.Sub(th.in.browserPress) < 400*time.Millisecond && e.Position.Sub(th.in.browserAt).X*e.Position.Sub(th.in.browserAt).X+e.Position.Sub(th.in.browserAt).Y*e.Position.Sub(th.in.browserAt).Y < 36 {
					clicks = th.in.browserClicks + 1
				}
				th.in.browserPress, th.in.browserAt, th.in.browserClicks, th.in.browserBtn = gtx.Now, e.Position, clicks, btn
				m["action"], m["button"], m["clickCount"] = "down", btn, clicks
			case pointer.Release:
				m["action"], m["button"], m["clickCount"] = "up", th.in.browserBtn, th.in.browserClicks
			case pointer.Scroll:
				send(map[string]any{"type": "wheel", "x": x, "y": y, "deltaX": e.Scroll.X / scale, "deltaY": e.Scroll.Y / scale, "modifiers": cdpMods(e.Modifiers)})
				continue
			default:
				// Moves go at most once a frame.
				if gtx.Now.Sub(th.in.browserMoved) < 16*time.Millisecond {
					continue
				}
				th.in.browserMoved = gtx.Now
				m["action"], m["button"] = "move", btn
			}
			send(m)
		case key.EditEvent:
			if e.Text != "" {
				send(map[string]any{"type": "text", "text": e.Text})
			}
			gtx.Execute(key.SnippetCmd{Tag: tag})
			gtx.Execute(key.SelectionCmd{Tag: tag})
		case transfer.DataEvent:
			b, _ := io.ReadAll(e.Open())
			if len(b) > 0 {
				send(map[string]any{"type": "text", "text": string(b)})
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			if e.Name == "V" && e.Modifiers.Contain(key.ModShortcut) {
				gtx.Execute(clipboard.ReadCmd{Tag: tag})
				continue
			}
			k, code, kc, ok := domKey(e)
			if !ok {
				continue
			}
			down := map[string]any{"type": "key", "action": "down", "key": k, "code": code, "keyCode": kc, "modifiers": cdpMods(e.Modifiers)}
			if k == "Enter" {
				down["text"] = "\r"
			}
			send(down)
			send(map[string]any{"type": "key", "action": "up", "key": k, "code": code, "keyCode": kc, "modifiers": cdpMods(e.Modifiers)})
		}
	}
}

// domKey is a key press as the page should see it: named keys, and
// letters and digits with Ctrl or ⌘ (copy, select all, undo). Typing
// arrives as text instead.
func domKey(e key.Event) (k, code string, keyCode int, ok bool) {
	if d, ok := domKeys[e.Name]; ok {
		return d[0].(string), d[1].(string), d[2].(int), true
	}
	short := e.Modifiers.Contain(key.ModCtrl) || e.Modifiers.Contain(key.ModCommand)
	if runtime.GOOS != "darwin" && e.Modifiers.Contain(key.ModAlt) {
		short = true
	}
	name := string(e.Name)
	if !short || len(name) != 1 {
		return "", "", 0, false
	}
	c := name[0]
	switch {
	case c >= 'A' && c <= 'Z':
		return strings.ToLower(name), "Key" + name, int(c), true
	case c >= '0' && c <= '9':
		return name, "Digit" + name, int(c), true
	}
	return "", "", 0, false
}
