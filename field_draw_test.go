package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestFieldSceneDrawBossEnemyDoesNotPanic(t *testing.T) {
	g := &Game{}
	scene := &FieldScene{
		game: g,
		enemies: []*EnemyField{{
			Type: "boss_1",
			x:    0,
			y:    0,
		}},
		playerCfg: FieldPlayerConfig{},
	}

	img := ebiten.NewImage(64, 64)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw panicked for boss_1 enemy: %v", r)
		}
	}()

	scene.Draw(img)
}
