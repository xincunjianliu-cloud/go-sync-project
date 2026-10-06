package main

import (
	"math/rand"
	"slices"
	"strconv"
	"strings"
)

func (s *BattleScene) executeRewind(p int) {
	s.consumeAllGaugePoints()
	s.rewindActive = true
	s.rewindTimer = 15.0
	s.rewindUsed = true

	if s.lastEnemyAttackDamage > 0 && s.lastEnemyAttackTarget >= 0 && s.lastEnemyAttackTarget < partySize {
		target := s.lastEnemyAttackTarget
		s.game.PlayerHP[target] = s.lastEnemyAttackPrevHP
		if s.game.PlayerHP[target] > s.game.PlayerMaxHP[target] {
			s.game.PlayerHP[target] = s.game.PlayerMaxHP[target]
		}
		s.battleLog = "巻き戻し：直前攻撃をなかったことにした"
	} else {
		s.battleLog = "巻き戻しを発動した"
	}
	s.battleLogTimer = battleLogDuration
	s.finishPlayerTurn(rewindReturnPosition)
}

func (s *BattleScene) rollNormalDamage(target int) int {
	p := s.waitingActor
	if p < 0 || p >= partySize {
		return 5
	}
	atk := s.effectiveAtk(p)
	def := s.effectiveEnemyDef(target, false)
	if def < 1 {
		def = 1
	}
	power := 100.0 + float64(s.gaugeAtkBonus())
	return s.rollDamage(float64(atk), power, float64(def), 1.0, s.game.PlayerLuck[p])
}

func (s *BattleScene) finishPlayerTurn(returnPos float64) {
	if s.waitingActor >= 0 && s.waitingActor < partySize {
		s.resetPlayerGaugeTo(s.waitingActor, returnPos)
	}
	s.waitingActor = -1
	if s.checkBattleEnd() {
		return
	}
	s.battlePhase = phaseATB
	s.tryStartNextActor()
}

func (s *BattleScene) countWaitStance() int {
	n := 0
	for i := 0; i < partySize; i++ {
		if s.waitStance[i] {
			n++
		}
	}
	return n
}

func (s *BattleScene) cancelWaitAfterDeath() {
	s.waitCancelOrder = append(s.waitCancelOrder[:0], s.waitOrder...)
	for i := 0; i < partySize; i++ {
		if !s.waitStance[i] {
			continue
		}
		s.waitStance[i] = false
		if s.game.PlayerHP[i] <= 0 {
			s.atbGauge[i] = 0
			s.deadWaitStuck[i] = true
			continue
		}
		s.waitCancelHold[i] = 2.0
		s.playerPose[i] = poseDefend
		s.playerAnimTimer[i] = 0
	}
	s.waitOrder = []int{}
	s.battleLog = "味方が倒れたため連携待機が解除された"
	s.battleLogTimer = battleLogDuration
}

func (s *BattleScene) othersAllInWaitStance(actor int) bool {
	for i := 0; i < partySize; i++ {
		if i != actor && !s.waitStance[i] {
			return false
		}
	}
	return true
}

func (s *BattleScene) enterWaitStance(actor int) {
	s.waitStance[actor] = true
	s.atbGauge[actor] = atbMax
	for _, actorIdx := range s.waitOrder {
		if actorIdx == actor {
			return
		}
	}
	s.waitOrder = append(s.waitOrder, actor)
}

// tryWaitSynergy fires the 4-person synergy attack on s.targetIndex once
// every party member is in wait stance.
func (s *BattleScene) tryWaitSynergy() bool {
	for i := 0; i < partySize; i++ {
		if !s.waitStance[i] {
			return false
		}
	}

	power := 200.0 + float64(s.gaugeAtkBonus())

	atk := 0
	luck := 0
	for i := 0; i < partySize; i++ {
		atk += s.effectiveAtk(i)
		luck += s.game.PlayerLuck[i]
	}

	s.pendingSynergy = false
	for _, slot := range s.enemyTargetsForAttack(false) {
		def := s.effectiveEnemyDef(slot, false)
		dmg := s.rollDamage(float64(atk), power, float64(def), 1.0, luck)
		if s.lastRollWasCrit {
			s.game.Audio.PlaySEByKey("critical")
		} else {
			s.game.Audio.PlaySEByKey("damage")
		}
		s.applyDamageToEnemySlot(slot, dmg)
		ex, ey, ew, eh := s.enemyDrawRect(slot)
		s.spawnDamagePop(DamagePop{
			Value:  dmg,
			X:      ex + ew/2,
			Y:      ey - eh*damagePopHeadOffsetRatio,
			Vy:     -180.0,
			Timer:  0.0,
			IsCrit: s.lastRollWasCrit,
		})
	}

	s.triggerShake(hitTierSynergy)
	if s.allEnemiesDead() {
		s.checkBattleEnd()
		return true
	}

	for i := 0; i < partySize; i++ {
		s.waitStance[i] = false
		s.resetPlayerGaugeTo(i, synergyReturnPosition)
	}
	s.waitOrder = []int{}
	s.battlePhase = phaseATB
	s.waitingActor = -1
	s.tryStartNextActor()
	return true
}

