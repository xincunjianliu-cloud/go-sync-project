package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// loadStoryDialogues must merge every file listed in story/_index.json
// into the global storyDialogues map, so splitting content across files
// (to limit the blast radius of a JSON typo) doesn't lose any events.
// This reads the real committed files directly (rather than hardcoding
// specific dialogue ids) so it keeps working as scenario content changes.
func TestLoadStoryDialoguesReadsSplitFiles(t *testing.T) {
	storyDialogues = map[string]storyDialogueEntry{}
	loadStoryDialogues("assets/dialogues")

	indexData, err := os.ReadFile("assets/dialogues/story/_index.json")
	if err != nil {
		t.Fatalf("failed to read story/_index.json: %v", err)
	}
	var files []string
	if err := json.Unmarshal(indexData, &files); err != nil {
		t.Fatalf("failed to parse story/_index.json: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("story/_index.json lists no files")
	}

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join("assets/dialogues/story", f))
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to parse %s: %v", f, err)
		}
		for id := range raw {
			if _, ok := storyDialogues[id]; !ok {
				t.Fatalf("expected id %q (from %s) to be loaded into storyDialogues", id, f)
			}
		}
	}
}

// resolveSpeaker: known player/boss keys resolve to their display names,
// everything else (an NPC's literal name) passes through unchanged so
// scenario writers can add NPCs without touching Go code.
func TestResolveSpeakerPassesThroughUnknownNames(t *testing.T) {
	if got := resolveSpeaker(""); got != "" {
		t.Fatalf("empty speaker should stay empty, got %q", got)
	}
	if got := resolveSpeaker("SYSTEM_COMMAND"); got != "SYSTEM_COMMAND" {
		t.Fatalf("SYSTEM_COMMAND should pass through, got %q", got)
	}
	if got := resolveSpeaker("player1"); got != PlayerNames[0] {
		t.Fatalf("player1 should resolve to %q, got %q", PlayerNames[0], got)
	}
	if got := resolveSpeaker("boss2"); got != BossNames[1] {
		t.Fatalf("boss2 should resolve to %q, got %q", BossNames[1], got)
	}
	if got := resolveSpeaker("リナ"); got != "リナ" {
		t.Fatalf("unknown NPC name should pass through as-is, got %q", got)
	}
}

// convertBossDialogue should carry the per-line expression index through
// into EventCommand so DrawChara can pick the right sprite-sheet frame.
func TestConvertBossDialoguePropagatesExpression(t *testing.T) {
	src := &bossDialogueJSON{
		Commands: []dialogueCommandJSON{
			{Speaker: "リナ", Text: "こんにちは", Expression: 2},
			{Speaker: "player1", Text: "どうも"},
		},
	}
	got := convertBossDialogue(src)
	if len(got.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(got.Commands))
	}
	if got.Commands[0].Speaker != "リナ" || got.Commands[0].Expression != 2 {
		t.Fatalf("expected リナ/expression 2, got %+v", got.Commands[0])
	}
	if got.Commands[1].Speaker != PlayerNames[0] || got.Commands[1].Expression != 0 {
		t.Fatalf("expected %s with expression 0 (default), got %+v", PlayerNames[0], got.Commands[1])
	}
}

// convertBossDialogue should resolve speakerSlots keys through
// resolveSpeaker too, so "player1": 1 in JSON ends up keyed by the
// player's display name (matching how EventCommand.Speaker is stored).
func TestConvertBossDialoguePropagatesSpeakerSides(t *testing.T) {
	src := &bossDialogueJSON{
		Commands:     []dialogueCommandJSON{{Speaker: "player1", Text: "hi"}},
		SpeakerSlots: map[string]int{"player1": 1, "リナ": 0},
	}
	got := convertBossDialogue(src)
	if got.SpeakerSides[PlayerNames[0]] != 1 {
		t.Fatalf("expected %s to be side 1, got %+v", PlayerNames[0], got.SpeakerSides)
	}
	if got.SpeakerSides["リナ"] != 0 {
		t.Fatalf("expected リナ to be side 0, got %+v", got.SpeakerSides)
	}
}

