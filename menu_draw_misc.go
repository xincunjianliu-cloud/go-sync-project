package main

// menu_draw_misc.go: ミニマップ・音量パネル・ヒント/説明文の描画
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

// スキルパネル用ヒントの表示開始位置（左上基準の絶対座標）
const (
	skillSubHintX   = 600.0 // ← 調整用：ヒントの開始X座標
	skillSubHintY   = 400.0 // ← 調整用：ヒントの開始Y座標
	skillSubMPDescX = 600.0
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

	if m.game.MinimapPlayerIconImg != nil {
		icon := m.game.MinimapPlayerIconImg
		iw := float64(icon.Bounds().Dx())
		ih := float64(icon.Bounds().Dy())

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-iw/2, -ih/2)
		op.GeoM.Rotate(dirToAngle(field.dir))
		op.GeoM.Translate(plx, ply)
		screen.DrawImage(icon, op)
	} else {
		dotSize := 5.0
		ebitenutil.DrawRect(screen, plx-dotSize/2, ply-dotSize/2, dotSize, dotSize, color.RGBA{255, 80, 80, 255})
	}

	if loc, ok := m.game.CurrentObjectiveLocation(); ok {
		if loc.MapPath == field.currentMap {
			// 目的地が同じマップにある場合：実座標にそのままアイコンを表示
			drawMinimapObjectiveIcon(screen, m.game, offsetX+loc.X*scale, offsetY+loc.Y*scale)
		} else if field.hasObjectiveDoor {
			// 目的地が別マップにある場合：そちらへ向かう誘導ドアの位置に
			// 同じアイコンを表示する（フィールド上に直接印を付けるのはやめて、ここに統一）
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
	volumeArrowGap      float64 = 10

	volumeResetGapY float64 = 44
)

func (m *MenuScene) drawVolumePanel(screen *ebiten.Image) {
	var volume float64
	if m.game.Audio != nil {
		volume = m.game.Audio.volume
	}

	adjustingVolume := m.menuState == menuStateOptionAdjust
	adjustingSpeed := m.menuState == menuStateMessageSpeedAdjust
	onVolumeRow := adjustingVolume || (m.menuState == menuStateOption && m.optionIndex == 0)
	// ↓ 【変更】メッセージ速度のカーソル判定を optionIndex == 2 に変更
	onSpeedRow := adjustingSpeed || (m.menuState == menuStateOption && m.optionIndex == 2)
	onResetRow := m.menuState == menuStateOption && m.optionIndex == 3
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
	if onVolumeRow && m.menuState == menuStateOption {
		labelCol = uiColorSelect
	}
	bgmFace := m.game.FontFace(22)
	if onVolumeRow && m.menuState == menuStateOption {
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

	if adjustingVolume {
		leftOp := &text.DrawOptions{}
		leftOp.GeoM.Translate(barX-volumeArrowGap, barY+volumeBarH/2)
		leftOp.SecondaryAlign = text.AlignCenter
		leftOp.PrimaryAlign = text.AlignEnd
		leftOp.ColorScale.ScaleWithColor(uiColorSelect)
		text.Draw(screen, "◀", m.game.FontFace(20), leftOp)

		rightOp := &text.DrawOptions{}
		rightOp.GeoM.Translate(barX+volumeBarW+volumeArrowGap, barY+volumeBarH/2)
		rightOp.SecondaryAlign = text.AlignCenter
		rightOp.ColorScale.ScaleWithColor(uiColorSelect)
		text.Draw(screen, "▶", m.game.FontFace(20), rightOp)
	}

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

	// ── 【変更】表示モード行を先に描画する（上に配置） ──
	displayRowY := lineY2 + 40
	displayLabelCol := uiColorText
	// ↓ 【変更】表示モードのカーソル判定を optionIndex == 1 に変更
	onDisplayRow := m.menuState == menuStateDisplayModeAdjust || (m.menuState == menuStateOption && m.optionIndex == 1)
	adjustingDisplay := m.menuState == menuStateDisplayModeAdjust
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
	text.Draw(screen, "表示モード", displayLabelFace, displayLabelOp)

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

	// フルスクリーン/ウィンドウの二択トグルなので、矢印は常に両方表示する
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

	// ── 【変更】メッセージ速度行を次に描画する（下に配置） ──
	speedRowY := displayRowY + 40
	speedLabelCol := uiColorText
	onSpeedLabelRow := onSpeedRow && m.menuState == menuStateOption
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

	// ── メッセージ速度のプレビュー描画 ──
	descRowY := speedRowY + 30 // 位置を調整（+40のオフセットは削除）

	previewRunes := []rune(messageSpeedPreviewText)
	revealCount := m.previewRevealCount()
	visiblePreview := string(previewRunes[:revealCount])

	descOp := &text.DrawOptions{}
	descOp.GeoM.Translate(labelX, descRowY)
	descOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, visiblePreview, m.game.FontFace(13), descOp)

	lineY3 := descRowY + 20
	ebitenutil.DrawRect(screen, labelX, lineY3, lineW, 1, lineCol)

	resetY := lineY3 + volumeResetGapY
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

	case menuStateReturnTitleConfirm: // ← 追加
		desc = menuCommandDescriptions["タイトルに戻る"]

	case menuStateOption, menuStateOptionAdjust, menuStateMessageSpeedAdjust:
		d, ok := menuOptionDescriptions[m.optionIndex]
		if !ok {
			return
		}
		desc = d

	case menuStateSkillCharSel:
		desc = "スキルを使うキャラクターを選択してください"
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
			// ── 行選択中（スキル名にカーソル）：シンプルな概要のみ ──
			descText = skillShortDescription(sk.Name)
			curLv := m.game.PlayerSkillLv[m.skillCharIndex][m.skillSubIndex]
			if curLv < 1 {
				curLv = 1
			}
			mpCost = sk.Levels[curLv-1].MPCost
		} else {
			// ── Lv選択中（数字にカーソル） ──
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
				// 未強化のレベル：長押しで強化できることを案内する
				descText = descText + "　長押しで強化"
			}
		}

		img := m.game.MenuSkillPanelImg
		winH := 200.0
		if img != nil {
			winH = float64(img.Bounds().Dy())
		}
		baseY := skillSubPanelY + winH - skillSubBottomOffsetY

		if descText != "" {
			descOp := &text.DrawOptions{}
			descOp.GeoM.Translate(skillSubPanelX+skillSubDescOffsetX, baseY)
			descOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, descText, m.game.FontFace(skillSubBottomFontSize), descOp)
		}
		if mpCost >= 0 {
			mpOp := &text.DrawOptions{}
			mpOp.GeoM.Translate(skillSubMPDescX, baseY)
			mpOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, fmt.Sprintf("MP:%d", mpCost), m.game.FontFace(skillSubBottomFontSize), mpOp)
		}
		return
	case menuStateHealTarget:
		hint := "→:全体回復に切替"
		if m.healTargetIndex == partySize {
			hint = "←:個人選択に戻す"
		}
		d, ok := menuSkillDescriptions[skillIdxHeal]
		if !ok {
			return
		}
		desc = d + "　" + hint

	case menuStateItemList:
		items := m.usableFieldItems()
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
		if def.Target == TargetAll || def.Target == TargetBoth {
			hint := "→:全体に切替"
			if m.itemTargetIndex == partySize {
				hint = "←:個人選択に戻す"
			}
			desc = desc + "　" + hint
		}

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