func (s *BattleScene) rollEnemyAction() {
	var aliveList []int
	for i := 0; i < partySize; i++ {
		if s.game.PlayerHP[i] > 0 {
			aliveList = append(aliveList, i)
		}
	}
	if len(aliveList) == 0 {
		return
	}

	skills := s.enemies[s.actingEnemySlot].Skills
	if len(skills) > 0 && rand.Intn(100) < EnemySkillChance {
		skill := skills[rand.Intn(len(skills))]
		s.rollEnemySkill(skill, aliveList)
		return
	}

	s.rollEnemyNormalAttack(aliveList)
}

func (s *BattleScene) rollEnemyNormalAttack(aliveList []int) {
	target := s.coverRedirect(aliveList[rand.Intn(len(aliveList))])

	s.pendingEnemySkillName = ""
	s.pendingEnemySkillEffects = nil
	s.pendingEnemyIsAll = false
	s.enemyActionReturnPosition = enemyNormalAttackReturnPosition

	if s.rollIsEvade(s.game.PlayerLuck[target]) {
		s.pendingEnemyHits = []pendingEnemyHit{{target: target, evaded: true}}
		s.pendingEnemyHitTier = hitTierNone
		s.beginEnemyHitStop()
		return
	}

	def := s.effectivePlayerDef(target, false)
	if def < 1 {
		def = 1
	}
	dmg := s.rollDamage(float64(s.effectiveEnemyAtk(s.actingEnemySlot, false)), 100.0, float64(def), 1.0, 0)

	s.pendingEnemyHits = []pendingEnemyHit{{target: target, dmg: dmg}}
	s.pendingEnemyHitTier = hitTierWeak
	s.beginEnemyHitStop()
}

func (s *BattleScene) rollEnemySkill(skill EnemySkill, aliveList []int) {
	var targets []int
	if skill.Target == TargetAll {
		targets = aliveList
	} else {
		targets = []int{s.coverRedirect(aliveList[rand.Intn(len(aliveList))])}
	}

	s.pendingEnemySkillName = skill.Name
	s.pendingEnemySkillEffects = skill.Effects
	s.pendingEnemyIsAll = skill.Target == TargetAll
	s.enemyActionReturnPosition = skill.ReturnPosition

	var hits []pendingEnemyHit
	anyHit := false
	for _, target := range targets {
		if s.rollIsEvade(s.game.PlayerLuck[target]) {
			hits = append(hits, pendingEnemyHit{target: target, evaded: true})
			continue
		}
		dmg := s.rollEnemySkillDamage(skill.Power, skill.Element, target)
		hits = append(hits, pendingEnemyHit{target: target, dmg: dmg})
		if dmg > 0 {
			anyHit = true
		}
	}
	s.pendingEnemyHits = hits
	if anyHit {
		s.pendingEnemyHitTier = hitTierStrong
	} else {
		s.pendingEnemyHitTier = hitTierNone
	}
	s.beginEnemyHitStop()
}

func (s *BattleScene) beginEnemyHitStop() {
	switch s.pendingEnemyHitTier {
	case hitTierNone:
		s.enemyHitStopTimer = 0
		s.applyEnemyPendingHits()
	case hitTierWeak:
		s.enemyHitStopTimer = hitStopWeak
	default:
		s.enemyHitStopTimer = hitStopStrong
	}
}

