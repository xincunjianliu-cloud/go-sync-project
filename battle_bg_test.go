package main

import "testing"

// 戦闘背景は「ボスの専用背景 → マップのbattlebg → なし(battle_bg.png)」の
// 順で決まること。
func TestBattleBgKeyOrder(t *testing.T) {
	const withBg, withoutBg = "test/with_bg.tmj", "test/without_bg.tmj"
	tiledMapCacheMu.Lock()
	tiledMapCache[withBg] = TiledMap{Properties: []TiledProperty{{Name: "battlebg", Value: "school"}}}
	tiledMapCache[withoutBg] = TiledMap{}
	tiledMapCacheMu.Unlock()
	saved := bossBattleBg
	bossBattleBg = map[int]string{2: "boss_special"}
	t.Cleanup(func() {
		bossBattleBg = saved
		tiledMapCacheMu.Lock()
		delete(tiledMapCache, withBg)
		delete(tiledMapCache, withoutBg)
		tiledMapCacheMu.Unlock()
	})

	cases := []struct {
		mapPath, enemyType, want string
	}{
		{withBg, "enemy", "school"},
		{withoutBg, "enemy", ""},
		{withBg, "boss_1", "school"},       // 専用背景なし → マップの背景
		{withBg, "boss_2", "boss_special"}, // 専用背景が優先
		{withoutBg, "boss_2", "boss_special"},
		{"test/not_loaded.tmj", "enemy", ""},
	}
	for _, c := range cases {
		if got := battleBgKey(c.mapPath, c.enemyType); got != c.want {
			t.Errorf("battleBgKey(%q, %q) = %q, want %q", c.mapPath, c.enemyType, got, c.want)
		}
	}
}
