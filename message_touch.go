package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ctrlIconR        = 12.0
	ctrlIconLabelGap = 6.0
	ctrlCellW        = 78.0
	ctrlMarginRight  = 16.0
	ctrlMarginTop    = 18.0
	ctrlKeyHintGapY  = 4.0

	ctrlLabelFontSize = 11.0

	// ctrlIconTapMargin widens the tap target for touch input only (not
	// mouse), since the 12px icon radius alone is too small to hit
	// reliably with a thumb.
	ctrlIconTapMargin = 16.0
)

func ctrlRowY() float64 {
	return ctrlMarginTop
}

func ctrlIconXAt(index int) float64 {
	startX := float64(gameWidth) - ctrlMarginRight - ctrlCellW*3 + ctrlIconR
	return startX + ctrlCellW*float64(index)
}

func msgControlPanelRect() (x, y, w, h float64) {
	x = ctrlIconXAt(0) - ctrlIconR - ctrlIconTapMargin
	y = ctrlRowY() - ctrlIconR - ctrlIconTapMargin
	bottom := ctrlRowY() + ctrlIconR + ctrlKeyHintGapY + 16
	w = float64(gameWidth) - x
	h = bottom - y
	return
}

func drawIconBackdrop(screen *ebiten.Image, cx, cy, r float64) {
	vector.FillCircle(screen, float32(cx), float32(cy), float32(r), color.NRGBA{0, 0, 0, 150}, true)
}

func fillTriPath(screen *ebiten.Image, col color.Color, pts [][2]float64) {
	var path vector.Path
	for i, p := range pts {
		if i == 0 {
			path.MoveTo(float32(p[0]), float32(p[1]))
		} else {
			path.LineTo(float32(p[0]), float32(p[1]))
		}
	}
	path.Close()
	var cs ebiten.ColorScale
	cs.ScaleWithColor(col)
	vector.FillPath(screen, &path, &vector.FillOptions{}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: cs})
}

func drawAutoIcon(screen *ebiten.Image, cx, cy, r float64, active bool) {
	drawIconBackdrop(screen, cx, cy, r)
	if active {
		barW := r * 0.28
		barH := r * 1.1
		gap := r * 0.3
		vector.FillRect(screen, float32(cx-gap/2-barW), float32(cy-barH/2), float32(barW), float32(barH), color.White, true)
		vector.FillRect(screen, float32(cx+gap/2), float32(cy-barH/2), float32(barW), float32(barH), color.White, true)
		return
	}
	s := r * 0.9
	fillTriPath(screen, color.White, [][2]float64{
		{cx - s*0.35, cy - s*0.5},
		{cx - s*0.35, cy + s*0.5},
		{cx + s*0.55, cy},
	})
}

func drawSkipIcon(screen *ebiten.Image, cx, cy, r float64) {
	drawIconBackdrop(screen, cx, cy, r)
	s := r * 0.6
	for _, offsetX := range []float64{-s * 0.35, s * 0.55} {
		fillTriPath(screen, color.White, [][2]float64{
			{cx + offsetX - s*0.45, cy - s*0.55},
			{cx + offsetX - s*0.45, cy + s*0.55},
			{cx + offsetX + s*0.45, cy},
		})
	}
}

func drawLogIcon(screen *ebiten.Image, cx, cy, r float64) {
	drawIconBackdrop(screen, cx, cy, r)
	lineW := r * 0.95
	for _, dy := range []float64{-r * 0.35, 0, r * 0.35} {
		vector.FillCircle(screen, float32(cx-lineW*0.42), float32(cy+dy), 1.6, color.White, true)
		vector.StrokeLine(screen,
			float32(cx-lineW*0.22), float32(cy+dy),
			float32(cx+lineW*0.42), float32(cy+dy),
			2, color.White, true)
	}
}

// ctrlIconHitRect returns the tap target covering both the icon circle and
// its label text, so touching the label counts the same as touching the
// icon. margin further pads the touch-only tap area.
func ctrlIconHitRect(game *Game, cx, cy float64, label string, margin float64) (x, y, w, h float64) {
	labelW := text.Advance(label, game.FontFace(ctrlLabelFontSize))
	x = cx - ctrlIconR - margin
	y = cy - ctrlIconR - margin
	w = ctrlIconR + ctrlIconLabelGap + labelW + ctrlIconR + margin*2
	h = ctrlIconR*2 + margin*2
	return
}

