package main

// menu_draw_status.go: ステータス画面・セーブ/ロードスロット一覧・確認ダイアログの描画
import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	slotCardStartX = 300.0
	slotCardStartY = 55.0
	slotCardGapY   = 0.0

	slotNumOffsetX = 40.0
	slotNumYRatio  = 0.5
	slotNumSize    = 26.0

	slotThumbOffsetX = 334.0
	slotThumbYRatio  = 0.51
	slotThumbW       = 108.0
	slotThumbAspect  = 16.0 / 9.0
	slotThumbH       = slotThumbW / slotThumbAspect

	slotTextOffsetX  = 90.0
	slotTextStartY   = 25.0
	slotTextLineGap  = 16.0
	slotTextFontSize = 13.0

	slotsPerPageView = 4

	scrollBarX    = 800.0
	scrollBarTopY = 40.0
)

const (
	// 共通フォントサイズ（日本語ラベル／数値／英語ラベルの3種で管理）
	statusFontSizeJP  = 20.0
	statusFontSizeNum = 25.0
	statusFontSizeEN  = 25.0

	// 顔・名前（左上ボックス）
	statusFaceBoxX     = 250.0
	statusFaceBoxY     = 8.0
	statusFaceBoxW     = 195.0
	statusFaceBoxH     = 352.0
	statusFaceR        = 78.0
	statusFaceCYRat    = 0.31
	statusNameYRat     = 0.60
	statusNameFontSize = 18.0

	// 左側ブロック（Lv/EXP/HP/MP）共通のX位置
	statusLeftLabelX = 220.0
	statusLeftValueX = 500.0
	statusLeftLineX  = 220.0
	statusLeftLineW  = 300.0

	// Y位置だけ行ごとに個別指定
	statusLvY  = 280.0
	statusExpY = 330.0
	statusHpY  = 390.0
	statusMpY  = 440.0

	// 右ボックス：6ステータス共通のX位置
	statusRightLabelX = 570.0
	statusRightValueX = 900.0
	statusRightLineX  = 570.0
	statusRightLineW  = 350.0

	statusRightBoxY = 50.0
	statusRightRowH = 55.0

	// 右下：パーティアイコン
	statusPartyIconY      = 440.0
	statusPartyIconGap    = 90.0
	statusPartyIconStartX = 610.0
	// フォールバック（画像未ロード時の円枠）用の半径
	statusPartyIconFallbackR = 30.0

	// 下線とテキストの間隔（フォントサイズに応じて調整。文字とかぶらないよう十分な余白を持たせる）
	statusLeftLineOffsetY  = statusFontSizeNum + 8.0 // 左側ブロック用（Lv/EXP/HP/MP、数値フォント基準）
	statusRightLineOffsetY = statusFontSizeNum + 8.0 // 右側ブロック用（6ステータス、数値フォント基準）
)

var statusLineColor = uiColorText

type statusStatRow struct {
	label string
	y     float64
}

var statusStatRows = []statusStatRow{
	{"物理攻撃力", statusRightBoxY + 0*statusRightRowH},
	{"魔法攻撃力", statusRightBoxY + 1*statusRightRowH},
	{"物理防御力", statusRightBoxY + 2*statusRightRowH},
	{"魔法防御力", statusRightBoxY + 3*statusRightRowH},
	{"すばやさ", statusRightBoxY + 4*statusRightRowH},
	{"運", statusRightBoxY + 5*statusRightRowH},
}

