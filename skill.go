package main

type SkillTarget int

const (
	TargetSingle SkillTarget = iota
	TargetAll
	TargetBoth
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
	EffectDebuffAtk
	EffectDebuffDefBoth
	EffectBuffAtkUp
)

type SkillEffect struct {
	Type       EffectType
	Percent    int
	PercentAll int
	Turns      int
}

type SkillLevelData struct {
	Description string
	Target      SkillTarget
	Element     Element
	PowerSingle int
	PowerAll    int
	MPCost      int
	Effects     []SkillEffect
	IsHeal      bool
	// ReturnPosition is where the actor's timeline icon reappears (0-100)
	// after this skill is used, independent of the Spd stat.
	ReturnPosition float64
	// GaugePoint is how many synergy gauge points using this skill adds.
	GaugePoint int
}

type SkillDef struct {
	Name   string
	Levels []SkillLevelData
	// UnlockLevel is the character level required to use this skill.
	UnlockLevel int
}

var characterSkillSets = [4][]SkillDef{
	HeroSkills,
	HeroSkills,
	HeroSkills,
	HeroSkills,
}

func (g *Game) CharacterSkills(charIdx int) []SkillDef {
	if charIdx < 0 || charIdx >= len(characterSkillSets) {
		return HeroSkills
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

var HeroSkills = []SkillDef{
	{
		Name:        "強撃",
		UnlockLevel: 1,
		Levels: []SkillLevelData{
			{
				Description:    "強い攻撃",
				Target:         TargetSingle,
				Element:        ElemPhysicalNone,
				PowerSingle:    150,
				MPCost:         5,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description:    "強い攻撃",
				Target:         TargetSingle,
				Element:        ElemPhysicalNone,
				PowerSingle:    170,
				MPCost:         7,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "強い攻撃 対象に10%の物理防御力低下付与",
				Target:      TargetSingle,
				Element:     ElemPhysicalNone,
				PowerSingle: 200,
				MPCost:      10,
				Effects: []SkillEffect{
					{Type: EffectDebuffPhysicalDef, Percent: 10, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
		},
	},
	{
		Name:        "全体攻撃",
		UnlockLevel: 2,
		Levels: []SkillLevelData{
			{
				Description:    "全体にダメージを与える",
				Target:         TargetAll,
				Element:        ElemMagicNone,
				PowerAll:       80,
				MPCost:         7,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description:    "全体にダメージを与える",
				Target:         TargetAll,
				Element:        ElemMagicNone,
				PowerAll:       100,
				MPCost:         9,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description:    "全体にダメージを与える",
				Target:         TargetAll,
				Element:        ElemMagicNone,
				PowerAll:       140,
				MPCost:         13,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
		},
	},
	{
		Name:        "炎魔法",
		UnlockLevel: 1,
		Levels: []SkillLevelData{
			{
				Description:    "炎でばぁん",
				Target:         TargetBoth,
				Element:        ElemFire,
				PowerSingle:    160,
				PowerAll:       80,
				MPCost:         7,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "炎でばぁん 魔法防御力を15%下げる",
				Target:      TargetBoth,
				Element:     ElemFire,
				PowerSingle: 180,
				PowerAll:    100,
				MPCost:      10,
				Effects: []SkillEffect{
					{Type: EffectDebuffMagicDef, Percent: 15, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "炎でばぁん 魔法防御力を30%下げる",
				Target:      TargetBoth,
				Element:     ElemFire,
				PowerSingle: 210,
				PowerAll:    140,
				MPCost:      14,
				Effects: []SkillEffect{
					{Type: EffectDebuffMagicDef, Percent: 30, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
		},
	},
	{
		Name:        "デバフ",
		UnlockLevel: 1,
		Levels: []SkillLevelData{
			{
				Description: "対象の物理魔法攻撃力を10%下げる",
				Target:      TargetSingle,
				Element:     ElemPhysicalNone,
				MPCost:      3,
				Effects: []SkillEffect{
					{Type: EffectDebuffAtk, Percent: 10, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "対象の物理魔法攻撃力を20%下げる 対象の物理防御力を15%下げる",
				Target:      TargetSingle,
				Element:     ElemPhysicalNone,
				MPCost:      9,
				Effects: []SkillEffect{
					{Type: EffectDebuffAtk, Percent: 20, Turns: 3},
					{Type: EffectDebuffPhysicalDef, Percent: 15, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "対象の物理魔法攻撃力を30%下げる 対象の物理魔法防御力を15%下げる",
				Target:      TargetAll,
				Element:     ElemPhysicalNone,
				MPCost:      15,
				Effects: []SkillEffect{
					{Type: EffectDebuffAtk, Percent: 30, Turns: 3},
					{Type: EffectDebuffDefBoth, Percent: 15, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
		},
	},
	{
		Name:        "回復",
		UnlockLevel: 1,
		Levels: []SkillLevelData{
			{
				Description:    "対象のHPを回復する",
				Target:         TargetBoth,
				Element:        ElemMagicNone,
				PowerSingle:    20,
				PowerAll:       10,
				MPCost:         6,
				IsHeal:         true,
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "対象のHPを回復する 対象に物理魔法攻撃力up",
				Target:      TargetBoth,
				Element:     ElemMagicNone,
				PowerSingle: 30,
				PowerAll:    15,
				MPCost:      10,
				IsHeal:      true,
				Effects: []SkillEffect{
					{Type: EffectBuffAtkUp, Percent: 15, PercentAll: 10, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
			{
				Description: "対象のHPを大きく回復する 対象に物理魔法攻撃力up",
				Target:      TargetBoth,
				Element:     ElemMagicNone,
				PowerSingle: 50,
				PowerAll:    30,
				MPCost:      20,
				IsHeal:      true,
				Effects: []SkillEffect{
					{Type: EffectBuffAtkUp, Percent: 30, PercentAll: 15, Turns: 3},
				},
				ReturnPosition: 5,
				GaugePoint:     2,
			},
		},
	},
}

func SkillUpgradeCost(currentLv int) int {
	switch currentLv {
	case 1:
		return 100
	case 2:
		return 500
	default:
		return -1
	}
}

const MaxSkillLevel = 3

func (g *Game) CanUpgradeSkill(charIdx, skillIdx int) bool {
	if charIdx < 0 || charIdx >= 4 {
		return false
	}
	skills := g.CharacterSkills(charIdx)
	if skillIdx < 0 || skillIdx >= len(skills) {
		return false
	}
	currentLv := g.PlayerSkillLv[charIdx][skillIdx]
	if currentLv < 1 {
		currentLv = 1
	}
	if currentLv >= MaxSkillLevel {
		return false
	}
	cost := SkillUpgradeCost(currentLv)
	if cost < 0 {
		return false
	}
	if currentLv >= len(skills[skillIdx].Levels) {
		return false
	}
	return g.PlayerSP[charIdx] >= cost
}

func (g *Game) UpgradeSkill(charIdx, skillIdx int) bool {
	if !g.CanUpgradeSkill(charIdx, skillIdx) {
		return false
	}
	currentLv := g.PlayerSkillLv[charIdx][skillIdx]
	cost := SkillUpgradeCost(currentLv)
	g.PlayerSP[charIdx] -= cost
	g.PlayerSkillLv[charIdx][skillIdx]++
	return true
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

type Debuff struct {
	Type    StatKind
	Percent int
	Turns   int
}

func mapEffectToDebuff(t EffectType) StatKind {
	switch t {
	case EffectDebuffPhysicalDef:
		return StatDef
	case EffectDebuffMagicDef:
		return StatMdf
	case EffectDebuffAtk:
		return StatAtk
	case EffectDebuffDefBoth:
		return StatDef
	default:
		return StatAtk
	}
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

func TickDebuffList(list []Debuff) []Debuff {
	var alive []Debuff
	for _, d := range list {
		d.Turns--
		if d.Turns > 0 {
			alive = append(alive, d)
		}
	}
	return alive
}

type Buff struct {
	Type    StatKind
	Percent int
	Turns   int
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

func TickBuffList(list []Buff) []Buff {
	var alive []Buff
	for _, b := range list {
		b.Turns--
		if b.Turns > 0 {
			alive = append(alive, b)
		}
	}
	return alive
}