func (s *BattleScene) applyEnemyPendingHits() {
	for _, hit := range s.pendingEnemyHits {
		target := hit.target
		targetX := s.partyScreenX[target] + spriteFrameW/2
		targetY := s.partyScreenY[target] - partyDamagePopOffsetY

		if hit.evaded {
			s.lastEnemyAttackTarget = target
			s.lastEnemyAttackPrevHP = s.game.PlayerHP[target]
			s.lastEnemyAttackDamage = 0
			s.evadeOffsetX[target] = evadeDodgeShiftX
			s.game.Audio.PlaySEByKey("evade")
			s.spawnDamagePop(DamagePop{
				X:      targetX,
				Y:      targetY,
				Vy:     -180.0,
				Timer:  0.0,
				IsMiss: true,
			})
			continue
		}

		dmg := hit.dmg
		prevHP := s.game.PlayerHP[target]
		s.lastEnemyAttackTarget = target
		s.lastEnemyAttackPrevHP = prevHP

		if s.tryGuard(target, dmg) {
			s.lastEnemyAttackDamage = 0
			s.game.Audio.PlaySEByKey("evade")
			s.spawnDamagePop(DamagePop{
				X:      targetX,
				Y:      targetY,
				Vy:     -180.0,
				IsMiss: true,
				Label:  "無敵",
			})
			continue
		}

		s.lastEnemyAttackDamage = dmg
		s.game.Audio.PlaySEByKey("damage")

		s.game.PlayerHP[target] -= dmg

		s.spawnDamagePop(DamagePop{
			Value: dmg,
			X:     targetX,
			Y:     targetY,
			Vy:    -180.0,
			Timer: 0.0,
		})

		if s.game.PlayerHP[target] > 0 {
			s.playerPose[target] = poseDamage
			s.playerAnimTimer[target] = 0.0
			s.playerFlashTimer[target] = spriteFlashDuration
		} else {
			s.game.PlayerHP[target] = 0
			s.coverCount[target] = 0
			s.counterTimer[target] = 0

			if s.countWaitStance() > 0 {
				s.cancelWaitAfterDeath()
			}
		}

		if s.pendingEnemySkillEffects != nil {
			s.applySkillEffects(s.pendingEnemySkillEffects, s.actingEnemySlot, false, target, s.pendingEnemyIsAll)
		}

		if dmg > 0 && s.game.PlayerHP[target] > 0 && s.counterTimer[target] > 0 {
			s.counterAttack(target, s.actingEnemySlot)
		}
	}

	// かばう's buffs stay through the last covered hit, then go.
	for i := 0; i < partySize; i++ {
		if s.coverCount[i] <= 0 {
			s.PlayerBuffs[i] = withoutCoverBuffs(s.PlayerBuffs[i])
		}
	}

	s.triggerShake(s.pendingEnemyHitTier)

	if s.pendingEnemySkillName != "" {
		s.battleLog = s.pendingEnemySkillName
		s.battleLogTimer = skillActionLogDuration
	} else {
		s.battleLog = ""
		s.battleLogTimer = silentActionLogDuration
	}
	s.enemyActionWaitTimer = enemyActionDuration

	s.waitingActor = -1
	s.checkBattleEnd()
}

