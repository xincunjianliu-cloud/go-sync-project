package main

import (
	"image/color"

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

	logEntryGap  = 50.0 // ウィンドウ同士の「間隔」（一番上の画像より後は、この間隔で自動的に並ぶ）
	logTextLineH = 20.0

	logFallbackImageHeight = 110.0 // 画像が読み込めなかった時の代わりの高さ

	// --- 画像の位置：一番上（最初）の1件だけ、px値で直接指定する ---
	// ここで指定するのは「一番上の画像」の左上座標。2件目以降は
	// この位置から (画像の高さ + logEntryGap) ずつ下にずれて自動的に並ぶ。
	logImageStartX = 200.0 // 一番上の画像の左上X座標（← ここを変えると横位置調整）
	logImageStartY = 50.0  // 一番上の画像の左上Y座標（← ここを変えると縦位置調整）

	// --- 話者名・本文の位置：画像の左上を基準に、px値で直接オフセットする ---
	logNameOffsetX = 180.0 // 画像左上からの話者名の横オフセット（← ここを変えると横位置調整）
	logNameOffsetY = -15.0 // 画像左上からの話者名の縦オフセット（← ここを変えると縦位置調整）
	logTextOffsetX = 180.0 // 画像左上からの本文の横オフセット（← ここを変えると横位置調整）
	logTextOffsetY = 10.0  // 画像左上からの本文の縦オフセット（← ここを変えると縦位置調整）

	logVisibleCount  = 4    // 画面に同時に表示するログの件数
	logScrollStep    = 1.0  // ↑/↓ 1回あたりのスクロール量（ログ1件分）
	logScrollBarGap  = 16.0 // ログ列とスクロールバーの間隔
	logScrollBarMinH = 24.0 // スクロールバーのつまみの最小高さ

	// 会話中の右下キーガイド（オート・ログ）
	keyGuideMarginX     = 16.0
	keyGuideLineSpacing = 16.0
)

// drawMessageKeyGuide は会話中、画面右下に短い操作ガイドを表示する。
// スキップは丸いプログレス表示（drawSkipHint）で示すので、ここではオートとログのみ。
// 丸の真上に来るよう配置する。
func drawMessageKeyGuide(screen *ebiten.Image, game *Game) {
	lines := []string{"A：オート", "L：ログ"}

	face := game.FontFace(11)
	// スキップの丸(半径skipHintRadius、中心がgameHeight-skipHintMargin)の
	// すぐ上に収まるよう、下端の基準位置を丸の上端より少し高くする
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

// wrapLines は与えられた文字列を maxWidth に収まるよう改行位置で分割する。
// message_system.go のタイプライター折り返しと同じ考え方の簡易版。
func wrapLines(s string, face *text.GoTextFace, maxWidth float64) []string {
	var lines []string
	var currentLine string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
			continue
		}
		testLine := currentLine + string(r)
		if text.Advance(testLine, face) > maxWidth {
			lines = append(lines, currentLine)
			currentLine = string(r)
		} else {
			currentLine = testLine
		}
	}
	if currentLine != "" || len(lines) == 0 {
		lines = append(lines, currentLine)
	}
	return lines
}

type logEntryLayout struct {
	cmd   EventCommand
	lines []string
}

