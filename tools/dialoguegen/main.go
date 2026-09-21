// Command dialoguegen regenerates every dialogue asset file under
// assets/dialogues/ from spreadsheet data, mirroring how tools/genstats
// regenerates stats_generated.go.
//
// Preferred setup: one Google Sheets file, one tab per output file. Share
// the file so "anyone with the link" can view it, then in config.json set
// spreadsheetId to the file's id (the long id in its URL, .../d/<id>/edit)
// and list each tab as a source with a name and that tab's gid (open the
// tab in the browser and copy the number after "gid=" in the address
// bar — the first tab is usually gid=0) — a blank url derives the CSV
// URL from spreadsheetId + gid:
//
//	{
//	  "spreadsheetId": "1AbC...xyz",
//	  "sources": [
//	    { "name": "chapter1", "gid": "123456" },
//	    { "name": "boss_1", "gid": "789012", "kind": "boss" }
//	  ]
//	}
//
// A source's url can also be an explicit local file path or full http(s)
// URL (e.g. a tab published to the web the old way) — useful for local
// test fixtures, or when a source isn't a tab of the same spreadsheet.
// Then run:
//
//	go run ./tools/dialoguegen
//
// There are two kinds of tab, set per source via "kind":
//
//   - "story" (default): becomes assets/dialogues/story/<name>.json, and
//     can hold any number of dialogue ids. Columns:
//     id          required. The dialogue id (event_story_<id> in Tiled).
//                 Repeat it on every row of the same conversation.
//     phase       "first" or "repeat" (default "first").
//   - "boss": becomes assets/dialogues/<name>.json directly, and the tab
//     name must be boss_1..boss_4 (one tab = one boss's whole file, no id
//     column). Columns:
//     phase       "battle" or "clear" (required).
//
// Both kinds share these columns (by header name, so reordering columns
// in the sheet is fine):
//
//	speaker     blank = narration, or "player1"/"boss1"/a literal NPC name.
//	expression  0=通常 1=笑顔 2=怒り 3=驚き (blank = 0).
//	text        the line.
//	bgm         optional, applies to the whole group (id+phase, or phase
//	            for boss tabs).
//	side        optional, 0 (left) or 1 (right); pins that speaker to a
//	            side for this group (see MessageSystem.SpeakerSides).
//	background  optional, an assets/images/backgrounds/<key>.png key drawn
//	            full-screen instead of the map for this group (e.g. for an
//	            opening/visual-novel-style scene). Blank = just show the map.
//
// The generated files are derived data: edit the CSV/spreadsheet and
// rerun this tool instead of hand-editing the JSON.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	configPath     = "tools/dialoguegen/config.json"
	storyOutputDir = "assets/dialogues/story"
	bossOutputDir  = "assets/dialogues"
)

var validBossTabNames = map[string]bool{"boss_1": true, "boss_2": true, "boss_3": true, "boss_4": true}

// The following mirror the JSON shapes event_data.go unmarshals at
// runtime (dialogueCommandJSON / bossDialogueJSON / storyDialogueFileJSON /
// bossDialogueFileJSON). Keep the field names/tags in sync with that file.
type dialogueCommandJSON struct {
	Speaker    string `json:"speaker"`
	Text       string `json:"text"`
	Expression int    `json:"expression,omitempty"`
}

type bossDialogueJSON struct {
	Commands     []dialogueCommandJSON `json:"commands"`
	BGM          string                `json:"bgm,omitempty"`
	SpeakerSlots map[string]int        `json:"speakerSlots,omitempty"`
	Background   string                `json:"background,omitempty"`
}

type storyDialogueFileJSON struct {
	First  *bossDialogueJSON `json:"first,omitempty"`
	Repeat *bossDialogueJSON `json:"repeat,omitempty"`
}

type bossDialogueFileJSON struct {
	Battle *bossDialogueJSON `json:"battle,omitempty"`
	Clear  *bossDialogueJSON `json:"clear,omitempty"`
}

