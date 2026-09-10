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
	menuStatusCursorX  = 0.0
	menuStatusNameX    = 20.0
	menuStatusLevelX   = 135.0
	menuBarW           = 160.0
)

// 左側のコマンド一覧（アイテム/スキル/ステータス/セーブ/ロード/オプション/タイトルに戻る）の文字サイズ・位置。
// メイン画面での選択可能表示と、他画面で背景として出すグレー表示の両方がこの値を共有する。
const (
	cmdListFontSize = 20.0
	cmdListStartX   = 20.0
	cmdListStartY   = 30.0
	cmdListRowGapY  = 40.0
)

// メニュー背景の区画線。元のメニュー画面.pngに焼き込まれていた白線と同じ位置になるよう、
// 元画像（960x540）を基準に計測した座標。線の太さは全て menuLineWidth で統一する。
const (
	menuFrameLeft   = 15.0
	menuFrameTop    = 8.0
	menuFrameRight  = gameWidth - menuFrameLeft // 左右の隙間を同じ数値にする
	menuFrameBottom = gameHeight - menuFrameTop // 上下の隙間を同じ数値にする

	menuFrameDividerX = 203.0 // 左カラムと右側メイン領域を分ける縦線
	menuFrameDividerY = 364.0 // 左カラムの上下ボックスを分ける横線

	menuLineWidth = 2.0
)

// drawMenuBgFrame はメニュー背景に重ねる区画線（外枠・左カラムの仕切り線）を描画する。
// 元画像の線ににじみ・太さのばらつきがあったため、太さ menuLineWidth で統一し、
// 角・交点で隙間なくつながるように矩形を少し重ねて描画する。
func drawMenuBgFrame(screen *ebiten.Image) {
	w := menuLineWidth

	// 外枠（上下は幅いっぱいに、左右はその内側の高さいっぱいに描いて角を隙間なくつなげる）
	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameTop, menuFrameRight-menuFrameLeft, w, uiColorMenuLine)
	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameBottom-w, menuFrameRight-menuFrameLeft, w, uiColorMenuLine)
	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameTop, w, menuFrameBottom-menuFrameTop, uiColorMenuLine)
	ebitenutil.DrawRect(screen, menuFrameRight-w, menuFrameTop, w, menuFrameBottom-menuFrameTop, uiColorMenuLine)

	// 左カラムを縦に貫く仕切り線（上下端は外枠と重ねてつなげる）
	ebitenutil.DrawRect(screen, menuFrameDividerX-w/2, menuFrameTop, w, menuFrameBottom-menuFrameTop, uiColorMenuLine)

	// 左カラム内を上下に分ける仕切り線（左端は外枠、右端は縦の仕切り線と重ねてつなげる）
	ebitenutil.DrawRect(screen, menuFrameLeft, menuFrameDividerY-w/2, menuFrameDividerX-menuFrameLeft+w/2, w, uiColorMenuLine)
}

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
	drawMenuBgFrame(screen)

	switch m.menuState {

	case menuStateStatus:
		bgCmdFace := m.game.FontFace(cmdListFontSize)
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(cmdListStartX+text.Advance("▶ ", bgCmdFace), cmdListStartY+float64(i)*cmdListRowGapY)
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, cmdName, bgCmdFace, cmdOp)
		}
		m.drawStatusScreen(screen)
		m.drawMenuDescription(screen)
		return

	case menuStateSaveConfirm, menuStateLoadConfirm:
		bgCmdFace := m.game.FontFace(cmdListFontSize)
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(cmdListStartX+text.Advance("▶ ", bgCmdFace), cmdListStartY+float64(i)*cmdListRowGapY)
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, cmdName, bgCmdFace, cmdOp)
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
		bgCmdFace := m.game.FontFace(cmdListFontSize)
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(cmdListStartX+text.Advance("▶ ", bgCmdFace), cmdListStartY+float64(i)*cmdListRowGapY)
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, cmdName, bgCmdFace, cmdOp)
		}
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, slotsPerPageView-1, slotCardStartX, slotCardStartY)
		drawConfirmDialog(screen, m.game, m.saveResultMsg, 0, confirmImageOffsetX, false)
		m.drawMenuDescription(screen)
		return

	case menuStateSaveSlot, menuStateLoadSlot:
		bgCmdFace := m.game.FontFace(cmdListFontSize)
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(cmdListStartX+text.Advance("▶ ", bgCmdFace), cmdListStartY+float64(i)*cmdListRowGapY)
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, cmdName, bgCmdFace, cmdOp)
		}
		drawSlotList(screen, m.game, m.slotIndex, m.slotData, m.slotThumbs, m.saveMode, slotsPerPageView-1, slotCardStartX, slotCardStartY)
		m.drawMenuDescription(screen)
		return

	case menuStateOption, menuStateOptionAdjust, menuStateMessageSpeedAdjust, menuStateDisplayModeAdjust:
		bgCmdFace := m.game.FontFace(cmdListFontSize)
		for i, cmdName := range m.commands {
			cmdOp := &text.DrawOptions{}
			cmdOp.GeoM.Translate(cmdListStartX+text.Advance("▶ ", bgCmdFace), cmdListStartY+float64(i)*cmdListRowGapY)
			cmdOp.ColorScale.ScaleWithColor(uiColorText)
			text.Draw(screen, cmdName, bgCmdFace, cmdOp)
		}
		m.drawVolumePanel(screen)
		m.drawMenuDescription(screen)
		return
	}

	statusX := menuStatusOffsetX
	cmdFace := m.game.FontFace(cmdListFontSize)
	cmdArrowGap := text.Advance("▶ ", cmdFace)
	for i, cmdName := range m.commands {
		baseX, baseY := cmdListStartX, cmdListStartY+float64(i)*cmdListRowGapY
		col := uiColorText
		if i == m.menuIndex && m.menuState == menuStateMain {
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
				prefix = "▶ "
			} else if m.healTargetIndex == i {
				prefix = "▶ "
			}
		}
		if m.menuState == menuStateItemTarget {
			if m.itemTargetIndex == partySize || m.itemTargetIndex == i {
				prefix = "▶ "
			}
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

	m.drawMinimap(screen)

	if m.menuState == menuStateSkillSub {
		m.drawSkillSubMenu(screen)
	}

	if m.menuState == menuStateItemList {
		m.drawItemListMenu(screen)
	}

	if m.menuState == menuStateReturnTitleConfirm {
		drawConfirmDialog(screen, m.game, "タイトルに戻りますか？", m.confirmIndex, confirmImageOffsetX)
	}

	m.drawMenuDescription(screen)
}

