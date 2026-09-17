package main

import "testing"

func newBlockDoorTestScene() (*FieldScene, *FieldBlock) {
	spotObj := TiledObject{
		X: 100, Y: 100, Width: 32, Height: 32,
		Properties: []TiledProperty{
			{Name: "type", Type: "string", Value: "event"},
			{Name: "text", Type: "string", Value: "event_blockspot"},
			{Name: "id", Type: "string", Value: "spot1"},
		},
	}
	doorObj := TiledObject{
		X: 300, Y: 300, Width: 32, Height: 32,
		Properties: []TiledProperty{
			{Name: "type", Type: "string", Value: "event"},
			{Name: "text", Type: "string", Value: "event_blockdoor"},
			{Name: "spots", Type: "string", Value: "spot1"},
		},
	}

	g := &Game{UnlockedBlockDoors: make(map[string]bool)}
	block := &FieldBlock{ID: "block1", X: spotObj.X, Y: spotObj.Y, Width: 32, Height: 32}
	s := &FieldScene{
		game:       g,
		currentMap: "test_map",
		blocks:     []*FieldBlock{block},
		tileMap: TiledMap{
			Width: 100, Height: 100, TileWidth: 16, TileHeight: 16,
			Layers: []TiledLayer{
				{Name: "events", Type: "objectgroup", Objects: []TiledObject{spotObj, doorObj}},
			},
		},
	}
	return s, block
}

func TestBlockBecomesLockedOnceDoorOpens(t *testing.T) {
	s, block := newBlockDoorTestScene()

	if s.blockIsLocked(block) {
		t.Fatal("expected block not to be locked before its door has opened")
	}

	s.updateBlockDoors()
	doorObj := s.tileMap.Layers[0].Objects[1]
	if !s.blockDoorIsOpen(doorObj) {
		t.Fatal("expected block door to unlock once its spot is filled")
	}

	if !s.blockIsLocked(block) {
		t.Fatal("expected block to become locked once its door has opened")
	}
}
