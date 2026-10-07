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

// Tiledのタイルアニメーションを読み、時間に合わせてコマが進むこと。
func TestTileAnimation(t *testing.T) {
	def, err := parseTSX([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<tileset name="water" tilewidth="32" tileheight="32" tilecount="4" columns="2">
 <image source="../images/tiles/water.png" width="64" height="64"/>
 <tile id="0">
  <animation>
   <frame tileid="0" duration="200"/>
   <frame tileid="1" duration="300"/>
  </animation>
 </tile>
</tileset>`))
	if err != nil {
		t.Fatal(err)
	}
	anims := tilesetAnimations([]mapTileset{{firstGID: 10, def: def}})
	frames := anims[10]
	if len(frames) != 2 || frames[0].GID != 10 || frames[1].GID != 11 {
		t.Fatalf("コマ = %+v", frames)
	}
	for ms, want := range map[int64]int{0: 10, 199: 10, 200: 11, 499: 11, 500: 10, 700: 11} {
		if got := animFrameGID(frames, ms); got != want {
			t.Errorf("%dミリ秒のコマ = %d, want %d", ms, got, want)
		}
	}
}
