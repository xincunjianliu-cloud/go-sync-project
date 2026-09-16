package main

import "testing"

func TestTryWaitSynergyLethalKeepsBattleEndPhase(t *testing.T) {
	g := &Game{}
	for i := 0; i < partySize; i++ {
		g.PlayerHP[i] = 100
		g.PlayerMaxHP[i] = 100
		g.PlayerAtk[i] = 999
		g.PlayerNextEXP[i] = 999999
	}

	s := &BattleScene{
		game:         g,
		enemyType:    "normal",
		enemies:      []EnemyUnit{{HP: 1, MaxHP: 1}},
		gaugePoint:   gaugePoolMax,
		waitingActor: -1,
		battlePhase:  phasePlayerMenu,
	}
	for i := 0; i < partySize; i++ {
		s.waitStance[i] = true
	}

	fired := s.tryWaitSynergy()
	if !fired {
		t.Fatalf("tryWaitSynergy() = false, want true when all party members are in wait stance")
	}
	if s.enemies[0].HP != 0 {
		t.Fatalf("enemies[0].HP = %d, want 0 after lethal synergy attack", s.enemies[0].HP)
	}
	if s.battlePhase != phaseBattleEnd {
		t.Fatalf("battlePhase = %d, want phaseBattleEnd (%d) after synergy attack defeats the enemy", s.battlePhase, phaseBattleEnd)
	}

	if !fired {
		s.waitingActor = -1
		s.battlePhase = phaseATB
	}
	if s.battlePhase != phaseBattleEnd {
		t.Fatalf("caller incorrectly reverted battlePhase to %d", s.battlePhase)
	}
}
