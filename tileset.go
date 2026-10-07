package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
)

// タイルセットの運用ルール(docs/マップ作りガイド.md の3-2も参照):
//   - タイルセットはテーマごとに分け、外部タイルセット(assets/tilesets/*.tsx)にする。
//     マップ(.tmj)には参照(firstgid + source)だけを書く。画像パスや当たり判定は
//     .tsx の1か所にしか無いので、差し替えると全マップに反映される。
//   - 1枚のマップで複数のタイルセットを使ってよい(firstgidで振り分ける)。
//   - タイルのカスタムプロパティ wall(bool) が true のタイルは、どのレイヤーに
//     置いても通れない。
// 違反はassets_audit_test.goのテストが検出し、CIがデプロイを止める。

// Tiledがタイル番号(GID)の上位ビットに書く反転フラグ。
const (
	gidFlipH    = 0x80000000
	gidFlipV    = 0x40000000
	gidFlipD    = 0x20000000
	gidFlipHex  = 0x10000000
	gidFlagMask = gidFlipH | gidFlipV | gidFlipD | gidFlipHex
)

// TiledTileset は.tmjの"tilesets"欄の1件。外部タイルセットならSourceだけが、
// マップに埋め込まれたタイルセットなら定義(tilesetDef)が入っている。
type TiledTileset struct {
	FirstGID int    `json:"firstgid"`
	Source   string `json:"source"`
	tilesetDef
}

// tilesetDef はタイルセット1つ分の定義(.tsx/.tsjの中身、または埋め込み)。
type tilesetDef struct {
	Name        string         `json:"name"`
	Image       string         `json:"image"`
	ImageWidth  int            `json:"imagewidth"`
	ImageHeight int            `json:"imageheight"`
	TileWidth   int            `json:"tilewidth"`
	TileHeight  int            `json:"tileheight"`
	TileCount   int            `json:"tilecount"`
	Columns     int            `json:"columns"`
	Margin      int            `json:"margin"`
	Spacing     int            `json:"spacing"`
	Tiles       []tiledTileDef `json:"tiles"`
}

type tiledTileDef struct {
	ID         int             `json:"id"`
	Properties []TiledProperty `json:"properties"`
	// Animation はTiledのタイルアニメーション(タイルセットエディタで作る)。
	Animation []tileFrame `json:"animation"`
}

// tileFrame はアニメーションの1コマ。TileIDは同じタイルセットの中の番号。
type tileFrame struct {
	TileID   int `json:"tileid" xml:"tileid,attr"`
	Duration int `json:"duration" xml:"duration,attr"` // ミリ秒
}

// mapTileset はマップが使うタイルセットを、パスを解決した状態で持つ。
type mapTileset struct {
	firstGID int
	// source は外部タイルセットファイルの実パス(埋め込みなら空)。
	source string
	// imagePath はタイルセット画像の実パス(assets/...)。
	imagePath string
	def       tilesetDef
}

// contains はgid(反転フラグを除いたもの)がこのタイルセットのタイルかを返す。
func (ts mapTileset) contains(gid int) bool {
	return gid >= ts.firstGID && gid < ts.firstGID+ts.def.TileCount
}

// tileRect はタイルセット内の番号localIDのタイルが画像のどこにあるかを返す。
func (ts mapTileset) tileRect(localID int) (x0, y0, x1, y1 int) {
	d := ts.def
	cols := d.Columns
	if cols <= 0 {
		cols = 1
	}
	x0 = d.Margin + (localID%cols)*(d.TileWidth+d.Spacing)
	y0 = d.Margin + (localID/cols)*(d.TileHeight+d.Spacing)
	return x0, y0, x0 + d.TileWidth, y0 + d.TileHeight
}

// resolveRelativeAssetPath はTiledが書き出す相対パス(relToは基準のファイル)を、
// リポジトリ内の実パスに変換する。
func resolveRelativeAssetPath(relTo, p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return path.Clean(path.Join(path.Dir(relTo), p))
}

// resolveMapTilesets はマップの"tilesets"欄を解決し、外部タイルセットを読み込む。
func resolveMapTilesets(mapPath string, refs []TiledTileset) ([]mapTileset, error) {
	out := make([]mapTileset, 0, len(refs))
	for _, ref := range refs {
		ts := mapTileset{firstGID: ref.FirstGID, def: ref.tilesetDef}
		defPath := mapPath
		if ref.Source != "" {
			ts.source = resolveRelativeAssetPath(mapPath, ref.Source)
			def, err := loadTilesetDef(ts.source)
			if err != nil {
				return nil, err
			}
			ts.def = def
			defPath = ts.source
		}
		if ts.def.Image != "" {
			ts.imagePath = resolveRelativeAssetPath(defPath, ts.def.Image)
		}
		out = append(out, ts)
	}
	return out, nil
}

var (
	tilesetDefCache   = map[string]tilesetDef{}
	tilesetDefCacheMu sync.Mutex
)

// loadTilesetDef は外部タイルセット(.tsx または .tsj/.json)を読み込む。
// 複数のマップが同じタイルセットを参照するので、パスごとにキャッシュする。
func loadTilesetDef(p string) (tilesetDef, error) {
	tilesetDefCacheMu.Lock()
	if d, ok := tilesetDefCache[p]; ok {
		tilesetDefCacheMu.Unlock()
		return d, nil
	}
	tilesetDefCacheMu.Unlock()

	data, err := loadAssetBytesCached(p)
	if err != nil {
		return tilesetDef{}, err
	}
	assetStore.release(p)
	var def tilesetDef
	if strings.EqualFold(path.Ext(p), ".tsx") {
		def, err = parseTSX(data)
	} else {
		err = json.Unmarshal(data, &def)
	}
	if err != nil {
		return tilesetDef{}, fmt.Errorf("タイルセット解析失敗 %s: %w", p, err)
	}

	tilesetDefCacheMu.Lock()
	tilesetDefCache[p] = def
	tilesetDefCacheMu.Unlock()
	return def, nil
}

