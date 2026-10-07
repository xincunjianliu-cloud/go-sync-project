// Command genstats regenerates ../../stats_generated.go from CSV data.
//
// The CSV sources are either local file paths or http(s) URLs (e.g. a
// Google Sheets tab published to the web as CSV: File > Share > Publish to
// web, choose the tab, format CSV). Configure the four sources in
// config.json, then run:
//
//	go run ./tools/genstats
//
// The enemy and boss sheets may have a 17th column, Skills, listing skill
// IDs from enemySkillTable (enemy_skill.go), e.g. "tackle,bite". Unknown
// IDs stop generation with an error.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	playerLevels       = 50 // must match maxPlayerLevel in game.go
	partySize          = 4  // must match partySize in battle_types.go
	elementalTypeCount = 4  // must match elementalTypeCount in skill.go

	outputPath = "stats_generated.go"
	configPath = "tools/genstats/config.json"
)

type config struct {
	PlayerStatsURL string `json:"playerStatsURL"`
	PlayerExpURL   string `json:"playerExpURL"`
	EnemiesURL     string `json:"enemiesURL"`
	BossesURL      string `json:"bossesURL"`
}

type playerStatRow struct {
	Level                                        int
	Slot                                         int
	HP, MP, PhysAtk, MagicAtk, PhysDef, MagicDef int
	Spd, Luck                                    int
}

type enemyRow struct {
	Name                        string
	Lv, Exp                     int
	HP, MP, PhysAtk, MagicAtk   int
	PhysDef, MagicDef, Spd, SP  int
	Element                     string
	ResistFire, ResistLightning int
	ResistIce, ResistWind       int
	Skills                      []string
}

type bossRow struct {
	Key                         string
	Lv, Exp                     int
	HP, MP, PhysAtk, MagicAtk   int
	PhysDef, MagicDef, Spd, SP  int
	Element                     string
	ResistFire, ResistLightning int
	ResistIce, ResistWind       int
	Skills                      []string
}

// skillsColumn is the optional 17th column (Skills) of the enemy and boss
// sheets: skill IDs from enemySkillTable, e.g. "tackle,bite". Rows with only
// 16 columns, or an empty cell, mean "normal attack only".
const skillsColumn = 16

// skillTablePath is where enemySkillTable lives; genstats reads its keys so
// a typo in the Skills column fails here instead of silently dropping the
// skill in game.
const skillTablePath = "enemy_skill.go"

var skillTableKeyRe = regexp.MustCompile(`(?m)^\t"([^"]+)":\s*\{`)

func loadSkillIDs(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, m := range skillTableKeyRe.FindAllSubmatch(data, -1) {
		ids[string(m[1])] = true
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s: no skill IDs found in enemySkillTable", path)
	}
	return ids, nil
}

// parseSkills splits a Skills cell. Commas (half/full width), 、, / and
// spaces all work as separators.
func parseSkills(row []string, n int, known map[string]bool) ([]string, error) {
	if len(row) <= skillsColumn {
		return nil, nil
	}
	fields := strings.FieldsFunc(row[skillsColumn], func(r rune) bool {
		switch r {
		case ',', '，', '、', '/', ' ', '　', '\t', '\n':
			return true
		}
		return false
	})
	for _, id := range fields {
		if !known[id] {
			return nil, fmt.Errorf("row %d, column Skills: unknown skill %q (see enemySkillTable in %s)", n+2, id, skillTablePath)
		}
	}
	return fields, nil
}

func hasSkillsColumn(rows [][]string) bool {
	for _, row := range rows {
		if len(row) > skillsColumn {
			return true
		}
	}
	return false
}

