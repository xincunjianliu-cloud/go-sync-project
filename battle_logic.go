package main

// battle_logic.go: バトルのメインUpdateループ・デバッグチート・敵死亡演出

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const debugModeEnabled = true

func (s *BattleScene) applyDebugCheats() {
	if !debugModeEnabled {
		return
	}
	if !ebiten.IsKeyPressed(ebiten.KeyShift) {
		return
	}

	switch {
	case inpututil.IsKeyJustPressed(ebiten.Key1):
		s.enemyHP = 0
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
		s.enemyExp = 9999

	case inpututil.IsKeyJustPressed(ebiten.Key5):
		for i := 0; i < partySize; i++ {
			s.game.PlayerMP[i] = 0
		}

	case inpututil.IsKeyJustPressed(ebiten.Key6):
		s.gaugePoint = gaugePoolMax
		s.recomputeGaugeStage()
		s.battleLog = "ゲージが満タンになった！"
		s.battleLogTimer = battleLogDuration

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
	}
}

func (s *BattleScene) Update(dt float64) Scene {
	if s == nil {
		return s
	}

	s.applyDebugCheats()

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
					s.playerAnimTimer[i] += dt // ★追加：スライドイン中もコマを進める
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
			s.enemyHP -= s.pendingDamage
			if s.enemyHP < 0 {
				s.enemyHP = 0
			}

			if s.pendingDamage > 0 {
				if s.pendingDamageShake >= 15.0 {
					s.flashAlpha = 0.6
					s.shakeType = 3
					s.shakeTimer = 0.6
					s.shakeMaxDur = 0.6
					s.shakePower = 16.0
				} else {
					s.shakeType = 2
					s.shakeTimer = 0.2
					s.shakeMaxDur = 0.2
					s.shakePower = 5.0
				}
			}

			initialTimer := 0.0
			dx := 0.0
			dy := 0.0
			if s.pendingDamage2Scheduled && s.pendingDamage2 == 0 {
				initialTimer = -0.18
				dx = -12.0
				dy = -8.0
				s.pendingDamage2Scheduled = false
			}

			if s.pendingDamage > 0 {
				s.damagePops = append(s.damagePops, DamagePop{
					Value: s.pendingDamage,
					X:     s.pendingDamageX + dx,
					Y:     s.pendingDamageY + dy,
					Vy:    -180.0,
					Timer: initialTimer,
				})
			}

			if s.pendingDamage2 > 0 {
				second := s.pendingDamage2
				s.pendingDamage2 = 0
				s.pendingDamage = second
				s.activeAttacker = s.waitingActor
				s.attackPhaseTimer = 0.0
				return s
			}

			s.enemyActionWaitTimer = 0.8
		}
		return s
	}

	if s.flashAlpha > 0 {
		s.flashAlpha -= dt * 4.0
		if s.flashAlpha < 0 {
			s.flashAlpha = 0
		}
	}

	const returnDelayDuration = 0.4 // ← 調整用：アニメーション終了後、戻り始めるまでの秒数

	for i := 0; i < partySize; i++ {
		if s.waitCancelHold[i] > 0 {
			s.waitCancelHold[i] -= dt
			if s.waitCancelHold[i] < 0 {
				s.waitCancelHold[i] = 0
				s.atbGauge[i] = 0
				s.readySlideX[i] = 0
			}
		}
		isAttacking := (s.activeAttacker == i) || (s.hitStopTimer > 0 && s.playerPose[i] != poseIdle && s.playerPose[i] != poseDamage)
		isSelecting := s.playerPose[i] == poseReady || s.playerPose[i] == poseReadyGlow || s.playerPose[i] == poseHealCast

		if isSelecting || isAttacking {
			s.readySlideX[i] += (-60.0 - s.readySlideX[i]) * (1.0 - math.Pow(0.001, dt))
			s.returnDelayTimer[i] = returnDelayDuration // 行動中は常に0.2秒にリセット
		} else if s.returnDelayTimer[i] > 0 {
			s.returnDelayTimer[i] -= dt // 待機中は位置を維持（decayさせない）
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

	// ← 変更：攻撃演出をanimNormal/animCharge/animFireMagicで分岐
	if s.activeAttacker >= 0 {
		s.attackPhaseTimer += dt
		p := s.activeAttacker

		switch s.attackAnimType {
		case animCharge:
			const approachDur = 0.20
			const attackDur = 0.45
			if s.attackPhaseTimer < approachDur {
				s.playerPose[p] = poseChargeApproach
				s.chargeApproachOffset = (s.attackPhaseTimer / approachDur) * 40.0
			} else if s.attackPhaseTimer < approachDur+attackDur {
				s.playerPose[p] = poseChargeAttack
			} else {
				s.hitStopTimer = 0.15
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
				s.hitStopTimer = 0.15
				s.activeAttacker = -1
			}

		default: // animNormal
			const atkDur = 0.45
			s.playerPose[p] = poseAttack
			if s.attackPhaseTimer >= atkDur {
				s.hitStopTimer = 0.15
				s.activeAttacker = -1
			}
		}
		return s
	}

	var activePops []DamagePop
	for _, pop := range s.damagePops {
		pop.Timer += dt
		pop.Vy *= math.Pow(0.05, dt)
		pop.Y += pop.Vy * dt
		if pop.Timer <= 1.6 {
			activePops = append(activePops, pop)
		}
	}
	s.damagePops = activePops

	if s.enemyActionWaitTimer > 0 {
		s.enemyActionWaitTimer -= dt
		if s.enemyActionWaitTimer <= 0 {
			s.enemyActionWaitTimer = 0

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
			s.finishPlayerTurn(true)
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
		if !s.isWon {
			if s.battleLogTimer > 0 {
				return s
			}
			if isMenuUpPressed() || isMenuDownPressed() {
				s.gameOverIdx = (s.gameOverIdx + 1) % 2
			}
			if isConfirmKeyPressed() {
				if s.gameOverIdx == 0 {
					for i := 0; i < partySize; i++ {
						s.game.PlayerHP[i] = s.preBattlePlayerHP[i]
						s.game.PlayerMP[i] = s.preBattlePlayerMP[i]
					}
					return NewBattleScene(s.game, s.originMap, s.originX, s.originY, s.originDir, s.enemyType, s.enemyName)
				}
				return NewTitleScene(s.game)
			}
			return s
		}

		if s.enemyDeathPhase < 3 {
			s.updateEnemyDeath(dt)
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
					ratio := float64(s.enemyExp) / float64(s.drawPlayerMaxEXP[i])
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
			if isConfirmKeyPressed() {
				for i := 0; i < partySize; i++ {
					if s.game.PlayerHP[i] <= 0 {
						continue
					}
					s.levelUpPauseTimer[i] = 0
					if s.drawPlayerLv[i] < s.game.PlayerLv[i] {
						s.drawPlayerLv[i]++
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

			const expPerSec = 5.0
			allFinished := true
			for i := 0; i < partySize; i++ {
				if s.game.PlayerHP[i] <= 0 {
					continue
				}

				// ゲージが右端まで到達済み：しばらく満タン表示のまま待ってからレベルアップ処理をする。
				if s.levelUpPauseTimer[i] > 0 {
					allFinished = false
					s.levelUpPauseTimer[i] -= dt
					if s.levelUpPauseTimer[i] <= 0 {
						s.levelUpPauseTimer[i] = 0
						s.drawPlayerLv[i]++
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
						// ゲージが右端に達した状態をひと呼吸見せてからレベルアップする。
						s.levelUpPauseTimer[i] = levelUpPauseDuration
						allFinished = false
					}
				}
			}
			if allFinished {
				s.resultSubPhase = resSubDoneWait
			}

		case resSubDoneWait:
			if isConfirmKeyPressed() {
				s.exitBattleToField()
			}
		}
		return s
	}
	return s
}

func (s *BattleScene) updateEnemyDeath(dt float64) {
	switch s.enemyDeathPhase {
	case 1:
		s.enemyDeathTimer += dt
		s.enemyAlpha = 1.0 - (s.enemyDeathTimer / 0.6)
		if s.enemyDeathTimer >= 0.6 {
			s.enemyAlpha = 0.0
			s.enemyDeathPhase = 2
			s.enemyDeathTimer = 0.0
			cx, cy := s.enemyCenter()
			_, _, w, h := s.enemyDrawRect()
			spreadX := w / 4
			spreadY := h / 8
			for i := 0; i < 50; i++ {
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
		s.enemyDeathTimer += dt
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
		if s.enemyDeathTimer >= 1.8 {
			s.enemyDeathPhase = 3
		}
	}
}

func (s *BattleScene) updateAllPoses(dt float64) {
	for i := 0; i < partySize; i++ {
		// ★追加：回避で右にずらしたスプライトを、時間経過で元の位置に戻す。
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
			if s.playerAnimTimer[i] >= 0.2 {
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

		// ← 変更：スキルメニュー中は発光pose、それ以外は通常ready
		if s.waitingActor == i &&
			(s.battlePhase == phasePlayerMenu ||
				s.battlePhase == phaseSkillMenu ||
				s.battlePhase == phaseTargetSelect ||
				s.battlePhase == phaseHealSelect ||
				s.battlePhase == phaseItemMenu ||
				s.battlePhase == phaseItemTarget) {

			if s.battlePhase == phaseSkillMenu {
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
