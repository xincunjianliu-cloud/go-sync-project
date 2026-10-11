package main

import (
	"testing"

	"golang.org/x/image/math/fixed"
)

func TestHorizontalTriangleGlyphDirections(t *testing.T) {
	const w, h = 10.0, 10.0

	face := &triangleGlyphFace{size: 18}
	leftBounds, leftAdvance, leftOK := face.GlyphBounds(leftTriangleRune)
	rightBounds, rightAdvance, rightOK := face.GlyphBounds(rightTriangleRune)
	if !leftOK || !rightOK {
		t.Fatal("horizontal triangle glyphs should be available")
	}
	if leftBounds != rightBounds || leftAdvance != rightAdvance {
		t.Fatal("left and right triangle glyphs should have identical dimensions")
	}

	_, leftMask, _, _, leftOK := face.Glyph(fixed.Point26_6{}, leftTriangleRune)
	_, rightMask, _, _, rightOK := face.Glyph(fixed.Point26_6{}, rightTriangleRune)
	if !leftOK || !rightOK {
		t.Fatal("horizontal triangle glyphs should rasterize")
	}
	inkMinX, inkMaxX := rightMask.Bounds().Dx(), 0
	for y := range rightMask.Bounds().Dy() {
		for x := range rightMask.Bounds().Dx() {
			alpha, _, _, _ := rightMask.At(x, y).RGBA()
			if alpha != 0 {
				if x < inkMinX {
					inkMinX = x
				}
				if x+1 > inkMaxX {
					inkMaxX = x + 1
				}
			}
		}
	}
	for y := range rightMask.Bounds().Dy() {
		for x := inkMinX; x < inkMaxX; x++ {
			leftAlpha, _, _, _ := leftMask.At(leftMask.Bounds().Min.X+x, leftMask.Bounds().Min.Y+y).RGBA()
			rightX := inkMinX + inkMaxX - 1 - x
			rightAlpha, _, _, _ := rightMask.At(rightMask.Bounds().Min.X+rightX, rightMask.Bounds().Min.Y+y).RGBA()
			if leftAlpha != rightAlpha {
				t.Fatalf("left and right glyph masks differ at mirrored pixel (%d, %d)", x, y)
			}
		}
	}

	if !triangleLit(rightTriangleRune, 9, 5, w, h) {
		t.Fatal("right triangle should include its tip on the right")
	}
	if triangleLit(rightTriangleRune, 9, 1, w, h) {
		t.Fatal("right triangle should narrow toward the top")
	}
	if !triangleLit(leftTriangleRune, 1, 5, w, h) {
		t.Fatal("left triangle should include its tip on the left")
	}
	if triangleLit(leftTriangleRune, 1, 1, w, h) {
		t.Fatal("left triangle should narrow toward the top")
	}

	if !triangleLit(downTriangleRune, 5, 1, w, h) {
		t.Fatal("down triangle should include its tip at the top")
	}
	if triangleLit(downTriangleRune, 1, 9, w, h) {
		t.Fatal("down triangle should narrow toward the bottom")
	}
}
