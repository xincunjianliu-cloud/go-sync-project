package main

import (
	"image/color"
	"testing"
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
