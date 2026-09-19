package main

// Drop tables and skill lists for enemies/bosses, keyed by Name (EnemyDatabase)
// or boss key (BossDatabase). Numeric stats (HP, ATK, etc.) live in the
// spreadsheet-driven stats_generated.go instead — see tools/genstats.

var enemyDrops = map[string][]ItemDrop{
	"スライム1":      {{ItemID: "potion", Percent: 100}},
	"スライム2":      {{ItemID: "potion", Percent: 100}},
	"スライム3":      {{ItemID: "potion", Percent: 100}},
	"ゴブリン1":      {{ItemID: "potion", Percent: 100}},
	"ゴブリン2":      {{ItemID: "potion", Percent: 100}},
	"ゴブリン3":      {{ItemID: "potion", Percent: 100}},
	"オーク1":       {{ItemID: "potion", Percent: 100}},
	"オーク2":       {{ItemID: "potion", Percent: 100}},
	"オーク3":       {{ItemID: "potion", Percent: 100}},
	"コボルト1":      {{ItemID: "potion", Percent: 100}},
	"コボルト2":      {{ItemID: "potion", Percent: 100}},
	"コボルト3":      {{ItemID: "potion", Percent: 100}},
	"リザードマン1":    {{ItemID: "potion", Percent: 100}},
	"リザードマン2":    {{ItemID: "potion", Percent: 100}},
	"リザードマン3":    {{ItemID: "potion", Percent: 100}},
	"ワイルドウルフ1":   {{ItemID: "potion", Percent: 100}},
	"ワイルドウルフ2":   {{ItemID: "potion", Percent: 100}},
	"ワイルドウルフ3":   {{ItemID: "potion", Percent: 100}},
	"ジャイアントバット1": {{ItemID: "potion", Percent: 100}},
	"ジャイアントバット2": {{ItemID: "potion", Percent: 100}},
	"ジャイアントバット3": {{ItemID: "potion", Percent: 100}},
	"マッドプラント1":   {{ItemID: "hi_potion", Percent: 100}},
	"マッドプラント2":   {{ItemID: "hi_potion", Percent: 100}},
	"マッドプラント3":   {{ItemID: "hi_potion", Percent: 100}},
	"ストーンゴーレム1":  {{ItemID: "hi_potion", Percent: 100}},
	"ストーンゴーレム2":  {{ItemID: "hi_potion", Percent: 100}},
	"ストーンゴーレム3":  {{ItemID: "hi_potion", Percent: 100}},
	"スケルトン1":     {{ItemID: "hi_potion", Percent: 100}},
	"スケルトン2":     {{ItemID: "hi_potion", Percent: 100}},
	"スケルトン3":     {{ItemID: "hi_potion", Percent: 100}},
}

var bossDrops = map[string][]ItemDrop{
	"boss_1": {{ItemID: "hi_potion", Percent: 100}},
	"boss_2": {{ItemID: "ether", Percent: 100}},
	"boss_3": {{ItemID: "phoenix_down", Percent: 100}},
	"boss_4": {{ItemID: "elixir", Percent: 100}},
}

var bossSkills = map[string][]EnemySkill{
	"boss_1": {SkillFlameBurst, SkillCrushingBlow},
	"boss_2": {SkillThunderBolt, SkillCrushingBlow},
	"boss_3": {SkillIceBreath, SkillCrushingBlow},
	"boss_4": {SkillWindSlash, SkillFlameBurst},
}
