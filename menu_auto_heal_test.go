package main

import "testing"

func newAutoHealTestGame() *Game {
	g := &Game{}
	for i := range partySize {
		g.PlayerLv[i] = 1
		g.PlayerMaxHP[i] = 100
		g.PlayerHP[i] = 100
		g.PlayerMagicAtk[i] = 10
	}
	return g
}

func TestAutoHealPartyHealsLivingMembersToFull(t *testing.T) {
	g := newAutoHealTestGame()
	g.PlayerHP = [4]int{50, 100, 0, 90}
	g.PlayerMP[1] = 100 // slot1 (白髪女) has スーパーヒーリング from Lv1

	if !g.AutoHealParty() {
		t.Fatal("AutoHealParty() = false, want true")
	}
	if want := [4]int{100, 100, 0, 100}; g.PlayerHP != want {
		t.Fatalf("HP after auto heal = %v, want %v (dead member stays down)", g.PlayerHP, want)
	}
	if g.PlayerMP[1] >= 100 {
		t.Fatalf("caster MP = %d, want it spent", g.PlayerMP[1])
	}
}

func TestAutoHealPartyStopsWhenOutOfMP(t *testing.T) {
	g := newAutoHealTestGame()
	g.PlayerHP[0] = 10
	g.PlayerMP[1] = 7 // exactly one Lv1 cast

	if !g.AutoHealParty() {
		t.Fatal("AutoHealParty() = false, want true")
	}
	if g.PlayerMP[1] != 0 {
		t.Fatalf("caster MP = %d, want 0", g.PlayerMP[1])
	}
	if g.PlayerHP[0] <= 10 || g.PlayerHP[0] >= 100 {
		t.Fatalf("HP = %d, want one partial heal", g.PlayerHP[0])
	}
}

func TestAutoHealPartyNothingToDo(t *testing.T) {
	g := newAutoHealTestGame()
	g.PlayerMP[1] = 100
	if g.AutoHealParty() {
		t.Fatal("full HP party: AutoHealParty() = true, want false")
	}
	g.PlayerHP[0] = 10
	g.PlayerMP[1] = 0
	if g.AutoHealParty() {
		t.Fatal("no MP: AutoHealParty() = true, want false")
	}
}
