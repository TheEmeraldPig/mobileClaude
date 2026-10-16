#!/bin/sh
# Builds the web version into web/out for playing in a browser or a Claude artifact.
set -e
cd "$(dirname "$0")/.."
mkdir -p web/out
GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" -o web/out/hop.wasm ./cmd/hop
# gzipped but named .wasm, since artifacts only serve known file types; index.html unzips it
gzip -9 -f web/out/hop.wasm
mv web/out/hop.wasm.gz web/out/hop.gz.wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/out/
cp web/index.html web/out/
