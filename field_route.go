package main

import (
	"fmt"
	"strconv"
	"strings"
)

// イベントの「歩く道順」(route)の読み方。2通りの書き方ができる。
//
//	上4,右2       … 向き(下・左・右・上)+ マス数。マップを作る人向け
//	up128,right64 … 向き(down/left/right/up)+ ピクセル(前からの書き方)
//
// 区切りは半角の「,」でも全角の「、」「，」でもよい。数字は全角でもよい。
var routeDirs = []struct {
	prefix string
	dir    int
	tiles  bool // true ならマス数、false ならピクセル
}{
	{"down", 0, false}, {"left", 1, false}, {"right", 2, false}, {"up", 3, false},
	{"下", 0, true}, {"左", 1, true}, {"右", 2, true}, {"上", 3, true},
}

const routeTileSize = 32

// parseRoute は道順を歩く手順にする。書き方が違えば、その理由を返す。
func parseRoute(route string) ([]MoveStep, string) {
	route = strings.NewReplacer("、", ",", "，", ",").Replace(route)
	if strings.TrimSpace(route) == "" {
		return nil, ""
	}
	var steps []MoveStep
	for step := range strings.SplitSeq(route, ",") {
		step = strings.TrimSpace(step)
		ok := false
		for _, d := range routeDirs {
			rest, found := strings.CutPrefix(step, d.prefix)
			if !found {
				continue
			}
			rest = strings.TrimSuffix(strings.TrimSpace(halfWidthDigits(rest)), "マス")
			dist, err := strconv.ParseFloat(rest, 64)
			if err != nil || dist < 0 {
				break
			}
			if d.tiles {
				dist *= routeTileSize
			}
			steps = append(steps, MoveStep{Dir: d.dir, Dist: dist})
			ok = true
			break
		}
		if !ok {
			return nil, fmt.Sprintf("%q は書き方が違います。向き(下・左・右・上)とマス数をカンマでつないでください(例: 上4,右2)", step)
		}
	}
	return steps, ""
}

// halfWidthDigits は全角の数字を半角にする。
func halfWidthDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		if r == '．' {
			return '.'
		}
		return r
	}, s)
}
