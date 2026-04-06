#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="$ROOT_DIR/.runtime/pids"
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

print_service() {
  local name="$1"
  local pid_file="$PID_DIR/${name}.pid"

  if [[ ! -f "$pid_file" ]]; then
    echo "$name: stopped"
    return
  fi

  local pid
  pid="$(cat "$pid_file")"
  if kill -0 "$pid" >/dev/null 2>&1; then
    echo "$name: running pid=$pid"
    return
  fi

  echo "$name: stale pid=$pid"
}

print_service "user-service"
print_service "video-service"
print_service "interaction-service"
print_service "chat-service"
print_service "gateway"

echo "--- docker compose ---"
compose -f "$ROOT_DIR/deploy/docker-compose.micro.yml" ps