func skillIDsLiteral(ids []string) string {
	q := make([]string, len(ids))
	for i, id := range ids {
		q[i] = strconv.Quote(id)
	}
	return "[]string{" + strings.Join(q, ", ") + "}"
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genstats:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	playerStatsCSV, err := fetchCSV(cfg.PlayerStatsURL)
	if err != nil {
		return fmt.Errorf("player stats: %w", err)
	}
	playerExpCSV, err := fetchCSV(cfg.PlayerExpURL)
	if err != nil {
		return fmt.Errorf("player exp: %w", err)
	}
	enemiesCSV, err := fetchCSV(cfg.EnemiesURL)
	if err != nil {
		return fmt.Errorf("enemies: %w", err)
	}
	bossesCSV, err := fetchCSV(cfg.BossesURL)
	if err != nil {
		return fmt.Errorf("bosses: %w", err)
	}

	playerStats, err := parsePlayerStats(playerStatsCSV)
	if err != nil {
		return fmt.Errorf("player stats: %w", err)
	}
	playerExp, err := parsePlayerExp(playerExpCSV)
	if err != nil {
		return fmt.Errorf("player exp: %w", err)
	}
	skillIDs, err := loadSkillIDs(skillTablePath)
	if err != nil {
		return fmt.Errorf("skill table: %w", err)
	}
	enemies, err := parseEnemies(enemiesCSV, skillIDs)
	if err != nil {
		return fmt.Errorf("enemies: %w", err)
	}
	bosses, err := parseBosses(bossesCSV, skillIDs)
	if err != nil {
		return fmt.Errorf("bosses: %w", err)
	}

	if !hasSkillsColumn(enemiesCSV) {
		fmt.Fprintln(os.Stderr, "genstats: warning: enemies sheet has no Skills column (17th); every enemy will use only its normal attack")
	}
	if !hasSkillsColumn(bossesCSV) {
		fmt.Fprintln(os.Stderr, "genstats: warning: bosses sheet has no Skills column (17th); every boss will use only its normal attack")
	}

	src, err := generate(playerStats, playerExp, enemies, bosses)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if err := os.WriteFile(outputPath, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	fmt.Printf("wrote %s (%d levels, %d enemies, %d bosses)\n", outputPath, playerLevels, len(enemies), len(bosses))
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
// http(s) URL.
func fetchCSV(source string) ([][]string, error) {
	var r io.Reader
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s: status %s", source, resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(body)
	} else {
		f, err := os.Open(source)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r = f
	}
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%s: empty", source)
	}
	return rows[1:], nil // drop header
}

func atoi(field, col string, n int) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(field))
	if err != nil {
		return 0, fmt.Errorf("row %d, column %s: %q is not a number", n+2, col, field)
	}
	return v, nil
}

