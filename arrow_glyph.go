package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// assets/fonts/k8x12.ttf has no glyph for "▶"/"◀" (U+25B6/U+25C0) — the
// characters used throughout the UI as the selection cursor — so text.Draw
// falls back to drawing nothing (an empty tofu box). See the
// project_k8x12_font_glyph_coverage memory for how this was confirmed by
// inspecting the font's cmap directly.
//
// triangleGlyphFace synthesizes those two glyphs procedurally instead of
// requiring a second font file: Glyph() below draws a plain hard-edged
// "play button" triangle from scratch (three straight edges converging to a
// point) — it does NOT reuse k8x12.ttf's outline data in any way, only its
// own hand-tuned proportions (triWidthRatio/triHeightRatio). Its vertical
// center is placed to match the actual ink-center of ordinary text in this
// font (measured from "あ"/"項"'s glyph bounds at ppem 14: center ≈
// -0.375*size from the baseline).
//
// k8x12.ttf DOES have its own "▼" (U+25BC) glyph, but its size/weight didn't
// match the synthesized "▶"/"◀" cursor, so "▼" is synthesized here too
// (as downTriangleRune) using the same construction, rotated 90°, so all
// three read as one consistent set. FontFace() in game.go lists this face
// BEFORE the real font in its MultiFace so these three runes always resolve
// here instead of falling through to k8x12.ttf's own "▼" outline.
type triangleGlyphFace struct {
	size float64
}

var _ font.Face = (*triangleGlyphFace)(nil)

const (
	rightTriangleRune = '▶'
	leftTriangleRune  = '◀'
	downTriangleRune  = '▼'
)

const (
	// Hand-tuned, slightly-narrower-than-tall proportions for a normal
	// "play button" cursor look (an early version derived these from
	// rotating ▼'s bounding box, which looked stretched/wrong once
	// rendered — not used anymore).
	triWidthRatio   = 0.45
	triHeightRatio  = 0.5837
	triCenterRatio  = -0.375 // ink-center target, matched to real text
	triTopRatio     = triCenterRatio - triHeightRatio/2
	triAdvanceRatio = 0.53
	triAscentRatio  = 0.8337
	triDescentRatio = 0.1663

	// downAdvanceRatio gives "▼" some side padding beyond its own spread
	// (downSpreadRatio below), matching the amount triAdvanceRatio adds
	// beyond triWidthRatio for "▶"/"◀".
	downSpreadRatio  = triHeightRatio
	downDepthRatio   = triWidthRatio
	downAdvanceRatio = downSpreadRatio + (triAdvanceRatio - triWidthRatio)
)

func (f *triangleGlyphFace) isTriangle(r rune) bool {
	return r == rightTriangleRune || r == leftTriangleRune || r == downTriangleRune
}

func (f *triangleGlyphFace) Close() error { return nil }

func (f *triangleGlyphFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:  floatToFixed(f.size),
		Ascent:  floatToFixed(f.size * triAscentRatio),
		Descent: floatToFixed(f.size * triDescentRatio),
	}
}

func (f *triangleGlyphFace) Kern(r0, r1 rune) fixed.Int26_6 { return 0 }

// glyphBox returns the glyph's ink box (w, h) and its offset from the glyph
// origin (top, left). "▶"/"◀" are flush with the origin on the X axis (left
// == 0), matching how they were always laid out; "▼" is instead centered
// within its own advance width, since a downward cursor reads naturally
// centered rather than flush-left.
func (f *triangleGlyphFace) glyphBox(r rune) (w, h, top, left float64) {
	if r == downTriangleRune {
		w = f.size * downSpreadRatio
		h = f.size * downDepthRatio
		adv := f.size * downAdvanceRatio
		left = (adv - w) / 2
		top = f.size*triCenterRatio - h/2
		return w, h, top, left
	}
	w = f.size * triWidthRatio
	h = f.size * triHeightRatio
	top = f.size * triTopRatio
	return w, h, top, 0
}

func (f *triangleGlyphFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	switch r {
	case rightTriangleRune, leftTriangleRune:
		return floatToFixed(f.size * triAdvanceRatio), true
	case downTriangleRune:
		return floatToFixed(f.size * downAdvanceRatio), true
	}
	return 0, false
}

func (f *triangleGlyphFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	if !f.isTriangle(r) {
		return fixed.Rectangle26_6{}, 0, false
	}
	w, h, top, left := f.glyphBox(r)
	adv, _ := f.GlyphAdvance(r)
	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: floatToFixed(left), Y: floatToFixed(top)},
		Max: fixed.Point26_6{X: floatToFixed(left + w), Y: floatToFixed(top + h)},
	}, adv, true
}

// triSuperSample is the per-axis supersampling factor used to antialias the
// triangle's diagonal edges — at the small sizes this glyph is drawn at
// (~14-24px), a hard-edged diagonal looked visibly jagged/staircased.
const triSuperSample = 4

// triangleLit reports whether local point (fx, fy) inside a w×h box falls
// inside the triangle for rune r: a flat base on the side opposite the
// direction r points, tapering to a single point in that direction.
func triangleLit(r rune, fx, fy, w, h float64) bool {
	switch r {
	case rightTriangleRune:
		d := math.Abs(fy-h/2) / (h / 2)
		if d > 1 {
			d = 1
		}
		return fx <= w*(1-d)
	case leftTriangleRune:
		d := math.Abs(fy-h/2) / (h / 2)
		if d > 1 {
			d = 1
		}
		return fx >= w*d
	case downTriangleRune:
		d := math.Abs(fx-w/2) / (w / 2)
		if d > 1 {
			d = 1
		}
		return fy <= h*(1-d)
	}
	return false
}

// Glyph rasterizes an antialiased "play button" triangle: pointing right for
// rightTriangleRune, mirrored for leftTriangleRune, and pointing down for
// downTriangleRune.
func (f *triangleGlyphFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	if !f.isTriangle(r) {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	w, h, top, left := f.glyphBox(r)
	adv, _ := f.GlyphAdvance(r)
	iw, ih := int(math.Ceil(w))+1, int(math.Ceil(h))+1
	mask := image.NewAlpha(image.Rect(0, 0, iw, ih))
	const ss = triSuperSample
	for y := 0; y < ih; y++ {
		for x := 0; x < iw; x++ {
			var hits int
			for sy := 0; sy < ss; sy++ {
				fy := float64(y) + (float64(sy)+0.5)/ss
				for sx := 0; sx < ss; sx++ {
					fx := float64(x) + (float64(sx)+0.5)/ss
					if triangleLit(r, fx, fy, w, h) {
						hits++
					}
				}
			}
			if hits > 0 {
				mask.SetAlpha(x, y, color.Alpha{A: uint8(hits * 255 / (ss * ss))})
			}
		}
	}
	top32 := int(math.Round(top))
	left32 := int(math.Round(left))
	dr := image.Rect(
		dot.X.Round()+left32, dot.Y.Round()+top32,
		dot.X.Round()+left32+iw, dot.Y.Round()+top32+ih,
	)
	return dr, mask, image.Point{}, adv, true
}

func floatToFixed(v float64) fixed.Int26_6 {
	return fixed.Int26_6(math.Round(v * 64))
}

var multiFaceCache = map[float64]*text.MultiFace{}

// triangleFallbackFace builds the "▶"/"◀"/"▼" face for the given size,
// combined with the real font in FontFace().
func triangleFallbackFace(size float64) *text.GoXFace {
	return text.NewGoXFace(&triangleGlyphFace{size: size})
}
