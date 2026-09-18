package main

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (s *BattleScene) drawTargetSelectUI(screen *ebiten.Image) {
	isAll := s.currentAttackIsAllTarget()

	if s.currentTargetAllowsAll() {
		s.drawEnemyAllTargetRow(screen, isAll)
	}

	for i := range s.enemies {
		if s.enemies[i].HP <= 0 || s.enemies[i].Image == nil {
			continue
		}
		x, y, _, h := s.enemyDrawRect(i)

		showArrow := false
		if isAll {
			showArrow = true
		} else {
			showArrow = s.targetIndex == i
		}
		if !showArrow {
			continue
		}

		cursorX := x - 24.0
		cursorY := y + h/2 - 8.0

		op := &text.DrawOptions{}
		op.GeoM.Translate(cursorX, cursorY)
		op.ColorScale.ScaleWithColor(uiColorSelect)
		text.Draw(screen, "▶", s.game.FontFace(15), op)
	}
}

func (s *BattleScene) drawHealTargetUI(screen *ebiten.Image) {
	isAll := s.healTargetIndex == partySize

	s.drawAllTargetRow(screen, isAll)

	for i := 0; i < partySize; i++ {
		centerX := s.partyScreenX[i]
		centerY := s.partyScreenY[i]

		showArrow := false
		if isAll {
			showArrow = s.game.PlayerHP[i] > 0
		} else {
			showArrow = s.healTargetIndex == i
		}

		if showArrow {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(centerX-4, centerY+20)
			arrowOp.ColorScale.ScaleWithColor(uiColorSelect)
			text.Draw(screen, "▶", s.game.FontFace(15), arrowOp)
		}
	}
}

func (s *BattleScene) targetSelectDescriptionAndHint() (string, string) {
	p := s.waitingActor
	if p < 0 || p >= partySize || s.pendingSkill < 1 {
		return "", ""
	}
	skillIdx := s.pendingSkill - 1
	skills := s.game.CharacterSkills(p)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return "", ""
	}
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 {
		lv = 1
	}
	if lv > len(skills[skillIdx].Levels) {
		lv = len(skills[skillIdx].Levels)
	}
	data := skills[skillIdx].Levels[lv-1]

	hint := ""
	if data.Target == TargetBoth && s.currentTargetAllowsAll() {
		if s.selectedSkillTarget == TargetAll {
			hint = "←:単体に切替"
		} else {
			hint = "→:全体に切替"
		}
	}
	return data.Description, hint
}

