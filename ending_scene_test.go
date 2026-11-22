package main

import (
	"math"
	"testing"
)

func TestCreditScrollProgressUsesMusicTimeline(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  float64
		duration float64
		want     float64
	}{
		{name: "start", elapsed: 0, duration: 98.5, want: 0},
		{name: "halfway", elapsed: 49.25, duration: 98.5, want: 0.5},
		{name: "clamped after song", elapsed: 100, duration: 98.5, want: 1},
		{name: "nonpositive duration", elapsed: 1, duration: 0, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := creditScrollProgress(tt.elapsed, tt.duration); got != tt.want {
				t.Fatalf("creditScrollProgress(%v, %v) = %v, want %v", tt.elapsed, tt.duration, got, tt.want)
			}
		})
	}
}

func TestCreditRollDistanceIncludesSections(t *testing.T) {
	pages := []CreditPage{
		{Lines: []string{"role", "name"}},
		{Lines: []string{"thanks"}},
		{Lines: nil},
	}
	want := float64(gameHeight) + 3*creditLineGap + creditSectionGap
	if got := creditRollDistance(pages); got != want {
		t.Fatalf("creditRollDistance() = %v, want %v", got, want)
	}
}

func TestEndingLogoAlpha(t *testing.T) {
	tests := []struct {
		name    string
		elapsed float64
		want    float64
	}{
		{name: "start transparent", elapsed: 0, want: 0},
		{name: "fade in", elapsed: endingLogoFadeInDuration / 2, want: 0.5},
		{name: "hold", elapsed: endingLogoFadeInDuration, want: 1},
		{name: "fade out", elapsed: endingLogoFadeInDuration + endingLogoHoldDuration + endingLogoFadeOutDuration/2, want: 0.5},
		{name: "end transparent", elapsed: endingLogoDuration, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endingLogoAlpha(tt.elapsed); math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("endingLogoAlpha(%v) = %v, want %v", tt.elapsed, got, tt.want)
			}
		})
	}
}
