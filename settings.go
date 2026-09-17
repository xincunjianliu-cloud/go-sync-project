package main

import (
	"encoding/json"

	"github.com/hajimehoshi/ebiten/v2"
)

const settingsFilePath = "settings.json"

const windowResizeSettleTicks = 30

type GameSettings struct {
	BGMVolume      float64 `json:"bgmVolume"`
	SEVolume       float64 `json:"seVolume"`
	MasterVolume   float64 `json:"masterVolume"`
	MessageSpeed   int     `json:"messageSpeed"`
	Fullscreen     bool    `json:"fullscreen"`
	WindowWidth    int     `json:"windowWidth"`
	WindowHeight   int     `json:"windowHeight"`
	RememberCursor bool    `json:"rememberCursor"`
}

func LoadSettings() GameSettings {
	s := GameSettings{
		BGMVolume:      defaultBGMVolume,
		SEVolume:       defaultSEVolume,
		MasterVolume:   defaultMasterVolume,
		MessageSpeed:   defaultMessageSpeed,
		Fullscreen:     defaultFullscreen,
		WindowWidth:    defaultWindowWidth,
		WindowHeight:   defaultWindowHeight,
		RememberCursor: defaultRememberCursor,
	}
	data, err := readRuntimeFile(settingsFilePath)
	if err != nil {
		return s
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
	_ = writeRuntimeFile(settingsFilePath, data)
}

func applyDisplayMode(fullscreen bool, winW, winH int) {
	ebiten.SetFullscreen(fullscreen)
	if fullscreen {
		return
	}

	ebiten.SetWindowSize(winW, winH)

	actualW, actualH := ebiten.WindowSize()
	monW, monH := ebiten.Monitor().Size()
	x := (monW - actualW) / 2
	y := (monH - actualH) / 2
	ebiten.SetWindowPosition(x, y)
}

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

func (g *Game) SaveGameSettings() {
	vol := defaultBGMVolume
	seVol := defaultSEVolume
	masterVol := defaultMasterVolume
	if g.Audio != nil {
		vol = g.Audio.volume
		seVol = g.Audio.seVolume
		masterVol = g.Audio.masterVolume
	}
	SaveSettings(GameSettings{
		BGMVolume:      vol,
		SEVolume:       seVol,
		MasterVolume:   masterVol,
		MessageSpeed:   g.MessageSpeed,
		Fullscreen:     g.Fullscreen,
		WindowWidth:    g.WindowWidth,
		WindowHeight:   g.WindowHeight,
		RememberCursor: g.RememberCursor,
	})
}
