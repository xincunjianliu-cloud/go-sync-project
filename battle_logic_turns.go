package main

import (
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func isLowHP(hp, maxHP int) bool {
	if maxHP <= 0 {
		return false
	}
	return float64(hp)/float64(maxHP) <= 0.25
}

func (s *BattleScene) startCast(actor int) {
	if actor < 0 || actor >= partySize {
		return
	}
	s.healingCaster = actor
	s.playerPose[actor] = poseHealCast
	s.playerAnimTimer[actor] = 0
}

func (s *BattleScene) endCast(actor int) {
	if actor < 0 || actor >= partySize {
		return
	}
	if s.healingCaster == actor {
		s.healingCaster = -1
	}
	s.playerPose[actor] = poseIdle
	s.playerAnimTimer[actor] = 0
	s.healingAnimTimer[actor] = 0
}

func (s *BattleScene) tickATB(dt float64) {
	for i := 0; i < partySize; i++ {
		if s.waitStance[i] || s.waitCancelHold[i] > 0 || s.atbGauge[i] >= atbMax || s.game.PlayerHP[i] <= 0 {
			continue
		}
		speed := atbBaseSpeed
		if s.rewindActive {
			speed *= 1.5
		}
		s.atbGauge[i] += speed * dt
		if s.atbGauge[i] > atbMax {
			s.atbGauge[i] = atbMax
		}
	}
	for i := range s.enemies {
		if s.enemies[i].HP <= 0 {
			continue
		}
		actor := s.enemyActorIndex(i)
		if s.atbGauge[actor] >= atbMax {
			continue
		}
		s.atbGauge[actor] += atbBaseSpeed * dt
		if s.atbGauge[actor] > atbMax {
			s.atbGauge[actor] = atbMax
		}
	}
}

func (s *BattleScene) updateRewind(dt float64) {
	if !s.rewindActive {
		return
	}
	s.rewindTimer -= dt
	if s.rewindTimer <= 0 {
		s.rewindTimer = 0
		s.rewindActive = false
		for i := 0; i < partySize; i++ {
			s.rewindExtraTurnAvailable[i] = false
		}
		s.battleLog = "巻き戻し効果が切れた"
		s.battleLogTimer = battleLogDuration
	}
}

func (s *BattleScene) fleeSuccessRate() int {
	if strings.HasPrefix(s.enemyType, "boss_") {
		return 0
	}
	alive := s.aliveEnemyIndices()
	if len(alive) == 0 {
		return 100
	}
	totalLv := 0
	for _, i := range alive {
		totalLv += s.enemies[i].Lv
	}
	avgLv := float64(totalLv) / float64(len(alive))
	rate := 70 + int((float64(s.game.PlayerLv[0])-avgLv)*10)
	if rate < 10 {
		return 10
	}
	if rate > 100 {
		return 100
	}
	return rate
}

const atbBaseSpeed = 10.0

const (
	// Return-to-timeline position (0-100) after each non-skill action.
	// Each of these is individually configurable; all default to 5 for now.
	normalAttackReturnPosition      = 5.0
	enemyNormalAttackReturnPosition = 5.0
	itemUseReturnPosition           = 5.0
	rewindReturnPosition            = 5.0
	synergyReturnPosition           = 5.0
	waitCancelReturnPosition        = 5.0
	fleeFailReturnPosition          = 5.0
)

// normalAttackGaugePoint is how many synergy gauge points a normal attack
// adds. Skill uses instead add their own SkillLevelData.GaugePoint.
const normalAttackGaugePoint = 1

// atbHeadStart determines the timeline position (0-100) an actor starts
// battle at, based on their Spd stat only. It is not used for the
// return position after subsequent actions.
func atbHeadStart(spd int) float64 {
	if spd <= 0 {
		return 0
	}
	return float64(spd) / 2.0
}

func clampAtbPosition(pos float64) float64 {
	if pos < 0 {
		return 0
	}
	if pos > atbMax {
		return atbMax
	}
	return pos
}

func (s *BattleScene) resetPlayerGaugeTo(actor int, pos float64) {
	if actor < 0 || actor >= partySize {
		return
	}
	s.atbGauge[actor] = clampAtbPosition(pos)
}

func (s *BattleScene) resetEnemyGaugeTo(slot int, pos float64) {
	if slot < 0 || slot >= len(s.enemies) {
		return
	}
	s.atbGauge[s.enemyActorIndex(slot)] = clampAtbPosition(pos)
}

// pendingActionReturnPosition resolves the return position for whatever
// player action is currently recorded in s.pendingSkill.
func (s *BattleScene) pendingActionReturnPosition() float64 {
	p := s.waitingActor
	if p < 0 || p >= partySize || s.pendingSkill < 1 {
		return normalAttackReturnPosition
	}
	skillIdx := s.pendingSkill - 1
	skills := s.game.CharacterSkills(p)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return normalAttackReturnPosition
	}
	levels := skills[skillIdx].Levels
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 || lv > len(levels) {
		lv = 1
	}
	return levels[lv-1].ReturnPosition
}

