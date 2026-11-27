package main

import "testing"

func TestBattleSubPanelTextPoint(t *testing.T) {
	x, y := battleSubPanelTextPoint(800, 400, 2)
	if x != 800 {
		t.Fatalf("text x = %v, want 800", x)
	}
	if y != 492 {
		t.Fatalf("text y = %v, want 492", y)
	}
}

func TestBattleSubPanelMPAndLevelPoints(t *testing.T) {
	mpX, mpY := battleSubPanelMPPoint(800, 400, 1)
	if mpX != 960 || mpY != 457 {
		t.Fatalf("MP point = (%v, %v), want (960, 457)", mpX, mpY)
	}

	levelX, levelY := battleSubPanelLevelPoint(800, 400, 1)
	if levelX != 917 || levelY != 457 {
		t.Fatalf("Lv point = (%v, %v), want (917, 457)", levelX, levelY)
	}
}