func (s *BattleScene) checkBattleEnd() bool {
	if s.allEnemiesDead() {
		if !s.isWon {
			if strings.HasPrefix(s.enemyType, "boss_") {
				numStr := strings.TrimPrefix(s.enemyType, "boss_")
				if bossNum, err := strconv.Atoi(numStr); err == nil {
					if bossNum >= 1 && bossNum <= 4 {
						s.game.BossDefeatedFlags[bossNum-1] = true
						s.game.UpdateObjective()
					}
				}
			}

			for i := 0; i < partySize; i++ {
				s.expStartEXP[i] = s.game.PlayerEXP[i]
				s.drawPlayerLv[i] = s.game.PlayerLv[i]
				s.drawPlayerMaxEXP[i] = s.game.PlayerNextEXP[i]
			}

			totalExp := s.totalEnemyExp()
			for i := 0; i < partySize; i++ {
				if s.game.PlayerLv[i] < maxPlayerLevel {
					s.game.PlayerEXP[i] += totalExp
				}
				for s.game.PlayerLv[i] < maxPlayerLevel && s.game.PlayerEXP[i] >= s.game.PlayerNextEXP[i] {
					s.game.PlayerEXP[i] -= s.game.PlayerNextEXP[i]
					s.game.PlayerLv[i]++

					s.game.ApplyLevelUpGrowth(i)

					if s.game.PlayerLv[i] >= maxPlayerLevel {
						s.game.PlayerLv[i] = maxPlayerLevel
						s.game.PlayerEXP[i] = 0
						s.game.PlayerNextEXP[i] = 0
					} else {
						s.game.PlayerNextEXP[i] = PlayerExpToNextByLevel[s.game.PlayerLv[i]-1]
					}
				}
			}

			totalSP := s.totalEnemySP()
			for i := 0; i < partySize; i++ {
				s.game.PlayerSP[i] += totalSP
			}

			s.earnedItems = nil
			for ei := range s.enemies {
				for _, drop := range s.enemies[ei].Drops {
					if drop.Percent <= 0 {
						continue
					}
					if rand.Intn(100) >= drop.Percent {
						continue
					}
					qty := drop.rollCount()
					s.game.AddItem(drop.ItemID, qty)
					name := drop.ItemID
					if def, ok := GetItemDef(drop.ItemID); ok {
						name = def.Name
					}
					s.earnedItems = addEarnedItem(s.earnedItems, name, qty)
				}
			}

			for i := range s.enemies {
				if s.enemies[i].HP <= 0 && s.enemies[i].DeathPhase == 0 {
					s.enemies[i].DeathPhase = 1
					s.enemies[i].DeathTimer = 0.0
					s.enemies[i].Alpha = 1.0
				}
			}
			s.isWon = true
			s.game.Audio.PlayBGMWithIntro(bgmVictoryIntro, bgmVictoryLoop)
			s.battlePhase = phaseBattleEnd
		}
		return true
	}

	allDead := true
	for i := 0; i < partySize; i++ {
		if s.game.PlayerHP[i] > 0 {
			allDead = false
			break
		}
	}
	if allDead {
		s.gameOverIdx = 0
		s.isWon = false
		s.battlePhase = phaseBattleEnd
		s.battleLog = "全滅した…"
		s.battleLogTimer = gameOverMessageDuration
		s.damagePops = nil
		return true
	}
	return false
}

func (s *BattleScene) restoreDefeatedPartyHP() {
	for i := 0; i < partySize; i++ {
		if s.game.PlayerHP[i] <= 0 {
			s.game.PlayerHP[i] = 1
		}
	}
}

func (s *BattleScene) exitBattleToField() {
	s.restoreDefeatedPartyHP()
	field, _ := NewRoomScene(s.game, s.originMap, s.originX, s.originY, "", s.originDir)
	if strings.HasPrefix(s.enemyType, "boss_") {
		numStr := strings.TrimPrefix(s.enemyType, "boss_")
		if bossNum, err := strconv.Atoi(numStr); err == nil {
			if bossNum >= 1 && bossNum <= 4 {
				field.justDefeatedBoss = bossNum
			}
		}
	}
	if !s.game.SeenSkillUpgradeTutorial && partyHasEnoughSPToUpgrade(s.game) {
		s.game.SeenSkillUpgradeTutorial = true
		field.skillUpgradeTutorialActive = true
	}
	s.game.ChangeSceneWithFade(field, 0.5)
}

// partyHasEnoughSPToUpgrade reports whether any party member can afford to
// upgrade one of their unlocked skills, i.e. whether skill enhancement in
// the menu has just become possible.
func partyHasEnoughSPToUpgrade(game *Game) bool {
	for i := 0; i < partySize; i++ {
		for j := range game.CharacterSkills(i) {
			if game.IsSkillUnlocked(i, j) && game.CanUpgradeSkill(i, j) {
				return true
			}
		}
	}
	return false
}

func (s *BattleScene) enemyTargetsForAttack(isAll bool) []int {
	if isAll {
		return s.aliveEnemyIndices()
	}
	if s.targetIndex >= 0 && s.targetIndex < len(s.enemies) && s.enemies[s.targetIndex].HP > 0 {
		return []int{s.targetIndex}
	}
	return []int{s.firstAliveEnemySlot()}
}

