package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// fillRect は非推奨になった ebitenutil.DrawRect の置き換え。
// float64座標のまま呼べるようにし、アンチエイリアス無しで塗りつぶす
// （ebitenutil.DrawRect と同じ見た目）。
func fillRect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), clr, false)
}