func parsePlayerStats(rows [][]string) ([]playerStatRow, error) {
	var out []playerStatRow
	for i, row := range rows {
		if len(row) < 10 {
			return nil, fmt.Errorf("row %d: expected 10 columns, got %d", i+2, len(row))
		}
		var r playerStatRow
		var err error
		if r.Level, err = atoi(row[0], "Level", i); err != nil {
			return nil, err
		}
		if r.Slot, err = atoi(row[1], "Slot", i); err != nil {
			return nil, err
		}
		if r.HP, err = atoi(row[2], "HP", i); err != nil {
			return nil, err
		}
		if r.MP, err = atoi(row[3], "MP", i); err != nil {
			return nil, err
		}
		if r.PhysAtk, err = atoi(row[4], "PhysAtk", i); err != nil {
			return nil, err
		}
		if r.MagicAtk, err = atoi(row[5], "MagicAtk", i); err != nil {
			return nil, err
		}
		if r.PhysDef, err = atoi(row[6], "PhysDef", i); err != nil {
			return nil, err
		}
		if r.MagicDef, err = atoi(row[7], "MagicDef", i); err != nil {
			return nil, err
		}
		if r.Spd, err = atoi(row[8], "Spd", i); err != nil {
			return nil, err
		}
		if r.Luck, err = atoi(row[9], "Luck", i); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if len(out) != playerLevels*partySize {
		return nil, fmt.Errorf("expected %d rows (%d levels x %d slots), got %d", playerLevels*partySize, playerLevels, partySize, len(out))
	}
	return out, nil
}

func parsePlayerExp(rows [][]string) ([]int, error) {
	var out []int
	for i, row := range rows {
		if len(row) < 2 {
			return nil, fmt.Errorf("row %d: expected 2 columns, got %d", i+2, len(row))
		}
		v, err := atoi(row[1], "ExpToNext", i)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if len(out) != playerLevels {
		return nil, fmt.Errorf("expected %d rows, got %d", playerLevels, len(out))
	}
	return out, nil
}

func parseEnemies(rows [][]string, skillIDs map[string]bool) ([]enemyRow, error) {
	var out []enemyRow
	for i, row := range rows {
		if len(row) < 16 {
			return nil, fmt.Errorf("row %d: expected 16 columns, got %d", i+2, len(row))
		}
		r := enemyRow{Name: row[0], Element: strings.TrimSpace(row[11])}
		var err error
		if r.Lv, err = atoi(row[1], "Lv", i); err != nil {
			return nil, err
		}
		if r.Exp, err = atoi(row[2], "Exp", i); err != nil {
			return nil, err
		}
		if r.HP, err = atoi(row[3], "HP", i); err != nil {
			return nil, err
		}
		if r.MP, err = atoi(row[4], "MP", i); err != nil {
			return nil, err
		}
		if r.PhysAtk, err = atoi(row[5], "PhysAtk", i); err != nil {
			return nil, err
		}
		if r.MagicAtk, err = atoi(row[6], "MagicAtk", i); err != nil {
			return nil, err
		}
		if r.PhysDef, err = atoi(row[7], "PhysDef", i); err != nil {
			return nil, err
		}
		if r.MagicDef, err = atoi(row[8], "MagicDef", i); err != nil {
			return nil, err
		}
		if r.Spd, err = atoi(row[9], "Spd", i); err != nil {
			return nil, err
		}
		if r.SP, err = atoi(row[10], "SP", i); err != nil {
			return nil, err
		}
		if r.ResistFire, err = atoi(row[12], "ResistFire", i); err != nil {
			return nil, err
		}
		if r.ResistLightning, err = atoi(row[13], "ResistLightning", i); err != nil {
			return nil, err
		}
		if r.ResistIce, err = atoi(row[14], "ResistIce", i); err != nil {
			return nil, err
		}
		if r.ResistWind, err = atoi(row[15], "ResistWind", i); err != nil {
			return nil, err
		}
		if r.Skills, err = parseSkills(row, i, skillIDs); err != nil {
			return nil, err
		}
		if r.Name == "" {
			return nil, fmt.Errorf("row %d: Name is empty", i+2)
		}
		out = append(out, r)
	}
	return out, nil
}

func parseBosses(rows [][]string, skillIDs map[string]bool) ([]bossRow, error) {
	var out []bossRow
	for i, row := range rows {
		if len(row) < 16 {
			return nil, fmt.Errorf("row %d: expected 16 columns, got %d", i+2, len(row))
		}
		r := bossRow{Key: strings.TrimSpace(row[0]), Element: strings.TrimSpace(row[11])}
		var err error
		if r.Lv, err = atoi(row[1], "Lv", i); err != nil {
			return nil, err
		}
		if r.Exp, err = atoi(row[2], "Exp", i); err != nil {
			return nil, err
		}
		if r.HP, err = atoi(row[3], "HP", i); err != nil {
			return nil, err
		}
		if r.MP, err = atoi(row[4], "MP", i); err != nil {
			return nil, err
		}
		if r.PhysAtk, err = atoi(row[5], "PhysAtk", i); err != nil {
			return nil, err
		}
		if r.MagicAtk, err = atoi(row[6], "MagicAtk", i); err != nil {
			return nil, err
		}
		if r.PhysDef, err = atoi(row[7], "PhysDef", i); err != nil {
			return nil, err
		}
		if r.MagicDef, err = atoi(row[8], "MagicDef", i); err != nil {
			return nil, err
		}
		if r.Spd, err = atoi(row[9], "Spd", i); err != nil {
			return nil, err
		}
		if r.SP, err = atoi(row[10], "SP", i); err != nil {
			return nil, err
		}
		if r.ResistFire, err = atoi(row[12], "ResistFire", i); err != nil {
			return nil, err
		}
		if r.ResistLightning, err = atoi(row[13], "ResistLightning", i); err != nil {
			return nil, err
		}
		if r.ResistIce, err = atoi(row[14], "ResistIce", i); err != nil {
			return nil, err
		}
		if r.ResistWind, err = atoi(row[15], "ResistWind", i); err != nil {
			return nil, err
		}
		if r.Skills, err = parseSkills(row, i, skillIDs); err != nil {
			return nil, err
		}
		if !strings.HasPrefix(r.Key, "boss_") {
			return nil, fmt.Errorf("row %d: Key %q must look like \"boss_N\"", i+2, r.Key)
		}
		out = append(out, r)
	}
	return out, nil
}

func elemConst(name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "none", "physical", "physicalnone":
		return "ElemPhysicalNone", nil
	case "fire":
		return "ElemFire", nil
	case "lightning":
		return "ElemLightning", nil
	case "ice":
		return "ElemIce", nil
	case "wind":
		return "ElemWind", nil
	default:
		return "", fmt.Errorf("unknown element %q (want none/fire/lightning/ice/wind)", name)
	}
}

// bossIndex extracts N from a "boss_N" key, returning a 0-based index.
func bossIndex(key string) (int, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(key, "boss_"))
	if err != nil {
		return 0, fmt.Errorf("boss key %q: %w", key, err)
	}
	return n - 1, nil
}

