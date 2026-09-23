package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (s *BattleScene) drawTimeline(screen *ebiten.Image) {
	drawX, drawY := s.timelineDrawOrigin()
	goalX := s.goalScreenX()
	centerY := trackCenterY()

	introX := 0.0
	if s.introActive && s.introPhase == 0 {
		introX = -s.introOffsetX
	}

	introVertY := 0.0
	if s.introActive && s.introPhase == 1 {
		introVertY = s.introVertOffsetY
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(drawX+introX, drawY)
	screen.DrawImage(s.game.TimelineBarImg, op)

	if !s.introActive || s.introPhase >= 1 {
		vertX, vertY := s.timelineVertDrawOrigin()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(vertX, vertY+introVertY)
		screen.DrawImage(s.game.TimelineBarVertImg, op)
	}

	if !s.introActive || s.introPhase >= 2 {
		gw := float64(s.game.GoalImg.Bounds().Dx())
		gh := float64(s.game.GoalImg.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(goalX-gw/2, centerY-gh/2)
		screen.DrawImage(s.game.GoalImg, op)
	}

	if s.introActive && s.introPhase < 3 {
		return
	}

	frozen := s.battlePhase == phaseBattleEnd && !s.isWon

	anyoneSelecting := s.waitingActor >= 0 &&
		(s.battlePhase == phasePlayerMenu || s.battlePhase == phaseSkillMenu ||
			s.battlePhase == phaseItemMenu || s.battlePhase == phaseItemTarget)

	order := make([]int, partySize+len(s.enemies))
	for i := range order {
		order[i] = i
	}

	heldAtGoal := func(actor int) bool {
		isDead := actor < partySize && s.game.PlayerHP[actor] <= 0
		isActive := s.waitingActor == actor && (s.battlePhase == phasePlayerMenu || s.battlePhase == phaseSkillMenu || s.battlePhase == phaseItemMenu || s.battlePhase == phaseItemTarget)
		isHoldingSize := !isDead && actor < partySize && s.returnDelayTimer[actor] > 0
		isMyselfWaiting := !isDead && ((actor < partySize && (s.waitStance[actor] || s.waitCancelHold[actor] > 0)) || (isActive && s.commandIndex == cmdWait))
		ready := !isDead && s.isActorReady(actor)
		if isEnemyActor(actor) && s.enemyIsActing && enemySlotFromActor(actor) == s.actingEnemySlot {
			ready = true
		}
		return ready || ((isActive || isHoldingSize) && !isMyselfWaiting)
	}
	less := func(a, b int) bool {
		ha, hb := heldAtGoal(a), heldAtGoal(b)
		if ha != hb {
			return hb
		}
		return s.atbGauge[a] < s.atbGauge[b]
	}
	for i := 0; i < len(order); i++ {
		for j := i + 1; j < len(order); j++ {
			if less(order[j], order[i]) {
				order[i], order[j] = order[j], order[i]
			}
		}
	}

	for _, actor := range order {
		if isEnemyActor(actor) && s.enemies[enemySlotFromActor(actor)].HP <= 0 {
			continue
		}

		useSnapshot := frozen && actor < partySize

		var x, y, currentIconSize float64
		var isEnlarged bool

		if useSnapshot {
			x = s.tlFrozenX[actor]
			y = s.tlFrozenY[actor]
			currentIconSize = s.tlFrozenSize[actor]
			isEnlarged = s.tlFrozenLarge[actor]
		} else {
			isDead := actor < partySize && s.game.PlayerHP[actor] <= 0
			isActive := s.waitingActor == actor && (s.battlePhase == phasePlayerMenu || s.battlePhase == phaseSkillMenu || s.battlePhase == phaseItemMenu || s.battlePhase == phaseItemTarget)
			isHoldingPosition := !isDead && actor < partySize && s.returnDelayTimer[actor] > 0
			isHoldingSize := isHoldingPosition
			isMyselfWaiting := !isDead && ((actor < partySize && (s.waitStance[actor] || s.waitCancelHold[actor] > 0)) || (isActive && s.commandIndex == cmdWait))
			currentIconSize = float64(iconSize)
			if (isActive || isHoldingSize) && !isMyselfWaiting {
				currentIconSize = float64(iconSize) * 1.5
			}

			x = s.actorPosX(actor)
			ready := !isDead && s.isActorReady(actor)
			if isEnemyActor(actor) && s.enemyIsActing && enemySlotFromActor(actor) == s.actingEnemySlot {
				ready = true
			}
			if actor < partySize && s.deadWaitStuck[actor] {
				x = timelineStartX - (currentIconSize / 2)
			}
			if ready || ((isActive || isHoldingPosition) && !isMyselfWaiting) {
				x = goalX - (currentIconSize / 2)
			}
			if actor < partySize && s.deadWaitStuck[actor] {
				x = timelineStartX - (currentIconSize / 2)
			}

			y = centerY - currentIconSize/2

			if isMyselfWaiting {
				downCount := -1
				waitingOrder := s.waitOrder
				if actor < partySize && s.waitCancelHold[actor] > 0 {
					waitingOrder = s.waitCancelOrder
				}
				for orderIdx, actorIdx := range waitingOrder {
					if actorIdx == actor {
						downCount = orderIdx
						break
					}
				}
				if downCount == -1 {
					downCount = len(s.waitOrder)
				}
				y += 75.0 + float64(downCount)*55.0
			}

			blockedByOther := anyoneSelecting && actor != s.waitingActor && actor < partySize
			isEnlarged = (ready || isActive || isHoldingSize) && !isMyselfWaiting && !blockedByOther

			if actor < partySize {
				s.tlFrozenX[actor] = x
				s.tlFrozenY[actor] = y
				s.tlFrozenSize[actor] = currentIconSize
				s.tlFrozenLarge[actor] = isEnlarged
			}
		}

		var iconImg *ebiten.Image
		if actor < partySize {
			if isEnlarged {
				iconImg = s.game.TimelineIconsLarge[actor]
			} else {
				iconImg = s.game.TimelineIcons[actor]
			}
		} else {
			name := s.enemies[enemySlotFromActor(actor)].Name
			if isEnlarged {
				iconImg = s.game.GetEnemyTimelineIconLarge(s.enemyType, name)
			} else {
				iconImg = s.game.GetEnemyTimelineIcon(s.enemyType, name)
			}
		}

		iw := float64(iconImg.Bounds().Dx())
		ih := float64(iconImg.Bounds().Dy())

		offsetX := (currentIconSize - iw) / 2
		offsetY := (currentIconSize - ih) / 2

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x+offsetX, y+offsetY)

		screen.DrawImage(iconImg, op)
	}
}

// statusBarAlpha computes the intro fade-in alpha shared by the status HUD
// (name plates, HP/MP, and the my-turn overlay drawn separately on top).
func (s *BattleScene) statusBarAlpha() float64 {
	alpha := 1.0
	if s.introActive {
		if s.introPhase < 3 {
			alpha = 0.0
		} else {
			ratio := 1.0 - s.introCharOffsetX/introStartOffsetX
			const fadeStart = 0.7
			if ratio < fadeStart {
				alpha = 0.0
			} else {
				alpha = (ratio - fadeStart) / (1.0 - fadeStart)
				if alpha > 1.0 {
					alpha = 1.0
				}
			}
		}
	}
	return alpha
}

func (s *BattleScene) drawStatusBar(screen *ebiten.Image) {

	alpha := s.statusBarAlpha()
	for i := 0; i < partySize; i++ {
		sx, sy := partyNamePosition(i)

		s.drawPartyName(screen, i, sx, sy, alpha)

		isDead := s.game.PlayerHP[i] <= 0
		valueColor := uiColorText
		if isDead {
			valueColor = uiColorDead
		}

		drawStatusBox(screen, s.game, sx, sy,
			s.game.PlayerHP[i], s.game.PlayerMaxHP[i],
			s.game.PlayerMP[i], s.game.PlayerMaxMP[i],
			alpha, valueColor, isDead)

	}
}

// statIconOrder is the fixed left-to-right order the battle HUD's per-stat
// icons are drawn in next to a party member's name: physical attack, magic
// attack, physical defense, magic defense, luck.
var statIconOrder = [...]StatKind{StatAtk, StatMat, StatDef, StatMdf, StatLuk}

// drawPartyStatIcons draws one icon per stat currently buffed or debuffed for
// party member i, packed left-to-right starting at (x, y) in statIconOrder.
// A stat with both a buff and a debuff active nets the two percentages and
// shows a single icon for whichever direction still wins, so at most one
// icon is drawn per stat: StatIconUpImgs[stat] (blue) when the net is
// positive, StatIconDownImgs[stat] (red) when negative, nothing when they
// cancel out exactly.
func (s *BattleScene) drawPartyStatIcons(screen *ebiten.Image, i int, x, y, alpha float64) {
	for _, stat := range statIconOrder {
		net := SumBuffPercent(s.PlayerBuffs[i], stat) - SumDebuffPercent(s.PlayerDebuffs[i], stat)
		if net == 0 {
			continue
		}
		img := s.game.StatIconDownImgs[stat]
		if net > 0 {
			img = s.game.StatIconUpImgs[stat]
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y+statIconOffsetY)
		op.ColorScale.ScaleAlpha(float32(alpha))
		screen.DrawImage(img, op)
		x += float64(img.Bounds().Dx()) + statIconGapX
	}
}

func (s *BattleScene) drawGaugeTriangle(screen *ebiten.Image) {
	x := gaugeTriX
	y := gaugeTriY
	w := float64(s.game.GaugeImg.Bounds().Dx())
	h := float64(s.game.GaugeImg.Bounds().Dy())
	pad := gaugeImgBorder
	ix := x + pad
	iy := y + pad
	iw := w - pad*2
	ih := h - pad*2

	completedStages := s.gaugePoint / gaugePointsPerStage
	remainder := s.gaugePoint % gaugePointsPerStage
	filledHeight := float64(completedStages)*(float64(gaugePointsPerStage)*gaugeFillPerPoint+gaugeDividerHeight) +
		float64(remainder)*gaugeFillPerPoint
	if filledHeight > ih {
		filledHeight = ih
	}
	if filledHeight < 0 {
		filledHeight = 0
	}

	ixI := int(math.Round(ix))
	iyI := int(math.Round(iy))
	iwI := int(math.Round(iw))
	ihI := int(math.Round(ih))

	stageSlot := float64(gaugePointsPerStage)*gaugeFillPerPoint + gaugeDividerHeight

	for stage := 0; stage < completedStages && stage < gaugeMaxStage; stage++ {
		hStart := float64(stage) * stageSlot
		hEnd := hStart + stageSlot
		if hEnd > ih {
			hEnd = ih
		}
		s.fillGaugeTriSegment(screen, ixI, iyI, iwI, ihI, hStart, hEnd, s.gaugeSegmentColor(stage))
	}

	if remainder > 0 && completedStages < gaugeMaxStage {
		hStart := float64(completedStages) * stageSlot
		hEnd := hStart + float64(remainder)*gaugeFillPerPoint
		if hEnd > ih {
			hEnd = ih
		}
		s.fillGaugeTriSegment(screen, ixI, iyI, iwI, ihI, hStart, hEnd, s.gaugeSegmentColor(completedStages))
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(s.game.GaugeImg, op)
}

func commandIconPositions() [4][2]float64 {
	centerX := 860.0
	centerY := 440.0
	spacing := 48.0

	attackX, attackY := attackIconCenter()

	return [4][2]float64{
		{attackX, attackY},
		{centerX - spacing, centerY},
		{centerX + spacing, centerY},
		{centerX, centerY + spacing},
	}
}

func (s *BattleScene) drawCommandMenu(screen *ebiten.Image) {
	positions := commandIconPositions()

	for i, pos := range positions {
		icon := s.game.CommandIcons[i]
		if s.commandIndex == i && s.game.CommandIconsSelected[i] != nil {
			icon = s.game.CommandIconsSelected[i]
		}

		iw := icon.Bounds().Dx()
		ih := icon.Bounds().Dy()

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pos[0]-float64(iw)/2, pos[1]-float64(ih)/2)

		if s.commandIndex != i {
			op.ColorScale.Scale(0.6, 0.6, 0.6, 1.0)
		}

		screen.DrawImage(icon, op)
	}
}

// drawSkillSubMenuEnlarged is used only by the skill-upgrade tutorial's
// mockup (field_tutorial.go): it renders the normal skill submenu into a
// scratch buffer, then composites that buffer back onto screen scaled up by
// battleSkillPanelScale around battleSkillPanelPivot (the panel's own right
// edge) so the level number and its ◀▶ switch read more clearly while being
// explained, without pushing anything off the right edge of the screen.
// Real battles keep drawing drawSkillSubMenu at normal size.
func (s *BattleScene) drawSkillSubMenuEnlarged(screen *ebiten.Image) {
	if s.skillSubMenuImg == nil {
		s.skillSubMenuImg = ebiten.NewImage(gameWidth, gameHeight)
	}
	buf := s.skillSubMenuImg
	buf.Clear()
	s.drawSkillSubMenu(buf)

	pivotX, pivotY := s.battleSkillPanelPivot()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-pivotX, -pivotY)
	op.GeoM.Scale(battleSkillPanelScale, battleSkillPanelScale)
	op.GeoM.Translate(pivotX, pivotY)
	screen.DrawImage(buf, op)
}

