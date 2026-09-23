package main

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
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

const (
	seDecide        = "assets/se/se_decide.mp3"
	seCancel        = "assets/se/se_cancel.mp3"
	seCursor        = "assets/se/se_cursor.mp3"
	seMenuToggle    = "assets/se/se_menu_toggle.mp3"
	seError         = "assets/se/se_error.mp3"
	seTextAdvance   = "assets/se/se_text_advance.mp3"
	seFootstep      = "assets/se/se_footstep.mp3"
	seDashFootstep  = "assets/se/se_dash_footstep.mp3"
	seDoor          = "assets/se/se_door.mp3"
	seSlidingDoor   = "assets/se/se_sliding_door.mp3"
	seLocker        = "assets/se/se_locker.mp3"
	seTreasureOpen  = "assets/se/se_treasure_open.mp3"
	seItemGet       = "assets/se/se_item_get.mp3"
	seSwitch        = "assets/se/se_switch.mp3"
	seLever         = "assets/se/se_lever.mp3"
	seWaterFlow     = "assets/se/se_water_flow.mp3"
	seWaterFull     = "assets/se/se_water_full.mp3"
	seWaterDrain    = "assets/se/se_water_drain.mp3"
	seEncounter     = "assets/se/se_encounter.mp3"
	seBattleStart   = "assets/se/se_battle_start.mp3"
	seAttackNormal  = "assets/se/se_attack_normal.mp3"
	seSkillAttack   = "assets/se/se_skill_attack.mp3"
	seSkillHeal     = "assets/se/se_skill_heal.mp3"
	seCritical      = "assets/se/se_critical.mp3"
	seHeal          = "assets/se/se_heal.mp3"
	seBuff          = "assets/se/se_buff.mp3"
	seDebuff        = "assets/se/se_debuff.mp3"
	seDamage        = "assets/se/se_damage.mp3"
	seEvade         = "assets/se/se_evade.mp3"
	seGuard         = "assets/se/se_guard.mp3"
	seEnemyDefeated = "assets/se/se_enemy_defeated.mp3"
	seLevelUp       = "assets/se/se_level_up.mp3"
)

// seByKey はSE名(キー)→実ファイルパスの対応表。bgmByKeyと同じ理由で、
// 実データの差し替え時はこのファイルだけ書き換えればよいようにしてある。
// 現時点では assets/se/ 以下のファイルはすべて中身が空のプレースホルダーで、
// 実際の効果音ファイルに置き換えるまでは再生時に無音のままスキップされる。
var seByKey = map[string]string{
	"decide":         seDecide,
	"cancel":         seCancel,
	"cursor":         seCursor,
	"menu_toggle":    seMenuToggle,
	"error":          seError,
	"text_advance":   seTextAdvance,
	"footstep":       seFootstep,
	"dash_footstep":  seDashFootstep,
	"door":           seDoor,
	"sliding_door":   seSlidingDoor,
	"locker":         seLocker,
	"treasure_open":  seTreasureOpen,
	"item_get":       seItemGet,
	"switch":         seSwitch,
	"lever":          seLever,
	"water_flow":     seWaterFlow,
	"water_full":     seWaterFull,
	"water_drain":    seWaterDrain,
	"encounter":      seEncounter,
	"battle_start":   seBattleStart,
	"attack_normal":  seAttackNormal,
	"skill_attack":   seSkillAttack,
	"skill_heal":     seSkillHeal,
	"critical":       seCritical,
	"heal":           seHeal,
	"buff":           seBuff,
	"debuff":         seDebuff,
	"damage":         seDamage,
	"evade":          seEvade,
	"guard":          seGuard,
	"enemy_defeated": seEnemyDefeated,
	"level_up":       seLevelUp,
}

func resolveSEKey(key string) (string, bool) {
	path, ok := seByKey[key]
	return path, ok
}

