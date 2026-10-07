package main

import "testing"

// water_bの机レイヤー(leveropen)は、対応するレバーを上げたときだけ出る。
func TestLeverOpenTilesFollowLever(t *testing.T) {
	tmap, err := loadTiledMap("assets/maps/water_b.tmj")
	if err != nil {
		t.Fatal(err)
	}
	var desk TiledLayer
	for _, layer := range tmap.Layers {
		if isLeverOpenLayer(layer) {
			desk = layer
		}
	}
	if desk.Data == nil {
		t.Fatal("leveropenレイヤーがありません")
	}
	if len(tmap.leverWallsWithTiles) == 0 {
		t.Fatal("机を置いたレバー壁がありません")
	}

	s := &FieldScene{game: &Game{RaisedLevers: map[string]bool{}}, tileMap: tmap}
	cell := -1
	for i, id := range desk.Data {
		if id != 0 {
			cell = i
			break
		}
	}
	lever := tmap.leverCellLevers[cell]
	if s.leverTileShown(cell) {
		t.Errorf("レバー%sを上げる前から机が出ています", lever)
	}
	s.game.RaisedLevers[lever] = true
	if !s.leverTileShown(cell) {
		t.Errorf("レバー%sを上げても机が出ません", lever)
	}
}
