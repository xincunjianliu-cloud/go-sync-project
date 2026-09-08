package main

// menu_draw.go: メニュー画面のメイン描画・スキル選択/強化ダイアログ

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	menuStatusOffsetX  = 275.0 // 220.0 → 260.0（+40px）
	menuStatusStartY   = 50.0
	menuStatusSpacingY = 116.0
	menuStatusFontSize = 15.0
	menuBarW           = 160.0
)

const (
	confirmPanelScale = 1.0

	// ── 確認ダイアログ画像そのものの位置(中心からのオフセット) ──
	confirmImageOffsetX = 0.0 // ← 50.0 から変更：画面中央に配置
	confirmImageOffsetY = 0.0

	// ── 確認文（「セーブしますか？」等）の位置 ──
	confirmTextOffsetX = 0.0
	confirmTextOffsetY = 10.0

	// ── 「はい/いいえ」選択肢の位置 ──
	confirmChoiceOffsetX = 0.0
	confirmChoiceStartY  = 35.0
	confirmChoiceGap     = 20.0
)

func (m *MenuScene) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(
		float64(gameWidth)/float64(m.game.MenuBgImg.Bounds().Dx()),
		float64(gameHeight)/float64(m.game.MenuBgImg.Bounds().Dy()),
	)
	screen.DrawImage(m.game.MenuBgImg, op)

	switch m.menuState {

	case menuStateStatus:
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(25, 40+float64(i*40))
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
		}
		m.drawStatusScreen(screen)
		m.drawMenuDescription(screen)
		return

	case menuStateSaveConfirm, menuStateLoadConfirm:
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(25, 40+float64(i*40))
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
		}
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, slotsPerPageView-1, slotCardStartX, slotCardStartY)

		line := "ロードしますか？"
		if m.menuState == menuStateSaveConfirm {
			line = "セーブしますか？"
		}
		drawConfirmDialog(screen, m.game, line, m.confirmIndex, confirmImageOffsetX)

		m.drawMenuDescription(screen)
		return

	case menuStateSaveDone:
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(25, 40+float64(i*40))
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
		}
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, slotsPerPageView-1, slotCardStartX, slotCardStartY)
		drawConfirmDialog(screen, m.game, m.saveResultMsg+"\n決定で戻る", 0, confirmImageOffsetX, false)
		m.drawMenuDescription(screen)
		return

	case menuStateSaveSlot, menuStateLoadSlot:
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(25, 40+float64(i*40))
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
		}
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, slotsPerPageView-1, slotCardStartX, slotCardStartY)
		m.drawMenuDescription(screen)
		return

	case menuStateOption, menuStateOptionAdjust, menuStateMessageSpeedAdjust, menuStateDisplayModeAdjust:
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(25, 40+float64(i*40))
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
		}
		m.drawVolumePanel(screen)
		m.drawMenuDescription(screen)
		return

	case menuStateSkillUpgrade:
		// 背景としてスキル画面をそのまま表示してから、確認ダイアログを重ねる
		m.drawSkillCharAndSubBase(screen)
		m.drawSkillUpgradeDialog(screen)
		return
	}

	statusX := menuStatusOffsetX
	for i, cmdName := range m.commands {
		cmdOp := &text.DrawOptions{}
		cmdOp.GeoM.Translate(25, 40+float64(i*40))
		if i == m.menuIndex && m.menuState == menuStateMain {
			cmdOp.ColorScale.ScaleWithColor(uiColorSelect)
			text.Draw(screen, "▶ "+cmdName, m.game.FontFace(20), cmdOp)
		} else {
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
		}
	}

	for i := 0; i < 4; i++ {
		itemY := menuStatusStartY + float64(i)*menuStatusSpacingY

		textColor := uiColorText
		if m.menuState == menuStateSkillCharSel {
			if i == m.skillCharIndex {
				textColor = uiColorSelect
			} else {
				textColor = uiColorText
			}
		}
		if m.game.PlayerHP[i] <= 0 {
			textColor = uiColorDead
		}
		alpha := 1.0
		if m.game.PlayerHP[i] <= 0 {
			alpha = 0.4
		}

		// ── パーティアイコン ──
		if i < len(m.game.PartyIconImgs) && m.game.PartyIconImgs[i] != nil {
			img := m.game.PartyIconImgs[i]
			iw := float64(img.Bounds().Dx())
			ih := float64(img.Bounds().Dy())
			iconOp := &ebiten.DrawImageOptions{}
			iconOp.GeoM.Translate(statusX-iw+10, itemY-ih/2+35) // -2→+6（さらに右へ）, +16→+26（さらに下へ）
			if m.game.PlayerHP[i] <= 0 {
				iconOp.ColorScale.Scale(1, 1, 1, float32(alpha))
			}
			screen.DrawImage(img, iconOp)
		}

		prefix := "  "
		if m.menuState == menuStateSkillCharSel && m.skillCharIndex == i {
			prefix = "▶ "
		}
		if m.menuState == menuStateHealTarget {
			if m.healTargetIndex == 4 {
				if m.game.PlayerHP[i] > 0 {
					prefix = "▶ "
				}
			} else if m.healTargetIndex == i {
				prefix = "▶ "
			}
		}
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(statusX+20, itemY)
		nameOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, fmt.Sprintf("%s%s  Lv %d", prefix, PlayerNames[i], m.game.PlayerLv[i]), m.game.FontFace(menuStatusFontSize), nameOp)

		drawStatusValue(screen,
			statusX+20, itemY+18,
			m.game.PlayerHP[i], m.game.PlayerMaxHP[i],
			m.game.FontFace(17.5), m.game.FontFace(14), alpha)

		hpRatio := 0.0
		if m.game.PlayerMaxHP[i] > 0 {
			hpRatio = float64(m.game.PlayerHP[i]) / float64(m.game.PlayerMaxHP[i])
		}
		drawSlantedStatusBar(screen,
			statusX+20-statusBarSlant, itemY+36,
			menuBarW, statusBarH, statusBarSlant, hpRatio,
			scaleAlpha(color.RGBA{75, 171, 120, 255}, alpha),
			scaleAlpha(color.RGBA{20, 50, 30, 255}, alpha),
			scaleAlpha(color.RGBA{41, 94, 66, 255}, alpha))

		drawStatusValue(screen,
			statusX+20, itemY+48,
			m.game.PlayerMP[i], m.game.PlayerMaxMP[i],
			m.game.FontFace(17.5), m.game.FontFace(14), alpha)

		mpRatio := 0.0
		if m.game.PlayerMaxMP[i] > 0 {
			mpRatio = float64(m.game.PlayerMP[i]) / float64(m.game.PlayerMaxMP[i])
		}
		drawSlantedStatusBar(screen,
			statusX+20-statusBarSlant, itemY+66,
			menuBarW, statusBarH, statusBarSlant, mpRatio,
			scaleAlpha(color.RGBA{75, 105, 171, 255}, alpha),
			scaleAlpha(color.RGBA{20, 30, 55, 255}, alpha),
			scaleAlpha(color.RGBA{41, 58, 94, 255}, alpha))

	}

	m.drawMinimap(screen)

	if m.menuState == menuStateSkillSub {
		m.drawSkillSubMenu(screen)
	}

	if m.menuState == menuStateReturnTitleConfirm {
		drawConfirmDialog(screen, m.game, "タイトルに戻りますか？", m.confirmIndex, confirmImageOffsetX)
	}

	m.drawMenuDescription(screen)
}

