package main

// mapDef は1つのマップに関する設定を1箇所にまとめたもの。以前はマップの
// パス一覧・使用タイルセット・ダンジョン自動回復フラグ・表示名がそれぞれ
// 別のファイルにハードコードされていて、新しいマップを追加するたびに
// 複数箇所を同時に直す必要があった。新しいマップを追加するときは
// ここに1エントリ足すだけでよい。
type mapDef struct {
	Path        string
	DisplayName string
	TilesetKey  string
	AutoHeal    bool
}

var mapRegistry = []mapDef{
	{
		Path:        "assets/maps/School_Map_1.tmj",
		DisplayName: "理科室",
		TilesetKey:  "rouka",
	},
	{
		Path:        "assets/maps/ダンジョンA.tmj",
		DisplayName: "ダンジョンA",
		TilesetKey:  "dungeon",
		AutoHeal:    true,
	},
}

func mapDefFor(mapPath string) (mapDef, bool) {
	for _, m := range mapRegistry {
		if m.Path == mapPath {
			return m, true
		}
	}
	return mapDef{}, false
}

func allRegisteredMapPaths() []string {
	paths := make([]string, len(mapRegistry))
	for i, m := range mapRegistry {
		paths[i] = m.Path
	}
	return paths
}
