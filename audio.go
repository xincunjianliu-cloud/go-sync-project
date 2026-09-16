package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

const sampleRate = 44100

const (
	bgmFieldSchool    = "assets/bgm/FIELD_School.mp3"
	bgmBattleNormal   = "assets/bgm/NORMAL_BATTLE1.mp3"
	bgmBattleBoss     = "assets/bgm/BATTLE_BOSS1.mp3"
	bgmBattleEndIntro = "assets/bgm/BATTLE_End_into.mp3"
	bgmBattleEndLoop  = "assets/bgm/BATTLE_End_roop.mp3"
	bgmMessage        = "assets/bgm/Message.mp3"
)

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
}

func NewAudioManager() *AudioManager {
	return &AudioManager{
		context: audio.NewContext(sampleRate),
		volume:  defaultBGMVolume,
	}
}

func (a *AudioManager) loadStreamPlayer(path string) (*audio.Player, error) {
	f, err := loadAssetReader(path)
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

func (a *AudioManager) loadLoopPlayer(path string) (*audio.Player, error) {
	f, err := loadAssetReader(path)
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