// convertBossDialogue should carry Background through so FieldScene can
// draw it full-screen instead of the map (see applyDialogue).
func TestConvertBossDialoguePropagatesBackground(t *testing.T) {
	src := &bossDialogueJSON{
		Commands:   []dialogueCommandJSON{{Speaker: "", Text: "むかしむかし……"}},
		Background: "opening_bg",
	}
	got := convertBossDialogue(src)
	if got.Background != "opening_bg" {
		t.Fatalf("expected background %q, got %q", "opening_bg", got.Background)
	}
}

// resolveEventDialogue must return the resolved BossDialogue (including
// Background) unchanged when the id is found in story data.
func TestResolveEventDialoguePropagatesBackground(t *testing.T) {
	storyDialogues["_test_opening"] = storyDialogueEntry{
		First: BossDialogue{
			Commands:   []EventCommand{{Speaker: "", Text: "むかしむかし……"}},
			Background: "opening_bg",
		},
	}
	defer delete(storyDialogues, "_test_opening")

	got := resolveEventDialogue(storyTextPrefix+"_test_opening", "", false)
	if got.Background != "opening_bg" {
		t.Fatalf("expected background %q, got %q", "opening_bg", got.Background)
	}
}

// beginMessage must not open the message box for a dialogue whose
// commands are all empty (e.g. a placeholder id with no real lines yet,
// like boss2-4's clear dialogue or an unwritten opening) — msgTexts[0]
// would otherwise be indexed with nothing in it as soon as the player
// (or the auto-fired opening trigger) tries to advance the message.
func TestBeginMessageSkipsEmptyDialogue(t *testing.T) {
	s := &FieldScene{game: &Game{}}
	s.msgTexts = nil
	s.beginMessage()
	if s.isMsgActive {
		t.Fatalf("expected beginMessage to stay inactive for an empty dialogue")
	}
}

func TestBeginMessageActivatesForNonEmptyDialogue(t *testing.T) {
	s := &FieldScene{game: &Game{}}
	s.msgTexts = []EventCommand{{Speaker: "", Text: "hi"}}
	s.beginMessage()
	if !s.isMsgActive {
		t.Fatalf("expected beginMessage to activate for a non-empty dialogue")
	}
}

// End-to-end check of the "はじめから" opening trigger wiring: resolving
// the "opening" story id and feeding it through applyDialogue+beginMessage
// (exactly what title_scene.go does for a new game) must actually open
// the message box once real content exists for that id.
func TestOpeningDialogueActivatesMessageBox(t *testing.T) {
	storyDialogues["opening"] = storyDialogueEntry{
		First: BossDialogue{
			Commands: []EventCommand{{Speaker: "", Text: "むかしむかし……"}},
		},
	}
	defer delete(storyDialogues, "opening")

	s := &FieldScene{game: &Game{}}
	s.applyDialogue(resolveEventDialogue(storyTextPrefix+"opening", "", false))
	s.msgIndex = 0
	s.beginMessage()

	if !s.isMsgActive {
		t.Fatalf("expected the opening dialogue to activate the message box")
	}
	if len(s.msgTexts) != 1 || s.msgTexts[0].Text != "むかしむかし……" {
		t.Fatalf("expected the opening's line to be loaded, got %+v", s.msgTexts)
	}
}

// UpdateCharaAnim must auto-assign the two portrait slots as speakers
// change, without any manual speakerSlots bookkeeping: a back-and-forth
// between two speakers keeps reusing the same two slots, and a third
// distinct speaker takes over whichever slot was least recently active.
func TestUpdateCharaAnimAutoAssignsSlots(t *testing.T) {
	var m MessageSystem
	m.Reset()

	m.UpdateCharaAnim("リナ", 0)
	if m.slots[0].speaker != "リナ" || !m.slots[0].spawned {
		t.Fatalf("first speaker should take slot 0, got %+v", m.slots[0])
	}

	m.UpdateCharaAnim("player1", 0)
	if m.slots[1].speaker != "player1" || !m.slots[1].spawned {
		t.Fatalf("second distinct speaker should take slot 1, got %+v", m.slots[1])
	}
	if m.slots[0].speaker != "リナ" {
		t.Fatalf("slot 0 should still hold リナ while she is not replaced, got %+v", m.slots[0])
	}

	// リナ speaks again: she is already in slot 0, so it must not move her
	// to slot 1 or spawn a duplicate.
	m.UpdateCharaAnim("リナ", 0)
	if m.slots[0].speaker != "リナ" || m.slots[1].speaker != "player1" {
		t.Fatalf("returning speaker should reuse her existing slot, got slot0=%+v slot1=%+v", m.slots[0], m.slots[1])
	}

	// A third, brand-new speaker should replace the least-recently-active
	// slot (slot 1, since player1 spoke before リナ's last line) and leave
	// リナ (the most recent speaker) alone.
	m.UpdateCharaAnim("boss1", 0)
	if m.slots[1].speaker != "boss1" {
		t.Fatalf("third speaker should take over the stale slot 1, got %+v", m.slots[1])
	}
	if m.slots[0].speaker != "リナ" {
		t.Fatalf("most recently active speaker should stay on screen, got %+v", m.slots[0])
	}
}

