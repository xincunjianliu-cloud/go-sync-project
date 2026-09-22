package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const debugModeEnabled = true

// nextUnlockedSkillIndex moves the skill menu cursor by dir (+1/-1),
// wrapping around, skipping over skills the character hasn't unlocked yet.
func (s *BattleScene) nextUnlockedSkillIndex(charIdx, from, dir, n int) int {
	idx := from
	for i := 0; i < n; i++ {
		idx = (idx + dir + n) % n
		if s.game.IsSkillUnlocked(charIdx, idx) {
			return idx
		}
	}
	return from
}

func (s *BattleScene) firstUnlockedSkillIndex(charIdx int, skills []SkillDef) int {
	for i := range skills {
		if s.game.IsSkillUnlocked(charIdx, i) {
			return i
		}
	}
	return 0
}

// skillUnlockedByDrawLevel reports whether the level animation currently
// displayed for character i (s.drawPlayerLv[i]) has reached or passed a
// skill's UnlockLevel that wasn't already reached before this battle's
// level-up animation started (s.resultStartLevel[i]).
func (s *BattleScene) skillUnlockedByDrawLevel(i int) bool {
	for _, sk := range s.game.CharacterSkills(i) {
		if sk.UnlockLevel > s.resultStartLevel[i] && sk.UnlockLevel <= s.drawPlayerLv[i] {
			return true
		}
	}
	return false
}

// spawnDamagePop adds a damage/heal popup, nudging it sideways when another
// very recent popup already occupies roughly the same spot so consecutive
// hits on the same target don't render on top of each other.
func (s *BattleScene) spawnDamagePop(pop DamagePop) {
	const overlapRadiusX = 26.0
	const overlapRadiusY = 30.0
	const overlapFreshWindow = 0.2
	const stackStepX = 22.0
	const stackStepY = -10.0

	stack := 0
	for _, existing := range s.damagePops {
		if existing.Timer < 0 || existing.Timer > overlapFreshWindow {
			continue
		}
		if math.Abs(existing.X-pop.X) < overlapRadiusX && math.Abs(existing.Y-pop.Y) < overlapRadiusY {
			stack++
		}
	}
	if stack > 0 {
		dir := 1.0
		if stack%2 == 0 {
			dir = -1.0
		}
		step := float64((stack + 1) / 2)
		pop.X += dir * step * stackStepX
		pop.Y += step * stackStepY
	}
	s.damagePops = append(s.damagePops, pop)
}

