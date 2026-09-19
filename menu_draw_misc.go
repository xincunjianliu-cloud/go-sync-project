package main

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
	minimapX = 500.0
	minimapY = 40.0
	minimapW = 400.0
	minimapH = 460.0
)

const (
	menuDescOffsetX  float64 = 20
	menuDescOffsetY  float64 = 30
	menuDescFontSize float64 = 17.0
)

func (m *MenuScene) drawMinimap(screen *ebiten.Image) {
	field, ok := m.backScene.(*FieldScene)
	if !ok {
		return
	}

	tm := field.tileMap
	if tm.Width == 0 || tm.Height == 0 {
		return
	}

	mapPixW := float64(tm.Width * tm.TileWidth)
	mapPixH := float64(tm.Height * tm.TileHeight)

	scaleX := minimapW / mapPixW
	scaleY := minimapH / mapPixH
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	drawW := mapPixW * scale
	drawH := mapPixH * scale

	offsetX := minimapX + (minimapW-drawW)/2
	offsetY := minimapY + (minimapH-drawH)/2

	for _, layer := range tm.Layers {
		if layer.Type != "tilelayer" {
			continue
		}
		isWall := layer.Name == wallTileLayerName
		isFloor := layer.Name == floorTileLayerName

		for i, id := range layer.Data {
			if id == 0 {
				continue
			}
			tx := float64(i%tm.Width) * float64(tm.TileWidth)
			ty := float64(i/tm.Width) * float64(tm.TileHeight)

			px := offsetX + tx*scale
			py := offsetY + ty*scale
			pw := float64(tm.TileWidth) * scale
			ph := float64(tm.TileHeight) * scale

			var c color.RGBA
			if isWall {
				c = color.RGBA{60, 70, 90, 255}
			} else if isFloor {
				c = color.RGBA{80, 90, 110, 255}
			} else {
				c = color.RGBA{70, 80, 100, 200}
			}
			ebitenutil.DrawRect(screen, px, py, pw, ph, c)
		}
	}

	ebitenutil.DrawRect(screen, offsetX, offsetY, drawW, 1, color.RGBA{80, 100, 140, 255})
	ebitenutil.DrawRect(screen, offsetX, offsetY+drawH-1, drawW, 1, color.RGBA{80, 100, 140, 255})
	ebitenutil.DrawRect(screen, offsetX, offsetY, 1, drawH, color.RGBA{80, 100, 140, 255})
	ebitenutil.DrawRect(screen, offsetX+drawW-1, offsetY, 1, drawH, color.RGBA{80, 100, 140, 255})

	plx := offsetX + field.px*scale
	ply := offsetY + field.py*scale

	icon := m.game.MinimapPlayerIconImg
	iw := float64(icon.Bounds().Dx())
	ih := float64(icon.Bounds().Dy())

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-iw/2, -ih/2)
	op.GeoM.Rotate(dirToAngle(field.dir))
	op.GeoM.Translate(plx, ply)
	screen.DrawImage(icon, op)

	if loc, ok := m.game.CurrentObjectiveLocation(); ok {
		if loc.MapPath == field.currentMap {
			drawMinimapObjectiveIcon(screen, m.game, offsetX+loc.X*scale, offsetY+loc.Y*scale)
		} else if field.hasObjectiveDoor {
			drawMinimapObjectiveIcon(screen, m.game, offsetX+field.objectiveDoorX*scale, offsetY+field.objectiveDoorY*scale)
		}
	}
}

