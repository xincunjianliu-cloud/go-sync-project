package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ObjectiveType は目的地の種類
type ObjectiveType int

const (
	ObjectiveBoss ObjectiveType = iota
	ObjectiveNPC
	ObjectiveStoryEvent
)

// ObjectiveLocation はマップスキャンで判明した「目的地の実座標」
type ObjectiveLocation struct {
	MapPath string
	X, Y    float64
}

// ObjectiveDef はゲーム進行上の「目的地候補」定義。
// ここを書き換えるだけで、表示する目的地の内容・優先順位をコントロールできる。
type ObjectiveDef struct {
	ID   string
	Type ObjectiveType

	// IsActive: この目的地が「今アクティブか」を判定する関数。
	// trueを返した最初の（先頭の）ものが現在の目的地として採用される。
	IsActive func(g *Game) bool
}

// ★★★ ここが「順番」と「内容」を決める場所 ★★★
var objectiveQueue = []ObjectiveDef{
	{
		ID:   "boss_1",
		Type: ObjectiveBoss,
		IsActive: func(g *Game) bool {
			return !isBossDefeated(g, 1)
		},
	},
	{
		ID:   "boss_2",
		Type: ObjectiveBoss,
		IsActive: func(g *Game) bool {
			return isBossDefeated(g, 1) && !isBossDefeated(g, 2)
		},
	},
	{
		ID:   "boss_3",
		Type: ObjectiveBoss,
		IsActive: func(g *Game) bool {
			return isBossDefeated(g, 2) && !isBossDefeated(g, 3)
		},
	},
	{
		ID:   "boss_4",
		Type: ObjectiveBoss,
		IsActive: func(g *Game) bool {
			return isBossDefeated(g, 3) && !isBossDefeated(g, 4)
		},
	},
}

// UpdateObjective は現在アクティブな目的地IDを再計算してGameにセットする。
func (g *Game) UpdateObjective() {
	for _, def := range objectiveQueue {
		if def.IsActive(g) {
			g.CurrentObjectiveID = def.ID
			return
		}
	}
	g.CurrentObjectiveID = ""
}

// CurrentObjectiveLocation は現在の目的地の実座標を返す（無ければ ok=false）
func (g *Game) CurrentObjectiveLocation() (ObjectiveLocation, bool) {
	if g.CurrentObjectiveID == "" {
		return ObjectiveLocation{}, false
	}
	loc, ok := objectiveLocationIndex[g.CurrentObjectiveID]
	return loc, ok
}

// ============================================================
// マップ一括スキャン（起動時に1回実行）
// ============================================================

var objectiveLocationIndex = map[string]ObjectiveLocation{}

// mapDoor はマップ間接続グラフの1エッジ
type mapDoor struct {
	FromMap      string
	ToMap        string
	DoorX, DoorY float64
}

var mapConnectionGraph = map[string][]mapDoor{}

// allMapPaths: プロジェクト内の全マップファイル一覧。
// マップを追加したらここにも追記する。
var allMapPaths = []string{
	"assets/maps/School_Map_1.tmj",
	"assets/maps/ダンジョンA.tmj",
}

// BuildObjectiveAndMapIndex は全マップを走査し、
// ①目的地IDインデックス ②マップ間接続グラフ を構築する。
func BuildObjectiveAndMapIndex() error {
	objectiveLocationIndex = map[string]ObjectiveLocation{}
	mapConnectionGraph = map[string][]mapDoor{}

	for _, mapPath := range allMapPaths {
		data, err := os.ReadFile(mapPath)
		if err != nil {
			return fmt.Errorf("マップ読み込み失敗 %s: %w", mapPath, err)
		}
		var tmap TiledMap
		if err := json.Unmarshal(data, &tmap); err != nil {
			return fmt.Errorf("マップ解析失敗 %s: %w", mapPath, err)
		}

		for _, layer := range tmap.Layers {
			if !strings.HasPrefix(layer.Name, "events") {
				continue
			}
			for _, obj := range layer.Objects {
				var objectiveID, targetMap string

				for _, prop := range obj.Properties {
					switch strings.ToLower(prop.Name) {
					case "objectiveid":
						objectiveID = fmt.Sprintf("%v", prop.Value)
					case "targetmap":
						targetMap = fmt.Sprintf("%v", prop.Value)
					}
				}

				// NPC/イベント：objectiveIdが付いていれば登録
				if objectiveID != "" {
					objectiveLocationIndex[objectiveID] = ObjectiveLocation{
						MapPath: mapPath, X: obj.X, Y: obj.Y,
					}
				}

				// ── マップ間接続グラフ登録（ドア） ──
				if targetMap != "" {
					mapConnectionGraph[mapPath] = append(mapConnectionGraph[mapPath], mapDoor{
						FromMap: mapPath,
						ToMap:   targetMap,
						DoorX:   obj.X,
						DoorY:   obj.Y,
					})
				}
			}
		}
	}
	return nil
}

// FindNextDoorTowards は現在マップから目的地マップへ向かう最初のドアをBFSで探す。
func FindNextDoorTowards(fromMap, targetMap string) (door mapDoor, ok bool) {
	if fromMap == targetMap {
		return mapDoor{}, false
	}

	type queueItem struct {
		mapPath   string
		firstDoor mapDoor
	}
	visited := map[string]bool{fromMap: true}
	queue := []queueItem{}

	for _, d := range mapConnectionGraph[fromMap] {
		queue = append(queue, queueItem{mapPath: d.ToMap, firstDoor: d})
		visited[d.ToMap] = true
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.mapPath == targetMap {
			return item.firstDoor, true
		}
		for _, d := range mapConnectionGraph[item.mapPath] {
			if !visited[d.ToMap] {
				visited[d.ToMap] = true
				queue = append(queue, queueItem{mapPath: d.ToMap, firstDoor: item.firstDoor})
			}
		}
	}
	return mapDoor{}, false
}