func (s *BattleScene) drawResultPanel(screen *ebiten.Image) {

	panelAlpha := float32(1.0)
	if s.resultSubPhase == resSubResultFadeIn {
		panelAlpha = float32(s.resultAnimTimer / 0.5)
		if panelAlpha > 1.0 {
			panelAlpha = 1.0
		}
	}

	panelW := float64(gameWidth) * resultPanelWidthRatio
	panelH := float64(gameHeight)
	ebitenutil.DrawRect(screen, 0, 0, panelW, panelH,
		scaleAlpha(resultPanelBgColor, float64(panelAlpha)))
	if s.resultFadeAlpha > 0 {
		alphaByte := uint8(255 * s.resultFadeAlpha)
		ebitenutil.DrawRect(screen, 0, 0, panelW, panelH, color.RGBA{0, 0, 0, alphaByte})
	}

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(resultTitleX, resultTitleY)
	titleOp.ColorScale.ScaleWithColor(uiColorText)
	titleOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "Battle Results", s.game.FontFace(resultTitleFontSize), titleOp)

	expLabelOp := &text.DrawOptions{}
	expLabelOp.GeoM.Translate(resultExpLabelX, resultExpY)
	expLabelOp.ColorScale.ScaleWithColor(uiColorText)
	expLabelOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "EXP", s.game.FontFace(resultExpFontSize), expLabelOp)

	expValueOp := &text.DrawOptions{}
	expValueOp.GeoM.Translate(resultExpValueX, resultExpY)
	expValueOp.ColorScale.ScaleWithColor(uiColorText)
	expValueOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, fmt.Sprintf("%d", s.totalEnemyExp()), s.game.FontFace(resultExpFontSize), expValueOp)

	spLabelOp := &text.DrawOptions{}
	spLabelOp.GeoM.Translate(resultSpLabelX, resultSpY)
	spLabelOp.ColorScale.ScaleWithColor(uiColorText)
	spLabelOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "SP", s.game.FontFace(resultSpFontSize), spLabelOp)

	spValueOp := &text.DrawOptions{}
	spValueOp.GeoM.Translate(resultSpValueX, resultSpY)
	spValueOp.ColorScale.ScaleWithColor(uiColorText)
	spValueOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, fmt.Sprintf("%d", s.totalEnemySP()), s.game.FontFace(resultSpFontSize), spValueOp)

	ebitenutil.DrawRect(screen, resultDividerX, resultDividerY, resultDividerW, resultDividerH, uiColorText)

	barW := resultBarWAbs

	for i := 0; i < partySize; i++ {
		nameColor := uiColorText
		if s.game.PlayerHP[i] <= 0 {
			nameColor = uiColorDead
		}

		barX := resultBarStartX
		barY := resultBarStartY + float64(i)*resultBarRowGap
		barH := resultBarH

		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(barX+resultNameOffsetX, barY+resultNameOffsetY)
		nameOp.ColorScale.ScaleWithColor(nameColor)
		nameOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, PlayerNames[i], s.game.FontFace(resultNameFontSize), nameOp)

		levelOp := &text.DrawOptions{}
		levelOp.GeoM.Translate(barX+resultLevelOffsetX, barY+resultNameOffsetY)
		levelOp.ColorScale.ScaleWithColor(nameColor)
		levelOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, fmt.Sprintf("Lv %d", s.drawPlayerLv[i]), s.game.FontFace(resultNameFontSize), levelOp)

		curExpStr := strconv.Itoa(s.drawPlayerEXP[i])
		maxExpStr := fmt.Sprintf("/%d", s.drawPlayerMaxEXP[i])
		expRightX := barX + barW
		expBaseY := barY + resultExpTextOffsetY + resultExpCurFontSize

		maxOp := &text.DrawOptions{}
		maxOp.PrimaryAlign = text.AlignEnd
		maxOp.SecondaryAlign = text.AlignEnd
		maxOp.GeoM.Translate(expRightX, expBaseY)
		maxOp.ColorScale.ScaleWithColor(uiColorText)
		maxOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, maxExpStr, s.game.FontFace(resultExpMaxFontSize), maxOp)

		maxExpW, _ := text.Measure(maxExpStr, s.game.FontFace(resultExpMaxFontSize), 0)

		curOp := &text.DrawOptions{}
		curOp.PrimaryAlign = text.AlignEnd
		curOp.SecondaryAlign = text.AlignEnd
		curOp.GeoM.Translate(expRightX-maxExpW, expBaseY)
		curOp.ColorScale.ScaleWithColor(uiColorText)
		curOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, curExpStr, s.game.FontFace(resultExpCurFontSize), curOp)

		expLabelOp2 := &text.DrawOptions{}
		expLabelOp2.GeoM.Translate(barX+resultExpLabelOffsetX, barY+resultExpLabelOffsetY)
		expLabelOp2.ColorScale.ScaleWithColor(uiColorText)
		expLabelOp2.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, "EXP", s.game.FontFace(resultExpLabelFontSize), expLabelOp2)

		if s.resultSubPhase >= resSubBarAnimate {
			ebitenutil.DrawRect(screen, barX, barY, barW, barH, resultBarBgColor)
			if s.game.PlayerHP[i] > 0 && s.drawPlayerMaxEXP[i] > 0 {
				ratio := s.drawPlayerEXPF[i] / float64(s.drawPlayerMaxEXP[i])
				if ratio > 1.0 {
					ratio = 1.0
				}
				if ratio < 0.0 {
					ratio = 0.0
				}
				if int(barW*ratio) >= 1 {
					ebitenutil.DrawRect(screen, barX, barY, barW*ratio, barH, resultBarFillColor)
				}
			}
		}

		if s.isLevelUp[i] && s.game.PlayerHP[i] > 0 && s.drawPlayerLv[i] >= s.game.PlayerLv[i] {
			lvOp := &text.DrawOptions{}
			lvOp.GeoM.Translate(barX+resultLevelUpOffsetX, barY+resultLevelUpOffsetY)
			lvOp.ColorScale.ScaleWithColor(resultLevelUpColor)
			lvOp.ColorScale.ScaleAlpha(panelAlpha)
			text.Draw(screen, "LEVEL UP!", s.game.FontFace(resultLevelUpFontSize), lvOp)
		}
	}

	headerOp := &text.DrawOptions{}
	headerOp.GeoM.Translate(resultItemsHeaderX, resultItemsHeaderY)
	headerOp.ColorScale.ScaleWithColor(uiColorText)
	headerOp.ColorScale.ScaleAlpha(panelAlpha)
	text.Draw(screen, "獲得アイテム", s.game.FontFace(resultItemsHeaderFontSize), headerOp)

	ebitenutil.DrawRect(screen, resultItemsDividerX, resultItemsDividerY, resultItemsDividerW, resultItemsDividerH,
		scaleAlpha(resultItemsDividerColor, float64(panelAlpha)))

	for i, item := range s.earnedItems {
		rowY := resultItemsStartY + float64(i)*resultItemsRowGap

		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(resultItemsNameX, rowY)
		nameOp.ColorScale.ScaleWithColor(uiColorText)
		nameOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, item.Name, s.game.FontFace(resultItemsFontSize), nameOp)

		countOp := &text.DrawOptions{}
		countOp.PrimaryAlign = text.AlignEnd
		countOp.GeoM.Translate(resultItemsCountX, rowY)
		countOp.ColorScale.ScaleWithColor(uiColorText)
		countOp.ColorScale.ScaleAlpha(panelAlpha)
		text.Draw(screen, fmt.Sprintf("x%d", item.Count), s.game.FontFace(resultItemsFontSize), countOp)
	}

	if s.resultSubPhase == resSubDoneWait {
		hintOp := &text.DrawOptions{}
		hintOp.GeoM.Translate(resultHintX, float64(gameHeight)-resultHintYFromBtm)
		hintOp.ColorScale.ScaleWithColor(uiColorText)
		text.Draw(screen, "Enter / Space / Z で進む", s.game.FontFace(resultHintFontSize), hintOp)
	}
}

