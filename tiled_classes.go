package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Tiledの「クラス」(カスタムタイプ)でしかけを作れるようにする。
//
// オブジェクトのクラスに「宝箱」などを選ぶと、Tiledに「中身」などの欄が出て、
// アイテムや敵は一覧から選べる。ゲームはマップを読んだときに、クラスと欄の
// 値を従来の名札(type=event, text=event_chest_potion など)に読み替えるので、
// ほかの処理は今までどおり動く。クラスを使わない従来の書き方もそのまま動く。
//
// 欄の定義はここが元で、rpg.tiled-project はここから作る
// (TestTiledProjectUpToDate。作り直し方はそのテストのエラーに出る)。
// Tiledは既定値のままの欄をファイルに書かないので、既定値は必ず
// 「何も書かなかったとき」と同じ意味にしておく。

// 一覧(enum)で「選んでいない」を表す値。
const (
	enumUnset   = "(選んでください)"
	enumNone    = "(なし)"
	enumDefault = "(いつもの)"
	enumKeep    = "(そのまま)"
)

// レイヤーのクラス名。働きのあるレイヤーは名前ではなくクラスで決める。
const (
	blockingLayerClass = "通れないレイヤー" // 壁・水などを通れなくする
	eventsLayerClass   = "しかけレイヤー"  // 宝箱・ドアなどを置く
	playerLayerClass   = "主人公の高さ"   // 主人公を描く重なり順
)

type classMember struct {
	Name string // Tiledに出る欄の名前
	Type string // string / int / bool / file
	Enum string // 一覧の名前(classEnums のキー)。一覧でなければ空
	// Default はTiledでの既定値。ゲームが「何も書かなかった」ときと同じ意味にする。
	Default any
	// Key は読み替え先の名札。空なら下の apply で扱う。
	Key string
}

type tiledClass struct {
	Name    string
	UseAs   string // object / map / layer
	Color   string // Tiled上の色(#AARRGGBB)
	Members []classMember
	// Type/Text は読み替えるときに付ける type と text(固定のもの)。
	Type, Text string
	// apply は欄の値から、名札を作る(Key で済まないもの)。
	apply func(v map[string]string, set func(key, value string))
}

// 名札に書く値への変換。一覧の表示名(日本語)を、ゲームのIDに戻す。
var bgmLabels = []struct{ Key, Label string }{
	{"field1", "フィールド1"}, {"field2", "フィールド2"}, {"field3", "フィールド3"}, {"field4", "フィールド4"},
	{"title", "タイトル"}, {"battle_normal", "戦闘"}, {"battle_boss", "ボス戦"}, {"battle_lastboss", "ラスボス戦"},
	{"ending", "エンディング"},
	{"talk_peaceful", "会話・平和"}, {"talk_tense", "会話・緊迫"}, {"talk_scary", "会話・恐怖"}, {"talk_sad", "会話・悲しみ"},
}

var dirLabels = []string{"下", "左", "右", "上"}

func storyField(v map[string]string, textKey string) string {
	if id := v["会話データ"]; id != "" && id != enumNone {
		return storyTextPrefix + id
	}
	return v[textKey]
}

func setObjective(v map[string]string, set func(string, string)) {
	if v["目的地の名前"] == "" {
		return
	}
	set("objectiveid", v["目的地の名前"])
	set("objectiveorder", v["目的地の順番"])
	if b := v["達成するボス"]; b != "" && b != enumNone {
		set("bossid", b)
	}
}

var objectiveMembers = []classMember{
	{Name: "目的地の名前", Type: "string", Default: ""},
	{Name: "目的地の順番", Type: "int", Default: 0},
	{Name: "達成するボス", Type: "string", Enum: "ボス", Default: enumNone},
}

