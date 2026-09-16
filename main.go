package main

import (
	"bytes"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func main() {
	fontData, err := loadAssetBytes("assets/fonts/PixelMplus10-Bold.ttf")
	if err != nil {
		log.Fatal("フォントファイルの読み込みに失敗しました: ", err)
	}

	source, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		log.Fatal("フォントソースの生成に失敗しました: ", err)
	}

	ebiten.SetWindowTitle("七不思議討滅録")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(960, 540)
	ebiten.SetTPS(60)

	g, err := NewGame(source)
	if err != nil {
		log.Fatal("ゲームの初期化に失敗しました: ", err)
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
