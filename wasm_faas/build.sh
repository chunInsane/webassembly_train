#!/bin/bash

set -eux
set -o pipefail

mkdir -p target
tinygo build -o target/hellogo.wasm -target=wasi hellogo/main.go
tinygo build -o target/goenv.wasm -target=wasi goenv/goenv.go
# GOOS=wasip1 GOARCH=wasm gotip build -o target/goenvgc.wasm examples/goenv/goenv.go

(cd hellors; cargo build --target wasm32-wasi --release)
cp hellors/target/wasm32-wasi/release/hellors.wasm target/