func (s *BattleScene) applyDebugCheats() {
	if !debugModeEnabled {
		return
	}
	if !ebiten.IsKeyPressed(ebiten.KeyShift) {
		return
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.Key1):
		for i := range s.enemies {
			s.enemies[i].HP = 0
		}
		s.checkBattleEnd()

	case inpututil.IsKeyJustPressed(ebiten.Key2):
		for i := 0; i < partySize; i++ {
			s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
			s.game.PlayerMP[i] = s.game.PlayerMaxMP[i]
		}

	case inpututil.IsKeyJustPressed(ebiten.Key3):
		for i := 0; i < partySize; i++ {
			if s.game.PlayerHP[i] > 0 {
				s.game.PlayerHP[i] = 1
			}
		}

	case inpututil.IsKeyJustPressed(ebiten.Key4):
		for i := range s.enemies {
			s.enemies[i].Exp = 9999
		}

	case inpututil.IsKeyJustPressed(ebiten.Key5):
		for i := 0; i < partySize; i++ {
			s.game.PlayerMP[i] = 0
		}

	case inpututil.IsKeyJustPressed(ebiten.Key6):
		s.gaugePoint = gaugePoolMax
		s.recomputeGaugeStage()

	case inpututil.IsKeyJustPressed(ebiten.Key7):
		for i := 0; i < partySize; i++ {
			skills := s.game.CharacterSkills(i)
			for j, sk := range skills {
				if j >= len(s.game.PlayerSkillLv[i]) {
					continue
				}
				maxLv := len(sk.Levels)
				if maxLv > 0 {
					s.game.PlayerSkillLv[i][j] = maxLv
				}
			}
		}
		s.battleLog = "全スキルLv最大化（デバッグ）"
		s.battleLogTimer = battleLogDuration

	case inpututil.IsKeyJustPressed(ebiten.Key8):
		for i := 0; i < partySize; i++ {
			s.game.PlayerSP[i] += 9999
		}
		s.battleLog = "SP+9999（デバッグ）"
		s.battleLogTimer = battleLogDuration

	case inpututil.IsKeyJustPressed(ebiten.Key9):
		for _, def := range ItemDatabase {
			s.game.AddItem(def.ID, 5)
		}
		s.battleLog = "全アイテムを5個ずつ入手（デバッグ）"
		s.battleLogTimer = battleLogDuration

	case inpututil.IsKeyJustPressed(ebiten.Key0):
		for i := 0; i < partySize; i++ {
			s.game.PlayerHP[i] = 0
		}
		s.checkBattleEnd()

	case inpututil.IsKeyJustPressed(ebiten.KeyB):
		if s.debugStatIconTest {
			for i := 0; i < partySize; i++ {
				s.PlayerBuffs[i] = nil
				s.PlayerDebuffs[i] = nil
			}
			s.debugStatIconTest = false
			s.battleLog = "ステータスアイコンのテスト表示を解除（デバッグ）"
		} else {
			const testTurns = 999
			s.PlayerBuffs[0] = []Buff{{Type: StatAtk, Percent: 20, Turns: testTurns}}
			s.PlayerBuffs[1] = []Buff{
				{Type: StatAtk, Percent: 20, Turns: testTurns},
				{Type: StatDef, Percent: 20, Turns: testTurns},
				{Type: StatLuk, Percent: 20, Turns: testTurns},
			}
			s.PlayerDebuffs[1] = []Debuff{
				{Type: StatMat, Percent: 20, Turns: testTurns},
				{Type: StatMdf, Percent: 20, Turns: testTurns},
			}
			s.PlayerDebuffs[2] = []Debuff{{Type: StatDef, Percent: 20, Turns: testTurns}}
			s.PlayerBuffs[3] = []Buff{{Type: StatLuk, Percent: 20, Turns: testTurns}}
			s.PlayerDebuffs[3] = []Debuff{{Type: StatDef, Percent: 20, Turns: testTurns}}
			s.debugStatIconTest = true
			s.battleLog = "ステータスアイコンのテスト表示（デバッグ）"
		}
		s.battleLogTimer = battleLogDuration

	case inpututil.IsKeyJustPressed(ebiten.KeyT):
		s.tutorialKind = battleTutorialKindBasics
		s.tutorialPage = 0
		s.tutorialActive = true

	case inpututil.IsKeyJustPressed(ebiten.KeyG):
		s.tutorialKind = battleTutorialKindGauge
		s.tutorialPage = 0
		s.tutorialActive = true

	case inpututil.IsKeyJustPressed(ebiten.KeyK):
		for i := 0; i < partySize; i++ {
			skills := s.game.CharacterSkills(i)
			for j, sk := range skills {
				if j >= len(s.game.PlayerSkillLv[i]) {
					continue
				}
				if len(sk.Levels) > 1 {
					s.game.PlayerSkillLv[i][j] = 2
				}
			}
		}
		s.battleLog = "全キャラのスキルLvを2に設定（デバッグ）"
		s.battleLogTimer = battleLogDuration
	}
}

func (s *BattleScene) triggerShake(tier hitTier) {
	switch tier {
	case hitTierNone:
		return
	case hitTierWeak:
		s.shakeType = 2
		s.shakeTimer = 0.2
		s.shakeMaxDur = 0.2
		s.shakePower = 5.0
	case hitTierStrong:
		s.shakeType = 3
		s.shakeTimer = 0.6
		s.shakeMaxDur = 0.6
		s.shakePower = 16.0
	case hitTierSynergy:
		s.shakeType = 3
		s.shakeTimer = 0.6
		s.shakeMaxDur = 0.8
		s.shakePower = 16.0
	}
}

