package main

import "testing"

func TestElementalDamageMultiplier(t *testing.T) {
	resistances := [elementalTypeCount]int{30, 0, 0, 0}
	if got := elementalDamageMultiplier(ElemFire, resistances); got != 0.7 {
		t.Fatalf("fire resistance 30 should produce 0.7x damage, got %v", got)
	}

	resistances[0] = -30
	if got := elementalDamageMultiplier(ElemFire, resistances); got != 1.3 {
		t.Fatalf("fire weakness -30 should produce 1.3x damage, got %v", got)
	}

	if got := elementalDamageMultiplier(ElemPhysicalNone, resistances); got != 1.0 {
		t.Fatalf("non-elemental damage should produce 1.0x damage, got %v", got)
	}
}

func TestEnemyDamageDoesNotCrit(t *testing.T) {
	s := &BattleScene{}
	for range 20 {
		s.rollDamageNoCrit(100, 100, 100, 1, 1000)
		if s.lastRollWasCrit {
			t.Fatal("enemy damage should not roll critical hits")
		}
	}
}
