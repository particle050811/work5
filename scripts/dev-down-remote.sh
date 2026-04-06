#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.runtime/micro.remote.env"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-fanone-micro-remote}"

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

echo "[dev-down-remote] 停止并清理远端镜像 compose 资源"
compose -f "$ROOT_DIR/deploy/docker-compose.remote.yml" down -v || true
