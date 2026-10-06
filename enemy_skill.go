package main

type EnemySkill struct {
	Name        string
	Description string
	Target      SkillTarget
	Element     Element
	Power       int
	// Effects are applied to the party members hit. Only debuffs and ATB
	// pushes make sense here: buff effects would land on the player. Give
	// every skill some Power: a 0-power hit still shows a "0" damage pop.
	Effects []SkillEffect
	// ReturnPosition is where the enemy's timeline icon reappears (0-100)
	// after this skill is used, independent of the Speed stat.
	ReturnPosition float64
}

const EnemySkillChance = 30

// enemySkillTable is every enemy/boss skill, keyed by skill ID. Which enemy
// uses which skills is set by ID in the spreadsheet's Skills column
// (EnemyStats.SkillIDs, via tools/genstats), so one skill can be shared by
// any number of enemies. genstats reads the keys from this file: keep each
// entry as `\t"id": {` on its own line.
var enemySkillTable = map[string]EnemySkill{
	// ---- ボス ----
	"ice_breath": {
		Name:           "氷結ブレス",
		Description:    "対象に魔法ダメージ",
		Target:         TargetSingle,
		Element:        ElemMagicNone,
		Power:          120,
		ReturnPosition: 5,
	},
	"wind_slash": {
		Name:        "風斬り",
		Description: "対象に物理ダメージ 行動順を少し下げる",
		Target:      TargetSingle,
		Element:     ElemPhysicalNone,
		Power:       110,
		Effects: []SkillEffect{
			{Type: EffectAtbDownSmall},
		},
		ReturnPosition: 5,
	},
	"thunder_bolt": {
		Name:           "雷撃",
		Description:    "対象に魔法ダメージ",
		Target:         TargetSingle,
		Element:        ElemMagicNone,
		Power:          125,
		ReturnPosition: 5,
	},
	"flame_burst": {
		Name:           "火炎放射",
		Description:    "味方全員に魔法ダメージ",
		Target:         TargetAll,
		Element:        ElemMagicNone,
		Power:          70,
		ReturnPosition: 5,
	},
	"crushing_blow": {
		Name:        "粉砕撃",
		Description: "対象に強い物理ダメージ 物理防御力を少し下げる",
		Target:      TargetSingle,
		Element:     ElemPhysicalNone,
		Power:       150,
		Effects: []SkillEffect{
			{Type: EffectDebuffPhysicalDef, Percent: 10, Seconds: 10},
		},
		ReturnPosition: 5,
	},

	// ---- 雑魚敵 ----
	"tackle": {
		Name:           "体当たり",
		Description:    "対象に物理ダメージ",
		Target:         TargetSingle,
		Element:        ElemPhysicalNone,
		Power:          115,
		ReturnPosition: 5,
	},
	"club_smash": {
		Name:           "こん棒殴り",
		Description:    "対象にやや強い物理ダメージ",
		Target:         TargetSingle,
		Element:        ElemPhysicalNone,
		Power:          125,
		ReturnPosition: 5,
	},
	"heavy_slam": {
		Name:        "叩きつけ",
		Description: "対象に強い物理ダメージ 攻撃力を少し下げる",
		Target:      TargetSingle,
		Element:     ElemPhysicalNone,
		Power:       130,
		Effects: []SkillEffect{
			{Type: EffectDebuffAtk, Percent: 10, Seconds: 10},
		},
		ReturnPosition: 5,
	},
	"bite": {
		Name:        "噛みつき",
		Description: "対象に物理ダメージ 行動順を少し下げる",
		Target:      TargetSingle,
		Element:     ElemPhysicalNone,
		Power:       105,
		Effects: []SkillEffect{
			{Type: EffectAtbDownSmall},
		},
		ReturnPosition: 5,
	},
	"tail_sweep": {
		Name:           "尻尾払い",
		Description:    "味方全員に物理ダメージ",
		Target:         TargetAll,
		Element:        ElemPhysicalNone,
		Power:          60,
		ReturnPosition: 5,
	},
	"supersonic": {
		Name:        "超音波",
		Description: "対象に魔法ダメージ 行動順を下げる",
		Target:      TargetSingle,
		Element:     ElemMagicNone,
		Power:       80,
		Effects: []SkillEffect{
			{Type: EffectAtbDownLarge},
		},
		ReturnPosition: 5,
	},
	"spore_cloud": {
		Name:        "胞子",
		Description: "味方全員に魔法ダメージ 魔法防御力を少し下げる",
		Target:      TargetAll,
		Element:     ElemMagicNone,
		Power:       50,
		Effects: []SkillEffect{
			{Type: EffectDebuffMagicDef, Percent: 10, Seconds: 10},
		},
		ReturnPosition: 5,
	},
	"rock_punch": {
		Name:           "岩石パンチ",
		Description:    "対象に強い物理ダメージ",
		Target:         TargetSingle,
		Element:        ElemPhysicalNone,
		Power:          140,
		ReturnPosition: 5,
	},
	"bone_cleave": {
		Name:        "骨断ち",
		Description: "対象に強い物理ダメージ 物理防御力を少し下げる",
		Target:      TargetSingle,
		Element:     ElemPhysicalNone,
		Power:       130,
		Effects: []SkillEffect{
			{Type: EffectDebuffPhysicalDef, Percent: 10, Seconds: 10},
		},
		ReturnPosition: 5,
	},
}

// enemySkillList looks up skill IDs in enemySkillTable. Unknown IDs are
// skipped (TestEnemySkillIDsExist catches typos).
func enemySkillList(ids []string) []EnemySkill {
	if len(ids) == 0 {
		return nil
	}
	out := make([]EnemySkill, 0, len(ids))
	for _, id := range ids {
		if sk, ok := enemySkillTable[id]; ok {
			out = append(out, sk)
		}
	}
	return out
}
