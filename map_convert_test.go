//go:build !js

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// 名札を手で書いた従来のしかけを、Tiledのクラス(tiled_classes.go)に書き直す。
//
//	PowerShell: $env:CONVERT_MAPS_TO_CLASSES=1; go test -run TestConvertMapsToClasses .; Remove-Item Env:CONVERT_MAPS_TO_CLASSES
//
// 書き直す前と後で、ゲームが読み込んだ結果(名札)が同じになることを毎回確かめる。
func TestConvertMapsToClasses(t *testing.T) {
	write := os.Getenv("CONVERT_MAPS_TO_CLASSES") != ""
	for _, mapPath := range allMapFiles(t) {
		before, err := os.ReadFile(mapPath)
		if err != nil {
			t.Fatal(err)
		}
		after, skipped, err := convertMapToClasses(before, mapPath)
		if err != nil {
			t.Fatalf("%s: %v", mapPath, err)
		}
		for _, s := range skipped {
			t.Logf("%s: クラスにできず、そのまま残したもの: %s", mapPath, s)
		}
		if diff := compareLoadedMaps(before, after, mapPath); diff != "" {
			t.Fatalf("%s: 書き直すとゲームの動きが変わります: %s", mapPath, diff)
		}
		if write && !bytes.Equal(before, after) {
			if err := os.WriteFile(mapPath, after, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s をクラスに書き直しました", mapPath)
		}
	}
}

func convertMapToClasses(data []byte, mapPath string) ([]byte, []string, error) {
	var m map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		return nil, nil, err
	}
	var skipped []string

	if cls, _ := m["class"].(string); cls == "" {
		props := propsOf(m["properties"])
		var members []any
		var rest []any
		for _, p := range props {
			name, val := strings.ToLower(p["name"].(string)), fmt.Sprint(p["value"])
			switch name {
			case "displayname":
				members = appendMember(members, "地名", "string", "", val, "")
			case "bgm":
				members = appendMember(members, "曲", "string", "曲", bgmLabelFromKey(val), "フィールド1")
			case "battlebg":
				if val != "" {
					members = appendMember(members, "戦闘背景", "string", "戦闘背景", val, enumDefault)
				}
			case "autoheal":
				if b, _ := propBool(p["value"]); b {
					members = appendMember(members, "入ると全回復", "bool", "", true, false)
				}
			default:
				rest = append(rest, p)
			}
		}
		m["class"] = "マップ設定"
		m["properties"] = append(members, rest...)
		if len(m["properties"].([]any)) == 0 {
			delete(m, "properties")
		}
	}

	layers, _ := m["layers"].([]any)
	for _, l := range layers {
		layer := l.(map[string]any)
		name, _ := layer["name"].(string)
		if layer["type"] == "tilelayer" {
			props := propsOf(layer["properties"])
			var rest []any
			lever := false
			for _, p := range props {
				if strings.EqualFold(p["name"].(string), "leveropen") {
					lever, _ = propBool(p["value"])
					continue
				}
				rest = append(rest, p)
			}
			if lever {
				layer["class"] = "レバーで出るレイヤー"
				if len(rest) == 0 {
					delete(layer, "properties")
				} else {
					layer["properties"] = rest
				}
			}
			continue
		}
		if !strings.HasPrefix(name, "events") {
			continue
		}
		objs, _ := layer["objects"].([]any)
		for _, o := range objs {
			obj := o.(map[string]any)
			if cls, _ := obj["type"].(string); cls != "" {
				continue
			}
			cls, members, ok := legacyToClass(obj)
			if !ok {
				if len(propsOf(obj["properties"])) > 0 {
					skipped = append(skipped, fmt.Sprintf("id%v", obj["id"]))
				}
				continue
			}
			obj["type"] = cls
			if len(members) == 0 {
				delete(obj, "properties")
			} else {
				obj["properties"] = members
			}
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(m); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), skipped, nil
}

func propsOf(v any) []map[string]any {
	list, _ := v.([]any)
	var out []map[string]any
	for _, p := range list {
		if pm, ok := p.(map[string]any); ok {
			out = append(out, pm)
		}
	}
	return out
}

// appendMember は既定値と違うときだけ欄を書く(Tiledと同じ)。
func appendMember(list []any, name, typ, enum string, value, def any) []any {
	if fmt.Sprint(value) == fmt.Sprint(def) {
		return list
	}
	p := map[string]any{"name": name, "type": typ, "value": value}
	if enum != "" {
		p["propertytype"] = enum
	}
	return append(list, p)
}

func bgmLabelFromKey(key string) string {
	for _, b := range bgmLabels {
		if b.Key == key {
			return b.Label
		}
	}
	return key
}

func itemLabelFromID(id string) string {
	for _, it := range ItemDatabase {
		if it.ID == id {
			return it.Name
		}
	}
	return id
}

// legacyToClass は従来の名札から、クラスと欄を作る。作れないものは ok=false。
func legacyToClass(obj map[string]any) (string, []any, bool) {
	p := map[string]string{}
	for _, prop := range propsOf(obj["properties"]) {
		p[strings.ToLower(prop["name"].(string))] = fmt.Sprint(prop["value"])
	}
	name, _ := obj["name"].(string)
	var mem []any
	add := func(n, typ, enum string, v, def any) { mem = appendMember(mem, n, typ, enum, v, def) }
	known := func(keys ...string) bool {
		allowed := map[string]bool{}
		for _, k := range keys {
			allowed[k] = true
		}
		for k := range p {
			if !allowed[k] {
				return false
			}
		}
		return true
	}
	boss := func(v string) string {
		if v == "" {
			return enumNone
		}
		return v
	}
	objective := func() {
		if p["objectiveid"] != "" {
			add("目的地の名前", "string", "", p["objectiveid"], "")
			if n, err := strconv.Atoi(p["objectiveorder"]); err == nil {
				add("目的地の順番", "int", "", n, 0)
			}
		}
	}
	textOrStory := func() {
		if id, ok := strings.CutPrefix(p["text"], storyTextPrefix); ok {
			add("会話データ", "string", "会話", id, enumNone)
		} else {
			add("セリフ", "string", "", p["text"], "")
		}
	}

	switch {
	case p["targetmap"] != "":
		if !known("targetmap", "targetpoint", "requireboss") {
			return "", nil, false
		}
		add("行き先マップ", "file", "", path.Base(p["targetmap"]), "")
		add("着地点", "string", "", p["targetpoint"], "")
		add("倒すまで通れないボス", "string", "ボス", boss(p["requireboss"]), enumNone)
		return "ドア", mem, true
	case len(p) == 0 && name != "", name != "" && known("dir"):
		if d, err := strconv.Atoi(p["dir"]); err == nil && d >= 0 && d < len(dirLabels) {
			add("向き", "string", "向き", dirLabels[d], enumKeep)
		}
		return "着地点", mem, true
	case p["type"] == "" && p["text"] == "" && p["lever"] != "":
		if !known("lever", "passable") {
			return "", nil, false
		}
		add("レバーの名前", "string", "", p["lever"], "")
		add("見た目だけ", "bool", "", p["passable"] == "false", false)
		return "レバーの壁", mem, true
	}

	text := p["text"]
	switch p["type"] {
	case evTypeEvent:
		switch {
		case strings.HasPrefix(text, chestKeyTextPrefix) && known("type", "text"):
			add("鍵の名前", "string", "", strings.TrimPrefix(text, chestKeyTextPrefix), "")
			return "鍵の宝箱", mem, true
		case strings.HasPrefix(text, chestTextPrefix) && known("type", "text"):
			add("中身", "string", "アイテム", itemLabelFromID(strings.TrimPrefix(text, chestTextPrefix)), enumUnset)
			return "宝箱", mem, true
		case text == evTextWall && p["lever"] != "" && known("type", "text", "lever", "passable"):
			add("レバーの名前", "string", "", p["lever"], "")
			add("見た目だけ", "bool", "", p["passable"] == "false", false)
			return "レバーの壁", mem, true
		case text == evTextWall && known("type", "text", "keys"):
			add("必要な鍵", "string", "", p["keys"], "")
			return "鍵の壁", mem, true
		case text == evTextLever && known("type", "text", "id", "oneway"):
			add("レバーの名前", "string", "", p["id"], "")
			add("一度きり", "bool", "", p["oneway"] == "true", false)
			return "レバー", mem, true
		case text == evTextBlock && known("type", "text", "id"):
			add("ブロックの名前", "string", "", p["id"], "")
			return "ブロック", mem, true
		case text == evTextBlockSpot && known("type", "text", "id"):
			add("置き場の名前", "string", "", p["id"], "")
			return "ブロック置き場", mem, true
		case text == evTextBlockDoor && known("type", "text", "spots"):
			add("置き場の名前", "string", "", p["spots"], "")
			return "ブロック扉", mem, true
		case !strings.HasPrefix(text, "event_") || strings.HasPrefix(text, storyTextPrefix):
			if !known("type", "text", "repeattext", "objectiveid", "objectiveorder", "bossid") {
				return "", nil, false
			}
			textOrStory()
			add("2回目からのセリフ", "string", "", p["repeattext"], "")
			objective()
			if p["objectiveid"] != "" {
				add("達成するボス", "string", "ボス", boss(p["bossid"]), enumNone)
			} else if p["bossid"] != "" {
				return "", nil, false
			}
			return "会話", mem, true
		}
	case "trigger":
		if !known("type", "text", "route", "bossid", "objectiveid", "objectiveorder") {
			return "", nil, false
		}
		textOrStory()
		add("ボス戦", "string", "ボス", boss(p["bossid"]), enumNone)
		add("歩く道順", "string", "", p["route"], "")
		objective()
		return "イベント", mem, true
	case "boss":
		if !known("type", "text", "bossid", "objectiveid") {
			return "", nil, false
		}
		add("ボス番号", "string", "ボス", boss(p["bossid"]), enumNone)
		add("目的地の名前", "string", "", p["objectiveid"], "")
		return "ボス", mem, true
	case "enemy":
		if !known("type", "text", "maxcount") {
			return "", nil, false
		}
		add("出る敵", "string", "敵", text, "")
		if n, err := strconv.Atoi(p["maxcount"]); err == nil {
			add("一度に出る最大数", "int", "", n, 1)
		}
		return "敵が出る場所", mem, true
	case "darkness":
		if !known("type", "text", "lever") {
			return "", nil, false
		}
		add("明るくするレバー", "string", "", p["lever"], "")
		return "暗い場所", mem, true
	}
	return "", nil, false
}

// compareLoadedMaps は2つのマップを読み込んだ結果(名札)を比べる。
func compareLoadedMaps(a, b []byte, mapPath string) string {
	load := func(data []byte) (TiledMap, error) {
		var tm TiledMap
		if err := json.Unmarshal(data, &tm); err != nil {
			return tm, err
		}
		applyTiledClasses(&tm, mapPath)
		return tm, nil
	}
	ta, err := load(a)
	if err != nil {
		return err.Error()
	}
	tb, err := load(b)
	if err != nil {
		return err.Error()
	}
	norm := func(props map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range props {
			switch {
			case v == "":
			case k == "maxcount" && v == "1", k == "objectiveorder" && v == "0",
				k == "bgm" && v == "field1", k == "passable" && v != "false",
				k == "oneway" && v != "true", k == "autoheal" && v != "true":
			default:
				out[k] = v
			}
		}
		return out
	}
	mapProps := func(tm TiledMap) map[string]string {
		m := map[string]string{}
		for _, p := range tm.Properties {
			m[strings.ToLower(p.Name)] = fmt.Sprint(p.Value)
		}
		if tm.mapAutoHeal() {
			m["autoheal"] = "true"
		} else {
			delete(m, "autoheal")
		}
		return norm(m)
	}
	if x, y := mapProps(ta), mapProps(tb); !reflect.DeepEqual(x, y) {
		return fmt.Sprintf("マップ設定 %v → %v", x, y)
	}
	if len(ta.Layers) != len(tb.Layers) {
		return "レイヤーの数"
	}
	for i := range ta.Layers {
		la, lb := ta.Layers[i], tb.Layers[i]
		if isLeverOpenLayer(la) != isLeverOpenLayer(lb) {
			return fmt.Sprintf("レイヤー%q の leveropen", la.Name)
		}
		if len(la.Objects) != len(lb.Objects) {
			return fmt.Sprintf("レイヤー%q のオブジェクト数", la.Name)
		}
		for j := range la.Objects {
			oa, ob := la.Objects[j], lb.Objects[j]
			pa, pb := objProps(oa), objProps(ob)
			// leverだけの壁と type=event, text=event_wall 付きの壁は同じもの。
			for _, p := range []map[string]string{pa, pb} {
				if isLeverControlledWallObj(p) {
					p["type"], p["text"] = evTypeEvent, evTextWall
				}
			}
			if b, ok := objPropBool(oa, "oneway"); ok && b {
				pa["oneway"] = "true"
			}
			if b, ok := objPropBool(ob, "oneway"); ok && b {
				pb["oneway"] = "true"
			}
			if x, y := norm(pa), norm(pb); !reflect.DeepEqual(x, y) || oa.Name != ob.Name {
				return fmt.Sprintf("id%d %v → %v", oa.ID, x, y)
			}
		}
	}
	return ""
}