func (s *BattleScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}

	s.applyDebugCheats()

	s.gaugeColorAnimTimer += dt
	s.updateEnemyDeath(dt)

	if s.introActive {
		s.introPhaseTimer += dt
		switch s.introPhase {
		case 0:
			s.introOffsetX += (0.0 - s.introOffsetX) * (1.0 - math.Pow(introHorizDecay, dt))
			s.introProgress = 1.0 - s.introOffsetX/introStartOffsetX
			if s.introProgress < 0 {
				s.introProgress = 0
			}
			if s.introProgress > 1 {
				s.introProgress = 1
			}
			if math.Abs(s.introOffsetX) < 0.5 {
				s.introOffsetX = 0
				s.introProgress = 1.0
			}
			if s.introOffsetX == 0 && s.introPhaseTimer >= introHorizToVertWait {
				s.introPhase = 1
				s.introPhaseTimer = 0
			}

		case 1:
			s.introVertOffsetY += (0.0 - s.introVertOffsetY) * (1.0 - math.Pow(introVertDecay, dt))
			if math.Abs(s.introVertOffsetY) < 0.5 {
				s.introVertOffsetY = 0
				s.introPhase = 2
				s.introPhaseTimer = 0
			}

		case 2:
			if s.introPhaseTimer >= introGoalWait {
				s.introPhase = 3
				s.introPhaseTimer = 0
			}

		case 3:
			for i := 0; i < partySize; i++ {
				if s.introCharOffsetX != 0 {
					if s.playerPose[i] != poseWalk {
						s.playerPose[i] = poseWalk
						s.playerAnimTimer[i] = 0
					}
					s.playerAnimTimer[i] += dt
				} else if s.playerPose[i] != poseIdle {
					s.playerPose[i] = poseIdle
					s.playerAnimTimer[i] = 0
				}
			}
			s.introCharOffsetX += (0.0 - s.introCharOffsetX) * (1.0 - math.Pow(introCharDecay, dt))
			if math.Abs(s.introCharOffsetX) < 0.5 {
				s.introCharOffsetX = 0
				s.introPhase = 4
				s.introPhaseTimer = 0
			}

		case 4:
			if s.introPhaseTimer >= introATBWait {
				s.introActive = false
				s.introProgress = 1.0
			}
		}
		return s
	}

	if s.tutorialActive {
		if isResultAdvancePressed() {
			s.game.Audio.PlaySEByKey("decide")
			s.tutorialPage++
			if s.tutorialPage >= battleTutorialPageCountFor(s.tutorialKind) {
				s.tutorialActive = false
			}
		}
		return s
	}

	if s.shakeTimer > 0 {
		s.shakeTimer -= dt
		if s.shakeTimer <= 0 {
			s.shakeTimer = 0
			s.shakeX = 0
			s.shakeY = 0
		} else {
			progress := s.shakeTimer / s.shakeMaxDur
			currentPower := s.shakePower * progress

			switch s.shakeType {
			case 1:
				s.shakeX = 0
				s.shakeY = math.Sin(s.shakeTimer*60.0) * currentPower
			case 2:
				s.shakeY = 0
				s.shakeX = math.Sin(s.shakeTimer*90.0) * currentPower
			case 3:
				bellCurve := math.Sin(progress * math.Pi)
				bellPower := s.shakePower * bellCurve
				s.shakeX = (rand.Float64()*2.0 - 1.0) * bellPower
				s.shakeY = (rand.Float64()*2.0 - 1.0) * bellPower
			}
		}
	}

	if s.hitStopTimer > 0 {
		s.hitStopTimer -= dt
		if s.hitStopTimer <= 0 {
			s.hitStopTimer = 0

			secondHasDamage := false
			for _, h := range s.pendingPlayerHits2 {
				if h.dmg > 0 {
					secondHasDamage = true
					break
				}
			}
			initialTimer, dy := 0.0, 0.0
			if s.pendingDamage2Scheduled && !secondHasDamage {
				initialTimer = -0.18
				dy = -8.0
				s.pendingDamage2Scheduled = false
			}

			maxDmg := 0
			for _, hit := range s.pendingPlayerHits {
				s.applyDamageToEnemySlot(hit.slot, hit.dmg)
				if hit.dmg > maxDmg {
					maxDmg = hit.dmg
				}
				if hit.dmg <= 0 {
					continue
				}
				if hit.crit {
					s.game.Audio.PlaySEByKey("critical")
				} else {
					s.game.Audio.PlaySEByKey("damage")
				}
				x, y, w, eh := s.enemyDrawRect(hit.slot)
				s.spawnDamagePop(DamagePop{
					Value:  hit.dmg,
					X:      x + w/2,
					Y:      y - eh*damagePopHeadOffsetRatio + dy,
					Vy:     -180.0,
					Timer:  initialTimer,
					IsCrit: hit.crit,
				})
			}

			if maxDmg > 0 {
				tier := hitTierWeak
				if s.attackAnimType != animNormal {
					tier = hitTierStrong
				}
				s.triggerShake(tier)
			}

			if s.pendingDamage2Scheduled && secondHasDamage {
				s.pendingDamage2Scheduled = false
				s.pendingPlayerHits = s.pendingPlayerHits2
				s.pendingPlayerHits2 = nil
				s.activeAttacker = s.waitingActor
				s.attackPhaseTimer = 0.0
				return s
			}
			s.pendingPlayerHits = nil
			s.pendingPlayerHits2 = nil

			s.enemyActionWaitTimer = 0.8
		}
		return s
	}

	if s.enemyHitStopTimer > 0 {
		s.enemyHitStopTimer -= dt
		if s.enemyHitStopTimer <= 0 {
			s.enemyHitStopTimer = 0
			s.applyEnemyPendingHits()
		}
		return s
	}

	for i := range s.enemies {
		if s.enemies[i].HitFlashTimer > 0 {
			s.enemies[i].HitFlashTimer -= dt
			if s.enemies[i].HitFlashTimer < 0 {
				s.enemies[i].HitFlashTimer = 0
			}
		}
	}

	for i := 0; i < partySize; i++ {
		if s.playerFlashTimer[i] > 0 {
			s.playerFlashTimer[i] -= dt
			if s.playerFlashTimer[i] < 0 {
				s.playerFlashTimer[i] = 0
			}
		}
	}

	if s.battlePhase == phaseBattleEnd && !s.isWon {
		if s.battleLogTimer > 0 {
			s.battleLogTimer -= dt
			if s.battleLogTimer < 0 {
				s.battleLogTimer = 0
			}
			return s
		}
		if isMenuUpPressed() || isMenuDownPressed() {
			s.gameOverIdx = (s.gameOverIdx + 1) % 2
			s.game.Audio.PlaySEByKey("cursor")
		}
		if isConfirmKeyPressed() {
			s.game.Audio.PlaySEByKey("decide")
			if s.gameOverIdx == 0 {
				for i := 0; i < partySize; i++ {
					s.game.PlayerHP[i] = s.preBattlePlayerHP[i]
					s.game.PlayerMP[i] = s.preBattlePlayerMP[i]
				}
				return NewBattleScene(s.game, s.originMap, s.originX, s.originY, s.originDir, s.enemyType, s.enemyNames)
			}
			return NewTitleScene(s.game)
		}
		return s
	}

	const returnDelayDuration = 0.4

	for i := 0; i < partySize; i++ {
		if s.waitCancelHold[i] > 0 {
			s.waitCancelHold[i] -= dt
			if s.waitCancelHold[i] < 0 {
				s.waitCancelHold[i] = 0
				s.resetPlayerGaugeTo(i, waitCancelReturnPosition)
				s.readySlideX[i] = 0
			}
		}
		isAttacking := (s.activeAttacker == i) || (s.hitStopTimer > 0 && s.playerPose[i] != poseIdle && s.playerPose[i] != poseDamage)
		isSelecting := s.playerPose[i] == poseReady || s.playerPose[i] == poseReadyGlow || s.playerPose[i] == poseHealCast

		if isSelecting || isAttacking {
			s.readySlideX[i] += (-100.0 - s.readySlideX[i]) * (1.0 - math.Pow(0.001, dt))
			s.returnDelayTimer[i] = returnDelayDuration
		} else if s.returnDelayTimer[i] > 0 {
			s.returnDelayTimer[i] -= dt
		} else {
			s.readySlideX[i] += (0.0 - s.readySlideX[i]) * (1.0 - math.Pow(0.01, dt))
		}
	}

	if s.battleLogTimer > 0 {
		s.battleLogTimer -= dt
		if s.battleLogTimer <= 0 {
			s.battleLogTimer = 0
			s.battleLog = ""
		}
	}

	if s.battlePhase == phaseATB &&
		s.activeAttacker < 0 && s.hitStopTimer <= 0 &&
		s.enemyActionWaitTimer <= 0 &&
		!s.introActive {
		s.updateRewind(dt)
	}

	s.updateAllPoses(dt)

	if s.activeAttacker >= 0 {
		s.attackPhaseTimer += dt
		p := s.activeAttacker

		switch s.attackAnimType {
		case animCharge:
			const approachDur = 0.20
			const attackDur = 0.45
			if s.attackPhaseTimer < approachDur {
				s.playerPose[p] = poseChargeApproach
				s.chargeApproachOffset = (s.attackPhaseTimer / approachDur) * 80.0
			} else if s.attackPhaseTimer < approachDur+attackDur {
				s.playerPose[p] = poseChargeAttack
			} else {
				s.hitStopTimer = hitStopStrong
				s.activeAttacker = -1
				s.chargeApproachOffset = 0
			}

		case animFireMagic:
			const castDur = 0.30
			const loopDur = 0.30
			if s.attackPhaseTimer < castDur {
				if s.playerPose[p] != poseFireCast {
					s.playerPose[p] = poseFireCast
					s.playerAnimTimer[p] = 0
				}
			} else if s.attackPhaseTimer < castDur+loopDur {
				if s.playerPose[p] != poseFireLoop {
					s.playerPose[p] = poseFireLoop
					s.playerAnimTimer[p] = 0
				}
			} else {
				s.hitStopTimer = hitStopStrong
				s.activeAttacker = -1
			}

		default:
			const atkDur = 0.45
			s.playerPose[p] = poseAttack
			if s.attackPhaseTimer >= atkDur {
				s.hitStopTimer = hitStopWeak
				s.activeAttacker = -1
			}
		}
		return s
	}

	var activePops []DamagePop
	for _, pop := range s.damagePops {
		pop.Timer += dt
		if pop.IsHeal {
			pop.Vy *= math.Pow(0.05, dt)
			pop.Y += pop.Vy * dt
		}
		if pop.Timer <= 1.6 {
			activePops = append(activePops, pop)
		}
	}
	s.damagePops = activePops

	if s.enemyWindupTimer > 0 {
		s.enemyWindupTimer -= dt
		if s.enemyWindupTimer <= 0 {
			s.enemyWindupTimer = 0
			s.rollEnemyAction()
		}
		return s
	}

	if s.enemyActionWaitTimer > 0 {
		s.enemyActionWaitTimer -= dt
		if s.enemyActionWaitTimer <= 0 {
			s.enemyActionWaitTimer = 0
			if s.enemyIsActing {
				s.enemyIsActing = false
				s.resetEnemyGaugeTo(s.actingEnemySlot, s.enemyActionReturnPosition)
			}

			for i := 0; i < partySize; i++ {
				if s.playerPose[i] != poseDead && s.playerPose[i] != poseWin {
					s.playerPose[i] = poseIdle
					s.playerAnimTimer[i] = 0
				}
			}

			s.battlePhase = phaseATB
			if s.checkBattleEnd() {
				return s
			}
			s.finishPlayerTurn(s.pendingActionReturnPosition())
		}
		return s
	}

	switch s.battlePhase {
	case phaseATB:
		if s.battleLogTimer <= 0 && !s.anyActorHolding() {
			s.tickATB(dt)
		}
		s.tryStartNextActor()

	case phasePlayerMenu:
		if next := s.updatePlayerMenu(); next != nil {
			return next
		}

	case phaseMessage:
		if s.battleLogTimer <= 0 {
			if s.fleeSucceeded {
				s.restoreDefeatedPartyHP()
				field, _ := NewRoomScene(s.game, s.originMap, s.originX, s.originY, "", s.originDir)
				return field
			}
			s.battlePhase = phasePlayerMenu
		}
	case phaseSkillMenu:
		s.updateSkillMenu(dt)

	case phaseTargetSelect:
		s.updateTargetSelect()

	case phaseHealSelect:
		s.updateHealTargetSelect()

	case phaseItemMenu:
		s.updateItemMenu(dt)

	case phaseItemTarget:
		s.updateItemTargetSelect()

	case phaseBattleEnd:
		if !s.allEnemyDeathAnimDone() {
			return s
		}

		switch s.resultSubPhase {

		case resSubStillWait:
			s.resultAnimTimer += dt
			if s.resultAnimTimer >= 0.8 {
				s.resultAnimTimer = 0.0
				s.resultSubPhase = resSubWinPose
				for i := 0; i < partySize; i++ {
					if s.game.PlayerHP[i] > 0 {
						s.playerPose[i] = poseWin
						s.playerAnimTimer[i] = 0
					}
				}
			}

		case resSubWinPose:
			s.resultAnimTimer += dt

			if s.resultAnimTimer >= 0.6 {
				s.resultAnimTimer = 0.0
				s.resultSubPhase = resSubResultFadeIn

				for i := 0; i < partySize; i++ {
					s.drawPlayerEXP[i] = s.expStartEXP[i]
					s.drawPlayerEXPF[i] = float64(s.expStartEXP[i])
					s.isLevelUp[i] = s.game.PlayerLv[i] > s.drawPlayerLv[i]
					s.resultStartLevel[i] = s.drawPlayerLv[i]
					ratio := float64(s.totalEnemyExp()) / float64(s.drawPlayerMaxEXP[i])
					if ratio > 1.0 {
						ratio = 1.0
					}
				}
			}

		case resSubResultFadeIn:
			s.resultAnimTimer += dt
			if s.resultAnimTimer >= 0.5 {
				s.resultAnimTimer = 0.0
				s.resultSubPhase = resSubBarAnimate
			}

		case resSubBarAnimate:
			if isResultAdvancePressed() {
				for i := 0; i < partySize; i++ {
					if s.game.PlayerHP[i] <= 0 {
						continue
					}
					s.levelUpPauseTimer[i] = 0
					if s.drawPlayerLv[i] < s.game.PlayerLv[i] {
						s.drawPlayerLv[i]++
						s.game.Audio.PlaySEByKey("level_up")
						if s.drawPlayerLv[i] == s.game.PlayerLv[i] {
							s.drawPlayerMaxEXP[i] = s.game.PlayerNextEXP[i]
						} else {
							s.drawPlayerMaxEXP[i] = s.drawPlayerLv[i] * 50
						}
						s.drawPlayerEXP[i] = 0
						s.drawPlayerEXPF[i] = 0.0
						s.expStartEXP[i] = 0
					} else {
						s.drawPlayerEXP[i] = s.game.PlayerEXP[i]
						s.drawPlayerEXPF[i] = float64(s.game.PlayerEXP[i])
						s.drawPlayerMaxEXP[i] = s.game.PlayerNextEXP[i]
						s.drawPlayerLv[i] = s.game.PlayerLv[i]
					}
				}

				allDone := true
				for i := 0; i < partySize; i++ {
					if s.game.PlayerHP[i] <= 0 {
						continue
					}
					if s.drawPlayerLv[i] < s.game.PlayerLv[i] || s.drawPlayerEXP[i] < s.game.PlayerEXP[i] {
						allDone = false
						break
					}
				}
				if allDone {
					s.resultSubPhase = resSubDoneWait
				}
				return s
			}

			allFinished := true
			for i := 0; i < partySize; i++ {
				if s.game.PlayerHP[i] <= 0 {
					continue
				}

				if s.levelUpPauseTimer[i] > 0 {
					allFinished = false
					s.levelUpPauseTimer[i] -= dt
					if s.levelUpPauseTimer[i] <= 0 {
						s.levelUpPauseTimer[i] = 0
						s.drawPlayerLv[i]++
						s.game.Audio.PlaySEByKey("level_up")
						if s.drawPlayerLv[i] == s.game.PlayerLv[i] {
							s.drawPlayerMaxEXP[i] = s.game.PlayerNextEXP[i]
						} else {
							s.drawPlayerMaxEXP[i] = s.drawPlayerLv[i] * 50
						}
						s.drawPlayerEXP[i] = 0
						s.drawPlayerEXPF[i] = 0.0
					}
					continue
				}

				var targetEXP int
				if s.drawPlayerLv[i] < s.game.PlayerLv[i] {
					targetEXP = s.drawPlayerMaxEXP[i]
				} else {
					targetEXP = s.game.PlayerEXP[i]
				}

				if s.drawPlayerEXP[i] < targetEXP {
					allFinished = false
				}

				s.drawPlayerEXPF[i] += float64(s.drawPlayerMaxEXP[i]) * 0.3 * dt
				if s.drawPlayerEXPF[i] > float64(targetEXP) {
					s.drawPlayerEXPF[i] = float64(targetEXP)
				}
				s.drawPlayerEXP[i] = int(s.drawPlayerEXPF[i])

				if s.drawPlayerEXP[i] >= targetEXP {
					s.drawPlayerEXP[i] = targetEXP
					s.drawPlayerEXPF[i] = float64(targetEXP)
					if s.drawPlayerLv[i] < s.game.PlayerLv[i] {
						s.levelUpPauseTimer[i] = levelUpPauseDuration
						allFinished = false
					}
				}
			}
			if allFinished {
				s.resultSubPhase = resSubDoneWait
			}

		case resSubDoneWait:
			if isResultAdvancePressed() {
				s.game.Audio.PlaySEByKey("decide")
				s.exitBattleToField()
			}
		}
		return s
	}
	return s
}

