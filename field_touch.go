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
	// touchPadCenterX/Y is the resting position shown before the player
	// touches down. The stick itself floats to wherever the player
	// actually places their thumb within touchStickZone*, so they don't
	// need to look down and hit an exact spot.
	touchPadCenterX = 110.0
	touchPadCenterY = float64(gameHeight) - 110.0
	stickBaseR      = 70.0
	stickKnobR      = 32.0
	stickDeadzone   = 8.0
	stickArrowSize  = 18.0

	// stickWalkFullAtRatio: pushing the stick past this fraction of its
	// radius already yields full walk speed, so small nudges give fine,
	// analog-feeling control while the rest of the throw is "free".
	stickWalkFullAtRatio = 0.5

	// Touches starting anywhere in this zone (roughly the left/lower
	// portion of the screen, clear of the action and menu buttons) spawn
	// the stick right under the thumb.
	touchStickZoneX = float64(gameWidth) * 0.62
	touchStickZoneY = 0.0

	touchActionX = float64(gameWidth) - 96.0
	touchActionY = float64(gameHeight) - 96.0
	touchActionR = 50.0

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

func activeTouchAndMousePoints() (touches []touchPoint, mouse []touchPoint) {
	for _, id := range ebiten.AppendTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		touches = append(touches, touchPoint{float64(x), float64(y)})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
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

func pointInStickZone(x, y float64) bool {
	return x >= 0 && x < touchStickZoneX && y >= touchStickZoneY && y < float64(gameHeight)
}

// stickOriginForTouch clamps the touch-down point so the whole stick base
// stays on screen, letting the joystick float to wherever the thumb lands.
func stickOriginForTouch(x, y float64) (float64, float64) {
	margin := stickBaseR + 6.0
	if x < margin {
		x = margin
	}
	if x > touchStickZoneX {
		x = touchStickZoneX
	}
	if y < margin {
		y = margin
	}
	if y > float64(gameHeight)-margin {
		y = float64(gameHeight) - margin
	}
	return x, y
}

func (s *FieldScene) setStickFromPoint(x, y float64) {
	dx := x - s.touchStickOriginX
	dy := y - s.touchStickOriginY
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
	s.touchStickOriginX = touchPadCenterX
	s.touchStickOriginY = touchPadCenterY
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
		if pointInStickZone(float64(x), float64(y)) {
			s.touchStickTouchID = id
			s.touchStickOriginX, s.touchStickOriginY = stickOriginForTouch(float64(x), float64(y))
			s.setStickFromPoint(float64(x), float64(y))
			return
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if pointInStickZone(float64(x), float64(y)) {
			s.touchStickUseMouse = true
			s.touchStickOriginX, s.touchStickOriginY = stickOriginForTouch(float64(x), float64(y))
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

const dashThreshold = stickBaseR - 10

func (s *FieldScene) touchStickDash() bool {
	if !s.touchStickActive {
		return false
	}
	return math.Hypot(s.touchStickDX, s.touchStickDY) >= dashThreshold
}

// touchStickSpeedScale gives analog control near the center of the stick:
// a light push moves slowly and precisely, while anything past
// stickWalkFullAtRatio of the radius already moves at full walk speed.
func (s *FieldScene) touchStickSpeedScale() float64 {
	if !s.touchStickActive {
		return 0
	}
	dist := math.Hypot(s.touchStickDX, s.touchStickDY)
	if dist <= stickDeadzone {
		return 0
	}
	fullAt := stickBaseR * stickWalkFullAtRatio
	if dist >= fullAt {
		return 1
	}
	t := (dist - stickDeadzone) / (fullAt - stickDeadzone)
	return t * t * (3 - 2*t)
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

// drawRestingStickHint shows a light, unobtrusive marker at the default
// stick position so the player still has a visual anchor, even though a
// touch anywhere in the movement zone will spawn the stick right there.
func drawRestingStickHint(screen *ebiten.Image) {
	drawRingOutline(screen, touchPadCenterX, touchPadCenterY, stickBaseR*0.7, color.NRGBA{255, 255, 255, 70})
	vector.FillCircle(screen, float32(touchPadCenterX), float32(touchPadCenterY), float32(stickKnobR*0.5), color.NRGBA{255, 255, 255, 60}, true)
}

func (s *FieldScene) drawActiveStick(screen *ebiten.Image) {
	cx, cy := s.touchStickOriginX, s.touchStickOriginY
	dashing := s.touchStickDash()

	baseColor := color.NRGBA{0, 0, 0, 150}
	arrowColor := color.NRGBA{255, 255, 255, 210}
	knobColor := color.NRGBA{255, 255, 255, 255}
	if dashing {
		knobColor = color.NRGBA{255, 214, 110, 255}
	}

	vector.FillCircle(screen, float32(cx), float32(cy), float32(stickBaseR), baseColor, true)
	drawRingOutline(screen, cx, cy, stickBaseR, color.NRGBA{255, 255, 255, 90})

	drawStickArrow(screen, cx, cy, 0, -1, arrowColor)
	drawStickArrow(screen, cx, cy, 0, 1, arrowColor)
	drawStickArrow(screen, cx, cy, -1, 0, arrowColor)
	drawStickArrow(screen, cx, cy, 1, 0, arrowColor)

	knobX := cx + s.touchStickDX
	knobY := cy + s.touchStickDY
	if dashing {
		vector.FillCircle(screen, float32(knobX), float32(knobY), float32(stickKnobR+4), color.NRGBA{255, 214, 110, 70}, true)
	}
	vector.FillCircle(screen, float32(knobX), float32(knobY), float32(stickKnobR), knobColor, true)
}

func drawActionButton(screen *ebiten.Image, pressed bool) {
	fillColor := color.NRGBA{255, 255, 255, 60}
	ringAlpha := uint8(160)
	if pressed {
		fillColor = color.NRGBA{255, 255, 255, 140}
		ringAlpha = 230
	}
	vector.FillCircle(screen, float32(touchActionX), float32(touchActionY), float32(touchActionR), fillColor, true)
	drawRingOutline(screen, touchActionX, touchActionY, touchActionR, color.NRGBA{255, 255, 255, ringAlpha})
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
		text.Draw(screen, "M", game.LatinFontFace(ctrlLabelFontSize), op)
	}
}

func (s *FieldScene) drawTouchControls(screen *ebiten.Image, showPad bool, showActionButton bool) {
	if showPad {
		if s.touchStickActive {
			s.drawActiveStick(screen)
		} else {
			drawRestingStickHint(screen)
		}
	}

	if showActionButton {
		pressed := false
		for _, p := range activeTouchPoints() {
			if p.inCircle(touchActionX, touchActionY, touchActionR) {
				pressed = true
			}
		}
		drawActionButton(screen, pressed)
	}
}
