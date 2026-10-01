package main

import "fmt"

type SkillTarget int

const (
	TargetSingle SkillTarget = iota
	TargetAll
	TargetBoth
	// TargetSelf skills act on the user only and skip target selection.
	TargetSelf
)

type Element int

const (
	ElemPhysicalNone Element = iota
	ElemMagicNone
	ElemFire
	ElemLightning
	ElemIce
	ElemWind
)

const elementalTypeCount = 4

type EffectType int

const (
	EffectAtbDownSmall EffectType = iota
	EffectAtbDownLarge
	EffectDebuffPhysicalDef
	EffectDebuffMagicDef
	// EffectDebuffAtk lowers both physical and magic attack.
	EffectDebuffAtk
	EffectDebuffDefBoth
	// EffectBuffAtkUp raises both physical and magic attack.
	EffectBuffAtkUp
	EffectDebuffPhysAtk
	EffectDebuffMagicAtk
	EffectBuffPhysAtkUp
	EffectBuffPhysDefUp
	EffectBuffDefBothUp
)

// SkillEffect is a buff/debuff (or ATB push) applied by a skill. Buffs and
// debuffs last Seconds of running battle time.
type SkillEffect struct {
	Type       EffectType
	Percent    int
	PercentAll int
	Seconds    float64
}

type SkillLevelData struct {
	Description string
	Target      SkillTarget
	Element     Element
	// Physical makes an elemental skill (Element other than
	// ElemPhysicalNone/ElemMagicNone) hit with physical attack against
	// physical defense instead of magic.
	Physical    bool
	PowerSingle int
	PowerAll    int
	MPCost      int
	Effects     []SkillEffect
	IsHeal      bool
	// Ally makes a non-heal skill (buffs, guards) target party members.
	Ally bool
	// ReturnPosition is where the actor's timeline icon reappears (0-100)
	// after this skill is used, independent of the Spd stat.
	ReturnPosition float64
	// GaugePoint is how many synergy gauge points using this skill adds.
	GaugePoint int

	// UpgradeSP is the SP needed to upgrade the skill into this level
	// (unused on Lv.1). StatBonus is permanently added to the user's stats
	// once the skill reaches this level.
	UpgradeSP int
	StatBonus PlayerStats

	// Knockback pushes each hit enemy's timeline icon back by this much
	// (0-100 units), plus KnockbackPerTG for every synergy gauge level
	// above 1.
	Knockback      float64
	KnockbackPerTG float64

	// While the synergy gauge level is at least TGPowerLevel, the
	// single/all power becomes TGPowerSingle/TGPowerAll (0 = unchanged).
	TGPowerLevel  int
	TGPowerSingle int
	TGPowerAll    int

	// While the synergy gauge level is at least TGMPHealLevel, a heal also
	// restores MPHealSingle/MPHealAll MP to each target.
	TGMPHealLevel int
	MPHealSingle  int
	MPHealAll     int

	// Scan reveals the targets' HP and elemental weaknesses. NonLethal
	// damage never takes an enemy below 1 HP.
	Scan      bool
	NonLethal bool

	// ReviveHPPercent > 0 makes the skill target fallen party members and
	// bring them back with that percentage of their max HP.
	ReviveHPPercent int

	// Drain restores dealt damage / DrainDivisor MP to the user (at least
	// 1, at most DrainMax).
	DrainDivisor int
	DrainMax     int

	// CritVsDefDown makes hits always crit on targets with lowered
	// physical defense.
	CritVsDefDown bool

	// CoverCount > 0 makes the user take the next CoverCount single-target
	// enemy attacks aimed at an ally. The skill's Effects become buffs on
	// the user that last until the cover runs out.
	CoverCount int

	// CounterSeconds > 0 puts the user in counter stance for that long:
	// every enemy hit on them is answered with a normal attack dealing
	// CounterBonus% extra damage. Re-using it refreshes rather than stacks.
	CounterSeconds float64
	CounterBonus   int

	// GuardHPPercent > 0 grants the target GuardCount charges; a hit that
	// would leave them at or below that % of max HP is nullified instead.
	// Re-using it overwrites rather than stacks.
	GuardHPPercent int
	GuardCount     int
}

// usesPhysical reports whether the skill's damage uses physical attack and
// the target's physical defense.
func (d SkillLevelData) usesPhysical() bool {
	return d.Element == ElemPhysicalNone || d.Physical
}

// targetsAlly reports whether the skill is aimed at party members rather
// than enemies (self-target skills included).
func (d SkillLevelData) targetsAlly() bool {
	return d.IsHeal || d.Ally || d.ReviveHPPercent > 0 || d.Target == TargetSelf
}

