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
