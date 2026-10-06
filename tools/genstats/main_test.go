package main

import (
	"strings"
	"testing"
)

// The seed CSVs (same layout as the published sheets) parse, Skills column
// included, and the generated code carries the skill IDs.
func TestSeedSkillsColumn(t *testing.T) {
	ids, err := loadSkillIDs("../../" + skillTablePath)
	if err != nil {
		t.Fatal(err)
	}
	enemyRows, err := fetchCSV("seed/enemies.csv")
	if err != nil {
		t.Fatal(err)
	}
	enemies, err := parseEnemies(enemyRows, ids)
	if err != nil {
		t.Fatal(err)
	}
	bossRows, err := fetchCSV("seed/bosses.csv")
	if err != nil {
		t.Fatal(err)
	}
	bosses, err := parseBosses(bossRows, ids)
	if err != nil {
		t.Fatal(err)
	}
	if got := enemies[0].Skills; len(got) != 1 || got[0] != "tackle" {
		t.Errorf("%s skills = %v, want [tackle]", enemies[0].Name, got)
	}
	if got := bosses[0].Skills; len(got) != 2 {
		t.Errorf("%s skills = %v, want 2 skills", bosses[0].Key, got)
	}

	playerRows, err := fetchCSV("seed/player_stats.csv")
	if err != nil {
		t.Fatal(err)
	}
	players, err := parsePlayerStats(playerRows)
	if err != nil {
		t.Fatal(err)
	}
	expRows, err := fetchCSV("seed/player_exp.csv")
	if err != nil {
		t.Fatal(err)
	}
	exp, err := parsePlayerExp(expRows)
	if err != nil {
		t.Fatal(err)
	}
	src, err := generate(players, exp, enemies, bosses)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `SkillIDs: []string{"flame_burst", "crushing_blow"}`) {
		t.Error("generated code is missing boss_1's SkillIDs")
	}
}

func TestParseSkills(t *testing.T) {
	known := map[string]bool{"tackle": true, "bite": true}
	row := func(skills string) []string {
		r := make([]string, skillsColumn+1)
		r[skillsColumn] = skills
		return r
	}
	cases := []struct {
		cell string
		want []string
	}{
		{"", nil},
		{"tackle", []string{"tackle"}},
		{"tackle,bite", []string{"tackle", "bite"}},
		{"tackle, bite", []string{"tackle", "bite"}},
		{"tackle、bite", []string{"tackle", "bite"}},
		{"tackle / bite", []string{"tackle", "bite"}},
	}
	for _, c := range cases {
		got, err := parseSkills(row(c.cell), 0, known)
		if err != nil {
			t.Errorf("%q: %v", c.cell, err)
			continue
		}
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%q = %v, want %v", c.cell, got, c.want)
		}
	}
	// 16-column rows (no Skills column yet) mean normal attack only.
	if got, err := parseSkills(make([]string, skillsColumn), 0, known); err != nil || got != nil {
		t.Errorf("16-column row = %v, %v; want nil, nil", got, err)
	}
	if _, err := parseSkills(row("tackel"), 0, known); err == nil {
		t.Error("typo \"tackel\" was accepted")
	}
}