type SkillDef struct {
	Name   string
	Levels []SkillLevelData
	// UnlockLevel is the character level required to use this skill.
	UnlockLevel int
}

// Each character's skill slice must be kept sorted by ascending UnlockLevel.
// The skill list and battle skill menu draw rows by array index and skip
// locked skills, relying on locked skills always being a trailing suffix
// (never a gap in the middle) so the visible rows stay packed from the top.

// characterSkillSets is indexed by party slot: 0 = player_1, 1 = player_2
// (the white-haired girl), 2 = player_3, 3 = player_4 (the purple-haired
// girl).
var characterSkillSets = [partySize][]SkillDef{
	Player1Skills,
	Player2Skills,
	Player3Skills,
	Player4Skills,
}

func (g *Game) CharacterSkills(charIdx int) []SkillDef {
	if charIdx < 0 || charIdx >= len(characterSkillSets) {
		return nil
	}
	return characterSkillSets[charIdx]
}

func (g *Game) IsSkillUnlocked(charIdx, skillIdx int) bool {
	skills := g.CharacterSkills(charIdx)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return false
	}
	if charIdx < 0 || charIdx >= len(g.PlayerLv) {
		return false
	}
	return g.PlayerLv[charIdx] >= skills[skillIdx].UnlockLevel
}

// SkillUpgradeCost is the SP needed to upgrade sk from currentLv to
// currentLv+1, or -1 if it can't go higher.
func SkillUpgradeCost(sk SkillDef, currentLv int) int {
	if currentLv < 1 {
		currentLv = 1
	}
	if currentLv >= MaxSkillLevel || currentLv >= len(sk.Levels) {
		return -1
	}
	return sk.Levels[currentLv].UpgradeSP
}

const MaxSkillLevel = 3

func (g *Game) CanUpgradeSkill(charIdx, skillIdx int) bool {
	if charIdx < 0 || charIdx >= partySize {
		return false
	}
	skills := g.CharacterSkills(charIdx)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return false
	}
	cost := SkillUpgradeCost(skills[skillIdx], g.PlayerSkillLv[charIdx][skillIdx])
	if cost < 0 {
		return false
	}
	return g.PlayerSP[charIdx] >= cost
}

func (g *Game) UpgradeSkill(charIdx, skillIdx int) bool {
	if !g.CanUpgradeSkill(charIdx, skillIdx) {
		return false
	}
	sk := g.CharacterSkills(charIdx)[skillIdx]
	currentLv := max(g.PlayerSkillLv[charIdx][skillIdx], 1)
	g.PlayerSP[charIdx] -= SkillUpgradeCost(sk, currentLv)
	g.PlayerSkillLv[charIdx][skillIdx] = currentLv + 1
	g.addPlayerStats(charIdx, sk.Levels[currentLv].StatBonus)
	return true
}

// SkillStatBonus totals the StatBonus of every skill level charIdx has
// upgraded into, i.e. how far their stats sit above the level-up table.
func (g *Game) SkillStatBonus(charIdx int) PlayerStats {
	var total PlayerStats
	for i, sk := range g.CharacterSkills(charIdx) {
		if i >= len(g.PlayerSkillLv[charIdx]) {
			break
		}
		for lv := 2; lv <= g.PlayerSkillLv[charIdx][i] && lv <= len(sk.Levels); lv++ {
			total = total.add(sk.Levels[lv-1].StatBonus)
		}
	}
	return total
}

func (a PlayerStats) add(b PlayerStats) PlayerStats {
	return PlayerStats{
		HP:       a.HP + b.HP,
		MP:       a.MP + b.MP,
		PhysAtk:  a.PhysAtk + b.PhysAtk,
		MagicAtk: a.MagicAtk + b.MagicAtk,
		PhysDef:  a.PhysDef + b.PhysDef,
		MagicDef: a.MagicDef + b.MagicDef,
		Spd:      a.Spd + b.Spd,
		Luck:     a.Luck + b.Luck,
	}
}

// addPlayerStats raises charIdx's stats by b; max HP/MP gains also raise
// current HP/MP by the same amount.
func (g *Game) addPlayerStats(charIdx int, b PlayerStats) {
	g.PlayerMaxHP[charIdx] += b.HP
	g.PlayerHP[charIdx] += b.HP
	g.PlayerMaxMP[charIdx] += b.MP
	g.PlayerMP[charIdx] += b.MP
	g.PlayerAtk[charIdx] += b.PhysAtk
	g.PlayerMagicAtk[charIdx] += b.MagicAtk
	g.PlayerDef[charIdx] += b.PhysDef
	g.PlayerMagicDef[charIdx] += b.MagicDef
	g.PlayerSpd[charIdx] += b.Spd
	g.PlayerLuck[charIdx] += b.Luck
}

