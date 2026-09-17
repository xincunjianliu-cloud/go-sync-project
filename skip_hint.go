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

func drawSkipHint(screen *ebiten.Image, game *Game, holdElapsed, holdMax float64) {
	cx, cy := skipHintCenter(game)

	drawRingOutline(screen, cx, cy, skipHintRadius, color.RGBA{255, 255, 255, 90})

	progress := 0.0
	if holdMax > 0 {
		progress = holdElapsed / holdMax
	}
	if progress > 0 {
		drawProgressArc(screen, cx, cy, skipHintRadius, progress)
	}

	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(cx, cy)
	hintOp.PrimaryAlign = text.AlignCenter
	hintOp.SecondaryAlign = text.AlignCenter
	hintOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "Ctrl", game.LatinFontFace(9), hintOp)

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

func drawProgressArc(screen *ebiten.Image, cx, cy, radius, progress float64) {
	if progress > 1 {
		progress = 1
	}
	var arc vector.Path
	const segs = 40
	steps := int(float64(segs) * progress)
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		ang := -math.Pi/2 + progress*2*math.Pi*float64(i)/float64(steps)
		x := cx + radius*math.Cos(ang)
		y := cy + radius*math.Sin(ang)
		if i == 0 {
			arc.MoveTo(float32(x), float32(y))
		} else {
			arc.LineTo(float32(x), float32(y))
		}
	}
	strokeOpts := &vector.StrokeOptions{Width: 3}
	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(uiColorSelect)
	vector.StrokePath(screen, &arc, strokeOpts, drawOpts)
}
