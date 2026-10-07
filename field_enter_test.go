package main

import "testing"

// 戦闘から戻ったときは、地名を出さず、自動回復もしない。
// ドアなどで入ったときは両方する。
func TestRoomSceneAfterBattleSkipsBannerAndAutoHeal(t *testing.T) {
	tmap, err := loadTiledMap(startMapPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tmap.mapDisplayName(); !ok || !tmap.mapAutoHeal() {
		t.Skip("開始マップに displayname と autoheal が無いので確かめられない")
	}

	g := &Game{}
	g.ResetForNewGame()
	g.PlayerHP[0] = 1
	field, err := NewRoomSceneAfterBattle(g, startMapPath, 224, 640, 0)
	if err != nil {
		t.Fatal(err)
	}
	if field.mapNameBannerActive {
		t.Error("戦闘から戻ったのに地名が出ます")
	}
	if g.PlayerHP[0] != 1 {
		t.Errorf("戦闘から戻ったのに回復しました(HP=%d)", g.PlayerHP[0])
	}

	field, err = NewRoomScene(g, startMapPath, 0, 0, "start_point", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !field.mapNameBannerActive {
		t.Error("マップに入ったのに地名が出ません")
	}
	if g.PlayerHP[0] == 1 {
		t.Error("autohealのマップに入ったのに回復しません")
	}
}

// Tiledで種類をstringのまま書いた数字やtrue/falseも読めること。
func TestPropIntAndBoolAcceptStrings(t *testing.T) {
	for _, c := range []struct {
		v    any
		want int
		ok   bool
	}{{float64(3), 3, true}, {"3", 3, true}, {" 4 ", 4, true}, {"abc", 0, false}, {true, 0, false}} {
		if got, ok := propInt(c.v); got != c.want || ok != c.ok {
			t.Errorf("propInt(%#v) = %d, %v", c.v, got, ok)
		}
	}
	for _, c := range []struct {
		v      any
		want   bool
		wantOK bool
	}{{true, true, true}, {"true", true, true}, {"True", true, true}, {"false", false, true}, {"yes", false, false}, {float64(1), false, false}} {
		if got, ok := propBool(c.v); got != c.want || ok != c.wantOK {
			t.Errorf("propBool(%#v) = %v, %v", c.v, got, ok)
		}
	}
}
