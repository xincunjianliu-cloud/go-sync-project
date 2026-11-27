package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSaveScrollBarTrackCenteredInViewport(t *testing.T) {
	trackY := slotScrollBarTop(slotCardStartY)
	if trackY+scrollBarTrackHeight/2 != float64(gameHeight)/2 {
		t.Errorf("save scrollbar center = %.1f, want %.1f",
			trackY+scrollBarTrackHeight/2, float64(gameHeight)/2)
	}
}

func TestLogScrollBarTrackUsesConfiguredLengthAndIsCentered(t *testing.T) {
	img := ebiten.NewImage(1, 89)
	defer img.Deallocate()
	game := &Game{LogEntryImg: img}

	_, trackY, trackHeight, _, _ := logScrollBarGeometry(game, 12)
	if trackHeight != logScrollBarTrackH {
		t.Errorf("log scrollbar track height = %.1f, want %.1f", trackHeight, logScrollBarTrackH)
	}
	if trackY+trackHeight/2 != float64(gameHeight)/2 {
		t.Errorf("log scrollbar center = %.1f, want %.1f",
			trackY+trackHeight/2, float64(gameHeight)/2)
	}
}

func TestStepScrollTopByWheelMovesAndClampsRows(t *testing.T) {
	scrollTop := 0
	accum := 0.0
	stepScrollTopByWheel(-1, 55, &scrollTop, 10, &accum)
	if scrollTop != 1 || accum != -5 {
		t.Fatalf("down wheel = scrollTop %d, remainder %.1f; want 1, -5", scrollTop, accum)
	}

	stepScrollTopByWheel(1, 55, &scrollTop, 10, &accum)
	if scrollTop != 0 || accum != 0 {
		t.Fatalf("up wheel = scrollTop %d, remainder %.1f; want 0, 0", scrollTop, accum)
	}

	scrollTop = 10
	stepScrollTopByWheel(-1, 55, &scrollTop, 10, &accum)
	if scrollTop != 10 || accum != 0 {
		t.Fatalf("wheel past bottom = scrollTop %d, remainder %.1f; want 10, 0", scrollTop, accum)
	}
}
