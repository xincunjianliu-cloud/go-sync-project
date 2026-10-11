package main

import "testing"

func TestNormalizeDevMapPath(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Users\a\OneDrive\デスクトップ\rpg\assets\maps\water_b.tmj`: "assets/maps/water_b.tmj",
		`"D:/work/go-sync-project/assets/maps/school_1.tmj"`:     "assets/maps/school_1.tmj",
		"water_b.tmj":             "assets/maps/water_b.tmj",
		"assets/maps/water_b.tmj": "assets/maps/water_b.tmj",
		"":                        "",
	} {
		if got := normalizeDevMapPath(in); got != want {
			t.Errorf("normalizeDevMapPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDevSpawnPicksStartPointThenFirstSpawn(t *testing.T) {
	events := TiledLayer{
		Properties: []TiledProperty{{Name: "eventlayer", Value: true}},
		Objects: []TiledObject{
			{Name: "door", Properties: []TiledProperty{{Name: "targetmap", Value: "other.tmj"}}},
			{Name: "landing_a"},
			{Name: "start_point"},
			{Name: "landing_b"},
		},
	}
	tmap := TiledMap{Width: 20, Height: 10, TileWidth: 32, TileHeight: 32, Layers: []TiledLayer{events}}

	if name, _, _ := devSpawn(tmap, ""); name != "start_point" {
		t.Errorf("start_point がある場合はそこから始まるはず: %q", name)
	}
	if name, _, _ := devSpawn(tmap, "指定した場所"); name != "指定した場所" {
		t.Errorf("-spawn が効いていません: %q", name)
	}

	tmap.Layers[0].Objects = []TiledObject{
		{Name: "door", Properties: []TiledProperty{{Name: "targetmap", Value: "other.tmj"}}},
		{Name: "landing_a"},
		{Name: "landing_b"},
	}
	if name, _, _ := devSpawn(tmap, ""); name != "landing_a" {
		t.Errorf("start_point がない場合は最初の着地点から始まるはず: %q", name)
	}
}

func TestDevSpawnFallsBackToMapCenter(t *testing.T) {
	tmap := TiledMap{Width: 20, Height: 10, TileWidth: 32, TileHeight: 32}
	name, x, y := devSpawn(tmap, "")
	if name != "" || x != 320 || y != 160 {
		t.Errorf("着地点がない場合はマップ中央から始まるはず: %q (%v, %v)", name, x, y)
	}
}
