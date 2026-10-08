package main

import (
	"flag"
	"strings"
)

// マップ作りの試し遊び用。Tiledで F5 を押すと tools/playmap/play.bat が
//
//	go run . -map <開いているマップのファイル>
//
// を実行し、タイトル画面を飛ばしてそのマップから始まる(ニューゲームの状態)。
//
//	-spawn <着地点の名前>  … 立つ場所(省略時は start_point、無ければ最初の
//	                          着地点、それも無ければマップの真ん中)
//	-noencounter          … 歩いていて敵に出会わない(F5。Shift+F5 は出会う)。
//	                          ボスやイベントの戦闘は起きる
var (
	devStartMap    = flag.String("map", "", "このマップから始める(試し遊び用)")
	devStartSpawn  = flag.String("spawn", "", "-map で始めるときの着地点の名前")
	devNoEncounter = flag.Bool("noencounter", false, "歩いていて敵に出会わない(試し遊び用)")
)

// parseDevFlags はコマンドラインを読み、-map をゲーム内のパス
// (assets/maps/○○.tmj)にそろえる。Tiledからはフルパスで渡ってくる。
func parseDevFlags() {
	if isWebBuild {
		return
	}
	flag.Parse()
	*devStartMap = normalizeDevMapPath(*devStartMap)
}

func normalizeDevMapPath(p string) string {
	p = strings.ReplaceAll(strings.Trim(p, `"`), `\`, "/")
	if i := strings.LastIndex(p, "assets/maps/"); i >= 0 {
		return p[i:]
	}
	if p != "" && !strings.Contains(p, "/") {
		return "assets/maps/" + p
	}
	return p
}

// devSpawn は -map で始めるときの立つ場所を決める。
func devSpawn(tmap TiledMap, requested string) (spawnName string, x, y float64) {
	if requested != "" {
		return requested, 0, 0
	}
	first := ""
	for _, layer := range tmap.Layers {
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			if obj.Name == "start_point" {
				return obj.Name, 0, 0
			}
			if first == "" && obj.Name != "" {
				if _, isDoor := objProps(obj)["targetmap"]; !isDoor {
					first = obj.Name
				}
			}
		}
	}
	if first != "" {
		return first, 0, 0
	}
	return "", float64(tmap.Width*tmap.TileWidth) / 2, float64(tmap.Height*tmap.TileHeight) / 2
}

// startDevMap はタイトル画面の代わりに -map のマップから始める。
func (g *Game) startDevMap() {
	mapPath := *devStartMap
	tmap, err := loadTiledMap(mapPath)
	if err != nil {
		loadTrace("試し遊び: マップを読めません %s: %v", mapPath, err)
		return
	}
	spawn, x, y := devSpawn(tmap, *devStartSpawn)
	loadTrace("試し遊び: %s の %q から始めます", mapPath, spawn)
	g.ChangeSceneAtLoadPoint(mapPath, func() Scene {
		g.ResetForNewGame()
		field, err := NewRoomScene(g, mapPath, x, y, spawn, 0)
		if err != nil {
			return nil
		}
		return field
	}, fadeTimeNewGame)
}