func (s *BattleScene) updateEnemyDeath(dt float64) {
	for i := range s.enemies {
		e := &s.enemies[i]
		switch e.DeathPhase {
		case 1:
			e.DeathTimer += dt
			e.Alpha = 1.0 - (e.DeathTimer / 0.6)
			if e.DeathTimer >= 0.6 {
				e.Alpha = 0.0
				e.DeathPhase = 2
				e.DeathTimer = 0.0
				cx, cy := s.enemyCenter(i)
				_, _, w, h := s.enemyDrawRect(i)
				spreadX := w / 4
				spreadY := h / 8
				for n := 0; n < 50; n++ {
					angle := rand.Float64() * math.Pi * 2
					speed := rand.Float64()*15 + 5
					s.deathParticles = append(s.deathParticles, DeathParticle{
						X:    cx + rand.Float64()*spreadX*2 - spreadX,
						Y:    cy + rand.Float64()*spreadY*2 - spreadY,
						Vx:   math.Cos(angle) * speed,
						Vy:   math.Sin(angle) * speed,
						Life: 1.0,
						Size: rand.Float64()*2.5 + 0.5,
					})
				}
			}
		case 2:
			e.DeathTimer += dt
			if e.DeathTimer >= 1.8 {
				e.DeathPhase = 3
			}
		}
	}

	if len(s.deathParticles) == 0 {
		return
	}
	alive := s.deathParticles[:0]
	for i := range s.deathParticles {
		p := &s.deathParticles[i]
		p.Vx += (rand.Float64()*40 - 20) * dt
		p.Vy += (rand.Float64()*40 - 20) * dt
		maxSpeed := 30.0
		if p.Vx > maxSpeed {
			p.Vx = maxSpeed
		}
		if p.Vx < -maxSpeed {
			p.Vx = -maxSpeed
		}
		if p.Vy > maxSpeed {
			p.Vy = maxSpeed
		}
		if p.Vy < -maxSpeed {
			p.Vy = -maxSpeed
		}
		p.X += p.Vx * dt
		p.Y += p.Vy * dt
		p.Life -= dt / 1.8
		if p.Life > 0 {
			alive = append(alive, *p)
		}
	}
	s.deathParticles = alive
}

