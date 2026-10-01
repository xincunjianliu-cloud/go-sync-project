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
	// PC版はウィンドウから離れたらゲームを止める（ATB戦闘がよそ見中に進まない
	// ように）。Web版では止めない: 投稿サイトに埋め込まれていると、ゲーム部分を
	// クリックするまでフォーカスが無い扱いになり、読み込みごと止まってしまう
	// （タブを切り替えたときは、ブラウザが自動で止める）。
	if !isWebBuild {
		ebiten.SetRunnableOnUnfocused(false)
	}

	// フォントやタイトル画面の画像はNewGame内でバックグラウンド読み込みが
	// 始まり、ウィンドウは読み込み中もローディング表示を出しながらすぐ開く。
	g := NewGame()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
