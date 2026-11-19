// Command winicon draws tern's icon and writes it as a Windows resource, so
// the exe and its title bar show the mark. Run it from this directory:
//
//	go run . -out ../../cmd/tern
//
// It's its own module so the app doesn't depend on winres.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
	"golang.org/x/image/vector"
)

var (
	bg     = color.NRGBA{0x16, 0x1a, 0x20, 0xff}
	edge   = color.NRGBA{0x2a, 0x30, 0x3a, 0xff}
	accent = color.NRGBA{0x4c, 0x8d, 0xff, 0xff}
)

// icon is the mark at n×n pixels: the wing from ui.Logo on a rounded tile.
func icon(n int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	s := float32(n)
	fill := func(col color.NRGBA, path func(r *vector.Rasterizer)) {
		r := vector.NewRasterizer(n, n)
		path(r)
		r.Draw(img, img.Bounds(), image.NewUniform(col), image.Point{})
	}
	tile := func(inset, radius float32) func(r *vector.Rasterizer) {
		return func(r *vector.Rasterizer) {
			a, b, k := inset, s-inset, radius
			r.MoveTo(a+k, a)
			r.LineTo(b-k, a)
			r.QuadTo(b, a, b, a+k)
			r.LineTo(b, b-k)
			r.QuadTo(b, b, b-k, b)
			r.LineTo(a+k, b)
			r.QuadTo(a, b, a, b-k)
			r.LineTo(a, a+k)
			r.QuadTo(a, a, a+k, a)
			r.ClosePath()
		}
	}
	fill(edge, tile(0, s*0.22))
	fill(bg, tile(max(1, s/64), s*0.21))
	// The wing is drawn in a 16-unit box; centre it in about 72% of the tile.
	u := s * 0.72 / 16
	off := (s - 16*u) / 2
	pt := func(x, y float32) (float32, float32) { return off + x*u, off + y*u + u*0.4 }
	fill(accent, func(r *vector.Rasterizer) {
		r.MoveTo(pt(1, 10))
		x1, y1 := pt(6, 3)
		x2, y2 := pt(15, 2)
		r.QuadTo(x1, y1, x2, y2)
		x1, y1 = pt(10, 5)
		x2, y2 = pt(8.5, 9)
		r.QuadTo(x1, y1, x2, y2)
		x1, y1 = pt(11, 9.5)
		x2, y2 = pt(13, 14)
		r.QuadTo(x1, y1, x2, y2)
		x1, y1 = pt(8, 10.5)
		x2, y2 = pt(1, 10)
		r.QuadTo(x1, y1, x2, y2)
		r.ClosePath()
	})
	return img
}

func main() {
	out := flag.String("out", "cmd/tern", "directory for rsrc_windows_*.syso")
	pngOut := flag.String("png", "", "also write the 256px icon here")
	flag.Parse()

	var imgs []image.Image
	for _, n := range []int{16, 20, 24, 32, 40, 48, 64, 128, 256} {
		imgs = append(imgs, icon(n))
	}
	ico, err := winres.NewIconFromImages(imgs)
	if err != nil {
		log.Fatal(err)
	}
	rs := &winres.ResourceSet{}
	// Gio looks for the window icon at IDI_APPLICATION, then at 1; Explorer
	// takes the first group for the exe.
	if err := rs.SetIcon(winres.ID(1), ico); err != nil {
		log.Fatal(err)
	}
	if err := rs.SetIcon(winres.ID(32512), ico); err != nil {
		log.Fatal(err)
	}
	for _, arch := range []winres.Arch{winres.ArchAMD64, winres.ArchARM64} {
		f, err := os.Create(filepath.Join(*out, "rsrc_windows_"+string(arch)+".syso"))
		if err != nil {
			log.Fatal(err)
		}
		if err := rs.WriteObject(f, arch); err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
	if *pngOut != "" {
		f, err := os.Create(*pngOut)
		if err != nil {
			log.Fatal(err)
		}
		big := image.NewNRGBA(image.Rect(0, 0, 256, 256))
		draw.Draw(big, big.Bounds(), imgs[len(imgs)-1], image.Point{}, draw.Src)
		if err := png.Encode(f, big); err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
}