func (s *BattleScene) updateTargetSelect() {
	p := s.waitingActor

	// Up/down cycles through the alive enemies and, for skills that can hit
	// everyone (and only while more than one enemy is alive), a trailing
	// "全体" row (targetIndex == maxEnemies) - the same way heal/item target
	// selection offers its "全員" row. Forced-all skills stay on that row.
	if alive := s.aliveEnemyIndices(); len(alive) > 0 && !s.currentTargetIsForcedAll() && (isMenuUpPressed() || isMenuDownPressed()) {
		options := alive
		if s.currentTargetAllowsAll() {
			options = append(append([]int(nil), alive...), maxEnemies)
		}
		curPos := 0
		for i, slot := range options {
			if slot == s.targetIndex {
				curPos = i
				break
			}
		}
		if isMenuDownPressed() {
			curPos = (curPos + 1) % len(options)
		} else {
			curPos = (curPos - 1 + len(options)) % len(options)
		}
		s.targetIndex = options[curPos]
		s.game.Audio.PlaySEByKey("cursor")
	}

	tappedIdx, tappedOk := s.hitTestEnemyTarget()
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.targetIndex, s.game.Audio)

	hadTouch := len(justPressedTouchPoints()) > 0
	if isEscapePressed() || (hadTouch && !tappedOk) {
		s.game.Audio.PlaySEByKey("cancel")
		s.pendingSynergy = false
		if s.pendingSkill >= 1 {
			s.battlePhase = phaseSkillMenu
		} else {
			s.battlePhase = phasePlayerMenu
		}
		return
	}

	if !isConfirmKeyPressed() && !tapped {
		return
	}

	if p < 0 || p >= partySize {
		return
	}

	if s.pendingSynergy {
		s.enterWaitStance(p)
		s.tryWaitSynergy()
		return
	}

	if s.pendingSkill >= 1 {
		skillIdx := s.pendingSkill - 1
		lv := s.lastSkillLevel[p][skillIdx]
		if lv < 1 {
			lv = 1
		}
		skills := s.game.CharacterSkills(p)
		data := skills[skillIdx].Levels[lv-1]
		isAll := s.currentAttackIsAllTarget()
		targets := s.enemyTargetsForAttack(isAll)

		s.attackAnimType = animSkill
		s.actionRow[p], s.actionFrames[p], _ = s.skillSpriteAnim(p, skillIdx)

		hits := make([]pendingPlayerHit, 0, len(targets))
		for _, slot := range targets {
			dmg := s.rollSkillDamage(p, skillIdx, lv, isAll, slot)
			hits = append(hits, pendingPlayerHit{slot: slot, dmg: dmg, crit: s.lastRollWasCrit, nonLethal: data.NonLethal})
		}
		s.pendingPlayerHits = hits

		if s.rewindActive {
			hits2 := make([]pendingPlayerHit, 0, len(targets))
			for _, slot := range targets {
				dmg2 := s.rollSkillDamage(p, skillIdx, lv, isAll, slot)
				hits2 = append(hits2, pendingPlayerHit{slot: slot, dmg: dmg2, crit: s.lastRollWasCrit, nonLethal: data.NonLethal})
			}
			s.pendingPlayerHits2 = hits2
			s.pendingDamage2Scheduled = true
		} else {
			s.pendingPlayerHits2 = nil
			s.pendingDamage2Scheduled = false
		}

		s.game.PlayerMP[p] -= s.effectiveMPCost(data.MPCost)
		s.battleLog = skills[skillIdx].Name
		s.battleLogTimer = skillActionLogDuration

		knockback := 0.0
		if data.Knockback > 0 {
			knockback = data.Knockback + data.KnockbackPerTG*float64(s.tgLevel()-1)
		}
		for _, slot := range targets {
			s.applySkillEffects(data.Effects, p, true, slot, isAll)
			if s.rewindActive {
				s.applySkillEffects(data.Effects, p, true, slot, isAll)
			}
			if knockback > 0 {
				s.reduceAtb(true, slot, knockback)
			}
			if data.Scan {
				s.enemies[slot].Scanned = true
			}
		}
		s.pendingDrainDivisor = data.DrainDivisor
		s.pendingDrainMax = data.DrainMax
		s.pendingDrainActor = p
		s.addGaugePoint(data.GaugePoint)

		s.activeAttacker = p
		s.attackPhaseTimer = 0.0
		s.hitStopTimer = 0.0
		s.battlePhase = phaseATB
		return
	}

	s.attackAnimType = animNormal
	s.pendingDrainDivisor = 0
	s.game.Audio.PlaySEByKey("attack_normal")
	targets := s.enemyTargetsForAttack(false)
	target := targets[0]
	firstDmg := s.rollNormalDamage(target)
	s.pendingPlayerHits = []pendingPlayerHit{{slot: target, dmg: firstDmg, crit: s.lastRollWasCrit}}
	if s.rewindActive {
		secondDmg := s.rollNormalDamage(target)
		s.pendingPlayerHits2 = []pendingPlayerHit{{slot: target, dmg: secondDmg, crit: s.lastRollWasCrit}}
		s.pendingDamage2Scheduled = true
	} else {
		s.pendingPlayerHits2 = nil
		s.pendingDamage2Scheduled = false
	}
	s.battleLog = ""
	s.battleLogTimer = silentActionLogDuration

	s.addGaugePoint(normalAttackGaugePoint)

	s.activeAttacker = p
	s.attackPhaseTimer = 0.0
	s.hitStopTimer = 0.0
	s.battlePhase = phaseATB
}

