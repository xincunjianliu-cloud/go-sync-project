package main

import "testing"

func newWallTestScene(unlocked bool, keys string) (*FieldScene, TiledObject) {
	wallObj := TiledObject{
		X: 100, Y: 100, Width: 32, Height: 32,
		Properties: []TiledProperty{
			{Name: "type", Type: "string", Value: "event"},
			{Name: "text", Type: "string", Value: "event_wall"},
			{Name: "keys", Type: "string", Value: keys},
		},
	}

	g := &Game{UnlockedWalls: make(map[string]bool), Keys: make(map[string]int)}
	cfg := DefaultFieldPlayerConfig()
	s := &FieldScene{
		game:       g,
		playerCfg:  cfg,
		currentMap: "test_map",
		tileMap: TiledMap{
			Width: 100, Height: 100, TileWidth: 16, TileHeight: 16,
			Layers: []TiledLayer{
				{Name: "events", Type: "objectgroup", Objects: []TiledObject{wallObj}},
			},
		},
	}
	if unlocked {
		g.UnlockedWalls[chestKey(s.currentMap, wallObj)] = true
	}
	return s, wallObj
}

func TestLockedWallBlocksMovementUntilUnlocked(t *testing.T) {
	s, wallObj := newWallTestScene(false, "鍵a")

	cx, cy := wallObj.X+wallObj.Width/2, wallObj.Y+wallObj.Height/2
	if !s.isWall(cx, cy) {
		t.Fatal("expected locked wall to block movement")
	}

	s.game.UnlockedWalls[chestKey(s.currentMap, wallObj)] = true
	if s.isWall(cx, cy) {
		t.Fatal("expected unlocked wall to no longer block movement")
	}
}

func TestExamineWallRequiresAllNamedKeys(t *testing.T) {
	s, wallObj := newWallTestScene(false, "鍵a,鍵b,鍵c")
	key := chestKey(s.currentMap, wallObj)

	s.examineWall(wallObj)
	if s.game.UnlockedWalls[key] {
		t.Fatal("wall should stay locked when player has none of the named keys")
	}
	if !s.isItemGetActive || s.itemGetName == "" || !s.itemGetPlainMessage {
		t.Fatal("expected the chest-style center popup to explain the wall needs keys")
	}

	s.game.Keys["鍵a"] = 1
	s.game.Keys["鍵b"] = 1
	s.examineWall(wallObj)
	if s.game.UnlockedWalls[key] {
		t.Fatal("wall should stay locked until every named key is collected")
	}

	s.game.Keys["鍵c"] = 1
	s.examineWall(wallObj)
	if s.game.UnlockedWalls[key] {
		t.Fatal("wall should not unlock immediately; it should start fading out first")
	}
	if !s.wallFadeActive || s.wallFadeKey != key {
		t.Fatal("expected the wall fade-out animation to start once all named keys are collected")
	}

	s.updateWallFade(wallFadeDuration + 0.01)
	if s.wallFadeActive {
		t.Fatal("expected the wall fade to finish")
	}
	if !s.game.UnlockedWalls[key] {
		t.Fatal("wall should be unlocked once the fade-out animation finishes")
	}
	if !s.isItemGetActive || s.itemGetName == "" {
		t.Fatal("expected a completion message once the wall finishes fading out")
	}
}

func TestOpenKeyChestGrantsNamedKey(t *testing.T) {
	obj := TiledObject{
		X: 0, Y: 0, Width: 32, Height: 32,
		Properties: []TiledProperty{
			{Name: "type", Type: "string", Value: "event"},
			{Name: "text", Type: "string", Value: "event_chest_key_鍵a"},
		},
	}
	g := &Game{OpenedChests: make(map[string]bool), Keys: make(map[string]int)}
	s := &FieldScene{game: g, currentMap: "test_map"}

	s.openKeyChest(obj, "鍵a")

	if g.Keys["鍵a"] != 1 {
		t.Fatalf("expected 1 of 鍵a in Keys, got %d", g.Keys["鍵a"])
	}
	if !s.isItemGetActive || s.itemGetName != "鍵a" {
		t.Fatalf("expected item-get popup for 鍵a, got active=%v name=%q", s.isItemGetActive, s.itemGetName)
	}

	s.openKeyChest(obj, "鍵a")
	if g.Keys["鍵a"] != 1 {
		t.Fatalf("expected chest to stay opened, got count %d", g.Keys["鍵a"])
	}
}

