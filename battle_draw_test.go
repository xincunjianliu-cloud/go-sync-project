package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDamagePopRecoveryColors(t *testing.T) {
	tests := []struct {
		name string
		pop  DamagePop
		want color.RGBA
	}{
		{name: "HP recovery is green", pop: DamagePop{IsHeal: true}, want: popColorHeal},
		{name: "MP recovery is blue", pop: DamagePop{IsHeal: true, IsMP: true}, want: popColorMPHeal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := damagePopColor(tt.pop); got != tt.want {
				t.Errorf("damagePopColor(%+v) = %v, want %v", tt.pop, got, tt.want)
			}
		})
	}
}

func TestAllTargetRowsGrowUpward(t *testing.T) {
	scene := &BattleScene{}

	allyRect := scene.allTargetRowRect()
	if allyRect.h != 39 || allyRect.y+allyRect.h != 422 {
		t.Errorf("ally all-target row = y:%v h:%v bottom:%v, want h:39 bottom:422", allyRect.y, allyRect.h, allyRect.y+allyRect.h)
	}

	for range maxEnemies {
		scene.enemies = append(scene.enemies, EnemyUnit{Image: ebiten.NewImage(20, 30)})
	}
	enemyRect := scene.enemyAllTargetRowRect()
	lowestEnemyBottom := 0.0
	for i := range scene.enemies {
		_, y, _, h := scene.enemyDrawRect(i)
		if y+h > lowestEnemyBottom {
			lowestEnemyBottom = y + h
		}
	}
	wantEnemyBottom := lowestEnemyBottom + enemyAllTargetRowGapY + allTargetRowH
	if enemyRect.h != 39 || enemyRect.y+enemyRect.h != wantEnemyBottom {
		t.Errorf("enemy all-target row = y:%v h:%v bottom:%v, want h:39 bottom:%v", enemyRect.y, enemyRect.h, enemyRect.y+enemyRect.h, wantEnemyBottom)
	}
}
