#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUNTIME_DIR="$ROOT_DIR/.runtime"
ENV_FILE="$RUNTIME_DIR/micro.env"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.micro.yml"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-fanone-micro}"

mkdir -p "$RUNTIME_DIR"

export JWT_SECRET="${JWT_SECRET:-fanone-microservices-secret-key-2024}"
export MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-hsr123456}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
export REDIS_DB="${REDIS_DB:-0}"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -p "$COMPOSE_PROJECT_NAME" "$@"
    return
  fi
  docker-compose -p "$COMPOSE_PROJECT_NAME" "$@"
}

cleanup_stale_containers() {
  local ids
  ids="$(docker ps -aq --filter "name=^/${COMPOSE_PROJECT_NAME}_" || true)"
  if [[ -n "$ids" ]]; then
    docker rm -f $ids >/dev/null 2>&1 || true
  fi
}

port_in_use() {
  local port="$1"
  ss -ltn "( sport = :$port )" | grep -q LISTEN
}

ensure_port_available() {
  local name="$1"
  local port="$2"
  if port_in_use "$port"; then
    echo "[dev-up] 端口已被占用 name=$name port=$port" >&2
    exit 1
  fi
}

wait_for_health() {
  local service="$1"
  local retries="${2:-120}"
  local container=""
  local status=""

  for _ in $(seq 1 "$retries"); do
    container="$(compose -f "$COMPOSE_FILE" ps -q "$service" 2>/dev/null | tail -n 1)"
    if [[ -z "$container" ]]; then
      sleep 1
      continue
    fi

    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      return 0
    fi
    sleep 1
  done

  echo "[dev-up] 等待容器健康状态超时 service=$service container=${container:-unknown} status=${status:-unknown}" >&2
  return 1
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local retries="${3:-60}"

  for _ in $(seq 1 "$retries"); do
    if curl --noproxy localhost -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "[dev-up] 等待 HTTP 服务超时 name=$name url=$url" >&2
  return 1
}

escape_env_value() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "$value"
}

ETCD_HOST_PORT="${ETCD_HOST_PORT:-22379}"
GATEWAY_HTTP_PORT="${GATEWAY_HTTP_PORT:-18888}"
CHAT_HTTP_PORT="${CHAT_HTTP_PORT:-18889}"

ensure_port_available "etcd" "$ETCD_HOST_PORT"
ensure_port_available "gateway" "$GATEWAY_HTTP_PORT"
ensure_port_available "chat-service" "$CHAT_HTTP_PORT"

export ETCD_HOST_PORT
export GATEWAY_HTTP_PORT CHAT_HTTP_PORT

export USER_DB_DSN="${USER_DB_DSN:-root:${MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:3306)/fanone_user?charset=utf8mb4&parseTime=True&loc=Local}"
export VIDEO_DB_DSN="${VIDEO_DB_DSN:-root:${MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:3306)/fanone_video?charset=utf8mb4&parseTime=True&loc=Local}"
export INTERACTION_DB_DSN="${INTERACTION_DB_DSN:-root:${MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:3306)/fanone_interaction?charset=utf8mb4&parseTime=True&loc=Local}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
export ETCD_ENDPOINTS="${ETCD_ENDPOINTS:-127.0.0.1:${ETCD_HOST_PORT}}"
export BASE_URL="http://localhost:${GATEWAY_HTTP_PORT}"
export CHAT_BASE_URL="http://localhost:${CHAT_HTTP_PORT}"

cleanup_stale_containers
compose -f "$COMPOSE_FILE" down -v --remove-orphans >/dev/null 2>&1 || true

echo "[dev-up] 构建并启动微服务容器"
compose -f "$COMPOSE_FILE" up -d --build

wait_for_health "etcd"
wait_for_health "user-service"
wait_for_health "video-service"
wait_for_health "interaction-service"
wait_for_health "chat-service"
wait_for_health "gateway"
wait_for_http "gateway" "$BASE_URL/ping"
wait_for_http "chat-service" "$CHAT_BASE_URL/ping"

{
  printf "COMPOSE_PROJECT_NAME=%s\n" "$(escape_env_value "$COMPOSE_PROJECT_NAME")"
  printf "ETCD_ENDPOINTS=%s\n" "$(escape_env_value "$ETCD_ENDPOINTS")"
  printf "MYSQL_ROOT_PASSWORD=%s\n" "$(escape_env_value "$MYSQL_ROOT_PASSWORD")"
  printf "USER_DB_DSN=%s\n" "$(escape_env_value "$USER_DB_DSN")"
  printf "VIDEO_DB_DSN=%s\n" "$(escape_env_value "$VIDEO_DB_DSN")"
  printf "INTERACTION_DB_DSN=%s\n" "$(escape_env_value "$INTERACTION_DB_DSN")"
  printf "REDIS_ADDR=%s\n" "$(escape_env_value "$REDIS_ADDR")"
  printf "REDIS_PASSWORD=%s\n" "$(escape_env_value "$REDIS_PASSWORD")"
  printf "REDIS_DB=%s\n" "$(escape_env_value "$REDIS_DB")"
  printf "JWT_SECRET=%s\n" "$(escape_env_value "$JWT_SECRET")"
  printf "BASE_URL=%s\n" "$(escape_env_value "$BASE_URL")"
  printf "CHAT_BASE_URL=%s\n" "$(escape_env_value "$CHAT_BASE_URL")"
} >"$ENV_FILE"

echo "[dev-up] gateway 地址: $BASE_URL"
echo "[dev-up] chat-service WebSocket 地址: $CHAT_BASE_URL"
echo "[dev-up] 运行环境文件: $ENV_FILE"
echo "[dev-up] 使用 scripts/dev-status.sh 查看状态"
