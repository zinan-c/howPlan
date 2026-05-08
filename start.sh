#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
ADMIN_MODE_VALUE="${1:-false}"

if [[ "$ADMIN_MODE_VALUE" != "true" && "$ADMIN_MODE_VALUE" != "false" ]]; then
  echo "Usage: ./start.sh [true|false]"
  echo "Example: ./start.sh true"
  exit 1
fi

pick_random_port() {
  local min_port=18080
  local max_port=19080
  local attempts=200
  local port
  local i

  for ((i = 0; i < attempts; i++)); do
    port=$((RANDOM % (max_port - min_port + 1) + min_port))
    if ! lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "$port"
      return 0
    fi
  done

  for ((port = min_port; port <= max_port; port++)); do
    if ! lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "$port"
      return 0
    fi
  done

  return 1
}

PORT_VALUE="$(pick_random_port || true)"
if [[ -z "$PORT_VALUE" ]]; then
  echo "No available port found in range 18080-19080"
  exit 1
fi

echo "Starting Travel Planner Viewer..."
echo "ADMIN_MODE=${ADMIN_MODE_VALUE}"
echo "PORT=${PORT_VALUE} (random in 18080-19080)"
echo "Open: http://localhost:${PORT_VALUE}/?admin=${ADMIN_MODE_VALUE}"

cd "$ROOT_DIR/backend"
mkdir -p .cache/go-build

(
  sleep 1
  open -a "Google Chrome" "http://localhost:${PORT_VALUE}/?admin=${ADMIN_MODE_VALUE}" >/dev/null 2>&1 || true
) &

GOCACHE="$PWD/.cache/go-build" ADMIN_MODE="$ADMIN_MODE_VALUE" PORT="$PORT_VALUE" go run ./cmd/server/main.go
