package main

import "testing"

func TestBattleCanUseHPItemAtFullHealth(t *testing.T) {
	g := &Game{}
	for i := range partySize {
		g.PlayerHP[i] = 100
		g.PlayerMaxHP[i] = 100
	}
	potion, ok := GetItemDef("potion")
	if !ok {
		t.Fatal("GetItemDef(potion) failed")
	}

	scene := &BattleScene{game: g}
	if !scene.applyItemToTargetInBattle(potion, 0) {
		t.Fatal("battle HP item at full health = unusable, want usable")
	}
	if g.PlayerHP[0] != 100 {
		t.Fatalf("HP after full-health item use = %d, want 100", g.PlayerHP[0])
	}
	if len(scene.damagePops) != 1 || scene.damagePops[0].Value != 50 {
		t.Fatalf("full-health potion popups = %+v, want one popup with value 50", scene.damagePops)
	}

	if _, _, applied := applyItemEffect(g, potion, 0); applied {
		t.Fatal("field item at full health = applied, want false")
	}
}

func TestBattleItemHPNumberShowsConfiguredAmount(t *testing.T) {
	tests := []struct {
		name        string
		itemID      string
		hp          int
		maxHP       int
		wantDisplay int
		wantHP      int
	}{
		{"partially healed", "potion", 95, 100, 50, 100},
		{"percent heal", "elixir", 90, 100, 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Game{}
			g.PlayerHP[0], g.PlayerMaxHP[0] = tt.hp, tt.maxHP
			def, ok := GetItemDef(tt.itemID)
			if !ok {
				t.Fatalf("GetItemDef(%q) failed", tt.itemID)
			}
			scene := &BattleScene{game: g}
			if !scene.applyItemToTargetInBattle(def, 0) {
				t.Fatal("item use failed")
			}
			if g.PlayerHP[0] != tt.wantHP {
				t.Errorf("HP = %d, want %d", g.PlayerHP[0], tt.wantHP)
			}
			if len(scene.damagePops) != 1 || scene.damagePops[0].Value != tt.wantDisplay {
				t.Errorf("HP popup = %+v, want one popup with value %d", scene.damagePops, tt.wantDisplay)
			}
		})
	}
}

func TestBattleFullHealthItemDoesNotAllowInvalidTargets(t *testing.T) {
	g := &Game{}
	g.PlayerHP[0] = 100
	g.PlayerMaxHP[0] = 100
	revive, ok := GetItemDef("phoenix_down")
	if !ok {
		t.Fatal("GetItemDef(phoenix_down) failed")
	}

	scene := &BattleScene{game: g}
	if scene.applyItemToTargetInBattle(revive, 0) {
		t.Fatal("revive item on living target = usable, want false")
	}
	g.PlayerHP[0] = 0
	potion, ok := GetItemDef("potion")
	if !ok {
		t.Fatal("GetItemDef(potion) failed")
	}
	if scene.applyItemToTargetInBattle(potion, 0) {
		t.Fatal("HP item on dead target = usable, want false")
	}
}

