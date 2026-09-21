package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSheetCSVURLEscapesGID(t *testing.T) {
	got := sheetCSVURL("1AbC-xyz", "123 456")
	want := "https://docs.google.com/spreadsheets/d/1AbC-xyz/export?format=csv&gid=123+456"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestConfigAllSourcesDerivesURLFromSpreadsheetIDAndGID(t *testing.T) {
	cfg := config{
		SpreadsheetID: "1AbC",
		Sources: []source{
			{Name: "gate_hint", GID: "0"},
			{Name: "local_test", URL: "dialogue_src/story/local_test.csv"},
		},
	}
	got, err := cfg.allSources()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sources, got %+v", got)
	}
	if got[0].URL != sheetCSVURL("1AbC", "0") {
		t.Fatalf("expected a blank url to be derived from spreadsheetId+gid, got %+v", got[0])
	}
	if got[1].URL != "dialogue_src/story/local_test.csv" {
		t.Fatalf("expected an explicit url to be left as-is, got %+v", got[1])
	}
}

func TestConfigAllSourcesErrorsWithoutGID(t *testing.T) {
	cfg := config{SpreadsheetID: "1AbC", Sources: []source{{Name: "gate_hint"}}}
	if _, err := cfg.allSources(); err == nil {
		t.Fatalf("expected an error for a blank url with no gid set")
	}
}

func TestConfigAllSourcesErrorsWithoutSpreadsheetID(t *testing.T) {
	cfg := config{Sources: []source{{Name: "gate_hint", GID: "0"}}}
	if _, err := cfg.allSources(); err == nil {
		t.Fatalf("expected an error for a blank url with no spreadsheetId set")
	}
}