type AudioManager struct {
	context *audio.Context

	bgmPlayer *audio.Player
	bgmName   string
	volume    float64

	seVolume     float64
	masterVolume float64
	sePlayers    []*audio.Player

	waitingLoopSwitch bool
	pendingLoopPath   string
	fadeInActive      bool
	fadeInTarget      float64
	fadeInSpeed       float64

	fadeOutActive     bool
	fadeOutSpeed      float64
	fadeOutNextPath   string
	fadeOutNextFadeIn float64
	fadeOutHardCut    bool

	// pcmCache/pcmLoading/pcmFailedはバックグラウンドのデコードgoroutineと
	// メインゴルーチンの両方から触るためpcmMuで保護する。
	pcmMu      sync.Mutex
	pcmCache   map[string][]byte
	pcmLoading map[string]bool
	pcmFailed  map[string]bool
	// bgmLRU は最近再生したBGMのパス(古い順)。BGMのPCMは1曲あたり数十MBに
	// なるため、キャッシュする曲数をmaxCachedBGMまでに制限する。
	bgmLRU []string

	// pendingStart は再生したいBGMのデコード待ち中に、デコード完了後に
	// 実行する再生開始処理。pendingPathsが全てデコード済み(または失敗)に
	// なった時点でUpdateから呼ばれる。pendingBGMはその待機中の曲名。
	pendingStart func()
	pendingPaths []string
	pendingBGM   string
}

// maxCachedBGM はPCMをメモリに保持しておくBGMの最大曲数。
const maxCachedBGM = 5

var errPCMFailed = errors.New("pcm decode failed")

func NewAudioManager() *AudioManager {
	return &AudioManager{
		context:      audio.NewContext(sampleRate),
		volume:       defaultBGMVolume,
		seVolume:     defaultSEVolume,
		masterVolume: defaultMasterVolume,
		pcmCache:     make(map[string][]byte),
		pcmLoading:   make(map[string]bool),
		pcmFailed:    make(map[string]bool),
	}
}

// effectiveBGMVolume はBGMスライダーと全体音量スライダーを掛け合わせた、
// 実際にプレイヤーへ渡す再生音量。
func (a *AudioManager) effectiveBGMVolume() float64 {
	return a.volume * a.masterVolume
}

// effectiveSEVolume はSEスライダーと全体音量スライダーを掛け合わせた、
// 実際にプレイヤーへ渡す再生音量。
func (a *AudioManager) effectiveSEVolume() float64 {
	return a.seVolume * a.masterVolume
}

// decodePCM はmp3を最後まで読み切り、PCMデータを丸ごとメモリに展開する。
// ストリーミングデコード(Playerが再生しながら少しずつmp3をデコードする方式)は、
// スマホの非力なCPUだとデコードが再生に間に合わずバッファが枯渇し、
// ブツブツ音の原因になる。事前に全部デコードしておけば再生中は
// メモリからコピーするだけになり、デコード負荷による音切れがなくなる。
// デコード結果は曲ごとにキャッシュし、2回目以降の再生(戦闘開始・終了の
// 繰り返しなど)で重いデコードが毎回走らないようにする。
// この関数はその場で同期的にデコードするため、BGMのような長い曲には
// 使わず(画面が固まる)、requestPCMでバックグラウンドデコードすること。
func (a *AudioManager) decodePCM(path string) (*bytes.Reader, int64, error) {
	a.pcmMu.Lock()
	pcm, ok := a.pcmCache[path]
	failed := a.pcmFailed[path]
	a.pcmMu.Unlock()
	if ok {
		return bytes.NewReader(pcm), int64(len(pcm)), nil
	}
	if failed {
		return nil, 0, errPCMFailed
	}

	pcm, err := decodePCMData(path)
	a.pcmMu.Lock()
	if err != nil {
		// 中身が空のプレースホルダーSEなどを毎回デコードし直さないよう、
		// 失敗も記録しておく。
		a.pcmFailed[path] = true
	} else {
		a.pcmCache[path] = pcm
	}
	a.pcmMu.Unlock()
	if err != nil {
		return nil, 0, err
	}
	return bytes.NewReader(pcm), int64(len(pcm)), nil
}

// decodePCMData はmp3を丸ごとPCMへデコードする。Web版はブラウザ組み込みの
// デコーダを使う(pcm_decode_js.go)。
func decodePCMData(path string) ([]byte, error) {
	data, err := loadAssetBytesCached(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		// assets/se/の未差し替えプレースホルダー(0バイト)。
		return nil, errPCMFailed
	}
	pcm, err := decodeMP3ToPCM(data)
	if err != nil {
		return nil, err
	}
	return trimTrailingSilence(pcm), nil
}

