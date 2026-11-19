package ui

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
)

// detailsW is the thread details card's width (--thread-details-panel-width).
const detailsW = cardW

// ShortcutShift is Shortcut with Shift: ⇧⌘B on a Mac, Ctrl+Shift+B
// elsewhere (keybindings.ts: ⌃⌥⇧⌘ order on a Mac, words elsewhere).
func ShortcutShift(k string) string {
	if s := Shortcut(k); s[0] != 'C' {
		return "⇧" + s
	}
	return "Ctrl+Shift+" + k
}

// ShortcutAlt is Shortcut with Alt: ⌥⌘B on a Mac, Ctrl+Alt+B elsewhere.
func ShortcutAlt(k string) string {
	if s := Shortcut(k); s[0] != 'C' {
		return "⌥" + s
	}
	return "Ctrl+Alt+" + k
}

// popUp is an open popover as popIn plays it; a mockup, one frame, shows
// it settled.
func (th *Theme) popUp(gtx C, s *Scene, id string, origin *f32.Point, w W) D {
	if !s.Live {
		return w(gtx)
	}
	return th.popIn(gtx, id, true, origin, w)
}

// popIn plays a popover's opening the way T3's do: it fades in over 150ms,
// growing from origin (in its own box) or, with none, rising a little.
// Each time id opens again it plays again.
func (th *Theme) popIn(gtx C, id string, open bool, origin *f32.Point, w W) D {
	if th.in.popAt == nil {
		th.in.popAt = map[string]time.Time{}
	}
	if !open {
		delete(th.in.popAt, id)
		return D{}
	}
	start, ok := th.in.popAt[id]
	if !ok {
		start = gtx.Now
		th.in.popAt[id] = start
	}
	t := easeOut(since(gtx.Now, start, 150*time.Millisecond))
	if t < 1 {
		gtx.Execute(op.InvalidateCmd{})
	}
	defer paint.PushOpacity(gtx.Ops, t).Pop()
	if origin != nil {
		k := 0.95 + 0.05*t
		defer op.Affine(f32.Affine2D{}.Scale(*origin, f32.Pt(k, k))).Push(gtx.Ops).Pop()
	} else {
		defer op.Offset(image.Pt(0, int((1-t)*float32(dp(gtx, 4))))).Push(gtx.Ops).Pop()
	}
	return w(gtx)
}

// details is T3's thread details popover: the machine and the run up top,
// the branch and folder below. Clicking the branch or folder copies it.
func (th *Theme) details(gtx C, s *Scene) D {
	r := s.Run
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = dp(gtx, detailsW), dp(gtx, detailsW)
	return popupFrame(gtx, func(gtx C) D {
		return pad(5, 5, 5, 5).Layout(gtx, func(gtx C) D {
			var kids []layout.FlexChild
			row := func(lead W, text string, mono bool, copyID string) {
				if text == "" {
					return
				}
				kids = append(kids, rigid(th.detailsRow(lead, text, mono, copyID)))
			}
			row(LucideW("server", ColDim, 16), r.Machine, false, "")
			status := r.Status
			if r.Elapsed != "" {
				status += " · " + r.Elapsed
			}
			row(func(gtx C) D { return layout.Center.Layout(gtx, th.statusGlyph(s)) }, status, false, "")
			row(ProviderLogo(r.Provider, 16), r.Model, false, "")
			lock := "lock"
			if r.Mode == "Full access" {
				lock = "lock-open"
			}
			row(LucideW(lock, ColDim, 16), r.Mode, false, "")
			if r.Branch != "" || r.Worktree != "" {
				kids = append(kids, rigid(func(gtx C) D {
					w := gtx.Constraints.Max.X
					hline(gtx, ColBorder, dp(gtx, 4), 0, w)
					return D{Size: image.Pt(w, dp(gtx, 9))}
				}))
			}
			row(LucideW("git-branch", ColDim, 16), r.Branch, true, "details.branch")
			row(LucideW("folder", ColDim, 16), r.Worktree, true, "details.folder")
			return Col(gtx, kids...)
		})
	})
}

// statusGlyph is the run's spinner or dot.
func (th *Theme) statusGlyph(s *Scene) W {
	switch s.State {
	case ThreadApproval, ThreadQuestion:
		return Dot(ColAmber, true)
	case ThreadError:
		return Dot(ColRed, false)
	case ThreadIdle:
		return Dot(ColGreen, false)
	}
	return Spinner(ColAccentHi, 11, s.Phase)
}

// detailsRow is one line of the popover, like a T3 menu item. With a
// copyID, a click copies text and the row says so for a moment.
func (th *Theme) detailsRow(lead W, text string, mono bool, copyID string) W {
	return func(gtx C) D {
		body := func(gtx C) D {
			return FixedH(gtx, 32, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return pad(0, 8, 0, 8).Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					f := Txt{MaxLines: 1, Color: ColText}
					if mono {
						f.Font, f.Size = Mono(), SizeMono
					}
					shown := text
					if th.copiedRecently(gtx, copyID) {
						shown, f.Color, f.Font, f.Size = "Copied", ColGreen, UI(font.Medium), SizeUI
					}
					return Row(gtx,
						rigid(func(gtx C) D { return FixedW(gtx, 16, lead) }),
						rigid(Spacer(10, 0)),
						flexed(func(gtx C) D { return th.Text(gtx, f, shown) }),
					)
				})
			})
		}
		if copyID == "" {
			return body(gtx)
		}
		if th.in.copies == nil {
			th.in.copies = map[string]string{}
		}
		if th.in.copied == nil {
			th.in.copied = map[string]time.Time{}
		}
		th.in.copies[copyID] = text
		return th.Click(copyID, body)(gtx)
	}
}

// copiedRecently reports whether id's text was copied in the last 1.5s.
func (th *Theme) copiedRecently(gtx C, id string) bool {
	at, ok := th.in.copied[id]
	if id == "" || !ok {
		return false
	}
	if left := 1500*time.Millisecond - gtx.Now.Sub(at); left > 0 {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(left)})
		return true
	}
	delete(th.in.copied, id)
	return false
}