// drawStatusRow はラベルと値を同じ書式（フォントサイズ・色）で描画し、下に区切り線を引く。
// x, y はラベルの描画開始位置。w は下線の幅。
// labelX: ラベルの描画開始X座標
// valueX: 値の描画終了X座標（右揃えの基準点）
// lineX, lineW: 下線の開始X座標と幅（見た目の区切り線用。labelX〜valueXの範囲と別に指定可能）
func drawStatusRow(screen *ebiten.Image, g *Game, label, value string, labelX, valueX, y, lineX, lineW, labelFontSize, valueFontSize, lineOffsetY float64) {
	labelFace := g.FontFace(labelFontSize)
	valueFace := g.FontFace(valueFontSize)

	labelOp := &text.DrawOptions{}
	labelOp.GeoM.Translate(labelX, y)
	labelOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, label, labelFace, labelOp)

	valOp := &text.DrawOptions{}
	valOp.GeoM.Translate(valueX, y)
	valOp.PrimaryAlign = text.AlignEnd
	valOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, value, valueFace, valOp)

	ebitenutil.DrawRect(screen, lineX, y+lineOffsetY, lineW, 1, statusLineColor)
}

func (m *MenuScene) drawStatusScreen(screen *ebiten.Image) {
	g := m.game
	i := m.statusCharIndex

	// ── 顔グラフィック＋名前 ──
	faceCX := statusFaceBoxX + statusFaceBoxW/2
	faceCY := statusFaceBoxY + statusFaceBoxH*statusFaceCYRat

	if i < len(g.PartyIconImgs) && g.PartyIconImgs[i] != nil {
		img := g.PartyIconImgs[i]
		iw := float64(img.Bounds().Dx())
		ih := float64(img.Bounds().Dy())
		const faceIconScale = 2.0
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(faceIconScale, faceIconScale)
		op.GeoM.Translate(faceCX-iw*faceIconScale/2, faceCY-ih*faceIconScale/2)
		screen.DrawImage(img, op)
	} else {
		var ring vector.Path
		const segs = 48
		for s := 0; s <= segs; s++ {
			ang := float64(s) / segs * 2 * math.Pi
			x := faceCX + statusFaceR*math.Cos(ang)
			y := faceCY + statusFaceR*math.Sin(ang)
			if s == 0 {
				ring.MoveTo(float32(x), float32(y))
			} else {
				ring.LineTo(float32(x), float32(y))
			}
		}
		strokeOpts := &vector.StrokeOptions{Width: 3}
		drawOpts := &vector.DrawPathOptions{AntiAlias: true}
		drawOpts.ColorScale.ScaleWithColor(uiColorText)
		vector.StrokePath(screen, &ring, strokeOpts, drawOpts)
	}

	nameOp := &text.DrawOptions{}
	nameOp.GeoM.Translate(faceCX, statusFaceBoxY+statusFaceBoxH*statusNameYRat)
	nameOp.PrimaryAlign = text.AlignCenter
	nameOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, PlayerNames[i], g.FontFace(statusNameFontSize), nameOp)

	// ── Lv・EXP ──
	drawStatusRow(screen, g, "Lv", fmt.Sprintf("%d/%d", g.PlayerLv[i], maxPlayerLevel),
		statusLeftLabelX, statusLeftValueX, statusLvY, statusLeftLineX, statusLeftLineW,
		statusFontSizeEN, statusFontSizeNum, statusLeftLineOffsetY)

	drawStatusRow(screen, g, "EXP", fmt.Sprintf("%d/%d", g.PlayerEXP[i], g.PlayerNextEXP[i]),
		statusLeftLabelX, statusLeftValueX, statusExpY, statusLeftLineX, statusLeftLineW,
		statusFontSizeEN, statusFontSizeNum, statusLeftLineOffsetY)

	drawStatusRow(screen, g, "HP", fmt.Sprintf("%d", g.PlayerMaxHP[i]),
		statusLeftLabelX, statusLeftValueX, statusHpY, statusLeftLineX, statusLeftLineW,
		statusFontSizeEN, statusFontSizeNum, statusLeftLineOffsetY)

	drawStatusRow(screen, g, "MP", fmt.Sprintf("%d", g.PlayerMaxMP[i]),
		statusLeftLabelX, statusLeftValueX, statusMpY, statusLeftLineX, statusLeftLineW,
		statusFontSizeEN, statusFontSizeNum, statusLeftLineOffsetY)

	statusValues := []int{
		g.PlayerAtk[i],
		g.PlayerMagicAtk[i],
		g.PlayerDef[i],
		g.PlayerMagicDef[i],
		g.PlayerSpd[i],
		g.PlayerLuck[i],
	}
	for idx, row := range statusStatRows {
		drawStatusRow(screen, g, row.label, fmt.Sprintf("%d", statusValues[idx]),
			statusRightLabelX, statusRightValueX, row.y, statusRightLineX, statusRightLineW,
			statusFontSizeJP, statusFontSizeNum, statusRightLineOffsetY)
	}

	for idx := 0; idx < partySize; idx++ {
		cx := statusPartyIconStartX + float64(idx)*statusPartyIconGap
		cy := statusPartyIconY

		isSelected := idx == m.statusCharIndex

		col := uiColorText
		width := float32(2)
		if isSelected {
			col = uiColorSelect
			width = 3
		}

		if idx < len(g.PartyIconImgs) && g.PartyIconImgs[idx] != nil {
			img := g.PartyIconImgs[idx]
			iw := float64(img.Bounds().Dx())
			ih := float64(img.Bounds().Dy())
			op := &ebiten.DrawImageOptions{}
			// 原寸のまま描画（スケールしない）
			op.GeoM.Translate(cx-iw/2, cy-ih/2)
			if !isSelected {
				op.ColorScale.Scale(0.45, 0.45, 0.45, 1.0)
			}
			screen.DrawImage(img, op)
		} else {
			r := statusPartyIconFallbackR
			var ring vector.Path
			const segs = 40
			for s := 0; s <= segs; s++ {
				ang := float64(s) / segs * 2 * math.Pi
				x := cx + r*math.Cos(ang)
				y := cy + r*math.Sin(ang)
				if s == 0 {
					ring.MoveTo(float32(x), float32(y))
				} else {
					ring.LineTo(float32(x), float32(y))
				}
			}
			strokeOpts := &vector.StrokeOptions{Width: width}
			drawOpts := &vector.DrawPathOptions{AntiAlias: true}
			drawOpts.ColorScale.ScaleWithColor(col)
			vector.StrokePath(screen, &ring, strokeOpts, drawOpts)
		}
	}

	// ── 列の両端に固定表示する矢印（常にuiColorText） ──
	// 画像があればその実際の幅（半分）を、無ければフォールバック半径を端の基準にする
	const arrowGap = 5.0

	firstCX := statusPartyIconStartX
	firstHalfW := statusPartyIconFallbackR
	if len(g.PartyIconImgs) > 0 && g.PartyIconImgs[0] != nil {
		firstHalfW = float64(g.PartyIconImgs[0].Bounds().Dx()) / 2
	}

	lastIdx := partySize - 1
	lastCX := statusPartyIconStartX + float64(lastIdx)*statusPartyIconGap
	lastHalfW := statusPartyIconFallbackR
	if lastIdx < len(g.PartyIconImgs) && g.PartyIconImgs[lastIdx] != nil {
		lastHalfW = float64(g.PartyIconImgs[lastIdx].Bounds().Dx()) / 2
	}

	leftOp := &text.DrawOptions{}
	leftOp.GeoM.Translate(firstCX-firstHalfW-arrowGap, statusPartyIconY)
	leftOp.SecondaryAlign = text.AlignCenter
	leftOp.PrimaryAlign = text.AlignEnd // ← 追加：文字の右端(アイコンに近い側)をgap位置に合わせる
	leftOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "◀", g.FontFace(18), leftOp)

	rightOp := &text.DrawOptions{}
	rightOp.GeoM.Translate(lastCX+lastHalfW+arrowGap, statusPartyIconY)
	rightOp.SecondaryAlign = text.AlignCenter
	// PrimaryAlign はデフォルト(Start)のままでOK：文字の左端(アイコンに近い側)がgap位置になる
	rightOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "▶", g.FontFace(18), rightOp)
}

