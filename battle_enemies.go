package main

func (s *BattleScene) enemyActorIndex(slot int) int {
	return partySize + slot
}

func isEnemyActor(actor int) bool {
	return actor >= partySize
}

func enemySlotFromActor(actor int) int {
	return actor - partySize
}

func (s *BattleScene) aliveEnemyIndices() []int {
	var alive []int
	for i := range s.enemies {
		if s.enemies[i].HP > 0 {
			alive = append(alive, i)
		}
	}
	return alive
}

func (s *BattleScene) allEnemiesDead() bool {
	for i := range s.enemies {
		if s.enemies[i].HP > 0 {
			return false
		}
	}
	return true
}

func (s *BattleScene) allEnemyDeathAnimDone() bool {
	for i := range s.enemies {
		if s.enemies[i].HP <= 0 && s.enemies[i].DeathPhase < 3 {
			return false
		}
	}
	return true
}

func (s *BattleScene) totalEnemyExp() int {
	total := 0
	for i := range s.enemies {
		total += s.enemies[i].Exp
	}
	return total
}

func (s *BattleScene) totalEnemySP() int {
	total := 0
	for i := range s.enemies {
		total += s.enemies[i].SP
	}
	return total
}

func (s *BattleScene) firstAliveEnemySlot() int {
	for i := range s.enemies {
		if s.enemies[i].HP > 0 {
			return i
		}
	}
	return 0
}

func (s *BattleScene) applyDamageToEnemySlot(slot int, dmg int) bool {
	if slot < 0 || slot >= len(s.enemies) || dmg <= 0 {
		return false
	}
	e := &s.enemies[slot]
	if e.HP <= 0 {
		return false
	}
	e.HP -= dmg
	if e.HP < 0 {
		e.HP = 0
	}
	e.HitFlashTimer = spriteFlashDuration
	if e.HP <= 0 && e.DeathPhase == 0 {
		e.DeathPhase = 1
		e.DeathTimer = 0
		e.Alpha = 1.0
		return true
	}
	return false
}