func generate(playerStats []playerStatRow, playerExp []int, enemies []enemyRow, bosses []bossRow) ([]byte, error) {
	var b strings.Builder
	b.WriteString("// Code generated by tools/genstats from spreadsheet data. DO NOT EDIT.\n")
	b.WriteString("// Regenerate with: go run ./tools/genstats\n")
	b.WriteString("// Drop tables live in enemy_content.go; skill definitions in enemy_skill.go.\n\n")
	b.WriteString("package main\n\n")

	fmt.Fprintf(&b, "var PlayerStatsByLevel = [%d][%d]PlayerStats{\n", playerLevels, partySize)
	for lv := range playerLevels {
		b.WriteString("\t{\n")
		for slot := range partySize {
			r := playerStats[lv*partySize+slot]
			if r.Level != lv+1 || r.Slot != slot+1 {
				return nil, fmt.Errorf("player stats out of order: expected Level=%d Slot=%d, got Level=%d Slot=%d (rows must be sorted by Level then Slot)", lv+1, slot+1, r.Level, r.Slot)
			}
			fmt.Fprintf(&b, "\t\t{HP: %d, MP: %d, PhysAtk: %d, MagicAtk: %d, PhysDef: %d, MagicDef: %d, Spd: %d, Luck: %d},\n",
				r.HP, r.MP, r.PhysAtk, r.MagicAtk, r.PhysDef, r.MagicDef, r.Spd, r.Luck)
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "var PlayerExpToNextByLevel = [%d]int{\n", playerLevels)
	for _, v := range playerExp {
		fmt.Fprintf(&b, "\t%d,\n", v)
	}
	b.WriteString("}\n\n")

	b.WriteString("var EnemyDatabase = []EnemyStats{\n")
	for _, e := range enemies {
		elem, err := elemConst(e.Element)
		if err != nil {
			return nil, fmt.Errorf("enemy %q: %w", e.Name, err)
		}
		fmt.Fprintf(&b, "\t{Name: %q, Lv: %d, Exp: %d, HP: %d, MP: %d, PhysAtk: %d, MagicAtk: %d, PhysDef: %d, MagicDef: %d, Spd: %d, SP: %d, Element: %s, ElementResist: [%d]int{%d, %d, %d, %d},\n",
			e.Name, e.Lv, e.Exp, e.HP, e.MP, e.PhysAtk, e.MagicAtk, e.PhysDef, e.MagicDef, e.Spd, e.SP, elem,
			elementalTypeCount, e.ResistFire, e.ResistLightning, e.ResistIce, e.ResistWind)
		fmt.Fprintf(&b, "\t\tDrops: enemyDrops[%q]", e.Name)
		if len(e.Skills) > 0 {
			fmt.Fprintf(&b, ", SkillIDs: %s", skillIDsLiteral(e.Skills))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n\n")

	b.WriteString("var BossDatabase = map[string]EnemyStats{\n")
	for _, boss := range bosses {
		idx, err := bossIndex(boss.Key)
		if err != nil {
			return nil, err
		}
		elem, err := elemConst(boss.Element)
		if err != nil {
			return nil, fmt.Errorf("boss %q: %w", boss.Key, err)
		}
		fmt.Fprintf(&b, "\t%q: {Name: BossNames[%d], Lv: %d, Exp: %d, HP: %d, MP: %d, PhysAtk: %d, MagicAtk: %d, PhysDef: %d, MagicDef: %d, Spd: %d, SP: %d, Element: %s, ElementResist: [%d]int{%d, %d, %d, %d},\n",
			boss.Key, idx, boss.Lv, boss.Exp, boss.HP, boss.MP, boss.PhysAtk, boss.MagicAtk, boss.PhysDef, boss.MagicDef, boss.Spd, boss.SP, elem,
			elementalTypeCount, boss.ResistFire, boss.ResistLightning, boss.ResistIce, boss.ResistWind)
		fmt.Fprintf(&b, "\t\tDrops: bossDrops[%q]", boss.Key)
		if len(boss.Skills) > 0 {
			fmt.Fprintf(&b, ", SkillIDs: %s", skillIDsLiteral(boss.Skills))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")

	out, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w (this is a bug in genstats)", err)
	}
	return out, nil
}
