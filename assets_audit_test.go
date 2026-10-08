//go:build !js

package main

import (
	"bytes"
	"encoding/json"
	"image"
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

// マップのタイルセットの決まり(tileset.goの冒頭を参照)。
// マップ一覧に載っていない作りかけのマップも含め、assets/maps の全.tmjを見る。
func TestMapTilesets(t *testing.T) {
	mapPaths, err := fs.Glob(embeddedAssets, "assets/maps/*.tmj")
	if err != nil {
		t.Fatal(err)
	}
	for _, mapPath := range mapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Errorf("%s: %v", mapPath, err)
			continue
		}
		if len(tmap.tilesets) == 0 {
			t.Errorf("%s: タイルセットが設定されていません", mapPath)
		}
		for i, ts := range tmap.tilesets {
			if ts.source == "" {
				t.Errorf("%s: タイルセット%q がマップに埋め込まれています。Tiledでassets/tilesets/の外部タイルセット(.tsx)を使ってください", mapPath, ts.def.Name)
			} else if !strings.HasPrefix(ts.source, "assets/tilesets/") {
				t.Errorf("%s: タイルセットは assets/tilesets/ に置いてください: %s", mapPath, ts.source)
			}
			for _, other := range tmap.tilesets[i+1:] {
				if ts.firstGID < other.firstGID+other.def.TileCount && other.firstGID < ts.firstGID+ts.def.TileCount {
					t.Errorf("%s: タイルセット %s と %s の番号(firstgid)が重なっています", mapPath, ts.source, other.source)
				}
			}
		}
		for _, layer := range tmap.Layers {
			bad := map[int]int{}
			for _, id := range layer.Data {
				if gid := id &^ gidFlagMask; gid != 0 {
					if _, ok := findTileset(tmap.tilesets, gid); !ok {
						bad[gid]++
					}
				}
			}
			for gid, n := range bad {
				t.Errorf("%s: レイヤー%q のタイル番号%d(%d個)がどのタイルセットにもありません", mapPath, layer.Name, gid, n)
			}
		}
	}
}

