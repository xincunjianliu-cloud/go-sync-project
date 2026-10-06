package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (s *BattleScene) addGaugePoint(pt int) {
	s.gaugePoint += pt
	if s.gaugePoint > gaugePoolMax {
		s.gaugePoint = gaugePoolMax
	}
	s.recomputeGaugeStage()
}

func (s *BattleScene) recomputeGaugeStage() {
	stage := 0
	for _, need := range gaugeCumThresholds {
		if s.gaugePoint >= need {
			stage++
		} else {
			break
		}
	}
	s.gaugeStage = stage
}

func (s *BattleScene) gaugeAtkBonus() int {
	return gaugeStageAtkBonus[s.gaugeStage]
}

func (s *BattleScene) hasFullPartyForSynergy() bool {
	for i := 0; i < partySize; i++ {
		if s.game.PlayerHP[i] <= 0 {
			return false
		}
	}
	return true
}

func (s *BattleScene) canUseRewind(actor int) bool {
	return actor >= 0 && actor < partySize && !s.rewindUsed && s.gaugePoint >= gaugePoolMax
}

func (s *BattleScene) consumeAllGaugePoints() {
	s.gaugePoint = 0
	s.gaugeStage = 0
}

func (s *BattleScene) effectiveMPCost(base int) int {
	if s.rewindActive {
		return (base + 1) / 2
	}
	return base
}

func (s *BattleScene) rollIsCrit(luck int) bool {
	chance := critChancePercent(luck)
	if chance <= 0 {
		return false
	}
	return rand.Intn(100) < chance
}

func (s *BattleScene) rollIsEvade(luck int) bool {
	chance := evadeChancePercent(luck)
	if chance <= 0 {
		return false
	}
	return rand.Intn(100) < chance
}

// loopFrame はn コマのくり返しアニメで、経過timer秒の時に表示するコマ。
func loopFrame(n int, timer float64) int {
	if n <= 0 {
		return 0
	}
	t := max(timer, 0)
	return int(t/battleSpriteFrameDur) % n
}

// actionFrame はnコマの1回きりのアニメで、経過timer秒の時に表示するコマ。
// 最後まで行ったら最後のコマで止まる。
func actionFrame(n int, timer float64) int {
	if n <= 0 {
		return 0
	}
	t := max(timer, 0)
	return min(int(t/battleActionFrameDur), n-1)
}

// actionAnimDuration はnコマの1回きりのアニメの長さ（秒）。
func actionAnimDuration(n int) float64 {
	return float64(n) * battleActionFrameDur
}

func glowFrameForLevel(level int, timer float64) int {
	if level < 0 {
		level = 0
	}
	if level > 3 {
		level = 3
	}
	start, count := glowLevelRange[level][0], glowLevelRange[level][1]
	if count <= 1 {
		return start
	}
	return start + loopFrame(count, timer)
}

var goalAnchorLayout = struct {
	X float64
	Y float64
}{
	X: 860.0,
	Y: 57.0,
}

var goalRelativeImgLayout = struct {
	TimelineHorizY float64
	TimelineVertX  float64
}{
	TimelineHorizY: 0.0,
	TimelineVertX:  -3.0,
}

var attackIconLayout = struct {
	XOffsetFromGoal float64
	YFromBottom     float64
}{
	XOffsetFromGoal: 0.0,
	YFromBottom:     148.0,
}

func (s *BattleScene) timelineDrawOrigin() (float64, float64) {
	imgW := float64(s.game.TimelineBarImg.Bounds().Dx())
	imgH := float64(s.game.TimelineBarImg.Bounds().Dy())
	x := float64(gameWidth)/2 - imgW/2
	cy := goalAnchorLayout.Y + goalRelativeImgLayout.TimelineHorizY
	return x, cy - imgH/2
}

func trackCenterY() float64 {
	return goalAnchorLayout.Y
}

func (s *BattleScene) goalScreenX() float64 {
	return goalAnchorLayout.X
}

