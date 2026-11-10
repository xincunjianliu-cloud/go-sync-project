package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	skipHintRadius = 14.0
	skipHintMargin = 24.0
)

func skipHintCenter(game *Game) (cx, cy float64) {
	labelFace := game.FontFace(12)
	labelWidth := text.Advance("スキップ", labelFace)
	rightX := float64(gameWidth) - skipHintMargin
	cx = rightX - labelWidth - 8 - skipHintRadius
	cy = float64(gameHeight) - skipHintMargin
	return
}

func drawSkipHint(screen *ebiten.Image, game *Game) {
	cx, cy := skipHintCenter(game)
	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(cx, cy)
	hintOp.PrimaryAlign = text.AlignCenter
	hintOp.SecondaryAlign = text.AlignCenter
	hintOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "Ctrl", game.FontFace(9), hintOp)

	labelOp := &text.DrawOptions{}
	labelOp.GeoM.Translate(cx+skipHintRadius+8, cy)
	labelOp.SecondaryAlign = text.AlignCenter
	labelOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "スキップ", game.FontFace(12), labelOp)
}

func drawRingOutline(screen *ebiten.Image, cx, cy, radius float64, col color.Color) {
	var ring vector.Path
	const segs = 40
	for i := 0; i <= segs; i++ {
		ang := float64(i) / segs * 2 * math.Pi
		x := cx + radius*math.Cos(ang)
		y := cy + radius*math.Sin(ang)
		if i == 0 {
			ring.MoveTo(float32(x), float32(y))
		} else {
			ring.LineTo(float32(x), float32(y))
		}
	}
	strokeOpts := &vector.StrokeOptions{Width: 2}
	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(col)
	vector.StrokePath(screen, &ring, strokeOpts, drawOpts)
}

func isSkipHintJustPressed(game *Game) bool {
	cx, cy := skipHintCenter(game)
	x := cx - skipHintRadius - 8
	y := cy - skipHintRadius
	w := float64(gameWidth) - x
	h := skipHintRadius * 2
	touches, mouse := justPressedTouchAndMousePoints()
	for _, p := range touches {
		if p.inRect(x, y, w, h) {
			return true
		}
	}
	for _, p := range mouse {
		if p.inRect(x, y, w, h) {
			return true
		}
	}
	return false
}
