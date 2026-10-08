//go:build !js

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io/fs"
	"sort"
	"strings"
)

// マップの書き間違いのチェック。ゲーム中はエラーにならず「何も起きない」
// ものを、ここで全部見つける(docs/マップ作りガイド.md の内容)。
//
// 同じチェックを2か所で使う。
//   - go test(GitHub が Push のたびに動かす。まちがいがあれば公開しない)
//   - Tiled の F5(試し遊び)。始める前にまちがいを画面に出す(map_check_scene.go)

type mapChecker struct {
	issues []mapIssue
}

func (c *mapChecker) add(path, format string, args ...any) {
	c.issues = append(c.issues, mapIssue{Path: path, Msg: fmt.Sprintf(format, args...)})
}

// mapCheckFuncs は全部のチェック。名前はテストの名前と同じ。
var mapCheckFuncs = []struct {
	Name string
	Fn   func(*mapChecker)
}{
	{"TestMapSettings", checkMapSettings},
	{"TestMapObjects", checkMapObjects},
	{"TestCollisionDoesNotBlockLeverPaths", checkCollisionDoesNotBlockLeverPaths},
	{"TestLeverWallsHaveOpenTiles", checkLeverWallsHaveOpenTiles},
	{"TestMapTilesets", checkMapTilesets},
	{"TestTilesetFiles", checkTilesetFiles},
	{"TestMapWarps", checkMapWarps},
	{"TestMapLeverObjects", checkMapLeverObjects},
	{"TestStartMapUpToDate", checkStartMap},
}

// checkAllMaps は全部のチェックをして、見つかったまちがいを返す。
func checkAllMaps() []mapIssue {
	c := &mapChecker{}
	for _, f := range mapCheckFuncs {
		f.Fn(c)
	}
	return c.issues
}

// devMapIssues は試し遊び(-map)を始める前のチェック。
func devMapIssues() []mapIssue { return checkAllMaps() }

// ゲームが読む名札の名前。これ以外は書き間違いとして止める。
var (
	knownObjectProps = map[string]bool{
		"type": true, "text": true, "repeattext": true,
		"objectiveid": true, "objectiveorder": true, "bossid": true,
		"keys": true, "lever": true, "passable": true,
		"id": true, "oneway": true, "spots": true,
		"route": true, "maxcount": true,
		"targetmap": true, "targetpoint": true, "requireboss": true, "dir": true,
	}
	knownMapProps      = map[string]bool{"bgm": true, "displayname": true, "autoheal": true, "battlebg": true, "startmap": true}
	knownTileLayerProp = map[string]bool{"leveropen": true, "blocking": true, "playerlayer": true}
	knownObjectTypes   = map[string]bool{"": true, evTypeEvent: true, "trigger": true, "boss": true, "enemy": true, "darkness": true}

	// objFieldLabel は読み替えたあとの名札を、Tiledに出る欄の名前に戻す(エラー文用)。
	objFieldLabel = map[string]string{
		"bossid": "ボス番号・ボス戦", "requireboss": "倒すまで通れないボス",
		"oneway": "一度きり", "passable": "見た目だけ",
	}

	// areaObjectClasses は、四角の中に入ったときに働くしかけ。大きさが0だと働かない。
	areaObjectClasses = map[string]bool{
		"ドア": true, "会話": true, "イベント": true, "敵が出る場所": true,
		"暗い場所": true, "レバーの壁": true, "鍵の壁": true,
	}
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

func mapFilePaths() []string {
	paths, _ := fs.Glob(embeddedAssets, "assets/maps/*.tmj")
	return paths
}

// loadMapsForCheck は全マップを読む。読めないマップはまちがいとして出し、外す。
func loadMapsForCheck(c *mapChecker) ([]string, map[string]TiledMap) {
	var paths []string
	maps := map[string]TiledMap{}
	for _, mapPath := range mapFilePaths() {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			c.add(mapPath, "読めません: %v", err)
			continue
		}
		paths = append(paths, mapPath)
		maps[mapPath] = tmap
	}
	return paths, maps
}

