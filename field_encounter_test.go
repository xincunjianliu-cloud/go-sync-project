package main

import "testing"

func TestEncounterEffectStartsBattleAfterAnimation(t *testing.T) {
	s := &FieldScene{}
	started := 0
	s.startEncounterEffect(func() {
		started++
	})

	s.Update(encounterEffectDuration - 0.01)
	if started != 0 {
		t.Fatal("battle started before the encounter effect finished")
	}

	s.Update(0.02)
	if started != 1 {
		t.Fatalf("battle start callback ran %d times, want once", started)
	}

	s.Update(0.02)
	if started != 1 {
		t.Fatalf("battle start callback ran %d times after completion, want once", started)
	}
}

func TestBossBattleCommandStartsEncounterEffect(t *testing.T) {
	s := &FieldScene{game: &Game{}}
	cmd := EventCommand{Speaker: systemSpeaker, Text: cmdStartBattlePrefix + "boss_1"}

	if !s.runSceneChangeCommand(cmd) {
		t.Fatal("boss battle command was not handled")
	}
	if !s.encounterEffectActive || s.pendingBattleStart == nil {
		t.Fatal("boss battle command did not start the encounter effect")
	}
	if s.game.fadeMode != FadeNone {
		t.Fatal("boss battle transition started before the encounter effect")
	}
}
