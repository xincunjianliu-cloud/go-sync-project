package main

// battle_calc.go: ゲージ・ダメージ/回復量・ステータス補正など戦闘計算まわり
import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

func (s *BattleScene) canUseSynergy() bool {
	return s.gaugePoint >= s.allAttackGaugeCost()
}

func (s *BattleScene) hasFullPartyForSynergy() bool {
	for i := 0; i < partySize; i++ {
		if s.game.PlayerHP[i] <= 0 {
			return false
		}
	}
	return true
}

func (s *BattleScene) allAttackGaugeCost() int {
	if s.gaugeStage == gaugeMaxStage-1 {
		return 15
	}
	return 10
}

func (s *BattleScene) canUseRewind(actor int) bool {
	return actor >= 0 && actor < partySize && !s.rewindUsed && s.gaugeStage >= gaugeMaxStage-1
}

func (s *BattleScene) gaugeFillRatio() float64 {
	ratio := float64(s.gaugePoint) / float64(gaugePoolMax)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return ratio
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

// ── 運（Luck）による会心・回避判定 ─────────────────────────
// 味方は運ステータスを持つが、敵は運を持たない
//（＝敵の攻撃は会心しない。味方の回避判定は味方自身の運のみで決まる）。

// rollIsCrit は luck から算出した会心率で判定する。
func (s *BattleScene) rollIsCrit(luck int) bool {
	chance := critChancePercent(luck)
	if chance <= 0 {
		return false
	}
	return rand.Intn(100) < chance
}

// rollIsEvade は luck から算出した回避率で判定する。
func (s *BattleScene) rollIsEvade(luck int) bool {
	chance := evadeChancePercent(luck)
	if chance <= 0 {
		return false
	}
	return rand.Intn(100) < chance
}

// ── 新スプライトシート対応：フレーム計算 ──────────────────────

// spriteFrame はループ系pose用のフレーム番号を返す。
func spriteFrame(pose int, timer float64) int {
	n, ok := poseFrameCount[pose]
	if !ok || n <= 0 {
		return 0
	}
	dur, ok2 := poseLoopFrameDur[pose]
	if !ok2 {
		dur = 0.15
	}
	t := timer
	if t < 0 {
		t = 0
	}
	return int(t/dur) % n
}

// frameFromProgress は攻撃演出など「進行度(0.0〜1.0)」で1回再生する系のフレームを返す。
func frameFromProgress(progress float64, pose int) int {
	n := poseFrameCount[pose]
	if n <= 0 {
		return 0
	}
	if progress < 0 {
		progress = 0
	}
	if progress >= 1 {
		progress = 0.999
	}
	return int(progress * float64(n))
}

// glowFrameForLevel はコマンド選択中の発光レベル(0-3)から実フレーム番号を返す。
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
	t := timer
	if t < 0 {
		t = 0
	}
	sub := int(t/0.15) % count
	return start + sub
}