func (m *MenuScene) lookupCharaImage(i int) *ebiten.Image {
	return nil
}

func drawSlotList(screen *ebiten.Image, g *Game, selectedIndex int, slotData [maxSaveSlots]*SaveData, slotThumbs [maxSaveSlots]*ebiten.Image, saveMode bool, selectedRow int, cardStartX, cardStartY float64) {

	if selectedRow < 0 {
		selectedRow = 0
	} else if selectedRow >= slotsPerPageView {
		selectedRow = slotsPerPageView - 1
	}

	maxScrollTop := maxSaveSlots - slotsPerPageView
	scrollTop := selectedIndex - selectedRow
	if scrollTop < 0 {
		scrollTop = 0
	} else if scrollTop > maxScrollTop {
		scrollTop = maxScrollTop
	}

	cardH := 0.0
	if g.SaveThumbFrameImg != nil {
		cardH = float64(g.SaveThumbFrameImg.Bounds().Dy())
	} else if g.SaveThumbFrameSelImg != nil {
		cardH = float64(g.SaveThumbFrameSelImg.Bounds().Dy())
	}

	for dispIdx := 0; dispIdx < slotsPerPageView; dispIdx++ {
		i := scrollTop + dispIdx
		if i >= maxSaveSlots {
			break
		}

		isSelected := i == selectedIndex

		cardX := cardStartX
		cardY := cardStartY + float64(dispIdx)*(cardH+slotCardGapY)

		if g.SaveThumbFrameImg != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(cardX, cardY)
			screen.DrawImage(g.SaveThumbFrameImg, op)
		}
		if isSelected && g.SaveThumbFrameSelImg != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(cardX, cardY)
			screen.DrawImage(g.SaveThumbFrameSelImg, op)
		}

		numOp := &text.DrawOptions{}
		numOp.GeoM.Translate(cardX+slotNumOffsetX, cardY+cardH*slotNumYRatio)
		numOp.SecondaryAlign = text.AlignCenter
		numOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, fmt.Sprintf("%02d", i+1), g.FontFace(slotNumSize), numOp)

		thumbX := cardX + slotThumbOffsetX
		thumbY := cardY + cardH*slotThumbYRatio - slotThumbH/2
		if slotThumbs[i] != nil {
			drawImageFitAspect(screen, slotThumbs[i], thumbX, thumbY, slotThumbW, slotThumbH)
		}

		d := slotData[i]
		tx := cardX + slotTextOffsetX
		ty := cardY + slotTextStartY

		if d != nil {
			locOp := &text.DrawOptions{}
			locOp.GeoM.Translate(tx, ty)
			locOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, d.LocationName, g.FontFace(slotTextFontSize), locOp)

			lvOp := &text.DrawOptions{}
			lvOp.GeoM.Translate(tx, ty+slotTextLineGap)
			lvOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, fmt.Sprintf("Lv %d", d.PlayerLv[0]), g.FontFace(slotTextFontSize), lvOp)

			timeOp := &text.DrawOptions{}
			timeOp.GeoM.Translate(tx, ty+slotTextLineGap*2)
			timeOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, FormatPlayTime(d.PlayTime), g.FontFace(slotTextFontSize), timeOp)

			dateOp := &text.DrawOptions{}
			dateOp.GeoM.Translate(tx, ty+slotTextLineGap*3)
			dateOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, d.SavedAt, g.FontFace(slotTextFontSize), dateOp)
		}
	}

	barX := scrollBarX + (cardStartX - slotCardStartX)
	barTop := scrollBarTopY + (cardStartY - slotCardStartY)
	drawSaveScrollBar(screen, g, selectedIndex, barX, barTop)
}

