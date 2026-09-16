package main

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

const (
	critChanceBase     = 5
	critChancePerLuck5 = 1
	critDamageMultiply = 1.5

	evadeChanceBase     = 1
	evadeChancePerLuck5 = 1
)

func critChancePercent(luck int) int {
	if luck < 0 {
		luck = 0
	}
	return critChanceBase + (luck/5)*critChancePerLuck5
}

func evadeChancePercent(luck int) int {
	if luck < 0 {
		luck = 0
	}
	return evadeChanceBase + (luck/5)*evadeChancePerLuck5
}
