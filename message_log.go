package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// =========================================================
// 会話ログ画面（簡易版）＋ 会話中のキー操作ガイド
//
// ログは「話者名＋本文」を1つの四角いウィンドウにまとめたものを
// 画面横中央・縦は画面いっぱいに古い順で積み上げて表示する。
// 積み上げた合計の高さが画面をはみ出す場合は ↑/↓ でスクロールする。
// =========================================================

const (
	logColumnWidth  = 640.0 // ログの四角ウィンドウの幅（画像が読み込めなかった時の代わりの幅／スクロールバー位置の計算に使用）
	logBottomMargin = 12.0  // 画面下の余白

	logEntryGap  = 35.0 // ウィンドウ同士の「間隔」（一番上の画像より後は、この間隔で自動的に並ぶ）
	logTextLineH = 20.0

	logFallbackImageHeight = 110.0 // 画像が読み込めなかった時の代わりの高さ

	// --- 画像の位置：一番上（最初）の1件だけ、px値で直接指定する ---
	logImageStartX = 200.0
	logImageStartY = 30.0

	// --- 話者名・本文の位置：画像の左上を基準に、px値で直接オフセットする ---
	logNameOffsetX = 190.0
	logNameOffsetY = 4.0
	logTextOffsetX = 180.0
	logTextOffsetY = 30.0

	logVisibleCount  = 4    // 画面に同時に表示するログの件数
	logScrollStep    = 1.0  // ↑/↓ 1回あたりのスクロール量（ログ1件分）
	logScrollBarGap  = 16.0 // ログ列とスクロールバーの間隔
	logScrollBarMinH = 24.0 // スクロールバーのつまみの最小高さ

	// 会話中の右下キーガイド（オート・ログ）
	keyGuideMarginX     = 16.0
	keyGuideLineSpacing = 16.0

	// 選択中ログの赤グラデーションの速さ（大きいほど速く点滅）
	logSelectPulseSpeed = 3.0
)

// ログ画面の話者名・本文だけに使う色（他のUIテキストには影響しない）
var logTextColor = color.RGBA{0, 0, 0, 255}

// lerpColor はaとbの間をt(0〜1)で線形補間する
func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

// selectedLogPulseColor は経過時間から薄い赤〜濃い赤の間を往復する色を返す
func selectedLogPulseColor(elapsed float64) color.NRGBA {
	lightRed := color.NRGBA{255, 140, 140, 255}
	darkRed := color.NRGBA{190, 80, 80, 255}
	t := (math.Sin(elapsed*logSelectPulseSpeed) + 1) / 2 // 0〜1を往復
	return lerpColor(lightRed, darkRed, t)
}

// drawMessageKeyGuide は会話中、画面右下に短い操作ガイドを表示する。
func drawMessageKeyGuide(screen *ebiten.Image, game *Game) {
	lines := []string{"A：オート", "L：ログ"}

	face := game.FontFace(11)
	guideBottomY := skipHintMargin + skipHintRadius + 12
	baseY := float64(gameHeight) - guideBottomY - float64(len(lines)-1)*keyGuideLineSpacing

	for i, line := range lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gameWidth)-keyGuideMarginX, baseY+float64(i)*keyGuideLineSpacing)
		op.PrimaryAlign = text.AlignEnd
		op.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, line, face, op)
	}
}

// wrapLines は自動折り返しをせず、文字列中の改行(\n)の位置でそのまま分割する。
// 改行位置は書き手が入力時に決める（このシステムが調整するのは開始X座標のみ）。
func wrapLines(s string) []string {
	return strings.Split(s, "\n")
}

type logEntryLayout struct {
	cmd   EventCommand
	lines []string
}

