package main

// stat_growth.go: レベルアップ処理と、運（Luck）による会心・回避の判定。
//
// ステータスの実数値そのもの（HP/MP/攻撃力/防御力/すばやさ/運、EXPカーブなど）は
// すべて stats_config.go の PlayerStatsByLevel / PlayerExpToNextByLevel に一本化されている。
// このファイルは「レベルが上がった時にその表を読み込んで反映する処理」と、
// 「運ステータスから会心率・回避率を算出する処理」のみを担当する。

// ApplyLevelUpGrowth は actor のレベルが1つ上がった際に、
// stats_config.go の PlayerStatsByLevel[新しいLv-1][actor] の値をそのまま適用する。
// （＝レベルごとに個別設定したステータス表を単純に読み込むだけなので、
//
//	特定レベルだけ手動で数値を変えても、その通りに反映される）
//
// HPもMPも、最大値の増加分だけ現在値が増える（全回復はしない）。
func (g *Game) ApplyLevelUpGrowth(actor int) {
	if actor < 0 || actor >= partySize {
		return
	}
	lv := g.PlayerLv[actor]
	if lv < 1 {
		lv = 1
	}
	if lv > maxPlayerLevel {
		lv = maxPlayerLevel
	}
	st := PlayerStatsByLevel[lv-1][actor]

	hpGain := st.HP - g.PlayerMaxHP[actor]
	g.PlayerMaxHP[actor] = st.HP
	g.PlayerHP[actor] += hpGain
	if g.PlayerHP[actor] > g.PlayerMaxHP[actor] {
		g.PlayerHP[actor] = g.PlayerMaxHP[actor]
	}
	if g.PlayerHP[actor] < 1 {
		g.PlayerHP[actor] = 1
	}

	mpGain := st.MP - g.PlayerMaxMP[actor]
	g.PlayerMaxMP[actor] = st.MP
	g.PlayerMP[actor] += mpGain
	if g.PlayerMP[actor] > g.PlayerMaxMP[actor] {
		g.PlayerMP[actor] = g.PlayerMaxMP[actor]
	}
	if g.PlayerMP[actor] < 0 {
		g.PlayerMP[actor] = 0
	}

	g.PlayerAtk[actor] = st.PhysAtk
	g.PlayerMagicAtk[actor] = st.MagicAtk
	g.PlayerDef[actor] = st.PhysDef
	g.PlayerMagicDef[actor] = st.MagicDef
	g.PlayerSpd[actor] = st.Spd
	g.PlayerLuck[actor] = st.Luck
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
