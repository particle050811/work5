#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="$ROOT_DIR/.runtime/pids"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return
  fi
  docker-compose "$@"
}

stop_service() {
  local name="$1"
  local pid_file="$PID_DIR/${name}.pid"

  if [[ ! -f "$pid_file" ]]; then
    echo "[dev-down] $name 未运行"
    return
  fi

  local pid
  pid="$(cat "$pid_file")"
  if kill -0 "$pid" >/dev/null 2>&1; then
    echo "[dev-down] 停止 $name pid=$pid"
    kill "$pid"
    wait "$pid" 2>/dev/null || true
  else
    echo "[dev-down] $name pid=$pid 已不存在"
  fi
  rm -f "$pid_file"
}

stop_service "gateway"
stop_service "chat-service"
stop_service "interaction-service"
stop_service "video-service"
stop_service "user-service"

echo "[dev-down] 停止基础设施容器"
compose -f "$ROOT_DIR/deploy/docker-compose.micro.yml" down -v