func TestItemDatabaseMatchesReplacementList(t *testing.T) {
	want := []struct {
		name   string
		target SkillTarget
		gp     int
		ct     float64
		field  bool
		battle bool
	}{
		{"ばんそうこう", TargetSingle, 1, 50, true, true},
		{"すごいばんそうこう", TargetSingle, 1, 50, true, true},
		{"伝説のばんそうこう", TargetSingle, 1, 50, true, true},
		{"エナジードリンク", TargetSingle, 1, 50, true, true},
		{"すごいエナジードリンク", TargetSingle, 1, 50, true, true},
		{"伝説のエナジードリンク", TargetSingle, 1, 50, true, true},
		{"ばんそこドリンク", TargetSingle, 1, 50, true, true},
		{"ファイナルばんそこドリンク", TargetAll, 3, 10, true, true},
		{"不死鳥の息吹缶", TargetSingle, 1, 50, true, true},
		{"タイマーブースター", TargetOther, 6, 50, false, true},
		{"けいけんブースター", TargetAll, 1, 50, false, true},
		{"さよならえんまく", TargetOther, 0, 0, false, true},
		{"スピードブースター", TargetSingle, 1, 50, false, true},
	}
	if len(ItemDatabase) != len(want) {
		t.Fatalf("item count = %d, want %d", len(ItemDatabase), len(want))
	}
	for i, expected := range want {
		got := ItemDatabase[i]
		if got.Name != expected.name || got.Target != expected.target ||
			got.GaugePoint != expected.gp || got.ReturnPosition != expected.ct ||
			got.UsableInField != expected.field || got.UsableInBattle != expected.battle {
			t.Errorf("item %d = {%q target=%d GP=%d CT=%v field=%t battle=%t}, want {%q target=%d GP=%d CT=%v field=%t battle=%t}",
				i, got.Name, got.Target, got.GaugePoint, got.ReturnPosition, got.UsableInField, got.UsableInBattle,
				expected.name, expected.target, expected.gp, expected.ct, expected.field, expected.battle)
		}
	}
}

func TestItemRecoveryEffects(t *testing.T) {
	g := &Game{}
	g.PlayerHP[0], g.PlayerMaxHP[0] = 10, 400
	g.PlayerMP[0], g.PlayerMaxMP[0] = 0, 80

	bandage, _ := GetItemDef("legendary_bandage")
	hp, _, applied := applyItemEffect(g, bandage, 0)
	if !applied || hp != 300 || g.PlayerHP[0] != 310 {
		t.Fatalf("legendary bandage HP = %d (applied=%t, total=%d), want 300 and 310", hp, applied, g.PlayerHP[0])
	}

	energy, _ := GetItemDef("legendary_energy_drink")
	_, mp, applied := applyItemEffect(g, energy, 0)
	if !applied || mp != 50 || g.PlayerMP[0] != 50 {
		t.Fatalf("legendary energy drink MP = %d (applied=%t, total=%d), want 50 and 50", mp, applied, g.PlayerMP[0])
	}

	revive, _ := GetItemDef("phoenix_down")
	g.PlayerHP[1], g.PlayerMaxHP[1] = 0, 200
	hp, _, applied = applyItemEffect(g, revive, 1)
	if !applied || hp != 50 || g.PlayerHP[1] != 50 {
		t.Fatalf("phoenix revival HP = %d (applied=%t, total=%d), want 50 and 50", hp, applied, g.PlayerHP[1])
	}
}

func TestBattleItemGaugeAndReturnPosition(t *testing.T) {
	g := &Game{Inventory: []InventorySlot{{ItemID: "timer_booster", Count: 1}}}
	scene := &BattleScene{
		game:          g,
		waitingActor:  0,
		gaugeUnlocked: true,
		enemies:       []EnemyUnit{{HP: 1}},
	}
	timerBooster, _ := GetItemDef("timer_booster")
	scene.finishUsingBattleItem(timerBooster, -1)

	if scene.gaugePoint != 6 {
		t.Errorf("gauge points = %d, want 6", scene.gaugePoint)
	}
	if scene.atbGauge[0] != 50 {
		t.Errorf("item return position = %v, want 50", scene.atbGauge[0])
	}
	if g.ItemCount("timer_booster") != 0 {
		t.Errorf("timer boosters remaining = %d, want 0", g.ItemCount("timer_booster"))
	}
}

func TestBattleItemListOnlyCountsRowsVisibleOnScreen(t *testing.T) {
	const panelTop, panelBottom = 297.0, float64(gameHeight)
	rows := visibleBattleItemRowCount(len(ItemDatabase), panelTop, panelBottom)
	if rows != 6 {
		t.Fatalf("visible battle item rows = %d, want 6", rows)
	}
	lastItemIndex := len(ItemDatabase) - 1
	start := max(0, lastItemIndex-rows+1)
	if start+rows != len(ItemDatabase) {
		t.Fatalf("visible range [%d, %d) does not include last item", start, start+rows)
	}
}

