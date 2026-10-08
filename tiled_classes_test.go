package main

import (
	"encoding/json"
	"testing"
)

// Tiledでクラスを選んで作ったしかけが、従来の名札に読み替えられること。
// (Tiledは既定値のままの欄を書かないので、書かれていない欄は既定値になる)
func TestTiledClassesBecomeLegacyProperties(t *testing.T) {
	const tmj = `{
	  "class": "マップ設定",
	  "properties": [
	    {"name": "地名", "type": "string", "value": "森"},
	    {"name": "曲", "type": "string", "propertytype": "曲", "value": "フィールド2"},
	    {"name": "入ると全回復", "type": "bool", "value": true}
	  ],
	  "layers": [
	    {"name": "机", "type": "tilelayer", "class": "レバーで出るレイヤー", "data": []},
	    {"name": "events", "type": "objectgroup", "class": "しかけレイヤー", "objects": [
	      {"id": 1, "type": "宝箱", "properties": [{"name": "中身", "type": "string", "propertytype": "アイテム", "value": "ハイポーション"}]},
	      {"id": 2, "type": "ドア", "properties": [
	        {"name": "行き先マップ", "type": "file", "value": "water_b.tmj"},
	        {"name": "着地点", "type": "string", "value": "梯子3"}]},
	      {"id": 3, "type": "レバーの壁", "properties": [
	        {"name": "レバーの名前", "type": "string", "value": "lever_a"},
	        {"name": "見た目だけ", "type": "bool", "value": true}]},
	      {"id": 4, "type": "レバーの壁", "properties": [{"name": "レバーの名前", "type": "string", "value": "lever_b"}]},
	      {"id": 5, "type": "敵が出る場所", "properties": [{"name": "出る敵", "type": "string", "propertytype": "敵", "value": "スライム1,オーク1"}]},
	      {"id": 6, "type": "会話", "properties": [{"name": "セリフ", "type": "string", "value": "こんにちは"}]},
	      {"id": 7, "type": "会話"},
	      {"id": 8, "type": "着地点", "name": "入口", "properties": [{"name": "向き", "type": "string", "propertytype": "向き", "value": "上"}]},
	      {"id": 9, "type": "ボス", "properties": [{"name": "ボス番号", "type": "string", "propertytype": "ボス", "value": "2"}]},
	      {"id": 10, "type": "", "properties": [{"name": "type", "type": "string", "value": "event"}, {"name": "text", "type": "string", "value": "従来の書き方"}]}
	    ]}
	  ]
	}`
	var tmap TiledMap
	if err := json.Unmarshal([]byte(tmj), &tmap); err != nil {
		t.Fatal(err)
	}
	applyTiledClasses(&tmap, "assets/maps/forest.tmj")

	if name, _ := tmap.mapDisplayName(); name != "森" {
		t.Errorf("地名 = %q", name)
	}
	if key, _ := tmap.mapBGMKey(); key != "field2" {
		t.Errorf("曲 = %q", key)
	}
	if !tmap.mapAutoHeal() {
		t.Error("入ると全回復 が効いていません")
	}
	if !isLeverOpenLayer(tmap.Layers[0]) {
		t.Error("レバーで出るレイヤー が leveropen になっていません")
	}

	objs := tmap.Layers[1].Objects
	p := func(i int) map[string]string { return objProps(objs[i]) }
	if got := p(0); !isItemChestObj(got) || got["text"] != "event_chest_hi_potion" {
		t.Errorf("宝箱 → %v", got)
	}
	if got := p(1); got["targetmap"] != "assets/maps/water_b.tmj" || got["targetpoint"] != "梯子3" || got["requireboss"] != "" {
		t.Errorf("ドア → %v", got)
	}
	if got := p(2); !isLeverControlledWallObj(got) || !isLeverWallVisualOnly(got) || got["lever"] != "lever_a" {
		t.Errorf("見た目だけのレバーの壁 → %v", got)
	}
	if got := p(3); !isLeverControlledWallObj(got) || isLeverWallVisualOnly(got) {
		t.Errorf("レバーの壁 → %v", got)
	}
	if got := p(4); got["type"] != "enemy" || got["text"] != "スライム1,オーク1" || got["maxcount"] != "1" {
		t.Errorf("敵が出る場所 → %v", got)
	}
	if got := p(5); got["type"] != "event" || got["text"] != "こんにちは" {
		t.Errorf("会話 → %v", got)
	}
	if got := p(6); got["type"] != "event" || got["text"] != "" || got["objectiveid"] != "" {
		t.Errorf("空の会話 → %v(「調べるとなにかあるかもしれない」になるはず)", got)
	}
	if dir, ok := objPropInt(objs[7], "dir"); !ok || dir != 3 {
		t.Errorf("着地点の向き = %d, %v", dir, ok)
	}
	if id, ok := objPropInt(objs[8], "bossid"); !ok || id != 2 {
		t.Errorf("ボス番号 = %d, %v", id, ok)
	}
	if got := p(9); got["type"] != "event" || got["text"] != "従来の書き方" {
		t.Errorf("従来の書き方が変わりました → %v", got)
	}
}