// requestPCM はpathのPCMがまだ無ければバックグラウンドでデコードを始める。
// 既にキャッシュ済み・デコード中・失敗済みなら何もしない。
func (a *AudioManager) requestPCM(path string) {
	if path == "" {
		return
	}
	a.pcmMu.Lock()
	_, cached := a.pcmCache[path]
	if cached || a.pcmLoading[path] || a.pcmFailed[path] {
		a.pcmMu.Unlock()
		return
	}
	a.pcmLoading[path] = true
	a.pcmMu.Unlock()

	go func() {
		pcm, err := decodePCMData(path)
		a.pcmMu.Lock()
		delete(a.pcmLoading, path)
		if err != nil {
			a.pcmFailed[path] = true
		} else {
			a.pcmCache[path] = pcm
		}
		a.pcmMu.Unlock()
	}()
}

// Prewarm は次に使いそうな曲・効果音をバックグラウンドで先にデコードしておく。
func (a *AudioManager) Prewarm(paths ...string) {
	if a == nil {
		return
	}
	for _, p := range paths {
		a.requestPCM(p)
	}
}

// pcmSettled はpathのデコードが終わっている(成功・失敗どちらでも)かを返す。
func (a *AudioManager) pcmSettled(path string) bool {
	a.pcmMu.Lock()
	defer a.pcmMu.Unlock()
	_, ok := a.pcmCache[path]
	return ok || a.pcmFailed[path]
}

// touchBGM はBGMのPCMキャッシュの使用順を更新し、上限を超えた古い曲を
// キャッシュから外す(再生中のPlayerは自前のReaderでPCMを参照し続けるので、
// キャッシュから外しても再生は途切れない)。メインゴルーチン専用。
func (a *AudioManager) touchBGM(path string) {
	if !strings.HasPrefix(path, "assets/bgm/") {
		return
	}
	for i, p := range a.bgmLRU {
		if p == path {
			a.bgmLRU = append(a.bgmLRU[:i], a.bgmLRU[i+1:]...)
			break
		}
	}
	a.bgmLRU = append(a.bgmLRU, path)
	for len(a.bgmLRU) > maxCachedBGM {
		evict := a.bgmLRU[0]
		a.bgmLRU = a.bgmLRU[1:]
		a.pcmMu.Lock()
		delete(a.pcmCache, evict)
		a.pcmMu.Unlock()
	}
}

// startWhenDecoded はpathsのPCMが揃った時点でstartを実行する。揃っていれば
// その場で実行し、まだならバックグラウンドデコードを始めてUpdateに任せる。
// 待機できるのは1件だけで、後から呼んだものが優先される。
func (a *AudioManager) startWhenDecoded(start func(), paths ...string) {
	for _, p := range paths {
		a.requestPCM(p)
	}
	a.pendingStart = start
	a.pendingPaths = paths
	a.tryPendingStart()
}

func (a *AudioManager) tryPendingStart() {
	if a.pendingStart == nil {
		return
	}
	for _, p := range a.pendingPaths {
		if !a.pcmSettled(p) {
			return
		}
	}
	start := a.pendingStart
	a.pendingStart = nil
	a.pendingPaths = nil
	a.pendingBGM = ""
	start()
}

// IsLoading は次に鳴らすBGMのデコード待ちがあるかを返す。Game側は
// シーン切り替えの暗転中にこれを見て、曲の準備ができるまでLoading表示で
// 待つ(画面が明けてから遅れて曲が鳴り始める、を防ぐ)。
func (a *AudioManager) IsLoading() bool {
	if a == nil {
		return false
	}
	if a.pendingStart != nil {
		return true
	}
	return a.fadeOutActive && a.fadeOutNextPath != "" && !a.pcmSettled(a.fadeOutNextPath)
}

// trimTrailingSilence はmp3→PCM変換後の完全な無音区間(エンコーダーが
// ファイル末尾に付与するパディング)を取り除く。これを残したまま
// NewInfiniteLoopで無限ループさせると、曲が実質的に終わった後も
// 無音のまま再生され続けてから先頭に戻るため、ループが即座に
// 繋がらず体感で大きな空白ができてしまう。
func trimTrailingSilence(pcm []byte) []byte {
	const bytesPerFrame = 4 // 16bit stereo
	i := len(pcm)
	for i >= bytesPerFrame {
		frame := pcm[i-bytesPerFrame : i]
		silent := frame[0] == 0 && frame[1] == 0 && frame[2] == 0 && frame[3] == 0
		if !silent {
			break
		}
		i -= bytesPerFrame
	}
	if i == 0 {
		return pcm
	}
	return pcm[:i]
}

