#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -H windowsgui -X main.Version=1.2.4" -o dist/Zoom-loader.exe .
echo "wrote dist/Zoom-loader.exe"
ls -lh dist/Zoom-loader.exe
