#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

# Copy wasm_exec.js from Go distribution (don't commit this file)
WASM_EXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
cp "$WASM_EXEC" .

# Build Go WASM binary
GOOS=js GOARCH=wasm go build -o digitalrain.wasm .