// drawConfirmDialog はセーブ/ロード/タイトルに戻る/終了などの
// 「はい・いいえ」確認ダイアログを共通の見た目で描画する。
// horizontalOffset だけ呼び出し側で調整可能（縦位置・画像・文字サイズは固定）。
// showChoices が false の場合は、選択肢を表示せずメッセージのみ表示する（デフォルトは true）。
func drawConfirmDialog(screen *ebiten.Image, game *Game, message string, selectedIndex int, horizontalOffset float64, showChoices ...bool) {
	img := game.SaveConfirmBgImg
	if img == nil {
		return
	}

	// デフォルト値の処理
	displayChoices := true
	if len(showChoices) > 0 {
		displayChoices = showChoices[0]
	}

	winW := float64(img.Bounds().Dx()) * confirmPanelScale
	winH := float64(img.Bounds().Dy()) * confirmPanelScale

	winX := float64(gameWidth)/2 - winW/2 + horizontalOffset
	winY := float64(gameHeight)/2 - winH/2 + confirmImageOffsetY

	drawImageFit(screen, img, winX, winY, winW, winH)

	face := game.FontFace(15)
	lineOp := &text.DrawOptions{}
	lineOp.GeoM.Translate(winX+winW/2+confirmTextOffsetX, winY+confirmTextOffsetY)
	lineOp.PrimaryAlign = text.AlignCenter
	lineOp.LineSpacing = face.Metrics().HAscent + face.Metrics().HDescent + 4
	lineOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, message, face, lineOp)

	if displayChoices {
		choices := []string{"はい", "いいえ"}
		for j, choice := range choices {
			choiceOp := &text.DrawOptions{}
			choiceOp.GeoM.Translate(
				winX+winW/2+confirmChoiceOffsetX,
				winY+confirmChoiceStartY+float64(j)*confirmChoiceGap,
			)
			choiceOp.PrimaryAlign = text.AlignCenter
			if j == selectedIndex {
				choiceOp.ColorScale.ScaleWithColor(uiColorSelect)
				text.Draw(screen, "▶ "+choice, game.FontFace(15), choiceOp)
			} else {
				choiceOp.ColorScale.ScaleWithColor(uiColorText)
				text.Draw(screen, "  "+choice, game.FontFace(15), choiceOp)
			}
		}
	}
}

