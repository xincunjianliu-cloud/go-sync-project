package main

import "testing"

func TestCharacterSkillSetsSortedByUnlockLevel(t *testing.T) {
	g := &Game{}
	for c := 0; c < partySize; c++ {
		skills := g.CharacterSkills(c)
		if len(skills) == 0 || len(skills) > len(g.PlayerSkillLv[c]) {
			t.Fatalf("char %d has %d skills, want 1..%d", c, len(skills), len(g.PlayerSkillLv[c]))
		}
		for i := 1; i < len(skills); i++ {
			if skills[i].UnlockLevel < skills[i-1].UnlockLevel {
				t.Errorf("char %d: %s (Lv%d) listed after %s (Lv%d)", c,
					skills[i].Name, skills[i].UnlockLevel, skills[i-1].Name, skills[i-1].UnlockLevel)
			}
		}
		for _, sk := range skills {
			if len(sk.Levels) != MaxSkillLevel {
				t.Errorf("char %d: %s has %d levels, want %d", c, sk.Name, len(sk.Levels), MaxSkillLevel)
			}
			for lv := 2; lv <= len(sk.Levels); lv++ {
				if sk.Levels[lv-1].UpgradeSP <= 0 {
					t.Errorf("char %d: %s Lv%d has no UpgradeSP", c, sk.Name, lv)
				}
			}
		}
	}
}

func TestUpgradeSkillStatBonusSurvivesLevelUp(t *testing.T) {
	g := &Game{}
	g.PlayerLv[0] = 1
	g.ApplyLevelUpGrowth(0)
	for j := range g.PlayerSkillLv[0] {
		g.PlayerSkillLv[0][j] = 1
	}
	baseAtk := g.PlayerAtk[0]

	// Player1Skills[0] is スマッシュ: Lv2 costs 100 SP and gives 物攻+20.
	g.PlayerSP[0] = 100
	if !g.UpgradeSkill(0, 0) {
		t.Fatalf("UpgradeSkill failed with exactly enough SP")
	}
	if g.PlayerSP[0] != 0 || g.PlayerAtk[0] != baseAtk+20 {
		t.Fatalf("after upgrade SP=%d Atk=%d, want 0 and %d", g.PlayerSP[0], g.PlayerAtk[0], baseAtk+20)
	}
	if g.CanUpgradeSkill(0, 0) {
		t.Fatalf("CanUpgradeSkill true with 0 SP (Lv3 costs 300)")
	}

	g.PlayerLv[0] = 2
	g.ApplyLevelUpGrowth(0)
	want := PlayerStatsByLevel[1][0].PhysAtk + 20
	if g.PlayerAtk[0] != want {
		t.Fatalf("Atk after level up = %d, want %d (table + skill bonus)", g.PlayerAtk[0], want)
	}
}

func TestTimedAndCoverBuffTicking(t *testing.T) {
	list := []Buff{
		{Type: StatDef, Percent: 20, Seconds: 5},
		{Type: StatMdf, Percent: 30, Cover: true},
	}
	list = TickBuffSeconds(list, 4.9)
	if len(list) != 2 {
		t.Fatalf("timed buff expired early")
	}
	list = TickBuffSeconds(list, 0.2)
	if len(list) != 1 || !list[0].Cover {
		t.Fatalf("after 5.1s want only the cover buff, got %+v", list)
	}
	if len(withoutCoverBuffs(list)) != 0 {
		t.Fatalf("withoutCoverBuffs kept a cover buff")
	}
}

func TestCoverRedirectAndGuard(t *testing.T) {
	g := &Game{}
	for i := 0; i < partySize; i++ {
		g.PlayerHP[i] = 100
		g.PlayerMaxHP[i] = 100
	}
	s := &BattleScene{game: g}
	s.coverCount[2] = 1

	if got := s.coverRedirect(0); got != 2 {
		t.Fatalf("coverRedirect(0) = %d, want 2", got)
	}
	if got := s.coverRedirect(0); got != 0 {
		t.Fatalf("coverRedirect(0) after charges ran out = %d, want 0", got)
	}

	s.guardHPPercent[1] = 10
	s.guardCount[1] = 1
	if s.tryGuard(1, 50) {
		t.Fatalf("guard fired on a hit leaving 50%% HP")
	}
	if !s.tryGuard(1, 95) {
		t.Fatalf("guard did not fire on a hit leaving 5%% HP")
	}
	if s.tryGuard(1, 95) {
		t.Fatalf("guard fired again with no charges left")
	}
}

func TestApplySkillEffectsBothStats(t *testing.T) {
	g := &Game{}
	s := &BattleScene{game: g, enemies: []EnemyUnit{{HP: 10, MaxHP: 10, Def: 100, MagicDef: 100}}}
	s.applySkillEffects([]SkillEffect{{Type: EffectDebuffDefBoth, Percent: 30, PercentAll: 10, Seconds: 30}}, 0, true, 0, true)
	if got := s.effectiveEnemyDef(0, false); got != 90 {
		t.Fatalf("physical def = %d, want 90 (PercentAll used for all-target)", got)
	}
	if got := s.effectiveEnemyDef(0, true); got != 90 {
		t.Fatalf("magic def = %d, want 90", got)
	}

	g.PlayerAtk[1], g.PlayerMagicAtk[1] = 100, 100
	s.applySkillEffects([]SkillEffect{{Type: EffectBuffPhysAtkUp, Percent: 20, Seconds: 20}}, 0, false, 1, false)
	if s.effectiveAtk(1) != 120 || s.effectiveMagicAtk(1) != 100 {
		t.Fatalf("物攻 buff: atk=%d mat=%d, want 120/100", s.effectiveAtk(1), s.effectiveMagicAtk(1))
	}
}