func (s *BattleScene) timelineVertDrawOrigin() (float64, float64) {
	imgW := float64(s.game.TimelineBarVertImg.Bounds().Dx())
	imgH := float64(s.game.TimelineBarVertImg.Bounds().Dy())
	cx := goalAnchorLayout.X + goalRelativeImgLayout.TimelineVertX
	y := float64(gameHeight)/2 - imgH/2
	return cx - imgW/2, y
}

func attackIconCenter() (float64, float64) {
	x := goalAnchorLayout.X + attackIconLayout.XOffsetFromGoal
	y := float64(gameHeight) - attackIconLayout.YFromBottom
	return x, y
}

func (s *BattleScene) rollSkillDamage(actor int, skillIdx int, lv int, isAll bool, targetSlot int) int {
	if lv < 1 {
		lv = 1
	}
	skills := s.game.CharacterSkills(actor)
	if skillIdx < 0 || skillIdx >= len(skills) || lv > len(skills[skillIdx].Levels) {
		return 0
	}
	data := skills[skillIdx].Levels[lv-1]

	power := data.PowerSingle
	if isAll {
		power = data.PowerAll
	}
	if data.TGPowerLevel > 0 && s.tgLevel() >= data.TGPowerLevel {
		if isAll && data.TGPowerAll > 0 {
			power = data.TGPowerAll
		} else if !isAll && data.TGPowerSingle > 0 {
			power = data.TGPowerSingle
		}
	}
	if power <= 0 {
		return 0
	}

	var atkStat, defStat float64
	if data.usesPhysical() {
		atkStat = float64(s.effectiveAtk(actor))
		defStat = float64(s.effectiveEnemyDef(targetSlot, false))
	} else {
		atkStat = float64(s.effectiveMagicAtk(actor))
		defStat = float64(s.effectiveEnemyDef(targetSlot, true))
	}
	if defStat < 1 {
		defStat = 1
	}
	luck := 0
	if actor >= 0 && actor < partySize {
		luck = s.game.PlayerLuck[actor]
	}
	resist := [elementalTypeCount]int{}
	if targetSlot >= 0 && targetSlot < len(s.enemies) {
		resist = s.enemies[targetSlot].ElementResist
		s.forceCrit = data.CritVsDefDown && SumDebuffPercent(s.enemies[targetSlot].Debuffs, StatDef) > 0
	}
	return s.rollDamage(atkStat, float64(power), defStat, elementalDamageMultiplier(data.Element, resist), luck)
}

func (s *BattleScene) rollDamage(atk float64, power float64, def float64, elementMultiplier float64, luck int) int {
	if def < 1 {
		def = 1
	}
	damage := atk * power / def
	damage *= float64(90+rand.Intn(21)) / 100.0
	damage *= elementMultiplier
	s.lastRollWasCrit = false
	forceCrit := s.forceCrit
	s.forceCrit = false
	if forceCrit || s.rollIsCrit(luck) {
		damage *= critDamageMultiply
		s.lastRollWasCrit = true
	}
	result := int(math.Round(damage))
	if result < 1 {
		result = 1
	}
	return result
}

func elementalDamageMultiplier(element Element, resistances [elementalTypeCount]int) float64 {
	index := int(element) - int(ElemFire)
	if index < 0 || index >= elementalTypeCount {
		return 1.0
	}

	multiplier := 1.0 - float64(resistances[index])/100.0
	if multiplier < 0 {
		return 0
	}
	return multiplier
}

func (s *BattleScene) rollEnemySkillDamage(power int, element Element, target int) int {
	if power <= 0 {
		return 0
	}
	magic := element != ElemPhysicalNone
	atkStat := float64(s.effectiveEnemyAtk(s.actingEnemySlot, magic))
	defStat := float64(s.effectivePlayerDef(target, magic))
	if defStat < 1 {
		defStat = 1
	}
	return s.rollDamage(atkStat, float64(power), defStat, 1.0, 0)
}

