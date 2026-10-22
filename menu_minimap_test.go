package main

import "testing"

func TestBuildMinimapPixelsReturnsRGBAForEveryTile(t *testing.T) {
	field := &FieldScene{
		tileMap: TiledMap{
			Width:      2,
			Height:     3,
			TileWidth:  32,
			TileHeight: 32,
		},
	}

	pixels := buildMinimapPixels(field)
	if got, want := len(pixels), 2*3*4; got != want {
		t.Fatalf("pixel buffer length = %d, want %d", got, want)
	}
}
