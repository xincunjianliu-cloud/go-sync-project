package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	touchPadCenterX = 110.0
	touchPadCenterY = float64(gameHeight) - 110.0
	stickBaseR      = 62.0
	stickKnobR      = 26.0
	stickCaptureR   = stickBaseR * 1.8
	stickDeadzone   = 10.0
	stickArrowSize  = 16.0

	touchActionX = float64(gameWidth) - 90.0
	touchActionY = float64(gameHeight) - 90.0
	touchActionR = 44.0

	touchMenuCenterX     = float64(gameWidth) - 60.0
	touchMenuCenterY     = 40.0
	touchMenuSize        = 32.0
	touchMenuKeyGapY     = 4.0
	touchMenuBorderWidth = 2.0

	touchActionTapMargin = 16.0
	touchMenuTapMargin   = 14.0
)

type touchPoint struct{ x, y float64 }

const touchStickNoTouch ebiten.TouchID = -1

var fieldMobileControlsEnabled bool

func activeTouchPoints() []touchPoint {
	var pts []touchPoint
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		pts = append(pts, touchPoint{float64(x), float64(y)})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pts = append(pts, touchPoint{float64(x), float64(y)})
	}
	return pts
}

func justPressedTouchPoints() []touchPoint {
	var pts []touchPoint
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		pts = append(pts, touchPoint{float64(x), float64(y)})
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pts = append(pts, touchPoint{float64(x), float64(y)})
	}
	return pts
}

func justPressedTouchAndMousePoints() (touches []touchPoint, mouse []touchPoint) {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		touches = append(touches, touchPoint{float64(x), float64(y)})
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		mouse = append(mouse, touchPoint{float64(x), float64(y)})
	}
	return touches, mouse
}

func (p touchPoint) inRect(x, y, w, h float64) bool {
	return p.x >= x && p.x < x+w && p.y >= y && p.y < y+h
}

func (p touchPoint) inCircle(cx, cy, r float64) bool {
	dx, dy := p.x-cx, p.y-cy
	return dx*dx+dy*dy <= r*r
}

func touchIDStillHeld(id ebiten.TouchID) bool {
	for _, tid := range ebiten.AppendTouchIDs(nil) {
		if tid == id {
			return true
		}
	}
	return false
}

func pointInStickCapture(x, y float64) bool {
	dx := x - touchPadCenterX
	dy := y - touchPadCenterY
	return dx*dx+dy*dy <= stickCaptureR*stickCaptureR
}

func (s *FieldScene) setStickFromPoint(x, y float64) {
	dx := x - touchPadCenterX
	dy := y - touchPadCenterY
	if dist := math.Hypot(dx, dy); dist > stickBaseR {
		scale := stickBaseR / dist
		dx *= scale
		dy *= scale
	}
	s.touchStickActive = true
	s.touchStickDX = dx
	s.touchStickDY = dy
}

func (s *FieldScene) releaseStick() {
	s.touchStickTouchID = touchStickNoTouch
	s.touchStickUseMouse = false
	s.touchStickActive = false
	s.touchStickDX = 0
	s.touchStickDY = 0
}

func (s *FieldScene) updateTouchStick() {
	if !fieldMobileControlsEnabled {
		s.releaseStick()
		return
	}

	if s.touchStickTouchID != touchStickNoTouch {
		if touchIDStillHeld(s.touchStickTouchID) {
			x, y := ebiten.TouchPosition(s.touchStickTouchID)
			s.setStickFromPoint(float64(x), float64(y))
			return
		}
		s.releaseStick()
	} else if s.touchStickUseMouse {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			s.setStickFromPoint(float64(x), float64(y))
			return
		}
		s.releaseStick()
	}

	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if pointInStickCapture(float64(x), float64(y)) {
			s.touchStickTouchID = id
			s.setStickFromPoint(float64(x), float64(y))
			return
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if pointInStickCapture(float64(x), float64(y)) {
			s.touchStickUseMouse = true
			s.setStickFromPoint(float64(x), float64(y))
		}
	}
}

func (s *FieldScene) touchMoveDir() (dx, dy int) {
	if !s.touchStickActive {
		return 0, 0
	}
	switch {
	case s.touchStickDX > stickDeadzone:
		dx = 1
	case s.touchStickDX < -stickDeadzone:
		dx = -1
	}
	switch {
	case s.touchStickDY > stickDeadzone:
		dy = 1
	case s.touchStickDY < -stickDeadzone:
		dy = -1
	}
	return dx, dy
}

const dashThreshold = stickBaseR - 6

