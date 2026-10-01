package main

import (
	"math"
	"testing"
)

func TestSpdBonus(t *testing.T) {
	cases := []struct{ spd, min, max, want float64 }{
		{10, 10, 30, 0},
		{30, 10, 30, 10},
		{20, 10, 30, 5},
		{15, 15, 15, 0}, // everyone equal: no bonus
	}
	for _, c := range cases {
		if got := spdBonus(c.spd, c.min, c.max, 10); got != c.want {
			t.Errorf("spdBonus(%v, %v, %v, 10) = %v, want %v", c.spd, c.min, c.max, got, c.want)
		}
	}
}

// TestInitialPlacementNoOverlap mirrors initTimelinePositions: a Lv1 party
// against three identical slimes, placed slowest first at their Spd bonus.
func TestInitialPlacementNoOverlap(t *testing.T) {
	spds := []float64{9, 9, 9, 23, 23, 25, 28} // ascending, as initTimelinePositions sorts
	sp := atbIconSpacing()
	var placed []timelineOccupant
	for _, spd := range spds {
		pos := resolveTimelinePos(spdBonus(spd, 9, 28, spdStartBonusMax), spd, placed, sp)
		placed = append(placed, timelineOccupant{pos, spd})
		t.Logf("spd %v -> start %.1f", spd, pos)
	}
	for i := range placed {
		for j := i + 1; j < len(placed); j++ {
			if d := math.Abs(placed[i].pos - placed[j].pos); d < sp-1e-6 {
				t.Errorf("actors %d and %d overlap: %v vs %v (spacing %v)", i, j, placed[i].pos, placed[j].pos, sp)
			}
		}
	}
}

func TestResolveTimelinePos(t *testing.T) {
	const sp = 5.0
	cases := []struct {
		name   string
		pos    float64
		spd    float64
		others []timelineOccupant
		want   float64
	}{
		{"free spot", 20, 10, []timelineOccupant{{50, 10}}, 20},
		{"faster goes in front", 20, 30, []timelineOccupant{{20, 10}}, 25},
		{"slower goes behind", 20, 10, []timelineOccupant{{20, 30}}, 15},
		{"equal spd goes behind", 20, 10, []timelineOccupant{{20, 10}}, 15},
		{"partial overlap resolves from occupant", 22, 30, []timelineOccupant{{20, 10}}, 25},
		{"chains past a second occupant", 20, 30, []timelineOccupant{{20, 10}, {25, 10}}, 30},
		{"falls back to other side at track start", 1, 10, []timelineOccupant{{0, 30}}, 5},
		{"falls back to other side near goal", 98, 30, []timelineOccupant{{98, 10}}, 93},
		{"goal is never resolved", 100, 10, []timelineOccupant{{100, 10}}, 100},
	}
	for _, c := range cases {
		got := resolveTimelinePos(c.pos, c.spd, c.others, sp)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
