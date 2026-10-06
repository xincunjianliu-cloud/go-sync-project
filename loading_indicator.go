package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:generate go run ./tools/genloadingfont

const (
	loadingLabel    = "Loading"
	loadingDotRune  = '.'
	loadingDotCount = 3
	// loadingScale は文字のドット1つを画面の何pxで描くか(整数倍でくっきり)。
	loadingScale   = 3
	loadingMarginX = 16
	loadingMarginY = 14

	// 点の動き: 1周期(loadingDotCycle秒)の頭で、点が左から順に
	// loadingDotStagger秒ずつずれて1回ずつ跳ね、残りの時間は止まって待つ。
	// 跳ねる高さは文字のドット単位で、ドット絵として段階的に動く。
	loadingDotCycle     = 1.6
	loadingDotStagger   = 0.2
	loadingDotHopTime   = 0.45
	loadingDotHopHeight = 2 // 文字のドット数

	// 進捗バーは"Loading..."の幅いっぱいに、文字のすぐ上へ描く。
	loadingBarH    = 4.0
	loadingBarGapY = 4.0
)

var (
	loadingBarBgColor = color.NRGBA{255, 255, 255, 50}
	loadingBarFgColor = color.NRGBA{255, 255, 255, 220}
)

// loadingGlyphImgs はloadingGlyphs(k8x12.ttfから取り出したドットパターン)を
// 1文字ずつ画像にしたもの。初めて描くときに作る。
var loadingGlyphImgs map[rune]*ebiten.Image

func loadingGlyphImage(r rune) *ebiten.Image {
	if loadingGlyphImgs == nil {
		loadingGlyphImgs = map[rune]*ebiten.Image{}
	}
	if img, ok := loadingGlyphImgs[r]; ok {
		return img
	}
	rows := loadingGlyphs[r]
	w := len(rows[0])
	pix := make([]byte, w*loadingGlyphHeight*4)
	for y, row := range rows {
		for x := range w {
			if row[x] == '#' {
				i := (y*w + x) * 4
				pix[i], pix[i+1], pix[i+2], pix[i+3] = 0xff, 0xff, 0xff, 0xff
			}
		}
	}
	img := ebiten.NewImage(w, loadingGlyphHeight)
	img.WritePixels(pix)
	loadingGlyphImgs[r] = img
	return img
}

// loadingGlyphWidth は文字rの幅(文字のドット数)。
func loadingGlyphWidth(r rune) int {
	return len(loadingGlyphs[r][0])
}

// loadingDotLift は点i(0始まり)が時刻tにどれだけ持ち上がっているか(文字の
// ドット数)を返す。
func loadingDotLift(t float64, i int) int {
	phase := math.Mod(t, loadingDotCycle) - float64(i)*loadingDotStagger
	if phase < 0 || phase >= loadingDotHopTime {
		return 0
	}
	return int(math.Round(loadingDotHopHeight * math.Sin(phase/loadingDotHopTime*math.Pi)))
}

// drawLoadingIndicator は画面右下に"Loading"と、その後ろで順に跳ねる"..."を描く。
// 文字はゲーム本編と同じk8x12の字形だが、フォントファイルではなく
// loading_glyphs.go(tools/genloadingfontで生成)のドットパターンから描く。
// フォントの読み込みが終わっていない起動直後でも同じ見た目で出せるように。
// showProgressなら、文字の上に進捗バー(progressは0〜1)も描く。
func drawLoadingIndicator(screen *ebiten.Image, t float64, progress float64, showProgress bool) {
	textW := 0
	for _, r := range loadingLabel {
		textW += loadingGlyphWidth(r)
	}
	textW += loadingDotCount * loadingGlyphWidth(loadingDotRune)

	totalW := float64(textW * loadingScale)
	x0 := float64(gameWidth-loadingMarginX) - totalW
	y0 := float64(gameHeight-loadingMarginY) - loadingGlyphHeight*loadingScale

	x := x0
	drawGlyph := func(r rune, liftDots int) {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(loadingScale, loadingScale)
		op.GeoM.Translate(x, y0-float64(liftDots*loadingScale))
		screen.DrawImage(loadingGlyphImage(r), op)
		x += float64(loadingGlyphWidth(r) * loadingScale)
	}
	for _, r := range loadingLabel {
		drawGlyph(r, 0)
	}
	for i := range loadingDotCount {
		drawGlyph(loadingDotRune, loadingDotLift(t, i))
	}

	if showProgress {
		// 跳ねた点と重ならないよう、跳ねる高さの分だけ上に置く。
		barY := y0 - loadingDotHopHeight*loadingScale - loadingBarGapY - loadingBarH
		fillRect(screen, x0, barY, totalW, loadingBarH, loadingBarBgColor)
		fillRect(screen, x0, barY, totalW*math.Max(0, math.Min(progress, 1)), loadingBarH, loadingBarFgColor)
	}
}