func checkMapSettings(c *mapChecker) {
	for _, mapPath := range mapFilePaths() {
		data, err := embeddedAssets.ReadFile(mapPath)
		if err != nil {
			c.add(mapPath, "読めません: %v", err)
			continue
		}
		var raw tmjRaw
		if err := json.Unmarshal(data, &raw); err != nil {
			c.add(mapPath, "読めません(タイルレイヤーの形式は CSV にしてください): %v", err)
			continue
		}
		if raw.Orientation != "orthogonal" {
			c.add(mapPath, "マップの向きは「直交(orthogonal)」にしてください(今: %s)", raw.Orientation)
		}
		if raw.Infinite {
			c.add(mapPath, "無限マップになっています。マップのプロパティで「無限」のチェックを外してください")
		}
		if raw.TileWidth != 32 || raw.TileHeight != 32 {
			c.add(mapPath, "タイルの大きさは 32×32 にしてください(今: %d×%d)", raw.TileWidth, raw.TileHeight)
		}
		for _, l := range allRawLayers(raw.Layers) {
			if l.Type == "imagelayer" {
				c.add(mapPath, "画像レイヤー%q はゲームに表示されません。絵はタイルセットにしてタイルレイヤーに描いてください", l.Name)
			}
			if l.OffsetX != 0 || l.OffsetY != 0 {
				c.add(mapPath, "レイヤー%q がずらしてあります(オフセット)。ゲームではずれないので 0 に戻してください", l.Name)
			}
			if (l.ParallaxX != nil && *l.ParallaxX != 1) || (l.ParallaxY != nil && *l.ParallaxY != 1) {
				c.add(mapPath, "レイヤー%q に視差(パララックス)が付いています。ゲームでは使えないので 1 に戻してください", l.Name)
			}
			if l.TintColor != "" {
				c.add(mapPath, "レイヤー%q に色合い(ティント)が付いています。ゲームでは使えないので外してください", l.Name)
			}
			for _, o := range l.Objects {
				if o.Rotation != 0 {
					c.add(mapPath, "レイヤー%q のオブジェクト(id%d)が回転しています。ゲームでは回転しないので 0 に戻してください", l.Name, o.ID)
				}
				if o.Ellipse && l.Class == blockingLayerClass {
					c.add(mapPath, "通れないレイヤー%q の楕円(id%d)は四角として扱われます。四角形か多角形で置いてください", l.Name, o.ID)
				}
				if o.Template != "" {
					c.add(mapPath, "レイヤー%q のオブジェクト(id%d)がテンプレート(.tx)を使っています。テンプレートは使えません", l.Name, o.ID)
				}
				if o.GID != 0 {
					c.add(mapPath, "レイヤー%q のオブジェクト(id%d)はタイルを置くオブジェクトです。ゲームには表示されません。絵はタイルレイヤーに描き、しかけは四角形で置いてください", l.Name, o.ID)
				}
			}
		}

		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			c.add(mapPath, "読めません: %v", err)
			continue
		}
		for _, p := range tmap.Properties {
			name := strings.ToLower(p.Name)
			if !knownMapProps[name] {
				c.add(mapPath, "マップに、ゲームで使われないプロパティ %q があります。マップ設定の欄(地名・曲・戦闘背景・入ると全回復・ゲームの最初のマップ)だけを使ってください", p.Name)
			}
		}
		if key, ok := tmap.mapBGMKey(); ok {
			if _, found := bgmByKey[key]; !found {
				c.add(mapPath, "マップ設定の「曲」%q という曲はありません。一覧から選び直してください", key)
			}
		}
		if bg := mapPropString(tmap, "battlebg"); bg != "" {
			if _, err := fs.Stat(embeddedAssets, "assets/images/battle/bg/"+bg+".png"); err != nil {
				c.add(mapPath, "マップ設定の「戦闘背景」%q の画像 assets/images/battle/bg/%s.png がありません", bg, bg)
			}
		}
		for _, p := range tmap.Properties {
			if strings.EqualFold(p.Name, "autoheal") {
				if _, ok := propBool(p.Value); !ok {
					c.add(mapPath, "マップ設定の「入ると全回復」はチェックで選んでください(今: %v)", p.Value)
				}
			}
		}

		players := 0
		spawnNames := map[string]int{}
		for _, layer := range tmap.Layers {
			// 前の決まり(レイヤーの名前で働きが決まる)のままのレイヤー。
			if want := oldLayerNameClass(layer); want != "" {
				c.add(mapPath, "レイヤー%q は名前だけでは働きません。クラスを「%s」にしてください", layer.Name, want)
			}
			if isPlayerLayer(layer) {
				players++
			}
			for _, o := range layer.Objects {
				switch {
				case isEventsLayer(layer) && o.Class == "":
					c.add(mapPath, "しかけレイヤー%q の四角(id%d)にクラスが選ばれていません(このままでは何も起きません)。クラスを選ぶか、要らなければ消してください", layer.Name, o.ID)
				case isEventsLayer(layer) && o.Class == "着地点":
					if o.Name == "" {
						c.add(mapPath, "着地点(id%d)の「名前(Name)」が空です。ドアの「着地点」と同じ名前を書いてください", o.ID)
					}
					spawnNames[o.Name]++
				case isBlockingLayer(layer) && o.Class != "":
					c.add(mapPath, "しかけ「%s」(id%d)が通れないレイヤー%q に置いてあります。しかけはしかけレイヤーに置いてください", o.Class, o.ID, layer.Name)
				}
				if len(o.Polygon) == 0 && (o.Width <= 0 || o.Height <= 0) {
					switch {
					case isBlockingLayer(layer):
						c.add(mapPath, "通れないレイヤー%q の四角(id%d)の幅か高さが 0 です(何もふさいでいません)。大きさを直すか、消してください", layer.Name, o.ID)
					case isEventsLayer(layer) && areaObjectClasses[o.Class]:
						c.add(mapPath, "「%s」(id%d)の幅か高さが 0 です(このままでは働きません)。大きさを直してください", o.Class, o.ID)
					}
				}
			}
			if isEventsLayer(layer) {
				checkDuplicateObjects(c, mapPath, layer)
			}
			switch {
			case layer.Type == "tilelayer":
				for _, p := range layer.Properties {
					if !knownTileLayerProp[strings.ToLower(p.Name)] {
						c.add(mapPath, "タイルレイヤー%q に、ゲームで使われないプロパティ %q があります。要らなければ消してください(働きはクラスで選ぶ)", layer.Name, p.Name)
					}
				}
			case layer.Type != "objectgroup":
			case isEventsLayer(layer), isBlockingLayer(layer):
			case isPlayerLayer(layer):
				if len(layer.Objects) > 0 {
					c.add(mapPath, "「%s」のレイヤーにはオブジェクトを置かないでください(%d個あります。しかけは「%s」のレイヤーへ)", playerLayerClass, len(layer.Objects), eventsLayerClass)
				}
			default:
				if len(layer.Objects) > 0 {
					c.add(mapPath, "オブジェクトレイヤー%q のオブジェクトは動きません。しかけはクラス「%s」の、通れない場所はクラス「%s」のレイヤーに置いてください", layer.Name, eventsLayerClass, blockingLayerClass)
				}
			}
		}
		for name, n := range spawnNames {
			if name != "" && n > 1 {
				c.add(mapPath, "着地点の名前 %q が%d個あります。1つのマップの中では別々の名前にしてください(ドアがどちらに着くか分からなくなる)", name, n)
			}
		}
		if players != 1 {
			c.add(mapPath, "クラス「%s」のレイヤーは1マップに1枚にしてください(今: %d枚)", playerLayerClass, players)
		}
	}
}

