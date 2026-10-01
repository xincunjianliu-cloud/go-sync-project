package main

import (
	"container/heap"
	"sync"
)

// アセット取得の優先度。値が小さいほど先に取得する。同じ優先度の中では
// 依頼した順。すでに待ち行列にあるアセットをより高い優先度で依頼し直すと、
// その場で順番が繰り上がる。
const (
	// prioUrgent は今まさに処理が止まって待っているアセット
	// （loadAssetBytesCachedでの同期読み込みなど）。
	prioUrgent = iota
	// prioBoot はタイトル画面を出すのに必要なアセット。
	prioBoot
	// prioSoon はもうすぐ使うと分かっているアセット（これから始まる会話の
	// 立ち絵、タイトルBGMなど）。
	prioSoon
	// prioTierBase+tier は段階読み込み(assetTier)の各段階。
	prioTierBase
)

// prioAudioPrefetch は隣のマップのBGMなど、使うかもしれない程度の先読み。
// 段階読み込みがすべて終わってから取得する。
const prioAudioPrefetch = prioTierBase + int(assetTierCount)

// assetFetchWorkersWeb/Native は同時に走らせる取得の数。Web版は数を絞る
// ことで、優先度の高いアセットに回線を先に回す（全部を一斉に始めると、
// 必要なものも不要なものも同じ速さで少しずつしか届かない）。ネイティブ版は
// 埋め込みアセットを読むだけなので待ちがない。
const (
	assetFetchWorkersWeb    = 6
	assetFetchWorkersNative = 4
)

type assetJob struct {
	path  string
	prio  int
	seq   int
	index int // 待ち行列(heap)内の位置。取得中・取得済みは-1

	done chan struct{}
	data []byte
	err  error
}

type assetJobHeap []*assetJob

func (h assetJobHeap) Len() int { return len(h) }
func (h assetJobHeap) Less(i, j int) bool {
	if h[i].prio != h[j].prio {
		return h[i].prio < h[j].prio
	}
	return h[i].seq < h[j].seq
}
func (h assetJobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *assetJobHeap) Push(x any) {
	job := x.(*assetJob)
	job.index = len(*h)
	*h = append(*h, job)
}
func (h *assetJobHeap) Pop() any {
	old := *h
	n := len(old)
	job := old[n-1]
	old[n-1] = nil
	job.index = -1
	*h = old[:n-1]
	return job
}

// assetLoader は優先度つきの取得待ち行列と、取得済みバイト列の置き場所を
// 兼ねる。取得済みのバイト列は、使い終わった側がreleaseするまで保持する
// （画像はデコードした時点、マップはパースした時点で手放す）。
type assetLoader struct {
	mu      sync.Mutex
	cond    *sync.Cond
	queue   assetJobHeap
	jobs    map[string]*assetJob
	seq     int
	fetchFn func(path string) ([]byte, error)

	// fetchedBytes はこれまでに取得したバイト数の合計（計測用）。
	fetchedBytes int64
}

func newAssetLoader(workers int, fetchFn func(string) ([]byte, error)) *assetLoader {
	l := &assetLoader{jobs: map[string]*assetJob{}, fetchFn: fetchFn}
	l.cond = sync.NewCond(&l.mu)
	for i := 0; i < workers; i++ {
		go l.worker()
	}
	return l
}

var assetStore = newAssetLoader(assetFetchWorkers(), loadAssetBytes)

func assetFetchWorkers() int {
	if isWebBuild {
		return assetFetchWorkersWeb
	}
	return assetFetchWorkersNative
}

func (l *assetLoader) worker() {
	for {
		l.mu.Lock()
		for len(l.queue) == 0 {
			l.cond.Wait()
		}
		job := heap.Pop(&l.queue).(*assetJob)
		l.mu.Unlock()

		data, err := l.fetchFn(job.path)

		l.mu.Lock()
		job.data, job.err = data, err
		l.fetchedBytes += int64(len(data))
		close(job.done)
		l.mu.Unlock()
	}
}

// request はpathの取得を優先度prioで依頼し、待たずに戻る。取得中・取得済み
// なら何もしない。待ち行列にあってprioの方が高ければ順番を繰り上げる。
func (l *assetLoader) request(path string, prio int) *assetJob {
	l.mu.Lock()
	defer l.mu.Unlock()
	if job, ok := l.jobs[path]; ok {
		if job.index >= 0 && prio < job.prio {
			job.prio = prio
			heap.Fix(&l.queue, job.index)
		}
		return job
	}
	l.seq++
	job := &assetJob{path: path, prio: prio, seq: l.seq, done: make(chan struct{})}
	l.jobs[path] = job
	heap.Push(&l.queue, job)
	l.cond.Signal()
	return job
}

// get はpathのバイト列を返す。まだ届いていなければ最優先に繰り上げて
// 届くまで待つ。取得に失敗した結果は1回だけ返して忘れる（次に頼まれたら
// 取得し直す）。
func (l *assetLoader) get(path string) ([]byte, error) {
	return l.getAt(path, prioUrgent)
}

// getAt はgetと同じだが、まだ届いていないときの優先度をprioにする
// （バックグラウンドの先読みが、今まさに待っている取得を追い越さないように）。
func (l *assetLoader) getAt(path string, prio int) ([]byte, error) {
	job := l.request(path, prio)
	<-job.done
	if job.err != nil {
		l.mu.Lock()
		if l.jobs[path] == job {
			delete(l.jobs, path)
		}
		l.mu.Unlock()
		return nil, job.err
	}
	return job.data, nil
}

// wait はpathsがすべて届く(または失敗する)まで待つ。優先度は変えない。
func (l *assetLoader) wait(paths []string) {
	for _, p := range paths {
		l.mu.Lock()
		job, ok := l.jobs[p]
		l.mu.Unlock()
		if ok {
			<-job.done
		}
	}
}

// release は取得済みのpathのバイト列を手放す。デコード後の画像や
// パース後のマップのように、元のバイト列がもう要らないときに呼ぶ。
// 取得中・待ち行列にあるものは手放さない。
func (l *assetLoader) release(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	job, ok := l.jobs[path]
	if !ok {
		return
	}
	select {
	case <-job.done:
		delete(l.jobs, path)
	default:
	}
}

// has はpathが取得済み(成功)で手元にあるかを返す。
func (l *assetLoader) has(path string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	job, ok := l.jobs[path]
	if !ok {
		return false
	}
	select {
	case <-job.done:
		return job.err == nil
	default:
		return false
	}
}

func (l *assetLoader) totalFetchedBytes() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.fetchedBytes
}