func (s *BattleScene) actorPosX(actor int) float64 {
	start := timelineStartX
	end := s.goalScreenX()
	ratio := s.atbGauge[actor] / atbMax
	if ratio > 1 {
		ratio = 1
	}
	return start + ratio*(end-start) - iconSize/2
}

func (s *BattleScene) isActorReady(actor int) bool {
	if actor >= 0 && actor < partySize && s.waitCancelHold[actor] > 0 {
		return false
	}
	return s.atbGauge[actor] >= atbMax
}

func (s *BattleScene) tryStartNextActor() {
	if s.battleLogTimer > 0 {
		return
	}
	best := -1
	for i := 0; i < partySize; i++ {
		if !s.isActorReady(i) || s.waitStance[i] {
			continue
		}
		if best < 0 || s.atbGauge[i] > s.atbGauge[best] {
			best = i
		}
	}
	if best >= 0 {
		s.openPlayerMenu(best)
		return
	}
	bestEnemy := -1
	for i := range s.enemies {
		if s.enemies[i].HP <= 0 {
			continue
		}
		actor := s.enemyActorIndex(i)
		if !s.isActorReady(actor) {
			continue
		}
		if bestEnemy < 0 || s.atbGauge[actor] > s.atbGauge[s.enemyActorIndex(bestEnemy)] {
			bestEnemy = i
		}
	}
	if bestEnemy >= 0 {
		s.actingEnemySlot = bestEnemy
		s.waitingActor = s.enemyActorIndex(bestEnemy)
		s.enemyIsActing = true
		s.enemyWindupTimer = enemyWindupDuration
	}
}

func (s *BattleScene) openPlayerMenu(actor int) {
	s.waitingActor = actor
	s.activePlayer = actor + 1
	s.commandIndex = s.lastCommandIndex[actor]
	s.commandTapArmed = false
	s.rewindButtonArmed = false
	s.itemButtonArmed = false
	s.battlePhase = phasePlayerMenu
	s.playerPose[actor] = poseReady
	s.playerAnimTimer[actor] = 0
	s.readySlideX[actor] = 0.0
}

func hitTestCommandMenu(game *Game) (int, bool) {
	positions := commandIconPositions()
	rects := make([]tapRect, 4)
	for i, pos := range positions {
		icon := game.CommandIcons[i]
		iw := float64(icon.Bounds().Dx())
		ih := float64(icon.Bounds().Dy())
		rects[i] = tapRect{x: pos[0] - iw/2, y: pos[1] - ih/2, w: iw, h: ih}
	}
	return hitTestTapRects(rects)
}

