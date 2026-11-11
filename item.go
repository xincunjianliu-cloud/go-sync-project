package main

import "math/rand"

type ItemDef struct {
	ID          string
	Name        string
	Description string

	Target SkillTarget

	HealHP        int
	HealHPPercent int
	HealMP        int
	HealMPPercent int

	Revive                bool
	CureDebuff            bool
	GaugePoint            int
	ReturnPosition        float64
	DoubleEXP             bool
	GuaranteedEscape      bool
	NextActionCTReduction int

	UsableInField  bool
	UsableInBattle bool
}

var ItemDatabase = []ItemDef{
	{
		ID:             "potion",
		Name:           "ばんそうこう",
		Description:    "味方1人のHPを50回復させます。",
		Target:         TargetSingle,
		HealHP:         50,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "hi_potion",
		Name:           "すごいばんそうこう",
		Description:    "味方1人のHPを150回復させます。",
		Target:         TargetSingle,
		HealHP:         150,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "legendary_bandage",
		Name:           "伝説のばんそうこう",
		Description:    "味方1人のHPを300回復させます。",
		Target:         TargetSingle,
		HealHP:         300,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "ether",
		Name:           "エナジードリンク",
		Description:    "味方1人のMPを15回復させます。",
		Target:         TargetSingle,
		HealMP:         15,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "great_energy_drink",
		Name:           "すごいエナジードリンク",
		Description:    "味方1人のMPを30回復させます。",
		Target:         TargetSingle,
		HealMP:         30,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "legendary_energy_drink",
		Name:           "伝説のエナジードリンク",
		Description:    "味方1人のMPを50回復させます。",
		Target:         TargetSingle,
		HealMP:         50,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "elixir",
		Name:           "ばんそこドリンク",
		Description:    "味方1人のHP,MPを全回復させます。",
		Target:         TargetSingle,
		HealHPPercent:  100,
		HealMPPercent:  100,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "party_elixir",
		Name:           "ファイナルばんそこドリンク",
		Description:    "味方全員のHP,MPを全回復させます。",
		Target:         TargetAll,
		HealHPPercent:  100,
		HealMPPercent:  100,
		GaugePoint:     3,
		ReturnPosition: 10,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "phoenix_down",
		Name:           "不死鳥の息吹缶",
		Description:    "味方1人を戦闘不能から復活させ、HPを25%回復します。",
		Target:         TargetSingle,
		HealHPPercent:  25,
		Revive:         true,
		GaugePoint:     1,
		ReturnPosition: 50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "timer_booster",
		Name:           "タイマーブースター",
		Description:    "効果はないが、ゲージポイントが多く手に入ります。",
		Target:         TargetOther,
		GaugePoint:     6,
		ReturnPosition: 50,
		UsableInBattle: true,
	},
	{
		ID:             "exp_booster",
		Name:           "けいけんブースター",
		Description:    "戦闘終了後の獲得EXPが2倍になります（ボス戦では無効）。",
		Target:         TargetAll,
		GaugePoint:     1,
		ReturnPosition: 50,
		DoubleEXP:      true,
		UsableInBattle: true,
	},
	{
		ID:               "smoke_bomb",
		Name:             "さよならえんまく",
		Description:      "確実に戦闘から逃げます（ボス戦では使用できません）。",
		Target:           TargetOther,
		GuaranteedEscape: true,
		UsableInBattle:   true,
	},
	{
		ID:                    "speed_booster",
		Name:                  "スピードブースター",
		Description:           "味方1人の次の行動のCTが1/2になります（2回）。",
		Target:                TargetSingle,
		GaugePoint:            1,
		ReturnPosition:        50,
		NextActionCTReduction: 2,
		UsableInBattle:        true,
	},
}

func GetItemDef(id string) (ItemDef, bool) {
	for _, it := range ItemDatabase {
		if it.ID == id {
			return it, true
		}
	}
	return ItemDef{}, false
}

