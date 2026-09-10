package main

// ============================================================
// skill.go
// スキル・スキル強化(SP)・デバフ関連のデータと処理をまとめたファイル
// ============================================================

// ── スキルの対象範囲 ─────────────────────────────────────────
type SkillTarget int

const (
	TargetSingle SkillTarget = iota // 単体のみ
	TargetAll                       // 全体のみ
	TargetBoth                      // 単体/全体をプレイヤーが選択可能
)

// ── 属性 ────────────────────────────────────────────────
type Element int

const (
	ElemPhysicalNone Element = iota // 物理無属性
	ElemMagicNone                   // 魔法無属性
	ElemFire                        // 火
	ElemLightning                   // 雷
	ElemIce                         // 氷
	ElemWind                        // 風
)

const elementalTypeCount = 4

// ── スキル追加効果の種類 ──────────────────────────────────────
type EffectType int

const (
	EffectAtbDownSmall      EffectType = iota // 行動順を少し下げる（ワンショット）
	EffectAtbDownLarge                        // 行動順を結構下げる（ワンショット）
	EffectDebuffPhysicalDef                   // 物理防御力ダウン（持続）
	EffectDebuffMagicDef                      // 魔法防御力ダウン（持続）
	EffectDebuffAtk                           // 物理魔法攻撃力ダウン（持続）
	EffectDebuffDefBoth                       // 物理魔法防御力ダウン（持続）
)

// SkillEffect はスキル使用時に発生する追加効果一つ分。
// Turns が 0 の場合はワンショット効果（ATB操作など）として扱う。
type SkillEffect struct {
	Type    EffectType
	Percent int // デバフ系のみ使用（%）
	Turns   int // 持続ターン数（行動回数ベース）。0ならワンショット
}

// SkillLevelData はスキルの特定レベルにおけるデータ一式。
type SkillLevelData struct {
	Description string
	Target      SkillTarget
	Element     Element
	PowerSingle int // 単体威力。使わないスキルは0
	PowerAll    int // 全体威力。使わないスキルは0
	MPCost      int
	Effects     []SkillEffect
	IsHeal      bool
}

// SkillDef は一つのスキルが持つ、レベル1〜3のデータをまとめたもの。
// Levels[0] = Lv1, Levels[1] = Lv2, Levels[2] = Lv3
type SkillDef struct {
	Name   string
	Levels []SkillLevelData
}

// ============================================================
// キャラクターごとのスキルセット管理
// ============================================================

// characterSkillSets は各キャラ(0〜3)が使用するスキルセット。
// 現在は全員 HeroSkills を共有しているが、将来的にキャラごとに
// 専用のスキルリストへ差し替えられるようにこの配列経由で参照する。
// 例）P2だけ専用スキルにしたい場合:
//
//	characterSkillSets[1] = MageSkills
var characterSkillSets = [4][]SkillDef{
	HeroSkills, // P1
	HeroSkills, // P2（暫定でP1と共有）
	HeroSkills, // P3（暫定でP1と共有）
	HeroSkills, // P4（暫定でP1と共有）
}

// CharacterSkills は指定キャラのスキルリストを返す。
func (g *Game) CharacterSkills(charIdx int) []SkillDef {
	if charIdx < 0 || charIdx >= len(characterSkillSets) {
		return HeroSkills
	}
	return characterSkillSets[charIdx]
}

