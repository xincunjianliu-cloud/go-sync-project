package main

import "testing"

func TestBattleSubPanelTextPoint(t *testing.T) {
	x, y := battleSubPanelTextPoint(800, 400, 2)
	if x != 807 {
		t.Fatalf("text x = %v, want 807", x)
	}
	if y != 488 {
		t.Fatalf("text y = %v, want 488", y)
	}
}

func TestBattleSubPanelMPAndLevelPoints(t *testing.T) {
	mpX, mpY := battleSubPanelMPPoint(800, 400, 1)
	if mpX != 960 || mpY != 453 {
		t.Fatalf("MP point = (%v, %v), want (960, 453)", mpX, mpY)
	}

	levelX, levelY := battleSubPanelLevelPoint(800, 400, 1)
	if levelX != 917 || levelY != 453 {
		t.Fatalf("Lv point = (%v, %v), want (917, 453)", levelX, levelY)
	}
}
