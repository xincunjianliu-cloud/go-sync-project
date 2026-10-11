package main

import "testing"

// 「上に絵を描いた所は通れる」水のレイヤーは、上に床・橋を描いたマスだけ通れる。
// 主人公より上のレイヤー・非表示のレイヤーの絵は数えない。
// チェックの無い通れないレイヤー(壁)は、上に絵があっても通れない。
func TestWalkOnTopLayer(t *testing.T) {
	const w, h = 8, 4
	fill := func(cells ...int) []int {
		d := make([]int, w*h)
		for _, i := range cells {
			d[i] = 1
		}
		return d
	}
	all := make([]int, w*h)
	for i := range all {
		all[i] = i
	}
	flags := func(names ...string) []TiledProperty {
		var ps []TiledProperty
		for _, n := range names {
			ps = append(ps, TiledProperty{Name: n, Type: "bool", Value: true})
		}
		return ps
	}
	hidden := false
	row := w * 1 // 2行目
	s := &FieldScene{
		tileMap: TiledMap{Width: w, Height: h, TileWidth: 32, TileHeight: 32, Layers: []TiledLayer{
			{Type: "tilelayer", Data: fill(all[:w*h/2]...), Properties: flags("blocking", "walkontop")}, // 水(上半分)
			{Type: "tilelayer", Data: fill(all[w*h/2:]...), Properties: flags("blocking")},              // 壁(下半分)
			{Type: "tilelayer", Data: fill(row+2, 3*w+2)},                                               // 橋(2行目3列)と壁の上の飾り
			{Type: "tilelayer", Data: fill(row + 4), Visible: &hidden},                                  // 非表示
			{Type: "objectgroup", Properties: flags("playerlayer")},
			{Type: "tilelayer", Data: fill(row + 6)}, // 主人公より上
		}},
	}

	cases := []struct {
		name        string
		l, t, r, b  float64
		wantBlocked bool
	}{
		{"水の上の橋", 70, 40, 90, 60, false},
		{"橋の外の水", 6, 40, 26, 60, true},
		{"橋から半分はみ出す", 80, 40, 100, 60, true},
		{"非表示のレイヤーの絵", 134, 40, 154, 60, true},
		{"主人公より上の絵", 198, 40, 218, 60, true},
		{"チェックの無い壁の上の飾り", 70, 104, 90, 124, true},
	}
	for _, c := range cases {
		if got := s.rectHitsTerrain(c.l, c.t, c.r, c.b); got != c.wantBlocked {
			t.Errorf("%s: 通れない=%v, want %v", c.name, got, c.wantBlocked)
		}
	}

	// オブジェクトの四角でも同じ。宝箱(WalkOnTop=false)は橋の上でも通れない。
	s.tileMap.Layers[0].Data = make([]int, w*h)
	s.collisions = []CollisionRect{{X: 0, Y: 0, Width: 256, Height: 64, WalkOnTop: true, Layer: 0}}
	if s.rectHitsTerrain(70, 40, 90, 60) {
		t.Error("四角の水の上の橋が通れません")
	}
	if !s.rectHitsTerrain(6, 40, 26, 60) {
		t.Error("四角の水で橋の外が通れます")
	}
	s.collisions = append(s.collisions, CollisionRect{X: 64, Y: 32, Width: 32, Height: 32})
	if !s.rectHitsTerrain(70, 40, 90, 60) {
		t.Error("橋の上の宝箱を通り抜けられます")
	}
}
