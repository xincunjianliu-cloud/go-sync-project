package main

import "testing"

func TestEventCompletionAndLoadedProgressAdvanceObjective(t *testing.T) {
	oldDefs := objectiveDefs
	t.Cleanup(func() { objectiveDefs = oldDefs })
	objectiveDefs = []objectiveDef{
		{ID: "first", SeenKey: "event-first"},
		{ID: "second", SeenKey: "event-second"},
	}

	g := &Game{}
	g.UpdateObjective()
	if g.CurrentObjectiveID != "first" {
		t.Fatalf("initial objective = %q, want first", g.CurrentObjectiveID)
	}
	g.markEventSeen("event-first")
	if g.CurrentObjectiveID != "second" {
		t.Fatalf("objective after event = %q, want second", g.CurrentObjectiveID)
	}

	g.setObjectiveProgress([4]bool{}, map[string]bool{
		"event-first":  true,
		"event-second": true,
	})
	if g.CurrentObjectiveID != "" {
		t.Fatalf("objective after loading completed events = %q, want none", g.CurrentObjectiveID)
	}
}
