package ui

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// The editors' brand marks T3's Open in editor shows (Icons.tsx,
// JetBrainsIcons.tsx), rasterised by tools/editormarks/gen.py: their
// masks, blurred shadows and gradients are past Gio's paths. Each comes at
// 16, 24, 32 and 48px, once per theme where it takes the theme's colour.

//go:embed editormarks/*.png
var editorMarkFiles embed.FS

var editorMarkSizes = []int{16, 24, 32, 48}

var (
	editorMarkMu sync.Mutex
	editorMarks  = map[string]*paint.ImageOp{} // nil for a file that isn't there
)

// editorMarkName is the mark editor id has here: the file manager's is
// the Finder on a Mac and File Explorer on Windows, and elsewhere none.
func editorMarkName(id string) string {
	if id != "file-manager" {
		return id
	}
	switch runtime.GOOS {
	case "darwin":
		return "finder"
	case "windows":
		return "file-explorer"
	}
	return ""
}

// editorMarkOp is the image of mark name at size px in this theme, nil
// when there's none.
func editorMarkOp(name string, size int) *paint.ImageOp {
	editorMarkMu.Lock()
	defer editorMarkMu.Unlock()
	for _, theme := range []string{"-dark", ""} {
		if theme != "" && !dark {
			continue
		}
		file := fmt.Sprintf("editormarks/%s%s-%d.png", name, theme, size)
		img, ok := editorMarks[file]
		if !ok {
			if f, err := editorMarkFiles.Open(file); err == nil {
				if m, err := png.Decode(f); err == nil {
					io := paint.NewImageOp(m)
					img = &io
				}
				f.Close()
			}
			editorMarks[file] = img
		}
		if img != nil {
			return img
		}
	}
	return nil
}

// editorIcon is editor id's icon, sizeDp square (OpenInPicker.tsx): its
// brand mark, or for the file manager without one lucide's folder, in
// col; with no editor at all, the icon T3 opens things with.
func editorIcon(id string, col color.NRGBA, sizeDp float32) W {
	return func(gtx C) D {
		name := editorMarkName(id)
		switch {
		case id == "":
			return LucideW("square-arrow-out-up-right", col, sizeDp)(gtx)
		case editorMarkOp(name, editorMarkSizes[0]) == nil:
			return LucideW("folder-closed", col, sizeDp)(gtx)
		}
		px := dp(gtx, sizeDp)
		// The smallest raster that covers it, scaled down to it.
		size := editorMarkSizes[len(editorMarkSizes)-1]
		for _, s := range editorMarkSizes {
			if s >= px {
				size = s
				break
			}
		}
		k := float32(px) / float32(size)
		defer op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(k, k))).Push(gtx.Ops).Pop()
		defer clip.Rect{Max: image.Pt(size, size)}.Push(gtx.Ops).Pop()
		editorMarkOp(name, size).Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		return D{Size: image.Pt(px, px)}
	}
}
