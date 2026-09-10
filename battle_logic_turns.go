package main

// battle_logic_turns.go: ターン進行・ATB・プレイヤーメニュー・スキルメニューの更新処理
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
		// ★変更：素早さは固定配列(playerSpeeds)ではなく、
		// レベルアップで個別成長するステータス PlayerSpd を使う
		// （＝「タイムライン上でアイコンが進む速さ」そのもの）。
		speed := float64(s.game.PlayerSpd[i])
		if s.rewindActive {
			speed *= 1.5
		}
		s.atbGauge[i] += speed * dt
		if s.atbGauge[i] > atbMax {
			s.atbGauge[i] = atbMax
		}
	}
	if s.atbGauge[enemyID] < atbMax {
		s.atbGauge[enemyID] += s.enemySpeed * dt
		if s.atbGauge[enemyID] > atbMax {
			s.atbGauge[enemyID] = atbMax
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
	rate := 70 + (s.game.PlayerLv[0]-s.enemyLv)*10
	if rate < 10 {
		return 10
	}
	if rate > 100 {
		return 100
	}
	return rate
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
	if s.isActorReady(enemyID) {
		s.waitingActor = enemyID
		s.executeEnemyAction()
		s.atbGauge[enemyID] = 0
		s.waitingActor = -1
		s.tickDebuffs(enemyID, true)
		if s.checkBattleEnd() {
			return
		}
	}
}

func (s *BattleScene) openPlayerMenu(actor int) {
	s.waitingActor = actor
	s.activePlayer = actor + 1
	s.commandIndex = s.lastCommandIndex[actor]
	s.battlePhase = phasePlayerMenu
	s.playerPose[actor] = poseReady
	s.playerAnimTimer[actor] = 0
	s.readySlideX[actor] = 0.0
}

func (s *BattleScene) updatePlayerMenu() Scene {
	if isMenuUpPressed() {
		s.commandIndex = 0
	}
	if isMenuLeftPressed() {
		s.commandIndex = 1
	}
	if isMenuRightPressed() {
		s.commandIndex = 2
	}
	if isMenuDownPressed() {
		s.commandIndex = 3
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		p := s.waitingActor
		if p >= 0 && p < partySize && s.canUseRewind(p) {
			s.executeRewind(p)
			return nil
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		p := s.waitingActor
		if p >= 0 && p < partySize && s.hasAnyBattleUsableItem() {
			s.itemIndex = 0
			s.battlePhase = phaseItemMenu
			return nil
		}
	}

	confirm := isConfirmKeyPressed()

	if !confirm {
		return nil
	}

	p := s.waitingActor
	if p < 0 || p >= partySize {
		s.battlePhase = phaseATB
		s.waitingActor = -1
		return nil
	}

	s.lastCommandIndex[p] = s.commandIndex

	switch s.commandIndex {
	case 0:
		s.pendingSkill = 0
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
		if !s.hasFullPartyForSynergy() {
			s.battleLog = "4人そろっていないため待機できない"
			s.battleLogTimer = 1.5
			return nil
		}
		if !s.canUseSynergy() {
			s.battleLog = "ゲージポイントが足りない"
			s.battleLogTimer = 1.5
			return nil
		}
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
		s.tryWaitSynergy()
		if s.waitStance[p] {
			s.waitingActor = -1
			s.battlePhase = phaseATB
			s.tryStartNextActor()
		}
		return nil
	case 3:
		if rand.Intn(100) >= s.fleeSuccessRate() {
			s.atbGauge[p] = 0
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
	}
	if isMenuUpPressed() {
		s.skillIndex = (s.skillIndex - 1 + menuLen) % menuLen
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
		}
	}
	if isMenuLeftPressed() {
		if s.skillLevelCursors[p][s.skillIndex] > 1 {
			s.skillLevelCursors[p][s.skillIndex]--
		}
	}
	if isEscapePressed() {
		s.battlePhase = phasePlayerMenu
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if s.canUseRewind(p) {
			s.executeRewind(p)
			return
		}
	}

	confirm := isConfirmKeyPressed()
	if !confirm {
		return
	}

	lv := s.skillLevelCursors[p][s.skillIndex]
	if lv < 1 {
		lv = 1
	}
	data := skills[s.skillIndex].Levels[lv-1]
	if s.game.PlayerMP[p] < s.effectiveMPCost(data.MPCost) {
		return
	}
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
	s.battlePhase = phaseTargetSelect
}
