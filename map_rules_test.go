//go:build !js

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"testing"
)

// マップの書き間違いのチェック。ゲーム中はエラーにならず「何も起きない」
// ものを、ここで全部止める(docs/マップ作りガイド.md の内容)。
// ドア・着地点は TestMapWarps、レバーは TestMapLeverObjects が見る。

// ゲームが読む名札の名前。これ以外は書き間違いとして止める。
var (
	knownObjectProps = map[string]bool{
		"type": true, "text": true, "repeattext": true,
		"objectiveid": true, "objectiveorder": true, "bossid": true,
		"keys": true, "lever": true, "passable": true, "img": true,
		"id": true, "oneway": true, "spots": true,
		"route": true, "maxcount": true,
		"targetmap": true, "targetpoint": true, "requireboss": true, "dir": true,
	}
	knownMapProps      = map[string]bool{"bgm": true, "displayname": true, "autoheal": true, "battlebg": true}
	knownTileLayerProp = map[string]bool{"leveropen": true, "blocking": true, "playerlayer": true}
	knownObjectTypes   = map[string]bool{"": true, evTypeEvent: true, "trigger": true, "boss": true, "enemy": true, "darkness": true}
)

const maxBossID = len(Game{}.BossDefeatedFlags)

// tmjRaw は TiledMap に無い、チェックだけに使う項目。
type tmjRaw struct {
	Orientation string     `json:"orientation"`
	Infinite    bool       `json:"infinite"`
	TileWidth   int        `json:"tilewidth"`
	TileHeight  int        `json:"tileheight"`
	Layers      []rawLayer `json:"layers"`
}

// rawLayer は、ゲームが使わない(無視してしまう)レイヤーの設定を調べるためのもの。
type rawLayer struct {
	Name      string     `json:"name"`
	Class     string     `json:"class"`
	Type      string     `json:"type"`
	OffsetX   float64    `json:"offsetx"`
	OffsetY   float64    `json:"offsety"`
	ParallaxX *float64   `json:"parallaxx"`
	ParallaxY *float64   `json:"parallaxy"`
	TintColor string     `json:"tintcolor"`
	Layers    []rawLayer `json:"layers"`
	Objects   []struct {
		ID       int     `json:"id"`
		Template string  `json:"template"`
		GID      int     `json:"gid"`
		Rotation float64 `json:"rotation"`
		Ellipse  bool    `json:"ellipse"`
	} `json:"objects"`
}

// allRawLayers はグループの中も含めて全部のレイヤーを返す。
func allRawLayers(layers []rawLayer) []rawLayer {
	var out []rawLayer
	for _, l := range layers {
		out = append(out, l)
		out = append(out, allRawLayers(l.Layers)...)
	}
	return out
}

