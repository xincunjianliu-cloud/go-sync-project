package main

import "testing"

func newDarknessTestScene(leverID string) (*FieldScene, TiledObject) {
	props := []TiledProperty{
		{Name: "type", Type: "string", Value: "darkness"},
	}
	if leverID != "" {
		props = append(props, TiledProperty{Name: "lever", Type: "string", Value: leverID})
	}

	zoneObj := TiledObject{
		X: 100, Y: 100, Width: 64, Height: 64,
		Properties: props,
	}

	g := &Game{RaisedLevers: make(map[string]bool)}
	s := &FieldScene{
		game:       g,
		currentMap: "test_map",
		tileMap: TiledMap{
			Width: 100, Height: 100, TileWidth: 16, TileHeight: 16,
			Layers: []TiledLayer{
				{Name: "events", Type: "objectgroup", Objects: []TiledObject{zoneObj}},
			},
		},
	}
	return s, zoneObj
}

func TestUpdateDarknessActivatesInsideZone(t *testing.T) {
	s, zone := newDarknessTestScene("")

	s.px, s.py = zone.X-100, zone.Y-100
	s.updateDarkness()
	if s.isDarknessActive {
		t.Fatal("expected darkness to be inactive outside the zone")
	}

	s.px, s.py = zone.X+zone.Width/2, zone.Y+zone.Height/2
	s.updateDarkness()
	if !s.isDarknessActive {
		t.Fatal("expected darkness to be active inside the zone")
	}
	if s.darknessRadius != defaultDarknessRadius {
		t.Fatalf("expected fixed radius %v, got %v", defaultDarknessRadius, s.darknessRadius)
	}
}

func TestUpdateDarknessDisabledByRaisedLever(t *testing.T) {
	s, zone := newDarknessTestScene("lever_a")
	s.px, s.py = zone.X+1, zone.Y+1

	s.updateDarkness()
	if !s.isDarknessActive {
		t.Fatal("expected darkness to be active before the lever is raised")
	}

	s.game.RaisedLevers["lever_a"] = true
	s.updateDarkness()
	if s.isDarknessActive {
		t.Fatal("expected darkness to be disabled once the linked lever is raised")
	}

	s.game.RaisedLevers["lever_a"] = false
	s.updateDarkness()
	if !s.isDarknessActive {
		t.Fatal("expected darkness to reactivate once the lever is lowered again")
	}
}
