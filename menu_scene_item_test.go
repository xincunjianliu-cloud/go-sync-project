package main

import "testing"

func TestMenuItemsIncludeBattleOnlyItems(t *testing.T) {
	scene := &MenuScene{game: &Game{Inventory: []InventorySlot{
		{ItemID: "timer_booster", Count: 1},
		{ItemID: "potion", Count: 2},
		{ItemID: "exp_booster", Count: 1},
		{ItemID: "smoke_bomb", Count: 0},
		{ItemID: "unknown", Count: 1},
	}}}

	got := scene.menuItems()
	if len(got) != 3 {
		t.Fatalf("menu item count = %d, want 3", len(got))
	}
	want := []string{"timer_booster", "potion", "exp_booster"}
	for i := range want {
		if got[i].ItemID != want[i] {
			t.Errorf("menu item %d = %q, want %q", i, got[i].ItemID, want[i])
		}
	}
	if itemUsableInField(got[0]) {
		t.Error("battle-only item is usable in the field")
	}
	if !itemUsableInField(got[1]) {
		t.Error("field-usable item is not usable in the field")
	}
}

func TestItemListNavigationIncludesBattleOnlyItems(t *testing.T) {
	tests := []struct {
		count int
		from  int
		dir   int
		want  int
	}{
		{4, 1, 1, 2},
		{4, 1, -1, 0},
		{1, 0, 1, 0},
	}
	for _, tt := range tests {
		got, ok := nextItemListIndex(tt.count, tt.from, tt.dir)
		if !ok || got != tt.want {
			t.Errorf("nextItemListIndex(%d, %d, %d) = (%d, %t), want (%d, true)",
				tt.count, tt.from, tt.dir, got, ok, tt.want)
		}
	}
	if _, ok := nextItemListIndex(0, 0, 1); ok {
		t.Error("navigation succeeded for an empty item list")
	}
}

func TestMenuItemVisibleRangeTracksSelection(t *testing.T) {
	visibleRows := menuItemVisibleRowCount(100)
	tests := []struct {
		itemCount int
		selected  int
		requested int
		wantFirst int
		wantLast  int
	}{
		{itemCount: 0, selected: 0, requested: 0, wantFirst: 0, wantLast: 0},
		{itemCount: visibleRows / 2, selected: 0, requested: 0, wantFirst: 0, wantLast: visibleRows / 2},
		{itemCount: visibleRows + 3, selected: 0, requested: 0, wantFirst: 0, wantLast: visibleRows},
		{itemCount: visibleRows + 3, selected: visibleRows + 2, requested: 0, wantFirst: 3, wantLast: visibleRows + 3},
		{itemCount: visibleRows + 3, selected: visibleRows + 1, requested: 2, wantFirst: 2, wantLast: visibleRows + 2},
	}

	for _, tt := range tests {
		first, last := menuItemVisibleRange(tt.itemCount, tt.selected, tt.requested)
		if first != tt.wantFirst || last != tt.wantLast {
			t.Errorf("menuItemVisibleRange(%d, %d, %d) = [%d, %d), want [%d, %d)",
				tt.itemCount, tt.selected, tt.requested, first, last, tt.wantFirst, tt.wantLast)
		}
	}
}

func TestMenuItemScrollBarGeometry(t *testing.T) {
	visibleRows := menuItemVisibleRowCount(100)
	if _, _, _, _, _, ok := menuItemScrollBarGeometry(visibleRows, 0, visibleRows); ok {
		t.Fatal("scrollbar shown when every item fits")
	}

	itemCount := visibleRows + 3
	_, trackY, trackHeight, topThumbY, thumbH, ok := menuItemScrollBarGeometry(itemCount, 0, visibleRows)
	if !ok {
		t.Fatal("scrollbar hidden when items overflow")
	}
	_, _, _, bottomThumbY, _, _ := menuItemScrollBarGeometry(itemCount, 3, itemCount)
	if trackHeight <= 0 || thumbH <= 0 || topThumbY >= bottomThumbY || topThumbY != trackY {
		t.Errorf("scrollbar geometry = track y %.1f h %.1f; thumb y %.1f..%.1f h %.1f",
			trackY, trackHeight, topThumbY, bottomThumbY, thumbH)
	}
	if trackHeight != itemScrollBarTrackH ||
		trackY+trackHeight/2 != (menuFrameTop+menuFrameBottom)/2 {
		t.Errorf("menu scrollbar track = y %.1f h %.1f, want height %.1f centered at y %.1f",
			trackY, trackHeight, itemScrollBarTrackH, (menuFrameTop+menuFrameBottom)/2)
	}
}