// drawSkillCharAndSubBase はskillUpgrade画面の背景として、
// 通常のスキル画面(ステータス一覧+サブメニュー)を描画する。
func (m *MenuScene) drawSkillCharAndSubBase(screen *ebiten.Image) {
	statusX := menuStatusOffsetX
	for i, cmdName := range m.commands {
		cmdOp := &text.DrawOptions{}
		cmdOp.GeoM.Translate(25, 40+float64(i*40))
		cmdOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "  "+cmdName, m.game.FontFace(20), cmdOp)
	}
	for i := 0; i < 4; i++ {
		itemY := menuStatusStartY + float64(i)*menuStatusSpacingY
		textColor := uiColorText
		if i == m.skillCharIndex {
			textColor = uiColorSelect
		}
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(statusX+20, itemY)
		nameOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, fmt.Sprintf("%s  Lv %d", PlayerNames[i], m.game.PlayerLv[i]), m.game.FontFace(menuStatusFontSize), nameOp)
	}
	m.drawSkillSubMenu(screen)
}

// スキルパネル専用の表示開始位置（左上基準）
// スキルパネル専用の表示開始位置（左上基準）
// スキルパネル専用の表示開始位置（左上基準）
const (
	skillSubPanelX = 250.0
	skillSubPanelY = 100.0

	skillSubNameOffsetX     = 20.0
	skillSubRowStartOffsetY = 30.0
	skillSubRowGapY         = 30.0

	// ── 所持SP／「決定で強化」ヒント：同じYで横並び ──
	skillSubSPOffsetX   = 550.0 // 所持SPのX
	skillSubHintOffsetX = 470.0 // 決定で強化のX（所持SPの右）

	// ── 説明文：別Xグループ ──
	skillSubDescOffsetX = 10.0

	// ── 所持SP／決定で強化／説明文で共通して使う、パネル下端からのYオフセット ──
	skillSubBottomOffsetY = 20.0

	// 強化結果メッセージだけ少し上にずらす追加オフセット
	skillSubResultMsgExtraOffsetY = 18.0
)

