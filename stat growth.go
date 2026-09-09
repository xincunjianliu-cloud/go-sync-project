package main

// stat_growth.go: レベルアップ時のステータス成長を「固定値」ではなく
// キャラクターごと・ステータスごとに個別設定できるようにするための定義。
//
// 使い方：
//   - PlayerGrowthRates[キャラIndex] の各フィールドを書き換えるだけで、
//     そのキャラの「レベルが1つ上がるごとの伸び幅」をステータス単位で個別に調整できる。
//   - 次レベルに必要なEXPは PlayerExpCurve(キャラIndex, 現在Lv) が返す値を使う。
//     expCurveParams を編集すればキャラごとにEXPカーブも個別に変えられる。
//   - レベル上限は maxPlayerLevel（game.go で 50 に設定）。

// LevelGrowth は「レベルが1つ上がるごとに、各ステータスがどれだけ伸びるか」を表す。
type LevelGrowth struct {
	HP       int // 最大HPの伸び
	MP       int // 最大MPの伸び
	PhysAtk  int // 物理攻撃力の伸び
	MagicAtk int // 魔法攻撃力の伸び
	PhysDef  int // 物理防御力の伸び
	MagicDef int // 魔法防御力の伸び
	Spd      int // すばやさ（タイムライン上でアイコンが進む速さ）の伸び
	Luck     int // 運（会心率・回避率に影響）の伸び
}

// PlayerGrowthRates: キャラクターごとの「レベル+1あたりの伸び幅」。
// ここを書き換えるだけで、キャラごとの成長曲線を個別に調整できる。
// （将来的にレベル帯によって伸び幅を変えたい場合は、この配列を
//   [partySize][maxPlayerLevel-1]LevelGrowth に拡張し、
//   ApplyLevelUpGrowth 側で currentLv を見て参照するよう変更すればよい）
var PlayerGrowthRates = [partySize]LevelGrowth{
	// プレイヤー1：物理アタッカー寄り
	{HP: 12, MP: 4, PhysAtk: 4, MagicAtk: 1, PhysDef: 2, MagicDef: 1, Spd: 1, Luck: 1},
	// プレイヤー2：魔法・回復寄り
	{HP: 8, MP: 7, PhysAtk: 1, MagicAtk: 4, PhysDef: 1, MagicDef: 3, Spd: 2, Luck: 2},
	// プレイヤー3：バランス型
	{HP: 11, MP: 5, PhysAtk: 3, MagicAtk: 2, PhysDef: 3, MagicDef: 2, Spd: 1, Luck: 1},
	// プレイヤー4：素早さ・運寄り
	{HP: 9, MP: 4, PhysAtk: 2, MagicAtk: 2, PhysDef: 2, MagicDef: 2, Spd: 3, Luck: 3},
}

// expCurveParam: 次レベルに必要なEXP = Base + PerLevel * 現在Lv
type expCurveParam struct {
	Base     int
	PerLevel int
}

// expCurveParams もキャラごとに個別設定可能（EXPが伸びやすい/にくいキャラを作れる）。
var expCurveParams = [partySize]expCurveParam{
	{Base: 20, PerLevel: 45}, // プレイヤー1
	{Base: 15, PerLevel: 50}, // プレイヤー2
	{Base: 20, PerLevel: 48}, // プレイヤー3
	{Base: 15, PerLevel: 47}, // プレイヤー4
}

// PlayerExpCurve は actor が currentLv から次のレベルへ上がるのに必要なEXPを返す。
func PlayerExpCurve(actor, currentLv int) int {
	if actor < 0 || actor >= partySize {
		return currentLv * 50
	}
	if currentLv < 1 {
		currentLv = 1
	}
	p := expCurveParams[actor]
	need := p.Base + p.PerLevel*currentLv
	if need < 1 {
		need = 1
	}
	return need
}

// ApplyLevelUpGrowth は actor のレベルが1つ上がった際に、
// PlayerGrowthRates[actor] に従って各ステータスを増加させる。
// HP/MPは増加後、全回復させる（既存仕様を踏襲）。
func (g *Game) ApplyLevelUpGrowth(actor int) {
	if actor < 0 || actor >= partySize {
		return
	}
	gr := PlayerGrowthRates[actor]

	g.PlayerMaxHP[actor] += gr.HP
	g.PlayerHP[actor] = g.PlayerMaxHP[actor]

	g.PlayerMaxMP[actor] += gr.MP
	g.PlayerMP[actor] = g.PlayerMaxMP[actor]

	g.PlayerAtk[actor] += gr.PhysAtk
	g.PlayerMagicAtk[actor] += gr.MagicAtk
	g.PlayerDef[actor] += gr.PhysDef
	g.PlayerMagicDef[actor] += gr.MagicDef
	g.PlayerSpd[actor] += gr.Spd
	g.PlayerLuck[actor] += gr.Luck
}

// ── 運（Luck）による会心・回避の判定 ──────────────────────────
// 敵は運ステータスを持たないため、敵の攻撃は会心せず、
// 敵に対する回避判定も（味方側の運のみで）通常通り行われる。

const (
	critChancePerLuck  = 2  // 運1につき会心率+2%
	critChanceMax      = 40 // 会心率の上限(%)
	critDamageMultiply = 1.5

	evadeChancePerLuck = 1  // 運1につき回避率+1%
	evadeChanceMax     = 25 // 回避率の上限(%)
)

// critChancePercent は luck から会心率(%)を算出する。
func critChancePercent(luck int) int {
	c := luck * critChancePerLuck
	if c < 0 {
		c = 0
	}
	if c > critChanceMax {
		c = critChanceMax
	}
	return c
}

// evadeChancePercent は luck から回避率(%)を算出する。
func evadeChancePercent(luck int) int {
	c := luck * evadeChancePerLuck
	if c < 0 {
		c = 0
	}
	if c > evadeChanceMax {
		c = evadeChanceMax
	}
	return c
}
