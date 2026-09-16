package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	menuStatusOffsetX  = 275.0
	menuStatusStartY   = 40.0
	menuStatusSpacingY = 116.0
	menuStatusFontSize = 15.0
	menuStatusCursorX  = 0.0
	menuStatusNameX    = 20.0
	menuStatusLevelX   = 135.0
	menuBarW           = 160.0
)

const (
	cmdListFontSize = 20.0
	cmdListStartX   = 20.0
	cmdListStartY   = 30.0
	cmdListRowGapY  = 40.0
)

const (
	menuFrameLeft   = 15.0
	menuFrameTop    = 8.0
	menuFrameRight  = gameWidth - menuFrameLeft
	menuFrameBottom = gameHeight - menuFrameTop

	menuFrameDividerX = 203.0
	menuFrameDividerY = 364.0

	menuLineWidth = 2.0
)

func drawMenuBgFrame(screen *ebiten.Image) {
	w := menuLineWidth

	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameTop, menuFrameRight-menuFrameLeft, w, uiColorMenuLine)
	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameBottom-w, menuFrameRight-menuFrameLeft, w, uiColorMenuLine)
	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameTop, w, menuFrameBottom-menuFrameTop, uiColorMenuLine)
	ebitenutil.DrawRect(screen, menuFrameRight-w, menuFrameTop, w, menuFrameBottom-menuFrameTop, uiColorMenuLine)

	ebitenutil.DrawRect(screen, menuFrameDividerX-w/2, menuFrameTop, w, menuFrameBottom-menuFrameTop, uiColorMenuLine)

	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameDividerY-w/2, menuFrameDividerX-menuFrameLeft+w/2, w, uiColorMenuLine)
}

const (
	confirmPanelW           = 500.0
	confirmPanelH           = 170.0
	confirmPanelBorderWidth = 2.0
	confirmFontSize         = 18.0

	confirmImageOffsetX = 0.0
	confirmImageOffsetY = 0.0

	confirmTextOffsetX = 0.0
	confirmTextOffsetY = 30.0

	confirmChoiceOffsetX = 0.0
	confirmChoiceStartY  = 90.0
	confirmChoiceGap     = 32.0
)

func (m *MenuScene) drawCommandList(screen *ebiten.Image) {
	cmdFace := m.game.FontFace(cmdListFontSize)
	cmdArrowGap := text.Advance("▶ ", cmdFace)
	for i, cmdName := range m.commands {
		baseX, baseY := cmdListStartX, cmdListStartY+float64(i)*cmdListRowGapY
		col := uiColorText
		if i == m.menuIndex {
			col = uiColorSelect
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(baseX, baseY)
			arrowOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, "▶", cmdFace, arrowOp)
		}
		cmdOp := &text.DrawOptions{}
		cmdOp.GeoM.Translate(baseX+cmdArrowGap, baseY)
		cmdOp.ColorScale.ScaleWithColor(col)
		text.Draw(screen, cmdName, cmdFace, cmdOp)
	}
}

func (m *MenuScene) allTargetRowState() (allowed, selected bool) {
	switch m.menuState {
	case menuStateSkillSub:
		return true, false
	case menuStateHealTarget:
		return true, m.healTargetIndex == partySize
	case menuStateItemTarget:
		def, ok := GetItemDef(m.pendingItemID)
		if !ok {
			return false, false
		}
		allowAll := def.Target == TargetAll || def.Target == TargetBoth
		return allowAll, allowAll && m.itemTargetIndex == partySize
	}
	return false, false
}

const drawAllTargetRowBoxH = 30.0

const drawAllTargetRowBoxInset = 16.0

const drawAllTargetRowYOffset = 12.0

