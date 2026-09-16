#!/bin/sh
set -e
cd "$(dirname "$0")"

GOOS=js GOARCH=wasm go build -o web/game.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

echo "OK: web/game.wasm と web/wasm_exec.js を更新しました。"
