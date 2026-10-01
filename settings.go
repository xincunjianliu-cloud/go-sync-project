package main

import (
	"encoding/json"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const settingsFilePath = "settings.json"

const windowResizeSettleTicks = 30

type GameSettings struct {
	BGMVolume      float64 `json:"bgmVolume"`
	SEVolume       float64 `json:"seVolume"`
	MasterVolume   float64 `json:"masterVolume"`
	MessageSpeed   int     `json:"messageSpeed"`
	Fullscreen     bool    `json:"fullscreen"`
	WindowWidth    int     `json:"windowWidth"`
	WindowHeight   int     `json:"windowHeight"`
	RememberCursor bool    `json:"rememberCursor"`
}

func LoadSettings() GameSettings {
	s := GameSettings{
		BGMVolume:      defaultBGMVolume,
		SEVolume:       defaultSEVolume,
		MasterVolume:   defaultMasterVolume,
		MessageSpeed:   defaultMessageSpeed,
		Fullscreen:     defaultFullscreen,
		WindowWidth:    defaultWindowWidth,
		WindowHeight:   defaultWindowHeight,
		RememberCursor: defaultRememberCursor,
	}
	data, err := readRuntimeFile(settingsFilePath)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	if s.WindowWidth <= 0 || s.WindowHeight <= 0 {
		s.WindowWidth = defaultWindowWidth
		s.WindowHeight = defaultWindowHeight
	}
	return s
}

func SaveSettings(s GameSettings) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = writeRuntimeFile(settingsFilePath, data)
}

const (
	// windowMinWidth/Height はウィンドウを縮められる下限。これより小さいと
	// 文字が読めなくなる。
	windowMinWidth  = gameWidth / 2
	windowMinHeight = gameHeight / 2

	// windowMaxMonitorRatio は保存されたウィンドウサイズがモニターより
	// 大きいときに縮める上限（タスクバーやタイトルバーの分の余白を残す）。
	windowMaxMonitorRatio = 0.9

	// fullscreenSyncHoldTicks はフルスクリーン切り替えを要求してから、実際の
	// 状態との同期を待つ時間。Web版はブラウザが非同期に切り替えるため、
	// その間に「まだ切り替わっていない」と判断して設定を戻さないようにする。
	fullscreenSyncHoldTicks = 60
)

// fitWindowToMonitor は winW×winH を縦横比を保ったままモニターに収まる
// 大きさ（かつ下限以上）にする。モニターサイズが取れない環境ではそのまま。
func fitWindowToMonitor(winW, winH int) (int, int) {
	if mon := ebiten.Monitor(); mon != nil {
		monW, monH := mon.Size()
		maxW := float64(monW) * windowMaxMonitorRatio
		maxH := float64(monH) * windowMaxMonitorRatio
		if monW > 0 && monH > 0 && (float64(winW) > maxW || float64(winH) > maxH) {
			scale := min(maxW/float64(winW), maxH/float64(winH))
			winW = int(float64(winW) * scale)
			winH = int(float64(winH) * scale)
		}
	}
	return max(winW, windowMinWidth), max(winH, windowMinHeight)
}

func applyDisplayMode(fullscreen bool, winW, winH int) {
	ebiten.SetFullscreen(fullscreen)
	if fullscreen {
		return
	}

	ebiten.SetWindowSize(fitWindowToMonitor(winW, winH))

	mon := ebiten.Monitor()
	if mon == nil {
		return
	}
	actualW, actualH := ebiten.WindowSize()
	monW, monH := mon.Size()
	x := max((monW-actualW)/2, 0)
	y := max((monH-actualH)/2, 0)
	ebiten.SetWindowPosition(x, y)
}

// setFullscreen は画面モードを切り替える。オプション画面・初期化・
// ショートカットキーのすべてがここを通る。
func (g *Game) setFullscreen(fullscreen bool) {
	g.Fullscreen = fullscreen
	g.webFullscreenPending = false
	w, h := g.WindowWidth, g.WindowHeight
	if w <= 0 || h <= 0 {
		w, h = defaultWindowWidth, defaultWindowHeight
	}
	applyDisplayMode(fullscreen, w, h)
	if !fullscreen {
		g.WindowWidth, g.WindowHeight = ebiten.WindowSize()
		g.lastWindowW, g.lastWindowH = g.WindowWidth, g.WindowHeight
	}
	g.fullscreenSyncHold = fullscreenSyncHoldTicks
}

// initDisplayMode は起動時に保存された画面モードを反映する。Web版の
// ブラウザはユーザー操作なしのフルスクリーン化を拒否するので、最初の入力
// まで待つ（updateAutoFullscreen）。スマホは横向きで画面の高さが足りないので、
// 保存された設定に関係なく起動ごとに最初の操作で全画面にする。
func (g *Game) initDisplayMode() {
	if isWebBuild {
		if (g.Fullscreen || g.MobileMode) && fullscreenSupported() {
			g.Fullscreen = true
			g.webFullscreenPending = true
		}
		return
	}
	g.setFullscreen(g.Fullscreen)
}