var tiledClasses = []tiledClass{
	{Name: "ドア", UseAs: "object", Color: "#ff3c78d8",
		Members: []classMember{
			{Name: "行き先マップ", Type: "file", Default: ""},
			{Name: "着地点", Type: "string", Default: "", Key: "targetpoint"},
			{Name: "倒すまで通れないボス", Type: "string", Enum: "ボス", Default: enumNone},
		},
		apply: func(v map[string]string, set func(string, string)) {
			set("targetmap", v["行き先マップ"])
			if b := v["倒すまで通れないボス"]; b != enumNone {
				set("requireboss", b)
			}
		}},
	{Name: "着地点", UseAs: "object", Color: "#ff6aa84f",
		Members: []classMember{{Name: "向き", Type: "string", Enum: "向き", Default: enumKeep}},
		apply: func(v map[string]string, set func(string, string)) {
			for i, l := range dirLabels {
				if v["向き"] == l {
					set("dir", strconv.Itoa(i))
				}
			}
		}},
	{Name: "会話", UseAs: "object", Color: "#ff8e7cc3", Type: evTypeEvent,
		Members: append([]classMember{
			{Name: "セリフ", Type: "string", Default: ""},
			{Name: "会話データ", Type: "string", Enum: "会話", Default: enumNone},
			{Name: "2回目からのセリフ", Type: "string", Default: "", Key: "repeattext"},
		}, objectiveMembers...),
		apply: func(v map[string]string, set func(string, string)) {
			set("text", storyField(v, "セリフ"))
			setObjective(v, set)
		}},
	{Name: "宝箱", UseAs: "object", Color: "#ffe69138", Type: evTypeEvent,
		Members: []classMember{{Name: "中身", Type: "string", Enum: "アイテム", Default: enumUnset}},
		apply: func(v map[string]string, set func(string, string)) {
			set("text", chestTextPrefix+itemIDFromLabel(v["中身"]))
		}},
	{Name: "鍵の宝箱", UseAs: "object", Color: "#fff1c232", Type: evTypeEvent,
		Members: []classMember{{Name: "鍵の名前", Type: "string", Default: ""}},
		apply: func(v map[string]string, set func(string, string)) {
			set("text", chestKeyTextPrefix+v["鍵の名前"])
		}},
	{Name: "鍵の壁", UseAs: "object", Color: "#ff999999", Type: evTypeEvent, Text: evTextWall,
		Members: []classMember{{Name: "必要な鍵", Type: "string", Default: "", Key: "keys"}}},
	{Name: "レバー", UseAs: "object", Color: "#ffcc0000", Type: evTypeEvent, Text: evTextLever,
		Members: []classMember{
			{Name: "レバーの名前", Type: "string", Default: "", Key: "id"},
			{Name: "一度きり", Type: "bool", Default: false},
		},
		apply: func(v map[string]string, set func(string, string)) {
			if v["一度きり"] == "true" {
				set("oneway", "true")
			}
		}},
	{Name: "レバーの壁", UseAs: "object", Color: "#ffe06666", Type: evTypeEvent, Text: evTextWall,
		Members: []classMember{
			{Name: "レバーの名前", Type: "string", Default: "", Key: "lever"},
			{Name: "見た目だけ", Type: "bool", Default: false},
		},
		apply: func(v map[string]string, set func(string, string)) {
			if v["見た目だけ"] == "true" {
				set("passable", "false")
			}
		}},
	{Name: "ブロック", UseAs: "object", Color: "#ffb45f06", Type: evTypeEvent, Text: evTextBlock,
		Members: []classMember{{Name: "ブロックの名前", Type: "string", Default: "", Key: "id"}}},
	{Name: "ブロック置き場", UseAs: "object", Color: "#fff6b26b", Type: evTypeEvent, Text: evTextBlockSpot,
		Members: []classMember{{Name: "置き場の名前", Type: "string", Default: "", Key: "id"}}},
	{Name: "ブロック扉", UseAs: "object", Color: "#ff783f04", Type: evTypeEvent, Text: evTextBlockDoor,
		Members: []classMember{{Name: "置き場の名前", Type: "string", Default: "", Key: "spots"}}},
	{Name: "イベント", UseAs: "object", Color: "#ff674ea7", Type: "trigger",
		Members: append([]classMember{
			{Name: "セリフ", Type: "string", Default: ""},
			{Name: "会話データ", Type: "string", Enum: "会話", Default: enumNone},
			{Name: "ボス戦", Type: "string", Enum: "ボス", Default: enumNone},
			{Name: "歩く道順", Type: "string", Default: "", Key: "route"},
		}, objectiveMembers[:2]...),
		apply: func(v map[string]string, set func(string, string)) {
			set("text", storyField(v, "セリフ"))
			if b := v["ボス戦"]; b != "" && b != enumNone {
				set("bossid", b)
			}
			if v["目的地の名前"] != "" {
				set("objectiveid", v["目的地の名前"])
				set("objectiveorder", v["目的地の順番"])
			}
		}},
	{Name: "ボス", UseAs: "object", Color: "#ff990000", Type: "boss",
		Members: []classMember{
			{Name: "ボス番号", Type: "string", Enum: "ボス", Default: enumNone},
			{Name: "目的地の名前", Type: "string", Default: "", Key: "objectiveid"},
		},
		apply: func(v map[string]string, set func(string, string)) {
			if b := v["ボス番号"]; b != enumNone {
				set("bossid", b)
			}
		}},
	{Name: "敵が出る場所", UseAs: "object", Color: "#40ff0000", Type: "enemy",
		Members: []classMember{
			{Name: "出る敵", Type: "string", Enum: "敵", Default: "", Key: "text"},
			{Name: "一度に出る最大数", Type: "int", Default: 1, Key: "maxcount"},
		}},
	{Name: "暗い場所", UseAs: "object", Color: "#80000000", Type: "darkness",
		Members: []classMember{{Name: "明るくするレバー", Type: "string", Default: "", Key: "lever"}}},

	{Name: "マップ設定", UseAs: "map", Color: "#ff3d85c6",
		Members: []classMember{
			{Name: "地名", Type: "string", Default: "", Key: "displayname"},
			{Name: "曲", Type: "string", Enum: "曲", Default: "フィールド1"},
			{Name: "戦闘背景", Type: "string", Enum: "戦闘背景", Default: enumDefault},
			{Name: "入ると全回復", Type: "bool", Default: false},
			{Name: "ゲームの最初のマップ", Type: "bool", Default: false},
		},
		apply: func(v map[string]string, set func(string, string)) {
			set("bgm", bgmKeyFromLabel(v["曲"]))
			if bg := v["戦闘背景"]; bg != enumDefault {
				set("battlebg", bg)
			}
			if v["入ると全回復"] == "true" {
				set("autoheal", "true")
			}
			if v["ゲームの最初のマップ"] == "true" {
				set("startmap", "true")
			}
		}},
	{Name: "レバーで出るレイヤー", UseAs: "layer", Color: "#ffe06666",
		apply: func(v map[string]string, set func(string, string)) {
			set("leveropen", "true")
		}},
	// タイルレイヤーなら描いたタイル、オブジェクトレイヤーなら置いた四角が通れない。
	{Name: blockingLayerClass, UseAs: "layer", Color: "#ff666666",
		apply: func(v map[string]string, set func(string, string)) {
			set("blocking", "true")
		}},
	// 宝箱・ドアなどのしかけを置くオブジェクトレイヤー。
	{Name: eventsLayerClass, UseAs: "layer", Color: "#ff674ea7",
		apply: func(v map[string]string, set func(string, string)) {
			set("eventlayer", "true")
		}},
	// 主人公を描く重なり順の位置。何も置かない。
	{Name: playerLayerClass, UseAs: "layer", Color: "#ff3d85c6",
		apply: func(v map[string]string, set func(string, string)) {
			set("playerlayer", "true")
		}},
}

