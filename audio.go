package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

const sampleRate = 44100

// bgmBufferSize はBGM再生バッファの長さ。スマホ(特にCPUが遅い端末)ではデフォルトの
// バッファが短く、mp3デコードが間に合わずノイズ(音割れ・ブツブツ音)が発生するため、
// BGMのようにレイテンシを気にしない用途では長めに確保して耐性を持たせる。
const bgmBufferSize = 500 * time.Millisecond

const (
	bgmTitle          = "assets/bgm/title.mp3"
	bgmField1         = "assets/bgm/field1.mp3"
	bgmField2         = "assets/bgm/field2.mp3"
	bgmField3         = "assets/bgm/field3.mp3"
	bgmField4         = "assets/bgm/field4.mp3"
	bgmBattleNormal   = "assets/bgm/battle_normal.mp3"
	bgmBattleBoss     = "assets/bgm/battle_boss.mp3"
	bgmBattleLastBoss = "assets/bgm/battle_lastboss.mp3"
	bgmVictoryIntro   = "assets/bgm/victory_intro.mp3"
	bgmVictoryLoop    = "assets/bgm/victory_loop.mp3"
	bgmEnding         = "assets/bgm/ending.mp3"
	bgmTalkPeaceful   = "assets/bgm/talk_peaceful.mp3"
	bgmTalkTense      = "assets/bgm/talk_tense.mp3"
	bgmTalkScary      = "assets/bgm/talk_scary.mp3"
	bgmTalkSad        = "assets/bgm/talk_sad.mp3"
)

// bgmByKey はマップ側(.tmjのbgmプロパティ)や会話JSON側(bgmキー)から
// 曲を指定する際に使う、キー文字列→実ファイルパスの対応表。
// 実データの差し替え時はこのファイルパスだけ書き換えればよく、
// マップ・会話データ側は曲名キーを変更する必要がない。
var bgmByKey = map[string]string{
	"title":           bgmTitle,
	"field1":          bgmField1,
	"field2":          bgmField2,
	"field3":          bgmField3,
	"field4":          bgmField4,
	"battle_normal":   bgmBattleNormal,
	"battle_boss":     bgmBattleBoss,
	"battle_lastboss": bgmBattleLastBoss,
	"ending":          bgmEnding,
	"talk_peaceful":   bgmTalkPeaceful,
	"talk_tense":      bgmTalkTense,
	"talk_scary":      bgmTalkScary,
	"talk_sad":        bgmTalkSad,
}

func resolveBGMKey(key string) (string, bool) {
	path, ok := bgmByKey[key]
	return path, ok
}

type AudioManager struct {
	context *audio.Context

	bgmPlayer *audio.Player
	bgmName   string
	volume    float64

	waitingLoopSwitch bool
	pendingLoopPath   string
	fadeInActive      bool
	fadeInTarget      float64
	fadeInSpeed       float64

	pcmCache map[string][]byte
}

func NewAudioManager() *AudioManager {
	return &AudioManager{
		context:  audio.NewContext(sampleRate),
		volume:   defaultBGMVolume,
		pcmCache: make(map[string][]byte),
	}
}

// decodePCM はmp3を最後まで読み切り、PCMデータを丸ごとメモリに展開する。
// ストリーミングデコード(Playerが再生しながら少しずつmp3をデコードする方式)は、
// スマホの非力なCPUだとデコードが再生に間に合わずバッファが枯渇し、
// ブツブツ音の原因になる。事前に全部デコードしておけば再生中は
// メモリからコピーするだけになり、デコード負荷による音切れがなくなる。
// デコード結果は曲ごとにキャッシュし、2回目以降の再生(戦闘開始・終了の
// 繰り返しなど)で重いデコードが毎回走らないようにする。
func (a *AudioManager) decodePCM(path string) (*bytes.Reader, int64, error) {
	if pcm, ok := a.pcmCache[path]; ok {
		return bytes.NewReader(pcm), int64(len(pcm)), nil
	}
	f, err := loadAssetReader(path)
	if err != nil {
		return nil, 0, err
	}
	d, err := mp3.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		return nil, 0, err
	}
	pcm, err := io.ReadAll(d)
	if err != nil {
		return nil, 0, err
	}
	a.pcmCache[path] = pcm
	return bytes.NewReader(pcm), int64(len(pcm)), nil
}

func (a *AudioManager) loadStreamPlayer(path string) (*audio.Player, error) {
	r, _, err := a.decodePCM(path)
	if err != nil {
		return nil, err
	}
	p, err := a.context.NewPlayer(r)
	if err != nil {
		return nil, err
	}
	p.SetBufferSize(bgmBufferSize)
	p.SetVolume(a.volume)
	return p, nil
}

func (a *AudioManager) loadLoopPlayer(path string) (*audio.Player, error) {
	r, length, err := a.decodePCM(path)
	if err != nil {
		return nil, err
	}
	loop := audio.NewInfiniteLoop(r, length)
	p, err := a.context.NewPlayer(loop)
	if err != nil {
		return nil, err
	}
	p.SetBufferSize(bgmBufferSize)
	p.SetVolume(a.volume)
	return p, nil
}

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
	p.SetVolume(0)
	p.Play()
	a.bgmPlayer = p
	a.bgmName = path
	a.fadeInActive = true
	a.fadeInTarget = a.volume
	a.fadeInSpeed = a.volume / duration
}

func (a *AudioManager) PlayBGMWithIntro(introPath, loopPath string) {
	if a == nil {
		return
	}
	a.stopCurrent()

	p, err := a.loadStreamPlayer(introPath)
	if err != nil {
		fmt.Printf("警告: イントロBGM再生に失敗しました(%s): %v\n", introPath, err)
		a.PlayBGM(loopPath)
		return
	}
	p.Play()
	a.bgmPlayer = p
	a.bgmName = introPath
	a.waitingLoopSwitch = true
	a.pendingLoopPath = loopPath
}

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
