package main

import (
	"math"
	"testing"
)

func TestVisibleTileRangeIncludesEveryVisibleTile(t *testing.T) {
	const (
		mapWidth      = 80
		mapHeight     = 50
		cellWidth     = 32
		cellHeight    = 32
		screenWidth   = 960
		screenHeight  = 540
		maxTileWidth  = 128
		maxTileHeight = 128
	)
	offsets := []float64{-2048, -540, -129, -32, -1, 0, 17, 320, 960, 2048}
	tileSizes := [][2]int{{32, 32}, {64, 64}, {96, 128}, {32, 96}}
	diagonalFlip := []bool{false, true}

	for _, camX := range offsets {
		for _, camY := range offsets {
			minCol, maxCol := visibleTileRange(camX, screenWidth, cellWidth, maxTileWidth, mapWidth)
			minRow, maxRow := visibleTileRange(camY, screenHeight, cellHeight, maxTileHeight, mapHeight)
			for row := range mapHeight {
				for col := range mapWidth {
					for _, size := range tileSizes {
						x := float64(col*cellWidth) + camX
						y := float64((row+1)*cellHeight-size[1]) + camY
						for _, flipped := range diagonalFlip {
							boundsW, boundsH := size[0], size[1]
							if flipped {
								boundsW, boundsH = boundsH, boundsW
							}
							boundsX := x + float64(size[0]-boundsW)/2
							boundsY := y + float64(size[1]-boundsH)/2
							visible := boundsX+float64(boundsW) > 0 && boundsX < screenWidth &&
								boundsY+float64(boundsH) > 0 && boundsY < screenHeight
							if !visible {
								continue
							}
							if col < minCol || col >= maxCol || row < minRow || row >= maxRow {
								t.Fatalf("visible tile (%d,%d), size %v, flipped %t, camera (%v,%v) outside columns [%d,%d), rows [%d,%d)",
									col, row, size, flipped, camX, camY, minCol, maxCol, minRow, maxRow)
							}
						}
					}
				}
			}
		}
	}
}

func TestVisibleTileRangeLimitsWorkToMapAndViewport(t *testing.T) {
	tests := []struct {
		name         string
		cameraOffset float64
		screenSize   float64
		cellSize     int
		maxTileSize  int
		cellCount    int
	}{
		{name: "visible map area", cameraOffset: -1600, screenSize: 960, cellSize: 32, maxTileSize: 96, cellCount: 200},
		{name: "map beyond viewport", cameraOffset: -10000, screenSize: 960, cellSize: 32, maxTileSize: 96, cellCount: 200},
		{name: "viewport before map", cameraOffset: 10000, screenSize: 960, cellSize: 32, maxTileSize: 96, cellCount: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleTileRange(tt.cameraOffset, tt.screenSize, tt.cellSize, tt.maxTileSize, tt.cellCount)
			if start < 0 || end > tt.cellCount || start > end {
				t.Fatalf("invalid range [%d,%d) for %d cells", start, end, tt.cellCount)
			}
			wantMax := int(math.Ceil(tt.screenSize/float64(tt.cellSize))) + 3 +
				int(math.Ceil(float64(tt.maxTileSize)/float64(tt.cellSize)))
			if end-start > wantMax {
				t.Fatalf("range [%d,%d) scans %d cells; want at most %d", start, end, end-start, wantMax)
			}
		})
	}
}
