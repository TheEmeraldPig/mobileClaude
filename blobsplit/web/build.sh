#!/bin/sh
# Builds the web version of Biote into a folder (default web/dist): index.html (the
# page, with Go's wasm_exec.js inlined) and biote-wasm.txt (the game compiled to
# WebAssembly, gzipped, base64). The page unpacks it in the browser.
#   sh web/build.sh [out-dir]      (run from blobsplit/)
set -e
out=${1:-web/dist}
mkdir -p "$out"
GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w -X github.com/TheEmeraldPig/mobileClaude/blobsplit/game.Version=web" -o "$out/biote.wasm" ./cmd/blobsplit
gzip -9 -c "$out/biote.wasm" | base64 -w0 > "$out/biote-wasm.txt"
rm "$out/biote.wasm"
exec_js="$(go env GOROOT)/lib/wasm/wasm_exec.js"
[ -f "$exec_js" ] || exec_js="$(go env GOROOT)/misc/wasm/wasm_exec.js"
python3 - "$exec_js" "$out" <<'PY'
import sys
js = open(sys.argv[1]).read()
page = open("web/page.html").read()
size = len(open(sys.argv[2] + "/biote-wasm.txt").read())
page = page.replace("/*WASM_EXEC*/", js).replace("5498612", str(size))
open(sys.argv[2] + "/index.html", "w").write(page)
PY
ls -la "$out"