func drawObjectiveStar(screen *ebiten.Image, cx, cy float64) {
	size := 6.0
	col := color.RGBA{255, 220, 60, 255}
	ebitenutil.DrawRect(screen, cx-size/2, cy-1, size, 2, col)
	ebitenutil.DrawRect(screen, cx-1, cy-size/2, 2, size, col)
	ebitenutil.DrawRect(screen, cx-size/2-1, cy-2, size+2, 1, color.RGBA{255, 255, 255, 180})
	ebitenutil.DrawRect(screen, cx-size/2-1, cy+1, size+2, 1, color.RGBA{255, 255, 255, 180})
}

// drawMinimapObjectiveIcon はミニマップ上の指定スクリーン座標(ox, oy)に
// 目的地アイコン（画像が無ければ星形フォールバック）を描画する。
// 「現在マップ内の目的地の実座標」「別マップへの誘導ドア座標」の両方で共通利用する。
func drawMinimapObjectiveIcon(screen *ebiten.Image, g *Game, ox, oy float64) {
	if g.MinimapObjectiveIconImg != nil {
		icon := g.MinimapObjectiveIconImg
		iw := float64(icon.Bounds().Dx())
		ih := float64(icon.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(ox-iw/2, oy-ih/2)
		screen.DrawImage(icon, op)
	} else {
		drawObjectiveStar(screen, ox, oy)
	}
}

func dirToAngle(dir int) float64 {
	switch dir {
	case 0: // 下
		return math.Pi
	case 1: // 左
		return -math.Pi / 2
	case 2: // 右
		return math.Pi / 2
	case 3: // 上
		return 0
	}
	return 0
}
