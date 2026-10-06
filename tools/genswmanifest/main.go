// genswmanifest はデプロイ前の公開フォルダ(web/)にある全ファイルの中身の
// ハッシュを計算し、Service Worker(sw.js)とindex.htmlのプレースホルダーへ
// 書き込む。GitHub Actionsのデプロイで、wasmのビルドと最適化が終わった後に実行する。
//
//	go run ./tools/genswmanifest web
//
// 書き込むもの:
//   - sw.js / index.html の "__ASSET_MANIFEST__" → {"相対パス": "ハッシュ", ...}
//     (ゲーム本体はindex.htmlの一覧を見て、ファイルを「パス?v=ハッシュ」で頼む)
//   - sw.js / index.html の __BUILD_ID__ → 一覧全体のハッシュ
//   - index.html の __WASM_EXEC_HASH__ → wasm_exec.js のハッシュ
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 書き換える側のファイルは一覧に入れない(中身が書き換わるので)。
var excluded = map[string]bool{"index.html": true, "sw.js": true}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genswmanifest <web dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "genswmanifest:", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	manifest, err := hashFiles(dir)
	if err != nil {
		return err
	}
	for _, required := range []string{"game.wasm", "wasm_exec.js"} {
		if _, ok := manifest[required]; !ok {
			return fmt.Errorf("%s が %s にありません(wasmのビルド後に実行すること)", required, dir)
		}
	}
	manifestJSON, err := json.Marshal(manifest) // キーは並べ替えて出力される
	if err != nil {
		return err
	}
	buildID := shortHash(manifestJSON)

	if err := replaceInFile(filepath.Join(dir, "sw.js"), map[string]string{
		`"__ASSET_MANIFEST__"`: string(manifestJSON),
		"__BUILD_ID__":         buildID,
	}); err != nil {
		return err
	}
	if err := replaceInFile(filepath.Join(dir, "index.html"), map[string]string{
		`"__ASSET_MANIFEST__"`: string(manifestJSON),
		"__BUILD_ID__":         buildID,
		"__WASM_EXEC_HASH__":   manifest["wasm_exec.js"],
	}); err != nil {
		return err
	}
	fmt.Printf("genswmanifest: %d files, build %s\n", len(manifest), buildID)
	return nil
}

// hashFiles はdir以下の全ファイルの、dirからの相対パス(/区切り) → 中身のハッシュを返す。
func hashFiles(dir string) (map[string]string, error) {
	manifest := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if excluded[rel] || strings.HasPrefix(filepath.Base(rel), ".") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		manifest[rel] = shortHash(data)
		return nil
	})
	return manifest, err
}

func shortHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:16]
}

// replaceInFile はpathの中のプレースホルダーを置き換える。どれか1つでも
// 見つからなければエラーにする(置き換え漏れのまま公開しないように)。
func replaceInFile(path string, repl map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(data)
	keys := make([]string, 0, len(repl))
	for k := range repl {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !strings.Contains(s, k) {
			return fmt.Errorf("%s に %s がありません", path, k)
		}
		s = strings.ReplaceAll(s, k, repl[k])
	}
	return os.WriteFile(path, []byte(s), 0o644)
}