func (s *FieldScene) touchStickDash() bool {
	if !s.touchStickActive {
		return false
	}
	return math.Hypot(s.touchStickDX, s.touchStickDY) >= dashThreshold
}

func fieldTouchActionPressed() bool {
	if !fieldMobileControlsEnabled {
		return false
	}
	touches, mouse := justPressedTouchAndMousePoints()
	for _, p := range touches {
		if p.inCircle(touchActionX, touchActionY, touchActionR+touchActionTapMargin) {
			return true
		}
	}
	for _, p := range mouse {
		if p.inCircle(touchActionX, touchActionY, touchActionR) {
			return true
		}
	}
	return false
}

func fieldTouchMenuPressed() bool {
	x, y, w, h := touchMenuCenterX-touchMenuSize/2, touchMenuCenterY-touchMenuSize/2, touchMenuSize, touchMenuSize
	touches, mouse := justPressedTouchAndMousePoints()
	m := touchMenuTapMargin
	for _, p := range touches {
		if p.inRect(x-m, y-m, w+2*m, h+2*m) {
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

func drawStickArrow(screen *ebiten.Image, cx, cy, ux, uy float64, clr color.Color) {
	tipDist := stickBaseR - 10
	baseDist := tipDist - stickArrowSize
	tipX, tipY := cx+ux*tipDist, cy+uy*tipDist
	baseX, baseY := cx+ux*baseDist, cy+uy*baseDist
	px, py := -uy, ux
	halfW := stickArrowSize * 0.55

	var path vector.Path
	path.MoveTo(float32(tipX), float32(tipY))
	path.LineTo(float32(baseX+px*halfW), float32(baseY+py*halfW))
	path.LineTo(float32(baseX-px*halfW), float32(baseY-py*halfW))
	path.Close()

	var cs ebiten.ColorScale
	cs.ScaleWithColor(clr)
	vector.FillPath(screen, &path, &vector.FillOptions{}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: cs})
}

func drawHamburgerMenuButton(screen *ebiten.Image, game *Game) {
	x, y, w, h := touchMenuCenterX-touchMenuSize/2, touchMenuCenterY-touchMenuSize/2, touchMenuSize, touchMenuSize
	ebitenutil.DrawRect(screen, x, y, w, h, color.NRGBA{0, 0, 0, 170})
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), touchMenuBorderWidth, color.White, true)

	barW := touchMenuSize * 0.6
	barH := 3.0
	barX := touchMenuCenterX - barW/2
	for _, dy := range []float64{-6, 0, 6} {
		barY := touchMenuCenterY + dy - barH/2
		ebitenutil.DrawRect(screen, barX, barY, barW, barH, color.NRGBA{255, 255, 255, 220})
	}

	if !game.MobileMode {
		op := &text.DrawOptions{}
		op.GeoM.Translate(touchMenuCenterX, touchMenuCenterY+touchMenuSize/2+touchMenuKeyGapY)
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "M", game.FontFace(ctrlKeyFontSize), op)
	}
}

func (s *FieldScene) drawTouchControls(screen *ebiten.Image, showPad bool, showActionButton bool) {
	if showPad {
		baseColor := color.NRGBA{0, 0, 0, 170}
		arrowColor := color.NRGBA{255, 255, 255, 210}
		knobColor := color.NRGBA{255, 255, 255, 235}
		if s.touchStickActive {
			knobColor = color.NRGBA{255, 255, 255, 255}
		}

		vector.FillCircle(screen, float32(touchPadCenterX), float32(touchPadCenterY), float32(stickBaseR), baseColor, true)

		drawStickArrow(screen, touchPadCenterX, touchPadCenterY, 0, -1, arrowColor)
		drawStickArrow(screen, touchPadCenterX, touchPadCenterY, 0, 1, arrowColor)
		drawStickArrow(screen, touchPadCenterX, touchPadCenterY, -1, 0, arrowColor)
		drawStickArrow(screen, touchPadCenterX, touchPadCenterY, 1, 0, arrowColor)

		knobX := touchPadCenterX + s.touchStickDX
		knobY := touchPadCenterY + s.touchStickDY
		vector.FillCircle(screen, float32(knobX), float32(knobY), float32(stickKnobR), knobColor, true)
	}

	if showActionButton {
		actionColor := color.NRGBA{255, 255, 255, 60}
		for _, p := range activeTouchPoints() {
			if p.inCircle(touchActionX, touchActionY, touchActionR) {
				actionColor = color.NRGBA{255, 255, 255, 130}
			}
		}
		ebitenutil.DrawRect(screen, touchActionX-touchActionR, touchActionY-touchActionR, touchActionR*2, touchActionR*2, actionColor)
	}
}