func TestSpeedBoosterReducesNextTwoActionReturnPositions(t *testing.T) {
	g := &Game{Inventory: []InventorySlot{{ItemID: "speed_booster", Count: 1}}}
	scene := &BattleScene{
		game:         g,
		waitingActor: 0,
		enemies:      []EnemyUnit{{HP: 1}},
	}
	speedBooster, _ := GetItemDef("speed_booster")
	scene.finishUsingBattleItem(speedBooster, 0)

	if scene.atbGauge[0] != 50 {
		t.Fatalf("speed booster action return position = %v, want 50", scene.atbGauge[0])
	}
	if scene.nextActionCTReductions[0] != 2 {
		t.Fatalf("remaining CT reductions = %d, want 2", scene.nextActionCTReductions[0])
	}
	scene.resetPlayerGaugeTo(0, 50)
	if scene.atbGauge[0] != 25 || scene.nextActionCTReductions[0] != 1 {
		t.Errorf("first boosted return = %v, reductions left = %d; want 25 and 1", scene.atbGauge[0], scene.nextActionCTReductions[0])
	}
	scene.resetPlayerGaugeTo(0, 50)
	if scene.atbGauge[0] != 25 || scene.nextActionCTReductions[0] != 0 {
		t.Errorf("second boosted return = %v, reductions left = %d; want 25 and 0", scene.atbGauge[0], scene.nextActionCTReductions[0])
	}
}

func TestExperienceBoosterDoublesOnlyNonBossRewards(t *testing.T) {
	scene := &BattleScene{
		enemies:        []EnemyUnit{{Exp: 40}, {Exp: 10}},
		expBoostActive: true,
	}
	if got := scene.totalBattleExp(); got != 100 {
		t.Errorf("boosted EXP = %d, want 100", got)
	}
	scene.enemyType = "boss_1"
	if got := scene.totalBattleExp(); got != 50 {
		t.Errorf("boss EXP = %d, want 50", got)
	}
}

func TestExperienceBoosterActivatesForBattle(t *testing.T) {
	scene := &BattleScene{game: &Game{}}
	booster, _ := GetItemDef("exp_booster")
	if !scene.applyItemToTargetInBattle(booster, 0) || !scene.expBoostActive {
		t.Fatal("experience booster did not activate")
	}
	scene.enemyType = "boss_1"
	scene.expBoostActive = false
	if scene.applyItemToTargetInBattle(booster, 0) || scene.expBoostActive {
		t.Fatal("experience booster activated in a boss battle")
	}
}

func TestBossBattleHidesDisabledItems(t *testing.T) {
	g := &Game{Inventory: []InventorySlot{
		{ItemID: "exp_booster", Count: 1},
		{ItemID: "smoke_bomb", Count: 1},
	}}
	scene := &BattleScene{game: g, enemyType: "boss_1"}
	if got := len(scene.battleUsableItems()); got != 0 {
		t.Errorf("boss-usable items = %d, want 0", got)
	}
	scene.enemyType = "enemy_1"
	if got := len(scene.battleUsableItems()); got != 2 {
		t.Errorf("normal battle-usable items = %d, want 2", got)
	}
}

func TestSmokeBombGuaranteesBattleEscape(t *testing.T) {
	g := &Game{Inventory: []InventorySlot{{ItemID: "smoke_bomb", Count: 1}}}
	scene := &BattleScene{
		game:         g,
		waitingActor: 0,
		enemies:      []EnemyUnit{{HP: 1}},
	}
	smokeBomb, _ := GetItemDef("smoke_bomb")
	scene.useBattleItemWithoutTarget(smokeBomb)

	if !scene.fleeSucceeded || scene.battlePhase != phaseMessage {
		t.Fatalf("escape state = (flee=%t, phase=%d), want true and phaseMessage", scene.fleeSucceeded, scene.battlePhase)
	}
	if g.ItemCount("smoke_bomb") != 0 {
		t.Errorf("smoke bombs remaining = %d, want 0", g.ItemCount("smoke_bomb"))
	}
}
