package main

import (
	"encoding/json"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const settingsFilePath = "settings.json"

type GameSettings struct {
	BGMVolume        float64 `json:"bgmVolume"`
	MessageSpeed     int     `json:"messageSpeed"`
	DisplayModeIndex int     `json:"displayModeIndex"` // 0:等倍 1:1.5倍 2:2倍 3:フルスクリーン
}

func LoadSettings() GameSettings {
	s := GameSettings{
		BGMVolume:        defaultBGMVolume,
		MessageSpeed:     defaultMessageSpeed,
		DisplayModeIndex: 0,
	}
	data, err := os.ReadFile(settingsFilePath)
	if err != nil {
		return s // ファイルがなければデフォルトを返す
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func SaveSettings(s GameSettings) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(settingsFilePath, data, 0644)
}

var displayModeLabels = []string{"ウィンドウ(等倍)", "ウィンドウ(1.5倍)", "ウィンドウ(2倍)", "フルスクリーン"}

func applyDisplayMode(idx int) {
	switch idx {
	case 0:
		ebiten.SetFullscreen(false)
		ebiten.SetWindowSize(gameWidth, gameHeight)
	case 1:
		ebiten.SetFullscreen(false)
		ebiten.SetWindowSize(int(float64(gameWidth)*1.5), int(float64(gameHeight)*1.5))
	case 2:
		ebiten.SetFullscreen(false)
		ebiten.SetWindowSize(gameWidth*2, gameHeight*2)
	case 3:
		ebiten.SetFullscreen(true)
	}

	// ── 追加：ウィンドウモードの場合のみ、画面を中央に配置する ──
	if !ebiten.IsFullscreen() {
		// 変更後のウィンドウサイズを取得
		winW, winH := ebiten.WindowSize()

		// 現在のモニター（ディスプレイ）の解像度を取得
		// ※もしここでエラーが出る場合は、古いEbitengineを使っている可能性があるため、
		// monW, monH := ebiten.ScreenSizeInFullscreen() に書き換えてください。
		monW, monH := ebiten.Monitor().Size()

		// 画面中央になる座標を計算
		x := (monW - winW) / 2
		y := (monH - winH) / 2

		// ウィンドウを中央の座標へ移動
		ebiten.SetWindowPosition(x, y)
	}
}