func (s *BattleScene) rollSkillHeal(actor int, skillIdx int, lv int, isAll bool) int {
	if lv < 1 {
		lv = 1
	}
	skills := s.game.CharacterSkills(actor)
	if skillIdx < 0 || skillIdx >= len(skills) || lv > len(skills[skillIdx].Levels) {
		return 0
	}
	data := skills[skillIdx].Levels[lv-1]
	power := data.PowerSingle
	if isAll {
		power = data.PowerAll
	}
	magicAtk := float64(s.effectiveMagicAtk(actor))
	heal := int(magicAtk * float64(power) / 100.0 * 10)
	if heal < 1 {
		heal = 1
	}
	return heal
}

func (s *BattleScene) effectiveAtk(actor int) int {
	return s.applyPlayerStatMods(actor, StatAtk, s.game.PlayerAtk[actor])
}

func (s *BattleScene) effectiveMagicAtk(actor int) int {
	return s.applyPlayerStatMods(actor, StatMat, s.game.PlayerMagicAtk[actor])
}

// applyPlayerStatMods applies party member actor's buffs and debuffs on
// stat to base.
func (s *BattleScene) applyPlayerStatMods(actor int, stat StatKind, base int) int {
	down := SumDebuffPercent(s.PlayerDebuffs[actor], stat)
	up := SumBuffPercent(s.PlayerBuffs[actor], stat)
	return int(float64(base) * (1.0 + float64(up)/100.0 - float64(down)/100.0))
}

// tgLevel is the synergy gauge level (1-5) as shown to the player.
func (s *BattleScene) tgLevel() int {
	return s.gaugeStage + 1
}

// reduceAtb は対象のATBゲージをamount減らす(0未満にはしない)。
func (s *BattleScene) reduceAtb(targetIsEnemy bool, targetIdx int, amount float64) {
	actor := targetIdx
	if targetIsEnemy {
		if targetIdx < 0 || targetIdx >= len(s.enemies) {
			return
		}
		actor = s.enemyActorIndex(targetIdx)
	}
	s.placeOnTimeline(actor, s.atbGauge[actor]-amount)
}

func (s *BattleScene) effectiveEnemyDef(slot int, magic bool) int {
	if slot < 0 || slot >= len(s.enemies) {
		return 0
	}
	e := &s.enemies[slot]
	var base int
	var t StatKind
	if magic {
		base = e.MagicDef
		t = StatMdf
	} else {
		base = e.Def
		t = StatDef
	}
	down := SumDebuffPercent(e.Debuffs, t)
	return int(float64(base) * (1.0 - float64(down)/100.0))
}

func (s *BattleScene) effectiveEnemyAtk(slot int, magic bool) int {
	if slot < 0 || slot >= len(s.enemies) {
		return 0
	}
	e := &s.enemies[slot]
	base, t := e.PhysAtk, StatAtk
	if magic {
		base, t = e.MagicAtk, StatMat
	}
	down := SumDebuffPercent(e.Debuffs, t)
	return int(float64(base) * (1.0 - float64(down)/100.0))
}

func (s *BattleScene) effectivePlayerDef(target int, magic bool) int {
	if target < 0 || target >= partySize {
		return 0
	}
	if magic {
		return s.applyPlayerStatMods(target, StatMdf, s.game.PlayerMagicDef[target])
	}
	return s.applyPlayerStatMods(target, StatDef, s.game.PlayerDef[target])
}

