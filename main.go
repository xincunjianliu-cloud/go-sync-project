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
	// PC版とホーム画面アプリは、画面から離れたらゲームと音声を止める
	// (ATB戦闘が進んだり、バックグラウンドでBGMが鳴り続けたりしないように)。
	// 通常のWeb版では、投稿サイトに埋め込まれた場合にゲーム部分をクリックする
	// までフォーカスが無く、読み込みごと止まってしまうため止めない。
	if !isWebBuild || runningAsInstalledApp() {
		ebiten.SetRunnableOnUnfocused(false)
	}

	// フォントやタイトル画面の画像はNewGame内でバックグラウンド読み込みが
	// 始まり、ウィンドウは読み込み中もローディング表示を出しながらすぐ開く。
	parseDevFlags()
	g := NewGame()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