const (
	volumeGroupX float64 = 320
	volumePanelY float64 = 60

	volumeBarW float64 = 300
	volumeBarH float64 = 30

	volumeRowGapY float64 = volumeBarH + 12

	volumeLabelBarGap   float64 = 110
	volumeBarPercentGap float64 = 30

	// volumeResetGapY はシステム設定の最後の行からリセットボタンまでの間隔。
	// 画面右下に出る説明ヒント(menuDescOffsetY=30)と重ならない範囲に収めてある。
	volumeResetGapY float64 = 40

	volumeTrackH   float64 = 6
	volumeKnobR    float64 = 6
	volumeKnobRSel float64 = 8

	// volumePercentColW は"100%"表示分の余白。区切り線やパネル枠の幅計算で
	// パーセント表示まで含めた実際のコンテンツ幅に合わせるために使う。
	volumePercentColW float64 = 70

	// volumeContentW は音量行の左端(ラベル)から右端(パーセント表示)までの
	// 実コンテンツ幅。区切り線・パネル当たり判定の両方をこれに合わせることで、
	// 見た目の枠とタップ領域がずれないようにする。
	volumeContentW float64 = volumeLabelBarGap + volumeBarW + volumeBarPercentGap + volumePercentColW

	// optionHeaderFontSize は「音量設定」「表示設定」見出しの文字サイズ。
	optionHeaderFontSize float64 = 23

	// optionItemFontSize は見出し以外（各行のラベル・値・矢印・パーセント表示・
	// プレビュー文・リセットボタン）すべてに共通で使う文字サイズ。
	optionItemFontSize float64 = 23
)

// drawVolumeRow はBGM/効果音/全体音量の各行を同じ見た目で描画する共通処理。
// バー本体は細いトラック線にし、現在値はつまみの丸で示す。
// バー・つまみの色は選択中かどうかに関わらず常に白系(uiColorText)のまま。
func (m *MenuScene) drawVolumeRow(screen *ebiten.Image, barY float64, label string, volume float64, selected bool) {
	labelX := volumeGroupX
	barX := labelX + volumeLabelBarGap
	midY := barY + volumeBarH/2

	knobCol := uiColorText

	trackY := midY - volumeTrackH/2
	vector.DrawFilledRect(screen, float32(barX), float32(trackY), float32(volumeBarW), float32(volumeTrackH), color.RGBA{30, 30, 40, 255}, true)
	fillW := volumeBarW * volume
	if fillW > 0 {
		vector.DrawFilledRect(screen, float32(barX), float32(trackY), float32(fillW), float32(volumeTrackH), knobCol, true)
	}

	knobR := volumeKnobR
	if selected {
		knobR = volumeKnobRSel
	}
	knobX := barX + fillW
	vector.DrawFilledCircle(screen, float32(knobX), float32(midY), float32(knobR), knobCol, true)

	labelCol := uiColorText
	if selected {
		labelCol = uiColorSelect
	}
	// ラベル列(labelX〜barX)の中央に文字を揃える。矢印はラベルの左に
	// 添える形にして、選択の有無で文字位置がずれないようにする。
	labelCenterX := labelX + volumeLabelBarGap/2
	labelFace := m.game.FontFace(optionItemFontSize)
	labelW, _ := text.Measure(label, labelFace, 0)
	if selected {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(labelCenterX-labelW/2-6, midY)
		arrowOp.PrimaryAlign = text.AlignEnd
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(labelCol)
		text.Draw(screen, "▶", labelFace, arrowOp)
	}
	labelOp := &text.DrawOptions{}
	labelOp.GeoM.Translate(labelCenterX, midY)
	labelOp.PrimaryAlign = text.AlignCenter
	labelOp.SecondaryAlign = text.AlignCenter
	labelOp.ColorScale.ScaleWithColor(labelCol)
	text.Draw(screen, label, labelFace, labelOp)

	percentOp := &text.DrawOptions{}
	percentOp.GeoM.Translate(barX+volumeBarW+volumeBarPercentGap, midY)
	percentOp.SecondaryAlign = text.AlignCenter
	percentOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("%d%%", int(volume*100+0.5)), labelFace, percentOp)
}

