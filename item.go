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

	Revive     bool
	CureDebuff bool

	UsableInField  bool
	UsableInBattle bool
}

var ItemDatabase = []ItemDef{
	{
		ID:             "potion",
		Name:           "ポーション",
		Description:    "対象のHPを50回復する",
		Target:         TargetSingle,
		HealHP:         50,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "hi_potion",
		Name:           "ハイポーション",
		Description:    "対象のHPを150回復する",
		Target:         TargetSingle,
		HealHP:         150,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "ether",
		Name:           "エーテル",
		Description:    "対象のMPを30回復する",
		Target:         TargetSingle,
		HealMP:         30,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "elixir",
		Name:           "エリクサー",
		Description:    "対象のHP・MPを全回復する",
		Target:         TargetSingle,
		HealHPPercent:  100,
		HealMPPercent:  100,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "phoenix_down",
		Name:           "フェニックスの尾",
		Description:    "戦闘不能を回復し、HPを少し回復する",
		Target:         TargetSingle,
		HealHPPercent:  30,
		Revive:         true,
		UsableInField:  true,
		UsableInBattle: true,
	},
	{
		ID:             "antidote",
		Name:           "万能薬",
		Description:    "対象の弱体効果をすべて解除する",
		Target:         TargetSingle,
		CureDebuff:     true,
		UsableInField:  true,
		UsableInBattle: true,
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
		amt := def.HealHP + g.PlayerMaxHP[target]*def.HealHPPercent/100
		if amt < 1 {
			amt = 1
		}
		g.PlayerHP[target] = amt
		if g.PlayerHP[target] > g.PlayerMaxHP[target] {
			g.PlayerHP[target] = g.PlayerMaxHP[target]
		}
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