func itemIDFromLabel(label string) string {
	for _, it := range ItemDatabase {
		if it.Name == label || it.ID == label {
			return it.ID
		}
	}
	if label == enumUnset {
		return ""
	}
	return label
}

func bgmKeyFromLabel(label string) string {
	for _, b := range bgmLabels {
		if b.Label == label || b.Key == label {
			return b.Key
		}
	}
	return label
}

func findTiledClass(name, useAs string) (tiledClass, bool) {
	if name == "" {
		return tiledClass{}, false
	}
	for _, c := range tiledClasses {
		if c.Name == name && c.UseAs == useAs {
			return c, true
		}
	}
	return tiledClass{}, false
}

// classProperties はクラスの欄の値を、従来の名札に読み替えた一覧を返す。
// 欄でない名札(従来の書き方で直接付けたもの)はそのまま残す。
// relTo はファイル欄(行き先マップ)の基準にするファイル。
func classProperties(c tiledClass, props []TiledProperty, relTo string) []TiledProperty {
	isMember := map[string]classMember{}
	for _, m := range c.Members {
		isMember[m.Name] = m
	}
	v := map[string]string{}
	for _, m := range c.Members {
		v[m.Name] = fmt.Sprintf("%v", m.Default)
	}
	var out []TiledProperty
	for _, p := range props {
		m, ok := isMember[p.Name]
		if !ok {
			out = append(out, p)
			continue
		}
		s := fmt.Sprintf("%v", p.Value)
		if m.Type == "file" && s != "" {
			s = resolveRelativeAssetPath(relTo, s)
		}
		v[p.Name] = s
	}
	have := map[string]bool{}
	for _, p := range out {
		have[strings.ToLower(p.Name)] = true
	}
	set := func(key, value string) {
		if value == "" || have[key] {
			return
		}
		have[key] = true
		out = append(out, TiledProperty{Name: key, Type: "string", Value: value})
	}
	set("type", c.Type)
	set("text", c.Text)
	if c.apply != nil {
		c.apply(v, set)
	}
	for _, m := range c.Members {
		if m.Key != "" {
			set(m.Key, v[m.Name])
		}
	}
	return out
}

// applyTiledClasses はマップの中のクラス付きのオブジェクト・レイヤーと、
// マップ自体のクラスを、従来の名札に読み替える。loadTiledMapから呼ぶ。
func applyTiledClasses(tmap *TiledMap, mapPath string) {
	if c, ok := findTiledClass(tmap.Class, "map"); ok {
		tmap.Properties = classProperties(c, tmap.Properties, mapPath)
	}
	for li := range tmap.Layers {
		layer := &tmap.Layers[li]
		if c, ok := findTiledClass(layer.Class, "layer"); ok {
			layer.Properties = classProperties(c, layer.Properties, mapPath)
		}
		for oi := range layer.Objects {
			obj := &layer.Objects[oi]
			if c, ok := findTiledClass(obj.Class, "object"); ok {
				obj.Properties = classProperties(c, obj.Properties, mapPath)
			}
		}
	}
}
