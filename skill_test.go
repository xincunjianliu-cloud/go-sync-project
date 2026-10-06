package main

import (
	"slices"
	"testing"
)

// スキルは技表の番号順に並べる（解放レベル順ではない）。古いセーブの
// 変換に使う legacySkillOrder と、名前の組み合わせが一致していること。
func TestCharacterSkillSetsValid(t *testing.T) {
	g := &Game{}
	for c := 0; c < partySize; c++ {
		skills := g.CharacterSkills(c)
		if len(skills) == 0 || len(skills) > len(g.PlayerSkillLv[c]) {
			t.Fatalf("char %d has %d skills, want 1..%d", c, len(skills), len(g.PlayerSkillLv[c]))
		}
		if got, want := len(legacySkillOrder[c]), len(skills); got != want {
			t.Errorf("char %d: legacySkillOrder has %d skills, want %d", c, got, want)
		}
		for _, name := range legacySkillOrder[c] {
			if !slices.ContainsFunc(skills, func(sk SkillDef) bool { return sk.Name == name }) {
				t.Errorf("char %d: legacySkillOrder の %s がスキルにありません", c, name)
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

// 並べ替える前のセーブ（位置で持つスキルレベル）を読むと、各スキルの
// レベルが名前どおりに今の並びへ移ること。
func TestLegacySkillLevelsFollowSkillNames(t *testing.T) {
	var old [partySize][8]int
	for c := range partySize {
		for j := range old[c] {
			old[c][j] = 1
		}
	}
	old[0][3] = 3 // 旧並びの4番目＝回復
	old[1][1] = 2 // 旧並びの2番目＝妖艶の舞

	got := skillLevelsFromNames(skillLevelsByName(old, legacySkillOrderOf))
	for c := range partySize {
		for j, sk := range characterSkillSets[c] {
			want := 1
			switch sk.Name {
			case "回復":
				want = 3
			case "妖艶の舞":
				want = 2
			}
			if got[c][j] != want {
				t.Errorf("char %d %s: Lv%d, want Lv%d", c, sk.Name, got[c][j], want)
			}
		}
	}

	// 名前で保存したものは、そのまま同じレベルに戻る。
	back := skillLevelsFromNames(skillLevelsByName(got, currentSkillOrder))
	if back != got {
		t.Errorf("round trip changed levels: %v -> %v", got, back)
	}
}

// 覚えていないスキルが途中にあっても、一覧は覚えているスキルだけを
// 上から詰めて並べ、行とスキルの添字が行き来できること。
func TestSkillDisplayRowsSkipLockedSkills(t *testing.T) {
	g := &Game{}
	g.PlayerLv[0] = 1 // 男: Lv1で覚えているのはスマッシュ(0)とみやぶる(5)だけ
	if got := g.SkillDisplayRow(0, 5); got != 1 {
		t.Errorf("みやぶるの行 = %d, want 1", got)
	}
	if idx, ok := g.SkillAtDisplayRow(0, 1); !ok || idx != 5 {
		t.Errorf("1行目のスキル = %d,%v, want 5,true", idx, ok)
	}
	if _, ok := g.SkillAtDisplayRow(0, 2); ok {
		t.Errorf("2行目にスキルがあってはいけない")
	}
}
