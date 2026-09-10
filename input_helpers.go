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

func isConfirmKeyPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyZ)
}

func isConfirmKeyDown() bool {
	return ebiten.IsKeyPressed(ebiten.KeyEnter) ||
		ebiten.IsKeyPressed(ebiten.KeyNumpadEnter) ||
		ebiten.IsKeyPressed(ebiten.KeySpace) ||
		ebiten.IsKeyPressed(ebiten.KeyZ)
}

func isEscapePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyX)
}

func isSkipKeyDown() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControl) ||
		ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight)
}

// isAutoTogglePressed は会話オート送りのON/OFFを切り替えるキーが「今押された瞬間」かを返す
func isAutoTogglePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyA)
}

// isLogTogglePressed は会話ログ画面の開閉キーが「今押された瞬間」かを返す
func isLogTogglePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyL)
}

// isDashTogglePressed はダッシュのオン/オフを切り替えるキーが「今押された瞬間」かを返す
func isDashTogglePressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyShift) ||
		inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyShiftRight)
}