func (m *MenuScene) drawAllTargetRow(screen *ebiten.Image, statusX float64) {
	allowed, selected := m.allTargetRowState()
	if !allowed {
		return
	}

	y := menuStatusStartY + partySize*menuStatusSpacingY - drawAllTargetRowYOffset

	interactive := m.menuState == menuStateHealTarget || m.menuState == menuStateItemTarget

	col := uiColorText
	boxFillCol := color.RGBA{45, 45, 55, 200}
	if !interactive {
		col = uiColorDisabled
	}
	if selected {
		col = uiColorSelect
		boxFillCol = color.RGBA{41, 58, 94, 220}
	}

	iconW := float64(m.game.PartyIconImgs[0].Bounds().Dx())
	boxX := statusX - iconW + 10 + drawAllTargetRowBoxInset
	boxRight := statusX + menuStatusNameX + menuBarW - drawAllTargetRowBoxInset
	boxW := boxRight - boxX
	boxY := y - drawAllTargetRowBoxH/2

	ebitenutil.DrawRect(screen, boxX, boxY, boxW, drawAllTargetRowBoxH, boxFillCol)

	nameFace := m.game.FontFace(menuStatusFontSize)
	textCenterX := boxX + boxW/2

	if selected {
		const bw = 2.0
		ebitenutil.DrawRect(screen, boxX, boxY, boxW, bw, col)
		ebitenutil.DrawRect(screen, boxX, boxY+drawAllTargetRowBoxH-bw, boxW, bw, col)
		ebitenutil.DrawRect(screen, boxX, boxY, bw, drawAllTargetRowBoxH, col)
		ebitenutil.DrawRect(screen, boxX+boxW-bw, boxY, bw, drawAllTargetRowBoxH, col)

		nameW := text.Advance("全体", nameFace)
		arrowOp := &text.DrawOptions{}
		arrowOp.GeoM.Translate(textCenterX-nameW/2-6, y)
		arrowOp.PrimaryAlign = text.AlignEnd
		arrowOp.SecondaryAlign = text.AlignCenter
		arrowOp.ColorScale.ScaleWithColor(col)
		text.Draw(screen, "▶", nameFace, arrowOp)
	}

	nameOp := &text.DrawOptions{}
	nameOp.GeoM.Translate(textCenterX, y)
	nameOp.PrimaryAlign = text.AlignCenter
	nameOp.SecondaryAlign = text.AlignCenter
	nameOp.ColorScale.ScaleWithColor(col)
	text.Draw(screen, "全体", nameFace, nameOp)
}