func allMapFiles(t *testing.T) []string {
	t.Helper()
	paths, err := fs.Glob(embeddedAssets, "assets/maps/*.tmj")
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func ensureStoryDialoguesLoaded() {
	if len(storyDialogues) == 0 {
		loadStoryDialogues("assets/dialogues")
	}
}

func TestMapSettings(t *testing.T) {
	for _, mapPath := range allMapFiles(t) {
		data, err := embeddedAssets.ReadFile(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		var raw tmjRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Errorf("%s: 読めません(タイルレイヤーの形式は CSV にしてください): %v", mapPath, err)
			continue
		}
		if raw.Orientation != "orthogonal" {
			t.Errorf("%s: マップの向きは「直交(orthogonal)」にしてください(今: %s)", mapPath, raw.Orientation)
		}
		if raw.Infinite {
			t.Errorf("%s: 無限マップになっています。マップのプロパティで「無限」のチェックを外してください", mapPath)
		}
		if raw.TileWidth != 32 || raw.TileHeight != 32 {
			t.Errorf("%s: タイルの大きさは 32×32 にしてください(今: %d×%d)", mapPath, raw.TileWidth, raw.TileHeight)
		}
		for _, l := range allRawLayers(raw.Layers) {
			if l.Type == "imagelayer" {
				t.Errorf("%s: 画像レイヤー%q はゲームに表示されません。絵はタイルセットにしてタイルレイヤーに描いてください", mapPath, l.Name)
			}
			if l.OffsetX != 0 || l.OffsetY != 0 {
				t.Errorf("%s: レイヤー%q がずらしてあります(オフセット)。ゲームではずれないので 0 に戻してください", mapPath, l.Name)
			}
			if (l.ParallaxX != nil && *l.ParallaxX != 1) || (l.ParallaxY != nil && *l.ParallaxY != 1) {
				t.Errorf("%s: レイヤー%q に視差(パララックス)が付いています。ゲームでは使えないので 1 に戻してください", mapPath, l.Name)
			}
			if l.TintColor != "" {
				t.Errorf("%s: レイヤー%q に色合い(ティント)が付いています。ゲームでは使えないので外してください", mapPath, l.Name)
			}
			for _, o := range l.Objects {
				if o.Rotation != 0 {
					t.Errorf("%s: レイヤー%q のオブジェクト(id%d)が回転しています。ゲームでは回転しないので 0 に戻してください", mapPath, l.Name, o.ID)
				}
				if o.Ellipse && l.Class == blockingLayerClass {
					t.Errorf("%s: 通れないレイヤー%q の楕円(id%d)は四角として扱われます。四角形か多角形で置いてください", mapPath, l.Name, o.ID)
				}
				if o.Template != "" {
					t.Errorf("%s: レイヤー%q のオブジェクト(id%d)がテンプレート(.tx)を使っています。テンプレートは使えません", mapPath, l.Name, o.ID)
				}
				if o.GID != 0 {
					t.Errorf("%s: レイヤー%q のオブジェクト(id%d)はタイルを置くオブジェクトです。ゲームには表示されません。絵はタイルレイヤーに描き、しかけは四角形で置いてください", mapPath, l.Name, o.ID)
				}
			}
		}

		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range tmap.Properties {
			name := strings.ToLower(p.Name)
			if !knownMapProps[name] {
				t.Errorf("%s: マップの名札 %q はゲームで使われません(綴りの間違い？ 使えるのは bgm / displayname / autoheal / battlebg)", mapPath, p.Name)
			}
		}
		if key, ok := tmap.mapBGMKey(); ok {
			if _, found := bgmByKey[key]; !found {
				t.Errorf("%s: bgm=%q という曲はありません", mapPath, key)
			}
		}
		if bg := mapPropString(tmap, "battlebg"); bg != "" {
			if _, err := fs.Stat(embeddedAssets, "assets/images/battle/bg/"+bg+".png"); err != nil {
				t.Errorf("%s: battlebg=%q の画像 assets/images/battle/bg/%s.png がありません", mapPath, bg, bg)
			}
		}
		for _, p := range tmap.Properties {
			if strings.EqualFold(p.Name, "autoheal") {
				if _, ok := propBool(p.Value); !ok {
					t.Errorf("%s: autoheal は true か false にしてください(今: %v)", mapPath, p.Value)
				}
			}
		}

		players := 0
		for _, layer := range tmap.Layers {
			// 前の決まり(レイヤーの名前で働きが決まる)のままのレイヤー。
			if want := oldLayerNameClass(layer); want != "" {
				t.Errorf("%s: レイヤー%q は名前だけでは働きません。クラスを「%s」にしてください", mapPath, layer.Name, want)
			}
			if isPlayerLayer(layer) {
				players++
			}
			switch {
			case layer.Type == "tilelayer":
				for _, p := range layer.Properties {
					if !knownTileLayerProp[strings.ToLower(p.Name)] {
						t.Errorf("%s: タイルレイヤー%q の名札 %q はゲームで使われません(使えるのは leveropen・blocking)", mapPath, layer.Name, p.Name)
					}
				}
			case layer.Type != "objectgroup":
			case isEventsLayer(layer), isBlockingLayer(layer):
			case isPlayerLayer(layer):
				if len(layer.Objects) > 0 {
					t.Errorf("%s: 「%s」のレイヤーにはオブジェクトを置かないでください(%d個あります。しかけは「%s」のレイヤーへ)", mapPath, playerLayerClass, len(layer.Objects), eventsLayerClass)
				}
			default:
				if len(layer.Objects) > 0 {
					t.Errorf("%s: オブジェクトレイヤー%q のオブジェクトは動きません。しかけはクラス「%s」の、通れない場所はクラス「%s」のレイヤーに置いてください", mapPath, layer.Name, eventsLayerClass, blockingLayerClass)
				}
			}
		}
		if players != 1 {
			t.Errorf("%s: クラス「%s」のレイヤーは1マップに1枚にしてください(今: %d枚)", mapPath, playerLayerClass, players)
		}
	}
}

// oldLayerNameClass は、前の決まりの名前(kabe・collision・events…・player)なのに
// クラスが付いていないレイヤーに、付けるべきクラスを返す。問題なければ空。
func oldLayerNameClass(layer TiledLayer) string {
	switch {
	case (layer.Name == "kabe" || layer.Name == "collision") && !isBlockingLayer(layer):
		return blockingLayerClass
	case strings.HasPrefix(layer.Name, "events") && layer.Type == "objectgroup" && !isEventsLayer(layer):
		return eventsLayerClass
	case layer.Name == "player" && !isPlayerLayer(layer):
		return playerLayerClass
	}
	return ""
}

func mapPropString(tmap TiledMap, name string) string {
	for _, p := range tmap.Properties {
		if strings.EqualFold(p.Name, name) {
			return fmt.Sprintf("%v", p.Value)
		}
	}
	return ""
}

func TestMapObjects(t *testing.T) {
	ensureStoryDialoguesLoaded()
	items := map[string]bool{}
	for _, it := range ItemDatabase {
		items[it.ID] = true
	}
	enemies := map[string]bool{}
	for _, e := range EnemyDatabase {
		enemies[e.Name] = true
	}

	// 鍵と目的地はマップをまたいで対応するので、先に全部集める。
	keyChests := map[string]bool{}
	objectiveIDs := map[string]string{}
	mapPaths := allMapFiles(t)
	maps := map[string]TiledMap{}
	for _, mapPath := range mapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		maps[mapPath] = tmap
		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			if isKeyChestObj(p) {
				keyChests[strings.TrimPrefix(p["text"], chestKeyTextPrefix)] = true
			}
		})
	}

	for _, mapPath := range mapPaths {
		tmap := maps[mapPath]
		errf := func(obj TiledObject, format string, args ...any) {
			t.Errorf("%s: オブジェクト(id%d): %s", mapPath, obj.ID, fmt.Sprintf(format, args...))
		}
		leverIDs := map[string]int{}
		blockIDs := map[string]bool{}
		spotIDs := map[string]bool{}
		var blockDoors []TiledObject

		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			for _, prop := range obj.Properties {
				if !knownObjectProps[strings.ToLower(prop.Name)] {
					errf(obj, "名札 %q はゲームで使われません(綴りの間違い？)", prop.Name)
				}
			}
			if !knownObjectTypes[p["type"]] {
				errf(obj, "type=%q という種類はありません(使えるのは event / trigger / boss / enemy / darkness)", p["type"])
			}
			for _, name := range []string{"bossid", "requireboss"} {
				if v, ok := p[name]; ok {
					if n, ok := propInt(v); !ok || n < 1 || n > maxBossID {
						errf(obj, "%s は 1〜%d の数字にしてください(今: %q)", name, maxBossID, v)
					}
				}
			}
			if v, ok := p["dir"]; ok {
				if n, ok := propInt(v); !ok || n < 0 || n > 3 {
					errf(obj, "dir は 0(下)・1(左)・2(右)・3(上) のどれかにしてください(今: %q)", v)
				}
			}
			if v, ok := p["objectiveorder"]; ok {
				if _, ok := propInt(v); !ok {
					errf(obj, "objectiveorder は数字にしてください(今: %q)", v)
				}
			}
			for _, name := range []string{"oneway", "passable"} {
				if v, ok := p[name]; ok {
					if _, ok := propBool(v); !ok {
						errf(obj, "%s は true か false にしてください(今: %q)", name, v)
					}
				}
			}
			if id := p["objectiveid"]; id != "" {
				if other, dup := objectiveIDs[id]; dup {
					errf(obj, "objectiveid=%q は %s でも使われています。目的地の名前は全部のマップで重ならないようにしてください", id, other)
				}
				objectiveIDs[id] = fmt.Sprintf("%s のid%d", mapPath, obj.ID)
			}

			text := p["text"]
			switch p["type"] {
			case evTypeEvent:
				switch {
				case isKeyChestObj(p):
					if strings.TrimPrefix(text, chestKeyTextPrefix) == "" {
						errf(obj, "鍵の宝箱に鍵の名前がありません(event_chest_key_鍵の名前)")
					}
				case isItemChestObj(p):
					if id := strings.TrimPrefix(text, chestTextPrefix); id == "" {
						errf(obj, "宝箱の中身が選ばれていません")
					} else if !items[id] {
						errf(obj, "宝箱のアイテム %q がありません(item.go の ItemDatabase の ID と同じにしてください)", id)
					}
				case strings.HasPrefix(text, storyTextPrefix):
					if id := strings.TrimPrefix(text, storyTextPrefix); storyDialogues[id].First.Commands == nil {
						errf(obj, "会話データ %q がありません(assets/dialogues/story の会話の名前と同じにしてください)", id)
					}
				case isLeverControlledWallObj(p):
					// レバーとのつながりは TestMapLeverObjects が見る。
				case isWallObj(p):
					for _, key := range splitKeyNames(p["keys"]) {
						if !keyChests[key] {
							errf(obj, "壁の鍵 %q が入った宝箱(event_chest_key_%s)がどのマップにもありません", key, key)
						}
					}
				case isLeverObj(p):
					if p["id"] == "" {
						errf(obj, "レバーに id がありません")
					}
					leverIDs[p["id"]]++
				case isBlockObj(p):
					if p["id"] == "" {
						errf(obj, "押すブロックに id がありません(id が無いとブロックが出ません)")
					}
					blockIDs[p["id"]] = true
				case isBlockSpotObj(p):
					if p["id"] == "" {
						errf(obj, "ブロックを乗せる場所に id がありません")
					}
					spotIDs[p["id"]] = true
				case isBlockDoorObj(p):
					blockDoors = append(blockDoors, obj)
				case strings.HasPrefix(text, "event_"):
					errf(obj, "text=%q は決まった言葉の書き間違いのようです(event_chest_ / event_wall / event_lever / event_block / event_blockspot / event_blockdoor / event_story_)", text)
				}
			case "trigger":
				if err := checkRoute(p["route"]); err != "" {
					errf(obj, "route=%q: %s", p["route"], err)
				}
				if strings.HasPrefix(text, storyTextPrefix) {
					if id := strings.TrimPrefix(text, storyTextPrefix); storyDialogues[id].First.Commands == nil {
						errf(obj, "会話データ %q がありません", id)
					}
				}
			case "boss":
				if _, ok := p["bossid"]; !ok {
					errf(obj, "ボスに bossid がありません(無いとボスが出ません)")
				}
			case "enemy":
				names := splitKeyNames(text)
				if len(names) == 0 {
					errf(obj, "敵が出る場所に敵の名前(text)がありません")
				}
				for _, name := range names {
					if !enemies[name] {
						errf(obj, "敵 %q がいません(tools/genstats/seed/enemies.csv の Name と同じにしてください)", name)
					}
				}
				if v, ok := p["maxcount"]; ok {
					if n, ok := propInt(v); !ok || n < 1 || n > 4 {
						errf(obj, "maxcount は 1〜4 にしてください(今: %q)", v)
					}
				}
			}
		})

		for id, n := range leverIDs {
			if n > 1 && id != "" {
				t.Errorf("%s: レバーの id %q が%d個あります。1つのマップの中では別々の名前にしてください", mapPath, id, n)
			}
		}
		for _, door := range blockDoors {
			spots := splitKeyNames(objProps(door)["spots"])
			if len(spots) == 0 {
				t.Errorf("%s: オブジェクト(id%d): ブロック扉に spots がありません(このままだと開きません)", mapPath, door.ID)
			}
			for _, s := range spots {
				if !spotIDs[s] {
					t.Errorf("%s: オブジェクト(id%d): ブロック扉の spots にある %q の、乗せる場所(event_blockspot, id=%s)がありません", mapPath, door.ID, s, s)
				}
			}
		}
		if len(spotIDs) > 0 && len(blockIDs) == 0 {
			t.Errorf("%s: ブロックを乗せる場所はあるのに、押すブロック(event_block)がありません", mapPath)
		}
	}
}

