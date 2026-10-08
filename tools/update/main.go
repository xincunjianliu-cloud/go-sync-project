// Command update はスプレッドシートの取り込みとTiledの一覧の作り直しを
// まとめて行う。作業フォルダ(go.mod のある場所)で:
//
//	go run ./tools/update
//
// 1. 会話を取り込む(tools/dialoguegen)
// 2. 強さを取り込む(tools/genstats)
// 3. rpg.tiled-project の一覧(会話・敵・戦闘背景など)と、ゲームの最初のマップ
//    (start_map_generated.go)を作り直す
//
// 1か2が失敗しても残りは進め、最後に失敗したものをまとめて表示する。
package main

import (
	"fmt"
	"os"
	"os/exec"
)

type step struct {
	label string
	args  []string
	env   []string
}

func main() {
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Fprintln(os.Stderr, "作業フォルダ(go.mod のある場所)で実行してください")
		os.Exit(1)
	}
	steps := []step{
		{label: "会話の取り込み", args: []string{"run", "./tools/dialoguegen"}},
		{label: "強さの取り込み", args: []string{"run", "./tools/genstats"}},
		{label: "Tiledの一覧と最初のマップの作り直し", args: []string{"test", "-count=1", "-run", "^(TestTiledProjectUpToDate|TestStartMapUpToDate)$", "."},
			env: []string{"UPDATE_TILED_PROJECT=1"}},
	}
	var failed []string
	for i, s := range steps {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(steps), s.label)
		cmd := exec.Command("go", s.args...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		cmd.Env = append(os.Environ(), s.env...)
		if err := cmd.Run(); err != nil {
			failed = append(failed, s.label)
		}
	}
	fmt.Println()
	if len(failed) > 0 {
		fmt.Println("==== 失敗したもの ====")
		for _, f := range failed {
			fmt.Println("  ×", f)
		}
		fmt.Println("上に出ている理由を見て直し、もう一度 go run ./tools/update を打ってください。")
		fmt.Println("(成功したものはそのまま反映されています)")
		os.Exit(1)
	}
	fmt.Println("==== 全部できました ====")
	fmt.Println("GitHub Desktop で Commit → Push してください。")
}
