package main

import (
	"encoding/json"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const settingsFilePath = "settings.json"

// ウィンドウがリサイズされてから、実際に設定ファイルへ保存するまでの猶予フレーム数。
// ドラッグ中に毎フレーム書き込まないための調整用。
const windowResizeSettleTicks = 30

type GameSettings struct {
	BGMVolume    float64 `json:"bgmVolume"`
	MessageSpeed int     `json:"messageSpeed"`
	Fullscreen   bool    `json:"fullscreen"`
	WindowWidth  int     `json:"windowWidth"`
	WindowHeight int     `json:"windowHeight"`
}

func LoadSettings() GameSettings {
	s := GameSettings{
		BGMVolume:    defaultBGMVolume,
		MessageSpeed: defaultMessageSpeed,
		Fullscreen:   defaultFullscreen,
		WindowWidth:  defaultWindowWidth,
		WindowHeight: defaultWindowHeight,
	}
	data, err := os.ReadFile(settingsFilePath)
	if err != nil {
		return s // ファイルがなければデフォルトを返す
	}
	_ = json.Unmarshal(data, &s)
	if s.WindowWidth <= 0 || s.WindowHeight <= 0 {
		s.WindowWidth = defaultWindowWidth
		s.WindowHeight = defaultWindowHeight
	}
	return s
}

func SaveSettings(s GameSettings) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(settingsFilePath, data, 0644)
}

// applyDisplayMode はフルスクリーン/ウィンドウを切り替える。
// ウィンドウモードの場合は指定サイズへ変更したうえで画面中央に配置する。
func applyDisplayMode(fullscreen bool, winW, winH int) {
	ebiten.SetFullscreen(fullscreen)
	if fullscreen {
		return
	}

	ebiten.SetWindowSize(winW, winH)

	// 実際に適用されたウィンドウサイズを取得して中央配置する
	actualW, actualH := ebiten.WindowSize()
	monW, monH := ebiten.Monitor().Size()
	x := (monW - actualW) / 2
	y := (monH - actualH) / 2
	ebiten.SetWindowPosition(x, y)
}

// updateWindowSizeTracking はドラッグでウィンドウサイズが変わった際、
// サイズが落ち着いてから設定ファイルへ保存する（毎フレーム書き込みを避けるため）。
func (g *Game) updateWindowSizeTracking() {
	if g.Fullscreen {
		return
	}
	w, h := ebiten.WindowSize()
	if w <= 0 || h <= 0 {
		return
	}
	if w != g.lastWindowW || h != g.lastWindowH {
		g.lastWindowW, g.lastWindowH = w, h
		g.windowResizeSettleTimer = windowResizeSettleTicks
		return
	}
	if g.windowResizeSettleTimer > 0 {
		g.windowResizeSettleTimer--
		if g.windowResizeSettleTimer == 0 {
			g.WindowWidth, g.WindowHeight = w, h
			g.SaveGameSettings()
		}
	}
}

// SaveGameSettings は現在のゲーム状態を settings.json に書き出す。
func (g *Game) SaveGameSettings() {
	vol := defaultBGMVolume
	if g.Audio != nil {
		vol = g.Audio.volume
	}
	SaveSettings(GameSettings{
		BGMVolume:    vol,
		MessageSpeed: g.MessageSpeed,
		Fullscreen:   g.Fullscreen,
		WindowWidth:  g.WindowWidth,
		WindowHeight: g.WindowHeight,
	})
}