func (m *MenuScene) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(
		float64(gameWidth)/float64(m.game.MenuBgImg.Bounds().Dx()),
		float64(gameHeight)/float64(m.game.MenuBgImg.Bounds().Dy()),
	)
	screen.DrawImage(m.game.MenuBgImg, op)
	drawMenuBgFrame(screen)

	drawBackButton(screen, m.game)

	switch m.menuState {

	case menuStateStatus:
		m.drawCommandList(screen)
		m.drawStatusScreen(screen)
		if m.showReturnTitleConfirm {
			drawConfirmDialog(screen, m.game, "タイトルに戻りますか？", m.confirmIndex, confirmImageOffsetX)
		}
		m.drawMenuDescription(screen)
		return

	case menuStateSaveConfirm, menuStateLoadConfirm:
		m.drawCommandList(screen)
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, m.slotScrollTop, slotCardStartX, slotCardStartY, m.slotDragAccum)

		line := "ロードしますか？"
		if m.menuState == menuStateSaveConfirm {
			line = m.saveConfirmMessage()
		}
		drawConfirmDialog(screen, m.game, line, m.confirmIndex, confirmImageOffsetX)

		m.drawMenuDescription(screen)
		return

	case menuStateSaveDone:
		m.drawCommandList(screen)
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, m.slotScrollTop, slotCardStartX, slotCardStartY, m.slotDragAccum)
		drawConfirmDialog(screen, m.game, m.saveResultMsg, 0, confirmImageOffsetX, false)
		m.drawMenuDescription(screen)
		return

	case menuStateSaveSlot, menuStateLoadSlot:
		m.drawCommandList(screen)
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, m.slotScrollTop, slotCardStartX, slotCardStartY, m.slotDragAccum)
		if m.showReturnTitleConfirm {
			drawConfirmDialog(screen, m.game, "タイトルに戻りますか？", m.confirmIndex, confirmImageOffsetX)
		}
		m.drawMenuDescription(screen)
		return

	case menuStateOption, menuStateOptionResetConfirm, menuStateOptionResetDone:
		m.drawCommandList(screen)
		m.drawVolumePanel(screen)
		if m.menuState == menuStateOptionResetConfirm {
			drawConfirmDialog(screen, m.game, "設定を初期値に戻しますか？", m.confirmIndex, confirmImageOffsetX)
		}
		if m.menuState == menuStateOptionResetDone {
			drawConfirmDialog(screen, m.game, "初期設定に戻しました", 0, confirmImageOffsetX, false)
		}
		if m.showReturnTitleConfirm {
			drawConfirmDialog(screen, m.game, "タイトルに戻りますか？", m.confirmIndex, confirmImageOffsetX)
		}
		m.drawMenuDescription(screen)
		return
	}

	statusX := menuStatusOffsetX
	m.drawCommandList(screen)

	for i := 0; i < 4; i++ {
		itemY := menuStatusStartY + float64(i)*menuStatusSpacingY

		textColor := uiColorText
		inSkillCharFlow := m.menuState == menuStateSkillCharSel ||
			m.menuState == menuStateSkillSub
		if inSkillCharFlow && i == m.skillCharIndex {
			textColor = uiColorSelect
		}
		if m.menuState == menuStateHealTarget &&
			(m.healTargetIndex == i || m.healTargetIndex == partySize) {
			textColor = uiColorSelect
		}
		if m.menuState == menuStateItemTarget &&
			(m.itemTargetIndex == i || m.itemTargetIndex == partySize) {
			textColor = uiColorSelect
		}
		if m.game.PlayerHP[i] <= 0 {
			textColor = uiColorDead
		}
		alpha := 1.0
		if m.game.PlayerHP[i] <= 0 {
			alpha = 0.4
		}

		img := m.game.PartyIconImgs[i]
		iw := float64(img.Bounds().Dx())
		ih := float64(img.Bounds().Dy())
		iconOp := &ebiten.DrawImageOptions{}
		iconOp.GeoM.Translate(statusX-iw+10, itemY-ih/2+35)
		if m.game.PlayerHP[i] <= 0 {
			iconOp.ColorScale.Scale(1, 1, 1, float32(alpha))
		}
		screen.DrawImage(img, iconOp)

		prefix := "  "
		if inSkillCharFlow && m.skillCharIndex == i {
			prefix = "▶ "
		}
		if m.menuState == menuStateHealTarget &&
			(m.healTargetIndex == i || m.healTargetIndex == partySize) {
			prefix = "▶ "
		}
		if m.menuState == menuStateItemTarget &&
			(m.itemTargetIndex == i || m.itemTargetIndex == partySize) {
			prefix = "▶ "
		}
		prefixOp := &text.DrawOptions{}
		prefixOp.GeoM.Translate(statusX+menuStatusCursorX, itemY)
		prefixOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, prefix, m.game.FontFace(menuStatusFontSize), prefixOp)

		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(statusX+menuStatusNameX, itemY)
		nameOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, PlayerNames[i], m.game.FontFace(menuStatusFontSize), nameOp)

		levelOp := &text.DrawOptions{}
		levelOp.GeoM.Translate(statusX+menuStatusLevelX, itemY)
		levelOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, fmt.Sprintf("Lv %d", m.game.PlayerLv[i]), m.game.FontFace(menuStatusFontSize), levelOp)

		drawStatusValue(screen,
			statusX+20, itemY+18,
			m.game.PlayerHP[i], m.game.PlayerMaxHP[i],
			m.game.FontFace(17.5), m.game.FontFace(14), alpha, uiColorText)

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
			m.game.FontFace(17.5), m.game.FontFace(14), alpha, uiColorText)

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

	m.drawAllTargetRow(screen, statusX)

	skillPanelVisible := m.menuState == menuStateSkillSub || m.menuState == menuStateHealTarget
	itemPanelVisible := m.menuState == menuStateItemList || m.menuState == menuStateItemTarget

	if !skillPanelVisible && !itemPanelVisible {
		m.drawMinimap(screen)
	}

	if skillPanelVisible {
		m.drawSkillSubMenu(screen)
	}

	if itemPanelVisible {
		m.drawItemListMenu(screen)
	}

	if m.showReturnTitleConfirm {
		drawConfirmDialog(screen, m.game, "タイトルに戻りますか？", m.confirmIndex, confirmImageOffsetX)
	}

	m.drawMenuDescription(screen)
}

const (
	skillNameX        = 480.0
	skillRowStartY    = 95.0
	skillRowGapY      = 55.0
	skillNameFontSize = 20.0

	skillLevelStartX       = 720.0
	skillLevelGapX         = 90.0
	skillLevelOffsetY      = 0.0
	skillLevelFontSize     = 40.0
	skillLevelArrowOffsetX = 30.0
	skillGaugeOffsetY      = 20.0
	skillGaugeWidth        = 40.0
	skillGaugeHeight       = 10.0
	skillRequiredSPOffsetX = 12.0
	skillRequiredSPOffsetY = 15.0
	skillBottomFontSize    = 16.0

	skillSPHeaderX = 920.0
	skillSPHeaderY = 20.0
)

