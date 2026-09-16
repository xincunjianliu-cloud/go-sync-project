package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	minimapX = 500.0
	minimapY = 40.0
	minimapW = 400.0
	minimapH = 460.0
)

const (
	menuDescOffsetX float64 = 20
	menuDescOffsetY float64 = 30
)

const (
	skillSubHintX = 600.0
	skillSubHintY = 400.0
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
		isWall := layer.Name == "kabe"
		isFloor := layer.Name == "floor" || layer.Name == "ground" || layer.Name == "michi"

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
	volumePanelY float64 = 70

	volumeBarW float64 = 300
	volumeBarH float64 = 30

	volumeLabelBarGap   float64 = 110
	volumeBarPercentGap float64 = 30

	volumeResetGapY float64 = 44
)

func (m *MenuScene) drawVolumePanel(screen *ebiten.Image) {
	var volume float64
	if m.game.Audio != nil {
		volume = m.game.Audio.volume
	}

	onOptionRow := func(idx int) bool { return m.menuState == menuStateOption && m.optionIndex == idx }
	onVolumeRow := onOptionRow(0)
	adjustingVolume := onVolumeRow
	onSpeedRow := onOptionRow(2)
	adjustingSpeed := onSpeedRow
	onResetRow := m.optionIndex == 4 && (m.menuState == menuStateOption ||
		m.menuState == menuStateOptionResetConfirm || m.menuState == menuStateOptionResetDone)
	labelX := volumeGroupX
	barX := labelX + volumeLabelBarGap
	lineW := volumeBarW + volumeLabelBarGap
	lineCol := uiColorText

	headerY := volumePanelY - 34
	headerOp := &text.DrawOptions{}
	headerOp.GeoM.Translate(labelX, headerY)
	headerOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "音量設定", m.game.FontFace(18), headerOp)

	lineY1 := headerY + 24
	ebitenutil.DrawRect(screen, labelX, lineY1, lineW, 1, lineCol)

	barY := volumePanelY

	barFillCol := uiColorText
	barFrameCol := uiColorText
	if adjustingVolume {
		barFillCol = uiColorSelect
		barFrameCol = uiColorSelect
	}

	ebitenutil.DrawRect(screen, barX, barY, volumeBarW, volumeBarH, color.RGBA{30, 30, 40, 255})
	ebitenutil.DrawRect(screen, barX, barY, volumeBarW*volume, volumeBarH, barFillCol)
	ebitenutil.DrawRect(screen, barX, barY, volumeBarW, 1, barFrameCol)
	ebitenutil.DrawRect(screen, barX, barY+volumeBarH-1, volumeBarW, 1, barFrameCol)
	ebitenutil.DrawRect(screen, barX, barY, 1, volumeBarH, barFrameCol)
	ebitenutil.DrawRect(screen, barX+volumeBarW-1, barY, 1, volumeBarH, barFrameCol)

	labelCol := uiColorText
	if onVolumeRow {
		labelCol = uiColorSelect
	}
	bgmFace := m.game.FontFace(22)
	if onVolumeRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(labelX, barY+volumeBarH/2)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(labelCol)
		text.Draw(screen, "▶", bgmFace, arrowOp)
	}
	labelOp := &text.DrawOptions{}
	labelOp.GeoM.Translate(labelX+text.Advance("▶ ", bgmFace), barY+volumeBarH/2)
	labelOp.SecondaryAlign = text.AlignCenter
	labelOp.ColorScale.ScaleWithColor(labelCol)
	text.Draw(screen, "BGM", bgmFace, labelOp)

	percentOp := &text.DrawOptions{}
	percentOp.GeoM.Translate(barX+volumeBarW+volumeBarPercentGap, barY+volumeBarH/2)
	percentOp.SecondaryAlign = text.AlignCenter
	percentOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("%d%%", int(volume*100+0.5)), m.game.FontFace(22), percentOp)

	dispHeaderY := barY + volumeBarH + 30
	dispHeaderOp := &text.DrawOptions{}
	dispHeaderOp.GeoM.Translate(labelX, dispHeaderY)
	dispHeaderOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, "表示設定", m.game.FontFace(18), dispHeaderOp)

	lineY2 := dispHeaderY + 24
	ebitenutil.DrawRect(screen, labelX, lineY2, lineW, 1, lineCol)

	displayRowY := lineY2 + 40
	displayLabelCol := uiColorText
	onDisplayRow := onOptionRow(1)
	adjustingDisplay := onDisplayRow
	if onDisplayRow {
		displayLabelCol = uiColorSelect
	}
	displayLabelFace := m.game.FontFace(20)
	if onDisplayRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(labelX, displayRowY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(displayLabelCol)
		text.Draw(screen, "▶", displayLabelFace, arrowOp)
	}
	displayLabelOp := &text.DrawOptions{}
	displayLabelOp.GeoM.Translate(labelX+text.Advance("▶ ", displayLabelFace), displayRowY)
	displayLabelOp.SecondaryAlign = text.AlignCenter
	displayLabelOp.ColorScale.ScaleWithColor(displayLabelCol)
	text.Draw(screen, "画面モード", displayLabelFace, displayLabelOp)

	displayValueX := barX + 180
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
	text.Draw(screen, displayModeLabel, m.game.FontFace(16), displayValueOp)

	leftOp := &text.DrawOptions{}
	leftOp.GeoM.Translate(displayValueX-70, displayRowY)
	leftOp.SecondaryAlign = text.AlignCenter
	leftOp.PrimaryAlign = text.AlignCenter
	leftOp.ColorScale.ScaleWithColor(displayArrowCol)
	text.Draw(screen, "◀", m.game.FontFace(18), leftOp)

	rightOp := &text.DrawOptions{}
	rightOp.GeoM.Translate(displayValueX+70, displayRowY)
	rightOp.SecondaryAlign = text.AlignCenter
	rightOp.PrimaryAlign = text.AlignCenter
	rightOp.ColorScale.ScaleWithColor(displayArrowCol)
	text.Draw(screen, "▶", m.game.FontFace(18), rightOp)

	speedRowY := displayRowY + 40
	speedLabelCol := uiColorText
	onSpeedLabelRow := onSpeedRow
	if onSpeedLabelRow {
		speedLabelCol = uiColorSelect
	}
	speedLabelFace := m.game.FontFace(20)
	if onSpeedLabelRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(labelX, speedRowY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(speedLabelCol)
		text.Draw(screen, "▶", speedLabelFace, arrowOp)
	}
	speedLabelOp := &text.DrawOptions{}
	speedLabelOp.GeoM.Translate(labelX+text.Advance("▶ ", speedLabelFace), speedRowY)
	speedLabelOp.SecondaryAlign = text.AlignCenter
	speedLabelOp.ColorScale.ScaleWithColor(speedLabelCol)
	text.Draw(screen, "メッセージ速度", speedLabelFace, speedLabelOp)

	speedOptions := []string{"遅い", "普通", "速い"}
	currentSpeedLabel := speedOptions[m.game.MessageSpeed]

	speedValueX := barX + 180

	arrowCol := uiColorText
	if adjustingSpeed {
		arrowCol = uiColorSelect
	}

	valueOp := &text.DrawOptions{}
	valueOp.GeoM.Translate(speedValueX, speedRowY)
	valueOp.SecondaryAlign = text.AlignCenter
	valueOp.PrimaryAlign = text.AlignCenter
	valueOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, currentSpeedLabel, m.game.FontFace(18), valueOp)

	if m.game.MessageSpeed > 0 {
		leftOp := &text.DrawOptions{}
		leftOp.GeoM.Translate(speedValueX-60, speedRowY)
		leftOp.SecondaryAlign = text.AlignCenter
		leftOp.PrimaryAlign = text.AlignCenter
		leftOp.ColorScale.ScaleWithColor(arrowCol)
		text.Draw(screen, "◀", m.game.FontFace(18), leftOp)
	}

	if m.game.MessageSpeed < 2 {
		rightOp := &text.DrawOptions{}
		rightOp.GeoM.Translate(speedValueX+60, speedRowY)
		rightOp.SecondaryAlign = text.AlignCenter
		rightOp.PrimaryAlign = text.AlignCenter
		rightOp.ColorScale.ScaleWithColor(arrowCol)
		text.Draw(screen, "▶", m.game.FontFace(18), rightOp)
	}

	descRowY := speedRowY + 30

	if onSpeedRow {
		previewRunes := []rune(messageSpeedPreviewText)
		revealCount := m.previewRevealCount()
		visiblePreview := string(previewRunes[:revealCount])

		descOp := &text.DrawOptions{}
		descOp.GeoM.Translate(labelX, descRowY)
		descOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, visiblePreview, m.game.FontFace(18), descOp)
	}

	lineY3 := descRowY + 20
	ebitenutil.DrawRect(screen, labelX, lineY3, lineW, 1, lineCol)

	cursorRowY := lineY3 + 30
	onCursorRow := onOptionRow(3)
	adjustingCursor := onCursorRow
	cursorLabelCol := uiColorText
	if onCursorRow {
		cursorLabelCol = uiColorSelect
	}
	cursorLabelFace := m.game.FontFace(20)
	if onCursorRow {
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(labelX, cursorRowY)
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(cursorLabelCol)
		text.Draw(screen, "▶", cursorLabelFace, arrowOp)
	}
	cursorLabelOp := &text.DrawOptions{}
	cursorLabelOp.GeoM.Translate(labelX+text.Advance("▶ ", cursorLabelFace), cursorRowY)
	cursorLabelOp.SecondaryAlign = text.AlignCenter
	cursorLabelOp.ColorScale.ScaleWithColor(cursorLabelCol)
	text.Draw(screen, "カーソル記憶", cursorLabelFace, cursorLabelOp)

	cursorValueX := barX + 180
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
	text.Draw(screen, cursorValueLabel, m.game.FontFace(16), cursorValueOp)

	cursorLeftOp := &text.DrawOptions{}
	cursorLeftOp.GeoM.Translate(cursorValueX-70, cursorRowY)
	cursorLeftOp.SecondaryAlign = text.AlignCenter
	cursorLeftOp.PrimaryAlign = text.AlignCenter
	cursorLeftOp.ColorScale.ScaleWithColor(cursorArrowCol)
	text.Draw(screen, "◀", m.game.FontFace(18), cursorLeftOp)

	cursorRightOp := &text.DrawOptions{}
	cursorRightOp.GeoM.Translate(cursorValueX+70, cursorRowY)
	cursorRightOp.SecondaryAlign = text.AlignCenter
	cursorRightOp.PrimaryAlign = text.AlignCenter
	cursorRightOp.ColorScale.ScaleWithColor(cursorArrowCol)
	text.Draw(screen, "▶", m.game.FontFace(18), cursorRightOp)

	resetY := cursorRowY + volumeResetGapY
	resetCol := uiColorText
	if onResetRow {
		resetCol = uiColorSelect
	}
	resetFace := m.game.FontFace(18)
	const resetLabel = "すべてを初期設定に戻す"
	resetCenterX := barX + volumeBarW/2
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

func (m *MenuScene) drawBottomRightHint(screen *ebiten.Image, hint string) {
	if hint == "" {
		return
	}
	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(float64(gameWidth)-16, float64(gameHeight)-16)
	hintOp.PrimaryAlign = text.AlignEnd
	hintOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, hint, m.game.FontFace(15), hintOp)
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
		text.Draw(screen, desc, m.game.FontFace(14), descOp)
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
	align := text.AlignEnd

	descOp := &text.DrawOptions{}
	descOp.GeoM.Translate(x, y)
	descOp.PrimaryAlign = align
	descOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, desc, m.game.FontFace(14), descOp)
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