type tsxProperty struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type tsxTileset struct {
	Name       string `xml:"name,attr"`
	TileWidth  int    `xml:"tilewidth,attr"`
	TileHeight int    `xml:"tileheight,attr"`
	TileCount  int    `xml:"tilecount,attr"`
	Columns    int    `xml:"columns,attr"`
	Margin     int    `xml:"margin,attr"`
	Spacing    int    `xml:"spacing,attr"`
	Image      struct {
		Source string `xml:"source,attr"`
		Width  int    `xml:"width,attr"`
		Height int    `xml:"height,attr"`
	} `xml:"image"`
	Tiles []struct {
		ID         int           `xml:"id,attr"`
		Properties []tsxProperty `xml:"properties>property"`
		Animation  []tileFrame   `xml:"animation>frame"`
	} `xml:"tile"`
}

// parseTSX はTiledの外部タイルセット(XML形式)を読む。
func parseTSX(data []byte) (tilesetDef, error) {
	var x tsxTileset
	if err := xml.Unmarshal(data, &x); err != nil {
		return tilesetDef{}, err
	}
	def := tilesetDef{
		Name:        x.Name,
		Image:       x.Image.Source,
		ImageWidth:  x.Image.Width,
		ImageHeight: x.Image.Height,
		TileWidth:   x.TileWidth,
		TileHeight:  x.TileHeight,
		TileCount:   x.TileCount,
		Columns:     x.Columns,
		Margin:      x.Margin,
		Spacing:     x.Spacing,
	}
	for _, t := range x.Tiles {
		td := tiledTileDef{ID: t.ID, Animation: t.Animation}
		for _, p := range t.Properties {
			td.Properties = append(td.Properties, TiledProperty{Name: p.Name, Type: p.Type, Value: tsxPropertyValue(p)})
		}
		def.Tiles = append(def.Tiles, td)
	}
	return def, nil
}

// tsxPropertyValue はXMLでは文字列で書かれるプロパティ値を、.tmj(JSON)を
// 読んだときと同じ型に揃える。
func tsxPropertyValue(p tsxProperty) any {
	switch p.Type {
	case "bool":
		return p.Value == "true"
	case "int", "float":
		if f, err := strconv.ParseFloat(p.Value, 64); err == nil {
			return f
		}
	}
	return p.Value
}

// animFrame はマップ上でのアニメーションの1コマ(GIDに直したもの)。
type animFrame struct {
	GID      int
	Duration int // ミリ秒
}

// tilesetAnimations はアニメーションの付いたタイルを、GID→コマの並びで返す。
// コマの長さが0以下のものは捨てる(Tiledで0にすると止まって見えるだけなので)。
func tilesetAnimations(tilesets []mapTileset) map[int][]animFrame {
	var anims map[int][]animFrame
	for _, ts := range tilesets {
		for _, t := range ts.def.Tiles {
			var frames []animFrame
			for _, f := range t.Animation {
				if f.Duration > 0 {
					frames = append(frames, animFrame{GID: ts.firstGID + f.TileID, Duration: f.Duration})
				}
			}
			if len(frames) == 0 {
				continue
			}
			if anims == nil {
				anims = map[int][]animFrame{}
			}
			anims[ts.firstGID+t.ID] = frames
		}
	}
	return anims
}

// animFrameGID は経過時間(ミリ秒)のときに見せるコマのGIDを返す。
// マップ全体で同じ時計を使うので、同じアニメーションのタイルはそろって動く。
func animFrameGID(frames []animFrame, elapsedMs int64) int {
	total := 0
	for _, f := range frames {
		total += f.Duration
	}
	if total <= 0 {
		return frames[0].GID
	}
	t := int(elapsedMs % int64(total))
	for _, f := range frames {
		if t < f.Duration {
			return f.GID
		}
		t -= f.Duration
	}
	return frames[len(frames)-1].GID
}

// tilesetWallGIDs はカスタムプロパティ wall=true が付いたタイルのGIDを集める。
func tilesetWallGIDs(tilesets []mapTileset) map[int]bool {
	var walls map[int]bool
	for _, ts := range tilesets {
		for _, t := range ts.def.Tiles {
			for _, p := range t.Properties {
				if b, _ := propBool(p.Value); b && strings.EqualFold(p.Name, "wall") {
					if walls == nil {
						walls = map[int]bool{}
					}
					walls[ts.firstGID+t.ID] = true
				}
			}
		}
	}
	return walls
}

// findTileset はgid(反転フラグを除いたもの)を含むタイルセットを返す。
func findTileset(tilesets []mapTileset, gid int) (mapTileset, bool) {
	for _, ts := range tilesets {
		if ts.contains(gid) {
			return ts, true
		}
	}
	return mapTileset{}, false
}

// mapTilesetImagePaths はマップが使うタイルセット画像の実パスを返す。
func mapTilesetImagePaths(tmap TiledMap) []string {
	var out []string
	for _, ts := range tmap.tilesets {
		if ts.imagePath != "" {
			out = append(out, ts.imagePath)
		}
	}
	return out
}