// With explicit SpeakerSides, 4 speakers split 2-and-2 across left/right
// must only swap within their own assigned side: two speakers pinned to
// the right take turns in slot 1 without ever disturbing slot 0, and
// vice versa.
func TestUpdateCharaAnimRespectsExplicitSides(t *testing.T) {
	var m MessageSystem
	m.Reset()
	m.SpeakerSides = map[string]int{
		"A": 0, "B": 0, // left
		"C": 1, "D": 1, // right
	}

	m.UpdateCharaAnim("A", 0)
	if m.slots[0].speaker != "A" {
		t.Fatalf("A (side 0) should take slot 0, got %+v", m.slots[0])
	}

	m.UpdateCharaAnim("C", 0)
	if m.slots[1].speaker != "C" {
		t.Fatalf("C (side 1) should take slot 1, got %+v", m.slots[1])
	}

	// D is also pinned right: it must replace C in slot 1 only, leaving
	// slot 0 (A) untouched.
	m.UpdateCharaAnim("D", 0)
	if m.slots[1].speaker != "D" {
		t.Fatalf("D (side 1) should replace C in slot 1, got %+v", m.slots[1])
	}
	if m.slots[0].speaker != "A" {
		t.Fatalf("slot 0 must stay untouched by right-side swaps, got %+v", m.slots[0])
	}

	// B is pinned left: it must replace A in slot 0 only, leaving slot 1
	// (D) untouched.
	m.UpdateCharaAnim("B", 0)
	if m.slots[0].speaker != "B" {
		t.Fatalf("B (side 0) should replace A in slot 0, got %+v", m.slots[0])
	}
	if m.slots[1].speaker != "D" {
		t.Fatalf("slot 1 must stay untouched by left-side swaps, got %+v", m.slots[1])
	}
}

// Reset must clear lastActiveSlot too, otherwise the first speaker of a
// brand new dialogue could be routed to slot 1 instead of slot 0.
func TestResetPutsFirstSpeakerInSlotZero(t *testing.T) {
	var m MessageSystem
	m.Reset()
	m.UpdateCharaAnim("boss1", 0)
	m.Reset()
	m.UpdateCharaAnim("player1", 0)
	if m.slots[0].speaker != "player1" {
		t.Fatalf("after Reset, the first speaker of a new scene should be slot 0, got %+v", m.slots[0])
	}
}

// GetCharaImage: a cached base image is returned as-is without touching disk.
func TestGetCharaImageReturnsCachedImage(t *testing.T) {
	g := &Game{CharaImgs: map[string]*ebiten.Image{}, charaSlugs: map[string]string{}, charaImgMissing: map[string]bool{}}
	sentinel := &ebiten.Image{}
	g.CharaImgs["リナ"] = sentinel
	if got := g.GetCharaImage("リナ", 0); got != sentinel {
		t.Fatalf("expected cached base image to be returned as-is")
	}
}

// GetCharaImage: a character with no chara_<name>_sheet.png at all falls
// back to their base portrait for any requested expression, so a writer
// can add "expression" to a line before the sheet art exists.
func TestGetCharaImageFallsBackToBaseWhenNoSheet(t *testing.T) {
	g := &Game{CharaImgs: map[string]*ebiten.Image{}, charaSlugs: map[string]string{}, charaImgMissing: map[string]bool{}}
	base := &ebiten.Image{}
	g.CharaImgs["リナ"] = base
	if got := g.GetCharaImage("リナ", 2); got != base {
		t.Fatalf("expected fallback to base image when no expression sheet exists")
	}
	if !g.charaImgMissing["リナ#sheet"] {
		t.Fatalf("missing sheet should be cached to avoid re-reading the asset every frame")
	}
}