func (a *AudioManager) loadStreamPlayer(path string) (*audio.Player, error) {
	a.touchBGM(path)
	r, _, err := a.decodePCM(path)
	if err != nil {
		return nil, err
	}
	p, err := a.context.NewPlayer(r)
	if err != nil {
		return nil, err
	}
	p.SetBufferSize(bgmBufferSize)
	p.SetVolume(a.effectiveBGMVolume())
	return p, nil
}

func (a *AudioManager) loadLoopPlayer(path string) (*audio.Player, error) {
	a.touchBGM(path)
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
	p.SetVolume(a.effectiveBGMVolume())
	return p, nil
}

func (a *AudioManager) PlayBGM(path string) {
	a.playBGM(path, 0)
}

func (a *AudioManager) PlayBGMFadeIn(path string, duration float64) {
	a.playBGM(path, duration)
}

// playBGM はpathをループ再生する(fadeIn>0ならその秒数かけてフェードイン)。
// PCMがまだ無ければバックグラウンドでデコードし、終わり次第Updateから
// 再生を開始する(その間は無音)。以前はここでmp3を丸ごと同期デコード
// していたため、曲が切り替わるたびに数百ms〜数秒画面が固まっていた。
func (a *AudioManager) playBGM(path string, fadeIn float64) {
	if a == nil {
		return
	}
	if a.bgmName == path && a.bgmPlayer != nil && a.bgmPlayer.IsPlaying() {
		return
	}
	if a.pendingStart != nil && a.pendingBGM == path {
		return
	}
	a.stopCurrent()

	a.pendingBGM = path
	a.startWhenDecoded(func() {
		p, err := a.loadLoopPlayer(path)
		if err != nil {
			return
		}
		a.bgmPlayer = p
		a.bgmName = path
		if fadeIn > 0 {
			p.SetVolume(0)
			a.fadeInActive = true
			a.fadeInTarget = a.effectiveBGMVolume()
			a.fadeInSpeed = a.fadeInTarget / fadeIn
		}
		p.Play()
	}, path)
}

func (a *AudioManager) PlayBGMWithIntro(introPath, loopPath string) {
	if a == nil {
		return
	}
	a.stopCurrent()

	a.pendingBGM = introPath
	// ループ部分もイントロと一緒に先にデコードしておき、イントロ終了時の
	// PlayBGM(loopPath)で待ちが発生しないようにする。
	a.startWhenDecoded(func() {
		p, err := a.loadStreamPlayer(introPath)
		if err != nil {
			a.PlayBGM(loopPath)
			return
		}
		p.Play()
		a.bgmPlayer = p
		a.bgmName = introPath
		a.waitingLoopSwitch = true
		a.pendingLoopPath = loopPath
	}, introPath, loopPath)
}

// FadeOutThenPlay は現在のBGMを fadeOutDuration 秒かけてフェードアウトし、
// 完全に無音になった瞬間に次の曲を再生する。シーン遷移の画面フェードと
// 同じ長さを fadeOutDuration に渡せば、画面が暗転しきるタイミングと
// 曲が切り替わるタイミングが自然に一致する(両方とも毎フレーム同じdtで
// 減っていくため)。次の曲は hardCut なら即座にフルボリュームで、
// そうでなければ fadeInDuration 秒かけてフェードインする。
func (a *AudioManager) FadeOutThenPlay(nextPath string, fadeOutDuration, fadeInDuration float64, hardCut bool) {
	if a == nil {
		return
	}
	// フェードアウトしている間に次の曲のデコードを進めておく。
	a.requestPCM(nextPath)
	if a.pendingStart != nil && a.pendingBGM == nextPath {
		return
	}
	if a.bgmName == nextPath && a.bgmPlayer != nil && a.bgmPlayer.IsPlaying() {
		a.fadeOutActive = false
		a.fadeInActive = false
		a.bgmPlayer.SetVolume(a.effectiveBGMVolume())
		return
	}
	if fadeOutDuration <= 0 || a.bgmPlayer == nil || !a.bgmPlayer.IsPlaying() {
		a.startNext(nextPath, fadeInDuration, hardCut)
		return
	}
	a.fadeInActive = false
	a.fadeOutActive = true
	a.fadeOutSpeed = a.bgmPlayer.Volume() / fadeOutDuration
	a.fadeOutNextPath = nextPath
	a.fadeOutNextFadeIn = fadeInDuration
	a.fadeOutHardCut = hardCut
}