func (g *Game) CurrentSkillLevelData(charIdx, skillIdx int) SkillLevelData {
	skills := g.CharacterSkills(charIdx)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return SkillLevelData{}
	}
	levels := skills[skillIdx].Levels
	if len(levels) == 0 {
		return SkillLevelData{}
	}
	lv := g.PlayerSkillLv[charIdx][skillIdx]
	if lv < 1 {
		lv = 1
	}
	if lv > len(levels) {
		lv = len(levels)
	}
	return levels[lv-1]
}

// StatKind identifies which stat a Buff or Debuff modifies. statIconOrder
// (battle_draw_hud.go) draws the battle HUD's per-stat up/down icons next to
// a party member's name in this same order.
type StatKind int

const (
	StatAtk StatKind = iota
	StatMat
	StatDef
	StatMdf
	StatLuk
)

// Debuff lasts Seconds of running battle time.
type Debuff struct {
	Type    StatKind
	Percent int
	Seconds float64
}

// effectStats returns the stats a buff/debuff EffectType changes and
// whether it raises them. ok is false for non-stat effects (ATB pushes).
func effectStats(t EffectType) (stats []StatKind, isBuff bool, ok bool) {
	switch t {
	case EffectDebuffPhysicalDef:
		return []StatKind{StatDef}, false, true
	case EffectDebuffMagicDef:
		return []StatKind{StatMdf}, false, true
	case EffectDebuffAtk:
		return []StatKind{StatAtk, StatMat}, false, true
	case EffectDebuffDefBoth:
		return []StatKind{StatDef, StatMdf}, false, true
	case EffectDebuffPhysAtk:
		return []StatKind{StatAtk}, false, true
	case EffectDebuffMagicAtk:
		return []StatKind{StatMat}, false, true
	case EffectBuffAtkUp:
		return []StatKind{StatAtk, StatMat}, true, true
	case EffectBuffPhysAtkUp:
		return []StatKind{StatAtk}, true, true
	case EffectBuffPhysDefUp:
		return []StatKind{StatDef}, true, true
	case EffectBuffDefBothUp:
		return []StatKind{StatDef, StatMdf}, true, true
	}
	return nil, false, false
}

func SumDebuffPercent(list []Debuff, t StatKind) int {
	total := 0
	for _, d := range list {
		if d.Type == t {
			total += d.Percent
		}
	}
	if total > 80 {
		total = 80
	}
	return total
}

// TickDebuffSeconds counts down timed debuffs by dt seconds.
func TickDebuffSeconds(list []Debuff, dt float64) []Debuff {
	var alive []Debuff
	for _, d := range list {
		if d.Seconds > 0 {
			d.Seconds -= dt
			if d.Seconds <= 0 {
				continue
			}
		}
		alive = append(alive, d)
	}
	return alive
}

// Buff lasts Seconds of running battle time, or - when Cover is set -
// until that actor's cover (かばう) runs out.
type Buff struct {
	Type    StatKind
	Percent int
	Seconds float64
	Cover   bool
}

func SumBuffPercent(list []Buff, t StatKind) int {
	total := 0
	for _, b := range list {
		if b.Type == t {
			total += b.Percent
		}
	}
	if total > 80 {
		total = 80
	}
	return total
}

// TickBuffSeconds counts down timed buffs by dt seconds. Cover buffs have
// no Seconds and are left alone.
func TickBuffSeconds(list []Buff, dt float64) []Buff {
	var alive []Buff
	for _, b := range list {
		if b.Seconds > 0 {
			b.Seconds -= dt
			if b.Seconds <= 0 {
				continue
			}
		}
		alive = append(alive, b)
	}
	return alive
}

// withoutCoverBuffs drops the buffs granted by かばう.
func withoutCoverBuffs(list []Buff) []Buff {
	var kept []Buff
	for _, b := range list {
		if !b.Cover {
			kept = append(kept, b)
		}
	}
	return kept
}

// statBonusText formats a skill level's StatBonus for the upgrade menu,
// e.g. "物攻+20 魔防+10".
func statBonusText(b PlayerStats) string {
	parts := []struct {
		name string
		v    int
	}{
		{"HP", b.HP}, {"MP", b.MP}, {"物攻", b.PhysAtk}, {"魔攻", b.MagicAtk},
		{"物防", b.PhysDef}, {"魔防", b.MagicDef}, {"素早さ", b.Spd}, {"運", b.Luck},
	}
	s := ""
	for _, p := range parts {
		if p.v == 0 {
			continue
		}
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%s+%d", p.name, p.v)
	}
	return s
}
