package main

import "testing"

func TestMigrateSaveMapPathsRenamesKeys(t *testing.T) {
	d := &SaveData{
		CurrentMap:           "assets/maps/School_Map_1.tmj",
		OpenedChests:         map[string]bool{"assets/maps/woter_MAP_B.tmj|id3": true, "other|id1": true},
		SeenAutoHealMapIntro: map[string]bool{"assets/maps/School_Map_1.tmj": true},
		BlockPositions:       map[string][2]float64{"assets/maps/woter_MAP_B.tmj|block1": {1, 2}},
	}
	migrateSaveMapPaths(d)
	if d.CurrentMap != "assets/maps/school_1.tmj" || d.spawnPoint != "" {
		t.Errorf("CurrentMap = %q, spawnPoint = %q", d.CurrentMap, d.spawnPoint)
	}
	if !d.OpenedChests["assets/maps/water_b.tmj|id3"] || !d.OpenedChests["other|id1"] || len(d.OpenedChests) != 2 {
		t.Errorf("OpenedChests = %v", d.OpenedChests)
	}
	if !d.SeenAutoHealMapIntro["assets/maps/school_1.tmj"] {
		t.Errorf("SeenAutoHealMapIntro = %v", d.SeenAutoHealMapIntro)
	}
	if _, ok := d.BlockPositions["assets/maps/water_b.tmj|block1"]; !ok {
		t.Errorf("BlockPositions = %v", d.BlockPositions)
	}
}

func TestMigrateSaveMapPathsRemovedMapRestartsAtStart(t *testing.T) {
	d := &SaveData{CurrentMap: "assets/maps/ダンジョンA.tmj", PlayerX: 736, PlayerY: 3392}
	migrateSaveMapPaths(d)
	if d.CurrentMap != startMapPath || d.spawnPoint != "start_point" {
		t.Errorf("CurrentMap = %q, spawnPoint = %q", d.CurrentMap, d.spawnPoint)
	}
}

// 対応表の新しいパスと開始マップのstart_pointが実在すること。
func TestMigrateSaveMapTargetsExist(t *testing.T) {
	for oldPath, newPath := range renamedMapPaths {
		if _, err := loadTiledMap(newPath); err != nil {
			t.Errorf("%s → %s: %v", oldPath, newPath, err)
		}
	}
	tmap, err := loadTiledMap(startMapPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range tmap.Layers {
		for _, obj := range layer.Objects {
			if obj.Name == "start_point" {
				return
			}
		}
	}
	t.Errorf("%s に start_point がありません", startMapPath)
}
