package main

import "strings"

// マップの.tmjを改名・削除したときに、古いセーブを読めるようにする対応表。
// セーブにはマップのパスがそのまま入る(現在地、宝箱やレバーのキー
// "パス|..."など)ので、改名したら旧→新をここに足す。
var renamedMapPaths = map[string]string{
	"assets/maps/School_Map_1.tmj": "assets/maps/school_1.tmj",
	"assets/maps/woter_MAP_B.tmj":  "assets/maps/water_b.tmj",
}

// removedMapPaths は削除したマップ。ここにいたセーブは開始マップの
// start_point から再開する。
var removedMapPaths = map[string]bool{
	"assets/maps/ダンジョンA.tmj": true,
}

// migrateSaveMapPaths は古いマップのパスを今のパスに置き換える。
func migrateSaveMapPaths(d *SaveData) {
	if newPath, ok := renamedMapPaths[d.CurrentMap]; ok {
		d.CurrentMap = newPath
	}
	if removedMapPaths[d.CurrentMap] {
		d.CurrentMap = startMapPath
		d.spawnPoint = "start_point"
	}
	d.OpenedChests = renameMapKeys(d.OpenedChests)
	d.UnlockedWalls = renameMapKeys(d.UnlockedWalls)
	d.RaisedLevers = renameMapKeys(d.RaisedLevers)
	d.SeenAutoHealMapIntro = renameMapKeys(d.SeenAutoHealMapIntro)
	d.SeenEvents = renameMapKeys(d.SeenEvents)
	d.UnlockedBlockDoors = renameMapKeys(d.UnlockedBlockDoors)
	d.BlockPositions = renameMapKeys(d.BlockPositions)
}

// renameMapKeys は「マップのパス」または「マップのパス|...」の形のキーを
// 新しいパスに置き換える。
func renameMapKeys[V any](m map[string]V) map[string]V {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]V, len(m))
	for k, v := range m {
		mapPath, _, _ := strings.Cut(k, "|")
		if newPath, ok := renamedMapPaths[mapPath]; ok {
			k = newPath + k[len(mapPath):]
		}
		out[k] = v
	}
	return out
}
