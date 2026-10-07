package main

import "strings"

// レバーで出し入れするタイル。
//
// タイルレイヤーにカスタムプロパティ leveropen(bool) = true を付けると、
// そのレイヤーのタイルは「レバー壁(type=event, text=event_wall, lever=<id>)が
// 開いている範囲」でだけ描かれ、当たり判定(wallタイルなど)も持つ。
// 例: 水の上に机の橋を並べたレイヤーにleveropenを付け、同じ範囲にレバー壁を
// 置くと、レバーを上げたときだけ机が現れて渡れるようになる。
// タイルを置いたレバー壁では、既定の開放画像(lever_wall_open*.png)は描かない。
// Tiledでの表示/非表示(目のアイコン)はゲームには影響しない。

// isLeverOpenLayer はleveropen=trueのタイルレイヤーか。
func isLeverOpenLayer(layer TiledLayer) bool {
	for _, p := range layer.Properties {
		if b, _ := propBool(p.Value); b && strings.EqualFold(p.Name, "leveropen") {
			return true
		}
	}
	return false
}

// buildLeverTileIndex はleveropenレイヤーのマスとレバー壁の対応を作る。
// leveropenレイヤーが無いマップでは何もしない。
func buildLeverTileIndex(tmap *TiledMap) {
	var leverLayers []TiledLayer
	for _, layer := range tmap.Layers {
		if layer.Type == "tilelayer" && isLeverOpenLayer(layer) {
			leverLayers = append(leverLayers, layer)
		}
	}
	if len(leverLayers) == 0 || tmap.Width <= 0 || tmap.TileWidth <= 0 || tmap.TileHeight <= 0 {
		return
	}

	cells := make([]string, tmap.Width*tmap.Height)
	cellWall := make([]int, len(cells))
	for _, layer := range tmap.Layers {
		if !strings.HasPrefix(layer.Name, "events") {
			continue
		}
		for _, obj := range layer.Objects {
			p := objProps(obj)
			if !isLeverControlledWallObj(p) {
				continue
			}
			x0, y0 := int(obj.X)/tmap.TileWidth, int(obj.Y)/tmap.TileHeight
			x1 := (int(obj.X+obj.Width) + tmap.TileWidth - 1) / tmap.TileWidth
			y1 := (int(obj.Y+obj.Height) + tmap.TileHeight - 1) / tmap.TileHeight
			for ty := max(y0, 0); ty < min(y1, tmap.Height); ty++ {
				for tx := max(x0, 0); tx < min(x1, tmap.Width); tx++ {
					i := ty*tmap.Width + tx
					if cells[i] == "" {
						cells[i] = p["lever"]
						cellWall[i] = obj.ID
					}
				}
			}
		}
	}

	walls := map[int]bool{}
	for _, layer := range leverLayers {
		for i, id := range layer.Data {
			if id != 0 && i < len(cells) && cells[i] != "" {
				walls[cellWall[i]] = true
			}
		}
	}
	tmap.leverCellLevers = cells
	tmap.leverWallsWithTiles = walls
}

// leverTileShown はleveropenレイヤーのマスiのタイルを今出すかどうか。
func (s *FieldScene) leverTileShown(i int) bool {
	cells := s.tileMap.leverCellLevers
	if i < 0 || i >= len(cells) || cells[i] == "" {
		return false
	}
	return s.game.RaisedLevers[cells[i]]
}
