package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ObjectiveLocation struct {
	MapPath string
	X, Y    float64
}

// objectiveDef は目的地1件分の定義。専用オブジェクトを置く必要はなく、
// ボスやトリガーなど既存のマップオブジェクトに以下のプロパティを足すだけで
// 目的地として登録される(BuildObjectiveAndMapIndexが読み取る):
//   - objectiveId    (string, 必須) このオブジェクトの目的地ID
//   - objectiveOrder (int, 任意)    小さいほど先に目的地になる。省略時は0
//   - bossId         (int, 任意)    指定するとそのボスを倒すまでが達成条件になる
//
// bossIdが無い場合は「このオブジェクトに一度でも調べる/会話した」
// (SeenEvents)ことが達成条件になるので、ボス以外(NPC・調べるだけの
// イベントなど)もそのまま目的地にできる。
type objectiveDef struct {
	ID      string
	Order   int
	BossID  int
	HasBoss bool
	SeenKey string
}

func (def objectiveDef) isComplete(g *Game) bool {
	if def.HasBoss {
		return isBossDefeated(g, def.BossID)
	}
	return g.SeenEvents[def.SeenKey]
}

func (g *Game) UpdateObjective() {
	for _, def := range objectiveDefs {
		if !def.isComplete(g) {
			g.CurrentObjectiveID = def.ID
			return
		}
	}
	g.CurrentObjectiveID = ""
}

func (g *Game) CurrentObjectiveLocation() (ObjectiveLocation, bool) {
	if g.CurrentObjectiveID == "" {
		return ObjectiveLocation{}, false
	}
	loc, ok := objectiveLocationIndex[g.CurrentObjectiveID]
	return loc, ok
}

var objectiveDefs []objectiveDef
var objectiveLocationIndex = map[string]ObjectiveLocation{}

type mapDoor struct {
	FromMap      string
	ToMap        string
	DoorX, DoorY float64
}

var mapConnectionGraph = map[string][]mapDoor{}

var allMapPaths = allRegisteredMapPaths()

func BuildObjectiveAndMapIndex() error {
	objectiveLocationIndex = map[string]ObjectiveLocation{}
	mapConnectionGraph = map[string][]mapDoor{}
	objectiveDefs = nil

	for _, mapPath := range allMapPaths {
		data, err := loadAssetBytes(mapPath)
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
				p := objProps(obj)
				objectiveID := p["objectiveid"]
				targetMap := p["targetmap"]
				order, _ := objPropInt(obj, "objectiveorder")
				bossID, hasBoss := objPropInt(obj, "bossid")

				if objectiveID != "" {
					objectiveLocationIndex[objectiveID] = ObjectiveLocation{
						MapPath: mapPath,
						X:       obj.X + obj.Width/2,
						Y:       obj.Y + obj.Height/2,
					}
					objectiveDefs = append(objectiveDefs, objectiveDef{
						ID:      objectiveID,
						Order:   order,
						BossID:  bossID,
						HasBoss: hasBoss,
						SeenKey: chestKey(mapPath, obj),
					})
				}

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

	sort.SliceStable(objectiveDefs, func(i, j int) bool {
		return objectiveDefs[i].Order < objectiveDefs[j].Order
	})

	return nil
}

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
