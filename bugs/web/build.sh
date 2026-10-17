#!/bin/sh
# Builds the web version into web/out for playing in a browser or a Claude artifact.
set -e
cd "$(dirname "$0")/.."
mkdir -p web/out
GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" -o web/out/bugs.wasm ./cmd/bugs
# gzipped but named .wasm, since artifacts only serve known file types; index.html unzips it
gzip -9 -f web/out/bugs.wasm
mv web/out/bugs.wasm.gz web/out/bugs.gz.wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/out/
cp web/index.html web/out/
