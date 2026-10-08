// Command newmap は空の新しいマップを assets/maps に作る。作業フォルダで:
//
//	go run ./tools/newmap dungeon_a          (120×40マス)
//	go run ./tools/newmap dungeon_a 60 40    (横60×縦40マス)
//
// 中身:
//   - タイルセット: assets/tilesets の .tsx を全部付ける
//   - レイヤー(下から): 床 / 壁 / しかけ / 主人公の高さ / 主人公より上 / 通れない四角
//     (床と主人公より上以外は、働きに合ったクラスを付けてある)
//   - マップのクラスは「マップ設定」(地名などはTiledで書く)
//   - events に着地点 start_point を1つ(真ん中)。すぐF5で試せる
//
// 既にある名前のマップは上書きしない。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	mapsDir     = "assets/maps"
	tilesetsDir = "assets/tilesets"
	tileSize    = 32
	defaultW    = 120
	defaultH    = 40
)

var (
	validName    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	tilecountAtt = regexp.MustCompile(`<tileset[^>]*\btilecount="(\d+)"`)
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "newmap:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("作業フォルダ(go.mod のある場所)で実行してください")
	}
	if len(args) != 1 && len(args) != 3 {
		return fmt.Errorf("使い方: go run ./tools/newmap 名前 [横のマス数 縦のマス数]")
	}
	name := strings.TrimSuffix(args[0], ".tmj")
	if !validName.MatchString(name) {
		return fmt.Errorf("名前 %q は使えません。英小文字で始め、英小文字・数字・_ だけにしてください(例: dungeon_a)", name)
	}
	w, h := defaultW, defaultH
	if len(args) == 3 {
		var err1, err2 error
		w, err1 = strconv.Atoi(args[1])
		h, err2 = strconv.Atoi(args[2])
		if err1 != nil || err2 != nil || w < 10 || h < 10 || w > 500 || h > 500 {
			return fmt.Errorf("大きさは 10〜500 の数で書いてください(例: 60 40)")
		}
	}
	outPath := filepath.Join(mapsDir, name+".tmj")
	if _, err := os.Stat(outPath); err == nil {
		return fmt.Errorf("%s はもうあります。別の名前にしてください", outPath)
	}

	tilesets, err := tilesetRefs()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(newMap(w, h, tilesets), "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("%s を作りました(%d×%dマス)。\n", outPath, w, h)
	fmt.Println("Tiledで開いて、「マップ」→「マッププロパティ」で地名を書き、ほかのマップとドアでつないでください。")
	return nil
}

// tilesetRefs は assets/tilesets の .tsx を名前順に並べ、firstgid を振る。
func tilesetRefs() ([]map[string]any, error) {
	paths, err := filepath.Glob(filepath.Join(tilesetsDir, "*.tsx"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%s に .tsx がありません", tilesetsDir)
	}
	sort.Strings(paths)
	var refs []map[string]any
	firstgid := 1
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		m := tilecountAtt.FindSubmatch(b)
		if m == nil {
			return nil, fmt.Errorf("%s の tilecount が読めません", p)
		}
		count, _ := strconv.Atoi(string(m[1]))
		refs = append(refs, map[string]any{
			"firstgid": firstgid,
			"source":   "../tilesets/" + filepath.Base(p),
		})
		firstgid += count
	}
	return refs, nil
}

func newMap(w, h int, tilesets []map[string]any) map[string]any {
	empty := func() []int { return make([]int, w*h) }
	tileLayer := func(id int, name string) map[string]any {
		return map[string]any{
			"data": empty(), "height": h, "id": id, "name": name, "opacity": 1,
			"type": "tilelayer", "visible": true, "width": w, "x": 0, "y": 0,
		}
	}
	objectLayer := func(id int, name string, objects []any) map[string]any {
		return map[string]any{
			"draworder": "topdown", "id": id, "name": name, "objects": objects, "opacity": 1,
			"type": "objectgroup", "visible": true, "x": 0, "y": 0,
		}
	}
	// withClass はレイヤーのクラス(通れないレイヤー・しかけレイヤーなど)を付ける。
	withClass := func(l map[string]any, class string) map[string]any {
		l["class"] = class
		return l
	}
	start := map[string]any{
		"height": tileSize, "id": 1, "name": "start_point", "rotation": 0, "type": "着地点",
		"visible": true, "width": tileSize, "x": (w / 2) * tileSize, "y": (h / 2) * tileSize,
	}
	return map[string]any{
		"class":            "マップ設定",
		"compressionlevel": -1,
		"height":           h,
		"infinite":         false,
		"layers": []any{
			tileLayer(1, "床"),
			withClass(tileLayer(2, "壁"), "通れないレイヤー"),
			withClass(objectLayer(3, "しかけ", []any{start}), "しかけレイヤー"),
			withClass(objectLayer(4, "主人公の高さ", []any{}), "主人公の高さ"),
			tileLayer(5, "主人公より上"),
			withClass(objectLayer(6, "通れない四角", []any{}), "通れないレイヤー"),
		},
		"nextlayerid":  7,
		"nextobjectid": 2,
		"orientation":  "orthogonal",
		"renderorder":  "right-down",
		"tiledversion": "1.12.1",
		"tileheight":   tileSize,
		"tilesets":     tilesets,
		"tilewidth":    tileSize,
		"type":         "map",
		"version":      "1.10",
		"width":        w,
	}
}
