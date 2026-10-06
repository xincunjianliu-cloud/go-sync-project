package main

import (
	"os"
	"regexp"
	"testing"
)

// 敵・ボスのSkillIDs(スプレッドシートのSkills列)が、すべて
// enemySkillTableにあること(書き間違えるとそのスキルは黙って使われなくなる)。
func TestEnemySkillIDsExist(t *testing.T) {
	check := func(owner string, ids []string) {
		for _, id := range ids {
			if _, ok := enemySkillTable[id]; !ok {
				t.Errorf("%s のスキル %q がenemySkillTableにありません", owner, id)
			}
		}
	}
	for _, e := range EnemyDatabase {
		check(e.Name, e.SkillIDs)
	}
	for key, b := range BossDatabase {
		check(key, b.SkillIDs)
	}
}

// genstatsがenemy_skill.goから読み取るスキルIDが、enemySkillTableと
// 一致すること(書き方を変えてgenstatsが読めなくなっていないか)。
func TestGenstatsCanReadSkillTable(t *testing.T) {
	data, err := os.ReadFile("enemy_skill.go")
	if err != nil {
		t.Fatal(err)
	}
	// tools/genstats の skillTableKeyRe と同じ形。
	found := map[string]bool{}
	for _, m := range genstatsSkillKeyRe.FindAllSubmatch(data, -1) {
		found[string(m[1])] = true
	}
	for id := range enemySkillTable {
		if !found[id] {
			t.Errorf("genstatsがスキル %q を読み取れません(`\\t\"id\": {` の形で1行に書くこと)", id)
		}
	}
	if len(found) != len(enemySkillTable) {
		t.Errorf("genstatsが読み取るID数 = %d, enemySkillTable = %d", len(found), len(enemySkillTable))
	}
}

// 敵のスキルは必ずダメージを持つこと(威力0だと「0」のダメージ表示が出る)。
func TestEnemySkillsDealDamage(t *testing.T) {
	for id, sk := range enemySkillTable {
		if sk.Power <= 0 {
			t.Errorf("スキル %q (%s) の威力が0以下です", id, sk.Name)
		}
	}
}

var genstatsSkillKeyRe = regexp.MustCompile(`(?m)^\t"([^"]+)":\s*\{`)