func (s *BattleScene) applySkillEffects(effects []SkillEffect, casterIdx int, targetIsEnemy bool, targetIdx int, isAll bool) {
	for _, e := range effects {
		switch e.Type {
		case EffectAtbDownSmall:
			s.reduceAtb(targetIsEnemy, targetIdx, 15)
			continue
		case EffectAtbDownLarge:
			s.reduceAtb(targetIsEnemy, targetIdx, 35)
			continue
		}
		stats, isBuff, ok := effectStats(e.Type)
		if !ok {
			continue
		}
		percent := e.Percent
		if isAll && e.PercentAll > 0 {
			percent = e.PercentAll
		}
		if targetIsEnemy {
			if targetIdx < 0 || targetIdx >= len(s.enemies) || isBuff {
				continue
			}
			for _, st := range stats {
				s.enemies[targetIdx].Debuffs = append(s.enemies[targetIdx].Debuffs,
					Debuff{Type: st, Percent: percent, Seconds: e.Seconds})
			}
			continue
		}
		if targetIdx < 0 || targetIdx >= partySize {
			continue
		}
		for _, st := range stats {
			if isBuff {
				s.PlayerBuffs[targetIdx] = append(s.PlayerBuffs[targetIdx],
					Buff{Type: st, Percent: percent, Seconds: e.Seconds})
			} else {
				s.PlayerDebuffs[targetIdx] = append(s.PlayerDebuffs[targetIdx],
					Debuff{Type: st, Percent: percent, Seconds: e.Seconds})
			}
		}
	}
}

// tickTimedStatus counts down everything that lasts a number of seconds
// (timed buffs/debuffs, counter stance). It runs only while the ATB
// timeline itself is moving.
func (s *BattleScene) tickTimedStatus(dt float64) {
	for i := 0; i < partySize; i++ {
		s.PlayerBuffs[i] = TickBuffSeconds(s.PlayerBuffs[i], dt)
		s.PlayerDebuffs[i] = TickDebuffSeconds(s.PlayerDebuffs[i], dt)
		if s.counterTimer[i] > 0 {
			s.counterTimer[i] = max(s.counterTimer[i]-dt, 0)
		}
	}
	for i := range s.enemies {
		s.enemies[i].Debuffs = TickDebuffSeconds(s.enemies[i].Debuffs, dt)
	}
}

func (s *BattleScene) anyActorHolding() bool {
	for i := 0; i < partySize; i++ {
		if s.readySlideX[i] < -0.5 {
			return true
		}
	}
	return false
}

func (s *BattleScene) gaugeSegmentColor(stage int) color.RGBA {
	if stage < 0 {
		stage = 0
	}
	if stage > gaugeMaxStage-1 {
		stage = gaugeMaxStage - 1
	}
	if s.gaugeStage >= gaugeMaxStage-1 {
		return s.gaugeRainbowColor()
	}
	return gaugeStageColors[stage]
}

func (s *BattleScene) gaugeRainbowColor() color.RGBA {
	n := len(gaugeStageColors)
	const secPerColor = 0.6
	t := s.gaugeColorAnimTimer / secPerColor
	idx := int(t) % n
	next := (idx + 1) % n
	frac := t - float64(int(t))
	c1, c2 := gaugeStageColors[idx], gaugeStageColors[next]
	return color.RGBA{
		R: lerpByte(c1.R, c2.R, frac),
		G: lerpByte(c1.G, c2.G, frac),
		B: lerpByte(c1.B, c2.B, frac),
		A: 255,
	}
}

func lerpByte(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

func (s *BattleScene) fillGaugeTriSegment(screen *ebiten.Image, ix, iy, iw, ih int, hStart, hEnd float64, col color.RGBA) {
	if hEnd <= hStart {
		return
	}
	yBottom := float64(iy+ih) - hStart
	yTop := float64(iy+ih) - hEnd
	rightAtBottom := float64(ix) + float64(iw)*hStart/float64(ih)
	rightAtTop := float64(ix) + float64(iw)*hEnd/float64(ih) + 1

	var path vector.Path
	path.MoveTo(float32(ix), float32(yBottom))
	path.LineTo(float32(ix), float32(yTop))
	path.LineTo(float32(rightAtTop), float32(yTop))
	path.LineTo(float32(rightAtBottom), float32(yBottom))
	path.Close()

	fillOpts := &vector.DrawPathOptions{AntiAlias: false}
	fillOpts.ColorScale.ScaleWithColor(col)
	vector.FillPath(screen, &path, nil, fillOpts)
}