func (m *MenuScene) drawVolumePanel(screen *ebiten.Image) {
	var bgmVol, seVol, masterVol float64
	if m.game.Audio != nil {
		bgmVol = m.game.Audio.volume
		seVol = m.game.Audio.seVolume
		masterVol = m.game.Audio.masterVolume
	}

	onOptionRow := func(idx int) bool { return m.menuState == menuStateOption && m.optionIndex == idx }
	onSpeedRow := onOptionRow(optionIdxMessageSpeed)
	adjustingSpeed := onSpeedRow
	onResetRow := m.optionIndex == optionIdxReset && (m.menuState == menuStateOption ||
		m.menuState == menuStateOptionResetConfirm || m.menuState == menuStateOptionResetDone)
	labelX := volumeGroupX
	lineW := volumeContentW
	lineCol := uiColorText

	headerY := volumePanelY - 34
	headerOp := &text.DrawOptions{}
	headerOp.GeoM.Translate(labelX, headerY)
	headerOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "音量設定", m.game.FontFace(optionHeaderFontSize), headerOp)

	lineY1 := headerY + 24
	ebitenutil.DrawRect(screen, labelX, lineY1, lineW, 1, lineCol)

	masterBarY, bgmBarY, seBarY, displayRowY, speedRowY, descRowY, sysHeaderY, cursorRowY, resetY := optionRowPositions()

	m.drawVolumeRow(screen, masterBarY, "全体", masterVol, onOptionRow(optionIdxMaster))
	m.drawVolumeRow(screen, bgmBarY, "BGM", bgmVol, onOptionRow(optionIdxBGM))
	m.drawVolumeRow(screen, seBarY, "効果音", seVol, onOptionRow(optionIdxSE))

	dispHeaderY := seBarY + volumeBarH + 30
	dispHeaderOp := &text.DrawOptions{}
	dispHeaderOp.GeoM.Translate(labelX, dispHeaderY)
	dispHeaderOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "表示設定", m.game.FontFace(optionHeaderFontSize), dispHeaderOp)

	lineY2 := dispHeaderY + 24
	ebitenutil.DrawRect(screen, labelX, lineY2, lineW, 1, lineCol)

	displayLabelCol := uiColorText
	onDisplayRow := onOptionRow(optionIdxDisplayMode)
	adjustingDisplay := onDisplayRow
	if onDisplayRow {
		displayLabelCol = uiColorSelect
	}
	displayLabelFace := m.game.FontFace(optionItemFontSize)
	if onDisplayRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(optionCtrlLabelX, displayRowY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(displayLabelCol)
		text.Draw(screen, "▶", displayLabelFace, arrowOp)
	}
	displayLabelOp := &text.DrawOptions{}
	displayLabelOp.GeoM.Translate(optionCtrlLabelX+text.Advance("▶ ", displayLabelFace), displayRowY)
	displayLabelOp.SecondaryAlign = text.AlignCenter
	displayLabelOp.ColorScale.ScaleWithColor(displayLabelCol)
	text.Draw(screen, "画面モード", displayLabelFace, displayLabelOp)

	displayValueX := optionValueX
	displayArrowCol := uiColorText
	if adjustingDisplay {
		displayArrowCol = uiColorSelect
	}
	displayValueOp := &text.DrawOptions{}
	displayValueOp.GeoM.Translate(displayValueX, displayRowY)
	displayValueOp.SecondaryAlign = text.AlignCenter
	displayValueOp.PrimaryAlign = text.AlignCenter
	displayValueOp.ColorScale.ScaleWithColor(uiColorText)
	displayModeLabel := "ウィンドウ"
	if m.game.Fullscreen {
		displayModeLabel = "フルスクリーン"
	}
	text.Draw(screen, displayModeLabel, m.game.FontFace(optionItemFontSize), displayValueOp)

	leftOp := &text.DrawOptions{}
	leftOp.GeoM.Translate(displayValueX-optionDisplayArrowGap, displayRowY)
	leftOp.SecondaryAlign = text.AlignCenter
	leftOp.PrimaryAlign = text.AlignCenter
	leftOp.ColorScale.ScaleWithColor(displayArrowCol)
	text.Draw(screen, "◀", m.game.FontFace(optionItemFontSize), leftOp)

	rightOp := &text.DrawOptions{}
	rightOp.GeoM.Translate(displayValueX+optionDisplayArrowGap, displayRowY)
	rightOp.SecondaryAlign = text.AlignCenter
	rightOp.PrimaryAlign = text.AlignCenter
	rightOp.ColorScale.ScaleWithColor(displayArrowCol)
	text.Draw(screen, "▶", m.game.FontFace(optionItemFontSize), rightOp)

	speedLabelCol := uiColorText
	onSpeedLabelRow := onSpeedRow
	if onSpeedLabelRow {
		speedLabelCol = uiColorSelect
	}
	speedLabelFace := m.game.FontFace(optionItemFontSize)
	if onSpeedLabelRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(optionCtrlLabelX, speedRowY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(speedLabelCol)
		text.Draw(screen, "▶", speedLabelFace, arrowOp)
	}
	speedLabelOp := &text.DrawOptions{}
	speedLabelOp.GeoM.Translate(optionCtrlLabelX+text.Advance("▶ ", speedLabelFace), speedRowY)
	speedLabelOp.SecondaryAlign = text.AlignCenter
	speedLabelOp.ColorScale.ScaleWithColor(speedLabelCol)
	text.Draw(screen, "メッセージ速度", speedLabelFace, speedLabelOp)

	speedOptions := []string{"遅い", "普通", "速い"}
	currentSpeedLabel := speedOptions[m.game.MessageSpeed]

	speedValueX := optionValueX

	arrowCol := uiColorText
	if adjustingSpeed {
		arrowCol = uiColorSelect
	}

	valueOp := &text.DrawOptions{}
	valueOp.GeoM.Translate(speedValueX, speedRowY)
	valueOp.SecondaryAlign = text.AlignCenter
	valueOp.PrimaryAlign = text.AlignCenter
	valueOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, currentSpeedLabel, m.game.FontFace(optionItemFontSize), valueOp)

	if m.game.MessageSpeed > 0 {
		leftOp := &text.DrawOptions{}
		leftOp.GeoM.Translate(speedValueX-optionSpeedArrowGap, speedRowY)
		leftOp.SecondaryAlign = text.AlignCenter
		leftOp.PrimaryAlign = text.AlignCenter
		leftOp.ColorScale.ScaleWithColor(arrowCol)
		text.Draw(screen, "◀", m.game.FontFace(optionItemFontSize), leftOp)
	}

	if m.game.MessageSpeed < 2 {
		rightOp := &text.DrawOptions{}
		rightOp.GeoM.Translate(speedValueX+optionSpeedArrowGap, speedRowY)
		rightOp.SecondaryAlign = text.AlignCenter
		rightOp.PrimaryAlign = text.AlignCenter
		rightOp.ColorScale.ScaleWithColor(arrowCol)
		text.Draw(screen, "▶", m.game.FontFace(optionItemFontSize), rightOp)
	}

	if onSpeedRow {
		previewFace := m.game.FontFace(optionItemFontSize)
		previewRunes := []rune(messageSpeedPreviewText)
		revealCount := m.previewRevealCount()
		visiblePreview := string(previewRunes[:revealCount])

		// 全文の幅で左端を決め、そこから通常のメッセージ表示と同じように
		// 左→右へ1文字ずつ流れる形にする（中央揃えは全文表示時の位置のみ）。
		fullWidth := text.Advance(messageSpeedPreviewText, previewFace)
		previewLeftX := optionBoxCenterX - fullWidth/2

		descOp := &text.DrawOptions{}
		descOp.GeoM.Translate(previewLeftX, descRowY)
		descOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, visiblePreview, previewFace, descOp)
	}

	sysHeaderOp := &text.DrawOptions{}
	sysHeaderOp.GeoM.Translate(labelX, sysHeaderY)
	sysHeaderOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "システム設定", m.game.FontFace(optionHeaderFontSize), sysHeaderOp)

	lineY4 := sysHeaderY + 24
	ebitenutil.DrawRect(screen, labelX, lineY4, lineW, 1, lineCol)

	onCursorRow := onOptionRow(optionIdxCursorMemory)
	adjustingCursor := onCursorRow
	cursorLabelCol := uiColorText
	if onCursorRow {
		cursorLabelCol = uiColorSelect
	}
	cursorLabelFace := m.game.FontFace(optionItemFontSize)
	if onCursorRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(optionCtrlLabelX, cursorRowY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(cursorLabelCol)
		text.Draw(screen, "▶", cursorLabelFace, arrowOp)
	}
	cursorLabelOp := &text.DrawOptions{}
	cursorLabelOp.GeoM.Translate(optionCtrlLabelX+text.Advance("▶ ", cursorLabelFace), cursorRowY)
	cursorLabelOp.SecondaryAlign = text.AlignCenter
	cursorLabelOp.ColorScale.ScaleWithColor(cursorLabelCol)
	text.Draw(screen, "カーソル記憶", cursorLabelFace, cursorLabelOp)

	cursorValueX := optionValueX
	cursorArrowCol := uiColorText
	if adjustingCursor {
		cursorArrowCol = uiColorSelect
	}
	cursorValueOp := &text.DrawOptions{}
	cursorValueOp.GeoM.Translate(cursorValueX, cursorRowY)
	cursorValueOp.SecondaryAlign = text.AlignCenter
	cursorValueOp.PrimaryAlign = text.AlignCenter
	cursorValueOp.ColorScale.ScaleWithColor(uiColorText)
	cursorValueLabel := "OFF"
	if m.game.RememberCursor {
		cursorValueLabel = "ON"
	}
	text.Draw(screen, cursorValueLabel, m.game.FontFace(optionItemFontSize), cursorValueOp)

	cursorLeftOp := &text.DrawOptions{}
	cursorLeftOp.GeoM.Translate(cursorValueX-optionCursorArrowGap, cursorRowY)
	cursorLeftOp.SecondaryAlign = text.AlignCenter
	cursorLeftOp.PrimaryAlign = text.AlignCenter
	cursorLeftOp.ColorScale.ScaleWithColor(cursorArrowCol)
	text.Draw(screen, "◀", m.game.FontFace(optionItemFontSize), cursorLeftOp)

	cursorRightOp := &text.DrawOptions{}
	cursorRightOp.GeoM.Translate(cursorValueX+optionCursorArrowGap, cursorRowY)
	cursorRightOp.SecondaryAlign = text.AlignCenter
	cursorRightOp.PrimaryAlign = text.AlignCenter
	cursorRightOp.ColorScale.ScaleWithColor(cursorArrowCol)
	text.Draw(screen, "▶", m.game.FontFace(optionItemFontSize), cursorRightOp)

	resetCol := uiColorText
	if onResetRow {
		resetCol = uiColorSelect
	}
	resetFace := m.game.FontFace(optionItemFontSize)
	const resetLabel = "すべてを初期設定に戻す"
	resetCenterX := optionBoxCenterX
	if onResetRow {
		labelW := text.Advance(resetLabel, resetFace)
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(resetCenterX-labelW/2-text.Advance("▶ ", resetFace), resetY)
		arrowOp.ColorScale.ScaleWithColor(resetCol)
		text.Draw(screen, "▶", resetFace, arrowOp)
	}
	resetOp := &text.DrawOptions{}
	resetOp.GeoM.Translate(resetCenterX, resetY)
	resetOp.PrimaryAlign = text.AlignCenter
	resetOp.ColorScale.ScaleWithColor(resetCol)
	text.Draw(screen, resetLabel, resetFace, resetOp)
}