type source struct {
	Name string `json:"name"`
	// URL is a local file path or full http(s) URL. Leave it blank to
	// derive it from config.SpreadsheetID + this GID — the normal case
	// for a real spreadsheet tab.
	URL string `json:"url,omitempty"`
	// GID is the tab's numeric sheet id, visible in the URL when that
	// tab is open in the browser (the number after "gid=", e.g. the
	// first tab is usually gid=0). Used to derive URL when URL is blank.
	//
	// This uses Sheets' plain CSV export (.../export?format=csv&gid=...)
	// rather than the gviz "fetch by tab name" endpoint: the gviz one
	// was observed serving a stale cached copy of a tab for several
	// minutes after an edit (reads back an older version of the sheet),
	// which is silently wrong instead of just an error, so it's not
	// used even though it's more convenient (no gid to look up).
	GID string `json:"gid,omitempty"`
	// Kind is "story" (default) or "boss". See the package doc comment.
	Kind string `json:"kind,omitempty"`
}

type config struct {
	// SpreadsheetID: the Google Sheets file to pull tabs from when a
	// source's URL is left blank (see source.URL / source.GID).
	SpreadsheetID string   `json:"spreadsheetId,omitempty"`
	Sources       []source `json:"sources"`
}

// sheetCSVURL builds the URL that exports one tab (by numeric gid) of a
// Google Sheets file as CSV. The file just needs to be shared as "anyone
// with the link" can view.
func sheetCSVURL(spreadsheetID, gid string) string {
	return fmt.Sprintf(
		"https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s",
		spreadsheetID, url.QueryEscape(gid),
	)
}

// allSources fills in a URL (derived from SpreadsheetID + GID) for any
// source that left URL blank.
func (c config) allSources() ([]source, error) {
	out := make([]source, len(c.Sources))
	copy(out, c.Sources)
	for i := range out {
		if out[i].URL != "" {
			continue
		}
		if out[i].GID == "" {
			return nil, fmt.Errorf(`source %q has no url and no gid (open that tab in the browser and copy the number after "gid=" in the address bar)`, out[i].Name)
		}
		if c.SpreadsheetID == "" {
			return nil, fmt.Errorf("source %q has no url and spreadsheetId is not set", out[i].Name)
		}
		out[i].URL = sheetCSVURL(c.SpreadsheetID, out[i].GID)
	}
	return out, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dialoguegen:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	sources, err := cfg.allSources()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return fmt.Errorf("no sources configured (set sources in %s)", configPath)
	}

	seenNames := map[string]bool{}
	for _, src := range sources {
		if seenNames[src.Name] {
			return fmt.Errorf("source name %q is listed more than once", src.Name)
		}
		seenNames[src.Name] = true
	}

	seenIDs := map[string]string{} // id -> which source defined it first
	var indexNames []string

	for _, src := range sources {
		rows, err := fetchCSV(src.URL)
		if err != nil {
			return fmt.Errorf("%s: %w", src.Name, err)
		}

		if src.Kind == "boss" {
			if !validBossTabNames[src.Name] {
				return fmt.Errorf(`%s: a "boss" source must be named boss_1, boss_2, boss_3, or boss_4`, src.Name)
			}
			fileJSON, err := buildBossFileJSON(rows)
			if err != nil {
				return fmt.Errorf("%s: %w", src.Name, err)
			}
			data, err := json.MarshalIndent(fileJSON, "", "  ")
			if err != nil {
				return fmt.Errorf("%s: marshal: %w", src.Name, err)
			}
			outPath := filepath.Join(bossOutputDir, src.Name+".json")
			if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", outPath, err)
			}
			fmt.Printf("wrote %s\n", outPath)
			continue
		}

		fileJSON, err := buildFileJSON(rows)
		if err != nil {
			return fmt.Errorf("%s: %w", src.Name, err)
		}

		for id := range fileJSON {
			if prev, dup := seenIDs[id]; dup {
				return fmt.Errorf("id %q is defined in both %q and %q", id, prev, src.Name)
			}
			seenIDs[id] = src.Name
		}

		data, err := json.MarshalIndent(fileJSON, "", "  ")
		if err != nil {
			return fmt.Errorf("%s: marshal: %w", src.Name, err)
		}

		outPath := filepath.Join(storyOutputDir, src.Name+".json")
		if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		indexNames = append(indexNames, src.Name+".json")
		fmt.Printf("wrote %s (%d ids)\n", outPath, len(fileJSON))
	}

	indexData, err := json.MarshalIndent(indexNames, "", "  ")
	if err != nil {
		return err
	}
	indexPath := filepath.Join(storyOutputDir, "_index.json")
	if err := os.WriteFile(indexPath, append(indexData, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", indexPath, err)
	}
	fmt.Printf("wrote %s (%d files)\n", indexPath, len(indexNames))
	return nil
}

