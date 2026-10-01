// Command genicon generates the placeholder PWA / home-screen icons in
// ../../web from the game font. Replace the output PNGs with real artwork
// whenever it's ready (keep the same file names and sizes). Run:
//
//	go run ./tools/genicon
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	fontPath = "assets/fonts/k8x12.ttf"
	outDir   = "web"
	glyph    = "七"

	// glyphRatio is the glyph height relative to the icon size. It stays
	// inside the 80% safe zone so "maskable" icons aren't clipped.
	glyphRatio = 0.6
)

var (
	bgColor    = color.RGBA{0x1e, 0x1e, 0x1e, 0xff}
	glyphColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

var icons = []struct {
	name string
	size int
}{
	{"icon-192.png", 192},
	{"icon-512.png", 512},
	{"apple-touch-icon.png", 180},
}

func main() {
	data, err := os.ReadFile(fontPath)
	if err != nil {
		log.Fatal(err)
	}
	f, err := opentype.Parse(data)
	if err != nil {
		log.Fatal(err)
	}
	for _, ic := range icons {
		if err := writeIcon(f, ic.name, ic.size); err != nil {
			log.Fatal(err)
		}
	}
}

func writeIcon(f *opentype.Font, name string, size int) error {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(size) * glyphRatio,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	bounds, _ := font.BoundString(face, glyph)
	glyphW := bounds.Max.X - bounds.Min.X
	glyphH := bounds.Max.Y - bounds.Min.Y
	dot := fixed.Point26_6{
		X: fixed.I(size)/2 - glyphW/2 - bounds.Min.X,
		Y: fixed.I(size)/2 - glyphH/2 - bounds.Min.Y,
	}
	d := &font.Drawer{Dst: img, Src: &image.Uniform{glyphColor}, Face: face, Dot: dot}
	d.DrawString(glyph)

	out, err := os.Create(filepath.Join(outDir, name))
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}
