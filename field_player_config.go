package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// 方向: 0=下, 1=左, 2=右, 3=上
type FieldDirectionConfig struct {
	IdleRow int `json:"idleRow"`
	WalkRow int `json:"walkRow"`
	DashRow int `json:"dashRow"`
	Frames  int `json:"frames"`
}

type PlayerMoveState int

const (
	MoveStateIdle PlayerMoveState = iota
	MoveStateWalk
	MoveStateDash
)

type FieldPlayerConfig struct {
	Sprite      string  `json:"sprite"`
	FrameWidth  int     `json:"frameWidth"`
	FrameHeight int     `json:"frameHeight"`
	Scale       float64 `json:"scale"`

	Directions [4]FieldDirectionConfig `json:"directions"`

	Draw struct {
		FootOffsetX float64 `json:"footOffsetX"`
		FootOffsetY float64 `json:"footOffsetY"`
	} `json:"draw"`

	Collision struct {
		Width   float64 `json:"width"`
		Height  float64 `json:"height"`
		OffsetX float64 `json:"offsetX"`
		OffsetY float64 `json:"offsetY"`
	} `json:"collision"`

	Camera struct {
		OffsetX float64 `json:"offsetX"`
		OffsetY float64 `json:"offsetY"`
	} `json:"camera"`

	AnimFramesPerStep   int     `json:"animFramesPerStep"`
	MoveSpeed           float64 `json:"moveSpeed"`
	DashSpeedMultiplier float64 `json:"dashSpeedMultiplier"`
}

func DefaultFieldPlayerConfig() FieldPlayerConfig {
	return FieldPlayerConfig{
		Sprite:      "assets/images/player_walk.png",
		FrameWidth:  64,
		FrameHeight: 96,
		Scale:       1,
		Directions: [4]FieldDirectionConfig{
			{IdleRow: 0, WalkRow: 4, DashRow: 8, Frames: 4},
			{IdleRow: 1, WalkRow: 5, DashRow: 9, Frames: 4},
			{IdleRow: 2, WalkRow: 6, DashRow: 10, Frames: 4},
			{IdleRow: 3, WalkRow: 7, DashRow: 11, Frames: 4},
		},
		Draw: struct {
			FootOffsetX float64 `json:"footOffsetX"`
			FootOffsetY float64 `json:"footOffsetY"`
		}{FootOffsetX: 16, FootOffsetY: 10},
		Collision: struct {
			Width   float64 `json:"width"`
			Height  float64 `json:"height"`
			OffsetX float64 `json:"offsetX"`
			OffsetY float64 `json:"offsetY"`
		}{Width: 20, Height: 12, OffsetX: 0, OffsetY: -4},
		Camera: struct {
			OffsetX float64 `json:"offsetX"`
			OffsetY float64 `json:"offsetY"`
		}{OffsetX: 0, OffsetY: -14},
		AnimFramesPerStep:   10,
		MoveSpeed:           300,
		DashSpeedMultiplier: 1.8,
	}
}
func LoadFieldPlayerConfig(path string) (FieldPlayerConfig, *ebiten.Image, error) {
	cfg := DefaultFieldPlayerConfig()

	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, nil, fmt.Errorf("field player config parse: %w", err)
		}
	}

	if cfg.FrameWidth <= 0 || cfg.FrameHeight <= 0 {
		return cfg, nil, fmt.Errorf("invalid frame size")
	}
	if cfg.Scale <= 0 {
		cfg.Scale = 1
	}
	for i := range cfg.Directions {
		if cfg.Directions[i].Frames <= 0 {
			cfg.Directions[i].Frames = 1
		}
	}
	if cfg.AnimFramesPerStep <= 0 {
		cfg.AnimFramesPerStep = 10
	}
	if cfg.MoveSpeed <= 0 {
		cfg.MoveSpeed = 300
	}

	if cfg.DashSpeedMultiplier <= 0 {
		cfg.DashSpeedMultiplier = 1.8
	}

	img, _, err := ebitenutil.NewImageFromFile(cfg.Sprite)
	if err != nil {
		return cfg, nil, fmt.Errorf("load player sprite %q: %w", cfg.Sprite, err)
	}

	return cfg, img, nil
}

func (cfg FieldPlayerConfig) CollisionRectAt(px, py float64) (left, top, right, bottom float64) {
	cx := px + cfg.Collision.OffsetX
	cy := py + cfg.Collision.OffsetY
	return cx, cy, cx + cfg.Collision.Width, cy + cfg.Collision.Height
}

func (cfg FieldPlayerConfig) DrawTopLeftAt(px, py float64) (x, y float64) {
	fw := float64(cfg.FrameWidth) * cfg.Scale
	fh := float64(cfg.FrameHeight) * cfg.Scale
	footX := px + cfg.Draw.FootOffsetX
	footY := py + cfg.Draw.FootOffsetY
	return footX - fw/2, footY - fh
}

func (cfg FieldPlayerConfig) CameraAnchorAt(px, py float64) (x, y float64) {
	return px + cfg.Camera.OffsetX, py + cfg.Camera.OffsetY
}

func (cfg FieldPlayerConfig) AnimFrame(dir, animCount int) int {
	if dir < 0 || dir >= 4 {
		dir = 0
	}
	if animCount <= 0 {
		return 0
	}
	count := cfg.Directions[dir].Frames
	if count <= 0 {
		count = 1
	}
	return (animCount / cfg.AnimFramesPerStep) % count
}

func (cfg FieldPlayerConfig) SourceRect(dir int, state PlayerMoveState, frame int) (sx, sy, sw, sh int) {
	if dir < 0 || dir >= 4 {
		dir = 0
	}
	row := cfg.Directions[dir].IdleRow
	switch state {
	case MoveStateWalk:
		row = cfg.Directions[dir].WalkRow
		if frame >= cfg.Directions[dir].Frames {
			frame = 0
		}
	case MoveStateDash:
		row = cfg.Directions[dir].DashRow
		if frame >= cfg.Directions[dir].Frames {
			frame = 0
		}
	default:
		frame = 0
	}
	return frame * cfg.FrameWidth, row * cfg.FrameHeight, cfg.FrameWidth, cfg.FrameHeight
}
