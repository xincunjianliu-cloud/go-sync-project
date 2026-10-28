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

func TestPolygonCollisionDetectsCrossingWithoutContainedCorners(t *testing.T) {
	s := &FieldScene{
		tileMap: TiledMap{Width: 20, Height: 20, TileWidth: 16, TileHeight: 16},
		collisionPolygons: []CollisionPolygon{{
			Points: []TiledPoint{
				{X: 50, Y: 60},
				{X: 110, Y: 64},
				{X: 110, Y: 68},
				{X: 50, Y: 64},
			},
		}},
	}

	if !s.rectHitsTerrain(75, 55, 85, 75) {
		t.Fatal("a rectangle crossing a thin polygon should collide even when all four corners are outside")
	}
}

func TestRectIntersectsPolygonWhenPolygonIsInsideRect(t *testing.T) {
	poly := []TiledPoint{{X: 4, Y: 4}, {X: 6, Y: 4}, {X: 6, Y: 6}, {X: 4, Y: 6}}
	if !rectIntersectsPolygon(0, 0, 10, 10, poly) {
		t.Fatal("polygon fully enclosed by rectangle should count as an intersection")
	}
	if rectIntersectsPolygon(11, 11, 12, 12, poly) {
		t.Fatal("separated rectangle should not intersect polygon")
	}
}