// drawMessageLog は会話ログを画面中央の縦一列のウィンドウ群として描画する。
func drawMessageLog(screen *ebiten.Image, game *Game, log []EventCommand, scrollOffset float64, cursorIndex int) float64 {
	ebitenutil.DrawRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{0, 0, 0, 170})

	colX := (float64(gameWidth) - logColumnWidth) / 2

	titleFace := game.FontFace(30)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(logBottomMargin, logBottomMargin)
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "EVENT LOG", titleFace, titleOp)

	var imgW, imgH float64
	if game.LogEntryImg != nil {
		bounds := game.LogEntryImg.Bounds()
		imgW = float64(bounds.Dx())
		imgH = float64(bounds.Dy())
	} else {
		imgW = logColumnWidth
		imgH = logFallbackImageHeight
	}

	imgStartX := logImageStartX
	imgStartY := logImageStartY

	if len(log) == 0 {
		return 0
	}

	nameFace := game.FontFace(13)
	textFace := game.FontFace(13)

	maxTextLinesF := (imgH - logTextOffsetY) / logTextLineH
	maxTextLines := int(maxTextLinesF)
	if maxTextLines < 1 {
		maxTextLines = 1
	}

	entries := make([]logEntryLayout, 0, len(log))
	for _, cmd := range log {
		lines := wrapLines(cmd.Text)
		if len(lines) > maxTextLines {
			lines = lines[:maxTextLines]
			last := lines[len(lines)-1]
			lines[len(lines)-1] = last + "…"
		}
		entries = append(entries, logEntryLayout{cmd: cmd, lines: lines})
	}

	maxScroll := float64(len(entries) - logVisibleCount)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	endIndex := len(entries) - int(scrollOffset)
	startIndex := endIndex - logVisibleCount
	if startIndex < 0 {
		startIndex = 0
	}
	visibleEntries := entries[startIndex:endIndex]

	for slot, e := range visibleEntries {
		imgY := imgStartY + float64(slot)*(imgH+logEntryGap)
		imgX := imgStartX
		isSelected := startIndex+slot == cursorIndex

		if game.LogEntryImg != nil {
			imgOp := &ebiten.DrawImageOptions{}
			imgOp.GeoM.Translate(imgX, imgY)
			if isSelected {
				pulse := selectedLogPulseColor(game.TotalPlayTime)
				imgOp.ColorScale.Scale(
					float32(pulse.R)/255,
					float32(pulse.G)/255,
					float32(pulse.B)/255,
					1,
				)
			}
			screen.DrawImage(game.LogEntryImg, imgOp)
		} else {
			boxColor := uiColorPanelBg
			if isSelected {
				boxColor = selectedLogPulseColor(game.TotalPlayTime)
			}
			ebitenutil.DrawRect(screen, imgX, imgY, imgW, imgH, boxColor)
		}

		speakerLabel := e.cmd.Speaker
		if speakerLabel == "" {
			speakerLabel = "・・・"
		}
		nameX := imgX + logNameOffsetX
		nameY := imgY + logNameOffsetY
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(nameX, nameY)
		nameOp.ColorScale.ScaleWithColor(logTextColor)
		text.Draw(screen, speakerLabel, nameFace, nameOp)

		textX := imgX + logTextOffsetX
		textY := imgY + logTextOffsetY
		for li, line := range e.lines {
			lineOp := &text.DrawOptions{}
			lineOp.GeoM.Translate(textX, textY+float64(li)*logTextLineH)
			lineOp.ColorScale.ScaleWithColor(logTextColor)
			text.Draw(screen, line, textFace, lineOp)
		}
	}

	if maxScroll > 0 {
		barX := colX + logColumnWidth + logScrollBarGap
		trackHeight := float64(logVisibleCount)*imgH + float64(logVisibleCount-1)*logEntryGap
		trackY := (float64(gameHeight) - trackHeight) / 2
		ebitenutil.DrawRect(screen, barX, trackY, scrollBarTrackWidth, trackHeight, scrollBarTrackColor)

		thumbH := trackHeight * (float64(logVisibleCount) / float64(len(entries)))
		if thumbH < logScrollBarMinH {
			thumbH = logScrollBarMinH
		}
		moveRange := trackHeight - thumbH
		if moveRange < 0 {
			moveRange = 0
		}
		ratio := 1 - scrollOffset/maxScroll
		thumbY := trackY + ratio*moveRange

		ebitenutil.DrawRect(screen, barX, thumbY, scrollBarCursorWidth, thumbH, scrollBarCursorColor)
	}

	return scrollOffset
}