func TestBuildFileJSONFirstBackgroundWins(t *testing.T) {
	rows := [][]string{
		{"id", "text", "background"},
		{"opening", "むかしむかし……", "opening_bg"},
		{"opening", "あるところに", "should_be_ignored"},
	}
	out, err := buildFileJSON(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := out["opening"].First.Background; got != "opening_bg" {
		t.Fatalf("expected background %q (first non-blank wins), got %q", "opening_bg", got)
	}
}

func TestBuildFileJSONGroupsByIDAndPhase(t *testing.T) {
	rows := [][]string{
		{"id", "phase", "speaker", "expression", "text", "bgm", "side"},
		{"gate_hint", "first", "", "", "校門の先は静かだ……", "talk_tense", ""},
		{"gate_hint", "first", "player1", "2", "先に進んでみよう", "", ""},
		{"gate_hint", "repeat", "player1", "", "今日はもう戻ろうか", "", ""},
	}

	out, err := buildFileJSON(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry, ok := out["gate_hint"]
	if !ok {
		t.Fatalf("expected gate_hint entry, got %+v", out)
	}
	if entry.First == nil || len(entry.First.Commands) != 2 {
		t.Fatalf("expected 2 commands in first, got %+v", entry.First)
	}
	if entry.First.BGM != "talk_tense" {
		t.Fatalf("expected bgm talk_tense, got %q", entry.First.BGM)
	}
	if entry.First.Commands[1].Expression != 2 {
		t.Fatalf("expected expression 2 on second line, got %d", entry.First.Commands[1].Expression)
	}
	if entry.Repeat == nil || len(entry.Repeat.Commands) != 1 {
		t.Fatalf("expected 1 command in repeat, got %+v", entry.Repeat)
	}
}

func TestBuildFileJSONFirstSideWinsPerSpeaker(t *testing.T) {
	rows := [][]string{
		{"id", "phase", "speaker", "text", "side"},
		{"scene", "first", "A", "hi", "0"},
		{"scene", "first", "B", "hi", "1"},
		{"scene", "first", "A", "again", "1"}, // should be ignored: A's side is already set
	}

	out, err := buildFileJSON(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sides := out["scene"].First.SpeakerSlots
	if sides["A"] != 0 {
		t.Fatalf("expected A's side to stay 0 (first value wins), got %d", sides["A"])
	}
	if sides["B"] != 1 {
		t.Fatalf("expected B's side to be 1, got %d", sides["B"])
	}
}

func TestBuildFileJSONSkipsBlankRows(t *testing.T) {
	rows := [][]string{
		{"id", "text"},
		{"scene", "hi"},
		{"", ""},
	}
	out, err := buildFileJSON(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected the blank row to be skipped, got %+v", out)
	}
}

func TestBuildFileJSONRejectsBadPhase(t *testing.T) {
	rows := [][]string{
		{"id", "phase", "text"},
		{"scene", "climax", "hi"},
	}
	if _, err := buildFileJSON(rows); err == nil {
		t.Fatalf("expected an error for an invalid phase value")
	}
}

func TestBuildFileJSONRejectsMissingID(t *testing.T) {
	rows := [][]string{
		{"id", "text"},
		{"", "hi"},
	}
	if _, err := buildFileJSON(rows); err == nil {
		t.Fatalf("expected an error when id is blank but text is not")
	}
}

func TestBuildFileJSONRejectsMissingColumns(t *testing.T) {
	rows := [][]string{{"speaker", "text"}}
	if _, err := buildFileJSON(rows); err == nil {
		t.Fatalf("expected an error when the id column is missing")
	}
}

// fetchCSV must tolerate the UTF-8 BOM that Excel's "CSV UTF-8" export
// prepends, which would otherwise get glued onto the "id" header text
// and make every row look like it's missing the id column.
func TestFetchCSVStripsUTF8BOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.csv")
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("id,text\nscene,hi\n")...)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	rows, err := fetchCSV(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0][0] != "id" {
		t.Fatalf("expected header %q, got %q (BOM not stripped)", "id", rows[0][0])
	}
}

func TestBuildBossFileJSONGroupsByPhase(t *testing.T) {
	rows := [][]string{
		{"phase", "speaker", "text", "bgm"},
		{"battle", "player1", "あれが噂のボスか...", "talk_tense"},
		{"battle", "boss1", "貴様...ここまで来るとはな", ""},
		{"clear", "boss1", "馬鹿な...この俺が...", "talk_peaceful"},
		{"clear", "SYSTEM_COMMAND", "START_ENDING", ""},
	}

	out, err := buildBossFileJSON(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Battle == nil || len(out.Battle.Commands) != 2 {
		t.Fatalf("expected 2 battle commands, got %+v", out.Battle)
	}
	if out.Battle.BGM != "talk_tense" {
		t.Fatalf("expected battle bgm talk_tense, got %q", out.Battle.BGM)
	}
	if out.Clear == nil || len(out.Clear.Commands) != 2 {
		t.Fatalf("expected 2 clear commands, got %+v", out.Clear)
	}
	if out.Clear.Commands[1].Speaker != "SYSTEM_COMMAND" || out.Clear.Commands[1].Text != "START_ENDING" {
		t.Fatalf("expected SYSTEM_COMMAND/START_ENDING as the last clear line, got %+v", out.Clear.Commands[1])
	}
}

func TestBuildBossFileJSONRejectsBadPhase(t *testing.T) {
	rows := [][]string{
		{"phase", "text"},
		{"intro", "hi"},
	}
	if _, err := buildBossFileJSON(rows); err == nil {
		t.Fatalf(`expected an error when phase is not "battle" or "clear"`)
	}
}

func TestBuildBossFileJSONKeepsPlaceholderEmptyText(t *testing.T) {
	rows := [][]string{
		{"phase", "speaker", "text"},
		{"battle", "boss2", ""},
	}
	out, err := buildBossFileJSON(rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Battle == nil || len(out.Battle.Commands) != 1 {
		t.Fatalf("expected the placeholder empty-text row to be kept, got %+v", out.Battle)
	}
}
