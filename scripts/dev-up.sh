#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="$ROOT_DIR/.runtime/pids"
LOG_DIR="$ROOT_DIR/.runtime/logs"
BIN_DIR="$ROOT_DIR/.runtime/bin"
ENV_FILE="$ROOT_DIR/.runtime/micro.env"

mkdir -p "$PID_DIR" "$LOG_DIR" "$BIN_DIR"

export JWT_SECRET="${JWT_SECRET:-fanone-microservices-secret-key-2024}"
export MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-123456}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
export REDIS_DB="${REDIS_DB:-0}"

RESERVED_PORTS=""

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return
  fi
  docker-compose "$@"
}

port_in_use() {
  local port="$1"
  ss -ltn "( sport = :$port )" | grep -q LISTEN
}

port_reserved() {
  local port="$1"
  [[ " $RESERVED_PORTS " == *" $port "* ]]
}

reserve_port() {
  local port="$1"
  if ! port_reserved "$port"; then
    RESERVED_PORTS="$RESERVED_PORTS $port"
  fi
}

choose_port() {
  local preferred="$1"
  local fallback="$2"
  local port

  for port in "$preferred" "$fallback"; do
    if ! port_in_use "$port" && ! port_reserved "$port"; then
      echo "$port"
      return
    fi
  done

  for port in $(seq "$fallback" $((fallback + 50))); do
    if ! port_in_use "$port" && ! port_reserved "$port"; then
      echo "$port"
      return
    fi
  done

  echo "[dev-up] 无法找到可用端口，起始端口=$preferred 备用端口=$fallback" >&2
  exit 1
}

wait_for_health() {
  local container="$1"
  local retries="${2:-60}"
  local status=""

  for _ in $(seq 1 "$retries"); do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      return 0
    fi
    sleep 1
  done

  echo "[dev-up] 等待容器健康状态超时 container=$container status=${status:-unknown}" >&2
  return 1
}

write_env_file() {
  {
    printf "ETCD_ENDPOINTS=%s\n" "$ETCD_ENDPOINTS"
    printf "USER_DB_DSN=%s\n" "$USER_DB_DSN"
    printf "VIDEO_DB_DSN=%s\n" "$VIDEO_DB_DSN"
    printf "INTERACTION_DB_DSN=%s\n" "$INTERACTION_DB_DSN"
    printf "MYSQL_ROOT_PASSWORD=%s\n" "$MYSQL_ROOT_PASSWORD"
    printf "REDIS_ADDR=%s\n" "$REDIS_ADDR"
    printf "REDIS_PASSWORD=%s\n" "$REDIS_PASSWORD"
    printf "REDIS_DB=%s\n" "$REDIS_DB"
    printf "JWT_SECRET=%s\n" "$JWT_SECRET"
    printf "STORAGE_ROOT=%s\n" "$STORAGE_ROOT"
    printf "USER_RPC_ADDR=%s\n" "$USER_RPC_ADDR"
    printf "VIDEO_RPC_ADDR=%s\n" "$VIDEO_RPC_ADDR"
    printf "INTERACTION_RPC_ADDR=%s\n" "$INTERACTION_RPC_ADDR"
    printf "CHAT_RPC_ADDR=%s\n" "$CHAT_RPC_ADDR"
    printf "CHAT_HTTP_ADDR=%s\n" "$CHAT_HTTP_ADDR"
    printf "GATEWAY_HTTP_ADDR=%s\n" "$GATEWAY_HTTP_ADDR"
    printf "BASE_URL=%s\n" "http://localhost:${GATEWAY_HTTP_ADDR##*:}"
    printf "CHAT_BASE_URL=%s\n" "http://localhost:${CHAT_HTTP_ADDR##*:}"
  } >"$ENV_FILE"
}

start_service() {
  local name="$1"
  local workdir="$2"
  local addr_key="$3"
  local addr_val="$4"
  local log_file="$LOG_DIR/${name}.log"
  local pid_file="$PID_DIR/${name}.pid"
  local bin_file="$BIN_DIR/${name}"
  local port="${addr_val##*:}"

  if [[ -f "$pid_file" ]]; then
    local old_pid
    old_pid="$(cat "$pid_file")"
    if kill -0 "$old_pid" >/dev/null 2>&1; then
      echo "[dev-up] $name 已在运行 pid=$old_pid"
      return
    fi
    rm -f "$pid_file"
  fi

  if port_in_use "$port"; then
    echo "[dev-up] 端口 $port 已被占用，无法启动 $name"
    return 1
  fi

  echo "[dev-up] 启动 $name"
  (
    cd "$workdir"
    go build -o "$bin_file" .
    env "$addr_key=$addr_val" \
      ETCD_ENDPOINTS="$ETCD_ENDPOINTS" \
      USER_DB_DSN="$USER_DB_DSN" \
      VIDEO_DB_DSN="$VIDEO_DB_DSN" \
      INTERACTION_DB_DSN="$INTERACTION_DB_DSN" \
      MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
      REDIS_ADDR="$REDIS_ADDR" \
      REDIS_PASSWORD="$REDIS_PASSWORD" \
      REDIS_DB="$REDIS_DB" \
      JWT_SECRET="$JWT_SECRET" \
      STORAGE_ROOT="$STORAGE_ROOT" \
      nohup "$bin_file" >"$log_file" 2>&1 &
    echo $! >"$pid_file"
  )

  local pid
  pid="$(cat "$pid_file")"
  for _ in $(seq 1 30); do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      echo "[dev-up] $name 启动失败，日志如下:"
      tail -n 50 "$log_file" || true
      return 1
    fi
    if port_in_use "$port"; then
      return 0
    fi
    sleep 1
  done

  echo "[dev-up] $name 启动超时，端口 $port 未监听，日志如下:"
  tail -n 50 "$log_file" || true
  return 1
}