// checkDuplicateObjects は、同じ場所・同じ大きさ・同じ中身のしかけが2つあるのを
// 見つける(コピーして貼ったまま動かし忘れたもの)。
func checkDuplicateObjects(c *mapChecker, mapPath string, layer TiledLayer) {
	seen := map[string]int{}
	for _, o := range layer.Objects {
		props, _ := json.Marshal(o.Properties)
		key := fmt.Sprintf("%s|%s|%v,%v,%v,%v|%s", o.Class, o.Name, o.X, o.Y, o.Width, o.Height, props)
		if first, dup := seen[key]; dup {
			c.add(mapPath, "「%s」(id%d)と(id%d)が、同じ場所に同じ中身で重なっています。片方を消してください", o.Class, first, o.ID)
			continue
		}
		seen[key] = o.ID
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

func checkMapObjects(c *mapChecker) {
	// ゲーム本体の一覧(storyDialogues)は変えずに、ここだけで読む。
	stories := readStoryDialogues("assets/dialogues")
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
	mapPaths, maps := loadMapsForCheck(c)
	for _, mapPath := range mapPaths {
		forEachEventObject(maps[mapPath], func(obj TiledObject, p map[string]string) {
			if isKeyChestObj(p) {
				keyChests[strings.TrimPrefix(p["text"], chestKeyTextPrefix)] = true
			}
		})
	}

	// レバーの上げ下げは名前だけで覚えている(マップをまたいで共通)ので、
	// 別のマップでも同じ名前を使うと、片方を上げるともう片方も上がる。
	leverMaps := map[string][]string{}

	for _, mapPath := range mapPaths {
		tmap := maps[mapPath]
		errf := func(obj TiledObject, format string, args ...any) {
			c.add(mapPath, "オブジェクト(id%d): %s", obj.ID, fmt.Sprintf(format, args...))
		}
		leverIDs := map[string]int{}
		blockIDs := map[string]bool{}
		spotIDs := map[string]bool{}
		var blockDoors []TiledObject

		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			for _, prop := range obj.Properties {
				if !knownObjectProps[strings.ToLower(prop.Name)] {
					errf(obj, "ゲームで使われないプロパティ %q があります。要らなければ消してください", prop.Name)
				}
			}
			if !knownObjectTypes[p["type"]] {
				errf(obj, "種類 %q というしかけはありません。クラスを一覧から選び直してください", p["type"])
			}
			for _, name := range []string{"bossid", "requireboss"} {
				if v, ok := p[name]; ok {
					if n, ok := propInt(v); !ok || n < 1 || n > maxBossID {
						errf(obj, "「%s」は 1〜%d から選んでください(今: %q)", objFieldLabel[name], maxBossID, v)
					}
				}
			}
			if v, ok := p["dir"]; ok {
				if n, ok := propInt(v); !ok || n < 0 || n > 3 {
					errf(obj, "着地点の「向き」を一覧から選び直してください(今: %q)", v)
				}
			}
			if v, ok := p["objectiveorder"]; ok {
				if _, ok := propInt(v); !ok {
					errf(obj, "「目的地の順番」は数字にしてください(今: %q)", v)
				}
			}
			for _, name := range []string{"oneway", "passable"} {
				if v, ok := p[name]; ok {
					if _, ok := propBool(v); !ok {
						errf(obj, "「%s」はチェックで選んでください(今: %q)", objFieldLabel[name], v)
					}
				}
			}
			if id := p["objectiveid"]; id != "" {
				if other, dup := objectiveIDs[id]; dup {
					errf(obj, "目的地の名前 %q は %s でも使われています。目的地の名前は全部のマップで重ならないようにしてください", id, other)
				}
				objectiveIDs[id] = fmt.Sprintf("%s のid%d", mapPath, obj.ID)
			}

			text := p["text"]
			switch p["type"] {
			case evTypeEvent:
				switch {
				case isKeyChestObj(p):
					if strings.TrimPrefix(text, chestKeyTextPrefix) == "" {
						errf(obj, "「鍵の宝箱」の「鍵の名前」が空です")
					}
				case isItemChestObj(p):
					if id := strings.TrimPrefix(text, chestTextPrefix); id == "" {
						errf(obj, "「宝箱」の「中身」が選ばれていません")
					} else if !items[id] {
						errf(obj, "「宝箱」の中身 %q というアイテムはありません。一覧から選び直してください", id)
					}
				case strings.HasPrefix(text, storyTextPrefix):
					if id := strings.TrimPrefix(text, storyTextPrefix); stories[id].First.Commands == nil {
						errf(obj, "「会話データ」%q という会話がありません。go run ./tools/update を打ってから一覧で選び直してください", id)
					}
				case isLeverControlledWallObj(p):
					// レバーとのつながりは checkMapLeverObjects が見る。
				case isWallObj(p):
					for _, key := range splitKeyNames(p["keys"]) {
						if !keyChests[key] {
							errf(obj, "「鍵の壁」の必要な鍵 %q が入った「鍵の宝箱」が、どのマップにもありません", key)
						}
					}
				case isLeverObj(p):
					if p["id"] == "" {
						errf(obj, "「レバー」の「レバーの名前」が空です")
					}
					leverIDs[p["id"]]++
				case isBlockObj(p):
					if p["id"] == "" {
						errf(obj, "「ブロック」の「ブロックの名前」が空です(空だとブロックが出ません)")
					}
					blockIDs[p["id"]] = true
				case isBlockSpotObj(p):
					if p["id"] == "" {
						errf(obj, "「ブロック置き場」の「置き場の名前」が空です")
					}
					spotIDs[p["id"]] = true
				case isBlockDoorObj(p):
					blockDoors = append(blockDoors, obj)
				case strings.HasPrefix(text, "event_"):
					errf(obj, "セリフ %q は、ゲームの決まった言葉(event_〜)の書き間違いのようです。ふつうのセリフなら event_ で始めないでください", text)
				}
			case "trigger":
				if _, err := parseRoute(p["route"]); err != "" {
					errf(obj, "「歩く道順」%q: %s", p["route"], err)
				}
				if strings.HasPrefix(text, storyTextPrefix) {
					if id := strings.TrimPrefix(text, storyTextPrefix); stories[id].First.Commands == nil {
						errf(obj, "「会話データ」%q という会話がありません。go run ./tools/update を打ってから一覧で選び直してください", id)
					}
				}
			case "boss":
				if _, ok := p["bossid"]; !ok {
					errf(obj, "「ボス」の「ボス番号」が選ばれていません(このままではボスが出ません)")
				}
			case "enemy":
				names := splitKeyNames(text)
				if len(names) == 0 {
					errf(obj, "「敵が出る場所」の「出る敵」が選ばれていません")
				}
				for _, name := range names {
					if !enemies[name] {
						errf(obj, "「出る敵」の %q という敵がいません。go run ./tools/update を打ってから一覧で選び直してください", name)
					}
				}
				if v, ok := p["maxcount"]; ok {
					if n, ok := propInt(v); !ok || n < 1 || n > 4 {
						errf(obj, "「一度に出る最大数」は 1〜4 にしてください(今: %q)", v)
					}
				}
			}
		})

		for id, n := range leverIDs {
			if id == "" {
				continue
			}
			if n > 1 {
				c.add(mapPath, "「レバーの名前」%q のレバーが%d個あります。別々の名前にしてください", id, n)
			}
			leverMaps[id] = append(leverMaps[id], mapPath)
		}
		for _, door := range blockDoors {
			spots := splitKeyNames(objProps(door)["spots"])
			if len(spots) == 0 {
				c.add(mapPath, "オブジェクト(id%d): 「ブロック扉」の「置き場の名前」が空です(このままだと開きません)", door.ID)
			}
			for _, s := range spots {
				if !spotIDs[s] {
					c.add(mapPath, "オブジェクト(id%d): 「ブロック扉」の置き場の名前 %q の「ブロック置き場」がありません", door.ID, s)
				}
			}
		}
		if len(spotIDs) > 0 && len(blockIDs) == 0 {
			c.add(mapPath, "「ブロック置き場」はあるのに、押す「ブロック」がありません")
		}
	}

	ids := make([]string, 0, len(leverMaps))
	for id := range leverMaps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if ms := leverMaps[id]; len(ms) > 1 {
			for _, m := range ms {
				c.add(m, "「レバーの名前」%q が、ほかのマップでも使われています(%s)。片方を上げるともう片方も上がってしまうので、全部のマップで別々の名前にしてください", id, strings.Join(ms, "・"))
			}
		}
	}
}

