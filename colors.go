package main

import "image/color"

var (
	uiColorText = color.RGBA{255, 255, 255, 255}

	uiColorSelect = color.RGBA{255, 220, 80, 255}

	uiColorDisabled = color.RGBA{140, 140, 140, 255}

	// 未解放のスキルレベル専用（使える/使えないとは無関係）
	uiColorLocked = color.RGBA{130, 110, 200, 255}

	uiColorDead =color.RGBA{90, 70, 70, 255}

	uiColorDanger = color.RGBA{255, 100, 100, 255}

	uiColorPanelBg = color.NRGBA{0, 0, 0, 210}

	uiColorConfirmBg = color.NRGBA{0, 0, 0, 240}

	uiColorMenuLine = color.RGBA{235, 240, 240, 230}
)
