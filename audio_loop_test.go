package main

import "testing"

func TestCancelPendingLoopSwitch(t *testing.T) {
	a := &AudioManager{
		waitingLoopSwitch: true,
		pendingLoopPath:   bgmVictoryLoop,
	}

	a.cancelPendingLoopSwitch()

	if a.waitingLoopSwitch {
		t.Fatal("waitingLoopSwitch = true, want false after a new BGM request")
	}
	if a.pendingLoopPath != "" {
		t.Fatalf("pendingLoopPath = %q, want empty after a new BGM request", a.pendingLoopPath)
	}
}

func TestFadeOutBGMCancelsPendingPlayback(t *testing.T) {
	started := false
	a := &AudioManager{
		pendingStart:    func() { started = true },
		pendingPaths:    []string{bgmTalkTense},
		pendingBGM:      bgmTalkTense,
		fadeOutActive:   true,
		fadeOutNextPath: bgmTalkTense,
	}

	a.FadeOutBGM(msgBGMFadeOut)
	a.tryPendingStart()

	if started {
		t.Fatal("pending BGM started after FadeOutBGM")
	}
	if a.pendingStart != nil || len(a.pendingPaths) != 0 || a.pendingBGM != "" {
		t.Fatal("pending playback was not cleared by FadeOutBGM")
	}
	if a.fadeOutActive || a.fadeInActive || a.fadeOutNextPath != "" {
		t.Fatal("fade state was not cleared when there was no active player")
	}
}

func TestChangingBGMVolumeCancelsOldFadeInTarget(t *testing.T) {
	a := &AudioManager{
		volume:       0.7,
		masterVolume: 0.7,
		fadeInActive: true,
		fadeInTarget: 0.49,
		fadeInSpeed:  0.49,
	}

	a.SetVolume(0.2)
	a.SetMasterVolume(0.5)

	if a.fadeInActive {
		t.Fatal("fadeInActive = true after an explicit volume change")
	}
	if got, want := a.effectiveBGMVolume(), 0.1; got != want {
		t.Fatalf("effective BGM volume = %v, want %v", got, want)
	}
}
