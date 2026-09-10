package main

import "math/rand"

// item.go: アイテム関連のデータと処理をまとめたファイル
// スキル(skill.go)と同じ考え方で、アイテムの効果はすべてここに定義したデータで
// 調整できるようにしてある（数値を変えたい・新アイテムを増やしたい場合はここを編集する）。

// ItemDef は1種類のアイテムが持つデータ一式。
type ItemDef struct {
	ID          string // インベントリ・ドロップテーブルで使う一意なID
	Name        string
	Description string

	Target SkillTarget // TargetSingle / TargetAll / TargetBoth（使用時に単体/全体を選べる）

	HealHP        int // 固定回復量（HP）
	HealHPPercent int // 最大HPに対する割合回復（%）。HealHPと合算される
	HealMP        int // 固定回復量（MP）
	HealMPPercent int // 最大MPに対する割合回復（%）。HealMPと合算される

	Revive     bool // trueなら戦闘不能のキャラにも使用でき、HealHP(Percent)分のHPで復活する
	CureDebuff bool // trueなら対象の弱体効果(デバフ)をすべて解除する

	UsableInField  bool // メニュー画面（フィールド）から使用できるか
	UsableInBattle bool // バトル中に使用できるか
}

// ItemDatabase：全アイテムの定義一覧。ここを直接編集すれば内容を調整できる。
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

// GetItemDef はIDからアイテム定義を探す。見つからなければ ok=false。
func GetItemDef(id string) (ItemDef, bool) {
	for _, it := range ItemDatabase {
		if it.ID == id {
			return it, true
		}
	}
	return ItemDef{}, false
}

// ItemDrop は敵撃破時のアイテムドロップ抽選テーブルの1エントリ。
// Percent(%)の確率で独立判定するので、1体の敵に複数のドロップを設定できる。
// MinCount/MaxCountは抽選成功時に得られる個数の範囲（両方0なら1個として扱う）。
type ItemDrop struct {
	ItemID   string
	Percent  int
	MinCount int
	MaxCount int
}

// rollCount はItemDropの個数範囲からランダムに1つ選ぶ。未設定(0,0)なら1を返す。
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

// InventorySlot は所持アイテム1種類分（ID＋個数）。
type InventorySlot struct {
	ItemID string `json:"item_id"`
	Count  int    `json:"count"`
}

// EarnedItemEntry は戦闘勝利時に入手したアイテム1種類分（リザルト画面表示用）。
// 同じアイテムが複数エントリ・複数回ドロップしても、表示時は名前ごとに個数をまとめる。
type EarnedItemEntry struct {
	Name  string
	Count int
}

// addEarnedItem はearnedItemsに1種類分の入手アイテムを加算する（同名があれば個数をまとめる）。
func addEarnedItem(list []EarnedItemEntry, name string, qty int) []EarnedItemEntry {
	for i := range list {
		if list[i].Name == name {
			list[i].Count += qty
			return list
		}
	}
	return append(list, EarnedItemEntry{Name: name, Count: qty})
}

// AddItem は指定アイテムを所持数に加算する（新規なら末尾に追加）。
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

// ConsumeItem は指定アイテムをqty個消費する。所持数が足りなければ何もせずfalseを返す。
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

// ItemCount は指定アイテムの所持数を返す（未所持なら0）。
func (g *Game) ItemCount(itemID string) int {
	for _, slot := range g.Inventory {
		if slot.ItemID == itemID {
			return slot.Count
		}
	}
	return 0
}

// applyItemEffect はアイテムの効果を対象1人に適用する。
// 戻り値：実際に回復したHP量・MP量、および何らかの効果があった(=消費すべき)かどうか。
// 戦闘不能の対象には、Revive付きのアイテム以外は効果がない(applied=false)。
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