func (s *BattleScene) drawControlHint(screen *ebiten.Image, hint string) {
	if hint == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(gameWidth)/2, hintY)
	op.PrimaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, hint, s.game.FontFace(15), op)
}

func (s *BattleScene) drawBottomDescription(screen *ebiten.Image, desc string, hint string) {
	descOp := &text.DrawOptions{}
	descOp.GeoM.Translate(descX, descY)
	descOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, desc, s.game.FontFace(descFontSize*descScale), descOp)

	if hint == "" {
		return
	}

	hintOp := &text.DrawOptions{}
	hintOp.GeoM.Translate(descX+hintOffsetX, descY)
	hintOp.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, hint, s.game.FontFace(hintFontSize*hintScale), hintOp)
}

func scaleAlpha(c color.RGBA, factor float64) color.RGBA {
	c.A = uint8(float64(c.A) * factor)
	return c
}

func drawStatusValue(screen *ebiten.Image, x, y float64, current, max int, faceLarge, faceSmall *text.GoTextFace, alpha float64, textColorBase color.RGBA) {
	curStr := strconv.Itoa(current)
	restStr := fmt.Sprintf(" / %d", max)

	for _, offset := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(x+offset[0], y+offset[1])
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, uint8(255 * alpha)})
		text.Draw(screen, curStr, faceLarge, shadowOp)

		restShadowOp := &text.DrawOptions{}
		restShadowOp.GeoM.Translate(x+text.Advance(curStr, faceLarge)+offset[0], y+2.5+offset[1])
		restShadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, uint8(255 * alpha)})
		text.Draw(screen, restStr, faceSmall, restShadowOp)
	}

	textColor := textColorBase
	textColor.A = uint8(255 * alpha)

	curOp := &text.DrawOptions{}
	curOp.GeoM.Translate(x, y)
	curOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, curStr, faceLarge, curOp)
	restX := x + text.Advance(curStr, faceLarge)
	restOp := &text.DrawOptions{}
	restOp.GeoM.Translate(restX, y+2.5)
	restOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, restStr, faceSmall, restOp)
}

