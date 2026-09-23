package main

import (
	"strings"
)

var wallKeyGroupsCache map[string][]string

func splitKeyNames(raw string) []string {
	var names []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			names = append(names, part)
		}
	}
	return names
}

func computeWallKeyGroups(maps []TiledMap) map[string][]string {
	groups := make(map[string][]string)
	for _, tmap := range maps {
		for _, layer := range tmap.Layers {
			if !strings.HasPrefix(layer.Name, "events") {
				continue
			}
			for _, obj := range layer.Objects {
				p := objProps(obj)
				if !isWallObj(p) {
					continue
				}
				names := splitKeyNames(p["keys"])
				for _, name := range names {
					groups[name] = names
				}
			}
		}
	}
	return groups
}

func loadWallKeyGroups() map[string][]string {
	if wallKeyGroupsCache != nil {
		return wallKeyGroupsCache
	}

	var maps []TiledMap
	for _, mapPath := range allMapPaths {
		tmap, err := loadTiledMap(mapPath)
		if err != nil {
			continue
		}
		maps = append(maps, tmap)
	}

	wallKeyGroupsCache = computeWallKeyGroups(maps)
	return wallKeyGroupsCache
}

func remainingKeysInGroup(names []string, owned map[string]int) int {
	remaining := 0
	for _, n := range names {
		if owned[n] < 1 {
			remaining++
		}
	}
	return remaining
}
