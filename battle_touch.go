package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	battleIconR = 12.0

	rewindBtnGapX = 20.0
)

func rewindButtonCenter(game *Game) (cx, cy float64) {
	w := float64(game.GaugeImg.Bounds().Dx())
	h := float64(game.GaugeImg.Bounds().Dy())
	cx = gaugeTriX + w + rewindBtnGapX + battleIconR
	cy = gaugeTriY + h/2
	return
}

func itemButtonCenter() (cx, cy float64) {
	positions := commandIconPositions()
	cx = positions[1][0]
	cy = positions[3][1]
	return
}

func drawIconSelectedRing(screen *ebiten.Image, cx, cy, r float64) {
	vector.StrokeCircle(screen, float32(cx), float32(cy), float32(r+4), 2, color.White, true)
}

func drawRewindIcon(screen *ebiten.Image, cx, cy, r float64, selected bool) {
	drawIconBackdrop(screen, cx, cy, r)
	if selected {
		drawIconSelectedRing(screen, cx, cy, r)
	}

	radius := r * 0.55
	startAngle := -20.0 * math.Pi / 180
	endAngle := 250.0 * math.Pi / 180

	var path vector.Path
	path.Arc(float32(cx), float32(cy), float32(radius), float32(startAngle), float32(endAngle), vector.Clockwise)

	strokeOpts := &vector.StrokeOptions{Width: 2.5}
	drawOpts := &vector.DrawPathOptions{AntiAlias: true}
	drawOpts.ColorScale.ScaleWithColor(color.White)
	vector.StrokePath(screen, &path, strokeOpts, drawOpts)

	sx := cx + radius*math.Cos(startAngle)
	sy := cy + radius*math.Sin(startAngle)
	tipAngle := startAngle - math.Pi/2.2
	tipLen := r * 0.32
	tx := sx + tipLen*math.Cos(tipAngle)
	ty := sy + tipLen*math.Sin(tipAngle)

	perp := startAngle + math.Pi/2
	baseHalf := r * 0.16
	b1x := sx + baseHalf*math.Cos(perp)
	b1y := sy + baseHalf*math.Sin(perp)
	b2x := sx - baseHalf*math.Cos(perp)
	b2y := sy - baseHalf*math.Sin(perp)

	fillTriPath(screen, color.White, [][2]float64{{tx, ty}, {b1x, b1y}, {b2x, b2y}})
}

func drawItemIcon(screen *ebiten.Image, cx, cy, r float64, selected bool) {
	drawIconBackdrop(screen, cx, cy, r)
	if selected {
		drawIconSelectedRing(screen, cx, cy, r)
	}
	w := r * 1.1
	h := r * 0.9
	x := cx - w/2
	y := cy - h/2 + r*0.15
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, color.White, true)

	handleW := w * 0.5
	vector.StrokeRect(screen, float32(cx-handleW/2), float32(y-r*0.25), float32(handleW), float32(r*0.25), 2, color.White, true)
}

func isRewindButtonJustPressed(game *Game) bool {
	cx, cy := rewindButtonCenter(game)
	for _, p := range justPressedTouchPoints() {
		if p.inCircle(cx, cy, battleIconR) {
			return true
		}
	}
	return false
}

func isItemButtonJustPressed() bool {
	cx, cy := itemButtonCenter()
	for _, p := range justPressedTouchPoints() {
		if p.inCircle(cx, cy, battleIconR) {
			return true
		}
	}
	return false
}

func (s *BattleScene) drawBattleShortcutButtons(screen *ebiten.Image) {
	rx, ry := rewindButtonCenter(s.game)
	drawRewindIcon(screen, rx, ry, battleIconR, s.rewindButtonArmed)

	ix, iy := itemButtonCenter()
	drawItemIcon(screen, ix, iy, battleIconR, s.itemButtonArmed)

	if !s.game.MobileMode {
		drawKeyLabel := func(cx, cy float64, key string) {
			op := &text.DrawOptions{}
			op.GeoM.Translate(cx, cy+battleIconR+4)
			op.PrimaryAlign = text.AlignCenter
			op.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, key, s.game.FontFace(ctrlKeyFontSize), op)
		}
		drawKeyLabel(rx, ry, "F")
		drawKeyLabel(ix, iy, "I")
	}
}

func isResultAdvancePressed() bool {
	return isConfirmKeyPressed() || len(justPressedTouchPoints()) > 0
}
