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

	// battleIconTapMargin widens the tap target for touch input only (not
	// mouse), since the 12px icon radius alone is too small to hit
	// reliably with a thumb.
	battleIconTapMargin = 16.0

	// itemButtonR is half of the 50x50 item_button.png, which is drawn at
	// its native size.
	itemButtonR = 25.0
)

func rewindButtonCenter(game *Game) (cx, cy float64) {
	w := float64(game.GaugeImg.Bounds().Dx())
	h := float64(game.GaugeImg.Bounds().Dy())
	cx = gaugeTriX + w + rewindBtnGapX + battleIconR
	cy = gaugeTriY + h/2
	return
}

// itemButtonCenter sits diagonally down-left of the skill/flee pair, far
// enough out that the item diamond doesn't touch either one even when
// they swap to their larger _selected sprites (92px).
func itemButtonCenter() (cx, cy float64) {
	positions := commandIconPositions()
	midX := (positions[1][0] + positions[3][0]) / 2
	midY := (positions[1][1] + positions[3][1]) / 2
	const diagonal = 38.0
	cx = midX - diagonal
	cy = midY + diagonal
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

// drawItemIcon dims the icon when not selected, matching drawCommandMenu.
func drawItemIcon(screen *ebiten.Image, img *ebiten.Image, cx, cy float64, selected bool) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cx-w/2, cy-h/2)
	if !selected {
		op.ColorScale.Scale(0.6, 0.6, 0.6, 1.0)
	}
	screen.DrawImage(img, op)
}

func isRewindButtonJustPressed(game *Game) bool {
	cx, cy := rewindButtonCenter(game)
	touches, mouse := justPressedTouchAndMousePoints()
	for _, p := range touches {
		if p.inCircle(cx, cy, battleIconR+battleIconTapMargin) {
			return true
		}
	}
	for _, p := range mouse {
		if p.inCircle(cx, cy, battleIconR) {
			return true
		}
	}
	return false
}

func isItemButtonJustPressed() bool {
	cx, cy := itemButtonCenter()
	touches, mouse := justPressedTouchAndMousePoints()
	for _, p := range touches {
		if p.inCircle(cx, cy, itemButtonR+battleIconTapMargin) {
			return true
		}
	}
	for _, p := range mouse {
		if p.inCircle(cx, cy, itemButtonR) {
			return true
		}
	}
	return false
}

func (s *BattleScene) drawBattleShortcutButtons(screen *ebiten.Image) {
	rx, ry := rewindButtonCenter(s.game)
	drawRewindIcon(screen, rx, ry, battleIconR, s.rewindButtonArmed)

	ix, iy := itemButtonCenter()
	drawItemIcon(screen, s.game.ItemButtonImg, ix, iy, s.itemButtonArmed)

	if !s.game.MobileMode {
		face := s.game.FontFace(ctrlLabelFontSize)

		op := &text.DrawOptions{}
		op.GeoM.Translate(rx, ry+battleIconR+4)
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "F", face, op)

		// The item button sits near the bottom edge, so its label goes to
		// the left instead of below.
		op = &text.DrawOptions{}
		op.GeoM.Translate(ix-itemButtonR-4, iy)
		op.PrimaryAlign = text.AlignEnd
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "I", face, op)
	}
}

func isResultAdvancePressed() bool {
	return isConfirmKeyPressed() || len(justPressedTouchPoints()) > 0
}
