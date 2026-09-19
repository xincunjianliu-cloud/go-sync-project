package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// メニューのステータスボックスは、位置(x/y/行間/レベルx/アイコン位置)以外
// すべてバトル側の定数・関数をそのまま使う。数値/バーはdrawStatusBox
// （battle_draw_panels.go、statusValueOffsetX / statusHPTextY / statusHPBarY /
// statusMPTextY / statusMPBarY / statusBlockW など battle_types.go 参照）、
// 名前のフォントサイズもpartyNameFontSize（battle_types.go）を流用する。
// メニュー画面独自に定数化してよいのは以下の6つ（ブロックの位置x/y、行間、
// レベルのx位置、アイコンの位置x/y）だけで、それ以外はここで新しく定数や
// 描画ロジックを作らないこと。カーソル「▶」は名前の描画位置から自動で
// 逆算して置くので、個別の位置調整は不要（かつ用意しない）。
const (
	menuStatusOffsetX  = 280.0
	menuStatusStartY   = 40.0
	menuStatusSpacingY = 116.0
	menuStatusLevelX   = 110.0

	// アイコン画像のオフセット（右端合わせ/縦中央合わせの基準座標からのずれ）。
	menuStatusIconOffsetX = -5.0
	menuStatusIconOffsetY = 38.0
)

const (
	cmdListFontSize = 23.0
	cmdListStartX   = 20.0
	cmdListStartY   = 30.0
	cmdListRowGapY  = 40.0
)

const (
	menuFrameLeft   = 15.0
	menuFrameTop    = 8.0
	menuFrameRight  = gameWidth - menuFrameLeft
	menuFrameBottom = gameHeight - menuFrameTop

	menuFrameDividerX = 190.0
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
	confirmFontSize         = 22.0

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
	boxX := statusX - iconW + menuStatusIconOffsetX + drawAllTargetRowBoxInset
	boxRight := statusX + partyNameOffsetX + statusBlockW - drawAllTargetRowBoxInset
	boxW := boxRight - boxX
	boxY := y - drawAllTargetRowBoxH/2

	ebitenutil.DrawRect(screen, boxX, boxY, boxW, drawAllTargetRowBoxH, boxFillCol)

	nameFace := m.game.FontFace(partyNameFontSize)
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

		isDead := m.game.PlayerHP[i] <= 0

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
		if isDead {
			textColor = uiColorDead
		}
		alpha := 1.0
		if isDead {
			alpha = 0.4
		}

		img := m.game.PartyIconImgs[i]
		iw := float64(img.Bounds().Dx())
		ih := float64(img.Bounds().Dy())
		iconOp := &ebiten.DrawImageOptions{}
		iconOp.GeoM.Translate(statusX-iw+menuStatusIconOffsetX, itemY-ih/2+menuStatusIconOffsetY)
		if isDead {
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
		// 名前・カーソル・レベルはすべて同じフォント(partyNameFontSize、
		// battle_types.go)と同じ基準（partyNameOffsetX/Y、常に0、縦方向の
		// アラインも指定しない＝バトルのdrawBattleOutlinedTextと同じ既定値）
		// をstatusX/itemYに適用しただけの位置（個別の位置調整はしない）。
		nameFace := m.game.FontFace(partyNameFontSize)
		nameX := statusX + partyNameOffsetX
		nameY := itemY + partyNameOffsetY

		prefixOp := &text.DrawOptions{}
		prefixOp.GeoM.Translate(nameX-text.Advance(prefix, nameFace), nameY)
		prefixOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, prefix, nameFace, prefixOp)

		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(nameX, nameY)
		nameOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, PlayerNames[i], nameFace, nameOp)

		levelOp := &text.DrawOptions{}
		levelOp.GeoM.Translate(statusX+menuStatusLevelX, itemY+partyNameOffsetY)
		levelOp.ColorScale.ScaleWithColor(textColor)
		text.Draw(screen, fmt.Sprintf("Lv %d", m.game.PlayerLv[i]), nameFace, levelOp)

		valueColor := uiColorText
		if isDead {
			valueColor = uiColorDead
		}

		drawStatusBox(screen, m.game, statusX, itemY,
			m.game.PlayerHP[i], m.game.PlayerMaxHP[i],
			m.game.PlayerMP[i], m.game.PlayerMaxMP[i],
			alpha, valueColor, isDead)
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
	skillNameFontSize = 25.0

	skillLevelStartX       = 710.0
	skillLevelGapX         = 90.0
	skillLevelOffsetY      = 0.0
	skillLevelFontSize     = 45.0
	skillLevelArrowOffsetX = 30.0
	skillGaugeOffsetY      = 20.0
	skillGaugeWidth        = 40.0
	skillGaugeHeight       = 10.0
	skillRequiredSPOffsetX = 23.0
	skillRequiredSPOffsetY = 18.0
	skillBottomFontSize    = 15.0
	skillSPHeaderFontSize  = 20.0

	skillSPHeaderX = 920.0
	skillSPHeaderY = 20.0
)

func (m *MenuScene) drawSkillSubMenu(screen *ebiten.Image) {
	spHeaderOp := &text.DrawOptions{}
	spHeaderOp.GeoM.Translate(skillSPHeaderX, skillSPHeaderY)
	spHeaderOp.PrimaryAlign = text.AlignEnd
	spHeaderOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("所持SP: %d", m.game.PlayerSP[m.skillCharIndex]), m.game.FontFace(skillSPHeaderFontSize), spHeaderOp)

	skills := m.game.CharacterSkills(m.skillCharIndex)

	for i, sk := range skills {
		if len(sk.Levels) == 0 {
			continue
		}
		if !m.game.IsSkillUnlocked(m.skillCharIndex, i) {
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
