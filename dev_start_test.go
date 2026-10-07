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
	school, err := loadTiledMap("assets/maps/school_1.tmj")
	if err != nil {
		t.Fatal(err)
	}
	if name, _, _ := devSpawn(school, ""); name != "start_point" {
		t.Errorf("school_1: %q から始まりました(start_point のはず)", name)
	}
	water, err := loadTiledMap("assets/maps/water_b.tmj")
	if err != nil {
		t.Fatal(err)
	}
	if name, _, _ := devSpawn(water, ""); name != "梯子3" {
		t.Errorf("water_b: %q から始まりました(梯子3 のはず)", name)
	}
	if name, _, _ := devSpawn(water, "梯子3"); name != "梯子3" {
		t.Errorf("-spawn が効いていません: %q", name)
	}
}