func TestSplitKeyNames(t *testing.T) {
	got := splitKeyNames(" 鍵a ,鍵b,, 鍵c")
	want := []string{"鍵a", "鍵b", "鍵c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestRemainingKeysInGroup(t *testing.T) {
	names := []string{"鍵a", "鍵b", "鍵c"}
	owned := map[string]int{"鍵a": 1}
	if got := remainingKeysInGroup(names, owned); got != 2 {
		t.Fatalf("expected 2 remaining, got %d", got)
	}
}

func TestComputeWallKeyGroups(t *testing.T) {
	tmap := TiledMap{
		Layers: []TiledLayer{
			{
				Name: "events",
				Objects: []TiledObject{
					{
						Properties: []TiledProperty{
							{Name: "type", Type: "string", Value: "event"},
							{Name: "text", Type: "string", Value: "event_wall"},
							{Name: "keys", Type: "string", Value: "鍵a,鍵b"},
						},
					},
				},
			},
		},
	}

	groups := computeWallKeyGroups([]TiledMap{tmap})
	if len(groups["鍵a"]) != 2 || groups["鍵a"][0] != "鍵a" || groups["鍵a"][1] != "鍵b" {
		t.Fatalf("unexpected group for 鍵a: %v", groups["鍵a"])
	}
	if len(groups["鍵b"]) != 2 {
		t.Fatalf("unexpected group for 鍵b: %v", groups["鍵b"])
	}
}

func newLeverWallTestScene(leverID string) (*FieldScene, TiledObject, TiledObject) {
	wallObj := TiledObject{
		X: 100, Y: 100, Width: 32, Height: 32,
		Properties: []TiledProperty{
			{Name: "type", Type: "string", Value: "event"},
			{Name: "text", Type: "string", Value: "event_wall"},
			{Name: "lever", Type: "string", Value: leverID},
		},
	}
	leverObj := TiledObject{
		X: 200, Y: 200, Width: 32, Height: 32,
		Properties: []TiledProperty{
			{Name: "type", Type: "string", Value: "event"},
			{Name: "text", Type: "string", Value: "event_lever"},
			{Name: "id", Type: "string", Value: leverID},
		},
	}

	g := &Game{UnlockedWalls: make(map[string]bool), RaisedLevers: make(map[string]bool)}
	s := &FieldScene{
		game:       g,
		currentMap: "test_map",
		tileMap: TiledMap{
			Width: 100, Height: 100, TileWidth: 16, TileHeight: 16,
			Layers: []TiledLayer{
				{Name: "events", Type: "objectgroup", Objects: []TiledObject{wallObj, leverObj}},
			},
		},
	}
	return s, wallObj, leverObj
}

func TestPullLeverOpensLinkedWallImmediately(t *testing.T) {
	s, wallObj, leverObj := newLeverWallTestScene("lever_a")

	cx, cy := wallObj.X+wallObj.Width/2, wallObj.Y+wallObj.Height/2
	if !s.isWall(cx, cy) {
		t.Fatal("expected lever-controlled wall to block movement before the lever is pulled")
	}

	s.pullLever(leverObj)

	if !s.game.RaisedLevers["lever_a"] {
		t.Fatal("expected lever_a to be recorded as raised")
	}
	if s.isWall(cx, cy) {
		t.Fatal("expected wall linked to the raised lever to open immediately without re-examining it")
	}
	if !s.wallIsOpen(wallObj) {
		t.Fatal("expected wallIsOpen to report the lever-linked wall as open")
	}
}

func TestPullLeverTogglesUpAndDown(t *testing.T) {
	s, wallObj, leverObj := newLeverWallTestScene("lever_a")
	cx, cy := wallObj.X+wallObj.Width/2, wallObj.Y+wallObj.Height/2

	s.pullLever(leverObj)
	if !s.game.RaisedLevers["lever_a"] {
		t.Fatal("expected lever_a to be raised after the first pull")
	}
	if s.isWall(cx, cy) {
		t.Fatal("expected linked wall to be open while the lever is raised")
	}

	s.pullLever(leverObj)
	if s.game.RaisedLevers["lever_a"] {
		t.Fatal("expected lever_a to be lowered after the second pull")
	}
	if !s.isWall(cx, cy) {
		t.Fatal("expected linked wall to block movement again once the lever is lowered")
	}

	s.pullLever(leverObj)
	if !s.game.RaisedLevers["lever_a"] {
		t.Fatal("expected lever_a to be raised again after a third pull")
	}
}
