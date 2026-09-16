package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func isMenuUpPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW)
}

func isMenuDownPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS)
}

func isMenuLeftPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA)
}

func isMenuRightPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD)
}

const (
	menuCursorRepeatDelayTicks    = 20
	menuCursorRepeatIntervalTicks = 5
)

func keysRepeatFire(keys ...ebiten.Key) bool {
	ticks := 0
	for _, k := range keys {
		if d := inpututil.KeyPressDuration(k); d > ticks {
			ticks = d
		}
	}
	return repeatFires(ticks, menuCursorRepeatDelayTicks, menuCursorRepeatIntervalTicks)
}

func isMenuUpRepeat() bool {
	return keysRepeatFire(ebiten.KeyUp, ebiten.KeyW)
}

func isMenuDownRepeat() bool {
	return keysRepeatFire(ebiten.KeyDown, ebiten.KeyS)
}

func isMenuCloseKeyPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyM)
}

func repeatFires(ticks, delay, every int) bool {
	if ticks <= 0 {
		return false
	}
	if ticks == 1 {
		return true
	}
	if ticks <= delay {
		return false
	}
	return (ticks-delay)%every == 0
}

func isConfirmKeyPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyZ) ||
		fieldTouchActionPressed()
}

func isConfirmKeyDown() bool {
	return ebiten.IsKeyPressed(ebiten.KeyEnter) ||
		ebiten.IsKeyPressed(ebiten.KeyNumpadEnter) ||
		ebiten.IsKeyPressed(ebiten.KeySpace) ||
		ebiten.IsKeyPressed(ebiten.KeyZ)
}

func isEscapePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyX) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) ||
		isTouchBackPressed()
}

func isSkipKeyDown() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControl) ||
		ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight)
}

func isAutoTogglePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyA)
}

func isLogTogglePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyL)
}

func isDashTogglePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyShift) ||
		inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyShiftRight)
}

func pressedDigitKey() (int, bool) {
	keys := [9]ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3,
		ebiten.Key4, ebiten.Key5, ebiten.Key6,
		ebiten.Key7, ebiten.Key8, ebiten.Key9,
	}
	for i, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return i + 1, true
		}
	}
	return 0, false
}