// スクロールバーの見た目（元は単色の画像だったため、コード側の単色描画に置き換え）
const (
	scrollBarTrackWidth  = 3.0
	scrollBarTrackHeight = 456.0
	scrollBarCursorWidth = 3.0
	scrollBarCursorH     = 129.0
)

var (
	scrollBarTrackColor  = color.NRGBA{255, 255, 255, 120} // 元画像相当：白・半透明
	scrollBarCursorColor = color.NRGBA{255, 0, 12, 235}    // 元画像相当：赤・不透明気味
)

func drawSaveScrollBar(screen *ebiten.Image, g *Game, selectedIndex int, scrollBarX, barTop float64) {
	ebitenutil.DrawRect(screen, scrollBarX, barTop, scrollBarTrackWidth, scrollBarTrackHeight, scrollBarTrackColor)

	if maxSaveSlots > 0 {
		moveRange := scrollBarTrackHeight - scrollBarCursorH
		if moveRange < 0 {
			moveRange = 0
		}

		ratio := 0.0
		if maxSaveSlots > 1 {
			ratio = float64(selectedIndex) / float64(maxSaveSlots-1)
		}
		cursorY := barTop + ratio*moveRange

		ebitenutil.DrawRect(screen, scrollBarX, cursorY, scrollBarCursorWidth, scrollBarCursorH, scrollBarCursorColor)
	}
}

func drawImageFit(screen *ebiten.Image, img *ebiten.Image, x, y, w, h float64) {
	if img == nil {
		return
	}
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	if iw == 0 || ih == 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/iw, h/ih)
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func drawImageFitAspect(screen *ebiten.Image, img *ebiten.Image, x, y, w, h float64) {
	if img == nil {
		return
	}
	iw := float64(img.Bounds().Dx())
	ih := float64(img.Bounds().Dy())
	if iw == 0 || ih == 0 {
		return
	}
	scale := w / iw
	if sh := h / ih; sh < scale {
		scale = sh
	}
	drawW := iw * scale
	drawH := ih * scale
	offsetX := x + (w-drawW)/2
	offsetY := y + (h-drawH)/2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(img, op)
}