ETCD_HOST_PORT="${ETCD_HOST_PORT:-$(choose_port 2379 22379)}"
reserve_port "$ETCD_HOST_PORT"
ETCD_PEER_HOST_PORT="${ETCD_PEER_HOST_PORT:-$(choose_port 2380 22380)}"
reserve_port "$ETCD_PEER_HOST_PORT"
MYSQL_HOST_PORT="${MYSQL_HOST_PORT:-$(choose_port 3306 23306)}"
reserve_port "$MYSQL_HOST_PORT"
REDIS_HOST_PORT="${REDIS_HOST_PORT:-$(choose_port 6379 26379)}"
reserve_port "$REDIS_HOST_PORT"

export ETCD_ENDPOINTS="${ETCD_ENDPOINTS:-127.0.0.1:$ETCD_HOST_PORT}"
export USER_DB_DSN="${USER_DB_DSN:-root:${MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:$MYSQL_HOST_PORT)/fanone_user?charset=utf8mb4&parseTime=True&loc=Local}"
export VIDEO_DB_DSN="${VIDEO_DB_DSN:-root:${MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:$MYSQL_HOST_PORT)/fanone_video?charset=utf8mb4&parseTime=True&loc=Local}"
export INTERACTION_DB_DSN="${INTERACTION_DB_DSN:-root:${MYSQL_ROOT_PASSWORD}@tcp(127.0.0.1:$MYSQL_HOST_PORT)/fanone_interaction?charset=utf8mb4&parseTime=True&loc=Local}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:$REDIS_HOST_PORT}"
export STORAGE_ROOT="${STORAGE_ROOT:-$ROOT_DIR/storage}"

export USER_RPC_ADDR="${USER_RPC_ADDR:-0.0.0.0:$(choose_port 9001 19001)}"
reserve_port "${USER_RPC_ADDR##*:}"
export VIDEO_RPC_ADDR="${VIDEO_RPC_ADDR:-0.0.0.0:$(choose_port 9002 19002)}"
reserve_port "${VIDEO_RPC_ADDR##*:}"
export INTERACTION_RPC_ADDR="${INTERACTION_RPC_ADDR:-0.0.0.0:$(choose_port 9003 19003)}"
reserve_port "${INTERACTION_RPC_ADDR##*:}"
export CHAT_RPC_ADDR="${CHAT_RPC_ADDR:-0.0.0.0:$(choose_port 9004 19004)}"
reserve_port "${CHAT_RPC_ADDR##*:}"
export CHAT_HTTP_ADDR="${CHAT_HTTP_ADDR:-:$(choose_port 8889 18889)}"
reserve_port "${CHAT_HTTP_ADDR##*:}"
export GATEWAY_HTTP_ADDR="${GATEWAY_HTTP_ADDR:-:$(choose_port 8888 18888)}"
reserve_port "${GATEWAY_HTTP_ADDR##*:}"

write_env_file

echo "[dev-up] 启动基础设施容器 etcd/mysql/redis"
ETCD_HOST_PORT="$ETCD_HOST_PORT" \
ETCD_PEER_HOST_PORT="$ETCD_PEER_HOST_PORT" \
MYSQL_HOST_PORT="$MYSQL_HOST_PORT" \
MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
REDIS_HOST_PORT="$REDIS_HOST_PORT" \
compose -f "$ROOT_DIR/deploy/docker-compose.micro.yml" up -d etcd mysql redis

wait_for_health "deploy_etcd_1"
wait_for_health "deploy_mysql_1"
wait_for_health "deploy_redis_1"

start_service "user-service" "$ROOT_DIR/services/user" "USER_RPC_ADDR" "$USER_RPC_ADDR"
start_service "video-service" "$ROOT_DIR/services/video" "VIDEO_RPC_ADDR" "$VIDEO_RPC_ADDR"
start_service "interaction-service" "$ROOT_DIR/services/interaction" "INTERACTION_RPC_ADDR" "$INTERACTION_RPC_ADDR"
start_service "chat-service" "$ROOT_DIR/services/chat" "CHAT_RPC_ADDR" "$CHAT_RPC_ADDR"
start_service "gateway" "$ROOT_DIR/services/gateway" "GATEWAY_HTTP_ADDR" "$GATEWAY_HTTP_ADDR"

echo "[dev-up] gateway 地址: http://localhost:${GATEWAY_HTTP_ADDR##*:}"
echo "[dev-up] chat-service WebSocket 地址: http://localhost:${CHAT_HTTP_ADDR##*:}"
echo "[dev-up] 运行环境文件: $ENV_FILE"
echo "[dev-up] 日志目录: $LOG_DIR"
echo "[dev-up] 使用 scripts/dev-status.sh 查看状态"
