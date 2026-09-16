package main

import "testing"

func TestCollisionRectAt(t *testing.T) {
	cfg := DefaultFieldPlayerConfig()
	left, top, right, bottom := cfg.CollisionRectAt(100, 200)
	if left >= right || top >= bottom {
		t.Fatalf("invalid rect: %v %v %v %v", left, top, right, bottom)
	}
}

func TestResolveEmbeddedPositionEscapesWall(t *testing.T) {
	cfg := DefaultFieldPlayerConfig()
	s := &FieldScene{
		px:        20,
		py:        20,
		playerCfg: cfg,
		tileMap:   TiledMap{Width: 100, Height: 100, TileWidth: 16, TileHeight: 16},
		collisions: []CollisionRect{{
			X: 10, Y: 10, Width: 20, Height: 20,
		}},
	}

	if !s.isWall(s.px, s.py) {
		t.Fatal("expected starting position to be inside the wall collision")
	}

	s.resolveEmbeddedPosition()
	if s.isWall(s.px, s.py) {
		t.Fatalf("player should escape wall collision after resolve; px=%v py=%v", s.px, s.py)
	}
	if s.px == 20 && s.py == 20 {
		t.Fatal("resolveEmbeddedPosition should move player out of the wall")
	}
}