// defaultFullscreenSetting は「初期設定に戻す」で使う画面モード。
// スマホのWeb版は全画面、それ以外はdefaultFullscreen。
func (g *Game) defaultFullscreenSetting() bool {
	if isWebBuild && g.MobileMode {
		return true
	}
	return defaultFullscreen
}

// displayModeOptionVisible はオプション画面に「画面モード」行を出すか。
// フルスクリーン非対応のブラウザ(iPhoneのSafariなど)では選んでも何も
// 起きないので出さない。
func (g *Game) displayModeOptionVisible() bool {
	return fullscreenSupported()
}

// showInstallHint はタイトル画面に「ホーム画面に追加すると全画面で遊べます」
// を出すか。全画面にできないスマホのブラウザ(主にiPhone)で、まだホーム画面
// から起動していないときだけ出す。
func (g *Game) showInstallHint() bool {
	return isWebBuild && g.MobileMode && !fullscreenSupported() && !runningAsInstalledApp()
}

// justActivatedByUser はブラウザがフルスクリーン要求を許可する種類の
// ユーザー操作がこのフレームにあったか。タッチは指を離した時(touchend)で
// ないとブラウザが操作と認めないので、押した時ではなく離した時を見る。
func justActivatedByUser() bool {
	if len(inpututil.AppendJustReleasedTouchIDs(nil)) > 0 ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		if !isFullscreenShortcutKey(k) && k != ebiten.KeyEscape {
			return true
		}
	}
	return false
}

// updateAutoFullscreen はWeb版で、起動後最初のユーザー操作をきっかけに
// フルスクリーンにする（initDisplayModeで予約したときだけ、1回きり）。
// その後に全画面を抜けた場合は、オプション画面かショートカットで戻す。
func (g *Game) updateAutoFullscreen() {
	if !g.webFullscreenPending {
		return
	}
	if ebiten.IsFullscreen() {
		g.webFullscreenPending = false
		return
	}
	if justActivatedByUser() {
		g.setFullscreen(true)
	}
}

func isAltKeyDown() bool {
	return ebiten.IsKeyPressed(ebiten.KeyAlt)
}

// isFullscreenShortcutKey はAltと組み合わせずに単独でフルスクリーンを
// 切り替えるキー。F4はRPGツクール製ゲームでおなじみの操作に合わせたもの。
func isFullscreenShortcutKey(k ebiten.Key) bool {
	return k == ebiten.KeyF4 || k == ebiten.KeyF11
}

// updateFullscreenShortcut はF4・F11・Alt+Enterで、どの画面からでも
// フルスクリーンを切り替える。
func (g *Game) updateFullscreenShortcut() {
	pressed := isAltKeyDown() && (inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter))
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		if isFullscreenShortcutKey(k) {
			pressed = true
		}
	}
	if !pressed || !g.displayModeOptionVisible() {
		return
	}
	g.setFullscreen(!g.Fullscreen)
	g.SaveGameSettings()
}

// syncFullscreenState は設定上の画面モードを実際の状態に合わせる。
// Web版では、起動時のフルスクリーン要求がブラウザに拒否されたときや、
// Escなどでブラウザ側からフルスクリーンが解除されたときにずれるため。
func (g *Game) syncFullscreenState() {
	if g.webFullscreenPending {
		// 最初の入力でフルスクリーンにするまでは、設定上の値を保つ。
		return
	}
	actual := ebiten.IsFullscreen()
	if g.fullscreenSyncHold > 0 {
		g.fullscreenSyncHold--
		if actual == g.Fullscreen {
			g.fullscreenSyncHold = 0
		}
		return
	}
	if actual == g.Fullscreen {
		return
	}
	g.Fullscreen = actual
	if !actual {
		if w, h := ebiten.WindowSize(); w > 0 && h > 0 {
			g.WindowWidth, g.WindowHeight = w, h
			g.lastWindowW, g.lastWindowH = w, h
		}
	}
	g.SaveGameSettings()
}

func (g *Game) updateWindowSizeTracking() {
	if g.Fullscreen {
		return
	}
	w, h := ebiten.WindowSize()
	if w <= 0 || h <= 0 {
		return
	}
	if w != g.lastWindowW || h != g.lastWindowH {
		g.lastWindowW, g.lastWindowH = w, h
		g.windowResizeSettleTimer = windowResizeSettleTicks
		return
	}
	if g.windowResizeSettleTimer > 0 {
		g.windowResizeSettleTimer--
		if g.windowResizeSettleTimer == 0 {
			g.WindowWidth, g.WindowHeight = w, h
			g.SaveGameSettings()
		}
	}
}

func (g *Game) SaveGameSettings() {
	vol := defaultBGMVolume
	seVol := defaultSEVolume
	masterVol := defaultMasterVolume
	if g.Audio != nil {
		vol = g.Audio.volume
		seVol = g.Audio.seVolume
		masterVol = g.Audio.masterVolume
	}
	SaveSettings(GameSettings{
		BGMVolume:      vol,
		SEVolume:       seVol,
		MasterVolume:   masterVol,
		MessageSpeed:   g.MessageSpeed,
		Fullscreen:     g.Fullscreen,
		WindowWidth:    g.WindowWidth,
		WindowHeight:   g.WindowHeight,
		RememberCursor: g.RememberCursor,
	})
}
