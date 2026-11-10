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

	if _, _, applied := applyItemEffect(g, potion, 0); applied {
		t.Fatal("field item at full health = applied, want false")
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
