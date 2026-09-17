package main

import (
	"bytes"
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

	pcmCache map[string][]byte
}

func NewAudioManager() *AudioManager {
	return &AudioManager{
		context:      audio.NewContext(sampleRate),
		volume:       defaultBGMVolume,
		seVolume:     defaultSEVolume,
		masterVolume: defaultMasterVolume,
		pcmCache:     make(map[string][]byte),
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
	pcm = trimTrailingSilence(pcm)
	a.pcmCache[path] = pcm
	return bytes.NewReader(pcm), int64(len(pcm)), nil
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
	if a == nil {
		return
	}
	if a.bgmName == path && a.bgmPlayer != nil && a.bgmPlayer.IsPlaying() {
		return
	}
	a.stopCurrent()

	p, err := a.loadLoopPlayer(path)
	if err != nil {
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
		return
	}
	p.SetVolume(0)
	p.Play()
	a.bgmPlayer = p
	a.bgmName = path
	a.fadeInActive = true
	a.fadeInTarget = a.effectiveBGMVolume()
	a.fadeInSpeed = a.fadeInTarget / duration
}

func (a *AudioManager) PlayBGMWithIntro(introPath, loopPath string) {
	if a == nil {
		return
	}
	a.stopCurrent()

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