// 通れないレイヤー(四角・タイル)が、通れるレバー壁(机の橋など)をふさいでいないこと。
func TestCollisionDoesNotBlockLeverPaths(t *testing.T) {
	for _, mapPath := range allMapFiles(t) {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		var walls []TiledObject
		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			if isLeverControlledWallObj(p) && !isLeverWallVisualOnly(p) {
				walls = append(walls, obj)
			}
		})
		for _, layer := range tmap.Layers {
			if !isBlockingLayer(layer) {
				continue
			}
			for _, c := range layer.Objects {
				if c.Width <= 0 || c.Height <= 0 || len(c.Polygon) > 0 {
					continue
				}
				for _, w := range walls {
					if rectsOverlap(c.X, c.Y, c.X+c.Width, c.Y+c.Height, w.X, w.Y, w.X+w.Width, w.Y+w.Height) {
						t.Errorf("%s: 通れないレイヤー%q の四角(id%d)が、レバーで通れるようになる壁(id%d)に重なっています。レバーを上げても通れません", mapPath, layer.Name, c.ID, w.ID)
					}
				}
			}
			tw, th := float64(tmap.TileWidth), float64(tmap.TileHeight)
			for i, gid := range layer.Data {
				if gid == 0 || tmap.Width == 0 {
					continue
				}
				x, y := float64(i%tmap.Width)*tw, float64(i/tmap.Width)*th
				for _, w := range walls {
					if x < w.X+w.Width && x+tw > w.X && y < w.Y+w.Height && y+th > w.Y {
						t.Errorf("%s: 通れないレイヤー%q のタイル(%d列目・%d行目)が、レバーで通れるようになる壁(id%d)に重なっています。レバーを上げても通れません", mapPath, layer.Name, i%tmap.Width+1, i/tmap.Width+1, w.ID)
					}
				}
			}
		}
	}
}