// drawMessageLog は会話ログを画面中央の縦一列のウィンドウ群として描画する。
// scrollOffset は「一番下（最新）を基準に、どれだけ上にスクロールしたか」(px)。
// クランプ後の scrollOffset を返すので、呼び出し側はこれを保存し直すこと。
func drawMessageLog(screen *ebiten.Image, game *Game, log []EventCommand, scrollOffset float64, cursorIndex int) float64 {
	// 背景を少し暗くする
	ebitenutil.DrawRect(screen, 0, 0, float64(gameWidth), float64(gameHeight), color.NRGBA{0, 0, 0, 170})

	colX := (float64(gameWidth) - logColumnWidth) / 2

	// --- 左上に「EVENT LOG」とタイトル表示 ---
	titleFace := game.FontFace(30)
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(logBottomMargin, logBottomMargin)
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "EVENT LOG", titleFace, titleOp)

	// 画像（1件分）のサイズ。原寸のまま使うので、実ファイルのサイズをそのまま使う。
	var imgW, imgH float64
	if game.LogEntryImg != nil {
		bounds := game.LogEntryImg.Bounds()
		imgW = float64(bounds.Dx())
		imgH = float64(bounds.Dy())
	} else {
		imgW = logColumnWidth
		imgH = logFallbackImageHeight
	}

	// 一番上（最初）の画像の左上座標。px値をそのまま使う。
	imgStartX := logImageStartX
	imgStartY := logImageStartY

	if len(log) == 0 {
		return 0
	}

	nameFace := game.FontFace(13)
	textFace := game.FontFace(13)
	innerWidth := imgW - logTextOffsetX*2

	// ウィンドウ内に収まる最大行数（はみ出す分は切り詰める）
	maxTextLinesF := (imgH - logTextOffsetY) / logTextLineH
	maxTextLines := int(maxTextLinesF)
	if maxTextLines < 1 {
		maxTextLines = 1
	}

	entries := make([]logEntryLayout, 0, len(log))
	for _, cmd := range log {
		lines := wrapLines(cmd.Text, textFace, innerWidth)
		if len(lines) > maxTextLines {
			lines = lines[:maxTextLines]
			last := lines[len(lines)-1]
			lines[len(lines)-1] = last + "…"
		}
		entries = append(entries, logEntryLayout{cmd: cmd, lines: lines})
	}

	// 画面には常に logVisibleCount 件だけを表示する。scrollOffset は
	// 「最新（一番下）から何件分さかのぼったか」という“件数”で表し、
	// 各件は常に固定スロット（imgStartY を先頭に imgH+logEntryGap 間隔）に
	// 描画するので、スクロールしても表示中のウィンドウの位置がずれない。
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

	// 一番下（最新）を末尾スロットに置き、scrollOffset件分だけ古い方へずらす。
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
			// 拡大縮小せず原寸のまま描画する。選択中は赤みを乗せる。
			imgOp := &ebiten.DrawImageOptions{}
			imgOp.GeoM.Translate(imgX, imgY)
			if isSelected {
				imgOp.ColorScale.Scale(1, 0.35, 0.35, 1)
			}
			screen.DrawImage(game.LogEntryImg, imgOp)
		} else {
			boxColor := uiColorPanelBg
			if isSelected {
				boxColor = color.NRGBA{200, 40, 40, 255}
			}
			ebitenutil.DrawRect(screen, imgX, imgY, imgW, imgH, boxColor)
		}

		// 話者名・本文は「画像の左上(imgX, imgY)」を基準に、pxオフセット(logNameOffsetX/Y等)で配置する。
		speakerLabel := e.cmd.Speaker
		if speakerLabel == "" {
			speakerLabel = "・・・"
		}
		nameX := imgX + logNameOffsetX
		nameY := imgY + logNameOffsetY
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(nameX, nameY)
		nameOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, speakerLabel, nameFace, nameOp)

		textX := imgX + logTextOffsetX
		textY := imgY + logTextOffsetY
		for li, line := range e.lines {
			lineOp := &text.DrawOptions{}
			lineOp.GeoM.Translate(textX, textY+float64(li)*logTextLineH)
			lineOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, line, textFace, lineOp)
		}
	}

	// --- スクロールバー（ログ4件分の高さに合わせ、画面縦方向の中央に来るように配置） ---
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
		// scrollOffset=0（最新/一番下を表示中）のときはつまみを一番下に、
		// scrollOffset=maxScroll（一番古い/一番上を表示中）のときは一番上にする。
		ratio := 1 - scrollOffset/maxScroll
		thumbY := trackY + ratio*moveRange

		ebitenutil.DrawRect(screen, barX, thumbY, scrollBarCursorWidth, thumbH, scrollBarCursorColor)
	}

	return scrollOffset
}
