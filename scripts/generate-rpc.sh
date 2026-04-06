#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KITEX_BIN="${KITEX_BIN:-$(go env GOPATH)/bin/kitex}"

if [[ ! -x "${KITEX_BIN}" ]]; then
  echo "未找到 kitex，可执行文件路径: ${KITEX_BIN}" >&2
  exit 1
fi

rm -rf "${ROOT_DIR}/gen/rpc/kitex_gen"

pushd "${ROOT_DIR}/gen/rpc" >/dev/null
"${KITEX_BIN}" -module example.com/fanone/gen-rpc -type protobuf -I "${ROOT_DIR}/idl/rpc" "${ROOT_DIR}/idl/rpc/user/v1/user.proto"
"${KITEX_BIN}" -module example.com/fanone/gen-rpc -type protobuf -I "${ROOT_DIR}/idl/rpc" "${ROOT_DIR}/idl/rpc/video/v1/video.proto"
"${KITEX_BIN}" -module example.com/fanone/gen-rpc -type protobuf -I "${ROOT_DIR}/idl/rpc" "${ROOT_DIR}/idl/rpc/interaction/v1/interaction.proto"
"${KITEX_BIN}" -module example.com/fanone/gen-rpc -type protobuf -I "${ROOT_DIR}/idl/rpc" "${ROOT_DIR}/idl/rpc/chat/v1/chat.proto"
popd >/dev/null