func isAutoIconJustPressed(game *Game) bool {
	cx, cy := ctrlIconXAt(0), ctrlRowY()
	touches, mouse := justPressedTouchAndMousePoints()
	tx, ty, tw, th := ctrlIconHitRect(game, cx, cy, "オート", ctrlIconTapMargin)
	for _, p := range touches {
		if p.inRect(tx, ty, tw, th) {
			return true
		}
	}
	mx, my, mw, mh := ctrlIconHitRect(game, cx, cy, "オート", 0)
	for _, p := range mouse {
		if p.inRect(mx, my, mw, mh) {
			return true
		}
	}
	return false
}

func isLogIconJustPressed(game *Game) bool {
	cx, cy := ctrlIconXAt(1), ctrlRowY()
	touches, mouse := justPressedTouchAndMousePoints()
	tx, ty, tw, th := ctrlIconHitRect(game, cx, cy, "ログ", ctrlIconTapMargin)
	for _, p := range touches {
		if p.inRect(tx, ty, tw, th) {
			return true
		}
	}
	mx, my, mw, mh := ctrlIconHitRect(game, cx, cy, "ログ", 0)
	for _, p := range mouse {
		if p.inRect(mx, my, mw, mh) {
			return true
		}
	}
	return false
}

func isSkipIconHeld(game *Game) bool {
	cx, cy := ctrlIconXAt(2), ctrlRowY()
	touches, mouse := activeTouchAndMousePoints()
	tx, ty, tw, th := ctrlIconHitRect(game, cx, cy, "スキップ", ctrlIconTapMargin)
	for _, p := range touches {
		if p.inRect(tx, ty, tw, th) {
			return true
		}
	}
	mx, my, mw, mh := ctrlIconHitRect(game, cx, cy, "スキップ", 0)
	for _, p := range mouse {
		if p.inRect(mx, my, mw, mh) {
			return true
		}
	}
	return false
}

func isMessageAdvancePressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		return true
	}

	px, py, pw, ph := msgControlPanelRect()
	for _, p := range justPressedTouchPoints() {
		if p.inRect(px, py, pw, ph) {
			continue
		}
		return true
	}
	return false
}

func drawLabel(screen *ebiten.Image, game *Game, cx, cy float64, label string) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx+ctrlIconR+ctrlIconLabelGap, cy)
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, label, game.FontFace(ctrlLabelFontSize), op)
}

func drawKeyHintPC(screen *ebiten.Image, game *Game, cx, cy float64, label, key string) {
	labelW := text.Advance(label, game.FontFace(ctrlLabelFontSize))
	leftX := cx - ctrlIconR
	rightX := cx + ctrlIconR + ctrlIconLabelGap + labelW
	midX := (leftX + rightX) / 2

	op := &text.DrawOptions{}
	op.GeoM.Translate(midX, cy+ctrlIconR+ctrlKeyHintGapY)
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, key, game.LatinFontFace(ctrlLabelFontSize), op)
}

func drawMessageControlPanel(screen *ebiten.Image, game *Game, autoOn bool, skipHoldElapsed, skipHoldMax float64) {
	y := ctrlRowY()

	autoX := ctrlIconXAt(0)
	drawAutoIcon(screen, autoX, y, ctrlIconR, autoOn)
	drawLabel(screen, game, autoX, y, "オート")

	logX := ctrlIconXAt(1)
	drawLogIcon(screen, logX, y, ctrlIconR)
	drawLabel(screen, game, logX, y, "ログ")

	skipX := ctrlIconXAt(2)
	drawSkipIcon(screen, skipX, y, ctrlIconR)
	drawLabel(screen, game, skipX, y, "スキップ")

	ringR := ctrlIconR + 2
	if skipHoldMax > 0 && skipHoldElapsed > 0 {
		drawRingOutline(screen, skipX, y, ringR, color.RGBA{255, 255, 255, 90})
		drawProgressArc(screen, skipX, y, ringR, skipHoldElapsed/skipHoldMax)
	}

	if !game.MobileMode {
		drawKeyHintPC(screen, game, autoX, y, "オート", "A")
		drawKeyHintPC(screen, game, logX, y, "ログ", "L")
		drawKeyHintPC(screen, game, skipX, y, "スキップ", "Ctrl")
	}
}
