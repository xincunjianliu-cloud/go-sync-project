//go:build !js

package main

import (
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
	"testing"
)

// 素材の登録のチェック。素材を追加・差し替えるたびにgo testで確かめられる
// ように、ファイル名ではなく「どこから読む決まりか」から確かめる。

// 段階読み込みの一覧の画像は、すべて存在し、二重に登録されていないこと。
func TestDeferredAssetsExistAndAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range deferredAssetAssignments() {
		if seen[a.path] {
			t.Errorf("二重に登録されています: %s", a.path)
		}
		seen[a.path] = true
		if _, err := embeddedAssets.ReadFile(a.path); err != nil && !a.optional {
			t.Errorf("ファイルがありません: %s (段階: %s)", a.path, a.tier)
		}
	}
}

// ボスの画像(ボスごとに読み込む)がすべて存在すること。
func TestBossImageAssetsExist(t *testing.T) {
	g := &Game{}
	for idx := range g.BossImgs {
		for _, a := range g.bossImageAssets(idx) {
			if _, err := embeddedAssets.ReadFile(a.path); err != nil {
				t.Errorf("ボス%dの画像がありません: %s", idx+1, a.path)
			}
		}
	}
}

// マップの"battlebg"とbossBattleBgで指定した戦闘背景が、すべて存在すること
// (無いとbattle_bg.pngで代わりに描かれ、指定の間違いに気づきにくい)。
func TestBattleBgsExist(t *testing.T) {
	if err := BuildObjectiveAndMapIndex(); err != nil {
		t.Fatal(err)
	}
	for _, mapPath := range allMapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Errorf("%s: %v", mapPath, err)
			continue
		}
		if key, ok := tmap.mapBattleBgKey(); ok {
			if _, err := embeddedAssets.ReadFile(battleBgImagePath(key)); err != nil {
				t.Errorf("%s の戦闘背景(battlebg=%q)がありません: %s", mapPath, key, battleBgImagePath(key))
			}
		}
	}
	for n, key := range bossBattleBg {
		if _, err := embeddedAssets.ReadFile(battleBgImagePath(key)); err != nil {
			t.Errorf("ボス%dの専用背景がありません: %s", n, battleBgImagePath(key))
		}
	}
}

// どのマップも、タイルセット画像が存在すること(無いとそのマップに入れない)。
func TestMapTilesetsExist(t *testing.T) {
	if err := BuildObjectiveAndMapIndex(); err != nil {
		t.Fatal(err)
	}
	for _, mapPath := range allMapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Errorf("%s: %v", mapPath, err)
			continue
		}
		p, ok := mapTilesetImagePath(tmap)
		if !ok {
			continue
		}
		if _, err := embeddedAssets.ReadFile(p); err != nil {
			t.Errorf("%s のタイルセット画像がありません: %s", mapPath, p)
		}
	}
}

// field_player.jsonのスプライトは、フィールド段階で先にデコードしておく画像と
// 同じであること(違うと、最初にマップへ入る瞬間に同期デコードで画面が止まる)。
func TestPlayerSpriteIsPreloaded(t *testing.T) {
	data, err := embeddedAssets.ReadFile(fieldPlayerConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Sprite string `json:"sprite"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, a := range deferredAssetAssignments() {
		if a.path == cfg.Sprite {
			if a.tier != assetTierField {
				t.Errorf("%s はフィールド段階で読み込むこと(今は%s段階)", cfg.Sprite, a.tier)
			}
			return
		}
	}
	t.Errorf("%s (field_player.jsonのsprite)が段階読み込みの一覧にありません", cfg.Sprite)
}

// どこからも読まれない画像を一覧する(失敗にはしない。go test -v で見える)。
// 公開フォルダに入るので、使わない画像はダウンロードの容量だけを増やす。
func TestListUnreferencedImages(t *testing.T) {
	if err := BuildObjectiveAndMapIndex(); err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{"assets/images/title/title_bg.png": true}
	for _, a := range deferredAssetAssignments() {
		used[a.path] = true
	}
	g := &Game{}
	for idx := range g.BossImgs {
		for _, a := range g.bossImageAssets(idx) {
			used[a.path] = true
		}
	}
	for _, mapPath := range allMapPaths {
		if tmap, err := loadTiledMap(mapPath); err == nil {
			if p, ok := mapTilesetImagePath(tmap); ok {
				used[p] = true
			}
			if key, ok := tmap.mapBattleBgKey(); ok {
				used[battleBgImagePath(key)] = true
			}
		}
	}
	for _, key := range bossBattleBg {
		used[battleBgImagePath(key)] = true
	}
	// 名前から実行中に決まるもの(立ち絵・会話の背景)。
	dynamic := func(p string) bool {
		return strings.HasPrefix(p, "assets/images/common/chara_") ||
			strings.HasPrefix(p, "assets/images/backgrounds/")
	}

	var unused []string
	err := fs.WalkDir(embeddedAssets, "assets/images", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".png") {
			return err
		}
		if !used[p] && !dynamic(p) {
			unused = append(unused, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(unused)
	for _, p := range unused {
		t.Logf("どこからも読まれていない画像: %s", p)
	}
}