func (a *AudioManager) startNext(path string, fadeInDuration float64, hardCut bool) {
	if hardCut || fadeInDuration <= 0 {
		a.PlayBGM(path)
	} else {
		a.PlayBGMFadeIn(path, fadeInDuration)
	}
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
	// フェード中に直接PlayBGM系が呼ばれてプレイヤーが差し替わった場合、
	// 古いフェード情報(fadeOutActive等)が残っていると、次のUpdateで
	// 新しく再生し始めたばかりの曲に対して古いフェードアウトが適用され、
	// 突然無音になって別の曲に切り替わってしまう(意図しないBGM停止の原因)。
	// プレイヤー差し替え時は必ず両方のフェード状態を破棄する。
	a.fadeOutActive = false
	a.fadeInActive = false
	a.pendingStart = nil
	a.pendingPaths = nil
	a.pendingBGM = ""
}

func (a *AudioManager) SetVolume(v float64) {
	if a == nil {
		return
	}
	a.volume = clampVolume(v)
	if a.bgmPlayer != nil {
		a.bgmPlayer.SetVolume(a.effectiveBGMVolume())
	}
}

func (a *AudioManager) SetSEVolume(v float64) {
	if a == nil {
		return
	}
	a.seVolume = clampVolume(v)
}

// SetMasterVolume はゲーム全体の音量を設定する。BGM/SEそれぞれのスライダー値に
// 掛け合わされる形で最終的な再生音量が決まる。
func (a *AudioManager) SetMasterVolume(v float64) {
	if a == nil {
		return
	}
	a.masterVolume = clampVolume(v)
	if a.bgmPlayer != nil {
		a.bgmPlayer.SetVolume(a.effectiveBGMVolume())
	}
}

func clampVolume(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// PlaySE は短い効果音を1回再生する(BGMとは独立した一回限りのPlayerを使う)。
// アセットが空のプレースホルダーのままだとデコードに失敗するが、これは
// 差し替え前の既知の状態であり警告するようなバグではないため、
// 黙って再生をスキップするだけにする。
func (a *AudioManager) PlaySE(path string) {
	if a == nil || path == "" {
		return
	}
	r, _, err := a.decodePCM(path)
	if err != nil {
		return
	}
	p, err := a.context.NewPlayer(r)
	if err != nil {
		return
	}
	p.SetVolume(a.effectiveSEVolume())
	p.Play()
	a.sePlayers = append(a.sePlayers, p)
}

// PlaySEByKey はseByKeyのキー経由でSEを再生する。未登録キーは何もしない。
func (a *AudioManager) PlaySEByKey(key string) {
	if a == nil {
		return
	}
	if path, ok := resolveSEKey(key); ok {
		a.PlaySE(path)
	}
}

func (a *AudioManager) Update(dt float64) {
	if a == nil {
		return
	}
	a.tryPendingStart()
	if len(a.sePlayers) > 0 {
		alive := a.sePlayers[:0]
		for _, p := range a.sePlayers {
			if p.IsPlaying() {
				alive = append(alive, p)
			} else {
				p.Close()
			}
		}
		a.sePlayers = alive
	}
	if a.waitingLoopSwitch && a.bgmPlayer != nil && !a.bgmPlayer.IsPlaying() {
		loopPath := a.pendingLoopPath
		a.waitingLoopSwitch = false
		a.PlayBGM(loopPath)
	}
	if a.fadeOutActive && a.bgmPlayer != nil {
		cur := a.bgmPlayer.Volume()
		cur -= a.fadeOutSpeed * dt
		if cur <= 0 {
			cur = 0
			a.bgmPlayer.SetVolume(cur)
			a.fadeOutActive = false
			a.startNext(a.fadeOutNextPath, a.fadeOutNextFadeIn, a.fadeOutHardCut)
		} else {
			a.bgmPlayer.SetVolume(cur)
		}
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
