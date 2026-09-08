package main

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// サウンドのサンプルレート（プロジェクト全体で統一する）
const sampleRate = 44100

// ── アセットパス定義 ──────────────────────────────────────────────
// 置き場所は他の画像読み込み（assets/images/...）に合わせて assets/audio/ に統一。
// パスが違う場合はここだけ書き換えればよい。
const (
	bgmFieldSchool    = "assets/bgm/FIELD_School.mp3"    // フィールド（学校）BGM・ループ
	bgmBattleNormal   = "assets/bgm/NORMAL_BATTLE1.mp3"  // 通常戦闘BGM・ループ
	bgmBattleBoss     = "assets/bgm/BATTLE_BOSS1.mp3"    // ボス戦BGM・ループ
	bgmBattleEndIntro = "assets/bgm/BATTLE_End_into.mp3" // 勝利イントロ・1回再生
	bgmBattleEndLoop  = "assets/bgm/BATTLE_End_roop.mp3" // 勝利ループ（イントロの後）
	bgmMessage        = "assets/bgm/Message.mp3"         // 会話シーンBGM
)

// AudioManager はBGMの再生・停止・イントロ→ループの自動切り替えを管理する。
// 1つの *Game に対して1つだけ生成して使う想定。
type AudioManager struct {
	context *audio.Context

	bgmPlayer *audio.Player
	bgmName   string // 現在再生中のファイルパス（同じBGMの多重再生を防ぐため）
	volume    float64

	// イントロ→ループ切り替え用
	waitingLoopSwitch bool
	pendingLoopPath   string
	fadeInActive      bool
	fadeInTarget      float64
	fadeInSpeed       float64 // 1秒あたりの音量増加量
}

// NewAudioManager は AudioManager を初期化する。Game の生成時に1回だけ呼ぶこと。
// ↓ AudioManager側のコード
func NewAudioManager() *AudioManager {
	return &AudioManager{
		context: audio.NewContext(sampleRate),
		volume:  defaultBGMVolume, // ★数字ではなく定数名を入れる！
	}
}

// loadStreamPlayer はmp3ファイルを1回再生用のPlayerとして読み込む。
func (a *AudioManager) loadStreamPlayer(path string) (*audio.Player, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	d, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil, err
	}
	p, err := a.context.NewPlayer(d)
	if err != nil {
		return nil, err
	}
	p.SetVolume(a.volume)
	return p, nil
}

// loadLoopPlayer はmp3ファイルを無限ループ用のPlayerとして読み込む。
func (a *AudioManager) loadLoopPlayer(path string) (*audio.Player, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	d, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil, err
	}
	loop := audio.NewInfiniteLoop(d, d.Length())
	p, err := a.context.NewPlayer(loop)
	if err != nil {
		return nil, err
	}
	p.SetVolume(a.volume)
	return p, nil
}

// PlayBGM は指定したファイルをループ再生する。
// 既に同じBGMが再生中なら何もしない（シーン遷移ごとに呼んでも再生が途切れない）。
func (a *AudioManager) PlayBGM(path string) {
	if a == nil {
		return
	}
	if a.bgmName == path && a.bgmPlayer != nil && a.bgmPlayer.IsPlaying() {
		return
	}
	a.stopCurrent()

	p, err := a.loadLoopPlayer(path)
	if err != nil {
		fmt.Printf("警告: BGM再生に失敗しました(%s): %v\n", path, err)
		return
	}
	p.Play()
	a.bgmPlayer = p
	a.bgmName = path
}

func (a *AudioManager) PlayBGMFadeIn(path string, duration float64) {
	if a == nil {
		return
	}
	if a.bgmName == path && a.bgmPlayer != nil && a.bgmPlayer.IsPlaying() {
		return
	}
	a.stopCurrent()

	p, err := a.loadLoopPlayer(path)
	if err != nil {
		fmt.Printf("警告: BGM再生に失敗しました(%s): %v\n", path, err)
		return
	}
	p.SetVolume(0) // 音量0から開始
	p.Play()
	a.bgmPlayer = p
	a.bgmName = path
	a.fadeInActive = true
	a.fadeInTarget = a.volume
	a.fadeInSpeed = a.volume / duration
}

// PlayBGMWithIntro はイントロ（1回再生）を再生し、終わったら自動的にloopPathへ切り替える。
// 勝利BGM（イントロ→ループ）のような演出に使う。
func (a *AudioManager) PlayBGMWithIntro(introPath, loopPath string) {
	if a == nil {
		return
	}
	a.stopCurrent()

	p, err := a.loadStreamPlayer(introPath)
	if err != nil {
		fmt.Printf("警告: イントロBGM再生に失敗しました(%s): %v\n", introPath, err)
		a.PlayBGM(loopPath) // イントロが読めない場合はループだけ再生
		return
	}
	p.Play()
	a.bgmPlayer = p
	a.bgmName = introPath
	a.waitingLoopSwitch = true
	a.pendingLoopPath = loopPath
}

// StopBGM は現在のBGMを停止する。
func (a *AudioManager) StopBGM() {
	if a == nil {
		return
	}
	a.stopCurrent()
}

func (a *AudioManager) stopCurrent() {
	if a.bgmPlayer != nil {
		a.bgmPlayer.Close()
		a.bgmPlayer = nil
	}
	a.bgmName = ""
	a.waitingLoopSwitch = false
	a.pendingLoopPath = ""
}

// SetVolume は今後再生するBGMの音量(0.0〜1.0)を設定する。
// 既に再生中のBGMにも即時反映する。
func (a *AudioManager) SetVolume(v float64) {
	if a == nil {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	a.volume = v
	if a.bgmPlayer != nil {
		a.bgmPlayer.SetVolume(v)
	}
}

// Update は毎フレーム Game.Update() から呼ぶこと。
// イントロ再生が終わったタイミングを監視し、ループBGMへ自動的に切り替える。
func (a *AudioManager) Update(dt float64) {
	if a == nil {
		return
	}
	if a.waitingLoopSwitch && a.bgmPlayer != nil && !a.bgmPlayer.IsPlaying() {
		loopPath := a.pendingLoopPath
		a.waitingLoopSwitch = false
		a.PlayBGM(loopPath)
	}
	if a.fadeInActive && a.bgmPlayer != nil {
		cur := a.bgmPlayer.Volume()
		cur += a.fadeInSpeed * dt
		if cur >= a.fadeInTarget {
			cur = a.fadeInTarget
			a.fadeInActive = false
		}
		a.bgmPlayer.SetVolume(cur)
	}
}