// 通れないレイヤー(四角・タイル)が、通れるレバー壁(机の橋など)をふさいでいないこと。
func checkCollisionDoesNotBlockLeverPaths(c *mapChecker) {
	mapPaths, maps := loadMapsForCheck(c)
	for _, mapPath := range mapPaths {
		tmap := maps[mapPath]
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
			for _, o := range layer.Objects {
				if o.Width <= 0 || o.Height <= 0 || len(o.Polygon) > 0 {
					continue
				}
				for _, w := range walls {
					if rectsOverlap(o.X, o.Y, o.X+o.Width, o.Y+o.Height, w.X, w.Y, w.X+w.Width, w.Y+w.Height) {
						c.add(mapPath, "通れないレイヤー%q の四角(id%d)が、レバーで通れるようになる「レバーの壁」(id%d)に重なっています。レバーを上げても通れません", layer.Name, o.ID, w.ID)
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
						c.add(mapPath, "通れないレイヤー%q のタイル(%d列目・%d行目)が、レバーで通れるようになる「レバーの壁」(id%d)に重なっています。レバーを上げても通れません", layer.Name, i%tmap.Width+1, i/tmap.Width+1, w.ID)
					}
				}
			}
		}
	}
}

// レバーの壁が開いたときの見た目が「レバーで出るレイヤー」に描いてあること。
// 描いていないと、レバーを上げても見た目が変わらない(既定の画像は無い)。
func checkLeverWallsHaveOpenTiles(c *mapChecker) {
	mapPaths, maps := loadMapsForCheck(c)
	for _, mapPath := range mapPaths {
		tmap := maps[mapPath]
		var leverLayers []TiledLayer
		for _, l := range tmap.Layers {
			if l.Type == "tilelayer" && isLeverOpenLayer(l) {
				leverLayers = append(leverLayers, l)
			}
		}
		tw, th := tmap.TileWidth, tmap.TileHeight
		if tw == 0 || th == 0 {
			continue
		}
		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			if !isLeverControlledWallObj(p) {
				return
			}
			missing := 0
			for cy := int(obj.Y) / th; cy*th < int(obj.Y+obj.Height); cy++ {
				for cx := int(obj.X) / tw; cx*tw < int(obj.X+obj.Width); cx++ {
					i := cy*tmap.Width + cx
					drawn := false
					for _, l := range leverLayers {
						if i >= 0 && i < len(l.Data) && l.Data[i] != 0 {
							drawn = true
							break
						}
					}
					if !drawn {
						missing++
					}
				}
			}
			if missing > 0 {
				c.add(mapPath, "レバーの壁(id%d, レバー %q)の開いたときの絵が %dマス足りません。クラス「レバーで出るレイヤー」のレイヤーに、開いたときの絵を描いてください", obj.ID, p["lever"], missing)
			}
		})
	}
}

