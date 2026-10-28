package main

import "testing"

func TestRewindSnapshotRestoresWholePartyAndAttackEffects(t *testing.T) {
	g := &Game{
		PlayerHP:    [partySize]int{70, 60, 50, 40},
		PlayerMaxHP: [partySize]int{100, 100, 100, 100},
	}
	s := &BattleScene{
		game:           g,
		PlayerBuffs:    [partySize][]Buff{{{Type: StatDef, Percent: 20, Seconds: 5}}},
		PlayerDebuffs:  [partySize][]Debuff{{{Type: StatAtk, Percent: 10, Seconds: 4}}},
		coverCount:     [partySize]int{1, 2, 0, 0},
		counterTimer:   [partySize]float64{3, 0, 0, 0},
		guardCount:     [partySize]int{1, 0, 0, 0},
		waitStance:     [partySize]bool{true, true, false, false},
		waitOrder:      []int{0, 1},
		waitCancelHold: [partySize]float64{0, 0, 0, 0},
	}
	s.atbGauge[0], s.atbGauge[1] = 100, 100
	s.captureRewindSnapshot()
	s.rewindSnapshot.valid = true

	g.PlayerHP = [partySize]int{0, 10, 1, 1}
	s.PlayerBuffs[0] = nil
	s.PlayerDebuffs[0] = nil
	s.atbGauge[0], s.atbGauge[1] = 0, 20
	s.coverCount = [partySize]int{}
	s.counterTimer = [partySize]float64{}
	s.guardCount = [partySize]int{}
	s.waitStance = [partySize]bool{}
	s.waitOrder = nil

	s.restoreRewindSnapshot()

	if got, want := g.PlayerHP, [partySize]int{70, 60, 50, 40}; got != want {
		t.Fatalf("party HP = %v, want %v", got, want)
	}
	if got, want := s.atbGauge[:partySize], []float64{100, 100, 0, 0}; !equalFloatSlices(got, want) {
		t.Fatalf("party ATB = %v, want %v", got, want)
	}
	if len(s.PlayerBuffs[0]) != 1 || len(s.PlayerDebuffs[0]) != 1 {
		t.Fatalf("buffs/debuffs were not restored: %v / %v", s.PlayerBuffs[0], s.PlayerDebuffs[0])
	}
	if s.coverCount != [partySize]int{1, 2, 0, 0} ||
		s.counterTimer != [partySize]float64{3, 0, 0, 0} ||
		s.guardCount != [partySize]int{1, 0, 0, 0} {
		t.Fatal("cover, counter, or guard state was not restored")
	}
	if !s.waitStance[0] || !s.waitStance[1] || len(s.waitOrder) != 2 {
		t.Fatal("wait stance was not restored")
	}
}

func equalFloatSlices(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
