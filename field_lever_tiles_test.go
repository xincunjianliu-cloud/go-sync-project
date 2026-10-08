package main

import (
	"strings"
	"testing"
)

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

// Tiledで非表示にしたレイヤーはゲームでも描かない。leveropenレイヤーだけは
// レバーで出し入れするので、非表示でも描く。
func TestHiddenLayersAreNotDrawn(t *testing.T) {
	no, yes := false, true
	if (TiledLayer{Visible: &no}).drawn() {
		t.Error("非表示のレイヤーが描かれます")
	}
	if !(TiledLayer{Visible: &yes}).drawn() || !(TiledLayer{}).drawn() {
		t.Error("表示のレイヤーが描かれません")
	}
	lever := TiledLayer{Visible: &no, Properties: []TiledProperty{{Name: "leveropen", Value: true}}}
	if !lever.drawn() {
		t.Error("非表示のleveropenレイヤーが描かれません")
	}
}

// グループレイヤー(Tiledのフォルダ)の中のレイヤーも、順番どおりに使われる。
// グループの非表示・不透明度は中のレイヤーに引き継がれる。
func TestFlattenLayerGroups(t *testing.T) {
	no := false
	half := 0.5
	layers := flattenLayerGroups([]TiledLayer{
		{Name: "床", Type: "tilelayer"},
		{Name: "グループ", Type: "group", Opacity: &half, Layers: []TiledLayer{
			{Name: "events", Type: "objectgroup", Properties: eventLayerProps},
			{Name: "中のグループ", Type: "group", Visible: &no, Layers: []TiledLayer{{Name: "影", Type: "tilelayer"}}},
		}},
		{Name: "屋根", Type: "tilelayer"},
	})
	var names []string
	for _, l := range layers {
		names = append(names, l.Name)
	}
	if got := strings.Join(names, ","); got != "床,events,影,屋根" {
		t.Fatalf("並び = %s", got)
	}
	if layers[2].drawn() {
		t.Error("非表示のグループの中のレイヤーが描かれます")
	}
	if a := layers[2].opacity(); a != 0.5 {
		t.Errorf("不透明度 = %v(グループの0.5を引き継ぐはず)", a)
	}
}

// eventLayerProps は、クラス「しかけレイヤー」を読み替えたあとの名札(テスト用)。
var eventLayerProps = []TiledProperty{{Name: "eventlayer", Type: "string", Value: "true"}}