func (m *MenuScene) drawSkillSubMenu(screen *ebiten.Image) {
	spHeaderOp := &text.DrawOptions{}
	spHeaderOp.GeoM.Translate(skillSPHeaderX, skillSPHeaderY)
	spHeaderOp.PrimaryAlign = text.AlignEnd
	spHeaderOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("所持SP: %d", m.game.PlayerSP[m.skillCharIndex]), m.game.FontFace(skillBottomFontSize), spHeaderOp)

	skills := m.game.CharacterSkills(m.skillCharIndex)

	for i, sk := range skills {
		if len(sk.Levels) == 0 {
			continue
		}
		curLv := m.game.PlayerSkillLv[m.skillCharIndex][i]
		if curLv < 1 {
			curLv = 1
		}
		if curLv > len(sk.Levels) {
			curLv = len(sk.Levels)
		}

		rowCenterY := skillRowStartY + float64(i)*skillRowGapY
		rowSelected := i == m.skillSubIndex

		curData := sk.Levels[curLv-1]
		usable := curData.IsHeal && m.game.PlayerMP[m.skillCharIndex] >= curData.MPCost

		nameCol := uiColorText
		if !usable {
			nameCol = uiColorDisabled
		}
		showNameArrow := false
		if rowSelected {
			nameCol = uiColorSelect
			if !m.skillLevelSelecting {
				showNameArrow = true
			}
		}

		nameFace := m.game.FontFace(skillNameFontSize)
		if showNameArrow {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(skillNameX, rowCenterY)
			arrowOp.SecondaryAlign = text.AlignCenter
			arrowOp.ColorScale.ScaleWithColor(nameCol)
			text.Draw(screen, "▶", nameFace, arrowOp)
		}
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(skillNameX+text.Advance("▶ ", nameFace), rowCenterY)
		nameOp.SecondaryAlign = text.AlignCenter
		nameOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, sk.Name, nameFace, nameOp)

		for lv := 1; lv <= len(sk.Levels); lv++ {
			numX := skillLevelStartX + float64(lv-1)*skillLevelGapX
			col := color.RGBA{120, 120, 120, 255}
			label := fmt.Sprintf("%d", lv)
			if lv <= curLv {
				col = uiColorText
			}

			selected := rowSelected && m.skillLevelSelecting && lv == m.skillLevelCursor

			numCenterY := rowCenterY + skillLevelOffsetY
			numOp := &text.DrawOptions{}
			numOp.GeoM.Translate(numX, numCenterY)
			numOp.PrimaryAlign = text.AlignCenter
			numOp.SecondaryAlign = text.AlignCenter
			numOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, label, m.game.FontFace(skillLevelFontSize), numOp)

			if lv > curLv {
				gaugeX := numX - skillGaugeWidth/2
				gaugeY := numCenterY + skillGaugeOffsetY
				ebitenutil.DrawRect(screen, gaugeX, gaugeY, skillGaugeWidth, skillGaugeHeight, color.RGBA{45, 45, 55, 255})
				gaugeRatio := 0.0
				if selected {
					gaugeRatio = m.upgradeProgress
				}
				ebitenutil.DrawRect(screen, gaugeX, gaugeY, skillGaugeWidth*gaugeRatio, skillGaugeHeight, uiColorSelect)
			}

			if lv > curLv {
				requiredSP := "-"
				if cost := SkillUpgradeCost(lv - 1); cost > 0 {
					requiredSP = fmt.Sprintf("%d", cost)
				}
				spOp := &text.DrawOptions{}
				spOp.GeoM.Translate(numX+skillRequiredSPOffsetX, numCenterY+skillRequiredSPOffsetY)
				spOp.ColorScale.ScaleWithColor(uiColorText)
				text.Draw(screen, requiredSP, m.game.FontFace(skillBottomFontSize), spOp)
			}

			if selected {
				arrowOp := &text.DrawOptions{}
				arrowOp.GeoM.Translate(numX-skillLevelArrowOffsetX, numCenterY)
				arrowOp.SecondaryAlign = text.AlignCenter
				arrowOp.ColorScale.ScaleWithColor(uiColorSelect)
				text.Draw(screen, "▶", m.game.FontFace(skillNameFontSize), arrowOp)
			}
		}
	}
}