func (s *BattleScene) drawSkillSubMenu(screen *ebiten.Image) {
	windowX, windowY, windowW, _ := s.battleSubPanelOrigin()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(windowX, windowY)
	screen.DrawImage(s.game.SkillPanelImg, op)

	p := s.waitingActor
	skills := s.game.CharacterSkills(p)

	skillListFace := s.game.FontFace(18)
	skillListArrowGap := text.Advance("▶ ", skillListFace)

	for i, sk := range skills {
		if !s.game.IsSkillUnlocked(p, i) {
			continue
		}
		curLv := s.game.PlayerSkillLv[p][i]
		if curLv < 1 {
			curLv = 1
		}
		if curLv > len(sk.Levels) {
			curLv = len(sk.Levels)
		}

		lv := s.skillLevelCursors[p][i]
		if lv < 1 {
			lv = 1
		}
		if lv > curLv {
			lv = curLv
		}

		data := sk.Levels[lv-1]
		label := sk.Name
		costText := fmt.Sprintf("MP%d", s.effectiveMPCost(data.MPCost))
		insufficient := s.game.PlayerMP[p] < s.effectiveMPCost(data.MPCost)

		selected := i == s.skillIndex

		arrowCol := uiColorText
		if selected {
			arrowCol = uiColorSelect
		}

		textCol := arrowCol
		if insufficient {
			textCol = uiColorDisabled
		}

		baseX := windowX + battleSubLabelOffsetX
		baseY := windowY + battleSubLabelOffsetY + float64(i)*battleSubRowHeight
		if selected {
			arrowOp := &text.DrawOptions{}
			arrowOp.GeoM.Translate(baseX, baseY)
			arrowOp.ColorScale.ScaleWithColor(arrowCol)
			text.Draw(screen, "▶", skillListFace, arrowOp)
		}
		labelOp := &text.DrawOptions{}
		labelOp.GeoM.Translate(baseX+skillListArrowGap, baseY)
		labelOp.ColorScale.ScaleWithColor(textCol)
		text.Draw(screen, label, skillListFace, labelOp)

		_, _, leftX, lvX, rightX, textY := s.skillLevelArrowRects(i, lv, skillListFace)

		lvOp := &text.DrawOptions{}
		lvOp.GeoM.Translate(lvX, textY)
		lvOp.ColorScale.ScaleWithColor(textCol)
		text.Draw(screen, skillLvText(lv), skillListFace, lvOp)

		if curLv > 1 {
			leftCol := textCol
			if lv <= 1 {
				leftCol = uiColorDisabled
			}
			leftOp := &text.DrawOptions{}
			leftOp.GeoM.Translate(leftX, textY)
			leftOp.ColorScale.ScaleWithColor(leftCol)
			text.Draw(screen, skillLvLeftArrow, skillListFace, leftOp)

			rightCol := textCol
			if lv >= curLv {
				rightCol = uiColorDisabled
			}
			rightOp := &text.DrawOptions{}
			rightOp.GeoM.Translate(rightX, textY)
			rightOp.ColorScale.ScaleWithColor(rightCol)
			text.Draw(screen, skillLvRightArrow, skillListFace, rightOp)
		}

		mpOp := &text.DrawOptions{}
		mpOp.GeoM.Translate(windowX+windowW-battleSubRightOffsetX, baseY)
		mpOp.PrimaryAlign = text.AlignEnd
		mpOp.ColorScale.ScaleWithColor(textCol)
		text.Draw(screen, costText, skillListFace, mpOp)
	}
}

