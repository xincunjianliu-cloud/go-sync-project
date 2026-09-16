package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ObjectiveType int

const (
	ObjectiveBoss ObjectiveType = iota
	ObjectiveNPC
	ObjectiveStoryEvent
)

type ObjectiveLocation struct {
	MapPath string
	X, Y    float64
}

type ObjectiveDef struct {
	ID   string
	Type ObjectiveType

	IsActive func(g *Game) bool
}

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

func (g *Game) UpdateObjective() {
	for _, def := range objectiveQueue {
		if def.IsActive(g) {
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

var objectiveLocationIndex = map[string]ObjectiveLocation{}

type mapDoor struct {
	FromMap      string
	ToMap        string
	DoorX, DoorY float64
}

var mapConnectionGraph = map[string][]mapDoor{}

var allMapPaths = []string{
	"assets/maps/School_Map_1.tmj",
	"assets/maps/ダンジョンA.tmj",
}

func BuildObjectiveAndMapIndex() error {
	objectiveLocationIndex = map[string]ObjectiveLocation{}
	mapConnectionGraph = map[string][]mapDoor{}

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
				var objectiveID, targetMap string

				for _, prop := range obj.Properties {
					switch strings.ToLower(prop.Name) {
					case "objectiveid":
						objectiveID = fmt.Sprintf("%v", prop.Value)
					case "targetmap":
						targetMap = fmt.Sprintf("%v", prop.Value)
					}
				}

				if objectiveID != "" {
					objectiveLocationIndex[objectiveID] = ObjectiveLocation{
						MapPath: mapPath, X: obj.X, Y: obj.Y,
					}
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
