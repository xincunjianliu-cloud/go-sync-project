package main

import "image/color"

// UIカラーパレット。ゲーム全体のテキスト・UI強調色はここで一元管理する。
// 色を変更したい場合はここだけを書き換えればよい。
var (
	// 通常テキスト・説明文・ヒント・セリフ・話者名など、地の文全般
	uiColorText = color.RGBA{255, 255, 255, 255}

	// 選択中・強調（カーソルが当たっている項目、決定可能な選択肢、コマンド選択中キャラの枠線など）
	uiColorSelect = color.RGBA{255, 220, 80, 255}

	// 無効・グレーアウト（MP不足、選択不可の項目など、一時的に選べない状態）
	uiColorDisabled = color.RGBA{140, 140, 140, 255}

	// 死亡・戦闘不能（HP0のキャラクター名など、回復しない限り無効な状態）
	uiColorDead = color.RGBA{90, 70, 70, 255}

	// 警告・ゲームオーバー系
	uiColorDanger = color.RGBA{255, 100, 100, 255}

	// ウィンドウ・パネル系の半透明な黒背景（選択肢ウィンドウ、確認ダイアログなど）
	uiColorPanelBg = color.NRGBA{0, 0, 0, 210}
)