// pendingSkillData is the level data of the skill s.pendingSkill at the
// level the acting party member picked for it.
func (s *BattleScene) pendingSkillData() (SkillLevelData, bool) {
	p := s.waitingActor
	if p < 0 || p >= partySize || s.pendingSkill < 1 {
		return SkillLevelData{}, false
	}
	skillIdx := s.pendingSkill - 1
	skills := s.game.CharacterSkills(p)
	if skillIdx >= len(skills) {
		return SkillLevelData{}, false
	}
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 || lv > len(skills[skillIdx].Levels) {
		lv = 1
	}
	return skills[skillIdx].Levels[lv-1], true
}

// allyTargetValid reports whether party member i can receive data: fallen
// members for revives, living ones for everything else.
func (s *BattleScene) allyTargetValid(data SkillLevelData, i int) bool {
	if data.ReviveHPPercent > 0 {
		return s.game.PlayerHP[i] <= 0
	}
	return s.game.PlayerHP[i] > 0
}

func (s *BattleScene) allyAllTargets(data SkillLevelData) []int {
	var targets []int
	for i := 0; i < partySize; i++ {
		if s.allyTargetValid(data, i) {
			targets = append(targets, i)
		}
	}
	return targets
}

// allyTargetOptions lists the healTargetIndex values the cursor can move
// between for data: the four party slots unless the skill always hits
// everyone, plus the "全員" row (partySize) when the skill can.
func allyTargetOptions(data SkillLevelData) []int {
	var options []int
	if data.Target != TargetAll {
		for i := 0; i < partySize; i++ {
			options = append(options, i)
		}
	}
	if data.Target == TargetAll || data.Target == TargetBoth {
		options = append(options, partySize)
	}
	return options
}

// firstAllyTarget is where the ally target cursor starts for data.
func (s *BattleScene) firstAllyTarget(data SkillLevelData) int {
	if data.Target == TargetAll {
		return partySize
	}
	for i := 0; i < partySize; i++ {
		if s.allyTargetValid(data, i) {
			return i
		}
	}
	return 0
}