func (s *BattleScene) updatePlayerMenu() Scene {
	if isMenuUpPressed() {
		s.commandIndex = 0
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuLeftPressed() {
		s.commandIndex = 1
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuRightPressed() {
		s.commandIndex = 2
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuDownPressed() {
		s.commandIndex = 3
		s.game.Audio.PlaySEByKey("cursor")
	}
	tappedIdx, tappedOk := hitTestCommandMenu(s.game)
	rewindTapped := isRewindButtonJustPressed(s.game)
	itemTapped := isItemButtonJustPressed()

	if tappedOk {
		s.rewindButtonArmed = false
		s.itemButtonArmed = false
	}
	if rewindTapped {
		s.commandTapArmed = false
		s.itemButtonArmed = false
	}
	if itemTapped {
		s.commandTapArmed = false
		s.rewindButtonArmed = false
	}

	tapped := tapArmSelectOrConfirm(tappedIdx, tappedOk, &s.commandIndex, &s.commandTapArmed, s.game.Audio)
	rewindConfirm := tapArmButtonConfirm(rewindTapped, &s.rewindButtonArmed, s.game.Audio)
	itemConfirm := tapArmButtonConfirm(itemTapped, &s.itemButtonArmed, s.game.Audio)

	if inpututil.IsKeyJustPressed(ebiten.KeyF) || rewindConfirm {
		p := s.waitingActor
		if p >= 0 && p < partySize && s.canUseRewind(p) {
			s.game.Audio.PlaySEByKey("decide")
			s.executeRewind(p)
			return nil
		}
		s.game.Audio.PlaySEByKey("error")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) || itemConfirm {
		p := s.waitingActor
		if p >= 0 && p < partySize && s.hasAnyBattleUsableItem() {
			s.game.Audio.PlaySEByKey("decide")
			s.itemIndex = 0
			s.battlePhase = phaseItemMenu
			return nil
		}
		s.game.Audio.PlaySEByKey("error")
	}

	confirm := isConfirmKeyPressed() || tapped

	if !confirm {
		return nil
	}

	p := s.waitingActor
	if p < 0 || p >= partySize {
		s.battlePhase = phaseATB
		s.waitingActor = -1
		return nil
	}

	if s.commandIndex == 2 {
		if !s.hasFullPartyForSynergy() {
			s.game.Audio.PlaySEByKey("error")
			return nil
		}
		if !s.canUseSynergy() {
			s.game.Audio.PlaySEByKey("error")
			return nil
		}
	}

	s.game.Audio.PlaySEByKey("decide")
	s.lastCommandIndex[p] = s.commandIndex

	switch s.commandIndex {
	case 0:
		s.pendingSkill = 0
		s.targetIndex = s.firstAliveEnemySlot()
		s.battlePhase = phaseTargetSelect
		return nil
	case 1:
		skills := s.game.CharacterSkills(p)
		s.skillIndex = s.lastSkillIndex[p]
		if s.skillIndex < 0 || s.skillIndex >= len(skills) {
			s.skillIndex = 0
		}
		for i := range s.skillLevelCursors[p] {
			if s.skillLevelCursors[p][i] < 1 {
				lv := s.lastSkillLevel[p][i]
				if lv < 1 {
					lv = 1
				}
				s.skillLevelCursors[p][i] = lv
			}
		}
		s.skillMenuOpenTimer = 0.1
		s.battlePhase = phaseSkillMenu
		return nil
	case 2:
		s.waitStance[p] = true
		s.atbGauge[p] = atbMax
		alreadyInOrder := false
		for _, actorIdx := range s.waitOrder {
			if actorIdx == p {
				alreadyInOrder = true
				break
			}
		}
		if !alreadyInOrder {
			s.waitOrder = append(s.waitOrder, p)
		}
		if !s.tryWaitSynergy() {
			s.waitingActor = -1
			s.battlePhase = phaseATB
			s.tryStartNextActor()
		}
		return nil
	case 3:
		if rand.Intn(100) >= s.fleeSuccessRate() {
			s.resetPlayerGaugeTo(p, fleeFailReturnPosition)
			s.waitingActor = -1
			s.battlePhase = phaseATB
			s.battleLog = "逃げられなかった"
			s.battleLogTimer = 1.5
			return nil
		}
		s.fleeSucceeded = true
		s.battleLog = "逃げきれた！"
		s.battleLogTimer = 1.5
		s.battlePhase = phaseMessage
		return nil
	}
	return nil
}

func (s *BattleScene) updateSkillMenu(dt float64) {
	if s.skillMenuOpenTimer > 0 {
		s.skillMenuOpenTimer -= 1.0 / 60.0
		return
	}
	p := s.waitingActor
	if p < 0 || p >= partySize {
		return
	}
	skills := s.game.CharacterSkills(p)
	menuLen := len(skills)

	if isMenuDownPressed() {
		s.skillIndex = (s.skillIndex + 1) % menuLen
		s.lastSkillIndex[p] = s.skillIndex
		s.game.Audio.PlaySEByKey("cursor")
	}
	if isMenuUpPressed() {
		s.skillIndex = (s.skillIndex - 1 + menuLen) % menuLen
		s.lastSkillIndex[p] = s.skillIndex
		s.game.Audio.PlaySEByKey("cursor")
	}

	arrowTapped := s.handleSkillLevelArrowTaps(p, skills)
	tappedIdx, tappedOk := -1, false
	if !arrowTapped {
		tappedIdx, tappedOk = s.hitTestBattleSubRows(menuLen)
	}
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.skillIndex, s.game.Audio)
	if tappedOk {
		s.lastSkillIndex[p] = s.skillIndex
	}

	curLv := s.game.PlayerSkillLv[p][s.skillIndex]
	if curLv < 1 {
		curLv = 1
	}
	if curLv > len(skills[s.skillIndex].Levels) {
		curLv = len(skills[s.skillIndex].Levels)
	}

	if s.skillLevelCursors[p][s.skillIndex] < 1 {
		s.skillLevelCursors[p][s.skillIndex] = 1
	}
	if s.skillLevelCursors[p][s.skillIndex] > curLv {
		s.skillLevelCursors[p][s.skillIndex] = curLv
	}

	if isMenuRightPressed() {
		if s.skillLevelCursors[p][s.skillIndex] < curLv {
			s.skillLevelCursors[p][s.skillIndex]++
			s.game.Audio.PlaySEByKey("cursor")
		}
	}
	if isMenuLeftPressed() {
		if s.skillLevelCursors[p][s.skillIndex] > 1 {
			s.skillLevelCursors[p][s.skillIndex]--
			s.game.Audio.PlaySEByKey("cursor")
		}
	}
	if isEscapePressed() || (!arrowTapped && !tappedOk && s.isTapOutsideBattleSubPanel()) {
		s.game.Audio.PlaySEByKey("cancel")
		s.battlePhase = phasePlayerMenu
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if s.canUseRewind(p) {
			s.game.Audio.PlaySEByKey("decide")
			s.executeRewind(p)
			return
		}
	}

	confirm := isConfirmKeyPressed() || tapped
	if !confirm {
		return
	}

	lv := s.skillLevelCursors[p][s.skillIndex]
	if lv < 1 {
		lv = 1
	}
	data := skills[s.skillIndex].Levels[lv-1]
	if !s.game.IsSkillUnlocked(p, s.skillIndex) {
		s.game.Audio.PlaySEByKey("error")
		return
	}
	if s.game.PlayerMP[p] < s.effectiveMPCost(data.MPCost) {
		s.game.Audio.PlaySEByKey("error")
		return
	}
	s.game.Audio.PlaySEByKey("decide")
	s.pendingSkill = s.skillIndex + 1

	if s.skillIndex < len(s.lastSkillLevel[p]) {
		s.lastSkillLevel[p][s.skillIndex] = lv
	}

	if data.IsHeal {
		s.healTargetIndex = 0
		s.battlePhase = phaseHealSelect
		return
	}

	if data.Target == TargetBoth {
		s.selectedSkillTarget = TargetSingle
	} else {
		s.selectedSkillTarget = data.Target
	}
	if data.Target == TargetAll {
		s.targetIndex = maxEnemies
	} else {
		s.targetIndex = s.firstAliveEnemySlot()
	}
	s.battlePhase = phaseTargetSelect
}