// ── 主人公のスキル定義 ────────────────────────────────────────
// index: 0=強撃(旧・強攻撃を置き換え) 1=全体攻撃 2=炎魔法 3=デバフ
var HeroSkills = []SkillDef{
	{ // 0: 強撃
		Name: "強撃",
		Levels: []SkillLevelData{
			{
				Description: "強い攻撃",
				Target:      TargetSingle,
				Element:     ElemPhysicalNone,
				PowerSingle: 150,
				MPCost:      5,
			},
			{
				Description: "強い攻撃 対象の行動順を少し下げる",
				Target:      TargetSingle,
				Element:     ElemPhysicalNone,
				PowerSingle: 170,
				MPCost:      7,
				Effects: []SkillEffect{
					{Type: EffectAtbDownSmall},
				},
			},
			{
				Description: "強い攻撃 対象の行動順を結構下げる 対象に10%の物理防御力低下付与",
				Target:      TargetSingle,
				Element:     ElemPhysicalNone,
				PowerSingle: 200,
				MPCost:      10,
				Effects: []SkillEffect{
					{Type: EffectAtbDownLarge},
					{Type: EffectDebuffPhysicalDef, Percent: 10, Turns: 3},
				},
			},
		},
	},
	{ // 1: 全体攻撃
		Name: "全体攻撃",
		Levels: []SkillLevelData{
			{
				Description: "全体にダメージを与える",
				Target:      TargetAll,
				Element:     ElemLightning,
				PowerAll:    80,
				MPCost:      7,
			},
			{
				Description: "全体にダメージを与える",
				Target:      TargetAll,
				Element:     ElemMagicNone,
				PowerAll:    100,
				MPCost:      9,
			},
			{
				Description: "全体にダメージを与える 対象の行動順を少し下げる",
				Target:      TargetAll,
				Element:     ElemMagicNone,
				PowerAll:    140,
				MPCost:      13,
				Effects: []SkillEffect{
					{Type: EffectAtbDownSmall},
				},
			},
		},
	},
	{ // 2: 炎魔法
		Name: "炎魔法",
		Levels: []SkillLevelData{
			{
				Description: "炎でばぁん",
				Target:      TargetBoth,
				Element:     ElemFire,
				PowerSingle: 160,
				PowerAll:    80,
				MPCost:      7,
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
			},
		},
	},
	{ // 3: デバフ
		Name: "デバフ",
		Levels: []SkillLevelData{
			{
				Description: "対象の物理魔法攻撃力を10%下げる",
				Target:      TargetSingle,
				Element:     ElemWind,
				MPCost:      3,
				Effects: []SkillEffect{
					{Type: EffectDebuffAtk, Percent: 10, Turns: 3},
				},
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
			},
		},
	},
	{ // 4: 回復
		Name: "回復",
		Levels: []SkillLevelData{
			{
				Description: "対象のHPを回復する",
				Target:      TargetBoth,
				Element:     ElemIce,
				PowerSingle: 120,
				PowerAll:    60,
				MPCost:      6,
				IsHeal:      true,
			},
			{
				Description: "対象のHPを回復する (回復量アップ)",
				Target:      TargetBoth,
				Element:     ElemMagicNone,
				PowerSingle: 150,
				PowerAll:    80,
				MPCost:      9,
				IsHeal:      true,
			},
			{
				Description: "対象のHPを大きく回復する",
				Target:      TargetBoth,
				Element:     ElemMagicNone,
				PowerSingle: 200,
				PowerAll:    110,
				MPCost:      13,
				IsHeal:      true,
			},
		},
	},
}

func healSkillIndex(skills []SkillDef) int {
	for i, sk := range skills {
		if len(sk.Levels) > 0 && sk.Levels[0].IsHeal {
			return i
		}
	}
	return -1
}

// ============================================================
// スキル強化（SP消費）関連
// ============================================================

// SkillUpgradeCost は現在のスキルレベルから次のレベルへ上げるのに
// 必要なSPを返す。Lv1→2:100, Lv2→3:500。
// 最大レベル(3)に達している場合は -1 を返す（強化不可の意味）。
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

// MaxSkillLevel はスキルの最大レベル。
const MaxSkillLevel = 3

// CanUpgradeSkill は指定キャラ・指定スキルが強化可能かどうかを返す。
// CanUpgradeSkill は指定キャラ・指定スキルが強化可能かどうかを返す。
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

// UpgradeSkill はSPを消費してスキルレベルを1上げる。
// 成功したらtrue、SP不足や最大レベル到達などで失敗したらfalseを返す。
// ★修正：MPCostは戦闘中にそのスキルを「使う」ための消費MPであり、
// 強化(レベルアップ)とは無関係。ここでMPを要求・消費していたため、
// SPが足りていても直前の強化でMPを使い切っていると強化不可になるバグがあった。
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

// CurrentSkillLevelData は指定キャラ・指定スキルの「現在のレベル」データを返す。
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

// ============================================================
// デバフ（持続効果）関連
// ============================================================

type DebuffType int

const (
	DebuffAtkDown      DebuffType = iota // 物理魔法攻撃力ダウン
	DebuffMagicAtkDown                   // （将来的に魔法攻撃力のみを個別に下げたい場合用）
	DebuffDefDown                        // 物理防御力ダウン
	DebuffMagicDefDown                   // 魔法防御力ダウン
)

// Debuff は持続系の弱体効果一つ分。
// Turns が 0 以下になったら除去する。
type Debuff struct {
	Type    DebuffType
	Percent int
	Turns   int
}

// mapEffectToDebuff は SkillEffect の EffectType を、
// 持続管理用の DebuffType に変換する。
// EffectAtbDownSmall / EffectAtbDownLarge はワンショットなので呼ばれない想定。
func mapEffectToDebuff(t EffectType) DebuffType {
	switch t {
	case EffectDebuffPhysicalDef:
		return DebuffDefDown
	case EffectDebuffMagicDef:
		return DebuffMagicDefDown
	case EffectDebuffAtk:
		return DebuffAtkDown
	case EffectDebuffDefBoth:
		// 物理魔法防御同時ダウンは呼び出し側で
		// DebuffDefDown / DebuffMagicDefDown の2件に分けて追加する想定
		return DebuffDefDown
	default:
		return DebuffAtkDown
	}
}

// SumDebuffPercent は指定タイプのデバフ合計%を返す（複数重複時は加算、上限80%）。
func SumDebuffPercent(list []Debuff, t DebuffType) int {
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

// TickDebuffList は行動終了時に呼び、各デバフの残りターンを1減らし、
// 0以下になったものを除去したリストを返す。
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
