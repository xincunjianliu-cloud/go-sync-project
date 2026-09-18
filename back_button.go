package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	backBtnR            = 12.0
	backBtnMarginLeft   = 22.0
	backBtnMarginBottom = 28.0
	backBtnKeyGapY      = 4.0
	backBtnBorderWidth  = 2.0
)

func backBtnCenter() (cx, cy float64) {
	return backBtnMarginLeft + backBtnR, float64(gameHeight) - backBtnMarginBottom
}

func backBtnRect() (x, y, w, h float64) {
	cx, cy := backBtnCenter()
	size := backBtnR * 2
	return cx - backBtnR, cy - backBtnR, size, size
}

func isTouchBackPressed() bool {
	x, y, w, h := backBtnRect()
	for _, p := range justPressedTouchPoints() {
		if p.inRect(x, y, w, h) {
			return true
		}
	}
	return false
}

func drawBackButton(screen *ebiten.Image, game *Game) {
	cx, cy := backBtnCenter()
	x, y, w, h := backBtnRect()
	ebitenutil.DrawRect(screen, x, y, w, h, color.NRGBA{0, 0, 0, 170})
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), backBtnBorderWidth, color.White, true)

	s := backBtnR * 0.55
	fillTriPath(screen, color.White, [][2]float64{
		{cx - s, cy},
		{cx + s*0.3, cy - s},
		{cx + s*0.3, cy + s},
	})

	if !game.MobileMode {
		op := &text.DrawOptions{}
		op.GeoM.Translate(cx, cy-backBtnR-backBtnKeyGapY)
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignEnd
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "Esc", game.FontFace(ctrlLabelFontSize), op)
	}
}