// マップのタイルセットの決まり(tileset.goの冒頭を参照)。
// マップ一覧に載っていない作りかけのマップも含め、assets/maps の全.tmjを見る。
func checkMapTilesets(c *mapChecker) {
	mapPaths, maps := loadMapsForCheck(c)
	for _, mapPath := range mapPaths {
		tmap := maps[mapPath]
		if len(tmap.tilesets) == 0 {
			c.add(mapPath, "タイルセットが設定されていません")
		}
		for i, ts := range tmap.tilesets {
			if ts.source == "" {
				c.add(mapPath, "タイルセット%q がマップに埋め込まれています。Tiledでassets/tilesets/の外部タイルセット(.tsx)を使ってください", ts.def.Name)
			} else if !strings.HasPrefix(ts.source, "assets/tilesets/") {
				c.add(mapPath, "タイルセットは assets/tilesets/ に置いてください: %s", ts.source)
			}
			for _, other := range tmap.tilesets[i+1:] {
				if ts.firstGID < other.firstGID+other.def.TileCount && other.firstGID < ts.firstGID+ts.def.TileCount {
					c.add(mapPath, "タイルセット %s と %s の番号(firstgid)が重なっています", ts.source, other.source)
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
				c.add(mapPath, "レイヤー%q のタイル番号%d(%d個)がどのタイルセットにもありません", layer.Name, gid, n)
			}
		}
	}
}

// assets/tilesets の全タイルセットが、assets内の画像を指し、画像の実際の
// 大きさと.tsxの記述が一致していること。画像の横幅が変わるとタイル番号が
// ずれて全マップが崩れるので、Tiledで開き直して保存するまでここで止める。
func checkTilesetFiles(c *mapChecker) {
	paths, _ := fs.Glob(embeddedAssets, "assets/tilesets/*")
	if len(paths) == 0 {
		c.add("assets/tilesets", "タイルセットがありません")
	}
	for _, p := range paths {
		def, err := loadTilesetDef(p)
		if err != nil {
			c.add(p, "%v", err)
			continue
		}
		// ゲームが使わない(黙って無視する)タイルセットの設定。
		if raw, err := embeddedAssets.ReadFile(p); err == nil {
			if bytes.Contains(raw, []byte("<objectgroup")) || bytes.Contains(raw, []byte(`"objectgroup"`)) {
				c.add(p, "タイルの当たり判定エディタで形が描いてあります。ゲームでは使われないので、通れなくしたいタイルには wall=true を付けてください")
			}
			if bytes.Contains(raw, []byte("<tileoffset")) || bytes.Contains(raw, []byte(`"tileoffset"`)) {
				c.add(p, "タイルのずらし(描画オフセット)が設定してあります。ゲームではずれないので 0 に戻してください")
			}
		}
		img := resolveRelativeAssetPath(p, def.Image)
		if !strings.HasPrefix(img, "assets/images/") {
			c.add(p, "画像が assets/images/ の外を指しています: %s (先に画像をassets/images/tiles/へコピーしてからTiledで指定してください)", def.Image)
			continue
		}
		data, err := embeddedAssets.ReadFile(img)
		if err != nil {
			c.add(p, "画像がありません: %s", img)
			continue
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			c.add(img, "%v", err)
			continue
		}
		if cfg.Width != def.ImageWidth || cfg.Height != def.ImageHeight {
			c.add(p, "画像 %s の大きさが %dx%d に変わっています(.tsxは%dx%d)。Tiledでタイルセットを開いて保存し直し、タイル番号がずれていないか確かめてください",
				img, cfg.Width, cfg.Height, def.ImageWidth, def.ImageHeight)
		}
		if def.TileWidth <= 0 || def.TileHeight <= 0 || def.Columns <= 0 {
			c.add(p, "タイルの大きさ・列数が不正です")
			continue
		}
		rows := (def.ImageHeight - 2*def.Margin + def.Spacing) / (def.TileHeight + def.Spacing)
		cols := (def.ImageWidth - 2*def.Margin + def.Spacing) / (def.TileWidth + def.Spacing)
		if cols != def.Columns || def.TileCount > rows*cols {
			c.add(p, "列数%d・タイル数%dが画像(%d列x%d行)と合いません", def.Columns, def.TileCount, cols, rows)
		}
		for _, tile := range def.Tiles {
			for _, f := range tile.Animation {
				if f.TileID < 0 || f.TileID >= def.TileCount {
					c.add(p, "タイル%d のアニメーションのコマ(タイル%d)がタイルセットの外です", tile.ID, f.TileID)
				}
				if f.Duration <= 0 {
					c.add(p, "タイル%d のアニメーションのコマの長さが0です(1ミリ秒以上にしてください)", tile.ID)
				}
			}
		}
	}
}

// ワープ(targetmapを持つオブジェクト)の行き先のマップがあり、targetpointと
// 同じ名前の着地点が行き先にあること。また、assets/maps のどのマップも開始
// マップからワープでたどれること(たどれないマップは目的地案内やドアの
// 経路探索の対象に入らない)。綴りミスはゲーム中ではエラーにならず
// 既定の座標に飛ぶだけなので、ここで止める。
func checkMapWarps(c *mapChecker) {
	mapPaths, maps := loadMapsForCheck(c)
	spawnNames := map[string]map[string]bool{}
	for _, mapPath := range mapPaths {
		names := map[string]bool{}
		forEachEventObject(maps[mapPath], func(obj TiledObject, p map[string]string) {
			if obj.Name != "" {
				names[obj.Name] = true
			}
		})
		spawnNames[mapPath] = names
	}
	for _, mapPath := range mapPaths {
		forEachEventObject(maps[mapPath], func(obj TiledObject, p map[string]string) {
			target := p["targetmap"]
			if target == "" {
				return
			}
			names, ok := spawnNames[target]
			if !ok {
				c.add(mapPath, "ドア(id%d)の「行き先マップ」%q がありません", obj.ID, target)
				return
			}
			if point := p["targetpoint"]; point == "" {
				c.add(mapPath, "ドア(id%d)の「着地点」が空です", obj.ID)
			} else if !names[point] {
				c.add(mapPath, "ドア(id%d)の着地点 %q が %s にありません(行き先マップの着地点の「名前」と同じにしてください)", obj.ID, point, target)
			}
		})
	}

	if err := BuildObjectiveAndMapIndex(); err != nil {
		c.add(startMapPath, "%v", err)
		return
	}
	reachable := map[string]bool{}
	for _, p := range allMapPaths {
		reachable[p] = true
	}
	for _, mapPath := range mapPaths {
		if !reachable[mapPath] {
			c.add(mapPath, "ゲームの最初のマップ(%s)からドアでたどれません。どこかのマップに、このマップへのドアを置いてください", startMapPath)
		}
	}
}

// leverプロパティを持つオブジェクトは、壁(type=event, text=event_wall)か
// 暗闇(type=darkness)でないとゲームに無視される。参照するレバーも同じマップに
// あること。leveropenレイヤーのタイルはレバー壁の範囲に置くこと(範囲外は
// 一生表示されない)。どれもゲーム中はエラーにならず何も起きないだけなので、
// ここで止める。
func checkMapLeverObjects(c *mapChecker) {
	mapPaths, maps := loadMapsForCheck(c)
	for _, mapPath := range mapPaths {
		tmap := maps[mapPath]
		levers := map[string]bool{}
		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			if isLeverObj(p) {
				levers[p["id"]] = true
			}
		})
		forEachEventObject(tmap, func(obj TiledObject, p map[string]string) {
			lever := p["lever"]
			if lever == "" {
				return
			}
			if !isWallObj(p) && p["type"] != "darkness" {
				c.add(mapPath, "オブジェクト(id%d)に lever がありますが、ほかの種類(type/text)になっているため壁になりません。レバー壁なら type と text を消すか type=event, text=event_wall にしてください", obj.ID)
			}
			if !levers[lever] {
				c.add(mapPath, "「レバーの壁」(id%d)のレバーの名前 %q の「レバー」が、このマップにありません", obj.ID, lever)
			}
		})
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
				c.add(mapPath, "レバーで出るレイヤー%q の絵が%dマス、どの「レバーの壁」の範囲にも入っていないので出てきません", layer.Name, n)
			}
		}
	}
}

