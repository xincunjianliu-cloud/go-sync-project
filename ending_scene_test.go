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

func TestCreditRollDistanceUsesWideSpecialThanksGap(t *testing.T) {
	pages := []CreditPage{
		{Lines: []string{"engine"}},
		{Lines: []string{"Special Thanks", "friend"}},
	}
	want := float64(gameHeight) + 3*creditLineGap + creditBlockGap
	if got := creditRollDistance(pages); got != want {
		t.Fatalf("creditRollDistance() = %v, want %v", got, want)
	}
}

func TestCreditRollDistanceUsesWideEngineGap(t *testing.T) {
	pages := []CreditPage{
		{Lines: []string{"role", "name"}},
		{Lines: []string{"ENGINE", "Ebitengine"}},
	}
	want := float64(gameHeight) + 4*creditLineGap + creditBlockGap
	if got := creditRollDistance(pages); got != want {
		t.Fatalf("creditRollDistance() = %v, want %v", got, want)
	}
}

func TestEnteringEndingLogo(t *testing.T) {
	s := &EndingScene{
		creditDuration: 98.5,
		creditElapsed:  90,
		phase:          endingPhaseCredits,
	}

	s.enterEndingLogo()

	if s.phase != endingPhaseLogo {
		t.Fatalf("phase = %d, want endingPhaseLogo", s.phase)
	}
	if s.logoElapsed != 0 {
		t.Fatalf("logoElapsed = %v, want 0", s.logoElapsed)
	}
}

func TestEndingCreditRollLeavesTimeForPauseAndLogo(t *testing.T) {
	const musicDuration = 100.0
	const musicOffset = 2.0

	creditDuration := endingCreditRollDuration(musicDuration, musicOffset)
	wantRemaining := endingCreditTailDuration
	if got := musicDuration - musicOffset - creditDuration; math.Abs(got-wantRemaining) > 1e-9 {
		t.Fatalf("time after credits = %v, want %v", got, wantRemaining)
	}
}

func TestCreditPausePrecedesEndingLogo(t *testing.T) {
	s := &EndingScene{phase: endingPhaseCreditPause}

	s.Update(endingCreditsClearPause / 2)
	if s.phase != endingPhaseCreditPause {
		t.Fatalf("phase = %d before pause ends, want endingPhaseCreditPause", s.phase)
	}

	s.Update(endingCreditsClearPause / 2)
	if s.phase != endingPhaseLogo {
		t.Fatalf("phase = %d after pause, want endingPhaseLogo", s.phase)
	}
}

func TestEndingLogoAlpha(t *testing.T) {
	tests := []struct {
		name    string
		elapsed float64
		want    float64
	}{
		{name: "start with credits cleared", elapsed: 0, want: 0},
		{name: "delay keeps logo transparent", elapsed: endingLogoFadeInDelay / 2, want: 0},
		{name: "fade in", elapsed: endingLogoFadeInDelay + endingLogoFadeInDuration/2, want: 0.5},
		{name: "hold", elapsed: endingLogoFadeInDelay + endingLogoFadeInDuration, want: 1},
		{name: "fade out", elapsed: endingLogoFadeOutStart + endingLogoFadeOutDuration/2, want: 0.5},
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
