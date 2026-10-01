package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowTitle("七不思議討滅録")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(gameWidth, gameHeight)
	ebiten.SetWindowSizeLimits(windowMinWidth, windowMinHeight, -1, -1)
	ebiten.SetTPS(60)
	// ウィンドウから離れたらゲームを止める（ATB戦闘がよそ見中に進まないように）。
	ebiten.SetRunnableOnUnfocused(false)

	// フォントやタイトル画面の画像はNewGame内でバックグラウンド読み込みが
	// 始まり、ウィンドウは読み込み中もローディング表示を出しながらすぐ開く。
	g := NewGame()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