// startMapCandidates は「ゲームの最初のマップ」にチェックを入れたマップ。
func startMapCandidates(mapPaths []string, maps map[string]TiledMap) []string {
	var starts []string
	for _, mapPath := range mapPaths {
		for _, p := range maps[mapPath].Properties {
			if b, _ := propBool(p.Value); b && strings.EqualFold(p.Name, "startmap") {
				starts = append(starts, mapPath)
			}
		}
	}
	return starts
}

// マップ設定で「ゲームの最初のマップ」にしたマップが1つだけあり、着地点
// start_point があり、start_map_generated.go と合っていること。
func checkStartMap(c *mapChecker) {
	mapPaths, maps := loadMapsForCheck(c)
	starts := startMapCandidates(mapPaths, maps)
	if len(starts) != 1 {
		c.add("assets/maps", "マップ設定の「ゲームの最初のマップ」にチェックを入れたマップを1つだけにしてください(今: %d個 %v)", len(starts), starts)
		return
	}
	start := starts[0]
	hasStartPoint := false
	forEachEventObject(maps[start], func(obj TiledObject, p map[string]string) {
		if obj.Name == "start_point" {
			hasStartPoint = true
		}
	})
	if !hasStartPoint {
		c.add(start, "ゲームの最初のマップには、名前が start_point の着地点を置いてください(ニューゲームでそこから始まる)")
	}
	if start != startMapPath {
		c.add(start, "「ゲームの最初のマップ」を変えたら go run ./tools/update を打ってください(今のゲームの最初のマップは %s のまま)", startMapPath)
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