// スキルパネル専用の表示開始位置（左上基準）
// スキルパネル専用の表示開始位置（左上基準）
// スキルパネル専用の表示開始位置（左上基準）
const (
	skillSubPanelX = 250.0
	skillSubPanelY = 100.0

	skillSubNameOffsetX     = 20.0
	skillSubRowStartOffsetY = 35.0 // 1行目の「行の中心Y」（ここを基準に名前・レベル数字とも縦中央揃え）
	skillSubRowGapY         = 55.0 // 行間（1行あたりの高さ）

	// レベル数字、ゲージ、必要MPの配置とサイズ。
	skillSubLevelStartX       = 350.0
	skillSubLevelGapX         = 75.0
	skillSubLevelOffsetY      = 0.0
	skillSubLevelFontSize     = 40.0
	skillSubLevelArrowOffsetX = 30.0
	skillSubGaugeOffsetY      = 20.0
	skillSubGaugeWidth        = 40.0
	skillSubGaugeHeight       = 10.0
	skillSubRequiredSPOffsetX = 23.0
	skillSubRequiredSPOffsetY = 15.0
	skillSubBottomFontSize    = 16.0

	// ── 所持SPの表示位置 ──
	skillSubSPOffsetX = 550.0 // 所持SPのX

	// ── 説明文：別Xグループ ──
	skillSubDescOffsetX = 10.0

	// ── 所持SP／説明文で共通して使う、パネル下端からのYオフセット ──
	skillSubBottomOffsetY = 20.0
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

	for i, sk := range skills {
		if len(sk.Levels) == 0 {
			continue
		}
		curLv := m.game.PlayerSkillLv[m.skillCharIndex][i]
		if curLv < 1 {
			curLv = 1
		}

		rowCenterY := winY + skillSubRowStartOffsetY + float64(i)*skillSubRowGapY // ← 変更：行の中心Y
		rowSelected := i == m.skillSubIndex

		nameCol := uiColorText
		showNameArrow := false
		if rowSelected {
			nameCol = uiColorSelect
			if !m.skillLevelSelecting {
				showNameArrow = true
			}
		}

		// ★変更：スキル名は行の中心Yを基準に縦中央揃えで描画する（数字側と高さの中心を合わせるため）。
		// 矢印とスキル名は別々に描画し、スキル名の開始X座標を固定することで選択有無による文字ズレを防ぐ。
		nameFace := m.game.FontFace(20)
		if showNameArrow {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(winX+skillSubNameOffsetX, rowCenterY)
			arrowOp.SecondaryAlign = text.AlignCenter
			arrowOp.ColorScale.ScaleWithColor(nameCol)
			text.Draw(screen, "▶", nameFace, arrowOp)
		}
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(winX+skillSubNameOffsetX+text.Advance("▶ ", nameFace), rowCenterY)
		nameOp.SecondaryAlign = text.AlignCenter
		nameOp.ColorScale.ScaleWithColor(nameCol)
		text.Draw(screen, sk.Name, nameFace, nameOp)

		for lv := 1; lv <= len(sk.Levels); lv++ {
			numX := winX + skillSubLevelStartX + float64(lv-1)*skillSubLevelGapX
			col := color.RGBA{120, 120, 120, 255}
			label := fmt.Sprintf("%d", lv)
			if lv <= curLv {
				col = uiColorText
			}

			selected := rowSelected && m.skillLevelSelecting && lv == m.skillLevelCursor

			numCenterY := rowCenterY + skillSubLevelOffsetY
			numOp := &text.DrawOptions{}
			numOp.GeoM.Translate(numX, numCenterY)
			numOp.PrimaryAlign = text.AlignCenter
			numOp.SecondaryAlign = text.AlignCenter
			numOp.ColorScale.ScaleWithColor(col)
			text.Draw(screen, label, m.game.FontFace(skillSubLevelFontSize), numOp)

			if lv > curLv {
				gaugeX := numX - skillSubGaugeWidth/2
				gaugeY := numCenterY + skillSubGaugeOffsetY
				ebitenutil.DrawRect(screen, gaugeX, gaugeY, skillSubGaugeWidth, skillSubGaugeHeight, color.RGBA{45, 45, 55, 255})
				gaugeRatio := 0.0
				if selected {
					gaugeRatio = m.upgradeProgress
				}
				ebitenutil.DrawRect(screen, gaugeX, gaugeY, skillSubGaugeWidth*gaugeRatio, skillSubGaugeHeight, uiColorSelect)
			}

			if lv > curLv {
				requiredSP := "-"
				if cost := SkillUpgradeCost(lv - 1); cost > 0 {
					requiredSP = fmt.Sprintf("%d", cost)
				}
				spOp := &text.DrawOptions{}
				spOp.GeoM.Translate(numX+skillSubRequiredSPOffsetX, numCenterY+skillSubRequiredSPOffsetY)
				spOp.ColorScale.ScaleWithColor(uiColorText)
				text.Draw(screen, requiredSP, m.game.FontFace(skillSubBottomFontSize), spOp)
			}

			if selected {
				// ★変更：Lv選択カーソルの矢印は、スキル名の選択矢印(▶)と全く同じ描き方にする。
				arrowOp := &text.DrawOptions{}
				arrowOp.GeoM.Translate(numX-skillSubLevelArrowOffsetX, numCenterY)
				arrowOp.SecondaryAlign = text.AlignCenter
				arrowOp.ColorScale.ScaleWithColor(uiColorSelect)
				text.Draw(screen, "▶", m.game.FontFace(20), arrowOp)
			}
		}
	}

	spOp := &text.DrawOptions{}
	spOp.GeoM.Translate(winX+skillSubSPOffsetX, winY+winH-skillSubBottomOffsetY)
	spOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, fmt.Sprintf("所持SP: %d", m.game.PlayerSP[m.skillCharIndex]), m.game.FontFace(skillSubBottomFontSize), spOp)

}