func (s *BattleScene) updateAllPoses(dt float64) {
	for i := 0; i < partySize; i++ {
		if s.evadeOffsetX[i] > 0 {
			s.evadeOffsetX[i] -= evadeDodgeReturnSpeed * dt
			if s.evadeOffsetX[i] < 0 {
				s.evadeOffsetX[i] = 0
			}
		}

		if s.playerPose[i] == poseHealCast && i != s.healingCaster {
			s.playerPose[i] = poseIdle
			s.playerAnimTimer[i] = 0
			s.healingAnimTimer[i] = 0
		}
		s.playerAnimTimer[i] += dt
		if s.game.PlayerHP[i] <= 0 {
			if s.playerPose[i] != poseDead {
				s.playerPose[i] = poseDead
				s.playerAnimTimer[i] = 0
			}
			continue
		}
		if s.waitCancelHold[i] > 0 {
			s.playerPose[i] = poseDefend
			continue
		}

		if s.activeAttacker == i {
			continue
		}

		if s.hitStopTimer > 0 {
			continue
		}

		if s.playerPose[i] == poseDamage {
			if s.playerAnimTimer[i] >= hitFlashDuration {
				s.playerPose[i] = poseIdle
				s.playerAnimTimer[i] = 0
			}
			continue
		}

		if s.battlePhase == phaseBattleEnd {
			if s.isWon && s.resultSubPhase >= resSubWinPose && s.playerPose[i] != poseWin {
				s.playerPose[i] = poseWin
				s.playerAnimTimer[i] = 0
			}
			continue
		}

		if s.playerPose[i] != poseReady && s.playerPose[i] != poseReadyGlow && s.playerPose[i] != poseHealCast && s.readySlideX[i] < -0.5 {
			if s.playerPose[i] != poseIdle {
				s.playerPose[i] = poseIdle
				s.playerAnimTimer[i] = 0
			}
			continue
		}

		if isLowHP(s.game.PlayerHP[i], s.game.PlayerMaxHP[i]) {
			if s.playerPose[i] != poseLowHP {
				s.playerPose[i] = poseLowHP
				s.playerAnimTimer[i] = 0
			}
			continue
		}

		if s.waitingActor != i && s.waitStance[i] {
			if s.playerPose[i] != poseDefend {
				s.playerPose[i] = poseDefend
				s.playerAnimTimer[i] = 0
			}
			continue
		}

		if s.waitingActor == i &&
			(s.battlePhase == phasePlayerMenu ||
				s.battlePhase == phaseSkillMenu ||
				s.battlePhase == phaseTargetSelect ||
				s.battlePhase == phaseHealSelect ||
				s.battlePhase == phaseItemMenu ||
				s.battlePhase == phaseItemTarget) {

			isSkillSelecting := s.battlePhase == phaseSkillMenu || s.battlePhase == phaseHealSelect ||
				(s.battlePhase == phaseTargetSelect && s.pendingSkill >= 1)

			if isSkillSelecting {
				lv := s.skillLevelCursors[i][s.skillIndex]
				if lv < 1 {
					lv = 1
				}
				if lv > 3 {
					lv = 3
				}
				s.skillGlowLevel = lv
				if s.playerPose[i] != poseReadyGlow {
					s.playerPose[i] = poseReadyGlow
					s.playerAnimTimer[i] = 0
				}
			} else if s.playerPose[i] != poseReady {
				s.playerPose[i] = poseReady
				s.playerAnimTimer[i] = 0
			}
			continue
		}

		if s.playerPose[i] == poseCast || s.playerPose[i] == poseHealCast {
			if s.playerPose[i] == poseHealCast && i == s.healingCaster && s.healingAnimTimer[i] > 0 {
				s.healingAnimTimer[i] -= dt
				if s.healingAnimTimer[i] <= 0 {
					s.healingAnimTimer[i] = 0
					s.endCast(i)
				}
			}
			continue
		}

		if s.playerPose[i] != poseIdle {
			s.playerPose[i] = poseIdle
			s.playerAnimTimer[i] = 0
		}
	}
}
