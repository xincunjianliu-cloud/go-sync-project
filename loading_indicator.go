package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	loadingLabel        = "Loading"
	loadingDotCount     = 3
	loadingCharW        = 6
	loadingCharH        = 16
	loadingScale        = 2.0
	loadingMarginX      = 16
	loadingMarginY      = 14
	loadingBounceHeight = 8.0
	loadingBouncePeriod = 0.7
	loadingDotStagger   = 0.15
)

// loadingGlyphBuf はDebugPrintの6x16px文字を拡大して描くための作業用画像。
var loadingGlyphBuf *ebiten.Image

// drawLoadingIndicator は画面右下に"Loading"と、その後ろではねる"..."を描く。
// 文字("Loading")は動かず、ドットだけが順番に上下にはねる。
// 起動直後(フォント/タイトル画像の取得中)と、戦闘・メニュー用画像を
// バックグラウンドで読み込んでいる間の両方で使う。g.fontSourceが
// まだ読み込まれていない起動直後でも描画できるよう、独自フォントの
// text.Drawではなくebitenutil.DebugPrintAtの組み込みビットマップフォント
// (ASCII専用、6x16px)を整数倍に拡大して使う。
func drawLoadingIndicator(screen *ebiten.Image, t float64) {
	if loadingGlyphBuf == nil {
		loadingGlyphBuf = ebiten.NewImage((len(loadingLabel)+loadingDotCount)*loadingCharW, loadingCharH)
	}

	cellW := loadingCharW * loadingScale
	totalW := float64(len(loadingLabel)+loadingDotCount) * cellW
	x0 := float64(gameWidth-loadingMarginX) - totalW
	y0 := float64(gameHeight-loadingMarginY) - loadingCharH*loadingScale - loadingBounceHeight

	drawText := func(s string, x, y float64) {
		loadingGlyphBuf.Clear()
		ebitenutil.DebugPrintAt(loadingGlyphBuf, s, 0, 0)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(loadingScale, loadingScale)
		op.GeoM.Translate(x, y)
		screen.DrawImage(loadingGlyphBuf, op)
	}

	drawText(loadingLabel, x0, y0+loadingBounceHeight)

	for i := 0; i < loadingDotCount; i++ {
		phase := t - float64(i)*loadingDotStagger
		bounce := loadingBounceHeight * math.Abs(math.Sin(phase*math.Pi/loadingBouncePeriod))
		x := x0 + float64(len(loadingLabel)+i)*cellW
		drawText(".", x, y0+loadingBounceHeight-bounce)
	}
}
