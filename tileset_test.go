package main

import "testing"

const testTSX = `<?xml version="1.0" encoding="UTF-8"?>
<tileset version="1.10" tiledversion="1.12.1" name="t" tilewidth="32" tileheight="32" spacing="2" margin="1" tilecount="4" columns="2">
 <image source="../images/tiles/t.png" width="68" height="68"/>
 <tile id="3">
  <properties>
   <property name="wall" type="bool" value="true"/>
  </properties>
 </tile>
</tileset>`

func TestParseTSX(t *testing.T) {
	def, err := parseTSX([]byte(testTSX))
	if err != nil {
		t.Fatal(err)
	}
	ts := mapTileset{firstGID: 10, def: def}
	if !ts.contains(10) || !ts.contains(13) || ts.contains(14) || ts.contains(9) {
		t.Errorf("contains: firstgid 10, tilecount 4 で 10〜13 だけを含むこと")
	}
	// 4番目(localID 3)は2列目・2行目。margin 1 + (32+spacing 2)。
	if x0, y0, x1, y1 := ts.tileRect(3); x0 != 35 || y0 != 35 || x1 != 67 || y1 != 67 {
		t.Errorf("tileRect(3) = %d,%d,%d,%d", x0, y0, x1, y1)
	}
	walls := tilesetWallGIDs([]mapTileset{ts})
	if !walls[13] || len(walls) != 1 {
		t.Errorf("wall=true のタイル(GID 13)だけが壁になること: %v", walls)
	}
	if got := resolveRelativeAssetPath("assets/tilesets/t.tsx", def.Image); got != "assets/images/tiles/t.png" {
		t.Errorf("画像パス = %s", got)
	}
}

// wall=true のタイルは、kabe以外のレイヤーに置いても(反転していても)通れないこと。
func TestWallTilePropertyBlocks(t *testing.T) {
	s := &FieldScene{
		tileMap: TiledMap{
			Width: 2, Height: 1, TileWidth: 32, TileHeight: 32,
			Layers:   []TiledLayer{{Name: "水", Type: "tilelayer", Data: []int{0, 5 | gidFlipH}}},
			wallGIDs: map[int]bool{5: true},
		},
	}
	if s.rectHitsObstacles(4, 4, 28, 28, "") {
		t.Error("何も無いマスが壁になっている")
	}
	if !s.rectHitsObstacles(36, 4, 60, 28, "") {
		t.Error("wall=true のタイルを通り抜けられる")
	}
}
