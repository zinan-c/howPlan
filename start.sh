#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
ADMIN_MODE_VALUE="${1:-false}"

if [[ "$ADMIN_MODE_VALUE" != "true" && "$ADMIN_MODE_VALUE" != "false" ]]; then
  echo "Usage: ./start.sh [true|false]"
  echo "Example: ./start.sh true"
  exit 1
fi

echo "Starting Travel Planner Viewer..."
echo "ADMIN_MODE=${ADMIN_MODE_VALUE}"
echo "Open: http://localhost:8080/?admin=${ADMIN_MODE_VALUE}"

cd "$ROOT_DIR/backend"
ADMIN_MODE="$ADMIN_MODE_VALUE" go run ./cmd/server/main.go
