#!/bin/bash

# Exit on error
set -e

# Project root directory
ROOT_DIR=$(pwd)

echo "Building web frontend..."
cd "$ROOT_DIR/web"
if command -v bun >/dev/null 2>&1; then
  bun install
  bun run build
else
  npm install
  npm run build
fi

echo "Preparing server public directory..."
mkdir -p "$ROOT_DIR/server/public"
rm -rf "$ROOT_DIR/server/public/*"
cp -r "$ROOT_DIR/web/dist/"* "$ROOT_DIR/server/public/"

echo "Building Windows binary..."
cd "$ROOT_DIR/server"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$ROOT_DIR/build/tavily-proxy-win.exe" main.go

echo "Building Linux binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$ROOT_DIR/build/tavily-proxy-linux" main.go

echo "Build complete! Binaries are in the build/ directory."