func (m *MenuScene) drawMenuDescription(screen *ebiten.Image) {
	var desc string

	if m.drawNotice(screen) {
		return
	}

	if m.showReturnTitleConfirm {
		desc = menuCommandDescriptions["タイトルに戻る"]
		if desc == "" {
			return
		}
		x := float64(gameWidth) - menuDescOffsetX
		y := float64(gameHeight) - menuDescOffsetY
		descOp := &text.DrawOptions{}
		descOp.GeoM.Translate(x, y)
		descOp.PrimaryAlign = text.AlignEnd
		descOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, desc, m.game.FontFace(menuDescFontSize), descOp)
		return
	}

	switch m.menuState {
	case menuStateMain:
		if m.menuIndex < 0 || m.menuIndex >= len(m.commands) {
			return
		}
		d, ok := menuCommandDescriptions[m.commands[m.menuIndex]]
		if !ok {
			return
		}
		desc = d

	case menuStateSaveSlot, menuStateSaveConfirm, menuStateSaveDone:
		desc = menuCommandDescriptions["セーブ"]

	case menuStateLoadSlot, menuStateLoadConfirm:
		desc = menuCommandDescriptions["ロード"]

	case menuStateOption, menuStateOptionResetConfirm, menuStateOptionResetDone:
		d, ok := menuOptionDescriptions[m.optionIndex]
		if !ok {
			return
		}
		desc = d

	case menuStateSkillCharSel:
		desc = "スキルを確認・強化するキャラクターを選択してください"
	case menuStateStatus:
		desc = "←/→: キャラ切替　ESC: 戻る"

	case menuStateSkillSub:
		skills := m.game.CharacterSkills(m.skillCharIndex)
		if m.skillSubIndex < 0 || m.skillSubIndex >= len(skills) {
			return
		}
		sk := skills[m.skillSubIndex]
		if len(sk.Levels) == 0 {
			return
		}

		var descText string
		mpCost := -1

		if !m.skillLevelSelecting {
			descText = skillShortDescription(sk.Name)
			curLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
			if curLv < 1 {
				curLv = 1
			}
			mpCost = sk.Levels[curLv-1].MPCost
		} else {
			lv := m.skillLevelCursor
			if lv < 1 {
				lv = 1
			}
			if lv > len(sk.Levels) {
				lv = len(sk.Levels)
			}

			curLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
			if curLv < 1 {
				curLv = 1
			}

			descText = sk.Levels[lv-1].Description
			mpCost = sk.Levels[lv-1].MPCost
			if lv > curLv {
				descText = descText + "　長押しで強化"
			}
		}

		if mpCost >= 0 {
			desc = fmt.Sprintf("%s　MP:%d", descText, mpCost)
		} else {
			desc = descText
		}
	case menuStateHealTarget:
		desc = "回復する相手を選んでください"
		skills := m.game.CharacterSkills(m.skillCharIndex)
		if si, lv := m.pendingSkill-1, m.pendingSkillLevel; si >= 0 && si < len(skills) && lv >= 1 && lv <= len(skills[si].Levels) {
			desc = fmt.Sprintf("%s　消費MP:%d（残りMP:%d）", desc, skills[si].Levels[lv-1].MPCost, m.game.PlayerMP[m.skillCharIndex])
		}

	case menuStateItemList:
		items := m.usableFieldItems()
		if len(items) == 0 {
			desc = "メニューから使えるアイテムを持っていません"
			break
		}
		if m.itemListIndex < 0 || m.itemListIndex >= len(items) {
			return
		}
		def, ok := GetItemDef(items[m.itemListIndex].ItemID)
		if !ok {
			return
		}
		desc = def.Description

	case menuStateItemTarget:
		def, ok := GetItemDef(m.pendingItemID)
		if !ok {
			return
		}
		desc = def.Description

	default:
		return
	}

	if desc == "" {
		return
	}

	x := float64(gameWidth) - menuDescOffsetX
	y := float64(gameHeight) - menuDescOffsetY
	descOp := &text.DrawOptions{}
	descOp.GeoM.Translate(x, y)
	descOp.PrimaryAlign = text.AlignEnd
	descOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, desc, m.game.FontFace(menuDescFontSize), descOp)
}

func drawMinimapObjectiveIcon(screen *ebiten.Image, g *Game, ox, oy float64) {
	icon := g.MinimapObjectiveIconImg
	iw := float64(icon.Bounds().Dx())
	ih := float64(icon.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox-iw/2, oy-ih/2)
	screen.DrawImage(icon, op)
}

func dirToAngle(dir int) float64 {
	switch dir {
	case 0:
		return math.Pi
	case 1:
		return -math.Pi / 2
	case 2:
		return math.Pi / 2
	case 3:
		return 0
	}
	return 0
}
