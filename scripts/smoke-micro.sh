#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.runtime/micro.env"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

GATEWAY_ADDR="${BASE_URL:-http://localhost:8888}"
CHAT_ADDR="${CHAT_BASE_URL:-http://localhost:8889}"

check_http() {
  local name="$1"
  local url="$2"
  echo "[smoke] 检查 $name -> $url"
  curl --noproxy localhost -fsS "$url" >/dev/null
}

check_http "gateway" "$GATEWAY_ADDR/ping"
check_http "chat-service" "$CHAT_ADDR/ping"

echo "[smoke] 运行微服务 e2e"
(
  cd "$ROOT_DIR/test"
  go run .
)

echo "[smoke] 微服务冒烟通过"