func (m *MenuScene) drawSkillSubMenu(screen *ebiten.Image) {
	img := m.game.MenuSkillPanelImg
	winH := 200.0
	if img != nil {
		winH = float64(img.Bounds().Dy())
	}
	winX := skillSubPanelX
	winY := skillSubPanelY

	if img != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(winX, winY)
		screen.DrawImage(img, op)
	}

	skills := m.game.CharacterSkills(m.skillCharIndex)

	const (
		numStartX = 150.0
		numGap    = 22.0
	)

	for i, sk := range skills {
		if len(sk.Levels) == 0 {
			continue
		}
		curLv := m.game.PlayerSkillLv[m.skillCharIndex][i]
		if curLv < 1 {
			curLv = 1
		}

		rowY := winY + skillSubRowStartOffsetY + float64(i)*skillSubRowGapY // ← 変更
		rowSelected := i == m.skillSubIndex

		prefix := "  "
		nameCol := uiColorText
		if rowSelected {
			if !m.skillLevelSelecting {
				prefix = "▶ "
				nameCol = uiColorSelect
			} else {
				nameCol = uiColorSelect
			}
		}

		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(winX+skillSubNameOffsetX, rowY) // ← 変更
		nameOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, prefix+sk.Name, m.game.FontFace(15), nameOp)

		for lv := 1; lv <= len(sk.Levels); lv++ {
			numX := winX + numStartX + float64(lv-1)*numGap
			col := uiColorText
			label := fmt.Sprintf("%d", lv)
			if lv <= curLv {
				col = uiColorText
			}

			selected := rowSelected && m.skillLevelSelecting && lv == m.skillLevelCursor
			if selected {
				col = uiColorSelect
			}

			numOp := &text.DrawOptions{}
			numOp.GeoM.Translate(numX, rowY)
			numOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, label, m.game.FontFace(14), numOp)

			if selected {
				arrowOp := &text.DrawOptions{}
				arrowOp.GeoM.Translate(numX+4, rowY-12)
				arrowOp.ColorScale.ScaleWithColor(uiColorSelect)
				text.Draw(screen, "▼", m.game.FontFace(11), arrowOp)
			}
		}
	}

	spOp := &text.DrawOptions{}
	spOp.GeoM.Translate(winX+skillSubSPOffsetX, winY+winH-skillSubBottomOffsetY)
	spOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("所持SP: %d", m.game.PlayerSP[m.skillCharIndex]), m.game.FontFace(13), spOp)

	if m.upgradeResultMsg != "" {
		msgOp := &text.DrawOptions{}
		msgOp.GeoM.Translate(winX+skillSubSPOffsetX, winY+winH-skillSubBottomOffsetY-skillSubResultMsgExtraOffsetY)
		msgOp.ColorScale.ScaleWithColor(uiColorSelect)
		text.Draw(screen, m.upgradeResultMsg, m.game.FontFace(12), msgOp)
	}
}

// drawSkillUpgradeDialog：スキル強化確認ダイアログ(はい/いいえ + 必要SP表示)
func (m *MenuScene) drawSkillUpgradeDialog(screen *ebiten.Image) {
	charIdx := m.skillCharIndex
	skillIdx := m.skillSubIndex

	winW, winH := 260.0, 120.0
	winX := float64(gameWidth)/2 - winW/2
	winY := float64(gameHeight)/2 - winH/2

	ebitenutil.DrawRect(screen, winX, winY, winW, winH, color.RGBA{10, 10, 25, 235})
	ebitenutil.DrawRect(screen, winX, winY, winW, 1, uiColorText)
	ebitenutil.DrawRect(screen, winX, winY+winH, winW, 1, uiColorText)
	ebitenutil.DrawRect(screen, winX, winY, 1, winH, uiColorText)
	ebitenutil.DrawRect(screen, winX+winW, winY, 1, winH, uiColorText)

	currentLv := m.game.PlayerSkillLv[charIdx][skillIdx]
	skillName := m.game.CharacterSkills(charIdx)[skillIdx].Name

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(winX+winW/2, winY+18)
	titleOp.PrimaryAlign = text.AlignCenter
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("%s Lv.%d → Lv.%d", skillName, currentLv, currentLv+1), m.game.FontFace(15), titleOp)

	cost := SkillUpgradeCost(currentLv)
	costText := "強化済み(最大Lv)"
	if cost > 0 {
		costText = fmt.Sprintf("必要SP: %d（所持: %d）", cost, m.game.PlayerSP[charIdx])
	}
	costOp := &text.DrawOptions{}
	costOp.GeoM.Translate(winX+winW/2, winY+42)
	costOp.PrimaryAlign = text.AlignCenter
	costOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, costText, m.game.FontFace(13), costOp)

	choices := []string{"強化する", "やめる"}
	for j, choice := range choices {
		choiceOp := &text.DrawOptions{}
		choiceOp.GeoM.Translate(winX+winW/2, winY+70+float64(j)*22)
		choiceOp.PrimaryAlign = text.AlignCenter
		if j == m.upgradeConfirmIndex {
			col := uiColorSelect
			if j == 0 && !m.game.CanUpgradeSkill(charIdx, skillIdx) {
				col = uiColorText
			}
			choiceOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, "▶ "+choice, m.game.FontFace(14), choiceOp)
		} else {
			choiceOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, "  "+choice, m.game.FontFace(14), choiceOp)
		}
	}

	if m.upgradeResultMsg != "" {
		msgOp := &text.DrawOptions{}
		msgOp.GeoM.Translate(winX+winW/2, winY+winH+16)
		msgOp.PrimaryAlign = text.AlignCenter
		msgOp.ColorScale.ScaleWithColor(uiColorSelect)
		text.Draw(screen, m.upgradeResultMsg, m.game.FontFace(13), msgOp)
	}
}
