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

func TestSkillHealUsesMagicAttackMultiplier(t *testing.T) {
	g := &Game{}
	g.PlayerMagicAtk[0] = 100
	g.PlayerMagicAtk[1] = 100
	s := &BattleScene{game: g}

	if got := s.rollSkillHeal(0, 4, 1, false); got != 50 {
		t.Errorf("ヒールウィンド single heal = %d, want 50", got)
	}
	if got := s.rollSkillHeal(0, 4, 1, true); got != 30 {
		t.Errorf("ヒールウィンド all heal = %d, want 30", got)
	}
	if got := s.rollSkillHeal(1, 0, 1, false); got != 70 {
		t.Errorf("スーパーヒーリング single heal = %d, want 70", got)
	}
}

func TestCommandGlowFramesBySkillLevel(t *testing.T) {
	tests := []struct {
		level int
		want  []int
	}{
		{level: 0, want: []int{0}},
		{level: 1, want: []int{1}},
		{level: 2, want: []int{1, 2, 3}},
		{level: 3, want: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}},
	}
	for _, tt := range tests {
		for i, want := range tt.want {
			got := glowFrameForLevel(tt.level, float64(i)*battleSpriteFrameDur, 10)
			if got != want {
				t.Errorf("level %d, frame %d: got %d, want %d", tt.level, i, got, want)
			}
		}
	}
	if got := glowFrameForLevel(3, 0, 14); got != 1 {
		t.Errorf("level 3 starts on second frame, got %d", got)
	}
	if got := glowFrameForLevel(3, 12*battleSpriteFrameDur, 14); got != 13 {
		t.Errorf("level 3 includes the last command frame, got %d", got)
	}
}