// assets/tilesets の全タイルセットが、assets内の画像を指し、画像の実際の
// 大きさと.tsxの記述が一致していること。画像の横幅が変わるとタイル番号が
// ずれて全マップが崩れるので、Tiledで開き直して保存するまでここで止める。
func TestTilesetFiles(t *testing.T) {
	paths, err := fs.Glob(embeddedAssets, "assets/tilesets/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("assets/tilesets にタイルセットがありません")
	}
	for _, p := range paths {
		def, err := loadTilesetDef(p)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		// ゲームが使わない(黙って無視する)タイルセットの設定。
		if raw, err := embeddedAssets.ReadFile(p); err == nil {
			if bytes.Contains(raw, []byte("<objectgroup")) || bytes.Contains(raw, []byte(`"objectgroup"`)) {
				t.Errorf("%s: タイルの当たり判定エディタで形が描いてあります。ゲームでは使われないので、通れなくしたいタイルには wall=true を付けてください", p)
			}
			if bytes.Contains(raw, []byte("<tileoffset")) || bytes.Contains(raw, []byte(`"tileoffset"`)) {
				t.Errorf("%s: タイルのずらし(描画オフセット)が設定してあります。ゲームではずれないので 0 に戻してください", p)
			}
		}
		img := resolveRelativeAssetPath(p, def.Image)
		if !strings.HasPrefix(img, "assets/images/") {
			t.Errorf("%s: 画像が assets/images/ の外を指しています: %s (先に画像をassets/images/tiles/へコピーしてからTiledで指定してください)", p, def.Image)
			continue
		}
		data, err := embeddedAssets.ReadFile(img)
		if err != nil {
			t.Errorf("%s: 画像がありません: %s", p, img)
			continue
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Errorf("%s: %v", img, err)
			continue
		}
		if cfg.Width != def.ImageWidth || cfg.Height != def.ImageHeight {
			t.Errorf("%s: 画像 %s の大きさが %dx%d に変わっています(.tsxは%dx%d)。Tiledでタイルセットを開いて保存し直し、タイル番号がずれていないか確かめてください",
				p, img, cfg.Width, cfg.Height, def.ImageWidth, def.ImageHeight)
		}
		if def.TileWidth <= 0 || def.TileHeight <= 0 || def.Columns <= 0 {
			t.Errorf("%s: タイルの大きさ・列数が不正です", p)
			continue
		}
		rows := (def.ImageHeight - 2*def.Margin + def.Spacing) / (def.TileHeight + def.Spacing)
		cols := (def.ImageWidth - 2*def.Margin + def.Spacing) / (def.TileWidth + def.Spacing)
		if cols != def.Columns || def.TileCount > rows*cols {
			t.Errorf("%s: 列数%d・タイル数%dが画像(%d列x%d行)と合いません", p, def.Columns, def.TileCount, cols, rows)
		}
		for _, tile := range def.Tiles {
			for _, f := range tile.Animation {
				if f.TileID < 0 || f.TileID >= def.TileCount {
					t.Errorf("%s: タイル%d のアニメーションのコマ(タイル%d)がタイルセットの外です", p, tile.ID, f.TileID)
				}
				if f.Duration <= 0 {
					t.Errorf("%s: タイル%d のアニメーションのコマの長さが0です(1ミリ秒以上にしてください)", p, tile.ID)
				}
			}
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
			for _, p := range mapTilesetImagePaths(tmap) {
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

// ワープ(targetmapを持つオブジェクト)の行き先のマップがあり、targetpointと
// 同じ名前の着地点が行き先にあること。また、assets/maps のどのマップも開始
// マップからワープでたどれること(たどれないマップは目的地案内やドアの
// 経路探索の対象に入らない)。綴りミスはゲーム中ではエラーにならず
// 既定の座標に飛ぶだけなので、ここで止める。
func TestMapWarps(t *testing.T) {
	mapPaths, err := fs.Glob(embeddedAssets, "assets/maps/*.tmj")
	if err != nil {
		t.Fatal(err)
	}
	spawnNames := map[string]map[string]bool{}
	for _, mapPath := range mapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Fatalf("%s: %v", mapPath, err)
		}
		names := map[string]bool{}
		for _, layer := range tmap.Layers {
			if isEventsLayer(layer) {
				for _, obj := range layer.Objects {
					if obj.Name != "" {
						names[obj.Name] = true
					}
				}
			}
		}
		spawnNames[mapPath] = names
	}
	for _, mapPath := range mapPaths {
		tmap, _ := loadTiledMap(mapPath)
		for _, layer := range tmap.Layers {
			if !isEventsLayer(layer) {
				continue
			}
			for _, obj := range layer.Objects {
				p := objProps(obj)
				target := p["targetmap"]
				if target == "" {
					continue
				}
				names, ok := spawnNames[target]
				if !ok {
					t.Errorf("%s: ワープ(id%d)の行き先 %q のマップがありません", mapPath, obj.ID, target)
					continue
				}
				if point := p["targetpoint"]; point == "" {
					t.Errorf("%s: ワープ(id%d)に targetpoint がありません", mapPath, obj.ID)
				} else if !names[point] {
					t.Errorf("%s: ワープ(id%d)の着地点 %q が %s にありません(着地点オブジェクトの「名前」欄と一致させてください)", mapPath, obj.ID, point, target)
				}
			}
		}
	}

	if err := BuildObjectiveAndMapIndex(); err != nil {
		t.Fatal(err)
	}
	reachable := map[string]bool{}
	for _, p := range allMapPaths {
		reachable[p] = true
	}
	for _, mapPath := range mapPaths {
		if !reachable[mapPath] {
			t.Errorf("%s: 開始マップ(%s)からワープでたどれません。どこかのマップにこのマップへのワープを置いてください", mapPath, startMapPath)
		}
	}
}

// leverプロパティを持つオブジェクトは、壁(type=event, text=event_wall)か
// 暗闇(type=darkness)でないとゲームに無視される。参照するレバーも同じマップに
// あること。leveropenレイヤーのタイルはレバー壁の範囲に置くこと(範囲外は
// 一生表示されない)。どれもゲーム中はエラーにならず何も起きないだけなので、
// ここで止める。
func TestMapLeverObjects(t *testing.T) {
	mapPaths, err := fs.Glob(embeddedAssets, "assets/maps/*.tmj")
	if err != nil {
		t.Fatal(err)
	}
	for _, mapPath := range mapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Fatalf("%s: %v", mapPath, err)
		}
		levers := map[string]bool{}
		for _, layer := range tmap.Layers {
			if isEventsLayer(layer) {
				for _, obj := range layer.Objects {
					if p := objProps(obj); isLeverObj(p) {
						levers[p["id"]] = true
					}
				}
			}
		}
		for _, layer := range tmap.Layers {
			if !isEventsLayer(layer) {
				continue
			}
			for _, obj := range layer.Objects {
				p := objProps(obj)
				lever := p["lever"]
				if lever == "" {
					continue
				}
				if !isWallObj(p) && p["type"] != "darkness" {
					t.Errorf("%s: オブジェクト(id%d)に lever がありますが、ほかの種類(type/text)になっているため壁になりません。レバー壁なら type と text を消すか type=event, text=event_wall にしてください", mapPath, obj.ID)
				}
				if !levers[lever] {
					t.Errorf("%s: オブジェクト(id%d)の lever=%q のレバー(text=event_lever, id=%q)がこのマップにありません", mapPath, obj.ID, lever, lever)
				}
			}
		}
		for _, layer := range tmap.Layers {
			if layer.Type != "tilelayer" || !isLeverOpenLayer(layer) {
				continue
			}
			n := 0
			for i, id := range layer.Data {
				if id != 0 && (i >= len(tmap.leverCellLevers) || tmap.leverCellLevers[i] == "") {
					n++
				}
			}
			if n > 0 {
				t.Errorf("%s: leveropenレイヤー%q のタイル%d個がレバー壁の範囲の外にあり、表示されません", mapPath, layer.Name, n)
			}
		}
	}
}