func drawBattleOutlinedText(screen *ebiten.Image, x, y float64, value string, face *text.GoTextFace, textColor color.RGBA, alpha float64) {
	for _, offset := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		shadowOp := &text.DrawOptions{}
		shadowOp.GeoM.Translate(x+offset[0], y+offset[1])
		shadowOp.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, uint8(255 * alpha)})
		text.Draw(screen, value, face, shadowOp)
	}

	textOp := &text.DrawOptions{}
	textOp.GeoM.Translate(x, y)
	textColor.A = uint8(float64(textColor.A) * alpha)
	textOp.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, value, face, textOp)
}

func fillSlantedPath(screen *ebiten.Image, path *vector.Path, c color.RGBA) {
	drawOpts := &vector.DrawPathOptions{AntiAlias: false}
	drawOpts.ColorScale.ScaleWithColor(c)
	vector.FillPath(screen, path, nil, drawOpts)
}
func drawSlantedQuad(screen *ebiten.Image, x, y, w, h, slant float64, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	var path vector.Path
	path.MoveTo(float32(x+slant), float32(y))
	path.LineTo(float32(x+w), float32(y))
	path.LineTo(float32(x+w-slant), float32(y+h))
	path.LineTo(float32(x), float32(y+h))
	path.Close()
	fillSlantedPath(screen, &path, c)
}
func drawSlantedStatusBar(screen *ebiten.Image, x, y, w, h, slant, ratio float64, fill, empty, edge color.RGBA) {
	drawSlantedQuad(screen, x-1, y-1, w+2, h+2, slant, color.RGBA{0, 0, 0, empty.A})

	innerSpan := w - slant
	if innerSpan <= 0 {
		return
	}
	drawSlantedQuad(screen, x, y, w, h, slant, empty)
	if ratio > 0 {
		if ratio >= 1 {
			drawSlantedQuad(screen, x, y, w, h, slant, fill)
		} else {
			filledSpan := innerSpan * ratio
			var path vector.Path
			path.MoveTo(float32(x+slant), float32(y))
			path.LineTo(float32(x+slant+filledSpan), float32(y))
			path.LineTo(float32(x+filledSpan), float32(y+h))
			path.LineTo(float32(x), float32(y+h))
			path.Close()
			fillSlantedPath(screen, &path, fill)
		}
	}
}

// 以下はすべて、ステータスブロックの左上角（baseX/baseY = drawStatusBar の
// xPos/winY）を基準にした個別オフセット。
const (
	namePlateTextOffsetX = 0.0
	namePlateTextOffsetY = 0.0

	namePlateLineOffsetX = -40.0
	namePlateLineOffsetY = 0.0

	namePlateMyTurnOffsetX = -70.0
	namePlateMyTurnOffsetY = -70.0
)

func (s *BattleScene) drawPartyName(screen *ebiten.Image, i int, xPos, winY, alpha float64) {
	baseX := xPos
	baseY := winY

	isDead := s.game.PlayerHP[i] <= 0

	nameCol := uiColorText
	if isDead {
		nameCol = uiColorDead
	}

	lineImg := s.game.NameImg
	if isDead {
		lineImg = s.game.NameDeadImg
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(baseX+namePlateLineOffsetX, baseY+namePlateLineOffsetY)
	op.ColorScale.ScaleAlpha(float32(alpha))
	screen.DrawImage(lineImg, op)

	drawBattleOutlinedText(screen, baseX+namePlateTextOffsetX, baseY+namePlateTextOffsetY, PlayerNames[i], s.game.FontFace(partyNameFontSize), nameCol, alpha)
}

// drawMyTurnOverlay draws the command-selection highlight image for whichever
// party member is currently acting. Called after every other battle UI so it
// always renders on top (e.g. above the command menu panel).
func (s *BattleScene) drawMyTurnOverlay(screen *ebiten.Image) {
	i := s.waitingActor
	isMyTurn := i >= 0 && i < partySize &&
		(s.battlePhase == phasePlayerMenu || s.battlePhase == phaseSkillMenu ||
			s.battlePhase == phaseTargetSelect || s.battlePhase == phaseHealSelect ||
			s.battlePhase == phaseItemMenu || s.battlePhase == phaseItemTarget)
	if !isMyTurn {
		return
	}

	xPos := statusStartX + float64(i)*(statusBlockW+statusBlockGap)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(xPos+namePlateMyTurnOffsetX, statusBarY+namePlateMyTurnOffsetY)
	op.ColorScale.ScaleAlpha(float32(s.statusBarAlpha()))
	screen.DrawImage(s.game.NameMyTurnImg, op)
}