func fitTextForWidth(face text.Face, txt string, maxWidth float64) string {
	runes := []rune(txt)
	for len(runes) > 0 {
		w, _ := text.Measure(string(runes), face, 0)
		if w <= maxWidth {
			return string(runes)
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}

var timelineImgLayout = struct {
	LineY           float64
	GoalMarginRight float64
	VertLineX       float64
}{
	LineY:           50.0,
	GoalMarginRight: 100.0,
	VertLineX:       53.0,
}

func (s *BattleScene) timelineDrawOrigin() (float64, float64) {
	if s.game.TimelineBarImg == nil {
		return 0, 0
	}
	imgW := float64(s.game.TimelineBarImg.Bounds().Dx())
	x := float64(gameWidth)/2 - imgW/2
	y := trackCenterY() - timelineImgLayout.LineY
	return x, y
}

func trackCenterY() float64 {
	return trackY + trackH/2
}

func (s *BattleScene) goalScreenX() float64 {
	imgX, _ := s.timelineDrawOrigin()
	if s.game.TimelineBarImg == nil {
		return trackX + trackW
	}
	barW := float64(s.game.TimelineBarImg.Bounds().Dx())
	return imgX + barW - timelineImgLayout.GoalMarginRight
}

// ── スキルダメージ・ステータス計算 ──────────────────────────

func (s *BattleScene) rollSkillDamage(actor int, skillIdx int, lv int, isAll bool) int {
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
	if power <= 0 {
		return 0
	}

	var atkStat, defStat float64
	if data.Element == ElemPhysicalNone {
		atkStat = float64(s.effectiveAtk(actor))
		defStat = float64(s.effectiveEnemyDef(false))
	} else {
		atkStat = float64(s.effectiveMagicAtk(actor))
		defStat = float64(s.effectiveEnemyDef(true))
	}
	if defStat < 1 {
		defStat = 1
	}
	luck := 0
	if actor >= 0 && actor < partySize {
		luck = s.game.PlayerLuck[actor]
	}
	return s.rollDamage(atkStat, float64(power), defStat, elementalDamageMultiplier(data.Element, s.enemyElementResist), luck)
}

func (s *BattleScene) rollDamage(atk float64, power float64, def float64, elementMultiplier float64, luck int) int {
	if def < 1 {
		def = 1
	}
	damage := atk * power / def
	damage *= float64(90+rand.Intn(21)) / 100.0
	damage *= elementMultiplier
	if s.rollIsCrit(luck) {
		damage *= critDamageMultiply
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
	base := s.game.PlayerAtk[actor]
	down := SumDebuffPercent(s.PlayerDebuffs[actor], DebuffAtkDown)
	return int(float64(base) * (1.0 - float64(down)/100.0))
}

func (s *BattleScene) effectiveMagicAtk(actor int) int {
	base := s.game.PlayerMagicAtk[actor]
	down := SumDebuffPercent(s.PlayerDebuffs[actor], DebuffAtkDown)
	return int(float64(base) * (1.0 - float64(down)/100.0))
}

func (s *BattleScene) effectiveEnemyDef(magic bool) int {
	var base int
	var t DebuffType
	if magic {
		base = s.enemyMagicDef
		t = DebuffMagicDefDown
	} else {
		base = s.enemyDef
		t = DebuffDefDown
	}
	down := SumDebuffPercent(s.EnemyDebuffs, t)
	return int(float64(base) * (1.0 - float64(down)/100.0))
}

// effectivePlayerDef は、敵からの攻撃に対する味方側の防御力を返す。
// isMagic が false の場合は物理防御力、true の場合は魔法防御力を使う
// （＝「物理攻撃に対しては物理防御が適応される」仕様）。
func (s *BattleScene) effectivePlayerDef(target int, magic bool) int {
	if target < 0 || target >= partySize {
		return 0
	}
	if magic {
		return s.game.PlayerMagicDef[target]
	}
	return s.game.PlayerDef[target]
}

func (s *BattleScene) applySkillEffects(effects []SkillEffect, casterIdx int, targetIsEnemy bool, targetIdx int) {
	for _, e := range effects {
		switch e.Type {
		case EffectAtbDownSmall:
			if targetIsEnemy {
				s.atbGauge[enemyID] -= 15
				if s.atbGauge[enemyID] < 0 {
					s.atbGauge[enemyID] = 0
				}
			} else {
				s.atbGauge[targetIdx] -= 15
				if s.atbGauge[targetIdx] < 0 {
					s.atbGauge[targetIdx] = 0
				}
			}
		case EffectAtbDownLarge:
			if targetIsEnemy {
				s.atbGauge[enemyID] -= 35
				if s.atbGauge[enemyID] < 0 {
					s.atbGauge[enemyID] = 0
				}
			} else {
				s.atbGauge[targetIdx] -= 35
				if s.atbGauge[targetIdx] < 0 {
					s.atbGauge[targetIdx] = 0
				}
			}
		case EffectDebuffDefBoth:
			d1 := Debuff{Type: DebuffDefDown, Percent: e.Percent, Turns: e.Turns}
			d2 := Debuff{Type: DebuffMagicDefDown, Percent: e.Percent, Turns: e.Turns}
			if targetIsEnemy {
				s.EnemyDebuffs = append(s.EnemyDebuffs, d1, d2)
			} else {
				s.PlayerDebuffs[targetIdx] = append(s.PlayerDebuffs[targetIdx], d1, d2)
			}
		default:
			d := Debuff{Type: mapEffectToDebuff(e.Type), Percent: e.Percent, Turns: e.Turns}
			if targetIsEnemy {
				s.EnemyDebuffs = append(s.EnemyDebuffs, d)
			} else {
				s.PlayerDebuffs[targetIdx] = append(s.PlayerDebuffs[targetIdx], d)
			}
		}
	}
}

func (s *BattleScene) tickDebuffs(actorIdx int, isEnemy bool) {
	if isEnemy {
		s.EnemyDebuffs = TickDebuffList(s.EnemyDebuffs)
	} else {
		s.PlayerDebuffs[actorIdx] = TickDebuffList(s.PlayerDebuffs[actorIdx])
	}
}

// anyActorHolding は、いずれかのキャラがまだ前進位置(readySlideX)から
// 元の位置へ戻りきっていない（＝行動アニメーション～戻り待機の途中）かを返す。
func (s *BattleScene) anyActorHolding() bool {
	for i := 0; i < partySize; i++ {
		if s.readySlideX[i] < -0.5 {
			return true
		}
	}
	return false
}
