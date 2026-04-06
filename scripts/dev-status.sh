#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.runtime/micro.env"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-fanone-micro}"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -p "$COMPOSE_PROJECT_NAME" "$@"
    return
  fi
  docker-compose -p "$COMPOSE_PROJECT_NAME" "$@"
}

echo "--- docker compose ---"
compose -f "$ROOT_DIR/deploy/docker-compose.micro.yml" ps

if [[ -n "${BASE_URL:-}" ]]; then
  echo "gateway: $BASE_URL"
fi
if [[ -n "${CHAT_BASE_URL:-}" ]]; then
  echo "chat-service: $CHAT_BASE_URL"
fi
