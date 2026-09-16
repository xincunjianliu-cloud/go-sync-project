package main

type EnemySkill struct {
	Name        string
	Description string
	Target      SkillTarget
	Element     Element
	Power       int
	Effects     []SkillEffect
}

const EnemySkillChance = 30

var SkillIceBreath = EnemySkill{
	Name:        "氷結ブレス",
	Description: "対象に魔法ダメージ",
	Target:      TargetSingle,
	Element:     ElemMagicNone,
	Power:       120,
}

var SkillWindSlash = EnemySkill{
	Name:        "風斬り",
	Description: "対象に物理ダメージ 行動順を少し下げる",
	Target:      TargetSingle,
	Element:     ElemPhysicalNone,
	Power:       110,
	Effects: []SkillEffect{
		{Type: EffectAtbDownSmall},
	},
}

var SkillThunderBolt = EnemySkill{
	Name:        "雷撃",
	Description: "対象に魔法ダメージ",
	Target:      TargetSingle,
	Element:     ElemMagicNone,
	Power:       125,
}

var SkillFlameBurst = EnemySkill{
	Name:        "火炎放射",
	Description: "味方全員に魔法ダメージ",
	Target:      TargetAll,
	Element:     ElemMagicNone,
	Power:       70,
}

var SkillCrushingBlow = EnemySkill{
	Name:        "粉砕撃",
	Description: "対象に強い物理ダメージ 物理防御力を少し下げる",
	Target:      TargetSingle,
	Element:     ElemPhysicalNone,
	Power:       150,
	Effects: []SkillEffect{
		{Type: EffectDebuffPhysicalDef, Percent: 10, Turns: 3},
	},
}
