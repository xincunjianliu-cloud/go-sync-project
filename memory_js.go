//go:build js

package main

import "runtime/debug"

// webMemoryLimit はWeb版のGoのメモリ使用量の目安(ソフトリミット)。
// Goのガベージコレクタは通常、使用中のメモリの2倍程度まで回収を待つ。BGMの
// PCM(1曲あたり数十MB)や、GPUへ送る前の画像デコード結果のような大きな
// 一時データが重なると、その分だけメモリの山が高くなる。WebAssemblyは一度
// 確保したメモリをブラウザに返さないので、スマホではその山がそのまま
// タブの使用メモリになってしまう。この値に近づいたら早めに回収させて山を
// 抑える(超えても動作は止まらない、あくまで回収を早める目安)。
// 想定する使用量: BGMのPCMキャッシュ上限(maxCachedBGMBytes)＋効果音＋画像。
const webMemoryLimit = 320 << 20

func init() {
	debug.SetMemoryLimit(webMemoryLimit)
}
