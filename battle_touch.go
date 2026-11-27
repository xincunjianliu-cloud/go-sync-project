package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	battleIconR = 12.0

	rewindBtnGapX = -25.0

	// battleIconTapMargin widens the tap target for touch input only (not
	// mouse), since the 12px icon radius alone is too small to hit
	// reliably with a thumb.
	battleIconTapMargin = 16.0

	// itemButtonR is half of the 50x50 item_button.png, which is drawn at
	// its native size.
	itemButtonR = 25.0

	// commandDimScale is the brightness of every unselected battle command
	// button (attack/skill/wait/flee icons, item, rewind), so they all dim
	// by the same amount.
	commandDimScale = 0.6
)

func rewindButtonCenter(game *Game) (cx, cy float64) {
	w := float64(game.GaugeImg.Bounds().Dx())
	h := float64(game.GaugeImg.Bounds().Dy())
	iconW, iconH := battleIconR*2, battleIconR*2
	if game.RewindIconImg != nil {
		iconW = float64(game.RewindIconImg.Bounds().Dx())
		iconH = float64(game.RewindIconImg.Bounds().Dy())
	}
	cx = gaugeTriX + w + rewindBtnGapX + iconW/2
	cy = gaugeTriY + h - iconH/2
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

func drawRewindIcon(screen *ebiten.Image, img *ebiten.Image, cx, cy float64, selected bool) {
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(cx-w/2, cy-h/2)
	if !selected {
		op.ColorScale.Scale(commandDimScale, commandDimScale, commandDimScale, 1)
	}
	screen.DrawImage(img, op)
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
		op.ColorScale.Scale(commandDimScale, commandDimScale, commandDimScale, 1.0)
	}
	screen.DrawImage(img, op)
}

func isRewindButtonJustPressed(game *Game) bool {
	cx, cy := rewindButtonCenter(game)
	radius := battleIconR
	if game.RewindIconImg != nil {
		radius = float64(game.RewindIconImg.Bounds().Dx()) / 2
	}
	touches, mouse := justPressedTouchAndMousePoints()
	for _, p := range touches {
		if p.inCircle(cx, cy, radius+battleIconTapMargin) {
			return true
		}
	}
	for _, p := range mouse {
		if p.inCircle(cx, cy, radius) {
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

// drawBattleShortcutButtons draws the rewind and item buttons, lit only when
// selected and dimmed otherwise, the same way drawCommandMenu treats the
// four command icons. The rewind button only exists once the gauge system is
// unlocked, and the item button only exists while a battle-usable item is held.
func (s *BattleScene) drawBattleShortcutButtons(screen *ebiten.Image, rewindSelected, itemSelected bool) {
	rx, ry := rewindButtonCenter(s.game)
	if s.gaugeUnlocked {
		drawRewindIcon(screen, s.game.RewindIconImg, rx, ry, rewindSelected)
	}

	hasBattleItem := s.hasAnyBattleUsableItem()
	if hasBattleItem {
		ix, iy := itemButtonCenter()
		drawItemIcon(screen, s.game.ItemButtonImg, ix, iy, itemSelected)

		if !s.game.MobileMode {
			face := s.game.FontFace(ctrlLabelFontSize)

			// The item button sits near the bottom edge, so its label goes to
			// the left instead of below.
			op := &text.DrawOptions{}
			op.GeoM.Translate(ix-itemButtonR-4, iy)
			op.PrimaryAlign = text.AlignEnd
			op.SecondaryAlign = text.AlignCenter
			op.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "I", face, op)
		}
	}

	if !s.game.MobileMode && s.gaugeUnlocked {
		iconW := battleIconR * 2
		if s.game.RewindIconImg != nil {
			iconW = float64(s.game.RewindIconImg.Bounds().Dx())
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(rx-iconW/2-4, ry)
		op.PrimaryAlign = text.AlignEnd
		op.SecondaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "F", s.game.FontFace(ctrlLabelFontSize), op)
	}
}

func isResultAdvancePressed() bool {
	return isConfirmKeyPressed() || len(justPressedTouchPoints()) > 0
}