func (s *BattleScene) updateHealTargetSelect() {
	p := s.waitingActor
	data, ok := s.pendingSkillData()
	if !ok {
		s.battlePhase = phaseSkillMenu
		return
	}

	options := allyTargetOptions(data)
	if isMenuUpPressed() || isMenuDownPressed() {
		curPos := 0
		for i, idx := range options {
			if idx == s.healTargetIndex {
				curPos = i
				break
			}
		}
		if isMenuDownPressed() {
			curPos = (curPos + 1) % len(options)
		} else {
			curPos = (curPos - 1 + len(options)) % len(options)
		}
		if len(options) > 1 {
			s.game.Audio.PlaySEByKey("cursor")
		}
		s.healTargetIndex = options[curPos]
	}

	tappedIdx, tappedOk := s.hitTestHealTargets(data.Target != TargetSingle)
	if tappedOk && data.Target == TargetAll {
		tappedIdx = partySize
	}
	tapped := tapSelectOrConfirm(tappedIdx, tappedOk, &s.healTargetIndex, s.game.Audio)

	hadTouch := len(justPressedTouchPoints()) > 0
	if isEscapePressed() || (hadTouch && !tappedOk) {
		s.game.Audio.PlaySEByKey("cancel")
		s.battlePhase = phaseSkillMenu
		return
	}
	if !isConfirmKeyPressed() && !tapped {
		return
	}

	skillIdx := s.pendingSkill - 1
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 {
		lv = 1
	}
	skills := s.game.CharacterSkills(p)
	cost := s.effectiveMPCost(data.MPCost)

	isAll := s.healTargetIndex == partySize
	targets := []int{s.healTargetIndex}
	if isAll {
		targets = s.allyAllTargets(data)
	} else if !s.allyTargetValid(data, s.healTargetIndex) {
		targets = nil
	}
	if len(targets) == 0 || s.game.PlayerMP[p] < cost {
		s.game.Audio.PlaySEByKey("error")
		return
	}
	s.game.Audio.PlaySEByKey("heal")
	// 回復のアニメは最後に自分が回復を受ける部分まで描かれているので、
	// 自分が対象に入っていない時はその手前で終える。
	castRow, castFrames, cut := s.skillSpriteAnim(p, skillIdx)
	if cut > 0 && !slices.Contains(targets, p) {
		castFrames = min(cut, castFrames)
	}
	s.startCast(p, castRow, castFrames)
	s.game.PlayerMP[p] -= cost

	mpHeal := 0
	if data.TGMPHealLevel > 0 && s.tgLevel() >= data.TGMPHealLevel {
		mpHeal = data.MPHealSingle
		if isAll {
			mpHeal = data.MPHealAll
		}
	}

	// 全体回復は1回だけ回復量を振って全員に同じ量を配る。巻き戻し中の
	// 2回目の回復は、全体/単体とも対象ごとに振り直す。
	healAmount := 0
	if data.IsHeal {
		healAmount = s.rollSkillHeal(p, skillIdx, lv, isAll)
	}
	for _, target := range targets {
		if data.ReviveHPPercent > 0 {
			s.revivePlayer(target, data.ReviveHPPercent)
		}
		if data.IsHeal {
			s.applySkillHeal(target, healAmount, false)
		}
		if mpHeal > 0 {
			s.applySkillMPHeal(target, mpHeal)
		}
		if data.GuardHPPercent > 0 {
			s.guardHPPercent[target] = data.GuardHPPercent
			s.guardCount[target] = data.GuardCount
		}
		s.applySkillEffects(data.Effects, p, false, target, isAll)
		if s.rewindActive {
			if data.IsHeal {
				s.applySkillHeal(target, s.rollSkillHeal(p, skillIdx, lv, isAll), true)
			}
			s.applySkillEffects(data.Effects, p, false, target, isAll)
		}
		if target != p {
			recvRow := spriteRowBuffRecv
			if data.IsHeal || data.ReviveHPPercent > 0 || mpHeal > 0 {
				recvRow = spriteRowHealRecv
			}
			s.startRecv(target, recvRow)
		}
	}
	s.battleLog = skills[skillIdx].Name
	if isAll {
		s.battleLog += "（全体）"
	}
	s.battleLogTimer = battleLogDuration
	s.addGaugePoint(data.GaugePoint)

	s.finishPlayerTurn(s.pendingActionReturnPosition())
}

// executeSelfSkill uses the pending TargetSelf skill (かばう, カウンター)
// on the acting party member right away.
func (s *BattleScene) executeSelfSkill() {
	p := s.waitingActor
	data, ok := s.pendingSkillData()
	if !ok {
		return
	}
	s.game.Audio.PlaySEByKey("heal")
	castRow, castFrames, _ := s.skillSpriteAnim(p, s.pendingSkill-1)
	s.startCast(p, castRow, castFrames)
	s.game.PlayerMP[p] -= s.effectiveMPCost(data.MPCost)

	if data.CoverCount > 0 {
		s.coverCount[p] = data.CoverCount
		s.PlayerBuffs[p] = withoutCoverBuffs(s.PlayerBuffs[p])
		for _, e := range data.Effects {
			stats, isBuff, ok := effectStats(e.Type)
			if !ok || !isBuff {
				continue
			}
			for _, st := range stats {
				s.PlayerBuffs[p] = append(s.PlayerBuffs[p], Buff{Type: st, Percent: e.Percent, Cover: true})
			}
		}
	} else {
		s.applySkillEffects(data.Effects, p, false, p, false)
	}
	if data.CounterSeconds > 0 {
		s.counterTimer[p] = data.CounterSeconds
		s.counterBonus[p] = data.CounterBonus
	}

	s.battleLog = s.game.CharacterSkills(p)[s.pendingSkill-1].Name
	s.battleLogTimer = battleLogDuration
	s.addGaugePoint(data.GaugePoint)

	s.finishPlayerTurn(s.pendingActionReturnPosition())
}