// GetCharaImage: a character with a sheet cached slices out the requested
// frame (frame width = sheet width / charaExprSheetFrames).
func TestGetCharaImageSlicesExpressionSheet(t *testing.T) {
	g := &Game{CharaImgs: map[string]*ebiten.Image{}, charaSlugs: map[string]string{}, charaImgMissing: map[string]bool{}}
	sheet := ebiten.NewImage(charaExprSheetFrames*100, 150)
	g.CharaImgs["テスト#sheet"] = sheet

	got := g.GetCharaImage("テスト", 2)
	if got == nil {
		t.Fatalf("expected a sliced frame from the cached sheet")
	}
	want := image.Rect(200, 0, 300, 150)
	if got.Bounds() != want {
		t.Fatalf("expected frame rect %v, got %v", want, got.Bounds())
	}
}

// GetCharaImage: an out-of-range expression index clamps to frame 0
// instead of panicking or reading past the sheet.
func TestGetCharaImageClampsOutOfRangeExpression(t *testing.T) {
	g := &Game{CharaImgs: map[string]*ebiten.Image{}, charaSlugs: map[string]string{}, charaImgMissing: map[string]bool{}}
	sheet := ebiten.NewImage(charaExprSheetFrames*100, 150)
	g.CharaImgs["テスト#sheet"] = sheet

	got := g.GetCharaImage("テスト", 99)
	want := image.Rect(0, 0, 100, 150)
	if got == nil || got.Bounds() != want {
		t.Fatalf("expected clamped frame rect %v, got %v", want, got)
	}
}

// GetCharaImage: a speaker with no portrait at all (typo, or art not
// added yet) returns nil instead of panicking, and remembers the miss.
func TestGetCharaImageMissingSpeakerReturnsNil(t *testing.T) {
	g := &Game{CharaImgs: map[string]*ebiten.Image{}, charaSlugs: map[string]string{}, charaImgMissing: map[string]bool{}}
	if got := g.GetCharaImage("存在しないNPC", 0); got != nil {
		t.Fatalf("expected nil for a speaker with no portrait file, got %v", got)
	}
	if !g.charaImgMissing["存在しないNPC"] {
		t.Fatalf("missing speaker should be cached")
	}
}

// GetCharaImage: player/boss slugs route to their existing numbered
// filenames, and the real embedded portrait actually decodes (no sheet
// exists for them yet, so it falls back to the plain base portrait).
func TestGetCharaImageLoadsRealBossPortrait(t *testing.T) {
	g := &Game{CharaImgs: map[string]*ebiten.Image{}, charaSlugs: buildCharaSlugs(), charaImgMissing: map[string]bool{}}
	img := g.GetCharaImage(BossNames[0], 0)
	if img == nil {
		t.Fatalf("expected boss1's real portrait (assets/images/common/chara_boss1.png) to load")
	}
}

// GetBackgroundImage: a blank key means "no custom background" (the
// field scene falls back to drawing the map), and an unknown key returns
// nil rather than panicking, caching the miss.
func TestGetBackgroundImageBlankAndMissingKeys(t *testing.T) {
	g := &Game{bgImgs: map[string]*ebiten.Image{}, bgImgMissing: map[string]bool{}}
	if got := g.GetBackgroundImage(""); got != nil {
		t.Fatalf("expected nil for a blank background key, got %v", got)
	}
	if got := g.GetBackgroundImage("no_such_background"); got != nil {
		t.Fatalf("expected nil for a background with no file, got %v", got)
	}
	if !g.bgImgMissing["no_such_background"] {
		t.Fatalf("missing background should be cached")
	}
}

// GetBackgroundImage: a cached image is returned as-is without touching disk.
func TestGetBackgroundImageReturnsCachedImage(t *testing.T) {
	g := &Game{bgImgs: map[string]*ebiten.Image{}, bgImgMissing: map[string]bool{}}
	sentinel := &ebiten.Image{}
	g.bgImgs["opening_bg"] = sentinel
	if got := g.GetBackgroundImage("opening_bg"); got != sentinel {
		t.Fatalf("expected cached background image to be returned as-is")
	}
}