// filterInventory は所持数が1以上で、定義がkeepを満たすスロットだけを返す。
func filterInventory(inv []InventorySlot, keep func(ItemDef) bool) []InventorySlot {
	var list []InventorySlot
	for _, slot := range inv {
		if slot.Count <= 0 {
			continue
		}
		if def, ok := GetItemDef(slot.ItemID); ok && keep(def) {
			list = append(list, slot)
		}
	}
	return list
}

type ItemDrop struct {
	ItemID   string
	Percent  int
	MinCount int
	MaxCount int
}

func (d ItemDrop) rollCount() int {
	min, max := d.MinCount, d.MaxCount
	if min <= 0 {
		min = 1
	}
	if max < min {
		max = min
	}
	if max == min {
		return min
	}
	return min + rand.Intn(max-min+1)
}

type InventorySlot struct {
	ItemID string `json:"item_id"`
	Count  int    `json:"count"`
}

type EarnedItemEntry struct {
	Name  string
	Count int
}

func addEarnedItem(list []EarnedItemEntry, name string, qty int) []EarnedItemEntry {
	for i := range list {
		if list[i].Name == name {
			list[i].Count += qty
			return list
		}
	}
	return append(list, EarnedItemEntry{Name: name, Count: qty})
}

func (g *Game) AddItem(itemID string, qty int) {
	if qty <= 0 {
		return
	}
	for i := range g.Inventory {
		if g.Inventory[i].ItemID == itemID {
			g.Inventory[i].Count += qty
			return
		}
	}
	g.Inventory = append(g.Inventory, InventorySlot{ItemID: itemID, Count: qty})
}

func (g *Game) ConsumeItem(itemID string, qty int) bool {
	for i := range g.Inventory {
		if g.Inventory[i].ItemID != itemID {
			continue
		}
		if g.Inventory[i].Count < qty {
			return false
		}
		g.Inventory[i].Count -= qty
		if g.Inventory[i].Count <= 0 {
			g.Inventory = append(g.Inventory[:i], g.Inventory[i+1:]...)
		}
		return true
	}
	return false
}

func (g *Game) ItemCount(itemID string) int {
	for _, slot := range g.Inventory {
		if slot.ItemID == itemID {
			return slot.Count
		}
	}
	return 0
}

func applyItemEffect(g *Game, def ItemDef, target int) (hpHealed int, mpHealed int, applied bool) {
	if target < 0 || target >= partySize {
		return 0, 0, false
	}

	isDead := g.PlayerHP[target] <= 0

	if isDead && !def.Revive {
		if def.CureDebuff {
			return 0, 0, true
		}
		return 0, 0, false
	}

	if !isDead && def.Revive {
		return 0, 0, false
	}

	if isDead && def.Revive {
		amt := max(def.HealHP+g.PlayerMaxHP[target]*def.HealHPPercent/100, 1)
		g.PlayerHP[target] = min(amt, g.PlayerMaxHP[target])
		return g.PlayerHP[target], 0, true
	}

	if def.HealHP > 0 || def.HealHPPercent > 0 {
		amt := def.HealHP + g.PlayerMaxHP[target]*def.HealHPPercent/100
		if amt > 0 {
			before := g.PlayerHP[target]
			g.PlayerHP[target] += amt
			if g.PlayerHP[target] > g.PlayerMaxHP[target] {
				g.PlayerHP[target] = g.PlayerMaxHP[target]
			}
			hpHealed = g.PlayerHP[target] - before
		}
	}
	if def.HealMP > 0 || def.HealMPPercent > 0 {
		amt := def.HealMP + g.PlayerMaxMP[target]*def.HealMPPercent/100
		if amt > 0 {
			before := g.PlayerMP[target]
			g.PlayerMP[target] += amt
			if g.PlayerMP[target] > g.PlayerMaxMP[target] {
				g.PlayerMP[target] = g.PlayerMaxMP[target]
			}
			mpHealed = g.PlayerMP[target] - before
		}
	}

	applied = hpHealed > 0 || mpHealed > 0 || def.CureDebuff
	return hpHealed, mpHealed, applied
}
