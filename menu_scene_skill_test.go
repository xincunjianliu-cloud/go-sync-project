package main

import "testing"

func TestMenuHealCanApplyOnlyWhenHPIsMissing(t *testing.T) {
	g := &Game{}
	for i := range partySize {
		g.PlayerHP[i] = 100
		g.PlayerMaxHP[i] = 100
	}

	if g.menuHealCanApply(0, 50) {
		t.Fatal("full-HP target: menuHealCanApply() = true, want false")
	}
	if g.menuHealCanApply(partySize, 50) {
		t.Fatal("full-HP party: menuHealCanApply() = true, want false")
	}

	g.PlayerHP[1] = 80
	if !g.menuHealCanApply(1, 50) {
		t.Fatal("injured target: menuHealCanApply() = false, want true")
	}
	if !g.menuHealCanApply(partySize, 50) {
		t.Fatal("injured party member: all-target menuHealCanApply() = false, want true")
	}

	g.PlayerHP[1] = 0
	if g.menuHealCanApply(1, 50) {
		t.Fatal("dead target: menuHealCanApply() = true, want false")
	}
	if g.menuHealCanApply(partySize, 0) {
		t.Fatal("zero heal amount: menuHealCanApply() = true, want false")
	}
}
