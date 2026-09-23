package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

var (
	tiledMapCache   = map[string]TiledMap{}
	tiledMapCacheMu sync.Mutex
)

// loadTiledMap は.tmjを読み込んでパースした結果をパスごとにキャッシュする。
// マップのJSONは数百KBあり、ドア移動・セーブ・ロード・戦闘からの復帰の
// たびに取得とjson.Unmarshalをやり直すと、その分だけ画面が固まる。
// 起動時のBuildObjectiveAndMapIndex(バックグラウンド)が到達可能な全マップを
// ここ経由で読むので、ゲーム中の呼び出しは通常キャッシュから即座に返る。
// 返り値のスライス類はキャッシュと共有しているため、呼び出し側で書き換えないこと。
func loadTiledMap(path string) (TiledMap, error) {
	tiledMapCacheMu.Lock()
	if m, ok := tiledMapCache[path]; ok {
		tiledMapCacheMu.Unlock()
		return m, nil
	}
	tiledMapCacheMu.Unlock()

	data, err := loadAssetBytesCached(path)
	if err != nil {
		return TiledMap{}, err
	}
	var tmap TiledMap
	if err := json.Unmarshal(data, &tmap); err != nil {
		return TiledMap{}, fmt.Errorf("マップ解析失敗 %s: %w", path, err)
	}

	tiledMapCacheMu.Lock()
	tiledMapCache[path] = tmap
	tiledMapCacheMu.Unlock()
	return tmap, nil
}

// mapTilesetImagePath は.tmjが指すタイルセット画像の実パスを返す。
func mapTilesetImagePath(tmap TiledMap) (string, bool) {
	if len(tmap.Tilesets) == 0 || tmap.Tilesets[0].Image == "" {
		return "", false
	}
	return resolveTilesetImagePath(tmap.Tilesets[0].Image), true
}