func forEachEventObject(tmap TiledMap, fn func(TiledObject, map[string]string)) {
	for _, layer := range tmap.Layers {
		if !isEventsLayer(layer) {
			continue
		}
		for _, obj := range layer.Objects {
			fn(obj, objProps(obj))
		}
	}
}

// checkRoute はトリガーの route(例: up120,right64)の書き方を確かめる。
func checkRoute(route string) string {
	if route == "" {
		return ""
	}
	for step := range strings.SplitSeq(route, ",") {
		step = strings.TrimSpace(step)
		ok := false
		for _, dir := range []string{"down", "left", "right", "up"} {
			if dist, found := strings.CutPrefix(step, dir); found {
				if _, err := strconv.ParseFloat(dist, 64); err == nil {
					ok = true
				}
			}
		}
		if !ok {
			return fmt.Sprintf("%q は書き方が違います。up/down/left/right + 距離(例: up120)をカンマでつないでください", step)
		}
	}
	return ""
}

// 書き間違いを本当に見つけられるかを、わざと間違えたマップで確かめる。
func TestMapRuleHelpers(t *testing.T) {
	if checkRoute("up120,right64") != "" || checkRoute("") != "" {
		t.Error("正しい route を間違いと判定しました")
	}
	if checkRoute("upp120") == "" || checkRoute("up") == "" {
		t.Error("間違った route を見逃しました")
	}
	_ = path.Join
}
