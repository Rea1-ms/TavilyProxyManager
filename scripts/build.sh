#!/usr/bin/env bash
set -euo pipefail

if command -v bun >/dev/null 2>&1; then
  (cd web && bun install)
  (cd web && bun run build)
else
  (cd web && npm install)
  (cd web && npm run build)
fi

mkdir -p server/public
cp -R web/dist/* server/public/

go build -o tavily-proxy ./server
echo "Built: tavily-proxy"