// revivePlayer brings fallen party member i back with percent% of max HP.
func (s *BattleScene) revivePlayer(i, percent int) {
	if s.game.PlayerHP[i] > 0 {
		return
	}
	hp := max(s.game.PlayerMaxHP[i]*percent/100, 1)
	s.game.PlayerHP[i] = hp
	s.deadWaitStuck[i] = false
	s.placeOnTimeline(i, 0)
	s.playerPose[i] = poseIdle
	s.playerAnimTimer[i] = 0
	s.spawnDamagePop(DamagePop{
		Value:  hp,
		X:      s.partyScreenX[i] + spriteFrameW/2,
		Y:      s.partyScreenY[i] - partyDamagePopOffsetY,
		Vy:     -45.0,
		IsHeal: true,
	})
}

// applySkillMPHeal restores amount MP to target (capped at max MP) and
// shows it just below the HP heal popup.
func (s *BattleScene) applySkillMPHeal(target, amount int) {
	s.game.PlayerMP[target] = min(s.game.PlayerMP[target]+amount, s.game.PlayerMaxMP[target])
	s.spawnDamagePop(DamagePop{
		Value:  amount,
		X:      s.partyScreenX[target] + spriteFrameW/2,
		Y:      s.partyScreenY[target] - partyDamagePopOffsetY + 30.0,
		Vy:     -45.0,
		Timer:  -0.12,
		IsHeal: true,
		IsMP:   true,
	})
}

// coverRedirect returns the party member who takes a single-target enemy
// attack aimed at target: a living ally with かばう charges left (spending
// one), or target itself.
func (s *BattleScene) coverRedirect(target int) int {
	for c := 0; c < partySize; c++ {
		if c == target || s.coverCount[c] <= 0 || s.game.PlayerHP[c] <= 0 {
			continue
		}
		s.coverCount[c]--
		return c
	}
	return target
}

// tryGuard spends one 守りの祈り charge on target if dmg would leave them at
// or below the guard's HP threshold, reporting whether the hit is nullified.
func (s *BattleScene) tryGuard(target, dmg int) bool {
	if dmg <= 0 || s.guardCount[target] <= 0 {
		return false
	}
	threshold := s.game.PlayerMaxHP[target] * s.guardHPPercent[target] / 100
	if s.game.PlayerHP[target]-dmg > threshold {
		return false
	}
	s.guardCount[target]--
	return true
}

// counterAttack answers an enemy hit on party member p (in カウンター stance)
// with a normal attack on the enemy in slot.
func (s *BattleScene) counterAttack(p, slot int) {
	if slot < 0 || slot >= len(s.enemies) || s.enemies[slot].HP <= 0 {
		return
	}
	def := max(s.effectiveEnemyDef(slot, false), 1)
	power := (100.0 + float64(s.gaugeAtkBonus())) * (1.0 + float64(s.counterBonus[p])/100.0)
	dmg := s.rollDamage(float64(s.effectiveAtk(p)), power, float64(def), 1.0, s.game.PlayerLuck[p])
	crit := s.lastRollWasCrit
	s.applyDamageToEnemySlot(slot, dmg)
	s.game.Audio.PlaySEByKey("attack_normal")
	x, y, w, eh := s.enemyDrawRect(slot)
	s.spawnDamagePop(DamagePop{
		Value:  dmg,
		X:      x + w/2,
		Y:      y - eh*damagePopHeadOffsetRatio,
		Vy:     -180.0,
		Timer:  -0.2,
		IsCrit: crit,
	})
}

// applySkillHeal はtargetのHPを最大HPを上限にamount回復し、回復量を表示する。
// secondは巻き戻し中の2回目の回復で、1回目と重ならないよう少し上に遅れて出す。
func (s *BattleScene) applySkillHeal(target, amount int, second bool) {
	s.game.PlayerHP[target] = min(s.game.PlayerHP[target]+amount, s.game.PlayerMaxHP[target])
	pop := DamagePop{
		Value:  amount,
		X:      s.partyScreenX[target] + spriteFrameW/2,
		Y:      s.partyScreenY[target] - partyDamagePopOffsetY,
		Vy:     -45.0,
		IsHeal: true,
	}
	if second {
		pop.Y -= 10.0
		pop.Timer = -0.18
	}
	s.spawnDamagePop(pop)
}