func (s *BattleScene) drawLogWindowBackground(screen *ebiten.Image) {
	winX := 0.0
	winW := float64(gameWidth)
	winY := logPanelY
	winH := 28.0
	ebitenutil.DrawRect(screen, winX, winY, winW, winH, color.RGBA{0, 0, 0, 200})
	borderColor := color.RGBA{255, 255, 255, 255}
	ebitenutil.DrawRect(screen, winX, winY, winW, 1, borderColor)
	ebitenutil.DrawRect(screen, winX, winY+winH-1, winW, 1, borderColor)
}

func (s *BattleScene) drawBattleMessage(screen *ebiten.Image) {
	if s.battleLog == "" {
		return
	}
	op := &text.DrawOptions{}
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.GeoM.Translate(float64(gameWidth)/2, logPanelY+logPanelH/2)
	op.ColorScale.ScaleWithColor(uiColorText)
	text.Draw(screen, s.battleLog, s.game.FontFace(18), op)
}

func (s *BattleScene) drawDirectMessages(screen *ebiten.Image) {
	if s.battlePhase == phaseBattleEnd && !s.isWon && s.battleLogTimer <= 0 {
		winX, winY, winW, winH := 380.0, 235.0, 220.0, 80.0
		ebitenutil.DrawRect(screen, winX, winY, winW, winH, color.RGBA{40, 10, 10, 220})
		ebitenutil.DrawRect(screen, winX, winY, winW, 1, uiColorDanger)
		ebitenutil.DrawRect(screen, winX, winY+winH, winW, 1, uiColorDanger)
		ebitenutil.DrawRect(screen, winX, winY, 1, winH, uiColorDanger)
		ebitenutil.DrawRect(screen, winX+winW, winY, 1, winH, uiColorDanger)

		gameOverFace := s.game.FontFace(18)
		gameOverArrowGap := text.Advance("▶ ", gameOverFace)

		retryCol := uiColorText
		retrySelected := s.gameOverIdx == 0
		if retrySelected {
			retryCol = uiColorSelect
			arrowOp1 := &text.DrawOptions{}
			arrowOp1.GeoM.Translate(winX+24, winY+22)
			arrowOp1.ColorScale.ScaleWithColor(retryCol)
			text.Draw(screen, "▶", gameOverFace, arrowOp1)
		}
		op1 := &text.DrawOptions{}
		op1.GeoM.Translate(winX+24+gameOverArrowGap, winY+22)
		op1.ColorScale.ScaleWithColor(retryCol)
		text.Draw(screen, "リトライ", gameOverFace, op1)

		titleCol := uiColorText
		titleSelected := s.gameOverIdx == 1
		if titleSelected {
			titleCol = uiColorSelect
			arrowOp2 := &text.DrawOptions{}
			arrowOp2.GeoM.Translate(winX+24, winY+50)
			arrowOp2.ColorScale.ScaleWithColor(titleCol)
			text.Draw(screen, "▶", gameOverFace, arrowOp2)
		}
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate(winX+24+gameOverArrowGap, winY+50)
		op2.ColorScale.ScaleWithColor(titleCol)
		text.Draw(screen, "タイトルへ戻る", gameOverFace, op2)
	}
}
