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
	err = json.Unmarshal(data, &tmap)
	// パース結果はキャッシュするので、元のJSONのバイト列は手放す。
	assetStore.release(path)
	if err != nil {
		return TiledMap{}, fmt.Errorf("マップ解析失敗 %s: %w", path, err)
	}
	tmap.tilesets, err = resolveMapTilesets(path, tmap.Tilesets)
	if err != nil {
		return TiledMap{}, fmt.Errorf("マップのタイルセット読み込み失敗 %s: %w", path, err)
	}
	tmap.wallGIDs = tilesetWallGIDs(tmap.tilesets)

	tiledMapCacheMu.Lock()
	tiledMapCache[path] = tmap
	tiledMapCacheMu.Unlock()
	return tmap, nil
}

// peekTiledMap はパース済みのマップがキャッシュにあれば返す。loadTiledMapと
// 違って取得しに行かないので、メインゴルーチンから呼んでも待たされない。
func peekTiledMap(path string) (TiledMap, bool) {
	tiledMapCacheMu.Lock()
	defer tiledMapCacheMu.Unlock()
	m, ok := tiledMapCache[path]
	return m, ok
}
