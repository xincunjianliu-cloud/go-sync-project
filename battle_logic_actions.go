package main

// battle_logic_actions.go: 巻き戻し実行・ダメージ確定・敵行動・勝敗判定・対象選択の更新処理
import (
	"fmt"
	"math/rand"
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
	s.finishPlayerTurn(true)
}

func (s *BattleScene) rollNormalDamage() int {
	p := s.waitingActor
	if p < 0 || p >= partySize {
		return 5
	}
	atk := s.effectiveAtk(p)
	def := s.effectiveEnemyDef(false)
	if def < 1 {
		def = 1
	}
	const normalAttackPower = 100.0
	raw := int(float64(atk) * normalAttackPower / 100.0 / float64(def) * 10)
	if raw < 1 {
		raw = 1
	}
	bonus := s.gaugeAtkBonus()
	return int(float64(raw) * (1.0 + float64(bonus)/100.0))
}

func (s *BattleScene) finishPlayerTurn(resetGauge bool) {
	if s.waitingActor >= 0 && s.waitingActor < partySize {
		s.tickDebuffs(s.waitingActor, false)
	}
	if resetGauge && s.waitingActor >= 0 && s.waitingActor < partySize {
		s.atbGauge[s.waitingActor] = 0
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

func (s *BattleScene) tryWaitSynergy() {
	for i := 0; i < partySize; i++ {
		if !s.waitStance[i] {
			return
		}
	}

	if !s.canUseSynergy() {
		s.battleLog = "ゲージが足りない！"
		s.battleLogTimer = battleLogDuration
		for i := 0; i < partySize; i++ {
			s.waitStance[i] = false
		}
		s.waitOrder = []int{}
		return
	}

	s.gaugePoint -= s.allAttackGaugeCost()
	if s.gaugePoint < 0 {
		s.gaugePoint = 0
	}
	s.recomputeGaugeStage()

	dmg := rand.Intn(50) + 70
	s.enemyHP -= dmg
	if s.enemyHP < 0 {
		s.enemyHP = 0
	}

	s.flashAlpha = 0.8
	s.shakeType = 3
	s.shakeTimer = 0.6
	s.shakeMaxDur = 0.8
	s.shakePower = 16.0
	s.damagePops = append(s.damagePops, DamagePop{
		Value: dmg,
		X:     s.enemyX - 8.0,
		Y:     100.0,
		Vy:    -180.0,
		Timer: 0.0,
	})
	for i := 0; i < partySize; i++ {
		s.waitStance[i] = false
		s.atbGauge[i] = 0
	}
	s.waitOrder = []int{}
	if s.checkBattleEnd() {
		return
	}
	s.battlePhase = phaseATB
	s.waitingActor = -1
	s.tryStartNextActor()
}

func (s *BattleScene) executeEnemyAction() {
	var aliveList []int
	for i := 0; i < partySize; i++ {
		if s.game.PlayerHP[i] > 0 {
			aliveList = append(aliveList, i)
		}
	}
	if len(aliveList) == 0 {
		return
	}
	target := aliveList[rand.Intn(len(aliveList))]
	dmg := rand.Intn(s.enemyDmgRange) + s.enemyDmgMin
	prevHP := s.game.PlayerHP[target]
	s.lastEnemyAttackTarget = target
	s.lastEnemyAttackPrevHP = prevHP
	s.lastEnemyAttackDamage = dmg

	s.game.PlayerHP[target] -= dmg

	targetX := s.partyScreenX[target]
	targetY := s.partyScreenY[target] - 30.0

	s.damagePops = append(s.damagePops, DamagePop{
		Value: dmg,
		X:     targetX,
		Y:     targetY,
		Vy:    -180.0,
		Timer: 0.0,
	})

	if s.game.PlayerHP[target] > 0 {
		s.playerPose[target] = poseDamage
		s.playerAnimTimer[target] = 0.0

		s.shakeType = 1
		s.shakeTimer = 0.35
		s.shakeMaxDur = 0.35
		s.shakePower = 12.0
	} else {
		s.game.PlayerHP[target] = 0

		if s.countWaitStance() > 0 {
			for i := 0; i < partySize; i++ {
				s.waitStance[i] = false
			}
			s.waitOrder = []int{}
			s.battleLog = "味方が倒れたため連携待機がリセットされた"
			s.battleLogTimer = battleLogDuration
		}
	}
	s.enemyActionWaitTimer = 1.5
}

func (s *BattleScene) checkBattleEnd() bool {
	if s.enemyHP <= 0 {
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

			for i := 0; i < partySize; i++ {
				if s.game.PlayerLv[i] < maxPlayerLevel {
					s.game.PlayerEXP[i] += s.enemyExp
				}
				for s.game.PlayerLv[i] < maxPlayerLevel && s.game.PlayerEXP[i] >= s.game.PlayerNextEXP[i] {
					s.game.PlayerEXP[i] -= s.game.PlayerNextEXP[i]
					s.game.PlayerLv[i]++
					if s.game.PlayerLv[i] >= maxPlayerLevel {
						s.game.PlayerLv[i] = maxPlayerLevel
						s.game.PlayerEXP[i] = 0
						s.game.PlayerNextEXP[i] = 0
					} else {
						s.game.PlayerNextEXP[i] = s.game.PlayerLv[i] * 50
					}
					s.game.PlayerMaxHP[i] += 10
					s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
					s.game.PlayerMaxMP[i] += 5
					s.game.PlayerMP[i] = s.game.PlayerMaxMP[i]
					s.game.PlayerAtk[i] += 3
				}
			}

			for i := 0; i < partySize; i++ {
				s.game.PlayerSP[i] += s.enemySP
			}

			s.enemyDeathPhase = 1
			s.enemyDeathTimer = 0.0
			s.enemyAlpha = 1.0
			s.isWon = true
			s.game.Audio.PlayBGMWithIntro(bgmBattleEndIntro, bgmBattleEndLoop)
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
		return true
	}
	return false
}

func (s *BattleScene) exitBattleToField() {
	field, _ := NewRoomScene(s.game, s.originMap, s.originX, s.originY, "", s.originDir)
	if strings.HasPrefix(s.enemyType, "boss_") {
		numStr := strings.TrimPrefix(s.enemyType, "boss_")
		if bossNum, err := strconv.Atoi(numStr); err == nil {
			if bossNum >= 1 && bossNum <= 4 {
				field.justDefeatedBoss = bossNum
			}
		}
	}
	s.game.ChangeSceneWithFade(field, 0.5)
}

// updateTargetSelect：通常攻撃・強撃・全体攻撃・炎魔法・デバフの実行分岐。
func (s *BattleScene) updateTargetSelect() {
	if isEscapePressed() {
		if s.pendingSkill >= 1 {
			s.battlePhase = phaseSkillMenu
		} else {
			s.battlePhase = phasePlayerMenu
		}
		return
	}

	p := s.waitingActor
	if p >= 0 && p < partySize && s.pendingSkill >= 1 {
		skillIdx := s.pendingSkill - 1
		skills := s.game.CharacterSkills(p)
		if skillIdx >= 0 && skillIdx < len(skills) {
			data := s.game.CurrentSkillLevelData(p, skillIdx)
			if data.Target == TargetBoth {
				if isMenuRightPressed() {
					s.selectedSkillTarget = TargetAll
				}
				if isMenuLeftPressed() {
					s.selectedSkillTarget = TargetSingle
				}
			}
		}
	}

	if !isConfirmKeyPressed() {
		return
	}

	if p < 0 || p >= partySize {
		fmt.Println("waitingActor が不正")
		return
	}

	if s.pendingSkill >= 1 {
		// ── 新スキル体系（全キャラ共通） ──
		skillIdx := s.pendingSkill - 1
		lv := s.lastSkillLevel[p][skillIdx]
		if lv < 1 {
			lv = 1
		}
		skills := s.game.CharacterSkills(p)
		data := skills[skillIdx].Levels[lv-1]
		isAll := data.Target == TargetAll || (data.Target == TargetBoth && s.selectedSkillTarget == TargetAll)

		// ← 追加：スキル属性でアニメ種別を決定
		if data.Element == ElemPhysicalNone {
			s.attackAnimType = animCharge
		} else {
			s.attackAnimType = animFireMagic
		}

		dmg := s.rollSkillDamage(p, skillIdx, lv, isAll)

		if s.rewindActive {
			second := s.rollSkillDamage(p, skillIdx, lv, isAll)
			s.pendingDamage = dmg
			s.pendingDamage2 = second
			s.pendingDamage2Scheduled = true
		} else {
			s.pendingDamage = dmg
			s.pendingDamage2 = 0
		}

		s.game.PlayerMP[p] -= s.effectiveMPCost(data.MPCost)
		s.battleLog = skills[skillIdx].Name
		s.battleLogTimer = battleLogDuration
		if dmg >= 40 {
			s.pendingDamageShake = 18.0
		} else {
			s.pendingDamageShake = 10.0
		}

		s.applySkillEffects(data.Effects, p, true, enemyID)
		if s.rewindActive {
			s.applySkillEffects(data.Effects, p, true, enemyID)
		}
		s.addGaugePoint(1)

		imgW, imgH := 0, 0
		if s.enemyImage != nil {
			imgW = s.enemyImage.Bounds().Dx()
			imgH = s.enemyImage.Bounds().Dy()
			if imgW <= 32 && len(s.game.BossImgs) > 0 && s.game.BossImgs[0] != nil {
				imgW = s.game.BossImgs[0].Bounds().Dx()
				imgH = s.game.BossImgs[0].Bounds().Dy()
			}
		}
		_ = imgW
		yPos := 240.0 - float64(imgH)/2
		if yPos < 12 {
			yPos = 12
		}
		s.pendingDamageX = 280.0 - 10.0
		s.pendingDamageY = yPos - 15.0

		s.activeAttacker = p
		s.attackPhaseTimer = 0.0
		s.hitStopTimer = 0.0
		s.battlePhase = phaseATB
		return
	}

	// ── 通常攻撃の処理 ──
	fmt.Printf("updateTargetSelect: p=%d pendingSkill=%d\n", p, s.pendingSkill)

	s.attackAnimType = animNormal // ← 追加
	firstDmg := s.rollNormalDamage()
	if s.rewindActive {
		secondDmg := s.rollNormalDamage()
		s.pendingDamage = firstDmg
		s.pendingDamage2 = secondDmg
		s.pendingDamage2Scheduled = true
	} else {
		s.pendingDamage = firstDmg
		s.pendingDamage2 = 0
	}
	s.battleLog = "通常攻撃"
	s.battleLogTimer = battleLogDuration
	s.pendingDamageShake = 10.0
	fmt.Printf("通常攻撃: dmg=%d\n", s.pendingDamage)

	s.addGaugePoint(1)

	imgW := 0
	imgH := 0
	if s.enemyImage != nil {
		imgW = s.enemyImage.Bounds().Dx()
		imgH = s.enemyImage.Bounds().Dy()
		if imgW <= 32 && len(s.game.BossImgs) > 0 && s.game.BossImgs[0] != nil {
			imgW = s.game.BossImgs[0].Bounds().Dx()
			imgH = s.game.BossImgs[0].Bounds().Dy()
		}
	}
	_ = imgW

	yPos := 240.0 - float64(imgH)/2
	if yPos < 12 {
		yPos = 12
	}

	s.pendingDamageX = 280.0 - 10.0
	s.pendingDamageY = yPos - 15.0

	s.activeAttacker = p
	s.attackPhaseTimer = 0.0
	s.hitStopTimer = 0.0
	s.battlePhase = phaseATB
}

func (s *BattleScene) updateHealTargetSelect() {
	p := s.waitingActor

	if isMenuUpPressed() {
		if s.healTargetIndex < partySize {
			s.healTargetIndex = (s.healTargetIndex - 1 + partySize) % partySize
		}
	}
	if isMenuDownPressed() {
		if s.healTargetIndex < partySize {
			s.healTargetIndex = (s.healTargetIndex + 1) % partySize
		}
	}
	if isMenuRightPressed() {
		s.healTargetIndex = partySize
	}
	if isMenuLeftPressed() {
		if s.healTargetIndex == partySize {
			s.healTargetIndex = 0
		}
	}
	if isEscapePressed() {
		s.battlePhase = phaseSkillMenu
		return
	}
	if !isConfirmKeyPressed() {
		return
	}

	skillIdx := s.pendingSkill - 1
	if skillIdx < 0 {
		s.battlePhase = phaseSkillMenu
		return
	}
	lv := s.lastSkillLevel[p][skillIdx]
	if lv < 1 {
		lv = 1
	}
	skills := s.game.CharacterSkills(p)
	data := skills[skillIdx].Levels[lv-1]
	cost := s.effectiveMPCost(data.MPCost)

	if s.healTargetIndex == partySize {
		if s.game.PlayerMP[p] < cost {
			return
		}
		s.startCast(p)
		s.game.PlayerMP[p] -= cost
		healAmount := s.rollSkillHeal(p, skillIdx, lv, true)
		for i := 0; i < partySize; i++ {
			if s.game.PlayerHP[i] <= 0 {
				continue
			}
			s.game.PlayerHP[i] += healAmount
			if s.game.PlayerHP[i] > s.game.PlayerMaxHP[i] {
				s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
			}
			s.damagePops = append(s.damagePops, DamagePop{
				Value:  healAmount,
				X:      s.partyScreenX[i],
				Y:      s.partyScreenY[i] - 30.0,
				Vy:     -80.0,
				Timer:  0.0,
				IsHeal: true,
			})
			if s.rewindActive {
				healAmount2 := s.rollSkillHeal(p, skillIdx, lv, true)
				s.game.PlayerHP[i] += healAmount2
				if s.game.PlayerHP[i] > s.game.PlayerMaxHP[i] {
					s.game.PlayerHP[i] = s.game.PlayerMaxHP[i]
				}
				s.damagePops = append(s.damagePops, DamagePop{
					Value:  healAmount2,
					X:      s.partyScreenX[i] - 12.0,
					Y:      s.partyScreenY[i] - 40.0,
					Vy:     -80.0,
					Timer:  -0.18,
					IsHeal: true,
				})
			}
		}
		// 回復アニメーション中は即座にendCastを呼ばず、アニメーション完了後に戻す
		s.healingAnimTimer[p] = 1.5 // アニメーション + 遅延時間
		s.battleLog = skills[skillIdx].Name + "（全体）"
		s.battleLogTimer = battleLogDuration
		s.addGaugePoint(1)
	} else {
		target := s.healTargetIndex
		if s.game.PlayerHP[target] <= 0 {
			return
		}
		if s.game.PlayerMP[p] < cost {
			return
		}
		s.startCast(p)
		s.game.PlayerMP[p] -= cost
		healAmount := s.rollSkillHeal(p, skillIdx, lv, false)
		s.game.PlayerHP[target] += healAmount
		if s.game.PlayerHP[target] > s.game.PlayerMaxHP[target] {
			s.game.PlayerHP[target] = s.game.PlayerMaxHP[target]
		}
		s.damagePops = append(s.damagePops, DamagePop{
			Value:  healAmount,
			X:      s.partyScreenX[target],
			Y:      s.partyScreenY[target] - 30.0,
			Vy:     -80.0,
			Timer:  0.0,
			IsHeal: true,
		})
		if s.rewindActive {
			healAmount2 := s.rollSkillHeal(p, skillIdx, lv, false)
			s.game.PlayerHP[target] += healAmount2
			if s.game.PlayerHP[target] > s.game.PlayerMaxHP[target] {
				s.game.PlayerHP[target] = s.game.PlayerMaxHP[target]
			}
			s.damagePops = append(s.damagePops, DamagePop{
				Value:  healAmount2,
				X:      s.partyScreenX[target] - 12.0,
				Y:      s.partyScreenY[target] - 40.0,
				Vy:     -80.0,
				Timer:  -0.18,
				IsHeal: true,
			})
		}
		// 回復アニメーション中は即座にendCastを呼ばず、アニメーション完了後に戻す
		s.healingAnimTimer[p] = 1.5 // アニメーション + 遅延時間
		s.battleLog = skills[skillIdx].Name
		s.battleLogTimer = battleLogDuration
		s.addGaugePoint(1)
	}

	s.finishPlayerTurn(true)
}
