package main

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	loadingLabel        = "Loading"
	loadingCharW        = 6
	loadingCharH        = 16
	loadingMarginX      = 16
	loadingMarginY      = 14
	loadingBounceHeight = 6.0
	loadingBouncePeriod = 0.7
	loadingDotPeriod    = 0.35
)

// drawLoadingIndicator は画面右下に上下にはねる"Loading..."を描く。
// 起動直後(フォント/タイトル画像の取得中)と、戦闘・メニュー用画像を
// バックグラウンドで読み込んでいる間の両方で使う。g.fontSourceが
// まだ読み込まれていない起動直後でも描画できるよう、独自フォントの
// text.Drawではなくebitenutil.DebugPrintAtの組み込みビットマップフォント
// (ASCII専用、6x16px)を使う。
func drawLoadingIndicator(screen *ebiten.Image, t float64) {
	dotCount := int(t/loadingDotPeriod) % 4
	label := loadingLabel + strings.Repeat(".", dotCount)

	// ドットの数で右端の位置がぶれないよう、最大幅で右寄せ位置を固定する。
	maxWidth := (len(loadingLabel) + 3) * loadingCharW
	x := gameWidth - loadingMarginX - maxWidth

	bounce := loadingBounceHeight * math.Abs(math.Sin(t*math.Pi/loadingBouncePeriod))
	y := gameHeight - loadingMarginY - loadingCharH - int(bounce)

	ebitenutil.DebugPrintAt(screen, label, x, y)
}
