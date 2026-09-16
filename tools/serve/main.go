package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	dir := flag.String("dir", "web", "配信するディレクトリ")
	port := flag.String("port", "8080", "listenするポート番号")
	watch := flag.Bool("watch", true, "起動時と.go/assets変更時に自動でgame.wasmを再ビルドする")
	flag.Parse()

	mime.AddExtensionType(".wasm", "application/wasm")

	if *watch {
		startWasmWatcher(*dir)
	}

	mux := http.NewServeMux()
	mux.Handle("/game.wasm", noCache(gzipWasmHandler(filepath.Join(*dir, "game.wasm"))))
	mux.Handle("/", noCache(http.FileServer(http.Dir(*dir))))

	addr := ":" + *port
	fmt.Printf("配信ディレクトリ: %s\n", *dir)
	for _, ip := range localIPv4s() {
		fmt.Printf("スマホから: http://%s:%s/\n", ip, *port)
	}
	log.Fatal(http.ListenAndServe(addr, mux))
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

var watchSkipDirs = map[string]bool{
	"web":   true,
	"tools": true,
	".git":  true,
}

func shouldSkipRootFile(name string) bool {
	if strings.HasPrefix(name, "save") && (strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".png")) {
		return true
	}
	switch filepath.Ext(name) {
	case ".exe", ".wasm":
		return true
	}
	return name == "settings.json"
}

func latestSourceModTime(root string) (time.Time, error) {
	var latest time.Time
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			if strings.HasPrefix(name, ".") || watchSkipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Dir(path) == root && shouldSkipRootFile(name) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest, err
}

func rebuildWasm(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	tmpPath := filepath.Join(outDir, "game.wasm.building")
	finalPath := filepath.Join(outDir, "game.wasm")

	cmd := exec.Command("go", "build", "-o", tmpPath, ".")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("%s", out)
	}
	return os.Rename(tmpPath, finalPath)
}

func startWasmWatcher(outDir string) {
	root, err := os.Getwd()
	if err != nil {
		log.Printf("watch: カレントディレクトリの取得に失敗したため自動再ビルドは無効化します: %v", err)
		return
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		log.Printf("watch: go.modが見つからないため自動再ビルドは無効化します（プロジェクトルートで実行してください）")
		return
	}

	build := func(reason string) {
		start := time.Now()
		fmt.Printf("🔨 %s: game.wasmを再ビルド中...\n", reason)
		if err := rebuildWasm(outDir); err != nil {
			fmt.Printf("❌ ビルド失敗:\n%s\n", err)
			return
		}
		fmt.Printf("✅ 再ビルド完了 (%.1fs) — スマホ側でページを再読み込みしてください\n", time.Since(start).Seconds())
	}

	lastMod, _ := latestSourceModTime(root)
	build("起動時")

	go func() {
		ticker := time.NewTicker(700 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			mod, err := latestSourceModTime(root)
			if err != nil || !mod.After(lastMod) {
				continue
			}
			lastMod = mod
			time.Sleep(150 * time.Millisecond)
			if mod2, err := latestSourceModTime(root); err == nil {
				lastMod = mod2
			}
			build("変更を検知")
		}
	}()
}

type gzipWasmCache struct {
	mu      sync.Mutex
	modTime time.Time
	data    []byte
}

func (c *gzipWasmCache) get(path string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if c.data != nil && c.modTime.Equal(info.ModTime()) {
		return c.data, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(raw); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	c.data = buf.Bytes()
	c.modTime = info.ModTime()
	fmt.Printf("game.wasmをgzip圧縮: %.1fMB → %.1fMB\n",
		float64(len(raw))/1e6, float64(len(c.data))/1e6)
	return c.data, nil
}

func gzipWasmHandler(path string) http.Handler {
	cache := &gzipWasmCache{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			http.ServeFile(w, r, path)
			return
		}
		data, err := cache.get(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/wasm")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	})
}

func localIPv4s() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			ips = append(ips, ip4.String())
		}
	}
	return ips
}