func loadConfig(path string) (config, error) {
	var cfg config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// fetchCSV loads rows (including the header row) from a local path or an
// http(s) URL, stripping a leading UTF-8 BOM if present (Excel's "CSV
// UTF-8" export adds one, which would otherwise corrupt the "id" header).
func fetchCSV(source string) ([][]string, error) {
	var data []byte
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s: status %s", source, resp.Status)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, err
		}
	}

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // spreadsheets often trim trailing empty cells unevenly
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: empty", source)
	}
	return rows, nil
}

func columnIndex(header []string, name string) int {
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), name) {
			return i
		}
	}
	return -1
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

type dialogueGroup struct {
	commands   []dialogueCommandJSON
	bgm        string
	sides      map[string]int
	background string
}

// buildFileJSON turns one CSV's rows (header included) into the id ->
// {first, repeat} map that a story/<name>.json file holds.
func buildFileJSON(rows [][]string) (map[string]storyDialogueFileJSON, error) {
	header := rows[0]
	data := rows[1:]

	idCol := columnIndex(header, "id")
	if idCol < 0 {
		return nil, fmt.Errorf(`missing required column "id"`)
	}
	textCol := columnIndex(header, "text")
	if textCol < 0 {
		return nil, fmt.Errorf(`missing required column "text"`)
	}
	phaseCol := columnIndex(header, "phase")
	speakerCol := columnIndex(header, "speaker")
	exprCol := columnIndex(header, "expression")
	bgmCol := columnIndex(header, "bgm")
	sideCol := columnIndex(header, "side")
	bgCol := columnIndex(header, "background")

	groups := map[[2]string]*dialogueGroup{}
	var order [][2]string

	for i, row := range data {
		lineNo := i + 2 // 1-indexed, plus the header row
		id := cell(row, idCol)
		text := cell(row, textCol)
		if id == "" && text == "" {
			continue // tolerate a stray blank row from the spreadsheet
		}
		if id == "" {
			return nil, fmt.Errorf("row %d: id is empty", lineNo)
		}

		phase := strings.ToLower(cell(row, phaseCol))
		if phase == "" {
			phase = "first"
		}
		if phase != "first" && phase != "repeat" {
			return nil, fmt.Errorf(`row %d: phase must be "first" or "repeat", got %q`, lineNo, phase)
		}

		speaker := cell(row, speakerCol)

		expr := 0
		if s := cell(row, exprCol); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("row %d: expression %q is not a number", lineNo, s)
			}
			expr = v
		}

		key := [2]string{id, phase}
		g, ok := groups[key]
		if !ok {
			g = &dialogueGroup{}
			groups[key] = g
			order = append(order, key)
		}
		g.commands = append(g.commands, dialogueCommandJSON{Speaker: speaker, Text: text, Expression: expr})

		if bgm := cell(row, bgmCol); bgm != "" && g.bgm == "" {
			g.bgm = bgm
		}
		if bg := cell(row, bgCol); bg != "" && g.background == "" {
			g.background = bg
		}
		if s := cell(row, sideCol); s != "" {
			if speaker == "" {
				return nil, fmt.Errorf("row %d: side is set but speaker is empty", lineNo)
			}
			side, err := strconv.Atoi(s)
			if err != nil || (side != 0 && side != 1) {
				return nil, fmt.Errorf("row %d: side must be 0 or 1, got %q", lineNo, s)
			}
			if g.sides == nil {
				g.sides = map[string]int{}
			}
			if _, exists := g.sides[speaker]; !exists {
				g.sides[speaker] = side
			}
		}
	}

	out := map[string]storyDialogueFileJSON{}
	for _, key := range order {
		id, phase := key[0], key[1]
		g := groups[key]
		bd := &bossDialogueJSON{Commands: g.commands, BGM: g.bgm, SpeakerSlots: g.sides, Background: g.background}
		entry := out[id]
		if phase == "first" {
			entry.First = bd
		} else {
			entry.Repeat = bd
		}
		out[id] = entry
	}
	return out, nil
}

// buildBossFileJSON turns one CSV's rows (header included) into the
// {battle, clear} structure a boss_N.json file holds. Unlike story files
// there's no id column: every row belongs to the one boss this tab
// represents, split only by phase (battle/clear).
func buildBossFileJSON(rows [][]string) (bossDialogueFileJSON, error) {
	header := rows[0]
	data := rows[1:]

	textCol := columnIndex(header, "text")
	if textCol < 0 {
		return bossDialogueFileJSON{}, fmt.Errorf(`missing required column "text"`)
	}
	phaseCol := columnIndex(header, "phase")
	speakerCol := columnIndex(header, "speaker")
	exprCol := columnIndex(header, "expression")
	bgmCol := columnIndex(header, "bgm")
	sideCol := columnIndex(header, "side")
	bgCol := columnIndex(header, "background")

	groups := map[string]*dialogueGroup{}

	for i, row := range data {
		lineNo := i + 2
		text := cell(row, textCol)
		phase := strings.ToLower(cell(row, phaseCol))
		if phase == "" && text == "" {
			continue // tolerate a stray blank row from the spreadsheet
		}
		if phase != "battle" && phase != "clear" {
			return bossDialogueFileJSON{}, fmt.Errorf(`row %d: phase must be "battle" or "clear", got %q`, lineNo, phase)
		}

		speaker := cell(row, speakerCol)

		expr := 0
		if s := cell(row, exprCol); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil {
				return bossDialogueFileJSON{}, fmt.Errorf("row %d: expression %q is not a number", lineNo, s)
			}
			expr = v
		}

		g, ok := groups[phase]
		if !ok {
			g = &dialogueGroup{}
			groups[phase] = g
		}
		g.commands = append(g.commands, dialogueCommandJSON{Speaker: speaker, Text: text, Expression: expr})

		if bgm := cell(row, bgmCol); bgm != "" && g.bgm == "" {
			g.bgm = bgm
		}
		if bg := cell(row, bgCol); bg != "" && g.background == "" {
			g.background = bg
		}
		if s := cell(row, sideCol); s != "" {
			if speaker == "" {
				return bossDialogueFileJSON{}, fmt.Errorf("row %d: side is set but speaker is empty", lineNo)
			}
			side, err := strconv.Atoi(s)
			if err != nil || (side != 0 && side != 1) {
				return bossDialogueFileJSON{}, fmt.Errorf("row %d: side must be 0 or 1, got %q", lineNo, s)
			}
			if g.sides == nil {
				g.sides = map[string]int{}
			}
			if _, exists := g.sides[speaker]; !exists {
				g.sides[speaker] = side
			}
		}
	}

	var out bossDialogueFileJSON
	if g, ok := groups["battle"]; ok {
		out.Battle = &bossDialogueJSON{Commands: g.commands, BGM: g.bgm, SpeakerSlots: g.sides, Background: g.background}
	}
	if g, ok := groups["clear"]; ok {
		out.Clear = &bossDialogueJSON{Commands: g.commands, BGM: g.bgm, SpeakerSlots: g.sides, Background: g.background}
	}
	return out, nil
}
